# Native layout evidence

`concept check` compiles a deterministic probe source against the selected project header and executes it using the selected external C++ compiler. The probe reports `sizeof`, `alignof`, and `offsetof` for every claimed field. Concept compares those results with its field geometry and the manifest claim before `concept build` or `concept test` proceeds. The probe uses the owning native target's standard, include paths, and defines. A packed header without a matching Concept representation is rejected before running the foreign call.

Successful checks write `.native-build/abi.json` with `NativeToolchainProbe` origin, compiler/version, target triple plus host OS/architecture, measured facts, native build-input hash, and semantic companion hash. It is deterministic for unchanged inputs. The build-input hash covers manifest, sources, headers, defines, command plan, and compiler version. `concept plan` lists ABI probe intentions. This artifact is inspectable evidence; it does not replace the native library.

Every `[[repr(C)]]` record in a native companion must have a manifest ABI claim.
An omitted claim fails with `NATIVE_ABI_CLAIM_MISSING` before native tests run.

The current semantic module artifact transports the `repr(C)` declaration and ordinary `CAbiLayout` proof, but does not embed measured `.native-build/abi.json` facts. Artifact-only downstream reuse of measured ABI evidence therefore remains an R7j blocker. Every native project check/build/test re-runs the probe instead of treating a copied semantic artifact as a fresh native measurement.
