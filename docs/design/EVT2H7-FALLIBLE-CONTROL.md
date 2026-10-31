# EVT2h7 fallible control-flow inventory

Preimplementation inventory, 2026-10-31. Actual clean HEAD is
`e7d4f0ee57ea7cc8ba3e6f962a8587ab2d9d9489` on the existing
`codex/implement-native-ordinary-enums` worktree. Fetched origin/main is
`065317d`, an ancestor nine commits behind. No newer implementation or unrelated
work is discarded. Compiler concept-evt1-stage0-go; Windows AMD64 / Microsoft x64.

## Existing source semantics

Inspected the six requested valid and four requested invalid EVT1 failure
fixtures, `parse.go` parseTryStmt/postfix expressions, `types.go` TryStmt and
FailureExpr, `validate.go` validateBlock/FailureExpr, `generate.go` MIR and C
failure/try lowering, h4/h5/h6 LIR/native tests and current authority records.

* `?` and `!` accept existing Option<T> and Result<T,E>, not arbitrary user sums
  (CV4541 / CV4544). They evaluate the operand once, then yield T on success.
  Result<void,E> success has no payload/value to fabricate.
* `!` escalates None or Error with concept_panic (the C bootstrap's abort path),
  with distinct Option absence / Result error messages and source site. It does
  not propagate, return an error or select an except handler. Native realization
  must use the existing terminal UD2 facility and qualify its Windows status.
* Option `?` always requires an Option-returning function (CV4542), including
  inside try. None returns the containing function's None. Local typed handlers
  handle Result errors, not Option absence.
* Plain Result `?` chooses the nearest lexically enclosing handler whose
  **canonical nominal E** matches exactly. Inner unmatched errors may select a
  matching outer handler. Equal physical representations do not imply equality.
  Without a matching handler, a Result-returning function with exact E receives
  the error; otherwise CV4545 (try has no matching arm), CV4542 (no channel), or
  CV4543 (wrong function E) applies. Operand T and containing function T may differ.
* Existing `? else expression` requires a Result operand and Result-returning
  function. It bypasses local handlers, evaluates the checked remap only on
  failure and returns the containing function's Error. It does not introduce an
  implicit binding for the old error. Preserve this precedence.
* try requires at least one except. Duplicate canonical error types reject with
  CV4547; unknown handler types reject with CV4550. Different error types have
  no dynamic priority: the checker already determines the exact destination.
* Handler bindings are immutable, initialized and scoped to their handler.
  Handlers see the enclosing lexical environment, not locals declared in the
  try body. The current try's handlers are not installed while checking or
  executing its handler bodies; failure there may reach an outer handler or
  the function boundary. Nested try reinstalls its own lexical set normally.
* Success skips every handler. A failure skips the remainder of its selected
  try body; normal handler completion resumes after that try. Ordinary returns,
  calls, loops and matches retain their existing semantics. No exception runtime
  or dynamic type registry participates. No new break/continue syntax is implied.
* Existing move/copy/Drop/lifetime checking applies (including CV4548 for a
  non-copyable carrier without a legal transfer). C failure edges perform their
  checked cleanup. Native h7 must reject paths needing unavailable cleanup;
  copyable, non-resource-bearing values are the admitted domain.

## Native boundary and smallest justified extension

H5 lowers Result function-level `?` through an exhaustive two-arm pattern flow;
h4 Option `?`/`!`, Result `!` and local TryStmt currently reject. Earlier operands
are saved in ordinary logical slots when a later Result `?` introduces blocks.
The same sequencing predicate must recognize the newly supported Option and
bang control flow for both checked LIR and the C oracle.

CMIRAMD9 already preserves canonical tagged types, active fields, readonly
scrutinee snapshots, lexical binding scopes, joins and operations. Its admission
requires **all variants** in every pattern flow. Therefore a success-only
require-variant guard cannot be admitted. Error-arm scalar/object terminators
have no general trap form, and using fake overflow or a dummy return would be a
brittle substitute. Its binding-region traversal also stops only at the one
pattern join, with no checked handler destinations/nonlocal exits.

Introduce **CMIRAMD10** as a Concept-owned, generated extension over the unchanged
WirePatternModule. Preserve CMIRAMD3–9 schemas, codecs, hashes and bytes. Retain
the established eight-byte magic/44-byte header: new wire magic `CMIRAMDA`
(hexadecimal A = version ten), numeric version 10. No old magic changes.

New checked records should carry only missing control authority: qualified
success-guard flow identities, lexical try/handler identities, and propagation
routes. Types, operands, variants, field layouts, scopes, source sites and CFG
edges remain in canonical existing records. Concept admission must independently
check exact type routing, nearest eligible scope, binding initialization, handler
entry predecessors, authorized region exits and source-operation correspondence.
Unqualified v9 flows keep their existing exhaustive/region rules.

Native physical operations remain ordinary branches, require_variant (UD2 on
failure), checked payload binding, logical copies, jumps/calls and returns.
Do not introduce a physical TRY/EXCEPT/BANG opcode, a trap-by-overflow trick,
another union/copier, register allocator or frame planner. Go owns checked
semantic production and generated transport; Concept owns admission and native
realization. Runtime error-type dispatch and heap/unwind machinery are unnecessary.

## Verification plan

Use one authoritative Concept fixture plus grouped .concept_test facts; external
Go/C hosts own IR mutation, source diagnostics, exact bytes, execution and
independent C11 comparisons. Qualify Option/Result bang success, void success,
scalar/floating/record/array values, exact floating bits, nearest/nested handlers,
handler re-propagation, remap precedence, loops/matches/early returns, ordering,
live values, calls and spills. Pin source-valid unsupported ownership/reference
boundaries and retain old source-invalid fixtures.

Expected native failures use isolated children of distinct completed executable
paths. Require UD2 inside the emitted image and Windows illegal-instruction
status, rather than accepting any crash. Failed artifact admission must preserve
roundtrip/code/image sentinels. Use ten fresh h7 generation/plan/emission samples,
retaining 100 via the phase marker and all historical gates unchanged.

Run Quick throughout; focused h7 and adjacent h3–h6/ABI/native regressions, C11,
Normal/Verify, race/vet, generation/frozen-format checks and lint/diff. The new
control-flow contract/admission helper reuse triggers expanded affected native
regressions. A full project-wide lane is warranted only if that impact cannot
be bounded by the selected invariants. Stop after h7, not h8/h9 or ownership.
