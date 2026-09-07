# Concept EVT1 Stage 0 compiler architecture

Status: R2 value/place and record foundation

## Authority

The Go compiler is the active Stage 0 implementation.

The compiler must remain capable of compiling Concept independently of Oct.
Oct/Prometheus is a consumer, not a build-time owner.

The retired Zig compiler remains independently runnable at
`legacy/poc3-zig`; it supplies reference behavior and migration pressure, not
new compiler authority.

## Pipeline

```text
Concept source
  -> lexer / tokens / one-based spans
  -> recursive parser
  -> typed Module / AST
  -> semantic validation and type environment
  -> bounded comptime evaluation
  -> concept closure and constrained template monomorphization
  -> typed deterministic MIR
  -> core C11 lowering plus selected profile admissions
  -> generated C/H
  -> source map and generation manifest
```

`internal/concept` is intentionally one coherent package in R0. Parser,
semantic, template, comptime, automata, MIR, and C-generation code share dense
private invariants inherited from the working compiler. Splitting them before
those boundaries stabilize would replace known coupling with exported coupling.

## Core stages

The following mechanisms are compiler-core responsibilities:

- tokenization with stable line and column provenance;
- recursive syntax parsing into typed declarations and expressions;
- type identity, declaration indexing, scope validation, and diagnostics;
- payload-enum construction and exhaustive match validation;
- local and parameter place mutability, record metadata, and bounded
  copyability/immovability validation;
- non-destructive record update typing and lowering;
- named concept requirements, constraint closure, and bounded template
  specialization;
- deterministic, fuel-bounded compile-time evaluation;
- fixed-array validation in the supported compile-time domain;
- MIR construction with source spans and stable ordering;
- artifact hashing, deterministic output ordering, and strict-C11 generation.

The general core API is `concept.Parse`, `concept.Generate`, `concept.Check`,
and `concept.Write`. Public representation names are neutral (`Module`, `Type`,
`MIR`, and declaration-specific types). The active compiler ID is
`concept-evt1-stage0-go`.

## Profiles and the Vulkan boundary

Every EVT1 source unit explicitly selects `profile Core;` or `profile Vulkan;`.
Core modules cannot admit domain imports, Vulkan runtime types, effects, or
actuators. This is enforced before ordinary semantic analysis.

R1 introduces a deliberately small data-owned `ProfileDefinition`:

```text
lexer / parser
    -> core semantic environment + selected ProfileDefinition
    -> validation
    -> typed MIR
    -> core C lowering + selected profile data/hooks
```

The definition registers builtin type identity, C spelling, required headers,
synthetic fields/declarations, compiler-owned profile enums, import markers,
and effect/actuator admission. The parser and validator no longer contain a
switch or prefix test for Vulkan builtin names, and the C lowerer no longer
contains Vulkan type-name cases.

`internal/concept/profile/vulkan` is the public in-repository Vulkan consumer
boundary. It admits only Vulkan modules, then delegates parsing, MIR, and C11
generation to the core. The tightly coupled effect/actuator validator remains
temporarily in `internal/concept/profile_vulkan.go`; the file is explicitly
marked as a profile implementation seam because it depends on private semantic
environment types.

Likewise, dense effect-batch and actuator C emission remains in the cohesive
lowerer behind profile admission. R1 records this as a profile lowering hook,
not as Core language law. Splitting it would require exporting unstable private
semantic state and is outside this isolation milestone.

Automata are not assigned to Vulkan by file location. Both implementation
lines contain machine-like semantics, but their surface and execution laws are
not yet reconciled. R0 therefore preserves the Go automata implementation as a
provisional core candidate and records the unresolved decision in the language
specification and matrix.

Vulkan C types, declarations, and includes are emitted only when a
Vulkan-profile module uses registered types. Core-profile proof tests reject
Vulkan vocabulary and assert that generated artifacts contain no Vulkan
coupling. Prometheus contributes only an admitted import marker; the compiler
has no Prometheus package, builtin type, or application binding dependency.

**Vulkan is an extension of Concept EVT1. Concept EVT1 is not an extension of
Vulkan.**

## R2 value/place implementation

R2 keeps these semantics in compiler core:

- `parse.go` recognizes `record struct`, binding `const`/typed `let`, and
  `WithExpr`, preserving modifier, keyword, field-name, and value spans;
- `types.go` stores record and binding metadata on the existing struct/local
  AST shapes and exposes `record` on MIR structs;
- `validate.go` owns the bounded place classification. Scope bindings carry
  mutability, lvalue projection propagates constness, record projections force
  a record-read-only reason, and existing recursive copyability supplies the
  ordinary/record copy law;
- the same validator owns immovable final-storage, whole-assignment,
  by-value boundary, struct/record embedding, and enum-payload checks;
- `WithExpr` validation requires a copyable record, resolves fields, rejects
  duplicates and type mismatches, and visits replacement expressions in
  source order;
- `comptime.go` performs value-map copy/update for bounded compile-time record
  expressions;
- MIR records a `record_with` semantic operation. `generate.go` evaluates the
  base once into a fresh C aggregate temporary and applies updates in source
  order.

The generated C struct representation is shared by ordinary, record, and
immovable structs. C mutation used to construct a fresh record is not exposed
as Concept-level mutability. C `const` is not relied upon for correctness; the
semantic validator is authoritative.

No R2 rule is registered in `ProfileDefinition` or implemented by the Vulkan
package. A Vulkan-positive corpus case proves that the profile inherits these
core semantics unchanged.

## CLI and artifacts

`cmd/concept` is the active driver:

```console
go run ./cmd/concept --help
go run ./cmd/concept check examples/evt1/core_language.concept
go run ./cmd/concept mir examples/evt1/core_language.concept
go run ./cmd/concept emit-c examples/evt1/core_language.concept
```

`check` exercises lexing, parsing, and semantic validation. `mir` and `emit-c`
exercise the full generation path and select the corresponding deterministic
artifact for standard output. No command is a placeholder.

Generated evidence uses these identities:

- MIR schema: `concept-evt1-mir.v1`
- source-map schema: `concept-evt1-source-map.v1`
- manifest schema: `concept-evt1-generation-manifest.v1`
- compiler: `concept-evt1-stage0-go`

Golden artifacts are retained only where the extracted tests perform exact
drift checks.

## Determinism and validation boundaries

Source order is preserved where semantically visible. Generated maps and MIR
retain source spans. Template instance identity, automata topology, effects,
and output filenames are derived deterministically. Manifests contain source
and output hashes and forbid timestamps.

The C backend is an implementation boundary, not a definition of language
semantics. Profile lowering may add admitted runtime mappings, but it may not
silently broaden core parsing or typing. A profile-specific construct must be
rejected outside its profile or explicitly promoted through the specification
and reconciliation process.

## R2 limitations

- The package remains deliberately cohesive rather than prematurely split.
- Effect/actuator validation and C runtime emission remain in-package profile
  hooks; their complete physical extraction is deferred until the private
  semantic boundary stabilizes.
- Diagnostic codes retain the historical `CV` namespace for compatibility;
  R2 maps its bounded slice to semantic family names and defers renumbering.
- The backend is the extracted strict-C11 path only.
- Imports are represented, but Core multi-module compilation is not yet active.
- Automata are implemented but provisional; PoC3 machine/decide/yield laws are
  not merged.
- The complete PoC3 ownership, allocation, C ABI, testing, panic/assert,
  interfaces/dyn, slices, FixedBuffer, Option, and Result surfaces are not
  ported.

R2 does not add a borrow checker, move/drop invalidation, runtime aggregates,
or a generalized place lattice. `const Type parameter` uses the natural
existing qualifier grammar only as a parameter-place restriction.

## R1 executable evidence

`internal/concept/conformance_test.go` and `language/evt1-r1/core` provide the
bounded differential harness and canonical source corpus. The evidence covers
structs, payload enums, exhaustive match, ordinary control flow, concepts,
templates, deterministic monomorphization, bounded comptime, and compile-time
fixed arrays. `docs/conformance/EVT1-R1-CONFORMANCE.md` records classifications
and provenance; `docs/compiler/EVT1-R1-VULKAN-ISOLATION.md` records the complete
R1 leakage audit.

## R2 executable evidence

`internal/concept/r2_conformance_test.go` and `language/evt1-r2/core` provide
32 readable cases with semantic-family diagnostics, MIR checks, exactly-once
base lowering proof, Core/Vulkan inheritance evidence, and native C11 runs for
mutable struct, record update, and immovable final-storage paths.
