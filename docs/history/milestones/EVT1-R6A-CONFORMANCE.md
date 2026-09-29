# EVT1 R6a first-class test tooling

Status: success

## Baseline and boundary

- Starting HEAD: `720b6dd747a3678ce9a8cdb78be0c6e4d7b5e854`
- Starting worktree: clean
- Compiler: `concept-evt1-stage0-go`
- Go: `go1.27.0 windows/amd64`
- Zig: `0.16.0`
- R5k freeze: current and preserved

R6a adds tooling metadata, discovery, a static harness, and execution. It does
not add a test DSL, test classes, reflection, a runtime registry, a scheduler,
or a second evaluator. One frozen-core bug was found by the new real path: the
failure-type visitor omitted conditional expressions, so native `Result`
constructors could be absent. The visitor now traverses `IfExpr`; no failure
law changed.

## Executable surface

`.concept_test` is canonical; `.con_test` is ignored by active discovery and
retained only as historical PoC3 evidence. Attributes are source-ordered AST
and `TestDeclaration` MIR metadata. Signature and combination diagnostics
cover facts, theories, benchmarks, prophecies, foretold, artifacts, duplicate
identity, and unknown attributes.

The extension audit found 77 `.con_test` files: 72 under the retired Phase 11
corpus and five PoC3-era examples/corpus specimens. None participates in the
active Go runner, so no compatibility alias was retained.

The canonical corpus contains 16 declarations: 11 facts (including callback,
dyn+async, automata, tensor, Result, and async composition), one three-row JSON
theory, one lightweight benchmark, and three prophecies. Its 18 executions
produce 14 PASS, three PROPHECY_FULFILLED, one BENCHMARK, and zero failures.
The static-invalid corpus has 13 files, including mandatory missing/blank
assertion reasons. Four runtime-negative files prove structured assertion
failure, unexpected panic, normal-return prophecy failure, and foretold abort.

All six `Assert.*` operations require a non-empty literal reason. Generated C
captures operands into source-ordered temporaries before comparison.
`Assert.Error` and `Assert.LGTM` are Result-only and their side-effect counters
prove exactly one evaluation. Failure records recover assertion kind, reason,
source line, actual, and expected values where applicable.

## Runner evidence

Discovery sorts normalized source paths, preserves function declaration order,
rejects escaping/missing artifacts, hashes artifacts, and emits byte-identical
`concept-test-manifest.v1` data across 100 runs. JSON theory rows bind by
position without implicit conversions. Async facts drive the existing Step/
Complete substrate without a scheduler.

Each static harness is compiled through Generic C11 and executed in a bounded
child process. This isolates every test and proves the host survives the
controlled native abort used by prophecy. Foretold evidence includes the last
checkpoint, exact stderr panic reason, artifact metadata, exit code, timing,
and build/compiler/target identity. Human output is concise; structured
`concept-test-results.v1` is written beside the deterministic manifest.

## Deferred tooling

CSV and richer theory providers, named/object row binding, exact prophecy
termination matching, parallel execution, fixture lifecycle hooks, property/
snapshot testing, richer structural diffs, and rigorous benchmark statistics
remain deferred to a later R6 milestone. R6b is not started here.
