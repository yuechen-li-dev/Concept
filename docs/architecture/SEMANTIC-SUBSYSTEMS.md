# EVT1 semantic subsystem index

This is the navigation map for R6 tooling. Semantic legality belongs to the
typed Module/MIR path; Planner validates and selects strategies from those
facts, and C11 output is derived evidence.

R6h gives logical value transport to one `SemanticValueFacts` record. Existing
Span and Tensor validators remain source authorities for their domain facts and
bridge into that record. Module summaries, persistent-field projections, proof
graphs, and Planner input consume it; none may reconstruct or strengthen facts.
Value facts and operation effects remain distinct, and all transport metadata
is erased before C lowering.

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
| Semantic facts | `semantic_facts.go`, `fact_transport.go` | section 17 | `EVT1-SEMANTIC-FACT-TRANSPORT-DIRECTION.md` | `language/evt1/semantic-facts`, `language/evt1/tooling/semantic-facts` | `semantic_facts_test.go`, `fact_transport_test.go` |
| Planner | `planner.go` | section 28 | Planner direction record | `language/evt1/planner` | `planner_test.go` |
| Composition | all semantic owners above | all relevant sections | R5k freeze report | `language/evt1/composition` | `semantic_corpus_test.go` |
| Test tooling | `testing_metadata.go`, `testing_assert.go`, `testing_lowering.go`, `testing_runner.go` | section 24 | `EVT1-TEST-PROPHECY-DIRECTION.md` | `language/evt1/tooling/tests`, `internal/concept/testdata/testing` | `testing_runner_test.go` |
| Concept assertions and proof debugging | `concept_assert.go`, `proof_graph.go` | sections 17 and 24 | `CONCEPT-PROOFS.md` | `language/evt1/tooling/proofs` | `proof_graph_test.go` |
| Generic runtime types and layout queries | `generic_types.go`, `layout.go` | post-freeze R6d extension | `EVT1-GENERIC-LIBRARY-DIRECTION.md` | focused R6d specimens | `operation_effects_test.go` |
| External C ABI | `parse.go`, `validate.go`, `generate.go` | post-freeze R6d extension | `EVT1-GENERIC-LIBRARY-DIRECTION.md` | focused R6d hosted specimen | `operation_effects_test.go` |
| Operation effects | `concept_assert.go`, `semantic_facts.go` | post-freeze R6d extension | `EVT1-GENERIC-LIBRARY-DIRECTION.md` | R6b proofs plus focused R6d specimens | `operation_effects_test.go`, `proof_graph_test.go` |
| Reusable semantic modules | `module_artifact.go`, `parse.go` | post-freeze R6e extension | `EVT1-SEMANTIC-MODULE-DIRECTION.md` | `language/evt1/tooling/modules` | `module_artifact_test.go` |
| Cross-module generics and effects | `generic_types.go`, `concept_assert.go`, `validate.go` | post-freeze R6e extension | `EVT1-SEMANTIC-MODULE-DIRECTION.md` | `language/evt1/tooling/modules` | `module_artifact_test.go` |
| Quantities and units | `quantity.go`, `parse.go`, `validate.go` | `QUANTITIES-AND-UNITS.md` | `EVT1-QUANTITY-ADDRESS-STORAGE-DIRECTION.md` | focused R6g specimens | `r6g_quantity_address_test.go` |
| Address and typed storage | `quantity.go`, `validate.go`, `generate.go` | `ADDRESS-AND-STORAGE.md` | `EVT1-QUANTITY-ADDRESS-STORAGE-DIRECTION.md` | `Standard.MemoryGeometry` | `r6g_quantity_address_test.go` |
| Standard Library: Memory, Memory Sources, Allocators, Typed Allocation Owners | not implemented | direct initialized ownership works; blocked on nested generic Result/Option transport | `EVT1-ALLOCATOR-DIRECTION.md`, `EVT1-INITIALIZED-STORAGE-OWNERSHIP-DIRECTION.md` | none | `r6l_initialized_storage_test.go` |

The three explicit state-bearing forms remain distinct: `automata with state`
is application-authored persistent state, callable `with (...)` is an explicit
capture environment, and async continuation state is compiler-derived. They
share ownership, Drop, and provenance laws without sharing grammar or inventing
a second state runtime.

Planner families consume semantic facts for tensor strategy, bounds, cleanup,
failure, dispatch, layout/stream/Span, automata/machine stack, async, and
callables. Planner validation rejects impossible strategies; it never creates
language legality.

Test attributes are source-ordered Module/MIR metadata. The runner consumes a
static manifest and generated C11 symbols; it does not add runtime reflection,
a registry, or a second evaluator. Assertion calls retain ordinary expression
semantics and evaluate each value argument exactly once.

Proof tooling consumes semantic authority. `Assert.Concept`, human diagnostics,
the explain CLI, JSON, and repairs project existing facts, requirements, and
provenance; they do not consult Planner or re-derive legality.
