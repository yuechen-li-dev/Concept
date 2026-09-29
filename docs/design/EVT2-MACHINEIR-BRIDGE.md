# EVT2d MachineIR bridge

The bridge is a typed, deterministic binary artifact between verified Go Stage-0 MachineIR and the Concept AMD64 backend. `concept machineir-bin <file> > out.cmir` writes it. The backend never parses `concept machineir` inspection text and does not read Concept source. The schema identity is the eight ASCII bytes `CMIRAMD1`; any other identity is `BridgeVersion` in Concept and `MIR_BRIDGE_SCHEMA_MISMATCH` in Go. A layout change requires a new identity.

All integers are little endian. Counts and byte lengths are `u32`; other numeric fields and enum tags are `i32`. Strings are UTF-8 byte lengths followed by bytes, without a terminator. Booleans are `i32` zero or one. The Go encoder first runs `VerifyMachineIR`; the Go decoder rejects unknown tags, truncation, trailing bytes, and invalid MachineIR. The Concept reader checks bounds and tags, keeps identity/provenance string slices as byte offsets, and builds typed enum and operand values. No Go struct memory layout or `gob` representation crosses the boundary.

The document order is:

1. Magic, function count.
2. Per function: identity, name, target, ABI, result type, source line/column, fact strings, decision strings, frame local size/alignment/shadow space/has calls.
3. Arguments: count, then type string and index/width/indirect/physical register/stack offset/on stack/virtual register.
4. Virtual registers: count, then ID/width/address role.
5. Slots: count, then ID/size/alignment/incoming indirect/base virtual register/source name.
6. Blocks: count, then ID/LIR block ID/instruction count.
7. Instructions: opcode tag, destination operand, source count and operands, width/flags definition/flags use/LIR block/LIR instruction, condition tag, source line/column, Planner decision string, fact strings.
8. Terminator: opcode tag, true/false block IDs, flags use, condition tag, source line/column.

Each operand is seven `i32` fields (`ID`, `width`, `base`, `base slot`, `index`, `scale`, `displacement`), then a kind tag, literal string, region string, and signed boolean. This retains physical constraints, memory geometry, ABI locations, branch targets, flags, stack metadata, and useful provenance. Irrelevant Go implementation pointers and printer layout are absent.

| Tag | Operand kind | Opcode | Condition |
| --- | --- | --- | --- |
| 0 | None | None | None |
| 1 | Virtual | MOV | E |
| 2 | Physical | LOAD | NE |
| 3 | Immediate | STORE | L |
| 4 | Slot | LEA | LE |
| 5 | Memory | ADD | G |
| 6 | Block | SUB | GE |
| 7 |  | IMUL | B |
| 8 |  | UMUL | BE |
| 9 |  | CMP | A |
| 10 |  | TEST | AE |
| 11 |  | SETCC | O |
| 12 |  | TRAP | C |
| 13 |  | JMP |  |
| 14 |  | JCC |  |
| 15 |  | RET |  |

Physical register tags are the existing MachineIR order: RAX, RBX, RCX, RDX, RSI, RDI, RBP, RSP, R8 through R15. The Concept reader converts wire integers to enums with literal `match` arms and an explicit error arm. This makes the schema tags the only numeric boundary; backend logic uses typed operations.

The Go round-trip test runs the full seven-function MachineIR through this format and re-encodes byte-identically, including fields not used by the current encoder. It also checks the same input 100 times, plus wrong-version, truncation, and trailing-byte rejection. The Concept reader consumes the complete artifact before selecting a function and rejects trailing bytes. Its native harness checks wrong-version, trailing-byte, and short output-buffer errors before encoding a valid artifact.
