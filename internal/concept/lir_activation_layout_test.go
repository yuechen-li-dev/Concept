package concept

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func activationFixturePlan(t *testing.T, file string) ActivationStackLayout {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	mir := buildMIR(module, env)
	if err := evt1ValidateMIR(mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Automata) != 1 {
		t.Fatalf("automata count %d", len(mir.Automata))
	}
	plan, err := planActivationStack(mir.Automata[0], env)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestEVT2x4ClosedActivationLayout(t *testing.T) {
	for _, tc := range []struct {
		file                         string
		names                        []string
		frameSizes                   []int
		maxSize, slotSize, totalSize int
	}{
		{"machine_parent_resume.concept", []string{"Parent", "Child"}, []int{8, 4}, 8, 12, 108},
		{"machine_multiple_nested_frames.concept", []string{"Parent", "Child", "Grandchild"}, []int{4, 4, 4}, 4, 8, 76},
		{"machine_recursive_frames.concept", []string{"Run"}, []int{8}, 8, 12, 108},
	} {
		t.Run(tc.file, func(t *testing.T) {
			first := activationFixturePlan(t, tc.file)
			if first.Capacity != 8 || first.RootTag != 0 || first.MaxFrameSize != tc.maxSize || first.MaxFrameAlign != 4 || first.SlotTagOffset != 0 || first.SlotDataOffset != 4 || first.SlotSize != tc.slotSize || first.SlotAlign != 4 || first.DepthOffset != 0 || first.DoneOffset != 4 || first.SlotsOffset != 12 || first.Size != tc.totalSize || first.Alignment != 4 {
				t.Fatalf("activation layout drift: %+v", first)
			}
			if len(first.Machines) != len(tc.names) {
				t.Fatalf("inventory: %+v", first.Machines)
			}
			for i, machine := range first.Machines {
				if machine.Name != tc.names[i] || machine.Tag != uint32(i) || machine.Size != tc.frameSizes[i] || machine.Alignment != 4 || machine.Fields[0].Name != "current_state" || machine.Fields[0].Offset != 0 {
					t.Fatalf("machine %d layout: %+v", i, machine)
				}
			}
			for i := 0; i < 100; i++ {
				again := activationFixturePlan(t, tc.file)
				if !reflect.DeepEqual(first, again) {
					t.Fatalf("nondeterministic layout run %d", i)
				}
			}
		})
	}
}

func TestEVT2x4RejectsOwnedActivationFieldBeforeNativeLayout(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", "machine_owned_child_cleanup.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	mir := buildMIR(module, env)
	_, err = planActivationStack(mir.Automata[0], env)
	if err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_ACTIVATION_FIELD_TYPE") {
		t.Fatalf("owned native frame accepted: %v", err)
	}
}

func TestEVT2x4RootActivationInitLIRAndMachineIR(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", "machine_parent_resume.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
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
	lir := LIRModule{Functions: []LIRFunction{init}}
	if err := VerifyLIR(lir); err != nil {
		t.Fatal(err)
	}
	if text := lir.String(); !strings.Contains(text, "activation-machine tag=0 Parent") || !strings.Contains(text, "activation-machine tag=1 Child") || !strings.Contains(text, "activation_address") {
		t.Fatalf("activation inventory or Init not inspectable:\n%s", text)
	}
	if init.Activation.InitFields[0].Offset != 0 || init.Activation.InitFields[1].Offset != 4 || init.Activation.InitFields[2].Offset != 8 || init.Activation.InitFields[3].Name != "root.tag" || init.Activation.InitFields[3].Offset != 12 || init.Activation.InitFields[4].Offset != 16 || init.Activation.InitFields[5].Offset != 20 {
		t.Fatalf("root init fields: %+v", init.Activation.InitFields)
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeMachineBridge(machine); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		again, err := lowerActivationRootInit(a, layout)
		if err != nil || (LIRModule{Functions: []LIRFunction{again}}).String() != lir.String() {
			t.Fatalf("nondeterministic Init LIR run %d: %v", i, err)
		}
	}
	broken := init
	broken.Blocks = append([]LIRBlock(nil), init.Blocks...)
	broken.Blocks[0].Instructions = append([]LIRInstruction(nil), init.Blocks[0].Instructions...)
	for i := range broken.Blocks[0].Instructions {
		if broken.Blocks[0].Instructions[i].Op == "activation_address" {
			broken.Blocks[0].Instructions[i].FrameOffset++
			break
		}
	}
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{broken}}); err == nil || !strings.Contains(err.Error(), "LIR_BAD_ACTIVATION_OFFSET") {
		t.Fatalf("malformed offset accepted: %v", err)
	}
	badGeometry := init
	meta := *init.Activation
	meta.Layout.SlotSize = 4
	badGeometry.Activation = &meta
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{badGeometry}}); err == nil || !strings.Contains(err.Error(), "LIR_BAD_ACTIVATION_FRAME") {
		t.Fatalf("undersized activation slot accepted: %v", err)
	}
	badDepth := init
	badDepth.Blocks = append([]LIRBlock(nil), init.Blocks...)
	badDepth.Blocks[0].Instructions = append([]LIRInstruction(nil), init.Blocks[0].Instructions...)
	for i := range badDepth.Blocks[0].Instructions {
		if badDepth.Blocks[0].Instructions[i].Op == "const" && badDepth.Blocks[0].Instructions[i].Literal == "1" {
			badDepth.Blocks[0].Instructions[i].Literal = "0"
			break
		}
	}
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{badDepth}}); err == nil || !strings.Contains(err.Error(), "LIR_BAD_ACTIVATION_INITIAL_VALUE") {
		t.Fatalf("zero live depth accepted: %v", err)
	}
	badTag := init
	badTag.Blocks = append([]LIRBlock(nil), init.Blocks...)
	badTag.Blocks[0].Instructions = append([]LIRInstruction(nil), init.Blocks[0].Instructions...)
	for i := range badTag.Blocks[0].Instructions {
		if badTag.Blocks[0].Instructions[i].Op == "const" && badTag.Blocks[0].Instructions[i].Literal == "0" && badTag.Blocks[0].Instructions[i].Type == "u32" {
			// The first u32 zero is the root state; a bad initial state must fail.
			badTag.Blocks[0].Instructions[i].Literal = "99"
			break
		}
	}
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{badTag}}); err == nil || !strings.Contains(err.Error(), "LIR_BAD_ACTIVATION_INITIAL_VALUE") {
		t.Fatalf("invalid root state accepted: %v", err)
	}
}
