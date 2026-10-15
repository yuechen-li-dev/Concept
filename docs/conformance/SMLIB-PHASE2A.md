# SMLib Phase 2a compiler generation qualification

This milestone changes portable C11 generation, without changing the spline
recurrences, language grammar, ownership rules, or floating-point policy.
The consumer experiment and complete paired results live in
`SMlib_Concept/phase2a`. Phase 2 geometry, modern C++ and original NLib sources
remain frozen there by SHA-256.

## Generation changes

* A literal unit-step builtin range advances directly. Its half-open loop
  condition proves the next cursor is representable and cannot overshoot the
  endpoint. This also holds at signed and unsigned integer limits. Larger,
  dynamic and named-range steps retain the overflow-safe distance/clamp path.
  Range construction checks and once-only source evaluation are unchanged.
* Validation marks fixed-shape array/ndarray access as statically valid only
  when every axis has a constant in-range index. The existing per-access
  bounds planner eliminates that guard in Normal. Verify retains it. Evidence
  is recomputed during validation and never trusted from a module artifact.
  Dynamic indices, tensor indices and span indices retain checks.
* C emission computes a conservative identifier dependency closure for private
  closed template definitions. Imported open-body validation can materialize
  closed instances that no emitted runtime operation references. Their C
  declarations and bodies are omitted, while validation, semantic MIR, ordinary
  exported functions, and runtime transitive dependencies remain intact.
  Async instances remain roots because they own several generated entry names.
  Identifier matches in literals/comments can retain extra code, safely.
* An unsupported early `typeof(Identity<int>(1))` query now diagnoses
  `CALLABLE_TYPE_QUERY_INVALID`, rather than dereferencing an unprepared template
  requirement environment. This does not extend exact-callable type inference.

## Focused regression tests

```
go test ./internal/concept -run '^(TestUnitStepRangePreservesIntegerLimitsAndGeneralFallback|TestStaticArrayBoundsPlanIsPerAccessAndVerifyRetainsChecks|TestImportedQueryOnlyTemplateBodiesAreNotEmitted|TestTemplateEmissionIdentifierClosureAndConservativeAsyncRoot|TestEarlyTemplateTypeQueryDiagnosesInsteadOfPanicking)$' -count=1
```

Tests execute strict-C11 native harnesses, cover Normal and Verify, signed
INT32 limits and UINT64 limits, dynamic steps, per-access proof selection,
unknown span lengths, invalid constant indices, imported transitive template
calls, dead instance cycles, identifier boundaries, conservative async roots,
deterministic generation, and unused-body validation.

No new proof of a dynamic spline index is claimed. Contiguous storage,
readonly capability, lifetime safety, and disjointness do not establish the
inequality needed to eliminate an index check. No `restrict` contract is added.
Checked signed arithmetic, conversion, parameter validation and projection
failure handling remain in force.

## Measured limitations

The unit-step change reduces general iterator machinery but changes Clang's
optimization choices. Some derivative workloads regress and object text grows;
the consumer report retains these results. The geometry translation unit has
one live runtime-degree template instance, which must remain emitted. The dead
body reduction is demonstrated separately with an artifact-imported library;
it is not credited with a geometry runtime improvement. Backend liveness still
lowers candidate bodies before selecting them and is not frontend pruning.

The final consumer qualification records full Go tests/vet, deterministic native
artifacts, all geometry comparisons and mathematical checks, semantic proofs,
and frozen Phase 1/1a parity. Root and legacy Zig suites are skipped because the
frozen Zig compiler and its build/test infrastructure are unchanged.
