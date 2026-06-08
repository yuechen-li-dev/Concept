# Phase 9 compile-time execution

P9-M0 is a documentation-only milestone. It defines Concept's compile-time execution model before any lexer, parser, AST, HIR, MIR, backend, fixture, or evaluator implementation work begins.

Phase 9 starts after Phase 8 closed the concepts/templates slice. Phase 8 established generic function templates, symbolic type parameters, monomorphization, concept declarations, impl declarations, marker concepts, constrained generic functions, witness calls, and concrete-only MIR/backend behavior. That ordering is deliberate: Concept's generic programming system is templates plus concepts, not compile-time execution.

## Thesis

```text
Phase 9 introduces compile-time execution without turning the compiler into a haunted build script.

`comptime` is allowed to compute.
It is not allowed to observe or mutate the host environment by default.

Compile-time magic must be typed, deterministic, capability-gated when impure,
and erased into ordinary checked program structure before MIR/backend.
```

## Core doctrine

```text
Templates/concepts are the generic programming system.
`comptime` is not the generic programming system.

`comptime` is for:
  constant computation
  static assertions
  table generation
  layout checks
  target metadata queries
  static reflection
  generated declarations

But Phase 9 v0 starts with hermetic deterministic constant evaluation.
```

Concept should not reintroduce C++ template metaprogramming by calling it `comptime`. Compile-time execution is a controlled evaluator for values and declarations the compiler can check, diagnose, and erase before runtime lowering.

## Relationship to PoC3

PoC3 positions `comptime` as static computation under strict defaults rather than as the primary generic system. Phase 8 already covers generic programming v0 with concept-checked templates, so Phase 9 should start narrower and keep `comptime` focused on deterministic compile-time evaluation.

The Phase 9 design follows these PoC3 rules:

- `comptime` runs on the host compiler.
- `comptime` may eventually query target metadata supplied by the configured build target.
- `comptime` cannot call arbitrary extern C by default.
- `comptime` allocation, when added, uses compiler-owned temporary arenas rather than runtime storage.
- `comptime` side effects are limited and explicit.
- `comptime` execution must be deterministic by default.
- Host-visible side effects require explicit capabilities:
  - `read_fs`
  - `write_fs`
  - `env`
  - `network`
  - `time`
  - `random`
  - `process`

This preserves PoC3's intended boundary: compile-time code may become powerful, but it must not silently become an unreviewed build script.

## Source spelling vs compiler naming

The source keyword is concise:

```cpp
comptime
```

Compiler implementation identifiers should use full English `CompileTime*` names. Preferred examples:

- `CompileTimeValue`
- `CompileTimeEvaluator`
- `CompileTimeFunction`
- `CompileTimeCapability`
- `CompileTimeArena`
- `CompileTimeResult`

Rationale:

- source syntax should be concise;
- compiler code should remain readable and explicit;
- `comptime` is a keyword, not an excuse for abbreviated internal names.

Internal names such as `ComptimeValue` should be avoided unless existing compatibility later forces a narrow exception.

## P9 v0 scope

Phase 9 v0 begins with hermetic deterministic constant evaluation only.

Initial v0 scope:

- compile-time expression syntax:

  ```cpp
  int x = comptime 40 + 2;
  ```

- a compile-time constant evaluator over typed HIR expressions;
- `CompileTimeValue` support for:
  - `int`
  - `bool`
- no host side effects;
- no function calls initially unless a later milestone explicitly scopes them;
- no allocation;
- no reflection;
- no generated declarations;
- no target metadata initially;
- no filesystem, environment, network, time, random, or process access.

This scope is intentionally smaller than PoC3's eventual `comptime` direction. It proves the evaluator boundary before adding more forms.

## P9-M1 internal evaluator note

P9-M1 adds the first internal compile-time execution implementation without exposing source-level `comptime expr` yet. The source keyword remains reserved for the future user-facing surface, while compiler implementation names use full English `CompileTime*` forms.

Implemented P9-M1 scope:

- `CompileTimeValue` currently represents only `int` and `bool`;
- `CompileTimeEvaluator` evaluates a small typed HIR expression subset: int literals, bool literals, grouping, integer negation, boolean not, checked integer `+`, `-`, `*`, safe `/`, integer comparisons, int/bool equality, and boolean `&&`/`||`;
- unsupported expression forms, including calls, locals, parameters, structs, field access, pointers, `try`, `decide`, enum construction, and concept requirement calls, are rejected with internal compile-time errors;
- division by zero and checked integer overflow are compile-time evaluation errors;
- no compile-time function calls, static assertions, host capabilities, reflection, generated declarations, local-variable evaluation, or constant propagation are implemented.

The evaluator is hermetic and deterministic by construction: it has no access to filesystem, environment, network, time, random, process spawning, arbitrary extern calls, or other host effects. Source-level `comptime expr` remains future P9 work so MIR/backend continue to see ordinary checked runtime expressions in P9-M1.

## Compile-time expression syntax

Preferred source syntax:

```cpp
int x = comptime 40 + 2;
```

Rules:

- `comptime expr` explicitly requests compile-time evaluation.
- The expression must be evaluable by the current `CompileTimeEvaluator`.
- The result must be representable as a normal runtime value.
- The evaluated value is lowered as ordinary checked program structure before MIR/backend.
- If evaluation fails, the compiler reports a compile-time diagnostic at the `comptime` expression.
- `comptime` does not silence normal type checking.

Example:

```cpp
module Main;

int main() {
    int answer = comptime 40 + 2;
    return answer;
}
```

In this example the runtime program should behave as if the checked initializer is the ordinary constant `42`. MIR and backend code should not need to know that the value originally came from a `comptime` expression.

## Compile-time functions

Compile-time functions are future work, not P9 v0.

Future syntax direction:

```cpp
comptime int hashSeed(int value) {
    return value * 16777619;
}

int seed = comptime hashSeed(42);
```

Rules:

- A `comptime` function is eligible for compile-time execution.
- Marking a function `comptime` does not automatically execute it.
- The call site still uses `comptime expr` or another explicitly compile-time context.
- `comptime` functions still parse and typecheck normally.
- Unsupported operations inside a compile-time function are rejected when evaluating or checking for compile-time eligibility.
- Function calls are not part of P9-M1 unless specifically scoped later.

This keeps eligibility separate from execution and avoids implicit host-side work during ordinary runtime calls.

## Static assertions

Static assertions are future work after or alongside the expression evaluator.

Future syntax direction:

```cpp
static_assert(1 + 1 == 2);
```

Rules:

- Static assertion arguments are implicitly compile-time evaluated.
- The expression must evaluate to `bool`.
- Failed assertions produce compile-time diagnostics.
- v0 `static_assert` should not require layout queries.
- `sizeof` and `alignof` layout assertions remain later work.

A minimal `static_assert` can exist before full layout support because boolean expressions over literals and constants do not require final ABI layout.

## CompileTimeValue

Initial design:

```text
CompileTimeValue:
  int
  bool
  void maybe later
```

Future extensions may include:

- enum values;
- struct values;
- arrays/tables;
- strings/bytes;
- type values;
- reflection metadata;
- target metadata values.

Rules:

- `CompileTimeValue` is compiler-owned.
- `CompileTimeValue` is not runtime storage.
- Compile-time allocations, when added, use compiler-owned temporary arenas.
- `CompileTimeValue` must convert or lower into ordinary checked constants or generated structures before MIR/backend.

The value model should remain small at first. Adding new value categories should come with explicit lowering, diagnostics, and fixture coverage.

## Evaluator layer

The recommended first implementation is a typed HIR evaluator. It should not evaluate raw AST directly and should not depend on backend C execution.

Evaluation order:

```text
Source
  -> AST
  -> typed HIR
  -> CompileTimeEvaluator for supported comptime expressions
  -> concrete HIR / constants
  -> MIR
  -> backend
```

Rules:

- Evaluate after semantic checking.
- Evaluate before MIR lowering.
- MIR/backend should receive concrete values, not unresolved compile-time expressions.

Rationale:

- AST is too early and lacks type truth.
- MIR/backend are too late and should not define compile-time semantics.
- HIR already knows resolved types, symbols, and checked expressions.

The evaluator should therefore operate on the same semantically checked facts that ordinary executable checking uses, while still producing a representation that later lowering treats as normal checked program structure.

## Deterministic and hermetic default

Default compile-time execution must not access:

- filesystem;
- environment variables;
- network;
- wall-clock time;
- randomness;
- process spawning;
- arbitrary extern C;
- host locale/global state;
- hidden global caches except compiler-managed deterministic caches.

If the same source and build configuration are provided, default compile-time execution should produce the same result.

Compiler-managed caches are allowed only when they are semantically invisible and deterministic. They may speed evaluation up, but they must not change evaluation results based on host state, cache history, time, or process environment.

## Capabilities

Capability-bearing compile-time functions are future work. P9 v0 implements no capabilities and no host-visible side effects.

Syntax direction from PoC3:

```cpp
comptime(read_fs)
HardwareManifest loadHardwareManifest(StringView path) {
    Bytes data = comptimeIO::readFile(path);
    return parseManifest(data);
}
```

Rules:

- A function declares requested host-visible capabilities.
- The build manifest or build configuration must grant those capabilities.
- Without a grant, compile-time evaluation is rejected.
- Capabilities are not implemented in v0.
- Phase 9 may add a parse/storage scaffold before real host effects, but actual effects require explicit scoping.

Capability list:

- `read_fs`
- `write_fs`
- `env`
- `network`
- `time`
- `random`
- `process`

Danger categories:

- `read_fs` and `env` can leak host or build-machine state.
- `time` and `random` break reproducibility.
- `write_fs`, `network`, and `process` are high-risk side effects.
- All host-visible effects require explicit declaration and explicit build grant.

Capabilities should be auditable in source and in build configuration. A dependency should not gain new host powers merely because it was imported.

## Target metadata

Future direction:

```cpp
when target.arch == "x86_64" {
    // target-specific declaration or implementation
}
```

or:

```cpp
static_assert(target.pointerSize == 8);
```

P9 v0 does not implement target metadata.

Target metadata rules for later milestones:

- Target queries are deterministic compiler-provided values.
- Target metadata does not require a host capability if it comes from the configured build target.
- Target metadata should be explicit and typed.
- Target metadata should not inspect the host machine unless the configured target explicitly is the host target.

Configured target facts are build inputs, not ambient host observations. That distinction keeps cross-compilation and reproducible builds sane.

## Layout queries

Future syntax direction:

```cpp
static_assert(sizeof(Header) == 16);
static_assert(alignof(Header) == 8);
```

P9 v0 does not implement full layout queries.

Rules for staging:

- Phase 7 struct layout is backend-local, not a final ABI contract.
- `sizeof` and `alignof` should wait until the layout model is deliberate.
- `static_assert` can exist before layout queries.
- Layout diagnostics should cite the queried type and the layout rule that determined the result once layout queries exist.

This avoids accidentally freezing a backend convenience layout as Concept's language-level ABI.

## Generated declarations

Compile-time reflection and generated declarations may exist in the future, but they are not P9 v0.

Generated declarations require a deliberate design for:

- hygiene;
- source mapping;
- module ordering;
- diagnostics;
- incremental rebuild behavior;
- visibility and export rules.

Generated declarations must not sneak into early constant evaluation. P9 v0 should produce constants only, not new top-level declarations, new functions, new types, or hidden backend artifacts.

## Reflection

Future reflection should be:

- static;
- typed;
- explicit;
- compatible with freestanding builds;
- not runtime RTTI by default.

P9 v0 implements no reflection.

Reflection should expose compiler-checked metadata for compile-time use. It should not imply runtime reflection tables, hidden dynamic type information, or a managed runtime.

## Diagnostics

Proposed diagnostic names:

- `CompileTimeUnsupportedExpression`
- `CompileTimeTypeMismatch`
- `CompileTimeEvaluationFailed`
- `CompileTimeFunctionRequired`
- `CompileTimeNonDeterministicOperation`
- `CompileTimeCapabilityRequired`
- `CompileTimeCapabilityNotGranted`
- `StaticAssertFailed`
- `StaticAssertRequiresBool`
- `CompileTimeRecursionLimit`
- `CompileTimeFuelExhausted`

Exact `CON` diagnostic codes can be assigned during implementation.

Diagnostic principles:

- Report at the `comptime` expression or static assertion that triggered evaluation.
- Preserve the normal type-checking diagnostic if the expression is not well typed.
- Distinguish unsupported evaluator forms from ordinary type mismatch.
- Distinguish a missing capability declaration from a capability that was declared but not granted.
- Include recursion/fuel diagnostics before compile-time functions become expressive enough to loop indefinitely.

## Fixture strategy

Planned fixtures:

- valid `comptime` integer arithmetic expression;
- valid `comptime` boolean expression;
- valid runtime fixture returning `comptime 40 + 2`;
- invalid unsupported expression;
- invalid type mismatch;
- `static_assert` pass/fail once implemented;
- deterministic evaluator snapshots;
- future function-call evaluator fixtures;
- future capability-denied fixtures.

Early fixtures should prove the real path: parse, semantic checking, compile-time evaluation, concrete HIR/constants, MIR lowering, backend emission, and native execution where applicable. Snapshot-only evaluator tests are useful, but they should not replace an end-to-end runtime fixture for the motivating `comptime 40 + 2` case once the source surface exists.

## Proposed Phase 9 milestone ladder

```text
P9-M0  Compile-time execution design doc

P9-M1  Compile-time value model and constant evaluator v0
       - CompileTimeValue int/bool
       - typed HIR evaluator for literals/grouping/unary/binary
       - no source `comptime` expression yet if staging needs separation

P9-M2  `comptime expr` surface
       - parse `comptime expr`
       - evaluate supported expressions
       - lower result as ordinary constants
       - run fixtures

P9-M3  `static_assert` v0
       - implicitly compile-time-evaluate bool expression
       - pass/fail diagnostics
       - no layout queries yet

P9-M4  Compile-time function surface
       - parse `comptime` functions
       - straight-line return evaluator
       - calls only from compile-time contexts

P9-M5  Compile-time interpreter expansion
       - local variables
       - if/while if scoped
       - calls between compile-time functions
       - recursion/fuel guard

P9-M6  Target metadata and layout-query design/scaffold
       - target metadata as deterministic configured values
       - sizeof/alignof design or limited implementation

P9-M7  Capability model scaffold
       - parse/store `comptime(read_fs)` style capabilities
       - no actual host effects unless explicitly scoped
       - build manifest grant model documented

P9-M8  Runtime/backend fixture stabilization
       - MIR/backend see concrete evaluated constants only
       - no compile-time artifacts leak into backend

P9-M9  Phase 9 closeout
```

It is acceptable during implementation to combine P9-M1 and P9-M2 if that is the convergent path, but the design should keep evaluator and source surface conceptually separate.

## Non-goals

Phase 9 v0 does not implement:

- `comptime` as the generic system;
- arbitrary host code execution;
- arbitrary extern C calls;
- filesystem, environment, network, time, random, or process access by default;
- reflection;
- generated declarations;
- layout/ABI finalization;
- full `sizeof` or `alignof`;
- macro system;
- hygienic macros;
- build scripts;
- package registry behavior;
- ownership/drop;
- dynamic dispatch/interfaces.

These exclusions protect convergence. Adding any of them should require a later scoped milestone with tests, diagnostics, and a clear lowering story.

## Close criteria

Phase 9 should be considered successful if:

- `comptime` expression syntax exists;
- typed HIR evaluator can compute int/bool expressions;
- evaluated values lower into ordinary MIR/backend constants;
- `static_assert` exists or is clearly deferred;
- compile-time functions exist or are clearly staged;
- default compile-time execution is hermetic and deterministic;
- capability-bearing host effects remain gated/future;
- no generic-system responsibilities migrate from concepts/templates to `comptime`.

## Roadmap note

Current roadmap direction at Phase 9 start:

```text
Phase 7   closed: runtime structs and places
Phase 8   closed: concepts and templates over runtime values
Phase 9   planned/in design: compile-time execution v0
Phase 10  planned: ownership, move, Drop, MaybeUninit, and richer storage-state analysis
```

Phase 9 should be judged by whether it adds deterministic compile-time computation without weakening the Phase 8 boundary. Templates and concepts remain responsible for generic abstraction; `comptime` computes checked values and, later, explicitly gated static artifacts.

## P9-M2 source-level compile-time expressions

P9-M2 exposes the first source surface for explicit compile-time evaluation:

```cpp
int answer = comptime 40 + 2;
```

The source keyword is `comptime`; compiler implementation names continue to use full-English `CompileTime*` forms. The parser represents `comptime expr` as a compile-time prefix expression and the HIR preserves that operand until checking.

Implemented P9-M2 behavior:

- supported operands are the P9-M1 typed-HIR evaluator subset: integer literals, boolean literals, grouping, integer negation, boolean not, checked integer `+`, `-`, `*`, safe `/`, integer comparisons, int/bool equality, and boolean `&&`/`||`;
- the HIR checker checks the operand normally first, so `comptime` does not suppress type errors;
- after successful type checking, the HIR checker invokes `CompileTimeEvaluator` and records the resulting `CompileTimeValue` by HIR expression id;
- evaluated `int` and `bool` values lower as ordinary MIR constants;
- MIR validation and the C backend see only ordinary concrete values, not unresolved compile-time expression artifacts;
- unsupported source forms, division by zero, overflow, evaluator type mismatches, and fallback evaluation failures have stable diagnostics.

Still out of scope after P9-M2:

- compile-time function calls;
- local or parameter value evaluation;
- struct, enum, pointer, or aggregate `CompileTimeValue` forms;
- field access evaluation;
- static assertions;
- host capabilities or host-visible effects;
- target metadata queries such as `sizeof` or `alignof`;
- reflection or generated declarations;
- constant propagation beyond explicit `comptime expr`.
