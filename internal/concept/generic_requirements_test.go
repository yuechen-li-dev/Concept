package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestGenericRequirementTargetsFourthParameter(t *testing.T) {
	source := `profile Core;
concept Provider<T> { requires int Provide(ref T self); }
struct A {}
struct B {}
struct C {}
struct D { int value; }
int Provide(ref D self) { return self.value; }
requires Provider<D>;
template <typename TA, typename TB, typename TC, typename TD>
requires Provider<TD>
int Get(ref TD provider) { return Provide(ref provider); }
int Use() { D provider = D{42}; return Get<A, B, C, D>(ref provider); }
`
	if _, err := Parse("fourth_parameter.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestRelationalConceptAndConcreteGenericArgument(t *testing.T) {
	source := `profile Core;
struct Source { int value; }
struct Destination { int value; }
concept Convertible<TSource, TDestination>
{
    requires TDestination Convert(TSource value);
}
Destination Convert(Source value) { return Destination{value.value}; }
requires Convertible<Source, Destination>;
template <typename TSource, typename TDestination>
requires Convertible<TSource, TDestination>
TDestination ConvertThrough(TSource value) { return Convert(value); }
template <typename T>
requires Convertible<T, Destination>
Destination ConvertToDestination(T value) { return Convert(value); }
Destination Use()
{
    Source value = Source{7};
    Destination first = ConvertThrough<Source, Destination>(value);
    return ConvertToDestination<Source>(value);
}
`
	if _, err := Parse("relational.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestComposedNestedDiamondRequirementClosureDeduplicates(t *testing.T) {
	source := `profile Core;
concept Readable<T> { requires int Read(ref T self); }
concept Writable<T> { requires void Write(ref T self, int value); }
concept ReadWrite<T> { requires Readable<T>; requires Writable<T>; }
concept Left<T> { requires ReadWrite<T>; }
concept Right<T> { requires Readable<T>; }
concept Diamond<T> { requires Left<T>; requires Right<T>; }
struct Device { int value; }
int Read(ref Device self) { return self.value; }
void Write(ref Device self, int value) { self.value = value; }
requires Diamond<Device>;
template <typename T>
requires Diamond<T>
int Update(ref T value)
{
    int current = Read(ref value);
    Write(ref value, current);
    return current;
}
int Use() { Device value = Device{3}; return Update<Device>(ref value); }
`
	module, err := Parse("diamond.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	if strings.Count(mir, `"name": "Read"`) < 1 {
		t.Fatalf("required operation closure omitted Read:\n%s", mir)
	}
}

func TestOpenGenericUndeclaredOperationRejectsImmediately(t *testing.T) {
	_, err := Parse("open_invalid.concept", `profile Core;
concept Readable<T> { requires int Read(ref T self); }
template <typename T>
requires Readable<T>
void Bad(ref T value) { Write(ref value, 1); }
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4176" {
		t.Fatalf("expected open generic requirement failure CV4176, got %v", err)
	}
}

func TestConcreteRelationalFailureNamesApplicationAndOperation(t *testing.T) {
	_, err := Parse("relational_missing.concept", `profile Core;
struct Source {}
struct Destination {}
concept Convertible<T, U> { requires U Convert(T value); }
template <typename T, typename U>
requires Convertible<T, U>
U ConvertThrough(T value) { return Convert(value); }
Destination Use() { Source value = Source{}; return ConvertThrough<Source, Destination>(value); }
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4153" || !strings.Contains(diagnostic.Message, "Convertible<Source, Destination>") || !strings.Contains(diagnostic.Message, "Destination Convert(Source)") {
		t.Fatalf("expected concrete relational missing-operation diagnostic, got %v", err)
	}
}

func TestGenericTypeConstraintTargetsSecondParameter(t *testing.T) {
	source := `profile Core;
concept Provider<T> { requires int Provide(ref T self); }
struct Device { int value; }
int Provide(ref Device self) { return self.value; }
requires Provider<Device>;
template <typename TValue, typename TProvider>
requires Provider<TProvider>
struct Owner { TValue value; TProvider provider; };
Owner<int, Device> Make();
`
	if _, err := Parse("generic_type_second_parameter.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestImportedRelationalConstraintRetainsArgumentsAndInstantiates(t *testing.T) {
	producer := `module Library;
profile Core;
concept Convertible<T, U> { requires U Convert(T value); }
template <typename T, typename U>
requires Convertible<T, U>
U ConvertThrough(T value) { return Convert(value); }
`
	artifact := buildSemanticArtifact(t, "Library.concept", producer, nil)
	_, decoded, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Concepts) != 1 || len(decoded.Concepts[0].Parameters) != 2 || len(decoded.Templates) != 1 || len(evt1ConstraintArguments(decoded.Templates[0].Constraint)) != 2 {
		t.Fatalf("semantic artifact omitted structural concept application: %#v %#v", decoded.Concepts, decoded.Templates)
	}
	consumer := `module App;
profile Core;
import Library;
struct Source { int value; }
struct Destination { int value; }
Destination Convert(Source value) { return Destination{value.value}; }
requires Convertible<Source, Destination>;
Destination Use() { Source value = Source{9}; return ConvertThrough<Source, Destination>(value); }
`
	if _, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library": artifact}); err != nil {
		t.Fatal(err)
	}
}

func TestImportedComposedConstraintBuildsClosureWithoutSource(t *testing.T) {
	producer := `module Library;
profile Core;
concept Readable<T> { requires int Read(ref T self); }
concept Writable<T> { requires void Write(ref T self, int value); }
concept ReadWrite<T> { requires Readable<T>; requires Writable<T>; }
template <typename TValue, typename TDevice>
requires ReadWrite<TDevice>
TValue Touch(ref TDevice device, TValue value)
{
    int current = Read(ref device);
    Write(ref device, current);
    return value;
}
`
	artifact := buildSemanticArtifact(t, "Library.concept", producer, nil)
	consumer := `module App;
profile Core;
import Library;
struct Device { int value; }
int Read(ref Device self) { return self.value; }
void Write(ref Device self, int value) { self.value = value; }
requires ReadWrite<Device>;
int Use() { Device device = Device{3}; return Touch<int, Device>(ref device, 8); }
`
	if _, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Library": artifact}); err != nil {
		t.Fatal(err)
	}
}

func TestRelationalModuleArtifactIsByteIdenticalAcross100Runs(t *testing.T) {
	source := `module Library;
profile Core;
concept Convertible<T, U> { requires U Convert(T value); }
template <typename T, typename U>
requires Convertible<T, U>
U ConvertThrough(T value) { return Convert(value); }
`
	first := buildSemanticArtifact(t, "Library.concept", source, nil)
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "Library.concept", source, nil); !bytes.Equal(got, first) {
			t.Fatalf("artifact changed on run %d", run)
		}
	}
}

func TestRelationalConstraintOutputsAreByteIdenticalAcross100Runs(t *testing.T) {
	source := `profile Core;
struct Source { int value; }
struct Destination { int value; }
concept Convertible<T, U> { requires U Convert(T value); }
Destination Convert(Source value) { return Destination{value.value}; }
requires Convertible<Source, Destination>;
template <typename T, typename U>
requires Convertible<T, U>
U ConvertThrough(T value) { return Convert(value); }
Destination Use() { return ConvertThrough<Source, Destination>(Source{4}); }
`
	var first Outputs
	for run := 0; run < 100; run++ {
		module, err := Parse("relational_determinism.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = outputs
		} else if !reflect.DeepEqual(outputs, first) {
			t.Fatalf("generated output changed on run %d", run)
		}
	}
}

func TestRelationalConceptProofRetainsNestedApplication(t *testing.T) {
	source := `profile Core;
struct Source { int value; }
struct Destination { int value; }
concept Converts<T, U> { requires U Convert(T value); }
concept SafeConvert<T, U> { requires Converts<T, U>; }
Destination Convert(Source value) { return Destination{value.value}; }
void Check()
{
    Assert.Concept<SafeConvert>(Source, Destination, "conversion contract must close");
}
`
	module, err := Parse("relational_proof.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal([]byte(moduleOutput(t, outputs, ".mir.json")), &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.ProofGraphs) != 1 {
		t.Fatalf("expected one proof graph, got %d", len(mir.ProofGraphs))
	}
	rendered := RenderProofVerbose(mir.ProofGraphs[0])
	if !strings.Contains(rendered, "Converts<Source, Destination>") || mir.ProofGraphs[0].Outcome != FactProven {
		t.Fatalf("relational proof omitted nested application:\n%s", rendered)
	}
	first, err := SerializeProof(mir.ProofGraphs[0])
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		got, err := SerializeProof(mir.ProofGraphs[0])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, first) {
			t.Fatalf("relational proof changed on run %d", run)
		}
	}
}

func TestOpenAndConcreteRequirementOriginsAreDistinct(t *testing.T) {
	source := `profile Core;
concept Provider<T> { requires int Provide(ref T self); }
struct Device { int value; }
int Provide(ref Device self) { return self.value; }
requires Provider<Device>;
template <typename TValue, typename TProvider>
requires Provider<TProvider>
TValue Get(ref TProvider provider, TValue value) { int ignored = Provide(ref provider); return value; }
int Use() { Device provider = Device{1}; return Get<int, Device>(ref provider, 2); }
`
	module, err := Parse("requirement_origins.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	if !strings.Contains(mir, `"origin": "GenericRequirement"`) || !strings.Contains(mir, `"origin": "ConcreteWitness"`) {
		t.Fatalf("MIR did not distinguish open assumption from concrete witness:\n%s", mir)
	}
	var decoded MIR
	if err := json.Unmarshal([]byte(mir), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Instances) != 1 {
		t.Fatalf("expected one concrete instance, got %d", len(decoded.Instances))
	}
	closed, err := json.Marshal(decoded.Instances[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(closed, []byte("TProvider")) || bytes.Contains(closed, []byte("TValue")) {
		t.Fatalf("closed instance retained bound template parameters: %s", closed)
	}
}

func TestRequiredOperationEffectIsAuthoritativeInOpenGenericMIR(t *testing.T) {
	source := `profile Core;
concept AllocatingProvider<T>
{
    requires int Provide(ref T self);
    requires compiler.Allocates(Provide);
}
template <typename TValue, typename TProvider>
requires AllocatingProvider<TProvider>
TValue Get(ref TProvider provider, TValue value)
{
    int ignored = provider.Provide();
    return value;
}
`
	module, err := Parse("required_effect.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	for _, want := range []string{`"kind": "requirement_call"`, `"may_allocate": true`, `"effect_origin": "GenericRequirement"`, `"effect": "Allocates"`} {
		if !strings.Contains(mir, want) {
			t.Fatalf("open generic effect closure omitted %s:\n%s", want, mir)
		}
	}
}

func TestNoAllocationTraversesConcreteRequiredOperation(t *testing.T) {
	_, err := Parse("required_effect_proof.concept", `profile Core;
concept AllocatingProvider<T>
{
    requires int Provide(ref T self);
    requires compiler.Allocates(Provide);
}
struct Device { int value; }
int Provide(ref Device self) { return self.value; }
requires compiler.Allocates(Provide);
requires AllocatingProvider<Device>;
template <typename TValue, typename TProvider>
requires AllocatingProvider<TProvider>
TValue Get(ref TProvider provider, TValue value)
{
    int ignored = provider.Provide();
    return value;
}
int Use()
{
    Device provider = Device{1};
    return Get<int, Device>(ref provider, 2);
}
void Verify()
{
    Assert.Concept<NoAllocation>(Use, "required allocation effect must remain visible");
}
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_DISPROVEN" || diagnostic.Proof == nil {
		t.Fatalf("expected required operation to disprove NoAllocation, got %v", err)
	}
	if rendered := RenderProofVerbose(*diagnostic.Proof); !strings.Contains(rendered, "Allocates") {
		t.Fatalf("NoAllocation proof omitted required operation effect:\n%s", rendered)
	}
}

func TestRepeatedAndConflictingRelationalRequirements(t *testing.T) {
	if _, err := Parse("same_parameter_twice.concept", `profile Core;
concept Compatible<T, U> { requires bool Same(T left, U right); }
struct Value {}
bool Same(Value left, Value right) { return true; }
requires Compatible<Value, Value>;
template <typename T>
requires Compatible<T, T>
bool Check(T left, T right) { return Same(left, right); }
bool Use() { Value value = Value{}; return Check<Value>(value, value); }
`); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("swapped_parameters.concept", `profile Core;
struct Source {}
struct Destination {}
concept Converts<T, U> { requires U Convert(T value); }
Destination Convert(Source value) { return Destination{}; }
requires Converts<Source, Destination>;
template <typename T, typename U>
requires Converts<U, T>
T Swapped(U value) { return Convert(value); }
Destination Use() { return Swapped<Destination, Source>(Source{}); }
`); err != nil {
		t.Fatalf("simultaneous concept substitution failed: %v", err)
	}

	_, err := Parse("conflicting_requirements.concept", `profile Core;
concept ReturnsInt<T> { requires int Read(ref T self); }
concept ReturnsBool<T> { requires bool Read(ref T self); }
concept Conflict<T> { requires ReturnsInt<T>; requires ReturnsBool<T>; }
template <typename T>
requires Conflict<T>
int ReadOne(ref T value) { return Read(ref value); }
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4177" {
		t.Fatalf("expected conflicting requirements to reject ambiguously, got %v", err)
	}
}
