# Packages and native builds

A package is a distribution and build unit. A module is an import and semantic
compilation unit. Both use ordinary Concept source; `manifest.concept` is a
conventional filename, not a second language.

## Start a package

```console
concept init MySimulation --author "Your name"
concept package build MySimulation
concept package test MySimulation
concept package test MySimulation --verify
```

`init` writes `manifest.concept`, `src/MySimulation.concept`, and a fact under
`tests/`. Its sample exports `Answer()` and the fact executes it. Package names
must be identifiers; use `--name Name` when the destination directory has a
nonidentifier name. Existing files are rejected before initialization and are
never overwritten, including a concurrent exclusive-create failure.

The manifest imports `Standard.Build.Metadata` and declares one immutable
`comptime PackageManifest Manifest` with name, author, version, kind, sorted
local dependencies, and their active count. Directory package builds put
semantic artifacts under `.concept/packages`. Existing identity commands such
as `concept package build Standard` remain available.

Set `CONCEPT_LIBRARY_ROOT` to an installed distribution's `libraries` directory.
The driver also checks for `libraries` beside its executable and supports its
source-checkout library root during development. `--library-root` selects the
root for ordinary directory package builds. Dependencies are local installed
packages; these commands do not fetch repositories or solve remote versions.

## Build native consumers

A manifest can additionally declare `comptime NativeProjectManifest Native`.
`concept package build <directory>` uses the existing native project builder:
structured compiler arguments, declared source roots, C11/C17 or C++17/20/23,
static libraries or executables, measured ABI claims, and input/output hashes.
Clang and GCC toolchains are supported. The native compiler and C++ standard
library remain authoritative for language and library support.

Native input paths are relative to the package. An explicit installed-package
input such as `@Vulkan/native/device_runtime.c` resolves through
`CONCEPT_LIBRARY_ROOT`; `@Vulkan/native` is an include/source root.
`@VulkanSDK/include` resolves through `VULKAN_SDK`, or the system `/usr` SDK on
Unix when its Vulkan headers exist. Dependency names and suffixes are checked;
parent traversal is rejected. Source and header bytes from these roots enter
the build identity. Output paths must remain inside the consumer package.
Static library objects use position-independent code on Unix for shared-library
consumers. Fixed-capacity source/include/link/test lists may end in empty padding.
ABI field arrays retain their separate paired field/offset padding rule.

A native package may build without companion tests. `package test` still rejects
a manifest with no test functions; building an archive is not test execution.
A manifest with companions must supply measured claims for its `repr(C)`
records. The Vulkan profile supplies its library's native producer evidence.

## Keep foreign authority explicit

`extern "C"` describes the ABI. `foreign concept` declares specific bounded
semantic facts about a foreign operation, for example allocation. A caller does
not gain freshness, disjointness, ownership, or host accessibility merely from
an extern signature. Keep owner lifetime in Concept wrappers and retain the
native layout and lifetime evidence required by the library.

The Prometheus consumer in the sibling Oct repository uses this path to build
its generated Concept reduction executor, a small C ABI facade, and C++23
package/queue adapters against the Vulkan library in this distribution.
