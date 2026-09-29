# EVT2x3 native automata progression

End state: **meaningful progression**. The generated finite-state machine Init
and Step now execute as AMD64 through the Concept-written allocator and encoder.
Native pushdown execution remains unsupported, so EVT2x3 is not complete.

Baseline: clean `4a867e77351990358dca968e6d1ce288b2decf85`, compiler
`concept-evt1-stage0-go`, `codex/evt2x-native-automata` managed worktree.
The main checkout was not edited.

## Native path established

The EVT2x2 frame uses `current_state:u32` at offset 0 and `completed:bool`
at offset 4, followed by scalar shared and machine fields under ordinary
Concept layout. The finite fixture has a 16-byte, 4-byte aligned frame.
Init receives its caller-owned frame pointer in Win64 RCX and returns void.
Step receives the same pointer in RCX and returns the 4-byte `StepResult` tag
in EAX: Active=0, Yielded=1, Completed=2. The generated invalid-state path
retains a native TRAP. No aggregate frame return or heap allocation is used.

MachineIR selection now accepts `ptr<frame:...>` as an 8-byte ABI argument,
folds verified constant frame-field offsets into normal memory operands,
and selects ordinary MOV, LOAD, STORE, CMP, JCC, RET, and TRAP operations.
The frame address folding avoids consuming a new virtual register for every
field access. The finite Step needs 21 virtual registers, within the existing
32-register Concept allocator bound. The CMIRAMD1 schema is unchanged; the
machine module verifies and round trips through the bridge. The Concept
encoder now supports byte-width frame LOAD/STORE for `completed`.

`TestEVT2x3FiniteNativeStep` compiles the Concept backend to C11, emits native
bytes for machine Init and Step, maps them executable, and calls them on two
interleaved caller-owned frames. It checks finite transitions, completion,
already-completed idempotence, yield re-entry at the start of the same state,
and repeated yield. Each function is re-encoded 100 times with byte identity.

## Remaining pushdown boundary

`GenerateLIR` still reports `EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP
Worker.Parent.Start` for the checked `machine_parent_resume.concept` fixture.
The source operation `push Child goto Resumed` saves the parent's continuation
state, initializes an inline *Child machine frame*, marks the active machine
tag, increments depth, and ends the Step. Child completion drops that child
frame and decrements depth. The existing EVT1 runtime has capacity 8, with
machine tags and specialized per-machine frame arrays. A stack containing
only state IDs plus one set of persistent fields cannot preserve this
behavior, especially across recursion or different child machine types.

The next bounded implementation needs one caller-owned automata instance
layout with depth, machine tags, per-slot state IDs, and specialized scalar
field storage; dispatch must select the active machine and slot. It also
needs verified overflow-before-write and underflow-before-read control flow,
plus native nested-push, goto, yield, pop, and two-instance evidence. The
current single-machine `LIRMachineFunction` verifier assumes one scalar
`current_state` and does not represent or verify indexed frame storage.
That is the precise next blocking interface. The code retains the explicit
diagnostic instead of pretending that finite-state execution proves pushdown.

No scheduler, async machinery, JIT, or general coroutine runtime was added.

## Qualification

Focused EVT2, machine-stack, semantic-corpus, bridge, and native encoder tests
pass. Standard Normal/Verify pass 35/35 each; DragonGod Normal/Verify pass
23/23 each; Golden Normal/Verify pass 130/130 each. `go vet ./...`, root
`zig build test`, and legacy `zig build test --build-file
legacy/poc3-zig/build.zig` pass. The existing EVT2d executable-memory test
also passes. The new native finite/yield test passes on Windows AMD64.
The backend source passes `concept lint`. Its `concept format --check` result
is noncanonical on both the unchanged baseline and this revision; this task
does not reformat the whole backend file.

The exact full `go test ./... -count=1` run fails in the previously recorded
host/checkout categories: five Windows Clang links cannot find `m.lib`, the
proof human golden differs, TinyXML2 upstream source is absent from this
worktree, and 18 EVT1 checked outputs are stale. The run reports no failure
in EVT2x3 native execution, EVT2d, or the focused machine tests.
