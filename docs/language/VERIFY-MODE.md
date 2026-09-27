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

Verification failures identify the violated runtime observation and the
semantic origin of the claim being checked. Array and span bounds reports have
compiler-derived origin. A `foreign concept` may declare
`requires compiler.NonNull(result);` for a pointer-returning extern function.
On a Verify call, generated C checks the returned pointer once and emits a
`DeclaredForeign` observation with the companion declaration source and the
observed call site. A null result fails that execution. This is a typed,
per-operation checker; it does not use runtime reflection.

`[[verify_foreign("ContractName")]]` on a `.concept_test` fact requests that
observation. The runner fails if the call was not observed or if the selected
contract has no runtime checker. Normal mode does not emit the observer.
Passing observations remain empirical test evidence and do not change proof
status. Native companion tests still perform the existing ABI/artifact identity
check before test execution.

Verification instrumentation must not add, remove, duplicate, or reorder MMIO
or machine operations. The present Verify lowering adds no instrumentation to
those paths; their equivalence still needs a dedicated regression gate.

## Current verifiability map

| Claim or operation | Runtime status | Current checker |
| --- | --- | --- |
| Dynamic array and span index | Runtime verifiable for this access | Verify bounds report |
| Statically invalid index | Compile-time only | Existing compiler diagnostic |
| C ABI layout, generic satisfaction | Compile-time/build-time only | Existing semantic or ABI probe |
| Pointer result of a declared foreign contract | Runtime verifiable for an observed call | Typed `NonNull(result)` observer |
| Universal `NoAllocation`, `ExactlyOnce`, single writer | Partially verifiable in selected executions | No R7m observer yet |
| Opaque foreign memory and arbitrary dangling pointers | Not runtime verifiable by this mode | None |

The mode does not yet poison storage or add allocator red zones. The current
`PoolAllocator` has 1,024 backing bytes, all of which are usable as sixteen
64-byte slots. Its semantic `SizeOf` and generated C representation agree on
that layout. An envelope therefore needs an explicit representation and layout
change in Verify; reusing a payload byte as a guard or silently enlarging the C
struct would be unsound.
