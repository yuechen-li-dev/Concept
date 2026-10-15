# SMLib Phase 1 compiler hardening

The unchanged external port at `../SMlib_Concept/phase1` is the regression
baseline: 239 cases and 478 Normal/Verify differential comparisons. Original
numerical algorithms, C++ oracle, attribution, and known-defect records remain
unchanged. Evidence logs are under ignored `artifacts/smlib-hardening`.

## Pass A: callback correctness

Callback return inference now participates in ordinary statement validation:
preceding locals are bound before return expressions are typed, branches share
one inferred result, and ordinary mutation, move, capture, and escape checks
remain active. There is no preliminary value-expression-only scan or second
ownership-consuming inference pass.

An erased callback reference records its semantically resolved concrete identity
and environment. Typed expression traversal covers nested constructs, closures,
loops, failure unwrap/propagation, and assertions. MIR witness collection and C
adapter emission consume one identity-deduplicated collection. Identical
signatures do not merge distinct callbacks or shadowed capture environments.
C emission no longer guesses identity by searching source variable names.

Four checked-in positive fixtures and negative safety tests supplement the
existing callable capture/types conformance corpus. They execute strict C11
in Normal and Verify, check deterministic artifacts, and verify one adapter per
semantic identity. The unchanged external port's three native facts pass in
both modes. No loop escape, C-style loop, automata, scheduler, or runtime frame
semantics were changed.
