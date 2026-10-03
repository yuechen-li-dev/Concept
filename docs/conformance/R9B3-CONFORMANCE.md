# R9b3 conformance

Compiler: `concept-evt1-stage0-go`. Baseline:
`b2e3388bd835687b8835d5358fb5732ccc921433` (clean checkout at task start).
Implementation and final documentation commits are recorded in R9B3-CONVERGENCE
and the final report. No next EVT2 milestone is started.

## Semantic result

PlainData is **ADMITTED** as an ordinary restrictive Standard concept. Its
self-contained semantic representation requirement is stricter than fixed layout
and independent of CAbiValue. Vulkan host Upload/Download enforce it; AMD64
checks existing WireHeader, WireMachineFrame and WireMachineOperand. Real module
checking evaluates the assertions even though their containing helper need not
run. They erase before runtime. Bridge codecs/schema/hash/wire format are unchanged.

R9B-RESEARCH-LEDGER gives the exact subject, truth, E/R types, Unknown causes,
provenance, trust, consumers and limitations for every candidate. Relocatable
requires unobserved bitwise invariants and a live compaction consumer; intrinsic
address restriction is separate from contextual pinning; finite enum tags are
separate from complete runtime values; closure requires an explicit boundary.
These precise deferrals are permitted by the task and do not grant hidden facts.
StaticExtent exact 1D, existing Contiguous and canonical Disjoint are retained.

## Executable qualification

| Boundary | Evidence |
| --- | --- |
| Scalar/record/array PlainData | bool, uint8/16/64, half, float, double, Header, fixed Words, closed DataBox<uint16>, payload-free Tag; each positive's byte evidence agrees with the existing geometry owner. C11 Header asserts sizeof=16, alignment=8 and Main returns 7. |
| Typed negatives | Drop resource -> HasDrop; owned record -> FieldProblem; borrowed/ref/string -> ContainsReference. Assert diagnostics pin CONCEPT_ASSERT_DISPROVEN. |
| Unknown | Opaque handle, payload sum, empty nominal, zero-length array, nested empty/zero field. CONCEPT_ASSERT_UNKNOWN, no refutation payload. Missing summary/geometry/initialization protocol is described conservatively. |
| ABI distinction | PlainData BoolHeader and half do not imply all extern-C requirements; ABI-valid foreign handle does not imply PlainData. Existing repr(C) and AsBytes rules stay authoritative. |
| Real Vulkan rejection | Validated native Vulkan companion artifact rejects Upload<ReferenceData> with FieldProblem and ReferenceData.text, and Upload<ForeignOpaque> with PREDICATE_REQUIREMENT_UNDECIDED and missing trusted representation summary. |
| Motivating before/after | Exact archived baseline parses/generates Upload<ForeignOpaque>. Final ordinary requirement rejects Unknown before byte transfer. Baseline already rejected owned/reference transfers via C ABI restrictions; no false claim that these newly became illegal. |
| Artifact-only | Provider and subject artifacts supply all imported types/predicate/helper bodies with no source dependency. Native artifact-only Use returns 7. CLI roots contain only dependency .concept-module.json files, no producer .concept files. Proven/refuted/Unknown explanations re-evaluate those artifacts. |
| User forgery | FakePlainData/FakeRelocatable/FakeClosedWorld/FakeFiniteDomain can prove local claims only. All transported FactAuthority lists are empty; real PlainData<Resource> stays Disproven. Registry empty. |
| Finite research | Payload enum finite cases Proven; its complete values Unknown. bool/payload-free enum cardinalities checked. Non-enum case assertion has NotVariantType. Ordered CaseName/Tag/PayloadCount stable for 100 runs/artifacts; no Standard admission or match rewrite. |
| Address research | Immovable relocation Disproven RequiresStableStorage; ordinary movable, Drop and owned remain Unknown. Immovable requirement Proven with typed evidence; absent marker Unknown. Existing allocation-owner move versus fixed pointee behavior retained. |
| Contextual closure | Real Worker activation fixture under Research.First and Research.Second has topology ID automata-e10a7267e29bc8b0 and three members each. Ordered inventories and typed boundary-coverage refutations stable for 100 runs. ID omits module boundary by design; no graph seal or runtime change claimed. |
| Geometry safety | Overflow of legal per-dimension extents cannot wrap into a proof. Existing recursive-by-value rejection CV4129 pinned. Valid 24-level repeated-field DAG computes size 67108864/alignment 4 in 50 local query accesses. Depth 32/access 4096 limit with checked arithmetic; no global cache/mutex. |
| Renderer bounds | UnknownDescribe subject/result signatures validated. A 4097-byte literal description is rejected with VERDICT_DESCRIPTION_LIMIT nested in the normal CONCEPT_ASSERT_UNKNOWN proof, not rendered unboundedly. |
| Determinism | 100 identical PlainData artifacts, typed evidence/refutations, explain JSON and refuted diagnostics; 100 finite inventories/artifacts, address outcomes and activation inventories/coverage refutations. CLI 100 source-free artifacts, generated C Normal/Verify and explain JSON for all three outcomes. |

## Authority and integration repairs

All algorithms remain ordinary Concept. New compiler observations project
existing checked geometry or enum declarations; none declares semantic truth.
The one Verdict projector, proof graph, artifact loader and comptime evaluator
remain authoritative. PlainData's structural helper is bounded(16), with shared
4096 fuel/depth 32. Evidence is two byte quantities; no large structural tree.

Routine integration fixes allow ordinary comptime calls in template bodies and
lexically known generic calls under predicate constraints while preserving
operation-requirement precedence; enum/handle assertion subjects preserve their
type category. Native companion checking/building resolves ordinary source
library dependencies through the existing module roots/artifact builder. Only
this dependency topology is qualified, not arbitrary mixed cycles. No parser,
linter, proof engine, optimizer or broad frontend feature is introduced.

No existing semantic path is replaced. Thus no duplicate production authority
needs retirement. Positive geometry shadow agrees for all 11 positive subjects;
native sizeof/alignment independently pins Header. Existing R9b2 ownership
shadow remains unchanged and its corpus agreement is rerun. CAbiValue/AsBytes
continues to govern its separate ABI contract. Existing activation closure and
finite dispatch are audited rather than wrapped in false new authority.

## Costs and performance

Measured on this Windows host, informational wall times rather than CI limits.
Baseline measurements use the exact archived SHA with TinyXML2 upstream restored;
final measurements use the same focused race instrumentation. There is no attempt
to infer regression percentages from single noisy concurrent samples.

PlainData Header: fuel **183**, maximum depth **4**, field-loop bound **2**.
Selected verdict metadata **1517 bytes**. Full assertion artifact increment
**4934 bytes** also includes AST/MIR/proof metadata, not just its typed payload.
Provider artifact **144882 bytes**; subject fixture artifact **119720 bytes**;
explain JSON **2427 bytes**. 100 artifact/explain/diagnostic runs took **2.197s**
in the non-race local sample; the focused race sample took **14.290s**.
These are bounded summaries, not proof trees.

Final library-scale and mutex measurements are recorded below. The innate mutex is retained; PlainData is ordinary evaluation and adds
no new shared lock. Query-local geometry memoization is necessary to bound repeated
type graph expansion, not a cross-module proof cache.

## Regression gates

Baseline full Go: PASS (413.241s); vet PASS; focused race/corpus/innate/artifact
PASS (61.735s). Standard 42, DragonGod 23, Golden 130 tests pass in each Normal
and Verify mode. GPU-free Vulkan is covered by the full suite's existing native
integration. Baseline corpus: 433 valid / 351 static-invalid / 15 runtime-negative;
semantic vocabulary 7 valid / 14 invalid. Current Verdict is Verdict<E,R> with
Proven(E), Disproven(R), Unknown (no payload); trusted registry empty. Existing
selected payload/body transport remains the artifact authority boundary.

Final gate results are recorded below. Current corpus:
**436 valid / 357 static-invalid / 15 runtime-negative**, semantic vocabulary
**10 valid / 20 invalid**. New valid examples cover scalar/record/array and finite
case/domain distinction. Six invalid examples pin three typed refutations and
three Unknown diagnostics. Research address/closure probes stay in testdata;
no admitted relocation/pin/global closure examples are fabricated.

All new standalone Concept and changed consumer sources use multiline Allman
format; CLI format --check and lint cover the provider, research modules, positive
corpus and consumer libraries. Intentional negative corpus files pass format and
are separately checked for their named diagnostics, not required to lint clean.
Embedded authored fixtures are multiline through the normal formatter; inherited
R9b2 fixtures retain their original source. Checked outputs are tested unchanged;
no golden update is used to turn a failure green.

Both frozen Zig suites are **SKIPPED**: no legacy compiler/build/test path changed.
No real GPU execution, macOS/Linux execution, general pinning, complete payload
value enumeration, globally sealed activation inventory or bitwise relocation
is claimed. Stage-0 readiness is updated without automatically freezing Stage-0.


## Final evidence log index

Ignored local evidence under artifacts/r9b3 is reproducible with the commands
below. Logs are not checked in as opaque generated output.

- baseline-go-complete.log: exact baseline Go suite, native TinyXML2 dependency present.
- baseline-vet.log, baseline-race.log and baseline-{Standard,DragonGod,Golden}-{Normal,Verify}.log.
- motivating-baseline.log: original compiler accepts/generates opaque Upload.
- final-go-frozen.log: go test ./... -count=1, PASS, internal/concept 384.279s.
- final-vet-qualified.log: go vet ./..., PASS.
- final-race-qualified.log: focused -race for R9b, innate/library scale, corpus, checked outputs and activation layout.
- final-{Standard,DragonGod,Golden}-{Normal,Verify}.log: serial concept test library gates.
- final-vulkan-qualified.log: GPU-free Vulkan library/profile native Normal/Verify integration.
- format-lint.log: 17 tracked Concept files format-check, eligible positive/provider/consumer lint.
- cli-determinism.log: 100 CLI artifact/C Normal/C Verify/explain runs, PASS, 17.040s.
- cli-explain-{positive,refuted,unknown}.txt: source-free human explanations.
- case-observations.log: 100 names/tags/payload-count observations and invalid-index rejection.
- unknown-bounds.log, zero-nesting.log: bounded rendering and nested zero-size negatives.

The CLI dependency roots contain only provider/subject .concept-module.json
artifacts; their source files live outside those roots. The generated-C gate
uses the production emitter; strict native execution is separately established
by the PlainData and artifact-only C11 harnesses. A CLI emit is not itself a
claim of execution.


## Comparable library and mutex measurements

Both samples use the same race-enabled scale harness; wall times are single
host samples, not statistical speedup claims. Artifact totals use matching
relative module roots and are byte-exact for these checked module fixtures.

| Group | Baseline / final modules | Innate evaluations baseline / final | Validation wall baseline / final | Artifact bytes baseline / final | Artifact build wall baseline / final |
| --- | --- | --- | --- | --- | --- |
| Standard | 26 / 27 | 966 / 973 | 1.071s / 1.066s | 5909950 / 6162938 | 2.136s / 1.966s |
| DragonGod | 22 / 22 | 237 / 237 | 0.920s / 0.778s | 2140248 / 2140248 | 1.286s / 1.040s |
| Golden | 31 / 31 | 593 / 593 | 0.933s / 0.827s | 5838621 / 5838621 | 1.676s / 1.517s |
| Vulkan | 1 / 1 | 194 / 194 | 0.017s / 0.141s | 1154629 / 1243587 | 0.253s / 0.284s |

Standard grows **252988 bytes**: the new vocabulary module (144882) plus AMD64's
ordinary checked imported representation/assertion material (108106). Vulkan's
checked artifact grows **88958 bytes**. These whole artifacts include transported
helper bodies/type/AST/MIR/proof metadata, not only selected evidence. DragonGod
and Golden sizes are unchanged. Largest Standard artifact: 2157797 -> 2265903.
The finite research subject artifact is 25927 bytes; it is test-only.

Repeated aggregate workload: baseline 320 modules/7960 innate evaluations,
final 324/7988 (one extra provider module per repetition). With one worker,
wall 12.025s -> 11.443s, lock wait sum 5.073ms -> 0ms, locked execution sum
146.478ms -> 133.921ms. With four workers, wall 3.926s -> 3.594s, wait sum
12.624ms -> 4.502ms, execution sum 189.755ms -> 162.923ms. This supplies no
contention evidence requiring lock removal or global proof caching. Maximum
innate fuel/depth/loop across groups remains 430/8/8; Standard remains 25/3/0.
These innate counters are separate from the ordinary PlainData Header 183/4/2.
Vulkan's extra validation work is expected from checking the imported restriction;
no optimization is justified from a single timing sample.

Existing typed ownership shadow: **228 field decisions / 4 refutations**, identical
truth, site and text. It is retained evidence, not a new R9b3 rule migration.


## Final gates

| Gate | Result |
| --- | --- |
| go test ./... -count=1 | PASS, internal/concept 384.279s, profile/vulkan 0.366s |
| go vet ./... | PASS |
| focused go test -race ./internal/concept -count=1 | PASS, 58.082s; R9b/innate/scale/corpus/checked outputs/activation tests, no race |
| semantic corpus + diagnostic pins | PASS, 436 valid / 357 static-invalid / 15 runtime-negative |
| GPU-free Vulkan native companion/library/profile | PASS, 18.053s; Normal/Verify paths and real restrictive transfer diagnostics |
| source-free C11 harnesses and geometry shadow | PASS |
| 100-run typed/artifact/explain/diagnostic/order checks | PASS; CLI sample 17.040s |
| format --check / positive lint | PASS; intentional negatives retain named diagnostics |
| frozen Zig compiler/unit and EVT1 suites | SKIPPED, unchanged legacy paths |

Serial CLI closeout: Standard **42/42**, DragonGod **23/23**, Golden **130/130**
passed in Normal/Verify respectively, zero failures.
The complete full Go pass also covers bridge schema/codec/native parity and
existing machine/native/LIR gates. None is inferred from cross-compilation.


Reproduction commands (from the repository root):

```powershell
go test ./... -count=1
go vet ./...
go test -race ./internal/concept -run 'Test(R9b|R9aInnateLibraryScale|R9aComptimeUsageAndLimitDiagnostics|Innate|CV4653|CReprIsTheInnate|SemanticCorpus|EVT1CheckedOutputsMatch|EVT2x4Activation)' -count=1 -v
go test ./internal/concept -run 'Test(VulkanLibraryNormalAndVerify|VulkanProfileExamplesNormalAndVerify|R9b3CAbiAndVulkanBoundaries)' -count=1 -v
go build -o artifacts/r9b3/concept.exe ./cmd/concept
# For each Standard, DragonGod, Golden, run serially:
artifacts/r9b3/concept.exe test libraries/Standard --verbose
artifacts/r9b3/concept.exe test libraries/Standard --verbose --verify
```


## Closeout

All requested baseline/final gates passed within their named execution scopes.
The worktree is clean after the implementation and documentation commits (final
SHA in the user report). Seven required documentation files are updated. Stage-0
is not automatically frozen; EVT2 is not resumed. Deferred public vocabulary has
no hidden execution/fact guarantee beyond the scoped research stated in the ledger.

**SUCCESS — R9b3 deferred semantic vocabulary qualified.**
