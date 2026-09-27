# Structured inline assembly (R7l bounded form)

Use typed machine intrinsics when an operation has known semantics. Use
inline assembly only when the instruction sequence itself is the point.

The first form is architecture-qualified, explicitly unsafe, and local to a
statement. For example:

```concept
unsafe asm AMD64 {
    "bswap {value}";
    inout register value;
    clobber flags;
    memory none;
}
```

`in`, `out`, and `inout` have distinct read/write roles. The operand is a
typed `uint8/16/32/64`, `uint`, `int`, `byte`, `usize`, or `isize` scalar in this bounded form. `out`
does not read its prior value. An aggregate, const output, unknown clobber,
undeclared register, branch/call/return, missing memory declaration, or
wrong architecture is rejected. The only named clobbers admitted here are
`flags`, `r10`, and `r11`; direct `%r10*`/`%r11*` use requires that clobber.
Memory must be declared `none`, `read`, `write`, or
`readwrite`; the chosen effect is retained in MIR and planner evidence.

The one register operand limit is explicit. Multiple operands, address
operands, broad direct register naming, and AArch64 asm are deferred. This form
does not expose GCC or Clang constraint strings. A generated `.machine.S`
helper handles the instruction, while strict C11 calls that helper through a
typed pointer. The ordinary function-call boundary is conservatively ordered
around memory and MMIO operations; `memory readwrite` does not claim a CPU
fence. The source declaration is preserved in `concept-module.v1`.

`[[machine]]` remains compiler-owned intrinsic metadata, not the application
inline assembly form.
