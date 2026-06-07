# Phase 5a judgment expressions: `decide`

Phase 5a introduces Concept's first judgment expression: `decide`, a typed, deterministic, stateless utility-scoring expression over enum variants. This milestone is documentation-only. It does not implement lexer, parser, AST, HIR, MIR, backend, or fixture-runner changes.

## Thesis

```text
match asks: what is this value?
decide asks: which value should win?

A decide expression makes utility scoring explicit, typed, deterministic,
and lowerable instead of hiding policy inside imperative if/else ladders.
```

Concept is a C++-lineage systems language, and many systems programs contain decision logic that is really utility scoring: choose a channel, mode, strategy, alarm level, scheduling class, fallback path, or recovery action based on several considerations. Ordinary imperative code can express those choices, but it often hides the decision model inside branches, local variables, and early returns.

A `decide` expression makes the model explicit:

- candidate outcomes are enum variants;
- eligibility is written as a `when` condition;
- utility is written as a required integer `score`;
- winner selection is defined by the language;
- tie policy is deterministic;
- fallback behavior is required in v0.

The language primitive is intentionally stateless. Stateful policy machinery such as hysteresis, minimum commit windows, cooldowns, smoothing, previous-winner memory, or explanation traces belongs in libraries or future richer judgment constructs, not in the bare expression.

## Motivation

Systems code frequently contains logic shaped like this:

```cpp
AlertChannel channel = AlertChannel::Nominal;
int best = 0;

if (fault && temperature > 900) {
    channel = AlertChannel::Critical;
    best = 120;
}
if (temperature > 750 && 85 > best) {
    channel = AlertChannel::Warning;
    best = 85;
}
if (pressure > threshold && 70 > best) {
    channel = AlertChannel::Advisory;
    best = 70;
}
```

This code is executable, but its decision shape is accidental. A reader has to reconstruct:

- which outcomes were candidates;
- which candidates were eligible;
- which score each consideration contributed;
- whether ties pick the first candidate, the last candidate, or depend on incidental control flow;
- whether a fallback always exists;
- whether prior state is involved.

Concept should expose this class of policy as a first-class expression because it is common, review-sensitive, and lowerable to normal control flow. `decide` is related to utility AI and scoring decisions, inspired by Oct's `when utility`, and conceptually close to Dominatus-style `Consideration` objects. Concept's v0 choice is smaller: no board memory, no behavior-tree or state-machine integration, no runtime framework, and no hidden state.

## Keyword choice

The Phase 5a keyword is:

```text
decide
```

Rejected alternatives:

- `select` is accurate but too generic and SQL-flavored; it reads more like data projection than policy.
- `choose` is readable but casual; it undersells that the construct has typed candidates, scores, totality rules, and deterministic tie-breaking.
- `judge` is expressive but should remain available for a future explanation-carrying judgment object, such as `Judgment<T>` with a winning reason and score.

`decide` best captures the intended role: a typed decision policy that computes which candidate should win.

## v0 syntax

Example:

```cpp
AlertChannel channel = decide AlertChannel {
    Critical when fault && temperature > 900 score 120;
    Warning  when temperature > 750          score 85;
    Advisory when pressure > threshold       score 70;
    Nominal                                  score 0;
};
```

Grammar sketch:

```text
decide_expr
  := "decide" TypeName "{" decide_arm* "}"

decide_arm
  := VariantName ("when" Expr)? "score" Expr ";"
```

Syntax rules:

- `decide` is an expression.
- It returns the enum type named after `decide`.
- Arm variant names are unqualified inside `decide EnumType { ... }`.
- A `when` condition is optional.
- Missing `when` means the arm is always eligible.
- `score` is required for every arm.

## v0 semantic rules

Semantic rules for Phase 5a v0:

- The type after `decide` must be an enum.
- Each arm must name a variant of that enum.
- v0 supports only zero-payload variants as candidates.
- Duplicate variants are allowed.
- A `when` condition must have type `bool`.
- A `score` expression must have type `int`.
- Scores may be negative.
- All arms are evaluated in source order.
- For each arm, the condition is evaluated before the score.
- The score is evaluated only if the condition is true.
- The highest score among eligible arms wins.
- Ties are broken by source order: the first eligible arm with the highest score wins.
- At least one unconditional arm is required in v0.
- The `decide` expression type is the enum type.
- `decide` has no hidden state or memory.

These rules deliberately make the expression total and deterministic while preserving ordinary expression evaluation for conditions and scores.

## Totality and fallback rule

v0 requires:

```text
A decide expression must include at least one unconditional arm.
```

This avoids runtime "no candidate" failures. Because at least one arm is always eligible, a v0 `decide` expression always produces an enum value. Future fallible selection can be considered later, for example as `decide?`, but v0 stays total and deterministic.

Valid:

```cpp
AlertChannel channel = decide AlertChannel {
    Critical when fault score 100;
    Nominal             score 0;
};
```

Invalid:

```cpp
AlertChannel channel = decide AlertChannel {
    Critical when fault score 100;
    Warning  when hot   score 80;
};
```

The invalid example has eligible candidates only when runtime conditions are true, so it lacks a static totality guarantee.

## Duplicate variants

Duplicate variants should be allowed in v0:

```cpp
AlertChannel channel = decide AlertChannel {
    Critical when reactorFault score 120;
    Critical when powerFault   score 110;
    Warning  when tempHigh     score 80;
    Nominal                    score 0;
};
```

Rationale:

- Multiple considerations may vote for the same outcome.
- This mirrors utility AI and Dominatus-style consideration scoring.
- The selected result is still just the enum value.
- The winning arm, score, and reason are not preserved by `decide` v0.
- A future richer `judge` or `Judgment<T>` form may preserve the winning reason, score, and arm identity.

Allowing duplicate variants keeps v0 useful without prematurely introducing explanation objects.

## Relationship to `match`

```text
match asks: what is this value?
decide asks: which value should win?
```

`match` inspects an existing value:

```cpp
match (status) {
    Status::Ok(value) => return value;
    Status::Err(code) => return code;
}
```

`decide` computes a value from candidates and scores:

```cpp
AlertChannel channel = decide AlertChannel {
    Critical when fault score 100;
    Nominal             score 0;
};
```

The two constructs are complementary. `match` is destructuring and control selection over a value that already exists. `decide` is value construction by ranked candidate selection.

## Relationship to Oct and Dominatus

Phase 5a is informed by existing utility-selection ideas but keeps the Concept primitive narrow.

- Oct's `when utility` is flow/state-machine oriented and may integrate with board memory.
- Dominatus `Consideration` is library/framework utility scoring.
- Utility AI generally decomposes decisions into candidate actions or outcomes with scores, then selects a winner by a policy.

Concept's `decide` is the stateless language primitive for the common core:

- candidate;
- eligibility;
- score;
- deterministic winner.

Stateful behavior is outside the language primitive. That includes board memory, previous winner tracking, switching costs, cooldown windows, smoothing, and explanation capture.

## Future stateful policy wrappers

Stateful behavior can be layered by libraries or future standard-library types around a stateless `decide` result or around a future inspectable judgment value.

Possible future library shape:

```cpp
SelectPolicy<AlertChannel> policy {
    current: AlertChannel::Nominal,
    hysteresis: 10,
    minCommit: 3,
};

AlertChannel channel = policy.evaluate(decide AlertChannel {
    Critical when fault && temperature > 900 score 120;
    Warning  when temperature > 750          score 85;
    Advisory when pressure > threshold       score 70;
    Nominal                                  score 0;
});
```

The following belong in library/stateful wrappers, not in the bare language expression:

- hysteresis;
- minimum commit windows, also spelled `min_commit` in some APIs;
- cooldowns;
- previous winner memory;
- smoothing;
- explanation traces.

The bare `decide` expression should remain a pure, stateless computation over the current program state and its arm expressions.

## Future `judge` / explanation object

A future explanation-carrying construct might look like this:

```cpp
Judgment<AlertChannel> judgment = judge AlertChannel {
    Critical reason "fault" when fault score 120;
    Warning  reason "hot"   when hot   score 80;
    Nominal  reason "base"           score 0;
};
```

This is future-only and not part of Phase 5a v0. It likely requires richer library types, generics or templates, and a representation capable of preserving score, reason, and winning arm metadata. `decide` v0 intentionally returns only the enum value.

## HIR impact

Eventual HIR additions:

- Decide expression.
- Decide arms.
- Resolved enum type.
- Resolved variant IDs.
- Condition expressions.
- Score expressions.
- Source-order arm index.
- Metadata that at least one unconditional arm exists.

Type checking responsibilities:

- The enum type exists.
- The named type is an enum.
- Variants exist on that enum.
- Candidate variants have no payload in v0.
- Conditions are `bool`.
- Scores are `int`.
- The `decide` expression type is the enum `TypeId`.

HIR should preserve source-order arm indices because tie-breaking is semantic, not an incidental lowering detail.

## MIR lowering strategy

A possible lowering uses ordinary control flow and assignments:

```text
hasWinner = false
bestScore = 0
bestValue = <enum temp>

for each arm in source order:
    if condition is absent or condition is true:
        scoreTemp = score
        if !hasWinner || scoreTemp > bestScore:
            hasWinner = true
            bestScore = scoreTemp
            bestValue = Enum::Variant

result = bestValue
```

Lowering notes:

- `>` preserves first-arm tie-breaking because equal scores do not replace the current winner.
- v0 requires an unconditional arm, so a winner is guaranteed.
- `hasWinner` avoids needing a special minimum-int literal.
- Lowering can use existing enum constructor support.
- No new MIR terminator is required.
- Lowering creates normal blocks, conditions, assignments, and temporaries.

## C backend impact

If `decide` lowers through existing MIR constructs, no special C backend feature should be needed beyond existing support for:

- enum constructor emission;
- enum assignment;
- condition/control-flow emission;
- local temporaries.

C output can remain label/goto-shaped, following the Phase 4 MIR-backed backend style. Deterministic `cpt_*` names continue to apply to any backend-owned temporaries, enum layouts, or helper names.

## Diagnostics

Proposed stable diagnostics:

- `CON0060 UnknownDecideEnum`
- `CON0061 UnknownDecideVariant`
- `CON0062 DecideVariantHasPayload`
- `CON0063 DecideConditionNotBool`
- `CON0064 DecideScoreNotInt`
- `CON0065 DecideMissingUnconditionalArm`

The numbering can be adjusted if conflicts exist when Phase 5a implementation begins. Once fixtures land, they should assert stable diagnostic codes.

## Fixture strategy

Planned fixtures:

- Parse fixture for basic `decide`.
- Check fixture for unknown enum.
- Check fixture for unknown variant.
- Check fixture for payload variant candidate rejection.
- Check fixture for non-bool condition.
- Check fixture for non-int score.
- Check fixture for missing unconditional arm.
- MIR snapshot for simple `decide`.
- MIR snapshot for duplicate variant arms.
- Run fixture for highest score.
- Run fixture for tie-breaking by source order.
- Run fixture for fallback unconditional arm.
- Run fixture for score evaluated only when eligible, if observable side effects or testable evaluation hooks exist later.

Fixture growth should follow the milestone ladder. Parse and check fixtures can land before runtime support. MIR and run fixtures should land only when the real parse -> HIR -> MIR -> MIR-backed backend path supports the feature.

## Milestone ladder

```text
P5a-M0  Judgment / decide design doc
P5a-M1  Parse decide expressions
P5a-M2  HIR/type checking for decide
P5a-M3  MIR lowering for decide
P5a-M4  Runtime fixtures and backend stabilization
P5a-M5  Phase 5a closeout
```

Phase 5a should remain compact. The purpose is to add one stateless, typed utility-scoring expression, not a full AI policy framework.

## Close criteria

Phase 5a is complete when:

- `decide` parses.
- HIR resolves enum and variants.
- Conditions check as `bool`.
- Scores check as `int`.
- Payload candidates are rejected in v0.
- At least one unconditional arm is required.
- Duplicate variants are allowed.
- Tie-breaking by source order is tested.
- MIR lowering works through existing constructs.
- Run fixtures prove winner selection.
- No stateful policy is built into the language primitive.

## P5a-M0 close criteria

P5a-M0 is complete when:

- This design document exists.
- Motivation, keyword choice, syntax, semantic rules, totality, duplicate variants, relation to `match`, Oct and Dominatus context, future wrappers, future `judge`, HIR impact, MIR lowering, C backend impact, diagnostics, fixture strategy, milestone ladder, close criteria, and non-goals are documented.
- No compiler implementation code has been added.
- `zig build test` still passes.

## Non-goals

Phase 5a deliberately does not include:

- Stateful hysteresis in the language primitive.
- `min_commit` or minimum commit windows in the language primitive.
- `judge` or any explanation object.
- A `Judgment<T>` generic type.
- Payload variant construction in `decide` arms.
- Fallible `decide?`.
- Generic score concepts.
- Float scores.
- Score normalization or curves.
- Policy memory.
- Optimizer or scheduler integration.
- Dominatus runtime integration.
- Lexer, parser, AST, HIR, MIR, backend, or fixture-runner implementation in P5a-M0.

## P5a-M1 implementation note

P5a-M1 adds parser and AST support for `decide` expressions. The lexer reserves `decide` and continues to reserve `when`; `score` remains contextual and still lexes as an ordinary identifier outside decide arms.

The parser now accepts `decide TypeName { ... }` wherever expressions are parsed, records simple-identifier arm variants, optional `when` conditions, required contextual `score` markers, required score expressions, and semicolon-terminated arms. Empty arm lists are accepted by the parser so later semantic validation can diagnose totality and enum-specific rules in one place.

HIR lowering, type checking, enum/variant resolution, condition and score type checks, unconditional-arm validation, MIR lowering, backend behavior, runtime fixtures, stateful policy wrappers, `judge`, and `Judgment<T>` remain future milestones.
