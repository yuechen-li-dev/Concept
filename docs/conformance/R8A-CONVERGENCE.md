# R8a convergence record

1. Started from clean `82c85dc5dfb8a27c1e07164aa2d94ef41aefba63`.
   Baseline `go test ./...`, `go vet ./...`, both Zig suites, BurnIn,
   Standard Normal, and DragonGod Normal passed.
2. The native probe established that `_Float16` runs with GCC's extension
   lane but GCC rejects it under pedantic C11. Clang's Windows/MSVC target
   accepts its syntax but cannot link the half conversion helpers. The bfloat16 probe linked
   with GCC 15 but failed on Clang 22's Windows target at `__truncsfbf2`.
   Neither compiler recognized the probed float8 C spellings.
3. Added typed representation metadata and canonical aliases. Added scalar
   facts in the existing semantic-analysis registry and enabled builtin type
   subjects for `Assert.Concept`.
4. Preserved quantity brackets and improved the permanent `float<32>`
   diagnostic. Fixed contextual C literal emission so declared `double`
   values do not pass through a C `float` suffix.
5. Focused tests demonstrate identity, reflection, artifact transport,
   generated C, strict binary64 C11, extension binary16, and 100-run output
   identity. The full frozen regression suite was rerun.

## Regression results

| Gate | Result |
| --- | --- |
| `go test ./...` | Pass, including the full EVT1 semantic corpus |
| `go vet ./...` | Pass |
| Root and retired PoC3 Zig `zig build test` | Pass |
| BurnIn `go test ./internal/concept -run R7d3 -count=1` | Pass |
| Standard Normal / Verify | 35 / 35 facts pass in each mode |
| DragonGod Normal / Verify | 23 / 23 facts pass in each mode |
| R7p goldens plus R7q geometry Normal / Verify | 28 / 28 facts pass in each mode |
| R7p native C++ companion Normal / Verify | Pass |
| R8a focused tests | Pass, including 100-run byte identity |
| Full EVT1 corpus | Covered by `go test ./...`; 400 valid, 269 static-invalid, 13 runtime-negative, 5 compatibility, 4 expected-divergence manifest entries |

**Current state: meaningful progression.** The remaining blocker is portable
strict-C11 binary16 arithmetic and storage lowering. `_Float16` is retained
as a clearly classified extension; it is not a strict-C11 claim. R8b and R8c
work has not begun.
