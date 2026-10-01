# R9a Stage-0 closure

Ordinary integral scalar representations may index storage, spans, strings and
tensors. Signed indices check negativity; all compare against extent at uint64
width before pointer-offset use, without narrowing to int or size_t first. Unknown
bounds retain the existing Planner guard. Fixed known bounds reject wide unsigned
out-of-range values rather than reading their unused signed evaluator field.
Verify reports unsigned indices at full width. Subspan's offset/length API and
raw-storage initialization operations keep their own existing contracts.

The literal suffix `u` defaults to canonical `uint` (32 bits) and contextually
types into unsigned representations when representable. Negative unsigned
literals and implicit signed targets reject; explicit checked casts remain.
`~` complements the operand representation width, returns that same scalar type,
and rejects bool/floating/quantity operands. `operator~` uses ordinary required
operation witnesses, including artifact-only generic instances.

Named rounding intrinsics accept dependent source/target types in open templates
and check their representations when each instance closes. They preserve the
explicit rounding choice and Result error model; this grants no optimizer facts.
Floating-to-integer `as` remains rejected. Record field defaults are not admitted
by this change; named initialization retains the established source expression
order and missing-field diagnostic.

CV4138 now belongs to the innate field concept
`MovableFieldDoesNotEmbedImmovable`. Its predicate projects immovability and
raw/sparse storage from existing semantic observations, preserves the former
Go diagnostic and field site, and attaches the ordinary innate proof graph.
Shadow and switch agreement covered every registered valid/invalid specimen
and directed field cases before the Go implementation was deleted. Closed
generics loaded through semantic artifacts are subject to the same rule.

R9a removes Stage-0 debt before frontend self-hosting. It does not freeze Stage-0
or begin R9b. Owner decisions supersede undecided alternatives in the October ledger.

## Structural identity

Type.Application and StructDecl.Application retain the defining module/declaration
and ordered tagged type/typed integer arguments. Nested applications stay structural;
aliases use normal canonical resolution. Cache keys exclude locations, import markers,
and nested display names. Generic C symbols carry a structural digest, preventing
Buffer<double>/Buffer<Double> collisions. Bare symbols acquire metadata from their
declarations, never by parsing display spelling. The concept-module.v1 envelope now
requires concept-generic-application.v1; old artifacts must be rebuilt. Inspectable
application summaries must match the typed payload. No legacy parser fallback exists.
Self-hosted frontend stages consume structured generic application identity; they do
not port Stage-0 string reparsing.

## Context and admission

Inherited scopes retain generic environment and evaluation mode. Expressions enter
and restore context; argument, binding, and assignable helpers reuse it during
revalidation. Old public flag signatures remain during this bounded migration.
Comptime function parameter/result types resolve before call checking. Array indices
use the caller's mode, replacing an erroneous hard-coded comptime path. Runtime-only
push, comptime-only loop bounds, and comptime function body entry remain explicit.
Comptime array-element assignment still rejects CV4213; mutation/alias semantics are
outside this context repair.

Unsupported unsafe/imported type modifiers reject; unsafe asm and module imports
retain their meanings. Conflicting/repeated qualifiers reject instead of overwriting.
Scoped validation precedes nominal early returns, preserving scoped callable/ref/dyn
behavior. Compiler-owned attributes reject on unsupported field/case positions;
user selector metadata remains observable through reflection/generators. Type attribute
validation is shared by check, code generation, and artifact production.

## Ownership and ABI

Generic substitution may change whether ownership has runtime destruction work,
but it does not change the authored ownership relationship. owned T value owns every
instantiation: trivial T needs no Drop; droppable T receives ordinary exact-once Drop.
CV4653 remains the innate DroppableFieldIsOwned rule.

Generic repr(C) record declarations store repr policy. Each closed instance satisfies
the ordinary innate C value rule; open declarations claim no concrete ABI layout.
Foreign boundaries retain existing native-probe requirements. Named initializers already
evaluate in source expression order and reject duplicate, unknown, and missing fields.
Field defaults remain unsupported.
