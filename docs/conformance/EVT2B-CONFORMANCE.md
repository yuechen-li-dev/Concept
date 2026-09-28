# EVT2b MIR to LIR core lowering

`language/evt2/valid/core.concept` covers `Add`, `Max`, `Sum4`, `CheckedIndex`, `StoreIndex`, `Choose`, and `Early`. `concept lir language/evt2/valid/core.concept` parses and validates the source, builds MIR, qualifies SemanticFacts, validates the existing General Planner output, lowers the MIR-owned semantic bodies, verifies LIR, and prints it.

`TestEVT2CoreLIRPlannerAndDeterminism` inspects core structure and compares complete LIR text across 100 runs. `Sum4` has an explicit `<` range exit, checked accumulation, index guard, address, load, and return; `CheckedIndex` retains a per-access guard with its Planner decision ID. `TestEVT2UnsupportedFeatureFailsExplicitly` checks that a float function fails with `EVT2_UNSUPPORTED_TYPE`. This is a structural lowering milestone; it does not claim native execution or a differential C/LIR runtime oracle.
