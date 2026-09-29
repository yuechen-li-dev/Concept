# R8d convergence record

R8d starts from clean R8c HEAD `f268d85816a16097de528bee5bf5af73185bab31`
with compiler `concept-evt1-stage0-go`. Its narrow semantic rule is scalar to
same-representation quantity only. Numeric and unit-scale conversion stay in
`as`; current-unit scalar extraction stays in `Magnitude`.

The authored boundary syntax is `interpret expression as T`. It is retained
as a separate typed expression and MIR operation. C lowering erases the
semantic operation to the original scalar expression. The artifact envelope
retains the interpretation's source, target, site, and declared origin. Explain
reports the origin and explicitly limits its conclusion to the programmer's
declaration, without upgrading the external convention to a universal proof.

## Regression gates

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | Pass, including the full semantic corpus |
| `go vet ./...` | Pass |
| Root and `legacy/poc3-zig` `zig build test` | Pass |
| Standard Normal / Verify | 35 / 35 in each mode |
| DragonGod Normal / Verify | 23 / 23 in each mode |
| R7p goldens plus R7q Frame Normal / Verify | 28 / 28 in each mode |
| R8d native, strict C11, artifact, comptime, generic, async, explain and 100-run determinism | Pass |
| Semantic corpus | 406 valid, 279 static-invalid, 13 runtime-negative, 5 compatibility, 4 expected-divergence entries |

The existing Go suite also includes BurnIn and the R8a/R8b/R8c focused
regressions. The legacy
`AssumeQuantity<T>` primitive remains source-compatible for historical tests.
New authored golden and corpus examples use `interpret`.

R8e remains project policy and first-contact tooling: `must_use`, explicit
discard, project concepts, foreign C/C++ policy concepts, `concept lint`,
`concept fmt` policy/configuration, naming conventions, and remaining
diagnostics. R8f follows with resumed differentiator goldens. R7q stays
paused until the R8 semantic cleanup series is complete.
