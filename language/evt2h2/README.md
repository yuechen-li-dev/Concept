# EVT2h2 specimens: checked frontend progression

Current status: **PARTIAL**. Payload enums reach verified LIR. Native MachineIR
production stops at `EVT2H2_TAGGED_TRANSPORT_REQUIRED`; there is no h2 native
artifact, image, ABI plan or native payload-access implementation yet.

The authoritative positive program is
[`Testing/TaggedValues.concept`](../../tests/runtime/Testing/TaggedValues.concept).
Its behavior assertions live in
[`tagged_values.concept_test`](../../tests/runtime/tagged_values.concept_test).
Both use the existing source language. Inspection uses existing checked match
patterns through the C bootstrap; no native match was added.

```powershell
./test.ps1
./test.ps1 -Lane Runtime -Filter Payload
./test.ps1 -Lane Runtime -Filter Payload -Verify
go run ./cmd/concept lir tests/runtime/Testing/TaggedValues.concept
go run ./cmd/concept machineir tests/runtime/Testing/TaggedValues.concept
```

The last command intentionally fails at the precise transport boundary.
`invalid/` pins existing source diagnostics. `unsupported/` separates valid
source/C programs from unsupported nested tagged LIR and native match.
Go tests own descriptor mutation, strict C11 layout/bit probes, imported
artifact identity and 100 fresh checked-LIR generations.

See the [design inventory](../../docs/design/EVT2H2-CHECKED-TAGGED-VALUES.md)
and [conformance record](../../docs/conformance/EVT2H-CONFORMANCE.md).
