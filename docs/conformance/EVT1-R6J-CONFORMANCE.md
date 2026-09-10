# EVT1 R6j conformance

Status: implemented

R6j separates native ABI declarations from explicit foreign semantic authority. The compiler transports `DeclaredForeign` allocation and storage facts through ordinary wrappers, function summaries, semantic modules, MIR, and proof graphs. `Platform.Host.Memory` demonstrates a fallible owned host allocation whose region can be bound and initialized only during the owner's lifetime; strict C11 validation observes one allocation and one release.

The fixture corpus contains ten valid sources, seven rejected/Unknown sources, and three `.concept_test` sources under `language/evt1/tooling/interop`. It covers ABI-only Unknown behavior, allocation effects, owner lifetime, typed binding, alignment, imported facts, firmware-style non-host storage, reconstructed address rejection, strengthening rejection, escape, missing authority, ABI mismatch, and missing host accessibility.

The parser provisionally accepts `foreign concept ... on ...` with `requires`
clauses. This is the selected R6j spelling but not yet a permanently stabilized
language commitment. The implemented semantic model is independent of retaining
that source form later.

Deferred: full C++ ABI, header importer, contract generation, VerifiedForeign, callbacks, shared foreign ownership, freshness/disjointness, MMIO/volatile, and the Standard.Memory allocator framework (R6k).
