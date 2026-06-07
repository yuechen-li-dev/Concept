# Phase 8 concepts and templates

P8-M0 was a documentation-only milestone. It defined Concept's direction for templates, concepts, marker concepts, `impl` coherence, generic checking, and staged monomorphization without implementing compiler code. P8-M1 is now complete for parser-only template function syntax: the front end parses `template<...>` function headers, stores type parameters and optional concept constraints in the AST, rejects malformed parameter lists and duplicate type parameter names, and intentionally performs no instantiation or concept satisfaction checks yet.

Phase 8 starts after the Phase 7 runtime structs and places work. Phase 7 gave Concept real user-defined runtime values, labeled struct literals, field access, field assignment, one-level field places, address-of fields, and by-value struct parameters/returns/calls. That substrate makes generic functions useful for more than primitive scalars.

## Thesis

```text
Phase 8 makes Concept live up to its name.

Templates provide reusable generic declarations.
Concepts provide explicit semantic constraints.
Generic code should be checked against declared constraints,
not instantiated until the compiler happens to find a substitution that works.
```

## Core design principle

```text
Concept templates are not C++ templates with nicer syntax.

No SFINAE.
No overload-resolution horror.
No substitution-as-control-flow.
No diagnostics novella as normal programming model.

A generic declaration should state what it requires.
The checker should validate the generic body against those requirements.
Instantiation should be boring.
```

Concept is a C++-lineage systems language, but Phase 8 should keep the parts of C++ generic programming that are useful for compiler, runtime, kernel, bare-metal, and high-performance native work while rejecting the accidental complexity that made templates a metaprogramming maze.

## Motivation

Concept is named after concepts because explicit semantic constraints are central, not decorative. A language intended for auditable systems programming needs generic code that says which operations and properties it depends on. The compiler should check those dependencies directly instead of discovering accidental success or failure after substitution.

Runtime structs and places now give templates and concepts meaningful user-defined values to operate on. Before Phase 7, generics would mostly have exercised primitive values, enums, and pointers. After Phase 7, generic code can become useful for small vector structs, IDs, handles, spans, packet headers, compiler IR records, allocator state, and other concrete systems-programming data.

Templates are needed for reusable infrastructure such as:

- data structures and algorithms;
- compiler tables, IDs, arenas, spans, and typed handles;
- allocator-aware containers;
- option/result-like abstractions once generic enums are available;
- test relations and reusable fixture helpers;
- low-level libraries that must avoid dynamic allocation and virtual dispatch;
- kernel, runtime, embedded, and bare-metal support code where static dispatch matters.

Concepts are needed so generic code can be checked against explicit contracts. A function that compares two `T` values should require an equality concept; a hash table should require hash and equality concepts; a byte-copying routine should require marker concepts such as `Copy<T>`, `Trivial<T>`, `Relocatable<T>`, or `Pod<T>` instead of relying on undocumented assumptions.

Phase 8 should allow powerful generic programming while avoiding C++ template pathology. Generic code should remain legible, diagnostics should point at missing or invalid constraints, and ordinary programming should not require `enable_if`, SFINAE, substitution failures, overload traps, or template metaprogramming as control flow.

## Relationship to PoC3

PoC3 is the constitutional reference for Phase 8.

PoC3-aligned commitments:

- Concept is C++-lineage, not Rust with a fake mustache.
- `template`, `concept`, `interface`, and `impl` are first-class surface concepts.
- Generics are concept-checked.
- `comptime` is not the primary generic system.
- `comptime` comes later for static computation, layout checks, reflection, and generated declarations.
- Concepts are compile-time constraints with no runtime representation.
- Interfaces are runtime vtable contracts with representation and ABI concerns.
- Interfaces and dynamic dispatch are not part of Phase 8 v0.
- No SFINAE.
- No `enable_if`.
- No `anytype`-style generic duck typing as the primary model.
- No substitution-as-control-flow.

This ordering matters. Phase 8 should establish explicit generic constraints before Phase 9 `comptime` work, because PoC3 treats `comptime` as a tool for deterministic static computation rather than as the primary way to express generics.

## Milestone status

- P8-M0: complete as the Phase 8 design baseline.
- P8-M1: complete for template function syntax parsing only. The parser accepts single and multiple type parameters, optional constraints such as `T: Equatable<T>`, and attaches the following function declaration to a `TemplateDecl` AST node.
- P8-M1 deliberately does not instantiate templates, lower templates into HIR/MIR, emit backend code for templates, enforce concept satisfaction, or allow templates on structs/enums.
- P8-M2 added symbolic HIR and type-store support for generic function templates: `type_param` TypeIds, generic function storage, unresolved constraint preservation, type-name resolution to type parameters in template scope, pointer-to-type-param representation, and explicit skipping of generic templates during executable checking, MIR lowering, and backend emission.
- P8-M3 adds simple unconstrained generic function instantiation. Calls to an unconstrained generic function infer type arguments from concrete call arguments, monomorphize a concrete HIR function, cache repeated instantiations, and let the existing HIR checker, MIR lowering, MIR validator, and C backend handle only concrete functions. Generic templates remain non-emitted, constrained generic calls remain unsupported, and generic structs/enums, concepts, concept satisfaction, impl declarations, explicit specialization, and comptime remain out of scope.

## Phase 8 v0 scope

Phase 8 v0 should focus on generic functions first. Generic functions are enough to prove parsing, type-parameter representation, checking in a symbolic environment, instantiation, and backend integration without immediately redesigning every nominal type declaration.

Recommended v0:

- template type parameters;
- generic function declarations;
- explicit instantiation and call-site inference if manageable;
- monomorphization into concrete functions;
- concept declarations as static requirements;
- concept-constrained template parameters;
- `impl` declarations for concrete type satisfaction;
- marker concepts as no-method constraints.

Deferred from v0:

- generic structs;
- generic enums;
- generic methods;
- associated types;
- higher-kinded types;
- specialization;
- partial specialization;
- default type parameters;
- variadic templates;
- const generics;
- `comptime` execution;
- reflection;
- interfaces and `dyn` dispatch;
- overload sets beyond the current function model.

## Syntax: template functions

PoC3's preferred first syntax is the prefix `template<...>` form:

```cpp
template<T>
T identity(T value) {
    return value;
}
```

A concept-constrained generic function attaches constraints to template parameters:

```cpp
template<T: Equatable<T>>
bool areEqual(T left, T right) {
    return equals(left, right);
}
```

Multiple type parameters use the same list form:

```cpp
template<K, V>
V getOrDefault(Map<K, V> map, K key, V fallback) {
    ...
}
```

Constraint conjunction uses ordinary boolean-looking syntax over concept predicates:

```cpp
template<T: Hashable<T> && Equatable<T>>
bool contains(HashSet<T> set, T value) {
    uint64 h = hash(value);
    return set.containsHash(h, value);
}
```

A compact postfix function syntax may be considered later:

```cpp
bool contains<T: Hashable<T> && Equatable<T>>(HashSet<T> set, T value) {
    ...
}
```

Implementation recommendation:

- implement the prefix `template<...>` form first;
- defer compact postfix function syntax until the parser and diagnostics have proven the prefix form;
- keep generic functions compatible with the current function declaration model where possible;
- avoid introducing overload-set complexity just to support generics.

## Syntax: concept declarations

PoC3 examples use references and operator requirements:

```cpp
concept Equatable<T> {
    bool operator==(T& a, T& b);
};

concept Hashable<T> {
    uint64 hash(T& value);
};
```

For the current implementation stage, references and operator overloading may not be ready. A v0 concept can use ordinary function requirements over by-value parameters:

```cpp
concept Equatable<T> {
    bool equals(T left, T right);
};
```

Concept declaration rules:

- a concept declaration defines compile-time requirements;
- a concept has no runtime representation;
- requirements may initially be function signatures only;
- operator requirements can remain future work if operator overloading is not implemented;
- references in concept signatures can wait until reference semantics exist;
- concept declarations are not interfaces;
- concept arity is part of the declaration's identity;
- duplicate requirement signatures inside one concept should be rejected.

## Syntax: impl declarations

PoC3 direction:

```cpp
impl Drawable<Sprite> {
    void draw(Sprite sprite, Canvas canvas) {
        ...
    }
}
```

A Phase 8 v0 example using function requirements and a concrete struct:

```cpp
concept Equatable<T> {
    bool equals(T left, T right);
};

struct Vec2 {
    int x;
    int y;
};

impl Equatable<Vec2> {
    bool equals(Vec2 left, Vec2 right) {
        return left.x == right.x && left.y == right.y;
    }
}
```

Impl rules:

- `impl Concept<Type>` provides required functions for a concrete type;
- v0 impls may be restricted to top-level concrete types;
- impl functions are concept witnesses, not runtime vtable methods;
- no implicit method syntax is required in v0;
- call resolution inside constrained generic functions should use visible concept requirements and witnesses;
- an impl must match the concept arity and provide every required signature;
- duplicate impls for the same visible `Concept<Type>` pair are rejected;
- conflicting visible impls are compile errors.

## Marker concepts

PoC3 marker concepts are concepts with no methods:

```cpp
marker concept Copy<T>;
marker concept Trivial<T>;
marker concept Relocatable<T>;
marker concept Pod<T>;
```

Marker concept rules:

- marker concepts have no methods;
- marker concepts have no runtime representation;
- some marker concepts may be compiler-proven or compiler-derived;
- some marker concepts may encode human-asserted safety claims;
- unsafe marker concepts require `unsafe impl`.

Examples:

```cpp
marker concept Copy<T>;
unsafe marker concept ThreadSafe<T>;

derive Copy for Vec2;
unsafe impl ThreadSafe<MyQueue>;
```

For Phase 8:

- document marker concepts as a core part of the generic model;
- implementation may defer `derive` if it is too large;
- implementation may defer `unsafe impl` if it is too large;
- marker concepts are important future substrate for ownership/drop and layout claims.

Important future marker concepts include:

- `Copy<T>`;
- `Move<T>`;
- `Trivial<T>`;
- `Relocatable<T>`;
- `Pod<T>`.

Marker concepts should eventually make dangerous invariants searchable. If a queue claims thread safety, auditors should be able to search for `unsafe impl ThreadSafe` and inspect the proof obligation.

## Negative concepts

PoC3 allows restricted negative concept bounds only when the compiler can treat the concept as closed:

```cpp
template<T: !Copy<T>>
void consumeOnly(T value);
```

Phase 8 v0 should not implement negative concepts.

Rules for later work:

- negative bounds are future;
- open-world negative bounds are deferred;
- negative bounds are only sound for compiler-intrinsic marker concepts, closed marker concepts, or properties that cannot be added by later bridge modules;
- ordinary open-world concepts should not be negated as a generic programming trick.

The reason is coherence. If another module can later provide a bridge implementation for `Serializable<T>`, then `!Serializable<T>` is not a stable fact in an open world. Bridge modules are explicitly import-scoped, so open-world negation risks making behavior depend on what the caller imports. Phase 8 should avoid that complexity.

## Concepts vs interfaces

```text
concept:
  compile-time constraint
  no runtime representation
  static dispatch/templates

interface:
  runtime vtable contract
  representation and ABI concerns
  dynamic dispatch behind dyn
```

Phase 8 is about concepts, not interfaces. Interfaces and dynamic dispatch remain future work.

An interface does not automatically satisfy a concept. If the language later wants a static concept view over something interface-like, that view should be explicitly defined as a concept and explicitly implemented. This keeps runtime ABI concerns separate from compile-time generic constraints.

## Coherence and impl lookup

PoC3 target model:

An `impl Concept<Type>` is legal only if one of these is true:

- the impl is in the module defining the concept;
- the impl is in the module defining the type;
- the impl is in an explicit bridge module.

Bridge modules explicitly connect concepts and types owned elsewhere. Bridge impls are import-scoped, not ambient global facts. A bridge impl must be explicitly imported where the concept is exercised. Conflicting visible impls are compile errors.

Phase 8 v0 can begin narrower because the full module/import system is not yet in place:

- impl lookup may start as single-file/single-module lookup;
- an impl may be required to appear in the same current compilation unit;
- duplicate impls in the current compilation unit are rejected;
- conflicting visible impls are rejected once visibility/imports exist;
- bridge modules can wait until the module/import system expands.

This staged approach keeps the coherence target aligned with PoC3 while avoiding premature module-system debt.

## Generic checking model

This is the core semantic difference from C++ templates.

C++ templates are often checked by instantiation. A template body can contain operations that are not meaningful for all possible substitutions, and the compiler may only complain when a particular instantiation walks into the invalid operation. That model encourages SFINAE, `enable_if`, overload-resolution tricks, and diagnostics that read like novels.

Concept generic functions should be checked against declared constraints.

Model:

- A generic function body is checked in a type-parameter environment.
- Operations on `T` are permitted only if:
  - they are universally valid for all types in the current language subset;
  - a declared concept constraint provides the operation; or
  - a compiler-intrinsic marker/concept permits the operation.
- Instantiation substitutes concrete `TypeId`s and verifies concept satisfaction.
- Monomorphization then generates concrete HIR/MIR/backend code.

Examples of operations that may be universally valid in a narrow v0 include assigning, returning, passing, and storing a value of symbolic type `T`, provided the body does not require arithmetic, equality, field access, or layout-specific behavior. Equality, hashing, ordering, byte copying, relocation claims, thread-safety claims, and layout claims should come from concepts or compiler-intrinsic marker concepts.

Suggested staged checking:

- M1/M2 parse and store templates without executing them.
- M3 allows unconstrained generic identity/choose-style functions where the body uses only assign/return/local/call patterns valid for all substituted types.
- M4+ adds concepts and concept-provided operations.
- M6 validates constrained generic functions and uses concept witnesses for requirement calls if feasible.

A failed generic check should point at the unsupported operation and the missing constraint, not at a later instantiation artifact.

## Initial monomorphization strategy

Recommended v0 strategy:

- monomorphize generic functions per concrete type-argument tuple;
- produce deterministic internal instantiated names;
- send instantiated functions through HIR, MIR, MIR validation, and the C backend like normal functions;
- cache instantiations to avoid duplicate generated functions;
- reject recursive or infinitely expanding instantiations with a depth limit;
- report the instantiation chain when a limit is hit.

Example internal names:

```text
identity<int>     -> cpt_f_identity__int
choose<Vec2>      -> cpt_f_choose__struct_Vec2
```

Backend names should use existing deterministic escaping policies. The exact spelling can change during implementation, but snapshots should make it stable once chosen.

## Type parameters and TypeStore impact

Type system additions are likely needed:

```text
type_param(name, owner/template_id)
```

A substitution map connects symbolic parameters to concrete types at instantiation time:

```text
T -> int
T -> Vec2
```

Rules:

- type parameters need a `TypeId` or `TypeKind` representation;
- instantiated types are concrete;
- generic bodies are checked under symbolic type parameters;
- substitutions should produce concrete `TypeId`s before MIR/backend lowering;
- duplicate type parameter names in one template parameter list are rejected;
- references to unknown type parameters are rejected.

Recommended staging:

1. parse and store templates;
2. introduce type-parameter `TypeId`s;
3. instantiate simple generic functions;
4. add concept-constrained checking.

This staging keeps TypeStore changes purposeful rather than redesigning all type representation in one milestone.

## HIR impact

HIR additions likely needed:

- generic function declaration/template item;
- type parameter declarations;
- concept declaration item;
- concept requirement signatures;
- marker concept declaration item;
- impl declaration item;
- instantiated function mapping;
- generic call representation before instantiation, or a policy that instantiates during semantic collection.

Phase 8 should avoid implementing all HIR changes at once. A good early HIR target is to represent generic declarations and type parameters without making them executable. Execution can come after symbolic types and instantiation are represented cleanly.

## MIR impact

MIR should only see concrete instantiated functions and concrete types in v0.

Recommended rule:

```text
Generic HIR is instantiated before MIR lowering.
```

Consequences:

- MIR does not need generic type parameters initially;
- MIR validation continues checking concrete `TypeId`s;
- the C backend path remains close to the existing runtime/backend fixture model;
- unsupported generic uses fail before or during instantiation rather than producing unresolved MIR.

This keeps MIR simple and preserves the current backend path.

## C backend impact

The C backend should emit only concrete instantiated declarations.

Rules:

- deterministic names for instantiations;
- no C templates;
- no generic C output;
- no unresolved type parameters reach the backend in v0;
- unsupported instantiated types fail with existing backend unsupported-type diagnostics where possible;
- snapshots should cover instantiated function declarations and calls once implementation reaches backend support.

The backend should not become a generic solver. It should receive ordinary concrete functions after semantic checking and monomorphization.

## Diagnostics plan

Suggested diagnostics:

- `DuplicateTemplateParameter`
- `UnknownTypeParameter`
- `TemplateArityMismatch`
- `GenericFunctionNotInstantiable`
- `RecursiveTemplateInstantiationLimit`
- `UnknownConcept`
- `DuplicateConceptRequirement`
- `ConceptArityMismatch`
- `UnsatisfiedConceptConstraint`
- `MissingConceptRequirementImpl`
- `DuplicateImpl`
- `ConflictingImpl`
- `InvalidImplTarget`
- `UnsupportedGenericFeature`

Exact diagnostic codes can be assigned during implementation. The important property is that diagnostics explain the violated generic rule directly instead of burying the reason in a concrete backend failure.

## Fixture strategy

Planned fixtures:

- parse generic identity function;
- parse concept declaration;
- parse constrained template;
- valid unconstrained `identity<int>`;
- valid generic `choose<T>` over int, bool, and struct;
- invalid template arity mismatch;
- invalid use of unconstrained unsupported operation;
- valid concept declaration with requirement signature;
- valid impl for concrete struct;
- valid constrained generic call;
- invalid constrained call where concept impl is missing;
- duplicate type parameter;
- duplicate impl;
- MIR snapshots for instantiated functions;
- C backend snapshots for instantiated functions.

Fixtures should follow the existing `.conception` style. Early milestones can be parse/check fixtures only; run, MIR, and backend-C fixtures should appear once instantiated functions travel the real executable path.

## Proposed Phase 8 milestone ladder

```text
P8-M0  Concepts/templates design doc

P8-M1  Template syntax parsing
       - template parameter lists
       - generic function declarations
       - no instantiation yet

P8-M2  Type parameter TypeIds and HIR generic declarations
       - symbolic type params
       - generic function HIR storage
       - no execution yet

       Current P8-M2 implementation note:
       - `TypeStore` has symbolic type-parameter `TypeId`s owned by a generic function owner/index.
       - Generic function declarations lower to HIR generic-function storage with owned type-parameter records.
       - Function return and parameter type names resolve against template type parameters while lowering the generic declaration.
       - Parser-level constraints such as `Equatable<T>` are preserved in HIR as unresolved constraint text for future concept work.
       - Generic bodies are stored in HIR for declaration-shape cases such as identity, but generic functions are skipped by executable HIR checking, MIR lowering, and backend emission.
       - Generic calls remain unsupported until P8-M3; ordinary call lookup does not instantiate templates.

P8-M3  Simple generic function instantiation
       - unconstrained identity/choose-style functions
       - monomorphization for concrete calls
       - MIR sees concrete functions only

P8-M4  Concept declaration parsing and HIR
       - concept items
       - function requirement signatures
       - marker concept declarations

P8-M5  Impl declarations and concrete satisfaction
       - impl Concept<Type>
       - requirement matching
       - duplicate/conflict checks in current module

P8-M6  Constrained generic functions
       - template<T: Concept<T>>
       - concept satisfaction at instantiation
       - concept requirement calls in generic bodies if feasible

P8-M7  Marker concepts and unsafe impl scaffold
       - marker concepts
       - compiler-known marker placeholders
       - unsafe impl surface if small
       - no ownership/drop yet

P8-M8  Runtime/backend fixture stabilization
       - instantiated MIR snapshots
       - C backend instantiated-name snapshots
       - valid/invalid fixture corpus

P8-M9  Phase 8 closeout
```

Milestones can combine if implementation naturally converges that way, but they should not blur the target: generic functions first, concrete MIR/backend output, concepts before `comptime`, and no C++ template pathology.

## Non-goals

Phase 8 does not implement:

- `comptime` execution;
- compile-time reflection;
- generic structs or enums unless explicitly added later;
- specialization;
- partial specialization;
- default type parameters;
- variadic templates;
- const generics;
- associated types;
- higher-kinded types;
- interfaces or `dyn` dispatch;
- overload-set complexity;
- C++ SFINAE;
- `enable_if`;
- template metaprogramming as control flow;
- full module/bridge coherence;
- ownership/drop semantics;
- operator overloading unless specifically scoped.

## Close criteria

Phase 8 is successful if:

- generic functions parse;
- type parameters are represented;
- simple generic functions instantiate and run for int, bool, and simple structs;
- concept declarations parse and lower;
- impl declarations can satisfy concrete concept requirements;
- constrained generic functions can reject missing concept impls;
- marker concepts are at least documented/scaffolded;
- MIR/backend receive only concrete instantiated functions;
- docs clearly position `comptime` as Phase 9 and ownership/drop as later work.

## Roadmap note

Current roadmap direction after Phase 8 M0:

```text
Phase 7   closed or in closeout stabilization: runtime structs and places
Phase 8   planned: concepts and templates over runtime values
Phase 9   planned: comptime v0
Phase 10  planned: ownership, move, Drop, MaybeUninit, and richer storage-state analysis
```

## Implementation guardrails

P8-M0 is documentation only. It should not add lexer, parser, AST, HIR, MIR, backend, fixture, type-parameter, concept, impl, or monomorphization code.

When implementation begins, keep each milestone convergent:

- parse/store before execution;
- symbolic type representation before generic body execution;
- simple unconstrained instantiation before concept solving;
- concept declarations before impl satisfaction;
- constrained calls before broad generic language features;
- concrete MIR/backend output before any generic MIR ambitions.

## P8-M3 simple generic function instantiation

P8-M3 establishes a deliberately narrow monomorphization path for function templates whose type parameters have no constraints. The generic HIR declaration remains symbolic and is still skipped as executable code. When a concrete function body calls that template, the HIR checker now performs call-site type argument inference from the actual argument types, substitutes those concrete TypeIds through the template signature and body, and creates an ordinary concrete HIR function for the instantiated body.

Supported inference patterns are intentionally small:

- a parameter of type `T` infers `T` from the argument type;
- a parameter of type `T*` infers `T` from the pointee type of a pointer argument;
- non-generic parameter positions must match exactly after substitution;
- all uses of the same type parameter must infer the same concrete type;
- every type parameter must be inferred from arguments;
- return types are substituted after inference, with no inference from expected return type.

Instantiated function names are deterministic and include a concrete type suffix, for example `identity__int` or `identity__struct_Vec2`. The existing C backend escaping then emits stable backend symbols such as `cpt_f_identity__int`. An instantiation cache keyed by the generic function and concrete type argument tuple ensures repeated calls such as `identity(3)` and `identity(4)` reuse one concrete function, while calls with different concrete type arguments produce distinct functions.

The MIR path remains concrete-only. Generic template functions are not lowered, and the MIR validator rejects any `type_param` TypeId that reaches executable MIR. Constrained generic calls report `CON0087` until concept declarations and satisfaction are implemented in later milestones. Conflicting type inference reports `CON0088`, uninferred type parameters report `CON0089`, and unsupported generic instantiation patterns report `CON0090`.
