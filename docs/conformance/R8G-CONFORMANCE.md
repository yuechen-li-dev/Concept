# R8g partial conformance record

Compiler: `concept-evt1-stage0-go`. Baseline: `7d4f9e7516f8ff9054ab4cab033ad24c4d08d685` (Claude golden commit); its first parent is `bab9503d14d93617410f02dad558c0b2c3dfbf52`, whose parent is the R8f baseline `002602fba67d41689f7d63991aa2826ef194009a`. Implementation commit: `cb82976c9547c883aaf4cac6f192a7d2f1bcbda8`; compatibility and checked-output commit: `978c0cd`.

The final full Go run after that implementation commit exposed old test sources that returned or passed `usize<byte>` layout queries where a scalar `usize` was declared. Those sources now use `Magnitude` explicitly, preserving the new quantity rule. The F4 enum representation also changed nine checked EVT1 fixtures; they were regenerated with `CONCEPT_UPDATE_CHECKED_OUTPUTS=1` and pass the byte-for-byte checked-output test. No source fixture or first draft was edited to hide a failed compiler behavior.

## Qualified changes

| Finding | Direct evidence | Limit |
| --- | --- | --- |
| B1 | `TestR8gContextualArrayLiteralLowering` checks returned, nested, enum payload, direct function argument, and closed generic aggregate literals with GCC/Clang strict C11 and a native result harness. | No broad source rewrite was done beyond the selected Claude golden. |
| B2 | `TestR8gQuantityCannotImplicitlyChangeMeaning` pins assignment, call, return, generic call, cast, scalar attachment, dimension mixing, and imported `Standard.Math` erasure rejection. Explicit `Magnitude`, `interpret`, and `as` remain valid. | Quantity-aware `Standard.Math` return types are still missing; quantity calls now reject safely. |
| F1 | `TestR8gContextualFloatLiterals` covers calls, comparison, binary arithmetic, unary minus, aggregates, arrays, tensor literals, match expression arms, returned Result arms, binary64 C literals, and half/float range errors. | The test does not establish all mathematical functions or generic operator closure. |
| F3 | `TestR8gBorrowedResultErrorRemap` exercises `? else` across `Result<ref Page, SourceError>` to `Result<ref Page, TargetError>`, both runtime branches, lazy once-only mapping, strict C11, negative diagnostics, and artifact-only import. | `MapError` was not added. |
| F4 | `TestR8gPayloadFreeEnumGeometry` checks fixed 4-byte tag/alignment, array, table column, enum-containing Span element, strict C11, and native execution; payload enum geometry still rejects. | No explicit source representation selector was added. |
| Determinism | `TestR8gGeneratedArtifactsDeterministic` compares generated C, MIR, and semantic artifact bytes over 100 runs. | This is representative, not a 100-run check of every requested output family. |

## Validation on this Windows host

| Gate | Result |
| --- | --- |
| Focused R8g tests | Passed, including GCC/Clang strict C11 and native harnesses. |
| R8c, R8d, R7p, R8f targeted Go suites | Passed. |
| Frozen semantic corpus manifest | Passed. |
| `go vet ./...` | Passed. |
| Root and legacy Zig suites | Passed. |
| Golden Normal and Verify | 125 passed, 2 benchmarks in each run. Claude-only Normal: 88 passed. |
| Standard Normal and Verify | 35 passed, 3 benchmarks in each run. |
| DragonGod Normal and Verify | 23 passed, 1 benchmark in each run. |
| `go test ./... -count=1` and R7d3 BurnIn | Final full run has only the same five baseline native-link failures: Windows Clang/lld-link cannot open `m.lib`. Both R7d3 runtime oracle tests fail the same way in BurnIn. No semantic, corpus, checked-output, or R8g test failed. |
| Checked EVT1 generated outputs | Regenerated through the repository update gate after F4; byte-for-byte `TestEVT1CheckedOutputsMatch` passes. |
| Golden lint | All 31 final `.concept` files passed per-file lint with `CONCEPT_MODULE_ROOTS=libraries`. Whole-tree lint stops before source analysis because `libraries/Golden/manifest.concept` does not exist. |
| Golden format check | Per-file check failed for 24 of 31 final `.concept` files. Whole-tree check stops at the missing Golden manifest. F9 remains open. |

The full Go baseline was run before editing and had the same five `m.lib` link failures. This record does not classify that host limitation as a language regression or as a passing full suite.
