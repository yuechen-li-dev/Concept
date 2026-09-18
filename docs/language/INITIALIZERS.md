# Initializers

`...` is initializer repetition. It is accepted only after an element inside
an array initializer.

```concept
int<array>[4] exact = [7 ... 4];
int<array>[4] remainder = [1, 2, 0 ...];
int<array>[8] segments = [1 ... 2, 2 ... 3, 3 ...];
```

`value ... N` repeats the initializer exactly `N` times. `N` must be a
compile-time nonnegative integer; zero is valid. `value ...` fills the
remaining statically known destination extent and must be the final element.
Overflow is rejected rather than truncated, and ordinary exact-initialization
underfill rules remain unchanged.

`Initializer ... N` means `N` semantic initializations. It does not mean
construct once and copy or `memcpy` the result. Calls and other side effects
execute once per resulting element. Repeating a move or other initializer that
cannot be independently legal is rejected.

`...` is not spread. It never unpacks or splices another container, has no
general expression precedence, and introduces no range or iterable-expansion
semantics. The typed AST retains one compact `RepeatInitializer` plus its
semantic count; strict C11 lowering uses a loop, so large repetition does not
create proportional AST nodes or duplicated generated statements.

