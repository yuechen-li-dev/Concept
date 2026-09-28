# R8g natural-authoring convergence: partial issue ledger

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
