# EVT2h4 qualification throughput evidence

Baseline main `408db1720fdf72381069cbc8f5a5feaa6633816a`; implementation
`10be4cb`. Compiler `concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64;
16 logical processors, approximately 31.1 GiB RAM. The historical h3 lane was
2,267.62 s. This investigation measures the actual checkout rather than treating
that older measurement as a controlled benchmark.

## Measurement and reproduction

The detached baseline runs unchanged h3 production/tests with only opt-in
observation from `679bc86`: Go test JSON and per-stage/command events. The main
checkout is used for implementation. The baseline observer patch and logs are
preserved in ignored `artifacts/evt2h4/`; the baseline checkout is restored clean.
Baseline full command:

```powershell
$env:CONCEPT_QUALIFICATION_PROFILE = '<absolute-run-directory>/commands.jsonl'
go test ./... -json -count=1 -parallel=4 -timeout=45m > tests.jsonl
go vet ./...
```

The optimized invocation uses the same full suite/count/concurrency/timeout:

```powershell
./test.ps1 -Lane Qualification -ProfileDirectory artifacts/evt2h4/after
```

Both use fresh test processes and `-count=1`. Go package/toolchain compilation
caches may be warm; no passing Go result is replayed. Baseline Go wall time is
2,208.186 s (compiler/native package 2,197.607 s, CLI 140.868 s, GPU-free Vulkan
0.490 s). Vet passed in the original baseline invocation; a separately timed
repeat is 0.622 s. The comparison's 2,208.808 s baseline is the sum of those
measured Go/vet stages, not a claim that one stopwatch included the intervening
pause. The historical 2,267.62 s is retained separately.

Go JSON records actual test-family/package elapsed time. Native-object and
Cathedral-generation events record actual stage wall time. Native command
events count started commands and ProcessState CPU time; their cleanup elapsed
includes the enclosing test lifetime and **is not process wall time**. Counts
cover the Go `nativeCommand` boundary, not every descendant, source-test runner,
CLI-internal compiler, linker or trap child. CPU times likewise cover observed
processes only. Peak qualification-tree memory is not measured. The baseline
overlaps development probes; these are observed local timings, not an idle-host
or statistically controlled throughput claim.

## Profile before optimization

| Baseline family/stage | Measured wall |
| --- | ---: |
| h3 native match execution, including full 100-run gates | 325.12 s |
| f5 native mixed calls | 181.42 s |
| g4 artifact-only planning | 157.48 s |
| h3 standalone artifact admission | 143.91 s |
| g5 native aggregate execution | 117.64 s |
| R9a innate library scale | 93.01 s |
| Batched native CLI | 89.44 s |
| g3b native references | 87.20 s |
| g3 native objects | 82.13 s |
| Shared Cathedral semantic/C production, Normal / Verify | 38.191 / 30.333 s |
| Repeated full Cathedral C object compilations | 43 cold, 304.263 s total |

These are overlapping family/stage costs and must not be summed as a critical
path. The C object profile demonstrates redundant setup: unrelated oracle
headers and harness-only macros changed keys for otherwise identical backend
translation units. Many independent native families also ran serially despite
already isolated paths. Execution of fresh match plans/emissions remains real
work and is retained.

## Implemented changes

`3c5b6eb` replaces the overly broad all-headers key with a conservative literal
include closure, recursively including generated headers that shadow system
headers. Compiler bytes/path, host target, exact flags, source path and included
source/header bytes participate. Used-header changes miss; unrelated-header
changes hit. Ambiguous filenames, missing quoted includes, macro/unknown
directives, continuations or unsupported external-include flags disable reuse.
Toolchain/system headers are immutable prerequisites within one invocation;
nothing persists across test processes. Unit and actual compile/link/execute
probes verify invalidation and value changes. Normal/Verify outputs can share
objects only if all relevant emitted bytes and flags coincide.

Harness-only selection macros move into their C hosts, preserving independent
native-only and domain-oracle execution without changing backend compiler flags.
`3c5b6eb` and `5ba39d8` schedule audited independent native-process tests through
the existing four-slot Go scheduler. Every case retains independent temp files,
input artifacts, mutable state, diagnostics, output images and executable paths.
Shared semantic outputs are immutable clones; object build population is locked.
Environment-mutating tests remain serial. No worker service or persistent cache
is created. Profile/report support is opt-in and test-only.

The CLI gets one additional Option case in the existing batch; its single cold
smoke remains. Both CLI backend builds are retained. Fresh generation, physical
allocation, admission, native encoding, fixups, debug descriptions, traps, C11
oracles and MSVC/clang-cl probes are never replaced with reused results.

## Before / after

The optimized full lane passes in **1,319.57 s** (21.99 minutes), including vet
and profile reporting. Compared with the measured baseline stage sum, this is
**1.674x**, or **40.26% less wall time**, despite new Option coverage. The
sub-20-minute stretch goal was not reached. The optimized Go stage is
1,317.301 s; vet is 1.400 s; the recorded pre-report script wall is 1,318.716 s.

| Metric | Baseline | Optimized |
| --- | ---: | ---: |
| Full Go + vet (optimized includes profile reporting) | 2,208.808 s | 1,319.57 s |
| Full compiler/native package | 2,197.607 s | 1,310.377 s |
| Shared full Cathedral semantic/C generations | 2 | 2 |
| Cathedral C object cold compilations | 43 | 12 |
| All observed reusable C objects, cold / hits | 46 / 59 | 17 / 97 |
| Observed nativeCommand launches | 1,976 | 1,965 |
| Observed GCC / Clang compiler launches | 837 / 135 | 816 / 135 |
| Explicit oracle/parity-named host compilations in observed commands | 16 | 18 |
| Top-level passed / skipped test families | 922 / 14 | 935 / 10 |
| Runtime + leaf backend facts per mode | 28 | 30 |
| Native CLI cases / cold backend builds | 12 / 2 | 13 / 2 |
| Existing fresh determinism trials | 100 per established gate | retained |
| New Option source/CFG/transport, physical plans, native emissions | absent | 100 each per qualified gate |
| New MSVC / clang-cl Option DLL probes | absent | 2 |
| Peak process-tree memory | unmeasured | unmeasured |

Compiler launch counts include syntax/probe builds as well as oracles; the
oracle/parity filename count is an explicitly named subset, not all semantic
oracles. Verification counts, family coverage and skips are reconciled after
the run rather than inferred from fewer launches. All malformed/sentinel and
Normal/Verify gates remain required. The sub-20-minute goal is a stretch goal;
no expensive semantic execution is removed to satisfy it.

The full backend cold C object stage falls from 304.263 s across 43 builds to
69.784 s across 12 builds. Shared semantic/C generation is deliberately still
two independent modes, measured 36.060 / 34.501 s after the change. Observed
native-command CPU rises from 1,097.344 to 1,201.484 s with expanded execution;
the wall reduction does not claim less qualified execution. Native CLI package
time is 143.971 s, with the batch at 96.64 s including the extra Option case;
no CLI speedup is claimed here. GPU-free Vulkan is 0.443 s. Go build/dispatch
outside the longest-running package is an upper-bound residual of 10.579 s
before / 6.924 s after; it is not an isolated compiler benchmark.

There are nine new top-level Go families and four previously skipped families
that execute in main: the baseline detached checkout lacks the TinyXML
submodule and adjacent Oct fixtures, whereas main has those prerequisites.
This increases the optimized run's verification work; it does not explain the
gain by omitting baseline coverage. All eight MSVC-environment legacy C11 skips
are additionally executed successfully in a developer environment (1.470 s
package). The only remaining intentional skips are the unrelated opt-in R7fR
collector burn-in and the subprocess-only worker in its ordinary parent role;
the existing h3 subprocess determinism driver executes its worker separately.
Frozen root/legacy Zig suites are not run because those paths are unchanged.

Quick passed in 24.31 s while baseline qualification ran, with the two new
facts. The final quiet-host Quick run passes in **18.21 s**, preserving the
approximately 18-second development loop. This is a different coverage lane;
it is not full-qualification evidence.

## Remaining bottlenecks and supplemental gates

The largest optimized families are h3 native match at 278.95 s, f5 mixed calls
at 181.37 s and standalone h3 artifact admission at 144.82 s. The mixed-call
family's elapsed time barely changes (181.42 s before); it now overlaps other
isolated families. Standalone validator builds retain their independent leaf
compilation route. Native match retains all actual 100-run work. These measured
costs remain; this milestone does not optimize compiler algorithms or weaken
oracles to chase the stretch target.

Full regression includes earlier native/ABI/frozen-schema tests, corpus, Golden,
package generation and GPU-free Vulkan; vet passes. Supplemental Standard
Normal/Verify runs pass 44 tests plus three benchmarks in 41.099 / 41.602 s;
DragonGod passes 23 tests plus one benchmark in 5.711 / 5.721 s. These CLI
executions run after the measured full lane and concurrently with focused race;
they are additional coverage, not part of the before/after stopwatch comparison.
Focused race passes all selected h4/cache/runtime/policy tests, retaining full
native repetition counts: 481.521 s package / 494.193 s wall. This is measured
separately, not silently added to either full-lane comparison. Format, lint,
generated inspection, build and policy checks pass; exact commands and final
Quick timing are recorded in convergence.
Raw local logs are ignored under `artifacts/evt2h4/`; committed records preserve
exact outcomes, identities, counts and reproduction commands.
