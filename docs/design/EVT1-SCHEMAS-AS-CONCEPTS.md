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
codecs, generated payload enums, and nominal refined values use this same
proposition. A missing element codec or admission operation fails ordinary
concept satisfaction; derivation is not a general proof of serializability
for arbitrary `T`.

A refinement policy can be another ordinary concept, such as a required
`IsAdmitted(ref const T)` operation. The codec reader must still use the
authoritative `AdmitOctagon` constructor; a schema proof alone cannot stuff a
raw representation into the refined value. Current refined derivation expects
a single-field nominal wrapper with an author-defined admission operation;
`OnlyField<T>()` rejects any other field count during derivation.
