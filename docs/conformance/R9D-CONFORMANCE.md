# R9d conformance

Compiler ID: `concept-evt1-stage0-go`.
Baseline: `22cdbf5934bf7096442bdbf23da786ce0ee934f1` (completed EVT2e5).
Branch: `codex/r9d-prefer-match`.

The initial workspace was main at `ad23a3909d02d1245fb17b47b95947f0ac80db87`.
The completed EVT2e commits were found on the existing local branch, and the
R9d work was carried onto its committed tip without changing that branch or
its separate worktree. The R9d commit identifies the final source revision.

## Qualified capability

Ordinary statement `else if` parses as an existing nested IfStmt in a synthetic
else block. The synthetic block shares its child's source site; authored braces
retain their own sites. The parser's old CV4185 expression-level ladder ban is
removed, and the two tests expecting that ban are retired. Existing lowering,
typing, C emission and Cathedral native codegen consume the same control-flow
representation; no backend implementation is changed.

The Concept-authored `PreferMatchOverElseIfLadder<declaration F>` requirement
uses the general declaration proof projector's
`compiler.NoMatchShapedElseIfLadder(F)` observation. The existing manifest
LintPolicy activates it at warning/error severity. Disabled means no activation
value. No separate config, statement AST exposure, grammar ban, or parallel lint
engine is introduced. Findings are projected from the ordinary proof graph at
each ladder root and carry project-policy provenance. Other composed policy
failures stay visible.

The supported threshold is three discriminating branches, excluding a final
else. Lexically bound storage identity and checked field paths establish a
common subject. Distinct tag-only enum variants and integer literal values are
recognized, including reversed equality, parentheses and negative literals.
Known enums/enum discriminator fields receive a readability/exhaustiveness
suggestion; integer discriminators receive only a case-structure suggestion.
Calls and reference/computed reads are excluded, duplicate values are excluded,
and unknown subjects supply no finding. Guard chains and simple two-way flow
stay quiet; categorical early returns can warn. No match semantics change occurs.

The observer is enabled only when a declared requirement uses it. It runs during
existing validation with the real lexical scopes, visits each ladder once,
and skips synthetic continuation roots. It adds no separate whole-source
traversal. Declaration proofs project the collected observations. No extensive
benchmark is warranted for this bounded observation. A five-run CLI sanity
measurement over unchanged copies of AMD64 and Win64ABI measured median total
lint time at 716.7 ms with no policy and 737.4 ms with the preference active
(about 2.9 percent). Both returned no findings. This includes process startup,
imports, manifest parsing and policy proof projection, ran alongside regressions,
and is not an isolated observer benchmark or a throughput claim.

Allman formatting keeps `}` / `else if (...)` / `{` on separate lines, with
`else if` together. Existing same-line formatting remains supported.
Formatter naming behavior is unchanged. Autofix is deferred: suggestion-only,
with no source rewriting or trivia risk.

## Specimens and executable checks

`tests/lint/r9d/manifest.concept` enables warning policy and existing Allman
format settings. Its enum Decode and integer Opcode produce exactly two
findings, while Validate and Exhaustive produce none. CLI lint, lint Verify,
ordinary check, format check and policy explain exercise this real project.
The enum sample is anchored at program.concept:9:5 and cites
manifest.concept:14. The proof is Disproven because the absence requirement
has a checked counterexample.

Four valid semantic-corpus specimens cover ordinary else-if, a long same-subject
ladder, guards and an exhaustive match equivalent. The manifest now records
440 valid, 357 static-invalid and 15 runtime-negative specimens; compatibility
and expected-divergence totals remain five and four. Lint findings are not
classified as invalid source.

Focused tests cover positive enum/integer/field/reversed/parenthesized/negative
cases; comptime functions; guard/duplicate/two-comparison/effectful/existing-match
negatives; nested and shadowed bindings; explicit nested else blocks; multiple
ladders per function; policy composition; absent/disabled/warning/error activation;
artifact independence; stable root sites; unavailable observation Unknown;
manifest provenance and explain. A regression first reproduced a template's
ladder being attributed to its caller; a lexical validation-scope marker now
prevents that attribution, including callable bodies with the same enclosing
function name. Both exclusions are pinned. Open templates/generated callables
are conservatively outside this initial authored-function observation.
Lint JSON and both formatter styles are pinned
across 100 runs, with separate comment preservation/idempotence checks.

Strict C11 execution compares ladder and match results, including final-else,
nested and expression-level else-if paths. A separate Windows AMD64 test sends
a ladder plus its direct-call wrapper through MachineIR, the CMIRAMD3 bridge,
Cathedral module emission and executable memory. Eight inputs agree with the
C11 oracle; Normal/Verify compare the complete emitted native bytes.

## Cathedral audit

The representative EVT2e owner already uses match for categorical dispatch:
Win64ABI's ABIClass validation, MoveLocation source/destination replay,
classification and physical-register decisions; AMD64's OperandKind validation,
register encoding, condition encoding and terminator dispatch. The remaining
validation/capacity chains are sequential guards. No repository-wide rewrite
or backend source churn is warranted. CMIRAMD3, call lowering, frame realization,
allocation, encoding and fixups are unchanged.

The actual lint command also ran over unchanged copies of both owner files
in a repository-local ignored audit project using the R9d warning manifest and
the ordinary `libraries` module roots. It returned zero findings. The checked-in
dogfood project supplies positive specimens independently of that audit.

Qualification remains EVT2e's bounded Win64 integer/bool direct-call subset.
Its documented pointer-source, external linkage, XMM, aggregate, unwind and
stack-probing limitations remain. No EVT2f implementation is begun.

## Final validation record

The full Go suite passed before the final scope-ownership regression repair
(internal/concept 516.570 s, CLI 23.488 s, Vulkan profile 0.353 s). Final-head
validation was rerun after the repair:

- `go test ./... -count=1`: passed; internal/concept 510.750 s,
  cmd/concept 23.260 s, GPU-free Vulkan profile 0.358 s. This includes the
  semantic corpus, formatter/policy suites, EVT2/EVT2x native execution, EVT2e2
  transport/exact bytes, EVT2e3 ABI/moves/liveness, EVT2e4 frame realization,
  EVT2e5 native internal-call parity and Normal/Verify image/fixup byte agreement,
  and the existing R9a2 frozen native byte oracle.
- `go vet ./...`: passed on the final implementation.
- Focused `go test -race ./internal/concept -run
  '^(TestR9d|TestManifestNamingLint|TestPolicySeverityDoesNotChangeArtifact|TestEVT2cDeterminism100)$'
  -count=1`: passed in 2.950 s, including the new ownership exclusions and native
  ladder test.
- Final focused R9d tests passed, including strict C11 and actual Win64 direct
  calls with the ladder function. The final full suite also includes those tests.
- Real CLI dogfood lint and lint Verify: exactly two warning findings; ordinary
  check and format check: passed. An ignored severity probe returned lint status
  1 with error activation, ordinary-check status 0, and lint status 0 with
  activation removed. Policy explain reports the checked ladder and manifest
  provenance through the existing proof graph.
- 100-run lint JSON and formatter checks: passed. Comments/trivia preservation,
  policy-artifact independence and declaration-policy composition: passed.
- Four new valid corpus files format and compile through the ordinary path.
  No valid else-if fixture is marked invalid because of style policy.
- Backend owner files, LIR, MachineIR, bridge codec and lowering/verifier files
  have an empty diff against the EVT2e5 baseline. No native algorithm or schema
  change is part of R9d.

Final serial `concept test libraries/<Package>` and the same command with
`--verify` passed:

| Package | Normal | Verify |
| --- | --- | --- |
| Standard | 42 passed, 0 failed | 42 passed, 0 failed |
| DragonGod | 23 passed, 0 failed | 23 passed, 0 failed |
| Golden | 130 passed, 0 failed | 130 passed, 0 failed |

Touched Go files are gofmt-clean and `git diff --check` passes. R9d is committed
on its branch; the final commit and worktree status are reported with the task.
Qualification logs and the unchanged-source audit/severity projects are retained
under ignored `artifacts/r9d-*` paths inside this repository.
Frozen root and legacy Zig suites are skipped because neither the legacy Zig
compiler nor its build/test infrastructure changed.

SUCCESS — R9d else-if compatibility and match-preference policy qualified.
