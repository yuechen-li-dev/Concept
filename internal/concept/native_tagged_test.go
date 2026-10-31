package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEVT2h2NativeExecution(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64")
	}
	m, source, _, machine, data := nativeTaggedFixture(t)
	var defs strings.Builder
	defs.WriteString("static const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&defs, "%d,", b)
	}
	defs.WriteString("};\n")
	var dlls []string
	for _, compiler := range []string{"cl", "clang-cl"} {
		if dll := buildTaggedProbe(t, compiler); dll != "" {
			dlls = append(dlls, dll)
		}
	}
	fmt.Fprintf(&defs, "static const char* probe_dlls[]={")
	for _, dll := range dlls {
		fmt.Fprintf(&defs, "%q,", filepath.ToSlash(dll))
	}
	defs.WriteString("NULL};\n")
	fmt.Fprintf(&defs, "#define FUNCTION_COUNT %d\n", len(machine.Tagged.Base.Objects.Functions))
	for i, f := range machine.Tagged.Base.Objects.Functions {
		fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
	}
	dump := filepath.ToSlash(filepath.Join(t.TempDir(), "native.bin"))
	host := strings.Replace(taggedNativeHost, "/* ARTIFACT */", defs.String(), 1)
	host = strings.Replace(host, "/* DUMP */", fmt.Sprintf("%q", dump), 1)
	debugStart := strings.Index(aggregateABIPlanningHost, "static unsigned long long digest;")
	debugEnd := strings.Index(aggregateABIPlanningHost, "static int run_plan(")
	host = strings.Replace(host, "/* PLAN DEBUG */", aggregateABIPlanningHost[debugStart:debugEnd], 1)
	var first []byte
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "tagged_native.c", "#define TAGGED_NATIVE_ONLY 1\n"+host, "-pedantic-errors", "-O2"))
			policy := ConservativeCompilationPolicy()
			if verify {
				policy = VerifyCompilationPolicy()
			}
			oracle, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			for n, b := range oracle {
				if strings.HasSuffix(n, ".generated.c") {
					outputs["oracle.c"] = b
				} else {
					outputs[n] = b
				}
			}
			t.Log(runFoundationNativeHarnessOutput(t, outputs, "tagged_oracle.c", host, "-pedantic-errors", "-O2"))
			actual, err := os.ReadFile(filepath.FromSlash(dump))
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = actual
			} else if !bytes.Equal(first, actual) {
				t.Fatal("Normal/Verify native bytes differ")
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(actual)); got != "524c364540dbb5efc8b1634f5d7087f8933fd4225d92afaa73dc39bccd7becc3" {
				t.Fatalf("exact native tagged bytes changed: %s", got)
			}
			t.Logf("native bytes=%d sha256=%x", len(actual), sha256.Sum256(actual))
		})
	}
}

func buildTaggedProbe(t *testing.T, compiler string) string {
	return buildIndependentSumProbe(t, compiler, "../../language/evt2h2/probes/tagged.c")
}

func buildIndependentSumProbe(t *testing.T, compiler, source string) string {
	t.Helper()
	cl, err := exec.LookPath("cl")
	if err != nil {
		t.Log("independent MSVC probe unavailable")
		return ""
	}
	tool, err := exec.LookPath(compiler)
	if err != nil {
		t.Log(compiler + " probe unavailable")
		return ""
	}
	vc := filepath.Dir(cl)
	for i := 0; i < 6; i++ {
		vc = filepath.Dir(vc)
	}
	setup := filepath.Join(vc, "Auxiliary", "Build", "vcvars64.bat")
	dir := t.TempDir()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "tagged.c"), body, 0600); err != nil {
		t.Fatal(err)
	}
	build := fmt.Sprintf("@echo off\r\ncall \"%s\" >nul\r\nif errorlevel 1 exit /b 1\r\n\"%s\" /nologo /std:c11 /O2 /MD /LD tagged.c /link /OUT:tagged.dll /IMPLIB:tagged.lib\r\nexit /b %%errorlevel%%\r\n", setup, tool)
	if err = os.WriteFile(filepath.Join(dir, "build.cmd"), []byte(build), 0600); err != nil {
		t.Fatal(err)
	}
	command := nativeCommand(t, "cmd.exe", "/d", "/c", "build.cmd")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("independent %s probe: %v\n%s", compiler, err, output)
	}
	t.Logf("independent tagged ABI probe: %s", tool)
	return filepath.Join(dir, "tagged.dll")
}

const taggedNativeHost = `#include "amd64.generated.h"
#ifndef TAGGED_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#include <stdlib.h>
#define F(n) concept_standard__backend__amd64_##n
/* ARTIFACT */
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
typedef struct {int count; double amount;} ProbeItem;
typedef struct {int value;} ProbeStamp;
typedef struct {int count;_Bool ready;ProbeItem item;int numbers[3];} ProbeMany;
typedef union {int integer;double decimal;float real;_Bool flag;ProbeMany many;ProbeStamp stamp;} ProbePayload;
typedef struct {uint32_t tag;ProbePayload payload;} ProbeToken;
typedef struct {uint32_t tag;int value;} ProbeSmall;
typedef struct {uint32_t tag;int first,second;} ProbeOrdered;
typedef struct {int data[3];} ProbeArray;
typedef ProbeToken(*EndFn)(void);
typedef ProbeToken(*IntFn)(int);
typedef ProbeToken(*DoubleFn)(double);
typedef ProbeToken(*FloatFn)(float);
typedef ProbeToken(*BoolFn)(_Bool);
typedef ProbeToken(*ForwardFn)(ProbeToken);
typedef ProbeToken(*ReplaceFn)(ProbeToken,int);
typedef ProbeToken(*PressureFn)(ProbeToken,int,double,ProbeToken,float,_Bool,ProbeToken);
typedef ProbeToken(*FloatingFn)(double,double,double,double,double,double,double);
typedef ProbeSmall(*SmallFn)(int);
typedef ProbeOrdered(*OrderedFn)(int*);
typedef int(*GetIntFn)(ProbeToken);
typedef double(*GetDoubleFn)(ProbeToken);
typedef float(*GetFloatFn)(ProbeToken);
typedef _Bool(*GetBoolFn)(ProbeToken);
typedef ProbeItem(*GetItemFn)(ProbeToken);
typedef ProbeArray(*GetArrayFn)(ProbeToken);
/* Native records without foreign representation use an explicit hidden result. */
typedef ProbeStamp*(*GetStampFn)(ProbeStamp*,ProbeToken);
static concept_native_image image,other;
static unsigned char code[262144],again[262144];
static concept_backend_function function;
static concept_object_placement placement;
static concept_allocation allocation;
static concept_finalized_frame frame;
static concept_aggregate_function_plan plan;
/* PLAN DEBUG */
#define GETTER(t) static t Resolve##t(unsigned char *memory,int ordinal){t f=NULL;void*p=memory+image.symbols.data[ordinal].offset;memcpy(&f,&p,sizeof f);return f;}
GETTER(EndFn) GETTER(IntFn) GETTER(DoubleFn) GETTER(FloatFn) GETTER(BoolFn) GETTER(ForwardFn) GETTER(ReplaceFn) GETTER(PressureFn) GETTER(FloatingFn) GETTER(SmallFn) GETTER(OrderedFn) GETTER(GetIntFn) GETTER(GetDoubleFn) GETTER(GetFloatFn) GETTER(GetBoolFn) GETTER(GetItemFn) GETTER(GetArrayFn) GETTER(GetStampFn)
#define FN(n,t) Resolve##t(memory,ORD_##n)
static uint64_t bits64(double v){uint64_t b;memcpy(&b,&v,8);return b;}
static uint32_t bits32(float v){uint32_t b;memcpy(&b,&v,4);return b;}
static unsigned char *native_begin;static int native_size;
static LONG CALLBACK expected_trap(PEXCEPTION_POINTERS e){
 if(e->ExceptionRecord->ExceptionCode==EXCEPTION_ILLEGAL_INSTRUCTION&&
 (uintptr_t)e->ExceptionRecord->ExceptionAddress>=(uintptr_t)native_begin&&
 (uintptr_t)e->ExceptionRecord->ExceptionAddress<(uintptr_t)native_begin+native_size)ExitProcess(0);
 return EXCEPTION_CONTINUE_SEARCH;
}
static int callback_count;
static ProbeToken foreign_token(ProbeToken value){callback_count++;return value;}
static ProbeSmall foreign_small(ProbeSmall value){callback_count++;return value;}
int main(int argc,char**argv){
 REQUIRE(sizeof(ProbeToken)==48&&sizeof(ProbeSmall)==8&&sizeof(ProbeOrdered)==12);
 concept_readonly_span_byte src={artifact,sizeof artifact};
 concept_result_void_tagged_error admission=F(validate_tagged_machine_bridge)(src);
 if(admission.tag){fprintf(stderr,"admission %s\n",F(describe_tagged_error)(admission.payload.error.error));return 1;}
 clock_t start=clock();
 memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);other=image;
 concept_result_int_native_error tiny=F(emit_module)(src,(concept_span_byte){code,1},&image);REQUIRE(tiny.tag&&!memcmp(&image,&other,sizeof image));for(size_t i=0;i<sizeof code;i++)REQUIRE(code[i]==0xa5);
 start=clock();
 concept_result_int_native_error emitted=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);
 if(emitted.tag){fprintf(stderr,"emission %s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}
 REQUIRE(image.finalized);
 printf("tagged emission %.3f ms; %d functions; %d calls; %d bytes\n",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC,image.symbolCount,image.fixupCount,image.byteCount);
 if(argc==1){
   concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);REQUIRE(!decoded.tag);concept_wire_object_module document=decoded.payload.ok.value;
   int min_frame=4096,max_frame=0;clock_t planning=clock();unsigned long long first[FUNCTION_COUNT];
   for(int run=0;run<100;run++)for(int fi=0;fi<document.Functions.count;fi++){
     concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,document,fi);
     REQUIRE(!owner.tag);function=F(empty_function)();placement=F(empty_object_placement)();allocation=F(empty_allocation)();frame=F(empty_finalized_frame)();plan=F(empty_aggregate_function_plan)();
     concept_result_void_aggregate_abierror prepared=F(prepare_aggregate_function)(&reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan);REQUIRE(!prepared.tag);
     unsigned long long hash=fingerprint();REQUIRE(!debugOverflow);
     if(run==0&&fi==ORD_FloatingPressure){int spills=0;for(int id=0;id<function.virtualRegCount;id++)spills+=allocation.spilled.data[id];REQUIRE(spills>0);printf("tagged floating-pressure spills=%d\n",spills);}
     if(run==0){first[fi]=hash;memcpy(firstText[fi],debugText,debugLength+1);if(frame.byteCount<min_frame)min_frame=frame.byteCount;if(frame.byteCount>max_frame)max_frame=frame.byteCount;}
     else{REQUIRE(first[fi]==hash&&!strcmp(firstText[fi],debugText));}
     if(run==99&&fi==ORD_Pressure){
       int saved=frame.slots.data[frame.slotCount-1].offset;frame.slots.data[frame.slotCount-1].offset=0;
       REQUIRE(F(validate_aggregate_function_plan)(&reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);frame.slots.data[frame.slotCount-1].offset=saved;
       plan.parameters.data[0].type.identity=plan.identity;REQUIRE(F(validate_aggregate_function_plan)(&reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);
     }
     if(run==99&&fi==ORD_MakeInteger){plan.result.category.tag=5;REQUIRE(F(validate_aggregate_function_plan)(&reader,document,owner.payload.ok.value,&function,&placement,&allocation,&frame,&plan).tag);}
   }
   printf("100 complete physical plans %.3f ms; frames %d..%d\n",1000.0*(double)(clock()-planning)/CLOCKS_PER_SEC,min_frame,max_frame);
 }
 for(int run=0;run<(argc>1?0:100);run++){
   concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){again,sizeof again},&other);
   REQUIRE(!r.tag&&other.finalized&&other.byteCount==image.byteCount&&other.symbolCount==image.symbolCount&&other.fixupCount==image.fixupCount&&!memcmp(code,again,image.byteCount));
   REQUIRE(!memcmp(&image,&other,sizeof image));
 }
 FILE*dump=fopen(/* DUMP */,"wb");REQUIRE(dump);REQUIRE(fwrite(code,1,image.byteCount,dump)==(size_t)image.byteCount);fclose(dump);
 unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,code,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){
   SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);native_begin=memory;native_size=image.byteCount;REQUIRE(AddVectoredExceptionHandler(1,expected_trap));
   SYSTEM_INFO info;GetSystemInfo(&info);size_t page=info.dwPageSize;
   unsigned char*guard=VirtualAlloc(NULL,page*2,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(guard);REQUIRE(VirtualProtect(guard+page,page,PAGE_NOACCESS,&old));
   uint32_t tag=atoi(argv[1])==1?0:99;memcpy(guard+page-4,&tag,4);
   const void*input=guard+page-4;ProbeToken decimal;memset(&decimal,0,sizeof decimal);decimal.tag=2;decimal.payload.decimal=3.5;
   if(atoi(argv[1])==3)input=&decimal;
   typedef int(*RawGet)(const void*);RawGet get;void*p=memory+image.symbols.data[ORD_NativeGetInteger].offset;memcpy(&get,&p,sizeof get);(void)get(input);return 99;
 }
 /* Child processes share a finished executable, never a linker output path. */
 for(int trap=1;trap<=3;trap++){
   char exe[MAX_PATH],command[MAX_PATH+32];REQUIRE(GetModuleFileNameA(NULL,exe,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %d",exe,trap);
   STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;
   REQUIRE(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));REQUIRE(WaitForSingleObject(process.hProcess,30000)==WAIT_OBJECT_0);DWORD status;REQUIRE(GetExitCodeProcess(process.hProcess,&status));CloseHandle(process.hThread);CloseHandle(process.hProcess);REQUIRE(status==0);
 }
 REQUIRE(FN(MakeEnd,EndFn)().tag==0);
 for(int run=0;run<100;run++){
   ProbeToken a=FN(MakeInteger,IntFn)(run-50);REQUIRE(a.tag==1&&a.payload.integer==run-50);REQUIRE(FN(NativeGetInteger,GetIntFn)(a)==run-50);
   ProbeToken b=FN(MakeDecimal,DoubleFn)(run?3.5:-0.0);REQUIRE(b.tag==2&&bits64(b.payload.decimal)==bits64(run?3.5:-0.0));REQUIRE(bits64(FN(NativeGetDouble,GetDoubleFn)(b))==bits64(b.payload.decimal));
   ProbeToken c=FN(MakeReal,FloatFn)(run?1.25f:-0.0f);REQUIRE(c.tag==3&&bits32(FN(NativeGetFloat,GetFloatFn)(c))==bits32(c.payload.real));
   ProbeToken d=FN(MakeFlag,BoolFn)(run%2);REQUIRE(d.tag==4&&FN(NativeGetFlag,GetBoolFn)(d)==(run%2!=0));
   ProbeToken e=FN(Nested,IntFn)(run);REQUIRE(e.tag==5&&e.payload.many.count==run&&e.payload.many.ready&&e.payload.many.item.count==run+1&&e.payload.many.item.amount==3.5);
   ProbeItem item=FN(NativeGetItem,GetItemFn)(e);REQUIRE(item.count==run+1&&item.amount==3.5);ProbeArray arr=FN(NativeGetArray,GetArrayFn)(e);REQUIRE(arr.data[0]==4&&arr.data[1]==5&&arr.data[2]==6);
   ProbeToken f=FN(MakeStamped,IntFn)(run);ProbeStamp stamp;REQUIRE(FN(NativeGetStamp,GetStampFn)(&stamp,f)==&stamp&&stamp.value==run);
   ProbeToken copied=FN(CopyAndReplace,ReplaceFn)(e,0);REQUIRE(copied.tag==5&&copied.payload.many.count==run&&e.tag==5);
   copied=FN(Recur,ReplaceFn)(e,run%5);REQUIRE(copied.tag==5&&copied.payload.many.item.count==run+1);
   ProbeToken changed=FN(CopyAndReplace,ReplaceFn)(e,run+1);REQUIRE(changed.tag==1&&changed.payload.integer==run&&e.payload.many.item.count==run+1);
   ProbeToken pressure=FN(Pressure,PressureFn)(e,3,2.5,a,1.25f,1,b);REQUIRE(pressure.tag==5&&pressure.payload.many.count==run);
   pressure=FN(Pressure,PressureFn)(e,3,2.5,a,1.25f,0,b);REQUIRE(pressure.tag==1&&pressure.payload.integer==2);
   pressure=FN(Pressure,PressureFn)(e,0,2.5,a,1.25f,0,b);REQUIRE(pressure.tag==2&&pressure.payload.decimal==b.payload.decimal);
   ProbeToken floating=FN(FloatingPressure,FloatingFn)(1,2,3,4,5,6,7+run);REQUIRE(floating.tag==2&&floating.payload.decimal==52.5+run);
   REQUIRE(FN(Select,BoolFn)(1).tag==2&&FN(Select,BoolFn)(0).tag==0);
   ProbeSmall s=FN(MakeOther,SmallFn)(run),t=FN(MakeTwin,SmallFn)(-run);REQUIRE(s.tag==1&&s.value==run&&t.tag==1&&t.value==-run);
   s=FN(SmallCall,SmallFn)(run);REQUIRE(s.tag==1&&s.value==run);
   int counter=run;ProbeOrdered order=FN(MakeOrdered,OrderedFn)(&counter);REQUIRE(order.tag==0&&order.first==run+1&&order.second==run+2&&counter==run+2);
#ifndef TAGGED_NATIVE_ONLY
   concept_token oracle=concept_testing__tagged_values_copy_and_replace(concept_testing__tagged_values_make_many(run),run+1);
   REQUIRE(oracle.tag==changed.tag&&oracle.payload.integer.value==changed.payload.integer);
   oracle=concept_testing__tagged_values_make_decimal(b.payload.decimal);REQUIRE(oracle.tag==b.tag&&bits64(oracle.payload.decimal.value)==bits64(b.payload.decimal));
   oracle=concept_testing__tagged_values_nested(run);REQUIRE(oracle.tag==e.tag&&oracle.payload.many.count==e.payload.many.count&&oracle.payload.many.item.count==e.payload.many.item.count&&oracle.payload.many.numbers.data[2]==e.payload.many.numbers[2]);
   oracle=concept_testing__tagged_values_floating_pressure(1,2,3,4,5,6,7+run);REQUIRE(oracle.tag==floating.tag&&bits64(oracle.payload.decimal.value)==bits64(floating.payload.decimal));
#endif
 }
 const uint64_t doubles[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};
 for(unsigned i=0;i<4;i++){double input;memcpy(&input,&doubles[i],8);ProbeToken v=FN(Forward,ForwardFn)(FN(MakeDecimal,DoubleFn)(input));REQUIRE(v.tag==2&&bits64(v.payload.decimal)==doubles[i]&&bits64(FN(NativeGetDouble,GetDoubleFn)(v))==doubles[i]);}
 const uint32_t floats[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};
 for(unsigned i=0;i<4;i++){float input;memcpy(&input,&floats[i],4);ProbeToken v=FN(Forward,ForwardFn)(FN(MakeReal,FloatFn)(input));REQUIRE(v.tag==3&&bits32(v.payload.real)==floats[i]&&bits32(FN(NativeGetFloat,GetFloatFn)(v))==floats[i]);}
 REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_READWRITE,&old));
 unsigned char*stub=memory+image.symbols.data[ORD_Forward].offset;ForwardFn token_target=foreign_token;stub[0]=0x48;stub[1]=0xb8;memcpy(stub+2,&token_target,8);stub[10]=0xff;stub[11]=0xe0;
 stub=memory+image.symbols.data[ORD_SmallForward].offset;typedef ProbeSmall(*SmallForwardFn)(ProbeSmall);SmallForwardFn small_target=foreign_small;stub[0]=0x48;stub[1]=0xb8;memcpy(stub+2,&small_target,8);stub[10]=0xff;stub[11]=0xe0;
 REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 for(int run=0;run<100;run++){ProbeToken value=FN(Nested,IntFn)(run);REQUIRE(value.tag==5&&value.payload.many.count==run);ProbeSmall value2=FN(SmallCall,SmallFn)(run);REQUIRE(value2.tag==1&&value2.value==run);}
 REQUIRE(callback_count==300);
 for(int di=0;probe_dlls[di];di++){
   HMODULE dll=LoadLibraryA(probe_dlls[di]);REQUIRE(dll);typedef int(*Observe)(const void*const*);typedef int(*Count)(void);Observe observe;Count count;
   FARPROC address=GetProcAddress(dll,"CallTagged");REQUIRE(address);memcpy(&observe,&address,sizeof observe);address=GetProcAddress(dll,"TaggedCalls");REQUIRE(address);memcpy(&count,&address,sizeof count);
   const void*table[]={memory+image.symbols.data[ORD_MakeInteger].offset,memory+image.symbols.data[ORD_MakeOther].offset,memory+image.symbols.data[ORD_MakeTwin].offset,memory+image.symbols.data[ORD_MakeOrdered].offset,memory+image.symbols.data[ORD_Pressure].offset,memory+image.symbols.data[ORD_Nested].offset,memory+image.symbols.data[ORD_FloatingPressure].offset};
   REQUIRE(observe(table)==0);
   REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_READWRITE,&old));
   stub=memory+image.symbols.data[ORD_Forward].offset;address=GetProcAddress(dll,"TaggedRelay");REQUIRE(address);memcpy(stub+2,&address,8);
   stub=memory+image.symbols.data[ORD_SmallForward].offset;address=GetProcAddress(dll,"SmallRelay");REQUIRE(address);memcpy(stub+2,&address,8);
   REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
   for(int run=0;run<100;run++){ProbeToken value=FN(Nested,IntFn)(run);REQUIRE(value.tag==5&&value.payload.many.count==run);ProbeSmall small=FN(SmallCall,SmallFn)(run);REQUIRE(small.tag==1&&small.value==run);}
   REQUIRE(count()==300);printf("independent tagged ABI passed: %s\n",probe_dlls[di]);
   /* Keep probes loaded while the test image contains their call targets. */
 }
 VirtualFree(memory,0,MEM_RELEASE);puts("100 native tagged executions and emissions passed");return 0;
}
`

func nativeTaggedFixture(t *testing.T) (Module, []byte, LIRModule, MachineModule, []byte) {
	t.Helper()
	module, source, lir := taggedFixture(t)
	var token LIRTaggedEnum
	for _, d := range lir.TaggedEnums {
		if d.Name == "Token" {
			token = d
		}
	}
	for _, p := range []struct {
		name           string
		variant, field int
	}{{"NativeGetInteger", 1, 0}, {"NativeGetDouble", 2, 0}, {"NativeGetFloat", 3, 0}, {"NativeGetFlag", 4, 0}, {"NativeGetItem", 5, 2}, {"NativeGetArray", 5, 3}, {"NativeGetStamp", 6, 0}} {
		typ := token.Variants[p.variant].Fields[p.field].Type
		function := LIRFunction{Identity: p.name + "|" + p.name + "(Token)", Name: p.name, Params: []LIRValue{{ID: 0, Type: token.Type}}, Result: typ, Source: Span{Line: 1, Column: 1}, Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{{Op: "checked_payload", Result: 1, Type: typ, Args: []int{0}, Slot: -1, Aggregate: token.Type, Member: p.variant, Extent: p.field, Source: Span{Line: 1, Column: 1}}}, Term: LIRTerminator{Op: "return", Value: 1, Source: Span{Line: 1, Column: 1}}}}}
		if p.name == "NativeGetArray" {
			for _, layout := range lir.Aggregates {
				if layout.Name == "ArrayBox" {
					function.Result = layout.Type
					function.Blocks[0].Instructions = append(function.Blocks[0].Instructions, LIRInstruction{Op: "make_aggregate", Result: 2, Type: layout.Type, Args: []int{1}, Slot: -1, Source: Span{Line: 1, Column: 1}})
					function.Blocks[0].Term.Value = 2
				}
			}
		}
		lir.Functions = append(lir.Functions, function)
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

func TestEVT2h2TaggedTransport(t *testing.T) {
	start := time.Now()
	_, _, lir, m, data := nativeTaggedFixture(t)
	decoded, err := DecodeMachineBridge(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bridgeSemanticEqual(reflect.ValueOf(*m.Tagged), reflect.ValueOf(*decoded.Tagged)) {
		t.Fatal("tagged round trip differs")
	}
	for i := 0; i < 100; i++ {
		_, _, again, other, raw := nativeTaggedFixture(t)
		if !bytes.Equal(raw, data) || again.String() != lir.String() || other.String() != m.String() {
			t.Fatalf("generation %d differs", i)
		}
	}
	path := "../../libraries/Standard/Backend/BridgeSchema8.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateMachineBridgeCodecs(schema)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{"../../internal/concept/machineir_bridge8_codec_generated.go": generated.Go, "../../libraries/Standard/Backend/BridgeCodec8.concept": generated.Concept, "../../docs/design/EVT2-MACHINEIR-BRIDGE8.schema.json": generated.Metadata} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("generated file differs %s: %v", path, err)
		}
	}
	for i := 0; i < 100; i++ {
		g, err := GenerateMachineBridgeCodecs(schema)
		if err != nil || !bytes.Equal(g.Go, generated.Go) || !bytes.Equal(g.Concept, generated.Concept) || !bytes.Equal(g.Metadata, generated.Metadata) {
			t.Fatalf("codec generation %d differs: %v", i, err)
		}
	}
	t.Logf("CMIRAMD8 bytes=%d sha256=%x functions=%d generation+codec100=%s", len(data), sha256.Sum256(data), len(m.Tagged.Base.Objects.Functions), time.Since(start))
}

var taggedAdmissionBuild [2]struct {
	once    sync.Once
	outputs Outputs
	err     error
}

func taggedAdmissionOutputs(t *testing.T, verify bool) Outputs {
	t.Helper()
	requireNativeQualification(t)
	index := 0
	if verify {
		index = 1
	}
	build := &taggedAdmissionBuild[index]
	build.once.Do(func() {
		path := "../../libraries/Standard/Backend/BridgeValidate8.concept"
		source, err := os.ReadFile(path)
		if err != nil {
			build.err = err
			return
		}
		m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
		if err != nil {
			build.err = err
			return
		}
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		build.outputs, build.err = GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
	})
	if build.err != nil {
		t.Fatal(build.err)
	}
	out := Outputs{}
	for k, v := range build.outputs {
		out[k] = v
	}
	return out
}

func rawTagged(t *testing.T, m TaggedModule) []byte {
	t.Helper()
	data, err := encodeTaggedMachineBridge(m)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEVT2h2LocalModulePreflight(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	path := "../../tests/runtime/Testing/TaggedLocals.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(module)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	var defs strings.Builder
	defs.WriteString("static const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&defs, "%d,", b)
	}
	defs.WriteString("};\n")
	host := `#include "amd64.generated.h"
#include <windows.h>
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
/* ARTIFACT */
static unsigned char code[65536];static concept_native_image image,before;
int main(void){concept_readonly_span_byte src={artifact,sizeof artifact};if(F(validate_tagged_machine_bridge)(src).tag)return 1;
concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);if(decoded.tag)return 8;concept_bridge_reader reader={src,0};if(F(object_module_has_aggregate_abi)(&reader,decoded.payload.ok.value))return 9;
memset(code,0xa5,sizeof code);memset(&image,0x5a,sizeof image);before=image;
concept_result_int_native_error tiny=F(emit_module)(src,(concept_span_byte){code,1},&image);if(!tiny.tag||memcmp(&image,&before,sizeof image))return 2;
for(size_t i=0;i<sizeof code;i++)if(code[i]!=0xa5)return 3;
concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){code,sizeof code},&image);if(r.tag||!image.finalized||image.symbolCount!=1)return 4;
unsigned char*memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 5;memcpy(memory,code,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 6;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
typedef int(*Entry)(int);Entry entry;void*p=memory+image.symbols.data[0].offset;memcpy(&entry,&p,sizeof entry);for(int i=-50;i<50;i++)if(entry(i)!=i)return 7;VirtualFree(memory,0,MEM_RELEASE);return 0;}`
	host = strings.Replace(host, "/* ARTIFACT */", defs.String(), 1)
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native execution requires Windows AMD64")
	}
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "tagged_local_preflight.c", host, "-pedantic-errors", "-O2")
	}
}
func TestEVT2h2TaggedArtifactAdmission(t *testing.T) {
	t.Parallel()
	requireNativeQualification(t)
	_, _, _, machine, valid := nativeTaggedFixture(t)
	blob, err := json.Marshal(machine.Tagged)
	if err != nil {
		t.Fatal(err)
	}
	type specimen struct {
		name string
		data []byte
	}
	cases := []specimen{}
	add := func(name string, mutate func(*TaggedModule)) {
		var m TaggedModule
		if err := json.Unmarshal(blob, &m); err != nil {
			t.Fatal(err)
		}
		mutate(&m)
		data := rawTagged(t, m)
		if _, err := DecodeMachineBridge(data); err == nil {
			t.Fatalf("Go accepted %s", name)
		}
		cases = append(cases, specimen{name, data})
	}
	add("missing domains", func(m *TaggedModule) { m.Domains = nil })
	add("duplicate domain", func(m *TaggedModule) { m.Domains = append(m.Domains, m.Domains[0]) })
	add("variant tag", func(m *TaggedModule) { m.Domains[0].Variants[0].Tag = 12 })
	add("variant identity", func(m *TaggedModule) { m.Domains[0].Variants[0].Identity = "other" })
	add("unknown domain type", func(m *TaggedModule) { m.Domains[0].Type = 999 })
	add("duplicate variant", func(m *TaggedModule) { m.Domains[1].Variants[1] = m.Domains[1].Variants[0] })
	add("variant capacity", func(m *TaggedModule) { m.Domains[0].Variants = make([]TaggedVariant, 257) })
	add("payload geometry", func(m *TaggedModule) { m.Domains[0].PayloadOffset = 1 })
	add("field overlap", func(m *TaggedModule) { m.Domains[0].Variants[0].Fields[1].Offset = 0 })
	add("copy authority", func(m *TaggedModule) { m.Base.Objects.Types[m.Domains[0].Type].Copy = "none" })
	add("opaque root fields", func(m *TaggedModule) {
		m.Base.Objects.Types[m.Domains[0].Type].Fields = m.Domains[0].Variants[0].Fields
	})
	add("invalid alignment", func(m *TaggedModule) { m.Base.Objects.Types[m.Domains[0].Type].Align = 3 })
	add("oversize storage", func(m *TaggedModule) { m.Base.Objects.Types[m.Domains[0].Type].Size = 1048577 })
	add("missing operation authority", func(m *TaggedModule) { m.Operations = nil })
	add("duplicate operation", func(m *TaggedModule) { m.Operations = append(m.Operations, m.Operations[0]) })
	add("unknown operation site", func(m *TaggedModule) { m.Operations[0].Instruction = 9999 })
	add("constructor initialization", func(m *TaggedModule) {
		s := m.Operations[0]
		m.Base.Objects.Functions[s.Function].Blocks[s.Block].Instructions[s.Instruction].Args = nil
	})
	add("unguarded project", func(m *TaggedModule) {
		for i, op := range m.Operations {
			if op.Kind == "checked_payload" {
				m.Operations = append(m.Operations[:i], m.Operations[i+1:]...)
				return
			}
		}
	})
	add("invalid project variant", func(m *TaggedModule) {
		for _, op := range m.Operations {
			if op.Kind == "checked_payload" {
				m.Base.Objects.Functions[op.Function].Blocks[op.Block].Instructions[op.Instruction].Member = 99
				return
			}
		}
	})
	add("invalid project field", func(m *TaggedModule) {
		for _, op := range m.Operations {
			if op.Kind == "checked_payload" {
				m.Base.Objects.Functions[op.Function].Blocks[op.Block].Instructions[op.Instruction].Extent = 99
				return
			}
		}
	})
	add("nominal return", func(m *TaggedModule) {
		for i := range m.Base.Objects.Functions {
			if m.Base.Objects.Functions[i].Name == "MakeOther" {
				m.Base.Objects.Functions[i].ResultType = m.Domains[3].Type
				return
			}
		}
	})
	add("nominal call", func(m *TaggedModule) {
		for fi := range m.Base.Objects.Functions {
			for bi := range m.Base.Objects.Functions[fi].Blocks {
				for ii := range m.Base.Objects.Functions[fi].Blocks[bi].Instructions {
					in := &m.Base.Objects.Functions[fi].Blocks[bi].Instructions[ii]
					if len(in.Calls) > 0 && strings.Contains(in.Calls[0].Target, "SmallForward|") {
						in.Calls[0].Arguments[0].Type = m.Domains[3].Type
						return
					}
				}
			}
		}
		t.Fatal("missing nominal call specimen")
	})
	for _, offset := range []int{0, 8, 12, len(valid) - 1} {
		raw := bytes.Clone(valid)
		raw[offset] ^= 255
		cases = append(cases, specimen{fmt.Sprintf("corrupt header or operation %d", offset), raw})
	}
	cases = append(cases, specimen{"truncated", valid[:len(valid)-1]}, specimen{"trailing", append(append([]byte(nil), valid...), 0)})
	for _, c := range cases {
		if _, err := DecodeMachineBridge(c.data); err == nil {
			t.Fatalf("Go accepted %s", c.name)
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.cmir8")
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	var defs strings.Builder
	fmt.Fprintf(&defs, "#define VALID %q\n", filepath.ToSlash(path))
	for i, c := range cases {
		p := filepath.Join(dir, fmt.Sprintf("invalid-%d.cmir8", i))
		if err := os.WriteFile(p, c.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&defs, "{FILE*f=fopen(%q,\"rb\");if(!f)return 1;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte s={input,n};memset(output,0xa5,sizeof output);concept_result_int_tagged_error r=F(validated_tagged_bridge_round_trip)(s,dst);if(!r.tag){fprintf(stderr,\"accepted %s\\n\");return 2;}/* NATIVE REJECTION */for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 3;}\n", filepath.ToSlash(p), c.name)
	}
	host := `#include "bridgevalidate8.generated.h"
#include <stdio.h>
#include <string.h>
#define F(n) concept_standard__backend__bridge_validate8_##n
static unsigned char input[2000000],output[2000000];
int main(void){concept_span_byte dst={output,sizeof output};
/* CASES */
FILE*f=fopen(VALID,"rb");if(!f)return 4;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte src={input,n};
for(int run=0;run<100;run++){concept_result_int_tagged_error r=F(validated_tagged_bridge_round_trip)(src,dst);if(r.tag){fprintf(stderr,"%s\n",F(describe_tagged_error)(r.payload.error.error));return 5;}if((size_t)r.payload.ok.value!=n||memcmp(input,output,n))return 6;}
return 0;}`
	// The host sees bytes and the Concept admission module only, no domain source.
	host = strings.Replace(host, "/* CASES */", defs.String(), 1)
	// Preprocessor definitions must precede their uses, including the valid path.
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, taggedAdmissionOutputs(t, verify), "tagged_admission.c", host, "-pedantic-errors", "-O2")
		if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
			nativeHost := strings.Replace(host, "bridgevalidate8.generated.h", "amd64.generated.h", 1)
			nativeHost = strings.ReplaceAll(nativeHost, "concept_standard__backend__bridge_validate8_", "concept_standard__backend__amd64_")
			nativeHost = strings.Replace(nativeHost, "int main(void)", "static concept_native_image image,before;int main(void)", 1)
			nativeHost = strings.ReplaceAll(nativeHost, "/* NATIVE REJECTION */", `memset(&image,0x5a,sizeof image);before=image;concept_result_int_native_error emitted=F(emit_module)(s,dst,&image);if(!emitted.tag||memcmp(&image,&before,sizeof image))return 7;`)
			runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "tagged_native_rejection.c", nativeHost, "-pedantic-errors", "-O2")
		}
	}
	t.Logf("artifact-only admission Normal/Verify: %d invalid cases, 100 exact valid round trips", len(cases))
}
