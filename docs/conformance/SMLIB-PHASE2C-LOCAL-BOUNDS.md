# Phase 2c local checked bounds

The typed semantic-body pass derives operation-local int32 intervals and view
length lower bounds before MIR collection. It recognizes checked subview
construction, dominating length/index guards, successful earlier accesses and
bounded unit-step ranges. Branch joins keep only common facts. Loop writes,
view reassignment, address exposure, captures and unsupported statements are
conservative boundaries. Narrow/unsigned mutable values are not interpreted as
int32 arithmetic. Floating arithmetic is untouched.

The proof states `0 <= index_min <= index_max < length_min` at one operation.
A successful runtime-dependent access establishes length above the LOWER index
endpoint, never the upper endpoint. Refining a condition does not re-execute its
accesses or authorize removing the check that established its own postcondition.
Proof fields are compiler-owned and absent from imported syntax serialization.

Closed MIR instances now receive separate planner entries keyed by semantic
instance identity. Span emission consumes the exact MIR operation decision:
source positions alone are insufficient after static expansion. Normal removes
only selected proven guards; Verify retains every observation guard. Evaluation
order, existing integer checks and first-failure diagnostics are retained.

Tests include imported generic artifacts, deterministic policies, strict C11,
runtime indices, loop-carried mutations, aliased/escaped descriptors, short
workspaces, integer overflow and mixed static/runtime accesses at one source
position. Native negative tests execute the terminal paths. The separate span
reference ABI regression fixes descriptor references lowering as values.

The analysis has no C representation. LIR can reuse these proof obligations
when it supports spans and double; the present Go LIR subset rejects those
types. This milestone does not claim native geometry execution or no-alias facts.
Measurements and frozen-source qualification live in SMlib_Concept/phase2c.

A second bounded pass propagates minimum span lengths into private closed
instances only when every direct call site is accounted for. It starts at
zero and performs at most four pessimistic rounds. Missing/unmodeled call sites
contribute zero; protocol witnesses, captures, erased calls, struct methods and
automata disable this inference. The parameter minima recorded in MIR are entry
facts, not no-alias promises or invariants across descriptor mutation. Exposed
descriptors are excluded from local reuse.

Straight-line integer helper summaries use exact int parameters, checked
arithmetic, local declarations and return expressions. They cannot read globals,
execute loops, summarize foreign code or discard side effects. Runtime calls
are retained. This permits an admitted output size to reach a private helper
without duplicating its checks. Native tests cover a guarded caller and a short
input; adding any unknown caller restores the callee's runtime check.

Integer `+`, `-` and `*` with operation-local proven int32 operand/result
intervals now select direct defined arithmetic in Normal. The proof includes
all intermediate checked operations: possible overflow remains on the original
helper path. Verify keeps checked arithmetic. Source floating-point operations,
contraction, division, shifts and modular narrow/unsigned semantics are untouched.
The planner records IntegerArithmeticPlan decisions, consumed by exact operation
identity. Regression tests exercise bounded signed intervals and preserve
potential-overflow helpers and terminal failure order.
