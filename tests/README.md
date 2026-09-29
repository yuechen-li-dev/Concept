# Concept tests outside the compiler package

The Go compiler's own tests live in `internal/concept`. This directory holds
larger inputs those tests read:

| Directory | Contents | Read by |
| --- | --- | --- |
| `dogfood/tinyxml2` | TinyXML2 native-project dogfood (upstream is a submodule) | `native_project_test.go`, `native_abi_chain_test.go`, and others |
| `goldens/companion` | C++ companion golden: Concept module, native bridge, proofs, tests | `native_companion_golden_test.go` |
| `goldens/perf-baselines` | hand-written C11 hot-path shapes for comparing generated code | manual review; not run by any test |
| `interop/octagon` | Oct-side `.octest` programs for the Octagon codec work | run with Oct; not referenced by the Go tests |
| `fixtures/octagon` | `.octagon` byte fixtures (LF-pinned in `.gitattributes`) | `codec_octagon_interop_test.go` |
| `verify` | bounds-heavy Verify-mode benchmark | manual: `go run ./cmd/concept test tests/verify --verify`; not run by any test |

Language conformance fixtures are in `language/evt1/`. The retired PoC3 test
layout (lexer/parser/run scaffolding and `corpus/`) is preserved in
`legacy/poc3-zig/tests/`.
