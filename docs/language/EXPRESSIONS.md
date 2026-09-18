# Expressions

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
