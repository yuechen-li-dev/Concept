# EVT1 suspension and scoped authority

Scoped authority crossing suspension must remain semantically live in generated machine state or be rejected. Suspension never silently drops authority.

R7f2 rejects a local reference struct that carries a reference to an invalidatable resource when it is live across `await`. It rejects a live lexical lease across machine `yield`, and excludes the invalidatable reference carrier from persistent automata and machine fields. The async persistence proof reports `SCOPED_AUTHORITY_CROSSES_AWAIT`; yield and field checks use directed `SCOPED_AUTHORITY_*` diagnostics. An awaited operation declared `InvalidatesBorrows` is checked before suspension.

This preserves ordinary async values, parameter-backed reference structs, and persistent spans that have no declared invalidation authority. It does not prove persistent leases, parent/child machine coexistence, or cancellation Drop. Those require a precise machine-lifetime and effect-summary design. A future collector can expose `Borrow(handle)` and destructive `Collect`/`Reclaim` without runtime borrow tracking only after those remaining paths are proved or rejected.
