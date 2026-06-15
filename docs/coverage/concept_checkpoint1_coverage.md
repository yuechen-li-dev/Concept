# Concept — Checkpoint 1 Coverage Matrix
## Phase 21 arrays/slices design start vs PoC3 Constitution

**Generated:** June 2026  
**Compiler:** Stage 0 (Zig, self-hosted Concept frontend, C backend via MIR)  
**Phases closed:** 1 through 20 / DG12
**Current phase:** Phase 21 P21-M0 — arrays, slices, and fixed buffers design-only milestone
**Fixture corpus:** 1151 total `.conception` fixture files; 99 under `language/phase20-dragongod-kernel/`.
**Stage target:** Stage 1 (MIR-complete, C backend from MIR, ownership/effects/machines)



## Phase 21 / P21-M0 arrays, slices, and fixed buffers design snapshot

Phase 21 is open as a design-only milestone. `docs/design/phase-21-arrays-slices-and-fixed-buffers.md` defines fixed-size owned arrays with `T[N]` syntax, array literals with `[1, 2, 3, 4]`, read-only `Slice<T>` borrowed views, explicit future `MutSlice<T>`, bounded `FixedBuffer<T, N>` value storage, `Len`, `Capacity`, bounds-checked indexing, stable panic reasons, and the `CON0400`-`CON0410` diagnostic range. No parser, lexer, semantic, MIR, backend, fixture, or DragonGod migration behavior is implemented by P21-M0.


## Phase 20 / DG12 DragonGod Kernel snapshot

Phase 20 is complete through DG12 as a DragonGod Kernel proof-of-concept milestone. The `dragon-god/` tree now covers Core, Automata, Memory, Mind, Decision, Actuation, Events, Trace, Replay, Persistence/checkpoint, and Graph metadata. This is deliberately still a proof-of-concept kernel: production runtime behavior, dynamic machine execution, scheduler/async behavior, durable persistence, parallel staged execution, and DragonGod compiler hooks remain deferred.

| DragonGod Kernel capability | Status | Notes |
|-----------------------------|--------|-------|
| DG0 Blueprint | ✅ | `docs/DragonGod-Kernel-Blueprint.md` is present. |
| DG1 Kernel core seed | ✅ | Module/type seed, examples, fixtures, and friction log added. |
| Memory storage | ✅ | DG3 fixed-slot integer Memory v0 is implemented. |
| Mind v0 static executor | ✅ | DG4 static executor/signal recorder with `Agent` and `World` shells is implemented; dynamic tick loop remains deferred. |
| Automata stack | ✅ | DG5 fixed-capacity Automata stack v0 is implemented. |
| Decision subsystem | ✅ | DG6 fixed-slot integer decision subsystem v0 is implemented. |
| Actuation subsystem | ✅ | DG7 fixed-slot Actuation subsystem v0 is implemented. |
| Events subsystem | ✅ | DG8 fixed-slot integer Events subsystem v0 is implemented. |
| Trace subsystem | ✅ | DG9 fixed-slot explicit Trace subsystem v0 is implemented. |
| Replay subsystem | ✅ | DG10 fixed-slot deterministic Replay v0 is implemented. |
| Persistence/checkpoint | ✅ | DG11 in-memory checkpoint v0 is implemented; file I/O and serialization remain deferred. |
| AutomataGraph metadata | ✅ | DG12 fixed-slot dynamic metadata v0 is implemented; dynamic machine execution remains deferred. |
| Parallel | ❌ | Not implemented. |
| Bare-metal proof | ❌ | Not implemented. |



## Phase 20 / DG12 closeout snapshot

Phase 20 is closed after DG12. DG0 through DG12 are all closed: blueprint, kernel core seed, static automata signals, Memory v0, Mind v0 static executor, Automata stack v0, Decision v0, Actuation v0, Events v0, Trace v0, Replay v0, in-memory Persistence/checkpoint v0, and AutomataGraph metadata v0.

DragonGod.Kernel remains a proof-of-concept kernel. The Phase 20 fixture corpus now has 99 DragonGod `.conception` fixtures under `language/phase20-dragongod-kernel/valid/`, covering Core IDs/Reason/Clock, AutomataSignal, Memory, Mind, AutomataStack, Decision, Actuation, Events, Trace, Replay, Persistence/checkpoint, and Graph metadata.

Key missing features surfaced by Phase 20 are arrays/slices/fixed buffers, a standard Option/Result surface, strings/StringView/byte slices, generic containers and payloads, callable values, stronger dyn/interface storage, and serialization primitives. Further DragonGod subsystem expansion should pause until those Concept features are implemented.

---
## Phase 18 M9 closeout snapshot

Phase 18 is closed after P18-M9. The machine composition substrate now has examples under `examples/phase18/` and 66 fixtures under `language/phase18-machines/`: 43 valid and 23 invalid. The closed Phase 18 substrate pins hierarchical by-value child frames, explicit child stepping, `Complete(child)`, `Result(child)`, numeric `State(machine)`, bool `transition match`, deterministic `transition decide`, shared `cpt_panic` runtime machine failures, and backend-shape assertions that exclude hidden heap allocation, scheduler, async, blackboard, mailbox, event-bus, and DragonGod runtime hooks. Phase 19 is closed after P19-M9 closeout, P19-M8 examples/docs, P19-M7 hardening, and the combined P19-M4-M6 fixture sweep for bare machine-state `yield;` validation, backend/runtime lowering, and interactions with Complete/Result/State, child machines, and transition match/decide. `yield;` now exits the current generated `Step` immediately without state/completion/result mutation, while scheduler, async, blackboard/mailbox/event keywords, dynamic child lists, heap-owned machines, `StateName`, reflection, `yield return`, `suspend`/`resume`, and DragonGod runtime hooks remain deferred.

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Implemented and tested in fixture corpus |
| 🔶 | Partially implemented — scope explicitly constrained by phase plan |
| ❌ | Not yet implemented |
| 🔀 | Implemented differently from PoC3 sketch — see note |

---

## Section 1–4: Thesis, Design Laws, Identity, Naming

| Feature | Status | Notes |
|---------|--------|-------|
| Language thesis and design laws | ✅ | Embodied in AGENTS.md and phase design docs |
| C++-lineage visual shape | ✅ | Return-type-first declarations, braces, semicolons |
| `module` declaration | ✅ | Required at top of every source file, name-checked |
| No textual preprocessor | ✅ | No `#define`, no `#include` anywhere in the pipeline |
| Keyword/language name overlap (`concept`) | ✅ | Intentional, working |

---

## Section 5–6: Visual Shape and Source Files

| Feature | Status | Notes |
|---------|--------|-------|
| C-family brace/semicolon syntax | ✅ | |
| `struct` declarations | ✅ | Fields, struct literals, field access, field address-of |
| `enum` declarations | ✅ | Payload enums with typed fields, unit variants |
| `template` keyword | ✅ | Phase 8 — `template<T>` unconstrained and `template<T: Concept<T>>` constrained |
| `concept` keyword | ✅ | Phase 8 — method-bearing and marker forms |
| `interface` keyword | 🔶 | Phase 14 closed — parser accepts `interface` signature blocks; P14 v0 preserves declarations in HIR, validates non-empty requirement signatures, validates `impl Interface<Type>` conformance, preserves dyn parameter types, supports concrete-to-dyn call coercion, represents dyn method calls explicitly in HIR/MIR, lowers the borrowed dyn subset to C vtables/fat refs, and has examples/fixtures/backend assertions for that subset |
| `impl` blocks | ✅ | Phase 8 — `impl Concept<Type>`, `unsafe impl`, `impl Drop<T>`; Phase 14 closed — separate interface impl conformance plus selected impl vtable/wrapper emission for dyn dispatch |
| `operator` overloading | ❌ | PoC3 §25 — not in any phase plan yet |
| `move` keyword | ✅ | Phase 10 — explicit move expression, use-after-move diagnosed |
| `try` keyword | ✅ | Phase 5 — propagates `Result` error arm |
| `unsafe` blocks and functions | ✅ | Phase 6 — quarantined, raw pointer deref and pointer arith require unsafe |
| Real module system (no headers) | 🔶 | Phase 16 closed for compilation-unit modules v0: harness/driver-supplied multi-file source sets, module declaration table, resolved import graph, module-aware HIR, per-module top-level symbol tables, qualified cross-module functions/types, imported repr(C) metadata, and multi-source MIR/backend/run in one generated C unit; packages, filesystem lookup, visibility, separate compilation, and linker driving remain deferred |
| `import` declarations | ✅ | Phase 16 v0 imports are implemented for qualified access only: parser/AST preservation, ordering diagnostic `CON0273`, resolved graph edges, `CON0271` unknown import, `CON0277` duplicate import, `CON0272` import cycle, qualified function calls, qualified type references, and no unqualified name injection |
| Multi-file fixture support | ✅ | Phase 16 supports hermetic `.conception` source sets with `=== file: <virtual-path> ===` sections and virtual-path preservation |
| Module declaration table | ✅ | Stable `ModuleId` records, source order, virtual paths, duplicate module detection `CON0270`, and missing module declaration `CON0276` |
| Import graph resolution | ✅ | Raw imports are preserved and resolved to module IDs before semantic lowering; unknown, duplicate, and cyclic imports are diagnosed |
| HIR module records | ✅ | HIR stores module records, resolved imports, and per-item module ownership metadata |
| Per-module symbol tables | ✅ | Ordinary duplicate top-level names are scoped per module; same item names across modules are accepted when referenced with qualification |
| Cross-module qualified function calls | ✅ | `Module.Function(...)` resolves for current/imported modules and lowers/runs through MIR/backend |
| Cross-module qualified type references | ✅ | `Module.Type` resolves in type positions for imported structs/enums and current modules |
| Imported repr(C) metadata across modules | ✅ | Imported repr(C) structs remain visible to semantic C ABI validation; imported non-repr structs remain rejected at C ABI boundaries |
| Multi-source backend/run lowering | ✅ | Supported multi-source programs lower to MIR and emit one generated C unit in v0 |
| Backend module name collision hardening | ✅ | Ordinary same-name functions and structs across modules receive collision-safe generated C names; `export "C"` symbols remain exact/unmangled |
| Phase 16 examples/fixtures | ✅ | `examples/phase16/` documents representative virtual-file examples; `language/phase16-imports/` contains 73 fixtures including closeout coverage |
| Import aliases | ❌ | Deferred; no alias syntax in v0 |
| Wildcard imports | ❌ | Deferred; wildcard imports are rejected |
| Re-exports | ❌ | Deferred; imports do not create a public export surface |
| Unqualified imported names | ❌ | Deferred by design; imports do not inject declarations into local scope |
| Module visibility (`public`/`private`) | ❌ | Deferred; all imported top-level declarations are accessible by qualified name in v0 |
| Package manager / cross-package dependencies | ❌ | Deferred; modules are compilation-unit boundaries, not packages |
| Filesystem import search / path-to-file mapping | ❌ | Deferred; source files are supplied by harness/driver rather than discovered from import paths |
| Module spanning multiple files | ❌ | Deferred; each v0 source declares exactly one module and one module does not span files |
| Multiple modules per file | ❌ | Deferred/rejected; multiple `module` declarations use existing parser diagnostic `CON0005` |
| Separate object files / linker driver / incremental compilation | ❌ | Deferred; Phase 16 emits one generated C unit for v0 |
| Conditional compilation `when target.*` | 🔶 | `target.isLittleEndian`, `target.pointerSize` available as comptime queries; full `when target.os` structured conditional compilation not implemented |

---

## Section 7–8: Standard Library Layers and Profiles

| Feature | Status | Notes |
|---------|--------|-------|
| `Core` (freestanding) library layer | ❌ | No stdlib shipped yet; compiler-intrinsic types only |
| `Alloc` library layer | ❌ | Arena types are compiler-intrinsic, not a shipped library |
| `Runtime` layer | ❌ | |
| `OS` layer | ❌ | |
| `Full` hosted layer | ❌ | |
| `profile freestanding` declaration | ❌ | Parser/HIR don't have profile declarations yet |
| `profile kernel` declaration | ❌ | |
| Profile-driven effect defaults | ❌ | Effects are function-level only; no profile defaults |

---

## Section 9: Effects

| Feature | Status | Notes |
|---------|--------|-------|
| `noalloc` effect annotation | ✅ | Phase 12 — function-level, checked transitively at direct call edges |
| `alloc` effect annotation | ✅ | Phase 12 — functions calling `Arena.alloc` carry alloc effect |
| Effect-tagged function types | ✅ | `noalloc` is part of function's HIR representation |
| Transitive effect checking | 🔶 | Direct call-edge `noalloc` violations caught; full transitive closure across all effect types not yet complete |
| `nothrow` effect | ❌ | |
| `nopanic` effect | ❌ | |
| `nostackgrow` effect | ❌ | |
| `nohost` / `freestanding` effect | ❌ | |
| `interrupt` effect | ❌ | |
| `pure` effect | ❌ | |
| `cold` / `hot` hints | ❌ | |
| Effects visible in MIR | ✅ | Allocation effect on functions is MIR-level data |
| Effect violations caught pre-codegen | ✅ | `noalloc` violations are HIR/MIR-stage errors |

---

## Section 10: Memory Model

| Feature | Status | Notes |
|---------|--------|-------|
| Use-before-initialization rejected | ✅ | Phase 10 — MIR storage-state analysis, `Uninitialized` state diagnosed |
| Use-after-move rejected | ✅ | Phase 10 — `Moved` storage state diagnosed |
| Moved values not double-dropped | ✅ | Phase 10 — moved locals skipped in drop cleanup |
| `must_use` checking | ✅ | Phase 5 — `Result` is must_use by default; `discard` keyword for intentional discard |
| Raw pointer deref requires `unsafe` | ✅ | Phase 6 |
| Allocation effects enforced for `noalloc` | ✅ | Phase 12 |
| Destructor/drop order deterministic | ✅ | Phase 10 — reverse initialization order in MIR |
| `immovable` types | ❌ | PoC3 §15 — not in any phase plan yet |
| `moved_state` declaration on structs | ❌ | PoC3 §14 — not yet |
| Null references rejected | ❌ | No reference type system yet; no nullable analysis |
| `nullable` access requires checking | ❌ | |

---

## Section 11: Values, References, and Pointers

| Feature | Status | Notes |
|---------|--------|-------|
| Value types by default | ✅ | Structs and enums are values |
| `T&` non-null read borrow | 🔶 | Reference types parsed and type-checked; no full lifetime/alias checking |
| `mut T&` mutable borrow | 🔶 | Parsed and checked for explicit `mut` requirement |
| `T*` raw nullable pointer | ✅ | Phase 6 — pointer types, deref in unsafe |
| `own T*` owning pointer | ❌ | Parsed by phase 6 but no ownership semantic distinction from `T*` yet |
| Pointer arithmetic requires `unsafe` | ✅ | Phase 6 |
| `span<T>` type | ❌ | Mentioned in PoC3 §11.4; not implemented |
| Copy vs non-copy semantics | ✅ | Phase 10 — `Copy` marker concept, implicit copy for scalars/Copy structs, error for non-Copy |
| Explicit `move` expression | ✅ | Phase 10 |

---

## Section 12: No Region Lifetimes

| Feature | Status | Notes |
|---------|--------|-------|
| No region lifetime system in core | ✅ | Explicitly not implemented per PoC3 §12 — correct by design |
| Explicit allocator passing | ✅ | Phase 12 — `Arena*` passed explicitly, no ambient heap |
| Index-based arena handles | 🔶 | `Arena.alloc<T>` returns raw value; typed `Id<T>` / `Store<T, Id>` pattern deferred from Phase 12 |
| Deterministic arena destruction | ✅ | Phase 12 — `Arena.destroy`, `Arena.reset` implemented |

---

## Section 13: Initialization

| Feature | Status | Notes |
|---------|--------|-------|
| Uninitialized variables rejected in safe code | ✅ | Phase 10 — MIR `Uninitialized` state |
| `ManualInit<T>` for low-level uninitialized storage | ✅ | Phase 10 — `ManualInit<T>`, `unsafe manualAssumeInit(move slot)` |
| Partial initialization tracked in MIR | ✅ | Phase 10 — partial direct-field state tracked |
| Only initialized fields dropped | ✅ | Phase 10 |

> **Implementation difference:** PoC3 §13 sketches `MaybeUninit<T>` as the type name. Phase 10 uses `ManualInit<T>` as the canonical Concept term. `MaybeUninit<T>` is treated as compatibility terminology only.

---

## Section 14–15: Move Semantics and Immovable Types

| Feature | Status | Notes |
|---------|--------|-------|
| Explicit `move` expression | ✅ | Phase 10 |
| Use-after-move is a compile error | ✅ | Phase 10 |
| Moved values are not dropped | ✅ | Phase 10 |
| `moved_state` declaration | ❌ | PoC3 §14 — not yet |
| `immovable` struct qualifier | ❌ | PoC3 §15 — not yet |

---

## Section 16: Drop and RAII

| Feature | Status | Notes |
|---------|--------|-------|
| `impl Drop<T>` intrinsic concept | ✅ | Phase 10 — `Drop` is compiler-intrinsic, `impl Drop<T>` with single `void drop(mut T&)` requirement |
| Drop inserted at MIR level | ✅ | Phase 10 — explicit drop points in MIR cleanup |
| Locals drop in reverse initialization order | ✅ | Phase 10 |
| Fields drop in reverse declaration order | 🔶 | Drop of struct fields via `Drop<T>.drop` called; field-level reverse order partially implemented |
| Moved values not dropped | ✅ | Phase 10 |
| Drop functions may not be comptime | ✅ | Phase 10 — diagnosed as error |
| `discard` keyword for intentional drop of must_use | ✅ | Phase 5 |

> **Implementation difference:** PoC3 §16 uses C++ `~Destructor()` syntax as background, but Concept uses `impl Drop<T>` as an intrinsic concept. This is already PoC3-correct — the C++ syntax is only mentioned as contrast.

---

## Section 17: Errors

| Feature | Status | Notes |
|---------|--------|-------|
| `Result<T, E>` as the error type | ✅ | Phase 5 — compiler-intrinsic type |
| `try` propagation | ✅ | Phase 5 |
| `Result<E>` sugar for `Result<void, E>` | 🔶 | HIR represents it; `return;` from `Result<E>` function partially supported |
| `must_use` on `Result` by default | ✅ | Phase 5 |
| `discard` for intentional error discard | ✅ | Phase 5 |
| Generalizable `must_use` on any type | 🔶 | `must_use` on enums implemented; arbitrary `must_use` on functions deferred |
| Nominal error types preferred | ✅ | Payload enums as error types work fully |
| `panic` / `assert` | ✅ | Phase 17 closed for statement-position `panic("reason")` and `assert(condition, "reason")`: AST/HIR/MIR/backend lowering, bool-only assert conditions, mandatory non-blank runtime failure reasons (`CON0280`-`CON0285`), deterministic exit code 101, backend-owned `cpt_panic`, helper de-duplication, machine `Result(machine)`-before-completion trap migration, and no Core.Test/test-runner dependency for runtime assertions |
| Mandatory runtime failure reasons | ✅ | Missing/wrong panic reasons use `CON0280`, missing/wrong assert reasons use `CON0281`, and empty/whitespace reasons use shared validation with `CON0282` |
| Runtime failure lowering substrate | ✅ | Panic/assert HIR and MIR lower to C through a shared backend helper; failing assert emits a conditional `cpt_panic` call, explicit panic emits a direct call, and migrated machine traps emit the same helper call |
| Deterministic runtime failure exit code | ✅ | Explicit panic, false runtime assert, and machine result-before-completion all exit 101 in hosted C v0 |
| Backend `cpt_panic` helper | ✅ | Backend-owned helper prints `panic: %s\n` to stderr, exits 101, is emitted once per generated C unit when needed, and is shared across panic/assert/migrated traps |
| Core.Test Assert doctrine alignment | ✅ | `Assert.True`/`Assert.False` require bool conditions and non-blank reasons and are classified internally as assertion/invariant failures; `Expect.*` remains expectation-style |
| `panic_handler` for freestanding | ❌ | |
| Panic modes (`abort`, `halt`, `unwind`, `custom`) | ❌ | |

---

## Section 18–19: Payload Enums and Pattern Matching

| Feature | Status | Notes |
|---------|--------|-------|
| Payload enum declarations | ✅ | Phase 1/5 — typed payload fields per variant |
| Unit enum variants | ✅ | |
| `match` expression | ✅ | Phase 2/5 — exhaustive by default |
| Exhaustiveness checking | ✅ | All variants must be covered or `_` wildcard used |
| Payload binding in match arms | ✅ | Phase 5 |
| Wildcard `_` arm | ✅ | |
| Literal matching in match | ✅ | Phase 2 — integer/bool literals |
| Match guards (`when`) | ❌ | PoC3 §19 — guard syntax not yet parsed |
| Struct destructuring in match | ❌ | |
| Binding by reference in match arms | ❌ | |
| `match` as expression (returns value) | 🔶 | `match` can appear in expression position; full expression-match with mixed arm types partially supported |

---

## Section 20–23: Concepts, Generics, Marker Concepts, Implementations

| Feature | Status | Notes |
|---------|--------|-------|
| `concept Foo<T> { ... }` declaration | ✅ | Phase 8 — method-bearing concepts with required signatures |
| `template<T>` unconstrained generics | ✅ | Phase 8 — monomorphized at instantiation |
| `template<T: Concept<T>>` constrained generics | ✅ | Phase 8 — constraint checked at call site |
| `&&` compound constraints | 🔶 | Parsed; checking of multi-constraint combinations partially implemented |
| Generic type inference | ✅ | Phase 8 — infers T from argument types |
| `impl Concept<Type>` blocks | ✅ | Phase 8 — with required method implementations verified |
| `impl Drop<T>` | ✅ | Phase 10 — compiler-intrinsic |
| `marker concept Foo<T>` | ✅ | Phase 8 — no methods, no runtime representation |
| `unsafe marker concept Foo<T>` | ✅ | Phase 8 — requires `unsafe impl` |
| `unsafe impl` for unsafe marker concepts | ✅ | Phase 8 — compiler enforces bidirectional unsafe/safe consistency |
| Safe marker concept rejects `unsafe impl` | ✅ | Phase 8 — CON0108 diagnostic |
| Built-in marker concepts (`Copy`, `Move`, `Trivial`, `Relocatable`, `Pod`) | ✅ | Phase 8 — declared as known marker kinds |
| `derive Copy` etc. | ❌ | Marker concepts declared and usable; derive shorthand not implemented |
| Negative concepts `template<T: !Concept<T>>` | ❌ | PoC3 §22 — not yet |
| Orphan rule for impl coherence | ❌ | Phase 8 impls work in single-module scope; multi-module coherence deferred |
| `bridge` modules | ❌ | PoC3 §23.2 — deferred to multi-module work |
| No SFINAE | ✅ | By design — constraint violations diagnosed at call site |
| No `anytype` style duck typing as primary model | ✅ | By design |

---

## Section 24: Interfaces and Dynamic Dispatch

| Feature | Status | Notes |
|---------|--------|-------|
| `interface Foo { ... }` declaration | 🔶 | Phase 14 closed — P14 v0 preserves top-level interface declarations, requirement signatures, spans, attributes, duplicate/overload checks, duplicate parameter checks, empty-interface rejection, interface-type runtime-use rejection, interface impl conformance, dyn parameter type surfaces, dyn coercion HIR/MIR, dyn method-call HIR/MIR, C backend vtable/fat-ref lowering for used borrowed dyn interfaces, and non-emission for unused interface/impl declarations |
| `impl Interface<Type>` | 🔶 | Phase 14 closed — P14 v0 recognizes interface impls as HIR entities separate from concept impls, checks missing/extra/duplicate methods, receiver convention, return type, and parameter types, uses matching `(interface, concrete type)` impls for dyn coercion lookup, emits selected wrapper thunks plus static vtable constants, and pins wrapper C shape plus vtable reuse |
| `dyn Foo&` dynamic dispatch | 🔶 | Phase 14 closed — P14 v0 implements borrowed `dyn Interface&` / `mut dyn Interface&` type syntax, TypeStore/HIR preservation for function parameters, call-boundary concrete-to-dyn coercion from addressable places with matching interface impls, dyn method calls resolved to interface requirement slots in HIR/MIR, explicit C fat-reference construction, indirect vtable calls, executable dispatch fixtures, examples, and mutability hardening fixtures; owning dyn boxes, dyn returns/fields/locals, RTTI, dynamic cast, interface inheritance, and mutation-through-dyn hardening remain unimplemented |
| `interface` vs `concept` distinction | ✅ | Phase 14 doctrine: concept is static proof/generic constraint; interface is runtime contract represented in HIR and lowered to explicit vtables only through `dyn`; dyn is explicit borrowed dynamic reference |

---

## Section 25: Operator Overloading

| Feature | Status | Notes |
|---------|--------|-------|
| Operator overloading | ❌ | Not in any phase plan yet |

---

## Section 26–27: Allocation and Index-Based Arenas

| Feature | Status | Notes |
|---------|--------|-------|
| Explicit allocator passing (no ambient heap) | ✅ | Phase 12 — `Arena*` parameter pattern |
| `Arena.alloc<T>(arena)` | ✅ | Phase 12 — compiler-intrinsic, type-checked, effect-checked |
| `Arena.reset(arena)` | ✅ | Phase 12 |
| `Arena.destroy(arena)` | ✅ | Phase 12 |
| `Arena.alloc` rejected in `noalloc` context | ✅ | Phase 12 |
| `Arena.alloc` rejected in `comptime` | ✅ | Phase 12 — CON0201 |
| `Arena.alloc` of Drop types | 🔶 | Drop-hardened arena restrictions implemented; Drop-in-arena full semantics deferred from Phase 12 |
| `Arena.create` (owned arena construction) | ❌ | Deferred from Phase 12 |
| `AllocatorLike<T>` concept | ❌ | Phase 12 uses compiler-intrinsic `Arena`/`Allocator` types; generic allocator concept deferred |
| `Id<T>` with index + generation | ❌ | Deferred from Phase 12 |
| Generic `Store<T, Id>` | ❌ | Deferred from Phase 12 |
| Named store IDs | ❌ | Deferred from Phase 12 |
| Generation counters | ❌ | Deferred from Phase 12 |
| C arena helper ABI | ✅ | Phase 12 — stable C helper function signatures emitted |

> **Implementation difference:** PoC3 §27 envisions `Arena` and `Allocator` as library types implementing a concept. Phase 12 implements them as compiler-intrinsic types with special-cased semantics. This is an explicit temporary deviation — the intrinsic path proves the semantics before the generic-concept path is designed.

---

## Section 28: Unsafe

| Feature | Status | Notes |
|---------|--------|-------|
| `unsafe { }` blocks | ✅ | Phase 6 |
| `unsafe fn` declarations | ✅ | Phase 6 |
| Raw pointer dereference requires unsafe | ✅ | Phase 6 |
| Pointer arithmetic requires unsafe | ✅ | Phase 6 |
| `unsafe impl` for unsafe marker concepts | ✅ | Phase 8 |
| Type checking not disabled inside unsafe | ✅ | All normal type rules still apply inside unsafe |
| `manualAssumeInit` requires unsafe | ✅ | Phase 10 |

---

## Section 29: Volatile, Atomics, Barriers, Address Spaces

| Feature | Status | Notes |
|---------|--------|-------|
| `volatileLoad<T>` / `volatileStore<T>` operations | ❌ | Not yet |
| Atomic operations (`atomicStore`, `atomicLoad`, `atomicFetchAdd`) | ❌ | Not yet |
| Memory barriers / fences | ❌ | Not yet |
| Address-space-qualified pointers (`addrspace`, `mmio`) | ❌ | PoC3 §29.1 — provisional/future |

---

## Section 30–31: Layout, ABI, and Bitfields

| Feature | Status | Notes |
|---------|--------|-------|
| `[Repr(C)]` staged layout annotation | ✅ | Phase 15 closed — staged `[Repr(C)]` source spelling for structs is implemented; HIR/debug docs use repr(C) as the representation concept name; keyword `repr(C)` syntax remains reserved/deferred |
| `repr(packed)` annotation | ❌ | Deferred/unsupported after Phase 15 |
| `align(n)` annotation | ❌ | Not yet |
| `static_assert(sizeof(...) == N)` | ❌ | `static_assert` exists for comptime bool expressions; `sizeof`/`alignof` builtins not yet |
| `bits` type for register definitions | ❌ | PoC3 §31 — not yet |

---

## Section 32: Inline Assembly

| Feature | Status | Notes |
|---------|--------|-------|
| `asm { ... }` inline assembly | ❌ | Not yet |
| `asm volatile { ... }` | ❌ | Not yet |

---

## Section 33: Compile-Time Execution

| Feature | Status | Notes |
|---------|--------|-------|
| `comptime` function declarations | ✅ | Phase 9 — scalar `int`/`bool` functions |
| `comptime expr` call-site evaluation | ✅ | Phase 9 |
| `static_assert(expr)` | ✅ | Phase 9 — evaluates comptime bool expression at compile time |
| Hermetic comptime by default | ✅ | Phase 9 — no host-visible side effects without capability grants |
| `comptime(read_fs, env, ...)` capability annotations | ✅ | Phase 9 — syntax parsed and checked; **no capability is actually granted yet** |
| `target.isLittleEndian` / `target.pointerSize` | ✅ | Phase 9 — deterministic target metadata queries |
| Deterministic fuel-bounded `while` in comptime | ✅ | Phase 9 |
| Comptime `if`/`else` | ✅ | Phase 9 |
| Comptime local mutation | ✅ | Phase 9 |
| Comptime type-level functions (returning types) | ❌ | Not in Phase 9 scope — deferred |
| Comptime reflection (`fieldsOf<T>()`) | ❌ | PoC3 §34 — deferred |
| Host-visible comptime capabilities (actual `read_fs` execution) | ❌ | Annotations exist; execution of capability-gated operations not implemented |

> **Implementation difference:** PoC3 §33.1 describes capability-based comptime permissions as a future-gating mechanism. Phase 9 implements the *annotation and checking* of capabilities but deliberately does not grant any capability yet. The "gremlin stays in the jar." This is a stricter interpretation than PoC3's description suggests, and it's the right call.

---

## Section 34–35: Reflection and Macros

| Feature | Status | Notes |
|---------|--------|-------|
| Compile-time reflection (`fieldsOf<T>`) | ❌ | Deferred — depends on comptime type-level functions |
| Hygienic macro system | ❌ | PoC3 §35 explicitly defers this — correct by design |
| No textual preprocessor | ✅ | By design |

---

## Section 36: C and C++ Interop

| Feature | Status | Notes |
|---------|--------|-------|
| C backend (compile Concept to C) | ✅ | Phase 4+ — MIR → C backend, primary bootstrap path |
| C-readable generated output | ✅ | Generated C is auditable and readable |
| `extern "C"` declarations | ✅ | Complete for Phase 15 v0 — block-form declarations parse, lower to HIR with ABI/linkage metadata and C symbol names, validate the strict C ABI subset, participate in semantic call resolution, lower extern calls through MIR, emit backend C prototypes/calls with declared C symbols, and reject duplicate C ABI symbols |
| `[Repr(C)]` struct layout | ✅ | Complete for Phase 15 v0 staged syntax — validates supported fields, rejects empty/unsupported/nested-by-value layouts, accepts validated structs and pointers at extern/export C ABI boundaries, emits named typedef structs before use, preserves source field order, and pins ordering/de-duplication |
| Export to C (`export "C"`) | ✅ | Complete for Phase 15 v0 — function definitions parse/lower as ordinary Concept bodies with C linkage metadata, validate C ABI signatures, reject duplicate C symbols, lower through MIR, and emit unmangled C symbols |
| C ABI-compatible type subset | ✅ | Complete for Phase 15 v0 — `void` returns, `int`, current C-backend `bool`/`AllocError` spelling, supported raw pointers, `Arena*`, `Allocator*`, `AllocError*`, validated `[Repr(C)]` structs by value, and pointers to validated `[Repr(C)]` structs; unsupported ABI types are rejected rather than guessed |
| C ABI duplicate symbol checking | ✅ | Phase 15 rejects duplicate extern/export C linkage symbols with pinned diagnostics and fixture coverage |
| C headers/includes | ❌ | Deferred/unsupported — `extern "C"` does not parse headers and backend C does not emit user `#include` lines |
| Automatic linking/linker driver | ❌ | Deferred/unsupported — `export "C"` and `extern "C"` do not imply library selection, linker flags, or a linker driver |
| Varargs | ❌ | Deferred/unsupported; parser rejects `...` in extern C declarations |
| Extern variables | ❌ | Deferred/unsupported |
| C++ interop (`extern "C++"`) | ❌ | PoC3 §36.2 — quarantined/future; unsupported ABI strings are rejected |

---

## Section 37: First-Class State Machines

| Feature | Status | Notes |
|---------|--------|-------|
| `machine Name(params) -> ReturnType { }` declaration | ✅ | Phase 13 |
| `state StateName { }` blocks | ✅ | Phase 13 |
| `transition StateName` literal transitions | ✅ | Phase 13 — runnable, C backend lowered |
| `transition match { ... }` transitions | ✅ | Phase 18 closed with executable runtime bool subset: bool scrutinees, `true`/`false`/`_` labels, deterministic branching, exhaustive/default validation, and shared-panic no-match reason reserved for future wider match subsets |
| `transition decide { ... }` transitions | ✅ | Phase 18 closed with deterministic v0 runtime lowering: optional bool guards, required int scores, missing `when` enabled, disabled candidates ignored, highest score wins, strict `>` preserves source-order ties, and no-enabled candidates call shared `cpt_panic` |
| `Step(machine)` as statement | ✅ | Phase 13 — statement-like, produces no value |
| `Complete(machine)` returning bool | ✅ | Phase 13 |
| `Result(machine)` returning result type | ✅ | Phase 13; Phase 17/18 route before-completion reads through shared `cpt_panic` with stable reason |
| `MachineName(args)` construction | ✅ | Phase 13 |
| `noalloc machine` effect checking | ✅ | Phase 13 — `noalloc` effect on machine declaration enforced |
| `yield` statement in machines | ✅ | Phase 19 closed: bare `yield;` is supported only in machine state bodies, including nested blocks and `if` branches; generated machine `Step` code returns immediately, preserves state, leaves the machine incomplete, writes no result, avoids panic/allocation/scheduler/async hooks, re-enters the same state body on the next `Step`, and is pinned by 23 validation/runtime/backend fixtures |
| Nested machine fields / child frames | ✅ | Phase 18 closed for zero-parameter child machine fields stored by value in parent frames, parent-constructor child initialization, and explicit parent `Step(child)` / `Complete(child)` / `Result(child)` operations |
| `State(machine) -> int` | ✅ | Phase 18 closed for local and nested child machine numeric state introspection; no `StateName(machine)`, reflection metadata, or state-name runtime surface |
| Machine lowers to explicit state struct in MIR | ✅ | Phase 13 — state enum and struct visible in MIR |
| Machine lowering visible in MIR (not hidden) | ✅ | Phase 13 |
| References crossing yield restricted | ❌ | Deferred with continuation yield, lifted locals, and sub-state program counters; Phase 19 state-level yield has no resume-after-yield continuation |
| Immovable machine frames | ❌ | Depends on `immovable` being implemented |
| Scalar `int`/`bool` params and results only (Phase 13 v0 constraint) | ✅ | Phase 13 — non-scalar params/results produce clear error |

> **Implementation difference:** PoC3 §37 presents `transition decide` as a core machine primitive alongside `transition`. Phase 13 implemented decide-transition *parse and HIR representation* and P18-M5 added runtime lowering for the deterministic bool-guard/int-score subset. The standalone `decide` expression (Phase 5a) remains a separate fully runnable expression feature.

---

## Section 38–39: SoA Layout and Audit Instrumentation

| Feature | Status | Notes |
|---------|--------|-------|
| `repr(SoA)` / `SoA<T>` container | ❌ | PoC3 §38 — provisional/experimental |
| Audit instrumentation profiles | ❌ | PoC3 §39 — future |
| Poison-on-drop / use-after-move traps | ❌ | Not yet |

---

## Section 40: Compiler Architecture

| Feature | Status | Notes |
|---------|--------|-------|
| AST layer | ✅ | Phase 1 — full AST with spans |
| Typed HIR layer | ✅ | Phase 3 — separate from AST, stable IDs, type store |
| Initialization / ownership / effect analysis in HIR/MIR | ✅ | Phases 10, 12 |
| MIR with basic blocks | ✅ | Phase 4 — explicit basic blocks, terminators |
| MIR: control flow | ✅ | |
| MIR: moves and drop points | ✅ | Phase 10 |
| MIR: enum discriminants and match lowering | ✅ | Phase 5 |
| MIR: error propagation (`try`) | ✅ | Phase 5 |
| MIR: allocation sites | ✅ | Phase 12 |
| MIR: unsafe regions | ✅ | Phase 6 |
| MIR: effect boundaries | ✅ | Phase 12 (noalloc) |
| MIR: machine states and transitions | ✅ | Phase 13 |
| MIR: yield points | ✅ | Phase 19 lowers HIR `yield_stmt` through machine-state step lowering as direct step return; no continuation program counter or coroutine MIR |
| MIR: partial initialization | ✅ | Phase 10 |
| MIR validator | ✅ | `mir_validator.zig` — enforces MIR invariants before codegen |
| C backend from MIR | ✅ | Phase 4+ — `backend_c_mir.zig` |
| MIR storage (arena-based) | ✅ | `mir_storage.zig` |
| LLVM backend | ❌ | Stage 3 goal |
| Custom native backends (x86_64, AArch64, RISC-V) | ❌ | Stage 3 goal |

---

## Section 41: Bringup Roadmap Stage Status

| Stage | Goal | Status |
|-------|------|--------|
| **Stage 0** | Zig frontend, C backend, basic type checker, test runner | ✅ **Complete** |
| **Stage 1** | Typed HIR→MIR, drop insertion, move checking, use-before-init, must_use, effect checking, match lowering, C backend from MIR | 🔶 **Substantially complete** — see gaps below |
| **Stage 2** | Self-hosting — rewrite compiler in Concept | ❌ Future |
| **Stage 3** | LLVM backend, native backends, optimization | ❌ Future |

**Stage 1 remaining gaps (blocking full Stage 1 completion):**
- remaining `interface` / `dyn` work beyond the borrowed C backend subset:
  owning dyn boxes, dyn returns/fields/locals, RTTI/dynamic cast decisions,
  upcasting/inheritance non-goals, and ABI policy
- repr(C) enums, nested by-value repr(C) fields, packed layout, custom alignment, bitfields, and platform ABI matrices beyond the closed Phase 15 v0 subset
- C headers/includes, automatic linking/linker driver, symbol aliasing, callbacks/function pointers, extern variables, and C++ ABI interop beyond the closed Phase 15 v0 subset
- broader module features beyond Phase 16 v0: visibility, aliases, wildcard imports, re-exports, packages, filesystem lookup/path mapping, module spanning multiple files, separate object files, linker driving, incremental compilation, and cross-package dependency resolution
- `yield` in machines
- Full `must_use` on arbitrary functions (not just enums)
- `panic` / `assert` infrastructure

---

## First-Class Testing (Phase 11)

| Feature | Status | Notes |
|---------|--------|-------|
| `.con_test` test files | ✅ | Phase 11 — separate test file type |
| `[Fact]` attribute | ✅ | Zero-argument test functions |
| `[Theory]` + `[InlineData]` | ✅ | Parameterized tests with inline row data |
| `Assert.True` / `Assert.False` with reason | ✅ | Mandatory reason strings |
| `Expect.Equal` for bool/int | ✅ | Type-checked equality |
| `Expect.That` relation primitive | ✅ | |
| HIR Fact/Theory runner | ✅ | Test discovery and execution |
| Stable test output format | ✅ | |
| Test attributes rejected in normal source | ✅ | `testAttributeOutsideTestFile` diagnosed |
| Named `because:` parameter | ❌ | Deferred from Phase 11 |
| CLI from-disk test runner | ❌ | Deferred |
| Generic `TestRelation<T>` | ❌ | Deferred |
| Custom test relations | ❌ | Deferred |
| Test filtering / hooks | ❌ | Deferred |

---

## Project Manifest (Phase 11 partial)

| Feature | Status | Notes |
|---------|--------|-------|
| `*.header.conception` manifest format | 🔶 | `manifest.zig` parses `concept-manifest-v0` format with kind/format/section structure; full build graph, target declarations, and dependency graph not yet implemented |
| Manifest as restricted declarative subset (not full code) | ✅ | `manifest.zig` parses a constrained key/value/section format, not arbitrary Concept source |

---

## Summary: Coverage by PoC3 Section

| PoC3 Sections | Coverage |
|--------------|----------|
| §1–4 Thesis, laws, identity | ✅ Complete |
| §5–6 Syntax and modules | ✅ Core complete for Phase 16 v0 imports/multi-module compilation-unit modules; packages, visibility, filesystem lookup, separate compilation/linking, broader interface object model, and `operator` remain deferred |
| §7–8 Stdlib layers and profiles | ❌ Not started |
| §9 Effects | 🔶 `noalloc`/`alloc` done; 8 other effects deferred |
| §10 Memory model | 🔶 Core guarantees done; `immovable`, `nullable`, `moved_state` deferred |
| §11 Values, refs, pointers | 🔶 Values/Copy/move/raw pointers done; `span`, `own`, full ref analysis deferred |
| §12 No region lifetimes | ✅ Correctly not implemented |
| §13 Initialization | ✅ Complete including `ManualInit<T>` |
| §14–15 Move and immovable | 🔶 Move done; `immovable` and `moved_state` deferred |
| §16 Drop/RAII | ✅ Substantially complete |
| §17 Errors | ✅ Core Stage 1 surface complete for `Result`, `try`, `must_use`, `discard`, statement-position `panic("reason")`, statement-position `assert(condition, "reason")`, shared runtime failure reasons, deterministic exit 101, and migrated machine result-before-completion trap; richer panic modes and freestanding hooks remain deferred |
| §18–19 Enums and match | 🔶 Core done; guards, struct destructure, ref binding deferred |
| §20–23 Concepts/generics/impl | 🔶 Core done; `derive`, negative concepts, orphan rule, bridge modules deferred |
| §24 Interfaces/dyn | 🔶 Phase 14 closed for borrowed dyn dispatch v0: declarations, impl conformance, borrowed dyn parameter types, concrete-to-dyn call coercion, dyn method-call HIR/MIR, C vtable/fat-reference lowering, executable borrowed dyn dispatch, examples, mutability hardening fixtures, and backend C-shape assertions; owning boxes, RTTI/dynamic cast, inheritance/upcast, dyn returns/fields/locals, mutation-through-dyn hardening, and ABI stability deferred |
| §25 Operator overloading | ❌ Not started |
| §26–27 Allocation and arenas | 🔶 Core done; `Id<T>`, `Store`, `Arena.create`, generic allocator deferred |
| §28 Unsafe | ✅ Complete |
| §29 Volatile/atomics/barriers | ❌ Not started |
| §30–31 Layout/ABI/bitfields | 🔶 Phase 15 closes the validated `[Repr(C)]` struct subset with examples and representative fixtures; repr(C) enums, nested by-value repr(C) fields, packed layout, custom alignment, and bitfields remain deferred |
| §32 Inline assembly | ❌ Not started |
| §33 Comptime | 🔶 Scalar hermetic comptime done; type-level comptime, capability execution deferred |
| §34–35 Reflection/macros | ❌ Correctly deferred per PoC3 |
| §36 C interop | ✅ Phase 15 v0 complete for single-compilation-unit C ABI: `extern "C"` declarations/calls, `export "C"` definitions, validated `[Repr(C)]` structs by value/pointer, strict ABI diagnostics, duplicate C symbol rejection, extern prototype/call emission, unmangled exported C definitions, no generated includes, no C++ text, examples, and invalid coverage for deferred non-goals |
| §37 State machines | ✅ Phase 19 closed bare machine-state `yield;` runtime interactions after P19-M3 lowering; continuation yield remains deferred; Phase 18 closed for the explicit machine substrate: local frames, by-value zero-parameter nested child fields, explicit `Step`/`Complete`/`Result`, runtime bool `transition match`, deterministic int-score/bool-guard `transition decide`, numeric `State(machine)`, shared-panic hardening, examples, and 66 Phase 18 fixtures; schedulers, async, dynamic child lists, heap-owned machines, parameterized child initialization, enum/int runtime match, `StateName`, reflection, and DragonGod runtime hooks remain deferred |
| §38–39 SoA/audit | ❌ Provisional/future |
| §40 Compiler architecture | 🔶 Core pipeline done; LLVM/native backends are Stage 3 |
| §41 Bringup roadmap | 🔶 Stage 0 complete; Stage 1 ~75% |

---

## Key Implementation Differences vs PoC3 Sketches

1. **`ManualInit<T>` not `MaybeUninit<T>`** — Phase 10 uses `ManualInit<T>` as the canonical name. PoC3 uses `MaybeUninit<T>` in examples, which is treated as compatibility terminology only.

2. **`Arena` and `Allocator` as compiler intrinsics, not library types** — Phase 12 special-cases these rather than implementing a generic `AllocatorLike<T>` concept first. Explicit temporary deviation pending the generic-concept path.

3. **`decide` as a standalone top-level expression (Phase 5a), not only inside machine transitions** — PoC3 §37.6 says utility scoring is "not core" and belongs in libraries. Concept implemented `decide` as a first-class language expression outside machines as a distinct phase 5a feature. Machine-internal `transition decide` now has the P18-M5 deterministic transition subset. These are separate features.

4. **Capability annotations present but inactive** — Phase 9 capability syntax (`comptime(read_fs, env)`) is parsed and validated but no capability grants are actually executable yet. This is stricter than PoC3 implies.

5. **Testing as `.con_test` files with `[Fact]`/`[Theory]` C#-influenced attributes** — PoC3 §11 leaves the testing model unspecified. Phase 11 chose a xUnit-inspired design with mandatory reason strings, which is not in PoC3.

6. **`transition match` and `transition decide` inside machines moved from clear backend failures to executable subsets** — Phase 13 deliberately rejected unsupported runtime lowering; P18-M4 and P18-M5 now lower the bool match subset and deterministic bool-guard/int-score decide subset.

7. **Manifest format as `concept-manifest-v0` key/value sections** — PoC3 §50 mentions a build manifest as a future document. Phase 11 partially implements a `manifest.zig` parser for a restricted key/value format, not the full declarative subset described in conversations.

---

*This matrix began as the Phase 13 closeout snapshot and has been updated
through Phase 18 closeout. Stage 1 is substantially implemented. Phase 14
closed runtime interfaces and borrowed dyn dispatch v0. Phase 15 closed the
single-compilation-unit C ABI v0 surface. Phase 16 closes imports and
multi-module compilation-unit modules v0: harness/driver-supplied multi-file
source sets, module table, import graph, module-aware HIR, per-module ordinary
symbol tables, qualified cross-module functions/types, imported repr(C)
metadata, and multi-source MIR/backend/run in one generated C unit. Remaining
Stage 1 gaps include visibility and package/filesystem
module resolution, separate compilation/linker driving, reference/receiver hardening, and broader ABI/layout
features deliberately deferred beyond Phase 15 (repr(C) enums, nested by-value
struct layout, packed/custom alignment, bitfields, headers/includes, automatic
linking, C++ ABI, varargs, extern variables, symbol aliasing, callbacks, and
platform ABI matrices). Phase 17 closes the basic runtime failure substrate, so basic panic, basic runtime assert, shared `cpt_panic`, machine Result-before-completion trap routing, nested machine fields, runtime bool transition match, runtime deterministic transition decide, and numeric `State(machine)` are no longer Stage 1 gaps.*


## Phase 17 M9 closeout update

P17-M9 closes Phase 17. P17-M8 added human-readable examples under `examples/phase17/` and representative fixtures for panic/assert/machine runtime failure behavior. The M8 fixtures pin exit code 101 for panic, false assert, and machine `Result(machine)` before completion; exit 0/ordinary result behavior for true assert and result-after-completion; one-per-C-unit `cpt_panic` helper sharing across panic/assert/migrated machine trap sites; absence of the old `cpt_machine_result_before_complete` helper and `__builtin_trap` at the migrated site; C-escaped reason strings; blank reason rejection; expression-position rejection; bool-only assert conditions; and ordinary runtime assert independence from Core.Test/test-runner symbols. The language fixture corpus now contains 1046 fixture files, including 57 under `language/phase17-runtime-failure/` and 11 under `language/phase18-machines/`.


## Phase 18 M1 update

P18-M1 hardens the machine frame/value baseline before nested fields. It adds 11 fixtures under `language/phase18-machines/` covering local frame construction, explicit stepping, completion/result stability, independent local instances, result-before-completion exit code 101, backend frame shape, shared `cpt_panic` routing, current by-value copy/assignment behavior, and invalid `Step`/`Complete`/`Result` operands. `Step` now requires an assignable machine place and reports `CON0291`; non-machine `Complete` and `Result` report `CON0292` and `CON0293`. Nested machine fields, child initialization, executable transition match, and executable transition decide remain future Phase 18 milestones.

### Phase 18 M2 coverage

Nested machine field coverage now includes HIR preservation, backend by-value child frame fields, deterministic parent-constructor initialization of zero-parameter children, rejection of parameterized child fields with `CON0290`, duplicate child field diagnostics through existing field-name checks, and no hidden heap/scheduler/async backend markers.

### Phase 18 M3 nested machine operation coverage

P18-M3 covers explicit nested machine composition for child fields: `Step(child)` mutates the child frame stored in the parent, `Complete(child)` reads that child completion flag, and `Result(child)` uses the shared result-before-completion panic path. Fixtures also prove that children are not implicitly stepped, multiple child fields compose independently, and runtime `transition match` and `transition decide` are implemented in later P18-M4/P18-M5 subsets.

P18-M4 updates `transition match { ... }` coverage from parsed/HIR-only scaffold to executable runtime lowering for the bool subset. Bool scrutinees, including `Complete(child)`, lower through machine step emission to deterministic C `if`/`else` state assignment with exhaustive true/false validation or wildcard default coverage. Non-bool runtime match and `transition decide` remain future work.

P18-M5 updates `transition decide { ... }` coverage from parsed/HIR-only scaffold to executable deterministic utility selection. The v0 subset supports ordered candidates with optional bool guards and required int scores, evaluates candidates in source order, evaluates scores only for enabled candidates, uses strict `>` so equal scores preserve the earlier candidate, assigns the selected state through the machine step backend path, and calls shared `cpt_panic("machine decision transition has no enabled candidates")` with exit code 101 when no candidate is enabled. Diagnostics now cover empty decides (`CON0299`), non-bool guards (`CON0297`), non-int scores (`CON0298`), and existing unknown target-state errors; fixtures include nested-machine composition via `Complete(child)` and `Result(child)`. Yield, schedulers, async/event buses, blackboards/mailboxes, floating-point scores, randomness, weighted selection, behavior-tree runtimes, GOAP/planner runtimes, and DragonGod runtime hooks remain out of scope.

P18-M6 extends machine coverage with `State(machine) -> int`, a non-reflective helper that reads the current backend/compiler numeric state id from a machine frame. Coverage includes local frames, nested child fields, before-completion reads with no panic requirement, after-step reads, backend direct `.state` emission, and `CON029D` rejection for non-machine operands. The helper deliberately does not expose string state names, reflection metadata, state lists, a source-level state enum surface, scheduler/yield/async/event-bus behavior, blackboards/mailboxes, dynamic child lists, heap-owned machines, or DragonGod runtime hooks. Remaining P18 scope is runtime failure hardening, examples/fixtures, and closeout.

### Phase 18 M7 runtime failure hardening coverage

P18-M7 hardens machine runtime failure coverage. The corpus now contains 1096 fixture files, including 60 under `language/phase18-machines/`. Backend fixtures pin `cpt_panic("machine result cannot be read before completion")` for local and nested child result-before-completion guards, `cpt_panic("machine decision transition has no enabled candidates")` for no-enabled `transition decide`, one-per-C-unit `cpt_panic` helper emission across multiple failure sites, and the defensive machine step default `cpt_panic("invalid machine state reached")`. Bool v0 `transition match` still relies on exhaustive static validation, so there is no ordinary runtime no-case fixture; the stable reason `machine transition match found no matching case` is documented for any future runtime no-match path. Static invalid machine programs remain diagnostics rather than runtime panics.


### Phase 19 M9 closeout for yielding machines and explicit suspension

P19-M9 closes Phase 19 after P19-M8 added examples/docs for the existing bare `yield;` implementation without adding semantics. The Phase 19 examples under `examples/phase19/` cover wait-until child completion, caller-controlled polling, long-running tickable behavior, yield before runtime `transition match`, yield before deterministic `transition decide`, `State(machine)` observation after yield, runtime failure notes, and DragonGod readiness. The fixture set remains twenty-three fixtures under `language/phase19-yielding-machines/`: ten parse/check validation fixtures, nine runtime fixtures, and four backend C fixtures. Runtime coverage proves that `yield;` leaves `Complete(machine)` false, preserves `State(machine)`, re-enters the same state body from the beginning on the next `Step`, works as a terminal inside an `if` branch, keeps `Result(machine)` after yield on the existing shared `cpt_panic` result-before-completion path with exit code 101, allows nested child machines to advance across parent yields, leaves `State(child)` readable after parent yield, supports repeated external polling, and composes with later non-yielding `transition match` and `transition decide` paths. Backend coverage now separately pins the yield-only path as direct `return;` with no `cpt_panic`, yield-specific panic reason, helper, state write, completion write, result write, allocation, scheduler/async/event/mailbox/blackboard marker, suspend/resume marker, or DragonGod hook; it also pins `Result(machine)` after yield to the exact existing reason `machine result cannot be read before completion`. Unsupported forms remain stable diagnostics or parser errors, compile-time/test-runner yield execution remains unsupported, and unreachable-after-yield diagnostics remain deferred. Phase 19 is closed as a DragonGod-ready yield substrate for tickable machines, not as coroutine, async, scheduler, generator, event-loop, blackboard/mailbox, dynamic-child, heap-owned-machine, or DragonGod runtime support.

### DragonGod DG2 coverage update

- DG0 Blueprint: complete.
- DG1 Kernel core seed: complete.
- DG2 Static automata signals and machine examples: complete.
- Fixture corpus count: 1068 total `.conception` files; 16 Phase 20 DragonGod valid fixtures.
- Covered: machines returning `AutomataSignal`; `Step`/`Complete`/`Result` for signal-returning machines; payload construction/matching for `Succeed`, `Fail`, `Goto`, `Act`, and `AwaitActuation`; `yield;` before eventual signal completion; runtime `transition match` and deterministic `transition decide` before signal return; backend C essentials for signal machine result slots and yield direct return.
- Deferred: Mind tick loop, Automata stack, dynamic graph, Memory storage, Actuation subsystem, Events, Trace, Replay, Persistence, parallel execution, scheduler/async, and DragonGod compiler hooks.


### DragonGod DG3 coverage update

- DG0 Blueprint: complete.
- DG1 Kernel core seed: complete.
- DG2 Static automata signals and machine examples: complete.
- DG3 Memory v0: complete.
- Fixture corpus count: 1075 total `.conception` files; 23 Phase 20 DragonGod valid fixtures.
- Supported DG3 Memory surface: fixed four-slot `Memory` storage, `MemoryKeyInt`, `memoryWrite`, `memoryRead(memory, key, fallback)`, `memoryHas`, `memoryRevision`, and revision increments on every successful write including existing-key updates.
- Backend coverage: Memory struct has revision and slots, `memoryWrite` mutates through an explicit mutable reference, `memoryRead` preserves a fallback path, and generated/source fixture text avoids heap allocation, scheduler/async/event/mailbox hooks, and Blackboard/BbKey public naming.
- Deferred after DG3: generic typed `MemoryKey<T>`, arbitrary value types, type erasure, hash maps, string-key maps, TTL expiration, changed-key tracking/filtering, arena allocation, serialization, Mind tick loop, Automata stack, Decision/Actuation/Events/Trace/Replay/Persistence subsystems, parallel staged tick, scheduler/async, and DragonGod compiler hooks.

### Phase 20 / DragonGod DG4 Mind v0 static executor

DG4 is complete. DragonGod coverage now includes Mind v0 as a static executor/signal recorder: `MindStatus`, `Mind` status/signal/reason fields, `mindEmpty`, `mindRecordSignal`, `mindIsDone`, and `mindLastReasonCode`. Supported static paths include recording `AutomataSignal::None` as no completion, `Succeed` as terminal success, `Fail` as terminal failure, `Goto` as recorded/deferred intent without stack execution, and static supervision of concrete machines through caller-owned `Step`/`Complete`/`Result` calls. Memory-driven static signal machines compose with DG3 Memory v0.

Feature matrix after DG4: DG0 Blueprint complete; DG1 Kernel core seed complete; DG2 static automata signals and machine examples complete; DG3 Memory v0 complete; DG4 Mind v0 static executor complete. Deferred after DG4: Automata stack, dynamic graph, `Goto`/`Push`/`Pop` execution, root frame semantics, transition/interrupt scanning, decision memory, type-erased machine storage, Actuation subsystem, Events, Trace, Replay, Persistence, and Parallel staged tick.


### Phase 20 / DragonGod DG5 Automata stack v0

DG5 is complete. DragonGod coverage now includes `AutomataFrame`, fixed-capacity `AutomataStack`, push/pop/top/replace/depth/empty/full helpers, `Mind.stack`, and structural `mindRecordSignal` handling for `Goto`, `Push`, and `Pop`. `Goto` pushes the target frame when no frame exists and replaces the current top frame otherwise; `Push` adds a frame; `Pop` removes one frame and returns Mind to `Idle` only when the stack becomes empty. Reason propagation is preserved through frame storage and `Mind.lastReason`.

The DG5 fixture corpus adds six run fixtures and one backend fixture, bringing Phase 20 DragonGod coverage to 37 valid fixtures and the full `.conception` corpus to 1089 files. Panic-backed stack overflow, underflow, and empty-top paths lower through the shared `cpt_panic` runtime failure helper with DragonGod-specific reason strings.

Feature matrix: DG0 Blueprint complete; DG1 Kernel core seed complete; DG2 static automata signals and machine examples complete; DG3 Memory v0 complete; DG4 Mind v0 static executor complete; DG5 Automata stack v0 complete. Supported after DG5: `AutomataFrame`; fixed-capacity `AutomataStack`; push/pop/top/replace helpers; Mind stack field; structural Mind handling of `Goto`/`Push`/`Pop`. Deferred: dynamic graph, type-erased machine storage, `AutomataMachineOps`, root-frame semantics, keep-root behavior, transition scanning, interrupt scanning, decision memory, actuation dispatch, events, trace, replay, persistence, parallel staged ticking, scheduler/async behavior, and DragonGod compiler hooks.

### Phase 20 / DG6 DragonGod Decision subsystem v0

DG6 is complete. DragonGod coverage now includes `DragonGod.Kernel.Decision` with integer `DecisionPolicy`, `DecisionMemory`, `UtilityOption`, and deterministic `decisionChoose4`. Supported behavior: highest score wins, disabled options are ignored, source-order tie preference, explicit fallback when no option is enabled, first-selection memory recording, same-option current retention with score refresh, min-commit switch blocking, hysteresis delta switch blocking, tie-epsilon current preference, switching after policy gates allow it, DG3 Memory integration, DG5 Mind/Goto stack integration, and a doctrine fixture proving native `transition decide` remains in use for local stateless machine decisions. Deferred: floating scores, probabilistic/random weighted selection, generic/dynamic option lists, function-pointer considerations, full consideration algebra, changed-key filtering, dynamic AutomatonGraph, transition scanning, actuation/events/trace/replay/persistence/parallel execution, scheduler/async, and DragonGod compiler hooks.

### Phase 20 / DG7 DragonGod Actuation subsystem v0

DG7 is complete. DragonGod coverage now includes `DragonGod.Kernel.Actuation` with `ActuationStatus`, `ActuationDecision`, `ActuationDispatchResult`, `ActuationSlot`, and a fixed four-slot `ActuatorHost`. Supported behavior includes deterministic id allocation, allow dispatch into a `Pending` slot, deny dispatch as a completed not-ok result without slot occupancy, `actuatorHas`, `actuatorStatus`, `actuatorComplete`, `actuatorFail`, and status-code `match` coverage. `AutomataSignal::Act` and `AutomataSignal::AwaitActuation` smoke fixtures prove payload construction/matching, and a Mind fixture proves explicit recording remains separate from host dispatch.

The DG7 fixture corpus adds eight run fixtures and one backend-C fixture, bringing Phase 20 DragonGod coverage to 60 `.conception` fixtures and the full `.conception` corpus to 1112 files. Backend coverage pins fixed host slots, `nextId` mutation, status updates, and absence of heap allocation, scheduler/async markers, event bus, mailbox/blackboard naming, dynamic handler tables, and DragonGod compiler hooks.

Feature matrix: DG0 Blueprint complete; DG1 Kernel core seed complete; DG2 static automata signals and machine examples complete; DG3 Memory v0 complete; DG4 Mind v0 static executor complete; DG5 Automata stack v0 complete; DG6 Decision subsystem v0 complete; DG7 Actuation subsystem v0 complete. Supported after DG7: `ActuationStatus`; `ActuationDecision`; `ActuationDispatchResult`; fixed-slot `ActuatorHost`; allow/deny dispatch; complete/fail status update; status query; Act/AwaitActuation signal smoke coverage. Deferred: command payload generics; `ActuationCommand` marker concept surface; handler registry; policy chains; dynamic dispatch; pending completion queues; actual external command execution; async/scheduler integration; Replay/Persistence/Parallel; and DragonGod compiler hooks.

### Phase 20 / DG8 DragonGod Events subsystem v0

DG8 is complete. DragonGod coverage now includes `DragonGod.Kernel.Events` with `EventTypeId`, `EventId`, `EventPayloadInt`, `EventSlot`, `EventCursor`, and fixed-slot `EventBus`. Supported behavior includes deterministic id allocation, fixed insertion-order append, capacity panic via `DragonGod.Events capacity exceeded`, cursor-start reads, future-only `eventTailCursor`, fallback reads for missing events, stream cursor advancement to the consumed index, missing-event advancement to `bus.count`, deterministic same-type order, and documented multi-type stream cursor semantics.

The DG8 fixture corpus adds eight run fixtures and one backend-C fixture, bringing Phase 20 DragonGod coverage to 69 `.conception` fixtures. Backend coverage pins fixed event slots, `nextId`/`count` mutation, `eventReadNextInt` cursor writes, and absence of heap allocation, scheduler/async markers, mailbox/blackboard naming, event-bus language runtime naming, and DragonGod compiler hooks.

Feature matrix: DG0 Blueprint complete; DG1 Kernel core seed complete; DG2 static automata signals and machine examples complete; DG3 Memory v0 complete; DG4 Mind v0 static executor complete; DG5 Automata stack v0 complete; DG6 Decision subsystem v0 complete; DG7 Actuation subsystem v0 complete; DG8 Events subsystem v0 complete. Supported after DG8: integer event type ids, integer event ids, integer payloads, fixed event slots, stream cursors, push, tail cursor, read-next with fallback, cursor advancement, future-only polling, deterministic order, Memory integration, and AutomataSignal machine integration. Deferred: generic typed payloads, type-erased payloads, dynamic event type registry, event bucket maps, trimming, cross-agent mailbox delivery, parallel staged event merge, replay integration, persistence integration, scheduler/async behavior, event/mailbox language keywords, and DragonGod compiler hooks.


### Phase 20 / DragonGod DG9 Trace subsystem v0

DG9 is complete. DragonGod coverage now includes `DragonGod.Kernel.Trace` with `TraceEventKind`, `TraceEvent`, fixed-slot `TraceRecorder`, and explicit helper calls for `traceEnter`, `traceExit`, `traceTransition`, `traceYield`, and `traceSignal`. `TraceEvent` stores `AutomataSignal` directly, and interpretation helpers use `match` for event-kind and signal-reason handling. The `TraceSink` interface/dyn shape is smoke-tested and supported for explicit calls, while `TraceRecorder` remains the v0 implementation surface.

Feature matrix after DG9: DG0 Blueprint complete; DG1 Kernel core seed complete; DG2 static automata signals and machine examples complete; DG3 Memory v0 complete; DG4 Mind v0 static executor complete; DG5 Automata stack v0 complete; DG6 Decision subsystem v0 complete; DG7 Actuation subsystem v0 complete; DG8 Events subsystem v0 complete; DG9 Trace subsystem v0 complete. Supported DG9 surface: `TraceEventKind`, `TraceEvent`, fixed-slot `TraceRecorder`, `traceEnter`, `traceExit`, `traceTransition`, `traceYield`, `traceSignal`, and explicit trace examples. Deferred after DG9: automatic trace injection, runtime logging backend, file/stdout sink, async sink, trace persistence/replay integration, parallel trace merge, dynamic sink registry, and DragonGod compiler hooks.

DG10 adds DragonGod Replay v0. Phase 20 DragonGod coverage now contains 83 valid fixtures. Replay coverage includes `ReplayEvent` v0, fixed-slot `ReplayLog`, value-shaped `ReplayDriver`, `replayPush`, `replayApplyNext`, `replayApplyAll`, deterministic cursor application, Memory integration, Events integration, Actuation integration, Agent memory integration, RNG seed metadata, and Trace marker integration. Deferred: persistence/checkpoint, file I/O, JSON, binary serialization, dynamic/string payloads, replay loading/saving, parallel deterministic merge, external input adapters, schedulers, async, and DragonGod compiler hooks.


### DG12 Dynamic AutomataGraph v0

Status: complete. Added fixed-slot `AutomatonGraph` metadata (`AutomatonNodeId`, `AutomatonNodeKind`, `AutomatonNode`) with add/find/root helpers, graph-to-Mind root `Goto`, Decision and Trace composition fixtures, and backend shape coverage. Deferred: type-erased machine execution, dynamic MachineOps, heap graph storage, dynamic transition scanning, graph persistence serialization, scheduler/async, and parallel execution. Phase 20 DragonGod fixture count increased by seven DG12 fixtures (six run, one backend C).
## DragonGod DG11 coverage

Persistence/checkpoint v0 coverage now includes 8 run fixtures and 1 backend-C fixture for `KernelCheckpoint`: world memory restore, agent memory restore, EventBus restore, ActuatorHost restore, TraceRecorder restore, RNG seed metadata, restore-all, replay/checkpoint composition, and generated-C shape/forbidden storage checks.

## Phase 21 M1 coverage note

Fixed-size array type syntax `T[N]` is now covered at parser/declaration-check level, including struct fields, function parameters, nested postfix array types, positive literal length validation, and invalid zero/negative/non-literal length fixtures. Runtime construction, indexing, literals, slices, fixed buffers, `Len`, and `Capacity` remain uncovered because they are explicitly deferred beyond M1.
