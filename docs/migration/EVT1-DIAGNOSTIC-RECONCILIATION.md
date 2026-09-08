# EVT1 diagnostic reconciliation

Status: R4f semantic layout and stream semantics

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
