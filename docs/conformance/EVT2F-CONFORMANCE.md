# EVT2f — scalar floating input progression

**MEANINGFUL PROGRESSION. EVT2f native XMM lowering is not qualified.**
The real `concept lir` path now accepts the scalar fixture. Its former failure
was `Arithmetic32: EVT2_UNSUPPORTED_TYPE float`. Both `concept machineir` and
`concept amd64` now reach the explicit next boundary:

```text
MIR_FLOAT_REQUIRES_CMIRAMD4 Arithmetic32: float32 requires explicit scalar and XMM register classes
```

## Baseline and schema decision

Baseline: `df010b72a841fbcc29bef7f7a2461fc10b793d0a`, clean `main`, containing
requested commit `aaa4cd57cceed05737f1998188f5b57e84c2be6a`.
Compiler: `concept-evt1-stage0-go`.
CMIRAMD3 remains v3, schema hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.
Generated Go transport, Concept decoder and schema declarations have **zero
delta**. There is no CMIRAMD4 artifact or hash yet.

CMIRAMD3 cannot honestly carry this new native subset: `WireMachineVReg` has
only ID, width and address; `Register` names GPRs only; `CallValueType` has
integer/bool/void cases only; there is no floating constant or operation
vocabulary. Four/eight-byte widths already mean integer/GPR data and cannot
silently acquire IEEE/XMM semantics. CMIRAMD4 is required before floating
MachineIR production. It must come from the existing sole checked Concept
schema and generated Go/Concept codecs, with stale-version rejection.

The baseline physical set is RAX..R15. The conservative no-call allocation
pool is R10/R11/RAX/R8/R9; the module call path can also use callee-saved GPRs.
Existing frame realization integrates locals, outgoing home/stack arguments,
callee saves, call-preservation spills and cycle temporaries. It does not yet
spill general register pressure. Incoming Win64 descriptors use positional
RCX/RDX/R8/R9, fifth and later parameters start at entry RSP+40, and integer
results use RAX. The existing `evt2e5_calls.concept` native fixture qualifies
direct integer/bool internal calls; EVT2x retains its native automata fixtures.

## Qualified checked input

Canonical `float`/`float32` become LIR `float32`; `double`/`float64` become
`float64`, using existing semantic `FloatRepresentation` evidence rather than
quantity parameter spelling. Binary16 and other representations remain
explicitly unsupported by LIR. The semantic layer still checks quantities
before representation projection. Same-unit `float<Pa>` and `double<m>`
arithmetic is covered. Same-representation `interpret` produces no runtime
instruction. Dimensionless quantity operations needing a nonidentity scale
stop at `EVT2_UNSUPPORTED_QUANTITY_SCALE`; they cannot silently lose scaling.
Explicit numeric casts remain unsupported in this LIR subset.

Floating constants store fixed-width lowercase hexadecimal IEEE bits:
`0x3dcccccd` for checked binary32 0.1 and `0x3ff0000000000001` for checked
binary64 1.0000000000000002. Projection starts from the existing checked AST
value, rounds once to binary32 where applicable, then records bits without a
decimal printer/reparse. This does not change the frontend's existing
decimal-to-binary64 parsing policy or claim a new arbitrary-decimal rounding
algorithm. Unary source `-0.0` is explicit `fneg` over positive-zero bits.
The constant contract admits both zero signs, infinities and NaN payloads
without canonicalization. Source NaN/infinity literal syntax is not added;
special bit patterns are covered by constructed checked LIR inputs and runtime
C oracle arguments.

Target-independent `fadd/fsub/fmul/fdiv/fneg` retain the operand width and
strict scalar semantics. They do not select SSE instructions, allocate XMMs,
or introduce fast-math/reassociation/contraction. Six ordinary comparison
operations accept same-width floating operands and return the existing bool
class. Their eventual Cathedral implementation must explicitly handle
unordered results; LIR acceptance is not native comparison qualification.

Scalar locals, indexed loads/stores and 4/8-byte array strides are verified.
Direct checked call arguments/results include both representations. The fixture
pins `[i32 float64 i32 float32]` and eight mixed argument types, plus a local
used across a call. These are typed input records, not physical placements or
live-across native preservation claims.

MachineIR admission scans the complete module before incoming ABI projection,
including return, parameter, slot and instruction types. Floating values cannot
reach the old GPR selection through a bool-returning comparison or unused local.
The existing scalar-width helper also refuses floating GPR argument/return
admission. Invalid widths, decimal/uppercase floating constants, integer
arithmetic on floating values, mixed-width operands and invalid negation arity
are rejected by the LIR verifier.

## Evidence and limits

`language/evt2f/valid/scalars.concept` has 32 functions. The focused tests pin
all four operations at both widths, six comparisons at both widths, negation,
constants, slots, arrays, typed mixed calls, quantities and zero-op interpretation.
100 fresh generations compare complete LIR text, including constant bits,
types, source metadata, calls and planner decisions.

The independent source oracle compiles generated C with strict C11
`-pedantic-errors -O2` in Normal and Verify, using exact bit checks for selected
arithmetic, binary64 precision and signed zero. Seven values at each width
(negative, positive zero, negative zero, positive, positive infinity, negative
infinity and payload-bearing NaN) exercise all six comparisons over 49 pairs:
588 comparison checks per mode. Separate assertions require NaN equality false,
inequality true and all ordered comparisons false. Typed source calls and
quantity/interpretation functions also execute.

**That execution is the Stage-0 C source oracle, not Cathedral floating native
execution.** No SSE bytes, XMM allocation, floating ABI probe, floating spills,
call-clobber preservation, XMM cycles, nonvolatile XMM saves or floating module
image exist yet. No floating native byte-determinism claim is made. Planning,
allocation, encoding time, native code size, image size and execution throughput
for XMM are unmeasured. The focused input/C oracle tests took 1.240 s locally;
that is a test wall time, not backend performance.

## Ownership and remaining boundary

Go changes are classified as follows:

| Files | Classification |
| --- | --- |
| `lir.go`, `lir_lower.go` | Checked semantic production and verification |
| `machineir.go`, `machineir_lower.go` | Compatibility seam: reject floating input before integer/GPR projection |
| `lir_float_test.go`, `lir_test.go`, `machineir_test.go` | Test orchestration and independent C source oracle |

No production XMM classification/allocation, floating ABI placement, SSE
instruction selection or encoding was added in Go. All remain Cathedral's
required implementation responsibilities. Authority ledger and migration matrix
record the input progress separately from unqualified native ownership.

Next: version the scalar/register/constant/call wire contract as CMIRAMD4 in
Concept, generate transport and decoder, then implement and independently
qualify Cathedral XMM allocation, Win64 positional mixed ABI, SSE encoding,
one integrated frame/spill model and real native/C11 parity. A proposed
conservative allocation policy is XMM0..XMM5 with live-across spills and
XMM6..XMM15 reserved; this is **not an implemented or qualified policy**.
SIMD, aggregate ABI, varargs, object output and AArch64 remain outside EVT2f.

## Validation

Baseline `go test ./... -count=1` passed (compiler 344.462 s, CLI 23.444 s,
GPU-free Vulkan 0.339 s). Baseline vet and focused LIR/call-contract race passed
(race 3.621 s). Standard 42, DragonGod 23 and Golden 130 passed in each of Normal
and Verify. The full suite includes semantic corpus, EVT2, EVT2x, EVT2e native,
frozen integer byte checks and representative 100-run determinism gates.
Optimized routine reuse policy was used; 100 cold-build/probe stress was not
requested. Frozen root and legacy Zig lanes are skipped because their compiler
and build/test infrastructure are unchanged.

Final regression results are recorded in the EVT2f convergence ledger. Concept
fixture format/lint passes; no Cathedral source was authored, so new backend
R9d match-policy checks are not applicable. The existing policy lanes remain
part of the full Go regression suite.

The unchanged EVT1 semantic manifest contains 440 valid, 357 static-invalid,
15 runtime-negative, five compatibility and four expected-divergence specimens.
The new EVT2f fixture is an additional input specimen, outside that manifest;
its three unsupported-source probes and 14 malformed-LIR mutations are covered
by the dedicated Go tests.

## EVT2f2 — CMIRAMD4 transport qualification

**SUCCESS for EVT2f2. Native floating execution remains unqualified.** The sections
above retain EVT2f1 history. Baseline is
`9b1b9ea25d5f017b9146db676d02f914d348601f`, compiler
`concept-evt1-stage0-go`, branch `codex/evt2f2-cmiramd4`. The former
`MIR_FLOAT_REQUIRES_CMIRAMD4` stop is removed for the qualified scalar fixture.
Both `concept lir` and `concept machineir` succeed; `concept amd64` validates the
entire artifact in Concept and stops at `AMD64_UNSUPPORTED_XMM_ALLOCATION`.

### Schema and compatibility

CMIRAMD3 remains version 3 and hash
`3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312`.
Its schema, generated codecs/metadata and frozen integer/native fixtures have no
diff. Its virtual record is ID/Width/Address; Register contains GPR identities
only; CallValueType contains void/bool and signed/unsigned integer widths. Its
selected operations have no exact float constant or floating semantic vocabulary.
Widths four/eight alone cannot distinguish scalar class.

The checked Concept `BridgeSchema4.concept` introduces CMIRAMD4 version 4/hash
`aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c`.
It is an explicitly separate virtual variant. Integer-only modules retain the
v3 path, including its existing incoming physical descriptors. Any supported
float-bearing module selects v4 for **all** functions before GPR/physical
projection. V4 carries no physical register names or argument/return locations.
Floating automata/frame forms are outside this contract and reject at
`MIR_CMIRAMD4_UNSUPPORTED_MACHINE_FRAME`; existing integer automata remain v3.

| Schema inventory | Frozen v3 | Additional v4 vocabulary |
| --- | --- | --- |
| Reflected records/enums | 21 | 17: 13 records, four enums |
| Reflected fields | 81 | 62 additional fields |
| Enum cases | 65 | 50 additional cases |
| `concept generated` codec declarations | 41 | 73 including inherited v3; delta +32 |

Generated Go structures, tags, read/write methods and generated Concept decoder
derive from the checked schema through `cmd/machinebridgegen`. Imported v3 byte,
text, sequence and header primitives are reused. The 44-byte header retains
magic/version/SHA256 geometry; strict v4 readers reject CMIRAMD1/2/3, wrong v4
number/hash, truncation and trailing bytes. The public Go dispatcher retains an
explicit frozen-v3 compatibility reader; it never treats v3 as v4 or silently
falls back. No hand-maintained float codec was introduced.

### Scalar and call model

RegisterClass is GPR/XMM. ScalarRepresentation is void/bool, i8/u8/i16/u16/i32/u32/
i64/u64, float32/float64 and address. A virtual value carries ID, class, exact
representation, checked width and address flag. Float32 requires XMM/4; float64
requires XMM/8; bool requires GPR/1. Addresses for qualified array access remain
GPR/8. Width is redundant checked geometry, never semantic authority.

Slots retain representation, byte size, alignment, extent and incoming array
value identity. Float arrays have strides four/eight; indexed address and load/
store retain element representation and bounds guards. Quantity-backed pressure
and distance project to float32/float64; same-representation interpret emits no
runtime instruction. Nonidentity scale conversion, numeric casts, binary16 and
other unsupported numeric formats keep their existing named LIR boundaries.

The 32 opcodes retain typed integer/storage/index operations and introduce
fconst, fadd/fsub/fmul/fdiv/fneg and feq/fne/flt/fle/fgt/fge. Float operations require
equal exact operand/result representation; comparisons produce GPR bool. Fconst
uses exactly `0x` plus eight/sixteen lowercase hex digits. There is no decimal
reparse or NaN canonicalization. Fneg retains semantic sign behavior including
signed zero. Later native lowering must preserve Concept semantics: NaN equality
false, inequality true and ordered comparisons false. This milestone transports
those semantic operations and makes no floating instruction/execution claim.

Calls retain callee identity, win64 convention, ordered argument value/type,
result representation, virtual result and source/LIR provenance. The mixed four
argument `[i32,float64,i32,float32]` and eight argument
`[float64,i32,float32,i32,float64,float32,i32,float64]` signatures roundtrip exactly.
Float/double and void results are distinct; void calls have no result value.
Live-across-call local identity/type survives without qualifying preservation.
Function/instruction proof IDs remain opaque provenance; transport does not
create semantic authority from them.

### Admission and evidence

The Go checker reuses existing checked LIR CFG/type verification after checking
v4 class/representation/provenance. Concept BridgeValidate4 streams bounded wire
views and validates the complete module, including unselected functions, exact
types, definitions, calls, slots and bounds guards before output. Runtime byte
failure uses Result; PlainData and NoAllocation assertions consume semantic
Verdict evidence for the virtual record, instruction/call views and bounded
validation/roundtrip. Limits are 128 functions/values/parameters/blocks, 64 slots,
512 instructions per function; generated decoding also caps sequence lengths at
1,048,576 and admits no silent truncation or heap fallback.

`TestEVT2f2MachineIRTransport`, `SchemaAndExactRecord`, `DigestSensitivity` and
`ConceptArtifactOnlyRoundTrip` establish:

- 100 independent checked schema generations and fixture projections with exact
  schema/hash/codec/text/artifact/reencoded bytes/digests agreement.
- A pinned 20-byte float32/XMM virtual record:
  `03000000010000000a0000000400000000000000` (transport bytes, not SSE).
- Exact float32 0.1 and float64 1.0000000000000002 bits; both widths' signed zero,
  signed infinities and payload-bearing NaN survive source-free Concept
  reserialization. Special constants are also operands of transported float
  comparisons; void-call transport is included.
- 52 malformed artifacts rejected independently by Go and compiled Concept in
  Normal/Verify: class/representation/width, arithmetic/compare/negation,
  constants, slots/stride/guard, call type/order/result/identity/convention,
  provenance, undefined values, duplicate identities, invalid tags and stale/
  malformed envelopes. Artifact roundtrip errors leave caller output untouched.
- Valid v4 native emission returns the precise XMM-allocation stop; malformed
  v4 admission returns invalid-input. Output and entire native image remain
  unchanged on these paths. No image is published.
- Exact float32 bits, float64 bits, opcode, representation and mixed signature/
  order mutations each change artifact digest. No physical XMM allocation appears
  in the printer or transport.

### Informational measurements

The 32-function scalar artifact is 19,396 bytes. The representative v3 integer
call module is 14,585 bytes (difference 4,811; different programs, not a schema
overhead benchmark). One focused run measured 1,000 checked encodes in 137.062 ms
and 1,000 checked decodes in 130.391 ms; timings include verification. Compiled
Concept performed 100 validated source-free roundtrips in approximately 49–51 ms
in Normal/Verify. These are informational wall/CPU measurements, not thresholds.

Frozen generated Go/Concept sizes are 44,205/8,944 bytes (1,392/163 lines); the
additional v4 files are 40,252/8,094 bytes (1,266/137 lines). Schema metadata adds
15,982 bytes/796 lines beside unchanged v3 18,860 bytes/992 lines. Declaration
counts come from the real `concept generated BridgeCodec[4].concept` commands
with checked semantic module artifacts, not source-text function counting.

### Ownership, validation and next step

| Go changes | Classification |
| --- | --- |
| `machineir_scalar.go`, `machineir_lower.go`, model/printer | Checked MachineIR production; virtual compatibility seam |
| `machineir_scalar_verify.go`, `machineir_verify.go` | Producer verifier using existing checked LIR rules |
| `machineir_bridge_generate.go`, generated v4 codec | Checked reflection materialization and generated bootstrap transport |
| `machineir_bridge_artifact.go` | Explicit version compatibility dispatcher |
| `cmd/machinebridgegen` | Generator orchestration |
| Scalar/LIR/MachineIR and CLI tests | Transport/native-boundary oracles and regression orchestration |

Concept owns the v4 contract and normal artifact admission. Stage-0 continues as
the durable seed; no physical floating backend algorithm was added to Go.
Authority ledger/migration matrix record XMM allocation, positional mixed Win64
ABI/XMM clobbers, floating spills/frame integration, SSE2 encoding and native float
execution as pending Cathedral responsibilities. EVT2f3 begins with the first
five planning/storage responsibilities; it was not started here. No constant
pool, physical XMM enum, floating frame geometry or SSE bytes were introduced.

Baseline and final validation details are in the convergence ledger. Standard
42, DragonGod 23 and Golden 130 passed each in Normal/Verify. The unchanged
semantic manifest remains 440 valid / 357 static-invalid / 15 runtime-negative /
five compatibility / four expected-divergence; the existing 32-function fixture
is outside it. New authored schema/validator format/check and Normal/Verify R9d
lint under the real match-preference manifest pass without findings. Categorical
schema/semantic cases use exhaustive match; sequential admission checks remain
ordinary conditionals. Generated codecs remain regeneration-exact. Frozen Zig
lanes are skipped; optimized shared-build reuse and lightweight 100-run gates
are retained, without opt-in cold-build stress.


## EVT2f3 — XMM allocation, mixed Win64 ABI and floating frames

**SUCCESS for the bounded allocation/ABI/frame slice. Native floating execution
remains unqualified.** The earlier sections record historical EVT2f1/EVT2f2
boundaries; the current real `concept amd64` path reaches
`AMD64_UNSUPPORTED_SSE_ENCODING` after Concept qualifies all scalar functions.
It publishes no native image and does not encode SSE operations.

### Derived physical authority

`Win64ABI.concept` owns physical `XMMRegister` identities XMM0 through XMM15,
argument table XMM0..XMM3, volatile allocation/clobber table XMM0..XMM5, and the
reserved XMM6..XMM15 table. These are derived state, outside both wire contracts.
`AMD64.concept` extends the existing allocator entry point and interval/liveness
analysis with a class-aware bounded scalar allocation branch. Floating values
use XMM; integers, bools and addresses use GPR. FLAGS remains separate.

Allocation visits instruction positions and virtual identities in stable order,
chooses the first free register in the owning class, and deterministically spills
the newly encountered XMM value when six volatile candidates are occupied. It
does not evict another value. GPR pools retain their existing five no-call or
thirteen call-bearing candidates; XMM pressure does not consume GPR capacity.
Pressure-spilled values stay in typed stack storage. Live-across resident volatile
values receive per-call preservation, using the existing CFG liveness predicate.
Call arguments dying at the call and the newly defined result are excluded.
XMM6..XMM15 allocation is rejected; no nonvolatile XMM saves were introduced.

The artifact-only scalar decoder projects admitted CMIRAMD4 instructions into
the existing BackendFunction analysis view, retaining exact representation,
scalar operation and source locations. Arithmetic, fneg, constants, slots,
array address/index uses and comparisons participate in allocation. Float compare
operands are XMM and the bool result is GPR. This is an abstract three-address
planning view, without SSE selection, constant materialization or instruction
encoding. Call types are derived from the checked function's 128-entry type map;
no duplicated type arrays were added to each instruction. CMIRAMD3's larger
virtual capacity is protected by checking the scalar discriminator before
accessing this map.

### One mixed call plan

Existing `CallRequest`/`CallPlan` carry typed scalar locations and moves. The
legacy integer request adapts into the same positional placement core; no
parallel floating call-plan model or Go placement policy exists. For arguments
0..3, the argument ordinal selects the GPR or XMM position:

| Signature | arg0 | arg1 | arg2 | arg3 |
| --- | --- | --- | --- | --- |
| i32, float64, i32, float32 | RCX | XMM1 | R8 | XMM3 |
| float64, i32, float32, i32 | XMM0 | RDX | XMM2 | R9 |
| i32, float64, bool, float32 | RCX | XMM1 | R8 | XMM3 |

The eight-argument fixture continues with float64, float32, i32, bool at outgoing
stack slots 0, 1, 2, 3, offsets 32, 40, 48, 56 from post-prologue RSP. Shadow space
remains exactly 32 bytes. Outgoing storage uses eight-byte ABI slots, while the
typed call-site destination and transfer metadata retain each value's exact
representation, class and copy width. Geometry is shared across calls; it does
not pretend a reused outgoing slot has one permanent semantic type.

Float32/float64 call results originate in XMM0; integer/bool results use RAX;
void has no result handoff. Call result capture precedes preservation reloads,
including a pressure-spilled result. Each call clobbers the existing volatile
GPR set, XMM0..XMM5, and FLAGS. Stale FLAGS, unsupported classes/forms/conventions,
invalid locations, reserved XMMs, incompatible moves and capacity failures have
named Concept errors. The native qualification path preserves ABI diagnostics
through the checked call-summary helper; the legacy wrapper retains its existing
BackendError interface.

### Shared frame and move realization

The existing finalizer retains its seven regions: shadow, outgoing arguments,
semantic locals, callee saves, spills, move temporaries, padding. There is one
frame and one post-prologue RSP coordinate system. Float32 spills occupy four
bytes with alignment four; float64 spills occupy eight with alignment eight.
Spill representation determines class and is checked alongside width, identity
and offset. No scalar spill uses a sixteen-byte XMM-sized allocation.

Move cycles use the existing generalized eight-byte, alignment-eight temporary
slots, with exact four/eight-byte typed transfers. GPR and XMM physical identities
occupy distinct graph domains. Two-way float32 and three-way float64 cycles,
acyclic moves, repeated sources, XMM-to-outgoing and spill/temp-to-XMM edges are
qualified. Mixed float32/float64 cycles are explicitly rejected with
`AMD64_MOVE_INCOMPATIBLE_REPRESENTATION`; no implicit conversion is assumed.
Pressure storage copies are abstract typed actions. Their eventual SSE scratch
register and instruction materialization are outside this slice.

| Planning fixture | Frame bytes | Relevant storage |
| --- | ---: | --- |
| Live32 | 40 | local float32 at 32, live spill at 36, each 4/4 |
| Live64 | 56 | local float64 at 32, live spill at 40, each 8/8; padding 8 |
| Pressure32 / Pressure64 | 8 / 24 | two pressure spills, 4/4 or 8/8 |
| CallA / CallB | 40 | shadow 32, padding 8 |
| Call8 | 72 | shadow 32, outgoing 32, padding 8 |
| CallFive | 40 | shadow 32, fifth scalar at 32 |
| Cycle2 / Cycle3 | 40 | shadow 32, generalized temporary 8 |
| MixedLive | 56 | i32, float32 and float64 preservation in one frame |
| MixedPressure | 104 | six XMM preserves, GPR/bool preserves, pressure-spilled result |

Call-bearing frames satisfy frame byte count modulo 16 equals eight, accounting
for the Win64 return-address entry offset. The existing integer frame harness
still qualifies its 136-byte maximum specimen. Integer native bytes and the
CMIRAMD3/CMIRAMD4 schemas, generated codecs and metadata remain unchanged.

### Evidence and limits

`TestEVT2f3ScalarAllocationABIAndFrames` runs 100 fresh allocation, ABI, frame and
concrete action plans per function for 61 functions (the existing 32 plus 29 new
planning functions), in Normal and Verify. It compares canonical assignment,
spill, frame and call-action observations across runs and modes. Independently
compiled C11 byte-stack replay checks exact float payload bits through abstract
calls and explicit architectural clobber tables. This executes compiled Concept
planning and a test-side call simulator, **not emitted floating machine code**.
Both mixed ABI examples have independent expected-location tables. All 256
four-source XMM combinations, simultaneous separate GPR/XMM cycles, repeated
sources, bool/RAX result placement, cycle temporaries, outgoing
ordering, capture-before-reload, live-across counts, class separation and exact
spill geometry are checked. Mutated transfer types, missing moves, stale FLAGS,
reserved XMM6..XMM15, insufficient spill capacity, frame offsets/widths and
capture/reload reordering are rejected. Representation-swapped cycles reject.

Three checked negative source specimens reject binary16, a four-lane record
wrapper and a floating aggregate before scalar MachineIR allocation. The wrapper
is an unsupported aggregate-shaped vector specimen; no SIMD frontend capability
is implied. Varargs reject at the typed ABI request seam. Existing transport
qualification still runs 100 schema/artifact repetitions, 52 malformed artifacts,
exact constants/digests and untouched output/native-image sentinels. Source C11
floating behavior remains the independent source oracle.

One informational run measured 100 plans per function over the existing 32:
allocation 13 ms, ABI below 1 ms timer resolution, frame 21 ms (Normal); 10/0/26 ms
(Verify). The additional 29 measured 20/3/27 and 20/1/23 ms. These are aggregate
CPU-clock samples with millisecond resolution, not performance thresholds.
The final shared C bootstrap metadata sizes are CallRequest 1,100 bytes,
CallPlan 1,588 (previous 660; delta 928), ScalarLocation 20, Preservation 56,
BackendFunction 158,052, Allocation 17,444, FinalizedFrame 15,868 and
CallSiteActions 4,156. Bounded shared-build reuse remains enabled; no heap fallback
or routine cold-build stress was added.

Go changes are bootstrap orchestration and diagnostic presentation
(`cmd/concept/amd64.go`), independent qualification oracles and CLI tests. Go
injects only the checked scalar function count; Concept produces allocation,
placement, clobbers, preservation, geometry and actions. Stage-0 remains the
seed and checked semantic/transport producer. No production Go floating backend
algorithm was introduced. Typed PlainData, StaticExtent-four and NoAllocation
assertions are checked through the real backend compiler.

The convergence ledger records exact baseline/final validation and commit IDs.
Frozen Zig is skipped because its paths are unchanged. Format checks and real
Normal/Verify R9d match-preference lint pass for authored planning sources.
The semantic manifest remains 440 valid / 357 static-invalid / 15
runtime-negative / five compatibility / four expected-divergence; EVT2f planning
and negative source specimens are checked by dedicated tests outside that
manifest. SSE encoding, constant materialization, nonvolatile XMM saves, SIMD,
aggregates, varargs, external/indirect calls and native float execution remain
unqualified. EVT2f4 was not begun.


## EVT2f4 — Scalar SSE2 native arithmetic

The EVT2f1–f3 sections above are historical qualification records. The new
no-call CMIRAMD4 path emits executable scalar SSE2 through Concept `EmitModule`.
Call-bearing scalar modules now return `AMD64_UNSUPPORTED_FLOAT_CALL_EMISSION`
before output/image mutation. EVT2f5 was not started.

### Baseline and authority

At the user's direction, the implementation uses the latest optimization
baseline `9d57d649b0edca082c0af6f15b9827a05ef363e1`, rather than reverting the
NURBS/SMLIB-related changes to the originally requested EVT2f3 commit
`6f03274541fcb9a6426c99b04dfbae3e0bf21233`. That latest branch lacked the EVT2f
prerequisites. Their changes reconciled cleanly onto it and are preserved in
checkpoint `aef7824adfd720f00bfdad28e882077879c8d7fa`. Compiler identity remains
`concept-evt1-stage0-go`. CMIRAMD3 and CMIRAMD4 schemas/codecs/metadata are unchanged
from EVT2f3. No SSE selection or encoding was added to Go.

| Changed code | Classification |
| --- | --- |
| `AMD64.concept` SSE selection, operand admission, encoders, constants, compare lowering, entry/return, module emission | Concept authoritative |
| Existing allocator and Win64ABI policy | Retained Concept authority; no allocator rewrite |
| `nativeTemporaryCount` and native-frame qualifier | Derived bounded frame request; existing finalizer owns every offset |
| `cmd/concept/amd64.go` | Bootstrap diagnostic presentation; calls the Concept native-frame qualifier |
| CLI boundary tests and `machineir_scalar_test.go` | Regression/orchestration oracle updates |
| `scalar_sse_encoding_test.go` | Independent C11/native/byte/ABI-entry oracle and executable-memory host |
| Restored EVT2f1–f3 Go files | Checked-input/transport prerequisites, reconciled with latest optimization work |

### Instruction selection and destructive destinations

Selection uses exhaustive semantic opcode and exact representation cases:

| Operation | float32 bytes before ModRM | float64 bytes before ModRM |
| --- | --- | --- |
| MOV load/register copy | F3 0F 10 | F2 0F 10 |
| MOV store | F3 0F 11 | F2 0F 11 |
| ADD | F3 0F 58 | F2 0F 58 |
| SUB | F3 0F 5C | F2 0F 5C |
| MUL | F3 0F 59 | F2 0F 59 |
| DIV | F3 0F 5E | F2 0F 5E |
| UCOMI | 0F 2E | 66 0F 2E |

REX follows mandatory prefixes and precedes 0F. Register/register,
register/RSP-memory and RSP-memory/register forms use typed operands. Indexed
floating storage first computes an admitted GPR address with LEA, then uses the
scalar load/store form at that address. Encoder-only high-XMM extension is byte
qualified (`F2 45 0F 58 C1` for ADDSD XMM8,XMM9); allocation still reserves
XMM6..XMM15. No AVX, packed arithmetic, FMA or nonvolatile XMM saves were added.

Three-address operations borrow XMM5 with an explicit save of its scalar bits,
compute in that register, copy/store the result into its validated allocation,
and restore XMM5 unless it is the result's assigned location. An operand already
in XMM5 is read from its saved storage. RAX is likewise saved/restored except
when it is the bool result. This handles destination/right-source overlap for
SUB and DIV without changing allocation decisions or destroying live values.
All actions consume the existing frame's resolved slot offsets.

Native no-call frames request four existing generalized eight-byte move-temp
slots. These initially capture the first four incoming argument positions, then
serve as scratch for XMM5, RAX and comparison parity materialization. Stack
parameters use post-prologue RSP plus finalized frame bytes plus the Win64 entry
stack argument offset. Floats use positional XMM0..XMM3; GPR parameters use the
existing ordinal table. Floating returns use XMM0; bool returns are normalized
GPR values. This qualifies directly invoked entrypoints, including array
pointer/index/value entry, without implementing generated floating call sites.

### Constants, negation and strict semantics

Constants are materialized as deterministic immediate bits: MOV EAX/ RAX with
four/eight little-endian literal bytes, followed by MOVD/MOVQ to XMM. The Concept
encoder decodes the admitted fixed lowercase hexadecimal literal directly,
without decimal conversion, process addresses or a literal pool. Therefore
constant-pool identities, alignment and relocations are not applicable. Branch
fixups remain bounded and are patched before the image becomes finalized and
before the host publishes RX memory. Failed emission is never executable output.

Negation uses MOVD/MOVQ to RAX, BTC at bit 31/63, and MOVD/MOVQ back. It does not
subtract from zero or use packed XOR. Signed zero, infinities and NaN payload bits
apart from the sign are preserved. Tests pin both negation sequences and execute
positive/negative zero and payload-bearing quiet NaNs.

No fast-math, reassociation, contraction, FMA or zero canonicalization is enabled.
Operation order follows admitted MachineIR. Native operations use the host's
MXCSR state; qualification assumes the ordinary default, masked exceptions,
round-to-nearest-even and gradual underflow. No floating environment/exception
subsystem was introduced. Arithmetic NaN payload propagation is not claimed to
be bit-identical across C compilers or processors: exceptional arithmetic is
checked by NaN category, otherwise by exact bits. Constant/copy/negation cases
check exact payloads. Explicit numeric conversions remain unsupported. Quantity
interpretation is a checked upstream zero-op; native identity cases preserve -0.

### Comparisons and FLAGS

UCOMISS/UCOMISD is consumed by adjacent bool materialization:

| Relation | Boolean construction |
| --- | --- |
| == | SETE AND SETNP |
| != | SETNE OR SETP |
| < | SETB AND SETNP |
| <= | SETBE AND SETNP |
| > | SETA |
| >= | SETAE |

SETP/SETNP uses saved volatile RDX; MOV-based save/restore does not change FLAGS.
Results are normalized into the assigned bool/GPR location. No XMM bool or
outstanding floating FLAGS value escapes this atomic semantic operation. Branch
lowering tests the materialized bool immediately before its conditional branch.
The existing numbered integer FLAGS/call restrictions remain in force, including
the EVT2f3 stale-FLAGS negative tests. Full floating calls remain rejected.

### Storage and native evidence

`native_scalar.concept` contains 50 no-call functions. Compiled Concept emits one
5,332-byte image, then the Windows AMD64 harness uses RW allocation, a complete
copy, RX protection, instruction-cache flush and correctly typed native
entrypoints. The independent strict-C11 oracle uses no fast-math and compares:

- Eight scalar arithmetic forms, both noncommutative orders and multi-operation
  expressions, plus both widths' sign-bit negation.
- 588 materialized comparisons and 588 comparison/branch cases over seven-value
  domains: negative finite, +0, -0, positive finite, +/-infinity and payload NaN.
- Exact float32 `0x3dcccccd` and float64 `0x3ff0000000000001` source constants.
  Primitive native constant tests additionally check signed zeros, infinities
  and payload-bearing NaNs as exact bits.
- Scalar slot loads/stores, typed incoming array loads/stores with stride 4/8,
  and payload-preserving array stores. Existing checked-index admission remains
  authoritative; native unsigned extent comparison rejects negative/too-large
  indices with UD2 rather than bypassing the guard. Invalid traps are not invoked
  inside the parent test process.
- Eight simultaneously live float32/float64 parameters, real pressure spills,
  reloads and arithmetic execution through typed Win64 entrypoints.
- Quantity interpretation, exact signed zero, and finite/infinite/zero division
  categories, including zero/zero and infinity/infinity.

Primitive exact bytes pin the twelve requested load/arithmetic/UCOMI forms,
MOVSS/MOVSD stores, one RSP spill/reload sequence, both negation sequences and
high-XMM REX. Typed malformed operands reject representation/class/width errors,
invalid memory form, negative/out-of-range displacement, unsupported operation,
invalid constant width and insufficient capacity. The shared byte writer now
also rejects a negative cursor before indexing caller storage; valid integer
bytes are unchanged. There is no literal pool requiring unresolved-pool tests.

`EmitModule` validates the entire artifact, rejects floating calls before any
output/image mutation, emits directly into the same caller-owned writer and
finalizes only after resolving function branches. It does not allocate a second
native runner or a temporary giant code array. All 100 complete module emissions
match final bytes, symbol metadata and layout. Normal/Verify match exactly.
Existing EVT2f3 61-function/100-run planner coverage and EVT2f2 transport/52-invalid
artifact tests remain qualified. New metadata has PlainData/NoAllocation proofs;
existing StaticExtent-four register-table proof remains checked.

### Measurements and limits

In one concurrent qualification run, 101 full 50-function emissions took
19,414 ms Normal and 18,907 ms Verify, including repeated artifact validation,
allocation and frame realization. The image contains 5,332 bytes. Ten thousand
selection and low-level encoding calls were below the millisecond timer's
resolution; immediate constant materialization measured 0–1 ms. The 100,000-call native arithmetic sample was below one millisecond at this
timer resolution; it must not be interpreted as zero execution cost. These are informational clock()-based
samples with host/call-loop overhead, not claims of superiority over C/C++.

The qualified subset is scalar float32/float64 no-call arithmetic, comparisons,
scalar locals, incoming checked indexed floating arrays, pressure spills and
minimal Win64 entry/return. Locally initialized indexed aggregates, additional GPR operations (including
integer/bool constants and slots) inside mixed CMIRAMD4 modules, conversions, binary16, SIMD,
AVX, external/indirect/variadic/aggregate calls, COFF and unwind/SEH are not
qualified. Pure integer modules retain their frozen CMIRAMD3 path. All generated
floating calls, internal call chains and actual live-across-call floating
execution remain EVT2f5 responsibilities, with the precise call boundary above.
Final validation/identities and updated latest-baseline counts are recorded in
the convergence ledger. Frozen Zig is skipped because its path was unchanged.
