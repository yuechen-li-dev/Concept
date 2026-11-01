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
	"time"
)

func TestEVT2h8NativeIntegration(t *testing.T) {
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64")
	}
	if integrationSamples() == 100 {
		t.Setenv("CONCEPT_TEST_NATIVE_TIMEOUT", "10m")
	}
	specimens := integrationSpecimens(t)
	dir := t.TempDir()
	var defs strings.Builder
	fmt.Fprintf(&defs, "#define FUNCTION_COUNT 64\n#define TRIALS %d\n#define MARKER %q\n", integrationSamples(), filepath.ToSlash(filepath.Join(dir, "trap")))
	defs.WriteString("static const char*artifact_paths[]={")
	for _, s := range specimens {
		p := filepath.Join(dir, s.name+".cmir10")
		if err := os.WriteFile(p, s.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&defs, "%q,", filepath.ToSlash(p))
		for fi, f := range s.machine.Fallible.Base.Base.Base.Objects.Functions {
			fmt.Fprintf(&defs, "\n#define ORD_%s_%s %d\n", s.name, f.Name, fi)
		}
	}
	defs.WriteString("};\nstatic const char*dump_paths[]={")
	for _, s := range specimens {
		fmt.Fprintf(&defs, "%q,", filepath.ToSlash(filepath.Join(dir, s.name+".bin")))
	}
	defs.WriteString("};\n")
	defs.WriteString("static const char*fresh_paths[TRIALS][4]={\n")
	for trial := 0; trial < integrationSamples(); trial++ {
		defs.WriteString("{")
		for i, s := range integrationSpecimens(t) {
			if !bytes.Equal(s.data, specimens[i].data) || s.lir.String() != specimens[i].lir.String() || s.machine.String() != specimens[i].machine.String() {
				t.Fatalf("fresh integration trial %d %s differs", trial, s.name)
			}
			path := filepath.Join(dir, fmt.Sprintf("%s-fresh-%d.cmir10", s.name, trial))
			if err := os.WriteFile(path, s.data, 0600); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&defs, "%q,", filepath.ToSlash(path))
		}
		defs.WriteString("},\n")
	}
	defs.WriteString("};\n")
	host := strings.Replace(integrationNativeHost, "/* DEFINITIONS */", defs.String(), 1)
	a, b := strings.Index(aggregateABIPlanningHost, "static unsigned long long digest;"), strings.Index(aggregateABIPlanningHost, "static int run_plan(")
	host = strings.Replace(host, "/* PLAN DEBUG */", aggregateABIPlanningHost[a:b], 1)
	var first [][]byte
	start := time.Now()
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "integration_native.c", "#define INTEGRATION_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
			var emitted [][]byte
			want := map[string]string{
				"Parser":     "a7b2383995d4d64d7b6ec1d33935865580806319f7743fda6657ec1e788bff35",
				"Controller": "e23ab5e5ffdb59bc3421f1e02c4184743816ad7746207450371c45d14aa23887",
				"Assertions": "951e78d24abe10bc965982415e74f70b6bccabdcbba940c44a38d27c2978dbde",
				"Plain":      "951e78d24abe10bc965982415e74f70b6bccabdcbba940c44a38d27c2978dbde",
			}
			for _, s := range specimens {
				code, err := os.ReadFile(filepath.Join(dir, s.name+".bin"))
				if err != nil {
					t.Fatal(err)
				}
				if s.name == "Controller" {
					if len(code) <= 52892 || fmt.Sprintf("%x", sha256.Sum256(code[:52892])) != "045a90da6389f2881769582a3c109ba108556812de5d4d79964454168d4348bf" {
						t.Fatal("qualified controller prefix changed")
					}
				}
				if got := fmt.Sprintf("%x", sha256.Sum256(code)); got != want[s.name] {
					t.Fatalf("%s native bytes changed: %s", s.name, got)
				}
				emitted = append(emitted, code)
				t.Logf("%s native=%d sha256=%x artifact=%d sha256=%x", s.name, len(code), sha256.Sum256(code), len(s.data), sha256.Sum256(s.data))
			}
			if !bytes.Equal(emitted[2], emitted[3]) {
				t.Fatal("assertion changed native bytes")
			}
			if first == nil {
				first = emitted
			} else {
				for i := range first {
					if !bytes.Equal(first[i], emitted[i]) {
						t.Fatal("Normal/Verify image mismatch", specimens[i].name)
					}
				}
			}
			policy := ConservativeCompilationPolicy()
			if verify {
				policy = VerifyCompilationPolicy()
			}
			for _, s := range specimens[:3] {
				oracle, err := GenerateForTargetWithPolicy(s.module, s.source, GenericC11Target(), policy)
				if err != nil {
					t.Fatal(err)
				}
				for name, body := range oracle {
					if strings.HasSuffix(name, ".generated.c") {
						body = append(body, []byte(integrationOracleWrappers[s.name])...)
					}
					outputs[name] = body
				}
			}
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "integration_oracle.c", host, "-pedantic-errors", "-O2"))
			for i, s := range specimens {
				after, err := os.ReadFile(filepath.Join(dir, s.name+".bin"))
				if err != nil || !bytes.Equal(after, emitted[i]) {
					t.Fatal("oracle changed image", s.name, err)
				}
			}
		})
	}
	t.Logf("batched native-only/C11/Normal/Verify integration wall=%s; backend builds reused within each mode", time.Since(start))
}

const integrationNativeHost = `#include "amd64.generated.h"
#ifndef INTEGRATION_NATIVE_ONLY
extern void oracle_parser(const int*,int,int,int*,int*);
extern void oracle_controller(int,int,int*,int*);
extern int oracle_assertion(_Bool,int);
extern int oracle_logic(int,int,int,int,int*);
#endif
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#define CHECK(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
/* DEFINITIONS */
static unsigned char artifacts[4][2000000],code[4][262144],wire[2000000],again[262144];
static unsigned char*artifact;
static concept_native_image images[4],other,before;
static unsigned char*memories[4];
static concept_backend_function function;static concept_object_placement placement;static concept_allocation allocation;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
typedef struct{int words[8];int length;} NInput;
typedef struct{int status,total,commands,position,actual;} NParseReport;
typedef struct{int kind,operand,next,optionalIndex;} NDecoded;typedef struct{int tag;NDecoded value;} NDecodedResult;typedef struct{int tag,value;} NToken;typedef NDecodedResult(*Decode)(NInput,int,int*);typedef NToken(*Materialize)(NDecoded);typedef int(*Evaluate)(NToken,NToken,int);
typedef struct{int state,total,sequence,denials,error,original;} NControllerReport;
typedef NParseReport(*ParseRun)(NInput,int,int*);typedef NControllerReport(*ControllerRun)(int,int,int*);
typedef int(*AssertionRun)(_Bool,int);typedef _Bool(*Gate)(_Bool,_Bool,int*);typedef int(*Handled)(_Bool,_Bool,int*);typedef _Bool(*Nested)(_Bool,_Bool,_Bool,int*);typedef int(*LogicalLoop)(int,int*);typedef int(*ReferenceSelection)(_Bool);typedef int(*SequenceBeforeBranch)(_Bool,int*);typedef int(*UnusedScalar)(NToken,int);
_Static_assert(sizeof(NInput)==36,"checked input geometry");_Static_assert(sizeof(NParseReport)==20,"checked report geometry");_Static_assert(sizeof(NControllerReport)==24,"checked controller geometry");
#define RESOLVE(t,name,m,o) t name;{void*entry=memories[m]+images[m].symbols.data[o].offset;memcpy(&name,&entry,sizeof name);}
static int trap_trace,trap_number;static uintptr_t trap_start,trap_end;
static LONG CALLBACK observe_trap(PEXCEPTION_POINTERS e){uintptr_t p=(uintptr_t)e->ExceptionRecord->ExceptionAddress;
 if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&p>=trap_start&&p+2<=trap_end&&((unsigned char*)p)[0]==0x0f&&((unsigned char*)p)[1]==0x0b){char path[MAX_PATH];snprintf(path,sizeof path,"%s.%d",MARKER,trap_number);HANDLE f=CreateFileA(path,GENERIC_WRITE,0,NULL,CREATE_ALWAYS,FILE_ATTRIBUTE_NORMAL,NULL);if(f!=INVALID_HANDLE_VALUE){DWORD written;int evidence[2]={trap_trace,1};WriteFile(f,evidence,sizeof evidence,&written,NULL);CloseHandle(f);TerminateProcess(GetCurrentProcess(),EXCEPTION_ILLEGAL_INSTRUCTION);}}
 return EXCEPTION_CONTINUE_SEARCH;
}
static int prepare(int module,int trials){
 artifact=artifacts[module];
 FILE*f=fopen(artifact_paths[module],"rb");CHECK(f);size_t n=fread(artifacts[module],1,sizeof artifacts[module],f);fclose(f);concept_readonly_span_byte src={artifacts[module],n};
 concept_result_void_fallible_error admitted=F(validate_fallible_machine_bridge)(src);if(admitted.tag){fprintf(stderr,"module %d: %s\n",module,F(describe_fallible_error)(admitted.payload.error.error));return 1;}
 memset(wire,0xa5,sizeof wire);CHECK(F(validated_fallible_bridge_round_trip)(src,(concept_span_byte){wire,1}).tag);for(size_t j=0;j<sizeof wire;j++)CHECK(wire[j]==0xa5);
 concept_result_int_fallible_error rt=F(validated_fallible_bridge_round_trip)(src,(concept_span_byte){wire,sizeof wire});CHECK(!rt.tag&&rt.payload.ok.value==(int)n&&!memcmp(wire,artifacts[module],n));
 memset(code[module],0xa5,sizeof code[module]);memset(&images[module],0x5a,sizeof images[module]);before=images[module];CHECK(F(emit_module)(src,(concept_span_byte){code[module],1},&images[module]).tag&&!memcmp(&before,&images[module],sizeof before));for(size_t j=0;j<sizeof code[module];j++)CHECK(code[module][j]==0xa5);
 clock_t start=clock();concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code[module],sizeof code[module]},&images[module]);if(emitted.tag){fprintf(stderr,"module %d: %s\n",module,F(describe_native_error)(emitted.payload.error.error));concept_bridge_reader diagnosticReader={src,0};concept_result_wire_object_module_backend_error diagnostic=F(native_object_document)(src);CHECK(!diagnostic.tag);for(int fi=0;fi<diagnostic.payload.ok.value.Functions.count;fi++){concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&diagnosticReader,diagnostic.payload.ok.value,fi);CHECK(!owner.tag);function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();concept_result_void_aggregate_abierror prepared=F(prepare_aggregate_function)(&diagnosticReader,diagnostic.payload.ok.value,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan);if(prepared.tag){fprintf(stderr,"function %d: %s regs=%d instructions=%d blocks=%d slots=%d\n",fi,F(describe_aggregate_abierror)(prepared.payload.error.error),function.virtualRegCount,function.instructionCount,function.blockCount,function.slotCount);}}return 2;}CHECK(images[module].finalized);
 printf("module=%d functions=%d calls=%d bytes=%d emission-ms=%.3f\n",module,images[module].symbolCount,images[module].fixupCount,images[module].byteCount,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);CHECK(!decoded.tag);concept_wire_object_module doc=decoded.payload.ok.value;
 int min_frame=4096,max_frame=0,spills=0;unsigned long long hashes[FUNCTION_COUNT];start=clock();
 for(int run=0;run<trials;run++){
  FILE*fresh=fopen(fresh_paths[run][module],"rb");CHECK(fresh);size_t freshSize=fread(wire,1,sizeof wire,fresh);fclose(fresh);CHECK(freshSize==n&&!memcmp(wire,artifacts[module],n));concept_readonly_span_byte trialSource={wire,freshSize};CHECK(!F(validate_fallible_machine_bridge)(trialSource).tag);decoded=F(native_object_document)(trialSource);CHECK(!decoded.tag);doc=decoded.payload.ok.value;reader.data=trialSource;
  for(int fi=0;fi<doc.Functions.count;fi++){
  concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc,fi);CHECK(!owner.tag);
  function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();
  CHECK(!F(prepare_aggregate_function)(&reader,doc,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);
  for(int i=0;i<plan.parameterCount;i++)for(int j=i+1;j<plan.parameterCount;j++){concept_aggregate_argument_plan a=plan.parameters.data[i],b=plan.parameters.data[j];if((a.type.category.tag==3||a.type.category.tag==4)&&(b.type.category.tag==3||b.type.category.tag==4)){concept_scalar_location x=allocation.scalarLocations.data[a.carrier],y=allocation.scalarLocations.data[b.carrier];CHECK(x.kind.tag==0&&y.kind.tag==0&&x.gpr.tag!=y.gpr.tag);CHECK(allocation.start.data[a.carrier]<=allocation.end.data[b.carrier]&&allocation.start.data[b.carrier]<=allocation.end.data[a.carrier]);}}
  unsigned long long hash=fingerprint();CHECK(!debugOverflow);if(!run){hashes[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;for(int v=0;v<function.virtualRegCount;v++)spills+=allocation.spilled.data[v];}else CHECK(hashes[fi]==hash&&!strcmp(firstText[fi],debugText));
 }
  CHECK(!F(emit_module)(trialSource,(concept_span_byte){again,sizeof again},&other).tag);CHECK(!memcmp(&images[module],&other,sizeof other)&&!memcmp(code[module],again,images[module].byteCount));
 }
 if(trials)printf("module=%d fresh-pipelines=%d planning-and-emission-ms=%.3f frames=%d..%d spills=%d\n",module,trials,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC,min_frame,max_frame,spills);
 if(trials){f=fopen(dump_paths[module],"wb");CHECK(f&&fwrite(code[module],1,images[module].byteCount,f)==(size_t)images[module].byteCount);fclose(f);}
 memories[module]=VirtualAlloc(NULL,images[module].byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);CHECK(memories[module]);memcpy(memories[module],code[module],images[module].byteCount);DWORD old;CHECK(VirtualProtect(memories[module],images[module].byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memories[module],images[module].byteCount);
 return 0;
}
static const struct{NInput input;int total,commands,position,status,actual,reads;}vectors[]={
 {{{0,99,99,99,99,99,99,99},1},2,0,1,0,0,1},
 {{{1,7,0,99,99,99,99,99},3},9,1,3,0,0,3},
 {{{2,3,1,4,0,99,99,99},5},10,2,5,0,0,5},
 {{{3,4,2,0,99,99,99,99},4},8,1,4,0,0,4},
 {{{3,4,-1,0,99,99,99,99},4},6,1,4,0,0,4},
 {{{3,4,0,0,99,99,99,99},4},6,1,4,0,0,4},
 {{{0,99,99,99,99,99,99,99},0},0,0,0,-2,0,0},
 {{{1,99,99,99,99,99,99,99},1},0,0,1,-2,0,1},
 {{{9,99,99,99,99,99,99,99},1},0,0,0,-3,9,1},
 {{{2,0,99,99,99,99,99,99},2},0,0,1,-4,0,2},
 {{{1,17,99,99,99,99,99,99},2},0,0,1,-4,17,2},
 {{{3,1,-2,99,99,99,99,99},3},0,0,2,-4,-2,3},
 {{{1,1,1,1,1,1,1,1},8},0,0,8,-2,0,8},
 {{{0,0,0,0,0,0,0,0},9},0,0,0,-1,9,0},
 {{{0,0,0,0,0,0,0,0},-1},0,0,0,-1,-1,0},
 {{{1,-3,0,9,9,9,9,9},8},-1,1,3,0,0,3}
};
int main(int argc,char**argv){setvbuf(stdout,NULL,_IONBF,0);
 if(argc>1){CHECK(!prepare(1,0));trap_number=atoi(argv[1]);trap_start=(uintptr_t)memories[1];trap_end=trap_start+images[1].byteCount;SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);CHECK(AddVectoredExceptionHandler(1,observe_trap));RESOLVE(Gate,and_gate,1,ORD_Controller_GateAnd);RESOLVE(Gate,or_gate,1,ORD_Controller_GateOr);if(trap_number==0)(void)and_gate(1,0,&trap_trace);else (void)or_gate(0,0,&trap_trace);return 99;}
 for(int module=0;module<4;module++){
#ifdef INTEGRATION_NATIVE_ONLY
 CHECK(!prepare(module,TRIALS));
#else
 CHECK(!prepare(module,1));
#endif
 }
 CHECK(images[2].byteCount==images[3].byteCount&&!memcmp(code[2],code[3],images[2].byteCount));
 RESOLVE(Decode,decode_word,0,ORD_Parser_DecodeWord);RESOLVE(Materialize,materialize,0,ORD_Parser_Materialize);RESOLVE(Evaluate,evaluate,0,ORD_Parser_Evaluate);
 RESOLVE(ParseRun,parse,0,ORD_Parser_Run);RESOLVE(ControllerRun,control,1,ORD_Controller_Run);RESOLVE(AssertionRun,asserted,2,ORD_Assertions_Run);RESOLVE(AssertionRun,plain,3,ORD_Plain_Run);
 RESOLVE(Gate,and_gate,1,ORD_Controller_GateAnd);RESOLVE(Gate,or_gate,1,ORD_Controller_GateOr);RESOLVE(Handled,handled,1,ORD_Controller_GateHandled);RESOLVE(Nested,nested,1,ORD_Controller_NestedGate);RESOLVE(Handled,guarded,1,ORD_Controller_Guarded);RESOLVE(LogicalLoop,loop,1,ORD_Controller_LogicalLoop);
 RESOLVE(UnusedScalar,unused_scalar,1,ORD_Controller_UnusedScalarAfterEvent);
 RESOLVE(ReferenceSelection,reference_selection,1,ORD_Controller_ReferenceSelection);RESOLVE(SequenceBeforeBranch,sequence_before_branch,1,ORD_Controller_SequenceBeforeBranch);
 {NToken event={1,7};CHECK(unused_scalar(event,123)==7);event.tag=0;event.value=19;CHECK(unused_scalar(event,456)==19);event.tag=2;CHECK(unused_scalar(event,999)==0);
#ifndef INTEGRATION_NATIVE_ONLY
  int ignoredTrace=0;for(int variant=0;variant<4;variant++){event.tag=variant;event.value=23;CHECK(unused_scalar(event,987)==oracle_logic(6,variant,23,987,&ignoredTrace));}
#endif
 }
 // Independent C callers exercise two simultaneous eight-byte sum carriers.
 {int reads=0;NDecodedResult word=decode_word(vectors[1].input,0,&reads);CHECK(!word.tag&&word.value.kind==1&&word.value.operand==7&&reads==2);NToken token=materialize(word.value),none={1,0},some={0,3};CHECK(token.tag==1&&token.value==7&&evaluate(token,none,2)==9);token.tag=3;CHECK(evaluate(token,some,2)==12);}
 for(int repeat=0;repeat<40;repeat++){
  for(int selected=0;selected<2;selected++){int trace=0;CHECK(reference_selection(selected)==(selected?4:9));CHECK(sequence_before_branch(selected,&trace)==(selected?13:14)&&trace==(selected?12:13));
#ifndef INTEGRATION_NATIVE_ONLY
   int sourceTrace=0;trace=0;CHECK(reference_selection(selected)==oracle_logic(0,selected,0,0,&sourceTrace));CHECK(sequence_before_branch(selected,&trace)==oracle_logic(1,selected,0,0,&sourceTrace)&&trace==sourceTrace);
#endif
  }
  for(size_t vi=0;vi<sizeof vectors/sizeof vectors[0];vi++){
   size_t index=(vi+(size_t)repeat)%(sizeof vectors/sizeof vectors[0]);int reads=0;NParseReport got=parse(vectors[index].input,2,&reads);
   if(got.status!=vectors[index].status||got.total!=vectors[index].total||got.commands!=vectors[index].commands||got.position!=vectors[index].position||got.actual!=vectors[index].actual||reads!=vectors[index].reads){fprintf(stderr,"protocol vector=%zu repeat=%d actual={%d,%d,%d,%d,%d} reads=%d expected={%d,%d,%d,%d,%d} reads=%d\n",index,repeat,got.status,got.total,got.commands,got.position,got.actual,reads,vectors[index].status,vectors[index].total,vectors[index].commands,vectors[index].position,vectors[index].actual,vectors[index].reads);return __LINE__;}
#ifndef INTEGRATION_NATIVE_ONLY
   int expected[5];int sourceReads=0;oracle_parser(vectors[index].input.words,vectors[index].input.length,2,expected,&sourceReads);
   CHECK(got.status==expected[0]&&got.total==expected[1]&&got.commands==expected[2]&&got.position==expected[3]&&got.actual==expected[4]&&reads==sourceReads);
#endif
  }
  for(int scenario=0;scenario<5;scenario++)for(int value=-1;value<=6;value++){
   int writes=0;NControllerReport got=control(scenario,value,&writes);int error=scenario==1?1:scenario==2?2:0;int amount=value+(scenario==4?2:0);int denial=error?0:amount<0?10:amount>(scenario==3?3:16)?1:0;
   CHECK(got.state==(error?0:denial?3:2)&&got.total==(error||denial?0:amount)&&got.sequence==(error?0:denial?2:3)&&got.denials==denial&&got.error==error&&got.original==0&&writes==!error);
#ifndef INTEGRATION_NATIVE_ONLY
   int sourceWrites=0,expected[6];oracle_controller(scenario,value,expected,&sourceWrites);
   CHECK(got.state==expected[0]&&got.total==expected[1]&&got.sequence==expected[2]&&got.denials==expected[3]&&got.error==expected[4]&&got.original==expected[5]&&writes==sourceWrites);
#endif
  }
  for(int valid=0;valid<2;valid++)for(int value=-2;value<10;value++){int got=asserted(valid,value);CHECK(got==(valid?(value>0?value+1:0):-1)&&plain(valid,value)==got);
#ifndef INTEGRATION_NATIVE_ONLY
   CHECK(got==oracle_assertion(valid,value));
#endif
  }
  for(int a=0;a<2;a++)for(int b=0;b<2;b++){
   int trace=0;CHECK(and_gate(0,b,&trace)==0&&trace==1);trace=0;CHECK(or_gate(1,b,&trace)==1&&trace==1);trace=0;CHECK(and_gate(a,1,&trace)==a&&trace==(a?12:1));trace=0;CHECK(or_gate(a,1,&trace)==1&&trace==(a?1:12));
   trace=0;CHECK(handled(a,b,&trace)==(a?(b?1:-1):0)&&trace==(a?12:1));trace=0;CHECK(guarded(a,b,&trace)==(a&&b?7:9)&&trace==(a?12:1));
   for(int c=0;c<2;c++){trace=0;CHECK(nested(a,b,c,&trace)==(a&&(b||c))&&trace==(a?(b?12:123):1));
#ifndef INTEGRATION_NATIVE_ONLY
    int sourceTrace=0;trace=0;CHECK(nested(a,b,c,&trace)==oracle_logic(2,a,b,c,&sourceTrace)&&trace==sourceTrace);
#endif
   }
#ifndef INTEGRATION_NATIVE_ONLY
   int sourceTrace=0;trace=0;CHECK(handled(a,b,&trace)==oracle_logic(3,a,b,0,&sourceTrace)&&trace==sourceTrace);sourceTrace=0;trace=0;CHECK(guarded(a,b,&trace)==oracle_logic(4,a,b,0,&sourceTrace)&&trace==sourceTrace);
#endif
  }
  for(int count=0;count<6;count++){int trace=0;CHECK(loop(count,&trace)==(count<3?count:3));
#ifndef INTEGRATION_NATIVE_ONLY
   int sourceTrace=0;trace=0;CHECK(loop(count,&trace)==oracle_logic(5,count,0,0,&sourceTrace)&&trace==sourceTrace);
#endif
  }
 }
 for(int trap=0;trap<2;trap++){char exe[MAX_PATH],command[MAX_PATH+32],marker[MAX_PATH];snprintf(marker,sizeof marker,"%s.%d",MARKER,trap);DeleteFileA(marker);CHECK(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %d",exe,trap);STARTUPINFOA startup={0};PROCESS_INFORMATION process;startup.cb=sizeof startup;CHECK(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));CHECK(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;CHECK(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);CHECK(status==EXCEPTION_ILLEGAL_INSTRUCTION);int evidence[2]={0};FILE*f=fopen(marker,"rb");CHECK(f&&fread(evidence,sizeof evidence,1,f)==1);fclose(f);CHECK(evidence[0]==12&&evidence[1]==1);}
 for(int module=0;module<4;module++)CHECK(VirtualFree(memories[module],0,MEM_RELEASE));
#ifdef INTEGRATION_NATIVE_ONLY
 puts("artifact-only integration passed: 16 protocol vectors, 40 interleavings, controller, proof erasure, logical CFG and exact traps; no domain C linked");
#else
 puts("strict C11/native integration parity passed: values, diagnostics, side effects, copy isolation and assertion erasure");
#endif
 return 0;
}`

// Keep each independent generated module in its own C translation unit.
// Shared built-in C type names need not be included together in the caller.
// Adapters transfer every observable field through plain C scalar boundaries.
var integrationOracleWrappers = map[string]string{
	"Parser": `
void oracle_parser(const int*words,int length,int seed,int*out,int*reads){
 concept_protocol_input input={0};input.length=length;for(int i=0;i<8;i++)input.words.data[i]=words[i];
 concept_parse_report r=concept_testing__integration_parser_run(input,seed,reads);
 out[0]=r.status;out[1]=r.total;out[2]=r.commands;out[3]=r.position;out[4]=r.actual;
}
`,
	"Controller": `
void oracle_controller(int scenario,int value,int*out,int*writes){
 concept_controller_report r=concept_testing__integration_controller_run(scenario,value,writes);
 out[0]=r.state;out[1]=r.total;out[2]=r.sequence;out[3]=r.denials;out[4]=r.error;out[5]=r.original;
}
int oracle_logic(int op,int a,int b,int c,int*trace){switch(op){
 case 0:return concept_testing__integration_controller_reference_selection(a);
 case 1:return concept_testing__integration_controller_sequence_before_branch(a,trace);
 case 2:return concept_testing__integration_controller_nested_gate(a,b,c,trace);
 case 3:return concept_testing__integration_controller_gate_handled(a,b,trace);
 case 4:return concept_testing__integration_controller_guarded(a,b,trace);
 case 5:return concept_testing__integration_controller_logical_loop(a,trace);
 case 6:{concept_controller_event event={0};event.tag=(concept_controller_event_tag)a;if(a==0)event.payload.start.limit=b;if(a==1)event.payload.add.amount=b;return concept_testing__integration_controller_unused_scalar_after_event(event,c);}
 default:abort();
}}
`,
	"Assertions": `
int oracle_assertion(_Bool valid,int value){return concept_testing__integration_assertions_run(valid,value);}
`,
}
