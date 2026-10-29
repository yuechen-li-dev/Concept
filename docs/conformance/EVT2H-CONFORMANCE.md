# EVT2h0 / EVT2h1 — ordinary native enums

Baseline: `535910e486b0f20f6ec1c6e6e7bee931d359b61a` (EVT2g5),
`concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64. Work starts on
`codex/evt2h1-native-enums`; the original clean checkout was `8b129ad`.

## EVT2h0 semantic inventory (before implementation)

* `EnumDecl` owns module/name nominal identity and ordered `VariantDecl`s.
  `parse.go` assigns zero-based tags; `validate.go` normalizes each tag to its
  declaration ordinal. The grammar has no explicit discriminant initializer or
  underlying-type clause. Attributes do not grant a different enum carrier.
* Both tag-only and payload declarations already exist. `ConstructExpr` carries
  the resolved enum type, variant name and checked arguments. Checking rejects
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

Qualification status: inventory complete; native ordinary enums not yet qualified.
