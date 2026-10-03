# EVT2e3 — Cathedral Win64 ABI and call-clobber planning

Baseline: `ea8f57cba65847e8466233f74d538c11d207b4dc`.
Compiler: `concept-evt1-stage0-go`.

This qualifies **planning**, through the real Concept backend path. It does not
qualify native calls. The final commit and clean-worktree status are recorded in
the handoff report.

## ABI authority and rules

`libraries/Standard/Backend/Win64ABI.concept` supplies checked record tables and
planner/validators. It imports the existing schema's canonical Register and
CallValueType identities. There is no second register universe or persisted
physical ABI artifact. The rules follow Microsoft's
[x64 calling convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention)
and [register conventions](https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions).

| Rule | Qualified plan |
| --- | --- |
| Integer arguments 0..3 | RCX, RDX, R8, R9 in that order |
| Later arguments | Symbolic outgoing slot `argumentIndex - 4`; eight-byte cell per slot |
| Home/shadow space | Explicit 32 bytes even for zero arguments |
| Scalar result | Fixed physical def RAX -> result virtual identity, with logical width retained |
| Void result | No result constraint; unused result descriptor remains Void/-1/0 |
| Caller-saved GPR order | RAX, RCX, RDX, R8, R9, R10, R11 |
| Allocatable callee-saved GPR order | RBX, RBP, RSI, RDI, R12, R13, R14, R15 |
| RSP | Fixed nonvolatile stack state; rejected as an allocated value or move source |
| FLAGS | Explicit clobber; a pre-call dependency cannot satisfy a later use |

The tables are ordinary checked data returned by `Win64Registers`. The exact
four-element argument table type has a StaticExtent<4> assertion. The compact
table, placement, move and preservation records have PlainData assertions via
the existing typed Verdict vocabulary. Mutable CallPlan's PlainData observation
is Unknown under the current predicate; no new semantic grant was invented to
force that assertion. Planner and validator operations return Result with
ABIError; NoAllocation assertions cover the real planning call graph.

Integer and bool arguments are right-justified logical values in a canonical
GPR. Widths 1/2/4/8 and signed/unsigned/bool classes survive placement. The plan
does not require sign or zero extension into the entire 64-bit register: upper
bits are outside the narrow ABI value. Bool is already a checked bool in LIR.
Cycle temporaries preserve a full eight-byte GPR before narrower final moves.
The primitive descriptor also qualifies an Address/u64/8-byte class in integer
registers. CMIRAMD3 source-call admission still excludes address-like arguments;
this is a descriptor-level qualification, not new pointer-call source support.

Float/aggregate classes, indirect/varargs/tail forms and unknown conventions
return explicit typed planner errors. XMM, sret, external-call admission and
alternate ABIs are not added.

## Derived state and pipeline

CMIRAMD3 remains version 3, hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.
BridgeSchema, both generated codecs, schema metadata and semantic call printing
are unchanged. Go continues to carry the checked target, ordered virtual values,
classes, result and provenance. No derived physical locations enter the bridge.

The selected Concept path is:

```text
generated decode + complete call admission
  -> bounded ProjectedCall in MachineInstruction
  -> block layout and call-aware virtual intervals
  -> fixed ABI argument-use and result-def plans before assignments
  -> deterministic physical allocation
  -> CFG live-across classification + verified parallel argument moves
  -> preservation/callee-save/temp/outgoing-area summary
  -> AMD64_UNSUPPORTED_CALL_FRAME_LOWERING
```

CallPlan contains the convention, stable borrowed target identity, call position,
ordered ArgumentPlacement records, result/RAX constraint, shadow bytes, stack
argument count/size, FLAGS clobber, ordered moves and symbolic temporary count.
Each placement identifies the argument index, source virtual value and exact
class/width, physical register or symbolic outgoing slot. The allocator derives
and checks the fixed constraints as input policy, before choosing general
assignments. The later move planner realizes call-point locations as metadata.
The scalar result remains an RAX def even if its general virtual assignment is
another register; result-copy emission is deferred.

The original five-register pool remains unchanged for no-call functions. Call
functions can additionally use the eight callee-saved candidates, in table
order, when the pool is exhausted. This admits six-argument planning without
general spilling. There is no special callee-saved preference for live-across
values: the planner reports caller-save requirements explicitly. Used-callee-
saved summaries are rebuilt from actual assignments in canonical table order.
RegisterExhausted remains the bounded allocation failure when all candidates
are occupied; this slice does not promise arbitrary register pressure.

## Call-aware liveness and preservation

The existing bounded CFG fixed point now includes every call argument use;
interval construction touches those values at the call point. LiveAcrossCall
starts from the block's computed live-out and walks instructions backward to
the call. It removes subsequent defs and adds ordinary/call uses, including
two-address destination uses. The call result is removed from the live-after
set because it is born after the call. Dead arguments are not preserved merely
because they are arguments, and nonlocal CFG edges use existing liveness.

For every live-across virtual value, a caller-owned Preservation row records:
virtual ID, logical width, address/GPR role, assigned canonical register, call
position, interval start/end and classification. Caller-saved assignments are
MustPreserve; callee-saved assignments are SafeInCalleeSaved with a used-save
summary. SafeInCalleeSaved requires a future function save/restore; it is not
permission to execute the current function. OutsideClobbers is a reserved
sentinel, not an admitted live allocated GPR in this 15-register vocabulary.
Wrong classification, RSP/unknown registers, invalid width/range and insufficient
caller output capacity reject. No preservation store/reload is inserted.
EVT2e4 must preserve these values before the argument moves: fixed argument
destinations can overwrite caller-saved assignments even before CALL executes.
Allocation's preservationCount is a count of requirements across sites, not a
final spill-slot count or a storage-reuse decision.

The source fixture `Across` computes `local=a+b`, calls Id(c), then uses local.
Its pre-call load v4 is live across call position 8: width 4, R10,
MustPreserve, interval 7..9. The result v5 is excluded. A checked allocation
fixture changes that value's assignment to R12 and qualifies SafeInCalleeSaved
plus the used-R12 save summary. This is planning evidence, not native behavior.

## Parallel move algorithm and verifier

Stack arguments are captured first so a register shuffle cannot destroy a
stack source. Identity register moves are omitted. Pending destinations are
visited in argument order; a destination can be written when no pending
physical source still needs it. A cycle saves the first pending destination's
full GPR to a symbolic temporary and replaces dependent source edges with that
temporary. No scratch register is selected. Temporaries have identities, not
final frame offsets; EVT2e4 must supply eight-byte storage.

The independent verifier replays original source identities and available
widths through all moves. It checks exact argument destinations/classes,
stack slot bounds/count/size, single writes, initialized temporaries and no
wider read from a narrowed register. It rejects missing moves/unresolved cycles,
destructive overlap, unknown/RSP registers and two distinct virtual sources
claiming the same physical register. Repeated positions for one virtual value
are legal.

Representative pinned plans:

```text
0 args: shadow=32, stack-bytes=0, no argument constraints
4 args: v0->RCX, v1->RDX, v2->R8, v3->R9
5 args: previous four plus v4->stack[0], stack-bytes=8
6 args: previous four plus v4->stack[0], v5->stack[1], stack-bytes=16

RCX<-RDX, RDX<-RCX:
  RCX -> temp[0] width=8
  RDX -> RCX width=4
  temp[0] -> RDX width=4

RCX<-RDX, RDX<-R8, R8<-RCX:
  RCX -> temp[0] width=8
  RDX -> RCX width=4
  R8 -> RDX width=4
  temp[0] -> R8 width=4

Across:
  arg0 v2:i32 width=4 -> RCX
  return RAX -> v5 width=4
  move RAX -> RCX width=4
  live v4 width=4 R10 MustPreserve range=7..9
```

## Qualification fixtures and CLI

`TestEVT2e3Win64PlansMovesAndLiveness` compiles the actual Concept backend with
`-std=c11 -pedantic-errors`, in Normal and Verify. Its host provides input and
an independent concrete width-limited move-value oracle; it does not implement
ABI placement or cycle policy. Qualification covers:

- Exact tables, zero through eight arguments, void/scalar/bool results, all
  signed/unsigned integer widths, Address descriptor placement.
- Float and aggregate arguments, float result, indirect/varargs/tail forms,
  invalid convention, over-capacity input and malformed placement/return/home/
  stack/FLAGS plans.
- Acyclic moves, two/three-way cycles, mixed bool/64-bit cycle, stack-source
  aliasing, repeated source values and all 256 source combinations among the
  four integer argument registers, checked by concrete replay.
- Nine actual artifact-only source-call cases from `evt2e3_calls.concept`:
  0/1/4/5/6 arguments, void, bool, repeated source and live-across-call.
- 100 identical placement/move/liveness/save-summary/debug observations per
  source case, 100 cycle repetitions, and exact Normal/Verify canonical output
  agreement. Result values are never classified as preexisting live values.
- Callee-saved allocation under six-argument pressure, forced R12 classification,
  stale FLAGS rejection, invalid preservation classification and insufficient
  preservation output capacity. EmitFunction stops without changing its output
  sentinel buffer for every call case.

`TestEVT2e3CLIDebugPlans` pins readable zero/four/five/live diagnostics using the
ordinary `concept amd64` command. The five-argument specimen puts Caller first
so the driver reaches its call plan before trying the independently unsupported
incoming-stack encoding of Target. This preserves that older boundary honestly;
no native call or five-argument native callee is claimed. LIR/MachineIR commands
and the original call fixture remain qualified.

The selected call stop is now:

```text
AMD64_UNSUPPORTED_CALL_FRAME_LOWERING:
  verified call plan requires preservation storage and call frame realization
```

The unchanged bridge BackendError::UnsupportedCallLowering tag carries this
later Concept diagnostic. ABIError is separate transient derived-state error
data. There is no silent wire change to add a new BackendError case. No CALL
rel32, symbol fixup, final RSP outgoing offsets, prologue/epilogue, save/restore,
general spill insertion or native call execution is implemented.

## Timing and bounded metadata

Focused, unoptimized C11 harness observations (informational only):

| Measurement | Normal | Verify |
| --- | ---: | ---: |
| 10,000 six-argument ABI plans | 8 ms | 9 ms |
| 256 four-register source move plans | 3 ms | 3 ms |
| 900 full artifact/allocation/liveness/emit-stop cases | 6,710 ms | 6,875 ms |
| 1,000 Across liveness passes | 15 ms | 14 ms |
| Paired call-use-disabled liveness control | 12 ms | 13 ms |
| Observed call-use delta | 3 ms | 1 ms |

The paired control is a synthetic ComputeLiveness input with the same projected
CFG and the call argument count temporarily zero; it is not an admitted source
call contract. These noisy observations make no throughput or native-call claim.

C11 sizes: CallRequest 172 bytes, CallPlan 660, ArgumentPlacement 32,
ArgumentMove 28, Preservation 32, BackendFunction 155,228 and Allocation 6,692.
One call plan is processed at a time. A caller-owned buffer admits at most 512
preservation rows (16,384 bytes). Plans have at most eight arguments, 12 moves
and four symbolic temporary slots; actual four-register cycles need at most
two temporaries. ProjectedCall adds 80 bytes per bounded instruction, 40,960
bytes over 512 instructions. Allocation metadata adds 36 bytes. No heap or
giant collection of per-call preservation buffers is created. Artifact sizes
and generated bridge declaration counts are unchanged by this slice.

## Stage-0 change classification

- `cmd/concept/amd64.go`: orchestration/presentation of Concept-produced plans.
  No placement, register-set, liveness or move-cycle algorithm.
- `cmd/concept/amd64_test.go`: orchestration of normal CLI qualification and
  compatibility update to the later named boundary.
- `internal/concept/win64_call_plan_test.go`: test orchestration/checked fixture
  input and independent native value oracle.
- `foundation_conformance_test.go`: test orchestration helper optionally returns
  output and accepts strict compiler flags for cross-policy comparison.
- `machineir_bridge_test.go`: compatibility seam removes a duplicate host
  typedef now supplied by the generated backend header. Native oracles unchanged.

## Validation and handoff

Baseline full Go suite passed in an isolated clean worktree: internal/concept
419.417 s, CLI 6.232 s, GPU-free Vulkan 0.362 s. Baseline vet and focused race
passed. Standard 42, DragonGod 23 and Golden 130 passed in both Normal/Verify.

Final full Go suite passed: internal/concept 460.083 s, CLI 17.704 s, GPU-free
Vulkan 0.396 s. Focused ABI/move/liveness/CLI checks, native frozen oracle/AddMax,
vet and focused race passed (38.701 s for the focused race lane). Standard 42,
DragonGod 23 and Golden 130 passed in both Normal/Verify, zero failures.
Concurrent library and focused runs make full-suite times unsuitable as a
compiler performance comparison. Full suites include the semantic corpus, GPU-free
Vulkan, EVT2/EVT2x native behavior and 100-run existing native byte determinism.
Corpus counts remain 436 valid, 357 static-invalid, 15 runtime-negative, five
compatibility and four expected-divergence specimens. New bounded ABI fixtures
live in internal/concept/testdata and the native qualification host.

Touched authored `.concept` files pass format and lint with repository library
roots. The touched `.concept_test` passes format; direct lint fails identically
on baseline and final because generated Standard.Build.Metadata is absent.
Standard's ordinary Normal/Verify test runner supplies that metadata and passes
all 42 facts. No unrelated module or language change was made to bypass this.
Diff whitespace checks pass. Both Zig suites are skipped because the frozen
compiler/build paths are unchanged.

The authority ledger, migration matrix, bridge/LIR/backend design and EVT2e
convergence notes record Concept ownership and deferred frame/spill/encoding.

SUCCESS — EVT2e3 Cathedral Win64 ABI and call-clobber planning qualified.
