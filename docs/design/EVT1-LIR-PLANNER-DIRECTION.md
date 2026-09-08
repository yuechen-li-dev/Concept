# EVT1 LIR Planner direction

Status: R5d yield/foreach planning consumer; LIR is future work

## Authority pipeline

```text
MIR
    semantic truth
SemanticFactSet
    proven legality evidence
Planner
    selected realization strategy
LoweringPlan
    deterministic inspectable contract
LIR
    future target-neutral machine-near operations
MachineIR
    future target-specific operations and registers
Machine code
```

The Planner is separate from semantic analysis, optimization, and emission.
MIR says what the program means. Facts say what is proven. The Planner chooses
among strategies legal under those proofs and records why. A future LIR will
spell out loops, loads, stores, branches, and accumulators implementing the
validated plan; it will not rediscover language meaning.

R4l plans existing tensor traversal, bounds guards, deterministic cleanup,
failure branches, aggregate/view representation, and static witness dispatch.
GenericC11 preserves the current correct backend path. X86_64_Generic and
AArch64_Generic prove target-independent planning without emitting machine
code. Unknown facts select conservative fallback. No allocation, copy, or
ownership transfer may be introduced silently.

R5a adds automata as a bounded Planner consumer. Semantic MIR, not the Planner,
classifies `AutomataState`, `MachinePersistent`, and `TransientLocal` storage.
`AutomataPlan` records an inline explicit environment; each `MachinePlan`
records its current-state slot, initial symbolic identity, transition edges,
and switch dispatch. R5d selects `ReenterStateFromStart` for bare yield and
explicitly records no scheduler or coroutine frame. A later Planner may select
branches or jump tables, but it cannot invent persistence or lift locals.

R5b adds explicit `TransitionMatchPlan` and `TransitionDecidePlan` records.
Match planning preserves categorical arms, one scrutinee evaluation, local
targets, panic policy, and cleanup. Decide planning preserves guard and enabled
score source order, exact score type, declaration-order-first-maximum ties,
no-enabled panic, and cleanup. GenericC11 uses switch/branches and static
hardmax temporaries. Native target records do not yet select branchless or
vectorized decisions; purity is not required, and Unknown purity therefore
forbids reordering. Future optimization may become eligible only when semantic
facts prove that guards and scores are pure.

R5c adds `InferencePlan` and `TransitionInferPlan`. They retain stable
softmax, max subtraction, fixed candidate count, inline storage,
no-enabled/NaN/infinity policy, cleanup, and explicit HardMax. Scalar
realization is selected; SIMD and hardmax-softmax elimination are not. A later
Planner may recognize linear models, tiny MLPs, lookup models, decision trees,
or quantized inference without changing distribution semantics.

R5d adds `YieldPlan` and `ForeachPlan`. Yield planning preserves the current
state tag and persistent storage, drops transient locals, and selects a direct
Step return. Foreach planning preserves exactly-once source evaluation,
value/ref/ref-const mode, deterministic item/iterator cleanup, and an inline
or explicit-protocol iterator strategy. Contiguous builtins record direct-index
eligibility without selecting a distinct optimization. Neither plan admits
allocation, a scheduler, a continuation frame, or hidden iterator persistence.

The 64-byte alignment of fixed inline storage remains a semantic guarantee
because current storage binding and qualified facts rely on it. A target's
preferred alignment is merely planning policy and cannot reduce proven
alignment.

## Tensor example

```text
MIR:  TensorContract(output C, inputs A and B)
Facts: fixed shape, contiguous, aligned, C disjoint from A and B
Plan: DirectLoopNest; vectorization Eligible; selected false
Future LIR: ordered free loops, reduction loop, loads, multiply/add, store
```

The fact set proves legality. The Planner never decides that pointers
"probably" do not alias. If disjointness or another optional proof is Unknown,
the plan remains direct and conservative.

## Validation and artifact

`concept-evt1-plan.v1` includes compiler, module/MIR identity, plan identity,
target, profile, policy, function plans, strategies, runtime guards, and fact
references. Validation rejects stale MIR identity, missing evidence, invalid
witnesses/tensors/cleanup, removed guards, unsupported targets, SIMD selection,
and nondeterministic plan identity.

The CLI surface is `concept plan <file>` beside `check`, `mir`, and `emit-c`.

## Non-normative reverse direction

Forward tooling may eventually implement `MIR -> Plan -> LIR`. Reverse tooling
might infer an implementation plan from LIR and recover higher-level MIR. That
is decompiler research direction only; R4l implements neither LIR nor reverse
translation.

## Deferred

R4l/R5d deliberately exclude SIMD generation, vectorization transforms, tiling,
fusion, SSA, virtual registers, register allocation, MachineIR, LLVM, MLIR,
native encoders, generalized alias solving, allocation policy, decompilation,
continuation/resume frames, scheduler policy, generator yield, branchless hardmax transforms, sampling,
and inference-model recognition.
