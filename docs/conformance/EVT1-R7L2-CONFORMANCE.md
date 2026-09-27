# EVT1 R7l2 conformance

Baseline `58c180966e0680968d8acb5b643cbb7ce4658308`, compiler
`concept-evt1-stage0-go`. The integration base also includes Claude's
`fd3bd85` test optimizations; R7l2 changes were isolated and then reconciled
against that commit.

## Implemented and checked

- Structured AMD64 asm has ordered typed `in`, `out`, and `inout` operands;
  fixed EAX/EBX/ECX/EDX requirements; explicit clobbers and memory effects;
  duplicate, overlap, clobber, type, and target diagnostics.
- Its deterministic helper uses one pointer to eight-byte operand slots,
  preserving RBX and R12 as required by the ABI. Native `add`, `xchg`, and
  `bswap` fixtures execute under strict C11 C compilation.
- `Cpuid` has a typed four-field record result. The fixed-register helper
  executes natively and preserves RBX. Its `NoAllocation` assertion passes.
- AMD64 `Cli`, `Sti`, `Hlt`, and LFENCE/SFENCE/MFENCE are compiler-known
  operations. Privilege and distinct ordering properties survive in MIR and
  planner evidence. `Privileged<Cli/Sti/Hlt>` proofs pass. Privileged
  instructions are assembled and linked, not run in user mode.
- AArch64 `Yield`, `Wfi`, `Dmb`, `Dsb`, and `Isb` compile for the explicit
  AArch64 target and reject AMD64. Clang cross-assembles the generated helper
  to an AArch64 object. No AArch64 execution is claimed.
- Artifact-only import preserves typed intrinsics, CPUID, asm, and barriers.
  Generic and async-generated-machine consumers generate target helpers.
  MMIO store, `Mfence` call, and MMIO load remain source ordered in MIR.
- DragonGod retains its port UART and uses `Pause` in its bounded transmitter
  poll. MMIO UART remains an address-space operation.
- Both machine and asm artifact tests compare 100 freshly generated output and
  semantic artifact bundles byte for byte.

## Gates

| Gate | Result |
| --- | --- |
| Focused native AMD64 asm and CPUID execution | Passed |
| AArch64 Clang cross-assembly | Passed |
| `go vet ./...` | Passed |
| Root and legacy `zig build test` | Passed on serial runs |
| BurnIn (`go test ./internal/concept -run R7d3 -count=1`) | Passed |
| Standard | 29 passed, 0 failed |
| DragonGod | 23 passed, 0 failed, 1 benchmark |
| EVT1 corpus | 398 valid, 262 static-invalid, 13 runtime-negative fixtures |
| Full `go test ./...` on reconciled checkout | Passed; `internal/concept` 111.198 s |

The first combined full Go run exposed `TestR7eNativeWorkers`'s short-run
thread participation assertion under concurrent suite load. Claude's test
optimization had marked that native concurrency test `t.Parallel()`.
Removing only the outer Go test parallelism made the focused test and the
subsequent full run pass; the native harness still starts 1, 2, and 4 workers.
See `EVT1-R7L2-CONVERGENCE.md`.

The bounded scope still excludes an AArch64 inline asm syntax, general
memory operands, a full assembler, SIMD, naked functions, custom calling
conventions, and a native EVT2 backend. A compiler-only barrier is not added;
the implemented fences are exact CPU instructions. The current target model
does not impose a privilege authority rule.
