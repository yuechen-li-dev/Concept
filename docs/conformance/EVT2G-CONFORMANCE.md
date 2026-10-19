# EVT2g — checked aggregates and CMIRAMD5

Current contract: [EVT2g2 CMIRAMD5](#evt2g2--typed-aggregate-and-explicit-layout-contract).
The EVT2g1 report below is retained as history. Its old bridge stop is superseded
by `AMD64_UNSUPPORTED_AGGREGATE_STORAGE`; native aggregate execution remains pending.

## EVT2g1 historical report

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

## EVT2g2 — typed aggregate and explicit layout contract

Baseline: `e51c184bf51e6073652073293ed5587e26546ba6`, initially clean.
Compiler: `concept-evt1-stage0-go`. This milestone qualifies transport and
independent Concept admission, before native object placement or encoding.

> Source and semantic checking establish object layout. MachineIR transports checked layout facts. Cathedral verifies and consumes them. It does not independently derive record geometry.

> Cathedral derives storage placement and instructions from checked semantic layout. It does not derive semantic layout from raw declarations.

> One source of truth governs aggregate type geometry. Stage-0 may transport it and Cathedral may verify it, but neither backend layer independently invents another record layout.

### Reproduction and schema ownership

```powershell
go run ./cmd/machinebridgegen .
go run ./cmd/concept package build DragonGod
$env:CONCEPT_MODULE_ROOTS = (Resolve-Path artifacts/Standard/modules).Path
go run ./cmd/concept generated libraries/Standard/Backend/BridgeCodec5.concept
go run ./cmd/concept generated libraries/Standard/Backend/BridgeValidate5.concept
go run ./cmd/concept lir language/evt2g/valid/objects.concept
go run ./cmd/concept machineir language/evt2g/valid/objects.concept
go run ./cmd/concept machineir language/evt2g2/valid/mixed.concept
go run ./cmd/concept amd64 language/evt2g/valid/objects.concept
go test ./internal/concept -run '^TestEVT2g' -count=1 -v
```

The last native command intentionally fails with
`AMD64_UNSUPPORTED_AGGREGATE_STORAGE`. Tests prove that admission leaves both
the native output buffer and `NativeImage` untouched, 100 times in each mode.

| Contract | Version | Schema SHA-256 |
| --- | --- | --- |
| CMIRAMD3, frozen | 3 | `3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312` |
| CMIRAMD4, frozen | 4 | `aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c` |
| CMIRAMD5 | 5 | `7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60` |

`Standard.Backend.BridgeSchema5` is the only wire schema. Checked reflection
drives `machinebridgegen`, which emits the Go model/codec, Concept codec
derivation requests, and `EVT2-MACHINEIR-BRIDGE5.schema.json`. Existing
`BridgeDerive` and `BridgeRead` materialize ordinary checked Concept readers
and writers. No second hand-maintained codec was added. Header identity is
8 magic bytes, little-endian version, and 32 schema-hash bytes. Version/hash
mismatches reject; v4 never supplies inferred v5 metadata.

V5 has 16 records, eight enums, 94 fields, and 60 enum cases. V4 retains its
62 fields and 50 enum cases. V5 adds 46 derived declarations. The combined
v3/v4/v5 inspection count is **119**, previous **73**, delta **46**.
Standalone `BridgeCodec5` shows 87, including 41 inherited v3 declarations;
standalone `BridgeCodec4` shows 73, including its 32 local derivations.
These are distinct inspection scopes, not conflicting totals.

### Layout, values and object state

Semantic declaration/application identity is retained separately from a
deterministic digest of the complete checked LIR layout snapshot. Representation,
size/alignment, field identity/order/type/offset/size/alignment/readonly facts,
nested layout identities and explicit array element/extent/stride facts survive.
Verified semantic import identities and content hashes participate in artifact
identity, including when an imported declaration's geometry is unchanged.
Closed generic identity uses the existing structural application model.

Field selection uses type-table and field indices plus checked identity/offset
facts, never parsing dotted source spelling. Array geometry is explicit;
Cathedral does not infer stride from an integer register width. Geometry gaps
describe padding, whose contents remain unspecified and nonsemantic. Semantic
copy eligibility is `checked_value`; this authorizes logical value copying,
not memcpy, deterministic padding serialization, or a native copy algorithm.

The Go producer verifier reconstructs the checked snapshot and checks its
layout digest through the existing LIR verifier. The Concept validator checks
indexed layout identities, geometry consistency, bounds, alignment, nonoverlap,
nested references, and operation types. Opaque layout digest text is a producer
identity, **not cryptographic authentication of an untrusted artifact**.
Concept does not reconstruct the producer's JSON hash or calculate packing.
Every mutation changes the artifact bytes/digest; the malformed corpus also
requires explicit independent Concept rejection. Coherent replacement contracts
still require their checked producer provenance; geometry validation cannot
prove their source-language origin.

> Storage capacity, initialized object lifetime, aggregate value identity, and live reference identity are separate semantic properties even when their native representations involve memory addresses.

V5 values have scalar, aggregate or internal location class. Slots describe
typed `SystemMemory` capacity; complete `initialize_slot` starts the admitted
local lifetime. `load_slot`, `store_slot`, typed `slot_address`, `field_address`,
`index_address`, `load` and `store` retain types and checked provenance.
The bounded forward must analysis rejects using aggregate storage before
initialization, including CFG joins. Readonly fields/records/slots restrict
writes. Readonly scalar initialization is explicit, including loop-local
reinitialization. Merely producing an address never starts an object lifetime.

Source `ref`, `ref const`, raw Storage, owned/Drop/move-only/reference-bearing
objects remain at their EVT2g1 checked-input boundary. Internal locations are
nonescaping derivations of live slots, not source borrows or integer addresses.
There are no new source address-space conversions, destruction states, heap
allocations, or general lifetime proof reconstruction. A module mixing automata
activation frames with aggregates rejects explicitly; scalar automata/native
frame qualification remains unchanged. Standalone scalar-array parameters in a
v5 mixed object module remain outside this bounded projector; scalar v3/v4
array paths and v5 aggregate array fields retain their existing contracts.

`with` becomes one `copy_value` followed by ordered `replace_member` values.
Base/replacement evaluation already occurs once and in source order in checked
LIR. There is no WITH machine opcode. Ordinary construction, nested updates,
array replacements and untouched fields remain logical value semantics.

Calls carry exact semantic argument order, type indices, aggregate values,
result type, target identity and calling convention. The mixed fixture covers
int, bool, float32, float64 and aggregate parameters, aggregate returns, readonly
scalars and a loop. No aggregate is disguised as a GPR/XMM virtual register.

> Aggregate ABI classification is derived later from the checked type/layout contract and the selected calling convention. It is not an intrinsic property of byte width alone.

There is no hidden sret pointer, indirect temporary, small-aggregate register
classification, physical stack placement or native aggregate encoding in v5.
C representation remains separate from a `CAbiValue` certificate. PlainData
neither supplies that certificate nor grants relocation/lifetime authority.

### Artifact-only evidence and bounds

`TestEVT2g2ConceptArtifactOnlyRoundTrip` supplies only CMIRAMD5 bytes to the
independently compiled Concept validator and derived reserializer. The domain
source, AST and semantic environment are absent from its C harness. The 20-function EVT2g1 fixture, the mixed fixture and imported closed
`Box<int>` / `Box<float>` instances roundtrip exactly 100 times in Normal and
Verify. `TestEVT2g2PolicyContractParity` separately compares exact
contract bytes from both planner policies. Fresh checked production, schema
generation, encoding, decoding, debug output and native-stop diagnostics are
also repeated 100 times. Existing g1 tests pin fresh LIR/layout determinism.

Five exact hex fixtures pin transport bytes, with these whole-artifact hashes:

| Case | Bytes | SHA-256 |
| --- | --- | --- |
| Minimal record | 876 | `c7eb71a5091132ae94912844688d0c1390e29187c18b363fed89c5849ff5e329` |
| Nested record | 1771 | `eaa6fc1c847242d2aabb70d70ca6f0bf696f7602abff9aaf242a73d31622bb08` |
| Array field | 1619 | `15218f44be18a7b6613f0a90bbf5172b662c8226f95ff0cb0cd9e079a7f80c11` |
| Aggregate call | 1927 | `0889864c477789de85b29ec9307b90b32a3f9beb12aee448817829950c0c10a8` |
| Aggregate result | 1348 | `ecac85ad843907fb393155e1e64aa92b4500ccf040889368c274b7792989985d` |

The **43-case** malformed corpus covers header/version/hash/truncation/trailing
bytes, counts, type/field identities and geometry, overlap, nested references,
array stride/extent, representation/evidence/address-space/value-class/opcode
tags, copy eligibility/type, field paths, live storage, immutable authority,
source origin, call parameters/count/results and returns. Both Go and Concept
reject every case; Normal/Verify diagnostics are exact and invalid roundtrips
leave caller output untouched. Corruption never becomes a semantic Disproven
Verdict: malformed transport is a typed Result error.

| Metadata | Bound |
| --- | --- |
| Types / aggregate layouts / dependencies | 256 / 128 / 128 |
| Fields per aggregate / nesting depth / object bytes | 64 / 16 / 1,048,576 |
| Fixed array extent | 256 |
| Functions / parameters per function | 128 / 32 |
| Values / slots / blocks per function | 512 / 64 / 128 |
| Instructions per function / operands / call arguments | 1024 / 256 / 32 |
| Generic wire sequence count | 1,048,576, followed by the narrower contract bounds |

Readers retain counted byte views; bounded metadata arrays and caller buffers
avoid dynamic allocation. Actual assertions prove `PlainData` for field/value
descriptors, `NoAllocation` for validation and validated roundtrip, and
`StaticExtent<64>` for one-dimensional slot-liveness storage. These are ordinary
checked semantic Verdicts, separate from decoder Results and physical ABI claims.

### Does Explicit Layout Pay Off?

At this contract layer, **zero** Cathedral record layout calculators and **zero**
field-packing rules were needed. Eight checked aggregate summaries and 13 field
descriptors are consumed in the main fixture (13 total types including scalar
and array descriptors). Size/alignment and every transmitted field fact agree
with the canonical semantic observation. An independently compiled C11 probe
also agrees with the `repr(C)` Sample's size, alignment and two offsets.
Nested geometry is consumed without source access; offsets are checked rather
than recomputed. Mutation tests detect inconsistent geometry/identities, and
dependency changes alter artifact identity.

Checked geometry is sufficient for this contract layer. Physical placement,
copy realization, reference execution and Win64 ABI rules remain unqualified;
no speed or optimizer-quality conclusion follows. Layout authority comes from
compiler-derived checked facts, never an arbitrary user concept returning Proven.
PlainData/NoAllocation/StaticExtent assertions are Proven for the stated subjects;
aggregate native placement and C ABI classification have no proof from v5 alone.

Informational local measurements: main artifact **51,320 bytes**; 100 Go encodes
**23.4 ms**, 100 decodes **30.6 ms**; 100 fresh production/schema/roundtrip cycles
**823 ms**. Three fixtures each roundtripped 100 times by Concept took **784 ms**
Normal and **764 ms** Verify. Schema metadata is **21,503 bytes**, versus v4's
**15,982 bytes** (increase **5,521**); generated Go/Concept transport is
**54,413 / 9,769 bytes**. These are machine-specific observations, not thresholds.
An independent C `sizeof` probe reports metadata descriptors of 76 bytes
(type), 56 (field), 20 (value), 36 (slot), 96 (instruction), and 64 per slot
liveness set. The 128-block liveness store is 8,192 bytes. These describe
validator metadata; no native aggregate object or frame was allocated.

### Qualification

Detailed final gate results and closeout are recorded in
[the convergence report](../history/milestones/EVT2G-CONVERGENCE.md#evt2g2-closeout).
Local logs are under `artifacts/evt2g2/`. Frozen Zig tests are skipped because
neither the legacy compiler nor its build/test infrastructure changed.

**SUCCESS — EVT2g2 CMIRAMD5 typed aggregate and explicit layout contract qualified.**
