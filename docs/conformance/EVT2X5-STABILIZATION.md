# EVT2x5 host and checkout stabilization

Baseline: `2fb04444d26e3c8d996aa4c74547318bb96b2264`, compiler
`concept-evt1-stage0-go`. This round addresses the four pre-existing full-Go
gate categories without changing compiler semantics or golden expectations.

| Failure | Cause | Repair and evidence |
| --- | --- | --- |
| Windows `m.lib` | The native test harness passed POSIX `-lm` to MSVC-target Clang, which requested an unavailable `m.lib`. | Windows host links omit `-lm`; Unix retains `-lm -pthread`. Focused literal, MMIO, UART, aggregate, and direct `expf` Clang/GCC execution pass. |
| Proof summary golden | Git converted tracked LF proof text to CRLF under `core.autocrlf=true`, while the renderer emits LF. | `.gitattributes` pins proof goldens to LF. The unchanged byte-exact `TestProofHumanGoldens` passes. |
| EVT1 checked outputs | Git converted checked artifacts to CRLF and source fixtures to CRLF; generated manifest source hashes then differed. | `.gitattributes` pins both fixture sources and checked outputs to LF. The unchanged byte-exact `TestEVT1CheckedOutputsMatch` passes; no snapshot contents or expected counts changed. |
| TinyXML2 source absent | The pinned Git submodule was uninitialized in this worktree. | Initialized commit `8224e427b655b83dae5e2298f1e6919523a78737`; native project Normal/Verify each pass two tests. The remaining ABI-chain test follows existing test policy and skips with an actionable message if a future checkout omits the submodule or Clang C++ toolchain. |

A later full-suite run exposed a separate native-thread test race: its reader
could exhaust a fixed iteration count before the writers were scheduled. The
reader now yields until it observes publication or a monotonic 10-second
deadline. It still requires the published value 42 and final disjoint write
100; ten repeated native runs pass. The deadline only bounds test waiting,
not generated Concept semantics.

The root README now calls out `git submodule update --init --
tests/dogfood/tinyxml2/upstream` for full coverage. A checkout without that
submodule does not claim TinyXML2 coverage.

Cross-platform preparation: the host-link argument selector is tested for
Windows, Linux, and macOS. On this Windows host, `GOOS=linux` and
`GOOS=darwin` test binaries and the CLI compile for both amd64 and arm64 with
`CGO_ENABLED=0`. These checks establish Go source/build portability, not
native runtime or C toolchain behavior on Linux or macOS hardware.

Final local gates: `go test ./... -count=1` and `go vet ./...` pass with the
submodule initialized. Root and legacy `zig build test` pass. The TinyXML2
native project passes Normal and Verify (2 tests each). The Windows math
link probe executes with both Clang 22.1.3 and GCC 15.2.0; the host Go toolchain
is 1.27.0. Linux and macOS native C execution remains unverified on this host.
Standard Normal/Verify pass 35 tests each, DragonGod pass 23 each, and Golden
pass 130 each.

On a Linux or macOS host, the remaining native qualification is to initialize
the pinned submodule, run `go test ./... -count=1` and `go vet ./...`, then run
`go run ./cmd/concept test tests/dogfood/tinyxml2` with and without `--verify`.
Those commands exercise the actual host C/C++ toolchain and native ABI probe;
cross-compilation from Windows does not substitute for them.
