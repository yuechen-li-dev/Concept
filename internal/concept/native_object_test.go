package concept

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2g3NativeObjects(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Win64 native execution requires Windows AMD64")
	}
	path := "../../language/evt2g3/valid/objects.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	fields, fieldPaths, strides, layouts := 0, 0, 0, 0
	for _, typ := range machine.Object.Types {
		fields += len(typ.Fields)
		if typ.Kind == "aggregate" {
			layouts++
		}
	}
	for _, fn := range machine.Object.Functions {
		for _, block := range fn.Blocks {
			for _, in := range block.Instructions {
				if in.Op == "field_address" {
					fieldPaths++
				}
				if in.Op == "index_address" {
					strides++
				}
			}
		}
	}
	t.Logf("native corpus: layouts=%d checked fields=%d field derivations=%d transported-stride operations=%d artifact=%d bytes", layouts, fields, fieldPaths, strides, len(artifact))
	var data strings.Builder
	data.WriteString("static unsigned char artifact[]={")
	for _, b := range artifact {
		fmt.Fprintf(&data, "%d,", b)
	}
	data.WriteString("};\n")
	for i, f := range machine.Object.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", f.Name, i)
	}
	fmt.Fprintf(&data, "#define OBJECT_COUNT %d\n", len(machine.Object.Functions))
	harness := strings.Replace(nativeObjectsHarness, "/* ARTIFACT */", data.String(), 1)
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			runFoundationNativeHarnessOutput(t, outputs, "native_objects_artifact_only.c", harness, "-pedantic-errors", "-O2", "-DOBJECT_NATIVE_ONLY=1")
			policy := ConservativeCompilationPolicy()
			if verify {
				policy = VerifyCompilationPolicy()
			}
			oracle, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
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
			runFoundationNativeHarnessOutput(t, outputs, "native_objects.c", harness, "-pedantic-errors", "-O2")
		})
	}
}

func TestEVT2g3BootstrapShortCircuitBounds(t *testing.T) {
	const source = `module Guarded; profile Core;
bool Both(int index){int<array>[2] values=[1,2];return index>=0 and values[index]>0;}
bool Either(int index){int<array>[2] values=[1,2];return index<0 or values[index]>0;}
int Bump(ref int trace){trace=trace+1;return 0;}
int Effects(){int trace=0;int<array>[2] values=[1,2];bool a=false and values[Bump(ref trace)]>0;bool b=true or values[Bump(ref trace)]>0;return trace;}`
	m, err := Parse("guarded.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		out, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarnessOutput(t, out, "guarded.c", "#include \"guarded.generated.h\"\nint main(void){return !concept_guarded_both(-1)&&concept_guarded_either(-1)&&concept_guarded_both(1)&&concept_guarded_either(1)&&concept_guarded_effects()==0?0:1;}\n", "-pedantic-errors", "-O2")
	}
}

func TestEVT2g3LayoutProofsAndReferenceBoundary(t *testing.T) {
	path := "../../language/evt2g3/valid/objects.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileSemanticModule(path, string(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources([]string{"../../libraries"}, []string{"Standard.Semantic.Vocabulary"})
	if err != nil {
		t.Fatal(err)
	}
	artifacts["EVT2g3.Objects"] = artifact
	const proof = `module ObjectFacts; profile Core; import Standard.Semantic.Vocabulary; import EVT2g3.Objects;
using SampleArray=double<array>[4];
void Facts(){Assert.Concept<PlainData>(Pair,"native numeric record fields");Assert.Concept<PlainData>(Group,"native nested record fields");Assert.Concept<PlainData>(RecordSamples,"native array record fields");Assert.Concept<StaticExtent<4>>(SampleArray,"qualified one-dimensional array field type");}`
	if _, err := ParseWithSemanticModules("facts.concept", proof, artifacts); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWithSemanticModules("mutablefacts.concept", strings.Replace(proof, "(Pair,", "(Position,", 1), artifacts); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("PlainData mutable-object boundary changed: %v", err)
	}
	const reference = `module SourceRefs; profile Core; struct Position{double x;double y;}
void Change(ref Position p){p.x=7.0;}
double Read(ref const Position p){return p.x+p.y;}
double Main(){Position p=Position{1.0,2.0};Change(ref p);return Read(ref const p);}`
	m, err := Parse("sourcerefs.concept", reference)
	if err != nil {
		t.Fatal(err)
	}
	if machine, err := GenerateMachineIR(m); err != nil || machine.Borrow == nil {
		t.Fatalf("source borrow contract must use CMIRAMD6: %v", err)
	}
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		out, err := GenerateForTargetWithPolicy(m, []byte(reference), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarnessOutput(t, out, "source_refs.c", "#include \"sourcerefs.generated.h\"\nint main(void){return concept_source_refs_main()==9.0?0:1;}\n", "-pedantic-errors", "-O2")
	}
}

func TestEVT2g3NativeAdmissionNegatives(t *testing.T) {
	fixturePath := "../../language/evt2g3/valid/objects.concept"
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(fixturePath, string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var fields []machineBridgeWireField
	w := &machineBridgeWriter{trace: &fields}
	if err := w.writeWireObjectModule(*machine.Object); err != nil {
		t.Fatal(err)
	}
	offset := ObjectMachineBridgeHeaderSize + bridgeFieldOffset(t, fields, "WireObjectField", "Offset")
	malformed := bytes.Clone(valid)
	binary.LittleEndian.PutUint32(malformed[offset:], 1048576)
	copyPermission := bytes.Clone(valid)
	copyOffset := ObjectMachineBridgeHeaderSize + bridgeFieldOffset(t, fields, "WireObjectType", "Copy")
	binary.LittleEndian.PutUint32(copyPermission[copyOffset:], 0)
	const large = `module Large; profile Core; struct Huge{double<array>[256] values;} double Main(double value,int index){Huge a=Huge{[value ...]};Huge b=a;return b.values[index];}`
	lm, err := Parse("large.concept", large)
	if err != nil {
		t.Fatal(err)
	}
	largeMachine, err := GenerateMachineIR(lm)
	if err != nil {
		t.Fatal(err)
	}
	largeBytes, err := EncodeMachineBridge(largeMachine)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, want string
		bytes      []byte
	}{{"field_bounds", "AMD64_NATIVE_INVALID_INPUT", malformed}, {"copy_permission", "AMD64_NATIVE_INVALID_INPUT", copyPermission}, {"probe", "AMD64_STACK_PROBE_REQUIRED", largeBytes}}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\nstatic unsigned char input[262144],output[262144];static concept_native_image image,before;int main(void){FILE *f;size_t n;concept_result_int_native_error r;concept_span_byte dst={output,sizeof output};\n")
	dir := t.TempDir()
	for i, tc := range cases {
		path := filepath.Join(dir, tc.name+".cmir")
		if err := os.WriteFile(path, tc.bytes, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&c, "f=fopen(%q,\"rb\");if(!f)return 99;n=fread(input,1,sizeof input,f);fclose(f);memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_readonly_span_byte src%d={input,n};r=concept_standard__backend__amd64_emit_module(src%d,dst,&image);if(!r.tag||strcmp(concept_standard__backend__amd64_describe_native_error(r.payload.error.error),%q)){fprintf(stderr,\"%s: %%s\\n\",r.tag?concept_standard__backend__amd64_describe_native_error(r.payload.error.error):\"ACCEPTED\");return %d;}if(memcmp(&image,&before,sizeof image))return 98;for(size_t j=0;j<sizeof output;j++){if(output[j]!=0xa5)return 97;}\n", filepath.ToSlash(path), i, i, tc.want, tc.name, i+1)
	}
	c.WriteString("puts(\"invalid geometry/copy and unprobed frames reject before output/image mutation\");return 0;}\n")
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "admission.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g3MemoryEncodingGoldens(t *testing.T) {
	path := "../../language/evt2g3/valid/objects.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	mm, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(mm)
	if err != nil {
		t.Fatal(err)
	}
	pair := -1
	for i, typ := range mm.Object.Types {
		if typ.Name == "Pair" {
			pair = i
		}
	}
	if pair < 0 {
		t.Fatal("missing record")
	}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\nstatic const unsigned char artifact[]={")
	for _, b := range artifact {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};\n")
	fmt.Fprintf(&c, `#define F(n) concept_standard__backend__amd64_##n
#define EXPECT(call,bytes) do{w.offset=0;if((call).tag||w.offset!=(int)(sizeof(bytes)-1)||memcmp(out,bytes,sizeof(bytes)-1)){fprintf(stderr,"golden at %%d\n",__LINE__);return __LINE__;}}while(0)
int main(void){unsigned char out[128];concept_byte_writer w={{out,sizeof out},0};concept_register rax={0},rdx={3},r10={10};concept_xmmregister xmm2={2};concept_scalar_representation f32={10},f64={11},i8={2},i16={4};
EXPECT(F(write_address_reg)(&w,139,r10,2,-1,0,8,8),"\x4c\x8b\x52\x08");
EXPECT(F(write_address_reg)(&w,137,r10,2,-1,0,8,8),"\x4c\x89\x52\x08");
EXPECT(F(write_address_reg)(&w,139,r10,2,-1,0,6,2),"\x66\x44\x8b\x52\x06");
EXPECT(F(write_scalar_address_sse)(&w,0,f64,xmm2,rdx),"\xf2\x0f\x10\x12");
EXPECT(F(write_scalar_address_sse)(&w,1,f32,xmm2,rdx),"\xf3\x0f\x11\x12");
EXPECT(F(write_address_reg)(&w,141,rax,0,-1,0,16,8),"\x48\x8d\x40\x10");
EXPECT(F(write_address_reg)(&w,141,rax,0,2,0,0,8),"\x48\x8d\x04\x10");
EXPECT(F(object_stack_address)(&w,rax,32),"\x48\x8d\x44\x24\x20");
EXPECT(F(extend_object_integer)(&w,r10,i8),"\x4d\x0f\xbe\xd2");
EXPECT(F(extend_object_integer)(&w,r10,i16),"\x4d\x0f\xbf\xd2");
concept_readonly_span_byte src={artifact,sizeof artifact};concept_bridge_reader reader={src,0};concept_result_wire_object_module_backend_error decoded=F(native_object_document)(src);if(decoded.tag)return 98;
EXPECT(F(copy_object_fields)(&w,&reader,decoded.payload.ok.value,%d,0,0),"\x4c\x8b\x10\x4c\x89\x12\x4c\x8b\x50\x08\x4c\x89\x52\x08");
if(!F(write_address_reg)(&w,139,rax,0,-1,4,0,8).tag)return 99;
puts("11 exact address/load/store/extension/copy goldens passed");return 0;}
`, pair)
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "memory_goldens.c", c.String(), "-pedantic-errors", "-O2")
	}
}

const nativeObjectsHarness = `#include "amd64.generated.h"
#ifndef OBJECT_NATIVE_ONLY
#include "oracle.c"
#define DETERMINISM_RUNS 100
#else
#define DETERMINISM_RUNS 1
#endif
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
/* ARTIFACT */
static unsigned char code[262144],again[262144];
static concept_native_image image,other;
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
typedef double(*Unary)(double);typedef double(*Index)(int);typedef double(*Padded)(_Bool,double,int);typedef double(*Early)(_Bool,double);
typedef double(*NestedArray)(int,double);typedef _Bool(*Flag)(_Bool);typedef uint16_t(*Short)(uint16_t);typedef uint8_t(*Bytes)(uint8_t,uint8_t);typedef uint16_t(*Words)(uint16_t,uint16_t);typedef ptrdiff_t(*SignedWide)(ptrdiff_t);typedef uint64_t(*Wide)(uint64_t);typedef int(*Integer)(int);typedef int(*Triple)(int,int,int);typedef double(*Mixed)(int,_Bool,float,double);typedef int(*Nullary)(void);
typedef double(*Index64)(ptrdiff_t);
typedef uint64_t(*WideConstant)(void);typedef uint32_t(*UnsignedConstant)(void);
typedef int(*Int2)(int,int);
#define GETTER(t) static t Get##t(unsigned char *memory,int ordinal){t f=NULL;void *p=memory+image.symbols.data[ordinal].offset;memcpy(&f,&p,sizeof f);return f;}
GETTER(Unary) GETTER(Index) GETTER(Padded) GETTER(Early)
GETTER(NestedArray) GETTER(Flag) GETTER(Short) GETTER(Bytes) GETTER(Words) GETTER(SignedWide) GETTER(Wide) GETTER(Integer) GETTER(Triple) GETTER(Mixed) GETTER(Nullary)
GETTER(Index64)
GETTER(WideConstant) GETTER(UnsignedConstant)
GETTER(Int2)
#define FN(n,t) Get##t(memory,ORD_##n)
static int trace=0;
static int tick(int tag){trace=trace*10+tag;return tag;}
static int describe_plan(concept_readonly_span_byte input,int ordinal,char *text){concept_backend_function f;concept_object_placement p;concept_allocation a;concept_finalized_frame frame;if(concept_standard__backend__amd64_plan_object_native_function(input,ordinal,&f,&p,&a,&frame).tag)return 0;int n=sprintf(text,"frame=%d slots=%d regs=%d",frame.byteCount,frame.slotCount,f.virtualRegCount);for(int i=0;i<frame.slotCount;i++){concept_frame_slot s=frame.slots.data[i];n+=sprintf(text+n," %d:%d:%d:%d:%d",s.category.tag,s.identity,s.offset,s.size,s.alignment);}for(int v=0;v<512;v++){if(p.values.data[v]>=0||p.registers.data[v]>=0)n+=sprintf(text+n," v%d:%d:%d",v,p.values.data[v],p.registers.data[v]);}for(int v=0;v<f.virtualRegCount;v++){concept_scalar_location s=a.scalarLocations.data[v];n+=sprintf(text+n," r%d:%d:%d:%d:%d:%d",v,s.kind.tag,s.gpr.tag,s.xmm.tag,a.start.data[v],a.end.data[v]);}return 1;}
int main(int argc,char **argv){
 concept_readonly_span_byte input={artifact,sizeof artifact};concept_span_byte output={code,sizeof code};
 clock_t emit_start=clock();
 concept_result_int_native_error result=concept_standard__backend__amd64_emit_module(input,output,&image);
 if(result.tag){fprintf(stderr,"emission: %s\n",concept_standard__backend__amd64_describe_native_error(result.payload.error.error));for(int ordinal=0;ordinal<OBJECT_COUNT;ordinal++){concept_backend_function f;concept_object_placement p;concept_allocation a;concept_finalized_frame frame;concept_result_void_backend_error planned=concept_standard__backend__amd64_plan_object_native_function(input,ordinal,&f,&p,&a,&frame);fprintf(stderr,"plan %d %.*s error=%d frame=%d calls=%d\n",ordinal,f.name.length,(const char*)artifact+f.name.offset,planned.tag?planned.payload.error.error.tag:-1,frame.byteCount,f.hasCalls);}}
 REQUIRE(!result.tag&&image.finalized);
 printf("combined planning/encoding %.3f ms\n",1000.0*(double)(clock()-emit_start)/CLOCKS_PER_SEC);
 clock_t plan_start=clock();
 for(int ordinal=0;ordinal<image.symbolCount;ordinal++){concept_backend_function f;concept_object_placement p;concept_allocation a;concept_finalized_frame frame;REQUIRE(!concept_standard__backend__amd64_plan_object_native_function(input,ordinal,&f,&p,&a,&frame).tag);printf("frame %.*s bytes=%d local-slots=%d scalar-registers=%d code=%d\n",f.name.length,(const char*)artifact+f.name.offset,frame.byteCount,f.slotCount,f.virtualRegCount,image.symbols.data[ordinal].size);
 if(ordinal==0){concept_finalized_frame bad=frame;bad.slots.data[0].size--;REQUIRE(concept_standard__backend__amd64_validate_object_frame_slots(&f,&bad).tag);bad=frame;bad.slots.data[0].offset++;REQUIRE(concept_standard__backend__amd64_validate_object_frame_slots(&f,&bad).tag);}}
 printf("all-function validation/planning %.3f ms\n",1000.0*(double)(clock()-plan_start)/CLOCKS_PER_SEC);
 unsigned char *memory=(unsigned char*)VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);
 memcpy(memory,code,image.byteCount);
 /* Existing controlled test-only foreign-stub mechanism; master native bytes stay address-independent. */
 Integer host_tick=tick;unsigned char *stub=memory+image.symbols.data[ORD_Tick].offset;REQUIRE(image.symbols.data[ORD_Tick].size>=12&&sizeof host_tick==8);stub[0]=0x48;stub[1]=0xb8;memcpy(stub+2,&host_tick,8);stub[10]=0xff;stub[11]=0xe0;
 DWORD old=0;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 if(argc>1){SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);if(argv[1][0]=='w'){FN(Array64,Index64)(INT64_C(0x100000001));}else{FN(Array,Index)(argv[1][0]=='-'?-1:4);}return 99;}
 REQUIRE(FN(Simple,Unary)(4.0)==11.0);
 REQUIRE(FN(Nested,Unary)(4.0)==9.0);
 REQUIRE(FN(Array,Index)(2)==10.0);
 REQUIRE(FN(Copy,Unary)(4.0)==19.0);
 REQUIRE(FN(Padding,Padded)(1,4.0,37)==13.0);
 REQUIRE(FN(With,Unary)(4.0)==20.0);
 REQUIRE(FN(NestedWith,Unary)(4.0)==16.0);
 REQUIRE(FN(ArrayWith,Index)(2)==5.0);
 REQUIRE(FN(ScalarCall,Unary)(4.0)==15.0);
 REQUIRE(FN(Early,Early)(1,4.0)==4.0&&FN(Early,Early)(0,4.0)==12.0);
 REQUIRE(FN(NestedArray,NestedArray)(1,8.0)==12.0);
 REQUIRE(!FN(Copy1,Flag)(1)&&FN(Copy1,Flag)(0));
 REQUIRE(FN(Copy2,Short)(65530)==65530&&FN(CopyUnsigned2,Short)(65535)==65535);
 REQUIRE(FN(FirstByte,Bytes)(250,239)==250&&FN(UnsignedByte,Bytes)(250,239)==239);
 REQUIRE(FN(FirstWord,Words)(65530,65500)==65530&&FN(UnsignedWord,Words)(65530,65500)==65500);
 REQUIRE(FN(Copy4,Integer)(-123)==-123);
 REQUIRE(FN(Copy8,SignedWide)(-INT64_C(0x123456789))==-INT64_C(0x123456789));
 REQUIRE(FN(CopyUnsigned8,Wide)(UINT64_C(0xfedcba9876543210))==UINT64_C(0xfedcba9876543210));
 REQUIRE(FN(Copy12,Triple)(4,5,6)==19);
 REQUIRE(FN(Mixed,Mixed)(37,1,2.5f,7.0)==14.0);
 REQUIRE(FN(WithOrder,Nullary)()==43&&trace==1234);
 REQUIRE(FN(Loop,NestedArray)(5,3.0)==25.0&&FN(Loop,NestedArray)(0,3.0)==0.0);
 REQUIRE(FN(Array64,Index64)(2)==3.0);
 REQUIRE(FN(WideConstant,WideConstant)()==UINT64_MAX&&FN(UnsignedConstant,UnsignedConstant)()==UINT32_MAX);
 REQUIRE(FN(ArrayStride12,Int2)(1,7)==16);
 REQUIRE(FN(LiveStackCall,Unary)(4.0)==6.0);
#ifndef OBJECT_NATIVE_ONLY
 for(int run=0;run<100;run++){double x=(double)run/8.0-5.0;int index=run%4;
 REQUIRE(FN(Simple,Unary)(x)==concept_evt2g3__objects_simple(x));
 REQUIRE(FN(Nested,Unary)(x)==concept_evt2g3__objects_nested(x));
 REQUIRE(FN(Array,Index)(index)==concept_evt2g3__objects_array(index));
 REQUIRE(FN(Copy,Unary)(x)==concept_evt2g3__objects_copy(x));
 REQUIRE(FN(With,Unary)(x)==concept_evt2g3__objects_with(x));
 REQUIRE(FN(NestedWith,Unary)(x)==concept_evt2g3__objects_nested_with(x));
 REQUIRE(FN(ArrayWith,Index)(index)==concept_evt2g3__objects_array_with(index));
 REQUIRE(FN(ScalarCall,Unary)(x)==concept_evt2g3__objects_scalar_call(x));
 REQUIRE(FN(NestedArray,NestedArray)(run%3,x)==concept_evt2g3__objects_nested_array(run%3,x));}
#endif
 VirtualFree(memory,0,MEM_RELEASE);
 printf("native object functions=%d image bytes=%d\n",image.symbolCount,image.byteCount);
 char baseline[32768],next[32768];REQUIRE(describe_plan(input,ORD_NestedArray,baseline));
 for(int run=0;run<DETERMINISM_RUNS;run++){concept_span_byte dst={again,sizeof again};result=concept_standard__backend__amd64_emit_module(input,dst,&other);REQUIRE(!result.tag&&other.byteCount==image.byteCount&&!memcmp(code,again,image.byteCount));REQUIRE(describe_plan(input,ORD_NestedArray,next)&&!strcmp(baseline,next));}
 for(int bad=0;bad<3;bad++){char path[MAX_PATH],command[MAX_PATH+32];REQUIRE(GetModuleFileNameA(NULL,path,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %s",path,bad==2?"wide":bad?"-1":"4");STARTUPINFOA start;PROCESS_INFORMATION process;memset(&start,0,sizeof start);start.cb=sizeof start;REQUIRE(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&start,&process));WaitForSingleObject(process.hProcess,30000);DWORD status=0;REQUIRE(GetExitCodeProcess(process.hProcess,&status)&&status==0xc000001dUL);CloseHandle(process.hThread);CloseHandle(process.hProcess);}
 puts("native objects, exact bytes and plan description passed");return 0;}
`
