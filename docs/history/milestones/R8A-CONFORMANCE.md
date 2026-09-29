# R8a floating representation conformance

Baseline: clean R7q honest-stop commit
`82c85dc5dfb8a27c1e07164aa2d94ef41aefba63`; compiler
`concept-evt1-stage0-go`.

| Surface | Current qualification |
| --- | --- |
| Canonical names | `half` = binary16, `float` = binary32, `double` = binary64 |
| Aliases | `float16`, `float32`, `float64` canonicalize to those identities |
| Geometry | `SizeOf` 2, 4, 8 bytes; target layout alignment 2, 4, 8 in current engine |
| Quantities | Representation and unit are independent; `float<K>` stays binary32 kelvin |
| Literals | Unsuffixed expression defaults to binary32; declared `double` literals retain binary64 decimal C spelling |
| Facts | `Floating`, `BinaryFloat`, `ScalarBits`, exponent and mantissa bits through compiler analyses and named concepts |
| Artifacts | Semantic payload carries the representation; artifact-only consumer resolves aliases and unit-qualified signatures |
| Reflection | Canonical field types carry explicit representation and unit dimensions |
| C | `float` and `double` are strict C11; `half` uses the `_Float16` extension |
| ABI | Existing C float behavior preserved; double admitted as C scalar; half has no C ABI claim |
| Deferred formats | `bfloat16`, `float8e4m3`, `float8e5m2`, ambiguous `float8`/`float4` |

The `float<32>` and `float<64>` negative probes now explain that `<...>` is a
unit/dimension expression and list the canonical and compatibility spellings.
The supported `concept explain <file> --verbose` path can show a quantity
alias such as `Kelvin : float<K>` beside the proved
`floating representation binary32` fact.
`half` arithmetic generates native `_Float16` syntax on both probed compilers.
GCC's pedantic C11 lane rejects the type itself, and the installed Clang
Windows/MSVC target cannot link the emitted conversion helpers. GCC's
extension lane does compile and run the binary16 arithmetic probe. Binary16 therefore prevents a
full R8a strict-C11 success claim. The implementation does not silently widen
half storage to C `float`.

Focused automated coverage is in
`internal/concept/r8a_floating_test.go`: semantic identity, facts,
`Assert.Concept`, reflection, artifact-only consumer, negative formats,
generated C, strict-C11 binary64, extension-only binary16, and 100-run
artifact/MIR/C/proof byte identity. The frozen R7 suites remain regression
gates rather than being rewritten for this amendment.
