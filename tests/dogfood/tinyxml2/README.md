# TinyXML2 R7i native companion fixture

This project uses [TinyXML2](https://github.com/leethomason/tinyxml2) as a Git
submodule pinned to `8224e427b655b83dae5e2298f1e6919523a78737`. Its C++
implementation and original `xmltest.cpp` are unchanged. TinyXML2 is small,
has no required third-party dependencies, owns XMLDocument resources, and has
an existing test executable. Its upstream license is `upstream/LICENSE.txt`.
Initialize the fixture with `git submodule update --init`.

## Original build and tests

Upstream retains `CMakeLists.txt`. The direct host equivalent used here is:

```text
clang++ -std=c++17 -I upstream upstream/tinyxml2.cpp upstream/xmltest.cpp -o xmltest.exe
```

Run `xmltest.exe` from `upstream/`, where its `resources/` inputs live. On the
qualified Windows host, the upstream executable reports `Pass 528, Fail 0`.

## Concept workflow

From the Concept repository root:

```text
go run ./cmd/concept plan tests/dogfood/tinyxml2
go run ./cmd/concept check tests/dogfood/tinyxml2
go run ./cmd/concept build tests/dogfood/tinyxml2
go run ./cmd/concept test tests/dogfood/tinyxml2
go run ./cmd/concept explain tests/dogfood/tinyxml2/proofs/declared_allocation.concept --verbose
go run ./cmd/concept explain tests/dogfood/tinyxml2/proofs/unknown_effect.concept --verbose
```

`manifest.concept` contains immutable typed project data: source roots,
targets, toolchain, language standard, include roots, defines, link inputs,
companion files, tests, and an ABI claim. `concept plan` prints the exact
structured command arguments. `concept build` compiles and links the upstream
test executable and a static library containing TinyXML2 and a small C bridge.
The generated metadata is `.native-build/build.json`.

`concept/Native.concept` is the super-header: it declares the C ABI, a trusted
allocation claim, and a schema over the selected C struct. The bridge only
exposes an opaque XMLDocument handle and scalar operations. `concept check`
compiles and runs a Clang ABI probe for the struct's size, alignment, and field
offsets. `tests/native.concept_test` calls the native implementation through
the linked archive and checks the create, parse, read, destroy path. It also
checks the `NativeStatsSchema` proposition at compile time. The overlapping
native and Concept tests pass on the same host.

The native source can remain C++ indefinitely. New Concept implementation
modules are optional. Exceptions, STL types, templates, inheritance, member
functions, and overloaded C++ symbols do not cross this R7i boundary.

Current limitation: EVT1 rejects an aggregate return from `extern "C"` with
`EXTERN_C_ABI_TYPE_INVALID`. The schema and ABI probe validate the selected
struct declaration, but the `.concept_test` calls scalar bridge functions.
