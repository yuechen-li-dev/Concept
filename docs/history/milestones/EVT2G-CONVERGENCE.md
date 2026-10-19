# EVT2g convergence — checked aggregate input

Current milestone: [EVT2g2 closeout](#evt2g2-closeout). The original EVT2g1
progression report below is retained with its historical boundary and evidence.

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
