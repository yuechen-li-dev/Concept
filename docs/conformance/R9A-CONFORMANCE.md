# R9a conformance

Historical R9a evidence. The remaining MachineIR bridge item was closed by R9a2; see [R9a2 convergence](R9A2-CONVERGENCE.md). The live bridge now derives both codecs from one checked schema.

Baseline: 9004990cd2019dcef6aa1fe91c7ee07ecc0e75a7, clean merged main.
Compiler: concept-evt1-stage0-go; Go 1.27.0 Windows/AMD64; Zig 0.16.0.
Baseline innate identity: innate-32c77687e1e9673d; normalized SHA256
32c77687e1e9673d6a787057626aab167dd9737a8188c8f14b9835c4f981c8be.
Module envelope: concept-module.v1; prior closed generics lost application metadata.
MachineIR bridge: CMIRAMD1. Native EVT2d and bounded push/pop EVT2x6 are present.
Baseline full Go passes in 319.842s; vet and both Zig suites pass.
Standard Normal/Verify: 42 each; DragonGod: 23 each; Golden: 130 each.
No handoff environment-only failure reproduced.

Directed R9a tests qualify structural int/double/Double/nested/value/alias identity,
opaque display-symbol inference/equality, old/mismatched artifact rejection, artifact-only
inference/generation, strict C11, and exactly 100 repeated artifact builds. Context tests
cover nested array arguments/aggregates/payloads, foreach, nested match, and indices;
runtime-dependent comptime calls reject CV4210. Context inheritance/restoration is pinned.
Thirteen admission cases compare check, generation, and artifact diagnostics. Generic
repr(C) Pair<int> executes through an artifact-only consumer; resource pairs reject.
Existing generated field dispatch, generic ownership, innate and semantic corpus tests
remain required. Final gates and migration measurements are recorded in R9A-CONVERGENCE.

## Final qualification

Qualified code head: 59465422023918b76d13f83d482f2d5c4ab95936. Compiler ID remains
concept-evt1-stage0-go. Semantic envelope is now concept-module.v2 with required
concept-generic-application.v1 metadata. Final normalized innate SHA256 is
b8fa1458c50f6499970361276b8245215b7bb2d963049635985314cb33208707;
identity innate-b8fa1458c50f6499. The closeout documentation commit follows this head.

| Gate | Result | Evidence |
| --- | --- | --- |
| Full Go, repaired source | PASS, internal/concept 332.265s | artifacts/r9a/final-go-repaired.log |
| go vet ./... | PASS | final-vet.log |
| Root / legacy Zig | PASS / PASS | final-zig-root.log / final-zig-legacy.log |
| Standard Normal / Verify | 42 / 42 passed | final-Standard-*.log |
| DragonGod Normal / Verify | 23 / 23 passed | final-DragonGod-*.log |
| Golden Normal / Verify | 130 / 130 passed | final-Golden-*.log |
| Vulkan GPU-free Normal / Verify | 12 / 12 passed | final-Vulkan-*.log |
| Innate scale + diagnostic race lane | PASS, 16.246s | final-innate-race.log |
| Touched valid source format/lint | PASS, seven fixtures and backend | final-format-touched.log / final-lint-touched.log |
| CLI check/emit static validity | 20 static negatives, identical diagnostics | final-cli-admission.log |
| Exact registered wide-index runtime specimen | Expected bounds panic, Normal / Verify | final-runtime-index.log |

The runtime specimen calls Read with 4294967297u against a two-element array.
Both actual C11 executables abort with the expected bounds diagnostic; Verify
prints the full index=4294967297 and extent=2. The Windows abort status is
3221226505, not a successful exit. Directed unsigned tests additionally execute
array/span/string/tensor/ndarray probes at 4294967297 and UINT64_MAX.

The full Go gate includes semantic corpus 426 valid / 337 static-invalid /
15 runtime-negative specimens, checked-output parity, artifact-only codecs,
EVT2 LIR/MachineIR/AMD64 and EVT2x3/5/6 native/C traces. Separate artifact-native
static-control tests cover Normal/Verify and exactly 100 artifact/C/MIR builds;
structural identity has its own exactly 100 artifact builds. The caller workspace
harness compares compatibility, exact-demand and larger-budget encoding bytes
100 times per native fixture and checks CapacityExceeded before output writes.
GPU hardware execution is optional and was not claimed.

The first final Go run was RED in 329.525s with six failures. Repairs normalized
table-column declaration provenance out of generic type arguments (without
mutating table field semantics), taught scheduler C harnesses structural symbols,
updated the deliberately unsupported schema test after the v2 bump, and updated
the repr diagnostic expectation after generic record admission. All six targeted
regressions passed in 17.137s before the repaired full run. These were change-caused
failures, not host failures or retries to green. No goldens were regenerated.

Touched Go formatting is clean after line-ending normalization; tracked main.go
CRLF was restored to keep its baseline diff one usage-description line. Existing
value_shaped_concept.concept formatting drift was observed in an exploratory
whole-directory check and left outside the touched-file format qualification.
The old amd64.go formatting issue remains untouched. Final diff check passes
with the tracked CRLF convention; final worktree is clean after closeout commit.
