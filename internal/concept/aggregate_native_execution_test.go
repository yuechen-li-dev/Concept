package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildAggregateProbe(t *testing.T, compiler string) string {
	t.Helper()
	cl, err := exec.LookPath("cl")
	if err != nil {
		t.Skip("MSVC environment unavailable for independent probes")
	}
	vc := filepath.Dir(cl)
	for i := 0; i < 6; i++ {
		vc = filepath.Dir(vc)
	}
	setup := filepath.Join(vc, "Auxiliary", "Build", "vcvars64.bat")
	if _, err := os.Stat(setup); err != nil {
		t.Fatal(err)
	}
	tool, err := exec.LookPath(compiler)
	if err != nil {
		t.Skip(compiler + " unavailable")
	}
	dir := t.TempDir()
	for _, name := range []string{"probe.c", "boundary.asm"} {
		body, err := os.ReadFile("../../language/evt2g5/probes/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	build := fmt.Sprintf("@echo off\r\ncall \"%s\" >nul\r\nif errorlevel 1 exit /b 1\r\nml64 /nologo /c /Foboundary.obj boundary.asm\r\nif errorlevel 1 exit /b 1\r\n\"%s\" /nologo /std:c11 /O2 /MD /LD probe.c boundary.obj /link /OUT:probe.dll /IMPLIB:probe.lib /EXPORT:ProbeMixedEntry /EXPORT:ProbeRoundForward /EXPORT:ProbeSaved\r\nexit /b %%errorlevel%%\r\n", setup, tool)
	if err := os.WriteFile(filepath.Join(dir, "build.cmd"), []byte(build), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := nativeCommand(t, "cmd.exe", "/d", "/c", "build.cmd")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("independent %s DLL: %v\n%s", compiler, err, out)
	}
	t.Logf("independent compiler: %s; Microsoft x64 default ABI; separate C and ml64 boundary observer", tool)
	return filepath.Join(dir, "probe.dll")
}

func TestEVT2g5ExternalABI(t *testing.T) {
	_, _, machine, data := aggregateExecutionFixture(t)
	var definitions strings.Builder
	definitions.WriteString("static const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&definitions, "%d,", b)
	}
	definitions.WriteString("};\n")
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&definitions, "#define ORD_%s %d\n", f.Name, i)
	}
	for _, compiler := range []string{"cl", "clang-cl"} {
		t.Run(compiler, func(t *testing.T) {
			dll := buildAggregateProbe(t, compiler)
			host := strings.Replace(aggregateExternalHost, "/* ARTIFACT */", definitions.String(), 1)
			host = strings.Replace(host, "/* DLL */", fmt.Sprintf("%q", filepath.ToSlash(dll)), 1)
			for _, verify := range []bool{false, true} {
				runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "independent_aggregate_abi.c", host, "-pedantic-errors", "-O2")
			}
		})
	}
}

func TestEVT2g5NearProbeBoundary(t *testing.T) {
	path := "../../language/evt2g5/valid/near_frame.concept"
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
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <windows.h>\n#include <stdio.h>\n#include <string.h>\n#include <stdint.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString(`};
typedef struct Near{double values[168];unsigned char tag;} Near;
static unsigned char code[65536];static concept_native_image image;
int main(void){concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={code,sizeof code};concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(r.tag){fprintf(stderr,"%s\n",F(describe_native_error)(r.payload.error.error));return 1;}
static concept_backend_function f;static concept_object_placement p;static concept_allocation a;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;if(F(plan_aggregate_function)(src,0,&f,&p,&a,&frame,&plan).tag||frame.byteCount!=4088)return 2;
unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);if(!memory)return 3;memcpy(memory,code,image.byteCount);DWORD old;if(!VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old))return 4;FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
typedef Near *(*Identity)(Near*,const Near*);Identity identity=NULL;void *entry=memory;memcpy(&identity,&entry,sizeof identity);
for(int run=0;run<100;run++){Near in;struct Guarded{uint64_t before;Near value;uint64_t after;} out;memset(&in,0xa5,sizeof in);memset(&out,0x5a,sizeof out);for(int i=0;i<168;i++)in.values[i]=(double)(run+i)/8.0;in.tag=(unsigned char)run;out.before=0x1122334455667788ULL;out.after=0x8877665544332211ULL;if(identity(&out.value,&in)!=&out.value)return 5;for(int i=0;i<168;i++)if(out.value.values[i]!=in.values[i])return 6;if(out.value.tag!=in.tag||out.before!=0x1122334455667788ULL||out.after!=0x8877665544332211ULL)return 7;for(size_t i=1345;i<sizeof(Near);i++)if(((unsigned char*)&out.value)[i]!=0x5a)return 8;}
printf("native maximum unprobed frame=%d code=%d; 100 sret, field and padding-isolation probes\n",frame.byteCount,image.byteCount);VirtualFree(memory,0,MEM_RELEASE);return 0;}
`)
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "near_probe_boundary.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g5MalformedPlans(t *testing.T) {
	_, _, machine, data := aggregateExecutionFixture(t)
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};\n")
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&c, "#define ORD_%s %d\n", f.Name, i)
	}
	c.WriteString(`static unsigned char bytes[65536];static concept_native_image image,before;static concept_backend_function function;static concept_object_placement placement;static concept_allocation allocation;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
int main(void){concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};concept_bridge_reader reader={src,0};concept_result_wire_object_module_aggregate_abierror decoded=F(admitted_aggregate_document)(src);if(decoded.tag)return 1;concept_wire_object_module doc=decoded.payload.ok.value;
`)
	cases := []struct{ name, owner, mutation, want string }{
		{"direct_width", "RoundFour", "plan.result.size=3;", "AMD64_AGGREGATE_ABI_INVALID_DIRECT_WIDTH"},
		{"native_return_eligibility", "RoundNative", "plan.result.category.tag=5;plan.result.carrier.tag=7;", "AMD64_AGGREGATE_ABI_RETURN_INELIGIBLE"},
		{"sret_identity", "Forward", "plan.hiddenPointer=-1;", "AMD64_AGGREGATE_ABI_INVALID_HIDDEN_RESULT"},
		{"sret_storage", "CallMixed", "plan.calls.data[0].resultStorage=0;", "AMD64_AGGREGATE_ABI_INVALID_HIDDEN_RESULT"},
		{"position_shift", "CallMixed", "plan.calls.data[0].arguments.data[0].physicalIndex=0;", "AMD64_AGGREGATE_ABI_INVALID_POSITION"},
		{"xmm_position", "CallMixed", "plan.calls.data[0].physical.scalar.destinations.data[3].xmm.tag=2;", "AMD64_AGGREGATE_ABI_INVALID_POSITION"},
		{"temporary_alias", "CallMixed", "plan.calls.data[0].arguments.data[1].storage=placement.values.data[plan.calls.data[0].arguments.data[1].value];", "AMD64_AGGREGATE_ABI_INVALID_TEMPORARY"},
		{"value_as_ref", "CallMixed", "plan.calls.data[0].arguments.data[1].type.category.tag=2;", "AMD64_AGGREGATE_ABI_STALE_LAYOUT"},
		{"ref_as_value", "CallMixed", "plan.calls.data[0].arguments.data[3].type.category.tag=4;", "AMD64_AGGREGATE_ABI_STALE_LAYOUT"},
		{"temporary_alignment", "CallMixed", "function.slots.data[placement.argumentTemps.data[0]].alignment=8;", "AMD64_AGGREGATE_ABI_TEMPORARY_ALIGNMENT"},
		{"temporary_size", "CallMixed", "function.slots.data[placement.argumentTemps.data[0]].size=8;", "AMD64_AGGREGATE_ABI_INVALID_TEMPORARY"},
		{"frame_overlap", "CallMixed", "frame.slots.data[frame.slotCount-1].offset=0;", "AMD64_AGGREGATE_ABI_INVALID_FRAME"},
		{"stale_layout", "RoundFour", "plan.parameters.data[0].type.layout=plan.parameters.data[0].type.identity;", "AMD64_AGGREGATE_ABI_STALE_LAYOUT"},
		{"call_result_carrier", "CallMixed", "plan.calls.data[0].physical.result.value=0;", "AMD64_AGGREGATE_ABI_INVALID_POSITION"},
		{"lifetime", "CallMixed", "plan.calls.data[0].arguments.data[1].end=0;", "AMD64_AGGREGATE_ABI_TEMPORARY_LIFETIME"},
		{"missing_copy", "CallMixed", "plan.transfers.data[0].type=-1;", "AMD64_AGGREGATE_ABI_TEMPORARY_LIFETIME"},
		{"capacity", "CallMixed", "plan.callCount=33;", "AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED"},
		{"stack_arguments", "CallMixed", "plan.calls.data[0].physical.stackArgumentBytes=0;", "AMD64_AGGREGATE_ABI_INVALID_POSITION"},
	}
	for _, tc := range cases {
		fmt.Fprintf(&c, "if(F(plan_aggregate_function)(src,ORD_%s,&function,&placement,&allocation,&frame,&plan).tag)return 2;{concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc,ORD_%s);if(owner.tag)return 3;%s memset(bytes,0xa5,sizeof bytes);memset(&image,0x5a,sizeof image);before=image;concept_byte_writer writer={dst,0,0};concept_encoding_plan branches=F(empty_encoding_plan)();concept_encoding_workspace workspace={{branches.blockOffsets.data,128},{branches.fixups.data,256},0,{image.fixups.data,512},0,ORD_%s,1};concept_result_void_native_error r=F(encode_checked_aggregate_function)(&writer,&reader,doc,owner.payload.ok.value,&function,&placement,&allocation,&frame,&workspace,&plan);if(!r.tag||strcmp(F(describe_native_error)(r.payload.error.error),%q)){fprintf(stderr,\"%s: %%s\\n\",r.tag?F(describe_native_error)(r.payload.error.error):\"ACCEPTED\");return 4;}if(writer.offset||memcmp(&image,&before,sizeof image))return 5;for(size_t j=0;j<sizeof bytes;j++)if(bytes[j]!=0xa5)return 6;}\n", tc.owner, tc.owner, tc.mutation, tc.owner, tc.want, tc.name)
	}
	fmt.Fprintf(&c, "puts(\"%d malformed derived plans rejected by production checked encoder before bytes/image mutation\");return 0;}\n", len(cases))
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "malformed_aggregate_plans.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g5MalformedArtifacts(t *testing.T) {
	_, _, machine, _ := aggregateExecutionFixture(t)
	base, _ := json.Marshal(machine.Borrow)
	ordinal := func(m *BorrowModule, name string) int {
		for i, f := range m.Objects.Functions {
			if f.Name == name {
				return i
			}
		}
		t.Fatal(name)
		return -1
	}
	type specimen struct {
		name, want string
		mutate     func(*BorrowModule)
	}
	cases := []specimen{
		{"aggregate_as_scalar", "CMIRAMD6_VALUE_TYPE", func(m *BorrowModule) { i := ordinal(m, "RoundFour"); m.Objects.Functions[i].Values[0].Class = "scalar" }},
		{"stale_value_layout", "CMIRAMD6_VALUE_TYPE", func(m *BorrowModule) {
			i := ordinal(m, "RoundFour")
			m.Objects.Functions[i].Values[0].LayoutIdentity = "stale"
		}},
		{"copy_ineligible", "CMIRAMD6_COPY_ELIGIBILITY", func(m *BorrowModule) { m.Objects.Types[1].Copy = "none" }},
		{"contradictory_geometry", "CMIRAMD6_LAYOUT_GEOMETRY", func(m *BorrowModule) { m.Objects.Types[1].Size = 0 }},
		{"wrong_call_type", "CMIRAMD6_REFERENCE_CALL_SIGNATURE", func(m *BorrowModule) {
			in := objectInstruction(t, &m.Objects, "TinyDriver", "call")
			in.Calls[0].Arguments[0].Type = m.Objects.Functions[ordinal(m, "RoundLarge")].ResultType
		}},
		{"wrong_call_target", "CMIRAMD6_REFERENCE_CALL_SIGNATURE", func(m *BorrowModule) {
			in := objectInstruction(t, &m.Objects, "CallMixed", "call")
			in.Calls[0].Target = "Absent|Absent()"
		}},
		{"wrong_result_type", "CMIRAMD6_REFERENCE_CALL_SIGNATURE", func(m *BorrowModule) {
			in := objectInstruction(t, &m.Objects, "CallMixed", "call")
			in.Calls[0].ResultType = m.Objects.Functions[ordinal(m, "RoundFour")].ResultType
		}},
		{"parameter_capacity", "CMIRAMD6_VALUE_TYPE", func(m *BorrowModule) {
			i := ordinal(m, "EightPositions")
			m.Objects.Functions[i].Params = append(m.Objects.Functions[i].Params, m.Objects.Functions[i].Params[0])
		}},
	}
	dir := t.TempDir()
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char input[262144],output[262144];static concept_native_image image,before;\nint main(void){FILE *file;size_t size;\n")
	for i, tc := range cases {
		var next BorrowModule
		if err := json.Unmarshal(base, &next); err != nil {
			t.Fatal(err)
		}
		tc.mutate(&next)
		if err := VerifyMachineIR(MachineModule{Borrow: &next}); err == nil {
			t.Fatalf("Go accepted %s", tc.name)
		}
		data, err := encodeBorrowMachineBridge(next)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, tc.name+".cmir")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&c, "file=fopen(%q,\"rb\");if(!file)return 1;size=fread(input,1,sizeof input,file);fclose(file);for(int run=0;run<100;run++){memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_readonly_span_byte src={input,size};concept_span_byte dst={output,sizeof output};concept_result_void_borrow_error checked=F(validate_borrow_machine_bridge)(src);if(!checked.tag||strcmp(F(describe_borrow_error)(checked.payload.error.error),%q)){fprintf(stderr,\"%s: %%s\\n\",checked.tag?F(describe_borrow_error)(checked.payload.error.error):\"ACCEPTED\");return %d;}concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(!r.tag||memcmp(&image,&before,sizeof image))return 20;for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 21;}\n", filepath.ToSlash(path), tc.want, tc.name, i+2)
	}
	fmt.Fprintf(&c, "puts(\"%d malformed aggregate artifacts rejected 100 times with precise admissions and unchanged output/image\");return 0;}\n", len(cases))
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "malformed_aggregate_artifacts.c", c.String(), "-pedantic-errors", "-O2")
	}
}

const aggregateExternalHost = `#include "amd64.generated.h"
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
/* ARTIFACT */
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(x) do{if(!(x)){fprintf(stderr,"line %d: %s\n",__LINE__,#x);return 1;}}while(0)
typedef struct Four {int value;} Four;typedef struct Large {double a,b,c;} Large;
typedef Large(*Shift)(int,double,int,float);typedef Large(*Mixed)(Four,Large,double,int*,float,const Four*);typedef int(*Isolation)(double);
typedef struct One{uint8_t value;} One;typedef struct Two{uint16_t value;} Two;typedef struct Eight{double value;} Eight;typedef int(*Tiny)(One,Two,Eight);
typedef int(*Callers)(void**,int);typedef int(*RawSret)(void*,int);typedef uint64_t(*Observe)(int);typedef void(*Forward)(void*);typedef int(*Saved)(void*);
#define PROC(t,n) t n=NULL;do{FARPROC p=GetProcAddress(dll,#n);REQUIRE(p);memcpy(&n,&p,sizeof n);}while(0)
#define GET(t,n) t n=NULL;do{void *p=memory+image.symbols.data[ORD_##n].offset;memcpy(&n,&p,sizeof n);}while(0)
static unsigned char bytes[262144];static concept_native_image image;
static void redirect(unsigned char *memory,int ordinal,FARPROC target){unsigned char *p=memory+image.symbols.data[ordinal].offset;p[0]=0x48;p[1]=0xb8;memcpy(p+2,&target,8);p[10]=0xff;p[11]=0xe0;}
int main(void){HMODULE dll=LoadLibraryA(/* DLL */);REQUIRE(dll);PROC(Callers,RunCallers);PROC(RawSret,RunRawSret);PROC(Observe,Observation);PROC(Forward,ForwardTo);PROC(Saved,ProbeSaved);
concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(r.tag){fprintf(stderr,"%s\n",F(describe_native_error)(r.payload.error.error));return 2;}
unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,bytes,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
void *entries[11]={memory+image.symbols.data[ORD_RoundOne].offset,memory+image.symbols.data[ORD_RoundTwo].offset,memory+image.symbols.data[ORD_RoundFour].offset,memory+image.symbols.data[ORD_RoundEight].offset,NULL,memory+image.symbols.data[ORD_Shifted].offset,memory+image.symbols.data[ORD_Mixed].offset,memory+image.symbols.data[ORD_EightPositions].offset,memory+image.symbols.data[ORD_SmallForward].offset,memory+image.symbols.data[ORD_Forward].offset,memory+image.symbols.data[ORD_At].offset};
ForwardTo(memory+image.symbols.data[ORD_RoundLarge].offset);FARPROC forwarded=GetProcAddress(dll,"ProbeRoundForward");REQUIRE(forwarded);memcpy(&entries[4],&forwarded,sizeof entries[4]);unsigned callerAlignments=0;
for(int run=0;run<100;run++){int checked=RunCallers(entries,run-50);if(checked){fprintf(stderr,"C caller case=%d seed=%d\n",checked,run-50);return 3;}REQUIRE(!RunRawSret(entries[5],run-50));REQUIRE(!ProbeSaved(memory+image.symbols.data[ORD_Chain].offset));uint64_t address=Observation(8);REQUIRE(address && !(address&7));callerAlignments|=1u<<(unsigned)(address&15);}
/* Test-only absolute tail thunks redirect admitted internal symbols into the
 * independently compiled DLL. There is no production external-linkage path,
 * alternate classifier, call encoder or frame policy in this harness. */
REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_READWRITE,&old));FARPROC mixed=GetProcAddress(dll,"ProbeMixedEntry"),shifted=GetProcAddress(dll,"ProbeShifted"),isolation=GetProcAddress(dll,"ProbeIsolation");REQUIRE(mixed&&shifted&&isolation);REQUIRE(image.symbols.data[ORD_Mixed].size>=12&&image.symbols.data[ORD_Shifted].size>=12&&image.symbols.data[ORD_ValueMutation].size>=12);redirect(memory,ORD_Mixed,mixed);redirect(memory,ORD_Shifted,shifted);redirect(memory,ORD_ValueMutation,isolation);REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
GET(Shift,CallShifted);GET(Mixed,CallMixed);GET(Isolation,Isolation);
REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_READWRITE,&old));FARPROC one=GetProcAddress(dll,"ProbeOne"),two=GetProcAddress(dll,"ProbeTwo"),eight=GetProcAddress(dll,"ProbeEight");REQUIRE(one&&two&&eight);redirect(memory,ORD_RoundOne,one);redirect(memory,ORD_RoundTwo,two);redirect(memory,ORD_RoundEight,eight);REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);GET(Tiny,TinyDriver);
for(int run=0;run<100;run++){REQUIRE(TinyDriver((One){(uint8_t)run},(Two){(uint16_t)(run*257)},(Eight){(double)run/8.0})==1);}
for(int run=0;run<100;run++){int seed=run-50;double x=(double)seed/8.0;Large value={x,x+1,x+2};Four small={seed},view={seed+2};int count=seed;Large out=CallMixed(small,value,0.5,&count,1.25f,&view);REQUIRE(out.a==x+0.5&&out.b==x+2.25&&out.c==x+2+seed&&count==3*seed+2);REQUIRE(value.a==x&&value.b==x+1&&value.c==x+2);REQUIRE(Observation(0)==8&&Observation(1)!=0&&Observation(3)!=0&&!(Observation(3)&15));REQUIRE((uint32_t)Observation(2)==(uint32_t)seed);double delta=0.5;uint64_t deltaBits;memcpy(&deltaBits,&delta,8);REQUIRE(Observation(4)==deltaBits);REQUIRE(Observation(5)==(uint64_t)(uintptr_t)&count);float bias=1.25f;uint32_t biasBits;memcpy(&biasBits,&bias,4);REQUIRE((uint32_t)Observation(6)==biasBits);REQUIRE(Observation(7)==(uint64_t)(uintptr_t)&view);REQUIRE(Isolation(x)==1&&Observation(13)!=Observation(14)&&!(Observation(13)&15));out=CallShifted(seed,x,seed+3,1.25f);REQUIRE(out.a==x&&out.b==seed&&out.c==seed+4.25);REQUIRE((uint32_t)Observation(11)==(uint32_t)seed&&(uint32_t)Observation(12)==(uint32_t)(seed+3));}
printf("100 independent C caller/callee ABI probes; native outgoing temp align=16; C incoming residue-mask=0x%x; RSP=8 mod16, shadow writable, GPR/XMM nonvolatiles preserved; raw RAX sret identity verified\n",callerAlignments);
VirtualFree(memory,0,MEM_RELEASE);FreeLibrary(dll);return 0;}
`

func TestEVT2g5NativeSmoke(t *testing.T) {
	machine, data := aggregateABIFixture(t)
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <windows.h>\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};\n")
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&c, "#define ORD_%s %d\n", f.Name, i)
	}
	c.WriteString(`
#ifndef G4_NATIVE_ONLY
#include "oracle.c"
#endif
#define REQUIRE(x) do{if(!(x)){fprintf(stderr,"line %d: %s\n",__LINE__,#x);return 1;}}while(0)
typedef struct Four { int value; } Four;
typedef struct Three { unsigned char a,b,c; } Three;
typedef struct Large { double a,b,c; } Large;
typedef struct Five{unsigned char values[5];} Five;typedef struct Six{unsigned char values[6];} Six;typedef struct Seven{unsigned char values[7];} Seven;
typedef struct Twelve{int a,b,c;} Twelve;typedef struct Sixteen{double a,b;} Sixteen;typedef struct ArrayEight{unsigned char values[8];} ArrayEight;
typedef Four(*RoundFour)(Four);
typedef Three(*RoundThree)(Three);
typedef Large(*RoundLarge)(Large);
typedef Large(*Shifted)(int,double,int,float);
typedef int(*Nullary)(void);
typedef Five(*FiveFn)(Five);typedef Six(*SixFn)(Six);typedef Seven(*SevenFn)(Seven);typedef Twelve(*TwelveFn)(Twelve);typedef Sixteen(*SixteenFn)(Sixteen);typedef ArrayEight(*ArrayFn)(ArrayEight);
static Five oracle_five(Five v){return v;}static Six oracle_six(Six v){return v;}static Seven oracle_seven(Seven v){return v;}static Twelve oracle_twelve(Twelve v){return v;}static Sixteen oracle_sixteen(Sixteen v){return v;}
#define GET(t,n) t n=NULL;do{void *entry=memory+image.symbols.data[ORD_##n].offset;memcpy(&n,&entry,sizeof n);}while(0)
static unsigned char bytes[262144];static concept_native_image image;
int main(void){concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};
concept_result_int_native_error emitted=F(emit_module)(src,dst,&image);if(emitted.tag){fprintf(stderr,"%s\n",F(describe_native_error)(emitted.payload.error.error));
concept_result_wire_object_module_aggregate_abierror doc=F(admitted_aggregate_document)(src);concept_bridge_reader reader={src,0};
for(int ordinal=0;ordinal<doc.payload.ok.value.Functions.count;ordinal++){static concept_backend_function f;static concept_object_placement p;static concept_allocation a;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
concept_result_void_aggregate_abierror planned=F(plan_aggregate_function)(src,ordinal,&f,&p,&a,&frame,&plan);if(planned.tag)continue;
concept_result_wire_object_function_aggregate_abierror owner=F(aggregate_plan_owner)(&reader,doc.payload.ok.value,ordinal);concept_byte_writer writer={dst,0,1};concept_encoding_plan branches=F(empty_encoding_plan)();concept_native_image scratch=F(empty_native_image)();concept_encoding_workspace workspace={{branches.blockOffsets.data,128},{branches.fixups.data,256},0,{scratch.fixups.data,512},0,ordinal,1};
concept_result_void_backend_error r=F(encode_object_function_plan)(&writer,&reader,doc.payload.ok.value,owner.payload.ok.value,&f,&p,&a,&frame,&workspace,1,&plan);if(r.tag)fprintf(stderr,"function %d %.*s offset=%d: %s\n",ordinal,f.name.length,(const char*)artifact+f.name.offset,writer.offset,F(describe_native_error)(F(scalar_backend_error)(r.payload.error.error)));}
return 2;}
unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,bytes,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
GET(RoundFour,RoundFour);Four four=RoundFour((Four){37});REQUIRE(four.value==37);
GET(RoundThree,RoundThree);Three three=RoundThree((Three){1,2,3});REQUIRE(three.a==1&&three.b==2&&three.c==3);
GET(RoundLarge,RoundTwentyFour);Large large=RoundTwentyFour((Large){2,3,4});REQUIRE(large.a==2&&large.b==3&&large.c==4);
GET(Shifted,Shifted);large=Shifted(1,7.0,2,3.0f);REQUIRE(large.a==1&&large.b==7&&large.c==3);
GET(Nullary,Driver);REQUIRE(Driver()==18);
GET(Nullary,Chain);REQUIRE(Chain()==9);
GET(Nullary,DirectChain);REQUIRE(DirectChain()==4);
GET(Nullary,StackArguments);REQUIRE(StackArguments()==3);
GET(Nullary,CallMixed);REQUIRE(CallMixed()==4);
GET(Nullary,Isolation);REQUIRE(Isolation()==7);
GET(FiveFn,RoundFive);GET(SixFn,RoundSix);GET(SevenFn,RoundSeven);GET(TwelveFn,RoundTwelve);GET(SixteenFn,RoundSixteen);GET(ArrayFn,RoundArray);
for(int run=0;run<100;run++){Five five;Six six;Seven seven;ArrayEight array;for(int i=0;i<5;i++)five.values[i]=(unsigned char)(run+i);for(int i=0;i<6;i++)six.values[i]=(unsigned char)(run+i+1);for(int i=0;i<7;i++)seven.values[i]=(unsigned char)(run+i+2);for(int i=0;i<8;i++)array.values[i]=(unsigned char)(run+i+3);Five f=RoundFive(five),fo=oracle_five(five);Six s=RoundSix(six),so=oracle_six(six);Seven v=RoundSeven(seven),vo=oracle_seven(seven);ArrayEight a=RoundArray(array);REQUIRE(!memcmp(f.values,fo.values,5)&&!memcmp(s.values,so.values,6)&&!memcmp(v.values,vo.values,7)&&!memcmp(a.values,array.values,8));Twelve t={run,run+1,run+2},tr=RoundTwelve(t),to=oracle_twelve(t);REQUIRE(tr.a==to.a&&tr.b==to.b&&tr.c==to.c);Sixteen d={(double)run/8.0,-(double)run/4.0},dr=RoundSixteen(d),dc=oracle_sixteen(d);REQUIRE(dr.a==dc.a&&dr.b==dc.b);}
printf("actual aggregate native image=%d functions=%d calls=%d\n",image.byteCount,image.symbolCount,image.fixupCount);
#ifndef G4_NATIVE_ONLY
for(int run=0;run<100;run++){Five f;Six s;Seven v;concept_five cf;concept_six cs;concept_seven cv;for(int i=0;i<5;i++)f.values[i]=(unsigned char)(run+i);for(int i=0;i<6;i++)s.values[i]=(unsigned char)(run+i);for(int i=0;i<7;i++)v.values[i]=(unsigned char)(run+i);memcpy(cf.values.data,f.values,5);memcpy(cs.values.data,s.values,6);memcpy(cv.values.data,v.values,7);Five rf=RoundFive(f);Six rs=RoundSix(s);Seven rv=RoundSeven(v);cf=concept_evt2g4__aggregate_abi_round_five(cf);cs=concept_evt2g4__aggregate_abi_round_six(cs);cv=concept_evt2g4__aggregate_abi_round_seven(cv);REQUIRE(!memcmp(rf.values,cf.values.data,5)&&!memcmp(rs.values,cs.values.data,6)&&!memcmp(rv.values,cv.values.data,7));}
#endif
VirtualFree(memory,0,MEM_RELEASE);return 0;}
`)
	for _, verify := range []bool{false, true} {
		outputs := backendTestOutputs(t, verify)
		runFoundationNativeHarnessOutput(t, outputs, "native_aggregate_smoke.c", c.String(), "-pedantic-errors", "-O2", "-DG4_NATIVE_ONLY=1")
		path := "../../language/evt2g4/valid/aggregate_abi.concept"
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
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
		runFoundationNativeHarnessOutput(t, outputs, "native_aggregate_odd_c_parity.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func aggregateExecutionFixture(t *testing.T) (Module, []byte, MachineModule, []byte) {
	t.Helper()
	path := "../../language/evt2g5/valid/calls.concept"
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
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if machine.Borrow == nil {
		t.Fatal("expected checked reference/aggregate CMIRAMD6")
	}
	return m, source, machine, data
}

func TestEVT2g5FreshCheckedInput(t *testing.T) {
	_, _, base, want := aggregateExecutionFixture(t)
	for run := 0; run < 100; run++ {
		_, _, next, data := aggregateExecutionFixture(t)
		if !bytes.Equal(data, want) || next.String() != base.String() {
			t.Fatalf("fresh input changed at %d", run)
		}
	}
	t.Logf("100 fresh parses/checks/LIR/MachineIR generations; functions=%d artifact=%d sha256=%x", len(base.Borrow.Objects.Functions), len(want), sha256.Sum256(want))
}

func TestEVT2g5NativeExecution(t *testing.T) {
	m, source, machine, data := aggregateExecutionFixture(t)
	dir := t.TempDir()
	var definitions strings.Builder
	definitions.WriteString("static const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&definitions, "%d,", b)
	}
	definitions.WriteString("};\n")
	for i, f := range machine.Borrow.Objects.Functions {
		fmt.Fprintf(&definitions, "#define ORD_%s %d\n", f.Name, i)
	}
	var normal []byte
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		policy := ConservativeCompilationPolicy()
		if verify {
			mode = "Verify"
			policy = VerifyCompilationPolicy()
		}
		outputs := backendTestOutputs(t, verify)
		dump := filepath.Join(dir, mode+".bin")
		host := strings.Replace(aggregateExecutionHost, "/* ARTIFACT */", definitions.String(), 1)
		host = strings.Replace(host, "/* IMAGE_DUMP */", fmt.Sprintf("%q", filepath.ToSlash(dump)), 1)
		runFoundationNativeHarnessOutput(t, outputs, "aggregate_artifact_only.c", host, "-pedantic-errors", "-O2", "-DG5_NATIVE_ONLY=1")
		image, err := os.ReadFile(dump)
		if err != nil {
			t.Fatal(err)
		}
		if !verify {
			normal = image
		} else if !bytes.Equal(normal, image) {
			t.Fatal("Normal/Verify native image mismatch")
		}
		t.Logf("%s native bytes=%d sha256=%x", mode, len(image), sha256.Sum256(image))
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
		runFoundationNativeHarnessOutput(t, outputs, "aggregate_c_parity.c", host, "-pedantic-errors", "-O2")
	}
}

const aggregateExecutionHost = `#include "amd64.generated.h"
#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <time.h>
#ifdef G5_NATIVE_ONLY
typedef struct One {uint8_t value;} One;
typedef struct Two {uint16_t value;} Two;
typedef struct Four {int32_t value;} Four;
typedef struct Eight {double value;} Eight;
typedef struct Three {uint8_t a,b,c;} Three;
typedef struct PaddedEight {uint8_t tag;int32_t value;} PaddedEight;
typedef struct PaddedSixteen {uint8_t tag;uint64_t value;} PaddedSixteen;
typedef struct Large {double a,b,c;} Large;
typedef struct ArrayThree {struct {int32_t data[3];} values;} ArrayThree;
typedef struct Nested {PaddedEight head;ArrayThree tail;} Nested;
#else
#include "oracle.c"
typedef concept_one One;typedef concept_two Two;typedef concept_four Four;typedef concept_eight Eight;typedef concept_three Three;
typedef concept_padded_eight PaddedEight;typedef concept_padded_sixteen PaddedSixteen;typedef concept_large Large;typedef concept_array_three ArrayThree;typedef concept_nested Nested;
#endif
/* ARTIFACT */
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(x) do{if(!(x)){fprintf(stderr,"line %d: %s\n",__LINE__,#x);return 1;}}while(0)
#define GET(t,n) t n=NULL;do{void *entry=memory+image.symbols.data[ORD_##n].offset;memcpy(&n,&entry,sizeof n);}while(0)
typedef One(*OneFn)(One);typedef Two(*TwoFn)(Two);typedef Four(*FourFn)(Four);typedef Eight(*EightFn)(Eight);typedef Three(*ThreeFn)(Three);
typedef PaddedEight(*PadFn)(PaddedEight);typedef PaddedSixteen(*Pad16Fn)(PaddedSixteen);typedef Large(*LargeFn)(Large);typedef Nested(*NestedFn)(Nested);
typedef Large(*ShiftFn)(int,double,int,float);typedef Large(*MixedFn)(Four,Large,double,int*,float,const Four*);typedef Large(*PairFn)(Large,Large);
typedef Four(*EightFnArgs)(Four,Four,Four,Four,Four,Four,Four,Four);typedef Four(*SmallRec)(int,Four);typedef Large(*LargeRec)(int,Large);
typedef int(*AtFn)(ArrayThree,int);typedef Nested(*EditFn)(Nested,int);typedef int(*IntFn)(int);typedef double(*DoubleFn)(double);typedef int(*IsolationFn)(double);
typedef float(*FloatFn)(float,Large);typedef double(*ScenarioFn)(double,int);typedef int(*TinyFn)(One,Two,Eight);typedef uint64_t(*RawPad)(uint64_t);
static unsigned char bytes[262144],again[262144];static concept_native_image image,other;
static ArrayThree make_array(int a,int b,int c){ArrayThree v;v.values.data[0]=a;v.values.data[1]=b;v.values.data[2]=c;return v;}
static Nested make_nested(int seed){Nested v;v.head.tag=9;v.head.value=seed;v.tail=make_array(seed,seed+1,seed+2);return v;}
static int eq(double a,double b){return memcmp(&a,&b,sizeof a)==0;}
static int large_eq(Large a,Large b){return eq(a.a,b.a)&&eq(a.b,b.b)&&eq(a.c,b.c);}
static int plan_text(concept_readonly_span_byte src,int ordinal,char *text){
static concept_backend_function f;static concept_object_placement p;static concept_allocation a;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;
if(F(plan_aggregate_function)(src,ordinal,&f,&p,&a,&frame,&plan).tag)return 0;int n=snprintf(text,65536,"frame=%d slots=%d regs=%d calls=%d",frame.byteCount,frame.slotCount,f.virtualRegCount,plan.callCount);
for(int i=0;i<frame.slotCount;i++){concept_frame_slot s=frame.slots.data[i];n+=snprintf(text+n,65536-(size_t)n," s%d:%d:%d:%d:%d",s.category.tag,s.identity,s.offset,s.size,s.alignment);}
for(int i=0;i<f.virtualRegCount;i++){concept_scalar_location x=a.scalarLocations.data[i];n+=snprintf(text+n,65536-(size_t)n," v%d:%d:%d:%d:%d:%d",i,x.kind.tag,x.gpr.tag,x.xmm.tag,a.start.data[i],a.end.data[i]);}
for(int i=0;i<plan.callCount;i++){concept_aggregate_call_plan c=plan.calls.data[i];n+=snprintf(text+n,65536-(size_t)n," c%d:%d:%d:%d:%d",i,c.position,c.result.category.tag,c.resultStorage,c.hiddenPointer);for(int j=0;j<c.physical.moveCount;j++){concept_scalar_move x=c.physical.scalar.moves.data[j];n+=snprintf(text+n,65536-(size_t)n," m%d:%d:%d:%d:%d:%d:%d",x.source.kind.tag,x.source.gpr.tag,x.source.slot,x.destination.kind.tag,x.destination.gpr.tag,x.destination.slot,x.representation.tag);}}
return n>0 && n<65536;}
int main(int argc,char **argv){concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={bytes,sizeof bytes};clock_t begin=clock();
concept_result_int_native_error emitted=F(emit_module)(src,dst,&image);if(emitted.tag){fprintf(stderr,"%s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}
printf("planning plus preflight plus emission ms=%.3f image=%d functions=%d calls=%d\n",1000.0*(double)(clock()-begin)/CLOCKS_PER_SEC,image.byteCount,image.symbolCount,image.fixupCount);
FILE *dump=fopen(/* IMAGE_DUMP */,"wb");REQUIRE(dump);REQUIRE(fwrite(bytes,1,image.byteCount,dump)==(size_t)image.byteCount);fclose(dump);
unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,bytes,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
GET(OneFn,RoundOne);GET(TwoFn,RoundTwo);GET(FourFn,RoundFour);GET(EightFn,RoundEight);GET(ThreeFn,RoundThree);GET(PadFn,RoundPaddedEight);GET(Pad16Fn,RoundPaddedSixteen);GET(LargeFn,RoundLarge);GET(NestedFn,RoundNested);
GET(ShiftFn,Shifted);GET(MixedFn,Mixed);GET(PairFn,PairValues);GET(EightFnArgs,EightPositions);GET(SmallRec,SmallForward);GET(LargeRec,Forward);GET(AtFn,At);GET(EditFn,Edit);GET(IntFn,NativeDriver);GET(IntFn,DirectDriver);GET(DoubleFn,Chain);GET(DoubleFn,Live);GET(FloatFn,FloatLive);GET(ScenarioFn,Scenario);GET(IsolationFn,Isolation);GET(DoubleFn,Pressure);GET(TinyFn,TinyDriver);GET(IntFn,BoolDriver);
if(argc>1){SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX);At(make_array(1,2,3),argv[1][0]=='-'?-1:3);return 99;}
for(int run=0;run<100;run++){int seed=run-50;double x=(double)seed/8.0;float f=(float)seed/4.0f;Large large={x,x+1,x+2};Four four={seed};One one={(uint8_t)run};Two two={(uint16_t)(run*257)};Three three={(uint8_t)run,2,3};PaddedEight pad={7,seed};PaddedSixteen pad16={3,0xfedcba9876543200ULL+(uint64_t)run};Nested nested=make_nested(seed);
REQUIRE(RoundOne(one).value==one.value && RoundTwo(two).value==two.value && RoundFour(four).value==seed);REQUIRE(eq(RoundEight((Eight){x}).value,x));Three r3=RoundThree(three);REQUIRE(r3.a==three.a&&r3.b==2&&r3.c==3);PaddedEight rp=RoundPaddedEight(pad);REQUIRE(rp.tag==7&&rp.value==seed);PaddedSixteen rp16=RoundPaddedSixteen(pad16);REQUIRE(rp16.tag==3&&rp16.value==pad16.value);REQUIRE(large_eq(RoundLarge(large),large));Nested rn=RoundNested(nested);REQUIRE(rn.head.value==seed&&rn.tail.values.data[2]==seed+2);
REQUIRE(NativeDriver(seed)==seed && DirectDriver(seed)==seed*8 && Isolation(x)==1);REQUIRE(eq(Pressure(x),10*x+33));REQUIRE(TinyDriver(one,two,(Eight){x})==1);REQUIRE(BoolDriver(seed)==(seed>0?seed+5:-1));
REQUIRE(EightPositions(four,four,four,four,four,four,four,four).value==seed*8);REQUIRE(SmallForward(run%6,four).value==seed+run%6);Large recursive=Forward(run%6,large);REQUIRE(recursive.a==x+run%6&&recursive.b==large.b&&recursive.c==large.c);
Large shifted=Shifted(seed,x,0,f);Large expected={x,seed>0?1.0:f>0?3.0:5.0,seed>0?2.0:f>0?4.0:6.0};REQUIRE(large_eq(shifted,expected));int count=seed;Four view={4};Large mixed=Mixed((Four){3},large,0.5,&count,f,&view);REQUIRE(count==seed+7);expected=large;if(f>0){expected.a+=0.5;expected.c+=1;}else expected.b+=0.5;REQUIRE(large_eq(mixed,expected));REQUIRE(large_eq(PairValues(large,large),(Large){x*2,(x+1)*2,(x+2)*2}));
REQUIRE(At(make_array(seed,seed+1,seed+2),run%3)==seed+run%3);Nested edited=Edit(nested,seed+7);REQUIRE(edited.head.tag==9&&edited.head.value==seed+7&&edited.tail.values.data[1]==seed+1&&nested.head.value==seed);
REQUIRE(eq(Chain(x),3*x+6));REQUIRE(eq(Live(x),10*x+36));REQUIRE(eq(Scenario(x,seed),3*x+4.5));REQUIRE(FloatLive(f,large)==f*2);
#ifndef G5_NATIVE_ONLY
REQUIRE(RoundOne(one).value==concept_evt2g5__calls_round_one(one).value);REQUIRE(RoundTwo(two).value==concept_evt2g5__calls_round_two(two).value);REQUIRE(RoundFour(four).value==concept_evt2g5__calls_round_four(four).value);REQUIRE(eq(RoundEight((Eight){x}).value,concept_evt2g5__calls_round_eight((Eight){x}).value));REQUIRE(large_eq(RoundLarge(large),concept_evt2g5__calls_round_large(large)));REQUIRE(large_eq(Forward(run%6,large),concept_evt2g5__calls_forward(run%6,large)));REQUIRE(SmallForward(run%6,four).value==concept_evt2g5__calls_small_forward(run%6,four).value);REQUIRE(EightPositions(four,four,four,four,four,four,four,four).value==concept_evt2g5__calls_eight_positions(four,four,four,four,four,four,four,four).value);
int oracleCount=seed;Large om=concept_evt2g5__calls_mixed((Four){3},large,0.5,&oracleCount,f,&view);REQUIRE(count==oracleCount&&large_eq(mixed,om));REQUIRE(large_eq(shifted,concept_evt2g5__calls_shifted(seed,x,0,f)));REQUIRE(eq(Chain(x),concept_evt2g5__calls_chain(x)));REQUIRE(eq(Live(x),concept_evt2g5__calls_live(x)));REQUIRE(eq(Scenario(x,seed),concept_evt2g5__calls_scenario(x,seed)));REQUIRE(Isolation(x)==concept_evt2g5__calls_isolation(x));REQUIRE(NativeDriver(seed)==concept_evt2g5__calls_native_driver(seed));REQUIRE(FloatLive(f,large)==concept_evt2g5__calls_float_live(f,large));REQUIRE(eq(Pressure(x),concept_evt2g5__calls_pressure(x)));REQUIRE(TinyDriver(one,two,(Eight){x})==concept_evt2g5__calls_tiny_driver(one,two,(Eight){x}));REQUIRE(BoolDriver(seed)==concept_evt2g5__calls_bool_driver(seed));Nested oe=concept_evt2g5__calls_edit(nested,seed+7);REQUIRE(edited.head.tag==oe.head.tag&&edited.head.value==oe.head.value&&edited.tail.values.data[2]==oe.tail.values.data[2]);
#endif
}
RawPad rawPad=NULL;void *padEntry=memory+image.symbols.data[ORD_RoundPaddedEight].offset;memcpy(&rawPad,&padEntry,sizeof rawPad);REQUIRE(rawPad(0x11223344a5a5a57fULL)==0x112233440000007fULL);
const uint64_t bits[]={0x8000000000000000ULL,0x7ff0000000000000ULL,0xfff0000000000000ULL,0x7ff8123456789abcULL};for(size_t i=0;i<sizeof bits/sizeof bits[0];i++){double x;memcpy(&x,&bits[i],8);Eight returned=RoundEight((Eight){x});REQUIRE(!memcmp(&returned.value,&bits[i],8));}
static char before[65536],after[65536];REQUIRE(plan_text(src,ORD_Pressure,before));
for(int run=0;run<100;run++){concept_span_byte out={again,sizeof again};concept_result_int_native_error r=F(emit_module)(src,out,&other);REQUIRE(!r.tag&&other.finalized&&other.byteCount==image.byteCount&&other.symbolCount==image.symbolCount&&other.fixupCount==image.fixupCount&&!memcmp(bytes,again,image.byteCount));for(int i=0;i<image.symbolCount;i++)REQUIRE(other.symbols.data[i].ordinal==image.symbols.data[i].ordinal&&other.symbols.data[i].offset==image.symbols.data[i].offset&&other.symbols.data[i].size==image.symbols.data[i].size&&other.symbols.data[i].identity.offset==image.symbols.data[i].identity.offset&&other.symbols.data[i].identity.length==image.symbols.data[i].identity.length);for(int i=0;i<image.fixupCount;i++)REQUIRE(other.fixups.data[i].displacement==image.fixups.data[i].displacement&&other.fixups.data[i].callOffset==image.fixups.data[i].callOffset);REQUIRE(plan_text(src,ORD_Pressure,after)&&!strcmp(before,after));}
int minimum=4096,maximum=0;for(int i=0;i<image.symbolCount;i++){static concept_backend_function fn;static concept_object_placement p;static concept_allocation a;static concept_finalized_frame frame;static concept_aggregate_function_plan plan;REQUIRE(!F(plan_aggregate_function)(src,i,&fn,&p,&a,&frame,&plan).tag);if(frame.byteCount<minimum)minimum=frame.byteCount;if(frame.byteCount>maximum)maximum=frame.byteCount;printf("frame %.*s bytes=%d spill-bytes=%d code=%d\n",fn.name.length,(const char*)artifact+fn.name.offset,frame.byteCount,frame.regions.data[4].size,image.symbols.data[i].size);}
for(int bad=0;bad<2;bad++){char path[MAX_PATH],command[MAX_PATH+32];REQUIRE(GetModuleFileNameA(NULL,path,MAX_PATH));snprintf(command,sizeof command,"\"%s\" %s",path,bad?"-1":"3");STARTUPINFOA startup;PROCESS_INFORMATION process;memset(&startup,0,sizeof startup);startup.cb=sizeof startup;REQUIRE(CreateProcessA(NULL,command,NULL,NULL,FALSE,CREATE_NO_WINDOW,NULL,NULL,&startup,&process));DWORD waited=WaitForSingleObject(process.hProcess,30000),status=0;if(waited!=WAIT_OBJECT_0){TerminateProcess(process.hProcess,99);WaitForSingleObject(process.hProcess,1000);}REQUIRE(GetExitCodeProcess(process.hProcess,&status)&&waited==WAIT_OBJECT_0&&status==0xc000001dUL);CloseHandle(process.hThread);CloseHandle(process.hProcess);}
VirtualFree(memory,0,MEM_RELEASE);printf("100 varied native aggregate vectors and 100 fresh native emissions/plans; frames=%d..%d\n",minimum,maximum);return 0;}
`
