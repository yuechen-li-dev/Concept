# EVT1 R6b semantic proof debugging

## Baseline

- Baseline and R6a commit: `81b1fc41003130d28b6f2ae4d84e3de71bbdddfa`
- Compiler: `concept-evt1-stage0-go`
- Go: `go1.27.0 windows/amd64`; Zig: `0.16.0`
- Worktree was clean.
- `go test ./...`, `go vet ./...`, root `zig build test`, and
  `legacy/poc3-zig/zig build test` passed.

## Surface and authority

R6b adds `Assert.Concept<Goal>(subjects..., "reason")` to function bodies in
`.concept` and `.concept_test`. Parameterized analyses use existing nested
spelling such as `Aligned<16>` and `Rank<2>`. Subjects are named values, types,
or non-overloaded operations and are never evaluated.

```text
SemanticFacts / concept requirements / lifetime provenance
    -> authoritative Proven / Disproven / Unknown result
    -> deterministic ProofGraph projection
    -> assertion diagnostic / concept explain / concept-proof.v1
```

Proof tooling does not re-derive semantics or consult Planner. Proven assertions
retain a no-allocation/no-copy/no-transfer MIR marker and emit no C statement.
Ordinary syntax and type failures remain ordinary diagnostics.

Graph IDs hash semantic goal, node role, source order, relation label, and source
position. Ordered slices make JSON deterministic. Human output is bounded to
depth eight and eight sibling branches; verbose/JSON retain the complete graph.

## Evidence specimens

- Callable lifetime proves parameter capture `config` and disproves local ref
  capture `scratch`, the shortest poisoning provenance.
- Span lifetime identifies the captured view's backing-storage lifetime
  dependency rather than treating the view as independent storage.
- Async lifetime projects `ref source -> Async persistent field -> operation
  lifetime` from the existing persistence validator.
- Alignment reports offset-one span alignment 4 versus required 16.
- Disjoint reports `UNKNOWN` for one region without fixed interval evidence.
- Rank reports known rank 2 versus required rank 1.
- NoAllocation proves the local `ProcessAudio -> Filter` graph.
- `MatrixReady` expands declared `Aligned` and `Rank` requirements.
- Interface failure identifies the missing `Reset(ref Device)` signature.

Repairs are graph-derived. Callable copy and move candidates are filtered by
known copyability and movability. Failure makes `concept check` and test
compilation nonzero; successful sync and async `[[fact]]` fixtures execute via
the existing strict-C11 runner. Focused tests serialize proven, disproven,
unknown, and NoAllocation graphs 100 times each and require byte identity.

## Deliberate boundaries

EVT1 has no authoritative allocation operation or effect summary that can
truthfully produce `DISPROVEN NoAllocation`; missing summaries remain unknown.
`NoCopy` and `NoOwnershipTransfer` retain their operation-preservation meaning
rather than being relabeled as type copyability.

Deferred: source rewriting, IDE/LSP graph visualization, arbitrary proof query
language, solver edits, LLM-ranked repairs, generalized effects, Planner
explanations, and module-scope placement. No formatter, lint engine, SMT solver,
global borrow checker, scheduler, runtime registry, heap, LIR, or native backend
is added.
