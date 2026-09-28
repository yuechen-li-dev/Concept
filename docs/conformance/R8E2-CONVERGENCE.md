# R8e2 convergence: bound declaration subjects

State: **meaningful progression**, not R8e2 completion.

Baseline: clean `86a8105b5098fd5607bf6a88fc0644934bbe60bd`.
Compiler: `concept-evt1-stage0-go`.

The first blocker named by R8e1 was the absence of semantic declaration-kind
subjects. The validated module now projects the nine requested kinds with
stable IDs, source sites, defining-module ownership, and
Authored/Generated/Foreign provenance. Import loading reattaches owner identity
for concept, interface, machine, and template declarations without changing
artifact serialization. The project-scoped projection excludes dependency
implementation declarations while retaining them in the composed semantic
module for ordinary proof queries.

Focused regressions cover a bound type, field, method, local, parameter,
concept, interface, machine, function, generated function, foreign function,
standalone file, and an artifact-only consumer. The 100-pass inventory check
pins stable order and identity. These tests do not claim policy enforcement.

Validation on this host:

| Gate | Result |
| --- | --- |
| Focused R8e2 tests | Pass |
| `go test ./... -count=1` | Pass, including R7p, R7q Frame, R8a–R8e1, and the semantic corpus |
| `go vet ./...` | Pass |
| Root and legacy Zig `zig build test` | Pass |
| BurnIn (`go test ./internal/concept -run R7d3 -count=1`) | Pass |
| Standard Normal / Verify | 35 / 35 in each mode |
| DragonGod Normal / Verify | 23 / 23 in each mode |

The next blocker is specific: `parseConceptDecl` in `internal/concept/parse.go`
accepts only type parameters, and `evt1BuildConceptAssertionGraph` in
`internal/concept/concept_assert.go` sends named concept arguments through
`[]Type`. A declaration subject cannot currently be passed to an ordinary
concept requirement. Manifest binding, severity, naming enforcement,
NoAllocation policy, `concept lint`, and policy explanation depend on that
semantic binding. Implementing a special-case AST naming visitor would bypass
the requested authority, so this progression stops at the reusable subject
model.

R8e3 and R8f were not started.
