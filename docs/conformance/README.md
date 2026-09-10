# EVT1 conformance chronology

Milestone names describe history. Semantic names describe the active
architecture. The historical reports in this directory remain milestone-named;
their fixtures now have exactly one authority under `language/evt1`.

| Milestone | Semantic result | Implementation commit | Current corpus |
|---|---|---|---|
| R0 | Go bootstrap extraction and authority cutover | `cde2b39`, `7a0a9e0` | foundation plus inherited tests |
| R1 | differential conformance and Vulkan isolation | `43fc516` | `language/evt1/foundation` |
| R2 | values, records, const places, immovability | `5617be4` | `language/evt1/values` |
| R3 | move, Drop, and references | `ba4db99` | `language/evt1/ownership` |
| R4a | lexical lifetime proofs | `858da44` | `language/evt1/lifetime/lexical` |
| R4b | relational lifetime proofs | `3314987` | `language/evt1/lifetime/relational` |
| R4c | Option, Result, and failure | `df1432d` | `language/evt1/failure` |
| R4d | arrays and ndarrays | `cb98af2` | `language/evt1/storage/arrays` |
| R4e | explicit storage binding | `175353f` | `language/evt1/storage/binding` |
| R4f | layouts and streams | `6f892d0` | `language/evt1/storage/layout-stream` |
| R4g | Span and ReadOnlySpan | `58e985a` | `language/evt1/storage/span` |
| R4h | tensor and Einstein semantics | `6681f0d` | `language/evt1/tensor/semantics` |
| R4i | inline tensor backing and aliases | `39eb8e7` | `language/evt1/tensor/backing` |
| R4j | semantic fact qualification | `c7610ae` | `language/evt1/semantic-facts` |
| R4k | classes, interfaces, and borrowed dyn | `e94c02a` | `language/evt1/interface` |
| R4l | general Planner foundation | `2c78793` | `language/evt1/planner` |
| R5a | automata state and capture | `481b5b5` | `language/evt1/automata/state` |
| R5b | transition match and decide | `655a66f` | `language/evt1/automata/transitions` |
| R5c | inference and transition infer | `3097b37` | `language/evt1/inference` |
| R5d | yield and foreach | `3f79e49` | `language/evt1/iteration` |
| R5e | machine stack and completion | `fc05058` | `language/evt1/machine-stack` |
| R5f | async and await | `0a0104f` | `language/evt1/async/basic` |
| R5g | structured async control flow | `46b3f4c` | `language/evt1/async/control-flow` |
| R5h | async interface and dyn composition | `d225f53` | `language/evt1/async/interface` |
| R5i | explicit callable capture | `e655866` | `language/evt1/callable/capture` |
| R5j | exact concrete callable storage | `01d90e0` | `language/evt1/callable/types` |
| R5k | semantic reconciliation and language-core freeze | this report's commit | all paths in `language/evt1/manifest.json` |
| R6a | first-class test discovery and execution | this report's commit | `language/evt1/tooling/tests` plus `internal/concept/testdata/testing` |
| R6b | semantic concept assertions and proof diagnostics | this report's commit | `language/evt1/tooling/proofs` |
| R6d | local generic/library substrate; semantic modules still blocked | `EVT1-R6D-CONFORMANCE.md` | focused compiler and strict-C11 hosted tests |
| R6k | allocator retry honestly stopped on initialized typed-storage ownership | `EVT1-R6K-CONFORMANCE.md` | focused general-gap regression tests; no allocator fixtures published |
| R6l | initialized typed-storage field ownership progressed; nested generic carriers remain blocked | `EVT1-R6L-CONFORMANCE.md` | focused generic bind, owner state, Value, move, Drop, and blocker tests |
| R6m | structural generic substitution closes nested carriers and imported owner types | `EVT1-R6M-CONFORMANCE.md` | structural substitution, module, owner, and strict-C11 regression tests |
| R6n | allocator retry honestly stopped on second-parameter capability constraints | `EVT1-R6N-CONFORMANCE.md` | focused general constraint/dispatch regressions; no allocator fixtures published |
| R6o | arbitrary-position, relational, and composed generic concept requirements | `0920b96` | 382 valid, 246 static-invalid, and 13 runtime-negative fixtures |
| R6p | ordinary Standard.Memory fixed, hosted, bump, pool, and typed ownership | this report's commit | semantic-module, `.concept_test`, strict-C11, effect, lifetime, and determinism evidence |
| R7a | production libraries, ordinary manifests, namespaces, and canonical DragonGod | this report's commit | package DAG/artifacts, namespace collisions, Standard.Memory dogfood, DragonGod strict-C11 evidence |

The manifest records 382 valid, 246 static-invalid, and 13 runtime-negative
fixtures. Four retained expected-divergence cases compare the frozen EVT1
structural-copy and bounded-comptime decisions against PoC3. No EVT1 gap or
unresolved specification ambiguity remains in the R5 core.
