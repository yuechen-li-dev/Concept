package concept

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func evt2MachineFixture(t *testing.T, file string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt2", "machine", "valid", file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	return module, source
}

func TestEVT2x2MachineFrameStepLIR(t *testing.T) {
	cases := []struct {
		file   string
		size   int
		states []string
		yields int
	}{
		{"finite.concept", 16, []string{"Start", "Middle", "Done"}, 0},
		{"yield_resume.concept", 12, []string{"Work"}, 1},
		{"multi_yield.concept", 12, []string{"Work"}, 2},
		{"terminal.concept", 12, []string{"Counting", "Done"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			module, _ := evt2MachineFixture(t, tc.file)
			lir, err := GenerateLIR(module)
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyLIR(lir); err != nil {
				t.Fatal(err)
			}
			if len(lir.Functions) != 2 {
				t.Fatalf("generated functions = %d", len(lir.Functions))
			}
			init, step := lir.Functions[0], lir.Functions[1]
			if init.Machine == nil || step.Machine == nil || init.Machine.Role != "init" || step.Machine.Role != "step" || init.Machine.Identity != step.Machine.Identity || init.Machine.Size != tc.size || step.Machine.Size != tc.size || step.Result != "machine_step_result" {
				t.Fatalf("frame or generated function drift: %+v %+v", init.Machine, step.Machine)
			}
			if lirWidth(step.Result) != 4 {
				t.Fatal("StepResult lost payload-free enum geometry")
			}
			if len(step.Machine.States) != len(tc.states) {
				t.Fatalf("states = %+v", step.Machine.States)
			}
			for i, state := range step.Machine.States {
				if state.ID != i || state.Name != tc.states[i] || step.Blocks[state.Block].Label != "state."+state.Name {
					t.Fatalf("state identity drift: %+v", state)
				}
			}
			if init.Machine.Fields[0].Offset != 0 || init.Machine.Fields[1].Offset != 4 || init.Machine.Fields[2].Offset != 8 {
				t.Fatalf("frame layout drift: %+v", init.Machine.Fields)
			}
			if step.Blocks[0].Term.True != 1 || step.Blocks[1].Label != "already-completed" || len(step.Blocks[1].Instructions) != 1 {
				t.Fatal("completed Step must return without mutating the frame")
			}
			if tc.file == "finite.concept" {
				for i, want := range []string{"1", "2"} {
					if got := machineStateStoreLiteral(step.Blocks[step.Machine.States[i].Block]); got != want {
						t.Fatalf("state %s transition stores %q, want %q", step.Machine.States[i].Name, got, want)
					}
				}
			}
			text := lir.String()
			for _, want := range []string{"frame_field_address", "dispatch.", "invalid-state", "trap invalid_machine_state", "machine_result machine_step_result Completed"} {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q:\n%s", want, text)
				}
			}
			if got := strings.Count(text, "machine_result machine_step_result Yielded"); got != tc.yields {
				t.Fatalf("yielded paths = %d, want %d", got, tc.yields)
			}
			machineIR, err := LowerLirToAmd64Machine(lir)
			if err != nil || len(machineIR.Functions) != 2 {
				t.Fatalf("machine MachineIR lowering: %v", err)
			}
			if machineIR.Functions[0].Args[0].Register != RCX || machineIR.Functions[1].Args[0].Register != RCX || machineIR.Functions[1].Result != "machine_step_result" {
				t.Fatal("generated functions lost Win64 frame-pointer and StepResult ABI")
			}
			trap := false
			for _, block := range machineIR.Functions[1].Blocks {
				if block.Term.Op == "TRAP" {
					trap = true
				}
				for _, in := range block.Instructions {
					if in.Op == "PUSH_STATE" || in.Op == "POP_STATE" {
						t.Fatal("machine pseudo-op escaped into AMD64")
					}
				}
			}
			if !trap {
				t.Fatal("invalid-state trap lost in MachineIR")
			}
			if bridge, err := EncodeMachineBridge(machineIR); err != nil || len(bridge) == 0 {
				t.Fatalf("machine CMIR bridge: %v", err)
			} else if decoded, err := DecodeMachineBridge(bridge); err != nil || decoded.String() != machineIR.String() {
				t.Fatalf("machine CMIR round trip: %v", err)
			}
			for i := 0; i < 100; i++ {
				again, err := GenerateLIR(module)
				if err != nil || again.String() != text {
					t.Fatalf("nondeterministic machine LIR run %d: %v", i, err)
				}
			}
		})
	}
}

func machineStateStoreLiteral(block LIRBlock) string {
	addresses := map[int]bool{}
	constants := map[int]string{}
	for _, in := range block.Instructions {
		if in.Op == "frame_field_address" && in.FrameField == 0 {
			addresses[in.Result] = true
		}
		if in.Op == "const" {
			constants[in.Result] = in.Literal
		}
		if in.Op == "store" && len(in.Args) == 2 && addresses[in.Args[0]] {
			return constants[in.Args[1]]
		}
	}
	return ""
}

func TestEVT2x2MachineUnsupportedBoundaries(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", "machine_parent_resume.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	module.Functions = nil // source Step is qualified; host instance wrappers need EVT2e calls.
	if lir, err := GenerateLIR(module); err != nil || len(lir.Functions) != 2 || lir.Functions[1].Activation == nil {
		t.Fatalf("push source lowering failed: %v", err)
	}
	unsupported := `profile Core; automata Counter with state { float value; } { machine Run { state Start { yield; } } }`
	module, err = Parse("unsupported_machine_field.concept", unsupported)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateLIR(module); err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_MACHINE_FIELD_TYPE") {
		t.Fatalf("unsupported frame field accepted: %v", err)
	}
	withInput := `profile Core; enum S { Go, } automata Door with input S { machine Run { state Start { on S::Go => Start; } } }`
	module, err = Parse("unsupported_machine_input.concept", withInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateLIR(module); err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_AUTOMATA_INPUT Door") {
		t.Fatalf("input reactions were not rejected at the EVT2 boundary: %v", err)
	}
}

func TestEVT2x2MachineSemanticArtifactRetainsBodies(t *testing.T) {
	_, source := evt2MachineFixture(t, "finite.concept")
	artifact, err := CompileSemanticModule("counter_module.concept", "module CounterModule;\n"+string(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, decoded, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Automata) != 1 || decoded.Automata[0].Machines[0].States[0].Body == nil || decoded.Automata[0].Machines[0].Fields[0].Initializer == nil {
		t.Fatal("semantic artifact lost machine body or persistent initializer")
	}
	lir, err := GenerateLIR(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(lir.Functions) != 2 || lir.Functions[1].Machine == nil {
		t.Fatal("decoded machine did not lower to Step LIR")
	}
}

func TestEVT2x2MachineCOracle(t *testing.T) {
	cases := []struct{ file, main string }{
		{"finite.concept", `int Main() {
    instance Counter a(0);
    instance Counter b(10);
    Step(a, Run);
    if (a.state.count != 2 or b.state.count != 10 or State(a, Run) != 1) { return 1; }
    Step(b, Run);
    Step(a, Run);
    if (a.state.count != 4 or b.state.count != 12 or State(a, Run) != 2) { return 2; }
    Step(a, Run);
    Step(a, Run);
    if (a.state.count != 4 or b.state.count != 12) { return 3; }
    return 0;
}`},
		{"yield_resume.concept", `int Main() {
    instance Counter a(0);
    Step(a, Run);
    if (a.state.count != 1 or State(a, Run) != 0) { return 1; }
    Step(a, Run);
    if (a.state.count != 2) { return 2; }
    Step(a, Run);
    if (a.state.count != 2) { return 3; }
    return 0;
}`},
		{"multi_yield.concept", `int Main() {
    instance Counter a(0);
    Step(a, Run);
    if (a.state.count != 1) { return 1; }
    Step(a, Run);
    if (a.state.count != 2) { return 2; }
    Step(a, Run);
    if (a.state.count != 3) { return 3; }
    Step(a, Run);
    if (a.state.count != 3) { return 4; }
    return 0;
}`},
		{"terminal.concept", `int Main() {
    instance Counter a(0);
    Step(a, Run);
    Step(a, Run);
    if (State(a, Run) != 0) { return 1; }
    Step(a, Run);
    if (a.state.count != 3 or State(a, Run) != 1 or Result(a, Run).tag != 1) { return 2; }
    Step(a, Run);
    if (a.state.count != 3) { return 3; }
    return 0;
}`},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			_, source := evt2MachineFixture(t, tc.file)
			path := strings.TrimSuffix(tc.file, ".concept") + ".concept"
			withMain := append(append([]byte(nil), source...), []byte("\n"+tc.main+"\n")...)
			module, err := Parse(path, string(withMain))
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, withMain)
			if err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\nint main(void) { return concept_" + base + "_main(); }\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}

func TestEVT2x2MachineLIRVerifierRejectsMalformed(t *testing.T) {
	module, _ := evt2MachineFixture(t, "yield_resume.concept")
	base, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	clone := func() LIRModule {
		body, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		var out LIRModule
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	checks := []struct {
		name   string
		mutate func(*LIRModule)
		want   string
	}{
		{"frame parameter", func(m *LIRModule) { m.Functions[1].Params[0].Type = "u32" }, "LIR_BAD_MACHINE_FRAME"},
		{"field offset", func(m *LIRModule) { m.Functions[1].Machine.Fields[2].Offset = 9 }, "LIR_BAD_MACHINE_FIELD"},
		{"duplicate state ID", func(m *LIRModule) {
			m.Functions[1].Machine.States = append(m.Functions[1].Machine.States, m.Functions[1].Machine.States[0])
		}, "LIR_BAD_MACHINE_STATE"},
		{"missing dispatch target", func(m *LIRModule) { m.Functions[1].Machine.States[0].Block = 4 }, "LIR_BAD_MACHINE_STATE"},
		{"dispatch edge", func(m *LIRModule) { m.Functions[1].Blocks[2].Term.True = 4 }, "LIR_MISSING_MACHINE_DISPATCH"},
		{"invalid state trap", func(m *LIRModule) { m.Functions[1].Blocks[4].Term.Reason = "" }, "LIR_BAD_TRAP"},
		{"frame address offset", func(m *LIRModule) { m.Functions[1].Blocks[0].Instructions[0].FrameOffset = 99 }, "LIR_BAD_FRAME_OFFSET"},
		{"yield without saved state", func(m *LIRModule) {
			b := &m.Functions[1].Blocks[5]
			b.Instructions = append(b.Instructions[:len(b.Instructions)-2], b.Instructions[len(b.Instructions)-1])
		}, "LIR_MISSING_MACHINE_RESUME_STATE"},
		{"yield with invalid state", func(m *LIRModule) {
			b := &m.Functions[1].Blocks[5]
			for i := range b.Instructions {
				if b.Instructions[i].Op == "const" && b.Instructions[i].Type == "u32" {
					b.Instructions[i].Literal = "99"
				}
			}
		}, "LIR_BAD_MACHINE_STATE_STORE"},
		{"completion without terminal status", func(m *LIRModule) {
			b := &m.Functions[1].Blocks[7]
			b.Instructions = append(b.Instructions[:len(b.Instructions)-2], b.Instructions[len(b.Instructions)-1])
		}, "LIR_MISSING_MACHINE_COMPLETION"},
		{"missing init field", func(m *LIRModule) {
			b := &m.Functions[0].Blocks[0]
			b.Instructions = b.Instructions[:len(b.Instructions)-2]
		}, "LIR_MISSING"},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			m := clone()
			tc.mutate(&m)
			if err := VerifyLIR(m); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed LIR: %v, want %s", err, tc.want)
			}
		})
	}
}
