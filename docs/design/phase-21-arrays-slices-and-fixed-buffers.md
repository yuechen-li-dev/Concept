# Phase 21: Arrays, slices, and fixed buffers

**Status:** P21-M0 design milestone. No compiler behavior is implemented by this document.

## Core doctrine

Contiguous storage is a language-level primitive, not a library afterthought.

Fixed-size arrays are owned value storage.

Slices are borrowed views, not owners.

Fixed buffers are bounded appendable storage, not heap vectors.

No hidden allocation.

No implicit lifetime magic.

No growable heap containers in Phase 21.

## 1. Motivation

DragonGod Phase 20 repeatedly needed small collections. Memory v0, AutomataStack v0, Actuation v0, Events v0, Trace v0, Replay v0, Persistence/checkpoint v0, and AutomataGraph v0 all converged on fixed-capacity storage, but the language did not yet have a native way to express contiguous fixed storage.

The current workaround was explicit fixed fields such as `slot0`, `slot1`, `slot2`, and `slot3`. That workaround was useful as a stress test: it forced the compiler, MIR, C backend, ownership analysis, struct value semantics, and module system to carry real workloads without a container abstraction hiding weaknesses. It should not become the Concept idiom.

Current DragonGod-style friction looks like this:

```cpp
struct EventBus {
    int nextId;
    int count;
    EventSlot slot0;
    EventSlot slot1;
    EventSlot slot2;
    EventSlot slot3;
};
```

The desired eventual shape is:

```cpp
struct EventBus {
    int nextId;
    FixedBuffer<EventSlot, 4> slots;
};
```

Concept needs first-class contiguous storage before DragonGod can mature beyond a proof-of-concept kernel. Arrays, slices, and fixed buffers are foundational for memory slots, event streams, trace logs, replay logs, AutomataGraph nodes, actuation slots, future strings and byte slices, future serialization, and future dynamic graph/runtime work.

## 2. Non-goals

Phase 21 deliberately defers:

- heap vectors;
- growable `Vec<T>`;
- allocators;
- arena-backed containers;
- hash maps;
- dictionaries;
- strings as a full feature;
- UTF-8 and string formatting;
- iterators as a full abstraction framework;
- comprehensions;
- SIMD/vector math arrays;
- generic collection algorithms;
- ownership-polymorphic containers;
- dynamic array resizing;
- hidden allocation;
- borrow checker or lifetime inference beyond existing Concept rules;
- DragonGod subsystem migrations.

## 3. Fixed-size arrays

Phase 21 should add fixed-size owned arrays with C-lineage syntax:

```cpp
int[4] values;
```

Decision: confirm `T[N]` as the Phase 21 v0 array type syntax. It is visually C-lineage, keeps the element type first, does not borrow Rust's `[T; N]` surface, and reads naturally beside Concept's existing return-type-first and field-first declarations.

Rules:

- `T[N]` is an owned value type containing exactly `N` contiguous elements of `T`.
- `N` must be a compile-time integer constant.
- v0 rule: `N >= 0`. Zero-length arrays are allowed only where backend lowering can preserve deterministic size/layout; otherwise implementation may initially reject them behind `CON0403` until backend support is proven.
- Element type `T` must be storable under existing Concept rules.
- Arrays have value semantics consistent with Concept structs.
- Array assignment copies or moves elements according to the existing ownership, move, and `Copy` rules.
- Arrays participate in ownership, move, and drop according to Phase 10 rules.
- Arrays of non-copy or move-only element types follow the existing ownership model: implicit copy is rejected when the element type is not copyable, explicit move is required where appropriate, and moved arrays leave the source unusable.
- Arrays of `ManualInit<T>` should be possible later, but Phase 21 should not over-design partial initialization before the compiler has a real use case.

Example:

```cpp
int[4] values = [1, 2, 3, 4];

return values[2];
```

## 4. Array literals

Phase 21 should add array literals:

```cpp
[1, 2, 3, 4]
```

Rules:

- Array literal length is known statically.
- All elements must have the same type or be coercible to a single element type under existing coercion rules.
- Assignment target can determine array type:

  ```cpp
  int[4] values = [1, 2, 3, 4];
  ```

- Literal length must match the target array length.
- If no target type exists, the literal type may be inferred as `T[N]` once the existing inference path can represent fixed arrays.
- Empty array literal is deferred unless type-annotated:

  ```cpp
  int[0] empty = [];
  ```

  This form is accepted only if zero-length arrays are supported in the same implementation milestone.

Suggested diagnostics:

```text
CON0400 ArrayLiteralLengthMismatch
CON0401 ArrayLiteralElementTypeMismatch
CON0402 ArrayLengthMustBeCompileTimeConstant
CON0403 ArrayLengthMustBeNonNegative
```

The `CON0400` range is unused by the current closed phases and is reserved here for Phase 21 contiguous-storage diagnostics.

## 5. Indexing

Phase 21 should add indexing for arrays, slices, and fixed buffers:

```cpp
values[0]
values[i]
```

Rules:

- Index expression type must be `int` or another accepted integer type if Concept later broadens integer indexing.
- Fixed array indexing yields an element place for mutable arrays in assignable context.
- Fixed array indexing yields the element value in rvalue context.
- Compile-time constant out-of-range indexes should be diagnosed statically when possible.
- Runtime out-of-range indexes use inserted bounds checks in safe code.

V0 semantics: safe indexing performs a runtime bounds check with a stable panic reason.

Stable panic strings:

```text
Concept array index out of bounds
Concept slice index out of bounds
Concept fixed buffer index out of bounds
```

Unchecked indexing is deferred to unsafe. A future surface may be:

```cpp
unsafe {
    values.uncheckedAt(i)
}
```

Phase 21 should not add unchecked indexing until method syntax, unsafe library intrinsics, and MIR unsafe markers can make it explicit and auditable.

## 6. Length

Length access should use the existing built-in style rather than adding property syntax:

```cpp
Len(values)
```

Decision: use `Len` for Phase 21 v0. Do not add `.length` until Concept standardizes properties or field-like compiler properties.

Rules:

- `Len(T[N]) == N`.
- For arrays, `Len(values)` returns a compile-time known integer value but can be used as an `int` expression.
- `Len(Slice<T>)` returns the slice runtime length.
- `Len(FixedBuffer<T, N>)` returns the current runtime count.

## 7. Mutable indexing / assignment

Element assignment should be ordinary place assignment:

```cpp
int[4] values = [1, 2, 3, 4];
values[1] = 99;
```

Rules:

- The array variable must be mutable or assignable under current assignment rules.
- The index target is a place.
- Assigned value type must match or coerce to the element type under existing rules.
- Assignment to an element of a temporary array is invalid:

  ```cpp
  makeArray()[0] = 1; // invalid
  ```

- Element assignment should reuse existing place machinery and extend it with an index projection.

Suggested diagnostics:

```text
CON0404 ArrayIndexTargetNotAssignable
CON0405 ArrayElementTypeMismatch
CON0406 ArrayIndexRequiresArrayOrSlice
```

## 8. Slices

Phase 21 should introduce borrowed slice/view types with C++-lineage generic spelling:

```cpp
Slice<int> view;
```

V0 model:

```text
Slice<T> = pointer + length view over contiguous T storage
```

Rules:

- Slices are not owners.
- Slices do not allocate.
- Slices do not extend lifetime.
- Slices can be passed to functions by value as a pointer+length view.
- Slice element access is bounds-checked.
- Slice length is runtime.

Decision: Phase 21 doctrine defines read-only `Slice<T>` first and reserves `MutSlice<T>` as the explicit mutable slice surface. `mut Slice<T>` is rejected as the primary design because it is too easy to confuse mutability of the slice value with mutability of the viewed elements. `MutSlice<T>` is louder and mirrors the existing Concept preference for exposing essential aliasing difficulty.

Creation should avoid range syntax in v0 unless the parser already supports it cleanly. Preferred v0 surface:

```cpp
Slice<int> view = Slice(values);
```

Additionally, Phase 21 should support implicit array-to-slice conversion only at function call boundaries where the parameter type is explicitly `Slice<T>`:

```cpp
int sum(Slice<int> values) { ... }

int[4] xs = [1, 2, 3, 4];
return sum(xs);
```

This keeps ordinary assignment explicit while making the common parameter-passing path usable. No hidden allocation or lifetime extension is implied.

## 9. Slice parameters

Example:

```cpp
int sum(Slice<int> values) {
    int total = 0;
    for i in 0..Len(values) {
        total = total + values[i];
    }
    return total;
}
```

Rules:

- Arrays can be passed to slice parameters through the explicit call-boundary array-to-slice conversion.
- Slices are passed by value as pointer+length.
- Passing a slice does not transfer ownership of array storage.
- Passing an array to a slice parameter does not allocate.
- Returning a slice to local array storage must eventually be rejected.
- If lifetime checking is insufficient in v0, slice creation and returning should be restricted conservatively: allow array-to-slice conversion for call arguments, allow explicit `Slice(values)` for locals whose use does not escape the containing function, and reject returning slices created from local arrays.

## 10. `FixedBuffer<T, N>`

`FixedBuffer<T, N>` is bounded appendable value storage. It exists to replace explicit slot fields in DragonGod fixed-capacity subsystems while staying out of heap-vector territory.

Purpose:

- not a heap vector;
- owned value storage;
- capacity known at compile time;
- current length tracked at runtime;
- storage equivalent to `T[N] + count`.

Ideal future surface:

```cpp
FixedBuffer<EventSlot, 4> slots = FixedBuffer<EventSlot, 4>::Empty();

slots.Append(event);
EventSlot first = slots[0];
int count = Len(slots);
int capacity = Capacity(slots);
```

Realistic Phase 21 v0 surface should prefer compiler-known helper functions until static methods and method calls are ready for this container shape:

```cpp
FixedBuffer<EventSlot, 4> slots = fixedBufferEmpty<EventSlot, 4>();
fixedBufferAppend(slots&, event);
EventSlot first = fixedBufferAt(slots, 0);
int count = Len(slots);
int capacity = Capacity(slots);
```

If ordinary indexing places are implemented for buffers in P21-M7, `slots[0]` should become the preferred surface and `fixedBufferAt` can remain a lowering/helper detail.

Rules:

- Append succeeds until capacity.
- V0 overflow behavior: panic with stable string `Concept fixed buffer capacity exceeded`.
- Panic is chosen for v0 because Result/Option ergonomics are Phase 22 work.
- Indexing is valid only for `0 <= index < Len(buffer)`, not for all reserved capacity.
- `FixedBuffer<T, N>` owns initialized elements.
- `FixedBuffer<T, N>` drops only initialized elements in `[0, count)`.

## 11. Capacity

Capacity access should use a compiler-known built-in/function:

```cpp
Capacity(values)
Capacity(buffer)
```

Rules:

- Arrays: capacity equals length and `Capacity(T[N]) == N`.
- Fixed buffers: capacity is compile-time `N` but returns an `int` expression.
- Slices: `Capacity(Slice<T>)` is not defined in v0. A slice knows length, not backing capacity.

Decision: add `Capacity` for arrays and fixed buffers only.

## 12. Iteration

Phase 21 does not require a full element-iteration abstraction. Existing range-style loops, where available, are enough:

```cpp
for i in 0..Len(values) {
    total = total + values[i];
}
```

Future element iteration can be designed later:

```cpp
for value in values { }
```

Element iteration, iterator traits, adaptors, and collection algorithms are deferred.

## 13. MIR / lowering design notes

Although P21-M0 is docs-only, the implementation shape should be planned now:

- AST type representation gains `ArrayType(element_type, length_expr)` for `T[N]`.
- AST expression representation gains an `ArrayLiteral` node with ordered element expressions.
- HIR type representation gains `FixedArray(element_type, length)` after compile-time length evaluation.
- HIR/MIR place representation gains an index projection.
- MIR value representation gains array values or an aggregate representation capable of carrying fixed arrays.
- MIR place projections should support:

  ```text
  local -> index
  field -> index
  ```

- Bounds checks should lower before backend emission using stable panic reasons.
- C backend lowering may use C arrays inside structs or locals where legal, but Concept value semantics must not be sacrificed.

C limitations matter:

- C arrays are not assignable as values.
- Returning or assigning Concept arrays may require wrapper structs or element-wise helper copies.
- A backend wrapper shape may be necessary:

  ```c
  typedef struct { int data[4]; } cpt_array_int_4;
  ```

The backend should choose predictable Concept semantics over clever C-array decay. Concept arrays must not silently become pointers.

## 14. Ownership / move / drop interaction

- Arrays own their elements.
- Moving an array moves its elements.
- Dropping an array drops initialized elements in deterministic order.
- Copying an array requires copy permission for the element type.
- `FixedBuffer<T, N>` owns its initialized elements and drops only elements up to `count`.
- `Slice<T>` does not own storage and does not drop elements.
- `Slice<T>` is a borrowed view and must not extend the lifetime of the viewed storage.
- `ManualInit<T>` arrays are deferred unless existing machinery makes them straightforward without partial-initialization debt.

## 15. Safety model

- Safe indexing bounds-checks.
- Safe array, slice, and fixed-buffer indexing never performs unchecked pointer arithmetic.
- Unchecked indexing is deferred to explicit unsafe surface.
- Slices must not outlive backing storage.
- If full lifetime analysis is not ready, v0 must restrict slice escaping conservatively rather than accept unsound lifetime magic.
- No hidden allocation is permitted.
- No implicit conversion to raw pointer is permitted except in explicit unsafe or C ABI contexts.
- Array-to-slice conversion is not C array-to-pointer decay; it produces a length-carrying view.

## 16. C ABI interaction

- Arrays in `repr(C)` structs require careful layout rules and should remain rejected until explicitly implemented and tested.
- C ABI array parameter passing is tricky because C function parameters declared as arrays decay to pointers.
- For v0, fixed arrays should remain internal Concept values unless C ABI lowering is explicitly implemented.
- `Slice<T>` ABI is pointer+length and is not necessarily C-compatible unless a `repr(C)` view type is specified.
- Phase 21 should not solve full C ABI array passing.

## 17. DragonGod migration plan

M0 does not migrate DragonGod. After arrays, slices, and fixed buffers are implemented and tested, a later spike should replace explicit fixed-slot fields:

- Memory slots -> `FixedBuffer<MemorySlot, N>`
- AutomataStack frames -> `FixedBuffer<AutomataFrame, N>`
- ActuatorHost slots -> `FixedBuffer<ActuationSlot, N>`
- EventBus slots -> `FixedBuffer<EventSlot, N>`
- TraceRecorder events -> `FixedBuffer<TraceEvent, N>`
- ReplayLog events -> `FixedBuffer<ReplayEvent, N>`
- AutomataGraph nodes -> `FixedBuffer<AutomatonNode, N>`

Suggested later milestone:

```text
P21-M8 DragonGod fixed-slot migration spike
```

## 18. Milestone plan

```text
P21-M0  Design doc: arrays, slices, and fixed buffers
P21-M1  Parser/type representation for fixed-size array types
P21-M2  Array literals and basic type checking
P21-M3  Array indexing, Len, and bounds diagnostics
P21-M4  Mutable array element places and assignment
P21-M5  MIR lowering and C backend array value representation
P21-M6  Slice type and array-to-slice parameter passing
P21-M7  FixedBuffer<T, N> v0
P21-M8  DragonGod fixed-slot migration spike
P21-M9  Closeout
```

## 19. Diagnostics

Reserve the Phase 21 contiguous-storage diagnostic range:

```text
CON0400 ArrayLiteralLengthMismatch
CON0401 ArrayLiteralElementTypeMismatch
CON0402 ArrayLengthMustBeCompileTimeConstant
CON0403 ArrayLengthMustBeNonNegative
CON0404 ArrayIndexTargetNotAssignable
CON0405 ArrayElementTypeMismatch
CON0406 ArrayIndexRequiresArrayOrSlice
CON0407 ArrayIndexMustBeInteger
CON0408 SliceEscapesLocalStorage
CON0409 FixedBufferCapacityExceeded
CON0410 FixedBufferIndexOutOfBounds
```

Runtime panic reasons reserved by this design:

```text
Concept array index out of bounds
Concept slice index out of bounds
Concept fixed buffer index out of bounds
Concept fixed buffer capacity exceeded
```

## 20. Roadmap / documentation updates

P21-M0 updates the main PoC3 roadmap and checkpoint coverage to mark Phase 21 as planned/in-progress design-only. No parser, lexer, semantics, MIR, backend, fixture, or DragonGod migration changes are part of this milestone.

## 21. Tests for M0

Even though this is docs-only, completion requires:

```bash
zig version
zig build test
git diff --check
git status --short
```

## 22. Commit

Commit message:

```text
Add Phase 21 arrays and slices design
```

## Syntax decisions summary

- Confirm fixed array type syntax: `T[N]`, for example `int[4] values;`.
- Confirm array literal syntax: `[1, 2, 3, 4]`.
- Confirm read-only slice spelling: `Slice<T>`.
- Reserve explicit mutable slice spelling: `MutSlice<T>`.
- Confirm fixed buffer spelling: `FixedBuffer<T, N>`.
- Use `Len(...)` and `Capacity(...)`; do not add `.length` in v0.
- Prefer explicit `Slice(values)` for local slice creation and implicit array-to-slice conversion only at explicit slice parameter call boundaries.

## P21-M1 status: fixed array type shell

P21-M1 teaches the Stage 0 compiler that `T[N]` is a type spelling. The parser accepts postfix fixed-array type suffixes after ordinary type names, generic type arguments, raw pointer suffixes, and borrowed-reference suffixes where those syntactic forms are otherwise accepted.

Supported M1 examples:

```cpp
int[4] values;
int[4][2] matrix;
```

Nested postfix array type syntax is interpreted as repeated postfix type construction: `T[N][M]` means an array of `M` elements, each of type `T[N]`.

M1 deliberately accepts positive integer literal lengths only. `int[4]` is accepted. `int[0]`, `int[-1]`, `int[x]`, and `int[1 + 2]` are rejected. General compile-time length expressions remain deferred.

M1 does not implement array literals, indexing, element assignment, bounds checks, `Len`, `Capacity`, `Slice<T>`, `MutSlice<T>`, `FixedBuffer<T, N>`, array-to-slice conversion, DragonGod migrations, or full backend value-copy semantics. DragonGod migration remains deferred until the later Phase 21 migration milestone.

## P21-M2 status: array literals and basic type checking

P21-M2 teaches the Stage 0 compiler to parse expression-position fixed array literals such as `[1, 2, 3, 4]` and lower them into typed HIR array literal expressions. Target-typed local initialization is supported for fixed arrays: `int[4] values = [1, 2, 3, 4];` checks as `int[4]` when the element count and element types match the target array type.

Nested fixed-array literals are supported through recursive target typing where the target type is nested, for example `int[2][2] matrix = [[1, 2], [3, 4]];`. Empty array literals remain rejected with `CON0412` because M1 rejects zero-length arrays. Non-empty array literals can infer their own fixed-array type internally when no target is supplied, but the language still has no general `let`/`auto` local declaration form, so the user-facing M2 path is target-typed initialization.

M2 adds diagnostics for array literal length mismatch (`CON0400`), element type mismatch (`CON0401`), and empty array literal rejection (`CON0412`). It does not implement array indexing, element mutation, bounds checks, `Len`, `Capacity`, slices, fixed buffers, array-to-slice conversion, heap vectors, C ABI array passing, or DragonGod migration. MIR and C backend representation is intentionally conservative and exists only to keep simple literal shapes representable; full array value copy/return/assignment semantics remain deferred to later Phase 21 milestones.

## P21-M3 implementation note

P21-M3 teaches the compiler to read from fixed arrays. It adds postfix expression indexing (`values[0]`, `values[i]`, and repeated nested indexing such as `matrix[1][0]`), semantic checking that the receiver is a fixed array, semantic checking that the index is `int`, and static diagnostics for constant out-of-bounds indexes. Dynamic index reads lower through MIR to C with a guard that calls `cpt_panic("Concept array index out of bounds")` before the C array access when the index is negative or greater than or equal to the fixed length.

`Len(array)` is compiler-known for fixed arrays and returns the fixed length as an `int` expression; nested cases such as `Len(matrix[0])` use the element array length after the first read. Mutable indexed assignment remains explicitly out of scope for M3 and is reserved for M4. Slices, `MutSlice<T>`, `FixedBuffer<T, N>`, `Capacity`, unchecked indexing, unsafe indexing APIs, DragonGod migration, full C ABI array passing, and full array value-copy semantics remain deferred.
