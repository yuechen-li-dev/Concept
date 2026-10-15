# SMLib Phase 2: span facts in closed generic parameters

A degree-specialized NURBS derivative kernel reduced to:

```concept
template <usize N>
double Head(Span<double> input) {
    Span<double> first = Subspan(input,0,1);
    return first[0] + (N as double);
}
```

Closing Head<2> rejected Subspan with CV4600, while direct indexing and an
ordinary function with the same span parameter worked. Closed template parameter
bindings initialized type, mutability and lifetime provenance but omitted the
span and semantic value facts already constructed for ordinary/open functions.

The existing parameter-fact builders now initialize both missing fields at that
same binding boundary. No span facts, proof outcomes, syntax, lifetime law or
new borrow analysis is invented. Imported positive tests execute mutable and
readonly subspans in Normal and Verify through strict C11. A generic local-span
escape still rejects with CV4521. Focused span, dependent-composition, Phase 2
and checked-output tests pass without regenerated artifacts.

Full validation: `go test -count=1 ./...` passed (cmd/concept 26.124s,
internal/concept 342.711s, profile/vulkan 0.418s); `go vet ./...` passed.
Root and legacy Zig suites were skipped because the frozen Zig paths were
unchanged. Fresh frozen-consumer rebuilds and proof evidence are kept in the
consumer's Phase 2 directory.

A distinct limitation remains: a span returned from a generic call lacks the
construction facts needed for a subsequent ReadOnlySpan conversion, although
ordinary call-result spans have a fact projection. This is not required by the
numerical kernel. The consumer retains a negative reproducer and does not claim
that this change resolves generic span-result composition.
