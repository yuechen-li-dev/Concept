# EVT1 semantic subsystem index

This is the navigation map for R6 tooling. Semantic legality belongs to the
typed Module/MIR path; Planner validates and selects strategies from those
facts, and C11 output is derived evidence.

| Subsystem | Implementation | Normative spec | Design record | Corpus | Tests |
|---|---|---|---|---|---|
| Values | `types.go`, `parse.go`, `validate.go`, `generate.go` | sections 5-11 | phase 1-10 records | `language/evt1/values` | `values_records_test.go` |
| Ownership | `validate.go`, `generate.go` | sections 7-8 | `phase-10-ownership-move-drop.md` | `language/evt1/ownership` | `ownership_references_test.go` |
| Lifetimes | `validate.go`, `types.go` | section 8.5 | `EVT1-REFERENCE-AND-SPAN-DIRECTION.md` | `language/evt1/lifetime` | `lifetime_lexical_test.go`, `lifetime_relational_test.go` |
| Failure | `failure.go` | section 12 | `EVT1-FAILURE-MODEL.md` | `language/evt1/failure` | `failure_test.go` |
| Storage | `storage.go`, `layout.go`, `span.go` | section 16 | layout/storage/span direction records | `language/evt1/storage` | `storage_arrays_test.go`, `storage_binding_test.go`, `layout_stream_test.go`, `span_test.go` |
| Tensor | `tensor.go` | section 16.4 | tensor direction record | `language/evt1/tensor` | `tensor_semantics_test.go`, `tensor_backing_test.go` |
| Interfaces | `interface_dyn.go` | sections 13 and 19 | interface/dyn direction records | `language/evt1/interface` | `interface_dyn_test.go` |
| Automata | `automata.go` | section 22 | automata and decide direction records | `language/evt1/automata` | `automata_state_test.go`, `automata_transitions_test.go` |
| Inference | `inference.go` | section 22 | `EVT1-DECIDE-INFER-DIRECTION.md` | `language/evt1/inference` | `inference_test.go` |
| Iteration | `iteration_yield.go` | section 22 | iterator/foreach direction record | `language/evt1/iteration` | `iteration_yield_test.go` |
| Machine stack | `machine_stack.go` | section 22 | machine-stack direction record | `language/evt1/machine-stack` | `machine_stack_test.go` |
| Async | `async.go`, `async_control_flow.go` | section 22 | async direction record | `language/evt1/async` | `async_test.go`, `async_control_flow_test.go`, `async_interface_test.go` |
| Callables | `callable_capture.go`, `callable_types.go` | sections 26-27 | callable direction record | `language/evt1/callable` | `callable_capture_test.go`, `callable_types_test.go` |
| Semantic facts | `semantic_facts.go` | section 17 | facts/optimization direction record | `language/evt1/semantic-facts` | `semantic_facts_test.go` |
| Planner | `planner.go` | section 28 | Planner direction record | `language/evt1/planner` | `planner_test.go` |
| Composition | all semantic owners above | all relevant sections | R5k freeze report | `language/evt1/composition` | `semantic_corpus_test.go` |

The three explicit state-bearing forms remain distinct: `automata with state`
is application-authored persistent state, callable `with (...)` is an explicit
capture environment, and async continuation state is compiler-derived. They
share ownership, Drop, and provenance laws without sharing grammar or inventing
a second state runtime.

Planner families consume semantic facts for tensor strategy, bounds, cleanup,
failure, dispatch, layout/stream/Span, automata/machine stack, async, and
callables. Planner validation rejects impossible strategies; it never creates
language legality.
