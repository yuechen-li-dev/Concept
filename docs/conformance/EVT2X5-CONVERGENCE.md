# EVT2x5 convergence ledger

End state: **meaningful progression**. The AMD64 scale restriction no longer
blocks native dynamic activation addressing. LIR retains the semantic 12-byte
stride; MachineIR legalizes it to ordinary multiply and scale-one address
arithmetic. Native tests reach the first three activation slots and load their
tags through `top = depth - 1`.

The next concrete blocker is the source-to-LIR Step lowering boundary:
`lowerAutomataToLIR` returns `EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP` before it can
build dynamic machine-tag/state dispatch or lower child push/pop bodies. Its
current `lirBuilder` frame field access assumes one fixed machine frame, while
a pushdown Step needs a tag-selected typed view of the current slot. No stub
Step has been emitted to hide that boundary.

The change is local to general LIR verification, MachineIR legalization,
native address tests, and the design record. CMIRAMD1 and the Concept-written
allocator/encoder are unchanged. No heap fallback, scheduler, async machinery,
or dynamic registry was added. EVT2e has not started.

Focused EVT2/machine-stack tests, semantic corpus, `go vet ./...`, and both
Zig suites passed. Standard Normal/Verify passed 35 each, DragonGod 23 each,
and Golden 130 each. The full `go test ./... -count=1` still fails only in
previously recorded baseline categories: five Windows Clang links missing
`m.lib`, proof-human golden drift, absent TinyXML2 upstream source, and 18
stale EVT1 checked outputs. No EVT2x5 or EVT2d native test failed.
