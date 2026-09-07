# EVT1 failure model

Status: R4c canonical

Concept has one explicit failure calculus and no hidden exception subsystem.

## Taxonomy

`Option<T>` means that no value may be present and absence is expected. It has
`Some(T)` and `None` and carries no error information. `Result<T,E>` means a
computation may fail recoverably and preserves a typed `E` in `Ok(T)` or
`Error(E)`. Panic means execution cannot meaningfully continue under the
current invariant assumptions. A runtime assertion is an invariant check;
`static_assert` is its compile-time counterpart.

Absence is not failure. Recoverable failure is not panic. Panic is not ordinary
control flow.

## Explicit handling and sugar

Exhaustive `match` over the carrier is semantic ground truth. Postfix `?`
branches on the tag, yields the success payload, and returns the same residual
channel. Result propagation requires the exact same error type; Option and
Result never implicitly convert. Postfix `!` yields success or explicitly
escalates at its source span to terminal panic.

`try` creates a lexical boundary around Result propagation. Each `except`
names one concrete error type. The compiler routes a statically known error to
its exact arm with ordinary local control flow. An unmatched error may leave
the block only through an enclosing Result with the identical error type.
There is no throw, exception object, error boxing, runtime type test,
cross-frame catch, or stack unwinding. Option remains handled by match, `?`, or
`!` in R4c because an absence-handler spelling is not yet justified.

## Assertions and panic

`assert(condition, reason)` is runtime `Assert.True`; false invokes the tiny
terminal helper and true continues. `static_assert(condition, reason)` is
`comptime Assert.True` evaluated by the existing bounded evaluator and emits
no runtime code. Neither construct creates a testing framework.

The C bootstrap panic helper prints a deterministic reason and source
line/column, then aborts. It allocates nothing and unwinds nothing. Existing
deterministic Drop behavior is not redefined as panic cleanup.

## Systems interactions

Option and Result are semantic payload enums. Payload copyability, explicit
move, exactly-once transfer, deterministic Drop, immovable embedding, and
ref-struct provenance follow the ordinary R2-R4b laws. Matching, propagation,
and unroll cannot silently copy owned payloads or promote a short lifetime.

## Future direction

A later milestone may define:

```concept
concept ErrorConvertible<From, To> {
    requires To ConvertError(From error);
}
```

Only an explicit satisfied witness could then permit cross-error propagation.
R4c contains no Rust-style residual/From/Into machinery. Other separate future
directions are no-panic concepts, ABI-safe error concepts, and realtime-safe
failure handling. All retain the law that no hidden exception runtime or
unwinding appears.
