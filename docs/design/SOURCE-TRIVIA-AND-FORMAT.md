# R8e3 source trivia and formatting

`concept format` changes presentation. `concept lint` checks semantic and
project policy. Renaming is an explicit semantic refactor, never an implicit
formatter action. `concept lint --fix` is not implemented.

The semantic parser still consumes ordinary tokens. `ParseSourceDocument`
retains an ordered token tape with byte offsets, source spans, and the exact
leading trivia gap for each token, plus trailing trivia at EOF. A comment is
anchored by the two adjacent token ordinals: a trailing line comment remains
after the preceding token; a leading comment remains before the next token.
The tape is separate from semantic ASTs, MIR, proofs, and generated names.

Formatting validates syntax before printing. It prints tokens in original
order, retains comment bytes, checks that re-lexing yields the same lexemes,
then parses the formatted result. A failure leaves the file untouched. A
project invocation prepares all edits before writing, skips nested projects
with their own `manifest.concept`, requires a root manifest, skips symlinks,
and never selects native `.c` or `.cpp`.

The default style uses four spaces, same-line braces, spacing around ordinary
operators, one statement per line, and a preferred 100-column width. The
width is guidance: the printer breaks after a comma when the line is already
long, and does not split literals or identifiers. It preserves existing blank
line boundaries. Allman braces can be selected by the project manifest.

The root `manifest.concept` may contain these ordinary immutable declarations:

```concept
comptime int FormatIndentWidth = 4;
comptime int FormatMaxLineLength = 100;
comptime string FormatBraceStyle = "same-line";
```

The brace value may also be `"allman"`. Width must be at least 40 and indent
width 1 through 16. Bad values receive `FORMAT_CONFIG_INVALID` rather than a
silent fallback. Omit all three declarations to use the defaults.

Canonical authored names are `snake_case` for files and attributes,
`camelCase` for locals, parameters, and fields, and `PascalCase` for types,
structs, records, enums, functions, methods, concepts, interfaces, and
machines. `concept format` preserves identifier spellings, including foreign
linkage names and generated identifiers. Foreign symbol spelling is preserved
for linkage/API identity; Concept-facing wrappers or aliases may use canonical
Concept names independently. Existing `concept lint` naming policy supplies
diagnostics and deterministic suggestions (`parse_http_request` →
`ParseHttpRequest`, `gpu_buffer` → `gpuBuffer` for a camelCase subject). A
bound-reference rename engine remains future work.

MIR and generated C can include source coordinates for diagnostics and checked
operations, so their raw bytes may change after formatting. Compare MIR after
excluding refreshed source spans. Generated C is byte-identical for the
qualified span-free case; checked arithmetic C embeds updated coordinates.
