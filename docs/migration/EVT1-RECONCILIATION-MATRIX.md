# EVT1 semantic reconciliation matrix

Status: R0 authority ledger with R1 executable evidence

`Port required?` means implementation or conformance work remains after R0; it
does not authorize that work in this milestone. Status is one of `Keep PoC3`,
`Keep Go`, `Merge`, `Redesign`, `Profile-only`, or `Deferred`.

| Area | PoC3 Zig behavior | Concept/Vulkan Go behavior | EVT1 decision | Status | Port required? | Test source | Notes |
|---|---|---|---|---|---|---|---|
| source unit/module syntax | module-oriented PoC3 surface | mandatory `profile Vulkan;` source unit | require explicit `profile Core;` or `profile Vulkan;` in R0 | Merge | No for R0 | `profile_test.go`; phase1 fixtures | broader module law deferred |
| imports | implemented multi-module compiler and diagnostics | dotted paths parsed; Vulkan import used as admission marker | preserve syntax; Core import resolution is not active | Keep PoC3 | Yes | phase16 fixtures | no fake resolver in R0 |
| naming | C++ lineage, but fixtures include older surface conventions | PascalCase types/functions/operations; camelCase locals/parameters | use Go line's explicit naming law | Keep Go | Translation | extracted examples | backend/MIR names may remain snake_case |
| declaration syntax | includes `fn`, `name: Type`, arrows, `let`/`var` in historical fixtures | return type first; `Type name`; braces | canonical C++-shaped Go syntax | Keep Go | Translation | phase1/2 plus extracted examples | material surface conflict resolved explicitly |
| primitive types | broader general systems-language set | `int`, `bool`, `void`, compile-time `string`, `uint64`, profile handles | canonize minimal core; widths/string runtime remain open | Merge | Yes | phase1; Go type tests | Vulkan handles are not core |
| structs | runtime structs, places, initialization, MIR/C lowering | mutable structs, positional construction, field checks | retain Go executable subset; reconcile broader place laws | Merge | Yes | phase7; M1B-A tests | core candidate |
| immovable/value types | `immovable`, move/drop integration | bounded `immovable struct`, copy rejection | preserve spelling and bounded rule provisionally | Merge | Yes | phase10; M1B-A tests | complete storage law missing |
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
| arrays | runtime fixed arrays | fixed arrays only in compile-time domain | canonical compile-time subset; runtime deferred | Merge | Yes | phase21; M1B-D tests | do not infer runtime storage law |
| runtime arrays | implemented with MIR/C in PoC3 | explicitly excluded | defer and port from PoC3 deliberately | Keep PoC3 | Yes | phase21 fixtures | not R0 |
| indexing | runtime and array indexing pressure | bounded compile-time array indexing | keep bounded subset | Merge | Yes | phase21; M1B-D diagnostics | runtime bounds law deferred |
| slices | read-only `Slice<T>` implemented | absent | preserve as legacy pressure | Keep PoC3 | Yes | phase21 fixtures | deferred |
| FixedBuffer | compiler-known bounded buffer implemented | absent | preserve design and fixtures; do not port in R0 | Keep PoC3 | Yes | phase21/22 fixtures | helper spree prohibited |
| Option | compiler-known `Option<T>` plus exhaustive Some/None match | absent | preserve PoC3 reference; defer EVT1 adoption | Keep PoC3 | Yes | phase22 fixtures | not R0 |
| Result | design only/incomplete at cutover; function fallibility exists | profile signatures use `Result<T,E>` shape | redesign unified failure model before core adoption | Redesign | Yes | phase5/22; Vulkan examples | no general Result implementation in R0 |
| ownership vocabulary | broad values/references/pointers/store doctrine | parses owned/borrow/const/imported/unsafe in bounded checks | retain spelling provisionally | Merge | Yes | phase6/10; Go diagnostics | vocabulary is not full semantics |
| borrow | place and call rules across PoC3 features | exact borrowed values and automata context restrictions | merge only after differential cases | Deferred | Yes | phase6/10; DragonGod M2 tests | profile context law may stay narrower |
| owned | move/drop/allocation interactions | bounded owned resource/value checks | retain as provisional syntax | Deferred | Yes | phase10/12; Go tests | no silent full ownership claim |
| move | explicit move and invalidation | narrow `move` pressure in historical mechanism path; no full model | port PoC3 semantics only after core tests | Keep PoC3 | Yes | phase10 fixtures | not R0 |
| unsafe | explicit unsafe blocks/operations | qualifier used for admitted foreign/Vulkan types | reconcile core escape hatch and profile admission | Merge | Yes | phase6; Vulkan examples | no broad allowlist yet |
| allocation | arenas/stores/allocators and allocation effects | no general model | retain PoC3 as design/reference pressure | Keep PoC3 | Yes | phase12 fixtures | no hidden heap |
| fallible functions | PoC3 fallible calls and propagation | profile `Result`/`?` direction | redesign with Result/panic distinction | Redesign | Yes | phase5/22; kernel examples | not canonical core |
| panic/assert | stable runtime panic/assert and test behavior | only compile-time `static_assert` | preserve PoC3 reference; defer runtime adoption | Keep PoC3 | Yes | phase17 fixtures | static_assert stays canonical |
| C ABI | extern/export/repr(C) implemented | emits C/H but does not define Concept FFI law | preserve PoC3 pressure; specify anew atop C11 backend | Redesign | Yes | phase15 fixtures | backend is not ABI spec |
| interfaces/dyn | implemented bounded interface/dyn dispatch | absent | preserve PoC3 reference and defer | Keep PoC3 | Yes | phase14 fixtures | dyn storage not in R0 |
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
| diagnostics/provenance | stable `CON` diagnostics and fixtures | stable `CV` diagnostics with spans/maps | preserve both; map deliberately in conformance work | Merge | Yes | invalid corpora; Go tests | code namespace rename deferred |

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
| arrays/indexing | yes | PASS only in compile-time domain | runtime arrays remain a separate Keep PoC3 row |
| effects/actuators | yes | PASS Core denial; Vulkan preservation suite PASS | general effect law remains deferred |
| profiles | yes | PASS through registered admissions and Core denial corpus | future profile API breadth remains deferred |
| MIR/C backend | yes | PASS via semantic MIR facts, deterministic goldens, and native C tests | broader feature lowering remains row-specific |
| diagnostics/provenance | yes | PASS by active semantic category | `CON`/`CV` namespace reconciliation remains |

Rows not listed in this R1 table are untouched. In particular, no evidence in
R1 changes the status of runtime arrays, Slice, FixedBuffer, Option/Result,
ownership, allocation, C ABI, interfaces/dyn, machines, decide, yield, testing,
or multi-module compilation.
