# R8 semantic project policy

> Project policy may require semantic facts, but it cannot create them.

> `concept lint` must operate on bound semantic declarations and proposition
> truth, not source-text pattern matching.

R8e2 begins with `DeclarationSubjects` over the validated `Module`. The
projection has stable declaration identity, kind, defining module, source site,
and Authored/Generated/Foreign provenance. `ProjectDeclarationSubjects` scopes
the view to the current module, excluding dependency implementation declarations
loaded from semantic artifacts. Generated functions use `GeneratedOrigin`;
foreign functions use the existing `ExternABI` declaration. These categories
are never inferred from spelling or path.

R9d adds a checked function-body observation through the same declaration
subject and proof path. `PreferMatchOverElseIfLadder` may require
`compiler.NoMatchShapedElseIfLadder(F)` in `manifest.concept`, normally with
warning severity. See [Else-if and match](ELSE-IF-AND-MATCH.md) for the supported
discriminants, conservative exclusions, source anchoring, and explain behavior.
Else-if remains ordinary legal syntax; this policy does not change semantics.

> Project policy is ordinary Concept over semantic program subjects, not a
> separate lint language. Policy may require truth. Policy cannot create truth.

Concepts may bind `declaration D` and use normal `requires` entries:

```concept
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
concept HotPathPolicy<declaration F> { requires compiler.NoAllocation(F); }
```

`NoAllocation` uses the same proof projector as `Assert.Concept`; Proven,
Disproven, and Unknown remain distinct. The manifest selects severity and
subjects, never a different truth. No policy enters MIR or runtime C.

Naming is project policy, not grammar. The intended canonical style is
PascalCase for types, functions, methods, concepts, interfaces, and machines;
camelCase for fields, locals, and parameters. Generated and foreign declarations
are exempt from the default authored policy by semantic provenance; an
explanation shows the exemption. Explicit `PascalCase`, `CamelCase`, and
`SnakeCase` requirements may apply to other provenance categories.

`manifest.concept` holds ordinary immutable `LintPolicy` values with four
strings: concept identity, `warning` or `error`, optional declaration kind, and
optional exact subject name. Empty selectors mean all project declarations.
Concept identities bind against the semantic concept environment. Contradictory
active naming requirements produce `LINT_POLICY_CONFLICT`. Only `LintPolicy`
values authored in the current manifest activate policy; imported compile-time
values do not activate root lint.

```concept
record struct LintPolicy {
    string concept; string severity; string kind; string subject;
}
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "warning", "", ""};
```

`concept lint file.concept` and `concept lint project/` use the ordinary
compiler module resolver. Warnings return zero; error findings return nonzero.
`--verify` runs the same compile-time policy evaluation.

Language errors such as ownership violations, invalid types, and ignored
MustUse results remain core compiler obligations. Naming and requirements such
as NoAllocation are project policy findings. Policy cannot change codegen or
upgrade Unknown or Disproven facts to Proven.

Interface/override naming constraints and preferred immutability remain
deferred. R8e3 owns trivia preservation and `concept fmt`. R8f follows R8e3;
R7q remains paused.
