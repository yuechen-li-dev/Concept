# Phase 12 Store Examples

These examples show the P12-M8 ID-store pattern using only the current Concept
surface.

P12-M8 does not add a generic `Store<T, Id>`, vector storage, arena-backed
runtime stores, generation counters, stale-ID tracking, or hidden heap behavior.
The point is narrower:

```text
Raw arena pointers are low-level.

ID stores are the preferred compiler-data pattern.
```

Compiler-shaped data such as AST, HIR, MIR, symbol, type, and diagnostic stores
should expose typed IDs as public handles. Store internals may later use arenas,
pages, segmented arrays, vectors, or fixed buffers, but callers should pass
`ExprId`, `TypeId`, `BasicBlockId`, and similar value handles rather than
long-lived raw node pointers.

The `.concept` files here are examples, not a complete container library. The
single-entry store is deliberately tiny so it can be expressed with existing
structs, field access, struct literals, `Copy<T>` marker impls, and `noalloc`
functions.

