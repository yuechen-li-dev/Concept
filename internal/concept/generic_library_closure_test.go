package concept

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7f1ClosedGenericLocalArrayIsFixedC11Storage(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "local_array_generic.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["local_array_generic.generated.c"])
	if strings.Contains(body, "[N]") || strings.Contains(body, "[runtime]") {
		t.Fatal("generated C retains symbolic or runtime extent")
	}
	runFoundationNativeHarness(t, outputs, "generic_local_array_harness.c", "#include \"local_array_generic.generated.h\"\nint main(void) { return concept_local_array_generic_main() == 50 ? 0 : 1; }\n")
}

func TestR7f1GenericClosureIsByteIdenticalAcross100Runs(t *testing.T) {
	provider := `module Library.Closure; profile Core;
concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); }
template <typename T> requires Transformable<T>
int Apply(ref const T value) { int<array>[4] scratch = [1 ...]; return Transform<int>(ref const value, scratch[0]); }
`
	consumer := `module App; profile Core; import Library.Closure;
struct Item { int value; };
template <typename V> V Transform(ref const Item value, V seed) { return seed; }
requires Transformable<Item>;
int Main() { Item item = Item{3}; return Apply<Item>(ref const item); }
`
	var firstArtifact, firstMIR, firstC []byte
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("Library/Closure.concept", provider, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library.Closure": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(consumer))
		if err != nil {
			t.Fatal(err)
		}
		mir := []byte(moduleOutput(t, outputs, ".mir.json"))
		generated := []byte(moduleOutput(t, outputs, ".generated.c"))
		if run == 0 {
			firstArtifact, firstMIR, firstC = artifact, mir, generated
			continue
		}
		if !bytes.Equal(firstArtifact, artifact) || !bytes.Equal(firstMIR, mir) || !bytes.Equal(firstC, generated) {
			t.Fatalf("generic closure output changed on run %d", run)
		}
	}
}

func TestR7f1GenericCallsGenericInStrictC11(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "generic_calls_generic.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "generic_call_harness.c", "#include \"generic_calls_generic.generated.h\"\nint main(void) { return concept_generic_calls_generic_main() == 42 ? 0 : 1; }\n")
}

func TestR7f1ValueShapedConceptAlreadyWorks(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "value_shaped_concept.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "value_shaped_concept_harness.c", "#include \"value_shaped_concept.generated.h\"\nint main(void) { return concept_value_shaped_concept_main() == 5 ? 0 : 1; }\n")
}

func TestR7f1GenericRequiredOperationInStrictC11(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "generic_required_operation.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "generic_required_operation_harness.c", "#include \"generic_required_operation.generated.h\"\nint main(void) { return concept_generic_required_operation_main() == 7 ? 0 : 1; }\n")
}

func TestR7f1DirectedGenericRejections(t *testing.T) {
	cases := []struct {
		name   string
		source string
		code   string
	}{
		{"runtime extent", "profile Core; template <usize N> int Work(int extent) { int<array>[extent] values = [7 ...]; return 0; }", "CV4558"},
		{"nested wrong arity", "profile Core; template <typename U> U Identity(U value) { return value; } template <typename T> T Forward(T value) { return Identity<T, T>(value); }", "CV4179"},
		{"nested unmet constraint", "profile Core; concept Resource<T> { requires int Measure(ref const T value); } template <typename U> requires Resource<U> int Score(ref const U value) { return Measure(value); } template <typename T> int Forward(ref const T value) { return Score<T>(value); }", "CV4176"},
		{"recursive instantiation", "profile Core; template <typename T> int Loop(T value) { return Loop<T>(value); } int Main() { return Loop<int>(7); }", "GENERIC_INSTANTIATION_RECURSIVE"},
		{"missing visitor", "profile Core; struct Pair { int left; }; struct Visitor { int total; }; concept Visit<T, V> { requires void VisitEdges(ref const T value, ref V visitor); } requires Visit<Pair, Visitor>;", "CV4153"},
		{"missing generic witness", "profile Core; struct Item { int value; }; concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); } requires Transformable<Item>;", "CV4153"},
		{"generic witness arity", "profile Core; struct Item { int value; }; concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); } template <typename A, typename B> A Transform(ref const Item value, A seed) { return seed; } requires Transformable<Item>;", "CV4154"},
		{"generic witness result", "profile Core; struct Item { int value; }; concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); } template <typename V> int Transform(ref const Item value, V seed) { return 1; } requires Transformable<Item>;", "CV4156"},
		{"generic witness parameter", "profile Core; struct Item { int value; }; concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); } template <typename V> V Transform(ref const int value, V seed) { return seed; } requires Transformable<Item>;", "CV4155"},
		{"operation parameter shadow", "profile Core; concept Transformable<T> { requires template <typename T> T Transform(ref const T value); }", "GENERIC_PARAMETER_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".concept", tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("diagnostic = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestR7f1NestedGenericCallFromArtifact(t *testing.T) {
	provider := `module Library.Nested;
profile Core;
template <typename U> U Identity(U value) { return value; }
template <typename T> T Forward(T value) { return Identity<T>(value); }
`
	artifact := buildSemanticArtifact(t, "Library/Nested.concept", provider, nil)
	consumer := `module App;
profile Core;
import Library.Nested;
int Main() { return Forward<int>(42); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library.Nested": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "nested_artifact_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 42 ? 0 : 1; }\n")
}

func TestR7f1GenericLocalArrayFromArtifact(t *testing.T) {
	provider := `module Library.Storage;
profile Core;
template <usize N> int Work() { int<array>[N] values = [7 ...]; return values[0] + Len(values); }
`
	artifact := buildSemanticArtifact(t, "Library/Storage.concept", provider, nil)
	consumer := `module App;
profile Core;
import Library.Storage;
int Main() { return Work<4>() + Work<32>(); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library.Storage": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "storage_artifact_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 50 ? 0 : 1; }\n")
}

func TestR7f1GenericRequiredOperationFromArtifact(t *testing.T) {
	provider := `module Library.Visit;
profile Core;
concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); }
template <typename T> requires Transformable<T>
int Apply(ref const T value) { return Transform<int>(ref const value, 7); }
`
	artifact := buildSemanticArtifact(t, "Library/Visit.concept", provider, nil)
	consumer := `module App;
profile Core;
import Library.Visit;
struct Item { int value; };
template <typename V> V Transform(ref const Item value, V seed) { return seed; }
requires Transformable<Item>;
int Main() { Item item = Item{3}; return Apply<Item>(ref const item); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library.Visit": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "required_artifact_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 7 ? 0 : 1; }\n")
}

func TestR7f1CombinedLibraryProbeFromArtifact(t *testing.T) {
	provider := `module Library.Probe; profile Core;
concept Transformable<T> { requires template <typename U> U Transform(ref const T value, U seed); }
struct Store { int value; };
ref struct Lease { ref const Store owner; };
Lease Borrow(ref const Store store) { return Lease{ref const store}; }
int Reclaim(owned Store store) { return store.value; }
template <typename T, usize N> requires Transformable<T>
int Apply(ref const T value) { int<array>[N] metadata = [7 ...]; return Transform<int>(ref const value, metadata[0]) + Len(metadata); }
`
	artifact := buildSemanticArtifact(t, "Library/Probe.concept", provider, nil)
	consumer := `module App; profile Core; import Library.Probe;
struct Item { int value; };
template <typename U> U Transform(ref const Item value, U seed) { return seed; }
requires Transformable<Item>;
int Main() { owned Store store = Store{5}; Item item = Item{3}; int result = 0; { Lease lease = Borrow(ref const store); result = Apply<Item, 4>(ref const item) + lease.owner.value; } return result + Reclaim(move store); }
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library.Probe": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "combined_probe_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 21 ? 0 : 1; }\n")
	invalid := strings.Replace(consumer, "result = Apply<Item, 4>(ref const item) + lease.owner.value;", "result = Apply<Item, 4>(ref const item) + lease.owner.value + Reclaim(move store);", 1)
	_, err = ParseWithSemanticModules("Invalid.concept", invalid, map[string][]byte{"Library.Probe": artifact})
	if err == nil || !strings.Contains(err.Error(), "STORAGE_MOVE_WITH_LIVE_REFERENCE") {
		t.Fatalf("combined overlapping reclaim = %v", err)
	}
}

func TestR7f1ScopedReclamationExclusion(t *testing.T) {
	valid := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "scoped_reclaim.concept")
	source, err := os.ReadFile(valid)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(valid), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "scoped_reclaim_harness.c", "#include \"scoped_reclaim.generated.h\"\nint main(void) { return concept_scoped_reclaim_main() == 30 ? 0 : 1; }\n")
	invalid := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "invalid", "overlapping_reclaim.concept")
	rejectedSource, err := os.ReadFile(invalid)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(filepath.ToSlash(invalid), string(rejectedSource))
	if err == nil || !strings.Contains(err.Error(), "STORAGE_MOVE_WITH_LIVE_REFERENCE") {
		t.Fatalf("overlapping reclaim diagnostic = %v", err)
	}
}

func TestR7f1ImportedScopedReclamationExclusion(t *testing.T) {
	provider := `module Library.Access;
profile Core;
struct Store { int value; };
ref struct Lease { ref const Store owner; };
Lease Borrow(ref const Store store) { return Lease{ref const store}; }
int Reclaim(owned Store store) { return store.value; }
`
	artifact := buildSemanticArtifact(t, "Library/Access.concept", provider, nil)
	dependencies := map[string][]byte{"Library.Access": artifact}
	valid := `module App; profile Core; import Library.Access;
int Main() { owned Store store = Store{7}; { Lease lease = Borrow(ref const store); assert(lease.owner.value == 7, "borrow"); } return Reclaim(move store); }
`
	module, err := ParseWithSemanticModules("App.concept", valid, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "imported_reclaim_harness.c", "#include \"App.generated.h\"\nint main(void) { return concept_app_main() == 7 ? 0 : 1; }\n")
	invalid := `module App; profile Core; import Library.Access;
int Main() { owned Store store = Store{7}; Lease lease = Borrow(ref const store); Reclaim(move store); return lease.owner.value; }
`
	_, err = ParseWithSemanticModules("Invalid.concept", invalid, dependencies)
	if err == nil || !strings.Contains(err.Error(), "STORAGE_MOVE_WITH_LIVE_REFERENCE") {
		t.Fatalf("imported overlapping reclaim diagnostic = %v", err)
	}
}

func TestR7f1BorrowSourcesSurviveAssignmentAndJoin(t *testing.T) {
	cases := []string{
		`profile Core; struct Store { int value; }; ref struct Lease { ref const Store owner; }; Lease Borrow(ref const Store store) { return Lease{ref const store}; } void Reclaim(owned Store store) { } int Main() { owned Store first = Store{1}; owned Store second = Store{2}; Lease held = Borrow(ref const first); held = Borrow(ref const second); Reclaim(move second); return held.owner.value; }`,
		`profile Core; struct Store { int value; }; ref struct Lease { ref const Store owner; }; Lease Borrow(ref const Store store) { return Lease{ref const store}; } void Reclaim(owned Store store) { } int Main() { owned Store first = Store{1}; owned Store second = Store{2}; Lease held = Borrow(ref const first); if (true) { held = Borrow(ref const second); } Reclaim(move second); return held.owner.value; }`,
		`profile Core; template <typename T> struct Store { T value; }; template <typename T> ref const T Borrow(ref const Store<T> store) { return ref const store.value; } template <typename T> T Reclaim(owned Store<T> store) { return store.value; } int Main() { owned Store<int> store = Store<int>{7}; ref const int held = Borrow<int>(ref const store); return Reclaim<int>(move store) + held; }`,
	}
	for i, source := range cases {
		_, err := Parse("borrow_source.concept", source)
		if err == nil || !strings.Contains(err.Error(), "STORAGE_MOVE_WITH_LIVE_REFERENCE") {
			t.Fatalf("case %d diagnostic = %v", i, err)
		}
	}
}
