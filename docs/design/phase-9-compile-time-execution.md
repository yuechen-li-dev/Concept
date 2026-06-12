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

Compile-time functions were deferred during P9-M1/P9-M3 and enter the source surface in P9-M4.

Future syntax direction:

```cpp
comptime int hashSeed(int value) {
    return value * 16777619;
}

int seed = comptime hashSeed(42);
```

General rules:

- A `comptime` function is eligible for compile-time execution.
- Marking a function `comptime` does not automatically execute it.
- The call site still uses `comptime expr` or another explicitly compile-time context.
- `comptime` functions still parse and typecheck normally.
- Unsupported operations inside a compile-time function are rejected when evaluating or checking for compile-time eligibility.
- Function calls are not part of P9-M1; P9-M4 scopes straight-line calls explicitly.

This keeps eligibility separate from execution and avoids implicit host-side work during ordinary runtime calls.

## Static assertions

P9-M3 adds source-level static assertions:

```cpp
static_assert(1 + 1 == 2);
```

Rules:

- `static_assert(expr);` implicitly evaluates `expr` at compile time through `CompileTimeEvaluator`.
- The expression must evaluate to `CompileTimeValue.bool`.
- `true` succeeds and produces no MIR or backend artifact.
- `false` reports `StaticAssertFailed` (`CON0125`) with the stable message `static assertion failed`.
- A successfully evaluated non-bool value reports `StaticAssertRequiresBool` (`CON0126`).
- Unsupported expression forms use the existing compile-time evaluation diagnostics, such as `CompileTimeUnsupportedExpression`.
- `static_assert` does not require spelling `comptime` at the call site.
- No custom user message is supported yet; `static_assert(condition, "message")` is deferred.
- No layout queries, `sizeof`, `alignof`, compile-time functions, host capabilities, target metadata, reflection, or generated declarations are added by P9-M3.

A minimal `static_assert` exists before full layout support because boolean expressions over literals and checked compile-time arithmetic do not require final ABI layout.

## P9-M4 compile-time function surface

P9-M4 adds the first source-level compile-time function surface:

```cpp
comptime int add(int a, int b) {
    return a + b;
}

int main() {
    return comptime add(20, 22);
}
```

Rules implemented in P9-M4:

- A function marked with source keyword `comptime` is eligible for compile-time execution.
- Marking a function `comptime` does not execute it automatically.
- Calls still require an explicit compile-time context: `comptime expr` or `static_assert(expr);`.
- Runtime calls to `comptime` functions are rejected in P9-M4. The function is treated as compile-time-only and is skipped by MIR/backend lowering.
- Calls to non-`comptime` functions from compile-time contexts are rejected with `CompileTimeFunctionRequired`.
- Supported signatures are limited to `int` and `bool` parameters and `int` or `bool` return values.
- Supported bodies are limited to a single braced block containing one `return expr;`.
- The return expression may use literals, parameters, grouping, supported unary/binary operators, and calls to other compile-time functions.
- Parameter binding is supported for `int` and `bool` values.
- Nested compile-time function calls are supported with a small recursion guard; recursion is not intentionally supported.
- `static_assert` may call compile-time functions without spelling `comptime` at the call site because the assertion expression is already an implicit compile-time context.

Still out of scope after P9-M4:

- local-variable evaluation inside compile-time functions;
- `if`, `while`, `match`, `try`, `decide`, unsafe blocks, field access, struct literals, pointers, and aggregate values in the evaluator;
- general compile-time interpretation;
- host capabilities or host-visible effects;
- reflection, generated declarations, layout queries, `sizeof`, `alignof`, target metadata, or arbitrary extern C calls.

P9-M4 keeps compile-time functions erased from runtime output unless a later phase deliberately introduces dual-use functions with explicit lowering and tests.

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

## P9-M5 compile-time locals and `if`

P9-M5 extends compile-time function execution from straight-line single-return
bodies to a deliberately small statement interpreter for checked HIR function
bodies. Compile-time functions now execute with an explicit `CompileTimeFrame`
containing parameter bindings, local bindings, and a scope stack for block-local
lifetime during evaluation.

Implemented P9-M5 behavior:

- compile-time functions may declare initialized `int` and `bool` locals;
- local initializers are evaluated by `CompileTimeEvaluator` and must match the
  local HIR type;
- parameter references and local references resolve through the active
  `CompileTimeFrame`;
- simple assignment to local variables is supported for `int` and `bool` values
  when the assigned value matches the local type;
- `if` / `else` statements are supported when the condition evaluates to `bool`;
- only the selected branch is executed by the compile-time evaluator;
- branches may return or fall through to following statements;
- multiple return statements are supported through ordinary control-flow branch
  selection;
- compile-time-only functions remain skipped by MIR lowering and backend output;
- explicit `comptime f(...)` expressions still lower to ordinary `int` or `bool`
  literals after successful evaluation;
- static assertions continue to evaluate in compile-time context and emit no MIR
  or backend artifact.

P9-M5 remains intentionally narrow:

- loops are still unsupported in compile-time function evaluation;
- this is not a general interpreter for arbitrary statements;
- host effects, allocation, filesystem/process/environment/network access, and
  target metadata queries are still unavailable;
- `CompileTimeValue` still has only `int` and `bool` variants;
- struct, enum, pointer, array, string, and aggregate compile-time values remain
  unsupported;
- calls from compile-time contexts to runtime functions are still rejected;
- runtime calls to `comptime` functions are still rejected;
- reflection and generated declarations remain future work.

## P9-M6 compile-time `while` and evaluation fuel

P9-M6 adds bounded compile-time loop execution for the existing scalar
compile-time function evaluator. The supported source form is ordinary
`while (condition) { ... }` inside a function marked `comptime`; there is no
new loop-specific source keyword.

Implemented P9-M6 behavior:

- compile-time functions may execute `while` statements during
  `CompileTimeEvaluator` function-body evaluation;
- each loop condition is evaluated as a `CompileTimeValue` and must be
  `CompileTimeValue.bool`;
- a false condition exits the loop and execution continues with following
  statements;
- a true condition executes only the loop body for that iteration;
- assignments to locals declared outside the loop persist across iterations;
- locals declared inside the loop body are scoped by the body block and do not
  leak after the iteration or after the loop;
- `return` inside a loop immediately exits the compile-time function;
- missing-return diagnostics remain path-sensitive to the actually executed
  compile-time path;
- calls from compile-time loop bodies to runtime functions are still rejected;
- runtime calls to compile-time-only functions remain rejected;
- compile-time-only functions still emit no MIR or backend artifact, and an
  explicit `comptime f(...)` loop result lowers as an ordinary scalar literal.

P9-M6 also gives compile-time evaluation a deterministic step budget. Each
compile-time evaluation starts with a fixed `CompileTimeBudget` default of
`100_000` steps. Statement execution consumes fuel, loop iterations consume
fuel, and compile-time function calls consume fuel. Exhaustion reports the
stable `CompileTimeFuelExhausted` diagnostic instead of allowing an infinite or
excessively expensive compile-time loop to hang the compiler. The existing
compile-time recursion-depth guard remains separate from this step budget.

P9-M6 remains intentionally narrow:

- `for` loops are not implemented;
- this is not a general runtime interpreter;
- `CompileTimeValue` still supports only `int` and `bool`;
- arrays, strings, structs, enums, pointers, aggregate values, raw-pointer
  dereference, address-of evaluation, and field access remain unsupported;
- `match`, `try`, `decide`, unsafe blocks, host effects, reflection, generated
  declarations, target metadata, and layout queries remain unsupported;
- the step budget is fixed internally and is not user-configurable yet.

## P9-M7 compile-time capability annotation scaffold

P9-M7 adds source syntax and metadata for future host-visible compile-time
capability requirements. A compile-time function may remain hermetic with the
existing modifier form:

```cpp
comptime int answer() {
    return 42;
}
```

or it may declare requested capabilities with a parenthesized identifier list:

```cpp
comptime(read_fs, env)
int loadTable() {
    return 1;
}
```

The known capability names are:

- `read_fs`
- `write_fs`
- `env`
- `network`
- `time`
- `random`
- `process`

Implemented P9-M7 behavior:

- the parser accepts `comptime(capability, ...)` only as a top-level compile-time
  function modifier;
- capability names are parsed as capability-list identifiers, not as ordinary
  runtime identifiers;
- AST and HIR function metadata store the requested capability list;
- the HIR checker rejects unknown capability names;
- the HIR checker rejects duplicate capabilities;
- capability-bearing compile-time functions are declaration-valid and may be
  typechecked as functions;
- evaluating a capability-bearing compile-time function through `comptime expr`
  or `static_assert` is denied with `CompileTimeCapabilityNotGranted`;
- plain `comptime` functions without a capability list continue to evaluate with
  the existing deterministic, hermetic scalar evaluator;
- compile-time-only functions, including capability-bearing ones, remain absent
  from MIR and backend output.

P9-M7 intentionally grants no capabilities. It adds no filesystem, environment,
network, time, random, process, or other host APIs. The future direction is a
build manifest or build configuration grant model where dependency capability
requirements are visible and auditable. Until that exists, every non-empty
compile-time capability set is denied, and default compile-time execution remains
hermetic and deterministic.

## P9-M8 deterministic target metadata scaffold

P9-M8 adds a small compiler-owned target metadata scaffold for compile-time
execution. The source-level metadata root is `target`, and the supported v0
facts are:

- `target.pointerSize` -> `int`
- `target.isLittleEndian` -> `bool`
- `target.isBigEndian` -> `bool`

These facts are configured compiler target facts, not ambient host inspection.
The Stage 0 compiler does not yet have a complete build-target abstraction, so
P9-M8 uses a deterministic default target object as a scaffold:

- pointer size: `8`
- little endian: `true`
- big endian: `false`

This default matches the current Stage 0 C-backend assumptions and must be
replaced by explicit build-target configuration when that layer exists. The
compile-time evaluator still has no filesystem, environment, network, time,
random, process spawning, host calls, or arbitrary extern-call access. Reading
configured target metadata does not require `read_fs`, `env`, `time`, or any
other compile-time capability because it is compiler-owned input, not a host
side effect.

`target` is contextual metadata root syntax in target metadata expressions, not
a runtime object. P9-M8 accepts target metadata in explicit compile-time
contexts:

```cpp
static_assert(target.pointerSize == 8);

int main() {
    return comptime target.pointerSize;
}
```

and target metadata lowers only after evaluation, as ordinary `int` or `bool`
constants. `static_assert(target.pointerSize == 8)` emits no MIR/backend artifact,
and `comptime target.pointerSize` lowers as the same integer constant as any
other successful `comptime` expression. Runtime use such as
`return target.pointerSize;` is rejected for now; no runtime target object is
introduced.

Unknown metadata fields are rejected with a stable compile-time target-field
diagnostic. String target facts are intentionally not present in P9-M8, so
queries such as `target.arch == "x86_64"` remain unsupported until
`CompileTimeValue` grows an intentional string representation and the language
surface chooses string target metadata.

### Layout-query boundary

P9-M8 deliberately does not implement `sizeof`, `alignof`, language-level struct
layout queries, ABI finalization, reflection, generated declarations, or
target-specific conditional declarations. `target.pointerSize` is a configured
target fact about the target machine model; it is not a general layout query.

`sizeof(T)` and `alignof(T)` require a deliberate Concept language-level layout
and ABI model. Phase 7 runtime structs currently lower through backend-local
layout choices, and those choices must not be frozen accidentally as the Concept
ABI. Future layout queries must specify how Concept types map to size,
alignment, padding, representation attributes, and target ABI rules before they
can be exposed to compile-time evaluation.

## P9-M9 closeout status

Phase 9 v0 is closed as a small, deterministic compile-time execution slice.
It intentionally stabilizes the evaluator boundary rather than expanding into the
larger PoC3 compile-time vision.

Implemented Phase 9 v0 support:

- `comptime expr` source expressions;
- `static_assert(expr);` with no custom message argument;
- compile-time function declarations and calls from explicit compile-time
  contexts;
- `CompileTimeValue` values for `int` and `bool`;
- `int` and `bool` parameters and locals in compile-time functions;
- local assignment in compile-time functions;
- scoped blocks plus `if` / `else`;
- `while` loops guarded by deterministic fuel/step accounting;
- nested compile-time calls with recursion/fuel protection;
- capability annotations as declaration metadata only;
- deterministic target metadata for:
  - `target.pointerSize`;
  - `target.isLittleEndian`;
  - `target.isBigEndian`.

The closeout invariant is:

```text
MIR/backend receive ordinary concrete values; unresolved compile-time expressions must not leak into MIR/backend.
```

Successful `comptime expr` evaluation lowers as ordinary constants,
`static_assert` emits no MIR/backend artifact, compile-time-only functions are
skipped by MIR/backend lowering, capability metadata is not emitted, and
`target` is not a runtime object. The hermetic and deterministic default remains
intact. The evaluator has no filesystem, environment, network, time, random,
process, arbitrary host-call, or other host-effect access.

Capability annotations are declaration metadata in Phase 9. They are not grants.
They do not enable host operations. All non-empty capability sets are denied
during evaluation until a future build-grant model exists.

Target metadata is deterministic configured compiler data. The current
implementation uses a deterministic scaffold default. It must not inspect the
ambient host machine at evaluation time. `target.arch` and string target facts
are deferred because `CompileTimeValue` has no string value yet.

Intentionally deferred beyond Phase 9 v0:

- host effects and capability grants;
- filesystem, environment, network, time, random, process, or arbitrary host APIs;
- reflection and generated declarations;
- strings, arrays, structs, enums, and pointers as `CompileTimeValue`;
- field access in compile-time evaluation;
- raw pointer/address-of/deref evaluation;
- a runtime `target` object;
- target string facts such as `target.arch`;
- `sizeof` and `alignof`;
- language-level ABI/layout queries;
- compile-time allocation;
- `static_assert(expr, "message")`;
- compile-time `for` loops;
- ownership, move, Drop, MaybeUninit, and richer storage-state analysis.

`sizeof` and `alignof` remain deferred. Phase 7 struct C layout is a backend
implementation detail, not yet Concept's language-level ABI. Layout queries
require a deliberate layout model and must not be inferred accidentally from
generated C.

Ownership, move, Drop, MaybeUninit, and storage-state analysis are Phase 10
work. Compile-time execution v0 does not implement ownership semantics.
