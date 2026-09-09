# EVT1 machine stack and completion direction

Status: R5h dyn-selected async construction implemented over the R5e stack law

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

R5f generates a parent continuation state, pushes the child, leaves a yielded
child on top, consumes its completion outcome once, and resumes the revealed
parent on a later explicit Step. A direct async call constructs an inline
operation; it does not run the body. A named operation may be adopted only by
explicit move. This is the same bounded LIFO and completion law, specialized
as generated frames; there is still no saved PC, futures framework, task
runtime, event loop, scheduler, hidden heap, or LIR.

R5g changes only how those parent states are produced. Reducible branches,
matches, loops, foreach iteration, and bounded local error handlers normalize
to ordinary named states and edges before lowering. A join is an explicit
state tag; a loop resumes through an explicit header/backedge; neither is a
saved instruction pointer. The capacity-eight LIFO stack, top-frame-only Step,
child completion transfer, and cleanup law are unchanged.

R5h changes only how a concrete async child constructor may be selected. A dyn
interface witness returns one ordinary concrete operation; await adopts its
frames and thereafter uses this same stack, completion slot, and result law.
There is no virtual child machine, per-Step witness dispatch, or second stack.
