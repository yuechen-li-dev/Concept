# EVT1 compatibility and post-R5 boundary

## Canonical and compatibility spellings

| Surface | Classification | Canonical formatter target | R6 formatter policy |
|---|---|---|---|
| `const auto name = value;` | Canonical | unchanged | emit for inferred immutable bindings |
| `auto name = value;` | Canonical | unchanged | emit for inferred mutable bindings |
| `using Name = Type;` | Canonical | unchanged | emit for aliases |
| `async` | Canonical | unchanged | emit by default |
| `await` | Canonical | unchanged | emit by default |
| typed `let Type name = value;` | Compatibility | `const Type name = value;` | rewrite when alias canonicalization is enabled |
| inferred `let name = value;` | Compatibility | `const auto name = value;` | rewrite when alias canonicalization is enabled |
| `var name = value;` | Compatibility | `auto name = value;` | rewrite when alias canonicalization is enabled |
| `type Name = Type;` | Compatibility | `using Name = Type;` | rewrite when alias canonicalization is enabled |
| `asynchronous` | Compatibility | `async` | rewrite when alias canonicalization is enabled |
| `awaitchronous` | Joke-but-permanent exact alias | `awaitchronous` | do not rewrite by default |

`asynchronous` and `async` share one AST and lowering path;
`awaitchronous` and `await` do likewise. The joke creates no semantic branch.
No accepted spelling in this ledger is deprecated at the R5 freeze. Public
types, functions, methods, concepts, interfaces, and compiler-known operations
use PascalCase; locals, parameters, and fields use camelCase. This remains an
R6 lint/formatter rule rather than a grammar rejection.

The machine-readable companion is
`docs/architecture/EVT1-CANONICALIZATION.json`. The semantic corpus contains
five dedicated compatibility fixtures; `var` and `type` also have focused
inline compiler tests.

## Post-R5 deferred-feature ledger

| Feature | Owner / phase | Classification | R5 decision |
|---|---|---|---|
| `.concept_test`, test attributes, deterministic test runner, `Assert.*` | R6a tooling/library | Canonical tooling | implemented without changing frozen function/body semantics |
| richer theory providers, exact prophecy termination matching, parallel tests, lifecycle hooks, statistical benchmark framework | later R6 tooling | Deferred | R6a keeps JSON positional rows, sequential reporting, and lightweight timing |
| formatter, lint, diagnostics UX, build/package tooling, LSP | R6 tooling | Deferred | consume the frozen semantic model |
| owning `dyn` | later language/runtime | Deferred | requires explicit erased storage and allocation policy |
| owning erased callback | later language/runtime | Deferred | no hidden closure box |
| open generic runtime async methods | later language | Deferred | closed witness construction remains authoritative |
| stable public `Async` ABI | native backend/runtime | Deferred | current C11 layout is bootstrap evidence |
| in-place async witness construction | later language/backend | Deferred | no new construction path in R5 |
| fallible capture construction | later language | Deferred | capture evaluation remains exact and non-fallible |
| async callback literals | later language | Deferred | ordinary async functions and callable values compose explicitly |
| runtime global storage | runtime/library | Deferred | no hidden lifetime or initialization authority |
| general raw/MMIO surface | native backend/later language | Deferred | layout/stream do not imply volatile or device I/O |
| scheduler policy | runtime/library | Deferred | explicit `Step` is authoritative |
| cancellation, channels, select/race | runtime/library | Deferred | not language-core async semantics |
| LIR/native backend | native backend | Deferred | MIR plus strict C11 remains the oracle |
| inheritance, RTTI, per-object vtables | language core | Intentionally rejected | interface composition plus static witnesses |
| hidden allocation, owning closure boxes, saved-PC coroutine fallback | language core | Intentionally rejected | fixed inline state and explicit ownership |

R6 may fix bugs, diagnostics, contradictions, or specification errors. Any new
language feature requires an explicit post-freeze proposal.

## R6a test tooling

```concept
[[fact]]
void AdditionWorks()
{
    Assert.Equals(Add(2, 3), 5, "integer addition should remain exact");
}
```

`.concept_test` is canonical and `.con_test` is historical PoC3 evidence only.
All `Assert.*` calls require a non-empty reason string. The attributes and
assertion surface are tooling/compiler metadata and calls; they do not create
runtime reflection or reopen frozen R5 language-core semantics.
