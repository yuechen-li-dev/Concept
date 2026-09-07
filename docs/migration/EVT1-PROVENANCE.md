# Concept EVT1 provenance

Cutover date: 2026-09-07

## Source records

| Line | Repository state | Source material |
|---|---|---|
| Concept PoC3 / Zig | `e60f9012d001474692112873cfd6e0f09671bd38` | `src/compiler/`, root `build.zig`, `language/`, `tests/corpus/`, `examples/`, `docs/Concept-PoC3.md`, and `docs/design/` |
| Concept/Vulkan EVT1 / Go | Oct `584bd176fd50664edadcb2bc3ae78431ac0f1e51` | `internal/conceptvulkan/evt1_*.go`, the EVT1 tests and required golden outputs, `examples/Concept-Vulkan/evt1_*.concept`, and `docs/Concept-Vulkan/CONCEPT_VULKAN_LANGUAGE_CONSTITUTION.md` |

The Oct compiler identities at extraction were `concept-vulkan-m1` for the
original narrow mechanism compiler and `concept-vulkan-evt1-m1b-d` for the
recursive EVT1 compiler line. R0 extracted the latter. The narrow M1 parser and
its handwritten kernel-54 generator were not copied into the general compiler.

The Concept/Vulkan constitution was copied without reinterpretation to
`docs/reference/CONCEPT_VULKAN_LANGUAGE_CONSTITUTION.md`. Its source status was
normative through EVT1 M1B-D, with experimental DragonGod typed-automata,
ordered-effect, and actuator extensions beyond that accepted base.

## Descent and authority

Concept EVT1 descends from two implementation lines:

1. Concept PoC3 / Zig: broad general-language semantic prototype and fixture
   corpus.
2. Concept/Vulkan EVT1 / Go: production-pressure compiler architecture grown
   from real Vulkan mechanisms.

EVT1 reconciles them deliberately.

The active Stage 0 implementation is now the Go compiler at
`internal/concept`, driven by `cmd/concept`. Its compiler identity is
`concept-evt1-stage0-go`. The package has no Oct dependency.

The Zig implementation is preserved at `legacy/poc3-zig` from the exact
pre-cutover Concept commit. It is a retired executable reference, not an active
development line. The top-level PoC3 fixtures and design documents remain a
shared legacy semantic/reference corpus so their history and paths do not
undergo needless R0 churn.

## Extraction and neutralization record

The following Oct implementation files supplied the seed:

- `evt1_types.go` -> `internal/concept/types.go`
- `evt1_parse.go` -> `internal/concept/parse.go`
- `evt1_validate.go` -> `internal/concept/validate.go`
- `evt1_comptime.go` -> `internal/concept/comptime.go`
- `evt1_automata.go` -> `internal/concept/automata.go`
- `evt1_actuator.go` -> `internal/concept/profile_vulkan.go`
- `evt1_generate.go` -> `internal/concept/generate.go`
- `evt1_test.go` -> `internal/concept/concept_test.go`

Public types such as `EVT1Type`, `EVT1Module`, `EVT1FunctionDecl`, and
`EVT1MIR` became `Type`, `Module`, `FunctionDecl`, and `MIR`. Public entry
points became `Parse`, `Generate`, and `MIRText`; parser, semantic environment,
and lowering implementation types were similarly neutralized. Artifact schemas
and generated symbols now identify general Concept rather than
Concept/Vulkan.

Lowercase `evt1*` helpers remain where a mechanical rename would add noise
without changing package or user-visible authority. They are implementation
details, not a second dialect API.

## Migration doctrine

- PoC3 Zig is retired, not discarded.
- Concept/Vulkan is extracted, not blindly renamed.
- EVT1 is a new canonical compiler line.
- The Go compiler architecture is the new Stage 0 implementation base.
- The Zig compiler and fixture corpus remain semantic reference material.
- Vulkan-specific semantics remain profile-specific unless explicitly admitted
  into core Concept.
- A difference is retained in the reconciliation matrix until an explicit
  language decision and test close it.

R0 does not assert feature parity. It establishes authority, provenance, a
working independent compiler seed, and durable evidence for subsequent ports.
