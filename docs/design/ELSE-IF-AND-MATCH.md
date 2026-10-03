# Else-if and match

Concept keeps `else if` as familiar C-family control flow. Projects may prefer
`match` when a ladder enumerates cases of one semantic subject, especially where
match improves readability or exhaustiveness.

`if` / `else if` expresses sequential control flow. `match` expresses alternatives
in a known decision space. `if` is appropriate for sequential guards and
heterogeneous conditions. `match` is preferred for categorical alternatives in
one decision space.

Style preference is expressed through project policy in manifest.concept rather
than by restricting the language grammar. Oct projects may prohibit else-if
ladders in favor of match/switch; Concept does not import that stricter rule.
Statements and value-selection expressions both support ordinary else-if.

## Manifest activation

Use the existing immutable four-string policy surface:

```concept
record struct LintPolicy {
    string concept; string severity; string kind; string subject;
}
concept PreferMatchOverElseIfLadder<declaration F> {
    requires compiler.NoMatchShapedElseIfLadder(F);
}
comptime LintPolicy MatchPreference = LintPolicy{
    "PreferMatchOverElseIfLadder", "warning", "FunctionDeclaration", ""
};
```

`warning` is the recommended severity. Existing `error` severity makes findings
fail `concept lint`; it does not invalidate ordinary compilation. Disable by
omitting the activation value. The existing surface has no `disabled` severity.
Use `MethodDeclaration`, or an empty kind selector, to include methods.
The policy name is ordinary Concept: another project may name its requirement
differently. The compiler does not select behavior by that name or filename.

PreferMatchOverElseIfLadder is project policy. It does not alter parsing,
typing, lowering, or runtime behavior of `else if`. `concept lint` and
`concept lint --verify` evaluate it. Ordinary compilation does not load these
lint requirements from a manifest. Existing explicit concept assertions retain
their normal semantics. No new optimizer fact or backend authority is created.

## Observation and conservative detection

`compiler.NoMatchShapedElseIfLadder(F)` is a declaration analysis over the
complete checked function body. Concept owns the requirement; Go supplies the
semantic observation through the existing declaration proof projector.
No Go AST type is exposed to Concept. The observation records the function's
bound owner/site, ladder site, discriminant identity/name, comparison count, and
whether the compiler knows a finite enum domain.

A match-shaped statement ladder has at least three equality comparisons, all
reading the same bound storage subject and comparing distinct stable cases.
The final `else` is not counted. Equality may be reversed. Parentheses are
transparent. Bound declaration sites distinguish shadowed locals; a checked
field path distinguishes fields. Source strings are not compared, and value
copies or reference aliases are not assumed to be identical storage.

Supported cases are integer literals (including negative values and equivalent
numeric spellings) and tag-only enum variants. Enum-typed state/tag fields use
the same finite-domain observation; integer discriminator fields use the scalar
observation. This does not add a tagged-union or machine-state comparison API.
Payload-bearing enum equality already requires an explicit comparison operation
and remains outside this observation. Opaque machine-state access, arbitrary
computed subjects, reference reads, indexing, calls and computed case expressions
are conservatively unrecognized. No subjectless-match suggestion is made.
Expression-level else-if is legal but is outside this initial statement lint.

Two-comparison chains, heterogeneous guards, duplicates, and side-effectful
pseudo-common subjects produce no preference finding. Early returns neither
trigger nor suppress the rule: only a common categorical discriminant triggers
it. An unrecognized outer chain is not reinterpreted as a warning-worthy suffix.
Explicit `else { if (...) ... }` remains a separate nested construct.

Unknown expressions supply no candidate. An unavailable or incomplete checked
body returns Unknown and supplies no preference finding; existing unrelated
policy requirements preserve their normal Unknown behavior. Proven means no
safely established candidate in the supported observation, not a theorem about
all possible transformations of the function.

Observations belong to the checked function's lexical validation scope. A
template instantiation or callable body validated while checking a caller does
not become a ladder in that caller. Open templates and generated callable
implementations are outside this initial authored-function observation.

## Findings, formatting, and fixes

One ladder produces one finding at its root `if`. Separate nested ladders may
produce their own findings. The message carries manifest source/line provenance:

```text
warning [PreferMatchOverElseIfLadder] this 3-branch chain repeatedly discriminates
`kind`; prefer `match (kind)` for readability and exhaustiveness;
policy from tests/lint/r9d/manifest.concept:14
```

Integer cases say `for clearer case structure`, without an exhaustiveness claim.
Finite enum cases mention the value of match's existing exhaustiveness checking,
without claiming the ladder itself covers every case.

`concept explain program.concept --policy PreferMatchOverElseIfLadder --subject
Decode --verbose` uses the existing proof graph. A candidate refutes the
absence requirement (Disproven); its compiler-derived observation explains the
branch count/discriminant and its manifest reason explains severity/provenance.

Allman formatting keeps the canonical sequence:

```concept
    }
    else if (kind == Kind::B)
    {
```

The existing default same-line brace style retains `} else if (...) {`.
Both styles keep `else if` together and are idempotent. Formatting remains
presentation-only and preserves comments and identifier spellings.

This milestone is **suggestion-only**. No `lint --fix` rewrite is implemented;
comment/trivia-preserving source transformation is deferred. Match semantics and
exhaustiveness rules are unchanged. R9d does not begin EVT2f.
