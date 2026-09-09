# Quantities and units

Status: EVT1 R6g compiler substrate

Concept separates numeric representation from measurement. `int`, `uint`,
`uint8`, `usize`, `isize`, and `float` select representation; `byte`, `bit`,
`m`, `s`, `K`, and `Hz` select units. Canonical examples are
`usize<byte>`, `usize<bit>`, `float<m>`, `float<s>`, `float<K>`, and
`float<Hz>`.

`byte` and `bit` are information units. They are not the canonical spelling of
an eight-bit scalar; `uint8` has that role. Legacy unqualified `byte` remains a
temporary source-compatible alias for `uint8` so existing storage does not
silently change meaning. New storage extents use `usize<byte>`.

The compiler stores a normalized fixed exponent vector and exact rational unit
scale. The coherent base set follows Oct: length, mass, time, current,
temperature, amount, luminous intensity, plus Concept's systems Information
dimension. `Hz` normalizes to `s^-1`; one `byte` is exactly eight `bit`. Unit
metadata is erased before C11 lowering.

Addition, subtraction, remainder, and comparison require identical units.
Multiplication and division combine normalized dimensions; cancellation yields
the ordinary dimensionless representation. Integral `Convert<Unit>` is exact:
a conversion that would discard a remainder traps rather than truncates.

`SizeOf<T>()`, `AlignOf<T>()`, `LayoutSize<T>()`, `LayoutAlign<T>()`, and
`LayoutOffset<T>()` use `usize<byte>` and the existing layout authority. A
compatibility bridge permits an R6g byte quantity through an old naked-`usize`
result signature, but it does not permit unrelated quantity arithmetic.

The `usize` operator surface includes `+`, `-`, `*`, `/`, `%`, comparisons,
`&`, `|`, `^`, `<<`, and `>>`. It follows existing native integer overflow
semantics; R6g adds no allocator-only arithmetic policy.

Signed modulo is normatively Euclidean: the remainder is nonnegative and
smaller than the divisor magnitude. R6g exposes `%` for unsigned
representations (where C and Euclidean results coincide) and reserves signed
`%` until its lowering has explicit zero and minimum-value handling. C's
signed remainder is not language semantics.

## Oct reconciliation

Directly retained are normalized exponent identity, coherent
multiply/divide/power rules, signed exponents, compile-time compatibility, and
`Hz = s^-1`. Concept adapts spelling to C++-recognizable
`representation<unit>` and adds exact rational scale for bit/byte conversion.
Oct's `px`/`ui`, Celsius sugar, runtime registries, and broad conversion catalog
are not ported.
