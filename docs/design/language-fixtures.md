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
language/phase4-mir/valid/
language/phase4-mir/invalid/
language/phase5-sum-types/valid/
language/phase5-sum-types/invalid/
language/phase5a-judgment/valid/
language/phase5a-judgment/invalid/
language/phase6-unsafe-ownership/valid/
language/phase6-unsafe-ownership/invalid/
language/phase8-concepts-templates/valid/
language/phase8-concepts-templates/invalid/
language/phase9-compile-time/valid/
language/phase9-compile-time/invalid/
language/phase10-ownership/valid/
language/phase10-ownership/invalid/
```

## Phase 5 sum-type fixtures

Phase 5 fixtures live under `language/phase5-sum-types/` and cover the concrete enum/failure surface: enum constructors, enum-variant `match`, payload binding, `must_use`, `discard`, strict Result-shaped metadata, and `try` diagnostics. Valid run fixtures execute through the full parse -> HIR -> MIR -> MIR-backed C -> `zig cc` -> native exit-code path; invalid check fixtures use code-based diagnostic matching.

Representative Phase 5 MIR and C backend snapshots live under `tests/corpus/phase5/`. These snapshots cover enum constructor MIR, enum match/payload-binding MIR, `try` MIR lowering, enum/result C output, and `try` C output.

## Phase 5a judgment fixtures

Phase 5a judgment fixtures live under `language/phase5a-judgment/` and cover the `decide` expression. Valid check fixtures cover accepted HIR/type-checking forms such as basic decisions, duplicate variants, and negative scores. Valid run fixtures execute through the full MIR-backed executable path and cover highest-score selection, unconditional fallback, first-arm tie-breaking, duplicate variant handling, negative scores, condition-gated scores, local initializers, return expressions, and call arguments. Invalid check fixtures use code-based diagnostic matching for unknown enum targets, non-enum targets, unknown variants, payload variant candidates, non-bool conditions, non-int scores, missing unconditional arms, and empty arm lists.

Phase 5a MIR coverage is currently code-based in the MIR lowering tests rather than file-based corpus snapshots. This keeps the corpus proportional because `decide` lowers to ordinary MIR/control flow and has no dedicated MIR or backend-C primitive.


## Phase 6 unsafe/ownership fixtures

Phase 6 fixtures live under `language/phase6-unsafe-ownership/valid/` and `language/phase6-unsafe-ownership/invalid/`. They cover the closed unsafe/raw-pointer slice: unsafe block returns, normal statements inside unsafe blocks, unsafe call enforcement, raw pointer type checking, pointer locals/returns/call arguments, address-of locals and parameters, read-only dereference runtime behavior, dereference in unsafe functions, pointer copies followed by dereference, and MIR-backed C execution for supported pointer reads.

Invalid fixtures cover unsafe calls outside unsafe context, dereference outside unsafe context, dereference of non-pointer values, pointer type mismatches, and address-of temporary/call-result expressions.

## Phase 8 concepts/templates fixtures

Phase 8 fixtures live under `language/phase8-concepts-templates/` and cover the closed concepts/templates v0 surface. Valid fixtures cover template declarations, HIR generic declarations, unconstrained generic instantiation, constrained generic success, marker constraints, impl declarations, unsafe marker declarations/impls, and the comprehensive runtime pipeline that flows through parse, HIR checking, MIR lowering, MIR validation, MIR-backed C emission, `zig cc`, and native execution.

Invalid fixtures cover malformed template syntax, type parameters out of scope, generic inference conflicts, uninferred type parameters, missing concept impls, invalid concept requirements, duplicate impls, unsafe marker audit failures, invalid marker witness bodies, constrained arity mismatches, and unsupported constrained forms outside the v0 shape.

Phase 8 MIR/backend coverage is intentionally concrete-only: the pipeline run fixture and targeted compiler tests assert that template declarations, concepts, marker concepts, and type-parameter types do not leak into executable MIR or backend C, while deterministic instantiated function names and referenced static witness calls are emitted.

Roadmap status: Phase 8 is closed for concepts/templates v0. Phase 9 is closed for compile-time execution v0.

## Phase 9 compile-time execution fixtures

Phase 9 fixtures live under `language/phase9-compile-time/` and cover the closed compile-time execution v0 surface. Valid fixtures cover `comptime expr` integer arithmetic and boolean logic, `static_assert`, compile-time functions with zero/int/bool arguments, nested compile-time calls, locals, local assignment, scoped blocks, `if` / `else`, `while`, declaration-only capability annotations, and deterministic target metadata for pointer size and endian facts.

Invalid fixtures cover unsupported compile-time expressions and statements, runtime calls to compile-time-only functions, compile-time calls to runtime functions, static-assert failure and type requirements, recursion/fuel exhaustion, unsupported local/signature shapes, capability unknown/duplicate/not-granted diagnostics, runtime target metadata use, unknown target fields, and intentionally unsupported string target facts.

Phase 9 MIR/backend coverage asserts that evaluated values lower into ordinary MIR/backend constants: unresolved compile-time expressions, `static_assert`, compile-time-only functions, capability metadata, and the `target` metadata root must not leak into executable output.

## Phase 10 ownership fixtures

Phase 10 fixtures live under `language/phase10-ownership/` and cover the closed ownership/storage-state v0 surface. Valid fixtures cover explicit moves of whole local and parameter places, Copy scalar/enum/raw-pointer flow, `impl Copy<T>` for structs, Drop cleanup snapshots, branch-aware ownership paths, assignment/reinitialization after move, and `ManualInit<T>` / `manualAssumeInit(move slot)` check behavior.

Invalid fixtures cover direct use-after-move, moving non-place expressions, unsupported field/partial moves, implicit non-Copy copies, maybe-moved use after branch joins, replacement rejection for non-Copy and Drop values, maybe-state cleanup/replacement rejection, and ManualInit errors such as missing type arguments, unsafe-boundary requirements, invalid implicit conversion to `T`, and rejected implicit wrapper copies.

Some Phase 10 behavior is intentionally covered by MIR/unit tests rather than source fixtures. Partial direct-field initialization, whole-use rejection for partially initialized structs, and partial cleanup of initialized Drop fields depend on MIR storage states that the ordinary source surface cannot yet express broadly because Concept still rejects broad ordinary uninitialized locals and does not expose the full `ManualInit<T>.write` / ptr APIs. `ManualInit<T>` is the canonical Concept term; `MaybeUninit<T>` may appear only as compatibility or familiarity terminology.

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
- `mir`

Implemented fixture phases:

- `phase: parse` fixtures pass `source` to the real parser path. Passing parse fixtures compare `ast` against the stable AST debug output; failing parse fixtures compare diagnostic codes listed in `diagnostics`. Full rendered diagnostic snapshot matching is reserved for later.
- `phase: check` fixtures pass `source` through parse and an explicit semantic check mode. `check: declarations` runs semantic declaration collection, declaration/type-name checks, and HIR lowering without invoking the HIR executable checker. `check: hir` runs semantic collection / HIR lowering and then invokes the HIR executable checker for executable-subset validation. When `phase: check` omits `check`, the default is `check: declarations`. Failing check fixtures still match stable diagnostic codes from `diagnostics`; full rendered diagnostic matching remains reserved for later.
- `phase: run` fixtures pass `source` through parse, semantic collection / HIR lowering, the HIR executable checker, HIR-to-MIR lowering, MIR validation, MIR-backed C emission, `zig cc`, and native process execution. For now run fixtures support only `expect: pass` and a `=== run ===` section containing `exit_code: N`. Stdout and stderr matching are not implemented yet and are reserved for later.
- `phase: mir` fixtures pass `source` through parse, semantic collection / HIR lowering, HIR executable checking, HIR-to-MIR lowering, MIR validation, and MIR debug snapshot rendering. Passing fixtures compare the raw `=== mir ===` section exactly against the stable MIR debug output. Failing MIR fixtures are reserved for future lowering/validation diagnostics and may use `=== diagnostics ===` when implemented. Phase 4 starts with raw MIR snapshots; later optimized MIR can use metadata such as `# mir: raw` or `# mir: optimized`.

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

## MIR fixture example

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
