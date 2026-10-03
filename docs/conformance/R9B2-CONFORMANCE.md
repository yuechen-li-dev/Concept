# R9b2 conformance

**SUCCESS — R9b2 typed semantic Verdict protocol established.**

Baseline: `9c0c1cb5d6797c126df50d97f6e141b3a1dcd767`, clean before work.
Qualified implementation: `bf0af5f165bc9e250bea3e7f957aade444994675`.
Compiler: `concept-evt1-stage0-go`; branch: `codex/r9b2-typed-verdicts`.
The final report records the documentation closeout HEAD separately.

## Capability evidence

| Requested boundary | Qualified real path |
| --- | --- |
| Typed protocol | Comptime-only `Verdict<E,R>`; contextual Proven(E), Disproven(R), payload-free Unknown. Existing SemanticFactCertainty is the only truth lattice. |
| Predicate admission | Declared typename/declaration predicates, root project policy and embedded innate rules admit closed typed verdicts. Arbitrary records still diagnose PREDICATE_REQUIREMENT_INVALID with the accepted forms. |
| Legacy | Bool and fixed Holds/Refuted forms normalize through the same projector. Existing ordinary-predicate tests and fixed legacy shape checks pass. |
| Closed template calls | Ordinary closed templates and explicit comptime templates execute with existing generic identity/body substitution. The CLI also checks closed non-type arguments and their artifact-only import. |
| Bounds | Shared fuel/depth, bounded recursion, loop limits and body availability are pinned. Open execution: COMPTIME_TEMPLATE_CLOSURE_REQUIRED; unbounded recursion: CV4217; exceeded recursion: CV4206; fuel/depth: CV4204/CV4211. Unsupported runtime/foreign calls remain CV4210. CV4201 remains for unsupported operation categories, rather than supported closed-template calls. |
| Typed payload | Selected resource-free closed record/enum/scalar/array/type/declaration values retained. Resources/runtime Verdict reject. Selected payload limits: 512 value nodes, nesting 8, 4096 string bytes; prior array/string/evaluator limits retained. |
| Proof and diagnostic | ProofNode verdict metadata and MIRSemanticProof verdicts retain identity/type/outcome/payload. Assert errors include reason and compact refutation. Explain shows evidence/refutation lines. Unknown has a distinct graph node and CONCEPT_ASSERT_UNKNOWN. |
| Policy | Ordinary Concept Describe match produces the reason; lint findings retain the typed reason/payload and configured warning severity. 100 policy explanation JSON comparisons pass. |
| Artifacts | Optional sorted/deduplicated predicate_verdicts in concept-module.v2; ordinary body/type/argument transport, no dependency source reparsing or huge proof-tree envelope. Producer/consumer native strict-C11 harnesses execute. Real CLI artifact-only check/explain/Verify emission pass; typed failure and unknown remain distinct. |
| Innate dogfood | DroppableFieldIsOwnedHolds returns Verdict<OwnershipEvidence,OwnershipRefutation>; DroppableFieldMustBeOwned(field) is rendered by Concept Describe/OwnershipProblem. CV4653 wording/site/category, including generic template fixes, are preserved. |
| Strangler | Retired legacy ownership code survives only as a test oracle. Final shadow comparison: 228 reachable corpus/explicit field decisions, four refutations, exact truth/site/failure-message agreement. Production has one ownership predicate. |
| Trust | Empty compiler-owned innate fact authority allowlist, guarded by embedded authority and innate declaration. Verdict metadata never hydrates semanticFacts. No typed fact-producing rule or positive authorized-fact projection is claimed; that qualification is deferred. |
| Forgery | User Proven claims do not grant StaticExtent/Disjoint/Outlives; a Span remains Unknown for exact extent, and an allocating operation remains Disproven for NoAllocation. Ordinary source cannot obtain innate authority through rule spelling. |
| StaticExtent | All R9b exact-array/mismatch/Unknown/artifact/explain/Planner-shadow/native-vector tests pass unchanged. No additional SIMD/check elimination permission. |
| Research/freeze | PlainData, FiniteDomain, ClosedWorld, StableAddress and Relocatable revisited in the ledger without admission. Stage-0 remains unfrozen. No AMD64 implementation work or automatic EVT2 continuation. |

The flagship moved into valid/typed_predicate_protocol.concept with real protocol
syntax. The former record-shaped negative is preserved as wrong_predicate_return.
Valid typed_verdict_values exhaustively matches successful values of all three
cases. Invalid companions exercise assertion refutation/unknown, resources,
runtime Verdict, open helpers, recursion bounds and extent/allocation forgery.

## Gates

| Gate | Baseline | Final |
| --- | --- | --- |
| `go test ./... -count=1` | PASS, internal/concept 379.836s | PASS, internal/concept 384.092s; Vulkan profile 0.327s |
| `go vet ./...` | PASS | PASS |
| Focused race, innate scale and predicate/authority | PASS, 21.346s | PASS, 32.310s; expanded typed/native/authority coverage |
| Semantic corpus | 428 valid / 342 static-invalid / 15 runtime-negative | PASS: 433 valid / 351 static-invalid / 15 runtime-negative |
| Standard Normal / Verify | 42 / 42 passed | 42 / 42 passed |
| DragonGod Normal / Verify | 23 / 23 passed | 23 / 23 passed |
| Golden Normal / Verify | 130 / 130 passed | 130 / 130 passed |
| GPU-free Vulkan Normal / Verify | PASS in baseline full suite | PASS, separate TestVulkanLibraryNormalAndVerify, 12.799s |
| Focused R9b/R9b2, innate, corpus and checked outputs | PASS | PASS, 2.709s after canonical source formatting |
| Native strict C11 | Existing baseline gates | PASS: typed producer and artifact-only consumer harnesses; existing vector/native/full gates |
| Determinism | R9b retained | PASS: 100 producer artifacts, 100 innate proofs, 100 policy explanations, and 100 real CLI explain JSON/check diagnostics for each typed outcome |
| Concept formatting/lint | Existing corpus qualified | All 21 vocabulary specimens format-check; seven valid specimens lint; 14 invalid specimens reject intentionally. Embedded innate source format-check passes and validates through its authority-bearing compiler path. |
| Checked generated outputs | PASS | PASS without golden regeneration |
| Legacy Zig | Unchanged | Both suites skipped: frozen compiler/build/test paths unchanged |
| Diff/worktree | Clean baseline | PASS: ordinary diff check; implementation and documentation committed, tracked worktree clean at closeout |

Artifact-only CLI roots contain semantic artifacts and consumer source, with no
producer source. Both closed type and non-type helper arguments also pass there.
The producer artifact from the focused typed fixture is 29255 bytes. Its exposed
metadata carries the selected value rather than both payload alternatives.

## Scale and mutex

Windows informational race-instrumented samples, no performance threshold. The
same four groups and module inventory were used before and after. Full-suite
activity and host scheduling can affect wall time; these are not causal benchmark
claims. Artifact totals use a separate normal instrumentation run with identical
absolute source roots. The baseline size harness ran against a git archive of the
baseline compiler inside ignored artifacts/r9b2/baseline-measure.

| Group | Modules / evaluations, unchanged | Validation wall before -> after | Max fuel before -> after | Max depth before -> after | Artifact total bytes before -> after |
| --- | --- | --- | --- | --- | --- |
| Standard | 26 / 966 | 1.0719021s -> 1.0549512s | 28 -> 25 | 3 -> 3 | 5908920 -> 5910020 |
| DragonGod | 22 / 237 | 807.7145ms -> 857.8196ms | 28 -> 25 | 3 -> 3 | 2139312 -> 2140248 |
| Golden | 31 / 593 | 845.976ms -> 916.1533ms | 221 -> 221 | 7 -> 7 | 5837321 -> 5838621 |
| Vulkan | 1 / 194 | 15.8884ms -> 17.3758ms | 430 -> 430 | 8 -> 8 | 1154585 -> 1154629 |

All artifact increases are below 0.05%; they include the transported template and
proof metadata schema additions. Largest artifact bytes before -> after:
Standard 2157757 -> 2157797; DragonGod 528302 -> 528342;
Golden 489909 -> 489953; Vulkan 1154585 -> 1154629.

Equal workload, 320 module validations / 7960 evaluations:

| Workers | Wall before -> after | Mutex wait sum before -> after | Locked execution sum before -> after |
| --- | --- | --- | --- |
| 1 | 10.9311202s -> 11.86973s | 0 -> 517.8us | 143.2514ms -> 149.6688ms |
| 4 | 3.2566647s -> 3.908199s | 10.0393ms -> 2.5214ms | 158.0551ms -> 196.1715ms |

Max fuel/depth remain 430/8 in both equal workloads; maximum loop count is eight,
maximum array literal zero. The measured aggregate wait does not show increased
mutex contention. The process-wide lock is retained; no cache or unsafe lock
removal was attempted.

## Qualification notes

An interim test binary compiled before fixture reclassification failed when it
later read the retired invalid filename. The updated focused and complete gates
pass on the stable classification; this was not fixed by loosening a diagnostic
or accepting an invalid program. The final full gate was rerun after canonical
innate source formatting to qualify the actual embedded identity.

A direct library build-module probe lacked dependency artifacts and correctly
reported MODULE_IMPORT_MISSING. Size measurement uses the established source-to-
artifact builder; actual artifact-only CLI consumption uses an explicit artifact
root. Mixing relative/absolute spellings of that root exposed the existing
MODULE_IDENTITY_DUPLICATE guard; consistently absolute roots pass. No resolver
semantics were changed for these setup errors.

The artifact Go file had tracked CRLF bytes. Its LF normalization is pinned for
that file in .gitattributes, so the ordinary diff check accepts future additions.
It changes no artifact semantics. Generated golden C/MIR outputs were not altered
to force a passing gate. Local logs/artifacts are under ignored artifacts/r9b2.

See R9-TYPED-VERDICTS for the public semantic contract and R9B2-CONVERGENCE for the
resolved blockers and intentionally deferred authority/vocabulary work.
