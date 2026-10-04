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
