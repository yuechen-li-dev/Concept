# EVT1 Octagon interop

Octagon is the shared data language. Concept concepts are the shared schema
language. Reflection generates ordinary typed conversion code.

The reader and writer are ordinary Standard functions over byte spans.
`derive` expands a reflected record or sized columnar table into `ReadOctagon` and
`WriteOctagon`. Reflected payload enums use `DeriveOctagonEnumRead` and
`DeriveOctagonEnumWrite`; single-field nominal refinements use the corresponding
`DeriveOctagonRefinedRead` and `DeriveOctagonRefinedWrite` generators. The record
reader uses checked, complete named aggregate
construction; the writer reads typed fields directly. Generated declarations
retain R7g1 provenance through semantic artifacts. No serializer operation
was added to MIR, and no runtime reflection registry or DOM exists.

The reader carries `OctagonType<T>` as an ordinary type witness. In each
reflected field expansion, `FieldType<field>` closes that witness to the field
type, so normal overload resolution chooses `ReadNamed`. This also permits
multiple generated record readers in one module.

Generated derivation covers codec-backed scalar and fixed-array fields,
including nested arrays and record arrays, and sized tables as complete
column arrays. Generated enum readers use an ordinary case test, typed payload
reads, and ordinary case construction; generated writers use exhaustive matches
and ordinary recursive codec calls. A refined reader invokes the author-supplied
admission operation after reading the underlying value. The Oct
repository owns payload enum data syntax and nominal validation. Its canonical
payload form is `Enum.Case(data)`; recursively nested arguments must be
Octagon data. An Octagon document has one final newline, emitted by
`FinishOctagon` after the composable value writer returns.

Closed generic witness proof and ordinary call resolution share type and
value-argument inference. The executable call still passes through ordinary
constraint checking and C11 monomorphization; there is no serializer-specific
compiler branch or runtime witness dispatch.

Semantic artifacts carry generated declarations and provenance. An A-to-B-to-C
regression derives array, payload, table, and refined codecs in B from imported
semantic artifacts and executes them from C without rerunning B's generator.
The shared fixtures are
loaded by Oct's interpreted and compiled paths, while Concept's generated
readers and writers execute through strict C11.
