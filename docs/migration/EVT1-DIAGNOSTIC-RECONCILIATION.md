# EVT1 diagnostic reconciliation

Status: R4k class, interface, and dyn reconciliation

Concept EVT1 compares diagnostics by semantic family. The active Go compiler
retains `CV` numbers and the retired PoC3 compiler retains `CON` numbers; R2
does not mass-renumber either namespace. `Diagnostic.SemanticCategory()` is
the executable mapping used by the R2 conformance harness.

| Semantic category | PoC3 code/family | Go code/family | EVT1 canonical diagnostic name | Numeric namespace status | Notes |
|---|---|---|---|---|---|
| not assignable | `CON0081` field-assignment non-place | `CV4127` | `NOT_ASSIGNABLE` | preserved, not numerically unified | temporaries and non-place targets |
| const mutation | no direct closed fixture | `CV4128` | `CONST_MUTATION` | Go number retained | covers whole local, projected field, and natural const parameter place |
| record mutation | no record syntax | `CV4142` | `RECORD_FIELD_MUTATION` | EVT1-new Go number | type-level read-only projection |
| unknown field | `CON0075`, `CON0080` | `CV4026`, `CV4144` | `UNKNOWN_FIELD` | both namespaces retained | ordinary access and `with` share a semantic family |
| duplicate with field | `CON0076` is analogous construction evidence | `CV4145` | `DUPLICATE_WITH_FIELD` | EVT1-new Go number | PoC3 has no `with` syntax |
| with type mismatch | `CON0078`, `CON0082` are analogous aggregate/assignment evidence | `CV4147` | `WITH_FIELD_TYPE_MISMATCH` | EVT1-new Go number | replacement value type must match exactly |
| with requires record | no direct counterpart | `CV4143` | `WITH_REQUIRES_RECORD` | EVT1-new Go number | includes ordinary structs and non-record values |
| non-copyable with | `CON0154` non-copyable flow is analogous | `CV4146` | `WITH_NONCOPYABLE_RECORD` | EVT1-new Go number | future move-aware update is not implied |
| immovable copy | `CON0154` non-copyable local family; no closed immovable fixture | `CV4134` | `IMMOVABLE_COPY` | both namespaces retained | direct final-storage construction remains valid |
| immovable assignment | `CON0154`, `CON0160` non-copyable assignment families | `CV4135` | `IMMOVABLE_ASSIGNMENT` | both namespaces retained | whole-value replacement is rejected |
| immovable by-value parameter | `CON0154` non-copyable argument family | `CV4136` | `IMMOVABLE_BY_VALUE_PARAMETER` | both namespaces retained | borrow/reference follow-up remains deferred |
| immovable return | `CON0154` non-copyable return family | `CV4137` | `IMMOVABLE_BY_VALUE_RETURN` | both namespaces retained | direct or local return is rejected at the signature boundary |
| immovable embedding | `CON0154` non-copyable-field pressure | `CV4138` | `IMMOVABLE_EMBEDDING` | both namespaces retained | applies to ordinary and record fields |
| immovable enum payload | `CON0154` non-copyable aggregate pressure | `CV4139` | `IMMOVABLE_ENUM_PAYLOAD` | both namespaces retained | payload storage would relocate/copy the value |
| movable-only copy | `CON0154` | `CV4501` | `COPY_OF_NONCOPYABLE` | EVT1 number, PoC3 family preserved | `owned T` requires explicit move from an existing owner |
| use after move | `CON0151` | `CV4502` | `USE_AFTER_MOVE` | EVT1 number | includes reads and projections |
| maybe-moved use | `CON0155` | `CV4503` | `MAYBE_MOVED_USE` | EVT1 number | simple control-flow join is conservative |
| second move | `CON0151` pressure | `CV4504` | `MOVE_AFTER_MOVE` | EVT1 number | distinct family for transfer intent |
| immovable move | immovable was design-only | `CV4505` | `MOVE_OF_IMMOVABLE` | EVT1-new | references remain legal |
| return without move | `CON0154` | `CV4506` | `RETURN_NONCOPYABLE_WITHOUT_MOVE` | EVT1 number | explicit local transfer required |
| move source is not a place | `CON0152`/`CON0153` pressure | `CV4507` | `MOVE_REQUIRES_PLACE` | EVT1 number | R3 supports whole local/parameter sources only |
| reference requires place | no selected direct PoC3 equivalent | `CV4508` | `REF_REQUIRES_PLACE` | EVT1-new | no temporary lifetime extension |
| mutable ref from const | PoC3 borrow qualifier pressure | `CV4509` | `REF_MUTABLE_FROM_CONST` | EVT1-new | composes with R2 const places |
| reference type mismatch | PoC3 borrow mismatch pressure | `CV4510` | `REF_TYPE_MISMATCH` | EVT1-new | element types must match |
| obvious reference escape | lifetime system deferred | `CV4511` | `REF_ESCAPE_LOCAL` | EVT1-new | all R3 reference returns rejected conservatively |
| pass without move | `CON0154` | `CV4512` | `PASS_NONCOPYABLE_WITHOUT_MOVE` | EVT1 number | existing owner must be transferred explicitly |
| mutation through ref const | PoC3 immutable-borrow pressure | `CV4513` | `MUTATION_THROUGH_REF_CONST` | EVT1-new | record immutability still has `CV4142` |
| use before initialization | `CON0150` | `CV4514` | `USE_BEFORE_INITIALIZATION` | reserved R3 state family | current source locals require initializers |
| ambiguous/moved drop state | `CON0155` cleanup pressure | `CV4516` | `DROP_OF_MOVED_VALUE` | EVT1 number | maybe-moved Drop owner is rejected; transferred owner emits no drop |
| ref struct escape | lifetime system deferred | `CV4521` | `REF_STRUCT_ESCAPE` | EVT1-new | lifetime-bound aggregate cannot carry local provenance outward |
| scoped escape | lifetime system deferred | `CV4522` | `SCOPED_ESCAPE` | EVT1-new | explicit scoped reference/value cannot escape |
| outlives violation | lifetime system deferred | `CV4523` | `LIFETIME_OUTLIVES_VIOLATION` | EVT1-new | shorter lexical provenance cannot initialize longer-lived storage |
| lifetime semantic requirement | compiler-known marker pressure | `CV4524` | `LIFETIME_REQUIREMENT_UNSATISFIED` | EVT1-new | concept-requested analysis rejected the concrete type |
| invalid lifetime-bound field | reference aggregate deferred | `CV4525` | `REF_STRUCT_INVALID_FIELD` | EVT1-new | references/ref structs require a ref struct container |
| compiler analysis declaration | compiler-known marker pressure | `CV4526` | `COMPILER_ANALYSIS_REQUIREMENT_FAILED` | EVT1-new | unknown analysis or wrong fixed arity |
| disproven Outlives requirement | lifetime system deferred | `CV4527` | `OUTLIVES_REQUIREMENT_FAILED` | EVT1-new | selected result derives from a different semantic source |
| unknown Outlives provenance | lifetime system deferred | `CV4528` | `OUTLIVES_PROVENANCE_UNKNOWN` | EVT1-new | required proof rejects unknown rather than maybe-passing |
| unknown call-result lifetime | lifetime system deferred | `CV4529` | `CALL_RESULT_LIFETIME_UNKNOWN` | EVT1-new | no summary can justify outward assignment |
| call result escapes source | lifetime system deferred | `CV4530` | `CALL_RESULT_ESCAPES_SOURCE` | EVT1-new | helper boundary preserves the shorter actual source |
| scoped result escape | lifetime system deferred | `CV4531` | `SCOPED_RESULT_ESCAPE` | EVT1-new | derived call result retains scoped state |
| invalid relational subject | no source counterpart | `CV4532` | `RELATIONAL_REQUIREMENT_INVALID_SUBJECT` | EVT1-new | subjects must select one required operation parameter/result relation |
| unsupported relational requirement | no source counterpart | `CV4533` | `RELATIONAL_REQUIREMENT_UNSUPPORTED` | reserved EVT1-new | bounded subject model rejects unsupported relation shapes |
| invalid `?` operand | PoC3 concrete `try` operand pressure | `CV4541` | `QUESTION_REQUIRES_OPTION_OR_RESULT` | EVT1-new | parser succeeds; semantic type is diagnosed |
| incompatible propagation channel | PoC3 nominal Result mismatch pressure | `CV4542` | `QUESTION_INCOMPATIBLE_RETURN_CHANNEL` | EVT1-new | includes Option/Result mixing |
| Result error mismatch | PoC3 same-nominal propagation pressure | `CV4543` | `RESULT_ERROR_TYPE_MISMATCH` | EVT1-new | exact `E` only |
| invalid `!` operand | Phase 17 panic pressure only | `CV4544` | `BANG_REQUIRES_OPTION_OR_RESULT` | EVT1-new | escalation requires a carrier |
| unhandled local error | no closed counterpart | `CV4545` | `TRY_EXCEPT_UNHANDLED_ERROR` | EVT1-new | exact typed arm or identical outer Result required |
| runtime assert condition | PoC3 assert bool family | `CV4546` | `ASSERT_REQUIRES_BOOL` | both namespaces retained | reason must also be string |
| duplicate except error | no closed counterpart | `CV4547` | `TRY_EXCEPT_DUPLICATE_ERROR` | EVT1-new | duplicate concrete type is rejected statically |
| nontransferable failure payload | PoC3 non-copyable family | `CV4548` | `RESULT_PAYLOAD_NONTRANSFERABLE` | EVT1-new | explicit move required |
| immovable failure payload | R2 immovable law | `CV4549` | `FAILURE_PAYLOAD_IMMOVABLE` | EVT1-new | Option/Result do not weaken embedding law |
| unknown except type | no closed counterpart | `CV4550` | `TRY_EXCEPT_UNKNOWN_ERROR_TYPE` | EVT1-new | no runtime type test |
| bind without contextual target | no counterpart | `CV4562` | `BIND_REQUIRES_CONTEXTUAL_TARGET` | EVT1-new | destination type must supply shape |
| bind target is not storage reference | no counterpart | `CV4563` | `BIND_REQUIRES_ARRAY_TARGET` | EVT1-new | target is ref/ref const array or ndarray |
| bind source is not contiguous storage | no counterpart | `CV4564` | `BIND_REQUIRES_STORAGE` | EVT1-new | arbitrary pointers/scalars reject |
| bind element type differs | no counterpart | `CV4565` | `BIND_ELEMENT_TYPE_MISMATCH` | EVT1-new | exact identity; no reinterpretation |
| fixed bind count differs | no counterpart | `CV4566` | `BIND_SHAPE_SIZE_MISMATCH` | EVT1-new | compile-time exact-count proof |
| mutable bind from const | no counterpart | `CV4567` | `BIND_MUTABLE_FROM_CONST` | EVT1-new | ordinary const/ref law |
| malformed bind MIR | no counterpart | `CV4568` | `BIND_MIR_INVALID` | EVT1-new | compiler invariant, not source recovery |
| invalid layout declaration | SDSL-V graph pressure | `CV4570`-`CV4576` | `LAYOUT_*` | SDSL-derived / EVT1-new surface | unique fixed, aligned, non-overlapping regions; runtime parameters deferred |
| invalid layout bind | no direct counterpart | `CV4577`-`CV4579`, `CV4590`-`CV4591` | `LAYOUT_BIND_*` | EVT1-new composition | whole contiguous storage, const, exact bytes, and alignment |
| invalid stream declaration or bind | SDSL-V channel pressure | `CV4580`-`CV4586` | `STREAM_*` | SDSL-derived / EVT1-new composition | known layout/regions, unique channels, exact layout, no const escalation |
| invalid layout query or semantic MIR | semantic graph invariant | `CV4592`-`CV4594` | `LAYOUT_QUERY_INVALID` / `*_MIR_INVALID` | EVT1-new | bounded queries and complete layout/stream graphs |
| Span source is not contiguous | Slice construction pressure | `CV4600` | `SPAN_REQUIRES_CONTIGUOUS_SOURCE` | reconciled pressure / EVT1-new | only existing arrays, ndarrays, bound views, regions, channels, and spans |
| Span element mismatch | Slice element identity pressure | `CV4601` | `SPAN_ELEMENT_TYPE_MISMATCH` | reconciled pressure / EVT1-new | exact element identity; no reinterpretation |
| mutable Span from const | readonly Slice pressure | `CV4602` | `SPAN_MUTABLE_FROM_CONST` | EVT1-new | composes with ref/ref const and place constness |
| invalid Subspan interval | bounds pressure | `CV4603` | `SPAN_BOUNDS_OUT_OF_RANGE` | reconciled pressure / EVT1-new | constant half-open failure diagnosed; runtime path panics |
| invalid Span index | bounds pressure | `CV4604` | `SPAN_INDEX_OUT_OF_BOUNDS` | reconciled pressure / EVT1-new | constant failure diagnosed; runtime path panics |
| ReadOnlySpan mutation | readonly Slice pressure | `CV4605` | `READONLY_SPAN_MUTATION` | reconciled pressure / EVT1-new | no const stripping |
| Span arithmetic overflow | no closed counterpart | `CV4606`-`CV4607` | `SPAN_OFFSET_OVERFLOW` / `SPAN_LENGTH_OVERFLOW` | EVT1-new | runtime uses deterministic panic reasons |
| malformed Span MIR | semantic descriptor invariant | `CV4608` | `SPAN_MIR_INVALID` | EVT1-new | complete borrowed-region facts required |
| invalid tensor type/source | SDSL-V tensor pressure | `CV4610`-`CV4615` | `TENSOR_*` type/source/rank/element families | EVT1-new / SDSL-V-derived | positive rank, exact element identity, shaped contiguous source |
| invalid tensor destination/shape/index | no closed counterpart | `CV4616`-`CV4618` | `TENSOR_READONLY_DESTINATION` / `TENSOR_SHAPE_MISMATCH` / `TENSOR_INDEX_ARITY_MISMATCH` | EVT1-new | exact shape and zero-based rank-exact indexing; no broadcasting |
| invalid Einstein index relation | SDSL-V indexed-expression pressure | `CV4619`-`CV4622` | `TENSOR_SYMBOLIC_INDEX_*` / `TENSOR_OUTPUT_INDEX_INVALID` / `TENSOR_REDUCTION_INDEX_INVALID` | EVT1-new / SDSL-V-derived | statement-local symbols with exact extent and multiplicity laws |
| invalid generalized contraction | no closed counterpart | `CV4624`-`CV4625` | `TENSOR_CONTRACTION_SHAPE_MISMATCH` / `TENSOR_ALIAS_HAZARD` | EVT1-new | last/first axis equality and conservative destination disjointness |
| malformed Tensor MIR | semantic operation invariant | `CV4626` | `TENSOR_MIR_INVALID` | EVT1-new | complete rank/shape/region/provenance/lowering facts required |
| inline tensor runtime shape | no counterpart | `CV4627` | `TENSOR_INLINE_REQUIRES_FIXED_SHAPE` | EVT1-new / R4h-derived | runtime/external backing must remain explicit |
| vector shaped rank mismatch | no counterpart | `CV4628` | `VECTOR_RANK_MISMATCH` | EVT1-new | exact rank-one shorthand |
| matrix shaped rank mismatch | no counterpart | `CV4629` | `MATRIX_RANK_MISMATCH` | EVT1-new | exact rank-two shorthand |
| invalid inline shape | fixed ndarray pressure | `CV4630` | `TENSOR_INLINE_SHAPE_INVALID` | EVT1-new / reused law | every inline extent is positive and fixed |
| inline fill element mismatch | tensor exact element law | `CV4631` | `TENSOR_INLINE_ELEMENT_TYPE_MISMATCH` | EVT1-new / reused law | no scalar conversion or broadcasting |
| scalar contraction destination mismatch | scalar type law | `CV4632` | `TENSOR_SCALAR_RESULT_TYPE_MISMATCH` | reserved R4i family | dot result is exact element scalar |
| unsatisfied or unknown semantic fact | no direct counterpart | `CV4640`-`CV4641` | `SEMANTIC_FACT_UNSATISFIED` / `SEMANTIC_FACT_UNKNOWN` | EVT1-new | unknown never satisfies a required proof |
| invalid semantic subject or parameter | bounded analysis subject precedent | `CV4642`-`CV4643` | `SEMANTIC_FACT_INVALID_SUBJECT` / `SEMANTIC_FACT_INVALID_PARAMETER` | EVT1-new / R4b-derived | alignment is positive power-of-two through 4096 |
| specialized fact failure | no direct counterpart | `CV4644`-`CV4648` | alignment/disjoint/contiguity/shape/mutability requirement families | EVT1-new | preserves useful semantic diagnostic families |
| malformed semantic fact MIR | MIR invariant precedent | `CV4649` | `SEMANTIC_FACT_MIR_INVALID` | EVT1-new | typed subject, origin, certainty, and stable unique ID required |
| invalid or duplicate class member | Phase 7 member pressure | `CLASS_DUPLICATE_MEMBER`, `CLASS_METHOD_RECEIVER_INVALID`, `CLASS_MEMBER_TYPE_INVALID` | same semantic family | EVT1-new names | class validation is independent of name-resolution accidents |
| private class access | no matching closed Phase 14 access family | `CLASS_PRIVATE_MEMBER_ACCESS` | `CLASS_PRIVATE_MEMBER_ACCESS` | EVT1-new | covers external field and method access |
| unsatisfied interface method | Phase 14 signature/implementation diagnostics | `CV4153`, `CV4156` | `INTERFACE_REQUIREMENT_UNSATISFIED` family | preserved Go numbers | missing and wrong-signature operations reuse ordinary concept matching |
| unsatisfied interface field | no direct counterpart | `INTERFACE_REQUIREMENT_UNSATISFIED` | same semantic family | EVT1-new name | exact visible field type required |
| private-only interface satisfaction | Phase 14 implementation visibility pressure | `INTERFACE_PRIVATE_MEMBER_CANNOT_SATISFY` | same semantic family | EVT1-new name | public contract capability cannot be supplied privately |
| invalid interface runtime shape | Phase 14 dyn compatibility pressure | `INTERFACE_NOT_DYN_COMPATIBLE` | same semantic family | Redesigned | rejects erased returns/parameters and invalid receivers |
| malformed interface witness MIR | Phase 14 MIR validation pressure | `INTERFACE_WITNESS_INVALID` | same semantic family | Redesigned | requires unique type-pair identity, entries, and no-allocation law |
| dyn target/concrete mismatch | Phase 14 coercion diagnostics | `DYN_REQUIRES_INTERFACE`, `DYN_CONCRETE_TYPE_DOES_NOT_SATISFY` | same semantic families | Redesigned | explicit ref conversion reuses concept satisfaction |
| dyn const violation | ownership qualifier pressure | `DYN_MUTABLE_FROM_CONST`, `DYN_READONLY_FIELD_MUTATION` | same semantic families | EVT1-new | no mutation capability is recovered through erasure |
| unknown dyn member | Phase 14 interface-call diagnostics | `DYN_METHOD_NOT_IN_INTERFACE`, `DYN_FIELD_NOT_IN_INTERFACE` | same semantic families | Redesigned | witness surface is closed by the declared interface |
| dyn lifetime escape | no closed Phase 14 lifetime system | `CV4511` and existing scoped/outlives families | `DYN_ESCAPE_SOURCE_LIFETIME` family | reused provenance diagnostics | dyn has no separate lifetime checker |

`static_assert` continues to reuse the bounded comptime diagnostic path:
runtime calls are `CV4210`, a non-boolean evaluated condition is `CV4207`, and
a false assertion is `CV4207` with its deterministic reason. R4c deliberately
does not add a second static-assert evaluator or mass-renumber those families.

Runtime bind failures use stable terminal reasons rather than recoverable
diagnostics: `Concept bind shape does not match storage size` for unequal
runtime totals and `Concept bind shape product overflow` for invalid or
overflowing runtime multiplication. `bind` does not introduce a Result channel
or exception family.

`VALUE_TYPE_MISMATCH` maps the existing Go `CV4107` assignment family to
PoC3 `CON0078`/`CON0082` aggregate and field mismatch evidence.
`EXPLICIT_LOCAL_TYPE_REQUIRED` maps the typed-`let` rejection path at
`CV4009`; it does not introduce local type inference.

Final numeric namespace selection remains deferred until a broader diagnostic
reconciliation can account for source compatibility, tooling, and all retained
PoC3 families.
