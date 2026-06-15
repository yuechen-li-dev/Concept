# Phase 22: Option, Result, and bounded collection mutation

**Status:** P22-M0 design milestone. No compiler behavior is implemented by this document.

## Core doctrine

Absence is not failure.

Recoverable operation failure is not panic.

Panic is not collection control flow.

Option represents possible absence.

Result represents recoverable operation outcome.

Fallible functions remain the function-level explicit failure channel.

Bounded collections need honest mutation APIs.

FixedBuffer remains bounded value storage, not a heap vector.

No hidden allocation.

No heap vectors in Phase 22.

## 1. Motivation

Phase 21 made contiguous storage real. Concept now has fixed-size arrays `T[N]`, array literals, nested arrays, read indexing, mutable array element assignment, `Len(array)`, array value lowering with C backend wrapper structs, array assignment/copy, array parameters, array returns, array struct fields, read-only `Slice<T>`, array-to-slice call-boundary conversion, `Len(slice)`, read-only slice indexing, `FixedBuffer<T, N>`, `fixedBufferEmpty<T, N>()`, `fixedBufferAppend(buffer&, value)`, initialized-range fixed-buffer reads, `Len(buffer)`, `Capacity(buffer)`, and `Capacity(array)`.

P21-M8 then migrated the DragonGod subsystems that matched append/read/count patterns. Trace, Events, Replay, and Graph metadata proved that `FixedBuffer<T, N>` is the right bounded value-storage primitive for many kernel-shaped collections.

The remaining DragonGod fixed-slot systems could not migrate cleanly because `FixedBuffer<T, N>` lacks mutation and search operations. The blockers are not accidental details:

- `Memory` needs find/update existing slot by key.
- `AutomataStack` needs push/pop/top/replace-top.
- `ActuatorHost` needs find/update existing slot by id.
- `EventBus` and `AutomataGraph` need better iteration/search ergonomics even after their append/read/count migration.

The obvious missing operations need better return semantics:

- `pop` needs to represent an empty buffer without inventing a fake `T`.
- `last`/`top` needs to represent no last element.
- `replaceAt`/`set` needs to report out-of-bounds indexes in recoverable paths.
- `findIndex` needs to represent no match.
- `tryAt` needs to represent missing initialized elements without exposing reserved storage.

Sentinel values are wrong because every integer, enum value, struct value, and pointer-like value may be valid domain data. `bool + out parameter` is C-ish and ugly; it makes ordinary value flow depend on mutation side channels. Panic-only APIs are also wrong for ordinary collection control flow. Empty stacks, missing keys, and full bounded buffers are often expected cases in kernel and runtime code.

Therefore Phase 22 introduces `Option<T>`, `Result<T, E>`, and bounded collection mutation APIs together. The return types and the collection operations are coupled: the right APIs cannot be designed honestly without the right value-level absence and recoverable-outcome types.

## 2. Non-goals

Phase 22 deliberately defers:

- heap vectors;
- growable `Vec<T>`;
- allocators;
- hash maps;
- dictionaries;
- generic collection algorithms;
- full iterator framework;
- closures/lambdas;
- generic predicate functions if callable values are not ready;
- function pointers unless already part of a later phase;
- strings/formatting;
- serialization;
- async/scheduler;
- DragonGod dynamic runtime;
- replacing existing function-level failure model;
- Rust-style wholesale error model rewrite.

## 3. `Option<T>`

The ideal Concept surface is a generic payload enum:

```cpp
enum Option<T> {
    Some(T value),
    None,
};
```

Examples:

```cpp
Option<int> maybe = Option<int>::Some(3);
Option<int> none = Option<int>::None;
```

If payload enum constructors use the existing shorter spelling, the surface may instead be:

```cpp
Option<int> maybe = Some(3);
Option<int> none = None<int>();
```

The design intent is independent of the exact constructor spelling. The Stage 0 implementation should choose the smallest surface consistent with current generic enum support and existing payload-constructor rules, while preserving the ideal source direction for later cleanup.

Rules:

- `Option<T>` is a normal value type.
- `Some(value)` stores a value of `T`.
- `None` stores no `T`.
- `Option<int>` and `Option<float>` are distinct types.
- `Option<T>` participates in move/copy/drop according to `T`.
- `Option<T>` does not allocate.
- Matching on `Option<T>` should be exhaustive.

Example:

```cpp
match maybeIndex {
    Some(index) => return index;
    None => return -1;
}
```

If generic payload enums are not yet implementable in the desired form, Phase 22 should still design the surface and use a conservative implementation path: compiler-known `Option<T>` lowering, restricted standard declarations, or staged intrinsic construction are acceptable temporary Stage 0 techniques if they keep user-visible semantics stable and do not add hidden allocation.

`Option<T>` is for absence:

- no matching element;
- no last element;
- empty pop;
- missing lookup.

It is not the function-level failure channel.

## 4. `Result<T, E>`

The conceptual surface is:

```cpp
enum Result<T, E> {
    Ok(T value),
    Err(E error),
};
```

Rules:

- `Result<T, E>` is a normal value type.
- `Ok(value)` stores success.
- `Err(error)` stores typed recoverable error.
- `Result<T, E>` does not replace fallible functions.
- `Result<T, E>` is useful when failure is data and should be handled locally.
- Matching on `Result<T, E>` should be exhaustive.
- `Result<T, E>` does not allocate.

Example:

```cpp
Result<int, BufferError> popped = fixedBufferTryPop(buffer&);

match popped {
    Ok(value) => return value;
    Err(error) => return 0;
}
```

`Result<T, E>` is for recoverable operation outcomes:

- failed append due to full buffer;
- failed set due to out-of-bounds index;
- failed replace due to missing slot;
- typed buffer errors.

## 5. Relationship to existing fallible functions

Concept already has function-level fallibility. `Option<T>` and `Result<T, E>` are value-level types. They do not replace the existing failure model, and the existing failure model should not be stretched to represent ordinary optional values.

Recommended doctrine:

```text
Use Option when absence is expected.

Use Result when the operation completed as a recoverable outcome with either success or typed failure.

Use function-level failure for operations whose contract cannot be completed and should propagate through the caller’s failure channel.

Use panic for violated invariants or programmer errors.
```

Examples:

- `fixedBufferFindIndex` returns `Option<int>`.
- `fixedBufferTryAppend` returns `Result<int, BufferError>` or `Result<void, BufferError>` depending on generic `void` support.
- `parseConfigFile(...) ! ParseError` remains fallible because file parsing is a function-level operation.
- `fixedBufferAppend(...)` may remain a panic-on-full convenience API.

## 6. Panic vs Option vs Result

| Situation | Recommended mechanism |
| --- | --- |
| Missing search result | `Option<T>` |
| Empty pop in normal logic | `Option<T>` or `Result<T, BufferError>` |
| Append full in recoverable path | `Result<int, BufferError>` |
| Internal invariant violated | `panic` |
| Parser cannot complete file parse | fallible function |

Panic APIs may exist as convenience or precondition APIs. They are appropriate when the caller has already established the precondition and violation indicates a bug. Try APIs should exist for ordinary recoverable cases and for code that treats full/empty/out-of-bounds as data.

`panic` remains for violated preconditions, violated invariants, or corrupted internal state. It is not collection control flow.

## 7. `BufferError`

Initial bounded-collection error enum:

```cpp
enum BufferError {
    Full,
    Empty,
    IndexOutOfBounds,
    NotFound,
};
```

Rules:

- `BufferError` is small and stable.
- It is for bounded collection operations, not all errors.
- More domain-specific errors remain possible later.
- Domain APIs may translate `BufferError` into domain-specific results when that improves clarity.

## 8. FixedBuffer mutation APIs

Existing Phase 21 surface:

```cpp
fixedBufferEmpty<T, N>()
fixedBufferAppend(buffer&, value)
buffer[index] // read only
Len(buffer)
Capacity(buffer)
```

Phase 22 should design both asserting APIs and try APIs.

Asserting APIs:

```cpp
fixedBufferSet(buffer&, index, value);       // panic on index >= Len
fixedBufferReplaceAt(buffer&, index, value); // alias or clearer name
fixedBufferPop(buffer&);                     // panic on empty, returns T
fixedBufferLast(buffer);                     // panic on empty, returns T
fixedBufferClear(buffer&);
```

Potential indexed assignment:

```cpp
buffer[index] = value;
```

Rules:

- If implemented, indexed assignment checks `index < Len(buffer)`, not capacity.
- Indexed assignment is equivalent to `fixedBufferSet`.
- Indexed assignment must not create new initialized elements.
- Append is the only v0 growth operation.
- `fixedBufferClear(buffer&)` sets initialized length to zero and drops/clears initialized elements according to `T` when drop semantics require it.
- None of these APIs allocate.

Try APIs:

```cpp
fixedBufferTryAt(buffer, index) -> Option<T>
fixedBufferTryLast(buffer) -> Option<T>
fixedBufferTryPop(buffer&) -> Option<T>
fixedBufferTrySet(buffer&, index, value) -> Result<T or void, BufferError>
fixedBufferTryAppend(buffer&, value) -> Result<int, BufferError> // inserted index
```

If generic `void` is not supported, prefer `Result<int, BufferError>` for append and set where success returns the affected index. Returning the affected index is also useful for callers that immediately update auxiliary metadata.

## 9. Find/update APIs

Generic predicate-based find is likely blocked until callable values/function pointers exist. Phase 22 should not over-design this shape prematurely:

```cpp
fixedBufferFindIndex(buffer, predicate)
```

M22 v0 should prioritize index-based helpers:

```cpp
fixedBufferSet(buffer&, index, value)
fixedBufferReplaceAt(buffer&, index, value)
```

Domain-specific find helpers remain in user/kernel code. This keeps the core collection primitive small while still unblocking DragonGod migrations.

Future surfaces may include:

```cpp
fixedBufferFindIndex(buffer, predicate)
fixedBufferUpdateWhere(buffer&, predicate, update)
```

Predicate helpers are deferred because:

- closures/lambdas are not available yet;
- function pointer/callable design belongs to a later phase;
- DragonGod can still write domain-specific find-index helpers returning `Option<int>`.

## 10. FixedBuffer-to-Slice conversion

Phase 22 should design, but may defer implementation of, read-only FixedBuffer-to-Slice conversion:

```cpp
Slice<T> view = fixedBufferSlice(buffer);
```

A future call-boundary conversion may also be possible:

```cpp
int sum(Slice<int> values);

FixedBuffer<int, 4> buffer = fixedBufferEmpty<int, 4>();
fixedBufferAppend(buffer&, 1);
fixedBufferAppend(buffer&, 2);

sum(buffer); // maybe future conversion to Slice<int> over initialized range
```

Rules:

- View length is `Len(buffer)`, not capacity.
- The view is read-only.
- Conversion does not allocate.
- Conversion does not expose uninitialized reserved storage.
- Mutable buffer-to-slice is deferred.

This matters for EventBus and AutomataGraph iteration cleanup later.

## 11. AutomataStack as FixedBuffer stack

Phase 22 should unblock AutomataStack migration without making stacks a language feature.

`fixedBufferAppend` already supports push. AutomataStack still needs:

- `fixedBufferPop`;
- `fixedBufferLast`;
- `fixedBufferSet` or `fixedBufferReplaceAt` for replace-top.

Example:

```cpp
fixedBufferAppend(stack.frames&, frame);
AutomataFrame top = fixedBufferLast(stack.frames);
AutomataFrame old = fixedBufferPop(stack.frames&);
fixedBufferSet(stack.frames&, Len(stack.frames) - 1, newFrame);
```

Stack wrapper APIs may remain DragonGod-specific. `FixedBuffer` should provide primitive bounded-storage operations, not a full stack abstraction.

## 12. Memory / ActuatorHost migration

Phase 22 should unblock Memory and ActuatorHost by making the canonical fixed-slot update pattern readable.

Memory can write a domain-specific helper:

```cpp
memoryFindSlotIndex(Memory memory, MemoryKeyInt key) -> Option<int>
```

ActuatorHost can write the same shape by `ActuationId`.

Updates then use `fixedBufferSet`; new slots use `fixedBufferAppend`:

```cpp
Option<int> existing = memoryFindSlotIndex(memory, key);

match existing {
    Some(index) => fixedBufferSet(memory.slots&, index, newSlot);
    None => fixedBufferAppend(memory.slots&, newSlot);
}
```

This is the canonical P22 DragonGod migration pattern: domain-specific search returning `Option<int>`, followed by primitive FixedBuffer set or append.

## Source formatting requirement

New `.concept` and `.conception` fixtures/examples in Phase 22 should be human-readable.

Use C++-like indentation and line breaks. Avoid compressed one-line function bodies. Prefer one statement per line. Use spaces after commas and around operators. Split multi-field literals across lines when nontrivial.

This requirement is for human maintainability, even if LLMs can read compressed code. Phase 22 does not implement `concept fmt`.

Bad:

```cpp
int main(){FixedBuffer<int,4> b=fixedBufferEmpty<int,4>();fixedBufferAppend(b&,1);return b[0];}
```

Good:

```cpp
int main() {
    FixedBuffer<int, 4> buffer = fixedBufferEmpty<int, 4>();

    fixedBufferAppend(buffer&, 1);

    return buffer[0];
}
```

## 14. Diagnostics

Reserve the following diagnostic names for Phase 22, adjusting numeric assignments only if an existing repo convention conflicts:

```text
CON0500 OptionRequiresValueType
CON0501 ResultRequiresValueAndErrorTypes
CON0502 OptionMatchMustHandleNone
CON0503 ResultMatchMustHandleErr
CON0504 FixedBufferSetTypeMismatch
CON0505 FixedBufferSetIndexOutOfBounds
CON0506 FixedBufferPopEmpty
CON0507 FixedBufferLastEmpty
CON0508 FixedBufferTryAppendFull
CON0509 FixedBufferTrySetOutOfBounds
CON0510 FixedBufferPredicateHelpersDeferred
```

Runtime panic strings:

```text
Concept fixed buffer index out of bounds
Concept fixed buffer capacity exceeded
Concept fixed buffer pop from empty buffer
Concept fixed buffer last of empty buffer
```

## 15. Implementation milestone plan

Preferred Phase 22 milestone split:

```text
P22-M0  Design doc: Option, Result, and bounded collection mutation
P22-M1  Option<T> type and payload construction
P22-M2  Option<T> match/use and standard helpers
P22-M3  Result<T, E> type and payload construction
P22-M4  Result<T, E> match/use and BufferError
P22-M5  FixedBuffer set/replace/last/pop/clear asserting APIs
P22-M6  FixedBuffer try APIs using Option/Result
P22-M7  DragonGod remaining migration: Memory, AutomataStack, ActuatorHost
P22-M8  Human-readable fixture cleanup pass
P22-M9  Closeout
```

If implementation scope proves too broad, split the work as:

- Phase 22: `Option<T>` / `Result<T, E>`.
- Phase 23: bounded collection mutation.

The preferred design keeps them together because P21-M8 showed they are coupled. The remaining DragonGod migration blockers are not only missing operations; they are missing operations with honest return semantics.

## 16. Non-goals for Phase 22

Explicitly deferred again for clarity:

- heap vectors;
- general `Vec<T>`;
- allocators;
- hash maps;
- iterators framework;
- closures/lambdas;
- predicate APIs requiring callable values;
- string formatting;
- serialization;
- DragonGod dynamic runtime;
- async/scheduler;
- `concept fmt`.

## 17. Roadmap and coverage update requirement

P22-M0 should update the project roadmap and coverage notes to mark Phase 22 as design/in-progress. The update should state that Phase 21 closeout led directly to Phase 22 by proving append/read/count migrations and isolating the remaining fixed-buffer mutation/search friction.

Roadmap notes should keep the boundary explicit: Phase 22 designs `Option<T>`, `Result<T, E>`, `BufferError`, and bounded FixedBuffer mutation/try APIs. It does not implement heap vectors, allocators, callables, generic predicate helpers, dynamic DragonGod runtime, or compiler behavior in M0.

## P22-M1 implementation status

P22-M1 is implemented with a deliberately small Stage 0 surface. `Option<T>` is a compiler-known generic value type with exact element-type identity and display spelling `Option<T>`. The accepted construction helpers are `optionSome<T>(value)` and `optionNone<T>()`; qualified payload constructor syntax such as `Option<int>::Some(3)` remains deferred until generic qualified constructors are introduced cleanly.

`optionSome<T>(value)` requires a non-`void` value type argument and checks that `value` has exactly type `T`. `optionNone<T>()` requires the explicit type argument because untyped `None` has no payload from which to infer `T`. The minimal inspection helpers added for executable fixtures are `optionIsSome(value)` and `optionOr(value, fallback)`. Full matching ergonomics, `optionUnwrap`, broader helper libraries, and target-typed constructor inference remain P22-M2 work.

HIR and MIR carry `Option.Some`, `Option.None`, `optionIsSome`, and `optionOr` as explicit value operations. The C backend emits deterministic tagged structs with `int tag` and a payload `value` field, initializes `Some` with tag `1`, initializes `None` with tag `0` and a deterministic zero payload, and uses no heap allocation or hidden failure channel.

Deferred after M1: `Result<T, E>`, `BufferError`, FixedBuffer try/mutation APIs, DragonGod Memory/AutomataStack/ActuatorHost migration, predicate helpers, callable values, and new fallible-function semantics.
