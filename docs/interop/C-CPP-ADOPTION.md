# C and C++ adoption

Status: EVT1 native semantic companion workflow implemented; source spelling for foreign semantic declarations remains provisional

Concept does not require an existing C or C++ codebase to adopt Concept syntax. The practical migration is incremental:

1. Keep production C/C++ unchanged.
2. Add focused `.concept_test` coverage.
3. Describe exported C entry points in a Concept ABI companion module.
4. Add explicit semantic declarations for the guarantees the native code owns.
5. Add proof assertions for units, effects, accessibility, and lifetime.
6. Write new low-level modules in Concept when that is useful.
7. Migrate old C++ only when the change pays for itself.

For C++, expose a narrow `extern "C"` wrapper around the existing implementation. A companion such as `Legacy.Audio.concept` describes that representation-level ABI and its separately reviewed semantics. Concept does not parse headers, infer contracts from names, or require annotations in the C++ source.

This lets `.concept_test` complement or replace individual GTest cases while the production implementation stays put. Concept tests can add unit checking, allocation-effect checks, lifetime/provenance checks, and structured proof output. They do not make a declared native guarantee compiler-verified.

CMake replacement is outside this milestone. Semantic modules already express Concept dependencies; future tooling may own native compilation and linking. Full C++ object ABI, header import, generated ABI declarations, callbacks, and automatic contract generation remain deferred.
