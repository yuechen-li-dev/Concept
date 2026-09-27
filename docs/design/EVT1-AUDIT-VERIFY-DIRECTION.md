# EVT1 audit and Verify direction

The shared semantic pipeline remains authoritative. Verify is a compilation
policy passed to the existing MIR, Planner, and strict-C11 lowerer. It preserves
runtime guards and disables synchronization simplification. Its first
instrumentation is local to array and span bounds checks: the same guard now
reports the observed index, extent, source location, and compiler-derived
obligation before aborting.

The allocator closeout uses existing allocator-owned raw storage.
`PoolAllocator` has an `int<array>[256]` backing: 1,024 raw bytes, entirely
usable as sixteen 64-byte slots. It tracks occupancy with sixteen booleans.
There is no free byte for a front or back guard at maximum capacity. Verify
cannot silently enlarge the generated C struct because the ordinary semantic
layout engine folds `SizeOf<PoolAllocator>()` to 1,056 bytes and that value can
be used by `bind<T>` and imported artifact consumers. Verify mode cannot invent
storage that the semantic type does not own. Physical red zones require an
allocator representation that explicitly reserves them. A separate guarded
pool is deferred rather than adding an allocator solely for this closeout.

Release poisoning is valid because the object's lifetime has ended and the
slot has returned to raw storage. The typed owner performs `Destroy` before
`Release`; the pool then writes `0xDD` through `unsigned char*` on a valid
Verify release. Normal emits no poison write. Occupancy and deterministic reuse
remain ordinary pool semantics. No typed store into dead `T`, hidden layout
field, or global shadow table is involved.

The foreign boundary has a bounded typed observer for
`compiler.NonNull(result)` on pointer-returning extern operations. The test
runner records `DeclaredForeign`, companion declaration source, observed call
site, strategy, and outcome. Passing executions remain empirical evidence;
the compiler never upgrades a declared contract to a universal proof.

No global shadow memory, runtime reflection registry, or machine/MMIO
instrumentation is part of this direction.
