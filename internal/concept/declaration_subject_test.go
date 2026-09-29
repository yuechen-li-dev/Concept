package concept

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBoundDeclarationKindsAndProvenance(t *testing.T) {
	const source = `module Subjects.Subjects; profile Core;
struct Widget { int ItemCount; int ReadValue(int InputValue) { int LocalValue = InputValue; return LocalValue; } }
concept Marker<T> {}
interface Resettable<T> { requires void Reset(ref T value); }
extern "C" int native_status();
generator <typename T> DeriveValue
int generated_helper(ref const T item) { return item.ItemCount; }
derive DeriveValue reflect<Widget>;
int terrible_name(int SomeParameter) { int SomeLocal = SomeParameter; return SomeLocal; }
`
	module, err := Parse("subjects.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	subjects := ProjectDeclarationSubjects(module)
	assertSubject := func(kind DeclarationKind, name string, provenance DeclarationProvenance) {
		t.Helper()
		for _, subject := range subjects {
			if subject.Kind == kind && subject.Name == name && subject.Provenance == provenance {
				if subject.ID == "" || subject.Owner != module.Name || subject.Site.Line == 0 {
					t.Fatalf("incomplete bound subject: %+v", subject)
				}
				return
			}
		}
		t.Fatalf("missing %s %s %s in %+v", provenance, kind, name, subjects)
	}
	assertSubject(TypeDeclaration, "Widget", DeclarationAuthored)
	assertSubject(FieldDeclaration, "ItemCount", DeclarationAuthored)
	assertSubject(MethodDeclaration, "ReadValue", DeclarationAuthored)
	assertSubject(ParameterDeclaration, "InputValue", DeclarationAuthored)
	assertSubject(LocalDeclaration, "LocalValue", DeclarationAuthored)
	assertSubject(ConceptDeclaration, "Marker", DeclarationAuthored)
	assertSubject(InterfaceDeclaration, "Resettable", DeclarationAuthored)
	assertSubject(FunctionDeclaration, "native_status", DeclarationForeign)
	assertSubject(FunctionDeclaration, "generated_helper", DeclarationGenerated)
	assertSubject(FunctionDeclaration, "terrible_name", DeclarationAuthored)
	baseline, err := json.Marshal(subjects)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		again, err := json.Marshal(ProjectDeclarationSubjects(module))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, baseline) {
			t.Fatalf("declaration inventory drifted on pass %d", i)
		}
	}
}

func TestProjectSubjectsExcludeArtifactDeclarations(t *testing.T) {
	const provider = `module Subjects.Provider; profile Core;
struct foreign_type { int ForeignField; }
concept ProviderRule<T> {}
int provider_name() { return 1; }
extern "C" int native_status();
generator <typename T> DeriveField
int generated_helper(ref const T item) { return item.ForeignField; }
derive DeriveField reflect<foreign_type>;
`
	artifact, err := CompileSemanticModule("provider.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module Subjects.Consumer; profile Core; import Subjects.Provider;
int consumer_name() { return provider_name(); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Subjects.Provider": artifact})
	if err != nil {
		t.Fatal(err)
	}
	all := DeclarationSubjects(module)
	project := ProjectDeclarationSubjects(module)
	if len(project) != 1 || project[0].Name != "consumer_name" {
		t.Fatalf("root policy scope: %+v", project)
	}
	for _, wanted := range []string{"foreign_type", "ProviderRule", "provider_name", "native_status", "generated_helper"} {
		found := false
		for _, subject := range all {
			if subject.Name == wanted && subject.Owner == "Subjects.Provider" {
				found = true
			}
		}
		if !found {
			t.Fatalf("artifact declaration %s lost owner: %+v", wanted, all)
		}
	}
	for _, expectation := range []struct {
		name       string
		provenance DeclarationProvenance
	}{
		{"native_status", DeclarationForeign},
		{"generated_helper", DeclarationGenerated},
	} {
		found := false
		for _, subject := range all {
			if subject.Name == expectation.name && subject.Provenance == expectation.provenance {
				found = true
			}
		}
		if !found {
			t.Fatalf("artifact provenance lost for %s: %+v", expectation.name, all)
		}
	}
}

func TestStandaloneSourceHasProjectSubjects(t *testing.T) {
	module, err := Parse("standalone.concept", "profile Core; void terrible_name() {}")
	if err != nil {
		t.Fatal(err)
	}
	subjects := ProjectDeclarationSubjects(module)
	if len(subjects) != 1 || subjects[0].Name != "terrible_name" || subjects[0].Owner != "standalone.concept" {
		t.Fatalf("standalone semantic subject: %+v", subjects)
	}
}

func TestMachineUsesBoundAutomataDeclaration(t *testing.T) {
	const source = `profile Core;
automata Worker {
    machine Run returns int {
        state Done { complete 17; }
    }
}
int Main() {
    instance Worker worker();
    Step(worker, Run);
    return Result(worker, Run).success;
}`
	module, err := Parse("machine.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range ProjectDeclarationSubjects(module) {
		if subject.Kind == MachineDeclaration && subject.Name == "Run" && subject.Provenance == DeclarationAuthored {
			return
		}
	}
	t.Fatal("bound machine declaration missing")
}
