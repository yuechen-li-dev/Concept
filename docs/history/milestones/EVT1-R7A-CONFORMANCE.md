# EVT1 R7a conformance

Baseline: `0c4588d01dc6817b1287da9ac33c9118eeb68e30`, compiler
`concept-evt1-stage0-go`, R6p commit `0c4588d` (`Implement EVT1
Standard.Memory`). Baseline toolchains were Go 1.27.0 and Zig 0.16.0; Go test,
Go vet, root Zig, and legacy PoC3 Zig were green.

R7a establishes two manifests, a deterministic acyclic package builder,
namespace transport/qualification, the production Standard root, and one
canonical DragonGod. `Make.oct` compiled-plan tests, plan emission, dry run, and
the executed default package path are evidence for bootstrap orchestration.

Focused gates cover:

- ordinary immutable manifest extraction and semantic checking;
- author credit (`CODEX`) in both Concept manifests and package graphs;
- Standard independence and DragonGod-after-Standard order;
- bounded cycle diagnostics;
- C++-style dotted namespace blocks, parent-module default namespaces,
  qualification, ambiguity candidates, artifact transport, and 100 reads;
- 100 identical package builds;
- byte-identical promoted Standard.Memory semantic artifacts;
- artifact-only DragonGod consumer compilation and strict-C11 execution;
- package-discovered Standard.Memory and DragonGod tests;
- BootInfo initialization and kernel object owner move/Drop/release.

The no-cheating audit rejects library-specific semantic branches: the compiler
implements only general namespace and package mechanisms. Package orchestration
may select package names; no DragonGod or Standard.Memory type name changes
language legality or lowering. Namespaces and manifests add no runtime metadata.

## Completion validation

- `go test ./...` passes the complete Go and semantic corpus suite, including
  the unchanged manifest totals of 382 valid, 246 static-invalid, and 13
  runtime-negative fixtures.
- `go vet ./...`, repository-root `zig build test`, and
  `legacy/poc3-zig` `zig build test` pass.
- The real Oct checkout compiles `Make.oct`, runs both compiled `Make.octest`
  facts, emits the expected Standard-before-DragonGod dry-run plan, and executes
  the default Test target with two Standard and two DragonGod package tests.
- `TestR7aStandardAndDragonGodPackagesBuildDeterministically` performs 100
  identical package builds, preserving name, author, version, kind,
  dependencies, manifest hash, module hashes, and graph hash.
- `TestR7aArtifactOnlyDragonGodConsumerRunsStrictC11` reads only package module
  artifacts during consumer compilation, then compiles and executes the emitted
  C under the repository's strict C11 harness.
- `git diff --check` passes, and the completion commit leaves a clean worktree.
