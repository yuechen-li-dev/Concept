package concept

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dependentCompositionHarness(header, function string) string {
	return fmt.Sprintf(`#include "%s.generated.h"
static int seen[17], bad;
void Observed(int id) { if(id < 0 || id >= 17) { bad=1; } else { seen[id]++; } }
int main(void) {
    if (%s()!=120.0 || bad) return 1;
    for(int i=0;i<17;i++) {
        int expected = i==1 || i==2 || i==3 || i==7 || i==8 || i==16;
        if(seen[i]!=expected) return 2;
    }
    return 0;
}`, header, function)
}

func dependentCompositionSource(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "language", "evt1", "dependent-composition", filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDependentGenericImportedComposition(t *testing.T) {
	dependentCompositionProvider := dependentCompositionSource(t, "Library/Dependent.concept")
	dependentCompositionConsumer := dependentCompositionSource(t, "Consumer.concept")
	artifact := buildSemanticArtifact(t, "Library/Dependent.concept", dependentCompositionProvider, nil)
	repeated := buildSemanticArtifact(t, "Library/Dependent.concept", dependentCompositionProvider, nil)
	if !bytes.Equal(artifact, repeated) {
		t.Fatal("nondeterministic open artifact")
	}
	closed := buildSemanticArtifact(t, "consumer.concept", dependentCompositionConsumer, map[string][]byte{"Library.Dependent": artifact})
	_, restored, err := LoadSemanticModuleArtifact(closed)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range restored.Structs {
		if decl.Application == nil {
			continue
		}
		if decl.Application.Declaration.Module != "Library.Dependent" {
			t.Fatalf("lost declaration owner: %+v", decl.Application)
		}
		for _, field := range decl.Fields {
			if err := evt1RequireClosedType(field.Type, decl.Name, field.Span); err != nil {
				t.Fatal(err)
			}
			if field.Type.ArrayElem != nil && (field.Type.ArrayLength == 0 || evt1StorageHasRuntimeShape(field.Type)) {
				t.Fatalf("unclosed field: %+v", field.Type)
			}
		}
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("consumer.concept", dependentCompositionConsumer, map[string][]byte{"Library.Dependent": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(dependentCompositionConsumer), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "consumer.generated.c")
		again, err := GenerateForTargetWithPolicy(module, []byte(dependentCompositionConsumer), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		for name, value := range outputs {
			if !bytes.Equal(value, again[name]) {
				t.Fatalf("nondeterministic artifact %s", name)
			}
		}
		runFoundationNativeHarness(t, outputs, "dependent_harness.c", dependentCompositionHarness("consumer", "concept_consumer_main"))
		bridge := `module Importer; profile Core; import Consumer;
        double Entry() { return Main(); }`
		imported, err := ParseWithSemanticModules("importer.concept", bridge, map[string][]byte{"Library.Dependent": artifact, "Consumer": closed})
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := GenerateForTargetWithPolicy(imported, []byte(bridge), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarness(t, roundTrip, "restored_harness.c", dependentCompositionHarness("importer", "concept_importer_entry"))
	}
}

func TestDependentGenericCompositionRejections(t *testing.T) {
	dependentCompositionProvider := dependentCompositionSource(t, "Library/Dependent.concept")
	dependentCompositionConsumer := dependentCompositionSource(t, "Consumer.concept")
	artifact := buildSemanticArtifact(t, "Library/Dependent.concept", dependentCompositionProvider, nil)
	cases := []struct {
		name, source, code string
		provider           bool
	}{
		{"copy owned field", strings.Replace(dependentCompositionProvider, "move item, Values", "item, Values", 1), "CV4501", true},
		{"wrong field dimension", strings.Replace(dependentCompositionProvider, "move item, Values<N>{", "move item, Values<2>{", 1), "CV4107", true},
		{"type as value", strings.Replace(dependentCompositionProvider, "move item, Values<N>{", "move item, Values<T>{", 1), "GENERIC_NON_TYPE_ARGUMENT_INVALID", true},
		{"missing record operation", strings.Replace(dependentCompositionProvider, "template <usize M> double Head(ref const Values<M> values) { return values.data[0]; }", "", 1), "CV4153", true},
		{"runtime dimension", strings.Replace(strings.Replace(dependentCompositionConsumer, "comptime usize Dimension = 3u;", "", 1), "double Main() {", "double Main() { usize Dimension = 3u;", 1), "GENERIC_NON_TYPE_ARGUMENT_INVALID", false},
		{"missing evaluator", strings.Replace(dependentCompositionConsumer, "double Weight(ref const Token item) { return item.id as double; }", "", 1), "CV4154", false},
		{"wrong evaluator precision", strings.Replace(dependentCompositionConsumer, "double Weight(ref const Token item) { return item.id as double; }", "float Weight(ref const Token item) { return 1.0; }", 1), "CV4156", false},
		{"zero dimension", strings.Replace(dependentCompositionConsumer, "Compute<Token,1>", "Compute<Token,0>", 1), "CV4207", false},
		{"move then read", strings.Replace(dependentCompositionConsumer, "Compute<Token,2>(move second)", "Compute<Token,2>(move first)", 1), "CV4504", false},
		{"copy workspace", strings.Replace(dependentCompositionProvider, "tensor<double,1> values =", "Workspace<T,N> copy = work; tensor<double,1> values =", 1), "CV4133", true},
		{"escaping field view", strings.Replace(strings.Replace(dependentCompositionProvider, "double Compute(owned T item)", "tensor<double,1> Compute(owned T item)", 1), "return work.values.data[(N as int) - 1] + Weight(ref const work.resource) + (Len(work.values.data) as double) + Head(ref const work.values) - 2.0;", "return values;", 1), "CV4521", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.provider {
				_, err = Parse("Library/Dependent.concept", tc.source)
				if err == nil {
					var changed []byte
					changed, err = CompileSemanticModule("Library/Dependent.concept", tc.source, nil)
					if err == nil {
						_, err = ParseWithSemanticModules("consumer.concept", dependentCompositionConsumer, map[string][]byte{"Library.Dependent": changed})
					}
				}
			} else {
				_, err = ParseWithSemanticModules("consumer.concept", tc.source, map[string][]byte{"Library.Dependent": artifact})
			}
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("got %v; want %s", err, tc.code)
			}
		})
	}
}

func TestDependentGenericLocalRecordComposition(t *testing.T) {
	const source = `profile Core;
template <usize N> struct Fixed { double<array>[N] data; }
template <usize N> double Read() {
    Fixed<N> local = Fixed<N>{[0.0 ...]};
    return local.data[0];
}
double Main() { return Read<3>(); }`
	module, err := Parse("dependent.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "dependent_harness.c", "#include \"dependent.generated.h\"\nint main(void) { return concept_dependent_main() == 0.0 ? 0 : 1; }\n")
}
