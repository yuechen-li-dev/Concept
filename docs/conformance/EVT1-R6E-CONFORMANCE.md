# EVT1 R6e conformance: reusable semantic modules

Status: success

Baseline HEAD: `1f16339b6ab228fce12866b3fade0d451b921f66`

Compiler identity: `concept-evt1-stage0-go`

R6d commit: `1f16339b6ab228fce12866b3fade0d451b921f66`

## Delivered boundary

R6e adds canonical `module`/`import` parsing, deterministic
`concept-module.v1`, exact configured-root resolution, DAG/cycle/duplicate and
integrity validation, source-staleness rejection, automatic test dependency
build ordering, and CLI `build-module`. Imported source is not read by ordinary
consumer compilation.

The artifact preserves ordinary declarations and source spans, aliases,
concepts/interfaces, generic type/function/method bodies, constraints, integer
non-type parameters, ownership/Drop and provenance inputs, normalized
signatures, type summaries, and operation effects. It contains no generated C,
object code, runtime metadata, generic dictionary, or backend realization.

## Semantic evidence

Focused tests cover:

- 100 byte-identical builds of a representative artifact;
- imported generic structs, classes, functions, methods, and fixed storage;
- imported constraints checked against valid and invalid consumer types;
- imported structural Drop exactly once and ordinary `CV4521` ref escape;
- `Allocates`, proven `NoAllocation`, and `Unknown` imported summaries;
- R6b proof origin `ModuleSummaryEffect`;
- imported interface allocation allowance and mismatch rejection;
- missing, cycle, schema, corruption, stale, and duplicate diagnostics;
- exact filesystem resolution and automatic `.concept_test` DAG builds;
- combined output compiled as strict C11, linked to an external C adapter, and
  executed successfully.

Fixtures live at `language/evt1/tooling/modules/` as separate module, consumer,
negative constraint/provenance, effect, interface, and test sources.

## Allocator blocker checklist

| Original R6c blocker | R6e status |
|---|---|
| User-defined generic runtime types | Resolved by R6d and reusable through R6e artifacts |
| `SizeOf<T>` / `AlignOf<T>` | Resolved across imported generic instantiations |
| Generic owner/Drop/provenance | Resolved across imports through structural analysis |
| Reusable multi-module compilation | Resolved by `concept-module.v1` and exact roots |
| Hosted external ABI seam | Resolved by bounded `extern "C"`, preserved across modules |
| Operation effects | Resolved by summaries, proof propagation, and interface allowance |

The prerequisite substrate exists; this does not claim an allocator library.

## Deferred boundaries

Package management, remote registries, import aliases, partitions, stable
public module ABI, generalized effects, variadic templates, partial
specialization, and native/LIR backends remain deferred. R6e adds no allocator,
GC, heap policy, runtime loader, or reflection registry.
