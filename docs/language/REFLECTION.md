# Reflection and bounded generation

Compile-time reflection is code generation with provenance. It is not a runtime object model.

The currently implemented structural query is an explicit module-scope declaration:

```concept
[[reflect]] struct Node { int value; private: int next; };
reflect<Node>;
```

`reflect<T>;` accepts a parsed type, resolves it through ordinary semantic analysis, and produces a compiler-only `TypeInfo` snapshot. `concept reflect file.concept` prints the snapshots as JSON in source order. A reflection-only query does not change emitted C, headers, or MIR. An explicit `derive Generator reflect<T>;` may instead construct an ordinary checked function declaration using the bounded structural generator path.

`TypeInfo` contains semantic type identity, kind, name, defining module, generic arguments, fields or enum cases, source spans, and structured attributes. `FieldInfo` contains declaring and field types, name, declaration index, visibility, mutability as currently represented by record versus mutable aggregate, and source span. Payload enum cases retain their tag and payload fields. Sized tables retain table kind, cardinality, and column types. Closed generic fields use substituted types. Array and storage distinctions remain in the ordinary `Type` representation; recursive declarations are not expanded recursively.

Local reflection can inspect the module's own declarations. Cross-module structural reflection requires `[[reflect]]` on the defining type, including generic type definitions. That annotation permits compile-time structural disclosure across the import boundary, including private fields; it does not request runtime metadata. An unmarked imported type reports `REFLECT_PERMISSION_REQUIRED`. The type-level annotation takes no arguments.

The reflection result is compiler data and is not exported as a result in `concept-module.v1`. Imported requests are not replayed. The artifact's readable exports list reflectable type names; structural details needed by the compiler remain in its private semantic payload. The dependency is loaded from its semantic artifact, without reparsing its source. No runtime field lookup, invocation, mutation, registry, or serializer exists.

Generated functions are inspectable with `concept generated <file> [symbol]` and attributable with `concept explain <file> --generated <symbol>`. The generator can expand source-ordered fields through `Fields<T>(attribute-or-concept)`, and enum payloads through `Cases<T>()` and `Payload<case>(selector)`. Nested ordinary loops traverse fixed arrays and table columns. This bounded API does not expose `TypeInfo` as an arbitrary runtime or comptime value. See [checked generated declarations](../design/EVT1-GENERATED-DECLARATIONS.md).

Inside a reflected field expansion, `FieldType<field>` substitutes the
field's semantic type into an ordinary type position. It can be nested in a
closed generic type, such as `OctagonType<FieldType<field>>`. The resulting
call is resolved and checked by normal overload rules; no field operation is
selected by a runtime type name.

EVT1 production code has no general runtime reflection. Runtime reflection restricted to `.concept_test` with explicit `[[reflect]]` is not implemented yet. Production runtime reflection remains uncommitted. If ever introduced, it must be explicit, bounded, measurable, and must not create an ambient reflection universe.
