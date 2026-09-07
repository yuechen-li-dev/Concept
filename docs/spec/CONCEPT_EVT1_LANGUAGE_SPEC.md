# Concept EVT1 language specification foundation

Status: R2 value/place semantics

This document defines the authority categories and the smallest currently
executable EVT1 language foundation. It is derived from the retired Concept
PoC3/Zig implementation and the extracted Concept/Vulkan/Go implementation.
Where they disagree, this document records a direction or defers the decision;
it does not treat implementation accident as language law.

The classifications used throughout are:

- **Canonical EVT1**: selected language law with an active Go proof.
- **Provisional EVT1**: implemented or strongly indicated, but not yet closed
  against both implementation lines.
- **Profile-specific**: admitted by a named profile and not general core.
- **Legacy PoC3**: preserved behavior with no current EVT1 promise.
- **Deferred reconciliation**: an open cross-line design decision.

## 1. Language identity

**Canonical EVT1.** Concept is a C++-lineage systems language for auditable
compiler, runtime, native, and bare-metal work. The canonical bootstrap line is
Concept EVT1 Stage 0 implemented in Go, compiler ID
`concept-evt1-stage0-go`. Concept/Vulkan is a source and a profile, not the
language identity. PoC3/Zig is a retired reference implementation.

## 2. Design laws

**Canonical EVT1.** Essential difficulty remains visible; accidental difficulty
is removed or quarantined; hazardous operations require deliberate spelling;
compile-time behavior is deterministic and bounded; generated authority is
inspectable; profiles may admit domain semantics but may not silently change
core meaning. No hidden heap, scheduler, lifetime calculus, or ambient effect
system is implied by syntax.

## 3. Source files and modules

**Canonical EVT1 foundation.** Source uses `.concept`. An EVT1 source unit begins
with exactly one explicit profile declaration:

```concept
profile Core;
```

or:

```concept
profile Vulkan;
```

**Provisional EVT1.** Dotted imports are represented after the profile line.
Core multi-module resolution is not implemented, and Core currently rejects
domain imports. PoC3 module/import fixtures remain the authority pressure for a
later multi-module decision.

## 4. Naming and formatting

**Canonical EVT1.** User-facing types, functions, and compiler-known operations
use `PascalCase`; parameters and locals use `camelCase`. Declarations follow the
C++ lineage:

```concept
Result<ProbeEvidence, PrometheusError> Execute(
    borrow MechanismContext context,
    unsafe imported borrow AccelerationStructure admittedTlas)
{
    // ...
}
```

Return type precedes function name. EVT1 does not use `fn`, `name: Type`,
`-> ReturnType`, inferred `let`, or `var`. R2 accepts `let Type name = value;`
only as an exact alias of `const Type name = value;`; `const` is canonical and
documentation should prefer it. Braces and indentation remain human-readable,
with one statement per line. This resolves a material PoC3 surface conflict in
favor of the newer Concept/Vulkan direction; PoC3 syntax remains accepted only
by the retired compiler until translated.

## 5. Declarations

**Canonical EVT1 foundation.** R2 parses named structs, record structs, immovable structs,
enums, concepts, constrained function templates, compile-time declarations,
static assertions, ordinary functions, and profile-admitted declarations.
Declarations use explicit types and semicolon-terminated fields/statements.

**Deferred reconciliation.** Visibility, namespaces, complete module exports,
attributes, overload/coherence breadth, and declaration ordering beyond the
implemented subset are not fixed here.

## 6. Primitive types

**Canonical EVT1 foundation.** `int`, `bool`, and `void` are core primitive
types in the implemented subset.

**Provisional EVT1.** `string` is available to bounded compile-time evaluation
and diagnostics but does not establish a general runtime string model.
`uint64` exists in the seed implementation but awaits cross-line primitive and
width rules.

**Profile-specific.** `PipelineLayout`, `Pipeline`, `VulkanError`, `VkBuffer`,
and `VkCommandPool` are Vulkan admissions, not core primitives.

## 7. Ownership and storage vocabulary

**Canonical EVT1 R2.** In a local declaration, `const` qualifies the binding
and its projected places, not the value type. A const local can be read and a
copyable value can be copied from it, but the local cannot be reassigned and a
mutable struct field cannot be written through it. R2 also applies the same
place rule to the already-natural `const Type parameter` qualifier. This is a
parameter-place rule, not a full borrowing model.

**Provisional EVT1.** The Go compiler additionally parses and validates
`borrow`, `owned`, `imported`, pointer, and `unsafe` vocabulary within its
bounded type rules. These spellings preserve the Concept/Vulkan evidence and
align with PoC3's requirement that storage and hazards be visible.

**Legacy PoC3 / deferred reconciliation.** Full move invalidation, explicit
`move`, drop scheduling, RAII, allocator/store ownership, and PoC3 borrow/place
laws are not yet ported. R2 does not infer them from the accepted vocabulary.

## 8. Values, structs, records, and places

**Canonical EVT1 R2.** `struct Name { Type field; }` declares an ordinary
mutable value type. Fields retain declaration order. Positional construction,
field reads, field writes through mutable places, whole-value assignment, and
copying are supported when every contained field is copyable. A struct that
contains a non-copyable field is itself non-copyable.

`record struct Name { Type field; }` declares an immutable value type using
the same declaration-ordered aggregate representation. Record fields are
readable but never user-assignable, even through a non-const binding. A record
is copyable exactly when all fields are copyable. This is type-level field
immutability; it is distinct from binding/place immutability supplied by
`const`. A non-const record binding may therefore be rebound as a whole to a
new copyable record value even though none of its field projections is mutable.

The bounded R2 place classes are:

- mutable local place;
- const local or const parameter place;
- record-field read-only projection;
- immovable final-storage place;
- temporary/rvalue, which is not assignable.

A mutable struct field projected from a mutable place remains mutable. The
same field projected from a const place is read-only. Every record-field
projection is read-only. An immovable local's fields may be mutated in its
final storage, but its whole value is never copy-assignable. This is not the
future ownership/place lattice and establishes no general alias or borrow law.

### 8.1 Non-destructive record update

`base with { field = value; }` evaluates `base` exactly once, requires a
copyable record value, copies its fields, evaluates replacements in source
order, and produces a fresh record value. Unspecified fields retain their
source values and the source is unchanged. Fields must exist, may appear only
once, and replacement types must match. R2 rejects `with` on ordinary structs,
non-record values, and records containing a non-copyable field. Dotted update
targets are not syntax; nested updates compose as nested `with` expressions.

The MIR records `record_with` aggregate-copy/update intent. The C backend may
construct the fresh value through a temporary and field assignments; those
assignments are compiler-generated construction and do not make record fields
mutable in Concept.

### 8.2 Immovable structs

**Canonical EVT1 R2 subset.** `immovable struct` declares mutable,
storage-bound data. It may be constructed directly in final local storage and
its fields may be mutated through that mutable place. Whole-value copy and
assignment, pass-by-value, return-by-value, struct or record embedding by
value, and enum payload embedding by value are rejected. Borrow/reference
semantics beyond the already-implemented bounded subset remain deferred; no
implicit relocation is permitted.

### 8.3 Structural equality

The existing bounded compile-time structural equality implementation applies
to record structs when every field supports equality. R2 adds no generated
runtime operators.

## 9. Enums and payload enums

**Canonical EVT1 foundation.** Enums may contain nullary and payload variants.
Variants have deterministic declaration-order tags. Payload fields are typed
and their construction arity is checked. The current spelling is:

```concept
enum Status
{
    Empty,
    Ready(int value)
}

Status status = Status::Ready(7);
```

Generic built-in Option/Result semantics are not implied by this facility.

## 10. Match

**Canonical EVT1 foundation.** `match (value) { ... }` supports exhaustive enum
matching in expression and supported statement positions. Arms use qualified
variants; payload patterns bind names without repeating types:

```concept
return match (status)
{
    Status::Empty => 0,
    Status::Ready(value) => value,
};
```

Missing or duplicate variants are errors. The bounded Go pattern subset does
not establish wildcard, guard, range, or recursive-pattern law for EVT1.

## 11. Functions

**Canonical EVT1 foundation.** Ordinary functions use return-type-first
declarations, typed parameters, block bodies, explicit return statements,
calls, local declarations, assignment, foundational `while`, and `if`
expressions. Function names are PascalCase; parameters and locals are
camelCase.

**Provisional EVT1.** Overload resolution is bounded to the implemented exact
signature and template requirement rules. Full PoC3 callable and module rules
are deferred.

## 12. Failure model

**Provisional/Profile-specific.** The extracted Vulkan line uses
`Result<T, E>`-shaped fallible signatures and `?` in admitted mechanisms. That
evidence does not canonize a general core Result value type or complete
propagation law.

**Legacy PoC3.** PoC3 has function-level fallibility and a compiler-known
Option subset. PoC3 Result was still deferred at cutover.

**Deferred reconciliation.** EVT1 must distinguish absence, recoverable value
errors, function-level propagation, and panic before Result becomes canonical.

## 13. Concepts

**Canonical EVT1 foundation.** Named concepts have one type parameter and a
finite set of named operation or prerequisite requirements. Explicit
`requires ConceptName<ConcreteType>;` assertions request a compile-time proof.
Requirement closure and cycles are diagnosed deterministically.

**Deferred reconciliation.** Multiple parameters, specialization, negative
concepts, orphan/coherence breadth, and the full PoC3 marker-concept system are
not admitted by R0.

## 14. Templates

**Canonical EVT1 foundation.** The seed supports one-type-parameter function
templates constrained by a named concept. Instantiations are validated against
the requirement closure and monomorphized deterministically into MIR/C symbols.

**Deferred reconciliation.** Template types, template structs/enums, multiple
type parameters, partial specialization, unconstrained templates, and broad
overload interaction remain outside the R0 subset.

## 15. Compile-time evaluation

**Canonical EVT1 foundation.** `comptime` declarations/functions and
`static_assert` evaluate deterministic pure values using explicit fuel, call
depth, loop bounds, and aggregate limits. Supported values include integers,
booleans, strings in the compile-time domain, enums, structs, and fixed arrays
composed from supported values. Compile-time `while` requires a static
`bounded(limit)` clause.

**Deferred reconciliation.** PoC3 capability-based compile-time permissions,
compiler-owned temporary allocation, reflection breadth, and any I/O are not
part of the current canonical subset.

## 16. Arrays and indexing

**Canonical EVT1 foundation.** Fixed `T[N]` arrays, literals, `Len`, and
indexing are supported in the bounded compile-time domain, with exact length,
element-type, and bounds checks.

**Legacy PoC3 / deferred reconciliation.** PoC3 runtime arrays and their MIR/C
behavior are not ported. Runtime array law must be reconciled before promotion.

## 17. Slices and bounded collections

**Legacy PoC3.** PoC3 implemented read-only `Slice<T>` and
`FixedBuffer<T, N>` foundations.

**Deferred reconciliation.** They are not in the Go seed. FixedBuffer mutation,
try helpers, and conversion rules remain future work; R0 does not continue the
old Phase 22 roadmap.

## 18. Interfaces and dyn

**Legacy PoC3.** PoC3 contains interface declarations and bounded dynamic
dispatch fixtures.

**Deferred reconciliation.** The Go seed has no equivalent core feature. No
interface object representation, dyn storage, vtable ABI, or ownership law is
canonical EVT1 yet.

## 19. Allocation

**Legacy PoC3.** PoC3 explored explicit allocator, arena, store, allocation
effect, and index-based identity semantics.

**Deferred reconciliation.** The Go seed adds no general allocation model.
EVT1 has no implied heap or default allocator. Profile-generated native code
may call admitted mechanisms without making those calls core allocation law.

## 20. C ABI

**Legacy PoC3.** PoC3 implements `extern`, export, and `repr(C)` pressure with
fixtures.

**Deferred reconciliation.** The current C11 generator is a bootstrap backend,
not a finalized Concept C ABI. Export naming, layout guarantees, header
contracts, and FFI-safe type derivation require explicit EVT1 decisions.

## 21. Machines and automata

**Provisional EVT1 / deferred reconciliation.** PoC3 defines core `machine`,
states, transitions, `decide`, nested machine composition, and `yield` behavior.
The Go seed defines bounded `automata -> machine -> state`, typed fixed local
instances, guarded deterministic dispatch, push/pop continuation constraints,
and compiler-owned outcome values.

R0 preserves the Go implementation as a provisional core candidate because
both lines demonstrate general control semantics. It does not declare either
surface canonical, does not alias `machine` and `automata`, and does not import
PoC3 decide/yield semantics into the Go model.

## 22. Effects, actuators, and profiles

**Profile-specific.** R0 admits `effect`, ordered emitted-effect batches, and
`actuator` mappings only under `profile Vulkan;`. The Vulkan profile owns
mechanism types, Prometheus imports, mapping admissibility, and Vulkan C
bindings.

**Deferred reconciliation.** PoC3 discusses broader effect/default-profile
laws, but R0 does not promote the Vulkan spelling or implementation to core.
Any general effect system requires a separate cross-line decision.

## 23. Testing

**Legacy PoC3.** `.con_test`, discovery, facts/theories, expectations, and the
Zig-hosted test runner remain executable reference behavior.

**Deferred reconciliation.** The Go compiler has Go-level compiler tests but no
canonical Concept testing language. R1 differential tests are infrastructure,
not automatic adoption of the PoC3 testing surface.

## 24. Panic and assert

**Legacy PoC3.** PoC3 implements runtime `panic`, runtime `assert`, test
assertions, stable reasons, and a deterministic failure path.

**Provisional EVT1.** `static_assert` is canonical only in the compile-time
domain described above.

**Deferred reconciliation.** Runtime panic/assert syntax, exit behavior,
reason representation, and interaction with Result/fallibility are not yet Go
EVT1 features.

## 25. Backend and runtime boundaries

**Canonical EVT1 foundation.** The active bootstrap backend produces strict
C11 C/H, deterministic MIR JSON, a source map, and a hash manifest. Backend
output is derived evidence; the typed Module, semantic checks, and MIR carry
compiler authority. Timestamps are forbidden. Source spans and stable ordering
are preserved.

**Profile-specific.** Vulkan includes, runtime types, capability names, and
mechanism mappings appear only when admitted by the Vulkan profile.

The C representation is not itself a language specification. Runtime services,
Oct, and Prometheus are consumers and may not become hidden compiler build
dependencies.

## 26. Deferred EVT1 items

The following remain explicit reconciliation or implementation work:

- general imports and multi-module compilation;
- complete primitive widths and conversion rules;
- full ownership, move, borrow, drop, and storage semantics;
- the failure-model relationship among Option, Result, fallibility, and panic;
- runtime arrays, slices, FixedBuffer, and bounded mutation;
- interfaces and dyn storage/dispatch;
- allocation, allocator effects, arenas, and stores;
- stable C ABI and layout law;
- canonical machine/automata syntax and decide/yield semantics;
- general effects versus profile-owned effects/actuators;
- Concept-native testing and runtime panic/assert;
- broader concepts/templates and capability-based comptime.

R0 intentionally adds none of these merely because they were next on the PoC3
roadmap. Promotion requires a matrix decision, a specification update, and
executable EVT1 evidence.
