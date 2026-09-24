# R7h Octagon and Concept schemas: architectural stop

## R7h1 continuation (2026-09-24)

The R7h stop above is historical. R7h1 began from clean Concept
`6ba377f0d01d6856f05f68f7a4316ee991e7a816` and clean Oct
`0da6044c6a5386b616ebb87ff18b64dea9b2269c`.
The Oct payload enum and fixture changes were committed as
`5ee3d0e7a5e989b9015d2b764eedc2c5d0235565`.

- General named aggregate construction now checks complete field coverage,
  duplicate/unknown names, ordinary field types, and moves. Generated
  `Fields<T>()` can expand directly into this syntax. A->B->C semantic
  artifact transport proves a generated constructor can be consumed without
  source reparsing. C11 lowering evaluates fields in source order and builds
  the aggregate only after every expression succeeds. A native regression
  caught and fixed a lost Drop when a later initializer propagated `?`.
- Oct accepts `Enum.Case(data)` recursively in its data-only parser and typed
  loader. The ordinary enum runtime value carries the payload. The writer and
  compiled Go data path emit/load the same syntax; the nested payload fixture
  executes interpreted and compiled with no fallback. Invalid arity, type,
  case, tag-only payload, and computed expressions have focused regressions.
- Standard now has an ordinary `OctagonCodec<T>` concept, direct byte-span
  reader/writer functions, and a reflected, generated integer-record codec.
  Its record round trip passes in `concept package test Standard`. This is
  real resumed codec work, though the codec currently covers integer fields
  only. No runtime DOM or reflection registry was introduced.

Remaining R7h1 work is explicit: other scalar codecs, payload enum codec,
arrays, columnar tables, refinement admission, cross-language Concept/Oct
goldens, and full deterministic/artifact qualification. This section records
progress, not a full R7h1 conformance claim.

### Qualification and next isolated boundary

The Concept writer's `Pair { X: 12, Y: -3 }` bytes match the Oct fixture
`Language/Data/Octagon/valid/concept_pair.octagon`; Oct loads the fixture in
interpreted and compiled execution. Concept reads those same bytes through
the generated reader. Its A->B->C aggregate artifact test passes, and a
100-run test compares generated aggregate artifacts and all emitted outputs
byte for byte. Oct's nested payload writer produces identical bytes over 100
runs.

`go test ./...`, `go vet ./...`, both Zig test lanes, BurnIn, Standard (16
facts), and DragonGod (21 facts) passed on the working changes. Oct Go tests
for parser, Octagon, interpreter, build, and CLI passed. The new payload and
record fixtures pass compiled without fallback. The wider compiled Octagon
corpus still fails on existing dimensioned scalar, nested record/array, and
record fixtures in its separate compiled data parser; interpreted execution
passes all ten facts. This is not a claim of full compiled Octagon parity.

The next codec blocker is type-directed field conversion. R7g1 can expose
field type and emit a complete aggregate, but Standard currently has only
`ReadNamedInt`/`WriteNamedInt`. A general typed codec dispatch for field
types is needed before generated payload, array, table, and refined fields
can be implemented without serializer-specific compiler branches. Keeping
integer-only generation explicit preserves a coherent checked library.

## Baseline and authorities

- Concept HEAD at the start: `1e96be2e1be9897207ab2d1ce0a9d6ad6ad98c80`; clean worktree.
- Compiler: `concept-evt1-stage0-go`; semantic artifact: `concept-module.v1`.
- R7fR anchor: `b516f40017d8b8b814b7c2b141f2ac3872aa07f4` (initial collector). R7g: `09fdc48107139152aa6179aa28eb3ec4f6e062fb`. R7g1: `6b845400769b7b95934545b71750668ad0010308`; HEAD also includes the follow-up `1e96be2`.
- Octagon authority read from the adjacent Oct repository's `Language/reference/tooling/34-octagon.md`, `internal/parse/data.go`, `internal/interpret/octagon_load.go`, and `internal/interpret/octagon_emit.go`. The data file contains one expression. Records carry nominal names and complete named fields. A record table adds exactly one array depth to every declared column. The current writer emits tag-only enum values as `Enum.Variant`; the data parser rejects calls, so it cannot admit payload enum construction syntax.
- Concept's `docs/language/TABLES.md`, `docs/language/CONCEPTS.md`, `docs/language/REFLECTION.md`, and `docs/design/EVT1-GENERATED-DECLARATIONS.md` describe the current table, proposition, reflection, and generated declaration authorities.

## Blocking reproducer

An R7g1 generator can expand statements over reflected fields, but it cannot form an aggregate initializer whose argument list is built from those fields. Concept requires every record field to be initialized before a value exists. This minimal attempt to construct a generated reader's result fails in the ordinary compiler path:

```concept
module R7hProbe; profile Core;
record struct Person { int id; bool active; }
generator <typename T> DeriveRead
T ReadOctagon() { T result = T{}; return result; }
derive DeriveRead reflect<Person>;
```

`concept check` reports:

```text
CV4126: wrong initializer count for Person: expected 2 but got 0
(generated ReadOctagon by DeriveRead at 5:1 from reflect<Person>)
```

`Fields<T>()` expands body statements such as `value.field = ...`, but it does not splice field expressions into a `T{...}` argument list. A caller-supplied initialized `ref T` would change the required reader contract and could bypass construction, ownership, and refinement admission. An uninitialized `Storage<T>` cannot be exposed field by field as an initialized `T`. Inserting a C-specific constructor or a serializer-specific MIR operation would bypass the ordinary Concept path. The missing capability is therefore a general, checked generated aggregate-construction form that can produce one complete initializer in declaration order and propagate failures without publishing a partial object.

The attempted alternative `T result; result.id = 1; result.active = true; return result;` initially panicked in template substitution because an uninitialized `VarDecl` has no expression to clone. That general compiler bug is fixed in `evt1SubstituteStatement` and covered by `TestGeneratedUninitializedLocalUsesOrdinaryChecker`. The ordinary checker now reports `CV4560: fixed storage local result requires an initializer`, with generated provenance. This removes the panic but confirms that field-by-field construction of an uninitialized local is not a legal reader implementation.

This is the request's stated honest-stop condition: generated declarations cannot presently express the typed reader. No Standard Octagon package or codec was added, because a writer-only surface would imply a viable bidirectional contract that the current language cannot fulfill. R7h remains unimplemented; no round trip, cross-language parity, or determinism claim is made.

## Additional Oct compatibility boundary

Oct's `validateDataValue` accepts `Enum.Variant` field access but rejects calls. Its loader explicitly reports `requires payload and is not supported in octagon data literals` for a payload variant, and `serializeOctagonValueAtDepth` emits only the variant name for an enum. Thus the current `.octagon` value surface does not round-trip payload enum contents. This must be resolved in Oct's data format authority before Concept claims payload enum parity. Tag-only enums remain representable.

## Baseline gates

At the baseline, `go test ./...`, `go vet ./...`, root `zig build test`, and `legacy/poc3-zig` `zig build test` passed. The adjacent Oct tool was built into a temporary executable in the Concept worktree to run `make BurnIn`, `make TestStandard`, and `make TestDragonGod`; all passed, with 14 Standard facts and 21 DragonGod facts plus one benchmark. After the compiler repair, the targeted regression, `go test ./...`, and `go vet ./...` passed again. No R7h conformance corpus exists yet.

## Required next design step

Define a general R7g1 extension that emits a checked, complete aggregate initializer from `Fields<T>()` without source-text generation. Prove it for imported `[[reflect]]` records, move-only fields, and a failed field conversion before implementing the Standard reader/writer. Keep Octagon grammar and columnar table semantics authoritative. Schema concepts should remain ordinary compile-time propositions, separate from the codec implementation.
