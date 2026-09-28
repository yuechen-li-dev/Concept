package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func evt2Fixture(t *testing.T) Module {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt2", "valid", "core.concept")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(body))
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestEVT2CoreLIRPlannerAndDeterminism(t *testing.T) {
	module := evt2Fixture(t)
	first, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyLIR(first); err != nil {
		t.Fatal(err)
	}
	if len(first.Functions) != 7 {
		t.Fatalf("got %d functions", len(first.Functions))
	}
	for _, f := range first.Functions {
		if f.Identity == "" || len(f.Blocks) == 0 {
			t.Fatalf("incomplete function %s", f.Name)
		}
	}
	text := first.String()
	for _, want := range []string{"fn Add", "checked_add i32", "fn Max", "branch v", "fn Sum4", "const i32 4", "lt bool", "check_index void", "index_address ptr<i32>", "load i32", "fn CheckedIndex", "fn StoreIndex", "store i32", "fn Choose", "fn Early"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in LIR:\n%s", want, text)
		}
	}
	if len(first.Functions[2].Blocks) < 4 {
		t.Fatal("range did not lower to CFG")
	}
	if !strings.Contains(text, "PerAccessRuntime") {
		t.Fatal("LIR lost the existing Planner bounds decision")
	}
	for i := 0; i < 100; i++ {
		next, err := GenerateLIR(module)
		if err != nil {
			t.Fatal(err)
		}
		if next.String() != text {
			t.Fatalf("nondeterministic LIR on run %d", i+2)
		}
	}
}

func TestEVT2VerifierRejectsMalformedLIR(t *testing.T) {
	base := LIRModule{Functions: []LIRFunction{{Identity: "F|F(int)", Name: "F", Params: []LIRValue{{ID: 0, Type: "i32"}}, Result: "i32", Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{{Op: "const", Result: 1, Type: "i32", Slot: -1, Literal: "1"}, {Op: "checked_add", Result: 2, Type: "i32", Slot: -1, Args: []int{0, 1}}}, Term: LIRTerminator{Op: "return", Value: 2}}}}}}
	if err := VerifyLIR(base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*LIRModule)
	}{
		{"undefined", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[1].Args[1] = 99 }},
		{"duplicate", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[1].Result = 1 }},
		{"operand type", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[0].Type = "bool" }},
		{"missing terminator", func(m *LIRModule) { m.Functions[0].Blocks[0].Term.Op = "" }},
		{"wrong return", func(m *LIRModule) { m.Functions[0].Blocks[0].Term.Value = 99 }},
		{"invalid branch", func(m *LIRModule) {
			m.Functions[0].Blocks[0].Term = LIRTerminator{Op: "branch", Value: 1, True: 2, False: 0}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := LIRModule{Functions: []LIRFunction{base.Functions[0]}}
			m.Functions[0].Blocks = append([]LIRBlock(nil), base.Functions[0].Blocks...)
			m.Functions[0].Blocks[0].Instructions = append([]LIRInstruction(nil), base.Functions[0].Blocks[0].Instructions...)
			for i := range m.Functions[0].Blocks[0].Instructions {
				m.Functions[0].Blocks[0].Instructions[i].Args = append([]int(nil), m.Functions[0].Blocks[0].Instructions[i].Args...)
			}
			tc.mutate(&m)
			if err := VerifyLIR(m); err == nil {
				t.Fatal("malformed LIR accepted")
			}
		})
	}
}

func TestEVT2UnsupportedFeatureFailsExplicitly(t *testing.T) {
	module, err := Parse("float.concept", "profile Core; float F(float x) { return x; }")
	if err != nil {
		t.Fatal(err)
	}
	_, err = GenerateLIR(module)
	if err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_TYPE") {
		t.Fatalf("wanted explicit unsupported type, got %v", err)
	}
}

func TestEVT2VerifierRequiresIndexedGuardAndCoherentLayout(t *testing.T) {
	f := LIRFunction{Identity: "Index|Index(int[4], int)", Name: "Index", Params: []LIRValue{{ID: 0, Type: "[4]i32"}, {ID: 1, Type: "i32"}}, Result: "i32", Slots: []LIRSlot{{ID: 0, Name: "values", Type: "[4]i32", Element: "i32", Extent: 4, ParamValue: 0}}, Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{{Op: "index_address", Result: 2, Type: "ptr<i32>", Args: []int{1}, Slot: 0, Extent: 4, Stride: 4}, {Op: "load", Result: 3, Type: "i32", Args: []int{2}, Slot: -1}}, Term: LIRTerminator{Op: "return", Value: 3}}}}
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{f}}); err == nil || !strings.Contains(err.Error(), "LIR_MISSING_INDEX_GUARD") {
		t.Fatalf("missing guard accepted: %v", err)
	}
	f.Blocks[0].Instructions = append([]LIRInstruction{{Op: "check_index", Result: -1, Type: "void", Args: []int{1}, Slot: -1, Extent: 4}}, f.Blocks[0].Instructions...)
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{f}}); err != nil {
		t.Fatal(err)
	}
	f.Blocks[0].Instructions[1].Stride = 8
	if err := VerifyLIR(LIRModule{Functions: []LIRFunction{f}}); err == nil || !strings.Contains(err.Error(), "LIR_BAD_INDEX_ADDRESS") {
		t.Fatalf("bad stride accepted: %v", err)
	}
}
