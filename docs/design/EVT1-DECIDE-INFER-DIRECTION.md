# EVT1 decide and infer direction

Status: R5c hardmax and normalized inference implemented

## Conceptual ladder

```text
match
    categorical selection
score
    scalar evidence or utility for one candidate
decide
    hardmax / argmax over enabled scores
infer
    normalized soft belief distribution
transition decide
    hard selected local state transition
transition infer
    belief plus explicit policy to transition
```

These layers are intentionally distinct. `match` answers which closed pattern
is present. A `score` is one ordered scalar, not a probability. `decide`
compares all enabled candidate scores and returns the hard maximum. `infer`
retains normalized uncertainty across candidates rather than discarding it.

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

## R5c normative inference boundary

`infer` returns `Inference<T>`, a fixed-inline distribution over a unique,
closed, source-ordered subset of payload-free enum candidates. Guards reuse
R5b ordering; scores are exactly float logits. Temperature-1 stable softmax
uses maximum subtraction. NaN and an empty enabled set panic. Positive
infinities split mass equally, negative infinity has zero mass, and
all-negative-infinity support panics.

`belief[Type::Candidate]`, `HardMax(belief)`, and `Confidence(belief)` are
the complete R5c query surface. Inference has no truthiness. Direct decide
remains the cheaper raw-score argmax.

`transition infer with HardMax` remains distinct MIR and Planner intent. The
baseline C11 path performs normalization before policy selection and uses the
ordinary cleanup edge. Future `Sample(ref rng)`, `Threshold`, and
`DeferBelow` must expose every dependency. Temperature syntax, TopK, entropy,
model declarations, and model runtimes remain absent.

DragonGod may consume beliefs through explicit state for hysteresis,
confidence thresholds, smoothing, or commitment. That state stays library
owned.
