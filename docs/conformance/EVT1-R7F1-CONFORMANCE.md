# EVT1 R7f1 generic library closure

R7f2 follow-up: [EVT1-R7F2-CONFORMANCE.md](EVT1-R7F2-CONFORMANCE.md) adds an explicit destructive reference effect and conservative suspension rejection for resource-sensitive leases. The wider machine/foreign boundary remains open.

Status: **Meaningful progression**. Baseline and R7e: `b1747d7d3ffba7510aff6484c6863f70c214cf6e`; compiler ID: `concept-evt1-stage0-go`. R7f stopped without a collector. R7f1 adds no collector or scheduler integration.

## Evidence

| Area | Original result | R7f1 evidence |
| --- | --- | --- |
| Generic calls | An open `Forward<T>` calling `Identity<T>` reported CV4174. | Closed calls typecheck and execute in strict C11, including artifact-only imports; recursive instances report `GENERIC_INSTANTIATION_RECURSIVE`. |
| Generic required operations | Requirements had only monomorphic signatures. | `requires template <typename U> U Transform(ref const T, U)` retains its local parameter in the AST and `concept-module.v1`. A generic caller binds `T` and `U` separately; concrete witness matching checks arity, parameters and return type. Artifact-only consumer executes through C11. |
| Value-shaped concept | The independent `Visit<T,V>` reproducer already compiled before R7f1. CV4102 in the R7f draft came from putting a non-type `Capacity` in a concept parameter list, not from a value or reference operation on `T`. | Multi-type-parameter value/reference operation, concrete witness and generic invocation execute through C11. Non-type concept parameters remain unsupported. |
| Generic local array | `int<array>[N]` in an open template reported CV4558. | Direct generic-value extents remain symbolic until instantiation. Closed `N=4` and `N=32` generate fixed C11 array types and execute. Runtime extents still report CV4558. |
| Borrow/reclaim | A live reference could coexist with an `owned` move into `Reclaim`. | Lexical reference and reference-struct leases block moving the same owner; unrelated owners and reclaim after scope exit work. Imported APIs and a combined artifact-only probe execute. |

The combined probe in `TestR7f1CombinedLibraryProbeFromArtifact` invokes a generic required operation from a generic body, uses `int<array>[N]` metadata, and holds a `Lease` during a legal read. Reclaim inside that lease scope is rejected. Focused native tests use the repository strict-C11 harness. The 100-run test compares artifact, MIR, and C bytes. The semantic corpus manifest includes the five valid and one invalid R7f1 fixtures.

Final gates: `go test ./...`, `go vet ./...`, both `zig build test` roots, `oct make BurnIn --file Make.oct`, and `oct make Test --file Make.oct` passed. The latter built and tested Standard (3 facts) and DragonGod (21 facts). `oct` was built from the adjacent Oct checkout to a temporary executable because it was not on PATH. The full Go suite includes R6m/R6o/R6p, R7d6, R7e, and table/ellipsis regressions. `git diff --check` passed.

## Boundary

The borrow exclusion is based on ordinary `owned` move authority. A function taking `ref Store` and mutating/releasing its pointee has no declared destructive effect and is not excluded by this check. Borrow provenance through every possible aggregate/control-flow form, and lease behavior across async/yield suspension, are not established by R7f1. These are the remaining reasons for **Meaningful progression** rather than Success. The collector retry must not assume `Reclaim(ref Store)` or suspension is safe.

`concept-module.v1` remains the schema; the typed requirement gains optional `generic_params`, which is included in deterministic serialization and hash coverage. No runtime generic dictionary, VLA, borrow registry, collector MIR, or collector-specific compiler branch was added.
