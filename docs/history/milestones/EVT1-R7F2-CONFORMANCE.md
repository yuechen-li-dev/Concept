# EVT1 R7f2 destructive resource effects: meaningful progression

R7fR follow-up: [EVT1-R7FR-CONVERGENCE.md](EVT1-R7FR-CONVERGENCE.md)
uses this borrow invalidation effect in an ordinary collector library.

R7f3 follow-up: [EVT1-R7F3-CONFORMANCE.md](EVT1-R7F3-CONFORMANCE.md)
records an honest stop at indexed storage-authority replacement.

Baseline: `58dae662d4559b2c0ea6843767e36a54dfbd706d` (`concept-evt1-stage0-go`). R7e: `b1747d7d3ffba7510aff6484c6863f70c214cf6e`. R7f1 is the baseline commit. The work began with a clean tree.

The independent destructive-ref reproducer accepted `Borrow(ref const store); Reset(ref store); Use(held)` before R7f2. A local reference-struct lease across `await` was also accepted. The new contract `requires compiler.InvalidatesBorrows(Reset, store);` declares that the named reference parameter may invalidate borrows from its resource. The ordinary operation-effect representation and `concept-module.v1` effect summary carry the parameter name. The checker compares the call's structural resource path with live reference/lease source paths. A different resource and reclaim after scope exit remain legal. Ordinary mutation without the contract remains legal.

A function forwarding an invalidating reference parameter must declare its own contract. Direct and generic forwarding have directed negative tests. An imported contract rejects the overlapping call using only its artifact. The diagnostic includes a deterministic proof graph with operation contract origin, resource identity, and live borrow; `SerializeProof` produces JSON. The focused test compares artifact, MIR, generated C, and proof bytes across 100 runs.

For suspension, R7f2 takes the conservative option for reference structs carrying references to resources with invalidation contracts. A local such value live across `await` is rejected by async persistence validation. A live local lease across `yield` is rejected, and a lease cannot be placed in persistent automata/machine fields. An awaited destructive call is rejected at its call site while a lease is live. Existing unrelated async reference structs and machine-backed spans remain valid. There is no runtime borrow registry, lock, or collector-specific compiler branch.

## Remaining boundary

This is **Meaningful progression**, not full R7f2 Success. A suspension-safe persistent lease has not been proved. Consequently, the requested different-resource awaited call with a lease held across suspension is rejected conservatively. Parent/child machine authority and exact lease Drop on cancellation/failure have not been established. `extern "C"` currently rejects `ref Store` parameters with `EXTERN_C_ABI_TYPE_INVALID`, so a foreign destructive-ref contract requires a separate bounded interop design. Required-operation and interface effect transport have not been proved. Borrow provenance through every aggregate field update remains outside the R7f1 lexical proof. No collector-shaped fixture or collector API is claimed.

Corpus additions are under `language/evt1/destructive-resource`: two valid and three static-invalid cases. Valid focused cases execute through the repository strict-C11 harness. The corpus manifest totals are updated. The general contract and suspension decisions are described in the adjacent design notes.

Validation: `go test ./...`, `go vet ./...`, both `zig build test` lanes, `oct make BurnIn --file Make.oct`, and `oct make Test --file Make.oct` passed. Oct was run from a temporary executable built from the adjacent checkout because `oct` was not on PATH; the Make test graph built and tested Standard (3 facts) and DragonGod (21 facts). The full Go suite includes the R7f1, R7e, and ownership/provenance regressions. `git diff --check` and the collector-name audit of changed compiler files passed.
