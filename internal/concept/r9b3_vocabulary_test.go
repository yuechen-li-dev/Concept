package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"strings"
	"testing"
	"time"
)

func r9b3VocabularyArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	path := "../../libraries/Standard/Semantic/Vocabulary.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{"Standard.Semantic.Vocabulary": buildSemanticArtifact(t, path, string(source), nil)}
}

const r9b3Subjects = `module Research.Subjects;
profile Core;
import Standard.Semantic.Vocabulary;

record struct Header
{
    uint32 count;
    double scale;
}
template <typename T>
record struct DataBox
{
    T data;
}
using ClosedBox = DataBox<uint16>;
using Words = uint16<array>[4];
using NoElements = uint16<array>[0];
record struct Empty
{
}
record struct MixedEmpty
{
    uint32 marker;
    Empty absent;
}
record struct MixedZero
{
    uint32 marker;
    NoElements absent;
}
enum Tag
{
    First,
    Second,
}
enum Payload
{
    Text(string value),
}
ref struct Borrowed
{
    ref int target;
}
class Resource
{
    public:
    int id;
}
void Drop(owned Resource resource)
{
}
record struct Owner
{
    owned Resource resource;
}
extern "C" handle ForeignOpaque;

int Main()
{
    Assert.Concept<PlainData>(SUBJECT, "self-contained data representation");
    return 7;
}
`

func TestR9b3PlainDataBoundary(t *testing.T) {
	artifacts := r9b3VocabularyArtifacts(t)
	for _, tc := range []struct{ subject, code, reason string }{
		{"bool", "", ""}, {"uint8", "", ""}, {"uint16", "", ""}, {"uint64", "", ""},
		{"half", "", ""}, {"float", "", ""}, {"double", "", ""},
		{"Header", "", ""}, {"Words", "", ""}, {"ClosedBox", "", ""}, {"Tag", "", ""},
		{"Resource", "CONCEPT_ASSERT_DISPROVEN", "HasDrop"},
		{"Owner", "CONCEPT_ASSERT_DISPROVEN", "FieldProblem"},
		{"Borrowed", "CONCEPT_ASSERT_DISPROVEN", "ContainsReference"},
		{"string", "CONCEPT_ASSERT_DISPROVEN", "ContainsReference"},
		{"ForeignOpaque", "CONCEPT_ASSERT_UNKNOWN", "no trusted"},
		{"Payload", "CONCEPT_ASSERT_UNKNOWN", "payload-sum"},
		{"Empty", "CONCEPT_ASSERT_UNKNOWN", "zero-size"},
		{"NoElements", "CONCEPT_ASSERT_UNKNOWN", "zero-size"},
		{"MixedEmpty", "CONCEPT_ASSERT_UNKNOWN", "no complete"},
		{"MixedZero", "CONCEPT_ASSERT_UNKNOWN", "no complete"},
	} {
		source := strings.Replace(r9b3Subjects, "SUBJECT", tc.subject, 1)
		module, err := ParseWithSemanticModules("subjects.concept", source, artifacts)
		if tc.code == "" {
			if err != nil {
				t.Fatalf("%s: %v", tc.subject, err)
			}
			env, err := analyzeModule(module)
			if err != nil {
				t.Fatal(err)
			}
			subject, err := evt1ResolveConceptAssertionSubject(env, newEVT1Scope(nil), &NameExpr{Name: tc.subject})
			if err != nil {
				t.Fatal(err)
			}
			verdict, err := evt1InvokePredicateOnMeasured(env, env, "PlainDataHolds", []Value{evt1TypenameValue(subject.typeValue)}, Span{}, nil)
			if err != nil || verdict.Evidence == nil {
				t.Fatalf("shadow %s: %v", tc.subject, err)
			}
			size, alignment, err := evt1TypeGeometry(env, subject.typeValue)
			if err != nil || verdict.Evidence.Fields["size"].UintValue != uint64(size) || verdict.Evidence.Fields["alignment"].UintValue != uint64(alignment) {
				t.Fatalf("geometry shadow disagreement %s: %+v %v", tc.subject, verdict, err)
			}
			if tc.subject == "Header" {
				outputs, err := Generate(module, []byte(source))
				if err != nil {
					t.Fatal(err)
				}
				cType := evt1CType(subject.typeValue)
				runFoundationNativeHarness(t, outputs, "plain_harness.c", "#include \"subjects.generated.h\"\n_Static_assert(sizeof("+cType+")==16, \"PlainData size\");\n_Static_assert(_Alignof("+cType+")==8, \"PlainData alignment\");\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Main")+"() == 7 ? 0 : 1; }\n")
			}
			continue
		}
		var d Diagnostic
		if !errors.As(err, &d) || d.Code != tc.code || d.Proof == nil || !strings.Contains(RenderProofVerbose(*d.Proof), tc.reason) {
			t.Fatalf("%s expected %s %s: %v", tc.subject, tc.code, tc.reason, err)
		}
	}
}

func TestR9b3PlainDataArtifactDeterminismAndAuthority(t *testing.T) {
	artifacts := r9b3VocabularyArtifacts(t)
	source := strings.Replace(r9b3Subjects, "SUBJECT", "Header", 1)
	artifact := buildSemanticArtifact(t, "subjects.concept", source, artifacts)
	artifacts["Research.Subjects"] = artifact
	consumer := `module Research.Consumer;
profile Core;
import Research.Subjects;

int Use()
{
    Assert.Concept<PlainData>(Header, "artifact-only representation");
    return Main();
}
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "artifact_plain.c", "#include \"consumer.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Use")+"() == 7 ? 0 : 1; }\n")
	start := time.Now()
	var firstProof []byte
	var firstDiagnostic string
	for run := 0; run < 100; run++ {
		again := buildSemanticArtifact(t, "subjects.concept", source, artifacts)
		if !bytes.Equal(again, artifact) {
			t.Fatalf("artifact drift %d", run)
		}
		parsed, err := ParseWithSemanticModules("consumer.concept", consumer, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := explainModule(parsed, 0)
		if err != nil {
			t.Fatal(err)
		}
		proof, err := json.Marshal(graph)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ParseWithSemanticModules("consumer.concept", strings.Replace(consumer, "(Header,", "(Resource,", 1), artifacts)
		var d Diagnostic
		if !errors.As(err, &d) || d.Code != "CONCEPT_ASSERT_DISPROVEN" {
			t.Fatal(err)
		}
		diagnostic := err.Error() + RenderProofVerbose(*d.Proof)
		if run == 0 {
			firstProof, firstDiagnostic = proof, diagnostic
		}
		if !bytes.Equal(proof, firstProof) || diagnostic != firstDiagnostic {
			t.Fatalf("explain/diagnostic drift %d", run)
		}
	}
	envelope, _, err := LoadSemanticModuleArtifact(artifact)
	if err != nil || len(envelope.PredicateVerdicts) != 1 {
		t.Fatalf("payload transport: %v", err)
	}
	verdict := envelope.PredicateVerdicts[0]
	if verdict.Evidence == nil || len(verdict.FactAuthority) != 0 || !strings.Contains(verdict.Evidence.Render(), "16") {
		t.Fatalf("evidence %+v", verdict)
	}
	metadata, err := json.Marshal(verdict)
	if err != nil {
		t.Fatal(err)
	}
	withoutAssertion := strings.Replace(source, "    Assert.Concept<PlainData>(Header, \"self-contained data representation\");\n", "", 1)
	control := buildSemanticArtifact(t, "subjects.concept", withoutAssertion, artifacts)
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	usage := &evt1ComptimeUsage{}
	measured, err := evt1InvokePredicateOnMeasured(env, env, "PlainDataHolds", []Value{evt1TypenameValue(Type{Name: "Header", Kind: TypeStruct})}, Span{}, usage)
	if err != nil || measured.Outcome != FactProven || usage.Fuel == 0 {
		t.Fatalf("usage: %+v %v", usage, err)
	}
	t.Logf("PlainData Header fuel=%d depth=%d loop=%d selected_verdict_bytes=%d complete_assertion_artifact_growth=%d", usage.Fuel, usage.Depth, usage.Loop, len(metadata), len(artifact)-len(control))
	t.Logf("100 artifact/explain/diagnostic runs wall=%s vocabulary_bytes=%d subject_bytes=%d proof_bytes=%d", time.Since(start), len(artifacts["Standard.Semantic.Vocabulary"]), len(artifact), len(firstProof))
	fake := `module Research.Forgery;
profile Core;
import Research.Subjects;
enum Claim
{
    Accepted
}
comptime Verdict<Claim, Claim> FakeHolds(typename subject)
{
    return Verdict::Proven(Claim::Accepted);
}
concept FakePlainData<T>
{
    requires FakeHolds(T);
}
concept FakeRelocatable<T>
{
    requires FakeHolds(T);
}
concept FakeClosedWorld<T>
{
    requires FakeHolds(T);
}
concept FakeFiniteDomain<T>
{
    requires FakeHolds(T);
}
int Forge()
{
    Assert.Concept<FakePlainData>(Resource, "local claim");
    Assert.Concept<FakeRelocatable>(Resource, "local claim");
    Assert.Concept<FakeClosedWorld>(Resource, "local claim");
    Assert.Concept<FakeFiniteDomain>(Resource, "local claim");
    Assert.Concept<PlainData>(Resource, "actual property");
    return 0;
}
`
	_, err = ParseWithSemanticModules("forgery.concept", fake, artifacts)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
		t.Fatalf("forged property: %v", err)
	}
	if len(evt1InnateFactAuthority) != 0 {
		t.Fatal("unexpected privileged authority")
	}
	localClaims := strings.Replace(fake, "    Assert.Concept<PlainData>(Resource, \"actual property\");\n", "", 1)
	claimArtifact := buildSemanticArtifact(t, "local-claims.concept", localClaims, artifacts)
	claimEnvelope, _, err := LoadSemanticModuleArtifact(claimArtifact)
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range claimEnvelope.PredicateVerdicts {
		if len(claim.FactAuthority) != 0 {
			t.Fatalf("user authority: %+v", claim)
		}
	}
}
