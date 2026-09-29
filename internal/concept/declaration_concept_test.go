package concept

import (
	"strings"
	"testing"
)

func TestDeclarationConceptAssertion(t *testing.T) {
	const source = `module Declarations.Basic; profile Core;
concept IsAuthoredFunction<declaration D> {
    requires compiler.Name(D);
    requires compiler.DeclarationKind(D);
    requires compiler.Authored(D);
    requires compiler.FunctionDeclaration(D);
}
void FastFunction() {}
void Check() { Assert.Concept<IsAuthoredFunction>(FastFunction, "bound authored function"); }
`
	module, err := Parse("declaration.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := evt1ConceptParameters(module.Concepts[0])[0].Kind; got != "declaration" {
		t.Fatalf("parameter kind = %s", got)
	}
	graph, err := ExplainSource("declaration.concept", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Outcome != FactProven || !strings.Contains(graph.Goal, "IsAuthoredFunction") {
		t.Fatalf("declaration proof = %+v", graph)
	}
}

func TestDeclarationConceptCategoryMismatch(t *testing.T) {
	for _, test := range []struct{ source, message string }{
		{`profile Core; concept DeclarationOnly<declaration D> {} void Check() { Assert.Concept<DeclarationOnly>(int, "wrong category"); }`, "expects a declaration subject"},
		{`profile Core; concept TypeOnly<T> {} void Function() {} void Check() { Assert.Concept<TypeOnly>(Function, "wrong category"); }`, "expects a type"},
	} {
		_, err := Parse("category.concept", test.source)
		if err == nil || !strings.Contains(err.Error(), "CONCEPT_ARGUMENT_CATEGORY_MISMATCH") || !strings.Contains(err.Error(), test.message) {
			t.Fatalf("wanted %s, got %v", test.message, err)
		}
	}
}

func TestDeclarationConceptNoAllocationTruth(t *testing.T) {
	const prefix = `module Declarations.Effects; profile Core;
concept HotPathPolicy<declaration F> { requires compiler.NoAllocation(F); }
void FastFunction() {}
extern "C" void NativeTick();
`
	if _, err := Parse("proven.concept", prefix+`void Check() { Assert.Concept<HotPathPolicy>(FastFunction, "hot path"); }`); err != nil {
		t.Fatal(err)
	}
	_, err := Parse("unknown.concept", prefix+`void Check() { Assert.Concept<HotPathPolicy>(NativeTick, "hot path"); }`)
	if diagnostic, ok := err.(Diagnostic); !ok || diagnostic.Code != "CONCEPT_ASSERT_UNKNOWN" || diagnostic.Proof == nil || diagnostic.Proof.Outcome != FactUnknown {
		t.Fatalf("unknown NoAllocation proof = %v", err)
	}
}

func TestDeclarationConceptCompositionAndArtifactOnly(t *testing.T) {
	const provider = `module Declarations.Provider; profile Core;
concept IsForeign<declaration D> { requires compiler.Foreign(D); }
concept ForeignPolicy<declaration D> { requires IsForeign<declaration D>; }
extern "C" int native_status();
`
	artifact, err := CompileSemanticModule("provider.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module Declarations.Consumer; profile Core; import Declarations.Provider;
requires ForeignPolicy<declaration native_status>;
void Check() { Assert.Concept<ForeignPolicy>(native_status, "artifact declaration identity"); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Declarations.Provider": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Assertions) != 1 || module.Assertions[0].Arguments[0].Kind != "declaration" {
		t.Fatalf("declaration application was not retained: %+v", module.Assertions)
	}
	for _, mismatch := range []struct{ source, want string }{
		{`module Declarations.Consumer; profile Core; import Declarations.Provider; requires ForeignPolicy<int>;`, "expects a declaration subject"},
		{`module Declarations.Consumer; profile Core; import Declarations.Provider; concept TypeOnly<T> {} requires TypeOnly<declaration native_status>;`, "expects a type"},
	} {
		_, err := ParseWithSemanticModules("mismatch.concept", mismatch.source, map[string][]byte{"Declarations.Provider": artifact})
		if err == nil || !strings.Contains(err.Error(), mismatch.want) {
			t.Fatalf("wanted %s, got %v", mismatch.want, err)
		}
	}
}

func TestCanonicalNamingIsAProposition(t *testing.T) {
	const source = `profile Core;
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
void terrible_name() {}
`
	if _, err := Parse("legal_without_policy.concept", source); err != nil {
		t.Fatal(err)
	}
	_, err := Parse("required.concept", source+`requires ProjectNaming<declaration terrible_name>;`)
	if diagnostic, ok := err.(Diagnostic); !ok || diagnostic.Code != "CONCEPT_ASSERT_DISPROVEN" || diagnostic.Proof == nil || diagnostic.Proof.Outcome != FactDisproven {
		t.Fatalf("naming requirement = %v", err)
	}
	if !declarationNameHasStyle("XMLDocument", "PascalCase") || declarationNameHasStyle("do_work", "PascalCase") || declarationNameSuggestion("do_work", "PascalCase") != "DoWork" || declarationNameSuggestion("SomeLocal", "camelCase") != "someLocal" {
		t.Fatal("naming classifier or suggestion changed")
	}
}

func TestGeneratedDeclarationConceptArtifactOnly(t *testing.T) {
	const provider = `module Declarations.Generated; profile Core;
struct Item { int value; }
generator <typename T> DeriveValue
int generated_helper(ref const T item) { return item.value; }
derive DeriveValue reflect<Item>;
concept IsGenerated<declaration D> { requires compiler.Generated(D); }
`
	artifact, err := CompileSemanticModule("generated.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module Declarations.Consumer; profile Core; import Declarations.Generated;
requires IsGenerated<declaration generated_helper>;
void Check() { Assert.Concept<IsGenerated>(generated_helper, "generated artifact identity"); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Declarations.Generated": artifact})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, subject := range DeclarationSubjects(module) {
		if subject.Name == "generated_helper" && subject.Owner == "Declarations.Generated" && subject.Provenance == DeclarationGenerated {
			found = true
		}
	}
	if !found {
		t.Fatal("generated declaration identity lost through artifact")
	}
}
