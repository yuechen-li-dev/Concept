package concept

import (
	"errors"
	"testing"
)

func TestR6nSecondTypeParameterConstraintRemainsUnsupported(t *testing.T) {
	_, err := Parse("r6n_second_parameter_constraint.concept", `profile Core;
concept Provider<T> { requires int Provide(ref T self, usize size); }
template <typename TValue, typename TProvider>
requires Provider<TProvider>
TValue Make(ref TProvider provider, TValue value) { return value; }
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4170" {
		t.Fatalf("expected CV4170 for a constraint on the second type parameter, got %v", err)
	}
}

func TestR6nOpenSecondTypeParameterCannotDispatchRequiredOperationUnconstrained(t *testing.T) {
	_, err := Parse("r6n_unconstrained_dispatch.concept", `profile Core;
concept Provider<T> { requires int Provide(ref T self, usize size); }
template <typename TValue, typename TProvider>
TValue Make(ref TProvider provider, TValue value)
{
    int ignored = Provide(ref provider, 1);
    return value;
}
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4027" {
		t.Fatalf("expected CV4027 for an unconstrained open operation call, got %v", err)
	}
}

func TestR6nTypedOwnerDropCannotConstrainAllocatorParameter(t *testing.T) {
	_, err := Parse("r6n_typed_owner_drop.concept", `profile Core;
struct SystemMemory {}
struct AllocationError {}
template <typename TSpace>
struct MemoryRegion { Address<TSpace> start; usize<byte> length; usize<byte> alignment; };
concept Releasable<T> {
    requires Result<void, AllocationError> Release(ref T self, MemoryRegion<SystemMemory> region);
}
template <typename TValue, typename TAllocator>
ref struct AllocationOwner
{
    ref TAllocator allocator;
    MemoryRegion<SystemMemory> region;
    Storage<TValue> storage;
};
template <typename TValue, typename TAllocator>
requires Releasable<TAllocator>
void Drop(owned AllocationOwner<TValue, TAllocator> owner) {}
`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4170" {
		t.Fatalf("expected CV4170 for typed-owner Drop's allocator constraint, got %v", err)
	}
}
