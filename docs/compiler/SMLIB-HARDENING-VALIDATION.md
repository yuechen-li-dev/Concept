# SMLib compiler hardening qualification

Qualification host: Windows AMD64, Go 1.27.0, Clang 22.1.3, 2026-10-15.
Compiler baseline: `68de20d`; qualified implementation: `e855765`.
The external port remains at `C:\Users\yuech\source\repos\SMlib_Concept\phase1`.
Its recorded upstream revision is `c9c5979f04cea8c38213004d4f04df0dcfb40d05`.

## Changes and review units

| Commit | Findings resolved | Principal files |
| --- | --- | --- |
| `dc7788f` | A1 local binding before callback return inference; A2 one semantic adapter collection, nested erasure, identity deduplication | `validate.go`, `callable_capture.go`, `typed_walk.go`, `types.go`, `generate.go` |
| `e0822d6` | B1 literal context and binary64 constant folding; B2 double math overloads; B3 ordinary symbol precedence; B4 safe explicit zero construction and correct named-field lowering | `smlib_numeric_context.go`, `comptime.go`, `Standard/Math.concept`, `math_primitives.go`, `value_intrinsic_resolution.go`, `aggregate_zero.go`, `generate.go` |
| `995fb63` | Preserve opaque-handle diagnostics and validate ownership-consuming contextual arguments once | `aggregate_zero.go`, `validate.go`, `typed_walk.go` |
| `52b951d` | C1 complete emit-c directory; C2 explicit consistent physical module roots and source/artifact modes; documentation and RFC | CLI options/main, `module_roots.go`, `testing_runner.go` |
| `e855765` | Prevent local erased callback environments escaping through Result or captured wrappers; order callback descriptor dependencies; allow borrowed Result forwarding | callback/provenance validation, `concept_assert.go`, `generate.go` |

Go compiler filenames above are relative to `internal/concept`, library filenames
to `libraries`. Detailed semantics are in [SMLIB-HARDENING.md](SMLIB-HARDENING.md),
the deliberately updated [language specification](../spec/CONCEPT_EVT1_LANGUAGE_SPEC.md),
and [artifact/module documentation](../tooling/SMLIB-ARTIFACTS-AND-MODULES.md).

## Regression coverage

- `smlib_callback_test.go`: local records; repeated/nested erasure including
  Result unwrap; different callbacks sharing signatures; shadowed captures;
  deterministic adapters; invalid binding, signatures, and local environment
  escapes; valid lifetime-bound parameter forwarding through Result.
- Four `language/evt1/callable/capture/valid/smlib_callback_*.concept` fixtures
  join the authoritative conformance corpus and its reconciled manifest counts.
- `smlib_numeric_test.go`: expected literal precision, mixed-precision rejection,
  ordinary intrinsic-name shadowing, named-field identity/evaluation order,
  complete safe zero construction, non-defaultable ownership/lifetime rejection,
  and exactly-once ownership validation and cleanup.
- `libraries/Standard/smlib_numerics.concept_test`: two native facts exercise
  binary64 math precision, negative zero, infinities, NaNs, domain results,
  allocation effects, contextual arithmetic, and aggregate initialization.
- `cmd/concept/smlib_tooling_test.go` and `smlib_module_roots_test.go`: complete
  deterministic emission, legacy stdout compatibility, nested import paths,
  source/artifact modes, missing/wrong/conflicting identities, physical aliases,
  strict C11 compilation, and Normal/Verify execution.
- Existing callable, provenance, checked-output, handle, math, and module tests
  remain part of full-package validation.

## Frozen port results

| Evidence | Before | After |
| --- | --- | --- |
| Mathematical cases | 239 passing | 239 passing |
| C++ differential comparisons, Normal + Verify | 478 passing | 478 passing |
| Port native facts | 3 per policy | 3 per policy |
| Deterministic reruns | passing | passing |
| Source integrity | 41 frozen files | all 41 hashes unchanged |
| Complete numerical reports | 239 baseline reports | all 239 exactly unchanged |

Exact report equality is additional evidence; differential acceptance continues
to use the original case-specific tolerances and independent mathematical checks.
No numerical algorithm, C++ oracle, known-defect documentation, or attribution was
modified. Optional port cleanup was deliberately left separate; the original
source remains the regression baseline. Nd remains bounded to 1–16 unknowns.

All four compiler repros now pass: local callback records, double literal context,
repeated erasure, and erasure inside Result unwrap. The break/continue repros
continue to reject, as required.

The complete CLI output's five files match the existing API artifacts byte for
byte. The CLI-generated solver compiles with `clang -std=c11 -pedantic-errors`
and runs from a directory containing only its executable. Scalar sqrt2 and
coupled Nd probes converge with residuals below `1e-10`, independently of C++.
The test harness's callback C aliases are supplied through its existing header;
the compiler-generated header is emitted directly without reconstruction.

## Reproduction

From Concept, with Go, Clang and Python available:

```powershell
go test ./...
go vet ./...
go test ./internal/concept ./cmd/concept -run SMLib -count=1
go run ./cmd/concept test libraries/Standard/smlib_numerics.concept_test
go run ./cmd/concept test libraries/Standard/smlib_numerics.concept_test --verify
```

From the unchanged external `phase1` directory:

```powershell
python run.py --benchmark
python tools/check_repros.py
```

The port runner builds the current Concept checkout, qualifies strict generated
C11 and both policies, runs native facts and independent mathematical/differential
checks, verifies deterministic outputs, then benchmarks. Its existing dependency
configuration and frozen upstream checkout remain authoritative.

For complete CLI artifacts (using the executable built by that runner):

```powershell
build/concept.exe emit-c SMLib/Solver.concept --out-dir build/cli
build/concept.exe emit-c SMLib/Solver.concept --verify --out-dir build/cli-verify
```

Local detailed logs, frozen source hashes, baseline executables, qualification
JSON, numerical reports, and paired measurements are retained in the ignored
`Concept/artifacts/smlib-hardening` directory. Reproduction does not depend on
committing those machine-specific files.

## Validation status

Pass A focused compiler/native checks and vet passed. Its first aggregate run
found stale conformance fixture counts; those were reconciled and rechecked.
Pass B focused/native checks and vet passed. Its aggregate run exposed an opaque
handle diagnostic regression, fixed in `995fb63`, and one intermittent existing
`TestNativeSchedulerWorkers` exit 6. That harness requires multiple OS workers
to become active even when one can finish all work first. Three isolated reruns
passed; neither scheduler nor its harness was changed. The complete Pass C Go
suite passed before the final callback lifetime addition.

Final `go test ./...`: passed (`cmd/concept` 27.250s, `internal/concept`
331.974s, `internal/concept/profile/vulkan` 0.501s; other packages have no tests).
The earlier scheduler failure did not recur in this full run.
Final `go vet ./...`: passed. The port runner passed after `e855765`, including
six native fact executions; Standard's two new facts also passed in both policies.
Focused final callback/lifetime/native regressions passed. Deterministic strict
C11 and latest CLI standalone checks passed. Frozen root/legacy Zig validation
was skipped: no retired compiler or its build infrastructure was changed.

## Performance after correctness

Seven paired rounds alternated old/new executable order, with 100,000 solves
per workload per round. Values below are median microseconds per solve on this
host; the old executable was preserved before compiler changes.

| Workload | Baseline | Hardened | Ratio |
| --- | ---: | ---: | ---: |
| sqrt2 | 0.126508 | 0.130734 | 1.0334 |
| Nd quadratic, 3 | 0.721515 | 0.750026 | 1.0395 |
| Nd coupled linear, 16 | 4.802911 | 4.769325 | 0.9930 |
| Nd Rosenbrock step cutting | 2.749963 | 2.773824 | 1.0087 |

An earlier paired run on the same unchanged algorithms measured ratios
1.0032, 0.9967, 0.9905, and 0.9911 respectively. Historical runner timings alone
had suggested a larger slowdown; paired execution did not reproduce it.
The latest measurements show up to 4% slower, within the observed run-to-run
variation. There is no established stable performance regression or speedup;
these measurements are host-specific and have no confidence interval.

## Deliberate boundaries

Binary32 and binary64 typed operands retain strict precision; no implicit mixed
promotion was introduced. Host libm results are not promised bit-identical across
platforms. Only Windows execution is qualified here.

Zero construction covers recursively defaultable plain data, rejecting owners,
borrows, Drop-bearing and other non-defaultable fields. Per-field omission,
user-defined constructors, and a general default expression remain deferred.
Modules retain explicit path-based discovery; there is no new ambient registry.

[The yield-return RFC](../design/SMLIB-YIELD-RETURN-RFC.md) recommends **defer**:
existing structured loops and functions suffice for this port. It specifies a
candidate grammar, targeting, typing, normal termination, cleanup/ownership,
lowering, valid/invalid examples, and initial exclusion from automata state
bodies. No prototype or implementation was included. Existing yield, complete,
fail, state transitions, ownership, scheduler, and frame semantics are preserved.
