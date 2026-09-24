# Octagon data interop (R7h2 progression)

Octagon is the shared data language. Concept concepts are the shared schema
language. Reflection generates ordinary typed conversion code.

`Standard.Octagon.Core` currently exposes byte-span readers and writers and an
ordinary `OctagonCodec<T>` requirement. `Standard.Octagon.Derive` expands
reflected fields into checked reader/writer functions for records and columnar
tables whose field types satisfy `OctagonCodec<T>`. `FieldType<field>` and
`OctagonType<T>` select the typed
reader through ordinary overload resolution. It performs direct typed parsing;
it has no dynamic object graph.
The writer emits the Oct record form `Type { field: value }` in declaration
order. The reader returns `Result<T, OctagonError>`.

The `int` and `bool` codecs and generated mixed-record round trip are implemented.
`ReadOctagon<T, N>` and `WriteOctagon<T, N>` handle fixed arrays when the
element operations satisfy their ordinary concepts. Tests execute generated
integer, boolean, record, and nested array fields, plus a derived
`record table<3>` with complete integer and boolean columns through strict C11.
The nested payload enum codec remains handwritten. General enum derivation,
other scalar types, and refinement admission are not implemented.

`WriteOctagon` writes one composable Octagon data expression. Call
`FinishOctagon(ref writer)` once at the document boundary to emit Oct's
canonical trailing newline. Nested values do not finish a document.

A concept witness that is Proven for a closed type must be executable through
the same ordinary required-operation identity. Concept does not maintain
separate proof-only and call-only witness universes. Closed array calls infer
both element type and extent and use ordinary constrained template
instantiation. `ReadNamed`/`WriteNamed` delegate to that witness from a
generated field call.

Aggregate construction initializes fields directly into a not-yet-live
aggregate. The aggregate becomes live only after all required fields are
successfully initialized. Named syntax is `T{field = expression, ...}`.
The compiler checks complete coverage and unique names, evaluates expressions
in source order, and drops completed owning fields if a later expression
propagates failure.
No runtime codec registry is involved.
