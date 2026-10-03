# EVT2f convergence

## Checked scalar input progression

Baseline `df010b72a841fbcc29bef7f7a2461fc10b793d0a` contains requested
`aaa4cd57cceed05737f1998188f5b57e84c2be6a`; compiler is
`concept-evt1-stage0-go`. Work proceeds on `codex/evt2f-scalar-input`.

Removed blocker: the real LIR path previously rejected canonical float/double.
It now produces independently verified width-specific scalar operations,
exact IEEE constant bits, typed calls, storage and comparisons. Semantic
quantity checks precede projection; nonidentity dimensionless scales reject
explicitly. Same-representation interpretation emits no runtime operation.
No language syntax or source conversion semantics changed.

Next isolated blocker: both normal `concept machineir` and `concept amd64`
reject the 32-function scalar specimen at `MIR_FLOAT_REQUIRES_CMIRAMD4`.
CMIRAMD3 v3/hash `3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`
has integer width/address vregs, GPR-only register identities and integer/bool
call value tags. Exact scalar/XMM/constant vocabulary must be versioned as
CMIRAMD4 through the sole Concept schema and generated codecs. No schema
mutation or generated-declaration delta occurred in this progression.

Stage-0 retains checked semantic input and C source oracle responsibilities.
Cathedral XMM allocation, physical ABI planning, SSE lowering, spill/frame
realization and encoding are still unimplemented. The strict Normal/Verify
C11 source checks and 100-run LIR determinism are input evidence, not floating
native execution or native-byte determinism.

See [EVT2f conformance](../../conformance/EVT2F-CONFORMANCE.md) for the exact
qualification and ownership limits. This commit is the checked-input tranche;
it does not close EVT2f.

## Final validation

`go test ./... -count=1` passed: compiler 337.536 s, CLI 22.060 s,
GPU-free Vulkan 0.327 s. The full lane includes EVT2, EVT2x, EVT2e native,
frozen integer bytes, generated bridge/schema checks, semantic corpus and
routine independent heavy/determinism gates. `go vet ./...` passed. Focused
EVT2f/core/call LIR race passed in 8.700 s; the dedicated semantic corpus
manifest/determinism/runtime lane passed in 1.038 s.

Standard 42, DragonGod 23 and Golden 130 passed separately in both Normal and
Verify. The new strict C11 source oracle passed both modes; 100-run complete
floating LIR determinism passed. Fixture format/lint, Go formatting and
`git diff --check` passed. The historical float rejection test now expects the
precise MachineIR boundary rather than its retired LIR unsupported-type stop.
No XMM native evidence is inferred from these regression checks.

Optimized routine infrastructure was retained; explicit 100 cold-build/probe
stress was not enabled. Frozen root and legacy Zig suites were skipped because
no legacy compiler or build/test infrastructure changed. Local command evidence
is under ignored `artifacts/evt2f-*`; there is no background service or persistent
build cache. The final commit includes code, fixture, tests and qualification
ledgers; tracked worktree cleanliness is checked after commit.

MEANINGFUL PROGRESSION — checked float32/float64 LIR works; CMIRAMD4 scalar/XMM
transport is the next explicit backend boundary.
