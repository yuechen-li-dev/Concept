# EVT1 semantic reconciliation matrix

Status: R0 authority ledger with R1-R5b executable evidence

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
| semantic layout | SDSL-V/Oct lineage carries semantic region graphs | fixed named regions with derived geometry and whole-storage binding | adopt bounded fixed graph in active Go compiler; no Oct dependency | SDSL-derived / EVT1-new surface | No for R4f fixed subset | R4f MIR/native corpus | runtime parameters and ABI deferred |
| layout alignment | explicit shader/storage alignment pressure | natural placement plus power-of-two `align(N)`, maximum total alignment, and tail padding | make requested alignment semantic and padding mechanical | SDSL-derived / EVT1-new | No for R4f subset | R4f align corpus/native C | executable inline backing guarantee is 64 bytes |
| layout explicit offset | register/packed-interface pressure | `at(N)` pins an aligned region; gaps allowed and overlap rejected | bounded explicit geometry without volatile/MMIO implication | EVT1-new | No for R4f subset | R4f offset/overlap corpus | unions and overlap deferred |
| layout region identity | semantic field/channel identity | stable `Layout.region` ID with parent, type, offset, extent, alignment, disjointness | preserve in typed model and MIR | SDSL-derived / Keep Go MIR | No for R4f subset | R4f MIR inspection | future noalias may consume; no solver yet |
| stream declaration | SDSL-V stream records | zero-storage named channel map over exactly one layout | generalize for Core CPU/GPU meaning | SDSL-derived / EVT1-new surface | No for R4f subset | R4f corpus | not iterator, I/O stream, or runtime process |
| stream channel mapping | channel-to-field maps | aliases declared regions without changing type/shape/identity | preserve semantic aliases and reject fabrication | SDSL-derived | No for R4f subset | R4f MIR/native alias evidence | composition/transforms deferred |
| stream binding | lineage mapping realized downstream | unary `bind` from matching bound layout, preserving const/provenance | reuse R4e verb and reference laws | EVT1-new composition | No for R4f subset | R4f native/provenance corpus | direct storage-to-stream bind deferred |
| tensor type | no canonical storage/interpretation split | non-owning `tensor<T,Rank>` over existing shaped storage | adopt mathematical interpretation without storage ownership | EVT1-new / SDSL-V-derived | No for R4h subset | R4h corpus and Tensor MIR | preserves shape, region, alignment, mutability, and provenance |
| symbolic Einstein indexing | no active PoC3 counterpart | statement-local inferred symbols with free/reduction validation | adopt bounded explicit contraction and initialization form | EVT1-new / SDSL-V-derived | No for R4h subset | R4h Einstein/native/comptime evidence | no persistent hidden state |
| tensor `@` | no active PoC3 counterpart | last-axis/first-axis generalized contraction plus rank-one scalar result | canonical deterministic default; matrix multiply and dot product are derived cases | EVT1-new / SDSL-V-derived | No for R4i subset | R4h rank-2/rank-3 and R4i rank-zero native evidence | no special dot operator |
| tensor elementwise shape law | no active PoC3 counterpart | `+`, `-`, `*` require exact shapes; exact-type scalar multiply | prohibit implicit broadcasting | EVT1-new | No for R4h subset | R4h valid/invalid corpus | no row/column/singleton promotion |
| inline tensor backing synthesis | absent | `tensor<T> name[fixed shape]` derives rank and fixed ndarray backing | adopt as sugar over R4h view semantics | EVT1-new / R4h-derived | No for R4i subset | R4i corpus, MIR, native C11 | runtime/external source remains explicit |
| Tensor backing witness | absent | closed Inline/NDArray/BoundNDArray/Span/LayoutRegion/StreamChannel classification | retain structured shape/provenance/region/alignment/mutability evidence | Keep Go / EVT1-new | No for R4i subset | R4i MIR inspection | compile-time classification; no runtime vtable |
| Tensor MIR | absent | explicit tensor views, rank-zero scalar contraction, and per-function semantic operations before loop lowering | retain math intent through validation, then lower deliberately | Keep Go / EVT1-new | No for R4i subset | R4h-R4i MIR validation | no separate vector/matrix MIR |
| vector / matrix aliases | no canonical split | exact parser normalization to `tensor<T,1>` / `tensor<T,2>` | adopt spelling-only identity | EVT1-new / R4h-derived | No for R4i subset | R4i API/native corpus | no separate semantics, witnesses, or storage |
| semantic fact qualification | scattered retained semantic fields | one typed fact/proof vocabulary with certainty, provenance, subjects, and deterministic evidence | expose known R4 storage/view/tensor facts without reinference | EVT1-new / R4-derived | No for R4j subset | R4j concept, MIR, and preservation corpus | optimizer consumes facts later; no optimizer now |
| contiguity/alignment qualification | layout and storage fields | `Contiguous` and parameterized `Aligned` proof | preserve structural and value-level guarantees separately | EVT1-new / R4-derived | No for R4j subset | R4j array/Span/layout/tensor cases | Subspan may reduce alignment |
| region identity and disjointness | layout sibling identity; tensor alias checks | `RegionIdentity`, `SameRegion`, and bounded `Disjoint` facts | retain declared regions and fixed half-open interval proof | EVT1-new / R4-derived | No for R4j subset | R4j layout/stream/Subspan/inline tensor cases | runtime overlap remains unknown; no alias solver |
| storage-neutrality qualification | per-operation booleans | `NoAllocation`, `NoCopy`, `NoOwnershipTransfer` facts | make bind/stream/Span/tensor evidence queryable | EVT1-new / R4-derived | No for R4j subset | R4j MIR and C erasure checks | fact applies to the qualified view/construction |
| optimizer fact consumer | absent | `SemanticFactSet` query API over MIR facts | future backend-independent consumption | EVT1-new | No for R4j subset | deterministic query tests | SIMD/LIR/noalias lowering deferred |
| indexing | runtime fixed-array indexing and bounds pressure | checked rank-1 and comma-separated rank-N indexing | merge array behavior; add EVT1-new ndarray arity/linearization | Merge / EVT1-new | No for fixed subset | phase21; R4d native/MIR tests | terminal panic, no Result default |
| slices | read-only `Slice<T>` implemented | superseded by R4g spans | preserve fixtures as pressure; do not port the old surface | Redesign -> superseded by Span/ReadOnlySpan | No direct port | phase21 fixtures; R4g corpus | bounds/read-only evidence only |
| Span | no canonical counterpart; Slice supplies pressure | mutable bounded borrowed contiguous interval | canonical EVT1 compiler-known ref-struct-like descriptor | EVT1 canonical / C#-inspired | No for R4g subset | R4g MIR/native corpus | same provenance and parent-region identity; no ownership/allocation |
| ReadOnlySpan | read-only Slice supplies pressure | readonly bounded borrowed contiguous interval | canonical readonly counterpart with explicit Span-to-ReadOnlySpan conversion | EVT1 canonical / reconciled pressure | No for R4g subset | R4g MIR/native corpus | no reverse conversion or const stripping |
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
| class | no canonical lightweight class model | ordinary value aggregate plus methods and access control | EVT1-new narrow encapsulation model | EVT1-new | No for R4k subset | R4k corpus | no implicit heap, identity, header, or inheritance |
| access control | broader object-model pressure | public/private class member boundaries | adopt public/private only; class defaults private and struct defaults public | EVT1-new | No for R4k subset | R4k valid/invalid corpus | no protected, friend, or package-private |
| methods | interface/function machinery | aggregate methods normalized to explicit receiver functions | reuse ordinary callable and value semantics | Redesign / EVT1-new | No for R4k subset | R4k MIR/native corpus | concrete calls are static; no virtual keyword |
| interface | implemented nominal bounded interface subsystem | specialized concept using ordinary requirement satisfaction | Redesign -> specialized concept | Redesign / implemented R4k | No for bounded R4k subset | phase14 pressure; R4k corpus | all interfaces are concepts; no parallel satisfaction engine |
| interface fields/composition | method-oriented legacy pressure | mechanical field accessors and prerequisite flattening | adopt as fixed-shape witness mechanics and concept composition | EVT1-new | No for R4k subset | R4k field/composition corpus | composition is not inheritance |
| witness | implementation records and dynamic call slots | deterministic concept proof reified only for dyn | one `(interface, concrete type)` static table | Redesign / implemented R4k | No for R4k subset | R4k MIR/C inspection | semantic facts erase; no per-object vtable |
| dyn | bounded runtime dispatch | explicit non-owning erased reference plus witness | Redesign -> witness reification | Redesign / implemented R4k | No for borrowed R4k subset | phase14 pressure; R4k native corpus | source provenance and constness preserved |
| owning dyn | legacy runtime-storage pressure | absent | require explicit erased-storage and allocator policy | Deferred | Yes | phase14 evidence only | no hidden allocation |
| inheritance | object-model lineage pressure | absent | reject from EVT1 narrow class model | Rejected | No | R4k parser/spec boundary | use interface composition |
| automata | application composition pressure | bounded outer runtime model | canonical outer persistent composition unit | Merge / implemented R5a | No for R5a subset | R5a corpus; DragonGod M0-M4 | no scheduler or hidden runtime |
| machine | stepping frames, fields, transitions, nested values | contained declarations and one active stack | canonical independently stepped unit contained by automata | Merge / implemented R5a | Yes beyond direct fields | phase13/18; R5a native corpus | nested machine values and completion/result deferred |
| state | named executable machine states | named signal-handler states | canonical named execution state with deterministic identity | Merge / implemented R5a | Translation | phase13/18; R5a MIR | first declaration is initial in R5a canonical form |
| automata shared state | implicit/application-owned pressure | one optional borrowed context | explicit `with state` environment shared by contained machines | EVT1-new / implemented R5a | No for R5a subset | R5a value/ref/owned/dyn/Span cases | identity is `Automata#state` |
| machine persistent field | PoC3 machine frame fields | absent from declarative Go lineage | inline machine-local persistent field | Merge / implemented R5a | No for R5a subset | phase18 pressure; R5a corpus | sibling access rejected |
| transient state local | re-entered state-body local | handlers did not admit ordinary bodies | ordinary local recreated per Step and cleaned at exit | Redesign / clarified R5a | No | phase19 re-entry pressure; R5a MIR/C | never compiler-lifted |
| basic transition | literal local target | signal handler `goto` | `transition Target;` changes only current machine state | Merge / implemented R5a | Translation only | phase13; R5a transition cases | transition exits the current Step after cleanup |
| transition match | runtime bool categorical transition | guarded signal and ordinary payload-enum match | exhaustive ordinary payload-enum pattern dispatch to local state | Merge / implemented R5b | Translation | phase13/18; R5b enum/Result native corpus | scrutinee once; no ranking |
| transition decide | deterministic int utility transition | guarded signal candidates | local stateless hardmax over exact `int` or `float` scores | Keep PoC3 semantic core / implemented R5b | Translation | phase5a/13/18; R5b native corpus | not DragonGod stateful policy |
| decision guard | optional bool candidate condition | handler guard | declaration-order bool gate evaluated once | Merge / implemented R5b | Translation | phase18; R5b evaluation counters | disabled score is not evaluated |
| decision score | scalar int utility | no scored transition primitive | exact per-block `int` or `float` candidate utility | Merge / implemented R5b | Partial | phase5a/18; R5b int/float cases | NaN is terminal invalid score |
| hardmax and ties | highest int; first source-order tie | ordered guarded selection without scores | first declared enabled candidate with maximum score | Canonical / implemented R5b | Translation | phase18; R5b tie case | strict `>` replacement |
| no enabled decision | stable runtime panic | dispatch may report unhandled | terminal `machine decision transition has no enabled candidates` | Merge / implemented R5b | Translation | phase18; R5b native panic | no implicit stay |
| Step | explicit caller-invoked machine step | signal-driven dispatch | `Step(instance, Machine)` executes one selected contained machine | Merge / implemented R5a | Translation only | phase13/18; R5a native corpus | no implicit tick-all order |
| Complete/Result | explicit completed/result frame operations | dispatch outcomes | reconcile user surface later | Deferred after R5a | Yes | phase13/18 and Go outcome evidence | not required for persistent-state law |
| DragonGod Decision | stateful kernel decision policy | application subsystem pressure | library-level hysteresis/commitment/tie-memory policy | Library-level / deferred | Yes | phase20 DragonGod evidence | never implicit in transition decide |
| infer | absent | absent | future normalized soft belief/distribution | Deferred / new direction | Yes | R5b decide/infer direction | no softmax or distribution in R5b |
| transition infer | absent | absent | future inference-policy-driven transition | Deferred / new direction | Yes | R5b decide/infer direction | non-normative |
| yield | explicit re-entry suspension without lifted locals | absent | preserve R5a storage law; define resume behavior in R5c | Deferred to R5c | Yes | phase19 fixtures | future yield preserves explicit persistent storage only |
| effects | general design plus allocation/effect vocabulary | ordered typed effect batches in automata | keep Go implementation Vulkan-only in R0 | Profile-only | No for profile subset | DragonGod M3 tests | general effect law deferred |
| actuators | not the same extracted mapping construct | typed exact effect-to-mechanism mapping | Vulkan profile only | Profile-only | No for profile subset | actuator tests | not core Concept |
| profiles | PoC3 general profile and effect-default design | only Vulkan profile | explicit Core/Vulkan admission with no silent broadening | Merge | Yes | profile tests; PoC3 docs | future profiles require registration design |
| testing | `.con_test`, facts/theories, assertions, runner | Go compiler tests only | preserve PoC3 surface as legacy reference | Keep PoC3 | Yes | phase11 fixtures | no Concept testing port in R0 |
| multi-module compilation | implemented module table and diagnostics | single source unit only | preserve PoC3 semantics as port candidate | Keep PoC3 | Yes | phase16 fixtures | imports alone are not support |
| MIR | typed PoC3 MIR plus validators/storage | typed deterministic Go MIR/source maps | keep Go architecture; port semantic facts selectively | Merge | Yes | phase4; Go exact artifacts | active MIR schema is `concept-evt1-mir.v1` |
| General Planner | absent as a distinct authority | explicit MIR/facts/target/profile/policy stage | EVT1 compiler architecture, not language surface | EVT1-new architecture | No | R4l plan/validator tests | consumes proofs; does not recreate sema |
| LoweringPlan | implicit in C lowering | deterministic `concept-evt1-plan.v1` artifact | expose realization choices and evidence | EVT1-new architecture | No | R4l CLI and identity tests | future LIR input |
| TargetCapabilities | backend assumptions | GenericC11, X86_64_Generic, AArch64_Generic bounded records | make planning inputs explicit | EVT1-new architecture | No | R4l three-target test | native targets planning-only |
| tensor planning | direct C loop selection | `DirectLoopNest`, traversal, guards, eligibility | move realization ownership after semantic Tensor MIR | EVT1-new architecture | No | R4l fixed/runtime tensor plans | SIMD eligible may remain unselected |
| bounds planning | checks selected directly by lowerer | static/per-access/hoisted/proven classifications | preserve existing guards explicitly | EVT1-new architecture | No | R4l Span/tensor tests | no aggressive elimination |
| cleanup planning | MIR cleanup plus direct lowerer order | reverse-order drops and transferred-owner suppression | expose existing lifecycle strategy | EVT1-new architecture | No | R4l cleanup validation | no semantic change |
| dyn dispatch planning | direct witness C emission | object/witness pair, indirect method, field accessor | consume R4k witnesses without rebuilding satisfaction | EVT1-new architecture | No | R4l dyn plan tests | no RTTI/allocation/vtable |
| optimization eligibility | fact queries only | evidence-only bounded record with conservative fallback | separate legality from selection | EVT1-new architecture | No | R4l tensor eligibility tests | no transform in R4l |
| LIR boundary | absent | Plan is stable input to future machine-near LIR | document only | Deferred after R4l | No | planner direction document | no LIR/MachineIR today |
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
diagnostics. Slice is superseded by the later R4g Span direction; FixedBuffer
and vector/matrix/tensor were still future layers at R4d; R4h now supplies the
tensor layer; R4i now supplies exact vector/matrix aliases.

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
PoC3 bind equivalent. Slice and FixedBuffer remain redesign evidence;
allocator-backed dynamic ownership and vector/matrix/tensor were deferred at
R4e; R4h supplies the tensor layer and R4i supplies exact vector/matrix aliases.

## R4f executable evidence

R4f adds 29 `PASS` cases: 16 valid and 13 invalid. Fixed semantic layout
geometry and zero-storage channel maps are informed by SDSL-V lineage, but the
canonical source surface, ordinary Core region types, layout/stream `bind`, and
strict-C11 lowering are implemented in the active Go compiler. MIR retains
stable region identity, size/alignment/offset/extent/disjointness,
channel-to-region maps, backing identity, constness, provenance, and explicit
no-copy/no-allocation/no-transfer facts. Native evidence demonstrates mutable
and const aliases plus lexical, call-result, and scoped provenance. There is no
Oct build/runtime dependency and no claim of ABI, allocation, stream runtime,
GPU, or tensor support in R4f itself. R4h later consumes these retained facts for
tensor views and contraction.

## R4g executable evidence

R4g adds 29 `PASS` cases: 15 valid and 14 invalid-path programs. PoC3 Slice
fixtures contribute design pressure for pointer-plus-length representation,
bounds checks, and readonly access; R4g is not a direct Slice port. The
canonical Span/ReadOnlySpan surface is C#-inspired and composed with R4a/R4b
provenance, R4d storage, R4e bind, and R4f region/channel facts. MIR retains
parent identity, relative interval, byte extent, safe alignment, mutability,
contiguity, provenance, and no-copy/no-allocation/no-transfer facts. Native
evidence proves bidirectional alias visibility, correct Subspan writes,
readonly access, immovable backing, and deterministic terminal bounds paths.
Slice is now superseded direction; FixedBuffer remains redesign/deferred.

## R4h executable evidence

R4h adds 29 `PASS` cases: 17 valid and 12 static rejection programs. The
active compiler constructs non-owning tensor views from fixed/bound ndarrays,
layout regions, stream channels, and rank-one Span while retaining storage,
region, alignment, mutability, and provenance facts. Tensor computations
remain explicit in per-function Tensor MIR until a dedicated lowering stage
synthesizes strict-C11 loops. Native evidence covers exact-shape elementwise
arithmetic, scalar multiply, matrix and rank-three `@`, Einstein contraction,
ordinary bounds, runtime contraction guards, readonly operands, and
statement-local comptime initialization. Broadcasting, contraction overlap,
invalid symbolic mappings, and unshaped rank-greater-than-one Span conversion
reject. PoC3 parity is not fabricated; SDSL-V is lineage evidence only.

## R4i executable evidence

R4i adds 27 `PASS` cases: 16 valid and 11 static rejection programs. It is
EVT1-new ergonomic sugar derived from R4h semantics, not a PoC3 source-form
divergence. The corpus covers fixed inline rank one through three, nested
literals, exact scalar fill, const access, Einstein initialization, matrix and
rank-three contraction, exact vector/matrix aliases, scalar dot product, and
preservation of ndarray/layout/stream `Tensor(source)` forms. Runtime inline
shape, literal mismatch, unsupported element type, shorthand rank mismatch,
broadcasting, and contraction aliasing reject. MIR proves deterministic
disjoint inline regions and one existing Tensor MIR path; strict-C11 numeric
evidence proves results without allocation or tensor runtime calls.

## R4j executable evidence

R4j adds 24 `PASS` cases: 17 valid and 7 static rejection programs. It exposes
facts already established by R4 storage, layout, stream, Span, and tensor
semantics through a typed three-certainty model. The corpus proves unary and
bounded relational concept consumption, stable proof/fact identities,
layout/stream/Span/tensor preservation, alignment degradation, declared and
fixed-interval disjointness, fixed/runtime tensor qualification, and total C
erasure. Exact source-level shape tuples, general alias solving, optimization,
SIMD, noalias syntax, MLIR, and native backend work remain deferred.

## R4k executable evidence

R4k adds 30 required `PASS` cases: 16 valid and 14 statically rejected. Class,
access control, implicit receiver ergonomics, field witnesses, and concept-based
interface satisfaction are EVT1-new or redesigned semantics. PoC3 Phase 14 is
reference pressure for declaration, diagnostic, erased-storage, and dispatch
shape only; no nominal impl table or object model was mechanically ported.
Static templates remain monomorphized, while dyn explicitly reifies a borrowed
two-pointer value and one deterministic static witness per concrete/interface
pair. Owning dyn is deferred and inheritance is rejected.

## R5a executable evidence

R5a adds 20 required `PASS` cases: 12 accepted and 8 statically rejected. It
merges the outer Go automata hierarchy with PoC3 machine/state/Step/field and
basic-transition pressure while adding explicit `with state` capture. MIR
retains stable environment, machine, state, field, and local identities plus
`AutomataState`, `MachinePersistent`, and `TransientLocal`. Planner and native
C11 evidence prove inline storage, independently stepped machines, shared
state, transition cleanup, ref aliasing, owned cleanup, no lifted locals, and
no heap/scheduler/coroutine runtime. Decide, yield, Complete/Result, nested
machine values, effects, and actuators remain explicitly deferred.

## R5b executable evidence

R5b adds 26 `PASS` cases: 14 accepted positive programs, ten statically
rejected programs, and two accepted runtime-negative programs. It translates
PoC3 Phase 18 categorical and source-order hardmax behavior into canonical R5a
automata, extends matching through ordinary payload enums/Result, and makes
`int`/`float` score type and NaN policy explicit. MIR and Planner preserve
candidate order, tie/no-enabled/cleanup policy, and strict-C11 evidence proves
exactly-once evaluation without a scheduler or decision runtime object.
