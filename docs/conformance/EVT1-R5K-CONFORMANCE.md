# EVT1 R5k semantic reconciliation and freeze

Status: success

## Baseline

- Starting HEAD: `01d90e0b3f0f9ea821a2ec38968ca5d13622fc52`
- Starting worktree: clean
- Compiler ID: `concept-evt1-stage0-go`
- Go: `go1.27.0 windows/amd64`
- Zig: `0.16.0`
- Semantic-organization commit: `33c0688`
- Baseline validation: `go test ./...`, `go vet ./...`, root `zig build
  test`, and `legacy/poc3-zig/zig build test` all passed.

R5k added no major language feature. It reconciled one specification
contradiction: `assert` and `static_assert` are direct runtime and compile-time
assertion forms. The future `Assert.*` library is R6 tooling/library direction,
not an R5 primitive used to define them.

## Milestone-name inventory and rename result

The pre-change inventory classified conformance reports, commit references,
and chronology as historical; active Go files, test files, corpus directories,
diagnostic text, and architecture headings as active; and historical
provenance strings inside tests as mixed. Historical names were retained only
where they identify facts about R0-R5.

Active implementation files now use `interface_dyn.go`,
`iteration_yield.go`, `machine_stack.go`, `async.go`,
`async_control_flow.go`, `callable_capture.go`, and `callable_types.go`.
All 25 milestone-oriented active test files were renamed for their subsystem.
The exact old-to-new implementation and test table is in
`docs/migration/EVT1-R5-SEMANTIC-RENAME-MAP.md`.

The 24 historical corpus roots from `language/evt1-r1/core` through
`language/evt1-r5j/core` moved once into `language/evt1`. No compatibility copy
or second fixture authority remains. `language/evt1/manifest.json` preserves
milestone provenance and exact class counts. Historical conformance filenames
remain milestone-named and now point to semantic fixture paths.

Post-rename scans find no active `internal/concept/r[0-9]`,
`language/evt1-r*`, `TestEVT1R*`, `analyzeR*`, `lowerR*`, or `planR*` path or
symbol outside the archaeology map and historical reports. Active compiler
diagnostics, MIR, Plan, and generated C contain no milestone identifiers.

## Canonical source audit

The active corpus uses C++-lineage declaration forms and PascalCase public,
type, function, method, concept, interface, and compiler-known names. Six
lowercase `main` declarations were corrected to canonical `Main`. The only
non-canonical binding/async spellings in the semantic corpus are dedicated
compatibility specimens.

The accepted alias ledger is:

| Spelling | Classification | Canonical form |
|---|---|---|
| `let` | Compatibility | typed `const` or `const auto` |
| `var` | Compatibility | `auto` |
| `type` | Compatibility | `using` |
| `asynchronous` | Compatibility | `async` |
| `awaitchronous` | Joke-but-permanent | exact `await` alias; preserve by default |

There are five dedicated compatibility corpus fixtures plus focused inline
tests for `var` and `type`. Formatter-ready policy is stored in
`docs/architecture/EVT1-CANONICALIZATION.json`; R5k does not implement a
formatter or lint rule.

## Semantic authority reconciliation

One ownership model applies everywhere: ordinary values copy structurally when
all fields are copyable; `owned` values are movable-only; immovable values stay
in final storage; transfer is explicit; assignment to live owning storage
drops the old value before initializing the new value; and borrowed dyn,
callback, Span, and tensor descriptors own neither backing nor referent.

Exactly-once Drop is covered across the freeze matrix:

| Storage situation | Executable evidence |
|---|---|
| ordinary local | `ownership/valid/drop_scope_order.concept` |
| aggregate elements/fields | `storage/arrays/valid/array_element_drop_order.concept` |
| owned Result payload | `failure/valid/result_owned_payload_move.concept` |
| persistent machine field | `automata/state/valid/machine_owned_field_cleanup.concept` |
| nested machine frame | `machine-stack/valid/machine_owned_child_cleanup.concept` |
| Async persistent field | `async/basic/valid/async_owned_local_across_await.concept` |
| callable environment field | `callable/capture/valid/callback_consumed_owned_capture.concept` |
| callable moved into machine field | `callable/types/valid/callable_machine_field_owned_capture.concept` |
| callable live across await | `callable/capture/valid/callback_async_live_across_await.concept` |

The existing cleanup tests inspect guarded generated paths, reverse order,
transfer suppression, frame pop/completion, and result consumption. R5k found
no competing cleanup authority and added no Drop rule.

Lexical/call-result provenance remains the one authority for `ref`, `ref
const`, `scoped`, Span, tensor views, borrowed dyn, erased callbacks, captured
references, machine fields, Async persistent fields, and async interface
receivers. Region identity, alignment, contiguity, shape, rank, mutability,
disjointness, no-allocation, no-copy, no-transfer, and lifetime facts survive
layout -> stream -> Span -> tensor and into callable/async descriptors where
applicable. Escape tests prove these wrappers do not extend provenance.

Async retains `Scheduler: None`, `SavedPC: None`, the capacity-eight inline
machine stack, compiler-derived continuation state, exactly-once await
evaluation/outcome consumption, and ordinary ownership/lifetime cleanup. Dyn
selects a static witness entry; callables have exact code/environment identity;
neither path boxes, allocates, adds RTTI, or creates a second runtime.

Automata `with state`, callable `with (...)`, and compiler-derived async state
are three distinct constructs governed by the same ownership/provenance laws.
Machine Failure is a lifecycle outcome; `Result::Error` is a language value
channel. `?`, `!`, try/except, machine fail, and async Result composition retain
that distinction.

Planner consumes typed Module/MIR facts for tensors, bounds, cleanup, failure,
dispatch, layout/stream/Span, automata/stack, async, and callables. It validates
strategy selection and deterministic serialization; it cannot invent semantic
legality. Stable schema/version identifiers were not renamed.

## Diagnostics and runtime reasons

The active compiler contains 425 unique semantic diagnostic codes. R5k removed
milestone labels from active diagnostic messages, Planner failures, async abort
reasons, and implementation comments. The stable panic ledger is recorded in
`docs/migration/EVT1-DIAGNOSTIC-RECONCILIATION.md`. In particular,
`invalid machine state reached` is now pinned by executable generated-C
evidence without leaking internal state indices.

## Unified corpus and composition results

| Classification | Count |
|---|---:|
| valid | 379 |
| static invalid | 243 |
| runtime negative | 13 |
| dedicated compatibility fixtures | 5 |
| expected PoC3 divergences | 4 |
| EVT1 gaps | 0 |
| unresolved specification ambiguities | 0 |

`TestSemanticCorpusManifest` is the unified path: it checks manifest ownership
and counts, compiles/generates all valid and runtime-negative fixtures, requires
every static-invalid fixture to reject, and rejects fixtures outside the
manifest. `TestSemanticCorpusForbiddenRuntime` scans generated C for every
valid fixture.

The semantic freeze suite includes:

- async + dyn + callable in
  `callable/capture/valid/callback_async_interface_composition.concept`;
- automata + callable + ownership in
  `callable/types/valid/callable_machine_field_owned_capture.concept`;
- callable + Span/lifetime and callable live across await;
- a DragonGod-style proof set covering owned automata state, nested machine
  frames, inference-driven transitions, and callable machine fields;
- `composition/valid/prometheus_storage_planning.concept`, one specimen using
  layout, stream, Span, tensor interpretation, symbolic elementwise work,
  qualified facts, and Planner output;
- `composition/valid/aetheris_edge_policy.concept`, a non-game edge/corner
  utility decision;
- `composition/valid/octetdb_transaction_policy.concept`, a transaction-agent
  workflow combining automata, Result matching, and retry decision policy;
- `inference/valid/infer_probability_access.concept`, proving probability mass
  matters downstream rather than reducing infer to HardMax.

## Differential, profile, backend, and determinism status

PoC3 remains historical/differential evidence. The four remaining expected
divergences are intentional EVT1 redesigns: bounded comptime and structural
ordinary-value copy. There are no unexplained divergences, EVT1 gaps, or PoC3
legacy cases admitted as current semantics. The retired Zig compiler remains
buildable in `legacy/poc3-zig`; it is not an authority. Oct and Prometheus are
consumers, not compiler build-time owners, and no sibling repository changed.

Core/Vulkan profile tests pass. Vulkan builtins remain profile-owned and Core
denial cases remain active. Representative and full-corpus generation retains
strict C11 as the bootstrap semantic oracle without making its representation
normative.

The forbidden-runtime full-valid-corpus scan rejects allocation, GC, RTTI,
per-object vtables, task/promise/future runtimes, executors, schedulers, event
loops, setjmp/longjmp, coroutine ABI, and closure boxes. No exception was
required.

One hundred repeated Generate runs of the async + dyn + callable specimen are
byte-identical across MIR, semantic facts, Plan, C/H, maps, and manifests.
Manifest decode/re-encode is also byte-identical for 100 iterations. Direct CLI
proof on that specimen produced semantic-only output for `check`, `mir`,
`plan`, and `emit-c` with no active milestone label.

## Freeze and R6 boundary

Deferred and rejected work is classified in
`docs/architecture/EVT1-COMPATIBILITY-AND-DEFERRED.md`. R6 begins with
first-class test discovery/execution and the proposed `[[fact]]`, `[[theory]]`,
`[[artifact]]`, `[[benchmark]]`, `[[prophecy]]`, and `Assert.*` library surface,
then formatter/lint, diagnostic UX, build/package tooling, and LSP. None was
implemented in R5k.

EVT1 R5 language-core semantics are frozen for the transition into R6 tooling.
R6 may fix bugs, diagnostics, contradictions, or specification errors, but
should not casually expand the language surface. New language features require
an explicit post-freeze proposal.

EVT1 R5 language-core semantics are frozen.
The next phase is R6 tooling.
