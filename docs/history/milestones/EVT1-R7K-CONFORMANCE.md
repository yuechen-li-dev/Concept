# EVT1 R7k conformance

Baseline: `37a11a2452f06b3311b7a6c55f209fedf7fa1f28` (`R7j1`), compiler
`concept-evt1-stage0-go`.

R7k keeps the established R6g affine address algebra for every space:
`Address<Space> + usize<byte>` yields an address, and subtracting two
addresses of the same space yields signed `isize<byte>`. The latter differs
from the unsigned spelling in the R7k request and remains signed so negative
distances and existing source contracts retain their meaning. Address plus
address still rejects; `DeviceMemory` receives no special arithmetic rule.

The focused `TestMmioAndBitsReachMIRPlannerAndC11` and
`TestUartBitsAndHardwareEffectsSurviveArtifactOnlyImport` exercise the real
parser, semantic analysis, MIR, planner, C11 generation, artifact import, and
hosted execution paths. The C adapter records three separate 8-bit reads
(including an unused result), two identical writes, then 16/32/64-bit
reads/writes, then a distinct-address store/read/store sequence, with exact
address/width/value/sequence assertions. The imported
UART runs status read, byte write, ready poll, and bounded unsuccessful poll
against hosted register bytes.

`TestBitsInvalidLayouts` covers reversed/out-of-width/overlapping/duplicate
fields and a field value that exceeds its range. MMIO negative tests cover
wrong width, aggregate type, wrong address space, ordinary reference, literal
overflow, and visible constant misalignment. Existing R6g tests retain the
affine address rule and unknown-provenance storage rejection.
The hosted harness also checks a valid dynamic multi-bit insertion preserves
reserved bits and an out-of-range dynamic value traps.

The generated C uses width-specific volatile expressions only at explicit
MMIO calls. The host tests compile with
`-std=c11 -pedantic -pedantic-errors -O2` under Clang and GCC.
`HardwareRead` and `HardwareWrite` facts are derived on operations and their
transitive callers and carried as artifact summaries; DragonGod package tests
prove both facts plus `NoAllocation` for UART paths.

The standalone corpus fixture
`language/evt1/tooling/hardware-memory/valid/mmio_facts.concept` permits direct proof
inspection:

```text
concept explain language/evt1/tooling/hardware-memory/valid/mmio_facts.concept:23 --json
  NoAllocation(ReadStatus) = proven
concept explain language/evt1/tooling/hardware-memory/valid/mmio_facts.concept:24 --json
  HardwareRead(ReadStatus) = proven
concept explain language/evt1/tooling/hardware-memory/valid/mmio_facts.concept:25 --json
  HardwareWrite(WriteStatus) = proven
```

`TestOrdinaryMemoryDoesNotAcquireMmioSemantics` confirms an ordinary reference
read emits neither volatile C nor a hardware MIR operation.
`TestDeviceRegionUsesExistingForeignAuthority` checks that a foreign
`ExternalStorage<DeviceMemory>` contract can establish a bounded
`MemoryRegion<DeviceMemory>` through the existing region authority.

`TestMmioArtifactsAndPlanAreByteIdenticalAcross100Runs` compares fresh MIR,
generated C, semantic proof outputs, planner output, and module artifacts.
The hosted adapter executes the same trace scenario 100 times per available
C compiler.

See `EVT1-R7K-CONVERGENCE.md` for commands and outcomes. No CPU-level fence,
real device mapping, or bare-metal boot claim is made.
