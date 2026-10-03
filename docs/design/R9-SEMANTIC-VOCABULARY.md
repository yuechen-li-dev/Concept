# R9 semantic vocabulary research

This is a partial R9b qualification, not R9b closure. Baseline:
`2d32827cfe2afab05e6436e647aefab67b991eda` (current main after R9a2).
Compiler: `concept-evt1-stage0-go`. The live bridge remains CMIRAMD2.
See the research ledger for candidate admission and the convergence report for
the executable evidence-model boundary.

> Semantic concepts name properties the compiler can reason about. Concepts that
> merely reject programs have a lower trust requirement than facts that authorize
> removal of runtime mechanisms.

## Baseline authority

`SemanticFactCertainty` remains Proven / Disproven / Unknown. Proof graphs explain
requirements; optimizer-facing `MIRSemanticFact` values are separately qualified
by compiler-owned derivations. `predicate_requirements.go` evaluates ordinary
Concept predicates with Declared provenance. Its successful boolean return does
not append a qualified optimizer fact. Innate rules currently reject declarations;
their embedded normalized source hash participates in semantic module identity.
Innate authority therefore does not mean arbitrary codegen authority.

`Assert.Concept<Goal>(subject, "reason")` requires a nonempty reason, resolves
semantic subjects without executing runtime expressions, and reports distinct
disproven/unknown diagnostics. `concept explain file:line --verbose` exposes
the proof graph and its derivation origins. The illustrative request syntax
without a reason is not the current language syntax.

The tensor Planner consumes Contiguous, Bounded, FixedShape, Aligned, Disjoint,
NoAllocation, NoCopy and NoOwnershipTransfer. Vectorization remains eligibility
only, with `Selected=false`. LIR/native lowering consumes independently qualified
layout, bounds and effect evidence. No new AMD64 architecture is introduced here.

## Trust rule

Class A is descriptive/restrictive: a concept can be queried, explain a property,
and reject a program. Class B is codegen-authorizing: its evidence may justify
removal of a runtime mechanism. Truth is insufficient to cross this boundary.

> User-authored concepts cannot grant codegen-authorizing facts merely by
> returning true. Optimization-authorizing facts require trusted provenance.

For a future B path require a bounded registered derivation, exact subjects and
scope, checked origin, and identity of the rule and input context. CompilerAnalysis,
DerivedFromLayout and NativeToolchainProbe already exist. ClosedWorldDerivation
and InnateTrustedRule are proposed origins, not implemented aliases for Proven.
An innate fact grant would need an explicit audited rule, hash-covered identity,
visible provenance, and shadow agreement before switching consumers. NoAllocation,
Outlives and Disjoint have not been migrated into innate fact generation.

The first structural prototype is StaticExtent. It is derived from checked array
shape and cannot be produced by an ordinary predicate. The Planner records it
as shadow evidence; existing FixedShape/alias/bounds authorities still select
behavior. It does not authorize new check elimination or SIMD.

## StaticExtent: qualified bounded form

`StaticExtent<N>` means an exact one-dimensional fixed array contains N elements.
N is from 0 through the existing fixed-storage limit 1048576. The subject may be
the array type or a value retaining that exact array type. Multidimensional
shape is deliberately left to Rank/Shape. Raw/sparse capacity and runtime Span
length do not qualify; absent exact evidence stays Unknown. A known mismatched
array extent is Disproven. Evidence records N with DerivedFromLayout provenance.

Example:

```concept
using Array4 = int<array>[4];
Assert.Concept<StaticExtent<4>>(Array4, "exact array length");
```

This complements FixedShape (all dimensions fixed); it does not replace Shape
or claim that backing capacity equals a borrowed view's live length. Qualified
MIR facts expose exact inline vector extent to the existing tensor Planner.
The prototype deliberately excludes tensor views from its structural grant.

## Observation vocabulary decision

> compiler.* observations form a stable semantic interface between compiler
> mechanisms and Concept-written rules. They must not expose incidental Stage-0
> implementation structure.

The inventory is the checked `evt1Observations` map in comptime_subjects.go.
Declaration observations cover identity, provenance, parent/type, fields,
parameter/result types, ownership, attributes and template ownership. Type
observations cover Shape, names, element/declaration, Drop/NeedsDrop, partial
storage, ownership/borrowing/immovability and compatibility shape booleans.
No new observation has been added in this slice.

Prefer exhaustive TypeShape matches for mutually exclusive representation
categories. CValueProblem already does so and its Record/Struct helpers preserve
semantic ownership constraints separately. HasDrop, NeedsDrop, IsBorrowLike,
IsOwnedType, RuntimeShape and IsPartialStorage describe orthogonal constraints.
Retain IsStruct/IsEnum/IsArray/IsPointer/IsCallable/IsDyn/IsAsync compatibility
queries until each caller is migrated and qualified; no abrupt deletion or
claim of completed migration. New observations must be deterministic, bounded,
semantic and independent of Go implementation objects.

## Limits of this slice

PlainData and finite-domain admission, generic typed verdict predicate admission,
typed innate refutation dogfood, backend/Vulkan vocabulary adoption, and additional
innate rule migration remain unqualified. The ledger specifies proposed meanings
without granting their truth or optimizer authority. There is no second proof
system, source scanner, SMT solver, new cache or automatic Stage-0 freeze.
