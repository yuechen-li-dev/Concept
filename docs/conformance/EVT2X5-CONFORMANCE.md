# EVT2x5 conformance: arbitrary activation stride

Status: **meaningful progression**. EVT2x is not complete.

Baseline: `d0fcbffec906dfe7b44578d3f3dad02ef555d6e4` on
`codex/evt2x-native-automata`, compiler `concept-evt1-stage0-go`.

| Claim | Evidence |
| --- | --- |
| Target-independent indexed address | `index_address` now accepts a guarded pointer base, an arbitrary positive constant stride, and a fixed byte displacement. |
| LIR safety | Verifier requires matching same-block `check_index`, valid pointer/index types, and bounded address displacement. Malformed cases have focused tests. |
| AMD64 legalization | Non-SIB stride uses ordinary MOV, IMUL, then scale-one LEA; no activation opcode or CMIRAMD1 schema change. |
| Native stride 12 | Concept backend executable-memory test writes slots 0, 1, 2 and verifies every other surrounding word remains a sentinel. |
| Native dynamic top tag | A direct LIR/backend probe loads depth from the caller-owned frame, computes `depth - 1`, checks capacity eight, loads tags at depths 1, 2, 3, and asserts MachineIR IMUL plus scale-one LEA. |
| Byte determinism | Native harness re-emits each tested function 100 times and compares emitted bytes. |
| Source pushdown boundary | `concept lir language/evt1/machine-stack/valid/machine_parent_resume.concept` still reports `EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP Worker.Parent.Start`. |

The direct LIR probes establish general native address capability, not a
generated source Step. Source-generated depth loading, machine tag/state dispatch, child frame
initialization, pop destruction, and C/native pushdown traces remain
unqualified. No NoAllocation proof for generated Step is claimed. The layout
remains capacity eight, Parent 8 bytes, Child 4 bytes, slot 12 bytes, total
caller-owned frame 108 bytes. The backend allocator and encoder are unchanged.

Focused EVT2 and machine-stack tests, semantic corpus, `go vet ./...`, and both
root and legacy Zig suites pass. Standard Normal/Verify each pass 35,
DragonGod Normal/Verify each pass 23, and Golden Normal/Verify each pass 130.
