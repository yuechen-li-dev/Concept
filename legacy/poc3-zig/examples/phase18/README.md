# Phase 18 examples: composable machines and runtime transitions

Phase 18 makes machines composable.

A child machine is stored as a by-value field. Parent machines explicitly `Step`
child machines. `Complete(child)` and `Result(child)` inspect child frames.
`State(machine)` exposes a numeric current-state id for tests, debugging, and
control loops.

`transition match` provides deterministic runtime branching. `transition decide`
provides deterministic utility selection. Runtime machine failures use the shared
`cpt_panic` path.

This is the substrate DragonGod can build on, not DragonGod itself.

## Non-goals

Phase 18 examples intentionally do not introduce:

- a scheduler;
- async;
- an event bus keyword;
- a blackboard keyword;
- a mailbox keyword;
- hidden heap allocation;
- a DragonGod runtime hook.
