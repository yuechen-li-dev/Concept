# R7h1 conformance status

R7h1 is partially conformant. Named complete aggregate construction,
generated aggregate construction through artifacts, failure cleanup, Oct
payload enum loading/emission in interpreted and compiled paths, and a
generated mixed `int`/`bool` Concept record round trip have focused executable
tests. Type-directed field selection uses the general `FieldType<field>`
reflection substitution and ordinary overloads. Oct's relevant compiled
Octagon load corpus now passes without fallback.

Full R7h1 conformance is pending payload enum and non-integer Concept codecs,
arrays, columnar tables, refinements, cross-language goldens, downstream
artifact codec use, and 100-run determinism. See
`EVT1-R7H-CONVERGENCE.md` for evidence and exact baselines.
