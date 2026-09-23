# EVT1 generic library closure

An operation requirement may declare its own type parameters:

```concept
concept Transformable<T> {
    requires template <typename U> U Transform(ref const T value, U seed);
}
```

The outer `T` and operation-local `U` are separate bindings. Concrete witness matching renames the implementation's generic type parameters structurally, then compares the full parameter and result types. Closed calls use ordinary template instantiation, MIR and C11 generation. Required generic operations are checked when called; they are not eagerly instantiated for every possible `U`. The module artifact transports the requirement's typed generic parameter list.

An open generic body may call another generic body. A call to a constrained callee is accepted when the caller has the same direct concept requirement; unsupported entailment remains a directed constraint error. Closed instantiation has a recursion/depth guard. Generated C declares closed instances before definitions, so imported callers can invoke local implementations regardless of source order.

`T<array>[N]` may remain open only when each symbolic extent is directly a non-type generic value parameter. At closure the ordinary extent validator and storage lowering see a concrete length and emit a fixed C11 array type. Runtime-valued lengths still require explicit storage. The local array test covers two `N` values, initializer repetition, artifact-only import, and strict C11 execution.

Multi-type-parameter concepts already permitted value and reference operation arguments. The R7f CV4102 diagnostic concerned a non-type concept parameter. R7f1 does not add non-type parameters to concept declarations.
