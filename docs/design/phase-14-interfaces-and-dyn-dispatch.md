# Phase 14: Interfaces and dyn dispatch

P14-M0 is a documentation-only milestone. It defines Concept's runtime
interface model, borrowed `dyn` references, concrete-to-dyn coercion direction,
HIR/MIR representation, and C backend lowering plan.

P14-M0 does not implement compiler behavior. It does not change lexer, parser,
semantic, HIR, MIR, backend, fixture, or runtime behavior. Existing parser
support for `interface` signature blocks remains only the starting point for
later milestones.

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

Suggested initial diagnostic inventory:

```text
CON0240 DuplicateInterfaceRequirement
CON0241 InterfaceRequiresBody
CON0242 InterfaceRequirementInvalid
CON0243 UnknownInterface
CON0244 InvalidInterfaceImplTarget
CON0245 MissingInterfaceRequirementImpl
CON0246 InvalidInterfaceRequirementImplSignature
CON0247 ExtraInterfaceImplFunction
CON0248 DynRequiresInterface
CON0249 DynCoercionRequiresImpl
CON0250 DynCoercionRequiresPlace
CON0251 UnknownInterfaceMethod
CON0252 InterfaceCallArityMismatch
CON0253 InterfaceCallTypeMismatch
CON0254 InterfaceCallRequiresMutableDyn
CON0255 InterfaceRuntimeUnsupported
```

P14-M0 does not implement these diagnostics. They are a planning list so later
milestones can produce specific errors instead of generic unsupported-runtime
failures.

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
