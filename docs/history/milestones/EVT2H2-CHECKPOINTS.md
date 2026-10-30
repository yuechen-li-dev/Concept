# EVT2h2 qualification-gap closure

Baseline: clean `main` at `934e5b4859b201354331b1eca0b2fbe286af47fd`,
equal to fetched origin/main. Compiler `concept-evt1-stage0-go`, Windows AMD64,
Microsoft x64. Existing h2 checked-LIR semantics and Windows cast-trap executable
isolation are retained. No native match or fallible-type milestone is started.

## 1. CMIRAMD8 contract and admission

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

Pending: extend the existing object storage/copy/selector owner with active-tag
dispatch and atomic checked payload reads, using the shared allocator and frame
planner. This section must record actual native execution before qualification.

## 3. Native ABI and closeout

Pending: existing Win64 aggregate machinery, independent C11 differential and
ABI probes, malformed native-output sentinels, 100 fresh emissions, exact bytes,
Normal/Verify parity and full regression. The trap-path regression remains part
of qualification. Earlier source-bootstrap evidence is not native h2 evidence.
