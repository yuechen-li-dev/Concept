# R8e2b convergence: declaration concept parameters and project policy

Baseline: clean R8e2 progression `504ce599ca79b2ff782d2041a292e78095acad43`.
Compiler: `concept-evt1-stage0-go`.

The R8e2 blocker was concrete: ordinary concept parameters and named proof
applications used only `Type`. `declaration D` is now a distinct parameter
category and binds `DeclarationSubject`, whose ID includes defining module,
kind, name, provenance, and source site. Explicit
`requires Policy<declaration Name>;` and ordinary `Assert.Concept<Policy>(Name,
"reason")` use the same proof graph as type concepts. Type/declaration category
mismatches have a dedicated diagnostic. Composed declaration concepts and
artifact-only foreign/generated arguments are regression covered.

`compiler.CanonicalName(D)` queries semantic kind, provenance, and identifier
spelling. It exempts Generated and Foreign under the authored default. Manifest
`LintPolicy` values bind named ordinary concepts, warning/error severity, and
bounded exact semantic selectors. `concept lint` evaluates the same concept
proof projector; it emits stable findings and detects incompatible name styles.
`concept explain --policy` exposes policy source, severity, subject provenance,
and proposition truth/evidence. `concept explain --must-use` projects the core
attribute and its artifact/foreign provenance. Neither path changes truth.

The flagship demo proves NoAllocation for `FastFunction`, disproves it for
`AllocatingFunction` through declared `Acquire Allocates`, and leaves an opaque
foreign function Unknown. Naming violations are warnings. Generated and
foreign spellings remain unflagged by default. The conflict fixture produces
`LINT_POLICY_CONFLICT`. A 100-pass test pins finding order; a severity-only
change leaves the semantic artifact and generated outputs byte-identical.

Validation:

| Gate | Result |
| --- | --- |
| R8e2/R8e2b focused tests | Pass |
| `go test ./... -count=1` | Pass, including R7p, R7q Frame, R8a–R8e1, and semantic corpus |
| `go vet ./...` | Pass |
| Root and legacy Zig `zig build test` | Pass |
| BurnIn R7d3 | Pass |
| Standard Normal / Verify | 35 / 35 each |
| DragonGod Normal / Verify | 23 / 23 each |

R8e3 remains trivia preservation and `concept fmt`. R8f follows R8e3; R7q
remains paused. No R8e3 or R8f work was started.
