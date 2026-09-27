# R8 floating representations

R8 thaws the EVT1 scalar model without changing quantity syntax. A floating
type has an encoding and, independently, may have a unit dimension. The
canonical spellings are `half` (IEEE binary16), `float` (IEEE binary32), and
`double` (IEEE binary64). `float16`, `float32`, and `float64` are explicit
compatibility spellings of those same types. The parser canonicalizes them
before semantic analysis, generic identity, artifact construction, and C
lowering. Unsuffixed floating literals continue to denote `float` in an
unconstrained expression; a literal initialized into a declared floating
target is rendered at that representation.

Unit arguments and scalar representation are orthogonal. `float<K>` means a
binary32 quantity measured in kelvin. It does not mean a 32-bit floating type
parameterized by K. `half<m>`, `float<m>`, and `double<m>` retain the same unit
axis. Numeric-looking `float<32>` and `float<64>` are invalid unit expressions;
they never select precision.

`Type.FloatRepresentation` carries the encoding in the semantic type and
serialized module payload. `FloatRepresentationInfo` exposes storage bits,
exponent bits, and fraction bits to compiler analyses. Named concepts can use
`compiler.Floating<T>()`, `compiler.BinaryFloat<T>()`,
`compiler.ScalarBits<T>(N)`, `compiler.FloatExponentBits<T>(N)`, and
`compiler.FloatMantissaBits<T>(N)`. These are exact compile-time facts, not
runtime numeric traits. The ordinary `Assert.Concept` proof path accepts
builtin scalar type subjects. `SizeOf` follows the representation sizes;
alignment follows the qualified target's layout rule.

## Current backend boundary

`float` emits C `float`; `double` emits C `double`. Both use strict C11.
`half` emits `_Float16`, a 2-byte binary16 extension on the probed Clang and
GCC toolchains. GCC 15 rejects this type with `-std=c11 -pedantic-errors`.
The installed Clang 22 Windows/MSVC target accepts the syntax but fails to
link arithmetic because `__extendhfsf2` and `__truncsfhf2` are unavailable;
GCC 15 compiles and runs the arithmetic probe.
Consequently the current implementation cannot claim strict C11 support for
binary16. It also does not claim `CAbiValue<half>`; ABI admission needs native
evidence. This is the specific remaining R8a blocker. A portable binary16
storage and arithmetic lowering would need a bounded, tested implementation;
silently using C `float` would violate the 2-byte promise.

`bfloat16` was probed with Clang 22 and GCC 15. GCC linked the small arithmetic
probe, while Clang's Windows target failed to link `__truncsfbf2`. Neither
toolchain recognized the probed `float8e4m3` and `float8e5m2` C spellings.
Those formats and all float4 encodings remain declared/deferred. Ambiguous
`float8` and `float4` are intentionally absent. Their current source uses
`FLOAT_REPRESENTATION_UNSUPPORTED` rather than a fabricated scalar layout.

## Post-R8a ledger

- R8b: explicit casts and representation conversion.
- R8c: multiplicative unit algebra and scaled unit conversion.
- R8d: semantic unit attachment replacing user-facing `AssumeQuantity`.
- R8e: resume differentiator goldens.
- Affine temperature units and exact mixed-precision policy are deferred.
- Portable binary16 and unsupported bfloat16/float8/float4 lowering need
  separate backend evidence before a strict-C11 claim.
