# R8f conformance

Compiler: `concept-evt1-stage0-go`. Baseline compiler commit:
`83ff006db657bf23bf6b86d4259a292b7236ef98` (parent of the clean merge
checkout `30262299fc78d10a9c8367636e48cd457e1974e5`). EVT2 was not begun.

| Mechanism | R8f evidence |
| --- | --- |
| Utility/inference and machines | `Agents/Squad.concept`; three-agent native fact, temporal Fire and Patrol states |
| Hysteresis/commitment | `Commitment` score bonus and bounded `commitTicks` |
| Async/await, scoped ownership | `Async/Journal.concept`; owned `Request`, local ref before two awaits, inline frames |
| Tensor shapes and Einstein notation | `Mechanics/Stress.concept`; `rotated[i,j] = frame[i,k] * stress[k,l] * frame[j,l]` |
| Units plus tensor, interpretation | Native pascals interpreted as `double<Pa>`; `SameDimension` proof; `CV4615` and `CV4620` rejection |
| Float/double specialization, comptime | One `Dot4<T>`; `SizeOf<T>()` closes to separate branch-free C bodies |
| Concepts and declaration policy | `Floating<T>` required-operation closure; manifest `HotPathPolicy<declaration F>` |
| Reflection and generated declarations | IRTools derives `OperandCount` from `[[operand]]` fields of imported reflected `Instruction`; `CheckedOperandPass<declaration F>` requires generated provenance and NoAllocation |
| Assert.Concept and explain | NoAllocation and SameDimension facts; imported generated summary and normalized dimensions explained |
| MustUse/discard | `Transfer` is MustUse; `SendBestEffort` explicitly discards optional status; ignored status negative |
| Artifact semantic transport | Go test imports IR A, generated B, and HPC/game/async modules from artifacts alone |
| Static-invalid diagnostics | `MACHINE_UNKNOWN_STATE`, `CV4615`, `CV4620`, `CV4640`, `CV4025`, `CONCEPT_ASSERT_DISPROVEN`, `ASYNC_PERSISTENT_REF_ESCAPE`, `MUST_USE_RESULT_IGNORED` |
| Lint/format | Project manifest lint passes in Normal/Verify; all 12 new sources/manifest pass individual format checks |
| Strict C11 | Focused test compiles and executes all five new domain C outputs with GCC and Clang using `-std=c11 -pedantic-errors` |
| Determinism | 100-run artifact and generated-C byte identity for HPC, game, async, and compiler generator |

Generated-C audit: no `malloc`/`calloc` is emitted by these new workloads.
Async uses bounded inline frames and explicit step operations. The HPC
representation choice is absent from the closed generated bodies. The stress
tensor still carries shape fields and extent guards despite fixed source
extents; this is a visible conservative backend cost and is recorded in the
freeze ledger. No runtime reflection registry or unit metadata is emitted.

Normal and Verify use the same native facts. Two bounded native benchmarks
retain observable work over 1,000 operations each. A five-iteration Normal
run measured 10.14 ms per patrol benchmark invocation and 7.22 ms per dot
benchmark invocation on this host; process startup and harness overhead are
included. Structural code inspection remains the stronger performance sanity
evidence. No cross-language or throughput claim is made.
