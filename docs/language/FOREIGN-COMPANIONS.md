# Foreign companions

An `extern "C"` declaration names an external ABI symbol. It does not prove
allocation, ownership, lifetime, purity, or native implementation behavior.
An adjacent `foreign concept` can state a precise trusted semantic contract:

```concept
extern "C" byte* ConceptXmlCreate();
foreign concept CreateDocumentContract on ConceptXmlCreate
{
    requires compiler.Allocates(ConceptXmlCreate);
}
```

Concept records this claim as `DeclaredForeign`. The native source remains
opaque to the Concept compiler. Unstated claims remain Unknown; for example
the TinyXML2 `ConceptXmlChildCount` declaration has no allocation contract.
`concept explain` displays either outcome and its origin. `.concept_test` is a
language-level semantic harness over native code, not merely another assertion
macro framework: it combines runtime calls with compile-time concept proofs.

Use an `extern "C"` bridge for C++ APIs that otherwise expose classes or STL
types. Keep exceptions inside the bridge. For layout claims, declare the
selected Concept record and run a probe compiled by the native toolchain.
The initial boundary permits opaque handles and scalar calls. Direct aggregate
returns currently report `EXTERN_C_ABI_TYPE_INVALID` and require a future
general ABI extension before they can be called.
