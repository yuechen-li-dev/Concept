# Phase 11: First-class testing, attributes, and reasoned expectations

P11-M0 is a documentation-only milestone. It defines Concept's language-native
testing substrate, the first attribute model, and the expectation doctrine that
every test claim must explain why it matters.

P11-M0 does not implement compiler behavior. It does not change the parser,
lexer, HIR checker, MIR lowering, backend, or runner. Implementation begins only
after the design boundary is clear.

## Core doctrine

```text
Testing is a first-class language workflow, not just a repository convention.

Attributes discover and describe tests. Attributes must not become a hidden second semantics system.

Expectations are typed semantic claims.

Every Assert or Expect must include a reason explaining why the claim matters.

The reason is part of the test, not a comment.
```

Concept's test surface should fit the language's larger identity: cleaned-up
C++-lineage systems programming with visible intent, explicit danger, and no
ambient magic.

## Motivation

Concept needs first-class testing before adding heavier features such as state
machines. State-machine work will create more control-flow, storage-state, and
diagnostic interactions than the current language can comfortably validate with
repository convention alone. A language-native test substrate gives those future
features a place to express expected behavior in Concept terms.

The Stage 0 compiler itself will benefit from tests whose assertions carry
machine-readable intent. Today, a failing assertion can usually say what value
was wrong. It often cannot say why that value mattered to the language rule being
protected. Concept should make the semantic intent visible.

That is especially valuable for LLM-assisted development. When an assistant sees
a failing test with an explicit reason, it gets the invariant being guarded
instead of only a mismatched pair of values. The reason becomes executable
context: not control flow, but durable intent attached to a checked claim.

Classic assertion APIs often stop at the immediate mismatch:

```cpp
Assert.Equal(TokenKind.Identifier, token.kind);
```

That tells the runner the expected and actual values. It does not say whether the
test is guarding lexical classification, parser recovery, token interning,
source span preservation, or a regression in some broader flow.

Concept expectations should make intent explicit:

```cpp
Expect.That(
    token.kind,
    Is.EqualTo(TokenKind.Identifier),
    because: "alphabetic source text should lex as an identifier token"
);
```

The reason is not decorative. It is part of the test and must appear in
diagnostics.

## File model

```text
.concept or existing normal source extension: ordinary source
.con_test: Concept test source
```

A `.con_test` file is a normal Concept module loaded in test mode. It should not
create a second language, a preprocessor dialect, or a reflection-only testing
island.

Test files:

- are normal Concept modules;
- may import ordinary Concept modules;
- may import `Core.Test`;
- may contain attributed test functions;
- are discovered by test mode / test runner;
- use ordinary Concept type checking, ownership, move, Drop, unsafe, and effect
  rules.

The extension marks intent to the toolchain. It does not suspend normal language
semantics.

## Attribute model

Initial attribute syntax:

```cpp
[Fact]
void ParsesIdentifier() {
}

[Theory]
[InlineData(1, 2, 3)]
void Adds(int a, int b, int expected) {
}
```

Attributes attach to declarations. In Phase 11 v0, the important target is
function declarations, but the syntax should be declaration metadata rather than
a special test-only prefix.

Rules:

- attributes attach to declarations;
- attributes are metadata;
- attributes do not secretly rewrite core semantics;
- attribute arguments are initially simple literals only;
- there are no arbitrary attribute macros;
- there are no arbitrary compile-time expression attributes in Phase 11 v0;
- test attributes must be recognized by the compiler or test discovery layer;
- unknown attributes should be diagnosed in v0 unless the compiler later gains a
  deliberate preservation model for external tooling.

The v0 design chooses diagnosis for unknown attributes because Concept does not
yet have package metadata, namespaced tooling attributes, or an attribute
preservation contract. Preserving unknown attributes without such a contract
would invite users to believe they have semantics the compiler does not
understand.

Explicit doctrine:

```text
Attributes may label a spell.
They must not secretly cast it.
```

Initial recognized attributes:

- `[Fact]`
- `[Theory]`
- `[InlineData(...)]`

Deferred optional attribute:

- `[Skip(...)]`

`[Skip(...)]` is useful, but not required for v0. If added later, its argument
must also carry a reason, for the same reason assertions do: skipped tests are
semantic absences, not invisible convenience.

## Assert vs Expect

Concept should define both `Assert.*` and `Expect.*`, with a deliberate semantic
distinction:

```text
Assert.* = test preconditions, invariants, and setup sanity.
Expect.* = behavior expectations about the unit under test.
```

For Phase 11 v0, both may fail the current test immediately. The distinction is
still worth reserving now because a future runner may collect multiple
`Expect.*` failures in a single test while `Assert.*` aborts the current test at
once.

Example:

```cpp
Assert.True(
    tokens.length > 0,
    because: "lexer should produce at least one token before checking token content"
);

Expect.That(
    tokens[0].kind,
    Is.EqualTo(TokenKind.Identifier),
    because: "alphabetic source text should lex as an identifier token"
);
```

This distinction keeps setup sanity separate from the behavior being specified.
It also helps failure reports distinguish "the test could not proceed" from "the
unit under test violated the expected behavior."

## Mandatory reasons

Every test assertion and expectation must have a reason.

Canonical future syntax:

```cpp
Expect.That(actual, Is.EqualTo(expected), because: "reason");
Assert.True(condition, because: "reason");
```

If named arguments are not implemented when Phase 11 implementation begins, the
compiler and `Core.Test` module may initially accept positional reason arguments:

```cpp
Expect.That(actual, Is.EqualTo(expected), "reason");
Assert.True(condition, "reason");
```

The semantic slot is still named `because`. The positional form is an
implementation bridge, not the preferred long-term design.

Rules:

- reason must be present;
- reason must be a string literal or compile-time known string-like metadata in
  v0;
- reason must be non-empty;
- whitespace-only reason should be rejected;
- reason must appear in failure output;
- reason should be stored as test metadata if possible.

Suggested diagnostics:

```text
CON0170 TestExpectationRequiresReason
CON0171 TestReasonMustBeNonEmpty
```

Invalid examples:

```cpp
Expect.That(value, Is.EqualTo(3));
Assert.True(ok);
Expect.Equal(3, value);
Expect.That(value, Is.EqualTo(3), because: "");
```

These are invalid even if the expected and actual values are obvious. The
language should not accept tests that hide the reason their claims matter.

## Expectation model

Phase 11 has two expectation layers.

### Layer A: pragmatic v0 API

```cpp
Expect.Equal(expected, actual, because: "reason");
Expect.True(condition, because: "reason");
Expect.False(condition, because: "reason");

Assert.True(condition, because: "reason");
Assert.False(condition, because: "reason");
```

This layer is intentionally simple. It can be implemented before the relation
framework is complete and before formatting/display concepts are mature.

### Layer B: Concept-native relation API

```cpp
Expect.That(actual, Is.EqualTo(expected), because: "reason");
Expect.That(result, Is.Ok(), because: "reason");
Expect.That(diagnostics, Has.Diagnostic(CON0154), because: "reason");
```

The relation API is the canonical long-term design.

`Is.EqualTo(expected)` returns a relation value. Relation values are checked
against actual values through a concept-like constraint. Invalid relation/type
combinations should fail at compile/check time rather than falling into a late
runtime assertion failure.

Possible future concepts:

```cpp
concept TestRelation<R, T> {
    bool check(R relation, T actual);
};

concept TestDisplay<T> {
    String display(T value);
};
```

The exact syntax may change as Concept's concept, template, string, and module
surfaces mature. The design requirement is stable: expectations are typed
semantic claims, and relation applicability should be checked by the language.

Failure output can improve when `T` implements display/formatting concepts. If a
value cannot be displayed, the runner should still report the failed expression,
source location, and reason, then say that expected/actual display is
unavailable.

## Core.Test module shape

Conceptual module:

```cpp
module Core.Test;
```

Possible namespaces:

```cpp
namespace Assert { }
namespace Expect { }
namespace Is { }
namespace Has { }
```

Initial v0 functions:

```cpp
Expect.True(bool condition, because: string);
Expect.False(bool condition, because: string);
Expect.Equal<T>(T expected, T actual, because: string);

Assert.True(bool condition, because: string);
Assert.False(bool condition, because: string);
```

Long-term relation entry point:

```cpp
Expect.That<T, R>(T actual, R relation, because: string)
    where TestRelation<R, T>;
```

The exact syntax may differ based on the current Concept parser and language
support. In early implementation, `Core.Test` may be compiler-recognized rather
than a complete ordinary library module if the standard library surface is not
ready. That recognition should remain narrow and auditable.

## Test discovery and runner

Test mode discovers `.con_test` files and loads them as normal Concept modules.
The test runner collects attributed test functions.

Rules:

- `[Fact]` functions are test cases;
- `[Fact]` functions must be zero-arg;
- `[Fact]` functions must return `void` or an accepted test result type,
  depending on current language support;
- `[Theory]` functions may have parameters;
- `[Theory]` functions require one or more `[InlineData]` attributes;
- inline data arity must match the test function parameter count;
- inline data literal types must be assignable to the corresponding parameter
  types;
- test names are fully qualified by module and function;
- no generated runtime reflection is required.

Failure output should include:

- test name;
- source location;
- failed assertion/expectation;
- reason;
- expected/actual values if available;
- fallback message if values cannot be displayed.

Example output:

```text
FAILED Core.Tests.Lexer.ParsesIdentifier

Expectation failed:
  Expect.That(token.kind, Is.EqualTo(TokenKind.Identifier))

Because:
  alphabetic source text should lex as an identifier token

Expected:
  TokenKind.Identifier

Actual:
  TokenKind.Number
```

The reason is printed as a first-class section, not buried as a trailing
annotation.

## InlineData and Theory

Design v0:

```cpp
[Theory]
[InlineData(1, 2, 3)]
[InlineData(10, 20, 30)]
void Adds(int a, int b, int expected) {
    Expect.Equal(
        expected,
        add(a, b),
        because: "add should return the arithmetic sum of its operands"
    );
}
```

Rules:

- `[Theory]` functions may have parameters;
- each `[InlineData]` creates one test case;
- inline data values are simple literals in v0;
- arity mismatch is a diagnostic;
- type mismatch is a diagnostic;
- no generated runtime reflection is required.

Suggested diagnostics:

```text
CON0172 InvalidTestAttribute
CON0173 FactRequiresZeroArgFunction
CON0174 TheoryRequiresInlineData
CON0175 InlineDataArityMismatch
CON0176 InlineDataTypeMismatch
```

Each generated test case should have a stable display name derived from the fully
qualified function name and the inline data row index. Argument values may be
included in display names later when formatting is mature, but they are not a v0
requirement.

## Interaction with existing phases

Phase 8 concepts/templates allow the future relation-constrained
`Expect.That<T, R>` API. The long-term testing design should lean on concepts
for relation applicability rather than using untyped runtime reflection.

Phase 9 compile-time evaluation can later help validate attribute data, produce
compile-time known reason strings, and check richer metadata. Phase 11 v0 should
not require arbitrary compile-time expressions inside attributes.

Phase 10 ownership, move, Drop, and `ManualInit<T>` semantics apply inside tests
exactly like ordinary code. Tests must not get ownership loopholes. A `.con_test`
file must not be able to copy non-Copy values, use moved values, skip Drop
cleanup, or treat manual initialization as safe merely because it is test code.

ManualInit, Drop, move semantics, unsafe boundaries, and must-use behavior must
behave identically in `.con_test` files and ordinary source files.

## Non-goals for Phase 11

Phase 11 does not include:

- full property-based testing;
- fuzzing;
- snapshot testing;
- mocks;
- fixtures/lifecycle hooks such as `beforeEach` / `afterEach`;
- async tests;
- state-machine tests;
- arbitrary attribute macros;
- arbitrary compile-time attribute expressions;
- runtime reflection;
- package-level test discovery beyond current project/test harness
  capabilities;
- rich generic relation framework if it blocks v0 test runner;
- full `TestDisplay<T>` formatting implementation if strings/formatting are not
  ready.

The v0 work should make direct tests real before it grows a test ecosystem.

## Proposed Phase 11 milestone ladder

```text
P11-M0  Design doc: first-class testing, attributes, reasoned expectations
P11-M1  Attribute syntax and declaration metadata
P11-M2  .con_test file recognition and test discovery scaffold
P11-M3  Mandatory reason metadata / string-literal reason validation
P11-M4  [Fact] zero-arg test runner v0
P11-M5  Assert.* and Expect.* v0 with mandatory reasons
P11-M6  Expect.That relation API scaffold
P11-M7  [Theory] + [InlineData] v0
P11-M8  Test reporting/filtering/examples/fixtures
P11-M9  Closeout
```

Implementation may adjust ordering if the dependency graph demands it, but the
design dependency should remain clear:

```text
attributes and discovery first,
mandatory reasons before broad assertion APIs,
relation constraints after simple runner value exists.
```

## P11-M1 implementation status

P11-M1 implements declaration attributes as inert parser/AST/HIR metadata.
Attributes may appear immediately before supported declarations and attach to the
following declaration. The initial recognized spellings are:

- `[Fact]`
- `[Theory]`
- `[InlineData(...)]`

Unknown attributes are diagnosed with `CON0172 InvalidAttribute`. This follows
the v0 policy that Concept should not preserve unknown metadata until it has a
deliberate preservation contract for external tooling.

Attribute arguments are syntax-only metadata and are deliberately narrow in this
milestone. The accepted argument forms are:

- integer literals;
- boolean literals;
- string literals.

Arbitrary expressions are rejected in attributes. Calls, binary expressions,
field access, `comptime` expressions, and other executable syntax are not
accepted as attribute arguments in Phase 11 v0.

Implemented boundaries:

- attributes parse and lower as declaration metadata;
- metadata is preserved through HIR;
- attributes do not alter type checking;
- there is no `.con_test` discovery yet;
- there is no test runner yet;
- `[Fact]`, `[Theory]`, and `[InlineData]` have no validation behavior yet;
- `Assert.*` and `Expect.*` are not implemented yet;
- mandatory reason validation is not implemented yet;
- the relation API is not implemented yet;
- there are no attribute macros and no runtime reflection.

## Close criteria

P11-M0 is successful if:

- `.con_test` files are defined as ordinary Concept test modules;
- attributes are documented as declaration metadata, not hidden semantics;
- `[Fact]`, `[Theory]`, and `[InlineData]` are defined as the initial recognized
  test attributes;
- `Assert.*` and `Expect.*` have a reserved semantic distinction;
- every assertion/expectation requires a non-empty reason;
- the semantic slot is named `because`;
- v0 pragmatic APIs and long-term relation APIs are both documented;
- `Core.Test` module shape is sketched without overcommitting parser syntax;
- discovery, runner, theory, inline data, and failure output rules are
  documented;
- interaction with Phase 8, Phase 9, and Phase 10 is explicit;
- non-goals prevent Phase 11 from becoming a macro, reflection, fixture, or
  property-testing project.
