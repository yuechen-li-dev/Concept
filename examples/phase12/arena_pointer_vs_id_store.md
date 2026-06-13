# Arena Pointers vs ID Stores

Arena allocation is explicit and useful, but `Arena.alloc<T>(arena)` returns a
raw pointer. That pointer is a low-level value whose storage is invalidated by
`Arena.reset(arena)` or `Arena.destroy(arena)`.

Compiler data should usually avoid exposing long-lived raw arena pointers as the
graph shape:

```cpp
Expr* expr = Arena.alloc<Expr>(arena);
```

The preferred compiler-facing shape is a typed ID returned by a store:

```cpp
ExprId id = ExprStoreAdd(store, expr);
Expr expr = ExprStoreGet(store, id);
```

P12-M8 keeps the implementation as ordinary structs and functions. The final
store syntax may later be method-shaped:

```cpp
ExprId id = ExprStore.add(store, Expr.IntLiteral(42));
Expr expr = ExprStore.get(store, id);
```

The semantic contract is the important part:

- the store owns the backing storage;
- the public handle is a typed ID, not an untyped integer;
- `ExprId` is not interchangeable with `TypeId`;
- IDs are small value handles and do not own stored objects;
- IDs are comparable, serializable, diagnostic-friendly, and stable across
  backing storage movement;
- store internals may later use arenas, pages, segmented arrays, vectors, or
  other storage;
- if a store is arena-backed, its IDs are invalidated when the store or backing
  arena is reset or destroyed;
- P12-M8 does not implement stale-ID tracking or region lifetime checking.

