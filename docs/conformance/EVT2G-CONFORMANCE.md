# EVT2g — checked aggregate input progression

**MEANINGFUL PROGRESSION. Native aggregate/reference/Win64 aggregate ABI support is not qualified.**

The first blocker, loss of ordinary aggregate type/layout and object operations
before Cathedral, is removed at the target-independent LIR boundary. The next
blocker is explicit: frozen CMIRAMD3/4 cannot transport those semantics.
`concept machineir` and `concept amd64` reject with
`MIR_AGGREGATE_REQUIRES_CMIRAMD5`. No CMIRAMD5 schema, codec, native aggregate
storage plan, reference representation, aggregate copy algorithm or ABI
classification is implemented by this change.

## Identity and reproduction

- Actual clean baseline: `8b129ad710e254e3cdc3b647418693cca175e24c`.
- Requested EVT2f baseline: `9c6ba2ca767e7f5212047bf41ef83f32498f18b3`.
- Preserve intervening commits `70855fc` and `8b129ad` (counted span bounds,
  shorthand-capture invalidation, maximum induction-start range validation).
- Compiler: `concept-evt1-stage0-go`; native target remains AMD64 Windows.

```powershell
go run ./cmd/concept lir language/evt2g/valid/objects.concept
go run ./cmd/concept machineir language/evt2g/valid/objects.concept
go run ./cmd/concept amd64 language/evt2g/valid/objects.concept
go test ./internal/concept -run '^TestEVT2g' -count=1 -v
```

The baseline compiler built from an archived `8b129ad` source snapshot rejects
the object fixture at `EVT2_UNSUPPORTED_EXPR *concept.StructConstructExpr`.
The changed compiler emits verified aggregate LIR; both native commands then
stop at the versioned bridge requirement. See the convergence report for
validation results. Local reproducibility logs are under `artifacts/evt2g/`.

## Existing language and compiler inventory

| Surface | Existing checked/source behavior | Native boundary after this change |
| --- | --- | --- |
| `struct` | Mutable value aggregate; structural copyability | Checked LIR admitted for the bounded numeric subset; no native aggregate emission |
| `record struct` | Immutable field projections; whole mutable binding may be reassigned | Same typed layout as its checked declaration, with record kind retained |
| `with` | Fresh copy of a copyable record, base evaluated once, replacements in source order | Explicit `record_with` LIR intent; no native copy strategy selected |
| Fields and nested fields | Ordinary checked places; record, const and reference restrictions already enforced | Typed local-slot/field derivations, scalar loads/stores; no integer address conversion |
| `owned T` | Ownership, move state and Drop responsibility | Rejected by this aggregate LIR slice; no byte-copy shortcut |
| `ref T`, `ref const T` | Existing mutable/readonly live-object borrows, provenance and escape checks | Explicit `EVT2_UNSUPPORTED_OBJECT_AUTHORITY`; no source reference ABI claimed |
| `Storage<T>` / `Initialize` / `Destroy` | Existing raw/live/dead typed-storage state machine and C bootstrap lowering | Unchanged; not conflated with ordinary local initialization |
| `Span<T>`, `ReadOnlySpan<T>` | Existing typed storage views, bounds and mutability | No new ordinary-record reference/span native support |
| `SizeOf`, `AlignOf`, typed `OffsetOf` | Existing checked geometry and field layout | `evt1StructFieldOffsets` / `evt1TypeGeometry` supply the LIR snapshot |
| `PlainData` | Ordinary Concept predicate returning typed evidence/refutation | Exercised separately; never grants a native copy or foreign ABI fact |
| `CAbiValue` | Separate proposition over `compiler.CAbiLayout`; imported foreign calls also require applicable measured evidence | Neither size nor PlainData substitutes for it |
| Fixed arrays | Existing rank-one `<array>[N]` storage, contextual literals and independent repetition | Scalar array fields, initialization, guarded indexed reads/writes in LIR |
| LIR/MachineIR memory vocabulary | Existing slots, typed `ptr<...>`, `load`, `store`, `index_address`, `check_index`, automata-frame derivations | Reused; no second stack allocator or machine-frame model |
| Win64 calls | Existing Concept-owned mixed scalar positional ABI, frames, spills, fixups and native execution | Aggregate arguments/results remain blocked before scalar projection |

The mission's mutable `record struct` example is incompatible with the existing
language contract: `CV4142` rejects record-field mutation, including through
`ref R`. Use ordinary mutable `struct`, or immutable record update:

```concept
RecordPair next = current with {
    x = 9.0;
};
```

The semantic model is preserved. This is not an architectural stop: mutable
structs and `with` cover the two existing source intents without changing record
immutability.

## Checked layout and object input

`LIRAggregateLayout` is a checked-layout snapshot, not a second type system or
layout calculator. It retains defining module/nominal identity, existing
structured closed-generic identity, record/struct kind, native versus explicit
C representation, size, alignment, declaration-ordered fields and padding gaps.
The content identity includes those inputs, nested field type identities,
const-field authority, and each complete checked semantic field type before
scalar representation erasure. Same-width unit changes therefore change identity.
Source/C symbol spelling alone cannot identify an aggregate.

New target-independent operations are `make_aggregate`, `make_array`,
`initialize_slot`, `slot_address`, `field_address`, and `record_with`. Existing
typed load/store and slot operations express field access and value assignment.
No native address addition, register/blob reinterpretation or byte copy occurs
in Stage-0. Nested addresses remain derivations of the same slot, without an
intermediate aggregate load/copy.

Complete typed initialization begins the local object's LIR lifetime. A
forward must analysis rejects reads/addresses without initialization on every
reachable predecessor. Loop-local initialization is distinguished from assignment.
These are bounded local value objects without Drop or escaping addresses;
this does not qualify general reference lifetimes. Record projections and
const places and const fields carry readonly authority. Object locations cannot escape through
calls, stored pointer values or returns.

`record_with` carries the base value plus ordered replacement operands and
checked member ordinals. Base evaluation occurs once before replacement
evaluation; scalar, nested-record and fixed-array replacements are qualified.
The original is unchanged. The C bootstrap's existing `with` path now supplies
the checked expected field type to its ordinary initializer lowering, fixing
previous invalid C for array-literal replacements.

Aggregate assignment/copy remains a typed value operation admitted by existing
source copyability checks. Only recursively admitted scalar/array/aggregate
fields are accepted. Resource-bearing, immovable, class, ref-struct, table,
bitfield and unknown field representations reject. There is no memcpy/memmove
plan, overlap strategy, relocation, hidden allocation or padding-byte guarantee.
The source oracle compares fields, never padding bytes. Aggregate equality,
hashing and serialization are not added.

Bounds: 128 layout descriptors, 64 fields per descriptor, 16 nesting levels,
256 elements per admitted array field, and 1 MiB checked object size. Capacity,
offset, alignment, width and nested-type inconsistencies reject; nothing
silently truncates. These are LIR metadata limits, not native frame guarantees.

## IR, artifacts and ABI limits

CMIRAMD3 stays v3 / hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.
CMIRAMD4 stays v4 / hash
`aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c`.
Their schemas, generated Go transport and Concept readers are unchanged.

Semantic artifacts containing only imported type declarations can supply this
LIR path without dependency source. This is **not** aggregate MachineIR
artifact-only qualification. Rehashed malformed layout graphs are checked;
layout/version evidence must still be designed into the future bridge.

No native aggregate ABI case is newly qualified: small direct aggregates,
indirect by-value temporaries, explicit references, sret, hidden-result positional
shifts, mixed aggregate signatures and fifth/later aggregate arguments remain
pending. No independent Cathedral/C aggregate ABI probe is claimed. Future
classification belongs in ordinary Concept against the
[Microsoft x64 calling convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170),
not SysV rules or a production Go classifier.

No aggregate frame regions or native bytes are emitted. The existing scalar
stack-probing boundary is unchanged. This change cannot authorize a large
unprobed aggregate frame because all aggregate native input rejects first.

## Qualification evidence

`objects.concept` has 20 functions and eight admitted layouts. It exercises numeric structs and records, nested fields,
mutation, scalar array fields, repeated initialization, typed copies and
self-assignment, padding-bearing objects, source by-value isolation, aggregate
results/calls, early returns, named initialization, and `with` updates.

`TestEVT2gCheckedAggregateLIR` pins those operations and 100 fresh identical
layout/object/CFG/debug generations. `TestEVT2gRecordWithEvaluationOrder` pins
base-once and authored replacement order; a separate C source test checks an
observable `123` evaluation trace. `TestEVT2gSourceStrictC11` independently
checks C geometry and semantic fields in Normal/Verify; it does not execute
LIR or Cathedral-generated aggregate bytes.

Malformed-input tests cover stale identities/layouts, representation, field
bounds/type/offset, construction completeness, missing initialization, guarded
indexing/stride, readonly authority, and invalid `with` operands/ordinals.
Additional tests cover CFG joins, loop-local const initialization, source-only
invalid reference escapes, immovable/resource stops, imported/closed-generic
layouts, capacity/depth, PlainData proof and distinct typed C ABI refutation.

For the saved PlainData specimen, build `Standard.Semantic.Vocabulary` with
`concept build-module` into a module root, set `CONCEPT_MODULE_ROOTS` to that
root, then run `concept explain language/evt2g/valid/plain_data.concept --verbose`.
The qualified result is `PROVEN PlainData(Flags)` with typed evidence size=1,
alignment=1. The separate negative CAbiValue assertion is Disproven because the
record lacks explicit foreign representation.

Native lowering/encoding time, image size, frame size and native record-copy cost
are **unmeasured**. Informational checked-LIR timing is emitted by the focused
test; no general performance conclusion follows.

The next implementation step is a checked Concept-authored versioned aggregate
bridge, generated transport/readers, and Cathedral artifact-only validation,
before physical object storage/copy/reference/ABI realization. EVT2h is not begun.
