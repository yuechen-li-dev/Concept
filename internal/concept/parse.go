package concept

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type parser struct {
	path               string
	tokens             []Token
	pos                int
	profileDef         *ProfileDefinition
	callableOrdinal    int
	templateTypeParams map[string]bool
	inGeneratorBody    bool
	suppressAs         bool
}

func Parse(path, text string) (Module, error) {
	module, err := parseSyntaxModule(path, text)
	if err != nil {
		return Module{}, err
	}
	if err := evt1MaterializeGeneratedDeclarations(&module); err != nil {
		return Module{}, err
	}
	env, err := analyzeModule(module)
	if err != nil {
		return Module{}, err
	}
	if err := evt1BuildReflectionResults(&module, env); err != nil {
		return Module{}, err
	}
	evt1MaterializeGenericInstances(&module, env)
	evt1MaterializeGenericProofSummaries(&module, env)
	evt1ApplyExactCallableTypes(&module, env)
	return module, nil
}

func parseSyntaxModule(path, text string) (Module, error) {
	tokens, err := lexEVT1(text)
	if err != nil {
		return Module{}, err
	}
	p := &parser{path: filepath.ToSlash(path), tokens: tokens}
	module, err := p.parseModule()
	if err != nil {
		return Module{}, err
	}
	for index := range module.Functions {
		module.Functions[index].Module = module.Name
	}
	for index := range module.ComptimeFns {
		module.ComptimeFns[index].Module = module.Name
	}
	for index := range module.GenericTypes {
		module.GenericTypes[index].Module = module.Name
		module.GenericTypes[index].Struct.Module = module.Name
	}
	for index := range module.Structs {
		module.Structs[index].Module = module.Name
	}
	for index := range module.Enums {
		module.Enums[index].Module = module.Name
	}
	for index := range module.Generators {
		module.Generators[index].Module = module.Name
	}
	for index := range module.Concepts {
		module.Concepts[index].Module = module.Name
	}
	for index := range module.Templates {
		module.Templates[index].Module = module.Name
	}
	for index := range module.Automata {
		module.Automata[index].Module = module.Name
	}
	return module, nil
}

func lexEVT1(text string) ([]Token, error) {
	var tokens []Token
	line, column := 1, 1
	for i := 0; i < len(text); {
		c := text[i]
		if c == '\n' {
			line++
			column = 1
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			column++
			i++
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '/' {
			for i < len(text) && text[i] != '\n' {
				i++
				column++
			}
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '*' {
			start := Span{Line: line, Column: column}
			i += 2
			column += 2
			closed := false
			for i < len(text) {
				if i+1 < len(text) && text[i] == '*' && text[i+1] == '/' {
					i += 2
					column += 2
					closed = true
					break
				}
				if text[i] == '\n' {
					line++
					column = 1
				} else {
					column++
				}
				i++
			}
			if !closed {
				return nil, evt1Diagnostic("CV4000", "unterminated block comment", start)
			}
			continue
		}
		start := Span{Line: line, Column: column}
		switch {
		case i+2 < len(text) && text[i:i+3] == "...":
			tokens = append(tokens, Token{Lexeme: "...", Span: start})
			i += 3
			column += 3
		case i+1 < len(text) && text[i:i+2] == "..":
			tokens = append(tokens, Token{Lexeme: "..", Span: start})
			i += 2
			column += 2
		case i+2 < len(text) && (text[i:i+3] == "<<=" || text[i:i+3] == ">>="):
			tokens = append(tokens, Token{Lexeme: text[i : i+3], Span: start})
			i += 3
			column += 3
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_':
			j := i + 1
			for j < len(text) {
				d := text[j]
				if (d >= 'a' && d <= 'z') || (d >= 'A' && d <= 'Z') || (d >= '0' && d <= '9') || d == '_' {
					j++
					continue
				}
				break
			}
			tokens = append(tokens, Token{Lexeme: text[i:j], Span: start})
			column += j - i
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			if c == '0' && j < len(text) && (text[j] == 'x' || text[j] == 'X') {
				j++
				for j < len(text) && ((text[j] >= '0' && text[j] <= '9') || (text[j] >= 'a' && text[j] <= 'f') || (text[j] >= 'A' && text[j] <= 'F')) {
					j++
				}
			} else {
				for j < len(text) && text[j] >= '0' && text[j] <= '9' {
					j++
				}
				if j < len(text) && text[j] == '.' && j+1 < len(text) && text[j+1] >= '0' && text[j+1] <= '9' {
					j++
					for j < len(text) && text[j] >= '0' && text[j] <= '9' {
						j++
					}
				}
				if j < len(text) && (text[j] == 'e' || text[j] == 'E') {
					k := j + 1
					if k < len(text) && (text[k] == '+' || text[k] == '-') {
						k++
					}
					if k < len(text) && text[k] >= '0' && text[k] <= '9' {
						for k < len(text) && text[k] >= '0' && text[k] <= '9' {
							k++
						}
						j = k
					}
				}
			}
			if j < len(text) && text[j] == 'u' && isNumber(text[i:j]) {
				j++
			}
			tokens = append(tokens, Token{Lexeme: text[i:j], Span: start})
			column += j - i
			i = j
		case c == '"' && i+1 < len(text):
			j := i + 1
			for j < len(text) && text[j] != '"' {
				if text[j] == '\n' {
					return nil, evt1Diagnostic("CV4000", "unterminated string literal", start)
				}
				j++
			}
			if j >= len(text) {
				return nil, evt1Diagnostic("CV4000", "unterminated string literal", start)
			}
			j++
			tokens = append(tokens, Token{Lexeme: text[i:j], Span: start})
			column += j - i
			i = j
		case i+1 < len(text) && text[i:i+2] == "::":
			tokens = append(tokens, Token{Lexeme: "::", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "..":
			tokens = append(tokens, Token{Lexeme: "..", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "!=":
			tokens = append(tokens, Token{Lexeme: "!=", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "==":
			tokens = append(tokens, Token{Lexeme: "==", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "<=":
			tokens = append(tokens, Token{Lexeme: "<=", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == ">=":
			tokens = append(tokens, Token{Lexeme: ">=", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "=>":
			tokens = append(tokens, Token{Lexeme: "=>", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "->":
			tokens = append(tokens, Token{Lexeme: "->", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "<<":
			tokens = append(tokens, Token{Lexeme: "<<", Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && strings.Contains(" += -= *= /= %= &= |= ^= ++ -- ", " "+text[i:i+2]+" "):
			tokens = append(tokens, Token{Lexeme: text[i : i+2], Span: start})
			i += 2
			column += 2
		case i+1 < len(text) && text[i:i+2] == "&&":
			return nil, evt1Diagnostic("CV4648", "use 'and' instead of '&&'", start)
		case i+1 < len(text) && text[i:i+2] == "||":
			return nil, evt1Diagnostic("CV4648", "use 'or' instead of '||'", start)
		case strings.ContainsRune("(){}[];,:.*+-/=<>!?@%^&|~", rune(c)):
			tokens = append(tokens, Token{Lexeme: string(c), Span: start})
			i++
			column++
		default:
			return nil, evt1Diagnostic("CV4000", fmt.Sprintf("invalid token %q", c), start)
		}
	}
	return tokens, nil
}

func (p *parser) parseModule() (Module, error) {
	module := Module{Path: p.path}
	if p.peekLexeme() == "module" {
		name, err := p.parseModuleNameDecl()
		if err != nil {
			return module, err
		}
		module.Name = name
	}
	if err := p.expectKeyword("profile"); err != nil {
		return module, err
	}
	profile, err := p.expectIdentifier("CV4001", "expected profile name")
	if err != nil {
		return module, err
	}
	profileDef, ok := evt1ProfileDefinition(profile.Lexeme)
	if !ok {
		return module, evt1Diagnostic("CV4001", "expected `profile Core;` or `profile Vulkan;`", profile.Span)
	}
	module.Profile = profile.Lexeme
	p.profileDef = profileDef
	if _, err := p.expect(";"); err != nil {
		return module, err
	}
	if module.Name == "" && p.peekLexeme() == "module" {
		name, err := p.parseModuleNameDecl()
		if err != nil {
			return module, err
		}
		module.Name = name
	}
	for p.peekLexeme() == "import" {
		p.next()
		var parts []string
		for {
			tok, err := p.expectIdentifier("CV4002", "expected import path")
			if err != nil {
				return module, err
			}
			parts = append(parts, tok.Lexeme)
			if p.peekLexeme() != "." {
				break
			}
			p.next()
		}
		if _, err := p.expect(";"); err != nil {
			return module, err
		}
		module.Imports = append(module.Imports, strings.Join(parts, "."))
	}
	if module.Profile == evt1VulkanProfileName && module.Name != "Vulkan" {
		implied := true
		for _, name := range module.Imports {
			if name == "Vulkan" {
				implied = false
			}
		}
		if implied {
			module.Imports = append(module.Imports, "Vulkan")
		}
	}
	for !p.done() {
		if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
			attributes, err := p.parseAttributes()
			if err != nil {
				return module, err
			}
			if p.peekLexeme() == "struct" || p.peekLexeme() == "record" || p.peekLexeme() == "table" || p.peekLexeme() == "class" || p.peekLexeme() == "immovable" || p.peekLexeme() == "ref" && (p.peekLexemeN(1) == "struct" || p.peekLexemeN(1) == "table") {
				var decl StructDecl
				var err error
				switch p.peekLexeme() {
				case "record":
					if p.peekLexemeN(1) == "table" {
						decl, err = p.parseTableDecl(true, false)
					} else {
						decl, err = p.parseStructDecl(false, true, false)
					}
				case "table":
					decl, err = p.parseTableDecl(false, false)
				case "class":
					decl, err = p.parseClassDecl()
				case "immovable":
					decl, err = p.parseStructDecl(true, false, false)
				case "ref":
					if p.peekLexemeN(1) == "table" {
						decl, err = p.parseTableDecl(false, true)
					} else {
						decl, err = p.parseStructDecl(false, false, true)
					}
				default:
					decl, err = p.parseStructDecl(false, false, false)
				}
				if err != nil {
					return module, err
				}
				decl.Attributes = attributes
				module.Structs = append(module.Structs, decl)
				module.Functions = append(module.Functions, decl.Methods...)
				continue
			}
			if p.peekLexeme() == "enum" {
				decl, err := p.parseEnumDecl()
				if err != nil {
					return module, err
				}
				decl.Attributes = attributes
				module.Enums = append(module.Enums, decl)
				continue
			}
			if p.peekLexeme() == "concept" || p.peekLexeme() == "innate" && p.peekLexemeN(1) == "concept" {
				innate := p.peekLexeme() == "innate"
				if innate {
					p.next()
				}
				decl, err := p.parseConceptDecl()
				if err != nil {
					return module, err
				}
				decl.Innate, decl.Attributes = innate, attributes
				module.Concepts = append(module.Concepts, decl)
				continue
			}
			if p.peekLexeme() == "template" && p.templateDeclIsRuntimeType() {
				decl, err := p.parseGenericTypeDecl()
				if err != nil {
					return module, err
				}
				decl.Struct.Attributes = attributes
				module.GenericTypes = append(module.GenericTypes, decl)
				continue
			}
			if p.peekLexeme() == "extern" {
				p.next()
				abi := p.current()
				if abi.Lexeme != `"C"` {
					return module, evt1Diagnostic("EXTERN_ABI_INVALID", "extern requires the supported ABI string \"C\"", abi.Span)
				}
				p.next()
				fn, err := p.parseFunctionDecl("", false)
				if err != nil {
					return module, err
				}
				if fn.Body != nil {
					return module, evt1Diagnostic("EXTERN_BODY_INVALID", "extern C declaration cannot have a body", fn.Span)
				}
				fn.ExternABI, fn.Attributes = "C", attributes
				module.Functions = append(module.Functions, fn)
				continue
			}
			async := false
			if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
				async = true
				p.next()
			}
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return module, err
			}
			fn.Attributes = attributes
			fn.Async = async
			module.Functions = append(module.Functions, fn)
			continue
		}
		switch p.peekLexeme() {
		case "generator":
			decl, err := p.parseGeneratorDecl()
			if err != nil {
				return module, err
			}
			module.Generators = append(module.Generators, decl)
		case "derive":
			request, err := p.parseGenerationRequest()
			if err != nil {
				return module, err
			}
			module.GenerationRequests = append(module.GenerationRequests, request)
		case "reflect":
			request, err := p.parseReflectionRequest()
			if err != nil {
				return module, err
			}
			module.ReflectionRequests = append(module.ReflectionRequests, request)
		case "namespace":
			if err := p.parseNamespaceBlock(&module); err != nil {
				return module, err
			}
		case "await", "awaitchronous":
			return module, evt1Diagnostic("AWAIT_OUTSIDE_ASYNC", "await is only valid inside an async function", p.currentSpan())
		case "async", "asynchronous":
			p.next() // exact lexical aliases normalize to FunctionDecl.Async
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return module, err
			}
			fn.Async = true
			module.Functions = append(module.Functions, fn)
		case "yield":
			return module, evt1Diagnostic("YIELD_OUTSIDE_STATE", "yield is only valid inside a runtime machine state body", p.currentSpan())
		case "comptime":
			isFunction, err := p.looksLikeComptimeFunction()
			if err != nil {
				return module, err
			}
			if isFunction {
				fn, err := p.parseFunctionDecl("", true)
				if err != nil {
					return module, err
				}
				module.ComptimeFns = append(module.ComptimeFns, fn)
				continue
			}
			decl, err := p.parseComptimeDecl()
			if err != nil {
				return module, err
			}
			module.ComptimeDecls = append(module.ComptimeDecls, decl)
		case "static_assert":
			assertion, err := p.parseStaticAssert()
			if err != nil {
				return module, err
			}
			module.StaticAsserts = append(module.StaticAsserts, assertion)
		case "template":
			if p.templateDeclIsRuntimeType() {
				decl, err := p.parseGenericTypeDecl()
				if err != nil {
					return module, err
				}
				module.GenericTypes = append(module.GenericTypes, decl)
				continue
			}
			templateDecl, err := p.parseTemplateDecl()
			if err != nil {
				return module, err
			}
			module.Templates = append(module.Templates, templateDecl)
		case "immovable":
			structDecl, err := p.parseStructDecl(true, false, false)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
			module.Functions = append(module.Functions, structDecl.Methods...)
		case "record":
			var structDecl StructDecl
			var err error
			if p.peekLexemeN(1) == "table" {
				structDecl, err = p.parseTableDecl(true, false)
			} else {
				structDecl, err = p.parseStructDecl(false, true, false)
			}
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
			module.Functions = append(module.Functions, structDecl.Methods...)
		case "ref":
			if p.peekLexemeN(1) == "table" {
				decl, err := p.parseTableDecl(false, true)
				if err != nil {
					return module, err
				}
				module.Structs = append(module.Structs, decl)
				module.Functions = append(module.Functions, decl.Methods...)
				continue
			}
			if p.peekLexemeN(1) != "struct" {
				fn, err := p.parseFunctionDecl("", false)
				if err != nil {
					return module, err
				}
				module.Functions = append(module.Functions, fn)
				continue
			}
			structDecl, err := p.parseStructDecl(false, false, true)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
			module.Functions = append(module.Functions, structDecl.Methods...)
		case "struct":
			structDecl, err := p.parseStructDecl(false, false, false)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
			module.Functions = append(module.Functions, structDecl.Methods...)
		case "bits":
			decl, err := p.parseBitsDecl()
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, decl)
		case "table":
			decl, err := p.parseTableDecl(false, false)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, decl)
			module.Functions = append(module.Functions, decl.Methods...)
		case "class":
			decl, err := p.parseClassDecl()
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, decl)
			module.Functions = append(module.Functions, decl.Methods...)
		case "layout":
			decl, err := p.parseLayoutDecl()
			if err != nil {
				return module, err
			}
			module.Layouts = append(module.Layouts, decl)
		case "stream":
			decl, err := p.parseStreamDecl()
			if err != nil {
				return module, err
			}
			module.Streams = append(module.Streams, decl)
		case "using", "type":
			decl, err := p.parseTypeAliasDecl(p.peekLexeme())
			if err != nil {
				return module, err
			}
			module.TypeAliases = append(module.TypeAliases, decl)
		case "enum":
			enumDecl, err := p.parseEnumDecl()
			if err != nil {
				return module, err
			}
			module.Enums = append(module.Enums, enumDecl)
		case "effect":
			return module, evt1RemovedSurface("effect", p.currentSpan())
		case "actuator":
			return module, evt1RemovedSurface("actuator", p.currentSpan())
		case "automata":
			automataDecl, err := p.parseAutomataDecl()
			if err != nil {
				return module, err
			}
			module.Automata = append(module.Automata, automataDecl)
		case "concept":
			conceptDecl, err := p.parseConceptDecl()
			if err != nil {
				return module, err
			}
			module.Concepts = append(module.Concepts, conceptDecl)
		case "innate":
			p.next()
			if p.peekLexeme() != "concept" {
				return module, evt1Diagnostic("INNATE_CONCEPT_SHAPE", "innate applies to a concept declaration: innate concept Name<FieldDeclaration F> { ... }", p.currentSpan())
			}
			conceptDecl, err := p.parseConceptDecl()
			if err != nil {
				return module, err
			}
			conceptDecl.Innate = true
			module.Concepts = append(module.Concepts, conceptDecl)
		case "interface":
			interfaceDecl, err := p.parseInterfaceDecl()
			if err != nil {
				return module, err
			}
			module.Concepts = append(module.Concepts, interfaceDecl)
		case "requires":
			if p.peekLexemeN(1) == "compiler" {
				effect, err := p.parseOperationEffectDecl()
				if err != nil {
					return module, err
				}
				module.OperationEffects = append(module.OperationEffects, effect)
				continue
			}
			assertion, err := p.parseConceptAssertion()
			if err != nil {
				return module, err
			}
			module.Assertions = append(module.Assertions, assertion)
		case "extern":
			p.next()
			abi := p.current()
			if abi.Lexeme != `"C"` {
				return module, evt1Diagnostic("EXTERN_ABI_INVALID", "extern requires the supported ABI string \"C\"", abi.Span)
			}
			p.next()
			if p.peekLexeme() == "handle" {
				start := p.next()
				name, err := p.expectIdentifier("HANDLE_DECL_INVALID", "expected a handle type name after `extern \"C\" handle`")
				if err != nil {
					return module, err
				}
				if _, err := p.expect(";"); err != nil {
					return module, err
				}
				module.Handles = append(module.Handles, HandleDecl{Name: name.Lexeme, Module: module.Name, Span: start.Span})
				continue
			}
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return module, err
			}
			if fn.Body != nil {
				return module, evt1Diagnostic("EXTERN_BODY_INVALID", "extern C declaration cannot have a body", fn.Span)
			}
			fn.ExternABI = "C"
			module.Functions = append(module.Functions, fn)
		case "foreign":
			contract, effect, err := p.parseForeignContractDecl()
			if err != nil {
				return module, err
			}
			contract.Module = module.Name
			module.ForeignContracts = append(module.ForeignContracts, contract)
			if effect.Operation != "" {
				effect.Module = module.Name
				module.OperationEffects = append(module.OperationEffects, effect)
			}
		default:
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return module, err
			}
			module.Functions = append(module.Functions, fn)
		}
	}
	addDefaultNamespaceSymbols(&module)
	return module, nil
}

func (p *parser) parseNamespaceBlock(module *Module) error {
	start, err := p.expect("namespace")
	if err != nil {
		return err
	}
	var parts []string
	for {
		part, err := p.expectIdentifier("NAMESPACE_NAME_INVALID", "expected namespace name")
		if err != nil {
			return err
		}
		parts = append(parts, part.Lexeme)
		if p.peekLexeme() != "." {
			break
		}
		p.next()
	}
	name := strings.Join(parts, ".")
	if _, err := p.expect("{"); err != nil {
		return err
	}
	for !p.done() && p.peekLexeme() != "}" {
		switch p.peekLexeme() {
		case "record":
			decl, err := p.parseStructDecl(false, true, false)
			if err != nil {
				return err
			}
			module.Structs = append(module.Structs, decl)
			module.Functions = append(module.Functions, decl.Methods...)
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: name, Module: module.Name, Name: decl.Name, Kind: "type", Span: decl.Span})
		case "struct":
			decl, err := p.parseStructDecl(false, false, false)
			if err != nil {
				return err
			}
			module.Structs = append(module.Structs, decl)
			module.Functions = append(module.Functions, decl.Methods...)
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: name, Module: module.Name, Name: decl.Name, Kind: "type", Span: decl.Span})
		case "class":
			decl, err := p.parseClassDecl()
			if err != nil {
				return err
			}
			module.Structs = append(module.Structs, decl)
			module.Functions = append(module.Functions, decl.Methods...)
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: name, Module: module.Name, Name: decl.Name, Kind: "type", Span: decl.Span})
		case "enum":
			decl, err := p.parseEnumDecl()
			if err != nil {
				return err
			}
			module.Enums = append(module.Enums, decl)
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: name, Module: module.Name, Name: decl.Name, Kind: "type", Span: decl.Span})
		case "namespace":
			return evt1Diagnostic("NAMESPACE_NESTING_INVALID", "use one dotted namespace name instead of nested namespace blocks", p.currentSpan())
		default:
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return err
			}
			module.Functions = append(module.Functions, fn)
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: name, Module: module.Name, Name: fn.Name, Kind: "function", Span: fn.Span})
		}
	}
	if _, err := p.expect("}"); err != nil {
		return err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	_ = start
	return nil
}

func addDefaultNamespaceSymbols(module *Module) {
	cut := strings.LastIndex(module.Name, ".")
	if cut <= 0 {
		return
	}
	namespace := module.Name[:cut]
	seen := map[string]bool{}
	for _, symbol := range module.NamespaceSymbols {
		seen[symbol.Kind+"|"+symbol.Name+"|"+fmt.Sprint(symbol.Span.Line)+"|"+fmt.Sprint(symbol.Span.Column)] = true
	}
	add := func(kind, name string, span Span) {
		key := kind + "|" + name + "|" + fmt.Sprint(span.Line) + "|" + fmt.Sprint(span.Column)
		if !seen[key] {
			module.NamespaceSymbols = append(module.NamespaceSymbols, NamespaceSymbol{Namespace: namespace, Module: module.Name, Name: name, Kind: kind, Span: span})
		}
	}
	for _, decl := range module.Structs {
		add("type", decl.Name, decl.Span)
	}
	for _, decl := range module.Enums {
		add("type", decl.Name, decl.Span)
	}
	for _, decl := range module.GenericTypes {
		add("type", decl.Name, decl.Span)
	}
	for _, decl := range module.Functions {
		add("function", decl.Name, decl.Span)
	}
	for _, decl := range module.Templates {
		add("function", decl.Name, decl.Span)
	}
}

func (p *parser) parseForeignContractDecl() (ForeignContractDecl, OperationEffectDecl, error) {
	// This source spelling is intentionally provisional. The semantic model is
	// kept in ForeignContractDecl so a later syntax decision does not change the
	// authority, artifact, proof, or lowering contracts.
	start, err := p.expect("foreign")
	if err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	if _, err = p.expect("concept"); err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	name, err := p.expectIdentifier("FOREIGN_CONTRACT_INVALID", "expected foreign contract name")
	if err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	if _, err = p.expect("on"); err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	op, err := p.expectIdentifier("FOREIGN_CONTRACT_INVALID", "expected bound foreign operation")
	if err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	if _, err = p.expect("{"); err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	decl := ForeignContractDecl{Name: name.Lexeme, Operation: op.Lexeme, SourcePath: p.path, Span: start.Span}
	var effect OperationEffectDecl
	for !p.done() && p.peekLexeme() != "}" {
		requirement, err := p.expect("requires")
		if err != nil {
			return ForeignContractDecl{}, OperationEffectDecl{}, err
		}
		if _, err = p.expect("compiler"); err != nil {
			return ForeignContractDecl{}, OperationEffectDecl{}, err
		}
		if _, err = p.expect("."); err != nil {
			return ForeignContractDecl{}, OperationEffectDecl{}, err
		}
		analysis, err := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected bounded foreign semantic fact")
		if err != nil {
			return ForeignContractDecl{}, OperationEffectDecl{}, err
		}
		switch analysis.Lexeme {
		case "NonNull":
			if decl.NonNullResult {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_DUPLICATE", "duplicate NonNull requirement", analysis.Span)
			}
			if _, err = p.expect("("); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			result, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected result subject")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			if result.Lexeme != "result" {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_INVALID", "NonNull subject must be result", result.Span)
			}
			if _, err = p.expect(")"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			decl.NonNullResult = true
		case "Allocates":
			if decl.Allocates {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_DUPLICATE", "duplicate Allocates requirement", analysis.Span)
			}
			if _, err = p.expect("("); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			target, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected operation name")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			if _, err = p.expect(")"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			if target.Lexeme != decl.Operation {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_TARGET_MISMATCH", "foreign Allocates fact must name the bound operation", target.Span)
			}
			decl.Allocates = true
			effect = OperationEffectDecl{Effect: "Allocates", Operation: decl.Operation, Origin: string(FactOriginDeclaredForeign), Module: decl.Name, Span: requirement.Span}
		case "ExternalStorage":
			if decl.AddressSpace.Name != "" {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_DUPLICATE", "duplicate ExternalStorage requirement", analysis.Span)
			}
			if _, err = p.expect("<"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			space, e := p.parseType("")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			decl.AddressSpace = space
			if _, err = p.expect(">"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			if _, err = p.expect("("); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			result, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected result subject")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			if result.Lexeme != "result" {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_INVALID", "ExternalStorage first subject must be result", result.Span)
			}
			if _, err = p.expect(","); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			extent, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected extent parameter")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			decl.ExtentParam = extent.Lexeme
			if _, err = p.expect(","); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			alignment, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected alignment parameter")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			decl.AlignmentParam = alignment.Lexeme
			if _, err = p.expect(")"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
		case "HostAccessible":
			if decl.HostAccessible {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_DUPLICATE", "duplicate HostAccessible requirement", analysis.Span)
			}
			if _, err = p.expect("("); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			result, e := p.expectIdentifier("FOREIGN_CONTRACT_FACT_INVALID", "expected result subject")
			if e != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, e
			}
			if result.Lexeme != "result" {
				return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_INVALID", "HostAccessible subject must be result", result.Span)
			}
			if _, err = p.expect(")"); err != nil {
				return ForeignContractDecl{}, OperationEffectDecl{}, err
			}
			decl.HostAccessible = true
		default:
			return ForeignContractDecl{}, OperationEffectDecl{}, evt1Diagnostic("FOREIGN_CONTRACT_FACT_INVALID", "foreign contracts support Allocates, ExternalStorage, HostAccessible, and NonNull", analysis.Span)
		}
		if _, err = p.expect(";"); err != nil {
			return ForeignContractDecl{}, OperationEffectDecl{}, err
		}
	}
	if _, err = p.expect("}"); err != nil {
		return ForeignContractDecl{}, OperationEffectDecl{}, err
	}
	return decl, effect, nil
}

func (p *parser) parseModuleNameDecl() (string, error) {
	if _, err := p.expect("module"); err != nil {
		return "", err
	}
	var parts []string
	for {
		tok, err := p.expectIdentifier("MODULE_NAME_INVALID", "expected module name")
		if err != nil {
			return "", err
		}
		parts = append(parts, tok.Lexeme)
		if p.peekLexeme() != "." {
			break
		}
		p.next()
	}
	if _, err := p.expect(";"); err != nil {
		return "", err
	}
	return strings.Join(parts, "."), nil
}

func (p *parser) parseOperationEffectDecl() (OperationEffectDecl, error) {
	start, err := p.expect("requires")
	if err != nil {
		return OperationEffectDecl{}, err
	}
	if _, err = p.expect("compiler"); err != nil {
		return OperationEffectDecl{}, err
	}
	if _, err = p.expect("."); err != nil {
		return OperationEffectDecl{}, err
	}
	effect, err := p.expectIdentifier("OPERATION_EFFECT_INVALID", "expected compiler operation effect")
	if err != nil {
		return OperationEffectDecl{}, err
	}
	if effect.Lexeme != "Allocates" && effect.Lexeme != "NoAllocation" && effect.Lexeme != "InvalidatesBorrows" {
		return OperationEffectDecl{}, evt1Diagnostic("OPERATION_EFFECT_INVALID", "supported operation effects are Allocates and InvalidatesBorrows", effect.Span)
	}
	if _, err = p.expect("("); err != nil {
		return OperationEffectDecl{}, err
	}
	op, err := p.expectIdentifier("OPERATION_EFFECT_INVALID", "expected operation name")
	if err != nil {
		return OperationEffectDecl{}, err
	}
	resource := ""
	if effect.Lexeme == "InvalidatesBorrows" {
		if _, err = p.expect(","); err != nil {
			return OperationEffectDecl{}, err
		}
		parameter, parameterErr := p.expectIdentifier("OPERATION_EFFECT_INVALID", "expected resource parameter name")
		if parameterErr != nil {
			return OperationEffectDecl{}, parameterErr
		}
		resource = parameter.Lexeme
	}
	if _, err = p.expect(")"); err != nil {
		return OperationEffectDecl{}, err
	}
	if _, err = p.expect(";"); err != nil {
		return OperationEffectDecl{}, err
	}
	return OperationEffectDecl{Effect: effect.Lexeme, Operation: op.Lexeme, Resource: resource, Span: start.Span}, nil
}

func (p *parser) parseAttributes() ([]Attribute, error) {
	var attributes []Attribute
	for p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
		start := p.next().Span
		p.next()
		name, err := p.expectIdentifier("TEST_ATTRIBUTE_INVALID", "expected test attribute name")
		if err != nil {
			return nil, err
		}
		attribute := Attribute{Name: name.Lexeme, Span: start}
		if p.peekLexeme() == "(" {
			p.next()
			if p.peekLexeme() != ")" {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					attribute.Args = append(attribute.Args, arg)
					if p.peekLexeme() != "," {
						break
					}
					p.next()
				}
			}
			if _, err := p.expect(")"); err != nil {
				return nil, err
			}
		}
		if _, err := p.expect("]"); err != nil {
			return nil, err
		}
		if _, err := p.expect("]"); err != nil {
			return nil, err
		}
		attributes = append(attributes, attribute)
	}
	return attributes, nil
}

func (p *parser) parseReflectionRequest() (ReflectionRequest, error) {
	start, err := p.expect("reflect")
	if err != nil {
		return ReflectionRequest{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return ReflectionRequest{}, err
	}
	typeArg, err := p.parseType("")
	if err != nil {
		return ReflectionRequest{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return ReflectionRequest{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return ReflectionRequest{}, err
	}
	return ReflectionRequest{Type: typeArg, Span: start.Span}, nil
}

func (p *parser) parseTypeAliasDecl(spelling string) (TypeAliasDecl, error) {
	start, err := p.expect(spelling)
	if err != nil {
		return TypeAliasDecl{}, err
	}
	name, err := p.expectIdentifier("CALLABLE_TYPE_QUERY_INVALID", "expected exact type alias name")
	if err != nil {
		return TypeAliasDecl{}, err
	}
	if _, err := p.expect("="); err != nil {
		return TypeAliasDecl{}, evt1Diagnostic("TYPE_ALIAS_INVALID", "type alias requires `=`", p.currentSpan())
	}
	if p.peekLexeme() != "typeof" {
		t, err := p.parseType("")
		if err != nil {
			return TypeAliasDecl{}, err
		}
		if _, err := p.expect(";"); err != nil {
			return TypeAliasDecl{}, err
		}
		return TypeAliasDecl{Name: name.Lexeme, Spelling: spelling, ResolvedType: t, Span: start.Span}, nil
	}
	p.next()
	if _, err := p.expect("("); err != nil {
		return TypeAliasDecl{}, err
	}
	query, err := p.parseExpr()
	if err != nil {
		return TypeAliasDecl{}, err
	}
	if _, err := p.expect(")"); err != nil {
		return TypeAliasDecl{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return TypeAliasDecl{}, err
	}
	return TypeAliasDecl{Name: name.Lexeme, Spelling: spelling, Query: query, Span: start.Span}, nil
}

func (p *parser) parseAutomataDecl() (AutomataDecl, error) {
	start, err := p.expect("automata")
	if err != nil {
		return AutomataDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4240", "expected automata name")
	if err != nil {
		return AutomataDecl{}, err
	}
	if p.peekLexeme() == "(" {
		return AutomataDecl{}, evt1RemovedSurface("signal automata", p.currentSpan())
	}
	decl := AutomataDecl{Name: nameTok.Lexeme, Span: start.Span}
	seenState := false
	for p.peekLexeme() == "with" {
		p.next()
		if p.peekLexeme() == "input" {
			inputTok := p.next()
			if decl.InputType.Name != "" {
				return AutomataDecl{}, evt1Diagnostic("AUTOMATA_INPUT_INVALID", "automata declares `with input` more than once", inputTok.Span)
			}
			decl.InputType, err = p.parseType("")
			if err != nil {
				return AutomataDecl{}, err
			}
			continue
		}
		if _, err := p.expect("state"); err != nil {
			return AutomataDecl{}, evt1Diagnostic("AUTOMATA_CAPTURE_INVALID", "automata `with` must be followed by `input` or `state`", p.currentSpan())
		}
		if seenState {
			return AutomataDecl{}, evt1Diagnostic("AUTOMATA_CAPTURE_INVALID", "automata declares `with state` more than once", p.currentSpan())
		}
		seenState = true
		if _, err := p.expect("{"); err != nil {
			return AutomataDecl{}, err
		}
		for !p.done() && p.peekLexeme() != "}" {
			field, err := p.parseAutomataStorageField("AUTOMATA_CAPTURE_INVALID", false)
			if err != nil {
				return AutomataDecl{}, err
			}
			decl.StateFields = append(decl.StateFields, field)
		}
		if _, err := p.expect("}"); err != nil {
			return AutomataDecl{}, err
		}
	}
	if _, err := p.expect("{"); err != nil {
		return AutomataDecl{}, err
	}
	for !p.done() && p.peekLexeme() != "}" {
		machine, err := p.parseMachineDecl()
		if err != nil {
			return AutomataDecl{}, err
		}
		decl.Machines = append(decl.Machines, machine)
	}
	if _, err := p.expect("}"); err != nil {
		return AutomataDecl{}, err
	}
	if len(decl.Machines) > 0 {
		for i := range decl.Machines {
			decl.Machines[i].Initial = i == 0
			for stateIndex := range decl.Machines[i].States {
				decl.Machines[i].States[stateIndex].Initial = stateIndex == 0
			}
		}
	}
	return decl, nil
}

func (p *parser) parseAutomataStorageField(code string, allowInitializer bool) (Field, error) {
	t, err := p.parseType("")
	if err != nil {
		return Field{}, err
	}
	name, err := p.expectIdentifier(code, "expected persistent state field name")
	if err != nil {
		return Field{}, err
	}
	var initializer Expr
	if p.peekLexeme() == "=" {
		if !allowInitializer {
			return Field{}, evt1Diagnostic(code, "automata `with state` fields are initialized explicitly by each instance", p.currentSpan())
		}
		p.next()
		initializer, err = p.parseExpr()
		if err != nil {
			return Field{}, err
		}
	}
	if _, err := p.expect(";"); err != nil {
		return Field{}, err
	}
	return Field{Type: t, Name: name.Lexeme, Visibility: "private", Span: name.Span, Initializer: initializer}, nil
}

func (p *parser) parseMachineDecl() (MachineDecl, error) {
	start := p.currentSpan()
	machine := MachineDecl{Span: start}
	if p.peekLexeme() == "initial" {
		return MachineDecl{}, evt1RemovedSurface("initial", p.currentSpan())
	}
	if _, err := p.expect("machine"); err != nil {
		return MachineDecl{}, evt1Diagnostic("CV4243", "expected machine declaration", p.currentSpan())
	}
	nameTok, err := p.expectIdentifier("CV4244", "expected machine name")
	if err != nil {
		return MachineDecl{}, err
	}
	machine.Name = nameTok.Lexeme
	machine.ResultType = Type{Name: "void", Kind: TypeBuiltin, Span: nameTok.Span}
	machine.ErrorType = Type{Name: "void", Kind: TypeBuiltin, Span: nameTok.Span}
	if p.peekLexeme() == "returns" {
		p.next()
		machine.ResultType, err = p.parseType("")
		if err != nil {
			return MachineDecl{}, err
		}
	}
	if p.peekLexeme() == "fails" {
		p.next()
		machine.ErrorType, err = p.parseType("")
		if err != nil {
			return MachineDecl{}, err
		}
	}
	if _, err := p.expect("{"); err != nil {
		return MachineDecl{}, err
	}
	for !p.done() && p.peekLexeme() != "}" {
		if p.peekLexeme() != "state" && p.peekLexeme() != "initial" && p.peekLexeme() != "terminal" {
			field, err := p.parseAutomataStorageField("MACHINE_STATE_ACCESS_INVALID", true)
			if err != nil {
				return MachineDecl{}, err
			}
			machine.Fields = append(machine.Fields, field)
			continue
		}
		state, err := p.parseStateDecl()
		if err != nil {
			return MachineDecl{}, err
		}
		machine.States = append(machine.States, state)
	}
	if _, err := p.expect("}"); err != nil {
		return MachineDecl{}, err
	}
	return machine, nil
}

func (p *parser) parseStateDecl() (StateDecl, error) {
	start := p.currentSpan()
	state := StateDecl{Span: start}
	if p.peekLexeme() == "initial" {
		return StateDecl{}, evt1RemovedSurface("initial", p.currentSpan())
	}
	if p.peekLexeme() == "terminal" {
		p.next()
		state.Terminal = true
	}
	if _, err := p.expect("state"); err != nil {
		return StateDecl{}, evt1Diagnostic("CV4247", "expected state declaration", p.currentSpan())
	}
	nameTok, err := p.expectIdentifier("CV4247", "expected state name")
	if err != nil {
		return StateDecl{}, err
	}
	state.Name = nameTok.Lexeme
	if _, err := p.expect("{"); err != nil {
		return StateDecl{}, err
	}
	body := Block{Span: state.Span}
	for !p.done() && p.peekLexeme() != "}" {
		var stmt Statement
		var err error
		switch {
		case p.peekLexeme() == "on" || (p.peekLexeme() == "otherwise" && p.peekLexemeN(1) == "=>"):
			stmt, err = p.parseOnStmt()
		case p.peekLexeme() == "finish" && p.peekLexemeN(1) == ";":
			err = evt1RemovedSurface("finish", p.currentSpan())
		default:
			stmt, err = p.parseStatement()
		}
		if err != nil {
			return StateDecl{}, err
		}
		body.Statements = append(body.Statements, stmt)
	}
	state.Body = &body
	if _, err := p.expect("}"); err != nil {
		return StateDecl{}, err
	}
	return state, nil
}

func (p *parser) templateDeclIsRuntimeType() bool {
	depth := 0
	for i := p.pos + 1; i < len(p.tokens); i++ {
		switch p.tokens[i].Lexeme {
		case "<":
			depth++
		case ">":
			depth--
			if depth == 0 && i+1 < len(p.tokens) {
				for j := i + 1; j < len(p.tokens); j++ {
					next := p.tokens[j].Lexeme
					if next == "struct" || next == "table" || next == "class" || ((next == "ref" || next == "record") && j+1 < len(p.tokens) && (p.tokens[j+1].Lexeme == "struct" || p.tokens[j+1].Lexeme == "table")) {
						return true
					}
					if next == ";" || next == "{" {
						return false
					}
				}
				return false
			}
		}
	}
	return false
}

func (p *parser) parseGenericTypeDecl() (GenericTypeDecl, error) {
	start, err := p.expect("template")
	if err != nil {
		return GenericTypeDecl{}, err
	}
	if _, err = p.expect("<"); err != nil {
		return GenericTypeDecl{}, err
	}
	var params []GenericParameter
	typeParams := map[string]bool{}
	for {
		if p.peekLexeme() == "typename" || p.peekLexeme() == "class" {
			p.next()
			name, e := p.expectIdentifier("GENERIC_PARAMETER_INVALID", "expected generic type parameter")
			if e != nil {
				return GenericTypeDecl{}, e
			}
			params = append(params, GenericParameter{Name: name.Lexeme, Kind: "type", Span: name.Span})
			typeParams[name.Lexeme] = true
		} else {
			valueType, e := p.parseType("")
			if e != nil {
				return GenericTypeDecl{}, e
			}
			name, e := p.expectIdentifier("GENERIC_PARAMETER_INVALID", "expected non-type template parameter")
			if e != nil {
				return GenericTypeDecl{}, e
			}
			params = append(params, GenericParameter{Name: name.Lexeme, Kind: "value", ValueType: valueType, Span: name.Span})
		}
		if p.peekLexeme() != "," {
			break
		}
		p.next()
	}
	if _, err = p.expect(">"); err != nil {
		return GenericTypeDecl{}, err
	}
	old := p.templateTypeParams
	p.templateTypeParams = typeParams
	defer func() { p.templateTypeParams = old }()
	var constraint TemplateConstraint
	if p.peekLexeme() == "requires" {
		req := p.next()
		if len(params) == 0 {
			return GenericTypeDecl{}, evt1Diagnostic("GENERIC_CONSTRAINT_INVALID", "generic type constraint requires a type parameter", req.Span)
		}
		ref, constraintErr := p.parseConceptUse(params[0].Name)
		if constraintErr != nil {
			return GenericTypeDecl{}, constraintErr
		}
		constraint = TemplateConstraint{ConceptName: ref.Name, TypeArg: ref.TypeArgs[0], TypeArgs: ref.TypeArgs, Span: req.Span}
	}
	var aggregate StructDecl
	if p.peekLexeme() == "ref" && p.peekLexemeN(1) == "table" {
		aggregate, err = p.parseTableDecl(false, true)
	} else if p.peekLexeme() == "record" && p.peekLexemeN(1) == "table" {
		aggregate, err = p.parseTableDecl(true, false)
	} else if p.peekLexeme() == "table" {
		aggregate, err = p.parseTableDecl(false, false)
	} else if p.peekLexeme() == "ref" {
		aggregate, err = p.parseStructDecl(false, false, true)
	} else if p.peekLexeme() == "struct" {
		aggregate, err = p.parseStructDecl(false, false, false)
	} else if p.peekLexeme() == "record" {
		aggregate, err = p.parseStructDecl(false, true, false)
	} else {
		aggregate, err = p.parseClassDecl()
	}
	if err != nil {
		return GenericTypeDecl{}, err
	}
	return GenericTypeDecl{Name: aggregate.Name, Parameters: params, Constraint: constraint, Struct: aggregate, Span: start.Span}, nil
}

func (p *parser) parseTemplateDecl() (TemplateDecl, error) {
	start, err := p.expect("template")
	if err != nil {
		return TemplateDecl{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return TemplateDecl{}, err
	}
	var params []GenericParameter
	typeParams := map[string]bool{}
	for {
		if p.peekLexeme() == "typename" || p.peekLexeme() == "class" {
			p.next()
			name, parseErr := p.expectIdentifier("CV4165", "expected template type parameter")
			if parseErr != nil {
				return TemplateDecl{}, parseErr
			}
			params = append(params, GenericParameter{Name: name.Lexeme, Kind: "type", Span: name.Span})
			typeParams[name.Lexeme] = true
		} else {
			valueType, parseErr := p.parseType("")
			if parseErr != nil {
				return TemplateDecl{}, parseErr
			}
			name, parseErr := p.expectIdentifier("CV4165", "expected non-type template parameter")
			if parseErr != nil {
				return TemplateDecl{}, parseErr
			}
			params = append(params, GenericParameter{Name: name.Lexeme, Kind: "value", ValueType: valueType, Span: name.Span})
		}
		if p.peekLexeme() != "," {
			break
		}
		p.next()
	}
	if len(params) == 0 {
		return TemplateDecl{}, evt1Diagnostic("CV4165", "template requires at least one parameter", p.currentSpan())
	}
	if _, err := p.expect(">"); err != nil {
		return TemplateDecl{}, err
	}
	oldTypeParams := p.templateTypeParams
	p.templateTypeParams = typeParams
	defer func() { p.templateTypeParams = oldTypeParams }()
	var constraint TemplateConstraint
	if p.peekLexeme() == "requires" {
		reqTok := p.next()
		ref, parseErr := p.parseConceptUse(params[0].Name)
		if parseErr != nil {
			return TemplateDecl{}, parseErr
		}
		constraint = TemplateConstraint{ConceptName: ref.Name, TypeArg: ref.TypeArgs[0], TypeArgs: ref.TypeArgs, Span: reqTok.Span}
	}
	async := false
	if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
		p.next()
		async = true
	}
	fn, err := p.parseFunctionDecl(params[0].Name, false)
	if err != nil {
		return TemplateDecl{}, err
	}
	if fn.Body == nil {
		return TemplateDecl{}, evt1Diagnostic("CV4167", "template declarations require a function body", fn.Span)
	}
	return TemplateDecl{
		Async:         async,
		Name:          fn.Name,
		Parameters:    params,
		TypeParam:     params[0].Name,
		TypeParamSpan: params[0].Span,
		Constraint:    constraint,
		ReturnType:    fn.ReturnType,
		Params:        fn.Params,
		Body:          fn.Body,
		Span:          start.Span,
	}, nil
}

func (p *parser) looksLikeComptimeFunction() (bool, error) {
	if p.peekLexeme() != "comptime" {
		return false, nil
	}
	save := p.pos
	defer func() { p.pos = save }()
	p.next()
	if _, err := p.parseType(""); err != nil {
		return false, err
	}
	if !isIdentifier(p.peekLexeme()) {
		return false, evt1Diagnostic("CV4183", "expected comptime declaration name", p.currentSpan())
	}
	p.next()
	return p.peekLexeme() == "(", nil
}

func (p *parser) parseComptimeDecl() (ComptimeDecl, error) {
	start, err := p.expect("comptime")
	if err != nil {
		return ComptimeDecl{}, err
	}
	t, err := p.parseType("")
	if err != nil {
		return ComptimeDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4183", "expected comptime declaration name")
	if err != nil {
		return ComptimeDecl{}, err
	}
	if _, err := p.expect("="); err != nil {
		return ComptimeDecl{}, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return ComptimeDecl{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return ComptimeDecl{}, err
	}
	return ComptimeDecl{Type: t, Name: nameTok.Lexeme, Value: value, Span: start.Span}, nil
}

func (p *parser) parseStaticAssert() (StaticAssert, error) {
	start, err := p.expect("static_assert")
	if err != nil {
		return StaticAssert{}, err
	}
	if _, err := p.expect("("); err != nil {
		return StaticAssert{}, err
	}
	condition, err := p.parseExpr()
	if err != nil {
		return StaticAssert{}, err
	}
	assertion := StaticAssert{Condition: condition, Span: start.Span}
	if p.peekLexeme() == "," {
		p.next()
		message, err := p.parseExpr()
		if err != nil {
			return StaticAssert{}, err
		}
		assertion.Message = message
	}
	if _, err := p.expect(")"); err != nil {
		return StaticAssert{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return StaticAssert{}, err
	}
	return assertion, nil
}

func (p *parser) parseStructDecl(immovable, record, refStruct bool) (StructDecl, error) {
	start := p.currentSpan()
	if refStruct {
		p.next()
	}
	if immovable {
		p.next()
	}
	recordSpan := Span{}
	if record {
		recordSpan = p.next().Span
	}
	if _, err := p.expect("struct"); err != nil {
		return StructDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4120", "expected struct name")
	if err != nil {
		return StructDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return StructDecl{}, err
	}
	decl := StructDecl{Name: nameTok.Lexeme, Immovable: immovable, Record: record, Ref: refStruct, Span: start, RecordSpan: recordSpan}
	visibility := "public"
	for !p.done() && p.peekLexeme() != "}" {
		if (p.peekLexeme() == "public" || p.peekLexeme() == "private") && p.peekLexemeN(1) == ":" {
			visibility = p.next().Lexeme
			p.next()
			continue
		}
		var attributes []Attribute
		if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
			attributes, err = p.parseAttributes()
			if err != nil {
				return StructDecl{}, err
			}
		}
		async := false
		if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
			p.next()
			async = true
		}
		fieldType, err := p.parseType("")
		if err != nil {
			return StructDecl{}, err
		}
		fieldName, err := p.expectIdentifier("CV4121", "expected field name")
		if err != nil {
			return StructDecl{}, err
		}
		if p.peekLexeme() == "(" {
			method, err := p.parseAggregateMethodTail(decl.Name, visibility, fieldType, fieldName)
			if err != nil {
				return StructDecl{}, err
			}
			method.Async = async
			decl.Methods = append(decl.Methods, method)
			continue
		}
		if async {
			return StructDecl{}, evt1Diagnostic("ASYNC_RETURN_TYPE_INVALID", "async is valid only on a function or method declaration", fieldName.Span)
		}
		if _, err := p.expect(";"); err != nil {
			return StructDecl{}, err
		}
		decl.Fields = append(decl.Fields, Field{Type: fieldType, Name: fieldName.Lexeme, Attributes: attributes, Visibility: visibility, Span: fieldName.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return StructDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	evt1RewriteAggregateMethods(&decl)
	return decl, nil
}

func (p *parser) parseTableDecl(record, refTable bool) (StructDecl, error) {
	start := p.currentSpan()
	if refTable {
		p.next()
	}
	recordSpan := Span{}
	if record {
		recordSpan = p.next().Span
	}
	if _, err := p.expect("table"); err != nil {
		return StructDecl{}, err
	}
	var cardinality Expr
	if p.peekLexeme() == "<" {
		p.next()
		var err error
		// `>` closes table cardinality syntax; parse arithmetic without treating
		// the delimiter as a comparison operator.
		cardinality, err = p.parseAdditive()
		if err != nil {
			return StructDecl{}, err
		}
		if _, err := p.expect(">"); err != nil {
			return StructDecl{}, err
		}
	}
	nameTok, err := p.expectIdentifier("TABLE_NAME_REQUIRED", "expected table name")
	if err != nil {
		return StructDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return StructDecl{}, err
	}
	decl := StructDecl{
		Name: nameTok.Lexeme, Record: record, Ref: refTable, Table: true,
		TableSized: cardinality != nil, Span: start, RecordSpan: recordSpan,
	}
	if cardinality != nil {
		decl.TableCardinalityExpression = evt1ExprIdentity(cardinality)
		if literal, ok := cardinality.(*IntLiteral); ok && !literal.Negative && literal.Magnitude <= uint64(^uint(0)>>1) {
			decl.TableCardinality = int(literal.Magnitude)
		}
	}
	visibility := "public"
	for !p.done() && p.peekLexeme() != "}" {
		if (p.peekLexeme() == "public" || p.peekLexeme() == "private") && p.peekLexemeN(1) == ":" {
			visibility = p.next().Lexeme
			p.next()
			continue
		}
		var attributes []Attribute
		if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
			attributes, err = p.parseAttributes()
			if err != nil {
				return StructDecl{}, err
			}
		}
		async := false
		if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
			p.next()
			async = true
		}
		fieldType, err := p.parseType("")
		if err != nil {
			return StructDecl{}, err
		}
		fieldName, err := p.expectIdentifier("TABLE_COLUMN_NAME_REQUIRED", "expected table column name")
		if err != nil {
			return StructDecl{}, err
		}
		if p.peekLexeme() == "(" {
			method, err := p.parseAggregateMethodTail(decl.Name, visibility, fieldType, fieldName)
			if err != nil {
				return StructDecl{}, err
			}
			method.Async = async
			decl.Methods = append(decl.Methods, method)
			continue
		}
		if async {
			return StructDecl{}, evt1Diagnostic("ASYNC_RETURN_TYPE_INVALID", "async is valid only on a function or method declaration", fieldName.Span)
		}
		if _, err := p.expect(";"); err != nil {
			return StructDecl{}, err
		}
		if cardinality != nil {
			element := fieldType
			fieldType = Type{
				Name: element.String() + "[]", Kind: TypeArray, ArrayElem: &element,
				ArrayLengthExpr: cardinality, StorageKind: StorageArray,
				Shape: []StorageDimension{{Expr: cardinality}}, Contiguous: true,
				Layout: "row-major", Column: true, Span: element.Span,
			}
		}
		decl.Fields = append(decl.Fields, Field{Type: fieldType, Name: fieldName.Lexeme, Attributes: attributes, Visibility: visibility, Span: fieldName.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return StructDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	evt1RewriteAggregateMethods(&decl)
	return decl, nil
}

func (p *parser) parseClassDecl() (StructDecl, error) {
	start, err := p.expect("class")
	if err != nil {
		return StructDecl{}, err
	}
	name, err := p.expectIdentifier("CV4650", "expected class name")
	if err != nil {
		return StructDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return StructDecl{}, err
	}
	decl := StructDecl{Name: name.Lexeme, Class: true, Span: start.Span}
	visibility := "private"
	for !p.done() && p.peekLexeme() != "}" {
		if (p.peekLexeme() == "public" || p.peekLexeme() == "private") && p.peekLexemeN(1) == ":" {
			visibility = p.next().Lexeme
			p.next()
			continue
		}
		var attributes []Attribute
		if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
			attributes, err = p.parseAttributes()
			if err != nil {
				return StructDecl{}, err
			}
		}
		async := false
		if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
			p.next()
			async = true
		}
		memberType, err := p.parseType("")
		if err != nil {
			return StructDecl{}, err
		}
		member, err := p.expectIdentifier("CV4651", "expected class member name")
		if err != nil {
			return StructDecl{}, err
		}
		if p.peekLexeme() == "(" {
			method, err := p.parseAggregateMethodTail(decl.Name, visibility, memberType, member)
			if err != nil {
				return StructDecl{}, err
			}
			method.Async = async
			method.Attributes = attributes
			decl.Methods = append(decl.Methods, method)
			continue
		}
		if len(attributes) != 0 {
			return StructDecl{}, evt1Diagnostic("TEST_ATTRIBUTE_REQUIRES_FUNCTION", "attributes inside a class require a method declaration", member.Span)
		}
		if async {
			return StructDecl{}, evt1Diagnostic("ASYNC_RETURN_TYPE_INVALID", "async is valid only on a function or method declaration", member.Span)
		}
		if _, err := p.expect(";"); err != nil {
			return StructDecl{}, err
		}
		decl.Fields = append(decl.Fields, Field{Type: memberType, Name: member.Lexeme, Visibility: visibility, Span: member.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return StructDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	evt1RewriteAggregateMethods(&decl)
	return decl, nil
}

func (p *parser) parseAggregateMethodTail(owner, visibility string, returnType Type, name Token) (FunctionDecl, error) {
	if _, err := p.expect("("); err != nil {
		return FunctionDecl{}, err
	}
	fn := FunctionDecl{Name: name.Lexeme, ReturnType: returnType, Span: name.Span, MethodOf: owner, Visibility: visibility}
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType("")
			if err != nil {
				return FunctionDecl{}, err
			}
			paramName, err := p.expectIdentifier("CV4652", "expected method parameter name")
			if err != nil {
				return FunctionDecl{}, err
			}
			fn.Params = append(fn.Params, Param{Type: paramType, Name: paramName.Lexeme, Span: paramName.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return FunctionDecl{}, err
	}
	if len(fn.Params) == 0 || !fn.Params[0].Type.isReference() || fn.Params[0].Type.valueType().Name != owner {
		selfType := Type{Name: owner, Kind: TypeStruct, Ownership: "ref", Span: name.Span}
		fn.Params = append([]Param{{Type: selfType, Name: "self", Span: name.Span}}, fn.Params...)
	}
	if p.peekLexeme() == ";" {
		p.next()
		return fn, nil
	}
	body, err := p.parseBlock()
	if err != nil {
		return FunctionDecl{}, err
	}
	fn.Body = &body
	return fn, nil
}

func (p *parser) parseLayoutDecl() (LayoutDecl, error) {
	start, err := p.expect("layout")
	if err != nil {
		return LayoutDecl{}, err
	}
	name, err := p.expectIdentifier("CV4570", "expected layout name")
	if err != nil {
		return LayoutDecl{}, err
	}
	if p.peekLexeme() == "(" {
		return LayoutDecl{}, evt1Diagnostic("CV4575", "runtime-parameterized layouts are deferred beyond EVT1", p.currentSpan())
	}
	if _, err := p.expect("{"); err != nil {
		return LayoutDecl{}, err
	}
	decl := LayoutDecl{Name: name.Lexeme, Span: start.Span}
	for !p.done() && p.peekLexeme() != "}" {
		region := LayoutRegion{}
		for p.peekLexeme() == "align" || p.peekLexeme() == "at" {
			modifier := p.next()
			if _, err := p.expect("("); err != nil {
				return LayoutDecl{}, err
			}
			valueTok := p.next()
			value, parseErr := strconv.ParseInt(valueTok.Lexeme, 0, 32)
			if parseErr != nil {
				return LayoutDecl{}, evt1Diagnostic("CV4572", modifier.Lexeme+" requires an integer constant", valueTok.Span)
			}
			if _, err := p.expect(")"); err != nil {
				return LayoutDecl{}, err
			}
			if modifier.Lexeme == "align" {
				region.RequestedAlign = int(value)
			} else {
				v := int(value)
				region.ExplicitOffset = &v
			}
		}
		regionType, err := p.parseType("")
		if err != nil {
			return LayoutDecl{}, err
		}
		regionName, err := p.expectIdentifier("CV4570", "expected layout region name")
		if err != nil {
			return LayoutDecl{}, err
		}
		if _, err := p.expect(";"); err != nil {
			return LayoutDecl{}, err
		}
		region.Type, region.Name, region.Span = regionType, regionName.Lexeme, regionName.Span
		decl.Regions = append(decl.Regions, region)
	}
	if _, err := p.expect("}"); err != nil {
		return LayoutDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return decl, nil
}

func (p *parser) parseStreamDecl() (StreamDecl, error) {
	start, err := p.expect("stream")
	if err != nil {
		return StreamDecl{}, err
	}
	name, err := p.expectIdentifier("CV4580", "expected stream name")
	if err != nil {
		return StreamDecl{}, err
	}
	if _, err := p.expect("over"); err != nil {
		return StreamDecl{}, err
	}
	layout, err := p.expectIdentifier("CV4580", "expected layout name after over")
	if err != nil {
		return StreamDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return StreamDecl{}, err
	}
	decl := StreamDecl{Name: name.Lexeme, LayoutName: layout.Lexeme, Span: start.Span}
	for !p.done() && p.peekLexeme() != "}" {
		channel, err := p.expectIdentifier("CV4580", "expected stream channel name")
		if err != nil {
			return StreamDecl{}, err
		}
		if _, err := p.expect("="); err != nil {
			return StreamDecl{}, err
		}
		region, err := p.expectIdentifier("CV4580", "expected layout region alias")
		if err != nil {
			return StreamDecl{}, err
		}
		if _, err := p.expect(";"); err != nil {
			return StreamDecl{}, err
		}
		decl.Channels = append(decl.Channels, StreamChannel{Name: channel.Lexeme, RegionName: region.Lexeme, Span: channel.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return StreamDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return decl, nil
}

func (p *parser) parseEnumDecl() (EnumDecl, error) {
	start := p.next().Span
	nameTok, err := p.expectIdentifier("CV4003", "expected enum name")
	if err != nil {
		return EnumDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return EnumDecl{}, err
	}
	enumDecl := EnumDecl{Name: nameTok.Lexeme, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		var attributes []Attribute
		if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
			attributes, err = p.parseAttributes()
			if err != nil {
				return EnumDecl{}, err
			}
		}
		variantTok, err := p.expectIdentifier("CV4004", "expected enum variant name")
		if err != nil {
			return EnumDecl{}, err
		}
		var payload []Field
		if p.peekLexeme() == "(" {
			p.next()
			if p.peekLexeme() != ")" {
				for {
					var payloadAttributes []Attribute
					if p.peekLexeme() == "[" && p.peekLexemeN(1) == "[" {
						payloadAttributes, err = p.parseAttributes()
						if err != nil {
							return EnumDecl{}, err
						}
					}
					fieldType, err := p.parseType("")
					if err != nil {
						return EnumDecl{}, err
					}
					fieldName, err := p.expectIdentifier("CV4005", "expected payload name")
					if err != nil {
						return EnumDecl{}, err
					}
					payload = append(payload, Field{Type: fieldType, Name: fieldName.Lexeme, Attributes: payloadAttributes, Span: fieldName.Span})
					if p.peekLexeme() != "," {
						break
					}
					p.next()
				}
			}
			if _, err := p.expect(")"); err != nil {
				return EnumDecl{}, err
			}
		}
		enumDecl.Variants = append(enumDecl.Variants, VariantDecl{Name: variantTok.Lexeme, Payload: payload, Attributes: attributes, Tag: len(enumDecl.Variants), Span: variantTok.Span})
		if p.peekLexeme() == "," {
			p.next()
		}
	}
	if _, err := p.expect("}"); err != nil {
		return EnumDecl{}, err
	}
	return enumDecl, nil
}

func (p *parser) parseConceptDecl() (ConceptDecl, error) {
	start := p.next().Span
	nameTok, err := p.expectIdentifier("CV4140", "expected concept name")
	if err != nil {
		return ConceptDecl{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return ConceptDecl{}, err
	}
	var params []GenericParameter
	conceptParams := map[string]bool{}
	for {
		kind, declarationKind := "type", ""
		if p.peekLexeme() == "declaration" {
			p.next()
			kind = "declaration"
		} else if _, narrowed := evt1InnateDeclarationKinds[p.peekLexeme()]; narrowed && p.peekLexemeN(1) != "," && p.peekLexemeN(1) != ">" {
			declarationKind = p.next().Lexeme
			kind = "declaration"
		}
		paramTok, parseErr := p.expectIdentifier("CV4141", "expected concept parameter")
		if parseErr != nil {
			return ConceptDecl{}, parseErr
		}
		params = append(params, GenericParameter{Name: paramTok.Lexeme, Kind: kind, DeclarationKind: declarationKind, Span: paramTok.Span})
		if kind == "type" {
			conceptParams[paramTok.Lexeme] = true
		}
		if p.peekLexeme() != "," {
			break
		}
		p.next()
	}
	if _, err := p.expect(">"); err != nil {
		return ConceptDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return ConceptDecl{}, err
	}
	oldTypeParams := p.templateTypeParams
	p.templateTypeParams = conceptParams
	defer func() { p.templateTypeParams = oldTypeParams }()
	decl := ConceptDecl{Name: nameTok.Lexeme, TypeParam: params[0].Name, Span: start}
	if len(params) > 1 || params[0].Kind != "type" {
		decl.Parameters = params
	}
	for !p.done() && p.peekLexeme() != "}" {
		req, err := p.parseConceptRequirement(params[0].Name)
		if err != nil {
			return ConceptDecl{}, err
		}
		decl.Requirements = append(decl.Requirements, req)
	}
	if _, err := p.expect("}"); err != nil {
		return ConceptDecl{}, err
	}
	return decl, nil
}

func (p *parser) parseInterfaceDecl() (ConceptDecl, error) {
	decl, err := p.parseConceptDecl()
	if err != nil {
		return ConceptDecl{}, err
	}
	decl.Interface = true
	return decl, nil
}

func (p *parser) parseConceptRequirement(typeParam string) (ConceptRequirement, error) {
	start, err := p.expect("requires")
	if err != nil {
		return nil, err
	}
	if p.peekLexeme() == "" {
		return nil, evt1Diagnostic("CV4142", "expected concept requirement", start.Span)
	}
	var operationParams []GenericParameter
	if p.peekLexeme() == "template" {
		p.next()
		if _, err := p.expect("<"); err != nil {
			return nil, err
		}
		old := p.templateTypeParams
		local := make(map[string]bool, len(old))
		for name, known := range old {
			local[name] = known
		}
		for {
			if p.peekLexeme() != "typename" && p.peekLexeme() != "class" {
				return nil, evt1Diagnostic("GENERIC_PARAMETER_INVALID", "required operation parameter must be a type", start.Span)
			}
			p.next()
			name, parseErr := p.expectIdentifier("GENERIC_PARAMETER_INVALID", "expected required operation type parameter")
			if parseErr != nil {
				return nil, parseErr
			}
			if local[name.Lexeme] {
				return nil, evt1Diagnostic("GENERIC_PARAMETER_INVALID", "required operation type parameter shadows another parameter", name.Span)
			}
			local[name.Lexeme] = true
			operationParams = append(operationParams, GenericParameter{Name: name.Lexeme, Kind: "type", Span: name.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(">"); err != nil {
			return nil, err
		}
		p.templateTypeParams = local
		defer func() { p.templateTypeParams = old }()
	}
	async := false
	if p.peekLexeme() == "async" || p.peekLexeme() == "asynchronous" {
		p.next()
		async = true
	}
	if p.peekLexeme() == "compiler" && p.peekLexemeN(1) == "." {
		p.next()
		p.next()
		analysis, err := p.expectIdentifier("CV4526", "expected compiler analysis name")
		if err != nil {
			return nil, err
		}
		req := &CompilerAnalysisRequirement{Analysis: analysis.Lexeme, Span: start.Span}
		if p.peekLexeme() == "<" {
			p.next()
			for {
				arg, err := p.parseType(typeParam)
				if err != nil {
					return nil, err
				}
				req.TypeArgs = append(req.TypeArgs, arg)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
			if _, err := p.expect(">"); err != nil {
				return nil, err
			}
		}
		if _, err := p.expect("("); err != nil {
			return nil, err
		}
		if p.peekLexeme() != ")" {
			for {
				if isNumber(p.peekLexeme()) {
					token := p.next()
					value, parseErr := strconv.ParseInt(token.Lexeme, 0, 32)
					if parseErr != nil {
						return nil, evt1Diagnostic("CV4643", "semantic fact parameter must be an integer", token.Span)
					}
					req.Parameters = append(req.Parameters, int(value))
					if p.peekLexeme() != "," {
						break
					}
					p.next()
					continue
				}
				subject, err := p.expectIdentifier("CV4527", "expected semantic subject name")
				if err != nil {
					return nil, err
				}
				req.SubjectArgs = append(req.SubjectArgs, SemanticSubjectRef{Name: subject.Lexeme, Span: subject.Span})
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return req, nil
	}
	if p.peekLexeme() == "sync" && p.peekLexemeN(1) == "." {
		analysis, nameSpan, err := p.parseQualifiedConceptName("CV4526", "expected synchronization concept name")
		if err != nil {
			return nil, err
		}
		req := &CompilerAnalysisRequirement{Analysis: analysis, Span: nameSpan}
		if _, err := p.expect("<"); err != nil {
			return nil, err
		}
		for {
			arg, err := p.parseType(typeParam)
			if err != nil {
				return nil, err
			}
			req.TypeArgs = append(req.TypeArgs, arg)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(">"); err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return req, nil
	}
	if p.isConceptApplicationAhead(typeParam) {
		name, _, arguments, err := p.parseConceptApplication(typeParam)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		var types []Type
		for _, argument := range arguments {
			if argument.Kind == "type" {
				types = append(types, argument.Type)
			}
		}
		requirement := &PrerequisiteRequirement{ConceptName: name, TypeArgs: types, Span: start.Span}
		for _, argument := range arguments {
			if argument.Kind == "declaration" {
				requirement.Arguments = arguments
				break
			}
		}
		if len(types) > 0 {
			requirement.TypeArg = types[0]
		}
		return requirement, nil
	}
	// `requires Predicate(D, ...);`: an operation requirement always spells
	// its result type first, so a name directly followed by `(` is a
	// predicate over declaration subjects.
	if p.peekLexemeN(1) == "(" && !async && len(operationParams) == 0 && isIdentifier(p.peekLexeme()) {
		name := p.next()
		p.next()
		req := &PredicateRequirement{Predicate: name.Lexeme, Span: name.Span}
		for p.peekLexeme() != ")" {
			subject, err := p.expectIdentifier("CV4527", "expected a declaration parameter")
			if err != nil {
				return nil, err
			}
			req.Subjects = append(req.Subjects, SemanticSubjectRef{Name: subject.Lexeme, Span: subject.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return req, nil
	}
	retType, err := p.parseType(typeParam)
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4143", "expected required operation name")
	if err != nil {
		return nil, err
	}
	if nameTok.Lexeme == "operator" {
		switch p.peekLexeme() {
		case "+", "-", "*", "/", "==", "!=", "<", ">", "<=", ">=", "~":
			nameTok.Lexeme += p.next().Lexeme
		default:
			return nil, evt1Diagnostic("CV4143", "expected supported required operator", p.currentSpan())
		}
	}
	if p.peekLexeme() == ";" {
		if async {
			return nil, evt1Diagnostic("INTERFACE_ASYNC_METHOD_SIGNATURE_MISMATCH", "async is valid only on an interface method requirement", start.Span)
		}
		p.next()
		return &FieldRequirement{Type: retType, Name: nameTok.Lexeme, Readonly: retType.Const, Span: start.Span}, nil
	}
	if p.peekLexeme() == "<" {
		code := "INTERFACE_NOT_DYN_COMPATIBLE"
		if async {
			code = "INTERFACE_ASYNC_METHOD_NOT_DYN_COMPATIBLE"
		}
		return nil, evt1Diagnostic(code, "open generic runtime interface methods do not have a fixed witness shape", nameTok.Span)
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	req := &OperationRequirement{ReturnType: retType, Name: nameTok.Lexeme, GenericParams: operationParams, Span: start.Span}
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType(typeParam)
			if err != nil {
				return nil, err
			}
			paramName, err := p.expectIdentifier("CV4144", "expected required parameter name")
			if err != nil {
				return nil, err
			}
			req.Params = append(req.Params, Param{Type: paramType, Name: paramName.Lexeme, Span: paramName.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	if async {
		req.ReturnType = evt1AsyncType(req.ReturnType, req.Name, start.Span)
	}
	return req, nil
}

func (p *parser) parseConceptAssertion() (ConceptAssertion, error) {
	start, err := p.expect("requires")
	if err != nil {
		return ConceptAssertion{}, err
	}
	name, _, arguments, err := p.parseConceptApplication("")
	if err != nil {
		return ConceptAssertion{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return ConceptAssertion{}, err
	}
	assertion := ConceptAssertion{ConceptName: name, Arguments: arguments, Span: start.Span}
	for _, argument := range arguments {
		if argument.Kind == "type" {
			assertion.TypeArgs = append(assertion.TypeArgs, argument.Type)
		}
	}
	if len(assertion.TypeArgs) > 0 {
		assertion.ConcreteType = assertion.TypeArgs[0]
	}
	return assertion, nil
}

func (p *parser) parseFunctionDecl(conceptParam string, comptime bool) (FunctionDecl, error) {
	if comptime {
		if _, err := p.expect("comptime"); err != nil {
			return FunctionDecl{}, err
		}
	}
	retType, err := p.parseType(conceptParam)
	if err != nil {
		return FunctionDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4006", "expected function name")
	if err != nil {
		return FunctionDecl{}, err
	}
	if _, err := p.expect("("); err != nil {
		return FunctionDecl{}, err
	}
	fn := FunctionDecl{Comptime: comptime, Name: nameTok.Lexeme, ReturnType: retType, Span: nameTok.Span}
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType(conceptParam)
			if err != nil {
				return FunctionDecl{}, err
			}
			paramName, err := p.expectIdentifier("CV4007", "expected parameter name")
			if err != nil {
				return FunctionDecl{}, err
			}
			fn.Params = append(fn.Params, Param{Type: paramType, Name: paramName.Lexeme, Span: paramName.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return FunctionDecl{}, err
	}
	if comptime && p.peekLexeme() == "bounded" {
		bounded := p.next()
		if _, err := p.expect("("); err != nil {
			return FunctionDecl{}, err
		}
		limit := p.next()
		value, err := strconv.Atoi(limit.Lexeme)
		if err != nil || value < 1 || value > evt1ComptimeMaxCallDepth {
			return FunctionDecl{}, evt1Diagnostic("CV4217", fmt.Sprintf("comptime recursion bound must be an integer from 1 to %d", evt1ComptimeMaxCallDepth), bounded.Span)
		}
		if _, err := p.expect(")"); err != nil {
			return FunctionDecl{}, err
		}
		fn.RecursionBound = value
	}
	if p.peekLexeme() == ";" {
		p.next()
		return fn, nil
	}
	block, err := p.parseBlock()
	if err != nil {
		return FunctionDecl{}, err
	}
	fn.Body = &block
	return fn, nil
}

func (p *parser) parseType(conceptParam string) (Type, error) {
	start := p.currentSpan()
	t := Type{Span: start}
	dyn := false
	if p.peekLexeme() == "dyn" {
		dyn = true
		p.next()
	}
	for {
		switch p.peekLexeme() {
		case "unsafe":
			return Type{}, evt1Diagnostic("TYPE_QUALIFIER_UNSUPPORTED", "unsafe type qualifiers have no qualified semantics; use an explicit foreign contract or unsafe asm", p.currentSpan())
		case "imported":
			return Type{}, evt1Diagnostic("TYPE_QUALIFIER_UNSUPPORTED", "imported type qualifiers have no qualified semantics; import the defining semantic module", p.currentSpan())
		case "owned":
			if t.Ownership != "" {
				return Type{}, evt1Diagnostic("OWNERSHIP_QUALIFIER_CONFLICT", "write exactly one ownership qualifier: owned, borrow, or ref", p.currentSpan())
			}
			t.Ownership = "owned"
			p.next()
		case "borrow":
			if t.Ownership != "" {
				return Type{}, evt1Diagnostic("OWNERSHIP_QUALIFIER_CONFLICT", "write exactly one ownership qualifier: owned, borrow, or ref", p.currentSpan())
			}
			t.Ownership = "borrow"
			p.next()
		case "ref":
			if t.Ownership != "" {
				return Type{}, evt1Diagnostic("OWNERSHIP_QUALIFIER_CONFLICT", "write exactly one ownership qualifier: owned, borrow, or ref", p.currentSpan())
			}
			t.Ownership = "ref"
			p.next()
		case "scoped":
			if t.Scoped {
				return Type{}, evt1Diagnostic("TYPE_QUALIFIER_DUPLICATE", "write scoped only once", p.currentSpan())
			}
			t.Scoped = true
			p.next()
		case "const":
			if t.Const {
				return Type{}, evt1Diagnostic("TYPE_QUALIFIER_DUPLICATE", "write const only once", p.currentSpan())
			}
			t.Const = true
			p.next()
		default:
			goto done
		}
	}
done:
	if dyn && p.peekLexeme() == "const" {
		t.Const = true
		p.next()
	}
	nameTok, err := p.expectIdentifier("CV4008", "expected type name")
	if err != nil {
		return Type{}, err
	}
	qualifiedName := nameTok.Lexeme
	for p.peekLexeme() == "." {
		p.next()
		part, partErr := p.expectIdentifier("NAMESPACE_QUALIFIED_NAME_INVALID", "expected qualified type name")
		if partErr != nil {
			return Type{}, partErr
		}
		qualifiedName += "." + part.Lexeme
	}
	nameTok.Lexeme = qualifiedName
	if !dyn && nameTok.Lexeme == "callback" && p.peekLexeme() == "<" {
		p.next()
		var params []Type
		if p.peekLexeme() != "->" {
			for {
				param, parseErr := p.parseType(conceptParam)
				if parseErr != nil {
					return Type{}, parseErr
				}
				params = append(params, param)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
		}
		if _, err := p.expect("->"); err != nil {
			return Type{}, evt1Diagnostic("CALLBACK_SIGNATURE_INVALID", "erased callback signature requires `->`", p.currentSpan())
		}
		result, parseErr := p.parseType(conceptParam)
		if parseErr != nil {
			return Type{}, parseErr
		}
		if _, err := p.expect(">"); err != nil {
			return Type{}, err
		}
		return Type{Name: "callback", Kind: TypeCallback, CallableParams: params, CallableResult: &result, Const: t.Const, Scoped: t.Scoped, Span: nameTok.Span}, nil
	}
	if !dyn && nameTok.Lexeme == "auto" {
		t.Name, t.Kind = "auto", TypeInferred
		return t, nil
	}
	if dyn {
		t.Name = nameTok.Lexeme
		t.Kind = TypeDyn
		return t, nil
	} else if builtin, ok := p.profileDef.builtinType(nameTok.Lexeme, nameTok.Span); ok {
		t.Name = builtin.Name
		t.Kind = builtin.Kind
		t.FloatRepresentation = builtin.FloatRepresentation
	} else if conceptParam != "" && nameTok.Lexeme == conceptParam {
		t.Name = nameTok.Lexeme
		t.Kind = TypeConceptParam
	} else if p.templateTypeParams != nil && p.templateTypeParams[nameTok.Lexeme] {
		t.Name, t.Kind = nameTok.Lexeme, TypeConceptParam
	} else {
		t.Name = nameTok.Lexeme
		t.Kind = TypeStruct
	}
	storageMarker := StorageKind("")
	storageElement := Type{}
	if p.peekLexeme() == "<" {
		storageElement = t
		p.next()
		_, knownUnit := quantityFromUnit(p.peekLexeme())
		if evt1NumericRepresentation(t) && (knownUnit || isUnsupportedStandardUnit(p.peekLexeme())) {
			dimension, err := p.parseQuantityDimension()
			if err != nil {
				return Type{}, err
			}
			if _, err := p.expect(">"); err != nil {
				return Type{}, err
			}
			if dimension.IsDimensionless() {
				if !dimension.Equal(dimensionlessQuantity()) {
					return Type{}, evt1Diagnostic("QUANTITY_DIMENSIONLESS_SCALE", "a scaled dimensionless type requires an explicit numeric scale conversion", nameTok.Span)
				}
			} else {
				t.Quantity = &dimension
			}
			// A quantity may still be a fixed-array element type.
			goto typeSuffix
		}
		if t.Name == "tensor" || t.Name == "vector" || t.Name == "matrix" {
			spelling := t.Name
			element, err := p.parseType(conceptParam)
			if err != nil {
				return Type{}, err
			}
			rank := 0
			if spelling == "vector" {
				rank = 1
			} else if spelling == "matrix" {
				rank = 2
			} else if p.peekLexeme() == "," {
				p.next()
				rankTok := p.current()
				if !isNumber(rankTok.Lexeme) {
					return Type{}, evt1Diagnostic("CV4610", "tensor rank must be a positive compile-time integer", rankTok.Span)
				}
				p.next()
				rank64, parseErr := strconv.ParseInt(rankTok.Lexeme, 10, 32)
				if parseErr != nil {
					return Type{}, evt1Diagnostic("CV4610", "tensor rank is outside the supported integer range", rankTok.Span)
				}
				if rank64 <= 0 {
					return Type{}, evt1Diagnostic("CV4610", "tensor rank must be positive", rankTok.Span)
				}
				rank = int(rank64)
			}
			if _, err := p.expect(">"); err != nil {
				return Type{}, err
			}
			t.Name, t.Kind, t.TypeArgs, t.TensorRank, t.TensorSpelling = "tensor", TypeTensor, []Type{element}, rank, spelling
			return t, nil
		}
		for {
			var arg Type
			if isNumber(p.peekLexeme()) {
				tok := p.next()
				arg = Type{Name: tok.Lexeme, Kind: TypeTemplateValue, Span: tok.Span}
			} else {
				var err error
				arg, err = p.parseType(conceptParam)
				if err != nil {
					return Type{}, err
				}
			}
			t.TypeArgs = append(t.TypeArgs, arg)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(">"); err != nil {
			return Type{}, err
		}
		t.Kind = TypeApplied
		if t.Name == "Async" {
			t.Kind = TypeAsync
		}
		if t.Name == "Address" {
			t.Kind = TypeAddress
		}
		if t.Name == "Storage" {
			t.Kind = TypeTypedStorage
		}
		if len(t.TypeArgs) == 1 && (t.TypeArgs[0].Name == string(StorageArray) || t.TypeArgs[0].Name == string(StorageNDArray) || t.TypeArgs[0].Name == string(StorageRaw) || t.TypeArgs[0].Name == string(StorageSparse)) {
			storageMarker = StorageKind(t.TypeArgs[0].Name)
			t = storageElement
		}
	}
typeSuffix:
	// An applied type may itself be the element of fixed storage, e.g.
	// Storage<T><array>[N]. Keep the storage suffix distinct from its type args.
	if p.peekLexeme() == "<" && (p.peekLexemeN(1) == string(StorageArray) || p.peekLexemeN(1) == string(StorageNDArray) || p.peekLexemeN(1) == string(StorageRaw) || p.peekLexemeN(1) == string(StorageSparse)) && p.peekLexemeN(2) == ">" {
		p.next()
		storageMarker = StorageKind(p.next().Lexeme)
		p.next()
	}
	if p.peekLexeme() == "*" {
		p.next()
		pointee := t
		t = Type{
			Name:      pointee.Name + "*",
			Kind:      TypePointer,
			Ownership: pointee.Ownership,
			Const:     pointee.Const,
			Imported:  pointee.Imported,
			Unsafe:    pointee.Unsafe,
			PointerTo: &pointee,
			Span:      nameTok.Span,
		}
	}
	for p.peekLexeme() == "[" {
		p.next()
		var dimensions []Expr
		for {
			dimension, err := p.parseExpr()
			if err != nil {
				return Type{}, err
			}
			dimensions = append(dimensions, dimension)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect("]"); err != nil {
			return Type{}, err
		}
		if (storageMarker == StorageArray || storageMarker == StorageRaw || storageMarker == StorageSparse || storageMarker == "") && len(dimensions) != 1 {
			return Type{}, evt1Diagnostic("CV4550", fmt.Sprintf("array storage requires exactly one extent, got %d", len(dimensions)), nameTok.Span)
		}
		elem := t
		ownership, isConst, scoped := elem.Ownership, elem.Const, elem.Scoped
		elem.Ownership, elem.Const, elem.Scoped = "", false, false
		kind := storageMarker
		if kind == "" {
			kind = StorageArray
		}
		shape := make([]StorageDimension, 0, len(dimensions))
		for _, dimension := range dimensions {
			shape = append(shape, StorageDimension{Expr: dimension})
		}
		t = Type{
			Name:        elem.String() + "[]",
			Kind:        TypeArray,
			Ownership:   ownership,
			Const:       isConst,
			Scoped:      scoped,
			ArrayElem:   &elem,
			StorageKind: kind,
			Shape:       shape,
			Contiguous:  kind != StorageRaw && kind != StorageSparse,
			Layout:      "row-major",
			Span:        nameTok.Span,
		}
		if len(dimensions) == 1 {
			t.ArrayLengthExpr = dimensions[0]
		}
		storageMarker = ""
	}
	if storageMarker != "" {
		return Type{}, evt1Diagnostic("CV4550", fmt.Sprintf("%s storage requires an extent list", storageMarker), nameTok.Span)
	}
	return t, nil
}

// parseQuantityDimension parses the deliberately bounded unit algebra used in
// numeric type qualifications: products, quotients, and signed integral
// powers. The normalized exponent vector, rather than source spelling, is the
// type identity (so float<Hz> and float<s^-1> are identical).
func (p *parser) parseQuantityDimension() (QuantityDimension, error) {
	result := dimensionlessQuantity()
	divide := false
	for {
		unitTok, err := p.expectIdentifier("QUANTITY_UNIT_UNKNOWN", "expected a unit name in quantity type")
		if err != nil {
			return QuantityDimension{}, err
		}
		unit, ok := quantityFromUnit(unitTok.Lexeme)
		if !ok {
			return QuantityDimension{}, quantityUnknownDiagnostic(unitTok)
		}
		exponent := 1
		if p.peekLexeme() == "^" {
			p.next()
			sign := 1
			if p.peekLexeme() == "-" {
				p.next()
				sign = -1
			}
			exponentTok := p.current()
			if !isNumber(exponentTok.Lexeme) {
				return QuantityDimension{}, evt1Diagnostic("QUANTITY_EXPONENT_INVALID", "unit exponent must be a compile-time integer", exponentTok.Span)
			}
			p.next()
			value, parseErr := strconv.ParseInt(exponentTok.Lexeme, 10, 32)
			if parseErr != nil {
				return QuantityDimension{}, evt1Diagnostic("QUANTITY_EXPONENT_INVALID", "unit exponent is outside the supported integer range", exponentTok.Span)
			}
			exponent = sign * int(value)
		}
		if divide {
			exponent = -exponent
		}
		powered, ok := unit.PowChecked(exponent)
		if !ok {
			return QuantityDimension{}, evt1Diagnostic("QUANTITY_SCALE_OVERFLOW", "exact unit scale exceeds the bounded compile-time rational range", unitTok.Span)
		}
		result, ok = result.MultiplyChecked(powered)
		if !ok {
			return QuantityDimension{}, evt1Diagnostic("QUANTITY_SCALE_OVERFLOW", "exact unit scale exceeds the bounded compile-time rational range", unitTok.Span)
		}
		switch p.peekLexeme() {
		case "*":
			p.next()
			divide = false
		case "/":
			p.next()
			divide = true
		default:
			return result.normalized(), nil
		}
	}
}

func (p *parser) parseConceptUse(conceptParam string) (Type, error) {
	name, span, arguments, err := p.parseConceptApplication(conceptParam)
	if err != nil {
		return Type{}, err
	}
	var types []Type
	for _, argument := range arguments {
		if argument.Kind != "type" {
			return Type{}, evt1Diagnostic("CONCEPT_ARGUMENT_CATEGORY_MISMATCH", "this concept application requires type arguments", argument.Span)
		}
		types = append(types, argument.Type)
	}
	return Type{Name: name, Kind: TypeApplied, TypeArgs: types, Span: span}, nil
}

func (p *parser) parseConceptApplication(conceptParam string) (string, Span, []ConceptArgumentRef, error) {
	name, span, err := p.parseQualifiedConceptName("CV4145", "expected concept name")
	if err != nil {
		return "", Span{}, nil, err
	}
	if _, err := p.expect("<"); err != nil {
		return "", Span{}, nil, err
	}
	var args []ConceptArgumentRef
	for {
		if p.peekLexeme() == "declaration" {
			p.next()
			identifier, parseErr := p.expectIdentifier("CONCEPT_DECLARATION_ARGUMENT_INVALID", "expected declaration subject name")
			if parseErr != nil {
				return "", Span{}, nil, parseErr
			}
			args = append(args, ConceptArgumentRef{Kind: "declaration", Declaration: identifier.Lexeme, Span: identifier.Span})
		} else {
			arg, parseErr := p.parseType(conceptParam)
			if parseErr != nil {
				return "", Span{}, nil, parseErr
			}
			args = append(args, ConceptArgumentRef{Kind: "type", Type: arg, Span: arg.Span})
		}
		if p.peekLexeme() != "," {
			break
		}
		p.next()
	}
	if _, err := p.expect(">"); err != nil {
		return "", Span{}, nil, err
	}
	return name, span, args, nil
}

func (p *parser) parseQualifiedConceptName(code, message string) (string, Span, error) {
	first, err := p.expectIdentifier(code, message)
	if err != nil {
		return "", Span{}, err
	}
	name := first.Lexeme
	for p.peekLexeme() == "." {
		p.next()
		part, err := p.expectIdentifier(code, message)
		if err != nil {
			return "", Span{}, err
		}
		name += "." + part.Lexeme
	}
	return name, first.Span, nil
}

func (p *parser) parseBlock() (Block, error) {
	open, err := p.expect("{")
	if err != nil {
		return Block{}, err
	}
	block := Block{Span: open.Span}
	for !p.done() && p.peekLexeme() != "}" {
		stmt, err := p.parseStatement()
		if err != nil {
			return Block{}, err
		}
		block.Statements = append(block.Statements, stmt)
	}
	if _, err := p.expect("}"); err != nil {
		return Block{}, err
	}
	return block, nil
}

func (p *parser) parseStatement() (Statement, error) {
	switch p.peekLexeme() {
	case "discard":
		start := p.next().Span
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &ExprStmt{Value: value, Discard: true, Span: start}, nil
	case "unsafe":
		return p.parseAsmStatement()
	case "derive":
		if p.inGeneratorBody {
			return nil, evt1Diagnostic("GENERATOR_RECURSION_UNSUPPORTED", "a generator cannot request another generation wave", p.currentSpan())
		}
		return nil, evt1Diagnostic("GENERATOR_REQUEST_SCOPE", "derive is only allowed at module scope", p.currentSpan())
	case "for":
		return p.parseForeachStmt()
	case "comptime":
		if p.pos+1 < len(p.tokens) && (p.tokens[p.pos+1].Lexeme == "if" || p.tokens[p.pos+1].Lexeme == "for") {
			start := p.next().Span
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			switch s := stmt.(type) {
			case *IfStmt:
				s.Comptime, s.Span = true, start
			case *ForeachStmt:
				s.Comptime, s.Span = true, start
			}
			return stmt, nil
		}
		return p.parseLocalComptimeDecl()
	case "static_assert":
		assertion, err := p.parseStaticAssert()
		if err != nil {
			return nil, err
		}
		return &StaticAssertStmt{Condition: assertion.Condition, Message: assertion.Message, Span: assertion.Span}, nil
	case "assert":
		return p.parseAssertStmt()
	case "try":
		return p.parseTryStmt()
	case "effects":
		return nil, evt1RemovedSurface("effects", p.currentSpan())
	case "actuation":
		return nil, evt1RemovedSurface("actuator", p.currentSpan())
	case "actuator":
		return nil, evt1RemovedSurface("actuator", p.currentSpan())
	case "emit":
		return nil, evt1RemovedSurface("emit", p.currentSpan())
	case "instance":
		return p.parseInstanceDecl()
	case "transition":
		start := p.next().Span
		if p.peekLexeme() == "match" {
			return p.parseTransitionMatchStmt(start)
		}
		if p.peekLexeme() == "decide" {
			return p.parseTransitionDecideStmt(start)
		}
		if p.peekLexeme() == "infer" {
			return p.parseTransitionInferStmt(start)
		}
		target, err := p.expectIdentifier("MACHINE_TRANSITION_INVALID", "expected local state name after transition")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &TransitionStmt{Target: target.Lexeme, Span: start}, nil
	case "yield":
		start := p.next().Span
		if p.peekLexeme() != ";" {
			return nil, evt1Diagnostic("YIELD_INVALID_CONTEXT", "yield is bare machine-step control; values and continuations are not supported", start)
		}
		p.next()
		return &YieldStmt{Span: start}, nil
	case "push":
		start := p.next().Span
		machine, err := p.expectIdentifier("MACHINE_PUSH_UNKNOWN", "expected child machine name after push")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect("goto"); err != nil {
			return nil, evt1Diagnostic("MACHINE_FRAME_INVALID", "push requires an explicit caller resume state: push Child goto Waiting;", p.currentSpan())
		}
		resume, err := p.expectIdentifier("MACHINE_UNKNOWN_STATE", "expected caller resume state after goto")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &PushMachineStmt{Machine: machine.Lexeme, ResumeState: resume.Lexeme, Span: start}, nil
	case "pop", "complete", "fail":
		kindTok := p.next()
		kind := map[string]string{"pop": "neutral", "complete": "neutral", "fail": "failure"}[kindTok.Lexeme]
		var value Expr
		if p.peekLexeme() != ";" {
			parsed, parseErr := p.parseExpr()
			if parseErr != nil {
				return nil, parseErr
			}
			value = parsed
			if kindTok.Lexeme == "complete" {
				kind = "success"
			}
		} else if kindTok.Lexeme == "fail" {
			return nil, evt1Diagnostic("MACHINE_FAIL_ERROR_TYPE_MISMATCH", "fail requires an error payload", kindTok.Span)
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &MachineCompleteStmt{Kind: kind, Operation: kindTok.Lexeme, Value: value, Span: kindTok.Span}, nil
	case "foreach":
		return p.parseForeachStmt()
	case "return":
		start := p.next().Span
		if p.peekLexeme() == ";" {
			p.next()
			return &ReturnStmt{Span: start}, nil
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &ReturnStmt{Value: value, Span: start}, nil
	case "if":
		return p.parseIfStmt()
	case "match":
		return p.parseMatchStmt()
	case "while":
		return p.parseWhileStmt()
	case "++", "--":
		operator := p.next()
		target, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return evt1MutationAssignment(target, operator.Lexeme, nil, operator.Span), nil
	case "bind":
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &ExprStmt{Value: value, Span: value.exprSpan()}, nil
	case "{":
		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &block, nil
	default:
		if p.peekLexeme() == "const" || p.peekLexeme() == "auto" || p.peekLexeme() == "let" || p.peekLexeme() == "var" || p.looksLikeVarDecl() {
			return p.parseVarDecl()
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peekLexeme() == "++" || p.peekLexeme() == "--" {
			operator := p.next()
			if _, err := p.expect(";"); err != nil {
				return nil, err
			}
			return evt1MutationAssignment(value, operator.Lexeme, nil, operator.Span), nil
		}
		if p.peekLexeme() == "=" || evt1CompoundOperator(p.peekLexeme()) != "" {
			operator := p.next()
			rhs, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(";"); err != nil {
				return nil, err
			}
			if operator.Lexeme != "=" {
				return evt1MutationAssignment(value, operator.Lexeme, rhs, operator.Span), nil
			}
			return &AssignStmt{Target: value, Value: rhs, Span: value.exprSpan()}, nil
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &ExprStmt{Value: value, Span: value.exprSpan()}, nil
	}
}

func evt1CompoundOperator(spelling string) string {
	switch spelling {
	case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=":
		return strings.TrimSuffix(spelling, "=")
	default:
		return ""
	}
}

func evt1MutationAssignment(target Expr, spelling string, rhs Expr, span Span) *AssignStmt {
	op := evt1CompoundOperator(spelling)
	if spelling == "++" || spelling == "--" {
		op = string(spelling[0])
		rhs = &IntLiteral{Magnitude: 1, Lexeme: "1", Span: span}
	}
	return &AssignStmt{Target: target, Value: &BinaryExpr{Op: op, Left: target, Right: rhs, Span: span}, CompoundOp: op, Span: target.exprSpan()}
}

func (p *parser) parseForeachStmt() (Statement, error) {
	start := p.next() // for and foreach share one iteration construct.
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	if start.Lexeme == "for" {
		for i := p.pos; i < len(p.tokens) && p.tokens[i].Lexeme != ")"; i++ {
			if p.tokens[i].Lexeme == ";" {
				return nil, evt1Diagnostic("FOREACH_ITERATOR_INVALID", "C-style for loops are unsupported; use `for (i in start..end step n)` or `descend` for reverse traversal; use bounded while for condition-driven loops", start.Span)
			}
		}
	}
	var itemType Type
	if p.pos+1 >= len(p.tokens) || p.tokens[p.pos+1].Lexeme != "in" {
		var err error
		itemType, err = p.parseType("")
		if err != nil {
			return nil, err
		}
	}
	item, err := p.expectIdentifier("FOREACH_ITERATOR_INVALID", "expected foreach item name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("in"); err != nil {
		return nil, evt1Diagnostic("FOREACH_ITERATOR_INVALID", "expected `in` in foreach", p.currentSpan())
	}
	source, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ForeachStmt{ItemType: itemType, ItemName: item.Lexeme, Source: source, Body: body, Span: start.Span}, nil
}

// parseOnStmt parses `on Pattern [when guard | otherwise] => Target;`,
// `on Pattern ... => { ... }`, and the state-level `otherwise => ...`.
func (p *parser) parseOnStmt() (Statement, error) {
	start := p.next()
	stmt := &OnStmt{Span: start.Span}
	if start.Lexeme == "otherwise" {
		stmt.CatchAll = true
	} else {
		pattern, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		stmt.Pattern = pattern
		switch p.peekLexeme() {
		case "when":
			p.next()
			stmt.Guard, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		case "otherwise":
			p.next()
			stmt.Otherwise = true
		case "goto", "push":
			return nil, evt1RemovedSurface("on goto", p.currentSpan())
		}
	}
	if p.peekLexeme() == "=>" && (p.peekLexemeN(1) == "goto" || p.peekLexemeN(1) == "push") {
		return nil, evt1RemovedSurface("on goto", p.currentSpan())
	}
	if _, err := p.expect("=>"); err != nil {
		return nil, evt1Diagnostic("ON_SYNTAX_INVALID", "input reactions use `on Pattern [when guard | otherwise] => Target;` or `=> { ... }`", p.currentSpan())
	}
	if p.peekLexeme() == "{" {
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Body = body
		return stmt, nil
	}
	target, err := p.expectIdentifier("ON_SYNTAX_INVALID", "expected a local state or `{` after `=>`")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	stmt.Target = target.Lexeme
	stmt.Body = Block{Span: target.Span, Statements: []Statement{&TransitionStmt{Target: target.Lexeme, Span: target.Span}}}
	return stmt, nil
}

func (p *parser) parseTransitionMatchStmt(start Span) (Statement, error) {
	p.next() // match
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	subject, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	stmt := &TransitionMatchStmt{Subject: subject, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		pattern, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		var guard Expr
		if p.peekLexeme() == "when" {
			p.next()
			guard, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		if _, err := p.expect("=>"); err != nil {
			return nil, err
		}
		target, err := p.expectIdentifier("TRANSITION_MATCH_UNKNOWN_TARGET", "expected local state target after `=>`")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		stmt.Arms = append(stmt.Arms, TransitionMatchArm{Pattern: pattern, Guard: guard, Target: target.Lexeme, Span: pattern.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return stmt, nil
}

func (p *parser) parseTransitionDecideStmt(start Span) (Statement, error) {
	p.next() // decide
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	stmt := &TransitionDecideStmt{Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		candidate, err := p.parseScoredCandidate("TRANSITION_DECIDE_UNKNOWN_TARGET", "expected local state candidate", len(stmt.Candidates))
		if err != nil {
			return nil, err
		}
		stmt.Candidates = append(stmt.Candidates, candidate)
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return stmt, nil
}

func (p *parser) parseTransitionInferStmt(start Span) (Statement, error) {
	p.next() // infer
	if p.peekLexeme() != "with" {
		return nil, evt1Diagnostic("TRANSITION_INFER_REQUIRES_POLICY", "transition infer requires an explicit policy introduced by `with`", p.currentSpan())
	}
	p.next()
	policy, err := p.expectIdentifier("TRANSITION_INFER_UNKNOWN_POLICY", "expected transition inference policy")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	stmt := &TransitionInferStmt{Policy: policy.Lexeme, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		candidate, err := p.parseScoredCandidate("TRANSITION_INFER_UNKNOWN_TARGET", "expected local state candidate", len(stmt.Candidates))
		if err != nil {
			return nil, err
		}
		stmt.Candidates = append(stmt.Candidates, candidate)
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return stmt, nil
}

func (p *parser) parseScoredCandidate(code, message string, order int) (ScoredCandidate, error) {
	name, err := p.expectIdentifier(code, message)
	if err != nil {
		return ScoredCandidate{}, err
	}
	candidate := ScoredCandidate{Identity: name.Lexeme, DeclarationOrder: order, Span: name.Span}
	if p.peekLexeme() == "when" {
		p.next()
		candidate.Guard, err = p.parseExpr()
		if err != nil {
			return ScoredCandidate{}, err
		}
	}
	if p.peekLexeme() != "score" {
		scoreCode := "INFER_SCORE_REQUIRES_FLOAT"
		if strings.HasPrefix(code, "TRANSITION_DECIDE") {
			scoreCode = "TRANSITION_DECIDE_SCORE_TYPE_INVALID"
		}
		if strings.HasPrefix(code, "DECIDE") {
			scoreCode = "DECIDE_SCORE_TYPE_INVALID"
		}
		return ScoredCandidate{}, evt1Diagnostic(scoreCode, "expected `score` in scored candidate", p.currentSpan())
	}
	p.next()
	candidate.Score, err = p.parseExpr()
	if err != nil {
		return ScoredCandidate{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return ScoredCandidate{}, err
	}
	return candidate, nil
}

func (p *parser) parseAssertStmt() (Statement, error) {
	start, _ := p.expect("assert")
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	var reason Expr
	if p.peekLexeme() == "," {
		p.next()
		reason, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &AssertStmt{Condition: condition, Reason: reason, Span: start.Span}, nil
}

func (p *parser) parseTryStmt() (Statement, error) {
	start, _ := p.expect("try")
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt := &TryStmt{Body: body, Span: start.Span}
	for p.peekLexeme() == "except" {
		exceptTok := p.next()
		if _, err := p.expect("("); err != nil {
			return nil, err
		}
		errorType, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		binding, err := p.expectIdentifier("CV4547", "expected error binding in except arm")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		armBody, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Except = append(stmt.Except, ExceptArm{ErrorType: errorType, Binding: binding.Lexeme, Body: armBody, Span: exceptTok.Span})
	}
	if len(stmt.Except) == 0 {
		return nil, evt1Diagnostic("CV4545", "try requires at least one except arm", start.Span)
	}
	return stmt, nil
}

func (p *parser) parseInstanceDecl() (Statement, error) {
	start, err := p.expect("instance")
	if err != nil {
		return nil, err
	}
	automataTok, err := p.expectIdentifier("CV4270", "expected automata name after instance")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4270", "expected local instance name")
	if err != nil {
		return nil, err
	}
	var args []Expr
	if p.peekLexeme() == "(" {
		p.next()
		if p.peekLexeme() != ")" {
			for {
				arg, argErr := p.parseExpr()
				if argErr != nil {
					return nil, argErr
				}
				args = append(args, arg)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &InstanceDecl{AutomataName: automataTok.Lexeme, Name: nameTok.Lexeme, StateArgs: args, Span: start.Span}, nil
}

func (p *parser) parseLocalComptimeDecl() (Statement, error) {
	start, err := p.expect("comptime")
	if err != nil {
		return nil, err
	}
	stmt, err := p.parseVarDecl()
	if err != nil {
		return nil, err
	}
	decl := stmt.(*VarDecl)
	decl.Comptime, decl.Span = true, start.Span
	return decl, nil
}

func (p *parser) looksLikeVarDecl() bool {
	if p.done() {
		return false
	}
	save := p.pos
	defer func() { p.pos = save }()
	if p.peekLexeme() == "const" || p.peekLexeme() == "let" || p.peekLexeme() == "var" {
		p.next()
	}
	if p.peekLexeme() == "auto" {
		p.next()
	}
	if _, err := p.parseType(""); err != nil {
		return false
	}
	if !isIdentifier(p.peekLexeme()) {
		return false
	}
	p.next()
	return p.peekLexeme() == "[" || p.peekLexeme() == "=" || p.peekLexeme() == ";"
}

func (p *parser) parseVarDecl() (Statement, error) {
	isConst := false
	constSpan := Span{}
	if p.peekLexeme() == "auto" {
		qualifier := p.next()
		return p.parseInferredVarDecl(false, qualifier.Span)
	}
	if p.peekLexeme() == "const" {
		qualifier := p.next()
		isConst, constSpan = true, qualifier.Span
		if p.peekLexeme() == "auto" {
			p.next()
			return p.parseInferredVarDecl(true, qualifier.Span)
		}
	} else if p.peekLexeme() == "let" || p.peekLexeme() == "var" {
		qualifier := p.next()
		isConst = qualifier.Lexeme != "var"
		constSpan = qualifier.Span
		if isIdentifier(p.peekLexeme()) && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Lexeme == "=" {
			return p.parseInferredVarDecl(isConst, constSpan)
		}
	}
	t, err := p.parseType("")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4009", "expected local name")
	if err != nil {
		return nil, err
	}
	var inline *InlineTensorDecl
	if evt1IsTensorType(t) && p.peekLexeme() == "[" {
		p.next()
		var shape []StorageDimension
		for {
			dimension, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			shape = append(shape, StorageDimension{Expr: dimension})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect("]"); err != nil {
			return nil, err
		}
		inline = &InlineTensorDecl{Spelling: t.TensorSpelling, Shape: shape}
		if t.TensorRank == 0 {
			t.TensorRank = len(shape)
		}
	}
	var value Expr
	if p.peekLexeme() == "=" {
		p.next()
		value, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &VarDecl{Const: isConst, Type: t, Name: nameTok.Lexeme, Value: value, InlineTensor: inline, Span: nameTok.Span, ConstSpan: constSpan}, nil
}

func (p *parser) parseInferredVarDecl(isConst bool, qualifierSpan Span) (Statement, error) {
	nameTok, err := p.expectIdentifier("CV4009", "expected inferred local name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("="); err != nil {
		return nil, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &VarDecl{Const: isConst, Type: Type{Name: "<inferred>", Kind: TypeInferred, Span: nameTok.Span}, Name: nameTok.Lexeme, Value: value, Span: nameTok.Span, ConstSpan: qualifierSpan}, nil
}

func (p *parser) parseMatchStmt() (Statement, error) {
	start := p.next().Span
	if p.peekLexeme() == "{" {
		return p.parseGuardedMatchStmt(start)
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	subject, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	stmt := &MatchStmt{Subject: subject, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		pattern, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect("=>"); err != nil {
			return nil, err
		}
		if p.peekLexeme() != "{" {
			return nil, evt1Diagnostic("CV4118", "statement-form match arms require braced blocks", p.currentSpan())
		}
		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Arms = append(stmt.Arms, StatementArm{Pattern: pattern, Block: block, Span: pattern.Span})
		if p.peekLexeme() == "," {
			p.next()
		}
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *parser) parsePattern() (Pattern, error) {
	if p.peekLexeme() == "_" {
		tok := p.next()
		return Pattern{Wildcard: true, Span: tok.Span}, nil
	}
	negative := p.peekLexeme() == "-" && isNumber(p.peekLexemeN(1))
	if isNumber(p.peekLexeme()) || negative {
		span := p.currentSpan()
		if negative {
			p.next()
		}
		tok := p.next()
		literal, err := evt1ParseIntegerLiteral(tok.Lexeme, negative, span)
		if err != nil {
			return Pattern{}, err
		}
		return Pattern{Literal: literal, Span: span}, nil
	}
	enumTok, err := p.expectIdentifier("CV4010", "expected enum name in match arm")
	if err != nil {
		return Pattern{}, err
	}
	if _, err := p.expect("::"); err != nil {
		return Pattern{}, err
	}
	variantTok, err := p.expectIdentifier("CV4011", "expected variant name in match arm")
	if err != nil {
		return Pattern{}, err
	}
	pattern := Pattern{EnumName: enumTok.Lexeme, VariantName: variantTok.Lexeme, Span: enumTok.Span}
	if p.peekLexeme() == "(" {
		p.next()
		if p.peekLexeme() != ")" {
			for {
				binding, err := p.expectIdentifier("CV4012", "expected payload binding name")
				if err != nil {
					return Pattern{}, err
				}
				pattern.Bindings = append(pattern.Bindings, binding.Lexeme)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
		}
		if _, err := p.expect(")"); err != nil {
			return Pattern{}, err
		}
	}
	return pattern, nil
}

func (p *parser) parseExpr() (Expr, error) {
	return p.parseIfExpr()
}

func (p *parser) parseIfExpr() (Expr, error) {
	if p.peekLexeme() != "if" {
		return p.parseWithExpr()
	}
	start := p.next().Span
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	thenExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("else"); err != nil {
		return nil, evt1Diagnostic("CV4184", "if expressions require an else branch", p.currentSpan())
	}
	elseExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if evt1IsDirectIfExpr(elseExpr) {
		return nil, evt1Diagnostic("CV4185", "else-if ladders are not supported; use match for multi-branch selection", elseExpr.exprSpan())
	}
	return &IfExpr{Condition: condition, Then: thenExpr, Else: elseExpr, Span: start}, nil
}

func (p *parser) parseWithExpr() (Expr, error) {
	base, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	if p.peekLexeme() != "with" {
		return base, nil
	}
	withToken := p.next()
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	expr := &WithExpr{Base: base, Span: withToken.Span}
	for !p.done() && p.peekLexeme() != "}" {
		name, err := p.expectIdentifier("CV4141", "expected record field name in with update")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect("="); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		expr.Updates = append(expr.Updates, FieldUpdate{Name: name.Lexeme, NameSpan: name.Span, Value: value})
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return expr, nil
}

func evt1IsDirectIfExpr(expr Expr) bool {
	switch e := expr.(type) {
	case *IfExpr:
		return true
	case *ParenExpr:
		return evt1IsDirectIfExpr(e.Value)
	default:
		return false
	}
}

func (p *parser) parseLogicalOr() (Expr, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "or" {
		op := p.next()
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseLogicalAnd() (Expr, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "and" {
		op := p.next()
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseEquality() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "==" || p.peekLexeme() == "!=" {
		op := p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parseRange()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "<" || p.peekLexeme() == ">" || p.peekLexeme() == "<=" || p.peekLexeme() == ">=" {
		op := p.next()
		right, err := p.parseRange()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseRange() (Expr, error) {
	start, err := p.parseShift()
	if err != nil || p.peekLexeme() != ".." {
		return start, err
	}
	operator := p.next()
	end, err := p.parseShift()
	if err != nil {
		return nil, err
	}
	rangeValue := Expr(&BinaryExpr{Op: "..", Left: start, Right: end, Span: operator.Span})
	if p.peekLexeme() == "step" || p.peekLexeme() == "descend" {
		modifier := p.next()
		magnitude, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		rangeValue = &BinaryExpr{Op: modifier.Lexeme, Left: rangeValue, Right: magnitude, Span: modifier.Span}
	}
	if p.peekLexeme() == "step" || p.peekLexeme() == "descend" {
		return nil, evt1Diagnostic("RANGE_DIRECTION_INVALID", "a range uses either step or descend, not both", p.currentSpan())
	}
	return rangeValue, nil
}

func (p *parser) parseBitwiseOr() (Expr, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "|" {
		op := p.next()
		right, err := p.parseBitwiseXor()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseBitwiseXor() (Expr, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "^" {
		op := p.next()
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseBitwiseAnd() (Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "&" {
		op := p.next()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseShift() (Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "<<" || (p.peekLexeme() == ">" && p.peekLexemeN(1) == ">") {
		op := p.next()
		if op.Lexeme == ">" {
			p.next()
			op.Lexeme = ">>"
		}
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseAdditive() (Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "+" || p.peekLexeme() == "-" {
		op := p.next()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseMultiplicative() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "*" || p.peekLexeme() == "/" || p.peekLexeme() == "%" || p.peekLexeme() == "@" {
		op := p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Lexeme, Left: left, Right: right, Span: op.Span}
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.peekLexeme() == "interpret" {
		start := p.next().Span
		previous := p.suppressAs
		p.suppressAs = true
		value, err := p.parseUnary()
		p.suppressAs = previous
		if err != nil {
			return nil, err
		}
		if _, err := p.expect("as"); err != nil {
			return nil, evt1Diagnostic("INTERPRET_SYNTAX", "interpret requires `interpret expression as T`; parenthesize compound source expressions", start)
		}
		target, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(&InterpretExpr{Value: value, Target: target, Span: start}, start)
	}
	if p.peekLexeme() == "await" || p.peekLexeme() == "awaitchronous" {
		op := p.next()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		// Postfix failure propagation binds to the awaited value, not to the
		// Async<T> operand: `await Child()?` == `(await Child())?`.
		if failure, ok := value.(*FailureExpr); ok {
			awaited := &AwaitExpr{Value: failure.Value, Span: op.Span}
			return &FailureExpr{Op: failure.Op, Value: awaited, Else: failure.Else, Span: failure.Span}, nil
		}
		return &AwaitExpr{Value: value, Span: op.Span}, nil
	}
	if p.peekLexeme() == "bind" && p.peekLexemeN(1) != "<" {
		op := p.next()
		source, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &BindExpr{Source: source, Span: op.Span}, nil
	}
	if p.peekLexeme() == "move" {
		op := p.next()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &MoveExpr{Value: value, Span: op.Span}, nil
	}
	if p.peekLexeme() == "ref" {
		op := p.next()
		isConst := false
		if p.peekLexeme() == "const" {
			p.next()
			isConst = true
		}
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &RefExpr{Value: value, Const: isConst, Span: op.Span}, nil
	}
	if p.peekLexeme() == "-" && isNumber(p.peekLexemeN(1)) {
		op := p.next()
		tok := p.next()
		literal, err := evt1ParseIntegerLiteral(tok.Lexeme, true, op.Span)
		if err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(literal, op.Span)
	}
	if p.peekLexeme() == "-" || p.peekLexeme() == "not" || p.peekLexeme() == "~" {
		op := p.next()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Lexeme, Value: value, Span: op.Span}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parseIfStmt() (Statement, error) {
	start, err := p.expect("if")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt := &IfStmt{Condition: condition, Then: thenBlock, Span: start.Span}
	if p.peekLexeme() == "else" {
		p.next()
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Else = &elseBlock
	}
	return stmt, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	switch {
	case p.done():
		return nil, evt1Diagnostic("CV4013", "unexpected end of expression", p.currentSpan())
	case p.peekLexeme() == "(":
		start := p.next().Span
		if _, numeric := evt1BuiltinType(p.peekLexeme(), p.currentSpan()); numeric && p.peekLexemeN(1) == ")" {
			return nil, evt1Diagnostic("C_STYLE_CAST_UNSUPPORTED", "C-style casts are unsupported; use `value as T` or `static_cast<T>(value)`; floating-to-integer conversion requires TruncTo<T>, FloorTo<T>, CeilTo<T>, or RoundTo<T>", start)
		}
		previous := p.suppressAs
		p.suppressAs = false
		expr, err := p.parseExpr()
		p.suppressAs = previous
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(&ParenExpr{Value: expr, Span: start}, start)
	case p.peekLexeme() == "[":
		return p.parseArrayLiteralExpr()
	case p.peekLexeme() == "if":
		return p.parseIfExpr()
	case p.peekLexeme() == "match":
		return p.parseMatchExpr()
	case p.peekLexeme() == "infer":
		return p.parseInferExpr()
	case p.peekLexeme() == "decide":
		return p.parseDecideExpr()
	case p.peekLexeme() == "callback" && p.looksLikeCallableLiteral():
		return p.parseCallableExpr()
	case p.peekLexeme() == "true" || p.peekLexeme() == "false":
		tok := p.next()
		return p.parsePostfixExpr(&BoolLiteral{Value: tok.Lexeme == "true", Span: tok.Span}, tok.Span)
	case strings.HasPrefix(p.peekLexeme(), "\""):
		tok := p.next()
		value, err := strconv.Unquote(tok.Lexeme)
		if err != nil {
			return nil, evt1Diagnostic("CV4000", "invalid string literal", tok.Span)
		}
		return p.parsePostfixExpr(&StringLiteral{Value: value, Span: tok.Span}, tok.Span)
	case isFloatNumber(p.peekLexeme()):
		tok := p.next()
		value, _ := strconv.ParseFloat(tok.Lexeme, 64)
		expr, err := p.quantityLiteral(&FloatLiteral{Value: value, Span: tok.Span}, tok, "float")
		if err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(expr, tok.Span)
	case isNumber(p.peekLexeme()):
		tok := p.next()
		literal, err := evt1ParseIntegerLiteral(tok.Lexeme, false, tok.Span)
		if err != nil {
			return nil, err
		}
		expr, err := p.quantityLiteral(literal, tok, "int")
		if err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(expr, tok.Span)
	default:
		return p.parseNameLikeExpr()
	}
}

func (p *parser) looksLikeCallableLiteral() bool {
	if p.peekLexeme() != "callback" || p.peekLexemeN(1) != "(" {
		return false
	}
	depth := 0
	for i := p.pos + 1; i < len(p.tokens); i++ {
		switch p.tokens[i].Lexeme {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i+1 < len(p.tokens) && (p.tokens[i+1].Lexeme == "with" || p.tokens[i+1].Lexeme == "{")
			}
		}
	}
	return false
}

func (p *parser) parseCallableExpr() (Expr, error) {
	start, _ := p.expect("callback")
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	expr := &CallableExpr{Ordinal: p.callableOrdinal, Span: start.Span}
	p.callableOrdinal++
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType("")
			if err != nil {
				return nil, err
			}
			name, err := p.expectIdentifier("CALLABLE_PARAMETER_INVALID", "expected callable parameter name")
			if err != nil {
				return nil, err
			}
			expr.Params = append(expr.Params, Param{Type: paramType, Name: name.Lexeme, Span: name.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if p.peekLexeme() == "with" {
		p.next()
		if _, err := p.expect("("); err != nil {
			return nil, evt1Diagnostic("CALLABLE_CAPTURE_INVALID", "callable `with` requires a parenthesized capture list", p.currentSpan())
		}
		for p.peekLexeme() != ")" {
			capture := CaptureBinding{Kind: CaptureCopy, Ordinal: len(expr.Captures), Span: p.currentSpan()}
			if p.peekLexeme() == "move" {
				capture.Kind = CaptureMove
				p.next()
			} else if p.peekLexeme() == "ref" {
				capture.Kind = CaptureRef
				p.next()
				if p.peekLexeme() == "const" {
					capture.Kind = CaptureRefConst
					p.next()
				}
			}
			name, err := p.expectIdentifier("CALLABLE_CAPTURE_INVALID", "capture source must be an identifier or self")
			if err != nil {
				return nil, err
			}
			capture.Name, capture.Span = name.Lexeme, name.Span
			if p.peekLexeme() == "=" {
				if capture.Kind != CaptureCopy {
					return nil, evt1Diagnostic("CALLABLE_CAPTURE_INVALID", "capture aliases use copy initialization; move/ref aliases are deferred", capture.Span)
				}
				p.next()
				source, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				capture.Source = source
			}
			expr.Captures = append(expr.Captures, capture)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	expr.Body = body
	return p.parsePostfixExpr(expr, start.Span)
}

func (p *parser) parseInferExpr() (Expr, error) {
	start := p.next().Span
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	expr := &InferExpr{Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		candidate, err := p.parseScoredCandidate("INFER_UNKNOWN_CANDIDATE", "expected inference candidate", len(expr.Candidates))
		if err != nil {
			return nil, err
		}
		expr.Candidates = append(expr.Candidates, candidate)
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return p.parsePostfixExpr(expr, start)
}

func (p *parser) parseDecideExpr() (Expr, error) {
	start := p.next().Span
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	expr := &DecideExpr{Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		candidate, err := p.parseScoredCandidate("DECIDE_UNKNOWN_CANDIDATE", "expected decision candidate", len(expr.Candidates))
		if err != nil {
			return nil, err
		}
		expr.Candidates = append(expr.Candidates, candidate)
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return p.parsePostfixExpr(expr, start)
}

func (p *parser) parseWhileStmt() (Statement, error) {
	start, err := p.expect("while")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	stmt := &WhileStmt{Condition: condition, Span: start.Span}
	if p.peekLexeme() == "bounded" {
		p.next()
		if _, err := p.expect("("); err != nil {
			return nil, err
		}
		bound, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		stmt.Bound = bound
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt.Body = body
	if p.peekLexeme() == "else" {
		p.next()
		if stmt.Bound == nil {
			return nil, evt1Diagnostic("CV4205", "while else requires bounded(limit)", p.currentSpan())
		}
		exhausted, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Else = &exhausted
	}
	return stmt, nil
}

func (p *parser) parseMatchExpr() (Expr, error) {
	start := p.next().Span
	if p.peekLexeme() == "{" {
		return p.parseGuardedMatchExpr(start)
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	subject, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	expr := &MatchExpr{Subject: subject, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		pattern, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect("=>"); err != nil {
			return nil, err
		}
		if p.peekLexeme() == "{" {
			return nil, evt1Diagnostic("CV4117", "expression-form match arms require a single expression, not a statement block", p.currentSpan())
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		expr.Arms = append(expr.Arms, ExprArm{Pattern: pattern, Value: value, Span: pattern.Span})
		if p.peekLexeme() == "," {
			p.next()
		} else if p.peekLexeme() != "}" {
			return nil, evt1Diagnostic("CV4014", "expression-form match arms must be comma-separated", p.currentSpan())
		}
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *parser) parseNameLikeExpr() (Expr, error) {
	if p.peekLexeme() == "dispatch" && p.peekLexemeN(1) == "(" {
		return nil, evt1RemovedSurface("dispatch", p.currentSpan())
	}
	nameTok, err := p.expectIdentifier("CV4015", "expected expression")
	if err != nil {
		return nil, err
	}
	switch nameTok.Lexeme {
	case "std":
		if p.peekLexeme() == "::" && p.peekLexemeN(1) == "move" {
			return nil, evt1Diagnostic("STD_MOVE_UNSUPPORTED", "Concept uses `move value` for explicit ownership transfer", nameTok.Span)
		}
	case "reinterpret_cast":
		return nil, evt1Diagnostic("REINTERPRET_CAST_UNSUPPORTED", "Concept has no general reinterpret_cast; use Storage/bind, Address operations, or an explicit bits/representation API as appropriate", nameTok.Span)
	case "const_cast":
		return nil, evt1Diagnostic("CONST_CAST_UNSUPPORTED", "Concept does not permit mutable authority to be cast into existence; acquire a mutable reference from its owner", nameTok.Span)
	case "dynamic_cast":
		return nil, evt1Diagnostic("DYNAMIC_CAST_UNSUPPORTED", "Concept has no dynamic_cast runtime type conversion", nameTok.Span)
	}
	if nameTok.Lexeme == "static_cast" {
		if _, err := p.expect("<"); err != nil {
			return nil, err
		}
		target, err := p.parseType("")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(">"); err != nil {
			return nil, err
		}
		if _, err := p.expect("("); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
		return p.parsePostfixExpr(&CastExpr{Value: value, Target: target, Span: nameTok.Span}, nameTok.Span)
	}
	constructName := nameTok.Lexeme
	constructType := Type{Name: nameTok.Lexeme, Kind: TypeStruct, Span: nameTok.Span}
	if p.genericConstructionAhead() {
		p.next()
		var args []Type
		for {
			if isNumber(p.peekLexeme()) {
				tok := p.next()
				args = append(args, Type{Name: tok.Lexeme, Kind: TypeTemplateValue, Span: tok.Span})
			} else {
				arg, parseErr := p.parseType("")
				if parseErr != nil {
					return nil, parseErr
				}
				args = append(args, arg)
			}
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(">"); err != nil {
			return nil, err
		}
		constructType = Type{Name: nameTok.Lexeme, Kind: TypeApplied, TypeArgs: args, Span: nameTok.Span}
		constructName = constructType.String()
	}
	var expr Expr = &NameExpr{Name: constructName, Span: nameTok.Span}
	if p.peekLexeme() == "::" {
		p.next()
		variantTok, err := p.expectIdentifier("CV4016", "expected variant name after ::")
		if err != nil {
			return nil, err
		}
		args := []Expr{}
		if p.peekLexeme() == "(" {
			p.next()
			if p.peekLexeme() != ")" {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peekLexeme() != "," {
						break
					}
					p.next()
				}
			}
			if _, err := p.expect(")"); err != nil {
				return nil, err
			}
		}
		expr = &ConstructExpr{EnumName: nameTok.Lexeme, VariantName: variantTok.Lexeme, Args: args, Span: nameTok.Span}
	}
	if p.peekLexeme() == "{" {
		p.next()
		args := []Expr{}
		argNames := []string{}
		var generatedFields *ForeachStmt
		if p.peekLexeme() == "foreach" {
			if !p.inGeneratorBody {
				return nil, evt1Diagnostic("GENERATOR_AGGREGATE_REQUIRED", "reflected aggregate fields are only valid in a generator", p.currentSpan())
			}
			statement, err := p.parseForeachStmt()
			if err != nil {
				return nil, err
			}
			generatedFields = statement.(*ForeachStmt)
		}
		named := p.peekLexemeN(1) == "="
		if p.peekLexeme() != "}" {
			for {
				if named {
					name, err := p.expectIdentifier("AGGREGATE_FIELD_NAME_REQUIRED", "expected aggregate field name")
					if err != nil {
						return nil, err
					}
					if _, err := p.expect("="); err != nil {
						return nil, err
					}
					argNames = append(argNames, name.Lexeme)
				} else if p.peekLexemeN(1) == "=" {
					return nil, evt1Diagnostic("AGGREGATE_INITIALIZER_MIXED", "cannot mix positional and named aggregate fields", p.currentSpan())
				}
				arg, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
		}
		if _, err := p.expect("}"); err != nil {
			return nil, err
		}
		expr = &StructConstructExpr{StructName: constructName, StructType: constructType, Args: args, ArgNames: argNames, GeneratedFields: generatedFields, Span: nameTok.Span}
	}
	return p.parsePostfixExpr(expr, nameTok.Span)
}

func (p *parser) genericConstructionAhead() bool {
	if p.peekLexeme() != "<" {
		return false
	}
	depth := 0
	for i := p.pos; i < len(p.tokens); i++ {
		switch p.tokens[i].Lexeme {
		case "<":
			depth++
		case ">":
			depth--
			if depth == 0 {
				return i+1 < len(p.tokens) && p.tokens[i+1].Lexeme == "{"
			}
		}
	}
	return false
}

func (p *parser) parseArrayLiteralExpr() (Expr, error) {
	start, err := p.expect("[")
	if err != nil {
		return nil, err
	}
	lit := &ArrayLiteralExpr{Span: start.Span}
	if p.peekLexeme() != "]" {
		for {
			element, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.peekLexeme() == "..." {
				ellipsis := p.next()
				repeat := &RepeatInitializer{Value: element, FillRemainder: true, Span: ellipsis.Span}
				if p.peekLexeme() != "," && p.peekLexeme() != "]" {
					count, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					repeat.Count = count
					repeat.FillRemainder = false
				}
				element = repeat
			}
			lit.Elements = append(lit.Elements, element)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
			if p.peekLexeme() == "]" {
				break
			}
		}
	}
	if _, err := p.expect("]"); err != nil {
		return nil, err
	}
	return p.parsePostfixExpr(lit, start.Span)
}

func (p *parser) parsePostfixExpr(expr Expr, span Span) (Expr, error) {
	for {
		switch p.peekLexeme() {
		case "as":
			if p.suppressAs {
				return expr, nil
			}
			operator := p.next()
			target, err := p.parseType("")
			if err != nil {
				return nil, err
			}
			expr = &CastExpr{Value: expr, Target: target, Span: operator.Span}
		case "?", "!":
			op := p.next()
			failure := &FailureExpr{Op: op.Lexeme, Value: expr, Span: op.Span}
			if op.Lexeme == "?" && p.peekLexeme() == "else" {
				p.next()
				remap, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				failure.Else = remap
			}
			expr = failure
		case "<":
			if fieldExpr, ok := expr.(*FieldExpr); ok {
				owner, ownerOK := fieldExpr.Receiver.(*NameExpr)
				if ownerOK && owner.Name == "Assert" && fieldExpr.Field == "Concept" {
					return p.parseConceptAssertionCall(fieldExpr, span)
				}
			}
			nameExpr, ok := expr.(*NameExpr)
			if !ok || !p.looksLikeTemplateInvocation() {
				return expr, nil
			}
			p.next()
			var typeArgs []Type
			for {
				var typeArg Type
				var err error
				if isNumber(p.peekLexeme()) {
					tok := p.next()
					typeArg = Type{Name: tok.Lexeme, Kind: TypeTemplateValue, Span: tok.Span}
				} else {
					typeArg, err = p.parseType("")
					if err != nil {
						return nil, err
					}
				}
				typeArgs = append(typeArgs, typeArg)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
			if _, err := p.expect(">"); err != nil {
				return nil, err
			}
			if _, err := p.expect("("); err != nil {
				return nil, err
			}
			var args []Expr
			if p.peekLexeme() != ")" {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peekLexeme() != "," {
						break
					}
					p.next()
				}
			}
			if _, err := p.expect(")"); err != nil {
				return nil, err
			}
			expr = &TemplateCallExpr{Callee: nameExpr.Name, TypeArg: typeArgs[0], TypeArgs: typeArgs, Args: args, Span: span}
		case "(":
			nameExpr, nameCall := expr.(*NameExpr)
			fieldExpr, memberCall := expr.(*FieldExpr)
			if !nameCall && !memberCall {
				return nil, evt1Diagnostic("CV4017", "call target must be a function or member", p.currentSpan())
			}
			p.next()
			var args []Expr
			if p.peekLexeme() != ")" {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peekLexeme() != "," {
						break
					}
					p.next()
				}
			}
			if _, err := p.expect(")"); err != nil {
				return nil, err
			}
			if memberCall {
				expr = &CallExpr{Callee: fieldExpr.Field, Receiver: fieldExpr.Receiver, Member: true, Args: args, Span: fieldExpr.Span}
			} else {
				expr = &CallExpr{Callee: nameExpr.Name, Args: args, Span: span}
			}
		case ".":
			p.next()
			fieldTok, err := p.expectIdentifier("CV4018", "expected field name after .")
			if err != nil {
				return nil, err
			}
			expr = &FieldExpr{Receiver: expr, Field: fieldTok.Lexeme, Span: fieldTok.Span}
		case "[":
			p.next()
			var indices []Expr
			for {
				index, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				indices = append(indices, index)
				if p.peekLexeme() != "," {
					break
				}
				p.next()
			}
			if _, err := p.expect("]"); err != nil {
				return nil, err
			}
			expr = &IndexExpr{Base: expr, Index: indices[0], Indices: indices, Span: span}
		default:
			return expr, nil
		}
	}
}

func (p *parser) parseConceptAssertionCall(field *FieldExpr, span Span) (Expr, error) {
	p.next() // <
	goal, _, err := p.parseQualifiedConceptName("CONCEPT_ASSERT_GOAL_INVALID", "Assert.Concept requires a concept or compiler analysis name")
	if err != nil {
		return nil, err
	}
	parameters := []int{}
	if p.peekLexeme() == "<" {
		p.next()
		for {
			parameter := p.current()
			if !isNumber(parameter.Lexeme) {
				return nil, evt1Diagnostic("CONCEPT_ASSERT_PARAMETER_INVALID", "analysis parameter must be a positive integer", parameter.Span)
			}
			value64, parseErr := strconv.ParseInt(parameter.Lexeme, 0, 32)
			if parseErr != nil || value64 <= 0 {
				return nil, evt1Diagnostic("CONCEPT_ASSERT_PARAMETER_INVALID", "analysis parameter must be a positive 32-bit integer", parameter.Span)
			}
			parameters = append(parameters, int(value64))
			p.next()
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
		if _, err := p.expect(">"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(">"); err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	var args []Expr
	if p.peekLexeme() != ")" {
		for {
			arg, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	return &CallExpr{Callee: "Concept", Receiver: field.Receiver, Member: true, Args: args, ConceptGoal: goal, ConceptParameters: parameters, Span: span}, nil
}

func (p *parser) looksLikeTemplateInvocation() bool {
	if p.peekLexeme() != "<" {
		return false
	}
	save := p.pos
	defer func() { p.pos = save }()
	p.next()
	for {
		if isNumber(p.peekLexeme()) {
			p.next()
		} else if _, err := p.parseType(""); err != nil {
			return false
		}
		if p.peekLexeme() != "," {
			break
		}
		p.next()
	}
	if p.peekLexeme() != ">" {
		return false
	}
	p.next()
	return p.peekLexeme() == "("
}

func (p *parser) expectKeyword(keyword string) error {
	if p.peekLexeme() != keyword {
		return evt1Diagnostic("CV4019", fmt.Sprintf("expected %s", keyword), p.currentSpan())
	}
	p.next()
	return nil
}

func (p *parser) expectIdentifier(code, message string) (Token, error) {
	tok := p.current()
	if !isIdentifier(tok.Lexeme) {
		return Token{}, evt1Diagnostic(code, message, tok.Span)
	}
	p.next()
	return tok, nil
}

func (p *parser) expect(lexeme string) (Token, error) {
	tok := p.current()
	if tok.Lexeme != lexeme {
		return Token{}, evt1Diagnostic("CV4020", fmt.Sprintf("expected %q", lexeme), tok.Span)
	}
	p.next()
	return tok, nil
}

func (p *parser) isConceptApplicationAhead(conceptParam string) bool {
	if !isIdentifier(p.peekLexeme()) {
		return false
	}
	save := p.pos
	_, _, _, err := p.parseConceptApplication(conceptParam)
	isRequirement := err == nil && p.peekLexeme() == ";"
	p.pos = save
	return isRequirement
}

func (p *parser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *parser) current() Token {
	if p.done() {
		if len(p.tokens) == 0 {
			return Token{Span: Span{Line: 1, Column: 1}}
		}
		last := p.tokens[len(p.tokens)-1]
		return Token{Span: Span{Line: last.Span.Line, Column: last.Span.Column + len(last.Lexeme)}}
	}
	return p.tokens[p.pos]
}

func (p *parser) currentSpan() Span {
	return p.current().Span
}

func (p *parser) peekLexeme() string {
	return p.peekLexemeN(0)
}

func (p *parser) peekLexemeN(offset int) string {
	if p.pos+offset >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos+offset].Lexeme
}

func (p *parser) next() Token {
	tok := p.current()
	p.pos++
	return tok
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
		return false
	}
	for i := 1; i < len(s); i++ {
		c = s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func isNumber(s string) bool {
	s = strings.TrimSuffix(s, "u")
	if s == "" {
		return false
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		if len(s) == 2 {
			return false
		}
		for i := 2; i < len(s); i++ {
			c := s[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isFloatNumber(s string) bool {
	return strings.ContainsAny(s, ".eE") && func() bool {
		_, err := strconv.ParseFloat(s, 64)
		return err == nil
	}()
}

// Adjacency is intentional: 1m is a quantity literal; 1 m remains two
// independent tokens and is rejected by the ordinary expression grammar.
func (p *parser) quantityLiteral(value Expr, number Token, representation string) (Expr, error) {
	unitToken := p.current()
	if unitToken.Span.Line != number.Span.Line || unitToken.Span.Column != number.Span.Column+len(number.Lexeme) || !isIdentifier(unitToken.Lexeme) {
		return value, nil
	}
	unit, ok := quantityFromUnit(unitToken.Lexeme)
	if !ok {
		return nil, quantityUnknownDiagnostic(unitToken)
	}
	p.next()
	target := evt1QuantityType(representation, unit, number.Span)
	return &TemplateCallExpr{Callee: "AssumeQuantity", TypeArg: target, TypeArgs: []Type{target}, Args: []Expr{value}, Span: number.Span}, nil
}

func quantityUnknownDiagnostic(token Token) error {
	message := fmt.Sprintf("unit '%s' is not in the standard unit catalog; declare/import a domain-specific unit or use a catalogued unit with scientific scaling", token.Lexeme)
	if token.Lexeme == "dm" {
		message = "unit 'dm' is not in the standard unit catalog; use scientific scaling such as 1e-1m or declare/import a domain-specific unit"
	}
	return evt1Diagnostic("QUANTITY_UNIT_UNKNOWN", message, token.Span)
}

func isUnsupportedStandardUnit(name string) bool {
	switch name {
	case "dm", "dam", "hm", "daN":
		return true
	}
	return false
}

// A guarded match has no subject: its arms are `when condition => ...`,
// tried in order, and `otherwise => ...` last. It reads as a decision table
// and means exactly an if/else chain, which is what it becomes: the
// expression form an IfExpr chain, the statement form an IfStmt chain.
type guardedArm struct {
	condition Expr
	value     Expr
	block     Block
	span      Span
}

func (p *parser) parseGuardedArms(statement bool) ([]guardedArm, *guardedArm, error) {
	if _, err := p.expect("{"); err != nil {
		return nil, nil, err
	}
	var arms []guardedArm
	var otherwise *guardedArm
	for !p.done() && p.peekLexeme() != "}" {
		if otherwise != nil {
			return nil, nil, evt1Diagnostic("MATCH_GUARD_ORDER", "otherwise is the last arm of a guarded match", p.currentSpan())
		}
		arm := guardedArm{span: p.currentSpan()}
		switch p.peekLexeme() {
		case "when":
			p.next()
			condition, err := p.parseExpr()
			if err != nil {
				return nil, nil, err
			}
			arm.condition = condition
		case "otherwise":
			p.next()
		default:
			return nil, nil, evt1Diagnostic("MATCH_GUARD_ARM", "a match without a subject has `when condition => ...` arms and a final `otherwise => ...`", p.currentSpan())
		}
		if _, err := p.expect("=>"); err != nil {
			return nil, nil, err
		}
		if statement {
			if p.peekLexeme() != "{" {
				return nil, nil, evt1Diagnostic("CV4118", "statement-form match arms require braced blocks", p.currentSpan())
			}
			block, err := p.parseBlock()
			if err != nil {
				return nil, nil, err
			}
			arm.block = block
			if p.peekLexeme() == "," {
				p.next()
			}
		} else {
			if p.peekLexeme() == "{" {
				return nil, nil, evt1Diagnostic("CV4117", "expression-form match arms require a single expression, not a statement block", p.currentSpan())
			}
			value, err := p.parseExpr()
			if err != nil {
				return nil, nil, err
			}
			arm.value = value
			if p.peekLexeme() == "," {
				p.next()
			} else if p.peekLexeme() != "}" {
				return nil, nil, evt1Diagnostic("CV4014", "expression-form match arms must be comma-separated", p.currentSpan())
			}
		}
		if arm.condition == nil {
			otherwise = &arm
		} else {
			arms = append(arms, arm)
		}
	}
	if _, err := p.expect("}"); err != nil {
		return nil, nil, err
	}
	if len(arms) == 0 {
		return nil, nil, evt1Diagnostic("MATCH_GUARD_ARM", "a guarded match needs at least one `when` arm", p.currentSpan())
	}
	return arms, otherwise, nil
}

func (p *parser) parseGuardedMatchExpr(start Span) (Expr, error) {
	arms, otherwise, err := p.parseGuardedArms(false)
	if err != nil {
		return nil, err
	}
	if otherwise == nil {
		return nil, evt1Diagnostic("MATCH_GUARD_OTHERWISE", "an expression-form guarded match must end with `otherwise => value`, so it always has a value", start)
	}
	result := otherwise.value
	for i := len(arms) - 1; i >= 0; i-- {
		result = &IfExpr{Condition: arms[i].condition, Then: arms[i].value, Else: result, Span: arms[i].span}
	}
	return result, nil
}

func (p *parser) parseGuardedMatchStmt(start Span) (Statement, error) {
	arms, otherwise, err := p.parseGuardedArms(true)
	if err != nil {
		return nil, err
	}
	var rest *Block
	if otherwise != nil {
		block := otherwise.block
		rest = &block
	}
	var result *IfStmt
	for i := len(arms) - 1; i >= 0; i-- {
		result = &IfStmt{Condition: arms[i].condition, Then: arms[i].block, Else: rest, Span: arms[i].span}
		rest = &Block{Statements: []Statement{result}, Span: arms[i].span}
	}
	return result, nil
}
