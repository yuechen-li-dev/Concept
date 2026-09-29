# A tour of Concept

Short, current-syntax examples, one subject per file. Every source here is
compiled to strict C11 and every test is run in Normal and Verify by
`TestTourExamplesGenerateStrictC11` and `TestTourTestsPassNormalAndVerify`,
so the tour cannot silently fall behind the compiler.

| File | Subject | Replaces PoC3 examples |
| --- | --- | --- |
| `01_values.concept` | records, structs, payload enums, `match`, ranges | phase2, phase3, phase4, phase5 (`payload_match`), phase7 |
| `02_concepts_and_templates.concept` | concepts, `requires`, templates | phase8 |
| `03_comptime.concept` | `comptime` values and functions, bounded loops, `static_assert` | phase9 |
| `04_ownership.concept` | `owned`, `move`, `Drop`, borrowing with `ref const` | phase10, phase6 |
| `05_failure.concept` | `Result`, `?`, `try`/`except`, `!`, `assert`, `discard` | phase5 (`result_try`, `must_use_discard`), phase17 |
| `06_machines.concept` | automata, `transition match`, `yield`, `transition decide` | phase13, phase18, phase19, phase5a |
| `07_interfaces.concept` | interfaces and `dyn` references | phase14 |
| `08_c_abi.concept` | `[[repr(C)]]`, layout assertions, `extern "C"` | phase15 |
| `09_arrays_spans_tables.concept` | fixed arrays, `Span`/`ReadOnlySpan`, `table<N>` | phase21, phase12 (fixed storage) |
| `modules/` | importing one module from another | phase16 |
| `testing/` | `[[fact]]`, `[[theory]]` with a JSON artifact, `Assert.FailsWith`, `Assert.Concept` | phase11 |

Run them yourself:

```console
go run ./cmd/concept check examples/tour/01_values.concept
go run ./cmd/concept emit-c examples/tour/06_machines.concept
go run ./cmd/concept test examples/tour/testing --verbose
go run ./cmd/concept test examples/tour/modules --verify
```

The original PoC3 examples are preserved unchanged under
`legacy/poc3-zig/examples/`. They use the retired dialect (`mut T&`,
`impl Interface<T>`, no `profile`) and do not compile with the current
compiler; these files are hand-written counterparts, not translations, per
the migration policy in
[POC3-FIXTURE-MIGRATION.md](../../docs/history/migration/POC3-FIXTURE-MIGRATION.md).
