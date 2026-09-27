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

`DenseStore<T, Capacity>` holds inline `Option<owned T>` slots. `Append` returns
`Result<Id<T>, StoreError>` and never removes or compacts a value. `DenseGet`
and `DenseGetMutable` return checked references. `Count` and `IdAt` support
allocation-free, deterministic iteration in append order:

```concept
for (ordinal in 0..Count<Expr, 64>(ref const exprs))
{
    Id<Expr> id = IdAt<Expr, 64>(ref const exprs, ordinal)!;
    ref const Expr value = DenseGet<Expr, 64>(ref const exprs, id)!;
}
```

`GenerationalStore<T, Capacity>` holds inline values, occupied flags,
generations, and a bounded free stack. `Insert`, `GenerationalGet`,
`GenerationalGetMutable`, `Contains`, and `Remove` are checked. Removal drops
the value, increments the generation, and pushes the slot on a LIFO free
stack. A generation at `2147483647` retires its slot. Double removal and stale
access return `StoreError::StaleId`. `Remove` declares
`compiler.InvalidatesBorrows`; release a borrow before removing an entry.

These stores contain no allocator and neither operation calls a hidden heap.
Capacity exhaustion is `StoreError::CapacityExceeded`. Invalid dense indices
return `StoreError::InvalidId`. Values with an ordinary `Drop` are destroyed
once on removal or when the store leaves scope. A failed insert leaves the
slot state unchanged. The store can hold noncopyable movable values.

## Current boundary

The inline slots are tagged options, so the payloads are not a contiguous
`Span<T>`. Iterate by ordinal and borrow one value at a time. `Span<T>` support
requires a different representation with sound partial initialization and
Drop semantics. Generic custom `foreach` iteration over an imported typed ID
range is not yet available; `Count` and `IdAt` keep the loop explicit.

Immovable values cannot pass through the current by-value `Append`/`Insert`
boundary. `Storage<T>` now constructs an aggregate initializer field by field
at its final address, including an immovable `T`. It evaluates all field
expressions before beginning that object's lifetime, so an early failure
cleans up completed field temporaries without leaving a live half-object.
The stores still need a contiguous, partially initialized backing type before
they can offer stationary insertion; the library does not fake a move.
Arena reset, a graph wrapper, hashing,
serialization, and cross-store domains are deferred. Graphs can already store
typed target IDs in node records and walk a dense edge store. No ABI layout is
promised without an explicit valid `repr(C)` declaration.

`GenerationalStore` is sparse after removal and deliberately offers no
contiguous payload Span. Its checked ID walk stays allocation-free. The
current `DenseStore` also offers no payload Span: its `Option<T>` payloads are
strided by tags rather than a contiguous `T[Count]` prefix.
