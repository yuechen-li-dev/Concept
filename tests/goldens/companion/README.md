# Native C++ counter companion

The `native/` C++ counter remains native. `manifest.concept` builds its
ordinary C bridge, probes the two-field `CounterStats` layout, and binds
`concept/Counter.concept`. `tests/counter.concept_test` observes behavior and
the declared foreign creation contract in Verify. The first attempted
Concept syntax is preserved in `first-draft.concept.txt`.

Run from the repository root:

```text
go run ./cmd/concept plan tests/goldens/companion
go run ./cmd/concept check tests/goldens/companion
go run ./cmd/concept build tests/goldens/companion
go run ./cmd/concept test tests/goldens/companion --verify
```

The benefit without migration is a typed, measured ABI snapshot and explicit
semantic claims over the existing implementation. Familiar: C ABI bridge.
New: manifest, schema concept, and `DeclaredForeign` provenance. Advanced:
Verify observer and artifact identity.
