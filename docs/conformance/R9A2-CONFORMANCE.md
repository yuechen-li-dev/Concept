# R9a2 conformance

Result: SUCCESS. Qualified code SHA:
`7cb01b039bc1a9c2fdf61073a67e5b3f8ebdbca9`.
Baseline: `653762bca74677c46760aad478a4e5450c41a5b1`, clean on
`codex/r9a-stage0-closure`. Compiler: `concept-evt1-stage0-go`.
This closeout changes the MachineIR bridge, not frontend self-hosting or EVT2 scope.

## Baseline and environment

Windows AMD64, Go 1.27.0, GCC 15.2.0, Zig 0.16.0. The baseline bridge was CMIRAMD1:
manual Go field encoder/decoder in internal/concept/machineir_bridge.go and manual
ReadOperand/ReadFunction/tag decoding in Standard.Backend.AMD64. Its header had
only eight magic bytes; no numeric version or content hash. All payload integers
were LE i32/u32, strings and sequences counted. Existing Fields/Cases/FieldType,
checked generated FunctionDecls and semantic artifacts supplied the derivation
seam; there was no build-time Go codec emitter.

The first full baseline command overlapped new library-file discovery and failed
two typed-store tests on a transient int32 schema spelling. That run is rejected
as baseline evidence, not described as an environment-only failure. An isolated
clean checkout of the exact baseline passed `go test ./... -count=1` in 332.545s.
Baseline vet, race, Standard 42/DragonGod 23/Golden 130/Vulkan 12 in Normal and
Verify, and both Zig suites passed. The isolated checkout's local TinyXML2 reference
clone was refused because the reference repository was shallow; its optional
submodule path was not initialized. The final main checkout retains the existing
initialized TinyXML2 submodule and passes the complete suite there.

The user then changed the validation policy: the frozen legacy Zig compiler and
its build/test infrastructure require Zig tests only when those paths change.
Both final Zig suites are explicitly SKIPPED because no such path changed.
The revised skill is installed at
C:/Users/yuech/.codex/skills/concept-evt1-milestone-validation/SKILL.md;
a memory update note points future runs to it. No frozen Zig source changed.

## Final regression gates

| Gate | Evidence / result |
| --- | --- |
| go test ./... -count=1 | PASS; internal/concept 377.079s; Vulkan profile 0.349s |
| go vet ./... | PASS |
| Race: R9a2 bridge/identity/native/determinism plus established innate scale | PASS, 117.863s |
| Standard Normal / Verify | 42 / 42 passed |
| DragonGod Normal / Verify | 23 / 23 passed |
| Golden Normal / Verify | 130 / 130 passed |
| GPU-free Vulkan Normal / Verify | 12 / 12 passed |
| Semantic corpus | PASS, 426 valid / 337 static-invalid / 15 runtime-negative specimens |
| Focused EVT2 native | Core seven functions, finite/yield machines, dynamic addressing passed |
| EVT2x6 native/C oracle traces | Normal and Verify; pushdown, nested frames, stride-12 source fixture, named invalid-state stops passed |
| Strict C11 | Generated backend, roundtrip, malformed-input and stale-schema harnesses passed |
| concept generated / explain | 31 codec declarations exposed; generated identity explanation passed |
| Concept format --check / lint | AMD64, BridgeSchema, BridgeRead, BridgeDerive, generated BridgeCodec all passed |
| Generated Go | gofmt output reproduced exactly; vet passed |
| Zig final | SKIPPED per updated user policy; baseline both passed |

Logs are ignored local evidence in artifacts/r9a2: baseline-isolated-go.log,
baseline-gates.log, final-go.log (exit 0), final-gates.log, production-native.log,
core-native-after-header.log, shadow-roundtrip.log, shadow-final-performance.log,
retired-focused.log, declaration-determinism.log, identity-malformed.log,
malformed-repaired.log, stale-native.log, generated-cli.json and explain-cli.json.
The first production native lane found a stale corruption probe assigning the new
valid magic byte; the probe now assigns the retired version and passes. No goldens
were rewritten to hide a failing regression. The new frozen binary oracles were
captured from the old codec before its retirement, during live shadow agreement.

## Single source, identity and coverage

Authority: libraries/Standard/Backend/BridgeSchema.concept, 12 records / 73 ordered
fields and four enums / 52 cases. Version is numeric 2 and magic CMIRAMD2.
SHA-256 of normalized checked metadata is:

```text
99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593
```

The generated fixed header is 44 bytes. Payload layout remains byte-identical to
CMIRAMD1. The inspectable metadata includes representation/count constants,
record fields, structural generic declaration/argument identity, fixed extents,
enum tags/bindings and typed conversions. Paths/spans/trivia/timestamps are excluded.
Artifact-only schema generation produces the identical hash and codec sources.
A checked fixture mutation adds a field to both generated sides, changes the hash,
and causes the native mutated Concept consumer to reject the old producer header
with BridgeSchemaMismatch. No source printer is reparsed to recover schema data.

## Agreement, roundtrip and determinism

Live manual-versus-derived shadow comparison covered core, finite, yield_resume,
multi_yield, pushdown, activation initialization and stride-12 addressing: 17
functions. Complete payloads agreed byte-for-byte. Both manual and derived Concept
backends emitted identical AMD64 bytes with 100 emissions per function. Frozen
cmir1/amd64 files retain this evidence without shipping the manual implementations.
Concept decode/re-encode preserves every byte, including facts/decisions/provenance
that the AMD64 semantic projection does not otherwise need.

Mandatory 100-run checks cover generated Go source, checked schema hash, generated
Concept derive source, materialized generated identity/text, C/MIR output, bridge
bytes, semantic decoded representation, complete Concept re-encoding, and native
AMD64 bytes. The actual reflected declaration/C/MIR lane completed in 9.43s in
the focused run. Source/metadata/generated-file agreement is a regression gate.
Field traces identify record/field, expected/actual byte offset and values.

## Malformed input and allocation

23 malformed specimens cover magic/version/hash mismatch, truncation at header
and payload boundaries, enum/bool/physical-register tags, negative/oversized
counts, impossible text lengths, invalid vreg/block references and trailing bytes.
Go rejects them. Native Concept returns the expected Result error, leaves output
untouched, and does not panic. A vreg bounds panic discovered during qualification
was fixed by establishing bounds before indexed width observation.

The generated Concept codec uses borrowed input views and caller output spans.
No heap, runtime reflection or serializer framework was added. Backend fixed
capacities and the R9a caller-supplied encoding workspace remain intact. The real
local codec call graph proves NoAllocation after removing imported effect grants
in the test; existing production EmitFunction/EmitFunctionWithWorkspace assertions
also pass. An ordinary proof defect was repaired: checked selected overload
signatures, re-derived rather than trusted from artifacts, replace name-only
expansion across every unrelated overload. Safe/allocating overload and artifact
regressions pin the authority boundary.

## Informational performance

One Windows sanity run, including verification for Go and whole backend emission
for Concept (100 emissions/function), with no threshold:

| Corpus | 1000 Go encodes manual / derived | Native pipeline manual / derived |
| --- | --- | --- |
| Core | 78.75 / 80.84 ms | 458.68 / 718.46 ms |
| Finite | 40.88 / 59.16 ms | 72.83 / 117.30 ms |
| Multi-yield | 33.83 / 33.40 ms | 65.11 / 104.52 ms |
| Activation init | 11.32 / 11.37 ms | 29.05 / 34.24 ms |
| Pushdown | 385.82 / 389.35 ms | 340.83 / 773.01 ms |
| Stride-12 | 10.32 / 9.22 ms | 28.82 / 32.62 ms |
| Yield | 29.71 / 31.47 ms | 63.12 / 98.43 ms |

These are informational pipeline timings, not isolated decoder benchmarks or a
universal performance promise. The Concept reader validates complete nested
records through bounded views and re-reads them for projection, explaining roughly
1-2.3x pipeline cost. No grotesque regression or allocation change was observed.

Bridge duplication is CLOSED. Stage-0 is ready for the explicitly separate R9b
research phase; no freeze action or R9b/EVT2 feature work was performed.
