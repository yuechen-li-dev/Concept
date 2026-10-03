# EVT2c AMD64 MachineIR

LIR is target-independent low-level semantics. MachineIR is target-aware operational lowering. MachineIR knows AMD64 registers, flags, addressing, and ABI rules. MachineIR still does not know final instruction bytes or final general-purpose physical register allocation.

The pipeline is MIR -> SemanticFacts -> validated General Planner / LoweringPlan -> verified LIR -> `LowerLirToAmd64Machine` -> verified MachineIR. `concept machineir <file>` runs this path and prints deterministic inspection text. `concept lir` retains its separate target-independent output. Machine lowering may choose target mechanisms, but it may not erase retained safety behavior. A checked addition becomes `MOV`, `ADD`, and a flags-dependent overflow edge to `TRAP`; a retained index guard becomes `CMP` and an unsigned failure edge. Planner decision IDs and fact references remain on selected machine instructions. There is no C emission in the MachineIR command.

EVT2d adds an artifact-only consumer of this verified structure. `concept machineir-bin <file>` writes the versioned typed [`CMIRAMD3` bridge](EVT2-MACHINEIR-BRIDGE.md); the Concept-authored [AMD64 backend](EVT2-CONCEPT-AMD64-BACKEND.md) decodes it, allocates registers, finalizes the frame, and emits instruction bytes. `concept amd64 <file>` runs that consumer through the existing C11 bootstrap and displays the bytes. The inspection printer is not a backend input.

## Function, control flow, and operands

Each function carries its semantic identity, source span, `amd64-windows` target, `win64` ABI, ordered blocks, virtual registers, stack slots, frame metadata, incoming argument locations, facts, and Planner decisions. Original LIR block IDs stay intact. A retained arithmetic or bounds check adds a success continuation and an explicit trap block. Every machine block ends in `JMP`, `JCC`, `RET`, or `TRAP`; no semantic edge depends on text layout. Instructions retain their LIR block/instruction index and source span. Source coordinates are diagnostic provenance, not identity.

Operands are tagged as virtual register, physical register, immediate, stack slot, or memory. Memory has base, optional index, hardware scale (1, 2, 4, 8), displacement, width, and region. Indexed array addresses use `LEA`; typed `LOAD`/`STORE` retain the originating slot region. Nonhardware strides require later legalization; the current LIR scalar widths produce legal scales. Virtual registers use stable integer IDs, explicit 8/16/32/64-bit GPR widths, and a separate address role for 64-bit pointers. `MOV`, `ADD`, `SUB`, `IMUL`, `UMUL`, `CMP`, `TEST`, `SETCC`, `LEA`, `LOAD`, `STORE`, and the four terminators are semantic machine operations. `UMUL` is a legalization pseudo-op for unsigned multiplication; its CF overflow edge must survive later legalization. No byte forms are selected here.

Flags are explicit numbered dependencies. Arithmetic and `CMP`/`TEST` define flags; `JCC` and `SETCC` consume an exact dependency. Signed comparisons select L/LE/G/GE, unsigned comparisons B/BE/A/AE, and equality E/NE. Signed checked arithmetic tests OF; unsigned checked addition/subtraction/multiplication tests CF. A comparison used immediately by a branch remains in flags without materializing a bool. Other bool values use `SETCC` or `TEST` as needed. A bounds check compares the index against extent and branches on unsigned AE, rejecting negative signed indices as well as indices at or above the extent.

## Windows x64 ABI and frame

EVT2e3 derives outgoing scalar ABI plans separately in Concept `Win64ABI` from
the unchanged CMIRAMD3 call records. It does not add physical call fields to
MachineIR or change its inspection printer. Plans retain fixed argument and
result constraints, symbolic stack slots/home space, FLAGS/register clobbers,
parallel moves and live-across preservation requirements. Final call frames,
spills, saves/restores and CALL encoding remain deferred. The existing incoming
ABI descriptors below remain the Stage-0 compatibility seam; no new outgoing
algorithm is added there. See [planning conformance](../conformance/EVT2E3-CONFORMANCE.md).

Integer and pointer argument positions 0..3 use RCX, RDX, R8, R9; the fifth starts at entry RSP+40. Parameters are copied into virtual registers on entry. An aggregate larger than 8 bytes, including `int[4]`, is passed indirectly through a caller-owned temporary; its pointer becomes the array slot base. Integer return values move to the RAX family immediately before `RET`. The physical register model has canonical parent identities RAX through R15 with 8/16/32/64-bit aliases. A 32-bit write zeroes the upper half of its 64-bit parent; the encoder and allocator must preserve that rule. Caller-saved GPRs are RAX, RCX, RDX, R8-R11. Callee-saved GPRs are RBX, RBP, RSI, RDI, R12-R15; RSP is separately fixed stack state. These lists, the 32-byte caller shadow space, and 16-byte stack alignment at calls follow the [Microsoft x64 calling convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention) and [register conventions](https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions).

Scalar mutable locals are abstract stack slots with size, alignment, and source name. Final RSP offsets remain unresolved. `Frame.LocalSize` records packed local storage and `Frame.Alignment` is 16; `ShadowSpace` stays zero for these call-free functions. Frame lowering will reserve the 32-byte outgoing shadow area only when calls exist. Entry RSP is 8 modulo 16 after the return address push, so a future nonleaf prologue must adjust RSP to 0 modulo 16 before a call. No prologue, epilogue, save/restore, unwind metadata, call, or final frame offset is emitted in EVT2c. Simple leaf functions may need no frame; local slots will need one when encoded.

The current scalar path supports integer widths and bool. Narrow loads retain their width, with signed or zero extension deferred until a wider consumer actually requires it. Float/SIMD, payload enums, complex Drop paths, atomics, MMIO, general calls, SysV, optimizer passes, physical allocation, bytes, executable memory, COFF, and PE remain outside EVT2c. Unsupported source semantics fail in the upstream LIR stage; unsupported MachineIR types and array base forms fail explicitly.

## Verification

`VerifyMachineIR` checks function target and ABI, ordered blocks, valid operand tags and widths, virtual-register definitions and references, physical register aliases, stack slots, legal memory scales and address bases, Win64 parameter locations, frame requirements, opcode operand forms, unique flags definitions, same-block flags use after the latest producer, condition codes, branch targets, terminators, and RAX-family return setup. Machine opcode effect metadata records memory reads/writes, flags reads/writes, termination, and traps. A later CALL opcode can extend that table with caller-saved clobbers. The verifier runs before the printer returns output.
