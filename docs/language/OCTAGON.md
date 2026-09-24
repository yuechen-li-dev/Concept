# Octagon data interop (R7h1)

Octagon is the shared data language. Concept concepts are the shared schema
language. Reflection generates ordinary typed conversion code.

`Standard.Octagon.Core` currently exposes byte-span readers and writers and an
ordinary `OctagonCodec<T>` requirement. `Standard.Octagon.Derive` expands
reflected fields into checked reader/writer functions for records whose fields
are `int` or `bool`. `FieldType<field>` and `OctagonType<T>` select the typed
reader through ordinary overload resolution. It performs direct typed parsing;
it has no dynamic object graph.
The writer emits the Oct record form `Type { field: value }` in declaration
order. The reader returns `Result<T, OctagonError>`.

The `int` and `bool` codecs and generated mixed-record round trip are implemented.
Other scalar types, payload enums, arrays, tables, and refinements remain
outside this implementation; they must not be inferred from the concept name.

Aggregate construction initializes fields directly into a not-yet-live
aggregate. The aggregate becomes live only after all required fields are
successfully initialized. Named syntax is `T{field = expression, ...}`.
The compiler checks complete coverage and unique names, evaluates expressions
in source order, and drops completed owning fields if a later expression
propagates failure.
