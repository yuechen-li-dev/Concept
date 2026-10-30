package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

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
		lir.Functions = append(lir.Functions, LIRFunction{Identity: p.name + "|" + p.name + "(Token)", Name: p.name, Params: []LIRValue{{ID: 0, Type: token.Type}}, Result: typ, Source: Span{Line: 1, Column: 1}, Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{{Op: "checked_payload", Result: 1, Type: typ, Args: []int{0}, Slot: -1, Aggregate: token.Type, Member: p.variant, Extent: p.field, Source: Span{Line: 1, Column: 1}}}, Term: LIRTerminator{Op: "return", Value: 1, Source: Span{Line: 1, Column: 1}}}}})
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
func TestEVT2h2TaggedArtifactAdmission(t *testing.T) {
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
	add("payload geometry", func(m *TaggedModule) { m.Domains[0].PayloadOffset = 1 })
	add("field overlap", func(m *TaggedModule) { m.Domains[0].Variants[0].Fields[1].Offset = 0 })
	add("copy authority", func(m *TaggedModule) { m.Base.Objects.Types[m.Domains[0].Type].Copy = "none" })
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
	cases = append(cases, specimen{"truncated", valid[:len(valid)-1]}, specimen{"trailing", append(append([]byte(nil), valid...), 0)})
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
		fmt.Fprintf(&defs, "{FILE*f=fopen(%q,\"rb\");if(!f)return 1;size_t n=fread(input,1,sizeof input,f);fclose(f);concept_readonly_span_byte s={input,n};memset(output,0xa5,sizeof output);concept_result_int_tagged_error r=F(validated_tagged_bridge_round_trip)(s,dst);if(!r.tag){fprintf(stderr,\"accepted %s\\n\");return 2;}for(size_t j=0;j<sizeof output;j++)if(output[j]!=0xa5)return 3;}\n", filepath.ToSlash(p), c.name)
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
	}
	t.Logf("artifact-only admission Normal/Verify: %d invalid cases, 100 exact valid round trips", len(cases))
}
