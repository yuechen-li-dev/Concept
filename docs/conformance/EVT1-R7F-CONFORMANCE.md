# EVT1 R7f collector collections: honest stop

R7f1 follow-up: [EVT1-R7F1-CONFORMANCE.md](EVT1-R7F1-CONFORMANCE.md) closes the generic-call and local-array compiler blockers and proves the value-shaped two-type-parameter concept case already worked. It adds a lexical `owned`-move borrow exclusion, but destructive operations through `ref` and suspension are not fully proved. R7f remains an honest stop; a collector remains deferred.

Baseline: `b1747d7d3ffba7510aff6484c6863f70c214cf6e` (R7e).
Compiler: `concept-evt1-stage0-go`.

R7f is not implemented. A library-only, generic collector draft was removed
after the current Concept constraints made the direct generic shape impractical
without additional general language work or a brittle fixed-type workaround.
There is no collector package, ambient GC, runtime
registry, or scheduler hook in this revision.

## What the attempted ordinary-library shape established

- `Standard.Memory.Core` provides `Allocator<A>` and
  `ReleasableAllocator<A>`; `Standard.Memory.Ownership` proves the required
  `Destroy(storage)` then `Release(region)` order.
- `MemoryRegion<SystemMemory>` can be recorded, and `bind<T>(region)` can
  recover typed storage without moving an object. A bounded slot table and
  generation counter are therefore plausible library metadata.
- A homogeneous `MarkSweepCollector<T,A,Capacity,RootCapacity>` was attempted
  with explicit roots, generation-checked handles, an allocation table, a
  bounded mark worklist, deterministic slot-order sweep, and explicit
  `TraceReferences`.

The draft could parse its basic metadata and allocation path, but it could not
compile its complete generic operations. It was not executable evidence and
was removed rather than leaving a misleading collector API.

## Exact language blockers

1. Concept concepts accept multiple *type* parameters, but do not accept a
   value parameter such as `usize Capacity`. A proposed
   `Trace<T, Capacity>` / `Collectible<T,A,Capacity>` contract fails with
   `CV4102: unknown enum or type Capacity`. A visitor type parameter may bridge
   that gap, but this still needs an executable proof across imported generic
   artifacts. A fixed-size visitor would impose an unrelated graph limit.
2. A generic collector function calling the generic `IsLive<T,A,N,R>` helper
   fails with `CV4174: templates cannot invoke templates in EVT1 M1B-B`.
   Implicit dispatch fails with `CV4176` because the helper is not guaranteed
   by the allocator constraint. Inlining each helper would duplicate validity
   and generation logic across allocate, root, read, trace, and sweep paths.
3. Generic local metadata arrays such as
   `CollectorRegion<array>[Capacity] regions = [...]` fail with
   `CV4558: runtime array CollectorRegion<array>[Capacity] requires explicit
   storage; shape does not allocate`. This can be worked around by forcing
   callers to construct every metadata array at concrete capacities, but that
   does not resolve the trace and generic-call blockers.

The compiler must also establish that no `ref T` into a collectible object
survives `Collect`. Returning only copied payloads avoids that specific hazard
but excludes noncopyable collected values and does not provide the requested
scoped borrow model. No unsafe `Value` accessor was shipped.

## Convergence decision

Value-parameterized concept constraints, constrained generic-to-generic calls,
and capacity-resolved local arrays would be general language milestones rather
than collector library policy. Inlining all generic helpers and forcing callers
to provide every metadata array would be a large fragile workaround while the
borrow-versus-collect law remains unproven. A collector-specific compiler
branch, hidden type registry, or fixed 64-slot trace visitor would violate the
R7f boundary.
The next safe step is to settle those general language contracts independently,
then implement and test the collector through ordinary Standard and DragonGod
modules. R7e remains the latest successful capability.
