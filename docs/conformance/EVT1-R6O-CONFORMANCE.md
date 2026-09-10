# EVT1 R6o conformance — concepts in Concept

Compiler: `concept-evt1-stage0-go`

R6n stopped at `CV4170` because a constraint on `TAllocator` was compared to
the first parameter `T`, and at `CV4027` because removing the constraint left
no required-operation authority. R6o generalizes the representation to a
concept application with structural argument lists.

## Pinned evidence

- `generic_requirements_test.go` covers parameter two, parameter four,
  relational and repeated arguments, concrete-plus-generic arguments, composed
  and three-level diamond closure, conflicts, open-definition rejection,
  concrete missing-operation diagnostics, generic types, module import without
  source, proof structure, effect authority, `NoAllocation`, origin distinction,
  and 100-run artifact determinism.
- `r6n_allocator_feasibility_test.go` now positively pins the former
  second-parameter/member-operation and constrained-Drop blockers while
  retaining the unconstrained `CV4027` regression.
- The semantic corpus adds three valid and three static-invalid generic
  requirement fixtures. Current corpus totals are 382 valid, 246 static
  invalid, and 13 runtime negative.

The module's compiler-private typed payload serializes concept and template
parameter lists, structural application arguments, nested prerequisites,
required operation declarations, compiler effect requirements, and bodies.
Consumers decode that payload and instantiate it without opening source.

## No-cheating audit

The implementation contains no allocator, owner, pool, arena, bump, or memory
policy recognition; no runtime witness; no source reparse; no boolean constraint
expressions; no SFINAE; and no dynamic-dispatch fallback. Member spelling
authorized by a required operation normalizes to the existing free-operation
representation rather than injecting a nominal method.

Deferred: Standard.Memory retry, partial specialization, variadics, boolean
constraint algebra, subsumption ordering, generic/runtime reflection,
source-level `T*`, MMIO, and native LIR/backend work.
