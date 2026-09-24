# R7j convergence log

Baseline: clean `e9a204356430422944f91559c869d6bdcb467cf7` (R7i SHA `e9a2043`), compiler `concept-evt1-stage0-go`.
Qualified host: Windows AMD64, Go 1.27.0, Zig 0.16.0, Clang 22.1.3, GCC 15.2.0. The `oct` executable is absent from PATH; the `Make.oct` Go command targets were run directly for BurnIn, Standard, and DragonGod.

| Finding | Change | Proof |
| --- | --- | --- |
| All aggregate extern C returns were rejected | Admit recursively safe `[[repr(C)]]` record values for return and parameter | TinyXML2 native aggregate return and by-value round trip pass |
| `record struct` had no explicit foreign representation claim | Add `[[repr(C)]]` validation and `CAbiLayout` compiler fact for ordinary propositions | `concept explain CAbiValue(ConceptXmlStats)` is Proven with size/align detail |
| Probe accepted only two `int` fields and compared only manifest with native | Compare Concept geometry, manifest, and native compiler for arbitrary claimed field types/counts | Pair, nested Outer, array Packet, and padded Padded probes pass; wrong offset and packed header fail |
| The historical C11 emitter uses `f()` for zero-argument functions | Keep the aggregate qualification on explicit-parameter functions so checked output remains stable | `-std=c11 -pedantic-errors -Wall -Wextra` aggregate fixture passes; global prototype cleanup remains separate |
| Probe evidence lived only in console/error results | Write deterministic `.native-build/abi.json` and list probes in build plan | Repeated probe bytes match; header edit changes input hash |
| A repr(C) declaration could be omitted from the manifest | Require a native ABI claim for every repr(C) companion record | Missing-claim regression fails before runtime |
| The first 100-run ABI test shared the TinyXML2 output path with another validation process | Give the determinism test a private temporary native project | Concurrent runs no longer overwrite each other's `abi.json` |

Deferred: semantic-module transport of measured ABI evidence, artifact-only native identity validation, explicit alignment overrides, packed layout, fixed-underlying enum ABI, pointer fields with nullability/ownership contract, unions, bitfields, and flexible arrays. No C++ parser, runtime ABI registry, or platform calling-convention classifier was added.

The semantic corpus adds two valid and six static-invalid ABI/layout cases. The 100-run ABI evidence test repeats the external probe and compares the Concept-side JSON bytes. The existing R7i 100-run test covers build plan, semantic companion artifact, generated C, and proof serialization.

Validation on the qualified host: final `go test ./...` passed (internal/concept 219.888s), as did `go vet ./...`, both `zig build test` roots, direct Go BurnIn (`go test ./internal/concept -run R7d3 -count=1`), Standard build/test, DragonGod build/test, TinyXML2 check/test, Clang and GCC strict-C11 aggregate round trips, and the ABI/layout corpus. `oct make BurnIn --file Make.oct` could not start because `oct` is not on PATH; the exact Go target from `Make.oct` passed.
