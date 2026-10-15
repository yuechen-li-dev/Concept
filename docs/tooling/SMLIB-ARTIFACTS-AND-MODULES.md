# Complete C11 emission and explicit module roots

The existing stdout command is unchanged:

```text
concept emit-c Kernel.concept > Kernel.c
```

To save the complete compiler output set, use:

```text
concept emit-c Kernel.concept --out-dir out
concept emit-c Kernel.concept --verify --out-dir out-verify
```

The directory receives the existing deterministic `<stem>.generated.c`,
`<stem>.generated.h`, `<stem>.mir.json`, `<stem>.manifest.json`, and
`<stem>.map.json` outputs. This saves the compiler's own artifacts without
reconstructing headers or patching C. Output naming remains unchanged; source,
compiler policy, and source-map provenance remain in the normal metadata.
`--out-dir` applies to emit-c only and must occur once.

## Exact import mapping

Module declarations are semantic identities, not source discovery requests:

- `Demo.Math` source maps to `<root>/Demo/Math.concept`.
- Its artifact maps to `<root>/Demo/Math.concept-module.json`.
- Import paths must declare the requested module identity.
- Two distinct matching roots are a conflict, never a priority choice.
- Aliases of the same physical root (relative/absolute, dot segments, symlinks,
  and Windows root casing) are deduplicated before lookup.
- Missing modules, wrong declarations, stale artifacts, and cycles are errors.
- Case-sensitive hosts require matching source path casing; Windows file lookup
  follows the filesystem's case behavior. No new cross-host casing guarantee is made.

A source file selected directly as a CLI root can have any filename; imported
modules must follow the exact mapping. A consumer normally resides outside the
dependency tree or receives that tree's root explicitly; selecting a nested
source does not implicitly make all its ancestors dependency roots.

```text
concept check app/Runner.concept --module-root artifacts
concept check app/Runner.concept --source-modules --module-root src
concept emit-c app/Runner.concept --source-modules --module-root src --out-dir out
concept test app/tests.concept_test --module-root src --verify
concept build-module src/Demo/Math.concept --source-modules --module-root src
```

`--module-root` is repeatable. Without `--source-modules`, ordinary checking,
build-module, and emission retain artifact-based imports. With that flag,
imports are built from the exact acyclic source graph into in-memory semantic
artifacts before the consumer is checked/emitted. No textual inclusion or
declaration scan is introduced. Tests already build source dependencies and now
accept the same explicit dependency-root option. They retain their existing
selection/source/project-anchored roots for compatibility. Existing configured
CONCEPT_MODULE_ROOTS values for compiler commands remain supported; explicit CLI
roots are preferred. No new ambient search location was added.

Packages retain manifest-owned package/dependency graphs and transport the same
semantic artifacts. Native companion projects retain manifest-owned roots and
ABI artifacts; generic emit-c consumes their checked source/artifacts through
its existing native-companion roots. These project workflows do not acquire an
ambient repository/module registry.

CLI regressions exercise nested import paths, artifact and source modes,
missing/wrong identities, physical-root aliases, conflicting roots, deterministic
complete output, strict C11, old stdout output, and Normal/Verify native tests.
Only Windows was executed for this hardening task.
