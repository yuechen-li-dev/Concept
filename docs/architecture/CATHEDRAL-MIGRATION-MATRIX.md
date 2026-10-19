# Cathedral migration matrix (R9c)

This inventory is a decision map, not a schedule to delete Stage-0. `Current`
classifies implementation as Go, Concept, or Shared. `Desired` means Stage-0
permanent, Cathedral, or Shared bootstrap seam. `Ready` means an existing
Concept contract can support a bounded migration; it does not mean a normal
Cathedral executable exists. “Retain seed” is a deliberate BOOTSTRAP-ONLY
outcome, not an unfulfilled deletion. Paths identify the audited owner.

| Subsystem | Current implementation | Current authority | Desired authority | Concept readiness | Bootstrap dependency | Shadow strategy | Switch condition | Go retirement condition | Expected milestone |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Lexer | Go `parse.go` | Go | Stage-0 permanent, later Cathedral frontend | Needs prerequisite | Source admission | Token/diagnostic corpus | Native frontend can parse same language | Retain seed | Later frontend |
| Parser | Go `parse.go` | Go | Shared bootstrap seam | Needs prerequisite | Syntax and spans | AST/diagnostic corpus | Checked Cathedral parser and artifacts | Retain seed | Later frontend |
| Syntax tree | Go `types.go` | Go | Shared bootstrap seam | Needs prerequisite | Stage-0 AST | Structural artifact comparison | Stable external syntax contract | Retain seed | Later frontend |
| Name resolution | Go `validate.go` | Go | Cathedral | Needs prerequisite | Declaration graph | Identity/error parity | Concept graph owns normal lookup | Retain seed | Later frontend |
| Type system | Go `validate.go`, `types.go` | Go | Cathedral | Needs prerequisite | Canonical type identity | Type/artifact parity | Concept types own normal legality | Retain seed | Later frontend |
| Generic closure | Go generic/validate code | Go | Cathedral | Needs prerequisite | Closed instance construction | Structural identity/corpus parity | Concept closure owns normal instances | Retain seed | Later frontend |
| Comptime evaluator | Go `comptime*.go` | Go | Cathedral | Needs prerequisite | Bounded evaluation | Values/errors/fuel parity | Concept evaluator owns normal decisions | Retain seed | Later frontend |
| Concept/proof evaluator | Go `innate*`, `proof_graph.go`; Concept predicates | Shared | Cathedral with trusted kernel | Needs prerequisite | Checked observations/provenance | Verdict and proof graph parity | Concept rules own decisions; kernel audited | Retain seed kernel | Later semantic |
| Innate concept host | Go `innate_apply.go`; `innate/Innate.concept` | Shared | Shared bootstrap seam | Keep Stage-0 kernel | Compiler observations | Valid/invalid rule corpus | Concept rule replaces individual Go rule | Retain seed host | Incremental |
| Artifact reader/writer | Go `module_artifact.go` | Go | Shared bootstrap seam | Needs prerequisite | Versioned module transport | Canonical roundtrip/parity | Cathedral can load/write checked artifacts | Retain seed | Later artifact |
| MIR | Go MIR/semantic facts | Go | Cathedral | Needs prerequisite | Validated typed input | MIR/semantic-fact parity | Concept owns normal MIR production | Retain seed | Later IR |
| Planner | Go `planner.go` | Go | Cathedral | Needs prerequisite | MIR facts | Plan/guard decisions | Concept planner selected normally | Retain seed | C2 |
| LIR | Go `lir*.go` | Go | Cathedral | Needs prerequisite | MIR/Planner | Verified LIR parity | Concept transform and verifier own normal path | Retain seed | C2 |
| Scalar floating input | Go `lir.go/lir_lower.go` and checked virtual MachineIR projection | Stage-0 checked semantic production | Cathedral native consumer | Exact float32/float64 LIR and CMIRAMD4 transport qualified | Existing semantic float and quantity model | 100-run LIR/MachineIR; 52 malformed artifacts; Normal/Verify artifact-only roundtrip | Concept physical allocation/ABI/frame, SSE and fixed-signature native calls qualified through EVT2f5 | Retain seed input only; no Go physical XMM algorithm | EVT2f5 native scalar closure |
| MachineIR | Go `machineir*.go`; Concept backend model | Shared | Cathedral | Direct scalar call contract feeds qualified native calls | CMIRAMD3 transport | Schema/IR/native parity | Concept native path owns new backend decisions | Retain seed producer | EVT2e5 |
| MachineIR bridge | Concept `BridgeSchema[4]/Derive/Read/Validate[4]`; generated Go/Concept codec | Concept contract/admission; shared bootstrap transport | Shared bootstrap seam | Frozen CMIRAMD3 and virtual CMIRAMD4 qualified | Stage-0 emits explicitly selected version | Exact bytes/hash/artifact-only roundtrip; 23 v3/52 v4 malformed cases | Versioned Concept schemas; normal whole-artifact admission selected | Generated Go codec retained seed | R9a2; EVT2e2; EVT2f2 |
| Liveness | Concept `Backend/AMD64.concept` | Concept | Cathedral | Call argument uses and exact live-across classification qualified | MachineIR input | Native/corpus oracle; 100-run summaries | Concept clobber analysis selected normally | No Go backend counterpart to retire | EVT2e3 |
| Register allocation | Concept `Backend/AMD64.concept` | Concept | Cathedral | Fixed call constraints, preservation and used-callee-saved summaries ready | CMIRAMD3 vregs | Assignment/native bytes and call plan fixtures | Concept call planning and concrete saves/spills selected | No Go backend counterpart | EVT2e3; realization EVT2e4 |
| Frame layout | Concept `RealizeFrame/FinalizeFrame` | Concept | Cathedral | Unified local/spill/save/temp/outgoing geometry qualified | Incoming ABI and slots | Independent frame/action replay, 100-run plans | Concept owns final native offsets and frame actions | Retain incoming seed descriptors only | EVT2e4 |
| ABI lowering | Go incoming `Win64Argument/Return`; Concept `Win64ABI` outgoing planner | Shared with distinct scopes | Cathedral | Outgoing scalar placement/clobber/move/frame actions and native execution qualified | CMIRAMD3 calls and incoming descriptors | C11 parity, native register/RSP/canary/alignment probes | Concept owns outgoing policy; incoming seed remains | Retain seed input only | EVT2e5 |
| Parallel argument moves | Concept `Win64ABI` | Concept | Cathedral | Cycles/fanout/stack stores qualified with concrete frame temporaries | Allocated call sources | Independent value replay; 256 combinations; 100-run output | Concept plans normally | No Go counterpart | EVT2e3 |
| Native encoder | Concept `Backend/AMD64.concept` | Concept | Cathedral | Direct internal CALL rel32 and concrete frame actions qualified | CMIRAMD3, C host compilation | Exact bytes, C11 parity and ABI probes | Normal CLI selects module emission | No Go encoder to retire | EVT2e5 |
| Native symbols and call fixups | Concept `EmitModule/ResolveNativeCalls` | Concept | Cathedral | Semantic identities and declaration-order layout qualified | Checked function identities | Forward/backward/recursive calls, range and patch rejection, 100-run final bytes | Concept owns normal layout and patching | Bootstrap host publishes only finalized images | EVT2e5 |
| C backend | Go C generator | Go | Stage-0 permanent | Keep Stage-0 | External C toolchain | Strict C11/native parity | Native becomes normal after qualification | Retain portable seed/fallback | Long-term |
| Diagnostics | Go diagnostics/proof graph; Concept `Describe` | Shared | Cathedral rules, shared bootstrap transport | Needs prerequisite | Source spans/proof IDs | Exact error/site/explain parity | Concept rule selects normal diagnostic | Retain seed renderer | Incremental |
| Formatter | Go `format.go` | Go | Shared bootstrap seam | Needs prerequisite | Parsed source | Idempotence/golden parity | Cathedral formatter if useful | Retain seed CLI | Later tooling |
| Linter | Go `project_policy.go`; Concept predicates | Shared | Cathedral policy, Stage-0 host | Ready for policy rules | Observation API | Finding/proof parity | Concept predicates own checked rules | Retain seed runner | Incremental |
| Generated declarations | Go `generated_declarations.go`; Concept authored generators | Shared | Cathedral | Needs prerequisite | Reflection and checked AST insertion | Declaration/artifact parity | Concept generation owns normal decisions | Retain seed materializer | Later frontend |
| Reflection | Go `reflection.go`; Concept reflection consumers | Shared | Shared bootstrap seam | Needs prerequisite | Typed declaration graph | Ordered field/case/identity parity | Cathedral graph supplies same contract | Retain seed observations | Later frontend |
| Policy engine | Go `project_policy.go`; Concept manifest rules | Shared | Cathedral rules, shared host | Ready for bounded rules | Manifest activation | Lint/explain parity | Concept rule authority maintained | Retain seed orchestration | Incremental |
| Condition vocabulary helper | Go verifier switch; Concept `BridgeSchema.Condition` | Shared before R9c | Cathedral contract, generated seed projection | Ready | CMIRAMD2 generated table | Frozen truth-table agreement | Go verifier now consumes Concept-derived table | RETIRE hand-written Go switch; retain generated table | R9c |

The current owner of `Standard.Machine.AMD64.concept` is typed hardware
intrinsics and wrappers, not backend allocation/encoding. `BridgeDerive` and
`BridgeRead` are Concept-owned checked bridge consumers; `BridgeSchema` is the
single wire authority. The table deliberately distinguishes code presence
from decision authority. See the authority ledger for live state and next step.

EVT2e4: call contracts, outgoing ABI planning, final native stack geometry,
spill/save/temp realization and prologue/epilogue/call actions are Concept
owned. Go changes supply checked fixtures, bootstrap orchestration, independent
test oracles and diagnostic presentation. EVT2e5 adds Concept-owned CALL
encoding, semantic symbol layout and checked fixups; native Win64 scalar
internal execution closes EVT2e for its bounded subset. CMIRAMD3 is unchanged. Native
stack slots have category provenance and post-prologue RSP offsets; they do not
replace DragonGod/automata persistent MachineFrame state.

EVT2f2 advances checked scalar floating input through a separate CMIRAMD4 virtual
contract. Existing integer-only modules still use frozen CMIRAMD3. The CMIRAMD4
prerequisite recorded in EVT2f1 is now qualified. The current bounded transitions are:

| Capability | Current authority/state | Bootstrap dependency | Next switch condition |
| --- | --- | --- | --- |
| Checked float32/float64 input | Stage-0 semantic producer/verifier; ready | Existing checked LIR | Later frontend migration only |
| CMIRAMD4 scalar/XMM schema | Concept authoritative; qualified | Reflection/materialization host | Version future changes explicitly |
| CMIRAMD4 transport/admission | Generated Go/Concept transport; Concept whole-artifact admission qualified | Checked virtual projection and C bootstrap host | Preserve artifact-only parity and exact bits |
| Physical XMM allocator | Concept authoritative; EVT2f3 qualified | Admitted virtual class/value identities | Later optimization/nonvolatile qualification |
| Mixed positional Win64 ABI and XMM clobbers | Concept authoritative; EVT2f3 qualified | Ordered checked types; presentation host | EVT2f5 native action realization and independent Win64 probes qualified |
| Float spills and native frame integration | Concept authoritative; EVT2f3 planning and EVT2f5 native qualified | Typed semantic locals | Preserve typed action/frame validation and native pressure regressions |
| Scalar SSE2 encoding | Concept authoritative; EVT2f4 qualified | Semantic scalar ops and exact bits | Later unsupported forms/optimizations only |
| No-call native float execution | Concept authoritative; EVT2f4 qualified | Existing Win64 RW-to-RX host and independent strict C11 oracle | EVT2f5 extends the same path with generated mixed calls |

The normal backend validates CMIRAMD4, derives and qualifies XMM allocation,
mixed Win64 calls, preservation, moves and one concrete frame, then stops at
`AMD64_UNSUPPORTED_SSE_ENCODING`. Go retains no physical floating algorithm.
The 100-run Normal/Verify plans and independent byte-stack replay qualify planning;
they do not establish native floating execution. CMIRAMD3/4 remain frozen.


EVT2f4 advances the no-call CMIRAMD4 path through real SSE2 encoding and native
Win64 execution. The current stop for call-bearing modules is
`AMD64_UNSUPPORTED_FLOAT_CALL_EMISSION`. Immediate exact-bit constants require no
pool or process-address relocation. Existing allocation and seven-region frame
geometry remain authoritative; derived scratch uses four generalized temporary
slots. Source/transport/byte/native oracles qualify the bounded arithmetic,
comparison, scalar-local, incoming-array and pressure subset. Earlier SSE-stop
statements above describe EVT2f3 history; no production Go SSE path exists.

EVT2f5 closes the fixed-signature scalar direct-call path: Concept realizes mixed
GPR/XMM actions, captures RAX/XMM0 results before reloads, and reuses native
semantic symbols and checked rel32 fixups. Source fixtures execute arities zero
through eight, nested/recursive/multiple calls, live-across state, cycles and
pressure spills in Normal/Verify. Compiled C probes independently observe the
Win64 positional ABI and deliberately clobber volatile XMM registers. The
production image remains address-independent and fully fixed before RX.

The historical EVT2f4 stop above no longer applies to this subset. Stage-0
remains checked input/transport, bootstrap host and diagnostics. No Go mixed-call
backend was introduced. CMIRAMD3/4 remain frozen; external linkage,
indirect/variadic/aggregate calls, XMM6-XMM15 allocation and additional targets
remain separate work.

## EVT2g aggregate input progression

| Capability | Current authority/state | Next switch condition |
| --- | --- | --- |
| Source aggregate layout/copyability/place semantics | Existing checked Stage-0 frontend; retained without a second type model | Future frontend migration only |
| Typed aggregate LIR and immutable `with` | Stage-0 checked MIR/LIR construction; qualified bounded input | New checked aggregate bridge must preserve layout, object and copy/update intent |
| Complete local initialization / derived locations | Stage-0 producer verifier; bounded, nonescaping value-object evidence | Concept artifact validator independently admits transported lifetime/provenance evidence |
| Aggregate MachineIR artifact | NOT_YET_QUALIFIED; CMIRAMD3/4 frozen | Checked Concept schema plus generated codecs/readers and artifact-only malformed-input tests |
| Native aggregate memory/copy/reference/ABI | NOT_YET_QUALIFIED; Cathedral intended owner | Ordinary Concept implementation, native execution, independent Win64 ABI probes |

The normal aggregate path stops at `MIR_AGGREGATE_REQUIRES_CMIRAMD5`, including
small and unused aggregate locals. Existing scalar native qualification is
unchanged. C bootstrap `with` array replacements use the existing expected-type
initializer seam; this is portable C seed correctness, not native copy policy.

## EVT2g2 checked object contract

The EVT2g1 table/stop above is retained as history. CMIRAMD5 now carries checked
objects; CMIRAMD3/4 continue serving their frozen scalar subsets.

| Capability | Current owner/state | Next switch condition |
| --- | --- | --- |
| Source aggregate type/layout/copyability | Existing checked frontend, qualified bounded input | Frontend migration retains the same canonical semantic layout observations |
| Typed aggregate MachineIR and explicit geometry | Concept schema authoritative; generated transport qualified | Preserve version/hash, typed values/storage/locations, nested identities and array geometry |
| Artifact-only consistency admission | Concept `BridgeValidate5`, qualified | Preserve source-free Normal/Verify exact roundtrip, typed errors, bounds and no-allocation proofs |
| Immutable `with` | Checked frontend evaluation order, ordinary logical v5 copy/member operations | Later Cathedral chooses realization without a WITH instruction |
| Semantic aggregate calls/results | V5 unclassified argument/result types, qualified | Concept Win64 aggregate classifier, separate from byte width and CAbiValue |
| Native aggregate complete-object storage | Cathedral, pending EVT2g3 | Native placement consumes supplied size/alignment; no field layout recalculation |
| Native aggregate field access/copy | Cathedral, pending | Typed address/instruction and copy strategies with real execution evidence |
| Source-reference/raw/owned/Drop aggregate transport | Existing explicit checked-input stop | Separate admitted semantics and lifetime evidence; no fabrication from internal locations |
| Aggregate Win64 register/indirect/sret/stack placement | Cathedral, pending | Dedicated semantic/layout ABI plan and independent ABI probes |

`concept lir` and `concept machineir` succeed on the bounded object corpus.
`concept amd64` validates its artifact then returns
`AMD64_UNSUPPORTED_AGGREGATE_STORAGE` before changing output/image. Native
scalar, SSE, direct calls and automata frames retain their earlier qualification.
Mixing aggregate objects with automata activation frames remains an explicit
v5 input boundary. EVT2g2 does not begin physical aggregate lowering.
