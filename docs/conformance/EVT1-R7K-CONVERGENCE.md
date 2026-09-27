# EVT1 R7k convergence

Baseline checkout was clean at `37a11a2452f06b3311b7a6c55f209fedf7fa1f28`
after R7j1; compiler identity is `concept-evt1-stage0-go`.

| Blocker | Reproducer and root cause | General fix | Regression and hardware advance |
| --- | --- | --- | --- |
| Missing 16/32-bit explicit spellings | Core admitted `uint8`, `uint`, and `uint64`; no `uint16`/`uint32` admission or geometry | Added scalar definitions, ranges, and geometry | Four exact MMIO widths compile and run |
| Observable access absent from MIR/C | Ordinary template calls had no device transaction contract | Added generic `mmio_read`/`mmio_write`, required planner retention, and width-specific C11 volatile lowering | Dead/repeated access and order trace checks |
| Register bits had no deterministic value layout | Struct fields would introduce C layout rather than declared ranges | Added scalar-backed `bits`, inclusive ranges, overlap diagnostics, value projection, and checked insertion | Decode, construction, bad ranges, and reserved-bit preservation |
| Artifact consumer lacked hardware effects | Existing allocation summary admits one effect per function | Added separate closed hardware read/write summaries to `concept-module.v1` | Artifact-only UART consumer proves effects without source reparse |
| Hosted UART needed an address-space path | Device access was absent from Standard and DragonGod | Added `DeviceMemory` and ordinary UART functions over MMIO primitives | DragonGod package proof and strict-C11 hosted UART run |
| Standalone proof fixture violated the frozen semantic corpus manifest | Full `go test ./...` reported `fixture is outside semantic manifest` for `language/evt1/hardware-memory/valid/mmio_facts.concept` | Moved the fixture under the established `language/evt1/tooling/` ownership area | `TestSemanticCorpusManifest` passes; direct `concept explain` still proves hardware facts |

The hosted simulator is a compile-time C adapter for the production MMIO
macro seam; production defaults remain volatile loads and stores. No runtime
registry, second pointer model, UART-specific compiler branch, optimizer
elision, inline assembly, or architecture-specific fence was introduced.

Validation on this Windows host used the exact Go commands in `Make.oct`
because `oct` was unavailable: `go run ./cmd/concept package build Standard`,
`go run ./cmd/concept package test Standard`, `go run ./cmd/concept package build
DragonGod`, `go run ./cmd/concept package test DragonGod`, and
`go test ./internal/concept -run R7d3 -count=1` (`BurnIn`). Standard passed 29
package tests, DragonGod passed 22 tests and one benchmark was enumerated.
`go vet ./...`, both root and `legacy/poc3-zig` `zig build test`, and Clang/GCC
strict C11 hosted checks at `-O2` also passed. The focused hosted test, including C
compilation and 100 trace executions per compiler, took 2.68 seconds in one
informational local run; this is not a device throughput benchmark.
The final `go test ./...` pass completed in 238.798 seconds for
`internal/concept` after the proof fixture moved into tooling ownership.
The EVT1 source tree contains 746 `.concept` files; the frozen semantic
manifest and the separately owned tooling fixtures retain their existing
test boundaries.
