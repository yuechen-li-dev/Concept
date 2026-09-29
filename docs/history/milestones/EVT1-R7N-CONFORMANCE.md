# EVT1 R7n conformance

Status: **Success — R7n closed by R7n3**. The R7n and R7n2 results below are
historical progression. Arena reset and graph convenience remain deferred.

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
The R7n baseline lacked contiguous payload `Span<T>` and in-place immovable
insertion; R7n3 addresses those criteria below.

## R7n2 follow-up

Baseline: clean `414bb636fafde148dc7e89b758844f1f43652581`;
compiler ID `concept-evt1-stage0-go`. The single-object `Storage<T>`
initialization path now constructs aggregate fields into the final address.
The strict C11 immovable specimen exercises a failed later field expression,
cleanup of the completed owned field, successful `Destroy`, and no complete
temporary `T` or hidden heap call in generated C. Owned fields now participate
in fixed layout geometry, and `Destroy` recursively drops owned fields when
there is no custom Drop for the containing record.

At R7n2, the result was **meaningful progression**: neither store
exposes stationary insertion, and `DenseStore` still has strided `Option<T>`
payloads rather than a contiguous `T[Count]` prefix. `GenerationalStore`
remains sparse by design and should keep checked ID traversal without a fake
payload Span. These criteria require a general, partially initialized inline
storage representation with explicit lifetime and relocation rules.

Validation of the R7n2 progression: `go test ./...` (including the existing
100-run R7n store artifact/MIR/C/proof comparison), `go vet ./...`, both Zig
test suites, R7d3 BurnIn, and the full EVT1 corpus manifest passed. Standard
passed 34 facts and two benchmarks in both Normal and Verify; DragonGod passed
23 facts and one benchmark in both modes. The new stationary `Storage<T>`
fixture ran natively and passed `-std=c11 -pedantic-errors -Wall -Wextra
-Werror -Wno-unused-function`. It is single-object substrate evidence, not
evidence for the still absent store Emplace or payload Span APIs. No R7n2
Emplace/Span performance comparison or NoAllocation proof can be claimed.

## R7n3 closeout

Baseline: clean `733d524bdba91ed61029b8003500cc9734888735`; compiler ID
`concept-evt1-stage0-go`. The old DenseStore used
`Option<owned T><array>[Capacity]`; the old GenerationalStore used tagged
option slots with occupied flags, generations, and a LIFO free stack. R7n2
had already proved direct `Initialize(Storage<T>, T{...})` for immovable
aggregate construction and partial-field cleanup.

DenseStore now uses general `T<raw>[Capacity]` backing: generated C has a
typed `T data[Capacity]` followed by one `int count`. Exactly `[0, count)`
contains live values. `Emplace(ref store, T{...})` writes fields directly into
`data[count]`, then increments count and returns `Id<T>{count}`. A failed
initializer drops completed field temporaries and leaves count unchanged.
`Values` and `ReadOnlyValues` borrow only the live prefix as mutable or const
Span. They cannot expose the uninitialized tail; a view cannot escape the
store lifetime, and an append with a live view is rejected. Store Drop
destroys the live prefix in reverse index order. Fixed backing never
reallocates; a DenseStore with immovable elements is itself immovable.

GenerationalStore retains its generation array and deterministic LIFO free
stack. Its payload backing is general `T<sparse>[Capacity]`: a typed array
with live bits. `Emplace` constructs directly at the selected free or next
slot and commits liveness and free-list state only after success. A failure
leaves the slot, generation, `next`, and free stack unchanged. `Remove`
destroys once, clears the live bit, and advances or retires generation as
before. Sparse slots deliberately provide no contiguous live Span.

The artifact-only immovable fixture imports Standard semantic modules with
no source reparse. It executes in Normal and Verify, exercises fresh dense
and sparse slots plus sparse removal/reuse, and observes nine exact Drops
across successful and failed construction. Failed reused-slot insertion is
followed by successful reuse with the expected index and generation. Static
negatives reject by-value immovable Insert, moving a live immovable store,
view escape, mutable view from const store, and append with a live view.
The same fixture passes strict C11 with `-pedantic-errors -Wall -Wextra
-Werror -Wno-unused-function` and 100 repeated MIR/C/proof output
comparisons in both policies; the package graph test separately covers
100-run Standard artifact identity. Alignment-sensitive layout assertions prove a
typed element stride, eight-byte alignment, and matching 56-byte backing
size for three 16-byte records. Zero-capacity raw/sparse backing rejects
before C generation.

Standard facts prove `NoAllocation` for bounded append, stationary dense and
sparse Emplace, Span view, insert, and remove. An explicitly allocating field
initializer disproves `NoAllocation` through Emplace. The AST fact and Span
traversal execute without an iterator allocation. The informational five-run
Normal benchmark means on this host were: stationary Emplace plus Span
traversal 7.05 ms and plain array insertion/traversal 6.87 ms. Process and
compiler overhead dominate; no throughput threshold is claimed. Generated
C remains direct typed arrays and indexed metadata.

Final R7n3 validation: `go test ./...` passes, including both 100-run
determinism gates and checked output snapshots. `go vet ./...`, root and
legacy `zig build test`, R7d3 BurnIn, and the full EVT1 semantic corpus
manifest pass. Standard passes 35 facts and three benchmarks in Normal and
Verify. DragonGod passes 23 facts and one benchmark in Normal and Verify.
The new artifact-only stationary fixture passes Normal and Verify native
execution and strict C11 pedantic compilation. No R7p implementation was
started.
