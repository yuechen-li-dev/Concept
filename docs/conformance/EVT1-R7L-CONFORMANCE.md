# EVT1 R7l conformance: meaningful progression

Baseline: `37babd3afd8eda6fedb49a7ea379af69cc45496c`; compiler
`concept-evt1-stage0-go`.

The landed slice consists of typed AMD64 `Pause`, `ReadTimestamp`, and
8/16/32-bit port input/output, deterministic `machine_intrinsic` MIR operations,
and one-operand structured AMD64 inline asm with typed `in`/`out`/`inout`,
clobber, and memory-effect fields,
inspectable `.machine.S` generation, target rejection for explicit AArch64,
artifact-only import, transitive hardware facts, and DragonGod port UART
dogfood. Native execution covers Pause and timestamp on the AMD64 host. Port
instructions are compiled and linked but not executed in user mode. The
generated C remains strict C11; target instruction text lives in `.machine.S`.
The native asm fixture executes separate input, output, and read/write cases,
including a declared named register clobber. An artifact-only consumer retains
asm and MMIO–asm–MMIO ordering in MIR and the declared memory effect in the
planner. These checks do not establish a CPU fence.
`TestAMD64MachineArtifactsAreByteIdenticalAcross100Runs` compares fresh MIR,
C, helper, manifest, maps, and semantic module artifact bytes for 100 runs.
`TestStructuredAsmArtifactsAreByteIdenticalAcross100Runs` applies the same
byte comparison to the bounded asm form.

This is not full R7l success. Multi-operand asm, CPUID, privileged
instruction classification, AArch64 operations, CPU barriers, async/machine
interaction, MMIO barrier ordering, and 100-run proof/explain evidence are
outstanding. The bounded asm form and its deliberate limits are described in
`docs/language/INLINE-ASSEMBLY.md`.

## Final gates

| Gate | Result |
| --- | --- |
| `go test ./...` | Passed; `internal/concept` 237.809 s |
| `go vet ./...` | Passed |
| Root and legacy `zig build test` | Both passed |
| `go test ./internal/concept -run R7d3 -count=1` (BurnIn command from `Make.oct`) | Passed; `oct` executable unavailable on this host |
| Standard package test | 29 passed, 0 failed |
| DragonGod package test | 23 passed, 0 failed, 1 benchmark |
| Full EVT1 semantic corpus in `TestSemanticCorpusManifest` | 398 valid, 262 static-invalid, 13 runtime-negative fixtures accepted at compile time |
| R7k/R7x and focused R7l tests | Passed |

Compiler source audit found no UART or DragonGod-specific branches outside
tests. MIR contains general `machine_intrinsic` and `machine_asm` operations;
there is no runtime intrinsic registry or helper allocator.
