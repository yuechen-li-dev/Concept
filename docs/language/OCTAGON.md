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
`ReadOctagon<T, N>` and `WriteOctagon<T, N>` handle fixed arrays when the
element operations satisfy their ordinary concepts. The tests execute an
integer array, a nested payload enum, and a `record table<3>` with complete
integer and boolean columns through strict C11. The latter two use ordinary
handwritten `OctagonType<T>` overloads; general enum/table derivation is still
pending. Other scalar types and refinement admission are not implemented.

`WriteOctagon` writes one composable Octagon data expression. Call
`FinishOctagon(ref writer)` once at the document boundary to emit Oct's
canonical trailing newline. Nested values do not finish a document.

The current fixed-array template must be called with explicit element type and
extent. Concept's closed concept witness lookup does not yet recognize this
constrained two-parameter template as `OctagonCodec<T<array>[N]>`, so a
generated record cannot yet dispatch an arbitrary array field through
`ReadNamed`/`WriteNamed`.

Aggregate construction initializes fields directly into a not-yet-live
aggregate. The aggregate becomes live only after all required fields are
successfully initialized. Named syntax is `T{field = expression, ...}`.
The compiler checks complete coverage and unique names, evaluates expressions
in source order, and drops completed owning fields if a later expression
propagates failure.
