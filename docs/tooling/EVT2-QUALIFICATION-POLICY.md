# EVT2 testing and qualification policy

Effective **2026-10-31**. This policy supersedes earlier instructions requiring
full project-wide qualification for every ordinary EVT2 milestone. Historical
milestone evidence remains valid for the exact revisions and scopes tested.
Only scheduling changes; test cases, scripts and verification logic remain intact.

## Development and completion

| Stage | Required evidence |
| --- | --- |
| Default development | `./test.ps1`, plus focused tests for changed components, negative cases and relevant regression boundaries |
| Ordinary milestone completion | Focused native execution, independent C11 differential checks, Normal/Verify parity, malformed-input tests, selected adjacent regressions and a small justified fresh determinism sample |
| Major checkpoint | All milestone checks plus complete qualification: 100 fresh determinism trials, all independent ABI oracles, full Go regression, required race checks and vet, corpus/Golden, Standard/DragonGod, GPU-free Vulkan, frozen artifacts, formatting/lint and policy checks |

The next planned major qualification checkpoint is **EVT2h phase closeout,
tentatively EVT2h9**. Ordinary feature completion does not depend on running the
exhaustive project-wide workflow. A focused selection must still exercise the
real changed path; C-bootstrap facts or internal verifier agreement alone do
not establish native execution.

Select tests by the changed boundary and its dependent behavior. Reuse immutable
Cathedral builds and batch compatible CLI cases under the existing workflow;
retain independent artifacts, oracles, processes and executable paths. Keep
language-level facts in Concept and external compiler/native probes in their
owning harnesses.

For milestone determinism, state the sample count and why it fits the change;
each sampled generation/emission must be fresh. `go test -count=1` means one
test invocation, not one fresh generation inside a test. Use an existing focused
sample path where available. A selected test with an embedded 100-run gate still
runs that gate unchanged; this policy does not authorize reducing its loop.
Do not use `-short` as native proof: it deliberately skips some native tests.

## Immediate risk escalation

Expand testing immediately when behavior changes affect shared ABI, allocator,
frame planner, encoder, transport compatibility or other high-risk
infrastructure. Exercise the affected invariants, negative/sentinel cases,
independent oracles and dependent native regressions. Expand to the complete
checkpoint workflow when compatibility or blast radius cannot be covered by a
justified focused selection. Record the trigger and selected coverage; do not
defer necessary checks until EVT2h9.

## Major checkpoint workflow

Start with `./test.ps1 -Lane Qualification` (optionally `-ProfileDirectory`),
then run the companion race, library Normal/Verify, ABI, formatting/lint,
generation/frozen-artifact and policy gates required by the checkpoint. The
script currently runs full Go/native tests and vet; invoking it alone is not a
claim that every companion gate ran. Use [TESTING.md](TESTING.md) and the
[workflow record](TEST-WORKFLOW-2026-10-29.md) for existing entrypoints and reuse.

Preserve all existing cases, qualification scripts, independent oracles and full
100-trial requirements. Do not delete coverage, change goldens to force a pass,
disable checks, convert failures to warnings or replay cached results as fresh
evidence. Existing documented frozen-Zig and prerequisite policies still apply;
report skips explicitly.

## Evidence status and reporting

| Status | Meaning at the reported revision |
| --- | --- |
| IMPLEMENTED | Code is present; required milestone verification is not yet complete |
| MILESTONE VERIFIED | The declared subset passes focused milestone checks, including the native/oracle evidence applicable to the change |
| FULLY QUALIFIED | The complete major-checkpoint workflow passes for the declared scope |

Report the exact revision, scope, commands, determinism sample counts,
pass/fail/skip outcomes, relevant timings, risk escalations and remaining
boundaries. Distinguish native execution, C-bootstrap semantics, artifact
validation and external ABI evidence. List expensive gates deferred to the
major checkpoint rather than marking them passed. A fully qualified baseline
does not make subsequent incremental changes fully qualified.
