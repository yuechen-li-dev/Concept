package concept

import (
	"strings"
	"testing"
)

func TestR9a2NoAllocationUsesCheckedOverloadTarget(t *testing.T) {
	const source = `module CheckedTargets; profile Core;
extern "C" int Allocate();
requires compiler.Allocates(Allocate);
int Work(int value) { return value; }
int Work(double value) { return Allocate(); }
int Safe() { return Work(3); }
int Unsafe() { return Work(3.0); }
void Proof() { Assert.Concept<NoAllocation>(Safe, "selected integer overload has no allocation"); }
`
	m, err := Parse("targets.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileSemanticModule("targets.concept", source, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, loaded, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	for _, function := range loaded.Functions {
		if function.Body != nil {
			for _, call := range evt1DirectCallTargets(*function.Body) {
				if call.Signature != "" {
					t.Fatal("resolved target trusted from artifact")
				}
			}
		}
	}
	consumer := `module Consumer; profile Core; import CheckedTargets; void ConsumerProof() { Assert.Concept<NoAllocation>(Safe, "artifact preserves selected target effect"); }`
	if _, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"CheckedTargets": artifact}); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("unsafe.concept", strings.Replace(source, "(Safe,", "(Unsafe,", 1)); err == nil {
		t.Fatal("allocating selected overload proved NoAllocation")
	}
	_ = m
}
