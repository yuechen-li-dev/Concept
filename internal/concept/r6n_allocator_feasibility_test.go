package concept

import (
	"errors"
	"testing"
)

func TestR6nSecondTypeParameterConstraintAndRequiredOperationAreSupported(t *testing.T) {
	_, err := Parse("r6n_second_parameter_constraint.concept", `profile Core;
concept Provider<T> { requires int Provide(ref T self, int size); }
struct Source { int value; }
struct ConcreteProvider { int value; }
int Provide(ref ConcreteProvider self, int size) { return self.value; }
requires Provider<ConcreteProvider>;
template <typename TValue, typename TProvider>
requires Provider<TProvider>
TValue Make(ref TProvider provider, TValue value)
{
    int ignored = provider.Provide(1);
    return value;
}
int Use()
{
    ConcreteProvider provider = ConcreteProvider{1};
    return Make<int, ConcreteProvider>(ref provider, 42);
}
`)
	if err != nil {
		t.Fatalf("expected second-parameter constraint and required operation closure to compile, got %v", err)
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

func TestR6nTypedOwnerDropCanConstrainAllocatorParameter(t *testing.T) {
	_, err := Parse("r6n_typed_owner_drop.concept", `profile Core;
struct SystemMemory {}
struct AllocationError {}
template <typename TSpace>
struct MemoryRegion { Address<TSpace> start; usize<byte> length; usize<byte> alignment; };
concept Releasable<T> {
    requires Result<void, AllocationError> Release(ref T self, borrow const MemoryRegion<SystemMemory> region);
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
void Drop(owned AllocationOwner<TValue, TAllocator> owner)
{
    owner.allocator.Release(owner.region);
}
`)
	if err != nil {
		t.Fatalf("expected typed-owner Drop constraint on its second parameter to compile, got %v", err)
	}
}
