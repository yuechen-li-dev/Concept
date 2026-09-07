# EVT1 R4c conformance record

Status: canonical failure semantics

## Baseline

```text
HEAD: 331498703527357aa0e4cb9571d4affbec7c75e5
compiler: concept-evt1-stage0-go
R4b: 11/11 PASS (6 valid, 5 invalid)
Go: go1.27.0 windows/amd64
Zig: 0.16.0
worktree: clean
```

The baseline Go suite passed. Both root and `legacy/poc3-zig` Zig suites
passed. The active compiler had ordinary concrete payload enums and exhaustive
match, a Vulkan-only `Result<void,E>` signature bridge, and bounded
`static_assert`; it had no Core Option/Result, postfix propagation/unroll,
local typed handling, or runtime assert/panic helper. Go had 65 top-level test
functions before R4c.

## Corpus and classification

```text
24 total
24 PASS
14 valid
10 invalid
0 EXPECTED-DIVERGENCE
0 EVT1-GAP
0 POC3-LEGACY
0 SPEC-AMBIGUITY
```

The readable corpus is under `language/evt1-r4c/core`. PoC3 Phase 5 concrete
Result-shaped propagation, Phase 17 panic/assert, and Phase 22 Option/match are
recorded as evidence. Generic Result, postfix `?`/`!`, exact typed
`try`/`except`, ownership integration, and fallible lifetime payloads are
EVT1-new evidence; parity is not fabricated.

## Executable model

Option and Result are compiler-known generic specializations with ordinary
tag/payload enum semantics and qualified constructors. Match uses the existing
exhaustiveness and payload-binding path. `?` validates the enclosing channel
and exact error identity. `!` lowers to a source-bearing terminal helper.
`try`/`except` lowers to local error temporaries and labels selected entirely
at compile time. No exception runtime or unwind path exists.

MIR records `option_propagate`, `result_propagate`, `option_unroll`,
`result_unroll`, `try_handler`, and `assert`. Nine successful native strict-C11
harnesses cover nested Option, Option and Result propagation, successful Result
unroll, two concrete handler types, runtime assert success, and owned
Option/Result payload transfer plus active-variant cleanup. MIR and generated-C
evidence name the synthesized carrier drop, which dispatches only the active
payload to its ordinary `Drop` function. Two subprocess harnesses prove failed Result
unroll and failed runtime assert terminate with deterministic source-bearing
panic output.

The lifetime pair proves that `Result<View,E>` remains parameter-derived
through `?`, while a local-derived carrier cannot escape. The immovable case
proves generic carriers do not weaken by-value embedding rules.

## Boundary

R4c adds no exception objects, throw, unwinding, arbitrary-frame catch,
implicit error conversion, Option `except` arm, generalized panic runtime,
allocation, testing framework, Span, dyn, async, or coroutine behavior.

## Recommended next milestone

R4d should close failure contracts across separately compiled module boundaries:
freeze the carrier C ABI, prove exact-error signatures through declarations and
calls, and pin the cleanup-before-terminal-panic rule. It should remain bounded:
no implicit error conversion, allocation, exception tables, or unwinding.
