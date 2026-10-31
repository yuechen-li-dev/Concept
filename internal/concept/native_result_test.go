package concept

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func resultSamples() int {
	if os.Getenv("CONCEPT_EVT2H_PHASE_QUALIFICATION") == "1" {
		return 100
	}
	return 10
}

func TestEVT2h5NativeResult(t *testing.T) {
	if resultSamples() == 100 {
		t.Setenv("CONCEPT_TEST_NATIVE_TIMEOUT", "8m")
	} else {
		t.Parallel()
	}
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64")
	}
	module, source, _, machine, data := resultFixture(t)
	dir := t.TempDir()
	artifact := filepath.Join(dir, "result.cmir9")
	dump := filepath.Join(dir, "native.bin")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	var defs strings.Builder
	defs.WriteString("static const char*probe_dlls[]={")
	for _, compiler := range []string{"cl", "clang-cl"} {
		if dll := buildIndependentSumProbe(t, compiler, "../../language/evt2h5/probes/results.c"); dll != "" {
			fmt.Fprintf(&defs, "%q,", filepath.ToSlash(dll))
		}
	}
	defs.WriteString("NULL};\n")
	fmt.Fprintf(&defs, "#define ARTIFACT %q\n#define DUMP %q\n#ifdef RESULT_NATIVE_ONLY\n#define TRIALS %d\n#else\n#define TRIALS 1\n#endif\n#define FUNCTION_COUNT %d\n", filepath.ToSlash(artifact), filepath.ToSlash(dump), resultSamples(), len(machine.Pattern.Base.Base.Objects.Functions))
	for i, f := range machine.Pattern.Base.Base.Objects.Functions {
		fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
	}
	host := strings.Replace(resultNativeHost, "/* DEFINITIONS */", defs.String(), 1)
	a, b := strings.Index(aggregateABIPlanningHost, "static unsigned long long digest;"), strings.Index(aggregateABIPlanningHost, "static int run_plan(")
	host = strings.Replace(host, "/* PLAN DEBUG */", aggregateABIPlanningHost[a:b], 1)
	var first []byte
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "result_native.c", "#define RESULT_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
			code, err := os.ReadFile(dump)
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = code
			} else if !bytes.Equal(first, code) {
				t.Fatal("Normal/Verify bytes differ")
			}
			t.Logf("native=%d sha256=%x", len(code), sha256.Sum256(code))
			if got := fmt.Sprintf("%x", sha256.Sum256(code)); got != "d556ccc3a47634f4697dfd5cf3c01572e46c00f50731445e9b8b2dad23ebfda2" {
				t.Fatalf("exact Result native bytes changed: %s", got)
			}
			policy := ConservativeCompilationPolicy()
			if verify {
				policy = VerifyCompilationPolicy()
			}
			oracle, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			for name, body := range oracle {
				if strings.HasSuffix(name, ".generated.c") {
					outputs["oracle.c"] = body
				} else {
					outputs[name] = body
				}
			}
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "result_oracle.c", host, "-pedantic-errors", "-O2"))
			after, err := os.ReadFile(dump)
			if err != nil || !bytes.Equal(code, after) {
				t.Fatalf("oracle changed code: %v", err)
			}
		})
	}
}

const resultNativeHost = `#include "amd64.generated.h"
#ifndef RESULT_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#define CHECK(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
/* DEFINITIONS */
typedef struct{uint32_t tag;union{int value;uint32_t error;}payload;} RInt;
typedef RInt(*Parse)(_Bool,int);typedef RInt(*Double)(_Bool,int);typedef int(*Code)(RInt);typedef RInt(*Validate)(_Bool);typedef RInt(*ValidateTwice)(_Bool,int*);
typedef struct{int count;double amount;} Item;
typedef struct{int first;int second;} Pair;
typedef struct{uint32_t tag;union{Pair value;uint16_t error;}payload;} RPair;
typedef struct{uint32_t tag;union{double value;uint32_t error;}payload;} RDouble;
typedef struct{uint32_t tag;union{float value;_Bool error;}payload;} RFloat;
typedef struct{uint32_t tag;union{_Bool value;uint8_t error;}payload;} RBool;
typedef struct{uint32_t tag;union{Item value;int integer;uint32_t error;}payload;} RItem;
typedef struct{uint32_t tag;union{int value[3];uint32_t error;}payload;} RArray;
typedef RDouble(*MakeDouble)(_Bool,double);typedef double(*DoubleValue)(RDouble);typedef RDouble(*DoubleCopy)(RDouble);
typedef RPair(*MakePair)(_Bool,int);typedef RPair(*PairCopy)(RPair);typedef int(*PairValue)(RPair);typedef uint16_t(*PairErrorValue)(RPair);typedef _Bool(*FloatErrorValue)(RFloat);typedef uint8_t(*BoolErrorValue)(RBool);typedef int(*Discard)(int*);
typedef RFloat(*MakeFloat)(_Bool,float);typedef float(*FloatValue)(RFloat);typedef RBool(*MakeBool)(_Bool,_Bool);typedef _Bool(*BoolValue)(RBool);
typedef RItem(*MakeItem)(_Bool,int);typedef Item(*ItemValue)(RItem);typedef RItem(*RecordCopy)(RItem);typedef RArray(*MakeArray)(_Bool);typedef int(*ArrayValue)(RArray);
typedef RItem(*RecordError)(_Bool,int);typedef int(*RecordErrorCode)(RItem);typedef RDouble(*DoubleError)(_Bool,double);typedef double(*DoubleErrorValue)(RDouble);typedef RArray(*ArrayError)(_Bool);
typedef RItem(*VoidRecord)(_Bool);typedef int(*VoidRecordCode)(RItem);typedef int(*LocalCopies)(int);typedef RInt(*Order)(int*,_Bool,int);typedef RItem(*RecordOrder)(int*,_Bool);
typedef RBool(*Positive)(_Bool,int);typedef RInt(*Recursive)(_Bool,int);typedef RInt(*Loop)(int,int,int*);typedef RDouble(*Mixed)(RInt,int,double,int*,RDouble,float,_Bool);typedef RDouble(*Caller)(_Bool,_Bool,int*);typedef RDouble(*Pressure)(double,double,double,double,double,double,double);
static concept_native_image image,other;
static unsigned char artifact[2000000],code[262144],again[262144];
static concept_backend_function function;static concept_object_placement placement;static concept_allocation allocation;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
#define ACCESS(t) static t Resolve##t(unsigned char*memory,int ordinal){t fn;void*p=memory+image.symbols.data[ordinal].offset;memcpy(&fn,&p,sizeof fn);return fn;}
ACCESS(Parse) ACCESS(Double) ACCESS(Code) ACCESS(Validate) ACCESS(ValidateTwice)
ACCESS(MakeDouble) ACCESS(DoubleValue) ACCESS(DoubleCopy) ACCESS(MakeFloat) ACCESS(FloatValue) ACCESS(MakeBool) ACCESS(BoolValue) ACCESS(MakeItem) ACCESS(ItemValue) ACCESS(RecordCopy) ACCESS(MakeArray) ACCESS(ArrayValue) ACCESS(RecordError) ACCESS(RecordErrorCode) ACCESS(DoubleError) ACCESS(DoubleErrorValue) ACCESS(ArrayError) ACCESS(VoidRecord) ACCESS(VoidRecordCode) ACCESS(LocalCopies) ACCESS(Order) ACCESS(RecordOrder) ACCESS(Positive) ACCESS(Recursive) ACCESS(Loop) ACCESS(Mixed) ACCESS(Caller) ACCESS(Pressure)
ACCESS(MakePair) ACCESS(PairCopy) ACCESS(PairValue) ACCESS(PairErrorValue) ACCESS(FloatErrorValue) ACCESS(BoolErrorValue) ACCESS(Discard)
#define FN(n,t) Resolve##t(memory,ORD_##n)
static unsigned char*native_begin;static int native_size;
static LONG CALLBACK expected_trap(PEXCEPTION_POINTERS e){if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&(uintptr_t)e->ExceptionRecord->ExceptionAddress>=(uintptr_t)native_begin&&(uintptr_t)e->ExceptionRecord->ExceptionAddress<(uintptr_t)native_begin+native_size)ExitProcess(0);return EXCEPTION_CONTINUE_SEARCH;}
int main(int argc,char**argv){
 FILE*f=fopen(ARTIFACT,"rb");CHECK(f);size_t n=fread(artifact,1,sizeof artifact,f);fclose(f);concept_readonly_span_byte src={artifact,n};
 concept_result_void_pattern_error admitted=F(validate_pattern_machine_bridge)(src);if(admitted.tag){fprintf(stderr,"%s\n",F(describe_pattern_error)(admitted.payload.error.error));return 1;}
 memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);other=image;
 CHECK(F(emit_module)(src,(concept_span_byte){code,1},&image).tag&&!memcmp(&image,&other,sizeof image));for(size_t i=0;i<sizeof code;i++)CHECK(code[i]==0xa5);
 clock_t start=clock();concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(emitted.tag){fprintf(stderr,"%s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}CHECK(image.finalized);
 printf("Result functions=%d calls=%d bytes=%d emission-ms=%.3f\n",image.symbolCount,image.fixupCount,image.byteCount,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);CHECK(!decoded.tag);concept_wire_object_module doc=decoded.payload.ok.value;
 int min_frame=4096,max_frame=0,spills=0;unsigned long long hashes[FUNCTION_COUNT];clock_t planning=clock();
 for(int run=0;run<(argc>1?0:TRIALS);run++)for(int fi=0;fi<doc.Functions.count;fi++){
  concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc,fi);CHECK(!owner.tag);
  function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();
  CHECK(!F(prepare_aggregate_function)(&reader,doc,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);
  unsigned long long hash=fingerprint();CHECK(!debugOverflow);
  if(!run){hashes[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;for(int id=0;id<function.virtualRegCount;id++)spills+=allocation.spilled.data[id];}else CHECK(hashes[fi]==hash&&!strcmp(firstText[fi],debugText));
 }
 if(argc==1){CHECK(spills>0);printf("%d plans ms=%.3f frames=%d..%d spills=%d\n",TRIALS,1000.0*(double)(clock()-planning)/CLOCKS_PER_SEC,min_frame,max_frame,spills);}
 for(int run=0;run<(argc>1?0:TRIALS);run++){CHECK(!F(emit_module)(src,(concept_span_byte){again,sizeof again},&other).tag);CHECK(!memcmp(&image,&other,sizeof image)&&!memcmp(code,again,image.byteCount));}
 f=fopen(DUMP,"wb");CHECK(f&&fwrite(code,1,image.byteCount,f)==(size_t)image.byteCount);fclose(f);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memory);memcpy(memory,code,image.byteCount);DWORD old;CHECK(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){SYSTEM_INFO info;GetSystemInfo(&info);size_t page=info.dwPageSize;unsigned char*guard=VirtualAlloc(NULL,page*2,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(guard&&VirtualProtect(guard+page,page,PAGE_NOACCESS,&old));uint32_t tag=argv[1][0]=='o'?0:99;memcpy(guard+page-4,&tag,4);typedef int(*Raw)(const void*);Raw read;void*p=memory+image.symbols.data[ORD_VoidRecordCode].offset;memcpy(&read,&p,sizeof read);if(!tag){CHECK(read(guard+page-4)==8);return 0;}SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);native_begin=memory;native_size=image.byteCount;CHECK(AddVectoredExceptionHandler(1,expected_trap));(void)read(guard+page-4);return 99;}
 for(int trap=0;trap<2;trap++){char exe[MAX_PATH],command[MAX_PATH+32];CHECK(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %s",exe,trap?"invalid":"ok");STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;CHECK(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));CHECK(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;CHECK(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);CHECK(status==0);}
 for(int i=0;i<40;i++)for(int valid=0;valid<2;valid++){
  RInt a=FN(Parse,Parse)(valid,i);CHECK(a.tag==(valid?0:1));CHECK(FN(Code,Code)(a)==(valid?i:-2));CHECK(FN(Code,Code)(FN(Double,Double)(valid,i))==(valid?i*2:-2));
  int calls=0;RInt v=FN(ValidateTwice,ValidateTwice)(valid,&calls);CHECK(FN(VoidCode,Code)(v)==(valid?7:-3)&&calls==(valid?2:1));int voidCalls=calls;
  CHECK(FN(OtherCode,Code)(FN(Other,Parse)(valid,i))==(valid?i:-1));
  CHECK(FN(DoubleValue,DoubleValue)(FN(MakeDouble,MakeDouble)(valid,3.5))==(valid?3.5:-2.0));CHECK(FN(FloatValue,FloatValue)(FN(MakeFloat,MakeFloat)(valid,1.25f))==(valid?1.25f:-1.0f));CHECK(FN(BoolValue,BoolValue)(FN(MakeBool,MakeBool)(valid,1))==valid);
  CHECK(FN(FloatErrorValue,FloatErrorValue)(FN(MakeFloat,MakeFloat)(valid,1.25f))==!valid);CHECK(FN(BoolErrorValue,BoolErrorValue)(FN(MakeBool,MakeBool)(valid,1))==(valid?0:7));
  RPair pair=FN(MakePair,MakePair)(valid,i);CHECK(FN(PairValue,PairValue)(FN(PairCopy,PairCopy)(pair))==(valid?i:-1));CHECK(FN(PairErrorValue,PairErrorValue)(pair)==(valid?0:50000));
  RItem item=FN(MakeItem,MakeItem)(valid,i);Item isolated=FN(ItemValue,ItemValue)(FN(RecordCopy,RecordCopy)(item));CHECK(isolated.count==(valid?i:-1));CHECK(FN(RecordErrorCode,RecordErrorCode)(FN(RecordError,RecordError)(valid,i))==(valid?i:i+1));
  CHECK(FN(ArrayValue,ArrayValue)(FN(MakeArray,MakeArray)(valid))==(valid?12:-1));CHECK(FN(MutateArrayBinding,ArrayValue)(FN(MakeArray,MakeArray)(valid))==(valid?15:-2));CHECK(FN(ArrayErrorCode,ArrayValue)(FN(ArrayError,ArrayError)(valid))==(valid?1:14));CHECK(FN(VoidRecordCode,VoidRecordCode)(FN(VoidRecord,VoidRecord)(valid))==(valid?8:7));
  CHECK(FN(ArrayValue,ArrayValue)(FN(ForwardArray,MakeArray)(valid))==(valid?12:-1));CHECK(FN(ItemValue,ItemValue)(FN(ForwardItem,MakeItem)(valid,i)).count==(valid?i:-1));
  CHECK(FN(LocalCopies,LocalCopies)(i)==2*i+8);CHECK(FN(Code,Code)(FN(Recursive,Recursive)(valid,3))==(valid?7:-2));CHECK(FN(PositiveValue,BoolValue)(FN(Positive,Positive)(valid,i))==(valid&&i>0));
  calls=0;CHECK(FN(Discard,Discard)(&calls)==1&&calls==1);CHECK(FN(Code,Code)(FN(Conditional,Parse)(valid,i))==(valid?(i?9:0):-2));
  int log=0;RInt ordered=FN(ExpressionOrder,Order)(&log,valid,i);CHECK(FN(Code,Code)(ordered)==(valid?i+4:-2)&&log==(valid?123:12));log=0;ordered=FN(ArgumentOrder,Order)(&log,valid,i);CHECK(FN(Code,Code)(ordered)==(valid?i+4:-2)&&log==(valid?123:12));log=0;ordered=FN(Remap,Order)(&log,valid,i);CHECK(FN(OtherCode,Code)(ordered)==(valid?i+1:-2)&&log==(valid?0:1));
  log=0;RItem recordOrder=FN(RecordOrder,RecordOrder)(&log,valid);CHECK(FN(ItemValue,ItemValue)(recordOrder).count==(valid?1:-1)&&log==1);
  calls=0;CHECK(FN(DoubleValue,DoubleValue)(FN(MixedCaller,Caller)(valid,1,&calls))==(valid?5.5:-2)&&calls==valid);calls=0;CHECK(FN(DoubleValue,DoubleValue)(FN(MixedCaller,Caller)(1,valid,&calls))==(valid?5.5:-2)&&calls==valid);
  CHECK(FN(DoubleValue,DoubleValue)(FN(FloatingPressure,Pressure)(1,2,3,4,5,6,7+i))==56.0+2*i);
  calls=0;RDouble fifth=FN(MakeDouble,MakeDouble)(valid,3.5);CHECK(FN(DoubleValue,DoubleValue)(FN(Mixed,Mixed)(a,100,2.0,&calls,fifth,1.25f,1))==(valid?5.5:-2)&&calls==valid);
#ifndef RESULT_NATIVE_ONLY
  CHECK(FN(OtherCode,Code)(FN(Other,Parse)(valid,i))==concept_testing__results_other_code(concept_testing__results_other(valid,i)));
  CHECK(FN(FloatValue,FloatValue)(FN(MakeFloat,MakeFloat)(valid,1.25f))==concept_testing__results_float_value(concept_testing__results_make_float(valid,1.25f)));
  CHECK(FN(BoolErrorValue,BoolErrorValue)(FN(MakeBool,MakeBool)(valid,1))==concept_testing__results_bool_error_value(concept_testing__results_make_bool(valid,1)));
  CHECK(FN(PairValue,PairValue)(FN(PairCopy,PairCopy)(pair))==concept_testing__results_pair_value(concept_testing__results_pair_copy(concept_testing__results_make_pair(valid,i))));
  CHECK(FN(RecordErrorCode,RecordErrorCode)(FN(RecordError,RecordError)(valid,i))==concept_testing__results_record_error_code(concept_testing__results_record_error(valid,i)));
  CHECK(FN(ArrayErrorCode,ArrayValue)(FN(ArrayError,ArrayError)(valid))==concept_testing__results_array_error_code(concept_testing__results_array_error(valid)));
  CHECK(FN(VoidRecordCode,VoidRecordCode)(FN(VoidRecord,VoidRecord)(valid))==concept_testing__results_void_record_code(concept_testing__results_void_record(valid)));
  CHECK(FN(ArrayValue,ArrayValue)(FN(ForwardArray,MakeArray)(valid))==concept_testing__results_array_value(concept_testing__results_forward_array(valid)));
  CHECK(FN(ItemValue,ItemValue)(FN(ForwardItem,MakeItem)(valid,i)).count==concept_testing__results_item_value(concept_testing__results_forward_item(valid,i)).count);
  CHECK(FN(Code,Code)(FN(Double,Double)(valid,i))==concept_testing__results_code(concept_testing__results_double(valid,i)));int oracleCalls=0;CHECK(FN(VoidCode,Code)(v)==concept_testing__results_void_code(concept_testing__results_validate_twice(valid,&oracleCalls))&&voidCalls==oracleCalls);
  int sourceLog=0;log=0;CHECK(FN(Code,Code)(FN(ExpressionOrder,Order)(&log,valid,i))==concept_testing__results_code(concept_testing__results_expression_order(&sourceLog,valid,i))&&log==sourceLog);sourceLog=0;log=0;CHECK(FN(Code,Code)(FN(ArgumentOrder,Order)(&log,valid,i))==concept_testing__results_code(concept_testing__results_argument_order(&sourceLog,valid,i))&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(OtherCode,Code)(FN(Remap,Order)(&log,valid,i))==concept_testing__results_other_code(concept_testing__results_remap(&sourceLog,valid,i))&&log==sourceLog);
  CHECK(FN(ItemValue,ItemValue)(FN(RecordCopy,RecordCopy)(item)).count==concept_testing__results_item_value(concept_testing__results_record_copy(concept_testing__results_make_item(valid,i))).count);CHECK(FN(MutateArrayBinding,ArrayValue)(FN(MakeArray,MakeArray)(valid))==concept_testing__results_mutate_array_binding(concept_testing__results_make_array(valid)));CHECK(FN(LocalCopies,LocalCopies)(i)==concept_testing__results_local_copies(i));
  CHECK(FN(Code,Code)(FN(Recursive,Recursive)(valid,3))==concept_testing__results_code(concept_testing__results_recursive(valid,3)));oracleCalls=0;calls=0;CHECK(FN(DoubleValue,DoubleValue)(FN(MixedCaller,Caller)(valid,1,&calls))==concept_testing__results_double_value(concept_testing__results_mixed_caller(valid,1,&oracleCalls))&&calls==oracleCalls);
  CHECK(FN(DoubleValue,DoubleValue)(FN(FloatingPressure,Pressure)(1,2,3,4,5,6,7+i))==concept_testing__results_double_value(concept_testing__results_floating_pressure(1,2,3,4,5,6,7+i)));
#endif
 }
 for(int fail=-1;fail<5;fail++){int calls=0;RInt value=FN(Loop,Loop)(5,fail,&calls);CHECK(FN(Code,Code)(value)==(fail<0?10:-2)&&calls==(fail<0?5:fail+1));
#ifndef RESULT_NATIVE_ONLY
 int sourceCalls=0;CHECK(FN(Code,Code)(value)==concept_testing__results_code(concept_testing__results_loop(5,fail,&sourceCalls))&&calls==sourceCalls);
#endif
 }
 uint64_t bits64[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};
 for(int i=0;i<4;i++){double value;uint64_t actual;memcpy(&value,&bits64[i],8);double found=FN(DoubleValue,DoubleValue)(FN(DoubleCopy,DoubleCopy)(FN(MakeDouble,MakeDouble)(1,value)));memcpy(&actual,&found,8);CHECK(actual==bits64[i]);found=FN(DoubleErrorValue,DoubleErrorValue)(FN(DoubleError,DoubleError)(0,value));memcpy(&actual,&found,8);CHECK(actual==bits64[i]);
#ifndef RESULT_NATIVE_ONLY
 found=concept_testing__results_double_value(concept_testing__results_double_copy(concept_testing__results_make_double(1,value)));memcpy(&actual,&found,8);CHECK(actual==bits64[i]);found=concept_testing__results_double_error_value(concept_testing__results_double_error(0,value));memcpy(&actual,&found,8);CHECK(actual==bits64[i]);
#endif
 }
 uint32_t bits32[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};
 for(int i=0;i<4;i++){float value,found;uint32_t actual;memcpy(&value,&bits32[i],4);found=FN(FloatValue,FloatValue)(FN(MakeFloat,MakeFloat)(1,value));memcpy(&actual,&found,4);CHECK(actual==bits32[i]);
#ifndef RESULT_NATIVE_ONLY
 found=concept_testing__results_float_value(concept_testing__results_make_float(1,value));memcpy(&actual,&found,4);CHECK(actual==bits32[i]);
#endif
 }
 for(int probe=0;probe_dlls[probe];probe++){
  HMODULE dll=LoadLibraryA(probe_dlls[probe]);CHECK(dll);typedef int(*Probe)(const void*const*);Probe call;FARPROC entry=GetProcAddress(dll,"CallResults");memcpy(&call,&entry,sizeof call);CHECK(call);
  const void*table[]={memory+image.symbols.data[ORD_Parse].offset,memory+image.symbols.data[ORD_MakeDouble].offset,memory+image.symbols.data[ORD_MakeItem].offset,memory+image.symbols.data[ORD_MakeArray].offset,memory+image.symbols.data[ORD_MakePair].offset,memory+image.symbols.data[ORD_Mixed].offset,memory+image.symbols.data[ORD_Validate].offset,memory+image.symbols.data[ORD_Double].offset};CHECK(!call(table));
  typedef RInt(*RelayInt)(RInt);typedef RDouble(*RelayDouble)(RDouble);typedef RPair(*RelayPair)(RPair);RelayInt ri;RelayDouble rd;RelayPair rp;
  entry=GetProcAddress(dll,"ResultIntRelay");memcpy(&ri,&entry,sizeof ri);entry=GetProcAddress(dll,"ResultDoubleRelay");memcpy(&rd,&entry,sizeof rd);entry=GetProcAddress(dll,"ResultPairRelay");memcpy(&rp,&entry,sizeof rp);CHECK(ri&&rd&&rp);
  for(int i=0;i<20;i++){CHECK(FN(Code,Code)(ri(FN(Parse,Parse)(1,i)))==i);CHECK(FN(Code,Code)(ri(FN(Parse,Parse)(0,i)))==-2);CHECK(FN(DoubleValue,DoubleValue)(rd(FN(MakeDouble,MakeDouble)(1,3.5)))==3.5);CHECK(FN(PairValue,PairValue)(rp(FN(MakePair,MakePair)(1,i)))==i);CHECK(FN(PairErrorValue,PairErrorValue)(rp(FN(MakePair,MakePair)(0,i)))==50000);}
  CHECK(FreeLibrary(dll));printf("independent Result ABI passed: %s\n",probe_dlls[probe]);
 }
 CHECK(VirtualFree(memory,0,MEM_RELEASE));
#ifdef RESULT_NATIVE_ONLY
 puts("artifact-only Result native execution passed (no domain C linked)");
#else
 puts("Result native execution and independent C11 parity passed");
#endif
 return 0;
}`
