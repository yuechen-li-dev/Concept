# EVT1 semantic layout and stream direction

Status: R4g spans consume R4f fixed semantic layout and zero-storage stream facts

## Purpose and lineage

R4f adds a small Core-language surface for describing fixed regions of one
existing contiguous storage object and named channels over those regions. It
carries forward
the durable SDSL-V distinction between semantic layout/stream declarations and
backend realization, but it is implemented in the active Go compiler and uses
ordinary EVT1 values, references, `bind`, const rules, and provenance. No Oct
syntax, compiler, runtime, artifact, or repository becomes an implementation
dependency.

A layout is compile-time semantic geometry. A stream is a zero-storage channel
map over one layout. Neither declaration is storage, a value allocator, a
runtime scheduler, or a second type system.

```text
Facts known ahead of time should be declared ahead of time.

Optimization should consume semantic facts,
not spend its budget rediscovering programmer intent.
```

## Canonical surface

```concept
layout PacketLayout
{
    uint header;
    align(16) byte<array>[256] payload;
    at(272) uint checksum;
}

stream PacketStream over PacketLayout
{
    Header = header;
    Payload = payload;
}
```

Regions use ordinary fixed-size Core types, including fixed `array` and
`ndarray` storage. Declaration order determines default placement. Each region
begins at the next offset satisfying its natural alignment or a stronger
`align(N)` request. `at(N)` selects an explicit byte offset, which must satisfy
the effective alignment and must not overlap another region. Alignment is a
power of two from 1 through 4096. Final layout size includes tail padding to the
maximum region alignment.

The compiler computes and retains for every layout: stable layout and region
identities, byte size, required alignment, region offset, byte extent, region
type, and pairwise disjointness. `LayoutSize<T>()`, `LayoutAlign<T>()`, and
`LayoutOffset<T>("region")` expose only those bounded compile-time facts.
`LayoutOffset` deliberately takes a region-name string; this is a closed query,
not general reflection or a stringly runtime lookup.

R4f requires all region geometry to be compile-time fixed. Runtime-parameterized
layouts are explicitly rejected. General layout parameters, strides, unions,
bitfields, endianness policy, runtime metadata, and host ABI inference remain
deferred.

## Binding and projections

```concept
byte<array>[280] backing;
ref PacketLayout packet = bind backing;
ref byte<array>[256] payload = packet.payload;
```

Layout binding reuses unary destination-typed `bind`. The source must be a whole
contiguous fixed array or ndarray place whose byte extent equals the semantic
layout size. Mutable targets require mutable backing; `ref const` layout targets
provide read-only region projections. R4f's executable inline-storage lowering
guarantees 64-byte backing alignment in layout-bearing modules; layouts that
require stronger alignment reject until an explicit aligned-storage mechanism
exists.

The bound layout and every projected region retain the same backing identity,
const state, lexical/call-result provenance, and scoped restriction. Projection
is typed pointer-plus-offset lowering over the existing storage. It does not
allocate, copy, move, resize, transfer ownership, or create independent region
storage. The semantic graph proves non-overlap at compile time; the C bootstrap
descriptor is only a realization of that graph.

## Streams

A stream names channels over exactly one declared layout. Every channel aliases
one existing region and inherits that region's exact type and stable identity.
Unknown regions and duplicate channels reject. Channel order is declaration
order and has no storage-layout effect.

```concept
ref PacketStream stream = bind packet;
ref byte<array>[256] payload = stream.Payload;
```

Stream binding accepts only a bound reference to the stream's declared layout.
It preserves constness and provenance and creates no storage or runtime object.
Channel projection is exactly the corresponding layout-region projection:
mutations through a mutable channel are visible through the layout and original
backing, while a channel reached through `ref const` is not assignable. MIR marks
streams as `zero_storage` and validates the channel-to-region mapping.

Streams do not add iteration, push/pull execution, queues, synchronization,
ownership, transport, effects, or scheduling. They are semantic names for
already-proven regions.

The term is intentionally SDSL-derived. A GPU/SDSL stream describes semantic
channels flowing through shader stages; a Core CPU stream describes semantic
channels projected over declared storage. The common meaning is a named
semantic memory/dataflow mapping. It is not a file or socket stream, a lazy
iterator, or an asynchronous byte stream.

## Ownership, future consumers, and optimization

One backing object retains the existing ownership and Drop responsibility.
Layouts, regions, streams, and channels are non-owning projections and never
drop independently. This preserves the law: one backing owner, many semantic
projections, one destruction responsibility.

R4g Span/ReadOnlySpan constructed from an array, ndarray linear region, layout
region, or stream channel receives backing provenance, region identity, offset,
extent, alignment, and mutability directly from this graph.
Future allocator-backed storage may provide a backing object without changing
layout ownership. The R4h tensor interpretation consumes the same region shape
without becoming storage; future vector/matrix aliases may do likewise. Future noalias optimization may
consume stable identity, common parent, and declared disjointness without R4f
claiming a generalized alias solver.

CPU and GPU backends may lower the same semantic graph differently. A fixed C
backend can erase offsets into address arithmetic; a later GPU path can use
channel meanings during shader/interface lowering. Neither changes the Core
declaration's meaning.

Explicit offsets are useful pressure for packet and firmware register maps.
Future MMIO use also requires a separately specified `volatile`/raw-address
acquisition law; R4f does not infer those semantics from `at(N)` and does not
bind device addresses.

The active semantic projection chain is:

```text
layout region
    down
stream channel
    down
Span / ReadOnlySpan
```

The stream name remains a zero-storage alias. Span construction consumes the
mapped layout region's exact stable identity, backing provenance, byte offset,
extent, alignment, shape-derived scalar count, constness, and contiguity. It
does not rediscover geometry or introduce a stream object translation layer.
Subspan retains that parent identity with a relative half-open interval, so
declared disjoint parent regions remain distinguishable for later optimization.

## Deliberate boundary

R4g does not implement Slice, FixedBuffer, `stackalloc`, allocator-backed
storage, raw pointers, implicit allocation,
runtime-sized layout declarations, vector/matrix/tensor mathematics, TensorIR,
general stream concepts, stream composition, a stream runtime, GPU lowering, or
a stable C ABI/layout contract. Generated strict C11 is executable bootstrap
evidence, not a public ABI promise.

## R4j fact contribution

Layout contributes stable region identity, byte interval, effective alignment,
fixed type/shape, and sibling disjointness. A stream channel preserves those
facts under its semantic channel name; it creates no storage. Span construction
preserves the mapped identity and interval evidence, and Subspan narrows the
interval and degrades alignment conservatively. Tensor then preserves the same
backing facts while adding rank and mathematical shape. The MIR fact registry
is the future optimizer authority; no pointer-intent reconstruction or runtime
stream/fact object is introduced.
