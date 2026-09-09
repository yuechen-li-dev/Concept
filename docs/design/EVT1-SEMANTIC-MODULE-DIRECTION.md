# EVT1 semantic module direction

Status: R6e implemented

## Authority pipeline

```text
Source Module
  -> parse and ordinary semantic analysis
  -> normalized declarations, generic bodies, contracts, and summaries
  -> concept-module.v1

Consumer
  -> exact local-root resolver
  -> integrity and compatibility checked semantic artifact loader
  -> ordinary generic instantiation, concept proof, and effect propagation
  -> MIR -> Planner -> backend
```

The typed Module representation remains the single semantic declaration model.
R6e serializes that representation; it does not introduce a second authoring
IR. Portable module/source identities replace machine-local paths in payloads.
Source spans remain attached to declarations for imported diagnostics.

Generic bodies must remain semantic because their concrete local type argument
does not exist when the library is built. Monomorphization therefore occurs in
the final consumer context, once per canonical application. Target-independent
layout declarations and ownership/provenance law travel in the artifact;
target realization does not.

Effect summaries are deliberately closed to `Allocates`, proven
`NoAllocation`, and `Unknown`. The library build derives transitive summaries;
the consumer trusts the checked artifact summary and the proof graph identifies
that origin. This avoids both source reanalysis and fake certainty.

Dependency identities and content hashes make a module graph deterministic and
closed. Cache presence is not semantic. A future build system may cache the
content-addressed result without changing this contract.

This is an internal semantic compatibility boundary for the current compiler,
not a stable binary-distribution promise. Compiler or schema mismatch rejects.
Package solving, registries, partitions, and long-term ABI evolution remain
future build/package work.
