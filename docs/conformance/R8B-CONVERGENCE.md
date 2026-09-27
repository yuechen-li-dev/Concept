# R8b convergence record

The implementation started from clean R8a HEAD
`60f7d9498e7874a710b75fe2a1e11d9cb797b50b` with compiler
`concept-evt1-stage0-go`. The baseline full Go test, Go vet, and both Zig
test suites passed. Existing arithmetic uses checked runtime traps for invalid
integer operations; ordinary `Result<T,E>` carries recoverable failures.

| R8a scalar | Alias | Representation | Unit position |
| --- | --- | --- | --- |
| `half` | `float16` | binary16, GCC extension lane | optional independent `<unit>` |
| `float` | `float32` | binary32 | optional independent `<unit>` |
| `double` | `float64` | binary64 | optional independent `<unit>` |

Before R8b there was no general explicit numeric cast. Runtime numeric
compatibility required exact representations (with the narrow pre-R6g
byte-quantity compatibility bridge); contextual integer and floating literals
were typed against their destination. `Standard.Math` already provided
`Floor`, `Ceil`, `Round`, and `Trunc` returning `float`. `Round` uses C11
`roundf`, whose halfway policy differs from R8b `RoundTo` ties-to-even.

The cast syntax shares one AST and validator path. Integer casts check range
before C conversion. Named float-to-integer operations return the ordinary
`Result` carrier and use a single compiler-owned `NumericCastError` enum.
The artifact encoder registers the new AST node, and MIR records each
conversion operation rather than burying it in C text.

Focused tests pin runtime values and failures, `?` propagation, type-wide
exactness proof, artifact-only use, Verify mode, strict C11 compilation,
GCC half extension behavior, NoAllocation, and 100-run byte identity.

## Regression gates

| Gate | Result |
| --- | --- |
| `go test ./...` | Pass on the serial full run |
| `go vet ./...` | Pass |
| Root and retired PoC3 `zig build test` | Pass |
| BurnIn R7d3 | Pass |
| Standard Normal / Verify | 35 / 35 facts pass in each mode |
| DragonGod Normal / Verify | 23 / 23 facts pass in each mode |
| R7p goldens and R7q geometry Normal / Verify | 28 / 28 facts pass in each mode |
| R7p native companion and R8a focused suite | Pass |
| R8b focused suite | Pass |

An earlier full Go run overlapped the corpus validation commands and failed
`TestR7eNativeWorkers` with harness exit 6: one two-worker trial observed
fewer than two participating workers. The isolated test then passed with both
two- and four-worker participation, and the subsequent serial full Go run
passed. No scheduler code was changed for R8b.

R8c–R8f remain as listed in the design record. No unit conversion or
`interpret` behavior was introduced.
