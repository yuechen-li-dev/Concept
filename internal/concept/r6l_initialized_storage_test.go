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

func TestR6lResultExtractionNestedGenericOwnerRemainsBlocked(t *testing.T) {
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

Result<int, MakeError> Read(MemoryRegion<SystemMemory> region)
{
    Owner<int> owner = MakeOwner<int>(region, 9)?;
    ref int value = owner.Value();
    int result = value;
    Destroy(owner.storage);
    return Result::Ok(result);
}
`
	_, err = ParseWithSemanticModules("typed_owner_result_return.concept", source, map[string][]byte{"Standard.MemoryGeometry": artifact})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4106" || !strings.Contains(diagnostic.Message, "expected Owner<int> but got Owner<T>") {
		t.Fatalf("expected pinned nested generic carrier blocker, got %v", err)
	}
}

func TestR6lImportedGenericOwnerInstantiationRemainsBlockedDeterministically(t *testing.T) {
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
	_, err = ParseWithSemanticModules("typed_owner_module.concept", consumer, map[string][]byte{"Standard.MemoryGeometry": geometry, "Standard.TypedStorage": artifact})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4148" {
		t.Fatalf("expected pinned imported generic storage blocker, got %v", err)
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
