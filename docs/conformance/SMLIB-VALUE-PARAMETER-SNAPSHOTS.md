# Private aggregate value-parameter materialization

Concept's typed MIR already distinguishes a by-value parameter from a reference.
On Win64, Clang may represent a C aggregate value parameter using an implicit
pointer. Loads from that descriptor can remain after stores through a view,
because the optimizer does not establish the independence of the ABI storage.
The qualified NURBS comparison against handwritten C exposes these reloads.

The portable C11 lowerer now binds small transparent aggregate values into a
local at private function entry. The incoming signature has the same C types,
calling convention and symbol; only parameter names differ. The original source
binding refers to the local. This exposes value semantics to scalar replacement
without a restrict/noalias promise and without changing typed MIR, proofs,
checked arithmetic, span guards, failure positions or floating-point operations.
Normal and Verify both retain their existing observation contracts.

Selection is deliberately bounded: private synchronous ordinary/closed functions,
2–16 transparent scalar descriptor leaves, recursively at most sixteen levels.
Spans copy only pointer/length fields, preserving backing aliasing and lifetime.
Ref structs containing spans remain value descriptors; explicit ref/borrow/owned
parameters and fields are excluded. Arrays/tensors, payload enums, handles,
opaque/volatile/atomic types, methods, classes, immovable/table storage, drop
requirements and unfamiliar attributes are excluded. No large-array copying or
ownership duplication is introduced. Public/foreign signatures and async entry
paths are unchanged. The bound limits exposure/copy cost, not mathematical legality.

The authority is the existing by-value parameter contract. Input and result views
may still overlap. Native positive regressions deliberately alias backing data
and verify that stores remain visible through the other view; reference mutation
must remain visible through ref descriptors. Tests also cover parameter mutation,
generated-name collisions, unsupported representation boundaries, imported
generic artifacts, deterministic Normal/Verify output and strict C11 execution.

This is a backend representation refinement, not a C-shaped semantic MIR pass.
Future LIR/native lowering should bind by-value aggregate fields to entry values
and materialize an addressable local only where required. It must retain actual
reference semantics and cannot infer pointee independence from this operation.
There is no need for a utility engine to decide semantic correctness. A future
cost model can compare legal representations, but may never override legality.

Consumer experiments/evidence live in SMlib_Concept/phase2c/emission-control.
