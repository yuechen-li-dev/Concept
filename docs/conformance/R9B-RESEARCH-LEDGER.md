# R9b research ledger

Status: **MEANINGFUL PROGRESSION**, not full milestone admission. These decisions
distinguish semantic definitions from implementation/consumer qualification.
Class A = descriptive/restrictive; B = codegen-authorizing with trusted evidence.
Deferred entries have no new Assert/explain/artifact behavior or measured cost.
They must not be inferred as Proven from a positive research description.

| Candidate | Status and exact scope | Trust/provenance | Existing consumer seams and unresolved admission |
| --- | --- | --- | --- |
| PlainData | Deferred admission. Fixed inspectable target representation, no Drop/hidden managed resource/reference lifetime, recursively qualified fields and inspectable geometry. | A structural description; any raw-copy/storage authorization is separate B, DerivedFromLayout with target context. | Vulkan push/mapped records and bridge/storage field schema are strategic seams, not qualified adopters. Schema serializability is not raw-object transport. Must qualify field geometry, resources and negative evidence before adding the name. |
| Relocatable | Deferred. A live value can change storage address without semantic move logic or invalidating invariants. Movable does not imply this. | B for actual compaction; trusted invariant/address analysis, not user declaration alone. | DenseStore growth/arena compaction need this only when introduced. Existing ownership tests show owners moving without relocating immovable pointees; they do not prove bitwise relocation. No immediate safe compaction consumer admitted. |
| StableAddress | Scoped, deferred admission. Intrinsic address stability and a pinned value/storage lifetime contract are different propositions. | A restriction; B only with checked lifetime/storage context. | Existing immovable fixed-store values are a real motivating case, but immovable does not enumerate all registration/pinning contracts. An intrinsic Proven result would refute intrinsic live Relocatable; a contextual pinned result would not refute an unrelated type globally. |
| FiniteDomain | Deferred admission. Compiler-known complete enumerable value set, not just a finite tag set or mathematically finite machine integers. Bool and payload-free enums are initial bounded candidates. | A; compiler structural enumeration, ordered cases/cardinality. Dispatch/check removal needs separate B qualification. | Exhaustive match and generated BridgeSchema enum dispatch already enumerate cases. Payload enum case/tag domain is finite while payload runtime value domain is not automatically enumerable. A FiniteCases observation/proposition would need a separate precise domain; no synonym has been installed. |
| ClosedWorld | Context resolved, admission deferred. Complete reachable implementation/variant inventory for an identified compilation boundary and artifact set. | A contextual description; B for devirtualization needs hash-covered closed graph and boundary, proposed ClosedWorldDerivation. | EVT2x activation inventory is the flagship seam; reflected schema dispatch is a second candidate. Neither proves completeness across foreign/plugin/dynamic boundaries. No globally intrinsic ClosedWorld type fact installed. |
| StaticExtent | Admitted bounded research form: exact one-dimensional fixed array length N, 0..1048576. Multidimensional shapes, runtime views and partial-storage capacity are excluded. | A assertions; compiler-owned DerivedFromLayout shadow fact is eligible for future B audit, not new optimization authority. | Actual fixed vector tensor Planner records the fact; generated C11 vector operation executes. Known mismatch Disproven; insufficient exact view/capacity evidence Unknown. Artifact-only type assertion, explain and 100-run deterministic truth/MIR proof pass. |
| Contiguous | Existing vocabulary retained; no competing range concept added. Property of a storage/view with linear element traversal; not of arbitrary element int. | Existing type/layout/span/tensor provenance. Contiguity alone cannot justify alias or bounds elimination. | Existing tensor Planner consumes it; Span/layout/stream transport already preserves or narrows view geometry. Further stride/element metadata API admission deferred. |
| Disjoint | Existing canonical vocabulary retained. Proven non-overlap of the named checked regions/intervals for the relevant operation, not unrestricted transitive reachability. | Existing layout/region/borrow/access derivations; unknown dynamic evidence remains Unknown. Guarded facts require dominance/scope before any unconditional B use. | Existing tensor vectorization eligibility consumes disjoint destination/input regions with checks disabled only under existing qualified policies. Distinct variables alone are insufficient. No NoAlias/Separate/Independent competitor introduced. |

## PlainData versus CAbiValue

CAbiValue retains representation/ABI and native-toolchain evidence requirements;
it is not replaced by PlainData. An internal packed/GPU/page record could satisfy
a future PlainData contract while failing C ABI compatibility. Conversely the
current CAbiValue admits explicit foreign handles; that does not establish that
all such handles have inspectable, ownership-free raw transport semantics under
the proposed PlainData contract. Do not install an unconditional implication
until these differing contracts are reconciled. Existing CReprIsPlainData is a
repr(C) declaration rule name, not an already-admitted generic PlainData property.

## Qualified evidence and costs

StaticExtent assertion syntax: `Assert.Concept<StaticExtent<4>>(Array4, "reason")`.
Explain gives the exact N and DerivedFromLayout reason. The fact is transported
in ordinary qualified MIR; semantic artifact imports re-derive it from checked
array type metadata. No new artifact schema or proof-tree transport is introduced.
Derivation is constant-time for rank one; the measured 100-run focused suite is
informational rather than a hard admission budget. FixedShape is retained and
the shadow test compares the extent with the existing shape metadata.

Closed typed evidence records are qualified values, not admitted semantic
verdicts. The cost and exact predicate admission blocker are documented in
R9-TYPED-VERDICTS. User predicate tests verify that returning true adds no
structural extent fact for a Span. Trusted innate fact generation is deferred;
NoAllocation/Outlives migration is intentionally absent.

Full R9b closure still requires several new high-value admissions, typed verdict
lattice projection and innate dogfood, useful typed diagnostic/artifact transport,
and backend/Vulkan/machine consumers and scaling evidence for whichever concepts
are actually admitted. This ledger does not convert those open items into a
principled rejection merely because implementation remains unfinished.
