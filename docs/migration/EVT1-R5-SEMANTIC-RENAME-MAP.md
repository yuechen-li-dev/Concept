# EVT1 R5 semantic rename map

This archaeology map records the R5k move from chronological names to active
semantic architecture. Historical conformance filenames and commit messages
remain unchanged.

| Historical active path | Current active path |
|---|---|
| `internal/concept/r4k.go` | `internal/concept/interface_dyn.go` |
| `internal/concept/r5d.go` | `internal/concept/iteration_yield.go` |
| `internal/concept/r5e.go` | `internal/concept/machine_stack.go` |
| `internal/concept/r5f.go` | `internal/concept/async.go` |
| `internal/concept/r5g.go` | `internal/concept/async_control_flow.go` |
| `internal/concept/r5i.go` | `internal/concept/callable_capture.go` |
| `internal/concept/r5j.go` | `internal/concept/callable_types.go` |
| `internal/concept/r2_conformance_test.go` | `internal/concept/values_records_test.go` |
| `internal/concept/r3_conformance_test.go` | `internal/concept/ownership_references_test.go` |
| `internal/concept/r4a_conformance_test.go` | `internal/concept/lifetime_lexical_test.go` |
| `internal/concept/r4b_conformance_test.go` | `internal/concept/lifetime_relational_test.go` |
| `internal/concept/r4c_conformance_test.go` | `internal/concept/failure_test.go` |
| `internal/concept/r4d_conformance_test.go` | `internal/concept/storage_arrays_test.go` |
| `internal/concept/r4e_conformance_test.go` | `internal/concept/storage_binding_test.go` |
| `internal/concept/r4f_conformance_test.go` | `internal/concept/layout_stream_test.go` |
| `internal/concept/r4g_conformance_test.go` | `internal/concept/span_test.go` |
| `internal/concept/r4h_conformance_test.go` | `internal/concept/tensor_semantics_test.go` |
| `internal/concept/r4i_conformance_test.go` | `internal/concept/tensor_backing_test.go` |
| `internal/concept/r4j_conformance_test.go` | `internal/concept/semantic_facts_test.go` |
| `internal/concept/r4k_conformance_test.go` | `internal/concept/interface_dyn_test.go` |
| `internal/concept/r4l_conformance_test.go` | `internal/concept/planner_test.go` |
| `internal/concept/r5a_conformance_test.go` | `internal/concept/automata_state_test.go` |
| `internal/concept/r5b_conformance_test.go` | `internal/concept/automata_transitions_test.go` |
| `internal/concept/r5c_conformance_test.go` | `internal/concept/inference_test.go` |
| `internal/concept/r5d_conformance_test.go` | `internal/concept/iteration_yield_test.go` |
| `internal/concept/r5e_conformance_test.go` | `internal/concept/machine_stack_test.go` |
| `internal/concept/r5f_conformance_test.go` | `internal/concept/async_test.go` |
| `internal/concept/r5g_conformance_test.go` | `internal/concept/async_control_flow_test.go` |
| `internal/concept/r5h_conformance_test.go` | `internal/concept/async_interface_test.go` |
| `internal/concept/r5i_conformance_test.go` | `internal/concept/callable_capture_test.go` |
| `internal/concept/r5j_conformance_test.go` | `internal/concept/callable_types_test.go` |

Corpus directories moved one-to-one from `language/evt1-r1/core` through
`language/evt1-r5j/core` to the semantic paths recorded by
`language/evt1/manifest.json`. There is no duplicated fixture tree. The exact
milestone-to-path mapping lives in each manifest entry's
`historical_milestones` field and in the conformance chronology index.
