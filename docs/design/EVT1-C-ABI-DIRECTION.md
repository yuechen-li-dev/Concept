# EVT1 C ABI direction

R7j completes a bounded C-compatible aggregate call path. It does not attempt arbitrary C++ object-model compatibility. The generated strict-C11 declarations use fixed-width C scalar types and native structs; the selected C compiler performs target ABI call lowering, including any hidden return pointer. No runtime ABI registry or dispatch was added.

Admission is semantic: a by-value aggregate must be a nonempty `[[repr(C)]] record struct` with recursively eligible fields and no Drop authority. Missing representation, unsupported fields, and nontrivial ownership receive field-specific `EXTERN_C_ABI_TYPE_INVALID` or `C_ABI_REPR_INVALID` diagnostics. A source-declared `CAbiValue<T>` concept can reuse the compiler's `CAbiLayout<T>` fact without a separate marker subsystem.

The TinyXML2 bridge keeps its XMLDocument handle opaque while returning and round-tripping a two-field `ConceptXmlStats` by value. This exercises real C++ library code through an `extern "C"` C-compatible bridge. C++ exceptions and object-model types do not cross it.
