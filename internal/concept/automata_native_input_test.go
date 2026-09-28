package concept

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// EVT2 must lower the validated state body and frame initializer, not reverse
// engineer them from MIR summary text or the C backend's emitted source.
func TestEVT2xMachineSemanticInputSurvivesMIR(t *testing.T) {
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
	mir := buildMIR(module, env)
	if err := evt1ValidateMIR(mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Automata) != 1 || len(mir.Automata[0].Machines) != 2 {
		t.Fatalf("machine hierarchy lost: %+v", mir.Automata)
	}
	parent := mir.Automata[0].Machines[0]
	if parent.Fields[0].Initializer == nil || parent.Fields[0].Classification != "MachinePersistent" {
		t.Fatalf("persistent field initializer lost: %+v", parent.Fields)
	}
	start := parent.States[0]
	if start.SemanticBody == nil || len(start.SemanticBody.Statements) != 1 {
		t.Fatalf("state body lost: %+v", start)
	}
	if _, ok := start.SemanticBody.Statements[0].(*PushMachineStmt); !ok {
		t.Fatalf("push became summary-only: %T", start.SemanticBody.Statements[0])
	}
	resumed := parent.States[1]
	if resumed.SemanticBody == nil || len(resumed.SemanticBody.Statements) != 1 {
		t.Fatalf("resume body lost: %+v", resumed)
	}
	if _, ok := resumed.SemanticBody.Statements[0].(*AssignStmt); !ok {
		t.Fatalf("persistent state write became summary-only: %T", resumed.SemanticBody.Statements[0])
	}
	// Checked MIR artifacts remain the existing summary contract.
	encoded, err := json.Marshal(mir)
	if err != nil {
		t.Fatal(err)
	}
	var checked MIR
	if err := json.Unmarshal(encoded, &checked); err != nil {
		t.Fatal(err)
	}
	if checked.Automata[0].Machines[0].States[0].SemanticBody != nil || checked.Automata[0].Machines[0].Fields[0].Initializer != nil {
		t.Fatal("in-memory lowering input leaked into checked MIR JSON")
	}
	// With no ordinary function present, the old lowering returned an empty,
	// verified module despite the executable automata declaration.
	automataOnly := module
	automataOnly.Functions = nil
	if _, err := GenerateLIR(automataOnly); err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_AUTOMATA_LOWERING Worker") {
		t.Fatalf("native LIR silently omitted automata: %v", err)
	}
}
