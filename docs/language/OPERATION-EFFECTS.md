# Operation allocation effects

R7f2 adds `compiler.InvalidatesBorrows(Operation, resourceParameter)` to the existing operation-effect declarations. The parameter must be a reference; the fact travels in module artifacts and excludes conflicting lexical borrows at call sites. See [destructive resource effects](../design/EVT1-DESTRUCTIVE-RESOURCE-EFFECTS.md).

Status: EVT1 R6d local authority implemented

The initial effect vocabulary contains one conservative semantic fact:

```concept
extern "C"
byte* ConceptHostAllocate(usize size);

requires compiler.Allocates(ConceptHostAllocate);
```

`Allocates` means "may allocate". It is an authoritative safe
over-approximation, so an implementation may allocate on only some paths or no
path in a particular build. It is not a tooling attribute and has no runtime
representation.

`NoAllocation` remains derived. Source cannot declare it: a negative
declaration is rejected as `OPERATION_EFFECT_NEGATIVE_LIE`. The R6b proof
projection follows uniquely resolved local calls:

```text
known allocation effect -> Disproven
closed local allocation-free graph -> Proven
opaque operation without a summary -> Unknown
```

Proof nodes identify `DeclaredEffect` and `ExternalContractEffect`; call nodes
preserve the operation chain. Cross-module summaries and interface-operation
effect compatibility remain part of the reusable-module blocker.
