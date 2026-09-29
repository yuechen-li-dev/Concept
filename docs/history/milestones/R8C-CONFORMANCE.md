# R8c conformance

Baseline: clean `13613eddd18ae62a1f6710de825bd335d54da189`, compiler
`concept-evt1-stage0-go`.

| Area | Executable evidence |
| --- | --- |
| Complete curated catalog and excluded spellings | `TestR8cStandardUnitCatalog` |
| Normalization, N/Pa/Hz, cancellation, exact ratios | `TestR8cDimensionAndScaleLaws` |
| Scientific literal lexer and attached literal typing | `TestR8cScientificTokenization`, `TestR8cLiteralsCastsAndMagnitude`, and `language/evt1/units/r8c/valid/scientific_literals.concept` |
| Scaled casts, combined representation change, current-unit `Magnitude` | Native C11 harness in `TestR8cLiteralsCastsAndMagnitude` |
| Scale conversion retained in MIR, ordinary scalar arithmetic in C | MIR and C assertions in `TestR8cLiteralsCastsAndMagnitude` |
| Scaled dimensionless cancellation | Native ratio assertion in `TestR8cLiteralsCastsAndMagnitude` |
| Comptime literals and exact unit facts | `TestR8cComptimeAndArtifactTransport` |
| Artifact-only imported quantity consumers | `TestR8cComptimeAndArtifactTransport` |
| Tensor quantity contraction | Native C11 harness in `TestR8cTensorComposition` |
| Table, fixed array, and Span with unit elements | `TestR8cLiteralsCastsAndMagnitude` |
| Half quantity conversion | GCC extension harness in `TestR8cHalfUnitExtension` |
| Normal/Verify parity and GCC/Clang strict C11 | `TestR8cNormalVerifyParity` |
| Artifact/MIR/C/proof byte determinism | `TestR8cDeterminism100` |
| `concept explain` exact scale evidence | `concept explain language/evt1/units/r8c/valid/scaled_casts.concept:10 --verbose` |
| Disallowed attachment, stripping cast, dimension change, integer scale cast, unknown prefix, scale overflow | `TestR8cRejections` and `language/evt1/units/r8c/invalid` |
| Full semantic corpus | `TestSemanticCorpusManifest` includes four R8c valid and six R8c static-invalid specimens |

The corpus specimens are ordinary `.concept` files and are counted in
`language/evt1/manifest.json`. No source reparse is required for the imported
R8c dependency in the artifact test. The `NoAllocation` assertion for a
scaled cast closes through the existing proof system.
