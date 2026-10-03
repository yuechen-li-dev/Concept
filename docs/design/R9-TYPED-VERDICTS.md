# Typed semantic verdicts (R9b2, R9b3)

> Verdict<Evidence, Refutation> carries typed payloads while projecting into the
> existing Proven / Disproven / Unknown lattice. It does not define a second
> notion of semantic truth.

`Verdict<E,R>` is a compiler semantic type available to declared, project-policy,
and innate predicates at comptime. It has exactly three logical cases:

| Case | Existing proof outcome | Selected metadata |
| --- | --- | --- |
| `Verdict::Proven(e)` | Proven | E evidence |
| `Verdict::Disproven(r)` | Disproven | R refutation |
| `Verdict::Unknown` | Unknown | No payload |

Constructors obtain their closed type from a return, local, or match context.
There is no new general generic-enum language. Unknown has no invented negative
reason. Both payload types must be supported closed comptime values, even when
one branch is not selected. Runtime `Result<T,E>` retains computation success/error
semantics; Verdict is erased from runtime types, C, and runtime template instances.

## Ordinary source

```concept
template <typename T> record struct Box { T value; }
enum Failure { Unsupported(typename subject), Missing }
template <typename T> comptime T Identity(T value) { return value; }
comptime Verdict<Box<uint8>, Failure> Judge(typename subject)
{
    return match
    {
        when compiler.TypeName(subject) == "int" => Verdict::Proven(Identity<Box<uint8>>(Box<uint8>{7})),
        when compiler.TypeName(subject) == "float" => Verdict::Unknown,
        otherwise => Verdict::Disproven(Failure::Unsupported(subject)),
    };
}
comptime string JudgeDescribe(Failure failure)
{
    return match (failure)
    {
        Failure::Unsupported(subject) => "unsupported type " + compiler.TypeName(subject),
        Failure::Missing => "missing evidence",
    };
}
concept Supported<T> { requires Judge(T); }
int Use() { Assert.Concept<Supported>(int, "typed proof"); return 7; }
```

The executable flagship is
`language/evt1/semantic-vocabulary/valid/typed_predicate_protocol.concept`.
Its Disproven and Unknown companions are intentional compile negatives. The
`typed_verdict_values` specimen retains all three as successful comptime values
and exhaustively matches them. An arbitrary record named EvidenceResult still
fails PREDICATE_REQUIREMENT_INVALID; record field spelling never establishes truth.

## One evaluator and one lattice

Declared and innate requirements execute ordinary comptime functions, then share
`evt1ProjectPredicateVerdict`. Its outcome feeds the existing concept proof graph.
The only retained value is the selected payload. A ProofNode's optional `verdict`
metadata records predicate identity, closed E/R type identity, existing outcome,
payload, diagnostic text and source site. It creates no separate proof evaluator.

During compatibility, declared bool predicates and the fixed legacy
`enum Verdict { Holds, Refuted(declaration at, string message) }` normalize through
that same projector. Holds maps to Proven, Refuted to Disproven. Their existing
messages/sites and unrelated proof JSON remain compatible. New rules should use
the typed cases; remaining embedded legacy predicates are not broadly migrated.

A renderer is optional: `<PredicateName>Describe(R)` must be a comptime function
with exactly one R parameter and a string result. It runs in the predicate's same
bounded evaluator state. Without it, the selected value supplies a compact
refutation description. An empty or invalid renderer result cannot establish a
proof. Assertions and lint findings include the rendered reason and compact
payload. Severity remains manifest data. Explain adds compact evidence/refutation
lines; it never dumps the payload AST into human output.

## Closed template functions

> Closed template function calls may execute during comptime when their type/value
> arguments are fully closed and all operations are supported by the bounded
> evaluator.

A comptime call may instantiate an ordinary template through the existing generic
identity, provided its concrete parameters/result are comptime-supported and its
body is available. Explicit `template <typename T> comptime ...` declarations also
support `bounded(N)` recursive helpers. Their bodies/signatures are transported in
semantic artifacts and excluded from runtime MIR/C emission. Ordinary templates
retain their runtime eligibility; unsupported operations still fail in the evaluator.

Open bodies are checked symbolically, then substituted and checked at their closed
call. This is not open execution. Unresolved parameters or a typename-valued source
argument used as an unresolved template type produce
COMPTIME_TEMPLATE_CLOSURE_REQUIRED. Missing transported bodies diagnose
COMPTIME_TEMPLATE_BODY_MISSING; unsupported result/value types diagnose
COMPTIME_TEMPLATE_UNSUPPORTED or CV4216. Runtime/foreign calls still fail CV4210.
CV4201 remains for unsupported expression/statement/operator categories; it is no
longer the blanket rejection for an otherwise supported closed template call.

Bounded comptime recursion may use a provisional closed signature during body
validation, with rollback on failure. Unbounded recursive instantiation fails
CV4217. Execution uses the caller's fuel/depth/loop state, never a fresh budget.
Existing limits remain fuel 4096, call depth 32, loop iterations 256. Exceeding a
recursion bound is CV4206; fuel/depth failures are CV4204/CV4211. Allocation,
foreign IO, machine instructions, and mutable runtime globals have no new evaluator
capability. No source reparsing, cache, or background execution was introduced.

## Payload and artifact boundaries

Supported payloads include resource-free records and closed generic records,
payload enums, bounded scalar/string/array values, and opaque declaration/type
handles. References, pointers, owned resources, Drop-bearing or immovable records,
open parameters and runtime extents remain rejected. Selected values additionally
obey total 512 value nodes, nesting 8, and 4096 string bytes; existing array
construction limits remain active. Overflow is VERDICT_PAYLOAD_LIMIT.

The concept-module.v2 inspectable envelope has optional `predicate_verdicts`,
deduplicated and sorted by deterministic JSON. It records final truth, predicate,
closed type identities and selected payload without serializing a proof tree.
MIRSemanticProof has analogous optional verdict metadata. Semantic payloads carry
ordinary predicate/template bodies and closed type declarations. Consumers evaluate
those bodies with their own subjects; envelope decisions never hydrate semanticFacts.
Stable declaration handles retain owner/name/parent/index/provenance/site, not Go
addresses. Innate identity still hash-covers the embedded rule set, so old artifacts
are stale after the rule migration.

## Innate dogfood and authority

> A predicate proving itself does not grant unrelated optimizer-authorizing facts.
> Trusted structural/codegen facts require explicit trusted provenance.

> Typed refutations are semantic data. Diagnostics may render them directly,
> avoiding duplicate compiler-side reason switches.

DroppableFieldIsOwnedHolds now returns
`Verdict<OwnershipEvidence,OwnershipRefutation>`. Its refutation is
`DroppableFieldMustBeOwned(declaration field)`. A Concept match in
DroppableFieldIsOwnedHoldsDescribe renders the exact former CV4653 text, including
`owned T value;` at the template declaration. The production legacy ownership
predicate is retired; a test-only snapshot checks shadow agreement on reachable
corpus fields and explicit invalid generic/plain cases.

A Proven payload proves its requirement only. It never grants StaticExtent,
NoAllocation, Disjoint or Outlives. The compiler-owned `evt1InnateFactAuthority`
allowlist is empty in R9b2 and requires both actual embedded authority and an
innate declaration. It is an auditable authority marker, not a new fact projector.
A future registered rule still needs a qualified provenance verifier in the
existing fact owner. No typed fact-producing innate rule is claimed here; the
conditional positive fact-projection qualification remains deferred.

At R9b2 closeout, StaticExtent's exact 1D layout-derived facts, Unknown cases and
Planner shadow remained unchanged, while PlainData, FiniteDomain, ClosedWorld,
StableAddress and Relocatable remained research entries. R9b3 supersedes their
admission decisions as described below. See R9B-RESEARCH-LEDGER and
R9B2-CONFORMANCE for the design consequences, measurements and gates.


## R9b3 semantic descriptions and imported requirements

Optional <PredicateName>UnknownDescribe has the same subject parameter types as
the typed predicate and returns comptime string. It executes only for Unknown
in the same evaluator state/fuel budget as the predicate. Signature failures are
VERDICT_UNKNOWN_DESCRIBE_INVALID; empty/invalid descriptions remain diagnosed.
All rendered descriptions are capped at 4096 bytes, including literal returned
strings (VERDICT_DESCRIPTION_LIMIT). Unknown has no selected payload and remains
undecided; explaining missing evidence cannot turn it into Disproven.

PlainDataHoldsUnknownDescribe names opaque missing summaries, partial storage's
live-element protocol, runtime extent, payload-sum geometry and zero-size physical
representation. Nested unsupported structure gets a conservative incomplete-proof
description. The renderer is ordinary Concept code, not a Go reason table.

Ordinary template-body comptime calls now use the existing comptime argument and
call path. Known lexical generic templates are available in predicate-constrained
bodies, while explicit operation requirements keep precedence. Enum and foreign
handle assertion subjects resolve as types while retaining declaration metadata.
Native companion checking/artifact building resolves source-library semantic
imports through the same checked module roots/artifact owner. Qualified topology
is a companion importing ordinary source libraries; arbitrary mixed dependency
cycles/imports are not a new admission.

Vulkan requirement failure diagnostics include the selected typed FieldProblem
or Unknown description via the same projector used by Assert/explain. Artifacts
carry helper bodies and checked types; consumer re-evaluation uses no source
fallback. PlainData evidence is size/alignment, refutation names the type/field,
and no result acquires FactAuthority. Existing legacy normalization and typed
innate ownership projection are unchanged. Research Verdicts never seal an
activation inventory or prove bitwise relocation merely by their E/R spelling.
