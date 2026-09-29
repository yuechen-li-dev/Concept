# Phase 19 examples: yielding machines

Phase 19 adds bare `yield;` as explicit machine-step suspension.

`yield;` exits the current `Step`.

The machine remains incomplete.

The machine remains in the same state.

The next `Step` re-enters the same state body from the beginning.

`yield;` is not async, not a scheduler, not a generator, not `yield return`, and not continuation resume.

This lets DragonGod-style behavior be modeled as explicit repeated `Step` calls over ordinary machine frames.

Phase 18 plus Phase 19 gives core Concept the substrate for tickable behavior:

- nested machine fields;
- explicit `Step(child)` ticking;
- `Complete(child)` and `Result(child)` observation;
- runtime `transition match`;
- deterministic runtime `transition decide`;
- numeric `State(machine)` observation;
- shared runtime failures through `cpt_panic`;
- bare `yield;` for state-level suspension.

DragonGod can later provide library/runtime orchestration over these primitives. Core Concept still does not contain a scheduler, async runtime, generator protocol, event loop, blackboard keyword, mailbox keyword, dynamic child-machine list, heap-owned machine model, or DragonGod runtime hook.

## Files

- `wait-until-child-complete.concept` shows a parent ticking a child until completion, yielding while the child is incomplete.
- `polling-loop.concept` shows an explicit caller-controlled polling loop over `Step(task)`.
- `long-running-behavior.concept` shows a multi-tick action built from child-machine progression and state re-entry, not continuation resume.
- `yield-transition-match.concept` shows yield before a later `transition match`.
- `yield-transition-decide.concept` shows yield before a later deterministic `transition decide`.
- `state-observation-after-yield.concept` shows `State(machine)` before and after a yielding step.
- `runtime-failure-notes.concept` documents that `yield;` itself does not panic, while `Result(machine)` before completion still does.
- `dragon-god-readiness.concept` summarizes the DragonGod v0 readiness boundary without adding DragonGod semantics.
