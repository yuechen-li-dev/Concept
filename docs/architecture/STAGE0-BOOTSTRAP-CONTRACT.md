# Stage-0 bootstrap contract (R9c)

Stage-0 is the durable, independently bootstrappable Go seed. Its preserved
external path is Concept source -> Stage-0 Go -> portable C -> GCC/Clang/MSVC.
The normal-path target is Cathedral, but success there does not retire this
path. The C backend remains useful for bootstrap, portability and fallback.

Permanent seed responsibilities are parsing and enough declaration/type and
semantic checking to compile Cathedral from source, versioned artifact/bootstrap
loading, bootstrap orchestration, C emission/transport, and small audited host
operations Concept cannot yet supply. These are permissions to keep a reliable
seed, not a mandate that the Go frontend permanently own normal semantics.

Temporary migration responsibilities are today's MIR/Planner/LIR/MachineIR
production, full validation and proof hosting, generated-declaration
materialization, ABI input selection, and other decisions still in Go. Track
them by semantic decision in the authority ledger. New AMD64 lowering,
allocation, call/ABI policy, spills, machine optimization, object emission,
generated codec rules and declarative validation belong to Cathedral by
default. Go can carry contract data and bootstrap orchestration; a permanent
new Go decision requires explicit justification. No Stage-0 freeze occurs in
R9c.

The trusted bootstrap core currently includes parser and source spans,
declaration and type identity, generic closure/comptime evaluator, artifact
reader, `compiler.*` observation projections, MIR/LIR/MachineIR production and
verification, proof provenance/renderer, C generation and external C compiler.
The Concept AMD64 library additionally trusts the CMIRAMD2 checked bridge
schema and host C ABI used to execute its generated C. Shrink the conceptual
TCB by moving normal decisions to checked Concept contracts; LOC is not a
sufficient metric.

`compiler.*` observations are versioned compatibility inputs to Concept rules,
not arbitrary access to Go internals. Typed declaration/type identities,
ordered fields/cases, checked effects and bounded geometry have active
consumers. Display names and implementation-shaped strings are not identity.
Unknown or out-of-scope observations remain Unknown, never manufactured proof.
Privileged APIs must be marked as semantic observation, bootstrap host
operation, or unsafe/native primitive. The trusted-fact registry is empty at
this R9c boundary.

Before a future Stage-0 freeze, require all of: a stable versioned MachineIR
bridge for the then-current Cathedral backend; stable semantic artifact
identity and observation contracts; closed generic/comptime capability enough
to compile Cathedral; Cathedral ownership of active native development;
reliable fresh-checkout bootstrap and regression tests; and no known major
Stage-0-only semantic feature needed for Cathedral. Freeze is a separate
decision with evidence, not a byproduct of R9c.

Fresh-checkout reproducibility plan: pin a known Go and C toolchain, build
Stage-0, build Cathedral libraries/executable through Stage-0 C output, run
Cathedral qualification, then use Cathedral to rebuild itself when that path
exists. Compare at increasing levels: semantic decisions, versioned artifacts,
MachineIR, object output, and only then byte-identical executables if justified.
Current evidence reaches Concept backend library compilation and native-byte
oracles; no Cathedral-on-Cathedral rebuild is claimed.
