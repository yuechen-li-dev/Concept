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
		probe.Functions[i].ReturnType = evt1CloneType(probe.Functions[i].ReturnType)
		probe.Functions[i].Params = append([]Param(nil), probe.Functions[i].Params...)
		for j := range probe.Functions[i].Params {
			probe.Functions[i].Params[j].Type = evt1CloneType(probe.Functions[i].Params[j].Type)
		}
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
		owner := ""
		if len(function.Params) != 0 && function.Params[0].Type.valueType().SameValueType(info.Type) {
			owner = function.Params[0].Name
		}
		body, inputs, err := evt1ExpandGeneratedFields(*function.Body, info, owner, env)
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

func evt1ExpandGeneratedFields(block Block, info TypeInfo, owner string, env *semanticEnv) (Block, []GeneratedInput, error) {
	var out Block
	out.Span = block.Span
	var inputs []GeneratedInput
	for _, statement := range block.Statements {
		if len(out.Statements) >= evt1GeneratedStatementLimit {
			return Block{}, nil, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", block.Span)
		}
		if scoped, ok := statement.(*TryStmt); ok {
			body, bodyInputs, err := evt1ExpandGeneratedFields(scoped.Body, info, owner, env)
			if err != nil {
				return Block{}, nil, err
			}
			scoped.Body = body
			inputs = append(inputs, bodyInputs...)
			for index := range scoped.Except {
				armBody, armInputs, err := evt1ExpandGeneratedFields(scoped.Except[index].Body, info, owner, env)
				if err != nil {
					return Block{}, nil, err
				}
				scoped.Except[index].Body = armBody
				inputs = append(inputs, armInputs...)
			}
			out.Statements = append(out.Statements, scoped)
			continue
		}
		loop, ok := statement.(*ForeachStmt)
		if !ok {
			aggregateInputs, err := evt1ExpandGeneratedAggregateStatement(statement, info)
			if err != nil {
				return Block{}, nil, err
			}
			evt1ReplaceGeneratedMetadataStatement(statement, info, "", "")
			inputs = append(inputs, aggregateInputs...)
			out.Statements = append(out.Statements, statement)
			continue
		}
		query, ok := loop.Source.(*TemplateCallExpr)
		if ok && query.Callee == "Cases" {
			caseStatement, caseInputs, err := evt1ExpandGeneratedCases(loop, query, info, owner, env)
			if err != nil {
				return Block{}, nil, err
			}
			out.Statements = append(out.Statements, caseStatement)
			inputs = append(inputs, caseInputs...)
			continue
		}
		if !ok || (query.Callee != "Fields" && query.Callee != "OnlyField") || loop.ItemType.Name != "FieldInfo" || !query.TypeArg.SameValueType(info.Type) || len(query.Args) > 1 {
			return Block{}, nil, evt1Diagnostic("GENERATOR_FIELD_QUERY_INVALID", "generator foreach requires FieldInfo item in Fields<T>(attribute) or OnlyField<T>()", loop.Span)
		}
		if query.Callee == "OnlyField" && (len(query.Args) != 0 || len(info.Fields) != 1) {
			return Block{}, nil, evt1Diagnostic("GENERATOR_SINGLE_FIELD_REQUIRED", "OnlyField<T>() requires exactly one reflected field and no selector", loop.Span)
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
			if selector != "" && !evt1GeneratedFieldMatches(env, selector, info.Type, field) {
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
				evt1ReplaceGeneratedMetadataStatement(cloned, info, field.Name, loop.ItemName)
				out.Statements = append(out.Statements, cloned)
			}
		}
	}
	return out, inputs, nil
}

func evt1ExpandGeneratedAggregateStatement(statement Statement, info TypeInfo) ([]GeneratedInput, error) {
	switch s := statement.(type) {
	case *ReturnStmt:
		return evt1ExpandGeneratedAggregateExpr(s.Value, info)
	case *VarDecl:
		return evt1ExpandGeneratedAggregateExpr(s.Value, info)
	case *AssignStmt:
		return evt1ExpandGeneratedAggregateExpr(s.Value, info)
	case *ExprStmt:
		return evt1ExpandGeneratedAggregateExpr(s.Value, info)
	default:
		return nil, nil
	}
}

func evt1ExpandGeneratedAggregateExpr(expression Expr, info TypeInfo) ([]GeneratedInput, error) {
	if expression == nil {
		return nil, nil
	}
	switch e := expression.(type) {
	case *StructConstructExpr:
		var inputs []GeneratedInput
		if loop := e.GeneratedFields; loop != nil {
			query, ok := loop.Source.(*TemplateCallExpr)
			if !ok || query.Callee != "Fields" || loop.ItemType.Name != "FieldInfo" || !query.TypeArg.SameValueType(info.Type) || len(query.Args) != 0 || len(loop.Body.Statements) != 1 {
				return nil, evt1Diagnostic("GENERATOR_AGGREGATE_FIELDS_INVALID", "aggregate field generation requires foreach (FieldInfo field in Fields<T>()) with one field assignment", loop.Span)
			}
			assignment, ok := loop.Body.Statements[0].(*AssignStmt)
			if !ok {
				return nil, evt1Diagnostic("GENERATOR_AGGREGATE_FIELDS_INVALID", "aggregate field generation requires assignment to the reflected field", loop.Span)
			}
			target, targetOK := assignment.Target.(*NameExpr)
			if !targetOK || target.Name != loop.ItemName {
				return nil, evt1Diagnostic("GENERATOR_AGGREGATE_FIELDS_INVALID", "aggregate field generation requires assignment to the reflected field", loop.Span)
			}
			for _, field := range info.Fields {
				value, err := evt1SubstituteExpr(assignment.Value, "__generated_field_clone__", Type{})
				if err != nil {
					return nil, err
				}
				if err := evt1ReplaceGeneratedFieldExpr(value, loop.ItemName, field.Name); err != nil {
					return nil, err
				}
				evt1ReplaceGeneratedMetadataExpr(&value, info, field.Name, loop.ItemName)
				e.Args = append(e.Args, value)
				e.ArgNames = append(e.ArgNames, field.Name)
				inputs = append(inputs, GeneratedInput{Name: field.Name, Type: field.Type, Span: field.Span})
			}
			e.GeneratedFields = nil
		}
		for _, arg := range e.Args {
			nested, err := evt1ExpandGeneratedAggregateExpr(arg, info)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, nested...)
		}
		return inputs, nil
	case *ConstructExpr:
		var inputs []GeneratedInput
		for _, arg := range e.Args {
			nested, err := evt1ExpandGeneratedAggregateExpr(arg, info)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, nested...)
		}
		return inputs, nil
	case *CallExpr:
		var inputs []GeneratedInput
		for _, arg := range e.Args {
			nested, err := evt1ExpandGeneratedAggregateExpr(arg, info)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, nested...)
		}
		return inputs, nil
	case *ParenExpr:
		return evt1ExpandGeneratedAggregateExpr(e.Value, info)
	case *MoveExpr:
		return evt1ExpandGeneratedAggregateExpr(e.Value, info)
	default:
		return nil, nil
	}
}

func evt1ReplaceGeneratedMetadataStatement(statement Statement, info TypeInfo, field, placeholder string) {
	switch s := statement.(type) {
	case *ReturnStmt:
		evt1ReplaceGeneratedMetadataExpr(&s.Value, info, field, placeholder)
	case *VarDecl:
		s.Type = evt1GeneratedFieldType(s.Type, info, field, placeholder)
		evt1ReplaceGeneratedMetadataExpr(&s.Value, info, field, placeholder)
	case *ExprStmt:
		evt1ReplaceGeneratedMetadataExpr(&s.Value, info, field, placeholder)
	case *AssignStmt:
		evt1ReplaceGeneratedMetadataExpr(&s.Value, info, field, placeholder)
	case *Block:
		for _, item := range s.Statements {
			evt1ReplaceGeneratedMetadataStatement(item, info, field, placeholder)
		}
	case *IfStmt:
		evt1ReplaceGeneratedMetadataExpr(&s.Condition, info, field, placeholder)
		for _, item := range s.Then.Statements {
			evt1ReplaceGeneratedMetadataStatement(item, info, field, placeholder)
		}
		if s.Else != nil {
			for _, item := range s.Else.Statements {
				evt1ReplaceGeneratedMetadataStatement(item, info, field, placeholder)
			}
		}
	case *TryStmt:
		for _, item := range s.Body.Statements {
			evt1ReplaceGeneratedMetadataStatement(item, info, field, placeholder)
		}
		for _, arm := range s.Except {
			for _, item := range arm.Body.Statements {
				evt1ReplaceGeneratedMetadataStatement(item, info, field, placeholder)
			}
		}
	}
}

func evt1ReplaceGeneratedMetadataExpr(expression *Expr, info TypeInfo, field, placeholder string) {
	if expression == nil || *expression == nil {
		return
	}
	switch e := (*expression).(type) {
	case *TemplateCallExpr:
		e.TypeArg = evt1GeneratedFieldType(e.TypeArg, info, field, placeholder)
		for i := range e.TypeArgs {
			e.TypeArgs[i] = evt1GeneratedFieldType(e.TypeArgs[i], info, field, placeholder)
		}
		if e.Callee == "TypeName" && e.TypeArg.SameValueType(info.Type) && len(e.Args) == 0 {
			*expression = &StringLiteral{Value: info.Type.Name, Span: e.Span}
			return
		}
		if e.Callee == "NameOf" && field != "" && e.TypeArg.Name == placeholder && len(e.Args) == 0 {
			*expression = &StringLiteral{Value: field, Span: e.Span}
			return
		}
		if field != "" && e.TypeArg.Name == placeholder && len(e.Args) == 0 {
			for caseIndex, enumCase := range info.EnumCases {
				if enumCase.Name != field {
					continue
				}
				if e.Callee == "HasPayload" {
					*expression = &BoolLiteral{Value: len(enumCase.Payload) > 0, Span: e.Span}
					return
				}
				if e.Callee == "ConstructCase" {
					constructed := &ConstructExpr{EnumName: info.Type.Name, VariantName: enumCase.Name, Span: e.Span}
					for fieldIndex := range enumCase.Payload {
						constructed.Args = append(constructed.Args, &NameExpr{Name: fmt.Sprintf("__generated_payload_%d_%d", caseIndex, fieldIndex), Span: e.Span})
					}
					*expression = constructed
					return
				}
			}
		}
		for i := range e.Args {
			evt1ReplaceGeneratedMetadataExpr(&e.Args[i], info, field, placeholder)
		}
	case *CallExpr:
		for i := range e.Args {
			evt1ReplaceGeneratedMetadataExpr(&e.Args[i], info, field, placeholder)
		}
	case *ConstructExpr:
		for i := range e.Args {
			evt1ReplaceGeneratedMetadataExpr(&e.Args[i], info, field, placeholder)
		}
	case *StructConstructExpr:
		e.StructType = evt1GeneratedFieldType(e.StructType, info, field, placeholder)
		e.StructName = e.StructType.String()
		for i := range e.Args {
			evt1ReplaceGeneratedMetadataExpr(&e.Args[i], info, field, placeholder)
		}
	case *FailureExpr:
		evt1ReplaceGeneratedMetadataExpr(&e.Value, info, field, placeholder)
	case *ParenExpr:
		evt1ReplaceGeneratedMetadataExpr(&e.Value, info, field, placeholder)
	case *MoveExpr:
		evt1ReplaceGeneratedMetadataExpr(&e.Value, info, field, placeholder)
	case *RefExpr:
		evt1ReplaceGeneratedMetadataExpr(&e.Value, info, field, placeholder)
	}
}

// FieldType<item> is a compile-time type query scoped to a reflected-field
// expansion. Replacing the type node preserves ordinary overload resolution
// and type checking for whatever operation the generator chooses to call.
func evt1GeneratedFieldType(t Type, info TypeInfo, field, placeholder string) Type {
	if field != "" && t.Name == "FieldType" && len(t.TypeArgs) == 1 && t.TypeArgs[0].Name == placeholder {
		for _, candidate := range info.Fields {
			if candidate.Name == field {
				return candidate.Type
			}
		}
	}
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1GeneratedFieldType(t.TypeArgs[i], info, field, placeholder)
	}
	if t.ArrayElem != nil {
		element := evt1GeneratedFieldType(*t.ArrayElem, info, field, placeholder)
		t.ArrayElem = &element
	}
	return t
}

func evt1GeneratedFieldMatches(env *semanticEnv, selector string, owner Type, field FieldInfo) bool {
	if concept, exists := env.concepts[selector]; exists {
		arguments := []Type{field.Type}
		if len(evt1ConceptParameters(concept)) == 2 {
			arguments = append(arguments, owner)
		}
		if len(evt1ConceptParameters(concept)) < 1 || len(evt1ConceptParameters(concept)) > 2 {
			return false
		}
		return checkConceptApplicationSatisfaction(env, selector, arguments, nil, field.Span) == nil
	}
	return evt1HasNamedAttribute(field.Attributes, selector)
}

func evt1ExpandGeneratedCases(loop *ForeachStmt, query *TemplateCallExpr, info TypeInfo, owner string, env *semanticEnv) (Statement, []GeneratedInput, error) {
	if info.Kind != "enum" || loop.ItemType.Name != "EnumCaseInfo" || !query.TypeArg.SameValueType(info.Type) || len(query.Args) != 0 {
		return nil, nil, evt1Diagnostic("GENERATOR_CASE_QUERY_INVALID", "generator foreach requires EnumCaseInfo item in Cases<T>() over an enum", loop.Span)
	}
	if owner == "" {
		return evt1ExpandGeneratedCaseConditions(loop, info, env)
	}
	match := &MatchStmt{Subject: &NameExpr{Name: owner, Span: loop.Span}, Span: loop.Span}
	var inputs []GeneratedInput
	statementCount := 0
	for caseIndex, enumCase := range info.EnumCases {
		pattern := Pattern{EnumName: info.Type.Name, VariantName: enumCase.Name, Span: enumCase.Span}
		for fieldIndex := range enumCase.Payload {
			pattern.Bindings = append(pattern.Bindings, fmt.Sprintf("__generated_payload_%d_%d", caseIndex, fieldIndex))
		}
		arm := StatementArm{Pattern: pattern, Block: Block{Span: loop.Body.Span}, Span: enumCase.Span}
		body, err := evt1ExpandGeneratedCasePayloads(loop.Body, info, enumCase, pattern, loop.ItemName, env, &inputs, &statementCount)
		if err != nil {
			return nil, nil, err
		}
		arm.Block = body
		match.Arms = append(match.Arms, arm)
	}
	return match, inputs, nil
}

func evt1ExpandGeneratedCasePayloads(block Block, info TypeInfo, enumCase EnumCaseInfo, pattern Pattern, casePlaceholder string, env *semanticEnv, inputs *[]GeneratedInput, statementCount *int) (Block, error) {
	out := Block{Span: block.Span}
	for _, statement := range block.Statements {
		if *statementCount >= evt1GeneratedStatementLimit {
			return Block{}, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", statement.statementSpan())
		}
		if scoped, ok := statement.(*TryStmt); ok {
			cloned, err := evt1SubstituteStatement(scoped, "__generated_case_clone__", Type{})
			if err != nil {
				return Block{}, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), statement.statementSpan())
			}
			try := cloned.(*TryStmt)
			try.Body, err = evt1ExpandGeneratedCasePayloads(try.Body, info, enumCase, pattern, casePlaceholder, env, inputs, statementCount)
			if err != nil {
				return Block{}, err
			}
			for index := range try.Except {
				try.Except[index].Body, err = evt1ExpandGeneratedCasePayloads(try.Except[index].Body, info, enumCase, pattern, casePlaceholder, env, inputs, statementCount)
				if err != nil {
					return Block{}, err
				}
			}
			evt1ReplaceGeneratedMetadataStatement(try, info, enumCase.Name, casePlaceholder)
			out.Statements = append(out.Statements, try)
			*statementCount++
			continue
		}
		payloadLoop, ok := statement.(*ForeachStmt)
		if !ok {
			if *statementCount >= evt1GeneratedStatementLimit {
				return Block{}, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", statement.statementSpan())
			}
			cloned, err := evt1SubstituteStatement(statement, "__generated_case_clone__", Type{})
			if err != nil {
				return Block{}, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), statement.statementSpan())
			}
			evt1ReplaceGeneratedMetadataStatement(cloned, info, enumCase.Name, casePlaceholder)
			out.Statements = append(out.Statements, cloned)
			*statementCount++
			continue
		}
		payloadQuery, ok := payloadLoop.Source.(*TemplateCallExpr)
		if !ok || payloadQuery.Callee != "Payload" || payloadLoop.ItemType.Name != "FieldInfo" || payloadQuery.TypeArg.Name != casePlaceholder || len(payloadQuery.Args) > 1 {
			return Block{}, evt1Diagnostic("GENERATOR_CASE_BODY_INVALID", "case body requires FieldInfo item in Payload<case>(attribute)", payloadLoop.Span)
		}
		selector := ""
		if len(payloadQuery.Args) == 1 {
			name, ok := payloadQuery.Args[0].(*NameExpr)
			if !ok {
				return Block{}, evt1Diagnostic("GENERATOR_CASE_BODY_INVALID", "payload selector must name an attribute", payloadLoop.Span)
			}
			selector = name.Name
		}
		for fieldIndex, field := range enumCase.Payload {
			if selector != "" && !evt1GeneratedFieldMatches(env, selector, info.Type, field) {
				continue
			}
			*inputs = append(*inputs, GeneratedInput{Name: enumCase.Name + "." + field.Name, Type: field.Type, Span: field.Span})
			for _, bodyStatement := range payloadLoop.Body.Statements {
				if *statementCount >= evt1GeneratedStatementLimit {
					return Block{}, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", payloadLoop.Span)
				}
				cloned, err := evt1SubstituteStatement(bodyStatement, "__generated_payload_clone__", Type{})
				if err != nil {
					return Block{}, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), bodyStatement.statementSpan())
				}
				if err := evt1ReplaceGeneratedFieldStatement(cloned, payloadLoop.ItemName, pattern.Bindings[fieldIndex]); err != nil {
					return Block{}, err
				}
				payloadInfo := info
				payloadInfo.Fields = enumCase.Payload
				evt1ReplaceGeneratedMetadataStatement(cloned, payloadInfo, field.Name, payloadLoop.ItemName)
				evt1ReplaceGeneratedMetadataStatement(cloned, info, enumCase.Name, casePlaceholder)
				out.Statements = append(out.Statements, cloned)
				*statementCount++
			}
		}
	}
	return out, nil
}

// Without an enum-valued subject, Cases<T>() expands into ordinary checked
// statements for each case. This is used by readers: the generator decides
// how to recognize a case, and ConstructCase<case>() closes its typed payload
// locals into an ordinary enum constructor.
func evt1ExpandGeneratedCaseConditions(loop *ForeachStmt, info TypeInfo, env *semanticEnv) (Statement, []GeneratedInput, error) {
	if len(loop.Body.Statements) != 1 {
		return nil, nil, evt1Diagnostic("GENERATOR_CASE_BODY_INVALID", "case condition generation requires one if statement", loop.Span)
	}
	condition, ok := loop.Body.Statements[0].(*IfStmt)
	if !ok {
		return nil, nil, evt1Diagnostic("GENERATOR_CASE_BODY_INVALID", "case condition generation requires one if statement", loop.Span)
	}
	result := &Block{Span: loop.Span}
	var inputs []GeneratedInput
	count := 0
	for caseIndex, enumCase := range info.EnumCases {
		cloned, err := evt1SubstituteStatement(condition, "__generated_case_clone__", Type{})
		if err != nil {
			return nil, nil, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), condition.Span)
		}
		branch := cloned.(*IfStmt)
		branch.Then, err = evt1ExpandGeneratedPayloadStatements(branch.Then, info, caseIndex, enumCase, loop.ItemName, env, &inputs, &count)
		if err != nil {
			return nil, nil, err
		}
		evt1ReplaceGeneratedMetadataStatement(branch, info, enumCase.Name, loop.ItemName)
		result.Statements = append(result.Statements, branch)
	}
	return result, inputs, nil
}

func evt1ExpandGeneratedPayloadStatements(block Block, info TypeInfo, caseIndex int, enumCase EnumCaseInfo, casePlaceholder string, env *semanticEnv, inputs *[]GeneratedInput, count *int) (Block, error) {
	out := Block{Span: block.Span}
	for _, statement := range block.Statements {
		if *count >= evt1GeneratedStatementLimit {
			return Block{}, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", statement.statementSpan())
		}
		if scoped, ok := statement.(*TryStmt); ok {
			cloned, err := evt1SubstituteStatement(scoped, "__generated_case_clone__", Type{})
			if err != nil {
				return Block{}, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), statement.statementSpan())
			}
			try := cloned.(*TryStmt)
			try.Body, err = evt1ExpandGeneratedPayloadStatements(try.Body, info, caseIndex, enumCase, casePlaceholder, env, inputs, count)
			if err != nil {
				return Block{}, err
			}
			for index := range try.Except {
				try.Except[index].Body, err = evt1ExpandGeneratedPayloadStatements(try.Except[index].Body, info, caseIndex, enumCase, casePlaceholder, env, inputs, count)
				if err != nil {
					return Block{}, err
				}
			}
			out.Statements = append(out.Statements, try)
			*count++
			continue
		}
		loop, ok := statement.(*ForeachStmt)
		if !ok {
			out.Statements = append(out.Statements, statement)
			*count++
			continue
		}
		query, ok := loop.Source.(*TemplateCallExpr)
		if !ok || query.Callee != "Payload" || loop.ItemType.Name != "FieldInfo" || query.TypeArg.Name != casePlaceholder || len(query.Args) != 0 {
			return Block{}, evt1Diagnostic("GENERATOR_CASE_BODY_INVALID", "case body requires FieldInfo item in Payload<case>()", loop.Span)
		}
		payloadInfo := info
		payloadInfo.Fields = enumCase.Payload
		for fieldIndex, field := range enumCase.Payload {
			*inputs = append(*inputs, GeneratedInput{Name: enumCase.Name + "." + field.Name, Type: field.Type, Span: field.Span})
			binding := fmt.Sprintf("__generated_payload_%d_%d", caseIndex, fieldIndex)
			for _, source := range loop.Body.Statements {
				if *count >= evt1GeneratedStatementLimit {
					return Block{}, evt1Diagnostic("GENERATOR_OUTPUT_LIMIT", "generated function exceeds statement limit", source.statementSpan())
				}
				cloned, err := evt1SubstituteStatement(source, "__generated_payload_clone__", Type{})
				if err != nil {
					return Block{}, evt1Diagnostic("GENERATOR_BODY_UNSUPPORTED", err.Error(), source.statementSpan())
				}
				if err := evt1ReplaceGeneratedFieldStatement(cloned, loop.ItemName, binding); err != nil {
					return Block{}, err
				}
				evt1ReplaceGeneratedMetadataStatement(cloned, payloadInfo, field.Name, loop.ItemName)
				out.Statements = append(out.Statements, cloned)
				*count++
			}
		}
	}
	return out, nil
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
	case *TryStmt:
		for _, item := range s.Body.Statements {
			if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
				return err
			}
		}
		for _, arm := range s.Except {
			for _, item := range arm.Body.Statements {
				if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
					return err
				}
			}
		}
	case *ForeachStmt:
		if err := evt1ReplaceGeneratedFieldExpr(s.Source, placeholder, field); err != nil {
			return err
		}
		for _, item := range s.Body.Statements {
			if err := evt1ReplaceGeneratedFieldStatement(item, placeholder, field); err != nil {
				return err
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
		if s.Name == placeholder {
			s.Name = field
		}
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
	case *NameExpr:
		if e.Name == placeholder {
			e.Name = field
		}
		return nil
	case *IntLiteral, *FloatLiteral, *StringLiteral, *BoolLiteral:
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
	case *StructConstructExpr:
		for _, arg := range e.Args {
			if err := evt1ReplaceGeneratedFieldExpr(arg, placeholder, field); err != nil {
				return err
			}
		}
		return nil
	case *ConstructExpr:
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
