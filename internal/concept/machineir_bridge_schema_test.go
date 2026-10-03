package concept

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func bridgeCheckedSchema(t *testing.T) Module {
	t.Helper()
	path := filepath.Join("..", "..", "libraries", "Standard", "Backend", "BridgeSchema.concept")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(data))
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestR9a2BridgeGeneratedOutputs(t *testing.T) {
	m := bridgeCheckedSchema(t)
	generated, err := GenerateMachineBridgeCodecs(m)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{
		"internal/concept/machineir_bridge_codec_generated.go": generated.Go,
		"libraries/Standard/Backend/BridgeCodec.concept":       generated.Concept,
		"docs/design/EVT2-MACHINEIR-BRIDGE.schema.json":        generated.Metadata,
	} {
		actual, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), want) {
			t.Fatalf("stale generated codec %s; run go run ./cmd/machinebridgegen .", path)
		}
	}
	for i := 0; i < 100; i++ {
		again, err := GenerateMachineBridgeCodecs(m)
		if err != nil {
			t.Fatal(err)
		}
		if again.Hash != generated.Hash || !bytes.Equal(again.Go, generated.Go) || !bytes.Equal(again.Concept, generated.Concept) {
			t.Fatalf("generation unstable run %d", i)
		}
	}
}
func TestR9a2BridgeFrozenPayloadParity(t *testing.T) {
	for name, m := range bridgeParityCorpus(t) {
		t.Run(name, func(t *testing.T) { bridgeFrozenParity(t, name, m) })
	}
}
func bridgeFrozenOracle(t *testing.T, name, suffix string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "machinebridge", strings.TrimSuffix(name, ".concept")+suffix))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func bridgeFrozenParity(t *testing.T, name string, m MachineModule) {
	old := bridgeFrozenOracle(t, name, ".cmir1")
	derived, err := EncodeMachineBridge(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(old[8:], derived[machineBridgeHeaderSize:]) {
		_, fields := bridgeTracedPayload(t, m)
		t.Fatal(machineBridgePayloadDifference(old[8:], derived[machineBridgeHeaderSize:], fields))
	}
	decoded, err := DecodeMachineBridge(derived)
	if err != nil {
		t.Fatal(err)
	}
	if !bridgeSemanticEqual(reflect.ValueOf(m), reflect.ValueOf(decoded)) {
		t.Fatal("roundtrip differs from transported semantic fields")
	}
	for run := 0; run < 100; run++ {
		again, err := EncodeMachineBridge(m)
		if err != nil || !bytes.Equal(again, derived) {
			t.Fatalf("bridge unstable run %d: %v", run, err)
		}
		roundtrip, err := DecodeMachineBridge(again)
		if err != nil || !bridgeSemanticEqual(reflect.ValueOf(decoded), reflect.ValueOf(roundtrip)) {
			t.Fatalf("decoded representation unstable run %d: %v", run, err)
		}
	}
}
func bridgeSemanticEqual(a, b reflect.Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !bridgeSemanticEqual(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !bridgeSemanticEqual(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

func bridgeParityCorpus(t *testing.T) map[string]MachineModule {
	t.Helper()
	corpus := map[string]MachineModule{"core": machineFixture(t)}
	for _, file := range []string{"finite.concept", "yield_resume.concept", "multi_yield.concept"} {
		m, _ := evt2MachineFixture(t, file)
		machine, err := GenerateMachineIR(m)
		if err != nil {
			t.Fatal(err)
		}
		corpus[file] = machine
	}
	pushdown, _ := pushdownFixture(t)
	pushdownMachine, err := GenerateMachineIR(pushdown)
	if err != nil {
		t.Fatal(err)
	}
	corpus["pushdown"] = pushdownMachine
	path := "../../language/evt1/machine-stack/valid/machine_parent_resume.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	a := buildMIR(module, env).Automata[0]
	layout, err := planActivationStack(a, env)
	if err != nil {
		t.Fatal(err)
	}
	init, err := lowerActivationRootInit(a, layout)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := LowerLirToAmd64Machine(LIRModule{Functions: []LIRFunction{init}})
	if err != nil {
		t.Fatal(err)
	}
	corpus["parent-resume"] = machine
	stride := LIRFunction{Identity: "StoreStride12", Name: "StoreStride12", Params: []LIRValue{{ID: 0, Type: "ptr<u32>"}, {ID: 1, Type: "u32"}, {ID: 2, Type: "u32"}}, Result: "void",
		Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{
			{Op: "check_index", Result: -1, Type: "void", Args: []int{1}, Slot: -1, Extent: 3},
			{Op: "index_address", Result: 3, Type: "ptr<u32>", Args: []int{0, 1}, Slot: -1, Extent: 3, Stride: 12, FrameOffset: 4},
			{Op: "store", Result: -1, Type: "u32", Args: []int{3, 2}, Slot: -1},
		}, Term: LIRTerminator{Op: "return"}}}}
	machine, err = LowerLirToAmd64Machine(LIRModule{Functions: []LIRFunction{stride}})
	if err != nil {
		t.Fatal(err)
	}
	corpus["stride12"] = machine
	return corpus
}
func TestR9a2BridgeConceptGeneration(t *testing.T) {
	path := filepath.Join("..", "..", "libraries", "Standard", "Backend", "BridgeCodec.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Functions) == 0 {
		t.Fatal("no generated declarations")
	}
	// Pin the actual local call graph, without accepting imported effect grants.
	m.OperationEffects = nil
	proof, err := parseSyntaxModule("bridge-proof.concept", `profile Core; void PinBridgeProof() { Assert.Concept<NoAllocation>(BridgeRoundTrip, "generated codec uses input views and caller output"); }`)
	if err != nil {
		t.Fatal(err)
	}
	m.Functions = append(m.Functions, proof.Functions...)
	if _, err := analyzeModule(m); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(m, source); err != nil {
		t.Fatal(err)
	}
}
func TestR9a2BridgeDeclarationDeterminism100(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeCodec.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	imports, err := SemanticModuleImports(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources([]string{"../../libraries"}, imports)
	if err != nil {
		t.Fatal(err)
	}
	var firstView []byte
	var firstOutputs Outputs
	for run := 0; run < 100; run++ {
		module, err := ParseWithSemanticModules(path, string(source), artifacts)
		if err != nil {
			t.Fatal(err)
		}
		view, err := InspectGeneratedDeclarations(module, "")
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			firstView = view
			firstOutputs = outputs
			continue
		}
		if !bytes.Equal(view, firstView) {
			t.Fatalf("generated Concept identities/declarations changed run %d", run)
		}
		for name, body := range firstOutputs {
			if !bytes.Equal(body, outputs[name]) {
				t.Fatalf("generated Concept artifact %s changed run %d", name, run)
			}
		}
	}
}

func TestR9a2BridgeConceptRoundTrip(t *testing.T) {
	path := "../../libraries/Standard/Backend/BridgeCodec.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var harness strings.Builder
	harness.WriteString("#include \"bridgecodec.generated.h\"\n#include <string.h>\nint main(void) {\n")
	for name := range bridgeParityCorpus(t) {
		old := bridgeFrozenOracle(t, name, ".cmir1")
		data := old[8:]
		fmt.Fprint(&harness, "{static const unsigned char input[]={")
		for _, b := range data {
			fmt.Fprintf(&harness, "%d,", b)
		}
		fmt.Fprintf(&harness, "};unsigned char output[%d]={0};concept_readonly_span_byte src={input,sizeof input};concept_span_byte dst={output,sizeof output};\n", len(data))
		fmt.Fprintln(&harness, "for (int run=0;run<100;++run) {concept_result_int_backend_error r=concept_standard__backend__bridge_codec_bridge_round_trip(src,dst);if(r.tag!=0 || r.payload.ok.value!=(int)sizeof input || memcmp(input,output,sizeof input)) return 1;} }")
		t.Logf("Concept canonical roundtrip: %s, %d payload bytes, 100 runs", name, len(data))
	}
	harness.WriteString("return 0;}\n")
	runFoundationNativeHarness(t, outputs, "bridge_roundtrip.c", harness.String())
}
