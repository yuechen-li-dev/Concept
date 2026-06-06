# Phase 3 semantic architecture

Phase 3 moves Concept from the Phase 2 AST-based executable subset toward the real semantic compiler architecture.

## Thesis

```text
Phase 2 proved Concept can run tiny programs.
Phase 3 gives Concept a semantic spine:
symbols, HIR, types, name resolution, and real checking.
```

Phase 2 intentionally proved execution through the smallest useful path:

```text
Source -> AST -> executable checker -> C backend
```

That path is useful, but it is not the long-term compiler spine. The Phase 2 checker validates a narrow executable subset directly from source-shaped AST nodes, and C backend v0 emits from that same transitional representation. Phase 3 introduces the semantic layer that later phases can build on without forcing HIR, type checking, or diagnostics to inherit source syntax or C backend constraints.

Phase 3 should preserve every Phase 2 run fixture while replacing the fragile parts of the pipeline with explicit semantic data: interned names, stable IDs, a HIR layer, a type store, declaration collection, name resolution, and HIR-based checking.

## Goals

Phase 3 should introduce the compiler structures needed for a real semantic pass while staying narrow enough to converge.

Required goals:

- Introduce string interning and `SymbolId`.
- Introduce semantic IDs:
  - `ItemId`
  - `FunctionId`
  - `StructId`
  - `EnumId`
  - `FieldId`
  - `VariantId`
  - `LocalId`
  - `ParamId`
  - `TypeId`
- Introduce HIR separate from AST.
- Introduce `TypeStore` and a real type model.
  - P3-M3 starts this as standalone infrastructure: `TypeId` values identify entries in a `TypeStore`, with stable builtin IDs for `void`, `int`, and `bool`, plus nominal struct/enum type entries that reference HIR declaration IDs.
- Resolve top-level names in one-file modules.
- Resolve type names for builtins and user declarations.
- Check duplicate top-level declarations.
- Check duplicate struct fields.
- Check duplicate enum variants.
- Lower function bodies into HIR.
- Replace the Phase 2 AST-based checker with HIR-based checking.
- Migrate C backend v0 to emit from HIR or a HIR-backed executable representation.
- Preserve all Phase 2 run fixtures.

Phase 3 is not about broadening the runtime language. It is about making the already-proven executable subset travel through the right semantic architecture, then adding declaration-level semantic checks that are impossible or awkward in the Phase 2 path.

## Non-goals

Phase 3 deliberately does not include:

- MIR.
- Borrow or move checking.
- `Drop<T>`.
- `Result<T, E>` or `try`.
- Generic concept solving.
- Interface or vtable semantics.
- A full multi-file module graph.
- Real imports beyond parsed imports.
- Payload enum runtime layout.
- Payload enum pattern matching.
- Arrays.
- Runtime strings, chars, or floats.
- Pointer or reference runtime semantics.
- C ABI or layout correctness.
- LLVM.
- Bare-metal target work.
- Self-hosting.

These exclusions keep Phase 3 focused on the semantic spine rather than expanding into the full language roadmap.

## AST vs HIR

```text
AST is source-shaped.
HIR is compiler-shaped.
AST preserves syntax.
HIR resolves names, normalizes items, assigns IDs, and prepares for type checking and later MIR.
```

The AST should keep representing what the user wrote. It is useful for source fidelity, parser tests, syntax diagnostics, and preserving spelling choices. It may retain raw token text, source order, syntactic grouping, and constructs that are not semantically meaningful yet.

HIR should represent what the compiler needs to know after syntax has been collected and resolved. HIR should not preserve every syntactic accident. It should attach stable semantic IDs, refer to interned symbols, resolve declarations, make scopes explicit, and give later phases a uniform representation for checking and lowering.

Example source that is AST-shaped:

```cpp
int main() {
    int x = 1 + 2;
    return x;
}
```

Example HIR-ish shape:

```text
FunctionId(0) main
  return_type: TypeId.Int
  params: []
  locals:
    LocalId(0) x: TypeId.Int
  body:
    LocalDecl LocalId(0) = Binary(+ Int(1), Int(2))
    Return LocalRef(LocalId(0))
```

In the AST, `x` is an identifier token in a declaration and another identifier token in a return expression. In HIR, those two pieces are connected: the declaration creates `LocalId(0)`, and the return expression references `LocalId(0)`. Similarly, the text `int` becomes `TypeId.Int`, and a call like `add(1, 2)` should become a call to a resolved `FunctionId`, not a string lookup repeated by later passes.

The same distinction applies to declarations. A parsed struct field may contain a type-name spelling and a field-name spelling. A HIR struct should contain `FieldId`s, `SymbolId`s for field names, and `TypeId`s for resolved field types.

## Proposed Phase 3 pipeline

Current Phase 2 pipeline:

```text
Source -> AST -> executable checker -> C backend
```

Phase 3 target pipeline:

```text
Source
  -> AST
  -> declaration collection
  -> HIR
  -> type/name checking
  -> HIR-backed C backend
```

Declaration collection should create the module-level semantic index before function bodies need to be checked. HIR lowering should then turn parsed declarations and bodies into compiler-shaped nodes with stable IDs. Type and name checking should operate over those IDs and `TypeId`s.

MIR is still Phase 4. Phase 3 HIR should be good enough to support name/type checking and the existing executable subset, but it should not become a disguised MIR. Control-flow normalization, dataflow analysis, borrow/move checking, and backend-oriented lowering belong later.

C backend v0 remains an audit/debug/temporary backend. It should continue to produce readable C for the executable subset, but HIR should not be forced to look like C. If the backend needs a small HIR-backed executable view as a stepping stone, that view should be treated as transitional backend plumbing, not as the semantic model itself.

## Symbol interner plan

Phase 3 should introduce an interner that maps identifier text to stable `SymbolId` values.

Expected properties:

- All names eventually become symbols.
- Name comparison becomes ID comparison.
- HIR, type, and name-resolution layers use interned names.
- The AST may continue to hold raw token text for source fidelity.
- Diagnostics can use the interner to render symbols back to user-facing text.

The interner should be shared by the semantic session for one compilation. When declaration collection sees top-level names, field names, variant names, parameters, locals, and type-name components, it should intern them. Later semantic passes should compare `SymbolId`s instead of repeatedly comparing strings.

P3-M1 introduces the standalone `Interner` and compact `SymbolId` value used by later semantic milestones. Parser integration and semantic consumers remain future work.

P3-M2 introduces the standalone HIR skeleton and compact semantic ID values for declarations and declaration children. It provides manual HIR store construction only; AST lowering, type checking, and backend migration remain future milestones.

P3-M5 extends declaration HIR with resolved `TypeId`s for function returns and parameters, struct fields, and enum variant payload fields. Declaration type-name resolution now accepts `void`, `int`, `bool`, and nominal top-level struct and enum names, while rejecting unknown type names, function names used as types, and unsupported syntax such as dotted, generic, pointer, or reference type forms. It also adds declaration-level duplicate checks for struct fields, enum variants, and function parameters. Function body lowering remains deliberately deferred.

This is especially important before introducing richer module summaries and imports. A one-file Phase 3 module can still use the same discipline that later multi-file resolution needs.

## TypeStore plan

Phase 3 should introduce `TypeStore` as the owner of semantic type identities. Types should be referred to by `TypeId`, not by raw AST type-name nodes or the Phase 2 `ExprType` executable classification.

Initial builtin types:

- `void`
- `int`
- `bool`

Initial nominal user types:

- Structs.
- Enums.

Future type categories:

- Pointers.
- References.
- Arrays.
- Function types.
- Generic instantiations.

Phase 3 only needs enough runtime typing to preserve and replace Phase 2 executable behavior plus semantic declaration checks. For example, function return types, parameter types, local types, struct field types, and enum declaration identities should resolve to `TypeId`s. That does not require implementing runtime layout for structs, enum payloads, arrays, pointers, or references.

`TypeStore` should make builtin and nominal type identity explicit:

```text
TypeId.Void
TypeId.Int
TypeId.Bool
TypeId.Struct(StructId(0))
TypeId.Enum(EnumId(0))
```

The exact representation can differ, but semantic passes should see stable type IDs rather than reparsing or re-resolving type spelling at every use site.

## Declaration collection and name resolution

Phase 3 name resolution is one-file only.

Declaration collection should collect top-level declarations before checking bodies:

- Functions.
- Structs.
- Enums.

Concepts, interfaces, and impls may be stored as placeholders or remain syntax-only initially. They are part of the Phase 1 parser surface but are not Phase 3 semantic runtime targets.

The collector should reject duplicate top-level names in the one-file module. This includes collisions such as a function and struct sharing the same top-level name unless the language later deliberately introduces separate namespaces.

Name resolution should:

- Resolve function declarations to `FunctionId`.
- Resolve struct declarations to `StructId`.
- Resolve enum declarations to `EnumId`.
- Resolve struct fields to `FieldId`.
- Resolve enum variants to `VariantId`.
- Resolve function parameters to `ParamId`.
- Resolve local declarations to `LocalId`.
- Resolve function calls to `FunctionId`.
- Resolve local and parameter references to `LocalId` or `ParamId`.
- Resolve builtin and user type names to `TypeId`.

Parsed imports remain inert for now. They should not affect name lookup, type lookup, duplicate detection, backend emission, or diagnostics except as syntax that the parser accepts. A later multi-file module phase can replace this with a real module graph and imported module summaries.

## HIR body lowering and checking

Function body lowering should turn parsed function bodies into HIR bodies:

- Function parameters become `ParamId`s.
- Local declarations become `LocalId`s.
- Identifier expressions become resolved references.
- Function calls become `FunctionId` calls.
- Expression checking uses `TypeId`, not `ExprType`.
- Block scopes are represented explicitly or through scoped symbol tables.
- Phase 2 behavior remains unchanged.

The Phase 2 subset already supports integer and boolean expressions, returns, locals, assignment, calls, `if`, `match`, and `while`. Phase 3 should not broaden that executable set by accident. Instead, it should prove the same behavior through HIR-backed checking.

A HIR body can represent resolved values like this:

```text
FunctionId(1) add_one
  params:
    ParamId(0) value: TypeId.Int
  return_type: TypeId.Int
  body:
    Return Binary(+ ParamRef(ParamId(0)), Int(1))
```

For scopes, either explicit HIR scope nodes or a lowering-time scoped symbol table is acceptable. The important semantic result is that each identifier reference points to exactly one declaration ID or produces a clear diagnostic. Shadowing policy can be specified and enforced once scoped symbol tables exist; Phase 3 should at least avoid ambiguous accidental lookups.

## Semantic diagnostics and `.conception` fixtures

Phase 3 should expand `.conception` fixtures beyond parse and run behavior with semantic check fixtures.

Planned support:

- `# phase: check`
- `=== diagnostics ===` code matching for semantic failures
- Valid declaration-check fixtures
- Invalid duplicate-declaration fixtures
- Invalid unknown-type fixtures

Exact future diagnostic codes may start as placeholders while the diagnostic catalog stabilizes, but fixtures should assert stable codes once implemented.

Example valid check fixture:

```text
# name: struct declaration checks
# phase: check
# expect: pass

=== source ===
module Example;

struct Vec3 {
    int x;
    int y;
    int z;
};
```

Example invalid check fixture:

```text
# name: duplicate field rejected
# phase: check
# expect: fail

=== source ===
module Example;

struct Vec3 {
    int x;
    int x;
};

=== diagnostics ===
CON0010
```

Useful Phase 3 fixture families:

- Valid builtin type references in function signatures and fields.
- Valid user type references in fields and signatures.
- Duplicate top-level function/function names.
- Duplicate top-level function/struct or struct/enum names.
- Duplicate struct fields.
- Duplicate enum variants.
- Unknown type names.
- Unknown local references.
- Unknown function calls.
- HIR-backed run fixtures copied from or shared with the Phase 2 run corpus.

The existing Phase 2 run fixtures must continue to run through the real parse, check, backend, compile, and execute path. The semantic check fixture corpus should add coverage without weakening the run corpus.

## Milestone ladder

```text
P3-M0  Phase 3 semantic architecture document
P3-M1  Interner / SymbolId
P3-M2  HIR skeleton / semantic IDs
P3-M3  TypeStore / TypeId model
P3-M4  Top-level declaration collection
P3-M5  Type-name resolution and declaration checks
P3-M6  Function body lowering to HIR
P3-M7  HIR-based checker for Phase 2 executable subset
P3-M8  C backend from HIR-backed representation
P3-M9  Phase 3 semantic fixture corpus and closeout
```

The order is intentional. Interning and IDs come before HIR. `TypeStore` comes before meaningful type-name resolution. Declaration collection comes before body checking so calls and type references can resolve against a known top-level index. Backend migration comes after HIR-based checking so the C backend follows semantics instead of defining them.

## Close criteria

Phase 3 is complete when:

- The HIR layer exists.
- Symbols are interned.
- `TypeStore` exists.
- Top-level names resolve.
- Builtin and user type names resolve.
- Duplicate top-level declarations are rejected.
- Duplicate struct fields are rejected.
- Duplicate enum variants are rejected.
- Function bodies lower and check through HIR.
- The Phase 2 run fixtures still pass.
- Semantic `.conception` check fixtures exist.
- The old AST-based executable checker is retired or clearly marked transitional.
- C backend v0 emits from HIR or a HIR-backed executable representation.

The convergence requirement for Phase 3 is not merely that new code exists. The motivating path must improve: Concept source should still become a native executable for the Phase 2 subset, but the route should now pass through semantic data structures that can support future MIR, richer diagnostics, modules, and real type checking.

## P3-M4 note: top-level declaration collection

P3-M4 adds AST-to-semantic declaration collection for one-file modules. The pass interns top-level function, struct, and enum names; creates stable HIR declaration IDs; registers nominal struct and enum types in `TypeStore`; and reports duplicate top-level names across the shared Phase 3 function/struct/enum namespace. Function bodies, type-name resolution, field checking, enum variant checking, and Phase 2 checker/backend migration remain later milestones.
