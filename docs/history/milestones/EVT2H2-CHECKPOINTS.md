# EVT2h2 qualification-gap closure

Baseline: clean `main` at `934e5b4859b201354331b1eca0b2fbe286af47fd`,
equal to fetched origin/main. Compiler `concept-evt1-stage0-go`, Windows AMD64,
Microsoft x64. Existing h2 checked-LIR semantics and Windows cast-trap executable
isolation are retained. No native match or fallible-type milestone is started.

## 1. CMIRAMD8 contract and admission

Committed as `8698781`.

Concept `BridgeSchema8` owns the v8 header, nominal sum domains, ordered variants
and operation identities. The envelope reuses frozen `WireBorrowModule` and
`WireObjectField` records; canonical identities, geometry, copy permission,
operands, values and source provenance are not duplicated. Sum type fields are
opaque in the base object table. No old record or integer acquires sum semantics.

The operation table binds construction/checked projection to exact function,
block and instruction positions. Complete construction checks every active
payload operand and its type. Atomic checked projection binds a logical sum
snapshot, selected variant and field; it never accepts an address as its subject.
Its eventual native selector must check that same snapshot's tag before reading.
There is no separately forgeable Boolean promise authorizing a byte read.

Generated Go/Concept codecs and JSON schema have SHA256
`8dc34f916ab8293e838db026d720825f10d2a7c12006c19924a0fffea94c0711`.
The header is 44 bytes. Checked limits retain 256 object types, 128 functions,
512 values, 64 slots and 1024 instructions per function. Sum domains are bounded
to 128, variants to 256, payload fields to 64, fixed arrays to 256 elements and
logical storage to 1 MiB. Operation metadata is bounded to 131072 sites.

`BridgeValidate8` independently admits artifacts and validates domains, variants,
field geometry, complete initialization, copy eligibility, operation sites and
the existing object/reference/CFG contracts. Legacy validator entrypoints pass
empty extension contexts and retain their old acceptance language. Schema,
generated-code and wire bytes for CMIRAMD3–7 remain unchanged.

The source-free native C host links only compiled Concept admission code. Normal
and Verify reject 16 malformed artifacts without modifying a round-trip output
sentinel, then perform 100 exact valid round trips. This is artifact-only
Cathedral admission, not yet h2 machine-code execution.

The 23-function transport specimen includes seven internal atomic projection
wrappers without adding source syntax. Artifact size is 31116 bytes, SHA256
`2fe36aaacdf23413eeca5a1b0cf6d4d200c1c6698d450abcc5318c0a3e082642`.
100 fresh source/LIR/MachineIR/debug generations plus 100 fresh codec generations
passed in 1.194 s; independent artifact admission passed in 14.04 s.

## 2. Native tagged operations

Implemented in the existing `AMD64.Objects.CopyObjectFields` recursion and
`AMD64.EncodeObjectMemoryInstruction`. Construction writes active fields then
the tag. Copy dispatches on the admitted tag and visits only active meaningful
fields through the same record/array/scalar copier. Invalid tags trap. Atomic
projection checks the same private snapshot before loading the selected field.
There is no second copy engine, union layout producer, allocator or frame plan.

The expanded 20-function source includes forward calls, recursion, nested calls,
mixed arguments, evaluation order, variant replacement and distinct nominal
types with identical geometry. Seven internal LIR projection wrappers add no
source syntax. Fixed-array projection returns an existing checked ArrayBox
record, preserving the standalone-array ABI rejection.

Normal/Verify artifact-only execution and independent strict C11 parity passed.
The image contains 27 functions / 29,745 bytes, SHA256
`188ac9b4bdeeaa2c2f0dcfa4e5496660f4dab4f7746ba593862895b3042b40c7`.
Frames measured 40..1096 bytes. 100 complete physical plans per function,
including canonical debug descriptions, and 100 emissions are identical.
Measured emission 72..75 ms; 100 whole-module physical plans 1.491..1.564 s.
The current 36,605-byte CMIRAMD8 artifact has SHA256
`1a7372b2cb15862274d6595c4028e559b39ccd77060d471c0bbd281fb45c8c06`;
100 fresh generations/codecs passed in 620.643 ms.

## 3. Native ABI and closeout

The existing Win64 aggregate classifier/transfers realize 8-byte sum carriers
directly, 12/48-byte values indirectly, and hidden results with argument shifts.
Ordinary native records retain their established conservative hidden-result
rule. Separately compiled MSVC and clang-cl C DLLs pass both calling directions
for tagged values, mixed positional arguments and fifth/later stack arguments.
These probes grant no source foreign enum representation or external linker.

22 malformed artifacts reject in Go and Concept, with native output and image
sentinels retained. Three child-process trap probes reject inactive and forged
tags, including payload storage behind a protected page. This detects accidental
inactive-field reads as access violations instead of the required UD2 trap.
Existing cast-trap executables remain separate per case; three fresh focused
runs pass in 9.245 s. Focused new semantic/transport race passes in 11.485 s.
EVT2h1 native/transport frozen hashes remain exact.

Full qualification is pending. The final conformance/convergence entries must
record its actual outcome before EVT2h2 success. Earlier source-bootstrap
evidence remains distinct from the native evidence above.
