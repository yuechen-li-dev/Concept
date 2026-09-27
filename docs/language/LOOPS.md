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

There is no C-style `for` loop. Finite `start..end` ranges, `step`, `descend`,
`break`, and `continue` are not yet accepted by this checkout. They require a
range/control-flow representation that can be validated and lowered across
ordinary functions, machines, and async code. Until then, counted iteration
uses `while (...) bounded(limit)` where boundedness is required.
