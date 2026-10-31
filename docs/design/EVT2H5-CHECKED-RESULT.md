# EVT2h5 checked Result inventory

Starting checkout: clean `codex/implement-native-ordinary-enums` at `7daab14`.
Fetched origin/main is integration merge `065317d262c85ce159219561b0709526e7973c1f`;
fast-forwarded before implementation. Its tree equals the policy checkpoint;
no newer implementation is discarded. Actual implementation baseline is
`065317d`. Compiler `concept-evt1-stage0-go`, Windows AMD64 / Microsoft x64.

## Existing language contract

`failure.go` identifies intrinsic Result by name plus exactly two type arguments.
Its checked synthetic enum has Ok(value: T), tag 0, and Error(error: E), tag 1.
For T=void, Ok has no fields or constructor arguments. Constructors use expected
closed types and retain ResolvedType; no new inference/syntax is needed.
Result is must_use; expression statements require handling/propagation or
explicit `discard`. Copyability, move obligations, reference provenance,
immovable and Drop restrictions are existing checker rules.

`validate.go` checks postfix `?` once as an expression returning T. Without a
local try handler it requires a Result-returning function and exact canonical
E equality (CV4542/CV4543); it does not convert arbitrary error enums. Success
types of operand and containing function may differ. `? else expression` is an
existing explicit remap: the expression is checked against the containing
function's E and evaluated only on failure. It has no implicit old-error binding.
Local try/except can intercept exact error types before function return. `!`
unwraps success and calls the panic path on failure; it is not propagation.
Existing match uses exhaustive Ok/Error arms with copied arm-local bindings;
void Ok binds nothing. Source facts/propositions remain checker-owned.

The C bootstrap evaluates the operand into one temporary, branches on tag 1,
performs the checked remap or extracts Error.error, executes lexical cleanup,
and returns the containing function's Error constructor. On success it yields
Ok.value, or `(void)0` for void. `!` calls concept_panic. Native v0 must reject
cleanup-bearing values and local try handling rather than omit those effects.

Important C boundary: emitted Result has canonical tag/union storage **plus**
legacy `is_error` and `error` compatibility fields for the Vulkan actuator seam.
Do not remove those fields or assume built-in Result has the native/C ABI of
a user enum. Native semantic/C11 differential wrappers must compare logical
values, while independent physical probes use explicitly agreed native shapes.

## Existing native boundary and intended reuse

H4 LIR recognizes only Option's intrinsic enum view. Result type/constructors
and matches therefore stop before physical lowering. `FailureExpr` is also
unsupported. H2's shared checked tagged layout and h3's ordinary CFG/pattern
associations already represent both variants, empty Ok, active bindings,
initialized joins and terminal Error returns. Try CMIRAMD8/9 without new fields.
Use the existing structured GenericApplication identity with both ordered
canonical arguments; do not use C names or scalar widths as identity.

Ordinary enum payloads are currently excluded by tagged LIR validation and
Concept's payload-eligibility mask. V9 already carries their canonical enum
domains and validated scalar aliases. Investigate admitting those checked
aliases as fields through the existing geometry/copy paths; V8 has no alias
records and must retain its old behavior. No encoder/allocator/frame/ABI policy
change is planned. Preserve all frozen schemas, codecs, hashes and byte fixtures.

Lower `?` to the existing exhaustive two-arm flow: immutable operand snapshot,
Ok binding/continuation, Error binding/construction/early return. Void success
needs neither a dummy value nor a payload access. Save earlier expression/call
values through ordinary logical slots when later propagation introduces blocks;
nonparameter LIR values are block-local. Do not introduce another copy engine.
Reject unsupported reference liveness, ownership, Drop, nested sums, panic and
try/except paths explicitly. Native remap can use the existing checked expression
on the Error arm; exact-E remains mandatory without that explicit remap.

## Verification plan

Use Quick and focused source/LIR/artifact/native/C11 Normal/Verify evidence.
Use ten fresh Result generation/plan/emission samples for this new milestone;
retain a separate 100-run Result checkpoint gate and every historical gate.
Selecting old native regression families retains their embedded 100 runs.
Run adjacent h2/h3/h4 admission and execution regressions because ordinary enum
payload eligibility touches shared admission. Add independent native-shape ABI
probes; do not claim general C Result interoperability. Expand qualification if
the wire format, encoder, allocator, frame or ABI policy actually changes.
The complete project-wide lane remains reserved for phase closeout, tentatively
EVT2h9, unless the risk assessment requires it earlier.
