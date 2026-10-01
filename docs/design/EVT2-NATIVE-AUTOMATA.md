# EVT2x native automata lowering

EVT2x6 establishes native bounded scalar pushdown Step through the real path:
validated source / MIR and Planner -> verified LIR -> ordinary AMD64 MachineIR
-> CMIRAMD1 -> Concept allocator and encoder -> executable memory.

> Existing C-backend machine semantics define Step boundaries and push/pop
> behavior during EVT2 bootstrap. Native traces must match the C oracle before
> optimization.

C Step is a void API. The oracle harness observes its existing yield marker and
reads the actual typed C instance to project Active, Yielded, or Completed. It
does not emulate transitions or replace any machine operation.

| Source operation | Step boundary and lifetime |
| --- | --- |
| push Child goto Resume | Store parent Resume, initialize/publish Child, return Active; child executes next Step |
| yield | Return Yielded; next Step starts the same top source state again |
| transition Target | Store top frame's state, return Active; depth/parent unchanged |
| neutral complete / pop in child | Destroy/remove child, return Active; parent executes next Step |
| neutral complete / explicit pop at root | Destroy/remove root, depth zero and completed true, return Completed |
| Step after completion | Completed, with no memory mutation |

The root-pop behavior corrects an inaccurate statement in this document's x5
version: the existing validator rejects a second terminal pop on a single path,
but does not reject a single root pop. The C oracle and native root-pop variants
now pin that behavior explicitly.

The parent continuation is an ordinary state ID in its existing frame. Machine
persistent fields remain independently live across suspension. There is no
separate continuation stack, saved program counter, copying of restored frames,
or same-Step child/parent execution for these operations.

Native Step selects its dynamic top machine tag, then that machine's state.
Invalid depth, tag, or state traps before frame reinterpretation or unsafe
addressing. Capacity failure and child scalar-initializer failure preserve old
live frames, continuation, tag, depth, and surrounding sentinels. Typed scalar
destruction regions precede reduced-depth publication; inactive slots remain raw
storage and are neither copied nor cleared.

State bodies reuse the ordinary lirBuilder. Only frame bindings and machine
control transfers have specialized handling. Existing finite-machine ABI and
execution stay intact. The source-only `language/evt2/machines/parent_child.concept`
and `pushdown.concept` fixtures run through `concept lir`, `concept machineir`,
and `concept amd64` without a host instance wrapper. The legacy EVT1 specimen's
Main wrapper still needs ordinary InstanceDecl/call lowering in EVT2e; its machine
declaration is qualified separately and is no longer rejected for push/pop.

## Shared backend repair, written in Concept

The source Step exposed two general backend limits. Liveness was restricted to
16 blocks / 32 registers, and check continuations appended by MachineIR lowering
created artificial overlapping linear intervals. Standard.Backend.AMD64 now:

- Supports 128 blocks and 512 virtual registers with bounded bitset liveness
  (17 positive 31-bit words per block), still 512 instructions and 64 slots.
- Groups blocks split from the same LIR block, moves their instruction ranges,
  and remaps branch edges before interval calculation.
- Retains the existing register pool, linear allocator, explicit
  RegisterExhausted boundary, no spills, and ordinary AMD64 encodings.

The bridge schema and Go MachineIR lowerer/verifier are unchanged. These are
ordinary-function repairs, with Concept-written tests at the high register/block
boundary and for branch remapping. The generated nested Step currently has 267
virtual registers and 106 MachineIR blocks.

## Reconciled features and self-hosting assessment

Current main includes Vulkan reconciliation and innate concepts. External `on`
reactions use StepOutcome and unordered matching/ambiguity semantics; guarded
transition-match uses ordered first-match semantics. They are not replacements
for terminal source push/pop. Their native forms remain explicit boundaries.

C terminal states invoke a settling loop that can complete several suspended
frames in the same Step. Native finite terminal support stays in its existing
path; pushdown terminal states diagnose EVT2_UNSUPPORTED_PUSHDOWN_TERMINAL rather
than silently treating an empty body as an active state.

Innate concepts currently observe declarations and types, not machine statements
or generated CFGs. Moving this AST-to-LIR integration into the innate module
would require a new typed statement/IR bridge and would otherwise duplicate the
existing expression lowerer. x6 keeps that narrow integration in the bootstrap
compiler. New reusable backend algorithms and their direct tests are authored in
Concept, extending the existing CMIRAMD1 strangler seam. A later lowering
migration should expose typed MIR/LIR data and run shadow agreement before
switching authority and deleting the Go implementation.

## Qualification limits

Windows AMD64 executable-memory execution is qualified. Native Linux/macOS
execution is unverified. Normal/Verify C oracle and Concept-backend hosts pass;
the native LIR path currently uses one conservative guarded policy, without a
separate native Verify observation runtime. Native owned activation fields,
non-neutral outcome payloads, Drop probes, signal reactions, guarded
transition-match, and pushdown terminal settling remain explicit boundaries.
EVT2e calls, spills, callee-saved registers, and outgoing ABI work are not started.
