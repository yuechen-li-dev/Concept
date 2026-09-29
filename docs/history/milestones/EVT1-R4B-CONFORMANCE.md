# EVT1 R4b conformance record

Status: relational lifetime proofs and selected call-result provenance

## Baseline

```text
HEAD: 858da44a4d4c35c2e672fbf931eec1dd1b95ec69
compiler: concept-evt1-stage0-go
R4a: 11/11 PASS (6 valid, 5 invalid)
Go tests: 65 top-level test functions
Go: go1.27.0 windows/amd64
Zig: 0.16.0
worktree: clean
```

The R4a registry contained `LifetimeSafe` and `NonEscaping` as executable
unary type analyses and a type-ranked, internally registered `Outlives` entry
that had no relational source proof. Lexical provenance categories were
`Local`, `Parameter`, `Static`, and conservative `Unknown`, with scope depth
and scoped state.

## R4b corpus

```text
11 total
11 PASS
6 valid
5 invalid
0 EXPECTED-DIVERGENCE
0 EVT1-GAP
0 POC3-LEGACY
0 SPEC-AMBIGUITY
```

The readable corpus is under `language/evt1/lifetime/relational`. Each harness entry
records `R4a foundation plus EVT1-new relational semantics` or `EVT1-new
relational semantics`; R4b does not fabricate PoC3 equivalence.

Valid evidence covers direct parameter-derived reference results, ref-struct
results, a concept-requested `Outlives` proof, non-escaping scoped use, nested
ref-struct provenance, and `ShortestOfParameters`. Invalid evidence covers a
short source hidden behind a pass-through helper, a disproven relation, unknown
declaration-only result provenance, scoped result escape, and rebinding a
longer-lived place from a shorter-derived local.

## Executable model

Compiler-analysis requirements may name semantic subjects while source
concepts remain single-type-parameter. The R4b syntax is:

```concept
concept ViewOf<T>
{
    requires T MakeView(ref const int source);
    requires compiler.Outlives(source, result);
}
```

The subjects bind to a parameter and result of exactly one selected required
operation. `Outlives` returns `proven`, `disproven`, or `unknown`; only proven
satisfies the concept. The analysis runs only when concept satisfaction is
requested by an assertion or constrained-template instantiation.

Functions returning references/ref structs expose `static`, `parameter`,
`shortest_of_parameters`, or `unknown` summaries in MIR. The inference pass is
straight-line and syntax-directed. It recognizes direct returns,
reference/ref-struct and nested construction, and acyclic direct pass-through
calls. It does not solve branches, recursion, callbacks, or arbitrary aliases.

MIR semantic proofs record a deterministic ID, analysis, concrete type,
parameter/result subjects, proven/disproven/unknown outcome, provenance facts,
and the requirement-origin span. These records are compile-time metadata and
have no C payload.

## Diagnostics

R4b adds semantic families for disproven and unknown `Outlives`, unknown call
results, call-result source escape, scoped-result escape, invalid relational
subjects, and unsupported relational shapes. The conformance negatives pin the
first six applicable paths; `RELATIONAL_REQUIREMENT_UNSUPPORTED` remains a
reserved bounded-model category.

## Native evidence

Strict C11 harnesses execute the direct parameter-derived reference result,
nested ref-struct result, and satisfied relational concept cases. Invalid
lifetime programs are rejected before C generation.

## Boundary

R4b adds no global borrow checker, named or non-lexical lifetimes, generalized
alias/region inference, Span, ReadOnlySpan, Slice, stackalloc, dyn, interface
syntax, arbitrary concept arity, or allocator/storage model.

## Known gaps and recommended R4c

Result inference deliberately yields `Unknown` for declaration-only functions,
recursive cycles, ambiguous overload sets in the summary pass, and
control-flow-dependent returns even when a broader solver might prove them.
Relational source syntax is limited to parameter/result subjects from one
required operation; there is no free-standing local proof assertion.

Recommended R4c scope is a bounded result-provenance contract for bodyless or
separately compiled functions, verified against bodies when both are present,
plus equal-summary branch joins and overload-aware pass-through selection.
That closes practical summary gaps without named lifetimes, NLL, or global
region inference. Span and dyn should remain deferred until those contracts
have a second concrete consumer and deterministic conformance evidence.
