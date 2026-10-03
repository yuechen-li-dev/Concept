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
| MachineIR | Go `machineir*.go`; Concept backend model | Shared | Cathedral | Needs prerequisite for calls | CMIRAMD2 transport | Schema/IR/native parity | Concept native path owns new backend decisions | Retain seed producer | EVT2e onward |
| MachineIR bridge | Concept `BridgeSchema/Derive/Read`; generated Go/Concept codec | Shared | Shared bootstrap seam | Ready for present schema | Stage-0 emits CMIRAMD2 | Roundtrip/hash/native bytes | Already switched to sole Concept schema in R9a2 | Generated Go codec retained seed | R9a2; version for EVT2e |
| Liveness | Concept `Backend/AMD64.concept` | Concept | Cathedral | Ready for current no-call CFG | MachineIR input | Native/corpus oracle | Already Concept-owned for existing path | No Go backend counterpart to retire | C1; extend EVT2e |
| Register allocation | Concept `Backend/AMD64.concept` | Concept | Cathedral | Ready for bounded current path | CMIRAMD2 vregs | Assignment/native bytes | Already Concept-owned | No Go backend counterpart | C1; extend EVT2e |
| Frame layout | Go `machineir.go` ABI helpers; Concept `FinalizeFrame` | Shared | Cathedral | Needs call contract | Incoming ABI and slots | Frame/ABI probes | Concept owns call and save frames | Retain seed ABI construction | EVT2e |
| ABI lowering | Go `Win64Argument/Return`; Concept partial frame/encoder | Shared | Cathedral | Needs call IR | Incoming ABI descriptors | Native ABI probes | Concept owns outgoing/incoming policy | Retain seed input only | EVT2e |
| Native encoder | Concept `Backend/AMD64.concept` | Concept | Cathedral | Ready for current operations | CMIRAMD2, C host compilation | Exact bytes and execution | Already Concept-owned | No Go encoder to retire | C1; extend EVT2e |
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
