# Declaration concept parameters

> Concept parameters may bind semantic declaration subjects as well as types.
> Declaration-subject parameters use stable bound declaration identity and the
> ordinary proposition/proof system.

`concept C<T>` keeps a type parameter. `concept C<declaration D>` declares a
different parameter category; a declaration never becomes a fake `Type`.
Top-level applications spell the category explicitly:

```concept
concept IsForeign<declaration D> { requires compiler.Foreign(D); }
concept ForeignPolicy<declaration D> { requires IsForeign<declaration D>; }
requires ForeignPolicy<declaration native_status>;
```

`Assert.Concept<ForeignPolicy>(native_status, "reason")` resolves the named
operation through the same semantic binder. A type where a declaration is
required, or a declaration where a type is required, emits
`CONCEPT_ARGUMENT_CATEGORY_MISMATCH`.

The tagged `conceptSemanticArgument` carries either `Type` or
`DeclarationSubject`. The latter contains its defining module, kind, spelling,
provenance, site, and stable ID. It survives semantic artifacts; an imported
declaration can satisfy a concept without provider source or generator rerun.

Compiler analyses on declaration parameters include `DeclarationKind`,
`Authored`, `Generated`, `Foreign`, `Name`, the nine kind predicates,
`CanonicalName`, explicit name styles, and `NoAllocation` for functions and
methods. They use the ordinary proof graph and preserve Proven, Unknown, and
Disproven. `NoAllocation` calls the existing function proof projector, including
imported effect summaries and foreign Unknown. Requiring it cannot grant it.

Declaration concepts accept compiler analyses and prerequisite concept
composition. Runtime operation and field requirements remain for type
concepts; interfaces remain type-parameterized. No module, token, or arbitrary
AST-node parameter category is exposed.
