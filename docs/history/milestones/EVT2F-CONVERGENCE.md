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


## EVT2f4 — Scalar SSE2 native arithmetic (latest optimization baseline)

The user superseded the original baseline request and directed use of latest
because NURBS/SMLIB optimization work had landed separately. Started with a clean
`9d57d649b0edca082c0af6f15b9827a05ef363e1`, compiler
`concept-evt1-stage0-go`. That checkout lacked EVT2f1–f3, so their prerequisites
from `6f03274541fcb9a6426c99b04dfbae3e0bf21233` were reconciled cleanly with the
latest changes. Checkpoint `aef7824adfd720f00bfdad28e882077879c8d7fa` preserves that
integration; no optimization work was reverted. The active checkout is now main.

Isolated baseline full Go qualification passed at both requested history and
latest: original EVT2f3 compiler 400.246 s / CLI 58.691 s / GPU-free Vulkan
0.451 s; latest optimization baseline compiler 400.189 s / CLI 27.504 s /
GPU-free Vulkan 0.452 s. Task-created detached checkouts were removed after their
processes completed and their clean state was checked; logs were copied to the
active checkout's ignored `artifacts/evt2f4-*`. Other checkouts were untouched.

Removed blocker: Concept now selects and encodes MOVSS/MOVSD, ADDSS/ADDSD,
SUBSS/SUBSD, MULSS/MULSD, DIVSS/DIVSD and UCOMISS/UCOMISD. Negation flips bit
31/63 through MOVD/MOVQ and BTC. Immediate materialization preserves exact IEEE
bytes without decimal conversion, literal pools or process-specific addresses.
All six comparisons guard unordered cases correctly; bool results remain GPR.
Branch fixups resolve before native-image finalization and RW-to-RX publication.

The existing allocator, class pools, reserved XMM6..XMM15 policy and one-frame
geometry remain authoritative. Native no-call frames request four generalized
move-temp slots for ordinal entry capture and bounded scratch. XMM5 is explicitly
borrowed and restored rather than secretly reserved from the allocator. Sources
remain live under destructive SSE operations, including noncommutative SUB/DIV
and pressure-spilled sources/results. RAX and parity scratch are preserved unless
they carry a result. No native floating call pipeline was added.

The real source -> verified LIR -> CMIRAMD4 -> allocation/frame -> Concept SSE2
-> finalized image -> typed Win64 entrypoint path passes in Normal and Verify.
The new fixture has 50 no-call functions and produces 5,332 bytes. Independent
strict-C11 oracles check eight arithmetic forms and multi-operation expressions,
negation, 588 bool comparisons and 588 branch cases, signed zeros, infinities,
NaN categories/payload copies, exact constants, scalar locals, checked incoming
array loads/stores, eight-parameter XMM pressure and quantity interpretation.
Primitive exact-byte fixtures pin the requested instruction families, store/
reload, negation and encoder-only high-XMM REX. Malformed typed operands,
representations, memory/displacements, constant widths and capacity reject.
A negative writer cursor is now rejected by the shared bounded byte writer.

One hundred complete images match bytes, symbol layout and metadata; Normal and
Verify match exactly. All 61 EVT2f3 planning functions and 100-run transport,
allocation/frame/move gates remain checked. CMIRAMD3/4 schemas and generated
codecs/metadata remain unchanged. Pure integer native modules retain frozen
CMIRAMD3 bytes. Source oracle, EVT2/EVT2x/EVT2e native paths and the frozen
pushdown byte oracle pass. Typed PlainData/NoAllocation assertions and the
existing StaticExtent-four table proof are checked through the real compiler.

New code/test implementation is committed as
`4af10109b373eade17e6e85baec41ea4a0d56ae2` (EVT2f4: encode scalar SSE2 and execute
no-call floating images). The following documentation commit records the final
qualification evidence; its full identity and final tracked-clean status are
reported at completion.

Final `go test ./... -count=1` passed: compiler 439.248 s, CLI 74.072 s,
GPU-free Vulkan 0.425 s. This includes latest optimization tests, semantic corpus,
EVT2/EVT2x/EVT2e native execution, frozen integer bytes, generated-output agreement,
EVT2f source oracle, all EVT2f3 planning gates and native SSE tests. Final vet
passed. Focused race including EVT2f3/transport/native passed (137.116 s); final
native-only race passed (117.325 s). The final native/CLI gate passed (compiler
62.325 s / CLI 11.762 s), and the strengthened primitive/error/timing gate passed
again (18.577 s).

Standard 44, DragonGod 23 and Golden 130 passed each in Normal/Verify. The latest
optimization baseline includes two additional Standard tests beyond the historical
42; they were retained. The unchanged latest semantic manifest is 444 valid /
357 static-invalid / 15 runtime-negative / five compatibility / four
expected-divergence, rather than historical EVT2f3's 440 valid. New EVT2f4 native specimens
are dedicated qualification files outside that manifest.

Real Concept check passed for the authored backend and native fixture. Canonical
format check and the real R9d match-preference manifest lint passed without
findings in Normal/Verify. Typed PlainData/NoAllocation assertions, the existing
StaticExtent-four proof, 100-run images/plans, malformed artifact/operand gates
and untouched output/image admission checks passed. Final whitespace/diff checks
passed. Frozen root and legacy Zig suites were skipped because the frozen Zig
compiler and its build/test infrastructure were unchanged. Routine optimized
shared-build reuse remained enabled; opt-in cold-build stress was not run.
Qualification logs remain under ignored `artifacts/evt2f4-*`.

One concurrent full run failed only `TestNativeSchedulerWorkers` with exit 6:
the harness requires at least two OS workers to receive work on every trial.
Its result/step-count assertions did not report a mismatch. The unchanged
scheduler test passed three isolated runs (14.440 s total), and latest-baseline
full qualification passed. No scheduler implementation, participation assertion
or test budget was weakened. The failure log is preserved in
`artifacts/evt2f4-before-scheduler-rerun-go.log`; the final full rerun is recorded
above.

Informational measurements are documented in the
[conformance report](../../conformance/EVT2F-CONFORMANCE.md#evt2f4--scalar-sse2-native-arithmetic).
Selection/encoding/constant samples and 100,000 native calls use millisecond
clock() resolution, without a performance threshold or superiority claim.
Locally initialized indexed aggregates and general integer arithmetic inside
mixed CMIRAMD4 remain unsupported scalar forms. Call-bearing scalar modules
stop at `AMD64_UNSUPPORTED_FLOAT_CALL_EMISSION` before output/image mutation.
Native mixed call chains, external/indirect/variadic/aggregate calls, live-across
actual floating calls, nonvolatile XMM saves, SIMD/AVX/FMA, binary16, conversions,
COFF and unwind/SEH remain unqualified. EVT2f5 was not begun.

SUCCESS — EVT2f4 Cathedral scalar SSE2 encoding and native arithmetic qualified.

## EVT2f5 — mixed scalar native call closure

Actual clean baseline: `64890e7d4927abd0d97e6329cd78777cfc0e8780` on `main`.
It contains the request's EVT2f4 baseline
`4c423a5ef548f92792b30329cf4cdfb00cc24771`, followed only by the requested AMD64
module split. NURBS/SMLIB improvements were preserved. Compiler identity remains
`concept-evt1-stage0-go`; CMIRAMD4 is v4/hash
`aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c`.
CMIRAMD3/4 schemas and generated codecs are unchanged.

Baseline focused qualification passed with shared build reuse: compiler 80.131 s
and CLI 9.394 s, including scalar SSE, 61-function planning, native integer calls
and frozen byte oracles. The immediately preceding clean-baseline full suite had
also passed during module-split closeout (compiler 479.249 s). No reset or
unrelated optimization change was made.

The shared `EmitAllocatedCall` now drives both integer and mixed scalar module
emission. Concept owns placement, physical allocation, typed concrete actions,
preservation, result capture, final frame geometry and native rel32 fixups. New
production code stays in the existing `AMD64.Model`, `AMD64.Analysis`,
`AMD64.Frame`, `AMD64.Encoding` and public `AMD64` modules. No allocator or move-cycle planner
was replaced. `Win64ABI` receives a narrow cycle correction: save the old
destination value with its own type, permitting a float32/float64 swap without
conversion. Native execution and the existing replay verifier qualify it.
Go production changes consist of one CLI presentation update
displaying concrete rel32 realization of abstract calls. Other Go changes are
qualification data/oracles and expectations for the newly supported path.

The frame layer reserves distinct cycle and scalar scratch identities and owns
incoming stack offsets. Temporary capacity is eight slots (up to four cycle
slots plus four scalar scratch slots); concrete action capacity is 64. Both
bounds reject overflow. Typed spill widths/alignments and the existing seven
regions remain. Bool captures zero-extend the payload. Scalar int32 constants,
slots, checked arithmetic and comparisons reuse the existing integer encoder
where mixed source fixtures need them. A pressure-entry correction consumes
staged GPR inputs before scratch can overwrite them. The EVT2f4 5,332-byte image
remains byte-identical in Normal/Verify; frozen CMIRAMD3 bytes are retained.

The 51-function source fixture emits 40 actual direct calls and an 11,744-byte
image. Both modes pass arities zero through eight, the two mandatory mixed
positional signatures, all eight mixed stack/register arguments, genuine void
calls, floating/bool/integer returns, RAX/XMM0 coexistence, two-/three-way cycles,
repeated sources, live-across values, mixed pressure, nested/multiple/recursive
calls, signed-zero/NaN/infinity pass-through and binary64 precision checks.
The strong native gate passed in 178.50 s (87.87 Normal / 90.61 Verify).

Independent strict C11 arithmetic/branch oracles and controlled native Win64
probes check register/stack placement, all 32 home bytes, entry alignment,
RSP/canary and nonvolatile GPR preservation. The GPR pressure fixture uses at
least three saved GPRs. Hostile callees replace all six volatile XMM values;
actual typed spill/reload execution retains the caller's live values. A typed C
void target records observable argument values. These foreign stubs are
harness-only RW-stage substitutions; no production external linker or
host-address-dependent deterministic image was introduced.

One hundred complete emissions per mode match bytes and symbol/fixup metadata.
Representative mixed/cycle/pressure plans receive 100 additional allocation,
frame, action and canonical-debug signature checks. One hundred independent
fixup replays reproduce the final image. CLI bytes/debug also match across two
independent builds. The CLI reports, for example, Call8 code size 146 bytes,
Cycle2 62, Cycle3 78, Live32/Live64 242 each and Pressure64 590. Informational
timings and frame sizes are in the [conformance report](../../conformance/EVT2F-CONFORMANCE.md#evt2f5--mixed-win64-floating-calls).

Ten new malformed mixed-call artifacts reject transactionally; existing 52
malformed CMIRAMD4 artifacts remain covered. Frame/action tests cover invalid
class, width, offset, storage size, temporary identity, scratch overlap and
capacity. Existing reserved-XMM, unsupported call-form/convention, stale-FLAGS
and move/replay gates remain. Passed NoAllocation assertions cover shared call
realization and typed action emission; existing PlainData and StaticExtent
evidence remains in force.
The contract-valid nine-argument capacity specimen is rejected before the
eight-entry native projection buffer is written, with untouched output/image
state in both modes (22.580 s focused gate). A separate four-function native
fixture qualifies positive forward fixups and a float32/float64 register swap;
the corrected 61-function planning/replay gate also passes. The final focused
race rerun covering that correction passed in 99.804 s.

Standard 44, DragonGod 23 and Golden 130 pass in Normal and Verify. Final vet,
canonical format checks, actual Concept source checking and Normal/Verify R9d
match-preference lint pass without findings. Focused LIR/call-contract/CMIRAMD4
and new native-call race checks passed in 249.459 s. Optimized shared builds
remain enabled; no 100-run cold-build loop was introduced.

Final `go test ./... -count=1 -timeout=20m -parallel=4` passed: compiler
641.795 s, CLI 84.650 s, GPU-free Vulkan 0.406 s. This includes the semantic
corpus, checked-output, EVT2/EVT2x/EVT2e native/frozen-byte, EVT2f1/f2 transport,
61-function planner/replay, EVT2f4 SSE and final EVT2f5 native/forward/capacity
gates. Final vet and diff checks pass. The longer outer Go timeout accommodates
the required 100-emission native stress; individual test assertions and runtime
timeouts were not weakened.

The first full concurrent run failed only
`TestRunnerExecutesFactsTheoryBenchmarkAsyncAndProphecy`: its child printed the
expected foretold panic and then reported `CONCEPT_TEST_TIMEOUT`. All backend
tests passed in that run. The unchanged runner test passed three isolated runs
(14.155 s total); the full suite passed after reducing outer test parallelism
to four. No runner implementation or assertion was changed. The failure log is
preserved as `artifacts/evt2f5-go-all.log`. A subsequent complete pre-capacity
run also passed (compiler 648.490 s); the final run above includes the capacity
guard and its regression.

The semantic corpus manifest remains 444 valid, 357 static-invalid, 15
runtime-negative, five compatibility and four expected-divergence cases. The new
EVT2f5 native fixture is dedicated qualification outside that unchanged manifest.
Root and legacy Zig suites are skipped under the frozen-path policy: neither
the legacy compiler nor its build/test infrastructure changed. Qualification
logs are under ignored `artifacts/evt2f5-*`.

EVT2f closure is bounded to fixed-signature internal scalar direct calls.
Aggregate ABI, indirect/variadic/unprototyped calls, production external linkage,
XMM6–XMM15 allocation, SIMD/AVX/FMA, binary16, numeric conversions, object output,
large-frame stack probing, and unwind/SEH remain deferred. No adjacent milestone
was started.

SUCCESS — EVT2f Cathedral scalar float/double native lowering and mixed Win64 calls qualified.
