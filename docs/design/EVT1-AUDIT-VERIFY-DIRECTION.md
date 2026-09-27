# EVT1 audit and Verify direction

The shared semantic pipeline remains authoritative. Verify is a compilation
policy passed to the existing MIR, Planner, and strict-C11 lowerer. It preserves
runtime guards and disables synchronization simplification. Its first
instrumentation is local to array and span bounds checks: the same guard now
reports the observed index, extent, source location, and compiler-derived
obligation before aborting.

The next engineering boundary is allocator-owned raw storage. `PoolAllocator`
currently stores 256 bytes through an `int<array>[256]` backing and tracks 16
occupied slots; neither slot red zones nor raw-byte lifetime state exist.
Adding them requires an explicit aligned slot layout and metadata owned by the
pool, without changing requested alignment or normal allocation behavior.
Generated C currently has no general verifier hook at allocator boundaries.
Foreign contracts also need a typed runtime checker associated with the
declared fact's provenance; the current test runner only observes process
success/failure. Passing executions cannot become universal proofs.

No global shadow memory, runtime reflection registry, or machine/MMIO
instrumentation is part of this direction.
