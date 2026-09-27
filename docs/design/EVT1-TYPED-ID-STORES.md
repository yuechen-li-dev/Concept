# EVT1 typed ID stores

R7n places the bounded toolbox in `Standard.Collection.Stores`. The nominal
generic ID carries no value reference. `DenseStore` owns values at append-only
stable positions; `GenerationalStore` owns reusable slots and validates both
index and generation. The free stack gives deterministic LIFO reuse, while a
retired exhausted generation can never alias an earlier handle.

The implementation composes ordinary generic structs, non-type capacity
parameters, partially initialized fixed storage, `Result`, `ref` returns, and
`InvalidatesBorrows`. It adds no runtime type registry or allocator. Generated
C has inline typed arrays and direct indexing. The general storage primitives
`T<raw>[N]` and `T<sparse>[N]` are compiler-recognized; the Standard store
names remain ordinary generic library types.

Three general substrate repairs came from the dogfood. An owned `Option` with a
payload lacking a custom destructor still needs move/replace authority; its
drop path is empty. `Storage<T>` retains its explicit Destroy authority and
cannot use that rule to discard storage. Matching a borrowed enum now emits pointer member access
for the tag and payload. Zero-argument failure constructors emit `(void)` in
C11. These changes serve ordinary programs independently of stores.

DenseStore now owns `T<raw>[Capacity]`: physically adjacent `T` slots and one
initialized-prefix count. Slots below count are live; the tail is uninitialized
at the Concept level. `RawAppend` or `Emplace` writes directly to slot `Count`
and increments count after success. `RawValues` borrows `data[0..count)` as an
ordinary `Span<T>` or `ReadOnlySpan<T>`. It never exposes the tail. Drop visits
the prefix in reverse order. A failing later field expression drops completed
field temporaries; count remains unchanged. The C representation uses a typed
`T data[N]` and `int count`, with C's natural element stride and alignment.

GenerationalStore now owns `T<sparse>[Capacity]`. Its typed payload array is
paired with live bits, while the library still owns generations, `next`, and
its deterministic LIFO free stack. Direct `Emplace` commits liveness and
free-list state only after construction succeeds. `SparseRemove` destroys the
selected live value once and clears its bit. Store Drop visits only live bits
in reverse slot order. The sparse store does not offer a contiguous live
payload Span. A single-store ID remains stable until removal, and a reused
slot must receive a new generation. An exhausted generation retires the slot.

The bounded storage types do not allocate. Normal borrow/provenance rules
apply to returned views. Fixed backing never reallocates. A store with an
immovable payload derives immovability from that backing; stationary insertion
does not relocate its live elements. `Append` and `Insert` remain the movable
by-value paths. An immovable store is constructed directly with
`{Uninitialized(), ...}` because a moving factory return is not valid.

## R7n2 stationary storage progression

`Initialize(Storage<T>, T{...})` now evaluates each aggregate field under the
ordinary temporary cleanup rules and writes the fields into the final bound
storage only after all expressions succeed. The generated C never creates a
complete temporary `T`. `Destroy` now follows structural field Drop when `T`
has no custom Drop, and a borrowed `ref T` is never treated as an owning
destructor target. A strict C11 specimen constructs an immovable record,
returns early from a failed later field expression, and observes one Drop for
the abandoned field and one for the successfully destroyed object.

R7n3 extends that same final-address construction to bounded store slots.
The R7n2 single-object proof remains the regression for field cleanup; the
R7n3 artifact-only store proof covers a fresh dense slot, a fresh sparse slot,
and sparse reuse under Normal and Verify.
