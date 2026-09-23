package concept

import (
	"fmt"
	"strings"
)

const evt1GeneratedStatementLimit = 4096

// GeneratorDecl retains an ordinary function AST as its declaration payload.
// The declaration is not a runtime function until a derive site materializes it.
type GeneratorDecl struct {
	Name          string       `json:"name"`
	Module        string       `json:"module,omitempty"`
	TypeParameter string       `json:"type_parameter"`
	Function      FunctionDecl `json:"function"`
	Span          Span         `json:"span"`
}

type GenerationRequest struct {
	Generator     string `json:"generator"`
	ReflectedType Type   `json:"reflected_type"`
	Span          Span   `json:"span"`
}

type GeneratedInput struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
	Span Span   `json:"span"`
}

type GeneratedOrigin struct {
	Kind             string           `json:"kind"`
	Identity         string           `json:"identity"`
	Generator        string           `json:"generator"`
	GeneratorModule  string           `json:"generator_module"`
	GeneratorSpan    Span             `json:"generator_span"`
	GenerationModule string           `json:"generation_module"`
	GenerationSite   Span             `json:"generation_site"`
	ReflectedType    Type             `json:"reflected_type"`
	ReflectedModule  string           `json:"reflected_module,omitempty"`
	Inputs           []GeneratedInput `json:"inputs,omitempty"`
}

func evt1WithGeneratedContext(err error, fn FunctionDecl) error {
	if err == nil || fn.Generated == nil {
		return err
	}
	origin := fn.Generated
	context := fmt.Sprintf("generated %s by %s at %d:%d from reflect<%s>", fn.Name, origin.Generator, origin.GenerationSite.Line, origin.GenerationSite.Column, origin.ReflectedType.String())
	if diagnostic, ok := err.(Diagnostic); ok {
		diagnostic.Message += " (" + context + ")"
		return diagnostic
	}
	return fmt.Errorf("%s: %w", context, err)
}

func (p *parser) parseGeneratorDecl() (GeneratorDecl, error) {
	start, err := p.expect("generator")
	if err != nil {
		return GeneratorDecl{}, err
	}
	if _, err := p.expect("<"); err != nil {
		return GeneratorDecl{}, err
	}
	if _, err := p.expect("typename"); err != nil {
		return GeneratorDecl{}, evt1Diagnostic("GENERATOR_PARAMETER_INVALID", "generator requires one typename parameter", p.currentSpan())
	}
	parameter, err := p.expectIdentifier("GENERATOR_PARAMETER_INVALID", "expected generator type parameter")
	if err != nil {
		return GeneratorDecl{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return GeneratorDecl{}, err
	}
	name, err := p.expectIdentifier("GENERATOR_NAME_INVALID", "expected generator name")
	if err != nil {
		return GeneratorDecl{}, err
	}
	old := p.templateTypeParams
	p.templateTypeParams = map[string]bool{parameter.Lexeme: true}
	previousGenerator := p.inGeneratorBody
	p.inGeneratorBody = true
	defer func() { p.templateTypeParams, p.inGeneratorBody = old, previousGenerator }()
	function, err := p.parseFunctionDecl(parameter.Lexeme, false)
	if err != nil {
		return GeneratorDecl{}, err
	}
	if function.Body == nil {
		return GeneratorDecl{}, evt1Diagnostic("GENERATOR_BODY_REQUIRED", "generator function requires a body", function.Span)
	}
	return GeneratorDecl{Name: name.Lexeme, TypeParameter: parameter.Lexeme, Function: function, Span: start.Span}, nil
}

func (p *parser) parseGenerationRequest() (GenerationRequest, error) {
	start, err := p.expect("derive")
	if err != nil {
		return GenerationRequest{}, err
	}
	name, err := p.expectIdentifier("GENERATOR_NAME_INVALID", "expected generator name")
	if err != nil {
		return GenerationRequest{}, err
	}
	if _, err := p.expect("reflect"); err != nil {
		return GenerationRequest{}, evt1Diagnostic("GENERATOR_REFLECT_REQUIRED", "derive requires explicit reflect<T>", p.currentSpan())
	}
	if _, err := p.expect("<"); err != nil {
		return GenerationRequest{}, err
	}
	target, err := p.parseType("")
	if err != nil {
		return GenerationRequest{}, err
	}
	if _, err := p.expect(">"); err != nil {
		return GenerationRequest{}, err
	}
	if _, err := p.expect(";"); err != nil {
		return GenerationRequest{}, err
	}
	return GenerationRequest{Generator: name.Lexeme, ReflectedType: target, Span: start.Span}, nil
}

// evt1MaterializeGeneratedDeclarations is a single bounded phase before the
// normal analyzer. No source text is produced or parsed.
func evt1MaterializeGeneratedDeclarations(module *Module) error {
	if len(module.GenerationRequests) == 0 {
		return nil
	}
	generators := map[string]GeneratorDecl{}
	for _, generator := range module.Generators {
		if previous, exists := generators[generator.Name]; exists {
			return evt1Diagnostic("GENERATOR_DUPLICATE", fmt.Sprintf("generator %s conflicts with declaration at %d:%d", generator.Name, previous.Span.Line, previous.Span.Column), generator.Span)
		}
		generators[generator.Name] = generator
	}
	// A signature-only probe supplies the existing type/reflection authority
	// without checking ordinary bodies that may call a generated function.
	probe := *module
	probe.Assertions, probe.StaticAsserts = nil, nil
	// The exact-callable prepass rewrites declaration types in place. A probe
	// must never canonicalize the source declaration slices used by the real
	// analysis: concrete generic names need its own instance cache.
	probe.Structs = append([]StructDecl(nil), module.Structs...)
	for i := range probe.Structs {
		probe.Structs[i] = evt1CloneStructDecl(probe.Structs[i])
	}
	probe.Enums = append([]EnumDecl(nil), module.Enums...)
	for i := range probe.Enums {
		probe.Enums[i].Variants = append([]VariantDecl(nil), probe.Enums[i].Variants...)
		for j := range probe.Enums[i].Variants {
			probe.Enums[i].Variants[j].Payload = append([]Field(nil), probe.Enums[i].Variants[j].Payload...)
		}
	}
	probe.TypeAliases = append([]TypeAliasDecl(nil), module.TypeAliases...)
	probe.Automata = append([]AutomataDecl(nil), module.Automata...)
	probe.Functions = append([]FunctionDecl(nil), module.Functions...)
	for i := range probe.Functions {
		if probe.Functions[i].ReturnType.Kind != TypeInferred {
			probe.Functions[i].Body = nil
		}
	}
	env, err := analyzeModule(probe)
	if err != nil {
		return fmt.Errorf("generation reflection probe: %w", err)
	}
	for ordinal, request := range module.GenerationRequests {
		generator, exists := generators[request.Generator]
		if !exists {
			return evt1Diagnostic("GENERATOR_UNKNOWN", "unknown generator "+request.Generator, request.Span)
		}
		info, err := evt1ReflectType(env, ReflectionRequest{Type: request.ReflectedType, Span: request.Span})
		if err != nil {
			return err
		}
		prototype := TemplateDecl{Name: generator.Function.Name, TypeParam: generator.TypeParameter, ReturnType: generator.Function.ReturnType, Params: generator.Function.Params, Body: generator.Function.Body, Span: generator.Function.Span}
		function, err := evt1InstantiateTemplateFunction(prototype, info.Type)
		if err != nil {
			return evt1Diagnostic("GENERATOR_DECLARATION_INVALID", fmt.Sprintf("generator %s: %v", generator.Name, err), request.Span)
		}
		body, inputs, err := evt1ExpandGeneratedFields(*function.Body, info)
		if err != nil {
			return err
		}
		function.Body = &body
		function.Module = module.Name
		identity := strings.Join([]string{module.Name, generator.Module, generator.Name, fmt.Sprint(request.Span.Line, request.Span.Column), info.Type.String(), fmt.Sprint(ordinal), function.Name, evt1FunctionParamSignature(function)}, "|")
		for _, input := range inputs {
			identity += "|" + input.Name + ":" + input.Type.String()
		}
		function.Generated = &GeneratedOrigin{Kind: "GeneratedByReflection", Identity: "generated-" + digest([]byte(identity))[:20], Generator: generator.Name, GeneratorModule: generator.Module, GeneratorSpan: generator.Span, GenerationModule: module.Name, GenerationSite: request.Span, ReflectedType: info.Type, ReflectedModule: info.Module, Inputs: inputs}
		for _, existing := range module.Functions {
			if existing.Name != function.Name || evt1FunctionParamSignature(existing) != evt1FunctionParamSignature(function) {
				continue
			}
			where := fmt.Sprintf("handwritten declaration at %d:%d", existing.Span.Line, existing.Span.Column)
			if existing.Generated != nil {
				where = fmt.Sprintf("generated by %s at %d:%d", existing.Generated.Generator, existing.Generated.GenerationSite.Line, existing.Generated.GenerationSite.Column)
			}
			return evt1Diagnostic("GENERATOR_DECLARATION_CONFLICT", fmt.Sprintf("%s generated by %s at %d:%d conflicts with %s", function.Name, generator.Name, request.Span.Line, request.Span.Column, where), request.Span)
		}
		module.Functions = append(module.Functions, function)
	}
	return nil
}

func evt1ExpandGeneratedFields(block Block, info TypeInfo) (Block, []GeneratedInput, error) {
	var out Block
	out.Span = block.Span
	var inputs []GeneratedInput
	for _, statement := range block.Statements {
		if len(out.Statements) >= evt1GeneratedStatementLimit {
			return Block{}, nil, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", block.Span)
		}
		loop, ok := statement.(*ForeachStmt)
		if !ok {
			out.Statements = append(out.Statements, statement)
			continue
		}
		query, ok := loop.Source.(*TemplateCallExpr)
		if !ok || query.Callee != "Fields" || loop.ItemType.Name != "FieldInfo" || !query.TypeArg.SameValueType(info.Type) || len(query.Args) > 1 {
			return Block{}, nil, evt1Diagnostic("GENERATOR_FIELD_QUERY_INVALID", "generator foreach requires FieldInfo item in Fields<T>(attribute)", loop.Span)
		}
		selector := ""
		if len(query.Args) == 1 {
			name, ok := query.Args[0].(*NameExpr)
			if !ok {
				return Block{}, nil, evt1Diagnostic("GENERATOR_FIELD_QUERY_INVALID", "field selector must name an attribute", loop.Span)
			}
			selector = name.Name
		}
		for _, field := range info.Fields {
			if selector != "" && !evt1HasNamedAttribute(field.Attributes, selector) {
				continue
			}
			inputs = append(inputs, GeneratedInput{Name: field.Name, Type: field.Type, Span: field.Span})
			for _, item := range loop.Body.Statements {
				if len(out.Statements) >= evt1GeneratedStatementLimit {
					return Block{}, nil, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", loop.Span)
				}
				cloned, err := evt1SubstituteStatement(item, "__generated_field_clone__", Type{})
				if err != nil {
					return Block{}, nil, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), item.statementSpan())
				}
				if err := evt1ReplaceGeneratedFieldStatement(cloned, loop.ItemName, field.Name); err != nil {
					return Block{}, nil, err
				}
				out.Statements = append(out.Statements, cloned)
			}
		}
	}
	return out, inputs, nil
}

func evt1HasNamedAttribute(attributes []Attribute, name string) bool {
	for _, attribute := range attributes {
		if attribute.Name == name {
			return true
		}
	}
	return false
}

func evt1ReplaceGeneratedFieldStatement(statement Statement, placeholder, field string) error {
	switch s := statement.(type) {
	case *Block:
		for _, item := range s.Statements {
			if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
				return err
			}
		}
	case *IfStmt:
		if err := evt1ReplaceGeneratedFieldExpr(s.Condition, placeholder, field); err != nil {
			return err
		}
		for _, item := range s.Then.Statements {
			if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
				return err
			}
		}
		if s.Else != nil {
			for _, item := range s.Else.Statements {
				if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
					return err
				}
			}
		}
	case *ExprStmt:
		return evt1ReplaceGeneratedFieldExpr(s.Value, placeholder, field)
	case *AssignStmt:
		if err := evt1ReplaceGeneratedFieldExpr(s.Target, placeholder, field); err != nil {
			return err
		}
		return evt1ReplaceGeneratedFieldExpr(s.Value, placeholder, field)
	case *ReturnStmt:
		if s.Value != nil {
			return evt1ReplaceGeneratedFieldExpr(s.Value, placeholder, field)
		}
	case *VarDecl:
		if s.Value != nil {
			return evt1ReplaceGeneratedFieldExpr(s.Value, placeholder, field)
		}
	default:
		return evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", fmt.Sprintf("statement %T is not supported in a reflected field body", statement), statement.statementSpan())
	}
	return nil
}

func evt1ReplaceGeneratedFieldExpr(expression Expr, placeholder, field string) error {
	switch e := expression.(type) {
	case *NameExpr, *IntLiteral, *FloatLiteral, *StringLiteral, *BoolLiteral:
		return nil
	case *FieldExpr:
		if err := evt1ReplaceGeneratedFieldExpr(e.Receiver, placeholder, field); err != nil {
			return err
		}
		if e.Field == placeholder {
			e.Field = field
		}
		return nil
	case *CallExpr:
		if e.Receiver != nil {
			if err := evt1ReplaceGeneratedFieldExpr(e.Receiver, placeholder, field); err != nil {
				return err
			}
		}
		for _, arg := range e.Args {
			if err := evt1ReplaceGeneratedFieldExpr(arg, placeholder, field); err != nil {
				return err
			}
		}
		return nil
	case *TemplateCallExpr:
		for _, arg := range e.Args {
			if err := evt1ReplaceGeneratedFieldExpr(arg, placeholder, field); err != nil {
				return err
			}
		}
		return nil
	case *BinaryExpr:
		if err := evt1ReplaceGeneratedFieldExpr(e.Left, placeholder, field); err != nil {
			return err
		}
		return evt1ReplaceGeneratedFieldExpr(e.Right, placeholder, field)
	case *UnaryExpr:
		return evt1ReplaceGeneratedFieldExpr(e.Value, placeholder, field)
	case *RefExpr:
		return evt1ReplaceGeneratedFieldExpr(e.Value, placeholder, field)
	case *MoveExpr:
		return evt1ReplaceGeneratedFieldExpr(e.Value, placeholder, field)
	case *ParenExpr:
		return evt1ReplaceGeneratedFieldExpr(e.Value, placeholder, field)
	case *FailureExpr:
		return evt1ReplaceGeneratedFieldExpr(e.Value, placeholder, field)
	default:
		return evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", fmt.Sprintf("expression %T is not supported in a reflected field body", expression), expression.exprSpan())
	}
}
