# EVT1 decide and infer direction

Status: R5b hardmax implemented; inference is design-only

## Conceptual ladder

```text
match
    categorical selection
score
    scalar evidence or utility for one candidate
decide
    hardmax / argmax over enabled scores
infer
    future normalized soft belief or distribution
transition decide
    hard selected local state transition
transition infer
    future inference-policy-driven transition
```

These layers are intentionally distinct. `match` answers which closed pattern
is present. A `score` is one ordered scalar, not a probability. `decide`
compares all enabled candidate scores and returns the hard maximum. Future
`infer` would retain uncertainty across candidates rather than discarding it.

## R5b normative boundary

R5b implements only `transition match` and `transition decide`. The latter is a
stateless local operation for one `Step`: guards run once in declaration order;
disabled scores do not run; enabled scores run once in declaration order; the
first maximum wins. It accepts one exact `int` or `float` score type per block,
panics on NaN, and panics when there is no enabled candidate. Selection cleans
transient state, updates the local machine state, and ends the Step.

Plain value-level `decide` is useful future syntax, but would broaden expression
grammar and is deferred. R5b does not add a generic ordering abstraction.

## DragonGod separation

DragonGod Decision is library-level stateful policy across Steps. Hysteresis,
minimum commitment, temporal smoothing, current-choice preference, tie memory,
and policy history belong to its explicit application-owned state. The language
hardmax primitive contains none of them and cannot silently consult them.

## Future inference boundary

`infer` and `transition infer` remain non-normative. A later milestone must
separately define candidate identity, normalization, numeric stability,
zero-mass behavior, distribution representation, sampling policy, ownership,
and Planner/backend contracts. R5b implements no softmax, probability
distribution, temperature, entropy, TopK, random sampling, or inference policy.
