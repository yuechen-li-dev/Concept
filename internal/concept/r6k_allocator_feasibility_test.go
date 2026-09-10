package concept

import (
	"errors"
	"testing"
)

func requireR6kDiagnostic(t *testing.T, source, code string) {
	t.Helper()
	_, err := Parse("r6k_allocator_feasibility.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestR6kGenericTypedBindIsNotYetAnOrdinaryTemplateOperation(t *testing.T) {
	requireR6kDiagnostic(t, `profile Core;
struct SystemMemory {}
template <typename T>
Storage<T> BindOne(ref T<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<T>();
    return bind<T>(address, extent);
}
`, "CV4148")
}

func TestR6kInitializedStorageCannotYetBeDestroyedThroughOwnerField(t *testing.T) {
	requireR6kDiagnostic(t, `profile Core;
struct SystemMemory {}
struct Owner { Storage<int> storage; }
void UseOwner(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<int>();
    Storage<int> storage = bind<int>(address, extent);
    ref int value = Initialize(storage, 7);
    Owner owner = Owner{move storage};
    Destroy(owner.storage);
}
`, "CV4027")
}

func TestR6kInitializedStorageCannotBeRecoveredByMovingOwnerField(t *testing.T) {
	requireR6kDiagnostic(t, `profile Core;
struct SystemMemory {}
struct Owner { Storage<int> storage; }
void UseOwner(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<int>();
    Storage<int> storage = bind<int>(address, extent);
    ref int value = Initialize(storage, 7);
    Owner owner = Owner{move storage};
    Storage<int> recovered = move owner.storage;
    Destroy(recovered);
}
`, "CV4507")
}

func TestR6kInitializedValueReferenceCannotYetEnterReturnedOwner(t *testing.T) {
	requireR6kDiagnostic(t, `profile Core;
struct SystemMemory {}
ref struct Owner { Storage<int> storage; ref int value; }
Owner MakeOwner(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<int>();
    Storage<int> storage = bind<int>(address, extent);
    ref int value = Initialize(storage, 7);
    return Owner{move storage, ref value};
}
`, "CV4521")
}
