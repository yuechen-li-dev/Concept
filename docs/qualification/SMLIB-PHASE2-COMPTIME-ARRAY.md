# SMLib Phase 2: runtime materialization of comptime arrays

The NURBS port exposed a C11 type-collection defect. A runtime index into a
module `comptime int<array>[25]` table passed semantics but emitted a compound
literal whose array typedef was absent. An imported five-element integer table
and three-element binary64 table reproduce the same problem independently.

`evt1CollectStorageTypes` now visits names in concrete runtime bodies and adds
the storage types of referenced comptime values through its existing recursive
type collector. This includes closed template and automata bodies. There is no
new constant evaluator, grammar, allocation, array law, or runtime facility.
The checked-output lane regenerated the affected comptime-table artifacts;
runtime C bodies are unchanged, and headers now include required storage types.

Validation on Windows AMD64:

- New imported semantic-artifact positive: Normal and Verify, strict C11,
  native integer and binary64 table reads.
- Negative: constant out-of-range table index still rejects with CV4233.
- Focused tests, checked-output regeneration and clean rerun passed.
- `go test -count=1 ./...`: cmd/concept 26.446s, internal/concept 333.771s,
  profile/vulkan 0.457s; remaining packages have no tests.
- `go vet ./...` passed.
- Frozen Zig paths were unchanged; root and legacy Zig suites were skipped.

The consumer keeps separate fresh baseline-requalification evidence under
SMlib_Concept/phase2/evidence; the frozen Phase 1/1a evidence is not overwritten.
