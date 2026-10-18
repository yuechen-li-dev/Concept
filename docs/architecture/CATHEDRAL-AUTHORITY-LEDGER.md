# Cathedral authority ledger (R9c)

Update this ledger when an EVT2 or frontend change moves a **decision**. The
migration matrix records the fuller subsystem inventory and prerequisites.

States: `GO_AUTHORITATIVE` means Stage-0 makes the normal decision;
`SHADOWED` means a checked Concept decision exists but is not selected;
`CONCEPT_AUTHORITATIVE` means Concept decides the normal path;
`BOOTSTRAP_ONLY` means Go remains for the independent seed, without evolving
normal semantics; `SHARED_BY_DESIGN` means the responsibilities are distinct
and named; `NOT_YET_QUALIFIED` means the required native implementation does
not yet exist. A generated Go projection of a Concept contract does not make Go a
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
| Planner and LIR | Stage-0 | None | Go Planner/LIR producer/verifier | GO_AUTHORITATIVE | Direct scalar call and EVT2f width-specific floating input contracts qualified; later Concept transformation parity |
| Scalar floating/XMM allocation, ABI and frames | Cathedral AMD64/Win64ABI | Independent byte-stack replay and strict C11 source oracle | Stage-0 produces checked floating LIR/CMIRAMD4 and hosts compiled Concept | CONCEPT_AUTHORITATIVE | EVT2f3 plans and EVT2f5 native execution qualify XMM0..XMM5, mixed positional ABI, typed spills/moves and one frame |
| Scalar SSE2 encoding and no-call native execution | Cathedral AMD64 | Independent strict C11/native and exact-byte oracles | Stage-0 checked input and executable-memory host | CONCEPT_AUTHORITATIVE | EVT2f4 arithmetic/comparisons/constants/slots/arrays/pressure retained; EVT2f5 qualifies fixed-signature scalar direct calls |
| CMIRAMD4 scalar/XMM contract | Concept BridgeSchema4 | Exact generated codec agreement and 52 malformed artifacts | Generated Go transport; checked LIR projection/verifier | CONCEPT_AUTHORITATIVE | Virtual class, exact representation, IEEE bits, semantic operations and ordered calls qualified in EVT2f2; no physical XMM identities |
| MachineIR producer and verifier | Stage-0 | Concept validates call transport before backend projection | Go producer/verifier | GO_AUTHORITATIVE | CMIRAMD3 input feeds qualified Concept native internal calls |
| MachineIR wire schema | Concept `BridgeSchema` | Go/Concept generated codec agreement | Generated Go codec | CONCEPT_AUTHORITATIVE | CMIRAMD3 v3 qualified by exact bytes, artifact-only roundtrip and 23 malformed cases |
| Call artifact admission | Concept `BridgeValidate` on normal AMD64 path | Stage-0 verifier checks producer input | Go semantic transport/verifier; generated codec | SHARED_BY_DESIGN | Typed transport through native scalar Win64 execution qualified |
| Condition-name legality in Go MachineIR verifier | Concept `BridgeSchema.Condition` | Frozen former Go table in `TestR9cConditionAuthorityShadowAgreement` | Generated Go tag table | CONCEPT_AUTHORITATIVE | Keep table generated; expand only via schema/versioned contract |
| AMD64 block layout, liveness, intervals, allocation | Concept `Standard.Backend.AMD64` | Native/C behavior oracle | No independent Go allocator | CONCEPT_AUTHORITATIVE | Call clobbers, save/spill storage and actions qualified in Concept |
| AMD64 native frame and module encoder | Concept `Standard.Backend.AMD64` | Frozen no-call bytes and strict C11/native call oracle | Stage-0 C transport compiles library | CONCEPT_AUTHORITATIVE | Direct internal Win64 calls, frame actions and native ABI probes qualified in EVT2e5 |
| Internal symbol layout and CALL fixups | Concept `EmitModule/ResolveNativeCalls/ValidateNativeImage` | 100 emissions, exact bytes, range/patch rejection and Normal/Verify parity | Go/C host transports artifacts and publishes finalized bytes only | CONCEPT_AUTHORITATIVE | Declaration-order semantic identities; no Go resolver to retire |
| Incoming Win64 argument/return selection | Stage-0 `machineir.go` | Concept backend reads descriptors | Go MachineIR builder | GO_AUTHORITATIVE | Contract-first separation; retain seed projection if needed |
| Outgoing Win64 scalar argument/return placement and register sets | Concept `Win64ABI` checked tables and planner | Native C11 fixture oracle | Go carries unchanged CMIRAMD3 input and displays Concept plans | CONCEPT_AUTHORITATIVE | Planning and concrete frame/save/spill realization qualified |
| Call-clobber analysis and parallel moves | Concept `AMD64` liveness and `Win64ABI` move planner | Independent move-value replay, 100-run plans and Normal/Verify agreement | Stage-0 orchestration only | CONCEPT_AUTHORITATIVE | Concrete preservation/temp storage qualified in EVT2e4 |
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

EVT2e4 extends the existing Concept finalizer into one native stack geometry.
Spill allocation, used-callee save storage, full-GPR temporary storage, maximum
outgoing home/argument area, offsets and inverse prologue/epilogue actions are
CONCEPT_AUTHORITATIVE. No production Go counterpart was introduced. The normal
emit path qualifies concrete actions with independent identity/width replay
before the `AMD64_UNSUPPORTED_CALL_ENCODING` stop. An executable input-driven
automata protocol checks pre/post-call action ordering. DragonGod kernel frames
remain persistent source-language execution state, distinct from native stack
storage; Standard retains its current dependency direction.
`TestEVT2e4FrameRealization` adds strict C11 value/byte-stack replay, all 256
source combinations, real call-bearing fixtures and 100-run frame/action output.
See [EVT2e4 conformance](../conformance/EVT2E4-CONFORMANCE.md).

EVT2e5 selects Concept module emission on the normal `concept amd64` path.
Concept owns E8 rel32, successor-relative range admission, declaration-order
symbols, forward/backward patch resolution and final image verification. Its
frame/action encoder realizes prologues, epilogues, spills, saves, temporaries,
outgoing stores and RAX handoff. No duplicate Go algorithm was introduced:
Go changes are BOOTSTRAP_ONLY orchestration and test/diagnostic presentation.
`TestEVT2e5CLIContractAndEmission` qualifies selection;
`TestEVT2e5NativeInternalCalls` qualifies strict C11 parity, native ABI sentinels,
cycles, alignment and 100-run bytes. CMIRAMD3 is unchanged. EVT2e is complete
for direct internal integer/bool Win64 calls through eight arguments; pointer
source calls and the deferred ABI/linkage/unwind features remain unqualified.
See [EVT2e5 conformance](../conformance/EVT2E5-CONFORMANCE.md).

EVT2f2 adds a separate virtual-only scalar contract without mutating CMIRAMD3.
BridgeSchema4 owns its enums and records; both codecs and the Go model are derived
from checked Concept reflection. BridgeValidate4 owns complete artifact admission
on the normal Concept EmitModule path, before output/image mutation. Stage-0
projects verified LIR, checks input consistency through the existing LIR verifier,
transports artifacts and orchestrates C bootstrap execution. Class metadata is
checked input, not physical allocation policy. The retained Go code is the seed
and independent producer checker; no XMM allocator, float ABI planner, spill
algorithm or SSE encoder exists in Go. XMM allocation, floating ABI/clobbers,
floating spills/frame integration, SSE encoding and native float execution remain
NOT_YET_QUALIFIED, with Cathedral as the intended owner. See the
[EVT2f2 qualification](../conformance/EVT2F-CONFORMANCE.md#evt2f2--cmiramd4-transport-qualification).


EVT2f3 completes Concept authority for physical XMM allocation, mixed positional
Win64 calls/XMM clobbers and typed floating storage in the existing frame. Derived
metadata never enters frozen CMIRAMD3/4. The same positional core admits legacy
integer and typed mixed requests; class-separated moves and the existing frame
finalizer own realization. Concept preserves exact result capture before reload,
reserved-register rejection and named planner errors. Go additions are
BOOTSTRAP_ONLY presentation and INDEPENDENT_ORACLE tests. The current stop is
`AMD64_UNSUPPORTED_SSE_ENCODING`; native floating execution remains
NOT_YET_QUALIFIED. Earlier EVT2f2 statements above describe that historical state.
See [EVT2f3 qualification](../conformance/EVT2F-CONFORMANCE.md#evt2f3--xmm-allocation-mixed-win64-abi-and-floating-frames).


EVT2f4 qualifies Concept-owned scalar SSE2 selection, encoding, immediate IEEE
constant materialization, sign-bit negation and unordered-aware comparisons.
Native entry/return, one-frame scratch, locals, incoming checked indexed arrays,
pressure spills and deterministic no-call images are qualified in Normal/Verify.
Go retains checked transport, presentation, RW-to-RX orchestration and independent
oracles. CMIRAMD3/4 remain unchanged; the latest SMLIB/NURBS optimization baseline
is preserved. Earlier SSE-stop statements describe EVT2f3 history. The current
remaining floating call boundary is `AMD64_UNSUPPORTED_FLOAT_CALL_EMISSION`.
No generated mixed floating calls or live-across actual calls are qualified.

The AMD64 implementation is organized into ordinary `AMD64.*.concept` modules
for model, analysis, frame realization and encoding, with `AMD64.concept`
retaining the public emission entry point. This organizational split preserves
the existing authority and contracts; see the [backend source map](../../libraries/Standard/Backend/README.md).

## EVT2f5 mixed native call authority

Cathedral now realizes CMIRAMD4 mixed direct calls using the existing ABI plan,
class-separated allocation, concrete frame actions and semantic-identity rel32
fixups. A single `EmitAllocatedCall` drives integer and mixed scalar emission.
Typed GPR/SSE transfers consume the verified action order, including result
capture before preservation reloads. The frame authority reserves distinct move
cycle and native scratch storage; it also owns incoming stack offsets.

New production code is ordinary Concept in `AMD64`, `AMD64.Encoding`,
`AMD64.Frame` and `AMD64.Model`. `Win64ABI` remains the sole positional/move-graph
authority. Go changes are BOOTSTRAP_ONLY diagnostic wording and
INDEPENDENT_ORACLE/qualification tests. No new Go lowering, allocation or
encoding algorithm exists. CMIRAMD3/4 and their generated codecs are unchanged.

The earlier floating-call boundary describes EVT2f4 history. Fixed-signature
scalar direct calls now execute natively in Normal and Verify. Independent
compiled Win64 probes cover positional arguments, stack values, home space,
entry alignment, volatile XMM clobbers and observable void calls. Controlled
foreign stubs are test-only RW-stage integration, not production linkage.
External/indirect/variadic/aggregate calls, XMM6–XMM15 allocation, object output
and unwind remain deferred. See [EVT2f5 qualification](../conformance/EVT2F-CONFORMANCE.md#evt2f5--mixed-win64-floating-calls).

## EVT2g checked aggregate input authority

| Changed surface | R9c classification | Authority and replacement condition |
| --- | --- | --- |
| `lir_aggregate.go`, aggregate additions in `lir_lower.go` | SHARED checked input / BOOTSTRAP_ONLY MIR-to-LIR construction | Existing checked types, geometry, copyability and places supply the snapshot and typed operations; migrates with the frontend/LIR producer, not into a second native engine |
| `lir.go`, `lir_aggregate_verify.go` | BOOTSTRAP_ONLY producer consistency/lifetime checks | Validate layout identity, complete initialization, bounded typed locations and immutable authority; future Concept bridge independently validates transported evidence |
| `machineir_lower.go` aggregate gate | BOOTSTRAP_ONLY contract admission | Frozen scalar contracts cannot encode objects; replace the named stop only after a new checked Concept schema and generated codecs exist |
| `generate.go` expected field type for `with` replacements | BOOTSTRAP_ONLY portable C path correctness | Reuse existing contextual initializer lowering; preserve the durable C seed path |
| `lir_aggregate_test.go`, EVT2g fixtures | INDEPENDENT_ORACLE / qualification | Source C11 values/geometry, malformed input, semantic artifacts and deterministic LIR; no native aggregate/ABI claim |

All native aggregate layout consumption, address realization, copying,
references, stack placement and Win64 aggregate classification remain
NOT_YET_QUALIFIED and Cathedral-owned. No production Go native aggregate
classifier, frame planner, copier or encoder was introduced. The checked LIR
snapshot is not a new competing source-language type/layout model.
