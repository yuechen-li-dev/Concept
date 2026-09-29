# Phase 11 failing output example

This is documentation, not a passing example fixture. Runtime-failing tests are
runner failures, not compiler-invalid fixtures.

```cpp
module Examples.Phase11.FailingOutput;

import Core.Test;

int Add(int a, int b) {
    return a + b;
}

[Theory]
[InlineData(10, 20, 30)]
void AddsInlineData(int a, int b, int expected) {
    Expect.That(
        Add(a, b) - 1,
        Is.EqualTo(expected),
        "add should return the arithmetic sum of both operands"
    );
}
```

Stable v0 runner output:

```text
FAIL tests=1 failures=1

FAILED Examples.Phase11.FailingOutput.AddsInlineData#0

Expect.That failed

Source:
  span start=<span-start> length=<span-length>

Because:
  add should return the arithmetic sum of both operands

Expected:
  30

Actual:
  29
```
