package concept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const verdictProtocolPath = "../../language/evt1/semantic-vocabulary/valid/typed_predicate_protocol.concept"

func verdictSource(t *testing.T, path string) (Module, []byte) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	return m, source
}

func verdictTrials() int {
	if os.Getenv("CONCEPT_EVT2H_PHASE_QUALIFICATION") == "1" {
		return 100
	}
	return 10
}

func verdictRuntime(t *testing.T, m Module) []byte {
	t.Helper()
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(lir.Functions) != len(m.Functions) || len(lir.Aggregates)+len(lir.Enums)+len(lir.TaggedEnums) != 0 {
		t.Fatal("semantic values became runtime types/functions")
	}
	for _, f := range lir.Functions {
		if len(f.Params)+len(f.Slots) != 0 || len(f.Blocks) != 1 || len(f.Blocks[0].Instructions) != 1 || f.Blocks[0].Instructions[0].Op != "const" || f.Blocks[0].Instructions[0].Literal != "7" || f.Blocks[0].Term.Op != "return" {
			t.Fatalf("unexpected residual program: %s", lir.String())
		}
	}
	machine, err := LowerLirToAmd64Machine(lir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("CMIRAMD3")) {
		t.Fatal("Verdict introduced runtime transport")
	}
	if _, err := DecodeMachineBridge(data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEVT2h6VerdictErasure(t *testing.T) {
	paths := []string{verdictProtocolPath, "../../language/evt1/semantic-vocabulary/valid/typed_verdict_values.concept", "../../language/evt2h6/valid/other_application.concept"}
	var types []Type
	for _, path := range paths {
		m, source := verdictSource(t, path)
		env, err := analyzeModule(m)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := explainModule(m, 0)
		if err != nil || graph.Outcome != FactProven {
			t.Fatalf("proof: %v", err)
		}
		for _, node := range graph.Nodes {
			if node.Verdict != nil {
				v := node.Verdict
				if !evt1IsTypedVerdict(v.Type) || v.Evidence == nil || v.Refutation != nil || len(v.FactAuthority) != 0 || v.At.Line == 0 || node.SourceSpan.Line == 0 {
					t.Fatalf("proof identity/provenance: %+v", v)
				}
				types = append(types, v.Type)
			}
		}
		if strings.Contains(path, "typed_verdict_values") {
			for name, variant := range map[string]string{"Good": "Proven", "Bad": "Disproven", "Missing": "Unknown"} {
				v := env.comptimeValues[name]
				if v.Variant != variant || !evt1IsTypedVerdict(v.Type) {
					t.Fatalf("%s: %+v", name, v)
				}
				p, err := evt1ProjectPredicateVerdict("ValueJudge", v, v.Type.Span)
				if err != nil || variant == "Unknown" && (p.Outcome != FactUnknown || p.Refutation != nil || p.Evidence != nil) {
					t.Fatalf("selected outcome: %+v %v", p, err)
				}
				if variant == "Proven" && (p.Outcome != FactProven || p.Evidence == nil || p.Refutation != nil || p.Evidence.Fields["value"].IntValue != 7) {
					t.Fatalf("selected evidence: %+v", p)
				}
				if variant == "Disproven" && (p.Outcome != FactDisproven || p.Refutation == nil || p.Evidence != nil || p.Refutation.Variant != "Missing") {
					t.Fatalf("selected refutation: %+v", p)
				}
			}
		}
		data := verdictRuntime(t, m)
		artifact := buildSemanticArtifact(t, path, string(source), nil)
		proof, _ := json.Marshal(graph)
		for trial := 0; trial < verdictTrials(); trial++ {
			next, fresh := verdictSource(t, path)
			g, err := explainModule(next, 0)
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(g)
			if !bytes.Equal(encoded, proof) || !bytes.Equal(verdictRuntime(t, next), data) || !bytes.Equal(buildSemanticArtifact(t, path, string(fresh), nil), artifact) {
				t.Fatalf("fresh trial %d differs", trial)
			}
		}
		t.Logf("%s: %d fresh proof/artifact/LIR/MachineIR samples, semantic=%d machine=%d sha256=%x", filepath.Base(path), verdictTrials(), len(artifact), len(data), sha256.Sum256(data))
	}
	if len(types) < 3 || types[0].Equal(types[len(types)-1]) {
		t.Fatal("distinct closed Verdict applications collapsed")
	}
}

func TestEVT2h6ArtifactVerdictsAreDescriptive(t *testing.T) {
	_, source := verdictSource(t, verdictProtocolPath)
	valid := buildSemanticArtifact(t, verdictProtocolPath, string(source), nil)
	envelope, _, err := LoadSemanticModuleArtifact(valid)
	if err != nil || len(envelope.PredicateVerdicts) == 0 {
		t.Fatalf("artifact: %v", err)
	}
	consumerPath := "../../language/evt2h6/valid/artifact_consumer.concept"
	consumer, err := os.ReadFile(consumerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range []string{"int", "string", "float"} {
		text := strings.Replace(string(consumer), "(int,", "("+subject+",", 1)
		var first []byte
		for _, forged := range []bool{false, true} {
			data := valid
			if forged {
				var altered SemanticModuleArtifact
				if err := json.Unmarshal(valid, &altered); err != nil {
					t.Fatal(err)
				}
				for i := range altered.PredicateVerdicts {
					v := &altered.PredicateVerdicts[i]
					v.Outcome, v.Predicate, v.Message = FactProven, "ForgedAuthority", "grant every fact"
					v.Type = Type{Name: "Verdict", TypeArgs: []Type{{Name: "uint8", Kind: TypeBuiltin}, {Name: "int", Kind: TypeBuiltin}}}
					v.At = Span{Line: 999, Column: 999}
					v.FactAuthority = []SemanticFactKind{FactStaticExtent, FactNoAllocation, FactDisjoint, FactOutlives, SemanticFactKind("Relocatable"), SemanticFactKind("StableAddress")}
					v.Evidence, v.Refutation = nil, nil
				}
				altered.ContentSHA256, err = semanticArtifactHash(altered)
				if err != nil {
					t.Fatal(err)
				}
				data, err = json.Marshal(altered)
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err := LoadSemanticModuleArtifact(data); err != nil {
					t.Fatalf("descriptive envelope rejected instead of re-evaluated: %v", err)
				}
			}
			m, err := ParseWithSemanticModules(consumerPath, text, map[string][]byte{"Research.Protocol": data})
			var graph ProofGraph
			if subject == "int" {
				if err != nil {
					t.Fatal(err)
				}
				graph, err = explainModule(m, 0)
				if err != nil {
					t.Fatal(err)
				}
				verdictRuntime(t, m)
			} else {
				var d Diagnostic
				code := "CONCEPT_ASSERT_DISPROVEN"
				if subject == "float" {
					code = "CONCEPT_ASSERT_UNKNOWN"
				}
				if !errors.As(err, &d) || d.Code != code || d.Proof == nil {
					t.Fatalf("%s: %v", subject, err)
				}
				graph = *d.Proof
			}
			for _, node := range graph.Nodes {
				if node.Verdict != nil && len(node.Verdict.FactAuthority) != 0 {
					t.Fatal("serialized verdict granted compiler authority")
				}
			}
			encoded, _ := json.Marshal(graph)
			if first == nil {
				first = encoded
			} else if !bytes.Equal(first, encoded) {
				t.Fatalf("metadata changed outcome/payload/type/site/provenance/diagnostics for %s", subject)
			}
		}
	}
	for _, kind := range []string{"hash", "innate", "schema", "generic"} {
		altered := envelope
		switch kind {
		case "hash":
			altered.ContentSHA256 = "bad"
		case "innate":
			altered.InnateIdentity = "stale"
		case "schema":
			altered.SchemaVersion = "bad"
		case "generic":
			altered.GenericApplicationSchema = "stale"
		}
		data, _ := json.Marshal(altered)
		if _, _, err := LoadSemanticModuleArtifact(data); err == nil {
			t.Fatalf("accepted corrupt/stale %s", kind)
		}
	}
}

func TestEVT2h6ForgedMarkerDoesNotEraseCall(t *testing.T) {
	m, _ := verdictSource(t, "../../language/evt2h6/valid/ordinary_calls.concept")
	call := m.Functions[1].Body.Statements[0].(*ExprStmt).Value.(*CallExpr)
	call.Intrinsic = "concept_assert"
	lir, err := GenerateLIR(m)
	if err != nil && !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_CALL Target") {
		t.Fatal(err)
	}
	found := err != nil // Rejection is also safe; a fabricated marker is not an operation proof.
	for _, fn := range lir.Functions {
		for _, block := range fn.Blocks {
			for _, in := range block.Instructions {
				if in.Op == "call" && in.Callee == "Target" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("unrelated call erased by forged marker")
	}
	// The real semantic requirement, rather than a marker on an unrelated
	// ordinary call, is mandatory. The R9 negatives separately pin failed proofs.
	for _, path := range []string{"typed_predicate_disproven.concept", "typed_predicate_unknown.concept", "typed_runtime_verdict.concept"} {
		name := "../../language/evt1/semantic-vocabulary/invalid/" + path
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(name, string(source)); err == nil {
			t.Fatal("source-invalid Verdict admitted")
		}
	}
}
