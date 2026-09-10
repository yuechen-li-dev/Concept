# Library and package direction

R7a establishes `libraries/` as the production source root. Corpus directories
contain consumers and regression evidence, never the authoritative library
source.

The three identities are deliberately separate:

| Identity | Meaning | Runtime effect |
|---|---|---|
| package | distribution and build unit selected by `manifest.concept` | none |
| module | semantic compilation, import, dependency, and artifact unit | existing module semantics |
| namespace | source-level symbol organization | erased before lowering |

A module `Standard.Memory.Bump` defaults its declarations into namespace
`Standard.Memory`. An explicit C++-recognizable block such as
`namespace DragonGod.Platform.AMD64 { ... }` may select another namespace.
Several modules may contribute to one namespace. Qualification is deterministic;
an unqualified collision reports `NAMESPACE_SYMBOL_AMBIGUOUS` with sorted
candidates. `concept-module.v1` carries `namespace_symbols` and inspectable
`qualified_symbols`, so artifact-only consumers never reparse dependency source.

`manifest.concept` is ordinary Concept source, not a manifest DSL. Its authority
is an immutable `comptime PackageManifest Manifest` assembled from record
structs, an enum, strings, a fixed array, and integers. R7a's schema contains
name, author, exact three-part version, package kind, sorted local dependencies,
and a bounded dependency count. Both initial packages credit `CODEX`. Filename
discovery is conventional; the contents are parsed by the normal frontend,
structurally extracted through the typed AST shape, and then semantically checked
against `Standard.Build.Metadata`. No reflection, serialization framework,
registry, download, solver, or runtime package metadata is involved.

The package DAG comes only from manifest values. Module edges still come only
from imports. R7b retains `Standard -> nothing` and `DragonGod -> Standard`;
cycles fail as `PACKAGE_DEPENDENCY_CYCLE`. Dominatus is a parity oracle, not a
third package: agentic-kernel capabilities live in semantic `DragonGod.*`
modules without creating a parallel product namespace.

`Make.oct` remains temporary bootstrap orchestration. It is retained because it
is adequate and avoids shell-script build logic. Its typed plan invokes the
Stage-0 `concept package build|test` surface, which discovers manifests, builds
package and module DAGs, and writes deterministic artifacts. Self-hosted Concept
build orchestration is not required yet.

Outputs live under `artifacts/<Package>/`: `package.json` plus
`modules/<Module/Identity>.concept-module.json`. Package identity includes the
compiler ID, manifest SHA-256, and each integrity-checked semantic module hash;
module hashes already include source and dependency hashes.
