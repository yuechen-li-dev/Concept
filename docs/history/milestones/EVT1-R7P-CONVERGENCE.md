# R7p burn-in convergence log

Baseline clean HEAD: `9fe4469930bf5cc973ad2d28fbe6099666c60c10`.
Compiler ID: `concept-evt1-stage0-go`. Baseline corpus: 398 valid,
262 static-invalid, 13 runtime-negative, 5 compatibility, 4 expected
divergence. Baseline `go test ./...`, `go vet ./...`, both Zig suites,
BurnIn, Standard Normal/Verify (35 each), DragonGod Normal/Verify (23 each),
and the EVT1 corpus passed. Artifact schema is `concept-module.v1`;
the proof and plan schemas are `concept-proof.v1` and
`concept-evt1-plan.v1`.
The root `README.md` indexes the formal `docs/spec/CONCEPT_EVT1_LANGUAGE_SPEC.md`,
the migration/reconciliation records, and compiler architecture. The active
language guides are under `docs/language/`; library and tooling guides are
under `docs/library/` and `docs/tooling/`. R7p adds the golden index under
`docs/examples/`. Verify uses the same source semantics and results as Normal,
retains guards and C11 atomics unless a plan has the required Proven facts,
and records observed foreign `NonNull` calls separately from static
`DeclaredForeign` authority. No runtime reflection authority was added.

## Embedded

First instinct: C-style indexed `for`, `&&`, `!`, implicit reference passing,
and an unchecked ring. Classification: intentional Concept differences.
The C-style loop now receives a direct teaching diagnostic. `uint8`
`Assert.Equals` support was a testing-library/compiler gap. Final source
uses a fixed queue, explicit MMIO, bits, Result, a command decoder, and a
bounded range loop. Normal/Verify facts pass; strict Clang/GCC C11 passes.

## Civilian aerospace

First instinct: nominal unit declarations and C-style `for` around a
fixed telemetry history. Classification: docs/intentional syntax difference.
The raw native float to unit-typed field boundary was a language gap:
`AssumeQuantity<T>` now explicitly attaches a unit only when the scalar
representation matches. It adds no runtime conversion. Commercial flight
sample transitions, fault handling, `repr(C)` native sample, and fixed
history pass Normal/Verify. Strict Clang/GCC C11 passes.

## Game

First instinct: C++ indexed loop and direct store construction. Classification:
intentional range-loop difference, plus a compiler lowering bug for nested
`DenseStore{Uninitialized()}` fixed in R7p. A hot `DenseStore` traversal and
`Patrol` machine with explicit yield pass Normal/Verify and strict C11.
The expanded game fixture adds a generational trail pool, a table/Span SoA
projection, and an application-owned tick budget. Importing the closed
`GenerationalStore.Remove` template initially omitted canonical
`Result<void, StoreError>` constructors in generated C; failure-type
collection now inspects instantiated template bodies. Both game modules pass
artifact-only import and strict Clang/GCC C11.

## HPC

First instinct: C-style loop and implicit conversion from a fixed table to
`Span`. Classification: intentional loop difference and an API discovery gap.
Use explicit `Span(backing)` or `ReadOnlySpan(backing)` to state borrowing.
`Step` collides with the machine intrinsic; `AdvanceHeat` is the final name.
The fixed-shape stencil, reduction, stability check, and residual shape
diagnostic pass Normal/Verify and strict C11. A local `NoAllocation` proof for
`Len` became Unknown after artifact import; the module effect summary now
uses the same compiler-known storage inspection rule. A test local named
`short` then exposed a generated-C keyword collision; local bindings now
receive a private C identifier while their Concept name stays intact.

## HFT

First instinct: `AtomicLoad`/`AtomicStore` names, C-style loop, and Option
comparison. Classification: Standard API discovery and intentional match/range
differences. The final bounded queue uses `LoadAtomic`/`StoreAtomic` with
explicit Acquire/Release and a timestamp intrinsic. Explicit feed and signal
contexts prove `SingleProducer` and `SingleConsumer`; the structural
`PublishedBefore` claim is Unknown and cannot erase atomics. A missing
NoAllocation summary for known atomic intrinsics was fixed. Normal/Verify
and strict C11 pass.

## Compiler/runtime authoring

First instinct: C-style loop, `Span` over payload tokens, and `!` for
recoverable parser errors. The payload-token `Span` hit fixed-element geometry
(`CV4601`); fixed `TokenWord` records form the stream while the AST remains a
payload enum. Explicit `?` retains `ParseError`. Dense node IDs and a bounded
operand stack pass Normal/Verify and strict C11. This is a current storage
constraint, not a proposal for variable-size contiguous elements.

## Native C++ companion

First instinct: C++-style pointers and a guessed foreign contract spelling.
Classification: intentional ABI seam and docs discoverability. The final
native C++ `Counter` remains native. A small C bridge, typed manifest,
measured two-field `repr(C)` snapshot, schema concept, and Verify observer
pass through the native project path. Static contract origin is
`DeclaredForeign`; runtime observation is separate evidence.

## Storage/database

First instinct: C++ raw pointers, `nullptr`, and a C-style indexed search.
Classification: intentional ownership/handle difference. `GenerationalStore`
gives bounded pages and stale-handle rejection. Eviction initially failed
`DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW`; a nested inspection scope now ends
the page borrow before removal. Normal/Verify and strict C11 pass.

## CAD/geometry

First instinct: C++ pointer arithmetic, `auto`, and vector operators.
Classification: intentional explicit IDs and value geometry. Unit literals
were not inferred through a function argument; typed local bindings made the
unit intent clear. The centroid exposed incorrect unit propagation in
`length / 3.0`: a dimensionless divisor had been given meters, erasing the
result's unit. Literal contextualization now distinguishes scaling from
addition/comparison. Dense vertex/face IDs, signed area, centroid, bounds,
and degeneracy census pass Normal/Verify and strict C11.

## Cross-cutting fixes and current limits

Generated C fixes cover unused range bindings, unused payload bindings,
redundant equality parentheses, exhaustive-match return analysis, nested
struct construction, and C-keyword local bindings. The SFIAEAIMTOPII corpus
adds six static-invalid
generic cases: invalid dependent type, missing member, invalid non-type
argument, missing required operation, ambiguous valid overload, and an
explicit constraint exclusion. Constraints decide applicability;
substitution failures remain diagnostics.

Artifact-only consumers of game, CAD, storage, and aerospace modules compile
without dependency source. The 100-run game and aerospace artifact/C/MIR
determinism check passes. Strict Clang and GCC C11 checks pass for the eight
Concept-only goldens. Zig cross-compiles the embedded generated C for
AArch64 Windows; it was not executed there. The first `zig cc` Linux target
attempt failed with a toolchain FileNotFound error and is not a target claim.

Generated hot C retains explicit range state and several bounds checks per
stencil iteration; no allocation calls or coroutine runtime appeared. This
is a structural performance concern for EVT2 measurement, not a reason to
change EVT1 semantics. A handwritten C loop has one induction variable and
direct neighboring loads, while current generated C has checked index
helpers and a range cursor. No timed performance equivalence is claimed.

| Golden | Main source LOC | Total fixture LOC (source, tests, first draft, README) |
| --- | ---: | ---: |
| Civilian aerospace | 129 | 225 |
| CAD | 140 | 211 |
| Compiler | 133 | 223 |
| Embedded | 164 | 314 |
| Game | 105 + 71 | 339 |
| HFT | 140 | 266 |
| HPC | 116 | 216 |
| Storage | 138 | 209 |
| Native companion | C++ bridge plus Concept | 213 |

The golden fixture totals now meet the roughly 200-line lower target. The
composition cases qualified by R7p and earlier permanent tests are:

| Composition | Evidence |
| --- | --- |
| MMIO plus bounded range loop | Embedded serial golden Normal/Verify |
| Generic store artifact plus proof environment | `TestR7pGoldenArtifactsWithoutDependencySource` imported game consumer |
| Verify plus foreign code | Native companion golden with typed `NonNull` observer |
| DenseStore plus immovable owned values | Standard `StoredNoncopyableValuesDropExactlyOnce` |
| Generated declaration plus ordinary ownership checking | `TestGeneratedUninitializedLocalUsesOrdinaryChecker`, `TestGeneratedFunctionSurvivesArtifactOnlyImports` |
| Reflection, schema, and Octagon | Standard derived record/table/enum codec facts |
| Async plus scoped authority | `TestR7f2ScopedLeaseCannotSilentlyCrossSuspension` |
| ABI evidence plus artifact-only reuse | Native companion golden and `TestR7jAggregateDeclarationSurvivesArtifactOnlyImport` |
| Machine intrinsic plus async artifact consumer | `TestMachineIntrinsicGenericAndAsyncArtifactConsumer` |

Human review: an experienced developer in each domain can follow the
application state and data flow without compiler internals. The parts that
need onboarding are explicit MMIO authority (embedded), unit attachment and
proof provenance (civilian aerospace), typed IDs and yield (game/compiler),
Span borrowing (HPC/CAD), memory-order proofs (HFT), generational invalidation
(storage), and measured ABI/DeclaredForeign evidence (native companion).
Each golden README separates familiar mechanics from these Concept concepts.

Final qualification passed: `go test ./...`, `go vet ./...`, both Zig test
roots, BurnIn, the full EVT1 corpus, Standard (35 Normal and 35 Verify),
DragonGod (23 Normal and 23 Verify), the eight Concept-only golden domains
(27 Normal and 27 Verify), and the native companion (2 Normal and 2 Verify).
The 100-run artifact/MIR/C check passed for game and aerospace. All nine
Concept source modules passed strict Clang and GCC C11 syntax checks. The
embedded generated C also cross-compiled for AArch64 Windows without an
execution claim. The final freeze decision is recorded in
`EVT1-R7P-FREEZE-REPORT.md`.
