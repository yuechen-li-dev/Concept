# Tables

`table` is a struct-shaped columnar aggregate whose columns share one row
count. It is an in-memory data-layout feature, not a database, relation,
dataframe, query engine, or runtime schema system.

```concept
table<4> Point
{
    float x;
    float y;
}
```

`table<4> Point` has two contiguous columns, conceptually `x[4]` followed by
`y[4]`. It is not `Point[4]` and is never lowered as an array of row structs.
`points.x` is the ordinary fixed-array column, so indexing and the existing
array-to-`Span<T>` conversion apply without a table-specific runtime.

The fixed cardinality is part of semantic type identity and layout. Every
column has exactly that extent. `SizeOf` and `AlignOf` follow the ordinary
aggregate and fixed-array layout rules, and strict C11 lowering is an ordinary
struct whose fields are fixed-array storage wrappers. No runtime row-count
metadata is emitted.

Existing aggregate modifiers retain their existing meaning:

```concept
record table<8> Samples { int value; }
ref table<8> SamplesView { int value; }
```

`record table` uses record value/equality laws. `ref table` uses scoped
ref-aggregate laws. Neither form introduces new copying, movement, lifetime,
or borrowing rules.

Generic cardinality uses the existing non-type parameter system:

```concept
template <typename T, usize N>
table<N> Samples
{
    T value;
    uint64 timestamp;
}
```

The current milestone admits plain `table Name { ... }` as an unsized schema
declaration only. Owning/runtime storage for that schema is rejected with
`TABLE_CARDINALITY_REQUIRED`; there is no hidden heap or implicit allocator.
Dynamic owning tables and row-proxy projection are deferred. Fixed tables use
column-shaped aggregate construction, for example `Point{[0.0 ...], [1.0 ...]}`.

