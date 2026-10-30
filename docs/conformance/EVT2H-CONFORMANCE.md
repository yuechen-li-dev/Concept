# EVT2h — enum conformance

Current EVT2h2 status: **PARTIAL — checked payload LIR, native transport still
unsupported**. No native tagged-value execution is qualified. See the
[EVT2h2 section](#evt2h2--checked-payload-lir-partial) below. EVT2h1 remains
qualified; its historical evidence follows.

## EVT2h0 / EVT2h1 — ordinary native enums

Baseline: `535910e486b0f20f6ec1c6e6e7bee931d359b61a` (EVT2g5),
`concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64. Work starts on
`codex/evt2h1-native-enums`; the original clean checkout was `8b129ad`.

## EVT2h0 semantic inventory (before implementation)

* `EnumDecl` owns module/name nominal identity and ordered `VariantDecl`s.
  `parse.go` assigns zero-based tags; `validate.go` normalizes each tag to its
  declaration ordinal. The grammar has no explicit discriminant initializer or
  underlying-type clause. Attributes do not grant a different enum carrier.
* Both tag-only and payload declarations already exist. `ConstructExpr` carries
  enum/variant names and arguments. Ordinary checking returns the enum type;
  its `ResolvedType` field is not populated for ordinary constructors (that field
  is used by special fallible forms). LIR resolves the checked declaration.
  Checking rejects
  unknown variants, wrong arity and payload type mismatches. An enum containing
  any payload is not an ordinary enum, including its payload-free cases.
* Same nominal ordinary enum supports `==` / `!=`. Payload equality rejects
  with `CV4830`; distinct enum types do not acquire equality or assignment from
  matching tags. Ordinary assignment, local copies and source calls are already
  checked. Numeric casts do not provide an enum-construction escape hatch.
* `layout.go:evt1TypeGeometryWithin` gives tag-only enums size/alignment 4/4;
  payload enums reject fixed scalar geometry (`CV4573`). This is checked enum
  geometry, not an inference from an integer's width or a variant's name.
* C11 emission uses enum-specific constructors and tag equality. Existing
  `enum_equality_c11_test.go`, foundation payload/match fixtures and native ABI
  negative tests exercise these existing semantics. Source C bootstrap support
  for payloads and `match` is not Cathedral native support.
* C ABI checking rejects ordinary enums without an explicit fixed C
  representation (`EXTERN_C_ABI_TYPE_INVALID`); `repr(C)` is record-only.
  This milestone must preserve that rule, even though internal native enum
  calls can use a qualified four-byte discriminant carrier.
* MIR retains enum declarations, tags and semantic bodies. Baseline ordinary
  LIR type lowering stops at `EVT2_UNSUPPORTED_TYPE`; its automata step-result
  pseudo-type is a separate qualified contract, not ordinary enum support.
* CMIRAMD3 transports selected integer operations, CMIRAMD4 virtual scalar
  representations, CMIRAMD5 checked object types/layouts/values and CMIRAMD6
  checked reference authority over the frozen object contract. None carries an
  ordinary enum domain. A scalar width or `i32` tag cannot establish enum
  nominal identity, its variants or legal discriminants.
* Cathedral already owns scalar selection/allocation, typed local storage,
  frame realization, positional Win64 GPR/XMM calls, stack arguments, RAX
  results, fixups and encoding. EVT2g5 adds direct/indirect aggregate carriers
  and hidden results. Go produces checked semantic inputs and transports them;
  these physical mechanisms must remain Concept-authored.

## Transport decision

Introduce CMIRAMD7 as a versioned envelope around the unchanged CMIRAMD5
logical object records, plus an ordinary-enum domain table. Each domain links
an explicit object type ID and nominal/layout identity to checked 4/4 geometry
and ordered names/discriminants. The old object scalar kind denotes the
physical carrier only inside the v7 admission contract. Nominal value/type IDs
remain distinct throughout admission; Cathedral validates the domain before
reusing scalar realization. No old artifact gains enum meaning.

CMIRAMD3–6 schemas, tags, generated codecs, hashes and specimens remain frozen.
Payloads, `match`, built-in fallible types, enum-bearing aggregate/reference ABI,
external enum declarations and new ownership semantics are outside EVT2h1.

## EVT2h1 implementation and native evidence

Inventory commit: `2292af7`. Implementation and executable tests:
`1fc1503c12ee5d7899affe75d837e75ba234b282`.
Imported dependency preservation and its source-free semantic-artifact test:
`191aacf`.
Frozen generated-output LF pinning: `0b8abf9`; this changes checkout policy only,
with no CMIRAMD3–6 schema, codec, metadata or specimen content changes.

`lir_enum.go` snapshots checked nominal identity, declaration-order variants and
existing enum geometry. Values use distinct `enum:<digest>` LIR types. The LIR
verifier admits only same-domain equality/inequality, typed local operations,
returns and checked direct calls; ordinary arithmetic and ordering remain
unavailable. `ConstructExpr` lowering looks up checked declaration tags, rather
than deriving discriminants from variant spelling. Assignments/copies use the
existing typed slot/load/store path.

`BridgeSchema7.concept` owns CMIRAMD7 v7. Schema SHA256:
`74992e3d74be0581484be7f3c6341b7e75654d59b76255f075771367858d33c2`.
Its 44-byte header uses the existing magic/version/schema-hash convention. The
payload is a frozen `WireObjectModule` followed by enum domains linking explicit
type IDs, nominal and layout identities, checked size/alignment and named tags.
Generated Go/Concept codecs reuse the frozen v5 record codecs. Generate with
`go run ./cmd/machinebridgegen .`; the focused test compares all three generated
files and performs 100 independent codec generations.

`BridgeValidate7` independently admits artifact bytes. It checks domains,
variant uniqueness/ordinals, representation, scalar-only values, literal tag
membership and permitted enum operations. The shared object verifier checks
nominal type IDs at every slot, operand, parameter, result, call and CFG edge.
Only v7 supplies its admitted enum alias set to the common type checker; v5/v6
always supply an empty set. A frozen v5 envelope containing enum aliases rejects.
Layout digests are consistency identities, not signatures or producer
authentication; admission does not reconstruct source declarations from a hash.

Cathedral chooses the admitted U32 discriminant carrier, projects it through
`AMD64.Objects`, and reuses scalar selection/allocation, call preservation,
Win64 positional carriers, frame finalization, encoding and rel32 resolution.
No enum allocator, frame planner, production Go instruction selector, register
placement or byte encoder was added. Go's changes are checked semantic
production/consistency checking, schema-driven transport and bootstrap hosting.

### Executed specimen

`language/evt2h1/valid/enums.concept` has 17 functions and three distinct nominal
domains (`Phase`, `Color`, `Only`), with 3/3/1 variants and identical 4/4 geometry.
Every variant is constructed and executed. Coverage includes local memory,
assignment and independent copies, equality/inequality, early returns, branches,
range loops, forwarding, recursive calls, values live across calls, eight enum
arguments and mixed int/enum/double/other-enum/float/bool positional arguments.

`TestEVT2h1NativeExecution` first runs using only artifact bytes and the compiled
Concept backend, without the domain source/header/oracle. It then compiles the
checked source as a separate strict C11 semantic oracle. One hundred varied
inputs compare the real native Scenario path with the C11 source result in
Normal and Verify. Both modes execute all constructors and direct parameter/
return probes. C callers use independently authored uint32 carrier signatures;
native callers also execute an independently compiled C callback through a
test-only tail thunk at an already resolved internal identity function. The
unmodified master image is used for byte determinism. This adds no production
external linker or source-level extern enum ABI.

One hundred fresh parses/checks/LIR/MachineIR/debug generations agree, and each
mode compares 100 fresh native emissions, symbols and fixup counts. Native
Normal/Verify bytes agree. Tests pin both artifact and native SHA256 values:

| Specimen | Size | SHA256 |
| --- | --- | --- |
| CMIRAMD7 ordinary enum module | 46377 bytes | `88032d2679c4730efcd7dc91130c21ebf6032bdc9c8e55d2d19d9a8798e5d9d3` |
| Native image, 17 functions / 16 calls | 8440 bytes | `76425ddc81825b0897c1efdeb515e502ca416e76b8090dc6c37c7a7b97faaf77` |

Focused local generation/codec checks took 2.154 seconds; combined native
admission/planning/encoding observations were 38–39 ms for the final specimen.
These are measured combined costs, not an isolated enum benchmark or a
performance comparison. Full gate measurements are recorded in the convergence
record after completion.
Actual CLI frame plans span 40–200 bytes. Pressure uses 136 bytes with a
four-byte enum local; Scenario uses 200 bytes with 52 bytes of source locals.
The independent C compiler selected by the harness was MSYS2 UCRT64 GCC 15.2.0,
target `x86_64-w64-mingw32`, using strict `-std=c11 -pedantic-errors -O2`
harnesses and the Windows x64 calling convention.

### Invalid inputs and bounded forms

Twelve source fixtures pin existing diagnostics: duplicate variants (`CV4100`),
unknown constructors (`CV4103`), unit constructor arity (`CV4104`), explicit
discriminant syntax (`CV4004`), nominal assignment/calls (`CV4107`), nominal
equality/ordering (`CV4028`), payload equality (`CV4830`), numeric/enum conversion
(`NUMERIC_CAST_REQUIRED`) and extern enum ABI (`EXTERN_C_ABI_TYPE_INVALID`).

Twenty-three malformed artifacts reject 100 times per mode before changing
either output bytes or NativeImage. Mutations cover absent/duplicate domains,
unknown type IDs, geometry/carrier/identity/layout, empty/duplicate variants,
invalid and noncanonical discriminants, ordering, nominal calls/results, stale
value layout, header/version/hash, truncation/trailing data and v5 alias misuse.
The nine-argument source is semantically valid and artifact-admitted; native
planning rejects with existing `AMD64_SCALAR_PLANNING_FAILURE`, preserving both
output sentinels. Eight physical arguments remain the qualified limit.

CMIRAMD7 is a scalar-and-ordinary-enum module contract. Coexistence with any
lowered aggregate, array, reference capability or generated automata frame is
deferred, including otherwise
unrelated aggregate/reference functions in the same native module. Source C
bootstrap accepts the saved payload/aggregate/reference boundary specimens;
native LIR rejects them explicitly. Payload enums reject even when selecting a
payload-free case of that declaration. Nonempty domains are bounded to 128
types / 256 variants per type; this specimen executes domain sizes one and
three. Existing object limits (256 types, 128 functions/blocks, 512 values,
64 slots and 1024 instructions per function) and native frame/call capacities
still apply. Foreign raw-carrier probes supply legal tags; they do not grant
source casts or validate arbitrary hostile machine-code callers at runtime.

Payload construction/copy/destruction, `match`, Option/Result/Verdict native
support, new ownership/Drop, optimization passes and object-file output remain
outside EVT2h1. Ordinary source C bootstrap support for richer forms is separate
from this native qualification. EVT2h2 requires separate review.

**SUCCESS — EVT2h0 inventory and EVT2h1 native ordinary enum execution qualified.**
The final full Go run at `0b8abf9` passes: compiler package 1394.705 s, CLI
297.737 s, GPU-free Vulkan profile 0.435 s. This includes the earlier native,
corpus, checked-output and Golden regressions. Focused race, vet, Standard and
DragonGod Normal/Verify, package, generated inspection, formatting/lint and diff
checks pass. Frozen Zig tests are skipped because their compiler/build/test
infrastructure is unchanged. See the [convergence ledger](../history/milestones/EVT2H-CONVERGENCE.md)
for the corrected LF-only failure and complete command/outcome record.

## EVT2h2 — checked payload LIR (PARTIAL)

Real baseline: `4e96e5cda4f932231129c057de1c00edbb81e7f8`, fetched `origin/main`,
including the merged EVT2h1 and test-workflow checkpoints. The original clean
local checkout was `8b129ad` on `main`. Work uses
`codex/evt2h2-native-payload-enums`; inventory commit `7e3db57`, checked-LIR and
specimen commit `784b34c`. Qualification-harness correction: `09b3d50` isolates
Windows cast-trap executable paths. Compiler and Windows AMD64 / Microsoft x64 target are
unchanged. See the [starting inventory](../design/EVT2H2-CHECKED-TAGGED-VALUES.md).

This removes the first blocker: `GenerateLIR` now preserves payload enums as
nominal `tagged:<digest>` logical values. `LIRTaggedEnum` records checked storage
geometry, per-variant identities/ordinals, ordered field identities and types,
record/array geometry, readonly fields and checked copy eligibility.
`make_tagged` selects a declaration-order variant and requires every active
field exactly once, in checked field order. Its operands are evaluated left to
right. This complete-value operation is its initialization evidence; there is
no partial payload initialization operation. Debug output includes the variant.

Logical tagged values reuse existing typed local/slot/call checking and the
aggregate definite-initialization analysis. The verifier treats their record
field surface as opaque: generic record construction, field projection and
member replacement cannot authorize a payload read. Unknown domains, malformed
constructors, nominal result mismatches and use of uninitialized slots reject.
Imported defining-module identity/digest survives source-free semantic-artifact
loading. This is frontend consistency verification, not independent Cathedral
artifact admission.

The checked sum layout uses an explicit 4/4 ordinal-tag contract followed by an
aligned union of checked variant records, with a minimum one-byte union storage
member matching the existing C bootstrap. It reuses existing checked record
offsets, scalar geometry and fixed-array geometry. Public payload SizeOf/AlignOf
still rejects with `CV4573`; this private checked layout does not add source
syntax, layout casts, foreign representation permission or a physical frame
plan. Both GCC and Clang independently check the fixture's C sizes, alignments,
union offsets, variant sizes and field offsets under strict C11.

| Checked domain | Size / alignment / payload offset | Fields exercised |
| --- | --- | --- |
| Token | 48 / 8 / 8 | empty, int, double, float, bool, mixed fields, nested Item, int[3], record Stamp |
| Other | 8 / 4 / 4 | empty and int |
| Twin | 8 / 4 / 4 | distinct nominal identity with Other's representation |
| Ordered | 12 / 4 / 4 | two int fields; reference-call evaluation order |

The authoritative program is `tests/runtime/Testing/TaggedValues.concept`:
16 functions, four sum domains, two aggregate layouts. Three facts in
`tests/runtime/tagged_values.concept_test` execute existing checked source
matching, construction, copying, variant replacement, branches, loops, nested
calls, mixed arguments and evaluation order through the C bootstrap in Normal
and Verify. They are part of the quick lane. A separate strict-C11 C probe checks
eight exact floating bit patterns, including signed zero, adjacent precision,
NaN payloads and subnormals, through C-bootstrap constructors in Normal mode.
Those probes do not establish native payload ABI or native semantic parity.

Twenty-two malformed LIR mutations reject, covering domains, variants/tags,
layout/copy permission, field and array metadata, constructor arity/types,
nominal results, unguarded record projection and initialization state. Eight
invalid source specimens pin existing diagnostics; two source/C-valid specimens
pin nested-tagged-LIR and native-match stops. Nested tagged payloads, enum fields
inside ordinary aggregates, reference-bearing payloads, owning/Drop forms, raw
storage and unknown geometry are outside the checked subset. Existing source
copy and authority checks remain in force.
Checked-LIR guards bound sum domains to 128, variants to 256 per domain,
payload fields to 64 per variant and total sum storage to 1 MiB. Existing
record-graph and fixed-array bounds also apply. These are frontend guards,
not measured native capacities; no tagged frame or ABI capacity is qualified.

One hundred fresh parse/check/LIR/debug generations agree: canonical LIR JSON
55,712 bytes, SHA256
`4e345711e1b1d0f75ad810b8d1fe7673fb7f775e61625b370b0204a4ca90ef06`.
Measured generation time was 606.767 ms. This is an LIR snapshot hash, not a
MachineIR artifact or native-byte hash. Focused checks passed in 1.831 s;
focused race passed in 5.727 s; quick passed in 20.48 s.
The post-qualification quick run at `09b3d50` passed in 19.53 s.

### Exact remaining native boundary

`LowerLirToAmd64Machine` stops with `EVT2H2_TAGGED_TRANSPORT_REQUIRED` before
projection. A CLI batch test removes the C toolchain from PATH and confirms the
same diagnostic before Cathedral bootstrap. No tagged MachineIR is published.
CMIRAMD3–7 schemas, generated codecs and byte contracts are unchanged.

CMIRAMD8 remains necessary and **has not been introduced**. The frozen v7 domain
only describes ordinary U32 aliases; v5/v6 record copying assumes all fields are
simultaneously live. A correct next implementation must carry active-variant
semantics into Concept-owned artifact admission, extend the existing recursive
copy owner with validated tag dispatch, and integrate those sum facts with the
existing aggregate ABI planner/emitter. No integer/blob erasure, record-field
escape, parallel copy engine, native allocator or production Go native algorithm
was added to route around that missing contract.

Native constructors, active-payload access, tagged copies/assignment, payload
argument/return ABI, artifact-only native execution, malformed tagged-artifact
sentinels and 100 native emissions are **not implemented or qualified**. Native
image size, frame range and planning/emission timings are N/A. Full regression
at `09b3d50` passed (1417.092 s compiler/native, 69.240 s CLI, 0.528 s GPU-free
Vulkan profile), followed by vet. This preserves the old native qualification;
it does not qualify new h2 execution. The earlier Windows trap-executable reuse
failure and its correction are retained in the convergence ledger. Native
`match`, fallible types,
ownership/Drop, object output and optimizations remain deferred. The next work
is completion of EVT2h2, not EVT2h3.
