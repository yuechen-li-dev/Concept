# EVT1 R7m conformance (meaningful progression)

Baseline: `2490fdef4c533dbf8299ce6930845e320c703012`, compiler
`concept-evt1-stage0-go`. The preceding R7l2 commits are `a657336` and
`5d97467`; baseline HEAD is the subsequent `2490fde` machine-artifact change.

The explicit Verify policy is visible in `concept plan --verify` and reachable
through `emit-c`, `mir`, and `.concept_test` execution. A native strict-C11
test uses the same Concept source in Normal and Verify, checks equal valid
output, checks a Verify-only bounds report for `index=4 extent=4`, and checks
generated output determinism. Normal C has no Verify helper. The existing
conservative synchronization policy is reused. `Assert.Concept` is unchanged.
The `.concept_test` runner classifies this observed violation as
`verification`. A direct Normal/Verify comparison of MMIO and AMD64 machine
fixtures finds identical generated C and machine helper artifacts.

The Standard Verify run passed 29 tests. The DragonGod Verify run passed 23
tests and one benchmark. These runs exercise the explicit mode, but their
existing fixtures do not yet contain new allocator, collector, or scheduler
audit checks.
TinyXML2's ordinary native companion run passed 2 tests; no foreign Verify
checker was added. `go vet ./...`, `zig build test`, focused Verify and corpus
Go tests passed. Two full `go test ./...` attempts did not complete: the first
was interrupted after more than ten minutes without a package result beyond
`cmd/concept`; the second was interrupted after more than four minutes with
the same symptom. Neither is recorded as a passing full gate.

This is not R7m success. Poison on Drop, allocator red zones, ownership-state
instrumentation, declared/foreign contract observers, collector and DragonGod
Verify audit checks, TinyXML2 foreign verification, and native hardware trace
equivalence remain unimplemented or unverified. No runtime observation is promoted to a
static fact.
