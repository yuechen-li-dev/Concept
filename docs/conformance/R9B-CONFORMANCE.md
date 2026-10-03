# R9b partial conformance

This report qualifies the implemented progression slice. It does not qualify
full R9b. Baseline `2d32827cfe2afab05e6436e647aefab67b991eda`;
compiler `concept-evt1-stage0-go`; branch `codex/r9b-semantic-vocabulary`.
Qualified code commit: `c0fba831a89c40a6f335b2c5146e9b42fa187b89`
(Qualify closed comptime evidence values and StaticExtent shadow facts).

## Evidence

Baseline full Go suite: PASS, internal/concept 367.344s. Baseline vet: PASS.
Focused baseline race tests: PASS, 2.337s (declared predicate authority, innate
module compilation and EVT2 LIR Planner/determinism). Full suite includes corpus,
Golden Normal/Verify and GPU-free Vulkan Normal/Verify. Separately recorded
Standard/DragonGod Normal/Verify baseline CLI totals were not collected; final
library totals are recorded below rather than mislabelled as baseline evidence.

R9b focused tests pass: closed nested typed values, artifact-only evaluation and
native C11 consumer, resource rejection, pinned typed predicate boundary, cost,
StaticExtent assertions/mismatch, artifact-only assertion, Planner fixed-shape
shadow/native vector execution, conservative Unknown and no predicate fact grant.

New corpus: 2 valid / 5 static-invalid examples; aggregate corpus becomes
428 valid / 342 static-invalid / 15 runtime-negative. Negative boundaries pin
CONCEPT_ASSERT_DISPROVEN, CONCEPT_ASSERT_UNKNOWN, CV4216 and
PREDICATE_REQUIREMENT_INVALID. No negative verdict is manufactured as success.

100-run checks cover closed typed evidence producer artifact bytes and repeated
extent Parse/Generate output (truth, proof graph and MIR facts). CLI explain
reports Proven, exact array length and DerivedFromLayout origin. Format/lint apply
to individual valid Concept files; negative specimens are format-checked and
validated as expected rejections. Project-directory format/lint require a
manifest.concept; the first directory attempts reported that missing manifest,
so individual-file checks are the relevant evidence.

Checked comptime_tables/Core and Vulkan MIR/map/manifest outputs are regenerated
through CONCEPT_UPDATE_CHECKED_OUTPUTS=1 after the two expected CV3001 failures
from added StaticExtent metadata; generated C/H output is unchanged. The checked
output gate then passes without the update environment variable.

Costs: ordinary versus closed generic evidence parses (100 each) were
30.40ms / 51.32ms in one informational run. Inspect fuel=5, depth=1, loop=0,
array=0. No innate rule or observation changed, no lock removal or cache addition;
large typed-verdict innate workload/mutex impact is not yet qualified.

## Final gates

| Gate | Result |
| --- | --- |
| go test ./... -count=1 | PASS; internal/concept 383.832s; Vulkan profile 0.335s |
| go vet ./... | PASS |
| Race: R9b, predicate authority, innate compilation/scale, EVT2 LIR Planner | PASS, 24.112s |
| Standard Normal / Verify | 42 / 42 passed |
| DragonGod Normal / Verify | 23 / 23 passed |
| Golden Normal / Verify | PASS through TestDomainGoldensNormalAndVerify in the full Go suite |
| GPU-free Vulkan Normal / Verify | PASS through TestVulkanLibraryNormalAndVerify in the full Go suite |
| Semantic corpus | PASS: 428 valid / 342 static-invalid / 15 runtime-negative |
| Focused innate / EVT2 | PASS in full suite and focused race lane |
| Strict C11 | PASS: artifact-only evidence consumer and fixed vector kernel |
| 100-run artifact / truth / MIR proof stability | PASS |
| Final focused R9b / corpus / checked outputs | PASS, 1.411s |
| New Concept valid-file lint / all-file format check | PASS |
| git diff --check | PASS |

The attributes file, manifest and new fixtures pin LF in .gitattributes; this intentionally
normalizes the formerly CRLF corpus manifest while changing its counts. The
initial focused log retains the expected stale-output failures; focused-final.log
records the passing closeout. The full suite preceded only the final additional
cost/negative diagnostic tests; those ran in the final focused lane with vet.
Raw local logs live in ignored artifacts/r9b. Zig suites are SKIPPED because the frozen
legacy compiler and its build/test infrastructure are unchanged, per the current
validation policy. Full typed verdict, typed diagnostic dogfood, new PlainData
and finite-domain semantics and Vulkan/backend/machine adoption remain unqualified.

All changes are local commits; nothing was pushed. The checkout was switched
back to main externally after creation of the research branch. The first code
commit was relocated onto the research branch, with main restored to the recorded
baseline and no other commit altered. The final documentation commit records this
qualification; worktree status was checked clean after committing.
