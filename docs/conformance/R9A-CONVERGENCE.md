# R9a convergence and issue ledger

This is a progressing closure record, not a freeze declaration. Final commits and
qualification are appended after closeout. R9b and frontend self-hosting are not begun.

| Item | Status / semantics | Evidence or remaining work |
| --- | --- | --- |
| A1 | Structured identity; required artifact sub-schema | R9a artifact/spelling/native/100-build tests |
| A2 | Bounded extraction | generic_identity, validation_context, attribute_placement; flag boundaries remain |
| A3 | Scoped admission audit repaired | Ledger below; 13 diagnostic agreement cases |
| A4 | Validator revalidation modes repaired | Context fixtures; explicit runtime push/comptime bounds retained |
| A5 | Existing diagnostic order retained | No promise migrated rules fire first |
| B1 | Explicit ownership retained | Generic Drop/CV4653 qualification |
| B2 | Named initializers already work; other friction pending | Unsigned indices/literals, complement, generic rounding, defaults need closure |
| B3 | Touched diagnostics name fixes | Qualifiers, scoped, attributes, stale artifacts, repr |
| B4 | Generic repr(C) records admitted after closure | Artifact/native Pair<int>; resource pair rejects |
| B5 | Pending | Parser lacks runtime comptime if/for; deferred open AST and closed expansion needed |
| C1 | CV4138 field family migrated | Corpus shadow agreement, authoritative switch, Go deletion; array/generic/artifact tests |
| C2 | Existing type/struct-field subjects retained | Enum payload and function observations pending |
| C3 | Pending | PREDICATE_REQUIREMENT_SCOPE; proof evaluator lacks predicate arm |
| C4 | Intentionally deferred | Fact-granting trust model requires later research |
| C5 | Intentionally deferred | Pattern guard binding/fallthrough unnecessary here |
| C6 | Intentionally deferred | Verdict<H,R> requires generic comptime support |
| C7 | Retained | Shape and boolean observations; no speculative redesign |
| C8 | Measurement pending | Large-library fuel/depth/mutex study follows |
| D1 | Pending | CMIRAMD1 manual producer/consumer; current differential/round-trip tests retained |
| D2 | Handoff capacity mismatch already fixed in EVT2x baseline | Liveness has 128 blocks; caller-supplied arena path missing |
| D3 | Existing differential tests retained | EVT2d backend/bridge and EVT2x native/C oracle |
| D4 | Intentionally deferred | Imported bytes reflection requires dependency identity work |
| D5 | Not frozen | Freeze-readiness report names remaining work |
| E | Vulkan regressions only | Broader ladder deferred; GPU optional |
| F | Outside repository/task | No Oct/Prometheus work |
| G | Baseline tooling | Handoff noise did not reproduce; pre-existing amd64.go formatting untouched |

## Accepted-but-inert audit ledger

All retained rows have semantic, metadata, diagnostic or generated behavior. User
selector metadata is deliberately observable; compiler attributes have strict positions.

| Spelling/surface | Semantic consumption | Artifact/codegen effect | Decision/diagnostic effect |
| --- | --- | --- | --- |
| owned/borrow/ref/const/scoped | Ownership/provenance/escape/place checks | Typed qualifiers, Drop/ref/readonly projection | Retain; conflicts/duplicates reject; scoped early-return hole fixed |
| unsafe/imported type modifiers | No qualified active semantics | Formerly flags only | Reject TYPE_QUALIFIER_UNSUPPORTED |
| repr(C) | Closed C value/layout legality | Repr policy; native evidence where required | Retain; generic records admitted; bad args/positions reject |
| extern C | Signature/domain/ownership | ABI linkage/foreign contracts | Retain; destruction rejection remains live |
| reflect | Imported structural permission | Reflection permission/results | Retain; shared validation rejects malformed/unknown attrs |
| user field/case selectors | Reflection/generator selection | Attributes/generated provenance | Retain observable metadata |
| must_use | Result discard checks | Declaration metadata/origin | Retain supported type/function positions |
| machine/unsafe asm | Signature/profile/target legality | Machine/native/C lowering | Retain; unknown intrinsic rejects |
| async/machine control | Persistence/state/transition legality | Typed MIR and explicit unsupported-native boundaries | Retain |
| fact/theory/benchmark/prophecy/artifact/foretold | Test validation/discovery | Test manifests/runner | Retain; invalid combinations/signatures reject |
| verify_foreign | Contract verification test metadata | Runner declaration/Verify instrumentation | Retain; test kind and args checked |
| execution_context/semantic_access/synchronization | Access/proof derivation | Summaries/planner evidence | Retain admitted function positions |
| diagnostic | Innate diagnostic identity | Innate metadata | Retain innate only; ordinary concept rejects |
| generic requires | Required-operation/constraint closure | Structural concepts/witness metadata | Retain; unsupported predicates explicitly reject |

Audit scope is the current Stage-0 surfaces above, not future features or retired Zig.
