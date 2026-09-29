# R7j conformance status

**Meaningful progression.** The aggregate C ABI path works for the motivating TinyXML2 boundary and for focused Clang/GCC fixtures. Full R7j success is not claimed.

| Requirement | Evidence | Status |
| --- | --- | --- |
| Explicit C representation and ABI admission | `[[repr(C)]] record struct`; recursive `CAbiLayout` and `CAbiValue` proof | Pass for supported values |
| Aggregate return and parameter | TinyXML2 `.concept_test`; strict-C11 Pair/Outer/Packet/Padded round trip | Pass |
| Nested, fixed array, padding | Strict-C11 execution plus native size/align/offset probes | Pass |
| Wrong layout and packed mismatch | Native probe rejects wrong claim or packed header | Pass |
| Concept layout operations | `SizeOf`, `AlignOf`, typed `OffsetOf<T>(T.field)` static assertions | Pass for supported shapes |
| Native evidence provenance and target scoping | `.native-build/abi.json` with origin, compiler, target and input hashes | Pass as standalone artifact |
| Artifact-only aggregate declaration | Consumer imports `Native` semantic artifact, proves `CAbiValue`, and generates aggregate extern prototype without companion source | Pass |
| TinyXML2 schema | `NativeStatsSchema` requires `CAbiValue` and typed accessor; `concept check` separately measures layout | Pass in native project path |
| Native evidence inside semantic module artifacts | Measured facts not transported into `concept-module.v1` | Open |
| Toolchains | Clang and GCC strict-C11 call execution; Clang and GCC C++ layout probes | Pass on this host |
| Semantic corpus | Two valid and six static-invalid ABI/layout fixtures in `language/evt1/abi-layout` | Pass |
| All generated FFI under pedantic strict C11 | Aggregate fixture passes; existing zero-argument `f()` prototypes emit a Clang strict-prototypes warning under `-pedantic-errors` | Open |
| Unsupported ABI forms | Packed, align override, union, bitfield, flexible array, pointer field, enum ABI not admitted | Deferred |

The next blocker is attaching measured evidence to a semantic module artifact with a verified native-input identity, then requiring that identity when an artifact-only consumer builds or tests. Current native project commands remain safe because they probe again before use.
