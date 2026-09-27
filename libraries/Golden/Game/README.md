# Agent simulation

`Agents.concept` holds agents in a fixed dense store, updates intent through
a hot ID traversal, and defines an explicit resumable patrol machine.
`WorldOps.concept` adds generational transient trails, a fixed SoA position
table, a Span reduction, and an application-owned tick budget.
The tests check motion, combat, bounded storage, stale trail handles,
projection, and yield.
The first draft is preserved in `first-draft.concept.txt`.

Friction: indexed C++ loop became a range loop; store mutation needs an
explicit invalidation contract. Familiar: agent state and per-tick update.
New: `Id<T>`, `GenerationalId<T>`, table/Span views, and machine state.
Advanced: borrow invalidation and proof. The imported closed
`GenerationalStore.Remove` revealed a missing `Result<void, StoreError>`
constructor in generated C; artifact-only codegen now emits it.
