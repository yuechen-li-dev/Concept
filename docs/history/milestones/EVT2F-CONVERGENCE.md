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

## EVT2f2 — scalar/XMM contract qualification

Started clean at requested baseline
`9b1b9ea25d5f017b9146db676d02f914d348601f`, compiler
`concept-evt1-stage0-go`, on `codex/evt2f2-cmiramd4`. Baseline full regression was
verified in an isolated detached checkout of that exact commit (compiler
327.518 s, CLI 24.375 s, GPU-free Vulkan 0.411 s). Baseline vet and focused race
passed (5.561 s); Standard 42, DragonGod 23 and Golden 130 passed each in
Normal/Verify. The full lane covers semantic corpus, EVT2/EVT2x/EVT2e native,
frozen bytes and representative determinism. The isolated baseline checkout is
removed after its completed run.

Removed blocker: CMIRAMD3 could not represent scalar floating class/semantics.
The separate checked Concept CMIRAMD4 virtual schema now carries exact scalar
representation, GPR/XMM class, IEEE bit literals, arithmetic/negation/comparisons,
typed slots/arrays, ordered mixed calls, return and provenance identities.
Generated Go model/codec and Concept decoder are schema-derived. Existing
CMIRAMD3 schema/codecs/metadata/native fixtures remain byte-for-byte unchanged.

CMIRAMD4 is version 4/hash
`aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c`.
Its separate schema adds 13 records/four enums, 62 fields and 50 enum cases.
Real `concept generated` codec discovery rises from 41 to 73 declarations
(+32 including inherited v3). Strict v4 readers reject older identities and
stale v4 hashes; the public dispatcher deliberately retains a separate v3 reader.

The 32-function fixture now succeeds in LIR and MachineIR. Concept validates
the complete v4 artifact before native output/image mutation, then stops at
`AMD64_UNSUPPORTED_XMM_ALLOCATION`. This is the first true unimplemented backend
step. EVT2f3 must qualify physical XMM allocation, positional mixed Win64 ABI,
XMM clobbers, float spills and frame integration; subsequent SSE encoding/native
execution remain unqualified. No physical XMM, floating frame/ABI or SSE feature
was implemented in this tranche.

Focused tests passed: 100-run schema/codec/text/transport/digest agreement, a
20-byte exact transport golden, five digest-sensitive mutation families, 52
malformed artifacts in Go and compiled Concept, exact special values and special
comparison operands, typed mixed and void calls, Normal/Verify source-free
roundtrip and untouched output/native-image checks. The real CLI boundary test
passed (5.530 s). V3 generated-output agreement passed without regenerating any
frozen fixture. Informational size/time measurements and the Stage-0 classification
are recorded in the [conformance report](../../conformance/EVT2F-CONFORMANCE.md#evt2f2--cmiramd4-transport-qualification).

Final `go test ./... -count=1` passed: compiler 358.086 s, CLI 35.709 s,
GPU-free Vulkan 0.348 s, including corpus, EVT2/EVT2x/EVT2e native, frozen
integer/native bytes and representative deterministic gates. Final vet passed;
focused race passed (49.941 s while other qualification lanes
ran). Standard 42, DragonGod 23 and Golden 130 passed in each Normal/Verify mode.
Authored schema/validator check/format and the real R9d match-preference manifest
lint in both modes passed with no findings. Typed PlainData/NoAllocation
assertions were checked through the real backend compile. Optimized routine
shared-build reuse remains intact; opt-in 100 cold-build/probe stress was not
enabled. Frozen root/legacy Zig suites were skipped because their paths were
unchanged. Qualification logs are under ignored `artifacts/evt2f2-*`.

Implementation and tests are committed as
`8d7d726c4e6bbed6d83d0e5eed4e6d59563d73a2` (EVT2f2: qualify CMIRAMD4 virtual
scalar and XMM transport). The following qualification commit contains this
conformance history and authority/migration ledgers; its full identity and clean
tracked worktree status are reported at completion. After the final whitespace
formatting, compiled Concept admission/native-boundary and existing EVT2e5
native internal-call tests passed again (17.589 s, both Normal/Verify, real native
ABI sentinels and 100 image emissions). No frozen v3 artifact changed.

SUCCESS — EVT2f2 CMIRAMD4 scalar/XMM contract qualified.


## EVT2f3 — Cathedral XMM allocation and mixed Win64 ABI

Started clean at EVT2f2 baseline
`2f8443865aab1f9d1407735426c32f27751efb98`, compiler
`concept-evt1-stage0-go`, on `codex/evt2f3-xmm-planning`. An isolated detached
checkout inside the repository ran the exact baseline: full Go passed (compiler
336.831 s, CLI 38.020 s, GPU-free Vulkan 0.414 s), vet passed, focused race passed
(11.196 s), Standard 42 / DragonGod 23 / Golden 130 passed each in Normal/Verify.
That clean task-created checkout was removed after its qualification processes
completed. No other checkout was modified.

Concept now owns derived XMM0..XMM15 identities, XMM0..XMM5 allocation, reserved
XMM6..XMM15 rejection, typed mixed positional Win64 placement, XMM clobbers,
class-separated argument moves, floating pressure/live-across spills and their
integration into the existing one-frame finalizer. The unified placement core
also handles legacy integer requests; it does not introduce a FloatCallPlan.
Both four-argument mixed examples are pinned, as are mixed eight-argument slots,
XMM0 float results, bool/RAX results, void, cycles, repeated sources, pressure
and capture-before-reload. The new planning file has 29 functions; the existing
32-function file remains unchanged. Three early-admission negative source files
cover binary16, a four-lane record wrapper and a floating aggregate. Typed
negative tests cover varargs, reserved XMMs, mixed representation moves, spill
capacity, malformed geometry and stale FLAGS.

CMIRAMD3 remains version 3/hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`;
CMIRAMD4 remains version 4/hash
`aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c`.
No schema, generated codec/model/metadata, frozen native byte specimen or corpus
manifest changed. Derived physical state is not serialized into either contract.
Go changes are bootstrap diagnostic presentation and independent oracles only.

Qualification exposed two concrete storage regressions, both repaired before
completion. Duplicated per-instruction call-type metadata enlarged the
unoptimized frame bootstrap beyond its stack budget. Deriving exact call types
from the owning function map removed 22,016 bytes from BackendFunction and the
unoptimized frame harness passed. A full regression then found a legacy GPR
value at index 128 eagerly accessing that scalar map (Concept boolean operators
do not short-circuit). The shared AllocatedGPR discriminator now guards the
scalar-only map explicitly. Both EVT2x6 Normal/Verify native trace parity and the
frozen R9a2 pushdown native byte oracle pass; a dedicated highest legacy index
511 check protects the discriminator. Evidence for the initial failure is kept
in ignored `artifacts/evt2f3-before-capacity-fix-go.log`.

`TestEVT2f3ScalarAllocationABIAndFrames` checks 100 fresh plans per function for
61 functions, exact Normal/Verify assignment/frame/call-action observations and
independent C11 byte-stack replay. The C simulator uses explicit architectural
clobber tables and preserves exact float payload bits; it is not floating native
execution. All 256 four-source XMM graphs, simultaneous GPR/XMM cycles and typed
negative mutations pass. Existing source C11 oracle, EVT2f2 schema/artifact
100-run gates, 52 malformed artifacts and unchanged-output/image sentinels pass.
The normal CLI prints physical placement and concrete frame actions, then stops
at `AMD64_UNSUPPORTED_SSE_ENCODING`.

Implementation/tests are committed as
`46b2813bd4f1e2f1d2a8625c1895e1fb09758178` (EVT2f3: qualify Cathedral XMM
allocation and mixed Win64 frames). The following documentation commit records
conformance, authority and migration state; its full identity and the final
tracked-clean status are reported at completion.

Final `go test ./... -count=1` passed: compiler 366.025 s, CLI 62.003 s,
GPU-free Vulkan 0.398 s. This includes corpus, EVT2/EVT2x/EVT2e native paths,
frozen integer bytes, generated-output agreement, scalar source oracle and
representative determinism. Final vet passed; focused race passed (68.038 s).
The final strengthened scalar planning/input gate passed again (13.547 s).
Standard 42, DragonGod 23 and Golden 130 passed each in Normal/Verify. Real CLI
check passed for both authored backend files and the planning fixture, and both
scalar fixture files reached the SSE boundary. Format checks and real R9d
match-preference lint in Normal/Verify passed without findings for all six
authored Concept files. PlainData/StaticExtent/NoAllocation dogfood assertions
were checked through the backend compile. Frozen root/legacy Zig suites were
skipped because neither legacy compiler nor its test/build infrastructure
changed. Routine optimized shared-build reuse was retained; no opt-in cold-build
stress or heap fallback was enabled. Qualification logs remain under ignored
`artifacts/evt2f3-*`. `git diff --check` passed.

The [conformance report](../../conformance/EVT2F-CONFORMANCE.md#evt2f3--xmm-allocation-mixed-win64-abi-and-floating-frames)
records pools, exact placements, shared frame geometry, generalized eight-byte
temporaries, sample frame sizes, metadata growth and informational timings.
SSE encoding, native floating execution, SIMD/aggregates/varargs, nonvolatile XMM
save/restore and external/indirect calls remain unqualified. EVT2f4 was not begun.

SUCCESS — EVT2f3 Cathedral XMM allocation and mixed Win64 ABI qualified.
