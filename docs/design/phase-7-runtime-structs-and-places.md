# Phase 7 runtime structs and places

P7-M0 is a documentation-only milestone. It defines Concept's direction for runtime struct values, field access, field assignment, and richer place representation without implementing compiler code.

P7-M1 adds the first executable/audit-facing backend step: supported struct declarations now emit deterministic backend-local C layout typedefs. This is not an ABI commitment and does not add struct literals, field access, field assignment, MIR place projections, or runtime construction.

P7-M3 adds read-only field access expressions for struct values. Source expressions of the form `expr.field` are resolved against the receiver's struct type, produce the declared field `TypeId`, lower through MIR as field-read rvalues, and emit backend-local C field names such as `.cpt_f_x_0`. The receiver must be a struct value: raw pointers are not auto-dereferenced, `->` is not supported, and `ptr.x` is rejected rather than implicitly dereferenced. Field assignment, address-of-field, pointer field sugar, and richer place projections remain future Phase 7 work.

Phase 7 starts after the closed Phase 6 unsafe/raw-pointer slice. Phase 6 intentionally deferred ownership, move, Drop, and `MaybeUninit` until Concept has runtime structs and richer places. This document makes that prerequisite explicit and scopes the next executable-data milestone.

## Thesis

```text
Phase 7 gives Concept real user-defined runtime data.

Ownership cannot be meaningful until the language has rich places.
Concept needs struct values, field access, field assignment, and MIR place projections
before move/drop/MaybeUninit can become more than paperwork.
```

Concept already has semantic struct declarations, but those declarations are not yet the executable substrate for ordinary runtime data. The current executable path has been enum-centric: payload enums, explicit failure, `decide`, and raw pointer flow can run through HIR, MIR, and the C backend, while user-defined product types still lack runtime layout, construction, field reads, and field writes.

Phase 7 bridges that gap. It turns a named struct declaration into a backend-supported value shape, introduces places beyond whole locals and parameters, and prepares the compiler for later ownership analysis over subobjects rather than only over bindings.

## Motivation

Concept is a C++-lineage systems language for compiler, runtime, kernel, bare-metal, and high-performance native work. Those domains need user-defined value types as a basic tool, not as a late ergonomic feature.

Runtime structs are needed for:

- compiler tables, HIR nodes, MIR nodes, indexes, handles, and arena records;
- runtime and kernel records whose layout must eventually be auditable;
- value-oriented APIs where Concept values are copied, returned, and stored deliberately;
- later concepts/templates over user-defined values;
- later ownership, move, Drop, and `MaybeUninit` analyses that operate over fields and storage state.

Ownership cannot become real while assignment targets and addressable storage are only whole locals or parameters. Real ownership analysis needs at least these place forms:

- local places;
- parameter places;
- field places;
- dereference places later;
- index places later;
- MIR place projections that preserve storage identity through lowering.

The distinction matters:

- expressions compute values;
- places designate storage;
- assignment targets are places;
- address-of takes a place;
- future move/drop/storage-state analysis operates over places.

Phase 7 is therefore the bridge from enum-centric execution to general user-defined data.

## v0 runtime struct model

The first runtime shape should be deliberately simple:

```cpp
struct Vec2 {
    int x;
    int y;
};
```

Rules:

- Struct fields have resolved `TypeId`s from the existing Phase 3 declaration/type-name resolution path.
- Runtime fields initially support only backend-renderable field types:
  - `int`;
  - `bool`;
  - raw pointers whose pointees have backend-supported type rendering;
  - enums with supported backend layout;
  - nested structs only if they are simple and acyclic, and only after C layout/name recursion is stable.
- Generic structs are not part of v0.
- Methods are not part of v0.
- Visibility/export semantics do not expand beyond existing declarations.
- Custom layout attributes are not part of v0.
- `Drop`, move semantics, and ownership distinctions are not part of v0.
- Partial initialization tracking is not part of v0.

Recommendation:

- The first runtime layout milestone should support scalar, pointer, and enum fields.
- Nested structs should be deferred unless implementation proves the C backend can render names, forward declarations, and acyclic dependency order trivially.
- Unsupported field types must produce diagnostics rather than silently degrading to `int` or another placeholder representation.

## C backend struct layout v0

Source:

```cpp
struct Vec2 {
    int x;
    int y;
};
```

Backend-local C shape:

```c
typedef struct {
    int cpt_f_x_0;
    int cpt_f_y_1;
} cpt_struct_Vec2;
```

Rules:

- C type names are deterministic and backend-owned.
- C field names are deterministic and backend-owned.
- The v0 layout is an internal/audit layout, not a final ABI contract.
- There is no `repr(C)` in Phase 7.
- There are no packed, aligned, or custom layout attributes in Phase 7.
- Backend layout must reject unsupported field types with diagnostics.
- Unsupported fields must never be silently rendered as `int`.
- Empty structs may be rejected or represented explicitly; the implementation milestone should choose one behavior and fixture it.
- Recursive structs by value must be rejected.
- Recursive references through raw pointers may become possible once struct pointer type rendering is stable, but that is separate from by-value recursion.

P7-M1 chosen behavior:

- The C backend emits enum layouts first, then supported struct layouts, then functions.
- Supported struct fields are `int`, `bool`, raw pointers to backend-renderable non-struct pointees, and enums with supported backend layouts.
- Struct type names render as `cpt_struct_<escaped_struct_name>`.
- Field names render as `cpt_f_<escaped_field_name>_<declaration_order_index>`.
- Empty structs are rejected with the existing unsupported C backend type diagnostic because portable C has no empty struct value representation and Concept has not defined zero-sized runtime types.
- Struct fields by value are rejected for now, including otherwise supported structs, so recursive/nested layout ordering remains deferred.
- Raw pointers to structs can render in function/local type positions after the struct layout is emitted, but struct pointer fields are still rejected in P7-M1 until forward-reference and recursive-reference rules are specified.

This keeps C output auditable without pretending Phase 7 has finalized Concept's ABI story.

## Struct literals / constructors

Preferred v0 syntax:

```cpp
Vec2 v = Vec2 {
    x: 3,
    y: 4,
};
```

Rules:

- A struct constructor/literal names the struct type explicitly.
- Field labels are required.
- All fields must be initialized exactly once.
- Field initializer types must match field types.
- Unknown fields are rejected.
- Duplicate field initializers are rejected.
- Missing field initializers are rejected.
- Source field order should be flexible: allow any order, then check duplicates and missing fields against declaration metadata.
- No shorthand field initialization in v0 unless it is proven trivial and does not obscure diagnostics.
- No positional struct literals in v0.
- No update/rest syntax in v0.
- No partial initialization in safe struct literals.

Labels are preferred because they are auditable, reorder-safe, compiler-friendly, and avoid C positional initializer hazards. A reviewer should not need to count fields to know what value is being constructed.

## Field access expressions

Syntax:

```cpp
int x = v.x;
```

Rules:

- The receiver type must be a struct value.
- The named field must exist on that struct.
- The expression type is the field's resolved `TypeId`.
- Field access over raw pointers is not automatic.
- There is no implicit `->` in v0.
- There is no auto-deref in v0.

Future explicit raw-pointer field access should remain visibly unsafe:

```cpp
unsafe {
    int x = (*ptr).x;
}
```

Deref-place field access may be deferred until dereference places exist. Phase 7 should not introduce pointer field sugar merely to imitate C/C++ convenience.

## Field assignment

Syntax:

```cpp
v.x = 10;
```

Rules:

- An assignment target can be a place projection.
- The receiver must be an assignable place.
- The assigned expression type must match the field type.
- Assigning to a field of a non-place temporary is invalid:

  ```cpp
  makeVec().x = 1; // invalid
  ```

- No mutation-through-immutability/reference model is introduced in v0.
- For v0, locals and parameters remain assignable as they are today; field assignment follows the existing assignment spirit.

Phase 7 should avoid inventing a borrow checker or reference mutability model. The goal is storage identity and field projection, not the final aliasing discipline.

## Place model

Current MIR places are effectively whole local storage plus recent pointer/address-of/deref rvalues. Phase 7 should evolve the language and MIR toward explicit place projections:

```text
Place:
  local
  param
  field(base_place, field_id)
  deref(pointer_operand or place)     future/possibly P7 if needed
  index(base_place, index_operand)    future
```

P7 v0 should add field place projection and keep deref/index places future unless an implementation milestone proves they are necessary for a tightly scoped feature.

This place representation is the foundation for later analysis:

- `v` and `v.x` are distinct storage designators related by projection.
- `&v.x` can eventually take the address of a field place.
- `move v.x` can eventually mark only a field as moved if partial moves are allowed.
- Drop insertion can eventually know whether a whole value or field is initialized.
- Diagnostics can describe the exact subobject involved.

## HIR impact

Expected HIR additions:

- struct literal expression;
- field access expression;
- assignment target extension for field places;
- possibly a dedicated `HirPlace` if the current assignment-target union is too narrow.

Recommended HIR shape if it is not too invasive:

```text
HirPlace:
  local
  param
  field(base, field_id)
```

Use `HirPlace` for assignment targets and address-of eligibility, while preserving the simple local/parameter fast path where possible. If the implementation first extends the existing assignment-target union recursively, it should still keep a clear migration path to a real HIR place representation before ownership/drop work begins.

## MIR impact

MIR should prefer places over special-purpose field rvalues.

Expected MIR additions:

```text
MirPlace.field(base, field_id)
```

Struct construction can be represented either as a dedicated rvalue:

```text
StructConstructor(struct_id, field_values)
```

or lowered as assignments into a compiler-created temporary when that better fits existing MIR validation. Either way, the constructor must preserve the field-to-value mapping and must not rely on source order as positional layout.

Field reads should use a field place when places can be operands/copies:

```text
use Place.field(...)
```

Field assignments should lower as ordinary assignments to projected places:

```text
Assign Place.field(local, field) = value
```

Recommendation:

- Prefer MIR place projection over a special field-access rvalue.
- Keep storage identity visible in MIR.
- Prepare for move/drop/storage-state analysis rather than adding a short-lived expression-only lowering path.

## Type checking

Rules:

- A struct literal type is the named struct `TypeId`.
- All fields must be initialized exactly once.
- Missing, duplicate, and unknown fields are rejected.
- Each field initializer is checked against the field's resolved type.
- Field access receiver type must be a struct.
- Field access field name must exist.
- Field assignment target must be an assignable place.
- Field assignment value type must match the field type.
- Struct equality is not automatic in v0.

Struct value flow:

- v0 may allow trivial by-value copies for backend-supported structs.
- v0 may allow struct returns/calls only after the backend path is clean.
- There is no move/drop distinction yet.
- There is no ownership distinction yet.

## Backend-supported value flow

Supported v0 should be staged cautiously:

1. struct local declaration from a labeled literal;
2. field access from local struct values;
3. field assignment to local struct values;
4. struct returns if C backend support is clean;
5. struct parameters and calls if by-value flow remains simple.

Phase 7 should not force a full ABI design. Backend-supported structs are an internal executable representation first. ABI-facing guarantees such as `repr(C)`, field packing, explicit alignment, and foreign-call layout belong to later layout work.

## Interaction with raw pointers

Phase 6 already supports `&local`, `&param`, and read-only `*ptr` in unsafe context. Phase 7 should integrate structs with that work without expanding pointer sugar prematurely.

Rules and direction:

- Address-of a struct local may produce `Struct*` only if the backend can render deterministic struct pointer type names.
- Address-of a field is future unless scoped explicitly:

  ```cpp
  int* px = &v.x;
  ```

- Field access through a pointer is not implicit.
- There is no `->` in v0.
- There is no implicit auto-deref in v0.
- Deref place projections are future or later Phase 7 work.
- Pointer arithmetic remains future work.

## Relationship to concepts/templates

Runtime structs and field access make concepts/templates useful over real user-defined values. Before Phase 7, a generic facility would mostly operate over primitive scalars, enums, and pointers. After Phase 7, examples like these become meaningful targets:

```cpp
concept Equatable<T> {
    bool equals(T left, T right);
}

template<T>
T choose(bool condition, T left, T right) {
    if (condition) {
        return left;
    }
    return right;
}
```

Concepts/templates should follow Phase 7. They do not need to wait for Drop or final ownership analysis, but they do need user-defined runtime values and field access to avoid becoming abstract syntax without motivating data.

## Relationship to ownership/drop

Phase 7 intentionally does not implement Drop, explicit move, ownership checking, or `MaybeUninit`.

Those features need Phase 7's substrate:

- Drop needs fields and struct storage.
- Field drop order needs declaration-ordered field metadata.
- Partial initialization needs field-level storage state.
- Move/drop checking needs place projections.
- Use-after-move diagnostics need precise places.
- `Drop<T>` can later be represented as a compiler-intrinsic concept, but runtime structs and places must exist first.

This preserves the Phase 6 closeout decision: ownership work should wait until it can be implemented over real places instead of only whole bindings.

## Diagnostics plan

Proposed diagnostics:

- `UnknownStructField`
- `DuplicateStructFieldInitializer`
- `MissingStructFieldInitializer`
- `StructFieldInitializerTypeMismatch`
- `FieldAccessNonStruct`
- `UnknownFieldAccess`
- `FieldAssignmentNonPlace`
- `FieldAssignmentTypeMismatch`
- `UnsupportedStructBackendType`
- `RecursiveStructByValue`

Exact diagnostic codes can be assigned during implementation. The important v0 rule is that unsupported runtime struct cases produce explicit diagnostics rather than compiling to misleading C output.

## Fixture strategy

Planned fixtures:

- valid struct literal local;
- valid field access;
- valid field assignment;
- valid field reorder if flexible ordering is implemented;
- invalid unknown field;
- invalid duplicate field initializer;
- invalid missing field initializer;
- invalid wrong field initializer type;
- invalid field access on `int`, `bool`, and enum values;
- invalid assignment to non-place temporary field;
- MIR snapshots for struct literal and field place projection;
- C backend snapshots for struct layout and field access;
- run fixtures returning a field value.

The first fixture slice should prove that a simple `Vec2` value can be constructed, read, mutated, lowered to MIR, emitted to C, compiled by `zig cc`, and executed through the existing run-fixture path.

## Proposed Phase 7 milestone ladder

```text
P7-M0  Runtime structs and places design doc

P7-M1  Struct C layout v0
       - supported field type rendering
       - deterministic struct/field names
       - backend rejection for unsupported fields

P7-M2  Struct literals / constructors
       - labeled field init
       - duplicate/missing/unknown checks
       - local struct values

P7-M3  Field access expressions
       - v.x reads
       - HIR/MIR field place or rvalue path
       - C backend field reads

P7-M4  Field assignment and HIR places
       - v.x = expr
       - assignment target becomes richer place
       - no deref/index places yet

P7-M5  MIR place projections
       - field place projection
       - address-of field if scoped
       - prepares ownership/drop

P7-M6  Struct params/returns/calls
       - by-value struct flow if backend-supported
       - no final ABI guarantee

P7-M7  Runtime/backend fixture stabilization
       - MIR snapshots
       - C snapshots
       - run fixtures

P7-M9  Phase 7 closeout
```

Implementation may combine adjacent milestones if the real code path converges naturally, but the closeout should still report which capabilities are present and which are intentionally deferred.

## Phase 7 non-goals

Phase 7 does not include:

- templates/concepts implementation;
- comptime;
- `Drop<T>`;
- move checking;
- `MaybeUninit`;
- partial initialization tracking;
- borrow checking;
- region/lifetime system;
- pointer field access sugar;
- `->`;
- implicit auto-deref;
- struct methods;
- visibility/export semantics beyond existing declarations;
- `repr(C)` or final ABI guarantees;
- packed/layout attributes;
- arrays, strings, or floats unless already supported elsewhere;
- inheritance, classes, or OOP.

## Close criteria

Phase 7 should be considered successful when:

- supported structs have deterministic C backend layout;
- struct literals create runtime values;
- field access works;
- field assignment works;
- MIR has field place/projection representation or an equivalent representation that preserves storage identity;
- representative run fixtures pass;
- docs clearly position concepts/templates next;
- ownership/drop remains deferred until richer places exist.

## Roadmap note

Current roadmap direction after this document:

```text
Phase 6   closed: unsafe blocks/functions, raw pointers, address-of locals/params, read-only unsafe deref
Phase 7   planned: runtime structs and places
Phase 8   planned: concepts/templates over runtime values
Phase 9   planned: comptime
Phase 10  planned: ownership, move, Drop, MaybeUninit, and richer storage-state analysis
```

This ordering keeps Concept value-oriented and MIR-first while avoiding paperwork ownership semantics before the compiler can name subobject storage.

## P7-M0 status

P7-M0 adds only this design document. It intentionally does not implement lexer, parser, AST, HIR, MIR, backend, fixture, struct literal, field access, field assignment, C layout, ownership, move, Drop, or `MaybeUninit` changes.

## P7-M2 struct literals / constructors

P7-M2 adds the first runtime construction path for supported struct values. In expression position, a simple top-level struct name followed by a brace-delimited labeled initializer constructs a struct value:

```cpp
Vec2 v = Vec2 {
    x: 3,
    y: 4,
};
```

The v0 constructor syntax is intentionally narrow:

- the literal type name must be a simple top-level struct;
- every initializer is labeled as `fieldName: expression`;
- source order is arbitrary, while declaration order remains the backend's deterministic layout/write order;
- every declared field must be initialized exactly once;
- unknown fields, duplicate fields, missing fields, and initializer type mismatches are rejected with stable diagnostics;
- optional trailing commas are accepted.

Struct literals lower through HIR with resolved `StructId`, `TypeId`, and `FieldId` values, then through MIR as a dedicated `struct_constructor` rvalue. The C backend emits constructor assignments as deterministic writes to backend-owned field names, so supported struct locals can be constructed and compiled before field reads exist.

P7-M2 still does not add field access (`v.x`), field assignment (`v.x = ...`), general MIR place projections, address-of-field, deref-field access, struct methods, ownership/drop behavior, or ABI/repr controls.
