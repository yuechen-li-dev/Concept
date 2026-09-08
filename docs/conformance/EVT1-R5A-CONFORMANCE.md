# EVT1 R5a automata state conformance

R5a contains 20 required `PASS` cases: 12 accepted programs and 8 statically
rejected programs. The corpus is `language/evt1-r5a/core`; executable checks
are in `internal/concept/r5a_conformance_test.go`.

Accepted cases cover the canonical hierarchy, value/const/ref/owned/dyn/Span
state, two independently stepped machines sharing one environment,
machine-local fields, deterministic transitions, transient local recreation,
owned persistent cleanup, and caller-visible state inspection. All 12 paths
compile and execute as strict C11.

Rejected cases cover implicit capture, automata lifetime escape, missing owned
move, duplicate machine/state declarations, unknown transitions, sibling
machine-field access, and attempts to use a local from another state body.

MIR inspection pins `AutomataState`, `MachinePersistent`, and `TransientLocal`
classifications plus stable `Automata#state` and
`Automata.Machine.State` identities. Planner inspection pins inline explicit
storage, one current-state slot per machine, switch dispatch, no scheduler, and
deferred yield. Generated-C inspection pins locals inside step functions,
reverse-order persistent Drop, transition cleanup, and absence of allocation,
coroutine, and scheduler mechanisms.
