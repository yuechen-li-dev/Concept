package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestNamedAggregateChecksCompleteFieldSet(t *testing.T) {
	const prefix = `module AggregateCheck; profile Core; record struct Pair { int x; int y; }
`
	cases := []struct{ source, diagnostic string }{
		{`Pair Make() { return Pair{x = 1}; }`, "AGGREGATE_FIELD_MISSING"},
		{`Pair Make() { return Pair{x = 1, x = 2}; }`, "AGGREGATE_FIELD_DUPLICATE"},
		{`Pair Make() { return Pair{x = 1, z = 2}; }`, "AGGREGATE_FIELD_UNKNOWN"},
		{`Pair Make() { return Pair{x = true, y = 2}; }`, "CV4107"},
	}
	for _, tc := range cases {
		_, err := Parse("aggregate_invalid.concept", prefix+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
			t.Fatalf("expected %s for %s, got %v", tc.diagnostic, tc.source, err)
		}
	}
}

func TestGeneratedAggregateConstructionAcrossArtifacts(t *testing.T) {
	const sourceA = `module AggregateA; profile Core;
[[reflect]] record struct Pair { int x; int y; }
`
	a, err := CompileSemanticModule("AggregateA.concept", sourceA, nil)
	if err != nil {
		t.Fatal(err)
	}
	const sourceB = `module AggregateB; profile Core; import AggregateA;
generator <typename T> DeriveCopy
T Copy(ref const T value) {
    return T{foreach (FieldInfo field in Fields<T>()) { field = value.field; }};
}
derive DeriveCopy reflect<Pair>;
`
	b, err := CompileSemanticModule("AggregateB.concept", sourceB, map[string][]byte{"AggregateA": a})
	if err != nil {
		t.Fatal(err)
	}
	_, transported, err := LoadSemanticModuleArtifact(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(transported.Functions) != 1 || transported.Functions[0].Generated == nil || len(transported.Functions[0].Generated.Inputs) != 2 {
		t.Fatalf("generated aggregate provenance missing: %+v", transported.Functions)
	}
	const sourceC = `module AggregateC; profile Core; import AggregateA; import AggregateB;
int Sum() { Pair pair = Pair{x = 1, y = 2}; Pair copy = Copy(ref const pair); return copy.x + copy.y; }
`
	module, err := ParseWithSemanticModules("AggregateC.concept", sourceC, map[string][]byte{"AggregateA": a, "AggregateB": b})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(sourceC)); err != nil {
		t.Fatal(err)
	}
}

func TestNamedAggregateFailureDropsEarlierMovedField(t *testing.T) {
	const source = `module AggregateFailure; profile Core;
extern "C" void ObserveDrop(int id);
extern "C" int ObservedDrops();
struct Resource { int id; }
void Drop(owned Resource resource) { ObserveDrop(resource.id); }
struct Holder { owned Resource first; int second; }
enum BuildError { Failed, }
Result<int, BuildError> Fail() { return Result::Error(BuildError::Failed); }
Result<Holder, BuildError> Build() {
    owned Resource resource = Resource{1};
    return Result::Ok(Holder{first = move resource, second = Fail()?});
}
int Main() { Result<Holder, BuildError> attempt = Build(); return ObservedDrops(); }
`
	module, err := Parse("AggregateFailure.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	const harness = `#include "aggregatefailure.generated.h"
#include <stdio.h>
static int drops = 0;
void ObserveDrop(int id) { if (id == 1) drops++; }
int ObservedDrops(void) { return drops; }
int main(void) { int value = concept_aggregate_failure_main(); if (value != 1) fprintf(stderr, "observed drops: %d\n", value); return value == 1 ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "aggregate_failure_harness.c", harness)
}

func TestNamedAggregateRejectsMoveAfterMove(t *testing.T) {
	const source = `profile Core;
struct Resource { int id; }
void Drop(owned Resource resource) { }
struct Pair { owned Resource first; owned Resource second; }
Pair Invalid() {
    owned Resource resource = Resource{1};
    return Pair{first = move resource, second = move resource};
}
`
	_, err := Parse("aggregate_move_twice.concept", source)
	if err == nil || !strings.Contains(err.Error(), "move") {
		t.Fatalf("expected move-after-move diagnostic, got %v", err)
	}
}

func TestGeneratedAggregateArtifactAndCDeterminism100Runs(t *testing.T) {
	const source = `module AggregateDeterminism; profile Core;
[[reflect]] record struct Pair { int X; int Y; }
generator <typename T> DeriveMake
T Make(int x, int y) {
    return T{foreach (FieldInfo field in Fields<T>()) { field = x; }};
}
derive DeriveMake reflect<Pair>;
`
	var firstArtifact []byte
	var firstOutputs map[string][]byte
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("AggregateDeterminism.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse("AggregateDeterminism.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			firstArtifact, firstOutputs = artifact, outputs
			continue
		}
		if !bytes.Equal(artifact, firstArtifact) {
			t.Fatalf("generated aggregate artifact changed on run %d", run+1)
		}
		if len(outputs) != len(firstOutputs) {
			t.Fatalf("generated aggregate output count changed on run %d", run+1)
		}
		{
			for path, initial := range firstOutputs {
				current, present := outputs[path]
				if !present || !bytes.Equal(initial, current) {
					t.Fatalf("generated aggregate output %s changed on run %d: first %q now %q", path, run+1, initial, outputs[path])
				}
			}
		}
	}
}
