# Cathedral authority ledger (R9c)

Current aggregate status is [EVT2g5 native execution](#evt2g5-aggregate-emission-authority).
Ordinary enum status is [EVT2h1 native execution](#evt2h1-ordinary-enum-authority).
Earlier milestone tables below retain their historical qualification boundaries.

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

## EVT2g2 CMIRAMD5 contract authority (historical stage boundary)

Source and semantic checking establish object layout. MachineIR transports
checked layout facts. Cathedral verifies and consumes them. It does not
independently derive record geometry. The preceding EVT2g bridge-stop statement
is historical: aggregate `machineir` now succeeds and native admission stops at
`AMD64_UNSUPPORTED_AGGREGATE_STORAGE`.

| Changed surface | Stage-0 classification | Authority |
| --- | --- | --- |
| `lir_aggregate.go`, `lir_lower.go`, `types.go`, `module_artifact.go` | Checked semantic production | Snapshot existing semantic geometry/copyability, array extent/stride and verified dependency identities; no packing or native placement algorithm |
| `BridgeSchema5` | Concept authoritative | Sole CMIRAMD5/version-5 schema, field order, tags, representation and schema identity |
| `machineir_bridge_generate.go`, `cmd/machinebridgegen`, generated Go/Concept codecs | Artifact transport | Checked reflection derives both codec sides; v3/v4 remain frozen |
| `machineir_object.go`, dispatch in `machineir*.go` | Artifact transport / compatibility seam | Project checked values/storage/locations and semantic calls; normalize `with` into logical copy/member replacements and readonly scalar initialization |
| `machineir_object_verify.go`, added logical LIR operation checks | Bridge verifier | Bootstrap consistency with checked LIR identities/operations; no production ABI or native object algorithm |
| `BridgeValidate5` | Concept authoritative | Artifact-only bounded type/geometry/copy/lifetime/provenance/call admission; typed Result diagnostics; consumes supplied field offsets |
| `AMD64`, `AMD64.Model`, `AMD64.Analysis` | Concept authoritative / CLI diagnostic boundary | Recognize and independently validate v5, then return precise pending-storage error without touching output/image |
| `generate.go` assignment expected-type repair | Compatibility seam | Portable C bootstrap contextual fixed-array assignments; necessary for ordinary validator liveness arrays, independently tested |
| `machineir_object_test.go`, exact hex artifacts and fixtures | Test oracle | Semantic geometry, independent C layout probes, source-free Concept roundtrip, malformed inputs, determinism and unchanged native boundaries |

There is no new CLI native algorithm and no Go aggregate frame allocator,
copier, field encoder or Win64 classifier. Source references and raw/owned/Drop
states remain outside this admitted input slice. Internal live-slot locations
are not a source-reference ABI. Aggregate copy eligibility is logical value
semantics, not permission to copy padding bytes or ignore lifetime.

Layout digests identify the checked producer snapshot; the Concept validator
checks relationships and geometry without recomputing JSON hashes. This is
consistency validation, not artifact authentication. PlainData stays separate
from CAbiValue and from native realization. The next Cathedral owners remain
native complete-object placement, address/copy realization, and aggregate ABI
classification. See [EVT2g2 evidence](../conformance/EVT2G-CONFORMANCE.md#evt2g2--typed-aggregate-and-explicit-layout-contract).

## EVT2g3 native object authority

This table retains the EVT2g3 stage boundary; the reference rows are superseded below.

The preceding native-storage stop is superseded for scalar-signature local
objects. Source semantic checking still establishes layout/copyability and
borrow legality; frozen CMIRAMD5 transports checked local object intent.

| Changed surface / decision | Classification and current owner | Evidence / next transition |
| --- | --- | --- |
| Complete-object placement, snapshots, field/array copy traversal | CONCEPT_AUTHORITATIVE: `AMD64.Objects` | Native contiguous source/private slots; consumes supplied geometry, skips padding; no duplicate record layout |
| Object operations, frame verification, module preflight and emission | CONCEPT_AUTHORITATIVE: `AMD64` | Real v5 dispatch; shared allocator/call/frame/fixup path; malformed input and unprobed frames fail before output mutation |
| Memory operands and typed scalar transfer | CONCEPT_AUTHORITATIVE: shared `AMD64.Encoding` | Existing ModRM/SIB/REX writer factored for checked addresses; width-2 support and 11 exact byte goldens |
| Native frame geometry and local-slot lookup | CONCEPT_AUTHORITATIVE: existing `AMD64.Frame` | Whole size/alignment enters one frame; local slots resolved by existing category-aware lookup |
| Aggregate-array/index LIR and scalar aliases | GO_AUTHORITATIVE checked-input production | Existing semantic element layout and bounds decisions retained; no native placement/copy decisions |
| Short-circuit RHS prelude placement and generated range declaration guard | BOOTSTRAP_ONLY portable C correctness | Independent Normal/Verify guarded bounds/side-effect tests; backend/oracle C headers coexist |
| `cmd/concept/amd64.go` object plan printing | Bootstrap orchestration / presentation | Calls actual Concept `PlanObjectNativeFunction`; does not duplicate planning |
| `native_object_test.go`, fixtures, updated older stop assertion | Test oracles | Artifact-only native, independent C, malformed/frame failures, bounds traps, exact bytes and 100-run determinism |
| Source reference LIR/transport/native realization | NOT_YET_QUALIFIED | Explicit source-authority stop; versioned borrow origin/access/lifetime contract required before location parameters/calls |
| Aggregate Win64 ABI classification | NOT_YET_QUALIFIED, Cathedral intended owner | Explicit `AMD64_UNSUPPORTED_AGGREGATE_ABI`; EVT2g4 remains separate |

The native copy algorithm realizes admitted value-copy permission; it does not
grant PlainData, CAbiValue, raw relocation, ownership or Drop authority. Source
borrow validity is not inferred from integer addresses or local-slot locations.
CMIRAMD3/4/5 schema/codecs remain frozen. See [EVT2g3 qualification](../conformance/EVT2G-CONFORMANCE.md#evt2g3--native-object-storage-progression).

## EVT2g3b reference authority

| Decision / surface | Owner and classification | Qualification / boundary |
| --- | --- | --- |
| Source borrow typing, liveness, mutability, lexical provenance and escape | GO_AUTHORITATIVE existing frontend | Existing place/lifetime/destructive-access checks retained; legal aliases remain legal |
| Checked reference-origin witness and typed LIR | Stage-0 checked semantic production | Internal witness follows existing validation; ref/ref_const types, source roots, scopes and canonical target digests; no native algorithm |
| CMIRAMD6 schema and metadata identity | CONCEPT_AUTHORITATIVE `BridgeSchema6` | Frozen v5 payload plus explicit authority tables; generated transport; v3/v4/v5 unchanged |
| Generated bridge support / `machineir_borrow.go` | Bootstrap artifact transport and verifier | Reuse frozen codecs; project checked identities; verify consistency; no physical pointers, GPR choices or caller offsets |
| Artifact reference admission | CONCEPT_AUTHORITATIVE `BridgeValidate6`, shared object consistency helpers | Source-free type/root/authority/scope/call preservation, typed Result errors, bounded arrays, NoAllocation; not a second source borrow checker |
| Native reference representation, dereference and calls | CONCEPT_AUTHORITATIVE `AMD64.Objects` / `AMD64` and existing allocator/Win64ABI/frame/encoder | Address virtuals after admission, existing positional ABI, stable source slots, original-object mutation; no Go native reference backend |
| Shared object validation / emission | CONCEPT_AUTHORITATIVE existing owners | Explicit v6 authority context; v5 public admission keeps empty context; one allocator/frame/memory/call/fixup path |
| RefExpr MIR child traversal / void LIR terminators | Stage-0 checked-input correctness | Preserve indexed-borrow bounds operations and canonical no-value returns; independent regression coverage |
| CLI inspection and diagnostics | Bootstrap presentation | Prints Concept plans and typed admission failures; no policy duplication |
| Trusted semantic facts | Existing registry, unchanged | User concepts cannot manufacture the checker-owned witness or grant liveness/outlives/NoAlias; metadata digests are not authentication |
| Reference-return / raw / owned / Drop / escaping reference semantics | NOT_YET_QUALIFIED at native boundary | Preserve explicit source/LIR stops; no weakening of memory ontology |
| Aggregate by-value/indirect/sret ABI | NOT_YET_QUALIFIED, Cathedral intended owner | Still `AMD64_UNSUPPORTED_AGGREGATE_ABI` in v5 and v6; EVT2g4 separate |

The native reference copy preserves an admitted address/capability; it does not
copy the aggregate value. Explicit source value copies retain the existing
fieldwise policy. Reference types do not gain PlainData or CAbiValue from pointer
width. See [EVT2g3b qualification](../conformance/EVT2G-CONFORMANCE.md#evt2g3b--checked-borrow-authority-and-native-references).

## EVT2g4 aggregate ABI authority

| Decision / surface | Owner and classification | Qualification / boundary |
| --- | --- | --- |
| Aggregate argument classification | CONCEPT_AUTHORITATIVE `Win64AggregateABI` | Exact 1/2/4/8-byte integer carriers; other record sizes indirect; checked copy evidence required |
| Aggregate return classification | CONCEPT_AUTHORITATIVE `Win64AggregateABI` | Separate C structural eligibility for direct RAX; hidden caller storage otherwise |
| Hidden pointer insertion and physical position shifts | CONCEPT_AUTHORITATIVE `AMD64.Objects`, existing `Win64ABI` | One ordered physical sequence; GPR/XMM and stack positions share the same ordinal |
| Caller argument temporaries / callee materialization | CONCEPT_AUTHORITATIVE `AMD64.Objects` / `AMD64.Aggregates` | Distinct storage identities, checked field-copy transfers, align 16 and call-covering lifetime; references remain separate |
| Result storage / capture / early-return transfers | CONCEPT_AUTHORITATIVE `AMD64.Aggregates` | Checked type/layout, stable storage, direct payload versus designated sret pointer, capture before preservation reload |
| Aggregate frame geometry | CONCEPT_AUTHORITATIVE existing `AMD64.Frame` | Shared locals/outgoing/saves/spills/move temps; maximum outgoing area, nonoverlap, align 16, stack-probe stop |
| Carrier allocation and parallel moves | CONCEPT_AUTHORITATIVE existing `AMD64.Analysis` / `Win64ABI` | Shared allocator and move replay; leaf GPR pressure may use existing callee-save pool |
| Checked source layouts, copy and borrow semantics | Existing Stage-0 authoritative semantic input | No packing calculator, ownership checker, fact registry change or ABI classifier added in Go |
| CMIRAMD6 and earlier schemas/codecs | Existing Concept schemas, unchanged | Existing semantic aggregate values/calls/results suffice; derived ABI state stays out of the artifact |
| CLI plan printing | BOOTSTRAP_ONLY presentation | Prints actual Concept classifications, placements, storage, moves and transfers |
| Classification tables, compiled Win64 C/LLVM probes and structural replay | Test oracles | Independent C record classifications; no production Go classifier |
| Aggregate call encoding / generated caller-callee execution / external conformance | NOT_YET_QUALIFIED | `AMD64_UNSUPPORTED_AGGREGATE_CALL_EMISSION`; EVT2g5 |

Static `layout` remains useful for bounded metadata and explicit storage maps;
it does not prove object initialization, copy permission or borrow authority.
Dynamic frame geometry continues to come from the single existing finalizer.

## EVT2g5 aggregate emission authority

| Decision / surface | Owner and classification | Qualification / boundary |
| --- | --- | --- |
| Aggregate ABI classification and physical plan | Existing CONCEPT_AUTHORITATIVE `Win64AggregateABI` / `AMD64.Aggregates` | EVT2g4 rules unchanged; separate argument/return eligibility; same typed plan validator |
| Direct carrier packing and parameter/result materialization | CONCEPT_AUTHORITATIVE `AMD64.AggregateEmission` | Zero physical carrier, copy meaningful fields through existing `CopyObjectFields`; no object padding read or packing inference |
| Indirect argument initialization and pointer placement | CONCEPT_AUTHORITATIVE transfer selector and existing scalar call machinery | Distinct typed temporary, actual pointer alignment 16, existing parallel moves and positional destinations |
| Hidden result binding / construction / RAX identity | CONCEPT_AUTHORITATIVE transfer selector | Stable caller result storage; shifted physical arguments; all return paths copy fields and return the designated pointer; call result pointer checked |
| Call encoding / preservation / fixups | Existing Concept `RealizeAllocatedCall`, scalar actions, placeholder writer and resolver | Aggregate transfers surround the same call actions; direct result capture precedes reload; forward/backward/recursive calls qualified |
| Frame and scratch storage | Existing Concept frame owner | One frame, fixed RSP, one shadow area, existing scratch/spill/save slots; observed 4088-byte native frame, larger probing requirement rejected |
| Failure preflight | CONCEPT_AUTHORITATIVE existing byte/fixup writers with count-only mode | Same opcode selectors and capacity checks before writes; checked plans before emission; measured images are never finalized/published |
| Go production change | BOOTSTRAP_ONLY generated C header correctness | Natural built-in array typedef guards allow backend/oracle headers to coexist; nominal and layout-strengthened declarations remain separately owned |
| Independent C ABI qualification | Test oracles: separately compiled MSVC/Clang C DLLs plus independent ml64 boundary observer | Both directions, positional GPR/XMM/stack, sret RAX, shadow/RSP, temporary pointers, GPR/XMM nonvolatiles; test-only tail thunks, no production external linker |
| Wire contract / source semantics | Existing checked source and CMIRAMD3/4/5/6 owners, unchanged | No new schema/hash/tags/codecs, source borrow rule, copy permission, CAbiValue or NoAlias grant |
| Owned/Drop, escaping reference returns, general storage views, varargs/indirect calls, object output | NOT_YET_QUALIFIED | Explicit existing boundaries retained; no EVT2h or object-file work begun |

See [EVT2g5 execution and ABI evidence](../conformance/EVT2G-CONFORMANCE.md#evt2g5--native-aggregate-calls-and-win64-abi-closeout).

## EVT2h1 ordinary enum authority

| Decision / surface | Owner and classification | Qualification / boundary |
| --- | --- | --- |
| Declaration identity, legal constructors/tags, equality, assignment and conversions | Existing Stage-0 source checks, GO_AUTHORITATIVE | Existing language retained; no enum/integer casts or payload equality |
| Checked ordinary enum LIR domains and producer verification | Stage-0 semantic production | Explicit nominal identities, checked geometry and ordered variants; no physical native decisions |
| CMIRAMD7 schema | CONCEPT_AUTHORITATIVE `BridgeSchema7` | Frozen v5 records plus enum domains; schema-generated Go/Concept transport; CMIRAMD3–6 unchanged |
| Native artifact admission and representation validation | CONCEPT_AUTHORITATIVE `BridgeValidate7` and common typed object verifier | Independent source-free domain/geometry/tag/op/call/CFG validation before mutation; aliases unavailable to v5/v6 |
| Native realization, scalar selection, storage, allocation, frames and Win64 calls | CONCEPT_AUTHORITATIVE existing `AMD64.Objects`, scalar and Win64 stack | Admitted U32 carriers reuse one allocator/frame/encoder; nominal IDs remain checked until physical projection |
| Bootstrap codec, CLI and tests | Generated transport / orchestration / independent oracles | No production Go enum lowering, allocation, ABI or encoding algorithm |
| Payloads, match, built-in fallible types, mixed object/reference enum modules | NOT_YET_QUALIFIED | Source C support does not imply native qualification; separate review after EVT2h1 |

See [the enum contract and executed evidence](../conformance/EVT2H-CONFORMANCE.md).
