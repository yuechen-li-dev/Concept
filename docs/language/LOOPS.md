# Structured iteration

`for (item in source)` is the canonical spelling for runtime container
iteration. The existing `foreach (Type item in source)` spelling remains a
compatibility alias to the same AST, validation, MIR, and C11 lowering.

```concept
for (value in values)
{
    total = total + value;
}
```

The item type may be omitted when it is the source element type. An explicit
`ref T` item borrows each element, while `ref const T` borrows it read-only.
Value iteration copies only copyable elements. Arrays and spans use inline
counted iteration without a runtime iterator allocation. Other iterable
sources use the existing `GetIterator`/`MoveNext`/`Current` protocol.

There is no C-style `for` loop or `break`/`continue`. Finite `start..end`
ranges are values: the start is included and the end excluded. For example,
`for (index in 0..32)` performs at most 32 iterations. `start..end step n`
and `start..end descend n` use a positive step, with runtime guards when an
endpoint or step is not known at compile time. Range loops use the same
inline, allocation-free iteration path as arrays and spans, including in
machines and async code. A loop that must stop on a runtime condition can use
`while (condition) bounded(limit)`.
