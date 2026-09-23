# R7g1 conformance status

**Result: success.** Generated functions and concept required-operation witnesses run through ordinary binding, semantic checking, MIR, and strict C11. Artifact-only derivation and artifact-only consumption pass. Standard.Collection's generated `TraceReferences` matches its handwritten witness on the same cycles-and-diamonds graph.

`TraceEdge<E,T>` selects direct handle fields and payload fields through ordinary concept satisfaction. An ordinary `TraceNested<F,T>` required-operation witness delegates nested composites. `[[trace_array]]` selects fixed arrays and table columns for ordinary iteration. `DeriveEnumTrace` generates exhaustive payload-enum matches. Native generated witnesses prove `NoAllocation` where their calls permit it. The collector remains independent of reflection and continues to accept handwritten Trace implementations.

The bounded generator API emits one closed function declaration per derive request. It intentionally does not synthesize structs, enums, open generic declarations, or recursive generation waves. Production and test runtime reflection remain deferred. The concept proof query currently accepts a single nominal type argument; generated declarations themselves can close over other reflected types through the normal type parser.

The syntax, phase, provenance, artifact model, supported field query, and limits are specified in [EVT1-GENERATED-DECLARATIONS.md](../design/EVT1-GENERATED-DECLARATIONS.md). The convergence log records the concrete blockers removed and the remaining boundary.

## Evidence

- Baseline: `09fdc48107139152aa6179aa28eb3ec4f6e062fb` (R7g), compiler `concept-evt1-stage0-go`, artifact schema `concept-module.v1`.
- Native Standard build/test: 14 passed, including generated/manual cycles-and-diamonds parity, enum payloads, fixed arrays, table columns, nested delegation, and ordinary generated `NoAllocation` proofs.
- Native DragonGod build/test: 21 passed, 1 benchmark.
- Full `go test ./...` (including the EVT1 semantic corpus), BurnIn, `go vet ./...`, and root and legacy Zig test gates pass.
- Artifact-only A/B/C Trace chain and `Trace<Node>` generated concept proof pass; an unmarked imported type fails with `REFLECT_PERMISSION_REQUIRED`.
- Two 100-run directed tests compare reflected metadata, generated identities, structured views, proofs, artifacts, MIR, C, and headers byte for byte; one uses the Standard.Collection Trace generator and artifact-only inputs.
- An input-field rename changes generated identity and artifact bytes in the incremental invalidation test.
- Informational local sample: compiling the derived Trace module took about 44 ms; its artifact was 471,051 bytes and the consumer's generated C was 10,538 bytes. These are whole-module figures, not isolated generator cost.
- The frozen EVT1 manifest remains 395 valid, 256 static-invalid, 13 runtime-negative, 5 compatibility, and 4 expected-divergence fixtures.
