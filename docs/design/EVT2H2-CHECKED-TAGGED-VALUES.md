# EVT2h2 checked tagged values: starting inventory

Starting checkout: clean `main`, `8b129ad710e254e3cdc3b647418693cca175e24c`.
After fetching, `origin/main` is
`4e96e5cda4f932231129c057de1c00edbb81e7f8`, merging the test-workflow
lineage through `bc84b51`. Work uses branch `codex/evt2h2-native-payload-enums`
from that merge; no checkout was reset and no unrelated changes were present.
Compiler: `concept-evt1-stage0-go`; target Windows AMD64 / Microsoft x64.

## Existing semantics

`EnumDecl` has nominal module/name identity and ordered `VariantDecl`s. The
checker normalizes discriminants to zero-based ordinals. Each variant owns an
ordered list of named, typed payload fields. Explicit discriminant syntax and
underlying representation clauses are not part of the grammar.

`validateConstructExpr` resolves the nominal enum and variant, checks exact
arity, contextually checks each argument against its field type, and requires
copyability (`CV4102`/`CV4103`/`CV4104`/`CV4106`/`CV4107`/`CV4133`). A
payload-free variant of an enum with any payload is still a value of that sum
type. Ordinary same-type assignment and parameter/return checking apply.
Payload enum equality remains rejected (`CV4830`). No numeric cast creates a
sum value. Source `extern` enum ABI is not qualified.

Payload projection is established by the existing exhaustive pattern checker,
which checks nominal subject identity, variant identity, binding arity and
binding types. Ordinary struct field access does not expose an enum's C union.
Native `match` and fallible wrappers remain outside this mission.

`generate.go:lowerExpr` evaluates constructor arguments in source order into
typed temporaries before invoking the constructor. The C representation is a
struct containing a generated C enum tag and a union of per-variant structs,
plus a one-byte `none` member. Constructors write the tag and selected fields;
inactive fields and padding are not semantic values. C aggregate assignment
does not establish native permission to copy inactive bytes.

`evt1TypeGeometryWithin` deliberately rejects payload-enum SizeOf/AlignOf with
`CV4573`. Payload storage must not acquire public scalar geometry as an
incidental native-lowering change. Existing record field geometry, fixed-array
extents, copyability and Drop queries are reusable checked facts. A private
native sum layout must be distinguished from the public source geometry query
and independently compared with the selected C compiler when parity is claimed.

## Existing native boundary

`lir_enum.go:lowerEnum` rejects every enum declaration containing payloads with
`EVT2_UNSUPPORTED_PAYLOAD_ENUM`, even when selecting a payload-free variant.
The ordinary constructor path only creates scalar constants. Neither native
LIR nor MachineIR currently records active payload initialization or access.

CMIRAMD7 is explicitly an ordinary-enum envelope over frozen v5 object records.
Its domain has names/tags and checked 4/4 U32 carriers, with no payload fields,
sum layout or active-variant authority. Its validators prohibit aggregates and
reference capabilities. CMIRAMD5/6 can describe checked record/array values,
locations and references, but cannot express overlapping alternatives or a
variant-dependent semantic copy. They must not accept a sum as an ordinary
record, integer or byte array.

Cathedral's `CopyObjectFields` recursively transfers meaningful scalar fields
of records/arrays without copying padding. A sum needs tag-controlled dispatch
within that owner; traversing every alternative would read inactive storage.
EVT2g5 already owns aggregate carriers, indirect argument temporaries, hidden
results, frame validation, register allocation and emission. Those owners must
be extended, not replaced by a Go native algorithm or another copy/frame engine.

The next bridge version is CMIRAMD8 if the implementation reaches transport.
It must be Concept-schema-owned and generated, retain nominal variants and
checked payload layouts/initialization, and carry sufficient checked reference
authority for any admitted mixed reference calls. No v3–7 contract is modified
or reinterpreted. No new wire contract is qualified by this inventory.

The initial quick loop passed in 26.90 seconds. Implementation and qualification
status will be recorded in the conformance and convergence ledgers.
