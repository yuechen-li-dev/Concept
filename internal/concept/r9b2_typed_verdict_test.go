package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const r9b2Protocol = `module Research.Protocol; profile Core;
template <typename T> record struct Box { T value; }
enum Failure { Unsupported(typename subject), Missing }
template <typename T> comptime T Identity(T value) { return value; }
comptime Verdict<Box<uint8>, Failure> Judge(typename subject) {
    return match {
        when compiler.TypeName(subject) == "int" => Verdict::Proven(Identity<Box<uint8>>(Box<uint8>{7})),
        when compiler.TypeName(subject) == "float" => Verdict::Unknown,
        otherwise => Verdict::Disproven(Failure::Unsupported(subject)),
    };
}
comptime string JudgeDescribe(Failure failure) {
    return match (failure) {
        Failure::Unsupported(subject) => "unsupported type " + compiler.TypeName(subject),
        Failure::Missing => "missing evidence",
    };
}
concept Supported<T> { requires Judge(T); }
int Use() { Assert.Concept<Supported>(int, "typed proof"); return 7; }
`

func TestR9b2TypedProtocol(t *testing.T) {
	for _, tc := range []struct{ subject, code, detail string }{
		{"int", "", ""}, {"string", "CONCEPT_ASSERT_DISPROVEN", "unsupported type string"}, {"float", "CONCEPT_ASSERT_UNKNOWN", "evidence unavailable"},
	} {
		source := strings.Replace(r9b2Protocol, "(int, \"typed proof\")", "("+tc.subject+", \"typed proof\")", 1)
		module, err := Parse("protocol.concept", source)
		if tc.code != "" {
			var d Diagnostic
			if !errors.As(err, &d) || d.Code != tc.code || !strings.Contains(d.Message, tc.detail) || d.Proof == nil {
				t.Fatalf("%s: %v", tc.subject, err)
			}
			found := false
			for _, node := range d.Proof.Nodes {
				if node.Verdict != nil {
					found = true
					if string(node.Verdict.Outcome) != string(node.Outcome) {
						t.Fatal("second truth lattice")
					}
				}
			}
			if !found {
				t.Fatal("typed metadata missing")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarness(t, outputs, "protocol_harness.c", "#include \"protocol.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Use")+"() == 7 ? 0 : 1; }\n")
		for name, body := range outputs {
			if strings.HasSuffix(name, ".generated.c") && bytes.Contains(body, []byte("concept_template_Identity")) {
				t.Fatal("comptime template leaked into C")
			}
		}
	}
}

func TestR9b2TypedProtocolArtifact(t *testing.T) {
	artifact := buildSemanticArtifact(t, "protocol.concept", r9b2Protocol, nil)
	consumer := `module Research.Consumer; profile Core; import Research.Protocol;
int Main() { Assert.Concept<Supported>(int, "artifact predicate"); return Use(); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Research.Protocol": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "consumer_harness.c", "#include \"consumer.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Main")+"() == 7 ? 0 : 1; }\n")
	envelope, _, err := LoadSemanticModuleArtifact(artifact)
	if err != nil || len(envelope.PredicateVerdicts) != 1 || envelope.PredicateVerdicts[0].Evidence == nil {
		t.Fatalf("inspectable payload: %+v, %v", envelope.PredicateVerdicts, err)
	}
	for _, tc := range []struct{ subject, code, payload string }{{"string", "CONCEPT_ASSERT_DISPROVEN", "Unsupported"}, {"float", "CONCEPT_ASSERT_UNKNOWN", "Unknown"}} {
		_, err := ParseWithSemanticModules("consumer.concept", strings.Replace(consumer, "(int,", "("+tc.subject+",", 1), map[string][]byte{"Research.Protocol": artifact})
		var d Diagnostic
		if !errors.As(err, &d) || d.Code != tc.code || d.Proof == nil || !strings.Contains(RenderProofVerbose(*d.Proof), tc.payload) {
			t.Fatalf("artifact %s: %v", tc.subject, err)
		}
	}
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "protocol.concept", r9b2Protocol, nil); !bytes.Equal(got, artifact) {
			t.Fatalf("artifact changed at %d", run)
		}
	}
	var data map[string]any
	if err := json.Unmarshal(artifact, &data); err != nil {
		t.Fatal(err)
	}
	t.Logf("typed artifact bytes=%d", len(artifact))
}

func TestR9b2TypedProjectPolicy(t *testing.T) {
	root := t.TempDir()
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
enum Evidence { Valid }
enum NamingFailure { Rename(declaration subject) }
comptime Verdict<Evidence, NamingFailure> JudgeName(declaration subject) {
    return match { when compiler.Name(subject) == "Good" => Verdict::Proven(Evidence::Valid), otherwise => Verdict::Disproven(NamingFailure::Rename(subject)), };
}
comptime string JudgeNameDescribe(NamingFailure failure) { return match (failure) { NamingFailure::Rename(subject) => "rename " + compiler.Name(subject), }; }
concept ProjectNaming<declaration D> { requires JudgeName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "warning", "FunctionDeclaration", ""};`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "program.concept"), []byte(`module Policy.Program; profile Core; void Good() {} void Bad() {}`), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(root, nil)
	if err != nil || len(findings) != 1 || findings[0].Severity != LintWarning || findings[0].Outcome != FactDisproven || !strings.Contains(findings[0].Message, "rename Bad") || !strings.Contains(findings[0].Message, "Rename") {
		t.Fatalf("typed policy: %+v, %v", findings, err)
	}
	graph, err := ExplainPolicy(filepath.Join(root, "program.concept"), "ProjectNaming", "Bad", nil)
	if err != nil || !strings.Contains(RenderProofVerbose(graph), "refutation:") {
		t.Fatalf("policy explain: %+v %v", graph, err)
	}
	baseline, _ := json.Marshal(graph)
	for run := 0; run < 100; run++ {
		again, err := ExplainPolicy(filepath.Join(root, "program.concept"), "ProjectNaming", "Bad", nil)
		encoded, _ := json.Marshal(again)
		if err != nil || !bytes.Equal(encoded, baseline) {
			t.Fatalf("policy drift at %d: %v", run, err)
		}
	}
}

func TestR9b2TypedVerdictRestrictions(t *testing.T) {
	for _, tc := range []struct{ source, code string }{
		{`enum E { Valid } ref struct R { ref int field; } comptime Verdict<E,R> Judge(typename t) { return Verdict::Unknown; }`, "VERDICT_PAYLOAD_INVALID"},
		{`template <typename T> comptime int Id(int value) { return value; } comptime int Open(typename subject) { return Id<subject>(1); }`, "COMPTIME_TEMPLATE_CLOSURE_REQUIRED"},
		{`enum E { Valid } int Bad(Verdict<E,int> verdict) { return 0; }`, "COMPTIME_ONLY_TYPE"},
		{`enum E { Valid } comptime Verdict<E,int> Judge(typename t) { return Verdict::Disproven("wrong"); }`, "CV4107"},
		{`enum E { Valid } comptime Verdict<E,int> Judge(typename t) { return Verdict::Proven(E::Valid); } comptime int JudgeDescribe(int r) { return r; } concept C<T> { requires Judge(T); }`, "VERDICT_DESCRIBE_INVALID"},
		{`template <typename T> comptime int Read(T value) { return Native(); } int Native() { return 1; } comptime int n = Read<int>(1);`, "CV4210"},
		{`template <typename T> comptime int Loop(T value) { while (true) { } return 0; } comptime int n = Loop<int>(1);`, "CV4205"},
		{`enum E { Valid } comptime Verdict<E,string> Judge(typename t) { return Verdict::Disproven("` + strings.Repeat("x", evt1ComptimeMaxStringBytes+1) + `"); } concept C<T> { requires Judge(T); } int Use() { Assert.Concept<C>(int, "bounded payload"); return 0; }`, "VERDICT_PAYLOAD_LIMIT"},
	} {
		_, err := Parse("restriction.concept", "module Research.Restriction; profile Core;\n"+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s: %v", tc.code, err)
		}
	}
	// Open execution is rejected by the evaluator itself, including callers
	// that have no source/body-validation context.
	module, err := Parse("protocol.concept", r9b2Protocol)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	_, err = evt1EvalExpr(newEVT1ComptimeState(env), evt1SeedComptimeScope(env), &TemplateCallExpr{Callee: "Identity", TypeArg: Type{Name: "T", Kind: TypeConceptParam}})
	if err == nil || !strings.Contains(err.Error(), "COMPTIME_TEMPLATE_CLOSURE_REQUIRED") {
		t.Fatalf("open execution: %v", err)
	}
}

func TestR9b2TypedProvenCannotAuthorizeFacts(t *testing.T) {
	source := `module Research.Claims; profile Core;
enum E { Valid } enum R { Invalid }
comptime Verdict<E,R> Claim(typename subject) { return Verdict::Proven(E::Valid); }
concept ClaimExtent<T> { requires Claim(T); }
concept ClaimNoAllocation<T> { requires Claim(T); }
concept ClaimOutlives<T> { requires Claim(T); }
concept ClaimDisjoint<T> { requires Claim(T); }
int Use(Span<int> input) {
    Assert.Concept<ClaimExtent>(input, "descriptive extent");
    Assert.Concept<ClaimNoAllocation>(input, "descriptive allocation");
    Assert.Concept<ClaimOutlives>(input, "descriptive lifetime");
    Assert.Concept<ClaimDisjoint>(input, "descriptive aliasing"); return 0;
}
`
	module, err := Parse("claims.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["claims.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactStaticExtent || fact.Kind == FactOutlives || fact.Kind == FactDisjoint {
			t.Fatalf("forged structural fact: %+v", fact)
		}
	}
	for _, proof := range mir.SemanticProofs {
		for _, verdict := range proof.Verdicts {
			if len(verdict.FactAuthority) != 0 {
				t.Fatal("user payload elevated authority")
			}
		}
	}
	if len(evt1InnateFactAuthority) != 0 {
		t.Fatal("R9b2 migrated fact generation")
	}
	if kinds := evt1TrustedPredicateFactKinds(&semanticEnv{}, ConceptDecl{Name: "DroppableFieldIsOwned", Innate: true}); len(kinds) != 0 {
		t.Fatal("source spelling granted authority")
	}
	_, err = Parse("claims.concept", source+`int Bad(Span<int> input) { Assert.Concept<ClaimExtent>(input, "claimed"); Assert.Concept<StaticExtent<4>>(input, "actual layout unknown"); return 0; }`)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("claim forged actual extent: %v", err)
	}
	allocation := `profile Core; enum E { Valid } enum R { Invalid }
comptime Verdict<E,R> Claim(declaration subject) { return Verdict::Proven(E::Valid); }
concept ClaimNoAllocation<declaration D> { requires Claim(D); }
extern "C" byte* Acquire(usize size); requires compiler.Allocates(Acquire);
int Allocating(usize size) { byte* storage = Acquire(size); return 1; }
void Verify() { Assert.Concept<ClaimNoAllocation>(Allocating, "user claim"); Assert.Concept<NoAllocation>(Allocating, "actual authority"); }
`
	_, err = Parse("allocation_claim.concept", allocation)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") || !strings.Contains(err.Error(), "Acquire Allocates") {
		t.Fatalf("typed claim forged NoAllocation: %v", err)
	}
}

func TestR9b2ClosedTemplateBudget(t *testing.T) {
	module, err := Parse("protocol.concept", r9b2Protocol)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	intType, _ := evt1BuiltinType("int", Span{})
	instance, err := instantiateTemplate(env, "Identity", intType, Span{})
	if err != nil {
		t.Fatal(err)
	}
	args := []Value{{Kind: ValueInt, Type: intType, IntValue: 7}}
	state := newEVT1ComptimeState(env)
	state.fuel = 0
	_, err = evt1InvokeClosedTemplateValues(state, instance, args, Span{})
	if err == nil || !strings.Contains(err.Error(), "CV4204") {
		t.Fatalf("template reset fuel: %v", err)
	}
	state = newEVT1ComptimeState(env)
	for i := 0; i < evt1ComptimeMaxCallDepth; i++ {
		if err := state.push("caller"); err != nil {
			t.Fatal(err)
		}
	}
	_, err = evt1InvokeClosedTemplateValues(state, instance, args, Span{})
	if err == nil || !strings.Contains(err.Error(), "CV4211") {
		t.Fatalf("template reset depth: %v", err)
	}
	missing := *instance
	missing.Function.Body = nil
	_, err = evt1InvokeClosedTemplateValues(newEVT1ComptimeState(env), &missing, args, Span{})
	if err == nil || !strings.Contains(err.Error(), "COMPTIME_TEMPLATE_BODY_MISSING") {
		t.Fatalf("missing body: %v", err)
	}
}

func TestR9b2ClosedTemplateRecursion(t *testing.T) {
	source := `module Research.Recursive; profile Core;
template <typename T> comptime int Sum(int n) bounded(8) {
    return match { when n == 0 => 0, otherwise => n + Sum<T>(n - 1), };
}
comptime int answer = Sum<int>(4);
static_assert(answer == 10, "bounded closed recursion");
int Use() { return answer; }
`
	if _, err := Parse("recursive.concept", source); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ source, code string }{
		{strings.Replace(source, "bounded(8)", "bounded(2)", 1), "CV4206"},
		{strings.Replace(source, "bounded(8)", "", 1), "CV4217"},
	} {
		_, err := Parse("recursive.concept", tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s: %v", tc.code, err)
		}
	}
}
