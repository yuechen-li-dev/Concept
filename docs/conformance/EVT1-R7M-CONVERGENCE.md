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
