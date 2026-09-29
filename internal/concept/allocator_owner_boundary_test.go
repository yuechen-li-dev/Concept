package concept

import (
	"errors"
	"testing"
)

func TestGenericTypedBindBoundaryResolved(t *testing.T) {
	if _, err := Parse("r6k_allocator_feasibility.concept", `profile Core;
struct SystemMemory {}
template <typename T>
Storage<T> BindOne(ref T<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<T>();
    return bind<T>(address, extent);
}
`); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerFieldDestroyBoundaryResolved(t *testing.T) {
	if _, err := Parse("r6k_allocator_feasibility.concept", `profile Core;
struct SystemMemory {}
struct Owner { Storage<int> storage; }
void UseOwner(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<int>();
    Storage<int> storage = bind<int>(address, extent);
    Initialize(storage, 7);
    Owner owner = Owner{move storage};
    Destroy(owner.storage);
}
`); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerFieldMoveBoundaryResolved(t *testing.T) {
	if _, err := Parse("r6k_allocator_feasibility.concept", `profile Core;
struct SystemMemory {}
struct Owner { Storage<int> storage; }
void UseOwner(ref int<array>[1] backing)
{
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = SizeOf<int>();
    Storage<int> storage = bind<int>(address, extent);
    Initialize(storage, 7);
    Owner owner = Owner{move storage};
    Storage<int> recovered = move owner.storage;
    Destroy(recovered);
}
`); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerMoveWithOutstandingReferenceRejectsConservatively(t *testing.T) {
	_, err := Parse("r6k_allocator_feasibility.concept", `profile Core;
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
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "STORAGE_MOVE_WITH_LIVE_REFERENCE" {
		t.Fatalf("expected STORAGE_MOVE_WITH_LIVE_REFERENCE, got %v", err)
	}
}
