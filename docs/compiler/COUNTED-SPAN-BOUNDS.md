# Counted span bounds

The local checked-control-flow analysis can prove `view[i]` in an ascending
`for (i in 0 .. Len(view))` range without knowing the runtime extent. This is
an optimization of existing semantics, not a new source contract or alias rule.

The semantic pass runs on validated, closed typed bodies before MIR collection.
It assigns a deterministic length snapshot to each tracked by-value descriptor.
The loop induction value is nonnegative and strictly below that snapshot. A
matching index operation receives `local_bounds.counted_span_range` in MIR.
The General Planner selects the existing `StaticEliminated` bounds strategy in
Normal policy; Verify retains the access observation. C emission consumes the
exact operation's plan. No generated-C rewriting or source annotations occur.

The upper bound is the builtin `Len`, evaluated exactly once under the existing
range semantics. Its checked int representation bounds the induction interval.
The proof requires a named span descriptor and named index. Arithmetic on the
index, another descriptor, while loops, user-defined `Len`, and unknown forms
retain checks. A descriptor assignment creates another snapshot; a branch join
keeps a snapshot only when both paths agree. Loop-carried assignments are
forgotten before analyzing the body. Exposed and captured descriptors/indices
are conservatively excluded for the entire function. Backing-element writes
do not mutate a descriptor's length and do not grant disjointness.

The same work repairs the existing capture scan: shorthand `with (ref view)`
stores the lexical name in `CaptureBinding.Name`, with a nil `Source`. Scanning
only `Source` could incorrectly preserve a numeric length proof after a
callback changed the descriptor. `TestLocalBoundsShorthandCaptureMutationKeepsFailure`
minimizes this case and executes the required failure in Normal and Verify.

`counted_span_bounds_test.go` covers dynamic/empty extents, identical and shifted
overlapping backing, a short secondary input, descriptor reassignment, loop
carried mutation, captures, escapes, index arithmetic/reassignment, shadowing,
ordinary while fallback, artifact-only generics and deterministic regeneration.
An `INT_MAX` starting value does not create a contradictory interval: the proof
is withheld and the existing range-direction failure remains observable.
MIR validation checks the proof variant's structural invariants; imported AST
annotations are never proof authority and are recomputed from validated bodies.

There is no `restrict` emission. Readonly, mutable, contiguous and fixed-shape
views can overlap. This proof concerns a length value, not a memory region's
exclusive access. It transfers to future native span lowering as an operation
precondition; the current EVT2 lowerer is not extended by this change.

Consumer experiments and Clang/GCC comparisons live in the sibling
`SMlib_Concept/phase3` repository, keeping workload sources outside the compiler.
