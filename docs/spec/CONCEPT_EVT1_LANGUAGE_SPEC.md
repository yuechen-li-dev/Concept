# Concept EVT1 language specification foundation

Status: R4h first-class tensor semantics and symbolic contraction

This document defines the authority categories and the smallest currently
executable EVT1 language foundation. It is derived from the retired Concept
PoC3/Zig implementation and the extracted Concept/Vulkan/Go implementation.
Where they disagree, this document records a direction or defers the decision;
it does not treat implementation accident as language law.

The classifications used throughout are:

- **Canonical EVT1**: selected language law with an active Go proof.
- **Provisional EVT1**: implemented or strongly indicated, but not yet closed
  against both implementation lines.
- **Profile-specific**: admitted by a named profile and not general core.
- **Legacy PoC3**: preserved behavior with no current EVT1 promise.
- **Deferred reconciliation**: an open cross-line design decision.

## 1. Language identity

**Canonical EVT1.** Concept is a C++-lineage systems language for auditable
compiler, runtime, native, and bare-metal work. The canonical bootstrap line is
Concept EVT1 Stage 0 implemented in Go, compiler ID
`concept-evt1-stage0-go`. Concept/Vulkan is a source and a profile, not the
language identity. PoC3/Zig is a retired reference implementation.

## 2. Design laws

**Canonical EVT1.** Essential difficulty remains visible; accidental difficulty
is removed or quarantined; hazardous operations require deliberate spelling;
compile-time behavior is deterministic and bounded; generated authority is
inspectable; profiles may admit domain semantics but may not silently change
core meaning. No hidden heap, scheduler, lifetime calculus, or ambient effect
system is implied by syntax.

## 3. Source files and modules

**Canonical EVT1 foundation.** Source uses `.concept`. An EVT1 source unit begins
with exactly one explicit profile declaration:

```concept
profile Core;
```

or:

```concept
profile Vulkan;
```

**Provisional EVT1.** Dotted imports are represented after the profile line.
Core multi-module resolution is not implemented, and Core currently rejects
domain imports. PoC3 module/import fixtures remain the authority pressure for a
later multi-module decision.

## 4. Naming and formatting

**Canonical EVT1.** User-facing types, functions, and compiler-known operations
use `PascalCase`; parameters and locals use `camelCase`. Declarations follow the
C++ lineage:

```concept
Result<ProbeEvidence, PrometheusError> Execute(
    borrow MechanismContext context,
    unsafe imported borrow AccelerationStructure admittedTlas)
{
    // ...
}
```

Return type precedes function name. EVT1 does not use `fn`, `name: Type`,
`-> ReturnType`, inferred `let`, or `var`. R2 accepts `let Type name = value;`
only as an exact alias of `const Type name = value;`; `const` is canonical and
documentation should prefer it. Braces and indentation remain human-readable,
with one statement per line. This resolves a material PoC3 surface conflict in
favor of the newer Concept/Vulkan direction; PoC3 syntax remains accepted only
by the retired compiler until translated.

## 5. Declarations

**Canonical EVT1 foundation.** R2 parses named structs, record structs, immovable structs,
enums, concepts, constrained function templates, compile-time declarations,
static assertions, ordinary functions, and profile-admitted declarations.
Declarations use explicit types and semicolon-terminated fields/statements.

**Deferred reconciliation.** Visibility, namespaces, complete module exports,
attributes, overload/coherence breadth, and declaration ordering beyond the
implemented subset are not fixed here.

## 6. Primitive types

**Canonical EVT1 foundation.** `int`, `uint`, `byte`, `float`, `bool`, and
`void` are core primitive types in the implemented subset. R4f fixes bootstrap
geometry for semantic layouts as `byte` as 8-bit/alignment 1, `int`, `uint`, and
`float` as 32-bit/alignment 4, and `uint64` as 64-bit/alignment 8. These are
Concept layout facts; they do not claim a general foreign ABI mapping.

**Provisional EVT1.** `string` is available to bounded compile-time evaluation
and diagnostics but does not establish a general runtime string model.
`uint64` exists in the seed implementation but awaits cross-line primitive and
width rules.

**Profile-specific.** `PipelineLayout`, `Pipeline`, `VulkanError`, `VkBuffer`,
and `VkCommandPool` are Vulkan admissions, not core primitives.

## 7. Ownership and storage vocabulary

**Canonical EVT1 R2-R3.** In a local declaration, `const` qualifies the binding
and its projected places, not the value type. A const local can be read and a
copyable value can be copied from it, but the local cannot be reassigned and a
mutable struct field cannot be written through it. R2 also applies the same
place rule to the already-natural `const Type parameter` qualifier. This is a
parameter-place rule, not a full borrowing model.

**Canonical EVT1 R3.** `owned T` is the bounded movable-only representation.
It owns a `T`, is not copyable, is movable when `T` is not immovable, and may
carry deterministic drop responsibility. This reuses the existing ownership
vocabulary without adding a trait or derive system. Unqualified ordinary
values retain structural copyability. `borrow` remains accepted compatibility
vocabulary; `ref` is the canonical R3 reference spelling.

**Deferred reconciliation.** Allocation/store ownership, partial moves,
general borrow checking, and named lifetime parameters are not implied by
R4a. Reference-containing aggregates are admitted only through the bounded
`ref struct` rules below.

## 8. Values, structs, records, and places

**Canonical EVT1 R2.** `struct Name { Type field; }` declares an ordinary
mutable value type. Fields retain declaration order. Positional construction,
field reads, field writes through mutable places, whole-value assignment, and
copying are supported when every contained field is copyable. A struct that
contains a non-copyable field is itself non-copyable.

`record struct Name { Type field; }` declares an immutable value type using
the same declaration-ordered aggregate representation. Record fields are
readable but never user-assignable, even through a non-const binding. A record
is copyable exactly when all fields are copyable. This is type-level field
immutability; it is distinct from binding/place immutability supplied by
`const`. A non-const record binding may therefore be rebound as a whole to a
new copyable record value even though none of its field projections is mutable.

The bounded R2 place classes are:

- mutable local place;
- const local or const parameter place;
- record-field read-only projection;
- immovable final-storage place;
- temporary/rvalue, which is not assignable.

A mutable struct field projected from a mutable place remains mutable. The
same field projected from a const place is read-only. Every record-field
projection is read-only. An immovable local's fields may be mutated in its
final storage, but its whole value is never copy-assignable. This is not the
future ownership/place lattice and establishes no general alias or borrow law.

### 8.1 Non-destructive record update

`base with { field = value; }` evaluates `base` exactly once, requires a
copyable record value, copies its fields, evaluates replacements in source
order, and produces a fresh record value. Unspecified fields retain their
source values and the source is unchanged. Fields must exist, may appear only
once, and replacement types must match. R2 rejects `with` on ordinary structs,
non-record values, and records containing a non-copyable field. Dotted update
targets are not syntax; nested updates compose as nested `with` expressions.

The MIR records `record_with` aggregate-copy/update intent. The C backend may
construct the fresh value through a temporary and field assignments; those
assignments are compiler-generated construction and do not make record fields
mutable in Concept.

### 8.2 Immovable structs

**Canonical EVT1 R2 subset.** `immovable struct` declares mutable,
storage-bound data. It may be constructed directly in final local storage and
its fields may be mutated through that mutable place. Whole-value copy and
assignment, pass-by-value, return-by-value, struct or record embedding by
value, and enum payload embedding by value are rejected. Borrow/reference
semantics beyond the already-implemented bounded subset remain deferred; no
implicit relocation is permitted.

### 8.3 Structural equality

The existing bounded compile-time structural equality implementation applies
to record structs when every field supports equality. R2 adds no generated
runtime operators.

### 8.4 Move state and deterministic drop

**Canonical EVT1 R3.** Copyability is structural, while movability is broader.
An ordinary copyable value may be assigned or passed by value without consuming
its source. `move place` is semantically unnecessary but legal for such a value
and does not make the source moved-from.

An `owned T` place is movable-only when `T` is movable. Transferring it from an
existing owner requires explicit `move`:

```concept
owned Resource second = move first;
```

The source becomes moved-from. Reading, projecting, passing, returning, or
moving it again is invalid until legal whole-place reassignment initializes it
again. Straight-line state and simple `if`/`while` joins use the bounded states
`Uninitialized`, `Initialized`, `Moved`, and `MaybeMoved`; a value initialized
on one path and moved on another is not usable after the join. R3 has no partial
or field-level moves.

An immovable value cannot be relocated, including with `move`, but remains
usable in final storage and through references.

`void Drop(owned T value)` is the narrow R3 drop witness. A live owning local
or by-value owning parameter with that witness is dropped exactly once at scope
exit, including early return, unless ownership was transferred. Return values
are evaluated before cleanup. Cleanup proceeds in reverse declaration order.
The MIR records owner, witness, order, and whether an owner is live or
transferred; the C backend emits only those deterministic cleanup calls. R4a
validates those obligations as MIR invariants: every recorded owner has one
ordered obligation and is either live (drop exactly once) or transferred (no
drop). A `MaybeMoved` Drop owner remains rejected because conditional cleanup
is not implemented. Assigning a fresh or explicitly transferred value to a
live `owned T` local performs `Drop` on the old value and then initializes the
same storage with the new value. R4a does not define unwinding, partial-field
drop, or dynamic cleanup stacks.

### 8.5 References

**Canonical EVT1 R3.** `ref T` aliases an existing mutable `T` place; `ref const
T` aliases an existing place as a read-only view. References do not copy or own
the referent and do not transfer ownership. Binding is explicit at the call or
local initializer site:

```concept
Increment(ref count);
Inspect(ref const widget);
```

The bounded binding rules are:

- mutable place to `ref T`: allowed;
- mutable place to `ref const T`: allowed;
- const place to `ref const T`: allowed;
- const place to `ref T`: rejected;
- temporary/rvalue to either reference form: rejected.

No temporary lifetime extension occurs. Mutation through `ref const` is
rejected. Record fields remain immutable through both reference forms because
record immutability is type-level. A mutable immovable final-storage value may
bind to `ref T`; this is the normal R3 way to manipulate it without relocation.
Borrowing an `owned T` does not change its ownership state, so it may later be
transferred with `move`.

**Canonical EVT1 R4a.** References carry bounded compile-time provenance:
`Local(scope)`, `Parameter(scope)`, static/global where supported, or
`Unknown`. Lexical checks reject a local reference returned from its function
and reject assignment of a shorter-lived reference into a longer-lived slot.
A reference derived from a non-scoped parameter may be returned when this
bounded provenance proves the relationship; unknown cases are rejected.

`ref struct Name { ... }` declares a value type that may contain `ref T`,
`ref const T`, or another ref struct. Its lifetime bound is the shortest-lived
referenced field. Copying the aggregate does not extend that bound, and the C11
representation remains an ordinary aggregate with pointer fields: no hidden
allocation or runtime lifetime object exists. An unrestricted struct or enum
payload may not embed a reference or ref struct. A ref struct may be passed by
value downward, but returning or assigning it outward requires the same
provenance proof as a reference.

`scoped ref T` and `scoped ref const T` state that the reference must not
escape the current semantic scope. Scoped parameters may be used and passed
downward, but cannot be returned or embedded into unrestricted storage. R4a
adds no named lifetimes, non-lexical lifetime inference, reborrow lattice, or
general mutable-alias analysis.

**Canonical EVT1 R4b.** A function returning a reference or ref struct has a
compile-time result-provenance summary when its body proves one by a bounded,
syntax-directed rule. The supported summaries are `Parameter(index)`,
`ShortestOfParameters(indices)`, and `Unknown`; `Static` is reserved for a
statically rooted result. Direct parameter returns, reference/ref-struct
construction, nested ref-struct construction, and direct pass-through calls to
an already summarizable function participate. Branch-sensitive unions,
callbacks, recursive inference, and arbitrary interprocedural alias analysis
do not.

At a call site, a parameter summary is instantiated with the corresponding
argument provenance. A shortest-of summary takes the shortest instantiated
argument bound. Scoped state is preserved from both the actual argument and
the selected parameter. Therefore:

```text
A function call does not erase or extend reference provenance.
Result provenance is derived, not reset.
```

Assignment and return apply the R4a lexical rules to that instantiated result.
A call derived from an inner local cannot initialize an outer lifetime-bound
place, and a result derived from scoped provenance remains non-escaping.
Unknown result provenance remains conservative and cannot justify outward
flow or a required relational proof.

## 9. Enums and payload enums

**Canonical EVT1 foundation.** Enums may contain nullary and payload variants.
Variants have deterministic declaration-order tags. Payload fields are typed
and their construction arity is checked. The current spelling is:

```concept
enum Status
{
    Empty,
    Ready(int value)
}

Status status = Status::Ready(7);
```

R4c specializes this same facility for the canonical compiler-known
`Option<T>` and `Result<T,E>` forms described in Section 24; it does not add a
second enum or match system.

## 10. Match

**Canonical EVT1 foundation.** `match (value) { ... }` supports exhaustive enum
matching in expression and supported statement positions. Arms use qualified
variants; payload patterns bind names without repeating types:

```concept
return match (status)
{
    Status::Empty => 0,
    Status::Ready(value) => value,
};
```

Missing or duplicate variants are errors. The bounded Go pattern subset does
not establish wildcard, guard, range, or recursive-pattern law for EVT1.

## 11. Functions

**Canonical EVT1 foundation.** Ordinary functions use return-type-first
declarations, typed parameters, block bodies, explicit return statements,
calls, local declarations, assignment, foundational `while`, and `if`
expressions. Function names are PascalCase; parameters and locals are
camelCase.

**Provisional EVT1.** Overload resolution is bounded to the implemented exact
signature and template requirement rules. Full PoC3 callable and module rules
are deferred.

**Canonical EVT1 R3.** An `owned T` by-value parameter receives ownership.
Passing an existing movable-only owner requires `move`; the callee drops its
live parameter unless it transfers ownership onward. Returning an `owned T`
local likewise requires explicit `return move value;`. Fresh function results
may initialize an owner directly. No NRVO or implicit-move law is specified.

## 12. Failure model

**Canonical EVT1 R4c.** Core has one explicit failure calculus. `Option<T>` is
expected absence, `Result<T,E>` is typed recoverable failure, and `panic` is a
terminal invariant failure. Explicit `match` is the ground truth; postfix `?`,
postfix `!`, lexical `try`/`except`, `assert`, and `static_assert` are bounded
sugar over ordinary payload-enum control flow and assertion evaluation. The
complete laws and source forms are specified in Section 24.

The pre-R4c Vulkan `Result<void,E>` signature bridge remains an isolated ABI
compatibility seam. Legacy PoC3 fallible syntax is evidence only and does not
create a second failure channel.

## 13. Concepts

**Canonical EVT1 R4b.** Named concepts have one type parameter and a finite set
of named operation, prerequisite-concept, or compiler-analysis requirements.
The bounded compiler-analysis spelling is:

```concept
concept LifetimeBound<T>
{
    requires compiler.LifetimeSafe<T>();
}
```

R4b keeps source concepts and templates single-parameter, but compiler-analysis
requirements may consume a bounded list of semantic subjects. `Outlives`
binds the named parameter and `result` subjects of exactly one required
operation:

```concept
concept ViewOf<T>
{
    requires T MakeView(ref const int source);
    requires compiler.Outlives(source, result);
}
```

`Outlives(Source, View)` succeeds only when the selected operation's result
summary proves that `Source` lives at least as long as `View`. A result derived
from that parameter, including as one member of a shortest-of-parameters
summary, proves the relation. A result derived only from another parameter
disproves it. `Unknown` is rejection for a required proof; there is no
maybe-satisfied outcome.

The semantic subjects implemented in R4b are operation parameters, operation
results, and concrete types for the existing unary analyses. Local/ref-struct
bindings participate through their instantiated provenance at call sites.
Relational analyses run only because concept satisfaction requests them. Their
deterministic MIR evidence records analysis, subjects, outcome, provenance
facts, and the requirement-origin span. Explicit
`requires ConceptName<ConcreteType>;` assertions request a compile-time proof.
Requirement closure and cycles are diagnosed deterministically.

The Concept compiler enforces local reference invariants universally.
Additional global or restrictive analyses are explicitly requested through
semantic requirements. A template constraint consumes the resulting
compile-time witness; lifetime analysis is not silently run across unrelated
code.

**Deferred reconciliation.** Multiple parameters, specialization, negative
concepts, orphan/coherence breadth, and the full PoC3 marker-concept system are
not admitted by R0.

## 14. Templates

**Canonical EVT1 foundation.** The seed supports one-type-parameter function
templates constrained by a named concept. Instantiations are validated against
the requirement closure and monomorphized deterministically into MIR/C symbols.

**Deferred reconciliation.** Template types, template structs/enums, multiple
type parameters, partial specialization, unconstrained templates, and broad
overload interaction remain outside the R0 subset.

## 15. Compile-time evaluation

**Canonical EVT1 foundation.** `comptime` declarations/functions and
`static_assert` evaluate deterministic pure values using explicit fuel, call
depth, loop bounds, and aggregate limits. Supported values include integers,
booleans, strings in the compile-time domain, enums, structs, and fixed arrays
composed from supported values. Compile-time `while` requires a static
`bounded(limit)` clause.

**Deferred reconciliation.** PoC3 capability-based compile-time permissions,
compiler-owned temporary allocation, reflection breadth, and any I/O are not
part of the current canonical subset.

## 16. Arrays and indexing

**Canonical EVT1 R4d.** `array` is rank-1 contiguous storage. `ndarray` is
rank-N contiguous storage and is not recursively nested array. The canonical
explicit spellings are:

```concept
int<array>[4] values = [1, 2, 3, 4];
int<ndarray>[2, 3] grid = [[1, 2, 3], [4, 5, 6]];
```

The earlier `T[N]` spelling remains accepted as a rank-1 compatibility form;
nested compatibility arrays remain semantically distinct from one ndarray.
An `array` shape contains exactly one dimension. An `ndarray` shape contains
one or more dimensions. Shape is semantic information retained in the typed
model and MIR; it is not merely parser syntax. Both families are contiguous by
definition. Fixed shapes use inline value storage. Copyability and movability
are structural, immovability propagates from the element type, and `owned`,
move, deterministic Drop, Option, and Result use the ordinary value rules.
When an element type has a `Drop` witness, fixed storage drops live elements
in reverse linear storage order; a moved-from container performs no cleanup.

`values[index]` indexes rank-1 storage. `grid[row, column]` indexes ndarray
storage with exactly one integer index per rank. `grid[row][column]` denotes
repeated indexing of nested rank-1 arrays and is not ndarray syntax. A
provably out-of-bounds constant is rejected. A runtime index emits a bounds
guard and terminal panic with `Concept array index out of bounds` or `Concept
ndarray index out of bounds`; it does not produce Result and does not unwind.

Ndarray linearization is row-major with the last index contiguous. For shape
`[rows, columns]`, `[row, column]` has offset `row * columns + column`. R4d has
no configurable layout, stride, sparse, tiled, or jagged storage policy.

`Len(x)` returns the extent of rank-1 storage. `Rank(x)` returns storage rank,
and `Shape(x, dimension)` returns one dimension without constructing a shape
object. Fixed-shape queries lower to constants. The dimension argument is
integer and bounds-checked.

Rank-1 literals retain ordered value semantics. Nested ndarray literal syntax
is only source convenience for one flat contiguous value. Its rank, exact
dimensions, rectangularity, and element types are checked; ragged literals are
not ndarray values.

Extents may be compile-time or value-level runtime expressions. A runtime
extent does not choose storage or allocation policy. EVT1 distinguishes fixed
inline storage, external/non-owning storage descriptors, and future
allocator-backed owned dynamic storage. Because EVT1 currently has no explicit
owned runtime storage constructor, a bare owning runtime-shaped local is rejected with
`RUNTIME_ARRAY_REQUIRES_EXPLICIT_STORAGE` or
`RUNTIME_NDARRAY_REQUIRES_EXPLICIT_STORAGE`. In particular:

```text
Declaring a runtime-sized array or ndarray does not imply heap allocation.
Shape does not imply allocation policy.
```

The C bootstrap backend represents fixed storage as an assignable wrapper
struct containing one C array. Ndarray data is one flat element array, never a
nested wrapper tree. Concept values therefore do not acquire C array decay.

### 16.1 Explicit storage binding

**Canonical EVT1 R4e.** Unary `bind` creates a non-owning shaped reference to
existing contiguous array or ndarray storage. Its destination supplies the
target type and shape; its source supplies the backing storage:

```concept
float<array>[16] storage;
ref float<ndarray>[4, 4] matrix = bind storage;
```

The destination must be `ref T<array>[...]`, `ref const T<array>[...]`,
`ref T<ndarray>[...]`, or `ref const T<ndarray>[...]`. The source must be an
assignable whole contiguous array/ndarray place. Element types must be
identical. A mutable source may produce either mutable or const access; a const
source may produce only `ref const`. A contextless `bind source` is ill-formed
because R4e does not infer the target shape.

`bind` consumes the entire source storage. The product of the target dimensions
must equal the source element count exactly. Fixed counts are proved at compile
time. Runtime dimensions are multiplied with overflow checks and compared
before the view is formed; mismatch terminates through the deterministic panic
path with `Concept bind shape does not match storage size`, and overflow uses
`Concept bind shape product overflow`. Prefix, offset, partial, strided, and raw
pointer binding are not part of R4e.

The result aliases the source in unchanged row-major linear order. `bind` does
not allocate, copy, move, resize, transfer ownership, or extend lifetime. It
preserves the source's lexical and call-result provenance and its scoped status;
therefore binding cannot launder a local or scoped source into an escaping
reference. Existing indexing, `Len`, `Rank`, and `Shape` operate on the target
shape, including runtime descriptor dimensions.

R4e preserves the R4d zero-extent law rather than defining a new one: an extent
is nonnegative, exact total-count equality still applies, and executable binding
requires backing storage already representable by the active backend.

### 16.2 Semantic layouts and streams

**Canonical EVT1 R4f.** A `layout` declares ordered, named regions with fixed
compile-time byte geometry. Region types are ordinary fixed-size Core value
types, including fixed arrays and ndarrays:

```concept
layout PacketLayout
{
    uint header;
    align(16) byte<array>[256] payload;
    at(272) uint checksum;
}

stream PacketStream over PacketLayout
{
    Header = header;
    Payload = payload;
}
```

Default placement aligns the next free byte offset to the region's natural
alignment. `align(N)` requests a stronger power-of-two alignment; `at(N)`
requests an explicit aligned byte offset. Regions may not overlap, and layout
size includes tail padding to the greatest region alignment. Stable identity,
type, byte offset, byte extent, alignment, and disjointness survive into MIR.
`LayoutSize<T>()`, `LayoutAlign<T>()`, and `LayoutOffset<T>("region")` are
bounded compile-time queries over this closed graph. Runtime-parameterized
layout geometry is rejected.

`ref Layout = bind storage` binds one whole contiguous fixed array/ndarray
backing object with exactly the layout's byte size and sufficient alignment.
`ref const Layout` is read-only. `layout.region` projects the region's exact
ordinary type over the same bytes. Binding and projection do not allocate,
copy, move, resize, transfer ownership, or extend lifetime; they preserve
backing identity, constness, provenance, and `scoped` state.

A `stream` declares named channels that alias regions of exactly one layout.
`ref Stream = bind layoutRef` creates a zero-storage semantic view, and channel
projection is the corresponding region projection. Unknown regions, duplicate
channels, layout mismatch, and const escalation are ill-formed. Streams do not
imply iteration, queues, scheduling, transport, ownership, or execution.

R4f adds no Slice, FixedBuffer, `stackalloc`, allocation,
raw-pointer binding, runtime layout parameters, tensor mathematics, general
reflection, stream composition/runtime, GPU lowering, or stable ABI law.

### 16.3 Bounded borrowed spans

**Canonical EVT1 R4g.** `Span<T>` is a mutable bounded borrowed contiguous
interval and `ReadOnlySpan<T>` is its readonly counterpart. Both are
compiler-known generic, ref-struct-like value descriptors. The canonical
constructors are `Span(source)` and `ReadOnlySpan(source)`; the destination or
inferred source element type supplies `T`. Sources are existing contiguous
array or ndarray storage, bound array/ndarray views, contiguous layout regions,
stream channels, or another compatible span. A mutable source may form either
kind. A const or readonly source may form only `ReadOnlySpan<T>`.

The canonical bounded narrowing operation is the free function
`Subspan(parent, offset, length)`, matching EVT1's current free-function call
surface. Its interval is half open:

```text
[baseOffset + offset, baseOffset + offset + length)
```

`offset` and `length` must be nonnegative and `length` must not exceed the
remaining parent extent. `offset == parent.length` with zero length is valid.
Provably invalid constant bounds reject with `SPAN_BOUNDS_OUT_OF_RANGE`.
Dynamic bounds use overflow-safe subtraction and terminate with `Concept span
bounds out of range`; element-to-byte offset and extent arithmetic also guards
overflow.

`span[index]` is rank-one linear indexing with `0 <= index < Len(span)`.
`Span<T>` permits reads and writes; `ReadOnlySpan<T>` permits reads only.
Provable constant failure is `SPAN_INDEX_OUT_OF_BOUNDS`; dynamic failure
terminates with `Concept span index out of bounds`. Span has no `Rank` or
`Shape` surface in R4g.

Every span retains its element type, source provenance, stable parent-region
identity, relative base offset, length, byte extent, safe starting alignment,
mutability, and contiguity. Layout-region and stream-channel construction
consumes the exact R4f region identity and geometry. Subspan narrows the same
parent region; it does not manufacture an independent identity. Its alignment
is conservatively derived from the parent alignment and element byte offset.
Distinct declared-disjoint parent regions therefore remain distinguishable,
while same-parent intervals retain enough facts for later analysis without
R4g adding a noalias solver.

`Span<T>` converts to `ReadOnlySpan<T>` through the explicit canonical
constructor `ReadOnlySpan(mutableSpan)`. The reverse conversion is invalid.
Construction and narrowing preserve R4a/R4b lexical, scoped, and call-result
provenance. They do not extend lifetime or add Span-specific lifetimes,
named lifetimes, NLL, or a second borrow checker.

Span owns no storage, allocates nothing, copies no backing elements, transfers
no ownership, and never drops backing storage. Copying its descriptor follows
the existing ref-struct value rule and does not copy or move elements. An
immovable backing object may be viewed because construction does not relocate
it. In compact form:

```text
Span is a bounded borrowed contiguous interval.
Span does not own, allocate, copy backing storage, or extend lifetime.

ReadOnlySpan is the readonly counterpart of Span.
Span may convert to ReadOnlySpan.
ReadOnlySpan may not convert to Span.
```

### 16.4 Tensor mathematics and Einstein indexing

**Canonical EVT1 R4h.** `tensor<T, Rank>` is a mathematical rank-`Rank` view
over existing shaped contiguous storage. `Rank` is a positive compile-time
integer and `T` is a supported scalar arithmetic type. Tensor is not nested
array storage: it owns no storage, allocates nothing, copies no backing
elements, transfers no ownership, and does not alter lifetime provenance or
region identity.

```concept
int<ndarray>[2, 3] storage = [[1, 2, 3], [4, 5, 6]];
tensor<int, 2> values = Tensor(storage);
int item = values[1, 2];
```

`Tensor(source)` requires an explicit tensor destination and a shaped
contiguous source of the same rank and element type. Fixed or bound ndarrays,
shaped layout regions, and shaped stream channels are admitted. A `Span<T>`
may form only `tensor<T, 1>`; R4h never invents a multidimensional shape from
a length. The view retains ordered shape, parent region, base offset,
alignment, mutability, contiguity, and lexical/call-result provenance.
Readonly storage produces a readonly tensor destination capability.

Ordinary indexing is zero-based, has exactly one integer index per rank, and
uses the existing deterministic bounds behavior. Tensor operations write into
an existing mutable destination; they never conjure result storage.

An indexed tensor assignment creates a statement-local symbolic-index scope:

```concept
C[i, j] = A[i, k] * B[k, j];
```

The compiler owns the hidden `extent`, `current`, and reduction accumulator
state. Those names are integer-like only within this statement and do not
escape as ordinary locals. LHS indices are distinct free indices. Every free
index must occur on the RHS. An RHS-only index must occur exactly twice in the
bounded R4h Einstein subset and is a reduction index. Every occurrence of an
index has one compatible extent. Mixed ordinary and symbolic indices,
duplicate output indices, missing output indices, inconsistent extents, and
other multiplicities reject rather than being guessed.

The non-contraction indexed form synthesizes a loop nest and can populate
fixed storage during bounded comptime evaluation:

```concept
T[i, j] = if (i == j) 1 else 0;
```

Outside indexed Einstein notation, `+`, `-`, and `*` are elementwise tensor
operators and require exact operand and destination shapes. Tensor-times-scalar
is supported for exact element scalar type. This scalar application is not
broadcasting. Concept tensor operations do not broadcast implicitly.

`A @ B` is generalized tensor contraction: the last axis of `A` contracts
with the first axis of `B`. For shapes `[a0, ..., aN]` and
`[b0, ..., bM]`, `aN == b0` is required and the result shape is
`[a0, ..., aN-1, b1, ..., bM]`. Therefore `[M,K] @ [K,N]` is matrix
multiplication, while `[A,B,K] @ [K,C,D]` produces `[A,B,C,D]`. Rank-one
dot-product scalar results are deferred because R4h tensor ranks are positive.
Einstein indexing is the explicit general contraction form; `@` is the
canonical last/first-axis shorthand.

Fixed mismatches reject statically. Runtime extents receive exact deterministic
guards before loop execution. Contraction rejects destination/input region
overlap. Exact corresponding-element operations may be in-place. The compiler
does not allocate scratch storage for uncertain aliases.

Tensor semantic MIR retains views, operands, symbolic maps, free/reduction
sets, shape relations, provenance/region/alignment/mutability, and alias
policy. A dedicated tensor lowering stage then synthesizes zero-based loops,
with arithmetic zero as the multiplication-sum reduction identity, before the
strict-C11 backend. R4h adds no tensor runtime, heap allocator, BLAS, MLIR,
SIMD, GPU lowering, broadcasting, strides, sparse storage, autograd, named
axes, or `vector`/`matrix` aliases.

## 17. Slices and bounded collections

**Legacy PoC3.** PoC3 implemented read-only `Slice<T>` and
`FixedBuffer<T, N>` foundations.

**Redesign.** The old `Slice<T>` surface is design pressure, not a direct-port
candidate. R4g supersedes its direction with canonical `Span<T>` and
`ReadOnlySpan<T>` over the active reference, provenance, storage, bind, layout,
and stream model. Slice remains legacy evidence only. FixedBuffer mutation and
collection rules, stack allocation, and owned dynamic storage remain future
work; `bind` remains an exact whole-storage association rather than a subregion.

## 18. Interfaces and dyn

**Legacy PoC3.** PoC3 contains interface declarations and bounded dynamic
dispatch fixtures.

**Redesign.** A concept is the semantic contract, compile-time satisfaction
produces a witness, and `template<T satisfies C>` consumes operation and
semantic-proof evidence statically. Future `dyn C` is a runtime-erased value/reference paired with a
reified witness; `interface` may become optional sugar or a marker for a
dyn-compatible concept. Runtime witness reification is downstream of
compile-time satisfaction and must not imply hidden allocation. Relational
lifetime proofs remain compile-time evidence and do not become runtime witness
baggage. R4b does not
implement dyn, interface objects, or a vtable ABI.

## 19. Allocation

**Legacy PoC3.** PoC3 explored explicit allocator, arena, store, allocation
effect, and index-based identity semantics.

**Deferred reconciliation.** The Go seed adds no general allocation model.
EVT1 has no implied heap or default allocator. Profile-generated native code
may call admitted mechanisms without making those calls core allocation law.

## 20. C ABI

**Legacy PoC3.** PoC3 implements `extern`, export, and `repr(C)` pressure with
fixtures.

**Deferred reconciliation.** The current C11 generator is a bootstrap backend,
not a finalized Concept C ABI. Export naming, layout guarantees, header
contracts, and FFI-safe type derivation require explicit EVT1 decisions.

## 21. Machines and automata

**Provisional EVT1 / deferred reconciliation.** PoC3 defines core `machine`,
states, transitions, `decide`, nested machine composition, and `yield` behavior.
The Go seed defines bounded `automata -> machine -> state`, typed fixed local
instances, guarded deterministic dispatch, push/pop continuation constraints,
and compiler-owned outcome values.

R0 preserves the Go implementation as a provisional core candidate because
both lines demonstrate general control semantics. It does not declare either
surface canonical, does not alias `machine` and `automata`, and does not import
PoC3 decide/yield semantics into the Go model.

## 22. Effects, actuators, and profiles

**Profile-specific.** R0 admits `effect`, ordered emitted-effect batches, and
`actuator` mappings only under `profile Vulkan;`. The Vulkan profile owns
mechanism types, Prometheus imports, mapping admissibility, and Vulkan C
bindings.

**Deferred reconciliation.** PoC3 discusses broader effect/default-profile
laws, but R0 does not promote the Vulkan spelling or implementation to core.
Any general effect system requires a separate cross-line decision.

## 23. Testing

**Legacy PoC3.** `.con_test`, discovery, facts/theories, expectations, and the
Zig-hosted test runner remain executable reference behavior.

**Deferred reconciliation.** The Go compiler has Go-level compiler tests but no
canonical Concept testing language. R1 differential tests are infrastructure,
not automatic adoption of the PoC3 testing surface.

## 24. Panic and assert

**Canonical EVT1.** Absence, recoverable failure, assertion failure, and panic
are distinct. `Option<T>` is expected absence and carries no error information.
`Result<T, E>` is recoverable success or typed error. Both are compiler-known
generic payload-enum forms and obey ordinary exhaustive match, value,
copy/move/drop, immovable-embedding, and lifetime-provenance laws:

```concept
Option::Some(value)
Option::None
Result::Ok(value)
Result::Error(error)
```

These qualified spellings are canonical; there is no competing unqualified
built-in constructor surface. `match` is the fully explicit ground truth.

Postfix `?` unwraps success or returns `None`/`Error(error)` from the current
function. `Option<T>?` requires an `Option<U>` return channel.
`Result<T,E>?` requires `Result<U,E>` with the exact same `E`, unless a lexical
`try` block has an exact matching `except (E error)` arm. Option and Result do
not convert into one another, and cross-error conversion is not implicit.

Postfix `!` unwraps success or calls the terminal panic path at its source
span. Therefore `?` never panics and `!` never silently propagates.

`try` establishes only a local propagation boundary. A `?` inside it branches
to an exact statically selected Result error arm. If no arm matches, the error
may propagate outward only through an identical enclosing Result error type;
otherwise compilation fails. R4c does not route Option through try/except.
There is no `throw`, exception object, runtime type test, exception table,
cross-frame catch, or stack unwinding. In particular:

```text
try/except is structured Result handling, not exceptions.
```

Panic terminates the current execution path without unwinding. The C bootstrap
helper reports a deterministic reason and one-based source line/column, then
aborts. It performs no allocation and implies no generalized panic runtime.

`assert(condition, reason)` is runtime sugar for
`Assert.True(condition, reason)`; the one-argument form uses the deterministic
reason `Concept assertion failed`. False escalates to the same terminal panic
primitive and true continues. `Assert.True` is the only required primitive;
this is not a testing framework.

`static_assert(condition, reason)` is exactly compile-time sugar for
`comptime Assert.True(condition, reason)`. It uses the existing bounded
comptime evaluator and emits no runtime code. Runtime values are rejected.

## 25. Backend and runtime boundaries

**Canonical EVT1 foundation.** The active bootstrap backend produces strict
C11 C/H, deterministic MIR JSON, a source map, and a hash manifest. Backend
output is derived evidence; the typed Module, semantic checks, and MIR carry
compiler authority. Timestamps are forbidden. Source spans and stable ordering
are preserved.

**Profile-specific.** Vulkan includes, runtime types, capability names, and
mechanism mappings appear only when admitted by the Vulkan profile.

The C representation is not itself a language specification. Runtime services,
Oct, and Prometheus are consumers and may not become hidden compiler build
dependencies.

## 26. Deferred EVT1 items

The following remain explicit reconciliation or implementation work:

- general imports and multi-module compilation;
- complete primitive widths and conversion rules;
- ownership beyond bounded local move/drop/reference and live-replacement
  accounting;
- generalized borrow checking, named lifetimes, and non-lexical lifetimes;
- explicit external runtime-array/ndarray descriptors, slices, FixedBuffer,
  and bounded dynamic storage construction;
- witness reification, interfaces, and dyn storage/dispatch;
- allocation, allocator effects, arenas, and stores;
- stable C ABI and layout law;
- canonical machine/automata syntax and decide/yield semantics;
- general effects versus profile-owned effects/actuators;
- Concept-native testing (runtime assertion sugar is canonical, but no testing
  framework is implied);
- broader concepts/templates and capability-based comptime.

EVT1 intentionally adds none of these merely because they were next on the PoC3
roadmap. Promotion requires a matrix decision, a specification update, and
executable EVT1 evidence.
