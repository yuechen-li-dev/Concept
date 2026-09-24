# R7i native build control plane

`manifest.concept` is ordinary immutable Concept data, not a build-script
language. The current native adapter reads typed `NativeProjectManifest` data
through the Concept frontend and emits a deterministic `concept-native-plan.v1`
record. The plan contains one command per compile/link/archive step with
separate argument elements. There are no shell fragments. The selected
external compiler owns C or C++ semantics and the native linker.

The current adapter supports Clang and GCC style drivers, C11/C17 and
C++17/C++20/C++23 modes, executable and static-library outputs. The manifest
enum also names ClangCL and MSVC, which currently fail explicitly as
unsupported adapters. Source files must be under declared roots. Defines are
typed name/value records. The plan hashes manifest content, source files,
headers in include and source directories, link inputs, and compiler version.
This conservatively tracks headers and rebuilds for correctness; it does not
implement a preprocessor or an incremental cache.

`BuildInputHash` identifies inputs and command policy.
`SemanticCompanionHash` identifies companion source.
`NativeOutputHashes` hash the resulting target bytes independently.
External compilers may put paths or nondeterministic metadata in native bytes;
Concept does not claim reproducible native output universally.

`concept plan <project>` is the inspection surface. `concept check <project>`
validates the manifest, companion modules, tests, and compiler-run ABI probes.
R7j plans list those probes explicitly, and successful checks write measured
`NativeToolchainProbe` facts to `.native-build/abi.json` with compiler, target,
and native-input identity. See
[EVT1-NATIVE-LAYOUT-EVIDENCE.md](EVT1-NATIVE-LAYOUT-EVIDENCE.md).
`concept build <project>` executes the plan and writes `.native-build/build.json`.
`concept test <project>` builds the targets, links `.concept_test` harnesses to
the selected static library with the C++ driver, and runs them. Native compiler
output and the exact failing command are returned on failure. Original CMake
files remain in the TinyXML2 submodule for comparison. `Make.oct` still owns
the repository's Standard, DragonGod, and BurnIn bootstrap orchestration.
