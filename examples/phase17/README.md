# Phase 17 runtime failure examples

Phase 17 makes ordinary runtime failure explicit without adding exceptions,
stack unwinding, panic hooks, hidden allocation, or stack traces.

## Runtime failure forms

- `panic(reason)` is an unconditional runtime failure. The hosted C backend emits
  a backend-owned `cpt_panic` helper, prints `panic: <reason>` to stderr through `fprintf`, and exits with
  code 101.
- `assert(condition, reason)` is the runtime invariant form of the same doctrine
  as `Assert.True(condition, reason)`. A true condition continues normally; a
  false condition routes to `cpt_panic` and exits with code 101.
- Reasons are mandatory and must not be empty or whitespace-only; violations are rejected before lowering.
- Statement-position `panic` and `assert` are the implemented v0 forms. Named
  `because:` syntax, expression-position panic/assert, and `never` are deferred.

## Runtime assertions and test assertions

Ordinary runtime `assert` is independent of `Core.Test` and does not require the
test runner. `Assert.True` and `Assert.False` in `.con_test` files share the
assertion/invariant doctrine but report through the test runner. `Expect.*`
remains the expectation-style test API.

## Runtime behavior notes

The hosted C helper is backend-owned, emitted once per generated C unit when needed, and shared by explicit `panic`, failing runtime `assert`, and the migrated machine trap. The fixture harness currently pins reason output through generated-C/backend assertions rather than stderr matching. Runtime failure uses no heap allocation, exceptions, unwinding, stack traces, or panic hooks in v0.

## Migrated machine trap

`Result(machine)` before completion now routes through the same `cpt_panic`
helper with the stable reason:

```text
machine result cannot be read before completion
```

Reading `Result(machine)` after completion is unchanged and still returns the
machine result.

## Examples

- `panic-example.concept` demonstrates unconditional runtime failure.
- `assert-example.concept` demonstrates a runtime invariant assertion.
- `machine-result-panic.concept` demonstrates the migrated machine-result trap
  and the successful after-completion path.
- `runtime-failure-notes.concept` summarizes v0 doctrine and deferred syntax.
