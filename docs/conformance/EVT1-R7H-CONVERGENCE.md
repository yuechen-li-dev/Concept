# R7h Octagon and Concept schemas: convergence log

## R7h2 derived codec continuation (2026-09-24)

The error-path continuation adds a bounded `OctagonError::At` payload with a
preserved error kind and up to 16 typed name/index parts. Ordinary Result
wrappers add record, field, array-index, case, and payload context at delegation
points. A strict-C11 regression verifies malformed nested input and a capacity
failure in a nested generated writer, each retaining
`Envelope.Items[index].Mode.Enabled.level`. A case-label capacity failure
retains `Mode.Enabled`. The first native run exposed that
the Standard codec helpers used `!`, which escalated errors before they could
be decorated. The library now uses ordinary `?` propagation, including inside
fixed-array construction. Paths beyond 16 parts retain the outer prefix.
The kind assertion in that regression exposed a general strict-C11 enum
equality bug: semantic checking accepted `==` on enums while lowering emitted
an invalid C struct comparison. Tag-only equality now compares variant tags;
payload enum equality requires an explicit comparison operation and reports
`CV4830`. Focused native and rejection regressions cover both cases.

The retained R7h2 array and table implementation was extended with generated
payload enum readers and writers. The earlier generator admitted only an
exhaustive match driven by an enum-valued parameter, so a reader could not
expand `Cases<T>()` into case tests and construct its result. The general
generator fix expands a reader's one-if case body into ordinary checked
branches, substitutes concrete `FieldType<field>` payload locals, and lowers
`ConstructCase<variant>()` to ordinary enum construction. Writer arms can now
contain ordinary statements around the payload loop. A regression retains a
malformed payload-query diagnostic. Standard's derived tag-only, multi-payload,
and nested-payload facts execute in strict C11.

A nominal single-field refined codec now reads its representation through its
ordinary witness and calls the author's `AdmitOctagon` operation. The missing
admission regression fails with generated provenance, while `Positive` accepts
42 and rejects zero. A separate ordinary schema concept proves `IsAdmitted`;
`concept explain` reports a missing operation as `CV4153`. This is a bounded
single-field wrapper derivation, not a general refinement declaration model.
The first field-loop implementation would have silently consumed one field of a
multi-field wrapper. The general `OnlyField<T>()` reflection query now rejects
zero or multiple fields during derivation; a regression pins the diagnostic.

An A-to-B-to-C artifact regression now executes its generated array, payload
enum, columnar table, and refined codecs in strict C11 without B source reparse
or generator replay. Its B semantic artifact, C MIR/C/header outputs, and
serialized concept-explain graph remain byte-identical over 100 runs. A
strict-C11 interop test runs generated readers
and writers for arrays, nested arrays, nested payload enums, columnar tables,
and refined values against exact shared fixture bytes. The native harness
repeats these byte comparisons 100 times, and the generated interop MIR/C/header
outputs are byte-identical across 100 compilations. Oct's compiled interop lane
loads the same shapes with five compiled cases and zero fallback. The
Concept refined fixture is LF-canonical: Oct's existing scalar fixture was
checked out as CRLF in this worktree, which did not match Concept's canonical
writer. No Oct source or fixture was changed.

The generated codec family now retains nested field/index paths. A local `NoAllocation` proof succeeds for the
fixed-storage scalar `ReadInt` and `WriteInt` operations. An imported operation
without a transported effect summary is `UNKNOWN`, so a blanket generated-codec
proof remains unqualified. The codec family is persistent
in-tree and exercised through real C11 and Oct paths.

## R7h2 continuation: closed generic witness calls (2026-09-24)

Baseline was clean Concept `4024e5fbf2491a9fa2344df7839e023d06b3f52e` and
clean Oct `81ba6eb87e5d2777df27991265b93f228f7edf89`. Oct source was not
changed. The independent reproducer is in `r7h2_witness_test.go`: a
`Codec<int<array>[4]>` requirement was accepted while the ordinary
`Encode(ref const values)` call failed with `CV4173`, demanding explicit
template arguments. This was a proof/call split, not an Octagon format issue.

Closed template argument inference now serves both required-operation proof
and ordinary call resolution. It binds type parameters and fixed extents from
the same parameter pattern, reopens nominal generic applications for
inference, checks the resulting closed signature, and then uses the ordinary
template instantiator. The instantiator still checks constraints and uses its
existing recursion guard and deterministic identity (`template name`, type
arguments, value arguments). The obsolete single-type-parameter proof helper
was removed. A handwritten strict-C11 call executes the inferred `int, 4`
instance, and its generated outputs are byte-identical over 100 runs.

The full Go gate exposed an artifact transport regression: the existing
generated Trace fixture carries `CollectorHandle<Node>` as a closed imported
nominal instance, without its open generic application in the consumer's
local map. Pattern inference initially rejected this required operation and
selected no reflected fields. The general fix compares the nominal instance
against the same pattern after earlier parameter positions have closed all
its arguments. The artifact-only Trace regression and R7h2 artifact chain
pass together. This does not parse a nominal name string.

The first derived codec blocker was that `ReadNamed`/`WriteNamed` had only
scalar overloads. Ordinary constrained generic overloads now delegate through
`OctagonCodec<T>`; the compiler has no Octagon-specific path. Generated
record fields round trip for `int<array>[3]`, `bool<array>[8]`, arrays of
generated `Pair` records, and nested arrays via a `Row` alias. A generic
array-element reader delegates to the ordinary codec witness. The existing
record generator also derives `record table<3> Catalog` column-wise, with
canonical bytes equal to the handwritten table fixture. All these run in the
Standard strict-C11 fact lane.

An A->B->C semantic-artifact regression exports the generic witness from A,
derives a field call in B, and consumes B's generated function from C without
source reparse or rerunning B's generator. The independent native fixture
qualifies execution; the artifact regression currently qualifies semantic
transport and output generation. Generic operation witnesses from R7f1 remain
in the full Go regression lane.

Remaining R7h2 work is explicit: general payload-enum reader/writer
derivation, refinement admission, full Concept/Oct bidirectional fixtures,
an executable artifact-only linked program, and the full 100-run multi-module
artifact/MIR/C/header/explain/fixture gate. Generated nested error paths and
NoAllocation proof are also not yet qualified. Current work meets the milestone's
meaningful-progression threshold through closed witness invocation, generated
array fields, and nested arrays; it is not full R7h2 conformance.

Final qualification for this progression: `go test ./...`, `go vet ./...`,
both Zig test roots, `oct make BurnIn`, and the Concept Make Standard and
DragonGod build/test graph passed. Standard ran 25 facts, DragonGod 21 facts
and one benchmark. Adjacent Oct parser/interpreter/build/CLI Go tests passed
at the unchanged baseline SHA. `git diff --check` passed; no compiler
Octagon-name branch was added.

## R7h1 continuation: fixed arrays and shared payload/table goldens

This continuation started from clean Concept `60ded73389687888a873ec369cd25ebf9f74c9b1`
and clean Oct `8a4fee3bb3fc503c5fc392b9a5bc9e467add4b2c`.
The Oct fixture, compiled table round trip, writer checks, and LF pinning are
commit `81ba6eb87e5d2777df27991265b93f228f7edf89`.
`Standard.Octagon.Core` now has scalar `OctagonType<T>` operations and a
fixed-array value reader/writer using complete array construction. An element
failure propagates before a completed array becomes live. The writer emits a
composable value, and `FinishOctagon` supplies Oct's one canonical document
newline. The strict C11 Standard test lane covers `[1, 2, 3]`.

Two representative ordinary typed codecs now exercise nested payload enums
and `record table<3>` with full `ID` and `Active` columns. Concept's output
matches Oct's payload and table fixtures byte for byte. Oct loads both in its
interpreted and compiled paths; the compiled valid load corpus has 11 facts
and zero fallback. Oct's writer reproduces the same payload/table fixture bytes
in 100 consecutive runs. These codecs are handwritten `OctagonType<T>`
overloads, so they do not establish general reflection-derived enum/table
families.
Oct's compiled `LoadOctagon<Catalog>` → `WriteOctagon` integration test also
reproduces the Concept table fixture byte for byte. The existing scalar-array
fixture was pinned to LF in Oct so its checked-out bytes match the canonical
writer on Windows; its writer has its own 100-run byte comparison.

The array work exposed and fixed a general strict-C11 lowering bug: closed
generic `ref const T<array>[N]` parameters were omitted from storage-view
typedef collection. A native harness now asserts their declaration and
execution. The next blocker is precise: `requires OctagonCodec<int<array>[3]>`
reports CV4154 because closed concept witness lookup recognizes only a single
unconstrained type-parameter template, while the array codec is constrained on
its element and has an extent parameter. Generated `ReadNamed` cannot select
the array family until closed template witnessing and generated typed dispatch
can carry this proof. General payload enum reader derivation also needs
compile-time case selection and typed payload construction; the current
`Cases<T>()` generator expansion produces a writer-style match only.

A bounded attempt to recognize constrained two-parameter array templates as
closed concept witnesses made the proof pass, but an actual generated helper
still failed at the required operation call: it could not select the array
template without explicit element and extent arguments. That partial proof
change was removed rather than leaving a concept witness that cannot execute.

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

### R7h1 continuation: typed field dispatch and compiled Oct corpus

`FieldType<field>` now substitutes a reflected semantic field type into an
ordinary type position. `OctagonType<T>` disambiguates reader overloads;
normal type checking selects the `int` or `bool` implementation. A mixed
record with two generated readers in one module passes the strict C11
Standard test lane. A separate native regression proves `FieldType` selects
ordinary overloads and uncovered a reflection-probe aliasing bug: the probe
had rewritten the real function signature's shared generic type nodes. The
probe now deep-clones signature types before analysis. A second regression
derives the mixed-field function from an imported `[[reflect]]` artifact and
consumes the generated function from a downstream artifact without source
reparsing. Generated mixed-field C output is byte-identical over 100 runs.

Oct's compiled data parser now accepts parenthesized data and dimensioned
numeric literals and checks their declared numeric dimensions. It also
rejects duplicate record fields. The full Octagon Load valid corpus now
passes compiled (10 facts, zero fallback), as does the invalid corpus (8
facts, zero fallback). The formerly failing dimensioned scalar, nested
record/array, and record fixtures are covered by these lanes and a dedicated
compiled dimensioned-record integration regression. This fixes the compiled
corpus failures recorded above; the earlier failure record is retained as
history. The Oct fix is commit
`8a4fee3bb3fc503c5fc392b9a5bc9e467add4b2c`.

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
