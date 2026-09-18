# EVT1 integer semantics direction

R7d3 makes integer meaning independent of the Go host and the C implementation.

- An integer literal retains its exact `uint64` magnitude, source spelling, and a separate negative flag until an expected semantic type resolves it.
- `int` is exactly signed 32-bit and lowers to `int32_t`.
- Fixed-width unsigned arithmetic is modulo 2^N.
- Signed `int` addition, subtraction, and multiplication are checked. Overflow terminates through the Concept panic path; it is never emitted as C signed-overflow UB.
- Signed division checks zero and `INT32_MIN / -1`. Unsigned division checks zero.
- Constant invalid divisions and shifts are rejected before C emission. Dynamic shift counts outside `[0, width)` terminate through the Concept panic path.
- Signed right shift is arithmetic floor division by a power of two. Signed left shift operates on the 32-bit representation without relying on a C signed shift.

The bounded compile-time evaluator may decline a value its internal bounded carrier cannot represent. It does not change or clamp the source literal.

Binary literals and digit separators remain deferred.
