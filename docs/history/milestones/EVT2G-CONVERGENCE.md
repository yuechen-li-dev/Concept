# EVT2g convergence — checked aggregate input

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
