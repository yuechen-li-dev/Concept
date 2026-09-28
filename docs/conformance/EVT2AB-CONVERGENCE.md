# EVT2a+b convergence ledger

Baseline: `2a7a922fce330dff49c61508536f9a4e1d3694eb`; compiler `concept-evt1-stage0-go`. The work is isolated in the `evt2-lir` worktree. The caller's `main` checkout had one later commit and was left untouched.

The existing General Planner was integrated as the LIR planning authority. Its validated decision ID and fact references travel to bounds guards. The C11 emitter remains on its established path. `concept lir` is a structural inspection command, separate from semantic `explain`.

Focused EVT2 tests, the semantic corpus manifest, `go vet ./...`, both Zig suites, Standard Normal/Verify (35 passed each), DragonGod Normal/Verify (23 passed each), and Golden Normal/Verify (130 passed each) passed. The baseline and post-change full `go test ./... -count=1` runs failed on this Windows host with `lld-link: could not open 'm.lib'`. Both also reported stale checked outputs, proof golden drift, and a missing TinyXML2 upstream file when run from the relocated managed worktree. These are unqualified baseline/host failures; no EVT2 test failed in the post-change run.

Current limits: integer core, bool, local mutation, if/early return, simple ascending range loops, and rank-one fixed arrays. Unsupported semantics fail explicitly. No EVT2c work was started.
