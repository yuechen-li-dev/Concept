# Standard compiler data

`Standard.Collection.Stores` supplies bounded inline storage for compiler and
runtime data structures. Dense stores own values. Typed IDs provide stable
logical identity. References are temporary views.

## API

`Id<T>` is a nominal `int` index. `Id<Expr>` and `Id<Stmt>` are distinct types;
there is no implicit conversion. `GenerationalId<T>` adds an `int` generation.
Both representations use checked, nonnegative indices. Use `Option<Id<T>>`
when an ID may be absent. An ID is not a reference and does not own its value.
IDs from separate stores of the same `T` are not distinguished by a domain.

`DenseStore<T, Capacity>` owns contiguous inline `T<raw>[Capacity]` storage
and a live initialized prefix `[0, Count)`. There are no option tags between
payloads. `Append` returns `Result<Id<T>, StoreError>` for movable values;
`Emplace(ref store, T{...})` constructs directly in the final slot, including
for immovable values. It commits `Count` only after construction succeeds.
Capacity overflow in `Emplace` traps; `Append` reports
`StoreError::CapacityExceeded`. The store never removes or compacts a value,
so `Id<T>.index` is its physical prefix index for that store lifetime.
`DenseGet` and `DenseGetMutable` return checked references. `Count` and `IdAt`
support deterministic ID iteration:

```concept
for (ordinal in 0..Count<Expr, 64>(ref const exprs))
{
    Id<Expr> id = IdAt<Expr, 64>(ref const exprs, ordinal)!;
    ref const Expr value = DenseGet<Expr, 64>(ref const exprs, id)!;
}
```

`Values(ref store)` returns a mutable `Span<T>`; `ReadOnlyValues(ref const
store)` returns `ReadOnlySpan<T>`. Both borrow the actual payload array and
have length `Count`, never `Capacity`. Ordinary borrow rules reject appending
while a view remains live. The fixed backing does not reallocate. A store
containing immovable `T` is itself immovable, and must be constructed directly
with `DenseStore<T, N>{Uninitialized()}` rather than returned by a moving
factory. A live value is destroyed once, in reverse index order, at store
Drop. A failed initializer drops completed field temporaries and leaves the
slot outside the live prefix.

`GenerationalStore<T, Capacity>` holds stationary sparse payload slots with
live bits, generations, and a bounded free stack. `Insert`, `Emplace`, `GenerationalGet`,
`GenerationalGetMutable`, `Contains`, and `Remove` are checked. Removal drops
the value, increments the generation, and pushes the slot on a LIFO free
stack. A generation at `2147483647` retires its slot. Double removal and stale
access return `StoreError::StaleId`. `Remove` declares
`compiler.InvalidatesBorrows`; release a borrow before removing an entry.
`Emplace` constructs directly at the selected slot, then marks it live and
commits the free-list or next-slot update. Failure preserves the slot,
generation, and free-list state. Sparse slots are never compacted and do not
expose a fake contiguous payload Span. Walk indices below `next`, form a
`GenerationalId<T>` from each current generation, then use `Contains` and
`GenerationalGet` for checked access.

These stores contain no allocator and their bounded operations call no hidden heap.
Capacity exhaustion in the Result-returning paths is `StoreError::CapacityExceeded`.
Invalid dense indices
return `StoreError::InvalidId`. Values with an ordinary `Drop` are destroyed
once on removal or when the store leaves scope. A failed insert leaves the
slot state unchanged. Both stores can hold noncopyable and immovable values
through stationary construction. By-value `Append`/`Insert` still require a
movable value.

## Current boundary

`T<raw>[Capacity]` and `T<sparse>[Capacity]` are general partially initialized
storage types. Ordinary indexing is unavailable; checked raw/sparse operations
mediate access and Drop. `T<raw>` uses one live-prefix count and can produce a
real Span. `T<sparse>` has per-slot live bits and cannot produce a live payload
Span. `Uninitialized()` creates empty backing without a live `T`.

Generic custom `foreach` iteration over an imported typed ID range is not
yet available; `Count` and `IdAt` keep the loop explicit. Arena reset, a graph wrapper, hashing,
serialization, and cross-store domains are deferred. Graphs can already store
typed target IDs in node records and walk a dense edge store. No ABI layout is
promised without an explicit valid `repr(C)` declaration.

DenseStore owns a contiguous live prefix and can expose Span views.
GenerationalStore intentionally trades contiguity for stable reusable
identities and therefore does not expose a contiguous payload span.
