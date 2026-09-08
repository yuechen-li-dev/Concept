# PoC3 fixture migration inventory

Status: R5b bounded migration evidence; no bulk port attempted

## Corpus snapshot

At pre-cutover Concept commit
`e60f9012d001474692112873cfd6e0f09671bd38`, the repository contains:

- 1,296 primary source fixtures under `language/`;
- 712 primary fixtures in `valid/` families;
- 584 primary fixtures in `invalid/` families;
- 26 additional Concept-like inputs under `tests/corpus/`;
- 88 additional Concept-like inputs under `examples/`;
- 1,478 tracked `.concept`, `.conception`, or `.con_test` inputs in total.

The 1,296 primary fixtures are distributed as follows:

| Family | Count | Initial migration pressure |
|---|---:|---|
| phase1-surface | 5 | syntax translation required |
| phase2-execution | 73 | syntax translation required; core candidate |
| phase3-semantics | 50 | syntax translation required; core candidate |
| phase4-mir | 3 | oracle translation required |
| phase5-sum-types | 50 | syntax translation required; core candidate |
| phase5a-judgment | 19 | semantic feature missing in Go EVT1 |
| phase6-unsafe-ownership | 19 | deferred reconciliation |
| phase7-runtime-structs | 42 | syntax translation required; core candidate |
| phase8-concepts-templates | 72 | syntax and constraint-subset review required |
| phase9-compile-time | 75 | fuel/capability/subset review required |
| phase10-ownership | 54 | semantic feature missing in Go EVT1 |
| phase11-testing | 85 | semantic feature missing in Go EVT1 |
| phase12-allocation | 83 | semantic feature missing in Go EVT1 |
| phase13-machines | 71 | R5b translates hierarchy/basic transition/Step/decide; result remains deferred |
| phase14-interfaces | 97 | semantic feature missing in Go EVT1 |
| phase15-c-abi | 108 | semantic feature missing in Go EVT1 |
| phase16-imports | 73 | multi-module implementation missing |
| phase17-runtime-failure | 57 | semantic feature missing in Go EVT1 |
| phase18-machines | 66 | R5b translates machine fields/basic Step/match/decide; nested values/completion/result deferred |
| phase19-yielding-machines | 23 | R5a adopts explicit persistence/re-entry law; yield surface deferred to R5c |
| phase20-dragongod-kernel | 99 | mixed core, automata, and application pressure |
| phase21-arrays-slices-fixed-buffers | 50 | fixed-array subset review; slices/buffer missing |
| phase22-option-result-bounded-collection-mutation | 22 | Option exists only in retired line; remainder deferred |

Counts describe the fixture families, not accepted EVT1 promises. Several
families contain both positive and negative cases, so readiness and oracle kind
are independent dimensions and should not be summed as disjoint totals.

## Migration classes

### Portable to EVT1 now

The extracted `examples/evt1` suite is the current executable Go baseline. The
new `core_language.concept` proves a non-Vulkan module through the active CLI.
PoC3 fixtures are not labeled byte-for-byte portable until R1 runs them through
an explicit translator or hand-authored equivalent; the two parsers use
materially different surface forms.

### Syntax translation required

PoC3 surface, execution, structs, sum types, concepts/templates, and comptime
families overlap the Go compiler semantically but may use `fn`, `name: Type`,
`->`, `let`, `var`, module spelling, construction spelling, or match forms that
are not canonical EVT1. These fixtures should retain their original text and
gain separate EVT1 counterparts rather than being rewritten in place.

### Semantic feature missing in Go EVT1

The principal missing families are PoC3 judgment/decide, ownership beyond the
R3 bounded local move/drop/reference foundation,
testing, allocation, interfaces/dyn, C ABI, multi-module resolution,
panic/assert, yield, runtime arrays/slices/FixedBuffer, and Option. Result and
bounded collection mutation were not complete in PoC3 and remain deferred;
their roadmap position is not authority to implement them in R0.

### Legacy-only

No entire family is declared legacy-only in R0. Individual fixtures may become
legacy-only after a reconciliation decision rejects their behavior. That
classification requires a matrix decision and an EVT1 diagnostic oracle, not a
directory-name inference.

### Profile-specific

The extracted Vulkan examples are profile-specific even when their filenames
contain `language`; they all explicitly select `profile Vulkan;`. Effect,
actuator, Prometheus import, and Vulkan runtime-type cases must remain behind
the Vulkan profile. R5b now supplies canonical Core automata hierarchy,
explicit shared/machine persistence, direct Step, basic transition,
categorical transition match, and hardmax transition decide cases;
signal/effect, yield, nested-machine, and completion/result fixtures remain
separately classified rather than bulk-ported.

### Invalid and diagnostic oracles

The 584 `language/**/invalid/` inputs are valuable semantic assets. Migration
must preserve the reason for rejection, not merely any failure. R1 should map
stable PoC3 `CON` diagnostics to the retained Go `CV` diagnostics or to a new
EVT1 code only after the semantic decision is recorded.

## Differential testing recommendation

R1 should establish a table-driven harness with one record per selected case:

```text
PoC3 source and expected outcome
  -> retired Zig compiler

explicit EVT1 translation and expected outcome
  -> active Go compiler

comparison
  -> agreed semantic result, diagnostic class, MIR fact, or generated-C fact
```

Start with a bounded representative set: declarations, ordinary functions,
control flow, structs, payload enums, exhaustive match, concepts, one template
specialization, and bounded comptime. Do not bulk-convert all 1,296 primary
fixtures before the translation rules and oracle mapping are proven.

## R1 bounded inventory

R1 inspected 17 distinct PoC3 reference fixtures across phases 2, 3, 5, 7,
8, 9, and 21. Sixteen canonical EVT1 sources retain direct PoC3 provenance;
one canonical source intentionally combines two closely related array
references. Five additional Core-denial sources use the R0 matrix and Vulkan
constitution as their authority, for 21 new `.concept` sources total.

| R1 classification | Count | Meaning |
|---|---:|---|
| reference fixtures inspected | 17 | Distinct PoC3 source oracles read for the bounded subset. |
| canonical EVT1 sources with PoC3 provenance | 16 | Hand translations; original fixtures are unchanged. |
| portable semantic references | 17 | Meaning can be tested after explicit surface translation. |
| references requiring syntax translation | 17 | Module/profile, declaration, constraint, comptime-loop, aggregate, or match spelling differs. |
| selected references blocked by missing Go features | 0 | Deferred feature families were not selected into the R1 harness. |
| selected references classified legacy-only | 0 | No R1 matrix decision rejected an entire selected behavior. |

The one executable disagreement is the historical unannotated compile-time
loop. It is classified `EXPECTED-DIVERGENCE`, because canonical EVT1 requires
`bounded(limit)`. It is not counted as an EVT1 feature gap.

## R2 value/place audit

R2 audited all 42 Phase 7 runtime-struct fixtures and all 54 Phase 10
ownership fixtures as reference families, then selected the narrow field,
copy, assignment, and non-copyable-flow cases named in the R2 conformance
metadata. Phase 7 supplies direct field-place and type-mismatch diagnostics.
Phase 10 supplies `CON0154`/`CON0160` pressure, but its opt-in `Copy` law differs
from EVT1's selected structural-copy rule for ordinary structs.

PoC3 has no `record struct` or `with` fixture syntax, and its archived coverage
ledger marks `immovable` as not implemented. R2 therefore uses those sources
only as design pressure and records executable immovable evidence from the Go
M1B-A lineage. New record/update cases are EVT1-new evidence rather than fake
legacy translations.

## R3 transfer and reference audit

R3 reuses Phase 10 explicit-move, use-after-move, branch-join, call/return
transfer, early-return cleanup, moved-local suppression, and reverse-drop-order
fixtures as semantic evidence. EVT1 keeps the R2 structural-copy decision, so
movable-only behavior is expressed by the already-parsed `owned T` qualifier
rather than making every ordinary struct non-Copy.

The R3 corpus contains explicit hand-authored EVT1 sources; no PoC3 fixture was
rewritten. Reference cases are EVT1-new evidence because PoC3 did not close the
selected `ref T` / `ref const T` surface. The old `Slice<T>` family is now
classified as redesign pressure for future lifetime-bound Span work, not as a
direct-port queue.

## R4a lifetime and witness audit

R4a inspected PoC3 ownership/reference pressure and the Phase 8/14
concept/template/interface implementation. PoC3 provides useful witness and
runtime-erasure pressure, but it does not provide the selected lexical
provenance, `ref struct`, `scoped`, or compiler-analysis-requirement semantics.
Those cases are EVT1-new evidence.

Phase 14 interface/dyn fixtures are no longer a mechanical port queue. Their
runtime dispatch behavior remains design pressure for a witness model in which
ordinary concept satisfaction produces the compile-time evidence that may
later be reified for `dyn C`. R4a does not port interface declarations, vtables,
or dyn storage. Slice fixtures likewise remain deferred until Span is designed
against the now-active lifetime machinery.

## R4k Phase 14 evidence audit

R4k closes the selected borrowed-dispatch pressure with separately authored
EVT1 sources. Phase 14's 97-fixture family established evidence for interface
declarations, implementation matching, erased storage, interface calls, and
diagnostics. Its nominal implementation records and broader legacy object
model were not copied.

The active design instead makes `interface` a specialized ordinary concept,
derives witnesses through existing satisfaction, and represents `dyn I` as a
non-owning object pointer plus static witness pointer. Class encapsulation,
interface field accessors, composition through prerequisites, semantic-fact
proof erasure, and provenance-preserving dyn are classified EVT1-new/redesign.
Owning dyn remains deferred; inheritance is rejected. The original PoC3
fixtures and compiler remain unchanged and continue to serve only as evidence.

## R4c failure-model audit

R4c inspected Phase 5 concrete Result-shaped propagation, Phase 17
panic/assert and stable reasons, and Phase 22 compiler-known Option plus Option
match. These are evidence and pressure, not executable authority. Option is
adopted with canonical qualified constructors. PoC3's concrete nominal
Result-shaped enums and prefix `try` inform early-return lowering, but EVT1
uses generic `Result<T,E>`, postfix `?`, and reserves `try`/`except` for a local
typed boundary. No closed PoC3 evidence was found for generic Result,
heterogeneous exact handlers, implicit error conversion, or unwinding; those
R4c cases are recorded as EVT1-new rather than parity.

## R4d contiguous-storage audit

R4d inspected the complete Phase 21 fixed-array, array-literal, indexing,
mutable-indexing, `Len`, bounds, value-copy, parameter/return/field, Slice, and
FixedBuffer evidence. Fixed arrays contribute durable semantic and C-wrapper
pressure: Concept arrays remain assignable values, indexing is checked, and C
array decay is not language law. These cases inform the canonical rank-1
`array` family.

PoC3 Slice and FixedBuffer are not ported. They remain pressure for the later
borrowed-view and bounded-collection layers after explicit storage provenance
and construction are designed. PoC3 did not implement the R4d ndarray family,
comma-separated rank-aware indexing, one-object row-major linearization, or
`Rank`/`Shape`; those cases are recorded as EVT1-new/SDSL-V-derived evidence.
Runtime shape expressions are accepted as semantic type facts only far enough
to issue the explicit-storage diagnostic. No PoC3 allocation behavior is used
to fabricate an implicit allocator in EVT1.

## R5b Phase 18 transition audit

R5b inspected Phase 13 syntax scaffolding and Phase 18 executable match/decide
fixtures and MIR lowering. The retained semantic core is deterministic:
runtime categorical selection, declaration-order guards and enabled scores,
strict-greater-than hardmax replacement, first-source-order ties, distinct
duplicate candidates, and stable terminal failure when no candidate is
enabled. The stable no-match and no-enabled reason strings are preserved.

EVT1 does not copy PoC3's bool-only Phase 18 match checker or standalone
machine-frame syntax. `transition match` instead delegates to EVT1's ordinary
closed payload-enum/Option/Result pattern machinery inside the R5a automata
hierarchy. `transition decide` adds exact float scores and deterministic NaN
rejection while keeping the PoC3 hardmax behavior. DragonGod's later stateful
Decision fixtures, yield, Complete/Result, and plain value-level decide are not
R5b parity targets.
