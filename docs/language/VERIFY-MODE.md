# Verify mode (R7m in progress)

`concept emit-c <file> --verify`, `concept mir <file> --verify`,
`concept plan <file> --verify`, and `concept test <directory> --verify`
select the explicit Verify compilation policy. The test runner records
`"verify": true` in its result file. Source syntax and semantic analysis are
unchanged. `Assert.Concept` remains a compile-time assertion.

Verify mode challenges runtime-verifiable assumptions without changing program
semantics. The current implementation retains the existing bounds and
synchronization guards and adds an observed index and extent to array and span
bounds failures. The report identifies the compiler-derived bounds obligation,
source path, line, and column, then aborts. Normal generated C has none of this
Verify-specific reporting. Existing ordinary bounds panics remain active.

A successful verification run is execution evidence, not a universal
compile-time proof. Runtime results never upgrade `Declared` or
`DeclaredForeign` facts to `Proven`.

Verification failures identify both the violated runtime observation and the
semantic origin of the claim being checked. At present this is implemented for
compiler-derived array and span bounds checks. Declared and foreign contract
origins do not yet have runtime checkers.

Verification instrumentation must not add, remove, duplicate, or reorder MMIO
or machine operations. The present Verify lowering adds no instrumentation to
those paths; their equivalence still needs a dedicated regression gate.

## Current verifiability map

| Claim or operation | Runtime status | Current checker |
| --- | --- | --- |
| Dynamic array and span index | Runtime verifiable for this access | Verify bounds report |
| Statically invalid index | Compile-time only | Existing compiler diagnostic |
| C ABI layout, generic satisfaction | Compile-time/build-time only | Existing semantic or ABI probe |
| Universal `NoAllocation`, `ExactlyOnce`, single writer | Partially verifiable in selected executions | No R7m observer yet |
| Opaque foreign memory and arbitrary dangling pointers | Not runtime verifiable by this mode | None |

The mode does not yet poison storage, add allocator red zones, observe Concept
allocations, or check declared foreign contracts. Those require bounded
resource-owned metadata and provenance before they can be reported honestly.
