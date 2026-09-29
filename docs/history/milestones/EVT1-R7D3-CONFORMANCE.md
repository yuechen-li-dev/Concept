# EVT1 R7d3 conformance — Burn the Sharp Edges

Status: language hardening before access-summary derivation.

Baseline: `4781e5d4e49a2775e2f2656d13e500413b4d77d0` (`concept-evt1-stage0-go`). R7c is `1faeb29`; R7d1 is `cea7969`; R7d2 is `4781e5d`.

Toolchain: Go `1.27.0 windows/amd64`, Zig `0.16.0`, Clang `22.1.3`. The active EVT1 corpus remains 712 `.concept` fixtures. The strict harness uses `clang -std=c11 -Wall -Wextra -Werror`; native package tests compile and link closed programs or explicitly supplied foreign dependencies.

## Correctness surface

- Decimal and hexadecimal literals use exact `uint64` magnitude plus lexical sign until target typing.
- `INT32_MIN` is exact; range failures report `CV4644` rather than clamping.
- Constant division by zero reports `CV4645`; invalid shifts `CV4646`; symbol collisions `CV4647`; logical spelling `CV4648`; constant signed overflow `CV4649`.
- `int` is `int32_t`; signed arithmetic is checked and dynamic division/shift operations are guarded.
- Expression precedence matches C/C++ across arithmetic, shifts, comparisons, bitwise operators, equality, `and`, and `or`.
- Call-site literals use expected parameter types without weakening overload ambiguity.
- Declared module paths own C symbol namespaces and case-fold collisions are disambiguated.

## Durable evidence

`go test ./internal/concept -run R7d3 -count=1` (also `oct make BurnIn --file Make.oct`) covers boundaries, seeded decimal/hex generation, runtime hash/mask oracle comparison, precedence, strict-C11 compilation, UB diagnostics, contextual calls, case identity, and a linked same-stem two-module program.

Final validation:

- `go test ./...`: pass (full corpus and native integration, 106.826 seconds on the recorded host)
- `go vet ./...`: pass
- root and `legacy/poc3-zig` `zig build test`: pass
- Standard: 3 facts passed
- DragonGod: 18 facts passed and one benchmark completed
- 100-run C/MIR/manifest/header byte identity: pass
- `git diff --check`: pass

The pass also fixed a general async persistence defect exposed by the stricter lowering: persistent views now retain initializer-provenance backing storage across suspension.

## Deliberate deferrals

- Binary literals and separators remain deferred.
- `move` on a copyable value remains a lint candidate because no warning-class diagnostics subsystem exists.
- PascalCase remains a later lint concern; no general `concept lint` facility exists.
- No access kinds, access summaries, derivation, synchronization elision, or worker-safe DragonGod integration are part of R7d3.
