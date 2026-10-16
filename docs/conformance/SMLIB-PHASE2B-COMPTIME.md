# Phase 2b scoped comptime values

The derivative staging experiment exposed a missing composition path:

```concept
template <usize N>
int Row(int value) {
    int result = 0;
    comptime for (k in 0u .. N) { result += Cell<k>(value); }
    return result;
}
```

The static-control evaluator knew `k`, but function-template instantiation
resolved named non-type arguments only through module declarations. It reported
`GENERIC_NON_TYPE_ARGUMENT_INVALID` for an actual compile-time integer.

Explicit function-template calls now close lexical comptime value arguments
through the existing evaluator before consulting the scope-free instance cache.
Open template value parameters remain symbolic until their owning closure.
The existing non-negative, bounded integer and parameter-type range checks
remain authoritative. Runtime shadows cannot fall through to module constants.
There is no grammar addition, fuel increase, or geometry-specific compiler path.

A second regression covered closed runtime template bodies reading imported
module comptime tables. Their validation scope now uses `evt1ModuleScope`, as
ordinary and open functions already do, before adding runtime parameters.
`TestClosedImportedTemplateRetainsModuleComptimeConstants` executes a table read
and preserves static out-of-bounds rejection in Normal and Verify.

`TestStaticIterationItemClosesImportedTemplateValue` covers artifact-only imports,
local and module constants, dependent and literal iteration bounds, deterministic
Normal/Verify output and strict-C11 native execution.
`TestScopedTemplateValueRejectsRuntimeAndInvalidInteger` rejects runtime values,
negative integers, booleans, excessive uint64 values and runtime shadows.

The experiment retains two separate limits: a flat large quotient expansion
can exceed the existing 4,096-step fuel budget; a dependent local declaration
inside an eagerly expanded constant-bound loop is still not generalized here.
Derivative-cell helpers use generic parameters directly and compile through the
ordinary compositional path. Consumer measurements and qualification are in
`SMlib_Concept/phase2b`.
