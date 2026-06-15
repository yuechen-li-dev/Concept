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

### P21-M4 status: mutable fixed-array element assignment

P21-M4 teaches Stage 0 to treat fixed-array indexing over assignable places as a mutable place projection. Assignments such as `values[1] = 99;` and nested projections such as `matrix[1][0] = 42;` now type-check and lower through HIR/MIR place machinery. The assigned value is checked against the projected element type, constant indexes keep the existing static out-of-bounds diagnostic, and generated C emits the same bounds guard used by read indexing before the store.

This milestone does not add slices, mutable slices, fixed buffers, `Capacity`, unchecked indexing, pointer decay, array-to-slice conversion, or DragonGod migration.

## P21-M5 implementation note: MIR and C backend value arrays

P21-M5 hardens fixed arrays as Concept values through MIR validation and C backend lowering. The C backend now emits each fixed-array type as a deterministic wrapper struct with a `data` member, for example `int[3]` lowers as a generated `cpt_array_int_array_3`-style typedef containing `int data[3]`. Nested arrays are emitted in type-store order, so the inner wrapper is declared before the outer wrapper.

This preserves the core doctrine: Concept arrays are values, while C arrays are only an implementation detail inside the wrapper. Local declarations, assignments/copies, by-value parameters, by-value returns, struct fields, nested arrays, indexed reads, and indexed writes use wrapper values and `.data[...]` element access rather than relying on C array assignment, C parameter adjustment, or pointer decay.

M5 remains intentionally internal to Concept lowering. It does not add slices, fixed buffers, `Capacity`, unchecked indexing, C ABI array passing, `repr(C)` array layout, heap vectors, generic containers, or DragonGod migration. Arrays of scalar values and arrays nested over scalar arrays are the proven M5 path; broader element layouts continue to follow the backend's existing supported-type rules.

### P21-M6 status: read-only slices

P21-M6 adds the initial read-only `Slice<T>` view model. A slice is a borrowed pointer-plus-runtime-length view over contiguous storage; it does not allocate, own, copy, drop, or extend the lifetime of its backing array. M6 supports array-to-slice conversion only at function call boundaries where the parameter type is explicitly `Slice<T>` and the argument is a fixed array with the same element type. `Len(slice)` returns the runtime slice length, and `slice[index]` is read-only with runtime bounds checks using the stable panic reason `Concept slice index out of bounds`.

Explicit local `Slice(values)` construction, slice returns, slice fields, mutable slices/`MutSlice<T>`, fixed buffers, `Capacity`, range slicing syntax, C ABI guarantees, and DragonGod migration remain deferred. Slice element assignment is rejected in M6.

### P21-M7 status: `FixedBuffer<T, N>` v0

P21-M7 adds the first bounded appendable value-storage abstraction: `FixedBuffer<T, N>`. The type carries the element type and positive literal capacity in semantic type identity, so `FixedBuffer<int, 4>`, `FixedBuffer<int, 5>`, and `FixedBuffer<bool, 4>` are distinct. Stage 0 represents fixed buffers as owned value storage with generated C wrapper structs containing fixed-array `storage` plus an `int count`; no heap allocation, allocator, vector growth, or dynamic capacity is introduced.

The implemented v0 surface is `fixedBufferEmpty<T, N>()`, `fixedBufferAppend(buffer, value)`, read indexing with `buffer[index]`, `Len(buffer)`, and `Capacity(buffer)`. Empty construction sets `count` to zero. Append writes into `storage.data[count]`, then increments `count`, and emits the stable runtime panic reason `Concept fixed buffer capacity exceeded` when full. Read indexing checks against the initialized range `[0, count)`, not capacity, and emits `Concept fixed buffer index out of bounds` on failure. `Capacity(array)` is also accepted for fixed arrays.

M7 intentionally does not add heap vectors, allocators, generic collection algorithms, iterator support, mutable slices, fixed-buffer-to-slice conversion, direct `buffer[i] = value` assignment, C ABI guarantees for fixed buffers, serialization, unchecked indexing, or DragonGod migration. V0 coverage is centered on backend-supported value elements such as scalars and nested fixed-array-shaped storage.

### P21-M8 status: DragonGod fixed-slot migration spike

P21-M8 migrates the append/read/count-only DragonGod fixed-slot subsystems onto `FixedBuffer<T, 4>` without expanding the FixedBuffer language surface. `TraceRecorder`, `EventBus`, `ReplayLog`, and `AutomatonGraph` now use fixed-buffer storage in the source tree and have backend fixtures that assert the generated fixed-buffer wrapper shape, absence of the old explicit slot fields, and absence of heap/vector/runtime-hook artifacts.

Subsystem-specific panic strings are preserved by checking `Len(buffer) >= Capacity(buffer)` before appending and by checking the initialized length before indexed reads. This keeps `DragonGod.Trace capacity exceeded`, `DragonGod.Events capacity exceeded`, `DragonGod.Replay capacity exceeded`, `DragonGod.Graph capacity exceeded`, and the existing subsystem index/root lookup strings rather than leaking the lower-level `Concept fixed buffer ...` panic reasons.

M8 deliberately leaves update/pop-heavy systems unchanged. Memory still needs direct element assignment or replace-at plus find/update by key. AutomataStack still needs pop/remove-last and replace-top or mutable indexing. ActuatorHost still needs find/update by id and replace-at for complete/fail status mutation. EventBus and AutomataGraph still expose hand-unrolled scans because M8 does not add iterators, range slicing, or fixed-buffer-to-slice conversion.

## P21-M9 closeout

Phase 21 is closed after P21-M8. P21-M9 is a closeout, documentation, and inventory milestone; it does not add compiler behavior, runtime behavior, language features, FixedBuffer mutation helpers, a formatter, or Phase 22 implementation work.

### Milestone summary

| Milestone | Status | Result |
| --- | --- | --- |
| P21-M0 | Closed | Design doc |
| P21-M1 | Closed | Fixed-size array type syntax |
| P21-M2 | Closed | Array literals |
| P21-M3 | Closed | Read indexing and `Len(array)` |
| P21-M4 | Closed | Mutable array element assignment |
| P21-M5 | Closed | Array value lowering and C backend wrappers |
| P21-M6 | Closed | Read-only `Slice<T>` |
| P21-M7 | Closed | `FixedBuffer<T, N>` v0 |
| P21-M8 | Closed | DragonGod FixedBuffer migration spike |
| P21-M9 | Closed | Closeout and formatting inventory |

### Final implemented surface

- Fixed-size array type syntax: `T[N]`.
- Positive integer literal lengths only.
- Array literals: `[1, 2, 3, 4]`.
- Nested arrays and nested array literals.
- Read indexing for arrays.
- Mutable fixed-array element assignment.
- `Len(array)`.
- Array wrapper value representation in the C backend.
- Array assignment/copy.
- Array parameters.
- Array returns.
- Array struct fields.
- Read-only `Slice<T>`.
- Array-to-slice conversion at function-call boundaries when the parameter is explicitly `Slice<T>`.
- `Len(slice)`.
- Read-only slice indexing.
- `FixedBuffer<T, N>`.
- `fixedBufferEmpty<T, N>()`.
- `fixedBufferAppend(buffer&, value)`.
- Initialized-range fixed-buffer read indexing.
- `Len(buffer)`.
- `Capacity(buffer)`.
- `Capacity(array)`.
- DragonGod migration spike for `TraceRecorder`, `EventBus`, `ReplayLog`, and `AutomataGraph`.

### Deferred surface

- `MutSlice<T>`.
- Slice returns and local slice construction.
- Fixed-buffer element assignment.
- Fixed-buffer-to-slice conversion.
- Fixed-buffer replace-at, pop/remove-last, find, and update helpers.
- Heap vectors and allocator-backed growable containers.
- Iterators and range ergonomics.
- Generic containers and collection algorithms.
- Full DragonGod migration of Memory, AutomataStack, ActuatorHost, and any future update/pop-heavy subsystems.

### FixedBuffer operations still needed

P21-M8 preserves the missing-operation inventory exposed by DragonGod. These are roadmap inputs, not P21-M9 implementation tasks.

| Missing operation | Blocked DragonGod use |
| --- | --- |
| `fixedBufferSet(buffer&, index, value)` or indexed assignment support for `FixedBuffer` | `Memory` update-existing slot by key; `ActuatorHost` complete/fail by id; `AutomataStack` top mutation and replace-top paths. |
| `fixedBufferReplaceAt(buffer&, index, value)` | `Memory` update-existing-key semantics; `ActuatorHost` status/reason replacement; `AutomataStack` replace-top. |
| `fixedBufferPop(buffer&)` / remove-last | `AutomataStack` pop. |
| `fixedBufferFindIndex` / find by predicate | `Memory` key lookup/update; `ActuatorHost` id lookup/update; `AutomataGraph` lookup ergonomics. |
| `fixedBufferUpdateWhere` or equivalent update-by-key/predicate pattern | `Memory` and `ActuatorHost` need in-place mutation without duplicating keys/ids. |
| Fixed-buffer-to-slice conversion | Shared read-only helper APIs over initialized storage. |
| Iteration/range ergonomics | `EventBus` and `AutomataGraph` still use hand-unrolled scans even after migration. |

The successful migrations were append/read/count-shaped. The intentionally unchanged systems remain unchanged because appending duplicate records or faking pop/update semantics would be a regression rather than progress.
