# Concept EVT1 Stage 0 compiler architecture

Status: R4i inline tensor backing synthesis and rank-zero contraction lowering

## Authority

The Go compiler is the active Stage 0 implementation.

The compiler must remain capable of compiling Concept independently of Oct.
Oct/Prometheus is a consumer, not a build-time owner.

The retired Zig compiler remains independently runnable at
`legacy/poc3-zig`; it supplies reference behavior and migration pressure, not
new compiler authority.

## Pipeline

```text
Concept source
  -> lexer / tokens / one-based spans
  -> recursive parser
  -> typed Module / AST
  -> semantic validation and type environment
  -> bounded comptime evaluation
  -> concept closure and constrained template monomorphization
  -> typed deterministic MIR
  -> core C11 lowering plus selected profile admissions
  -> generated C/H
  -> source map and generation manifest
```

`internal/concept` is intentionally one coherent package in R0. Parser,
semantic, template, comptime, automata, MIR, and C-generation code share dense
private invariants inherited from the working compiler. Splitting them before
those boundaries stabilize would replace known coupling with exported coupling.

## Core stages

The following mechanisms are compiler-core responsibilities:

- tokenization with stable line and column provenance;
- recursive syntax parsing into typed declarations and expressions;
- type identity, declaration indexing, scope validation, and diagnostics;
- payload-enum construction and exhaustive match validation;
- local and parameter place mutability, record metadata, and bounded
  copyability/immovability validation;
- non-destructive record update typing and lowering;
- named concept requirements, constraint closure, and bounded template
  specialization;
- deterministic, fuel-bounded compile-time evaluation;
- typed array/ndarray rank, shape, contiguity, and row-major storage facts;
- MIR construction with source spans and stable ordering;
- artifact hashing, deterministic output ordering, and strict-C11 generation.

The general core API is `concept.Parse`, `concept.Generate`, `concept.Check`,
and `concept.Write`. Public representation names are neutral (`Module`, `Type`,
`MIR`, and declaration-specific types). The active compiler ID is
`concept-evt1-stage0-go`.

## Profiles and the Vulkan boundary

Every EVT1 source unit explicitly selects `profile Core;` or `profile Vulkan;`.
Core modules cannot admit domain imports, Vulkan runtime types, effects, or
actuators. This is enforced before ordinary semantic analysis.

R1 introduces a deliberately small data-owned `ProfileDefinition`:

```text
lexer / parser
    -> core semantic environment + selected ProfileDefinition
    -> validation
    -> typed MIR
    -> core C lowering + selected profile data/hooks
```

The definition registers builtin type identity, C spelling, required headers,
synthetic fields/declarations, compiler-owned profile enums, import markers,
and effect/actuator admission. The parser and validator no longer contain a
switch or prefix test for Vulkan builtin names, and the C lowerer no longer
contains Vulkan type-name cases.

`internal/concept/profile/vulkan` is the public in-repository Vulkan consumer
boundary. It admits only Vulkan modules, then delegates parsing, MIR, and C11
generation to the core. The tightly coupled effect/actuator validator remains
temporarily in `internal/concept/profile_vulkan.go`; the file is explicitly
marked as a profile implementation seam because it depends on private semantic
environment types.

Likewise, dense effect-batch and actuator C emission remains in the cohesive
lowerer behind profile admission. R1 records this as a profile lowering hook,
not as Core language law. Splitting it would require exporting unstable private
semantic state and is outside this isolation milestone.

Automata are not assigned to Vulkan by file location. Both implementation
lines contain machine-like semantics, but their surface and execution laws are
not yet reconciled. R0 therefore preserves the Go automata implementation as a
provisional core candidate and records the unresolved decision in the language
specification and matrix.

Vulkan C types, declarations, and includes are emitted only when a
Vulkan-profile module uses registered types. Core-profile proof tests reject
Vulkan vocabulary and assert that generated artifacts contain no Vulkan
coupling. Prometheus contributes only an admitted import marker; the compiler
has no Prometheus package, builtin type, or application binding dependency.

**Vulkan is an extension of Concept EVT1. Concept EVT1 is not an extension of
Vulkan.**

## R2 value/place implementation

R2 keeps these semantics in compiler core:

- `parse.go` recognizes `record struct`, binding `const`/typed `let`, and
  `WithExpr`, preserving modifier, keyword, field-name, and value spans;
- `types.go` stores record and binding metadata on the existing struct/local
  AST shapes and exposes `record` on MIR structs;
- `validate.go` owns the bounded place classification. Scope bindings carry
  mutability, lvalue projection propagates constness, record projections force
  a record-read-only reason, and existing recursive copyability supplies the
  ordinary/record copy law;
- the same validator owns immovable final-storage, whole-assignment,
  by-value boundary, struct/record embedding, and enum-payload checks;
- `WithExpr` validation requires a copyable record, resolves fields, rejects
  duplicates and type mismatches, and visits replacement expressions in
  source order;
- `comptime.go` performs value-map copy/update for bounded compile-time record
  expressions;
- MIR records a `record_with` semantic operation. `generate.go` evaluates the
  base once into a fresh C aggregate temporary and applies updates in source
  order.

The generated C struct representation is shared by ordinary, record, and
immovable structs. C mutation used to construct a fresh record is not exposed
as Concept-level mutability. C `const` is not relied upon for correctness; the
semantic validator is authoritative.

No R2 rule is registered in `ProfileDefinition` or implemented by the Vulkan
package. A Vulkan-positive corpus case proves that the profile inherits these
core semantics unchanged.

## R3 move, drop, and reference implementation

R3 remains entirely in compiler core:

- `parse.go` recognizes `move` and `ref` prefix expressions, `ref T` / `ref
  const T` types, and statement-form `if`, retaining keyword and operand spans;
- `types.go` represents move/reference expressions directly. The existing
  ownership field carries `owned`, compatibility `borrow`, and canonical `ref`
  rather than introducing a parallel type hierarchy;
- `validate.go` decides structural copyability, movability, immovability, and
  exact `void Drop(owned T)` witness matching. `owned T` is the bounded
  movable-only representation;
- scope bindings carry `Uninitialized`, `Initialized`, `Moved`, or
  `MaybeMoved` state. Whole-local/parameter moves update that state,
  reassignment restores a moved mutable owner, and cloned simple branch/loop
  scopes merge conservatively;
- call and return validation distinguishes fresh values, copyable reads, and
  explicit transfer from an existing owner. The type checker rejects implicit
  copies before C lowering;
- reference binding reuses the R2 lvalue classifier. It requires an existing
  place, checks const-to-mutable binding, preserves record-field read-only
  status, and allows immovable final storage to be mutated through `ref`;
- all reference returns are rejected in R3. This is an explicit bounded escape
  rule, not a lifetime inference framework;
- MIR operations expose `move`, `ref`, and `ref_const`. Function cleanup
  metadata lists each Drop owner in reverse declaration order with its witness
  and `live` or `transferred` state;
- `generate.go` lowers `ref T` to `T*`, `ref const T` to `const T*`, and
  materializes return expressions before emitting semantic cleanup calls.
  Moved sources are suppressed, by-value owned parameters are callee-owned,
  and the backend uses the validated Drop witness rather than inferring C
  destructors.

No R3 rule is registered in `ProfileDefinition`. The Vulkan-positive immovable
reference case passes through the same parser, validator, MIR, and C lowerer as
Core.

## R4a-R4b lifetime and semantic-requirement implementation

R4a extends the existing concept resolver rather than adding a template-local
lifetime subsystem:

```text
concept requirement
    -> requirement resolver
    -> operation / prerequisite concept / compiler analysis
    -> semantic proof result
```

`CompilerAnalysisRequirement` is the third requirement variant. A small
internal registry owns `LifetimeSafe`, `NonEscaping`, and `Outlives` with fixed
arities and check functions; it is not a plugin framework. Concept assertions
and template instantiation both pass through `checkConceptSatisfaction`, so a
lifetime proof runs because the selected concept requests it. Successful
proofs are retained in MIR as `semantic_proofs`; compiler-analysis
requirements remain compile-time-only and have no C representation.

R4b generalizes only the compiler-analysis argument boundary. A requirement
may name bounded semantic subjects while concepts and templates retain one
type parameter. `requires compiler.Outlives(source, result);` resolves
`source` to a named parameter and `result` to the result of exactly one
required operation. The internal subject record distinguishes `type`,
`parameter`, and `result`; parameter subjects carry a stable zero-based index
and all value subjects carry the selected function identity and type.

Selected result provenance follows this pipeline:

```text
function body
    -> syntax-directed result provenance summary
    -> call-site argument instantiation
    -> semantic relation request
    -> Outlives evaluator
    -> deterministic MIR proof evidence
```

`resultProvenance` is keyed by exact function signature. The bounded summary
forms are `static`, `parameter`, `shortest_of_parameters`, and `unknown`.
Direct parameter/reference return, ref-struct construction, nested ref-struct
construction, and acyclic direct pass-through calls are recognized. Recursive
cycles, ambiguous overloads, control-flow-dependent returns, and unsupported
expressions become `unknown`; there is no fixed-point region inference.

At a call site, parameter indices select actual argument provenance. Scoped
state from either the actual value or formal parameter is retained. Existing
R4a assignment/return checks then operate on that instantiated provenance, so
a helper call cannot extend an inner local or hide scoped state.

The `Outlives` evaluator is invoked only during concept satisfaction. It
returns `proven`, `disproven`, or `unknown`; only `proven` satisfies the
requirement. MIR function records expose `result_provenance`. MIR semantic
proofs expose deterministic identity, analysis name, relational subjects,
outcome, provenance facts, and requirement-origin span. None has a runtime
payload.

`Type` carries the `scoped` bit and existing ownership qualifier. `StructDecl`
and `MIRStruct` carry the `ref` aggregate classification. Semantic scope
bindings carry lexical provenance kind, depth, and scoped state. Construction
of a ref struct derives the shortest bound from reference/ref-struct fields;
assignment and return compare that bound against the destination. Unknown
provenance is rejected conservatively. Unrestricted structs and enum payloads
cannot contain references or ref structs.

The C backend lowers ref structs as ordinary aggregate values and their
reference fields as pointers. No lifetime metadata is emitted at runtime.
Scoped parameters use the same pointer ABI as references after semantic
validation.

R4a also validates MIR cleanup obligations before artifact generation. Each
recorded Drop owner must be unique, ordered, and either live or transferred.
Live `owned T` replacement emits the old Drop call before evaluation/assignment
of the new initializer and retains one final cleanup obligation for the new
value.

## CLI and artifacts

`cmd/concept` is the active driver:

```console
go run ./cmd/concept --help
go run ./cmd/concept check examples/evt1/core_language.concept
go run ./cmd/concept mir examples/evt1/core_language.concept
go run ./cmd/concept emit-c examples/evt1/core_language.concept
```

`check` exercises lexing, parsing, and semantic validation. `mir` and `emit-c`
exercise the full generation path and select the corresponding deterministic
artifact for standard output. No command is a placeholder.

Generated evidence uses these identities:

- MIR schema: `concept-evt1-mir.v1`
- source-map schema: `concept-evt1-source-map.v1`
- manifest schema: `concept-evt1-generation-manifest.v1`
- compiler: `concept-evt1-stage0-go`

Golden artifacts are retained only where the extracted tests perform exact
drift checks.

## Determinism and validation boundaries

Source order is preserved where semantically visible. Generated maps and MIR
retain source spans. Template instance identity, automata topology, effects,
and output filenames are derived deterministically. Manifests contain source
and output hashes and forbid timestamps.

The C backend is an implementation boundary, not a definition of language
semantics. Profile lowering may add admitted runtime mappings, but it may not
silently broaden core parsing or typing. A profile-specific construct must be
rejected outside its profile or explicitly promoted through the specification
and reconciliation process.

## R4c failure lowering

```text
ordinary payload-enum semantics
  -> compiler-known Option<T> / Result<T,E> specialization
  -> explicit match or sugar (? / ! / try / except)
  -> ordinary MIR tag branches, extraction, return, local goto, or panic
  -> deterministic strict-C11 tagged aggregate lowering
```

The semantic checker supplies specialized payload fields to the existing
match validator. It also validates exact return channels and exact error-type
handler selection. MIR retains `option_propagate`, `result_propagate`,
`option_unroll`, `result_unroll`, `try_handler`, and `assert` provenance while
the backend reuses ordinary tag/payload control flow. There are no runtime
exception tables, SEH, `setjmp`/`longjmp`, personality functions, or unwinding.

```text
assert        -> runtime Assert.True -> terminal concept_panic on false
static_assert -> comptime Assert.True -> no runtime code
```

Generic carriers recursively inherit copyability, transfer, immovable, and
lifetime-bound status from their payloads. `?` and match extraction preserve
the R4b provenance associated with the carrier; they do not create a new
static lifetime. Carriers with droppable payloads synthesize a deterministic
tag-dispatching cleanup that invokes ordinary `Drop` only for the active
variant.

## R4d storage lowering

```text
source array / ndarray
  -> typed StorageKind + element + rank + shape
  -> MIR storage type and index/query operations
  -> fixed C wrapper with contiguous data[N0 * ... * Nk]
  -> guarded row-major scalar projection
```

`array` and `ndarray` share ordinary value/place analysis but retain distinct
storage identities. The type record carries `StorageKind`, ordered dimensions,
the fixed/runtime status of each extent, contiguity, and row-major layout.
`MIR.storage_types` repeats these facts explicitly with the storage ownership
category (`fixed_inline` for the executable R4d subset). `array_index`,
`ndarray_index`, `rank_query`, and `shape_query` operations retain intent and
source spans; ndarray is never rewritten into nested array types.

The C backend dependency-orders fixed storage wrappers with user structs and
enums, then declares failure carriers and function prototypes. Wrapper assignment supplies Concept
copy semantics without C array decay. Ndarray literals flatten in source order,
and comma-separated indices lower to a checked row-major offset. Runtime shape
expressions are typed as value-level shape facts, but a local with such a shape
is rejected before lowering because no explicit storage constructor exists.
Fixed storage with droppable elements lowers cleanup in reverse linear order.
There is no `malloc`, allocator selection, variable-length C array, descriptor
fabrication, or hidden runtime in this path.

R4d adds no TensorIR. Span/ReadOnlySpan, Slice, FixedBuffer, vector, matrix,
tensor, alternative layouts, alias analysis, vectorization, GPU lowering, and
allocator integration remain separate milestones.

## R4e storage-binding lowering

```text
source contiguous array / ndarray storage
  -> unary bind with an expected ref/ref const storage target
  -> exact element/count, const, and provenance validation
  -> MIR bind_storage with target rank/shape and proof mode
  -> non-owning C descriptor over the original data member
  -> existing guarded array/ndarray indexing and shape queries
```

`bind_storage` remains explicit in MIR. It records source and target storage
kinds, target rank and ordered shape, fixed proof or overflow-safe runtime
equality check, mutability, provenance, and the facts `no_copy`,
`no_allocation`, and `no_ownership_transfer`. MIR validation rejects an
operation missing those storage or provenance facts.

The C representation of a bound view is a local descriptor containing an
element pointer and an inline `size_t shape[rank]`. Const targets use a pointer
to const elements. Runtime shape expressions are evaluated once; checked
multiplication and exact source-count equality precede descriptor construction.
Fixed extents may be constant-folded. Descriptor assignment never copies
backing elements, and no `malloc`, VLA, metadata allocation, or ownership hook
is emitted. Both directions continue through the original wrapper's `data`
member, so source and view observe the same bytes in the same linear order.

Bind provenance is derived from the source expression using the existing
R4a/R4b representation. Scoped state survives, function-result summaries may
select a bound source parameter, and ref-struct lifetime derivation remains
unchanged. R4e does not introduce a second view, borrow, or indexing system.

## R4f semantic layout and stream lowering

```text
layout declaration -> fixed geometry/disjointness -> MIR layout graph
fixed storage + ref Layout -> layout_bind -> same-backing descriptor
stream over Layout -> checked channel map -> zero-storage MIR stream graph
bound layout + ref Stream -> stream_bind -> channel region_projection
```

Semantic analysis computes canonical byte size, alignment, offsets, extents,
stable identities, and pairwise disjointness. Runtime-dependent geometry is
rejected before MIR. Closed layout queries execute in the bounded comptime
evaluator rather than through runtime reflection.

Layout bind validates whole-storage byte size, the executable inline alignment
guarantee, constness, and source provenance. Stream bind validates declared
layout identity and reuses that provenance. `layout_bind`, `stream_bind`, and
`region_projection` remain explicit in MIR with no-copy, no-allocation,
no-transfer, and same-backing facts; MIR validation rejects inconsistency.

Strict C11 emits typed pointer descriptors and constant offset arithmetic over
the original fixed wrapper. Array/ndarray regions reuse the existing view and
row-major indexing. Layout-bearing modules align fixed wrappers to 64 bytes;
stronger requirements reject until explicit aligned storage exists. Streams
emit no backing storage or dispatch. This is backend evidence, not stable ABI.

## R4g bounded-span lowering

```text
known contiguous storage or R4f region/channel
  -> Span / ReadOnlySpan construction
  -> interval and element-identity validation
  -> existing lexical/call-result provenance + stable parent-region identity
  -> Subspan narrowing over the same half-open parent interval
  -> explicit MIR borrowed-region facts
  -> guarded pointer-plus-length strict-C11 descriptor
```

`Span<T>` and `ReadOnlySpan<T>` reuse the applied-type architecture and are
recognized as compiler-known ref-struct-like descriptors. Scope bindings carry
the same lifetime provenance used by R4a/R4b plus a compact region payload:
element type, stable parent identity, backing byte offset, relative element
offset, length, byte extent, safe alignment, mutability, and contiguity. Bound
storage descriptors retain their backing identity; layout and stream
projections consume the R4f region facts directly.

MIR keeps `span_from_region`, `span_to_readonly`, `span_subregion`, and
`span_index` operations until validation. Each operation must retain region,
provenance, interval, alignment, mutability, contiguity, same-backing, and
no-copy/no-allocation/no-transfer facts. Subregions additionally require the
overflow-safe half-open bounds proof. Malformed operations reject before C
generation.

The C bootstrap representation is a typed pointer and `size_t length`.
Readonly pointers are const-qualified. Construction points at existing
storage, Subspan adjusts that pointer only after guarded bounds and byte-offset
checks, and indexing guards before dereference. There is no allocation,
backing copy, ownership hook, or Drop obligation. The runtime descriptor is a
lowering detail; the richer MIR facts remain compiler authority.

## R4h tensor semantic MIR and lowering

R4h adds one mathematical consumer of the established storage/region facts:

```text
array / ndarray / bound storage / layout region / stream channel / rank-1 Span
  -> Tensor(source), preserving shape + region + provenance + alignment
  -> Tensor semantic MIR
  -> exact-shape, symbolic-index, contraction, mutability, and alias validation
  -> dedicated tensor lowering
  -> explicit zero-based loop/reduction code
  -> existing strict-C11 backend
```

The parser records `@` and indexed expressions; it does not synthesize loops.
Semantic analysis creates statement-local symbolic index scopes and attaches a
validated `TensorSemantic` plan to tensor assignments. `MIRFunction` exposes
those plans separately as `tensor_operations`, while ordinary MIR operations
retain `tensor_view`, `tensor_index`, and `tensor_symbolic_index` facts. MIR
validation requires rank, shape, region, provenance, mutability, alignment,
alias policy, and the selected lowering stage before code generation.

The C representation is a typed pointer plus a fixed-rank `size_t shape[]`
descriptor. Construction points at existing storage. Elementwise lowering uses
one checked linear loop. Default `@` and Einstein contraction use explicit
free-index loops, an arithmetic-zero accumulator, and reduction loops.
Runtime shape products and extent equalities are guarded deterministically.
No parser-time loop rewriting, heap temporary, tensor runtime, BLAS, MLIR,
SIMD, or GPU kernel path is present.

Fixed symbolic initialization reuses the same validated TensorSemantic plan in
the bounded comptime evaluator. The evaluator mutates the aliased fixed-array
value and therefore proves that symbolic indexing removes manual nested source
loops without creating persistent hidden state.

## R4h boundary inherited by R4i

- Source-level tensor ranks remain positive. R4i admits rank zero only as the
  internal scalar result of vector dot product.
- R4i closes the R4h shorthand deferral by normalizing `vector<T>` and
  `matrix<T>` exactly to `tensor<T,1>` and `tensor<T,2>`.
- The Einstein subset accepts unambiguous multiplication-sum reductions and
  fixed symbolic initialization; explicit index-declaration syntax and
  noncanonical multi-axis `Contract` syntax remain future work.
- Tensor sources are contiguous and shape-aware. Arbitrary strides, slicing,
  sparse/tiled storage, named axes, broadcasting, autograd, and allocator-owned
  dynamic tensors are absent.

## R4i shaped declarations and aliases

R4i adds one front-end normalization path above the existing R4h machinery:

```text
tensor<T> name[fixed shape] / vector<T> / matrix<T>
  -> derive and enforce rank
  -> synthesize fixed ndarray backing with stable source identity
  -> validate ndarray literal or exact scalar fill
  -> construct ordinary tensor semantic view
  -> existing Tensor MIR
  -> existing tensor-to-loop and strict-C11 lowering
```

The AST retains `InlineTensorDecl` source metadata for diagnostics and MIR
provenance. Semantic analysis resolves its shape through the ordinary fixed
storage resolver, rejects runtime extents, records an `Inline` backing witness,
and gives every declaration a deterministic region identity. Ordinary MIR
contains `tensor_inline_storage` followed by `tensor_view`; the latter retains
same-region/no-copy/no-allocation/no-transfer facts. Independent identities
make ordinary inline destinations disjoint from their operands.

Backing origins use the closed `TensorBackingKind` classification: `Inline`,
`NDArray`, `BoundNDArray`, `Span`, `LayoutRegion`, and `StreamChannel`. These
are compile-time facts, not runtime tags or user-facing matching syntax.
`vector<T>` and `matrix<T>` normalize to rank-one/rank-two tensor types in the
parser and never reach MIR as distinct types.

The C backend emits the existing fixed ndarray wrapper as the synthesized
local, initializes its flat data from the reused literal path or one evaluated
fill scalar, and constructs the existing tensor pointer/shape descriptor over
that local. No allocator, heap ownership, tensor runtime, or backing copy is
introduced.

Rank-one `@` validation produces `tensor_scalar_contract` with a rank-zero MIR
output and two rank-one operands. The backend lowers it directly to one scalar
accumulator and reduction loop. Source-level rank-zero tensor variables remain
absent.

## R4j semantic fact qualification

R4j extends the existing analysis registry and MIR proof objects with one typed
fact model:

```text
storage/layout/view/tensor semantics
    -> semantic fact qualification
    -> fact/proof registry
    -> concept requirement resolver
    -> MIR semantic facts
    -> future optimizer / LIR
```

`SemanticFactKind`, typed subjects, integer parameters,
`SemanticFactCertainty`, origins, and bounded evidence are shared by concept
proofs and MIR. Qualification reads the existing `LayoutRegion`, span interval,
`TensorViewFacts`, and closed `TensorBackingKind` witnesses. It does not infer a
second representation. Proof/fact IDs use stable source and semantic identity,
never runtime addresses.

The concept registry accepts unary type guarantees, parameterized alignment
and rank, and bounded relational subjects. Unknown is a first-class outcome
and never satisfies a requirement. MIR validation checks fact identity,
subjects, origin, and certainty before lowering.

`SemanticFactSet` is the backend-independent consumer surface: `FactsFor`,
`Prove`, `KnownAlignment`, `RegionOf`, and `AreDisjoint`. Future optimization
passes consume these retained facts instead of re-running validation. R4j adds
no optimizer, SIMD, noalias syntax, runtime tables, or C metadata.

## R4k class, interface, and dyn path

```text
class declaration
    -> aggregate member/access table
    -> method receiver normalization
    -> ordinary value/storage analysis
    -> direct strict-C11 struct and functions

interface declaration
    -> ordinary concept requirements
    -> fixed-shape dyn compatibility validation
    -> ordinary satisfaction proof
    -> deterministic static witness description

concrete ref -> dyn Interface
    -> validate satisfaction, constness, and provenance
    -> reify (interface, concrete type) witness
    -> erased object pointer + witness pointer
    -> dyn method/field dispatch
```

Classes reuse aggregate layout, copy/move/Drop, reference, and C lowering. The
parser hoists their methods as ordinary functions with `MethodOf`, visibility,
and an explicit first receiver; semantic lookup enforces private access before
lowering. No class-specific allocator or runtime exists.

Interfaces reuse the concept resolver. Runtime witness MIR records interface
and concrete identities, method/field entries, prerequisite links, and the
no-allocation invariant. MIR validation rejects missing/duplicate identities,
runtime entries, or allocation-law metadata. C lowering emits one static const
witness table per used specialization, adapter functions, and a two-pointer dyn
value. Semantic compiler facts remain compile-time proofs and add no runtime
table fields.

## General limitations after R4k

- The package remains deliberately cohesive rather than prematurely split.
- Effect/actuator validation and C runtime emission remain in-package profile
  hooks; their complete physical extraction is deferred until the private
  semantic boundary stabilizes.
- Diagnostic codes retain the historical `CV` namespace for compatibility;
  R2 maps its bounded slice to semantic family names and defers renumbering.
- The backend is the extracted strict-C11 path only.
- Imports are represented, but Core multi-module compilation is not yet active.
- Automata are implemented but provisional; PoC3 machine/decide/yield laws are
  not merged.
- Ownership beyond whole-local explicit transfer, deterministic local/
  parameter cleanup, and live whole-owner replacement remains deferred. There
  is no implicit move, field move, partial drop, unwinding, or dynamic cleanup
  stack.
- Named lifetimes, generalized borrow checking, non-lexical lifetimes, mutable
  alias analysis, and unrestricted reference-containing
  aggregates are not implemented.
- The remaining PoC3 allocation, stable C ABI, testing framework, owning dyn,
  slices, and FixedBuffer surfaces are not ported. R4f has no
  `throw`, unwinding, Option handler arm, implicit error conversion, or
  generalized panic runtime.

R4b does not add a generalized place lattice or prove global alias/lifetime
safety. `borrow` compatibility behavior remains narrower than the canonical
explicit `ref` spelling.

## R1 executable evidence

`internal/concept/conformance_test.go` and `language/evt1-r1/core` provide the
bounded differential harness and canonical source corpus. The evidence covers
structs, payload enums, exhaustive match, ordinary control flow, concepts,
templates, deterministic monomorphization, bounded comptime, and compile-time
fixed arrays. `docs/conformance/EVT1-R1-CONFORMANCE.md` records classifications
and provenance; `docs/compiler/EVT1-R1-VULKAN-ISOLATION.md` records the complete
R1 leakage audit.

## R2 executable evidence

`internal/concept/r2_conformance_test.go` and `language/evt1-r2/core` provide
32 readable cases with semantic-family diagnostics, MIR checks, exactly-once
base lowering proof, Core/Vulkan inheritance evidence, and native C11 runs for
mutable struct, record update, and immovable final-storage paths.

## R3 executable evidence

`internal/concept/r3_conformance_test.go` and `language/evt1-r3/core` cover 31
readable cases: 30 PASS and one retained ordinary-struct-copy
EXPECTED-DIVERGENCE. Evidence includes moved-state diagnostics and joins,
call/return transfer, MIR cleanup ownership, generated-C ordering and
suppression, mutable and const references, record/const interaction,
immovable-by-reference use, Vulkan inheritance, and native C11 execution.

## R4a executable evidence

`internal/concept/r4a_conformance_test.go` and `language/evt1-r4a/core`
provide six valid and five invalid readable cases. They cover local and
downward-passed ref structs, scoped calls, concept-triggered lifetime proof,
local/ref-struct/scoped/outlives rejection, unrestricted aggregate rejection,
live owned replacement ordering, MIR proof metadata, and native C11 execution.

## R4b executable evidence

`internal/concept/r4b_conformance_test.go` and `language/evt1-r4b/core`
provide six valid and five invalid readable cases. They cover direct and
ref-struct call results, nested and multi-source summaries, acyclic
pass-through, scoped propagation, longer-place rejection, relational
pass/fail/unknown outcomes, deterministic proof identity, MIR inspection, and
three native C11 executions.

## R4c executable evidence

`internal/concept/r4c_conformance_test.go` and `language/evt1-r4c/core`
provide 24 readable cases: 14 valid and 10 invalid, all classified PASS.
Evidence includes deterministic MIR sugar operations, nine successful native C11 paths,
exact typed local handlers, owned success transfer, immovable rejection, and a
valid/invalid pair proving Result success payloads do not launder R4b lifetime
provenance.

## R4d executable evidence

`internal/concept/r4d_conformance_test.go` and `language/evt1-r4d/core`
provide 30 readable cases: 17 valid and 13 invalid, all classified PASS.
The suite inspects rank/shape/contiguity/layout/ownership MIR, compiles and runs
16 strict-C11 value paths, and executes a separate native out-of-bounds process
to prove the deterministic terminal panic reason. PoC3 Phase 21 supplies array
value/index/bounds/wrapper evidence; ndarray and its rank-aware row-major
surface are EVT1-new storage semantics.

## R4e executable evidence

`internal/concept/r4e_conformance_test.go` and `language/evt1-r4e/core`
provide 23 readable cases: 12 valid and 11 invalid, all classified PASS. The
suite inspects explicit bind MIR and generated non-owning descriptors, executes
12 successful strict-C11 alias/shape/provenance compositions, and executes two
terminal processes for exact-count mismatch and checked-product overflow.
PoC3 supplies no bind counterpart; R4e composes its wrapper evidence with the
active R4a/R4b provenance model and R4d row-major storage model.

## R4f executable evidence

`internal/concept/r4f_conformance_test.go` and `language/evt1-r4f/core`
provide 29 readable cases: 16 valid and 13 invalid, all classified PASS. The
suite inspects semantic layout/stream graphs, bind/projection MIR, and forbidden
runtime mechanisms. Strict-C11 harnesses execute scalar, array/ndarray,
alignment, mutable/const alias, and lexical/call-result/scoped provenance paths.
The conformance record distinguishes SDSL-derived semantic structure from the
EVT1-new surface and ordinary Go-compiler lowering.

## R4g executable evidence

`internal/concept/r4g_conformance_test.go` and `language/evt1-r4g/core`
provide 29 readable cases: 15 valid and 14 invalid-path programs, all
classified PASS. Runtime-failure sources remain in the invalid-path directory
but compile successfully and are executed in isolated terminal-process tests.
The suite inspects explicit borrowed-region MIR, compiles and runs 15 strict-C11
success paths, executes runtime Subspan and index failures, and checks the
generated path for pointer-plus-length descriptors with no allocation or
backing copy.

## R4h executable evidence

`internal/concept/r4h_conformance_test.go` and `language/evt1-r4h/core`
provide 29 readable conformance cases: 17 valid and 12 statically rejected,
all classified `PASS`. The suite separately checks Tensor MIR validation,
15 successful strict-C11 numeric paths, runtime index/contraction guards, and
bounded comptime symbolic initialization. Generated evidence contains no heap,
copy helper, BLAS, MLIR, SIMD, or GPU tensor path.

## R4i executable evidence

`internal/concept/r4i_conformance_test.go` and `language/evt1-r4i/core`
provide 27 readable conformance cases: 16 valid and 11 statically rejected,
all classified `PASS`. The suite checks shaped-declaration rank inference,
all six backing classifications, stable/disjoint inline regions, exact alias
identity, rank-zero Tensor MIR, malformed inline MIR rejection, default and
const laws, and 12 strict-C11 numeric paths. Generated evidence contains no
heap, backing-copy, BLAS, MLIR, or separate vector/matrix path.

## R4j executable evidence

`internal/concept/r4j_conformance_test.go` and `language/evt1-r4j/core`
provide 24 readable cases: 17 valid and 7 statically rejected, all classified
`PASS`. The suite checks typed concept proofs, deterministic fact identities,
malformed fact rejection, declared and interval disjointness, fixed/runtime
tensor shapes, inline backing independence, safe alignment degradation, and
the layout-to-stream-to-Span-to-tensor preservation chain. Generated C is
checked to contain no runtime fact tables, allocation, noalias, or SIMD path.

## R4k executable evidence

`internal/concept/r4k_conformance_test.go` and `language/evt1-r4k/core`
provide 30 required readable cases: 16 accepted and 14 statically rejected,
all classified `PASS`. The suite additionally validates malformed witness MIR,
inspects direct/dynamic MIR and static witness C shape, and executes strict-C11
class, private-member, struct/class dyn dispatch, field get/set, composition,
immovable-reference, and scoped-provenance paths. Generated evidence contains
no allocation, per-object vtable, RTTI, object registry, or GC path.
