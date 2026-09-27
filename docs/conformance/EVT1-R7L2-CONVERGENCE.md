# EVT1 R7l2 convergence log

Baseline: `58c180966e0680968d8acb5b643cbb7ce4658308`, compiler
`concept-evt1-stage0-go`. R7l commits: `a3a07eb`, `c6b332e`, `2d0f38f`,
`58c1809`.

| Blocker | Reproducer and root cause | Fix and scope |
| --- | --- | --- |
| One asm operand | Second `in` failed with `ASM_OPERAND_LIMIT`; AST, MIR, and helper accepted one pointer | Ordered typed operands and a single deterministic 64-bit-slot frame ABI; AMD64 native test covers two inputs, two outputs, and fixed registers |
| CPUID needs four fixed registers | Instruction implicitly reads EAX/ECX and writes EAX/EBX/ECX/EDX | Bounded fixed-register declarations, shared input/output register rules, RBX preservation, typed `CpuidResult`; native CPUID test |
| Artifact helper identity drift | Symbol hash included resolved `Type`, which changes between local and imported views | Hash source-stable operand mode, name, fixed register, instruction and span; artifact-only asm consumer passes |
| Worktree baseline checkout bytes | Managed worktree checkout converted fixture bytes to CRLF and omitted ignored tinyxml2 upstream files; checked-output and ABI tests failed | Copied fixture bytes and ignored upstream fixture from the original repository checkout into the isolated worktree; no compiler workaround |
| Parallel Zig cache collision | Concurrent root/legacy Zig run reported an unexpected standard-library load error | Serial root rerun passed; legacy run passed |

The original checkout contained unrelated uncommitted compiler/test edits.
R7l2 was built in a managed isolated worktree at the baseline commit. The
original checkout was not modified by R7l2.
