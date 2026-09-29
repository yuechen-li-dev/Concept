# R8e3 conformance

- `concept format <file>` writes a `.concept` or `.concept_test` in place.
- `concept format <project-directory>` formats root-owned Concept source and
  `manifest.concept`, skipping nested projects.
- `concept format <target> --check` reports paths that would change and exits
  nonzero when any do. It writes nothing.
- Syntax errors, invalid configuration, token changes, and invalid output
  prevent a write.
- The source tape retains token spans, byte offsets, and exact inter-token
  trivia. Line and block comments remain in order with unchanged text.
- The formatter retains token order and identifier, string, number, unit, and
  attribute spellings. It does not perform a semantic rename.
- Focused tests run 100 idempotence passes on comment-rich source and current
  unit, tensor, concept, machine, `.concept_test`, and manifest specimens.
- The valid EVT1 corpus round-trip test checks syntax and idempotence across
  441 source files.
- A focused semantic test compares MIR after removing refreshed source spans
  and checks generated C identity in Normal and Verify for a span-free program. R8c scientific-unit
  and R8d interpretation specimens retain semantic MIR; an Assert.Concept
  fixture retains proof truth.

Raw MIR identity is intentionally limited by diagnostic source coordinates.
Checked arithmetic generated C similarly embeds current source positions.
`concept lint --fix` and a safe bound-reference naming rewrite are deferred.
Lint naming suggestions normalize acronym words deterministically; no source
identifier is changed by either lint or format.
