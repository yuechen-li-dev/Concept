# R9a innate evaluator measurements

Measured on Windows/AMD64, Go 1.27.0, with the R9a CV4138 migration active.
`go test ./internal/concept -run '^TestR9aInnateLibraryScale$|^TestR9aComptimeUsage' -count=1 -v`
uses the ordinary parser, artifact composition, materialization, namespace resolver
and semantic analyzer. Source dependencies are compiled to artifacts before timing.
Every measured validation owns its environment; the innate environment and mutex
remain shared. Clock reads and usage collection are opt-in internal analysis options.
This measures production `.concept` modules, excluding package manifests and test
units; it does not time native builds or GPU execution.

| Workload | Modules | Predicate evaluations | Validation wall | Sum of time holding lock | Peak fuel | Peak stack depth | Peak loop bound | Peak array literal |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Standard | 22 | 380 | 63.288ms | 1.002ms | 28 | 3 | 0 | 0 |
| DragonGod | 22 | 237 | 84.125ms | 3.541ms | 28 | 3 | 0 | 0 |
| Golden | 31 | 593 | 102.782ms | 4.538ms | 221 | 7 | 3 | 0 |
| Vulkan | 1 | 194 | 4.461ms | 1.646ms | 430 | 8 | 8 | 0 |
| All, four repetitions, one worker | 304 | 5616 | 993.744ms | 31.021ms | 430 | 8 | 8 | 0 |
| Same work, four workers | 304 | 5616 | 388.935ms | 47.438ms | 430 | 8 | 8 | 0 |

The four-worker run recorded 3.109ms summed mutex wait; serial waits rounded to
zero. These are one host/run's observations, not performance guarantees. Small
durations can round to zero on this clock, and summed worker durations are not
wall time. On this workload the lock does not materially serialize compilation:
the same work finishes about 2.55 times faster with four workers. Keep the lock.
The shared innate environment contains mutable generic/comptime analysis caches;
removing synchronization would need a separate ownership/isolation change.

The largest measured sources were Standard.Backend.AMD64 (52,053 bytes), Vulkan
(25,158), Standard.Octagon.Core (15,200), DragonGod.Scheduling.Parallel (11,964),
and Standard.Collection.Core (9,658). Source size is not evaluator complexity:
most function bodies are validated rather than executed by innate predicates.

Configured limits remain fuel 4096, loop bound 256, stack/call depth 32, ordinary
comptime fixed-array extent 64. The stack limit includes loop frames. No measured
predicate constructs array literals; the instrumentation test constructs three
elements and iterates them to qualify both counters. Storage/reflection array
extents are separate from arrays created during evaluator execution.

Four identical passes create repeated predicate/subject candidates, but that is
not a safe cross-compilation Verdict cache: results and messages depend on parent,
ownership, attributes, dependency identities and source sites. No cache is added.
Fuel and depth diagnostics now state current/configured values and the evaluation
stack; loop and array diagnostics already state both values and carry source sites.
Innate failures additionally identify the concept, predicate and judged declaration.

## Final-source race qualification

The earlier table is the non-race measurement captured before the workspace MVP.
On the final source, the same scale/limit tests also passed under `go test -race`:
76 modules; Standard has 387 predicate evaluations (seven additional declarations)
and the other library counts remain 237/593/194. Equal repeated workloads contain
304 validations and 5644 evaluations. With race instrumentation, serial wall was
8.929s and four-worker wall 2.685s, with summed lock wait 6.123ms in the parallel
run. Largest backend source is now 53,437 bytes. Peaks remain fuel 430, depth 8,
loop 8 and array 0. Instrumented timings are a separate workload and must not be
compared directly with the original non-race table. No race was reported.
