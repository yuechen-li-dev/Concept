# EVT1 initialized typed-storage ownership direction

Status: R6l meaningful progression; one general generic-carrier blocker pinned

R6k stopped on four allocator-independent boundaries. R6l resolves three
directly and replaces the unsafe fourth shape with an owner-relative borrow:

| R6k boundary | R6l result |
|---|---|
| open-template `bind<T>` | resolved, including trusted region input |
| `Destroy(owner.storage)` | resolved by field-sensitive object state |
| `move owner.storage` | resolved as authority transfer without object relocation |
| returned live `ref T` | resolved through `Value(owner.storage)` and owner-bounded provenance; self-referential returned aggregates remain invalid |

The implementation extends the existing place-state machinery. Each binding
may carry a bounded map from field path to storage-object state. Geometry and
provenance remain in `SemanticValueFacts`; object state remains a distinct
operational channel. Scope cloning and joins preserve both. No allocator or
owner spelling is recognized.

`Initialize` changes Uninitialized to Initialized. `Destroy` requires
Initialized, runs ordinary `T` Drop, changes the state back to Uninitialized,
and invalidates borrows derived from that object lifetime. Reinitialization is
therefore a new object lifetime over unchanged raw storage. A user-defined
generic Drop is instantiated for matching concrete generic aggregates and is
the explicit destruction authority.

Generated C continues to represent `Storage<T>` as the existing typed address.
Owner and storage moves copy only that descriptor. There is no `memcpy` of the
live object, hidden flag, heap, registry, RTTI, or proof payload.

## Remaining blocker

The required `Result<Owner<T>, E>` proof currently fails with `CV4106`:
`expected Owner<int> but got Owner<T>`. The failure is pinned in
`r6l_initialized_storage_test.go`. Fixing it requires the general generic type
substitution/canonicalization pipeline to retain nested applications through
failure carriers and imported generic-owner bodies. The imported case is pinned
separately as `CV4148: unknown concept parameter T`. Patching Result, module
payloads, or an owner name would
violate the milestone doctrine, so allocator work remains deferred.

Deferred with it are Option/wrapper carrier parity, cross-module factory state
summaries, callable/async/machine owner persistence, and the requested product
specimens. General `T*`, MMIO, shared ownership, variadics, GC, allocators, and
native LIR remain outside R6l.
