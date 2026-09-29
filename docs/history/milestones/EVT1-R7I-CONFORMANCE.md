# R7i conformance status

State: **meaningful progression**. Baseline Concept SHA:
`e1a7ea2331bb69f3657cb249dfb92be11afefc64`;
compiler ID: `concept-evt1-stage0-go`.

| Requirement | Evidence | Status |
| --- | --- | --- |
| Real C++ project without implementation rewrite | TinyXML2 submodule at `8224e427`; unchanged `tinyxml2.cpp` and `xmltest.cpp` | Pass |
| Structured direct native build | `concept plan/build` compile original `xmltest` and bridge archive with Clang 22.1.3 | Pass |
| Native test parity | upstream `xmltest`: 528 pass, 0 fail; Concept runtime fixture: pass | Pass for overlapping parse case |
| Companion and foreign origin | `Native.concept`; `concept explain` shows `DeclaredForeign` on allocation | Pass |
| Conservative unknown | `concept explain` shows Unknown for `ConceptXmlChildCount` effect | Pass |
| ABI/layout probe | Clang reports `ConceptXmlStats` size 8, alignment 4, offsets 0 and 4; wrong-size regression is rejected | Pass for selected type |
| Schema concept | `NativeStatsSchema<ConceptXmlStats>` Proven; missing accessor Disproven | Pass |
| Native `.concept_test` link and call | two facts pass via Clang-linked C++ archive | Pass |
| Concept-side determinism | 100 byte-identical plans, companion artifacts, generated C harnesses, and proof JSON outputs | Pass |
| Direct aggregate C ABI calls | `EXTERN_C_ABI_TYPE_INVALID` for `ConceptXmlStats` return | Open |
| Full C++ toolchains and shared libraries | Clang/GCC-style executable/static subset only | Open |
| Incremental caching | conservative source/header identity and full rebuild | Deferred |

Host gates: `go test ./...`, `go vet ./...`, both Zig suites, and the Go BurnIn target
(`go test ./internal/concept -run R7d3 -count=1`) pass. The active package
driver builds Standard and DragonGod; Standard has 29 passing tests and
DragonGod has 21 passing tests plus one benchmark. The `oct make ...` entry
point could not run because `oct` is not on this host's PATH; the commands
declared by `Make.oct` were run through the Go driver instead.

The additive fixture has one Concept manifest, one companion module, one
`.concept_test` file, and a two-file C bridge. It keeps upstream `CMakeLists.txt`
and `xmltest.cpp`. `concept plan`, `check`, `build`, and `test` are four explicit
CLI actions; this is workflow evidence, not a claimed complexity metric.

This milestone does not claim full R7i success. The next general compiler
boundary is direct, validated aggregate C ABI representation. Native struct
schema/layout checks work, but current tests call scalar bridge functions.
See the convergence log for exact diagnostics and gate results.
