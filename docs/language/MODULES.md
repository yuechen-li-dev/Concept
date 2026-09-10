# Reusable semantic modules

Status: EVT1 R6e implemented compiler substrate

## Syntax and authority

```concept
module Standard.Generic;
profile Core;

template <typename T>
struct Box { T value; };
```

```concept
module App;
profile Core;
import Standard.Generic;
```

`module` gives the unit a semantic name; it is not a filesystem identity.
`import` consumes a compiled `concept-module.v1` artifact. It never performs
text inclusion and normal consumer compilation never opens imported source.
Source is authoritative while building a module; the integrity-checked
artifact is authoritative while compiling a consumer.

R6e uses one intentionally simple visibility rule: top-level declarations in
a module are import-visible. Class `public`/`private` member rules remain
unchanged. Imports are not namespace re-exports, although artifact dependency
closure remains available to resolve semantic references. Alias imports,
partitions, and C++ header compatibility are not supported.

## Artifact

`concept build-module path/to/Generic.concept` writes deterministic JSON to
stdout. The envelope records schema/compiler identity, module/portable source
identity, source/content SHA-256, sorted dependencies, exports, normalized
signatures, structural type summaries, and the operation-summary lattice. Its
payload is a compiler-private, backend-independent encoding of typed syntax,
including generic bodies, constraints, ownership, Drop, provenance, concepts,
interfaces, source spans, and implementation dependencies.

The artifact contains neither C nor object code. Imported generic declarations
are instantiated in the consumer compilation and then follow the ordinary
layout, ownership, provenance, MIR, Planner, and backend pipeline. There are no
runtime dictionaries, reflection tables, or module loader.

## Resolution

Semantic path mapping is exact:

```text
Standard.Generic -> <root>/Standard/Generic.concept-module.json
```

The importing file's directory is the first CLI root. Additional explicit
roots come from the platform path-list environment variable
`CONCEPT_MODULE_ROOTS`. Resolution does not scan other directories or access a
network. Multiple matches are errors. Missing artifacts, cycles, incompatible
compiler/schema versions, corrupt hashes, mismatched dependency hashes, and
stale adjacent `.concept` sources are rejected deterministically.

`.concept_test` discovery has a bounded source-build mode. It resolves the
same exact mapping with `.concept`, topologically compiles the dependency DAG
to in-memory semantic artifacts, and then compiles the test as an artifact
consumer. This is build ordering, not textual inclusion.

## Cross-module semantics

Exported generic structs, classes, functions, methods, integer/`usize`
non-type parameters, `using`, `typeof`, `SizeOf<T>()`, and `AlignOf<T>()` retain
their existing meanings. A declaration constraint is serialized and checked
against the consumer's concrete local type; failure is explicit, not SFINAE.
Applied fields feed existing copy/move/Drop and `ref struct` provenance law.

Operation summaries are import authority. A proven pure wrapper supplies
`NoAllocation`; a wrapper reaching an admitted allocating operation supplies
`Allocates`; an opaque operation without a summary remains `Unknown`. R6b proof
graphs label the boundary `ModuleSummaryEffect`.

R6g unit-qualified types, `Address<Space>`, `Storage<T>`, and ordered
multi-parameter function templates use the same typed payload. Consumer
instantiation includes every argument in deterministic identity; unit metadata
does not become an ABI object.

R6j artifacts also retain explicitly bound foreign semantic declarations and
their `DeclaredForeign` effect/value-summary origins. Consumers use the checked
artifact without reparsing the companion source; missing authority remains
Unknown. Exact declaration syntax is provisional.

R6l serializes Initialized certainty in ordinary bounded value-fact summaries
and permits open-template `bind<T>`. Operational field state remains an analysis
channel rather than runtime or proof metadata. R6m subsequently resolved nested
generic owner extraction and imported `Storage<T>` substitution, enabling
cross-module typed-owner factories.

Within an interface, `requires compiler.Allocates(Operation);` means the
operation is permitted to allocate. An allocation-free implementation is also
compatible. An allocating implementation cannot satisfy an operation lacking
that allowance. This is a bounded variance rule, not a generalized effect
algebra or a user-authored negative promise.

R6e deliberately provides no package manager, registry, remote fetch, lockfile,
stable binary module ABI, import aliases, partial specialization, variadics,
module partitions, or runtime module system.

R6o generalizes serialized generic constraints from one first-parameter target
to structural concept applications. The typed payload retains concept parameter
lists, all application arguments, nested prerequisite declarations,
required-operation signatures, and effect requirements. Imported constrained
functions and generic types build their operation closure and instantiate from
that payload without source reparse; closed instance MIR contains concrete
arguments and concrete witness bindings.

R6p uses this path for `Standard.Memory`. Its artifacts carry allocator concept
composition, required Allocate/Release signatures, effect summaries, generic
typed-owner bodies, and region/provenance summaries through the ordinary
schema. An artifact-only consumer instantiates bump, pool, `Allocation<T,A>`,
typed Allocate, and constrained Drop without opening module source.
