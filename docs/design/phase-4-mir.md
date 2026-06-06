# Phase 4 MIR architecture

Phase 4 starts the compiler's explicit control-flow layer. It is a design and implementation phase for MIR, but **P4-M0 is documentation only**: no compiler code, fixture runner behavior, backend migration, or language feature expansion is part of this milestone.

## P4-M1 implementation note

P4-M1 adds the standalone MIR skeleton in `src/compiler/mir.zig`: compact MIR IDs, functions, locals, blocks, allocator-owned storage, manual construction APIs, lookup helpers, and debug snapshot rendering. Places, operands, rvalues, statements, terminators, validation, and HIR-to-MIR lowering remain future Phase 4 milestones.

## P4-M2 implementation note

P4-M2 adds the core MIR computation and control-flow vocabulary for manual construction: local places, literal/copy operands, use/unary/binary/call rvalues, assignment statements, and goto/return/switch/unreachable terminators. HIR-to-MIR lowering, MIR validation, and backend migration remain future work.

## P4-M3 implementation note

P4-M3 adds straight-line HIR-to-MIR lowering for function bodies with parameters, local declarations, assignments, returns, literals, references, grouping, unary/binary expressions, and calls. Structured `if`/`while`/`match` lowering, MIR validation, MIR fixture running, and backend migration remain future work. Straight-line bodies that fall through receive an `Unreachable` terminator.

## P4-M4s implementation note

P4-M4s extends HIR-to-MIR lowering from straight-line bodies to structured control flow. `if`/`else` now lowers to `SwitchBool` plus branch and join blocks, `while` lowers to explicit entry/condition/body/exit blocks with a back edge, and `match` over checked bool/int scrutinees lowers to `SwitchBool` or `SwitchInt` with one block per arm and a join/default target. This sweep intentionally does not add `break`, `continue`, payload enum matching, guards, destructuring, exhaustiveness checking, MIR validation, or backend migration.


## P4-M7 implementation note

P4-M7 adds `src/compiler/mir_validator.zig`, a MIR validation pass for the current lowered representation. The pass checks structural references for functions, locals, and blocks, requires block terminators, and verifies simple `TypeId` consistency across places, operands, rvalues, assignments, returns, switches, and calls. This is deliberately a MIR sanity pass only: it is not borrow checking, move checking, drop elaboration, optimization, backend legalization, or storage live/dead analysis. Backend migration remains future P4-M8 work.

## Thesis

```text
Phase 3 made Concept understand programs through HIR.
Phase 4 lowers checked HIR into explicit compiler control flow:
MIR functions, locals, basic blocks, statements, operands, rvalues, and terminators.
```

At Phase 3 closeout the authoritative Stage 0 run path is:

```text
Concept source
  -> parseSource
  -> semantic collection / HIR lowering
  -> HIR executable checker
  -> HIR-backed C backend
  -> zig cc
  -> native executable / exit-code fixtures
```

Phase 4 keeps that path working while introducing the next compiler representation between checked HIR and backend emission.

The target Phase 4 pipeline is:

```text
Concept source
  -> parseSource
  -> semantic collection / HIR lowering
  -> HIR executable checker
  -> HIR-to-MIR lowering
  -> MIR validation
  -> MIR-backed C backend
  -> zig cc
  -> native executable / exit-code fixtures
```

## Goals

Phase 4 should add the compiler structures needed for explicit control-flow lowering without broadening the runtime language.

Required goals:

- Introduce a MIR module/function representation.
- Introduce MIR locals and compiler temporaries.
- Introduce MIR basic blocks.
- Introduce MIR places.
- Introduce MIR operands.
- Introduce MIR rvalues.
- Introduce MIR statements.
- Introduce MIR terminators.
- Lower HIR to MIR for the full Phase 2 executable subset.
- Add MIR debug dump / snapshot support.
- Add a MIR validation pass.
- Eventually migrate C backend emission from HIR to MIR.
- Preserve Phase 2 and Phase 3 run/check fixtures.

The first implementation milestones should keep MIR deliberately small and readable. The MIR must be powerful enough to express the existing executable subset, but not powerful enough to become a dumping ground for future language features before those features have semantic designs.

## Non-goals

Phase 4 deliberately does not include:

- New Concept syntax.
- Arrays.
- Runtime strings, chars, or floats.
- Struct values.
- Field access.
- Enum runtime layout.
- Payload enum matching.
- Borrow or move checking.
- `Drop<T>`.
- `Result<T, E>` / `try`.
- Generics or concept solving.
- Interfaces or vtables.
- Multi-file modules or imports.
- LLVM.
- Bare-metal targets.
- Self-hosting.
- Optimization passes beyond trivial formatting and validation.
- An SSA requirement.

Phase 4 is about making checked executable HIR explicit. It is not about adding language power.

## HIR vs MIR

```text
HIR is semantic and still structured.
MIR is explicit control flow.
HIR keeps if/while/match as structured constructs.
MIR lowers them into blocks and terminators.
```

HIR answers questions such as:

- Which declaration does this name reference?
- Which `TypeId` does this expression have?
- Which local or parameter does this identifier resolve to?
- Which function does this call target?
- Which structured statement did the programmer write?

MIR answers different questions:

- Which block executes next?
- Which local or temporary stores this value?
- Which statement computes this rvalue?
- Which terminator exits the block?
- Which branch target receives control for this condition?

HIR-ish structured loop:

```text
While
  Condition: Binary < LocalRef x, Int 7
  Body:
    Assignment x = Binary + LocalRef x, Int 1
```

MIR-ish explicit control flow:

```text
bb0:
  goto bb1

bb1:
  _tmp0 = Lt(copy _x, 7)
  switch_bool copy _tmp0 -> [true: bb2, false: bb3]

bb2:
  _tmp1 = Add(copy _x, 1)
  _x = copy _tmp1
  goto bb1

bb3:
  return copy _x
```

The important distinction is not syntax. The important distinction is that MIR has no structured `while`; it has blocks, statements, and terminators.

## Proposed MIR data model

The initial MIR should use stable IDs and dense arrays, matching the compiler's existing preference for explicit indexes over pointer-shaped node graphs.

Proposed top-level structures:

```text
MirModule
  functions: []MirFunction

MirFunction
  id: MirFunctionId
  hir_function: FunctionId
  name: SymbolId
  params: []MirLocalId
  locals: []MirLocal
  blocks: []MirBlock
  return_type: TypeId
  source_span: ?SourceSpan

MirLocal
  id: MirLocalId
  name: ?SymbolId
  kind: MirLocalKind
  type: TypeId
  source_span: ?SourceSpan

MirBlock
  id: MirBlockId
  statements: []MirStatement
  terminator: MirTerminator
  source_span: ?SourceSpan

MirStatement
  kind: MirStatementKind
  source_span: ?SourceSpan

MirTerminator
  kind: MirTerminatorKind
  source_span: ?SourceSpan

MirPlace
  kind: MirPlaceKind
  source_span: ?SourceSpan

MirOperand
  kind: MirOperandKind
  source_span: ?SourceSpan

MirRvalue
  kind: MirRvalueKind
  source_span: ?SourceSpan
```

Proposed IDs:

```text
MirFunctionId
MirLocalId
MirBlockId
MirStatementId    // optional; useful only if diagnostics or tooling need stable statement IDs
```

`MirStatementId` should be deferred unless snapshots, diagnostics, or later analyses clearly benefit from it. Dense statement order inside a `MirBlock` may be enough for Phase 4 v0.

## MIR locals

MIR locals are the addressable storage slots inside one MIR function.

Phase 4 local sources:

- Function parameters become MIR locals.
- User local declarations become MIR locals.
- Compiler temporaries become MIR locals.
- A dedicated return slot is optional and should be deferred unless it simplifies backend emission or later lowering.

Suggested local kind:

```text
param
user
temp
return_slot maybe later
```

Example local table:

```text
locals:
  _0: param value: int
  _1: user x: int
  _2: temp: int
```

A MIR local's `TypeId` comes from HIR checking. MIR lowering should not rediscover types from source text.

## MIR places

A place identifies where a value can be read from or assigned to.

Phase 4 v0 places are local-only:

```text
Place.Local(MirLocalId)
```

Example:

```text
_x = copy _tmp0
```

Future places may include:

- Field projection.
- Index projection.
- Dereference projection.

Those projections are not Phase 4. Keeping v0 places local-only prevents MIR from implying support for field access, arrays, references, or pointer semantics before those language features exist in the executable subset.

## MIR operands

An operand is an immediate value or a read from a place.

Phase 4 operands:

```text
Operand.IntLiteral("1")
Operand.BoolLiteral(true)
Operand.Copy(Place.Local(_x))
```

`copy` is intentionally modest in Phase 4. It means "read this place" for the current executable subset.

Phase 4 explicitly does not have real move semantics:

- No moved-from state is tracked.
- No destructor or drop behavior is inserted.
- No copy/move distinction affects program validity.
- Real move/drop semantics are later MIR and semantic work.

## MIR rvalues

An rvalue describes how to compute a value assigned into a place.

Phase 4 rvalues:

```text
Use(Operand)
Unary(op, Operand)
Binary(op, Operand, Operand)
Call(FunctionId, []Operand)
```

Calls may be rvalues in Phase 4 because the current executable subset has no unwind edges, effects, checked error paths, or multiple continuations. Calls could become terminators later if Concept needs explicit unwind, effect, cancellation, async, or error-path control flow.

## MIR statements

Phase 4 v0 statements should start with assignment only:

```text
Assign(Place, Rvalue)
```

Examples:

```text
_tmp0 = Add(copy _x, 1)
_x = copy _tmp0
```

Optional future statements:

```text
StorageLive(LocalId)
StorageDead(LocalId)
```

Storage markers should be deferred unless validation, diagnostics, or backend emission clearly need them. Premature storage markers can make MIR snapshots noisy without improving the Phase 4 motivating path.

## MIR terminators

Every basic block ends with exactly one terminator. The preferred Phase 4 direction is to start with explicit specialized switches for the existing subset and generalize only when needed.

Phase 4 terminators:

```text
Goto(BlockId)
Return(?Operand)
SwitchBool {
  discriminant: Operand,
  true_target: BlockId,
  false_target: BlockId,
}
SwitchInt {
  discriminant: Operand,
  cases: []SwitchIntCase,
  default_target: BlockId,
}
Unreachable
```

Rationale:

- `SwitchBool` maps directly to `if`, boolean conditions, and simple loop conditions.
- `SwitchInt` maps directly to Phase 2 integer `match` statements.
- A generalized `Switch` can replace both later if richer discriminants or shared tooling make it worthwhile.
- `Unreachable` is useful for malformed, impossible, or deliberately unreachable blocks, but validation should ensure ordinary lowering does not leave accidental unterminated blocks.

## Lowering strategy

MIR lowering runs only after semantic collection, HIR lowering, and HIR executable checking succeed.

General rules:

- Functions lower one HIR function at a time.
- Function parameters become `param` locals before body lowering.
- User local declarations allocate `user` locals.
- Complex expression lowering introduces `temp` locals.
- Expressions lower into operands; complex rvalues are first assigned into temporaries.
- Statements append MIR statements to the current block.
- Blocks lower sequentially until a terminator is emitted.
- `return` emits a `Return` terminator.
- `if` creates condition, then, else, and join blocks as needed.
- `while` creates condition, body, and exit blocks.
- `match` lowers to a switch terminator plus arm blocks.
- Source spans should be carried from HIR into MIR nodes where useful for diagnostics and snapshot readability.

A useful lowering helper shape is:

```text
lowerExprToOperand(expr, current_block) -> Operand
lowerExprToPlace(expr, current_block) -> Place     // mostly future work
lowerStatement(stmt, current_block) -> current_block
lowerTerminatedBlock(block) -> BlockId
newTemp(type, span) -> MirLocalId
newBlock(span) -> MirBlockId
```

For Phase 4 v0, `lowerExprToPlace` should not grow beyond local assignment targets.

## Example lowerings

### Return literal

Source:

```cpp
int main() {
    return 7;
}
```

MIR-ish:

```text
fn main() -> int {
bb0:
  return 7
}
```

### Local declaration and assignment

Source:

```cpp
int main() {
    int x = 1;
    x = x + 2;
    return x;
}
```

MIR-ish:

```text
locals:
  _x: user int
  _tmp0: temp int

bb0:
  _x = 1
  _tmp0 = Add(copy _x, 2)
  _x = copy _tmp0
  return copy _x
```

### If/else

Source:

```cpp
int main() {
    if (true) {
        return 1;
    } else {
        return 2;
    }
}
```

MIR-ish:

```text
bb0:
  switch_bool true -> [true: bb1, false: bb2]

bb1:
  return 1

bb2:
  return 2
```

If both arms fall through, lowering should create a join block:

```text
bb0:
  switch_bool copy _cond -> [true: bb1, false: bb2]

bb1:
  _x = 1
  goto bb3

bb2:
  _x = 2
  goto bb3

bb3:
  return copy _x
```

### While loop

Source:

```cpp
int main() {
    int x = 0;
    while (x < 7) {
        x = x + 1;
    }
    return x;
}
```

MIR-ish:

```text
locals:
  _x: user int
  _tmp0: temp bool
  _tmp1: temp int

bb0:
  _x = 0
  goto bb1

bb1:
  _tmp0 = Lt(copy _x, 7)
  switch_bool copy _tmp0 -> [true: bb2, false: bb3]

bb2:
  _tmp1 = Add(copy _x, 1)
  _x = copy _tmp1
  goto bb1

bb3:
  return copy _x
```

### Match statement

Source:

```cpp
int main() {
    int x = 2;
    match (x) {
        1 => return 10;
        2 => return 20;
        _ => return 30;
    }
}
```

MIR-ish:

```text
locals:
  _x: user int

bb0:
  _x = 2
  switch_int copy _x -> [1: bb1, 2: bb2, default: bb3]

bb1:
  return 10

bb2:
  return 20

bb3:
  return 30
```

## MIR validation

MIR validation is a structural and type-consistency pass. It is not borrow checking, move checking, drop elaboration, exhaustiveness checking, optimization, or backend legalization.

Planned validation rules:

- All `MirFunctionId` references are valid.
- All `MirBlockId` values are valid inside the owning function.
- All `MirLocalId` values are valid inside the owning function.
- Every block has exactly one terminator.
- Terminators reference valid blocks.
- Operands reference valid locals.
- Assignment rvalue type matches place type.
- Return operand type matches the function return type.
- `Return(null)` is used only for `void` functions.
- Switch discriminant type is valid for the switch kind.
- `SwitchBool` discriminants are `bool`.
- `SwitchInt` discriminants are `int`.
- All switch target blocks are valid.
- Optional: reject unreachable blocks only if the project wants strict MIR snapshots; otherwise leave reachability analysis for later.

Validation should produce stable diagnostics rather than panics for invalid MIR that comes from compiler bugs or hand-authored invalid MIR fixtures.

## MIR fixture strategy

`.conception` fixtures should grow a MIR phase once MIR dumping exists.

Initial MIR fixture shape:

```text
# name: while lowers to mir
# phase: mir
# expect: pass

=== source ===
module Main;

int main() {
    int x = 0;
    while (x < 7) {
        x = x + 1;
    }
    return x;
}

=== mir ===
fn main() -> int {
  ...
}
```

MIR fixture matching should start with raw MIR snapshots. Raw snapshots should be stable enough for review but should not pretend to be optimized output.

Later, if optimized MIR exists, fixtures can use metadata such as:

```text
# mir: raw
# mir: optimized
```

Phase 4 only needs raw MIR.

Invalid MIR validation fixtures can use diagnostics:

```text
# name: missing terminator rejected
# phase: mir
# expect: fail

=== source ===
...

=== diagnostics ===
CON0042
```

The fixture runner should keep normal `.concept` source loading separate from `.conception` internal language-definition fixtures.

## Diagnostic code range

Reserve the `CON004x` range for MIR validation and MIR lowering diagnostics.

Initial proposed stable codes:

```text
CON0040 InvalidMirBlock
CON0041 InvalidMirLocal
CON0042 MissingTerminator
CON0043 InvalidMirType
CON0044 InvalidMirOperand
```

These codes do not need to exist during P4-M0. They are reserved so future MIR fixtures can become stable without colliding with Phase 2 or Phase 3 diagnostics.

## Backend migration plan

The current authoritative backend is HIR-backed C emission. Phase 4 should migrate backend emission only after MIR lowering and validation can represent the full Phase 2 executable subset.

Planned migration:

1. Lower checked HIR to MIR.
2. Validate MIR.
3. Add raw MIR snapshots for representative programs.
4. Emit readable C from MIR.
5. Make the run harness use MIR-backed C emission.
6. Preserve all Phase 2 and Phase 3 run/check fixtures.
7. Mark the HIR-backed C backend transitional or retire it once MIR-backed emission is authoritative.

The migration should keep generated C auditable. MIR-backed C is still a bootstrap backend, not the final production backend story.

## Milestone ladder

```text
P4-M0  MIR design document
P4-M1  MIR skeleton and IDs
P4-M2  Places, operands, rvalues, statements, terminators
P4-M3  Lower straight-line HIR to MIR
P4-M4  Lower if/else
P4-M5  Lower while
P4-M6  Lower match
P4-M7  MIR validation pass
P4-M8  C backend from MIR
P4-M9  MIR fixture corpus and Phase 4 closeout
```

The ladder intentionally keeps representation before lowering, lowering before validation-heavy fixture work, and backend migration after MIR can represent all existing executable control flow.

## Phase 4 close criteria

Phase 4 is complete when:

- MIR module/function/block representation exists.
- HIR lowers to MIR for the full Phase 2 executable subset.
- MIR snapshots exist.
- MIR validation exists.
- The run harness executes through MIR-backed C emission.
- Phase 2 and Phase 3 run/check fixtures still pass.
- The HIR-backed backend is retired or marked transitional.
- Docs and fixtures are updated.

The convergence requirement for Phase 4 is that the real motivating path improves: existing Concept programs should still parse, check, compile through `zig cc`, and run, but the backend-facing control flow should now be explicit MIR rather than structured HIR.

## P4-M8 implementation note

P4-M8 adds MIR-backed C emission as the authoritative executable backend path. The run harness now follows:

```text
Concept source
  -> parseSource
  -> semantic collection / HIR lowering
  -> HIR executable checker
  -> HIR-to-MIR lowering
  -> MIR validation
  -> MIR-backed C backend
  -> zig cc
  -> executable
```

The MIR C backend validates the MIR module before emitting C, so backend entry points do not trust unvalidated MIR. Generated C is intentionally direct and boring: basic blocks become `bbN` labels, branches become `goto`, `switch_bool` becomes an `if (...) goto ... else goto ...`, and `switch_int` becomes a C `switch` whose cases jump to MIR block labels. P4-M8 does not reconstruct pretty source-level `if`, `while`, or `match` structures.

The older HIR-backed C backend remains in the repository as transitional snapshot and compatibility coverage, but it is no longer the default run-fixture path. MIR fixture corpus expansion and Phase 4 closeout remain P4-M9 work.

## P4-M0 close criteria

P4-M0 is complete when:

- This design document exists.
- The MIR architecture, lowering plan, fixture strategy, milestone ladder, and Phase 4 close criteria are documented.
- No compiler implementation code has been added for MIR.
- `zig build test` still passes.
