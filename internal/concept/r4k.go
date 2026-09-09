package concept

import (
	"fmt"
	"sort"
	"strings"
)

// evt1InterfaceWitness is derived from ordinary concept satisfaction. It is
// retained only when a dyn value reifies that proof at runtime.
type evt1InterfaceWitness struct {
	ID            string
	Interface     ConceptDecl
	Concrete      Type
	Methods       []FunctionDecl
	Fields        []FieldRequirement
	Prerequisites []string
}

func evt1WitnessID(interfaceName string, concrete Type) string {
	return interfaceName + "__" + evt1TypeIdentity(concrete.valueType())
}

func evt1FieldVisibility(decl StructDecl, name string) string {
	for _, field := range decl.Fields {
		if field.Name == name {
			if field.Visibility != "" {
				return field.Visibility
			}
			if decl.Class {
				return "private"
			}
			return "public"
		}
	}
	return ""
}

func evt1MethodCandidates(env *semanticEnv, owner, name string) []FunctionDecl {
	var out []FunctionDecl
	for _, fn := range env.functions[name] {
		if fn.MethodOf == owner {
			out = append(out, fn)
		}
	}
	return out
}

func evt1InterfaceRuntimeRequirements(env *semanticEnv, name string, seen map[string]bool) ([]OperationRequirement, []FieldRequirement, []string) {
	if seen[name] {
		return nil, nil, nil
	}
	seen[name] = true
	decl := env.concepts[name]
	var methods []OperationRequirement
	var fields []FieldRequirement
	var prerequisites []string
	for _, raw := range decl.Requirements {
		switch req := raw.(type) {
		case *OperationRequirement:
			methods = append(methods, *req)
		case *FieldRequirement:
			fields = append(fields, *req)
		case *PrerequisiteRequirement:
			prerequisites = append(prerequisites, req.ConceptName)
			m, f, p := evt1InterfaceRuntimeRequirements(env, req.ConceptName, seen)
			methods, fields, prerequisites = append(methods, m...), append(fields, f...), append(prerequisites, p...)
		}
	}
	return methods, fields, prerequisites
}

func evt1BuildInterfaceWitness(env *semanticEnv, interfaceName string, concrete Type, span Span) (*evt1InterfaceWitness, error) {
	if existing := env.dynWitnesses[evt1WitnessID(interfaceName, concrete)]; existing != nil {
		return existing, nil
	}
	decl, ok := env.concepts[interfaceName]
	if !ok || !decl.Interface {
		return nil, evt1Diagnostic("DYN_REQUIRES_INTERFACE", fmt.Sprintf("dyn requires an interface, got %s", interfaceName), span)
	}
	if err := checkConceptSatisfaction(env, interfaceName, concrete, nil, span); err != nil {
		return nil, evt1Diagnostic("DYN_CONCRETE_TYPE_DOES_NOT_SATISFY", err.Error(), span)
	}
	methodReqs, fieldReqs, prerequisites := evt1InterfaceRuntimeRequirements(env, interfaceName, map[string]bool{})
	w := &evt1InterfaceWitness{ID: evt1WitnessID(interfaceName, concrete), Interface: decl, Concrete: concrete.valueType(), Fields: fieldReqs, Prerequisites: prerequisites}
	for _, raw := range methodReqs {
		required := evt1SubstituteRequirement(raw, decl.TypeParam, concrete.valueType())
		fn, err := evt1LookupRequiredOperation(env, required, span, interfaceName)
		if err != nil {
			return nil, evt1Diagnostic("INTERFACE_WITNESS_INVALID", err.Error(), span)
		}
		w.Methods = append(w.Methods, fn)
	}
	env.dynWitnesses[w.ID] = w
	return w, nil
}

func evt1SortedWitnesses(env *semanticEnv) []*evt1InterfaceWitness {
	ids := make([]string, 0, len(env.dynWitnesses))
	for id := range env.dynWitnesses {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*evt1InterfaceWitness, 0, len(ids))
	for _, id := range ids {
		out = append(out, env.dynWitnesses[id])
	}
	return out
}

func evt1PrepareDynInitializer(env *semanticEnv, target Type, value Expr, span Span) error {
	ref, ok := value.(*RefExpr)
	if !ok {
		return evt1Diagnostic("DYN_REQUIRES_EXPLICIT_REF", "dyn construction requires an explicit ref or ref const source", span)
	}
	lvalue, ok := ref.Value.(*NameExpr)
	if !ok {
		return evt1Diagnostic("DYN_REQUIRES_EXPLICIT_REF", "dyn construction requires a named storage place", ref.Value.exprSpan())
	}
	_ = lvalue
	if !target.Const && ref.Const {
		return evt1Diagnostic("DYN_MUTABLE_FROM_CONST", "mutable dyn cannot be constructed from ref const", ref.Span)
	}
	ref.DynInterface = target.Name
	return nil
}

func evt1ValidateMemberCall(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	receiverType, err := validateExpr(env, scope, call.Receiver, templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	if receiverType.Kind == TypeDyn {
		decl := env.concepts[receiverType.Name]
		methods, _, _ := evt1InterfaceRuntimeRequirements(env, receiverType.Name, map[string]bool{})
		var matches []OperationRequirement
		for _, req := range methods {
			if req.Name == call.Callee {
				matches = append(matches, req)
			}
		}
		if len(matches) == 0 {
			return Type{}, evt1Diagnostic("DYN_METHOD_NOT_IN_INTERFACE", fmt.Sprintf("method %s is not in interface %s", call.Callee, receiverType.Name), call.Span)
		}
		if len(matches) != 1 {
			return Type{}, evt1Diagnostic("INTERFACE_WITNESS_INVALID", fmt.Sprintf("interface method %s is ambiguous", call.Callee), call.Span)
		}
		req := matches[0]
		if len(req.Params) == 0 || req.Params[0].Type.Kind != TypeConceptParam || !req.Params[0].Type.isReference() {
			return Type{}, evt1Diagnostic("INTERFACE_NOT_DYN_COMPATIBLE", fmt.Sprintf("interface method %s requires ref %s self as its first parameter", req.Name, decl.TypeParam), req.Span)
		}
		if receiverType.Const && !req.Params[0].Type.Const {
			return Type{}, evt1Diagnostic("DYN_MUTABLE_FROM_CONST", fmt.Sprintf("mutable method %s cannot be called through dyn const %s", req.Name, receiverType.Name), call.Span)
		}
		if len(call.Args) != len(req.Params)-1 {
			return Type{}, evt1Diagnostic("CV4106", fmt.Sprintf("dyn method %s expects %d argument(s), got %d", req.Name, len(req.Params)-1, len(call.Args)), call.Span)
		}
		for i, arg := range call.Args {
			argType, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			param := evt1SubstituteType(req.Params[i+1].Type, decl.TypeParam, Type{Name: "void", Kind: TypeBuiltin})
			if err := validateCallArgument(env, scope, param, arg, argType, templateInfo); err != nil {
				return Type{}, err
			}
		}
		call.DynDispatch = true
		call.DynInterface = receiverType.Name
		return evt1CanonicalType(env, req.ReturnType), nil
	}

	selfConst := false
	if place, placeErr := validateAssignable(env, scope, call.Receiver, templateInfo); placeErr == nil {
		selfConst = !place.mutable
	}
	self := &RefExpr{Value: call.Receiver, Const: selfConst, Span: call.Span}
	args := append([]Expr{self}, call.Args...)
	if templateInfo != nil && evt1TypeDependsOnParam(receiverType, templateInfo.Decl.TypeParam) {
		synthetic := CallExpr{Callee: call.Callee, Args: args, Span: call.Span}
		return validateTemplateCallExpr(env, scope, synthetic, templateInfo)
	}
	candidates := evt1MethodCandidates(env, receiverType.valueType().Name, call.Callee)
	if len(candidates) == 0 {
		return Type{}, evt1Diagnostic("DYN_METHOD_NOT_IN_INTERFACE", fmt.Sprintf("unknown member method %s.%s", receiverType.valueType().Name, call.Callee), call.Span)
	}
	argTypes := make([]Type, 0, len(args))
	for _, arg := range args {
		t, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		argTypes = append(argTypes, t)
	}
	all := env.functions[call.Callee]
	env.functions[call.Callee] = candidates
	fn, err := evt1ResolveOrdinaryCall(env, scope, call.Callee, args, argTypes, templateInfo, call.Span)
	env.functions[call.Callee] = all
	if err != nil {
		return Type{}, err
	}
	if fn.MethodOf == "" || fn.MethodOf != receiverType.valueType().Name {
		return Type{}, evt1Diagnostic("CLASS_METHOD_RECEIVER_INVALID", fmt.Sprintf("%s is not a method of %s", call.Callee, receiverType.String()), call.Span)
	}
	if fn.Visibility == "private" && env.validatingMethod != fn.MethodOf {
		return Type{}, evt1Diagnostic("CLASS_PRIVATE_MEMBER_ACCESS", fmt.Sprintf("method %s.%s is private", fn.MethodOf, fn.Name), call.Span)
	}
	for i, arg := range args {
		if err := validateCallArgument(env, scope, fn.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
			return Type{}, err
		}
	}
	if fn.Async {
		return evt1AsyncType(evt1CanonicalType(env, fn.ReturnType), fn.Name, call.Span), nil
	}
	return evt1CanonicalType(env, fn.ReturnType), nil
}

// Aggregate method bodies allow unambiguous bare member names. This syntax
// normalization makes the explicit receiver visible to all later stages.
func evt1RewriteAggregateMethods(decl *StructDecl) {
	fields := map[string]bool{}
	methods := map[string]bool{}
	for _, f := range decl.Fields {
		fields[f.Name] = true
	}
	for _, m := range decl.Methods {
		methods[m.Name] = true
	}
	for i := range decl.Methods {
		fn := &decl.Methods[i]
		if fn.Body == nil {
			continue
		}
		shadow := map[string]bool{"self": true}
		for _, p := range fn.Params {
			shadow[p.Name] = true
		}
		evt1RewriteMemberBlock(fn.Body, fields, methods, shadow)
	}
}

func evt1RewriteMemberBlock(block *Block, fields, methods, inherited map[string]bool) {
	shadow := map[string]bool{}
	for k, v := range inherited {
		shadow[k] = v
	}
	for _, raw := range block.Statements {
		switch s := raw.(type) {
		case *VarDecl:
			s.Value = evt1RewriteMemberExpr(s.Value, fields, methods, shadow)
			shadow[s.Name] = true
		case *AssignStmt:
			s.Target = evt1RewriteMemberExpr(s.Target, fields, methods, shadow)
			s.Value = evt1RewriteMemberExpr(s.Value, fields, methods, shadow)
		case *ReturnStmt:
			s.Value = evt1RewriteMemberExpr(s.Value, fields, methods, shadow)
		case *ExprStmt:
			s.Value = evt1RewriteMemberExpr(s.Value, fields, methods, shadow)
		case *IfStmt:
			s.Condition = evt1RewriteMemberExpr(s.Condition, fields, methods, shadow)
			evt1RewriteMemberBlock(&s.Then, fields, methods, shadow)
			if s.Else != nil {
				evt1RewriteMemberBlock(s.Else, fields, methods, shadow)
			}
		case *WhileStmt:
			s.Condition = evt1RewriteMemberExpr(s.Condition, fields, methods, shadow)
			evt1RewriteMemberBlock(&s.Body, fields, methods, shadow)
		case *ForeachStmt:
			s.Source = evt1RewriteMemberExpr(s.Source, fields, methods, shadow)
			evt1RewriteMemberBlock(&s.Body, fields, methods, shadow)
		case *Block:
			evt1RewriteMemberBlock(s, fields, methods, shadow)
		}
	}
}

func evt1RewriteMemberExpr(expr Expr, fields, methods, shadow map[string]bool) Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *NameExpr:
		if fields[e.Name] && !shadow[e.Name] {
			return &FieldExpr{Receiver: &NameExpr{Name: "self", Span: e.Span}, Field: e.Name, Span: e.Span}
		}
	case *FieldExpr:
		e.Receiver = evt1RewriteMemberExpr(e.Receiver, fields, methods, shadow)
	case *CallExpr:
		if e.Receiver != nil {
			e.Receiver = evt1RewriteMemberExpr(e.Receiver, fields, methods, shadow)
		} else if methods[e.Callee] {
			e.Receiver = &NameExpr{Name: "self", Span: e.Span}
			e.Member = true
		}
		for i := range e.Args {
			e.Args[i] = evt1RewriteMemberExpr(e.Args[i], fields, methods, shadow)
		}
	case *BinaryExpr:
		e.Left = evt1RewriteMemberExpr(e.Left, fields, methods, shadow)
		e.Right = evt1RewriteMemberExpr(e.Right, fields, methods, shadow)
	case *UnaryExpr:
		e.Value = evt1RewriteMemberExpr(e.Value, fields, methods, shadow)
	case *RefExpr:
		e.Value = evt1RewriteMemberExpr(e.Value, fields, methods, shadow)
	case *MoveExpr:
		e.Value = evt1RewriteMemberExpr(e.Value, fields, methods, shadow)
	case *ParenExpr:
		e.Value = evt1RewriteMemberExpr(e.Value, fields, methods, shadow)
	case *IndexExpr:
		e.Base = evt1RewriteMemberExpr(e.Base, fields, methods, shadow)
		for i := range e.Indices {
			e.Indices[i] = evt1RewriteMemberExpr(e.Indices[i], fields, methods, shadow)
		}
	}
	return expr
}

func evt1DynCName(t Type) string {
	name := "concept_dyn_" + evt1CName(t.Name)
	if t.Const {
		name += "_const"
	}
	return name
}

func evt1InterfaceWitnessCName(name string) string { return "concept_" + evt1CName(name) + "_witness" }
func evt1WitnessTableCName(id string) string       { return "concept_witness_" + evt1CName(id) }
func evt1WitnessAdapterCName(id, member string) string {
	return "concept_witness_" + evt1CName(id) + "_" + evt1CName(member)
}

func evt1WitnessSelfArgument(witness *evt1InterfaceWitness, fn FunctionDecl, concretePtr string) string {
	object := "((" + concretePtr + ")object)"
	if witness.Concrete.ArrayElem == nil || len(fn.Params) == 0 {
		return object
	}
	dimensions := make([]string, 0, evt1StorageRank(witness.Concrete))
	if len(witness.Concrete.Shape) == 0 {
		dimensions = append(dimensions, fmt.Sprintf("%d", witness.Concrete.ArrayLength))
	} else {
		for _, dimension := range witness.Concrete.Shape {
			dimensions = append(dimensions, fmt.Sprintf("%d", dimension.Extent))
		}
	}
	return fmt.Sprintf("(%s){ .data = %s->data, .shape = { %s } }", evt1CType(fn.Params[0].Type), object, strings.Join(dimensions, ", "))
}

func evt1InterfaceMethodRequirement(env *semanticEnv, interfaceName, method string) (OperationRequirement, bool) {
	methods, _, _ := evt1InterfaceRuntimeRequirements(env, interfaceName, map[string]bool{})
	for _, req := range methods {
		if req.Name == method {
			return req, true
		}
	}
	return OperationRequirement{}, false
}

func (l *lowering) interfaceWitnessDeclarations() string {
	var b strings.Builder
	for _, decl := range l.module.Concepts {
		if !decl.Interface {
			continue
		}
		methods, fields, _ := evt1InterfaceRuntimeRequirements(l.env, decl.Name, map[string]bool{})
		wname := evt1InterfaceWitnessCName(decl.Name)
		b.WriteString("typedef struct " + wname + " {\n")
		for _, req := range methods {
			selfType := "void*"
			if len(req.Params) > 0 && req.Params[0].Type.Const {
				selfType = "const void*"
			}
			b.WriteString(fmt.Sprintf("  %s (*%s)(%s", evt1CType(req.ReturnType), req.Name, selfType))
			for _, param := range req.Params[1:] {
				b.WriteString(fmt.Sprintf(", %s %s", evt1CType(param.Type), param.Name))
			}
			b.WriteString(");\n")
		}
		for _, req := range fields {
			b.WriteString(fmt.Sprintf("  %s (*get_%s)(const void* object);\n", evt1CType(req.Type.valueType()), req.Name))
			if !req.Readonly {
				b.WriteString(fmt.Sprintf("  void (*set_%s)(void* object, %s value);\n", req.Name, evt1CType(req.Type.valueType())))
			}
		}
		b.WriteString("} " + wname + ";\n")
		b.WriteString(fmt.Sprintf("typedef struct { void* object; const %s* witness; } %s;\n", wname, evt1DynCName(Type{Name: decl.Name, Kind: TypeDyn})))
		b.WriteString(fmt.Sprintf("typedef struct { const void* object; const %s* witness; } %s;\n\n", wname, evt1DynCName(Type{Name: decl.Name, Kind: TypeDyn, Const: true})))
	}
	return b.String()
}

func (l *lowering) interfaceWitnessDefinitions() string {
	var b strings.Builder
	for _, witness := range evt1SortedWitnesses(l.env) {
		methodReqs, fieldReqs, _ := evt1InterfaceRuntimeRequirements(l.env, witness.Interface.Name, map[string]bool{})
		for i, fn := range witness.Methods {
			req := methodReqs[i]
			adapter := evt1WitnessAdapterCName(witness.ID, req.Name)
			selfType := "void*"
			concretePtr := evt1CType(witness.Concrete) + "*"
			if req.Params[0].Type.Const {
				selfType, concretePtr = "const void*", "const "+evt1CType(witness.Concrete)+"*"
			}
			b.WriteString(fmt.Sprintf("static %s %s(%s object", evt1CType(req.ReturnType), adapter, selfType))
			for _, param := range req.Params[1:] {
				b.WriteString(fmt.Sprintf(", %s %s", evt1CType(param.Type), param.Name))
			}
			b.WriteString(") { ")
			if req.ReturnType.Name != "void" {
				b.WriteString("return ")
			}
			b.WriteString(evt1FunctionSymbolForDecl(l.outputBase, l.env, fn) + "(" + evt1WitnessSelfArgument(witness, fn, concretePtr))
			for _, param := range req.Params[1:] {
				b.WriteString(", " + param.Name)
			}
			b.WriteString("); }\n")
		}
		for _, field := range fieldReqs {
			get := evt1WitnessAdapterCName(witness.ID, "get_"+field.Name)
			b.WriteString(fmt.Sprintf("static %s %s(const void* object) { return ((const %s*)object)->%s; }\n", evt1CType(field.Type.valueType()), get, evt1CType(witness.Concrete), field.Name))
			if !field.Readonly {
				set := evt1WitnessAdapterCName(witness.ID, "set_"+field.Name)
				b.WriteString(fmt.Sprintf("static void %s(void* object, %s value) { ((%s*)object)->%s = value; }\n", set, evt1CType(field.Type.valueType()), evt1CType(witness.Concrete), field.Name))
			}
		}
		b.WriteString(fmt.Sprintf("static const %s %s = {\n", evt1InterfaceWitnessCName(witness.Interface.Name), evt1WitnessTableCName(witness.ID)))
		for _, req := range methodReqs {
			b.WriteString(fmt.Sprintf("  .%s = %s,\n", req.Name, evt1WitnessAdapterCName(witness.ID, req.Name)))
		}
		for _, field := range fieldReqs {
			b.WriteString(fmt.Sprintf("  .get_%s = %s,\n", field.Name, evt1WitnessAdapterCName(witness.ID, "get_"+field.Name)))
			if !field.Readonly {
				b.WriteString(fmt.Sprintf("  .set_%s = %s,\n", field.Name, evt1WitnessAdapterCName(witness.ID, "set_"+field.Name)))
			}
		}
		b.WriteString("};\n\n")
	}
	return b.String()
}
