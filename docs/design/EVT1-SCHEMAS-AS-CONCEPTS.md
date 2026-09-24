# Schemas as concepts

The current codec contract is an ordinary Concept proposition:

```concept
concept OctagonCodec<T>
{
    requires Result<T, OctagonError> ReadOctagon(ref OctagonReader reader, OctagonType<T> type);
    requires Result<void, OctagonError> WriteOctagon(ref const T value, ref OctagonWriter writer);
}
```

This contract uses normal concept proof tooling. It does not define a second
schema language. Derivation is explicit and checked as ordinary generated
functions. Closed generic array witnesses and derived record and columnar-table
codecs use this same proposition. A missing element codec fails ordinary
concept satisfaction; derivation is not a general proof of serializability
for arbitrary `T`.
