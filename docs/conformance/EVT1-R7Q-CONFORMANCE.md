# EVT1 R7q qualification: honest stop

Baseline: clean `43a6af3fd566020a480231187d1925b4121c6b17`, compiler
`concept-evt1-stage0-go`. The frozen corpus contains 400 valid, 269
static-invalid, 13 runtime-negative, 5 compatibility, and 4 expected-divergence
fixtures. Standard has 35 facts and 3 benchmarks; DragonGod has 23 facts and 1
benchmark. The nine R7p domains remain indexed in
[EVT1-GOLDENS.md](../examples/EVT1-GOLDENS.md).

## Qualification completed before the stop

| Gate | Result |
| --- | --- |
| `go vet ./...` | Pass |
| Root and legacy Zig `zig build test` | Pass |
| BurnIn `go test ./internal/concept -run R7d3 -count=1` | Pass |
| Standard Normal / Verify | 35 / 35 facts pass |
| DragonGod Normal / Verify | 23 / 23 facts pass |
| R7p Concept goldens plus new geometry Normal / Verify | 28 / 28 facts pass (27 R7p, 1 R7q) |
| R7p native C++ companion Normal / Verify | 2 / 2 facts pass |
| Full `go test ./...` including EVT1 corpus | Pass |
| New geometry generated C, Clang and GCC strict C11 | Pass (`-std=c11 -pedantic-errors -fsyntax-only`) |

The new [geometry source](../../libraries/Golden/Differentiators/Geometry/Frame.concept)
is 15 source lines (plus a 10-line test). It
uses a 3D quarter-turn basis and the Einstein contraction
`world[i] = basis[i, j] * local[j]`. `basis` has dimensionless `float`
elements; `local` and `world` have `float<m>` elements. The result is `-2 m`
in both modes. The [negative source](../../libraries/Golden/Differentiators/Geometry/dimensional-mismatch.concept.txt)
is rejected with `CV4615`: an `m * m` contraction produces `float<m^2>` and
cannot initialize `float<m>`.
Generated C retains fixed backing arrays, direct nested loops, and shape
metadata with runtime extent guards even for this fixed example. It contains
no tensor heap or runtime reflection. This review is structural, not a timed
performance claim.

## Frozen semantic contradiction

R7q requires an HPC algorithm parameterized over scalar precision with
`float<K>`, exercising at least two widths and comparing their generated C.
The frozen [quantity guide](../language/QUANTITIES-AND-UNITS.md) defines
`float<K>` as a temperature measured in Kelvin. The compiler maps the base
`float` representation to C `float` in `internal/concept/profile_definition.go`.
The preserved [precision probe](../../libraries/Golden/Differentiators/Hpc/precision-probe.concept.txt)
tries `float<32>` and `float<64>`; checking its first declaration reports
`CV4102: unknown type application float<32>`. These are not two supported
precision instances. Renaming a template parameter to `K` cannot add a scalar
width to the frozen type system.

**POST-EVT1 PROPOSAL:** Introduce distinct, explicit scalar representations
for at least 32-bit and 64-bit floating point, with arithmetic, generic
specialization, artifact, MIR, and C emission laws. Preserve `float<K>` as
Kelvin; repurposing it would silently change existing quantity programs.
Choose the final syntax in a separate milestone. No precision semantics were
added in R7q.

Result: **HONEST STOP**. A required differentiator cannot be expressed under
the frozen EVT1 semantics. The game, HPC, compiler, and async-system R7q
goldens, artifact-only consumers, representative 100-build checks, and full
performance/codegen comparison were not claimed as completed.
