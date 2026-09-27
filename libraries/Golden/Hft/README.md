# Bounded market path

`Market.concept` stamps events, publishes to a fixed 64-slot queue using
Release, consumes using Acquire, and drains under an explicit budget.
`market.concept_test` checks order and inline storage. The first draft is
preserved in `first-draft.concept.txt`.

Friction: guessed `AtomicLoad`/`AtomicStore` names were wrong; Standard uses
`LoadAtomic`/`StoreAtomic`. Option is matched rather than compared to null.
Familiar: ring queue and memory orders. New: explicit Concept match and
bounded range loop. Advanced: feed and signal contexts prove
`sync.SingleProducer` and `sync.SingleConsumer`. The structural
`sync.PublishedBefore(MarketQueue, MarketQueue)` claim remains Unknown, so
the C11 atomic operations remain in Normal and Verify.
