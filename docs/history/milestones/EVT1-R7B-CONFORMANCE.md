# EVT1 R7b conformance: DragonGod agentic kernel parity

R7b restores the useful execution substrate consumed by the C# Dominatus
application kernel as one native Concept package. Dominatus is the parity oracle,
not a second Concept package or namespace. All production APIs remain under
`DragonGod.*`.

## Baseline

- Baseline HEAD: `f291c74bd23b1ae410b539a6725f77194299617f`
- R7a commit: `Reforge DragonGod and establish Concept libraries`
- Compiler: `concept-evt1-stage0-go`
- Package graph: `DragonGod -> Standard`; `Standard` has no dependencies.
- Active implementation: `libraries/DragonGod`.
- Archived oracle: `legacy/dragon-god-poc`.
- C# consumer oracle: sibling repository `Dominatus`.
- Native proof path: Concept -> MIR -> Planner -> generated strict C11 -> host C
  compiler -> native fact executable.

The baseline Go, Zig, legacy Zig, Standard-package, and DragonGod-package checks
passed before implementation.

## Conformance results

| Requirement | Evidence | Result |
|---|---|---|
| One canonical package | Manifest still declares only `DragonGod -> Standard`; no Dominatus package or compatibility adapter was added | Pass |
| Configurable bounded storage | `MemoryState`, `EventBus`, `ActuatorHost`, `TraceRecorder`, `ReplayLog`, and `Mind` accept compile-time capacities; tests instantiate 2, 3, 6, and 16 | Pass |
| Explicit execution and batch order | `Mind.Step` and `Mind.ExecuteBatch` process a caller-owned array in source order under an explicit budget | Pass |
| Events and dirty memory | Cursor-based event reading and revision/dirty-key behavior are native package facts | Pass |
| Actuation obligations | Dispatch creates stable IDs and explicit pending/completed/failed states | Pass |
| Time and random state | Unit-typed monotonic seconds and caller-visible seed are checkpointed/replayed | Pass |
| Trace, replay, checkpoint | Typed bounded records, semantic inputs, explicit cursor, and by-value checkpoint restore are exercised end to end | Pass |
| Ordinary failures | Full capacity and missing obligations return typed `Result` errors | Pass |
| Native execution | Seven DragonGod package facts compile as strict C11 and execute natively | Pass |
| No scheduler/collector | No ready queue, fairness, priority, preemption, GC root, mark, or sweep surface exists | Pass |

The representative scenario dispatches and consumes an event, drives an
application-owned native automaton through guarded transitions, updates dirty
memory, creates an actuation obligation, executes four Mind signals as an
ordered batch, records typed trace entries, checkpoints, replays time/completion/
seed inputs, and restores the checkpoint. Its golden outcome is state 30,
Succeeded status, memory value 7, pending actuation before replay, completed
actuation after replay, time 2 s, seed 42, and restored pending/seed-zero state.

## General compiler repairs

R7b required no DragonGod-specific semantic node or lowering. Four bounded,
regression-tested artifact/generic defects were repaired:

1. statement substitution now traverses `foreach`;
2. symbolic non-type arguments remain open and close recursively in nested
   generic applications;
3. non-type parameters in generic method signatures are substituted and every
   distinct method instance is materialized by signature, not name;
4. imported aggregate methods and generated calls preserve ordinary nominal
   overload resolution.

## Audits and boundaries

Canonical kernel code has no legacy import, hidden heap, reflection registry,
saved-PC coroutine runtime, global event bus, scheduler, collector, file I/O,
networking, or Vulkan policy. Decision scoring and graph meaning stay in the
application or in ordinary Concept `match`, `transition match`, `decide`, and
`infer` semantics.

Replay input application remains a typed `match` inside the generic replay
driver. Current automata declarations cannot be generic over the caller-owned
`AgentContext<Configuration, Capacity>`; introducing a concrete shadow context
and a two-step classifier machine would duplicate authority. When generic
automata contexts become a general language capability, this is a legitimate
candidate for direct state/transition spelling without changing replay meaning.

R7c may assume stable IDs, explicit machine stepping, deterministic bounded batch
execution, monotonic time, events, interrupt facts, actuation obligations,
structured traces, replay/checkpoints, and Standard.Memory ownership. Scheduler
work begins only after the canonical kernel can already perform the useful
historical Dominatus execution model without it.

## Final validation evidence

- `go test ./...`: pass (`internal/concept` 98.003 s; Vulkan profile 0.280 s).
- `go vet ./...`: pass.
- root `zig build test`: pass with Zig 0.16.0.
- `legacy/poc3-zig` `zig build test`: pass.
- Oct `BuildStandard`, `TestStandard`, `BuildDragonGod`, and `TestDragonGod`:
  pass after syncing the already-declared local `Make@0.1.0` dependency.
- Standard native facts: 2 passed. DragonGod native facts: 7 passed.
- Full active EVT1 corpus: 712 `.concept` fixtures (749 files total under
  `language/evt1`) covered by the passing Go suite.
- Artifact-only agentic consumer: generated-C audit and strict-C11 native run
  pass.
- 100 DragonGod package builds: identical output SHA-256
  `36B844CF34EC680820E0FF7424153C418B8E5D57BF1FAB5C7A29CA6FA80C1341`.
- 100 complete native DragonGod package test runs: identical output SHA-256
  `92D4A7B4CBF37D00A4CEFAB780F0FD1C95D1FC5C183F13E8478797B32B7AB755`.
- `git diff --check`: pass.
