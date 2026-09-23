# EVT1 destructive resource effects

Ordinary mutation does not imply lifetime invalidation. Operations that may end lifetime or reclaim backing storage must carry explicit destructive-resource semantics.

The bounded spelling is:

```concept
void Reset(ref Store store) { ... }
requires compiler.InvalidatesBorrows(Reset, store);
```

`store` must name a reference parameter. The declaration uses the existing operation-effect channel, rather than a new function modifier or runtime effect. Its parameter name is included in the module's hash-covered effect summary. At a call, the checker compares the actual reference argument's structural resource path with lexical borrow sources. Exact overlap rejects; a distinct owner remains legal. An alias carrying a known borrow path retains that identity. If the target path cannot be established while a borrow is live, the checker rejects conservatively.

Forwarding an invalidating reference parameter requires the forwarding operation to declare the same effect. This keeps a wrapper from hiding the authority boundary. The rule applies to ordinary and instantiated generic calls. Conflict diagnostics retain declared or imported effect origin and the live borrow in a proof graph. This is compile-time metadata; generated C uses ordinary calls.

Current closure is deliberately bounded. A ref-accepting foreign C operation is rejected by existing ABI validation, and no general foreign pointer provenance is inferred. Effects on concept-required operations, dynamic interfaces, and arbitrary aggregate field writes remain to be established before a collector relies on this rule.
