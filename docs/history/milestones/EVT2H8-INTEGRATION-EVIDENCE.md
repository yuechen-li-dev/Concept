# EVT2h8 integration evidence

Status: **MILESTONE VERIFIED**, ordinary milestone scope. Full phase qualification
remains EVT2h9 and was not run here.

## Revision and authority

Initial Quick passed before implementation (Go component 17.091 s, CLI 3.046 s);
no full baseline requalification was inferred from the historical checkpoint.

Baseline: `b39088beb9631ea4a449c65797bbc76fde9a265a`, initially clean on
`codex/implement-native-ordinary-enums`, worktree
`C:/Users/yuech/.codex/worktrees/evt2h4-profile-baseline/Concept`.
Fetched origin/main `065317d262c85ce159219561b0709526e7973c1f` was an ancestor
13 commits behind. No reset, replay, branch removal or remote mutation.
Compiler: `concept-evt1-stage0-go`; target: Windows AMD64/Microsoft x64.

Inventory: `df3339f`; Concept physical projection correction: `1ff7c7b`;
checked CFG/record-field projection: `8ca3d9b`; authoritative fixtures and
external verification: `928b4e0`; same-sized identity negatives: `5ef60a6`;
complete entry-use correction and unused-scalar regression: `bfb2e3c`. The final evidence-only closeout commit
contains this record; its exact hash is reported in the mission closeout.

Go continues to own the existing semantic checker, checked LIR production,
transport consistency and orchestration. Concept owns independent artifact
admission, active-field copying, physical projection, allocation, frame planning,
Win64 realization and encoding. No new sum layout, copier, allocator, frame
planner, ABI classifier, runtime registry or production Go native algorithm.

CMIRAMD3–10 schemas, generated codecs, validators and historical wire fixtures
are unchanged. All integration artifacts use existing CMIRAMD10, magic
`CMIRAMDA`, numeric version 10, 44-byte header, schema SHA-256
`ea57ae74f5ae54ece90ce429edbc4a41ce309d106582c84fdb5d70d15242941d`.
The native implementation change is outside the frozen transport.

## Programs and observables

Authoritative sources are `tests/runtime/Testing/IntegrationParser.concept`,
`IntegrationController.concept` and `IntegrationAssertions.concept`. Five facts
in the three corresponding `.concept_test` files join Quick in both modes.
These source-to-C facts are separate from the native execution evidence below.

* Parser: an eight-word input record, End/Add/Scale/Adjust payload tokens,
  checked operand/length/opcode failures, multiple Result propagation levels,
  Option correction lookup, token matching, a bounded while loop and a local
  typed handler. A copyable ParseFault record crosses Result calls. Structured
  diagnostics materialize and match a separate payload-bearing ParseError enum.
  This is the qualified equivalent of Result<Token,ParseError>; putting tagged
  enums inside Result remains explicitly unsupported. ReadBounded factors common
  bounds validation and exercises six declared arguments plus a hidden result.
* Controller: ordinary states and payload events, Option hints, copied mutable
  sessions, nested Result calls, propagated configuration errors, locally handled
  device errors and three-event loops. ConfigError and DeviceError have the same
  physical geometry and remain distinct nominal types. Reports expose state,
  total, sequence, denial/error code, original-copy isolation and write count.
* Checked semantics: WordJudge produces typed WordEvidence{capacity=8}; existing
  proof tooling retains selected evidence, source sites and provenance with no
  trusted fact authority. A valid Assert.Concept erases before LIR. The harness
  removes only that statement from source, preserves residual source positions,
  parses/checks afresh and requires no remaining assertion obligation. Runtime
  LIR shapes agree after ignoring source-plan/decision metadata, and actual
  native images agree byte-for-byte. The two artifacts differ by 36 bytes of
  checked-source metadata. A disproven assertion and a forged erasure marker
  reject before native artifact production. There is no runtime Verdict carrier.

Each native host interleaves 16 valid/malformed protocol vectors 40 times,
40 controller scenarios/values 40 times, both asserted/plain runtime forms,
logical truth tables, guarded selection, while termination and checked local
references before pure conditional expressions. Expected values and side effects
are checked independently; repeated calls expose stale stack/active storage.

Native-only hosts link the C-bootstrap implementation of Cathedral and their
C orchestration, **without domain C oracle code**. They independently admit
source-free artifacts, emit actual AMD64, map it executable and invoke it.
Separate differential hosts link freshly generated domain C11 code. Every
observable report field and read/write/trace count is compared. Each domain C
module remains a separate translation unit; scalar adapters avoid including two
headers defining the same built-in Option C type in one translation unit. This
is harness isolation, not a new source representation or compiler change.
Compilation uses `-std=c11 -Wall -Wextra -pedantic-errors -O2`.

Eight isolated trap children across the four hosts verify both required RHS
failures: real UD2 within the emitted image, prior trace 12, and Windows status
`0xC000001D`. The vectored observer checks the real fault before terminating the
child with that exact status, avoiding WER dialogs. Skipped RHS failures do not
trap. Each completed host has a distinct executable path; no running binary is
replaced. Successful/handled paths have independent C11 parity; C panic/abort
text is not claimed to equal native UD2.

## Fixed integration defects and escalation

1. Source-valid and/or and while lacked native checked projection. They now use
   existing bool slots, branches, joins and loop backedges. RHS evaluation occurs
   only when required, including calls, bang and locally handled question paths.
2. Previously evaluated operands before a nested if/match/logical CFG need the
   same logical snapshots already used before fallible expressions. The shared
   source sequencing predicate covers those forms for LIR and C bootstrap.
   Existing checked local-reference witnesses survive pure conditionals; local
   addresses before actual failure exits retain their explicit unsupported fence.
3. Ordinary enum fields in records were blocked by h1's container/address fence
   and scalar-only aggregate field geometry verification. Composite projection
   now consumes the independently verified nominal enum size/alignment and permits
   its checked internal field address. Existing public reference/array fences and
   all subsequent field/type/borrow checks remain. v9/v10 retain enum aliases.
4. A real native miscompile was isolated: input add(7), seed 2 consumed three words
   but returned total 2 instead of 9. Decode and token construction retained 7.
   Disassembly showed two aggregate entry carriers both assigned R11 before
   unpacking. Aggregate parameters have entry-only physical uses absent from
   source scalar operands. Concept's existing object projection now adds common
   entry liveness observations for every aggregate parameter, including a single
   aggregate beside an unused scalar. The appended UnusedScalarAfterEvent probe
   verifies that case with native C callers and the independent C11 domain oracle.
   These pseudo uses emit no instructions. The existing allocator then keeps
   their locations distinct. The native tests assert overlapping carrier live
   ranges and distinct registers as well as independent C-caller results.

The fourth change triggered targeted shared-call regression escalation: g5
aggregate smoke, malformed plans/artifacts and MSVC/clang-cl ABI observers;
h3 conditional matching; h4 Option and h5 Result native/C11/ABI families; h6
residual erasure; h7 local handling, traps, historical promotions and admission.
This is expanded focused testing, not a project-wide qualification claim.

The initial larger decoder hit the existing 64-object-slot capacity. Factoring
ReadBounded and sharing the evaluator call kept the useful parser below that
bound. No capacity, stack-probe or validation limit was raised. The existing native
model retains 64 object slots and 128 scalar carriers per function and the
frame < 4096 stack-probe fence. The parser input remains eight words with a
four-iteration command bound. Existing scalar
instruction selection, fieldwise active copying and frame planning are reused.

## Determinism, bytes and limits

Three fresh pipelines per checked form per mode are appropriate here: this is
composition through frozen transport and an existing physical path, with one
bounded liveness correction additionally covered by old ABI/native families.
Each pipeline freshly parses/checks the authoritative source, generates LIR/CFG
and MachineIR, writes its own artifact file, admits/decodes that file in Concept,
plans every function and emits/fixes up an image. Canonical LIR, artifacts,
MachineIR/debug descriptions, complete derived plan fingerprints, code and
image/fixup records agree. Normal and Verify agree exactly. Differential hosts
add a fresh pipeline per form. No prior physical plan or checked result is reused.
The phase marker promotes the new gates to 100; historical 100-trial tests are
unchanged. This milestone does not claim the h8 100-trial phase gate ran.

| Form | Functions / calls | Artifact bytes | Native bytes | Frame bytes / spills |
| --- | ---: | ---: | ---: | ---: |
| Parser | 11 / 12 | 143357 | 61259 | 120..1128 / 0 |
| Controller | 24 / 27 | 158422 | 55330 | 40..632 / 0 |
| Asserted runtime | 2 / 1 | 12690 | 3381 | 72..152 / 0 |
| Assertion removed from source | 2 / 1 | 12654 | 3381 | 72..152 / 0 |

Native SHA-256, pinned in the native test:

```text
Parser      a7b2383995d4d64d7b6ec1d33935865580806319f7743fda6657ec1e788bff35
Controller  e23ab5e5ffdb59bc3421f1e02c4184743816ad7746207450371c45d14aa23887
Assertions  951e78d24abe10bc965982415e74f70b6bccabdcbba940c44a38d27c2978dbde
Plain       951e78d24abe10bc965982415e74f70b6bccabdcbba940c44a38d27c2978dbde
```

Artifact SHA-256, pinned in the checked test:

```text
Parser      a22f7c6e20cd6c15210dd4929be40da9da1285d304aa4a91eed70e57f00a1951
Controller  7100448fb789e1234041110111352374dd0550c97bd2cc57bdac1eb2fba9c73e
Assertions  3be840ce69436a865e69c16d3fba81da5e73356904c2ef12fcb4301e457ca84f
Plain       2679883df393fa6f612b99aaa2d6cd0a8a0b9696519277de6ec8d7203ceb3dcb
```

The original 23-function Controller prefix remains 52892 bytes with SHA-256
`045a90da6389f2881769582a3c109ba108556812de5d4d79964454168d4348bf`,
independently pinned alongside the complete 24-function image. The added function
is an explicit entry-liveness regression, not a replacement for the controller.

The source invalid fixtures require CV4116 (closed Option mismatch), CV4028
(non-bool logical operands), CV4545 (nominal handler mismatch). Five mutated LIR
cases reject inactive binding, missing failure authority, bad successor/input and
uninitialized logical joins. Twenty-seven mutated artifacts reject independently
in Go and Concept: inactive reads, undefined inputs, uninitialized results,
handler identity/type/join, missing failure/pattern/tagged correspondence,
nominal identity, same-sized Option/Result constructor and record-enum field
swaps, copy eligibility, payload/enum-field/frame geometry, capacity,
header/version/hash/truncation/trailing corruption. Concept roundtrip, code and
image sentinels remain unchanged on every failure in Normal and Verify.
Observed diagnostics retain CMIRAMD10_IDENTITY/MALFORMED/CAPACITY_EXCEEDED,
PATTERN_CONTRACT and FALLIBLE_CONTROL; no verifier is weakened.

## Executed commands and measurements

Local logs are under ignored `artifacts/evt2h8/`. Results below are uncached test
invocations (`-count=1`); immutable tool/object builds are reused, not test results.
Commands use the repository root. Component timings overlap and must not be summed
as an end-to-end wall time.

```powershell
./test.ps1
./test.ps1 -Lane Runtime -Filter Integration

go test ./internal/concept -run '^TestEVT2h8' -count=1 -v -timeout 8m
# Final whole-image pins and the additional unused-scalar C11 comparison:
go test ./internal/concept -run '^TestEVT2h8NativeIntegration$' -count=1 -v -timeout 8m
# Additional same-sized identity negatives were also admitted independently:
go test ./internal/concept -run '^TestEVT2h8(MalformedIntegration|ArtifactAdmission)$' -count=1 -v -timeout 5m

go test ./internal/concept -run '^Test(EVT2h[3457](Checked.*|Transport|FreshDeterminism|Determinism|Source.*|ConstructorContracts|LIRAdmission|JoinDefiniteInitialization|RejectCrossArmEntry|GeneratedContract|NativeBoundaries|MalformedContracts)|EVT2h6(VerdictErasure|ArtifactVerdictsAreDescriptive|ForgedMarkerDoesNotEraseCall)|EVT2h8(CheckedIntegration|FreshIntegration|ProofAuthority|SourceDiagnostics|LIRMutations|MalformedIntegration)|R9a2Bridge(FrozenPayloadParity|GeneratedOutputs)|EVT2g3MemoryEncodingGoldens)$' -count=1 -v -timeout 5m

go test ./internal/concept -run '^Test(EVT2g5(NativeSmoke|ExternalABI|MalformedPlans|MalformedArtifacts)|EVT2h3ConditionalNativeExecution|EVT2h4NativeOption|EVT2h5NativeResult|EVT2h6ResidualNativeExecution|EVT2h7(NativeFallible|HistoricalPromotions|ArtifactAdmission)|EVT2h8ArtifactAdmission)$' -count=1 -v -parallel=4 -timeout 15m

go test ./cmd/concept -run '^(TestAMD64CLIBatch|TestNativeCLICasesUseBatch)$' -count=1 -v -timeout 10m

go test -race ./internal/concept -run '^Test(EVT2h8(CheckedIntegration|FreshIntegration|ProofAuthority|SourceDiagnostics|LIRMutations|MalformedIntegration)|EVT2h7LIRAdmission|RuntimeWorkflowPolicy)$' -count=1 -timeout 4m

go vet ./...
go build ./...
go build -o artifacts/evt2h8/concept-final.exe ./cmd/concept
```

Formatting checked all nine new Concept sources/facts and AMD64.Objects;
CLI lint passed the three implementation modules and three fact files.
Go formatting and `git diff --check` passed. Frozen bridge/schema/codec/validator
paths have no diff from baseline; selected generated and exact-byte tests pass.

| Check | Measured result |
| --- | --- |
| Quick | Final contended run 25.13 s script / 25.46 s process wall; earlier 23.14 s process wall; both modes, five additional facts |
| Focused Integration facts, standalone Normal | 4.49 s; 5 facts |
| h8 source/LIR/CFG/artifact generation | 3 fresh four-form sets in 0.254 s |
| h8 complete focused suite | 153.946 s; 8 families, no skips |
| Native/C11 batch within h8 suite | 146.116 s; final whole-image/unused-scalar-oracle rerun 147.544 s (147.942 s package); one immutable Cathedral build per mode for all forms |
| Checked adjacent/frozen selection | 101.631 s; 36 families, 151 including subtests; no skips |
| Expanded native/ABI selection | Final 401.606 s (earlier 408.420 s); 12 families, 32 including subtests; no skips |
| CLI batch/policy | Final 309.392 s (earlier 299.987 s); all 20 sources with one backend build, including three new sources |
| Focused race | Final 6.638 s; pass |

Representative initial module emission was 588..693 ms for Parser/Controller
and 11..16 ms for each asserted/plain module in the contended focused run.
Separate earlier planning-only measurements for three complete plans were
108 ms Parser, 114 ms Controller and 2..3 ms for each proof residual. The final
fresh-pipeline timer includes independent admission/decoding/planning/emission:
about 2.5..3.0 s per three Parser or Controller trials and 69..71 ms per residual.
These are local measurements, not portable timing thresholds.

The new harness shares immutable Cathedral/object prerequisites within the
invocation; no per-source cold backend CLI launches were added. Five language
facts stay in the fast lane without importing the whole Cathedral. Cold backend compilation remains a substantial setup cost. No exhaustive gate,
independent oracle or existing test count was removed to improve this timing.

Not run: full Qualification lane, h8 phase100 gate, full Go/race regression,
complete corpus/Golden, Standard/DragonGod, Vulkan and every historical ABI
family. These remain h9 responsibilities, not passes inferred from baseline.
Frozen Zig suites are skipped under the unchanged legacy-infrastructure policy.

## Remaining boundaries and h9 readiness

* Local fixed arrays remain a checked projection limitation (A); qualified arrays
  inside records work. The parser uses that real input representation.
* Checked local references preceding pure conditional CFG now work. Local address
  temporaries preceding actual failure exits remain fenced; extending their
  failure-edge lifetime/dominance evidence is separate B/C work.
* Nested v10 error remaps remain B/C: existing admission requires a straight-line
  function-failure transfer/return. Older qualified v9 remaps remain intact.
* Nested sums require separate layout/copy/active-authority qualification (E).
  No general nested Result/Option payload support is claimed.
* Reference-bearing/owning/Drop/move-only payloads and cleanup are D and deferred.
  No foreign enum ABI contract, optimizer pass, COFF output or self-hosting work.

The bounded integration surface is ready for h9 comprehensive qualification.
This work stops at h8; it does not automatically start h9 or reclassify these
boundaries as qualified.
