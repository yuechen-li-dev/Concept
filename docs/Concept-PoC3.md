# Concept: Proof of Concept 3

## Initial Design Document

### A C++-lineage systems language for auditable compiler, runtime, and bare-metal programming

---

## Status

This document defines the initial design direction for **Concept**, a speculative low-level systems programming language.

Concept is intended for writing:

```text
compilers
interpreters
JITs
runtimes
kernels
drivers
bootloaders
embedded systems
game engines
physics engines
databases
allocators
virtual machines
linkers
debuggers
profilers
GPU/runtime plumbing
high-performance native libraries
```

Concept is not a full formal specification yet.

This document is a design constitution: it defines the language identity, core constraints, major semantic commitments, and implementation direction. Future documents should define grammar, MIR format, module layout, ABI details, standard library APIs, package/build tooling, and implementation milestones.

Implementation note: Phase 12 now includes function-level `alloc`/`noalloc`
syntax, direct `noalloc` call-edge checking, compiler-known allocation surface
types, and a narrow `Arena.alloc<T>(arena)` intrinsic for non-Drop concrete
types, plus explicit `Arena.reset(arena)` and `Arena.destroy(arena)` surface
operations. Phase 12 also hardens arena allocation against
Drop/resource-management loopholes, including generic instantiations,
Drop-containing structs, and `ManualInit<DropType>`. Allocation failure paths,
transitive checking, runtime allocation support, deeper arena invalidation
analysis, and hidden heap behavior remain unimplemented.

---

# 1. Thesis

Concept exists because C++ remains one of the only mainstream languages powerful enough for compiler, runtime, kernel, and bare-metal work, but its power is buried under decades of accidental complexity.

The goal is not to make systems programming easy.

Systems programming is inherently hard.

The goal is to make systems programming honest.

Concept’s central thesis:

> Compiler, runtime, and bare-metal programming are hard.
> Concept exists to make them explicit, auditable, and sane — not haunted.

Concept is not safe systems programming at all costs.

Concept is auditable systems programming.

---

# 2. Design Laws

Concept has three design laws.

## 2.1 Essential difficulty remains visible

The language should not abstract away machine reality when machine reality is the point.

The following are inherently difficult:

```text
manual memory
ownership
aliasing
ABI
layout
alignment
calling conventions
interrupt safety
allocator design
cache behavior
code generation
atomic ordering
device memory
runtime initialization
platform targets
```

Concept should expose these realities clearly.

It should not turn bare-metal work into cinematic compiler magic.

## 2.2 Accidental difficulty is removed or quarantined

Concept should reject complexity created by historical accident, ambiguous defaults, or hidden behavior.

The following should not be normal:

```text
implicit narrowing conversions
ambient exceptions
textual macros
header-order dependency
array-to-pointer decay
global initialization order roulette
SFINAE
uninitialized variables
implicit constructor conversions
implicit virtual dispatch
ordinary-looking raw pointer danger
ambiguous ownership
discarded errors
```

Dangerous operations may exist.

They must look dangerous.

## 2.3 No accidental footguns, only deliberate ones

Concept allows low-level power.

It does not try to stop expert programmers from doing expert things.

But if a programmer reaches for a footgun, the language should make them pick it up deliberately.

Examples:

```text
discarding errors      => explicit discard
raw pointer deref      => unsafe
thread-safety claims   => unsafe impl
layout claims          => repr(...) or unsafe impl Pod<T>
allocation in kernels  => explicit alloc effect
self-referential types => immovable
foreign ABI exposure   => repr(C) and ABI annotations
```

The language should leave fingerprints around dangerous decisions.

---

# 3. Identity

Concept is:

```text
C++-lineage
C-family syntax
manual-memory capable
MIR-first
allocator-explicit
RAII-based
concept-constrained
freestanding-capable
effect-aware
layout-aware
ABI-aware
state-machine aware
```

Concept is not:

```text
Rust 2
C++ with more features stacked on top
a garbage-collected language
a borrow-checker-first language
a scripting language
a web/backend application language
a beginner language
a managed runtime language
```

Concept should feel like:

> C++ if the language was allowed to remove bad ideas instead of only accumulating workarounds.

---

# 4. Naming

The language is called **Concept**.

The name is intentional.

C++ concepts are one of the better modern additions to C++, and Concept treats explicit semantic constraints as central rather than decorative.

The language name and the `concept` keyword overlap by design.

This is acceptable.

Languages have survived worse names.

---

# 5. Visual Shape

Concept uses C-family syntax.

It keeps:

```text
braces
semicolons
struct
enum
module
import
template
concept
interface
impl
operator
explicit move
explicit try
explicit unsafe
```

Example:

```cpp
module Game.Assets;

import Core.Memory;
import Core.IO;
import Core.GPU;

enum LoadError {
    NotFound(Path path),
    DecodeFailed(String message),
    GpuUploadFailed(int code)
};

struct Texture {
    GpuHandle handle;
    int width;
    int height;
};

impl Drop<Texture> {
    void drop(mut Texture& tex) {
        gpuDestroy(tex.handle);
    }
}

Result<Texture, LoadError> loadTexture(mut Allocator& alloc, Path path) {
    Bytes bytes = try IO::readAll(alloc, path);
    Image image = try decodeImage(alloc, bytes);
    Texture texture = try uploadTexture(image);

    return move texture;
}
```

Concept should look like cleaned-up C++.

It should not look like Rust with a fake mustache.

---

# 6. Source Files and Modules

Concept uses real modules.

There is no textual header inclusion model.

```cpp
module Core.Memory;

export struct Allocator;
export struct Arena;

export void* allocate(mut Allocator& alloc, usize size, usize align)
    -> Result<void*, AllocError>;

export void deallocate(mut Allocator& alloc, void* ptr, usize size, usize align);
```

Importing:

```cpp
import Core.Memory;
import Core.Diagnostics;
```

Conditional compilation is structured.

```cpp
when target.os == "none" {
    import Core.BareMetal;
}

when target.arch == "x86_64" {
    import Core.Arch.X64;
}
```

There is no `#include`.

There is no textual preprocessor.

There are no include guards.

---

# 7. Standard Library Layers

Concept’s standard library is layered by runtime assumptions.

```text
Core
    No heap.
    No OS.
    No libc.
    No threads.
    No files.
    No hidden runtime.
    Suitable for freestanding builds.

Alloc
    Allocator interfaces.
    Arena, bump, pool, page, and stack allocators.
    Containers requiring explicit allocators.

Runtime
    Optional startup, panic handling, unwinding, TLS, and threading hooks.

OS
    Files, sockets, processes, environment, platform APIs.

Full
    Hosted convenience layer for normal application programming.
```

Bare-metal code should be able to use only `Core`.

```cpp
module Kernel.Main;

profile freestanding;

import Core;
import Core.Memory;
import Core.Arch.X64;
```

No heap or runtime is assumed unless imported and configured.

---

# 8. Profiles

A module may declare one or more profiles.

```cpp
module Kernel.Interrupts;

profile freestanding;
profile kernel;
```

Profiles affect default effects, imports, runtime assumptions, and permitted operations.

Possible profiles:

```text
hosted
freestanding
kernel
embedded
safe
systems
unsafe
audit
no_runtime
```

## 8.1 Profile-driven effect defaults

Concept avoids annotation spam by allowing profiles to define default effects.

In hosted code, the default may be permissive.

In freestanding, kernel, embedded, or no-runtime code, deterministic constraints become the baseline.

Example:

```cpp
module Kernel.Scheduler;

profile kernel;
```

A `profile kernel` module may imply:

```text
default noalloc
default nothrow
default nostackgrow
default nohost
default explicit panic behavior
```

A function that allocates must say so:

```cpp
alloc Result<TaskId, AllocError>
createTask(mut KernelHeap& heap, EntryPoint entry) {
    Task* task = try heap.allocate<Task>();
    return TaskId::from(task);
}
```

This is the quarantine model:

> In low-level code, determinism is normal.
> Allocation and runtime dependence are suspicious.

## 8.2 Explicit effects still exist

Functions may still use explicit effects.

```cpp
interrupt noalloc nothrow nostackgrow
void timerInterrupt(InterruptFrame& frame) {
    atomicFetchAdd(ticks&, 1, Relaxed);

    unsafe {
        outb(PIC_COMMAND, PIC_EOI);
    }
}
```

Effects are part of the function type and are checked transitively.

A `noalloc` function may not call an allocating function.

A `nothrow` function may not call a throwing function unless it handles the failure locally.

A `nopanic` function may not call a function that can panic.

---

# 9. Effects

Concept supports practical effect annotations.

These are not intended as academic effect-system theater.

They exist because systems programmers need hard constraints.

Possible effects:

```text
noalloc       cannot perform heap allocation
alloc         may allocate
nothrow       cannot throw or unwind
nopanic       cannot panic
nostackgrow   cannot dynamically grow stack or use split stacks
nohost        cannot depend on hosted runtime facilities
freestanding  no hosted runtime assumptions
interrupt     valid interrupt handler constraints
cold          unlikely path
hot           performance-sensitive path
unsafe        requires unsafe operations internally or caller acknowledgment
pure          no observable side effects
```

Invalid:

```cpp
interrupt noalloc nothrow
void badInterrupt() {
    Vector<int> xs = Vector<int>::new(defaultAllocator&);
    // compile error: allocation inside noalloc interrupt handler
}
```

Effects should be visible in MIR.

Effect violations should be caught before backend code generation.

---

# 10. Memory Model: Honest, Not Rust-Style Safe

Concept is not Rust-style memory safe.

It catches important local mistakes and enforces visible danger boundaries, but it does not attempt to globally prove all lifetimes.

Concept guarantees or checks:

```text
use before initialization is rejected
use after move is rejected
null references are rejected
nullable access requires checking
discarded must_use values are rejected
raw pointer deref requires unsafe
allocation effects are enforced for direct noalloc call edges
destructor/drop order is deterministic
moved values are not double-dropped
immovable values cannot be moved
```

Concept does not generally guarantee:

```text
all use-after-free bugs are caught
all dangling references are caught
all aliasing bugs are caught
all data races are impossible
all unsafe invariants are verified
all foreign ABI assumptions are correct
```

Concept’s safety story is:

> The compiler catches accidental hazards where it can.
> The programmer explicitly marks deliberate hazards.

---

# 11. Values, References, and Pointers

## 11.1 Values

Types are value types by default.

```cpp
struct Vec3 {
    float x;
    float y;
    float z;
};

Vec3 a = { 1.0, 2.0, 3.0 };
Vec3 b = a; // allowed only if Vec3 is copyable
```

Resource-owning types are movable but not copyable by default.

```cpp
struct Buffer {
    byte* data;
    usize length;
};

impl Drop<Buffer> {
    void drop(mut Buffer& buffer) {
        free(buffer.data);
    }
}
```

## 11.2 References

References are non-null borrowed views.

```cpp
T&          // non-null read borrow
mut T&      // non-null mutable borrow
```

Examples:

```cpp
void draw(Texture& texture);

void update(mut Entity& entity) {
    entity.position += entity.velocity;
}
```

A mutable reference must be explicit.

```cpp
void normalize(mut Vec3& v) {
    float len = sqrt(v.x * v.x + v.y * v.y + v.z * v.z);
    v.x /= len;
    v.y /= len;
    v.z /= len;
}
```

Concept performs local checks against obvious dangling references, but it does not require full Rust-style lifetime annotations for ordinary references.

## 11.3 Raw pointers

Raw pointers are nullable and unsafe to dereference.

```cpp
T*          // raw nullable pointer
own T*      // owning pointer
```

Dereference requires unsafe:

```cpp
unsafe {
    byte value = *ptr;
}
```

Pointer arithmetic requires unsafe:

```cpp
unsafe {
    byte* next = ptr + 1;
}
```

Raw pointers are for low-level work, FFI, allocators, drivers, runtimes, and performance-critical internals.

Safe APIs should prefer references, spans, IDs, and explicit ownership types.

## 11.4 Spans

A span is a borrowed pointer plus length.

```cpp
span<byte> bytes;
span<byte const> readonlyBytes;
```

Bounds-checked access is safe.

```cpp
Result<uint32, MemoryError> readU32(span<byte const> bytes, usize offset) {
    if (offset + 4 > bytes.length) {
        return MemoryError::OutOfBounds(offset);
    }

    return loadUnaligned<uint32>(bytes.subspan(offset, 4));
}
```

---

# 12. No Region Lifetime System in Core

PoC2 sketched region-scoped arena escape checking.

PoC3 removes it from core.

Concept should not split the difference between Zig-style explicit allocator passing and Rust-style lifetime checking.

A half-lifetime system risks becoming hidden compiler magic. If region checking stays purely local, it fails across function boundaries. If it works across function boundaries, it becomes lifetime parameters.

Concept instead prefers:

```text
explicit allocator passing
deterministic arena destruction
index-based arena handles
stable IDs instead of borrowed node pointers
optional static analysis tooling for arena escape bugs
```

Compiler-oriented code should usually use ID-based arenas.

```cpp
struct AstStore {
    Vector<Expr> exprs;
    Vector<Stmt> stmts;
    Vector<Item> items;
};

using ExprId = Id<Expr>;
using StmtId = Id<Stmt>;
using ItemId = Id<Item>;
```

This is honest:

> Nodes live as long as the store.
> Handles are stable.
> Pointers are temporary views.

---

# 13. Initialization

Uninitialized variables are rejected in safe code.

```cpp
int x;
print(x); // compile error: use of uninitialized variable
```

Explicit uninitialized storage is available for low-level code.

```cpp
MaybeUninit<PageTable> table;

unsafe {
    initializePageTable(table.ptr());
}

PageTable ready = table.assumeInit();
```

Partial initialization is tracked in MIR.

Only initialized fields are dropped.

---

# 14. Move Semantics

Moves are explicit.

```cpp
Buffer a = createBuffer();
Buffer b = move a;

use(a); // compile error: use after move
```

After a move, the original binding is considered uninitialized unless the type declares a valid moved-from state.

```cpp
struct Socket {
    Handle handle;

    moved_state Closed;
};
```

Default rule:

> Moved means gone.

This avoids C++’s “valid but unspecified” moved-from husks.

---

# 15. Immoveable Types

Some types must not move after construction.

Concept supports `immovable`.

```cpp
immovable struct MachineFrame {
    byte* resumePoint;
    byte storage[256];
};
```

Invalid:

```cpp
MachineFrame a = createFrame();
MachineFrame b = move a;
// compile error: MachineFrame is immovable
```

An immovable type must be constructed in place or behind a stable owning location.

```cpp
MachineFrame& frame = emplace<MachineFrame>(arena&, args...);
```

Use cases:

```text
machine frames
coroutine frames
self-referential objects
intrusive data structures
DMA buffers
kernel objects
memory-mapped objects
runtime internals
```

`immovable` is Concept’s explicit answer to self-reference and stable-address invariants.

---

# 16. Drop and RAII

Concept keeps deterministic RAII, but it does not use C++ destructor syntax.

Destruction is represented through compiler-intrinsic `Drop<T>`.

```cpp
struct File {
    Handle handle;
};

impl Drop<File> {
    void drop(mut File& file) {
        close(file.handle);
    }
}
```

RAII example:

```cpp
Result<IOError> writeLog(Path path, StringView message) {
    File file = try File::create(path);
    try file.write(message);
    return;
} // file is dropped here
```

Drop rules:

```text
locals drop in reverse successful initialization order
fields drop in reverse declaration order after Drop<T>.drop runs
array elements drop in reverse index order
partially initialized objects drop only initialized fields
moved values are not dropped unless they define moved_state
temporary destruction points are specified by the language
drop functions must not throw
```

Function arguments evaluate left to right.

Temporaries drop at the end of the full expression unless moved or lifetime-extended by a specified rule.

The compiler lowers destruction into explicit MIR drop points.

---

# 17. Errors

Recoverable errors are values.

```cpp
enum IOError {
    NotFound(Path path),
    PermissionDenied(Path path),
    ReadFailed(int osCode)
};

Result<File, IOError> open(Path path);
```

Propagation uses `try`.

```cpp
Result<String, IOError> readText(mut Allocator& alloc, Path path) {
    File file = try File::open(path);
    String text = try file.readAll(alloc);
    return move text;
}
```

Fallible void-returning functions use `Result<E>`.

```cpp
Result<IOError> writeText(Path path, StringView text) {
    File file = try File::create(path);
    try file.write(text);
    return;
}
```

`Result<E>` is sugar for `Result<void, E>`.

No `return {};` success wart is required.

## 17.1 Named errors

Concept prefers nominal error types at API boundaries.

```cpp
enum ParseError {
    ExpectedExpression(SourceSpan span),
    UnexpectedToken(Token token),
    UnterminatedString(SourceSpan span)
};
```

Concept does not use inferred structural error sets as the primary API model.

An API should communicate what it can return.

## 17.2 must_use

`Result` is `must_use` by default.

```cpp
open(path); // compile error: discarded Result
```

Intentional discard must be explicit.

```cpp
discard open(tempPath);
```

`must_use` is generalizable.

```cpp
must_use
Token nextToken(mut Lexer& lexer);
```

Discarding a `must_use` value without `discard` is an error.

## 17.3 Panic

Fatal programmer errors use `panic`, `assert`, or configured trap behavior.

```cpp
assert(index < array.length);
panic("unreachable parser state");
```

In freestanding profiles, panic behavior must be configured.

```cpp
panic_handler void kernelPanic(PanicInfo info) noreturn {
    serialWrite(info.message);
    halt();
}
```

Possible panic modes:

```text
panic = abort
panic = halt
panic = unwind
panic = custom(kernelPanic)
```

Ambient C++-style exceptions are not the primary error model.

---

# 18. Payload Enums

Concept supports payload enums as first-class tagged unions.

```cpp
enum Token {
    Identifier(StringView name),
    Number(double value),
    Plus,
    Minus,
    End
};
```

Pattern matching is exhaustive by default.

```cpp
void printToken(Token token) {
    match (token) {
        Identifier(name) => print("identifier: {}", name);
        Number(value)    => print("number: {}", value);
        Plus             => print("+");
        Minus            => print("-");
        End              => print("<end>");
    }
}
```

Adding a new enum variant breaks non-exhaustive matches unless they contain an explicit default.

```cpp
match (token) {
    Identifier(name) => handleName(name);
    Number(value)    => handleNumber(value);
    _                => handleOther(token);
}
```

Payload enums are essential for compiler authoring.

```cpp
enum Expr {
    IntLiteral(int64 value, SourceSpan span),
    Name(SymbolId symbol, SourceSpan span),
    Binary(BinaryOp op, ExprId left, ExprId right, SourceSpan span),
    Call(ExprId callee, span<ExprId const> args, SourceSpan span)
};
```

---

# 19. Pattern Matching

Pattern matching supports:

```text
enum variants
struct destructuring
literal matching
guards
binding by value
binding by reference
exhaustiveness checking
```

Example:

```cpp
Result<Type, TypeError> checkExpr(mut TypeContext& ctx, ExprId id) {
    Expr& expr = ctx.ast.exprs[id.index];

    return match (expr) {
        IntLiteral(value, span) =>
            BuiltinTypes::Int,

        Name(symbol, span) =>
            try ctx.lookup(symbol, span),

        Binary(op, left, right, span) =>
            try checkBinary(ctx, op, left, right, span),

        Call(callee, args, span) =>
            try checkCall(ctx, callee, args, span)
    };
}
```

With guard:

```cpp
match (token) {
    Number(value) when value < 0.0 => handleNegative(value);
    Number(value)                  => handlePositive(value);
    _                              => handleOther(token);
}
```

---

# 20. Concepts and Generics

Concept’s generic system is based on explicit `concept` constraints.

```cpp
concept Equatable<T> {
    bool operator==(T& a, T& b);
};

concept Hashable<T> {
    uint64 hash(T& value);
};
```

Generic function:

```cpp
template<T: Hashable<T> && Equatable<T>>
bool contains(HashSet<T>& set, T& value) {
    uint64 h = hash(value);
    return set.containsHash(h, value);
}
```

Compact syntax may be allowed:

```cpp
bool contains<T: Hashable<T> && Equatable<T>>(HashSet<T>& set, T& value) {
    return set.contains(value);
}
```

Design goal:

> Generics should be checked against concepts, not merely instantiated and prayed over.

No SFINAE.

No `enable_if`.

No `anytype`-style generic duck typing as the primary model.

No template diagnostics novella as normal control flow.

---

# 21. Marker Concepts

A marker concept has no methods and no runtime representation.

```cpp
marker concept Copy<T>;
marker concept Trivial<T>;
marker concept Relocatable<T>;
marker concept Pod<T>;
```

Some marker concepts are compiler-proven or compiler-derived.

```cpp
derive Copy for Vec3;
derive Trivial for Matrix4;
derive Pod for ElfHeader;
```

Some marker concepts encode human safety claims.

```cpp
unsafe marker concept ThreadSafe<T>;
unsafe marker concept InterruptSafe<T>;
unsafe marker concept LockFree<T>;
```

Human-asserted marker concepts require `unsafe impl`.

```cpp
unsafe impl ThreadSafe<MyLockFreeQueue>;
```

This creates an audit trail.

If thread safety breaks, search for:

```text
unsafe impl ThreadSafe
```

Deliberate invariants should be searchable.

## 21.1 Pod and FFI

`Pod<T>` means a type is safe to treat as plain old data for low-level copying, layout, or FFI contexts.

The compiler may auto-derive `Pod<T>` for simple `repr(C)` trivial structs.

For complex types, forcing `Pod<T>` requires an unsafe implementation.

```cpp
unsafe impl Pod<MyFfiPacket>;
```

This makes FFI layout claims visible.

---

# 22. Negative Concepts

Concept supports restricted negative concept bounds.

```cpp
template<T: !Copy<T>>
void consumeOnly(T value);
```

Negative bounds are allowed for:

```text
compiler-intrinsic marker concepts
closed marker concepts
properties that cannot be added by later bridge modules
```

Open-world negative bounds are deferred.

Risky example, not allowed in PoC3:

```cpp
template<T: !Serializable<T>>
void fallbackSerialize(T& value);
```

Reason: another module might later provide `Serializable<T>` through a bridge implementation.

PoC3 rule:

> Negative concepts are permitted only when the compiler can treat the concept as closed.

---

# 23. Implementations and Coherence

Concept separates concept definition from implementation.

```cpp
concept Drawable<T> {
    void draw(T& value, mut Canvas& canvas);
};
```

Implementation:

```cpp
impl Drawable<Sprite> {
    void draw(Sprite& sprite, mut Canvas& canvas) {
        canvas.drawTexture(sprite.texture, sprite.position);
    }
}
```

## 23.1 Orphan rule

An `impl Concept<Type>` is legal only if at least one of the following is true:

```text
the impl is in the module defining the concept
the impl is in the module defining the type
the impl is in an explicit bridge module
```

Example:

```cpp
module Graphics.Sprite;

import Graphics.Drawing;

struct Sprite {
    Texture texture;
    Vec2 position;
};

impl Drawable<Sprite> {
    void draw(Sprite& sprite, mut Canvas& canvas) {
        canvas.drawTexture(sprite.texture, sprite.position);
    }
}
```

## 23.2 Bridge modules

A bridge module explicitly connects a concept and a type owned by different modules.

```cpp
bridge module MyGame.SpriteHash
    bridges Graphics.Sprite, Core.Hashing;

impl Hashable<Sprite> {
    uint64 hash(Sprite& sprite) {
        return hash(sprite.id);
    }
}
```

Bridge impls are import-scoped.

They are not ambient global facts.

A bridge impl must be explicitly imported at the call site where the concept is exercised.

Conflicting visible impls are compile errors.

---

# 24. Interfaces and Dynamic Dispatch

Concepts and interfaces are different.

```text
concept:
    compile-time constraint
    no runtime representation
    used for templates/static dispatch

interface:
    runtime vtable contract
    has representation and ABI concerns
    used behind dyn
```

Example concept:

```cpp
concept Drawable<T> {
    void draw(T& value, mut Canvas& canvas);
};
```

Example interface:

```cpp
interface Renderer {
    void drawMesh(Mesh& mesh);
    void drawTexture(Texture& texture);
};
```

Dynamic dispatch is explicit.

```cpp
void renderScene(dyn Renderer& renderer, Scene& scene) {
    for (Mesh& mesh : scene.meshes) {
        renderer.drawMesh(mesh);
    }
}
```

An interface does not automatically satisfy a concept.

If desired, a concept view can be explicitly defined.

```cpp
concept RendererLike<T> {
    void drawMesh(mut T& renderer, Mesh& mesh);
    void drawTexture(mut T& renderer, Texture& texture);
};
```

No implicit virtual dispatch.

No accidental slicing.

No multiple inheritance in the core language.

---

# 25. Operator Overloading

Concept supports operator overloading for domain-relevant operations.

```cpp
Vec3 operator+(Vec3 a, Vec3 b) {
    return {
        a.x + b.x,
        a.y + b.y,
        a.z + b.z
    };
}
```

Concept restricts operator overloads that enable semantic fraud.

Likely disallowed or heavily restricted:

```text
overloaded comma
overloaded address-of
overloaded logical && and || without short-circuit semantics
implicit conversion operators
surprising allocation overloads
```

Operator overloading should support math and low-level abstractions, not hide control flow or allocation.

---

# 26. Allocation

Concept has explicit allocator-aware allocation.

```cpp
Arena arena = Arena::create(pageAllocator&);

AstStore ast = AstStore::create(arena&);
ExprId expr = parseExpr(arena&, ast&, source);
```

Containers do not secretly rely on a process-global heap in systems/freestanding profiles.

```cpp
Vector<Entity> entities = Vector<Entity>::withAllocator(frameArena&);
```

Hosted profiles may provide a default allocator.

Systems and freestanding profiles should not.

## 26.1 Allocator concept

Allocator-like types can satisfy an explicit concept.

```cpp
concept AllocatorLike<T> {
    void* allocate(mut T& alloc, usize size, usize align)
        -> Result<void*, AllocError>;

    void deallocate(mut T& alloc, void* ptr, usize size, usize align);
};
```

Generic allocator-aware code:

```cpp
template<A: AllocatorLike<A>>
Vector<int> makeVector(mut A& alloc) {
    return Vector<int>::withAllocator(alloc);
}
```

This avoids untyped generic duck typing while preserving explicit allocator power.

---

# 27. Index-Based Arenas

Concept’s canonical compiler-authoring pattern is index-based arenas.

No global heap.

No object graph of fragile heap pointers.

No `shared_ptr` spiderweb.

Use dense stores and stable IDs.

```cpp
struct Id<T> {
    uint32 index;
    uint32 generation;
};
```

Type aliases:

```cpp
using ExprId = Id<Expr>;
using StmtId = Id<Stmt>;
using TypeId = Id<Type>;
using SymbolId = Id<Symbol>;
```

AST store:

```cpp
struct AstStore {
    Vector<Expr> exprs;
    Vector<Stmt> stmts;
    Vector<Item> items;
    Vector<TypeRef> types;
};

Expr& get(mut AstStore& ast, ExprId id) {
    return ast.exprs[id.index];
}
```

AST nodes:

```cpp
enum Expr {
    IntLiteral(int64 value, SourceSpan span),
    Name(SymbolId symbol, SourceSpan span),
    Binary(BinaryOp op, ExprId left, ExprId right, SourceSpan span),
    Call(ExprId callee, span<ExprId const> args, SourceSpan span)
};
```

This provides:

```text
compact storage
stable handles
cache-friendly traversal
easy serialization
easy incremental invalidation
no pointer lifetime mess
simple arena reset
natural MIR/HIR table layouts
```

HIR and MIR should use the same philosophy.

```cpp
struct MirFunction {
    Vector<BasicBlock> blocks;
    Vector<Local> locals;
};

struct BasicBlock {
    Vector<Instruction> instructions;
    Terminator terminator;
};

using BasicBlockId = Id<BasicBlock>;
using LocalId = Id<Local>;
```

---

# 28. Unsafe

Unsafe operations are allowed but quarantined.

```cpp
unsafe {
    byte value = *ptr;
}
```

Unsafe functions:

```cpp
unsafe void copyBytes(byte* dst, byte const* src, usize count) {
    for (usize i = 0; i < count; i += 1) {
        dst[i] = src[i];
    }
}
```

Safe wrapper:

```cpp
Result<CopyError> copy(span<byte> dst, span<byte const> src) {
    if (dst.length < src.length) {
        return CopyError::DestinationTooSmall;
    }

    unsafe {
        copyBytes(dst.ptr, src.ptr, src.length);
    }

    return;
}
```

Unsafe means:

> The programmer is asserting invariants the compiler cannot prove.

Unsafe does not mean typechecking is disabled.

Unsafe permits otherwise-forbidden operations, but the compiler still checks all rules it can.

---

# 29. Volatile, Atomics, Barriers, and Address Spaces

Concept avoids C/C++’s confusing `volatile` type infection.

Volatile hardware access uses explicit operations.

```cpp
uint32 status = volatileLoad<uint32>(device.status);
volatileStore<uint32>(device.command, START);
```

Atomics are explicit.

```cpp
atomicStore(flag&, true, Release);
bool ready = atomicLoad(flag&, Acquire);
```

Barriers:

```cpp
compilerFence();
fence(SeqCst);
```

## 29.1 Address-space pointers

Concept may support address-space-qualified pointers for MMIO, DMA, uncached memory, and GPU-visible memory.

Sketch:

```cpp
addrspace(mmio) DeviceRegister* ctrl;
addrspace(uncached) byte* dmaBuffer;
```

Or with shorthand:

```cpp
mmio DeviceRegister* ctrl;
uncached byte* dmaBuffer;
```

MMIO operations can require MMIO address-space pointers.

```cpp
uint32 status = mmioLoad<uint32>(ctrl.status);
mmioStore<uint32>(ctrl.command, START);
```

Address-space pointer design is important for bare-metal targets but remains provisional in PoC3.

---

# 30. Layout and ABI

Concept makes layout explicit.

Default structs have compiler-defined efficient layout unless otherwise specified.

C ABI layout:

```cpp
repr(C)
struct ElfHeader {
    uint8  magic[4];
    uint8  class;
    uint8  endian;
    uint16 type;
    uint64 entry;
};
```

Packed layout:

```cpp
repr(packed)
struct IdtEntry {
    uint16 offsetLow;
    uint16 selector;
    uint8  ist;
    uint8  flags;
    uint16 offsetMid;
    uint32 offsetHigh;
    uint32 zero;
};
```

Alignment:

```cpp
align(64)
struct CacheLine<T> {
    T value;
};
```

Compile-time layout checks:

```cpp
static_assert(sizeof(IdtEntry) == 16);
static_assert(alignof(CacheLine<int>) == 64);
```

---

# 31. Bitfields and Registers

Concept supports sane bitfield/register definitions.

```cpp
bits UartLineStatus {
    dataReady:        0;
    overrunError:     1;
    parityError:      2;
    framingError:     3;
    breakInterrupt:   4;
    transmitterEmpty: 5;
    txEmpty:          6;
    fifoError:        7;
};
```

Usage:

```cpp
UartLineStatus status =
    unsafe volatileIn<UartLineStatus>(uart.basePort + 5);

if (status.transmitterEmpty) {
    unsafe {
        outb(uart.basePort, value);
    }
}
```

Bitfields must have explicit width, position, signedness, and representation.

No implementation-defined C bitfield behavior.

---

# 32. Inline Assembly

Inline assembly exists and is unsafe.

```cpp
unsafe uint64 readCr3() {
    uint64 value;

    asm {
        "mov {}, cr3"
        out(reg) value
    }

    return value;
}
```

Interrupt control:

```cpp
unsafe void disableInterrupts() {
    asm volatile {
        "cli"
    }
}
```

Assembly should support typed constraints where possible.

---

# 33. Compile-Time Execution

Concept supports compile-time execution, but conservatively.

`comptime` is not the primary generic system.

Generics are concept-checked.

Comptime is used for:

```text
table generation
layout checks
target metadata queries
static reflection
generated declarations
constant computation
```

Example:

```cpp
comptime uint32 hashString(StringView value) {
    uint32 h = 2166136261;

    for (char c : value) {
        h = (h ^ c) * 16777619;
    }

    return h;
}

static uint32 PlayerTag = comptime hashString("Player");
```

PoC3 `comptime` rules:

```text
comptime runs on the host compiler
comptime may query target metadata
comptime cannot call arbitrary extern C by default
comptime allocation uses compiler-owned temporary arenas
comptime side effects are limited and explicit
comptime execution must be deterministic by default
```

## 33.1 Capability-based comptime permissions

Comptime code that performs host-visible side effects must request capabilities.

```cpp
comptime(read_fs)
HardwareManifest loadHardwareManifest(StringView path) {
    Bytes data = comptimeIO::readFile(path);
    return parseManifest(data);
}
```

Possible capabilities:

```text
read_fs
write_fs
env
network
time
random
process
```

Default `comptime` should be hermetic and deterministic.

Build configuration must explicitly grant capabilities.

This prevents third-party compile-time code from silently behaving like an unreviewed build script.

Phase 9 roadmap note: Concept's Phase 8 templates and concepts cover generic programming v0, so `comptime` should begin as hermetic deterministic compile-time evaluation rather than as the primary generic system. Host-visible capabilities such as `read_fs`, `env`, `network`, `time`, `random`, and `process` remain future gated capabilities. No build-script demon gets unsupervised filesystem access. Little gremlin stays in the jar.

Implementation status note: Phase 8 is closed for concepts/templates v0. Phase 9 is closed for compile-time execution v0: `comptime expr`, `static_assert(expr);`, scalar `int`/`bool` compile-time functions, local mutation, `if` / `else`, `while` with deterministic fuel, declaration-only capability annotations, and deterministic target metadata (`target.pointerSize`, `target.isLittleEndian`, `target.isBigEndian`). Phase 10 is closed for ownership/storage-state v0: MIR storage-state analysis tracks initialized, moved, maybe, and partial states; explicit `move` consumes non-Copy places; Copy scalars/enums/raw pointers and `impl Copy<T>` structs may copy; moved and maybe-moved values are diagnosed; `Drop<T>` lowers to explicit MIR cleanup with reverse local order and moved-local skipping; live non-Copy/Drop replacement is rejected; partial direct-field state is tracked in MIR; and `ManualInit<T>` plus unsafe `manualAssumeInit(move slot)` exist as the manual-initialization scaffold. Phase 10 does not implement Rust-style lifetimes or full pointer-safety proofs. `ManualInit<T>` is the canonical Concept term; `MaybeUninit<T>` is only compatibility/familiarity terminology if it appears. Phase 11 closes first-class testing v0: `.con_test` files, `[Fact]`, `[Theory]`, `[InlineData]`, mandatory reasoned `Assert.*` / `Expect.*`, primitive `Expect.That` relations, HIR Fact/Theory runner, and stable test output. Deferred: named `because:`, CLI/from-disk test command, generic `TestRelation`, custom relations, richer equality, filtering, hooks, and inline test blocks.

---

# 34. Reflection

Compile-time reflection may exist, but runtime reflection is not assumed.

```cpp
comptime void generateSerializer<T>() {
    for (Field field : fieldsOf<T>()) {
        // emit serialization logic
    }
}
```

Reflection should be:

```text
static
typed
explicit
compatible with freestanding builds
not dependent on runtime RTTI unless opted in
```

Reflection is for compiler-checked generation, not runtime soup.

---

# 35. Macros

PoC3 does not specify a macro system.

There is no textual preprocessor.

No `#define`.

No include guards.

For now, Concept prefers:

```text
templates
concepts
comptime functions
compile-time reflection
generated declarations
```

A future hygienic macro system may be considered, but it is not part of PoC3.

---

# 36. C and C++ Interop

## 36.1 C interop

C interop is first-class.

Concept should be able to call C and be called from C.

```cpp
extern "C" {
    void* malloc(usize size);
    void free(void* ptr);
}
```

C-compatible layout:

```cpp
repr(C)
struct CBuffer {
    byte* data;
    usize length;
};
```

Exporting to C:

```cpp
export "C" void kernelMain(BootInfo* info) {
    Kernel kernel = Kernel::init(info);
    kernel.run();
}
```

C ABI interop is a core requirement.

It is not a future afterthought.

## 36.2 C++ interop

C++ interop is useful but quarantined.

```cpp
extern "C++" namespace std {
    type vector<T>;
}
```

Concept should not attempt to make all C++ semantics native.

C++ interop is a border crossing.

It is not the soul of the language.

---

# 37. First-Class State Machines

Concept supports first-class stackless state machines.

This is not async.

This is not C# coroutine magic.

This is not heap-allocated runtime machinery.

A Concept state machine is a compiler-lowered, explicit, noalloc-capable state structure.

State machines are core because many Concept target domains are naturally state-machine-shaped:

```text
lexers
parsers
compilers
drivers
protocol handlers
interrupt flows
cooperative schedulers
runtime fibers
incremental compilation
game AI
behavior systems
GPU command pipelines
```

A compiler is a state machine over graph-like data.

A driver is a state machine over hardware state.

A parser is a state machine over tokens.

Concept should express these honestly.

## 37.1 Syntax sketch

```cpp
machine LexerMachine(mut Lexer& lexer) -> Result<Token, LexError> {
    state Start {
        char c = lexer.peek();

        if (isDigit(c)) {
            transition Number;
        }

        if (isAlpha(c)) {
            transition Identifier;
        }

        transition Error;
    }

    state Number {
        Token token = try lexer.readNumber();
        return token;
    }

    state Identifier {
        Token token = try lexer.readIdentifier();
        return token;
    }

    state Error {
        return LexError::UnexpectedCharacter(lexer.peekSpan());
    }
}
```

A machine may yield.

```cpp
machine DevicePoller(mut Device& device) -> Result<DeviceEvent, DeviceError> {
    state Waiting {
        if (device.hasEvent()) {
            transition ReadEvent;
        }

        yield;
    }

    state ReadEvent {
        DeviceEvent event = try device.readEvent();
        return event;
    }
}
```

## 37.2 Lowering model

A machine lowers to an explicit state struct.

Conceptual lowering:

```cpp
struct LexerMachineState {
    enum State {
        Start,
        Number,
        Identifier,
        Error
    };

    State current;

    // locals that survive yield are lifted here
};
```

The generated state struct is not hidden semantically.

The programmer may not write it, but the compiler can expose it in diagnostics and MIR dumps.

## 37.3 Rules

State machine rules:

```text
no hidden heap allocation
state storage is explicit at construction site
machines can be stack-allocated, arena-allocated, or embedded
locals crossing yield are lifted into machine storage
lifted locals have explicit drop points
references crossing yield are restricted
immovable state frames are supported
noalloc/nothrow/nopanic effects apply
machine lowering is visible in MIR
```

Invalid:

```cpp
machine BadMachine() {
    state Start {
        int local = 42;
        int& ref = local;

        yield;

        print(ref);
        // compile error: reference to stack local crosses yield
    }
}
```

Valid:

```cpp
machine GoodMachine(mut Context& ctx) {
    state Start {
        ctx.value = 42;
        yield;
        print(ctx.value);
    }
}
```

## 37.4 Machine effects

Machines participate in the effect system.

```cpp
noalloc machine RealtimeController(mut Device& device) -> Result<DeviceEvent, DeviceError> {
    state Poll {
        if (device.ready()) {
            transition Read;
        }

        yield;
    }

    state Read {
        return try device.readEvent();
    }
}
```

The compiler rejects allocation in a `noalloc machine`.

## 37.5 Nested machines

Machines may run child machines.

```cpp
machine Driver(mut Device& device) -> Result<DriverEvent, DriverError> {
    state Init {
        run initializeDevice(device);
        transition Ready;
    }

    state Ready {
        run pollDevice(device);
        yield;
    }
}
```

Nested machine semantics must remain explicit:

```text
child machine storage is known
parent suspension is explicit
child drop behavior is explicit
transition away from a child drops its lifted state
```

## 37.6 Domain policies are not core

Utility scoring, AI behavior selection, and scheduler policies are not part of the core language.

They may be implemented as libraries or compiler extensions.

Core state machines provide:

```text
state
transition
yield
run
explicit lowering
effect checking
drop checking
```

Domain-specific policies sit above that.

---

# 38. Structure-of-Arrays Layout

Concept may support first-class data-oriented layout tools.

`repr(SoA)` is experimental in PoC3.

A true Structure-of-Arrays layout is not a normal contiguous array of structs. Therefore Concept should model it through a dedicated container.

```cpp
repr(SoA)
struct Particle {
    Vec3 position;
    Vec3 velocity;
    uint32 color;
};

SoA<Particle> particles = SoA<Particle>::create(alloc&, 1000);

particles[i].velocity = velocity;

span<Vec3> velocities = particles.field(.velocity);
```

This gives logical field access while allowing cache-friendly field storage.

Rules are provisional.

The key principle:

> Layout transformations must be explicit and auditable.

---

# 39. Audit Instrumentation

Concept may support audit profiles that insert debugging instrumentation at MIR level.

Examples:

```text
poison on drop
bounds check hard traps
allocator red zones
use-after-move traps
strict uninitialized memory poisoning
atomic ordering diagnostics
```

Example profile:

```text
profile audit {
    poison_on_drop = true;
    poison_pattern = 0xDE;
}
```

Conceptual MIR instrumentation:

```text
drop(value)
poison(storage)
```

Audit features are not normal runtime semantics.

They are explicit debugging/auditing modes.

---

# 40. Compiler Architecture

Concept should be implemented with a custom MIR.

Suggested pipeline:

```text
Source
  ↓
AST
  ↓
Typed HIR
  ↓
Initialization / ownership / effect analysis
  ↓
MIR
  ↓
MIR optimization
  ↓
Backends:
    C backend for bootstrap/fallback/portability
    LLVM for serious native optimization
    optional Cranelift/custom backend for fast debug builds
```

C and C++ do not define Concept semantics.

MIR owns semantics.

Backends own representation.

Interop owns ugliness.

## 40.1 MIR responsibilities

MIR should represent:

```text
control flow
moves
borrows
mutable borrows
drop points
partial initialization
enum discriminants
pattern matching lowering
bounds checks
error propagation
allocation sites
unsafe regions
effect boundaries
layout-sensitive operations
volatile operations
atomic operations
machine states
yield points
lifted locals
```

Example source:

```cpp
Result<String, IOError> readText(mut Allocator& alloc, Path path) {
    File file = try File::open(path);
    String text = try file.readAll(alloc);
    return move text;
}
```

Conceptual MIR:

```text
bb0:
    r0 = call File.open(path)
    switch r0:
        Ok(file) -> bb1
        Err(e)   -> bb_err0

bb1:
    r1 = call File.readAll(&mut file, alloc)
    switch r1:
        Ok(text) -> bb2
        Err(e)   -> bb_err1

bb2:
    drop(file)
    return Ok(move text)

bb_err1:
    drop(file)
    return Err(e)

bb_err0:
    return Err(e)
```

This makes destruction, error paths, and moves explicit before codegen.

## 40.2 Machine MIR

A machine lowers into MIR as:

```text
machine state enum
state storage struct
resume function
drop function
lifted locals
explicit transitions
explicit yield terminators
```

Conceptual MIR:

```text
resume(machine):
    switch machine.current:
        Start      -> bb_start
        Number     -> bb_number
        Identifier -> bb_identifier
        Error      -> bb_error

bb_start:
    c = call lexer.peek()
    if isDigit(c) -> set_state(Number); goto resume
    if isAlpha(c) -> set_state(Identifier); goto resume
    set_state(Error)
    goto resume

bb_number:
    r = call lexer.readNumber()
    switch r:
        Ok(token) -> return Ok(token)
        Err(e)    -> return Err(e)
```

This is explicit state-machine lowering, not hidden coroutine machinery.

---

# 41. Bringup Roadmap

Concept should not begin with LLVM.

The initial implementation should prefer a pragmatic bootstrap path.

## 41.1 Stage 0: Zig frontend with C backend

Use Zig to implement the Stage 0 compiler.

Reasons:

```text
excellent cross-compilation tooling
strong C interop
explicit allocator culture
no hidden runtime by default
good fit for compiler bringup
```

Stage 0 goals:

```text
lexer
parser
AST
basic type checker
simple HIR
C backend
basic test runner
```

Initial compilation path:

```text
Concept Source
  -> Zig Stage 0 Compiler
  -> C Backend
  -> zig cc / clang
  -> Native Binary
```

The C backend should be readable.

Generated C is an audit tool.

## 41.2 Stage 1: MIR

Once basic control flow, values, payload enums, functions, and allocator calls compile, introduce custom MIR.

Stage 1 goals:

```text
typed HIR to MIR
basic blocks
explicit drop insertion
move checking
use-before-init checking
must_use checking
effect checking
pattern match lowering
C backend generated from MIR
```

The output is still C.

But it is C generated from audited MIR, not directly from AST.

## 41.3 Stage 2: Self-hosting

With stable MIR and C backend, rewrite the compiler frontend and MIR lowerer in Concept.

Compile the Concept compiler using the Stage 1 compiler.

Once the Concept compiler can compile itself, Zig becomes optional.

## 41.4 Stage 3: Native backends

After self-hosting:

```text
LLVM backend
Cranelift or custom debug backend
target-specific lowering
better debug info
optimization passes
```

LLVM is for serious release performance.

C remains useful for bootstrap, fallback, and auditability.

---

# 42. Example: Compiler Frontend

```cpp
module Concept.Compiler.Parser;

import Core.Memory;
import Core.Diagnostics;

enum Token {
    Identifier(SymbolId symbol, SourceSpan span),
    Number(int64 value, SourceSpan span),
    KeywordLet(SourceSpan span),
    KeywordStruct(SourceSpan span),
    LeftParen(SourceSpan span),
    RightParen(SourceSpan span),
    End(SourceSpan span)
};

enum Expr {
    IntLiteral(int64 value, SourceSpan span),
    Name(SymbolId symbol, SourceSpan span),
    Binary(BinaryOp op, ExprId left, ExprId right, SourceSpan span),
    Call(ExprId callee, span<ExprId const> args, SourceSpan span)
};

struct Parser {
    Arena& arena;
    AstStore& ast;
    span<Token const> tokens;
    usize position;
    DiagnosticSink& diagnostics;
};

Result<ExprId, ParseError> parseExpr(mut Parser& parser) {
    Token& token = parser.peek();

    return match (token) {
        Number(value, span) => {
            parser.advance();
            ExprId id = parser.ast.exprs.push(Expr::IntLiteral(value, span));
            return id;
        }

        Identifier(symbol, span) => {
            parser.advance();
            ExprId id = parser.ast.exprs.push(Expr::Name(symbol, span));
            return id;
        }

        _ => {
            parser.diagnostics.error(token.span(), "expected expression");
            return ParseError::ExpectedExpression(token.span());
        }
    };
}
```

---

# 43. Example: MIR Function

```cpp
module Concept.Compiler.MIR;

enum Instruction {
    Assign(LocalId dst, Value value),
    Call(LocalId dst, FunctionId function, span<Operand const> args),
    Drop(LocalId local),
    Move(LocalId dst, LocalId src),
    Store(Place place, Operand value),
    Load(LocalId dst, Place place)
};

enum Terminator {
    Return(Operand value),
    Goto(BasicBlockId target),
    Branch(Operand condition, BasicBlockId thenBlock, BasicBlockId elseBlock),
    SwitchEnum(Operand value, span<SwitchCase const> cases),
    Panic(PanicInfo info),
    Unreachable
};

struct BasicBlock {
    Vector<Instruction> instructions;
    Terminator terminator;
};

struct MirFunction {
    Vector<Local> locals;
    Vector<BasicBlock> blocks;
};
```

---

# 44. Example: Kernel Interrupt Handler

```cpp
module Kernel.Interrupts;

profile freestanding;
profile kernel;

import Core.Arch.X64;
import Core.Atomic;

static Atomic<uint64> ticks;

interrupt
void timerInterrupt(InterruptFrame& frame) {
    atomicFetchAdd(ticks&, 1, Relaxed);

    unsafe {
        outb(PIC_COMMAND, PIC_EOI);
    }
}
```

In `profile kernel`, `interrupt` implies strict constraints unless overridden:

```text
noalloc
nothrow
nostackgrow
nohost
```

Rejected:

```cpp
interrupt
void badInterrupt() {
    Vector<int> xs = Vector<int>::new(defaultAllocator&);
    // error: allocation inside kernel interrupt handler
}
```

---

# 45. Example: Device Register

```cpp
module Kernel.Device.Uart;

import Core.Volatile;

bits UartLineStatus {
    dataReady:        0;
    overrunError:     1;
    parityError:      2;
    framingError:     3;
    breakInterrupt:   4;
    transmitterEmpty: 5;
    txEmpty:          6;
    fifoError:        7;
};

struct Uart {
    usize basePort;
};

bool canWrite(Uart& uart) {
    UartLineStatus status =
        unsafe volatileIn<UartLineStatus>(uart.basePort + 5);

    return status.transmitterEmpty;
}

void writeByte(Uart& uart, byte value) {
    while (!canWrite(uart)) {
        spinHint();
    }

    unsafe {
        outb(uart.basePort, value);
    }
}
```

---

# 46. Example: State Machine Lexer

```cpp
module Concept.Compiler.Lexer;

machine LexerMachine(mut Lexer& lexer) -> Result<Token, LexError> {
    state Start {
        char c = lexer.peek();

        if (isWhitespace(c)) {
            lexer.advance();
            transition Start;
        }

        if (isDigit(c)) {
            transition Number;
        }

        if (isAlpha(c)) {
            transition Identifier;
        }

        if (c == '\0') {
            transition End;
        }

        transition Error;
    }

    state Number {
        Token token = try lexer.readNumber();
        return token;
    }

    state Identifier {
        Token token = try lexer.readIdentifier();
        return token;
    }

    state End {
        return Token::End(lexer.currentSpan());
    }

    state Error {
        return LexError::UnexpectedCharacter(lexer.currentSpan());
    }
}
```

Conceptual lowered shape:

```cpp
immovable struct LexerMachineState {
    enum State {
        Start,
        Number,
        Identifier,
        End,
        Error
    };

    State current;
    Lexer* lexer;
};
```

No hidden heap.

No implicit runtime scheduler.

No async machinery.

Just an explicit compiler-lowered state machine.

---

# 47. Example: Marker Concept Audit Trail

```cpp
unsafe marker concept ThreadSafe<T>;

struct LockFreeQueue<T> {
    Atomic<Node*> head;
    Atomic<Node*> tail;
};

unsafe impl ThreadSafe<LockFreeQueue<T>>
    where T: ThreadSafe<T>;
```

A code reviewer can search:

```text
unsafe impl ThreadSafe
```

Every manual thread-safety claim is visible.

---

# 48. Removed or Rejected from C++

Concept deliberately avoids:

```text
textual preprocessor
header/source split
implicit narrowing conversions
implicit bool conversions from integers/pointers
null references
uninitialized variables in safe code
use-after-move
array-to-pointer decay
ambient exceptions as primary error model
SFINAE
std::enable_if-style generic constraints
multiple inheritance
implicit virtual dispatch
accidental slicing
global initialization order roulette
implicit constructor conversions
operator overload abuse
include-order-dependent semantics
implementation-defined C bitfields
ordinary-looking raw pointer dereference
discarded error values
unspecified destructor/drop order
hidden allocation in constrained contexts
C++ destructor syntax
half-magic region lifetimes
```

Dangerous operations may still exist, but behind explicit syntax, effects, profiles, or unsafe blocks.

---

# 49. Added or Elevated

Concept adds or elevates:

```text
real modules
first-class concepts
concept-checked generics
marker concepts
unsafe impl
restricted negative concepts
payload enums
exhaustive pattern matching
explicit moves
immovable types
Drop<T>
deterministic destruction
explicit allocator-aware containers
profile-driven effects
freestanding standard library profile
safe/unsafe boundary
must_use and discard
explicit volatile operations
explicit atomics and fences
layout and ABI annotations
sane bitfield/register definitions
conservative compile-time execution
capability-based comptime permissions
compile-time reflection direction
first-class C ABI interop
index-based compiler arenas
first-class stackless state machines
MIR-first compiler architecture
structured panic configuration
audit instrumentation
```

---

# 50. Future Design Documents

PoC3 is not the final spec.

The next documents should be narrower.

Suggested documents:

```text
Concept Syntax Sketch
Concept Lexer and Parser Grammar
Concept Type System Notes
Concept Concepts and Coherence
Concept Effects and Profiles
Concept MIR Design
Concept Machine/State Lowering
Concept Memory and Drop Semantics
Concept C Backend Plan
Concept Compiler Bringup Roadmap
Concept Core/Alloc Standard Library
Concept Build Manifest
Concept ABI and Layout Rules
```

PoC3 defines the constitution.

The future documents define the ministries.

---

# 51. One-Sentence Pitch

Concept is a C++-lineage language for compilers, runtimes, kernels, and bare-metal systems: explicit enough for dragons, disciplined enough for humans, and designed so accidental footguns disappear while deliberate ones leave fingerprints.

---

# 52. Motto

> We know compiler and bare-metal programming is hard.
> Concept exists to make it honest, not haunted.
