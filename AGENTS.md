# AGENTS.md

## Convergence rule

Every substantial task must end in exactly one of three states:

1. **Success**  
   The intended capability works in the real path and the real motivating case materially improves.

2. **Meaningful progression**  
   The capability is not complete, but one genuine blocker is removed and the next blocker is isolated with evidence.

3. **Honest stop**  
   Further work would require overbroad scope expansion, excessive debt, brittle patching, or tangled logic. Stop and report the reason with concrete evidence.

Do not continue producing patches once the work stops converging.

Do not confuse activity with progress.
A failed attempt is only acceptable if it leaves behind a narrower problem, stronger evidence, or a justified stop.

Any partial work must leave the codebase in a cleaner, more legible, and more diagnosable state than before.

## Test ownership and iteration

- Follow [the EVT2 qualification policy](docs/tooling/EVT2-QUALIFICATION-POLICY.md),
  effective 2026-10-31. Ordinary milestones require focused native/C11,
  Normal/Verify, malformed-input and adjacent regression evidence, with a small,
  justified fresh determinism sample. Full project qualification is not an
  ordinary milestone prerequisite. The next planned major checkpoint is EVT2h
  phase closeout, tentatively EVT2h9.
- Expand testing immediately for changes to shared ABI, allocation, frame
  planning, encoding, transport compatibility or other high-risk infrastructure;
  do not wait for the planned checkpoint. Preserve existing cases, scripts,
  independent oracles, golden expectations and full 100-run gates; change their
  scheduling, not verification logic.
- Report evidence status as IMPLEMENTED, MILESTONE VERIFIED or FULLY QUALIFIED
  using that policy. Do not carry a baseline's full qualification onto new work.
- Run `./test.ps1` for the default quick loop. It runs Concept runtime and leaf
  backend component facts in Normal and Verify, plus workflow/CLI frontend checks.
- Runtime behavior assertions belong in `.concept_test` files under `tests/runtime`
  or the owning library. Backend algorithm facts belong in `tests/backend` and
  import the smallest leaf module. Group related facts so the Concept runner
  compiles their module once. Do not embed a second copy of the program in Go.
- Go tests own compiler diagnostics, AST/MIR/LIR inspection, artifact mutation,
  determinism, tool orchestration and independent C/native/ABI oracles. A Go
  inspection test should read the same authoritative Concept fixture.
- The quick lane must not import the full `Standard.Backend.AMD64` graph or launch
  a cold native backend. Add native CLI cases to `TestAMD64CLIBatch`, which uses
  the real multi-input CLI with one backend build. Keep the existing single-input
  cold smoke; do not add per-case `go run . amd64 ...` launches.
- Use `./test.ps1 -Lane Runtime -Filter <fact>` for a focused Concept fact, or
  `-Path tests/backend -Filter <fact>` for a backend component. Add `-Verify` for
  Verify mode. Use a focused `go test -run` for an owning compiler boundary.
- `./test.ps1 -Lane Compiler` is the broader short Go compiler/corpus sweep;
  `./test.ps1 -Lane Qualification` retains the complete Go/native/differential
  and 100-run gates plus vet. Neither is the default edit loop. Do not replace
  independent oracles or cache test results to make these gates faster.

## Branch integration

- Treat current `main` as the integration baseline. Fetch and inspect its
  relationship to HEAD before starting work; milestone hashes are historical
  evidence, not instructions to discard newer changes.
- Use a task branch when isolation is needed. After an authorized merge, verify
  that its commits are reachable from main and remove obsolete branch refs.
- Check patch equivalence and checkpoint history before replaying old divergent
  work. Preserve newer implementations when older changes are already included.
- Preserve worktree files during branch cleanup. Retain historical checkouts as
  detached snapshots when appropriate; never discard uncommitted work to tidy
  the branch list.

