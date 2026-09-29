# EVT1 R7n convergence

Baseline: clean `bf84656b47c79b1d623b1f7593cceaa46573a1ea` after R7m3.
Compiler ID: `concept-evt1-stage0-go`.

| Awkward compiler-authoring pattern | Abstraction or fix | Evidence |
| --- | --- | --- |
| AST nodes used untyped integer positions | Nominal `Id<T>` and append-only `DenseStore<T, Capacity>` | Standard AST fact traverses `Add` edges through checked IDs |
| Reused positions could alias stale handles | `GenerationalId<T>` with occupied metadata, LIFO free stack, increment or retirement | Standard stale access, double removal, reuse, and retirement facts |
| Indexed owned option replacement rejected a payload without custom Drop | General owned failure-payload drop/replace authority in `failure.go`, excluding explicit `Storage<T>` authority | Plain `Expr` artifact consumer and static authority regression |
| Borrowed enum matching emitted `.` against a C pointer | General reference-aware match lowering in `generate.go` | Native AST traversal fact |
| Zero-argument `Option::None` / `Result::Ok` constructors emitted old-style C prototypes | Emit `(void)` in `failure.go` | Artifact-only strict C11 check |
| Imported generic `GetIterator` was not resolved as a `foreach` source | Explicit `Count` and checked `IdAt` loop | Native zero-allocation ID walk; iterator sugar remains deferred |
| `Result<ref const T, E>` in `Assert.Error` emitted invalid pointer member syntax | Match the result explicitly in the fixture | Checked invalid IDs run natively; assertion lowering remains a separate issue |
| Inline owned option slots are strided payloads | Documented ordinal/ref traversal | No false `Span<T>` claim |
| Immovable `T` cannot cross by-value insert | Reject the attempted insertion | Static artifact-only negative test |

The bounded library, compiler substrate repairs, and dogfood remain in-tree.
Clang's `-Werror` gate suppresses only `-Wunused-function` for unused emitted
generic ID constructors; `-pedantic-errors` and other warnings stay active.
The next blocker is a sound stationary initialization representation that can
own immovable values while exposing a contiguous payload span. A separate
arena or graph framework would not resolve that representation problem.

## R7n2 follow-up

| Awkward pattern | Substrate fix | Evidence |
| --- | --- | --- |
| `Initialize(Storage<Immovable>, T{...})` lowered through a complete aggregate assignment | Evaluate all fields, then write directly to the final storage address | Strict C11 immovable fixture; generated C checked for whole-object assignment |
| A later fallible field could abandon an earlier owned field | Register field temporaries for normal cleanup until commitment | Failing and succeeding constructor paths observe exactly two Drops |
| `Destroy` of an immovable aggregate with an owned field skipped structural cleanup | Reuse structural `lowerDropValue` for typed storage destruction | Successful path destroys its owned field once |
| A borrowed record reference was considered an owner for recursive cleanup | Exclude `ref T` from generated Drop traversal | Native strict C11 fixture with a borrowed immovable value |
| Fixed layout rejected an owned field despite unchanged physical representation | Resolve the value type for field geometry | `SizeOf<Pinned>` accepted for bound storage |

The R7n2 blocker was store-owned, partially initialized contiguous storage.
`Option<owned T>[Capacity]` cannot form `Span<T>` because its payloads have a
tagged stride. A generic raw byte array sized by `Capacity * SizeOf<T>()`
currently rejects with `CV4200` because the capacity name is unavailable in
that compile-time extent; `SizeOf<T>()` alone rejects with `CV4573` because
open `T` has no fixed layout geometry. A sound solution must also constrain
moving a store after an immovable entry becomes live. R7n3 implements the raw
and sparse backing described below.

## R7n3 closeout

| Awkward pattern | Local substrate or library fix | Evidence |
| --- | --- | --- |
| `Option<owned T>[N]` makes dense payloads strided by tags | General `T<raw>[N]` with typed contiguous data and one live-prefix count | Native raw prefix and Standard Span facts; generated C has `T data[N]; int count` |
| Final-slot construction of an immovable value could not cross a by-value store API | Stationary `Emplace(ref store, T{...})` and raw append commit after completed field initialization | Artifact-only immovable dense fixture in Normal and Verify |
| Failed later field left a prior owned temporary live after an always-returning `if` | Correct live-owner merge in `lowerIfStmt` | Failed initializer observes exactly one Drop and no live store element |
| Generic fixed array extent failed to substitute `Capacity * SizeOf<T>()` | Recursive value and type substitution in fixed extents | Generic layout regression in Go suite |
| A wrapper around generic `ReadOnlyValues` lost the returned Span's parameter provenance | Derive return provenance through template calls and raw views | Standard `StatementValues` wrapper and `NoAllocation` fact |
| An allocation inside a stationary aggregate field was omitted from a `NoAllocation` proof | Traverse aggregate, array, and conditional initializers for direct calls and template instances | Allocating `Emplace` field disproves a false `NoAllocation` assertion |
| `Option<owned Immovable>` rejects the payload before a generational store can construct it | General `T<sparse>[N]` with checked live bits; preserve generation and free-stack policy | Artifact-only fresh/reused sparse Emplace, failure cleanup, stale handle, and Drop evidence |

DenseStore owns contiguous raw storage for Capacity values and maintains a
live initialized prefix `[0, Count)`. Only that prefix may be viewed as
`Span<T>`. GenerationalStore intentionally trades contiguity for stable
reusable identities and therefore does not expose a contiguous payload span.
Stationary insertion constructs the object directly in its final storage and
therefore supports immovable values without relocation.
