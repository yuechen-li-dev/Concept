# EVT1 R7m convergence log

| Blocker | Evidence | Progress / next boundary |
| --- | --- | --- |
| No intentional Verify build | `CompilationPolicy` had conservative and optimized constructors, but CLI emission and test runner always called default `Generate` | Added `VerifyCompilationPolicy` and explicit CLI/test selection through the same compiler path. |
| Bounds failure lacks observed values | Ordinary array/span checks called `concept_panic` with only a reason and source coordinates | Verify C now reports index, extent, source, and compiler-derived origin before aborting; normal C is unchanged. |
| Resource audit lacks bounded ownership | `PoolAllocator` has typed backing and occupancy bits, with no red-zone or poison metadata; test runner has no typed foreign contract observer | Next work must establish allocator-owned aligned raw-byte layout and a provenance-bearing observer before claiming red zones or foreign verification. |

The first two blockers are removed. The third is the concrete next blocker for
the full milestone. See `VERIFY-MODE.md` for the present runtime-verifiability
boundary.
