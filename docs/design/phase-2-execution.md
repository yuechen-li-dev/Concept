# Phase 2 execution closeout

Phase 2 turns the Stage 0 compiler from a surface-syntax parser into a small end-to-end execution pipeline for Concept programs.

## Phase 2 status

Phase 2 proved Concept can compile and run a small executable subset through the real Stage 0 path. At Phase 2 closeout that path was AST-shaped:

```text
Concept source
  -> parseSource
  -> executable subset checker
  -> C backend v0
  -> zig cc
  -> native executable
  -> exit-code verification
```

Phase 3 preserves the same executable subset, but the implementation path has evolved: run fixtures now pass through semantic collection / HIR lowering, the HIR executable checker, and HIR-backed C emission before `zig cc` and native exit-code verification. The old AST checker/backend path is transitional legacy support, not the authoritative post-Phase-3 run path.

C backend v0 is an audit/debug backend, not the final compiler architecture. It exists to keep generated output readable and to make Stage 0 execution behavior easy to inspect while the language and compiler seams stabilize. MIR remained deferred during Phase 2 and remains a Phase 4 concern after the Phase 3 semantic-spine closeout.

## Supported executable subset

The final Phase 2 executable subset supports:

- One-file Stage 0 compilation.
- `module Main;`.
- Top-level functions.
- `int main()` as the executable entry point.
- `int` and `bool` executable categories.
- Integer literals.
- Bool literals.
- Parenthesized/grouped expressions.
- Unary `-` and `!`.
- Binary arithmetic:
  - `+`
  - `-`
  - `*`
  - `/`
  - `%`
- Comparisons:
  - `<`
  - `<=`
  - `>`
  - `>=`
- Equality:
  - `==`
  - `!=`
- Logical operators:
  - `&&`
  - `||`
- Return statements.
- Local declarations with required initializers.
- Identifier expressions.
- Assignments to simple local or parameter identifiers.
- Function calls to simple identifier callees.
- Parameters.
- `if`/`else` statements.
- Direct `else if` rejection.
- Explicit nested `if` inside `else { ... }` blocks.
- `match` statements with:
  - int literal patterns
  - bool literal patterns
  - `_` wildcard patterns
- `while` loops.
- C backend v0.
- Run harness compilation through `zig cc`.
- `.conception` run fixtures with `exit_code: N`.

The executable subset is intentionally narrower than the parser surface. Syntax that parses in the Phase 1 surface can still be unsupported in Phase 2 execution.

## Control-flow spelling

`if` is the binary branching construct. The executable subset supports a braced then block and an optional braced `else` block:

```cpp
if (condition) {
    return 1;
} else {
    return 0;
}
```

Direct `else if` ladders are rejected so that multi-way branching has one spelling in this milestone. Use `match` for multi-way branching. Explicit nesting remains allowed when the nested `if` is written inside an `else { ... }` block:

```cpp
if (a) {
    return 1;
} else {
    if (b) {
        return 2;
    }
}
```

The current statement-form `match` supports integer literal patterns, boolean literal patterns, and `_` wildcard arms, and lowers to a C `switch` in the Stage 0 backend.

## C backend v0 shape

C backend v0 emits straightforward C source from the supported executable subset:

- Generated C should be boring, readable, and stable.
- Unsupported Concept constructs should fail clearly instead of being partially translated.
- C is a temporary execution vehicle for Stage 0, not the final representation of Concept semantics.
- The run harness compiles generated C with `zig cc`, runs the host executable, and verifies its process exit code.

## Run fixture convention

Phase 2 run fixtures use `.conception` metadata and a `run` section to declare the expected process exit code:

```text
# phase: run
# expect: pass

=== source ===
module Main;

int main() {
    return 1 + 2 * 3;
}

=== run ===
exit_code: 7
```

The run harness v0 uses the real parse path, the Phase 2 executable checker, and the C backend, then writes `main.c` into a temporary directory, compiles it with `zig cc`, runs the native executable, and compares the process exit code.

The Phase 2 `.conception` fixtures live under `language/phase2-execution/`. Generated-C snapshots live under `tests/corpus/phase2/`.

## Explicit non-goals

Phase 2 deliberately does not include:

- HIR.
- MIR.
- A full type checker.
- Multi-file modules.
- Imports with semantic effect.
- Struct runtime semantics.
- Enum runtime semantics.
- Payload enum matching.
- Match exhaustiveness checking.
- Match expressions.
- Assignment expressions.
- Compound assignment.
- Field assignment.
- Index assignment.
- Dereference assignment.
- `for` loops.
- `break`.
- `continue`.
- `do while` loops.
- Global variables.
- Arrays.
- Strings as executable semantic values.
- Chars as executable semantic values.
- Floats as executable semantic values.
- Pointers at runtime.
- References at runtime.
- C headers.
- A real linker abstraction.
- Cross-compilation.
- Stdout or stderr run assertions.

These boundaries close Phase 2 around the smallest useful executable subset and leave HIR/MIR, richer semantics, and production backend architecture to later phases.

## Milestone history

1. **M0: execution planning and scaffolding** documented the Phase 2 scope and non-goals.
2. **M1: statement parser slice** parsed function bodies and return statements.
3. **M2: expression parser slice** added integer, bool, unary, binary, grouped, local, identifier, and call expressions needed by the executable subset.
4. **M3: minimal executable validation** introduced the executable subset checker.
5. **M4: C backend v0** emitted readable C for supported executable programs.
6. **M5: run harness v0** compiled generated C with `zig cc`, ran the executable, and checked exit codes.
7. **M6-M8: subset expansion and stabilization** consolidated functions, locals, assignment, `if`, `match`, `while`, diagnostics, C snapshots, and `.conception` fixtures.
8. **M9: executable subset consolidation and Phase 2 closeout** freezes this documented subset as the Phase 2 result.
