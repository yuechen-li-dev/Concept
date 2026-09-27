# R8b explicit numeric conversions

`as` is valid when the source value and target type uniquely determine the
intended conversion. If another semantic choice is required, that choice must
be named. `x as T` and `static_cast<T>(x)` lower to the same `CastExpr`, semantic
operation, MIR kind, effects, and C path. `static_cast` evaluates runtime
values; it does not invoke `comptime`.

## Conversion rules

| Source → target | Rule |
| --- | --- |
| Integer → floating | Explicit numeric conversion; large values may round |
| Floating → floating | Explicit target-representation conversion, including narrowing |
| Integer → integer | Checked range conversion; an unrepresentable runtime value traps with `integer cast out of range` before any C cast |
| Floating → integer | `as` rejects with `FLOAT_TO_INT_ROUNDING_REQUIRED` |

The integer cast retains expression type `T`. A trap is the existing checked
failure convention used for ordinary integer arithmetic and bounds errors;
the recoverable float-to-integer operations below return `Result`. No integer
cast wraps or saturates. Literal assignment remains separately checked for
representability and retains its existing ergonomics.

`TruncTo<T>(x)`, `FloorTo<T>(x)`, `CeilTo<T>(x)`, and `RoundTo<T>(x)` require an
unqualified floating source and an integer target. They return
`Result<T, NumericCastError>`, where `NumericCastError` has `NotFinite` and
`OutOfRange`. `?` uses ordinary `Result` propagation. `RoundTo` chooses the
nearest integer with ties to even. The chosen rounding occurs before the
range check. NaN and both infinities return `NotFinite`. Negative zero yields
integer zero. The finite rounded value must lie in the mathematical target
range; negative fractional inputs may therefore succeed for an unsigned
target if the chosen rule rounds them to zero.

The C backend evaluates the operand once. It checks `isfinite`, rounds with
`trunc`, `floor`, or `ceil`, or uses an explicit floor/fraction/parity rule for
ties to even. Exclusive powers-of-two upper bounds avoid double rounding of
`INT64_MAX` and `UINT64_MAX` into an unsafe C cast. Only a proven in-range
rounded value reaches the C integer conversion.

Quantity casts may change numeric representation only when the normalized unit
identity, including scale, is identical. Casts cannot attach, remove, rescale,
or change a dimension. `interpret`, scaled unit conversion, and semantic
attachment remain outside R8b. `reinterpret_cast`, `const_cast`, and
`dynamic_cast` have no semantics in Concept.

## Facts and implementation boundary

`compiler.ExactConversion<Source, Target>()` proves type-wide numeric exactness
for identical representations, range-containing integer conversions, integer
values that fit a floating target's precision and finite range, and floating
widening. This describes numeric value exactness, not a claim about NaN payload
bits. Inexact type-wide conversions are Disproven. Alias spellings canonicalize
before the check. `Assert.Concept<ExactConversion>(Source, Target, reason)` and
declared concept requirements use the same analysis.

MIR retains `integer_checked_range`, `integer_to_float`,
`float_representation`, and `float_round_to_integer` operations. Their
`NoAllocation` evidence is checked in the focused tests. `float` and `double`
generated conversion code is qualified under pedantic C11 with GCC and Clang.
`half` remains the GCC extension lane identified by R8a; portable strict-C11
binary16 is not claimed. Bounded comptime evaluation supports numeric `as`
with `float` and `double` targets; its preexisting value model does not admit
`half` as a comptime target.

## Subsequent ledger

- R8c: multiplicative unit algebra and scaled unit conversion.
- R8d: `interpret value as T` semantic attachment at external boundaries.
- R8e: project policy concepts, `must_use`/discard, lint/fmt, broader first-contact diagnostics.
- R8f: resume differentiator goldens.
