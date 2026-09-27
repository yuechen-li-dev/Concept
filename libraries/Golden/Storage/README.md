# Fixed page cache

`PageCache.concept` stores eight 256-byte pages under generational IDs, rejects
stale handles, tracks dirty state, and prevents eviction of dirty or pinned
pages. `page_cache.concept_test` covers write/read/evict/reuse. The C++-style
first draft is preserved in `first-draft.concept.txt`.

Friction: raw pointers and `nullptr` became typed Result-returning lookups.
A live borrowed page initially prevented removal; an inspection scope now
ends before eviction. Familiar: page headers, dirty state, cache capacity.
New: generational handles and explicit ownership. Advanced: invalidation
proof and NoAllocation.
