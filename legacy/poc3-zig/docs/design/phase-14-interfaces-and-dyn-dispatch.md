# Phase 14: Interfaces and dyn dispatch

P14-M0 is a documentation-only milestone. It defines Concept's runtime
interface model, borrowed `dyn` references, concrete-to-dyn coercion direction,
HIR/MIR representation, and C backend lowering plan.

P14-M0 does not implement compiler behavior. It does not change lexer, parser,
semantic, HIR, MIR, backend, fixture, or runtime behavior. Existing parser
support for `interface` signature blocks remains only the starting point for
later milestones.

## Phase 14 closeout

Phase 14 is closed. The closed v0 surface is runtime dispatch through borrowed
dyn references only: interface declarations, validated interface requirements,
validated `impl Interface<Type>` conformance, explicit `dyn Interface&` and
`mut dyn Interface&` parameter types, call-boundary concrete-to-dyn coercion,
dyn method calls through `mut dyn Interface&`, MIR-visible dyn coercions and
interface calls, and inspectable C backend lowering to fat references,
vtables, wrapper thunks, static vtable constants, and indirect calls.

The final supported v0 surface is:

- interface declarations;
- interface requirement validation;
- interface impl conformance;
- `impl Interface<Type>`;
- separate concept impls and interface impls;
- borrowed `dyn Interface&`;
- borrowed `mut dyn Interface&`;
- concrete-to-dyn call-boundary coercion;
- an addressable-place requirement for concrete-to-dyn coercion;
- matching `impl Interface<ConcreteType>` lookup for concrete-to-dyn coercion;
- dyn method calls on `mut dyn Interface&`;
- `HirExprKind.dyn_coerce`;
- `HirExprKind.interface_call`;
- `MirRvalue.dyn_coerce`;
- `MirRvalue.interface_call`;
- C backend fat-reference structs passed by value;
- C backend vtable structs;
- C backend wrapper thunks;
- C backend static vtable constants;
- indirect vtable dispatch;
- no hidden heap allocation;
- no RTTI;
- no dynamic cast;
- no inheritance or class hierarchy.

The exact v0 boundaries are deliberate:

- `dyn Interface&` and `mut dyn Interface&` are borrowed references, not owned
  objects.
- `mut dyn Interface&` is required for all dyn method calls in v0 because
  interface requirement receiver mutability is not modeled yet.
- Concrete-to-dyn coercion happens only at function call boundaries.
- A concrete coercion source must be an addressable place, such as a local,
  parameter, or already-addressable field place.
- The concrete source type must have a matching interface impl for the target
  interface.
- Dyn fat references are passed by value in generated C as `{ data, vtable }`
  structs.
- Vtables are emitted as static constants for selected interface impls.
- Wrapper thunks receive `void* self`, cast it to the concrete pointer type,
  and call the hidden interface-impl method ABI.
- Owning dyn boxes are not implemented.

Final deferred work:

- owning dyn boxes;
- heap boxing;
- `Box<dyn Interface>`;
- RTTI and dynamic cast;
- reflection;
- interface inheritance and upcasting;
- default methods;
- associated types;
- generic interface methods;
- Drop through dyn;
- destructor slots in vtables;
- dyn fields;
- dyn returns;
- dyn locals with initializers;
- dyn locals as explicit variables beyond declaration-only unsupported forms;
- cross-module ABI stability;
- import and multi-module interface coherence;
- effect checking through dyn beyond the existing direct checks;
- unsafe interface methods;
- mutation-through-dyn semantic hardening until receiver references are
  first-class TypeStore and ABI values.

P14 v0 supports runtime dispatch through borrowed dyn refs. It does not fully
harden mutation-through-dyn semantics beyond the conservative `mut dyn`
call gate and current backend thunk ABI. The backend currently casts
`void* self` to the concrete pointer type and calls the hidden impl ABI with
`*typed`. That is acceptable for v0's read-style and side-effect-free dispatch
coverage, but must be revisited when references become first-class TypeStore
values.

## Core doctrine

```text
A concept is a compile-time constraint.

An interface is a runtime dispatch contract.

A dyn reference is an explicit fat reference: data pointer plus vtable pointer.

Interfaces are not inheritance.

dyn is not hidden heap allocation.

Interface dispatch must lower to inspectable HIR/MIR/C constructs.

Phase 14 v0 supports borrowed dynamic references only.

Owning dyn boxes are deferred.

No class hierarchy enters Concept core.
```

Doctrine contrast:

```text
concept = static proof / generic constraint
interface = runtime contract / dynamic dispatch
dyn = explicit dynamic reference
```

## 1. Motivation

Concept now has static polymorphism through concepts and templates. Phase 12
closed explicit storage through allocation effects, arenas, and ID-store
doctrine. Phase 13 closed explicit control through machines and
decision-driven transitions. The next missing spine is runtime polymorphism.

Runtime dispatch is needed for:

- compiler pass APIs;
- diagnostic sinks;
- emitters;
- allocators and runtime service boundaries;
- platform backends;
- DragonGod-like sensors, actuators, and policies later;
- plugin-like architecture later.

Without `interface` and `dyn`, Concept must either hardcode everything,
template everything, or manually pass function tables through ordinary
parameters. Manual function tables are sometimes the right low-level tool, but
they should not be the only way to express a runtime contract.

Concept should support runtime dispatch without importing C++ inheritance,
Java object identity, garbage-collection assumptions, hidden heap boxing, or
virtual inheritance. The user should be able to see the dispatch boundary in
source, HIR, MIR, and generated C.

## 2. Concept vs interface

A `concept` is a compile-time constraint:

- checked at generic instantiation or call site;
- monomorphized/static;
- no runtime vtable;
- no runtime object identity;
- no `dyn` dispatch.

An `interface` is a runtime contract:

- defines callable slots;
- implemented by concrete types;
- can be used through `dyn Interface&` or the equivalent borrowed dynamic
  reference;
- lowers to vtable-shaped runtime data;
- does not imply inheritance.

```text
An interface implementation does not automatically satisfy a concept constraint.

A concept implementation does not automatically provide a runtime interface
vtable.

Future bridge mechanisms may exist, but Phase 14 v0 keeps them separate.
```

This separation is important. A concept witness is a static proof used by
generic instantiation. An interface impl is runtime data and code used through a
vtable slot. They may share source signatures, but they are not the same
semantic object.

## 3. Non-inheritance doctrine

```text
Interfaces are not base classes.

Implementing an interface does not put fields into a type.

Interfaces do not create a class hierarchy.

Interfaces do not control construction or destruction.

Interfaces do not imply heap allocation.

Interfaces do not provide virtual inheritance.

Interfaces do not create an "everything is an object" model.
```

C++ abstract base classes mix runtime dispatch, inheritance, construction and
destruction rules, layout, pointer adjustment, object identity, and sometimes
virtual inheritance. Concept should separate those concerns. Phase 14 only
designs runtime dispatch contracts and explicit borrowed dynamic references.

## 4. Source syntax: interface declarations

Initial interface syntax follows the shape the parser already accepts:

```cpp
interface Writer {
    void Write(int value);
}
```

Rules:

- interfaces are top-level declarations;
- interface bodies contain required method signatures only;
- no fields;
- no default method bodies in v0;
- no associated types in v0;
- no generic interface methods in v0 unless the current parser and HIR model
  can support them cleanly;
- return and parameter types use ordinary type syntax;
- overloads should be rejected in v0 unless the existing signature model
  handles them clearly;
- duplicate method names are rejected.

Possible future syntax:

```cpp
interface Writer<T> {
    void Write(T value);
}
```

Phase 14 v0 should avoid generic interfaces unless they turn out to be trivial
after non-generic interfaces are implemented.

## 5. Source syntax: interface implementations

Preferred long-term readable shape:

```cpp
impl Writer for ConsoleWriter {
    void Write(mut ConsoleWriter& self, int value) {
        ...
    }
}
```

The existing lower-friction parser shape is already:

```cpp
impl Writer<ConsoleWriter> {
    void Write(mut ConsoleWriter& self, int value) {
        ...
    }
}
```

Phase 14 v0 should use `impl Interface<Type>` for implementation planning
because it is closest to the current parser and AST shape. If the long-term
`impl Interface for Type` spelling is later added, it should be source sugar or
a deliberate migration, not a semantic change.

When `impl Interface<Type>` is used for interfaces, its semantics differ from
`impl Concept<Type>`:

- a concept impl provides static witnesses for generic constraint
  satisfaction;
- an interface impl provides runtime vtable slots for borrowed `dyn` dispatch.

Rules:

- an interface impl targets one concrete type;
- all required interface methods must be implemented;
- extra methods are rejected in v0;
- method names must match interface requirements;
- return types must match;
- all parameters after the receiver must match the interface requirement;
- impl method first parameter is the concrete receiver;
- receiver mutability must be explicit;
- no automatic inheritance;
- no default method bodies.

Receiver convention:

- interface requirements omit the receiver;
- interface implementation methods include the receiver as the first parameter;
- dynamic calls supply that receiver automatically.

Example:

```cpp
interface Writer {
    void Write(int value);
}

struct ConsoleWriter {
    int handle;
}

impl Writer<ConsoleWriter> {
    void Write(mut ConsoleWriter& self, int value) {
        // implementation body
    }
}
```

## 6. dyn type model

Phase 14 v0 designs borrowed dynamic references only.

Preferred source direction:

```cpp
dyn Writer&
mut dyn Writer&
```

If the current reference syntax is not strong enough for the first
implementation, an early milestone may stage through pointer-shaped dyn
references internally. The design target remains borrowed dynamic references.

Rules:

- `dyn Writer&` is a non-owning dynamic reference;
- it contains a data pointer/reference and a vtable pointer;
- it does not own the underlying object;
- it does not allocate;
- it does not extend the lifetime of the underlying object;
- it does not run Drop for the underlying object;
- it is valid only while the underlying object remains valid;
- `mut dyn Writer&` permits methods requiring mutable receiver;
- `dyn Writer&` permits only read/immutable receiver methods;
- owning dyn boxes are deferred.

Explicitly deferred:

```cpp
dyn Writer value;       // owning dyn object: deferred
own dyn Writer* value;  // owning dyn pointer: deferred
Box<dyn Writer> value;  // deferred; no Box yet
```

## 7. Runtime representation

Conceptual C shape:

```c
typedef struct {
    void* data;
    const cpt_itf_Writer_vtable* vtable;
} cpt_dyn_Writer;

typedef struct {
    void (*Write)(void* self, int value);
} cpt_itf_Writer_vtable;
```

For a concrete implementation:

```c
static void cpt_impl_ConsoleWriter_Write(void* self, int value) {
    cpt_s_ConsoleWriter* typed = (cpt_s_ConsoleWriter*)self;
    ...
}

static const cpt_itf_Writer_vtable cpt_impl_ConsoleWriter_as_Writer = {
    .Write = cpt_impl_ConsoleWriter_Write,
};
```

Call lowering:

```c
writer.vtable->Write(writer.data, 42);
```

Rules:

- vtables are compiler-emitted static data;
- dyn values are fat references;
- no heap allocation is needed;
- no runtime type reflection is required;
- no dynamic cast in v0;
- no RTTI in v0.

## 8. Concrete-to-dyn coercion

Preferred v0 coercion rules:

- a concrete value/place may coerce to `dyn Interface&` only at an explicit dyn
  parameter or variable boundary;
- the concrete type must have an interface impl;
- the source object must be an addressable place;
- rvalue temporaries should not be coerced to borrowed dyn references unless
  lifetime extension is explicitly designed;
- no heap boxing;
- no implicit owning conversion.

Example:

```cpp
void Emit(mut dyn Writer& writer) {
    writer.Write(42);
}

int main() {
    ConsoleWriter writer = ConsoleWriter { handle: 1 };
    Emit(writer);
    return 0;
}
```

This call may build a temporary fat reference at the call boundary.

Rules:

- `Emit(writer)` is allowed if `ConsoleWriter` implements `Writer`;
- `Emit(ConsoleWriter { handle: 1 })` should be rejected in v0 because the
  temporary borrowed dyn lifetime is not specified;
- `Emit(other)` is rejected if `other`'s type does not implement `Writer`;
- coercion should be explicit in HIR/MIR, not hidden in backend codegen.

If implementation needs an explicit cast or builtin first, a possible staged
syntax is:

```cpp
dyn Writer& writer_ref = dynRef<Writer>(writer);
```

This document does not require adding that source syntax. It is a contingency
for implementation staging only.

## 9. Interface method calls

Preferred dynamic method call syntax:

```cpp
writer.Write(42);
```

Rules:

- receiver type must be `dyn Writer&` or `mut dyn Writer&`;
- method name must be a requirement of the interface;
- argument arity and types must match the interface requirement;
- return type is the requirement return type;
- mutable receiver methods require mutable dyn reference;
- immutable dyn reference cannot call mutating methods;
- method lookup through dyn is interface-slot lookup, not struct field lookup;
- no overloading in v0 unless the current compiler model supports it clearly.

HIR shape conceptually:

```text
interface_call {
    receiver: ExprId,
    interface_id: InterfaceId,
    requirement_index: u32,
    args: []ExprId,
    result_type: TypeId
}
```

MIR shape conceptually:

```text
dyn_call receiver slot args
```

C lowering:

```c
receiver.vtable->Method(receiver.data, args...);
```

## 10. HIR model

Proposed HIR additions:

```text
InterfaceId
InterfaceImplId
InterfaceRequirementId

HirInterface {
    item
    name
    requirements
    span
}

HirInterfaceRequirement {
    parent
    name
    return_type
    params
    span
}

HirInterfaceImpl {
    interface_id
    target_type
    functions
    span
}

HirDynCoercion {
    source_expr
    interface_id
    impl_id
    result_type
}

HirInterfaceCall {
    receiver
    interface_id
    requirement_index
    args
    result_type
}
```

Type-store direction:

```text
interface_type: InterfaceId
dyn_interface: InterfaceId / mutability / reference-ness
```

Exact representation may vary, but Phase 14 requires:

- stable IDs;
- explicit dyn coercion in HIR;
- explicit interface call in HIR;
- no backend-only magic.

## 11. MIR model

Proposed MIR additions:

```text
MirRvalue.dyn_coerce {
    source_place/source_operand
    interface_id
    impl_id
}

MirRvalue.interface_call {
    receiver
    interface_id
    slot_index
    args
}

MirDynValue {
    data pointer
    vtable pointer
}
```

Equivalent names are fine if they fit the current MIR style.

Requirements:

- MIR exposes dyn coercion and interface calls;
- MIR validator can check that dyn calls target valid interface slots;
- MIR should not treat dyn dispatch as an ordinary direct function call;
- vtable identity must be explicit enough for backend emission.

## 12. Backend C model

Backend responsibilities:

- emit interface vtable structs;
- emit per-impl vtable constants;
- emit wrapper/thunk functions when receiver type needs casting from `void*`;
- emit dyn fat references as structs;
- emit dyn calls through vtable slots;
- no `malloc`;
- no scheduler;
- no reflection/RTTI.

Potential generated C naming:

```text
cpt_itf_Writer_vtable
cpt_dyn_Writer
cpt_impl_ConsoleWriter_Write
cpt_impl_ConsoleWriter_as_Writer
```

Exact names should follow existing backend naming conventions. Once backend
fixtures exist, snapshots should pin the generated C shape.

## 13. Ownership and Drop interaction

For v0 borrowed dyn refs:

- dyn reference is non-owning;
- underlying concrete object remains owned by its original storage;
- dyn reference does not Drop underlying object;
- concrete object drops normally according to existing Phase 10 rules;
- dyn ref itself has no Drop;
- moving a dyn reference copies or moves only the fat reference, not the
  object;
- owning dyn boxes are deferred.

Explicitly deferred:

- destructor slots in vtable;
- Drop through dyn;
- owning trait objects;
- dynamic allocation;
- object lifetime extension;
- borrowed dyn escaping analysis beyond existing reference/place rules.

## 14. Allocation and effects interaction

Rules:

- dyn coercion does not allocate;
- dyn method call has the effect of the concrete implementation function, but
  v0 may conservatively restrict or defer effect propagation;
- no hidden heap;
- no default allocator;
- no allocation to create dyn refs.

Future effect behavior:

- interface requirements should eventually carry effect annotations;
- if calling through `dyn Interface&` from a `noalloc` function, the interface
  requirement and impl must carry enough effect metadata to prove the call;
- effect checking should be based on the interface contract visible at the call
  site, not on backend optimism.

Recommended staged path:

```text
Interface requirements should eventually carry effect annotations.
V0 may start with unspecified effects and reject dyn calls from noalloc contexts
unless the interface requirement is known noalloc.
```

This is conservative and preserves Phase 12's rule that allocation-sensitive
code must not accidentally call an unknown allocating operation.

## 15. Unsafe interaction

Rules:

- unsafe implementation functions still require normal unsafe rules;
- type checking is not disabled inside interface impls;
- calling an unsafe interface method should require an unsafe call context if
  unsafe interface methods are later supported;
- unsafe interfaces are deferred unless already needed.

Phase 14 v0 should not use `unsafe` to punch through ordinary type or receiver
checks. Unsafe marks obligations; it does not erase the interface contract.

## 16. Non-goals for Phase 14 v0

Phase 14 v0 explicitly defers:

- classes;
- inheritance;
- base classes;
- subclassing;
- virtual inheritance;
- default method bodies;
- interface fields;
- associated types;
- generic interface methods;
- owning dyn boxes;
- heap boxing;
- `Box<dyn Interface>`;
- dynamic cast;
- RTTI;
- reflection;
- interface object identity;
- interface inheritance;
- multiple interface upcasts;
- downcasts;
- object-safe generic methods;
- async interface methods;
- Drop through dyn;
- vtable ABI stability across compilation units;
- cross-module/orphan-rule coherence beyond current single-module compiler.

## 17. Diagnostics

Final implemented Phase 14 diagnostic inventory:

```text
CON0240 DuplicateInterfaceRequirement
CON0241 InterfaceRequiresRequirement
CON0244 InvalidInterfaceImplTarget
CON0245 MissingInterfaceRequirementImpl
CON0246 InvalidInterfaceRequirementImplSignature
CON0247 ExtraInterfaceImplFunction
CON0248 DuplicateInterfaceImplFunction
CON0249 DynCoercionRequiresImpl
CON0250 DynCoercionRequiresPlace
CON0251 UnknownInterfaceMethod
CON0252 InterfaceCallArityMismatch
CON0253 InterfaceCallTypeMismatch
CON0254 InterfaceCallRequiresMutableDyn
CON0255 InterfaceRuntimeUnsupported
CON0256 DuplicateInterfaceImpl
CON0257 DynRequiresInterface
CON0258 DynRequiresBorrowedReference
```

Unknown interface names and unknown types use the existing name/type
diagnostics outside this Phase 14 range.

P14-M0 recorded the intended Phase 14 diagnostic range; M1-M8 implemented the
final inventory above.

## 18. Milestone plan

Proposed Phase 14 milestones:

```text
P14-M0  Design doc: interfaces and dyn dispatch
P14-M1  Interface HIR declaration
P14-M2  Interface requirement validation
P14-M3  Interface impl conformance
P14-M4  dyn interface type surface
P14-M5  concrete-to-dyn coercion scaffold
P14-M6  dyn method call HIR/MIR scaffold
P14-M7  C backend vtable lowering
P14-M8  examples/fixtures/hardening
P14-M9  Closeout
```

The implementation path should first make declarations and conformance
visible, then add the type surface, then expose coercions and calls in HIR/MIR,
and only then lower vtables in C. That order keeps runtime dispatch inspectable
instead of becoming a backend trick.

## 19. Docs index / status

P14-M0 should be reflected in PoC3 status, the fixture/design index, and the
Checkpoint 1 coverage matrix if those files are maintained in the repository.
The update should be light: Phase 14 has a design document, but compiler
behavior remains unchanged.

## 20. Close criteria

P14-M0 is successful if:

- the concept/interface/dyn distinction is documented;
- non-inheritance doctrine is explicit;
- v0 interface declaration and impl syntax direction is documented;
- borrowed `dyn Interface&` is the v0 type model;
- owning dyn boxes and heap boxing are deferred;
- concrete-to-dyn coercion rules reject temporary/lifetime ambiguity;
- HIR and MIR must expose dyn coercion and interface calls explicitly;
- C backend vtable and fat-reference lowering is sketched;
- ownership, Drop, allocation effects, and unsafe interactions are documented;
- non-goals prevent class hierarchy, RTTI, hidden heap, and object-model drift;
- milestone plan and diagnostic inventory are recorded.

Closeout completion should run:

```bash
zig build test
git diff --check
git status --short
```

## P14-M0 status

P14-M0 adds only this design document and light status/index updates. It
intentionally does not implement lexer, parser, AST, HIR, MIR, backend,
fixtures, runtime vtables, `dyn` type parsing, dyn coercions, dynamic method
calls, allocation behavior, or compiler diagnostics.

## P14-M1 status

P14-M1 represents interface declarations in HIR. Top-level `interface`
declarations are collected as `HirItem.interface_`, reserve their names in the
top-level declaration namespace, preserve declaration spans and attributes, and
own stable `InterfaceId`, `InterfaceRequirementId`, and interface-parameter IDs.

Requirement signatures are lowered in source order. Requirement names, spans,
return types, parameter names, parameter types, and parameter spans are
preserved, with type names resolved to `TypeId`. Duplicate top-level interface
names use the existing duplicate top-level declaration diagnostic. Duplicate
requirement names are rejected with `CON0240 DuplicateInterfaceRequirement`;
duplicate requirement parameter names use the existing duplicate-parameter
diagnostic.

P14-M1 adds `interface_type` only so interface names can resolve consistently
inside the type store. Interface types are non-Copy and ordinary runtime use of
interface values or pointers remains unsupported. The HIR checker reports
`CON0255 InterfaceRuntimeUnsupported` if an interface type reaches an ordinary
function signature or local.

Still unimplemented after P14-M1: `dyn` type syntax, borrowed dyn references,
concrete-to-dyn coercion, interface impl conformance, vtable representation,
interface method calls, MIR dyn/interface-call lowering, backend C interface or
vtable emission, owning dyn boxes, heap boxing, dynamic cast, RTTI, reflection,
Drop through dyn, interface inheritance, default methods, associated types, and
generic interface methods.

## P14-M2 status

P14-M2 validates the narrow v0 shape of interface requirements while keeping
interfaces HIR-only. Interface declarations must contain at least one
requirement; empty runtime interfaces are rejected with
`CON0241 InterfaceRequiresRequirement` because marker-style static behavior is
already covered by marker concepts, while empty runtime interfaces would invite
RTTI/tag-like semantics that Phase 14 v0 explicitly defers.

Requirement signatures are semantically well-formed only when names and types
stay inside the v0 contract:

- requirement names must be unique within an interface;
- overloads are rejected by that duplicate-name rule, regardless of arity;
- requirement parameter names must be unique;
- requirement return and parameter types must resolve through the ordinary type
  resolver;
- ordinary resolved types such as `void`, `int`, `bool`, structs, enums,
  machine types, and raw pointers to ordinary types are accepted;
- interface types are rejected in requirement return and parameter positions
  with `CON0255 InterfaceRuntimeUnsupported`, including raw pointers to
  interface declarations, until the explicit `dyn Interface&` type surface
  exists.

Pure interface declarations still do not lower to executable MIR concepts,
runtime dyn values, vtables, or backend C interface artifacts. A backend path
containing an otherwise ordinary `main` plus a pure interface declaration emits
only the ordinary executable code.

### P14-M3 implementation status

P14-M3 validates interface implementation conformance without introducing dyn
runtime dispatch. The existing `impl Name<Type>` parser surface now routes by
the resolved top-level `Name`: concepts keep the Phase 8 concept-impl path,
while interfaces use a separate interface-impl path.

Implemented in M3:

- `impl Interface<Type>` is recognized when `Interface` resolves to an
  interface declaration;
- interface impls are represented in HIR as distinct `HirInterfaceImpl` values
  with stable `InterfaceImplId`;
- interface impls are not stored as concept impls, and concept impls do not
  create interface impls;
- duplicate `(interface, target type)` impls are rejected with
  `CON0256 DuplicateInterfaceImpl`;
- missing required methods are rejected with
  `CON0245 MissingInterfaceRequirementImpl`;
- extra methods are rejected with `CON0247 ExtraInterfaceImplFunction`;
- duplicate impl method names are rejected with
  `CON0248 DuplicateInterfaceImplFunction`;
- method signatures are checked against requirements with
  `CON0246 InvalidInterfaceRequirementImplSignature`;
- implementation methods must include the receiver as their first parameter;
- the M3 receiver convention is `mut Target& self`, checked from the AST
  spelling because reference types are not yet first-class TypeStore entries;
- all non-receiver parameters and return types match the interface requirement
  exactly by resolved `TypeId`;
- invalid interface impl targets such as `void`, interface declarations, and
  unresolved/invalid types are rejected with `CON0244 InvalidInterfaceImplTarget`;
- interface impl method bodies are lowered and checked like existing hidden impl
  witness bodies;
- pure interface declarations and interface impl declarations still emit no C
  interface structs, dyn fat references, vtable structs, or vtable constants.

Still unimplemented after P14-M3: `dyn` type syntax, dyn type checking,
concrete-to-dyn coercion, dyn method calls, method-call syntax beyond impl
function declarations, vtable representation, backend C vtable/interface
emission, owning dyn boxes, heap boxing, dynamic cast, RTTI, reflection,
interface inheritance, default methods, associated types, generic interface
methods, Drop through dyn, effect checking through dyn, unsafe interface
methods, and cross-module/orphan interface coherence.

### P14-M4 implementation status

P14-M4 adds the borrowed dyn interface type surface without adding runtime
dispatch. The accepted source spellings are:

```cpp
dyn Writer&
mut dyn Writer&
```

Implemented in M4:

- `dyn` is a reserved keyword;
- `dyn Interface&` parses as an immutable borrowed dynamic interface
  reference;
- `mut dyn Interface&` parses as a mutable borrowed dynamic interface
  reference;
- HIR/type resolution preserves dyn interface reference parameters in function
  signatures;
- TypeStore interns dyn interface reference types as distinct from bare
  `interface_type`;
- mutable and immutable dyn references have distinct `TypeId`s;
- dyn targets must resolve to an interface declaration;
- `dyn` targeting structs, enums, concepts, compiler-known scalar/allocation
  types, and type parameters is rejected with `CON0257 DynRequiresInterface`;
- unknown dyn targets use the existing unknown-type diagnostic;
- `dyn Interface` by value and `dyn Interface*` raw pointer spellings are
  rejected with `CON0258 DynRequiresBorrowedReference`;
- ordinary bare interface runtime values and raw pointers to interfaces remain
  rejected with `CON0255 InterfaceRuntimeUnsupported`;
- dyn return types, dyn locals, dyn struct fields, and dyn use inside interface
  requirement signatures are deferred and rejected with
  `CON0255 InterfaceRuntimeUnsupported`;
- backend C emission treats dyn interface references as unsupported C types and
  does not emit fat-reference structs, vtable structs, vtable constants, or
  placeholder `cpt_dyn_*` artifacts.

Still unimplemented after P14-M4: concrete-to-dyn coercion, dyn method calls,
method-call syntax beyond impl function declarations, vtable representation,
MIR lowering for dyn fat references, backend C vtable/interface emission,
owning dyn boxes, heap boxing, dynamic cast, RTTI, reflection, interface
inheritance, default methods, associated types, generic interface methods, Drop
through dyn, effect checking through dyn, unsafe interface methods, and
cross-module/orphan interface coherence.

### P14-M5 implementation status

P14-M5 adds the concrete-to-dyn coercion scaffold at function call boundaries.
When a parameter expects `dyn Interface&` or `mut dyn Interface&`, an
addressable concrete argument may be coerced if its concrete type has a
matching `impl Interface<ConcreteType>`.

Implemented in M5:

- call arguments can coerce from an addressable concrete place to `dyn
  Interface&` or `mut dyn Interface&`;
- accepted source places include locals, parameters, and addressable field
  places already modeled by HIR/MIR;
- struct literals, function-call results, binary expressions, `move` operands,
  and other rvalues are rejected with `CON0250 DynCoercionRequiresPlace`;
- missing interface impls are rejected with
  `CON0249 DynCoercionRequiresImpl`;
- concept impls do not count as interface impls;
- impls for another interface or another concrete type do not count;
- exact dyn-to-dyn argument passing remains an ordinary exact type match;
- the coercion is represented explicitly in HIR as `dyn_coerce` with source
  expression, target interface id, selected interface impl id, and result dyn
  type;
- MIR lowering preserves the coercion explicitly as `MirRvalue.dyn_coerce`
  with source place, interface id, impl id, and result type;
- the MIR validator checks that the selected interface impl exists, targets the
  same interface, and matches the source place type;
- backend C emission still rejects dyn runtime use as unsupported and emits no
  fat-reference structs, vtable structs, vtable constants, or `cpt_dyn_*`
  artifacts;
- no heap allocation, owning dyn box, dynamic cast, RTTI, reflection, Drop
  through dyn, or interface inheritance behavior is introduced.

M5 preserves `is_mut` in the dyn interface type but does not yet enforce
method-call receiver mutability because dyn method calls are still deferred.
For this scaffold, both immutable and mutable dyn parameter coercions are
accepted from addressable concrete places when the interface impl exists.

Still unimplemented after P14-M5: dyn method calls, method-call syntax beyond
impl function declarations, interface-call HIR/MIR, vtable representation,
backend C vtable/interface/fat-reference emission, dyn locals with
initializers, dyn returns, dyn struct fields, owning dyn boxes, heap boxing,
dynamic cast, RTTI, reflection, interface inheritance, default methods,
associated types, generic interface methods, Drop through dyn, effect checking
through dyn, unsafe interface methods, and cross-module/orphan interface
coherence.

### P14-M6 implementation status

P14-M6 adds the dyn interface method-call scaffold without adding runtime
dispatch. The accepted source shape is:

```cpp
writer.Write(42);
return writer.Count();
```

Implemented in M6:

- receiver-preserving method-call parsing exists for lowercase/expression
  receivers such as `writer.Write(42)`;
- existing uppercase qualified-call parsing remains available for
  namespace-like builtins such as `Arena.alloc<T>(arena)`, `Assert.True`, and
  `Expect.That`;
- when the receiver type is `mut dyn Interface&`, semantic lowering resolves
  the method name to a requirement of that interface;
- unknown interface methods are rejected with
  `CON0251 UnknownInterfaceMethod`;
- interface method call arity mismatches are rejected with
  `CON0252 InterfaceCallArityMismatch`;
- interface method call argument type mismatches are rejected with
  `CON0253 InterfaceCallTypeMismatch`;
- P14-M6 conservatively requires mutable dyn receivers for every interface
  method call, because requirement receiver mutability is not modeled yet;
- calls through immutable `dyn Interface&` are rejected with
  `CON0254 InterfaceCallRequiresMutableDyn`;
- HIR represents dyn calls explicitly as `interface_call`, preserving the
  receiver expression, interface id, requirement id, requirement slot index,
  argument list, and result type;
- MIR lowering preserves dyn calls explicitly as `MirRvalue.interface_call`,
  with receiver operand, interface id, requirement id, slot index, arguments,
  and result type;
- the MIR validator checks that the interface and requirement ids exist, the
  slot index names the same requirement, the receiver is a dyn reference for
  the same interface, and argument count/types match the requirement;
- backend C emission reports `CON0255 InterfaceRuntimeUnsupported` for dyn
  interface runtime use and emits no vtable structs, vtable constants, dyn
  fat-reference structs, `cpt_dyn_*` helpers, or fake direct calls;
- concrete-to-dyn coercion from M5 composes with M6 at HIR/MIR level: callers
  can coerce concrete arguments into a dyn parameter, and the callee body can
  contain explicit interface calls.

Still unimplemented after P14-M6: backend C vtable/interface/fat-reference
emission, owning dyn boxes, heap boxing, dynamic cast, RTTI, reflection,
interface inheritance, default methods, associated types, generic interface
methods, Drop through dyn, effect checking through dyn beyond existing trivial
checks, unsafe interface methods, and cross-module/orphan interface coherence.

### P14-M7 implementation status

P14-M7 lowers the Phase 14 borrowed dyn dispatch subset through the C backend.
The runtime representation is explicit and inspectable in generated C: each
used interface gets a vtable struct and a dyn fat-reference struct, dyn
parameters and temps use the fat-reference value type, concrete-to-dyn
coercions build `{ data, vtable }` values, and dyn interface calls dispatch
through the selected vtable slot.

Implemented in M7:

- `dyn Interface&` / `mut dyn Interface&` parameters lower as pass-by-value
  fat references in generated C;
- used interfaces emit `cpt_itf_Interface_vtable` structs with one function
  pointer per requirement in requirement source order;
- used interfaces emit `cpt_dyn_Interface` structs containing `void* data` and
  `const cpt_itf_Interface_vtable* vtable`;
- interface impls selected by dyn coercion emit wrapper/thunk functions with
  `void* self` as the first parameter;
- wrappers cast `self` to the concrete target pointer and call the existing
  hidden interface-impl method function;
- each selected `(interface, concrete type)` impl emits one static const vtable
  whose slots point at the wrappers;
- `MirRvalue.dyn_coerce` lowers to explicit fat-reference construction with
  `.data = &source_place` and `.vtable = &static_impl_vtable`;
- `MirRvalue.interface_call` lowers to `receiver.vtable->Slot(receiver.data,
  args...)`;
- void-return interface calls in expression-statement position lower as
  statement calls rather than impossible `void` temporaries;
- simple int, bool, multiple-argument, void-call, two-impl, two-interface,
  call-boundary coercion, and exact dyn passthrough fixtures execute through
  MIR-backed C and native exit-code testing;
- pure interface declarations and unused interface impls still emit no vtable
  or dyn artifacts;
- no `malloc`, scheduler helper, RTTI table, dynamic cast metadata, reflection,
  hidden heap allocation, or per-instance vtable construction is introduced.

Borrowed dyn remains non-owning. A dyn fat reference points at existing concrete
storage and does not extend lifetime, allocate storage, own the object, or run
Drop through the dyn reference. The C backend stores and passes the fat
reference by value because it is a small pair of pointers.

Still unimplemented after P14-M7: owning dyn boxes, heap boxing,
`Box<dyn Interface>`, dynamic cast, RTTI, reflection, interface inheritance,
interface upcasting, default methods, associated types, generic interface
methods, Drop through dyn/destructor vtable slots, cross-module vtable ABI
stability, unsafe interface methods, broader effect checking through dyn, and
first-class dyn returns/fields/locals with initializers. Receiver mutability is
still conservative: M6's rule that dyn calls require `mut dyn Interface&`
remains in force.

### P14-M8 implementation status

P14-M8 stabilizes the borrowed dyn subset without adding new runtime object
features. It adds source examples under `examples/phase14/`, strengthens the
fixture corpus, and pins the generated C shape more tightly.

Implemented in M8:

- examples document int-return dyn dispatch, void-return dyn calls, two
  concrete impls for one interface, and exact dyn passthrough at a call
  boundary;
- the runtime subset remains borrowed dyn references only: concrete-to-dyn
  coercion builds a `{ data, vtable }` fat reference from an addressable place
  and a matching interface impl;
- fixture coverage now pins mutable dyn calls as accepted and immutable dyn
  calls as rejected with `CON0254 InterfaceCallRequiresMutableDyn`;
- missing impls, temporaries, dyn locals with initializers, dyn returns, dyn
  fields, unknown methods, and wrong argument types remain explicitly rejected;
- backend-C assertions cover vtable structs, dyn fat-reference structs,
  `void* data`, `const cpt_itf_X_vtable* vtable`, wrapper thunks taking
  `void* self`, wrapper casts to concrete pointers, hidden impl method calls,
  static vtable constants, source-order vtable slots, fat-reference
  `.data = &source_place` / `.vtable = &impl_vtable` construction, indirect
  `.vtable->Method(.data, ...)` calls, and reuse of one vtable constant across
  repeated coercions to the same impl;
- backend assertions also pin the absence of hidden heap allocation, RTTI,
  dynamic-cast helpers, reflection helpers, scheduler/async helpers, and
  class/inheritance-like output for the supported dyn path;
- pure unused interface declarations and unused interface impls still emit no
  dyn/vtable artifacts.

Receiver ABI limitation:

```text
Receiver references are not yet first-class TypeStore values.

M8 verifies the current wrapper path:
    void* self -> ConcreteType* typed -> hidden impl ABI call with *typed

This is enough for read-style dispatch and side-effect-free void calls, but
mutation observed through dyn dispatch is not hardened yet and remains
deferred until the reference TypeStore/ABI model is made first-class.
```

Still unimplemented after P14-M8: owning dyn boxes, heap boxing,
`Box<dyn Interface>`, dynamic cast, RTTI, reflection, interface inheritance,
interface upcasting, default methods, associated types, generic interface
methods, Drop through dyn/destructor vtable slots, cross-module vtable ABI
stability, unsafe interface methods, broader effect checking through dyn,
first-class dyn returns/fields/locals with initializers, and the full
first-class reference TypeStore model needed to harden mutation-through-dyn.

### P14-M9 closeout status

P14-M9 closes Phase 14 without adding new feature work. The final audit keeps
the M8 runtime subset intact: borrowed dyn refs only, call-boundary coercion
from addressable concrete places only, `mut dyn Interface&` required for dyn
method calls, no owning boxes, no hidden heap allocation, no RTTI, no dynamic
cast, no reflection, and no inheritance or class hierarchy.

The fixture corpus remains at 725 fixtures. The Phase 14 corpus covers valid
interface declarations, requirement validation, interface impl conformance,
concept/interface separation, borrowed dyn parameter types, call-boundary
coercion, exact dyn passthrough, dyn interface calls, executable C-backed dyn
dispatch, backend C shape assertions, and the unsupported forms listed above.
No extra closeout fixture was added because the existing focused fixtures
already cover the proposed closeout smoke path.
