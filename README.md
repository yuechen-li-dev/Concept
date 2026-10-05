# Concept

Concept is a C-lineage systems language for auditable kernels, compiler, runtime, native, safety critical and bare-metal programming. 

Concept was conceived to address the issues with current low level systems programming languages, primarily C/C++, as well Rust, Go, and Zig. It's designed to primarily fill the niche currently occupied by C++ and for which Rust and Zig are inadequate replacements for. Rust's borrow checker prevents certain class of memory bugs, but lifetime propagation makes refactoring existing codebases and compile time painful. Zig's `comptime` was one of the most revolutionary ideas in programming language history, yet Zig the language is needlessly verbose and frustrating to write due to its overreliance on `comptime`, in addition to questionable governance in recent years. 

Concept was conceived, designed, and implemented by me and frontier LLMs as the "Programmer's programming language": It aims to be the fastest, most powerful, safest, and most readable programming language with all the lessons we've learned over the years, a greenfield low level systems programming language with zero unprincipled compromises.

It's a lofty north star of a goal. I hope it is achievable, so I'm going to try. Crazier things have happened.

Concept is probably too complex for most application code, you probably don't need to use Concept if you don't know exactly why you would want to use it. A GC'd language like Go/C#/TypeScript is probably a better fit for most applications. 

# Why is the language called Concept?

The language is called "Concept" for a few reasons:
1. The primary idea is that it expands the definition of C++20 concepts from compile-time template constraints to the fundamental concept behind the language.
2. It takes the multitude of the best language concepts from Rust, Zig, and modern C++/low level C#, so it's a combination of their concepts.
3. It's a very experimental language designed to explore the conceptual cutting edge of programming language design for the AI era.

Design-wise, it's a very weird programming language based off a whole bunch of weird ideas, but with a boring, very readable syntax. 

## Project status

```text
Active compiler:             EVT1 Stage 0 / Go
Compiler ID:                 concept-evt1-stage0-go
Retired reference compiler:  PoC3 / Zig
Current backend:             deterministic MIR and strict C11 C/H
Backend in progress:         EVT2 target-independent LIR and AMD64 MachineIR
```

Compiles to strict subset C11 via GCC/LLVM currently. Go compiler is being strangler-figged for self-hosting.

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

The full Go suite includes the pinned TinyXML2 native companion when its Git
submodule is present. After a fresh clone or worktree creation, run
`git submodule update --init -- tests/dogfood/tinyxml2/upstream` before the
suite to exercise that companion. Tests that require its source report a skip
when the submodule has not been initialized or Clang C++ is unavailable.

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

The PoC3 corpus lives with the compiler that reads it: fixtures in
`legacy/poc3-zig/language/`, the corpus and test scaffolding in
`legacy/poc3-zig/tests/`, examples in `legacy/poc3-zig/examples/`, and PoC3
design documents in `legacy/poc3-zig/docs/`. Their text is unchanged; current
counterparts are authored separately, per the migration policy.

## Repository layout

```text
cmd/concept, internal/concept   active Go compiler and its tests
libraries/                      Standard, DragonGod, and the Golden programs
language/evt1/                  semantic corpus, one directory per subsystem
                                (manifest.json records historical milestones)
examples/tour/                  current-syntax introduction, one subject per file
examples/evt1/                  specimens with checked-in outputs (plain = Core, _vulkan = Vulkan)
tests/                          dogfood, goldens, interop, verify, fixtures
docs/                           language, library, spec, tooling, design, examples
docs/history/                   milestone conformance logs and migration records
legacy/evt1-specimens/          frozen M-era Concept Vulkan specimens and outputs
legacy/poc3-zig/                retired Zig compiler with its fixtures, corpus,
                                examples, and design documents
legacy/dragon-god-poc/          earlier DragonGod proof of concept
```

New readers should start with `examples/tour/`.

## EVT1 authority documents

- `docs/spec/CONCEPT_EVT1_LANGUAGE_SPEC.md` defines the formal EVT1 foundation
  and labels canonical, provisional, profile-specific, legacy, and deferred
  areas.
- `docs/history/migration/EVT1-RECONCILIATION-MATRIX.md` records every major semantic
  difference and its R0 disposition.
- `docs/history/migration/EVT1-PROVENANCE.md` records exact source commits, extraction
  paths, compiler identities, and migration doctrine.
- `docs/history/migration/POC3-FIXTURE-MIGRATION.md` inventories the 1,296-fixture
  primary PoC3 corpus and defines the differential migration strategy.
- `docs/compiler/EVT1-COMPILER-ARCHITECTURE.md` describes the active Go
  pipeline and the Core/Vulkan boundary.
- `docs/library/VULKAN.md` describes Vulkan in ordinary Concept
  (`libraries/Vulkan`); `docs/history/vulkan/` preserves the M-era Concept
  Vulkan constitution and profile audit.

## Libraries

The [EVT1 domain goldens](docs/examples/DOMAIN-GOLDENS.md) exercise embedded,
civilian aerospace, game, HPC, HFT, compiler, native companion, storage, and
CAD workloads.

- `libraries/Standard` is the production home of reusable Standard modules,
  beginning with `Standard.Memory` and package metadata definitions.
- `libraries/DragonGod` is the canonical bare-metal kernel seed. It consumes
  Standard through semantic module artifacts and keeps platform support split
  between AMD64 and AArch64 modules.
- `legacy/dragon-god-poc` preserves the earlier proof-of-concept history. EVT1
  language/profile specimens remain conformance evidence, not a competing
  product tree.
