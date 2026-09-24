package concept

import (
	"strings"
	"testing"
)

func TestR7h2SchemaFailureExplainsMissingAdmission(t *testing.T) {
	const source = `module SchemaFailure; profile Core;
struct Unchecked { int Value; }
concept Admitted<T> { requires bool IsAdmitted(ref const T value); }
`
	module, err := Parse("SchemaFailure.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ExplainGeneratedConcept(module, "Admitted<Unchecked>")
	if err == nil || !strings.Contains(err.Error(), "CV4153") || !strings.Contains(err.Error(), "IsAdmitted") {
		t.Fatalf("schema failure did not explain missing admission: %v", err)
	}
}

func TestR7h2RefinedDerivationRequiresAdmission(t *testing.T) {
	artifacts := map[string][]byte{}
	artifacts["Standard.Octagon.Core"] = buildSemanticArtifact(t,
		"Standard/Octagon/Core.concept", standardMemorySource(t, "Standard/Octagon/Core.concept"), artifacts)
	artifacts["Standard.Octagon.Derive"] = buildSemanticArtifact(t,
		"Standard/Octagon/Derive.concept", standardMemorySource(t, "Standard/Octagon/Derive.concept"), artifacts)
	const source = `module MissingAdmission; profile Core;
import Standard.Octagon.Core; import Standard.Octagon.Derive;
record struct Unchecked { int Value; }
derive DeriveOctagonRefinedRead reflect<Unchecked>;
`
	_, err := CompileSemanticModule("MissingAdmission.concept", source, artifacts)
	if err == nil || !strings.Contains(err.Error(), "AdmitOctagon") || !strings.Contains(err.Error(), "DeriveOctagonRefinedRead") {
		t.Fatalf("unchecked refinement derivation was not rejected with generator context: %v", err)
	}
}

func TestR7h2RefinedDerivationRejectsMultipleRepresentationFields(t *testing.T) {
	artifacts := map[string][]byte{}
	artifacts["Standard.Octagon.Core"] = buildSemanticArtifact(t,
		"Standard/Octagon/Core.concept", standardMemorySource(t, "Standard/Octagon/Core.concept"), artifacts)
	artifacts["Standard.Octagon.Derive"] = buildSemanticArtifact(t,
		"Standard/Octagon/Derive.concept", standardMemorySource(t, "Standard/Octagon/Derive.concept"), artifacts)
	const source = `module MultipleRepresentationFields; profile Core;
import Standard.Octagon.Core; import Standard.Octagon.Derive;
record struct Ambiguous { int First; int Second; }
derive DeriveOctagonRefinedRead reflect<Ambiguous>;
`
	_, err := CompileSemanticModule("MultipleRepresentationFields.concept", source, artifacts)
	if err == nil || !strings.Contains(err.Error(), "GENERATOR_SINGLE_FIELD_REQUIRED") {
		t.Fatalf("refined derivation accepted multiple representation fields: %v", err)
	}
}

func TestR7h2ScalarCodecCoreProvesNoAllocation(t *testing.T) {
	source := standardMemorySource(t, "Standard/Octagon/Core.concept") + `
int VerifyCodecEffects() {
    Assert.Concept<NoAllocation>(ReadInt, "bounded scalar reader uses fixed storage");
    Assert.Concept<NoAllocation>(WriteInt, "bounded scalar writer uses fixed storage");
    return 0;
}
`
	if _, err := Parse("Core.concept", source); err != nil {
		t.Fatal(err)
	}
}
