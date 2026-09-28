# Claude reinterpretations of the Golden programs

**R8g follow-up (2026-09-28):** The final original and Claude sources now run together: 130 passed, 0 failed, 2 benchmarks in both Normal and Verify. All 61 final `.concept` and `.concept_test` files pass per-file `concept format --check`; first drafts remain historical evidence. The original behavior bugs described below have been fixed and pinned in original runtime tests. The resolved/deferred issue ledger is [R8G-CONVERGENCE.md](../conformance/R8G-CONVERGENCE.md). The comparison below records what the Claude pass found at first contact; superseded limitations remain as historical evidence.

All thirteen Golden domains, the eight R7p migration goldens and the five R8f
differentiators, rewritten the way Claude would write them. They live in
`libraries/Golden/Claude/`, mirroring the original layout. Each domain keeps
its `first-draft.concept.txt`, the code as written before any compiler
feedback, next to the final source.

| | Originals | Claude versions |
| --- | ---: | ---: |
| Test functions | 38 | 83 (88 results; one theory has 6 rows) |
| Normal / Verify | pass / pass | pass / pass |
| Final runtime results | 42 facts + 2 benchmarks | 88 facts; combined tree: 130 facts + 2 benchmarks |

`TestR7pDomainGoldensNormalAndVerify` passes over the combined tree. Its
accounting compared raw result counts with discovered tests, which a
`[[theory]]` breaks (six row results, one test). It now requires every
discovered test ID to report and none to fail, the same class of fix R8f
made for benchmarks.

The final sources now pass `concept format --check`; only first-draft `.txt` files retain their original layout.

## How to run

```powershell
$env:CONCEPT_MODULE_ROOTS = (Resolve-Path 'libraries').Path
go run ./cmd/concept test libraries/Golden --filter Claude --verbose
go run ./cmd/concept test libraries/Golden --filter Claude --verify
go test ./internal/concept -run '^TestR7p' -count=1
```

## What changed, and why

The originals read like competent GPT code: plausible, documented in the
README, and shaped by the path of least resistance through each diagnostic.
The rewrites apply a different set of habits.

| Habit | Original | Claude version |
| --- | --- | --- |
| Name invariants, not numbers | `32`, `64`, `0x50`, `0/1/2` token kinds inline | `QueueCapacity`, `PingReply`, `Operator` enum, `CommitTicks` |
| One error per failure | `CacheError::Missing` for a bad offset; `DirtyPage` for a pinned page; `SensorInvalid` for an empty history | `OffsetOutOfRange`, `Pinned`, `NoSamples`, and so on |
| Tests check *which* error | `Assert.Error(...)` everywhere | Direct built-in `Assert.FailsWith(result, Error::X)` after F8 |
| Make bad states hard to build | Unchecked `AdvanceHeat(diffusion)` beside a checked wrapper | `Diffusion` can only come from `StableDiffusion()`; the kernel takes a `Diffusion` |
| Separate deciding from doing | Utility scoring inside a machine state | Pure `Choose()` tested on its own; the machine only sequences |
| Derive state, don't duplicate it | Ring with `head`, `tail` *and* `count` | `(head, count)`; tail is computed |
| One function per idea | `ReduceAdd`/`ReduceMul`, `Drain`/`DrainSignal`, `CountPositive`/`CountDegenerate` | `Shift`, `DrainInto`, `TakeCensus` |
| Loops say why they stop | `for (i in 0..N) { if (active) {...} }` flags | `while (cond) bounded(N)` with the reason in the condition |
| Test edges, not the happy path | Mostly a single scenario per fact | Wraparound, capacity plus one, boundary rows, order independence |
| Tests read as sentences | `CacheHandlesRejectStalePages` covering 10 behaviours | `DirtyPagesStayUntilWrittenBack`, `OneSampleNeverSkipsAPhase` |

## Behaviour bugs in the original goldens

These go beyond style. Each was first fixed in the Claude version and is now also fixed and pinned in the original Golden.

| Domain | Defect | Pinning test |
| --- | --- | --- |
| Hft | `head`/`tail` are `AtomicInt` counters that grow forever; signed overflow traps, so the queue panics after 2^31 events. | `SequenceNumbersWrapWithoutLosingOrder` (sequences kept in `[0, 2*Capacity)`) |
| Aerospace | Mode updates are sequential `if`s, so one sample can go Preflight → Climb → Cruise at once. | `OneSampleNeverSkipsAPhase` |
| Aerospace | History refuses samples after 16 (`CapacityExceeded`): telemetry stops 16 samples into a flight. | `HistoryKeepsTheLatestSamplesForever` (ring) |
| Game | `Seek` only increments, so agents never move left or down. | `SeekersStepTowardTheirTargetFromEitherSide` |
| Game | `Strike(int)` compares the target with the loop ordinal; the test is a self-strike. | `MutualStrikesBothLand` (`Strike(Id<Agent>)`, two-phase resolve) |
| Cad | `SignedDoubleArea` `!`-panics on a bad face ID although it declares `FaceMissing`. | `AFaceIdFromNowhereIsReportedNotTrusted` |
| Async | `Drive` calls `Result()` after a `bounded(8)` loop; a task needing more steps would read an unfinished result and panic. | `TooSmallABudgetStallsInsteadOfPanicking` |
| Storage | The test reaches into store internals to clear `dirty`; the cache has no way to become clean. | `DirtyPagesStayUntilWrittenBack` (`WriteBack`) |

## Compiler defects

**B1: array literal in a returned or nested constructor gets the wrong element type.**
It type-checks, then emits C that doesn't compile:

```concept
struct Inner { uint8[4] bytes; int count; }
Inner Make() { return Inner{[0, 0, 0, 0], 0}; }   // C: concept_array_4_int undeclared
```

The same literal works as a local initializer. It also fails as an argument
to a nested constructor (`Outer{Inner{[0 ... 4], 0}}`). Worked around in
`Embedded/Serial.concept`.

**B2: quantities silently lose their unit through `float`.**

```concept
float Plain(float x) { return x; }
float A(float<m> d) { float x = d; return x; }     // accepted
float B(float<m> d) { return Plain(d); }           // accepted
float C(float<m> d) { return d as float; }         // rejected: use Magnitude(value)
```

Metres to seconds is rejected, but metres to dimensionless is implicit. As a
result, `Standard.Math` `Min`, `Max` and `Abs` accept `float<m>` and return
`float`. Explicit erasure is correctly forbidden while implicit erasure is
allowed, so the unit system can be bypassed without writing anything
suspicious. I'd treat this as the highest-priority finding here, since unit
safety is a headline feature.

## Frictions, ranked

Each row names the goldens where it forced a workaround, and whether it
explains a pattern in the originals that looked like a GPT habit.

**F1. Float literals only take their type from a direct initializer.**
`double x = 1.0` works; `double x = -1.0`, `c*c + s*s - 1.0`, `F(1.0)` for a
`double` parameter, `x == 20.0`, and tensor literal elements do not. Integer
literals are typed from operands and parameters. This is the whole source of
the `as double` noise in R8f. *Hit in:* Mechanics, Dot. *Suggest:* give float
literals the integer-literal rules, including through unary minus.

**F2. No operators on open generic types.** `error == expected` for an open
`F` and `a * b` for an open `T` both fail with `CV4028`. That forces
`Multiply`/`Add` wrappers in HPC and per-type `FailsWith` overloads in every
test file. *Suggest:* equality and arithmetic as requirable operations
(`requires T operator*(T, T)` or a built-in `compiler.Arithmetic<T>`).

**F3. `?` cannot change the error type, and a borrowed `Result` cannot be
re-wrapped.** Re-wrapping `Result<ref Page, StoreError>` as
`Result<ref Page, CacheError>` fails with `CV4522`, because the borrow's
provenance is lost through `match`. This is why the originals repeat the
`match … Result::Error(error) => return CacheError::…` block at every call
site. The Claude versions use one adapter per store for values (`PointAt`,
`FaceAt`) and "check residency, then borrow with `!`" for references.
*Suggest:* `expr ? else Error::X` or a provenance-preserving `MapError`.
*Originals' pattern forced:* yes.

**F4. Enums have no fixed element geometry.** A struct containing even a
payload-free enum can't be a `Span` element (`CV4601`), so tokens hold `int`
codes, and `ReadOnlyValues<Agent>` is impossible because `Agent` holds an
`Intent`. *Suggest:* payload-free enums get integer geometry. *Originals'
pattern forced:* yes (Postfix's `int kind`).

**F5. No multi-way branch over integers.** `else if` is rejected with "use
`match` for multi-branch selection", but `match` has no integer arms
(`CV4010`). The diagnostic points to something that doesn't work. Byte
decoders and sign functions become early-return ladders. *Suggest:* integer
and literal patterns in `match`, or allow `else if`, and fix the diagnostic.

**F6. `comptime` constants are second-class.** They support no unsigned
types (`comptime uint8`/`uint32`: `CV4216`), can't be template arguments
(`GENERIC_NON_TYPE_ARGUMENT_INVALID`) although they work as array extents,
and aren't contextually typed (`flags & Mask` fails). So
`GenerationalStore<Page, 8>`, `Insert<Page, 8>` and `Remove<Page, 8>` must
repeat the literal. *Originals' pattern forced:* yes; much of the
"magic number" style was the language's doing.

**F7. `bounded(N)` truncates silently.** Exceeding the bound just ends the
loop, in both Normal and Verify. That's harmless for `Flush`, but dangerous
wherever the loop was expected to finish (the Async bug above). *Suggest:*
trap in Verify, or `bounded(N) else { … }` so the overflow case is written
down.

**F8. Test surface gaps.**
- `Assert.Equals(u32, 0)` fails: the literal isn't typed from the other side,
  hence all the `uint8 expected = 0x7F;` locals.
- `Assert.Near` rejects `double` and quantities (`Magnitude(x)` is required).
- There's no assertion that checks *which* error a `Result` holds.
- `Result::Ok(x)` inside a `match` expression arm gets no expected type
  (`CV4540`).

**F9. The formatter makes code worse.** It turns `Shape{0, 0}` into three
lines, `used++` into `used ++`, and `if (x) { return 0; }` into three lines.
It changed 22–31 lines per file on the new goldens, all for the worse. That's
why these files are unformatted. *Suggest:* keep short brace initializers
and single-statement blocks on one line; no space before postfix `++`/`--`.

**F10. Smaller items.**
- `concept test` can't be pointed at a sub-directory; a package's own
  sibling modules only resolve from the package root.
- `Subspan` needs a named span, not a temporary (`CV4600`).
- `Standard.Math` is `float`-only and erases units (see B2).
- Standard atomics are `AtomicInt` only.
- `GenerationalStore` has no live-count query.
- Generators can't take the attribute name as a parameter: `OperandCount`
  and `SuccessorCount` are identical apart from `operand` vs `successor`.
- `instruction.field.value` reads as access to a member named `field`.

## What worked well

- **HFT, Squad and IR passed on the first compile.** SPSC ownership proofs,
  `infer` inside an ordinary function, and two reflection-derived passes
  under `compiler.Generated` all worked as documented.
- **Units compose where they are used.** Kelvin-typed table columns,
  `double<Pa^2>` from a product, `comptime float<m>` thresholds, and
  Einstein contraction over `double<Pa>` all work.
- **Diagnostics usually point at the fix:** `CV4501` (use `move`), the
  `Magnitude` hints, `CV4142` (record fields are read-only), and
  `ASYNC_PERSISTENT_REF_ESCAPE`.
- **The borrow model rewards a good design.** The two-phase game tick, which
  is order-independent by construction, is the natural shape in Concept.
- **Floating addition order is controllable.** Per-type `Reduce4` closes to
  separate C bodies, and a test proves the binary32 order is observable:
  products `1e8, 1, -1e8, 1` give `0` in binary32 and `2` in binary64.
- **Verify agrees with Normal** on all 88 results.

## Suggested order

1. **B2** (unit erasure) and **B1** (wrong C for valid code): correctness.
2. **F1** (float literal typing): one rule change, and removes most visible
   noise.
3. **F3 and F4** (error conversion; enum geometry): remove the two biggest
   structural workarounds.
4. **F9** (formatter): showcase code should look its best.
5. **F2, F5, F6, F7**: language-surface work, larger in scope.

A finding worth keeping: roughly half of what looked like "GPT style" in the
originals was forced by the language. That includes repeated literal
template arguments, `int` token codes, and remap-`match` at every call site.
Fixing F3, F4 and F6 would change how any model writes Concept, not just
Claude.

## First R8g batch status (2026-09-27; superseded by follow-up above)

Implementation commit `cb82976c9547c883aaf4cac6f192a7d2f1bcbda8` fixed
B1, contextual float literals (F1), borrowed Result `? else` remapping (F3),
and fixed geometry for payload-free enums (F4). It closed the implicit
quantity-erasure paths in B2. `Standard.Math` now rejects quantity arguments
to plain scalar signatures; dimension-preserving generic Abs/Min/Max remain
open with F2 operator requirements. Match-expression arms now inherit a Result
expected type, resolving that part of F8.

The final Claude Postfix source now uses `TokenKind` instead of the three
integer token codes. Its first draft remains unchanged. Payload-bearing Game
`Intent` still has no scalar geometry; that Golden retains its indexed loop.
All 88 Claude runtime results and the combined Golden Normal/Verify tree pass.
The stricter quantity rule also required old layout-test callers of scalar
`usize` APIs to write `Magnitude(SizeOf<T>())`; F4 required regeneration of
the checked EVT1 C outputs and manifests.
The remaining R8g queue and gate evidence are recorded in
`docs/conformance/R8G-CONVERGENCE.md` and `R8G-CONFORMANCE.md`.
