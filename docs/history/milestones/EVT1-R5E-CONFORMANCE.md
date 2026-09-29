# EVT1 R5e conformance

Baseline: `3f79e4962a13a421dd5151caef2702099cadbfcd`
(`concept-evt1-stage0-go`).

Inventory classification: **B, partial substrate exists**. Legacy signal
automata has bounded push/pop continuation storage; DragonGod DG5 has a fixed
four-frame StateId/Reason stack; canonical R5a-R5d had explicit state and
persistent storage but no frame stack. R5e completes the canonical substrate.

The canonical corpus has twelve valid sources, five static-invalid sources,
and two runtime-negative sources. R5a/R5b/R5c/R5d retain respectively 20, 26,
30, and 26 conformance cases.

PoC3 Phase 13/18 supplies Step, completion/result storage, nested machine
pressure, and the stable pre-completion panic. R5e replaces manual child
ticking with top-frame dispatch and distinguishes Neutral, Success, Failure.

DragonGod supplies LIFO, fixed capacity, overflow/underflow, and restoration
pressure. Its StateId/Reason frames remain library policy rather than compiler
machine storage. Current Oct Remember/Resume is one overwriting state-target
slot, cleared on successful resume with an empty-slot runtime error. It is
policy pressure, not another compiler frame stack.

Generated GenericC11 uses fixed inline storage. Tests reject heap, scheduler,
setjmp, and program-counter markers and execute native C for nested outcomes,
re-entry, shared/private fields, recursion, overflow, and result gating.
