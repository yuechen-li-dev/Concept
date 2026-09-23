# R7g1 conformance status

**Result: meaningful progression.** Generated functions and required-operation witnesses run through the normal compiler path. Artifact-only derivation and artifact-only consumption work in focused tests. Standard.Collection's generated `TraceReferences` passes native collector fixtures, including the same cycles-and-diamonds graph as the handwritten witness (10 Standard tests pass).

The complete R7g1 goal is open. The current generator filters fields by explicit annotation and expands ordinary statements. It does not yet classify fields through general trace concepts, recurse through nested composites, or generate array/table loops or payload-enum matches. The generated `TraceReferences` passes an ordinary `NoAllocation` assertion in the Standard fixture. The concept proof query handles a single nominal type argument and a generated required operation. Runtime reflection remains deferred.

The syntax, phase, provenance, artifact model, supported field query, and limits are specified in [EVT1-GENERATED-DECLARATIONS.md](../design/EVT1-GENERATED-DECLARATIONS.md). The convergence log records the concrete blockers removed and the remaining boundary.

## Evidence

- Baseline: `09fdc48107139152aa6179aa28eb3ec4f6e062fb` (R7g), compiler `concept-evt1-stage0-go`, artifact schema `concept-module.v1`.
- Native Standard build/test: 10 passed, including generated/manual cycles-and-diamonds parity and ordinary generated `NoAllocation` proof.
- Native DragonGod build/test: 21 passed, 1 benchmark.
- BurnIn, `go vet ./...`, root and legacy Zig test gates pass.
- Artifact-only A/B/C Trace chain and `Trace<Node>` generated concept proof pass; an unmarked imported type fails with `REFLECT_PERMISSION_REQUIRED`.
- A 100-run directed test compares generated identity, structured view, proof, artifact, MIR, C, and header bytes.
- Informational local sample: compiling the derived Trace module took about 48 ms; its artifact was 471,021 bytes and the consumer's generated C was 10,538 bytes. These are whole-module figures, not isolated generator cost.
- The frozen EVT1 manifest remains 395 valid, 256 static-invalid, 13 runtime-negative, 5 compatibility, and 4 expected-divergence fixtures.
