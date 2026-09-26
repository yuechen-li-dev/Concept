# Native layout evidence

`concept check` compiles a deterministic probe source against the selected project header and executes it using the selected external C++ compiler. The probe reports `sizeof`, `alignof`, and `offsetof` for every claimed field. Concept compares those results with its field geometry and the manifest claim before `concept build` or `concept test` proceeds. The probe uses the owning native target's standard, include paths, and defines. A packed header without a matching Concept representation is rejected before running the foreign call.

Successful checks write `.native-build/abi.json` with `NativeToolchainProbe` origin, compiler/version, target triple plus host OS/architecture, measured facts, native build-input hash, and semantic companion hash. It is deterministic for unchanged inputs. The build-input hash covers manifest, sources, headers, defines, command plan, and compiler version. `concept plan` lists ABI probe intentions. This artifact is inspectable evidence; it does not replace the native library.

Every `[[repr(C)]]` record in a native companion must have a manifest ABI claim.
An omitted claim fails with `NATIVE_ABI_CLAIM_MISSING` before native tests run.

R7j1 adds optional hash covered `native_abi` evidence to `concept-module.v1`. It uses the same `NativeABIReport` as `abi.json`, including a structured compiler, target, native input, and companion identity. An artifact only consumer supplies its current identity; a match permits rederivation of `CAbiValue`, and a mismatch rejects reuse. Native companion artifact compilation may reuse a matching report or reprobe. Ordinary native check/build/test still run the probe before foreign execution and validate built output hashes before link/test. A separately supplied archive is not yet bound to artifact proof validation.
