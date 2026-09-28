# R8e convergence record: MustUse/discard seam

Baseline: clean R8d `0c87962fe4a8d88adeab1c7afd85ebe0c62299eb`.
Compiler: `concept-evt1-stage0-go`.

Baseline inventory: `manifest.concept` is an ordinary Concept module whose
package path uses an immutable six-field `PackageManifest Manifest` value;
the package loader extracts that exact shape. `concept check` parses and
semantically validates one source or checks a native project. No formatter or
`concept lint` command exists at baseline. Ignored `Result` was accepted as
an ordinary expression statement. Foreign companions already provide trusted
declared contracts, and checked generated functions retain `GeneratedOrigin`.

This pass establishes an unconditional semantic obligation for ignored
`[[must_use]]` values and the `discard expression;` statement. It covers a
non-Result nominal type, `Result<T, E>`, an attributed function or method
returning a plain integer, a closed generic return, type aliases, and artifact-only
consumers. A foreign `extern "C"` return can carry the same attribute through
an artifact. Normal and Verify C11 harnesses verify one side effect and one
Drop for explicit discards. A 100-run test checks artifact, MIR, generated C,
and diagnostic byte identity. Existing effects `discard(batch)` remains a
call. Fixtures that intentionally ignored `Result` now say so explicitly.

The current source parser discards comments and whitespace in `lexEVT1`, and
there is no `concept fmt` implementation (see
`docs/Concept-Source-Formatting-Inventory.md`). The current `concept`
declaration model expresses type and operation requirements, but has no
semantic subject for a declaration kind such as “authored function name” or
“generated field name.” These are the next concrete design seams for project
policy concepts, manifest binding, naming lint, and a trivia-preserving
formatter. A regex over source text would not satisfy the requested policy
authority. No R8e completion claim is made here.

## Verified gates for this bounded progression

| Gate | Result |
| --- | --- |
| Focused R8e tests | Pass: MustUse, artifact-only, foreign, generic, alias, method, exact-once, Drop, first-contact, 100-run determinism |
| Strict C11 Normal / Verify harnesses | Pass: one side effect, one Drop |
| `go test ./... -count=1` | Pass (183.444 s for `internal/concept`), including R7p, R7q Frame, R8a–R8d, and semantic corpus |
| `go vet ./...` | Pass |
| Root and legacy Zig `zig build test` | Pass |
| BurnIn `go test ./internal/concept -run R7d3 -count=1` | Pass |
| Standard Normal / Verify | 35 / 35 in each mode |
| DragonGod Normal / Verify | 23 / 23 in each mode |
