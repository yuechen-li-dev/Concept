# Phase 15: C ABI, extern declarations, and repr(C)

P15-M0 was a documentation-only milestone. It defined Concept's explicit C ABI
source boundary, foreign C declarations, C-exported Concept functions,
`repr(C)` struct layout promises, ABI type validation, HIR/MIR representation,
and C backend emission plan. P15-M1 started implementation with the
`extern "C"` parser/AST scaffold.

P15-M1 adds the parser/AST scaffold for block-form `extern "C"` declarations.
The lexer recognizes `extern`, the parser accepts `extern "C" { ... }`, and the
AST preserves the ABI string span, block span, declaration order, and foreign
function signatures.

P15-M2 lowers valid extern C function declarations into HIR. Extern functions
reuse `HirFunction` with explicit `is_extern`, ABI, ABI-span, and C symbol-name
metadata. They have no body, are visible to ordinary call resolution, and are
type-checked with ordinary call arity/type rules. P15-M2 also validates the v0 C
ABI type subset, rejects duplicate extern C symbols, and stops emitting
`CON0259` for valid declarations. P15-M3 lowers extern C calls through MIR,
emits backend C prototypes for extern declarations, and emits calls with declared
C symbol names. P15-M4 implements `export "C"` function definitions: exported
functions parse as ordinary bodies with C linkage metadata, lower through HIR and
MIR, validate C ABI-compatible parameter and return types, reject duplicate C
ABI symbols across extern declarations and exports, and emit unmangled C function
definitions while ordinary functions retain internal backend names. Headers,
includes, linker flags, and `repr(C)` remain deferred.

## Core doctrine

```text
C interop is explicit.

extern "C" declares foreign C symbols; it does not parse headers.

export "C" exposes Concept functions with C linkage.

repr(C) promises C-compatible layout only for supported structs.

Unsupported ABI types are rejected, not guessed.

C ABI lowering must be inspectable in HIR/MIR/C output.

No hidden linker magic.

No C++ interop in Phase 15.

No bindgen, no header parser, no varargs in v0.
```

## 1. Motivation

Concept already has a MIR-backed C backend and explicit systems-language goals,
but it has no user-visible source-level C ABI boundary. That gap matters for
runtime work, OS and libc calls, kernel and embedded interfaces, backend helper
interactions, and future self-hosting.

Existing generated C helper declarations for arenas, machines, and borrowed
dyn internals are backend-owned implementation details. Phase 15 is about the
source surface a Concept author writes when crossing a real C boundary:

- foreign C functions called from Concept;
- Concept functions exposed to C;
- data layout promises for C-compatible structs.

C interop must be explicit so Concept does not accidentally rely on ambient
headers, hidden linker behavior, C++ ABI assumptions, or guessed layout. The
language should make the boundary visible in source, HIR, MIR, and generated C.

`extern "C"`, `export "C"`, and `repr(C)` belong together because they define
the same edge from three directions: imported functions, exported functions,
and data representation.

## 2. Source syntax: extern "C" declarations

Preferred source syntax:

```cpp
extern "C" {
    int puts(char* s);
    int abs(int value);
}
```

Rules:

- `extern "C"` introduces foreign declarations.
- Declarations inside have no bodies.
- Only function declarations are required for v0.
- Foreign functions are callable from Concept.
- Foreign functions are not lowered as Concept function bodies.
- Foreign function names use C symbol names.
- No C header parsing.
- No implicit `#include`.
- No automatic library linking.
- No varargs in v0.
- No C++ mangling.
- No overloaded extern C functions.
- Duplicate foreign names are rejected consistently with top-level name rules.

The single-declaration form is a reasonable future convenience:

```cpp
extern "C" int puts(char* s);
```

Recommended v0 implementation path: implement block form only. It keeps parsing
clear, gives the ABI region a visible scope, and avoids ambiguity with ordinary
function modifiers while the parser surface is still small.

P15-M1 implementation status:

- block-form `extern "C" { ... }` parses and is preserved in AST debug output;
- the ABI token must be the exact string literal `"C"`;
- empty extern blocks are allowed as a parser scaffold;
- declarations inside the block are semicolon-terminated function signatures
  only;
- function bodies are rejected with `CON0262`;
- non-function entries are rejected with `CON0261`;
- varargs are rejected with `CON0269` when the parser sees `...`;
- extern variables are not supported;
- `extern "C++"` and all other ABI strings are rejected with `CON026A`.

P15-M2/P15-M3 implementation status:

- valid extern C function declarations lower to HIR functions;
- HIR stores `is_extern`, `extern_abi = c`, the ABI span, and the C symbol name;
- the C symbol name currently equals the declared function name;
- extern functions have no body and are not compile-time or concept witness
  functions;
- extern declarations are visible to ordinary call resolution and normal HIR
  call type checking;
- empty extern blocks remain accepted;
- duplicate extern C symbols are rejected with `CON0265`;
- duplicate extern-vs-ordinary top-level names use the existing `CON0020`;
- `CON0259` is no longer emitted for valid extern declarations;
- P15-M3 removes the deferred MIR extern-call error for supported declarations;
- MIR calls distinguish Concept-internal callees from extern C callees;
- extern C calls lower through MIR with the HIR function id, C symbol name,
  result type, and lowered arguments;
- the MIR validator checks extern call linkage, argument count/type matching,
  result type metadata, and symbol presence against the HIR declaration;
- the C backend emits one plain prototype for each extern C declaration in the
  compiled unit;
- extern declarations do not produce MIR functions or generated C bodies;
- extern calls emit the declared C symbol name instead of `cpt_f_*` names;
- the `abs` smoke fixture runs through the hosted C toolchain with exit code 7;
- linker behavior, headers/includes, `export "C"`, and `repr(C)` remain
  deferred.

## 3. Source syntax: export "C" functions

Preferred long-term source syntax:

```cpp
export "C" int Add(int a, int b) {
    return a + b;
}
```

Rules:

- `export "C"` gives a Concept function C linkage.
- The exported function body is ordinary Concept code.
- The exported function name is emitted as a stable C symbol.
- Exported functions must use C ABI-compatible parameter and return types.
- No overloading for exported C names.
- No generic/template exported functions in v0.
- No concept witness functions exported in v0.
- No machine functions or dyn internals exported in v0.
- Exported functions still participate in ordinary Concept checking before
  backend emission.

If the parser makes `export "C"` expensive to stage, an attribute form may be
accepted temporarily:

```cpp
[Export(C)]
int Add(int a, int b) {
    return a + b;
}
```

The preferred long-term syntax remains `export "C"` because linkage is not an
ordinary annotation. It is part of the function's external identity.

## 4. Source syntax: repr(C) structs

Preferred long-term source syntax:

```cpp
repr(C)
struct Point {
    int x;
    int y;
}
```

Alternative staged attribute syntax:

```cpp
[Repr(C)]
struct Point {
    int x;
    int y;
}
```

P15-M5 implementation status: the staged attribute spelling `[Repr(C)]` is
implemented. The preferred keyword spelling `repr(C)` remains reserved and
deferred until it can be added without disrupting the parser. The semantic
marker is represented as `repr(C)` in HIR debug output.

Rules:

- `repr(C)` applies to structs only in v0.
- It promises C-compatible field order and layout for supported field types.
- Field order is declaration order.
- Structs must be non-empty unless C empty-struct behavior is explicitly
  designed.
- P15-M5 preserves the marker in AST/HIR and rejects invalid targets or
  unsupported repr arguments. Full C ABI field validation is deferred to M6.
- No `repr(C)` enums in v0.
- No packed layout in v0.
- No custom alignment in v0.
- No bitfields in v0.
- No layout randomization.
- No hidden fields.
- No inheritance or base layout.
- No dynamically sized fields.

## 5. C ABI-compatible type subset

Supported v0 C ABI types:

- `void` as a function return only;
- `int`;
- `bool`, lowered as C `int` under the existing backend convention;
- raw pointers to supported ABI types;
- raw pointers to opaque allocation handles, specifically `Arena*` and
  `Allocator*`;
- `repr(C)` structs whose fields are ABI-compatible.

Enum ABI is deferred. Even non-payload unit enums should wait until the enum
layout contract is explicitly designed as a user-visible ABI promise.

Rejected in v0:

- bare `Arena` or `Allocator` by value;
- bare interface types;
- dyn interface types;
- machine frame types;
- non-`repr(C)` structs across a C ABI boundary;
- payload enums across a C ABI boundary;
- `ManualInit<T>`;
- generic type parameters;
- `Result<T, E>` unless explicitly represented as `repr(C)` later;
- templates/generic functions;
- concept/interface impl functions;
- function pointers unless already supported and deliberately included;
- varargs.

ABI type validation happens before backend codegen. An unsupported ABI type
must produce a clear diagnostic, not guessed C emission.

P15-M2 implements a strict extern C declaration subset:

- return types: `void`, `int`, `bool`, `AllocError`, or raw pointers to
  supported pointee types;
- parameter types: `int`, `bool`, `AllocError`, or raw pointers to supported
  pointee types;
- supported raw pointer pointees: `void`, `int`, `bool`, `Arena`, `Allocator`,
  and `AllocError`;
- `void` is rejected as a parameter type;
- structs, struct pointers, enums, enum pointers, interfaces, dyn interfaces,
  machines, `ManualInit<T>`, type parameters, and generic extern functions are
  rejected until their ABI contracts are deliberately designed.

This means non-`repr(C)` structs are rejected both by value and by pointer in
M2. Opaque handle support is limited to the compiler-known allocation handles
`Arena*` and `Allocator*`, plus scalar/void pointer forms above.

Suggested diagnostic:

```text
CON0260 UnsupportedCAbiType
```

## 6. HIR/MIR representation

Possible HIR additions:

```text
ExternAbi {
    c
}

HirExternFunction {
    name
    c_symbol_name
    return_type
    params
    span
}

HirFunction {
    export_abi: ?ExternAbi
    export_symbol_name: ?Symbol
}
```

P15-M2 uses `HirFunction` entries with no body plus `is_extern`, ABI kind,
ABI span, and symbol-name metadata. The important rule is that body-less
foreign declarations must not be confused with ordinary incomplete Concept
functions.

HIR requirements:

- extern declarations are visible to normal call resolution;
- extern declarations are not required to have bodies;
- extern declarations do not trigger missing-body errors;
- P15-M3 lowers extern calls explicitly in MIR rather than as internal Concept
  calls;
- exported functions lower as ordinary functions with C symbol metadata;
- the `repr(C)` marker is stored on HIR structs.

MIR addition, or equivalent:

```text
MirFunction {
    linkage: internal | extern_c | export_c
    c_symbol_name
}
```

MIR requirements:

- MIR can distinguish Concept-internal calls from foreign C calls;
- backend emission can distinguish bodies from prototypes;
- the C backend can emit prototypes for extern C declarations;
- the C backend can emit exported C function names without Concept mangling.

## 7. Backend C behavior

For:

```cpp
extern "C" {
    int abs(int value);
}
```

Generated C should include:

```c
int abs(int value);
```

Calls should emit:

```c
abs(value)
```

For:

```cpp
export "C" int Add(int a, int b) {
    return a + b;
}
```

Generated C should emit:

```c
int Add(int a, int b) { ... }
```

It should not use the internal mangled name, such as `cpt_f_Add_0`, for the
exported C symbol. Non-exported Concept functions keep the existing backend
naming policy.

Backend rules:

- emit extern prototypes once;
- do not emit bodies for extern declarations;
- do not require headers;
- do not emit `#include` for user externs;
- do not link libraries;
- do not invent symbol names;
- diagnose unsupported ABI types before emission;
- preserve existing arena helper declarations as backend-owned helpers, not
  user extern declarations.

## 8. repr(C) backend layout

For:

```cpp
repr(C)
struct Point {
    int x;
    int y;
}
```

Generated C should use a deterministic C struct layout equivalent to:

```c
typedef struct {
    int cpt_f_x_0;
    int cpt_f_y_1;
} cpt_s_Point;
```

or the backend naming style current at implementation time.

Rules:

- field order is source declaration order;
- fields use C ABI-compatible types;
- no hidden padding fields are emitted by Concept;
- the C compiler handles natural padding and alignment;
- `repr(C)` promises compatibility with the C compiler's layout for the
  equivalent field sequence under the target ABI;
- layout validation rejects unsupported fields.

If the backend already emits structs, `repr(C)` is the point where layout
becomes a user-visible promise. Non-`repr(C)` structs may continue using
backend-local layout conventions.

## 9. Interaction with Phase 12 allocation helpers

Backend-owned arena helper declarations are not user `extern "C"`
declarations. The Phase 12 helper ABI remains a backend runtime contract:

```text
cpt_arena_alloc
cpt_arena_reset
cpt_arena_destroy
```

User extern declarations can declare C functions, but they do not replace
backend-owned helper emission. Future runtime profiles may provide these
helpers externally; that is a runtime/profile decision, not the P15 extern
surface.

## 10. Interaction with Phase 14 dyn

Dyn fat refs and vtables are Concept backend runtime representations, not
C ABI-stable user-facing types in Phase 15.

Rules:

- `dyn Interface&` is not C ABI-compatible in v0.
- Interface vtables are not exported as a stable C ABI in v0.
- Exporting or extern-declaring dyn interface types is rejected.
- Concept's backend-owned dyn wrappers, vtables, and fat refs remain internal
  lowering artifacts.

## 11. Name/linkage model

Rules:

- Concept internal functions keep current mangled/backend-local names.
- `extern "C"` declarations use declared C symbol names.
- `export "C"` functions use declared function names as C symbol names unless
  explicit symbol override is later designed.
- Duplicate C export symbols are rejected.
- Duplicate extern declarations with incompatible signatures are rejected.
- Duplicate extern declarations with identical signatures are rejected in v0.
- No explicit symbol aliasing in v0.

The simple v0 rule is: reject duplicate extern declarations by name in one
module, and reject duplicate export symbols by name in one module.

Future aliasing syntax, deferred:

```cpp
extern "C" symbol("puts") int C_Puts(char* s);
export "C" symbol("concept_add") int Add(int a, int b) { ... }
```

## 12. Diagnostics

Initial diagnostic planning list:

```text
CON0260 UnsupportedCAbiType
CON0261 ExternCRequiresFunctionDeclaration
CON0262 ExternCFunctionCannotHaveBody
CON0263 ExportCRequiresFunctionDefinition
CON0264 ExportCFunctionCannotBeGeneric
CON0265 DuplicateCAbiSymbol
CON0266 ReprCInvalidTarget
CON0267 ReprCUnsupportedFieldType
CON0268 ReprCEmptyStructUnsupported
CON0269 VarargsUnsupported
```

P15-M0 does not implement these diagnostics.

## 13. Non-goals for Phase 15 v0

Explicitly deferred:

- C header parser;
- bindgen;
- automatic `#include`;
- automatic library linking;
- build/linker driver;
- C++ interop;
- `extern "C++"`;
- C name aliasing or symbol attributes;
- varargs;
- function pointers unless already present and deliberately included;
- callbacks;
- `repr(packed)`;
- `align(n)`;
- bitfields;
- volatile/atomics/MMIO;
- stable ABI for payload enums;
- `repr(C)` enums;
- dyn/interface ABI stability;
- machine frame ABI stability;
- cross-module ABI visibility;
- import/multi-module compilation;
- platform ABI matrix beyond the current C backend target.

## 14. Milestone plan

```text
P15-M0  Design doc: C ABI, extern declarations, and repr(C)
P15-M1  extern "C" parser/AST scaffold
P15-M2  extern C HIR declarations and ABI type validation
P15-M3  extern C call lowering and backend prototype emission
P15-M4  export "C" function surface (implemented)
P15-M5  repr(C) struct staged attribute and HIR marker (implemented)
P15-M6  repr(C) field/type validation and backend layout hardening
P15-M7  C ABI diagnostics and symbol/linkage hardening
P15-M8  examples/fixtures: extern calls, exports, repr(C) structs
P15-M9  Closeout
```

## 15. Closeout criteria

Phase 15 should close only when:

- `extern "C"` function declarations parse, lower, type-check, and call through
  the real MIR-backed C path;
- `export "C"` functions emit stable C names with ordinary Concept body
  checking;
- `repr(C)` structs carry a HIR marker and validate ABI-compatible fields;
- unsupported ABI types fail before backend emission with clear diagnostics;
- generated C shows extern prototypes, unmangled exports, and deterministic
  `repr(C)` struct layouts;
- tests and fixtures cover accepted and rejected extern/export/repr(C) paths;
- docs clearly state that C++ interop, bindgen, headers, linker driving,
  varargs, packed layout, and enum ABI remain deferred.
