package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func enumFixture(t *testing.T) (Module, []byte, MachineModule, []byte) {
	t.Helper()
	path := "../../language/evt2h1/valid/enums.concept"
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
	if machine.Enum == nil {
		t.Fatal("missing enum contract")
	}
	return m, source, machine, data
}
func TestEVT2h1Transport(t *testing.T) {
	start := time.Now()
	_, _, machine, data := enumFixture(t)
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "88032d2679c4730efcd7dc91130c21ebf6032bdc9c8e55d2d19d9a8798e5d9d3" {
		t.Fatalf("exact CMIRAMD7 specimen changed: %s", got)
	}
	baseModule, _, _, _ := enumFixture(t)
	lir, err := GenerateLIR(baseModule)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeMachineBridge(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bridgeSemanticEqual(reflect.ValueOf(*machine.Enum), reflect.ValueOf(*decoded.Enum)) {
		t.Fatal("roundtrip semantic mismatch")
	}
	for run := 0; run < 100; run++ {
		m, _, again, artifact := enumFixture(t)
		againLIR, err := GenerateLIR(m)
		if err != nil || againLIR.String() != lir.String() {
			t.Fatalf("LIR generation %d: %v", run, err)
		}
		if !bytes.Equal(data, artifact) || again.String() != machine.String() {
			t.Fatalf("generation %d differs", run)
		}
	}
	path := "../../libraries/Standard/Backend/BridgeSchema7.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateMachineBridgeCodecs(m)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{
		"internal/concept/machineir_bridge7_codec_generated.go": generated.Go,
		"libraries/Standard/Backend/BridgeCodec7.concept":       generated.Concept,
		"docs/design/EVT2-MACHINEIR-BRIDGE7.schema.json":        generated.Metadata,
	} {
		actual, err := os.ReadFile(filepath.Join("../..", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), want) {
			t.Fatalf("stale %s", path)
		}
	}
	for run := 0; run < 100; run++ {
		again, err := GenerateMachineBridgeCodecs(m)
		if err != nil || !bytes.Equal(again.Go, generated.Go) || !bytes.Equal(again.Concept, generated.Concept) || !bytes.Equal(again.Metadata, generated.Metadata) {
			t.Fatalf("codec generation %d: %v", run, err)
		}
	}
	t.Logf("100 fresh generations and codecs; %d bytes sha256=%x schema=%s elapsed=%s", len(data), sha256.Sum256(data), generated.Hash, time.Since(start))
}
func TestEVT2h1NativeExecution(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64")
	}
	m, source, machine, data := enumFixture(t)
	var defs strings.Builder
	defs.WriteString("static const unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&defs, "%d,", b)
	}
	defs.WriteString("};\n")
	for i, f := range machine.Enum.Objects.Functions {
		fmt.Fprintf(&defs, "#define ORD_%s %d\n", f.Name, i)
	}
	dump := filepath.ToSlash(filepath.Join(t.TempDir(), "native.bin"))
	host := strings.Replace(enumNativeHost, "/* ARTIFACT */", defs.String(), 1)
	host = strings.Replace(host, "/* DUMP */", fmt.Sprintf("%q", dump), 1)
	var first []byte
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			outputs := backendTestOutputs(t, verify)
			runFoundationNativeHarnessOutput(t, outputs, "enum_artifact.c", host, "-pedantic-errors", "-O2", "-DENUM_NATIVE_ONLY=1")
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
			runFoundationNativeHarnessOutput(t, outputs, "enum_oracle.c", host, "-pedantic-errors", "-O2")
			actual, err := os.ReadFile(filepath.FromSlash(dump))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(actual)); got != "76425ddc81825b0897c1efdeb515e502ca416e76b8090dc6c37c7a7b97faaf77" {
				t.Fatalf("exact native specimen changed: %s", got)
			}
			if first == nil {
				first = actual
			} else if !bytes.Equal(first, actual) {
				t.Fatal("Normal/Verify bytes differ")
			}
			t.Logf("native bytes=%d sha256=%x", len(actual), sha256.Sum256(actual))
		})
	}
}

func TestEVT2h1Admission(t *testing.T) {
	_, _, machine, valid := enumFixture(t)
	type specimen struct {
		name string
		data []byte
	}
	var cases []specimen
	add := func(name string, change func(*EnumModule)) {
		raw, _ := json.Marshal(machine.Enum)
		var next EnumModule
		if err := json.Unmarshal(raw, &next); err != nil {
			t.Fatal(err)
		}
		change(&next)
		data, err := encodeEnumMachineBridge(next)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeMachineBridge(data); err == nil {
			t.Fatalf("Go admitted %s", name)
		}
		cases = append(cases, specimen{name, data})
	}
	add("missing_domains", func(m *EnumModule) { m.Domains = nil })
	add("duplicate_domain", func(m *EnumModule) { m.Domains = append(m.Domains, m.Domains[0]) })
	add("unknown_type", func(m *EnumModule) { m.Domains[0].Type = 999 })
	add("wrong_geometry", func(m *EnumModule) { m.Domains[0].Size = 8 })
	add("wrong_carrier", func(m *EnumModule) { m.Objects.Types[m.Domains[0].Type].Kind = "i32" })
	add("wrong_identity", func(m *EnumModule) { m.Domains[0].Identity = "different" })
	add("wrong_layout", func(m *EnumModule) { m.Domains[0].LayoutIdentity = "enum:" + strings.Repeat("0", 64) })
	add("duplicate_tag", func(m *EnumModule) { m.Domains[0].Variants[1].Tag = 0 })
	add("duplicate_name", func(m *EnumModule) { m.Domains[0].Variants[1].Name = m.Domains[0].Variants[0].Name })
	add("empty_variants", func(m *EnumModule) { m.Domains[0].Variants = nil })
	instruction := func(m *EnumModule, name, op string) *ObjectInstruction {
		for fi := range m.Objects.Functions {
			f := &m.Objects.Functions[fi]
			if f.Name == name {
				for bi := range f.Blocks {
					for ii := range f.Blocks[bi].Instructions {
						in := &f.Blocks[bi].Instructions[ii]
						if in.Op == op {
							return in
						}
					}
				}
			}
		}
		t.Fatalf("missing %s %s", name, op)
		return nil
	}
	add("out_of_domain", func(m *EnumModule) { instruction(m, "Ready", "const").Literal = "3" })
	add("negative_tag", func(m *EnumModule) { instruction(m, "Ready", "const").Literal = "-1" })
	add("noncanonical_tag", func(m *EnumModule) { instruction(m, "Ready", "const").Literal = "00" })
	add("enum_ordering", func(m *EnumModule) { instruction(m, "Next", "eq").Op = "lt" })
	add("nominal_call", func(m *EnumModule) { instruction(m, "Choose", "call").Calls[0].Arguments[0].Type = m.Domains[1].Type })
	add("nominal_return", func(m *EnumModule) { m.Objects.Functions[0].ResultType = m.Domains[1].Type })
	add("stale_value_layout", func(m *EnumModule) { m.Objects.Functions[0].Values[0].LayoutIdentity = "u32" })
	for _, offset := range []int{0, 8, 12} {
		data := bytes.Clone(valid)
		data[offset] ^= 1
		cases = append(cases, specimen{fmt.Sprintf("header_%d", offset), data})
	}
	cases = append(cases, specimen{"truncated", valid[:len(valid)-1]}, specimen{"trailing", append(bytes.Clone(valid), 0)})
	legacy, err := encodeObjectMachineBridge(machine.Enum.Objects)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeMachineBridge(legacy); err == nil {
		t.Fatal("v5 silently acquired enum aliases")
	}
	cases = append(cases, specimen{"frozen_v5_alias", legacy})
	dir := t.TempDir()
	var host strings.Builder
	host.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char input[262144],output[262144];static concept_native_image image,before;int main(void){FILE *f;size_t n;concept_readonly_span_byte src;concept_span_byte dst={output,sizeof output};\n")
	for i, c := range cases {
		p := filepath.Join(dir, c.name+".cmir")
		if err := os.WriteFile(p, c.data, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&host, "f=fopen(%q,\"rb\");if(!f)return 1;n=fread(input,1,sizeof input,f);fclose(f);src=(concept_readonly_span_byte){input,n};for(int run=0;run<100;run++){memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_result_int_enum_error rt=F(validated_enum_bridge_round_trip)(src,dst);if(!rt.tag){fprintf(stderr,\"accepted %s\\n\");return %d;}concept_result_int_native_error r=F(emit_module)(src,dst,&image);if(!r.tag||memcmp(&image,&before,sizeof image))return 90;for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 91;}\n", filepath.ToSlash(p), c.name, i+2)
	}
	fmt.Fprintf(&host, "puts(\"%d malformed enum artifacts rejected 100 times with unchanged output/image\");return 0;}\n", len(cases))
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "enum_admission.c", host.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2h1SourceBoundaries(t *testing.T) {
	paths, err := filepath.Glob("../../language/evt2h1/invalid/*.concept")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 12 {
		t.Fatalf("invalid fixture count %d", len(paths))
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(path, string(source))
		codes := map[string]string{"duplicate_variant": "CV4100", "enum_conversion": "NUMERIC_CAST_REQUIRED", "explicit_discriminant": "CV4004", "extern_abi": "EXTERN_C_ABI_TYPE_INVALID", "integer_conversion": "NUMERIC_CAST_REQUIRED", "nominal_assignment": "CV4107", "nominal_call": "CV4107", "nominal_equality": "CV4028", "ordering": "CV4028", "payload_equality": "CV4830", "unknown_variant": "CV4103", "wrong_arity": "CV4104"}
		want := codes[strings.TrimSuffix(filepath.Base(path), ".concept")]
		if err == nil || want == "" || !strings.Contains(err.Error(), want) {
			t.Fatalf("accepted invalid %s", path)
		}
		t.Logf("%s: %v", filepath.Base(path), err)
	}
	for name, want := range map[string]string{"payload": "EVT2_UNSUPPORTED_PAYLOAD_ENUM", "aggregate": "LIR_UNSUPPORTED_ENUM_CONTAINER", "reference": "LIR_UNSUPPORTED_ENUM_CONTAINER"} {
		path := "../../language/evt2h1/unsupported/" + name + ".concept"
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Generate(m, source); err != nil {
			t.Fatalf("source C bootstrap %s: %v", name, err)
		}
		if _, err = GenerateMachineIR(m); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("native boundary %s: %v", name, err)
		}
	}
}

func TestEVT2h1NativeCapacity(t *testing.T) {
	path := "../../language/evt2h1/unsupported/nine_arguments.concept"
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
	c.WriteString("#include \"amd64.generated.h\"\n#include <string.h>\n#define F(n) concept_standard__backend__amd64_##n\nstatic unsigned char artifact[]={")
	for _, b := range data {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString("};static unsigned char output[8192];static concept_native_image image,before;int main(void){memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;concept_readonly_span_byte src={artifact,sizeof artifact};if(F(validate_enum_machine_bridge)(src).tag)return 1;concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){output,sizeof output},&image);if(!r.tag||strcmp(F(describe_native_error)(r.payload.error.error),\"AMD64_SCALAR_PLANNING_FAILURE\"))return 2;if(memcmp(&image,&before,sizeof image))return 3;for(size_t i=0;i<sizeof output;i++)if(output[i]!=0xa5)return 4;return 0;}\n")
	for _, verify := range []bool{false, true} {
		runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "enum_capacity.c", c.String(), "-pedantic-errors", "-O2")
	}
}

const enumNativeHost = `#include "amd64.generated.h"
#ifndef ENUM_NATIVE_ONLY
#include "oracle.c"
#endif
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#define F(n) concept_standard__backend__amd64_##n
/* ARTIFACT */
static unsigned char code[262144],again[262144],roundtrip[262144];
static concept_native_image image,other;
#define REQUIRE(c) do{if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;}}while(0)
typedef uint32_t(*Nullary)(void);
typedef uint32_t(*Unary)(uint32_t);
typedef uint32_t(*Loop)(int,uint32_t);
typedef uint32_t(*Choose)(_Bool,uint32_t,uint32_t);
typedef uint32_t(*Eight)(uint32_t,uint32_t,uint32_t,uint32_t,uint32_t,uint32_t,uint32_t,uint32_t);
typedef uint32_t(*Mixed)(int,uint32_t,double,uint32_t,uint32_t,float,_Bool,uint32_t);
typedef int(*Scenario)(int);
#define GETTER(t) static t Get##t(unsigned char *memory,int ordinal){t f=NULL;void *p=memory+image.symbols.data[ordinal].offset;memcpy(&f,&p,sizeof f);return f;}
GETTER(Nullary) GETTER(Unary) GETTER(Loop) GETTER(Choose) GETTER(Eight) GETTER(Mixed) GETTER(Scenario)
#define FN(n,t) Get##t(memory,ORD_##n)
static int callback_count;
static uint32_t foreign_identity(uint32_t value){callback_count++;return value;}
int main(void){
 concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={code,sizeof code};
 concept_result_void_enum_error admitted=F(validate_enum_machine_bridge)(src);
 if(admitted.tag){fprintf(stderr,"admission: %s\n",F(describe_enum_error)(admitted.payload.error.error));return 1;}
 clock_t start=clock();
 concept_result_int_native_error emitted=F(emit_module)(src,dst,&image);
 if(emitted.tag){fprintf(stderr,"emission: %s\n",F(describe_native_error)(emitted.payload.error.error));return 2;}
 REQUIRE(image.finalized);
 printf("enum emission %.3f ms, %d functions, %d bytes\n",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC,image.symbolCount,image.byteCount);
 for(int run=0;run<100;run++){
  concept_result_int_enum_error rt=F(validated_enum_bridge_round_trip)(src,(concept_span_byte){roundtrip,sizeof roundtrip});REQUIRE(!rt.tag&&rt.payload.ok.value==sizeof artifact&&!memcmp(roundtrip,artifact,sizeof artifact));
  concept_result_int_native_error r=F(emit_module)(src,(concept_span_byte){again,sizeof again},&other);REQUIRE(!r.tag&&other.finalized&&other.byteCount==image.byteCount&&other.symbolCount==image.symbolCount&&other.fixupCount==image.fixupCount&&!memcmp(code,again,image.byteCount));
  for(int s=0;s<image.symbolCount;s++){REQUIRE(image.symbols.data[s].offset==other.symbols.data[s].offset&&image.symbols.data[s].size==other.symbols.data[s].size&&image.symbols.data[s].calls==other.symbols.data[s].calls);}
 }
 FILE *dump=fopen(/* DUMP */,"wb");REQUIRE(dump);REQUIRE(fwrite(code,1,image.byteCount,dump)==(size_t)image.byteCount);fclose(dump);
 unsigned char *memory=VirtualAlloc(NULL,image.byteCount,MEM_COMMIT|MEM_RESERVE,PAGE_READWRITE);REQUIRE(memory);memcpy(memory,code,image.byteCount);DWORD old;REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 REQUIRE(FN(Ready,Nullary)()==0&&FN(Active,Nullary)()==1&&FN(Done,Nullary)()==2);
 REQUIRE(FN(Red,Nullary)()==0&&FN(Green,Nullary)()==1&&FN(Blue,Nullary)()==2&&FN(Single,Nullary)()==0);
 for(int run=0;run<100;run++){
  uint32_t tag=run%3,other_tag=(run+1)%3;
  REQUIRE(FN(Identity,Unary)(tag)==tag&&FN(ColorIdentity,Unary)(tag)==tag);
  REQUIRE(FN(Next,Unary)(tag)==(tag+1)%3);
  REQUIRE(FN(Choose,Choose)(run%2,tag,other_tag)==(run%2?tag:other_tag));
  REQUIRE(FN(Loop,Loop)(run,tag)==(tag+run)%3);
  REQUIRE(FN(Recur,Loop)(run,tag)==(tag+run)%3);
  REQUIRE(FN(Pressure,Eight)(tag,1,2,0,1,2,0,tag)==(tag+1)%3);
  REQUIRE(FN(Eight,Eight)(0,1,2,0,1,2,0,tag)==tag);
  REQUIRE(FN(Mixed,Mixed)(1,0,2.5,1,1,3.5f,1,tag)==tag);
  REQUIRE(FN(Scenario,Scenario)(run)==0);
#ifndef ENUM_NATIVE_ONLY
  REQUIRE(FN(Scenario,Scenario)(run)==concept_evt2h1_scenario(run));
#endif
 }
 /* Separately compiled C calling convention: already resolved native calls
    target this test-only absolute tail thunk. Master image stays unmodified. */
 REQUIRE(image.symbols.data[ORD_Identity].size>=12);
 REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_READWRITE,&old));
 unsigned char *stub=memory+image.symbols.data[ORD_Identity].offset;Unary target=foreign_identity;stub[0]=0x48;stub[1]=0xb8;memcpy(stub+2,&target,8);stub[10]=0xff;stub[11]=0xe0;
 REQUIRE(VirtualProtect(memory,image.byteCount,PAGE_EXECUTE_READ,&old));FlushInstructionCache(GetCurrentProcess(),memory,image.byteCount);
 for(int run=0;run<100;run++){REQUIRE(FN(Loop,Loop)(run,run%3)==(uint32_t)((run+run%3)%3));}
 REQUIRE(callback_count==4950);VirtualFree(memory,0,MEM_RELEASE);
 puts("100 artifact roundtrips, native emissions, varied C caller and native-to-C enum probes passed");return 0;
}
`
