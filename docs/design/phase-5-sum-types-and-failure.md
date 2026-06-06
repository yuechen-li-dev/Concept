# Phase 5 sum types and explicit failure

Phase 5 starts Concept's first core systems-language semantics on top of the closed Phase 4 MIR path. P5-M0 is documentation only: no compiler code, parser changes, HIR changes, MIR changes, backend changes, fixture runner changes, runtime layout implementation, `try`, `must_use`, or `discard` implementation is part of this milestone.

## Thesis

```text
Phase 5 teaches Concept how to fail correctly.

Payload enums make sum types executable.
must_use and discard make ignored values explicit.
Result-shaped enums give Concept concrete checked failure before generics.
try lowers explicit failure propagation into MIR.
```

The authoritative executable path entering Phase 5 is:

```text
Concept source
  -> parseSource
  -> semantic collection / HIR lowering
  -> HIR executable checker
  -> HIR-to-MIR lowering
  -> MIR validation
  -> MIR-backed C backend
  -> zig cc
  -> native executable / exit-code fixtures
```

Phase 5 broadens that path only where needed for payload enum values and checked explicit failure. It must preserve the Phase 2 executable subset, the Phase 3 semantic spine, and the Phase 4 MIR-backed run path while adding the first executable nominal sum type behavior.

## Goals

Phase 5 should make Concept's parsed and semantically collected enum declarations executable runtime values and use them to model explicit failure without waiting for generics.

Required goals:

- Make enum values executable runtime values.
- Implement payload enum layout v0.
- Implement enum variant constructors.
- Implement `match` over enum variants.
- Implement payload binding in `match` arms.
- Implement `must_use`.
- Implement `discard`.
- Define a concrete Result-shaped enum convention.
- Implement `try` propagation over Result-shaped enums.
- Add HIR support for enum constructors, variant patterns, payload bindings, `must_use`, `discard`, Result-shaped metadata, and `try`.
- Add MIR support for enum values, construction, tag extraction, payload extraction, variant switching, discard lowering, and `try` lowering.
- Add C backend support for executable enum layout, constructors, tag switches, payload extraction, and deterministic backend-owned names.
- Preserve Phase 2, Phase 3, and Phase 4 fixtures.

Phase 5 is successful only if the real executable path materially improves: a Concept program can produce, inspect, unwrap, explicitly ignore, and propagate concrete failure values through the MIR-backed backend.

## Non-goals

Phase 5 deliberately does not include:

- Generics or templates.
- Generic `Result<T, E>`.
- Concept solving.
- `Drop<T>`.
- Move or copy checking.
- Borrow checking.
- Unsafe or raw pointer semantics.
- Storage live/dead modeling.
- A heap allocator or runtime library.
- Arrays, strings, chars, or floats runtime expansion unless explicitly required by the enum/failure work.
- Imports or multi-file modules.
- Payload enum ABI guarantees beyond backend v0.
- Optimization.
- LLVM, bare-metal, or self-hosting targets.

Phase 5 should not become a general resource-management, generic-programming, module-system, or production-ABI phase. Its scope is checked, explicit failure over concrete payload enums.

## Enum runtime representation v0

Enum declarations already exist in the Phase 1 parser surface and are collected by the Phase 3 semantic spine. Before Phase 5, those declarations are not executable nominal runtime values in the authoritative backend path. Phase 5 makes supported enum values executable.

Representation rules:

- Every enum value has a tag identifying the active variant.
- Payload enum variants may carry fields.
- Field order is declaration order.
- Backend v0 may lower supported executable enums to a C `struct` containing a tag and a `union` of variant payload structs.
- Only `int` and `bool` payload fields are required initially unless existing `TypeStore` and backend support make more types straightforward.
- Zero-payload variants need a tag but do not need payload storage.
- The layout is nominal, backend-local, internal, and not a final ABI guarantee.

Example Concept:

```cpp
enum ParseResult {
    Ok(int value),
    Err(int code),
};
```

Example C-ish backend v0 layout:

```c
typedef struct {
    int tag;
    union {
        struct { int value; } Ok;
        struct { int code; } Err;
    } payload;
} cpt_enum_ParseResult;
```

The exact emitted names should follow the deterministic `cpt_*` backend naming policy, but Phase 5 should keep that policy backend-local rather than treating it as source-level ABI.

### P5-M1 enum runtime representation checkpoint

P5-M1 implements the first backend runtime representation for supported enum declarations on the MIR-backed C path. The C backend emits deterministic, backend-owned `typedef struct` layouts for non-empty enums collected in the semantic module before function prototypes and bodies. Each layout contains an integer `tag`; enums with payload variants additionally contain a `payload` union with one struct entry per payload-carrying variant. Tag-only variants do not allocate payload union entries.

This representation is backend v0 only: it is internal to the current C backend, is not ABI-stable, and is not a source-level naming contract. Payload field support is intentionally limited to fields whose semantic types are `int` or `bool`, both rendered as C `int`. Empty enums and enum payloads containing structs, other enums, `void`, or invalid types are rejected by the backend with the existing unsupported-C-backend-type diagnostic if layout emission is attempted.

P5-M1 does not add source-level enum construction, `EnumName::Variant`, matching over enum variants, payload binding, Result-shaped conventions, `try`, `must_use`, or `discard`; those remain future Phase 5 milestones.

## Enum constructor syntax

Recommended constructor syntax:

```cpp
EnumName::Variant
EnumName::Variant(args)
```

Examples:

```cpp
Token end() {
    return Token::End;
}

ParseResult ok(int value) {
    return ParseResult::Ok(value);
}
```

Rules:

- A constructor must be qualified by the enum name.
- The enum name must resolve to an enum declaration.
- The variant must exist in that enum.
- Constructor arity must match the variant payload field count.
- Constructor argument types must match the variant payload field types.
- Zero-payload variants may be constructed without parentheses.
- No unqualified variants are supported in v0.
- No generic enum constructors are supported in Phase 5.

HIR should represent construction as an enum constructor expression rather than as a general function call. MIR should lower construction to an enum-valued rvalue or equivalent assignment sequence that sets the tag and payload fields deterministically.

## Match over enum variants

Phase 4 `match` supports checked integer and boolean patterns. Phase 5 should add enum variant patterns.

Example without payload binding:

```cpp
match (result) {
    ParseResult::Ok => return 1;
    ParseResult::Err => return 0;
}
```

With payload binding later in Phase 5:

```cpp
match (result) {
    ParseResult::Ok(value) => return value;
    ParseResult::Err(code) => return code;
}
```

Rules:

- The scrutinee enum type determines the allowed variants.
- The variant pattern enum must match the scrutinee enum.
- The variant must exist.
- Duplicate concrete variant arms are rejected.
- Wildcard `_` remains allowed.
- Exhaustiveness may be deferred, or implemented for enum matches if straightforward.
- Payload binding comes after basic variant matching in the milestone ladder.

MIR lowering should not preserve source-level `match` as a structured construct. It should extract or inspect the enum tag and branch to one block per relevant arm, with payload extraction placed in the selected arm when payload bindings exist.

## Payload binding

Payload binding lets a variant pattern introduce names for the active variant's payload fields.

Example:

```cpp
match (result) {
    ParseResult::Ok(value) => return value;
    ParseResult::Err(code) => return 0;
}
```

Rules:

- Binding names are introduced only inside the arm body.
- Binding types come from the matched variant payload field types.
- Binding arity must match payload arity.
- Duplicate binding names in one pattern are rejected.
- Bindings do not leak outside the arm.
- V0 payload binding may support positional binding only.
- Named binding can be future work.

A binding should behave like a scoped immutable local value for the arm unless and until Concept defines richer binding mutability rules. MIR lowering can materialize payload extractions into arm-local temporaries or locals before lowering the arm body.

## `must_use`

Proposed v0 syntax:

```cpp
must_use enum ParseResult {
    Ok(int value),
    Err(int code),
};
```

Meaning:

- Expressions producing a `must_use` type may not be ignored as expression statements.
- Returning a `must_use` value counts as use.
- Assigning or initializing a local with a `must_use` value counts as use.
- Matching a `must_use` value counts as use.
- Applying `try` to a `must_use` value counts as use.
- `discard expr;` explicitly evaluates and ignores the value.

Examples:

```cpp
parseInt(text);          // error if ParseResult is must_use
discard parseInt(text);  // allowed
```

V0 should attach `must_use` to enum/type declarations only. Function-level `must_use` can be future work.

The diagnostic should make the programmer choose between consuming the value or writing an explicit discard. This keeps ignored failures visible in code review and searchable in large systems codebases.

## `discard`

Syntax:

```cpp
discard expression;
```

Meaning:

- Explicitly evaluates and ignores the expression.
- Suppresses the ignored-value diagnostic for `must_use` values.
- Is visible and searchable source syntax.
- Is not a function call.
- Lowers as expression evaluation followed by value discard.

`discard` should not introduce destructor, lifetime, or storage semantics in Phase 5. It only makes an ignored evaluated value explicit.

## Result-shaped enum convention

Phase 5 does not implement generics/templates, so it must not introduce a generic `Result<T, E>`. Instead, Phase 5 defines a concrete convention that later templates and concepts can generalize.

An enum is Result-shaped when:

- It has a success variant named `Ok`.
- It has an error variant named `Err`.
- `Ok` has exactly one payload field.
- `Err` has exactly one payload field.
- It may be marked `must_use`.

Example:

```cpp
must_use enum ParseIntResult {
    Ok(int value),
    Err(int code),
};
```

Result-shaped metadata should be derived from the enum declaration during semantic analysis. Later, templates and concepts can generalize this convention into `Result<T, E>` and a `Tryable` concept. Phase 5 should keep the convention intentionally concrete.

## `try` propagation v0

Syntax:

```cpp
int value = try parseInt(text);
```

Requirements:

- The operand expression type must be a Result-shaped enum.
- The enclosing function return type must be a compatible Result-shaped enum.
- The `Ok` payload type becomes the expression type of `try`.
- The operand `Err` payload type must be compatible with the enclosing function return enum's `Err` payload type.
- On `Err`, control returns the enclosing enum's `Err(error)`.
- On `Ok`, control unwraps the success payload and continues.

Example:

```cpp
ParseIntResult addParsed(int aText, int bText) {
    int a = try parseInt(aText);
    int b = try parseInt(bText);
    return ParseIntResult::Ok(a + b);
}
```

Lowering sketch:

```text
tmp = parseInt(text)
switch tmp.tag:
  Ok:
    value = tmp.payload.Ok.value
    continue
  Err:
    return ParseIntResult::Err(tmp.payload.Err.error)
```

The enclosing return enum does not have to be the same nominal enum as the operand if the Phase 5 compatibility rule permits rebuilding the enclosing `Err` from the operand error payload. V0 may choose the stricter rule that the nominal Result-shaped enum must match if that is needed for convergence, but the design target is explicit error-payload compatibility plus construction of the enclosing enum's `Err` variant.

## HIR, MIR, and C backend impact

HIR additions:

- Enum constructor expression.
- Enum variant pattern.
- Payload binding names.
- Discard statement.
- Try expression.
- `must_use` flag on type declarations.
- Result-shaped enum metadata.

MIR additions:

- Enum value locals and temporaries.
- Enum constructor rvalue.
- Enum tag extraction.
- Enum payload extraction.
- Variant switch lowering.
- Try lowering to MIR blocks.
- Discard expression lowering.

C backend additions:

- Emit enum C layout for executable enum types.
- Emit constructor assignments.
- Emit tag switches.
- Emit payload extraction.
- Preserve deterministic `cpt_*` naming policy.

The implementation should keep the architecture layered: semantic analysis decides which enum, variant, payload field, and Result-shaped convention is meant; MIR represents explicit control flow and data movement; the C backend emits only already-validated enum operations.

## Fixture strategy

Phase 5 should add fixtures incrementally and preserve the Phase 2, Phase 3, and Phase 4 fixture corpus.

Planned fixture families:

- Parse/check fixtures for enum constructors.
- Check fixtures for constructor arity errors.
- Check fixtures for constructor type errors.
- MIR snapshots for enum construction.
- MIR snapshots for enum variant match lowering.
- Run fixtures for simple enum values.
- Run fixtures for `match` over enum variants.
- Run fixtures for payload binding.
- Run fixtures for `try` propagation.
- Invalid fixtures for ignored `must_use` values.
- Valid fixtures using `discard`.

Example `.conception` run fixture:

```text
# name: enum constructor returns ok
# phase: run
# expect: pass

=== source ===
module Main;

enum ParseResult {
    Ok(int value),
    Err(int code),
};

int main() {
    ParseResult result = ParseResult::Ok(7);
    match (result) {
        ParseResult::Ok(value) => return value;
        ParseResult::Err(code) => return code;
    }
}

=== run ===
exit_code: 7
```

Fixture growth should follow the milestone ladder. Each new runtime feature should have at least one real run fixture once it reaches the backend path, and each new semantic restriction should have at least one failing check fixture.

## Milestone ladder

```text
P5-M0  Sum types and explicit failure design doc
P5-M1  Enum runtime representation v0
P5-M2  Enum variant constructors
P5-M3  Match over enum variants
P5-M4  Payload binding in match
P5-M5  must_use and discard
P5-M6  Result-shaped enum convention
P5-M7  try propagation v0
P5-M8  Backend/runtime stabilization for enum/result/try
P5-M9  Phase 5 closeout
```

## Close criteria

Phase 5 is complete when:

- Enum values are executable runtime values.
- Enum constructors work for supported variants and payloads.
- `match` over enum variants works.
- Payload binding works for supported payloads.
- `must_use` ignored-value diagnostics exist.
- `discard` works.
- The Result-shaped enum convention exists.
- `try` propagation works for concrete Result-shaped enums.
- The HIR/MIR/C backend path supports these features.
- Representative parse/check/MIR/run fixtures exist.
- Existing Phase 2, Phase 3, and Phase 4 fixtures still pass.
- No generics, templates, `Drop<T>`, move checking, borrow checking, unsafe semantics, or raw pointer semantics are required.

The convergence requirement for Phase 5 is not that syntax exists or that enums have a backend shape in isolation. The motivating capability must work through the real path: Concept programs should be able to construct concrete result values, force callers to consume or explicitly discard them, and propagate failures through MIR-backed code generation.

## P5-M0 close criteria

P5-M0 is complete when:

- This design document exists.
- Phase 5 goals, non-goals, enum representation v0, constructor syntax, enum matching, payload binding, `must_use`, `discard`, Result-shaped enums, `try`, HIR/MIR/C backend impact, fixture strategy, milestone ladder, and close criteria are documented.
- No compiler implementation code has been added.
- `zig build test` still passes.


## P5-M2 enum variant constructors

P5-M2 adds source-level construction for enum values using qualified variant
constructor syntax: `EnumName::Variant`, `EnumName::Variant()`, and
`EnumName::Variant(args)`. The Stage 0 path resolves constructors during
semantic lowering, checks variant existence and payload arity, verifies payload
argument types in the HIR executable checker, lowers constructors through MIR,
and emits backend-owned C assignments for enum tags and payload fields.

Variant matching remains future P5-M3 work, and payload binding remains future
P5-M4 work. Result-shaped conventions, `try`, `must_use`, and discard checking
remain out of scope for this milestone.

### P5-M3 match over enum variants checkpoint

P5-M3 adds executable matching over resolved enum variant patterns in the authoritative MIR-backed path. A `match` arm may now use a qualified, tag-only pattern such as `Status::Ok` or `ParseResult::Err`, and `_` remains available as a wildcard/default arm. Pattern enum names and variant names are simple identifiers in this checkpoint; unqualified variants, dotted names, generic names, and payload-binding forms are deliberately outside this milestone.

Payload-bearing enum values can be matched by active variant, but their payloads are not bound or extracted yet. A pattern like `ParseResult::Ok(value)` is rejected as unsupported payload binding and remains the P5-M4 target.

Semantic lowering resolves each enum variant pattern to the matched enum and variant IDs. HIR checking allows enum variant patterns only when the scrutinee has the same enum type, rejects int/bool literal patterns against enum scrutinees, rejects variant patterns against int/bool scrutinees, and rejects duplicate concrete enum variant arms. Exhaustiveness checking is still not required in P5-M3.

MIR lowering implements enum matches by assigning an `EnumTag` rvalue to an integer temporary and reusing `SwitchInt` with deterministic variant indexes (`0` for the first declared variant, `1` for the second, and so on). The C backend emits tag extraction through the backend-local `.tag` field from the P5-M1 enum layout and then emits the existing integer `switch` form.

### P5-M4 enum match payload binding checkpoint

P5-M4 extends enum variant match arms from tag-only patterns to positional payload binding patterns. A match arm may now write `EnumName::Variant(name)` or `EnumName::Variant(left, right)` to bind the active variant payload fields as immutable arm-local values, and those bindings are visible only while lowering and checking that arm body. Payload variants may still be matched without bindings (`EnumName::Variant`) to inspect only the tag and ignore the payload.

Bindings are positional only: named payload binding, nested/destructuring patterns, rest patterns, and expression patterns in binding position are not part of this milestone. Binding arity is either zero or exactly the variant payload field count, duplicate binding names in one pattern are rejected, and binding names follow the existing no-shadowing rule for visible locals and parameters.

The MIR path now represents compiler-generated enum payload extraction after the enum tag switch, and the MIR-backed C backend emits extraction through the Phase 5 layout form `.payload.<variant>.<field>`. This is not user field access syntax and does not define a final ABI.

P5-M4 still does not implement Result-shaped enum conventions, `try`, `must_use`, `discard`, generic `Result<T, E>`, templates, concepts, drop/move/borrow checking, unsafe/raw pointers, user field access syntax, struct runtime layout, or final ABI mangling.
