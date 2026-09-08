# EVT1 R5b conformance: deterministic transition matching and decision

Baseline: `481b5b54ee369a48b69c18f927c31b0bd2476d70`

Compiler: `concept-evt1-stage0-go`

## Result

R5b contains 26 canonical `PASS` cases: 14 accepted positive sources, ten
statically rejected sources, and two accepted runtime-negative sources. The
positive set includes one combined evaluation-count specimen beyond the 13
named feature specimens. No R5b case is an expected divergence, EVT1 gap,
PoC3-only legacy case, or unresolved specification ambiguity.

| Classification | Count |
|---|---:|
| PASS | 26 |
| EXPECTED-DIVERGENCE | 0 |
| EVT1-GAP | 0 |
| POC3-LEGACY | 0 |
| SPEC-AMBIGUITY | 0 |

R5a remains 20 `PASS` cases: 12 accepted and eight statically rejected.

## Authority and translation

PoC3 Phase 18 is executable evidence for runtime bool `transition match`,
guard-before-score evaluation, unconditional fallback, highest integer score,
source-order ties, repeated candidate handling, and the stable no-enabled
panic. EVT1 translates that behavior into the canonical outer `automata` /
contained `machine` hierarchy and broadens match through the already-canonical
ordinary payload-enum, Option, and Result checker. It adds exact `float` scores
with an explicit NaN panic. Syntax differs; matching behavior after translation
is `PASS`.

The pre-R5a Go typed-signal automata implementation also supplies guarded
declaration-order transition pressure. It remains a compatibility path and is
not reused as a scheduler or decision runtime. DragonGod Phase 20 stateful
Decision behavior is explicitly not a parity target.

## Semantic and native evidence

The corpus under `language/evt1-r5b/core` proves:

- exhaustive enum, payload-enum, and Result transition matching;
- duplicate arm, missing arm, invalid pattern, and unknown local target checks;
- integer and float hardmax with exact uniform score types;
- bool guards, unguarded fallback, duplicate targets, and current-state targets;
- declaration-order first-maximum ties;
- shared automata and machine-private score inputs;
- exactly one scrutinee evaluation, one evaluation per guard, zero disabled
  score evaluations, and one evaluation per enabled score;
- transient Drop before state update and Step return;
- stable no-enabled and NaN panic reasons;
- strict-C11 static switch/branch/comparison lowering with no heap, scheduler,
  coroutine, candidate vector, or decision runtime object.

MIR retains categorical match arms and guarded score candidates rather than
parser-time if chains. Planner artifacts retain evaluation order, hardmax/tie
policy, failure policy, and cleanup edges. Malformed decision MIR and altered
automata plans are rejected deterministically.

## Diagnostics

R5b adds the semantic families
`TRANSITION_MATCH_REQUIRES_MATCHABLE`,
`TRANSITION_MATCH_DUPLICATE_ARM`,
`TRANSITION_MATCH_NONEXHAUSTIVE`,
`TRANSITION_MATCH_UNKNOWN_TARGET`,
`TRANSITION_MATCH_MIR_INVALID`,
`TRANSITION_DECIDE_EMPTY`,
`TRANSITION_DECIDE_GUARD_REQUIRES_BOOL`,
`TRANSITION_DECIDE_SCORE_TYPE_INVALID`,
`TRANSITION_DECIDE_SCORE_TYPE_MISMATCH`,
`TRANSITION_DECIDE_UNKNOWN_TARGET`, and
`TRANSITION_DECIDE_MIR_INVALID`. Ordinary pattern diagnostics remain
authoritative for invalid enum names, variants, payload arity, and bindings.
NaN is a validated score type with a runtime terminal policy, not a static
diagnostic.

## Deferred

R5b does not implement plain value-level `decide`, infer, transition infer,
softmax, probability distributions, sampling, temperature, entropy, TopK,
hysteresis, minimum commitment, tie memory, yield, scheduling, async/coroutines,
effects/actuator redesign, closures, or LIR.

Recommended R5c is the already-scoped yield/re-entry milestone over R5a storage:
explicit automata state and machine fields survive, transient locals do not,
and no scheduler or coroutine runtime is implied.
