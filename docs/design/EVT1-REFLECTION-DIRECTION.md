# EVT1 reflection direction

R7g uses the ordinary type AST and semantic module artifact as its structural authority. `reflect<T>;` is a compile-time request; the compiler resolves `T`, creates a one-level typed snapshot, and erases the request before MIR. Field and case order follows declaration order. Generic applications use the existing monomorphization/substitution path. The dependency artifact supplies imported structure, and type-level `[[reflect]]` supplies explicit cross-module permission.

R7g1 adds a bounded checked declaration-generation operation. A generator consumes typed reflection results, constructs an ordinary function AST with stable identity and source provenance, appends it before ordinary semantic analysis, and uses the existing MIR/backend path. Generator site and reflected fields survive in diagnostics, module artifacts, generated inspection, and a proof view. The ordinary bounded `comptime` value evaluator remains separate. Token substitution is not used. See [EVT1-GENERATED-DECLARATIONS.md](EVT1-GENERATED-DECLARATIONS.md) for the supported shape and remaining limits.

`Trace<T>` derivation should live in Concept library code and select collector-managed fields by an explicit semantic rule. The compiler must have no `CollectorHandle`, `Trace`, or serializer-specific generation branch. Handwritten `TraceReferences` remains valid, and ordinary coherence must reject duplicate handwritten/generated declarations.

Test-only runtime structural metadata is a separate later slice. It must be requested by a `.concept_test` function marked `[[reflect]]`, materialized only for the reflected types, and excluded from production artifacts. No general runtime invocation or mutation surface is planned for EVT1.
