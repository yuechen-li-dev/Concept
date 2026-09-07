# Concept PoC3 / Zig

Status: retired reference implementation.

This compiler is no longer the active Stage 0 implementation of Concept.

It remains preserved for:

- semantic archaeology;
- differential testing;
- fixture migration;
- regression comparison;
- implementation reference.

New Concept language development targets EVT1 / Go.

The archived source is the compiler from Concept commit
`e60f9012d001474692112873cfd6e0f09671bd38`. Its implementation is under
`src/compiler`, and its original build definition is `build.zig` in this
directory.

Run the preserved compiler tests from this directory with:

```console
zig build test
```

The repository-root `zig build test` command is retained as a compatibility
path to the same reference suite. The top-level `language/`, `tests/`,
`examples/`, and PoC3 design documents remain the legacy semantic/reference
corpus shared with the EVT1 migration; they are intentionally not duplicated
or moved in R0.
