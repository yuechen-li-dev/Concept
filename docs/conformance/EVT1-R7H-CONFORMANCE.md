# R7h conformance status

R7h2: closed generic witness proof and ordinary calls infer the same type and
value arguments. Generated codecs cover scalar record fields, fixed arrays,
nested arrays, payload enums, nested payloads, columnar tables, and a nominal
single-field refinement through ordinary `OctagonCodec<T>` delegation.
Standard and the A-to-B-to-C artifact regression execute through strict C11.
Concept generated codecs consume and reproduce shared canonical Octagon bytes;
Oct loads the same array, nested-array, payload, table, and refined fixtures in
interpreted and compiled modes with zero fallback. The family artifact, MIR, C,
and header outputs, plus an array codec concept-explain graph, are byte-identical
over 100 runs.
Nested runtime error paths and a conditional `NoAllocation` proof remain
unqualified; see `EVT1-R7H-CONVERGENCE.md`.

R7h1 is partially conformant. Named complete aggregate construction,
generated aggregate construction through artifacts, failure cleanup, Oct
payload enum loading/emission in interpreted and compiled paths, and a
generated mixed `int`/`bool` Concept record round trip have focused executable
tests. Type-directed field selection uses the general `FieldType<field>`
reflection substitution and ordinary overloads. Oct's relevant compiled
Octagon load corpus now passes without fallback.

The R7h1 historical checkpoint included handwritten typed nested-payload-enum
and columnar-table codecs. R7h2 now derives those shapes and retains manual
codecs as valid ordinary alternatives. See `EVT1-R7H-CONVERGENCE.md` for the
earlier blocker history and current qualification evidence.
