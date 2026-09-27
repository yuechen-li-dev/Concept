# EVT1 R7n conformance

Status: **Meaningful progression** against the R7n threshold. This is not a
full R7n success because immovable insertion and `Span<T>` over payloads are
not implemented. Arena reset and graph convenience are deferred.

Baseline: `bf84656b47c79b1d623b1f7593cceaa46573a1ea`, compiler
`concept-evt1-stage0-go`. Existing Standard.Memory provides explicit
allocator regions, bump/pool allocation, typed `Storage<T>`, initialization,
Destroy/Release ordering, and collector owners. Arrays and `table<N>` provide
fixed storage; `Span` and `ReadOnlySpan` borrow contiguous arrays. Generic
types and `usize` non-type parameters, explicit move/Drop, generated
declarations, semantic artifacts, and `reflect<T>` already exist.

The new `Standard.Collection.Stores` API uses inline bounded arrays and no
allocator. `Id<T>` is a nominal 32-bit Concept `int` index;
`GenerationalId<T>` adds a 32-bit generation. `DenseStore` is append-only.
`GenerationalStore` uses deterministic LIFO reuse and retires exhausted
generations. Both report bounded errors. The AST fixture builds and evaluates
an `Add` node through IDs; a mismatched ID type is statically rejected.
Noncopyable movable values work, and removal plus scope exit drop each live
value exactly once. Normal and Verify Standard facts exercise invalid and
stale IDs. Concrete wrappers prove `NoAllocation` for append, insert, and
remove. An artifact-only consumer builds from Standard semantic modules and
executes under strict C11 in Normal and Verify; Clang also checks
`-pedantic-errors -Wall -Wextra -Werror -Wno-unused-function`. Generated C is
checked for heap calls and repeated 100 times without output drift.
The artifact-only fixture also emits a `NoAllocation` proof into MIR, so the
100-run output comparison covers that proof as well. The existing package
determinism test rebuilds Standard and DragonGod artifacts 100 times.

Informational five-run native benchmark means on this host: combined typed
dense insert/get/iteration plus generation reuse 7.96 ms; plain array
insert/iteration 6.36 ms. Process overhead dominates these measurements; they
are not a throughput ratio. Generated C uses direct indexed arrays and a
bounded metadata stack, with no heap-node path.

Final validation on the settled source: `go vet ./...`, root and legacy
`zig build test`, R7d3 BurnIn, and the full semantic manifest test pass.
`go test ./...` passes (internal/concept 146.198 seconds), including the
100-run Standard/DragonGod package graph and R7n artifact consumer gates.
Standard passes 34 facts and two informational benchmarks in Normal and
Verify; DragonGod passes 23 facts and one benchmark in both modes. The EVT1
manifest remains 398 valid, 262 static-invalid, 13 runtime-negative, five
compatibility, and four expected-divergence fixtures. `oct` is not on PATH;
the exact Go commands named by the BurnIn, Standard, and DragonGod Make targets
were run directly.
The library currently lacks contiguous payload `Span<T>` and in-place
immovable insertion, so those success criteria remain open.
