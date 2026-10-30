package concept

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func patternFixture(t *testing.T, ordinary bool) (Module, []byte, LIRModule, MachineModule, []byte) {
	t.Helper()
	m, source, lir := matchFixture(t, ordinary)
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	return m, source, lir, machine, data
}
func TestEVT2h3Transport(t *testing.T) {
	start := time.Now()
	for _, ordinary := range []bool{true, false} {
		_, _, lir, m, data := patternFixture(t, ordinary)
		want := map[bool]string{true: "b3f04bb7352fec2209349bb26c55cd837158e50e32c2c656651d8bb41248c0ed", false: "e68f23900d6a28cc14c81209b97bc561ddfef0b0c2dd53dd150ca0fcb6d4e560"}[ordinary]
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
			t.Fatalf("frozen v9 artifact changed: %s", got)
		}
		for run := 0; run < 100; run++ {
			_, _, other, next, raw := patternFixture(t, ordinary)
			if !bytes.Equal(raw, data) || other.String() != lir.String() || next.String() != m.String() {
				t.Fatalf("fresh generation %d differs", run)
			}
			decoded, err := DecodeMachineBridge(raw)
			if err != nil || decoded.String() != m.String() {
				t.Fatalf("roundtrip %d: %v", run, err)
			}
		}
		t.Logf("ordinary=%t functions=%d artifact=%d sha256=%x", ordinary, len(m.Pattern.Base.Base.Objects.Functions), len(data), sha256.Sum256(data))
	}
	path := "../../libraries/Standard/Backend/BridgeSchema9.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := GenerateMachineBridgeCodecs(schema)
	if err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string][]byte{"../../internal/concept/machineir_bridge9_codec_generated.go": first.Go, "../../libraries/Standard/Backend/BridgeCodec9.concept": first.Concept, "../../docs/design/EVT2-MACHINEIR-BRIDGE9.schema.json": first.Metadata} {
		actual, err := os.ReadFile(file)
		if err != nil || !bytes.Equal(actual, want) {
			t.Fatalf("stale generated %s: %v", file, err)
		}
	}
	for i := 0; i < 100; i++ {
		next, err := GenerateMachineBridgeCodecs(schema)
		if err != nil || !bytes.Equal(next.Go, first.Go) || !bytes.Equal(next.Concept, first.Concept) || !bytes.Equal(next.Metadata, first.Metadata) {
			t.Fatalf("codec generation %d: %v", i, err)
		}
	}
	t.Logf("100 fresh source/LIR/CFG/MachineIR/debug + codecs: %s; schema=%s", time.Since(start), first.Hash)
}

var patternAdmissionBuild [2]struct {
	sync.Once
	outputs Outputs
	err     error
}

func patternAdmissionOutputs(t *testing.T, verify bool) Outputs {
	t.Helper()
	requireNativeQualification(t)
	index := 0
	if verify {
		index = 1
	}
	build := &patternAdmissionBuild[index]
	build.Do(func() {
		path := "../../libraries/Standard/Backend/BridgeValidate9.concept"
		source, err := os.ReadFile(path)
		if err != nil {
			build.err = err
			return
		}
		module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
		if err != nil {
			build.err = err
			return
		}
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		build.outputs, build.err = GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
	})
	if build.err != nil {
		t.Fatal(build.err)
	}
	return build.outputs
}
func TestEVT2h3ArtifactAdmission(t *testing.T) {
	requireNativeQualification(t)
	dir := t.TempDir()
	var cases strings.Builder
	for _, ordinary := range []bool{true, false} {
		_, _, _, _, data := patternFixture(t, ordinary)
		path := filepath.Join(dir, fmt.Sprintf("match-%t.cmir9", ordinary))
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&cases, `{FILE*f=fopen(%q,"rb");if(!f)return 1;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte src={input,n};for(int run=0;run<100;run++){concept_result_int_pattern_error r=F(validated_pattern_bridge_round_trip)(src,dst);if(r.tag){fprintf(stderr,"ordinary=%t admission %%s\n",F(describe_pattern_error)(r.payload.error.error));return 2;}if((size_t)r.payload.ok.value!=n||memcmp(input,output,n))return 3;}}`, filepath.ToSlash(path), ordinary)
	}
	host := `#include "bridgevalidate9.generated.h"
#include <stdio.h>
#include <string.h>
#define F(n) concept_standard__backend__bridge_validate9_##n
static unsigned char input[2000000],output[2000000];
int main(void){concept_span_byte dst={output,sizeof output};/* CASES */return 0;}`
	host = strings.Replace(host, "/* CASES */", cases.String(), 1)
	host = strings.ReplaceAll(host, "return 2;", `concept_bridge_reader reader={src,0};concept_result_wire_pattern_module_backend_error decoded=F(read_pattern_document)(&reader);if(!decoded.tag){concept_enum_domain_context aliases=F(empty_enum_domain_context)();concept_result_void_enum_error e=F(validate_enum_domain_records)(&reader,decoded.payload.ok.value.Base.Base.Objects,decoded.payload.ok.value.Enums,&aliases);fprintf(stderr,"enum domains: %s\n",e.tag?F(describe_enum_error)(e.payload.error.error):"ok");concept_object_authority_context types=F(empty_object_authority)();concept_result_void_tagged_error s=F(validate_tagged_domains_with_aliases)(&reader,decoded.payload.ok.value.Base,&types,aliases.aliases);fprintf(stderr,"tagged domains: %s\n",s.tag?F(describe_tagged_error)(s.payload.error.error):"ok");concept_result_void_object_error o=F(validate_object_type_extensions)(&reader,decoded.payload.ok.value.Base.Base.Objects,decoded.payload.ok.value.Base.Domains.count>0,aliases.aliases,types.taggedTypes);fprintf(stderr,"object types: %s; sums=%d aliases=%d,%d\n",o.tag?F(describe_object_error)(o.payload.error.error):"ok",decoded.payload.ok.value.Base.Domains.count,aliases.aliases.data[1],aliases.aliases.data[2]);}return 2;`)
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, patternAdmissionOutputs(t, verify), "match_admission.c", host, "-pedantic-errors", "-O2")
	}
}

func TestEVT2h3NativeExecution(t *testing.T) {
	requireNativeQualification(t)
	// The 35-function CFG specimen performs 100 fresh admissions, complete
	// physical plans and emissions. Keep that gate bounded independently of
	// the ordinary two-minute single-specimen deadline.
	t.Setenv("CONCEPT_TEST_NATIVE_TIMEOUT", "8m")
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64")
	}
	first := map[bool][]byte{}
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			for _, ordinary := range []bool{true, false} {
				module, source, _, machine, data := patternFixture(t, ordinary)
				dir := t.TempDir()
				artifact := filepath.Join(dir, "match.cmir9")
				dump := filepath.Join(dir, "native.bin")
				if err := os.WriteFile(artifact, data, 0600); err != nil {
					t.Fatal(err)
				}
				var defs strings.Builder
				fmt.Fprintf(&defs, "#define ARTIFACT %q\n#define DUMP %q\n#define FUNCTION_COUNT %d\n#define MATCH_ORDINARY %d\n", filepath.ToSlash(artifact), filepath.ToSlash(dump), len(machine.Pattern.Base.Base.Objects.Functions), map[bool]int{true: 1, false: 0}[ordinary])
				last := map[string]int{}
				for i, f := range machine.Pattern.Base.Base.Objects.Functions {
					last[f.Name] = i
				}
				for i, f := range machine.Pattern.Base.Base.Objects.Functions {
					if last[f.Name] == i {
						fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
					}
				}
				host := strings.Replace(patternNativeHost, "/* DEFINITIONS */", defs.String(), 1)
				a := strings.Index(aggregateABIPlanningHost, "static unsigned long long digest;")
				b := strings.Index(aggregateABIPlanningHost, "static int run_plan(")
				host = strings.Replace(host, "/* PLAN DEBUG */", aggregateABIPlanningHost[a:b], 1)
				outputs := backendTestOutputs(t, verify)
				t.Log(runFoundationNativeHarnessOutput(t, outputs, "pattern_native.c", "#define MATCH_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
				nativeCode, err := os.ReadFile(dump)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("artifact-only ordinary=%t native=%d sha256=%x", ordinary, len(nativeCode), sha256.Sum256(nativeCode))
				want := map[bool]string{true: "48dee1c8a55f462854fe930e64fb34d3eefac77d0c18a2d6c6cdfdc4ca40e83b", false: "398445c4b5e1bf1ff09c190f897843fb58985fe96625295e71f8ba6f2628484f"}[ordinary]
				if got := fmt.Sprintf("%x", sha256.Sum256(nativeCode)); got != want {
					t.Fatalf("frozen native match bytes changed: %s", got)
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
				t.Log(runFoundationNativeHarnessOutput(t, outputs, "pattern_oracle.c", host, "-pedantic-errors", "-O2"))
				code, err := os.ReadFile(dump)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(code, nativeCode) {
					t.Fatal("source oracle changed native bytes")
				}
				if prior := first[ordinary]; prior == nil {
					first[ordinary] = bytes.Clone(code)
				} else if !bytes.Equal(prior, code) {
					t.Fatal("Normal/Verify match bytes differ")
				}
				t.Logf("ordinary=%t native=%d sha256=%x", ordinary, len(code), sha256.Sum256(code))
			}
		})
	}
}

const patternNativeHost = `#include "amd64.generated.h"
#ifndef MATCH_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
#ifdef MATCH_NATIVE_ONLY
#define DETERMINISM_RUNS 100
#else
#define DETERMINISM_RUNS 1
#endif
/* DEFINITIONS */
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
static unsigned char artifact[2000000],code[262144],again[262144];
static concept_native_image image,other;
static concept_backend_function function;
static concept_object_placement placement;
static concept_allocation allocation;
static concept_finalized_frame frame;
static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
static unsigned char*native_begin;static int native_size;
static LONG CALLBACK expected_trap(PEXCEPTION_POINTERS e){if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&(uintptr_t)e->ExceptionRecord->ExceptionAddress>=(uintptr_t)native_begin&&(uintptr_t)e->ExceptionRecord->ExceptionAddress<(uintptr_t)native_begin+native_size)ExitProcess(0);return EXCEPTION_CONTINUE_SEARCH;}
typedef struct{int count;double amount;} Item;
typedef struct{int count;_Bool ready;Item item;int numbers[3];} Many;
typedef union{int integer;double decimal;float real;_Bool flag;Many many;int stamp;} Payload;
typedef struct{uint32_t tag;Payload payload;} Token;
typedef uint32_t(*EnumAt)(int);
typedef int(*EnumCode)(uint32_t);
typedef int(*IntInt)(int);
typedef Token(*Empty)(void);
typedef Token(*MakeInt)(int);
typedef Token(*MakeDouble)(double);
typedef Token(*MakeFloat)(float);
typedef Token(*MakeBool)(_Bool);
typedef Token(*Replace)(Token);
typedef int(*Inspect)(Token);
typedef double(*Double)(Token);
typedef float(*Float)(Token);
typedef Item(*Record)(Token);
typedef int(*Guards)(Token,int*);
typedef int(*Once)(int*,int);
typedef int(*Loop)(Token,int);
typedef double(*Spill)(double,double,double,double,double,double,double);
#define ACCESS(t) static t Resolve##t(unsigned char*memory,int ordinal){t f;void*p=memory+image.symbols.data[ordinal].offset;memcpy(&f,&p,sizeof f);return f;}
ACCESS(EnumAt) ACCESS(EnumCode) ACCESS(IntInt) ACCESS(Empty) ACCESS(MakeInt) ACCESS(MakeDouble) ACCESS(MakeFloat) ACCESS(MakeBool) ACCESS(Replace) ACCESS(Inspect) ACCESS(Double) ACCESS(Float) ACCESS(Record) ACCESS(Guards) ACCESS(Once) ACCESS(Loop) ACCESS(Spill)
#define FN(n,t) Resolve##t(memory,ORD_##n)
static uint64_t bits64(double v){uint64_t b;memcpy(&b,&v,8);return b;}
static uint32_t bits32(float v){uint32_t b;memcpy(&b,&v,4);return b;}
int main(int argc,char**argv){
 setvbuf(stdout,NULL,_IONBF,0);(void)argv;
 FILE*f=fopen(ARTIFACT,"rb");REQUIRE(f);size_t size=fread(artifact,1,sizeof artifact,f);fclose(f);concept_readonly_span_byte src={artifact,size};
 concept_result_void_pattern_error admission=F(validate_pattern_machine_bridge)(src);if(admission.tag){fprintf(stderr,"admission %s\n",F(describe_pattern_error)(admission.payload.error.error));return 1;}
 memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);other=image;concept_result_int_native_error tiny=F(emit_module)(src,(concept_span_byte){code,1},&image);REQUIRE(tiny.tag&&!memcmp(&image,&other,sizeof image));for(size_t i=0;i<sizeof code;i++)REQUIRE(code[i]==0xa5);
 clock_t start=clock();concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(emitted.tag){fprintf(stderr,"emission %s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}REQUIRE(image.finalized);
 printf("match ordinary=%d functions=%d calls=%d bytes=%d emission-ms=%.3f\n",MATCH_ORDINARY,image.symbolCount,image.fixupCount,image.byteCount,1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);
 if(argc==1){
  concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);REQUIRE(!decoded.tag);concept_wire_object_module document=decoded.payload.ok.value;
  int min_frame=4096,max_frame=0,spills=0;clock_t planning=clock();unsigned long long first[FUNCTION_COUNT];
  for(int run=0;run<DETERMINISM_RUNS;run++)for(int fi=0;fi<document.Functions.count;fi++){
   concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,document,fi);REQUIRE(!owner.tag);
   function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();concept_result_void_aggregate_abierror prepared=F(prepare_aggregate_function)(&reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan);REQUIRE(!prepared.tag);unsigned long long hash=fingerprint();REQUIRE(!debugOverflow);
   if(!run){first[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;for(int id=0;id<function.virtualRegCount;id++)spills+=allocation.spilled.data[id];}else{REQUIRE(first[fi]==hash&&!strcmp(firstText[fi],debugText));}
  }
  printf("%d physical plans ms=%.3f frame=%d..%d spills=%d\n",DETERMINISM_RUNS,1000.0*(double)(clock()-planning)/CLOCKS_PER_SEC,min_frame,max_frame,spills);
  for(int run=0;run<DETERMINISM_RUNS;run++){concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){again,sizeof again},&other);REQUIRE(!r.tag&&!memcmp(&image,&other,sizeof image)&&!memcmp(code,again,image.byteCount));}
 }
 f=fopen(DUMP,"wb");REQUIRE(f);REQUIRE(fwrite(code,1,image.byteCount,f)==(size_t)image.byteCount);fclose(f);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,code,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);native_begin=memory;native_size=image.byteCount;REQUIRE(AddVectoredExceptionHandler(1,expected_trap));
#if MATCH_ORDINARY
  (void)FN(StateCode,EnumCode)(99);
#else
  Token bad;memset(&bad,0,sizeof bad);bad.tag=99;(void)FN(Inspect,Inspect)(bad);
#endif
  return 99;
 }
 char exe[MAX_PATH],command[MAX_PATH+32];REQUIRE(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" trap",exe);STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;REQUIRE(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));REQUIRE(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;REQUIRE(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);REQUIRE(status==0);
 for(int i=0;i<100;i++){
#if MATCH_ORDINARY
  uint32_t state=FN(StateAt,EnumAt)(i%3);REQUIRE(state==(uint32_t)(i%3));REQUIRE(FN(StateCode,EnumCode)(state)==11*(i%3+1));REQUIRE(FN(AllReturn,EnumCode)(state)==10*(i%3+1));
  uint32_t color=FN(ColorAt,EnumAt)(i%3);REQUIRE(FN(ColorCode,EnumCode)(color)==i%3+1);
  REQUIRE(FN(StateLoop,IntInt)(i)==(i==0?0:i==1?1:3*i-3));REQUIRE(FN(Wildcard,IntInt)(i-1)==(i==0?7:i==1?9:13));
#ifndef MATCH_NATIVE_ONLY
  REQUIRE(FN(StateCode,EnumCode)(state)==concept_testing__ordinary_matches_state_code(concept_testing__ordinary_matches_state_at(i%3)));REQUIRE(FN(StateLoop,IntInt)(i)==concept_testing__ordinary_matches_state_loop(i));
#endif
#else
  Token a=FN(MakeInteger,MakeInt)(i-50);REQUIRE(FN(Inspect,Inspect)(a)==i-50);Token b=FN(MakeDecimal,MakeDouble)(3.5);REQUIRE(FN(Inspect,Inspect)(b)==35&&FN(DoubleResult,Double)(b)==3.5);
  Token c=FN(MakeReal,MakeFloat)(1.25f);REQUIRE(FN(Inspect,Inspect)(c)==125&&FN(FloatResult,Float)(c)==1.25f);Token flag=FN(MakeFlag,MakeBool)(i%2);REQUIRE(FN(Inspect,Inspect)(flag)==(i%2?1:-1));
  Token many=FN(MakeMany,MakeInt)(i);REQUIRE(FN(Inspect,Inspect)(many)==2*i+7);Item item=FN(RecordResult,Record)(many);REQUIRE(item.count==i+1&&item.amount==3.5);
  Token stamp=FN(MakeStamped,MakeInt)(i);REQUIRE(FN(Inspect,Inspect)(stamp)==i);Token changed=FN(Replace,Replace)(stamp);REQUIRE(FN(Inspect,Inspect)(changed)==i+1&&stamp.payload.stamp==i);
  Token empty=FN(MakeEnd,Empty)();REQUIRE(FN(Inspect,Inspect)(empty)==0&&FN(Nested,Loop)(empty,i)==i);
  int log=0;REQUIRE(FN(Guards,Guards)(FN(MakeInteger,MakeInt)(7),&log)==7&&log==123);log=0;REQUIRE(FN(Guards,Guards)(FN(MakeInteger,MakeInt)(-7),&log)==-7&&log==1);log=0;REQUIRE(FN(Guards,Guards)(FN(MakeInteger,MakeInt)(0),&log)==20&&log==12);log=0;REQUIRE(FN(Guards,Guards)(empty,&log)==0&&log==9);log=0;REQUIRE(FN(Guards,Guards)(b,&log)==2&&log==0);
  int calls=0;REQUIRE(FN(Once,Once)(&calls,i)==i+1&&calls==1);REQUIRE(FN(Statements,Loop)(FN(MakeInteger,MakeInt)(4),3)==15);REQUIRE(FN(Statements,Loop)(FN(MakeInteger,MakeInt)(-4),3)==-4);REQUIRE(FN(Statements,Loop)(many,2)==18);REQUIRE(FN(Statements,Loop)(FN(MakeFlag,MakeBool)(0),4)==0);
  REQUIRE(FN(SpillMatch,Spill)(1,2,3,4,5,6,7+i)==52.5+i);
#ifndef MATCH_NATIVE_ONLY
  REQUIRE(FN(Inspect,Inspect)(many)==concept_testing__tagged_matches_inspect(concept_testing__tagged_matches_make_many(i)));int oracleLog=0;log=0;REQUIRE(FN(Guards,Guards)(a,&log)==concept_testing__tagged_matches_guards(concept_testing__tagged_matches_make_integer(i-50),&oracleLog)&&log==oracleLog);REQUIRE(FN(SpillMatch,Spill)(1,2,3,4,5,6,7+i)==concept_testing__tagged_matches_spill_match(1,2,3,4,5,6,7+i));
#endif
#endif
 }
#if !MATCH_ORDINARY
 const uint64_t doubles[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};for(int i=0;i<4;i++){double input;memcpy(&input,&doubles[i],8);REQUIRE(bits64(FN(DoubleResult,Double)(FN(MakeDecimal,MakeDouble)(input)))==doubles[i]);
#ifndef MATCH_NATIVE_ONLY
 REQUIRE(bits64(concept_testing__tagged_matches_double_result(concept_testing__tagged_matches_make_decimal(input)))==doubles[i]);
#endif
 }
 const uint32_t floats[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};for(int i=0;i<4;i++){float input;memcpy(&input,&floats[i],4);REQUIRE(bits32(FN(FloatResult,Float)(FN(MakeReal,MakeFloat)(input)))==floats[i]);
#ifndef MATCH_NATIVE_ONLY
 REQUIRE(bits32(concept_testing__tagged_matches_float_result(concept_testing__tagged_matches_make_real(input)))==floats[i]);
#endif
 }
#endif
 VirtualFree(memory,0,MEM_RELEASE);printf("native match execution, invalid-tag trap and %d plans/emissions passed\n",DETERMINISM_RUNS);return 0;
}
`
