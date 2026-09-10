# EVT1 R6k conformance: Standard.Memory feasibility stop

Status: honest stop

Historical note: R6l subsequently resolved the first three direct boundary
tests and replaced the fourth self-referential shape with `Value(storage)`.
This report records the R6k baseline; current status is in
`EVT1-R6L-CONFORMANCE.md`.

Baseline HEAD: `96249a7254bc22915bbe7be3ba45f675b0713551`

Compiler identity: `concept-evt1-stage0-go`

R6j commit: `96249a7 Add EVT1 native semantic interop contracts`

## Baseline

The starting worktree was clean. `go test ./...`, `go vet ./...`, root
`zig build test`, and `legacy/poc3-zig` `zig build test` passed. The toolchain
was Go 1.27.0 and Zig 0.16.0.

The ordinary module roots remain
`language/evt1/tooling/modules/Standard` and
`language/evt1/tooling/modules/Platform`. `Standard.MemoryGeometry` owns
`MemoryRegion<SystemMemory>`, `AlignUp`, and `Subregion`;
`Platform.Host.Memory` owns the R6j declared-foreign host allocation owner,
`RegionOf`, and exactly-once host release.

## Blocker checklist

| prerequisite | status |
|---|---|
| generic runtime types | resolved |
| `SizeOf<T>` / `AlignOf<T>` | resolved |
| generic ownership/provenance/Drop | unresolved for initialized typed-storage owners |
| semantic modules | resolved |
| `extern "C"` | resolved |
| `Allocates` | resolved |
| quantities/address/storage | unresolved for open-template `bind<T>` |
| semantic fact transport | resolved for facts, not operational storage state in owner fields |
| trusted external storage authority | resolved |

The required gate says to stop if any item is not genuinely resolved. R6k
therefore did not add `Standard.Memory`, allocator policies, typed-owner
placeholders, fixtures that bypass the real path, or compiler name checks.

## Reproduced general gaps

`internal/concept/r6k_allocator_feasibility_test.go` pins four minimal programs
which contain no allocator names:

1. `bind<T>` inside an ordinary open template rejects with `CV4148`.
2. `Destroy(owner.storage)` rejects with `CV4027`.
3. `move owner.storage` rejects with `CV4507`.
4. returning the initialized `ref T` with its storage owner rejects with
   `CV4521` rather than preserving the backing parameter's provenance.

Together these prevent the required mechanical sequence
`region -> Storage<T> -> Initialize -> Allocation<T>` and prevent an owner Drop
from expressing `Destroy(T) -> Release(region)`. Implementing around them would
require allocator-specific compiler behavior, runtime state metadata, a hidden
copy, or a false owner contract.

## Audit result

No allocator implementation was added. Compiler and Planner searches contain
no semantic branch for `BumpAllocator`, `PoolAllocator`, `ArenaAllocator`,
`Allocation<`, or `Standard.Memory`. Existing allocator wording occurs only in
tests and documentation. No MIR node, runtime registry, heap, global allocator,
`new`/`delete`, source-level `T*`, or backend allocation path was introduced.

## Deferred

All requested R6k allocator capabilities remain deferred together: memory
sources, bump/pool/arena policies, typed allocation owners, Span/tensor
projections, allocator proof fixtures, hosted/firmware/DragonGod/Prometheus
specimens, and allocator determinism/performance evidence. The broader deferred
ledger remains unchanged: MMIO/volatile, physical/virtual/DMA/Vulkan memory,
thread-safe/debug/fallback allocators, variadic construction, GC, and stronger
allocation-effect concepts.

The next milestone should be chosen after reviewing this result. It must be a
general initialized typed-storage ownership milestone, not an allocator
implementation disguised as compiler support.
