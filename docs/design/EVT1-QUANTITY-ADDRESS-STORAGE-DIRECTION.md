# EVT1 quantity, address, and storage direction

R6g establishes one direction:

```text
numeric scalar + dimension/unit -> quantity
quantity displacement + nominal space -> affine address geometry
address + byte extent/alignment -> ordinary MemoryRegion
region geometry + T -> Storage<T>
Storage<T> + initialization -> live T / ref T
```

Oct is the quantity engine's semantic ancestor. Concept keeps compact
normalized dimensions and compile-time algebra, adapts authoring to
`usize<byte>`, and extends scale identity for exact bit/byte conversion. No
parallel annotation engine or symbolic CAS is introduced.

Address geometry deliberately departs from C++ pointers. `Address<Space> + 3`
is invalid because no byte displacement or object scaling was stated. Ordinary
libraries spell element offset as `index * SizeOf<T>()`.

`MemoryRegion` remains library meaning. `Storage<T>` is the smallest
compiler-known static view at the live-object boundary, not an owner or runtime
descriptor. Existing initialized/uninitialized tracking authorizes
Initialize/Destroy. Strict C11 sees only scalar quantities, `uintptr_t`,
ordinary structs, and a compiler-generated `T*` inside bind lowering.

Function templates now accept ordered type and dimensionless `usize` non-type
parameters. Identity includes every argument and `concept-module.v1` preserves
the list for consumer-side instantiation without source reparse. Variadics and
metaprogramming remain deferred.

Allocator work remains deferred until the semantic-fact owner transports region
origin, interval, alignment, and lifetime through ordinary values and can
distinguish a trusted platform region from arbitrary reconstructed bits.
