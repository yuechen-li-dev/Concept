# R8e MustUse and explicit discard (implemented core seam)

`[[must_use]]` is declaration metadata on a value type or value-returning
function. It is independent of a type's spelling. Standard `Result<T, E>`
receives the same metadata from its compiler-owned declaration factory.
Ignoring a MustUse value in an expression statement is a semantic error;
binding, returning, propagating, or explicitly discarding it handles the value.
The diagnostic is unconditional, not a project-style severity setting.

`discard expression;` evaluates the expression once and explicitly ignores
its result. The statement is represented as an expression statement with a
discard marker, and MIR emits `discard_value`. C11 lowering evaluates once
and invokes ordinary Drop for a returned owned value that needs it. Calls and
their effects remain in the generated program. Existing `discard(batch)`
effects calls remain valid; the parenthesized spelling selects the existing
operation, while `discard expression;` selects the new statement.

The semantic module payload retains type and function attributes. Artifact
consumers enforce MustUse without reading provider source. This is verified
for an authored type, a value-returning function, and a foreign `extern "C"`
declaration. Generic returns are checked after closing the result type.

Current boundary: this implements the correctness-significant MustUse seam.
Project policy concepts, manifest severity binding, `concept lint`,
`concept fmt`, and policy explanations are not implemented by this change.
