# R7i C/C++ companion modules

Concept is not a C++ compiler. GCC, Clang, and MSVC remain the C/C++ compilers.
Concept provides the project, semantic, testing, and migration layer around
them. A C/C++ project can adopt Concept without rewriting implementation code.
Migration to Concept source is optional and incremental.

A C/C++ header describes what can be called. A Concept companion module
describes what those calls mean. The TinyXML2 fixture keeps the C++ source
untouched and adds a narrow C ABI bridge and `concept/Native.concept`.
`extern "C"` states linkage. `foreign concept` states only the named trusted
claim; `ConceptXmlCreate` has `Allocates` with `DeclaredForeign` origin.
`ConceptXmlChildCount` has no declared effect, so its allocation effect remains
Unknown. Neither assertion analyzes the C++ implementation.

The selected `ConceptXmlStats` record and the native header agree on two `int`
fields. `concept check` compiles and executes a native compiler probe for size,
alignment, and offsets, then checks the corresponding Concept record's field
names and types. `NativeStatsSchema<ConceptXmlStats>` is a separate Concept
proof over its accessor. A missing accessor is Disproven. The probe does not
make implementation behavior universally proven.

The current `extern "C"` validator accepts scalar and builtin pointer ABI
shapes but rejects direct aggregate return. R7i uses scalar bridge calls while
retaining the checked struct as a first layout/schema boundary. C++ exceptions
must not cross the bridge. Arbitrary templates, STL containers, inheritance,
overloads, and compiler-specific object models are outside this milestone.

The migration ladder is additive: stage 0 keeps the original C++ build; stage 1
adds `manifest.concept`; stage 2 adds companion ABI declarations; stage 3 adds
`.concept_test`; stage 4 adds trusted semantic contracts and checked ABI facts.
Stage 5 may replace selected implementation modules with Concept source later.
No stage requires a flag-day rewrite, and source compatibility with all of C++
is not a design goal.
