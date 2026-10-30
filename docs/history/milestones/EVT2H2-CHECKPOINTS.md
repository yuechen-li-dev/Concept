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

Committed as `0da865b`; the measurements in this section describe that interim
27-function specimen. The completed qualification fixture follows in checkpoint 3.

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

28 malformed artifacts reject in Go and Concept, with native output and image
sentinels retained. Three child-process trap probes reject inactive and forged
tags, including payload storage behind a protected page. This detects accidental
inactive-field reads as access violations instead of the required UD2 trap.
Existing cast-trap executables remain separate per case; three fresh focused
runs pass in 9.245 s. Final semantic/transport race passes in 15.247 s, with
source-free imported transport covered by a further 2.053 s race run.
EVT2h1 native/transport frozen hashes remain exact.

`4a175e6` completes the 22-function source fixture (29 native functions with
internal observers). FloatingPressure uses seven positional double arguments,
including stack positions shifted by a hidden result, and demonstrably spills
nine floating values through the existing allocator. Separate MSVC/clang-cl
callers exercise that path too. A qualification attempt was interrupted to add
this missing coverage, then restarted; no failure was waived.

Final CMIRAMD8: 42,064 bytes, SHA256
`1215b1c779093e9fc149929d8ca4ad0b6f81a1a6aa0d77c56341d5e3372125e5`.
100 fresh generations/codecs: 668.665 ms. Final native image: 29 functions,
18 calls, 31,871 bytes, SHA256
`524c364540dbb5efc8b1634f5d7087f8933fd4225d92afaa73dc39bccd7becc3`.
Normal/Verify exact parity remains; frames 40..1096 bytes. Emission measured
82..84 ms, and 100 complete physical plans per function 1.699..1.719 s.
Independent native/C11 execution, ABI probes, checked trap guards, tiny-output
sentinels and corrupt derived argument/result/frame plans pass. The final quick
lane passes in 24.16 s. Standard/DragonGod pass in both modes (44/23 tests), as
do package build, format/lint and derived-code inspection.

Full qualification at `4a175e6` passed in 1572 s including vet (compiler/native
1569.289 s, CLI 74.861 s, GPU-free Vulkan 0.506 s). A final audit found a local-only
module route without count-only preflight. `59788a4` sends all v8 modules through
the existing aggregate preflight and adds `Testing.TaggedLocals` as an actual
checked source regression. Normal/Verify execution and unchanged tiny-output /
image sentinels pass, focused package 70.666 s; quick passes in 24.14 s.
The repeated final lane at `59788a4` passes: compiler/native 1557.665 s, CLI
81.251 s, GPU-free Vulkan 0.480 s; total 1564.97 s including vet. Frozen native,
corpus, Golden and package gates are included. Format/lint and derived inspection
remain clean. Frozen Zig is skipped because unchanged. **EVT2h2 SUCCESS**; work
stops here, with EVT2h3 requiring separate review. Earlier source-bootstrap
evidence remains distinct from native execution.
