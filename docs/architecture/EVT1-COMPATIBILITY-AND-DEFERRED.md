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
| unqualified scalar `byte` | Compatibility | `uint8` for values, `usize<byte>` for extents | retain while legacy fixtures are classified; never infer quantity meaning |

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
| `Assert.Concept`, proof diagnostics, `concept explain`, `concept-proof.v1` | R6b tooling | Canonical tooling | compile-time semantic query with no runtime lowering |
| generic reusable-library substrate | R6d-R6e approved post-freeze extension | Implemented | local generics and layout plus deterministic semantic modules, cross-module instantiation, and effect summaries |
| quantity/address/storage substrate | R6g-R6h approved post-freeze extension | Implemented | unit algebra, affine addresses, typed bind/init/destroy, multi-parameter templates, and cross-value/module fact transport implemented |
| foreign semantic contracts and external storage authority | R6j approved post-freeze extension | Implemented | ABI remains separate; bounded foreign facts use DeclaredForeign and trusted regions are tied to ordinary live owner/lease wrappers; exact source spelling remains provisional |
| reference allocator framework | R6k | Deferred | trusted external storage blocker is resolved; do not begin allocator policy in R6j |
| richer theory providers, exact prophecy termination matching, parallel tests, lifecycle hooks, statistical benchmark framework | later R6 tooling | Deferred | R6a keeps JSON positional rows, sequential reporting, and lightweight timing |
| formatter, lint, diagnostics UX, build/package tooling, LSP | R6 tooling | Deferred | consume the frozen semantic model |
| source auto-fix, IDE proof visualization, arbitrary proof queries, solver edits, LLM-ranked repairs, Planner explanations | post-R6b tooling | Deferred | consume authoritative proof graphs without moving semantic authority |
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
| partial specialization, variadic templates, template-template parameters, generic metaprogramming | later language, if ever | Deferred | R6d is ordinary monomorphized library generics only |
| stable module ABI, package manager, generalized effect algebra, `NoHeapAllocation` | later library/tooling | Deferred | R6e provides only internal `concept-module.v1` and the bounded allocation-effect lattice |
| full C++ ABI, header parser/importer, automatic contract generation, VerifiedForeign analysis, foreign callbacks, shared ownership, fresh-region guarantees | later interop | Deferred | R6j is C exports plus hand-authored bounded semantic companion declarations only |

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
