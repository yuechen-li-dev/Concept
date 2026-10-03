package concept

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
)

func bridgeTracedPayload(t *testing.T, m MachineModule) ([]byte, []machineBridgeWireField) {
	t.Helper()
	var fields []machineBridgeWireField
	w := &machineBridgeWriter{trace: &fields}
	if err := w.writeWireMachineModule(m); err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(w.Bytes()), fields
}
func bridgeFieldOffset(t *testing.T, fields []machineBridgeWireField, record, field string) int {
	t.Helper()
	for _, f := range fields {
		if f.Record == record && f.Field == field {
			return f.Offset
		}
	}
	t.Fatalf("missing generated field trace %s.%s", record, field)
	return 0
}
func TestR9a2BridgeFieldDifferential(t *testing.T) {
	expected, fields := bridgeTracedPayload(t, machineFixture(t))
	actual := bytes.Clone(expected)
	offset := bridgeFieldOffset(t, fields, "WireMachineOperand", "Width")
	actual[offset] ^= 1
	err := machineBridgePayloadDifference(expected, actual, fields)
	if err == nil || !strings.Contains(err.Error(), "WireMachineOperand.Width") || !strings.Contains(err.Error(), fmt.Sprintf("expected_offset=%d actual_offset=%d", offset, offset)) {
		t.Fatalf("opaque differential: %v", err)
	}
}

func TestR9a2BridgeSchemaMutationAndArtifactIdentity(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeSchema.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module := bridgeCheckedSchema(t)
	original, err := GenerateMachineBridgeCodecs(module)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileSemanticModule(path, string(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, loaded, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := evt1BuildReflectionResults(&loaded, env); err != nil {
		t.Fatal(err)
	}
	fromArtifact, err := GenerateMachineBridgeCodecs(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if original.Hash != fromArtifact.Hash || !bytes.Equal(original.Go, fromArtifact.Go) || !bytes.Equal(original.Concept, fromArtifact.Concept) {
		t.Fatal("artifact-only schema generation changed identity or codecs")
	}
	// A checked fixture mutation changes both generators and the semantic hash.
	mutatedSource := strings.Replace(string(source), "int Disp;", "int Disp; int AddedWireField;", 1)
	if mutatedSource == string(source) {
		t.Fatal("schema mutation anchor missing")
	}
	mutated, err := Parse("mutated-schema.concept", mutatedSource)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := GenerateMachineBridgeCodecs(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Hash == original.Hash || !bytes.Contains(changed.Go, []byte("AddedWireField")) || !bytes.Contains(changed.Metadata, []byte("AddedWireField")) {
		t.Fatal("schema change missing on a generated side")
	}
	// Concept's reader is reflection-derived; its generated input list must see
	// the new checked field, even though its derive-site source remains the same.
	codecSource, err := os.ReadFile("../../libraries/Standard/Backend/BridgeCodec.concept")
	if err != nil {
		t.Fatal(err)
	}
	imports, err := SemanticModuleImports("BridgeCodec.concept", string(codecSource))
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources([]string{"../../libraries"}, imports)
	if err != nil {
		t.Fatal(err)
	}
	changedArtifact, err := CompileSemanticModule("mutated-schema.concept", mutatedSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["Standard.Backend.BridgeSchema"] = changedArtifact
	// Rebuild dependencies from the fixture artifact, retaining hash checks.
	for _, name := range []string{"BridgeRead", "BridgeDerive"} {
		data, err := os.ReadFile("../../libraries/Standard/Backend/" + name + ".concept")
		if err != nil {
			t.Fatal(err)
		}
		body, err := CompileSemanticModule(name+".concept", string(data), artifacts)
		if err != nil {
			t.Fatal(err)
		}
		artifacts["Standard.Backend."+name] = body
	}
	consumer, err := ParseWithSemanticModules("BridgeCodec.concept", string(changed.Concept), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range consumer.Functions {
		if fn.Generated != nil && fn.Generated.ReflectedType.Name == "WireMachineOperand" {
			for _, input := range fn.Generated.Inputs {
				if input.Name == "AddedWireField" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("Concept generated decoder did not see schema field")
	}
	data, err := EncodeMachineBridge(machineFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(consumer, changed.Concept)
	if err != nil {
		t.Fatal(err)
	}
	var staleHarness strings.Builder
	staleHarness.WriteString("#include \"bridgecodec.generated.h\"\nint main(void) { const unsigned char input[]={")
	for _, b := range data[:MachineBridgeHeaderSize] {
		fmt.Fprintf(&staleHarness, "%d,", b)
	}
	staleHarness.WriteString("};concept_bridge_reader reader={{input,sizeof input},0};concept_result_void_backend_error r=concept_standard__backend__bridge_codec_read_bridge_identity(&reader);return r.tag==1 && r.payload.error.error.tag==7 ? 0:1;}\n")
	runFoundationNativeHarness(t, outputs, "stale_schema.c", staleHarness.String())
	changedHash := bytes.Clone(data)
	copy(changedHash[12:44], bytes.Repeat([]byte{0}, 32))
	if _, err := DecodeMachineBridge(changedHash); err == nil || !strings.Contains(err.Error(), "MIR_BRIDGE_SCHEMA_HASH_MISMATCH") {
		t.Fatalf("stale schema accepted: %v", err)
	}
}

func bridgeMalformedCases(t *testing.T) map[string][]byte {
	t.Helper()
	m := machineFixture(t)
	data, err := EncodeMachineBridge(m)
	if err != nil {
		t.Fatal(err)
	}
	_, fields := bridgeTracedPayload(t, m)
	cases := map[string][]byte{}
	for name, offset := range map[string]int{"magic": 0, "version": 8, "hash": 12} {
		wrong := bytes.Clone(data)
		wrong[offset] ^= 1
		cases[name] = wrong
	}
	for name, spec := range map[string]struct {
		record, field string
		value         uint32
	}{
		"enum":            {"WireMachineInstruction", "Op", 0x7fffffff},
		"bool":            {"WireMachineFrame", "HasCalls", 2},
		"register":        {"WireMachineArg", "Register", 16},
		"count-negative":  {"WireMachineModule", "Functions", 0xffffffff},
		"count-large":     {"WireMachineModule", "Functions", 1048577},
		"text-length":     {"WireMachineFunction", "Identity", 1048576},
		"vreg-reference":  {"WireMachineOperand", "ID", 0x7fffffff},
		"block-reference": {"WireMachineTerminator", "True", 0x7fffffff},
	} {
		wrong := bytes.Clone(data)
		offset := machineBridgeHeaderSize + bridgeFieldOffset(t, fields, spec.record, spec.field)
		binary.LittleEndian.PutUint32(wrong[offset:offset+4], spec.value)
		cases[name] = wrong
	}
	for _, n := range []int{0, 1, 7, 8, 11, 12, 43, 44, 45, len(data) / 2, len(data) - 1} {
		cases[fmt.Sprintf("truncated-%d", n)] = bytes.Clone(data[:n])
	}
	cases["trailing"] = append(bytes.Clone(data), 1)
	return cases
}
func TestR9a2BridgeMalformedInput(t *testing.T) {
	for name, data := range bridgeMalformedCases(t) {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeMachineBridge(data); err == nil {
				t.Fatal("malformed bridge accepted")
			}
		})
	}
}
func TestR9a2BridgeConceptMalformedInput(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Backend/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots("../../libraries/Standard/Backend/AMD64.concept", string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var harness strings.Builder
	harness.WriteString("#include \"amd64.generated.h\"\n#include <string.h>\nint main(void) {\n")
	for name, data := range bridgeMalformedCases(t) {
		if len(data) == 0 {
			data = []byte{0}
			fmt.Fprint(&harness, "{static const unsigned char input[]={0};size_t size=0;")
		} else {
			fmt.Fprint(&harness, "{static const unsigned char input[]={")
			for _, b := range data {
				fmt.Fprintf(&harness, "%d,", b)
			}
			fmt.Fprint(&harness, "};size_t size=sizeof input;\n")
		}
		expectedTag := 0
		if name == "magic" || name == "version" {
			expectedTag = 1
		}
		if name == "hash" {
			expectedTag = 7
		}
		fmt.Fprintf(&harness, "unsigned char output[4096];memset(output,0xa5,sizeof output);concept_readonly_span_byte src={input,size};concept_span_byte dst={output,sizeof output};concept_result_int_backend_error r=concept_standard__backend__amd64_emit_function(src,0,dst);if(r.tag!=1 || r.payload.error.error.tag!=%d)return 1;for(size_t i=0;i<sizeof output;++i)if(output[i]!=0xa5)return 2; }", expectedTag)
		t.Logf("Concept safely rejects %s", name)
	}
	harness.WriteString("return 0;}\n")
	runFoundationNativeHarness(t, outputs, "bridge_malformed.c", harness.String())
}
