# EVT1 R6m conformance: structural generic substitution

Status: implemented

Baseline HEAD: `bcd4e614a203b24691e6b756063aa63f31753904`

Compiler identity: `concept-evt1-stage0-go`

R6m centralizes recursive type substitution, keeps open applied types
structural, materializes only closed generic instances, and validates closure
before executable generic bodies reach MIR. Type and non-type bindings traverse
applied arguments, storage, arrays/ndarrays, qualifiers, and callable
signatures. Generic construction bodies retain their applied type in semantic
module payloads.

Focused structural tests cover nested applications, `Storage<T>`, qualifiers,
array/ndarray element and non-type extent bindings, callback parameters/results,
callable capture-environment fields, callable body declarations, and the
closed-instantiation rejection path.

The two R6l acceptance failures are now positive tests:

- `Result<Owner<T>, MakeError>` instantiated with `int` extracts as
  `Owner<int>` and retains initialized `Storage<int>` state;
- an imported `Owner<Widget>` materializes with a `Storage<Widget>` field from
  the compiled semantic artifact, byte-identically across 100 builds.

The local nested Result/Option owner program is also generated 100 times and
compared output-by-output, covering MIR, Plan, generated C, maps/manifests, and
proof JSON. Its generated C is compiled as strict C11 and executes with the
expected value. Generated outputs are scanned for unresolved `Owner<T>` and
`Storage<T>` spellings.

A two-type-parameter `AllocationOwner<T, TAllocator>` readiness specimen closes
both fields (`ref TAllocator` and `Storage<T>`), returns the owner through
`Result`, recovers `Value`, destroys the object, and lowers without implementing
an allocator.

No `Result`, `Option`, `Storage`, `Owner`, allocator, or module-name branch was
added. OctGen was inspected and not used because its current bounded host model
does not describe arbitrary Go record visitors; the handwritten mapper is
smaller and keeps the type algebra authoritative in Concept.

Allocator implementation, partial specialization, variadics, generic
reflection, source pointers, MMIO, and native LIR remain deferred.
