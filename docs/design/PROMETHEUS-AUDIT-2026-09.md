# Prometheus audit: preparing the Concept rewrite

Date: 2026-09-30. Snapshot: Oct `a8bbdd60` (2026-09-28), `internal/prometheus/`.
Scope: native reactor, SDSL-V production kernels, shader registry and package,
public ABI and consumers, and readiness of Concept + `libraries/Vulkan`.
Every number below was measured on that snapshot; the method and caveats are at
the end.

## Verdict

The architecture is sound and should be kept: *Dominatus decides, mechanisms
execute and report facts*. What has decayed is enforcement, and the size
problem is mostly not Vulkan ceremony.

1. **The controller governs one path.** All 132 Dominatus/judgment call sites
   are in `reactor_vulkan_sgemm.c`. The production batch engine, reduction,
   FFT, ray query, transformer and model block make zero controller calls.
2. **38% of native function code is unreachable from the public ABI.**
   15,346 of 40,139 function lines are reached only by Marionette tests;
   11,490 of them are in `reactor_vulkan_transformer.c` (M42–M49 executors).
3. **The Z-Image model plan is hard-coded inside a mechanism file.** This is
   the missing Dominatus seam, and it is also where the memory efficiency
   actually comes from.
4. **Model kernels are hand copies of existing kernel families.** The four
   Z-Image GEMMs replicate `reg2x2_tile16x16` by hand, even though SDSL-V
   already has `concept`/`config`/`template` and production SGEMM uses them.
5. **Kernel management is 80% built.** A content-addressed package exists and
   production already loads SPIR-V from it. The hand-authored manifest, the C
   metadata copy, and 22.7k lines of test-only `*_spirv.h` are leftovers.

Consequence for the rewrite: porting as-is to Concept would carry about 15k
dead lines and a model plan into the new code. Delete and extract first, then
port what remains. The direct Concept win on mechanism ceremony is real but
moderate (roughly a quarter of mechanism lines, see F6); the large reductions
come from F2 and F3.

## Sizes

| Area | Lines |
| --- | ---: |
| Hand-written native C/C++/H (55 top-level files) | 54,661 |
| Mechanism `.c` (`reactor_vulkan_*.c`, `reactor_batch.c`) | 34,008 |
| Control `.c` (Dominatus, judgment, slot HFSM, policy memory) | 5,940 (+2,246 `.h`) |
| Public header `reactor_api.h` / internal `reactor_vulkan.h` | 2,824 / 3,636 |
| Embedded SPIR-V headers (58 files) | 22,694 |
| Marionette tests (464 registered facts/tests/benchmarks) | 33,603 |

## Findings

### F1. The controller does not govern production outside single SGEMM

- `prom_reactor_runtime_sgemm_impl` passes `selector_controls_dispatch_variant = 1`,
  so the occupancy selector does choose the variant on the sync path. The
  earlier "decision computed, never read" bug is fixed there.
- `reactor_batch.c:173-176` plans every batch entry as `PROM_VK_PATH_DIRECT`,
  `PROM_VK_COMPUTE_BASELINE`, `requested_variant = 0` ("M31's current real path
  is the existing direct baseline dispatch"). The README calls this "the sole
  production batch engine".
- Controller calls per file: `reactor_vulkan_sgemm.c` 132; `reactor_batch.c`,
  `fused_reduction`, `fft`, `ray_query`, `transformer`, `model_block` all 0.
  The transformer has its own 155-line `reactor_vulkan_transformer_control.c`,
  separate from Dominatus.
- Direction is inverted even where it works: the mechanism file calls the
  judgment engine and drives the slot HFSM, instead of the controller issuing
  committed work to a mechanism.
- P15 shadow machinery is default-off (`p15_shadow_canary_enabled`), and the
  public ABI exports a test seed
  (`prometheus_reactor_runtime_p15_test_seed_matured_reservation`).

### F2. Test-only code: 15.3k function lines unreachable from the ABI

A call graph over the production translation units, rooted at the 84 exported
`PROM_REACTOR_API` functions, leaves 292 of 1,007 functions unreachable:

| File | Unreachable lines |
| --- | ---: |
| `reactor_vulkan_transformer.c` | 11,490 |
| `reactor_numerical_research.c` | 1,292 |
| `reactor_vulkan_fused_reduction.c` | 936 |
| `reactor_vulkan_sgemm.c` | 713 |
| everything else | 915 |

The largest: `m48_execute_stack` (836), `m45_execute_composed_core` (761),
`m44_execute_composed` (467), `m43_execute` (390), `m42_execute` (343),
`m49a_execute_ffn_suffix` (333), `m40b_execute` (226). Each is referenced by
exactly one Marionette file. They are milestone scaffolds that each added a new
executor instead of evolving one, and they stayed alive through their tests.

Control layer: 24 of 145 exported control functions are called only by tests.
The telling ones are `prom_slot_hfsm_push_state`/`pop_state` (the slot HFSM
runs flat in production; the hierarchy is never used),
`prom_dominatus_reservation_cancel` (cancellation never happens in production),
and `prom_judgment_engine_select_sgemm_mode` (superseded by the layout-precision
variant).

### F3. The model plan lives in the mechanism layer

- `prometheus_reactor_runtime_compiled_model_retarget` →
  `reactor_vulkan_model_block.c:5839-5870`: an `if` chain over
  `retarget_position`. Positions 0–1 are noise refiners, 2–3 context refiners,
  and 4–33 are the 30 main-transformer layers. Each branch hard-codes family,
  parameter set, expected weight counts and byte tables.
- This is where the "insane" memory efficiency comes from: **one resident model
  block is retargeted across 34 positions**, with weights streamed through one
  upload buffer and prefetch/activate for overlap. That is a *plan* property:
  arena reuse plus a weight-streaming schedule. It is not a property of
  per-model kernel source, which confirms the kernel-family direction in F4.
- Ownership smell: the model block and compiled session are fields of
  `prom_reduction_runtime_state`, reached through
  `prom_reactor_runtime_reduction_state(handle)`.
- Public ABI: 85 exported symbols, of which about 26 are Z-Image or Gemma
  specific (`noise_refiner*`, `context_refiner*`, `main_transformer*`,
  `compiled_model_*`, `gemma4e2b_m1_*`), and `ModelBlockM1B/M1C/M1D` request
  types. `reactor_api.h` contains 353 distinct milestone-named identifiers.
  Across native sources there are 3,669 occurrences of milestone-named
  identifiers (674 distinct), e.g. `p13_m16b1_requested_occupancy_variant`.
  Names that encode history rather than meaning are the main reason Codex
  cannot navigate this code.

### F4. Model kernels are instances of families that already exist

The 24 Z-Image production kernels fall into six families:

| Family | Kernels | Notes |
| --- | --- | --- |
| GEMM + prologue/epilogue | `nr0_fused_qkv`, `nr0_attention_projection`, `nr0_ffn_w1_w3`, `main_transformer_ffn_w1_w3`, `nr0_ffn_w2_residual`, `nr0_adaln` | The first four use `reg_tile<f32,2u,2u>` + 16×16 tiles: a hand copy of `sgemm_reg2x2_tile16x16` with K/N/stride constants (240u, 3840u, 11520u) and an FP16x2 weight-load prologue |
| Elementwise / convert | `nr0_ffn_gate`, `main_transformer_ffn_gate`, `nr0_attention_residual`, `nr0_bf16_ingress` | Gate, residual, dtype widening |
| Norm + modulate | `nr0_attention_norm_modulate`, `nr0_ffn_norm_modulate` | Row RMSNorm with modulation |
| Head norm + RoPE | `nr0_q_norm_rope`, `nr0_k_norm_rope`, `main_transformer_joint_qk_norm_rope`, `context_refiner_qk_norm_rope` | Differ by segment offset and position frame; Gemma's `rope_half_split` is a layout variant |
| Streaming attention | `nr0_attention_streaming`, `_parallel`, `main_transformer_joint_attention_streaming`, `_builtin_topology`, `_subgroup_owned32`, `context_refiner_attention_streaming` | Differ by token count (32/1024/1056), subgroup topology, and 1/√128 |
| Audit / probe | `nr0_persistent_audit_summary`, `main_transformer_subgroup_owned32_topology_probe` | Not compute |

SDSL-V already supports `concept RegBlockedSgemmConfig`,
`config Reg2x2Tile16x16Fp32`, `template<C: …> shader`, `static assert` and
`comptime for`. The production SGEMM kernels use them; the model GEMMs do not.
What the config cannot yet express: **fixed shape constants** (K, N, leading
dimensions), **operand encoding** (F32, F16x2, BF16 packed loads), and an
**epilogue hook** (residual, bias, gate, Q/K/V split).

### F5. Kernel management: the parts exist, authority is split three ways

- `native/shaders/manifest.json`: 69 assets + 11 experimental + 18 compute
  implementations; 51 distinct asset keys; 1,063 asset fields. Of those,
  321 are derivable by SPIR-V reflection (bindings, push-constant bytes,
  workgroup, capabilities, subgroup requirements, entry point), 310 are build
  recipe (header, symbol, inline-HLSL counts, targets), and 68 are selector
  policy (`selector_eligible`, `default_route_eligible`, `benchmark_enabled`,
  `dispatchable`, …).
- `reactor_shader_registry.c` holds a second hand-maintained copy of the asset
  metadata as `META_DETAIL(...)` rows.
- The content-addressed package already exists: `prometheus.shader-package.v1`,
  objects stored at `objects/sha256/<sha256>.spv` (Go `shaderpackage`, C
  `reactor_shader_package.c`), and the runtime accepts
  `PrometheusReactorConfig.shader_package_root`.
- No production `.c` file includes a `*_spirv.h`. The 58 headers (22,694 lines)
  are consumed only by 23 includes in Marionette tests.
- The generator is `generate_sdslv_shaders.ps1` (Windows-only), driven by the
  manifest.

### F6. Mechanism layer: what Concept would actually remove

Heuristic line classification of the mechanism files (34k `.c` + internal
headers, 39,304 lines):

| Category | Share |
| --- | ---: |
| Vulkan calls | 2.5% |
| Vulkan struct fills (`Vk*Info`, barriers, writes) | 4.6% |
| Error and cleanup plumbing | 9.0% |
| Diagnostics and telemetry | 6.8% |
| Declarations, includes, macros | 14.2% |
| Blank, comment, brace | 12.2% |
| Other logic (validation, planning, CPU references, selection, math) | 50.7% |

So direct ceremony (calls, struct fills, cleanup, diagnostics fills) is about
23%. Ownership, Drop, `?`, generated descriptors and derived barriers remove
most of that. Most of the "logic" bucket is F2 dead code and F3 plan code,
which should be deleted or extracted rather than ported.

Specific mechanism debts:

- 71 `vkCmdPipelineBarrier` calls and 40 `VkBufferMemoryBarrier` fills, all by
  hand. There is no synchronization2 (`VkMemoryBarrier2`) even though
  Vulkan 1.4, which makes it core, is the required floor.
- 31 `vkCreateComputePipelines` calls and no `VkPipelineCache`.
- 95 `vkCmdCopyBuffer` (staging), 72 `vkCmdWriteTimestamp`, 33
  `vkCmdPushConstants`, no specialization constants, no timeline semaphores.
- Buffer creation is already centralized: one `vkCreateBuffer`, one
  `vkAllocateMemory` (the Stage 5 helper). Keep this shape.
- `prom_reactor_runtime_sgemm_impl_with_variant` is a single 1,693-line
  function; `sgemm_policy_diagnostics_fill` is 640 lines of field copies.

### F7. Consumers

- The Oct Go bridge (`bridge.go`) resolves 14 symbols: create, destroy, probe,
  sync SGEMM, 4 async SGEMM, and 6 Gemma M1 operations.
- Z-Image is consumed only through `tools/prometheus_zimage_bridge`.
- Batch, reduction, FFT and ray query have no Go consumer; they are reached by
  Marionette and by tools (`tools/prometheus_ray_query_camera_diagnostic`).

The stable contract the rewrite must preserve is therefore small. Everything
else is free to reshape.

### F8. Concept readiness

Available today: owned Context, Buffer, ComputePipeline and Submission with
Drop; `extern "C" handle`; step machines with must-use `StepOutcome`;
`Outbox<T,N>`; `layout`/`stream` (which parallel SDSL-V `stream`);
`Span<T>`/`ReadOnlySpan<T>` over existing storage; host allocation
(`Standard.Memory.Host`), bump allocation, and `template <typename T>`.
It was verified on a GPU in VK8.

Gaps, in the order Prometheus needs them:

1. **Typed runtime-length views over foreign memory.** Caller `float*` + count,
   and mapped device memory. The spec lists "explicit external
   runtime-array/ndarray descriptors" as deferred (§29). This is the one
   language-level blocker; the rest is library work.
2. Typed buffers with placement (device-local vs host-visible), mapped views,
   staging copy, fill.
3. A command recording scope with many dispatches, copies and timestamps per
   command buffer, with sync2 barriers derived from declared access.
4. Kernels loaded from the package by key, a reflected binding record, typed
   push constants, and a pipeline cache.
5. Submission rings: fences, reusable slots, async tokens as step machines.
6. Capability negotiation: feature and extension chains (subgroups, fp16,
   cooperative matrix) under the Vulkan 1.4 floor.
7. Later: transfer queue, ray query and acceleration structures.

DragonGod (1,828 lines: scheduling, actuation claims, replay, trace, events)
has the authorization, commitment and replay vocabulary Dominatus needs. It
has no blackboard, measurement filters, predictor/reservations, or
selection/judgment yet.

## Recommended actions

Ordered, with an expected effect for each. Phase 0 is independent of Concept
and pays off immediately.

### Phase 0: cut and freeze, in C (before any port)

| # | Action | Effect |
| --- | --- | --- |
| A1 | Delete the unreachable M42–M49 executors, M40b and numerical-research paths, with their Marionette files; keep the reports as history | About −15k native lines, −11.5k from the transformer file |
| A2 | Freeze the public ABI to what is consumed (F7) plus the Z-Image session; remove the P15 test seed from the public header | Smaller contract to reproduce in Concept |
| A3 | Decide batch authority: route batch planning through the selector, or declare batch baseline-only in the contract | Ends the silent split between "production" and "governed" |
| A4 | Delete test-only control functions or give them production callers (slot HFSM push/pop, reservation cancel, superseded `select_sgemm_mode`) | Control layer means what it says |
| A5 | Rename milestone identifiers at the boundaries that survive (public header, controller facts) | Code that Codex can navigate |

### Phase 1: kernel families and a single kernel authority

| # | Action | Effect |
| --- | --- | --- |
| K1 | Extend `RegBlockedSgemmConfig` with fixed shape constants, operand encoding and an epilogue hook; re-express the 4 Z-Image GEMMs as instances; check outputs against the current SPIR-V on the same inputs | First proof that a model is a set of family instances |
| K2 | Family templates for head-norm+RoPE, norm+modulate, elementwise/convert, and streaming attention (token count and topology as config) | Covers 22 of 24 Z-Image kernels plus Gemma |
| K3 | Make the package the only authority: generate its index from SDSL-V sources plus SPIR-V reflection; use stable string keys (`sgemm/reg2x2_tile16/f16x2+residual`) | Replaces 1,063 hand-authored manifest fields |
| K4 | Move selector policy fields to controller configuration; delete `reactor_shader_registry.c` metadata rows, `reactor_shader_ids.generated.h` and the 58 `*_spirv.h`; migrate the 23 test includes to the package | −22.7k generated lines, one authority |
| K5 | Replace `generate_sdslv_shaders.ps1` with a cross-platform build step (`oct make` target or a Concept native-project kernel target) | Builds on Linux and in CI |

### Phase 2: extract the model plan

| # | Action | Effect |
| --- | --- | --- |
| P1 | Express Z-Image as data: 34 positions → family instances, weight tables, arena layout, streaming and prefetch schedule | The retarget `if` chain becomes a plan the controller walks |
| P2 | Give progression to the control layer (the missing Dominatus seam); the mechanism only executes committed positions | Restores decide/execute at the model level |
| P3 | Separate model-block state from reduction state | Clean ownership before porting |

### Phase 3: Concept libraries (next turn)

Build in `libraries/Vulkan` (and `Standard` where general), each with test-device
facts and a GPU pass through `tools/vk8`:

1. L1 foreign and mapped typed views (the language gap in F8.1)
2. L2 `Buffer<T>` with placement, staging copy and fill
3. L3 recording scope with declared access → sync2 barriers, timestamps
4. L4 package kernels by key, reflected binding records, typed push constants,
   pipeline cache
5. L5 submission rings and async tokens as step machines
6. L6 capability negotiation

### Phase 4: port behind the C ABI (strangler)

Concept emits strict C11 behind the same `prometheus_reactor_*` symbols, so the
Go bridge does not change and Marionette stays the oracle. Suggested order:

1. runtime/context and the Stage 5 buffer helper
2. fused reduction (2.7k lines, self-contained, no controller calls) as the
   first full family port
3. SGEMM mechanism, with the controller moved out of it
4. model block, executing the Phase 2 plan

In parallel: Dominatus → DragonGod, adding a typed blackboard, filters,
predictor/reservations and selection as step machines on top of the existing
actuation and replay vocabulary.

## Owner decisions (2026-09-30)

Governing rule: 5S. Sort and remove what is not needed first, then set in
order, then port. Nothing is kept "just in case".

1. **A1: delete and extract.** Unreachable milestone executors are deleted,
   not archived on a tag. Their DevelopmentReport entries stay as history.
   Anything worth keeping is extracted into a named, reachable form first.
2. **A3: batch is planned through the selector.** Per-entry decisions reach
   the batch plan. **A4 is widened:** the shadow HFSM machinery gets
   production callers (lifecycle advanced in production), not test-only
   drivers. Authority still passes through the canary gate.
3. **Port scope:** ray query and FFT are in scope but deferred. They stay in C
   until the SGEMM, reduction and model-block ports land.
4. **K5: the kernel build is an `oct make` target.** SDSL-V stays in Oct;
   Concept consumes the package. A Concept-native `profile spirv` is far
   future.

## Phase 0 status (first pass, 2026-09-30)

Branch `claude/prometheus-phase0` in the Oct repository, four commits on top
of `a8bbdd60`:

| Commit | Change |
| --- | --- |
| `ab0c9816` | A1: delete code unreachable from the reactor ABI |
| `4c7b9794` | A2 + A4: shadow controller on by default; test seed leaves the public header |
| `1ce34dc2` | Rename `reactor_vulkan_transformer.c` → `reactor_vulkan_gemma4e2b.c` (it now holds only the Gemma M1 path) |
| `e9ed0411` | `tools/prometheus_phase0/run_phase0.ps1`: baseline-vs-branch GPU comparison |

| Measure | Before | After |
| --- | ---: | ---: |
| Hand-written native C/C++/H | 54,661 | 38,325 |
| Mechanism `.c` | 34,008 | 21,232 |
| Function lines unreachable from the ABI | 15,346 | 1,405 (all retained instruments) |
| Marionette lines | 33,603 | 26,152 |
| Internal types removed | | 102 |

What was done:

- **A1.** The M42–M49b transformer executors, the M49b numerical shadow
  (`reactor_numerical_research.*`, `reactor_vulkan_transformer_control.*`),
  the M40b experiment, and the orphaned Concept/Vulkan kernel54 set were
  deleted, together with 58 tests that only exercised them. The M49b shadow
  could only run inside the deleted M48 stack, so it could not be switched
  on; it is deleted rather than kept off.
- **Kept as instruments:** SGEMM placement benchmark, audit harness,
  model-block fault injection, blackboard/HFSM observers, the M46 RMSNorm CPU
  oracle, and the Gemma M46 hardware proof. Test-only compositions moved to
  `Marionette/reactor_test_compositions.h` and `reactor_test_numerics.h`.
- **A4.** The P15 shadow controller (canary, authority gate, agree-and-confirm
  feedforward) runs by default; `PrometheusReactorConfig.p15_shadow_disabled`
  opts out. It cannot override the judgment-selected variant.
- **A2.** `prometheus_reactor_runtime_p15_test_seed_matured_reservation` is no
  longer exported.
- **5S outside git:** 75 ignored `.obj`/probe files were removed from the Oct
  root, along with the untracked `internal/conceptvulkan/generated` leftover.

Verification: a Linux build against a stub Vulkan loader (every entry point
fails cleanly), with a per-test comparison to the pre-change baseline. There
were 0 regressions; the runtime-level P15 tests need a device.
**GPU verification is pending:** run `tools\prometheus_phase0\run_phase0.ps1`
on Windows.

Deferred (listed, not started):

1. **A1b: reclaim resources of deleted executors.** Reduction slots still
   allocate descriptor sets for the M48 stack (4 layers × M43–M47 sets) and
   keep `m48_layer`/`m48_descriptors` fields that only cleanup touches. This
   changes descriptor-pool arithmetic, so it needs GPU validation.
2. **A3: batch planned through the selector.** Per-entry variants need
   per-slot pipeline binding and mixed variants within one submit.
3. **A5: milestone identifiers** at surviving boundaries (the public header
   still carries hundreds; `m46`/`m48` names now sit inside the live Gemma
   path).
4. **Prestage scaffold (P15 M6):** evaluate-only, with no action path. Either
   build the pre-transfer action or delete it; it cannot simply be turned on.
5. **Feedforward authority:** the shadow controller now runs, but only in
   agree-and-confirm mode. Letting it steer (override) is the real authority
   transfer and should be its own milestone.
6. **Two cancel paths:** `prom_dominatus_reservation_cancel` (tested
   primitive) and the correction path cancel differently. Unify them in the
   DragonGod port.
7. **Registry drift (pre-existing):** shader-registry metadata tests fail
   without a device, e.g. `PrometheusShaderRegistryIdsAreUnique` expects 40
   package-only assets and finds 44. This is evidence for K3/K4 (one kernel
   authority).
8. Phase 1–4 (K1–K5, P1–P3, L1–L6, the port) as above.

## Method and caveats

- Reachability: brace-matched function extraction over production `.c`/`.cpp`
  in `native/` (tests, `sdslv_test_host.c` and the M40a probe excluded), with
  a call graph by identifier reference rooted at `PROM_REACTOR_API`
  declarations. Calls through function pointers or macros would be missed;
  none were found in the transformer file, and the top unreachable functions
  were checked by hand to be referenced only by their definition and one
  Marionette file.
- Line classification (F6) is regex-based and approximate. Use it as
  proportions, not exact counts.
- The kernel family table (F4) is from file names, header comments, literals,
  and a full read of `nr0_fused_qkv`. Confirm each family boundary during K1/K2.
- "Consumers" (F7) is a `git grep` of symbol names across Go, C and C++ in the
  Oct repo; external consumers outside the repo are not covered.
- Scripts used: `classify.py`, `funcs.py`, `reach.py` (kept in the audit
  session's scratch space; they can be checked in on request).
