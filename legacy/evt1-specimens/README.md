# EVT1 M-era specimens (frozen)

Snapshot of the paired `examples/evt1` specimens and their checked compiler
outputs, taken at VK0 of the Concept Vulkan reconciliation
(`docs/design/CONCEPT-VULKAN-RECONCILIATION-LADDER.md`), before the signal
automata dialect (`automata X(Signal, borrow context: T)`, `on … =>`,
`dispatch`, `effect`, `effects`, `actuator`, `initial`/`finish`) is ported to
step machines and removed.

Nothing here is compiled or tested. Files are never edited; the live
counterparts are in `examples/evt1/`.

- `sources/`: the 20 specimens exactly as they were (all `profile Vulkan`).
- `generated/`: the 90 checked outputs from `internal/concept/generated/`
  (`lifecycle_actuators_*` was never in the checked set).

| Specimen | Original milestone name | Subject |
| --- | --- | --- |
| `extern_calls_and_match` | `evt1_m1a` | `extern "C"` calls, `match` |
| `resource_concepts` | `evt1_m1b_a` | structs, concepts over resources |
| `concept_templates` | `evt1_m1b_b` | templates with concept requirements |
| `comptime_values` | `evt1_m1b_c` | compile-time values |
| `comptime_tables` | `evt1_m1b_d` | compile-time tables, array indexing |
| `lifecycle_automata` | `evt1_dragongod_m0` | step automata |
| `automata_dispatch` | `evt1_dragongod_m1` | signal `dispatch` outcomes |
| `guarded_transitions` | `evt1_dragongod_m2` | `on … when … =>`, `otherwise` |
| `lifecycle_effects` | `evt1_dragongod_m3` | `effect`, `emit`, effect batches |
| `lifecycle_actuators` | `evt1_dragongod_m4` | `actuator` mappings |

`*_core` was the `_language` half of each pair (renamed during the repository
reorganization); `*_vulkan` binds the same logic to Vulkan handle types.
