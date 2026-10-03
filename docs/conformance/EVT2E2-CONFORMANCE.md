# EVT2e2 — CMIRAMD3 call contract qualification

Compiler: `concept-evt1-stage0-go`. Baseline:
`b7e8d69900718455947eb3e8515c6488370d9621`. This milestone qualifies direct
scalar call transport before outgoing physical ABI lowering.

## Contract and authority

The real path is checked source -> MIR/SemanticFacts -> existing Planner ->
verified LIR -> MachineIR CALL -> generated CMIRAMD3 transport -> generated
Concept wire views -> Concept call admission -> explicit later backend boundary.

| Part | Representation |
| --- | --- |
| Instruction | `Op=CALL`, exactly one `Calls` record; other instructions have zero |
| Callee | `Kind=direct`, exact checked declaration identity such as `Add\|Add(int, int)`; same-module declaration must exist uniquely |
| Arguments | Ordered `{Value: virtual-register ID, Type: exact scalar class}` records |
| Classes | `bool`, `i8/u8/i16/u16/i32/u32/i64/u64`; widths 1/2/4/8 bytes; address-role values excluded |
| Result | `void` with no destination and width zero, or scalar class with one width-matched non-address virtual destination |
| Convention | Checked enum `Win64`, semantic spelling `win64`; no physical placement |
| Provenance | Instruction source line/column and original LIR block/instruction index, decision and facts preserved |
| Effects | Opaque memory read/write, may trap, invalidates prior abstract flags; no physical clobber register set |

The declaration identity is supplied by semantic admission, never inferred from
printer text. The module call verifier matches ordered parameter classes and
result against the target's signature descriptors. `BridgeValidate` checks
every call, including calls in unselected functions, before normal projection.
It bounds-checks virtual IDs before width observation. Calls in selected native
functions return `UnsupportedCallLowering` before allocation or encoding.

`BridgeSchema.concept` owns the representation, enum domains, count limit and
version. Go codecs and Concept derive sites come from checked reflection;
`BridgeDerive`/`BridgeRead` materialize ordinary Concept readers and writers.
Stage-0 produces checked semantic transport and verifies producer consistency.
There is no new Go Win64 algorithm, allocator, frame planner or encoder.
The authority ledger and migration matrix record these shared responsibilities.

No outgoing argument registers, shadow space, return register assignment,
caller/callee save sets, spill slots, call frames, CALL instruction encoding,
unwind metadata or native call execution is qualified. Existing incoming ABI
and function return machinery remains the prior qualified implementation.
`Frame.HasCalls` is not used to authorize call lowering in this slice; CALL
records establish the transport boundary. External/indirect/varargs/tail,
floating-point/vector/aggregate/pointer-like source calls remain unsupported.

## Typed Verdict use

`PinCallContractProof` proves NoAllocation for validation and roundtrip using
the checked call graph, and PlainData for the two call wire views through
`PlainDataHolds`, whose return is
`Verdict<PlainDataEvidence, PlainDataRefutation>`. Qualification checks retained
typed evidence: call view size 28/alignment 4, argument view size 8/alignment 4.
These are internal view geometries, not encoded byte lengths or C ABI grants.
Runtime admission and byte IO use `Result<..., BackendError>`: malformed bytes
are computation errors. No new proof evaluator or boolean proof wrapper exists.

## Version and generated outputs

| Identity | Numeric version | SHA-256 schema hash |
| --- | --- | --- |
| Previous CMIRAMD2 | 2 | `99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593` |
| Qualified CMIRAMD3 | 3 | `3ebaeb0e5b79bea00d320991521b86015974cf994a3b1e1f498c944999dff312` |

The header remains 44 bytes. CMIRAMD2, stale hashes, invalid enum tags and
truncation reject deterministically; there is no silent upgrade or legacy
runtime decoder. Existing enum tags are retained and CALL is appended. There
are 14 wire records/81 fields and seven wire enums/65 cases. Generated Concept
declarations increase from 31 to 41 (+10): four call-record read/write functions
and six enum functions. Generated outputs are checked against regeneration;
100 regenerations are identical. Ordinary `concept generated` exposes the new
records through generated declaration provenance.

## Bridge and verifier evidence

`internal/concept/testdata/evt2e_calls.concept` includes zero-arg, bool, void,
scalar-result, five-arg and mixed-class calls, and a value live across a call.
No native call behavior is claimed from these fixtures.

`TestEVT2e2CallTransportAndMalformed` verifies 100 identical bridge encodings
and canonical decode/printer results. `TestEVT2e2ConceptArtifactOnlyCallRoundTrip`
compiles the real Concept validator and generated codecs to strict C11 and
executes both Normal and Verify policy outputs. Its runtime input is only
artifact bytes and caller output. 100 valid full-header roundtrips are exact;
each malformed artifact is attempted twice and leaves sentinel output untouched.

The 23 invalid cases are: missing callee, bare identity, matched malformed
identity, unavailable target, wrong argument count, reordered mixed-class
metadata, mismatched argument class, void argument, void with result, missing
scalar destination, mismatched result class, oversized virtual ID, negative ID,
duplicate contract, missing source provenance, wrong LIR provenance, unknown
convention tag, unsupported indirect kind tag, invalid result enum, invalid
argument enum, truncated call record, CMIRAMD2 identity, stale schema hash.
Both Go and Concept reject all 23. Go semantic models reject before encoding;
tests use the generated writer directly only to construct malformed wire inputs.
Go diagnostics name `MIR_CALL_*`, `MIR_BAD_CALL_SHAPE` or bridge identity/tag
errors. Concept returns BridgeInvalid, BridgeVersion or BridgeSchemaMismatch.

The exact minimal call record is 38 bytes, independently pinned as:

```text
000000000a00000049647c496428696e74290000000001000000070000000600000006000000
```

It represents direct `Id|Id(int)`, Win64, argument v7/i32 and result i32.
Target, argument ID/class/order, result and convention-byte changes alter the
artifact digest. A changed unknown convention is rejected rather than admitted
as a second convention. Provenance remains in the enclosing instruction.

## CLI boundary

`TestEVT2e2CLIContractAndBoundary` executes ordinary CLI subprocesses:

- `concept lir`: succeeds and prints checked direct calls.
- `concept machineir`: succeeds and prints target, ordered values/classes,
  result, convention and source/LIR provenance.
- `concept amd64`: emits the existing no-call FortyTwo function, then stops at
  ZeroArgCall with `AMD64_UNSUPPORTED_CALL_LOWERING: call ABI lowering is not implemented (tag=8)`.
- `concept generated`: exposes WireMachineCall declarations with built semantic
  dependency artifacts through `CONCEPT_MODULE_ROOTS`.

This removes the old `MIR_UNSUPPORTED_LIR_OP call` blocker. Win64 physical
lowering is the intentionally deferred next milestone, not a failing contract.

## Size and timing

The call fixture artifact is 14,585 bytes. Every non-call instruction gains a
four-byte empty call-sequence count. The minimal call record adds 38 bytes plus
its containing sequence count; larger targets/argument lists vary by encoded
text and eight bytes per argument. Frozen no-call examples:

| Shape | CMIRAMD2 equivalent | CMIRAMD3 | Increase |
| --- | ---: | ---: | ---: |
| core | 16,726 | 17,042 | 316 |
| finite | 8,237 | 8,429 | 192 |
| pushdown | 73,783 | 75,667 | 1,884 |
| parent-resume | 2,284 | 2,336 | 52 |
| stride12 | 1,835 | 1,875 | 40 |

The seven frozen payloads are preserved after removing only traced empty Calls
counts; the frozen files are unchanged. All corresponding native bytes remain
exactly equal. Timing is informational, measured under concurrent validation:
1,000 full Go fixture encodes 73.042 ms and decodes 72.034 ms; 10,000 minimal
record encodes 1.522 ms; Concept 100 validated full roundtrips 190 ms Normal and
191 ms Verify in a focused run. These are observations, not performance gates
or native-call benchmarks. Concept admission deliberately scans bounded views
without an allocated index; future throughput work must preserve admission.

## Validation

Baseline qualification uses a separate clean detached worktree at the exact
baseline SHA. Both baseline and final lanes include:

- `go test ./... -count=1`, including semantic corpus and GPU-free Vulkan,
  EVT2/EVT2x native execution and existing 100-run native determinism.
- `go vet ./...` and focused race checks for EVT2e, frozen payload parity and
  R9c condition authority agreement.
- Corpus: 436 valid, 357 static-invalid, 15 runtime-negative, five compatibility
  and four expected-divergence fixtures; manifest counts unchanged.
- Standard 42/42, DragonGod 23/23 and Golden 130/130 in Normal and Verify,
  with zero failures.
- Focused EVT2e2 and frozen bridge/native malformed/schema regression tests.
- Canonical format checks and lint for all touched authored Concept sources;
  lint uses the repository libraries as module roots. `git diff --check` passes.

Root and legacy Zig suites are skipped: the frozen Zig compiler and its build
infrastructure are unchanged. The working change is committed and clean at
handoff; the final SHA and command results are reported with the handoff.

Full-suite observations: baseline internal/concept 384.161 s, CLI 2.909 s,
GPU-free Vulkan 0.313 s; final internal/concept 414.486 s, CLI 5.076 s,
GPU-free Vulkan 0.358 s. Concurrent library/focused runs make these unsuitable
as a compiler performance comparison. Added typed-evidence assertions and the
fresh-source 100-run MachineIR generation check also pass in subsequent focused
runs; those changes affect tests only. The temporary baseline worktree is
archived after qualification.

SUCCESS — EVT2e2 CMIRAMD3 call contract qualified.
