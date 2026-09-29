# R8b explicit conversion conformance

Baseline: clean `60f7d9498e7874a710b75fe2a1e11d9cb797b50b`, compiler
`concept-evt1-stage0-go`. R8a supplies binary16 `half`/`float16`, binary32
`float`/`float32`, and binary64 `double`/`float64`. The unit argument is
orthogonal to the scalar representation.

| Requirement | Evidence |
| --- | --- |
| Canonical `as` and `static_cast` alias | `TestR8bCastAliasSemantics` compares generated C and semantic MIR operations |
| Runtime cast, float/int and float/float | `TestR8bExplicitCastCore` executes generated C with GCC and Clang |
| Integer range and no wrap | `TestR8bExplicitCastCore` checks signed/unsigned successes and four runtime traps |
| Float-to-int `as` rejection | Both spellings pinned to `FLOAT_TO_INT_ROUNDING_REQUIRED` |
| Four named rounding rules | `TestR8bNamedRoundingBuilds` executes generated C with GCC and Clang |
| Ties to even, negative zero, signed/unsigned limits | Executable harness in `TestR8bNamedRoundingBuilds` |
| NaN and infinities | Executable harness checks `NotFinite` for all three inputs |
| `Result` and `?` | `Propagate` fixture checks success and error propagation |
| Quantity authority | Same-unit representation cast succeeds; attachment, stripping, and dimension change reject |
| Exactness facts and `Assert.Concept` | `TestR8bExactConversionFacts` proves int32→double and float→double; rejects isize→float |
| Artifact-only transport | `TestR8bArtifactOnlyConversions` loads two compiled modules without dependency source, then links and executes generated C |
| Verify parity | `TestR8bVerifyConversionParity` compiles and executes Verify C |
| Strict C11 | Focused GCC/Clang `-std=c11 -pedantic-errors` builds for float/double paths |
| Half qualification | `TestR8bHalfExtensionLane` executes GCC `_Float16` extension path |
| NoAllocation | Compile-time assertions in cast and rounding fixtures |
| Determinism | `TestR8bDeterminism100` compares artifact and all generated outputs for 100 runs |

Existing implicit runtime compatibility remains exact-type, apart from the
documented pre-R6g byte-quantity bridge. Contextual numeric literals are
checked against their declared target type. No new implicit precision
narrowing or float-to-integer conversion was added.
