# R8 project policy design seam

> Project policy may require semantic facts, but it cannot create them.

> `concept lint` must operate on bound semantic declarations and proposition
> truth, not source-text pattern matching.

R8e2 begins with `DeclarationSubjects` over the validated `Module`. The
projection has stable declaration identity, kind, defining module, source site,
and Authored/Generated/Foreign provenance. `ProjectDeclarationSubjects` scopes
the view to the current module, excluding dependency implementation declarations
loaded from semantic artifacts. Generated functions use `GeneratedOrigin`;
foreign functions use the existing `ExternABI` declaration. These categories
are never inferred from spelling or path.

The current compiler's `ConceptDecl` parameters denote types. Its proof engine
also accepts operation and value subjects for `Assert.Concept`, but a concept
parameter cannot yet bind a declaration as a first-class semantic subject.
The next R8e2 step is to make declaration parameters bind through that existing
concept/proof machinery, then evaluate naming and proposition requirements
without modifying semantic truth. Manifest policy references and severities must
bind against those concepts before adding `concept lint`; a hardcoded naming
visitor would misrepresent the policy system.

Naming is project policy, not grammar. The intended canonical style is
PascalCase for types, functions, methods, concepts, interfaces, and machines;
camelCase for fields, locals, and parameters. Generated and foreign declarations
are exempt from the default authored policy by semantic provenance. The current
progression does not enforce naming or expose a lint command.

Language errors such as ownership violations, invalid types, and ignored
MustUse results remain core compiler obligations. Naming and requirements such
as NoAllocation are project policy findings. Policy cannot change codegen or
upgrade Unknown or Disproven facts to Proven.

R8e3 remains a separate milestone for source trivia, formatter configuration,
and `concept fmt`. R8f differentiator goldens follow R8e3; R7q remains paused.
