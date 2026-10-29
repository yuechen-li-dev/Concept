package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func objectMachineFixture(t *testing.T) MachineModule {
	t.Helper()
	m, _ := evt2gObjectFixture(t)
	machine, err := GenerateMachineIR(m)
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func objectInstruction(t *testing.T, m *ObjectModule, function, op string) *ObjectInstruction {
	t.Helper()
	for fi := range m.Functions {
		f := &m.Functions[fi]
		if function != "" && f.Name != function {
			continue
		}
		for bi := range f.Blocks {
			for ii := range f.Blocks[bi].Instructions {
				in := &f.Blocks[bi].Instructions[ii]
				if in.Op == op {
					return in
				}
			}
		}
	}
	t.Fatalf("missing %s/%s", function, op)
	return nil
}

func TestEVT2g2MalformedArtifactCorpus(t *testing.T) {
	base := *objectMachineFixture(t).Object
	encoded, _ := json.Marshal(base)
	position, array, particle, scalar := -1, -1, -1, -1
	for i, typ := range base.Types {
		switch typ.Name {
		case "Position":
			position = i
		case "Particle":
			particle = i
		case "float64":
			scalar = i
		}
		if typ.Kind == "array" {
			array = i
		}
	}
	if position < 0 || particle < 0 || array < 0 || scalar < 0 {
		t.Fatal("fixture types")
	}
	type specimen struct {
		name, diagnostic string
		mutate           func(*ObjectModule)
		bytes            func([]byte) []byte
	}
	cases := []specimen{
		{name: "version", diagnostic: "VERSION_MISMATCH", bytes: func(b []byte) []byte { b[8] = 4; return b }},
		{name: "schema", diagnostic: "SCHEMA_HASH_MISMATCH", bytes: func(b []byte) []byte { b[12] ^= 1; return b }},
		{name: "truncated_header", diagnostic: "MALFORMED", bytes: func(b []byte) []byte { return b[:16] }},
		{name: "truncated_payload", diagnostic: "MALFORMED", bytes: func(b []byte) []byte { return b[:len(b)-1] }},
		{name: "trailing", diagnostic: "MALFORMED", bytes: func(b []byte) []byte { return append(b, 0) }},
		{name: "oversized_wire_count", diagnostic: "MALFORMED", bytes: func(b []byte) []byte { binary.LittleEndian.PutUint32(b[44:], 1048577); return b }},
		{name: "type_count", diagnostic: "CAPACITY_EXCEEDED", mutate: func(m *ObjectModule) {
			for len(m.Types) < 257 {
				m.Types = append(m.Types, m.Types[scalar])
			}
		}},
		{name: "type_identity", diagnostic: "TYPE_IDENTITY", mutate: func(m *ObjectModule) { m.Types[position].Identity = "" }},
		{name: "duplicate_type", diagnostic: "TYPE_IDENTITY", mutate: func(m *ObjectModule) { m.Types[position].Identity = m.Types[scalar].Identity }},
		{name: "size", diagnostic: "FIELD_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Size = 8 }},
		{name: "alignment", diagnostic: "LAYOUT_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Align = 3 }},
		{name: "field_offset", diagnostic: "FIELD_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Fields[0].Offset = 1048576 }},
		{name: "field_overlap", diagnostic: "FIELD_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Fields[1].Offset = 0 }},
		{name: "field_size", diagnostic: "FIELD_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Fields[0].Size = 4 }},
		{name: "field_type", diagnostic: "NESTED_LAYOUT_IDENTITY", mutate: func(m *ObjectModule) { m.Types[position].Fields[0].Type = position }},
		{name: "field_identity", diagnostic: "FIELD_IDENTITY", mutate: func(m *ObjectModule) { m.Types[position].Fields[0].Identity = "forged" }},
		{name: "duplicate_field", diagnostic: "FIELD_IDENTITY", mutate: func(m *ObjectModule) { m.Types[position].Fields[1].Identity = m.Types[position].Fields[0].Identity }},
		{name: "nested_layout", diagnostic: "NESTED_LAYOUT_IDENTITY", mutate: func(m *ObjectModule) { m.Types[particle].Fields[0].LayoutIdentity = "stale" }},
		{name: "stride", diagnostic: "ARRAY_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[array].Stride++ }},
		{name: "extent", diagnostic: "ARRAY_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[array].Extent = 0 }},
		{name: "array_size", diagnostic: "ARRAY_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[array].Size++ }},
		{name: "representation", diagnostic: "LAYOUT_GEOMETRY", mutate: func(m *ObjectModule) { m.Types[position].Representation = "scalar" }},
		{name: "copy_eligibility", diagnostic: "COPY_ELIGIBILITY", mutate: func(m *ObjectModule) { m.Types[position].Copy = "none" }},
		{name: "copy_operation", diagnostic: "COPY_ELIGIBILITY", mutate: func(m *ObjectModule) { objectInstruction(t, m, "With", "copy_value").Copy = "none" }},
		{name: "copy_type", diagnostic: "VALUE_TYPE", mutate: func(m *ObjectModule) { objectInstruction(t, m, "With", "copy_value").Type = position }},
		{name: "field_path", diagnostic: "FIELD_GEOMETRY", mutate: func(m *ObjectModule) { objectInstruction(t, m, "Read", "field_address").Offset++ }},
		{name: "value_layout", diagnostic: "VALUE_TYPE", mutate: func(m *ObjectModule) { m.Functions[0].Values[0].LayoutIdentity = "stale" }},
		{name: "storage_layout", diagnostic: "VALUE_TYPE", mutate: func(m *ObjectModule) { m.Functions[0].Slots[0].LayoutIdentity = "stale" }},
		{name: "raw_storage", diagnostic: "OBJECT_NOT_LIVE", mutate: func(m *ObjectModule) { in := objectInstruction(t, m, "Read", "initialize_slot"); in.Op = "store_slot" }},
		{name: "immutable", diagnostic: "IMMUTABLE_ACCESS", mutate: func(m *ObjectModule) {
			for i := range m.Functions {
				if m.Functions[i].Name == "Nested" {
					m.Functions[i].Slots[0].Readonly = true
				}
			}
		}},
		{name: "location_class", diagnostic: "LOCATION_PROVENANCE", mutate: func(m *ObjectModule) {
			in := objectInstruction(t, m, "Read", "slot_address")
			for i := range m.Functions {
				if m.Functions[i].Name == "Read" {
					m.Functions[i].Values[in.Result].Class = "aggregate"
				}
			}
		}},
		{name: "source_origin", diagnostic: "LOCATION_PROVENANCE", mutate: func(m *ObjectModule) { objectInstruction(t, m, "Read", "field_address").Source.Line = 0 }},
		{name: "call_count", diagnostic: "CALL_SIGNATURE", mutate: func(m *ObjectModule) { objectInstruction(t, m, "Calls", "call").Calls[0].Arguments = nil }},
		{name: "call_parameter", diagnostic: "CALL_SIGNATURE", mutate: func(m *ObjectModule) { objectInstruction(t, m, "Calls", "call").Calls[0].Arguments[0].Type = position }},
		{name: "call_result", diagnostic: "CALL_SIGNATURE", mutate: func(m *ObjectModule) { objectInstruction(t, m, "Calls", "call").Calls[0].ResultType = scalar }},
		{name: "return_type", diagnostic: "VALUE_TYPE", mutate: func(m *ObjectModule) {
			for i := range m.Functions {
				if m.Functions[i].Name == "Make" {
					m.Functions[i].ResultType = scalar
				}
			}
		}},
	}
	for _, item := range []struct{ name, record, field string }{
		{"invalid_representation_tag", "WireObjectType", "Representation"},
		{"invalid_evidence_tag", "WireObjectType", "Evidence"},
		{"invalid_address_space", "WireObjectSlot", "Space"},
		{"invalid_value_class", "WireObjectValue", "Class"},
		{"invalid_opcode", "WireObjectInstruction", "Op"},
	} {
		var fields []machineBridgeWireField
		w := &machineBridgeWriter{trace: &fields}
		if err := w.writeWireObjectModule(base); err != nil {
			t.Fatal(err)
		}
		offset := ObjectMachineBridgeHeaderSize + bridgeFieldOffset(t, fields, item.record, item.field)
		cases = append(cases, specimen{name: item.name, diagnostic: "MALFORMED", bytes: func(data []byte) []byte { binary.LittleEndian.PutUint32(data[offset:], 65535); return data }})
	}
	cases = append(cases, specimen{name: "field_capacity", diagnostic: "CAPACITY_EXCEEDED", mutate: func(m *ObjectModule) {
		for len(m.Types[position].Fields) < 65 {
			m.Types[position].Fields = append(m.Types[position].Fields, m.Types[position].Fields[0])
		}
	}})
	cases = append(cases, specimen{name: "function_capacity", diagnostic: "CAPACITY_EXCEEDED", mutate: func(m *ObjectModule) {
		for len(m.Functions) < 129 {
			m.Functions = append(m.Functions, m.Functions[0])
		}
	}})
	path := "../../libraries/Standard/Backend/BridgeValidate5.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var c strings.Builder
	c.WriteString("#include \"bridgevalidate5.generated.h\"\n#include <stdio.h>\n#include <string.h>\nint main(void){static unsigned char data[1048576],output[1048576];FILE *file;size_t n;concept_readonly_span_byte src;concept_span_byte dst={output,sizeof output};concept_result_int_object_error r;\n")
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mutated ObjectModule
			if err := json.Unmarshal(encoded, &mutated); err != nil {
				t.Fatal(err)
			}
			if tc.mutate != nil {
				tc.mutate(&mutated)
			}
			data, err := encodeObjectMachineBridge(mutated)
			if err != nil {
				t.Fatal(err)
			}
			if tc.bytes != nil {
				data = tc.bytes(data)
			}
			if _, err := DecodeObjectMachineBridge(data); err == nil {
				t.Fatal("bootstrap verifier accepted malformed artifact")
			}
			valid, _ := encodeObjectMachineBridge(base)
			if sha256.Sum256(data) == sha256.Sum256(valid) {
				t.Fatal("mutation identity unchanged")
			}
			file := filepath.Join(dir, tc.name+".cmir")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&c, "file=fopen(%q,\"rb\");if(!file)return 200;n=fread(data,1,sizeof data,file);fclose(file);src=(concept_readonly_span_byte){data,n};memset(output,0xa5,sizeof output);for(int repeat=0;repeat<%d;repeat++){r=concept_standard__backend__bridge_validate5_validated_object_bridge_round_trip(src,dst);if(!r.tag||strcmp(concept_standard__backend__bridge_validate5_describe_object_error(r.payload.error.error),\"CMIRAMD5_%s\")){fprintf(stderr,\"%s: got %%s\\n\",r.tag?concept_standard__backend__bridge_validate5_describe_object_error(r.payload.error.error):\"ACCEPTED\");return %d;}}for(size_t j=0;j<sizeof output;j++){if(output[j]!=0xa5)return 201;}\n", filepath.ToSlash(file), map[bool]int{true: 100, false: 1}[i == 0], tc.diagnostic, tc.name, i+1)
		})
	}
	fmt.Fprintf(&c, "puts(\"%d malformed CMIRAMD5 artifacts rejected without output mutation\");return 0;}\n", len(cases))
	for _, verify := range []bool{false, true} {
		name, policy := "Normal", ConservativeCompilationPolicy()
		if verify {
			name, policy = "Verify", VerifyCompilationPolicy()
		}
		t.Run(name, func(t *testing.T) {
			out, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			runFoundationNativeHarnessOutput(t, out, "object_malformed.c", c.String(), "-pedantic-errors", "-O2")
		})
	}
}

func TestEVT2g2ExactBytes(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"minimal", "record struct R { int x; } R F(int x){return R{x};}", "c7eb71a5091132ae94912844688d0c1390e29187c18b363fed89c5849ff5e329"},
		{"nested", "record struct R { int x; } record struct S { R r; double y; } S F(int x){return S{R{x},2.0};}", "eaa6fc1c847242d2aabb70d70ca6f0bf696f7602abff9aaf242a73d31622bb08"},
		{"array", "record struct R { double<array>[4] values; } R F(){return R{[1.0 ...]};}", "15218f44be18a7b6613f0a90bbf5172b662c8226f95ff0cb0cd9e079a7f80c11"},
		{"call", "record struct R { int x; } int Read(R r){return r.x;} int F(int x){return Read(R{x});}", "0889864c477789de85b29ec9307b90b32a3f9beb12aee448817829950c0c10a8"},
		{"result", "record struct R { int x; } R Make(int x){return R{x};} R F(int x){return Make(x);}", "ecac85ad843907fb393155e1e64aa92b4500ccf040889368c274b7792989985d"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Parse(tc.name+".concept", "module Exact; profile Core; "+tc.source)
			if err != nil {
				t.Fatal(err)
			}
			mm, err := GenerateMachineIR(m)
			if err != nil {
				t.Fatal(err)
			}
			data, err := EncodeMachineBridge(mm)
			if err != nil {
				t.Fatal(err)
			}
			fixturePath := filepath.Join("testdata", "evt2g2_"+tc.name+".hex")
			exact := []byte(hex.EncodeToString(data) + "\n")
			if os.Getenv("CONCEPT_UPDATE_CHECKED_OUTPUTS") == "1" {
				if err := os.WriteFile(fixturePath, exact, 0644); err != nil {
					t.Fatal(err)
				}
			}
			pinned, err := os.ReadFile(fixturePath)
			if err != nil || !bytes.Equal(bytes.TrimSpace(pinned), bytes.TrimSpace(exact)) {
				t.Fatalf("exact transport bytes changed: %s (%v)", fixturePath, err)
			}
			sum := sha256.Sum256(data)
			got := hex.EncodeToString(sum[:])
			if tc.want != "" && got != tc.want {
				t.Fatalf("exact bytes: %s", got)
			}
			t.Logf("%s: bytes=%d sha256=%s", tc.name, len(data), got)
		})
	}
}

func TestEVT2g2CanonicalGeometryAndDependencyIdentity(t *testing.T) {
	module, _ := evt2gObjectFixture(t)
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	machine := objectMachineFixture(t)
	fields := 0
	for _, typ := range machine.Object.Types {
		if typ.Kind != "aggregate" {
			continue
		}
		decl, ok := evt1StructView(env, Type{Name: typ.Name})
		if !ok {
			t.Fatalf("unknown semantic declaration %s", typ.Name)
		}
		offsets, size, alignment, err := evt1StructFieldOffsets(env, decl)
		if err != nil {
			t.Fatal(err)
		}
		if size != typ.Size || alignment != typ.Align || len(offsets) != len(typ.Fields) {
			t.Fatal("semantic layout summary changed in transport")
		}
		for i, f := range typ.Fields {
			fs, fa, err := evt1TypeGeometry(env, decl.Fields[i].Type)
			if err != nil || f.Offset != offsets[i] || f.Size != fs || f.Align != fa {
				t.Fatalf("semantic field geometry %s.%s: %v", typ.Name, f.Name, err)
			}
			fields++
		}
	}
	t.Logf("canonical semantic oracle: 8 aggregate layouts, %d field descriptors", fields)
	var previous []byte
	for revision := 1; revision <= 2; revision++ {
		library := fmt.Sprintf("module Geometry; profile Core; record struct R { int value; } int Revision(){return %d;}", revision)
		artifact, err := CompileSemanticModule("Geometry.concept", library, nil)
		if err != nil {
			t.Fatal(err)
		}
		consumer, err := ParseWithSemanticModules("consumer.concept", "module Consumer; profile Core; import Geometry; R Make(int x){return R{x};}", map[string][]byte{"Geometry": artifact})
		if err != nil {
			t.Fatal(err)
		}
		mm, err := GenerateMachineIR(consumer)
		if err != nil {
			t.Fatal(err)
		}
		if len(mm.Object.Dependencies) != 1 || mm.Object.Dependencies[0].Identity != "Geometry" {
			t.Fatalf("dependency identity lost: %+v", mm.Object.Dependencies)
		}
		data, err := EncodeMachineBridge(mm)
		if err != nil {
			t.Fatal(err)
		}
		if revision == 2 && bytes.Equal(previous, data) {
			t.Fatal("artifact dependency change did not alter aggregate contract identity")
		}
		previous = data
	}
}

func TestEVT2g2CRepresentationOracle(t *testing.T) {
	path := "../../language/evt2g2/valid/mixed.concept"
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
	var sample ObjectType
	for _, typ := range mm.Object.Types {
		if typ.Name == "Sample" {
			sample = typ
		}
	}
	if sample.Representation != "C" || len(sample.Fields) != 2 {
		t.Fatal("missing C-repr layout")
	}
	harness := fmt.Sprintf(`#include "mixed.generated.h"
#include <stddef.h>
typedef struct OracleSample { int code; double value; } OracleSample;
_Static_assert(sizeof(OracleSample)==%d,"transmitted size");
_Static_assert(_Alignof(OracleSample)==%d,"transmitted alignment");
_Static_assert(offsetof(OracleSample,code)==%d,"transmitted code offset");
_Static_assert(offsetof(OracleSample,value)==%d,"transmitted value offset");
int main(void){concept_sample r=concept_evt2g2__mixed_exercise(3,0,1.0f,2.0);return r.code==7&&r.value==2.0?0:1;}
`, sample.Size, sample.Align, sample.Fields[0].Offset, sample.Fields[1].Offset)
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		out, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarnessOutput(t, out, "layout_oracle.c", harness, "-pedantic-errors", "-O2")
	}
}

func TestEVT2g2GeneratedDeclarations(t *testing.T) {
	counts := []int{}
	for _, version := range []int{4, 5} {
		path := fmt.Sprintf("../../libraries/Standard/Backend/BridgeCodec%d.concept", version)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
		if err != nil {
			t.Fatal(err)
		}
		view, err := InspectGeneratedDeclarations(m, "")
		if err != nil {
			t.Fatal(err)
		}
		var entries []json.RawMessage
		if err := json.Unmarshal(view, &entries); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, len(entries))
		t.Logf("BridgeCodec%d checked generated declaration count=%d", version, len(entries))
		local := 0
		for _, fn := range m.Functions {
			if fn.Generated != nil && fn.Generated.GenerationModule == m.Name {
				local++
			}
		}
		t.Logf("BridgeCodec%d own derived declarations=%d", version, local)
		if version == 5 && !bytes.Contains(view, []byte("WireObjectField")) {
			t.Fatal("CMIRAMD5 derivations not inspectable")
		}
	}
	t.Logf("cumulative generated declaration delta=%d", counts[1]-counts[0])
	path := "../../libraries/Standard/Backend/BridgeValidate5.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := InspectGeneratedDeclarations(m, "")
	if err != nil {
		t.Fatal(err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(view, &entries); err != nil {
		t.Fatal(err)
	}
	t.Logf("combined CMIRAMD3/4/5 generated declaration count=%d, previous=%d, delta=%d", len(entries), counts[0], len(entries)-counts[0])
}

func TestEVT2g2NativeBoundaryNoMutation(t *testing.T) {
	artifact, err := EncodeMachineBridge(objectMachineFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var c strings.Builder
	c.WriteString("#include \"amd64.generated.h\"\n#include <stdio.h>\n#include <string.h>\nstatic unsigned char artifact[]={")
	for _, b := range artifact {
		fmt.Fprintf(&c, "%d,", b)
	}
	c.WriteString(`};
static unsigned char output[65536];
static concept_native_image image,before;
int main(void){memset(output,0xa5,sizeof output);memset(&image,0x5a,sizeof image);before=image;
concept_readonly_span_byte src={artifact,sizeof artifact};concept_span_byte dst={output,sizeof output};
for(int i=0;i<100;i++){concept_result_int_native_error r=concept_standard__backend__amd64_emit_module(src,dst,&image);
if(!r.tag||strcmp(concept_standard__backend__amd64_describe_native_error(r.payload.error.error),"AMD64_UNSUPPORTED_AGGREGATE_CALL_EMISSION"))return 1;
if(memcmp(&image,&before,sizeof image))return 2;for(size_t j=0;j<sizeof output;j++){if(output[j]!=0xa5)return 3;}}
puts("100 precise aggregate-ABI stops, output and NativeImage untouched");return 0;}
`)
	for _, verify := range []bool{false, true} {
		out := backendTestOutputs(t, verify)
		runFoundationNativeHarnessOutput(t, out, "object_boundary.c", c.String(), "-pedantic-errors", "-O2")
	}
}

func TestEVT2g2ContextualArrayAssignmentC11(t *testing.T) {
	const source = `module ArrayAssignment; profile Core; int Check(){ bool<array>[4] flags=[true ...]; flags=[false ...]; flags=[true,false,false,true]; int<array>[3] numbers=[1 ...]; numbers=[4,5,6]; if(flags[0] and not flags[1] and flags[3]){return numbers[0]+numbers[2];}return 0;}`
	m, err := Parse("arrayassignment.concept", source)
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
		runFoundationNativeHarnessOutput(t, out, "array_assignment.c", "#include \"arrayassignment.generated.h\"\nint main(void){return concept_array_assignment_check()==10?0:1;}\n", "-pedantic-errors", "-O2")
	}
}

func TestEVT2g2GeneratedSchemaAndTransport(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeSchema5.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	g, err := GenerateMachineBridgeCodecs(m)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{"internal/concept/machineir_bridge5_codec_generated.go": g.Go, "libraries/Standard/Backend/BridgeCodec5.concept": g.Concept, "docs/design/EVT2-MACHINEIR-BRIDGE5.schema.json": g.Metadata} {
		got, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil || !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
			t.Fatalf("stale generated %s: %v", path, err)
		}
	}
	fixture := objectMachineFixture(t)
	input, _ := evt2gObjectFixture(t)
	artifact, err := EncodeMachineBridge(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact[:8]) != "CMIRAMD5" || bytes.Contains(artifact, []byte("record_with")) {
		t.Fatal("wrong version/source-only opcode survived")
	}
	start := time.Now()
	var encodeTime, decodeTime time.Duration
	text := fixture.String()
	for i := 0; i < 100; i++ {
		regenerated, err := GenerateMachineIR(input)
		if err != nil || regenerated.String() != text {
			t.Fatalf("checked input/printer run %d: %v", i, err)
		}
		encodeStart := time.Now()
		got, err := EncodeMachineBridge(regenerated)
		encodeTime += time.Since(encodeStart)
		if err != nil || !bytes.Equal(got, artifact) {
			t.Fatalf("encode %d: %v", i, err)
		}
		decodeStart := time.Now()
		round, err := DecodeMachineBridge(got)
		decodeTime += time.Since(decodeStart)
		if err != nil || round.String() != text {
			t.Fatalf("decode %d: %v", i, err)
		}
		again, err := EncodeMachineBridge(round)
		if err != nil || !bytes.Equal(again, artifact) {
			t.Fatal("Go roundtrip changed contract")
		}
		next, err := GenerateMachineBridgeCodecs(m)
		if err != nil || next.Hash != g.Hash || !bytes.Equal(next.Go, g.Go) || !bytes.Equal(next.Concept, g.Concept) {
			t.Fatal("schema nondeterminism")
		}
	}
	t.Logf("CMIRAMD5 artifact=%d bytes types=%d functions=%d; 100 encode/decode/schema cycles=%s", len(artifact), len(fixture.Object.Types), len(fixture.Object.Functions), time.Since(start))
	t.Logf("100 encodes=%s; 100 decodes=%s; schema metadata=%d bytes; generated Go=%d bytes, Concept=%d bytes", encodeTime, decodeTime, len(g.Metadata), len(g.Go), len(g.Concept))
}

func TestEVT2g2PolicyContractParity(t *testing.T) {
	module, _ := evt2gObjectFixture(t)
	var previous []byte
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		mir := buildMIR(module, env)
		evt1ProjectPersistentFactSubjects(&mir)
		evt1QualifyMIRFacts(&mir)
		evt1QualifyHardwareFacts(&mir, env)
		if err := evt1ValidateMIR(mir); err != nil {
			t.Fatal(err)
		}
		facts := NewSemanticFactSet(mir.SemanticFacts)
		plan, err := PlanModule(&mir, &facts, GenericC11Target(), *env.profile, policy)
		if err != nil {
			t.Fatal(err)
		}
		lir, err := LowerMirToLir(&mir, plan, env)
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
		if previous != nil && !bytes.Equal(previous, data) {
			t.Fatal("Normal/Verify semantic aggregate contracts differ")
		}
		previous = data
	}
}

func TestEVT2g2ConceptArtifactOnlyRoundTrip(t *testing.T) {
	fixture := objectMachineFixture(t)
	artifact, err := EncodeMachineBridge(fixture)
	if err != nil {
		t.Fatal(err)
	}
	path := "../../libraries/Standard/Backend/BridgeValidate5.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	var c strings.Builder
	c.WriteString("#include \"bridgevalidate5.generated.h\"\n#include <stdio.h>\n#include <string.h>\n#include <time.h>\n")
	mixedPath := "../../language/evt2g2/valid/mixed.concept"
	mixedSource, err := os.ReadFile(mixedPath)
	if err != nil {
		t.Fatal(err)
	}
	mixed, err := Parse(mixedPath, string(mixedSource))
	if err != nil {
		t.Fatal(err)
	}
	mixedMachine, err := GenerateMachineIR(mixed)
	if err != nil {
		t.Fatal(err)
	}
	mixedBytes, err := EncodeMachineBridge(mixedMachine)
	if err != nil {
		t.Fatal(err)
	}
	// Closed instances are supplied through a semantic artifact, not reopened
	// declarations. Same-width int/float fields must retain distinct identities.
	declarations, err := CompileSemanticModule("Boxes.concept", "module Boxes; profile Core; template <typename T> record struct Box { T value; }", nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := ParseWithSemanticModules("boxuse.concept", `module BoxUse; profile Core; import Boxes;
int IntBox(int x){Box<int> box=Box<int>{x};return box.value;}
float FloatBox(float x){Box<float> box=Box<float>{x};return box.value;}
`, map[string][]byte{"Boxes": declarations})
	if err != nil {
		t.Fatal(err)
	}
	closed, err := GenerateMachineIR(consumer)
	if err != nil {
		t.Fatal(err)
	}
	var identities []string
	for _, typ := range closed.Object.Types {
		if typ.Kind == "aggregate" {
			identities = append(identities, typ.Identity)
		}
	}
	if len(identities) != 2 || identities[0] == identities[1] || len(closed.Object.Dependencies) != 1 {
		t.Fatal("closed generic/import identity lost")
	}
	closedBytes, err := EncodeMachineBridge(closed)
	if err != nil {
		t.Fatal(err)
	}
	for i, data := range [][]byte{artifact, mixedBytes, closedBytes} {
		fmt.Fprintf(&c, "static const unsigned char artifact%d[]={", i)
		for _, b := range data {
			fmt.Fprintf(&c, "%d,", b)
		}
		c.WriteString("};\n")
	}
	c.WriteString("int main(void){unsigned char output[sizeof artifact0];concept_readonly_span_byte inputs[]={{artifact0,sizeof artifact0},{artifact1,sizeof artifact1},{artifact2,sizeof artifact2}};concept_span_byte dst={output,sizeof output};clock_t start=clock();for(int specimen=0;specimen<3;specimen++){concept_readonly_span_byte src=inputs[specimen];for(int i=0;i<100;i++){concept_result_int_object_error r=concept_standard__backend__bridge_validate5_validated_object_bridge_round_trip(src,dst);if(r.tag){fprintf(stderr,\"%s\\n\",concept_standard__backend__bridge_validate5_describe_object_error(r.payload.error.error));return 1;}if(r.payload.ok.value!=(int)src.length||memcmp(src.data,output,src.length))return 2;}}printf(\"3 x 100 Concept validated CMIRAMD5 roundtrips: %.3f ms\\n\",1000.0*(double)(clock()-start)/CLOCKS_PER_SEC);return 0;}\n")
	for _, verify := range []bool{false, true} {
		name, policy := "Normal", ConservativeCompilationPolicy()
		if verify {
			name, policy = "Verify", VerifyCompilationPolicy()
		}
		t.Run(name, func(t *testing.T) {
			out, err := GenerateForTargetWithPolicy(m, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			runFoundationNativeHarnessOutput(t, out, "object_contract.c", c.String(), "-pedantic-errors", "-O2")
		})
	}
}
