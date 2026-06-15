# Concept source formatting inventory

Concept source should be human-readable.

`.concept` and `.conception` files should use C++-like indentation, line breaks, and spacing.

LLM readability is not the only readability target.

Human reviewers should not have to read compressed TypeScript-like one-liners.

This inventory is human process guidance. P21-M9 does not implement a formatter, does not add `concept fmt`, and does not mass-reformat the repository. A real formatting tool and formatter/linter policy are deferred.

## Desired style

Bad/compressed:

```cpp
int main(){int[3] values=[1,2,3];return values[1];}
```

Good:

```cpp
int main() {
    int[3] values = [1, 2, 3];

    return values[1];
}
```

Style expectations:

- Use 4-space indentation, or the nearest project-standard indentation when editing existing files.
- Keep opening braces on the same line for functions, structs, and blocks where existing Concept style does so.
- Prefer one statement per line.
- Put spaces around binary operators and assignment.
- Put spaces after commas.
- Split nontrivial multi-field struct literals across lines.
- Prioritize human readability in fixtures even when compact forms compile.
- Keep backend assertion blocks readable; dense assertions should be grouped and named rather than compressed.

## Representative formatting inventory

The table is intentionally representative rather than exhaustive. A full repo-wide formatter/linter is future work.

| File | Issue | Suggested cleanup |
| --- | --- | --- |
| `language/phase20-dragongod-kernel/valid/kernel_checkpoint_restore_all_run.valid.conception` | Very dense checkpoint fixture with many multi-statement lines and long restore setup. | Split setup, restore, and assertions into readable blocks before future checkpoint edits. |
| `language/phase20-dragongod-kernel/valid/kernel_checkpoint_events_restore_run.valid.conception` | Compressed persistence/event setup and assertions. | Use one statement per line and group event fixture phases. |
| `language/phase20-dragongod-kernel/valid/kernel_checkpoint_actuation_restore_run.valid.conception` | Dense actuation checkpoint construction and status checks. | Expand constructor calls and status assertions. |
| `language/phase20-dragongod-kernel/valid/kernel_replay_determinism_run.valid.conception` | Replay scenario is readable to the compiler but compressed for humans. | Split replay log construction, driver setup, apply, and assertions. |
| `language/phase20-dragongod-kernel/valid/kernel_replay_trace_rng_agent_run.valid.conception` | Multi-subsystem replay scenario packs too much state into compact lines. | Use section-like blank lines and one statement per line. |
| `language/phase20-dragongod-kernel/valid/kernel_checkpoint_backend_shape.valid.conception` | Backend assertion fixture contains dense source and assertion blocks. | Preserve assertions but split source snippets and forbidden-shape checks into readable groups. |
| `language/phase20-dragongod-kernel/valid/kernel_graph_decision_integration_run.valid.conception` | Integration fixture has long graph/decision setup lines. | Break graph node setup and decision assertions into separate lines. |
| `language/phase20-dragongod-kernel/valid/kernel_graph_mind_root_goto_run.valid.conception` | Dense graph-to-Mind smoke path. | Expand setup and assertion phases. |
| `language/phase20-dragongod-kernel/valid/kernel_automata_stack_push_pop_run.valid.conception` | Stack fixture is historically compact and now blocks easy comparison with future FixedBuffer pop migration. | Reformat before or during any future stack migration. |
| `language/phase21-arrays-slices-fixed-buffers/valid/fixed_buffer_append_read_run.valid.conception` | Newer fixture is compact but small. | Prefer expanded C++-style blocks for future Phase 21-adjacent fixtures. |

## Future formatter direction

A future `concept fmt` should be designed explicitly rather than inferred from compressed historical fixtures. It should preserve semantic fixture expectations, avoid rewriting expected diagnostics accidentally, and probably start as an opt-in check over new/changed `.concept` and `.conception` files before becoming a whole-repository tool.
