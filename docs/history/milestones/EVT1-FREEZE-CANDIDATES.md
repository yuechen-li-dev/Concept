# EVT1 freeze candidates from R7p

| Candidate | Golden / evidence | Compatibility impact | Decision |
| --- | --- | --- | --- |
| Teach C-style `for` rejection | Embedded first draft | Diagnostic only | Implemented: points to `for (i in start..end step n)` and bounded `while` |
| `AssumeQuantity<T>(scalar)` | Civilian aerospace native sample | Additive, explicit trust boundary | Implemented with numeric representation check; erases to scalar C |
| Dimensionless literals in `*` and `/` | CAD centroid `length / 3.0` | Correctness repair for quantity arithmetic | Fixed: multiplication and division keep literals dimensionless; additive and comparison context remains unit-aware |
| `T[N]` as canonical fixed array | Embedded, storage, CAD | Source and formatter churn | Deferred. Both `T[N]` and legacy `T<array>[N]` appear; no golden showed a semantic obstruction demanding migration. |
| Payload enum in `Span` | Compiler token stream | Storage geometry change | Deferred. Fixed scalar token record is the current answer; no implicit variable-size element layout. |
| Implicit quantity literal argument conversion | CAD | Overload resolution and inference change | Deferred; typed local is clear enough for EVT1. |
| Imported `CAbiValue` without native probe | Civilian aerospace `NativeSample` | None | Intentional Unknown. The companion golden demonstrates measured `NativeToolchainProbe` authority. |
| `else if` sugar | Compiler parser | Parser-only surface change | Deferred; nested `else { if (...) }` works. |

All candidates are resolved for EVT1. The additive quantity attachment and
quantity-arithmetic correction passed the full R7p qualification; the other
entries have explicit intentional or deferred decisions. No further syntax,
type, ownership, or effect/proof changes are admitted after this point except
correctness repairs.
