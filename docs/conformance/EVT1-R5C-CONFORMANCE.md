# EVT1 R5c conformance

Status: implemented and executable

Baseline: `fe418616af48453a0f6178e0a32fb37d871c457f`

Compiler: `concept-evt1-stage0-go`

## Scope and classification

R5c is EVT1-new; no PoC3 parity is fabricated. The corpus contains 30 `PASS`
cases: 17 valid, ten static rejection, and three runtime-negative. There are
zero `EXPECTED-DIVERGENCE`, `EVT1-GAP`, `POC3-LEGACY`, or
`SPEC-AMBIGUITY` cases.

## Semantic evidence

- `Inference<T>` is fixed-inline, copyable, non-owning, and allocation-free.
- Unique payload-free enum candidates share R5b's exactly-once guard/score law;
  inference scores are float logits.
- Temperature-1 stable softmax handles `[0,0]`, `[0,ln(3)]`, and
  `[1000,1001]` without naive exponential overflow.
- Disabled and negative-infinity candidates have zero mass. Positive
  infinities split mass equally. NaN, no-enabled, and all-negative-infinity
  paths terminate with stable reasons.
- Probability indexing, `HardMax`, and `Confidence` are explicit. HardMax
  uses first source-order maximum and agrees with direct decide, including ties.
- `transition infer with HardMax` normalizes, selects, cleans transients,
  commits the state tag, and returns. Missing/unknown policies reject.

## Compiler evidence

MIR retains candidate type/order, float score domain, `StableSoftMax`,
temperature 1, exceptional-value policies, and transition cleanup/policy.
Planner selects `ScalarStableSoftMax`, `InlineFixed`, max subtraction, and
no SIMD. Strict-C11 uses fixed arrays/scalars and `expf`, with no heap, model
runtime, RNG, scheduler, GPU, or inference registry.

## Deferred after R5c

Sampling/RNG, temperature syntax, thresholds, TopK, entropy, beam search, model
declarations/training/formats, quantization, SIMD/GPU, DragonGod stateful
policy, yield, effects/actuators, and LIR remain deferred. Recommended R5d is
bounded yield/resume over the explicit R5a storage law, without a scheduler.
