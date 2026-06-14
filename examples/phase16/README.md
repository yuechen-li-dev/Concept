# Phase 16 multi-module examples

Phase 16 v0 treats modules as compilation-unit boundaries supplied by the
harness or driver. Imports make modules visible for qualified access only; they
do not map filesystem paths, paste declarations into local scope, or create
packages.

The `.concept` files in this directory are note-style examples with virtual file
sections. They document the same harness-supplied source-set model used by the
`.conception` fixtures; they are not filesystem import-resolution inputs.

## Examples

- `cross-module-add.concept` demonstrates `Main` importing `Math` and calling
  `Math.Add(...)` through the full multi-source model.
- `same-names-qualified.concept` demonstrates modules with the same ordinary
  item names selected by explicit module qualification.
- `imported-types.concept` demonstrates imported struct and enum type references
  such as `Geometry.Point` in type positions.
- `imported-c-abi.concept` demonstrates an imported `extern "C"` declaration
  called by module-qualified name.
- `module-boundary-notes.concept` summarizes the v0 boundary rules and shows
  that imports are not textual inclusion or package lookup.

## Supported v0 behavior

- multi-source fixtures lower through MIR/backend/run for the supported subset;
- ordinary functions and types from all modules are emitted into one generated C
  unit;
- qualified cross-module calls run;
- qualified imported struct/enum type references are resolved;
- imported repr(C) metadata is available to C ABI validation;
- backend C names are hardened for same ordinary item names in different
  modules;
- `export "C"` symbols remain exact/unmangled;
- C ABI symbol uniqueness remains a whole-compilation-unit semantic rule.

## Non-goals

Phase 16 examples intentionally do not imply support for aliases, wildcard
imports, re-exports, unqualified imported names, public/private visibility,
packages, filesystem lookup, linker driving, separate object files, incremental
compilation, module spanning multiple files, multiple modules per file, or
cross-package dependency resolution.
