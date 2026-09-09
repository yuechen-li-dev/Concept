# EVT1 concept witness and dyn direction

Status: R5h async callable composition implemented over R4k reification

## One contract system

```text
concept C
    generalized semantic contract

interface I<T>
    dyn-compatible specialized concept

Witness<I, T>
    deterministic evidence that T satisfies I

template<T satisfies I>
    compile-time witness consumer and static dispatch

dyn I
    erased non-owning reference plus runtime witness
```

The ordinary concept resolver is the sole satisfaction authority. An
interface does not create an `implements` table or a second matching engine.
Public struct and class methods are normalized to ordinary callable signatures
before matching. Compiler semantic-fact requirements are proven by the same
resolver at dyn construction and erase before runtime.

## Static and runtime witnesses

Static template specialization consumes the resolved operations and proofs
without emitting runtime dispatch. A concrete `ref T` to `dyn I` conversion
reifies the same satisfaction result as one deterministic witness identified
by `(I, T)`. Its runtime entries bind required methods and mechanically
synthesized field getters/setters. Interface prerequisites are flattened into
the witness entries; their semantic proof identity remains inspectable in MIR.

Mutable field requirements receive getter and setter entries. `requires const`
fields receive only a getter. Readonly dyn values use `const void*` storage and
cannot recover mutation through a method receiver or field setter.

## Representation and lifetime

```c
typedef struct {
    void* object;
    const concept_I_Witness* witness;
} concept_dyn_I;
```

`dyn I value = ref place;` stores the place address and the address of a static
const witness table. It performs no allocation, copy, relocation, or ownership
transfer. The dyn local retains the source place's lexical/scoped provenance,
so the existing reference escape analysis rejects an outward escape. This also
allows an immovable value to be viewed through dyn because the value never
moves.

Objects do not carry vtables, headers, RTTI, registries, or universal base-type
metadata. Runtime dispatch can recover only the interface operations present
in its witness; there is no downcast or type recovery.

R4l leaves satisfaction and deterministic witness construction entirely in
semantic analysis. The Planner consumes the witness-bearing MIR and selects
`ObjectWitnessPair`, `WitnessIndirect`, or `WitnessFieldAccessor`; the C backend
consumes those strategies. Planning cannot create a witness, change an entry,
allocate an object, or add a per-object vtable.

## Boundary

R4k implements only borrowed runtime erasure. `owned dyn`, inline erased
storage, heap-backed erasure, and allocator integration remain deferred and
must name an explicit storage policy. Associated types, open generic runtime
methods, arbitrary compile-time runtime payloads, inheritance, and a generalized
object model are outside this design.

## Async method entries

R5h treats `requires async T Method(ref U self);` as the ordinary callable
signature `(ref U) -> Async<T>`. Satisfaction, composition, witness identity,
const checking, and prerequisite flattening are unchanged. One ordinary witness
entry returns `concept_async_operation` by value and its adapter invokes the
concrete generated constructor exactly once.

Dynamic dispatch ends at construction. The returned operation owns concrete
generated step functions and inline frames; it does not consult an async vtable
on each Step. Await adopts those frames into the ordinary bounded machine
stack. The operation's concrete `self` frame field retains receiver provenance,
so local or scoped backing cannot escape through witness dispatch. Static facts
still erase, and mixed sync/async entries share one static witness table.

Large by-value returns are an internal strict-C11 choice, not a frozen public
ABI. A future Planner may select caller-provided construction storage only if
that choice is explicit, allocation-free, and semantically identical. Owning
dyn, virtual coroutine objects, open generic runtime methods, generalized
callables, and hidden storage remain outside R5h.
