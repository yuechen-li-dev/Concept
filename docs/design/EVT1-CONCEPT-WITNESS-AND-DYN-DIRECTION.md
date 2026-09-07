# EVT1 concept witness and dyn direction

Status: R4b semantic-proof direction; runtime reification is not implemented

## Model

```text
concept C
    semantic contract

witness(T, C)
    evidence that concrete T satisfies C

template <T satisfies C>
    consumes a compile-time witness

dyn C
    runtime-erased value or reference + reified witness

interface
    optional sugar or marker for a dyn-compatible concept
```

Concept requirements may be ordinary operation signatures, prerequisite
concepts, static predicates in a future bounded form, or compiler-known
semantic analyses. R4a implements a small internal compiler-analysis registry
and proves `LifetimeSafe<T>` through the ordinary concept-satisfaction path.
The requirement resolver remains the single owner of satisfaction; templates
do not contain a parallel lifetime checker.

R4b permits a compiler analysis to consume bounded semantic subjects from one
required operation. `Outlives(source, result)` produces inspectable relational
proof evidence containing the selected parameter, result, provenance facts,
outcome, and requirement origin. This extends the compile-time witness with a
semantic proof; it does not expand source concepts beyond one type parameter.

Compile-time satisfaction is authoritative. A template specialization uses
the selected operations and semantic proof results statically. Runtime witness
reification, if added, is downstream of that same satisfaction result rather
than an independent interface implementation subsystem.

Runtime dyn reification must therefore originate from an already-satisfied
compile-time semantic contract. Relational lifetime evidence is compile-time
only and must not be copied into a runtime witness table unless a future,
separately specified runtime invariant actually needs it.

## Runtime direction

A future `dyn C` contains or references erased data and carries a reified
witness sufficient for the operations admitted by `C`. Storage policy must be
explicit: dyn must not imply hidden allocation. Inline, borrowed, owned,
arena-backed, or other representations require their own visible laws.

An `interface` declaration is therefore not planned as a parallel nominal
abstraction hierarchy. It may become syntax for declaring that a concept is
dyn-reifiable, or remain unnecessary if `dyn C` is sufficient.

## Boundaries

R4b does not implement `dyn`, interface syntax, witness tables, object safety,
associated types, multi-parameter source concepts, ABI layout, allocation, or
ownership rules for erased values. Future unsafe should ideally bypass a
specific semantic requirement, for example `unsafe(LifetimeSafety)`, rather
than disabling all compiler checking. No unsafe bypass is implemented here.
