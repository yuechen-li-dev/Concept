package concept

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestR6lOpenGenericBindAcceptsTrustedMemoryRegion(t *testing.T) {
	library, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(library), nil)
	source := `module App; profile Core; import Standard.MemoryGeometry;
template <typename T>
Storage<T> BindStorage(MemoryRegion<SystemMemory> region)
{
    return bind<T>(region);
}
Storage<int> Use(MemoryRegion<SystemMemory> region)
{
    return BindStorage<int>(region);
}
`
	if _, err := ParseWithSemanticModules("storage_generic_bind.concept", source, map[string][]byte{"Standard.MemoryGeometry": artifact}); err != nil {
		t.Fatal(err)
	}
}

const r6lOwnerPrelude = `profile Core;
struct SystemMemory {}
template <typename T>
class InitializedOwner
{
public:
    Storage<T> storage;
    ref T Value(ref InitializedOwner self) { return Value(self.storage); }
    ref const T Value(ref const InitializedOwner self) { return Value(self.storage); }
};
`

func TestR6lGenericOwnerInitializesMovesBorrowsAndDestroysField(t *testing.T) {
	source := r6lOwnerPrelude + `
int Main()
{
    int<array>[1] backing = [0];
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    Storage<int> storage = bind<int>(address, SizeOf<int>());
    Initialize(storage, 17);
    InitializedOwner<int> first = InitializedOwner<int>{move storage};
    InitializedOwner<int> second = move first;
    ref int value = second.Value();
    int result = value;
    Destroy(second.storage);
    ref int reinitialized = Initialize(second.storage, 23);
    result = result + reinitialized;
    Destroy(second.storage);
    return result;
}
`
	module, err := Parse("typed_owner_basic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["typed_owner_basic.generated.c"])
	if strings.Contains(body, "malloc") || strings.Contains(body, "memcpy") {
		t.Fatalf("owner lowering introduced allocation or object relocation:\n%s", body)
	}
	if !strings.Contains(body, ".storage") {
		t.Fatalf("owner storage field was not lowered ordinarily:\n%s", body)
	}
}

func TestR6lOwnerStorageStateRejectsInvalidTransitions(t *testing.T) {
	cases := []struct {
		name string
		body string
		code string
	}{
		{"double_initialize", "ref int again = Initialize(owner.storage, 2);", "STORAGE_DOUBLE_INITIALIZE"},
		{"double_destroy", "Destroy(owner.storage); Destroy(owner.storage);", "STORAGE_DESTROY_UNINITIALIZED"},
		{"value_after_destroy", "Destroy(owner.storage); ref int bad = owner.Value();", "STORAGE_VALUE_UNINITIALIZED"},
		{"maybe_initialized", "Destroy(owner.storage); if (true) { Initialize(owner.storage, 2); } ref int bad = owner.Value();", "STORAGE_VALUE_MAYBE_UNINITIALIZED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := r6lOwnerPrelude + `
void Check(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    Storage<int> storage = bind<int>(address, SizeOf<int>());
    Initialize(storage, 1);
    InitializedOwner<int> owner = InitializedOwner<int>{move storage};
` + tc.body + `
}
`
			_, err := Parse(tc.name+".concept", source)
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestR6lStorageFieldMoveTransfersObjectState(t *testing.T) {
	source := r6lOwnerPrelude + `
void Check(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    Storage<int> storage = bind<int>(address, SizeOf<int>());
    Initialize(storage, 1);
    InitializedOwner<int> owner = InitializedOwner<int>{move storage};
    Storage<int> recovered = move owner.storage;
    Destroy(recovered);
}
`
	if _, err := Parse("storage_field_move.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestR6lGenericOwnerDropDefinition(t *testing.T) {
	source := r6lOwnerPrelude + `
template <typename T>
void Drop(owned InitializedOwner<T> owner)
{
    Destroy(owner.storage);
}
int Main()
{
    int<array>[1] backing = [0];
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    Storage<int> storage = bind<int>(address, SizeOf<int>());
    Initialize(storage, 4);
    owned InitializedOwner<int> owner = InitializedOwner<int>{move storage};
    return 0;
}
`
	module, err := Parse("typed_owner_drop.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["typed_owner_drop.generated.c"]), "concept_template_drop__int") {
		t.Fatal("generic owner Drop was not instantiated")
	}
}

func TestR6mResultExtractionClosesNestedGenericOwner(t *testing.T) {
	library, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(library), nil)
	source := `module App; profile Core; import Standard.MemoryGeometry;
enum MakeError { Failed }
template <typename T>
class Owner
{
public:
    Storage<T> storage;
    ref T Value(ref Owner self) { return Value(self.storage); }
};
template <typename T>
Result<Owner<T>, MakeError> MakeOwner(MemoryRegion<SystemMemory> region, T value)
{
    Storage<T> storage = bind<T>(region);
    Initialize(storage, move value);
    Owner<T> owner = Owner<T>{move storage};
    return Result::Ok(move owner);
}
template <typename T>
Option<Owner<T>> MaybeOwner(MemoryRegion<SystemMemory> region, T value)
{
    Storage<T> storage = bind<T>(region);
    Initialize(storage, move value);
    Owner<T> owner = Owner<T>{move storage};
    return Option::Some(move owner);
}

Result<int, MakeError> Read(MemoryRegion<SystemMemory> region)
{
    Owner<int> owner = MakeOwner<int>(region, 9)?;
    ref int value = owner.Value();
    int result = value;
    Destroy(owner.storage);
    return Result::Ok(result);
}
Option<int> ReadOptional(MemoryRegion<SystemMemory> region)
{
    Owner<int> owner = MaybeOwner<int>(region, 11)?;
    ref int value = owner.Value();
    int result = value;
    Destroy(owner.storage);
    return Option::Some(result);
}
int Main()
{
    int<array>[1] backing = [0];
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    MemoryRegion<SystemMemory> region = MemoryRegion<SystemMemory>{address, SizeOf<int>(), AlignOf<int>()};
    return Read(region)! + ReadOptional(region)!;
}
`
	module, err := ParseWithSemanticModules("typed_owner_result_return.concept", source, map[string][]byte{"Standard.MemoryGeometry": artifact})
	if err != nil {
		t.Fatalf("nested generic carrier did not close: %v", err)
	}
	for _, template := range module.Templates {
		if template.Name != "MakeOwner" {
			continue
		}
		storage := template.Body.Statements[0].(*VarDecl)
		bind := storage.Value.(*TemplateCallExpr)
		if storage.Type.String() != "Storage<T>" || bind.TypeArg.String() != "T" {
			t.Fatalf("open template was mutated by concrete instantiation: storage=%s bind=%s", storage.Type.String(), bind.TypeArg.String())
		}
		instance, instantiateErr := evt1InstantiateTemplateFunctionArgs(template, template.Parameters, []Type{{Name: "int", Kind: TypeBuiltin}})
		if instantiateErr != nil {
			t.Fatal(instantiateErr)
		}
		closedStorage := instance.Body.Statements[0].(*VarDecl)
		closedBind := closedStorage.Value.(*TemplateCallExpr)
		if closedStorage.Type.String() != "Storage<int>" || closedBind.TypeArg.String() != "int" {
			t.Fatalf("function body did not substitute structurally: storage=%s bind=%s", closedStorage.Type.String(), closedBind.TypeArg.String())
		}
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatalf("nested generic carrier did not lower: %v", err)
	}
	for name, body := range outputs {
		if strings.Contains(string(body), "Owner<T>") || strings.Contains(string(body), "Storage<T>") {
			t.Fatalf("%s retains an unresolved generic placeholder", name)
		}
	}
	for run := 1; run < 100; run++ {
		again, generateErr := Generate(module, []byte(source))
		if generateErr != nil {
			t.Fatalf("nested generic generation run %d: %v", run, generateErr)
		}
		if len(again) != len(outputs) {
			t.Fatalf("nested generic output count changed on run %d", run)
		}
		for name, expected := range outputs {
			if !bytes.Equal(again[name], expected) {
				t.Fatalf("nested generic output %s changed on run %d", name, run)
			}
		}
	}
	runFoundationNativeHarness(t, outputs, "typed_owner_result_return_harness.c", "#include \"typed_owner_result_return.generated.h\"\nint main(void) { return concept_typed_owner_result_return_main() == 20 ? 0 : 1; }\n")
}

func TestR6mImportedGenericOwnerInstantiationClosesDeterministically(t *testing.T) {
	geometrySource, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	geometry := buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(geometrySource), nil)
	library := `module Standard.TypedStorage; profile Core; import Standard.MemoryGeometry;
template <typename T>
class Owner
{
public:
    Storage<T> storage;
    ref T Value(ref Owner self) { return Value(self.storage); }
};
template <typename T>
Owner<T> MakeOwner(MemoryRegion<SystemMemory> region, T value)
{
    Storage<T> storage = bind<T>(region);
    Initialize(storage, move value);
    Owner<T> owner = Owner<T>{move storage};
    return move owner;
}

template <typename T>
void Drop(owned Owner<T> owner) { Destroy(owner.storage); }
`
	deps := map[string][]byte{"Standard.MemoryGeometry": geometry}
	artifact := buildSemanticArtifact(t, "Standard/TypedStorage.concept", library, deps)
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "Standard/TypedStorage.concept", library, deps); !bytes.Equal(got, artifact) {
			t.Fatalf("typed owner artifact changed on run %d", run)
		}
	}
	consumer := `module App; profile Core; import Standard.MemoryGeometry; import Standard.TypedStorage;
struct Widget { int value; }
int Read(MemoryRegion<SystemMemory> region)
{
    owned Owner<Widget> owner = MakeOwner<Widget>(region, Widget{31});
    ref Widget widget = owner.Value();
    return widget.value;
}
`
	module, err := ParseWithSemanticModules("typed_owner_module.concept", consumer, map[string][]byte{"Standard.MemoryGeometry": geometry, "Standard.TypedStorage": artifact})
	if err != nil {
		t.Fatalf("imported generic owner did not close: %v", err)
	}
	foundClosedField := false
	for _, decl := range module.Structs {
		if decl.Name == "Owner<Widget>" && len(decl.Fields) == 1 {
			foundClosedField = decl.Fields[0].Type.String() == "Storage<Widget>"
		}
	}
	if !foundClosedField {
		t.Fatal("imported Owner<Widget> did not materialize a Storage<Widget> field")
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatalf("imported generic owner did not lower: %v", err)
	}
	for name, body := range outputs {
		if strings.Contains(string(body), "Owner<T>") || strings.Contains(string(body), "Storage<T>") {
			t.Fatalf("%s retains an unresolved imported generic placeholder", name)
		}
	}
}

func TestR6mAllocationOwnerShapeIsStructurallyReady(t *testing.T) {
	geometrySource, err := os.ReadFile("../../language/evt1/tooling/modules/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	geometry := buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(geometrySource), nil)
	source := `module App; profile Core; import Standard.MemoryGeometry;
enum MakeError { Failed }
struct LocalPolicy { int marker; }
template <typename T, typename TAllocator>
ref struct AllocationOwner
{
public:
    ref TAllocator allocator;
    Storage<T> storage;
};
template <typename T, typename TAllocator>
Result<AllocationOwner<T, TAllocator>, MakeError> MakeAllocationOwner(ref TAllocator allocator, MemoryRegion<SystemMemory> region, T value)
{
    Storage<T> storage = bind<T>(region);
    Initialize(storage, move value);
    AllocationOwner<T, TAllocator> owner = AllocationOwner<T, TAllocator>{ref allocator, move storage};
    return Result::Ok(move owner);
}
Result<int, MakeError> Use(ref LocalPolicy policy, MemoryRegion<SystemMemory> region)
{
    AllocationOwner<int, LocalPolicy> owner = MakeAllocationOwner<int, LocalPolicy>(ref policy, region, 17)?;
    ref int value = Value(owner.storage);
    int result = value;
    Destroy(owner.storage);
    return Result::Ok(result);
}
`
	module, err := ParseWithSemanticModules("allocation_owner_ready.concept", source, map[string][]byte{"Standard.MemoryGeometry": geometry})
	if err != nil {
		t.Fatalf("allocator owner shape retains an open parameter: %v", err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatalf("allocator owner shape did not lower: %v", err)
	}
}

func TestR6lOwnerDropExecutesExactlyOnceInStrictC11(t *testing.T) {
	source := `profile Core;
struct SystemMemory {}
extern "C" void ObserveDrop(int value);
struct Resource { int value; }
void Drop(owned Resource value) { ObserveDrop(value.value); }
template <typename T>
class Owner
{
public:
    Storage<T> storage;
    ref T Value(ref Owner self) { return Value(self.storage); }
};
template <typename T>
void Drop(owned Owner<T> owner) { Destroy(owner.storage); }
int Main()
{
    int<array>[1] backing = [0];
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    Storage<Resource> storage = bind<Resource>(address, SizeOf<Resource>());
    Initialize(storage, Resource{7});
    owned Owner<Resource> owner = Owner<Resource>{move storage};
    ref Resource value = owner.Value();
    return value.value;
}
`
	module, err := Parse("typed_owner_native.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["typed_owner_native.generated.c"])
	if strings.Contains(body, "malloc") || strings.Contains(body, "memcpy") {
		t.Fatalf("native owner path contains hidden allocation or object relocation:\n%s", body)
	}
	harness := `#include "typed_owner_native.generated.h"
static int drops = 0;
static int dropped_value = 0;
void ObserveDrop(int value) { drops += 1; dropped_value = value; }
int main(void) {
  int value = concept_typed_owner_native_main();
  return value == 7 && drops == 1 && dropped_value == 7 ? 0 : 1;
}
`
	runFoundationNativeHarness(t, outputs, "typed_owner_native_harness.c", harness)
}
