# R8e2b semantic project policy conformance

Baseline: clean R8e2 subject progression `504ce599ca79b2ff782d2041a292e78095acad43`.
Compiler: `concept-evt1-stage0-go`.

| Requirement | Current evidence |
| --- | --- |
| Nine semantic declaration kinds | `TestR8e2BoundDeclarationKindsAndProvenance` and `TestR8e2MachineUsesBoundAutomataDeclaration` |
| Authored, Generated, Foreign | Bound function provenance in `TestR8e2BoundDeclarationKindsAndProvenance` |
| Artifact-only provenance and dependency scope | `TestR8e2ProjectSubjectsExcludeArtifactDeclarations` |
| Root lint excludes dependency implementation | `TestR8e2bRootPolicyExcludesDependencyImplementation`; imported declaration remains semantically visible |
| Standalone source scope | `TestR8e2StandaloneSourceHasProjectSubjects` |
| Stable subject order and identity | 100-pass inventory check in `TestR8e2BoundDeclarationKindsAndProvenance` |
| Declaration parameter and explicit application | `TestR8e2bDeclarationConceptAssertion`, `TestR8e2bDeclarationConceptCompositionAndArtifactOnly` |
| Category mismatch | `TestR8e2bDeclarationConceptCategoryMismatch`; both directions diagnosed |
| Normal proof truth and NoAllocation | `TestR8e2bHotPathProofOutcomes`: Proven, Disproven, Unknown and original proof evidence |
| Naming and provenance exemption | `TestR8e2bManifestNamingLint`; demo `concept explain --policy ProjectNaming --subject generated_helper` |
| Manifest binding, warning/error, artifact noninterference | `TestR8e2bPolicySeverityDoesNotChangeArtifact` |
| Malformed manifest policy | `TestR8e2bMalformedManifestPolicy`; invalid severity is diagnosed |
| Manifest activation stays local | `TestR8e2bImportedPolicyValueDoesNotActivateRootLint`; imported comptime values remain inert |
| Conflicting name styles | `TestR8e2bConflictingPolicyStyles`; `LINT_POLICY_CONFLICT` |
| Artifact-only declaration concept | `TestR8e2bGeneratedDeclarationConceptArtifactOnly` and foreign composition test |
| MustUse artifact provenance and core enforcement | `TestR8e2bMustUseArtifactExplanation` |
| 100-run finding order | `TestR8e2bManifestNamingLint` repeats full lint evaluation 100 times |
| 100-run policy proof output | `TestR8e2bHotPathProofOutcomes` repeats all three proof graphs 100 times |
| 100-run artifact and MIR/C identity | `TestR8e2bPolicySeverityDoesNotChangeArtifact` repeats compilation and generation 100 times |
| CLI file/project and exit status | `concept lint tests/corpus/r8e2b/valid_unconfigured.concept` exits 0; warnings project exits 0; demo and conflict projects exit 1 |

The corpus under `tests/corpus/r8e2b` includes both category-mismatch inputs,
a malformed manifest, warning-only and conflicting projects, and a valid
unconfigured source. The demo has an authored bad type, field,
function, and local; generated and foreign names; a Proven leaf;
Disproven declared allocation; Unknown opaque foreign operation; and a
foreign MustUse return. `concept explain --policy` uses the same graph as
`Assert.Concept`. Severity does not change truth or exported artifact bytes.

The bounded policy surface does not add formatter work, arbitrary plugins,
runtime code, interface-name exemptions without semantic evidence, or global
immutability analysis.
