# EVT2h8 integration inventory and bounded design

Starting point: clean `b39088beb9631ea4a449c65797bbc76fde9a265a` on the existing
`codex/implement-native-ordinary-enums` branch, Windows AMD64/Microsoft x64,
`concept-evt1-stage0-go`. Fetched origin/main `065317d` is an ancestor 13 commits
behind. No intervening commits, reset, new branch or remote mutation.

This inventory precedes implementation. Full phase qualification remains h9.

## Capability matrix at the baseline

| Composition | Source / checked production | Transport / Cathedral | Existing evidence / remaining gap |
| --- | --- | --- | --- |
| Ordinary state enums, equality, calls | Existing checker and LIR preserve nominal enum identity | Frozen v7 or v9/v10 enum aliases; Concept admission/Win64 realization | h1/h3 fixtures; supported |
| Payload tokens/events with scalar or record fields, exhaustive match | Existing tagged layout, scoped bindings and explicit CFG | v8 constructors/copy; v9 matching; v10 failure interactions | TaggedValues/TaggedMatches/FallibleControl; supported bounded copyable domain |
| Option<int> lookup followed by match or bang | Closed application identity and checked active Some access | Existing tagged machinery, v9/v10 | Options/FallibleControl; supported |
| Result<Record, nominal ordinary enum or copyable error record>, ?, calls | Source exact E matching; LIR checked success/error CFG | v9 function propagation, v10 local handlers/bang | Results/FallibleControl, record/floating errors and Win64 aggregate probes |
| Match arm containing try/except, typed handler returning a record | Existing lexical scopes and active payload authority | v10 scope-aware region exits and existing copier/frame | FallibleControl MatchTry/ItemLocal; compose and execute in a realistic pipeline |
| Checked Assert.Concept with runtime enum/Result code | Go checker/proof graph owns the proposition; checked intrinsic erases in LIR | Residual ordinary runtime transport, no Verdict carrier | h6 residual leaves/source-free semantic consumers; integration evidence needed |
| Short-circuit and/or with calls/failure inside an operand | Source-valid bool operators; C lowerExpr already emits conditional RHS preludes | Existing bool slots/branches suffice; no physical opcode or wire field needed | Native LIR BinaryExpr lacks and/or dispatch and rejects EVT2_UNSUPPORTED_BINARY; category A projection gap |
| Local fixed arrays | Source/C valid; explicit VarDecl currently rejects in native LIR | Qualified arrays inside records and tagged payloads already have geometry/copy | Category A projection boundary, not needed: use a genuine fixed-capacity input record |
| Local address temporary preceding a fallible argument | Source/C valid; saveBeforePropagation explicitly rejects nonparameter addresses | Checked reference lifetime/location authority must survive the CFG | Category D/E lifetime work; retain diagnostic, use qualified scalar ref parameters |
| Conditional/fallible remapping inside a v10 Error arm | Source/C valid; v10 production requires a straight-line function-failure arm | Frozen v10 admission requires that arm's return/transfer correspondence | Category B/C checked region/admission extension, nonessential here; preserve explicit unsupported diagnostic and old v9 behavior |
| Result<Token, payload ParseError>, Option<Result<...>>, other nested sums | Source/C can represent closed applications; lir_tagged rejects nested tagged fields | Recursive sum layout/copy/authority qualification is intentionally absent | Category E separate architecture/qualification milestone; do not add a second layout/copy engine |
| Reference/owned/Drop-bearing payloads | Source may permit checked authority/move semantics | Native payload eligibility rejects unqualified reference/ownership/cleanup | Category D; no cleanup omission, erased authority or new ownership semantics |

Owners inspected: parse.go/validate.go/failure.go/generate.go; lir_lower.go,
lir_match.go/lir_result.go/lir_fallible*.go/lir_tagged.go; generated CMIRAMD8–10;
BridgeValidate8–10; AMD64.Objects/Aggregates and Win64AggregateABI. Source enum,
match, failure and typed-verdict fixtures remain the language contract.

## Programs and scope decision

1. **Bounded word-protocol parser/evaluator.** A fixed-capacity input record
   models an actual wire buffer. Checked decoding returns a record plus a typed
   fault record through Result, propagating through several functions. A separate
   materialization step produces a payload Token enum. Optional fields use Option;
   evaluation matches tokens; a supervisor catches typed faults and materializes
   a payload-bearing nominal ParseError for structured diagnostics. This is the
   supported equivalent of Result<Token,ParseError>: wire decoding, typed command
   materialization and diagnostic rendering remain useful separate operations.
   No nested-sum layout is smuggled into a record or reinterpreted integer blob.
2. **Typed controller.** Ordinary state, payload events, copied mutable session
   values, optional hints, configuration propagation and locally handled device
   failures. State and event matching dispatch ordinary functions; no machine/
   async redesign. Distinct same-representation error enums must stay distinct.
3. **Checked semantic assertion plus runtime fallibility.** Typed evidence and
   provenance remain available through the existing proof tooling; the assertion
   erases. Compare its residual artifact/code/behavior with the same checked AST
   after removing only the assertion. False/Unknown and forged authority reject.

Implement and/or as ordinary LIR bool-slot/branch/join projection: evaluate and
snapshot the left bool once; branch to the RHS only when needed; update the
result on RHS success; return/trap/handle RHS failure through existing control.
Nested expressions, loop/if/guard contexts and skipped failing RHS receive direct
native/C11 evidence. No source, bridge, allocator, frame, copier or ABI change is
planned. A verifier deficiency, if discovered, must be isolated and escalated.

## Evidence plan

Authoritative .concept examples plus grouped .concept_test facts; external Go/C
hosts read those same files for diagnostics/IR/artifacts/native execution/oracles.
Batch the three source-free artifacts in one host per mode with one immutable
Cathedral build per mode. Oracle code is linked only into the differential host;
the native-only host executes without domain C. Use distinct completed paths for
expected trap children. Repeated interleaved vectors expose stale active fields,
stack storage and cross-call contamination.

Use three genuinely fresh source/LIR/artifact/plan/emission trials for this
composition milestone; the existing phase marker promotes new gates to 100.
Compare debug plans, fixups, bytes, roundtrips and Normal/Verify, and independently
inspect proof payload/site/provenance/erasure. Preserve all frozen v3–10 files and
existing 100-run gates. Run Quick, focused native/malformed/adjacent checks and
appropriate race/vet/build/format/lint. Measure added facts and native test cost.
Do not automatically run the complete project lane, and do not begin h9.
