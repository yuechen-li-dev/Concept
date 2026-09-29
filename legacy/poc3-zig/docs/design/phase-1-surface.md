# Phase 1 surface syntax

Phase 1 defines the source forms that the Stage 0 compiler can lex and parse into a shallow AST. It is a corpus and parser-surface milestone, not an execution milestone.

## Source files

- Normal source files use the `.concept` extension.
- Test source files use the `.con_test` extension.
- Stage 0 has no user-facing header files. Declarations live directly in source files.
- Stage 0 expects one `module` declaration at the start of each file.

## Supported declarations

- `import` declarations name other modules with dotted names.
- `struct` declarations contain field declarations with parsed type names.
- Payload `enum` declarations contain variants, optionally with typed payload fields.
- `concept`, `interface`, and `impl` declarations contain signature-only member declarations.
- Top-level function declarations support parsed signatures and either a trailing semicolon or a shallow body.
- Attributes use bracket syntax such as `[Fact]`, `[Theory]`, and `[InlineData(...)]` and can be attached to supported items.

## Function bodies

Function bodies are captured shallowly. The parser balances braces so it can resume parsing later top-level items, but it does not parse the statements or expressions inside the body.

## Intentionally unsupported in Phase 1

Phase 1 intentionally does not support:

- statement parsing
- expression parsing
- type checking
- name resolution
- execution
- backend generation
