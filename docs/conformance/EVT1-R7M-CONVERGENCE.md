# EVT1 R7m convergence log

| Blocker | Evidence | Progress / next boundary |
| --- | --- | --- |
| No intentional Verify build | `CompilationPolicy` had conservative and optimized constructors, but CLI emission and test runner always called default `Generate` | Added `VerifyCompilationPolicy` and explicit CLI/test selection through the same compiler path. |
| Bounds failure lacks observed values | Ordinary array/span checks called `concept_panic` with only a reason and source coordinates | Verify C now reports index, extent, source, and compiler-derived origin before aborting; normal C is unchanged. |
| Resource audit lacks bounded ownership | `PoolAllocator` has typed backing and occupancy bits, with no red-zone or poison metadata; test runner has no typed foreign contract observer | Next work must establish allocator-owned aligned raw-byte layout and a provenance-bearing observer before claiming red zones or foreign verification. |

## R7m2 progress

| Boundary | Evidence | Result |
| --- | --- | --- |
| Full-suite liveness | All package tests passed in 112.987s, then Go consumed CPU processing a 58,063,203-byte test log with 822,263 `stat` records. Two native 100-run tests repeatedly searched 67 Windows PATH directories and 12 PATHEXT forms. | Test-local PATH/PATHEXT narrowing resolves tools once and retains both 100-run loops. Focused lane passed in 29.390s; exact `go test ./...` passed in 170.221s before subsequent foreign changes. |
| Foreign provenance | TinyXML2 native `--verify` reports `CreateDocumentContract`, `DeclaredForeign`, companion source, call source, `NonNull`, pass. A fake NULL-returning native function reports a violation; unsupported runtime claims fail discovery. | Typed observer added without proof promotion or runtime registry. Native ABI/artifact identity remains checked first. The violation observation was stable across 100 executions. |
| Verify overhead | Bounds-heavy benchmark mean 6.574ms Normal / 6.570ms Verify; scheduler mean 6.439ms / 6.586ms. Pool fact process 32.339ms / 33.775ms; collector fact 33.803ms / 35.462ms. | Informational single-run figures; pool lacks its requested red-zone work, so its figure only measures existing Verify policy. |
| Aligned pool envelope | `int<array>[256]` is 1,024 bytes and sixteen maximum-size slots consume all of it. Semantic `SizeOf<PoolAllocator>()` is 1,056. | Real front/back guards require a coherent Verify representation and semantic geometry. Merely appending C fields would contradict `SizeOf`, `bind`, and imported layout. Red zones and poison remain blocked by this representation boundary. |

Verify bounds MIR/C generation passed 100 byte-equality runs. Normal and Verify
hardware fixture C and `.machine.S` remain byte-identical. In the pool C
comparison, Normal output was 14,648 bytes and Verify was 15,541 bytes; the
difference is the bounds helper and call sites, with no storage envelope.
`go vet ./...` passed. Standard Verify passed 29 tests. DragonGod Verify passed
23 tests and one benchmark. The full Go suite passed after the foreign
implementation in 242.010 seconds and on the committed implementation in
244.412 seconds. The final `go vet ./...` and artifact-only TinyXML2 Verify
run also passed.

## R7m3 closeout

| Boundary | Evidence | Decision |
| --- | --- | --- |
| Exact pool storage | Sixteen 64-byte payloads consume all 1,024 backing bytes; `SizeOf<PoolAllocator>() == 1056` and `AlignOf == 8` in both semantic and strict-C11 native checks. Normal and Verify generated headers match. | No hidden red-zone field or capacity reduction. Defer a separate, explicit guarded allocator policy. |
| Release poison | The typed owner destroys `Tracked{17}` before `Release`; C11 witness observes the destructor's value and then four `0xDD` raw bytes. Normal has no poison call. | `VerifyPoisonReleasedRegion(Address<SystemMemory>, usize<byte>)` lowers to a bounded `unsigned char*` loop only in Verify. Successful occupancy check precedes poisoning, then the slot becomes free. Reuse initializes 42 successfully. Direct raw-region callers must end object lifetime first. |
| Stale state | Existing pool fact detects double release, rejects foreign/interior regions, and reuses the first free slot. Owner lifetime checks reject value access after destruction; collector stale-handle facts pass under Verify. | No arbitrary dangling native-pointer claim. |
| Hardware isolation | Poison intrinsic rejects DeviceMemory; Normal/Verify MMIO C and AMD64 `.machine.S` comparison remains byte-identical. | No DeviceMemory or machine helper writes added. |
| Determinism | Existing 100-run proof JSON and semantic artifact/C/MIR gates, Verify bounds MIR/C gate, 100-run foreign violation report, and new 100-run Verify pool output gate all passed. | No profile-specific semantic artifact facts. |
| Informational overhead | Bounds mean 8.258ms Normal / 6.689ms Verify; pool fact process 33.829ms / 33.741ms; collector fact process 35.310ms / 35.159ms; scheduler mean 6.456ms / 8.551ms. | Single host measurements include launch noise; no performance target. |

Standard and DragonGod package builds passed. Standard Verify passed 29 facts,
DragonGod Verify 23 facts and one benchmark, and artifact-only TinyXML2 Verify
two facts. The strict-C11 pool poison/reuse witness and foreign/bounds negative
regressions passed. Owner access after Drop is rejected as `CV4502`; the pool
fact reports double release as an ordinary error. Pool generated C was 14,337
bytes Normal and 15,529 bytes Verify; only Verify contains the raw-byte poison
helper. The first full `go test ./...` passed in 262.509 seconds and
`go vet ./...` passed. BurnIn and the semantic corpus passed with unchanged
manifest counts (398 valid, 262 static-invalid, 13 runtime-negative,
5 compatibility, 4 expected-divergence). The final clean-HEAD full gate
follows the documentation commit.
