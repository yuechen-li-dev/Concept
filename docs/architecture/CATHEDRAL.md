# Cathedral transition (R9c)

Cathedral is the self-hosted Concept compiler and compiler-library stack. It
grows upward from MachineIR by replacing semantic authority piecemeal, not by
rewriting Stage-0 wholesale. The current checkout has Concept-authored AMD64
allocation and encoding libraries, but no complete Cathedral executable or
Concept-owned frontend. “Cathedral” names the destination and the growing
library stack, not a claim that the current `concept` binary is self-hosted.

Stage-0 is the independently bootstrappable Go implementation required to seed
Cathedral. Stage-0 is a durable bootstrap seed, not technical debt scheduled
for deletion. Its long-term job is to make Cathedral buildable from widely
available external toolchains. The Stage-0 -> C -> GCC/Clang/MSVC path remains
intentionally supported even after Cathedral becomes the primary compiler.
`concept-evt1-stage0-go` remains the compiler ID; R9c renames no binary.

New compiler/backend functionality is Concept-first by default. New permanent
Go ownership requires explicit bootstrap justification. If ordinary Concept
cannot express a feature, identify whether the missing piece is language
expressiveness, Standard library, compiler observation, artifact transport,
bootstrap cycle, performance, or an architectural contradiction. Fix the
smallest general prerequisite first. A Stage-0 escape hatch requires a recorded
reason, migration issue and Cathedral replacement condition, and is justified
only if that blocker cannot reasonably be resolved in the milestone and the
feature is needed for bootstrap progress. Stage-0 still evolves for bootstrap
correctness, security, minimum Cathedral compilation support, and contract
compatibility. It is not frozen by R9c.

## Paths and authority

```text
bootstrap: Concept source -> Stage-0 Go -> C -> GCC/Clang/MSVC
target:    Concept source -> Cathedral frontend/semantic libraries -> Planner
           -> LIR -> MachineIR -> native backend libraries -> object/executable
```

The target path is a roadmap, not a currently runnable pipeline. The current
Concept-authored slice consumes Stage-0-produced MachineIR through CMIRAMD2.
Its present conceptual entrypoint is the composition of
`Standard.Backend.Bridge{Schema,Derive,Read}` with
`Standard.Backend.AMD64`, invoked by the Stage-0 `concept amd64` host. There is
no Cathedral executable or `Standard.Compiler` umbrella module yet.
The C backend is a portable bootstrap and potentially long-lived fallback.
Native Cathedral output is the intended normal path once qualified. Object
emission is future Cathedral ownership. Neither a native object writer nor an
AArch64 backend is implemented by R9c.

Self-hosting is measured by semantic authority, not by which language happens
to contain the most compiler source lines. A bootstrap copy may remain if it
implements a frozen checked contract and does not independently evolve normal
semantics. Existing Go semantic authority migrates by shadow, agreement,
switch, and retirement or bootstrap-only retention:

1. Define an explicit contract and freeze a behavior corpus, including errors.
2. Run Concept and existing Go paths in shadow on the same inputs; compare
   semantic/artifact/native results appropriate to the seam.
3. Switch the real normal-path decision only after agreement and oracle checks.
4. RETIRE duplicate Go logic if unnecessary; mark it BOOTSTRAP-ONLY if the seed
   still needs it; or mark it SHARED if responsibilities remain distinct.

For brand-new features, use Concept against an independent C/native/ABI/vector
oracle. Do not invent a duplicate Go implementation to manufacture parity.
Bootstrap reproducibility is a third mode: Stage-0-built Cathedral eventually
rebuilds Cathedral and agrees at a chosen semantic, artifact, MachineIR, or
object boundary. Byte-identical executables are not a current promise.

The R9c process proof switches the production Go MachineIR condition verifier
to `bridgeTagsCondition`, generated from Concept's checked
`Standard.Backend.BridgeSchema.Condition`. The former hand-maintained Go switch
is retired. The generated Go table stays BOOTSTRAP-ONLY as contract transport;
the Concept enum owns the allowed names. The frozen pre-switch truth table,
including `None` and invalid names, is compared in
`TestR9cConditionAuthorityShadowAgreement`. This is a narrow vocabulary
authority switch, not a claim that the Go verifier as a whole has migrated.
The direct `concept amd64` Stage-0 host now builds the backend's ordinary
Concept imports from the checkout library root before C generation. This
repairs a previously failing CLI bootstrap route exposed by the audit; it
adds no backend semantic rule.

## Growth boundary and trust

MachineIR is the strongest current Cathedral growth boundary: checked
CMIRAMD2 version/hash, generated Go/Concept codecs from one Concept schema,
virtual registers, flags, frames, slots, arguments, source provenance, and
native byte oracles. `Standard.Backend.AMD64` already owns bounded block
layout, liveness, interval allocation, frame finalization for no-call frames,
encoding, and caller-owned encoding workspace. `BridgeRead` and `BridgeDerive`
own Concept-side reading/derivation. `Standard.Machine.AMD64` contains typed
hardware wrappers and is not the compiler backend. The R9c condition helper
uses this seam without changing its bytes, hash, or wire version.

CMIRAMD2 is stable for the present no-call corpus, not sufficient as-is for
EVT2e. `BridgeSchema.Opcode` has no CALL; `WireMachineFrame` has only local
size, alignment, shadow space and HasCalls; the function has incoming arguments
but no per-call outgoing argument/return locations, call clobbers, callee-save
set, spill slots or frame-unwind contract. FLAGS uses are local to current
blocks, with no call-kill edge. Concept `FinalizeFrame` explicitly rejects
HasCalls or nonzero shadow space. Any additions must be versioned in the sole
schema, generated into both codecs, and verified before native execution.
LIR likewise has no call instruction, helper-call target, outgoing arguments,
return projection, or aggregate/scalar call ABI record. Its existing `return`
terminator covers function results, not call results. See the EVT2e handoff.

The current bootstrap trusted computing base comprises Stage-0 lexing/parsing,
syntax/declaration/type identity and name resolution, generic closure and
comptime evaluation, artifact loading, semantic observation projections, MIR
production, the Planner and MIR-to-LIR bridge, LIR/MachineIR verification and
MachineIR production, plus the C transport compiler and external C toolchain.
Some language restrictions already live in ordinary innate Concept concepts;
that reduces rule ownership but does not eliminate trust in observation or
proof/evaluation mechanisms. Track this conceptual TCB by contract and
decision owner, not source line count. Each migration must say which trusted
decision it removes, or explicitly say that it only relocates code.

Compiler mechanisms (syntax/type representation, declaration graph, CFG and
ownership dataflow, proof and comptime engines, artifact IO) can remain in
Stage-0 longer. Language rules over checked observations, declaration validity,
policy and layout/resource conditions should migrate gradually to ordinary
Concept. There is no wholesale parser, resolver, or typechecker rewrite.

`compiler.*` observations are a compatibility contract. Structural declaration
identity, ordered fields/cases, type identity, signedness/geometry and checked
ownership/effects have active Concept consumers and must preserve bounds,
provenance and Unknown behavior. `Name`/`TypeName` are display projections,
not stable identity keys. String-shaped `TemplateName` and raw layout/detail
projections are Stage-0-shaped conveniences; do not build new authority on
their spelling. Existing fact-granting observations remain trusted mechanisms,
not user-authored proofs. Do not expose new Go data structures as observations
without a contract. R9c retires no observation; the empty trusted-fact registry
remains empty.

## `validate.go` strangler inventory

This is a rule-family audit of the present monolithic validator, not a plan to
move the file by line count. `innate/Innate.concept`, Standard predicates and
the R9a/R9b agreement tests supply existing rule precedents.

| Family and current Go seam | Classification | Next useful boundary |
| --- | --- | --- |
| Droppable field ownership and movable-field embedding | Already innate | Keep exact diagnostics and typed Verdict agreement; do not re-add Go decision |
| Declaration naming, authored/generated markers and project naming policy | Ready for innate/policy | Existing declaration observations and lint proof graph; migrate only an active rule |
| Closed repr(C), C ABI legality and foreign signature restrictions | Requires new compiler observation for full migration | Checked ABI/layout descriptor and target-specific uncertainty before a Concept rule owns legality |
| Array extent, scalar width and fixed geometry restrictions | Partly already innate; partly ready | Existing bounded geometry observations, preserve overflow/Unknown |
| Borrow invalidation, ownership joins, move/drop and result provenance | Mechanism-level stay Go for now | Publish checked dataflow summaries before moving individual restrictions |
| Name resolution, overload ranking, type canonicalization and generic substitution | Mechanism-level stay Go for now | Stable graph/type contract, not predicate emulation of lookup |
| Comptime body checking, fuel, closure and selected branches | Mechanism-level stay Go for now | Bounded evaluator contract and transported selected body |
| Reflection-driven generated declaration validity | Ready for Concept-authored rule; materialization stays Go | Checked ordered reflection and generated-input identity |
| Legacy duplicate field rule CV4138 | Obsolete/duplicate | Already removed in R9a after innate agreement; do not recreate |

The audit does not claim every nested branch in the large validator is
individually classified. A future migration names the exact diagnostic and
decision site, freezes a valid/invalid corpus, and updates the ledger when
normal authority switches. `compiler.*` projections should likewise be
reviewed by stability: typed identity/ordered structure are contract inputs;
display names are diagnostics; internal string encodings should be retired as
decision keys when structural alternatives exist.

Cathedral should be built primarily from ordinary Concept libraries so the
same compiler infrastructure can eventually be reused by other language
frontends. A future language frontend targeting Concept should be able to
reuse Cathedral's semantic lowering, Planner, MachineIR and native backend
libraries without reimplementing the full compiler stack. This is a library
design constraint, not an SDK built by R9c. Keep current paths until a move
solves an actual boundary problem. A likely later shape is
`Standard.Compiler.{IR,Planning,Analysis,Semantics}` and
`Standard.Backend.{AMD64,AArch64}`; no namespace churn is required now.
Cathedral code uses ordinary Concept, Standard systems primitives and narrowly
audited semantic-observation, bootstrap-host, or unsafe/native APIs. No secret
compiler dialect is introduced. Use `Assert.Concept`, `concept explain`,
`concept generated` and semantic artifacts where they improve debugging.

Cathedral should be one of the strongest consumers of Concept's semantic
propositions, evidence model, reflection, generated declarations, memory model,
and compiler-authoring libraries. A failure to express compiler logic is
language feedback, not permission for untracked permanent Go ownership.

## Staged self-hosting and development ratchet

| Level | Evidence needed | Present status |
| --- | --- | --- |
| C0 | Stage-0 builds compiler libraries | Reached for the current AMD64/bridge libraries through Stage-0 C output |
| C1 | Concept backend owns active native lowering | Partial: allocation/encoding yes; call/ABI path absent |
| C2 | Concept owns Planner and LIR transforms | Future |
| C3 | Concept owns most semantic rule decisions | Future; bounded innate rules already migrated |
| C4 | Cathedral compiles substantial Cathedral source natively | Future |
| C5 | Cathedral rebuilds itself through native output | Future |
| C6 | Stage-0 is only the durable seed | Future |

Self-hosting is a gradient of semantic ownership, not a single flag day.
Track per-subsystem state in the authority ledger. Reviewers of compiler work
ask: why Go, could this be Concept, what general capability is missing if not,
and is the proposed Go work bootstrap transport or semantic authority? Measure
correctness, determinism, and owner first; then measure and profile performance.
A slower Concept path is a dogfood issue to quantify, not an automatic reason
to move authority back to Go. `Bazaar` is reserved for a future package and
dependency distribution tool; Cathedral is the compiler. The familiar names
make a brief joke, not an R9c package-manager project.
