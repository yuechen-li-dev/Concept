# R8f differentiator goldens

These programs live in `libraries/Golden/Differentiators`. They are independent
of the permanent R7p migration goldens and the R7q Frame. The Concept source,
not this note, is the executable authority. First drafts are retained beside
each new program as `first-draft.concept.txt`.

| Program | Source LOC | Problem and architecture | Concept mechanisms | Ordinary C++ or Rust machinery |
| --- | ---: | --- | --- | --- |
| Agents/Squad | 140 | Three tactical agents turn bounded observations into features, `Inference<Policy>` scores, and a policy-specific temporal machine. A score bonus plus state duration gives commitment. | Inference, hard maximum, automata, `transition match`, `yield`, `NoAllocation` | A utility scorer, state-machine code, and an explicit commitment convention |
| Mechanics/Stress | 22 | Rotate a 2D plane-stress tensor from raw native pascals and report shear at explicit display precision. | Einstein contraction, fixed tensor extents, `double<Pa>`, `interpret`, `as float<Pa>`, `SameDimension`, `NoAllocation` | Unit and tensor libraries with compatible expression templates and separate raw-data conventions |
| Hpc/Dot | 66 | One four-lane dot algorithm closes for `float` and `double`; binary32 uses paired accumulation, binary64 uses sequential accumulation. A direct binary32 baseline shows the expected code shape. | `Floating<T>` and required operations, fixed arrays and spans, `SizeOf<T>()` specialization, `NoAllocation` | Traits/operator bounds, templates, and specialization/codegen review |
| Compiler/IR + IRTools | 54 | A typed instruction schema marks operand IDs. A second module reflects that schema and generates a checked operand census used by an arity verifier. | `[[reflect]]`, `Fields<T>(operand)`, `derive`, generated provenance, artifact-only import, `NoAllocation` | Visitor or code generator plus schema synchronization |
| Async/Journal | 64 | A bounded journal transaction fetches, transforms, and commits; errors propagate through two suspensions while owned request state persists. | `async`/`await`, owned state, scoped ref, `Result`, `[[must_use]]`, `discard`, `NoAllocation` | Coroutine/task storage and explicit ownership/lifetime discipline |

## First-draft friction

| Program | Expected mechanism; natural attempt and actual response | Docs sufficient? | Final composition and classification |
| --- | --- | --- | --- |
| Agents | Expected inference to drive a temporal machine. The draft used a ternary expression for `armed` (`CV4020`) and put a transition after `yield` (`MACHINE_FRAME_INVALID`). | Partial: terminal-step rule needed clearer examples. | Bound the feature with a short `if`; kept each machine step terminal and represented minimum commitment in state. Existing semantic feature and documentation gap. |
| Mechanics | Expected tensor contraction over pressures. Native float literals would not initialize `tensor<double>` (`CV4227`). | Partial: representation rules exist, but no double-tensor example. | Cast representation explicitly before constructing the tensor or interpreting pressure. Units and Einstein contraction compose. Documentation gap. |
| HPC | Expected `Floating<T>` to allow arithmetic. Open `T` received `CV4175`; native C then lacked closed `ReadOnlySpan<float/double>` typedefs. | Partial: generic operation closure is documented, but not this numeric composition. | Used required operations with scalar witnesses; fixed the header collector and erased the closed size choice. Existing generic boundary and compiler correctness fixes. |
| Compiler | Expected reflection to generate a checked IR verifier. The first derivation worked, but IR and generator initially shared a file, so artifact transport was unproven. | Yes for reflection and derive; artifact-only example design needed work. | Split schema A and generator B; consumer C uses semantic artifacts alone. Example design correction. |
| Async | Expected await and Result to preserve owned request state and errors. Early error returns emitted value returns from a `void` C step; `[[must_use]]` was mistaken for a test annotation. | Partial: examples did not expose these implementation defects. | Split return-bearing branches in async CFG lowering and excluded `must_use` from test discovery. Ownership and error propagation now compose. Compiler and tooling correctness fixes. |

The first drafts reflect the domain structure before compiler feedback. The
final versions were formatted with `concept format` and checked individually;
the R7q Frame files were restored untouched after a project-wide format pass.

## Mechanism evidence

The game fact creates an armed fighter, an unthreatened scout, and an unarmed
wounded agent. Their machines choose Engage, Patrol, and Retreat respectively;
the fighter reaches Fire and the scout remains on patrol across steps. The
scorer has no heap allocation. The model keeps utility selection distinct from
the progression of AcquireTarget, Fire, and Cooldown.

The mechanics fact checks that a quarter-turn reverses the shear component.
The dimension-invalid contraction reports `CV4615`; incompatible contraction
axes report `CV4620`. `concept explain` proves that `double<Pa>` and
`double<N/m^2>` have normalized equal dimensions. Raw sensor scalars enter with
`interpret` rather than an implicit unit assumption.

The HPC fact runs the same source algorithm for two scalar representations.
Generated C has distinct `concept_template_dot4__float` and
`concept_template_dot4__double` functions and no constant `if` branch. Fixed
arrays and spans are inline; required-operation witnesses are ordinary static
calls. This is a code-shape observation, not a throughput claim.

`concept generated` reports the checked `OperandCount` declaration, its
`GeneratedByReflection` identity, and the two `NodeId` operands that supplied
it. `concept explain` proves its `NoAllocation` summary through an imported
semantic artifact. The artifact-only consumer imports A and B and calls both
`OperandCount` and `ValidArity` without source reparse.

The async fact checks successful transfer, missing input, and full journal
failure. A source-negative borrow across suspension reports
`ASYNC_PERSISTENT_REF_ESCAPE`; ignoring a MustUse return reports
`MUST_USE_RESULT_IGNORED`. `SendBestEffort` deliberately uses `discard` for
optional journal telemetry. Generated C uses bounded inline async frames and
explicit step/finish operations.

## Qualification

Run from the repository root:

```powershell
$env:CONCEPT_MODULE_ROOTS = (Resolve-Path 'libraries').Path
go run ./cmd/concept lint libraries/Golden/Differentiators
go run ./cmd/concept test libraries/Golden --filter Differentiators --verbose
go run ./cmd/concept test libraries/Golden --filter Differentiators --verify
go test ./internal/concept -run '^TestR8f' -count=1
```

The project manifest applies `ProjectNaming` and exact `NoAllocation`
requirements to the dot, stress, and agent-feature hot paths. Formatting is
checked on each new `.concept`, `.concept_test`, and manifest file. The focused
Go tests pin negative diagnostic categories, strict GCC and Clang C11 builds,
artifact-only consumption, and 100-run artifact/C identity for representative
HPC, game, async, and compiler modules.
