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

R6h resolves that transport prerequisite. Address space, origin, relative
interval, extent, alignment, host accessibility, and lifetime now survive
ordinary locals, parameters/results, Result/Option payloads, fields, generic
wrappers, persistent fields, and checked module summaries. Unknown runtime
offsets degrade interval/alignment; `AddressFromBits` remains unknown.

Allocator work itself remains deferred. The allocator-readiness specimen now
demonstrates `AlignUp -> Subregion -> imported return -> generic field ->
bind<T> -> Initialize -> Destroy` without runtime fact metadata.

R6j adds the second legitimate storage-origin path: an explicitly declared
foreign result may establish an ordinary region when extent and alignment keep
their `usize<byte>` meaning and a live owner/lease supplies provenance.
Representation-level foreign scalars should stay ABI-honest; semantic units are
introduced by an ordinary Concept wrapper. `AddressFromBits` remains Unknown.
