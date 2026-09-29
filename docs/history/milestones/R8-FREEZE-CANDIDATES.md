# R8 freeze candidates after R8f

| Finding | Classification | Resolution |
| --- | --- | --- |
| Closed generic `ReadOnlySpan<T>` had no concrete C typedef | Correctness fix | Header collection now includes closed template instances; native and strict C11 tests pin both widths. |
| Return inside a non-await branch of an async function emitted a C value-return from a void step | Correctness fix | Async CFG splits return-bearing branches; journal success/error paths run natively. |
| `[[must_use]]` helper was discovered as a test in `.concept_test` | Correctness fix | Test discovery ignores the semantic attribute. |
| Closed `SizeOf<T>()` choice retained `if (true/false)` and dead body in generated C | Correctness/codegen fix | Closed template lowering emits only the selected scoped branch. |
| Open `T` arithmetic requires named operation closure despite `compiler.Floating<T>()` | Ergonomics/library issue; future proposal | R8f uses `Multiply`/`Add` witnesses. Operator requirements could be considered after R8; no new semantics were added here. |
| Fixed tensor contraction retains runtime shape fields/guards | Future codegen proposal | Correct results and static shape rejection are proven. A later backend may erase guards when fixed extents are proven; no R8 semantic change is required. |
| A float literal in a double tensor requires an explicit cast | Documentation | R8f first draft and final source show the representation boundary. |
| R7p Golden aggregate compared fact passes with a manifest that now includes benchmarks | Test-harness correctness fix | Count facts and benchmarks separately; retain all R7p domain presence assertions. |

No discovered issue requires EVT2 or a new R8 semantic rule to qualify these
goldens. The two future opportunities remain explicit and unimplemented.
