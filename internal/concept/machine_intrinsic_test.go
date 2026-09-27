package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestAMD64TypedIntrinsicMIRHelperAndTarget(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["amd64.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "machine_intrinsic" {
				if !op.NoAllocation {
					t.Fatalf("machine operation may allocate: %+v", op)
				}
				got = append(got, op.Detail)
			}
		}
	}
	for _, want := range []string{"AMD64.Pause", "AMD64.ReadTimestamp", "AMD64.In8", "AMD64.In16", "AMD64.In32", "AMD64.Out8", "AMD64.Out16", "AMD64.Out32"} {
		if !strings.Contains(strings.Join(got, ","), want) || !strings.Contains(string(outputs["amd64.machine.S"]), machineIntrinsics[want].symbol+":") {
			t.Fatalf("missing %s from MIR or helper: %+v", want, got)
		}
	}
	if !strings.Contains(string(outputs["amd64.generated.c"]), "ConceptAMD64Pause(") {
		t.Fatal("typed Pause wrapper did not call target helper")
	}
	_, err = GenerateForTarget(module, source, AArch64GenericTarget())
	if err == nil || !strings.Contains(err.Error(), "MACHINE_ARCHITECTURE_MISMATCH") {
		t.Fatalf("wrong target was accepted: %v", err)
	}
}

func TestAMD64PauseAndTimestampExecuteNatively(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("AMD64 execution requires an AMD64 host")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC is not available")
	}
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := evt1SemanticSymbolBase(module)
	harness := fmt.Sprintf("#include \"amd64.generated.h\"\nint main(void) { %s(); uint64_t a = %s(); uint64_t b = %s(); return b < a; }\n", evt1FunctionSymbol(base, "Pause"), evt1FunctionSymbol(base, "ReadTimestamp"), evt1FunctionSymbol(base, "ReadTimestamp"))
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "machine-test")
	build := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-O2", "amd64.generated.c", "amd64.machine.S", "harness.c", "-o", binary)
	build.Dir = dir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 wrapper/helper build failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("native Pause/timestamp execution failed: %v\n%s", err, output)
	}
}

func TestAMD64MachineIntrinsicSignatureRejectsForgery(t *testing.T) {
	source := `module Standard.Machine.AMD64; profile Core;
[[machine("AMD64.In8")]] extern "C" uint8 ConceptAMD64In8(uint32 port);`
	_, err := Parse("bad_machine.concept", source)
	if err == nil || !strings.Contains(err.Error(), "MACHINE_INTRINSIC_SIGNATURE") {
		t.Fatalf("invalid port width was accepted: %v", err)
	}
}

func TestAMD64PortCallRejectsOutOfRangeLiteral(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(source), nil)
	consumer := `module BadPort; profile Core; import Standard.Machine.AMD64;
uint8 Bad() { return In8(65536); }`
	_, err = ParseWithSemanticModules("bad_port.concept", consumer, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err == nil || !strings.Contains(err.Error(), "CV4107") {
		t.Fatalf("out-of-range port was accepted: %v", err)
	}
}

func TestAMD64MachineIntrinsicArtifactOnlyConsumer(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(source), nil)
	consumer := `module MachineConsumer; profile Core; import Standard.Machine.AMD64;
uint64 Probe() { Pause(); return ReadTimestamp(); }
void Prove() {
    Assert.Concept<NoAllocation>(Probe, "machine call has no allocation");
    Assert.Concept<HardwareRead>(Probe, "timestamp read survives import");
}`
	module, err := ParseWithSemanticModules("machine_consumer.concept", consumer, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(consumer), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["machine_consumer.machine.S"]), "ConceptAMD64ReadTimestamp:") {
		t.Fatal("artifact-only consumer lost machine helper")
	}
}

func TestAMD64MachineArtifactsAreByteIdenticalAcross100Runs(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	var expected Outputs
	var expectedArtifact []byte
	for i := 0; i < 100; i++ {
		module, err := Parse("amd64.concept", string(source))
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("amd64.concept", string(source), nil)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && (!reflect.DeepEqual(outputs, expected) || !reflect.DeepEqual(artifact, expectedArtifact)) {
			t.Fatalf("machine output changed on run %d", i+1)
		}
		expected, expectedArtifact = outputs, artifact
	}
}
