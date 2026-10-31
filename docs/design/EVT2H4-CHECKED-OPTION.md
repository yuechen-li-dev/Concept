# EVT2h4 checked Option inventory and starting state

Actual baseline: clean synchronized main and fetched origin/main at
`408db1720fdf72381069cbc8f5a5feaa6633816a`; no intervening commits.
Compiler `concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64. Historical
detached worktrees remain unchanged. A detached baseline worktree runs the
unchanged h3 tests with opt-in observer instrumentation while main evolves.
The initial Quick lane passes; the full measured baseline is recorded in the
throughput evidence after completion, not inferred from historical 2,267.62 s.

## Existing source semantics

`failure.go` defines intrinsic Option by name and exactly one checked type
argument, independently of ordinary user enum declarations. Its synthetic
checked enum view is Some(value: T), tag 0, and None, tag 1. Constructors need
an expected Option type; the checker records the resolved complete type on
ConstructExpr. Unknown variants, arity/type mismatch and missing context keep
existing CV4103/CV4106/CV4107/CV4540 diagnostics. Bare Some does not infer a
new generic instantiation. Alias/closed record arguments use existing canonical
type resolution and structured generic-application identity.

`validateMatchSubject` already selects the intrinsic enum view. Both variants
are required exactly once; enum wildcard and variant-attached guards are not
existing syntax. Some binds one mutable arm-local value; None binds nothing.
The h3 subjectless when/otherwise guards, lexical scopes, expression types,
single evaluation and ordinary control-flow semantics apply unchanged.
Option equality is not provided. Payload value copying follows the existing
recursive copyability check; owning/Drop/immovable values retain their existing
source restrictions. Nested Option is source-supported, without implying
native admission in this milestone.

Existing optional operations are `?` propagation, `!` escalation and the
checked reference-producing `OptionValue(ref value)` intrinsic. Their existing
C/source support is distinct from this initial native construction/match/copy
subset. Unsupported native operations must diagnose explicitly.

The C bootstrap emits a closed struct with a 32-bit signed tag and union
Some.value. It initializes constructor storage to zero and uses only the active
payload semantically. This is not a source promise of C enum interoperability,
native zeroed padding, niche packing or null-pointer representation. Public
SizeOf/AlignOf on Option currently has no fixed-layout authority; do not widen
that source facility. Source assertions/propositions and failure fact transport
remain authoritative and are not replaced by native tag tests.

## Native seam and transport decision

At baseline LIR type lowering recognizes user enums and copyable record
applications, but not Option. Constructor lowering looks only in env.enums;
match lowering likewise obtains variants from that ordinary-enum table.
Option therefore fails before native selection. Existing checked tagged-union
layout production already consumes checked payload types/struct geometry;
Cathedral independently admits and realizes that representation.

Attempt CMIRAMD9 reuse: project the existing intrinsic declaration into the
same tagged-domain path, with an explicit closed application identity built
using existing generic identity records. Preserve the intrinsic Option
definition, ordered canonical argument, Some/None identities and field type.
Do not key all applications by the bare name Option or by C/display spelling.
The current transport already carries nominal identity, layout, canonical
field types, copy eligibility, operations, scopes/CFG, active-variant authority
and call signatures. A successor is justified only by a demonstrated missing
fact; CMIRAMD3–9 remain frozen.

The initial native payload domain is the already-qualified scalar/record/fixed
array copyable domain. Nested sums/options, ownership, Drop, unqualified
references, raw storage and unknown layouts reject. Existing h3 match regions,
runtime checked field reads, object copier, allocator, frame planner, encoder
and Win64 classifier must be reused. This is an inventory/plan, not execution
qualification.
