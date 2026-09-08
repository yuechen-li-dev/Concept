# EVT1 semantic reconciliation matrix

Status: R0 authority ledger with R1-R4e executable evidence

`Port required?` means implementation or conformance work remains after R0; it
does not authorize that work in this milestone. Status is one of `Keep PoC3`,
`Keep Go`, `Merge`, `Redesign`, `Profile-only`, or `Deferred`.

| Area | PoC3 Zig behavior | Concept/Vulkan Go behavior | EVT1 decision | Status | Port required? | Test source | Notes |
|---|---|---|---|---|---|---|---|
| source unit/module syntax | module-oriented PoC3 surface | mandatory `profile Vulkan;` source unit | require explicit `profile Core;` or `profile Vulkan;` in R0 | Merge | No for R0 | `profile_test.go`; phase1 fixtures | broader module law deferred |
| imports | implemented multi-module compiler and diagnostics | dotted paths parsed; Vulkan import used as admission marker | preserve syntax; Core import resolution is not active | Keep PoC3 | Yes | phase16 fixtures | no fake resolver in R0 |
| naming | C++ lineage, but fixtures include older surface conventions | PascalCase types/functions/operations; camelCase locals/parameters | use Go line's explicit naming law | Keep Go | Translation | extracted examples | backend/MIR names may remain snake_case |
| declaration syntax | includes `fn`, `name: Type`, arrows, `let`/`var` in historical fixtures | return type first; `Type name`; braces | canonical C++-shaped Go syntax; typed `let` is only a const alias | Keep Go | Translation | phase1/2 plus R2 corpus | inference remains absent |
| primitive types | broader general systems-language set | `int`, `bool`, `void`, compile-time `string`, `uint64`, profile handles | canonize minimal core; widths/string runtime remain open | Merge | Yes | phase1; Go type tests | Vulkan handles are not core |
| structs | runtime structs and places; PoC3 later requires explicit `Copy` conformance | mutable structs, positional construction, field checks, structural copy | ordinary structs are mutable values and copy when all fields are copyable | Merge | No for R2 subset | phase7; R2 corpus | default structural copy is an expected PoC3 divergence; drop remains open |
| record struct | no corresponding syntax or type-level record immutability | absent before R2 | immutable value aggregate with copy/update construction | Keep Go | No for R2 subset | `language/evt1-r2/core` | EVT1-new evidence, not a PoC3 divergence |
| const binding/place | historical qualifiers and ownership-era place rules | type qualifier existed; local binding was mutable before R2 | `const` freezes reassignment and projected mutable places, independently of record type immutability | Merge | No for local and natural parameter subset | R2 const corpus | full borrow/alias law remains deferred |
| with expression | no corresponding syntax | absent before R2 | fresh copyable-record update; source evaluated once and preserved | Keep Go | No for R2 subset | R2 MIR/C/native tests | ordinary structs and non-copyable records rejected |
| immovable/value types | design pressure but no closed PoC3 fixture implementation; ownership fixtures cover non-copyable flow | bounded `immovable struct`, final-storage construction and reference access | canonize final-storage law: no copy or relocation, mutable access through `ref` | Merge | Yes beyond R3 subset | phase10 pressure; M1B-A; R2-R3 tests | generalized pinning and reference aggregates remain deferred |
| payload enums | implemented sum types and payload construction | typed payload enums with deterministic tags | canonical bounded payload enum subset | Merge | No for subset | phase5; M1A tests | generic built-ins separate |
| variant construction | PoC3 constructor forms, including built-in forms | qualified `Enum::Variant(args)` | prefer qualified Go form for EVT1 | Keep Go | Translation | phase5; M1A examples | built-in Option syntax deferred |
| match syntax | PoC3 match and several later machine forms | `match (value)` with qualified enum arms | canonical Go expression spelling for base enum match | Keep Go | Translation | phase5; M1A tests | machine match not covered |
| match exhaustiveness | exhaustive payload-enum checking | exhaustive exact variants; duplicate/missing diagnostics | preserve strict exhaustiveness | Merge | No for subset | phase5 invalid; M1A diagnostics | advanced patterns deferred |
| concept declarations | broad named/marker/negative concept design and implemented subset | one type parameter; named operation/prerequisite requirements | canonical bounded Go subset | Merge | Yes | phase8; M1B-A tests | negative concepts not admitted |
| concept assertions | PoC3 concept conformance forms | `requires Concept<Type>;` explicit proof | keep explicit Go proof spelling | Keep Go | Translation | M1B-A tests | no implicit wholesale satisfaction claim |
| templates | broader PoC3 templates/concepts | one-parameter constrained function templates | canonical bounded Go subset | Merge | Yes | phase8; M1B-B tests | template types deferred |
| template constraints | PoC3 constraint vocabulary | named concept requirement before function | keep exact named constraint surface | Keep Go | Translation | M1B-B diagnostics | multiple constraints deferred |
| monomorphization | PoC3 compiler specialization evidence | deterministic instance identity and C/MIR symbols | preserve deterministic Go mechanism | Keep Go | No for subset | M1B-B deterministic tests | artifact identity is compiler evidence |
| comptime | broader PoC3 execution and capability direction | pure fuel/depth/aggregate-bounded evaluator | use Go bounded evaluator; reconcile capabilities later | Merge | Yes | phase9; M1B-C/D tests | no general I/O |
| static_assert | implemented PoC3 compile-time assertion | typed bounded compile-time assertion | canonical in bounded evaluator | Merge | No for subset | phase9; M1B-C tests | runtime assert is separate |
| while | runtime and compile-time loops | foundational runtime loop; comptime requires `bounded(limit)` | retain Go bounded compile-time law | Merge | Translation | phase2/9; M1B-C tests | wider loop surface deferred |
| if expressions | PoC3 control flow variants | typed `if (...) value else value` plus statements | retain Go implemented subset | Merge | Translation | phase2; M1B-C tests | statement/expression closure later |
| arrays | runtime fixed arrays and wrapper values | canonical rank-1 contiguous fixed-inline storage | expand to explicit `T<array>[N]`; retain `T[N]` compatibility | Merge / canonical R4d | No for fixed subset | phase21; R4d corpus | shape does not choose allocation |
| runtime arrays | implemented only as fixed-shape values in PoC3 | runtime extent is typed; non-owning exact-storage bind is active; owning declaration still rejects | redesign around explicit storage and later allocator-backed ownership | Redesign / active R4e subset | Yes for owned dynamic storage | R4d-R4e corpus | no hidden heap or VLA |
| ndarray | absent | canonical rank-N contiguous storage with row-major layout and exact-storage binding | EVT1-new storage family | EVT1-new | No for fixed/bound subset | R4d-R4e corpus; SDSL-V direction | not nested arrays; math layer deferred |
| explicit storage binding | absent | unary destination-typed `bind` reshapes entire contiguous storage without ownership | EVT1-new provenance-preserving view operation | EVT1-new / Keep Go | No for R4e subset | R4e MIR/native corpus | exact count; no allocation/copy/transfer |
| vector / matrix / tensor | no canonical storage/interpretation split | absent | defer mathematical interpretations above storage and views | Deferred | Yes | R4d direction note | no TensorIR or Einstein notation |
| indexing | runtime fixed-array indexing and bounds pressure | checked rank-1 and comma-separated rank-N indexing | merge array behavior; add EVT1-new ndarray arity/linearization | Merge / EVT1-new | No for fixed subset | phase21; R4d native/MIR tests | terminal panic, no Result default |
| slices | read-only `Slice<T>` implemented | absent | redesign against explicit references, bound-storage evidence, and future lifetime-bound spans | Redesign | Yes | phase21 fixtures; R4e direction note | do not port mechanically; bind is whole-storage only |
| FixedBuffer | compiler-known bounded buffer implemented | absent | preserve pressure; redesign after explicit views/storage construction | Redesign / deferred | Yes | phase21/22 fixtures | not ported in R4d |
| Option | compiler-known `Option<T>` plus exhaustive Some/None match | canonical compiler-known payload enum | adopt qualified `Option::Some/None`, exhaustive match, `?`, and `!` | Merge / adopt | Yes | phase22 plus R4c corpus | canonical in R4c |
| Result | design only/incomplete at cutover; function fallibility exists | canonical generic payload enum | adopt `Result::Ok/Error` as the only recoverable typed failure value | Redesign -> canonical payload enum | Partial | phase5/22 plus R4c corpus | `Error`, not PoC3 concrete `Err`, is canonical |
| ownership vocabulary | broad values/references/pointers/store doctrine | `owned`, compatibility `borrow`, canonical `ref`, imported/unsafe | make bounded local ownership/reference spellings canonical without claiming allocation or lifetime completeness | Merge | Yes beyond R3 | phase6/10; R2-R3 diagnostics | vocabulary is not a generalized ownership system |
| copyability | opt-in `Copy` for structs | structural copyability for ordinary values | ordinary values copy structurally; `owned T` is non-copyable | Keep Go | No for R3 subset | Phase 10; R2-R3 corpus | deliberate ordinary-struct divergence remains |
| movability | non-Copy values move explicitly | `owned T` supplies movable-only category | movability is broader than copyability; immovable is the exception | Merge | No for R3 subset | Phase 10; R3 corpus | no implicit move or partial move |
| borrow/ref | place and call rules across PoC3 features | legacy `borrow` plus explicit R3 `ref` expressions/types | `ref T` and `ref const T` are canonical non-owning aliases | Redesign | Yes beyond R3 | phase6/10 pressure; R3 EVT1-new cases | no generalized borrow checker |
| owned | move/drop/allocation interactions | bounded `owned T` movable-only ownership | canonize for local/call/return transfer and deterministic drop only | Merge | Yes beyond R3 | phase10; R3 tests | allocation/store ownership remains deferred |
| move | explicit move and invalidation | explicit whole-local/parameter move with branch joins | canonize explicit non-copyable transfer; copyable move is non-consuming | Merge | No for R3 subset | phase10 fixtures; R3 corpus | no implicit, field, or partial moves |
| drop | explicit MIR cleanup from `Drop<T>` witness | `void Drop(owned T)` witness, MIR validation, deterministic C cleanup, and live replacement | canonize reverse-order cleanup, transfer suppression, and drop-old-then-initialize-new | Merge | Yes beyond R4a | phase10 fixtures; R3-R4a MIR/native tests | no unwinding or partial drop |
| reference escape | lifetime analysis deferred | lexical provenance, `ref struct`, `scoped`, selected result summaries, and concept-requested proof | enforce local invariants universally and stronger analysis on semantic demand | Redesign | Yes beyond R4b | R3-R4b EVT1-new cases | no named/NLL/global borrow system |
| call-result provenance | lifetime analysis deferred | `Parameter`, `ShortestOfParameters`, reserved `Static`, and conservative `Unknown` summaries | preserve source bounds across selected helper calls | Redesign | Yes beyond R4b | R4b corpus and MIR tests | no arbitrary interprocedural inference |
| relational lifetime proof | lifetime analysis deferred | semantic parameter/result subjects and demand-driven `Outlives` | prove requested relations; reject disproven or unknown | Redesign | Yes beyond R4b | R4b concept and proof tests | source concepts remain single-parameter |
| scoped result propagation | lifetime analysis deferred | scoped formal/actual provenance survives call-result instantiation | helpers cannot hide a non-escape restriction | Redesign | Yes beyond R4b | R4b valid/invalid scoped cases | no general reborrow lattice |
| unsafe | explicit unsafe blocks/operations | qualifier used for admitted foreign/Vulkan types | reconcile core escape hatch and profile admission | Merge | Yes | phase6; Vulkan examples | no broad allowlist yet |
| allocation | arenas/stores/allocators and allocation effects | no general model | retain PoC3 as design/reference pressure | Keep PoC3 | Yes | phase12 fixtures | no hidden heap |
| fallible functions | PoC3 separate fallible syntax/channel | explicit `Result<T,E>` return type | canonicalize as Result-returning functions; legacy syntax is deferred sugar | Merge / Redesign | Translation | phase5 plus R4c | no second hidden failure channel |
| propagation | PoC3 concrete Result-shaped prefix `try` | postfix `?` over Option/Result | exact channel and exact `E`; no implicit conversion | Redesign | Partial | phase5 plus R4c | `?` never panics |
| try/except | no closed local typed handler subsystem | lexical exact-type Result handler boundary | EVT1-new ordinary branch/goto lowering | Redesign / EVT1-new | No | R4c corpus | no throw, cross-frame catch, or unwind |
| panic/assert | stable runtime panic/assert and test behavior | terminal helper, runtime assert, existing static_assert | merge stable reason pressure; assert/static_assert are Assert.True sugar | Merge | Partial | phase17 plus R4c | panic is not recoverable control flow |
| error conversion | no closed general conversion law | exact `E` only | defer concept-driven `ErrorConvertible<From,To>` | Deferred concept-driven | No | R4c negative corpus | no From/Into or residual machinery |
| C ABI | extern/export/repr(C) implemented | emits C/H but does not define Concept FFI law | preserve PoC3 pressure; specify anew atop C11 backend | Redesign | Yes | phase15 fixtures | backend is not ABI spec |
| interfaces/dyn | implemented bounded interface/dyn dispatch | absent | redesign as semantic concept + operation/semantic-proof witness + downstream runtime witness reification; interface may be sugar/marker | Redesign | Yes | phase14 fixtures; R4b witness direction | relational lifetime proof stays compile-time; dyn is not implemented |
| machines | `machine`, states, transitions, completion/result | nested `machine` declarations inside automata | do not alias surfaces in R0 | Deferred | Yes | phase13/18; DragonGod tests | general semantics likely core |
| automata | PoC3 uses Automata mainly in application architecture | bounded `automata -> machine -> state` runtime model | preserve provisionally pending reconciliation | Deferred | Yes | phase20; DragonGod M0-M4 | not automatically Vulkan-only |
| decide | judgment-driven deterministic transition selection | absent | preserve PoC3 reference | Keep PoC3 | Yes | phase5a/13/18 | compare with guarded candidates later |
| yield | explicit state-preserving suspension | absent | preserve PoC3 reference; defer | Keep PoC3 | Yes | phase19 fixtures | no scheduler implied |
| effects | general design plus allocation/effect vocabulary | ordered typed effect batches in automata | keep Go implementation Vulkan-only in R0 | Profile-only | No for profile subset | DragonGod M3 tests | general effect law deferred |
| actuators | not the same extracted mapping construct | typed exact effect-to-mechanism mapping | Vulkan profile only | Profile-only | No for profile subset | actuator tests | not core Concept |
| profiles | PoC3 general profile and effect-default design | only Vulkan profile | explicit Core/Vulkan admission with no silent broadening | Merge | Yes | profile tests; PoC3 docs | future profiles require registration design |
| testing | `.con_test`, facts/theories, assertions, runner | Go compiler tests only | preserve PoC3 surface as legacy reference | Keep PoC3 | Yes | phase11 fixtures | no Concept testing port in R0 |
| multi-module compilation | implemented module table and diagnostics | single source unit only | preserve PoC3 semantics as port candidate | Keep PoC3 | Yes | phase16 fixtures | imports alone are not support |
| MIR | typed PoC3 MIR plus validators/storage | typed deterministic Go MIR/source maps | keep Go architecture; port semantic facts selectively | Merge | Yes | phase4; Go exact artifacts | active MIR schema is `concept-evt1-mir.v1` |
| C backend | PoC3 backend supports its broad surface | strict-C11 generator for Go subset/profile | use Go backend as Stage 0 base | Keep Go | Yes | golden and native harness tests | neutral `concept_*` symbols |
| diagnostics/provenance | stable `CON` diagnostics and fixtures | stable `CV` diagnostics with spans/maps and R2 semantic categories | compare canonical semantic families; preserve both numeric namespaces for now | Merge | Yes | invalid corpora; R2 diagnostic table | mass renumbering deferred |

## R0 interpretation

Rows marked `Keep PoC3` do not reactivate the Zig roadmap. They preserve a
known semantic source for an eventual port. Rows marked `Keep Go` choose the
working Go architecture or syntax direction, not every historical
Concept/Vulkan restriction. `Merge`, `Redesign`, and `Deferred` rows require an
explicit later specification change and executable evidence. `Profile-only`
rows must remain rejected by `profile Core;`.

## R1 executable evidence

R1 preserves every R0 decision above and adds evidence only for the bounded
overlap. Full case metadata and provenance live in
`internal/concept/conformance_test.go`; the human summary is
`docs/conformance/EVT1-R1-CONFORMANCE.md`.

| Matrix area | R1 executable evidence | Conformance status | Remaining port requirement |
|---|---|---|---|
| primitive types | yes | PASS for `int`/`bool` values used by the corpus; profile denial PASS | widths and runtime string rules remain open |
| structs | yes | PASS | broader place/layout/drop law remains |
| payload enums | yes | PASS | generic built-in Option/Result remains excluded |
| variant construction | yes | PASS after qualified EVT1 translation | historical syntax remains translation-only |
| match syntax and exhaustiveness | yes | PASS | guards/wildcards/recursive patterns remain deferred |
| while | yes | PASS at runtime; EXPECTED-DIVERGENCE for missing compile-time `bounded(limit)` | wider loop surface remains deferred |
| if expressions | yes | PASS | statement/expression closure remains deferred |
| concept declarations/assertions | yes | PASS for one parameter, operation, prerequisite, and explicit assertion | broader concept system remains |
| templates/constraints | yes | PASS for one named constraint and explicit concrete invocation | broader templates remain |
| monomorphization | yes | PASS for deterministic two-type identity and duplicate reuse | none for selected subset |
| comptime/static_assert | yes | PASS for bounded pure subset and runtime-call rejection | capability model remains deferred |
| arrays/indexing | yes | superseded by R4d fixed runtime value/index/bounds evidence | runtime-shaped ownership still requires explicit storage design |
| effects/actuators | yes | PASS Core denial; Vulkan preservation suite PASS | general effect law remains deferred |
| profiles | yes | PASS through registered admissions and Core denial corpus | future profile API breadth remains deferred |
| MIR/C backend | yes | PASS via semantic MIR facts, deterministic goldens, and native C tests | broader feature lowering remains row-specific |
| diagnostics/provenance | yes | PASS by active semantic category | `CON`/`CV` namespace reconciliation remains |

Rows not listed in this R1 table are untouched. In particular, no evidence in
R1 changes the status of runtime arrays, Slice, FixedBuffer, Option/Result,
ownership, allocation, C ABI, interfaces/dyn, machines, decide, yield, testing,
or multi-module compilation.

## R2 executable evidence

R2 adds 32 value/place cases: 30 `PASS`, 2 `EXPECTED-DIVERGENCE`, and zero
`EVT1-GAP`, `POC3-LEGACY`, or `SPEC-AMBIGUITY`. The expected divergences make
the selected ordinary-struct structural copy and assignment law explicit
against PoC3 Phase 10's opt-in `Copy` behavior. `record struct` and `with` are
EVT1-new evidence and therefore remain `PASS`, not artificial PoC3
divergences.

The executable slice closes mutable struct fields, bounded structural copy,
const local and natural parameter places, immutable record fields, fresh
record update, existing bounded record structural equality, and immovable
final-storage/boundary/embed restrictions. Full ownership, move, drop, borrow,
aliasing, allocation, runtime collections, C ABI, interfaces, and automata
reconciliation remain open.

## R4b executable evidence

R4b adds 11 `PASS` cases: six valid and five invalid, with no fabricated PoC3
parity classification. Provenance is recorded as either the R4a foundation
plus EVT1-new relational semantics or EVT1-new relational semantics. The
active compiler preserves parameter-derived, nested, pass-through, scoped,
and shortest-of-parameters result bounds; executes requested `Outlives`
relations; rejects disproven or unknown proofs; and records deterministic MIR
subjects, facts, outcomes, and result summaries. Slice remains `Redesign`, and
Span, dyn, interface syntax, named/NLL lifetimes, and generalized alias
analysis remain unimplemented.

## R4c executable evidence

R4c adds 24 `PASS` cases: 14 valid and 10 invalid. PoC3 Phase 5 supplies
concrete early-return propagation pressure, Phase 17 supplies stable
panic/assert pressure, and Phase 22 supplies Option construction/match pressure.
Generic Result, postfix `?`/`!`, and local typed `try`/`except` are EVT1-new;
the matrix does not fabricate parity for them. The active rule is exact error
identity only. A future `ErrorConvertible<From, To>` concept may authorize an
explicit conversion witness, but R4c implements no generalized conversion.

## R4d executable evidence

R4d adds 30 `PASS` cases: 17 valid and 13 invalid. Fixed array literals,
indexing, mutation, `Len`, bounds behavior, copy semantics, and wrapper C
lowering merge the durable PoC3 Phase 21 evidence into the active compiler.
Ndarray, comma-separated rank-aware indexing, rectangular nested-literal
flattening, `Rank`/`Shape`, and row-major linearization are EVT1-new and are not
misclassified as PoC3 parity. Runtime extents are value-level type facts, but
both bare runtime array and ndarray locals reject with explicit-storage
diagnostics. Slice and FixedBuffer remain redesign/deferred, while
Span/ReadOnlySpan and vector/matrix/tensor remain future layers.

## R4e executable evidence

R4e adds 23 `PASS` cases: 12 valid and 11 invalid. It is primarily EVT1-new:
destination-typed unary `bind` reshapes an entire array/ndarray storage object,
preserves provenance and scoped state, and emits an explicit non-owning MIR/C
descriptor. Fixed mismatches are rejected statically; runtime products are
checked for overflow and exact equality before binding. Native evidence covers
array-to-ndarray binding, flattening, ndarray reshape, bidirectional alias
visibility, readonly access, runtime `Rank`/`Shape`, ref-struct storage, prior
bound-view rebinding, call-result provenance, and bind after `?`. PoC3 supplies
only supporting fixed-wrapper/value evidence; this matrix does not fabricate a
PoC3 bind equivalent. Slice, FixedBuffer, Span/ReadOnlySpan, allocator-backed
dynamic ownership, and vector/matrix/tensor remain deferred.
