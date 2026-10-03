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
