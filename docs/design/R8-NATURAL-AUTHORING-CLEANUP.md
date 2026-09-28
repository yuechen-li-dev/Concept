# R8 natural-authoring cleanup: landed semantics and boundary

R8g implementation commit `cb82976c9547c883aaf4cac6f192a7d2f1bcbda8` keeps Concept semantic types as the authority; generated C is checked output.

## Landed source shapes

```concept
float Plain(float x) { return x; }
float Explicit(float<m> distance) { return Plain(Magnitude(distance)); }
float<m> Attach(float raw) { return interpret raw as float<m>; }
float<mm> Convert(float<m> distance) { return distance as float<mm>; }
```

The implicit assignment/call/return routes from `float<m>` to `float` now reject. An explicit `as float` remains rejected with `Magnitude(value)` guidance. Imported `Standard.Math` functions with plain float signatures cannot erase a quantity. Their typed quantity equivalents are still open work.

```concept
struct Inner { uint8[4] bytes; int count; }
Inner Make() { return Inner{[0, 1, 2, 3], 4}; }
```

The array literal uses the field's element type during validation and C lowering, including nested constructors, function arguments, enum payloads, and closed generic aggregates. Constant folding yields to the typed field path when the aggregate contains an array literal.

```concept
double f(double x) { return x * x - 1.0; }
double g() { return -1.0; }
```

Only a literal acquires the expected floating representation. The parsed value remains binary64 until target rendering, and a target range check rejects values outside half/float/double representation. Existing runtime conversion rules are unchanged.

```concept
ref Page page = FindPage(ref cache, id) ? else CacheError::Missing;
```

The `else` expression is evaluated once on the error branch. The successful Result payload is extracted without rebuilding it. Its borrow provenance follows the source Result; bounded `if` return summaries now combine the shortest provenance from branches. The remap is a return from the caller and requires a caller `Result` error type matching the `else` expression.

Payload-free enums use a stable 32-bit tag member independent of case count. Their 4-byte geometry admits arrays, tables, and structs used as Span elements. Payload enums retain the previous representation restriction. No explicit enum backing syntax was introduced.

## Next design decision

`requires T Multiply(T, T)` exists for named functions, while `requires T operator*(T, T)` has no parser, declaration identity, witness resolution, or artifact closure. Open binary operators are rejected with `CV4175`. A general quantity-preserving `Standard.Math` template and F2 require one coherent operator authority, including primitive, quantity/tensor, and future user-defined operations. The current batch leaves these calls rejected rather than adding a Math-specific compiler exception.
