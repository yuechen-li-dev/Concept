# Phase 14 borrowed dyn examples

These examples show the supported Phase 14 borrowed-dyn runtime subset.

- `dyn Interface&` and `mut dyn Interface&` are borrowed dynamic references.
- A dyn reference is a fat reference: `{ data, vtable }`.
- Dyn fat references are passed by value in generated C.
- Concrete-to-dyn coercion requires an addressable concrete place and a matching `impl Interface<ConcreteType>`.
- Concrete-to-dyn coercion is call-boundary only.
- Temporaries, call results, binary expressions, and `move` operands are not valid coercion sources.
- Dyn coercion does not allocate and does not create an owning box.
- Owning dyn boxes, heap boxing, RTTI, dynamic casts, reflection, inheritance, and interface upcasting are not part of Phase 14 v0.
- There is no hidden heap allocation, no RTTI, no dynamic cast, and no inheritance/class hierarchy in the supported lowering.
- Current v0 dyn method calls require `mut dyn Interface&`.
- Receiver references are not yet first-class TypeStore values, so mutation-through-dyn remains a documented backend ABI limitation. The examples avoid relying on mutation observed through dyn dispatch.
