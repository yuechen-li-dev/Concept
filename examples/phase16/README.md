# Phase 16 multi-module examples

Phase 16 v0 treats modules as compilation-unit boundaries supplied by the harness or driver. Imports make modules visible for qualified access; they do not map filesystem paths, paste declarations into local scope, or create packages.

The `.concept` files in this directory are note-style examples with virtual file sections. They are documentation examples, not filesystem import-resolution inputs.

Implemented in P16-M8:

- multi-source fixtures lower through MIR/backend/run for the v0 subset;
- ordinary functions and types from all modules are emitted into one generated C unit;
- qualified cross-module calls run;
- backend C names are hardened for same ordinary item names in different modules;
- C ABI symbol uniqueness remains a semantic whole-compilation-unit rule.
