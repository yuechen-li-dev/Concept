# Concept

Concept is a C++-lineage systems language for auditable compiler, runtime,
native, and bare-metal programming.

## Project status

```text
Active compiler:             EVT1 Stage 0 / Go
Compiler ID:                 concept-evt1-stage0-go
Retired reference compiler:  PoC3 / Zig
Current backend:             deterministic MIR and strict C11 C/H
Current language status:     EVT1 R7c DragonGod scheduling
```

R0 starts a new canonical compiler line. The Go compiler architecture extracted
from Oct's Concept/Vulkan work is now the implementation seed for general
Concept. Concept/Vulkan was not renamed wholesale: Vulkan is an explicit
profile consumer, while the language specification reconciles its evidence
with the broader PoC3 corpus.

## Active compiler

The active compiler package is `internal/concept`; the active command is
`cmd/concept`. Production Concept libraries live under `libraries/`, with
ordinary Concept manifests and deterministic semantic artifacts. The root
`Make.oct` is the typed bootstrap build entrypoint; it depends only on Oct's
ordinary `Make` library and invokes the Concept package command.

```console
go run ./cmd/concept --help
go run ./cmd/concept check examples/evt1/core_language.concept
go run ./cmd/concept mir examples/evt1/core_language.concept
go run ./cmd/concept emit-c examples/evt1/core_language.concept
go run ./cmd/concept package build Standard
go run ./cmd/concept package test DragonGod
oct make Test --file Make.oct
go test ./...
go vet ./...
```

Sources explicitly select `profile Core;` or `profile Vulkan;`. Vulkan domain
imports, runtime types, effects, and actuators are rejected from the Core
profile. The Vulkan consumer API lives at
`internal/concept/profile/vulkan`.

## Retired reference compiler

The PoC3 Zig compiler is preserved under `legacy/poc3-zig` at the exact R0
cutover state. It remains runnable for semantic archaeology, differential
testing, fixture migration, and regression comparison:

```console
cd legacy/poc3-zig
zig build test
```

The repository-root `zig build test` command is also retained as a compatibility
path to the retired suite. New language development does not continue the old
Phase 22 roadmap inside the Zig compiler.

The top-level `language/`, `tests/`, `examples/phase*`,
`docs/Concept-PoC3.md`, and `docs/design/` trees remain the legacy
semantic/reference corpus shared with migration work.

## EVT1 authority documents

- `docs/spec/CONCEPT_EVT1_LANGUAGE_SPEC.md` defines the formal EVT1 foundation
  and labels canonical, provisional, profile-specific, legacy, and deferred
  areas.
- `docs/migration/EVT1-RECONCILIATION-MATRIX.md` records every major semantic
  difference and its R0 disposition.
- `docs/migration/EVT1-PROVENANCE.md` records exact source commits, extraction
  paths, compiler identities, and migration doctrine.
- `docs/migration/POC3-FIXTURE-MIGRATION.md` inventories the 1,296-fixture
  primary PoC3 corpus and defines the differential migration strategy.
- `docs/compiler/EVT1-COMPILER-ARCHITECTURE.md` describes the active Go
  pipeline and the Core/Vulkan boundary.
- `docs/reference/CONCEPT_VULKAN_LANGUAGE_CONSTITUTION.md` preserves the source
  constitution used for extraction.

## Libraries

- `libraries/Standard` is the production home of reusable Standard modules,
  beginning with `Standard.Memory` and package metadata definitions.
- `libraries/DragonGod` is the canonical bare-metal kernel seed. It consumes
  Standard through semantic module artifacts and keeps platform support split
  between AMD64 and AArch64 modules.
- `legacy/dragon-god-poc` preserves the earlier proof-of-concept history. EVT1
  language/profile specimens remain conformance evidence, not a competing
  product tree.

The next bounded milestone is R7b. R7a does not add registries, remote fetching,
lockfiles, scheduler semantics, collectors, or a stable binary ABI.
