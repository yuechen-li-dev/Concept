# EVT1 R7n convergence

Baseline: clean `bf84656b47c79b1d623b1f7593cceaa46573a1ea` after R7m3.
Compiler ID: `concept-evt1-stage0-go`.

| Awkward compiler-authoring pattern | Abstraction or fix | Evidence |
| --- | --- | --- |
| AST nodes used untyped integer positions | Nominal `Id<T>` and append-only `DenseStore<T, Capacity>` | Standard AST fact traverses `Add` edges through checked IDs |
| Reused positions could alias stale handles | `GenerationalId<T>` with occupied metadata, LIFO free stack, increment or retirement | Standard stale access, double removal, reuse, and retirement facts |
| Indexed owned option replacement rejected a payload without custom Drop | General owned failure-payload drop/replace authority in `failure.go`, excluding explicit `Storage<T>` authority | Plain `Expr` artifact consumer and static authority regression |
| Borrowed enum matching emitted `.` against a C pointer | General reference-aware match lowering in `generate.go` | Native AST traversal fact |
| Zero-argument `Option::None` / `Result::Ok` constructors emitted old-style C prototypes | Emit `(void)` in `failure.go` | Artifact-only strict C11 check |
| Imported generic `GetIterator` was not resolved as a `foreach` source | Explicit `Count` and checked `IdAt` loop | Native zero-allocation ID walk; iterator sugar remains deferred |
| `Result<ref const T, E>` in `Assert.Error` emitted invalid pointer member syntax | Match the result explicitly in the fixture | Checked invalid IDs run natively; assertion lowering remains a separate issue |
| Inline owned option slots are strided payloads | Documented ordinal/ref traversal | No false `Span<T>` claim |
| Immovable `T` cannot cross by-value insert | Reject the attempted insertion | Static artifact-only negative test |

The bounded library, compiler substrate repairs, and dogfood remain in-tree.
Clang's `-Werror` gate suppresses only `-Wunused-function` for unused emitted
generic ID constructors; `-pedantic-errors` and other warnings stay active.
The next blocker is a sound stationary initialization representation that can
own immovable values while exposing a contiguous payload span. A separate
arena or graph framework would not resolve that representation problem.
