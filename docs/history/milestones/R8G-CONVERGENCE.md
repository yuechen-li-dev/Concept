# R8g natural-authoring convergence

Baseline `baa6f34e657b6ef8c73308fba70beadf14a5715e`; compiler `concept-evt1-stage0-go`. The follow-up commits are `1696935` (operators and Math), `1ae3cc4` (integer match), `ee0fae1` (comptime), `53ce525` (bounded exhaustion), `a797f74` (assertions), `9453bb0` (formatter), `08e1b1d` (F10), and `0799ade` (original behavior). EVT2 was not started.

## Final issue ledger

| Finding | Previous state | Final state and source simplification | Commit | Tests / deferred reason |
| --- | --- | --- | --- | --- |
| B1 arrays | Nested and returned literals emitted wrong element C types. | Closed: target type reaches aggregate, enum, argument, and generic contexts; Claude Postfix uses `TokenKind`. | `cb82976` | `TestR8gContextualArrayLiteralLowering`, strict C11/native. |
| B2 quantities | Implicit `float<unit>` to `float` erasure. | Closed: implicit erasure rejects; `Magnitude`, `interpret`, same-dimension `as` remain explicit; Math preserves units. | `cb82976`, `1696935` | `TestR8gQuantityCannotImplicitlyChangeMeaning`, typed Math artifact test. |
| F1 floats | Context reached only direct initialization. | Closed: calls, operations, aggregates, match and Result arms type literals; redundant casts removed. | `cb82976` | `TestR8gContextualFloatLiterals`. |
| F2 operators | Open `T` rejected operators. | Closed: operators are required operations with ordinary witnesses; primitive, quantity, enum, and user closure passes artifact-only import. Typed generic Math Abs/Min/Max uses this model. | `1696935` | `TestR8gRequiredOperatorsCloseThroughArtifact`, diagnostics, typed Math, 100-run output. |
| F3 Result remap | Borrowed Result could not change error type. | Closed: `? else` maps error only and preserves success provenance; original CAD returns declared missing errors. | `cb82976`, `0799ade` | `TestR8gBorrowedResultErrorRemap`, CAD runtime fact. |
| F4 enum geometry | Nullary enum had no fixed Span/array geometry. | Closed: current qualified C target uses uint32 tag, size/alignment 4; payload enums distinct. | `cb82976`, `978c0cd` | `TestR8gPayloadFreeEnumGeometry`, checked outputs. |
| F5 integer match | Diagnostics recommended unsupported integer match. | Closed: contextual integer/bool literals and wildcard; open integer domains require wildcard where completeness matters. | `1ae3cc4` | `TestR8gIntegerMatchContextAndExecution`, diagnostic test, C11/native. Range patterns deferred outside R8g. |
| F6 comptime | Unsigned, named generic args, and contextual mask operations failed. | Closed for bounded integer values: typed constants canonicalize with equivalent literal closure and inline into C. Storage Frames and Embedded PingReply replace repeated invariants. | `ee0fae1` | `TestR8gNamedComptimeGenericArtifactAndContext`, artifact-only C11/native. Generic values above host `int` range remain unsupported. |
| F7 exhaustion | Bounded loop silently truncated unfinished async work. | Closed: `while ... bounded(N) ... else` runs else on still-true continuation at N; Normal without else retains truncation; Verify traps. Both Journals return Stalled before Result. | `53ce525`, `0799ade` | `TestR8gBoundedExhaustionBoundary`, Verify trap, async test, Golden Normal/Verify. |
| F8 assertions | Equals literal, double/quantity Near, exact Result error awkward. | Closed: contextual Equals both orders; typed Near; FailsWith checks exact error. Claude per-type helpers and unit-erasing test locals removed. | `a797f74` | `TestR8gAssertionContextDoubleQuantityAndExactError`, negatives, Claude suite. |
| F9 formatter | Postfix, aggregate and simple block forms expanded poorly. | Closed: compact forms survive; all 61 final Golden source/test files pass per-file format check. First drafts untouched. | `9453bb0` | `TestR8gFormatterKeepsCompactNaturalForms`. |
| F10 subdirectory | Nested test selection broke sibling imports. | Closed: selected directory is a filter, with ancestor module roots in the checkout. | `08e1b1d` | `TestR8gNestedTestSelectionRetainsSiblingModuleRoots`; Claude Storage 7/0. |
| F10 Subspan | Temporary Span expression required a named local. | Closed for known backing/lifetime; Claude Game directly uses `Subspan(ReadOnlySpan(frame.x), ...)`. Local-backed return still rejects. | `08e1b1d` | `TestR8gSubspanTemporaryPreservesBackingLifetime`, C11/native, CV4521 negative. |
| F10 Standard.Math | Safe quantity rejection lacked typed alternative. | Closed via required operator generic Abs/Min/Max for float/double/half and same-unit quantities. | `1696935` | Typed Math artifact/native test. |
| F10 atomics | Only AtomicInt available. | Deferred: HFT now keeps AtomicInt in `[0,128)` and passes wrap/order; no Golden needs unsafe unsigned atomic representation. | `0799ade` | HFT facts; wider atomics need target ABI and real use case. |
| F10 live count | Store consumers scanned occupancy. | Closed: `GenerationalCount` reads O(1) live counter. Retired slots make `next - freeCount` wrong. Original Storage uses it. | `08e1b1d`, `0799ade` | Store artifact-only C11/native test covers empty/insert/remove/reuse. |
| F10 generator attribute | OperandCount/SuccessorCount repeat shape. | Deferred: typed attribute identity cannot be a parameter yet; string selector would weaken schema authority. | — | Both small generators pass; post-R8 typed reflection meta-object proposal needed. |
| F10 reflection field UX | `instruction.field.value` resembles literal `field` member. | Clarified in both IRTools sources that `field` is the FieldInfo loop binding and projects the selected member. | final docs batch | Generator facts pass; no semantic ambiguity found. |

## Original Golden behavior and authoring

HFT sequence counters wrap without signed overflow; Aerospace advances one phase per sample and overwrites oldest history after 16; Game seek moves on both axes and typed `Id<Agent>` strikes resolve in two phases; CAD maps invalid IDs to errors; Journal reports insufficient budgets as Stalled; Storage exposes WriteBack and public live count. New facts pin each path. Original and Claude remain separate authored passes; first drafts are preserved. Combined Golden Normal and Verify each report 130 passed, 0 failed, 2 benchmarks.

The original source now uses the language fixes directly where relevant: CAD `? else`, Journal bounded `else`, HFT/Storage named invariants, and Storage's count API. Claude tests use built-in exact-error assertions and direct temporary Subspan. Generated C keeps named comptime values inline and nullary enums as fixed tags. Bounded while emits continuation checks; no new operator runtime registry was introduced. A representative integer-match/comptime/bounded source was stable over 100 CLI runs for C, MIR, lint, explain, and format check; operator/artifact output has a separate 100-run test. This is not a 100-run claim for every CLI family.

## Previous partial ledger (historical)

The first implementation batch is `cb82976c9547c883aaf4cac6f192a7d2f1bcbda8`; compatibility migration and checked-output refresh are `978c0cd`. Status refers to the whole requested finding, so a safely rejected operation is not marked complete when its typed replacement is still absent. `CLAUDE-GOLDENS.md` remains the source issue ledger and its first drafts remain unchanged.

| Finding | Status | Commit | Final semantics or next blocker | Evidence |
| --- | --- | --- | --- | --- |
| B1 array literals | Landed | `cb82976` | Target element type reaches nested and returned aggregate, enum payload, function argument, and closed generic C lowering. | `TestR8gContextualArrayLiteralLowering` |
| B2 quantity erasure | Partial | `cb82976` | Implicit quantity/scalar mixing rejects. `Magnitude`, `interpret`, and same-dimension `as` remain explicit. Typed `Standard.Math` quantity operations need operator-generic closure. | `TestR8gQuantityCannotImplicitlyChangeMeaning` |
| F1 float literals | Landed | `cb82976` | Expected floating type reaches calls, arithmetic, comparison, unary minus, aggregate/array/tensor literals, and match arms; runtime float conversions remain explicit. | `TestR8gContextualFloatLiterals` |
| F2 open generic operators | Open | — | Parser names required operations as identifiers; template validation rejects dependent operators with `CV4175`. A unified operator declaration, witness, and artifact closure is required. | `parseConceptRequirement`, `validateExpr` binary path |
| F3 error remapping | Landed | `cb82976` | `expr ? else error` remaps only the error branch and preserves successful borrow provenance. | `TestR8gBorrowedResultErrorRemap` |
| F4 payload-free enum geometry | Landed | `cb82976` | Stable `uint32_t` tag in C; `SizeOf`/`AlignOf` 4; payload enums stay distinct. | `TestR8gPayloadFreeEnumGeometry` |
| F5 integer match | Open | — | Integer literal arms and the contradictory `else if` diagnostic remain. | Claude report F5 |
| F6 comptime constants | Open | — | Unsigned declarations, generic value arguments, and contextual operations remain. | Claude report F6 |
| F7 bounded exhaustion | Open | — | Explicit exhaustion behavior and async budget semantics remain. | Claude report F7 |
| F8 test surface | Partial | `cb82976` | Match expression Result constructors receive expected type. Assert.Equals literal context, typed Near, and specific Result error assertion remain. | `TestR8gContextualFloatLiterals` |
| F9 formatter | Open | — | Postfix, compact literals, and compact blocks remain; 24 of 31 final goldens fail per-file format check. Golden root has no manifest for whole-tree check. | Claude report F9; per-file CLI check |
| F10 subdirectory test targeting | Open | — | Package-root module resolution remains. | Claude report F10 |
| F10 Subspan temporary | Open | — | Named span workaround remains. | Claude report F10 |
| F10 Standard.Math | Partial | `cb82976` | Silent erasure rejects; quantity-preserving Abs/Min/Max still require generic operators. | B2 imported Math test |
| F10 atomics | Open | — | Only existing atomic surface remains. | Claude report F10 |
| F10 GenerationalStore live count | Open | — | No live count query added. | Claude report F10 |
| F10 generator attribute parameter | Open | — | Attribute name parameter remains unsupported. | Claude report F10 |
| F10 reflection field access | Open | — | `instruction.field.value` remains ambiguous. | Claude report F10 |

## Authoring comparison

Claude Postfix previously encoded token kinds as three integer constants and an `int code` field because a Span element could not contain even a nullary enum. It now uses `TokenKind` directly. The three code constants are gone; parser behavior and all six Claude Postfix tests pass. The payload-bearing `Intent` in Claude Game still blocks `Span<Agent>`, so its index loop was retained and its comment made precise. The original Golden behavior bugs listed in `CLAUDE-GOLDENS.md` were not yet ported from the Claude corrections.

The next ordered obstacle is F2. There is no existing operator declaration/overload authority to carry a required `operator*`, `operator+`, or `operator==` witness through open generic bodies and semantic artifacts. Adding a separate operator lookup just for `Standard.Math` would create a second overload system; the next batch should resolve that authority before expanding Math or generic equality helpers. Existing tests that relied on the old byte-quantity-to-`usize` bridge now use `Magnitude` explicitly, and the enum checked outputs are regenerated. This is meaningful progression, not R8g completion. EVT2 was not started.
