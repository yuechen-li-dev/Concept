# R9c conformance

Baseline: `3cb83bc4e4d9ac96fc86b151380a2c129cc60ca8`, clean checkout.
Compiler ID: `concept-evt1-stage0-go`. R9c establishes ownership and a small
MachineIR verifier authority switch; it does not implement EVT2e calls or
freeze Stage-0.

## Executable migration proof

The existing Go `machineConditionValid` switch duplicated the accepted
Condition names in Concept `Standard.Backend.BridgeSchema`. Its production
verifier call now resolves the name against generated `bridgeTagsCondition`,
rejecting the None tag. That array is emitted from checked Concept reflection
by the existing bridge generator; it is also the live wire codec's tag table.
The old Go list is retained only as a frozen test oracle in
`TestR9cConditionAuthorityShadowAgreement`, where every accepted name and
representative invalid names agree. The normal-path verifier chooses the
Concept-authored vocabulary. The manually maintained Go switch is RETIRED;
the generated Go codec remains BOOTSTRAP-ONLY. No CMIRAMD2 bytes, hash or
schema fields change. This is the smallest real authority switch, not a
claim that all MachineIR verification is Concept-owned.

The direct Stage-0 `concept amd64` CLI previously parsed the imported backend
module as a standalone source, failing CV4401 before C generation. Its host
now resolves and builds the ordinary source imports from the checkout library
root. `TestAMD64BootstrapLoadsBackendImports` executes the real CLI on the
finite machine specimen and requires emitted native bytes. A direct
`parent_child.concept` CLI probe also emits both activation functions. This
fixes bootstrap orchestration; call/ABI policy is unchanged.

## Contract findings

MachineIR has virtual/physical operands, flags, slots, frame local size,
alignment, shadow-space and HasCalls metadata, plus incoming argument
descriptors. It has no CALL opcode, per-call argument/return locations,
clobber set, callee-save/spill plan, or call-kill FLAGS semantics. The Concept
frame finalizer rejects HasCalls. LIR has no call/helper-call instruction or
call-result/aggregate ABI contract. CMIRAMD2 is stable for the present
no-call corpus and must be versioned for EVT2e rather than silently extended.

## Validation record

The focused test `go test ./internal/concept -run
'^(TestR9cConditionAuthorityShadowAgreement|TestEVT2cVerifierRejectsMalformed|TestR9a2BridgeGeneratedOutputs)$'
-count=1` passed. Full `go test ./... -count=1` passed, including the semantic
corpus, GPU-free Vulkan, EVT2/EVT2x native and bridge byte-oracle tests;
The final rerun after the CLI repair passed, with `internal/concept` at
377.557s and the new `cmd/concept` integration test at 1.988s. `go vet ./...`
passed. Focused `-race`
for the R9c helper, malformed verifier, generated bridge, EVT2c 100-run
determinism and frozen native byte oracle passed in 10.746s; the new CLI test
passed separately under `-race` in 3.051s. The audited
AMD64 Concept source passed `format --check` and `lint` (with
`CONCEPT_MODULE_ROOTS=libraries`). The unchanged BridgeSchema source is not
currently canonical under `format --check`; its failure is a pre-existing
formatting issue, not an R9c source change. Serial package tests passed:
Standard 42, DragonGod 23 and Golden 130, each in both Normal and Verify,
with zero failures. The full Go suite includes `TestSemanticCorpusManifest`,
GPU-free Vulkan native library/profile tests and EVT2/EVT2x qualification.

The helper performs at most 13 generated-table comparisons for each FLAGS
condition check; runtime emitted code and CMIRAMD2 are unchanged. This is a
bounded verifier cost, not a measured throughput claim. The CLI now builds
backend imports per invocation before C generation; that bootstrap-only cost
was not separately benchmarked. Correctness, deterministic bytes and
ownership were the R9c acceptance metrics.
Frozen Zig is unchanged, so its two suites are skipped under the current
policy. No full Cathedral executable, object output, native Linux/macOS run,
or C5 self-rebuild is claimed.
