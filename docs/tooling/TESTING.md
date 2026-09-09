# Concept testing

Concept tests are ordinary Concept functions in `.concept_test` files. Test
attributes are tooling metadata; there is no test DSL, test class, macro
framework, or runtime reflection registry.

```concept
profile Core;

[[fact]]
void AdditionWorks()
{
    Assert.Equals(Add(2, 3), 5, "two plus three should equal five");
}
```

Every assertion requires a non-empty string-literal reason. This applies to
`Assert.True(condition, reason)`, `False`, `Equals(actual, expected, reason)`,
`Near(actual, expected, tolerance, reason)`, `Error(result, reason)`, and
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

## Compile-time semantic assertions

`Assert.Concept<Goal>(subjects..., "reason")` is available in `.concept` and
`.concept_test`. A test executes only after every semantic assertion is proven.
`DISPROVEN` and `UNKNOWN` fail compilation with distinct proof graphs; they are
not runtime assertion failures. See `CONCEPT-PROOFS.md`.
