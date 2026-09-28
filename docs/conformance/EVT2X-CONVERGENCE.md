# EVT2x convergence ledger

Baseline: clean `35b539b88ebe081e3814c2e02a574821f569384c`, compiler
`concept-evt1-stage0-go`, in an isolated managed worktree. The caller's main
checkout was not changed.

The first blocker was that ordinary MIR functions retain their validated
semantic body for EVT2, whereas machine states retained only operation
summaries. Persistent machine-field initializers were likewise absent from
the in-memory lowering input. The change retains both in MIR without changing
checked JSON or C11 generation. A focused test proves the real
`machine_parent_resume.concept` push, resume, and initialized field survive.
LIR now refuses to silently omit automata. The observed next blocker is:

```text
concept lir language/evt1/machine-stack/valid/machine_parent_resume.concept
EVT2_UNSUPPORTED_AUTOMATA_LOWERING Worker
```

Executable state-body LIR, caller-owned native frame access, step dispatch,
the pointer ABI, and bridge/encoder support remain unimplemented. There is no
native C differential or NoAllocation proof for a machine step. Existing
state ordinals and capacity eight come from EVT1 MIR; there is no native
MachineFrame layout yet. No CMIR schema, allocator, or encoder change was made.

Validation of the revised tree: focused EVT2/machine tests, semantic corpus,
`go vet ./...`, root and legacy Zig suites passed. Standard Normal/Verify
each passed 35 tests; DragonGod each passed 23; Golden each passed 130. The
full `go test ./... -count=1` failed in the same pre-existing EVT2d baseline
categories: five Windows Clang/lld-link cases cannot open `m.lib`;
`TestProofHumanGoldens` has alignment-proof text drift; TinyXML2's upstream
`tinyxml2.cpp` is absent from this managed worktree; and 18 EVT1 checked
outputs are stale. No EVT2x or EVT2d native test failed. The first attempts
to run DragonGod under `dragon-god` and Goldens under `tests/goldens` used
invalid test roots; the correct roots are `libraries/DragonGod` and
`libraries/Golden`, and both modes passed there.

The work stops at **meaningful progression**: the MIR semantic-input blocker
is removed and the next executable LIR blocker is explicit and testable. EVT2e
has not started.
