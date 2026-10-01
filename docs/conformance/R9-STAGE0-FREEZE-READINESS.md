# Stage-0 freeze readiness

Stage-0 is not frozen. A1's frontend self-hosting blocker is removed: closed generic
identity survives artifacts structurally without nominal-name reparsing. Explicit
ownership remains. Modifier/attribute and context holes are repaired; generic repr(C)
records are checked after closure.

Runtime-function static selection and bounded expansion now work in the C11 path.
Ordinary predicate requirements
now narrow concept satisfaction using declared proof provenance, including project
policy and artifact-only Assert.Concept. CV4138 has completed innate migration;
large-library measurements support keeping the current limits and mutex.
The bridge now has a concrete bounded derivation path and current round-trip/native
qualification, but its Go producer/Concept consumer still duplicate field order.
The isolated next closure item is a build-time Go codec emitter from checked Concept
reflection, followed by old-versus-derived wire comparison and schema-hash identity.
Backend encoding has a real caller-supplied scratch path with explicit exhaustion;
the liveness capacity mismatch is already absent at EVT2x6. Other backend phases
remain bounded. Neither frontend self-hosting nor R9b is begun.
