# EVT1 R4l conformance

Status: complete General Planner foundation

## Baseline

- starting HEAD: `e94c02abe8c18f4e29fb278a3be3f36193dfdb4a`
- compiler: `concept-evt1-stage0-go`
- Go: `go1.27.0 windows/amd64`
- Zig: `0.16.0`
- MIR schema: `concept-evt1-mir.v1`
- R4j: 24 PASS; R4k: 30 PASS
- pre-R4l Go inventory: 129 top-level tests across the compiler packages
- post-R4l Go inventory: 135 top-level tests across the compiler packages
- baseline worktree: clean

R4l adds 13 required readable valid programs under `language/evt1/planner`.
They cover scalar and aggregate code, owned cleanup, Result `?`, try/except,
Span index/Subspan, fixed layout, zero-storage stream, dyn dispatch, tensor
elementwise/contraction, and runtime-shaped contraction.

`concept plan` emits deterministic `concept-evt1-plan.v1`. Focused tests prove
fixed 4x4 tensor eligibility from R4j facts with no SIMD selection, runtime
shape guards and conservative fallback, exact cleanup order, witness-indirect
dispatch, static layout offsets, erased stream representation, and identical
MIR identity across GenericC11/X86_64_Generic/AArch64_Generic plans.

Malformed internal artifacts reject missing guards, unsupported strategies,
invalid tensor plans/witnesses, incomplete cleanup, absent facts, stale MIR,
and nondeterministic plan IDs. Strict-C11 native execution proves the
planner-backed tensor path preserves numeric behavior.

R4l introduces no source semantics, LIR, MachineIR, native backend, allocator,
SIMD, tiling, loop fusion, LLVM, or MLIR. Recommended R4m scope is a small
target-neutral LIR for the already-selected scalar control-flow and memory
operations, with exact Plan-to-LIR validation and no optimization expansion.
