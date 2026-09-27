# EVT1 typed ID stores

R7n places the bounded toolbox in `Standard.Collection.Stores`. The nominal
generic ID carries no value reference. `DenseStore` owns values at append-only
stable positions; `GenerationalStore` owns reusable slots and validates both
index and generation. The free stack gives deterministic LIFO reuse, while a
retired exhausted generation can never alias an earlier handle.

The implementation composes ordinary generic structs, non-type capacity
parameters, fixed arrays, `Option<owned T>`, `Result`, `ref` returns, and
`InvalidatesBorrows`. It adds no runtime type registry, allocator, or compiler
recognition of store names. Generated C has inline arrays and direct indexing.

Three general substrate repairs came from the dogfood. An owned `Option` with a
payload lacking a custom destructor still needs move/replace authority; its
drop path is empty. `Storage<T>` retains its explicit Destroy authority and
cannot use that rule to discard storage. Matching a borrowed enum now emits pointer member access
for the tag and payload. Zero-argument failure constructors emit `(void)` in
C11. These changes serve ordinary programs independently of stores.

The current representation trades an option tag per slot for sound partial
initialization and automatic Drop. This keeps insertion local and permits
noncopyable values, but it cannot promise a `Span<T>` over payloads. An
immovable `T` needs in-place construction in a store-owned slot before this
store can own it. These are the next concrete blockers for full R7n success.

## R7n2 stationary storage progression

`Initialize(Storage<T>, T{...})` now evaluates each aggregate field under the
ordinary temporary cleanup rules and writes the fields into the final bound
storage only after all expressions succeed. The generated C never creates a
complete temporary `T`. `Destroy` now follows structural field Drop when `T`
has no custom Drop, and a borrowed `ref T` is never treated as an owning
destructor target. A strict C11 specimen constructs an immovable record,
returns early from a failed later field expression, and observes one Drop for
the abandoned field and one for the successfully destroyed object.

This solves final-address construction for a single `Storage<T>` but does not
change either store's slots. `DenseStore` still contains `Option<owned T>[N]`,
whose payload stride includes the option tag. It cannot yield `Span<T>` over
the live prefix. `GenerationalStore` has intentional holes after removal and
must not yield a live payload Span. A general partially initialized inline
array with safe per-slot lifetime and store relocation rules is the remaining
substrate for DenseStore stationary insertion and contiguous views.
