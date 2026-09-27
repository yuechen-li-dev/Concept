# EVT1 R7p semantic freeze report

Baseline clean HEAD: `9fe4469930bf5cc973ad2d28fbe6099666c60c10`.
The final qualification commit is the commit containing this report; use
`git rev-parse HEAD` in that checkout for its full SHA. Compiler ID:
`concept-evt1-stage0-go`. The worktree was clean at baseline and is to be
committed clean at freeze.

The active documentation index is the root `README.md`, with the language
reference at `docs/spec/CONCEPT_EVT1_LANGUAGE_SPEC.md` and the golden index
at `docs/examples/EVT1-GOLDENS.md`. The semantic artifact schema remains
`concept-module.v1`; proof and plan schemas remain `concept-proof.v1` and
`concept-evt1-plan.v1`. Verify preserves Normal semantic results, retains
runtime guards and C11 atomics unless the required facts are Proven, and
records observed foreign behavior separately from declared foreign authority.

## Permanent goldens and first-draft friction

Each directory contains intent, a preserved natural first draft, final source,
and `.concept_test` facts. Detailed classification and the new Concept ideas
required for each domain are in `EVT1-R7P-CONVERGENCE.md` and each README.

| Domain | Main Concept LOC | Fixture LOC | First-draft friction and resolution |
| --- | ---: | ---: | --- |
| Embedded UART | 164 | 314 | C-style `for`, truthiness and reference passing: intentional differences; teaching loop diagnostic added |
| Civilian aerospace telemetry | 129 | 225 | Unit attachment from a native scalar: explicit `AssumeQuantity<T>` added |
| Game agents | 105 + 71 | 339 | Nested store construction and imported closed template C constructors: compiler bugs fixed |
| HPC heat stencil | 116 | 216 | Explicit Span borrowing; imported `Len` proof and C keyword binding bugs fixed |
| HFT market path | 140 | 266 | Atomic API spelling documented; Unknown publication proof retains synchronization |
| Compiler postfix pipeline | 133 | 223 | Payload enum is not a fixed Span element; fixed token records are the current answer |
| Native C++ companion | C++ bridge plus Concept | 213 | Manifest, measured ABI, and foreign contracts document the unchanged C++ module |
| Storage page cache | 138 | 209 | Scoped borrow ends before generational slot removal |
| CAD mesh | 140 | 211 | Typed unit arguments; dimensionless scaling literal bug fixed |

The defense-themed candidate became civilian commercial flight telemetry.
The control workload remains deterministic and bounded. `NativeSample` has a
local `repr(C)` declaration; imported `CAbiValue<NativeSample>` is correctly
Unknown without a probe. The native C++ companion demonstrates measured
`CAbiValue` and Verify foreign observation through a separate real ABI seam.

## Changes accepted for EVT1

The only new language primitive is explicit scalar-to-unit attachment through
`AssumeQuantity<T>` with a representation check. Multiplication and division
now treat numeric literals as dimensionless scaling values. No fixed-array
syntax change was made: `T[N]` works and legacy `T<array>[N]` remains. No new
Standard library abstraction was added. The six-case SFIAEAIMTOPII static
negative family confirms that constraints decide applicability and invalid
substitution is a diagnostic, not silent candidate removal.

Compiler correctness fixes cover C-style `for` guidance, uint8/16/32/64 and
size-typed test equality, unused range and match bindings in strict C,
exhaustive-match return analysis, nested struct construction, C keyword local
bindings, imported closed-template failure constructors, known atomic and
storage-inspection NoAllocation summaries, and signature-aware proof call
resolution. Documentation corrections cover loops, generated declarations,
quantity boundaries, synchronization names, and proof provenance. The
candidate ledger records accepted, intentional, and deferred decisions.

## Qualification evidence

| Gate | Result |
| --- | --- |
| `go test ./...`, `go vet ./...` | Pass |
| Root and `legacy/poc3-zig` `zig build test` | Pass |
| BurnIn (`go test ./internal/concept -run R7d3 -count=1`) | Pass |
| Standard Normal / Verify | 35 / 35 facts, 3 benchmarks |
| DragonGod Normal / Verify | 23 / 23 facts, 1 benchmark |
| Eight Concept-only golden domains Normal / Verify | 27 / 27 facts |
| Native C++ companion Normal / Verify | 2 / 2 facts, measured ABI and foreign observer |
| Full EVT1 corpus | 400 valid, 269 static-invalid, 13 runtime-negative, 5 compatibility, 4 expected-divergence |
| SFIAEAIMTOPII | Six static-invalid diagnostics pinned |
| Artifact-only | Game, CAD, storage, aerospace consumers generated without dependency source |
| Determinism | 100 game and 100 aerospace artifact/MIR/C builds byte-identical |
| Strict C11 | Clang and GCC passed all nine Concept source modules and handwritten baseline shapes |
| AArch64 structural | Embedded C cross-compiled with Zig for AArch64 Windows; not executed |

Cross-subsystem evidence includes MMIO with bounded ranges, generic store
artifacts with proofs, Verify with foreign code, DenseStore with immovable
values, generated declarations with ownership, reflection/schema/Octagon,
async with scoped authority, ABI evidence with artifacts, and machine
intrinsics with an async artifact consumer. The convergence log names their
permanent tests. Bounded parser truncation stress covers generic, array,
payload, generated declaration, asm, bits, machine, and manifest syntax.

Generated hot C keeps direct fixed storage and has no allocation or coroutine
runtime. It does retain a range cursor and repeated checked-index helpers in
the stencil loop; the handwritten C baseline has a simpler induction loop.
This is structural performance evidence, not a timed equivalence claim or an
EVT1 semantic reason to weaken bounds. The HFT Unknown `PublishedBefore`
claim leaves Acquire/Release atomics in generated C. POSIX and MSVC/clang-cl
were not qualified. A Linux cross-target attempt failed at toolchain file
lookup and supplies no platform claim.

Deferred after EVT1: native LIR/MachineIR/AMD64, broader SIMD, runtime
reflection, collector expansion, full C++ interop, scheduler/framework
policy, and measured performance work. They do not change the frozen EVT1
semantic contract.

**EVT1 SEMANTIC SURFACE FROZEN**

Recommended next phase: **EVT2 — LIR → MachineIR → AMD64**.
