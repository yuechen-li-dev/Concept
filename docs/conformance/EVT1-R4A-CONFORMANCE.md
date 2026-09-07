# EVT1 R4a conformance record

Status: bounded lifetime-aware references, semantic proof requirements, and
cleanup hardening

## Summary

```text
11 total
11 PASS
6 valid
5 invalid
```

The executable corpus is under `language/evt1-r4a/core`. The Go harness checks
the diagnostic family for every negative case, inspects MIR ref-struct and
semantic-proof facts, verifies generated-C replacement order, and executes
four strict C11 native cases.

## Proven behavior

- `ref struct` is a value aggregate with a lifetime bound derived from its
  reference/ref-struct fields and no runtime lifetime object.
- `scoped ref T` and `scoped ref const T` are explicit non-escape constraints.
- lexical `Local`, `Parameter`, static/global, and conservative `Unknown`
  provenance support bounded return and longer-storage checks.
- unrestricted structs and enum payloads cannot contain references or ref
  structs.
- concept satisfaction resolves operation, prerequisite, and compiler-analysis
  requirements through one resolver. `LifetimeSafe<T>` is invoked by a real
  constrained template and recorded as a MIR semantic proof.
- MIR cleanup obligations are validated before emission. Live `owned T`
  replacement drops the old value, initializes the new value, and later drops
  that new live value exactly once.

## Explicit boundary

R4a does not implement Span, ReadOnlySpan, stackalloc, runtime Slice, named or
non-lexical lifetimes, a global borrow checker, general alias exclusivity,
interface/dyn runtime reification, or unsafe analysis bypasses.
