# R7g conformance status

**Result: honest stop for full R7g.** The structural reflection boundary works on the real parser, semantic analysis, artifact import, and CLI. The generating flagship and test-only runtime reflection remain open. The new declaration-result and provenance path would be a broader compiler architecture change; adding a source substitution shortcut here would not meet the milestone's normal-checking and inspectability rules.

| Requirement | Status |
| --- | --- |
| Explicit typed source query | `reflect<T>;` implemented; result is compiler-only Go `TypeInfo`, not yet a first-class Concept comptime value. |
| Records, fields, payload enums, tables, closed generics, attributes | Implemented and covered by directed tests. |
| Cross-module permission and artifact-only query | Implemented with defining type `[[reflect]]`; no source reparse. |
| Reflection erases before runtime | Generated C/header/MIR byte equality test passes. |
| Determinism | 100-run byte equality for module artifact and structural JSON passes. |
| Generated ordinary declarations, provenance, inspection, `concept explain` | Not implemented. |
| Derived Standard.Collection `Trace<T>` and parity | Not implemented; handwritten collector tests still pass. |
| Test-only runtime reflection gate and bounded metadata | Not implemented. |
| Strict C11 / Standard / DragonGod / BurnIn | Existing paths pass; no generated Trace C exists to qualify. |

The next blocker is a typed declaration-result operation in bounded comptime, with stable provenance and insertion before normal semantic checking. Until that exists, the query is useful for structural inspection and artifact boundary validation but cannot satisfy the R7g generation doctrine.
