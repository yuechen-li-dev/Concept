# Concept EVT1 language specification foundation

Status: R5g reducible async control-flow normalization

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

Return type precedes function name. EVT1 does not use `fn`, `name: Type`, or
`-> ReturnType`. Ordinary declarations remain `Type name` and
`const Type name`. R5i admits inferred `auto name = value` and
`const auto name = value` only for generated concrete callable types that users
cannot spell. `var` and inferred `let` are compatibility aliases for `auto` and
`const auto`; canonical documentation must use the C/C++-shaped forms. R2's
typed `let Type name = value;` remains an exact compatibility alias of
`const Type name = value;`. Braces and indentation remain human-readable, with
one statement per line.

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

**Canonical EVT1 R4i.** A concrete fixed tensor may instead state its shape
after the declared name:

```concept
tensor<int> A[2, 2] = [[1, 2], [3, 4]];
vector<float> v[3] = [1.0, 2.0, 3.0];
matrix<float> m[2, 2] = [[1.0, 0.0], [0.0, 1.0]];
```

For `tensor<T> name[d0, ..., dN]`, the element type is `T`, rank is the
number of dimensions, and every dimension must be a positive compile-time
integer accepted by the fixed-storage rules. The compiler synthesizes one
fixed `T<ndarray>[d0, ..., dN]` backing region, initializes it, and constructs
the ordinary tensor view over that region. A fixed shaped tensor declaration
may synthesize its backing storage because the declaration fully determines
that storage. It does not create an owning tensor type, heap allocation,
allocator call, runtime tensor object, ownership transfer, or backing copy.

Nested initialization reuses the ndarray literal laws for exact rank, shape,
rectangularity, and element type. An exact element-typed scalar initializer
fills every fixed backing element; this is initialization sugar, not
broadcasting. An omitted initializer remains ill-formed under the existing
local fixed-storage initialization law. `const` makes both the declared tensor
capability and its synthesized backing readonly through that declaration.

`vector<T>` is exactly `tensor<T, 1>` and `matrix<T>` is exactly
`tensor<T, 2>`. They are type identities, not nominal wrappers, and therefore
have identical parameter compatibility, concept satisfaction, Tensor MIR,
operators, and storage behavior. Their shaped declarations require exactly
one and two dimensions respectively. Shape remains explicit; R4i does not
infer it from a literal alone.

Runtime or external storage remains explicit. A runtime-shaped inline tensor
declaration is rejected; existing storage, layout regions, stream channels,
bound ndarrays, and Span sources continue to use `tensor<T, Rank> name =
Tensor(source)`. This preserves source provenance, region, alignment,
mutability, and storage-policy decisions.

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
`[K] @ [K]` contracts directly to scalar `T`; rank-zero exists in Tensor MIR
for this result but R4i adds no source-level `tensor<T, 0>` variable form.
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
strict-C11 backend. R4i adds no tensor runtime, heap allocator, BLAS, MLIR,
SIMD, GPU lowering, broadcasting, strides, sparse storage, autograd, named
axes, tensor slicing, or separate vector/matrix MIR.

## 17. Semantic facts and proof requirements

EVT1 exposes a bounded compiler-owned semantic-fact vocabulary:
`Contiguous`, `Bounded`, `FixedShape`, `RuntimeShape`, `Rank`, `Aligned`,
`Mutable`, `Readonly`, `SameRegion`, `Disjoint`, `NoAllocation`, `NoCopy`, and
`NoOwnershipTransfer`. Existing `LifetimeSafe`, `NonEscaping`, and `Outlives`
analyses use the same certainty discipline.

A fact is `Proven`, `Disproven`, or `Unknown`. A required compiler analysis
succeeds only for `Proven`; `Unknown` fails conservatively. Parameterized facts
use positive compile-time integers. Alignment parameters must be powers of two
from 1 through 4096, and a proven alignment implies divisible weaker
alignments.

Type-level requirements consume only guarantees true for every value of the
type. Value and region facts retain concrete subject identity and evidence in
MIR. Thus `Contiguous<Span<int>>` and `Rank<Span<int>>(1)` are type guarantees,
while alignment of a particular Span depends on its source and offset.
`FixedShape` is distinct from finite runtime-carried `Bounded` extent.

Relational facts use the bounded semantic-subject mechanism. `SameRegion`
means stable parent region identity, not equal subranges. `Disjoint` is proven
only from declared distinct regions or statically known non-overlapping
half-open intervals. No general alias inference is implied.

Semantic fact proofs erase before C lowering. Optimization behavior is not
normative in R4j.

## 18. Slices and bounded collections

**Legacy PoC3.** PoC3 implemented read-only `Slice<T>` and
`FixedBuffer<T, N>` foundations.

**Redesign.** The old `Slice<T>` surface is design pressure, not a direct-port
candidate. R4g supersedes its direction with canonical `Span<T>` and
`ReadOnlySpan<T>` over the active reference, provenance, storage, bind, layout,
and stream model. Slice remains legacy evidence only. FixedBuffer mutation and
collection rules, stack allocation, and owned dynamic storage remain future
work; `bind` remains an exact whole-storage association rather than a subregion.

## 19. Interfaces and dyn

**Legacy PoC3.** PoC3 contains interface declarations and bounded dynamic
dispatch fixtures.

**Canonical EVT1 / R4k.** `interface C<T>` is a specialized concept whose
requirements have a fixed runtime witness shape. All interfaces are concepts;
not all concepts are interfaces. Interface satisfaction uses the ordinary
concept requirement resolver. Instance methods must begin with `ref T self` or
`ref const T self`; concrete field requirements use `requires U field;`, and
`requires const U field;` is getter-only. Interface composition is ordinary
`requires Other<T>;` composition, not inheritance.

`template<T satisfies I>` consumes the compile-time witness and remains
statically dispatched. `dyn I` is an explicit, non-owning erased reference plus
a runtime-reified witness. It is constructed only with `ref place` or `ref const
place`, preserves that place's provenance and mutability, performs no allocation
or object copy, and owns no backing storage. One deterministic static witness
table is emitted per `(interface, concrete type)` specialization. Required
fields mechanically synthesize getter and, for mutable requirements, setter
entries. Compiler semantic requirements are proven when dyn is constructed and
have no runtime payload.

Objects carry no vtable, header, RTTI, registry, or universal base. R4k has no
downcast or owning dyn. `dyn const I` cannot call mutable receiver methods or
write fields.

## 19a. Lightweight classes

**Canonical EVT1 / R4k.** A `class` groups ordinary fields and methods behind
`public:` and `private:` sections. Class members default private; struct members
remain public. External access to a private field or method is rejected
semantically, while methods of the declaring class may use private members.
Interface satisfaction considers only public capability.

Class storage is ordinary Concept value storage. A class does not imply heap
allocation, identity, reference semantics, GC, an object header, RTTI, or a
vtable. Copy, move, immovability, and Drop behavior derive from fields under the
existing rules. Existing aggregate construction remains the only constructor
mechanism and may not externally initialize private fields.

Methods normalize before semantic matching to ordinary functions with an
explicit first receiver parameter. An omitted receiver becomes `ref Class self`;
an explicit `ref const Class self` supplies a readonly method. Member-call syntax
inserts the receiver reference. Bare member names inside a method normalize to
`self.member` when unambiguous. There is no `virtual`, override hierarchy,
abstract class, protected access, friend access, or inheritance syntax.

> A class is encapsulation, not an allocation or inheritance model.
>
> An interface is a dyn-reifiable specialized concept.
>
> `dyn Interface` is a non-owning erased reference plus runtime witness.

## 20. Allocation

**Legacy PoC3.** PoC3 explored explicit allocator, arena, store, allocation
effect, and index-based identity semantics.

**Deferred reconciliation.** The Go seed adds no general allocation model.
EVT1 has no implied heap or default allocator. Profile-generated native code
may call admitted mechanisms without making those calls core allocation law.

## 21. C ABI

**Legacy PoC3.** PoC3 implements `extern`, export, and `repr(C)` pressure with
fixtures.

**Deferred reconciliation.** The current C11 generator is a bootstrap backend,
not a finalized Concept C ABI. Export naming, layout guarantees, header
contracts, and FFI-safe type derivation require explicit EVT1 decisions.

## 22. Automata, machines, states, and persistent capture

**Canonical EVT1 R5a.** `automata` is the outer persistent composition unit;
`machine` is an independently stepped state machine contained by that unit;
`state` is a named execution state inside one machine; and `transition` changes
only the current state of that machine. These levels are distinct and are not
called HFSMs in the canonical language.

```concept
automata Worker
with state
{
    ref Device device;
    owned Buffer scratch;
    int retryCount;
}
{
    machine Run
    {
        int attempts;

        state Idle
        {
            retryCount = retryCount + 1;
            transition Working;
        }

        state Working
        {
        }
    }
}
```

`with state` materializes one deterministic environment with identity
`Worker#state`, shared by all contained machines. Machine fields are separate
machine-local persistent storage. Ordinary state-body locals are transient:
every `Step(instance, Machine)` invocation creates them anew and destroys them
at state-body/step exit. Persistent state must be declared. A value survives a
step, transition, or future suspension boundary only in automata state, a
machine field, or another explicitly persistent object reachable by ordinary
legal reference/ownership state.

State fields use ordinary value, `const`, `ref`, `ref const`, `scoped`,
`owned`, dyn, Span, array, and tensor storage laws. Capture mode is never
inferred. Instance construction uses the narrow existing instance form with
arguments in state-field declaration order, for example
`instance Worker worker(ref device, move scratch, 0);`. Owned arguments require
explicit move and are dropped exactly once in reverse field order. Borrowed
fields preserve ordinary provenance and cannot launder an escaping or scoped
reference.

Name lookup inside a state body is transient local, then machine field, then
automata state field. `state.field` explicitly selects shared state and
`machine.field` selects the current machine's storage. Sibling machine fields
are private. Outside a body, `instance.state.field` and
`instance.Machine.field` provide explicit inspection; an instance is otherwise
not copyable or returnable as an ordinary value in R5a.

The first declared state is initial. Symbolic identity is
`Automata.Machine.State`; numeric tags use declaration order.
`Step(instance, Machine)` executes exactly one current state body and never
steps siblings. `State(instance, Machine)` reads its current tag.
`transition Target;` targets only the current machine, performs transient
cleanup, updates the tag, and returns from the step. No hidden heap,
closure/coroutine frame, scheduler, event loop, or runtime registration exists.

**Canonical EVT1 R5b transition match.** A state body may perform categorical
dispatch directly to local states:

```concept
transition match (signal)
{
    Signal::Start => Running;
    Signal::Stop => Idle;
    Signal::Fault(code) => Failed;
}
```

The scrutinee is evaluated exactly once. Arms use the ordinary payload-enum,
`Option`, or `Result` pattern validator, including payload arity and binding
types, duplicate rejection, and exhaustive coverage of closed variants. Every
target resolves in the current machine. Selection performs ordinary transient
cleanup, updates the current-state tag, and returns from the current `Step`.
There is no arm fallthrough, scoring, or ranking. The stable defensive runtime
failure is `machine transition match found no matching case`.

**Canonical EVT1 R5b transition decide.** The canonical non-redundant candidate
syntax is:

```concept
transition decide
{
    Running when CanRun() score RunScore();
    Waiting when CanWait() score WaitScore();
    Failed score FailureScore();
}
```

Candidates are visited in declaration order. A candidate without `when` is
always enabled. A guard is evaluated exactly once and must be `bool`; a false
guard excludes the candidate and its score is not evaluated. Each enabled
score is then evaluated exactly once, in declaration order. All scores in one
block have one exact type, either `int` or `float`; there is no implicit mixed
numeric promotion. Float NaN terminates with `machine decision transition
score is NaN`.

The selected candidate is the hard maximum over all enabled candidates:
`transition decide` selects the first declared enabled candidate whose score
equals the maximum enabled score. Equality therefore never replaces the
current best. Duplicate target states remain distinct candidates, and the
current state may be named explicitly without special semantics. If no
candidate is enabled, execution terminates with `machine decision transition
has no enabled candidates`; remaining in the current state is never implicit.
After selection, transient cleanup precedes the state-tag update and the
current `Step` returns.

`transition decide` is a local, stateless hardmax primitive. It is not ordinary
`if`/`else`, because all enabled scores participate in the global maximum; it
is not a scheduler or DragonGod's stateful Decision policy. Guards and scores
are ordinary expressions and retain source-order side effects, ownership,
reference, and explicit failure behavior.

**Canonical EVT1 R5c inference.** `Inference<T>` is an ordinary non-owning,
copyable value over a payload-free enum candidate type. It contains fixed
inline probabilities, source-order candidate identities, and a fixed count.

```concept
Inference<Action> belief = infer
{
    Attack when CanAttack() score AttackLogit();
    Search score SearchLogit();
    Idle score IdleLogit();
};

float attackProbability = belief[Action::Attack];
Action selected = HardMax(belief);
float confidence = Confidence(belief);
```

Candidate names are unique variants of `T`. Guards run once in source order;
a disabled score does not run. Enabled scores run once in source order and
must be exactly `float`. NaN and no-enabled paths terminate with `inference
score is NaN` and `inference has no enabled candidates`.

`infer` applies temperature-1 stable softmax with maximum subtraction.
Disabled and `-Inf` candidates have zero probability. One `+Inf` receives
all mass; multiple `+Inf` candidates split mass equally. All enabled
`-Inf` terminates with `inference normalization has no finite support`.

Indexed access requires a qualified candidate of `T`. `HardMax` selects the
first source-declared maximum; `Confidence` returns that probability.
Inference has no truthiness. For identical finite candidates, guards, float
scores, and order, `HardMax(infer(scores))` agrees with direct `decide`, but
`decide` remains raw-score argmax and need not normalize.

`transition infer with HardMax { ... }` is the only R5c transition-inference
policy. It infers, selects, cleans transients, updates the tag, and returns from
`Step`. Missing or unknown policy is invalid. Randomness is never implicit;
sampling and explicit RNG are deferred. The older typed-signal automata form
remains compatibility surface. Plain value-level decide,
completion/result, nested machine values, and effects/actuators remain later.

**Canonical EVT1 R5d yield.** Bare `yield;` is valid only in a runtime machine
state body. Taking it cleans transient locals in reverse lexical declaration
order, preserves the current-state tag and all explicit automata/machine
persistent storage, leaves completion and result storage untouched, and ends
the current `Step`. The next explicit `Step(instance, Machine)` enters that
same state from its first statement. It does not resume after the yield site.

`yield` does not preserve ordinary transient locals and does not extend or
launder reference provenance. An owned transient local drops before Step
returns; an owned persistent field does not drop at yield. `yield value`, saved
program counters, coroutine frames, schedulers, async, and generator semantics
are not part of R5d.

**Canonical EVT1 R5d foreach.** The required spelling is
`foreach (Item item in source) { ... }`; the item type is explicit. Its
semantic expansion obtains iterator state once, calls `MoveNext(ref iterator)`
once per attempt, calls `Current(ref const iterator)` once per successful
attempt, recreates the item binding for each iteration, and cleans item and
iterator storage by ordinary lexical cleanup. The source expression is
evaluated exactly once.

Fixed arrays, ndarrays, `Span<T>`, and `ReadOnlySpan<T>` have compiler-known
inline iterators. Ndarrays iterate scalar elements in canonical row-major
storage order. User types participate through ordinary free functions
`GetIterator(source)`, `MoveNext(ref iterator) -> bool`, and
`Current(ref const iterator) -> Element` (or a compatible reference). No
backing array is copied and no iterator heap object, registry, or virtual
dispatch is implicit. Value iteration copies only copyable elements; movable-
only elements require explicit reference iteration. Mutable references cannot
be obtained from readonly sources.

A foreach inside a state body is transient. If its body yields, its iterator is
cleaned and the next Step restarts the state and therefore the foreach from the
beginning. If iteration progress matters across Steps, iterator state must be
stored explicitly in automata or machine persistent storage and advanced with
the protocol operations directly.

**Canonical EVT1 R5e machine stack and completion.** Each automata instance has
an inline bounded stack of at most eight machine frames. A frame contains a
machine declaration tag, current state, and that frame's machine-persistent
fields. All frames reference the one outer `with state` environment. The stack
contains no saved instruction pointer and implies no scheduler or heap.

At depth one, `Step(instance, Machine)` retains R5a's explicit top-level
machine selection. At greater depth every Step spelling advances the active
top frame exactly once. `yield;` keeps that frame; `transition Target;` changes
only its state. The child call is `push Child goto ResumeState;`: initialize a
new frame, make it topmost, and record an explicit parent state. Overflow
panics with `automata machine stack capacity exceeded`. Same-machine frames
are legal and independent.

Machine declarations may spell `machine M returns T fails E`; either clause is
optional. `pop;` and `complete;` are Neutral semantic pops, `complete value;`
is Success, and `fail error;` is Failure. Payloads exactly match the declared
types. Cleanup runs transient-then-frame; explicit `move` into an outcome
transfers ownership and suppresses the source drop. Root pop empties the stack.

`Result(instance, Machine)` reads the last outcome; `Result(Child)` is its
parent-state form. Tags are 1 Neutral, 2 Success, and 3 Failure, with typed
`success`/`failure` fields when declared. A pre-completion read panics with
`machine result cannot be read before completion`. Machine outcome is not
`Result<T,E>`, and Neutral fabricates no value. Internal Step status remains
distinct and non-public.

**Canonical EVT1 R5g async/await.** `async` and `asynchronous` are exact
lexical aliases on a function declaration. `await` and `awaitchronous` are
likewise exact aliases. Parsing normalizes each pair to one AST form, so they
have identical typing, MIR, diagnostics, and lowering.

Calling `async T F(...)` constructs an explicit movable-only `Async<T>` value.
Construction initializes fixed inline generated-machine storage but executes
no function body. The caller advances it with `Step(operation)`, tests it with
`Complete(operation)`, and reads its eventual value with `Result(operation)`.
A copyable result may be read again; a movable-only result is consumed once.
Reading a result before completion, copying an active operation, consuming a
movable result twice, or exceeding the bounded depth
of eight frames is a deterministic error. There is no truth conversion.

An async function has stable generated identities `F#async`,
`F#async#state`, and `F#async#machine`. Each await point has an explicit
generated continuation state. Its operand is evaluated once, the child
machine is pushed, and only the top frame is advanced. Child yield leaves the
child active. Child completion pops it, records one outcome, and a later Step
dispatches the revealed parent continuation, which consumes that outcome
once. Awaiting an already-completed moved operation uses the same outcome law
without another child step.

Parameters and locals live across an await become fields of the generated
inline frame. Transitive backing objects needed by ref, Span, tensor, or
storage views are retained as required by ordinary provenance. Dead locals
remain lexical and are cleaned before the child push. Persistent owned values
are cleaned once at function completion or transferred by explicit `move`.
References stored across await must be proven to outlive the operation; no
unrelated lexical value is captured.

`await Child()?` first awaits `Async<Result<T,E>>` and then applies the
ordinary exact-error `?` law to the resulting `Result<T,E>`. Async machine
Success carrying a `Result` is not `Result::Ok`; these are distinct layers.
Value-returning async functions complete with a value, while `async void`
completes neutrally. Bare return from a value-returning async function is
invalid.

Async structured control flow is normalized into explicit generated machine
states. Reducible sequence, `if`/`else`, exhaustive `match`, bounded `while`,
`foreach`, nested combinations, early returns, and bounded local
`try`/`except` regions may contain multiple awaits. Await splits the containing
region into explicit suspend and continuation states. Branch and match arms
enter deterministic arm states and converge through generated join states;
loops use explicit header and backedge states. The ordinary definite-
assignment and moved-value lattices still decide whether a joined value is
usable.

Match scrutinees, branch and loop conditions, foreach sources, iterator
operations, and await operands retain ordinary evaluation counts. Async
foreach persists source, iterator/index, and current-item progress across
awaits; bare `yield` does not acquire that persistence. Generated-frame
liveness includes values crossing any generated state boundary and retains
ordinary ownership, cleanup, and reference provenance laws. Arm-local and
iteration-local values remain transient when no later state uses them.

If structured control flow cannot be represented by the supported explicit
state graph, the compiler rejects it rather than introducing a hidden saved-PC
coroutine model. Await in `if`/`while` conditions, a match scrutinee, or a
foreach source remains deferred; write an explicit preceding await instead.
Arbitrary goto, computed jumps, irreducible graphs, exception unwinding, and
general continuation capture are not part of EVT1.

Async/await is source-level automation of explicit machine-stack state
progression. Await continues after the suspension point; bare machine yield
re-enters the state from its beginning. Async implies no scheduler, executor,
thread, event loop, VM, coroutine ABI, saved instruction pointer, or hidden
heap allocation.

**Canonical EVT1 R5h async interface composition.** An async interface method
is an ordinary interface method whose result is `Async<T>`. The spellings
`requires async T F(ref U self);` and `requires asynchronous T F(ref U self);`
normalize to `(ref U) -> Async<T>` before ordinary concept satisfaction and
witness matching. Receiver type and constness, parameters, `Async<T>`, and its
eventual `T` must match exactly. Sync and async methods never implicitly wrap
or unwrap to satisfy one another.

Struct and class methods satisfy that normalized requirement through the same
resolver as every other operation. A constrained template remains statically
dispatched, and synchronous code may retain the returned movable-only
`Async<T>` without becoming async.

Dynamic async dispatch selects the concrete `Async<T>` constructor through the
ordinary `(interface, concrete type)` witness. Construction evaluates its
operands once, executes no async body, allocates nothing, and transfers the
fresh operation once. The returned operation contains concrete generated
machine behavior; later `Step`, `Complete`, `Result`, or `await` uses ordinary
R5f/R5g machinery and does not redispatch through the interface.

A dyn receiver remains non-owning. A generated frame that retains `self` must
not outlive its backing object; local and scoped dyn provenance cannot be
laundered into an escaping operation. `dyn const I` may invoke only readonly
receivers. Interface composition and compiler facts remain unchanged: static
facts erase, while one mixed sync/async witness carries fixed runtime entries.

R5h defines no async vtable, virtual `Step`, task/promise/future runtime,
scheduler, executor, continuation allocation, hidden heap, RTTI registry, or
stable public C ABI. Open generic runtime async methods remain rejected. A
future internal ABI may construct into caller-provided storage only with the
same explicit allocation-free semantics.

## 23. Effects, actuators, and profiles

**Profile-specific.** R0 admits `effect`, ordered emitted-effect batches, and
`actuator` mappings only under `profile Vulkan;`. The Vulkan profile owns
mechanism types, Prometheus imports, mapping admissibility, and Vulkan C
bindings.

**Deferred reconciliation.** PoC3 discusses broader effect/default-profile
laws, but R0 does not promote the Vulkan spelling or implementation to core.
Any general effect system requires a separate cross-line decision.

## 24. Testing

**Legacy PoC3.** `.con_test`, discovery, facts/theories, expectations, and the
Zig-hosted test runner remain executable reference behavior.

**Deferred reconciliation.** The Go compiler has Go-level compiler tests but no
canonical Concept testing language. R1 differential tests are infrastructure,
not automatic adoption of the PoC3 testing surface.

## 25. Panic and assert

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

## 26. Explicit callable capture environments

A callable literal has one canonical form:

```concept
callback(int value)
with (factor, move owner, ref total, ref const config)
{
    return value * factor;
}
```

The `with` clause is optional only when the environment is empty; `with ()` is
also an empty environment. Each literal denotes a unique, source-order concrete
type consisting of a code identity and a generated inline environment. Its
identity is `<function>#callback<n>` and its environment identity is
`<function>#callback<n>#environment`.

Capture entries are evaluated exactly once, left-to-right, and become fields in
that same order. `x` copies `x`; `move x` transfers it; `ref x` stores a mutable
borrow; and `ref const x` stores a readonly borrow. The bounded named copy form
`field = expression` proves evaluation order. Move/ref aliases and generalized
fallible capture construction are deferred; the four canonical forms otherwise
accept lexical identifier/place sources.

Outer lexical values are not captured implicitly. A local, parameter, or
normalized `self` used by the body must be listed in `with (...)`; `self` is
never implicitly captured and an unbound field name never means `self.field`.
Module functions, types, concepts, and compile-time symbols remain ordinary
module lookup and need no environment field. Within a body, locals shadow
parameters, parameters shadow captures, and captures shadow module symbols.

Capture bindings obey ordinary Concept copy, move, lifetime, provenance,
constness, and Drop rules. Duplicate captures reject. A non-copyable source
cannot use copy capture. Borrowed environment provenance is bounded by the
borrowed source, including Span, tensor, and dyn backing. Owned fields drop in
reverse capture order; borrowed fields never own their referents. Callable
copyability is structurally derived from its fields. Mutation of copied/moved
fields requires mutable callable storage; moving a field out makes that field
moved and suppresses its later environment Drop. These are derived receiver and
storage facts, not `Fn`, `FnMut`, or `FnOnce` categories.

The erased spelling is `callback<P1, P2 -> R>`. It is a non-owning pair of an
environment reference and one deterministic static invoke entry for the
`(signature, concrete callable type)` pair. Construction requires an explicit
borrow of named concrete storage. Owning temporaries are rejected rather than
boxed. Erased values retain signature, mutability, and environment provenance;
they allocate no storage and carry no RTTI or registry.

Record `with`, automata `with state`, and callable `with (...)` are distinct
grammar productions. They share only the principle that attached state is
explicit.

## 27. Backend and runtime boundaries

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

## 28. Deferred EVT1 items

The following remain explicit reconciliation or implementation work:

- general imports and multi-module compilation;
- complete primitive widths and conversion rules;
- ownership beyond bounded local move/drop/reference and live-replacement
  accounting;
- generalized borrow checking, named lifetimes, and non-lexical lifetimes;
- explicit external runtime-array/ndarray descriptors, slices, FixedBuffer,
  and bounded dynamic storage construction;
- owning dyn and explicit erased-storage policies;
- owning erased callbacks, generalized move/ref capture initializers, and async
  callback literal syntax;
- allocation, allocator effects, arenas, and stores;
- stable C ABI and layout law;
- plain value-level `decide`, continuation resume, generator yield,
  completion/result, and nested machine-value reconciliation;
- general effects versus profile-owned effects/actuators;
- Concept-native testing (runtime assertion sugar is canonical, but no testing
  framework is implied);
- broader concepts/templates and capability-based comptime.

EVT1 intentionally adds none of these merely because they were next on the PoC3
roadmap. Promotion requires a matrix decision, a specification update, and
executable EVT1 evidence.
