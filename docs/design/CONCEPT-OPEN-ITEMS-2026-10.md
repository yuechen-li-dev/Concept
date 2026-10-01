# Concept: state and open items (2026-10-01)

A handoff for the next agent. Part 1 is what changed in the last two days and
where it lives. Part 2 is what remains, by area, each with the current state
and the suggested next step. Part 3 is working conventions.

## Part 1: what landed

### Vulkan library ladder (VL1–VL6), complete

`libraries/Vulkan` is now a Vulkan 1.4 compute library that automates the
mechanical parts of Vulkan. Plan and status:
`docs/design/CONCEPT-VULKAN-LIBRARY-LADDER.md`; user guide:
`docs/library/VULKAN.md`.

- `Buffer<T>` with placements (`Device`, `Upload`, `Readback`, `Shared`);
  `Upload`/`Download` stage automatically.
- Push descriptors, synchronization2, pipeline cache, device choice
  (`CONCEPT_VULKAN_DEVICE`), validation (`CONCEPT_VULKAN_VALIDATION=1`: debug
  messenger, synchronization validation through `VK_EXT_layer_settings`;
  validation messages count as hazards).
- `Recording` derives barriers from each command's declared Read/Write
  access. The test device (`native/test_device.c`) checks synchronization
  and leaks.
- `concept vulkan-bind` generates a kernel module from SPIR-V (workgroup
  size, push-constant `layout` with `static_assert`, `Load<K>`, `Record<K>`
  with access taken from `NonWritable`); stale bindings fail `concept test`.
- `tools/vulkan/run_gpu.ps1`: 13/13 pass on RTX 3070 and Radeon 780M,
  including Khronos sync validation with zero findings.

### Innate concepts MVP (IC0–IC6), complete

The compiler's own validation rules are becoming concepts written in Concept.
Design and as-built notes: `docs/design/EVT2-INNATE-CONCEPTS.md`.

- `internal/concept/innate/Innate.concept` is embedded in the compiler. Its
  `innate concept`s apply to every declaration of their kind. Its hash is the
  innate identity in every module artifact (`MODULE_INNATE_STALE`).
- Compile-time `declaration` / `typename` values and a closed set of
  `compiler.*` observations (`comptime_subjects.go`); `compiler.Shape`
  returns the exhaustive `TypeShape` enum.
- Two rules moved from Go to Concept through the strangler protocol
  (shadow, switch, delete; agreement test in `innate_agreement_test.go`):
  CV4653 (`DroppableFieldIsOwned`) and C_ABI_REPR_INVALID
  (`CReprIsPlainData`).

### Language and compiler changes

| Change | Where |
| --- | --- |
| Template functions call template functions with deduced arguments | spec §31 |
| Builtin-named types (`Double`) get distinct C names | failure.go, validate.go |
| CV4653: a field whose type has a Drop must be `owned` | innate concept |
| `concept check` validates generic instance structs (was weaker than `emit-c`) | validate.go |
| `concept test <native project>` accepts `--verify`/`--verbose` in any order | cmd/concept/main.go |
| Comptime `if`/`else`, `for` over ranges and fixed arrays, `string + string` | spec §15 |
| Comptime recursion with an explicit bound: `comptime T F(...) bounded(N)` | spec §15 |
| repr(C) / extern "C" aggregates with a Drop are rejected (the check was dead) | evt1CABIValue |
| Guarded match: `match { when c => x, otherwise => y }`, runtime and comptime | spec §10 |
| Spans cross `extern "C"`; `AsBytes` | spec §16.3, §21 |

## Part 2: open items

### A. Compiler structure

1. **String-keyed generic type identity.** Imported generic instances arrive
   as closed nominal structs (`Buffer<int>`); `evt1GenericApplicationOf`
   re-parses the name to recover its arguments, and the `Double`/`double`
   C-name collision came from the same root. *Next:* carry structured type
   applications in `concept-module.v1` artifacts and delete the re-parse.
   Do this before any front-end self-hosting, so it is not ported.
2. **`validate.go` is about 8,400 lines.** Rules land in long switch arms with
   threaded flags (`templateInfo`, `inComptimeFn`). *Next:* keep moving
   declarative rules into innate concepts (see C) and split the rest into
   per-feature passes.
3. **Accepted-but-inert sweep.** Three bugs of this class were found in two
   days: CV4653 (fields that never drop), `check` weaker than `emit-c` on
   generic instances, and the dead destruction check in the C ABI rule.
   *Next:* audit for attributes that are parsed but unused, qualifiers
   ignored in some positions, and checks that cannot fire. Rule: a
   declaration that changes nothing is rejected.
4. **Comptime context threading.** Two validation paths ignored compile-time
   context (enum payload construction, foreach); both are fixed. Others may
   remain wherever a helper hard-codes `inComptimeFn: false`; `grep
   "templateInfo, false)" validate.go` lists them.
5. **Diagnostic order.** Innate concepts run after the rest of analysis, so a
   moved rule reports later than its Go original in files with several
   errors. Acceptable, but tools that expect the first error should not
   depend on rule order.

### B. Language decisions for the owner

1. **Ownership of generic fields.** Today a generic container writes
   `owned T value;` (works for `Box<int>` too). The alternative is that
   structural Drop releases every Drop-typed field, owned or not, and CV4653
   disappears. Undecided.
2. **Friction without safety value:** int-only indexing (forces `as int`),
   no `1u` literal, no `~`, `TruncTo` plus CV4028 in float-to-int code,
   positional construction of many-field records (no named field
   initializers or defaults). Recommend separating deliberate restrictions
   from not-yet-implemented ones in the spec.
3. **Diagnostics should name the fix,** as CV4653 does (`declare it
   ``owned T value;`` in Box`). Several older messages describe compiler
   internals ("closed concrete type arguments").
4. **Generic repr(C) records do not parse.** `[[repr(C)]] template <typename
   T> record struct Pair` fails with `expected "class"`. Either admit
   template records or reject the attribute with a clear message.
5. **`comptime if` / `comptime for` inside runtime functions** (static
   branching and unrolling, like `if constexpr`). Wanted; not started.

### C. Innate concepts: next steps

1. **Next rules to move** (all declarative, all have Go originals to agree
   with): struct field embedding (CV4525 reference fields,
   CV4138 immovable fields, CALLABLE_FIELD_REF_ESCAPE), enum payload rules
   (CV4525, CV4139, CV4133), then the extern "C" signature domain.
2. **Subject kinds.** Innate concepts apply to `TypeDeclaration` and
   `FieldDeclaration`; enum payload fields are not subjects yet. Function
   declarations need observations (parameters, result, ABI) before the
   extern "C" rule can move.
3. **Predicate requirements in declared concepts** are rejected
   (`PREDICATE_REQUIREMENT_SCOPE`). They are safe (they only narrow) and
   useful for project lint; lifting needs evaluation in `concept_assert.go`
   and `project_policy.go` proof graphs.
4. **Fact-granting innate concepts** (moving `NoAllocation`, `Outlives` into
   Concept) are out of the MVP. They need the trust argument in the design
   doc applied to facts that license code generation.
5. **Guards on enum-pattern arms** (`Status::Ready(v) when v > 0 => ...`).
   Subject-less guarded match exists; pattern guards need binding scope in
   the guard and a lowering that falls through to the next arm.
6. **`Verdict<H, R>`** (typed evidence and refutations) needs templates at
   compile time (`CV4201: templates are not available during comptime
   evaluation`). Until then, a rule-local reason enum plus a `Describe`
   match gives the R half.
7. **Observation vocabulary.** `TypeShape` overlaps the boolean type
   observations (`IsStruct`, `IsArray`, ...). Decide whether to keep both or
   prune the booleans once rules use `Shape`.
8. **Comptime limits** (fuel 4096 per evaluation, loop bound 256, array 64,
   call depth 32) are untested on large programs. Innate evaluation runs per
   declaration under a process-wide mutex; measure on the largest library.

### D. Self-hosting and compile-time reflection

1. **The MachineIR bridge is written twice** (Go `EncodeMachineBridge`,
   Concept `Standard.Backend.AMD64` decoder). *Next:* derive codecs from one
   declaration with compile-time reflection over declarations.
2. **Growable storage.** `Standard/Backend/AMD64.concept` uses fixed arrays
   whose capacities disagree (128 virtual registers / 128 blocks in
   `BackendFunction`, 32 / 16 in liveness). Self-hosted compiler stages need
   caller-supplied arenas with per-phase bounds.
3. **Differential testing:** Go `cmd/concept/amd64.go` against the Concept
   backend, byte-for-byte over the corpus.
4. **Compile-time reflection over bytes:** `import kernel ScaleKernel from
   "kernels/scale.spv";` would replace `vulkan-bind`'s checked-in generated
   files. The content hash must enter module identity.
5. **Stage0 freeze:** decide when the Go compiler stops gaining features. The
   generated strict C11 can serve as the bootstrap seed.

### E. Vulkan library

Deferred, in rough order: typed views over mapped memory (a `layout` bound
over a mapped range, the same machinery `stream` uses), timeline semaphores,
a transfer queue, more than one context per process in the device runtime,
images and samplers, ray queries, FFT. A Concept-native SPIR-V profile is far
off; kernels come from glslc/dxc.

### F. Oct / Prometheus (pending with the owner)

- GPU run of `tools\prometheus_phase0\run_phase0.ps1` and merge of branch
  `claude/prometheus-phase0` in the Oct repo.
- Deferred audit items: descriptor reclaim (A1b), batch through the selector
  (A3), renames (A5), prestage decision, feedforward authority, unified
  cancel paths, registry drift, Phases 1–4. See
  `docs/design/PROMETHEUS-AUDIT-2026-09.md`.

### G. Known test and tooling noise

- `TestDifferentiatorStrictC11BothCompilers` fails where clang is absent.
- `TestNativeSchedulerWorkers` is flaky under load; it passes alone.
- `cmd/concept/amd64.go` is not gofmt-clean (pre-existing).

## Part 3: conventions

- **Line endings.** The Windows checkout uses CRLF. From Linux, run git as
  `git -c core.autocrlf=true`; edit CRLF files with a read-modify-write that
  preserves `\r\n`; check gofmt on an LF-normalized copy.
- **Corpus.** `language/evt1/<subsystem>/{valid,invalid}`; every new file
  updates `language/evt1/manifest.json` (subsystem and total counts). `concept
  check` alone must reject every static-invalid file.
- **Moving a rule into the innate module:** shadow (both run, agreement test
  over the corpus and targeted cases), switch (add the code to
  `evt1RetiredGoRules`), delete. One commit each.
- **Innate module:** `innate concept Name<FieldDeclaration F>` with
  `[[diagnostic("CODE")]]` and `requires Predicate(F);`; predicates are
  `comptime Verdict P(declaration d)`. Recursive helpers state `bounded(N)`.
- **GPU verification** runs on the owner's Windows machine:
  `powershell -ExecutionPolicy Bypass -File tools\vulkan\run_gpu.ps1`, log in
  `tools/vulkan/out/gpu.log`.
