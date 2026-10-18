# Cathedral AMD64 backend

Import `Standard.Backend.AMD64` to use the backend. The public entry module and
its existing declarations remain available through that import.

| Source | Responsibility |
| --- | --- |
| [AMD64.Model.concept](AMD64.Model.concept) | Backend records, enums, empty values and operand constructors |
| [AMD64.Analysis.concept](AMD64.Analysis.concept) | MachineIR projection and validation, liveness, register allocation and call planning |
| [AMD64.Frame.concept](AMD64.Frame.concept) | Unified stack geometry, frame validation, spill storage and call-action realization |
| [AMD64.Encoding.concept](AMD64.Encoding.concept) | Integer and scalar SSE2 instruction selection, operand encoding and byte writers |
| [AMD64.concept](AMD64.concept) | Public entry point, function/module emission, fixups and final image validation |

Dependencies follow one direction:

```text
AMD64 -> AMD64.Encoding -> AMD64.Frame -> AMD64.Analysis -> AMD64.Model
```

These are ordinary Concept modules with distinct declared identities. Source
lookup accepts trailing namespace components in a dotted filename, so
`Standard.Backend.AMD64.Encoding` resolves to `AMD64.Encoding.concept` beside the
entry module. Directory-based source paths remain supported. Supplying both
layouts for one module is an identity error; neither silently takes precedence.
Compiled module artifacts retain their canonical directory-based paths and
source-digest validation.

The split changes organization, not backend authority, MachineIR contracts or
native behavior. Full floating call emission remains the EVT2f5 boundary
`AMD64_UNSUPPORTED_FLOAT_CALL_EMISSION`.
