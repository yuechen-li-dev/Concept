# R6g byte migration

Before R6g, unqualified `byte` meant the C11-backed eight-bit scalar used by
legacy fixed storage. R6g reserves `byte` as the Information unit inside a
numeric qualification while retaining that old unqualified spelling as a
compatibility alias.

Classify and migrate deliberately:

| Old intent | Canonical R6g spelling |
|---|---|
| eight-bit numeric value or raw cell | `uint8` |
| byte count, size, alignment, offset, capacity | `usize<byte>` or `isize<byte>` |
| bit count | `usize<bit>` |
| historical fixture whose intent is not yet classified | retain unqualified `byte` temporarily |

There is no contextual inference that turns an unqualified `byte` value into a
quantity. This prevents legacy storage cells from silently acquiring extent
semantics. New APIs must not add more scalar-`byte` surface.
