# R9b research ledger

Status: R9b remains **MEANINGFUL PROGRESSION** for vocabulary admissions;
R9b2 typed predicate protocol is **SUCCESS**. These decisions
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

Closed typed evidence records can now be selected payloads of admitted comptime
Verdict<E,R>. R9-TYPED-VERDICTS specifies projection and bounded transport.
User Proven tests add no structural extent/lifetime/disjointness authority and
cannot turn an allocating operation into NoAllocation. The innate fact authority
allowlist remains empty; no trusted typed fact projection or NoAllocation/Outlives
migration is admitted.

Full R9b closure still requires additional high-value vocabulary admissions and
backend/Vulkan/machine consumers for whichever concepts are actually admitted.
R9b2 closes typed lattice projection, ownership-rule dogfood, bounded typed
diagnostic/artifact transport, and the library/mutex scale qualification. This ledger does not convert those open items into a
principled rejection merely because implementation remains unfinished.

## R9b2 evidence-model revisit

Admission status in the table is unchanged. Typed refutations improve precision
without settling what a property authorizes:

| Candidate | Effect of the typed protocol | Remaining semantic/consumer question |
| --- | --- | --- |
| PlainData | HasDrop(field), ContainsReference(field), RuntimeManaged(type) and UnknownRepresentation(type) can be distinct typed reasons; unknown layout can stay Unknown. | The target representation/geometry contract and raw transport authorization still need qualification. Ownership dogfood demonstrates declaration-bearing reasons, not a new PlainData property. |
| FiniteDomain | A tag-only versus payload-bearing reason can distinguish enumerable complete values from finite case tags. | Which bounded domains are enumerable, and with what ordered inventory/cost? No FiniteCases synonym or new fact is installed. |
| ClosedWorld | Evidence can identify an inventory and boundary; a typed refutation can name an open dependency. | Artifact/plugin/foreign completeness is contextual; a Proven marker does not seal a compilation graph. |
| StableAddress | Refutation can identify a movable/unpinned field or storage contract; Unknown can retain missing lifetime evidence. | Intrinsic address invariants and contextual pinning/lifetime remain different propositions. |
| Relocatable | Refutation can name an address-sensitive invariant or immovable member. | Bitwise relocation and live compaction still require trusted invariant/storage analysis and a real consumer. Movable remains insufficient. |

No candidate is silently admitted on the strength of a payload representation.
The preparation for trusted innate projection is an empty compiler-owned
allowlist, guarded by embedded authority; a future fact-producing rule needs its
own proof/provenance qualification and positive authorized-fact test.
