package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type integrationSpecimen struct {
	name, path string
	module     Module
	source     []byte
	lir        LIRModule
	machine    MachineModule
	data       []byte
}

func integrationSamples() int {
	if os.Getenv("CONCEPT_EVT2H_PHASE_QUALIFICATION") == "1" {
		return 100
	}
	return 3
}
func integrationSpecimens(t *testing.T) []integrationSpecimen {
	t.Helper()
	var out []integrationSpecimen
	for _, name := range []string{"Parser", "Controller", "Assertions"} {
		path := "../../tests/runtime/Testing/Integration" + name + ".concept"
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, integrationChecked(t, name, path, m, source))
	}
	// Remove only the checked assertion from the authoritative source and run
	// a new parse/check. Keep whitespace so residual source sites stay equal.
	plainSource := bytes.Clone(out[2].source)
	start := strings.Index(string(plainSource), "Assert.Concept<WireWord>(int,")
	if start < 0 {
		t.Fatal("missing source assertion")
	}
	end := strings.Index(string(plainSource[start:]), ");")
	if end < 0 {
		t.Fatal("unterminated source assertion")
	}
	for i := start; i < start+end+2; i++ {
		if plainSource[i] != '\n' && plainSource[i] != '\r' {
			plainSource[i] = ' '
		}
	}
	plain, err := Parse(out[2].path, string(plainSource))
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Assertions) != 0 {
		t.Fatal("assertion-free source retained a proof obligation")
	}
	out = append(out, integrationChecked(t, "Plain", out[2].path, plain, plainSource))
	runtimeShape := func(input LIRModule) string {
		raw, _ := json.Marshal(input)
		var value LIRModule
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		value.PlanID = ""
		for i := range value.Functions {
			value.Functions[i].Decisions = nil
			for bi := range value.Functions[i].Blocks {
				for ii := range value.Functions[i].Blocks[bi].Instructions {
					value.Functions[i].Blocks[bi].Instructions[ii].Decision = ""
				}
			}
		}
		return value.String()
	}
	if runtimeShape(out[2].lir) != runtimeShape(out[3].lir) {
		t.Fatal("assertion changed residual runtime instructions or types")
	}

	return out
}
func integrationChecked(t *testing.T, name, path string, m Module, source []byte) integrationSpecimen {
	t.Helper()
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(name, err)
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(name, err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if machine.Fallible == nil {
		t.Fatal("expected existing v10 composition")
	}
	return integrationSpecimen{name, path, m, source, lir, machine, data}
}
func TestEVT2h8CheckedIntegration(t *testing.T) {
	want := map[string]string{
		"Parser":     "a22f7c6e20cd6c15210dd4929be40da9da1285d304aa4a91eed70e57f00a1951",
		"Controller": "7100448fb789e1234041110111352374dd0550c97bd2cc57bdac1eb2fba9c73e",
		"Assertions": "3be840ce69436a865e69c16d3fba81da5e73356904c2ef12fcb4301e457ca84f",
		"Plain":      "2679883df393fa6f612b99aaa2d6cd0a8a0b9696519277de6ec8d7203ceb3dcb",
	}
	for _, s := range integrationSpecimens(t) {
		if got := fmt.Sprintf("%x", sha256.Sum256(s.data)); got != want[s.name] {
			t.Fatalf("%s canonical artifact changed: %s", s.name, got)
		}
		decoded, err := DecodeMachineBridge(s.data)
		if err != nil || decoded.String() != s.machine.String() {
			t.Fatal(s.name, err)
		}
		if strings.Contains(s.lir.String(), "Verdict") || strings.Contains(s.lir.String(), "WordEvidence") {
			t.Fatal("compile-time semantic value escaped")
		}
		if s.name == "Parser" {
			// The independent C callers use these checked carriers. Pin their
			// geometry explicitly; byte width alone grants no nominal authority.
			carriers := 0
			for _, domain := range s.lir.TaggedEnums {
				if domain.Name == "ParsedToken" || strings.Contains(domain.Identity, "\"name\":\"Option\"") {
					carriers++
					if domain.Size != 8 || domain.Alignment != 4 || domain.PayloadOffset != 4 {
						t.Fatal("ABI probe requires checked eight-byte sum geometry")
					}
				}
			}
			if carriers != 2 {
				t.Fatal("missing checked ABI carrier domains")
			}
		}
		t.Logf("%s functions=%d records=%d sums=%d enums=%d artifact=%d sha256=%x", s.name, len(s.lir.Functions), len(s.lir.Aggregates), len(s.lir.TaggedEnums), len(s.lir.Enums), len(s.data), sha256.Sum256(s.data))
	}
}
func TestEVT2h8FreshIntegration(t *testing.T) {
	start := time.Now()
	first := integrationSpecimens(t)
	for trial := 0; trial < integrationSamples(); trial++ {
		for i, s := range integrationSpecimens(t) {
			if !bytes.Equal(s.data, first[i].data) || s.lir.String() != first[i].lir.String() || s.machine.String() != first[i].machine.String() {
				t.Fatalf("fresh %s trial %d differs", s.name, trial)
			}
		}
	}
	t.Logf("%d fresh source/LIR/CFG/artifact generations for four checked forms: %s", integrationSamples(), time.Since(start))
}
func TestEVT2h8ProofAuthority(t *testing.T) {
	s := integrationSpecimens(t)[2]
	graph, err := explainModule(s.module, 0)
	if err != nil || graph.Outcome != FactProven {
		t.Fatal(err)
	}
	selected := 0
	for _, node := range graph.Nodes {
		if v := node.Verdict; v != nil {
			selected++
			if v.Evidence == nil || v.Evidence.Fields["capacity"].IntValue != 8 || v.Refutation != nil || len(v.FactAuthority) != 0 || v.At.Line == 0 || node.SourceSpan.Line == 0 {
				t.Fatalf("typed proof/provenance: %+v", v)
			}
		}
	}
	if selected == 0 {
		t.Fatal("missing selected evidence")
	}
	invalid := strings.Replace(string(s.source), "Assert.Concept<WireWord>(int,", "Assert.Concept<WireWord>(bool,", 1)
	if _, err := Parse(s.path, invalid); err == nil {
		t.Fatal("accepted disproven assertion")
	} else {
		t.Log(err)
	}
	// A caller cannot forge the erasure marker on an arbitrary ordinary call.
	m, err := Parse(s.path, string(s.source))
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Functions {
		for _, st := range m.Functions[i].Body.Statements {
			if expr, ok := st.(*ExprStmt); ok {
				if call, ok := expr.Value.(*CallExpr); ok && call.ConceptGoal != "" {
					call.ConceptGoal = ""
				}
			}
		}
	}
	if _, err := GenerateLIR(m); err == nil {
		t.Fatal("forged assertion marker erased without a goal")
	}
	t.Log("typed evidence/site/provenance retained; no trusted fact authority; invalid and forged assertions reject")
}
func TestEVT2h8LIRMutations(t *testing.T) {
	base := integrationSpecimens(t)[1].lir
	raw, _ := json.Marshal(base)
	owner := func(m *LIRModule) *LIRFunction {
		for i := range m.Functions {
			if m.Functions[i].Name == "GateAnd" {
				return &m.Functions[i]
			}
		}
		t.Fatal("GateAnd")
		return nil
	}
	for name, mutate := range map[string]func(*LIRModule){
		"uninitialized logical join": func(m *LIRModule) {
			f := owner(m)
			for bi := range f.Blocks {
				for ii, in := range f.Blocks[bi].Instructions {
					if (in.Op == "store_slot" || in.Op == "initialize_slot") && strings.HasPrefix(f.Slots[in.Slot].Name, "$logical") {
						f.Blocks[bi].Instructions = append(f.Blocks[bi].Instructions[:ii], f.Blocks[bi].Instructions[ii+1:]...)
						return
					}
				}
			}
			t.Fatal("logical initialization")
		},
		"invalid successor": func(m *LIRModule) { f := owner(m); f.Blocks[0].Term.True = 999 },
		"inactive binding": func(m *LIRModule) {
			f := owner(m)
			p := f.Patterns[0]
			f.Blocks[p.Arms[0].Body].Instructions[1].Member = 1
		},
		"missing failure authority":   func(m *LIRModule) { owner(m).Fallible = nil },
		"nondominating logical input": func(m *LIRModule) { f := owner(m); f.Blocks[0].Term.Value = 9999 },
	} {
		t.Run(name, func(t *testing.T) {
			var next LIRModule
			if err := json.Unmarshal(raw, &next); err != nil {
				t.Fatal(err)
			}
			mutate(&next)
			if err := VerifyLIR(next); err == nil {
				t.Fatal("accepted malformed integration CFG")
			} else {
				t.Log(err)
			}
		})
	}
}
