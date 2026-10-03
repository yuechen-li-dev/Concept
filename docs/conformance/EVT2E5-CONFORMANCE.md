# EVT2e5 — Cathedral direct CALL encoding and native Win64 execution

Baseline: `5b2e2b3cc630f1bfc7ec9077837fdd1918503168`.
Compiler: `concept-evt1-stage0-go`.
CMIRAMD3 remains version 3, schema hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.

## Normal path and authority

`concept amd64` passes the checked artifact once to Concept `EmitModule`.
Concept decodes every function, lays out blocks, allocates registers, qualifies
the EVT2e4 frame/actions, emits bodies and resolves internal calls. The C11
bootstrap compiles and runs that library and renders its returned metadata.
Go implements no E8 encoding, displacement calculation, function layout,
symbol resolution or patch policy. Its changes are bootstrap orchestration,
diagnostic presentation and independent test input/oracles.

The single-function compatibility APIs still reject calls before writing bytes,
now with `AMD64_CALL_REQUIRES_MODULE_IMAGE`: a body alone cannot establish an
internal target layout. The underlying bridge enum tag remains unchanged.
Normal CLI selection uses module emission and succeeds for the qualified calls.
The CLI emits and displays bytes; the native test runner executes them.

The baseline encoder emitted one selected no-call function, with per-function
branch fixups. It had no internal symbol table or call patch facilities. EVT2e5
retains those branch semantics, uses one writer coordinate system for a module,
and adds transient Concept-owned native records without changing the wire.

## Symbols, layout and calls

Functions occupy contiguous bodies in checked declaration order, without
inter-function alignment padding. Symbols retain semantic declaration identity,
diagnostic name, ordinal, offset, size and expected call count. Identity is the
checked signature identity, not a display-name-only key or host pointer.
Lookup detects missing and duplicate identities; fixed storage holds at most
128 symbols and 512 call fixups. The caller supplies the byte buffer.

Every call emits `E8 00 00 00 00` plus a `Rel32Call` record containing source
ordinal/identity, target identity, opcode and patch offsets, displacement and
resolution state. Both forward and backward calls enter the same resolver
after all bodies have been laid out. Fixups occur in function/body instruction
order, strictly increasing by nonoverlapping five-byte call sites.

For opcode offset C and target offset T, **delta = T - (C + 5)**. The checker
proves the successor address does not wrap, then uses unsigned magnitudes to
admit exactly [-2147483648,2147483647]. Concept has uint64 but no signed int64
builtin; this avoids signed overflow while returning a proved signed int32.
Tests pin both endpoints, both one-past-range cases and successor overflow.
There is no truncating cast before range admission.

The resolver validates the complete layout, identities, expected call counts,
opcode, body containment, patch offset, four zero placeholder bytes and target
range before modifying bytes. It patches exactly four little-endian bytes and
marks each record resolved once. Resolving a finalized image is an error.
An independent final validator rechecks layout, identity/order/counts, E8,
resolution state, target displacement and all four encoded bytes. A missing
fixup cannot hide behind a reduced fixup count.

The minimal `Forward -> Zero` body starts with `48 83 EC 28`, then
`E8 0B 00 00 00` at offset 4: Zero starts at 20, so 20-(4+5)=11.
`Back -> Zero` has site 449, delta -434 and bytes `E8 4E FE FF FF`.
The shallow recursive Fact call at 1861 targets 1783: delta -83,
`E8 AD FF FF FF`. These are calls in the actual native image.

## Concrete action encoding

The shared encoder consumes qualified EVT2e4 actions in their prescribed order:
prologue allocation and saves; per-site spills, stack stores and register/temp
moves; call; RAX result capture and reloads; epilogue restores, release and RET
at every bound return. The existing action automata still admits the pre/post
call protocol; no DragonGod dependency or second runtime was introduced.

All frame memory addresses use post-prologue RSP. A dedicated RSP ModRM/SIB
writer supports 1/2/4/8-byte scalar accesses, 66 for words, REX.W for qwords,
extended registers and bare REX for the low-byte aliases that otherwise name
AH/CH/DH/BH. It selects disp8 or disp32. Register copies use the same alias
discipline. Incoming fifth through eighth scalar parameters load from
finalFrameBytes + entryOffset (40..64); outgoing stores begin at RSP+32.
Stack allocation/release selects imm8 or imm32 through the existing encoder.
Exact-byte checks also pin a 136-byte imm32 stack subtraction and a 16-bit
R10 store with disp32 at RSP+136, plus the BPL immediate's required bare REX.

Representative forms: `48 83 EC 28` / `48 83 C4 28 C3` allocate/release the
minimal call frame; `48 89 5C 24 xx` / `48 8B 5C 24 xx` save/restore RBX;
`44 89 54 24 xx` / `44 8B 54 24 xx` spill/reload R10d;
`4C 89 54 24 xx` stores a full canonical R10 temporary;
`44 89 54 24 20` stores an i32 fifth argument at RSP+32.
Here xx is the qualified concrete slot displacement, not an emitted placeholder.
Full module hex and fixup bytes are printed by the native test and compared.

Module allocation can use callee-saved registers for leaf callees with eight
incoming values. Its qualified frame saves/restores each used register. The
legacy five-register leaf allocator/encoder API remains available, and the
existing no-call frozen byte corpus is unchanged. Call-preservation spills
retain their logical widths; general register-exhaustion spilling is not added.

## Execution evidence

`TestEVT2e5NativeInternalCalls` compiles Concept backend and the independent
source C oracle with strict C11 `-pedantic-errors -O2`, in Normal and Verify.
Its source fixture is `internal/concept/testdata/evt2e5_calls.concept`:

| Case | Real execution |
| --- | --- |
| Forward / Back | Zero-argument forward/backward internal calls |
| One / Call4 / Call5 / Call8 | 1, 4, 5 and 8 meaningful scalar arguments |
| CallVoid / CallBool | Void return, false and true through RAX |
| CallNarrow8 / CallNarrow16 / CallWide | 8/16/64-bit values, including high bits |
| Keep8 / Keep16 / Keep64 / KeepBool | Narrow/wide/bool live-across spill/reload |
| Across / Pressure | Computed local across a call and multiple i32 spills |
| Chain -> Middle -> Id | Nested chain with 40/56/0-byte frames |
| Multi | Three calls sharing the maximum outgoing reservation |
| Repeat | Repeated source arguments |
| FullSaved | Eight meaningful arguments exercise leaf callee saves |
| Early | Both return paths restore frames |
| Fact | Bounded shallow recursion, not tail-call optimization |
| Cycle2 / Cycle3 | Controlled allocation with concrete two/three-way cycles |

All ordinary fixtures execute 100 times against the independently generated
C11 function results. Cycle probes use checked literal definitions and controlled
allocation assignments through the real Concept action encoder and resolver,
then execute the native result against a noncommutative Ordered C oracle.
No production allocation override hook is introduced.

An independent Win64 assembly wrapper seeds all eight nonvolatile GPRs,
records RSP and a stack canary, calls Pressure, FullSaved, Chain and both Early
paths, and checks all state afterward. A separate writable probe image replaces
only the Zero leaf body with a tiny entry-RSP observer before RX; Forward returns
8, proving callee entry RSP is 8 mod 16 through the actual emitted call site.
That instrumentation is an ABI probe, not source semantic parity evidence.

Each image is allocated RW, fully copied and patched before VirtualProtect RX
and FlushInstructionCache, invoked through a typed function pointer, then freed.
No call patches happen after RX and no persistent RWX mapping is used. Native
qualification is Windows AMD64; other hosts explicitly skip this execution test.

100 complete repeated emissions compare image layout/debug text and every byte.
Normal and Verify compare the full emitted hex, symbols and fixups. Invalid
probes cover duplicate/missing symbols, unresolved or missing records, bad E8,
duplicate sites, mismatched source identity, invalid symbol count, signed range
and short output buffers. Existing EVT2e3/4 tests retain independent move/frame
replay, stale FLAGS rejection and malformed ABI/frame admission.

## Boundaries

Qualified: module-local direct Win64 calls, zero through eight fixed-width
integer/bool arguments, scalar/void results, frame saves and call-preservation
spills/temps, stack arguments, nested and multiple calls, shallow recursion.
U64 preserves address-sized bits, but pointer/address-like **source calls remain
unqualified** because the checked CMIRAMD3 producer contract rejects them.
An abstract ABI Address plan is not end-to-end pointer-call support.

Still rejected/deferred: external symbol/helper linkage, float/double/XMM,
vectors, aggregates/sret, varargs, indirect calls, tail calls, object/COFF
relocations, SEH/unwind metadata and general allocation spilling. Unwind metadata
and guard-page stack probing are future production requirements. Native frames
of 4096 bytes or more stop at `AMD64_STACK_PROBE_REQUIRED`; planned metadata may
describe larger frames but that is not native qualification. Existing unsupported
arithmetic/memory forms still have explicit encoding stops.

Runtime encoding/preparation errors use typed Result. NativeSymbol and CallFixup
are immutable PlainData records; assertions consume Vocabulary's typed Verdict.
NoAllocation assertions admit EmitModule and ResolveNativeCalls. PlainData does
not confer wire or foreign ABI equivalence. Named errors distinguish invalid
input, capacity, duplicate/unresolved symbol, rel32 range, malformed patch,
unresolved fixup, unsupported encoding/memory operand, frame failure and probing.

## Measurements and regression qualification

The flagship image is **2368 bytes, 37 functions, 29 calls**. Forward is 20 bytes,
Pressure 149 bytes; the Chain/Middle/Id bodies total 220 bytes within this module.
NativeSymbol is 32 bytes, CallFixup 40 and NativeImage 24,592. The final focused
C11 probe measured emission at 4 ms Normal / 4 ms Verify and 1000 fixup passes
at 23 / 24 ms. The latter includes pending-image copying and placeholder reset,
resolution and final validation; it is not isolated arithmetic throughput.
100 native parity passes were below the host clock resolution (reported 0 ms).
Timings are informational, not performance acceptance gates.

Baseline all Go tests passed: internal/concept 470.130 s, CLI 21.374 s,
GPU-free Vulkan 0.336 s. Vet and focused race (16.220 s) passed. Standard 42,
DragonGod 23 and Golden 130 passed in Normal and Verify.
Final full Go passed: internal/concept 498.128 s, CLI 26.549 s, GPU-free Vulkan
0.455 s. Vet passed; focused race covering EVT2e3/4/5 passed (75.008 s).
Standard 42, DragonGod 23 and Golden 130 passed in both Normal and Verify.
The semantic corpus retains 436 valid, 357 static-invalid, 15 runtime-negative,
five compatibility and four expected-divergence specimens; its dedicated
manifest/determinism/runtime lane passed as well (2.300 s).

The full suite includes EVT2 native Add/Max and ModRM/SIB checks, EVT2x native
finite/dynamic-address/pushdown traces, 100-run existing native byte determinism,
the R9a2 frozen native byte oracle and unchanged bridge/schema goldens. Final
focused CALL/alias/imm32/disp32 execution and CLI emission passed. Touched Go
files are gofmt-clean; Concept backend/fixture format and lint pass with the
library roots configured; git diff --check passes. Authority ledger, migration
matrix, current design notes and separate EVT2e5 history entry are updated.
Frozen root and legacy Zig tests are skipped: no legacy Zig compiler or
build/test infrastructure changed.

EVT2e closes for the exact bounded subset above. No XMM milestone is begun.
