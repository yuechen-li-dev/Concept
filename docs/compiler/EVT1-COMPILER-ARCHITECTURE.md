# Concept EVT1 Stage 0 compiler architecture

Status: R4b relational lifetime proofs and selected result provenance

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

## R3 move, drop, and reference implementation

R3 remains entirely in compiler core:

- `parse.go` recognizes `move` and `ref` prefix expressions, `ref T` / `ref
  const T` types, and statement-form `if`, retaining keyword and operand spans;
- `types.go` represents move/reference expressions directly. The existing
  ownership field carries `owned`, compatibility `borrow`, and canonical `ref`
  rather than introducing a parallel type hierarchy;
- `validate.go` decides structural copyability, movability, immovability, and
  exact `void Drop(owned T)` witness matching. `owned T` is the bounded
  movable-only representation;
- scope bindings carry `Uninitialized`, `Initialized`, `Moved`, or
  `MaybeMoved` state. Whole-local/parameter moves update that state,
  reassignment restores a moved mutable owner, and cloned simple branch/loop
  scopes merge conservatively;
- call and return validation distinguishes fresh values, copyable reads, and
  explicit transfer from an existing owner. The type checker rejects implicit
  copies before C lowering;
- reference binding reuses the R2 lvalue classifier. It requires an existing
  place, checks const-to-mutable binding, preserves record-field read-only
  status, and allows immovable final storage to be mutated through `ref`;
- all reference returns are rejected in R3. This is an explicit bounded escape
  rule, not a lifetime inference framework;
- MIR operations expose `move`, `ref`, and `ref_const`. Function cleanup
  metadata lists each Drop owner in reverse declaration order with its witness
  and `live` or `transferred` state;
- `generate.go` lowers `ref T` to `T*`, `ref const T` to `const T*`, and
  materializes return expressions before emitting semantic cleanup calls.
  Moved sources are suppressed, by-value owned parameters are callee-owned,
  and the backend uses the validated Drop witness rather than inferring C
  destructors.

No R3 rule is registered in `ProfileDefinition`. The Vulkan-positive immovable
reference case passes through the same parser, validator, MIR, and C lowerer as
Core.

## R4a-R4b lifetime and semantic-requirement implementation

R4a extends the existing concept resolver rather than adding a template-local
lifetime subsystem:

```text
concept requirement
    -> requirement resolver
    -> operation / prerequisite concept / compiler analysis
    -> semantic proof result
```

`CompilerAnalysisRequirement` is the third requirement variant. A small
internal registry owns `LifetimeSafe`, `NonEscaping`, and `Outlives` with fixed
arities and check functions; it is not a plugin framework. Concept assertions
and template instantiation both pass through `checkConceptSatisfaction`, so a
lifetime proof runs because the selected concept requests it. Successful
proofs are retained in MIR as `semantic_proofs`; compiler-analysis
requirements remain compile-time-only and have no C representation.

R4b generalizes only the compiler-analysis argument boundary. A requirement
may name bounded semantic subjects while concepts and templates retain one
type parameter. `requires compiler.Outlives(source, result);` resolves
`source` to a named parameter and `result` to the result of exactly one
required operation. The internal subject record distinguishes `type`,
`parameter`, and `result`; parameter subjects carry a stable zero-based index
and all value subjects carry the selected function identity and type.

Selected result provenance follows this pipeline:

```text
function body
    -> syntax-directed result provenance summary
    -> call-site argument instantiation
    -> semantic relation request
    -> Outlives evaluator
    -> deterministic MIR proof evidence
```

`resultProvenance` is keyed by exact function signature. The bounded summary
forms are `static`, `parameter`, `shortest_of_parameters`, and `unknown`.
Direct parameter/reference return, ref-struct construction, nested ref-struct
construction, and acyclic direct pass-through calls are recognized. Recursive
cycles, ambiguous overloads, control-flow-dependent returns, and unsupported
expressions become `unknown`; there is no fixed-point region inference.

At a call site, parameter indices select actual argument provenance. Scoped
state from either the actual value or formal parameter is retained. Existing
R4a assignment/return checks then operate on that instantiated provenance, so
a helper call cannot extend an inner local or hide scoped state.

The `Outlives` evaluator is invoked only during concept satisfaction. It
returns `proven`, `disproven`, or `unknown`; only `proven` satisfies the
requirement. MIR function records expose `result_provenance`. MIR semantic
proofs expose deterministic identity, analysis name, relational subjects,
outcome, provenance facts, and requirement-origin span. None has a runtime
payload.

`Type` carries the `scoped` bit and existing ownership qualifier. `StructDecl`
and `MIRStruct` carry the `ref` aggregate classification. Semantic scope
bindings carry lexical provenance kind, depth, and scoped state. Construction
of a ref struct derives the shortest bound from reference/ref-struct fields;
assignment and return compare that bound against the destination. Unknown
provenance is rejected conservatively. Unrestricted structs and enum payloads
cannot contain references or ref structs.

The C backend lowers ref structs as ordinary aggregate values and their
reference fields as pointers. No lifetime metadata is emitted at runtime.
Scoped parameters use the same pointer ABI as references after semantic
validation.

R4a also validates MIR cleanup obligations before artifact generation. Each
recorded Drop owner must be unique, ordered, and either live or transferred.
Live `owned T` replacement emits the old Drop call before evaluation/assignment
of the new initializer and retains one final cleanup obligation for the new
value.

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

## R4b limitations

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
- Ownership beyond whole-local explicit transfer, deterministic local/
  parameter cleanup, and live whole-owner replacement remains deferred. There
  is no implicit move, field move, partial drop, unwinding, or dynamic cleanup
  stack.
- Named lifetimes, generalized borrow checking, non-lexical lifetimes, mutable
  alias analysis, Span types, and unrestricted reference-containing
  aggregates are not implemented.
- The remaining PoC3 allocation, C ABI, testing, panic/assert,
  interfaces/dyn, slices, FixedBuffer, Option, and Result surfaces are not
  ported.

R4b does not add a generalized place lattice or prove global alias/lifetime
safety. `borrow` compatibility behavior remains narrower than the canonical
explicit `ref` spelling.

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

## R3 executable evidence

`internal/concept/r3_conformance_test.go` and `language/evt1-r3/core` cover 31
readable cases: 30 PASS and one retained ordinary-struct-copy
EXPECTED-DIVERGENCE. Evidence includes moved-state diagnostics and joins,
call/return transfer, MIR cleanup ownership, generated-C ordering and
suppression, mutable and const references, record/const interaction,
immovable-by-reference use, Vulkan inheritance, and native C11 execution.

## R4a executable evidence

`internal/concept/r4a_conformance_test.go` and `language/evt1-r4a/core`
provide six valid and five invalid readable cases. They cover local and
downward-passed ref structs, scoped calls, concept-triggered lifetime proof,
local/ref-struct/scoped/outlives rejection, unrestricted aggregate rejection,
live owned replacement ordering, MIR proof metadata, and native C11 execution.

## R4b executable evidence

`internal/concept/r4b_conformance_test.go` and `language/evt1-r4b/core`
provide six valid and five invalid readable cases. They cover direct and
ref-struct call results, nested and multi-source summaries, acyclic
pass-through, scoped propagation, longer-place rejection, relational
pass/fail/unknown outcomes, deterministic proof identity, MIR inspection, and
three native C11 executions.
