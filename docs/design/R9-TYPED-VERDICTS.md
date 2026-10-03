# R9 typed verdicts: evidence values before predicate admission

> Verdict<Evidence, Refutation> enriches Concept's existing Proven /
> Disproven / Unknown lattice with typed explanations. It does not create a
> second proof system.

The intended tagged result is Proven(E), Disproven(R), Unknown(reason), with a
checked mapping into the existing lattice. An evidence value must not become an
optimizer fact merely because it uses these case names.

## Removed blocker

Closed generic records previously failed `CV4216: comptime return type
EvidenceResult<Box<uint8>, Failure> is not supported`. Comptime type admission
now closes applications through the existing generic identity and checks their
fields recursively. Record construction resolves the same structured application
and evaluates each payload with its declared field type. This also preserves
contextual small-integer representation inside nested typed evidence.

References, pointers, ownership, destruction authority, immovability, runtime
extents and open concept parameters remain outside this value boundary. A
visiting set prevents recursive aggregate admission. Existing evaluator fuel,
call depth, loop and array limits remain unchanged. No display-name reparsing or
runtime template-call admission is involved.

The executable value prototype uses ordinary Concept types:

```concept
enum Truth { Proven, Disproven, Unknown }
enum Failure { None, ContainsReference, UnknownLayout }
template <typename E, typename R> record struct EvidenceResult
{
    Truth truth;
    E evidence;
    R refutation;
}
comptime EvidenceResult<uint8, Failure> Inspect()
{
    return EvidenceResult<uint8, Failure>{Truth::Disproven, 7, Failure::ContainsReference};
}
```

This is a payload/identity prototype, **not an admitted verdict protocol**. Both
record payload fields are populated; only a future explicit protocol can choose
which payload is meaningful. Do not interpret a record's `truth` field as proof.

## Isolated next blocker

`language/evt1/semantic-vocabulary/invalid/typed_predicate_protocol.concept` passes
the new closed-value boundary, then fails predicate admission:

```text
PREDICATE_REQUIREMENT_INVALID: predicate Judge must return Verdict
(or bool in a declared concept), not EvidenceResult<int, Failure>
```

`evt1ValidatePredicateRequirement` still requires the fixed legacy Verdict enum
shape. `evt1EvaluateDeclaredPredicate` and `evt1EvaluateInnateConcept` decode its
Holds/Refuted cases. The latter is two-valued for valid innate results; an
undecidable innate requirement remains a compiler defect rather than acceptance.
Ordinary proof evaluation still preserves Unknown. Tests pin both the successful
value boundary and the predicate rejection; no manufactured Proven result hides
the remaining work.

Next qualification must define one closed typed result identity, validate its
case/payload shape, share lattice projection across declared and innate paths,
and retain E/R payload provenance in explain/artifacts. Closed template *function*
calls still stop at CV4201; constructor support has not enabled them. Their
admission must retain ordinary closure identity, comptime-only type checks and
bounded evaluation rather than routing through runtime template evaluation.

## Transport and cost

Closed record values and generic declaration/argument identity survive semantic
artifacts through existing Value and GenericApplication transport. Artifact-only
imports evaluate Inspect without dependency source reparse and run a strict-C11
consumer. A 100-run artifact byte check covers the producer. This does **not**
transport typed proof/refutation results: none have yet been admitted as proofs.

One informational Windows run: 100 ordinary-record parses 30.40ms versus 100
closed-generic parses 51.32ms. The nested Inspect evaluation used fuel 5, depth 1,
no loops and no arrays. These small-fixture numbers are not a large-library
scaling promise or mutex benchmark. Innate rules and their workload are unchanged;
the process-wide evaluator mutex is retained. No memoization was added.
