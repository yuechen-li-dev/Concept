# R7l convergence log

Baseline: `37babd3afd8eda6fedb49a7ea379af69cc45496c`, compiler
`concept-evt1-stage0-go`.

| Blocker | Reproducer | Root cause | Fix | Affected target and dogfood |
| --- | --- | --- | --- | --- |
| Machine annotation rejected as test metadata | `concept check libraries/Standard/Machine/AMD64.concept` returned `TEST_ATTRIBUTE_UNKNOWN` | Function attributes were all treated as test attributes | Admit compiler-owned machine metadata and exclude it from test discovery | Standard artifact and DragonGod port UART test |
| Imported machine declarations discovered as tests | `concept package test DragonGod` first reported four `port_uart::ConceptAMD64*` failures | Test discovery treated every non-access attribute as a test | Exclude machine metadata from discovery | DragonGod package test returns 23 passed, 0 failed |
| C11 cannot encode port instructions | Strict C11 generated wrapper compilation | Port I/O is a CPU instruction, not C memory access | Emit deterministic AMD64 `.machine.S` alongside strict C11 wrappers and link it in Concept tests | GCC AMD64 native Pause/timestamp test; port codegen only |
| Imported NoAllocation proof became Unknown | `TestAMD64MachineIntrinsicArtifactOnlyConsumer` after making local proof traverse machine calls | Artifact effect summarizer still treated the typed extern as opaque | Classify validated machine declarations as compiler-known nonallocating before the opaque-extern rule | Artifact-only Standard consumer and DragonGod package |
| Structured asm had no semantic statement | `TestStructuredAMD64AsmOperandsMIRAndExecution` | Source could express only opaque extern calls or typed known intrinsics | Add bounded `unsafe asm AMD64` with one typed register operand, clobbers, memory effect, MIR transport, and generated helper | Native AMD64 in/out/inout execution and artifact-only import |

The remaining asm blocker is multiple operands with a general explicit
register allocation and clobber contract. CPUID, privileged operations, and
AArch64 support also remain open; none are simulated by the current helper
path.
