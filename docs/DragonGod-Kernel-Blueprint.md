# Phase 20: DragonGod Kernel Blueprint

## Concept-native behavioral kernel design

**Status:** DG0 blueprint complete; DG1 kernel core seed complete
**Target:** Concept Stage 1+
**Root module:** `DragonGod.Kernel`
**Derived from:** Dominatus doctrine and production architecture
**Not a port:** DragonGod is the native Concept form of the Dominatus idea
**License:** GPL-3.0, unless project policy changes later

---

## Core doctrine

```text
DragonGod is a kernel, not a library.

DragonGod.Kernel is the behavioral nucleus that user agents, game AI, simulation controllers, robotics controllers, and embedded decision systems orbit.

Dominatus proved the doctrine.

Concept gives the native substrate.

DragonGod.Kernel is not a line-for-line Dominatus port.

Concept machines replace NodeRunner.

Bare yield replaces IEnumerator tick suspension.

AutomataSignal replaces yielded AiStep records as upward automata intent.

Mind replaces BrainInstance as the agent-owned automata executor.

Memory replaces Blackboard as typed contextual state.

DragonGod.Kernel is built from explicit values, explicit memory, explicit machines, explicit ticks, explicit signals, and explicit failure.

No hidden heap.

No hidden scheduler.

No hidden event loop.

No hidden async runtime.

No hidden coroutine protocol.

No hidden blackboard.

No hidden DragonGod compiler magic.
```

---

## 1. Purpose

DragonGod.Kernel is a Concept-native behavioral kernel for building deterministic, inspectable, replayable, tick-driven intelligent systems.

It targets:

```text
game AI
agent orchestration
simulation agents
robotics-style controllers
bare-metal controllers
RISC-V behavioral kernels
high-performance native decision systems
LLM-actuated agent control planes
```

DragonGod is the successor-form of the Dominatus architecture, but not a C# port.

Dominatus used C# constructs such as delegates, records, `IEnumerator<AiStep>`, object dictionaries, managed allocation, and host runtime facilities to express a behavioral kernel.

Concept has the primitives Dominatus wanted:

```text
machine
yield
transition match
transition decide
nested machine frames
State(machine)
explicit allocation
arena storage
interfaces / dyn dispatch
imports
C ABI
repr(C)
panic/assert runtime failure
```

DragonGod exists to express the same doctrine natively.

---

## 2. Relationship to Dominatus

Dominatus remains the reference implementation for doctrine and behavioral semantics.

DragonGod is the native Concept kernel inspired by it.

Mapping:

```text
Dominatus C# concept          DragonGod.Kernel Concept concept
──────────────────────────    ─────────────────────────────────────
AiNode delegate               machine
IEnumerator<AiStep>           machine + yield + AutomataSignal result
AiStep record hierarchy       AutomataSignal payload enum
NodeRunner                    compiler-lowered machine execution
HfsmGraph / HfsmInstance      AutomataGraph / Mind
Blackboard                    Memory
Dictionary<string, object?>   MemoryKey<T> + MemoryEntry storage
UtilityLite                   Decision subsystem
ActuatorHost                  Actuation subsystem
AiWorld                       World
AiAgent                       Agent
Trace sink interfaces         TraceSink interface
Replay log                    Replay subsystem
Parallel runner               Parallel staged tick subsystem
Checkpoint                    Persistence subsystem
```

Important difference:

```text
Dominatus yielded steps because C# needed IEnumerator to model cooperative behavior.

DragonGod machines use bare yield; for suspension.

DragonGod machines return AutomataSignal only when they complete with upward intent.
```

Therefore, DragonGod must not blindly copy the Dominatus `AiStep` mental model.

---

## 3. Terminology

### 3.1 Kernel

The root module is:

```cpp
module DragonGod.Kernel;
```

DragonGod.Kernel is the behavioral kernel.

It is not “just a library.” It defines the system’s core execution model, signal model, memory model, decision model, replay model, and deterministic tick doctrine.

### 3.2 Machine

A `machine` is the Concept language primitive.

A machine is an explicit runtime frame lowered by the compiler.

DragonGod uses Concept machines as behavior nodes.

### 3.3 Automata

`Automata` is the DragonGod subsystem for state graphs, stack frames, transitions, and upward signals.

Avoid the acronym `HFSM` in DragonGod public names.

Use:

```text
Automata
Automaton
AutomatonGraph
AutomataFrame
AutomataStack
AutomataTransition
AutomataSignal
```

### 3.4 Mind

`Mind` is the agent-owned executor.

It replaces `BrainInstance`.

A Mind owns or references the active automata stack, decision memory, timing gates, transition scan state, and current execution context.

### 3.5 Memory

`Memory` replaces `Blackboard`.

Memory is typed contextual state.

Use:

```text
Memory
MemoryKey<T>
MemoryEntry
memoryRead
memoryWrite
memoryChanged
memoryClearChanges
```

Do not use `BbKey`, `bbGetOrDefault`, or public “blackboard” naming in DragonGod.Kernel.

### 3.6 AutomataSignal

`AutomataSignal` replaces `AiStep`.

It represents upward intent emitted by a completed behavior machine.

It is not a yielded stream item.

It is not C# `yield return`.

It is not the per-tick suspension mechanism.

---

## 4. Module map

Full cathedral module map:

```text
DragonGod.Kernel

  Core
    ids.concept
    clock.concept
    reason.concept
    status.concept
    platform_profile.concept

  Automata
    signal.concept
    state_id.concept
    graph.concept
    transition.concept
    frame.concept
    stack.concept
    ops.concept
    static_automata.concept
    dynamic_automata.concept

  Mind
    mind.concept
    tick.concept
    options.concept
    context.concept
    active_path.concept

  Memory
    key.concept
    entry.concept
    memory.concept
    revision.concept
    changed_keys.concept
    ttl.concept

  Decision
    consideration.concept
    policy.concept
    slot.concept
    memory.concept
    utility_option.concept

  Actuation
    command.concept
    policy.concept
    host.concept
    result.concept
    pending.concept

  Events
    bus.concept
    bucket.concept
    cursor.concept

  Trace
    sink.concept
    null_sink.concept
    trace_event.concept

  Persistence
    checkpoint.concept
    snapshot.concept
    serializer.concept

  Replay
    replay_event.concept
    replay_log.concept
    replay_driver.concept

  Parallel
    staged_tick.concept
    conflict_policy.concept
    staged_surface.concept

  Platform
    hosted.concept
    freestanding.concept
    pthread.concept
    riscv.concept
```

Implementation phases may create a smaller subset first.

The blueprint describes the full cathedral.

---

## 5. Concept primitives used

DragonGod.Kernel is designed around these Concept features:

```text
machine
yield
transition match
transition decide
nested machine fields
State(machine)
explicit panic/assert
explicit allocation effects
Arena
C ABI extern/export
repr(C)
imports/multi-module compilation
interfaces and dyn dispatch
concept constraints
payload enums
records/structs
```

Phase dependency:

```text
Phase 18: composable machines and runtime transitions
Phase 19: yielding machines and explicit suspension
Phase 20: DragonGod.Kernel blueprint
Future: DragonGod.Kernel implementation slices
```

DragonGod.Kernel should not depend on unimplemented language magic.

When the full blueprint describes future facilities, it must clearly label them as future.

---

## 6. Execution modes

DragonGod.Kernel has two execution modes.

### 6.1 Static automata mode

Static mode is the first implementation path.

It uses concrete Concept machine types directly.

Properties:

```text
no type-erased machine storage
no dynamic graph configuration
no runtime machine factory
no heap-owned machine frames
compile-time known machine types
best for bare-metal proof of concept
best for RISC-V kernel demo
best first implementation slice
```

Example:

```cpp
machine Patrol(MindCtx const& ctx) -> AutomataSignal {
    state Start {
        transition Watch;
    }

    state Watch {
        if (!memoryRead(ctx.agent.memory, Keys.ThreatDetected, false)) {
            yield;
        }

        return AutomataSignal::Goto(Keys.CombatState, "threat detected");
    }
}
```

A static Mind or Agent can own concrete machines directly:

```cpp
struct PatrolAgent {
    Patrol patrol;
};
```

Static mode proves the kernel substrate before dynamic graph machinery is introduced.

### 6.2 Dynamic automata mode

Dynamic mode is the full cathedral.

It supports data-driven AutomatonGraph execution with type-erased active frames.

Properties:

```text
AutomatonGraph
AutomataFrame
type-erased machine storage
explicit MachineOps table
Mind stack
state entry/exit
transition scans
decision memory
trace hooks
checkpoint/replay integration
```

Dynamic frame sketch:

```cpp
struct AutomataFrame {
    StateId id;
    void* machineStorage;
    AutomataMachineOps const* ops;
    float enterTime;
};
```

Ops sketch:

```cpp
struct AutomataMachineOps {
    void step(void* frame);
    bool complete(void const* frame);
    AutomataSignal result(void const* frame);
    void reset(void* frame);
    void drop(void* frame);
};
```

Dynamic mode is deferred until static mode, core signal handling, Memory, and Mind are proven.

---

## 7. Core subsystem

Module:

```cpp
module DragonGod.Kernel.Core;
```

Core contains small, stable types used across the kernel.

Suggested types:

```cpp
struct AgentId {
    uint64 value;
};

struct StateId {
    uint32 value;
};

struct ActuationId {
    uint64 value;
};

struct DecisionSlotId {
    uint32 value;
};

struct Clock {
    float time;
};

struct Reason {
    StringView text;
};
```

Reason strings are not optional for behavioral transitions.

A reason string explains why a transition/signal/action happened.

This doctrine is inherited from Dominatus and aligned with Concept’s runtime failure reason doctrine.

---

## 8. AutomataSignal

Module:

```cpp
module DragonGod.Kernel.Automata;
```

`AutomataSignal` is a payload enum returned by behavior machines to the Mind.

It replaces Dominatus `AiStep`.

It is not a yielded value.

It is not used for ordinary tick suspension.

Bare `yield;` handles tick suspension.

Recommended core signal enum:

```cpp
enum AutomataSignal {
    None,

    Goto(StateId target, StringView reason),
    Push(StateId target, StringView reason),
    Pop(StringView reason),

    Succeed(StringView reason),
    Fail(StringView reason),

    Act(ActuationId id),
    AwaitActuation(ActuationId id),
};
```

Deferred / not core v0:

```text
WaitSeconds
WaitUntil
Decide
yielded values
stream events
generator items
```

Rationale:

```text
WaitSeconds and WaitUntil are better expressed as machine state + yield patterns.

Decide is better expressed as native transition decide inside machines when the decision is local to the machine.

AutomataSignal is for upward automata / mind intent.
```

Example:

```cpp
machine Patrol(MindCtx const& ctx) -> AutomataSignal {
    state Watch {
        if (!memoryRead(ctx.agent.memory, Keys.ThreatDetected, false)) {
            yield;
        }

        return AutomataSignal::Goto(Keys.Combat, "threat detected");
    }
}
```

---

## 9. Automata subsystem

Module:

```cpp
module DragonGod.Kernel.Automata;
```

The Automata subsystem defines state graph structures, stack frames, transitions, and runtime machine operations.

### 9.1 State IDs

```cpp
struct StateId {
    uint32 value;
};
```

State IDs are kernel-level graph identifiers.

They are distinct from Concept machine internal numeric state IDs returned by `State(machine)`.

`State(machine)` is a debug/control observation of a concrete machine frame.

`StateId` identifies AutomataGraph states.

### 9.2 AutomataTransition

```cpp
struct AutomataTransition {
    StateId from;
    StateId to;
    Reason reason;

    // future:
    // span<MemoryKeyName const> dependsOn;
    // cadence gate
    // priority
    // guard function
};
```

Transition scanning belongs to the Mind / AutomataGraph layer.

Native `transition match` and `transition decide` belong inside Concept machine bodies.

Both are useful.

They operate at different layers:

```text
machine transition match/decide = local machine control
AutomataTransition scan         = graph/stack-level control
```

### 9.3 AutomatonGraph

```cpp
struct AutomatonGraph {
    StateId root;
    // state definitions
    // transition definitions
    // interrupt definitions
};
```

Dynamic graph implementation is future work.

Static mode may not need `AutomatonGraph` at first.

### 9.4 AutomataStack

```cpp
struct AutomataStack {
    // arena/vector of AutomataFrame
};
```

Stack operations:

```cpp
void automataPush(mut Mind& mind, StateId target, StringView reason);
void automataPop(mut Mind& mind, StringView reason);
void automataReplaceTop(mut Mind& mind, StateId target, StringView reason);
void automataUnwindAbove(mut Mind& mind, int indexInclusive, StringView reason);
```

All stack-changing operations require reason strings.

---

## 10. Mind subsystem

Module:

```cpp
module DragonGod.Kernel.Mind;
```

`Mind` is the agent-owned automata executor.

It replaces `BrainInstance`.

Suggested shape:

```cpp
struct Mind {
    AutomatonGraph const* graph;
    MindOptions options;
    Arena* arena;

    // AutomataStack stack;
    // Decision memory;
    // active path;
    // trace sink;
    // scan timing gates;

    float nextInterruptScanTime;
    float nextTransitionScanTime;
    uint32 lastMemoryRevisionScanned;
};
```

### 10.1 Mind context

Behavior machines should receive an explicit context.

```cpp
struct MindCtx {
    World const* world;
    Agent* agent;
    Mind* mind;
    Clock clock;
};
```

Future effect-typed contexts may distinguish:

```text
noalloc
alloc
hosted
freestanding
interrupt-safe
```

### 10.2 Tick order

Full dynamic Mind tick order:

```text
1. Memory revision check + cadence gate
2. Interrupt scan, top to bottom
3. Transition scan, top to bottom
4. Root overlay tick if keepRootFrame
5. Leaf machine tick
6. AutomataSignal handling
7. Trace emission
```

Static mode may implement only a smaller subset.

### 10.3 Keep-root-frame doctrine

`keepRootFrame` preserves a root behavior frame while leaf states execute above it.

This supports intent-driven AI:

```text
root = global intent / world awareness
leaf = current tactical behavior
```

Deferred for first static implementation if needed.

---

## 11. Memory subsystem

Module:

```cpp
module DragonGod.Kernel.Memory;
```

Memory replaces Blackboard.

Memory is typed contextual state.

### 11.1 Types

```cpp
struct MemoryKey<T> {
    StringView name;
};

struct MemoryEntry {
    void* value;
    float expiresAt;
    bool hasTtl;
};

struct Memory {
    Arena* arena;
    uint32 revision;
    // map StringView -> MemoryEntry
    // changed key set
};
```

### 11.2 API

Preferred API names:

```cpp
void memoryWrite<T>(mut Memory& memory, MemoryKey<T> key, T value);

void memoryWriteFor<T>(
    mut Memory& memory,
    MemoryKey<T> key,
    T value,
    float now,
    float ttlSeconds
);

T memoryRead<T: Copy<T>>(
    Memory const& memory,
    MemoryKey<T> key,
    T fallback
);

bool memoryHas<T>(Memory const& memory, MemoryKey<T> key);

int memoryExpire(mut Memory& memory, float now);

bool memoryChanged(Memory const& memory, StringView keyName);

void memoryClearChanges(mut Memory& memory);
```

Use `memoryRead`, not `getOrDefault`.

Reason:

```text
read = non-mutating lookup with fallback
write = mutating update
has = existence check
expire = TTL cleanup
changed = transition gating
```

### 11.3 Changed-key doctrine

Use “changed keys,” not “dirty keys,” in public DragonGod.Kernel terminology.

Transition scans can be gated by changed keys:

```text
Only evaluate transitions whose dependency keys changed, unless cadence/force rules require a scan.
```

### 11.4 Memory doctrine

```text
Memory is explicit.
Memory is typed at the key boundary.
Memory may use type erasure internally.
Memory uses explicit allocator/arena storage.
Memory is not a hidden global blackboard.
Memory changes increment revision.
Memory changes can drive transition scan cadence.
```

---

## 12. Decision subsystem

Module:

```cpp
module DragonGod.Kernel.Decision;
```

Decision handles utility scoring, hysteresis, min-commit windows, and tie rules.

Native machine `transition decide` already provides deterministic local utility selection.

The Decision subsystem exists for Mind-level and Automata-level decision memory.

### 12.1 Types

```cpp
struct DecisionSlot {
    StringView id;
};

struct DecisionPolicy {
    float hysteresis;
    float minCommitSeconds;
    float tieEpsilon;
};

struct DecisionMemory {
    StateId currentOption;
    float currentScore;
    float lastSwitchTime;
};

struct UtilityOption {
    StringView id;
    StateId target;
};
```

### 12.2 Arbitration doctrine

Dominatus-derived decision arbitration:

```text
1. Evaluate candidate option scores.
2. If best == current, refresh memory and keep current.
3. If inside min-commit window, block switch.
4. If hysteresis threshold not exceeded, block switch.
5. If within tie epsilon of current, prefer current.
6. Otherwise switch to best and record time.
```

### 12.3 Relationship to transition decide

```text
transition decide = local machine control
Decision subsystem = memoryful policy-driven mind/automata arbitration
```

Do not force all decisions upward into Mind.

Prefer native `transition decide` for local state choice.

---

## 13. Actuation subsystem

Module:

```cpp
module DragonGod.Kernel.Actuation;
```

Actuation dispatches commands to external systems.

### 13.1 Concepts

```cpp
marker concept ActuationCommand<C>;

concept ActuationPolicy<P> {
    ActuationDecision evaluate(
        P const& self,
        MindCtx const& ctx,
        void const* command,
        TypeId commandType
    ) noalloc;
};
```

### 13.2 Types

```cpp
struct ActuationDecision {
    bool allowed;
    StringView reason;
};

struct ActuationDispatchResult {
    ActuationId id;
    bool accepted;
    bool completed;
    bool ok;
    StringView error;
};

struct ActuatorHost {
    Arena* arena;
    // handler table
    // policy list
    // pending completions
};
```

### 13.3 API

```cpp
void hostRegister<C: ActuationCommand<C>, H>(
    mut ActuatorHost& host,
    H handler
);

ActuationDispatchResult hostDispatch<C: ActuationCommand<C>>(
    mut ActuatorHost& host,
    MindCtx const& ctx,
    C const& command
);

void hostCompleteLater(
    mut ActuatorHost& host,
    MindCtx const& ctx,
    ActuationId id,
    float dueTime,
    bool ok,
    StringView error
);

void hostTick(mut ActuatorHost& host, mut World& world);
```

Actuation is an explicit subsystem.

It is not a scheduler.

It is not an async runtime.

---

## 14. Events subsystem

Module:

```cpp
module DragonGod.Kernel.Events;
```

Events are typed append-only streams with cursors.

### 14.1 Types

```cpp
struct EventBucket {
    Arena* arena;
    TypeId eventType;
    // vector of event payloads
};

struct EventCursor {
    int index;
};

struct EventBus {
    Arena* arena;
    // map TypeId -> EventBucket
};
```

### 14.2 API

```cpp
void eventPush<E>(mut EventBus& bus, E const& event);

bool eventTryConsume<E>(
    mut EventBus& bus,
    mut EventCursor& cursor,
    E& out
);

EventCursor eventTailCursor<E>(EventBus const& bus);
```

### 14.3 Trimming doctrine

Dominatus-style event bucket trimming is preserved as an important memory-management doctrine.

If all cursors have advanced sufficiently, old event prefixes may be trimmed.

Trimming must be deterministic and safe.

---

## 15. World and Agent

Module:

```cpp
module DragonGod.Kernel.Core;
```

### 15.1 World

```cpp
struct World {
    Arena* arena;
    Clock clock;
    Memory memory;
    ActuatorHost* actuator;
    // agents
    // public snapshots
};
```

### 15.2 Agent

```cpp
struct Agent {
    AgentId id;
    Memory memory;
    EventBus events;
    Mind mind;
};
```

### 15.3 Tick

```cpp
void worldTick(mut World& world, float dt);
void worldAdd(mut World& world, mut Agent& agent);
```

Tick doctrine:

```text
worldTick is explicit.
agents are ticked explicitly.
time advances explicitly.
no hidden scheduler.
no hidden async.
```

---

## 16. Trace subsystem

Module:

```cpp
module DragonGod.Kernel.Trace;
```

Trace sinks observe behavior.

```cpp
interface TraceSink {
    void onEnter(StateId state, float time, StringView reason);
    void onExit(StateId state, float time, StringView reason);
    void onTransition(StateId from, StateId to, float time, StringView reason);
    void onYield(StateId state, float time);
    void onSignal(AutomataSignal const& signal, float time);
};
```

Do not model `yield` as an `AutomataSignal`.

`yield;` has no value.

Trace may observe yield as an event, but yield does not emit a signal upward.

---

## 17. Persistence subsystem

Module:

```cpp
module DragonGod.Kernel.Persistence;
```

Persistence handles checkpoint structures and serialization policy.

### 17.1 Doctrine

```text
Machines are not serialized.

Active automata paths are serialized.

Memory snapshots are serialized.

In-flight actuation state may be serialized.

Machines are reconstructed by re-entry and replay.

Replay restores behavior deterministically.
```

### 17.2 Types

```cpp
struct AgentCheckpoint {
    AgentId agentId;
    // memory snapshot
    // active automata path
    // in-flight actuation IDs
    // event cursor positions
};

struct DragonGodCheckpoint {
    float clockTime;
    // world memory snapshot
    // agent checkpoints
};
```

### 17.3 Binary checkpoint future

Once `repr(C)` and binary serialization are mature:

```cpp
repr(C)
struct MemoryCheckpointHeader {
    uint32 magic;
    uint32 version;
    uint32 entryCount;
    uint32 dataSize;
};
```

Persistence may provide hosted and freestanding profiles.

---

## 18. Replay subsystem

Module:

```cpp
module DragonGod.Kernel.Replay;
```

Replay is the deterministic input stream.

### 18.1 Replay doctrine

```text
All nondeterministic input enters through ReplayEvent.

Replay is the audit path.

Machines are reconstructed; replay drives them forward.

Same replay log plus same kernel version should produce the same behavior.
```

### 18.2 Types

```cpp
enum ReplayEvent {
    Advance(AgentId agent),
    Text(AgentId agent, StringView value),
    Choice(AgentId agent, StringView choiceKey),
    External(AgentId agent, StringView type, StringView payload),
    RngSeed(uint64 seed),
};

struct ReplayLog {
    // vector ReplayEvent
};

struct ReplayDriver {
    World* world;
    ReplayLog* log;
    int cursor;
};
```

### 18.3 API

```cpp
void replayApplyNext(mut ReplayDriver& driver);
void replayApplyAll(mut ReplayDriver& driver);
```

---

## 19. Parallel subsystem

Module:

```cpp
module DragonGod.Kernel.Parallel;
```

Parallel tick is deterministic staged execution.

### 19.1 Doctrine

```text
Agents compute independently from a tick-N snapshot.

World writes are staged.

A merge barrier commits effects in deterministic order.

Sequential and parallel tick should be hash-identical when conflict policy is deterministic.

No hidden concurrency model.
```

### 19.2 Types

```cpp
enum WorldWriteConflictPolicy {
    Fail,
    LastWriterByAgentId,
    FirstWriterByAgentId,
};

struct ParallelTickOptions {
    int maxDegreeOfParallelism;
    WorldWriteConflictPolicy conflictPolicy;
    bool advanceWorldClock;
    bool expireWorldMemory;
    bool tickActuator;
};

struct ParallelTickResult {
    int agentsTicked;
    int worldWritesStaged;
    int worldWritesCommitted;
    int mailboxMessagesDelivered;
    int actuationsDispatched;
};
```

Parallel uses explicit platform thread primitives later.

No hidden scheduler is introduced into Concept core.

---

## 20. Platform profiles

Module:

```cpp
module DragonGod.Kernel.Platform;
```

DragonGod.Kernel should support multiple profiles:

```text
Hosted
Freestanding
BareMetal
InterruptSafe
ParallelHosted
```

### 20.1 Hosted

Hosted profile may use:

```text
C ABI
stdio for diagnostics
host allocator adapters
pthread / platform threads
file-backed replay logs
```

### 20.2 Freestanding / bare metal

Freestanding profile must avoid:

```text
stdio
malloc
OS threads
filesystem
hidden allocator dependencies
```

It should use explicit platform hooks.

### 20.3 Interrupt-safe subset

Interrupt-safe machines should be expressible later with effects:

```cpp
noalloc nothrow machine KernelController(Device const& dev) -> AutomataSignal {
    ...
}
```

This is future work.

The blueprint preserves it as a direction.

---

## 21. Determinism doctrine

DragonGod.Kernel is replay-first.

Non-negotiable properties:

```text
All nondeterministic input enters through ReplayEvent or explicit platform hooks.

Transition reason strings are mandatory.

Decision tie-breaking is deterministic.

Parallel merge order is deterministic.

Memory revision and changed-key gating are deterministic.

Machines are reconstructed by re-entry/replay, not serialized as opaque continuations.

yield does not preserve hidden continuation points.

yield re-enters state body from the beginning.
```

---

## 22. Memory and allocation doctrine

DragonGod.Kernel is explicit-memory software.

Rules:

```text
No hidden heap allocation.

Every dynamic structure has an explicit allocator or arena.

Memory subsystem owns its storage policy.

Mind stack storage is explicit.

Dynamic automata storage is explicit.

Actuation pending state is explicit.

Replay log storage is explicit.

Parallel staged surfaces are explicit.
```

Bare-metal profile must be able to run without a hosted heap.

---

## 23. Runtime failure doctrine

Runtime failures use Concept’s shared runtime failure substrate.

Stable machine/automata failure reasons should remain readable and fixture-pinned.

Existing machine reasons:

```text
machine result cannot be read before completion
machine decision transition has no enabled candidates
machine transition match found no matching case
invalid machine state reached
```

DragonGod.Kernel should add reason strings only deliberately.

Automata signals require reasons where they change behavior stack/control state.

No silent transition.

No reasonless stack mutation.

---

## 24. DragonGod v0 implementation slice

The full cathedral is not built in one phase.

Recommended first implementation slice:

```text
DragonGod.Kernel.Core
DragonGod.Kernel.Automata.Signal
DragonGod.Kernel.Memory minimal shell
DragonGod.Kernel.Mind static shell
examples/dragongod/patrol
```

### 24.1 P20 / P21 split recommendation

P20 should be this blueprint.

P21 should implement the first minimal kernel seed.

Minimal seed:

```text
module DragonGod.Kernel.Core
module DragonGod.Kernel.Automata
module DragonGod.Kernel.Memory
module DragonGod.Kernel.Mind
```

Types:

```text
AgentId
StateId
ActuationId
Clock
AutomataSignal
MemoryKey<T> design shell
Memory minimal struct
Agent minimal struct
World minimal struct
Mind minimal struct
```

Examples:

```text
static machine returning AutomataSignal::Succeed
manual Step loop
yield wait-until example
Memory read/write if available
```

Do not implement full dynamic graph in the first code slice.

---

## 25. Later implementation phases

Suggested roadmap:

```text
P20  DragonGod.Kernel full blueprint
P21  Kernel core seed: IDs, Clock, AutomataSignal, minimal Agent/World/Mind
P22  Memory v0: MemoryKey<T>, memoryRead/write, revision, changed keys
P23  Static Mind v0: concrete machine ticking and AutomataSignal handling
P24  Automata stack v0: Goto/Push/Pop/Succeed/Fail, keep-root-frame doctrine
P25  Decision subsystem v0: consideration, policy, decision memory
P26  Actuation subsystem v0: command concepts, host dispatch, pending completion
P27  Events subsystem v0: typed event bus and cursors
P28  Trace subsystem v0: TraceSink interface/dyn integration
P29  Persistence/replay v0: checkpoints and replay logs
P30  Dynamic automata graph v0: type-erased machine storage and ops
P31  Parallel staged tick v0
P32  Bare-metal/RISC-V DragonGod proof
```

This roadmap may shift as Concept capabilities mature.

---

## 26. Non-goals for the blueprint phase

Do not implement during blueprint creation:

```text
compiler changes
new language syntax
DragonGod runtime hooks in compiler
scheduler
async runtime
event loop
blackboard keyword
mailbox keyword
dynamic child machines
heap-owned machine frames
full dynamic graph
parallel runner
checkpoint serializer
replay driver
actuation host
trace sink implementation
```

The blueprint is conceptual.

Implementation comes later.

---

## 27. Final naming decisions

Locked public naming:

```text
DragonGod.Kernel
Automata
Automaton
AutomataSignal
Mind
Memory
MemoryKey<T>
memoryRead
memoryWrite
memoryChanged
World
Agent
Decision
Actuation
Events
Trace
Persistence
Replay
Parallel
Platform
```

Avoid public DragonGod naming:

```text
HFSM
Hfsm
Brain
BrainInstance
Blackboard
BbKey
bbGetOrDefault
AiStep
NodeRunner
IEnumerator
```

These may appear in historical notes about Dominatus, but not as native DragonGod.Kernel public names.

---

## 28. Closing doctrine

```text
Dominatus was the prototype.

Concept is the native substrate.

DragonGod.Kernel is the cathedral.

Machines are the stones.

yield is the breath.

AutomataSignal is intent.

Mind is execution.

Memory is context.

Replay is truth.

Trace is explanation.

Parallel is staged.

Persistence is resurrection.

DragonGod is not compiler magic.

DragonGod is explicit behavioral architecture in Concept.
```


---

## DG1 implementation seed status

Phase 20 / DG1 creates the first `dragon-god/` source tree and seeds the `DragonGod.Kernel` root with Core, Automata, Memory, and Mind module/type skeletons. Public names are pinned to Automata, Mind, Memory, and AutomataSignal; historical names such as HFSM, Brain, Blackboard, AiStep, and NodeRunner are not part of the public DragonGod.Kernel source surface.

DG1 is intentionally not the full kernel. It does not implement Memory storage, changed-key tracking, TTL, Mind ticking, Automata stacks, dynamic graphs, MachineOps erasure, actuation, events, trace, replay, persistence, parallel staged ticks, a scheduler, async behavior, blackboard/mailbox/event keywords, or DragonGod compiler hooks. The DG1 Concept friction log starts in `docs/DragonGod-Concept-Friction.md`.
