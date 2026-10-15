# SMLib Phase 1 compiler hardening

The unchanged external port at `../SMlib_Concept/phase1` is the regression
baseline: 239 cases and 478 Normal/Verify differential comparisons. Original
numerical algorithms, C++ oracle, attribution, and known-defect records remain
unchanged. Evidence logs are under ignored `artifacts/smlib-hardening`.

## Pass A: callback correctness

Callback return inference now participates in ordinary statement validation:
preceding locals are bound before return expressions are typed, branches share
one inferred result, and ordinary mutation, move, capture, and escape checks
remain active. There is no preliminary value-expression-only scan or second
ownership-consuming inference pass.

An erased callback reference records its semantically resolved concrete identity
and environment. Typed expression traversal covers nested constructs, closures,
loops, failure unwrap/propagation, and assertions. MIR witness collection and C
adapter emission consume one identity-deduplicated collection. Identical
signatures do not merge distinct callbacks or shadowed capture environments.
C emission no longer guesses identity by searching source variable names.

Four checked-in positive fixtures and negative safety tests supplement the
existing callable capture/types conformance corpus. They execute strict C11
in Normal and Verify, check deterministic artifacts, and verify one adapter per
semantic identity. The unchanged external port's three native facts pass in
both modes. No loop escape, C-style loop, automata, scheduler, or runtime frame
semantics were changed.

## Pass B: numerical ergonomics

Expected float representation reaches literal arithmetic and conditionals without
converting typed leaves. Unique parameter declarations supply context before
literal range checking. Compile-time evaluation now honors resolved floating
representations. Binary64 Standard.Math overloads call the corresponding double
C11 operations; known allocation-free effects require exact, dimensionless native
signatures. Numeric domain and nonfinite cases execute through the actual library.

Unqualified value-query builtins are fallback names; ordinary declarations,
imports, namespace-qualified functions, and lexical callables retain identity in
semantic analysis, MIR classification, constant evaluation, and C emission.
Structured storage/machine operations and compiler observations keep their laws.

Named-field construction already checked completeness and duplicates. Its local
lowering now uses field identity while retaining source evaluation order. Explicit
T{} zero construction is restricted to complete recursively defaultable plain
data, using ordinary typed aggregate initialization. Owners, borrows, Drop-bearing
values, class/ref/immovable/table/bits aggregates, attributed/default-initializer
fields, and unsupported storage reject. General default expressions, per-field
omission/defaults, constructors, and implicit initialization remain deferred.

Native Standard numerical facts cover double precision, positive/negative zero,
infinities, NaNs, domain results, primitive operations, typed context, named/zero
construction, and NoAllocation. Negative tests retain mixed-precision and lifetime
rejection. New semantics are documented explicitly in the language specification.

## Pass C: artifact and module tooling

`emit-c --out-dir` saves the existing complete deterministic output set while
retaining stdout-only use. `--module-root` and explicit `--source-modules` align
checking, source-built emission, and test dependency resolution without implicit
declaration discovery. Physical aliases of roots are normalized consistently by
source and artifact loaders; distinct identities still conflict. Package and
native-companion manifests retain ownership of their dependency/ABI graphs.
See ../tooling/SMLIB-ARTIFACTS-AND-MODULES.md for precise mode and path rules.

The separate yield-return RFC recommends deferral. No implementation, prototype,
break/continue, C-style for, state-machine change, hidden frame, or scheduling
mechanism is included. See ../design/SMLIB-YIELD-RETURN-RFC.md.

## Additional closure lifetime regression

A negative Result-payload test exposed a pre-existing erased-callback escape:
`return Result::Ok(ref localCallable)` could be admitted. Erasure now borrows
concrete environment storage even when its captured values have static provenance.
Erased callback pairs participate in the existing lifetime-bound type rules,
including Result/Option propagation and captured borrowed values. Forwarding a
borrowed parameter remains valid and executes in both policies; local-environment
escapes reject. Already-erased typed expressions (including Result unwrap) reuse
their callback pair and do not manufacture another adapter.

Callback descriptor types now participate in the existing runtime header dependency
graph, so Result/aggregate fields see the required descriptor before use. This
fixes valid forwarding without C declaration-order workarounds or runtime machinery.
