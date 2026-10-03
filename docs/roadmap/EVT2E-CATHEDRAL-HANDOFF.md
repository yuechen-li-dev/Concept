# EVT2e Cathedral handoff (R9c plan only)

EVT2e implementation is Concept-first. Win64 AMD64 is the first qualification
target. Calls, callee-saved preservation, Win64 outgoing/shadow space,
call-clobber liveness, required spills, runtime helper calls, ABI lowering,
encoding and object-generation preparation belong to Cathedral's ordinary
Concept backend libraries. No EVT2e call instruction is implemented in R9c.

Keep `libraries/Standard/Backend/AMD64.concept` as the present backend owner.
Split into `Backend/AMD64/{ABI,Registers,Liveness,Frames,Calls,Encode}.concept`
only when a real dependency or compilation boundary warrants it. Keep
`BridgeSchema.concept` the sole versioned MachineIR wire authority;
`BridgeDerive.concept` and `BridgeRead.concept` project it. The Go
`machineir.go`, `machineir_lower.go`, `machineir_verify.go` and `lir*.go` may
remain Stage-0 bootstrap input, verified transport and orchestration. They
must not become the default permanent owner of new call allocation, frame,
clobber, spill, or encoder algorithms.

## Contract work before feature work

1. Extend target-independent LIR with an explicit call operation, typed target
   identity, ordered arguments, optional return value, helper-call identity,
   effect/trap information, and scalar/aggregate ABI intent. Keep function
   `return` distinct from a call result. Specify ownership/Drop and evaluation
   ordering before allowing resource-bearing arguments. Verify def/use and
   control-flow effects before MachineIR lowering.
2. Version the sole MachineIR schema after designing CALL operands, explicit
   physical/virtual constraints, per-call outgoing argument and return
   locations, caller-clobber set, live-across-call preservation, stack slots,
   frame alignment/shadow/outgoing area, callee-saved save/restore records,
   unwind/FLAGS kill semantics, and source provenance. Current CMIRAMD2 has
   none of these per-call records; its `HasCalls` and `ShadowSpace` fields are
   insufficient. Do not silently reinterpret CMIRAMD2.
3. Generate Go and Concept codecs from the revised checked Concept schema,
   reject old/new version mismatch explicitly, and keep bridge roundtrip,
   malformed-input, artifact-only identity and 100-run byte stability checks.
4. Extend Concept liveness/allocation/frame/encoder in that order. Current
   `FinalizeFrame` rejects calls; treat that as a named capability stop until
   the new contract and Win64 policy are qualified. Use caller-owned bounded
   workspace and explicit Result exhaustion; do not hide new allocation.

Qualify Win64 integer/pointer register arguments (RCX, RDX, R8, R9), stack
arguments, 32-byte shadow space, stack alignment at each call, RAX scalar
returns, caller-saved clobbers and callee-saved preservation. Large aggregates,
floating-point, variadics and platform-specific unwind details require
separate explicit admission; do not infer support from scalar tests. Runtime
helpers use the same call contract rather than a private opcode shortcut.

For **existing** Go behavior replaced by a Concept decision, use checked
shadow agreement then switch normal authority and mark the retained Go path
RETIRE or BOOTSTRAP-ONLY. For **new** calls with no Go implementation, compare
Concept output with strict-C11 behavior, exact ABI probes, known result
vectors, stable machine bytes and native execution; do not add duplicate Go
call lowering as an oracle. Test negative frames/clobbers and deterministic
diagnostics. Eventually compare Stage-0-built Cathedral and
Cathedral-built Cathedral at an agreed semantic/artifact boundary.

Expected Go changes are limited to versioned bridge generation/transport,
verified bootstrap LIR/MachineIR input if needed, CLI/test orchestration, and
the smallest general language or observation prerequisites that Concept cannot
yet express. A temporary Go escape requires a documented blocker, migration
issue and replacement condition. New permanent Go call lowering, register
allocation, ABI policy, spill selection, frame layout, native encoding and
object emission are forbidden by the R9c default ownership rule.

The future AArch64 backend should consume target-independent LIR and compiler
analysis contracts, while its own machine schema/ABI/encoder remain target
specific. This is a boundary sanity check only; R9c adds no AArch64 feature.
