# R9a bounded bridge convergence path

CMIRAMD1 remains the live wire format. The manual producer is
`internal/concept/machineir_bridge.go`: scalar/tag/text primitives and
`machineBridgeWriter.operand`, followed by EncodeMachineBridge's nested function,
argument, virtual-register, slot, block and instruction sequences. The manual
consumer is `libraries/Standard/Backend/AMD64.concept`: ReadOperand and
DecodeFunctionBody. Field order still exists in both implementations. R9a has not
silently converted inspection text into a schema or introduced a third language.

## Selected source and implementation boundary

Use ordinary Concept record and enum declarations in a dedicated
Standard.Backend.BridgeSchema module as the wire-model source. Move the existing
BridgeText representation (offset/length), operand/tag enums and each wire record
there. Keep backend MachineOperand/BackendFunction as semantic projections rather
than conflating wire data with backend scratch layout. Variable sequences are
represented explicitly by counts and borrowed views; repr(C) is unnecessary
because the format is endian-stable bytes, not host ABI memory.

Existing `reflect<T>;`, Fields<T>, Cases<T>, FieldType/NameOf and declaration
provenance supply ordered, checked field metadata. Existing generated-declaration
machinery can derive the Concept record reader, as Standard.Octagon.Derive already
demonstrates. Add a bounded build-time Go source emitter over that same checked
metadata, with exhaustive int/count/bool/text/tag/record/sequence cases. It emits
Go codec source; it must not introduce a parser, runtime reflection interpreter,
or a second handwritten field-order list. Numeric little-endian/bounds primitives
stay small audited primitives in each language.

Start with the operand record (seven integers, kind tag, two text slices and bool).
Next migrate the fixed argument/register/slot/terminator/frame records; finish with
counted block/instruction/function sequences. Each family must execute in the
actual EncodeMachineBridge -> Concept DecodeFunction path before the next family
moves. Keep the old Go encoder only as a test oracle during shadow qualification;
compare entire artifact bytes, then switch and remove that family's manual order.
A partial derivation must name exactly which families remain manual.

The missing implementation seam is the build-time Go codec emitter from checked
Concept reflection plus the counted-sequence lowering for this wire model. Current
GeneratorDecl retains a Concept FunctionDecl and materializes Concept functions;
it has no Go output target. There is no architectural contradiction: this is an
isolated remaining implementation item, not an excuse to add a general codegen
framework. No new bootstrap compiler or schema language is needed.

## Identity and acceptance gates

Hash normalized checked schema records, ordered fields and types, enum cases/tags,
and wire-representation selectors. Exclude spans, formatter trivia and generated
files. Embed that hash in both generated codec metadata and the new bridge header;
a schema migration must change the bridge version. Both readers reject mismatched
version/hash before consuming a function. Do not reuse CMIRAMD1 for a changed wire
layout. Validate generation by reproducing the checked Go/Concept outputs from the
schema and rejecting edits to generated output.

Run the old/manual and derived encoder on every MachineIR fixture, with exact byte
comparison (including flags, facts, decisions, spans, frame, ABI and memory fields),
100 repeated builds, and Go decode/reencode. Run the derived Concept reader on those
same artifacts and compare every projected field, then the existing native/C
oracle traces. Wrong version/hash, unknown tags, negative/large counts, truncation,
trailing bytes and workspace exhaustion remain named failures. Only then delete
the manual record/sequence writer and reader.

## Evidence available now

TestEVT2dMachineBridgeRoundTrip qualifies the current seven-function wire artifact,
including 100 identical encodings and complete Go decode/reencode.
TestEVT2dMachineBridgeRejectsMismatchAndCorruption pins wrong-version, truncated and
trailing-byte rejection. TestEVT2dConceptBackendNativeAddMax consumes the real Go
artifact in the Concept reader and compares native behavior to the C oracle.
R9a additionally compares exact-demand and larger caller encoding workspaces to
compatibility output byte-for-byte 100 times per native fixture, and rejects
undersized scratch before output writes. These are current bridge/backend
qualification gates. They are not a fabricated old-versus-derived codec result;
that shadow comparison remains required when the emitter exists.
