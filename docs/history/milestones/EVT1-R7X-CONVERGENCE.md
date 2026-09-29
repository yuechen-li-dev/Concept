# EVT1 R7x convergence record

## Baseline and scope

Started from clean `37a11a2` after R7k. The compiler ID is
`concept-evt1-stage0-go`. Existing runtime iteration was `foreach` over
arrays, spans, and explicit iterator protocols; it already lowered known
storage to inline counted C11 loops. `for` was rejected with `CV4237`.

The modulo audit found a discrepancy with the R7x premise: signed `%` was
documented as Euclidean but rejected with
`SIGNED_EUCLIDEAN_MODULO_DEFERRED`. Unsigned `%` previously emitted raw C
remainder without a zero-divisor check. Both gaps are now closed in the
ordinary validation and C11 lowering paths.

## Additional source audit

The DragonGod event loop used `foreach (Event event in self.events)`. It now
uses `for (Event event in self.events)` and exercises the same iteration path.
No broad source rewrite was made.

## Next blocker

Range authoring is the next structural blocker. There is no range value or
literal in the current language AST. The lexer recognizes `...` for array
initializers but not `..`, and the `for` source is a general expression.
Boundedness is attached to `while (...) bounded(...)`. The requested range
extent, step, descend, overflow, and loop-control semantics need to be
represented and verified together before emitting C11. Adding a private
parser rewrite would make the code less legible and would bypass those
contracts.
