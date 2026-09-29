# EVT1 R7m conformance

Baseline: `84590c423ce5110c34ea3f4068de79261f018a70`, compiler
`concept-evt1-stage0-go`. The R7m commits are `1974db1`, `01892c9`, and
`84590c4`.

Verify remains an explicit policy of the ordinary semantic pipeline. It is
visible through `emit-c`, `mir`, `plan`, and `.concept_test`; the native test
command also accepts `--verify`. `Assert.Concept` stays compile-time. The
strict C11 bounds negative reports observed index, extent, source, and
compiler-derived origin. The Normal and Verify hardware fixture outputs are
byte-identical for MMIO C and AMD64 `.machine.S`.

The typed foreign observer checks `compiler.NonNull(result)` on an observed
extern call. TinyXML2's `CreateDocumentContract` exercises the artifact-only
native path. Its test result records `DeclaredForeign`, declaration source,
call site, `NonNull`, and pass. A controlled false native implementation
produces a runtime violation with the same provenance; an unsupported contract
is rejected. Neither result changes static proof status. The R7j1 native
artifact identity gate remains before native test execution.

The earlier `go test ./...` apparent hang was Go test cache processing after
all tests had passed. A preserved 58,063,203-byte test log contained 822,263
`stat` entries from repeated Windows PATH/PATHEXT tool searches in two native
100-run determinism tests. The test-local PATH narrowing in `7563d89` retains
their probes and 100 iterations. The exact full command then passed in
170.221 seconds. After the foreign implementation it passed again in 242.010
seconds and on the committed implementation in 244.412 seconds.

The outstanding architectural boundary is PoolAllocator's storage geometry:
1,024 backing bytes are fully usable at maximum capacity, while semantic
`SizeOf<PoolAllocator>()` is 1,056. Neither red zones nor release poison is
implemented. A Verify-only C field without corresponding semantic layout
would make `SizeOf`, `bind`, and artifact consumers unsound. R7m2 cannot claim
success until the aligned raw-byte envelope is represented consistently.

The bounds fixture's Verify MIR/C generation was repeated 100 times without
output drift. The foreign violation was executed 100 times with identical
structured observations. Informational Normal/Verify timings and the final
gates are recorded in the convergence log.

## R7m3 closeout

R7m3 starts from clean `516fc53ea65b3f81b698f56d558252bf30663a24`;
compiler ID remains `concept-evt1-stage0-go`. Its predecessors are R7m
`1974db1`, `01892c9`, `84590c4` and R7m2 `7563d89`, `ddf121d`, `516fc53`.

The exact inline `PoolAllocator` remains 1,056 bytes, aligned to eight bytes,
with 1,024 backing bytes and sixteen occupied flags. Verify now poisons a
validated released slot with `0xDD` through an `unsigned char*` byte loop.
`Allocation<T, PoolAllocator>.Drop` destroys `T` before Release; Normal emits
no byte write. A strict-C11 native witness observes the destructor's live value,
then the poisoned bytes, then successful deterministic slot reuse and new
initialization. The generated header, semantic `SizeOf`/`AlignOf`, capacity,
and ordinary slot offsets remain shared by Normal and Verify. DeviceMemory is
rejected by the poison intrinsic.

Physical red zones are explicitly deferred. Adding them to this exact pool
would consume usable payload or invalidate semantic layout. A separate guarded
allocator is possible only with its own declared storage representation; it
would be new library policy, not hidden Verify instrumentation.

The foreign violation and bounds negative remain green. Standard Verify passes
29 facts, DragonGod Verify passes 23 facts and one benchmark, and artifact-only
TinyXML2 Verify passes two facts. Collector facts are included in Standard.
MMIO and AMD64 C/helper equality is covered by the hardware regression.
The proof/explain JSON, semantic artifact/C/MIR, Verify C/MIR, foreign report,
and Verify pool output each passed their 100-run byte/structure stability gates.
The first R7m3 full `go test ./...` gate passed in 262.509 seconds;
`go vet ./...` passed. BurnIn and the semantic corpus passed, with manifest
counts 398 valid, 262 static-invalid, 13 runtime-negative, 5 compatibility,
and 4 expected-divergence. The final clean-HEAD gate is reported with the
closeout result.
