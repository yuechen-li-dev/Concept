# EVT1 R3 conformance record

Status: bounded resource transfer, deterministic drop, and first-class
reference evidence

## Summary

```text
31 total
30 PASS
1 EXPECTED-DIVERGENCE
0 EVT1-GAP
0 POC3-LEGACY
0 SPEC-AMBIGUITY
```

The retained divergence is ordinary structurally-copyable struct copy. EVT1
keeps the R2 law that an ordinary struct copies when all fields copy; PoC3
Phase 10 required an opt-in `Copy` witness.

## Model

`owned T` is R3's already-parsed movable-only representation. The ownership
qualifier makes an owning value non-copyable while retaining movability unless
the underlying type is immovable. Existing owners transfer only through
`move`; fresh construction and function results initialize ownership directly.
Unqualified values remain structurally copyable.

`void Drop(owned T value)` is the bounded destructor witness. MIR cleanup
metadata records each owner, type, witness, reverse order, and whether the
owner remains live or was transferred. Generated C materializes a return value
before cleanup, drops live locals and owned parameters, and omits transferred
sources.

`ref T` and `ref const T` are explicit aliases. Both require places and never
transfer ownership. Mutable binding from a const place is rejected; const
binding from mutable or const storage is accepted. Record immutability remains
type-level, and immovable values are mutable through `ref` without relocation.
All reference returns are conservatively rejected in R3.

## Evidence classification

| Area | Evidence | Provenance | Classification |
|---|---|---|---|
| ordinary structural copy | source remains usable after copy | R2 decision versus Phase 10 | `EXPECTED-DIVERGENCE` |
| explicit move and reinitialization | success, use-after-move, second move, restored state | Phase 10 | `PASS` |
| simple control flow | both-path move and maybe-moved rejection | Phase 10 | `PASS` |
| call and return transfer | explicit success and implicit-copy rejection | Phase 10 | `PASS` |
| immovable relocation | move rejected | R2 plus Phase 10 pressure | `PASS` |
| deterministic drop | early return, reverse order, moved-source suppression, callee parameter | Phase 10 | `PASS` |
| mutable and const references | explicit calls and native C11 | EVT1-new | `PASS` |
| const/record interaction | invalid mutable binding and field mutation | EVT1-new plus R2 | `PASS` |
| immovable by reference | Core and Vulkan positive paths | EVT1-new plus R2 | `PASS` |
| owned borrow then move | references preserve ownership state | EVT1-new | `PASS` |
| obvious escape | reference return rejected | EVT1-new | `PASS` |

The executable corpus is under `language/evt1-r3/core`. The table-driven
harness requires provenance and semantic-family diagnostics for every negative
case. New reference behavior is labeled EVT1-new rather than presented as a
PoC3 divergence.

## Explicit boundary

R3 does not implement implicit moves, field/partial moves, generalized borrow
checking, named lifetimes, safe reference returns, `ref struct`, `scoped`,
Span types, temporary lifetime extension, partial drop, unwinding, live
resource replacement, allocation, runtime collections, Option/Result,
interfaces/dyn, C ABI, testing syntax, panic/assert, automata reconciliation,
or multi-module compilation.

## Recommended R4 scope

R4 should remain a bounded ownership hardening milestone: validate cleanup
metadata as a MIR invariant, decide and implement path-sensitive conditional
cleanup or explicitly reject maybe-moved Drop owners at scope exit, and define
drop-old-then-initialize-new replacement for a live owner. It should not begin
Span, `ref struct`, allocation, runtime collections, or generalized lifetime
inference without a separate design milestone.
