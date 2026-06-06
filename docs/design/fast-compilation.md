# Fast Compilation as a Language Design Constraint

Concept treats compile speed as a language design goal, not merely a compiler implementation detail.

Fast compilers are not made only by clever backends, incremental build caches, or heroic engineering. They are made by languages that are easy to parse, easy to resolve, easy to typecheck, easy to lower, and easy to cache.

Concept is designed for compiler, runtime, kernel, and bare-metal work. Those domains demand fast iteration. Slow compilers make programmers cautious, make tests expensive, and make large-scale refactoring painful. A systems language should not make its users wait around because the language design invited unnecessary complexity into the compiler.

Concept should compile quickly because the language itself refuses to become a haunted maze.

---

# 1. Thesis

Concept’s compile-speed thesis is:

> Fast compilation is a semantic constraint.

This means language features should be judged partly by how they affect parsing, name resolution, typechecking, lowering, caching, and incremental rebuilds.

A feature that is expressive but forces global analysis, repeated reparsing, hidden dependency discovery, or open-ended compile-time interpretation must justify itself.

Concept wants power, but not at the price of turning every build into archaeology.

---

# 2. Goals

Concept should optimize for:

```text
fast clean builds
fast incremental builds
fast test loops
predictable module dependencies
compact compiler data structures
bounded generic checking
simple parse strategy
low ceremony diagnostics
clear caching boundaries
```

The ideal development loop is:

```text
edit
build
test
diagnose
repeat
```

with as little delay as possible.

Fast builds are especially important because Concept is expected to support human/LLM collaborative development. The faster the compiler and test loop, the more aggressively the project can use review, testing, and iteration.

---

# 3. Non-Goals

Concept does not aim to be fast by becoming simplistic.

Concept is still a low-level systems language. It needs:

```text
manual memory
explicit layout
ABI control
unsafe operations
payload enums
concept-checked generics
state machines
compile-time execution
C interop
```

But these features must be shaped so the compiler can reason about them predictably.

Concept should avoid features that are powerful primarily because they make compilation open-ended, implicit, or globally entangled.

---

# 4. No Headers

Concept has no user-facing headers.

This is one of the most important compile-speed decisions.

C and C++ compilation is slowed dramatically by textual inclusion. A source file is not merely parsed; it becomes the result of recursively pasting other files into it. The compiler repeatedly reparses the same declarations across many translation units.

Concept rejects that model.

Concept uses modules:

```cpp
module Core.Memory;

export struct Allocator;
export Result<void*, AllocError> allocate(mut Allocator& alloc, usize size, usize align);
```

Imported by name:

```cpp
import Core.Memory;
```

There is no `#include`.

There are no include guards.

There are no user-authored `.h` files.

A backend may generate C headers as private artifacts, but those headers are not part of the Concept source model.

Rule:

> Concept source uses modules.
> Headers are backend artifacts only.

---

# 5. Module Summaries

A Concept module should compile into a compact summary artifact.

Possible internal artifact:

```text
build/concept/modules/Core.Memory.cmod
```

A module summary should contain the information importers need:

```text
exported declarations
exported type signatures
exported concept declarations
visible impl metadata
effect signatures
layout metadata where required
ABI-relevant metadata
dependency hashes
```

A module summary should not normally contain:

```text
private function bodies
private locals
unneeded source trivia
full private AST
implementation-only declarations
```

When another module imports `Core.Memory`, the compiler should read the module summary instead of reparsing and rechecking the full source.

This is essential for fast incremental builds.

---

# 6. Interface Hash vs Implementation Hash

Each module should track at least two hashes:

```text
interface hash
implementation hash
```

The interface hash changes when exported declarations or ABI-visible information changes.

The implementation hash changes when private implementation details change.

Example:

```cpp
export int add(int a, int b);
```

If the implementation changes from:

```cpp
return a + b;
```

to:

```cpp
int result = a + b;
return result;
```

the interface hash should remain unchanged.

Downstream modules should not need to reparse, re-resolve, or re-typecheck merely because a private function body changed.

Rule:

> Private implementation changes should not invalidate importers unless the public interface changes.

This is one of the main ways Concept can achieve fast incremental builds.

---

# 7. Explicit Imports

Concept imports are explicit.

```cpp
import Core.Memory;
import Core.Diagnostics;
```

Concept should avoid ambient global namespace behavior.

Wildcard imports should be avoided in the core design, at least initially. If selective imports are later added, they should be explicit:

```cpp
import Core.Math.{Vec3, Matrix4};
```

The compiler should be able to build a clear dependency graph:

```text
A imports B and C
B imports D
C imports D
```

No hidden textual includes.

No dependency discovery through macro expansion.

No include-order dependency.

No import behavior that changes based on unrelated source files.

---

# 8. Simple Grammar

Concept should remain easy to parse.

The parser should not require type information to decide the basic syntactic shape of most code.

Good:

```cpp
struct Vec3 {
    float x;
    float y;
    float z;
};
```

Good:

```cpp
Result<Token, LexError> nextToken(mut Lexer& lexer);
```

Bad:

```cpp
T<U<V>>::template f<decltype(x)>(y)
```

Concept should avoid C++-style context-sensitive parsing traps where the parser must know whether a name is a type before it can understand the syntax.

Rule:

> Type positions and expression positions should be syntactically obvious.

This keeps parsing fast and diagnostics simpler.

---

# 9. Concept-Checked Generics

Concept uses concept-checked generics.

Generic functions should be checked against declared constraints, not typechecked only after substitution.

Example:

```cpp
concept Hashable<T> {
    uint64 hash(T& value);
};

template<T: Hashable<T>>
uint64 useHash(T& value) {
    return hash(value);
}
```

The compiler should be able to check `useHash` once against the `Hashable<T>` contract.

Instantiation may still be required for layout and code generation, but semantic checking should not become C++-style substitution archaeology.

Concept rejects:

```text
SFINAE
enable_if
substitution-driven overload discovery
generic bodies checked only after arbitrary instantiation
```

Rule:

> Generics should compile like contracts, not like throwing templates into a volcano and reading the smoke.

---

# 10. Controlled Overload Resolution

Concept should restrict overload resolution to keep it predictable and fast.

The language should avoid:

```text
implicit user-defined conversions
argument-dependent lookup
global overload soup
ambiguous constraint-driven overload sets
overloads that require speculative deep typechecking
```

In particular, Concept should not have C++-style ADL.

Argument-dependent lookup makes name resolution harder to reason about and harder to make fast. A call should not require the compiler to search a large implicit universe of namespaces based on argument types.

Concept should prefer explicit lookup and explicit concept-constrained dispatch.

Rule:

> Name resolution should be local, deterministic, and cacheable.

---

# 11. No Core Macro System

Concept has no textual preprocessor.

No `#define`.

No include guards.

No macro-driven conditional parsing.

PoC3 does not specify a macro system. This is not an accident.

Macro systems can be powerful, but they often destroy compile-time predictability. They can force expansion before parsing, hide dependencies, generate surprising declarations, and make caching harder.

Concept should prefer:

```text
templates
concepts
comptime functions
compile-time reflection
generated declarations
```

If Concept later gains hygienic macros, they should be tightly integrated with modules, dependency tracking, and caching. They should not become a second language running ahead of the compiler.

---

# 12. Conservative Comptime

Concept supports compile-time execution, but `comptime` is not the generic system.

Comptime should be used for:

```text
constant computation
table generation
layout checks
target metadata queries
static reflection
generated declarations
```

Comptime should not become an unbounded substitute for type checking, generic constraints, imports, and build scripts.

Default `comptime` should be deterministic and hermetic.

Host-visible side effects require declared capabilities:

```cpp
comptime(read_fs)
HardwareManifest loadHardwareManifest(StringView path) {
    Bytes data = comptimeIO::readFile(path);
    return parseManifest(data);
}
```

This makes compile-time behavior auditable and cacheable.

Rule:

> Comptime should be powerful, but not allowed to quietly become the build system, package manager, and generic engine in a trench coat.

---

# 13. Compact Compiler Data Structures

Concept’s compiler should use compact, index-based storage.

AST, HIR, and MIR should avoid per-node heap allocation spiderwebs.

Preferred shape:

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

This provides:

```text
compact memory layout
stable handles
fast traversal
easy serialization
easy incremental invalidation
simple arena reset
less pointer lifetime complexity
```

The language design supports this by encouraging index-based arenas for compiler-authoring code.

This matters because Concept is expected to eventually host its own compiler, Cathedral.

---

# 14. String Interning

The compiler should intern identifiers early.

Instead of storing and comparing repeated strings:

```text
"Core"
"Memory"
"Allocator"
```

the compiler should store symbols:

```cpp
SymbolId
```

Name comparison then becomes integer comparison.

String interning improves:

```text
name resolution
module lookup
field lookup
concept impl lookup
diagnostics
AST/HIR/MIR storage
serialization
incremental caching
```

This is a simple technique with large compile-speed benefits.

---

# 15. Type Interning

Once type checking exists, Concept should intern types.

Instead of recursively comparing type structures like:

```text
Result<Vector<Token>, ParseError>
```

the compiler can compare:

```cpp
TypeId
```

A type store may look like:

```cpp
struct TypeStore {
    Vector<Type> types;
    HashMap<TypeKey, TypeId> interned;
};
```

Type interning helps:

```text
type equality
generic instantiation
concept constraint checking
layout caching
diagnostics
incremental compilation
```

Type interning should be a Cathedral design goal, even if Stage 0 begins simpler.

---

# 16. Layered IR

Concept should use clear compiler layers:

```text
Source
  ↓
Tokens
  ↓
AST
  ↓
HIR
  ↓
MIR
  ↓
Backend
```

Each layer should remove ambiguity.

AST is source-shaped.

HIR is name-resolved and compiler-shaped.

MIR is control-flow-shaped.

The compiler should not drag unnecessary source trivia through every layer. Keep spans where needed for diagnostics, but avoid carrying entire syntax structures into semantic and backend phases.

Rule:

> Each compiler layer should make the next layer simpler.

---

# 17. Fast Parser Recovery

Concept should have useful parser recovery, but early recovery should remain simple.

Heroic parser recovery can become a compiler project of its own.

Early recovery should usually advance to a known synchronization point:

```text
semicolon
closing brace
EOF
known top-level item keyword
```

The goal is to produce good diagnostics without turning the parser into a speculative maze.

Rule:

> Parser recovery should be useful, bounded, and boring.

---

# 18. Profile Defaults Reduce Inference

Concept uses profiles such as:

```cpp
profile kernel;
profile freestanding;
```

Profiles can imply default effects.

For example, `profile kernel` may imply:

```text
noalloc
nothrow
nostackgrow
nohost
```

This improves correctness and compile speed.

The compiler checks declarations against declared defaults instead of inferring broad effect behavior across arbitrary code.

Explicit contracts are faster than inference.

Rule:

> The language should prefer declared constraints over global inference.

---

# 19. State Machines Lower Predictably

Concept has first-class stackless state machines.

They should lower deterministically:

```cpp
machine LexerMachine(mut Lexer& lexer) -> Result<Token, LexError> {
    state Start {
        char c = lexer.peek();

        if (isDigit(c)) {
            transition Number;
        }

        transition Error;
    }

    state Number {
        Token token = try lexer.readNumber();
        return token;
    }

    state Error {
        return LexError::UnexpectedCharacter(lexer.currentSpan());
    }
}
```

Conceptual lowering:

```cpp
struct LexerMachineState {
    enum State {
        Start,
        Number,
        Error
    };

    State current;
    Lexer* lexer;
};
```

State machines should not require:

```text
hidden heap allocation
runtime schedulers
implicit async machinery
global analysis
```

They lower to:

```text
state enum
state storage struct
resume function
drop function
explicit transitions
explicit yield points
```

This keeps both semantics and compile-time cost predictable.

---

# 20. C Backend First

Concept should initially use a C backend.

This is not because C is the ideal final backend. It is because C is useful for bootstrap, auditability, and portability.

Early path:

```text
Concept Source
  -> Stage 0 Zig compiler
  -> C backend
  -> zig cc / clang
  -> native binary
```

The C backend gives the project a readable output target while the frontend, type system, and MIR stabilize.

Later:

```text
MIR -> LLVM for optimized release builds
MIR -> Cranelift or custom backend for fast debug builds
```

LLVM should not be required for the earliest debug loop.

Rule:

> Use C as a bootstrap and audit backend. Do not let C define Concept semantics.

---

# 21. Incremental Compilation

Concept should be designed for incremental compilation from the beginning, even if Stage 0 does not implement it immediately.

Incremental compilation requires:

```text
stable module graph
module summaries
interface hashes
implementation hashes
interned names
interned types
compact AST/HIR/MIR IDs
clear dependency tracking
```

When a file changes, the compiler should know:

```text
Did this change affect only implementation?
Did it affect exported signatures?
Did it affect layout or ABI?
Did it affect concept impl visibility?
Did it affect downstream modules?
```

The ideal behavior:

```text
private implementation changed
  -> rebuild current module
  -> do not recheck importers

exported signature changed
  -> update module summary
  -> recheck affected importers

layout changed
  -> recompile ABI/layout-dependent modules
```

This should be a Cathedral design goal.

---

# 22. Build Output Should Expose Reuse

Eventually, Concept should report build reuse clearly.

Example future output:

```text
Parsed 184 files in 42 ms
Reused 181 module summaries
Typechecked 3 changed modules in 18 ms
Lowered MIR in 11 ms
Generated C in 24 ms
Total build: 101 ms
```

This is not just vanity.

Visible build reuse helps users understand performance, caching, and dependency structure.

It also provides excellent dunk material.

No headers were harmed in the making of this compiler.

---

# 23. What Concept Rejects for Compile Speed

Concept should be skeptical of features that cause:

```text
textual source expansion
global overload search
context-sensitive parsing
implicit dependency discovery
unbounded compile-time execution
substitution-based generic checking
hidden module dependencies
macro-generated imports
implicit conversions during overload resolution
whole-program analysis for basic correctness
```

Some powerful features may still be added, but they must be designed around fast, explicit, cacheable compilation.

---

# 24. Summary

Concept should compile quickly because the language is shaped for it.

The major compile-speed choices are:

```text
no headers
real modules
module summaries
interface vs implementation hashes
explicit imports
simple grammar
concept-checked generics
restricted overload resolution
no ADL
no textual macros
conservative comptime
index-based compiler arenas
string interning
type interning
layered IR
profile-driven effects
predictable state-machine lowering
C backend first
incremental compilation as a design goal
```

Fast compilation is not an optimization pass.

It is an architectural promise.

---

# 25. Motto

> Slow compilers make cowards of everyone.
> Concept should not.
