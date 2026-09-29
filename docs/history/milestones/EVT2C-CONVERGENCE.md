# EVT2c convergence ledger

Baseline requested: `23a5c0f836aa5ed658601a1bac48b4a6923ddb9d` (`concept-evt1-stage0-go`). The already merged `origin/main` checkout used for implementation is `235e80e21c67a54a87034ad0ef79ee6037c9b083`, which contains that commit without changing the EVT2a+b LIR implementation. Work is isolated in the clean `evt2-machineir` worktree. The caller's `main` checkout was left untouched.

The baseline `go test ./... -count=1` failed in seven pre-existing categories: five Windows Clang/lld-link tests could not open `m.lib`, `TestProofHumanGoldens` saw alignment-proof text drift, `TestR7j1TinyXML2ArtifactABIChainAndIdentity` could not find the upstream `tinyxml2.cpp` in the relocated worktree, and `TestEVT1CheckedOutputsMatch` reported 18 stale checked outputs. These failures were recorded before EVT2c edits and were not converted into backend semantics changes. Baseline `go vet ./...` passed.

EVT2c adds an AMD64 Windows MachineIR function/block/operand model, Win64 argument and return placement, parent physical registers and width aliases, virtual GPRs, numbered flags dependencies, explicit checked-overflow and bounds trap edges, indexed LEA/load/store, abstract slots and frame metadata, instruction effect metadata, verifier, deterministic printer, and `concept machineir <file>`. `concept lir` and the C emitter remain on their previous paths. The real fixture emits Add, Max, Sum4, CheckedIndex, StoreIndex, Choose, and Early MachineIR and verifies all seven.

Qualification:

| Gate | Result |
|---|---|
| Focused EVT2 tests, including 100-run MachineIR determinism and malformed verifier tests | Pass |
| Semantic corpus manifest | Pass |
| `go vet ./...` | Pass |
| Root and legacy `zig build test` | Pass / Pass |
| Standard Normal / Verify | 35 / 35 pass, 3 benchmarks each |
| DragonGod Normal / Verify | 23 / 23 pass, 1 benchmark each |
| Golden Normal / Verify | 130 / 130 pass, 2 benchmarks each |
| Full `go test ./... -count=1` | Pass after the qualification follow-up: `internal/concept` 262.767 s; Vulkan profile package pass. |

The current MachineIR has no encoder, executable memory, native execution, object files, full register allocator, calls, SysV ABI, or float/SIMD. `UMUL` remains a target legalization pseudo-op. EVT2d was not started.

## Qualification follow-up

The first post-change full run also failed `TestR7d5BlackboardPublicationDisjointWritersAndMPSCNativeThreads` with native exit 3. Its reader used a million tight polling iterations, which could finish before the writer was scheduled. The test now yields after an unsuccessful poll and uses the existing monotonic clock to bound the wait to five seconds. The semantic assertion remains `observed == 42`; the focused test passed ten consecutive runs and the final full run passed.

Three baseline failures were checkout or host setup: `.gitattributes` now pins LF for the EVT1 source fixtures, byte-exact checked outputs, and proof text (the local worktree was normalized to match); Windows native harnesses no longer pass POSIX `-lm` to the MSVC-target Clang driver; and the declared TinyXML2 submodule was initialized at its pinned `8224e427b655b83dae5e2298f1e6919523a78737` commit. The proof and checked-output tests, the five formerly `m.lib`-blocked tests, and the TinyXML2 ABI test passed individually before the full run. No checked output was regenerated or changed, and the compiler, C emitter, LIR, and MachineIR semantics were unchanged by this follow-up.

After the full Go pass, `go vet ./...`, both Zig suites, Standard Normal/Verify (35 each), DragonGod Normal/Verify (23 each), and Golden Normal/Verify (130 each) passed again. The worktree has only the follow-up source, attribute, and ledger changes; the initialized upstream submodule remains at its declared commit.
