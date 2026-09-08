# EVT1 iterator and foreach direction

Status: R5d bounded protocol and mechanical consumption implemented

An iterator is a small explicit state machine. A source participates through
`GetIterator`, its iterator advances through `MoveNext`, and the current value
is observed through `Current`. The compiler-known array, ndarray, Span, and
ReadOnlySpan adapters use inline index/source state. Custom iterators use the
same ordinary typed free-function protocol. Neither path allocates implicitly.

`foreach (T item in source)` evaluates source once, obtains iterator state
once, advances once per attempt, observes once per successful iteration, and
recreates the item binding each time. Value iteration copies only copyable
elements. `ref T` is limited to mutable sources; `ref const T` preserves
readonly provenance. Iterator and item cleanup use ordinary reverse lexical
cleanup on normal and terminal control-flow edges.

Arrays and ndarrays borrow their backing storage rather than copy it. Ndarray
order is linear row-major scalar order. Span iteration carries its existing
pointer, length, provenance, bounds, and constness facts. The Planner records
direct-index eligibility for contiguous builtins but R5d retains the iterator
semantic plan as authority.

Inside an automata state, foreach locals are transient. A `yield;` in its body
ends the Step and drops that transient iterator; re-entry starts the state and
foreach again. To resume iteration across Steps, authors store iterator state
explicitly in automata or machine persistent storage and call the protocol
operations directly.

```text
foreach automates iterator state progression.
yield automates machine Step progression.
Neither requires a coroutine runtime.
```

Generator-style `yield value`, associated-type machinery, ranges, LINQ-style
pipelines, lazy runtime registries, and implicit iterator persistence are
deferred. They are not extensions of the R5d bare-yield law.
