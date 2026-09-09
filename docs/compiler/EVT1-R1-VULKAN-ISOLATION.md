# EVT1 R1 Vulkan profile isolation

Status: R1 audited and executable

## Architectural result

Vulkan builtin types, C spellings, required headers, synthetic fields, the
ActuationOutcome enum, and the Prometheus Vulkan import marker are registered
by `profile_vulkan_definition.go`. The shared parser and semantic environment
ask the selected `ProfileDefinition` what is admitted. The C lowerer asks the
same registration for type spelling, declarations, and headers.

Core's registry contains only `int`, `void`, `bool`, `string`, `uint64`, and
the provisional general automata outcome. It contains no Vulkan or Prometheus
type. `profile Core;` therefore cannot acquire domain types merely because the
compiler binary also supports Vulkan.

## Source inventory

| Location | Classification | R1 disposition |
|---|---|---|
| `profile_definition.go` | core language infrastructure | Small data contract and Core definition. |
| `profile_vulkan_definition.go` | profile-owned registration | Owns Vulkan type names, C mappings, header needs, synthetic error field, ActuationOutcome, feature admissions, and `Prometheus.Vulkan`. |
| `parse.go` profile selection | core language infrastructure | Selects a registered profile; type parsing consults that profile. Syntax nodes for effect/actuator remain generic AST forms. |
| `validate.go` admission prelude | profile validation hook | Uses profile booleans and registered types; no Vulkan builtin name test remains. |
| `profile_vulkan.go` | profile validation hook | Actuator validation stays in-package because it shares private semantic invariants. |
| `automata.go` effect paths | profile validation hook | Shared provisional automata owns effect emission mechanics after profile admission. |
| `generate.go` builtin type/header paths | profile lowering hook | Uses profile data rather than Vulkan name switches. |
| `generate.go` effect/actuator runtime support | profile lowering hook | Remaining dense coupling; bounded and documented, not split in R1. |
| `profile/vulkan/profile.go` | profile-owned consumer | Vulkan-only driver over shared parsing/generation. |
| `support.go` | provenance comment | Mentions extraction history only; no semantic dependency. |

Generated golden files and tests contain Vulkan names as evidence and are not
compiler ownership leaks.

## Leakage audit

The pre-R1 accidental leakage was:

- a shared builtin switch naming `PipelineLayout`, `Pipeline`, `VulkanError`,
  `VkBuffer`, and `VkCommandPool`;
- a Core-denial predicate that duplicated those names and the `Vk` prefix;
- a C-type switch, Vulkan-header predicate, and VulkanError declaration
  predicate in the shared lowerer;
- unconditional insertion of VulkanError fields and ActuationOutcome into
  every semantic environment.

Those items are removed. Search results that remain in shared files are one of
generic effect/actuator AST and MIR structure, generic admitted-feature
validation, the explicitly named profile diagnostic, or the acknowledged
profile validation/lowering hook. No Prometheus repository package is imported
by the compiler.

## Remaining coupling

Effect batching and actuator runtime C emission still live in the cohesive
`concept` package. Moving them would export a large private semantic surface,
so R1 leaves them behind the selected profile's admission gate and records the
boundary. This is not accidental Core admission: the Core denial corpus proves
the declarations are rejected before ordinary analysis.

The current Vulkan profile has no compiler-owned mechanism-call builtin.
Mechanism operations are explicit imported function declarations in Vulkan
source. Accordingly, Core treats an undeclared mechanism-shaped call as an
ordinary unknown function; R1 does not invent a builtin merely to test denial.

## Verification

- `TestProfileDefinitionsOwnBuiltinAdmissions` checks registry ownership and
  the absence of Prometheus application types.
- `TestVulkanProfileBuiltinLoweringUsesRegistration` checks the registered
  import/type path and generated Vulkan C spellings.
- `TestFoundationDifferentialConformance` includes five Core profile denial cases.
- Existing exact generated-output, deterministic double-generation, native
  C11, and Vulkan-profile tests remain the preservation suite.

The layering law is:

```text
Concept Core
    -> selected Concept Vulkan profile
        -> Prometheus bindings/integration
```

Concept EVT1 is not an extension of Vulkan. Vulkan is an extension of Concept
EVT1.
