# EVT2c MachineIR conformance

Compiler: `concept-evt1-stage0-go`. Baseline LIR commit: `23a5c0f836aa5ed658601a1bac48b4a6923ddb9d`; implementation starts from merge commit `235e80e21c67a54a87034ad0ef79ee6037c9b083` on `origin/main`.

`language/evt2/valid/core.concept` is the real CLI and structural-test input. `TestEVT2cCoreStructure` checks Add (Win64 parameters, checked ADD, overflow trap, RAX return), Max (CMP/JCC without `SETCC`), Sum4 (indirect array parameter, loop backedge, retained guard, LEA/load/store, checked addition), CheckedIndex and StoreIndex (bounds failure and typed memory operations), and Early (checked subtraction and early return). `Choose` checks the ordinary bool parameter and `TEST` path. `TestEVT2cUnsignedAndUnsupported` pins unsigned branch and carry conditions and explicit float rejection. `TestEVT2cABIAndAliases` pins argument positions 1 through 5, integer returns, register aliases, and save sets. `TestEVT2cVerifierRejectsMalformed` mutates real MachineIR and expects rejection. `TestEVT2cDeterminism100` compares the complete machine printer across 100 independent lowerings.

The printer is an inspection format, not Concept syntax or final assembly. Example from Add:

```text
fn Add [Add|Add(int, int)] amd64-windows win64 -> i32
  arg 0 i32 ecx -> v0
  arg 1 i32 edx -> v1
b0 (lir b0):
  MOV v0:32 ecx
  MOV v1:32 edx
  MOV v2:32 v0:32
  ADD v2:32 v1:32 flags=f0
  JCC O f0 b1 b2
b1 (lir b0):
  TRAP [overflow]
  TRAP
b2 (lir b0):
  MOV eax v2:32
  RET
```

This milestone proves structural operational lowering and verifier behavior. It does not claim encoded instructions, physical allocation, native execution, or a C/MachineIR runtime differential.
