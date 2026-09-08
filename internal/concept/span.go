package concept

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	evt1SpanMutableName  = "Span"
	evt1SpanReadonlyName = "ReadOnlySpan"
)

type evt1SpanFacts struct {
	ElementType          Type
	RegionID             string
	BackingByteOffset    int
	BaseOffsetExpression string
	LengthExpression     string
	StaticLength         int
	LengthStatic         bool
	ByteExtentExpression string
	Alignment            int
	Mutable              bool
	Provenance           evt1LifetimeProvenance
}

func evt1IsSpanType(t Type) bool {
	return t.Kind == TypeSpan || ((t.Name == evt1SpanMutableName || t.Name == evt1SpanReadonlyName) && len(t.TypeArgs) == 1)
}

func evt1SpanMutable(t Type) bool {
	return evt1IsSpanType(t) && t.Name == evt1SpanMutableName
}

func evt1SpanElement(t Type) Type {
	if len(t.TypeArgs) != 1 {
		return Type{}
	}
	return t.TypeArgs[0]
}

func evt1SpanType(name string, element Type, span Span) Type {
	return Type{Name: name, Kind: TypeSpan, TypeArgs: []Type{element.valueType()}, Span: span}
}

func evt1ParameterSpanFacts(env *semanticEnv, name string, t Type, provenance evt1LifetimeProvenance) *evt1SpanFacts {
	if !evt1IsSpanType(t) {
		return nil
	}
	element := evt1SpanElement(t)
	_, alignment, err := evt1SpanElementGeometry(env, element)
	if err != nil {
		alignment = 1
	}
	return &evt1SpanFacts{
		ElementType: element, RegionID: "parameter:" + name, BaseOffsetExpression: "runtime",
		LengthExpression: "runtime", ByteExtentExpression: "runtime", Alignment: alignment,
		Mutable: evt1SpanMutable(t), Provenance: provenance,
	}
}

func evt1SpanCName(t Type) string {
	prefix := "concept_span_"
	if t.Name == evt1SpanReadonlyName {
		prefix = "concept_readonly_span_"
	}
	return prefix + evt1TypeIdentity(evt1SpanElement(t))
}

func evt1GCD(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a < 1 {
		return 1
	}
	return a
}

func evt1SpanElementGeometry(env *semanticEnv, element Type) (int, int, error) {
	size, alignment, err := evt1TypeGeometry(env, element.valueType())
	if err != nil {
		return 0, 0, evt1Diagnostic("CV4601", "Span element type requires fixed element geometry: "+element.String(), element.Span)
	}
	return size, alignment, nil
}

func evt1StaticInt(env *semanticEnv, scope *evt1Scope, expr Expr) (int, bool) {
	value, err := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), expr)
	return value.IntValue, err == nil && value.Kind == ValueInt
}

func evt1SpanSourceFacts(env *semanticEnv, scope *evt1Scope, expr Expr, sourceType Type, place evt1LValue) (evt1SpanFacts, error) {
	if name, ok := expr.(*NameExpr); ok {
		if binding, found := scope.lookup(name.Name); found {
			if binding.spanFacts != nil {
				return *binding.spanFacts, nil
			}
			if binding.regionFacts != nil {
				facts := *binding.regionFacts
				facts.Provenance = evt1ExprProvenance(env, scope, expr)
				return facts, nil
			}
		}
	}
	if call, ok := expr.(*CallExpr); ok && call.SpanElementType != nil && call.RegionID != "" {
		return evt1SpanFactsFromCall(call), nil
	}
	if field, ok := expr.(*FieldExpr); ok && field.RegionID != "" {
		element := *sourceType.ArrayElem
		count := evt1StorageElementCount(sourceType)
		return evt1SpanFacts{
			ElementType: element, RegionID: field.RegionID, BackingByteOffset: field.RegionOffset,
			BaseOffsetExpression: "0", LengthExpression: fmt.Sprint(count), StaticLength: count, LengthStatic: true,
			ByteExtentExpression: fmt.Sprint(field.RegionExtent), Alignment: field.RegionAlignment,
			Mutable: place.mutable, Provenance: evt1ExprProvenance(env, scope, expr),
		}, nil
	}
	if sourceType.ArrayElem == nil || !sourceType.Contiguous {
		return evt1SpanFacts{}, evt1Diagnostic("CV4600", "Span requires an existing contiguous array, ndarray, layout region, stream channel, or Span", expr.exprSpan())
	}
	element := *sourceType.ArrayElem
	size, alignment, err := evt1SpanElementGeometry(env, element)
	if err != nil {
		return evt1SpanFacts{}, err
	}
	length := "runtime_total_extent"
	byteExtent := "length * sizeof(" + element.String() + ")"
	static := !evt1StorageHasRuntimeShape(sourceType)
	count := 0
	if static {
		count = sourceType.ArrayLength
		if len(sourceType.Shape) > 0 {
			count = 1
			for _, dimension := range sourceType.Shape {
				if dimension.Extent != 0 && count > math.MaxInt/dimension.Extent {
					return evt1SpanFacts{}, evt1Diagnostic("CV4607", "Span element count overflows the compiler's bounded extent representation", expr.exprSpan())
				}
				count *= dimension.Extent
			}
		}
		if count != 0 && size > math.MaxInt/count {
			return evt1SpanFacts{}, evt1Diagnostic("CV4607", "Span byte extent overflows the compiler's bounded extent representation", expr.exprSpan())
		}
		length = fmt.Sprint(count)
		byteExtent = fmt.Sprint(count * size)
	}
	regionID := "storage:" + place.path.Root
	for _, part := range place.path.Fields {
		regionID += "." + part
	}
	return evt1SpanFacts{
		ElementType: element, RegionID: regionID, BaseOffsetExpression: "0", LengthExpression: length,
		StaticLength: count, LengthStatic: static, ByteExtentExpression: byteExtent, Alignment: alignment,
		Mutable: place.mutable, Provenance: evt1ExprProvenance(env, scope, expr),
	}, nil
}

func evt1SpanFactsFromCall(call *CallExpr) evt1SpanFacts {
	staticLength, static := 0, false
	if call.LengthExpression != "runtime" && call.LengthExpression != "runtime_total_extent" {
		_, err := fmt.Sscan(call.LengthExpression, &staticLength)
		static = err == nil
	}
	return evt1SpanFacts{
		ElementType: *call.SpanElementType, RegionID: call.RegionID, BackingByteOffset: call.BackingByteOffset,
		BaseOffsetExpression: call.BaseOffsetExpression, LengthExpression: call.LengthExpression,
		StaticLength: staticLength, LengthStatic: static, ByteExtentExpression: call.ByteExtentExpression,
		Alignment: call.Alignment, Mutable: call.Mutability == "mutable",
		Provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceKind(call.ProvenanceKind), Scoped: call.ProvenanceScoped},
	}
}

func evt1KnownSpanFacts(scope *evt1Scope, expr Expr) (evt1SpanFacts, bool) {
	if name, ok := expr.(*NameExpr); ok {
		if binding, found := scope.lookup(name.Name); found && binding.spanFacts != nil {
			return *binding.spanFacts, true
		}
	}
	if call, ok := expr.(*CallExpr); ok && call.SpanElementType != nil && call.RegionID != "" {
		return evt1SpanFactsFromCall(call), true
	}
	return evt1SpanFacts{}, false
}

func evt1ApplySpanFacts(call *CallExpr, facts evt1SpanFacts, intrinsic, mutability string) {
	element := facts.ElementType
	call.Intrinsic = intrinsic
	call.SpanElementType = &element
	call.RegionID = facts.RegionID
	call.BackingByteOffset = facts.BackingByteOffset
	call.BaseOffsetExpression = facts.BaseOffsetExpression
	call.LengthExpression = facts.LengthExpression
	call.ByteExtentExpression = facts.ByteExtentExpression
	call.Alignment = facts.Alignment
	call.Mutability = mutability
	call.ProvenanceKind = string(facts.Provenance.Kind)
	call.ProvenanceScoped = facts.Provenance.Scoped
}

func validateSpanCall(env *semanticEnv, scope *evt1Scope, call *CallExpr, expected *Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if call.Callee == "Subspan" {
		if len(call.Args) != 3 {
			return Type{}, evt1Diagnostic("CV4603", fmt.Sprintf("Subspan expects span, offset, and length; got %d argument(s)", len(call.Args)), call.Span)
		}
		parentType, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !evt1IsSpanType(parentType) {
			return Type{}, evt1Diagnostic("CV4600", "Subspan requires Span or ReadOnlySpan as its first argument", call.Args[0].exprSpan())
		}
		for _, arg := range call.Args[1:] {
			t, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if t.Name != "int" {
				return Type{}, evt1Diagnostic("CV4603", "Subspan offset and length must be int", arg.exprSpan())
			}
		}
		place, err := validateAssignable(env, scope, call.Args[0], templateInfo)
		if err != nil {
			return Type{}, evt1Diagnostic("CV4600", "Subspan requires an existing Span value", call.Args[0].exprSpan())
		}
		parent, err := evt1SpanSourceFacts(env, scope, call.Args[0], parentType, place)
		if err != nil {
			return Type{}, err
		}
		offset, offsetStatic := evt1StaticInt(env, scope, call.Args[1])
		length, lengthStatic := evt1StaticInt(env, scope, call.Args[2])
		if offsetStatic && offset < 0 || lengthStatic && length < 0 || offsetStatic && lengthStatic && parent.LengthStatic && (offset > parent.StaticLength || length > parent.StaticLength-offset) {
			return Type{}, evt1Diagnostic("CV4603", "Span subregion is outside the parent half-open interval", call.Span)
		}
		size, elementAlign, err := evt1SpanElementGeometry(env, parent.ElementType)
		if err != nil {
			return Type{}, err
		}
		facts := parent
		facts.BaseOffsetExpression = "(" + parent.BaseOffsetExpression + "+" + evt1ExprIdentity(call.Args[1]) + ")"
		facts.LengthExpression = evt1ExprIdentity(call.Args[2])
		facts.LengthStatic, facts.StaticLength = lengthStatic, length
		facts.ByteExtentExpression = facts.LengthExpression + " * sizeof(" + parent.ElementType.String() + ")"
		if lengthStatic && length <= math.MaxInt/size {
			facts.ByteExtentExpression = fmt.Sprint(length * size)
		}
		if offsetStatic && offset <= math.MaxInt/size {
			facts.Alignment = evt1GCD(parent.Alignment, offset*size)
		} else {
			facts.Alignment = evt1GCD(parent.Alignment, elementAlign)
		}
		facts.Mutable = evt1SpanMutable(parentType)
		facts.Provenance = evt1ExprProvenance(env, scope, call.Args[0])
		mutability := "readonly"
		if facts.Mutable {
			mutability = "mutable"
		}
		evt1ApplySpanFacts(call, facts, "span_subregion", mutability)
		call.RuntimeBounds = !(offsetStatic && lengthStatic && parent.LengthStatic)
		return parentType, nil
	}

	if call.Callee != evt1SpanMutableName && call.Callee != evt1SpanReadonlyName {
		return Type{}, evt1Diagnostic("CV4600", "unknown Span intrinsic "+call.Callee, call.Span)
	}
	if len(call.Args) != 1 {
		return Type{}, evt1Diagnostic("CV4600", fmt.Sprintf("%s expects exactly one contiguous source", call.Callee), call.Span)
	}
	sourceType, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	place, err := validateAssignable(env, scope, call.Args[0], templateInfo)
	if err != nil {
		return Type{}, evt1Diagnostic("CV4600", call.Callee+" requires an existing contiguous source place", call.Args[0].exprSpan())
	}
	if sourceType.ArrayElem == nil && !evt1IsSpanType(sourceType) {
		return Type{}, evt1Diagnostic("CV4600", call.Callee+" requires a contiguous array, ndarray, layout region, stream channel, or Span", call.Args[0].exprSpan())
	}
	facts, err := evt1SpanSourceFacts(env, scope, call.Args[0], sourceType, place)
	if err != nil {
		return Type{}, err
	}
	targetElement := facts.ElementType.valueType()
	if expected != nil && evt1IsSpanType(*expected) {
		targetElement = evt1SpanElement(*expected).valueType()
		if !evt1CanonicalType(env, targetElement).Equal(evt1CanonicalType(env, facts.ElementType.valueType())) {
			return Type{}, evt1Diagnostic("CV4601", fmt.Sprintf("Span element type mismatch: source %s cannot form %s", facts.ElementType.String(), expected.String()), call.Args[0].exprSpan())
		}
	}
	if call.Callee == evt1SpanMutableName && (!facts.Mutable || sourceType.Name == evt1SpanReadonlyName) {
		return Type{}, evt1Diagnostic("CV4602", "mutable Span cannot originate from const or readonly storage", call.Args[0].exprSpan())
	}
	mutability := "readonly"
	if call.Callee == evt1SpanMutableName {
		mutability = "mutable"
	}
	facts.ElementType = targetElement
	facts.Mutable = mutability == "mutable"
	facts.Provenance = evt1ExprProvenance(env, scope, call.Args[0])
	intrinsic := "span_from_region"
	if evt1IsSpanType(sourceType) && call.Callee == evt1SpanReadonlyName && sourceType.Name == evt1SpanMutableName {
		intrinsic = "span_to_readonly"
	}
	evt1ApplySpanFacts(call, facts, intrinsic, mutability)
	return evt1SpanType(call.Callee, targetElement, call.Span), nil
}

func evt1SpanFactsForValue(env *semanticEnv, scope *evt1Scope, expr Expr, t Type) *evt1SpanFacts {
	if !evt1IsSpanType(t) {
		return nil
	}
	if call, ok := expr.(*CallExpr); ok && call.SpanElementType != nil {
		facts := evt1SpanFactsFromCall(call)
		facts.Provenance = evt1ExprProvenance(env, scope, expr)
		return &facts
	}
	if call, ok := expr.(*CallExpr); ok {
		element := evt1SpanElement(t)
		_, alignment, _ := evt1SpanElementGeometry(env, element)
		regionID := "derived-call-region:" + call.Callee
		backingByteOffset := 0
		argTypes := make([]Type, 0, len(call.Args))
		for _, arg := range call.Args {
			argType, err := validateExpr(env, scope, arg, nil, false)
			if err != nil {
				argTypes = nil
				break
			}
			argTypes = append(argTypes, argType)
		}
		if argTypes != nil {
			if fn, err := evt1ResolveOrdinaryCall(env, scope, call.Callee, call.Args, argTypes, nil, call.Span); err == nil {
				summary := env.resultProvenance[evt1FunctionProvenanceKey(fn)]
				if len(summary.ParameterIndices) == 1 {
					index := summary.ParameterIndices[0]
					if index >= 0 && index < len(call.Args) {
						if sourceFacts, ok := evt1SpanFactsFromCallArgument(env, scope, call.Args[index]); ok {
							regionID = sourceFacts.RegionID
							backingByteOffset = sourceFacts.BackingByteOffset
							alignment = evt1GCD(sourceFacts.Alignment, alignment)
						}
					}
				}
			}
		}
		return &evt1SpanFacts{
			ElementType: element, RegionID: regionID, BackingByteOffset: backingByteOffset, BaseOffsetExpression: "runtime",
			LengthExpression: "runtime", ByteExtentExpression: "runtime", Alignment: alignment,
			Mutable: evt1SpanMutable(t), Provenance: evt1ExprProvenance(env, scope, expr),
		}
	}
	if name, ok := expr.(*NameExpr); ok {
		if binding, found := scope.lookup(name.Name); found && binding.spanFacts != nil {
			facts := *binding.spanFacts
			return &facts
		}
	}
	return nil
}

func evt1SpanFactsFromCallArgument(env *semanticEnv, scope *evt1Scope, expr Expr) (evt1SpanFacts, bool) {
	for {
		switch wrapped := expr.(type) {
		case *RefExpr:
			expr = wrapped.Value
			continue
		case *MoveExpr:
			expr = wrapped.Value
			continue
		}
		break
	}
	t, err := validateExpr(env, scope, expr, nil, false)
	if err != nil {
		return evt1SpanFacts{}, false
	}
	place, err := validateAssignable(env, scope, expr, nil)
	if err != nil {
		return evt1SpanFacts{}, false
	}
	facts, err := evt1SpanSourceFacts(env, scope, expr, t, place)
	return facts, err == nil
}

func evt1RegionFactsForValue(env *semanticEnv, scope *evt1Scope, expr Expr, t Type) *evt1SpanFacts {
	if bind, ok := expr.(*BindExpr); ok {
		sourceType, err := validateExpr(env, scope, bind.Source, nil, false)
		if err == nil {
			place, placeErr := validateAssignable(env, scope, bind.Source, nil)
			if placeErr == nil {
				facts, factsErr := evt1SpanSourceFacts(env, scope, bind.Source, sourceType, place)
				if factsErr == nil {
					return &facts
				}
			}
		}
	}
	if ref, ok := expr.(*RefExpr); ok {
		sourceType, err := validateExpr(env, scope, ref.Value, nil, false)
		if err == nil {
			place, placeErr := validateAssignable(env, scope, ref.Value, nil)
			if placeErr == nil {
				facts, factsErr := evt1SpanSourceFacts(env, scope, ref.Value, sourceType, place)
				if factsErr == nil {
					return &facts
				}
			}
		}
	}
	_ = t
	return nil
}

func evt1CollectSpanTypes(module Module) []Type {
	types := map[string]Type{}
	var add func(Type)
	add = func(t Type) {
		if evt1IsSpanType(t) {
			types[evt1SpanCName(t)] = t
		}
		for _, arg := range t.TypeArgs {
			add(arg)
		}
		if t.ArrayElem != nil {
			add(*t.ArrayElem)
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, statement := range block.Statements {
			switch s := statement.(type) {
			case *VarDecl:
				add(s.Type)
			case *Block:
				visitBlock(*s)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitBlock(s.Body)
			case *ForeachStmt:
				add(s.SourceType)
				add(s.ItemType)
				visitBlock(s.Body)
			case *MatchStmt:
				for _, arm := range s.Arms {
					visitBlock(arm.Block)
				}
			case *TryStmt:
				visitBlock(s.Body)
				for _, arm := range s.Except {
					visitBlock(arm.Body)
				}
			}
		}
	}
	for _, decl := range module.Structs {
		for _, field := range decl.Fields {
			add(field.Type)
		}
	}
	for _, decl := range module.Enums {
		for _, variant := range decl.Variants {
			for _, field := range variant.Payload {
				add(field.Type)
			}
		}
	}
	for _, fn := range module.Functions {
		add(fn.ReturnType)
		for _, param := range fn.Params {
			add(param.Type)
		}
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	for _, automata := range module.Automata {
		for _, field := range automata.StateFields {
			add(field.Type)
		}
		for _, machine := range automata.Machines {
			for _, field := range machine.Fields {
				add(field.Type)
			}
			for _, state := range machine.States {
				if state.Body != nil {
					visitBlock(*state.Body)
				}
			}
		}
	}
	keys := make([]string, 0, len(types))
	for key := range types {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Type, 0, len(keys))
	for _, key := range keys {
		out = append(out, types[key])
	}
	return out
}

func evt1SpanDeclarations(types []Type) string {
	var out strings.Builder
	for _, t := range types {
		pointer := evt1CType(evt1SpanElement(t)) + "*"
		if t.Name == evt1SpanReadonlyName {
			pointer = "const " + evt1CType(evt1SpanElement(t)) + "*"
		}
		out.WriteString(fmt.Sprintf("typedef struct { %s data; size_t length; } %s;\n", pointer, evt1SpanCName(t)))
	}
	if len(types) > 0 {
		out.WriteByte('\n')
	}
	return out.String()
}

func (f *evt1FunctionLowerer) lowerSpanCall(call *CallExpr, indent int) (string, string, Type) {
	resultType := evt1SpanType(call.MutabilityToTypeName(), *call.SpanElementType, call.Span)
	if call.Intrinsic == "span_subregion" {
		parentPrelude, parentValue, parentType := f.lowerExpr(call.Args[0], indent)
		offsetPrelude, offsetValue, _ := f.lowerExpr(call.Args[1], indent)
		lengthPrelude, lengthValue, _ := f.lowerExpr(call.Args[2], indent)
		parentName := f.nextTemp("span_parent")
		offsetName := f.nextTemp("span_offset")
		lengthName := f.nextTemp("span_length")
		var out strings.Builder
		out.WriteString(parentPrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(parentType), parentName, parentValue))
		out.WriteString(offsetPrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", offsetName, offsetValue))
		out.WriteString(lengthPrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", lengthName, lengthValue))
		if f.plannedStrategy("span_subregion", "HoistedRuntime") == "HoistedRuntime" {
			out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || %s < 0 || (size_t)%s > %s.length || (size_t)%s > %s.length - (size_t)%s) { concept_panic(%q, %d, %d); }\n", offsetName, lengthName, offsetName, parentName, lengthName, parentName, offsetName, "Concept span bounds out of range", call.Span.Line, call.Span.Column))
		}
		out.WriteString(ind(indent) + fmt.Sprintf("if ((size_t)%s != 0u && sizeof(%s) > SIZE_MAX / (size_t)%s) { concept_panic(%q, %d, %d); }\n", offsetName, evt1CType(*call.SpanElementType), offsetName, "Concept span offset overflow", call.Span.Line, call.Span.Column))
		out.WriteString(ind(indent) + fmt.Sprintf("if ((size_t)%s != 0u && sizeof(%s) > SIZE_MAX / (size_t)%s) { concept_panic(%q, %d, %d); }\n", lengthName, evt1CType(*call.SpanElementType), lengthName, "Concept span length overflow", call.Span.Line, call.Span.Column))
		data := fmt.Sprintf("%s.data + (size_t)%s", parentName, offsetName)
		if resultType.Name == evt1SpanReadonlyName {
			data = "(const " + evt1CType(*call.SpanElementType) + "*)(" + data + ")"
		}
		return out.String(), fmt.Sprintf("(%s){ .data = %s, .length = (size_t)%s }", evt1CType(resultType), data, lengthName), resultType
	}

	sourcePrelude, sourceValue, sourceType := f.lowerExpr(call.Args[0], indent)
	length := "0u"
	if evt1IsSpanType(sourceType) {
		length = fmt.Sprintf("(%s).length", sourceValue)
	} else if staticLength := 0; call.LengthExpression != "" {
		if _, err := fmt.Sscan(call.LengthExpression, &staticLength); err == nil {
			length = fmt.Sprintf("%du", staticLength)
		} else if sourceType.ArrayElem != nil {
			dimensions := make([]string, 0, len(sourceType.Shape))
			for i, dimension := range sourceType.Shape {
				if sourceType.isReference() && dimension.Runtime {
					dimensions = append(dimensions, fmt.Sprintf("(%s).shape[%d]", sourceValue, i))
				} else {
					dimensions = append(dimensions, fmt.Sprint(dimension.Extent))
				}
			}
			var checked strings.Builder
			checked.WriteString(sourcePrelude)
			length = f.lowerCheckedSpanProduct(&checked, dimensions, "span_source_length", call.Span, indent)
			checked.WriteString(ind(indent) + fmt.Sprintf("if (%s != 0u && sizeof(%s) > SIZE_MAX / %s) { concept_panic(%q, %d, %d); }\n", length, evt1CType(*call.SpanElementType), length, "Concept span length overflow", call.Span.Line, call.Span.Column))
			sourcePrelude = checked.String()
		}
	} else if sourceType.ArrayElem != nil {
		length = fmt.Sprintf("%du", evt1StorageElementCount(sourceType))
	}
	data := fmt.Sprintf("(%s).data", sourceValue)
	pointer := evt1CType(*call.SpanElementType) + "*"
	if resultType.Name == evt1SpanReadonlyName {
		pointer = "const " + evt1CType(*call.SpanElementType) + "*"
	}
	data = fmt.Sprintf("(%s)(%s)", pointer, data)
	return sourcePrelude, fmt.Sprintf("(%s){ .data = %s, .length = %s }", evt1CType(resultType), data, length), resultType
}

func (f *evt1FunctionLowerer) lowerCheckedSpanProduct(out *strings.Builder, dimensions []string, hint string, span Span, indent int) string {
	name := f.nextTemp(hint)
	out.WriteString(ind(indent) + fmt.Sprintf("size_t %s = 1u;\n", name))
	for _, dimension := range dimensions {
		out.WriteString(ind(indent) + fmt.Sprintf("if ((size_t)%s != 0u && %s > SIZE_MAX / (size_t)%s) { concept_panic(%q, %d, %d); }\n", dimension, name, dimension, "Concept span length overflow", span.Line, span.Column))
		out.WriteString(ind(indent) + fmt.Sprintf("%s *= (size_t)%s;\n", name, dimension))
	}
	return name
}

func (call *CallExpr) MutabilityToTypeName() string {
	if call.Mutability == "mutable" {
		return evt1SpanMutableName
	}
	return evt1SpanReadonlyName
}

func (f *evt1FunctionLowerer) lowerSpanIndex(index *IndexExpr, indent int) (string, string, Type) {
	basePrelude, baseValue, baseType := f.lowerExpr(index.Base, indent)
	indexPrelude, indexValue, _ := f.lowerExpr(evt1StorageIndices(index)[0], indent)
	baseName := f.nextTemp("span")
	indexName := f.nextTemp("span_index")
	var out strings.Builder
	out.WriteString(basePrelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(baseType), baseName, baseValue))
	out.WriteString(indexPrelude)
	out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", indexName, indexValue))
	if f.plannedStrategy("span_index", "PerAccessRuntime") == "PerAccessRuntime" {
		out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= %s.length) { concept_panic(%q, %d, %d); }\n", indexName, indexName, baseName, "Concept span index out of bounds", index.Span.Line, index.Span.Column))
	}
	return out.String(), fmt.Sprintf("%s.data[(size_t)%s]", baseName, indexName), evt1SpanElement(baseType)
}
