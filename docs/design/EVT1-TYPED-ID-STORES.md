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
immovable `T` needs an in-place construction mechanism before this store can
own it. These are the next concrete blockers for full R7n success.
