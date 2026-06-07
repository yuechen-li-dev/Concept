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

## P6-M3 address-of and dereference v0

P6-M3 adds the first raw-pointer operations without expanding beyond read-only pointer flow. Address-of supports local and parameter places, and the operation lowers into MIR as `AddressOf(...)`. Dereference lowers into MIR as `Deref(...)`, requires an unsafe context, and reads through the pointer into a value; assignment through dereference remains future work.

The HIR checker rejects address-of operands that are not supported places, including temporaries and call results, and rejects dereference outside an unsafe block or unsafe function. Dereferencing a non-pointer is also rejected. The MIR validator and C backend understand address-of and read-only dereference so the executable path can emit ordinary C `&` and `*` for the supported v0 cases.

P6-M3 intentionally does not add pointer arithmetic, null literals, pointer comparisons, arbitrary place address-of, field/index address-of, write-through dereference, ownership, move, drop, references, borrow checking, volatile, atomics, or address-space semantics.

## P6-M4 pointer runtime/backend stabilization

P6-M4 is a stabilization and coverage milestone for the unsafe/raw-pointer slice already introduced by P6-M1 through P6-M3. It adds representative run fixtures for address-of locals and parameters, dereference inside unsafe blocks and unsafe functions, unsafe-call plus dereference interaction, and local pointer copies followed by dereference. It also keeps invalid fixtures around the unsafe boundary: unsafe calls outside unsafe context, dereference outside unsafe context, dereference of a non-pointer, and address-of call/temporary results.

The C backend coverage asserts that supported pointer locals render as pointer types, local address-of emits `&` on local storage, parameter address-of emits `&` on parameter storage, dereference emits `*` on the pointer local, and unsafe blocks leave no special C marker in this milestone. MIR debug coverage confirms both `AddressOf(...)` and `Deref(...)`, including address-of a parameter.

P6-M4 adds no new pointer semantics. Pointer arithmetic, assignment through dereference, null literals, arbitrary/field/index address-of, pointer comparisons, owning pointers, references and borrow checking, `MaybeUninit`, move/drop, unsafe impl, volatile/atomics/address spaces, and struct runtime layout remain future work.

### Unsafe/raw pointer slice status

P6-M1 through P6-M4 establish the unsafe permission boundary and read-only raw-pointer flow through HIR, MIR, validation, C emission, fixtures, and runtime checks. Phase 6 closes around that implemented slice rather than forcing ownership/drop into an architecture that does not yet have runtime structs or rich places.

## Phase 6 closeout status

Phase 6 successfully implemented Concept's first unsafe permission boundary and read-only raw pointer flow. The closed P6 behavior is deliberately narrow: programmers can spell unsafe regions and unsafe functions, raw pointer values can move through ordinary checked code, and actual pointer reads require an explicit unsafe context.

Unsafe blocks are the lexical permission boundary:

```cpp
unsafe {
    // checked normally, but permits unsafe operations
}
```

Unsafe functions mark a call-site obligation and check their bodies in unsafe context:

```cpp
unsafe int read(int* p) {
    return *p;
}
```

The implemented unsafe-call rules are:

- Calling unsafe functions requires an unsafe context.
- Unsafe function bodies are checked in unsafe context.
- `unsafe` does not disable normal parsing, name resolution, type checking, executable-subset checking, MIR validation, or backend legality checks.

The implemented raw pointer type surface includes builtin, void, and nominal pointees:

```cpp
int*
bool*
void*
EnumType*
```

Pointer values now flow through supported checked paths as:

- Function parameters.
- Function return values.
- Local declarations.
- Assignments and local copies.
- Function-call arguments.
- MIR locals/operands/rvalues where supported.
- MIR-backed C output for supported pointer reads.

Address-of v0 supports locals and parameters only:

```cpp
int* p = &x;
```

Read-only dereference is implemented as a value expression and requires unsafe context:

```cpp
unsafe {
    int value = *p;
}
```

The dereference result type is the raw pointer pointee type. Assignment through dereference is intentionally not part of Phase 6.

MIR and backend behavior for the closed slice is:

- Address-of lowers to MIR `AddressOf(...)`.
- Dereference lowers to MIR `Deref(...)`.
- MIR validation checks supported address-of and dereference type relationships.
- The C backend emits ordinary C `&...` and `*...` for supported cases.
- No dedicated unsafe marker is emitted in generated C for P6 v0.

## Phase 6 known limitations

Phase 6 v0 explicitly does not include:

- Pointer arithmetic.
- Assignment through dereference.
- A null literal.
- Arbitrary address-of.
- Field address-of.
- Index address-of.
- Pointer comparisons.
- Owning pointer `own T*`.
- References or borrow checking.
- `MaybeUninit` implementation.
- Explicit move implementation.
- `Drop<T>` implementation.
- Storage live/dead model.
- Partial initialization tracking.
- `unsafe impl`.
- Volatile operations.
- Atomics.
- Address spaces.
- Struct runtime layout.
- Struct field access.
- Templates or concept solving.
- Comptime or reflection.
- Region lifetime system.
- Lifetime parameters.
- Global proof that dangling references cannot occur.
- Global proof that data races cannot occur.
- Full effect system implementation.
- State machines.
- LLVM or native backend.
- Self-hosting.

## Ownership prerequisite map

Phase 6 should close before implementing `MaybeUninit`, explicit move, or `Drop<T>` because meaningful ownership/drop needs a richer place model than the compiler currently has. Current executable places are mostly locals and parameters, with only the first read-only pointer operations added in P6.

Real ownership/drop semantics need values and places that can describe more than whole locals:

- Field places such as `value.field`.
- Deref places such as `*ptr` when write-through and ownership-sensitive operations exist.
- Later index places such as `array[i]` or slice element places.
- Runtime struct values with defined layout and construction rules.
- Field access and field assignment.
- MIR place projections that preserve the path from an aggregate root to a subplace.

Without runtime structs and place projections, `Drop<T>`, partial initialization, and move semantics are mostly paper machinery: the compiler can describe whole-local states, but it cannot yet express the user-defined aggregate cases that make ownership and deterministic destruction valuable. Forcing ownership/drop into Phase 6 would either overfit to locals or create broad scaffolding with little real runtime behavior. Ownership, drop, move, and `MaybeUninit` should therefore be deferred until runtime structs and places exist.

## Recommended next phase

```text
Phase 7: Runtime structs and places
```

Suggested ladder:

```text
P7-M0  Runtime structs and places design doc
P7-M1  Struct runtime layout v0
P7-M2  Struct literals / constructors
P7-M3  Field access expressions
P7-M4  Field assignment and place projections
P7-M5  MIR place projections
P7-M6  Struct params/returns/calls
P7-M7  Runtime/backend fixtures and snapshots
P7-M9  Closeout
```

This gives Concept real user-defined runtime data instead of only builtin scalars, enums, and raw pointer values. It is the prerequisite for meaningful move/drop semantics because ownership must track aggregate fields, partial initialization, and destruction order. It is also the prerequisite for meaningful concepts/templates over user-defined types: generic constraints become much more valuable once programs can define and pass real runtime structs.

## Concepts/templates/comptime ordering

Recommended order after Phase 6 closeout:

```text
Phase 7: runtime structs and places
Phase 8: concepts and templates
Phase 9: comptime v0
Phase 10: ownership, move, Drop, MaybeUninit
```

Concepts and templates are central to Concept's identity and should not be deferred forever. They become much more meaningful after runtime structs/places exist because users can write constraints over real user-defined data rather than only the early scalar/enum subset.

Ownership/drop should benefit from marker concepts before it becomes a full runtime feature. Useful markers include:

- `Copy<T>`
- `Move<T>`
- `Trivial<T>`
- `Relocatable<T>`
- `Pod<T>`

Comptime should follow concepts/templates because PoC3 treats comptime as not being the primary generic system. Phase 9 comptime should start hermetic and deterministic before any capability-bearing host effects are considered. Ownership, move, `Drop<T>`, and `MaybeUninit` should follow after marker concepts and a richer place model exist.

## Non-goals

The closeout limitations above are the operative Phase 6 non-goals. In short, Phase 6 closes unsafe/raw pointer v0 only; it does not add ownership, runtime structs, concepts/templates, comptime, effects, region lifetimes, or new backend targets.

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

## Closed Phase 6 milestone ladder

```text
P6-M0  Unsafe and ownership design doc

P6-M1  unsafe block/function surface
       - parser/AST/HIR for unsafe blocks/functions
       - unsafe context tracking
       - unsafe call enforcement

P6-M2  raw pointer type surface
       - T* type support in HIR/TypeStore
       - pointer locals/params/returns
       - C backend pointer type rendering

P6-M3  address-of and dereference v0
       - &x and *ptr for supported local/parameter places
       - read-only dereference requires unsafe
       - no pointer write-through or arithmetic

P6-M4  pointer runtime fixtures and stabilization
       - C backend/run fixtures for simple pointer flow
       - invalid fixtures for unsafe boundary and unsupported address-of operands

P6-M5  closeout and ownership prerequisite map
       - close unsafe/raw pointer v0
       - defer ownership/drop/MaybeUninit until runtime structs and richer places exist
       - recommend Phase 7 runtime structs and places
```

The originally-considered ownership implementation milestones are intentionally not part of closed Phase 6. Explicit move, `Drop<T>`, full initialization tracking, storage live/dead states, and immovable types move to a later ownership phase after runtime structs, field places, MIR place projections, marker concepts, and the broader prerequisite map exist.

## Close criteria

Phase 6 is considered successful because:

- Unsafe blocks/functions are implemented.
- Unsafe call enforcement is implemented.
- Raw pointer types exist.
- Raw pointer dereference requires `unsafe`.
- Simple pointer programs run through MIR and the C backend.
- Initialization, move, and Drop direction is documented.
- Ownership is explicitly deferred with prerequisite evidence.
- MIR storage/drop responsibilities are documented as future work.
- No region lifetime system is introduced.

## P6-M0 status

P6-M0 adds only this design document. It intentionally does not implement lexer, parser, AST, HIR, MIR, backend, fixture, unsafe syntax, pointer type, move, Drop, or ownership-checking changes.

## P6-M3 status: address-of and dereference v0

P6-M3 introduces the first raw pointer value operations. `&local` and `&param` create raw pointers to existing local or parameter storage, and read-only `*ptr` dereferences a raw pointer to produce the pointee value. Dereference is an unsafe operation: it is accepted only inside an `unsafe` block or an unsafe function body.

This milestone deliberately keeps the surface narrow. Address-of does not yet apply to fields, indexes, temporaries, or arbitrary place expressions. Dereference is read-only; assignment through `*ptr` is not part of P6-M3. Pointer arithmetic, null literals, ownership, move/drop, references, borrow checking, MaybeUninit, volatile/atomics, address spaces, and struct runtime layout remain future work.
