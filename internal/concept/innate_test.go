package concept

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const innateTestVerdict = `enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}
`

const innateTestConcept = `[[diagnostic("CV9999")]]
innate concept FieldsAreNamed<FieldDeclaration F>
{
    requires FieldIsNamed(F);
}

comptime Verdict FieldIsNamed(declaration field)
{
    if (compiler.Name(field) == "")
    {
        return Verdict::Refuted(field, "a field needs a name");
    }
    return Verdict::Holds;
}
`

func innateTestSource(body string) string {
	return "module Innate;\nprofile Core;\n\n" + innateTestVerdict + "\n" + body
}

func TestEmbeddedInnateModuleCompiles(t *testing.T) {
	t.Parallel()
	set, err := evt1LoadInnate()
	if err != nil {
		t.Fatal(err)
	}
	if set.module.Name != "Innate" || set.env.enums["Verdict"].Name != "Verdict" {
		t.Fatal("the innate module declares Verdict")
	}
	identity := InnateIdentity()
	if !strings.HasPrefix(identity, "innate-") || len(identity) != len("innate-")+16 {
		t.Fatalf("innate identity: %q", identity)
	}
}

func TestInnateIdentityIgnoresLineEndings(t *testing.T) {
	t.Parallel()
	lf := strings.ReplaceAll(evt1InnateSource, "\r\n", "\n")
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	if evt1InnateIdentityOf(lf) != evt1InnateIdentityOf(crlf) || evt1InnateIdentityOf(lf) != InnateIdentity() {
		t.Fatal("checkout line endings change the innate identity")
	}
	if evt1InnateIdentityOf(lf+"\n// changed\n") == InnateIdentity() {
		t.Fatal("a changed innate module must change the identity")
	}
}

func TestInnateConceptDeclaration(t *testing.T) {
	t.Parallel()
	module, env, err := evt1CompileInnateSource(evt1InnateModulePath, innateTestSource(innateTestConcept))
	if err != nil {
		t.Fatal(err)
	}
	var decl ConceptDecl
	for _, concept := range module.Concepts {
		if concept.Name == "FieldsAreNamed" {
			decl = concept
		}
	}
	if !decl.Innate {
		t.Fatal("innate concept parsed as declared")
	}
	parameters := evt1ConceptParameters(decl)
	if len(parameters) != 1 || parameters[0].Kind != "declaration" || parameters[0].DeclarationKind != string(FieldDeclaration) {
		t.Fatalf("parameter: %+v", parameters)
	}
	if code, ok := evt1InnateDiagnosticCode(decl); !ok || code != "CV9999" {
		t.Fatalf("diagnostic code: %q %v", code, ok)
	}
	predicate, ok := decl.Requirements[0].(*PredicateRequirement)
	if !ok || predicate.Predicate != "FieldIsNamed" || len(predicate.Subjects) != 1 || predicate.Subjects[0].Name != "F" {
		t.Fatalf("requirement: %#v", decl.Requirements[0])
	}
	if !env.innateAuthority {
		t.Fatal("innate compilation carries authority")
	}
}

func TestInnateConceptRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		innate bool
		body   string
		code   string
	}{
		{"outside the compiler", false, innateTestVerdict + innateTestConcept, "INNATE_OUTSIDE_COMPILER"},
		{"missing diagnostic", true, strings.Replace(innateTestConcept, `[[diagnostic("CV9999")]]`+"\n", "", 1), "INNATE_CONCEPT_SHAPE"},
		{"plain declaration parameter", true, strings.Replace(innateTestConcept, "<FieldDeclaration F>", "<declaration F>", 1), "INNATE_CONCEPT_SHAPE"},
		{"bad diagnostic code", true, strings.Replace(innateTestConcept, `"CV9999"`, `"two words"`, 1), "CONCEPT_ATTRIBUTE_INVALID"},
		{"predicate result", true, strings.Replace(strings.Replace(innateTestConcept, "comptime Verdict FieldIsNamed", "comptime bool FieldIsNamed", 1), "return Verdict::Refuted(field, \"a field needs a name\");", "return false;", 1), "PREDICATE_REQUIREMENT_INVALID"},
		{"predicate subject", true, strings.Replace(innateTestConcept, "requires FieldIsNamed(F);", "requires FieldIsNamed(G);", 1), "CONCEPT_DECLARATION_ARGUMENT_INVALID"},
		{"unknown predicate", true, strings.Replace(innateTestConcept, "requires FieldIsNamed(F);", "requires Missing(F);", 1), "PREDICATE_REQUIREMENT_INVALID"},
		{"verdict shape", true, "enum Verdict\n{\n    Holds,\n    Refuted(string message),\n}\n", "INNATE_VERDICT_SHAPE"},
		{"kind parameter on a declared concept", false, "concept Fields<FieldDeclaration F>\n{\n    requires compiler.Authored(F);\n}\n", "INNATE_KIND_PARAMETER"},
		{"diagnostic on a declared concept", false, "[[diagnostic(\"CV9999\")]]\nconcept Fields<declaration F>\n{\n    requires compiler.Authored(F);\n}\n", "CONCEPT_ATTRIBUTE_INVALID"},
		{"predicate in a declared concept", false, innateTestVerdict + "concept Fields<declaration F>\n{\n    requires Check(F);\n}\ncomptime Verdict Check(declaration d) { return Verdict::Holds; }\n", "PREDICATE_REQUIREMENT_SCOPE"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tc.innate {
				source := "module Innate;\nprofile Core;\n\n" + tc.body
				if !strings.Contains(tc.body, "enum Verdict") {
					source = innateTestSource(tc.body)
				}
				_, _, err = evt1CompileInnateSource(evt1InnateModulePath, source)
			} else {
				_, err = Parse("user.concept", "module User;\nprofile Core;\n\n"+tc.body)
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestSemanticArtifactsCarryTheInnateIdentity(t *testing.T) {
	t.Parallel()
	body, err := CompileSemanticModule("lib.concept", "module Lib;\nprofile Core;\n\nint Answer() { return 42; }\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact, _, err := LoadSemanticModuleArtifact(body)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.InnateIdentity != InnateIdentity() {
		t.Fatalf("artifact innate identity %q, want %q", artifact.InnateIdentity, InnateIdentity())
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	raw["innate_identity"] = "innate-0000000000000000"
	stale, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadSemanticModuleArtifact(stale); err == nil || !strings.Contains(err.Error(), "MODULE_INNATE_STALE") {
		t.Fatalf("an artifact from another innate set must be stale, got %v", err)
	}
}
