# Numerics

Decimal and hexadecimal integer literals carry an exact non-negative magnitude plus lexical sign information until target typing. A range error names the literal and target width; valid hexadecimal syntax is never reclassified as a missing expression.

| Type | Range / meaning | C11 lowering |
|---|---|---|
| `int` | -2147483648 through 2147483647 | `int32_t` |
| `uint` | 0 through 4294967295, modulo 2^32 | `uint32_t` |
| `uint8`, `byte` | 0 through 255, modulo 2^8 | `uint8_t` |
| `uint64` | 0 through 18446744073709551615, modulo 2^64 | `uint64_t` |
| `usize`, `isize` | pointer-sized interop integers | `size_t`, `ptrdiff_t` |

`-2147483648` is one signed literal resolution, not a clamped positive literal followed by negation.

Signed `int` arithmetic is checked. Constant division by zero, invalid shifts, and constant signed overflow are compile-time errors. Binary literals and digit separators are not part of R7d3.

`%` is Euclidean for signed integers: `0 <= a % b < |b|` for nonzero `b`.
It differs from C's signed remainder for negative dividends. Signed and
unsigned modulo trap on a runtime zero divisor, and literal zero divisors are
rejected during checking. `INT32_MIN % -1` is zero without evaluating C's
undefined division case. Signed `/` is checked and truncates toward zero; it
is not the Euclidean quotient paired with `%` for negative dividends.
