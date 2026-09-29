# EVT2 target-independent LIR

EVT2x is a sequence break after EVT2d. Machine-state semantic bodies and
persistent initializers survive in-memory MIR construction. EVT2x2 lowers the
single-machine finite-state subset into verified caller-owned frame Init/Step
LIR; pushdown and MachineIR lowering remain explicit boundaries. See
`EVT2-NATIVE-AUTOMATA.md` and `EVT2-MACHINE-FRAME.md`.

MIR preserves Concept language semantics. SemanticFacts record what the compiler knows. Planner decides which mechanisms remain necessary. LIR makes low-level control/dataflow explicit without becoming target-specific. MachineIR will later perform target/ABI/instruction lowering.

Lowering may erase syntax but not semantic authority required by later stages. A retained bounds guard is explicit; an exact conversion, atomic order, MMIO distinction, or proof-backed removal must likewise have an explicit representation before that feature is supported by LIR.

## Existing authority and EVT2 entry

The existing `concept-evt1-plan.v1` General Planner is the LIR planner layer described in `EVT1-LIR-PLANNER-DIRECTION.md`. EVT2 calls `PlanModule` and `ValidateLoweringPlan` rather than building a second planner. The pipeline is validated module -> MIR -> qualified SemanticFacts -> General Planner -> `LowerMirToLir` -> verifier -> text. `concept lir <file>` exposes this path. It does not run the C emitter or produce native code.

The existing MIR JSON contains semantic operation summaries, not expression operands or CFG. EVT2 retains the already validated, typed function body in `MIRFunction.SemanticBody` in memory. That field is excluded from the legacy MIR JSON so checked artifacts and the C11 backend retain their contract. LIR lowering consumes this MIR-owned body plus the validated plan. Unsupported body nodes fail with an `EVT2_UNSUPPORTED_*` error. A future round-trippable MIR would need explicit operands and control flow; this milestone does not claim that the current MIR JSON is such an IR.

## Constitution

Functions have semantic declaration identities including parameter signatures, ordered abstract parameters, a result type, source location, ordered blocks, fact IDs, and Planner decisions. `i8/u8/i16/u16/i32/u32/i64/u64`, `bool`, `void`, fixed arrays, and typed internal addresses are independent of Go types. One instruction defines at most one virtual value. Parameters are available in every block; computed values are local to the defining block. Mutable source locals use explicit slots. Array parameters bind a fixed-extent slot to an abstract parameter value. There are no phi nodes or block parameters in EVT2a+b.

Every block has one explicit `return`, `jump`, or `branch` terminator. Integer arithmetic uses `checked_add/sub/mul`; a later proof-backed Planner decision may authorize removing a check. Comparisons produce `bool`. Counted `start..end` loops use a cursor slot, `<` exit test, body, checked increment, and back edge, so the end is excluded. Indexing uses `check_index`, `index_address` with extent/element stride, and typed `load`/`store`. The bound guard carries its existing Planner decision ID and referenced fact IDs. EVT2 accepts only `PerAccessRuntime` bounds plans; other strategies fail explicitly. Slots, instructions, and functions can carry semantic fact references. Instruction and terminator source spans support diagnostics but are not identities.

The verifier checks unique value and slot definitions, block ordering and terminators, block-local use before definition, operand/result types, targets, return types, slot types, fixed-array extent/stride/address coherence, and a matching preceding index guard. There is no optimizer, interpreter, LIR parser, machine register, ABI location, target instruction, or serialization in this milestone. Future float, aggregates, Drop, Span, atomics, MMIO, async, tensor, and vector work must extend the semantic operations and verifier before lowering them.

EVT2c adds a separate AMD64 MachineIR after the LIR verifier. LIR remains target-independent: its bool comparisons, checked arithmetic, explicit guards, slots, and CFG are the semantic authority. `LowerLirToAmd64Machine` selects target flags, register widths, Win64 argument/return locations, memory addressing, and explicit failure blocks without changing the LIR printer or the C backend. See `EVT2-MACHINEIR-AMD64.md`. The later EVT2s glyph experiment may add alternate renderers for the **same** internal LIR or MachineIR representations.
