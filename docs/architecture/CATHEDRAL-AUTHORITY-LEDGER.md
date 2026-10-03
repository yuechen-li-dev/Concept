# Cathedral authority ledger (R9c)

Update this ledger when an EVT2 or frontend change moves a **decision**. The
migration matrix records the fuller subsystem inventory and prerequisites.

States: `GO_AUTHORITATIVE` means Stage-0 makes the normal decision;
`SHADOWED` means a checked Concept decision exists but is not selected;
`CONCEPT_AUTHORITATIVE` means Concept decides the normal path;
`BOOTSTRAP_ONLY` means Go remains for the independent seed, without evolving
normal semantics; `SHARED_BY_DESIGN` means the responsibilities are distinct
and named. A generated Go projection of a Concept contract does not make Go a
second contract author.

| Subsystem / decision | Authority owner | Shadow implementation | Bootstrap implementation | Status | Next transition |
| --- | --- | --- | --- | --- | --- |
| Parse, syntax, names, type identity | Stage-0 | None | Go parser/validator | GO_AUTHORITATIVE | Extract individual rule contracts; no wholesale rewrite |
| Generic closure and comptime | Stage-0 | Concept closed bodies are consumers, not a second evaluator | Go evaluator | GO_AUTHORITATIVE | Stable evaluated-value/artifact contract before switching |
| Semantic observation API | Stage-0 checked projections | Concept predicates consume observations | Go observation host | SHARED_BY_DESIGN | Version/qualify observations, avoid display names as identity |
| Innate language-rule predicates | Concept for migrated rules; Go for remaining rules | Rule-specific pre-switch parity only | Go proof/evaluation host | SHARED_BY_DESIGN | Migrate one rule family only with checked observations and exact diagnostics |
| Proof graph and fact trust | Stage-0 mechanism; Concept authored claims | None | Go proof host | GO_AUTHORITATIVE | Separate decision rule from evaluator/provenance transport |
| Artifact identity and loading | Stage-0 | None | Go `module_artifact.go` | GO_AUTHORITATIVE | Versioned read/write agreement |
| MIR and semantic facts | Stage-0 | None | Go MIR producer | GO_AUTHORITATIVE | Typed MIR contract and shadow |
| Planner and LIR | Stage-0 | None | Go Planner/LIR producer/verifier | GO_AUTHORITATIVE | Direct scalar call contract qualified; later Concept transformation parity |
| MachineIR producer and verifier | Stage-0 | Concept validates call transport before backend projection | Go producer/verifier | GO_AUTHORITATIVE | CMIRAMD3 and Concept outgoing plans qualified; preservation/frame realization next |
| MachineIR wire schema | Concept `BridgeSchema` | Go/Concept generated codec agreement | Generated Go codec | CONCEPT_AUTHORITATIVE | CMIRAMD3 v3 qualified by exact bytes, artifact-only roundtrip and 23 malformed cases |
| Call artifact admission | Concept `BridgeValidate` on normal AMD64 path | Stage-0 verifier checks producer input | Go semantic transport/verifier; generated codec | SHARED_BY_DESIGN | Typed transport and derived ABI planning qualified; frame/spills/encoding deferred |
| Condition-name legality in Go MachineIR verifier | Concept `BridgeSchema.Condition` | Frozen former Go table in `TestR9cConditionAuthorityShadowAgreement` | Generated Go tag table | CONCEPT_AUTHORITATIVE | Keep table generated; expand only via schema/versioned contract |
| AMD64 block layout, liveness, intervals, allocation | Concept `Standard.Backend.AMD64` | Native/C behavior oracle | No independent Go allocator | CONCEPT_AUTHORITATIVE | Add call clobbers and save/spill policy in Concept |
| AMD64 no-call frame and encoder | Concept `Standard.Backend.AMD64` | Frozen bytes/native execution | Stage-0 C transport compiles library | CONCEPT_AUTHORITATIVE | Add EVT2e frame/call encoding from checked contract |
| Incoming Win64 argument/return selection | Stage-0 `machineir.go` | Concept backend reads descriptors | Go MachineIR builder | GO_AUTHORITATIVE | Contract-first separation; retain seed projection if needed |
| Outgoing Win64 scalar argument/return placement and register sets | Concept `Win64ABI` checked tables and planner | Native C11 fixture oracle | Go carries unchanged CMIRAMD3 input and displays Concept plans | CONCEPT_AUTHORITATIVE | Qualified planning; frame and save/spill realization pending |
| Call-clobber analysis and parallel moves | Concept `AMD64` liveness and `Win64ABI` move planner | Independent move-value replay, 100-run plans and Normal/Verify agreement | Stage-0 orchestration only | CONCEPT_AUTHORITATIVE | Consume preservation requirements and symbolic cycle temporaries in EVT2e4 |
| C emission | Stage-0 | Native oracle for qualified slice | Go C emitter | BOOTSTRAP_ONLY target; currently normal path | Preserve external bootstrap and portable fallback |
| Formatter | Stage-0 | None | Go formatter | GO_AUTHORITATIVE | Migrate only if a useful Concept tool emerges |
| Lint/project policy | Concept predicates choose bounded policy; Go applies manifest | Proof/explain parity | Go policy runner | SHARED_BY_DESIGN | Expand rules over stable observations |
| Generated declarations/reflection | Stage-0 mechanism; Concept authored generators | Artifact-only consumers | Go materializer/reflection | SHARED_BY_DESIGN | Move rule decisions before AST machinery |

The R9c helper switch is genuine but small: the verifier now consults the
Concept-authored condition enum via generated `bridgeTagsCondition`. It does
not route all verification through Concept. The former Go switch is RETIRED;
the generated table is BOOTSTRAP-ONLY transport. No new normal backend feature
was implemented twice. Future EVT2e entries should cite a contract version,
oracle, authority-switch test and RETIRE/BOOTSTRAP-ONLY/SHARED outcome.

EVT2e2 preserves that boundary: `BridgeSchema` owns call tags and wire shape;
generated codecs preserve the checked schema. `BridgeValidate` owns call artifact
admission on the normal Concept backend path, including unselected functions.
Stage-0 transports checked LIR identities and virtual values into MachineIR and
checks producer consistency. No outgoing Win64 algorithm was added to Go.
`TestEVT2e2ConceptArtifactOnlyCallRoundTrip` qualifies the Concept consumer;
`TestEVT2e2CLIContractAndBoundary` qualifies normal selection and the precise
`AMD64_UNSUPPORTED_CALL_LOWERING` boundary. PlainData assertions consume the
Vocabulary's typed Verdict evidence; runtime decoding returns Result.

EVT2e3 derives transient physical call plans from CMIRAMD3 without changing the
wire schema. The Concept allocator derives fixed call constraints, extends call
argument liveness, and admits additional callee-saved candidates only for call
functions. Per-call preservation and used-callee-saved summaries are explicit;
the no-call pool is unchanged. `TestEVT2e3Win64PlansMovesAndLiveness` qualifies
tables, cycles, aliasing, stack slots and liveness in compiled Concept code.
`TestEVT2e3CLIDebugPlans` qualifies normal CLI planning and the later
`AMD64_UNSUPPORTED_CALL_FRAME_LOWERING` stop. No new Go ABI decision exists.
