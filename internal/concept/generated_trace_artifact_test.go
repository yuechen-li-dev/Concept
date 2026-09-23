package concept

import (
	"strings"
	"testing"
	"time"
)

func TestGeneratedTraceAcrossArtifactOnlyModules(t *testing.T) {
	artifacts := r7fRCollectionArtifacts(t)
	artifacts["Standard.Collection.DeriveTrace"] = buildSemanticArtifact(t,
		"Standard/Collection/DeriveTrace.concept", standardMemorySource(t, "Standard/Collection/DeriveTrace.concept"), artifacts)
	sourceA := `module TraceInput; profile Core; import Standard.Collection.Core;
[[reflect]] struct Node {
    [[trace]] CollectorHandle<Node> left;
    [[trace]] CollectorHandle<Node> right;
};
`
	a, err := CompileSemanticModule("TraceInput.concept", sourceA, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["TraceInput"] = a
	sourceB := `module TraceDerived; profile Core;
import TraceInput; import Standard.Collection.DeriveTrace;
derive DeriveTrace reflect<Node>;
`
	unmarked, err := CompileSemanticModule("TraceInput.concept", strings.Replace(sourceA, "[[reflect]] ", "", 1), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	blocked := make(map[string][]byte, len(artifacts))
	for key, value := range artifacts {
		blocked[key] = value
	}
	blocked["TraceInput"] = unmarked
	if _, err := CompileSemanticModule("TraceDerived.concept", sourceB, blocked); err == nil || !strings.Contains(err.Error(), "REFLECT_PERMISSION_REQUIRED") {
		t.Fatalf("unmarked imported Node was reflected: %v", err)
	}
	start := time.Now()
	b, err := CompileSemanticModule("TraceDerived.concept", sourceB, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generated Trace module compile: %s; artifact: %d bytes", time.Since(start), len(b))
	artifacts["TraceDerived"] = b
	checkedDerived, err := ParseWithSemanticModules("TraceDerived.concept", sourceB, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := ExplainGeneratedConcept(checkedDerived, "Trace<Node>")
	if err != nil || proof.Outcome != FactProven || proof.Generated == nil || proof.Generated.Generator != "DeriveTrace" {
		t.Fatalf("generated Trace<Node> proof: %+v, %v", proof, err)
	}
	_, transported, err := LoadSemanticModuleArtifact(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(transported.Functions) != 1 || transported.Functions[0].Generated == nil || len(transported.Functions[0].Generated.Inputs) != 2 {
		t.Fatalf("transported generated Trace missing: %+v", transported.Functions)
	}
	sourceC := `module TraceConsumer; profile Core;
import TraceDerived; import Standard.Collection.Core;
void UseTrace(ref const Node value, ref TraceVisitor<Node> visitor) {
    TraceReferences(ref const value, ref visitor);
}
`
	consumer, err := ParseWithSemanticModules("TraceConsumer.concept", sourceC, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(consumer, []byte(sourceC))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generated Trace consumer C: %d bytes", len(moduleOutput(t, outputs, ".generated.c")))
}
