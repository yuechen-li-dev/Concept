# PoC3 fixture migration inventory

Status: R0 inventory; no bulk port attempted

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

The principal missing families are PoC3 judgment/decide, full move/drop,
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
