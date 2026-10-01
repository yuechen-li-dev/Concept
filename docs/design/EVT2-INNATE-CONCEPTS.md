# EVT2 innate concepts (MVP design)

Status: proposed.

> A concept is something that must be true and can be validated. Innate
> concepts are the ones the compiler holds before it reads any program: no
> program can omit, weaken, or redefine them. They are written in Concept and
> ship with the compiler.

Innate concepts are how `validate.go` moves into Concept piece by piece, one
rule at a time, with the Go rule and the Concept rule run side by side until
they agree on the whole corpus.

## Terms

- **Innate concept**: declared in the compiler's innate module, applied by the
  compiler to every declaration of its kind. Innate concepts *establish*
  truth: they are part of what a valid program is.
- **Declared concept**: any other concept. Declared concepts *require* truth
  that the compiler establishes; they cannot create it. This is the existing
  rule in `PROJECT-POLICY.md` ("policy may require truth, policy cannot create
  truth"), which now follows from the definitions.
- **Observation**: a compiler-provided, compile-time query over the checked
  semantic model ("the type of field F", "does this type have a Drop
  witness"). Observations project; they never judge. The judgment, which is the
  rule, is Concept code.

What moves is the rule. What stays in Go is a small, closed list of
observations. Each observation is a projection of state the compiler already
computes. A rule that needs an observation that is not on the list is not
ready to move.

## What already exists

Most of the MVP is composition of shipped machinery:

| Need | Existing piece |
| --- | --- |
| Concepts over declarations | `concept C<declaration D>` (DECLARATION-CONCEPT-PARAMETERS.md) |
| Declaration subjects, including fields | `DeclarationSubjects(module)`; kinds include `FieldDeclaration`, `TypeDeclaration`, `FunctionDeclaration` |
| Three-valued proof with evidence | proof graph (`concept-proof.v1`), Proven / Disproven / Unknown |
| Compile-time execution | bounded `comptime` evaluator with fuel, call depth, structs, enums, fixed arrays, `while ... bounded` |
| Explanation | `concept explain --concept`, `--policy` |
| Applying concepts to every matching declaration | `concept lint` subject selection by kind (project_policy.go) |

## What is missing

1. A requirement that runs Concept code. Today every `requires` on a
   declaration concept ends in a Go analysis (`compiler.NoAllocation(D)`).
2. Compile-time values that stand for declarations and types, plus the
   observations over them.
3. Statement forms in the `comptime` evaluator: it has `while`, `match`, and
   assignment, but no `if` statement, no `for (i in a..b)`, and no string
   concatenation.
4. A way to say "apply this concept to every field declaration, mandatorily".

## Surface

### The innate module

`internal/concept/innate/Innate.concept`, `module Innate;`, embedded in the
compiler with `go:embed`. The compiler compiles it once per process, as the
Vulkan profile does with its library today. Its source hash is part of the
compiler identity and is recorded in every `concept-module.v1` artifact; an
artifact built under a different innate set is rejected as stale.

`innate` is admitted only in this module (`INNATE_OUTSIDE_COMPILER`).

### Innate concepts

```concept
[[diagnostic("CV4653")]]
innate concept DroppableFieldIsOwned<FieldDeclaration F>
{
    requires DroppableFieldIsOwnedHolds(F);
}
```

- The parameter category is a declaration kind (`FieldDeclaration`,
  `TypeDeclaration`, `FunctionDeclaration`, …), a narrowing of the existing
  `declaration` category. The kind is the applicability rule: the concept
  applies to every declaration of that kind, and to nothing else.
- `[[diagnostic("CODE")]]` gives the rule a stable code, so the corpus,
  tooling, and the Go rule it replaces share one identity.
- An innate concept is an ordinary concept: a declared concept may
  `requires DroppableFieldIsOwned<declaration F>;` and gets Proven for every
  accepted program.

### Predicates

A requirement may call a `comptime` function whose result is `Verdict`:

```concept
enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}

comptime Verdict DroppableFieldIsOwnedHolds(declaration F)
{
    typename t = compiler.TypeOf(F);
    if (compiler.Owned(F) or compiler.BorrowLike(t) or not compiler.HasDrop(t))
    {
        return Verdict::Holds;
    }
    string type = compiler.TypeName(t);
    return Verdict::Refuted(F, "field " + compiler.QualifiedName(F) + " holds " + type +
        ", which has a Drop; declare it `owned " + type + " " + compiler.Name(F) + ";`");
}
```

`Holds` maps to Proven. `Refuted` maps to Disproven, with the message as
evidence and `at`'s site as the diagnostic span. Fuel exhaustion, an
unsupported observation, or an evaluator error map to Unknown, which rejects
the program as `INNATE_UNDECIDED`. That outcome is a compiler defect, never a
user error, and the diagnostic says so.

### Subject values and observations

`declaration` and `typename`, both existing keywords, become compile-time-only
opaque value types. They can be passed, stored in locals, and compared with
`==`, and they are only inspected through observations. They cannot reach
runtime code, MIR, or C.

MVP observation list (closed, versioned with the innate module):

| Observation | Result |
| --- | --- |
| `compiler.Kind(declaration)` | `DeclarationKind` enum |
| `compiler.Name(declaration)`, `compiler.QualifiedName(declaration)` | `string` |
| `compiler.Provenance(declaration)` | `Authored`, `Generated`, `Foreign` |
| `compiler.TypeOf(declaration)` | field, parameter, or local type |
| `compiler.Owned(declaration)` | `bool`: the declaration carries `owned` |
| `compiler.FieldCount(declaration)`, `compiler.Field(declaration, int)` | a type declaration's fields, in order |
| `compiler.HasAttribute(declaration, string name, string argument)` | `bool` |
| `compiler.IsRecord`, `IsClass`, `IsRefStruct` (`declaration`) | `bool` |
| `compiler.TypeName(typename)` | canonical source spelling |
| `compiler.TypeKind(typename)` | `Scalar`, `Handle`, `Struct`, `Enum`, `Array`, `Pointer`, `Borrow`, `Callable`, `Dyn`, `Async`, `Parameter` |
| `compiler.BorrowLike(typename)`, `HasDrop(typename)`, `HasTypeArguments(typename)` | `bool` |
| `compiler.Element(typename)`, `RuntimeShape(typename)` | array element; runtime extent |
| `compiler.Declaration(typename)` | the struct or enum declaration of a nominal type |

`HasDrop` is the one observation that consults a witness. It reports whether a
`Drop` operation exists, which the compiler already resolves. It does not
decide whether anything is right.

### Evaluator additions

These are general `comptime` improvements, not innate-only features:

- `if` / `else` statements;
- `for (i in a..b)`, with the bound counted against fuel;
- `string + string`;
- the two opaque subject value kinds.

## Semantics

**When.** After the Go validator accepts a module, before MIR. The innate
concepts therefore observe a well-formed semantic model. A Go check whose
violation would corrupt that model cannot move: duplicate fields (CV4124),
unresolved names, and layout recursion stay ahead of the model. That is the
boundary of this mechanism, and it is principled: innate concepts constrain
programs, and the model's own integrity is a precondition for observing them.

**Where.** Every declaration in `ProjectDeclarationSubjects(module)` whose kind
matches. Closed generic instances are included: `Box<Pipeline>` is checked
with its substituted field types. Imported declarations are not re-checked;
their artifact records the innate set hash under which they passed.

**Order.** Declarations in source order, innate concepts in declaration order
within the innate module. The first Refuted verdict is the diagnostic, which
matches today's first-error behavior.

**Narrowing only in the MVP.** Innate concepts can reject programs; their
Proven results are not consumed by any analysis that licenses code generation
or optimization. Fact-granting innate concepts, such as moving `NoAllocation`
into Concept, are later work under the same trust rules.

**Explanation.** Each evaluation is a proof-graph node: concept, subject,
predicate, verdict, message, site. `concept explain file:line` on a field shows
the innate concepts that applied to it.

## Strangler protocol

Each Go rule moves in three commits:

1. **Shadow.** The innate concept lands. Both rules run; the Go rule is
   authoritative. An agreement test runs both over the corpus: every valid
   file holds everywhere, and every invalid file that the Go rule rejects
   produces the same code at the same span from the innate concept.
2. **Switch.** The innate concept is authoritative, and the Go rule runs in
   the agreement test only.
3. **Delete.** The Go rule is removed.

## Pilots

1. **CV4653, droppable fields are owned.** Per field, no iteration, a corpus
   case already exists, and the Go rule is about fifteen lines. It also
   settles generic instances: `Box<T> { T value; }` instantiated as
   `Box<Inner>` must be checked on its substituted fields.
2. **C_ABI_REPR_INVALID, `[[repr(C)]]` aggregates.** Per type declaration,
   with a field loop and a recursive walk over field types (array elements,
   nested repr(C) structs). It proves that iteration, recursion under fuel,
   attributes, and type observations are enough for a structural rule.

If both pilots read better in Concept than in Go, and the agreement test is
clean, the mechanism is ready for the rest of the declarative layer.

## Ladder

| Step | Content |
| --- | --- |
| IC0 | Make `concept check` agree with `emit-c` on generic instances (see Findings) |
| IC1 | `comptime` `if`, `for`, string `+`; tests |
| IC2 | `declaration` / `typename` subject values; observation list; `Verdict` |
| IC3 | Embedded innate module, `innate concept`, kind-narrowed parameters, `[[diagnostic]]`, artifact hash |
| IC4 | Application engine, proof-graph nodes, `explain`, `INNATE_UNDECIDED` |
| IC5 | CV4653: shadow, switch, delete |
| IC6 | C_ABI_REPR_INVALID: shadow, switch, delete; spec section |

## Not in the MVP

- fact-granting innate concepts (`NoAllocation`, `Outlives`);
- body-level analyses (types, moves, borrows, lifetimes), which move as
  ordinary Concept passes through the bottom-up strangler, not as concepts;
- user-written innate concepts;
- `or` / `not` in `requires` (propositional composition is useful but caps
  quickly, and every rule written that way still needs a new Go analysis);
- runtime reflection.

## Findings from the design pass

- **`check` and `emit-c` disagree on generic instances.** `struct Box<T> { T
  value; }` instantiated with a droppable type passes `concept check` but is
  rejected by `concept emit-c` with CV4653. The gate is weaker than the code
  generator. This is the same accepted-but-inert class as CV4653 itself, so IC0
  fixes it before innate concepts depend on instance checking.
- **Ownership of generic fields is a language question.** Under CV4653, a
  generic container must write `owned T value;` (which works, and `Box<int>`
  still compiles). The alternative is that structural Drop releases every field
  whose type has a Drop, owned or not, and CV4653 disappears. The MVP keeps
  CV4653 as the pilot either way, but the language should choose deliberately.
