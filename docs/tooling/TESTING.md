# Concept testing

Test-only runtime reflection with `[[reflect]]` is not yet implemented. Type-level `[[reflect]]` is a compile-time cross-module structural permission and does not enable test runtime metadata.

Concept tests are ordinary Concept functions in `.concept_test` files. Test
attributes are tooling metadata; there is no test DSL, test class, macro
framework, or runtime reflection registry.

```concept
profile Core;

[[fact]]
void AdditionWorks()
{
    Assert.Equal(Add(2, 3), 5, "two plus three should equal five");
}
```

Every assertion requires a non-empty string-literal reason. This applies to
`Assert.True(condition, reason)`, `False`, `Equal(actual, expected, reason)`
(also `Equals`), `Near(actual, expected, tolerance, reason)`,
`Error(result, reason)`, `FailsWith(result, expectedError, reason)`, and
`LGTM(result, reason)`. Values evaluate once, left to right. `Error` expects the
ordinary `Result<T,E>::Error` channel; `LGTM` expects `Result<T,E>::Ok` and does
not unwrap it.

## Test kinds

- `[[fact]]`: zero-parameter `void` or `async void` deterministic test.
- `[[theory]]`: `void` or `async void` test whose primitive parameters bind to
  positional JSON rows from the first JSON `[[artifact]]`.
- `[[benchmark]]`: zero-parameter `void` test with one warmup and five measured
  child-process wall-time iterations by default.
- `[[prophecy]]`: zero-parameter `void` test fulfilled by abnormal termination;
  normal return fails.
- `[[foretold]]`: prophecy-only diagnostic capture. `Foretell.Checkpoint` keeps
  the last 16 non-empty literal checkpoints.
- `[[artifact("path")]]`: repeatable source-relative metadata. Paths must stay
  under the selected test root and are recorded with SHA-256 identity.
- `[[verify_foreign("ContractName")]]`: on a fact, requests a Verify-mode
  runtime observation for a typed `compiler.NonNull(result)` foreign contract.
  Unsupported contracts and unobserved call paths fail explicitly. The result
  retains `DeclaredForeign`, declaration and call sources, strategy, and
  pass/fail without changing the static proof status.

Theory JSON is deliberately bounded:

```json
[
  [2, 3, 5],
  [-4, 9, 5]
]
```

Rows bind by position to `int`, `uint`, `byte`, `float`, `bool`, or `string`
parameters. Rich providers and implicit conversion are deferred.

## CLI and results

```text
concept test
concept test path/to/tests
concept test Addition
concept test --filter Addition
concept test --list
concept test --verbose
concept test path/to/native-project --verify
```

Discovery is normalized path order followed by source declaration order.
Infrastructure directories named `testdata`, `.test-results`, `.git`, or a Zig
cache are not traversed during project-root discovery; pass an explicit file or
directory to inspect a compiler fixture corpus.
Execution is sequential and result reporting remains in that order. The
runner writes `.test-results/manifest.json` using
`concept-test-manifest.v1` and `.test-results/results.json` using
`concept-test-results.v1`. Foretold evidence is retained under
`.test-results/prophecy/<test-id>/latest/` as JSON plus exact stdout/stderr.
Any failed fact/theory, unfulfilled prophecy, timeout, compile error, or
unexpected abnormal exit makes `concept test` return nonzero.

Each run generates a checked source module once per compilation mode and
compiles its generated C and optional machine helper once. Facts and theory
rows link their own entry harnesses against those objects; identical theory
rows reuse the same executable. Every execution, including benchmark samples
and prophecies, still starts a separate child process. Normal and Verify have
separate builds. Reuse ends when the run returns, and temporary build files
are removed. There is no persistent cache or source-path cache: discovery
snapshots include the checked imported artifacts, and changed native link
inputs select a different build. Vulkan currently shares generation only;
runtime selection, kernel binding checks and native compilation remain per
test.

## Go suite setup and determinism

`go test ./...` retains the compiler, diagnostic, generated-code, artifact,
formatter and independent C/native differential oracles. Pure runtime checks
for mutation shortcuts, counted/async/machine ranges and else-if behavior live
in `tests/runtime/*.concept_test` and run in both Normal and Verify. Go output
checks read those same files rather than maintaining embedded copies.

The fixed AMD64 backend fixture is generated once per mode for runtime tests.
Its object files are shared within the Go test process using the compiler,
flags and emitted source/supporting bytes as the key. Each test keeps its own
harness, oracle and execution. Generation determinism tests still generate
independently. Unique small specimens use a single compile/link invocation
because separate object compilation would add overhead.

Small compiler/artifact determinism tests retain 100 repetitions (10 with
`-short`). The complete Standard/DragonGod integration instead performs two
independent cold builds and compares every emitted artifact as well as the
package graph. A small dependency package fixture exercises the actual package
builder 100 times. ABI integration measures the native toolchain twice and
retains 100 independent report/probe encodings. The foreign verification
provenance test still executes its failure in 100 separate processes, using
one compiled fixture.

To request the original 100 cold package builds and native ABI probes:

```powershell
$env:CONCEPT_TEST_STRESS = "1"
go test ./internal/concept -run '^(TestStandardAndDragonGodPackagesBuildDeterministically|TestNativeABIEvidenceAndEncodingDeterminism)$' -count=1
Remove-Item Env:CONCEPT_TEST_STRESS
```

This stress lane intentionally costs more than the default integration checks;
100 report encodings are not a claim of 100 independent native probes.

### Measured optimization round (2026-10-03, Windows AMD64)

Both runs used `-count=1`. The baseline compiler-package run collected a CPU
profile; the final run exercised all Go packages together. These are local
wall times, with parallel-test contention and profiling overhead, rather than
portable performance thresholds.

| Check | Before | After |
| --- | ---: | ---: |
| Compiler package | 497.55 s | 331.34 s |
| Golden facts, Normal + Verify | 75.32 s | 51.26 s |
| Foreign verification, 100 executions | 27.85 s | 5.89 s |
| Full Standard/DragonGod package integration | 101.21 s (100 builds) | 6.22 s (2 builds, all artifact bytes) |
| Native ABI integration | 24.77 s (100 probes) | 0.50 s (2 probes, 100 encodings) |

The full final `go test ./... -count=1` run took 334.50 s wall time and passed.
The small package builder's 100-build gate took 5.02 s. Golden test identities
and Normal/Verify execution coverage were preserved. CPU profiling and test
event logs are local artifacts under `artifacts/test-optimization-*`.

Further substantial savings need work on the remaining real native trace
execution and large artifact determinism paths; they have not been replaced
with cached results in this round.

## Compile-time semantic assertions

`Assert.Concept<Goal>(subjects..., "reason")` is available in `.concept` and
`.concept_test`. A test executes only after every semantic assertion is proven.
`DISPROVEN` and `UNKNOWN` fail compilation with distinct proof graphs; they are
not runtime assertion failures. See `CONCEPT-PROOFS.md`.
