# EVT2h9 phase qualification and ownership handoff

Status: **FULLY QUALIFIED for the bounded EVT2h native domain**. The complete
run, corrective full-suite reruns and companion results are distinguished below.

## Revisions and scope

Baseline `2c93386f02d218c667ef2b78c026d9ed25f5ae9b`, initially clean existing
`codex/implement-native-ordinary-enums` worktree at
`C:/Users/yuech/.codex/worktrees/evt2h4-profile-baseline/Concept`.
Fetched origin/main `065317d262c85ce159219561b0709526e7973c1f` is an ancestor
20 commits behind. No intervening valid work was discarded; no reset, branch
cleanup, remote push or merge was performed. Scope inventory commit `3e2060c`
precedes the complete run; test-only qualification changes are `7024514`.
Parent/child completion scope was corrected in `ee40b10`; the scheduler test start gate is `919d204`.
The final documentation commit contains this evidence and is reported in the
mission closeout. Historical h1–h8 records retain their original scopes.

Compiler `concept-evt1-stage0-go`; Windows AMD64, Microsoft x64 ABI.
Go 1.27.0 windows/amd64; GCC 15.2.0 (MSYS2 UCRT64 Rev13); clang-cl 22.1.3,
target x86_64-pc-windows-msvc; MSVC 14.51.36231 tools with ml64 and the installed
x64 developer environment. Vulkan SDK 1.4.350.0 headers are available. These are
actual local toolchains, not a portability or general C++ ABI claim.

The [frozen scope inventory](../../design/EVT2H9-QUALIFICATION-SCOPE.md) records
source legality, checked LIR, transport, Concept admission, realization, ABI,
independent evidence and deferred boundaries for each h1–h8 capability. No
production compiler/backend/language/codec implementation changes were made for
h9. Only test orchestration and adversarial assertions changed.

## Fresh generation and adversarial additions

The major checkpoint retains the existing 100-trial gates and enables
`CONCEPT_EVT2H_PHASE_QUALIFICATION=1`. The h8 harness already generates, writes,
admits and emits 100 separate fresh artifact files per form. Older h1–h7 tests
had separate 100-source-generation gates and 100 emissions from one artifact.
`phaseFreshNativeHost` closes that evidence gap: the existing native tests now
produce 100 separate freshly parsed/checked/encoded files, require canonical
byte equality and independent Go decoding, and route each native emission
through a newly read file and the corresponding generated Concept codec's
validated roundtrip. Ordinary Concept EmitModule then performs its own
admission, physical allocation/frame/call planning, encoding and fixups.
An exit check requires all 100 distinct inputs to have positive native emissions
in every parent native-only host. Isolated guard-page/trap children intentionally
emit one image and do not owe the parent's determinism loop. Existing plan/debug/byte/image/sentinel assertions,
varied executions and independent C11/ABI probes remain unchanged. No replayed
artifact, semantic result, physical plan or emitted image substitutes for a fresh
trial. Immutable compiler/tool builds remain shared. The hook is inactive for
ordinary milestone runs, preserving their existing sample scheduling.

This covers h1 ordinary enums, h2 tagged values, both h3 match domains, h4
Options, h5 Results and h7 fallibility. h6 runs 100 fresh typed proof/semantic
artifact/LIR/transport checks, with runtime erasure exercised by its residual
probes and h8's complete 100-pipeline assertion/counterfactual cases. Verdict
remains comptime-only; its evaluator is still Go-owned. The phase does not
manufacture a runtime Verdict ABI or a second proof lattice.

The original native parser bug is permanently named
`IncomingRegisterOverlapParserAddSevenReturnsNine`, a Normal/Verify native
subtest backed by the actual emitted parser invocation, complete report fields
and read-count assertion. It is additional to all 16 vectors and 40 interleavings.
The unused-scalar probe and the original 23-function controller prefix remain
byte-pinned along with the complete controller image.

The g5 native test adds different-valued arguments to previously equal-valued
vectors: eight direct records use offsets 1/2/4/8/16/32/64/128, paired indirect
records carry different fields and a hidden return, and BoolMix combines distinct
records, integer/bool, float/double, references and fifth-to-eighth stack slots.
All old cases remain. Separate strict-C11 domain results check the new vectors.
This tests preservation before destructive entry writes beyond the original
parser example and without changing any source feature or expected native byte.

| Required ABI property | Actual evidence family |
| --- | --- |
| Several incoming aggregates / disjoint live carriers | h8 Evaluate/FromRunning and derived carrier checks; g5 distinct PairValues and EightPositions |
| Direct/indirect passing and hidden-result shifting | g5 small/large records, Shifted/Mixed/Forward; h2/h4/h5 independent sum probes |
| Integer/bool/float/double and late stack positions | New distinct BoolMix vectors; existing mixed scalar/aggregate and Option/Result calls |
| Caller-live values, nested calls, recursion and spills | g5 Live/Pressure/Forward, h3 SpillMatch, h4/h5 FloatingPressure/Recursive, h7 Pressure/Recursive |
| Payload/Result transfer, active storage and isolation | h2–h8 native/C11 fixtures and constructor/copy/match/handler negatives |
| Caller clobbers, callee saves, RSP/shadow and raw sret identity | Independent MSVC and clang-cl DLLs plus ml64 boundary observer |
| Ordered evaluation and value copies | g5 ValueMutation/Isolation, h5 ArgumentOrder/RecordOrder, h8 read/write/trace and SequenceBeforeBranch |
| Original loss and entry beside an unused scalar | Named expected9 parser case and UnusedScalarAfterEvent, native C callers plus independent C11 domain |

Cathedral's generated caller temporaries remain 16-byte aligned. The previously
observed simple clang-cl incoming temporary at alignment 8 / [rsp+40] remains
documented; richer probes observed 16 with both compilers. These are separate
observations. Passing this phase does not resolve the discrepancy, certify every
foreign aggregate call, or qualify C++ std::variant/exception interoperability.

## Frozen contract inventory

All42 tracked bridge schema/codec/validator/metadata files have identical
canonical Git bytes to baseline. Each version's generated-output, schema, hash,
malformed and byte fixtures run through the full suite. No wire version or
physical field was added. Schema SHA-256 constants:

| Wire | Schema SHA-256 |
| --- | --- |
| CMIRAMD3 | `3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312` |
| CMIRAMD4 | `aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c` |
| CMIRAMD5 | `7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60` |
| CMIRAMD6 | `67d71de72ece8b4b6ce438840543e6220fd3ddcc035e2142147c679b4c77e73d` |
| CMIRAMD7 | `74992e3d74be0581484be7f3c6341b7e75654d59b76255f075771367858d33c2` |
| CMIRAMD8 | `8dc34f916ab8293e838db026d720825f10d2a7c12006c19924a0fffea94c0711` |
| CMIRAMD9 | `77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c` |
| CMIRAMD10 | `ea57ae74f5ae54ece90ce429edbc4a41ce309d106582c84fdb5d70d15242941d` |

V10 magic is CMIRAMDA, numeric version 10, with the existing 44-byte header. The
fresh test adapter is file I/O/orchestration around production codecs/emitter,
not another transport. The local 42-file SHA-256 inventory is
`artifacts/evt2h9/frozen-contracts.json`.

## Gates and reproducibility

Full run: `./test.ps1 -Lane Qualification -ProfileDirectory artifacts/evt2h9/full`.
It invokes `go test ./... -json -count=1 -parallel=4 -timeout=75m` with the
phase-100 marker scoped/restored by the existing script. The initial test failure
prevented its automatic vet; the final source is checked by separate vet below. The deadline
increased from 45 to 75 minutes to allow the added fresh major-checkpoint work;
no repetitions, oracle, mode, test, expected byte or validation was removed.

Local full evidence: tests.jsonl, commands.jsonl, timing.json, summary.json,
workflow logs, run.json and five-second memory.jsonl snapshots under
`artifacts/evt2h9/full`. The profile captures nativeCommand calls and their
process CPU, immutable object build/reuse and Cathedral generation. It does not
count every descendant or uninstrumented process as a native command. The
sampled tree RSS is a five-second observed maximum, not an exact OS peak.

Supplemental commands retain their own logs/timing outside that directory:

```powershell
go build -o artifacts/evt2h9/concept.exe ./cmd/concept
concept test libraries/Standard --module-root libraries --verbose
concept test libraries/Standard --module-root libraries --verbose --verify
concept test libraries/DragonGod --module-root libraries --verbose
concept test libraries/DragonGod --module-root libraries --verbose --verify
concept package build Standard
concept package build DragonGod

# With CONCEPT_EVT2H_PHASE_QUALIFICATION=1:
go test -race ./internal/concept -run '^Test(EVT2h[1-9].*(Transport|Checked.*|Fresh.*|Determinism|Source.*|Boundaries|LIRAdmission|ConstructorContracts|GeneratedContract|MalformedContracts|ImportedIdentity|VerdictErasure|ProofAuthority|LIRMutations|MalformedIntegration)|NativeFixtureHeaderClosure.*|RuntimeWorkflowPolicy|RuntimeConceptSpecifications|BackendConceptSpecifications|RunnerReusesBuildsAndIsolatesProcesses|NativeFixtureObjectReuseTracksInputs|R9b2TypedProvenCannotAuthorizeFacts|R9b3UnknownRendererContract)$' -count=1 -v -parallel=4 -timeout=15m

go build ./...
```

`cl` was on PATH but the initial shell lacked INCLUDE. Eight old EVT1
MSVC/std/Vulkan C11 smoke wrappers skipped. Loading the installed vcvars64.bat
in an isolated qualification process and rerunning their exact family selection
passes all eight with no skips, including Vulkan headers and no GPU requirement.
No compiler source or test was modified to mask that environment issue. Other
full results are reusable because the source state is unchanged and those
families either use GCC or already run their own developer setup.

```powershell
# Installed MSVC x64 developer environment, phase marker 1:
go test ./internal/concept -run '^TestEVT1(Language|Vulkan|TemplateLanguage|TemplateVulkan|M1BCLanguage|M1BCVulkan|M1BDLanguage|M1BDVulkan)SpecimenNativeC11$' -count=1 -v -timeout 5m
```

Changed Go files pass gofmt; the six h8 source/fact files plus schema 10 and
validator 10 pass CLI format/lint. Actual `generated` inspection of codec 10 uses
semantic artifacts from the real Standard package build and succeeds. The parser
has no generated declaration, so its inspector's expected no-match diagnostic
is recorded rather than called a successful generation. Full Go also runs
formatter/corpus, checked-output, bridge-generation and architecture/workflow
policy tests. `git diff --check` and final worktree checks complete closeout.

One redundant immutable build key was removed after inspecting the actual build arguments: h1's native-only selection
macro now lives in the host translation unit, like the other native harnesses,
instead of changing the entire Cathedral compiler flags/build key. The flag is
not used in Cathedral source. Its pinned native bytes remain unchanged. This
is immutable-build reuse, not test-result or fresh-pipeline caching.

## Final executed gates and measurements

Status: **FULLY QUALIFIED for the bounded matrix below**. This is aggregate
final-state evidence from the complete uncached run and complete affected-suite
reruns. The first full command did **not** exit successfully: retain its exit 1,
958 passing / 3 failing / 14 skipped families. All three failures and eleven
prerequisite skips are resolved below. The final evidence covers **975 top-level
families: 972 passed, zero unresolved failures, three documented optional/worker
skips**, plus **1859 named subtests**. The `cmd/machinebridgegen` package has no
test files; the three tested packages are CLI, compiler and Vulkan profile.
No result caching is used. The [machine-readable evidence](EVT2H9-QUALIFICATION-RESULTS.json)
retains the raw counts, revisions, resolutions, timings, profile scope and 21
log SHA-256 values. Local raw logs remain under `artifacts/evt2h9`.

| Gate / actual command or family | Count and result | Wall time / log |
| --- | --- | --- |
| `./test.ps1 -Lane Qualification -ProfileDirectory artifacts/evt2h9/full` | Complete Go/native/semantic run; original exit 1 with three test-harness failures, all resolved by full selected reruns | Go 3652.630 s; observed workflow 3656.930 s; `full/tests.jsonl`, `full/run.json` |
| `go test ./internal/concept -run '^TestEVT2h5NativeResult$' -count=1 -v -timeout 20m`, phase marker 1 | PASS; 100 distinct fresh files/emissions and plans, native-only/C11/ABI; Normal 468.70 s, Verify 491.51 s; unchanged bytes | 967.165 s package / 972.146 s shell; `result-rerun.log` |
| `go test ./internal/concept -run '^TestEVT2h4NativeOption$' -count=1 -v -timeout 15m`, phase marker 1, native child budget 8m | PASS; same 100-file, native/C11/ABI, guard-page and trap requirements; Normal 104.61 s, Verify 110.19 s | 220.051 s package / 220.571 s shell; `option-rerun.log` |
| `go test ./internal/concept -run '^TestNativeSchedulerWorkers$' -count=5 -v -timeout 8m` | PASS five complete repetitions of all original worker/quantum/state/failure/shared-agent/wake cases with the ready/start gate | 119.418 s package; `scheduler-final.log` |
| Eight exact EVT1 MSVC C11 smoke families, installed x64 developer environment | PASS all eight, zero skips; Vulkan header cases GPU-free | 1.370 s package; `msvc-environment-rerun.log` |
| Three exact TinyXML2 native artifact families after pinned submodule initialization | PASS identity/stale-evidence chain, 100-run native artifact/C generation and wrong-layout rejection | 9.413 s package / 9.986 s shell; `tinyxml-rerun.log` |
| h1–h8 focused families inside full run | 67 families, including the intentional worker parent skip and two subsequently resolved failures; all mandatory h families pass with final reruns | h8 486.56 s, h7 610.70 s, h3 432.13 s, h2 70.60 s, h1 52.51 s; `full/tests.jsonl` |
| Earlier EVT2e/f/g regressions | PASS 71 families / 160 named subtests; includes native scalar/SSE/calls/references/objects, fieldwise copies, frames, bytes and malformed input | g5 native 108.61 s; independent external ABI 6.34 s; f5 mixed 203.36 s; full log |
| EVT1 semantic manifest and negative-runtime/compatibility checks | PASS 42 subsystems: 444 valid, 357 static-invalid, 15 runtime-negative; five compatibility and four expected-divergence entries retained | Manifest 0.57 s; forbidden runtime 0.43 s; full log |
| Corpus/Conformance-named families | PASS 40 families / 725 named subtests; counts are test names, not the manifest's unique specimen count | Full log |
| Golden | PASS eight Golden-named families; all nine permanent domain goldens in Normal/Verify, source-free artifacts and retained 100-run generation | Domain family 48.01 s; full log |
| GPU-free Vulkan | PASS 16 Vulkan-named families after four MSVC prerequisite reruns; Vulkan profile Go package also passes | Library 8.14 s, examples 3.09 s, profile package 0.802 s; full and MSVC logs |
| Standard `concept test`, Normal / Verify | PASS 44 facts + three benchmarks in each mode | 53.914 / 49.315 s; `Standard-Normal.log`, `Standard-Verify.log` |
| DragonGod `concept test`, Normal / Verify | PASS 23 facts + one benchmark in each mode | 5.613 / 5.670 s; `DragonGod-Normal.log`, `DragonGod-Verify.log` |
| `concept package build Standard` / `DragonGod` | PASS real semantic/package artifacts; full Go also passes deterministic package builds (47.62 s) | 16.877 / 16.762 s companion commands |
| Focused race selection shown above, phase marker 1 | PASS proof/comptime/artifact/transport/fresh generation, workflow/process isolation and input-cache invalidation | 416.270 s package / 429.181 s shell; `race.log` |
| `go test -race ./internal/concept -run '^TestNativeSchedulerWorkers$' -count=1 -v -timeout 5m` | PASS final scheduler harness; Go race instrumentation does not claim to instrument external C threads | 143.635 s package / 158.330 s shell; `scheduler-race.log` |
| Final `go vet ./...` / `go build ./...` | PASS on final test revision `919d204`; full script's automatic vet was not reached after its initial failure | 1.386 / 1.246 s; `final-vet.log`, `final-build.log` |
| Final `./test.ps1` | PASS Concept runtime/leaf facts Normal/Verify and CLI/policy fast lane; C bootstrap facts remain distinct from native evidence | 24.75 s reported; `final-quick.log` (initial concurrent quick 25.28 s) |
| Real single-input CLI cold smoke / `TestAMD64CLIBatch` | PASS original cold source path plus all 20 cases through one shared backend build | 44.33 / 307.18 s; full CLI package 355.124 s |
| Generated code, format/lint, architecture/workflow and diff | PASS generated Concept codec 10 inspection from actual Standard artifacts; six integration source/fact files + schema/validator 10 format/lint; changed Go gofmt; 12 Policy/Workflow-named families; final diff/integrity checks | `generated-codec10.json`, full test log and final check logs |

The three retained skips are explicit existing optional boundaries:

- `TestCollectorHundredRunDeterminism`: separate opt-in R7fR collector burn-in;
  collection implementation is unchanged and this is not EVT2h's 100-trial gate.
- `TestEVT2h3ArtifactProcessWorker`: intentionally skips in the parent invocation;
  its passing determinism driver actually executes 100 independent workers.
- `TestGeneratedCodecsReadAndWriteSharedOctBytes`: adjacent external Oct fixture
  files are unavailable at this worktree-relative path. No other repository was
  modified to manufacture that optional interoperability environment.

Root/legacy frozen Zig suites were not scheduled because no legacy Zig compiler
or build/test infrastructure changed. Existing Go-hosted Zig tool probes still
ran; this is not a blanket claim that no Zig executable was launched. No required
native/C11/ABI, Normal/Verify, malformed/sentinel, corpus/library or frozen gate
was skipped. No GPU execution is claimed.

### Throughput and determinism scope

The observed qualification window is **3656.93 s (60.95 min)**, including the
main complete run; companion checks and corrective reruns overlapped that
window. Do not sum their individual durations. Early local g5/enum development
smokes, environment setup and final documentation review are outside that
measurement. The historical **1319.57 s** h4 result has different coverage:
this closeout adds h5/h7/h8 phase-100 execution, later regressions and 100 fresh
codec/admission paths. It is not an equivalent-coverage speedup comparison.
No timing target or repetition count was weakened.

| Measured main-run boundary | Actual observation |
| --- | ---: |
| Cathedral source generation requests | 115: two cold (Normal/Verify), 113 reused |
| Cold source generation time | 37.165 + 39.346 s |
| Full AMD64 C backend object compilations | 10, five genuinely different flag sets per mode |
| All instrumented native C objects | 19 cold / 139 reused (10 full backend + four domain oracle + five tiny cache-validation objects cold) |
| CLI builds and backend launches outside that object cache | Two real cold source/C builds; 20 batch processes + one single-input process |
| Instrumented nativeCommand launches | 2039 (includes compilers and test executables; excludes uninstrumented descendants and CLI) |
| Instrumented compiler launches | 996 GCC/Clang/MSVC boundary invocations, including syntax-only cases |
| Observed nativeCommand process CPU | 3331.453 s; excludes descendants |
| Extra corrected Option/Result reruns | Four cold source/C mode builds and 24 instrumented launches; immutable sharing stays within each invocation |
| Passing h1–h8 C11 oracle compilation | 24 oracle host compiles, including eight h6 residual-erasure hosts, plus four separately compiled h8 domain objects; independent ABI compiler jobs separate |
| Sampled main process-tree RSS maximum | 2423832576 bytes (2.257 GiB), 707 samples nominally five seconds apart; separate companion processes excluded |

The main critical serial families are Result, local fallibility, integrated h8
and match. Each now executes full fresh admission/planning/emission rather than
only replaying a canonical artifact. Cold Cathedral generation is about 76.5 s,
a small part of the complete window. The 10 C objects are not 10 identical
rebuilds: optimization, strict floating flags and modes differ, and those keys
must remain distinct. The h1 host-only macro fix shares its compatible O2
backend object; actual full profile events show reuse. No controlled old/new
whole-suite speedup is claimed. Remaining cost is predominantly repeated
semantic/native qualification work, not one cold build per fixture.

All phase runtime forms retain 100 genuinely fresh checked generations, separate
artifact files, generated Concept roundtrip/admission, physical allocation/frame/
call plans, encoding/fixups and stable image records. Normal/Verify bytes match.
H3 covers both tag-only and payload forms. H8 covers parser, controller, asserted
and assertion-free programs with all 16 parser vectors, 40 controller combinations
and repeated/interleaved calls. Its named expected-9 regression passes in both
modes. H6 separately checks 100 fresh typed proof/semantic artifact/LIR/MachineIR
samples for each of three specimens; four residual native probes run in both
modes, while h8 supplies the 100-pipeline asserted/counterfactual execution gate.
There is no runtime Verdict representation or claim of one. Existing earlier
100-run/debug/fixup/byte and malformed sentinel checks remain intact.

Representative host clock totals for 100 h8 planning/emission pipelines are
parser 114.039 s Normal / 100.131 s Verify and controller 116.233 / 99.068 s;
asserted/plain 3.210/3.191 s Normal and 2.477/2.365 s Verify. These totals include
the harness's fresh pipeline work, not isolated encoder microbenchmarks. H7's
100 explicit plan fingerprints take 6.698 / 6.586 s, and its initial emission
reports 2.283 / 2.268 s. Actual timings vary under concurrent qualification.



## Native images and unchanged byte pins

The complete images below use the historical pinned SHA-256 values. The
asserted/counterfactual pair has different semantic artifacts and identical
native images. A hash check is coupled to real native execution and independent
C11 behavior; a hash alone is not semantic evidence.

| Native form | Bytes | SHA-256 |
| --- | ---: | --- |
| h1 ordinary enums | 8440 | `76425ddc81825b0897c1efdeb515e502ca416e76b8090dc6c37c7a7b97faaf77` |
| h2 tagged values | 31871 | `524c364540dbb5efc8b1634f5d7087f8933fd4225d92afaa73dc39bccd7becc3` |
| h3 ordinary match | 3772 | `48dee1c8a55f462854fe930e64fb34d3eefac77d0c18a2d6c6cdfdc4ca40e83b` |
| h3 payload match | 122509 | `398445c4b5e1bf1ff09c190f897843fb58985fe96625295e71f8ba6f2628484f` |
| h4 Option | 45901 | `5d5b8746bd74e74b50d40f8cc7dbe5e680597c6c00ba732d33be33fa8bc90316` |
| h5 Result | 107444 | `34c260bb98b31437371fcc19df53a8f192dd9da870bcd795dc27521ddbe5c464` |
| h7 fallibility | 82687 | `78407eeb465a416e7a406d05d3e382cb9c8c21049f8162c4e7d504196907c242` |
| h8 parser | 61259 | `a7b2383995d4d64d7b6ec1d33935865580806319f7743fda6657ec1e788bff35` |
| h8 controller | 55330 | `e23ab5e5ffdb59bc3421f1e02c4184743816ad7746207450371c45d14aa23887` |
| h8 asserted and plain, each | 3381 | `951e78d24abe10bc965982415e74f70b6bccabdcbba940c44a38d27c2978dbde` |

Largest EVT2h image is the 122509-byte payload match specimen; this is bounded
image output, not an object-file or general executable-linking claim. That
specimen exercises 11 spills and frames 40..1928. Result has six spills and
frames 40..488. Parser/controller frames are 120..1128 and 40..632 with zero
spills; asserted/plain frames are 72..152. Earlier g5 separately exercises the
4088-byte near-probe frame with 100 sret/field/padding-isolation probes. Frames
at or beyond the existing 4096-byte unprobed fence remain rejected.

The h8 artifacts remain 143357/158422/12690/12654 bytes for parser/controller/
asserted/plain, with their historical hashes asserted in checked tests. The
original controller prefix remains 52892 bytes,
`045a90da6389f2881769582a3c109ba108556812de5d4d79964454168d4348bf`.
No expected byte, schema hash or existing golden was updated.

## Qualification failures and corrections

The first complete invocation is retained as evidence, including failures. Its
Result Normal/Verify hosts reported `line 177: status==0` after both parent
100-file loops completed. The new h9 completion adapter also ran at normal exit
in the existing guard-page child, which deliberately skips the parent loops.
It therefore returned harness status 74 after a correct empty-success read.
`ee40b10` makes the requirement parent-only using the host's established argc
convention and rejects unrecognized main signatures. It does not weaken the
100-file parent check, child read checks, illegal-instruction observers, codecs,
admission or native byte assertions. Result and Option receive complete
100-sample Normal/Verify reruns on this correction. Other argc hosts only have
trap children that terminate via the existing exact exception observer; their
successful parent path and all runtime native bytes are unaffected. The void
main enum host is unchanged apart from the unused child-scope flag.

The same full run reported `TestNativeSchedulerWorkers` status 6: only one
worker participated in one of the existing multiworker cases. Three isolated
unmodified repetitions passed, exposing scheduling sensitivity rather than a
repeatable Concept state error. The C harness launched worker one before
creating worker two, so its tiny workload could finish before all participants
existed. A local C11 atomic ready/start gate now releases the workload only
once all workers are ready. The original worker/quantum/trial counts, at-least-
two-active requirement, exact decision totals, final states, failure isolation,
shared-agent and wake-race tests remain. Failure output now identifies worker
count, quantum, trial, active count and decisions. Five full repetitions plus
the selected final race check exercise this test-only setup change, committed as `919d204`. Neither
DragonGod production code nor the shared threading shim changed.

Final test revision is `919d204`; only documentation changes follow it.
Final evidence reuses only unaffected full-run results: production compiler,
Concept backend, fixtures, frozen codecs and external probes are identical to
the baseline. The process-scope fix affects native child harness bookkeeping;
the scheduler gate affects only its C host. The changed suites are rerun in
full, with all repetitions. Vet/build/quick/format/diff checks are checked on
final test sources. No failed gate is converted to a warning or changed golden.
The exact initial/final counts and times appear above and in retained logs.

Three TinyXML2 native artifact gates initially skipped because the pinned
submodule was uninitialized. `git submodule update --init --
tests/dogfood/tinyxml2/upstream` checks out the recorded
`8224e427b655b83dae5e2298f1e6919523a78737`; all three then pass, with no source or
expected-output edits. The compiler/toolchain they discover works in the default
shell; no MSVC environment claim is inferred for these separate tests.

## Capability classification and limits

The following classifications apply only to the stated bounded domain on the
qualified revision. The per-stage source/LIR/admission/ABI inventory is frozen in
[the scope matrix](../../design/EVT2H9-QUALIFICATION-SCOPE.md).

| Capability | Phase classification | Exact supported boundary |
| --- | --- | --- |
| h1 ordinary enums | FULLY QUALIFIED | Checked nominal variants/tags, locals/copies/assignments/equality, branches/loops and qualified direct calls/returns; v7 or v9/v10 aliases |
| h2 payload enums | FULLY QUALIFIED | Copyable integer/bool/float/double fields, nested copyable records/structs and qualified fixed-array fields; active-field construction/copy/access, direct/indirect arguments and eligible/hidden returns; v8–10 |
| h3 match | FULLY QUALIFIED | Existing exhaustive tag/payload/integer patterns, integer default where legal, value bindings, subject once, nested branches/loops/returns, scalar/aggregate joins; accepted subjectless ordered guards lower through existing conditionals; v9/v10 |
| h4 Option<T> | FULLY QUALIFIED | Qualified closed scalar/record/fixed-array Some/None applications, replacement/copy/match and aggregate ABI; no niches; v9/v10 |
| h5 Result<T,E> and ? | FULLY QUALIFIED | Qualified closed success/error domains, Result<void,E>, constructors/copy/replacement/match, exact E propagation, admitted explicit remaps, nested/recursive calls and spills; v9/v10 |
| h6 typed Verdict and assertion erasure | FULLY QUALIFIED for comptime contract / residual native code | Go-owned bounded evaluator and single proof lattice, selected typed payload/provenance and descriptive semantic artifacts; assertions erase, residual bytes match counterfactual; no runtime Verdict carrier |
| h7 ! and try/except | FULLY QUALIFIED | Success extraction or real UD2, nearest exact Result handler/function destination, Option function propagation, void success, nested handlers, joins and existing explicit remaps; v10 |
| h8 parser/controller and and/or | FULLY QUALIFIED | Existing 16 parser vectors and 40 controller combinations, interleavings, typed records/enums/sums/handlers, while/short-circuit CFG, pure-conditional reference witnesses, operand ordering and incoming materialization; v10 |
| Earlier e/f/g native execution | FULLY QUALIFIED for existing documented domains | Direct scalar/floating calls, memory/qualified borrows, aggregate calls and one unified allocator/frame/encoder; unchanged historical limits and independent probes |
| General ownership/Drop, move-only/owning payloads and early-exit cleanup | DEFERRED | Existing source/C semantics do not imply native cleanup; unsupported native values reject |
| Unqualified reference-bearing payloads, escaping references | UNSUPPORTED natively | Qualified bounded function borrows remain supported; payload/escape authority is not inferred from an address |
| Nested sum representations | UNSUPPORTED natively; SOURCE/C BOOTSTRAP ONLY for accepted forms | Requires separate checked layout/copy/active-authority evidence; no integer-blob fallback |
| Raw local arrays | SOURCE/C BOOTSTRAP ONLY | Native local array declaration projection remains fenced; arrays inside qualified records/payloads work |
| Local address temporaries preceding actual failure exits | UNSUPPORTED natively | Explicit existing unsupported diagnostic; pure conditional-CFG references are separately qualified |
| Nested-control-flow function remaps in v10 | UNSUPPORTED | Frozen v10 requires the admitted straight-line failure transfer/return; older qualified v9 remaps remain intact |
| Runtime Verdict storage/ABI or typed evidence granting trusted facts | UNSUPPORTED | Comptime-only source restrictions and registered-provenance boundary remain enforced |
| General foreign enum/Result/C++ ABI, exceptions/unwind | DEFERRED | Independent probes qualify explicit private shapes and Win64 observations, not std::variant or general external representation |
| Indirect/variadic calls, general native external linkage, COFF/object output | UNSUPPORTED by the bounded native contracts | Source/C capabilities where accepted remain distinct from Cathedral's closed generated direct-call image |
| New optimization passes or self-hosting migration | DEFERRED | No such work began in h9; Concept-authored predicate bodies do not mean the evaluator migrated |

Counts/limits remain those of the frozen contracts and native realization:
128 module functions, bounded types/domains/blocks/scopes, 64 object slots and 128
scalar carriers per native function, existing image/fixup capacities, and the
frame `< 4096` stack-probe fence. The tested near-boundary native frame is 4088.
Capacity or unsupported-form errors retain deterministic preflight rejection.
The parser input is eight words with a four-iteration command bound; these
examples do not claim arbitrary protocols or unrestricted integer ranges.


## Ownership/Drop handoff

This is a qualification closeout, not the start of ownership implementation.
The next reviewed phase should first inventory existing source cleanup rules
and carry explicit initialization/move/destruction obligations through checked
LIR and every scope/return/failure edge. Current active-variant authority and
nominal applications must remain intact. Reuse the object copier, allocator,
Win64/frame/encoder facilities; raw representation copying is not ownership
permission. Scope exits, reassignment/variant replacement, partial initialization,
match-arm exits, ? to handlers/functions and ! traps need separately qualified
cleanup semantics. Do not infer an unwinding runtime from today's UD2 behavior.
Any new bridge is a separately justified Concept-owned/generated contract.
Trusted-fact production and the Go-owned comptime evaluator do not migrate
merely because typed evidence exists. No EVT2i, Drop, unwind, COFF, optimizer or
new backend work begins in this task.
