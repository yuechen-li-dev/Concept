# EVT2h — enum conformance

Current EVT2h5 status: **MILESTONE VERIFIED — bounded native Result and `?`**.
Historical sections retain the evidence and boundaries at their respective
checkpoints; h5 does not claim full project qualification.

## EVT2h5 — closed native Result and propagation

Actual baseline: `065317d262c85ce159219561b0709526e7973c1f`, fetched integration
main, fast-forwarded from clean `7daab14` before edits. Their trees agree.
Compiler `concept-evt1-stage0-go`; Windows AMD64 / Microsoft x64. Inventory
`c78b310` records the inspected semantics before implementation; bootstrap
sequencing correction `8b064a6` is independently reviewable. See the
[checked inventory](../design/EVT2H5-CHECKED-RESULT.md) and
[focused evidence](../history/milestones/EVT2H-CONVERGENCE.md#evt2h5-native-result-milestone).
Native semantic production, fixtures and external harnesses are `f283a0b`;
phase-test scheduling is separately identifiable in `2ccaf3a`. Final closeout
changes the evidence records only. Focused aggregate/exact-bit Error propagation
coverage and its expanded fixture pins are `ac840ae`.

### Existing source semantics and checked control flow

Intrinsic Result has two ordered canonical type arguments. Ok(value: T) is tag
0; Error(error: E) is tag 1. Result<void,E>::Ok has no field or argument.
Constructors retain the contextually resolved closed type. The existing checker
owns must_use/explicit discard, exact nominal error compatibility, exhaustiveness,
copy eligibility, initialization and ownership restrictions. No new syntax,
equality rule, inference system or implicit error conversion is introduced.

Postfix `?` evaluates its operand once. Its checked LIR uses the existing
immutable match snapshot, exhaustive Ok/Error CFG and arm-local bindings.
Ok initializes the ordinary result slot and continues; Error constructs the
containing function's checked Result::Error and terminates that path with a
return. Void success needs no slot, fabricated value or payload read. The
existing result-join/dominance and source-operation verifiers remain operative.
Without `else`, E must match exactly, even if two enums have identical geometry.
Existing `? else expression` evaluates a checked explicit error remap only on
the Error path. Operand T and containing function T may differ.

Already evaluated binary operands, ordinary call arguments and record constructor
fields survive a later `?` through ordinary typed logical slots. Cathedral
chooses their storage and lifetime. Parameters retain their existing global LIR
identity. Computed references that would need a new cross-propagation lifetime
contract reject explicitly. Native `!` panic and local try/except remain outside
this milestone. Owned/Drop-bearing values reject rather than skip cleanup.

The independent C11 run exposed a bootstrap ordering defect: a later `?` prelude
could return before an earlier call argument or binary operand executed. C
emission now materializes eligible earlier values before that prelude, preserving
the existing source order; the operand is still evaluated once. Existing failure
cleanup/panic/remap tests pass. The emitted C Result retains its legacy is_error
and error compatibility fields used by Vulkan. It is deliberately not treated
as the native physical shape.

### Representation, transport and Cathedral ownership

The producer reuses h4's structured GenericApplication identity with intrinsic
declaration `(module="", name="Result")` and both ordered canonical arguments.
Result<int,ParseError> and Result<int,OtherError> remain distinct types and call
signatures. The same tagged layout, constructor, assignment, match, copy and
return operations serve user enums, Option and Result.

CMIRAMD8 suffices for scalar constructor-only Results. CMIRAMD9 carries matches,
propagation and ordinary enum payload aliases. No new version, fields, schema,
generated codec, hash or historical byte fixture is introduced. Frozen CMIRAMD3–9
remain unchanged. Two bounded admission additions are needed: independently
validated v9 ordinary enum aliases may be tagged fields, and a v9 constructor-only
module with enum and tagged domains may have no pattern flows/operations. An
artifact containing pattern operations still requires complete flow authority.
V8 supplies no enum aliases and keeps its prior eligibility boundary.

Cathedral's Concept validators independently check type/layout/variant identity,
geometry, source correspondence, scopes, initialized joins and active payload
authority. The existing logical copier dispatches on the active variant and
recurses only through meaningful fields. Storage, direct/indirect arguments,
hidden results, scalar encoding, allocator and unified frame planner are reused
without policy changes. No production Go Result layout/copy/ABI/encoding engine
or Result-specific physical implementation is added.

| Executed closed applications | Size / alignment | Existing Win64 route |
| --- | --- | --- |
| Result<int,ParseError>, Result<int,OtherError>, Result<void,ParseError>, Result<bool,ParseError> | 8 / 4 | direct eight-byte aggregate |
| Result<float,bool>, Result<bool,uint8> | 8 / 4 | direct eight-byte aggregate |
| Result<Pair,uint16>, Pair = two-int copyable struct | 12 / 4 | indirect argument / hidden result |
| Result<double,ParseError>, Result<int,double> | 16 / 8 | indirect argument / hidden result |
| Result<int<array>[3],ParseError>, Result<int,int<array>[3]> | 16 / 4 | indirect argument / hidden result |
| Result<Item,ParseError>, Result<int,Item>, Result<void,Item>; Item = { int; double; } record | 24 / 8 | indirect argument / hidden result |

The ABI probes use explicitly agreed private shapes, not generated C Result
structs. They establish the tested Win64 carrier paths; general foreign Result
interoperability is not claimed. The existing eight physical argument positions
include the hidden result pointer. Seven explicit mixed arguments exercise stack
positions; eight explicit arguments plus a hidden result reject with
`AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED` and intact code/image sentinels.

### Focused execution and rejection evidence

`tests/runtime/Testing/Results.concept` owns 61 functions and fourteen applications.
Two `.concept_test` facts execute source semantics through C in Normal/Verify;
they do not count as native proof. External hosts first execute the artifact
without domain C linked, then compile the same source with strict C11
`-pedantic-errors -O2` and compare values, effects and ordering.

Native cases cover both variants, scalar/record/struct/array successes and errors,
copies and isolated bindings, variant replacement, void propagation, explicit
matches, loops, early-return ordering, conditional results, differing success
types, explicit error remapping, recursion, mixed calls, stack arguments and six
floating spills. Exact binary32/64 checks include negative zero, adjacent finite
values, quiet NaN payloads and subnormals, including Error payloads rethrown by
`?`. Record and fixed-array errors also propagate, including void success with
a record error. MSVC and clang-cl DLLs call native
constructors/propagation and mixed signatures, then relay direct and indirect
values back to native consumers, twenty trials per shape per host.

Protected-page probes put void Ok immediately before inaccessible Error storage;
Ok executes without touching it. An invalid tag must raise the existing native
UD2 inside the emitted image, rather than authorize a payload read. Trap children
use their completed host executable and never overwrite a running Windows binary.
Constructor-only v8/v9 artifacts also execute without domain source.

Fourteen source-invalid specimens pin constructor/type/arity, nominal identity,
must_use, void binding, exhaustiveness and propagation/remap diagnostics. Six
source-valid native-unsupported specimens pin nested sums, ownership/Drop,
references, `!`, try/except and computed-reference liveness. Seven malformed LIR
mutations reject. Twenty-seven malformed artifacts reject independently in Go
and Concept Normal/Verify, preserving round-trip, code and image sentinels.
They cover identity/type swaps, invalid tags/arity, absent variant authority,
incorrect early-return edges, uninitialized/nondominating bindings, fabricated
void fields, scope/copy/geometry/CFG errors, capacity and corrupt wire records.
Concept diagnostics include CMIRAMD9_PATTERN_DOMAIN, PAYLOAD_BINDING,
VARIANT_OPERATION, OBJECT_CONTRACT, CAPACITY_EXCEEDED, VERSION_MISMATCH,
SCHEMA_HASH_MISMATCH and MALFORMED under their existing CMIRAMD9 prefix.

The pinned artifact is **261,777 bytes**, SHA256
`df95976a73a53afc06d280ebc13b31d00aeb47086aa6dd9274947bb938fdad2d`.
The identical Normal/Verify native image is **107,444 bytes**, 61 functions,
39 call fixups, frames **40..488 bytes**, six spills; SHA256
`34c260bb98b31437371fcc19df53a8f192dd9da870bcd795dc27521ddbe5c464`.
Ten fresh source/LIR/CFG/transport/debug generations and ten complete native
planning/emission samples per mode are the ordinary milestone gate. The phase
Qualification lane promotes the new gate to 100; it was not run for h5. Earlier
selected regression families keep their existing 100-run gates.

### Remaining boundaries

No Result-specific runtime exceptions, native `!`, local try/except, ownership,
Drop, move-only or reference-bearing payloads, nested sums, fallible built-ins
beyond Result/Option, public layout query, general foreign representation,
new object output or optimization pass is introduced. Computed references across
a later `?` reject explicitly. The qualified expression contexts are the tested
ordinary calls, binary operands, record construction and existing match/if/loop
paths; arbitrary record-with/index/short-circuit propagation is not claimed.
Standalone fixed-array ABI limitations remain unchanged; arrays inside admitted
Result payloads execute through the existing aggregate route. Verdict and EVT2h6
remain unimplemented.

## EVT2h4 — closed native Option applications

Actual baseline: clean main and fetched origin/main
`408db1720fdf72381069cbc8f5a5feaa6633816a`, with no intervening commits.
Compiler `concept-evt1-stage0-go`; Windows AMD64 / Microsoft x64.
Inventory `609494d`; native checked production/fixtures `10be4cb`.
Harness-only profiling/reuse changes are independently reviewable in
`679bc86`, `3c5b6eb` and `5ba39d8`. No remote branches were changed.

### Source and checked representation

The [inventory](../design/EVT2H4-CHECKED-OPTION.md) records the existing intrinsic
definition: `Option<T>`, Some(value: T) tag **0**, None tag **1**. Constructors
require an expected closed type; Stage-0 preserves that complete type on the
checked constructor. Exhaustive Some/None matching, arm-local mutable copied
bindings, subjectless ordered guards, copyability, initialization and trusted
semantic propositions retain their existing checker authority. Equality and
public SizeOf/AlignOf are not added. Variant-attached guards and enum wildcards
remain invalid source forms, as in h3.

Closed applications use the existing structured GenericApplication identity:
intrinsic declaration `(module="", name="Option")` and one ordered canonical
type argument, including nested generic application identity where applicable.
Display names and ordinary integer width are not identity. Same-sized
Option<int>, Option<float>, Option<bool> and narrow integer applications remain
distinct throughout layouts, constructors, projections and call signatures.

Only semantic production changes in Go: lower the checked intrinsic enum view
through the existing tagged layout/constructor/match paths. No production Go
Option storage, copy, allocation, ABI, instruction selection or encoding engine
is introduced. Cathedral's existing Concept implementation admits and realizes
these values without an Option-specific physical path.

### Frozen transport and native realization

No successor schema is needed. Constructor-only applications use CMIRAMD8;
applications with matches use CMIRAMD9. The full fixture's canonical artifact
is **117,108 bytes**, SHA256
`410976620e998aac75eb2e79e1cc0949c01cb05f595516bb3f4fc6b7c1794f88`.
CMIRAMD3–9 schema, generated Go/Concept codecs, metadata and historical byte
fixtures are unchanged. V9 schema SHA256 remains
`77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c`.

Canonical object type/domain/variant/field records retain nominal application,
checked geometry, copy permission and initialization. Existing v9 test/require
sites, immutable scrutinee slots, ordered CFG edges, binding scopes and result
joins govern active access. Concept admission independently verifies these
relationships from artifacts; no domain source or production runtime registry
is needed. A forged numeric tag cannot establish payload access authority.

Construction, variant replacement and copies reuse h2's tag dispatch and
recursive fieldwise logical copier. Only live fields are semantic values;
inactive bytes and padding are not copied as payload. Storage uses ordinary
object slots and the unified frame planner, with no hidden allocation. The
same scalar encoder/allocation path realizes the selected live fields.

| Executed closed application | Size / alignment | Win64 carrier |
| --- | --- | --- |
| Option<int>, Option<float>, Option<bool>, Option<uint8>, Option<uint16> | 8 / 4 | direct eight-byte aggregate |
| Option<double> | 16 / 8 | indirect argument / hidden result |
| Option<Item>, Item = C-represented copyable record { int; double; } | 24 / 8 | indirect argument / hidden result |
| Option<int<array>[3]> | 16 / 4 | indirect argument / hidden result |

The checked private tagged storage policy is shared with h2; it does not grant
source-level C enum or C++ variant interoperability. Independent C probes use
explicitly agreed struct shapes. No niche or null-pointer packing is claimed.
Eight physical arguments, including the hidden result pointer, remain the ABI
limit. The source-valid eight-explicit-argument hidden-return specimen transports
successfully but Cathedral preflight rejects it with
`AMD64_AGGREGATE_ABI_CAPACITY_EXCEEDED`, preserving code/image sentinels.

### Independent execution and negative evidence

`tests/runtime/Testing/Options.concept` is the authoritative 34-function fixture.
Two `.concept_test` facts exercise existing source semantics through C in both
modes; they are not native evidence. External native hosts first execute the
artifact without the domain C oracle linked, then independently compile and
execute the same checked source with strict C11 (`-pedantic-errors -O2`).
MSVC and clang-cl DLLs independently call native constructors and mixed calls,
and relay direct/indirect values back to native consumers, 100 times per shape.

Native coverage includes None/Some, all eight applications, eligible local
copies and isolated array/record bindings, same/different variant replacement,
branches, loops, early returns, recursion, nested matches, Option-valued result
joins, single scrutinee evaluation, ordered side effects/guards, seven explicit
mixed arguments plus hidden result, fifth/later stack positions and live XMM
spills. Binary32/64 probes preserve negative zero, an adjacent finite value,
quiet-NaN payload bits and the least subnormal. A None value at a protected
page boundary proves that inactive payload storage is not read. A forged tag
traps inside native code in a separate finished executable path.

Six malformed checked LIR cases reject closed identity/layout, inactive access,
wrong field type, missing pattern authority and copy permission. Nine invalid
source specimens retain CV4106/CV4540/CV4028/CV4114/CV4573/CV4115/CV4116/CV4107/
CV4103. Six source-valid native boundaries reject nested sums, owning payloads,
reference payloads and `?`, `!`, OptionValue with precise existing h2 or explicit
`EVT2H4_UNSUPPORTED_OPTION_OPERATOR` diagnostics.

Twenty-nine malformed v9 artifacts independently reject in Go and Concept:
closed-type/call/result mismatches, inactive None access, constructor arity/
initialization, variants/tags/identities, geometry/types/copy representation,
missing selection/correspondence/bindings, scope/CFG/result/provenance corruption,
capacity, version/hash, truncation and trailing data. Normal and Verify retain
round-trip, code and image sentinels on every rejection. Diagnostics include
CMIRAMD9_PATTERN_DOMAIN, VARIANT_OPERATION, OBJECT_CONTRACT, PAYLOAD_BINDING, RESULT_JOIN,
PATTERN_CFG, CAPACITY_EXCEEDED, VERSION_MISMATCH, SCHEMA_HASH_MISMATCH and MALFORMED.

The native image is **45,901 bytes**, 34 symbols / 22 resolved calls, frames
**40..408 bytes**, with **four floating spills**. Both modes pin SHA256
`5d5b8746bd74e74b50d40f8cc7dbe5e680597c6c00ba732d33be33fa8bc90316`.
Qualification performs 100 fresh source/LIR/CFG/MachineIR/debug generations,
100 exact valid Concept round trips per mode, 100 complete physical plans per
function and 100 full fresh emissions per mode, comparing code, symbols and
fixups. Oracle executions are separate from those fresh native gates.

In the final full run at `10be4cb`, native emission samples are 318 ms in each
mode, including admission/preflight; 100 complete physical-plan sets are
3.559 s Normal / 3.677 s Verify. Fresh source/CFG/artifact generation and codecs
take 1.550 s for 100 runs. Native Option execution is 84.84 s for both modes;
independent artifact/sentinel admission is 22.88 s. Canonical formatting/LF
follow-up `a5588b9` changes only negative/ABI-boundary specimens and is rechecked
with focused diagnostics and race admission. The authoritative executed positive
source and all pinned artifact/native bytes remain unchanged.

Full-run measurements and gate outcomes are recorded in the
[convergence record](../history/milestones/EVT2H-CONVERGENCE.md#evt2h4-native-option-qualification)
and [throughput evidence](../tooling/EVT2H4-QUALIFICATION-THROUGHPUT.md).
Full uncached Go/native qualification and vet pass: 1,319.57 s, 935 passed
families; compiler/native 1,310.377 s, CLI 143.971 s, GPU-free Vulkan 0.443 s.
Focused race passes (481.521 s package / 494.193 s wall), including both native
modes, malformed admission, include-closure invalidation, Concept facts and
workflow policy. Standard/DragonGod pass Normal/Verify. Eight legacy C11 tests
skipped for missing MSVC shell includes in the full lane are executed and pass
in the developer environment. The remaining two ordinary-run skips are the
unrelated opt-in collector burn-in and the subprocess-only worker parent role.
Frozen Zig suites are skipped under the unchanged-infrastructure policy.
Nested Option/sums, owning/move-only/Drop/reference-bearing payloads, raw storage,
unknown layouts, general foreign ABI, native `?`, `!`, OptionValue and wider
integer domains without specific Option execution evidence remain unqualified.
Result/Verdict, ownership/Drop, object output and optimizations are deferred.
Stop at EVT2h4; a subsequent mission requires separate review.

## EVT2h3 — native checked match

Actual baseline: clean main and freshly fetched origin/main at
`745b1ec35679743bed555a89e19611c6dea4712a`. Compiler
`concept-evt1-stage0-go`; Windows AMD64 / Microsoft x64. Inventory `ee80418`;
checked CFG/contract/admission `6aa55c2`. Historical detached worktrees are
preserved. The [inventory](../design/EVT2H3-CHECKED-MATCH.md) records actual
source semantics before implementation.

Native implementation `9f50f82`, arm-dominance correction `2b93c3c` and
conditional nesting qualification `1737006` complete the scoped changes.

Bootstrap determinism follow-up `1058c86` reserves the compiler's closed Gob
semantic type graph before encoding any AST. Previously an identical typed
dependency acquired different private wire type IDs after earlier AST forms
were encoded, changing its content digest. V9 correctly preserves that digest;
the correction stabilizes its input rather than dropping provenance. The v2
self-describing payload remains readable; no native algorithm or CMIRAMD
schema/codec changes. Cold/warm 100-subprocess checks retain this regression.

Final full qualification at `1058c86` passes: compiler/native 2,264.969 s,
CLI 154.470 s, GPU-free Vulkan 0.780 s; complete lane 2,267.62 s including
`go vet ./...`. Focused race, strict C11/native Normal/Verify, 100-run gates,
Standard/DragonGod, package, generation, format/lint and diff checks pass.
Final closeout changes these records and the h3 LF fixture rule only. Frozen
Zig suites are skipped because their compiler/build/test paths are unchanged.

### Existing semantics and native subset

Enum matches list every variant exactly once. Payload patterns bind every
field, in declaration order, as mutable arm-local copied values. There is no
reference-pattern syntax. Enum wildcards reject CV4109; integer matches
require a final wildcard. Guards use the existing subjectless
`match { when condition => ...; otherwise => ... }`, parsed into ordered
if chains. Guards attached to variants are not existing syntax. Native
fixtures nest guards inside selected enum arms, preventing irrelevant guards
from executing. Expression arms require one expression; statement arms require
braced blocks and permit early returns.

Qualified payload fields retain EVT2h2 integer/bool, binary32/binary64,
copyable nested record/struct, fixed-array and mixed empty/payload forms.
Results include integer/floating values, Item records and Token values through
existing aggregate ABI rules. Authoritative modules:
`Testing.OrdinaryMatches` (seven functions) and `Testing.TaggedMatches`
(35 functions, including imported h2 helpers). Two facts in
`tests/runtime/native_match.concept_test` exercise behavior through Concept's
C bootstrap; separate native hosts execute the same checked source modules.
The language facts alone are not native evidence.

### Checked control flow and transport

LIR snapshots the scrutinee once in a readonly typed slot. Ordered dispatch
blocks test that snapshot through nominal `test_variant` operations. The last
enum arm uses `require_variant`, trapping an invalid tag. Integer matches use
existing literal equality/branches. Selected bodies project checked fields into
ordered binding slots. Expression results join through typed slots initialized
on every reachable normal completion path; all-return statements need no join.
Nested matches and guarded if expressions use ordinary blocks/branches/slots.

CMIRAMD9 schema SHA256:
`77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c`.
The authoritative Concept schema generates both codecs and metadata. V9 reuses
v8 tagged domains/operations and v7 ordinary domains, adding typed test/require
sites and subject/result/join/arm/binding associations with canonical object
CFG, operands and lexical scopes. Geometry, field types, offsets and source
sites remain in their existing records. CMIRAMD3–8 schemas/hashes/codecs remain
byte-for-byte frozen.

Independent `BridgeValidate9` checks nominal domains/operations, exact
exhaustive mappings, unique dispatch/body predecessors, ordered false edges,
immutable single initialization of subjects, binding projection prefixes,
arm-local regions and body dominance, lexical scopes and result-join correspondence. Existing
object/borrow admission supplies SSA/dominance, lifetime and all-slot CFG
must-initialization. Go independently reconstructs checked LIR for consistency;
it does not select or encode native code. Every payload projection retains
the v8 runtime tag check against the same private snapshot. An offset or
forged numeric tag cannot authorize arbitrary memory access.

### Cathedral and Win64

Concept AMD64 selects compare/SETZ/branch and compare/UD2 using existing scalar
allocation and scratch preservation. Tagged storage, active-field recursive
logical copying, argument transfers, unified frames, aggregate ABI and encoder
reuse EVT2g/h2. There is no physical MATCH instruction, alternate layout/copy
engine, allocator, frame planner, heap allocation or Go native match algorithm.
V9 emission uses existing count-only preflight before publishing code/image.
The CLI admits v9 and displays existing plans; its bounded output buffer is
262,144 bytes for the larger 122,509-byte specimen.

Win64 coverage includes scalar enum carriers, indirect Token parameters,
hidden Token/Item results, mixed reference/integer/floating calls, nested
calls, seven floating arguments/stack positions, eleven spills and live
values across calls/branches. This is not general foreign enum or C++ variant
interoperability. EVT2h2 independent MSVC/clang-cl probes remain regressions.

### Limits and diagnostics

V9 admission caps: 128 functions, 256 types, 128 ordinary and 128 tagged domains,
128 blocks/scopes per function, 512 values, 64 slots and 64 match flows per
function (8,192 module flows). Object admission permits 1,024 instruction
sites; native planning retains its tighter qualified capacities. Existing
frame/image bounds apply. Capacity failures reject before output mutation.

Eighteen source negatives preserve CV4020/CV4024, CV4109–CV4118, CV4186 and
guarded-match parser diagnostics. Native authority errors include
`EVT2H3_UNSUPPORTED_MATCH_AUTHORITY`, `EVT2H3_UNSUPPORTED_MACHINE_MATCH`,
`EVT2H3_UNSUPPORTED_RESULT` and existing native-subset errors. V9 diagnostics
use prefix `CMIRAMD9_` with MALFORMED, VERSION_MISMATCH,
SCHEMA_HASH_MISMATCH, CAPACITY_EXCEEDED, PATTERN_DOMAIN, VARIANT_OPERATION,
PATTERN_CFG, PAYLOAD_BINDING, BINDING_SCOPE, RESULT_JOIN and OBJECT_CONTRACT.

Reference/move-only/Drop-bearing subjects, escaping references, nested tagged
payload fields, fallible built-ins and automaton matching remain outside the
subset. Standalone fixed-array ABI results remain unsupported; array bindings
use qualified indexing and record results. Only existing bounded range loops
and qualified native statements are claimed. No new patterns, layout, generic
wrappers, ownership/Drop, object files, jump tables or optimizations are added.

Exact execution, hashes, timings and gates are recorded in
[convergence evidence](../history/milestones/EVT2H-CONVERGENCE.md#evt2h3-native-match-qualification).


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

Historical entry at `09b3d50`, superseded by the native closeout below.

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

## EVT2h2 — native tagged values

Actual baseline: clean `main` at `934e5b4859b201354331b1eca0b2fbe286af47fd`,
equal to fetched origin/main. Historical checkpoints were preserved; no task
branch was revived or reset. Compiler `concept-evt1-stage0-go`, Windows AMD64 /
Microsoft x64. Contract/admission commit: `8698781`. Native realization:
`0da865b`. Completed spill/negative qualification specimen: `4a175e6`.
Local-only module preflight correction and imported-transport follow-up: `59788a4`.
The three [internal checkpoints](../history/milestones/EVT2H2-CHECKPOINTS.md)
record the implementation progression and completed qualification.

The existing source contract remains authoritative: declaration-order tags,
nominal identity, constructor arity/type/order, checked copy eligibility and
pattern-only payload inspection. The source program has 22 functions, four sum
domains and three ordinary record/struct layouts. Token's seven variants cover
empty, integer, double, float, bool, mixed fields, a nested Item, fixed int[3]
and Stamp record. Other/Twin have identical 8/4 geometry and distinct identities;
Ordered has 12/4 geometry and two evaluated integer fields. Token remains 48/8.
The private semantic layout is still the explicit 4/4 tag plus aligned union of
checked variant records. Public payload SizeOf/AlignOf and extern enum ABI still
reject. No native layout is inferred from integer width or variant spelling.

CMIRAMD8 is owned by `Standard.Backend.BridgeSchema8`; its generated Go and
Concept codecs and JSON schema have SHA256
`8dc34f916ab8293e838db026d720825f10d2a7c12006c19924a0fffea94c0711`.
It embeds frozen v6 object/reference records and adds nominal sum domains with
ordered variants/field geometry plus sparse canonical operation sites. Existing
type/value/operand/provenance/copy metadata is reused. Sum roots are opaque in
the ordinary field table, so record projections cannot expose inactive bytes.
The defining semantic dependency identity/digest survives imported transport.
CMIRAMD3–7 schemas, generated codecs and byte contracts are unchanged.

`BridgeValidate8` independently validates artifact-only geometry, nominal and
variant identities, initialization, field order/type/layout, copy eligibility,
operation authority and existing object/reference/CFG rules. This is consistency
validation, not producer authentication. Construct operations bind every active
payload operand; the constructor establishes the tag after writing fields.
Internal `checked_payload` binds a logical snapshot, variant and field. Its
native selector compares that snapshot's tag and traps before a mismatched
read. There is no separate Boolean authority that an artifact can forge, no
untyped address as projection subject, and no payload address result. Seven
verified internal observer wrappers exercise scalar, record and array reads
without introducing source projection syntax or native match.

Cathedral extends the existing `AMD64.Objects.CopyObjectFields` recursion with
admitted tag dispatch. It reads/writes the tag and copies only active meaningful
fields through the same scalar/record/array copier. Inactive storage and padding
are not semantic data. Locals, snapshots, source/call temporaries and hidden
results use the existing frame planner and object placement. Allocation, scalar
encoding, transfers, calls and fixups remain Concept-authored. Go produces
checked semantics and generated transport, orchestrates tests and supplies the
independent C oracle; there is no production Go tagged native algorithm.

The existing Win64 aggregate machinery passes/returns the 8-byte sums directly,
uses indirect arguments for larger sums, and hidden results for 12/48-byte
returns. Hidden pointers shift positional GPR/XMM/stack arguments. Outgoing
indirect temporaries retain the existing 16-byte alignment and value isolation.
Ordinary native records retain their conservative hidden-result rule: the
4-byte Stamp observer uses an explicit result pointer in its independent C
probe. Standalone array returns remain unsupported; the array observer returns
an ordinary ArrayBox record. Neither probe grants C enum, C++ variant or general
foreign-source interoperability.

Native execution covers construction, local/call copies, same/different variant
replacement, original-value isolation, conditional returns, loops, forward and
backward calls, recursion, reference-observable evaluation order, mixed arguments
and fifth/later stack positions. FloatingPressure demonstrably spills nine
floating values using the existing allocator/frame path. Signed zero, adjacent
precision, NaN payloads and subnormals retain exact binary32/binary64 bits through
native constructors, copies and guarded reads. Normal/Verify execute the bytes
in a source-free host before separately linking the strict C11 source oracle.
Separate MSVC and clang-cl DLLs test C-to-native and native-to-C tagged calls.
Earlier EVT2g5 assembly ABI boundary probes remain frozen regression evidence.

Three child-process guard tests require an illegal-instruction trap for inactive
or forged tags. Two put all payload storage behind a protected page: an inactive
payload read would instead cause an access violation and fail the test. The
existing cast-trap harness retains a distinct executable path per case. Tagged
trap children execute a finished binary without relinking an open executable.

28 malformed artifacts reject in both independent admissions and before native
output/NativeImage mutation. Cases cover unknown/duplicate domains and variants,
nominal calls/returns, layout/alignment/overlap/size, copy permission, complete
initialization, missing/duplicate/invalid operation authority, bad projection
variant/field, capacity, header/hash/version, unknown operation and truncation/
trailing bytes. Tiny valid-output capacity and malformed derived argument,
return and overlapping frame plans also reject. Diagnostics include
`CMIRAMD8_{MALFORMED,VERSION_MISMATCH,SCHEMA_HASH_MISMATCH,CAPACITY_EXCEEDED,
TAGGED_DOMAIN,TAGGED_VARIANT,PAYLOAD_GEOMETRY,PAYLOAD_INITIALIZATION,
ACTIVE_VARIANT_AUTHORITY,COPY_ELIGIBILITY,OBJECT_CONTRACT}` and existing
`AMD64_AGGREGATE_ABI_*` errors. Existing eight source diagnostics and 22 malformed
LIR cases remain; neither numeric casts nor payload equality become legal.

`Testing/TaggedLocals.concept` separately exercises construction, copying and
replacement in a module with scalar-only function signatures. CMIRAMD8 always
uses the existing aggregate count-only preflight, including this case; it
cannot select the older object emission path merely because no signature needs
an aggregate carrier. The regression verifies this classification, executes
the module in both modes, and requires a one-byte output failure to preserve
every output byte and the previous NativeImage. CMIRAMD3–7 routing is unchanged.

| Measured artifact / gate | Result |
| --- | --- |
| Canonical CMIRAMD8 | 42,064 bytes; SHA256 `1215b1c779093e9fc149929d8ca4ad0b6f81a1a6aa0d77c56341d5e3372125e5` |
| Native image | 29 functions, 18 resolved calls, 31,871 bytes |
| Pinned native SHA256, both modes | `524c364540dbb5efc8b1634f5d7087f8933fd4225d92afaa73dc39bccd7becc3` |
| Native frame range | 40..1096 bytes |
| 100 fresh domain/LIR/MachineIR/debug generations and schema/codecs | PASS; 668.665 ms |
| 100 complete physical plans per function | PASS; canonical debug descriptions compared; 1.699 s in the recorded Normal run |
| 100 fresh native emissions per host | PASS; exact code, symbols and fixups; Normal/Verify identical |
| Emission | 84 ms in the recorded Normal run, including admission and count-only preflight |
| MSVC / clang-cl ABI and strict C11 differential | PASS in Normal/Verify |
| Full Go/vet at `4a175e6` | PASS; compiler/native 1569.289 s, CLI 74.861 s, GPU-free Vulkan 0.506 s, lane 1572 s |
| Final full Go/vet including local-only preflight at `59788a4` | PASS; compiler/native 1557.665 s, CLI 81.251 s, GPU-free Vulkan 0.480 s; lane 1564.97 s including vet |

Admission retains 256 object types, 128 functions/blocks, 512 values, 64 source
slots and 1024 instructions per function; sum domains are bounded to 128,
variants to 256, fields to 64, arrays to 256 elements, storage to 1 MiB and
operation sites to 131072. Native placement retains 128 scalar virtuals, 512
flattened object instructions, 64 total object slots, 32 calls / 256 aggregate
transfers per function, 128 symbols / 512 call fixups, eight physical arguments
including hidden results, recursive copy depth 16 and frames below 4096 bytes.
Exhausted GPR allocation rejects; existing floating spills are qualified. These
are bounded capacities, not claims that every combination at every maximum fits.

Nested payload sums, sums inside ordinary objects, coexisting ordinary enum
domains (`EVT2H2_UNSUPPORTED_ORDINARY_ENUM_CONTEXT`), reference-bearing/owning/
move-only/Drop payloads, raw reinterpretation, unknown layout and foreign enum
representations remain unsupported. Native match, Option/Result/Verdict,
ownership/Drop, new object output and optimization are deferred. Source C
bootstrap support for these richer forms does not establish native execution.
The next milestone is EVT2h3 only after separate review.

**SUCCESS — EVT2h2 native tagged values qualified.** The full final Go run
includes frozen native regressions, corpus, Golden, package and Vulkan gates.
Focused race, independent strict C11/MSVC/clang-cl, Standard/DragonGod in both
modes, generated inspection, format/lint and diff checks pass. Frozen Zig is
skipped because its compiler/build/test infrastructure is unchanged. Work stops
after EVT2h2; source-only match support is not native match qualification.
