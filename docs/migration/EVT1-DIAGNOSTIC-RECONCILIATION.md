# EVT1 diagnostic reconciliation

Status: R2 bounded value/place slice

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

`VALUE_TYPE_MISMATCH` maps the existing Go `CV4107` assignment family to
PoC3 `CON0078`/`CON0082` aggregate and field mismatch evidence.
`EXPLICIT_LOCAL_TYPE_REQUIRED` maps the typed-`let` rejection path at
`CV4009`; it does not introduce local type inference.

Final numeric namespace selection remains deferred until a broader diagnostic
reconciliation can account for source compatibility, tooling, and all retained
PoC3 families.
