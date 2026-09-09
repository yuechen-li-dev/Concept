# EVT1 semantic fact transport direction

Status: R6h implemented

## Authority and subject identity

Runtime values keep only their ordinary representation. `SemanticValueFacts` is the compile-time record transported by semantic analysis and projected into MIR. It does not add fields to Address, MemoryRegion, Storage, Span, Tensor, callable environments, async frames, or machines.

A **place** is a storage location such as a local or field. A **value** is the meaning copied, moved, borrowed, returned, or embedded through that place. Assignment changes the destination place while transporting applicable value facts. Stable subject kinds are `Value`, `Place`, `Parameter`, `FunctionResult`, `Field`, `CaptureField`, `AsyncPersistentField`, `MachineField`, and `ImportedValueSummary`. Operations remain separate effect/fact subjects.

R4 Span/Tensor facts remain their subsystem validators. They bridge into the unified record; they are not a second transport authority.

## Fact-loss inventory

| Boundary | R6g state | R6h state |
|---|---|---|
| expression to local | region geometry lost | preserved |
| local to assignment target | Span-only refresh | preserved or conservatively joined |
| argument to parameter | provenance only | exact concrete call facts; symbolic otherwise |
| return to caller | provenance summary only | bounded value-result summary |
| Result/Option payload | lifetime only | payload facts preserved on success extraction |
| aggregate/generic field store and load | type structure only | field-sensitive facts preserved |
| callable capture/invocation | capture provenance only | capture-field facts preserved |
| async live-across-await | liveness and backing identity only | persistent-field facts projected |
| machine state field | type/provenance only | persistent-field facts projected |
| module export/import | effects and provenance only | checked value-result summaries; missing is Unknown |

## Authoritative preservation table

`P` means exact preservation, `W` conservative weakening is allowed, `R` means recomputed from a proven relation, and `N/A` means the fact is not a value fact at that boundary.

| Fact | copy | move | ref | field | return/parameter | generic | module |
|---|---:|---:|---:|---:|---:|---:|---:|
| AddressSpace | P | P | P | P | P | P | P |
| RegionOrigin / RegionIdentity | P | P | P | P | P/W | P | P/W |
| ParentRegion | P | P | P | P | P/W | P | P/W |
| RelativeOffset / ByteInterval | P | P | P | P | P/R/W | P | P/R/W |
| ByteExtent | P | P | P | P | P/R/W | P | P/R/W |
| Alignment | P | P | P | P/R | P/R/W | P | P/R/W |
| Contiguous / Bounded | P | P | P | P | P/W | P | P/W |
| Mutability | value invariant | value invariant | R from ref kind | P | R from formal | P | P/W |
| Lifetime / Provenance | P | P | shortened borrow | P | shortest, never longer | P | P/W |
| Disjointness basis | P | P | P | P | retained only with origin and interval | P | P/W |
| Shape / Rank | P | P | P | P | P/W | P | P/W |
| NoCopy / NoOwnershipTransfer | P | transfer law enforced | P | structural | P/W | structural | P/W |
| HostAccessible | P | P | P | P | P/W | P | P/W |
| Initialized | P subject to move state | transferred | view of same object | field-sensitive | P/W | P | P/W |
| NoAllocation | N/A operation fact | N/A | N/A | N/A | existing call summary | N/A | existing effect summary |

Unknown never becomes known merely through transport. `AddressFromBits` stays origin/provenance/alignment Unknown. Local provenance cannot become Static. Missing or old module summaries become an `OpaqueBoundary`, not consumer-side body inference.

## Result summaries and joins

Function bodies derive a bounded, deterministic expression tree using only parameter selection, field selection, fact selection, integer constants, byte-offset addition, storage origin, and common guaranteed alignment. Result/Option success wrappers are transparent to these compile-time payload facts. Call sites bind actual arguments, evaluate known expressions, and leave unresolved expressions Unknown.

The `Subregion` summary is derived structurally from its ordinary result construction. Common alignment is recognized from the bounded Euclidean algorithm's typed body shape, not a library or function name. A known parent at offset 12 therefore degrades alignment by `gcd(parent alignment, 12)`. Runtime offsets retain origin and extent where known but make relative interval and derived alignment Unknown.

Summary expressions are limited to 128 nodes and depth 16. Artifact ordering is canonical and duplicate operation/signature summaries are rejected.

Branch, match, and carrier joins are bounded. Equal origin, interval, extent, or exact guarantee survives. Different origins become Unknown. Known alignments join by greatest common divisor. Certainties survive only when equal. Provenance selects the shorter/conservative bound. Loops rely on ordinary validation and do not synthesize arbitrary invariants.

## Region and storage transitions

Address addition preserves space/origin/provenance, adds a known relative byte offset, and degrades alignment by greatest common divisor. Structurally constructed region-like values derive geometry from an `Address<Space>` field and typed byte extent/alignment fields; no `MemoryRegion` name is recognized.

`bind<T>` accepts only a transported live storage origin in `SystemMemory`; reconstructed bits remain rejected. It preserves address-space, origin, extent, alignment, provenance, contiguity, boundedness, and no-allocation facts. `Initialize` retains that origin, begins the object lifetime, sets object extent/alignment facts, and marks the value initialized. `Destroy` remains the existing semantic state transition: it ends the object and leaves raw storage geometry available to its owner.

## Modules and zero cost

`concept-module.v1` adds `value_fact_summaries`, alongside but semantically separate from `operation_effect_summaries`. The source body is authority when producing the artifact; the validated artifact summary is authority in a consumer. Older artifacts without a needed summary load compatibly but yield Unknown. Consumers do not manufacture a summary from the payload body.

MIR exposes transported facts and function result summaries to the Planner. The Planner consumes this record and does not reconstruct region geometry. Proof graphs render the transport chain and identify opaque boundaries. Generated C is unchanged except for ordinary program behavior: no provenance field, region ID, proof table, RTTI, registry, allocation, or instrumentation is emitted.

## Deferred

Source-level `T*`, MMIO/volatile, generalized FFI semantic contracts, symbolic inequality solving, full range analysis, allocator framework work, GC, DMA/IOMMU mapping, and native backend/LIR remain deferred.
