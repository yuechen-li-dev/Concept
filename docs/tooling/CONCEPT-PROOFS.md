# Concept semantic proofs

`Assert.Concept` is a permanent compile-time source contract over facts the
compiler already knows. It does not evaluate its subject, emit a runtime check,
allocate, or lower to C.

```concept
Assert.Concept<Aligned<16>>(
    weights,
    "cooperative matrix path requires 16-byte alignment"
);
```

The final argument is a nonblank string literal. Earlier arguments name local
values, types, or non-overloaded operations. Relational analyses such as
`Disjoint` and `Outlives` take two subjects. Other expressions are rejected
rather than evaluated.

## Outcomes and graphs

- `PROVEN`: evidence closes every displayed obligation.
- `DISPROVEN`: known facts contradict an obligation.
- `UNKNOWN`: evidence is missing; this is not false.

Failures show the reason and a graph made from `Goal`, `Requirement`, `Subgoal`,
`KnownFact`, `DerivedFact`, `MissingFact`, `Contradiction`, `Dependency`, and
`Subject` nodes. Edges are `Requires`, `DerivedFrom`, `DependsOn`,
`ConflictsWith`, and `BlockedBy`. Default output is sliced and bounded to depth
eight and eight sibling branches. Verbose and JSON output expose the complete
bounded assertion graph.

## Explain CLI

```text
concept explain path/to/file.concept
concept explain path/to/file.concept:42
concept explain path/to/file.concept --verbose
concept explain path/to/file.concept --json
```

Explain inspects an existing assertion and does not mutate source semantics.
`--json` emits deterministic `concept-proof.v1`: goal, outcome, subjects,
ordered nodes and edges, failed/unknown node IDs, repair classes, reason, and
source location. IDs derive from semantic identity, relation, order, and source
position, never process addresses. Arbitrary ad-hoc query syntax is deferred.

## Examples and repair boundary

A callable capturing `ref const config` from a parameter and `ref scratch` from
a local proves `config` and identifies `scratch` as the poisoning provenance.
Copy capture is offered only for copyable types and move only for movable types.
A captured span or tensor also names its preserved backing-storage lifetime
dependency; the view is not diagnosed as independent owned storage.

An offset-one `Span<int>` shows effective alignment 4 against required 16. Two
spans with one backing region but no fixed non-overlap evidence make `Disjoint`
`UNKNOWN`, not `DISPROVEN`.

`NoAllocation(Operation)` follows local calls. A local leaf is proven because
EVT1 has no allocation-capable construct. External, overloaded, and recursive
summaries remain `UNKNOWN`; the compiler does not invent an allocating edge.

Repairs are semantic classes licensed by graph rules, not guaranteed rewrites.
An IDE or LLM may later rank them in broader context, while compiler facts,
edges, and legal repair classes remain authoritative. There is no network or
LLM dependency in the compiler.

`static_assert` proves a compile-time boolean. `Assert.Concept` queries semantic
proof authority and preserves its proof structure.

## Transported value facts

R6h proof evidence may include `Transported through` steps such as local copy,
field store/load, Result or Option extraction, `AddOffset`,
`CommonAlignment`, imported result return, capture, async persistence, and
machine persistence. These are compile-time provenance for the proof, not
runtime metadata.

For region relations, equal origins plus known half-open intervals can prove
non-overlap or disprove it. A runtime offset remains `UNKNOWN`. When an
imported or external operation has no semantic result summary, the proof names
the `Fact lost at` boundary and does not infer facts from ABI or runtime shape.
The checked R6h goldens cover alignment through multiple transports, returned
disjoint regions, returned overlap, and an opaque `AddressFromBits` boundary.
