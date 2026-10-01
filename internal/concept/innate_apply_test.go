package concept

import (
	"errors"
	"strings"
	"testing"
)

const innateApplyRules = `module Innate;
profile Core;

enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}

[[diagnostic("TEST_FIELD_NAME")]]
innate concept FieldsAreNotX<FieldDeclaration F>
{
    requires FieldIsNotX(F);
}

comptime Verdict FieldIsNotX(declaration field)
{
    if (compiler.Name(field) == "x")
    {
        return Verdict::Refuted(field, "field " + compiler.QualifiedName(field) + " is named x; name it for what it holds");
    }
    return Verdict::Holds;
}

[[diagnostic("TEST_TYPE_NAME")]]
innate concept TypesAreNotNamedBad<TypeDeclaration T>
{
    requires TypeIsNotNamedBad(T);
}

comptime Verdict TypeIsNotNamedBad(declaration type)
{
    if (compiler.Name(type) == "Bad")
    {
        return Verdict::Refuted(type, "Bad is not a name");
    }
    return Verdict::Holds;
}
`

func innateApplySet(t *testing.T, source string) evt1InnateSet {
	t.Helper()
	module, env, err := evt1CompileInnateSource(evt1InnateModulePath, source)
	if err != nil {
		t.Fatal(err)
	}
	return evt1InnateSet{module: module, env: env}
}

func innateApply(t *testing.T, set evt1InnateSet, source string) (*semanticEnv, error) {
	t.Helper()
	module, err := Parse("user.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	return env, evt1ApplyInnateConcepts(env, module, set)
}

const innateApplyUser = `module User;
profile Core;

struct Point
{
    int left;
    int right;
}

template <typename T>
struct Box
{
    T value;
}

int Use()
{
    Box<int> box = Box<int>{1};
    Point point = Point{2, 3};
    return box.value + point.left;
}
`

func TestInnateConceptsJudgeEveryDeclarationOfTheirKind(t *testing.T) {
	t.Parallel()
	env, err := innateApply(t, innateApplySet(t, innateApplyRules), innateApplyUser)
	if err != nil {
		t.Fatal(err)
	}
	judged := map[string][]string{}
	for _, evaluation := range env.innateEvaluations {
		if evaluation.Outcome != FactProven {
			t.Fatalf("%s on %s: %s", evaluation.Concept, evaluation.Subject.qualifiedName(), evaluation.Outcome)
		}
		judged[evaluation.Concept] = append(judged[evaluation.Concept], evaluation.Subject.qualifiedName())
	}
	if got := strings.Join(judged["TypesAreNotNamedBad"], ","); got != "Point,Box<int>" {
		t.Fatalf("type subjects in source order, generic instances included: %s", got)
	}
	if got := strings.Join(judged["FieldsAreNotX"], ","); got != "Point.left,Point.right,Box<int>.value" {
		t.Fatalf("field subjects: %s", got)
	}

	graph, ok := evt1ExplainInnate(env, "user.concept", 6)
	if !ok || graph.Outcome != FactProven || len(graph.Nodes) != 2 || !strings.Contains(graph.Nodes[1].Label, "FieldsAreNotX (TEST_FIELD_NAME)") {
		t.Fatalf("explain the field on line 6: %+v", graph)
	}
}

func TestInnateRefutationIsTheDiagnostic(t *testing.T) {
	t.Parallel()
	set := innateApplySet(t, innateApplyRules)
	_, err := innateApply(t, set, `module User;
profile Core;

struct Bad
{
    int y;
}

struct Pair
{
    int x;
    int y;
}
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) {
		t.Fatalf("expected a refutation, got %v", err)
	}
	if diagnostic.Code != "TEST_TYPE_NAME" || diagnostic.Message != "Bad is not a name" || diagnostic.Span.Line != 4 {
		t.Fatalf("the first refutation in source order decides: %s at %d: %s", diagnostic.Code, diagnostic.Span.Line, diagnostic.Message)
	}
	if diagnostic.Proof == nil || diagnostic.Proof.Outcome != FactDisproven || len(diagnostic.Proof.FailedNodes) != 1 {
		t.Fatalf("the refutation carries its proof: %+v", diagnostic.Proof)
	}

	_, err = innateApply(t, set, "module User;\nprofile Core;\n\nstruct Pair\n{\n    int x;\n}\n")
	if !errors.As(err, &diagnostic) || diagnostic.Code != "TEST_FIELD_NAME" || diagnostic.Span.Line != 6 {
		t.Fatalf("field refutation points at the field: %v", err)
	}
	if diagnostic.Message != "field Pair.x is named x; name it for what it holds" {
		t.Fatalf("message: %s", diagnostic.Message)
	}
}

func TestInnateAnalysesAndPrerequisites(t *testing.T) {
	t.Parallel()
	set := innateApplySet(t, `module Innate;
profile Core;

enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}

[[diagnostic("TEST_AUTHORED")]]
innate concept TypesAreAuthored<TypeDeclaration T>
{
    requires compiler.Authored(T);
}

[[diagnostic("TEST_COMPOSED")]]
innate concept Composed<TypeDeclaration T>
{
    requires TypesAreAuthored<declaration T>;
}
`)
	_, err := innateApply(t, set, "module User;\nprofile Core;\n\nstruct Plain\n{\n    int value;\n}\n")
	if err != nil {
		t.Fatalf("authored types hold: %v", err)
	}
	_, err = innateApply(t, set, innateApplyUser)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "TEST_AUTHORED" || !strings.Contains(diagnostic.Message, "Box<int>") {
		t.Fatalf("a generic instance is generated, not authored: %v", err)
	}
}

func TestUndecidedInnateConceptIsACompilerDefect(t *testing.T) {
	t.Parallel()
	set := innateApplySet(t, `module Innate;
profile Core;

enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}

[[diagnostic("TEST_BROKEN")]]
innate concept Broken<TypeDeclaration T>
{
    requires ReadsPastTheEnd(T);
}

comptime Verdict ReadsPastTheEnd(declaration type)
{
    if (compiler.IsField(compiler.Field(type, 99)))
    {
        return Verdict::Holds;
    }
    return Verdict::Holds;
}
`)
	_, err := innateApply(t, set, "module User;\nprofile Core;\n\nstruct Plain\n{\n    int value;\n}\n")
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "INNATE_UNDECIDED" || !strings.Contains(diagnostic.Message, "not in your program") || diagnostic.Span.Line != 4 {
		t.Fatalf("expected INNATE_UNDECIDED at the struct, got %v", err)
	}
}

func TestInnateConceptKinds(t *testing.T) {
	t.Parallel()
	_, _, err := evt1CompileInnateSource(evt1InnateModulePath, strings.Replace(innateApplyRules, "<TypeDeclaration T>", "<FunctionDeclaration T>", 1))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "INNATE_CONCEPT_SHAPE" {
		t.Fatalf("function-kind innate concepts are not applicable yet, got %v", err)
	}
}
