# EVT2e4 — native stack frames and concrete call preservation

Baseline: `24ea223e2f372be9418c1c02de92e5df44c5128e`.
Compiler: `concept-evt1-stage0-go`.
CMIRAMD3 remains version 3, schema hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.

## Ownership and real path

`Standard.Backend.AMD64` owns frame geometry, spill/save/temp assignment,
outgoing storage, action insertion, prologue/epilogue and structural admission.
`EmitFunction` and `EmitFunctionWithWorkspace` decode, lay out blocks, allocate,
derive ABI plans, finalize storage and qualify every call action before returning
`AMD64_UNSUPPORTED_CALL_ENCODING`. Output bytes remain untouched on that stop.
The abstract target remains the checked CMIRAMD3 identity and source/LIR provenance.
No CALL rel32, relocation, external lookup or native call execution is implemented.

The old `FinalizeFrame` wrapper uses the same `RealizeFrame` geometry for
non-call functions. Its public Result remains a compatibility seam; the richer
frame APIs return `FrameError` with named capacity, alignment, width, overlap,
offset, unresolved spill/temp, save/restore, action and checked-input errors.
The C bootstrap prints those precise frame errors when qualification fails.
The generated bridge's BackendError enum is unchanged.

The checked producer currently leaves legacy `Frame.HasCalls` false and shadow
size zero even when CALL instructions exist. Concept derives call presence
from verified instructions and cross-checks its allocator count; that producer
summary does not select native geometry.

## One geometry

All offsets use **RSP after the prologue subtraction**. Regions occur in order:

1. OutgoingShadow: [0,32) for a function containing any call, otherwise empty.
2. OutgoingArguments: eight-byte cells at 32,40,48,56 for arg4..arg7.
3. LocalSlots: existing logical slots, ordered by original slot index.
4. SavedRegisters: only used callee-saved GPRs, in Win64 table order.
5. SpillSlots: ascending virtual value identity.
6. MoveTemps: ascending symbolic temporary identity.
7. AlignmentPadding: final padding.

Region descriptors precede concrete slot offsets. Each nonempty storage region
includes its own leading/interior alignment gaps. The final padding field
reports only the tail padding; unaligned size includes those internal gaps.
An incoming-indirect logical slot remains external storage and has offset -1.
Local logical size is checked independently against the transported local size.

The function reserves the **maximum outgoing area once**, not the sum of calls.
Zero-argument calls still reserve 32 bytes. No action adjusts RSP at a call site.
Each slot retains its category, identity, size and alignment, independently of
the map from vreg/register/temp/outgoing identities to slots.

Entry RSP is 8 mod 16 because the return address is already present. A frame
containing calls has final size 8 mod 16, making post-prologue RSP 0 mod 16.
Final size is the smallest size at least the unaligned size with that residue.
A no-call function without storage retains size zero and its existing bytes.
All additions check the bound before arithmetic; maximum planned size is 65,528
bytes. This is a metadata bound, not native qualification of large stack
allocations or a stack probing implementation.

RBP remains allocatable and preserved like other callee-saved GPRs; no frame
pointer mode is introduced.

## Spills, saves and temporaries

One dedicated slot per virtual value requiring MustPreserve anywhere in the
function. No lifetime-based slot reuse. Each call stores that site's live values
before any ABI argument move, then reloads them after result capture. The same
slot is reused for the same virtual value at successive calls, with a fresh
store/reload pair each time. SafeInCalleeSaved values require no call-site spill.
The existing deterministic allocator and its preference order remain intact.

Spills use natural scalar widths and matching alignments: 1,2,4,8 bytes.
Address preservation requires width 8. Narrow values keep their right-justified
logical semantics. Outgoing argument cells reserve eight bytes but stores retain
their logical width; unspecified upper bytes are not treated as value semantics.

Callee saves use dedicated eight-byte MOV storage, not PUSH/POP. The prologue
subtracts the frame, then saves only actually assigned callee GPRs in order:
RBX,RBP,RSI,RDI,R12,R13,R14,R15. The epilogue restores the reverse order, adds
the frame size and returns. RAX is never a save/restore scratch register.
Each Ret block has an explicit binding to this common epilogue action sequence;
coverage validation rejects an omitted or changed return binding.
Actions describe future lowering; no call-bearing prologue bytes are emitted.
Allocate/Release are stack arithmetic actions and do not promise preservation
of FLAGS. No pre-return FLAGS dependency is carried through the epilogue.

Cycle temporaries have dedicated eight-byte slots containing full canonical
GPR values. Slot count is the maximum symbolic requirement over calls. Two- and
three-way cycles need one slot; two disjoint two-way cycles need two with the
current scheduler. All 256 four-register source combinations pin a maximum of
two. The bounded descriptor capacity remains four; no scratch register is chosen.

## Concrete call sites and independent verification

Each `CallSiteActions` retains its abstract ABI plan and ordered concrete actions:

- Store preservation values.
- Execute the ABI move schedule: outgoing stores first, physical copies and
  concrete temporary stores/loads.
- One AbstractCall marker, with target/provenance retained in the ABI/function.
- Capture result RAX into the allocated result register (also explicit if RAX).
- Reload preserved values.

Result capture precedes reloads so a preserved pre-call RAX cannot erase the
call result before handoff. Replay also rejects result/preservation conflicts.
Every memory action names a concrete slot and its final RSP-relative offset.

`ValidateFrame` checks region coverage/order, tail padding, final alignment,
slot bounds/alignment, duplicate identities, pairwise overlap and category maps.
`ValidateFunctionFrame` independently checks exact spill and used-save coverage,
maximum outgoing/temp requirements, logical locals and every return binding.
`ValidateFrameEnds` checks the exact inverse save/restore and RSP adjustment
sequence. No value register in the epilogue is RAX.

`ReplayCallActions` independently executes register identities, available
widths and initialized stack slots, then checks arguments at the marker,
applies Win64 clobbers, inserts a symbolic result and checks every live value
and result after the sequence. It does not regenerate the move schedule or
compare against planner-produced action bytes.

The independent C11 test oracle uses actual 64-bit values and byte-addressed
stack storage. It executes saves, stores, loads, copies, clobbers and restores,
checks stack arguments against pre-call values and RSP balance, and checks
callee registers and RAX after the epilogue. Per-site preservation composes
across the shared frame; native function execution with calls remains deferred.

## Automata, DragonGod and semantic evidence

The production replay checker instantiates `CallReplayProtocol`, an input-driven
`automata` with one `machine` and PreCall/PostCall `state`s. Its transition
table admits store/load/copy before the marker and load/copy afterwards. A second
marker or post-call store returns InvalidActions through an unhandled input.
Register/stack value checks remain explicit and independent of this protocol.
No persistent source-language MachineFrame becomes native stack storage.

DragonGod.Machine.Execution and DragonGod.Replay.Core were assessed. They
operate on kernel Mind frames, trace logs, agent memory/events and actuation.
Those facilities add no necessary scheduling or ownership capability to this
synchronous checker. Standard does not acquire a dependency on DragonGod,
which already depends on Standard. The language automata seam supplies the
useful ordering check directly, with bounded inline state.

PlainData assertions cover FrameSlot, FrameRegion and FrameAction; these consume
the existing typed Verdict evidence. Mutable aggregate plans are not forced
into PlainData. Existing StaticExtent ABI table evidence remains applicable.
Runtime planning failures use Result, not semantic Verdict.
NoAllocation assertions cover layout, action lowering and independent replay.

## Bounded fixtures and determinism

`TestEVT2e4FrameRealization` generates checked CMIRAMD3 from
`testdata/evt2e4_frames.concept` and compiles the actual Concept backend with
strict C11, `-pedantic-errors`, in Normal and Verify modes.
Eight real call-bearing functions qualify:

| Fixture | Final bytes | Spills | Callee saves | Extra outgoing cells |
|---|---:|---:|---:|---:|
| Calls0 | 40 | 0 | 0 | 0 |
| Calls5 | 40 | 0 | 0 | 1 |
| Calls8 | 88 | 0 | 3 | 4 |
| Across | 40 | 1 | 0 | 0 |
| Multi | 136 | 5 | 4 | 4 |
| Early | 40 | 0 | 0 | 0 |
| Pressure | 56 | 4 | 0 | 0 |
| VoidCall | 40 | 0 | 0 | 0 |

Multi's zero/five/eight argument calls share the 32-byte extra outgoing area.
Across spills v4 at [rsp+36], width/alignment four. Pressure exercises four
simultaneous caller-saved values. Early binds two returns.
Additional allocated-plan probes exercise a single RBX save, all 256 source
combinations (including repeated sources and two/three-way cycles), two disjoint
cycles, and natural spill widths 1/2/4/8 including address preservation.
Those probes mutate derived descriptors; they do not claim new source/wire
admission for address arguments or arbitrary forged MachineIR.

Invalid probes cover bounded count/size overflow, invalid alignment/width,
overlapping slots/regions, missing spill/temp maps, missing epilogue/return,
malformed save/restore, absent marker, repeated marker and post-call store.
Every rejected normal EmitFunction call leaves its output buffer unchanged.

Eight full frame/action/debug descriptions repeat identically 100 times.
Explicit one- and two-temp cycle cases repeat 100 times too. Normal/Verify
canonical frame and action observations agree. Existing EVT2e3 abstract ABI
qualification remains in place; existing native-byte determinism is separate.

## Bootstrap change classification

| Go file | Classification | Change |
|---|---|---|
| cmd/concept/amd64.go | orchestration and diagnostic presentation | Invoke Concept qualification and render returned frame/actions/errors |
| cmd/concept/amd64_test.go | test oracle | Pin the later encoding stop and concrete CLI output |
| internal/concept/frame_realization_test.go | checked-input production, orchestration, test oracle | Generate fixture artifact; compile Concept; replay returned metadata |

No production Go spill selection, frame arithmetic, storage offsets or
save/restore insertion exists. The C host is bootstrap presentation; all
decisions it displays are supplied by the compiled Concept functions.

## Measurements and validation

Baseline layout was a 260-byte FinalizedFrame (byte count and 64 local offsets).
The EVT2e3 ABI CallPlan remains 660 bytes. Final C11 host measurements:

| Metadata | Bytes |
|---|---:|
| FinalizedFrame | 13,820 |
| FrameSlot | 20 |
| FrameAction | 28 |
| CallSiteActions including CallPlan | 1,564 |

Storage is bounded to 512 concrete slots, 512 spill identities, 15 register
indices, four temps/outgoing arguments, ten prologue/epilogue actions,
128 return bindings and 32 actions per call. The language's fixed-array cell
limit is 512; a combined frame exceeding that slot count rejects explicitly.
No hidden heap storage or spill fallback is introduced.

For 1,000 passes over Across in this Windows C11 host: frame planning took
44 ms Normal / 60 ms Verify; preservation requirement derivation through
PlanAllocatedCall took 4 / 4 ms; concrete action lowering took 1 / 1 ms.
Frame planning includes requirement derivation and geometry verification.
These overlapping phase probes are not an additive pipeline breakdown.
They ran alongside other regression lanes and are indicative fixture timings.
Typical qualified frames are 40 or 56 bytes; the largest real fixture is 136.

Baseline was measured in a detached managed worktree at the requested SHA:
full Go passed (internal/concept 426.038 s, CLI 17.124 s, GPU-free Vulkan
0.374 s), vet passed, focused race passed (22.718 s), and Standard 42,
DragonGod 23, Golden 130 passed in both Normal and Verify.

Final full Go passed: internal/concept 465.768 s, CLI 22.918 s, GPU-free
Vulkan 0.361 s. Vet passed; focused race covering EVT2e3/EVT2e4 passed
(47.300 s). Standard 42, DragonGod 23, Golden 130 passed in Normal and Verify.
The full suite includes semantic corpus admission, GPU-free Vulkan, EVT2
native AddMax/encoder/ABI checks, EVT2x native finite/dynamic-address/pushdown
traces, 100-run native-byte determinism and the frozen native byte oracle.
No-call native bytes remain unchanged.

The semantic corpus retains 436 valid, 357 static-invalid, 15 runtime-negative,
five compatibility and four expected-divergence specimens. Backend plan
fixtures are additional typed/allocated metadata specimens, not new language
admission categories. Strict C11 frame/action probes, 100-run plans, byte-stack
replay and Normal/Verify equality passed. The CLI assertions also pin concrete
frame regions, prologue/epilogue and pre/abstract/post-call diagnostics.

Touched Concept source/fixture format checks and lint passed; Go files were
gofmt formatted; whitespace/diff checks passed. No .concept_test source was
changed. The baseline direct test-file lint limitation involving generated
Standard.Build.Metadata is unchanged; package test commands supply that
metadata and pass. Both Zig suites were skipped under the frozen-path policy:
neither legacy compiler nor its build/test infrastructure changed.

The baseline worktree was archived after qualification. The implementation
is committed on codex/evt2e-cathedral-calls, with temporary test logs removed;
the primary checkout and existing PR publishing state were not changed.

## Remaining boundaries

No CALL encoding/fixups/native calls, floating/XMM storage, aggregate/sret,
varargs, indirect/tail calls, unwind/SEH, frame pointer debug mode, or stack
probing. Unsupported call classes/forms continue to reject in the ABI owner.
The CLI may still encounter an unsupported incoming stack-argument callee
before a caller when that callee appears first; the dedicated five-argument
CLI case puts the caller first. This existing bootstrap boundary is independent
of the qualified outgoing frame/action path.
