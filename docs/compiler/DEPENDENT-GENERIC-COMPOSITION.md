# Dependent generic composition qualification

The motivating failure was SMLib Phase 1a's `Fixed<N>` local construction:
`GENERIC_NON_TYPE_ARGUMENT_INVALID: N is not a compile-time integer`. The fix
retains a structural application until its bound arguments close, using the
existing type algebra and closed generic identity metadata.

## One representation across the pipeline

| Boundary | Authority |
| --- | --- |
| Parsing and open checking | Lexically bound value parameters are `TypeTemplateValue` leaves; type parameters remain `TypeConceptParam`. Applications retain ordered children. Arity and argument kinds are checked before an open application is returned. |
| Substitution | `evt1SubstituteType` traverses both type and value children, including symbolic-to-symbolic renaming and extent-expression operands. The former separate integer-only extent substitution delegates to it. |
| Instantiation | Closed arguments normalize before cache lookup. Existing `GenericApplication` stores declaration/module identity and ordered arguments; no open instance is cached. |
| Local construction | Semantic validation retains the resolved `StructType` and name on the actual constructor node. Direct initialization compares semantic types. |
| Field projection | `evt1StructView` projects the same application structure. Expression/place checking resolves projected fields in the active scope, without writing scope-specific results into shared field maps. Diagnostics use sorted field names. |
| Ownership | Open-record copyability inspects projected fields with a recursion guard. Concrete bodies still use the existing move, drop, borrow and escape checks. |
| Module artifacts | Existing structural type/value nodes and closed identity metadata round-trip through semantic artifacts. Neither a second source parser nor a new artifact payload format is introduced. |
| Emission | C lowering consumes the constructor's semantically resolved type. A named comptime argument no longer leaves a stale display spelling that indexes a missing struct declaration. |

Analysis detaches argument lists before normalizing them, so separate concrete
instances do not rewrite shared open declarations. Open storage dimensions may
remain unresolved during checking; concrete field resolution evaluates them
before C11 emission. This is not runtime-shaped owning storage or a VLA facility.

## Combined regression

`language/evt1/dependent-composition/Library/Dependent.concept` defines a required
binary64 evaluator operation, a `Values<Capacity>` record with `Capacity * 2u`
backing extent, a constrained `Workspace<T, Capacity>` with an owned resource,
a function returning `Values<N>`, and a relational `Numerical<T, Values<N>>`
requirement whose Head operation is supplied by a generic function. Dependency
detection and inference retain symbolic value arguments as well as type arguments. The
operation constructs nested records locally, moves the resource into the
workspace, projects its backing array into a tensor, demands Contiguous/NoCopy/
NoOwnershipTransfer proofs, updates it, calls the evaluator, and exits through
ordinary lexical cleanup.

The consumer uses two resource types, a named comptime dimension, dimensions
1/2/3/8/16, and a FixedShape proof on a projected closed field. Native C witnesses
check the result and each individual resource's destruction count. The consumer
is exported and imported by a third module, which executes the same checks.

Eleven negative mutations of this same program cover copying an owned field,
copying the complete workspace, use after move, escaping a field-backed view,
missing and wrong-precision evaluator operations, a missing generic operation
on the numerical record, type-as-value arguments,
runtime dimensions, a mismatched nested dimension and zero dimensions.

The regression also checks deterministic open artifacts and all five generated
artifacts, closed field extents and defining-module identity after serialization,
and strict C11 native execution in Normal and Verify. The original small
`Fixed<N>` reproduction remains a separate directed test.

```powershell
go test ./internal/concept -run '^TestDependentGeneric' -count=1
go test -count=1 ./...
go vet ./...
```

Frozen Zig tests are skipped because the retired compiler and its build
infrastructure are unchanged. Nested whole-view tensor expression fusion remains
outside this change; the combined test uses existing indexed tensor semantics.
