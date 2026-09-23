# EVT1 checked generated declarations

Generated declarations are ordinary Concept declarations with explicit generated provenance. Generation changes authorship, not semantics.

R7g1 introduces a bounded structural generator declaration and an explicit derivation site:

```concept
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }

derive DeriveValue reflect<Item>;
```

The generator body is a normal function syntax tree retained as a template, not a runtime function. `derive` closes its type parameter with the reflected type and inserts a `FunctionDecl` into the current module before final name binding, type checking, effect and ownership analysis, MIR, and C lowering. The reflected target must be closed. Imported targets obey the existing type-level `[[reflect]]` permission. There is one generation pass; generated declarations cannot trigger another generation wave.

The bounded field query `foreach (FieldInfo field in Fields<T>(attribute))` expands the loop body once per source-ordered field carrying that attribute. `item.field` inside that body becomes the field selector in the ordinary expression AST. Fields without the annotation are skipped. The generated body is then checked as ordinary Concept; the generator has no direct symbol-table or backend access. Invalid field queries and unsupported structural operations are diagnosed.

Concept reflection generation is structural. It does not generate source text and reparse it. The existing structural type substitution and statement cloning code constructs the final declaration. Neither the generator nor `[[reflect]]` adds runtime reflection metadata.

Identity is a digest over module, generator identity, derivation site, reflected type, selected field names and types, local ordinal, and operation signature. Each generated function stores generator module and source span, derivation module and site, reflected module and type, and selected field names, types, and spans. Duplicate generated functions and collisions with handwritten functions report the generation site and the competing origin. The normal analyzer remains the authority for other semantic conflicts.

Generated declarations are materialized during module compilation and transported as semantic declarations through module artifacts. Consumers need not rerun generators. The `concept-module.v1` semantic payload stores the checked function syntax and provenance; the envelope hash covers that payload. A consumer may also derive against an imported `[[reflect]]` type using only its artifact. Imported derivation requests are not replayed.

`concept generated <file> [symbol]` returns a structured view of the ordinary declaration AST and its origin. `concept explain <file> --generated <symbol>` emits a `concept-proof.v1` graph with generator, derivation site, reflected type, and selected field inputs. `concept explain <file> --concept 'Trace<Node>'` first checks the concept through ordinary required-operation lookup, then attributes its generated witness. Existing source-position `concept explain` behavior is unchanged.

This is a bounded function and required-operation-witness path. It does not yet synthesize structs, enum declarations, arbitrary control flow from enum cases, or open generic declarations. Runtime reflection remains deferred.
