# EVT2h convergence

Current EVT2h4 status: **SUCCESS — bounded native Option and full qualification**.
Earlier sections retain historical outcomes and boundaries.

## EVT2h4 native Option qualification

Started from clean synchronized main
`408db1720fdf72381069cbc8f5a5feaa6633816a`, not from an assumed historical branch.
Inventory `609494d` follows inspection of source Option, generic identity,
proof/failure facts, C lowering, h2/h3 LIR/transport/admission and native ABI.
Compiler `concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64.

Implementation `10be4cb` adds only checked semantic production and fixtures.
Existing intrinsic Some tag0 / None tag1 and contextual constructor resolution
feed the same tagged and match paths. Structured closed GenericApplication
identity replaces the ordinary enum declaration key for Option. CMIRAMD8/9
already express every required fact; all frozen v3–9 contracts remain unchanged.
Concept's existing admission, guarded reads, active-field copier, slots, scalar
allocator, frame, ABI and encoder execute the applications. No new Option
native algorithm, generic engine, bridge or layout/copy engine was needed.

Focused native/C11 Normal/Verify and both independent MSVC/clang-cl caller/callee
probes pass. The source-free native host runs before linking the domain oracle.
The 34-function image is 45,901 bytes, 22 calls, frames 40..408, four XMM spills;
SHA256 `5d5b8746bd74e74b50d40f8cc7dbe5e680597c6c00ba732d33be33fa8bc90316`.
The canonical v9 artifact is 117,108 bytes, SHA256
`410976620e998aac75eb2e79e1cc0949c01cb05f595516bb3f4fc6b7c1794f88`.
All 100 fresh source/transport/plan/emission gates retain full counts. Six LIR,
29 artifact, nine source-invalid and six native-unsupported cases reject. The
eight-explicit-argument hidden-return boundary is independently rejected by
Cathedral physical preflight with unchanged output/image sentinels.

Iteration exposed two existing source/ABI limits without changing them: record
fields are immutable (use an ordinary replacement value for copied bindings),
and a hidden result consumes one of the existing eight physical argument slots.
The positive mixed-call specimen uses seven explicit parameters; the rejected
nine-position form remains a permanent negative probe. None-at-page-boundary
and invalid-tag child probes retain unique executable paths on Windows.

The mandatory throughput work is separately identifiable: observer `679bc86`,
exact include-closure reuse / isolated test scheduling `3c5b6eb`, and profiled
full workflow plus scalar scheduling `5ba39d8`. Baseline measurement at h3 is
2,208.19 s uncached Go, with a separately measured 0.62 s vet stage. The
[throughput record](../../tooling/EVT2H4-QUALIFICATION-THROUGHPUT.md) distinguishes
observed wall time, process counts, immutable build reuse and actual fresh gates.
No validation count or independent oracle is removed.

Final implementation is `10be4cb`; canonical negative specimen formatting/LF
follow-up is `a5588b9`. Formatting changes no executable positive source or
pinned bytes and is rechecked by focused diagnostics and race admission. Final
closeout changes evidence records only. Detailed local logs are under ignored
`artifacts/evt2h4/`; the profiling command and summarizer are committed.

| Executed gate | Outcome |
| --- | --- |
| Final default `./test.ps1` | PASS, 18.21 s; 30 facts per mode, no whole-Cathedral build |
| Full `./test.ps1 -Lane Qualification -ProfileDirectory artifacts/evt2h4/after` | PASS, 1,319.57 s including vet/report; uncached Go 1,317.301 s |
| Compiler/native / CLI / GPU-free Vulkan packages | PASS, 1,310.377 / 143.971 / 0.443 s |
| Full top-level families | 935 passed / 10 prerequisite-or-role skips, zero failures |
| Supplemental eight legacy strict C11 specimens in MSVC developer environment | PASS, 1.470 s package; clears eight full-lane environment skips |
| Focused race: h4, cache invalidation/reuse, runtime/backend facts and workflow policy | PASS, 481.521 s package / 494.193 s wall; full native counts retained |
| Standard CLI Normal / Verify | 44 passed plus three benchmarks each, 41.099 / 41.602 s |
| DragonGod CLI Normal / Verify | 23 passed plus one benchmark each, 5.711 / 5.721 s |
| Fresh h4 source/LIR/CFG/MachineIR/debug + codecs, 100 | PASS, 1.550 s in full run |
| H4 source-free valid round trips / malformed artifacts | PASS, 100 per mode / 29 cases; all sentinels; 22.88 s family |
| H4 native-only followed by strict C11/native oracle, both modes | PASS, 84.84 s family; exact bits, protected None, trap and both DLL toolchains |
| H4 physical plans / emissions | PASS, 100 complete sets per mode; plans 3.559 / 3.677 s; emission samples 318 / 318 ms |
| Frozen v3–9, h1–h3/g5, corpus, Golden, generated outputs, checked outputs, package, policy | PASS within the full lane; no frozen bridge file changes |
| Windows trap executable regression | PASS, TestExplicitCastCore 2.46 s; distinct paths retained |
| Go build, gofmt, new Concept fixture/specimen format, lint, generated inspection, diff | PASS; generated inspection correctly has no new public generated declarations |

The two remaining ordinary-run skips are TestCollectorHundredRunDeterminism
(unrelated R7fR opt-in burn-in) and TestEVT2h3ArtifactProcessWorker (subprocess-only
worker; the existing driver separately exercises its 100-process gate). Root
and legacy Zig suites are skipped because their infrastructure is unchanged.
No required h4 test is skipped. TinyXML and adjacent Oct checks execute in main;
they were absent from the isolated baseline checkout's prerequisites.

Exact focused race command:

```powershell
go test -race ./internal/concept -run '^(TestEVT2h4.*|TestNativeFixtureHeaderClosure.*|TestRuntimeWorkflowPolicy|TestRuntimeConceptSpecifications|TestBackendConceptSpecifications)$' -count=1 -parallel=4 -timeout=15m -v
```

Library commands are `concept test libraries/Standard --module-root libraries
--verbose` and the corresponding DragonGod command, each repeated with
`--verify`. `go build ./...` and changed-Go `gofmt -l` pass. The new authoritative
Option fixture and fact file pass CLI format/lint; all sixteen negative and
native-boundary specimens pass canonical formatting and diagnostics recheck.

Observed throughput improves **2,208.808 -> 1,319.57 s**, **1.674x / 40.26%**,
with 13 more passed families and additional Option oracles. Shared semantic/C
builds remain two; backend C object cold builds fall 43 -> 12, all observed
reusable C objects 46 -> 17, and nativeCommand launches 1,976 -> 1,965. The named
oracle/parity compilation subset grows 16 -> 18. All old 100-run gates remain;
new h4 gates add fresh source, plans and emissions. Peak memory is unmeasured.
The sub-20-minute stretch goal is missed; remaining h3 native and independent
oracle/validator execution costs are measured, not declared inevitable.

The baseline observer patch/logs are preserved before restoring its checkout
clean. The app prevents archiving that attached checkout because it designates
it as this task's primary checkout. Closeout mirrors qualified main there as a
clean detached snapshot, avoiding a stale h3 starting point. No additional task
branch or remote changes are created, and unrelated worktrees remain intact.

**SUCCESS — EVT2h4 native Option and throughput qualification complete.**

The [conformance record](../../conformance/EVT2H-CONFORMANCE.md#evt2h4--closed-native-option-applications)
states the exact tested domain and unsupported boundaries. The two new Concept
facts are C-bootstrap evidence; independent emitted code establishes native
execution. No Result/Verdict/Drop/ownership/COFF or EVT2h5 work is begun.

## EVT2h3 native match qualification

Actual baseline: clean synchronized main
`745b1ec35679743bed555a89e19611c6dea4712a`, compiler
`concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64.
Inventory `ee80418`; checked CFG/CMIRAMD9 admission `6aa55c2`;
native realization/fixtures `9f50f82`; arm-dominance correction `2b93c3c`.
The additional `1737006` conditional nesting probe adds tests/fixtures only.
The final full lane runs at `1058c86`, including deterministic initialization
of the existing private bootstrap semantic wire types and all probes.
Closeout changes these records and pins h3 specimen line endings only.
No unrelated edits or historical worktree files were reset.

[Conformance](../../conformance/EVT2H-CONFORMANCE.md#evt2h3--native-checked-match)
defines semantics, authority and bounds. Existing enum matches are exhaustive,
without enum wildcards. Integer matches have a final wildcard. Existing
subjectless guards become ordered if chains; variant-attached guards are not
source syntax. Payload binding copies the full checked field list into
arm-local values. No new source pattern language is introduced.

CMIRAMD9 schema:
`77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c`.
Added typed test/require sites and flow associations reuse canonical v8/v7
domains, object CFG, operands, borrow scopes and checked payload operations.
No duplicate layout facts or physical MATCH instruction. Concept independently
validates selection, field bindings, initialized joins and arm-body dominance;
atomic v8 runtime guards remain on the same private snapshot. Cathedral reuses
its recursive logical copier, object storage, allocator, unified frame planner,
Win64 aggregate machinery and encoder. Go retains semantic production,
generated transport, consistency verification and external oracles.

| Specimen | Canonical artifact bytes / SHA256 | Native bytes / SHA256 |
| --- | --- | --- |
| OrdinaryMatches: 7 functions, 5 matches, 8 typed tests | 21,265 / `b3f04bb7352fec2209349bb26c55cd837158e50e32c2c656651d8bb41248c0ed` | 3,772 / `48dee1c8a55f462854fe930e64fb34d3eefac77d0c18a2d6c6cdfdc4ca40e83b` |
| TaggedMatches: 35 functions, 9 matches, 54 typed tests, 81 field reads | 163,395 / `e68f23900d6a28cc14c81209b97bc561ddfef0b0c2dd53dd150ca0fcb6d4e560` | 122,509 / `398445c4b5e1bf1ff09c190f897843fb58985fe96625295e71f8ba6f2628484f` |

The two-function conditional/region specimen additionally emits 1,000 bytes,
SHA256 `1f709df6fccb5d081aeccb59a48072d6902a5a788fb1c10d6022076132415ecf`
in both modes. It tests match inside if and if inside match arms, comparing
typed C constructors with native enum carriers without assuming general C
enum interoperability. Its 100 fresh source/artifact generations and 100
emissions per mode passed in 77.651 s package.

Both native images are identical in Normal and Verify, with one / 38 call
fixups. Ordinary frames: 40..88 bytes. Tagged frames: 40..1,928 bytes, eleven
spills across the module. Artifact-only native execution precedes linking the
domain C oracle. The oracle host then independently compiles strict C11
(`-std=c11 -Wall -Wextra -pedantic-errors -O2`) and executes the same
observables against emitted AMD64.

Native positive coverage includes all ordinary and mixed empty/payload
variants, int/bool/float/double/multiple fields, records and fixed arrays,
scalar/aggregate results, reassignment/copy isolation, nested matches and calls,
range loops, if statements, early return, integer wildcard, single scrutinee
evaluation, guard/call/mutable-log order, irrelevant-variant guard suppression,
result joins and pressure/spills. Exact binary32/64 checks cover signed zero,
adjacent precision, quiet NaN payload and subnormal. Existing h2/g5 independent
external ABI probes remain required regression evidence, rather than a new
general foreign enum ABI claim.

| Gate | Measured outcome |
| --- | --- |
| Default Quick at completed native checkpoint | PASS; 18.95 s |
| Quick after dominance correction | PASS; 24.95 s while full qualification runs; 24 runtime + 4 backend facts per mode |
| Checked source/LIR/CFG/MachineIR/debug + generated codecs, 100 fresh runs | PASS; 5.444 s, pinned artifact hashes above |
| Independent artifact-only Concept roundtrip | PASS; 100 per specimen per mode; initial package 132.110 s |
| Native-only then strict C11/native execution, Normal/Verify | PASS; initial focused package 401.215 s, Normal 198.04 / Verify 203.04 s; repeated successfully in final lane after dominance and bootstrap corrections |
| 100 complete native physical plans per function | PASS; ordinary 0.150 s, tagged 6.476 s Normal / 6.710 s Verify; exact existing ABI/frame/transfer debug equality |
| 100 fresh complete admissions/emissions per native-only host/mode | PASS; exact full code, NativeImage, symbols and fixups; C-oracle host independently repeats one complete plan/emission |
| Individual emission sample | Ordinary 12 ms; tagged 1,213 ms Normal / 1,328 ms Verify, including admission/preflight |
| Native invalid-tag execution | PASS; ordinary/payload child processes require UD2 within generated image |
| Source diagnostics | PASS; 18 invalid source specimens and retained h2 source-only match promoted to verified native input |
| Missing predecessor result | PASS; dedicated LIR must-initialization mutation |
| Malformed artifact admission and output atomicity | PASS after dominance correction; 38 independent Go/Concept cases in both modes, 92.758 s package; roundtrip/code/NativeImage sentinels preserved |
| Focused race after bootstrap correction | PASS; source, CFG/dominance, 38 malformed contracts, 100 fresh subprocesses and source/transport/codecs; 212.768 s package (process gate 151.61 s, codec/source gate 57.315 s) |
| Native CLI batch | PASS; 12 scenarios / 81.007 s package; both h3 sources share the existing single backend batch |
| Standard Normal/Verify after bootstrap correction | PASS; 44 tests and three informational benchmarks each; CLI 53.158 / 51.965 s |
| DragonGod Normal/Verify after bootstrap correction | PASS; 23 tests and one informational benchmark each; CLI 6.559 / 6.616 s |
| Actual package build after bootstrap correction | PASS; DragonGod with Standard dependencies, 17.305 s |
| Generated inspection / fixture format and lint | PASS; BridgeCodec9 declarations, schema/validator/native owners, positive fixtures and facts |
| Frozen schemas/codecs | PASS; direct baseline diff for CMIRAMD3–8 empty; regenerated outputs unchanged |
| 100 fresh compiler subprocesses, alternating cold/prior AST encoding | PASS; 10.18 s before full rerun; exact artifact and dependency identity |
| Semantic artifact / focused enum checks after bootstrap correction | PASS; 21.208 s short package; native probes deliberately skipped here and retained in full lane |
| Quick after bootstrap correction | PASS; 23.74 s |
| Complete Compiler lane after bootstrap correction | PASS; `go test -short ./... -count=1 -parallel=4 -timeout=10m`, compiler package 554.080 s, CLI 3.802 s, GPU-free Vulkan 0.863 s |
| Full qualification at `1058c86` | PASS; compiler/native package 2,264.969 s, CLI 154.470 s, GPU-free Vulkan 0.780 s; lane 2,267.62 s including `go vet ./...` |
| Final codec regeneration and generated inspection | PASS after full lane; no generated Go/Concept codec or schema metadata diff |
| Frozen root / legacy Zig | SKIPPED; legacy compiler/build/test infrastructure unchanged |

Five intentionally parser-invalid fixtures cannot pass `concept format`:
expression block, missing guard default, guard ordering, statement-expression
arm and variant-attached guard. Their unchanged syntax errors are tested;
all syntactically valid negative fixtures are formatted. There is no lint
waiver for valid source. Quick facts run through C bootstrap; only the native
hosts establish AMD64 execution.

One full-lane attempt was stopped during audit when a forged edge entered
another arm's inner if block. Region reachability alone accepted it, even
though checked payload operations retained runtime guards. The
`arm_regions.concept` mutation reproduces that pre-fix failure. Both Go and
Concept now require arm-local instructions to be dominated by the selected
body; independent artifact admission rejects it as CMIRAMD9_BINDING_SCOPE,
preserving all sentinels. The full lane was restarted after `2b93c3c`.
No failing gate was waived. An earlier native-host run also exceeded the
ordinary two-minute child deadline during 100 full large-module emissions.
The h3 host now has a bounded eight-minute deadline; the full package deadline
is 45 minutes. All 100-run gates remain enabled; Quick's scope is unchanged.

Final required command: `./test.ps1 -Lane Qualification` runs
`go test ./... -count=1 -parallel=4 -timeout=45m` and `go vet ./...`.
This includes earlier EVT2h/g native tests, strict C11 and external ABI probes,
corpus, Golden, GPU-free Vulkan, package determinism and authority/workflow
policies. Focused race and actual Standard/DragonGod CLI tests run separately.
Detailed local logs are under ignored `artifacts/evt2h3/`.

The completed full run at `2b93c3c` found exactly one failing gate:
TestEVT2h3Transport, with compiler package 2,088.960 s, CLI 134.581 s and
GPU-free Vulkan 0.667 s. The payload artifact hash changed only in that process;
focused reexecution passed its former pin. A subprocess regression reproduced
identical typed AST JSON with different semantic envelopes after encoding a
different AST first. Gob reserves global wire IDs lazily. Stage-0 is the sole
Gob owner in this compiler; `1058c86` deterministically reserves its closed
semantic type graph before any AST encode/decode. This retains ordinary v2 Gob
reading and full dependency digests, without a parallel codec or native Go
algorithm. The prior focused payload artifact was
`451bb0a2ea632a6d8012d56b045892103e0b219eded5921cec559d79abb33098`;
the corrected canonical artifact is pinned above. Native hashes and frozen
CMIRAMD3–8 contracts are unchanged. The full lane is repeated for this fix;
the failing run is retained, and vet was not claimed from that failed lane.

No Option/Result/Verdict, ownership/Drop, move/reference patterns, nested tagged
payloads, foreign representation expansion, object output or optimizer work is
qualified here. Source/C support for those forms remains distinct. EVT2h4
requires separate review.

**SUCCESS — EVT2h3 native checked match is qualified at `1058c86`.** Final
documentation closeout records the exact evidence above and pins h3 specimen LF
endings; no algorithm changes follow qualification. Earlier failed/interrupted
runs remain recorded. No gate is waived. Planning/emission samples are local
focused-run measurements, not portable performance thresholds. Native code
hashes, scopes/joins, 38 malformed artifacts, 18 source diagnostics, 100 fresh
process/source/artifact/native gates and frozen earlier contracts pass. Root
and legacy Zig alone are skipped because the frozen Zig paths are unchanged;
the subprocess-only worker is intentionally skipped as a standalone test and
is executed by its 100-run owning test. Stop here; EVT2h4 is not started.


## EVT2h0–h1 historical closeout

Baseline: `535910e486b0f20f6ec1c6e6e7bee931d359b61a`, qualified EVT2g5.
Branch: `codex/evt2h1-native-enums`. Compiler: `concept-evt1-stage0-go`.
Target: Windows AMD64 / Microsoft x64.

Inventory: `2292af7`. Implementation/tests:
`1fc1503c12ee5d7899affe75d837e75ba234b282`.
Final checked dependency-provenance correction: `191aacf`.
Exact generated-output LF pinning on Windows: `0b8abf9`.

The baseline stopped ordinary enum LIR at `EVT2_UNSUPPORTED_TYPE`. Checked nominal
domains now reach CMIRAMD7, independent Cathedral admission, the existing native
object/scalar realization and actual executable Win64 calls. The final motivating
specimen executes all seven variants from three types, local copies/assignment,
equality, branches/range loops, early returns, recursion, call pressure, eight
enum parameters and mixed scalar/enum positional calls. Strict C11 source parity
and artifact-only execution pass in Normal and Verify.

The [conformance record](../../conformance/EVT2H-CONFORMANCE.md) inventories source
semantics, explains the minimal versioned extension, names every authority owner
and records exact artifact/native hashes and limitations. CMIRAMD3–6 schemas,
codecs, hashes and specimens have zero content delta from the baseline.

## Qualification ledger

Logs are under ignored `artifacts/evt2h1/`; this committed record retains the
commands and outcomes. All required gates are complete.

| Gate | Outcome |
| --- | --- |
| Focused `go test ./internal/concept -run '^TestEVT2h1' -count=1 -v -timeout=10m` | PASS; 122.024 s |
| Actual CLI LIR/MachineIR/native/frame inspection | PASS; CLI test 30.740 s |
| 100 fresh checked LIR/MachineIR/debug and codec generations | PASS; 2.154 s |
| Native Normal/Verify, 100 fresh emissions per mode | PASS; exact 8440 bytes |
| Strict C11 oracle / independent C caller and callback | PASS; 100 varied inputs per mode |
| Malformed artifact admission | PASS; 23 cases x 100 per mode, output/image unchanged |
| Source semantic negatives | PASS; 12 exact diagnostic cases |
| Native nine-argument stop | PASS; existing scalar-planning failure, unchanged output/image |
| Initial full run at `191aacf` | One failure: raw CMIRAMD6 generated codec had CRLF after restore; compiler sweep 1414.264 s, CLI 300.842 s, Vulkan 0.497 s |
| Corrected frozen `TestEVT2g3bGeneratedAndExactTransport` | PASS; 56.473 s; original v6 hash and exact 784-byte specimen retained |
| Final `go test ./... -count=1 -parallel=4 -timeout=35m` at `0b8abf9` | PASS; compiler 1394.705 s, CLI 297.737 s, GPU-free Vulkan profile 0.435 s; includes frozen native, corpus, checked-output and Golden tests |
| Focused race, `go test -race ./internal/concept -run '^TestEVT2h1' -count=1 -v -timeout=15m` | PASS; 453.555 s |
| Imported-identity Normal / focused race after provenance correction | PASS; race 2.186 s |
| Vet | PASS |
| Standard Normal / Verify | PASS; 44 tests each, 3 informational benchmarks |
| DragonGod Normal / Verify | PASS; 23 tests each, 1 informational benchmark |
| Package graph, generated inspection, format/lint, diff | PASS |
| Frozen root / legacy Zig | SKIPPED; legacy compiler/build/test infrastructure unchanged |

## Corrections during qualification

Ordinary ConstructExpr checking returns its type without populating the special
ResolvedType field; native LIR correctly resolves the checked declaration instead.
The fixture uses the existing native range-loop contract; general WhileStmt LIR
was not added. The nine-argument boundary retains the existing
`AMD64_SCALAR_PLANNING_FAILURE` diagnostic rather than inventing a new capacity
classification. These corrections preserve existing owners and scope.
Review also found that the existing dependency projection was conditional only
on aggregate layouts. It now includes enum domains, retaining the defining
semantic-module identity/digest. A source-free imported/local enum test confirms
distinct identities and exact transport. The full Go run was restarted at
`191aacf` to cover the final implementation; the focused race lane covers the
native implementation plus a separate post-correction imported-identity race
test. The correction does not change the pinned local native specimen bytes.
The full sweep then exposed a Windows checkout issue: the frozen v6 codec
comparison requires exact generated LF bytes, while unpinned files returned as
CRLF after `git restore`. Regeneration restores the original generated bytes;
three `.gitattributes` patterns now pin the v4/v5/v6 generated Go, Concept and JSON
outputs to LF. Their Git content, versions, schema hashes and wire/native
specimens remain unchanged. The failed exact check passed after correction,
and the full suite was repeated at `0b8abf9`; no failing test was waived.

## Remaining boundaries

The v7 module admits scalar and ordinary enum types only. Any aggregate/array or
reference capability in the lowered module is outside this slice. No payload,
match, native built-in fallible type, new ownership/Drop, object output or
optimization pass was implemented. Source C bootstrap acceptance of richer
forms is explicitly separate from native execution. EVT2h2 remains deferred.

**SUCCESS — EVT2h0 semantic inventory and EVT2h1 native ordinary enum execution qualified.**
The work stops here. EVT2h2 has not begun.

## EVT2h2 checked frontend progression

Historical progression at `09b3d50`; the native gap is addressed below.

Actual initial local checkout: clean `main` at `8b129ad`. Fetch exposed newer
`origin/main` at `4e96e5cda4f932231129c057de1c00edbb81e7f8`, merging the
workflow through `bc84b51`, with merge commits `ff5e398`, `d554b27` and `4e96e5c`
after the recorded workflow lineage. Created `codex/evt2h2-native-payload-enums`
from that fetched merge, preserving the local main branch. No reset, unrelated
worktree edit or other-repository change was performed.

Inventory commit `7e3db57` records the existing syntax/checker, C union semantics,
left-to-right constructor evaluation, public geometry rejection and native
transport/copy boundary. Implementation/specimens: `784b34c`.

The first blocker is removed: checked source now generates and verifies nominal
payload LIR, including variant field layouts, complete initialization, local
value copies/replacement, typed calls/returns, branches/loops and defining
artifact identity. Four domains include two distinct 8/4 sum types with identical
physical geometry. Constructor order is checked against retained call result
operands. Three Concept-authored behavior facts execute in both C-bootstrap
modes, including checked matches used only by the oracle.

The next blocker is concrete: MachineIR rejects the saved positive source with
`EVT2H2_TAGGED_TRANSPORT_REQUIRED`. CMIRAMD3–7 cannot encode a sum's active payload
and checked conditional copy/access authority. No new bridge, Cathedral sum
admission, tag-dispatched semantic copy, tagged ABI realization or native image
exists yet. Treating the sum as an ordinary record would permit inactive-field
reads; using its tag as the value would discard the payload. Neither fallback
was introduced. This is meaningful frontend progression, **not EVT2h2 success**.

| Gate | Outcome |
| --- | --- |
| Starting quick lane | PASS; 26.90 s |
| Final implementation quick lane | PASS; 20.48 s, 26 facts per mode plus workflow/CLI checks |
| Closeout quick lane at `09b3d50` | PASS; 19.53 s |
| Focused `go test ./internal/concept -run '^TestEVT2h2' -count=1 -v -timeout=3m` | PASS; 1.831 s |
| 100 fresh source/check/LIR/debug generations | PASS; 606.767 ms; 55,712-byte JSON SHA256 `4e345711e1b1d0f75ad810b8d1fe7673fb7f775e61625b370b0204a4ca90ef06` |
| Malformed LIR | PASS; 22 mutations, including independently rehashed invalid descriptors |
| Invalid source / retained unsupported forms | PASS; 8 source diagnostics and 2 source/C-valid native boundaries |
| Strict C11 layout probes | PASS; GCC and Clang independently check all four sum layouts/fields |
| Exact floating payload bits | PASS; 4 binary32 + 4 binary64 patterns through Normal C bootstrap, strict C11 |
| Source-free semantic artifact | PASS; imported nominal identity and dependency digest retained in LIR |
| Focused race `go test -race ./internal/concept -run '^TestEVT2h2' -count=1 -v -timeout=5m` | PASS; 5.727 s |
| EVT2h1 transport/source boundary focus | PASS; original v7 schema and exact 46,377-byte artifact unchanged; payload boundary deliberately advances from unsupported LIR to unsupported MachineIR |
| Workflow and architecture authority policy focus | PASS; `TestRuntimeWorkflowPolicy` and `TestR9cConditionAuthorityShadowAgreement`, 0.975 s |
| Changed-source formatting / lint | PASS; all 12 new source/specimen files formatted; positive source and Concept facts lint clean |
| Generated inspection | PASS; frozen BridgeCodec7 using built Standard semantic artifacts; the new source has no generated declarations |
| Standard Normal / Verify | PASS; 44 tests each, 3 informational benchmarks |
| DragonGod Normal / Verify | PASS; 23 tests each, 1 informational benchmark |
| Package graph | PASS; `go run ./cmd/concept package build DragonGod`, including Standard |
| Frozen bridge/schema/golden diff | Empty against actual baseline `4e96e5c`, including v7 |
| Initial full workflow at `784b34c` | FAIL: `TestExplicitCastCore` reused `trap.exe` after abnormal termination; GCC linker reported Permission denied. Compiler sweep 1417.133 s; CLI 69.354 s; GPU-free Vulkan profile 0.553 s; no other failure reported |
| Windows trap-probe isolation, `09b3d50` | Each negative specimen now has its own executable path; no sleeps, retries or skipped probes. Three focused runs passed in 8.296 s |
| Repeated full qualification workflow at `09b3d50` | PASS; compiler/native package 1417.092 s, CLI 69.240 s, GPU-free Vulkan profile 0.528 s; total 1419.61 s including vet |
| Native h2 execution, parity, artifact-only admission and 100 emissions | NOT RUN / NOT IMPLEMENTED |
| Native h2 frames, images, plans, ABI and emission measurements | N/A |
| Frozen root / legacy Zig | SKIPPED; legacy compiler/build/test infrastructure unchanged |

Local logs are under ignored `artifacts/evt2h2-*`. The conformance record names
the checked representation, exact limits and diagnostics. Remaining native work
must continue EVT2h2; EVT2h3 match has not begun.

The initial full sweep exposed a qualification-harness file reuse hazard, not
an accepted failing compiler check. `explicit_conversion_test.go` linked all four
aborting cast probes to the same `trap.exe`; Windows denied the third overwrite.
The correction gives each executable a distinct case identity and retains every
strict-C11 compilation, abnormal exit and diagnostic assertion. Full qualification
passed on `09b3d50`; the failed log is preserved as
`artifacts/evt2h2-qualification-initial-permission-failure.log`.

The successful full run retains the established EVT2g5/EVT2h1 native execution,
malformed artifact sentinels, 100-emission and exact-byte gates, corpus, checked
outputs, Golden, package and Vulkan checks. These qualify regression preservation;
they are not h2 native execution evidence. `go vet ./...` passed as the final
workflow stage. No failing test was waived. The native h2 acceptance criteria
remain unmet, and the final milestone status is **PARTIAL**.

## EVT2h2 native gap closure

Baseline `934e5b4859b201354331b1eca0b2fbe286af47fd`, clean `main`, equal to
fetched origin/main. Historical EVT2g5/h1/workflow commits were not replayed.
Work stays on main, with no new task branch or historical worktree mutation.
Compiler `concept-evt1-stage0-go`; Windows AMD64 / Microsoft x64.

| Checkpoint | Commit / result |
| --- | --- |
| Concept-owned CMIRAMD8 and independent admission | `8698781`; generated schema/codecs; no CMIRAMD3–7 changes |
| Native tagged operations and existing aggregate ABI integration | `0da865b`; native construction, active-field copy, guarded projection, calls and returns |
| Completed spill/negative/byte fixture | `4a175e6`; nine floating spills, 29 functions / 18 calls, exact native bytes pinned |
| Local-only preflight correction | `59788a4`; all v8 modules use existing count-only preflight; imported transport follow-up included |
| Source-free imported transport follow-up | Focused race passes; defining semantic identity/digest survives CMIRAMD8 encode/decode |

The [checkpoint record](EVT2H2-CHECKPOINTS.md) retains the initial 23-function
contract specimen and interim 27-function native evidence. The final source has
22 functions plus seven internal guarded observers. Current artifact: 42,064
bytes, SHA256 `1215b1c779093e9fc149929d8ca4ad0b6f81a1a6aa0d77c56341d5e3372125e5`.
Current native image: 31,871 bytes, SHA256
`524c364540dbb5efc8b1634f5d7087f8933fd4225d92afaa73dc39bccd7becc3`,
identical in Normal and Verify; frames 40..1096 bytes. Fresh generation/codec100
took 668.665 ms; one recorded emission 84 ms; 100 complete physical plans per
function 1.699 s. These are local measurements, not performance thresholds.

| Gate | Current outcome |
| --- | --- |
| Default `./test.ps1` after completed fixture | PASS; 24.16 s; 26 Concept facts per mode plus workflow/CLI frontend policies |
| Default quick including local-only source regression | PASS; 24.14 s |
| Source/LIR/transport/debug and generated codecs, 100 fresh runs | PASS; final artifact/hash above |
| 100 complete Concept physical plans per function | PASS; reused existing canonical ABI/frame/move/transfer debug descriptions |
| 100 fresh native emissions per host | PASS; exact code, symbols and fixups; Normal/Verify identical |
| Artifact-only native execution before source oracle | PASS in both modes |
| Strict C11 source/native differential | PASS; scalar/mixed/record/array payloads, local isolation, variant replacement, calls, loops, recursion, evaluation order and floating spill return |
| Independent Microsoft x64 ABI probes | PASS; separate MSVC and clang-cl C DLLs, both calling directions, direct/indirect values and hidden/stack argument shifts |
| Binary32/binary64 exact bits | PASS; signed zero, adjacent precision, NaN payload and subnormal through native construction/copy/guarded read |
| Malformed artifact and native sentinels | PASS; 28 cases, independent Go/Concept rejection; output and NativeImage unchanged |
| Checked active-variant execution guards | PASS; three child processes, including two protected-payload-page probes; require UD2 rather than access violation |
| Derived tagged ABI/frame corruption and tiny output | PASS; argument identity, return class, overlap and preflight capacity reject |
| Module with tagged locals and no aggregate ABI signatures | PASS; checked source, Normal/Verify execution and tiny-output/image atomicity; 70.666 s focused package |
| Existing source / malformed LIR boundaries | PASS; eight invalid source cases, 22 LIR mutations; nested sum and native match remain unsupported |
| Focused final semantic/transport race | PASS; 15.247 s; 100-run semantic/transport/codec checks retained |
| Imported artifact transport follow-up race | PASS; 2.053 s |
| Windows cast-trap executable isolation | PASS; three fresh `TestExplicitCastCore` runs, 9.245 s; unique executable per case retained |
| Frozen EVT2h1 | PASS focused; original 46,377-byte artifact and 8,440-byte native hashes unchanged |
| Native CLI | PASS focused before spill fixture; ten scenarios share one cold backend, plus existing single-input smoke; final expanded fixture covered by full run |
| Standard Normal / Verify | PASS; 44 tests each, three informational benchmarks |
| DragonGod Normal / Verify | PASS; 23 tests each, one informational benchmark |
| Package graph | PASS; actual `concept package build DragonGod`, including Standard |
| Changed source/specimen format / lint | PASS; schema, codec, validator, native owners and Concept facts |
| Generated inspection | PASS for derived `BridgeCodec8` declarations using built semantic artifacts; schema itself has no generated declarations to inspect |
| Full `./test.ps1 -Lane Qualification` at `4a175e6` | PASS; compiler/native 1569.289 s, CLI 74.861 s, GPU-free Vulkan profile 0.506 s; lane 1572 s including vet |
| Final full lane at `59788a4` | PASS; compiler/native 1557.665 s, CLI 81.251 s, GPU-free Vulkan profile 0.480 s; lane 1564.97 s including vet |
| Frozen root / legacy Zig | SKIPPED; legacy compiler/build/test infrastructure unchanged |

One qualification attempt was deliberately interrupted after coverage review
found that the mixed-call fixture did not force floating spills. No failing
test was waived. Adding Seven/FloatingPressure proves nine spills through the
existing allocator, then the complete lane was restarted on `4a175e6`.
Earlier iteration failures narrowed the standalone-array return boundary; the
array observer now returns a checked ArrayBox record, retaining the existing
ABI restriction. No union-specific layout/copy engine, frame planner or Go
native algorithm was introduced. Local detailed logs are under ignored
`artifacts/evt2h2-close/`.

The final audit also found that a v8 module containing only local sums could
select the older object emitter without count-only emission preflight. That
legacy path advertises scratch output until success, which is insufficient for
v8 atomic capacity failures. `59788a4` routes all v8 modules through the existing
aggregate preflight and adds an authoritative local-only Concept specimen. The
regression requires a scalar-only signature classification and checks native
execution plus unchanged tiny-output/image sentinels in both modes. Older
versions retain their routes. The full lane was repeated for this correction;
the earlier passing run is retained rather than claimed as final-code evidence.

The [conformance record](../../conformance/EVT2H-CONFORMANCE.md#evt2h2--native-tagged-values)
defines source/transport/native authority, caps, unsupported forms and exact
diagnostics. Source `.concept_test` matching still runs through C bootstrap;
the separately emitted native image establishes h2 execution. Native match,
fallible wrappers, ownership/Drop, object output and optimization remain outside
this closeout. EVT2h3 requires separate review.

**SUCCESS — native EVT2h2 qualified at `59788a4`.** The final full lane includes
all frozen native, corpus, Golden, Vulkan and package regressions. The subsequent
closeout commit changes these records only. Native bytes remain pinned above;
no failing gate is waived, no later milestone is started, and all implementation,
specimens and evidence are committed on main. The worktree is clean at closeout.
