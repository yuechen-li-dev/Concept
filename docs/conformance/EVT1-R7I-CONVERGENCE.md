# R7i convergence log

Baseline: clean `e1a7ea2331bb69f3657cb249dfb92be11afefc64`, compiler
`concept-evt1-stage0-go`. Existing R7a package manifests use immutable
`comptime PackageManifest`; R7h supplies checked generated declarations and
schema concepts. Existing EVT1 has `extern "C"`, `foreign concept` with
`DeclaredForeign`, `concept-module.v1`, `.concept_test`, strict C11 generation,
`concept explain`, and Make.oct Standard/DragonGod/BurnIn targets.
The checked structural reflection and declaration sequence is `09fdc48`,
`6b84540`, `1e96be2`; R7h1 begins its documented codec boundary at
`6ba377f` and continues through `c608918` and `4986028`. R7h2 proceeds
through `60ded73`, `4024e5f`, `b200bf6`, `65aa4e6`, `0645ac6`,
`421b93e`, and baseline `e1a7ea2`.

| Area | Encountered issue | General fix or boundary |
| --- | --- | --- |
| Build graph | No native target model in active R7a package driver | Added typed native project extraction, command plan, external Clang/GCC-style adapter, and independent hashes. |
| Tool invocation | Native paths and defines need safe argument handling | Uses `exec.Command` argument arrays; test builds a C target with spaces in paths. |
| Dependency identity | Bridge header was initially omitted from input hash | Hash headers under both include roots and source directories; regression changes a header and checks identity. |
| Test harness | `.concept_test` linker accepted only generated C files | Added explicit native link inputs and C++ linker selection to existing test runner. |
| Windows linking | `-lm` became missing `m.lib` under clang++ | Native C++ link path omits that POSIX-only argument. |
| ABI declaration | Current EVT1 parser does not accept old `[Repr(C)]` syntax; direct struct return reports `EXTERN_C_ABI_TYPE_INVALID` | Uses current `record struct` syntax and compiler-run layout probe; direct aggregate call remains open. |
| ABI mismatch | A false companion size could otherwise pass | Wrong-size probe regression reports `NATIVE_ABI_MISMATCH` with both values. |
| Foreign proof | An imported opaque effect could be treated optimistically | `concept explain` keeps creation `DeclaredForeign` and getter effect Unknown. |
| Test parity | Upstream test needs resource files as current directory | Run original test from `upstream/`; 528 pass, 0 fail. |
| Artifact transport | `concept explain` expected built module artifacts | Native project explain builds companion source into in-memory semantic artifacts, as test discovery already does. |
| Bootstrap gates | `oct make BurnIn --file Make.oct` reports that `oct` is not recognized | Ran the exact Go BurnIn target and the Standard/DragonGod package build/test commands declared by `Make.oct`; the Oct entry point remains unqualified on this host. |

Concept is not a C++ parser or compiler. No C++ grammar, object model, runtime
reflection registry, or second semantic authority was added. The bounded next
blocker is aggregate C ABI validation and lowering, beyond the scalar bridge.
