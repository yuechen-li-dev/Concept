# EVT2x6 convergence ledger

Working baseline: d576339ec5ff1a8c78805eb9daa3308c94c93eaa (merged main).
Requested clean x5 HEAD: bca0d0adf242f5b79c36acd922d790be33f5764b.
Exact x5 addressing commit: 2fb04444d26e3c8d996aa4c74547318bb96b2264.
Compiler: concept-evt1-stage0-go.

## Removed blockers

1. C oracle pinned before implementation: push/pop stop Step; continuation is an
   ordinary parent state ID; child complete implicitly pops; yield re-enters the
   same state; root pop completes. The root-pop contradiction in the initial
   request was explicitly resolved with the user in favor of C semantics.
2. Source push/pop no longer stops at EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP. Complete
   activation CFG uses ordinary state-body lowering and existing addressing.
   Typed lifetimes, publication order, dispatch, guards, and layout are verified.
3. CapacityExceeded isolated to Concept backend's 16-block / 32-register
   liveness limit; bounded bitsets now support 128 blocks / 512 registers.
4. RegisterExhausted isolated to artificial intervals spanning remotely appended
   check continuations; Concept block grouping/remapping fixes layout before
   intervals. No spills, register-pool expansion, new instruction or bridge
   schema, or automata-specific backend handling was introduced.
5. Exact C/native traces agree through depth three, yield/transition, repeated
   activation, completion, and interleaved instances. Source stride-12 parity and
   retained x5 addressing probes pass too.

## Reuse and self-hosting decisions

Fast-forwarding to main includes Vulkan reconciliation, terminal states,
guarded transitions, input reactions, and innate concepts. Input StepOutcome
and terminal settling have distinct control-flow semantics, so native pushdown
retains explicit boundaries for them. Existing finite support is preserved.

Innate concepts currently observe declarations/types, not machine statements or
CFGs. Moving source lowering there would require a typed bridge and agreement
protocol. The local Go AST/LIR integration is retained; new backend layout and
liveness algorithms, and their direct tests, are written in Concept. A future
typed MIR/LIR seam can shadow, switch, and delete the bootstrap implementation
without duplicating expression lowering.

## Qualification

Source/native acceptance passes under Normal and Verify C-hosted runs on Windows
AMD64. Complete LIR/layout, MachineIR/CMIR, physical-assignment, native-byte and
trace determinism pass 100 runs. 17 malformed-LIR cases and every negative
frame/sentinel snapshot pass. Owned Drop remains an explicit boundary; formal
generated-Step NoAllocation and native Linux/macOS execution are not claimed.

Initial main gates: focused EVT2/machine/corpus/innate tests passed; Standard
Normal/Verify 39 each, DragonGod 23 each, Golden Verify 130. Initial
`package test Golden` reported PACKAGE_DEPENDENCY_MISSING because Golden is test
programs, not a registered package. Correct `test libraries/Golden` passes 130 in
Normal. This was command selection, not an ignored product or host failure.

## Final closeout

| Gate | Result |
| --- | --- |
| Full Go suite (`-count=1`) | PASS, internal/concept 315.308 seconds; profile/vulkan also passes |
| go vet ./... | PASS |
| Focused EVT2, machine, semantic corpus, innate and reconciliation tests | PASS |
| Final x6 oracle/native/verifier acceptance | PASS, 13.652 seconds |
| Standard Normal / Verify | 42 / 42 pass; three benchmarks each |
| DragonGod Normal / Verify | 23 / 23 pass; one benchmark each |
| Golden Normal / Verify | 130 / 130 pass; two benchmarks each |
| Root / legacy Zig suites | Both PASS |
| Source Parent/Child and nested CLI lir / machineir / amd64 | All six PASS |
| Existing EVT2d, finite machines, root Init and x5 native addressing | PASS in focused and full Go suites |
| Change-caused or retained host-only failing gates | None; Golden command-selection correction described above |

Commits: `bfdc563172b94c1ddb50223e6178fe84047d27ab` contains the Concept-written
backend repairs and tests. The source/conformance commit carrying this document
is titled `Lower bounded source push/pop to verified native CFG`; its final SHA
is recorded in the final report rather than embedded recursively in itself.

Worktree: `codex/evt2x-native-automata`, based on merged main. All milestone files
are committed; final status is checked after commit. Main is not advanced by this
worktree. No remote publication is performed. EVT2e is not begun.

SUCCESS — EVT2x native bounded pushdown automata lowering established.
EVT2x COMPLETE.
