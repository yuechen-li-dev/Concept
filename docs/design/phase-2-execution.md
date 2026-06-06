# Phase 2 execution plan

Phase 2 turns the Stage 0 compiler from a surface-syntax parser into the smallest useful execution pipeline for Concept programs.

## Goal

The Phase 2 goal is a minimal executable Concept subset: compile a one-file program with a top-level `int main()`, lower it through the simplest practical path, and run the resulting native executable well enough to verify its process exit code.

The motivating first program is:

```cpp
module Main;

int main() {
    return 0;
}
```

The next motivating program adds expression evaluation:

```cpp
module Main;

int main() {
    return 1 + 2 * 3;
}
```

Success for Phase 2 means this subset works through the real Stage 0 path, not only through isolated unit tests.

## Explicit non-goals

Phase 2 deliberately avoids building the full middle and runtime architecture. In particular, Phase 2 does not include:

- HIR.
- MIR.
- A full type checker.
- Name resolution beyond minimal detection of the executable `main` entry point.
- Struct or enum runtime semantics.
- Generics.
- Modules beyond one-file Stage 0 compilation.
- A real linker abstraction.

These boundaries keep the milestone focused on end-to-end execution rather than on committing prematurely to long-term compiler architecture.

## Execution subset

The planned executable subset is intentionally small and ordered by milestone risk:

1. One source file beginning with `module Main;`.
2. A top-level `int main() { ... }` declaration with no parameters.
3. `return` statements in `main`.
4. Integer literals.
5. Parenthesized expressions.
6. Arithmetic expressions, starting with `+`, `-`, `*`, `/`, and `%` using normal precedence.
7. Bool literals, if useful for early checker and codegen seams.
8. Comparison and logical expressions, if they can be added without expanding Phase 2 into a general semantic checker.

Anything outside this subset should continue to parse as Phase 1 surface syntax where already supported, but it should not be given runtime meaning in Phase 2.

## C backend v0 shape

The first backend should be an audit/debug C backend:

- Emit straightforward C source from the supported executable subset.
- Prefer readable generated C over clever or optimized output.
- Use C as a temporary execution vehicle, not as the final representation of Concept semantics.
- Compile the generated C via the system C compiler in a later run harness milestone.
- Keep the backend narrow enough that unsupported Concept constructs fail clearly instead of being partially translated.

The generated C should make it easy to inspect Stage 0 behavior while the parser, checker seam, and run harness are still evolving.

## Run-test convention

Phase 2 run fixtures use a source comment to declare the expected process exit code:

```cpp
// run-exit-code: N
```

The planned run-test layout is:

- `tests/run/pass/` for programs expected to compile, run, and match their declared exit code.
- `tests/run/fail/` for programs expected to fail compilation, fail execution setup, or intentionally produce a mismatched outcome once negative run tests exist.
- `tests/corpus/phase2/` for milestone fixtures and snapshots that document target programs before the full run harness is required to execute them.

The initial Phase 2 corpus files are target fixtures. They are allowed to be ahead of the implementation while function bodies are still only shallow-captured by the Phase 1 parser.

## Milestone ladder

1. **M0: execution planning and scaffolding**
   - Document the Phase 2 scope and non-goals.
   - Add run-test directories and target fixtures.
   - Keep `zig build test` green.
2. **M1: statement parser slice**
   - Parse `return` statements inside `main` without generalizing to all statement forms.
   - Preserve diagnostics for unsupported body content.
3. **M2: expression parser slice**
   - Parse integer literals, parenthesized expressions, and arithmetic precedence needed by `1 + 2 * 3`.
   - Add bool/comparison/logical parsing only if required by the executable subset.
4. **M3: minimal executable validation**
   - Detect the one-file `int main()` entry point.
   - Reject missing, duplicate, parameterized, or non-`int` `main` in the executable path.
   - Avoid full name resolution and full type checking.
5. **M4: C backend v0**
   - Emit readable C for the supported `main` and return-expression subset.
   - Produce clear unsupported-feature diagnostics for declarations that cannot execute yet.
6. **M5: run harness v0**
   - Compile generated C with a system C compiler.
   - Run the executable and compare its exit code with `// run-exit-code: N`.
   - Wire the initial pass/fail run directories into `zig build test` only when stable.
7. **M6: corpus stabilization**
   - Promote Phase 2 fixtures from targets to enforced tests.
   - Add negative fixtures for unsupported executable forms.
   - Keep generated-output expectations small and reviewable.
