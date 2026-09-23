package concept

import (
	"bytes"
	"testing"
)

func TestGeneratedTraceHundredRunArtifactAndBackendDeterminism(t *testing.T) {
	artifacts := r7fRCollectionArtifacts(t)
	artifacts["Standard.Collection.DeriveTrace"] = buildSemanticArtifact(t,
		"Standard/Collection/DeriveTrace.concept", standardMemorySource(t, "Standard/Collection/DeriveTrace.concept"), artifacts)
	sourceA := `module TraceDeterminismInput; profile Core; import Standard.Collection.Core;
[[reflect]] struct Node { CollectorHandle<Node> left; CollectorHandle<Node> right; };
`
	a, err := CompileSemanticModule("TraceDeterminismInput.concept", sourceA, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["TraceDeterminismInput"] = a
	sourceB := `module TraceDeterminismOutput; profile Core;
import TraceDeterminismInput; import Standard.Collection.DeriveTrace;
derive DeriveTrace reflect<Node>;
`
	var expectedArtifact, expectedView, expectedProof []byte
	var expectedOutputs Outputs
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("TraceDeterminismOutput.concept", sourceB, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		module, err := ParseWithSemanticModules("TraceDeterminismOutput.concept", sourceB, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		view, err := InspectGeneratedDeclarations(module, "TraceReferences")
		if err != nil {
			t.Fatal(err)
		}
		graph, err := ExplainGeneratedConcept(module, "Trace<Node>")
		if err != nil {
			t.Fatal(err)
		}
		proof, err := SerializeProof(graph)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(sourceB))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			expectedArtifact, expectedView, expectedProof, expectedOutputs = artifact, view, proof, outputs
			continue
		}
		if !bytes.Equal(artifact, expectedArtifact) || !bytes.Equal(view, expectedView) || !bytes.Equal(proof, expectedProof) {
			t.Fatalf("generated Trace artifact, view, or proof changed on run %d", run)
		}
		for name, expected := range expectedOutputs {
			if !bytes.Equal(outputs[name], expected) {
				t.Fatalf("generated Trace %s changed on run %d", name, run)
			}
		}
	}
}
