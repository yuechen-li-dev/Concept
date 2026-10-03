# R9b research ledger

Current status: **R9b3 qualified**; the historical R9b/R9b2 entries below
retain their original milestone scope. The R9b3 decisions at the end supersede
those admission statuses. See R9B3-CONFORMANCE for current evidence.

## Historical R9b and R9b2 decisions

Status at the earlier closeout: R9b remained **MEANINGFUL PROGRESSION** for vocabulary admissions;
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


## R9b3 qualification decisions

| Candidate | Status and exact subject/truth | Typed evidence, refutations, Unknown | Provenance, trust and consumers |
| --- | --- | --- | --- |
| PlainData | **ADMITTED**, bounded intrinsic type proposition in Standard.Semantic.Vocabulary. Positive fixed inspectable geometry and recursively self-contained semantic value representation. | PlainDataEvidence contains size/alignment as usize<byte>. PlainDataRefutation: HasDrop, HasOwnedResource, ContainsReference, RuntimeManaged, AddressSensitive, FieldProblem(declaration field, string nestedProblem). Opaque handles, partial/runtime storage, non-record aggregates, payload sums, unresolved/missing geometry, zero-size representations (including nested fields) and budget exhaustion stay Unknown. | Ordinary restrictive Concept derivation over checked observations and the existing layout owner. Vulkan Upload/Download and AMD64 bridge representation assertions are real consumers. No optimizer authority. |
| Relocatable | **DEFERRED**. A live value moves bitwise to new storage, ending the old storage lifetime without semantic move logic or broken invariants. | Research RelocationEvidence::BitwiseLiveMove is never emitted. RequiresStableStorage(type) refutes immovable. Movable/Drop/owned types stay Unknown; they are not blanket negatives. | Existing immovable observation is precise; external registration/self-reference/Drop invariants lack summaries. No current safe compaction consumer. Requires both trusted invariant analysis and a real relocation operation before admission. |
| RequiresStableAddress | **SCOPED research; public admission DEFERRED**. Intrinsic invariant, distinct from a particular location's pinning. | AddressRequirementEvidence::Immovable(type) covers the existing restriction only. Absent immovable is Unknown. Typed refutation slot is reserved; no contradictory invariant is observed and no Disproven is emitted. | Existing immovable fixed-storage and owner/pointee checks remain authoritative. A public alias would overstate the observation without adding a consumer. |
| StableAddress | **REJECTED as an ambiguous combined name**. | No invented Verdict combining intrinsic and contextual truths. | This rejects conflation, not existing immovable storage legality. |
| Pinned | **DEFERRED contextual contract**, particular live storage plus lifetime/region identity. | Evidence would need storage identity and lifetime coverage; refutation would need an observed move or escaped lifetime. Naked type observations cannot supply either, so no artificial payload/result is installed. | Existing scoped provenance/NoCopy/storage checks retained. A general lifetime/pinning mechanism would exceed the milestone. Moving an owning allocation need not move its immovable pointee. |
| FiniteCases | **SCOPED executable research; standard admission DEFERRED**. Exact checked enum tag inventory including payload sums. | FiniteEvidence.cardinality; NotVariantType(type). Case names/tags/payload counts follow declaration order; invalid indices are OBSERVATION_INVALID. | Research.Finite ordinary/artifact probes. Existing exhaustive match, machine input matching and BridgeSchema reflection already own complete inventories. No distinct production consumer needs another alias; broad replacement would duplicate authority. |
| FiniteDomain | **DEFERRED**, bool/payload-free enum research scoped. Complete compiler-enumerable runtime values, not mathematical finiteness of machine bits. | FiniteEvidence cardinality 2 for bool or payload-free case count. Payload sums and unenumerated scalar domains Unknown. The shared refutation type exists but the domain probe emits no false unbounded-domain theorem. | No complete payload value enumeration producer or actual value-table consumer. No enormous integer enumeration. |
| ClosedWorld | **SCOPED existing local activation inventory; general admission DEFERRED**. Complete inventory relative to an explicit compilation/artifact boundary. | Prospective InventoryEvidence(boundary, inventoryHash, memberCount) is never emitted as Proven. IdentityDoesNotCoverBoundary(identity, firstBoundary, secondBoundary) refutes topology ID alone as a boundary seal. | Actual EVT2x Worker fixture in two distinct modules has the same topology ID and three members each. Existing local activation layout remains authoritative. General artifact/plugin/foreign completeness needs a boundary-covered seal; no identity/runtime redesign merely to admit a name. |
| StaticExtent | **Previously ADMITTED, RETAINED**: exact 1D fixed N, 0..1048576. | Existing layout-derived N; known mismatch Disproven, runtime views/multidimensional/partial capacity Unknown. | Existing Assert/explain/artifact/Planner shadow unchanged. No typed migration or new optimization needed. |
| Contiguous | **Existing vocabulary RETAINED**: linear traversal of checked storage/view with its geometry. | Existing layout/span/tensor facts and scope. | Existing Planner/storage consumers. PlainData and finite tags do not establish view geometry. No competing generic range name. |
| Disjoint | **Canonical region vocabulary RETAINED**: non-overlap of checked named regions/intervals for a relevant operation. | Existing layout/borrow/access provenance; dynamic Unknown and guarded dominance retained. | Existing Planner/access consumers. PlainData does not imply non-aliasing; no NoAlias competitor or new check removal. |

### Required semantic distinctions

PlainData describes self-contained semantic representation, not merely fixed
byte layout. Fixed layout alone is insufficient when ownership, references,
Drop, or hidden runtime state remain meaningful.

Relocatable means a live value may change storage address by bitwise relocation
without semantic move logic. It is distinct from source-level Movable. Drop and
ownership may be compatible, but their post-relocation invariants are not proved.
No relocation-positive corpus is invented.

A type requiring stable address and a particular value being pinned are separate
propositions. One describes a type invariant; the other describes a storage
guarantee. Absence of the requirement does not imply Relocatable.

A finite set of variant cases does not imply a finite set of runtime values.
Payload-bearing sums may have FiniteCases while lacking FiniteDomain. The probe
leaves complete payload values Unknown rather than claiming every payload domain
is unbounded.

Closed-world knowledge is relative to an explicitly known compilation or
inventory boundary. It is not an intrinsic global property of a type.

### Contracts and transport

Neither PlainData/CAbiValue implication is installed. Ordinary bool records and
half can satisfy PlainData without meeting extern-C requirements. CAbiValue
admits explicit foreign handles; pointer-sized layout does not establish semantic
PlainData. CReprIsPlainData remains a different repr(C) declaration rule. Vulkan
AsBytes C ABI restrictions are retained. Wire encoding/version/field order remain
owned by BridgeSchema/BridgeCodec, not by PlainData.

PlainData copies an already valid value representation. It does not authorize
arbitrary-byte decoding, Zeroable, canonical padding, cross-target wire formats,
or validation of IDs/offsets into an external blob. Geometry is the existing
supported 64-bit target model; wider target-layout equivalence is not claimed.

Ordinary Assert.Concept<PlainData>(T, "reason") transports size/alignment on
Proven, typed offending type/field on Disproven, and bounded semantic explanation
on Unknown. Vulkan requirements use the same projection and Concept renderer,
without a Go-side reason switch. Artifact-only consumers re-evaluate transported
predicate/helper bodies against checked types; selected metadata never hydrates
optimizer facts.

Finite/address/closure probes live only under testdata/r9b3 and are not Standard
admissions. Finite assertions/artifacts and address typed outcomes qualify the
research distinction. Closure is a direct contextual typed probe, not a naked
ClosedWorld<T> assertion. Unadmitted Pinned has no public assertion/explain/artifact
guarantee: spelling evidence cannot supply missing storage observations.

The trusted-fact registry remains **empty**. FakePlainData/FakeRelocatable/
FakeClosedWorld/FakeFiniteDomain prove only their own identities, transport no
fact authority, and cannot make the real PlainData<Resource> assertion succeed.
All admitted vocabulary is restrictive; no new innate rule or optimizer pass.

R9B3-CONFORMANCE records costs, budgets, 100-run ordering/results, shadow agreement
and gates. These deferrals are grounded in missing semantic observations or a
redundant consumer boundary, not unfinished routine implementation work.
