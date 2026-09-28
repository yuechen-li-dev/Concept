# Expressions

R8e adds `discard expression;` as an explicit statement. It evaluates the
expression exactly once, retains its effects, and drops a returned owned value
when required. A `[[must_use]]` type or value-returning function makes an
ignored expression-statement result a semantic error. `Result<T, E>` is
MustUse by its compiler-owned declaration. `discard` explicitly acknowledges
that the result is intentionally unused.

The existing `discard(batch)` effects operation is a call and keeps its
established meaning.

Concept uses C/C++ operator precedence. From strongest to weakest within the R7d3 integer and logical surface:

1. unary operators
2. `*`, `/`, `%`
3. `+`, `-`
4. `<<`, `>>`
5. comparisons
6. equality
7. `&`
8. `^`
9. `|`
10. `and`
11. `or`

Parentheses remain authoritative and generated C explicitly parenthesizes the parsed tree. Canonical logical spelling is `and` / `or`; `&&` and `||` receive a directed correction.

Integer literals are contextually typed by initializers, returns, binary operands, and unambiguous call parameters. Multiple admitted overloads remain ambiguous.
