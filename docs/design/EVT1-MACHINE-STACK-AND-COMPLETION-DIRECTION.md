# EVT1 machine stack and completion direction

Status: R5e implemented

R5e finishes the partial substrate already present in Concept. The older
signal automata compiler had a bounded continuation stack, PoC3 Phase 13/18 had
explicit Step/Complete/Result machine frames, and DragonGod DG5 had four fixed
StateId/Reason frames. None represented canonical R5 machine-persistent
storage, so R5e specializes one bounded stack per automata instance.

## Lifecycle law

| Operation | Effect on active frame |
|---|---|
| Step | execute the top frame once |
| yield | keep frame and state; return from Step |
| transition | change only top-frame state; return from Step |
| push | record parent state; initialize and add child |
| pop | clean/remove top frame with Neutral outcome |
| complete | clean/pop with Neutral or Success outcome |
| fail | clean/pop with Failure outcome |

Completion is semantic pop. Root pop empties the stack. Child pop reveals its
parent. The child outcome remains in an explicit per-machine result slot and is
unavailable before completion.

## Representation and ownership

Generic C uses capacity-eight inline arrays specialized for every machine, a
tag array, depth, shared state, and typed last outcomes. A frame's state and
fields are independent even for recursive same-machine pushes. Identity is
declaration tag plus stack index, never an address. Overflow and malformed
underflow paths fail deterministically.

Owned fields survive yield/transition and drop when their frame is destroyed.
Completion evaluates a payload before cleanup. Explicit `move` marks its frame
source moved, so ownership transfers into the outcome without a second drop.

## Remember, Resume, and async

Push already remembers the parent. `goto ResumeState` is the continuation;
there is no suspended instruction pointer. DragonGod/Oct Remember/Resume may
name or apply policy to this workflow without another stack. Current Oct
`remember`/`resume` is one overwriting state-target slot, cleared by resume;
that policy is not promoted into Concept core.

A future async compiler can generate a parent continuation state, push a child,
Step it until pop, inspect its outcome, and Step the revealed parent. R5e adds
no async/await, futures, task runtime, event loop, scheduler, or LIR.
