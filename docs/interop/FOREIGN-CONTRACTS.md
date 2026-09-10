# Foreign semantic declarations

Status: EVT1 R6j semantics implemented; exact source syntax is provisional and must be reviewed before language stabilization

`extern "C"` declares only representation and linkage. It never makes a pointer trusted storage and never invents an effect. A separate foreign semantic declaration may bind one named extern operation to the bounded facts `Allocates`, `ExternalStorage<Space>`, and `HostAccessible`.

The current parser prototype writes this as:

```concept
foreign concept HostAllocationContract on ConceptHostAllocate
{
    requires compiler.Allocates(ConceptHostAllocate);
    requires compiler.ExternalStorage<SystemMemory>(result, size, alignment);
    requires compiler.HostAccessible(result);
}
```

R6j provisionally chooses this `foreign concept ... on ...` and `requires`
spelling. It is the accepted implementation and fixture syntax, but is not yet a
permanently stabilized language commitment. The compiler representation, module
summaries, trust origin, and storage rules do not depend on later retaining it.

## Trust and storage authority

Accepted declarations have proof origin `DeclaredForeign`. They participate in the ordinary Proven/Disproven/Unknown truth model, but the origin remains visible in MIR, proof graphs, and `concept-module.v1`; it is not `CompilerProven` or `VerifiedForeign`.

`EstablishExternalRegion<Space>` is a guarded compiler operation available only while validating a function from the declaring semantic module. It requires a live reference to an owner/lease wrapper, the wrapper's address, extent, and alignment fields, and the exact declaration identity. The result is an ordinary `MemoryRegion<Space>`. Metadata erases before C lowering; there is no runtime contract registry or provenance table.

For `SystemMemory`, the declaration must also state `HostAccessible`. The resulting address can reach `bind<T>` only while its wrapper remains live. Returning the region beyond a local owner, assigning it to longer-lived storage, using it after owner move/drop, or reconstructing it with `AddressFromBits` rejects. Distinct wrappers do not imply freshness or disjointness.

The platform wrapper must map null/failure to `Result::Error` before exposing a region. The contract author is responsible for the truth of declared native claims and for runtime validation that cannot be established from the ABI. Tests may observe alignment, extent, and release counts, but do not upgrade the origin to `VerifiedForeign`.

Foreign ABI declarations should remain representation-level. A wrapper may add units or safely narrow guarantees; it may not strengthen a declared alignment or extent. The current bounded establishment form conservatively requires geometry to come from the authority wrapper.

Deferred: full C++ ABI, header parsing, generated contracts, VerifiedForeign IR analysis, callbacks, shared ownership, fresh-region guarantees, MMIO/volatile, and allocator policy.
