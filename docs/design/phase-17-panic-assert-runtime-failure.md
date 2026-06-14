# Phase 17: Panic, assertions, and runtime failure reporting

## Status

P17-M4 is now implemented for `assert(condition, "reason")` MIR/backend lowering. P17-M2 remains closed for `panic(reason)` MIR/backend lowering:

- `panic("reason");` parses in statement position.
- The reason string literal and source span are preserved in AST/HIR.
- HIR/debug output shows dedicated panic statements.
- Missing reasons, wrong arity, and non-string reasons are rejected.
- Expression-position `panic(...)` remains unsupported and is diagnosed.
- HIR panic statements lower to dedicated MIR panic statements preserving reason text and reason span.
- MIR debug output exposes `Panic "reason"`, and MIR validation accepts statement-position panic with no value result.
- The C backend emits a backend-owned `static void cpt_panic(const char* reason)` helper exactly once per generated C unit when panic is used.
- Hosted C v0 includes `<stdio.h>` and `<stdlib.h>` for this backend-owned helper, prints `panic: %s\n` to `stderr`, and terminates with deterministic `exit(101)`.
- `panic("reason");` lowers to `cpt_panic("reason");` with backend C string-literal escaping and no test-runner dependency.
- `assert(condition, "reason");` parses in statement position as a compiler-known runtime invariant assertion.
- Assert AST/HIR preserve the condition expression, reason string literal text, full statement span, condition span, and reason span.
- Assert HIR/debug output exposes dedicated assert statements and their reason.
- Assert conditions must type-check as `bool`; missing/wrong reason forms and non-string-literal reasons are rejected.
- Expression-position `assert(...)` remains unsupported and is diagnosed.
- HIR assert statements lower to dedicated MIR assert statements preserving the lowered bool condition operand, reason text, condition span, and reason span.
- MIR debug output exposes `Assert "reason"`, MIR storage analysis reads the condition operand, and MIR validation requires a bool condition plus present reason metadata.
- The C backend lowers `assert(condition, "reason")` to `if (!(condition)) { cpt_panic("reason"); }`, reusing backend C string escaping.
- The backend-owned `cpt_panic` helper is shared by panic and assert and is emitted exactly once per generated C unit when either or both are used.
- Runtime fixtures prove assert-true continues, assert-false exits 101, and condition expressions continue normally. Reason output remains pinned through generated C/backend assertions while runtime stderr matching is unsupported.
- Named `because:` syntax, expression-position assert, `never`, empty/whitespace reason hardening, Core.Test `Assert.True` alignment, test-runner panic catching, exceptions, unwinding, and stack traces remain deferred.

## Core doctrine

```text
Runtime failure is explicit.

panic(reason) is unconditional runtime failure.

assert(condition, reason) is the runtime invariant form of Assert.True(condition, reason).

Test assertions and runtime assertions share one failure doctrine.

Reasons are mandatory.

A failure without a reason is not a useful failure.

Runtime failure lowering must be inspectable in HIR/MIR/C output.

No exceptions in Phase 17.

No stack unwinding in Phase 17.

No hidden allocation for panic/assert in Phase 17.

No test-runner dependency for ordinary runtime assertions.
```

## 1. Motivation

Concept already has a first-class testing surface from Phase 11. `.con_test`,
`[Fact]`, `[Theory]`, `[InlineData]`, `Assert.*`, `Expect.*`, and
`Expect.That(...)` all established a key rule: assertion and expectation reasons
are not comments. They are part of the assertion.

Runtime code now needs the same discipline. Concept already has runtime traps in
ad-hoc places, including bounds-style failures and Phase 13 machine operations
such as `Result(machine)` before completion. Future DragonGod-enabling work will
add nested machines, runtime `transition match`, runtime `transition decide`,
and `yield` / suspend / resume. Those features must not each invent their own
trap mechanism.

DragonGod-style machine systems especially need clear failure reasons. Invalid
machine states, impossible transitions, reading a result too early, stepping a
completed machine, future nested-machine result misuse, and future yield misuse
are all runtime failures whose reports should explain the violated invariant.

Classic assertion APIs often fail with a generic message such as `assertion
failed`. That says what happened, but not why the claim mattered. Concept already
rejected that model for tests in Phase 11 by requiring reasons. Phase 17 extends
that doctrine to ordinary runtime code.

## 2. Conceptual layering

Phase 17 defines three conceptual layers:

```text
panic(reason)
assert(condition, reason)
Core.Test.Assert / Core.Test.Expect
```

`panic(reason)` is unconditional runtime failure.

`assert(condition, reason)` is runtime invariant checking. It is semantically the
runtime version of:

```cpp
Assert.True(condition, reason);
```

`Assert.True(condition, reason)` in tests should conceptually use the same
failure data model as runtime `assert`. `Assert.Equal(...)` and
`Expect.That(...)` remain richer test-facing assertion and expectation APIs;
they can report expected/actual values and relation-specific details.

Runtime `assert` is not `Expect.True`. It is closer to `Assert.True` because it
expresses an invariant, precondition, or internal truth the program requires in
order to proceed. `Expect.True` expresses a test expectation whose failure may be
collected by a runner.

## 3. Source syntax

Canonical future syntax uses a named reason slot:

```cpp
assert(index < len, because: "index must be within bounds before reading");
panic(because: "unreachable machine state reached");
```

If named arguments are not implemented when Phase 17 behavior begins, v0 may use
staged positional syntax:

```cpp
assert(index < len, "index must be within bounds before reading");
panic("unreachable machine state reached");
```

Rules:

- `reason` is mandatory.
- `reason` must be a string literal or compile-time known string-like metadata
  in v0.
- `reason` must be non-empty.
- Whitespace-only `reason` is invalid.
- `reason` is semantically named `because` even if positional syntax is used in
  v0.
- `assert(condition, reason)` condition must be `bool`.
- `panic(reason)` never returns.

Invalid examples:

```cpp
assert(ok);
assert(ok, "");
assert(ok, "   ");
assert(123, "numeric condition is not bool");
panic();
panic("");

`panic("")` and whitespace-only reasons remain planned for P17-M5 diagnostic
hardening rather than enforced in P17-M1.
```

## 4. `panic(reason)`

`panic(reason)` is unconditional runtime failure. The reason is required and must
appear in generated failure output when the target environment supports output.
Conceptually, `panic` returns `never`. Until a `never` type exists, the compiler
may accept it only in statement position and may give it special control-flow
treatment where explicitly implemented.

Simple v0 use:

```cpp
int unreachable_case() {
    panic("unreachable token kind reached");
}
```

Staged behavior:

- If functions must return a value and control-flow analysis understands
  non-returning calls, `panic` as the final statement of a non-void function may
  satisfy the return requirement.
- If that control-flow treatment is not ready, v0 should restrict `panic` to
  statement positions where the surrounding control flow remains otherwise
  valid.
- Expression-position use, including `return panic("reason");`, is deferred
  until `never` coercion is designed.

## 5. `assert(condition, reason)`

`assert(condition, reason)` is a runtime invariant assertion:

- The condition type must be `bool`.
- If the condition is true, execution continues.
- If the condition is false, the assertion lowers to `panic(reason)` or an
  equivalent shared runtime failure path.
- The reason is required.
- It produces no value.
- It is statement-position only in v0.

Example:

```cpp
int read(bool ok, int value) {
    assert(ok, "caller must prove value is readable before read");
    return value;
}
```

## 6. Shared failure data model

Runtime failures use this conceptual record:

```text
RuntimeFailure {
    kind: panic | assertion
    reason: string
    location: source location
    expression: optional source expression
}
```

Test failures use this conceptual record:

```text
TestFailure {
    kind: assert | expect | panic
    reason: string
    location: source location
    expected: optional display value
    actual: optional display value
}
```

Runtime and test failures need not have identical runtime representation. They
should share doctrine and reporting fields: reason, location, failure kind, and
optional expression/details. `Assert.Equal` can include expected/actual values.
Runtime `assert` and test `Assert.True` can include the condition expression if
that is feasible without overcomplicating v0 lowering.

## 7. Runtime behavior

Phase 17 v0 runtime behavior:

- Failed `assert` calls the panic path.
- `panic` terminates the program.
- There is no recovery.
- There are no exceptions.
- There is no stack unwinding.
- There is no stack trace requirement in v0.
- There is no allocation requirement.
- The backend C path emits a helper or inline failure path.
- The failure exit code should be deterministic and documented.
- Output should include the reason if the environment supports `stderr` or
  `stdout`.
- Bare-metal and no-stdio profiles may route panic differently later.

Recommended hosted C v0 behavior:

```c
void cpt_panic(const char* reason);
```

The helper prints the reason to `stderr` if available and aborts or exits with a
stable nonzero code. Tests can assert failure behavior through expected-failure
fixtures.

If the current C backend avoids libc includes, the implementation should choose
one minimal helper strategy and document it in the implementation milestone:

- declare minimal external C functions explicitly;
- use `abort()` with a prototype; or
- emit a simple trap helper that does not require formatting.

The exact helper body is intentionally deferred from M0.

## 8. HIR/MIR representation

HIR should preserve panic and assert explicitly, for example:

```text
HirExpr/HirStmt panic(reason)
HirStmt assert(condition, reason)
```

MIR may either preserve dedicated failure operations:

```text
MirTerminator/Panic(reason)
MirStmt Assert(condition, reason)
```

or lower assertion to an explicit branch plus panic:

```text
if !condition -> panic(reason)
```

Requirements:

- Panic/assert are visible in HIR/MIR debug output.
- The reason literal is preserved.
- The source span is preserved.
- The assert condition is type checked as `bool`.
- Panic/assert can be optimized later, but must remain inspectable in v0.
- Failed bounds checks and machine traps can later lower to the same panic
  representation.

## 9. Backend C lowering

Possible C helper shape:

```c
static void cpt_panic(const char* reason) {
    /* implementation-defined hosted behavior */
}
```

`panic("reason");` lowers to:

```c
cpt_panic("reason");
```

`assert(condition, "reason");` lowers to:

```c
if (!(condition)) {
    cpt_panic("reason");
}
```

Rules:

- The helper is emitted once per generated C unit.
- The reason string is emitted as a C string literal.
- No heap allocation.
- No exceptions.
- No unwinding.
- No dependency on the test runner.
- No hidden control-flow magic.

If using `fprintf(stderr, ...)` or `abort()`, the implementation milestone must
document required includes or prototypes and how that interacts with the Phase
15 no-hidden-header doctrine. A minimal helper path is acceptable for v0; rich
reporting can be deferred.

## 10. Interaction with Phase 11 testing

`assert(condition, reason)` should align with `Assert.True(condition, reason)`.
A `.con_test` `Assert.True` may later lower or route through shared assertion
machinery. Tests may catch and report assertion failures as test failures rather
than aborting the entire runner if the runner supports it.

`Assert.Equal` and `Expect.That` remain richer test APIs and are not replaced by
runtime `assert`. Mandatory reason diagnostics should align with Phase 11 reason
diagnostics where possible.

Ordinary runtime code must not depend on the test runner. The test runner may
depend on shared failure concepts.

## 11. Interaction with Phase 13 machines

Future machine failure sites include:

- `Result(machine)` before completion;
- stepping a completed machine if policy forbids it;
- impossible state id;
- invalid transition target;
- future `yield` misuse;
- future nested machine result misuse.

Phase 17 provides the shared panic/assert substrate for those sites. M0 does not
migrate existing machine traps; it only defines the target model.

## 12. Interaction with Phase 15/16 C ABI and modules

Panic/assert helpers are backend runtime helpers, not user `extern "C"`
declarations. Helper names should remain backend-owned, for example
`cpt_panic`.

Multi-module compilation emits the helper once per generated C unit. Exported C
functions may call `assert` or `panic` internally, but `panic` does not become C
ABI exception behavior and does not introduce unwinding across the ABI boundary.

## 13. Diagnostics planning

Suggested diagnostics:

```text
CON0280 PanicRequiresReason
CON0281 AssertRequiresReason
CON0282 FailureReasonMustBeNonEmpty
CON0283 AssertConditionMustBeBool
CON0284 PanicExpressionUseUnsupported
CON0285 AssertExpressionUseUnsupported
```

If Phase 11 reason-required diagnostics can be generalized, the implementation
should reuse them rather than duplicating behavior under new codes. Reason
validation should mirror Phase 11:

- missing reason;
- empty reason;
- whitespace-only reason.

## 14. Non-goals for Phase 17 v0

Phase 17 v0 explicitly defers:

- exceptions;
- stack unwinding;
- recoverable panic;
- panic payloads beyond string reason;
- formatted panic messages;
- interpolated strings;
- stack traces;
- source expression stringification if hard;
- test runner catching arbitrary runtime panics if hard;
- panic hooks;
- profile-specific panic handlers;
- no_std/bare-metal panic ABI;
- `never` type if not ready;
- `defer` / cleanup-on-panic;
- Drop during panic unwinding;
- `Result` replacement;
- broad runtime standard library.

## 15. Milestone plan

```text
P17-M0  Design doc: panic, assertions, and runtime failure reporting
P17-M1  panic(reason) parser/AST/HIR scaffold
P17-M2  panic(reason) MIR/backend lowering (implemented: MIR panic statement, C cpt_panic helper, exit 101)
P17-M3  assert(condition, reason) parser/AST/HIR scaffold (implemented: statement syntax, AST/HIR assert, bool condition check, reason diagnostics)
P17-M4  assert(condition, reason) MIR/backend lowering (implemented: MIR assert statement, storage/validation, backend if-not guard calling shared cpt_panic, run/backend fixtures)
P17-M5  reason validation and diagnostics hardening
P17-M6  align Core.Test Assert.True with shared assertion doctrine
P17-M7  migrate existing runtime trap sites to shared panic path
P17-M8  examples/fixtures/hardening
P17-M9  Closeout
```
