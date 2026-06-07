# Phase 6 unsafe and ownership foundations

P6-M0 is a documentation-only milestone. It defines Concept's direction for unsafe operations, raw pointers, initialization, explicit move, and deterministic Drop/RAII without implementing compiler code.

Phase 6 starts after the closed Phase 5 sum-type/failure work and the closed Phase 5a `decide`/judgment work. It builds on the closed Phase 4 MIR path: MIR owns semantics, and backends own representation.

## Thesis

```text
Phase 6 teaches Concept how to touch memory without lying.

Concept is not Rust-style memory safety.
Concept is auditable systems programming:
the compiler catches local hazards where it can,
and deliberate hazards must be explicit.
```

Concept should make memory power explicit and auditable. Unsafe code is not an alternate language where ordinary checking disappears. Unsafe is a permission boundary for operations whose invariants the compiler cannot prove locally or globally.

Phase 6 therefore chooses this contract:

- Unsafe does not disable parsing, name resolution, type checking, control-flow checking, or unrelated diagnostics.
- Unsafe permits specific otherwise-forbidden operations whose safety invariants are the programmer's responsibility.
- Ownership, initialization, move, and drop semantics should become visible in MIR rather than being hidden backend conventions.
- The compiler should catch local mistakes such as use-before-initialization and use-after-move where feasible.
- The language does not attempt to globally prove all lifetimes, all aliasing, all data-race freedom, or all foreign ABI assumptions.

## PoC3 design-law alignment

Phase 6 follows the PoC3 laws directly.

### Essential difficulty remains visible

Manual memory, raw addresses, aliasing, ABI, layout, allocator choice, interrupt safety, and device memory are real systems-programming concerns. Concept should expose them clearly instead of pretending they are solved by hidden compiler magic.

Examples:

- Raw pointer dereference is written in an `unsafe` context.
- Thread-safety claims are written as `unsafe impl`.
- Layout claims are written as `repr(...)` or a future unsafe marker concept such as `Pod<T>`.
- Allocation in restricted profiles is an explicit effect violation.
- Discarded errors are already made visible by `must_use` and `discard` from Phase 5.

### Accidental difficulty is removed or quarantined

Concept should not reproduce C and C++ hazards where a dangerous operation looks ordinary. Unsafe operations may exist, but the syntax and lowering should leave audit trails. A reviewer should be able to search for `unsafe`, `unsafe impl`, `repr`, `discard`, allocation effects, and marker concepts to find deliberate hazards.

### Deliberate footguns leave fingerprints

Low-level power is allowed. The requirement is evidence. If a program dereferences a raw pointer, asserts a thread-safety property, depends on a layout property, performs allocation in a constrained profile, or intentionally discards a checked failure, the source and MIR should contain a visible fingerprint.

## P6-M1 unsafe surface

P6-M1 adds the first compiler surface for the `unsafe` permission boundary. The lexer reserves `unsafe`; the parser accepts `unsafe { ... }` wherever a statement is allowed; AST and HIR retain an unsafe-block node; semantic lowering lowers the body with ordinary block scope and ordinary statement/expression rules. The HIR checker tracks unsafe-context depth when it enters an unsafe block, and nested unsafe blocks are valid.

P6-M1 also accepts `unsafe` as a function modifier, for example `unsafe int helper() { return 1; }`. The modifier is copied into HIR function metadata. Unsafe function bodies are checked in unsafe context, and calls to unsafe functions require an unsafe context (`CON0070`) so callers must write an unsafe block or be an unsafe function.

This milestone intentionally enables no dangerous operation. Raw pointer types, address-of, dereference, pointer arithmetic, ownership, move, drop, borrow checking, volatile, atomics, address spaces, and effects remain unimplemented. Unsafe blocks lower through MIR like ordinary blocks for now, and MIR/backend representation is unchanged until future unsafe operations need an explicit region fingerprint.


## P6-M2 raw pointer type surface

P6-M2 adds the raw pointer type surface for ordinary type positions. Builtin and nominal pointees can now be written as `int*`, `bool*`, `void*`, and `SomeEnum*`; the semantic type store interns pointer types by pointee so repeated spellings of the same raw pointer share one stable `TypeId`.

Pointer values may flow through the existing checked HIR/MIR path as parameters, return values, local declarations, assignments, and function-call arguments. Type equality remains exact: `int*` and `bool*` are distinct types, and mismatched pointer returns, initializers, assignments, and call arguments are rejected by the existing type-mismatch diagnostics.

The MIR C backend renders supported data pointers directly (`int*`, `void*`, and `cpt_enum_Name*`; `bool*` follows the current bool-as-C-`int` lowering). Struct pointer types may be represented semantically, but C emission still rejects them with the existing unsupported-backend-type diagnostic until struct layout and C naming are completed.

This milestone intentionally does not add address-of, dereference, pointer arithmetic, null literals, ownership, moves, drops, references, borrow checking, volatile, atomics, or address-space qualifiers. Raw pointer dereference remains planned for P6-M3 and will require an unsafe context. Ownership and nullability refinements remain future work.

## Non-goals

Phase 6 v0 explicitly does not include:

- Rust-style borrow checking.
- A region lifetime system.
- Lifetime parameters.
- A global proof that dangling references cannot occur.
- A global proof that data races cannot occur.
- A full effect system implementation.
- State machines.
- Comptime or reflection.
- Templates or concept-solving implementation.
- Volatile, atomics, or address-space implementation.
- Struct layout or ABI finalization.
- An LLVM or native backend.
- Self-hosting.

## No region lifetime system

Concept should not add a hidden region lifetime system to core language semantics.

The reason is structural:

- A local-only region checker is weak. It can reject a few obvious mistakes but cannot express cross-function ownership and storage relationships with enough precision for systems code.
- A cross-function region checker becomes lifetime parameters, region variables, and lifetime propagation through APIs.
- Once that machinery exists, Concept has become borrow-checker-first in practice, even if the syntax differs.

Concept instead prefers explicit mechanisms that are already natural for compiler, runtime, kernel, and bare-metal work:

- Explicit allocator passing.
- Deterministic arena destruction.
- Stable ID handles for long-lived graph and table relationships.
- Temporary pointer/reference views for local access.
- Optional static-analysis tooling later, outside the required core language model.

This does not mean Concept ignores lifetime bugs. It means Phase 6 should catch local, obvious hazards without pretending to solve all global aliasing and lifetime questions.

## References

PoC3 points toward C-lineage references:

```cpp
T&      // non-null read borrow
mut T&  // non-null mutable borrow
```

For Phase 6, references are a future target or limited semantic surface, not a commitment to full borrow checking.

Reference direction:

- A reference is a non-null borrowed view of an existing value.
- A mutable reference is a non-null borrowed view that permits mutation through the view.
- References should be preferred by safe APIs when the callee does not need ownership or nullability.
- Local obvious dangling cases may be rejected later.
- A reference lifetime system is not part of P6 v0.
- No hidden region model should be introduced to make references work.

## Raw pointers

PoC3 points toward raw and eventually owning pointers:

```cpp
T*      // raw nullable pointer
own T*  // owning pointer, future
```

Phase 6 v0 should make raw pointer type semantics explicit before full ownership machinery exists.

Raw pointer direction:

- Raw pointer locals, parameters, and returns should be representable.
- Nullability is part of raw pointer semantics; a `T*` may be null unless refined by a later type or check.
- Dereferencing a raw pointer requires an `unsafe` context.
- Pointer arithmetic requires `unsafe` and is likely future work after initial dereference support.
- Raw pointers exist for FFI, allocators, drivers, runtimes, kernels, bare-metal code, and low-level compiler internals.
- Safe APIs should prefer references, spans, stable IDs, or explicit owner types when possible.
- `own T*` is future direction, not a P6 v0 ownership implementation requirement.

## Unsafe blocks

Unsafe blocks are the primary lexical permission boundary:

```cpp
unsafe {
    byte value = *ptr;
}
```

Rules:

- `unsafe` is a lexical permission boundary.
- `unsafe` allows otherwise-forbidden unsafe operations inside the block.
- `unsafe` does not disable normal parsing or type checking.
- `unsafe` does not silence unrelated diagnostics.
- Unsafe regions should be visible in HIR and MIR.
- Nested `unsafe` blocks are allowed but redundant.

Future unsafe operations include:

- Raw pointer dereference.
- Pointer arithmetic.
- Inline assembly.
- Volatile and MMIO operations.
- Calls to unsafe functions.
- `unsafe impl` for human-asserted safety contracts.

## Unsafe functions

Unsafe functions mark call-site obligations:

```cpp
unsafe void copyBytes(byte* dst, byte const* src, usize count) {
    // implementation omitted
}
```

Rules:

- Calling an unsafe function requires an unsafe context.
- Unsafe functions still parse and typecheck normally.
- An unsafe function may perform unsafe operations internally.
- The final function effect model is future work.
- The unsafe marker may later integrate with effect summaries, but P6 v0 should not implement a full effect system.

## Address-of and dereference v0

Phase 6 should preserve Concept's C-family identity while allowing implementation to stage cautiously.

Recommended v0 direction:

```cpp
byte value = x;
byte* ptr = &value;

unsafe {
    byte read = *ptr;
}
```

Rules:

- Prefer C-family `&x` for address-of if the lexer and parser can support it cleanly when implementation begins.
- Prefer `*ptr` for dereference.
- Address-of a local creates a raw pointer only in allowed contexts.
- Dereference requires an unsafe context.
- Assignment through dereference is future work unless a P6 milestone explicitly scopes it tightly.
- Pointer arithmetic is future work.

If `&` conflicts with reference-type syntax or existing parser assumptions, implementation may stage an explicit builtin-style operation instead:

```cpp
byte* ptr = ptrOf(value);
```

The preferred final surface remains `&x` and `*ptr` for C-lineage readability.

## Initialization

PoC3 direction:

- Uninitialized variables are rejected in safe code.
- Explicit uninitialized storage uses `MaybeUninit<T>`.
- Partial initialization should be tracked in MIR.
- Only initialized fields are dropped.

Phase 6 should design this direction and implement only scaffold initially. Full partial-field tracking should not be forced into the first ownership milestone if it would over-expand scope.

Example:

```cpp
MaybeUninit<PageTable> table;

unsafe {
    initializePageTable(table.ptr());
}

PageTable ready = table.assumeInit();
```

`MaybeUninit<T>` is an explicit unsafe-adjacent tool. It should make uninitialized storage searchable and auditable rather than allowing ordinary locals to start life uninitialized.

MIR should eventually know storage states well enough to distinguish:

- Storage exists but is uninitialized.
- A value is fully initialized.
- A value is partially initialized.
- A value was moved out.
- A value is dead and should not be used or dropped.

## Explicit move

PoC3 direction:

```cpp
Buffer a = createBuffer();
Buffer b = move a;

use(a); // error: use after move
```

Rules:

- Moves are explicit.
- After a move, the source binding is uninitialized unless the type declares a valid moved-from state.
- The default rule is: moved means gone.
- `move` should lower into MIR explicitly.
- Use-after-move should be a compile-time error for local cases.

This keeps ownership transfer visible in source review and in MIR. It also avoids C++'s problem where a moved-from value often remains syntactically ordinary but semantically fragile.

## Immovable types

PoC3 direction:

```cpp
immovable struct MachineFrame {
    byte* resumePoint;
    byte storage[256];
};
```

Rules:

- Immovable values cannot move after construction.
- They are useful for machine frames, coroutine frames, self-reference, intrusive structures, DMA buffers, kernel objects, and memory-mapped objects.
- Construction must happen in place or behind stable ownership.
- Immovable type checking is likely beyond P6 v0.

Immovable types are part of the ownership direction because they make address stability an explicit type property instead of an incidental calling convention.

## Drop and RAII

PoC3 direction:

```cpp
impl Drop<File> {
    void drop(mut File& file) {
        close(file.handle);
    }
}
```

Rules:

- Concept has deterministic RAII.
- Concept does not use C++ destructor syntax.
- Destruction is represented through compiler-intrinsic `Drop<T>`.
- Locals drop in reverse successful initialization order.
- Fields drop in reverse declaration order after `Drop<T>.drop` runs.
- Moved values are not dropped unless a type declares a valid moved state.
- Drop functions must not throw.
- The compiler lowers destruction into explicit MIR drop points.

Phase 6 should document and possibly scaffold Drop before full insertion. Full automatic drop insertion requires MIR storage live/dead analysis, initialization state, move state, and error-path control flow.

## Marker concepts and unsafe impl

PoC3 direction:

```cpp
marker concept Copy<T>;
marker concept Trivial<T>;
marker concept Relocatable<T>;
marker concept Pod<T>;

unsafe marker concept ThreadSafe<T>;
unsafe impl ThreadSafe<MyLockFreeQueue>;
```

Phase 6 should treat marker concepts as future direction or design scaffold, not a requirement to implement concept solving.

Ownership-related markers include:

- `Copy<T>` for values that can be duplicated without ownership transfer.
- `Move<T>` for values that can be moved.
- `Drop<T>` for deterministic cleanup.
- `Trivial<T>` for values with trivial construction/destruction behavior.
- `Relocatable<T>` for values that can be relocated without breaking invariants.
- `Pod<T>` for plain-old-data layout and bitwise assumptions.

Human-asserted safety claims require `unsafe impl`. This provides a searchable audit trail for claims the compiler cannot prove, such as thread safety, interrupt safety, lock freedom, or foreign layout validity.

## Effects relation

PoC3's effect vocabulary includes:

- `noalloc` / `alloc`
- `nothrow`
- `nopanic`
- `nostackgrow`
- `nohost`
- `interrupt`
- `unsafe`
- `pure`

For Phase 6:

- `unsafe` may become part of the function/effect surface.
- Full effect checking is future work.
- P6 v0 should not implement the full effect system.
- Effect boundaries should eventually be visible in MIR.
- Restricted-profile violations, such as allocation where `noalloc` is required, should become explicit diagnostics when effect checking exists.

## MIR responsibilities

Phase 6 keeps the MIR-first rule: MIR owns semantics, and backends own representation.

MIR should eventually represent:

- Unsafe regions.
- Raw pointer operations.
- Moves.
- Drop points.
- Storage live/dead state.
- Partial initialization.
- Use-before-initialization checks.
- Use-after-move checks.
- Effect boundaries.
- Layout-sensitive operations.

Conceptual move/drop lowering sketch:

```text
bb0:
    file = call File.open(...)
    text = call file.readAll(...)
    drop(file)
    return move text

err:
    drop(file)
    return Err(e)
```

This shape makes destruction, error paths, and ownership transfer explicit before code generation. A backend should render pointer types, calls, and storage according to target representation, but it should not invent ownership semantics.

## Proposed Phase 6 milestone ladder

```text
P6-M0  Unsafe and ownership design doc

P6-M1  unsafe block/function surface
       - parser/AST/HIR for unsafe blocks/functions
       - unsafe context tracking
       - no raw pointer deref yet or only guarded stubs

P6-M2  raw pointer type surface
       - T* type support in HIR/TypeStore
       - pointer locals/params/returns
       - C backend pointer type rendering
       - no pointer arithmetic yet

P6-M3  address-of and dereference v0
       - &x and *ptr or chosen explicit syntax
       - dereference requires unsafe
       - simple pointer read/write if scoped tightly

P6-M4  pointer runtime fixtures and stabilization
       - C backend/run fixtures for simple pointers
       - invalid fixtures for deref outside unsafe

P6-M5  initialization and MaybeUninit design/scaffold
       - use-before-init direction
       - MaybeUninit surface/scaffold
       - partial initialization deferred if too large

P6-M6  explicit move expression
       - move expr
       - local use-after-move v0
       - no full drop insertion yet

P6-M7  Drop<T> surface/scaffold
       - parse/semantic representation of impl Drop<T>
       - no full automatic drop insertion unless ready

P6-M8  MIR storage live/dead and drop-point scaffold
       - explicit MIR storage states
       - drop point representation
       - use-before-init/use-after-move groundwork

P6-M9  Phase 6 closeout or split ownership/drop into Phase 7
```

Phase 6 may close after unsafe blocks/functions and raw pointers if ownership/drop scope grows too large. In that case, explicit move, Drop, full initialization tracking, and immovable types should move to Phase 7 with the Phase 6 direction preserved.

## Close criteria

Phase 6 should be considered successful if:

- Unsafe blocks/functions are designed and at least partially implemented.
- Raw pointer types exist.
- Raw pointer dereference requires `unsafe`.
- Simple pointer programs run through MIR and the C backend.
- Initialization, move, and Drop direction is documented.
- At least one ownership scaffold exists, or ownership is explicitly split into Phase 7.
- MIR storage/drop responsibilities are documented.
- No region lifetime system is introduced.

## P6-M0 status

P6-M0 adds only this design document. It intentionally does not implement lexer, parser, AST, HIR, MIR, backend, fixture, unsafe syntax, pointer type, move, Drop, or ownership-checking changes.
