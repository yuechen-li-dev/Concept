# Concept-owned iteration workflow, 2026-10-29

The default development command is `./test.ps1`. It executes the ordinary
Concept test framework in Normal and Verify for runtime and leaf backend facts,
then checks the CLI frontend and workflow policies. Full native qualification
is explicit: `./test.ps1 -Lane Qualification`. See [TESTING.md](TESTING.md).

EVT2h3 adds CMIRAMD9 artifact/CFG admission and 100 complete native match
planning/emission gates. Qualification now permits 45 minutes per Go package;
the larger native determinism host has its own eight-minute child deadline.
Quick remains unchanged. This extends the execution budget, preserving every
gate and repetition count; historical timings below describe their checkpoints.

## Scope and provenance

The starting point was EVT2h1 closeout `c250308`, following native enum
qualification at `0b8abf9`. The existing backend-session and multi-input CLI
changes were included in intervening commit `77336be`; they were preserved.
Commit `596f8ae` adds the migration, batch CLI qualification, lane commands and
ownership checks. No EVT2h2 work or language semantics changed.
Commit `b426da1` retains the five pre-existing strict C11 syntax oracles after
the runtime assertions move; both GCC and Clang pass those checks in 1.421 s.

The initial syntax-based audit found 522 of 912 Go test functions containing
Concept-looking literals, across 210 Go test files. Across test files it found
1,021 such literals and 6,054 source lines. These are heuristic inventory counts,
not a claim that 522 complete tests can move: diagnostics, compiler structures,
artifact mutation and independent native/ABI oracles require an external host.

This bounded migration adds thirteen language facts in nine authoritative
`tests/runtime/*.concept_test` modules and four Win64 ABI facts in
`tests/backend/win64_abi.concept_test`. Together with the existing six language
facts, Quick executes 23 facts per mode, 46 executions total. Coverage includes
contextual arrays and floating literals, enum storage/equality, error remapping,
integer matches, ranges, value snapshots, shared span backing, binding names,
Win64 argument positions, shadow space, register tables and bounded rejection.

Go compiler-inspection tests read those fixtures. Their duplicate C runtime
expectations were removed. Foreign aliasing into record fields, descriptor
mutation, C move replay and machine-code execution remain independent probes.
The Concept runner uses `-std=c11 -Wall -Wextra`; the retained syntax oracles
add `-pedantic-errors` independently. Moving an expectation into Concept does
not imply that every runner invocation is a strict-C11 conformance oracle.
Payloads, matches and fallible types in migrated language facts exercise
existing source-to-C support; they add no Cathedral native support for those
forms. The new backend component facts also execute through the C bootstrap.

## Reuse and regression protection

The multi-input `concept amd64` command checks every source before compiling
Cathedral once. Nine existing native CLI scenarios now share that invocation.
Each artifact still runs in a fresh backend process with independent admission,
planning and emission. One separate cold single-input smoke retains coverage of
that entrypoint. There are two backend builds instead of ten in CLI qualification.
No persistent cache, cached test result or new Go native algorithm is involved.

Policy tests require migrated compiler inspectors to use the authoritative
fixtures, reject duplicated runtime harness calls there, reject new cold native
CLI scenarios outside the batch, and prevent whole-Cathedral dependencies in
the quick specifications. AGENTS.md records these ownership rules.

The explicit Compiler lane uses `-short`; native harnesses and Golden execution
report qualification skips there. The full suite preserves their execution and
the existing full determinism counts. Quick is a focused behavior loop and does
not substitute for compiler conformance or native qualification.

## Measurements and validation

Windows AMD64, `concept-evt1-stage0-go`, local GCC/Clang bootstrap toolchain.
Uncached Go test execution (`-count=1`); compilation/tool caches may be warm.
These are local wall times, not portable thresholds or equivalent-coverage
comparisons between different lanes.

| Check | Measured result |
| --- | ---: |
| Default Quick, before final commit | 17.82 s |
| Quick at `596f8ae`, with qualification running | 20.41 s |
| Quick at `b426da1`, after qualification | 18.01 s |
| One backend argument-planning fact, Verify via CLI | 2.91 s |
| Broad `go test -short ./...` during development | 320.55 s wall |
| CLI qualification before batching, EVT2h1 baseline | 297.737 s package |
| CLI qualification after batching, focused run | 59.955 s package / 60.70 s wall |
| CLI qualification in final full run | 63.136 s package |
| Focused migrated specifications/compiler checks | 14.699 s package |
| Focused race: facts, policies and runner build/process reuse | 32.403 s package |
| Full compiler/native qualification | 1,350.479 s compiler package / 1,352.38 s wall |

The CLI comparison preserves the native scenarios and inspection expectations;
the quick/full comparison selects different responsibilities. Broad `-short`
alone did not solve iteration speed and is intentionally not the default.

Full qualification passed: `go test ./... -count=1 -parallel=4 -timeout=35m`
at `596f8ae`, including frozen native regressions, EVT2h1 native parity and
exact-byte checks, full determinism counts, corpus, Golden, Vulkan and
Standard/DragonGod package gates. The later `b426da1` changes only the five
restored syntax-check calls; those five tests passed separately on that commit.
`go vet ./...` also passed on `b426da1`. The full compiler suite remains costly;
this work improves the default iteration path and reduces repeated CLI backend
builds, without claiming a comparable whole-suite speedup.

Focused race checks passed for the runtime/backend specifications, workflow
policy, module-build reuse/process isolation and native-input invalidation.
Every migrated fixture passed formatting checks; the nine migrated language
fixtures also passed CLI lint. The backend fixture was checked and executed
with its explicit `libraries` module root through the Concept framework.
Local execution logs are under `artifacts/test-workflow/`. Frozen Zig suites are
skipped because no legacy Zig compiler or build/test infrastructure changed.

Remaining migration is incremental: move source-level expectations to the owning
Concept module, retain compiler-boundary and independent oracle checks, and use
leaf backend imports. This is not a wholesale conversion of the audit inventory.
