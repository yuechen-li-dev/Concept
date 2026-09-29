# R8d conformance

Baseline: clean R8c `f268d85816a16097de528bee5bf5af73185bab31`.
Compiler: `concept-evt1-stage0-go`.

| Requirement | Evidence |
| --- | --- |
| Same-representation scalar attachment and value preservation | `TestR8dInterpretBoundary`, native C11 harness |
| No scale at attachment; later exact `as` scale | `TestR8dInterpretBoundary`, mm and MPa harness results, MIR operations |
| `Magnitude` after interpretation | `TestR8dInterpretBoundary` restore result |
| Representation, unit-conversion, stripping, bit, authority, storage, and address-space rejections | `TestR8dInterpretRejectsMisuse` |
| Foreign and decoded packet boundaries | `TestR8dForeignAndProtocolBoundary`, strict C11, Normal/Verify |
| NoAllocation | `TestR8dForeignAndProtocolBoundary` proof |
| Explicit provenance and `concept explain` | `TestR8dForeignAndProtocolBoundary`; `ExplicitInterpretation` MIR and proof nodes |
| Artifact-only consumer and inspectable provenance | `TestR8dArtifactOnlyAndDeterminism` |
| 100-run artifact, MIR, C and other output byte identity | `TestR8dArtifactOnlyAndDeterminism` |
| Full corpus | `language/evt1/units/r8d` and `TestSemanticCorpusManifest` |

R8d does not introduce enum/bitfield reinterpretation, tensor-wide casts,
runtime quantity metadata, or new authority over storage and lifetimes.
