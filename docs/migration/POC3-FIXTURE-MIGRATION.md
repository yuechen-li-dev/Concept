# PoC3 fixture migration inventory

Status: R3 bounded migration evidence; no bulk port attempted

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
| phase13-machines | 71 | machine/automata reconciliation required |
| phase14-interfaces | 97 | semantic feature missing in Go EVT1 |
| phase15-c-abi | 108 | semantic feature missing in Go EVT1 |
| phase16-imports | 73 | multi-module implementation missing |
| phase17-runtime-failure | 57 | semantic feature missing in Go EVT1 |
| phase18-machines | 66 | machine/automata reconciliation required |
| phase19-yielding-machines | 23 | semantic feature missing in Go EVT1 |
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
the Vulkan profile. General automata cases remain provisional pending machine
reconciliation.

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
