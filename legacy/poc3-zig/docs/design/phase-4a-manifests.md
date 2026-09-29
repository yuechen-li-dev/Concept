# Phase 4a manifest architecture

Phase 4a introduces Concept's package, build, dependency, and registry metadata model.

This is **not** a return to C/C++ source headers. Concept still has no textual include headers, no duplicated declarations, and no `.h` / `.cpp` split.

Instead, Concept has **manifests**: deterministic, sectioned `.conception` metadata files that describe packages, module roots, build targets, dependency sources, registries, profiles, and restricted build-time behavior.

## Thesis

```text
Concept has no source headers.
Concept has manifests.

A Concept manifest is a deterministic, restricted .conception file
that declares package identity, module layout, dependencies, registries,
targets, build profiles, and build-time permissions.
```

Concept source remains:

```text
*.concept
```

Concept test source remains:

```text
*.con_test
```

Concept meta-language files use:

```text
*.conception
```

The canonical package manifest is:

```text
Concept.manifest.conception
```

A registry repository may declare itself with:

```text
Concept.registry.conception
```

Optional lock files may exist later, but **Phase 4a v0 has no lock files**.

## Why this exists

C and C++ split source meaning, build configuration, dependency resolution, and package identity across headers, build scripts, package managers, IDE project files, generated metadata, and tribal knowledge.

Concept should not repeat that.

A Concept package should be inspectable as one deterministic metadata object:

```text
package identity
module roots
dependencies
registries
targets
profiles
build permissions
test targets
examples
native/link settings later
```

This replaces the practical need that C/C++ headers and build scripts partly tried to satisfy, without adopting their textual inclusion model.

## Terminology

### Source files

Normal Concept source:

```text
*.concept
```

Test Concept source:

```text
*.con_test
```

### Manifest files

Root package manifest:

```text
Concept.manifest.conception
```

Registry manifest:

```text
Concept.registry.conception
```

Optional future lock file:

```text
Concept.lock.conception
```

Lock files are intentionally deferred. Reproducibility in Phase 4a starts with immutable manifest pins.

## Headers vs manifests

Concept has no C-style source headers.

Rejected:

```text
Foo.h
Foo.cpp
#include "Foo.h"
duplicated declarations
textual include order
forward-declaration ceremony
```

Accepted:

```text
Concept.manifest.conception
module roots
package targets
dependency pins
registry sources
build profiles
```

The rule:

```text
Source headers duplicate program meaning.
Manifests describe package and build meaning.
```

Concept may still generate C headers internally for a C backend one day, but those are backend artifacts, not Concept language artifacts.

## `.conception` as meta-language

`.conception` is Concept's deterministic meta-language format.

It is used for:

```text
language fixtures
manifests
registries
future lock files
possibly generated IR snapshots
```

A `.conception` file is sectioned, explicit, and designed for tooling.

Example shape:

```text
# kind: package
# format: concept-manifest-v0

=== package ===
package Compiler;
version "0.1.0";

=== modules ===
module Compiler.Source {
    root "src/compiler/source";
}

=== targets ===
target Cathedral {
    kind executable;
    module Compiler.Main;
}
```

## Restricted Concept subset

Manifest `.conception` files are not arbitrary full Concept programs.

Phase 4a manifests should be:

```text
deterministic
declarative
safe to parse before compiling package source
safe to cache
safe for LLM tooling
safe for CI
explicit about external effects
```

Allowed in v0:

```text
string literals
integer/bool literals where needed
qualified names
records/blocks
lists
package/module/target/dependency declarations
version strings
git URLs
git revisions
simple symbolic references
```

Not allowed in v0:

```text
arbitrary loops
arbitrary filesystem reads
ambient environment variables
network access outside declared dependency/registry fetching
arbitrary process execution
full comptime execution
Turing-complete build scripts
```

A future advanced build system may permit restricted `comptime(build)` functions, but only with explicit capabilities.

## Package manifest example

```text
# kind: package
# format: concept-manifest-v0

=== package ===
package Compiler;
version "0.1.0";
license "MIT";

=== modules ===
module Compiler.Source {
    root "src/compiler/source";
}

module Compiler.Parser {
    root "src/compiler/parser";
}

module Compiler.Semantics {
    root "src/compiler/semantics";
}

module Compiler.Main {
    root "src/compiler/main";
}

=== registries ===
registry Yuechen {
    git "https://github.com/yuechen-li-dev/concept-registry";
    rev "3d4f1a2b";
}

=== dependencies ===
dependency Core {
    registry Yuechen;
    package Core;
    version "0.3.0";
}

dependency Prometheus {
    git "https://github.com/yuechen-li-dev/Prometheus";
    rev "a13f9c2d";
}

=== targets ===
target Cathedral {
    kind executable;
    module Compiler.Main;
}

target CompilerTests {
    kind tests;
    modules [
        Compiler.Source.Tests,
        Compiler.Parser.Tests,
        Compiler.Semantics.Tests
    ];
}

=== profiles ===
profile debug {
    optimize none;
    diagnostics full;
}

profile release {
    optimize speed;
    diagnostics compact;
}

=== resolution ===
mode pinned;
```

This example is illustrative. Phase 4a v0 should implement only a small model, not every section immediately.

## P4a-M2 package modules and targets

P4a-M2 extends the restricted `Concept.manifest.conception` parser for package manifests with deterministic, manifest-local declarations for module roots and build targets.

Implemented in this milestone:

```text
=== modules ===
module Compiler.Source {
    root "src/compiler/source";
}

=== targets ===
target Cathedral {
    kind executable;
    module Compiler.Main;
}

target CompilerTests {
    kind tests;
    modules [
        Compiler.Source.Tests,
        Compiler.Parser.Tests
    ];
}
```

The supported target kinds are:

```text
executable
library
tests
```

Module and target references are validated only against declarations in the same manifest. This is intentionally local metadata validation, not build planning. P4a-M2 does **not** add filesystem scanning, dependency resolution, registry lookup, lock files, package driver integration, native linker settings, workspaces, or arbitrary build scripting.

## Dependency resolution policy

Concept should support reproducibility without requiring lock files.

Phase 4a v0 should support dependency declarations that are either floating or pinned.

Floating dependency:

```text
dependency Core {
    registry Yuechen;
    package Core;
    version "^0.3";
}
```

Pinned dependency:

```text
dependency Prometheus {
    git "https://github.com/yuechen-li-dev/Prometheus";
    rev "a13f9c2d";
}
```

Manifest-level resolution policy:

```text
=== resolution ===
mode pinned;
```

Possible modes:

```text
floating
pinned
```

Suggested v0 behavior:

```text
mode floating:
  version ranges and branch/tag dependencies are allowed.

mode pinned:
  dependencies must resolve to immutable revisions.
  direct git dependencies must specify rev.
  registry dependencies must resolve through a pinned registry revision to an exact package revision.
```

This makes lock files optional rather than mandatory.

## Lock files

Phase 4a v0 has no lock files.

Future optional lock file:

```text
Concept.lock.conception
```

A future lock file may record a resolved transitive graph for apps, CI, release builds, vendoring, or reproducible binary distribution.

But lock files should not be the only way to get reproducible builds.

The manifest itself must be able to say:

```text
what I want is exactly this dependency graph root,
pinned through exact registry and dependency revisions.
```

Recommended rule:

```text
Libraries may use floating compatibility ranges.
Applications and release builds should use pinned manifests or optional lock files.
```

## Git registries

Concept should not require a central registry.

Any git repository may declare itself to be a Concept package registry by containing:

```text
Concept.registry.conception
```

Example:

```text
# kind: registry
# format: concept-registry-v0

=== registry ===
name Yuechen;
namespace yuechen;

=== packages ===

package Core {
    version "0.3.0" {
        git "https://github.com/yuechen-li-dev/concept-core";
        rev "abc123";
    }

    version "0.4.0" {
        git "https://github.com/yuechen-li-dev/concept-core";
        rev "def456";
    }
}

package Prometheus {
    version "0.1.0" {
        git "https://github.com/yuechen-li-dev/Prometheus";
        rev "9912aaa";
    }
}
```

Package manifests can then reference registries:

```text
=== registries ===

registry Yuechen {
    git "https://github.com/yuechen-li-dev/concept-registry";
    rev "3d4f1a2b";
}
```

Pinning the registry revision is important for reproducibility. Otherwise the registry itself floats.

This avoids the NPM problem by making registry choice:

```text
explicit
forkable
auditable
reproducible
not centralized by default
```

## Templates, concepts, and comptime in manifests

Concept's manifest model should eventually use Concept's own strengths, but not all at once.

### Concepts

A future build system can model build capabilities as concepts:

```cpp
concept BuildTarget<T> {
    BuildArtifact build(T& target, BuildContext& ctx);
}

concept Platform<P> {
    String triple(P& platform);
    bool supportsThreads(P& platform);
}
```

A manifest target can then be checked against target concepts.

Example future direction:

```text
target Cathedral : Executable {
    module Compiler.Main;
}
```

The manifest does not need to implement this in v0. The important design direction is that build targets are not untyped strings forever; they can become checked manifest concepts.

### Templates

Templates are useful when package/build metadata has repeated typed structure.

Future examples:

```cpp
template<T: TestTarget>
target TestSuite<T> {
    kind tests;
    module T.module;
}
```

Templates should not be part of Phase 4a v0. They are a future way to avoid repetitive manifest declarations while staying typed and constrained.

### Comptime

Comptime is useful for deterministic build-plan generation, but must be capability-limited.

Future example:

```cpp
comptime(build)
BuildPlan plan(BuildContext& ctx) {
    return discoverPackageTargets(ctx.package);
}
```

But this must not become CMake in robes.

Any comptime manifest execution must declare capabilities:

```text
allow read_fs "manifest/";
deny network;
deny process;
```

Phase 4a v0 should remain declarative.

## Determinism rules

Manifest evaluation must be deterministic by default.

Phase 4a v0 should avoid:

```text
ambient current directory surprises
environment-variable lookup
system clock access
host filesystem scanning outside declared roots
network access outside declared registries/dependencies
arbitrary command execution
```

Declared roots are allowed:

```text
module Compiler.Parser {
    root "src/compiler/parser";
}
```

Declared registries/dependencies are allowed:

```text
registry Yuechen {
    git "...";
    rev "...";
}
```

Everything else should be explicit future capability work.

## Build targets

Phase 4a v0 target kinds can be minimal:

```text
executable
library
tests
```

Example:

```text
=== targets ===

target Cathedral {
    kind executable;
    module Compiler.Main;
}

target CompilerTests {
    kind tests;
    modules [
        Compiler.Tests
    ];
}
```

Targets should eventually compile into a deterministic build plan.

A build plan is not the same as the manifest. The manifest is user-authored package metadata; the build plan is derived tool output.

## Profiles

Profiles describe build settings:

```text
=== profiles ===

profile debug {
    optimize none;
    diagnostics full;
}

profile release {
    optimize speed;
    diagnostics compact;
}
```

Phase 4a v0 can parse profiles without applying them deeply.

Future profile settings may include:

```text
optimization level
diagnostic verbosity
target triple
backend choice
debug info
warnings/errors policy
comptime permissions
unsafe policy
```

## Native/link settings

Native and linker settings are important but should not dominate v0.

Possible future section:

```text
=== native ===

link system "m";
link framework "CoreFoundation";
include_path "vendor/include";
library_path "vendor/lib";
```

This should be explicit and target/profile-scoped later.

Do not implement this in Phase 4a v0 unless needed.

## Manifest parser strategy

Phase 4a should not reuse the full Concept parser.

Manifests are `.conception` files and should have a restricted parser/model.

Initial implementation should parse:

```text
headers:
  # kind: package
  # format: concept-manifest-v0

sections:
  === package ===
  === modules ===
  === dependencies ===
  === registries ===
  === targets ===
  === profiles ===
  === resolution ===
```

Within sections, v0 may parse only a small declaration grammar.

Do not make manifests arbitrary scripts.

## Phase 4a milestone ladder

```text
P4a-M0  Manifest/package design document
P4a-M1  Manifest parser skeleton and data model
P4a-M2  Package identity, module roots, and target declarations
P4a-M3  Dependency and registry declarations
P4a-M4  Resolution policy and pinned dependency validation
P4a-M5  Manifest fixture corpus and diagnostics
P4a-M6  Minimal package/build-plan skeleton
P4a-M7  CLI/package-driver integration skeleton
P4a-M8  Registry manifest skeleton
P4a-M9  Phase 4a closeout
```

For now, only P4a-M0 through P4a-M1 or P4a-M2 need to happen before Phase 5. The rest can wait.

## Phase 4a v0 close criteria

Phase 4a is complete enough to pause when:

* `Concept.manifest.conception` format is documented.
* A manifest parser/model exists.
* Package identity can be parsed.
* Module roots can be parsed.
* Basic target declarations can be parsed.
* Basic dependency/registry declarations are at least represented or documented.
* Resolution policy exists in the model.
* Manifest diagnostics exist for malformed required fields.
* No build execution is required yet.
* No lock files are required.
* No arbitrary build scripting is supported.

## Known limitations

Phase 4a v0 intentionally does not include:

```text
lock files
full dependency solving
network fetch implementation
git fetch implementation
registry resolution implementation
full build planning
native linker integration
target triples
workspace support
comptime build scripts
template expansion in manifests
concept solving in manifests
C ABI/export integration
multi-file module compilation
```

These belong to later package/build-system work.

## Design summary

Concept source files are not headers.

Concept packages have manifests.

The manifest is:

```text
sectioned
deterministic
restricted
auditable
LLM-friendly
git-registry-aware
capable of immutable dependency pins without requiring lock files
```

The root package file is:

```text
Concept.manifest.conception
```

The registry file is:

```text
Concept.registry.conception
```

Future lock files may be:

```text
Concept.lock.conception
```

but they are optional and not part of Phase 4a v0.

The guiding rule:

```text
No source headers.
No central registry assumption.
No hidden build scripts.
No mandatory lock files.
Manifests are explicit package/build law.
```

## P4a-M1 implementation note

P4a-M1 adds the initial Stage 0 manifest parser and data model skeleton in `src/compiler/manifest.zig`.

Implemented in this milestone:

* package and registry manifest kind representation
* `concept-manifest-v0` and `concept-registry-v0` format representation
* deterministic `# kind:` and `# format:` header parsing and kind/format validation
* section parsing with source-order retention and duplicate-section rejection
* minimal package identity parsing for `package`, optional `version`, and optional `license` statements

Still deferred to later Phase 4a milestones:

* module roots
* targets
* dependencies
* registries beyond header/section representation
* resolution policy
* lock files
* package-driver/build-plan integration
* git/network fetching
* template expansion
* concept solving
* comptime build execution
