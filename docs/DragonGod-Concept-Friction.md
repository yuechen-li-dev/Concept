# DragonGod Concept Friction Log

DragonGod.Kernel is intentionally being implemented before self-hosting to stress-test Concept as a real systems programming language.

Each DragonGod phase should record friction discovered while writing real Concept code.

## Entry format

- Phase:
- Area:
- Pain observed:
- Workaround used:
- Classification:
  - language design issue
  - compiler implementation gap
  - standard library/container gap
  - documentation gap
  - expected limitation
- Severity:
  - low
  - medium
  - high
  - blocking
- Suggested future fix:

## DG1 entries

- Phase: 20 / DG1
- Area: Memory key generics
- Pain observed: A kernel-facing `MemoryKey<T>` shape is desirable, but DG1 avoids betting the DragonGod surface on generic storage APIs before the memory model exists.
- Workaround used: Use `MemoryKeyInt` as a temporary integer-backed key shell.
- Classification: expected limitation
- Severity: low
- Suggested future fix: Revisit typed keys when Memory storage and generic API ergonomics are implemented.

- Phase: 20 / DG1
- Area: Reason text
- Pain observed: DragonGod reasons eventually want stable string/StringView semantics, but DG1 only needs deterministic reason identity.
- Workaround used: Use integer-backed `Reason { int code; }` and document it as a placeholder.
- Classification: expected limitation
- Severity: low
- Suggested future fix: Replace or extend `Reason` with a string/StringView-backed surface once Concept string support is kernel-ready.

- Phase: 20 / DG1
- Area: Module source organization
- Pain observed: Multi-source fixtures model one source per module name; the DragonGod tree uses multiple files with the same module name for human organization.
- Workaround used: Keep source-tree skeleton files split by concern, while fixtures aggregate each tested module into a single virtual source.
- Classification: compiler implementation gap
- Severity: medium
- Suggested future fix: Add package/file aggregation semantics for multiple source files contributing to one module.

- Phase: 20 / DG1
- Area: Payload enum construction
- Pain observed: Payload enum variants are usable for DG1, but cross-module examples require verbose fully qualified names.
- Workaround used: Use fully qualified `DragonGod.Kernel.Automata.AutomataSignal::...` references in examples and fixtures.
- Classification: documentation gap
- Severity: low
- Suggested future fix: Document idiomatic import and qualified-name conventions for multi-module Concept code.

## DG2 static automata signals and machine examples

Status: no blocking DG2 friction observed.

DG2 proved the existing Concept machine substrate can return `AutomataSignal` payload enum values without DragonGod-specific compiler hooks. `Step`, `Complete`, and `Result` work on signal-returning machines, and payload variants can be constructed, returned, matched, and inspected for `Succeed`, `Fail`, `Goto`, `Act`, and `AwaitActuation` in static fixtures.

Observed non-blocking friction:

- Imported payload enum examples still use fully qualified `DragonGod.Kernel.Automata.AutomataSignal::...` variant names for clarity and to avoid relying on alias/import sugar that does not exist yet.
- `Reason` remains integer-backed; string/StringView-backed human-readable reasons are still deferred.
- Backend C assertions for payload-enum machine results should pin only stable essentials, because exact generated helper and temporary names are not intended as public DragonGod API.


## DG3 Kernel Memory v0

Status: DG3 Memory v0 complete; no blocking DG3 friction observed.

DG3 replaces the DG1 `Memory` shell with a deterministic fixed-slot integer store. The public names remain `Memory`, `memoryWrite`, `memoryRead`, `memoryHas`, and `memoryRevision`; no Blackboard terminology was introduced.

- Phase: 20 / DG3
- Area: Generic typed Memory keys
- Pain observed: The desired `MemoryKey<T>` shape is still premature for the current minimal store because DG3 has no type-erased payload arena or generic Memory API to bind to.
- Workaround used: Keep `MemoryKeyInt { id: int }` as a v0 scaffold for integer memory values.
- Suggested future fix: Revisit `MemoryKey<T>` when arbitrary typed payload storage and type-erased Memory entries exist.

- Phase: 20 / DG3
- Area: Memory storage containers
- Pain observed: Hash maps, string-key maps, changed-key sets, TTL indexes, and arena-backed containers are intentionally outside the current Concept/DragonGod milestone.
- Workaround used: Use four explicit `MemorySlot` fields (`slot0` through `slot3`) rather than arrays or maps. This keeps generated code direct and fixture-pinnable.
- Suggested future fix: Replace the fixed-slot scaffold with typed arena/hash storage in a later DragonGod Memory phase.

- Phase: 20 / DG3
- Area: Capacity behavior
- Pain observed: A stable Memory-specific error/Result surface is not available yet, and adding one would broaden DG3 beyond API-shape proof.
- Workaround used: A full v0 store leaves overflow writes as no-ops. Successful writes increment `revision`; overflow does not. This limitation is documented rather than hidden.
- Suggested future fix: Add deterministic capacity failure through the eventual Memory error model or shared kernel panic policy.

- Phase: 20 / DG3
- Area: Struct initialization and mutable reference syntax
- Pain observed: Fixed-slot storage requires verbose full struct initialization, and mutation calls require explicit `memory&` address syntax.
- Workaround used: Add `memoryEmptySlot()` and `memoryEmpty()` helpers; fixtures call `memoryWrite(memory&, key, value)` explicitly.
- Suggested future fix: Revisit default field values/constructors and reference ergonomics as the language matures.

- Phase: 20 / DG3
- Area: Memory plus machine parameters
- Pain observed: Passing `Memory` by value into a machine works for read-only decision fixtures, but it is not the final agent-context borrowing model.
- Workaround used: The DG3 signal-machine fixture reads a copied `Memory` and returns `AutomataSignal`; Mind tick and mutable Memory context remain deferred.
- Suggested future fix: Introduce the real Mind/Agent tick context and borrowing shape in a later DragonGod phase.

## DG4 Mind v0 static executor

Status: DG4 Mind v0 static executor complete; no blocking DG4 friction observed.

DG4 upgrades the Mind shell into a minimal static executor/signal recorder. The implementation proved that `Mind` can contain a typed `MindStatus`, remember an `AutomataSignal`, record integer-backed `Reason` values from `Succeed`, `Fail`, and deferred `Goto`, and expose small helpers (`mindEmpty`, `mindRecordSignal`, `mindIsDone`, `mindLastReasonCode`) without DragonGod compiler hooks.

Observed non-blocking friction:

- Phase: 20 / DG4
- Area: Generic machine stepping
- Pain observed: Concept still does not have a clean generic surface for “any concrete machine returning `AutomataSignal`”.
- Workaround used: DG4 keeps static supervision in fixtures and examples: callers own concrete machine values, call `Step`, check `Complete`, then pass `Result` to `mindRecordSignal`.
- Suggested future fix: Revisit a typed/generic static-machine helper after the language has an explicit generic or interface shape for machine operations.

- Phase: 20 / DG4
- Area: Mutable Mind recording ergonomics
- Pain observed: Kernel helper calls require explicit `mind&` syntax for mutation.
- Workaround used: Keep mutation explicit in all examples and fixtures because it documents that Mind recording changes the caller-owned value.
- Suggested future fix: Improve reference ergonomics only if it preserves Concept's explicit mutation doctrine.

- Phase: 20 / DG4
- Area: Context borrowing shape
- Pain observed: `MindCtx` with live `World`/`Agent` references would broaden DG4 into borrowing/lifetime design.
- Workaround used: `MindCtx` stores value-shaped `Clock` and `Memory`, while `Agent` and `World` continue as shells with enough integration for static fixtures.
- Suggested future fix: Introduce borrowed world/agent context when DG5+ stack/tick semantics need it.


## DG5 Automata stack v0

Status: DG5 Automata stack v0 complete; no blocking DG5 friction observed.

DG5 proved that Concept can express a small kernel-owned stack as explicit value fields, return `AutomataFrame` structs by value, mutate `AutomataStack` through `mut AutomataStack&`, and route payload enum variants `Goto`, `Push`, and `Pop` through `mindRecordSignal` without DragonGod compiler hooks.

- Phase: 20 / DG5
- Pain observed: DragonGod wants an eventual vector/arena-backed stack, but the current stable kernel slice intentionally avoids generic containers and allocation.
- Workaround used: `AutomataStack` v0 uses four explicit frame fields plus `depth`.
- Suggested future fix: replace the fixed slots only when the real graph/machine-storage phase needs an arena-backed stack.

- Phase: 20 / DG5
- Pain observed: Fixed slots make push/top/replace code repetitive and field-oriented.
- Workaround used: keep helper functions small and deterministic so callers do not duplicate slot dispatch.
- Suggested future fix: add kernel container primitives later rather than adding a DragonGod-specific compiler feature.

- Phase: 20 / DG5
- Pain observed: Empty-top, overflow, and underflow would be cleaner with an `Option`/`Result` surface for non-panicking callers.
- Workaround used: DG5 uses `panic` with stable DragonGod reasons, matching existing runtime failure doctrine.
- Suggested future fix: introduce explicit optional/result-shaped APIs once Concept's kernel-facing error vocabulary is settled.

## DG6 Decision subsystem v0 friction

DG6 adds `DragonGod.Kernel.Decision` as kernel policy memory, not as language-level decision machinery. The implementation deliberately keeps local stateless machine choice on native `transition decide`, and uses the Decision subsystem only for current-option retention, min-commit, hysteresis, and tie/current preference.

Observed friction:

- Lack of arrays/vectors/spans keeps DG6 on a fixed four-option `decisionChoose4` API, matching DG3 Memory and DG5 AutomataStack fixed-slot style.
- Integer policy is a good Stage 0 fit; floating scores, probability, random weighted selection, and consideration curves remain deferred.
- Lack of generics/function pointers means reusable `ConsiderationFn`/option collections are not expressible cleanly yet.
- `StateId` and `Reason` construction remains verbose in examples and integration fixtures.
- Reusable utility-option collections are hard to express without arrays or slices.
- `match` remains useful for `AutomataSignal` handling in Mind integration, avoiding tag if-chains.
- `transition decide` remains useful for local machine decisions; DG6 fixtures keep it separate from memoryful policy arbitration.
- The only ordered checks in `decisionBestOption4` are deterministic fixed-slot source-order comparisons, not enum/payload tag chains.

## Phase 20 / DG7 Actuation subsystem v0 friction

DG7 added `DragonGod.Kernel.Actuation` as a fixed-slot, explicit host surface instead of a generic command runtime.

- Marker concept attempt: `marker concept ActuationCommand<C>;` matches Phase 8's marker concept direction, but DG7 did not keep it in source because command payloads and generic dispatch would be ceremonial without arrays/vectors or a handler registry. Workaround: concrete integer-backed `ActuationId` plus fixed `ActuatorHost` slots. Future fix: first-class marker-only concepts that can document kernel traits without requiring an unused witness surface.
- Interfaces/dyn handlers: Phase 14 borrowed `dyn Interface&` can express a small handler parameter shape, but a useful actuator handler table would need arrays/vectors/function pointers or stored dyn fields. Workaround: no handler table in DG7. Future fix: stable dyn fields/arrays or explicit noalloc handler-table primitives.
- Generic dispatch: generic command payload dispatch was not clean for v0 because there is no dynamic command registry and arbitrary payload storage would require type erasure. Workaround: `actuatorDispatch(host&, ActuationDecision)` only records host status.
- Payload enum `match`: `match` remained usable for `AutomataSignal::Act`, `AutomataSignal::AwaitActuation`, and `ActuationStatus` smoke fixtures. This was the best-fit Concept feature for signal/status inspection.
- Fixed slots: repeating slot0/slot1/slot2/slot3 logic is still noisy in `ActuatorHost`, as it was for Memory and AutomataStack. Lack of arrays/vectors keeps the v0 code explicit but repetitive.
- `Reason`: integer-backed `Reason { code: ... }` was sufficient for allow/deny/complete/fail fixtures, but it still cannot carry descriptive kernel policy text.
- `mut ActuatorHost&`: explicit mutable host references compiled and made mutation sites clear; the ergonomics are verbose but acceptable for kernel-facing examples.
- Module import verbosity: fully-qualified DragonGod names remain long in examples. Local aliases/import member syntax would make examples easier to read.
- Missing arrays/vectors/function pointers: these are the main blockers for a real deterministic handler registry, policy chains, pending completion queues, and command payload tables. DG7 intentionally defers those designs.

## DG8 Events v0 friction

- Events v0 uses four explicit slots because Concept still lacks a standard Vector/container, arrays/spans ergonomic enough for this kernel proof, and generic fixed-capacity storage helpers.
- Event payloads are integer-only (`EventPayloadInt`) because generic typed payloads, type-erased arbitrary values, and event bucket maps would add more language and runtime surface than DG8 needs.
- Event reads use `eventReadNextInt(bus, cursor&, type, fallback)` because Concept does not yet have a standard `Option`/`Result` return or out-parameter convention that is clearer for small fixtures.
- Missing-event behavior advances the stream cursor to `bus.count`; this avoids rescanning old fixed slots but has to be documented because cursors are generic stream positions, not per-type bucket cursors.
- Slot scanning is written explicitly rather than with loops over arrays because the current DragonGod fixed-slot style is more predictable for backend fixtures.


## DG9 Trace subsystem v0 friction

No blocking DG9 friction observed. `TraceEvent` can store `AutomataSignal` directly, and `match` over both `TraceEventKind` and `AutomataSignal` was usable for compact interpretation helpers. A minimal `TraceSink` interface with `StateId`, `Reason`, and payload enum `AutomataSignal` arguments compiled in a smoke fixture, and borrowed `mut dyn TraceSink&` calls worked for explicit sink emission.

Non-blocking friction remains:

- The fixed-slot recorder repeats field and branch boilerplate (`event0` through `event3`) because Concept still lacks the arrays/vectors/spans needed for a cleaner trace ring/buffer surface.
- Lack of stable strings/StringView keeps trace reasons integer-backed and prevents useful textual trace payloads without turning Trace into formatting/logging work.
- Module-qualified signatures for kernel interfaces are verbose, especially for `TraceSink` methods crossing Core and Automata types.
- Backend C fixture assertions remain shape-oriented and intentionally avoid overfitting temporary names, but they still require explicit absence checks for heap/logging/scheduler markers.

Workaround: DG9 keeps `TraceRecorder` as the v0 implementation and treats `TraceSink` as smoke-proven interface surface only. Future Concept improvements that would materially improve Trace include fixed arrays, slices/spans, string views, and lower-noise qualified import ergonomics.

## DG10 Replay v0 friction

- `ReplayEvent` payload enums are expressive enough for fixed integer variants, but fixture stability still benefits from compact standalone source because cross-module fixture imports are heavier than inline backend/run sources.
- `match (event)` cleanly expresses replay dispatch and avoids a giant imperative if/else chain; each arm currently returns explicitly because statement-expression fallthrough ergonomics are still narrow.
- `ReplayDriver` is value-shaped rather than reference-shaped. Pointer/reference fields for long-lived kernel shell ownership remain awkward for a tiny deterministic fixture surface, so v0 copies `World`, `Agent`, `EventBus`, `ActuatorHost`, and `TraceRecorder` into the driver.
- Mutating nested driver fields works (`driver.world.memory&`, `driver.events&`, `driver.actuator&`, `driver.trace&`) but remains verbose when fully-qualified module names are involved.
- Lack of arrays/vectors keeps `ReplayLog` at named fixed slots (`event0` through `event3`) and forces `replayApplyAll` to use four bounded calls instead of a capacity-driven loop abstraction.
- `replayApplyNext` returns `bool` for exhausted/applied because there is no ergonomic Option/Result surface for the DG10 driver status yet.
- No file/string/serialization substrate is used; replay loading/saving, JSON, binary checkpoints, and Persistence are intentionally deferred.
- Events and Actuation integrate through their existing fixed-slot APIs. Trace integration uses an explicit `TraceMark` mapped onto `traceEnter` as a marker-shaped v0 workaround until Trace has a dedicated mark kind.
- Backend fixtures assert broad no-hidden-runtime properties, but avoid overfitting every generated temporary name beyond the replay enum/log/driver/switch shape.

## DG11 Persistence/checkpoint v0 friction

- Large value-shaped snapshots are possible, but ergonomics are repetitive: every fixed-slot kernel shape must be named explicitly in `KernelCheckpoint` and constructor/restore helpers.
- Nested struct restore works through whole-struct assignment, but field-by-field alternatives would become noisy for World/Agent memory, EventBus, ActuatorHost, and TraceRecorder.
- Multiple `mut &` restore parameters compiled in fixtures, but the call shape is visually dense; v0 keeps explicit helpers plus `checkpointRestoreAll`.
- Persistence does not import Replay in v0 to avoid coupling checkpoint state to replay log mechanics; RNG seed metadata is captured as an `int`.
- No serialization/string/file primitives were used; this is correct for v0, but future disk persistence will need separate design rather than stretching checkpoint helpers.
- No generic snapshot traits/interfaces or array/vector/spans exist, so fixed-slot state snapshot repetition remains visible.
- Backend fixtures assert broad generated-C shape and forbidden terms; exact temporary-name assertions remain intentionally avoided for stability.
