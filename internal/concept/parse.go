package concept

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type parser struct {
	path       string
	tokens     []Token
	pos        int
	profileDef *ProfileDefinition
}

func Parse(path, text string) (Module, error) {
	tokens, err := lexEVT1(text)
	if err != nil {
		return Module{}, err
	}
	p := &parser{path: filepath.ToSlash(path), tokens: tokens}
	module, err := p.parseModule()
	if err != nil {
		return Module{}, err
	}
	if err := validateModule(module); err != nil {
		return Module{}, err
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
		start := Span{Line: line, Column: column}
		switch {
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
			for j < len(text) && text[j] >= '0' && text[j] <= '9' {
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
		case strings.ContainsRune("(){}[];,:.*+-=<>!", rune(c)):
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
	for !p.done() {
		switch p.peekLexeme() {
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
			templateDecl, err := p.parseTemplateDecl()
			if err != nil {
				return module, err
			}
			module.Templates = append(module.Templates, templateDecl)
		case "immovable":
			structDecl, err := p.parseStructDecl(true, false)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
		case "record":
			structDecl, err := p.parseStructDecl(false, true)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
		case "struct":
			structDecl, err := p.parseStructDecl(false, false)
			if err != nil {
				return module, err
			}
			module.Structs = append(module.Structs, structDecl)
		case "enum":
			enumDecl, err := p.parseEnumDecl()
			if err != nil {
				return module, err
			}
			module.Enums = append(module.Enums, enumDecl)
		case "effect":
			effectDecl, err := p.parseEffectDecl()
			if err != nil {
				return module, err
			}
			module.Effects = append(module.Effects, effectDecl)
		case "actuator":
			actuatorDecl, err := p.parseActuatorDecl()
			if err != nil {
				return module, err
			}
			module.Actuators = append(module.Actuators, actuatorDecl)
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
		case "requires":
			assertion, err := p.parseConceptAssertion()
			if err != nil {
				return module, err
			}
			module.Assertions = append(module.Assertions, assertion)
		default:
			fn, err := p.parseFunctionDecl("", false)
			if err != nil {
				return module, err
			}
			module.Functions = append(module.Functions, fn)
		}
	}
	return module, nil
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
	if _, err := p.expect("("); err != nil {
		return AutomataDecl{}, err
	}
	signalType, err := p.parseType("")
	if err != nil {
		return AutomataDecl{}, err
	}
	var context *Field
	if p.peekLexeme() == "," {
		p.next()
		if p.peekLexeme() != "borrow" {
			return AutomataDecl{}, evt1Diagnostic("CV4276", "automata context parameter must use `borrow name: Type`", p.currentSpan())
		}
		p.next()
		nameTok, err := p.expectIdentifier("CV4277", "expected automata context binding name")
		if err != nil {
			return AutomataDecl{}, err
		}
		if _, err := p.expect(":"); err != nil {
			return AutomataDecl{}, err
		}
		contextType, err := p.parseType("")
		if err != nil {
			return AutomataDecl{}, err
		}
		context = &Field{Name: nameTok.Lexeme, Type: contextType, Span: nameTok.Span}
		if p.peekLexeme() == "," {
			return AutomataDecl{}, evt1Diagnostic("CV4278", "automata declarations admit at most one borrowed context parameter", p.currentSpan())
		}
	}
	if _, err := p.expect(")"); err != nil {
		return AutomataDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return AutomataDecl{}, err
	}
	decl := AutomataDecl{Name: nameTok.Lexeme, SignalType: signalType, Context: context, Span: start.Span}
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
	return decl, nil
}

func (p *parser) parseEffectDecl() (EffectDecl, error) {
	start, err := p.expect("effect")
	if err != nil {
		return EffectDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4298", "expected effect name")
	if err != nil {
		return EffectDecl{}, err
	}
	if _, err := p.expect("("); err != nil {
		return EffectDecl{}, err
	}
	decl := EffectDecl{Name: nameTok.Lexeme, Span: start.Span}
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType("")
			if err != nil {
				return EffectDecl{}, err
			}
			paramName, err := p.expectIdentifier("CV4299", "expected effect parameter name")
			if err != nil {
				return EffectDecl{}, err
			}
			decl.Params = append(decl.Params, Param{Type: paramType, Name: paramName.Lexeme, Span: paramName.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return EffectDecl{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return EffectDecl{}, err
	}
	return decl, nil
}

func (p *parser) parseActuatorDecl() (ActuatorDecl, error) {
	start, err := p.expect("actuator")
	if err != nil {
		return ActuatorDecl{}, err
	}
	nameTok, err := p.expectIdentifier("CV4313", "expected actuator name")
	if err != nil {
		return ActuatorDecl{}, err
	}
	if _, err := p.expect("("); err != nil {
		return ActuatorDecl{}, err
	}
	automataTok, err := p.expectIdentifier("CV4314", "expected automata name in actuator declaration")
	if err != nil {
		return ActuatorDecl{}, err
	}
	if _, err := p.expect(","); err != nil {
		return ActuatorDecl{}, err
	}
	mechanismType, err := p.parseType("")
	if err != nil {
		return ActuatorDecl{}, err
	}
	mechanismTok, err := p.expectIdentifier("CV4315", "expected actuator mechanism binding name")
	if err != nil {
		return ActuatorDecl{}, err
	}
	if _, err := p.expect(","); err != nil {
		return ActuatorDecl{}, err
	}
	errorType, err := p.parseType("")
	if err != nil {
		return ActuatorDecl{}, err
	}
	if _, err := p.expect(")"); err != nil {
		return ActuatorDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return ActuatorDecl{}, err
	}
	decl := ActuatorDecl{
		Name:          nameTok.Lexeme,
		AutomataName:  automataTok.Lexeme,
		MechanismType: mechanismType,
		MechanismName: mechanismTok.Lexeme,
		ErrorType:     errorType,
		Span:          start.Span,
	}
	for !p.done() && p.peekLexeme() != "}" {
		mapping, err := p.parseActuatorMapping()
		if err != nil {
			return ActuatorDecl{}, err
		}
		decl.Mappings = append(decl.Mappings, mapping)
	}
	if _, err := p.expect("}"); err != nil {
		return ActuatorDecl{}, err
	}
	return decl, nil
}

func (p *parser) parseActuatorMapping() (ActuatorMapping, error) {
	start, err := p.expect("on")
	if err != nil {
		return ActuatorMapping{}, err
	}
	effectTok, err := p.expectIdentifier("CV4316", "expected effect name in actuator mapping")
	if err != nil {
		return ActuatorMapping{}, err
	}
	if _, err := p.expect("("); err != nil {
		return ActuatorMapping{}, err
	}
	mapping := ActuatorMapping{EffectName: effectTok.Lexeme, Span: start.Span}
	if p.peekLexeme() != ")" {
		for {
			paramType, err := p.parseType("")
			if err != nil {
				return ActuatorMapping{}, err
			}
			paramName, err := p.expectIdentifier("CV4317", "expected actuator mapping parameter name")
			if err != nil {
				return ActuatorMapping{}, err
			}
			mapping.Params = append(mapping.Params, Param{Type: paramType, Name: paramName.Lexeme, Span: paramName.Span})
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return ActuatorMapping{}, err
	}
	if _, err := p.expect("=>"); err != nil {
		return ActuatorMapping{}, err
	}
	callExpr, err := p.parseExpr()
	if err != nil {
		return ActuatorMapping{}, err
	}
	call, ok := callExpr.(*CallExpr)
	if !ok {
		return ActuatorMapping{}, evt1Diagnostic("CV4318", "actuator mappings require one direct implementation call", callExpr.exprSpan())
	}
	mapping.ImplementationName = call.Callee
	mapping.ImplementationArgs = call.Args
	if _, err := p.expect(";"); err != nil {
		return ActuatorMapping{}, err
	}
	return mapping, nil
}

func (p *parser) parseMachineDecl() (MachineDecl, error) {
	start := p.currentSpan()
	machine := MachineDecl{Span: start}
	if p.peekLexeme() == "initial" {
		p.next()
		machine.Initial = true
	}
	if _, err := p.expect("machine"); err != nil {
		return MachineDecl{}, evt1Diagnostic("CV4243", "expected machine declaration", p.currentSpan())
	}
	nameTok, err := p.expectIdentifier("CV4244", "expected machine name")
	if err != nil {
		return MachineDecl{}, err
	}
	machine.Name = nameTok.Lexeme
	if _, err := p.expect("{"); err != nil {
		return MachineDecl{}, err
	}
	for !p.done() && p.peekLexeme() != "}" {
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
		p.next()
		state.Initial = true
		if p.peekLexeme() == "terminal" {
			p.next()
			state.Terminal = true
		}
	} else if p.peekLexeme() == "terminal" {
		p.next()
		state.Terminal = true
		if p.peekLexeme() == "initial" {
			return StateDecl{}, evt1Diagnostic("CV4248", "state modifiers must use `initial terminal state`, not `terminal initial state`", p.currentSpan())
		}
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
	for !p.done() && p.peekLexeme() != "}" {
		switch p.peekLexeme() {
		case "on":
			handler, err := p.parseAutomataHandler()
			if err != nil {
				return StateDecl{}, err
			}
			state.Handlers = append(state.Handlers, handler)
		case "pop", "finish":
			completion, err := p.parseAutomataCompletion()
			if err != nil {
				return StateDecl{}, err
			}
			state.Completion = append(state.Completion, completion)
		default:
			return StateDecl{}, evt1Diagnostic("CV4264", "state bodies only allow `on`, `pop`, or `finish` clauses", p.currentSpan())
		}
	}
	if _, err := p.expect("}"); err != nil {
		return StateDecl{}, err
	}
	return state, nil
}

func (p *parser) parseAutomataHandler() (TransitionDecl, error) {
	start, err := p.expect("on")
	if err != nil {
		return TransitionDecl{}, err
	}
	signal, err := p.parseQualifiedEnumMember()
	if err != nil {
		return TransitionDecl{}, err
	}
	handler := TransitionDecl{Signal: signal, Span: start.Span}
	switch p.peekLexeme() {
	case "when":
		p.next()
		guard, err := p.parseExpr()
		if err != nil {
			return TransitionDecl{}, err
		}
		handler.Guard = guard
		if _, err := p.expect("=>"); err != nil {
			return TransitionDecl{}, evt1Diagnostic("CV4279", "guarded handlers require `=>` before the control action", p.currentSpan())
		}
	case "otherwise":
		p.next()
		handler.Otherwise = true
		if _, err := p.expect("=>"); err != nil {
			return TransitionDecl{}, evt1Diagnostic("CV4280", "fallback handlers require `=>` before the control action", p.currentSpan())
		}
	case "=>":
		p.next()
		if err := p.parseAutomataEffectBody(&handler); err != nil {
			return TransitionDecl{}, err
		}
		return handler, nil
	}
	if p.peekLexeme() == "{" {
		if err := p.parseAutomataEffectBody(&handler); err != nil {
			return TransitionDecl{}, err
		}
		return handler, nil
	}
	if err := p.parseAutomataControlAction(&handler); err != nil {
		return TransitionDecl{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return TransitionDecl{}, err
	}
	return handler, nil
}

func (p *parser) parseAutomataEffectBody(handler *TransitionDecl) error {
	if _, err := p.expect("{"); err != nil {
		return err
	}
	for p.peekLexeme() == "emit" {
		emit, err := p.parseAutomataEmit()
		if err != nil {
			return err
		}
		handler.Emits = append(handler.Emits, emit)
	}
	if err := p.parseAutomataControlAction(handler); err != nil {
		return err
	}
	if _, err := p.expect(";"); err != nil {
		return err
	}
	if _, err := p.expect("}"); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseAutomataEmit() (EmitStmt, error) {
	start, err := p.expect("emit")
	if err != nil {
		return EmitStmt{}, err
	}
	nameTok, err := p.expectIdentifier("CV4301", "expected effect name after emit")
	if err != nil {
		return EmitStmt{}, err
	}
	if _, err := p.expect("("); err != nil {
		return EmitStmt{}, err
	}
	stmt := EmitStmt{EffectName: nameTok.Lexeme, Span: start.Span}
	if p.peekLexeme() != ")" {
		for {
			arg, err := p.parseExpr()
			if err != nil {
				return EmitStmt{}, err
			}
			stmt.Args = append(stmt.Args, arg)
			if p.peekLexeme() != "," {
				break
			}
			p.next()
		}
	}
	if _, err := p.expect(")"); err != nil {
		return EmitStmt{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return EmitStmt{}, err
	}
	return stmt, nil
}

func (p *parser) parseAutomataControlAction(handler *TransitionDecl) error {
	switch p.peekLexeme() {
	case "goto":
		p.next()
		target, err := p.parseStateRef()
		if err != nil {
			return err
		}
		handler.Kind = TransitionGoto
		handler.TargetState = target
		return nil
	case "push":
		p.next()
		machineTok, err := p.expectIdentifier("CV4255", "expected pushed machine name")
		if err != nil {
			return err
		}
		if _, err := p.expect("goto"); err != nil {
			return err
		}
		continuation, err := p.parseStateRef()
		if err != nil {
			return err
		}
		handler.Kind = TransitionPush
		handler.PushMachine = machineTok.Lexeme
		handler.Continuation = continuation
		return nil
	default:
		return evt1Diagnostic("CV4264", "state handlers require `goto` or `push ... goto ...`", p.currentSpan())
	}
}

func (p *parser) parseAutomataCompletion() (CompletionDecl, error) {
	tok := p.next()
	if _, err := p.expect(";"); err != nil {
		return CompletionDecl{}, err
	}
	return CompletionDecl{Kind: tok.Lexeme, Span: tok.Span}, nil
}

func (p *parser) parseQualifiedEnumMember() (QualifiedEnumMember, error) {
	enumTok, err := p.expectIdentifier("CV4252", "expected enum name")
	if err != nil {
		return QualifiedEnumMember{}, err
	}
	if _, err := p.expect("::"); err != nil {
		return QualifiedEnumMember{}, err
	}
	memberTok, err := p.expectIdentifier("CV4252", "expected enum member name")
	if err != nil {
		return QualifiedEnumMember{}, err
	}
	return QualifiedEnumMember{EnumName: enumTok.Lexeme, MemberName: memberTok.Lexeme, Span: enumTok.Span}, nil
}

func (p *parser) parseStateRef() (StateRef, error) {
	first, err := p.expectIdentifier("CV4254", "expected state name")
	if err != nil {
		return StateRef{}, err
	}
	ref := StateRef{StateName: first.Lexeme, Span: first.Span}
	if p.peekLexeme() == "::" {
		p.next()
		second, err := p.expectIdentifier("CV4254", "expected qualified state name")
		if err != nil {
			return StateRef{}, err
		}
		ref.MachineName = first.Lexeme
		ref.StateName = second.Lexeme
	}
	return ref, nil
}

func (p *parser) parseTemplateDecl() (TemplateDecl, error) {
	start, err := p.expect("template")
	if err != nil {
		return TemplateDecl{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return TemplateDecl{}, err
	}
	if _, err := p.expect("typename"); err != nil {
		return TemplateDecl{}, evt1Diagnostic("CV4165", "template declarations require exactly `template <typename T>`", p.currentSpan())
	}
	paramTok, err := p.expectIdentifier("CV4165", "expected one template type parameter")
	if err != nil {
		return TemplateDecl{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return TemplateDecl{}, err
	}
	reqTok, err := p.expect("requires")
	if err != nil {
		return TemplateDecl{}, evt1Diagnostic("CV4166", "template declarations require exactly one named concept constraint", p.currentSpan())
	}
	ref, err := p.parseConceptUse(paramTok.Lexeme)
	if err != nil {
		return TemplateDecl{}, err
	}
	fn, err := p.parseFunctionDecl(paramTok.Lexeme, false)
	if err != nil {
		return TemplateDecl{}, err
	}
	if fn.Body == nil {
		return TemplateDecl{}, evt1Diagnostic("CV4167", "template declarations require a function body", fn.Span)
	}
	return TemplateDecl{
		Name:          fn.Name,
		TypeParam:     paramTok.Lexeme,
		TypeParamSpan: paramTok.Span,
		Constraint: TemplateConstraint{
			ConceptName: ref.Name,
			TypeArg:     ref.TypeArgs[0],
			Span:        reqTok.Span,
		},
		ReturnType: fn.ReturnType,
		Params:     fn.Params,
		Body:       fn.Body,
		Span:       start.Span,
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

func (p *parser) parseStructDecl(immovable, record bool) (StructDecl, error) {
	start := p.currentSpan()
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
	decl := StructDecl{Name: nameTok.Lexeme, Immovable: immovable, Record: record, Span: start, RecordSpan: recordSpan}
	for !p.done() && p.peekLexeme() != "}" {
		fieldType, err := p.parseType("")
		if err != nil {
			return StructDecl{}, err
		}
		fieldName, err := p.expectIdentifier("CV4121", "expected field name")
		if err != nil {
			return StructDecl{}, err
		}
		if _, err := p.expect(";"); err != nil {
			return StructDecl{}, err
		}
		decl.Fields = append(decl.Fields, Field{Type: fieldType, Name: fieldName.Lexeme, Span: fieldName.Span})
	}
	if _, err := p.expect("}"); err != nil {
		return StructDecl{}, err
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
		variantTok, err := p.expectIdentifier("CV4004", "expected enum variant name")
		if err != nil {
			return EnumDecl{}, err
		}
		var payload []Field
		if p.peekLexeme() == "(" {
			p.next()
			if p.peekLexeme() != ")" {
				for {
					fieldType, err := p.parseType("")
					if err != nil {
						return EnumDecl{}, err
					}
					fieldName, err := p.expectIdentifier("CV4005", "expected payload name")
					if err != nil {
						return EnumDecl{}, err
					}
					payload = append(payload, Field{Type: fieldType, Name: fieldName.Lexeme, Span: fieldName.Span})
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
		enumDecl.Variants = append(enumDecl.Variants, VariantDecl{Name: variantTok.Lexeme, Payload: payload, Tag: len(enumDecl.Variants), Span: variantTok.Span})
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
	paramTok, err := p.expectIdentifier("CV4141", "expected one concept type parameter")
	if err != nil {
		return ConceptDecl{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return ConceptDecl{}, err
	}
	if _, err := p.expect("{"); err != nil {
		return ConceptDecl{}, err
	}
	decl := ConceptDecl{Name: nameTok.Lexeme, TypeParam: paramTok.Lexeme, Span: start}
	for !p.done() && p.peekLexeme() != "}" {
		req, err := p.parseConceptRequirement(paramTok.Lexeme)
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

func (p *parser) parseConceptRequirement(typeParam string) (ConceptRequirement, error) {
	start, err := p.expect("requires")
	if err != nil {
		return nil, err
	}
	if p.peekLexeme() == "" {
		return nil, evt1Diagnostic("CV4142", "expected concept requirement", start.Span)
	}
	if p.isConceptApplicationAhead(typeParam) {
		ref, err := p.parseConceptUse(typeParam)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &PrerequisiteRequirement{ConceptName: ref.Name, TypeArg: ref.TypeArgs[0], Span: start.Span}, nil
	}
	retType, err := p.parseType(typeParam)
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4143", "expected required operation name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	req := &OperationRequirement{ReturnType: retType, Name: nameTok.Lexeme, Span: start.Span}
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
	return req, nil
}

func (p *parser) parseConceptAssertion() (ConceptAssertion, error) {
	start, err := p.expect("requires")
	if err != nil {
		return ConceptAssertion{}, err
	}
	ref, err := p.parseConceptUse("")
	if err != nil {
		return ConceptAssertion{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return ConceptAssertion{}, err
	}
	return ConceptAssertion{ConceptName: ref.Name, ConcreteType: ref.TypeArgs[0], Span: start.Span}, nil
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
	for {
		switch p.peekLexeme() {
		case "unsafe":
			t.Unsafe = true
			p.next()
		case "imported":
			t.Imported = true
			p.next()
		case "owned":
			t.Ownership = "owned"
			p.next()
		case "borrow":
			t.Ownership = "borrow"
			p.next()
		case "const":
			t.Const = true
			p.next()
		default:
			goto done
		}
	}
done:
	nameTok, err := p.expectIdentifier("CV4008", "expected type name")
	if err != nil {
		return Type{}, err
	}
	if builtin, ok := p.profileDef.builtinType(nameTok.Lexeme, nameTok.Span); ok {
		t.Name = builtin.Name
		t.Kind = builtin.Kind
	} else if conceptParam != "" && nameTok.Lexeme == conceptParam {
		t.Name = nameTok.Lexeme
		t.Kind = TypeConceptParam
	} else {
		t.Name = nameTok.Lexeme
		t.Kind = TypeStruct
	}
	if p.peekLexeme() == "<" {
		p.next()
		for {
			arg, err := p.parseType(conceptParam)
			if err != nil {
				return Type{}, err
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
		lengthExpr, err := p.parseExpr()
		if err != nil {
			return Type{}, err
		}
		if _, err := p.expect("]"); err != nil {
			return Type{}, err
		}
		elem := t
		t = Type{
			Name:            elem.String() + "[]",
			Kind:            TypeArray,
			ArrayElem:       &elem,
			ArrayLengthExpr: lengthExpr,
			Span:            nameTok.Span,
		}
	}
	return t, nil
}

func (p *parser) parseConceptUse(conceptParam string) (Type, error) {
	nameTok, err := p.expectIdentifier("CV4145", "expected concept name")
	if err != nil {
		return Type{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return Type{}, err
	}
	arg, err := p.parseType(conceptParam)
	if err != nil {
		return Type{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return Type{}, err
	}
	return Type{Name: nameTok.Lexeme, Kind: TypeApplied, TypeArgs: []Type{arg}, Span: nameTok.Span}, nil
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
	case "for":
		return nil, evt1Diagnostic("CV4237", "for loops are not supported in EVT1 M1B-D; use while (...) bounded(N)", p.currentSpan())
	case "comptime":
		return p.parseLocalComptimeDecl()
	case "static_assert":
		assertion, err := p.parseStaticAssert()
		if err != nil {
			return nil, err
		}
		return &StaticAssertStmt{Condition: assertion.Condition, Message: assertion.Message, Span: assertion.Span}, nil
	case "effects":
		return p.parseEffectsDecl()
	case "actuation":
		return p.parseActuationDecl()
	case "actuator":
		return p.parseActuatorLocalDecl()
	case "instance":
		return p.parseInstanceDecl()
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
	case "match":
		return p.parseMatchStmt()
	case "while":
		return p.parseWhileStmt()
	case "{":
		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &block, nil
	default:
		if p.peekLexeme() == "const" || p.peekLexeme() == "let" || p.looksLikeVarDecl() {
			return p.parseVarDecl()
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peekLexeme() == "=" {
			p.next()
			rhs, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(";"); err != nil {
				return nil, err
			}
			return &AssignStmt{Target: value, Value: rhs, Span: value.exprSpan()}, nil
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
		return &ExprStmt{Value: value, Span: value.exprSpan()}, nil
	}
}

func (p *parser) parseEffectsDecl() (Statement, error) {
	start, err := p.expect("effects")
	if err != nil {
		return nil, err
	}
	automataTok, err := p.expectIdentifier("CV4300", "expected automata name after effects")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4300", "expected local effects name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &EffectsDecl{AutomataName: automataTok.Lexeme, Name: nameTok.Lexeme, Span: start.Span}, nil
}

func (p *parser) parseActuatorLocalDecl() (Statement, error) {
	start, err := p.expect("actuator")
	if err != nil {
		return nil, err
	}
	actuatorTok, err := p.expectIdentifier("CV4319", "expected actuator name after actuator")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4319", "expected local actuator name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	mechanism, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &ActuatorLocalDecl{ActuatorName: actuatorTok.Lexeme, Name: nameTok.Lexeme, Mechanism: mechanism, Span: start.Span}, nil
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
	var context Expr
	if p.peekLexeme() == "(" {
		p.next()
		context, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(")"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &InstanceDecl{AutomataName: automataTok.Lexeme, Name: nameTok.Lexeme, Context: context, Span: start.Span}, nil
}

func (p *parser) parseActuationDecl() (Statement, error) {
	start, err := p.expect("actuation")
	if err != nil {
		return nil, err
	}
	actuatorTok, err := p.expectIdentifier("CV4320", "expected actuator name after actuation")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4320", "expected actuation result name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("="); err != nil {
		return nil, err
	}
	if _, err := p.expect("actuate"); err != nil {
		return nil, evt1Diagnostic("CV4321", "actuation declarations require actuate(batch, executor)", p.currentSpan())
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	batchTok, err := p.expectIdentifier("CV4321", "expected effects batch name in actuate")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(","); err != nil {
		return nil, err
	}
	executorTok, err := p.expectIdentifier("CV4321", "expected actuator local name in actuate")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	return &ActuationDecl{
		ActuatorName: actuatorTok.Lexeme,
		Name:         nameTok.Lexeme,
		BatchName:    batchTok.Lexeme,
		ExecutorName: executorTok.Lexeme,
		Span:         start.Span,
	}, nil
}

func (p *parser) parseLocalComptimeDecl() (Statement, error) {
	decl, err := p.parseComptimeDecl()
	if err != nil {
		return nil, err
	}
	return &VarDecl{Comptime: true, Type: decl.Type, Name: decl.Name, Value: decl.Value, Span: decl.Span}, nil
}

func (p *parser) looksLikeVarDecl() bool {
	if p.done() {
		return false
	}
	save := p.pos
	defer func() { p.pos = save }()
	if p.peekLexeme() == "const" || p.peekLexeme() == "let" {
		p.next()
	}
	if _, err := p.parseType(""); err != nil {
		return false
	}
	if !isIdentifier(p.peekLexeme()) {
		return false
	}
	p.next()
	return p.peekLexeme() == "="
}

func (p *parser) parseVarDecl() (Statement, error) {
	isConst := false
	constSpan := Span{}
	if p.peekLexeme() == "const" || p.peekLexeme() == "let" {
		isConst = true
		constSpan = p.next().Span
	}
	t, err := p.parseType("")
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expectIdentifier("CV4009", "expected local name")
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
	return &VarDecl{Const: isConst, Type: t, Name: nameTok.Lexeme, Value: value, Span: nameTok.Span, ConstSpan: constSpan}, nil
}

func (p *parser) parseMatchStmt() (Statement, error) {
	start := p.next().Span
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
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "and" {
		op := p.next()
		right, err := p.parseEquality()
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
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.peekLexeme() == "<" || p.peekLexeme() == ">" || p.peekLexeme() == "<=" || p.peekLexeme() == ">=" {
		op := p.next()
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
	for p.peekLexeme() == "*" {
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
	if p.peekLexeme() == "-" || p.peekLexeme() == "not" {
		op := p.next()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Lexeme, Value: value, Span: op.Span}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	switch {
	case p.done():
		return nil, evt1Diagnostic("CV4013", "unexpected end of expression", p.currentSpan())
	case p.peekLexeme() == "(":
		start := p.next().Span
		expr, err := p.parseExpr()
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
	case isNumber(p.peekLexeme()):
		tok := p.next()
		value, _ := strconv.Atoi(tok.Lexeme)
		return p.parsePostfixExpr(&IntLiteral{Value: value, Span: tok.Span}, tok.Span)
	default:
		return p.parseNameLikeExpr()
	}
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
	return stmt, nil
}

func (p *parser) parseMatchExpr() (Expr, error) {
	start := p.next().Span
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
	if p.peekLexeme() == "dispatch" {
		return p.parseDispatchExpr()
	}
	nameTok, err := p.expectIdentifier("CV4015", "expected expression")
	if err != nil {
		return nil, err
	}
	var expr Expr = &NameExpr{Name: nameTok.Lexeme, Span: nameTok.Span}
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
		if p.peekLexeme() != "}" {
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
		if _, err := p.expect("}"); err != nil {
			return nil, err
		}
		expr = &StructConstructExpr{StructName: nameTok.Lexeme, Args: args, Span: nameTok.Span}
	}
	return p.parsePostfixExpr(expr, nameTok.Span)
}

func (p *parser) parseDispatchExpr() (Expr, error) {
	start, err := p.expect("dispatch")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("("); err != nil {
		return nil, err
	}
	instanceTok, err := p.expectIdentifier("CV4273", "expected local instance name in dispatch")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(","); err != nil {
		return nil, err
	}
	signal, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	batchName := ""
	if p.peekLexeme() == "," {
		p.next()
		batchTok, err := p.expectIdentifier("CV4302", "expected local effects batch name in dispatch")
		if err != nil {
			return nil, err
		}
		batchName = batchTok.Lexeme
	}
	if _, err := p.expect(")"); err != nil {
		return nil, err
	}
	return p.parsePostfixExpr(&DispatchExpr{InstanceName: instanceTok.Lexeme, Signal: signal, BatchName: batchName, Span: start.Span}, start.Span)
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
		case "<":
			nameExpr, ok := expr.(*NameExpr)
			if !ok || !p.looksLikeTemplateInvocation() {
				return expr, nil
			}
			p.next()
			typeArg, err := p.parseType("")
			if err != nil {
				return nil, err
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
			expr = &TemplateCallExpr{Callee: nameExpr.Name, TypeArg: typeArg, Args: args, Span: span}
		case "(":
			nameExpr, ok := expr.(*NameExpr)
			if !ok {
				return nil, evt1Diagnostic("CV4017", "only simple function calls are supported in EVT1 expressions", p.currentSpan())
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
			expr = &CallExpr{Callee: nameExpr.Name, Args: args, Span: span}
		case ".":
			p.next()
			fieldTok, err := p.expectIdentifier("CV4018", "expected field name after .")
			if err != nil {
				return nil, err
			}
			expr = &FieldExpr{Receiver: expr, Field: fieldTok.Lexeme, Span: fieldTok.Span}
		case "[":
			p.next()
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect("]"); err != nil {
				return nil, err
			}
			expr = &IndexExpr{Base: expr, Index: index, Span: span}
		default:
			return expr, nil
		}
	}
}

func (p *parser) looksLikeTemplateInvocation() bool {
	if p.peekLexeme() != "<" {
		return false
	}
	save := p.pos
	defer func() { p.pos = save }()
	p.next()
	if _, err := p.parseType(""); err != nil {
		return false
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
	if !isIdentifier(p.peekLexeme()) || p.peekLexemeN(1) != "<" {
		return false
	}
	save := p.pos
	_, err := p.parseConceptUse(conceptParam)
	p.pos = save
	return err == nil
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
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
