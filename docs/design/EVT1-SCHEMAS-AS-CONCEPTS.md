# Schemas as concepts

The current codec contract is an ordinary Concept proposition:

```concept
concept OctagonCodec<T>
{
    requires Result<T, OctagonError> ReadOctagon(ref OctagonReader reader);
    requires Result<void, OctagonError> WriteOctagon(ref const T value, ref OctagonWriter writer);
}
```

This contract uses normal concept proof tooling. It does not define a second
schema language. Derivation is explicit and checked as ordinary generated
functions. The R7h1 Standard implementation proves this pattern for integer
records; it is not a general proof of serializability for arbitrary `T`.
