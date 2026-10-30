package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func matchFixture(t *testing.T, ordinary bool) (Module, []byte, LIRModule) {
	t.Helper()
	path := "../../tests/runtime/Testing/TaggedMatches.concept"
	if ordinary {
		path = "../../tests/runtime/Testing/OrdinaryMatches.concept"
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../tests/runtime"})
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	return m, source, lir
}

func TestEVT2h3SourceDiagnostics(t *testing.T) {
	codes := map[string]string{"arity": "CV4111", "binding_name": "CV4024", "duplicate": "CV4113", "enum_wildcard": "CV4109", "escape": "CV4114", "expr_block": "CV4117", "guard_default": "MATCH_GUARD_OTHERWISE", "guard_order": "MATCH_GUARD_ORDER", "guard_type": "CV4186", "integer_unreachable": "CV4113", "integer_wildcard": "CV4115", "missing": "CV4115", "nominal": "CV4109", "result": "CV4116", "stmt_expression": "CV4118", "unit_binding": "CV4112", "unknown": "CV4110", "variant_guard": "CV4020"}
	paths, err := filepath.Glob("../../language/evt2h3/invalid/*.concept")
	if err != nil || len(paths) != 18 {
		t.Fatalf("specimens %d: %v", len(paths), err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(path, string(source))
			code := codes[strings.TrimSuffix(filepath.Base(path), ".concept")]
			if err == nil || code == "" || !strings.Contains(err.Error(), code) {
				t.Fatalf("want %s, got %v", code, err)
			}
			t.Log(err)
		})
	}
	// The source-only h2 boundary is now a native h3 regression.
	path := "../../language/evt2h2/unsupported/match.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = LowerLirToAmd64Machine(lir); err != nil {
		t.Fatal(err)
	}
}
func TestEVT2h3CheckedLIR(t *testing.T) {
	for _, ordinary := range []bool{true, false} {
		_, _, lir := matchFixture(t, ordinary)
		machine, err := LowerLirToAmd64Machine(lir)
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeMachineBridge(machine)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = DecodeMachineBridge(data); err != nil {
			t.Fatal(err)
		}
		flows, tests, reads := 0, 0, 0
		for _, f := range lir.Functions {
			flows += len(f.Patterns)
			for _, b := range f.Blocks {
				for _, in := range b.Instructions {
					if in.Op == "test_variant" {
						tests++
					}
					if in.Op == "checked_payload" {
						reads++
					}
				}
			}
		}
		if flows == 0 || tests == 0 || !ordinary && reads == 0 {
			t.Fatal("missing checked flow")
		}
		t.Logf("ordinary=%t functions=%d matches=%d tests=%d reads=%d", ordinary, len(lir.Functions), flows, tests, reads)
	}
}

func TestEVT2h3JoinDefiniteInitialization(t *testing.T) {
	_, _, lir := matchFixture(t, false)
	for _, f := range lir.Functions {
		if len(f.Patterns) == 0 || f.Patterns[0].ResultSlot < 0 {
			continue
		}
		result := f.Patterns[0].ResultSlot
		for bi := range f.Blocks {
			for ii, in := range f.Blocks[bi].Instructions {
				if in.Op == "initialize_slot" && in.Slot == result {
					f.Blocks[bi].Instructions = append(f.Blocks[bi].Instructions[:ii:ii], f.Blocks[bi].Instructions[ii+1:]...)
					if err := verifyPatternSlotInitialization(f); err == nil || !strings.Contains(err.Error(), "UNINITIALIZED_JOIN") {
						t.Fatalf("missing predecessor result: %v", err)
					}
					return
				}
			}
		}
	}
	t.Fatal("no expression result")
}

func TestEVT2h3RejectCrossArmEntry(t *testing.T) {
	path := "../../language/evt2h3/valid/arm_regions.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	f := &lir.Functions[0]
	p := f.Patterns[0]
	// Bypass the other arm's discriminant and enter its inner if block.
	f.Blocks[p.Arms[1].Body].Term.True = f.Blocks[p.Arms[0].Body].Term.True
	if err := VerifyLIR(lir); err == nil {
		t.Fatal("accepted cross-arm entry into nested control flow")
	}
}
