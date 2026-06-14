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
language/phase12-allocation/valid/
language/phase12-allocation/invalid/
language/phase13-machines/valid/
language/phase13-machines/invalid/
language/phase14-interfaces/valid/
language/phase14-interfaces/invalid/
language/phase15-c-abi/valid/
language/phase15-c-abi/invalid/
language/phase16-imports/valid/
language/phase16-imports/invalid/
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

Roadmap status: Phase 8 is closed for concepts/templates v0. Phase 9 is closed for compile-time execution v0. Phase 10 is closed for ownership/storage-state v0. Phase 11 is closed for first-class testing, attributes, and reasoned expectations v0. Phase 14 is closed for runtime interfaces and borrowed dyn dispatch v0: interface declaration, requirement validation, interface impl conformance, borrowed `dyn Interface&` type-surface fixtures, call-boundary concrete-to-dyn coercion fixtures, dyn method-call HIR/MIR fixtures, C backend vtable/fat-reference lowering fixtures, executable borrowed dyn dispatch fixtures, examples, mutability hardening fixtures, and backend C-shape assertions for the borrowed dyn subset. Phase 15 is closed for C ABI v0: extern C parser/AST/HIR fixtures, ABI type validation, extern call MIR/backend prototype and call-shape fixtures, export C parser/check/backend/run fixtures, staged `[Repr(C)]` marker fixtures, repr(C) field validation, ABI-boundary acceptance, backend layout/order/de-duplication fixtures, examples, and representative invalid fixtures for the pinned non-goals.

## Phase 16 import/multi-module fixtures

Phase 16 fixtures should introduce hermetic multi-file `.conception` cases. A
single fixture may contain multiple virtual Concept source files, each with a
stable virtual path for diagnostics, and the harness should pass those files to
one compilation unit. The recommended embedded form is `=== file: Name.concept
===` sections followed by the usual `=== diagnostics ===`, `=== mir ===`, or
`=== run ===` expectations.

Representative Phase 16 fixtures should cover module declaration collection,
duplicate module diagnostics, import ordering, unknown imports, duplicate
imports, import cycles, qualified cross-module calls, qualified type references,
repr(C) metadata used across modules, and generated C collision avoidance. The
fixtures should not require filesystem import search paths or sidecar files in
v0.

## Phase 9 compile-time execution fixtures

Phase 9 fixtures live under `language/phase9-compile-time/` and cover the closed compile-time execution v0 surface. Valid fixtures cover `comptime expr` integer arithmetic and boolean logic, `static_assert`, compile-time functions with zero/int/bool arguments, nested compile-time calls, locals, local assignment, scoped blocks, `if` / `else`, `while`, declaration-only capability annotations, and deterministic target metadata for pointer size and endian facts.

Invalid fixtures cover unsupported compile-time expressions and statements, runtime calls to compile-time-only functions, compile-time calls to runtime functions, static-assert failure and type requirements, recursion/fuel exhaustion, unsupported local/signature shapes, capability unknown/duplicate/not-granted diagnostics, runtime target metadata use, unknown target fields, and intentionally unsupported string target facts.

Phase 9 MIR/backend coverage asserts that evaluated values lower into ordinary MIR/backend constants: unresolved compile-time expressions, `static_assert`, compile-time-only functions, capability metadata, and the `target` metadata root must not leak into executable output.

## Phase 10 ownership fixtures

Phase 10 fixtures live under `language/phase10-ownership/` and cover the closed ownership/storage-state v0 surface. Valid fixtures cover explicit moves of whole local and parameter places, Copy scalar/enum/raw-pointer flow, `impl Copy<T>` for structs, Drop cleanup snapshots, branch-aware ownership paths, assignment/reinitialization after move, and `ManualInit<T>` / `manualAssumeInit(move slot)` check behavior.

Invalid fixtures cover direct use-after-move, moving non-place expressions, unsupported field/partial moves, implicit non-Copy copies, maybe-moved use after branch joins, replacement rejection for non-Copy and Drop values, maybe-state cleanup/replacement rejection, and ManualInit errors such as missing type arguments, unsafe-boundary requirements, invalid implicit conversion to `T`, and rejected implicit wrapper copies.

Some Phase 10 behavior is intentionally covered by MIR/unit tests rather than source fixtures. Partial direct-field initialization, whole-use rejection for partially initialized structs, and partial cleanup of initialized Drop fields depend on MIR storage states that the ordinary source surface cannot yet express broadly because Concept still rejects broad ordinary uninitialized locals and does not expose the full `ManualInit<T>.write` / ptr APIs. `ManualInit<T>` is the canonical Concept term; `MaybeUninit<T>` may appear only as compatibility or familiarity terminology.

## Phase 11 testing fixtures

Phase 11 fixtures live under `language/phase11-testing/` and cover the closed first-class testing v0 surface. Parse fixtures cover declaration attributes, `[Fact]`, `[Theory]`, `[InlineData(...)]`, supported literal attribute arguments, and rejected malformed attribute placement or expressions.

Valid `.con_test` check fixtures cover `.con_test` source-kind classification, helper functions that are not discovered as tests, `[Fact]` signature validation, `[Theory]` plus `[InlineData]` validation, primitive `Assert.*` / `Expect.*` reason validation, HIR runner-supported facts and theories, the primitive `Expect.That` relation scaffold (`Is.True()`, `Is.False()`, `Is.EqualTo(int)`, and `Is.EqualTo(bool)`), reporting/output stabilization through runner unit tests, and a closeout smoke fixture combining one fact, one two-row theory, `Assert.True`, `Expect.Equal`, and `Expect.That(..., Is.EqualTo(...))`.

Invalid fixtures are compile-time parse or semantic invalid cases: test attributes in normal source, invalid test signatures, missing or mismatched InlineData, missing or empty reasons, wrong intrinsic arity, wrong primitive operand types, unsupported equality/relation types, standalone relation constructors, and rejected inline test blocks.

Runtime-failing test examples are not compile-invalid fixtures. A `.con_test` file that type-checks but fails an expectation is a runner failure, not a compiler diagnostic. Current failing runner behavior, including summaries, ordered failures, row-indexed theory names, source spans, reasons, and expected/actual scalar display, is covered by unit tests in `src/compiler/test_runner.zig` rather than by fixture invalid cases.

## Phase 12 allocation fixtures

Phase 12 fixtures live under `language/phase12-allocation/` and currently cover
the P12-M1 allocation-effect syntax and metadata scaffold, P12-M2 direct
call-edge checking, P12-M3 compiler-known allocation surface types, P12-M4
arena allocation, P12-M5 arena reset/destroy operations, P12-M6 arena
Drop/storage-state hardening, P12-M7 backend helper ABI coverage, and P12-M8
ID-based store doctrine examples. Valid parse
fixtures cover `alloc`, `noalloc`,
omitted/default effects, and coexistence with `unsafe`, `comptime`, and
template functions. Valid check fixtures cover `noalloc` callers calling known
`noalloc` callees, recursive `noalloc` calls, generic `noalloc` calls,
comptime `noalloc` calls, unsafe/noalloc coexistence, and permissive `alloc` or
unspecified callers. P12-M3 valid fixtures cover `Arena*` and `Allocator*`
parameter positions, `AllocError` type positions, allocation surface types in
`noalloc` functions, `alloc` functions, and generic functions.
P12-M4/P12-M6 valid fixtures cover `Arena.alloc<T>(arena)` for `int`, `bool`,
non-Drop structs, unspecified-effect functions, and generic functions after
instantiation.
P12-M5 valid fixtures cover `Arena.reset(arena)` and `Arena.destroy(arena)` in
`noalloc`, `alloc`, and unspecified functions, including after arena allocation.
P12-M7 backend-c fixtures cover the explicit arena helper output paths for
allocation, reset, destroy, struct size/alignment, and multiple operations in a
single generated C file.
P12-M8 valid fixtures cover nominal typed ID structs, noalloc ID helper
functions, a deliberately tiny single-entry `ExprStore` add/get pattern, and ID
value copying through the current `Copy<T>` marker model.
P12-M9 adds a closeout smoke fixture that combines an `alloc` function, a
`noalloc` helper, an `Arena*` parameter, `Arena.alloc<int>(arena)`,
`Arena.reset(arena)`, `Arena.destroy(arena)`, and the typed-ID/store handle
pattern in one HIR-check path.

Invalid fixtures cover conflicting `alloc`/`noalloc` specifiers, duplicate
effect specifiers, the reachable non-function target diagnostic, and `noalloc`
callers directly calling `alloc` or unspecified callees, including generic and
comptime cases. P12-M3 invalid fixtures cover rejected `Arena`/`Allocator`
struct literals, field access on opaque allocation handles, and by-value
opaque-handle signatures.
P12-M4/P12-M6 invalid fixtures cover arena allocation inside `noalloc`, missing
type arguments, wrong value arity, non-`Arena*` operands, direct Drop allocated
types, generic Drop allocation after instantiation, structs containing Drop
fields, `ManualInit<DropType>` allocation, opaque allocation handles allocated
by value, and compile-time allocation rejection.
P12-M5 invalid fixtures cover reset/destroy arity, rejected type arguments,
non-`Arena*` operands, and unsupported compile-time reset/destroy.
P12-M8 invalid fixtures cover wrong typed-ID use through the existing nominal
type mismatch diagnostics, including passing a `TypeId` where an `ExprId` is
required.

Phase 12 fixtures intentionally do not execute arena allocation at runtime. They
do not cover allocator runtime behavior, allocation failure paths, transitive
effect checking, profile defaults, region checking, generic ID stores,
arena-backed stores, reset/destroy runtime execution, or runtime-backed
allocation because those remain future work. Exact helper
declaration and call text is pinned by
backend unit tests; backend-c fixtures currently assert the real lowering path
accepts the source because fixture C-output snapshot matching is not implemented
yet.

## Phase 13 machine fixtures

Phase 13 fixtures live under `language/phase13-machines/` and cover the closed
Phase 13 explicit machine/state syntax, machine-local state validation,
literal/match/decide transition target preservation, and the P13-M8 executable
literal-transition runtime subset.
Valid parse fixtures cover machine declarations, parameters, effects,
attributes, ordinary state-body statements, literal transitions, match-driven
transition targets, and decide-driven transition targets. Invalid fixtures cover
malformed machine/state/transition syntax plus semantic validation failures for
zero states, duplicate states, unknown transition targets, and cross-machine
state targets.

Valid run fixtures cover explicit frame construction through `MachineName(...)`,
`Step(machine);` as a statement-like no-value advance, one-step and multi-step
literal transition behavior, completion via `return`, `Complete(machine)` as a
`bool` query before and after completion, `Result(machine)` after completion,
scalar `int` and `bool` results, captured scalar parameters, and extra
`Step(machine)` calls after completion as no-ops. Invalid HIR-check fixtures pin
that `Step(machine)` cannot initialize locals, be returned as a value, be used
as a condition, be passed as a call argument, participate in binary expressions,
or be discarded with `discard`. An invalid run fixture pins the P13-M8
hardening that `Result(machine)` before completion traps instead of silently
reading raw result storage.

Backend-C fixtures and unit assertions pin the generated machine shape:
state enum, frame struct, current-state field, completion flag, result storage,
scalar captured parameter fields, constructor initialization, step dispatch,
literal transition assignment, return-result storage, completed-step no-op,
and absence of `malloc`, scheduler helpers, and async runtime helpers.
A closeout smoke fixture combines scalar bool parameter capture, multiple
states, conditional literal transitions, `Step`, `Complete`, `Result`, and an
extra completed `Step` no-op in one run path.

Match and decide transition runtime lowering remains deferred: those forms are
still parsed, validated, and preserved in HIR, but backend execution attempts
fail clearly with `CON0231 MachineSemanticsNotImplemented`. Phase 13 fixtures
continue to exclude DragonGod features, `board`, stack HFSM, blackboards,
mailbox buses, actuators, persistence, hysteresis, `min_commit`, policy memory,
hidden heap behavior, scheduler behavior, and async behavior from Concept core
v0.

Phase 13 closed: explicit machines, machine-local states, literal/match/decide
transition scaffolds, runnable literal-transition machine frames,
statement-like `Step`, `Complete`/`Result` value queries, and C backend support
for the literal-transition subset.

## Phase 14 interface/dyn fixtures

Phase 14 fixtures live under `language/phase14-interfaces/` and cover the
closed P14 v0 surface: interface declaration HIR behavior, requirement
validation, interface impl conformance, the borrowed dyn interface type
surface, call-boundary concrete-to-dyn coercion, dyn interface method-call
HIR/MIR scaffolding, and C backend lowering for the borrowed dyn dispatch
subset. Valid
fixtures cover parser preservation for
`interface Writer { void Write(int value); }`, multiple requirements,
requirement source order, void/scalar/pointer/struct/enum requirement
signatures, requirement parameter names/types, and backend-C non-emission for
pure interface declarations and interface impl declarations. They also cover
basic interface impls, multiple requirements, void/scalar signatures, struct,
enum, and builtin targets, body checking for impl methods, declaration-only
`dyn Interface&` parameters, `mut dyn Interface&` parameters, multiple dyn
interface parameters in one signature, concrete locals passed to dyn
parameters, mutable dyn parameters, multiple coerced call arguments, exact
dyn-to-dyn forwarding, and concept/interface separation when both impl kinds
exist. They also cover void-return dyn calls, value-return dyn calls, multiple
dyn-call arguments, and dyn calls inside a callee reached through concrete-to-dyn
coercion. M7/M8 run fixtures execute int-return dispatch, bool-return dispatch,
void-call dispatch, multiple-argument dispatch, two concrete impls for one
interface, two interfaces for one concrete type, call-boundary coercion, and
exact dyn passthrough through the MIR-backed C path. M8 also pins that mutable
dyn receiver calls remain accepted while immutable dyn receiver calls remain
rejected. Backend-C fixtures assert
vtable struct emission, dyn fat-reference struct emission, wrapper/thunk
emission, per-impl static vtable constants, explicit data/vtable fat-ref
construction, indirect `.vtable->Slot` calls, wrapper `void* self` casts to
concrete pointers, current hidden impl receiver ABI calls with `*typed`,
requirement source-order slots, vtable constant reuse, and absence of hidden
allocation, RTTI, dynamic-cast, reflection, scheduler, async, and
inheritance-like artifacts. Invalid fixtures
cover duplicate top-level interface names, empty interfaces, unknown
requirement return and parameter types, duplicate and overloaded requirement
names, duplicate requirement parameter names, interface types in requirement
return/parameter positions, ordinary runtime interface values, and invalid
interface impls for missing, extra, duplicate, wrong receiver, wrong return,
wrong parameter, duplicate impl, unknown interface, invalid target, and
concept/interface separation cases. They also cover rejected dyn targets for
structs, enums, concepts, builtins, and unknown names; rejected by-value and raw
pointer dyn spellings; rejected dyn locals, returns, and struct fields;
missing or wrong interface impls for dyn coercion; concept impls that do not
count as interface impls; non-place coercion sources such as temporaries, call
results, binary expressions, and `move`; unknown dyn methods, wrong dyn-call
arity, wrong dyn-call argument types, immutable dyn call receivers, non-dyn
method-call receivers, and void dyn calls used as values.

Phase 14 examples live under `examples/phase14/` and document the supported
borrowed dyn subset: no hidden heap allocation, no owning dyn boxes, no RTTI,
no dynamic casts, no inheritance, explicit `{ data, vtable }` lowering, and the
current `mut dyn Interface&` method-call rule. Mutation observed through dyn
dispatch remains deferred because receiver references are not yet first-class
TypeStore values.

Phase 14 closed: runtime interfaces and borrowed dyn dispatch v0. Interfaces
are HIR declarations with validated requirements and impl conformance. Borrowed
dyn references support concrete-to-dyn call-boundary coercion, dyn method
calls, MIR-visible dyn coercion/interface calls, and C backend vtable lowering
for the supported subset.

Phase 14 fixtures intentionally do not cover owning dyn boxes, heap boxing,
dynamic cast, RTTI, reflection, inheritance, interface upcasting, default
methods, associated types, generic interface methods, Drop through dyn, or
cross-module vtable ABI stability. Those remain deferred follow-on work beyond
the closed Phase 14 v0 subset.

## Phase 15 C ABI fixtures

Phase 15 fixtures live under `language/phase15-c-abi/`. P15-M1 parse fixtures
cover the extern C parser/AST scaffold: basic block-form `extern "C"`
declarations, multiple foreign functions preserving source order, pointer
parameters, and empty blocks.

P15-M2 check fixtures cover HIR extern declarations, multiple extern functions,
empty extern blocks, supported pointer parameters, `bool` ABI use,
`Arena*`/`Allocator*` opaque-handle parameters, and semantic resolution of
ordinary calls to extern declarations. P15-M3 backend and run fixtures cover
extern C call lowering through MIR, plain C prototype emission for declarations,
declared-symbol call emission, multiple prototypes, bool ABI spelling as C
`int`, pointer parameter prototypes/calls, empty extern blocks, and an `abs`
runtime smoke fixture that exits with code 7.

Invalid P15-M1 parse fixtures cover unsupported ABI strings such as `"C++"`,
missing ABI strings, missing blocks, extern function bodies, missing
semicolons, non-function entries, extern variables, and varargs.

Invalid P15-M2 check fixtures cover duplicate extern symbols in one block and
across blocks, extern-vs-ordinary duplicate top-level names, unsupported struct
returns, struct parameters, struct pointer parameters, interface and dyn
parameters, `ManualInit<T>` parameters, and `void` parameters. P15-M3 invalid
fixtures cover extern call arity/type mismatches and using a `void` extern call
as a value. P15-M5 fixtures cover staged `[Repr(C)]` struct markers in AST and
HIR debug output, ordinary backend emission without C ABI boundary use, invalid
Repr targets, missing/unsupported/extra Repr arguments, duplicate markers, and
the deliberate continued rejection of non-repr structs at C ABI boundaries, and the P15-M6 acceptance of validated repr(C) structs. P15-M7 adds hardening fixtures for duplicate C ABI symbols, extern prototype de-duplication and deterministic order, repr(C) typedef-before-prototype/definition order, exported-vs-internal backend names, void C ABI rules, and bool/AllocError C spelling. P15-M8 adds representative example fixtures for the final Phase 15 surface: executable extern/export examples, repr(C) by-value and pointer prototypes, bool/AllocError signatures, no-header backend output, exported-vs-helper naming, typedef-before-use ordering, and invalid fixtures for the pinned non-goals. P15-M9 closes Phase 15 without adding fixtures because the existing 108 Phase 15 fixtures already cover the final supported surface and representative non-goals.

The fixture corpus now contains 833 `.conception` files total, including 108 Phase 15 C ABI fixtures. Phase 15 is closed, and examples live under `examples/phase15/` mirroring the representative fixture surface.

Valid coverage includes `export "C"` functions, validated `[Repr(C)]` structs,
unmangled exported C names, deterministic supported struct field order, extern prototype order/de-duplication, repr(C) typedef order/de-duplication, void C exports, and current bool/AllocError C ABI spelling.

Invalid coverage includes generic exports, duplicate C ABI symbols, `repr(C)` on non-struct
declarations, empty `repr(C)` structs, unsupported/nested-by-value `repr(C)` fields, non-repr struct values/pointers at C ABI boundaries, void parameters/void-as-value calls, machine
frame ABI exposure, and payload enum ABI exposure.

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
- `c`

Implemented fixture phases:

- `phase: parse` fixtures pass `source` to the real parser path. Passing parse fixtures compare `ast` against the stable AST debug output; failing parse fixtures compare diagnostic codes listed in `diagnostics`. Full rendered diagnostic snapshot matching is reserved for later.
- `phase: check` fixtures pass `source` through parse and an explicit semantic check mode. `check: declarations` runs semantic declaration collection, declaration/type-name checks, and HIR lowering without invoking the HIR executable checker. `check: hir` runs semantic collection / HIR lowering and then invokes the HIR executable checker for executable-subset validation. When `phase: check` omits `check`, the default is `check: declarations`. Failing check fixtures still match stable diagnostic codes from `diagnostics`; full rendered diagnostic matching remains reserved for later.
- `phase: run` fixtures pass `source` through parse, semantic collection / HIR lowering, the HIR executable checker, HIR-to-MIR lowering, MIR validation, MIR-backed C emission, `zig cc`, and native process execution. For now run fixtures support only `expect: pass` and a `=== run ===` section containing `exit_code: N`. Stdout and stderr matching are not implemented yet and are reserved for later.
- `phase: mir` fixtures pass `source` through parse, semantic collection / HIR lowering, HIR executable checking, HIR-to-MIR lowering, MIR validation, and MIR debug snapshot rendering. Passing fixtures compare the raw `=== mir ===` section exactly against the stable MIR debug output. Failing MIR fixtures are reserved for future lowering/validation diagnostics and may use `=== diagnostics ===` when implemented. Phase 4 starts with raw MIR snapshots; later optimized MIR can use metadata such as `# mir: raw` or `# mir: optimized`.
- `phase: backend-c` fixtures pass `source` through parse, semantic collection / HIR lowering, HIR executable checking, HIR-to-MIR lowering, MIR validation, and MIR-backed C emission. Passing fixtures may include a `=== c ===` section with line-based `contains:` and `not_contains:` substring assertions. Failing backend-C fixtures match diagnostic codes from `=== diagnostics ===`.

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
