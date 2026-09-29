# R8c multiplicative unit algebra

Quantity types have three independent parts: scalar representation (`half`,
`float`, `double`, or an integer representation), an eight-component exponent
vector, and an exact positive rational scale relative to `m`, `kg`, `s`, `A`,
`K`, `mol`, `cd`, and `bit`. For example, `float<mm>` is binary32, length
dimension, and scale `1/1000`. `float<kg*m/s^2>` and `float<N>` share both
dimension and scale. `float<MPa>` and `float<Pa>` share dimension but differ in
scale by exactly one million.

Products add exponents and multiply rational scales; quotients subtract
exponents and divide scales. Integer powers use exponentiation by squaring.
Composed rational factors are reduced before multiplication. The current
artifact schema stores scale numerator and denominator as machine integers;
an expression exceeding that exact range rejects with
`QUANTITY_SCALE_OVERFLOW` instead of wrapping. Algebraic cancellation to a
dimensionless result returns an ordinary scalar. When the cancelled units have
different scales, floating arithmetic applies the exact rational scale in
generated C before rounding to the scalar representation. Integer expressions
that would need fractional dimensionless scaling reject.

## Standard vocabulary

Standard names units and prefixes commonly written in engineering and
scientific source code. It does not expose every theoretically legal SI
prefixed spelling. The catalog is one explicit table in `quantity.go`:

| Family | Spellings |
| --- | --- |
| Length | `nm um mm cm m km` |
| Mass | `mg g kg` |
| Time | `ns us ms s` |
| Current | `A mA` |
| Other bases | `K mol cd bit byte` |
| Frequency | `Hz kHz MHz GHz` |
| Force | `N mN kN` |
| Pressure | `Pa kPa MPa GPa` |
| Power | `W mW kW MW` |
| Voltage | `V mV kV` |

`dm`, `dam`, `hm`, and `daN` are intentionally absent. A literal such as
`1dm` diagnoses the unknown unit and suggests `1e-1m`. This is a catalog
policy, not a parser limitation. Domain units may be designed later without
turning Standard into a universal prefix generator.

## Source operations

Adjacent suffixes form quantity literals: `1m` has the existing integer
literal representation; `1.0m`, `1e-1m`, and `1e3m` use the existing default
floating representation (`float`). A space separates tokens, so `1 m` is not
a quantity literal. Scientific notation without a suffix remains a floating
literal. There are no new precision suffixes.

`value as float<m>` permits an explicit same-dimension scale conversion from
`float<mm>` and combines scale conversion with representation conversion from
`float<mm>` to `double<m>`. The conceptual order is value in the target unit,
then target scalar representation. Generated C computes the scale with binary64
intermediates and then casts to the target representation; its intermediate
operations follow C11 binary64 rounding. The exact ratio remains visible in
MIR. An integer source may scale into a floating
target; a scaled cast to an integer target rejects because arbitrary values
may not be integral. Existing same-unit integer casts retain R8b range checks.

`Magnitude(value)` returns the stored numeric scalar in its current unit with
the unit removed. It is an explicit builtin, so `Magnitude(5mm)` is `5` and
does not silently convert to meters. Normal `as` still cannot attach or strip
units. R8d makes `interpret value as T` canonical for explicit semantic
attachment at native and decoded-data boundaries. The earlier
`AssumeQuantity<T>` spelling remains a compatibility primitive.

Named derived units are ordinary dimension-and-scale aliases: `Hz = s^-1`,
`N = kg*m/s^2`, `Pa = N/m^2`, `W = kg*m^2/s^3`, and
`V = kg*m^2/(s^3*A)`. The compiler can display a conventional name while
retaining the normalized vector and exact scale in types and artifacts.
`SameDimension`, `UnitScale<numerator, denominator>`, and `Dimensionless`
are compiler analyses usable through `Assert.Concept` and proof output.

R8c does not add runtime unit objects, a unit-symbol expression language,
affine temperature units, logarithmic units, currency, or arbitrary domain
unit declarations. R8d retains the broader foreign-data interpretation and
attachment boundary; R8e retains policy concepts and formatting policy; R8f
retains resume differentiator goldens.
