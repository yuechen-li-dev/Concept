# EVT2x4 conformance: bounded activation layout and native root Init

Status: **meaningful progression**, not EVT2x completion.

Baseline: clean `30db8f09e43031b9eb085299a8db3e3997ff875c` on
`codex/evt2x-native-automata`, compiler `concept-evt1-stage0-go`.

| Claim | Evidence |
| --- | --- |
| Closed machine tags | Validated MIR `RuntimeOrdinal`, `Reachable`, root and push-target checks in `planActivationStack` |
| Exact typed scalar frame geometry | `evt1StructFieldOffsets` and `evt1TypeGeometry`; 100-run layout identity for parent/child, three-machine nested, and recursive fixtures |
| Bounded slot geometry | Capacity 8, max frame size/alignment, slot/tag/storage offsets, total frame size pinned in `TestEVT2x4ClosedActivationLayout` |
| Root publication order | Init stores state/fields, tag, then depth=1; LIR verifier rejects an incorrect depth value, undersized slot, or wrong field offset |
| Native root Init | Concept backend emits and executes AMD64 for the 108-byte parent/child layout; host checks depth, tag, shared state, root state, and initializer |
| Finite native regression | Existing finite, yield, repeated-yield, and EVT2d executable-memory tests pass |
| Ownership boundary | Owned native activation fields explicitly reject with `EVT2_UNSUPPORTED_ACTIVATION_FIELD_TYPE`; typed child Destroy is not claimed |
| Pushdown boundary | `concept lir language/evt1/machine-stack/valid/machine_parent_resume.concept` still reports `EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP Worker.Parent.Start` |

The new root Init has no heap operation. No formal generated-function
NoAllocation proof is claimed. The capacity is a storage bound, not a per-step
work bound. CMIRAMD1 is unchanged; MachineIR and the Concept encoder see
ordinary addresses, stores, and returns.
