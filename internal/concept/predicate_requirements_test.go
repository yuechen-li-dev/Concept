package concept

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func r9aPredicateSource(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile("../../language/evt1/generic-library-closure/valid/predicate_requirements.concept")
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestR9aDeclaredPredicateArtifactAndNative(t *testing.T) {
	producer := r9aPredicateSource(t)
	body := buildSemanticArtifact(t, "predicates.concept", producer, nil)
	consumer := `module Closure.PredicateUse; profile Core; import Closure.Predicates;
int Use() { Assert.Concept<IntegerOnly>(int, "artifact type predicate"); Assert.Concept<Small>(Good, "artifact declaration predicate"); return Identity<int>(7); }
`
	module, err := ParseWithSemanticModules("predicate_use.concept", consumer, map[string][]byte{"Closure.Predicates": body})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "predicate_harness.c", "#include \"predicate_use.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Use")+"() == 7 ? 0 : 1; }\n")
	if !strings.Contains(string(outputs["predicate_use.mir.json"]), `"kind": "predicate"`) {
		t.Fatal("MIR lost predicate requirement structure")
	}
	bad := strings.Replace(consumer, "Identity<int>(7)", "Identity<bool>(true)", 1)
	_, err = ParseWithSemanticModules("bad.concept", bad, map[string][]byte{"Closure.Predicates": body})
	if err == nil || !strings.Contains(err.Error(), "PREDICATE_REQUIREMENT_UNSATISFIED") {
		t.Fatalf("closed predicate was ignored: %v", err)
	}
}

func TestR9aDeclaredPredicateProofOutcomesAndAuthority(t *testing.T) {
	source := r9aPredicateSource(t) + "\ncomptime bool PastEnd(declaration fn) { return compiler.TypeName(compiler.ParameterType(fn, 99)) == \"int\"; }\nconcept Unknown<declaration F> { requires PastEnd(F); }\nstruct Record { int number; bool flag; }\n"
	module, err := Parse("predicates.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	before := len(env.semanticProofs)
	for _, tc := range []struct {
		concept, name string
		want          SemanticFactCertainty
	}{
		{"Small", "Good", FactProven}, {"Small", "Many", FactDisproven}, {"Unknown", "Good", FactUnknown}, {"IntegerField", "number", FactProven}, {"IntegerField", "flag", FactDisproven},
	} {
		var subject *DeclarationSubject
		for i := range env.declarationSubjects {
			if env.declarationSubjects[i].Name == tc.name {
				subject = &env.declarationSubjects[i]
				break
			}
		}
		if subject == nil {
			t.Fatalf("missing subject %s", tc.name)
		}
		graph, err := buildPolicyProof(env, LintPolicy{Identity: tc.concept}, *subject)
		if err != nil || graph.Outcome != tc.want {
			t.Fatalf("%s(%s): graph=%+v err=%v", tc.concept, tc.name, graph, err)
		}
		found := false
		for _, node := range graph.Nodes {
			if strings.HasPrefix(node.Label, "predicate ") {
				found = true
				if node.Origin != FactOriginDeclared {
					t.Fatalf("predicate granted compiler authority: %+v", node)
				}
			}
		}
		if !found {
			t.Fatal("proof omitted predicate requirement")
		}
		if tc.want == FactUnknown && !strings.Contains(RenderProofVerbose(graph), "out of bounds") {
			t.Fatal("unknown lost the concrete observation failure")
		}
	}
	if len(env.semanticProofs) != before {
		t.Fatal("predicate evaluation minted semantic facts")
	}
}

func TestR9aPredicateProjectPolicyLexicalEnvironment(t *testing.T) {
	root := t.TempDir()
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
comptime bool CountIsSmall(declaration function) { return compiler.ParameterCount(function) <= 1; }
comptime bool IsSmall(declaration function) { return CountIsSmall(function); }
concept HotPath<declaration F> { requires IsSmall(F); requires compiler.NoAllocation(F); }
comptime LintPolicy Hot = LintPolicy{"HotPath", "error", "FunctionDeclaration", ""};
`
	program := `module Policy.Program; profile Core;
int Good(int value) { return value; }
int Many(int first, int second) { return first+second; }
int IsSmall(int value) { return value; }
`
	manifestPath, programPath := filepath.Join(root, "manifest.concept"), filepath.Join(root, "program.concept")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(programPath, []byte(program), 0644); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(root, nil)
	if err != nil || len(findings) != 1 || findings[0].Name != "Many" || findings[0].Outcome != FactDisproven {
		t.Fatalf("policy predicates: findings=%+v err=%v", findings, err)
	}
	graph, err := ExplainPolicy(programPath, "HotPath", "Many", nil)
	if err != nil || graph.Outcome != FactDisproven || !strings.Contains(RenderProofVerbose(graph), "IsSmall returned false") {
		t.Fatalf("policy explain: %+v %v", graph, err)
	}
}

func TestR9aDeclaredPredicateDiagnostics(t *testing.T) {
	for _, tc := range []struct{ file, code string }{{"predicate_false", "PREDICATE_REQUIREMENT_UNSATISFIED"}, {"predicate_bad_result", "PREDICATE_REQUIREMENT_INVALID"}} {
		body, err := os.ReadFile("../../language/evt1/generic-library-closure/invalid/" + tc.file + ".concept")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(tc.file+".concept", string(body))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
			t.Fatalf("%s: %v", tc.file, err)
		}
	}
}
