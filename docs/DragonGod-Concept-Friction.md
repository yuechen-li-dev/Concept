# DragonGod Concept Friction Log

DragonGod.Kernel is intentionally being implemented before self-hosting to stress-test Concept as a real systems programming language.

Each DragonGod phase should record friction discovered while writing real Concept code.

## Entry format

- Phase:
- Area:
- Pain observed:
- Workaround used:
- Classification:
  - language design issue
  - compiler implementation gap
  - standard library/container gap
  - documentation gap
  - expected limitation
- Severity:
  - low
  - medium
  - high
  - blocking
- Suggested future fix:

## DG1 entries

- Phase: 20 / DG1
- Area: Memory key generics
- Pain observed: A kernel-facing `MemoryKey<T>` shape is desirable, but DG1 avoids betting the DragonGod surface on generic storage APIs before the memory model exists.
- Workaround used: Use `MemoryKeyInt` as a temporary integer-backed key shell.
- Classification: expected limitation
- Severity: low
- Suggested future fix: Revisit typed keys when Memory storage and generic API ergonomics are implemented.

- Phase: 20 / DG1
- Area: Reason text
- Pain observed: DragonGod reasons eventually want stable string/StringView semantics, but DG1 only needs deterministic reason identity.
- Workaround used: Use integer-backed `Reason { int code; }` and document it as a placeholder.
- Classification: expected limitation
- Severity: low
- Suggested future fix: Replace or extend `Reason` with a string/StringView-backed surface once Concept string support is kernel-ready.

- Phase: 20 / DG1
- Area: Module source organization
- Pain observed: Multi-source fixtures model one source per module name; the DragonGod tree uses multiple files with the same module name for human organization.
- Workaround used: Keep source-tree skeleton files split by concern, while fixtures aggregate each tested module into a single virtual source.
- Classification: compiler implementation gap
- Severity: medium
- Suggested future fix: Add package/file aggregation semantics for multiple source files contributing to one module.

- Phase: 20 / DG1
- Area: Payload enum construction
- Pain observed: Payload enum variants are usable for DG1, but cross-module examples require verbose fully qualified names.
- Workaround used: Use fully qualified `DragonGod.Kernel.Automata.AutomataSignal::...` references in examples and fixtures.
- Classification: documentation gap
- Severity: low
- Suggested future fix: Document idiomatic import and qualified-name conventions for multi-module Concept code.
