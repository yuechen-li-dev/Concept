# EVT1 R7x2 convergence record

Baseline: clean `1196c60` after R7x. This milestone keeps the existing
`ForeachStmt` and C11 backend as the only runtime iteration path.

## Range values and loops

`start..end` is a copyable `Range<T>` value for a dimensionless integer `T`.
The start is included and the end is excluded. Equal endpoints produce an
empty range. `start..end step magnitude` advances upward; `start..end descend
magnitude` advances downward. Both magnitudes must be positive. The default
magnitude is one. Ascending requires `start <= end`, and descending requires
`start >= end`. Known contradictory literal endpoints and literal zero
magnitudes fail validation; dynamic values are guarded at runtime.

The value works through `for (item in source)` and compatible `foreach`.
Validation assigns the ordinary built-in inline iterator strategy. Generated
C evaluates the range source once, keeps a scalar cursor, and clamps the final
advance to the exclusive end before arithmetic could overflow. The cursor
also persists in the existing async state graph across `await`. A machine
state uses the same counted lowering. `Range<int>` is the built-in generic
value; an existing nongeneric user struct named `Range` remains a distinct
type.

## Mutations and math

Prefix and postfix `++`/`--` are statements only. They and all ten compound
assignments use one place evaluation, the existing numeric validation and
checked arithmetic lowering. `%=` uses the existing Euclidean `%` helper.
An indexed side effect executes once in the native C11 test.

`Standard.Math` supplies `Abs`, `Min`, `Max`, and `Clamp` for `int` and `float`,
plus the requested single-precision transcendental, rounding, trigonometric,
and classification functions. The C11 `<math.h>` functions are called with
their exact float signatures. Only those exact known C11 primitives receive
a compiler-derived `NoAllocation` summary; other extern C calls retain their
unknown effect. `^` remains bitwise XOR.

## Dogfood and remaining boundary

One loop each in DragonGod scheduling, Standard.Collection, and
Standard.Octagon now uses a finite range. No additional laundry syntax was
added.

Source-level MMIO/volatile operations and fixtures are absent at this
baseline: the repository's EVT1 compatibility and conformance documents
explicitly defer them. Proving MMIO observability inside a range loop would
require that separate semantic subsystem. The R7x2 change does not fabricate
a volatile access or treat ordinary field access as MMIO.

## Evidence

Focused native C11 tests cover range values, ascending/step/descend/empty
ranges, signed and unsigned overflow edges, machine and async loops, all
mutation shortcuts, Euclidean `%=`, the math surface, and 100 byte-identical
range MIR/C/H generations. The range comparison test checks ordinary `while`
and range `for` generated C for inline loops and verifies the range's existing
allocation-free foreach MIR strategy. `go test ./...`, `go vet ./...`, root
and legacy Zig tests, and R7d3 BurnIn passed. The full EVT1 manifest retained
398 valid, 262 static-invalid, and 13 runtime-negative fixtures. The Standard
package test passed 29/29 tests; DragonGod passed 21/21 tests and one
benchmark. `Standard.Math` also executed under strict C11 with `-Werror` and
linked `<math.h>` primitives through `-lm`.
