# Concept Vulkan reconciliation: mini-milestone ladder

Status: **complete** (2026-09-29). VK0–VK9 landed; the compute-dispatch
vertical passed on a real GPU in Normal and Verify modes.

## Completion report

| Rung | Commit | Outcome |
| --- | --- | --- |
| VK0 | `d831ef2` | Specimens renamed to Core; originals frozen in `legacy/evt1-specimens/` |
| VK1 | `feceb3d` | Guarded `transition match` arms (ordered, final unguarded arm required) |
| VK2 | `5e0f553` | `with input T` and `on Pattern [when g] => S;` with must-use `StepOutcome` |
| VK3 | `779a80e` | `terminal state`; ambiguous reactions reported as `Ambiguous` |
| VK4 | `c7c6bf6` | `Standard.Collection.Outbox<T,N>` and step-machine ports |
| VK5 | `646b008` | M-era automata, `dispatch`, effects and actuators removed, each with a `REMOVED_*` diagnostic |
| VK6 | `e6e0935` | `extern "C" handle Name;` opaque foreign handles in Core |
| VK7 | `8f74a88` | `libraries/Vulkan` in ordinary Concept: owned Context, Buffer, ComputePipeline, Submission |
| VK9a | `4a458ed` | Profile semantics removed; constitution and isolation audit moved to `docs/history/vulkan/` |
| VK9b | `b54deb7` | `profile Vulkan` kept as setup automation only (implied import, runtime linking) |
| VK8 | `d8b0b71`, `3ad5760` | Compute dispatch on the test device and a real GPU; lifetime observers on both runtimes |

Device run (Windows, Vulkan SDK 1.4.350, MinGW gcc 15.2, `tools/vk8/run_vk8.ps1` (now `tools/vulkan/run_gpu.ps1`)):
all nine steps pass. The C reference and `compute_dispatch` both return 4032;
`buffer_lifetime` sees zero live buffers after scope exit on the GPU runtime.

Left open:

- The stale `concept-vulkan.exe`/`.obj` in the Oct repository root (VK9) is
  in a different repository and is left for the owner.
- The follow-up rung for access and barrier derivation, informed by the VK8
  friction note below.

## Goal

`profile Vulkan` stops being a second language. After this ladder:

- Concept has **one** machine model (step machines). The useful M-era ideas
  (`on Signal::A when C =>`, `terminal state`, the ambiguity check, and
  must-use step outcomes) become ordinary Core features.
- Vulkan support is **a library** (`libraries/Vulkan`) written in ordinary
  Concept, plus one small Core addition: opaque foreign handles.
- The parallel M-era surface is removed: `dispatch`, `effect`, `effects`
  batches, `actuator`, `initial machine/state`, and `borrow context:`
  automata parameters. The originals are kept in `legacy/`, not deleted from
  history.

## Rules for every rung

Each rung is one commit (or a short series), rebased on the latest main, and
ends in **Success**, **Meaningful progression**, or **Honest stop** (AGENTS.md).

**Standing gates.** Every rung must pass all of these:

| Gate | Command / check |
| --- | --- |
| Vet | `go vet ./...` |
| Go suite | `go test ./...` (prebuilt binary, ≤170 s chunks; known flakes: `TestNativeSchedulerWorkers` on 2 cores, `TestDifferentiatorStrictC11BothCompilers` without clang) |
| Zig compatibility | `zig build test`, 402/402 |
| Goldens | `concept test libraries/Golden`, Normal and Verify |
| Packages | Standard and DragonGod package tests, Normal and Verify |
| Tour | `TestTourExamples*` |
| Checked outputs | `TestEVT1CheckedOutputsMatch` (regenerated only where a rung says so, and the diff is reviewed) |

**No hoarding.** When a rung replaces a feature, the old path is removed in
the same ladder (VK5 at the latest), not left behind a flag. A removed
spelling gets a migration diagnostic that names its replacement.

**Style rule, to be written into the spec.** Use `on` for reactions to
external input. Use `transition` for a next state the machine computes
itself.

## Decisions baked into this plan (flag any you disagree with)

- **D1. Two guard semantics, on purpose.**
  - `transition match (e) { P when c => S; }` is **ordered first-match**,
    exactly like `match`.
  - `on` clauses in a state form an **unordered set**. Before choosing an
    arm, all guards for the incoming variant are evaluated.
    - Exactly one true: take it.
    - None true: take `otherwise`, or report `Unhandled` if there is no
      `otherwise`.
    - More than one true: report `Ambiguous`, and the state is unchanged.

  This keeps the M-era determinism guarantee where it was valuable (reactive
  input) without changing `match`.
- **D2. Ambiguity is a runtime outcome in both modes.** Normal mode does not
  silently pick one arm. A static overlap check (same variant, both
  unguarded, or syntactically identical guards) is a compile error where it
  is decidable.
- **D3. Outcomes are must-use.** Stepping with input returns
  `[[must_use]] StepOutcome { Transitioned, Unhandled, Ambiguous, Finished, AlreadyFinished }`.
  `EffectBatchOccupied` disappears with effect batches. Input-less `Step`
  keeps its current signature.
- **D4. Effects are data, not a language feature.** Machines append to a
  Standard `Outbox<Effect, N>`, which is a bounded, owned, must-drain queue
  that returns `Result` on overflow. `emit E;` is **not** added up front. It
  is added in VK4 only if the ported specimens read measurably worse without
  it, and then only as sugar for an outbox push.
- **D5. Actuators are functions.** An actuator is an exhaustive
  `match (effect)` in ordinary code. DragonGod.Actuation already proves the
  pattern.
- **D6. Handles are the only Vulkan-motivated Core addition.**
  `extern "C" handle VkBuffer;` declares an opaque, equality-only,
  non-arithmetic, `[[repr(C)]]`-compatible pointer-sized type. Null is
  expressed as `Option<VkBuffer>`, never as a sentinel.
- **D7. `profile Vulkan` ends as policy or not at all.** In VK9 it becomes
  "implicit `import Vulkan` + HotPathPolicy defaults + link settings". If
  that turns out to be nothing a manifest can't say, it is deleted.

## The ladder

### VK0: honest names and quarantine prep

*Entry:* the current main.

- Rename the 8 `examples/evt1/*_core.concept` specimens that are really
  profile Vulkan but compile as Core. Switch them to `profile Core` and give
  them subject names. This fixes my earlier `_core` misnomer.
- Copy all 22 specimens, with their checked outputs, into
  `legacy/evt1-specimens/`. Add a README with provenance, stating that they
  are frozen and not compiled.
- Regenerate the checked outputs for the renamed Core specimens.

*Exit:* standing gates. The checked-output diff shows only renames and
`profile` changes. No compiler code changes.

### VK1: guarded `transition match`

- Parser: allow `P when expr =>` in `transition match` arms. This removes the
  CV4020 case. Type-check the guard as `bool`, with no effects.
- Semantics: ordered first-match (D1). The exhaustiveness check ignores
  guarded arms, as `match` does.
- Lower in EVT1 C generation. *(Amended at VK1: EVT2 LIR has no enum values yet, so
  `transition match` of any kind stays behind its existing
  `EVT2_UNSUPPORTED_STATEMENT` boundary; EVT2 parity waits for EVT2 enums.)*

*New tests:*

- a valid fixture with guard fallthrough
- an invalid fixture for a non-bool guard
- a missing-exhaustive-arm diagnostic when every arm is guarded
- a Normal/Verify `.concept_test` with a Golden-style theory over guard
  inputs

*Exit:* standing gates.

### VK2: `with input` and `on`

- `automata X with input Sig with state { … }`, where `Sig` is any enum,
  including payload enums.
- In a state: `on Sig::A(x) when c => S;`, `on Sig::A => { …; S }`, and
  `otherwise => S;`. The unordered-set semantics are D1.
- `Step(inst, M, signal)` returns `StepOutcome` (D3). A state with no `on`
  clauses for the variant yields `Unhandled`.
- An automaton `with input` may still use internal `transition` in states
  that have no `on` clauses. Mixing both in one state is an error, which
  keeps the style rule enforceable.
- Lower in EVT1. EVT2 support follows EVT2 enum support (see VK1).

*New tests:*

- valid/invalid fixtures: unknown variant, payload binding, `otherwise`
  twice, mixed `on` and `transition`, discarded outcome → must-use
  diagnostic
- a runtime theory over every outcome except `Finished`

*Exit:* standing gates.

### VK3: `terminal state` and ambiguity

*(Amended at VK2: the runtime `Ambiguous` outcome and the static overlap checks
landed with `on` in VK2, since they define its selection rule. VK3 is
`terminal state`, `Finished`/`AlreadyFinished` coverage, and the parity table.)*

- `terminal state S { }` has no outgoing arms. Entering it makes
  `Complete()` true. A later `Step` returns `AlreadyFinished`, and the step
  that enters it returns `Finished`. This replaces `finish;`.
- Ambiguity (D2): the runtime `Ambiguous` outcome, plus a static overlap
  error for the decidable cases.

*New tests:*

- a door/lifecycle theory covering all five outcomes in Normal and Verify
- an invalid fixture for a statically overlapping `on` pair
- an invalid fixture for a transition out of a terminal state

*Exit:* standing gates. Feature parity with the M-era `dispatch` is reached
at this point, and a parity table goes in the commit message.

### VK4: Outbox and ports

- `Standard.Outbox<T, N>`: owned, `Push → Result`, `Drain`, must-use, and
  NoAllocation-provable. It gets its own `.concept_test`.
- Port the effect and actuator specimens to step machines with `on` plus an
  Outbox parameter or state field, with actuators as `match` functions.
  Decide on `emit` sugar (D4) here, with a before/after snippet in the
  commit.
- Port DragonGod's `HistoricalDominatusBehavior` from `dispatch` to `Step`.
- Update the checked outputs for the ported specimens. The legacy copies
  from VK0 stay untouched.

*Exit:* standing gates. `rg 'dispatch\(|effects |actuator ' libraries examples language/evt1/**/valid`
finds nothing outside `legacy/`.

### VK5: removal

- Delete the signal dialect, `effect`, `effects`, `actuator`, `dispatch`,
  `initial machine/state`, `borrow context:` automata parameters, and
  `AutomataDispatchOutcome`. That is about 430 lines across parse, generate,
  validate, types, automata and `profile_vulkan.go`.
- Replace them with migration diagnostics, one per removed spelling, each
  naming the VK1–VK4 replacement. Invalid fixtures pin each diagnostic.
- Remove `EVT2_UNSUPPORTED_SIGNAL_AUTOMATA`.
- Delete, not rewrite, the tests that existed only to exercise the removed
  paths. List them in the commit.

*Exit:* standing gates, and a net negative line count in `internal/concept`
that is reported.

### VK6: opaque handles (independent, may run in parallel with VK1–VK5)

- `extern "C" handle Name;` in Core (D6). It emits a
  `typedef struct Name_T* Name;`-compatible declaration, or reuses the header
  typedef when a foreign contract names the header.
- Allowed: `==`, `!=`, copy, passing by value across `extern "C"`, and
  `Option<Name>` with a null niche.
- Rejected: arithmetic, ordering, conversion to or from integers outside
  `unsafe`, and `SizeOf` on anything but the handle itself.
- Remove the four builtin Vulkan handle types from the profile definition.
  Existing Vulkan specimens declare them instead.

*New tests:* valid and invalid fixtures, a C ABI round-trip through a
companion C file, and an `Option<handle>` niche layout assertion.

*Exit:* standing gates.

### VK7: `libraries/Vulkan` core

All of this is ordinary Core Concept with no profile hooks:

- `Vulkan.Handles`: handle declarations.
- `Vulkan.Result`: `VkResult` interpreted into `Result<T, VulkanError>`,
  must-use.
- `Vulkan.Flags`: `bits` for usage, memory property and access flags.
- `Vulkan.Memory`/`Buffer`: owned `Buffer<Memory, Usage, T>` with a `Drop`
  that frees, and sizes as `usize<byte>`.
- A foreign contract for the subset of `vulkan.h` that is used. It compiles
  against the real header when present, and a stub header in tests.
- Package tests for Result mapping, flags, and ownership (double-free and
  leak diagnostics), in Normal and Verify, with no GPU needed.
- `Prometheus.Vulkan` is dropped from `AdmittedImports`, and `profile_test.go`
  is updated.

*Exit:* standing gates, plus the Vulkan package tests.

### VK8: compute-dispatch vertical

This is the first real proof: create buffer → pipeline → record → submit →
wait → read back.

- `Vulkan.Commands`: a scoped `CommandRecording` and an owned, must-use
  `Submission` that is consumed by `Wait`.
- `Vulkan.Descriptors`: a descriptor layout generated from `[[binding]]`
  record fields via `generator`/`derive`.
- A Golden pair: `compute_dispatch.concept` and an equivalent hand-written
  `compute_dispatch.c`, compared for generated C shape and line count, and
  run on a real device when `vulkaninfo` is available (skipped otherwise,
  and reported as skipped, never as passed).
- A friction note: what still felt worse than C, which feeds a follow-up
  rung for access and barrier derivation (constitution §5–6). That is **not**
  in this ladder.

*Exit:* standing gates. The Golden compiles as strict C11 with gcc and
clang. A device run is attempted where hardware exists.

*(Amended at VK8: the vertical shipped as one `Vulkan` module rather than
`Vulkan.Commands`/`Vulkan.Descriptors`. Recording is internal to `Dispatch`,
and bindings are explicit `BindStorage` calls; `[[binding]]` derivation moved
to the follow-up rung. The Golden pair is `examples/vulkan/ComputeDispatch.concept`
(33 lines) and `examples/vulkan/reference/compute_dispatch.c` (209 lines); the
mechanics the C spells out live once in `libraries/Vulkan/native/device_runtime.c`.)*

**VK8 friction note.** What still feels worse than C, or no better:

- Bindings are positional. The storage-buffer count passed to
  `CreateComputePipeline` and the `BindStorage` indices must agree with the
  shader's layout by hand; nothing checks them against `double.comp`.
- Buffers are untyped bytes with int accessors. The size (256 bytes) and the
  element count (64) are stated separately, and each `WriteInt`/`ReadInt` is
  a runtime call rather than a mapped, typed view.
- Synchronization is invisible and fixed. The runtime records one
  shader-write → host-read barrier; Concept cannot express or check any
  other. This is the core of the access-and-barrier derivation rung.
- The kernel is a path string resolved at run time via
  `CONCEPT_VULKAN_KERNELS`, not a compile-time dependency.
- The device runtime supports one context per process.

What is clearly better: every object is owned and destroyed once in reverse
order without any cleanup code; every failure is a `VulkanError` through `?`;
a `Submission` that is dropped without `Wait` is waited by `Drop`; and the same
lifetime facts run unchanged on the test device and a GPU.

### VK9: shrink the profile and write the docs

*(Amended at VK9b: by decision, `profile Vulkan` is kept as setup automation:
implied `import Vulkan;` and the runtime linked by `concept test`; it adds no
semantics or builtins.)*

- Apply D7: reduce `profile Vulkan` to import, policy and link settings, or
  delete it. `profile_vulkan.go` and `profile_vulkan_definition.go` go away.
  `internal/concept/profile/vulkan` becomes a thin driver or is removed.
- Rewrite the Core-denial fixtures to test what still differs, or delete
  them.
- Docs:
  - The constitution moves to `docs/history/` with a header naming what
    survived.
  - Spec §23 is rewritten to cover `on`, `terminal`, `StepOutcome`, and
    Outbox.
  - A new `docs/library/VULKAN.md` is added, and
    `EVT1-VULKAN-PROFILE-ISOLATION.md` moves to history.
  - The style rule from the "Rules" section goes into the style guide.
- Remove the stale `concept-vulkan.exe`/`.obj` from the Oct root.

*Exit:* standing gates. `rg -i 'profile Vulkan'` finds only the thin
profile, its tests, and history docs.

## Order and parallelism

```text
VK0 ─ VK1 ─ VK2 ─ VK3 ─ VK4 ─ VK5 ─┐
VK6 ───────────────────────────────┴─ VK7 ─ VK8 ─ VK9
```

VK6 touches different files from VK1–VK5, so it can land at any point
before VK7.

## Explicitly out of scope

- Barrier and access derivation (constitution §5–6), which follows VK8.
- Ray-query and SGEMM milestones (constitution §15 M2–M4).
- A general effect system. The constitution's §8 rejection stands, and
  Outbox is a library.
- The `evt1` Go identifier prefix.

## Risks

| Risk | Mitigation |
| --- | --- |
| Codex edits the same files in parallel | Small commits, rebase before each, and VK5 lands as one fast commit |
| The unordered-set `on` semantics surprise readers who expect `match` | The spec states D1 in one sentence, and the style rule steers internal logic to `transition match` |
| Checked-output churn hides real regressions | Only VK0/VK4 regenerate them, and each diff is reviewed and summarized |
| No GPU in CI | VK7 needs none. VK8 reports device runs as skipped, never as passed |
| `emit` gets added "just in case" | D4 gates it on a before/after comparison |
