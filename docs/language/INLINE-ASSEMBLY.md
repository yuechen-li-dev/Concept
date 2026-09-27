# Structured inline assembly

If Concept knows what the machine operation means, use a typed intrinsic.
Inline assembly exists only for operations or sequences the language does not
otherwise model.

The bounded AMD64 form is an unsafe, local statement:

```concept
unsafe asm AMD64 {
    "add {left}, {right}";
    in register left;
    inout register right;
    clobber flags;
    memory none;
}
```

Operands are ordered and typed. `in` is read, `out` is written without reading
its old value, and `inout` is read and written. A register-class operand may be
`uint8/16/32/64`, `byte`, `uint`, `int`, `usize`, or `isize`. Aggregates and
const outputs are rejected. Each backend-selected register operand has a named
template placeholder. Multiple occurrences of that placeholder are allowed.

An instruction with implicit architectural registers can name `eax`, `ebx`,
`ecx`, or `edx` instead of `register`. Fixed registers currently require a
32-bit scalar. At most one input and one output may share a fixed register;
two inputs, two outputs, an `inout` overlap, a duplicate operand name, or an
operand/clobber collision is rejected. Fixed operands need no placeholder
when the instruction names them implicitly. The backend selects among
`r8`–`r11` for ordinary operands, excluding declared clobbers.

`clobber flags` and bounded named register clobbers are explicit. `memory`
must be one of `none`, `read`, `write`, or `readwrite`. Omitting a memory
declaration is an error. The compiler preserves the declaration in MIR and
planner evidence. A memory effect orders compiler operations; it is not a CPU
fence. There is no asm goto, call/return escape, naked function, or general
assembler facility.

The backend passes a pointer to an ordered array of 64-bit slots to a
deterministic `.machine.S` helper. Each slot stores one scalar operand; output
slots are initialized to zero on the C side. The helper saves its frame
register and ABI-preserved registers it uses, loads inputs, executes the one
instruction, and writes outputs back. Generated C is strict C11. Concept
source and MIR contain no GCC constraint strings.

The asm declaration is preserved in `concept-module.v1`; artifact-only
consumers regenerate the same helper. An AMD64 asm statement on an AArch64
target is rejected. The current source syntax does not support AArch64 asm;
use typed AArch64 operations where available.
