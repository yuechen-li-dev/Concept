# Language fixtures

Concept's language behavior is defined over time by an executable fixture corpus. The corpus is intended to make each accepted or rejected language surface explicit, reviewable, and runnable through the real compiler paths.

## File roles

- `.concept` files are normal Concept source files.
- `.con_test` files are future Concept unit test source files. They are Concept source, but they are intended to be compiled and run by the future Concept test harness.
- `.conception` files are internal language-definition fixtures. They are not normal Concept source and should not be accepted by normal source-loading paths.

## Fixture naming

- Valid fixtures use `*.valid.conception`.
- Invalid fixtures use `*.invalid.conception`.

The valid or invalid classification is scoped to the fixture's declared phase. A fixture that is valid for `parse` means the parse-phase expectation should pass; it does not necessarily imply that the source is a complete executable program, type-checks, runs, or can be lowered to a backend. Likewise, a valid `run` fixture is expected to execute successfully, while an invalid fixture for another phase may still contain syntax that parses successfully before failing in that phase.

## Suggested corpus layout

```text
language/phase1-surface/valid/
language/phase1-surface/invalid/
language/phase2-execution/valid/
language/phase2-execution/invalid/
language/phase3-semantics/valid/
language/phase3-semantics/invalid/
```

## `.conception` format

A `.conception` file is a small sectioned text format. Headers appear before any section and use `# key: value` metadata lines.

Required headers:

- `# name: ...`
- `# phase: lex | parse | run | check | mir | backend-c`
- `# expect: pass | fail`

Optional `phase: check` header:

- `# check: declarations`
- `# check: hir`

The `check` header is valid only with `# phase: check`. If a check fixture omits it, the fixture runner defaults to `check: declarations`.

Sections are introduced with `=== section-name ===` on a line by itself. The initial reserved section names are:

- `source`
- `tokens` (reserved for later)
- `ast`
- `diagnostics`
- `run`
- `mir` (reserved for Phase 4 raw MIR snapshots)

Implemented fixture phases:

- `phase: parse` fixtures pass `source` to the real parser path. Passing parse fixtures compare `ast` against the stable AST debug output; failing parse fixtures compare diagnostic codes listed in `diagnostics`. Full rendered diagnostic snapshot matching is reserved for later.
- `phase: check` fixtures pass `source` through parse and an explicit semantic check mode. `check: declarations` runs semantic declaration collection, declaration/type-name checks, and HIR lowering without invoking the HIR executable checker. `check: hir` runs semantic collection / HIR lowering and then invokes the HIR executable checker for executable-subset validation. When `phase: check` omits `check`, the default is `check: declarations`. Failing check fixtures still match stable diagnostic codes from `diagnostics`; full rendered diagnostic matching remains reserved for later.
- `phase: run` fixtures pass `source` through parse, semantic collection / HIR lowering, the HIR executable checker, HIR-backed C emission, `zig cc`, and native process execution. For now run fixtures support only `expect: pass` and a `=== run ===` section containing `exit_code: N`. Stdout and stderr matching are not implemented yet and are reserved for later.

Reserved fixture phases:

- `phase: mir` is reserved for Phase 4 raw MIR snapshots. Planned passing fixtures will compare a `=== mir ===` section after parse, semantic collection / HIR lowering, HIR executable checking, HIR-to-MIR lowering, and MIR validation. Planned failing fixtures may compare MIR diagnostic codes in `=== diagnostics ===`. Phase 4 should start with raw MIR snapshots; later optimized MIR can use metadata such as `# mir: raw` or `# mir: optimized`.

Example:

```text
# name: payload enum surface
# phase: parse
# expect: pass

=== source ===
module Example;

enum Token {
    Identifier(StringView name),
    End,
};

=== ast ===
CompilationUnit
  Module Example
  Enum Token
    Variant Identifier
      Payload StringView name
    Variant End
```

## Check fixture example

```text
# name: executable type check
# phase: check
# check: hir
# expect: fail

=== source ===
module Main;

int main() {
    return false;
}

=== diagnostics ===
CON0029 error: return expression type does not match function return type
```

Diagnostic matching for check fixtures remains code-based: the fixture harness compares the leading diagnostic codes in `=== diagnostics ===` and does not yet require rendered spans or complete diagnostic text snapshots.

## Run fixture example

```text
# name: arithmetic return
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

## MIR fixture example (reserved for Phase 4)

```text
# name: while lowers to mir
# phase: mir
# expect: pass

=== source ===
module Main;

int main() {
    int x = 0;
    while (x < 7) {
        x = x + 1;
    }
    return x;
}

=== mir ===
fn main() -> int {
  ...
}
```

`phase: mir` is not implemented yet. The Phase 4 design in `docs/design/phase-4-mir.md` defines the intended MIR snapshot and validation strategy.
