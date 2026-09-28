package concept

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceToken is an ordinal syntax anchor. Leading contains the exact bytes
// between this token and its predecessor, including comments. Neither it nor
// SourceDocument participates in semantic ASTs, proofs, or artifact hashes.
type SourceToken struct {
	Token
	Start   int
	End     int
	Leading string
}

type SourceDocument struct {
	Tokens   []SourceToken
	Trailing string
}

func ParseSourceDocument(source string) (SourceDocument, error) {
	tokens, err := lexEVT1(source)
	if err != nil {
		return SourceDocument{}, err
	}
	lines := []int{0}
	for i, c := range []byte(source) {
		if c == '\n' {
			lines = append(lines, i+1)
		}
	}
	doc := SourceDocument{Tokens: make([]SourceToken, 0, len(tokens))}
	end := 0
	for _, token := range tokens {
		start := lines[token.Span.Line-1] + token.Span.Column - 1
		if start < end || start+len(token.Lexeme) > len(source) || source[start:start+len(token.Lexeme)] != token.Lexeme {
			return SourceDocument{}, fmt.Errorf("FORMAT_SOURCE_SPAN_INVALID: %d:%d", token.Span.Line, token.Span.Column)
		}
		doc.Tokens = append(doc.Tokens, SourceToken{Token: token, Start: start, End: start + len(token.Lexeme), Leading: source[end:start]})
		end = start + len(token.Lexeme)
	}
	doc.Trailing = source[end:]
	return doc, nil
}

type FormatOptions struct {
	IndentWidth   int
	MaxLineLength int
	BraceStyle    string
}

func DefaultFormatOptions() FormatOptions {
	return FormatOptions{IndentWidth: 4, MaxLineLength: 100, BraceStyle: "same-line"}
}

// FormatSource accepts only syntax the current parser understands. The final
// token check prevents whitespace edits from joining or splitting lexemes.
func FormatSource(path, source string, options FormatOptions) (string, error) {
	if options.IndentWidth < 1 || options.IndentWidth > 16 || options.MaxLineLength < 40 || options.BraceStyle != "same-line" && options.BraceStyle != "allman" {
		return "", fmt.Errorf("FORMAT_CONFIG_INVALID: invalid indent width, maximum line length, or brace style")
	}
	if _, err := parseSyntaxModule(path, source); err != nil {
		return "", err
	}
	doc, err := ParseSourceDocument(source)
	if err != nil {
		return "", err
	}
	printer := sourcePrinter{options: options}
	compactEnd, compactKind := -1, ""
	for i, token := range doc.Tokens {
		prev := ""
		if i > 0 {
			prev = doc.Tokens[i-1].Lexeme
		}
		next := ""
		if i+1 < len(doc.Tokens) {
			next = doc.Tokens[i+1].Lexeme
		}
		if i > compactEnd {
			printer.trivia(token.Leading)
		}
		if token.Lexeme == "{" && options.BraceStyle == "same-line" {
			compactKind, compactEnd = evt1CompactBrace(doc.Tokens, i, options.MaxLineLength-printer.lineWidth())
		}
		compact := compactEnd >= i
		if printer.pendingBreak {
			if !compact || i != compactEnd {
				printer.newline()
			}
			printer.pendingBreak = false
		}
		if token.Lexeme == "}" {
			if printer.depth > 0 {
				printer.depth--
			}
			if prev != "{" && !compact {
				printer.newline()
			}
		}
		if token.Lexeme == "{" && options.BraceStyle == "allman" && prev != "" && prev != "{" {
			printer.newline()
		}
		if !(compact && compactKind == "aggregate" && (token.Lexeme == "{" || prev == "{" || token.Lexeme == "}")) {
			printer.spaceBefore(token.Lexeme, prev, token.Leading)
		}
		printer.write(token.Lexeme)
		switch token.Lexeme {
		case "{":
			printer.depth++
			if next != "}" && !compact {
				printer.newline()
			}
		case ";":
			printer.pendingBreak = true
		case ",":
			if printer.lineWidth() >= options.MaxLineLength {
				printer.newline()
			}
		case "}":
			if compact && i == compactEnd {
				compactEnd, compactKind = -1, ""
			}
			if next != ";" && next != "," && next != ")" && next != "]" && next != "else" && next != "except" && next != "{" {
				printer.pendingBreak = true
			}
		}
	}
	printer.trivia(doc.Trailing)
	result := strings.TrimRight(printer.buf.String(), " \t\r\n") + "\n"
	check, err := lexEVT1(result)
	if err != nil {
		return "", fmt.Errorf("FORMAT_OUTPUT_INVALID: %w", err)
	}
	if len(check) != len(doc.Tokens) {
		return "", fmt.Errorf("FORMAT_TOKEN_CHANGED: token count %d -> %d", len(doc.Tokens), len(check))
	}
	for i := range check {
		if check[i].Lexeme != doc.Tokens[i].Lexeme {
			return "", fmt.Errorf("FORMAT_TOKEN_CHANGED: token %d %q -> %q", i, doc.Tokens[i].Lexeme, check[i].Lexeme)
		}
	}
	if _, err := parseSyntaxModule(path, result); err != nil {
		return "", fmt.Errorf("FORMAT_OUTPUT_INVALID: %w", err)
	}
	return result, nil
}

// Compact only comment-free, flat forms whose full spelling fits the line.
// The token boundary is deliberately narrow: declarations and nested control
// flow keep the ordinary multiline layout.
func evt1CompactBrace(tokens []SourceToken, open, available int) (string, int) {
	if open == 0 || available < 8 {
		return "", -1
	}
	close := -1
	for i := open + 1; i < len(tokens); i++ {
		if tokens[i].Lexeme == "{" {
			return "", -1
		}
		if tokens[i].Lexeme == "}" {
			close = i
			break
		}
		if strings.Contains(tokens[i].Leading, "//") || strings.Contains(tokens[i].Leading, "/*") {
			return "", -1
		}
	}
	if close <= open+1 {
		return "", -1
	}
	prev := tokens[open-1].Lexeme
	kind := ""
	if prev == "else" || prev == "=>" || prev == ")" && evt1BraceFollowsIf(tokens, open) {
		kind = "simple"
	} else if isIdentifier(prev) || prev == ">" {
		declaration := false
		if open >= 2 {
			switch tokens[open-2].Lexeme {
			case "struct", "class", "enum", "concept", "interface", "automata", "layout", "machine", "table", "namespace", "module":
				declaration = true
			}
		}
		if !declaration {
			kind = "aggregate"
		}
	}
	if kind == "" {
		return "", -1
	}
	semicolons, width := 0, 2
	for i := open + 1; i < close; i++ {
		lexeme := tokens[i].Lexeme
		if lexeme == ";" {
			semicolons++
		}
		if lexeme == "=>" {
			return "", -1
		}
		width += len(lexeme) + 1
	}
	if kind == "aggregate" && semicolons != 0 {
		return "", -1
	}
	if kind == "simple" && (semicolons != 1 || tokens[close-1].Lexeme != ";" || tokens[open+1].Lexeme != "return") {
		return "", -1
	}
	if width > available {
		return "", -1
	}
	return kind, close
}

func evt1BraceFollowsIf(tokens []SourceToken, open int) bool {
	depth := 0
	for i := open - 1; i >= 0; i-- {
		switch tokens[i].Lexeme {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				return i > 0 && tokens[i-1].Lexeme == "if"
			}
		}
	}
	return false
}

type sourcePrinter struct {
	buf          bytes.Buffer
	options      FormatOptions
	depth        int
	line         bool
	pendingBreak bool
}

func (p *sourcePrinter) write(s string) {
	if !p.line {
		p.buf.WriteString(strings.Repeat(" ", p.depth*p.options.IndentWidth))
		p.line = true
	}
	p.buf.WriteString(s)
}

func (p *sourcePrinter) newline() {
	if p.line {
		p.buf.WriteByte('\n')
		p.line = false
	}
}

func (p *sourcePrinter) lineWidth() int {
	content := p.buf.Bytes()
	last := bytes.LastIndexByte(content, '\n')
	return len(content) - last - 1
}

func (p *sourcePrinter) spaceBefore(current, previous, original string) {
	if !p.line || p.buf.Len() == 0 || previous == "" {
		return
	}
	last := p.buf.Bytes()[p.buf.Len()-1]
	if last == ' ' || last == '\n' {
		return
	}
	if (current == "++" || current == "--") && (isIdentifier(previous) || previous == "]" || previous == ")") {
		return
	}
	if current == ";" || current == "," || current == ":" || current == ")" || current == "]" || current == "." || current == "::" || current == "(" && previous != "if" && previous != "while" && previous != "for" && previous != "match" || current == "[" || previous == "(" || previous == "[" || previous == "." || previous == "::" || previous == "@" {
		return
	}
	if original == "" && len(previous) > 0 && previous[0] >= '0' && previous[0] <= '9' && len(current) > 0 && (current[0] >= 'a' && current[0] <= 'z' || current[0] >= 'A' && current[0] <= 'Z') {
		return // scientific unit suffix
	}
	// Keep authored adjacency at grammar-sensitive boundaries (generics,
	// units, symbolic indices, and unary operators).
	if current == "<" || current == ">" || previous == "<" || previous == ">" || current == "!" || previous == "!" || current == "-" && previous == "(" {
		if original == "" {
			return
		}
	}
	p.buf.WriteByte(' ')
}

// trivia prints comments from their original token gap. An end-of-line
// comment stays after the preceding syntax; a leading comment stays before
// the following token. Blank-line boundaries remain visible.
func (p *sourcePrinter) trivia(raw string) {
	for len(raw) > 0 {
		if strings.HasPrefix(raw, "//") {
			end := strings.IndexByte(raw, '\n')
			if end < 0 {
				end = len(raw)
			}
			if p.line && p.buf.Len() > 0 && p.buf.Bytes()[p.buf.Len()-1] != ' ' {
				p.buf.WriteByte(' ')
			}
			p.write(strings.TrimSuffix(raw[:end], "\r"))
			p.newline()
			raw = raw[end:]
			continue
		}
		if strings.HasPrefix(raw, "/*") {
			end := strings.Index(raw, "*/") + 2
			if end < 2 {
				return
			}
			if p.line {
				p.buf.WriteByte(' ')
			}
			p.write(raw[:end])
			if strings.Contains(raw[:end], "\n") {
				p.line = true
			}
			raw = raw[end:]
			continue
		}
		if raw[0] == '\n' {
			p.newline()
			if len(raw) > 1 && raw[1] == '\n' && p.buf.Len() > 0 && p.buf.Bytes()[p.buf.Len()-1] == '\n' {
				p.buf.WriteByte('\n')
			}
			raw = raw[1:]
			continue
		}
		if raw[0] == ' ' || raw[0] == '\r' || raw[0] == '\t' {
			raw = raw[1:]
			continue
		}
		// The lexer has already validated trivia. This guard makes an
		// unexpected gap fail closed in FormatSource's token check.
		raw = raw[1:]
	}
}

// LoadFormatOptions reads literal, typed settings from the root manifest.
// These are ordinary Concept declarations, not a second manifest grammar.
func LoadFormatOptions(manifestPath string) (FormatOptions, error) {
	options := DefaultFormatOptions()
	body, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return options, nil
	}
	if err != nil {
		return options, err
	}
	module, err := parseSyntaxModule(filepath.ToSlash(manifestPath), string(body))
	if err != nil {
		return options, err
	}
	seen := map[string]bool{}
	for _, decl := range module.ComptimeDecls {
		if !strings.HasPrefix(decl.Name, "Format") {
			continue
		}
		if seen[decl.Name] {
			return options, fmt.Errorf("FORMAT_CONFIG_INVALID: duplicate %s", decl.Name)
		}
		seen[decl.Name] = true
		switch decl.Name {
		case "FormatIndentWidth", "FormatMaxLineLength":
			literal, ok := decl.Value.(*IntLiteral)
			if !ok || decl.Type.Name != "int" {
				return options, fmt.Errorf("FORMAT_CONFIG_INVALID: %s must be a comptime int literal", decl.Name)
			}
			value, ok := literal.boundedInt()
			if !ok {
				return options, fmt.Errorf("FORMAT_CONFIG_INVALID: %s is out of bounds", decl.Name)
			}
			if decl.Name == "FormatIndentWidth" {
				options.IndentWidth = value
			} else {
				options.MaxLineLength = value
			}
		case "FormatBraceStyle":
			literal, ok := decl.Value.(*StringLiteral)
			if !ok || decl.Type.Name != "string" {
				return options, fmt.Errorf("FORMAT_CONFIG_INVALID: FormatBraceStyle must be a comptime string literal")
			}
			options.BraceStyle = literal.Value
		default:
			// Other Format-prefixed declarations may belong to the project.
		}
	}
	if options.IndentWidth < 1 || options.IndentWidth > 16 || options.MaxLineLength < 40 || options.BraceStyle != "same-line" && options.BraceStyle != "allman" {
		return options, fmt.Errorf("FORMAT_CONFIG_INVALID: expected indent width 1..16, max line length >=40, brace style same-line or allman")
	}
	return options, nil
}

// FormatPath returns proposed edits so the CLI can validate all files before
// writing any. Nested projects with their own manifests are separate owners.
func FormatPath(target string) (map[string]string, error) {
	linkInfo, err := os.Lstat(target)
	if err != nil {
		return nil, err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("FORMAT_SOURCE_INVALID: symlink target %s", target)
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(target)
	if info.IsDir() {
		root = target
		if _, err := os.Stat(filepath.Join(root, "manifest.concept")); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("FORMAT_MANIFEST_MISSING: %s", filepath.Join(root, "manifest.concept"))
			}
			return nil, err
		}
	}
	options, err := LoadFormatOptions(filepath.Join(root, "manifest.concept"))
	if err != nil {
		return nil, err
	}
	paths := []string{target}
	if info.IsDir() {
		paths = nil
		err = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				if path != target {
					if _, err := os.Stat(filepath.Join(path, "manifest.concept")); err == nil {
						return filepath.SkipDir
					}
				}
				if entry.Name() == ".git" || entry.Name() == "artifacts" || entry.Name() == ".native-build" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".concept") || strings.HasSuffix(entry.Name(), ".concept_test") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	edits := map[string]string{}
	for _, path := range paths {
		if !strings.HasSuffix(path, ".concept") && !strings.HasSuffix(path, ".concept_test") {
			return nil, fmt.Errorf("FORMAT_SOURCE_INVALID: %s", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		formatted, err := FormatSource(filepath.ToSlash(path), string(body), options)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if formatted != string(body) {
			edits[path] = formatted
		}
	}
	return edits, nil
}
