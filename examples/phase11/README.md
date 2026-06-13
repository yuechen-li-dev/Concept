# Phase 11 Testing Examples

These examples show the closed Phase 11 v0 testing surface:

- `.con_test` test modules;
- `[Fact]`, `[Theory]`, and `[InlineData(...)]`;
- mandatory positional reason strings on every `Assert.*`, `Expect.*`, and
  `Expect.That` call;
- primitive `Expect.Equal(int, int)`, `Expect.Equal(bool, bool)`, and
  `Expect.That` relations with `Is.True()`, `Is.False()`, `Is.EqualTo(int)`,
  and `Is.EqualTo(bool)`.

Named `because:` syntax is deferred. The current examples intentionally use the
positional reason bridge:

```cpp
Expect.That(actual, Is.EqualTo(expected), "reason");
```

The current runner is the internal HIR v0 runner. CLI/from-disk `.con_test`
execution is deferred unless a future milestone wires the runner into a public
command.
