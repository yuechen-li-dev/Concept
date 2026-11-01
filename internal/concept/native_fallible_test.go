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

const fallibleFixturePath = "../../tests/runtime/Testing/FallibleControl.concept"

func fallibleFixture(t *testing.T) (Module, []byte, LIRModule, MachineModule, []byte) {
	t.Helper()
	source, err := os.ReadFile(fallibleFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(fallibleFixturePath, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	return module, source, lir, machine, data
}

func TestEVT2h7NativeFallible(t *testing.T) {
	if resultSamples() == 100 {
		t.Setenv("CONCEPT_TEST_NATIVE_TIMEOUT", "8m")
	}
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64")
	}
	module, source, _, machine, data := fallibleFixture(t)
	dir := t.TempDir()
	artifact := filepath.Join(dir, "fallible.cmir10")
	dump := filepath.Join(dir, "native.bin")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	var defs strings.Builder
	fmt.Fprintf(&defs, "#define ARTIFACT %q\n#define DUMP %q\n#define MARKER %q\n#define FUNCTION_COUNT %d\n#ifdef FALLIBLE_NATIVE_ONLY\n#define TRIALS %d\n#else\n#define TRIALS 1\n#endif\n", filepath.ToSlash(artifact), filepath.ToSlash(dump), filepath.ToSlash(filepath.Join(dir, "trap")), len(machine.Fallible.Base.Base.Base.Objects.Functions), resultSamples())
	for i, f := range machine.Fallible.Base.Base.Base.Objects.Functions {
		fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
	}
	host := strings.Replace(fallibleNativeHost, "/* DEFINITIONS */", defs.String(), 1)
	a, b := strings.Index(aggregateABIPlanningHost, "static unsigned long long digest;"), strings.Index(aggregateABIPlanningHost, "static int run_plan(")
	host = strings.Replace(host, "/* PLAN DEBUG */", aggregateABIPlanningHost[a:b], 1)
	host = phaseFreshNativeHost(t, host, data, "fallible", func() []byte { _, _, _, _, raw := fallibleFixture(t); return raw })
	var first []byte
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)

			t.Log(runFoundationNativeHarnessOutput(t, outputs, "fallible_native.c", "#define FALLIBLE_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
			code, err := os.ReadFile(dump)
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = code
			} else if !bytes.Equal(first, code) {
				t.Fatal("Normal/Verify bytes differ")
			}
			t.Logf("image=%d sha256=%x artifact=%d sha256=%x", len(code), sha256.Sum256(code), len(data), sha256.Sum256(data))
			if got := fmt.Sprintf("%x", sha256.Sum256(code)); got != "78407eeb465a416e7a406d05d3e382cb9c8c21049f8162c4e7d504196907c242" {
				t.Fatalf("exact fallible native bytes changed: %s", got)
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
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "fallible_oracle.c", host, "-pedantic-errors", "-O2"))
			after, err := os.ReadFile(dump)
			if err != nil || !bytes.Equal(code, after) {
				t.Fatalf("oracle changed code: %v", err)
			}
		})
	}
}

const fallibleNativeHost = `#include "amd64.generated.h"
#ifndef FALLIBLE_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <stdlib.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#define CHECK(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
/* DEFINITIONS */
typedef int(*Require)(_Bool,int*);typedef int(*Local)(_Bool,_Bool,int*);typedef int(*Loop)(int,int,int*);typedef int(*Order)(int*);
typedef struct{uint32_t tag;int value;} NOption;typedef NOption(*Forward)(_Bool,int*);typedef int(*OptionCode)(NOption);
typedef struct{int first,second;} NPair;typedef NPair*(*PairBang)(NPair*,_Bool,int);typedef int(*PairLocal)(_Bool,int);
typedef struct{int count;double amount;} NItem;typedef NItem(*ItemLocal)(_Bool,int);
typedef double(*Real)(_Bool,double);typedef double(*MixedCaller)(_Bool,int*);typedef double(*Pressure)(double,double,double,double,double,double,double);typedef int(*Recursive)(int,_Bool,int*);
static concept_native_image image,other;
static unsigned char artifact[2000000],wire[2000000],code[262144],again[262144];
static concept_backend_function function;static concept_object_placement placement;static concept_allocation allocation;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
#define ACCESS(t) static t Resolve##t(unsigned char*memory,int ordinal){t fn;void*p=memory+image.symbols.data[ordinal].offset;memcpy(&fn,&p,sizeof fn);return fn;}
typedef float(*SmallReal)(_Bool,float);ACCESS(SmallReal)
ACCESS(Require) ACCESS(Local) ACCESS(Loop) ACCESS(Order) ACCESS(Forward) ACCESS(OptionCode) ACCESS(PairBang) ACCESS(PairLocal) ACCESS(ItemLocal) ACCESS(MixedCaller) ACCESS(Real) ACCESS(Pressure) ACCESS(Recursive)
#define FN(n,t) Resolve##t(memory,ORD_##n)
static unsigned char*native_begin;static int native_size,trap_log,trap_number;
static LONG CALLBACK expected_trap(PEXCEPTION_POINTERS e){
 uintptr_t address=(uintptr_t)e->ExceptionRecord->ExceptionAddress;
 if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&address>=(uintptr_t)native_begin&&address+2<=(uintptr_t)native_begin+native_size){
  unsigned char*p=(unsigned char*)address;
  if(p[0]==0x0f&&p[1]==0x0b){char path[MAX_PATH];snprintf(path,sizeof path,"%s.%d",MARKER,trap_number);HANDLE file=CreateFileA(path,GENERIC_WRITE,0,NULL,CREATE_ALWAYS,FILE_ATTRIBUTE_NORMAL,NULL);if(file!=INVALID_HANDLE_VALUE){DWORD written;int record[2]={trap_log,1};WriteFile(file,record,sizeof record,&written,NULL);CloseHandle(file);TerminateProcess(GetCurrentProcess(),EXCEPTION_ILLEGAL_INSTRUCTION);}}
 }
 return EXCEPTION_CONTINUE_SEARCH;
}
static const char*stage="entry";
static LONG CALLBACK unexpected(PEXCEPTION_POINTERS e){fprintf(stderr,"unexpected exception %lx at image offset %lld stage %s\n",(unsigned long)e->ExceptionRecord->ExceptionCode,(long long)((uintptr_t)e->ExceptionRecord->ExceptionAddress-(uintptr_t)native_begin),stage);ExitProcess(94);return EXCEPTION_CONTINUE_SEARCH;}
int main(int argc,char**argv){
 setvbuf(stdout,NULL,_IONBF,0);fprintf(stderr,"admitting fallible artifact\n");
 FILE*f=fopen(ARTIFACT,"rb");CHECK(f);size_t n=fread(artifact,1,sizeof artifact,f);fclose(f);concept_readonly_span_byte src={artifact,n};
 concept_result_void_fallible_error admitted=F(validate_fallible_machine_bridge)(src);if(admitted.tag){fprintf(stderr,"%s\n",F(describe_fallible_error)(admitted.payload.error.error));return 1;}
 memset(wire,0xa5,sizeof wire);CHECK(F(validated_fallible_bridge_round_trip)(src,(concept_span_byte){wire,1}).tag);for(size_t i=0;i<sizeof wire;i++)CHECK(wire[i]==0xa5);
 concept_result_int_fallible_error roundtrip=F(validated_fallible_bridge_round_trip)(src,(concept_span_byte){wire,sizeof wire});CHECK(!roundtrip.tag&&roundtrip.payload.ok.value==(int)n&&!memcmp(wire,artifact,n));
 fprintf(stderr,"admitted; checking output atomicity\n");
 memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);other=image;
 CHECK(F(emit_module)(src,(concept_span_byte){code,1},&image).tag&&!memcmp(&image,&other,sizeof image));for(size_t i=0;i<sizeof code;i++)CHECK(code[i]==0xa5);
 clock_t start=clock();concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(emitted.tag){fprintf(stderr,"%s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}CHECK(image.finalized);
 printf("fallible functions=%d calls=%d bytes=%d emission-ms=%.3f\n",image.symbolCount,image.fixupCount,image.byteCount,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);CHECK(!decoded.tag);concept_wire_object_module doc=decoded.payload.ok.value;
 fprintf(stderr,"emitted; planning\n");
 int min_frame=4096,max_frame=0,spills=0;unsigned long long hashes[FUNCTION_COUNT];clock_t planning=clock();
 for(int run=0;run<(argc>1?0:TRIALS);run++)for(int fi=0;fi<doc.Functions.count;fi++){
  concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc,fi);CHECK(!owner.tag);
  function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();
  CHECK(!F(prepare_aggregate_function)(&reader,doc,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);
  unsigned long long hash=fingerprint();CHECK(!debugOverflow);
  if(!run){hashes[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;for(int id=0;id<function.virtualRegCount;id++)spills+=allocation.spilled.data[id];}else CHECK(hashes[fi]==hash&&!strcmp(firstText[fi],debugText));
 }
 if(argc==1){CHECK(spills>0);printf("%d fresh plans ms=%.3f frames=%d..%d spills=%d\n",TRIALS,1000.0*(double)(clock()-planning)/CLOCKS_PER_SEC,min_frame,max_frame,spills);}
 for(int run=0;run<(argc>1?0:TRIALS);run++){CHECK(!F(emit_module)(src,(concept_span_byte){again,sizeof again},&other).tag);CHECK(!memcmp(&image,&other,sizeof image)&&!memcmp(code,again,image.byteCount));}
 if(argc==1){f=fopen(DUMP,"wb");CHECK(f&&fwrite(code,1,image.byteCount,f)==(size_t)image.byteCount);fclose(f);}
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memory);memcpy(memory,code,image.byteCount);DWORD old;CHECK(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){trap_number=atoi(argv[1]);SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);native_begin=memory;native_size=image.byteCount;CHECK(AddVectoredExceptionHandler(1,expected_trap));
  switch(trap_number){case 0:(void)FN(RequireResult,Require)(0,&trap_log);break;case 1:(void)FN(RequireOption,Require)(0,&trap_log);break;case 2:(void)FN(RequireVoid,Require)(0,&trap_log);break;case 3:(void)FN(BangIgnoresHandler,Require)(0,&trap_log);break;case 4:(void)FN(BangStopsArguments,Order)(&trap_log);break;default:return 98;}return 99;
 }
 fprintf(stderr,"executing isolated traps\n");
 for(int trap=0;trap<5;trap++){char exe[MAX_PATH],command[MAX_PATH+32],marker[MAX_PATH];snprintf(marker,sizeof marker,"%s.%d",MARKER,trap);DeleteFileA(marker);CHECK(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %d",exe,trap);STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;CHECK(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));CHECK(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;CHECK(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);CHECK(status==EXCEPTION_ILLEGAL_INSTRUCTION);int record[2]={0,0};f=fopen(marker,"rb");CHECK(f&&fread(record,sizeof record,1,f)==1);fclose(f);CHECK(record[1]==1&&record[0]==(trap==1?3:trap==2?4:trap==4?21:1));}
 puts("5 isolated UD2 traps: exact Windows status 0xC000001D, instruction, operand effects and skipped handlers verified");
 native_begin=memory;native_size=image.byteCount;SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);CHECK(AddVectoredExceptionHandler(1,unexpected));
 fprintf(stderr,"executing successful and handled calls\n");
 for(int i=0;i<20;i++)for(int valid=0;valid<2;valid++){
  int log=0;stage="RequireResult";CHECK(FN(RequireResult,Require)(1,&log)==14&&log==15);log=0;stage="RequireOption";CHECK(FN(RequireOption,Require)(1,&log)==7&&log==35);log=0;stage="RequireVoid";CHECK(FN(RequireVoid,Require)(1,&log)==9&&log==45);
  for(int otherValid=0;otherValid<2;otherValid++){
   log=0;int found=FN(Local,Local)(valid,otherValid,&log);CHECK(found==(valid?(otherValid?5:20):10)&&log==(valid?(otherValid?1258:1278):168));
   log=0;stage="Nested";CHECK(FN(Nested,Local)(valid,otherValid,&log)==(valid?(otherValid?5:20):10));
   log=0;stage="Nearest";CHECK(FN(Nearest,Local)(valid,otherValid,&log)==(otherValid?(valid?6:34):40));
   log=0;stage="HandlerFailure";CHECK(FN(HandlerFailure,Local)(valid,otherValid,&log)==(valid?2:(otherValid?9:60)));
#ifndef FALLIBLE_NATIVE_ONLY
   int oracleLog=0;log=0;stage="Local";CHECK(FN(Local,Local)(valid,otherValid,&log)==concept_testing__fallible_control_local(valid,otherValid,&oracleLog)&&log==oracleLog);
   oracleLog=0;log=0;stage="Nested";CHECK(FN(Nested,Local)(valid,otherValid,&log)==concept_testing__fallible_control_nested(valid,otherValid,&oracleLog)&&log==oracleLog);
   oracleLog=0;log=0;stage="Nearest";CHECK(FN(Nearest,Local)(valid,otherValid,&log)==concept_testing__fallible_control_nearest(valid,otherValid,&oracleLog)&&log==oracleLog);
   oracleLog=0;log=0;stage="HandlerFailure";CHECK(FN(HandlerFailure,Local)(valid,otherValid,&log)==concept_testing__fallible_control_handler_failure(valid,otherValid,&oracleLog)&&log==oracleLog);
   oracleLog=0;log=0;CHECK(FN(MatchTry,Local)(valid,otherValid,&log)==concept_testing__fallible_control_match_try(valid,otherValid,&oracleLog)&&log==oracleLog);
#endif
  }
  log=0;stage="VoidLocal";CHECK(FN(VoidLocal,Require)(valid,&log)==(valid?7:9)&&log==4);
  log=0;CHECK(FN(OptionCode,OptionCode)(FN(OptionInsideTry,Forward)(valid,&log))==(valid?9:-1)&&log==13);
  log=0;CHECK(FN(OptionCode,OptionCode)(FN(ForwardOption,Forward)(valid,&log))==(valid?8:-1)&&log==(valid?35:3));
  log=0;stage="RemapCode";CHECK(FN(RemapCode,Require)(valid,&log)==(valid?13:-2)&&log==(valid?11:1));
  log=0;stage="BangOrder";CHECK(FN(BangOrder,Order)(&log)==12&&log==213);
  log=0;CHECK(FN(OptionCode,OptionCode)(FN(OptionOrder,Forward)(valid,&log))==(valid?13:-1)&&log==(valid?234:23));
  log=0;stage="Recursive";CHECK(FN(Recursive,Recursive)(3,valid,&log)==(valid?8:13));
  stage="PairLocal";CHECK(FN(PairLocal,PairLocal)(valid,i)==(valid?2*i+1:2*i+2));NPair pair;CHECK(FN(PairBang,PairBang)(&pair,1,i)==&pair);CHECK(pair.first==i&&pair.second==i+1);
  log=0;CHECK(FN(BoundaryCode,Local)(valid,1,&log)==(valid?5:-1));log=0;CHECK(FN(Conditional,Local)(1,valid,&log)==(valid?7:10));log=0;CHECK(FN(MixedCaller,MixedCaller)(valid,&log)==(valid?8.0:4.5)&&log==1);
  NItem item=FN(ItemLocal,ItemLocal)(valid,i);CHECK(item.count==(valid?i:i+1)&&item.amount==(valid?3.5:4.5));
  stage="Pressure";CHECK(FN(Pressure,Pressure)(1,2,3,4,5,6,7)==56.0);
#ifndef FALLIBLE_NATIVE_ONLY
  int sourceLog=0;log=0;CHECK(FN(RequireResult,Require)(1,&log)==concept_testing__fallible_control_require_result(1,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;stage="RequireOption";CHECK(FN(RequireOption,Require)(1,&log)==concept_testing__fallible_control_require_option(1,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;stage="RequireVoid";CHECK(FN(RequireVoid,Require)(1,&log)==concept_testing__fallible_control_require_void(1,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;stage="VoidLocal";CHECK(FN(VoidLocal,Require)(valid,&log)==concept_testing__fallible_control_void_local(valid,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(OptionCode,OptionCode)(FN(OptionInsideTry,Forward)(valid,&log))==concept_testing__fallible_control_option_code(concept_testing__fallible_control_option_inside_try(valid,&sourceLog))&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(OptionCode,OptionCode)(FN(ForwardOption,Forward)(valid,&log))==concept_testing__fallible_control_option_code(concept_testing__fallible_control_forward_option(valid,&sourceLog))&&log==sourceLog);
  sourceLog=0;log=0;stage="RemapCode";CHECK(FN(RemapCode,Require)(valid,&log)==concept_testing__fallible_control_remap_code(valid,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;stage="BangOrder";CHECK(FN(BangOrder,Order)(&log)==concept_testing__fallible_control_bang_order(&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(OptionCode,OptionCode)(FN(OptionOrder,Forward)(valid,&log))==concept_testing__fallible_control_option_code(concept_testing__fallible_control_option_order(valid,&sourceLog))&&log==sourceLog);
  sourceLog=0;log=0;stage="Recursive";CHECK(FN(Recursive,Recursive)(3,valid,&log)==concept_testing__fallible_control_recursive(3,valid,&sourceLog)&&log==sourceLog);
  stage="PairLocal";CHECK(FN(PairLocal,PairLocal)(valid,i)==concept_testing__fallible_control_pair_local(valid,i));CHECK(pair.first==concept_testing__fallible_control_pair_bang(1,i).first&&pair.second==concept_testing__fallible_control_pair_bang(1,i).second);
  sourceLog=0;log=0;CHECK(FN(BoundaryCode,Local)(valid,1,&log)==concept_testing__fallible_control_boundary_code(valid,1,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(Conditional,Local)(1,valid,&log)==concept_testing__fallible_control_conditional(1,valid,&sourceLog)&&log==sourceLog);
  sourceLog=0;log=0;CHECK(FN(MixedCaller,MixedCaller)(valid,&log)==concept_testing__fallible_control_mixed_caller(valid,&sourceLog)&&log==sourceLog);
  CHECK(item.count==concept_testing__fallible_control_item_local(valid,i).count&&item.amount==concept_testing__fallible_control_item_local(valid,i).amount);
  stage="Pressure";CHECK(FN(Pressure,Pressure)(1,2,3,4,5,6,7)==concept_testing__fallible_control_pressure(1,2,3,4,5,6,7));
#endif
 }
 for(int fail=-1;fail<5;fail++){int log=0;stage="Loop";CHECK(FN(Loop,Loop)(5,fail,&log)==(fail<0?10:20-fail)&&log==11111);
#ifndef FALLIBLE_NATIVE_ONLY
 int sourceLog=0;log=0;stage="Loop";CHECK(FN(Loop,Loop)(5,fail,&log)==concept_testing__fallible_control_loop(5,fail,&sourceLog)&&log==sourceLog);
#endif
 }
 uint64_t bits[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};
 for(int i=0;i<4;i++){double value,found;uint64_t actual;memcpy(&value,&bits[i],8);found=FN(RealBang,Real)(1,value);memcpy(&actual,&found,8);CHECK(actual==bits[i]);for(int valid=0;valid<2;valid++){found=FN(RealLocal,Real)(valid,value);memcpy(&actual,&found,8);CHECK(actual==bits[i]);
#ifndef FALLIBLE_NATIVE_ONLY
 found=concept_testing__fallible_control_real_local(valid,value);memcpy(&actual,&found,8);CHECK(actual==bits[i]);found=concept_testing__fallible_control_real_bang(1,value);memcpy(&actual,&found,8);CHECK(actual==bits[i]);
#endif
 }}
 uint32_t bits32[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};
 for(int i=0;i<4;i++)for(int valid=0;valid<2;valid++){float value,found;uint32_t actual;memcpy(&value,&bits32[i],4);found=FN(SmallRealLocal,SmallReal)(valid,value);memcpy(&actual,&found,4);CHECK(actual==bits32[i]);
#ifndef FALLIBLE_NATIVE_ONLY
 found=concept_testing__fallible_control_small_real_local(valid,value);memcpy(&actual,&found,4);CHECK(actual==bits32[i]);
#endif
 }
 CHECK(VirtualFree(memory,0,MEM_RELEASE));
#ifdef FALLIBLE_NATIVE_ONLY
 puts("artifact-only fallible native execution passed (no domain C linked)");
#else
 puts("native fallible control flow and independent strict C11 parity passed");
#endif
 return 0;
}`
