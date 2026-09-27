# Postfix compiler pipeline

The parser turns a fixed token stream into a payload-enum AST held by
`DenseStore`; expression references are stable `Id<Expr>` values. The
first natural draft remains in `first-draft.concept.txt`; the executable
checks are in `postfix.concept_test`.

Friction: C-style `for` gets a teaching diagnostic. Payload enums have no
fixed Span element geometry, so the token stream uses fixed `TokenWord`
records while AST nodes remain payload values. `?` propagates recoverable
errors. Familiar: stack parser and dense traversal. New: typed IDs, Results,
and exact borrow scopes. Advanced: effect proof.
