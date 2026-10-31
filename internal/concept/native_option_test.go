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

func TestEVT2h4NativeOption(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64")
	}
	module, source, _, machine, data := optionFixture(t)
	dir := t.TempDir()
	artifact := filepath.Join(dir, "option.cmir9")
	dump := filepath.Join(dir, "native.bin")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	var defs strings.Builder
	defs.WriteString("static const char*probe_dlls[]={")
	for _, compiler := range []string{"cl", "clang-cl"} {
		if dll := buildIndependentSumProbe(t, compiler, "../../language/evt2h4/probes/options.c"); dll != "" {
			fmt.Fprintf(&defs, "%q,", filepath.ToSlash(dll))
		}
	}
	defs.WriteString("NULL};\n")
	fmt.Fprintf(&defs, "#define ARTIFACT %q\n#define DUMP %q\n#define FUNCTION_COUNT %d\n", filepath.ToSlash(artifact), filepath.ToSlash(dump), len(machine.Pattern.Base.Base.Objects.Functions))
	for i, f := range machine.Pattern.Base.Base.Objects.Functions {
		fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
	}
	host := strings.Replace(optionNativeHost, "/* DEFINITIONS */", defs.String(), 1)
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
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "option_native.c", "#define OPTION_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
			native, err := os.ReadFile(dump)
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = native
			} else if !bytes.Equal(first, native) {
				t.Fatal("Normal/Verify native bytes differ")
			}
			t.Logf("native bytes=%d sha256=%x", len(native), sha256.Sum256(native))
			if got := fmt.Sprintf("%x", sha256.Sum256(native)); got != "5d5b8746bd74e74b50d40f8cc7dbe5e680597c6c00ba732d33be33fa8bc90316" {
				t.Fatalf("exact Option native bytes changed: %s", got)
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
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "option_oracle.c", host, "-pedantic-errors", "-O2"))
			after, err := os.ReadFile(dump)
			if err != nil || !bytes.Equal(native, after) {
				t.Fatalf("oracle changed native bytes: %v", err)
			}
		})
	}
}

const optionNativeHost = `#include "amd64.generated.h"
#ifndef OPTION_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
/* DEFINITIONS */
#ifdef OPTION_NATIVE_ONLY
#define TRIALS 100
#else
#define TRIALS 1
#endif
#define CHECK(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
typedef struct{int count;double amount;} Item;
typedef struct{int tag;int value;} OInt;
typedef struct{int tag;double value;} ODouble;
typedef struct{int tag;float value;} OFloat;
typedef struct{int tag;_Bool value;} OBool;
typedef struct{int tag;uint8_t value;} OByte;
typedef struct{int tag;uint16_t value;} OWord;
typedef struct{int tag;Item value;} OItem;
typedef struct{int tag;int value[3];} OArray;
typedef OInt(*MakeInt)(_Bool,int);typedef ODouble(*MakeDouble)(_Bool,double);typedef OFloat(*MakeFloat)(_Bool,float);typedef OBool(*MakeBool)(_Bool,_Bool);typedef OByte(*MakeByte)(_Bool,uint8_t);typedef OWord(*MakeWord)(_Bool,uint16_t);typedef OItem(*MakeItem)(_Bool,int);typedef OArray(*MakeArray)(_Bool);
typedef int(*GetInt)(OInt);typedef double(*GetDouble)(ODouble);typedef float(*GetFloat)(OFloat);typedef _Bool(*GetBool)(OBool);typedef uint8_t(*GetByte)(OByte);typedef uint16_t(*GetWord)(OWord);typedef Item(*GetItem)(OItem);typedef int(*GetArray)(OArray);
typedef OInt(*Replace)(OInt,int);typedef int(*Local)(int);typedef OItem(*CopyItem)(OItem);typedef OInt(*Loop)(OInt,int);typedef int(*Guard)(OInt,int*);typedef int(*Once)(int*,int);typedef ODouble(*Mixed)(OInt,int,double,int*,ODouble,float,_Bool);typedef double(*Caller)(int,int,int*);typedef OInt(*Match)(OInt);typedef int(*Nested)(OInt,_Bool);typedef ODouble(*CopyDouble)(ODouble);typedef double(*Spill)(double,double,double,double,double,double,double);
static concept_native_image image,other;
static unsigned char artifact[2000000],code[262144],again[262144];
static concept_backend_function function;static concept_object_placement placement;static concept_allocation allocation;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
#define ACCESS(t) static t Resolve##t(unsigned char*memory,int ordinal){t fn;void*p=memory+image.symbols.data[ordinal].offset;memcpy(&fn,&p,sizeof fn);return fn;}
ACCESS(MakeInt) ACCESS(MakeDouble) ACCESS(MakeFloat) ACCESS(MakeBool) ACCESS(MakeByte) ACCESS(MakeWord) ACCESS(MakeItem) ACCESS(MakeArray) ACCESS(GetInt) ACCESS(GetDouble) ACCESS(GetFloat) ACCESS(GetBool) ACCESS(GetByte) ACCESS(GetWord) ACCESS(GetItem) ACCESS(GetArray) ACCESS(Replace) ACCESS(Local) ACCESS(CopyItem) ACCESS(Loop) ACCESS(Guard) ACCESS(Once) ACCESS(Mixed) ACCESS(Caller) ACCESS(Match) ACCESS(Nested) ACCESS(CopyDouble) ACCESS(Spill)
#define FN(n,t) Resolve##t(memory,ORD_##n)
static uint64_t bits64(double v){uint64_t b;memcpy(&b,&v,8);return b;}static uint32_t bits32(float v){uint32_t b;memcpy(&b,&v,4);return b;}
static unsigned char*native_begin;static int native_size;
static LONG CALLBACK expected_trap(PEXCEPTION_POINTERS e){if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&(uintptr_t)e->ExceptionRecord->ExceptionAddress>=(uintptr_t)native_begin&&(uintptr_t)e->ExceptionRecord->ExceptionAddress<(uintptr_t)native_begin+native_size)ExitProcess(0);return EXCEPTION_CONTINUE_SEARCH;}
int main(int argc,char**argv){
 setvbuf(stdout,NULL,_IONBF,0);
 FILE*f=fopen(ARTIFACT,"rb");CHECK(f);size_t n=fread(artifact,1,sizeof artifact,f);fclose(f);concept_readonly_span_byte src={artifact,n};
 concept_result_void_pattern_error admitted=F(validate_pattern_machine_bridge)(src);if(admitted.tag){fprintf(stderr,"%s\n",F(describe_pattern_error)(admitted.payload.error.error));return 1;}
 memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);other=image;
 CHECK(F(emit_module)(src,(concept_span_byte){code,1},&image).tag&&!memcmp(&image,&other,sizeof image));for(size_t i=0;i<sizeof code;i++)CHECK(code[i]==0xa5);
 clock_t start=clock();concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(emitted.tag){fprintf(stderr,"%s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}CHECK(image.finalized);
 printf("Option functions=%d calls=%d bytes=%d emission-ms=%.3f\n",image.symbolCount,image.fixupCount,image.byteCount,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 if(argc==1){
  concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);CHECK(!decoded.tag);concept_wire_object_module doc=decoded.payload.ok.value;
  int min_frame=4096,max_frame=0,spills=0;unsigned long long hashes[FUNCTION_COUNT];clock_t planning=clock();
  for(int run=0;run<TRIALS;run++)for(int fi=0;fi<doc.Functions.count;fi++){
   concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc,fi);CHECK(!owner.tag);
   function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();
   CHECK(!F(prepare_aggregate_function)(&reader,doc,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);
   unsigned long long hash=fingerprint();CHECK(!debugOverflow);
   if(!run){hashes[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;for(int id=0;id<function.virtualRegCount;id++)spills+=allocation.spilled.data[id];}else CHECK(hashes[fi]==hash&&!strcmp(firstText[fi],debugText));
  }
  CHECK(spills>0);printf("%d plans ms=%.3f frames=%d..%d spills=%d\n",TRIALS,1000.0*(double)(clock()-planning)/CLOCKS_PER_SEC,min_frame,max_frame,spills);
  for(int run=0;run<TRIALS;run++){CHECK(!F(emit_module)(src,(concept_span_byte){again,sizeof again},&other).tag);CHECK(!memcmp(&image,&other,sizeof image)&&!memcmp(code,again,image.byteCount));}
 }
 f=fopen(DUMP,"wb");CHECK(f&&fwrite(code,1,image.byteCount,f)==(size_t)image.byteCount);fclose(f);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memory);memcpy(memory,code,image.byteCount);DWORD old;CHECK(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){
  SYSTEM_INFO info;GetSystemInfo(&info);size_t page=info.dwPageSize;unsigned char*guard=VirtualAlloc(NULL,page*2,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(guard&&VirtualProtect(guard+page,page,PAGE_NOACCESS,&old));
  uint32_t tag=argv[1][0]=='n'?1:99;memcpy(guard+page-4,&tag,4);typedef double(*Raw)(const void*);Raw read;void*p=memory+image.symbols.data[ORD_DoubleValue].offset;memcpy(&read,&p,sizeof read);
  if(tag==1){CHECK(read(guard+page-4)==-1.0);return 0;}
  SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);native_begin=memory;native_size=image.byteCount;CHECK(AddVectoredExceptionHandler(1,expected_trap));(void)read(guard+page-4);return 99;
 }
 for(int trap=0;trap<2;trap++){char exe[MAX_PATH],command[MAX_PATH+32];CHECK(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %s",exe,trap?"invalid":"none");STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;CHECK(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));CHECK(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;CHECK(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);CHECK(status==0);}
 for(int i=0;i<100;i++){
  OInt a=FN(MakeInt,MakeInt)(1,i-50),none=FN(MakeInt,MakeInt)(0,123);CHECK(a.tag==0&&none.tag==1&&FN(IntValue,GetInt)(a)==i-50&&FN(IntValue,GetInt)(none)==-1);
  OInt copy=FN(Replace,Replace)(a,i);CHECK(copy.tag==0&&copy.value==i&&a.value==i-50);CHECK(FN(LocalCopies,Local)(i)==2*i+9);
  OItem item=FN(MakeItem,MakeItem)(1,i);Item record=FN(ItemValue,GetItem)(FN(RecordCopy,CopyItem)(item));CHECK(record.count==i&&record.amount==3.5);
  OArray array=FN(MakeArray,MakeArray)(1),noarray=FN(MakeArray,MakeArray)(0);CHECK(FN(ArrayValue,GetArray)(array)==12&&FN(ArrayValue,GetArray)(noarray)==-1&&FN(MutateArrayBinding,GetArray)(array)==111&&array.value[0]==3);
  CHECK(FN(BoolValue,GetBool)(FN(MakeBool,MakeBool)(1,i%2))==(i%2));CHECK(!FN(BoolValue,GetBool)(FN(MakeBool,MakeBool)(0,1)));CHECK(FN(ByteValue,GetByte)(FN(MakeByte,MakeByte)(1,(uint8_t)i))==i);CHECK(FN(WordValue,GetWord)(FN(MakeWord,MakeWord)(1,(uint16_t)(40000+i)))==40000+i);
  CHECK(FN(IntValue,GetInt)(FN(AcrossBranches,Loop)(FN(MakeInt,MakeInt)(1,4),3))==7);CHECK(FN(IntValue,GetInt)(FN(Recursive,Loop)(none,3))==2);
  CHECK(FN(IntValue,GetInt)(FN(MatchResult,Match)(a))==i-49&&FN(MatchResult,Match)(none).tag==1);CHECK(FN(Nested,Nested)(a,1)==i-50&&FN(Nested,Nested)(a,0)==0&&FN(Nested,Nested)(none,1)==-1);
  int log=0,calls=0;CHECK(FN(Guarded,Guard)(FN(MakeInt,MakeInt)(1,7),&log)==7&&log==12);log=0;CHECK(FN(Guarded,Guard)(none,&log)==-2&&log==0);CHECK(FN(Once,Once)(&calls,i)==i&&calls==1);
  ODouble d=FN(MakeDouble,MakeDouble)(1,3.5);calls=0;CHECK(FN(DoubleValue,GetDouble)(FN(Mixed,Mixed)(a,101,2.0,&calls,d,1.25f,1))==5.5&&calls==1);calls=0;CHECK(FN(MixedCaller,Caller)(10,20,&calls)==5.5&&calls==1);
  CHECK(FN(SpillMatch,Spill)(1,2,3,4,5,6,7+i)==56.0+2*i);
#ifndef OPTION_NATIVE_ONLY
  CHECK(FN(LocalCopies,Local)(i)==concept_testing__options_local_copies(i));CHECK(FN(ArrayValue,GetArray)(array)==concept_testing__options_array_value(concept_testing__options_make_array(1)));int sourceLog=0;log=0;CHECK(FN(Guarded,Guard)(a,&log)==concept_testing__options_guarded(concept_testing__options_make_int(1,i-50),&sourceLog)&&log==sourceLog);CHECK(FN(SpillMatch,Spill)(1,2,3,4,5,6,7+i)==concept_testing__options_spill_match(1,2,3,4,5,6,7+i));
  CHECK(FN(IntValue,GetInt)(a)==concept_testing__options_int_value(concept_testing__options_make_int(1,i-50)));CHECK(copy.value==concept_testing__options_int_value(concept_testing__options_replace(concept_testing__options_make_int(1,i-50),i)));
  CHECK(record.count==concept_testing__options_item_value(concept_testing__options_record_copy(concept_testing__options_make_item(1,i))).count);CHECK(FN(MutateArrayBinding,GetArray)(array)==concept_testing__options_mutate_array_binding(concept_testing__options_make_array(1)));
  CHECK(FN(IntValue,GetInt)(FN(AcrossBranches,Loop)(FN(MakeInt,MakeInt)(1,4),3))==concept_testing__options_int_value(concept_testing__options_across_branches(concept_testing__options_make_int(1,4),3)));CHECK(FN(IntValue,GetInt)(FN(Recursive,Loop)(none,3))==concept_testing__options_int_value(concept_testing__options_recursive(concept_testing__options_make_int(0,123),3)));
  CHECK(FN(IntValue,GetInt)(FN(MatchResult,Match)(a))==concept_testing__options_int_value(concept_testing__options_match_result(concept_testing__options_make_int(1,i-50))));CHECK(FN(Nested,Nested)(a,1)==concept_testing__options_nested(concept_testing__options_make_int(1,i-50),1));
  int sourceCalls=0;calls=0;CHECK(FN(MixedCaller,Caller)(10,20,&calls)==concept_testing__options_mixed_caller(10,20,&sourceCalls)&&calls==sourceCalls);sourceCalls=0;calls=0;CHECK(FN(Once,Once)(&calls,i)==concept_testing__options_once(&sourceCalls,i)&&calls==sourceCalls);
  CHECK(FN(BoolValue,GetBool)(FN(MakeBool,MakeBool)(1,i%2))==concept_testing__options_bool_value(concept_testing__options_make_bool(1,i%2)));CHECK(FN(ByteValue,GetByte)(FN(MakeByte,MakeByte)(1,(uint8_t)i))==concept_testing__options_byte_value(concept_testing__options_make_byte(1,(uint8_t)i)));CHECK(FN(WordValue,GetWord)(FN(MakeWord,MakeWord)(1,(uint16_t)(40000+i)))==concept_testing__options_word_value(concept_testing__options_make_word(1,(uint16_t)(40000+i))));
#endif
 }
 uint64_t inputs64[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};
 for(int i=0;i<4;i++){double d;memcpy(&d,&inputs64[i],8);CHECK(bits64(FN(DoubleValue,GetDouble)(FN(FloatCopy,CopyDouble)(FN(MakeDouble,MakeDouble)(1,d))))==inputs64[i]);
#ifndef OPTION_NATIVE_ONLY
 CHECK(bits64(concept_testing__options_double_value(concept_testing__options_float_copy(concept_testing__options_make_double(1,d))))==inputs64[i]);
#endif
 }
 uint32_t inputs32[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};
 for(int i=0;i<4;i++){float d;memcpy(&d,&inputs32[i],4);CHECK(bits32(FN(FloatValue,GetFloat)(FN(MakeFloat,MakeFloat)(1,d)))==inputs32[i]);
#ifndef OPTION_NATIVE_ONLY
 CHECK(bits32(concept_testing__options_float_value(concept_testing__options_make_float(1,d)))==inputs32[i]);
#endif
 }
 for(int probe=0;probe_dlls[probe];probe++){
  HMODULE dll=LoadLibraryA(probe_dlls[probe]);CHECK(dll);typedef int(*CallOptions)(const void*const*);CallOptions call;FARPROC entry=GetProcAddress(dll,"CallOptions");memcpy(&call,&entry,sizeof call);CHECK(call);
  const void*table[]={memory+image.symbols.data[ORD_MakeInt].offset,memory+image.symbols.data[ORD_MakeDouble].offset,memory+image.symbols.data[ORD_MakeFloat].offset,memory+image.symbols.data[ORD_MakeBool].offset,memory+image.symbols.data[ORD_MakeItem].offset,memory+image.symbols.data[ORD_MakeArray].offset,memory+image.symbols.data[ORD_Mixed].offset};CHECK(!call(table));
  typedef OInt(*RI)(OInt);typedef ODouble(*RD)(ODouble);typedef OFloat(*RF)(OFloat);typedef OBool(*RB)(OBool);typedef OItem(*RR)(OItem);typedef OArray(*RA)(OArray);
  RI ri;RD rd;RF rf;RB rb;RR rr;RA ra;
  entry=GetProcAddress(dll,"OptionIntRelay");memcpy(&ri,&entry,sizeof ri);entry=GetProcAddress(dll,"OptionDoubleRelay");memcpy(&rd,&entry,sizeof rd);entry=GetProcAddress(dll,"OptionFloatRelay");memcpy(&rf,&entry,sizeof rf);entry=GetProcAddress(dll,"OptionBoolRelay");memcpy(&rb,&entry,sizeof rb);entry=GetProcAddress(dll,"OptionItemRelay");memcpy(&rr,&entry,sizeof rr);entry=GetProcAddress(dll,"OptionArrayRelay");memcpy(&ra,&entry,sizeof ra);CHECK(ri&&rd&&rf&&rb&&rr&&ra);
  for(int i=0;i<100;i++){CHECK(FN(IntValue,GetInt)(ri(FN(MakeInt,MakeInt)(1,i)))==i);CHECK(FN(DoubleValue,GetDouble)(rd(FN(MakeDouble,MakeDouble)(1,3.5)))==3.5);CHECK(FN(FloatValue,GetFloat)(rf(FN(MakeFloat,MakeFloat)(1,1.25f)))==1.25f);CHECK(FN(BoolValue,GetBool)(rb(FN(MakeBool,MakeBool)(1,1))));CHECK(FN(ItemValue,GetItem)(rr(FN(MakeItem,MakeItem)(1,i))).count==i);CHECK(FN(ArrayValue,GetArray)(ra(FN(MakeArray,MakeArray)(1)))==12);}
  CHECK(FreeLibrary(dll));printf("independent Option ABI probe passed: %s\n",probe_dlls[probe]);
 }
 VirtualFree(memory,0,MEM_RELEASE);puts("Option native execution, active None page protection, invalid-tag trap and C11 parity passed");return 0;
}`
