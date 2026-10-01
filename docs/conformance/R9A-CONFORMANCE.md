# R9a conformance

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
