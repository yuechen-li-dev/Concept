# Legacy

Retired implementations and their evidence, kept for semantic archaeology,
differential testing, and migration provenance. Nothing here is built by the
active Go compiler.

| Directory | What it is |
| --- | --- |
| `poc3-zig/` | The PoC3 Zig compiler at the R0 cutover commit, laid out as its original repository: `src/`, `build.zig`, `language/phase*` (1,296 fixtures), `tests/` (corpus and scaffolding), `examples/phase*`, and `docs/` (phase designs, the PoC3 journal, coverage). Run with `zig build test` here or at the repository root. |
| `dragon-god-poc/` | The earlier DragonGod proof of concept and its Phase 20 blueprint and friction log (`docs/`). The active kernel is `libraries/DragonGod`. |

Fixture text is never rewritten in place. Current counterparts are authored
separately under `language/evt1/` and `examples/tour/`.
