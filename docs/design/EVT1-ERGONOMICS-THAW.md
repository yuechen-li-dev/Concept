# Bounded EVT1 ergonomics thaw — 2026-11-05

Authorized by the user for the three concrete Prometheus authoring frictions.
This is an explicit post-freeze extension, not a reopening of unrelated EVT1
semantics or the deferred native/foreign ABI surface.

## Scope

1. Supply ordinary function parameter context to `Option`/`Result` arguments,
   recursively through payloads, parentheses, conditional and match expressions.
   First narrow targets by existing ordinary argument compatibility and scores;
   require all remaining targets to agree on each contextual parameter. Preserve
   `CV4182` ambiguity, payload diagnostics, integer bounds and explicit moves.
   Explicit generic calls already have closed parameter context; ordinary calls
   inside templates use the same rule. Inferring an otherwise unknown generic
   argument from an untyped carrier remains outside this change.
2. Add `export "C"` bodies with exact non-overloaded C/C++ identifiers. Emit a
   standalone `.exports.h` and include its hash in the generation manifest. The
   initial subset is scalars, foreign handles and opaque `ref`/`ref const` concrete
   object parameters. No by-value private objects, owning parameters, borrowed
   returns, implicit allocation, automatic failure translation or C++ object ABI.
   Caller lifetime/non-null/type/aliasing/synchronization obligations remain
   explicit. Storage construction and disposal stay with their existing owner.
3. Promote existing `T[N]` to a first-class spelling alongside `T<array>[N]`.
   Prefer concise authored fixed arrays, including nested generic element types.
   No parser storage extension, representation change or blanket source rewrite.

The public interface is currently an EVT1 C11-backend contract. Cathedral rejects
export-bearing code with `EVT2_C_EXPORT_UNSUPPORTED`; its native export linker and
cross-target object/lifetime ABI need independent qualification later. Ordinary
non-export code keeps its existing path and checked artifacts.

Export linkage uses a compiler-owned marker in the existing transported attribute
metadata. The parser rejects direct source spelling of that reserved marker.
Adding a field to gob-serialized `FunctionDecl` changes even non-export module
identities, so the implementation deliberately preserves that frozen schema.
The unchanged `TestEVT2h3Transport` gate confirms both frozen hashes and all
100 fresh-generation comparisons. No frozen artifact or codec was rewritten.
Test validation and discovery recognize this as semantic metadata, rather than
a test annotation; the export fixture has exactly one discovered fact.

## Motivating consumer

Oct's `PrometheusModelBlock.concept` uses concise arrays and explicit exported
owner operations. C no longer inspects the owner's buffer carrier slots, pending
submission carrier tags, flags, completion state, counters or timing arrays.
It retains a private allocation/result-construction seam and a projection of
borrowed native frames/recordings. Removing that last construction seam requires
an explicit owner allocation/lifecycle design; hidden heap boxing is not a fix.

## Evidence

Runtime contracts: `tests/runtime/contextual_carriers.concept_test` and
`tests/runtime/c_exports.concept_test`. Go boundary checks:
`TestErgonomics*`, including malformed signatures, payload/move diagnostics,
array representation equivalence, formatter and semantic-module roundtrips,
strict C11/C++23 consumers using only the opaque header, and three fresh complete
generations per Normal/Verify mode. The native-only unqualified export boundary
has an explicit rejection test. Final command outcomes are recorded in the
milestone verification report alongside the Oct GPU consumer evidence.

## Milestone checks

The real CLI executes four carrier/array/projection facts and one export-body
fact in both Normal and Verify modes (ten fact executions). `TestErgonomics*`
passes, including independent C11/C++23 consumers and three fresh complete
generations per mode. Focused failure/context/extern/storage/scoped-cleanup
regressions and unchanged checked artifacts pass; `go vet ./...` passes.

The real Oct Windows native build and all 29 selected shared-device GPU/
integration cases pass, zero skips and no Vulkan synchronization/lifetime
findings. Its detailed producer/consumer evidence is recorded in Oct's
`docs/internal/concept_evt1_ergonomics_thaw.md`. This is **MILESTONE VERIFIED**
for the declared C11/public-header/Prometheus subset; complete checkpoint
qualification is not claimed. Frozen Zig paths were unchanged and not run.

All 22 selected ergonomic/test-runner/native match/Option/Result regression roots
pass, with independent MSVC/Clang ABI probes and Normal/Verify execution. The full
short compiler sweep exposed export-marker discovery treating C entry points as
tests; this was fixed, and the complete Quick runtime/backend/CLI lane passes.
Final focused checks also reject a mismatched already-typed conditional arm and
retain parenthesized live-borrow rejection. Frozen match/Option/Result transport
checks pass after those typing checks; no native code or checked artifact changed.
The original interrupted unfiltered run and the initial failed short sweep remain
recorded as such; a complete unfiltered rerun is not claimed. See the consumer
report for commands, timings, evidence paths and the reason unchanged native
execution results apply to the final code.
