# EVT2g — checked aggregates and CMIRAMD5

Current qualification: [EVT2g5 native aggregate calls and Win64 ABI](#evt2g5--native-aggregate-calls-and-win64-abi-closeout).
EVT2g1–g4 below preserve historical boundaries. Local objects, checked references,
aggregate calls/results and the bounded C-compatible Win64 ABI slice now execute.
Ownership/Drop, general external linkage and object-file output remain excluded.

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

## EVT2g3 — native object storage progression

The source-reference stop in this historical section is superseded by EVT2g3b.

Baseline: `7ccb968ab79d4ad700db7afa396970aa78868d06`, clean at entry.
Compiler ID remains `concept-evt1-stage0-go`; target is Windows AMD64.
This stage qualifies complete local objects through the real CMIRAMD5 native
path. It does not qualify source references or begin EVT2g4 aggregate ABI work.

```powershell
go test ./internal/concept -run '^TestEVT2g3' -count=1 -v
go run ./cmd/concept machineir language/evt2g3/valid/objects.concept
go run ./cmd/concept amd64 language/evt2g3/valid/objects.concept
go run ./cmd/concept machineir language/evt2g3/boundaries/source_references.concept
```

The first native command emits 34 functions. The final command deliberately
stops at `EVT2_UNSUPPORTED_OBJECT_AUTHORITY ref Position`. Its ordinary source
and independent Normal/Verify C bootstrap execution are valid and return 9.
The older `language/evt2g/valid/objects.concept` includes aggregate function
signatures and now reaches `AMD64_UNSUPPORTED_AGGREGATE_ABI`. Neither stop
substitutes a scalar blob or an unclassified pointer signature.

### Object storage and operations

`AMD64.Objects` assigns each aggregate value a private, contiguous typed local
slot. Source storage slots retain their own identities. The existing allocator
receives only compact scalar/location virtual registers; fields do not become
independent fake aggregate registers. Whole-object size/alignment feeds the
existing local-slot/frame planner. Source-slot capacity and initialized live
values remain separate concepts in the admitted contract. Internal addresses
are derived from admitted live slots, checked field identities and checked array
indices, retaining the existing location type and readonly authority checks.

`PrepareObjectFunction` reuses allocation, scalar call planning and the unified
frame. `ValidateObjectFrame` checks exact slot size/alignment, mapping uniqueness,
containment, overlap and resolution before encoding. Module emission preflights
all functions before output/image mutation. Final frame sizes at or above 4096
bytes reject with `AMD64_STACK_PROBE_REQUIRED`; no probing is implemented.
The existing prologue, epilogue, callee saves, outgoing shadow/stack arguments,
spill areas, branch/call fixups and scratch regions remain authoritative.

The memory selector realizes initialization, slot loads/stores, aggregate
loads/stores, logical copies/member replacements, typed field/index addresses,
bounds checks and scalar field transfers. One shared memory operand writer
handles ModRM/SIB/REX and displacement encoding. It supports 1/2/4/8-byte GPR
transfers and existing width-correct scalar XMM transfers. Signed/unsigned
integer loads extend to the allocated register. Source qualification includes
bool, byte/u8, u16, int/u32, isize/u64, float and double representations.
Signed 8/16-bit extension instructions have exact byte goldens only: Core has
no corresponding ordinary source type names, so source execution is not claimed.

Nested addressing adds transported field offsets; array addressing multiplies
by the transported stride, including stride 12. Full 64-bit index comparisons
reject negative, out-of-range and high-bit indices before access. Even currently
static-safe index plans conservatively retain a native check. Failed native
bounds checks execute UD2, verified as `STATUS_ILLEGAL_INSTRUCTION` in separate
child processes rather than accepted as arbitrary crashes.

### Copies, padding and immutable updates

The initial copy policy recursively transfers meaningful scalar fields using
the checked nested descriptors and fixed-array extent/stride. It neither reads
nor writes padding. Floating fields are copied bitwise through integer memory
transfers. Copy permission comes from checked value semantics; byte width,
PlainData and CAbiValue are not substitutes. There is no memcpy call, vector
copy strategy or new ownership/relocation rule.

Private value slots provide snapshots before assignment and replacements.
Disjoint copies and self-assignment are qualified, including nested and array
values. Arbitrary partial-overlap/raw-pointer copies are not qualified and are
not expressible through this admitted local-value path. `with` uses the existing
logical copy and member replacement operations; there is no native WITH opcode.
The `WithOrder` native test records constructor/replacement calls as 1234 and
returns 43, proving base-once evaluation and ordered replacements. Its controlled
test-only foreign stub uses the existing writable-image test mechanism; it does
not add production external linkage. Record fields remain immutable.

### Reference and lifetime boundary

Internal slot/field/index locations execute and mutate their original storage.
They are not the source `ref T` / `ref const T` contract. Checked aggregate LIR
still rejects those source authorities. In v5, location parameters, call
arguments and returns are prohibited by `BridgeValidate5`; locations must have
a local live-slot root. It has no admitted cross-function borrow origin,
access-authority and lifetime contract. Treating a borrow as an ordinary scalar
pointer would erase those distinctions; copying its target would break identity.

The next isolated step is checked source-borrow LIR and an explicitly versioned
transport correction, followed by native parameter/call/address realization and
mutable/const/field/array-element identity and escape tests. This stage leaves
that boundary explicit rather than widening CMIRAMD5 silently. Existing source
escape, immutability, illegal-copy and initialization negatives continue to run
in EVT2g1/g2 tests. Mutable/const source reference execution, reference aliasing
and escape in native code remain **unqualified**. Raw Storage/Initialize/Destroy,
owned/Drop/move states, device spaces, automata/object mixing and unrestricted
pointers remain outside this slice. Aggregate by-value/indirect/sret ABI is
EVT2g4 work, independently gated before emission.

### Does Explicit Layout Actually Pay Off?

For this native slice, yes: **zero new record-layout algorithms**. The corpus
transports 21 aggregate layouts with 35 checked field descriptors; native code
consumes them at 98 field derivations and 12 array-index operations. Nested
composition uses supplied offsets and array strides (8, 12 and 16 in the
fixtures), rather than source field names, AST access or C packing conventions.
The 109,556-byte artifact is sufficient for backend compilation/emission and
native execution without the domain source/header or source-oracle functions.
Separate native/C comparisons establish semantic behavior. Existing EVT2g2
semantic-layout summaries and compiled C-representation probes remain in the
regression suite; malformed field offsets and missing copy permission reject
before output/image mutation.

Cathedral computes complete-object placement and native address displacements;
it does not recompute record field placement. Fixtures exercise object alignments
1, 2, 4 and 8; the existing frame base is 16-byte aligned. The admitted alignment
limit remains 16, without a claim of source over-alignment qualification. Layout
digests remain checked producer identities, not cryptographic authentication.
The remaining source-borrow metadata gap is an authority/lifetime gap rather
than missing record geometry.

### Proofs, bounds and architecture

`ObjectStorageFacts` checks PlainData for the immutable `FrameSlot` descriptor
and NoAllocation for projection/copy. `NativeScalarSSEFacts` adds NoAllocation
checks for whole-object emission and memory selection. Artifact-imported
PlainData is proven for immutable Pair/Group/RecordSamples, and StaticExtent<4>
for SampleArray. Mutable Position deliberately remains Unknown under the
existing record-only predicate rule. Copyable mutable values do not widen that
predicate. Native runtime checks are not counted as semantic proofs.

The bounded implementation permits 64 total local slots (source plus private
values), 128 compact scalar/location virtuals, 512 projected instructions,
eight fixed scalar parameters/arguments and copy traversal depth 16. It retains
the existing register pools, floating spills and explicit register-exhaustion
failure. Slots are not reused across aggregate values. These are implementation
limits, not general language limits.

All native object policy is Concept-owned: `AMD64.Objects`, `AMD64`, shared
`AMD64.Encoding`, and the existing frame/allocator/call modules. Stage-0 changes
are checked aggregate-array/index input, existing scalar representation mapping,
portable C short-circuit prelude ordering, a single generated range-type guard,
CLI presentation and independent tests. The short-circuit correction is checked
with guarded negative indices and side effects in Normal/Verify. The range guard
permits independently generated C oracle/backend headers to coexist. No Go
frame planner, field copier, object encoder or aggregate ABI classifier was added.

CMIRAMD5/version 5 is unchanged, including generated codecs and schema. Frozen
schema hashes are:

| Version | SHA-256 |
| --- | --- |
| CMIRAMD3 | `3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312` |
| CMIRAMD4 | `aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c` |
| CMIRAMD5 | `7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60` |

### Native evidence and measurements

`TestEVT2g3NativeObjects` first builds an artifact-only executable without the
domain oracle, then a separate strict C11 executable comparing actual Win64
native calls with the independent C bootstrap over 100 input vectors, in both
modes. Executable memory changes from RW to RX; replay is not execution evidence.
The corpus covers simple/nested records, fixed/nested arrays, padding, mixed
GPR/XMM fields, 1/2/4/8/12/16/24-byte object copies, self-assignment, immutable
updates, loops, early returns, scalar calls and live objects across an outgoing
eight-argument call. Eleven small encoding goldens pin load/store/address/copy
and signed extension bytes. Malformed geometry, copy authority and invalid frame
slots reject; a valid large object reaches the unprobed-frame rejection.

Each mode emits a 34-function, 32,504-byte image. Within each mode, 100 repeated
emissions are identical, as are 100 descriptions of the representative nested
array frame (including slots, offsets and value mapping). Frame sizes range from
40 to 328 bytes; the simple object is 72 bytes. Representative measurements:

| Function | Frame bytes | Local slots | Native code bytes |
| --- | ---: | ---: | ---: |
| Simple | 72 | 2 | 717 |
| Copy | 168 | 6 | 2064 |
| Padding | 136 | 4 | 1296 |
| With | 152 | 7 | 1564 |
| NestedArray | 328 | 8 | 2797 |
| ArrayStride12 | 136 | 5 | 1533 |
| LiveStackCall | 168 | 3 | 1401 |

One local acceptance run measured about 55 ms for combined planning/encoding,
and about 1.28 s for inspecting all function plans. Inspection revalidates the
whole artifact per function; this is not an isolated planning benchmark or a
speedup claim. Full per-function measurements and emitted plans are in local
`artifacts/evt2g3/acceptance.log` and `amd64.txt`. Final validation is recorded in
[the convergence report](../history/milestones/EVT2G-CONVERGENCE.md#evt2g3-closeout).

**MEANINGFUL PROGRESSION — native local aggregate storage, fields, copies and
immutable updates execute; source-borrow authority/lifetime needs checked LIR
and a versioned transport contract before native reference qualification.**

## EVT2g3b — checked borrow authority and native references

Baseline: `2a8b6487adbc72c800120e0340db27f0aa5ee2ac`, clean at entry.
Compiler ID: `concept-evt1-stage0-go`; native target: AMD64 Windows/Win64.

> An address locates storage. A reference names a live object and carries checked access authority.

The former `Change: EVT2_UNSUPPORTED_OBJECT_AUTHORITY ref Position` specimen
now emits CMIRAMD6 and executes natively to 9 in Normal and Verify. An independent
CLI built from an archived complete baseline revision reproduces the old stop.
The new image has three functions, two calls and 859 bytes. The broader native
corpus is [references.concept](../../language/evt2g3b/valid/references.concept).

```powershell
go test ./internal/concept -run '^TestEVT2g3b' -count=1 -v
go run ./cmd/concept lir language/evt2g3b/valid/references.concept
go run ./cmd/concept machineir language/evt2g3b/valid/references.concept
go run ./cmd/concept amd64 language/evt2g3/boundaries/source_references.concept
```

### Source authority and admitted forms

The existing Stage-0 checker establishes readable/live places, reference target
types, mutable/const access, lexical outlives constraints and escape restrictions.
`ref T` permits authorized referent mutation; `ref const T` permits reads and
qualified value copies, without granting writes. Assignment to a reference name
writes its referent. No new reference rebinding operation or qualifier is added.
Record fields retain their existing immutable projection rule.

Ordinary references are not globally exclusive or affine: the current language
permits legal aliases and ordinary mutation visible through those aliases.
Destructive operations retain the existing checked borrow conflicts. The new
source negatives pin const stores/escalation, same-sized wrong nominal targets,
local/temporary escape, uninitialized objects, use after Destroy, and declared
destructive access while a conflicting borrow is live. Native tests pass the
same object to two separate mutable reference parameters and observe writes.
No Disjoint/NoAlias fact is inferred from distinct parameter or reference IDs.

After the existing reference checks, the frontend retains an internal checked
origin witness on RefExpr. LIR consumes that witness and the validated typed body;
unknown/missing origin returns `EVT2_REFERENCE_AUTHORITY_UNKNOWN`. Local and
parameter roots are admitted. User concepts cannot construct that compiler-owned
witness or turn their propositions into liveness, outlives or exclusive-access
authority. The existing trusted-fact registry remains unchanged. These are checked
source decisions, not a new Cathedral borrow checker or new user proof primitive.

### LIR and CMIRAMD6

LIR distinguishes `ref<T>` and `ref_const<T>` from internal `ptr<T>` locations and
integer scalars. A compact contract retains the reference SSA value, checked
representation and semantic target identity, local-slot or parameter root, parent
derivation, access authority, lexical lifetime scope, origin span and provenance.
Scope IDs follow deterministic typed-body traversal. Slot scopes and instruction
sites bind uses to the containing lexical scope. Dominance checks prevent a
reference definition from being used on a path that did not establish it.

CMIRAMD6/version 6 is a new envelope containing the frozen WireObjectModule
payload and ordered per-function authority tables. It reuses supplied type/layout,
field and array geometry. It does not mutate or reserialize a weaker v5 artifact
for admission. All v3/v4/v5 schemas, generated codecs, hashes and byte contracts
remain frozen. V6 has six reflected records and three enums; the checked Concept
schema owns their order/tags, and the generator reuses imported frozen codecs.

- V6 schema SHA-256: `67d71de72ece8b4b6ce438840543e6220fd3ddcc035e2142147c679b4c77e73d`.
- Exact minimal reference artifact: 784 bytes, pinned as
  [evt2g3b_ref.hex](../../internal/concept/testdata/evt2g3b_ref.hex), SHA-256
  `eb23941570f01a685fe922386a9ecf9b16b24aff18eaf1058fd58b492f7106af`.
- The 29-function corpus transports 164 location/reference rows and 58 lexical
  scopes. Its 121,629 bytes comprise a 44-byte header, 80,370-byte embedded object
  payload and 41,215 bytes of authority metadata. The embedded payload is not
  claimed to be independently valid v5 input when it contains reference calls.

Existing slot/field/index address, load/store and direct-call operations remain
canonical. A LIR borrow normalizes to logical CopyValue of a location, with no
aggregate-copy permission. Only v6 admits that capability-preserving reference
copy. Internal place locations cannot silently become reference arguments;
explicit checked borrow or parameter evidence is required. A canonical semantic
target digest survives representation erasure and is compared across calls.
Aggregate nominal/layout identity is also retained; equal size never matches types.

The Concept v6 verifier checks target/layout/root bindings, inherited authority,
source-operation correspondence, scope ancestry, definition dominance, bounds,
initialization and call compatibility. Reference stores require writable authority.
Location returns, stores into value slots/aggregates and unclassified escapes are
rejected. Existing object validation is shared through an explicit context;
legacy v5 entrypoints always use the empty context and retain their old limits.
This verifies preservation/consistency of checked input, not the complete source
borrow graph. Evidence tags and digests identify the producer's checked claims;
they are not artifact authentication or a way to trust arbitrary user assertions.

### Native realization and object lifetime

After admission, references use pointer-sized Address virtuals in the existing
GPR allocator. They are not aggregate scalar blobs. The existing Win64 positional
call planner places their physical addresses in registers or stack arguments;
the callee retains a typed reference contract and never needs caller frame offsets.
There is no new Go native reference lowering, allocator, ABI planner or encoder.

Slot/field/index addresses reuse LEA and the existing checked offset/stride
machinery. Typed dereferences reuse existing GPR/XMM loads/stores and extension
rules. Borrow copies preserve the address, while explicit source aggregate copies
create independent memory values. Whole-object assignment through a mutable
reference updates the original storage. Frame placement remains the existing
unified model: source/private slots are distinct and not coalesced, so storage
remains stable across valid borrows, nested scopes, calls and early returns.
This is a property of these storage instances, not a global stable-address type law.

Only complete initialized local objects and live reference parameters are admitted.
Storage capacity alone does not create an object. Raw Storage/Initialize/Destroy,
owned/move/Drop lowering, reference-bearing aggregate copies, arbitrary escapes,
general standalone array/layout views and hardware address spaces remain outside
this qualification. Existing source lifetime rules still reject known invalid uses.
Valid source reference returns currently stop at `LIR_REFERENCE_RETURN_UNSUPPORTED`;
they need a separately qualified return-lifetime contract. Aggregate by-value,
indirect and sret parameters/results still stop at
`AMD64_UNSUPPORTED_AGGREGATE_ABI`, including in a valid v6 module. EVT2g4 is not begun.

The metadata limits are 128 functions/scopes/blocks, 512 location/reference rows
per function, 64 source slots and 1024 instruction sites. Existing native limits
remain 64 total source/private slots, 128 compact scalar/location virtuals,
512 projected instructions, eight scalar/address parameters/arguments and frames
below 4096 bytes. Overflow fails explicitly; no backend heap fallback is added.
NoAllocation assertions cover v6 admission and roundtrip. A reference-type
PlainData assertion is pinned as DISPROVEN; pointer width grants no raw transport
permission. Fixed-array borrows retain checked extent/stride and runtime guards.

### Does explicit layout still pay off? What about `layout`?

Reference lowering adds zero record packing rules and no reference-specific
layout engine. Nested field and array-element borrows use the existing checked
parent/member identities, offsets, element types, extents and strides. The
selected array index is evaluated when the borrow is constructed; changing the
index variable afterwards does not retarget the reference.

The source `layout` keyword is useful for static storage-region maps, alignment,
typed bind views and disjoint-region facts. Examples are
[fixed storage binding](../../language/evt1/storage/layout-stream/valid/layout_bind_fixed_storage.concept)
and [aligned regions](../../language/evt1/semantic-facts/valid/fact_disjoint_layout_regions.concept).
It could describe a fixed backend workspace or ABI probe buffer. A native frame's
locals/spills/saves/outgoing arguments vary per function, so the current planner
continues to own final frame placement. Layout geometry does not prove object
liveness or borrow authority. Native layout/bind-view admission is a separate
extension, not a new prerequisite or second planner in this milestone.

### Evidence and measurements

Normal/Verify tests execute 29 native functions in a 23,820-byte image, first
without any domain source/header/C oracle, then with a separately compiled strict
C11 logical oracle over 100 varied vectors. They cover local mutable/const
references, integer/float/double fields, nested fields, fixed-array elements,
legal local and parameter aliases, copy versus reference behavior, mutable/const
reference parameters, nested calls, live references across scalar calls, early
returns, whole-object replacement/snapshot and stable storage across shadowed
lexical scopes. Separate children require UD2 status for negative/out-of-range
element indices. Replay is not counted as native execution.

Independently compiled C probes call native scalar-reference functions with
pointer arguments in positions 0, 4 and 7, verifying mutation of the original
int32/double objects. Generated calls also mix references with floating arguments.
This qualifies those pointer-parameter placements, not complete external object ABI.
The exact minimal artifact executes a signed scalar reference read without any
aggregate type table entry. The original three-function boundary returns 9 natively.

One hundred fresh LIR/artifact/debug generations match; each mode performs 100
identical native emissions and representative frame/allocation descriptions.
Normal and Verify images are compared byte-for-byte. Valid artifacts roundtrip
100 times through the source-free Concept codec in each mode. Twenty-seven
corruptions reject deterministically, including authority, target, semantic call
target, root/provenance, layout/field/stride, scope/lifetime, known-not-live object,
call/result, version/hash, tags/counts, address space and truncation. Roundtrip
and native failures preserve output/image sentinels.

| Function | Frame bytes | Local/private slots | Native code bytes |
| --- | ---: | ---: | ---: |
| Add | 40 | 0 | 171 |
| Change | 72 | 0 | 407 |
| Local | 88 | 2 | 911 |
| Element | 376 | 6 | 2733 |
| AliasParameters | 40 | 0 | 456 |
| StackScalar | 104 | 0 | 363 |
| WholeObject | 136 | 3 | 883 |
| ScopeStable | 168 | 4 | 1875 |
| SelectedElement | 360 | 7 | 3305 |

Frames range from 40–376 bytes. Representative local combined admission/planning/
encoding observations are about 55–58 ms; all-function inspection is about 1.3 s
and revalidates the artifact per function. The harness separately times repeated
native reference calls including their checked body and host-loop overhead;
ten million calls took 34–35 ms in one local run. This is not an isolated ABI
cost or a broad speed claim. Final timing and gate
results are in [the closeout report](../history/milestones/EVT2G-CONVERGENCE.md#evt2g3b-closeout).

Full Go regressions, vet, focused race, Standard/DragonGod in both modes, package
build, generated inspection, source format/lint and diff checks pass. Frozen Zig
suites are skipped because their compiler/build/test paths are unchanged.

**SUCCESS — EVT2g3b Cathedral checked borrow authority and native references qualified.**

## EVT2g4 — Win64 aggregate ABI planning

Baseline: `e7c10a81324abb7511ef4a42d49d27e31f543f4b`, clean checkout;
compiler `concept-evt1-stage0-go`; target AMD64 Windows, Microsoft default x64
calling convention. This stage qualifies derived classification, storage and
physical call/frame plans. It does **not** qualify native aggregate call encoding
or complete external ABI conformance.

### Artifact and authority

CMIRAMD6 remains version 6 with schema hash
`67d71de72ece8b4b6ce438840543e6220fd3ddcc035e2142147c679b4c77e73d`.
Its existing semantic aggregate values, call arguments, results, checked layouts
and separate reference capabilities suffice. No schema, generated codec or
physical ABI metadata was added; CMIRAMD3/4/5 remain frozen. V5 aggregate-only
artifacts also use the new planner without changing their semantic contract.

`Win64AggregateABI.concept` is the only aggregate classifier. It reads checked
type/layout identities, representation, size, alignment and copy evidence.
`AMD64.Aggregates.concept` owns typed signature, temporary, result and transfer
plans and their verification. `AMD64.Objects` extends the existing object
projection to synthesize physical carriers. The existing scalar allocator,
Win64 positional planner, parallel-move replay and `FinalizedFrame` remain the
physical owners. Go adds CLI presentation and tests, with no production ABI
classifier, temporary allocator, hidden-pointer algorithm or packing engine.

### Classification contract

| Semantic input | Classification | Physical carrier / condition |
| --- | --- | --- |
| Qualified copyable record argument, exact size 1/2/4/8 bytes | Direct integer aggregate argument | Same-width unsigned GPR bits or outgoing stack slot; fields stay one aggregate |
| Other qualified record argument sizes, including 3/5/6/7/12/16/24 | Indirect by-value argument | Pointer to distinct caller-created temporary; alignment 16 |
| Eligible checked C record result, exact size 1/2/4/8 | Direct aggregate return | RAX payload with exact logical width; distinct from scalar source values |
| Other qualified record result | Hidden aggregate return | Caller-owned typed result storage, hidden pointer at physical zero/RCX, same pointer returned in RAX |
| `ref T` / `ref const T` | Checked reference | Existing Address carrier to original live object; no by-value temporary |
| Scalar / void | Existing scalar / void class | Existing integer/bool/GPR or floating/XMM policy |

Direct return qualification is stricter than argument size eligibility. It
requires recursively checked `repr(C)` record structure, scalar/fixed-array
members and checked copy permission. Native records lack that C structural
evidence and conservatively use hidden returns, even when small. This bounded
internal convention is not an external C++ ABI claim. Existing source C-repr
admission excludes ownership, references and unsupported representations;
the classifier does not turn PlainData into CAbiValue or add innate trust.
Standalone array ABI, constructors/destructors/inheritance, vectorcall, varargs,
indirect calls, move-only/Drop objects and arbitrary escaping values remain out
of scope. Fixed arrays inside qualified records are covered.

### Physical sequence and storage

One physical ordinal drives GPR, XMM and outgoing-stack placement. For
`TwentyFour Shifted(int a, double b, int c, float d)` the plan is exactly:

| Physical position | Value | Location |
| ---: | --- | --- |
| 0 | Hidden result pointer | RCX |
| 1 | a | RDX |
| 2 | b | XMM2 |
| 3 | c | R9 |
| 4 | d | outgoing stack slot 0 |

`Blend(Eight, TwentyFour, double, ref const Four, float, int)` additionally
combines a direct small aggregate, indirect value, float and explicit reference
under a hidden return. The result pointer is part of the ordered argument
sequence. It has no independent hidden-pointer side channel.

Each indirect value argument has its source value ID, checked type/layout,
distinct temporary slot, exact object size, alignment 16, carrier vreg and
call-covering start/end points. Initialization is a typed meaningful-field copy,
following EVT2g3 `CopyObjectFields` semantics. Padding is never meaningful copy
data. Each aggregate parameter has private materialization storage, so mutable
value parameters cannot acquire authority over the caller's original object.
Explicit reference parameters use the original reference path instead.

Direct carriers pack/unpack meaningful fields; carrier padding may be zeroed
without becoming semantic object data. Direct results are captured before
caller-preservation reloads. Hidden results bind a stable typed result object;
the return transfer must use the same incoming hidden pointer. Every return
block has the corresponding result transfer and the shared inverse epilogue.
Nested call results remain in separate private slots before subsequent argument
copies. Storage is conservatively distinct; there is no lifetime-based reuse
optimizer or general destructor elaboration.

Source slots, private values, argument temporaries and result storage share the
existing `LocalSlots` region with the same frame as saves, spills, move temps,
outgoing arguments and one 32-byte shadow area. The maximum outgoing area is
reserved once. Finalized post-prologue RSP is 16-byte aligned; local slot offsets
enforce temporary alignment and the existing frame verifier rejects overlaps.
Frames at or above 4096 bytes stop at `AMD64_STACK_PROBE_REQUIRED`.

The shared scalar allocator also permits its existing callee-save GPR fallback
for leaf entry pressure. Its first five choices are unchanged. A real scalar
CMIRAMD4 native specimen exercises six incoming integer arguments, a saved
register and `Sum(1,2,3,4,5,6) == 21`; this is not an aggregate native-call claim.

### Evidence and stage boundaries

`language/evt2g4/valid/aggregate_abi.concept` has 29 functions. The matrix covers
1/2/4/8-byte direct arguments/returns; 3/5/6/7/12/16/24-byte indirect values;
padding, nested records, fixed-array records, a small native hidden return,
mixed int/bool/float/double/aggregate/reference signatures, stack arguments,
hidden shifts, multiple calls, two early returns and nested aggregate calls.

| Stage | Qualification |
| --- | --- |
| Checked semantic aggregate artifact | CMIRAMD6, 54,171 bytes; no new version; V5 regression artifact also planned |
| Classification and physical call plan | Artifact-only Concept execution, source/AST/domain header absent |
| Frame and typed transfer plans | Verified against admitted types, values, source slots, physical projection and shared frame; structural mutation isolation and padding replay |
| Independent C classification | 14 real Win64 C/LLVM compiler signatures agree with literal expected direct/indirect and result classes |
| Determinism | 100 plans per function per mode; classifications, argument order/placement, hidden pointers, storage/lifetimes, frame regions, moves and transfer/debug fields identical |
| Normal / Verify | Same semantic plan fingerprints and frame geometry |
| New aggregate native emission | Not qualified; precise next boundary before bytes/image mutation |
| Existing scalar/object/reference native execution | Retained by the existing native regression lanes |
| Full external aggregate ABI execution | Deferred to EVT2g5 |

Independent fixtures are in `language/evt2g4/probes/win64_aggregate.c`, compiled
with `clang --target=x86_64-pc-windows-msvc -std=c11 -pedantic-errors -S -emit-llvm -O0`.
The 14 C record classifications agree. This Clang version nevertheless gives a
24-byte caller temporary natural alignment 8 (assembly uses `[rsp+40]`), unlike
Microsoft's documented 16-byte caller-temporary rule. Cathedral enforces 16.
This discrepancy is retained as evidence, not silently treated as external
alignment conformance. The target contract is
[Microsoft x64 calling convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170).

Derived-plan corruptions reject wrong physical shifts/XMM positions, direct
width, return eligibility, stale layouts, by-value/reference confusion,
original-storage aliasing, shortened lifetimes, missing/altered copies,
temporary alignment, invalid hidden pointer, outgoing area, frame overlap and
capacity. Missing layout lookup is a separate typed error. Valid source boundary
specimens pin eight source parameters plus hidden pointer against the existing
eight-physical-argument bound, and a large frame against the probing boundary.
Existing v6 malformed-reference admissions remain exact and output preserving.

The bounded plan holds 32 calls and 256 typed transfers; existing projection
limits remain 64 source/private/temp objects, 128 physical vregs, 512 projected
instructions and eight physical arguments (including hidden parameters). No heap
fallback is introduced. Resident `AggregateFunctionPlan` metadata is 78,016 bytes
on the tested C host; Driver's five active call records occupy 10,820 bytes.

Representative frames range from 40 to 552 bytes: Shifted 56, Blend 120,
Mixed 104, Early 136, Chain 120, Driver 552, Isolation 168, DirectChain 104,
StackArguments 168 and CallMixed 184. Informational local observations for
1600 classifications were 55–66 ms; 100 projections including temporary planning
were 57–65 ms; 100 full Driver call/frame plans were 184–215 ms. These include
their reader/classification work and are not isolated ABI or superiority claims.

`concept amd64 language/evt2g4/valid/aggregate_abi.concept` prints the derived
classifications, hidden shifts, frame, storage/lifetimes, moves and typed
transfers, then stops at `AMD64_UNSUPPORTED_AGGREGATE_CALL_EMISSION`.
The native byte buffer and `NativeImage` stay untouched. General external
linkage, final generated caller/callee execution, object output, unwind/SEH,
ownership/Drop and broader native ABI closure are not added in this stage.

Validation and final result are recorded in [EVT2g4 closeout](../history/milestones/EVT2G-CONVERGENCE.md#evt2g4-closeout).

Full Go regression, semantic corpus, Golden Normal/Verify, GPU-free Vulkan and
existing native integer/scalar/automata/object/reference lanes pass. Focused race,
vet, Standard/DragonGod Normal/Verify, package build, backend admission, actual CLI
planning, formatting, lint and diff checks pass. Frozen Zig suites are skipped
because their compiler/build/test infrastructure is unchanged.

**SUCCESS — EVT2g4 Win64 aggregate ABI classification and verified call/frame planning qualified. Native aggregate ABI execution and external closure remain EVT2g5.**

## EVT2g5 — native aggregate calls and Win64 ABI closeout

Baseline: `bcfe25c52627127873242e91a077a14dbe7e186b`, clean checkout;
compiler `concept-evt1-stage0-go`; target Windows AMD64 / Microsoft default x64
ABI. EVT2g4's classifier, eligibility rules, physical plans and unified frame
remain the authority. This stage realizes those plans and qualifies actual
native execution, checked-source C parity and independent C ABI boundaries.

### Implementation and contracts

`AMD64.AggregateEmission.concept` selects typed entry, argument, result and
return transfers. The direct path zero-initializes an eight-byte physical
carrier workspace, uses existing `CopyObjectFields` operations to copy meaningful
fields into it, and sends the exact qualified carrier through the existing
scalar call actions. Parameter and result materialization copy meaningful fields
out of that carrier. Source object padding is never read as semantic data.
Direct carrier padding is zeroed as a physical policy, not a C semantic guarantee.

Indirect arguments initialize distinct typed caller temporaries with that same
field copier, then place their addresses in the EVT2g4 destinations. Entry
transfers materialize private callee parameter values. Explicit references retain
their existing address/authority path and continue to name the original object.
No pointer-sized carrier is itself proof of copy permission, object liveness,
reference authority or NoAlias.

Hidden returns bind caller result storage, shift all explicit integer/floating/
reference arguments by one position, construct the meaningful result fields and
return the designated pointer in RAX. The caller checks that RAX denotes its
planned result slot. Direct result field capture happens immediately after CALL,
before preservation reloads. Every early return follows the same typed return
contract and shared inverse epilogue. Recursive and forward/backward calls use
the existing semantic symbol identities and rel32 resolver.

`AMD64.concept` and the existing scalar frame/call infrastructure remain the
instruction, move, spill/save and fixup owners. `EncodeCheckedAggregateFunction`
calls the existing EVT2g4 plan validator before selecting bytes. Aggregate module
preflight uses the same byte writer and selectors in count-only mode, checking
capacity, opcodes, branches and symbol/fixup geometry before either caller output
is changed. Measured images are never marked executable/finalized. The subsequent
write pass uses the same immutable admitted input and plans; only its completed,
resolved image is published. There is no heap fallback or separate aggregate
stack allocator.

| Signature class | Qualified realization |
| --- | --- |
| Copyable record arguments of exact size 1/2/4/8 | Integer carriers in positional RCX/RDX/R8/R9 or outgoing stack slots |
| Other qualified record argument sizes | Caller-owned aligned indirect value temporaries, private callee materialization |
| Structurally eligible checked C record results of size 1/2/4/8 | RAX payload, exact meaningful fields |
| Other qualified record results | Hidden caller storage at physical position zero/RCX; same pointer in RAX |
| Mixed int/bool, float32/float64, checked refs and aggregates | Existing positional GPR/XMM policy; stack after physical position three |
| Native records without C direct-return evidence | Existing conservative internal hidden-return convention; no arbitrary C++ interoperability claim |

Eight physical arguments remain the capacity. Hidden results consume one of
them, so their supported source signatures have at most seven explicit arguments.
Standalone array ABI, ownership-bearing/noncopyable/reference-bearing aggregate
copies, classes, varargs, indirect calls, general reference returns and layout-
erased values retain their explicit source/native boundaries. No source ownership
or borrow rule was added or weakened.

CMIRAMD3/4/5/6 schemas, versions, hashes, generated codecs, tags and exact wire
specimens are unchanged. CMIRAMD6 remains v6 with hash
`67d71de72ece8b4b6ce438840543e6220fd3ddcc035e2142147c679b4c77e73d`.
The existing artifact already carries all required semantic values and identities.
No RCX/RAX choices or native placement is serialized back into it. Checked layout
geometry remains supplied upstream; no field offsets or packing are inferred.

### Native corpus and C semantic parity

`language/evt2g5/valid/calls.concept` contains 38 functions covering direct integer
and floating-member aggregate carriers, odd-size and padding-bearing records,
nested records, fixed arrays, native small hidden results, scalar/ref mixtures,
all register positions and stack arguments, two indirect temporaries, nested
results, direct/hidden recursive forwarding, early returns, immutable `with`,
mutable value-versus-reference isolation and pressure/spill cases.

`TestEVT2g5NativeExecution` first runs with the domain source and generated domain
header absent, using only the CMIRAMD6 bytes and Concept-authored backend compiled
through the bootstrap C path. It then links a separate strict-C11 oracle generated
from the checked source. Each mode compares 100 varied finite inputs and logical
fields/mutations, with bitwise floating identity where required. Negative zero,
infinities and a quiet NaN payload pass through an eight-byte aggregate unchanged.
Padding is not compared as semantic value data.

`TestEVT2g5NativeSmoke` executes the prior EVT2g4 29-function artifact. Additional
100-vector calls cover 5/6/7-byte records, 12/16-byte records and an eight-byte
fixed-array record. The odd-size source also compiles through the checked-source
C oracle. Bounds failures on aggregate array parameters are child-process UD2
checks, with timeout cleanup.

`near_frame.concept` independently executes a 1352-byte C record with 168 double
fields and a byte tag. Its three private object regions and existing scratch
produce a 4088-byte native frame, the largest aligned frame below the existing
4096-byte probing boundary. One hundred physical-sret calls verify returned
pointer identity, every field, caller guards and unchanged destination padding.
The code image is 9398 bytes. Larger frames still reject with
`AMD64_STACK_PROBE_REQUIRED` before output mutation.

### Independent Microsoft x64 C ABI probes

`language/evt2g5/probes/probe.c` is separately compiled as a DLL with both MSVC
and Clang in Microsoft x64 mode. It includes no Cathedral types, classifier,
frame description or generated domain source. `boundary.asm`, assembled by
ml64, observes registers/stack addresses before C parameter materialization and
checks all eight nonvolatile GPRs and XMM6–XMM15 with full-width sentinels.

| Direction / observation | Actual evidence in each compiler and mode |
| --- | --- |
| Compiled C caller to unmodified Cathedral callee | 100 varied calls: direct 1/2/4/8, floating member bits, hidden return, mixed GPR/XMM/ref/stack, eight explicit arguments, recursion and array parameters |
| Cathedral caller to compiled C callee | 100 varied calls through admitted internal call sites redirected by test-only tail thunks: direct 1/2/8 identities, mixed direct four/indirect 24, sret, shifted int/double/float, explicit mutable/const references |
| By-value temporary versus reference | Actual indirect pointer differs from original ref address; mutation of the temporary leaves original value fields intact; authorized reference mutation remains visible |
| Native outgoing temporary alignment | Observed raw argument pointer is 0 modulo 16 |
| Stack / shadow | Entry RSP is 8 modulo 16, giving aligned caller call sites; the 32-byte shadow area is writable without damaging stack arguments |
| Hidden result return | Independently compiled physical-signature caller receives exactly its supplied result pointer in RAX; guarded result fields are correct |
| Register preservation | Native nested/recursive execution preserves full-width RBX/RBP/RSI/RDI/R12–R15 and XMM6–XMM15 sentinels |

The test-only absolute tail thunks redirect already resolved internal symbols to
the independent DLL; they do not introduce a production external-symbol linker,
classifier, move planner or call encoder. Native byte determinism is measured
on the unmodified emitted image, before probe redirection.

MSVC was found at `VC/Tools/MSVC/14.51.36231/bin/Hostx64/x64/cl.exe` (file version
19.51.36260.0); Clang 22.1.3 uses `clang-cl.exe`, target
`x86_64-pc-windows-msvc`, with the MSVC environment and runtime. Both execute
the same independently authored C contract. The complex C caller probe observed
16-byte-aligned incoming temporaries with both compilers. The earlier simple
Clang probe still shows alignment 8 / `[rsp+40]`; the new observation does not
resolve or erase that compiler-specific discrepancy. Cathedral consistently
enforces its documented outgoing align-16 policy. The external target remains
[Microsoft's default x64 calling convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention?view=msvc-170), not SysV or vectorcall.

### Determinism, rejection and measurements

One hundred fresh parses/checks/LIR/MachineIR generations agree. Each mode also
performs 100 fresh native emissions and plans, compares the full bytes, symbol
order/offsets/sizes, call fixups and representative pressure-case allocation,
frame and move descriptions. Normal/Verify native images are byte-identical.

| Artifact / native image | Size / identity |
| --- | --- |
| Final EVT2g5 CMIRAMD6 corpus | 144387 bytes; SHA256 `5a1b3c2d0a55e382a84bf502757c8e93ddebd2c302079bab1fd9a3f98cabbc69` |
| EVT2g5 native module | 38 functions, 21 calls, 40381 bytes; SHA256 `41c1e7e01c3dd80e8c1b46f131cd85812feff64a780adf4c3e4816b363c46a4b` |
| EVT2g4 artifact now executed | 29 functions, 12 calls, 15859 bytes |
| Near-probing-boundary native specimen | Frame 4088; image 9398 bytes |
| Existing EVT2g3 / EVT2g3b images | Retained at 32504 / 23820 bytes; original reference specimen still returns 9 |

The main corpus frames range from 40–344 bytes. Pressure has a 312-byte frame
with 104 spill bytes; FloatLive, Scenario and BoolDriver also exercise typed
floating spills. CallShifted and CallMixed preserve the incoming result pointer
across a nested hidden-return call. Combined local admission/planning/count-only
preflight/emission observations were roughly 109–114 ms for the 38-function
image in a focused run. These are informational combined costs, not isolated
allocation/ABI timings or a performance-superiority claim.

Eighteen malformed derived plans are rejected by the production checked encoder
before byte/image/workspace mutation. Cases include direct width, structural
return eligibility, missing/wrong hidden pointer or storage, shifted positions,
XMM placement, original/temporary aliasing, both reference/value reinterpretations,
temporary alignment and size, frame overlap, stale layout, wrong result carrier,
shortened lifetime, missing copy, plan capacity and outgoing area.

Eight additional malformed aggregate artifacts repeat rejection 100 times per
mode: scalar/aggregate source-class confusion, stale value layout, missing copy
permission, contradictory geometry, call argument type, call target, call result
type and malformed parameter capacity. Existing 27 reference corruptions and
the prior object/bridge corpus remain intact. Output/image sentinels prove
admission and preflight failures do not publish partial native code. The old
valid-aggregate-stop tests now assert successful emission plus atomic insufficient-
buffer rejection. Physical argument/frame-probing boundaries remain explicit.

Admission checks supplied layout relationships and identities, not producer
authentication. A padding-only size alteration can remain geometrically legal;
the malformed geometry test uses a genuine parent/field contradiction instead.
No second packing engine was added to invent a different layout authority.

### Bootstrap correction and validation

Independent oracle compilation exposed duplicate natural built-in array C
typedefs in two generated headers (`concept_array_8_byte`). The existing Go C
header owner now guards canonical natural built-in wrappers. Nominal element
types and layout-strengthened storage are not folded into that rule. Only two
portable-C checked headers and their manifests required regeneration, through
the checked-output test; frozen bridge/native bytes remain unchanged. No Go
production native ABI classification or physical emission algorithm was added.

Final commands, outcomes and closeout status are recorded in
[the EVT2g5 convergence report](../history/milestones/EVT2G-CONVERGENCE.md#evt2g5-closeout).
The full Go regression passed (compiler 1391.101 s, CLI 278.923 s, GPU-free Vulkan
profile 0.478 s), as did the focused EVT2g5 race run (572.161 s), vet, checked
outputs, Standard's 44 tests and DragonGod's 23 tests per mode. Frozen Zig lanes
were skipped under the unchanged compiler/build/test infrastructure policy.
Ownership/Drop, general raw storage, escaping reference returns, reference-bearing
copies, C++ object ABI, varargs, indirect calls, general external linkage,
COFF/object output, unwind/SEH, SysV and AArch64 remain outside this qualification.

**SUCCESS — EVT2g5 native aggregate call/return execution and the bounded Microsoft x64 C ABI slice qualified.**
