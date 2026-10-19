# EVT2g convergence — checked aggregate input

Current milestone: [EVT2g3 closeout](#evt2g3-closeout). EVT2g1/g2 below retain
their historical boundaries and evidence.

Status: **MEANINGFUL PROGRESSION**; full native aggregate qualification remains open.

Baseline: `8b129ad710e254e3cdc3b647418693cca175e24c`, clean `main`, containing
EVT2f5 `9c6ba2ca767e7f5212047bf41ef83f32498f18b3` and the two subsequent
range/bounds improvements. Compiler ID is `concept-evt1-stage0-go`.

## Removed blocker and real-path comparison

An independent baseline binary built from `git archive HEAD` rejects the
object fixture at `EVT2_UNSUPPORTED_EXPR *concept.StructConstructExpr`.
The new compiler emits verified typed aggregate LIR with checked layouts,
typed field derivations, complete initialization, copies/assignment and calls.
It also lowers immutable `with` intent, including nested and array updates.
The C bootstrap's contextual array-initializer bug in `with` is repaired.

The baseline C compiler also independently reproduces that bug:
`objects.generated.c` fails at `array_literal_requires_target`. The new
Normal/Verify strict-C11 object oracle passes, including preservation of the
original record and a separate observable base/replacement evaluation trace.

`concept machineir` and `concept amd64` now isolate the next missing contract:

```text
MIR_AGGREGATE_REQUIRES_CMIRAMD5 ...: checked record layouts and typed object operations require a versioned aggregate bridge
```

The exact source line/name depends on the fixture's declaration/function order.
This is a deliberate pre-projection gate; neither scalar bridge may erase object
semantics. CMIRAMD3/4 and generated codecs are unchanged. There is no new Go
aggregate native backend, and there are no claimed aggregate native bytes.

## Semantic decision

Existing records have immutable fields. Mutable-field examples use ordinary
copyable structs, while immutable updates use `with`. The checker continues
rejecting record mutation (`CV4142`), mutation through const reference
(`CV4513`), escaping local/field references (`CV4511`), and uninitialized reads
(`CV4560`). No convenient source semantics were invented for the milestone.

## Validation

| Gate | Local result |
| --- | --- |
| `go test ./... -count=1 -parallel=4 -timeout=30m` | PASS; main compiler package 663.931 s, CLI 92.214 s |
| Latest focused aggregate + changed EVT2f boundary + checked-output tests | PASS; includes the final const-field and semantic-type identity refinements |
| `go test -race ./internal/concept -run '^TestEVT2g' -count=1` | PASS, 8.366 s |
| `go vet ./...` | PASS |
| Semantic corpus, Golden Normal/Verify, GPU-free Vulkan, EVT2/EVT2x/EVT2e/EVT2f | PASS in the full Go regression run |
| Standard explicit CLI Normal / Verify | 44 passed, 0 failed in each mode; 3 informational benchmarks |
| DragonGod explicit CLI Normal / Verify | 23 passed, 0 failed in each mode; 1 informational benchmark |
| New object and `with` source C11 oracles | PASS in Normal / Verify, `-pedantic-errors -O2` |
| 100 fresh LIR/layout/CFG/debug generations | PASS; 20 functions, 8 layouts, 38,268 inspection bytes |
| Saved PlainData assertion and CLI explanation | PROVEN; size=1, alignment=1; separate CAbiValue refutation retained |
| `concept format <each new fixture> --check`, fixture lint, R9d tests | PASS |
| Whitespace/staged diff checks | PASS |
| New Cathedral native aggregates / references / independent aggregate ABI probes | NOT QUALIFIED; named pre-bridge stop |

The first broad run reached Go's default ten-minute package timeout with
`TestDenseImmovableArtifactEmplace` only three seconds into its own execution.
It also exposed two old EVT2f diagnostic assertions: aggregate records now
reach the explicit versioned-bridge gate instead of `EVT2_UNSUPPORTED_TYPE`.
Those assertions were updated to require the new rejection. The longer full
run passes without skipped/disabled failing gates. Final checked-layout guard
refinements are covered by the latest focused and race lanes; no physical
native algorithm changed after the broad run was dispatched.

One informational focused observation was 1.044 s for 100 complete checked LIR
generations/inspections while other validation was running. This includes the
existing semantic/MIR/planning work and is not an isolated native benchmark.
Native lowering/encoding/copy time, image size and aggregate frame size remain
unmeasured. The [conformance report](../../conformance/EVT2G-CONFORMANCE.md)
contains the exact support/limit inventory, test responsibilities and next seam.

Frozen Zig is unchanged; root and legacy Zig suites are skipped under the
established frozen-path policy. No other repository is changed. EVT2h is not begun.

**MEANINGFUL PROGRESSION — checked aggregate/immutable-update LIR is qualified;
native object storage, references, copies and Win64 aggregate ABI require the
new checked Concept aggregate bridge before Cathedral realization.**

## EVT2g2 closeout

Baseline: `e51c184bf51e6073652073293ed5587e26546ba6`, clean at entry.
Compiler: `concept-evt1-stage0-go`. EVT2g1 is preserved above as history.

CMIRAMD5/version 5 removes the previous bridge blocker. The checked Concept
schema derives both transport sides; the Concept validator consumes types,
layout geometry, arrays, logical copies, initialization, field locations,
immutable authority and unclassified aggregate calls/results from bytes alone.
There is no record packing calculation in Cathedral. The real CLI now emits
typed aggregate MachineIR and native admission stops precisely at
`AMD64_UNSUPPORTED_AGGREGATE_STORAGE` without changing output/image.
An independently built binary from an archived complete baseline SHA reproduces
`MIR_AGGREGATE_REQUIRES_CMIRAMD5 RecordPair` on the same source fixture; the
updated CLI emits the typed contract and reaches the new native boundary.

The contract carries 94 fields and 60 enum cases. It adds 46 ordinary derived
declarations, increasing combined inspection from 73 to 119. Its schema hash
is `7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60`.
CMIRAMD3/4 sources, generated payload codecs, schema hashes and native byte
contracts remain frozen. The conformance report records both frozen hashes.

One necessary portable bootstrap correction uses assignment target type when
lowering contextual array literals. The new validator's fixed liveness arrays
exposed the general mechanism; an independent Normal/Verify C11 regression
checks repeated and listed scalar array reassignment. This is not a native
aggregate implementation. Checked readonly scalar stores are represented as
initialization in v5 so loop-local const bindings retain the correct operation.

| Gate | Local result |
| --- | --- |
| Full `go test ./... -count=1 -parallel=4 -timeout=30m` | PASS; compiler 766.635 s, CLI 125.739 s, Vulkan profile 0.469 s |
| `go vet ./...` | PASS |
| Focused race `^TestEVT2g` | PASS; 191.986 s |
| EVT2g1 LIR/source/generic/lifetime regressions and EVT2g2 focused suite | PASS |
| Semantic corpus, Golden Normal/Verify, GPU-free Vulkan, EVT2/EVT2x/EVT2e/EVT2f native | PASS in full suite |
| Standard explicit CLI Normal / Verify | 44 passed, 0 failed each; 3 informational benchmarks |
| DragonGod explicit CLI Normal / Verify | 23 passed, 0 failed each; 1 informational benchmark |
| Artifact-only Concept roundtrip | PASS; three fixtures, each 100 times in each mode |
| Malformed artifacts | PASS; 43 exact typed failures in each mode; Go rejects all; output untouched |
| Exact transport bytes | PASS; five pinned hex artifacts and SHA-256 identities |
| Independent layout agreement | PASS; 8 semantic summaries / 13 fields; compiled C-repr size/alignment/offset probe |
| Policy and 100-run determinism | PASS; fresh checked production, layouts, schema, bytes, printer, roundtrip, diagnostics |
| Generated CLI inspection | PASS; standalone v5 codec 87, combined validator 119 |
| PlainData / StaticExtent / NoAllocation | PASS; checked assertions in `ObjectContractFacts` |
| Changed authored/generated Concept format checks and lint | PASS |
| Whitespace / frozen codec diff / final clean worktree | PASS; committed closeout, no v3/v4 blob changes |
| Frozen root and legacy Zig | SKIPPED; compiler/build/test paths unchanged |
| Native aggregate storage, fields, copies, references, Win64 aggregate ABI | DEFERRED to later Cathedral stages |

The broad run was dispatched on the final production code. A subsequent
test-only extension added artifact-imported `Box<int>` and `Box<float>` to the
roundtrip corpus; its Normal/Verify run passes separately (8.872 s), as does
its focused race run (22.397 s). No production code changed after dispatch.

### Does Explicit Layout Pay Off?

For this contract, yes: zero duplicate record layout calculators and zero
field-offset recomputations were required. Supplied semantic geometry suffices
for independent consistency admission and source-free reserialization. The
eight main summaries, 13 field descriptors, nested identities, array extent and
stride all agree with canonical semantic layout; C representation has an
additional compiled C oracle. Corrupt geometry rejects; relevant source/import
changes alter contract identity. Opaque layout hashes are producer identities,
not authentication or a replacement for checked provenance.

No native speed, frame size, aggregate instruction bytes, physical copy cost,
or Win64 aggregate execution was measured. PlainData is distinct from CAbiValue
and lifetime permission. The exact capacities, source-reference boundary,
automata/mixed-array limits, Stage-0 classifications and local informational
measurements are in the [EVT2g2 conformance section](../../conformance/EVT2G-CONFORMANCE.md#evt2g2--typed-aggregate-and-explicit-layout-contract).
EVT2g3 has not begun.

**SUCCESS — EVT2g2 CMIRAMD5 typed aggregate and explicit layout contract qualified.**

## EVT2g3 closeout

Baseline: `7ccb968ab79d4ad700db7afa396970aa78868d06`, clean at entry, containing
the EVT2g2 contract qualification. Compiler: `concept-evt1-stage0-go`.
The closeout commit contains the implementation, fixtures and this report;
`git log -1 --format=%H` identifies the final revision. No other repository is
modified, and EVT2g4 is not begun.

An independent CLI built from an archived complete baseline revision rejects
the same small `Position` local/field-update program with
`AMD64_UNSUPPORTED_AGGREGATE_STORAGE`. The updated CLI finalizes its one-function
717-byte native image. The identical `Simple` body is executed in the native
corpus (input 4 returns 11). Local comparison inputs and outputs are saved as
`storage-comparison.concept`, `baseline-storage-stop.txt` and
`current-storage-native.txt` under the stage's artifact directory.

The real CMIRAMD5 native path now emits and executes complete aggregate locals,
nested/array fields, mixed GPR/XMM transfers, initialized values, fieldwise
copies, self-assignment and immutable `with` updates. It reuses the existing
Concept allocator, unified frame, Win64 scalar calls and encoder. All production
native object policy is Concept-owned. A new `AMD64.Objects` module supplies
typed object placement and copy traversal; shared scalar addressing/encoding
helpers avoid a parallel backend. Private aggregate value slots preserve
snapshots and copy/update identity. Copy traversal skips padding.

The checked artifact contains 21 aggregate layouts and 35 fields; native lowering
uses 98 field derivations and 12 transported-stride operations. Both modes emit
34 functions / 32,504 bytes from a 109,556-byte artifact. Actual artifact-only
Win64 calls work without domain source/header/oracle functions. A separate
strict C11 run compares native results with the C bootstrap over 100 vectors.
The corpus includes 12-byte strides, narrow fields, 64-bit bounds and constants,
early returns, loops and an outgoing eight-argument call with a live object.
`with` is ordinary copy/member replacement, with a native trace proving
base-once and replacement order. There is no native WITH opcode.

### Exact remaining boundary

`language/evt2g3/boundaries/source_references.concept` is valid source and
executes to 9 through the independent C bootstrap in Normal/Verify. Its checked
native LIR stops at `Change: EVT2_UNSUPPORTED_OBJECT_AUTHORITY ref Position`.
This is the next blocker, not a native memory-operand or record-layout failure.
CMIRAMD5's validator rejects location parameters, call arguments and returns;
its derived locations require live local slot roots. Source borrow identity,
access authority and cross-function lifetime are not represented by that slice.
Those need checked LIR and a versioned transport contract before native reference
parameter/call realization. Reusing a scalar pointer or detached copy would erase
authority or break source storage identity. No such shortcut was added.

Native mutable/const/field/array-element source references remain unqualified.
Existing source escape/immutability/initialization/illegal-copy tests remain
active. The older object corpus includes aggregate function signatures and
now reaches `AMD64_UNSUPPORTED_AGGREGATE_ABI`, before output/image mutation.
Aggregate by-value/indirect/sret qualification is deferred to EVT2g4.

### Does Explicit Layout Actually Pay Off?

Yes for local object realization: zero new Cathedral record packing rules.
Supplied geometry determines whole-object size/alignment, field displacement,
nested composition and fixed-array stride/extent. Native frame placement is the
only new physical placement decision; field geometry is never reconstructed
from source/AST or C conventions. Artifact-only execution, separate C semantic
comparisons, existing semantic/C-repr layout probes, invalid-geometry admission
and exact address/load/store/copy goldens support that claim. Layout digest
validation remains consistency checking, not authentication. The remaining borrow
gap concerns authority/lifetime, not geometry.

CMIRAMD3/4/5 schemas and generated codecs are unchanged, including their frozen
hashes. Source fixtures exercise alignments 1/2/4/8 and frame sizes 40–328 bytes;
the admitted alignment cap is 16. Valid frames at or above 4096 reject with
`AMD64_STACK_PROBE_REQUIRED`. There is no stack probing or general source
over-alignment qualification. Slots are conservative, distinct and bounded;
partial-overlap/raw-pointer copies and ownership/Drop are not qualified.

One acceptance run measured about 55 ms combined planning/encoding and 1.28 s
for all-function inspection (which revalidates the artifact per function).
These local timings are informational, without an isolated planner or speedup
claim. Per-function frame/code measurements, bounds and exact copy policy are
recorded in [conformance](../../conformance/EVT2G-CONFORMANCE.md#evt2g3--native-object-storage-progression).

### Stage-0 changes

Checked LIR reuses semantic aggregate-array element layouts, supports indexed
record places, retains bounds checks even for static-safe plans, and recognizes
existing byte/isize/usize representations. Portable C generation now keeps RHS
bounds/side-effect preludes inside short-circuit guards; independently generated
headers share one built-in range-type declaration set. The CLI prints actual
Concept plans. Remaining Go changes are test oracles and the new expected ABI
boundary. None is a production native object algorithm.

### Validation

Local logs are under `artifacts/evt2g3/`.

| Gate | Local result |
| --- | --- |
| Full `go test ./... -count=1 -parallel=4 -timeout=30m` | PASS; compiler 851.785 s, CLI 138.860 s, Vulkan profile 0.486 s |
| Semantic corpus, Golden Normal/Verify, GPU-free Vulkan, EVT2/EVT2x/EVT2e/EVT2f, EVT2g1/g2/g3 | PASS in the full Go regression run |
| `go vet ./...` | PASS |
| Final `^TestEVT2g3` focused suite | PASS; 92.288 s |
| Final `go test -race ./internal/concept -run '^TestEVT2g3' -count=1 -timeout=15m` | PASS; 214.760 s |
| Focused EVT2g1/g2, frozen native bytes, EVT2f source native and checked-output tests | PASS; 138.407 s; final g3 refinements covered by the final suite above |
| Artifact-only native and separate independent C11/native oracle, Normal/Verify | PASS; 34 functions each mode; 100 semantic vectors in separate oracle run |
| Determinism | PASS; 100 identical final module emissions and 100 representative frame/placement debug descriptions in each mode |
| Exact memory instruction bytes | PASS; 11 address/load/store/extension/copy goldens in both modes |
| Malformed geometry/copy, invalid frame slots, unprobed frame | PASS; precise rejection before output/image mutation |
| Bounds traps | PASS; negative, extent and high-64-bit indices produce UD2 status in child processes |
| Artifact-imported PlainData/StaticExtent and backend NoAllocation assertions | PASS; mutable-record PlainData Unknown retained |
| Source borrow specimen | C bootstrap Normal/Verify PASS; native checked-input stop reproduced |
| Real `machineir` / `amd64` CLI | PASS; native image finalized, object frame plans printed; aggregate ABI boundary reproduced |
| Standard CLI Normal / Verify | 44 passed, 0 failed each; 3 informational benchmarks |
| DragonGod CLI Normal / Verify | 23 passed, 0 failed each; 1 informational benchmark |
| Standard/DragonGod module package build | PASS |
| Eight changed Concept sources: format check / lint | PASS |
| Go formatting, whitespace/diff checks | PASS |
| Frozen root and legacy Zig | SKIPPED; frozen compiler/build/test paths unchanged |

The broad run, final focused suite and race lane used the final production code.
Only documentation and an ignored independent baseline comparison were added
after dispatch. No failing gates were disabled or skipped. The closeout is
committed with a clean worktree; generated checked outputs and frozen codecs
require no changes.

**MEANINGFUL PROGRESSION — native local aggregate storage, fields, copies and
immutable updates execute; source-borrow authority/lifetime needs checked LIR
and a versioned transport contract before native reference qualification.**
