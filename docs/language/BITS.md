# Scalar-backed register bits (R7k)

A bits type is an ordinary value with an explicit scalar representation and
explicit deterministic bit ranges. It is not a C bitfield.

```concept
bits UartLineStatus : uint8
{
    dataReady: 0;
    mode: 1..3;
    transmitterEmpty: 5;
}
```

The range `1..3` includes bits 1 and 3. Positions count from the least
significant bit. The representation is exactly one `uint8`, `uint16`,
`uint32`, or `uint64` scalar stored as the `raw` member of an ordinary record
value. `SizeOf` and `AlignOf` match that scalar; no `repr(C)` claim is implied.
Duplicate names, reversed/out-of-width ranges, and overlaps reject at parse
time. Alias fields and signed fields are not available.

`UartLineStatus{raw}` creates a value. A one-bit field read has type `bool`;
a range field read has the underlying unsigned scalar type. Field reads only
inspect the value. `status with { mode = 3; }` creates a fresh value, replaces
only that field's bits, and preserves all others, including reserved bits.
Constant values that exceed a field reject. Dynamic values are checked before
insertion and trap if they do not fit. Bits equality compares raw scalar values.

Writing a register requires an explicit `MmioStore` of the scalar `raw` value.
Reading a register requires an explicit `MmioLoad` followed by bits value
construction. No proxy register or implicit hardware transaction exists.
