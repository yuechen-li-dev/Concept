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
language/phase17-runtime-failure/valid/
language/phase17-runtime-failure/invalid/
language/phase18-machines/valid/
language/phase18-machines/invalid/
language/phase19-yielding-machines/valid/
language/phase19-yielding-machines/invalid/
language/phase20-dragongod-kernel/valid/
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

## Phase 17 runtime failure fixtures

Phase 17 is closed. Its fixtures live under `language/phase17-runtime-failure/`. P17-M5
covers statement-position `assert(condition, "reason");` through parser, HIR,
MIR, backend C, and runtime execution. Assert fixtures now pin HIR-check
acceptance for basic assertions, reason preservation, bool condition expressions,
statements after ordinary code, invalid missing/rejected reason forms, shared empty/whitespace runtime reason validation,
expression-position rejection, backend lowering to `if (!(condition)) { cpt_panic(...) }`,
shared one-per-C-unit `cpt_panic` helper emission, escaped reason literals, no
test-runner dependency, assert-true exit 0, assert-false exit 101, and condition
expression runtime success. P17-M2 panic fixtures continue to cover
statement-position `panic("reason");`, invalid reason/expression forms,
backend-owned `cpt_panic` lowering, helper de-duplication, escaping, no
test-runner dependency, and exit code 101. P17-M5 adds empty, whitespace-only, and tab-only reason rejection fixtures plus non-empty reason acceptance fixtures for both panic and assert. P17-M6 adds Core.Test `Assert.True` / `Assert.False` alignment fixtures for valid reasons, empty and whitespace-only reason rejection, bool-only conditions, and runtime/test assertion separation in a `.con_test` body. P17-M7 adds runtime trap consolidation fixtures proving machine `Result(machine)` before completion exits through shared `cpt_panic` with code 101, successful result-after-completion behavior remains unchanged, and migrated traps share one helper with runtime assert. P17-M8 adds human-readable examples under `examples/phase17/` plus representative example-named fixtures for panic/assert/machine runtime exit behavior, backend helper sharing/de-duplication, reason validation, expression-position rejection, bool-only assert conditions, and runtime assert separation from Core.Test/test-runner symbols. The language fixture corpus contains
1060 fixture files after adding the P18-M1 machine audit fixtures, including 57 under `language/phase17-runtime-failure/`, 25 under `language/phase18-machines/`, and 85 files under `language/phase11-testing/`.

## Phase 16 import/multi-module fixtures

Phase 16 is closed. Its fixtures live under `language/phase16-imports/` and
cover the v0 module/import surface for harness-supplied multi-source
compilation units. The full corpus now contains 1060 `.conception`/`.con_test` fixtures, including 974 `.conception` fixtures, 72 `.con_test` fixtures, 73 Phase 16 fixtures, 57 Phase 17 fixtures, 25 Phase 18 machine fixtures, and 84 Phase 11 testing fixtures across 85 files under `language/phase11-testing/`.

A single `.conception` fixture may contain multiple virtual Concept source
files, each with a stable virtual path for diagnostics. The embedded form is
`=== file: Name.concept ===` sections followed by the usual fixture assertion
sections. Legacy `=== source ===` fixtures still parse as exactly one internal
source. Duplicate or empty virtual file paths are fixture-format errors.

The Phase 16 corpus covers multi-file parser fixtures, module declaration table
collection, duplicate and missing module diagnostics, import declarations and
ordering, raw import preservation, resolved import graph edges, unknown import
diagnostics (`CON0271`), duplicate import diagnostics (`CON0277`), import cycle
diagnostics (`CON0272`), HIR module records, per-item module ownership,
per-module duplicate top-level checking, compilation-unit-wide C ABI duplicate
symbol checking, qualified module function calls, current-module qualification,
qualified imported struct and enum type references, imported repr(C) metadata at
C ABI boundaries, multi-source MIR/backend/run lowering, generated-C name
collision hardening, and examples under `examples/phase16/`.

Representative run fixtures include cross-module function calls, same ordinary
function and struct names selected by qualification, imported struct values,
imported `extern "C"` calls, and the closeout surface fixture combining a
qualified function call, imported struct type, and imported extern C call.

Phase 16 fixtures intentionally do not require filesystem import search paths or
sidecar files. Aliases, wildcard imports, re-exports, unqualified imported
names, visibility, packages, filesystem lookup, import path-to-file mapping,
module spanning multiple files, multiple modules per file, separate object
files, linker driving, incremental compilation, and cross-package dependencies
remain deferred.

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


## Phase 18 machine fixtures

Phase 18 implementation fixtures now live under `language/phase18-machines/`. P18-M1 adds 11 machine frame/value audit fixtures: local machine construction, `Step`/`Complete`/`Result` runtime stability, independent local instances, current by-value copy/assignment behavior, result-before-completion exit code 101, backend frame shape, shared `cpt_panic` backend routing, and invalid `Step`/`Complete`/`Result` operand diagnostics.

Nested machine fields, child initialization, nested operations, executable bool `transition match`, executable deterministic `transition decide`, numeric `State(machine)`, runtime failure hardening, and P18-M8 integration examples are covered by Phase 18 milestones through P18-M8.

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

## Phase 18 M2 fixtures

Phase 18 M2 adds nested machine field fixtures under `language/phase18-machines/`. Valid fixtures cover backend child-frame storage, zero-parameter child constructor initialization, multiple child fields, absence of heap/scheduler/async markers, and a parent machine that runs to completion while holding an unused child frame. Invalid fixtures cover parameterized child fields (`CON0290`) and duplicate child field names (`CON0022`).

## Phase 18 machine fixtures

Phase 18 machine fixtures live under `language/phase18-machines/`. P18-M3 expands the corpus to 25 fixtures and pins nested `Step(child)`, `Complete(child)`, and `Result(child)` over child machine fields, including explicit stepping only, multiple child fields, child result-before-completion panic via shared `cpt_panic` exit code 101, backend address-of-child-field step emission, completion/result field reads, and unknown child-field diagnostics. Runtime `transition match` is covered by P18-M4 and runtime `transition decide` is covered by P18-M5.

### Phase 18 M4 runtime transition match fixtures

P18-M4 adds runtime, backend, and invalid fixtures for executable bool `transition match` under `language/phase18-machines/`. The valid fixtures cover true/false parameter matches, backend C `if`/`else` state assignment, and composition with nested-machine `Complete(child)`. Invalid fixtures cover unsupported non-bool scrutinees, missing true/false coverage, duplicate bool labels, case type mismatch, unknown target state, and empty matches. P18-M5 adds the runtime `transition decide` fixture set described below.

### Phase 18 M5 runtime transition decide fixtures

P18-M5 adds runtime, backend, and invalid fixtures for executable `transition decide` under `language/phase18-machines/`. The valid fixtures cover highest-score selection, false guards being ignored, source-order tie-breaking for equal scores, unguarded candidates, no-enabled shared panic with exit code 101, backend C lowering with strict `>` comparisons and the stable panic reason, and nested-machine composition using `Complete(child)` as a guard and `Result(child)` as the winning branch result. Invalid fixtures cover empty decides (`CON0299`), non-bool guards (`CON0297`), non-int scores (`CON0298`), and unknown target states via the existing machine-state diagnostic. The v0 subset remains deterministic and does not add yield, scheduler, async, event bus, blackboard/mailbox, heap-owned machines, random/weighted selection, behavior-tree runtime, GOAP/planner runtime, or DragonGod hooks.

### Phase 18 P18-M6 machine state introspection fixtures

P18-M6 adds fixtures for `State(machine) -> int` under `language/phase18-machines/`. The valid fixtures cover local machine state reads before stepping, state reads after an explicit `Step`, nested child-field state reads inside a parent machine, and state reads before completion without using the shared panic path. The backend fixture pins direct `.state` field reads for both local and nested child frames and asserts that the helper does not introduce completion checks, result guards, panic-only state behavior, heap allocation, scheduler, async, or blackboard markers. Invalid fixtures cover `CON029D StateRequiresMachineValue` for non-machine operands plus existing invalid-call arity diagnostics. Later P18-M7/P18-M8 fixtures raise the Phase 18 machine corpus to 66 fixtures.

### Phase 18 M7 runtime failure hardening fixtures

P18-M7 raised the Phase 18 machine fixture corpus to 60 files before P18-M8 expanded it to 66 files. The new backend fixtures pin stable shared-panic lowering for local `Result(machine)` before completion, nested `Result(child)` before completion, `transition decide` with no enabled candidates, a generated C unit with multiple panic sites sharing one `cpt_panic` helper, and the defensive invalid-machine-state step-dispatch default. The runtime fixtures for local result, nested child result, and decide no-enabled continue to pin exit code 101. Bool v0 `transition match` does not gain a runtime no-case fixture because exhaustive validation prevents ordinary no-match programs from reaching backend execution.

## Phase 18 machine composition fixtures

Phase 18 machine fixtures live under `language/phase18-machines/`. P18-M9 closes
the corpus at 66 fixtures: 43 valid fixtures and 23 invalid fixtures. The final Phase 18 categories cover frame/value hardening, nested machine fields, nested operations, runtime match, runtime decide, `State(machine)`, runtime failure hardening, and integration examples/backend shape. The added
integration fixtures cover hierarchical child machines, match-driven child
completion, deterministic utility selection, nested utility controllers, numeric
state observation, and combined backend shape assertions.

The backend integration fixture intentionally asserts the absence of hidden
runtime machinery: no `malloc`, scheduler, async marker, blackboard, mailbox,
event bus, or DragonGod hook appears in generated C. This pins Phase 18 as a
pre-DragonGod substrate rather than a DragonGod runtime implementation. The full language fixture corpus now contains 1052 `.conception` files.

## Phase 19 yielding machine fixtures

Phase 19 fixtures live under `language/phase19-yielding-machines/`. P19-M9 closes the corpus at 23 fixtures: 16 valid fixtures and 7 invalid fixtures. By category, the final Phase 19 set has 10 validation fixtures, 9 runtime fixtures, and 4 backend C fixtures; the validation category consists of 3 accepted check/HIR fixtures plus 7 rejected parse/diagnostic fixtures. The full language fixture corpus now contains 1052 `.conception` files.

The Phase 19 run fixtures cover `yield + Complete`, `yield + Result`, `yield + State`, branch-local yield, nested child ticking, child `State` readability after parent yield, repeated external `Step` polling, `transition match`, and `transition decide`. The backend fixtures prove direct `return;` lowering; absence of yield-specific panic/helper/state/completion/result/allocation/scheduler/async/event/mailbox/blackboard/DragonGod/suspend/resume markers; and `Result(machine)` after yield using the existing `machine result cannot be read before completion` shared panic reason. Validation fixtures cover accepted bare `yield;` in machine state bodies, nested blocks, and `if` branches, plus rejections for yield outside machine states, top-level yield, function-block yield, yield values, `yield return`, expression-position yield, and missing semicolon syntax.

Phase 19 also adds documentation examples under `examples/phase19/` for wait-until child completion, caller-controlled polling, long-running tickable behavior, yield before `transition match`, yield before `transition decide`, `State(machine)` observation after yield, runtime failure notes, and DragonGod readiness. Examples are separate from fixture counts. Unsupported yield forms remain diagnostics or stable parse errors, compile-time/test-runner execution remains unsupported for unexpected yield HIR, unreachable-after-yield diagnostics remain deferred, and Phase 19 is closed.


## Phase 20 DragonGod Kernel fixtures

Phase 20 fixtures live under `language/phase20-dragongod-kernel/valid/` and pin the DG1 module/type seed. The five valid run fixtures cover foundational ID/time/reason structs, payload `AutomataSignal` construction and matching, the minimal Memory shell, Mind/Agent/World shell structs, and qualified multi-module imports for the `DragonGod.Kernel` root. They intentionally do not test a DragonGod runtime: Memory storage, Mind tick loops, Automata stacks, schedulers, async, blackboard/mailbox/event-bus behavior, and compiler hooks remain outside DG1.

## Phase 20 / DG2 DragonGod fixtures

Phase 20 DragonGod coverage now contains 16 valid fixtures. DG2 added signal-returning machine run fixtures for `Succeed`, `Fail`, `Goto`, yield-then-succeed, transition-match true/false, transition-decide high/low, `Act`, and `AwaitActuation`, plus a backend C fixture for the static automata signal machine shape.

These fixtures intentionally remain static and specialized. They do not introduce Mind ticking, Automata stacks, dynamic graph storage, Memory storage, Actuation implementation, Events, Trace, Replay, Persistence, parallel staged ticking, scheduler/async behavior, or DragonGod compiler hooks.


## Phase 20 / DG3 DragonGod Memory fixtures

Phase 20 DragonGod coverage now contains 23 valid fixtures. DG3 adds seven fixtures under `language/phase20-dragongod-kernel/valid/`: six run fixtures for Memory write/read, fallback read, `memoryHas`, revision behavior, update-existing-key behavior, and Memory-driven `AutomataSignal` machines; plus one backend C fixture for the fixed-slot Memory shape.

The backend fixture pins the presence of the `Memory` struct with `revision` and explicit slots, direct mutation through `memoryWrite`, fallback return behavior in `memoryRead`, and absence of heap allocation, scheduler/async/mailbox markers, and Blackboard/BbKey public naming. The fixtures intentionally do not cover generic `MemoryKey<T>`, hash maps, TTL, changed-key filtering, type erasure, persistence, replay, Mind ticking, or DragonGod compiler hooks.

## Phase 20 / DG4 DragonGod Mind fixtures

Phase 20 DragonGod coverage now contains 30 valid fixtures out of 1082 `.conception` files. DG4 adds seven fixtures under `language/phase20-dragongod-kernel/valid/`: six run fixtures for `mindRecordSignal` with `Succeed`, `Fail`, deferred `Goto`, static succeed-machine supervision, yielding static-machine supervision, and Memory-driven signal-machine supervision; plus one backend C fixture for the Mind v0 shape.

The backend fixture pins `Mind` with typed status/signal fields, `mindRecordSignal`, `Succeed`/`Fail` handling, and absence of heap allocation, dynamic graph/stack storage, scheduler markers, and DragonGod runtime hooks. The DG4 fixtures intentionally do not cover dynamic Automata graphs, stack frame semantics, `Goto` execution, transition scanning, interrupts, decision memory, type-erased machine storage, Actuation dispatch, Events, Trace, Replay, Persistence, or parallel staged ticking.

## Phase 20 / DG5 DragonGod Automata stack fixtures

Phase 20 DragonGod coverage now contains 37 valid fixtures out of 1089 `.conception` files. DG5 adds seven fixtures under `language/phase20-dragongod-kernel/valid/`: six run fixtures for Automata stack push/pop, replace-top, Mind `Goto` pushing an empty stack, Mind `Goto` replacing the current top frame, Mind `Push`, and Mind `Pop`; plus one backend C fixture for the fixed-frame stack shape and `mindRecordSignal` stack-control lowering.

The backend fixture pins `AutomataStack` as explicit `frame0` through `frame3` fields with no heap/vector allocation, and pins `mindRecordSignal` handling for `Goto`, `Push`, and `Pop`. The fixtures intentionally do not cover dynamic Automaton graphs, root-frame semantics, keep-root behavior, type-erased machine storage, generic machine stepping, transition/interrupt scanning, decision memory, Actuation dispatch, Events, Trace, Replay, Persistence, parallel staged ticking, scheduler/async behavior, or DragonGod compiler hooks.

## Phase 20 / DG6 DragonGod Decision fixtures

DG6 adds fourteen Phase 20 fixtures: ten direct `decisionChoose4` runtime fixtures for highest score, disabled option ignoring, source-order tie, fallback, first-selection recording, same-current refresh, min-commit, hysteresis, tie-epsilon, and allowed switching; one DG3 Memory score integration fixture; one DG5 Mind/Goto stack integration fixture; one native `transition decide` doctrine fixture; and one backend-C shape fixture asserting deterministic integer policy structures and excluding floats, randomness, allocation, scheduler/async, Blackboard, and mailbox markers.

## Phase 20 DragonGod DG7 actuation fixtures

DG7 adds fixtures under `language/phase20-dragongod-kernel/valid/` for the fixed-slot actuation host. Run fixtures cover allow dispatch, deny dispatch, completed status, failed status, missing-id `Empty` status, `AutomataSignal::Act` matching, `AutomataSignal::AwaitActuation` matching, and Mind recording of Act/AwaitActuation without automatic dispatch. A backend-C fixture pins the shape of `ActuationStatus`, `ActuatorHost`, `actuatorDispatch`, `nextId` mutation, fixed slots, complete/fail status emission, and absence of heap allocation, scheduler/async/event-bus, mailbox/blackboard, dynamic handler-table, and DragonGod compiler-hook markers. Phase 20 DragonGod now has 60 `.conception` fixtures; the full language corpus has 1112 `.conception` fixtures.

## Phase 20 DragonGod DG8 events fixtures

DG8 adds fixtures under `language/phase20-dragongod-kernel/valid/` for the fixed-slot Events subsystem. Run fixtures cover event push/read, missing-event fallback with cursor advancement to `bus.count`, future-only tail cursor behavior, same-type insertion order, multi-type reads with independent stream cursors, single-cursor skip behavior, Memory integration, and an `AutomataSignal` machine driven by an event payload. A backend-C fixture pins `EventBus`, `EventCursor`, `eventPush`, `nextId`/`count` mutation, fixed slots, `eventReadNextInt` cursor assignment, the capacity-panic reason, and absence of heap allocation, scheduler/async, mailbox, blackboard, event-bus runtime naming, and DragonGod compiler-hook markers. Phase 20 DragonGod now has 69 `.conception` fixtures; the full language corpus has 1121 `.conception` fixtures.


## Phase 20 DragonGod DG9 trace fixtures

DG9 adds fixtures under `language/phase20-dragongod-kernel/valid/` for the fixed-slot Trace subsystem. Run fixtures cover enter/exit recording, transition payloads, yield observations, `AutomataSignal::Succeed` tracing, deterministic event order, capacity panic behavior, direct signal-reason matching, and a `TraceSink` interface/dyn smoke path. A backend-C fixture pins `TraceEventKind`, `TraceEvent`, `TraceRecorder`, fixed slots, append count mutation, signal tracing, the capacity panic reason, and the absence of heap allocation, scheduler/async markers, event-bus integration, mailbox/blackboard terminology, DragonGod compiler hooks, and stdout/stderr logging. Phase 20 DragonGod now has 78 `.conception` fixtures; the full language corpus has 1130 `.conception` fixtures.

Phase 20 DragonGod coverage now contains 83 valid fixtures after DG10. DG10 adds five replay fixtures under `language/phase20-dragongod-kernel/valid/`: four run fixtures for deterministic apply-all, event push, actuation completion/cursor exhaustion, and agent-memory/RNG/Trace marker integration; plus one backend C fixture pinning the fixed-slot ReplayEvent/ReplayLog/ReplayDriver shape, switch-like `match` lowering, and absence of heap, file I/O, JSON, scheduler/async, blackboard/mailbox, persistence serializer, and DragonGod compiler hook artifacts.


### DG12 Dynamic AutomataGraph v0

Status: complete. Added fixed-slot `AutomatonGraph` metadata (`AutomatonNodeId`, `AutomatonNodeKind`, `AutomatonNode`) with add/find/root helpers, graph-to-Mind root `Goto`, Decision and Trace composition fixtures, and backend shape coverage. Deferred: type-erased machine execution, dynamic MachineOps, heap graph storage, dynamic transition scanning, graph persistence serialization, scheduler/async, and parallel execution. Phase 20 DragonGod fixture count increased by seven DG12 fixtures (six run, one backend C).
### Phase 20 DG11 Persistence/checkpoint fixtures

Added 9 Phase 20 fixtures for DragonGod Persistence/checkpoint v0: 8 run fixtures covering world/agent memory, events, actuation, trace, RNG seed, restore-all, and replay/checkpoint composition, plus 1 backend-C shape fixture. The fixtures keep persistence in-memory only and assert no file/JSON/binary/scheduler/async/storage-adapter behavior.


## Phase 20 DragonGod closeout fixtures

Phase 20 is closed after DG12. The DragonGod fixture suite lives under `language/phase20-dragongod-kernel/valid/` and currently contains 99 valid `.conception` fixtures. The groups cover DG1 kernel core IDs/imports, DG2 static automata signals and signal-returning machines, DG3 Memory v0, DG4 Mind v0, DG5 Automata stack v0, DG6 Decision v0, DG7 Actuation v0, DG8 Events v0, DG9 Trace v0, DG10 Replay v0, DG11 Persistence/checkpoint v0, and DG12 AutomataGraph metadata v0.

These fixtures now serve as a Concept stress/regression suite for upcoming language and library work. They intentionally preserve fixed-slot v0 shapes until arrays/slices/fixed buffers, standard Option/Result ergonomics, strings/byte slices, generic containers, callable values, and stronger dyn storage are available.

## Phase 21 arrays, slices, and fixed buffers fixtures

`language/phase21-arrays-slices-fixed-buffers/` contains M1 fixtures for fixed-size array type syntax. Valid fixtures cover `T[N]` in struct fields and function parameters plus nested postfix spelling `T[N][M]`, interpreted as an array of `M` elements of `T[N]`. Invalid fixtures document the M1 length restriction: only positive integer literals are accepted; zero, negative, identifier, and expression lengths are rejected. Array literals, indexing, length/capacity operations, slices, mutable slices, and fixed buffers intentionally have no positive fixtures yet.

- Phase 21 arrays/slices/fixed-buffers fixtures now include fixed array literal valid cases and invalid diagnostics for `CON0400`, `CON0401`, and `CON0412`.

Phase 21 M3 expands the arrays/slices/fixed-buffers corpus with run fixtures for constant fixed-array reads, local variable fixed-array reads, nested repeated reads, `Len(values)`, and `Len(matrix[0])`. Invalid fixtures pin non-array indexing (`CON0406`), non-integer indexes (`CON0407`), and constant high/negative out-of-bounds indexes (`CON0414`). Mutable indexed assignment remains deferred beyond M3.

### P21-M4 status: mutable fixed-array element assignment

P21-M4 teaches Stage 0 to treat fixed-array indexing over assignable places as a mutable place projection. Assignments such as `values[1] = 99;` and nested projections such as `matrix[1][0] = 42;` now type-check and lower through HIR/MIR place machinery. The assigned value is checked against the projected element type, constant indexes keep the existing static out-of-bounds diagnostic, and generated C emits the same bounds guard used by read indexing before the store.

This milestone does not add slices, mutable slices, fixed buffers, `Capacity`, unchecked indexing, pointer decay, array-to-slice conversion, or DragonGod migration.

### P21-M5 fixed-array value lowering fixtures

P21-M5 expands `language/phase21-arrays-slices-fixed-buffers/` with run fixtures for fixed-array value assignment/copy, by-value function parameters, by-value returns, struct fields containing arrays, nested array assignment, and dynamic indexed reads after a copy. Invalid fixtures pin exact fixed-array assignment typing by rejecting different lengths and different element types. These fixtures verify that the C backend's wrapper-struct representation does not leak C array non-assignability or parameter decay into Concept semantics.

### P21-M6 status: read-only slices

P21-M6 adds the initial read-only `Slice<T>` view model. A slice is a borrowed pointer-plus-runtime-length view over contiguous storage; it does not allocate, own, copy, drop, or extend the lifetime of its backing array. M6 supports array-to-slice conversion only at function call boundaries where the parameter type is explicitly `Slice<T>` and the argument is a fixed array with the same element type. `Len(slice)` returns the runtime slice length, and `slice[index]` is read-only with runtime bounds checks using the stable panic reason `Concept slice index out of bounds`.

Explicit local `Slice(values)` construction, slice returns, slice fields, mutable slices/`MutSlice<T>`, fixed buffers, `Capacity`, range slicing syntax, C ABI guarantees, and DragonGod migration remain deferred. Slice element assignment is rejected in M6.

### P21-M7 FixedBuffer fixtures

P21-M7 expands `language/phase21-arrays-slices-fixed-buffers/` with run fixtures for `FixedBuffer<int, 4>` empty length, append plus read indexing, fixed capacity, and value-copy behavior. Invalid fixtures pin zero capacity (`CON0420`), malformed `FixedBuffer<T>` spelling (`CON0419`), and append element mismatch (`CON0421`). The fixtures preserve the M7 boundary: no heap vector, allocator, iterator, mutable slice, fixed-buffer-to-slice conversion, direct element assignment, or DragonGod migration coverage is added.

## Phase 21 / M8 DragonGod FixedBuffer migration fixtures

P21-M8 keeps the Phase 20 DragonGod fixture count stable but updates four backend-C shape fixtures: `kernel_trace_backend_shape.valid.conception`, `kernel_events_backend_shape.valid.conception`, `kernel_replay_backend_shape.valid.conception`, and `kernel_graph_backend_shape.valid.conception`. These fixtures now assert generated fixed-buffer wrapper storage (`storage` plus `count`), prove the migrated append/read/count paths still compile, preserve DragonGod-specific panic strings, and assert that old explicit event/slot/node fields are absent for the migrated types. Existing run fixtures continue to cover behavior for Trace, Events, Replay, Graph, checkpoint composition, and integrations.

## Phase 21 closeout fixture status

Phase 21 closes with 50 fixtures under `language/phase21-arrays-slices-fixed-buffers/`: 26 valid fixtures and 24 invalid fixtures. The full `language/` corpus currently contains 1201 `.conception` fixture files, and the Phase 20 DragonGod stress suite remains at 99 valid fixtures.

The Phase 21 fixtures now cover fixed array type syntax, array literals, fixed-array indexing, mutable fixed-array element assignment, value lowering, read-only slices, fixed buffers, capacity/length builtins, and the DragonGod migration spike through updated backend-shape coverage for Trace, Events, Replay, and AutomataGraph.

Future `.conception` fixtures should prefer human-readable C++-style formatting over compressed one-liners: one statement per line, spaces around assignment and binary operators, spaces after commas, and readable block indentation. Compact syntax may remain useful for narrow parser tests, but feature and backend fixtures should be readable by human reviewers.

## Phase 22 Option/Result/bounded mutation fixtures

P22-M1 adds the initial fixture set under `language/phase22-option-result-bounded-collection-mutation/`. Valid fixtures cover `Option<int>` `optionSome<int>(value)`, `optionNone<int>()`, assignment/copy, and function parameter/return value paths through `optionOr`. Invalid fixtures pin malformed `Option` spelling (`CON0500`), `optionNone()` without a type argument (`CON0511`), and `optionSome<int>(true)` payload mismatch (`CON0512`). Match ergonomics, Result, BufferError, FixedBuffer try APIs, and DragonGod migrations are intentionally absent from this milestone's fixture surface.
