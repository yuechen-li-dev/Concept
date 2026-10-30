package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const taggedFixturePath = "../../tests/runtime/Testing/TaggedValues.concept"

func taggedFixture(t *testing.T) (Module, []byte, LIRModule) {
	t.Helper()
	source, err := os.ReadFile(taggedFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(taggedFixturePath, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	return module, source, lir
}

func TestEVT2h2CheckedTaggedLIR(t *testing.T) {
	_, _, lir := taggedFixture(t)
	if len(lir.TaggedEnums) != 4 || len(lir.Aggregates) != 3 {
		t.Fatalf("missing checked types: %+v", lir.TaggedEnums)
	}
	byName := map[string]LIRTaggedEnum{}
	for _, domain := range lir.TaggedEnums {
		byName[domain.Name] = domain
	}
	for name, geometry := range map[string][3]int{"Token": {48, 8, 8}, "Other": {8, 4, 4}, "Twin": {8, 4, 4}, "Ordered": {12, 4, 4}} {
		e := byName[name]
		if [3]int{e.Size, e.Alignment, e.PayloadOffset} != geometry {
			t.Fatalf("%s: %+v", name, e)
		}
	}
	if byName["Other"].Type == byName["Twin"].Type || byName["Other"].Identity == byName["Twin"].Identity {
		t.Fatal("coincident layouts erased nominal identity")
	}
	token := byName["Token"]
	if token.Variants[2].Fields[0].Type != "float64" || token.Variants[3].Fields[0].Type != "float32" || token.Variants[4].Fields[0].Type != "bool" {
		t.Fatal("scalar payload representation erased")
	}
	if fields := token.Variants[5].Fields; len(fields) != 4 || fields[2].Offset != 8 || fields[3].Offset != 24 || fields[3].Extent != 3 || fields[3].Stride != 4 {
		t.Fatal("mixed payload ordering/layout erased")
	}
	for _, f := range lir.Functions {
		if f.Name != "MakeOrdered" {
			continue
		}
		var calls []int
		for _, b := range f.Blocks {
			for _, in := range b.Instructions {
				if in.Op == "call" {
					calls = append(calls, in.Result)
				}
				if in.Op == "make_tagged" && (len(calls) != 2 || len(in.Args) != 2 || in.Args[0] != calls[0] || in.Args[1] != calls[1]) {
					t.Fatal("constructor argument evaluation order changed")
				}
			}
		}
	}
	if m, err := LowerLirToAmd64Machine(lir); err != nil || m.Tagged == nil {
		t.Fatalf("missing tagged contract: %v", err)
	}
}

func TestEVT2h2FreshTaggedLIRDeterminism(t *testing.T) {
	start := time.Now()
	_, _, first := taggedFixture(t)
	expected, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, _, again := taggedFixture(t)
		actual, err := json.Marshal(again)
		if err != nil || !bytes.Equal(expected, actual) || again.String() != first.String() {
			t.Fatalf("fresh generation %d differs: %v", i, err)
		}
	}
	t.Logf("100 fresh checked payload LIR generations: %s; snapshot bytes=%d sha256=%x functions=%d", time.Since(start), len(expected), sha256.Sum256(expected), len(first.Functions))
}

func retagLIR(m *LIRModule) {
	old := m.TaggedEnums[0].Type
	next := lirTaggedIdentity(m.TaggedEnums[0])
	m.TaggedEnums[0].Type = next
	remap := func(t LIRType) LIRType { return LIRType(strings.ReplaceAll(string(t), string(old), string(next))) }
	for fi := range m.Functions {
		f := &m.Functions[fi]
		f.Result = remap(f.Result)
		for i := range f.Params {
			f.Params[i].Type = remap(f.Params[i].Type)
		}
		for i := range f.Slots {
			f.Slots[i].Type = remap(f.Slots[i].Type)
		}
		for bi := range f.Blocks {
			for ii := range f.Blocks[bi].Instructions {
				in := &f.Blocks[bi].Instructions[ii]
				in.Type = remap(in.Type)
				in.Aggregate = remap(in.Aggregate)
				for i := range in.ArgTypes {
					in.ArgTypes[i] = remap(in.ArgTypes[i])
				}
			}
		}
	}
}

func TestEVT2h2MalformedTaggedLIR(t *testing.T) {
	_, _, original := taggedFixture(t)
	// Select the Token domain for mutations without relying on first-use order.
	for i, domain := range original.TaggedEnums {
		if domain.Name == "Token" {
			original.TaggedEnums[0], original.TaggedEnums[i] = original.TaggedEnums[i], original.TaggedEnums[0]
			break
		}
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	constructor := func(m *LIRModule) *LIRInstruction {
		for fi := range m.Functions {
			if m.Functions[fi].Name == "MakeInteger" {
				for bi := range m.Functions[fi].Blocks {
					for ii := range m.Functions[fi].Blocks[bi].Instructions {
						in := &m.Functions[fi].Blocks[bi].Instructions[ii]
						if in.Op == "make_tagged" {
							return in
						}
					}
				}
			}
		}
		panic("constructor missing")
	}
	cases := []struct {
		name, code string
		change     func(*LIRModule)
	}{
		{"no domain", "LIR_TAGGED_CONSTRUCTOR", func(m *LIRModule) { m.TaggedEnums = nil }},
		{"duplicate domain", "LIR_TAGGED_DOMAIN", func(m *LIRModule) { m.TaggedEnums = append(m.TaggedEnums, m.TaggedEnums[0]) }},
		{"noncanonical tag", "LIR_TAGGED_VARIANT", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Tag = 7; retagLIR(m) }},
		{"duplicate variant", "LIR_TAGGED_VARIANT", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Name = "End"; retagLIR(m) }},
		{"variant identity", "LIR_TAGGED_VARIANT", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Identity = "foreign"; retagLIR(m) }},
		{"copy permission", "LIR_TAGGED_COPY_ELIGIBILITY", func(m *LIRModule) { m.TaggedEnums[0].CopyEligibility = "none"; retagLIR(m) }},
		{"payload offset", "LIR_TAGGED_LAYOUT", func(m *LIRModule) { m.TaggedEnums[0].PayloadOffset = 4; retagLIR(m) }},
		{"storage size", "LIR_TAGGED_LAYOUT", func(m *LIRModule) { m.TaggedEnums[0].Size = 40; retagLIR(m) }},
		{"overlap", "LIR_TAGGED_FIELD", func(m *LIRModule) { m.TaggedEnums[0].Variants[5].Fields[1].Offset = 0; retagLIR(m) }},
		{"field size", "LIR_TAGGED_FIELD", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Fields[0].Size = 8; retagLIR(m) }},
		{"field identity", "LIR_TAGGED_FIELD", func(m *LIRModule) {
			m.TaggedEnums[0].Variants[1].Fields[0].DeclarationIdentity = "foreign"
			retagLIR(m)
		}},
		{"field type identity", "LIR_TAGGED_FIELD", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Fields[0].TypeIdentity = ""; retagLIR(m) }},
		{"array stride", "LIR_TAGGED_ARRAY", func(m *LIRModule) { m.TaggedEnums[0].Variants[5].Fields[3].Stride = 8; retagLIR(m) }},
		{"unproven reference payload", "LIR_TAGGED_PAYLOAD_TYPE", func(m *LIRModule) { m.TaggedEnums[0].Variants[1].Fields[0].Type = "ref<i32>"; retagLIR(m) }},
		{"unknown variant", "LIR_TAGGED_CONSTRUCTOR", func(m *LIRModule) { constructor(m).Member = 9 }},
		{"missing initialization", "LIR_TAGGED_CONSTRUCTOR", func(m *LIRModule) { constructor(m).Args = nil }},
		{"extra payload", "LIR_TAGGED_CONSTRUCTOR", func(m *LIRModule) { in := constructor(m); in.Args = append(in.Args, in.Args[0]) }},
		{"constructor field type", "LIR_TAGGED_CONSTRUCTOR_TYPE", func(m *LIRModule) {
			for fi := range m.Functions {
				if m.Functions[fi].Name == "MakeFlag" {
					for bi := range m.Functions[fi].Blocks {
						for ii := range m.Functions[fi].Blocks[bi].Instructions {
							in := &m.Functions[fi].Blocks[bi].Instructions[ii]
							if in.Op == "make_tagged" {
								in.Member = 1
								return
							}
						}
					}
				}
			}
		}},
		{"nominal constructor result", "LIR_BAD_RETURN", func(m *LIRModule) {
			for _, domain := range m.TaggedEnums {
				if domain.Name == "Other" {
					constructor(m).Type = domain.Type
					return
				}
			}
		}},
		{"record constructor escape", "LIR_BAD_RECORD_INIT", func(m *LIRModule) { in := constructor(m); in.Op = "make_aggregate"; in.Args = nil }},
		{"unguarded projection", "LIR_BAD_FIELD_ADDRESS", func(m *LIRModule) {
			in := constructor(m)
			in.Op = "field_address"
			in.Aggregate = in.Type
			in.Type = lirPointer("i32")
			in.Member = 0
		}},
		{"uninitialized slot", "LIR_OBJECT_NOT_LIVE", func(m *LIRModule) {
			for fi := range m.Functions {
				for bi := range m.Functions[fi].Blocks {
					for ii := range m.Functions[fi].Blocks[bi].Instructions {
						in := &m.Functions[fi].Blocks[bi].Instructions[ii]
						if in.Op == "initialize_slot" && lirTaggedType(in.Type) {
							in.Op = "store_slot"
							return
						}
					}
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m LIRModule
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			tc.change(&m)
			if err := VerifyLIR(m); err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
}

func TestEVT2h2SourceBoundaries(t *testing.T) {
	codes := map[string]string{"unknown_variant": "CV4103", "arity": "CV4106", "payload_type": "CV4107", "nominal": "CV4116", "projection": "CV4025", "public_geometry": "CV4573", "equality": "CV4830", "extern_abi": "EXTERN_C_ABI_TYPE_INVALID"}
	for name, code := range codes {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("../../language/evt2h2/invalid", name+".concept")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Parse(path, string(source)); err == nil || !strings.Contains(err.Error(), code) {
				t.Fatalf("want %s, got %v", code, err)
			}
		})
	}
	for name, code := range map[string]string{"nested": "EVT2H2_NESTED_TAGGED_ENUM", "match": "EVT2_UNSUPPORTED_EXPR"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("../../language/evt2h2/unsupported", name+".concept")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m, err := Parse(path, string(source))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Generate(m, source); err != nil {
				t.Fatalf("C bootstrap: %v", err)
			}
			if _, err = GenerateLIR(m); err == nil || !strings.Contains(err.Error(), code) {
				t.Fatalf("want %s, got %v", code, err)
			}
		})
	}
}

func TestEVT2h2StrictC11Layout(t *testing.T) {
	m, source, lir := taggedFixture(t)
	outputs, err := Generate(m, source)
	if err != nil {
		t.Fatal(err)
	}
	var header string
	for name := range outputs {
		if strings.HasSuffix(name, ".generated.h") {
			header = name
		}
	}
	if header == "" {
		t.Fatal("missing header")
	}
	var probe strings.Builder
	fmt.Fprintf(&probe, "#include %q\n#include <stddef.h>\n", header)
	for _, e := range lir.TaggedEnums {
		ct := evt1CType(Type{Name: e.Name, Kind: TypeEnum})
		fmt.Fprintf(&probe, "_Static_assert(sizeof(%s)==%d,\"sum size\");\n_Static_assert(_Alignof(%s)==%d,\"sum alignment\");\n_Static_assert(sizeof(((%s*)0)->tag)==4,\"qualified C tag\");\n_Static_assert(offsetof(%s,payload)==%d,\"union offset\");\n", ct, e.Size, ct, e.Alignment, ct, ct, e.PayloadOffset)
		for _, v := range e.Variants {
			if len(v.Fields) == 0 {
				continue
			}
			member := evt1PayloadFieldName(v.Name)
			fmt.Fprintf(&probe, "_Static_assert(sizeof(((%s*)0)->payload.%s)==%d,\"variant size\");\n", ct, member, v.Size)
			for _, f := range v.Fields {
				fmt.Fprintf(&probe, "_Static_assert(offsetof(%s,payload.%s.%s)==%d,\"payload field offset\");\n_Static_assert(sizeof(((%s*)0)->payload.%s.%s)==%d,\"payload field size\");\n", ct, member, f.Name, e.PayloadOffset+f.Offset, ct, member, f.Name, f.Size)
			}
		}
	}
	outputs["tagged_layout_probe.c"] = []byte(probe.String())
	assertR8cStrictC11(t, outputs, "tagged_layout_probe.c")
}

func TestEVT2h2ImportedTaggedIdentity(t *testing.T) {
	_, source, _ := taggedFixture(t)
	artifact, err := CompileSemanticModule(taggedFixturePath, string(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module TaggedConsumer; profile Core; import Testing.TaggedValues; Token ForwardImported(int value) { return MakeInteger(value); }`
	m, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Testing.TaggedValues": artifact})
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(lir.Dependencies) != 1 || lir.Dependencies[0].ModuleIdentity != "Testing.TaggedValues" || !objectDigest(lir.Dependencies[0].ContentSHA256) {
		t.Fatalf("lost defining artifact: %+v", lir.Dependencies)
	}
	for _, e := range lir.TaggedEnums {
		if e.Name == "Token" && e.Identity == "Testing.TaggedValues|Token" {
			return
		}
	}
	t.Fatal("imported nominal sum identity was lost")
}

func TestEVT2h2C11PayloadBits(t *testing.T) {
	m, source, _ := taggedFixture(t)
	outputs, err := Generate(m, source)
	if err != nil {
		t.Fatal(err)
	}
	// Independent C representation probe. These are C-bootstrap values, not
	// Cathedral bytes; source facts own the ordinary behavior expectations.
	runFoundationNativeHarnessOutput(t, outputs, "payload_bits.c", `#include "taggedvalues.generated.h"
#include <stdint.h>
#include <string.h>
int main(void) {
 const uint64_t doubles[]={UINT64_C(0x8000000000000000),UINT64_C(0x3ff0000000000001),UINT64_C(0x7ff8000000001234),UINT64_C(1)};
 for(unsigned i=0;i<sizeof doubles/sizeof doubles[0];i++) {
  double input;memcpy(&input,&doubles[i],8);
  concept_token value=concept_testing__tagged_values_make_decimal(input);
  uint64_t output;memcpy(&output,&value.payload.decimal.value,8);
  if(output!=doubles[i])return 1;
 }
 const uint32_t floats[]={UINT32_C(0x80000000),UINT32_C(0x3f800001),UINT32_C(0x7fc01234),UINT32_C(1)};
 for(unsigned i=0;i<sizeof floats/sizeof floats[0];i++) {
  float input;memcpy(&input,&floats[i],4);
  concept_token value=concept_testing__tagged_values_make_real(input);
  uint32_t output;memcpy(&output,&value.payload.real.value,4);
  if(output!=floats[i])return 2;
 }
 return 0;
}`, "-pedantic-errors", "-O2")
}
