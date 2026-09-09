package concept

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclaredAllocatesDisprovesNoAllocationThroughLocalCalls(t *testing.T) {
	source := `profile Core;

extern "C"
byte* Acquire(usize size);
requires compiler.Allocates(Acquire);

int Prepare(usize size)
{
    byte* storage = Acquire(size);
    return 1;
}

int Verify()
{
    Assert.Concept<NoAllocation>(
        Prepare,
        "preparation must not allocate"
    );
    return 0;
}
`
	_, err := Parse("effect_local_propagation.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_DISPROVEN" || diagnostic.Proof == nil {
		t.Fatalf("expected disproven allocation proof, got %v", err)
	}
	rendered := RenderProofVerbose(*diagnostic.Proof)
	for _, want := range []string{"Prepare", "Acquire Allocates", "ExternalContractEffect"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("allocation proof omitted %q:\n%s", want, rendered)
		}
	}
}

func TestExternCUsesSourceSymbolAndExplicitABIType(t *testing.T) {
	source := `profile Core;

extern "C"
byte* ConceptHostAllocate(usize size);
requires compiler.Allocates(ConceptHostAllocate);

int Acquire(usize size)
{
    byte* storage = ConceptHostAllocate(size);
    return 1;
}
`
	module, err := Parse("extern_c_effect.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["extern_c_effect.generated.h"])
	body := string(outputs["extern_c_effect.generated.c"])
	if !strings.Contains(header, "uint8_t* ConceptHostAllocate(size_t size);") {
		t.Fatalf("extern prototype did not preserve source symbol:\n%s", header)
	}
	if !strings.Contains(body, "ConceptHostAllocate(") {
		t.Fatalf("extern call did not preserve source symbol:\n%s", body)
	}
	if !strings.Contains(string(outputs["extern_c_effect.mir.json"]), `"allocation_effect_origin": "ExternalContractEffect"`) {
		t.Fatal("MIR omitted external allocation contract")
	}
}

func TestExternCLinksAndRunsThroughStrictC11Host(t *testing.T) {
	compiler, err := exec.LookPath("zig")
	if err != nil {
		t.Skip("zig is not installed")
	}
	source := `profile Core;
extern "C"
byte* ConceptHostAllocate(usize size);
requires compiler.Allocates(ConceptHostAllocate);
int Acquire(usize size)
{
    byte* storage = ConceptHostAllocate(size);
    return 1;
}
`
	module, err := Parse("extern_c_host.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, contents := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			if err := os.WriteFile(filepath.Join(dir, name), contents, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	host := `#include "extern_c_host.generated.h"
static uint8_t storage[64];
uint8_t* ConceptHostAllocate(size_t size) { return size <= sizeof(storage) ? storage : 0; }
int main(void) { return concept_extern_c_host_acquire(8) == 1 ? 0 : 1; }
`
	hostPath := filepath.Join(dir, "host.c")
	if err := os.WriteFile(hostPath, []byte(host), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "extern_c_host.exe")
	command := exec.Command(compiler, "cc", "-std=c11", "-Wall", "-Wextra", "-I", dir, filepath.Join(dir, "extern_c_host.generated.c"), hostPath, "-o", exe)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 host compile failed: %v\n%s", err, output)
	}
	if output, err := exec.Command(exe).CombinedOutput(); err != nil {
		t.Fatalf("strict C11 host execution failed: %v\n%s", err, output)
	}
}

func TestNoAllocationCannotBeDeclared(t *testing.T) {
	source := `profile Core;
int Pure() { return 1; }
requires compiler.NoAllocation(Pure);
`
	_, err := Parse("effect_negative_lie.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "OPERATION_EFFECT_NEGATIVE_LIE" {
		t.Fatalf("expected negative-effect rejection, got %v", err)
	}
}

func TestExternCRejectsUnsupportedAggregateABI(t *testing.T) {
	source := `profile Core;
struct Payload { int value; }
extern "C"
Payload Exchange(Payload value);
`
	_, err := Parse("extern_c_invalid_type.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "EXTERN_C_ABI_TYPE_INVALID" {
		t.Fatalf("expected bounded extern C ABI rejection, got %v", err)
	}
}

func TestGenericTypeLayoutQueriesUseOrdinaryLayoutAuthority(t *testing.T) {
	source := `profile Core;
struct Header
{
    byte tag;
    uint value;
}
static_assert(SizeOf<Header>() == 8, "header size");
static_assert(AlignOf<Header>() == 4, "header alignment");
usize Main() { return SizeOf<Header>(); }
`
	module, err := Parse("generic_sizeof.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["generic_sizeof.generated.c"]), "return 8;") {
		t.Fatal("SizeOf did not fold through the shared type geometry engine")
	}
}

func TestUserGenericRuntimeTypeGetsConcreteLayout(t *testing.T) {
	source := `profile Core;
template <typename T>
struct Pair
{
    T first;
    T second;
};
static_assert(SizeOf<Pair<int>>() == 8, "pair size");
static_assert(AlignOf<Pair<int>>() == 4, "pair alignment");
usize Main() { return SizeOf<Pair<int>>(); }
`
	module, err := Parse("generic_struct_basic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["generic_struct_basic.generated.h"])
	if !strings.Contains(header, "concept_pair_int_") || !strings.Contains(header, "int first;") {
		t.Fatalf("generic instance did not become a concrete C type:\n%s", header)
	}
}

func TestGenericNonTypeParameterShapesConcreteStorage(t *testing.T) {
	source := `profile Core;
template <typename T, usize Capacity>
struct FixedBuffer
{
    T<array>[Capacity] values;
};
static_assert(SizeOf<FixedBuffer<int, 4>>() == 16, "fixed buffer size");
usize Main() { return SizeOf<FixedBuffer<int, 4>>(); }
`
	module, err := Parse("generic_non_type_parameter.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["generic_non_type_parameter.generated.h"]), "data[4]") {
		t.Fatal("non-type template argument did not specialize fixed storage")
	}
}

func TestGenericRuntimeValueUsesConcreteConstruction(t *testing.T) {
	source := `profile Core;
template <typename T>
struct Box { T value; };
int Main()
{
    Box<int> box = Box<int>{42};
    return box.value;
}
`
	module, err := Parse("generic_runtime_value.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["generic_runtime_value.generated.c"])
	if !strings.Contains(body, "box.value = cv_init_") {
		t.Fatalf("generic construction was not lowered:\n%s", body)
	}
}

func TestUnconstrainedGenericFunctionInstantiatesNormally(t *testing.T) {
	source := `profile Core;
template <typename T>
T Identity(T value)
{
    return value;
}
int Main() { return Identity<int>(42); }
`
	module, err := Parse("generic_function_basic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["generic_function_basic.generated.c"])
	if !strings.Contains(body, "concept_template_identity__int") {
		t.Fatalf("unconstrained generic function did not instantiate:\n%s", body)
	}
}

func TestGenericFunctionCanQueryConcreteTypeLayout(t *testing.T) {
	source := `profile Core;
template <typename T>
usize RequiredBytes()
{
    return SizeOf<T>();
}
usize Main() { return RequiredBytes<int>(); }
`
	module, err := Parse("generic_required_bytes.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["generic_required_bytes.generated.c"])
	if !strings.Contains(body, "return 4;") {
		t.Fatalf("generic layout query did not fold after substitution:\n%s", body)
	}
}

func TestNestedGenericApplicationsShareConcreteIdentity(t *testing.T) {
	source := `profile Core;
template <typename T>
struct Box { T value; };
static_assert(SizeOf<Box<Box<int>>>() == 4, "nested box size");
usize Main() { return SizeOf<Box<Box<int>>>(); }
`
	module, err := Parse("generic_nested_application.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["generic_nested_application.generated.h"])
	if strings.Count(header, "typedef struct concept_box_int_") != 1 {
		t.Fatalf("inner generic instance was not emitted once:\n%s", header)
	}
}

func TestGenericOwnedFieldDerivesStructuralDrop(t *testing.T) {
	source := `profile Core;
struct Resource { int handle; }
void Drop(owned Resource value);
template <typename T>
struct Owner { owned T value; };
int Main()
{
    owned Owner<Resource> owner = Owner<Resource>{Resource{7}};
    return 0;
}
`
	module, err := Parse("generic_drop_field.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["generic_drop_field.generated.c"])
	if strings.Count(body, "concept_generic_drop_field_drop((owner).value);") != 1 {
		t.Fatalf("generic owned field did not derive exactly one structural drop:\n%s", body)
	}
}

func TestGenericRefFieldPreservesOrdinaryEscapeRules(t *testing.T) {
	source := `profile Core;
template <typename T>
ref struct View { ref T value; };
View<int> Bad()
{
    int local = 1;
    View<int> view = View<int>{ref local};
    return view;
}
`
	_, err := Parse("generic_ref_escape.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4521" {
		t.Fatalf("expected ordinary reference escape rejection, got %v", err)
	}
}

func TestRecursiveGenericInstantiationIsBounded(t *testing.T) {
	source := `profile Core;
template <typename T>
struct Bad { Bad<Bad<T>> value; };
int Main() { return SizeOf<Bad<int>>(); }
`
	_, err := Parse("generic_recursive_instantiation.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "GENERIC_RECURSIVE_INSTANTIATION" {
		t.Fatalf("expected bounded recursive-instantiation diagnostic, got %v", err)
	}
}

func TestGenericClassMethodUsesConcreteFieldType(t *testing.T) {
	source := `profile Core;
template <typename T>
class Holder
{
public:
    T value;
    T Get(ref const Holder self) { return self.value; }
};
int Main()
{
    Holder<int> holder = Holder<int>{9};
    return holder.Get();
}
`
	module, err := Parse("generic_class_basic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["generic_class_basic.generated.c"])
	if !strings.Contains(body, "const concept_holder_int_* self") {
		t.Fatalf("generic method did not lower with a concrete owner type:\n%s", body)
	}
}

func TestGenericAndEffectArtifactsAreByteIdenticalAcross100Runs(t *testing.T) {
	source := `profile Core;
template <typename T, usize N>
struct Storage { T<array>[N] data; };
extern "C"
byte* HostAcquire(usize size);
requires compiler.Allocates(HostAcquire);
usize Main() { return SizeOf<Storage<int, 4>>(); }
`
	module, err := Parse("generic_effect_determinism.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if len(outputs) != len(first) {
			t.Fatalf("run %d changed output count", run)
		}
		for name, expected := range first {
			if !bytes.Equal(outputs[name], expected) {
				t.Fatalf("run %d changed %s", run, name)
			}
		}
	}
}
