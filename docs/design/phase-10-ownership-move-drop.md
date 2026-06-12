# Phase 10 ownership, move, Drop, and storage state

P10-M0 is a documentation-only milestone. It defines Concept's ownership and
storage-state model before implementation begins.

Phase 10 starts after the closed Phase 9 compile-time execution v0 work and the
P9-Fix1 short-circuiting fix. Earlier phases established the required substrate:
Phase 6 introduced unsafe/raw pointer v0, Phase 7 introduced runtime structs and
one-level places, Phase 8 introduced concepts/templates and marker concept
metadata, and Phase 9 kept compile-time execution separate from ownership
semantics.

P10-M0 does not implement compiler behavior. It does not add parser, HIR, MIR,
backend, diagnostic, or fixture changes requiring unsupported language behavior.

P10-M1 adds the first MIR storage-state analysis skeleton. The pass lives in
`src/compiler/mir_storage.zig`, tracks whole-local `Uninitialized`,
`Initialized`, and `Moved` states, initializes parameters as live values, marks
assigned locals initialized, and diagnoses direct reads of uninitialized or moved
storage in MIR. It is wired into the MIR C backend before MIR validation and is
covered primarily by MIR unit tests because source `move` and ordinary
uninitialized local syntax do not exist yet.

P10-M1 remains intentionally conservative: there is no source `move` syntax, no
Drop insertion, no `MaybeUninit`, no branch maybe-state diagnostics, no partial
field state, and no Copy marker integration. Field places currently require the
base local to be usable and are otherwise tracked at whole-local granularity.

P10-M2 adds explicit source-level `move place` expressions. `move` is a reserved
keyword and parses with prefix-expression precedence, so `move a + b` is parsed
as `(move a) + b`. The initial supported move sources are whole local places and
parameter places; moving fields, dereferences, temporaries, calls, literals,
struct literals, enum constructors, compile-time expressions, and target
metadata is rejected before MIR lowering. MIR now carries an explicit move
rvalue, and the storage-state pass treats it as a read of the source place
followed by consumption for non-Copy storage. For this milestone, `int` and
`bool` are intrinsic Copy types and remain initialized after `move`; structs are
non-Copy by default and a successful `move` marks the source local `Moved`.
Reading that source later produces `CON0151 UseAfterMove`, and moving an
uninitialized source reports the existing `CON0150 UseBeforeInitialization`.

P10-M2 still intentionally defers Drop, `MaybeUninit`, branch maybe-state
diagnostics beyond the existing conservative lattice, partial field moves,
`Copy<T>` marker integration, user-defined move/copy constructors, replacement
assignment semantics for Drop types, borrow checking, lifetime analysis,
destructor syntax, and implicit move.

P10-M3 adds copyability model v0. The central copyability predicate lives on the
type store: `int`, `bool`, raw pointer values, and enums are Copy; structs are
non-Copy by default unless a visible marker `impl Copy<StructType>` exists.
Ordinary MIR `Copy(place)` from initialized local and parameter places is now an
implicit copy check. If the source type is non-Copy, the storage pass reports
`CON0154 ImplicitCopyRequiresCopy` at the read and requires either `Copy<T>` or
an explicit `move`. Explicit `move` remains represented as `Move(place)`: it
requires initialized storage, consumes non-Copy local/param places, and remains
non-consuming for Copy types. Fresh values such as struct literals, function
returns held in temps, enum constructors, and literals are not implicit copies
from an existing user place.

P10-M3 still intentionally defers Drop, `MaybeUninit`, branch maybe-state
diagnostics, partial field moves, partial field copyability, replacement
assignment semantics for Drop types, user-defined copy/move constructors,
borrow checking, lifetime analysis, and implicit move.

## Thesis

```text
Phase 10 introduces local storage-state accounting for explicit moves,
deterministic cleanup, and deliberate uninitialized storage.

Concept is not adopting a Rust-style lifetime system.

Concept will not prove every pointer is safe.
Concept will know whether a local/place has been initialized, moved, partially
initialized, or must be dropped.

Ownership in Concept is local, explicit, MIR-visible, and auditable.
```

## Core doctrine

```text
Ownership is not a lifetime proof system.
Ownership is storage-state accounting.

Moves are explicit.
Drops are deterministic.
Uninitialized storage is explicit.
Moved values are gone unless a later moved-state model says otherwise.
```

Concept should catch local use-before-initialization, use-after-move, double-drop,
and maybe-state mistakes where the compiler has enough MIR information. It should
not pretend to prove global pointer lifetime, aliasing, or data-race safety.

## Existing compiler grounding

The current Stage 0 implementation has useful pieces, but not the Phase 10
semantics yet.

- HIR has local and parameter identity, initialized local declarations, assignment
  statements, one-level field assignment targets, unsafe blocks, templates,
  concepts, marker concept metadata, and compile-time-only function metadata.
- MIR has locals, params, temps, basic blocks, local places, one-level field
  places, copy operands, address-of places, dereference rvalues, struct
  constructors, field access rvalues, branches, loops, returns, and unreachable
  terminators.
- MIR lowering builds ordinary control flow and joins for `if`, `while`, `match`,
  and `try`, but it does not perform storage-state analysis.
- MIR validation checks type and shape legality, not initialized/moved/drop state.
- The HIR checker enforces the executable subset, type rules, unsafe call/deref
  boundaries, `must_use`, concepts/templates, and compile-time evaluation rules,
  but it does not track local ownership state.

This means Phase 10 should add a dedicated MIR-level storage-state pass instead
of hiding ownership rules in the backend or overloading the existing HIR checker.

## Values and places

```text
A value has a type.
A place has storage state.
```

Examples:

```text
Buffer is a type.
local `buffer` is a place.
`buffer` may be initialized, moved, uninitialized, maybe-initialized, or partially initialized.
```

Type checking answers:

```text
what is this?
```

Storage-state analysis answers:

```text
is this place currently usable?
```

The two are related but distinct. A `Buffer` local may still have type `Buffer`
after being moved, but the place is no longer readable or droppable. A
`MaybeUninit<PageTable>` value has an ordinary type, while the storage it exposes
is deliberately not a usable `PageTable` until initialization is explicitly
asserted or transferred.

## Storage states

Phase 10's full design vocabulary is:

```text
StorageState:
  Uninitialized
  Initialized
  Moved
  MaybeInitialized
  MaybeMoved
  PartiallyInitialized
```

Staging:

- P10 early implementation can begin with `Uninitialized`, `Initialized`, and
  `Moved`.
- Later milestones add `MaybeInitialized`, `MaybeMoved`, and
  `PartiallyInitialized`.
- Audit and poison states are future instrumentation, not normal language
  semantics.

Initial meanings:

- `Uninitialized`: storage exists but no value of the place's type is live.
- `Initialized`: storage contains a live value of the place's type.
- `Moved`: the value was consumed; for P10 v0, moved means gone.
- `MaybeInitialized`: control flow reaches the point with initialized state on
  some paths and uninitialized state on others.
- `MaybeMoved`: control flow reaches the point with initialized state on some
  paths and moved state on others.
- `PartiallyInitialized`: aggregate storage has field-level initializedness that
  is not uniformly initialized or uninitialized.

## State joins

Storage-state analysis must join states at control-flow merge points.

```text
Initialized + Initialized => Initialized
Moved + Moved => Moved
Uninitialized + Uninitialized => Uninitialized
Initialized + Moved => MaybeMoved
Initialized + Uninitialized => MaybeInitialized
Partials join field-wise later
```

`MaybeMoved` and `MaybeInitialized` are not silently usable states. Reading or
dropping such a place is an error unless later control flow resolves the state.
`return`, `panic`, and unreachable paths should not contribute to joins.

## Explicit move

Source direction:

```cpp
Buffer a = createBuffer();
Buffer b = move a;

use(a); // error: use after move
```

Rules:

- `move place` consumes the source place.
- The source place becomes `Moved`.
- A moved place may not be read.
- A moved place may not be dropped.
- A moved place may be reinitialized if assignment/reinitialization semantics
  permit.
- For P10 v0, moved means gone.
- `moved_state` is future work and should not be implemented early.

The move source must be a place, not an arbitrary temporary:

```cpp
Buffer b = move makeBuffer(); // invalid unless a later rule explicitly permits this form
```

The initial useful target is whole locals and parameters. Field moves and partial
moves should wait until partial field state is deliberately designed.

## Implicit copy and explicit move

`int` and `bool` are copyable intrinsic scalar types. Reading them through
ordinary value flow does not consume the source place.

Future marker concept:

```cpp
marker concept Copy<T>;
```

Eventually, implicit copy of type `T` is allowed when `Copy<T>` holds. Non-Copy
values require `move`.

P10-M3 hardcodes scalar, pointer, and enum categories as copyable and recognizes
visible marker `impl Copy<T>` satisfaction for structs. Later milestones should
integrate richer marker relationships for:

```text
Copy<T>
Trivial<T>
Relocatable<T>
Pod<T>
```

Examples:

```cpp
int a = 1;
int b = a; // copy

Buffer x = createBuffer();
Buffer y = x;      // error if Buffer is not Copy
Buffer z = move x; // ok
```

## Assignment and replacement

Assignment must not accidentally become C++ copy assignment.

Rules and direction:

- Local declaration initializes storage.
- Assignment to initialized scalar/copyable values is value replacement.
- Assignment to uninitialized storage may be reinitialization if the language
  permits it.
- Assignment to a moved place may reinitialize the place if allowed.
- Assignment to a Drop type is not automatically C++ copy assignment.

Suggested P10 v0 rule:

```text
Assignment remains valid for currently supported scalar/copyable values.
Assignment to a non-Copy/Drop type should be rejected until replacement
semantics are deliberately implemented.
```

Future replacement semantics may lower as:

```text
drop old initialized value
initialize new value
```

That lowering must be explicit in MIR and must not imply hidden C++ `operator=`
behavior.

Potential diagnostics:

- assignment to a moved place may be accepted as reinitialization only where the
  language has chosen that rule;
- assignment to uninitialized storage may be accepted only where uninitialized
  storage exists by an explicit rule;
- assignment to non-Copy/Drop values should be rejected until replacement
  semantics and drop ordering are implemented.

## Use before initialization

Existing ordinary locals require initializers. That should remain true for early
P10.

Future direction:

```cpp
int x;
return x; // invalid: use before initialization
```

If ordinary uninitialized local syntax is introduced later:

- reads before initialization are errors;
- drops before initialization are errors;
- branch joins can produce maybe-initialized state;
- ordinary uninitialized locals should not become broadly valid merely because
  `MaybeUninit<T>` exists.

Future example:

```cpp
int x;

if (condition) {
    x = 1;
}

return x; // error: maybe uninitialized
```

This source syntax is not supported today and should be treated as future
direction.

## Use after move

Direct moved use is an error:

```cpp
Buffer b = createBuffer();
consume(move b);
use(b); // error
```

Maybe-moved use after a branch merge is also an error:

```cpp
Buffer b = createBuffer();

if (condition) {
    consume(move b);
}

use(b); // error: maybe moved
```

Reinitialization can restore `Initialized` state if the language permits
reinitialization for that place and type.

## Drop model

Concept keeps deterministic RAII, but it does not use C++ destructor syntax.

Source direction:

```cpp
impl Drop<File> {
    void drop(mut File& file) {
        close(file.handle);
    }
}
```

Rules:

- Drop is deterministic RAII.
- Drop is represented through compiler-intrinsic `Drop<T>`.
- `Drop<T>.drop` must not throw.
- Drop lowering is explicit in MIR.
- Moved values are not dropped.
- Uninitialized values are not dropped.
- Initialized locals drop in reverse successful initialization order.
- Fields drop in reverse declaration order after `Drop<T>.drop` runs.
- Partially initialized objects drop only initialized fields.
- Drop functions should be visible and auditable.

P10 v0 should design Drop now but not implement Drop before storage-state
analysis exists. Drop insertion depends on knowing which places are initialized
on each path.

## Drop order

Rules:

- Locals drop in reverse successful initialization order.
- Early returns drop initialized locals in scope.
- Branch-specific initialized locals drop according to the actual executed path.
- Moved locals do not drop.
- Partially initialized structs drop initialized fields only.
- Drop insertion belongs in the MIR storage-state pass.

Example:

```cpp
File a = openA();
File b = openB();

return 0;
// drop b
// drop a
```

The phrase "successful initialization order" matters for fallible construction
and partial initialization. A local whose initializer did not complete is not
initialized and must not be dropped as a whole value.

## MaybeUninit

Source direction:

```cpp
MaybeUninit<PageTable> table;

unsafe {
    initializePageTable(table.ptr());
}

PageTable ready = table.assumeInit();
```

Rules:

- `MaybeUninit<T>` is explicit uninitialized storage.
- Ordinary uninitialized locals remain rejected unless deliberately introduced
  later.
- `assumeInit` is unsafe.
- Reading uninitialized storage is never safe.
- Dropping `MaybeUninit<T>` does not drop `T` unless initialized state is
  explicitly transferred.
- P10 early implementation may only document `MaybeUninit`; implementation can
  come later.

`MaybeUninit<T>` should make uninitialized storage searchable and auditable. It
is not a loophole that makes ordinary reads from uninitialized memory safe.

## Partial initialization

Partial initialization is especially important for structs.

Future `MaybeUninit` field-writing direction:

```cpp
MaybeUninit<Pair> pair;

unsafe {
    pair.field(.left).write(makeLeft());
    // right not initialized yet
}
```

Ordinary construction path:

```cpp
Pair p = Pair {
    left: makeLeft(),
    right: makeRight(),
};
```

Rules:

- If construction fails after `left` initializes but before `right`, only `left`
  is dropped.
- Field-level initializedness is a future milestone.
- P10 v0 can start with local-level state and defer partial field states.

Partial state should eventually attach to place projections, so the compiler can
distinguish `p.left` from `p.right` instead of treating the whole aggregate as a
single bit.

## Branch and state merge

Example:

```cpp
Buffer b = create();

if (cond) {
    consume(move b);
} else {
    use(b);
}

use(b); // error: maybe moved
```

Storage-state analysis must join states at control-flow merge points. The MIR
lowerer already creates branch and join blocks for `if` and `match`; the Phase 10
pass should reason over those blocks rather than relying on source nesting.

Join diagnostics should be delayed until a maybe-state is used or dropped, unless
an implementation milestone deliberately chooses eager diagnostics for clearer
messages. Eager diagnostics must still point at the control-flow cause and the
later use.

## Loops

Loops require conservative fixed-point analysis.

Initial staging:

- P10 early implementation can reject or conservatively handle loops involving
  moves.
- Later milestones should implement loop fixed-point state analysis.

Example:

```cpp
Buffer b = create();

while (condition) {
    consume(move b);
    b = create();
}

use(b); // requires loop analysis
```

The first implementation should not over-promise. It is better to reject
ownership-sensitive loop patterns with a clear diagnostic than to infer an
unsound initialized state.

## MIR storage-state pass

Suggested file names:

```text
mir_storage.zig
mir_ownership.zig
storage_state.zig
```

Responsibilities:

- local initialized/moved state tracking;
- use-before-initialization diagnostics;
- use-after-move diagnostics;
- branch merge state;
- drop insertion;
- partial initialization later;
- maybe-state diagnostics;
- preventing double drop.

Preferred pipeline:

```text
HIR
  -> MIR lowering
  -> MIR storage-state / ownership analysis
  -> MIR validation
  -> backend
```

Division of labor:

```text
HIR checker handles syntax/type facts.
MIR storage-state pass handles place state and drop insertion.
```

This keeps ownership visible over lowered control flow and avoids requiring the
HIR checker to duplicate MIR's block graph.

## Diagnostics

Proposed diagnostic names:

```text
UseAfterMove
MaybeMovedUse
UseBeforeInitialization
MaybeUninitializedUse
MoveRequiresPlace
MoveRequiresNonCopyOrAllowedMove
ImplicitCopyRequiresCopy
DropRequiresInitialized
DoubleDrop
AssignToMovedPlace
AssignToNonCopyRequiresReplacement
PartialInitializationUnsupported
MaybeUninitAssumeInitRequiresUnsafe
DropFunctionMustNotThrow
InvalidDropImpl
```

Exact `CON` codes should be assigned during implementation.

Diagnostic principles:

- diagnose at the use, drop, move, or assignment that violates the state rule;
- include the earlier move or maybe-state branch when available;
- distinguish direct moved use from maybe-moved use;
- distinguish uninitialized from maybe-uninitialized;
- do not bury ownership errors as generic MIR validation failures.

## Examples

Valid future examples:

```cpp
int a = 1;
int b = a;

Buffer x = createBuffer();
Buffer y = move x;
```

Invalid future use after move:

```cpp
Buffer x = createBuffer();
Buffer y = move x;
use(x);
```

Invalid future maybe-moved branch:

```cpp
Buffer x = createBuffer();

if (condition) {
    consume(move x);
}

use(x);
```

Drop:

```cpp
impl Drop<File> {
    void drop(mut File& file) {
        close(file.handle);
    }
}
```

MaybeUninit:

```cpp
MaybeUninit<PageTable> table;

unsafe {
    initializePageTable(table.ptr());
}

PageTable ready = table.assumeInit();
```

## Non-goals

Phase 10 is not:

- Rust borrow checker;
- full lifetime system;
- region lifetime system;
- global alias analysis proof;
- all use-after-free prevention;
- all data-race prevention;
- C++ copy constructor/destructor model;
- implicit move-from husks;
- hidden allocator/heap model.

P10-M0 does not implement:

- parser changes;
- `move` syntax;
- Drop lowering;
- `MaybeUninit`;
- storage-state pass;
- diagnostics;
- fixtures.

## Proposed Phase 10 milestone ladder

```text
P10-M0  Ownership/move/Drop/MaybeUninit design doc

P10-M1  MIR storage-state analysis skeleton
        - local states: Initialized / Moved
        - simple linear MIR analysis
        - no source syntax changes if possible

P10-M2  Explicit move expression
        - parse/lower/check `move place`
        - use-after-move for locals
        - copyable scalar exemptions

P10-M3  Copyability model
        - int/bool Copy intrinsic
        - non-Copy structs require explicit move
        - begin integration with marker concept Copy<T> if clean

P10-M4  Branch merge states
        - MaybeMoved / MaybeInitialized
        - diagnostics at joins
        - early-return-aware joins

P10-M5  Drop<T> design/lowering v0
        - recognize Drop<T> impls
        - insert drops for initialized locals
        - moved locals not dropped
        - reverse local drop order

P10-M6  Assignment/replacement semantics
        - assignment to moved/uninitialized places
        - reject non-Copy replacement unless explicit replacement semantics exist
        - prepare for Drop replacement

P10-M7  Partial initialization and struct fields
        - field-level initializedness
        - partially initialized structs
        - drop initialized fields only

P10-M8  MaybeUninit scaffold
        - explicit uninitialized storage wrapper
        - unsafe assumeInit direction
        - no ambient uninitialized locals

P10-M9  Stabilization and closeout
```

Implementation may adjust ordering if the real dependency graph demands it, but
the design dependency should stay clear:

```text
storage-state first, move syntax second, Drop later
```

## Close criteria

P10-M0 is successful if:

- the value/place distinction is documented;
- initial storage states and joins are documented;
- explicit move semantics and copyability staging are documented;
- assignment/replacement boundaries are documented;
- use-before-init and use-after-move rules are documented;
- Drop and drop-order direction is documented without implementation;
- `MaybeUninit` and partial initialization are documented as explicit/future;
- branch and loop analysis staging is documented;
- a MIR storage-state pass location and responsibilities are documented;
- non-goals prevent accidental lifetime-system or C++ destructor drift.

P10 implementation should not begin by adding Drop or `MaybeUninit`. It should
begin by making local storage-state accounting real and testable.
