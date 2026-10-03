# EVT2d Concept AMD64 backend

> The native AMD64 backend is authored in Concept and currently bootstrapped through Concept's existing C11 backend.

> The C backend is used to build the backend that will eventually replace it for native AMD64 emission.

> The AMD64 encoder is not an external assembler wrapper. It emits instruction bytes directly from verified, allocated MachineIR.

> Register allocation is deterministic and authored in Concept. The initial strategy is intentionally simple and correctness-first.

## Staging and command

Go Stage-0 owns parsing, semantic analysis, MIR, Planner, LIR, and verified AMD64 Windows MachineIR. `EncodeMachineBridge` transports that structure as `CMIRAMD3`. `libraries/Standard/Backend/AMD64.concept` owns bridge decoding, liveness, register assignment, frame finalization, and byte emission. The existing C11 backend compiles that Concept module; the `concept amd64 <file>` driver uses a temporary C11 executable to run it and prints raw-code hex by function. The temporary executable is a bootstrap host, not an emitted target object or application. `concept machineir-bin <file>` writes the exact bridge for independent backend use. Later phases may move upstream lowering stages into Concept; EVT2d does not do that.

The native test harness allocates writable memory, copies the bytes, changes protection to executable/readable, flushes the Windows instruction cache, invokes a typed Win64 function pointer, and releases the mapping. It isolates the UD2 failure path structurally rather than intentionally crashing the Go test process. The generated target code is never permanently RWX.

## Allocation and frame

The Concept reader uses typed `Register`, `OperandKind`, `Opcode`, and `Condition` enums. It maintains fixed-capacity storage: eight arguments, 128 virtual registers, 64 slots, 128 blocks, and 512 instructions. The current liveness pass is bounded to 16 blocks and 32 virtual registers because Concept's fixed-array cell limit is 512. Larger inputs return `CapacityExceeded`. It builds block use/def sets, iterates live-in/live-out to a fixed point across CFG backedges, then extends deterministic instruction-order intervals at block boundaries. Linear scan assigns the first free register in the R10, R11, RAX, R8, R9 pool. A register may be reused when its final read and the next definition share an instruction, since MOV/LEA/LOAD read before writing. Incoming ABI register copies occur first; R8/R9 are not selected until the corresponding incoming values have been copied. RSP is fixed, RBP is reserved, and no callee-saved register is allocated. A sixth simultaneous value returns `RegisterExhausted`; spill slots and reload/store code are deferred rather than silently miscompiled.

`FinalizeFrame` packs local slots by their declared size and alignment and checks the layout against MachineIR frame metadata. Leaf functions with no slots have no prologue. A leaf with locals subtracts the smallest stack size that covers the locals and makes RSP 16-byte aligned; every return restores it. The MVP emits no calls and reserves no outgoing 32-byte shadow area. Incoming aggregate arguments are Win64 indirect pointers. No general call or unwind metadata is emitted.

## Encoding

The bounded Concept `ByteWriter` writes bytes and little-endian `i32` values, then patches `rel32` branches after deterministic block layout. Errors return `Result`; callers discard the buffer when encoding fails. The encoder handles MOV, LOAD, STORE, LEA, ADD, SUB, IMUL, CMP, TEST, SETCC, JMP, Jcc, RET, stack adjustment, and UD2. It emits only the forms used by the qualified integer fixture; unsupported widths, operands, pseudo-operations (including UMUL), calls, and spill needs return explicit errors. Signed and unsigned condition codes have distinct encodings. Checked overflow uses JO to a UD2 block. Bounds checks use unsigned JAE, which also rejects negative signed indexes after a 32-bit compare.

Register encoding has one typed mapping. REX selection accounts for operand width and extended ModRM/SIB register fields. Memory forms use base plus optional index times 1, 2, 4, or 8 plus displacement. RSP/R12 force SIB; RBP/R13 with zero displacement force a zero `disp8`; larger displacement uses `disp32`. The direct C11-hosted encoder test pins simple MOV/ADD/CMP bytes and an addressing matrix for low, extended, RSP/R12, RBP/R13, indexed, all scales, and `disp32` cases. Add and Max end-to-end tests pin their whole-function bytes.

`BackendFacts` proves `NoAllocation` for `WriteByte`, `AllocateRegisters`, `EncodeFunction`, and `EmitFunction` against Concept's semantic checker. The entire path uses caller-provided output storage and fixed arrays. Every recoverable backend failure is a `Result` and propagated or returned; a failed encode does not publish a byte count.

## Current boundary

The seven EVT2 fixtures emit native bytes: Add, Max, Sum4, CheckedIndex, StoreIndex, Choose, and Early. Representative native calls are compared with the C backend, including a loop, stack locals, indexed reads, indexed writes, and multiple return paths. Source semantics still come from Stage-0 MachineIR. Float/SIMD, general calls, object files, spills, nonleaf unwind metadata, and a production native failure runtime remain future work. A UD2 encodes both overflow and bounds failure in this bootstrap slice.
