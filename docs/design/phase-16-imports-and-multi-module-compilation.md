# Phase 16: Imports and multi-module compilation

P16-M0 is a documentation-only milestone. It defines Concept's first explicit
multi-module compilation model: source files declare modules, imports build a
module graph, qualified names cross module boundaries, and the existing HIR,
MIR, and backend paths become module-aware. Phase 16 does not implement compiler
behavior in M0 and does not introduce packages, dependency resolution, linker
behavior, visibility, or filesystem-based import lookup.

## Core doctrine

```text
Modules are compilation-unit boundaries, not packages.

A source file belongs to exactly one module.

Imports make modules visible by name.

Imports do not paste declarations into local scope in v0.

Qualified access is the v0 import model.

The compiler resolves the module graph before semantic lowering.

Module names are unique inside one compilation unit.

Import cycles are rejected in v0.

The filesystem is not the module system in v0.

The test/build harness supplies the source files in v0.

No package manager, dependency resolver, linker driver, or visibility system in Phase 16.
```

## 1. Motivation

Concept has grown beyond single-file toy compilation. Phases 12 through 15 added
explicit allocation and stores, machines, runtime interfaces and borrowed dyn
dispatch, and an explicit single-compilation-unit C ABI. Those features are
useful in isolation, but real compiler, runtime, kernel, bare-metal, and
high-performance native programs need more than one source file.

C ABI boundaries, runtime interfaces, machines, ID stores, fixture coverage, and
future self-hosting all need a coherent way to organize multiple Concept source
files into one program. Phase 16 should make multiple Concept files compile as
one compilation unit with a real, inspectable module graph.

The goal is not a package manager. The goal is not a build system. The goal is
the first real module pipeline:

```text
source files -> module declarations -> import graph -> module-aware HIR -> MIR/backend
```

Imports must be explicit and inspectable. They must not become textual
inclusion, ambient namespace pollution, or Python/JavaScript-style path chaos.
The v0 rule is deliberately plain: the harness or driver supplies the files;
each file names its module; imports name modules; qualified access crosses the
resolved graph.

## 2. Source file model

Phase 16 v0 source-file rules:

- one source file declares exactly one module;
- `module Name;` must appear at the top of the file before imports and
  declarations;
- module declaration is required for all multi-module compilation units;
- module name is unique within the compilation unit;
- a compilation unit consists of a set of source files provided by the
  harness/driver;
- v0 does not search the filesystem to resolve imports;
- v0 does not infer module names from paths;
- v0 does not allow multiple modules per file;
- v0 does not allow one module to span multiple files.

Example:

```cpp
module Math;

int Add(int a, int b) {
    return a + b;
}
```

Single-file compatibility remains important. Current single-file mode should be
modeled as a one-source-file, one-module compilation unit rather than as a
separate language. Future work may permit a module to span multiple files, but
that is explicitly deferred because it complicates duplicate-item rules,
diagnostics, and incremental ordering before the first graph exists.

## 3. Import syntax

Phase 16 v0 import syntax:

```cpp
module Main;

import Math;
import Compiler.Lexer;

int main() {
    return Math.Add(20, 22);
}
```

Rules:

- imports appear after the `module` declaration and before other top-level
  declarations;
- import target is a module path, not a string path;
- no string imports in v0;
- no wildcard imports;
- no aliases;
- no import lists;
- no re-export;
- no conditional imports;
- no import from C headers;
- duplicate imports of the same module are rejected with a clear diagnostic.

Rejecting duplicate imports keeps the graph simple and makes accidental fixture
or driver duplication visible early. Idempotent duplicate imports can be
considered later if a larger build model needs it.

## 4. Qualified access model

`import Math;` makes the module name `Math` available as a qualified root.
Imported names are accessed with module-qualified syntax:

```cpp
Math.Add(1, 2)
```

Imports do not make `Add` directly visible as an unqualified name. Unqualified
lookup remains local/current-module only. Qualified access may refer to imported
top-level functions, structs, enums, concepts, interfaces, machines, and other
supported top-level items.

Field access and module-qualified access both use `.`, but resolution
separates them by receiver kind:

- value expression receiver -> field/method access;
- imported module root -> module-qualified item access.

In expression position, `Math.Add` resolves to the top-level function `Add` in
imported module `Math`. In type position, `Math.Point` resolves to the top-level
type `Point` in imported module `Math`.

Imported enum references should follow the existing enum-qualified syntax after
the enum type itself is resolved. The recommended v0 spelling is therefore:

```cpp
Math.Color::Red
```

where `Math.Color` resolves the imported enum type and `::Red` selects the
variant using the established enum-variant qualification model. If the existing
implementation only supports a different enum-qualified spelling, Phase 16
should adapt to that existing rule rather than inventing unqualified `using`
behavior.

## 5. Module graph resolution

The intended pipeline is:

```text
parse all source files
collect module declarations
build module table
collect imports
resolve import edges
reject duplicate modules
reject unknown imports
reject duplicate imports
reject import cycles
lower modules to HIR in dependency order or with module-aware symbol tables
resolve qualified names across imported modules
lower whole compilation unit to MIR/backend
```

Rules:

- module graph is resolved before semantic lowering that depends on imports;
- unknown imports are diagnosed before deep semantic work where possible;
- import cycles are rejected in v0;
- source locations in diagnostics should point to the import declaration or
  module declaration that caused the issue.

The implementation may lower in dependency order or build module-aware symbol
tables before resolving bodies. The design requirement is not a specific data
structure; it is that import edges are known before cross-module names are
accepted.

## 6. Multi-file fixture/test harness model

Phase 16 cannot be tested well with single-file fixtures only. Multi-file
fixtures must be hermetic: the fixture itself provides every virtual source file
in the compilation unit.

Recommended embedded fixture format for `.conception` files:

```text
# name: cross-module add
# phase: run
# expect: pass

=== file: Math.concept ===
module Math;

int Add(int a, int b) {
    return a + b;
}

=== file: Main.concept ===
module Main;

import Math;

int main() {
    return Math.Add(20, 22);
}

=== run ===
exit_code: 42
```

Requirements:

- multi-file fixtures are hermetic;
- a fixture can contain multiple source files;
- each source file has a stable virtual path/name for diagnostics;
- the test harness passes all source files to one compilation unit;
- diagnostics include file/source spans for the correct virtual file;
- backend/run fixtures can compile the whole unit into one generated C output.

Actual sidecar files are not required for v0 unless the existing harness already
supports them cleanly. Embedded virtual files keep language fixtures reviewable
and avoid making filesystem layout part of module semantics.

## 7. HIR model

HIR must become module-aware. One possible shape:

```text
HirModule {
    id
    name
    source_file_id
    imports
    items
    symbol_table
}

HirItem {
    module_id
    item_kind
}
```

Requirements:

- HIR store can represent multiple modules;
- top-level items know their defining module;
- each module has a top-level symbol table;
- import declarations are preserved or represented as resolved import edges;
- duplicate names are checked within a module;
- duplicate module names are checked across the compilation unit;
- cross-module lookup goes through resolved imports;
- current single-file mode remains a one-module compilation unit.

Existing global HIR stores should not assume that top-level item names are
unique across the whole program. Function IDs, type IDs, and item IDs should
remain the canonical references after resolution.

## 8. Name resolution rules

Lookup order and context rules:

- unqualified names resolve in current module only;
- qualified module roots resolve among imported modules and possibly the current
  module by name;
- imported modules do not inject unqualified declarations;
- local variables/params still shadow ordinary unqualified names within
  expressions;
- module names are not values;
- module-qualified roots are only valid in qualified access positions;
- unknown module-qualified roots produce a module/import diagnostic;
- known module root + unknown item produces a qualified item diagnostic.

Valid:

```cpp
module Main;

import Math;

int main() {
    return Math.Add(20, 22); // valid
}
```

Invalid:

```cpp
module Main;

import Math;

int main() {
    return Add(20, 22); // invalid in v0: import does not inject Add
}
```

Invalid:

```cpp
module Main;

int main() {
    return Math.Add(20, 22); // invalid: Math not imported
}
```

Invalid:

```cpp
module Main;

import Math;

int main() {
    return Math.Missing(1); // invalid: imported module has no Missing
}
```

## 9. Qualified types

Qualified type references are part of v0.

```cpp
module Geometry;

[Repr(C)]
struct Point {
    int x;
    int y;
}
```

```cpp
module Main;

import Geometry;

int Sum(Geometry.Point p) {
    return p.x + p.y;
}
```

Rules:

- type resolver can resolve `Geometry.Point`;
- qualified type roots must be imported modules;
- non-type items in type position are rejected with existing or new diagnostics;
- imported structs, enums, interfaces, concepts, and machines can be referenced
  by qualified type names if already supported as ordinary types;
- C ABI validation sees imported repr(C) metadata correctly.

Qualified type support is required for useful C ABI and runtime-interface use
across modules. Function-only imports would be too narrow for real systems code.

## 10. Cross-module functions and calls

Example:

```cpp
module Math;

int Add(int a, int b) {
    return a + b;
}
```

```cpp
module Main;

import Math;

int main() {
    return Math.Add(20, 22);
}
```

Rules:

- function call resolver can resolve module-qualified function names;
- HIR call references the defining function id across modules;
- MIR lowering supports cross-module call function ids;
- backend emits both functions into one generated C unit for v0;
- ordinary internal backend naming remains deterministic and collision-free
  across modules.

Backend function internal names must avoid collisions between same function
names in different modules. If current backend names are per-function unique by
id, this may already be true. Module-qualified Concept symbols must not collide
in generated C even if different modules define the same local item name.

## 11. Cross-module top-level duplicate rules

Rules:

- duplicate names within the same module are rejected;
- identical item names in different modules are allowed;
- duplicate module names are rejected;
- local variables may shadow ordinary unqualified lookup, but they do not shadow
  imported module-root qualified lookup;
- module roots are only considered in qualified lookup contexts;
- a module root conflict with a top-level item in the current module is rejected
  in v0.

The final rule keeps the model simple: if `module Main` defines a top-level item
named `Math` and imports module `Math`, then `Math.X` would be too ambiguous for
an early implementation. Reject the conflict with a precise diagnostic rather
than relying on context-sensitive surprises.

## 12. Import cycle detection

Import cycles are rejected in v0.

```cpp
// A.concept
module A;
import B;
```

```cpp
// B.concept
module B;
import A;
```

The diagnostic should point to the import that completes or participates in the
cycle. Suggested diagnostic:

```text
CON0272 ImportCycle
```

No cyclic modules in v0. Allowing cycles requires a much more careful story for
partial symbol availability, initialization ordering, interface/concept
coherence, and diagnostics.

## 13. Visibility model

Phase 16 v0 does not introduce public/private visibility.

Rules:

- all top-level declarations in an imported module are accessible by qualified
  access in v0;
- this is a temporary v0 model;
- a future visibility phase may add `public`, `private`, or module-internal
  controls;
- do not use `export` for module visibility because `export "C"` already means
  C ABI export.

```text
export "C" is C linkage, not module visibility.
```

## 14. Interaction with Phase 15 C ABI

Phase 16 does not add linker behavior. Multi-module Concept compilation still
produces one generated C unit in v0 unless the backend already supports another
safe path.

Rules and interactions:

- `extern "C"` declarations may appear in any module;
- `export "C"` functions may appear in any module;
- duplicate C ABI symbols across the whole compilation unit must remain
  rejected;
- repr(C) metadata must be visible across modules for C ABI validation;
- headers/includes/linker driver remain deferred.

Module names do not change C ABI symbol names. An `export "C" int Add(...)` in
module `Math` still exports the C symbol `Add`, and duplicate exported or extern
C symbols anywhere in the compilation unit remain an error.

## 15. Interaction with tests

`.con_test` files from Phase 11 may eventually import ordinary modules. Phase 16
v0 may allow test modules to import implementation modules if the test harness
supports multi-file input.

Test discovery across multiple files is deferred unless it falls out naturally
from the multi-file harness. Phase 16 should not overbuild test package
discovery, project discovery, or workspace behavior. A hermetic fixture that
contains one test source plus one implementation source is enough to prove the
module graph.

## 16. Interaction with machines/interfaces/templates

Imported modules may define concepts, templates, interfaces, and machines if
those already exist as top-level items. Phase 16 v0 should support qualified
access to top-level declarations where current semantics allow it.

No new cross-module interface coherence or orphan rules are introduced beyond
same-compilation-unit checking. There is no package-level coherence yet. There
is no separate compilation ABI yet. Template instantiation, interface impl
selection, and machine lowering should use resolved item IDs rather than textual
module paths after name resolution.

## 17. Diagnostics planning

Suggested diagnostic names:

```text
CON0270 DuplicateModule
CON0271 UnknownImport
CON0272 ImportCycle
CON0273 ImportMustAppearBeforeDeclarations
CON0274 ModuleQualifiedNameUnknown
CON0275 ModuleQualifiedNameNotImported
CON0276 ModuleDeclarationRequired
CON0277 DuplicateImport
CON0278 MultipleModulesInFile
CON0279 DuplicateModuleItem
```

Use exact names later based on implementation. The important part is diagnostic
shape: report the failing module declaration, import declaration, or qualified
access span, and keep unknown-import failures early in the pipeline.

## 18. Non-goals for Phase 16 v0

Explicitly deferred:

- package manager;
- dependency resolver;
- semantic versions;
- package manifests;
- filesystem import search paths;
- import aliases;
- wildcard imports;
- unqualified using imports;
- re-exports;
- public/private visibility;
- friend/internal visibility;
- cyclic modules;
- incremental compilation;
- separate object files;
- linker driver;
- cross-package dependencies;
- module spanning multiple files;
- multiple modules in one file;
- import from C headers;
- automatic C header includes;
- build profile integration;
- language server/project model.

## 19. Milestone plan

```text
P16-M0  Design doc: imports and multi-module compilation
P16-M1  Multi-file fixture/test harness scaffold
P16-M2  Module declaration table and duplicate module diagnostics
P16-M3  Import declaration parser/AST and ordering diagnostics
P16-M4  Import graph resolution: unknown imports, duplicate imports, cycles
P16-M5  Module-aware HIR store and top-level symbol tables
P16-M6  Qualified module name resolution for functions and values
P16-M7  Qualified type resolution and cross-module type use
P16-M8  Multi-module MIR/backend lowering and examples/fixtures
P16-M9  Closeout
```

## P16-M1 implementation status: multi-file fixture scaffold

P16-M1 adds the hermetic fixture harness scaffold needed before real module
semantics. A `.conception` fixture can now embed one or more virtual Concept
source files with `=== file: <virtual-path> ===` sections. The fixture parser
stores these as an ordered source set, preserves each virtual path for parser
diagnostics and future source IDs, and continues to map legacy `=== source ===`
fixtures to a one-source case with no fixture changes.

Implemented M1 behavior:

- embedded multi-file `.conception` source sections are supported by the fixture
  parser;
- virtual file paths are preserved in fixture source records;
- source order is deterministic and follows fixture order;
- duplicate virtual file paths are rejected as fixture-format errors;
- missing/empty virtual file paths are rejected as fixture-format errors;
- parser-only multi-file fixtures parse each virtual source independently;
- existing single-file fixtures, run fixtures, backend fixtures, and snapshot
  sections remain unchanged.

Deferred after M1:

- import syntax implementation;
- module table construction;
- duplicate module diagnostics;
- import resolution and import-cycle rejection;
- module-aware HIR/MIR/backend lowering;
- cross-module qualified lookup;
- package management, filesystem search paths, linker-driver behavior, and
  visibility.

Semantic, run, MIR, and backend fixtures remain single-source until the Phase 16
module graph exists. The harness must not silently drop extra virtual files.

## P16-M2 implementation status: module table scaffold

P16-M2 adds the compilation-unit module inventory pass used by later import
graph milestones. Multi-source parser fixtures now parse every virtual source,
collect each file's top-level `module` declaration, and build an ordered module
table that preserves stable module IDs, module names, virtual source paths,
source indexes, and declaration spans.

Implemented M2 behavior:

- multi-source parser fixtures build a module table after all virtual sources
  parse;
- exactly one module declaration is required for each source file in
  multi-source fixtures;
- duplicate module names are rejected across one compilation unit;
- one module cannot span multiple files in v0;
- multiple module declarations in one source file remain rejected by the parser;
- dotted module names supported by the existing parser are preserved in the
  table;
- existing single-source fixtures keep their historical behavior.

Diagnostics added or used by M2:

- `CON0270 DuplicateModule` for duplicate module names in distinct virtual
  source files;
- `CON0276 ModuleDeclarationRequired` when a multi-source file has no module
  declaration;
- the existing parser duplicate-module diagnostic continues to reject multiple
  module declarations inside one file.

Still deferred after M2:

- import syntax and import declarations in the AST;
- import graph resolution, unknown import diagnostics, duplicate imports, and
  cycle detection;
- module-aware HIR item ownership beyond this table;
- qualified cross-module name lookup;
- semantic, run, MIR, and backend multi-source compilation.

## P16-M3 implementation status: import parser/AST and ordering diagnostics

P16-M3 implements the parser-only import declaration surface needed before graph
resolution. The lexer reserves `import`, the parser accepts module-path imports
after the file `module` declaration and before all ordinary top-level
declarations, and the AST preserves import declaration order, dotted path
components, declaration spans, and path spans. Stable AST debug output now emits
imports in source order after the module declaration.

Implemented M3 behavior:

- `import` is a keyword token;
- v0 source syntax is `import Qualified.Module.Name;` with a module path, not a
  string or filesystem path;
- imports are parsed only after `module` and before non-import top-level items;
- dotted import paths are preserved in the AST;
- AST debug output shows `Import <path>` entries in source order;
- imports after ordinary declarations are rejected with
  `CON0273 ImportMustAppearBeforeDeclarations`;
- imports before the module declaration continue to use the existing parser
  module-first diagnostic path;
- unsupported string, wildcard, alias, list, and re-export import forms are
  rejected syntactically;
- the module table preserves raw import declarations per module in source order,
  including path text and spans, without resolving them.

Still deferred after M3:

- unknown import diagnostics (`CON0271`);
- duplicate import diagnostics (`CON0277`);
- import cycle diagnostics (`CON0272`);
- module graph resolution;
- module-aware HIR item ownership;
- qualified cross-module lookup for values, functions, and types;
- cross-module lowering, backend emission, filesystem lookup, package management,
  re-export, aliases, wildcard imports, and visibility.

## P16-M4 implementation status: import graph resolution

Phase 16 M4 resolves the raw import records collected in M3 against the module
table produced in M2. Each `ModuleUnit` now preserves `resolved_imports` in
source order as stable `ModuleId` edges while retaining raw import text and spans
for diagnostics and later qualified lookup work.

Implemented M4 behavior:

- unknown imports are rejected with `CON0271 UnknownImport` before semantic
  lowering or cross-module name resolution;
- duplicate imports within one module are rejected with `CON0277 DuplicateImport`,
  including dotted import paths;
- duplicate imports in different modules remain valid because visibility is per
  importer module;
- deterministic DFS cycle detection rejects self-imports, direct cycles, and
  longer cycles with `CON0272 ImportCycle`;
- import graph resolution runs only after structural module-table collection
  succeeds, so duplicate module names and missing module declarations remain the
  earlier diagnostics;
- multi-source parser/module-table fixtures now exercise graph resolution; valid
  multi-source import fixtures must supply every imported module in the same
  fixture compilation unit.

Still deferred after M4:

- qualified lookup such as `Math.Add` or `Math.Type`;
- unqualified imported names;
- cross-module semantic resolution, HIR ownership, MIR lowering, backend
  lowering, and executable multi-module compilation;
- import aliases, wildcard imports, re-exports, visibility, packages, filesystem
  module search, and dependency resolution.

## P16-M5 implementation status: module-aware HIR scaffold

P16-M5 adds the first module-aware HIR architecture layer. The HIR store now has
module records corresponding to resolved `ModuleUnit`s, preserving module name,
source index, virtual source path, module declaration span, resolved import
edges, and item order. Top-level HIR item IDs are associated with their defining
HIR module so later qualified lookup can map declarations back to module roots.

Semantic lowering now has a multi-source, HIR-only path for fixtures: the module
table is built and imports are resolved first, HIR modules are created in module
table/source order, and each source file lowers into its own current module.
Top-level duplicate checking is therefore per module for ordinary Concept items:
`Math.Add` and `Main.Add` may both exist. Duplicate ordinary names inside the
same module still use the existing duplicate top-level diagnostic.

C ABI symbol checking intentionally remains compilation-unit-wide. Duplicate
`extern "C"` or `export "C"` symbols across different modules still diagnose as
`CON0265`; Phase 16 has not introduced C symbol aliases, package/linkage scopes,
or per-module C namespaces.

Still deferred:

- P16-M6 owns qualified module lookup such as `Math.Add`;
- P16-M7 owns qualified cross-module type lookup such as `Geometry.Point`;
- imports still do not inject unqualified names;
- visibility, aliases, wildcard imports, re-exports, package management, and
  filesystem import search remain out of scope;
- multi-source MIR/backend/run lowering remains deferred to P16-M8.

## P16-M6 implementation status: qualified module function lookup

P16-M6 implements expression-position qualified module function lookup for the semantic/HIR fixture path. `Module.Function(...)` resolves the left root against the current module or the current module's resolved imports, then resolves the right-hand name against the target module's top-level HIR items. Calls preserve the defining cross-module `HirFunctionId`, so modules may define the same function name and callers can select the intended function with qualification.

Imports still do not inject unqualified names: `import Math;` permits `Math.Add(...)`, but a bare `Add(...)` in another module remains an ordinary unknown-function error. Known but non-imported module roots are rejected with `CON0275`; unknown module roots and missing qualified items are rejected with `CON0274`. Qualified current-module calls such as `Main.Helper()` are accepted.

Qualified type references such as `Geometry.Point`, cross-module type use, and multi-source MIR/backend/run lowering remain deferred to P16-M7 and P16-M8 respectively.
