# Verify mode

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
or machine operations. The MMIO and AMD64 machine fixtures compare Normal and
Verify C and helper output directly; Verify adds no instrumentation there.

## Current verifiability map

| Claim or operation | Runtime status | Current checker |
| --- | --- | --- |
| Dynamic array and span index | Runtime verifiable for this access | Verify bounds report |
| Statically invalid index | Compile-time only | Existing compiler diagnostic |
| C ABI layout, generic satisfaction | Compile-time/build-time only | Existing semantic or ABI probe |
| Pointer result of a declared foreign contract | Runtime verifiable for an observed call | Typed `NonNull(result)` observer |
| Released `PoolAllocator` slot | Verify poisons raw payload bytes after a valid release | `0xDD` byte writes through `unsigned char*`; existing occupancy rejects double release |
| Released owner or collector handle | Mediated by existing ownership/handle state | Static owner-lifetime rejection and collector stale-handle checks |
| Universal `NoAllocation`, `ExactlyOnce`, single writer | Partially verifiable in selected executions | No R7m observer yet |
| Opaque foreign memory and arbitrary dangling pointers | Not runtime verifiable by this mode | None |

`PoolAllocator.Release` calls `VerifyPoisonReleasedRegion` only after validating
the exact slot and live occupancy. The ordinary typed owner calls
`Destroy(storage)` before `Release(region)`, so the object's lifetime has ended
when Verify writes the deterministic `0xDD` pattern through raw byte access.
Normal lowers the intrinsic to a no-op. The pattern is an implementation detail;
it does not change program results or provide arbitrary dangling-pointer
detection. Direct low-level `Release` callers remain responsible for ending any
object lifetime first.

Verify mode cannot invent storage that the semantic type does not own. Physical
red zones require an allocator representation that explicitly reserves them.
The exact inline pool has 1,024 usable backing bytes for sixteen 64-byte slots,
so no front/back guards exist in it. A separate guarded allocator policy is
deferred; Verify never changes `SizeOf<PoolAllocator>()` or its generated C
representation.
