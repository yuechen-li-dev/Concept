# R8e2 conformance status

R8e2 is **not complete**. This record covers only the declaration-subject
prerequisite. The requested `concept lint` and policy semantics are not yet
available.

| Requirement | Current evidence |
| --- | --- |
| Nine semantic declaration kinds | `TestR8e2BoundDeclarationKindsAndProvenance` and `TestR8e2MachineUsesBoundAutomataDeclaration` |
| Authored, Generated, Foreign | Bound function provenance in `TestR8e2BoundDeclarationKindsAndProvenance` |
| Artifact-only provenance and dependency scope | `TestR8e2ProjectSubjectsExcludeArtifactDeclarations` |
| Standalone source scope | `TestR8e2StandaloneSourceHasProjectSubjects` |
| Stable subject order and identity | 100-pass inventory check in `TestR8e2BoundDeclarationKindsAndProvenance` |
| Manifest policy, severity, conflicts, proposition requirements, lint CLI and explain | Pending declaration-subject binding in the ordinary concept/proof path |

The convergence record identifies the next compiler seam. No R8e2 success
claim or policy diagnostic claim should be inferred from this partial gate.
