# EVT2d conformance

The compiler ID is `concept-evt1-stage0-go`. The native backend source is [`Standard.Backend.AMD64`](../../libraries/Standard/Backend/AMD64.concept), compiled through the existing C11 backend. Stage-0 Go retains parsing through verified MachineIR and writes the [`CMIRAMD1` bridge](../design/EVT2-MACHINEIR-BRIDGE.md). No Go allocator or encoder, external assembler, object file, or JIT was added.

## Qualified fixture path

The tests compile the Concept backend to strict C11, transport the real seven-function EVT2 source fixture, emit bytes with the Concept backend, and call those bytes through Windows W^X executable memory. The existing C backend compiles the same source as the differential oracle. Each encoded function is regenerated 100 times in the native harness and compared byte-for-byte; representative native calls run 100 times each and are compared with the C result.

| Function | Native evidence | Safety/ABI evidence |
| --- | --- | --- |
| Add | `(2,3)=5`, `(-7,5)=-2`, `(1000000,-100)=999900` | Win64 integer arguments and return, JO to UD2 |
| Max | `(2,3)=3`, `(7,-2)=7`, `(-5,-1)=-1` | signed JG, both return paths |
| Sum4 | `{1,2,3,4}=10`, `{-1,5,3,4}=11` | indirect aggregate pointer, local frame, loop backedge, checked accumulation, indexed load |
| CheckedIndex | element 0 is 10, element 3 is 40 | unsigned JAE bounds failure, indexed LEA/LOAD |
| StoreIndex | writes/returns 77 at index 2 and −9 at index 0 | indexed STORE, resulting memory inspected |
| Choose | true chooses −8, false chooses 7 | 8-bit bool copy and TEST, two return paths |
| Early | `−2→−2`, `0→0`, `5→4` | local slot, frame restoration on both returns, checked subtraction |

Exact Add bytes: `4189ca4189d34589d24501da0f8005000000e9020000000f0b4489d0c3`.

Exact Max bytes: `4189ca4189d34539da0f8f05000000e9040000004489d0c3e9000000004489d8c3`.

The direct Concept encoder matrix pins MOV/ADD/CMP register forms and memory forms for RAX, R8, R12 SIB, R13 zero displacement, R12+R9*4+8, scales 1/2/4/8, and `disp32`. The native fixture bytes include direct `rel32` fixups, signed and unsigned Jcc, REX, ModRM, SIB, `UD2`, and prologue/epilogue instructions. Add and bounds fixtures structurally assert their failure branches and UD2 targets. Trap execution is not performed in the main Go test process.

The bridge test verifies complete seven-function Go MachineIR round trip, 100-run bridge byte identity, unknown schema rejection, truncation, and trailing-byte rejection. The C11-hosted Concept backend test verifies version, trailing-byte, and output-capacity errors; its direct allocator test proves register exhaustion, while each native fixture checks 100-run assignment and frame identity. `concept check`, `concept lint`, and `concept format --check` qualify the backend source. `concept explain` proves `NoAllocation(EmitFunction)` and shows the fixed-storage allocator and encoder calls below it.

## Deliberate boundaries

The Concept liveness workspace supports at most 16 blocks and 32 virtual registers; the artifact reader permits larger structures but allocation returns `CapacityExceeded`. The five caller-saved register pool has no spills and returns `RegisterExhausted` when necessary. Calls, outgoing shadow space, saved callee registers, 64-bit immediates, unsigned multiplication pseudo-op legalization, floating point/SIMD, object files, and unwind metadata are outside the qualified native subset. The encoder reports unsupported forms through `Result` rather than publishing a byte count.
