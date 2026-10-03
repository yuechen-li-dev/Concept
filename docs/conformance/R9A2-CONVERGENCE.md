# R9a2 convergence

Result: SUCCESS — one checked Concept MachineIR schema derives both live codecs;
shadow agreement, switch and retirement are complete. R9a's remaining D1 closure
item is CLOSED. R9b and EVT2 feature expansion were not begun.

Baseline: `653762bca74677c46760aad478a4e5450c41a5b1`.
Qualified code: `7cb01b039bc1a9c2fdf61073a67e5b3f8ebdbca9`.
Compiler: `concept-evt1-stage0-go`.

| Commit | Convergence step |
| --- | --- |
| ef9c1efe44bc8ca90e7d6e86ab970e1c8a551b54 | Fix NoAllocation's name-only overload expansion using compiler-selected targets; keep unresolved behavior conservative and artifact inputs untrusted |
| 86f6a5cf6bb7f586ede258ab4ed216a351525a96 | Add checked schema, typed build-time Go emission and reflection-derived Concept codecs; qualify live shadow payload/canonical roundtrip |
| a588dc2b79877bb812bad0cc4380a4f2f28ecbfc | Switch production; qualify exact old/derived native bytes, version/hash, malformed fields and artifact identity; capture frozen baseline oracles |
| 7cb01b039bc1a9c2fdf61073a67e5b3f8ebdbca9 | Delete manual Go field encoder/decoder and test fallback; pin frozen byte parity and 100-run declaration/backend determinism |

The final documentation-only closeout commit follows the qualified code SHA.

## Closure ledger

| Requirement | Status / concrete evidence |
| --- | --- |
| Single semantic authority | BridgeSchema.concept; 73 wire fields / 52 enum cases; typed selectors and representation constants |
| Generated Go producer | machineir_bridge_codec_generated.go, generated from checked TypeInfo metadata |
| Generated Concept consumer | BridgeDerive + generated BridgeCodec derive sites; checked FunctionDecls visible through concept generated |
| Fixed wire representation | LE scalar widths, explicit fixed byte header arrays, counted text/sequences; no host struct memory |
| R9a structured applications | Metadata follows declaration/ordered arguments, including module identity, without splitting nominal display names |
| Schema version/hash | CMIRAMD2 / numeric 2 / SHA-256 99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593 |
| Old-vs-new payload | Live shadow agreement over seven shapes / 17 functions; frozen CMIRAMD1 oracles retained |
| Old-vs-new native bytes | Byte-identical; 100 emissions/function before retirement and against frozen oracles afterward |
| Complete roundtrip | Go semantic field comparison and native Concept canonical decode/re-encode preserve every transported field |
| Malformed/stale input | 23 ordinary Result/error specimens; checked schema fixture mutation rejects stale producer in native Concept |
| Artifact-only identity | Loaded schema artifact regenerates identical sources/hash; changed fixture derives checked consumer without dependency source reparsing |
| Single-side omission guard | All schema records/enums require reflection; fixture field addition reaches both generated outputs and Concept generated inputs |
| NoAllocation | Actual local codec call graph and existing backend/caller-workspace proofs pass |
| Bound/capacity behavior | Borrowed counted views; fixed backend arrays and caller encoding scratch; named errors remain |
| Manual retirement | No manual field order/tag tables or codec fallback in production or tests; audited generic byte primitives and semantic projection only |
| Determinism | 100 runs: hash, Go source, Concept declarations/identity/text, C/MIR, bridge, decoded representation, canonical re-encoding and native bytes |
| Full gates | Full Go, vet, established/new race, corpus, native EVT2/EVT2x, Standard/DragonGod/Golden/Vulkan Normal+Verify pass |
| Zig policy | Baseline suites passed; final suites skipped because the user froze legacy Zig and no Zig path changed; installed validation skill updated |
| Freeze readiness | Bridge item CLOSED; ready for R9b as a separate task; no automatic freeze |

## Boundaries retained

Octagon's existing reflection/generation machinery supplies the pattern, but the
transport remains compact binary to preserve payload/native parity. repr(C) would
unnecessarily couple the wire to host ABI padding. There is no new schema language,
universal serializer, runtime reflection registry or new bootstrap compiler.
Stage-0's producer remains Go; the AMD64 algorithms remain Concept-owned.

The existing bounded allocator/liveness/decoder model and register exhaustion
remain. Enum-payload innate migration, generic Verdict, fact-granting trust,
imported-byte reflection, broader hardware/Vulkan qualification and frontend
self-hosting remain separate research or milestones. They are not new R9a2 bridge
blockers and are not silently claimed complete.

R9A2-CONFORMANCE records exact gates, diagnostics, timings and local logs;
EVT2-MACHINEIR-BRIDGE records source/identity/bootstrap paths. The tracked worktree
is clean at final closeout; local logs and historical snapshots are ignored evidence.

SUCCESS — R9a2 single-source MachineIR bridge established.
R9a closure complete; ready for R9b.
