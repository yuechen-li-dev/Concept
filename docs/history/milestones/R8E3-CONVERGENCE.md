# R8e3 convergence

The implementation uses a token-anchored source tape rather than attaching
comments to semantic AST nodes. This keeps formatting data out of semantic
identity and makes comment order explicit. The formatter refuses a rewrite if
its output changes the token sequence or stops parsing.

The verified boundary includes formatter idempotence, parseability, comments,
project ownership, and selected semantic outputs. Source-aware MIR and C
representations refresh their coordinate payload after layout changes. A
future semantic artifact comparison should separate those coordinates from
the semantic payload rather than freeze stale locations.

Baseline: `d35cafc96b334e79a64c980a7b9610f44b4e4dae`, with a clean
worktree. The baseline `go test ./... -count=1` passed. The active compiler is
`concept-evt1-stage0-go`.

Final validation includes focused R8e3 tests, `go test ./... -count=1`,
`go vet ./...`, root and legacy Zig suites, R7d3 BurnIn,
Standard Normal/Verify (35 facts in each mode), and DragonGod Normal/Verify
(23 facts in each mode). The full Go suite contains the R7p goldens, R7q
Frame, R8a through R8e2, and the semantic corpus. The 441-file valid source
formatting pass and 100-run large-library pass finished without parse or
idempotence failures.

The first full run after acronym suggestion normalization exposed the existing
`SomeLocal` → `someLocal` compatibility assertion. The suggestion logic now
retains interior capitals in mixed-case single words while normalizing
underscore-separated acronym segments. The focused R8e2/R8e3 tests pass.
