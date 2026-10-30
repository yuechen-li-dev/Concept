# EVT2h3 semantic inventory and native seam

Starting state: clean main at `745b1ec35679743bed555a89e19611c6dea4712a`,
equal to freshly fetched origin/main. No intervening commits or unrelated edits.
Compiler: concept-evt1-stage0-go. Windows AMD64 / Microsoft x64. Historical
detached worktrees remain untouched.

The existing parser/checker defines three forms:

* Enum subject matches require every variant exactly once, with declaration
  identity checked nominally. Each payload must bind its full ordered field
  list. Bindings are mutable arm-local values of the checked field types;
  they copy eligible values, rather than introducing reference patterns.
* Integer subject matches use literal cases and a required final wildcard.
  Enum wildcards reject with CV4109. Duplicate/unreachable cases reject with
  CV4113; missing coverage CV4115, wrong binding count CV4111, duplicate/unit
  bindings CV4112, escaping bindings CV4114, incompatible results CV4116.
* Guarded `match { when condition => ...; otherwise => ... }` has no subject.
  The parser expands it to ordered IfStmt/IfExpr chains. Guards must be bool;
  expression form requires otherwise; statement form permits no otherwise.
  Variant patterns with an attached guard are not current source syntax.
  Guards over payloads can appear inside an already selected variant arm.

Expression arms contain one expression (CV4117 rejects statement blocks).
Statement arms require braces (CV4118); normal statements, loops, nested
matches and early returns retain their ordinary semantics. The source/C
backend evaluates the subject once, dispatches by tag, copies ordered payload
bindings inside the selected arm, and joins expression results through a
temporary. Invalid enum tags abort. Source ownership/copy/borrow checks remain
in force; fallible built-ins and owning/move-only subjects are outside this
native milestone.

Before this mission, native LIR rejects MatchExpr/MatchStmt generically and
also lacks IfExpr lowering. CMIRAMD7 supports ordinary enum equality/branches.
CMIRAMD8 admits complete tagged construction and atomic checked payload reads,
whose runtime tag check protects the same private snapshot. It has no typed
variant test, and neither frozen contract carries match/arm/result-slot
correspondence or lexical binding regions.

The bounded extension is CMIRAMD9: reuse frozen v8/v7 records, ordinary CFG,
typed slots, existing borrow scopes and checked payload reads. Add typed
variant-test/require operation sites and a compact source-flow association of
subject/result slots, dispatch/body blocks, arm scopes and binding slots.
Geometry, operands, types and CFG edges remain canonical in existing records.
The require operation rejects an invalid final enum tag without adding a
physical MATCH instruction or changing old terminator enums. Native lowering
uses deterministic compares, branches and the existing object/ABI/frame stack.
CMIRAMD3–8 schemas/codecs and prior admission entrypoints remain frozen.

Admission checks connect each typed test to its subject snapshot,
selected arm and ordered bindings; validate exhaustive mappings and result
joins; enforce binding regions and lexical scope; and reject uninitialized
slot reads through CFG must analysis. Checked payload reads retain their v8
runtime check as well. No guard is speculated or reordered, and no source
syntax, payload layout, allocator, copier or optimizer is introduced.

Arm-local instructions must also be dominated by the selected arm body.
Region reachability alone is insufficient: a forged edge from another arm
could enter a nested block. The retained cross-arm-entry specimen demonstrates
that failure before the dominance correction, independently of payload guards.

This inventory is a design/implementation record, not native qualification.
The conformance/convergence records must separately record executed evidence.

Contract checkpoint: CMIRAMD9 schema SHA256
`77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c`.
Generated Go/Concept codecs reuse canonical v8/v7 records. Independent
artifact-only Concept validation/roundtrip passed 100 runs per specimen in
Normal and Verify (132.110 s package). Fresh source/LIR/CFG/MachineIR/debug and
codec generation passed 100 runs (5.532 s); this is consistency/determinism
evidence, distinct from later machine-code execution.
