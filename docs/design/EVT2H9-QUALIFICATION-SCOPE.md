# EVT2h9 frozen qualification scope

Inventory before the complete run. Baseline `2c93386f02d218c667ef2b78c026d9ed25f5ae9b`,
clean existing `codex/implement-native-ordinary-enums` worktree at
`C:/Users/yuech/.codex/worktrees/evt2h4-profile-baseline/Concept`.
Fetched origin/main `065317d262c85ce159219561b0709526e7973c1f` is an ancestor
20 commits behind; no intervening work, reset or remote mutation.
Compiler `concept-evt1-stage0-go`, Windows AMD64/Microsoft x64.

This freezes the bounded h1–h8 behavior to qualify, not a new feature phase.
Historical h8 and earlier records remain unchanged. Final status requires actual
full gates, independent execution and 100 fresh trials; the initial status is
**qualification pending**.

| Stage | Checked source and LIR | Transport / admission | Native / ABI / independent evidence | Excluded boundary |
| --- | --- | --- | --- | --- |
| h1 ordinary enums | Nominal variants, checked tags, construction, local copies/equality, scalar calls/returns | v7; aliases in v9/v10; Concept nominal domain admission | Checked discriminant scalar realization and Win64 scalar calls; enums fixture, C11 and byte pins | Payloads are not ordinary integer enums; public enum-reference/array fences remain |
| h2 tagged enums | Closed copyable payload layouts, active initialization/copy/projection, field order | v8, composed into v9/v10; independent active-field/geometry/copy admission | Existing object slots and recursive meaningful-field copier; integer/bool/float/double, records and fixed-array fields; direct small or indirect/hidden-return Win64; native/C11/MSVC/clang-cl probes | Nested sums, owning/move-only/Drop and unqualified reference payloads |
| h3 match | Exhaustive checked patterns, scoped bindings, ordered guards/subjectless guards where accepted, result joins, nested branches/loops/returns | v9; v10 failure-region composition; CFG/active-variant/scope/init admission | Ordinary comparisons/branches, active payload reads, existing scalar/aggregate joins; source/C11/native corpus and frozen bytes | No new pattern syntax, jump tables or ownership-aware arm cleanup |
| h4 Option | Canonical closed Some/None applications, replacement/copy, checked Some binding and match | v9/v10, no Option-specific transport or copier | Qualified scalar/record/fixed-array payloads, exact floating bits; existing aggregate ABI, independent C11/C compiler probes | Nested Option/sums, niches, general foreign representation |
| h5 Result and ? | Canonical Ok/Error, exact E, void success, must-use/discard, function propagation and admitted remaps | v9/v10; checked destination and active payload authority | Same tagged/Win64 machinery; success continues, failure returns; Result fixtures, native/C11/ABI including recursion/spills | Arbitrary error conversion and owned early-exit cleanup |
| h6 Verdict/assertions | Go comptime evaluator, typed selected evidence/refutation/Unknown and single proof projection; checked assertion erases before runtime LIR | Semantic artifacts retain descriptive proof metadata; no runtime Verdict transport | Residual native image equals freshly checked assertion-free program; source/proof/artifact and C11/native erasure probes | Runtime Verdict/ABI and fabricated trusted facts are rejected; evaluator has not migrated to Concept |
| h7 ! and local handlers | Operand once, success access or trap; nearest exact Result handler, scope/init and lexical failure destinations | v10 adds checked try/handler/site associations to frozen v9 | Existing branch/copy/trap facilities, void success, nested handlers, Option function propagation; isolated real UD2/status checks, native/C11/ABI | No exception runtime/unwinding; v10 function remap arm remains straight-line |
| h8 integration | Real word parser/controller, records containing ordinary enums, short-circuit and/or/while and earlier-operand snapshots | v10; existing independent validators | Source-free artifacts, no domain C in native-only host; separate strict-C11 domains; entry carrier liveness; pinned images and proof erasure | Raw local arrays, local address temporaries across actual failure exits, nested sums/ownership |

## Major-checkpoint execution plan

Run the actual `./test.ps1 -Lane Qualification -ProfileDirectory ...` entrypoint
with uncached Go results, phase100 marker and all existing native/C11/ABI/byte,
corpus/Golden/Vulkan/package/generated/policy tests. Run documented supplemental
focused race, Standard and DragonGod Normal/Verify, formatting/lint, build and
worktree checks. Keep the unchanged frozen Zig policy; do not enable unrelated
collector research burn-in. Explain subprocess-only worker skips separately.

The audit found that h1–h7 native loops re-emit one canonical artifact while
separate tests freshly check source100 times. Strengthen only the phase harness:
each emission will read one of100 separately generated artifacts, require exact
canonical agreement and independently roundtrip it through the Concept codec
before ordinary EmitModule admission/planning/emission. Existing loops,
observables, sentinels, ABI probes and byte expectations remain intact. h8
already has complete fresh-file pipelines. No production codec or backend change.

Name the original parser expected9 regression and retain the exact 16 vectors,
40 controller combinations, proof counterfactual, isolated trap executables and
unused-scalar probe. Existing g5 and sum ABI families cover direct/indirect,
hidden result shifts, positions beyond4, bool/integer/floating mixtures, caller
live state, recursion/spills, callee preservation, evaluation order and isolation.
The native caller's16-byte indirect temporary alignment versus compiler-observed
weaker incoming residues remains an explicit observation, not a resolved general
interop contract. The native subset rejects foreign/variadic/indirect dispatch,
escaping references and owning cleanup outside its qualified contracts.

Profile only test orchestration/build/process boundaries. Preserve independent
oracles and all100 trials; no fresh checked result/plan/artifact/output caching.
Stop after qualification and an ownership/Drop handoff, without starting it.
