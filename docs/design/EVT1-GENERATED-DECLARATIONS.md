# EVT1 checked generated declarations

Generated declarations are ordinary Concept declarations with explicit generated provenance. Generation changes authorship, not semantics.

R7g1 introduces a bounded structural generator declaration and an explicit derivation site:

```concept
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }

derive DeriveValue reflect<Item>;
```

The generator body is a normal function syntax tree retained as a template, not a runtime function. `derive` closes its type parameter with the reflected type and inserts a `FunctionDecl` into the current module before final name binding, type checking, effect and ownership analysis, MIR, and C lowering. The reflected target must be closed. Imported targets obey the existing type-level `[[reflect]]` permission. There is one generation pass; generated declarations cannot trigger another generation wave.

The bounded field query `foreach (FieldInfo field in Fields<T>(selector))` expands its body in source field order. The selector may name an attribute or a one- or two-parameter concept. A concept selector tests the field type (and, for two parameters, the reflected owner type) through ordinary concept satisfaction; fields without a witness are skipped. `item.field` becomes an ordinary field expression. The body can contain ordinary nested iteration for fixed arrays and table columns.

`OnlyField<T>()` is the bounded single-field query. It expands exactly one reflected field and rejects a target with zero or multiple fields at the derivation site. It is useful when generated code requires one representation value rather than an arbitrary field sequence.

For payload enums, `foreach (EnumCaseInfo variant in Cases<T>())` in a writer with an enum-valued first parameter constructs one exhaustive ordinary `MatchStmt`. Pattern bindings are created for every payload position; `Payload<variant>()` expands typed statements inside the arm. An enum reader can instead use one `if` in that case loop. It expands to ordinary branches; `HasPayload<variant>()` becomes a boolean literal, `FieldType<field>` becomes the concrete payload type, and `ConstructCase<variant>()` becomes ordinary typed enum construction from payload locals. The normal checker validates all calls, payload types, and exhaustive writer matches. Malformed queries and unsupported structural operations are diagnosed.

The generated body is checked as ordinary Concept; the generator has no direct symbol-table or backend access. One function is limited to 4,096 generated statements.

Concept reflection generation is structural. It does not generate source text and reparse it. The existing structural type substitution and statement cloning code constructs the final declaration. Neither the generator nor `[[reflect]]` adds runtime reflection metadata.

Identity is a digest over module, generator identity, derivation site, reflected type, selected field names and types, local ordinal, and operation signature. Each generated function stores generator module and source span, derivation module and site, reflected module and type, and selected field names, types, and spans. Duplicate generated functions and collisions with handwritten functions report the generation site and the competing origin. The normal analyzer remains the authority for other semantic conflicts.

Generated declarations are materialized during module compilation and transported as semantic declarations through module artifacts. Consumers need not rerun generators. The `concept-module.v1` semantic payload stores the checked function syntax and provenance; the envelope hash covers that payload. A consumer may also derive against an imported `[[reflect]]` type using only its artifact. Imported derivation requests are not replayed.

`concept generated <file> [symbol]` returns a structured view of the ordinary declaration AST and its origin. `concept explain <file> --generated <symbol>` emits a `concept-proof.v1` graph with generator, derivation site, reflected type, and selected field inputs. `concept explain <file> --concept 'Trace<Node>'` first checks the concept through ordinary required-operation lookup, then attributes its generated witness. Existing source-position `concept explain` behavior is unchanged.

This is a bounded function and required-operation-witness path. It does not synthesize new structs, enum declarations, open generic declarations, or arbitrary multi-wave generation. Runtime reflection remains deferred.
