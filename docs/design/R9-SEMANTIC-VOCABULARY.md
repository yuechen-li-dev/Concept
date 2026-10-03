# R9 semantic vocabulary research

Current: R9b3 qualification is complete; PlainData is admitted as an ordinary
restrictive concept. Other admissions are precisely scoped/deferred in
R9B-RESEARCH-LEDGER. The original R9b design below is historical. Its baseline:
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

## Historical limits of the R9b/R9b2 slice

PlainData and finite-domain admission, backend/Vulkan vocabulary adoption, and
additional innate rule migration remain unqualified. R9b2 qualifies the comptime
typed Verdict protocol, closed template calls and ownership refutation dogfood;
see R9-TYPED-VERDICTS and R9B2-CONFORMANCE. The ledger specifies proposed meanings
without granting their truth or optimizer authority. There is no second proof
system, source scanner, SMT solver, new cache or automatic Stage-0 freeze.


## R9b3: bounded semantic representation

Standard.Semantic.Vocabulary owns PlainData<T>. Its ordinary Concept predicate
first traverses checked representation structure and then obtains positive size
and alignment from the existing layout owner. PlainDataEvidence is two byte
quantities, not a proof tree. Records recurse through checked fields and fixed
arrays through their element type once; scalar and payload-free enum geometry
is accepted. Drop, ownership, references, managed storage and immovability carry
explicit typed refutations. Opaque foreign handles, partial/runtime storage,
payload sums, non-record aggregates, zero-size fields/values and unavailable
geometry remain Unknown. No reference or runtime object graph is traversed.

PlainData describes self-contained semantic representation, not merely fixed
byte layout. Fixed layout alone is insufficient when ownership, references,
Drop, or hidden runtime state remain meaningful. It is not CAbiValue,
Relocatable, Serializable, Zeroable or arbitrary-byte validity. CAbiValue and
Vulkan AsBytes retain their existing authority. The supported 64-bit geometry
model does not promise a portable wire encoding or canonical padding.

The new checked projections are HasFixedGeometry(typename),
RepresentationSize/RepresentationAlignment(typename) returning usize<byte>, and
CaseCount/CaseName/CaseTag/CasePayloadCount on enum declarations. They return
semantic observations, not PlainData/finite/closure facts. The same bounded
comptime evaluator owns every predicate and renderer. Geometry uses the existing
layout authority with per-query scratch memoization, checked arithmetic, depth
32 and 4096 graph accesses; this prevents overflow/cyclic/exponential observation
queries. PlainData's closed structural helper is bounded(16), sharing the usual
4096 fuel/depth 32 limits. Unknown never acquires a refutation payload.

Real consumers: Vulkan host Upload/Download require PlainData; AMD64 asserts it
for WireHeader, WireMachineFrame and WireMachineOperand in the ordinary checked
module. Assertions erase before runtime. Copy and wire codecs retain their
existing semantics. No old semantic authority is replaced; the new restriction
is additive. Positive evidence agrees with the existing geometry owner and C11
Header sizeof/alignment. The old compiler accepts opaque-handle Upload; the new
requirement explains the missing self-contained representation summary.

Relocatable remains deferred without a real compaction consumer or trusted
post-relocation invariants. Intrinsic RequiresStableAddress and contextual Pinned
remain separate; the immovable research observation proves only its exact
restriction. FiniteCases research covers enum tags, FiniteDomain only bool and
payload-free enums; complete payload values stay Unknown. Existing typed match
and codec inventories retain authority. ClosedWorld is contextual: equal real
activation topology IDs in two modules do not seal a compilation boundary.
Existing module-local activation layout remains unchanged. No globally closed
interface, new planner, runtime semantics or optimizer fact is installed.

The trusted-fact registry is still empty. User Proven results cannot forge any
real concept identity or grant a MIR fact. StaticExtent's exact 1D scope and
canonical Contiguous/Disjoint derivations remain unchanged. See R9B3-CONFORMANCE
and R9B-RESEARCH-LEDGER for all typed contracts, consumers and limits.
