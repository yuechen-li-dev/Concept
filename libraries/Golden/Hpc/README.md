# Heat stencil

`Stencil.concept` advances a one-dimensional fixed heat field, checks an
explicit stability envelope, rejects mismatched residual extents, and
computes reductions over contiguous data. `stencil.concept_test` checks a
spike step, error paths, and the allocation effect. The first draft is in
`first-draft.concept.txt`.

Friction: C-style loop and implicit array-to-Span conversion are replaced by
a range loop and explicit `Span(backing)` or `ReadOnlySpan(backing)`.
`AdvanceHeat` avoids the machine intrinsic named `Step`. Familiar: stencil
arithmetic and fixed extents. New: table/Span borrowing. Advanced: proof.
The burn-in also caught an artifact NoAllocation summary gap for `Len` and a
generated-C collision with local names that are C keywords; both are fixed.
