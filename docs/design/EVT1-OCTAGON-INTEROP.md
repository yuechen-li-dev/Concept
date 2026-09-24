# EVT1 Octagon interop

Octagon is the shared data language. Concept concepts are the shared schema
language. Reflection generates ordinary typed conversion code.

The R7h1 reader and writer are ordinary Standard functions over byte spans.
`derive` expands a reflected integer record into `ReadOctagon` and
`WriteOctagon`. The reader uses checked, complete named aggregate
construction; the writer reads typed fields directly. Generated declarations
retain R7g1 provenance through semantic artifacts. No serializer operation
was added to MIR, and no runtime reflection registry or DOM exists.

This implementation is bounded to integer records. The Oct repository owns
payload enum data syntax and nominal validation. Its canonical payload form
is `Enum.Case(data)`; recursively nested arguments must be Octagon data.
