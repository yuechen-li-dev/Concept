# R7h1 conformance status

R7h1 is partially conformant. Named complete aggregate construction,
generated aggregate construction through artifacts, failure cleanup, Oct
payload enum loading/emission in interpreted and compiled paths, and a
generated mixed `int`/`bool` Concept record round trip have focused executable
tests. Type-directed field selection uses the general `FieldType<field>`
reflection substitution and ordinary overloads. Oct's relevant compiled
Octagon load corpus now passes without fallback.

The continuation additionally proves a fixed integer-array round trip and
handwritten typed nested-payload-enum and columnar-table round trips in strict
C11. Their bytes match Oct's canonical fixtures. Oct loads the catalog fixture
in interpreted and compiled modes, and its writer reproduces the payload and
catalog fixtures over 100 runs. A general enum/table derived codec, nested
array family, refined values, generated array fields, and complete
cross-language families remain open. See
`EVT1-R7H-CONVERGENCE.md` for evidence and exact baselines.
