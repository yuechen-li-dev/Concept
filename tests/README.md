# Concept tests

Phase 1 starts with the Zig test harness before Concept `.con_test` execution exists.
For now, `zig build test` runs the compiler harness unit tests through `src/compiler/tests.zig`.

Planned test roots:

- `tests/lexer/pass/` — lexer cases that should be accepted.
- `tests/lexer/fail/` — lexer cases that should be rejected.
- `tests/parser/pass/` — parser cases that should be accepted.
- `tests/parser/fail/` — parser cases that should be rejected.
- `tests/diagnostics/` — diagnostic rendering and message cases.

These directories are placeholders only in M0. They do not imply lexer or parser behavior yet.

Phase 2 adds execution-target scaffolding before run tests are enforced:

- `tests/corpus/phase2/` — executable-language target fixtures that may be ahead of implementation.
- `tests/run/pass/` — future programs expected to compile, run, and match `// run-exit-code: N`.
- `tests/run/fail/` — future negative run cases for compile, setup, or exit-code failures.
