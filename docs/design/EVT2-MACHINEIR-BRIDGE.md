# EVT2 MachineIR bridge: one checked schema

> The MachineIR bridge has one checked semantic schema. Producer and consumer
> codecs are derived from that schema; field order and representation are not
> maintained manually in Go and Concept independently.

> Stage-0 Go produces MachineIR using a generated bridge codec. The
> Concept-written AMD64 backend consumes it using a generated Concept codec.
> Both are generated from the same schema authority.

## Authority and staging

`libraries/Standard/Backend/BridgeSchema.concept` is the authority. It contains
ordinary checked records, payload-free enums, typed wire selectors, and checked
magic/version/count-limit/representation constants. Explicit reflection requests
produce TypeInfo/FieldInfo/EnumCaseInfo through the existing semantic analyzer.
The generator consumes those objects, including R9a structured generic
applications; it never reconstructs field types or enum identity from printer text.

The bounded staging is:

```text
checked Concept BridgeSchema + typed reflection
    -> cmd/machinebridgegen (build-time only)
        -> machineir_bridge_codec_generated.go
        -> BridgeCodec.concept (identity, derive sites, counted-view wrappers)
    -> BridgeDerive Fields<T>/Cases<T>/FieldType/TagOf generators
        -> ordinary checked Concept FunctionDecls
Stage-0 MachineIR -> generated Go codec -> CMIRAMD3
    -> generated Concept reader -> typed wire views -> AMD64 semantic projection
```

`BridgeDerive.concept` supplies ordinary record and enum read/write generators.
Generated declarations carry GeneratedByReflection provenance and appear through
`concept generated`; `concept explain --generated <identity>` uses the existing
explain machinery. Imported artifacts retain the schema, structured applications,
generator bodies, generated declarations, constants and provenance. Artifact-only
schema regeneration and decoder derivation require no dependency source.

The emitter is specific to this bridge. Its supported byte primitives are fixed
integers, exact booleans, text, fixed header byte arrays, typed enums, records and
counted sequences. It rejects unsupported wire types and unreflected schema
records/enums. `bridge_value` binds semantic MachineIR string tags to checked enum
cases; `bridge_go_type` supplies typed Go conversions for LIRType and MachineReg.
The `Wire` prefix maps checked declaration identifiers to existing Go semantic
record identifiers; generic applications are handled structurally, never by
splitting a generated nominal name.

## Coverage and representation

There are 14 wire records with 81 ordered fields and seven enums with 65 cases.
The normalized inspectable definition is
`docs/design/EVT2-MACHINEIR-BRIDGE.schema.json`; it is generated, not another schema
language or authority.

| Checked record | Transported information |
| --- | --- |
| WireHeader | Eight magic bytes, u32 version, 32 SHA-256 bytes |
| WireSpan | Source line and column |
| WireMachineFrame | Local size, alignment, shadow space, calls |
| WireMachineOperand | ID, width, base, base-slot, index, scale, displacement, kind, literal, region, signedness |
| WireMachineArg | ABI type, index, width, indirectness, physical register, stack offset, stack placement, vreg |
| WireMachineVReg | ID, width, address role |
| WireMachineStackSlot | ID, size, alignment, incoming indirectness, base vreg, source |
| WireMachineCallArgument | Virtual value ID and exact scalar class |
| WireMachineCall | Direct kind, stable target identity, convention, ordered arguments, result class |
| WireMachineInstruction | Opcode, destination, sources, width, flags definition/use, LIR block/instruction, condition, span, decision, facts, optional call contract |
| WireMachineTerminator | Opcode, true/false targets, flags use, condition, span |
| WireMachineBlock | ID, LIR block, instructions, terminator |
| WireMachineFunction | Identity, name, target, ABI, result, span, facts, decisions, frame, arguments, vregs, slots, blocks |
| WireMachineModule | Functions |

All scalar wire values are explicitly little endian. EVT1's canonical `int` is
signed 32-bit (pinned by SizeOf), `uint` is unsigned 32-bit, and `uint8` is eight
bits; formatting canonicalizes the uint32 alias to uint. Booleans and enum tags
occupy four bytes. Text is a u32 UTF-8 byte length followed by bytes. A sequence
is a u32 count followed by recursively encoded elements. Count/text limits come
from the checked BridgeMaxItems constant (1,048,576). Neither Go struct layout
nor C padding crosses the boundary. repr(C) would add an irrelevant host ABI;
this is a compact byte protocol.

BridgeText's offset/length and BridgeSequence<T>'s offset/count are bounded input
views, not fields transported as host structs. BridgeType<T> is an overload
witness, not runtime reflection. Their view layout is internal to the consumer.
The current MachineIR has only general-purpose scalar register views; register
width, address role and all physical constraints are transported. No extra
register-class field existed in CMIRAMD1. Canonical closed LIR ABI descriptors
are preserved as text; no Concept nominal/generic semantic identity is recovered
from those descriptors. No untransported Go pointers or inspection-printer data
were added.

## Version and schema hash

The live identity is `CMIRAMD3`, numeric version 3. The generated header size is
44 bytes: magic, version, raw SHA-256. EVT2e2 adds a counted call-contract field
to instructions. A non-call instruction carries count zero; CALL carries one
record. Existing fields and tags retain their representation. CMIRAMD2 and its
hash are rejected; no implicit upgrade or runtime compatibility codec exists.

Current hash:

```text
3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312
```

SHA-256 covers deterministic normalized checked metadata: schema module, magic,
version, representation contract, count limit, source-ordered record fields and
structural type arguments, fixed array extents, enum names/tags/string bindings,
and typed Go conversions. Spans, paths, timestamps, trivia, declaration printer
output and generated code are excluded. Header hash contents are derived output,
so there is no self-hashing cycle. Wire changes require a deliberate version
migration; a checked fixture field mutation changes the hash and both generated
sides, and a stale producer is rejected by the mutated native Concept consumer.

## Generation and inspection

From the repository root:

```powershell
go run ./cmd/machinebridgegen .
go run ./cmd/concept package build Standard
$env:CONCEPT_MODULE_ROOTS = "$PWD/artifacts/Standard/modules"
go run ./cmd/concept generated libraries/Standard/Backend/BridgeCodec.concept
# Select an origin.identity from generated output:
go run ./cmd/concept explain libraries/Standard/Backend/BridgeCodec.concept --generated <identity> --json
```

Go output is gofmt-normalized; Concept derive-site output uses the canonical
formatter. Checked generated-file regressions fail on stale outputs. The emitter
is a development/bootstrap tool, not part of runtime EncodeMachineBridge. Changing
the schema regenerates both sides from the checked declarations. Concept readers
are materialized as AST declarations by the existing generation phase, and
artifact-only imports consume that AST directly.

## Decode, bounds and errors

Generic audited byte IO remains handwritten in BridgeRead and machineir_bridge.go.
Record field order and enum tables are generated. The production Go invocation
and typed AMD64 projection contain no alternate manual wire decoder or fallback.
Header identity is checked before the function payload. Go distinguishes
MIR_BRIDGE_SCHEMA_MISMATCH, MIR_BRIDGE_VERSION_MISMATCH and
MIR_BRIDGE_SCHEMA_HASH_MISMATCH. Concept returns BridgeVersion for magic/version,
BridgeSchemaMismatch for hash, and BridgeInvalid for ordinary malformed input.
Truncation, invalid enum/bool/register tags, negative/oversized counts, impossible
text lengths, trailing bytes and invalid block/vreg references are rejected.
Virtual-register bounds are established before indexed width observation, so an
invalid reference returns Result rather than triggering a bounds panic.

The generated Concept reader validates all elements, including unselected
functions. Typed views retain every transported fact, decision and provenance
field. AMD64 projects only fields needed by the current backend; it does not
re-encode or discard the canonical wire model. Native canonical decode/re-encode
proves complete byte preservation, including fields the old projection skipped.
Views borrow input bytes; there is no hidden heap, runtime serializer or new
arena ownership. Backend capacities remain arguments 8, vregs 512, slots 64,
blocks 128 and instructions 512. CapacityExceeded remains explicit. Encoding
keeps the R9a caller-supplied workspace and compatibility entry points.
NoAllocation is pinned through the real checked call graph and existing backend
assertions, not a codec-specific fact grant.

`BridgeValidate` adds direct-call semantic admission to the normal backend path:
exact target lookup, ordered scalar classes, virtual widths and result shape,
source/LIR provenance and single-record cardinality. `ValidatedBridgeRoundTrip`
validates before writing caller output, then uses generated readers/writers.
Neither decoding nor roundtrip needs the originating source. Runtime errors use
Result; the wire-record PlainData assertions use Vocabulary's typed Verdict
with size/alignment evidence. No PlainData assertion grants ABI equivalence.
EVT2e3 derives transient physical ABI plans from these same semantic records.
Selected CALL functions stop at `AMD64_UNSUPPORTED_CALL_FRAME_LOWERING` after
verified placement, moves and preservation analysis, before frame realization
or encoding. The schema is unchanged. See [EVT2e2 conformance](../conformance/EVT2E2-CONFORMANCE.md)
and [EVT2e3 planning](../conformance/EVT2E3-CONFORMANCE.md).

## Migration and evidence

The strangler sequence is complete: shadow, whole-payload agreement, native-byte
agreement, production switch, deletion. Commits 86f6a5c and a588dc2 retain the
reviewable shadow/switch history; 7cb01b0 removes manual Go record/sequence
encoding and decoding. ReadOperand/ReadFunction field decoding and manual Concept
tag tables are gone. The old Go and Concept implementations do not ship as test
fallbacks. Frozen `.cmir1` and `.amd64` outputs in testdata/machinebridge were
captured from the exact baseline after live differential qualification.

The seven corpus shapes cover 17 functions: Add/Max/Sum4/CheckedIndex/StoreIndex/
Choose/Early, finite/yield/multi-yield machines, pushdown automata, activation
initialization and arbitrary stride-12 addressing. Tests compare every payload
byte and semantic field, preserve complete Concept canonical wire data, and
compare final AMD64 bytes. EVT2e2 removes only the traced four-byte empty call
sequence per non-call instruction when comparing to frozen legacy payloads;
the frozen artifacts and native bytes remain untouched. Optional generated field traces explain a mismatch
with record, field, expected/actual offset and byte values. There is no opaque
manual byte-array debugging requirement.

Octagon's existing checked reflection/generation seam was reused. Switching the
transport to Octagon would break established compact-byte parity and require
more migration machinery; R9a2 therefore keeps the bounded binary bridge.
No universal serialization framework, R9b work or EVT2 feature expansion is part
of this change. Qualification, timing and freeze-readiness evidence are recorded
in R9A2-CONFORMANCE and R9A2-CONVERGENCE.
