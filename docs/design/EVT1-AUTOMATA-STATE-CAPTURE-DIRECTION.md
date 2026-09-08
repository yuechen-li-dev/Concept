# EVT1 automata state-capture direction

Status: R5b implemented; yield and effect reconciliation deferred

## Reconciliation

Concept EVT1 uses the Concept/Vulkan word `automata` for an outer persistent
composition unit and the PoC3 word `machine` for each contained independently
stepped state machine. A `state` is a named execution state inside exactly one
machine. These levels are not aliases and the canonical documentation does not
call them HFSMs.

PoC3 Phase 13 supplies the concrete pressure for state identity, literal
transition, Step, and transient state-body execution. Phase 18 supplies
machine-field and nested-machine pressure. Phase 19 pins re-entry and the law
that yield need not imply a scheduler or continuation frame. The Go lineage
supplies the outer automata grouping, multiple machine declarations,
deterministic identities, and DragonGod integration pressure. R5a merges only
the shared subset needed to establish storage authority.

## Explicit environments

```concept
automata Worker
with state
{
    ref Device device;
    owned Buffer scratch;
    int retryCount;
}
{
    machine Run
    {
        int attempts;

        state Idle
        {
            int observed = retryCount;
            transition Working;
        }

        state Working
        {
        }
    }
}
```

The compiler creates an explicit environment with stable identity
`Worker#state`; it does not infer one from uses. Its fields retain their
ordinary Concept types, ownership, provenance, copyability, layout, and Drop
obligations. All machines refer to the same environment. Each machine has a
separate inline storage record containing its current-state tag and declared
machine fields. State-body locals remain ordinary block locals.

The persistence classes visible in MIR are:

| Class | Lifetime | Visibility |
|---|---|---|
| `AutomataState` | automata instance | shared by contained machines |
| `MachinePersistent` | automata instance | declaring machine only |
| `TransientLocal` | one Step/body invocation | lexical state body |

No local is promoted between these classes by the Planner or backend.

## Construction, ownership, and lifetime

R5a retains the existing explicit instance declaration rather than inventing
constructors:

```concept
instance Worker worker(ref device, move scratch, 0);
```

Arguments correspond to `with state` fields in declaration order. Values copy
only when their ordinary type is copyable. `owned` fields require an explicit
move and become one reverse-order automata Drop obligation. Ref, scoped,
ref-struct, Span, and dyn fields reuse the R4 provenance model; an automata
instance cannot be used to escape its captured lifetime. Machine fields may
use a declaration initializer; copyable fields without one receive their
ordinary zero value. Non-copyable machine fields require an explicit direct
initializer.

The environment and all machine storage are inline. There is no hidden heap,
closure object, coroutine frame, registry, scheduler, async runtime, event
loop, or implicit capture.

## Stepping and transition

`Step(worker, Run)` enters the state selected by Run's current-state tag,
executes one state body, and returns. It does not step sibling machines.
`State(worker, Run)` reads the deterministic declaration-order tag. Symbolic
identity remains `Worker.Run.Idle` in MIR and plans.

`transition Working;` is a local-machine state change. Validation rejects
unknown or cross-machine targets. Lowering cleans transient locals, writes the
new tag, and returns from the current Step. Persistent automata and machine
fields remain live.

## Match and decide authoring

R5b prefers `transition match (value) { Pattern => State; }` for categorical
selection and `transition decide { State when guard score utility; }` for a
scored candidate set. Match reuses the ordinary exhaustive payload-enum
checker. Decide evaluates guards in declaration order, skips scores for false
guards, evaluates every enabled score once in declaration order, and selects
the first candidate with the maximum exact-typed `int` or `float` score.

Both forms are local Step-ending transitions: transient cleanup, state update,
return. Repeated targets and an explicit current-state target are ordinary
candidates. No enabled candidate and NaN float score are deterministic terminal
failures. Neither form introduces scheduling, suspension, allocation, a
runtime candidate collection, or new capture rules.

## Planner and C realization

Semantic analysis fixes persistence before planning. `AutomataPlan` records
an `InlineExplicitStruct` environment, `MachinePlan` records one explicit
current-state slot and `Switch` dispatch, and the artifact says
`scheduler: None`, `yield_strategy: Deferred`. The strict-C11 bootstrap emits
inline structs, deterministic enums, and one switch function per machine.
This is proof lowering, not a permanent choice against future direct-branch or
jump-table plans.

R5b retains match arms and decision candidates in MIR. `TransitionMatchPlan`
records categorical switch realization and scrutinee-once order;
`TransitionDecidePlan` records guarded straight-line hardmax, declaration-order
guard/score evaluation, first-maximum ties, panic policy, and the cleanup edge.
GenericC11 emits static branches and comparisons. Native target plans may later
record stronger eligibility, but R5b selects no branchless transform.

## DragonGod boundary

DragonGod remains a library/application consumer using Automata, Machine,
State, AutomataSignal, Mind, and Memory vocabulary. Its Decision, Actuation,
Events, Trace, Replay, Persistence, and graph subsystems do not become language
semantics. The older typed-signal automata path stays compatible during R5b;
its effect/actuator assumptions do not define the new Core environment.

## Future unification and deferred work

Automata capture, closure capture, callback environments, and deferred-callable
state should share one explicit environment model where their laws genuinely
match. R5a does not implement closures or callbacks to force that abstraction.

DragonGod Decision remains a later library-level stateful policy across Steps:
hysteresis, minimum commitment, temporal smoothing, tie memory, and policy
memory. None of that state is implicit in `transition decide`.

The conceptual next layer is deliberately non-normative: `score` is scalar
evidence/utility; `decide` is hardmax; future `infer` is a normalized soft
belief/distribution. R5b implements neither plain value-level `decide` nor
`infer`, softmax, sampling, temperature, TopK, or `transition infer`.

Recommended R5c scope is yield over the storage law established in R5a:
automata state, machine fields, and current-state tags survive; transient locals
do not.
Complete/Result, nested machine values, effects, and actuators remain separate
later decisions.
