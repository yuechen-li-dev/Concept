# SMLib Phase 1a compiler findings

The separate consumer project is `../SMlib_Concept/phase1a`. Its original Phase 1
sources and evidence remain frozen, with all 41 source and six evidence hashes
checked independently. No retired Zig changes, grammar extensions, new runtime
machinery, or numerical algorithm changes were needed.

Three bounded correctness fixes were required:

1. `2614c95`: tensor descriptor collection now includes closed generic bodies,
   matching actual C emission. Previously imported tensor templates type-checked
   and then produced undeclared C tensor descriptor names.
2. `45f3e0f`: dependent static assertions retain typed syntax until closure,
   reusing static-control dependency validation. Invalid concrete dimensions,
   runtime operands, non-boolean conditions and non-string messages still reject.
3. Closed-template callback environments and erasure adapters now use the same
   closed-body projection for semantic/MIR and C collection. Literal identities
   include the concrete lexical instantiation; inferred open literal types are
   rebound on substitution. Operation/effect declaration identity stays unchanged.
   This enables an imported evaluator contract with a locally captured adapter.

`internal/concept/smlib_phase1a_test.go` covers imported tensor templates and
dimension rejection, imported positive/negative static assertions, two concrete
callback instantiations sharing a signature, repeated instantiation/witness
deduplication, generic owned-capture cleanup exactly once, and local escape
rejection. Native tests execute in Normal/Verify; tensor/callback output also
passes strict C11. Existing generic effect-propagation tests protect the distinction
between operation authority and concrete callback identity.

Two limitations are retained as consumer reproductions rather than patched around:

- Open local `Fixed<N>` construction is rejected with
  `GENERIC_NON_TYPE_ARGUMENT_INVALID`, although a closed `Fixed<3>` and generic
  `double<array>[N]` storage work. The failure spans known-type validation,
  structural value arguments and materialization; the consumer uses the supported
  array form and records the composition gap for a separate dependent-type fix.
- Nested destination tensor expressions such as `Y = X + D * scale` reject with
  CV4617. Current whole-view tensor lowering accepts one binary operation; indexed
  Einstein notation expresses the same elementwise update in one statement.
  General nested expression fusion is deferred, not implemented as implicit scratch.

The consumer report records qualification, code/stack inspection, paired timing,
and feature selection. `go test -count=1 ./...` and `go vet ./...` are the full
compiler qualification commands; frozen Zig suites are deliberately skipped.

Final qualification on Windows AMD64 passed at `6dc97b2`: full Go tests
(`internal/concept` 329.022s), vet, focused native regressions, and the unchanged
external Phase 1 corpus in Normal/Verify. Each of the five consumer candidates
passes 239 cases / 478 comparisons, with exact historical numerical-report parity.
The selected executable also passes all 239 cases from an executable-only directory.
Nine paired benchmark rounds show a small-system gain and a roughly 2% 16-variable
regression; the consumer report retains dispersion, increased code/stack costs,
tensor equivalence, and the higher cost of explicit stepping.
