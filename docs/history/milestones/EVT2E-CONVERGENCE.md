# EVT2e call handoffs

## EVT2e1: meaningful progression (historical)

Baseline: `a85747a2ff061848e06ecbc10e68d88710cf278c` (`concept-evt1-stage0-go`). The existing bridge remains `CMIRAMD2`, version 2, schema hash `99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593`. Its wire contract has not been changed.

The first missing link, a first-class LIR call, is present for checked direct functions with bodies, `bool` and fixed-width integer arguments, scalar or void results, and `win64` convention identity. The call preserves the validator-selected declaration signature, ordered argument values and types, and source span. The LIR verifier checks the callee and argument contract before the machine backend sees it. Stage-0 owns only this checked semantic transport; no new Go ABI placement, allocation, frame, or encoder algorithm was added.

`internal/concept/testdata/evt2e_calls.concept` exercises zero-argument, bool, void, scalar-result, and five-argument calls, including a local value live across a call. At the EVT2e1 handoff, `concept lir` printed these calls and `concept machineir` failed with `MIR_UNSUPPORTED_LIR_OP call`. The next boundary was a versioned MachineIR call contract. The fixture had no native call execution claim.

The frozen Zig compiler path is unchanged.

## EVT2e2: success

Baseline `b7e8d69900718455947eb3e8515c6488370d9621`, compiler
`concept-evt1-stage0-go`. Direct scalar calls now reach verified MachineIR and
CMIRAMD3 v3. Its sole Concept schema has hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.
Go and Concept codecs are generated; Concept validates call semantics before
normal backend projection or publishing artifact-only roundtrip output.
23 malformed artifacts reject in both paths; valid bytes and printers repeat
identically for 100 runs. The new record PlainData assertions consume typed
Verdict geometry evidence; runtime artifact admission uses Result.

`concept lir` and `concept machineir` succeed. `concept amd64` reaches the later
named boundary `AMD64_UNSUPPORTED_CALL_LOWERING: call ABI lowering is not implemented`.
No Win64 placement, shadow space, call frame, save/spill, CALL bytes or native
call execution was added. See [EVT2e2 conformance](../../conformance/EVT2E2-CONFORMANCE.md)
for complete transport, regression and qualification evidence.

## EVT2e1 qualification (historical)

`go test ./... -count=1` passed after the LIR code change (`internal/concept` 381.348s); `go vet ./...`, focused `-race` call tests, the semantic corpus manifest test, and the existing EVT2/EVT2x focused native tests passed. Standard passed 42/42, DragonGod 23/23, and Golden 130/130 in both Normal and Verify modes. The fixture passed `concept check`, `concept format --check`, and `concept lint`. The new call LIR repeated identically in 100 runs. No new native call bytes, Win64 ABI probe, call frame size, spill count, or call performance measurement exists yet.

## EVT2e3: success in ABI and clobber planning

Baseline `ea8f57cba65847e8466233f74d538c11d207b4dc`; compiler
`concept-evt1-stage0-go`. CMIRAMD3 v3 and its schema hash are unchanged.
Concept `Win64ABI` owns checked register tables, scalar argument/return
placement, home space and symbolic stack arguments. Concept liveness includes
all call argument uses and independently walks backward from CFG live-out to
classify values live across each call. Fixed argument uses/result defs,
preservation records, used-callee-saved registers and cycle temporaries are
inspectable derived state.

Parallel moves capture stack arguments first, resolve safe register moves in
stable order, and break cycles with symbolic full-GPR temporaries. Qualification
includes two/three-way cycles, all 256 four-register source combinations,
repeated sources, narrow/wide classes, and six arguments. Nine real source call
fixtures repeat 100 times, and Normal/Verify canonical output agrees exactly.
The motivating local value is identified as `MustPreserve` across the call;
callee-saved placement records a later save requirement.

`concept amd64` prints a verified call plan and then stops at
`AMD64_UNSUPPORTED_CALL_FRAME_LOWERING`. No preservation storage, save/restore
emission, final outgoing frame offsets, CALL bytes, fixups or native calls are
implemented. Existing no-call bytes and EVT2x behavior remain qualified. See
[EVT2e3 conformance](../../conformance/EVT2E3-CONFORMANCE.md) for evidence,
timings, metadata bounds and the existing test-file lint limitation.

## EVT2e4: success in frame and preservation realization

Baseline `24ea223e2f372be9418c1c02de92e5df44c5128e`; compiler
`concept-evt1-stage0-go`; CMIRAMD3 v3/hash unchanged. The Concept finalizer now
owns one coherent native stack frame: outgoing shadow/extra arguments, locals,
used-callee saves, per-vreg preservation spills, cycle temps and tail padding.
RSP remains stable after one prologue subtraction. Every return binds to the
same inverse restore/release/return actions; RAX survives that epilogue.

Concrete call sites store preservation values before argument moves, realize
stack/temp offsets, retain an abstract call target, capture RAX results and
reload live values. Independent Concept identity/width replay and C11
byte-stack/value replay qualify actions; the executable CallReplayProtocol
automata checks legal pre/post ordering. DragonGod kernel scheduling/state
was assessed; no extra kernel dependency or conflation with MachineFrame was
needed. Typed Verdict remains semantic evidence through PlainData assertions;
runtime frame failures retain precise Result errors.

Eight real frame-bearing fixtures, 256 source combinations and explicit cycle
plans repeat deterministically in Normal/Verify. Qualified fixture frames span
40..136 bytes. The normal CLI prints concrete frames and actions, then stops at
`AMD64_UNSUPPORTED_CALL_ENCODING`. CALL bytes, fixups, native call execution
and unwind/SEH remain deferred. See
[EVT2e4 conformance](../../conformance/EVT2E4-CONFORMANCE.md) for full evidence,
metadata, measurements and boundary details.
