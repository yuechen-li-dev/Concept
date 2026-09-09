# EVT1 R4k conformance

Status: complete bounded class/interface/non-owning-dyn slice

## Classification

The canonical corpus contains 30 required human-readable sources under
`language/evt1/interface`: 16 accepted and 14 statically rejected. Every case is
classified `PASS`; there are no `EXPECTED-DIVERGENCE`, `EVT1-GAP`,
`POC3-LEGACY`, or `SPEC-AMBIGUITY` results in the selected slice. The additional
`class_interface_dyn.concept` specimen combines the runtime acceptance paths.

Class semantics are EVT1-new. Interface and dyn are redesigns informed by PoC3
Phase 14, not source or implementation parity. PoC3 supplies declaration,
signature, diagnostic, erased-storage, and dynamic-call pressure. EVT1 instead
uses structural concept satisfaction, borrowed provenance, static witness
tables, and ordinary value storage.

## Executable evidence

`internal/concept/interface_dyn_test.go` checks all 30 required sources and
their semantic diagnostic families. It inspects deterministic witness MIR,
class/method and dyn operations, two-pointer generated types, static const
witness tables, field accessors, and the absence of allocation/RTTI/object
runtime mechanisms. It separately corrupts witness MIR to prove invariant
validation.

Strict-C11 native execution covers class mutation, private internal access,
static struct/class interface satisfaction, struct/class dyn method dispatch,
mutable and readonly interface fields, interface composition, immovable
backing values, and scoped provenance-safe use. Rejection evidence covers
private external access, missing/wrong/private-only interface members,
non-interface or unsatisfied dyn construction, const weakening, local escape,
unknown methods, and readonly field mutation.

## Boundary

R4k does not include owning dyn, erased inline/heap storage, inheritance,
protected/friend access, abstract classes, virtual declarations, overload
expansion, constructors/destructors, RTTI, downcast, object headers, GC, or an
allocator framework. A recommended next milestone is a bounded optimizer/LIR
consumer of the R4j semantic facts; owning erasure should remain separate until
an explicit storage policy is designed.
