# R8c convergence record

The work started from clean R8b HEAD
`13613eddd18ae62a1f6710de825bd335d54da189` with compiler
`concept-evt1-stage0-go`. R8b already had a normalized eight-axis quantity
identity and rational scale fields, but only base names, `Hz`, and `byte` were
accepted. Multiplication and division already composed exponent vectors;
R8c extends the existing authority with a finite catalog, checked exact scale
arithmetic, scientific quantity literals, explicit scaled casts, unit facts,
and `Magnitude`.

The first tensor pressure specimen exposed invalid C identifiers for negative
dimension exponents. The type identity encoder now spells a negative exponent
as `negN`, and the native tensor C11 harness passes. A scaled dimensionless
ratio such as `float<mm> / float<m>` is lowered with its rational factor;
integer cases that would need fractional scaling reject.

## Regression gates

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | Pass; full semantic corpus included |
| `go vet ./...` | Pass |
| Root and `legacy/poc3-zig` `zig build test` | Pass |
| BurnIn R7d3 | Pass |
| Standard Normal / Verify | 35 / 35 facts pass in each mode |
| DragonGod Normal / Verify | 23 / 23 facts pass in each mode |
| R7p goldens plus R7q Frame Normal / Verify | 28 / 28 facts pass in each mode |
| R8c focused native, artifact, comptime, proof, C11, Verify and determinism tests | Pass |
| Semantic corpus | 404 valid, 275 static-invalid, 13 runtime-negative, 5 compatibility, 4 expected-divergence entries |

The 100-run deterministic fixture compares its semantic artifact and every
generated output, including MIR, C, and proof output. Generated quantity
operations remain plain scalar C; no runtime unit object or registry is
emitted. `concept explain` for the millimeter scale fact displays normalized
length dimensions and the exact ratio `1/1000`.

R8d `interpret value as T`, boundary attachment/stripping policy, and
replacement of user-facing `AssumeQuantity` remain deferred. R8e policy
concepts and R8f resume differentiator goldens remain separate milestones.
Domain-unit declaration syntax and standalone compile-time unit values are
not introduced here.
