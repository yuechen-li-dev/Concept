# EVT1 R6n conformance: Standard.Memory generic-contract stop

Status: honest stop

Baseline HEAD: `7baabc41d60ff8a890dfed89ef333b028f0eb194`

Compiler identity: `concept-evt1-stage0-go`

R6m commit: `7baabc4 Complete EVT1 structural generic substitution`

## Baseline

The starting worktree was clean. `go test ./...`, `go vet ./...`, root
`zig build test`, and `legacy/poc3-zig` `zig build test` passed with Go 1.27.0
and Zig 0.16.0.

The active semantic-library roots remain
`language/evt1/tooling/modules/Standard` and
`language/evt1/tooling/modules/Platform`. `Standard.MemoryGeometry` supplies
ordinary region geometry and `Platform.Host.Memory` supplies the R6j owned
foreign allocation, declared-foreign region authority, and exactly-once host
release.

## Prerequisite checklist

| prerequisite | R6n result |
|---|---|
| generic runtime types | resolved |
| `SizeOf<T>` / `AlignOf<T>` | resolved |
| generic ownership/provenance/Drop | resolved |
| semantic modules | resolved |
| `extern "C"` | resolved |
| `Allocates` | resolved |
| quantities/address/storage | resolved |
| semantic fact transport | resolved |
| trusted external storage authority | resolved |
| initialized typed-storage owner | resolved |
| structural generic substitution | resolved |

## Newly isolated general blocker

The required typed allocation function has `T` first for the
`Allocate<T>(...)` API and must constrain its second type parameter:

```concept
template <typename T, typename TAllocator>
requires Allocator<TAllocator>
Result<Allocation<T, TAllocator>, AllocationError> Allocate(...);
```

The current generic validator accepts one named one-parameter concept only
when it applies to the template's first type parameter. The declaration above
therefore fails with:

```text
CV4170: template constraint Provider must apply to template parameter TValue
```

An unconstrained body cannot use the concept operation speculatively; its open
call fails with:

```text
CV4027: unknown function Provide
```

The identical `CV4170` occurs for a generic typed owner's `Drop` when it tries
to constrain `TAllocator` with the release capability. These minimal cases are
pinned in `internal/concept/r6n_allocator_feasibility_test.go`; the first two
use neutral `Provider`/`Provide` names to demonstrate that the gap is not
allocator-name behavior.

Reordering the type parameters would replace the required
`Allocation<T, A>` and `Allocate<T>` surface. Concrete policy-specific owner
and Drop overloads would require manual specialization and would not establish
the requested ordinary capability model. Either workaround is explicitly out
of scope for R6n.

## Result

No `Standard.Memory` source, placeholder allocator contract, allocator policy,
typed owner, fixture, or documentation claiming a usable API was added. A
successful implementation first needs a general extension that permits a
named capability constraint to target any type parameter and carries its
required-operation closure through generic instantiation and semantic-module
artifacts.

Consequently Bump, Pool, Arena, memory-source wrappers, typed allocation,
Span/tensor integration, allocator proof outcomes, hosted/firmware/DragonGod/
Prometheus specimens, and allocator determinism/performance measurements remain
deferred as one coherent framework rather than being represented by partial
contracts.

## No-cheating audit

Production compiler and Planner code contain no semantic branches for
`BumpAllocator`, `PoolAllocator`, `ArenaAllocator`, `Allocation<`,
`MonotonicAllocation`, or `Standard.Memory`. R6n adds no allocator MIR node,
Planner strategy, hidden heap, allocator or region registry, proof metadata,
global allocator, `new`/`delete`, source-level `T*`, GC, or RTTI.

The general regression tests are the only new executable code. The existing
hosted and firmware runtime scans remain unchanged because no allocator runtime
path exists to qualify honestly.

## Deferred ledger

The broader ledger remains unchanged: general source-level `T*`, MMIO and
volatile, physical pages, virtual memory, DMA, real Vulkan memory, thread-safe,
realtime, debug, guard, fallback and heap-specific allocation concepts,
variadic construction, and GC/collection.
