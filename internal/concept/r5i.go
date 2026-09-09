package concept

import (
	"fmt"
	"sort"
	"strings"
)

// R5i deliberately reuses the ordinary binding scope. A callable body sees
// parameters, locals, explicit captures, and module symbols; its lexical outer
// scope is retained only to produce CALLABLE_CAPTURE_REQUIRED diagnostics.
func evt1ValidateCallableExpr(env *semanticEnv, outer *evt1Scope, expr *CallableExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if inComptimeFn {
		return Type{}, evt1Diagnostic("CALLABLE_COMPTIME_DEFERRED", "runtime callable literals are not compile-time values", expr.Span)
	}
	name := outer.functionName
	if name == "" {
		name = env.validatingFunction
	}
	if name == "" {
		name = "module"
	}
	expr.Identity = fmt.Sprintf("%s#callback%d", name, expr.Ordinal)
	expr.EnvironmentID = expr.Identity + "#environment"

	captureScope := evt1ModuleScope(env)
	captureScope.functionName = name
	captureScope.callableOuter = outer
	captureScope.inAsync = outer.inAsync
	seen := map[string]bool{}
	copyable, hasDrop := true, false
	overall := evt1LifetimeProvenance{Kind: evt1ProvenanceStatic}
	for i := range expr.Captures {
		capture := &expr.Captures[i]
		capture.Ordinal = i
		if seen[capture.Name] {
			return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_DUPLICATE", fmt.Sprintf("capture %s is listed more than once", capture.Name), capture.Span)
		}
		seen[capture.Name] = true
		if capture.Source != nil {
			t, err := validateExpr(env, outer, capture.Source, templateInfo, false)
			if err != nil {
				return Type{}, err
			}
			if !evt1TypeCopyable(env, t) {
				return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_COPY_NONCOPYABLE", fmt.Sprintf("capture initializer for %s has non-copyable type %s", capture.Name, t.String()), capture.Span)
			}
			provenance := evt1ExprProvenance(env, outer, capture.Source)
			if !t.isBorrowLike() && t.Kind != TypeDyn && !evt1IsSpanType(t) && !evt1IsTensorType(t) {
				provenance = evt1LifetimeProvenance{Kind: evt1ProvenanceStatic}
			}
			capture.Type, capture.Mutable, capture.Provenance = evt1CanonicalType(env, t), true, string(provenance.Kind)
			if provenance.Scoped {
				capture.Provenance += ":scoped"
			}
			if evt1CaptureCarriesProvenance(*capture) && (provenance.Kind == evt1ProvenanceLocal || (overall.Kind != evt1ProvenanceLocal && provenance.Kind == evt1ProvenanceParameter)) {
				overall = provenance
			}
			captureScope.declare(capture.Name, evt1ValueBinding{t: capture.Type, mutable: true, state: evt1StorageInitialized, provenance: provenance})
			continue
		}
		binding, ok := outer.lookup(capture.Name)
		if !ok {
			return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_UNKNOWN", fmt.Sprintf("capture source %s is not a lexical binding", capture.Name), capture.Span)
		}
		if err := evt1CheckReadableBinding(capture.Name, binding, capture.Span); err != nil {
			return Type{}, err
		}
		provenance := binding.provenance
		capture.Provenance = string(provenance.Kind)
		if provenance.Scoped {
			capture.Provenance += ":scoped"
		}
		switch capture.Kind {
		case CaptureCopy:
			if !evt1TypeCopyable(env, binding.t) {
				return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_COPY_NONCOPYABLE", fmt.Sprintf("capture %s has non-copyable type %s; use `move %s`", capture.Name, binding.t.String(), capture.Name), capture.Span)
			}
			capture.Type, capture.Mutable = evt1CanonicalType(env, binding.t), true
			captureScope.declare(capture.Name, evt1ValueBinding{t: capture.Type, mutable: true, state: evt1StorageInitialized, provenance: provenance, spanFacts: binding.spanFacts, regionFacts: binding.regionFacts, tensorFacts: binding.tensorFacts})
		case CaptureMove:
			move := &MoveExpr{Value: &NameExpr{Name: capture.Name, Span: capture.Span}, Span: capture.Span}
			t, err := validateExpr(env, outer, move, templateInfo, false)
			if err != nil {
				return Type{}, err
			}
			capture.Type, capture.Mutable = t, true
			copyable = copyable && evt1TypeCopyable(env, t)
			hasDrop = hasDrop || evt1TypeHasDrop(env, t)
			captureScope.declare(capture.Name, evt1ValueBinding{t: t, mutable: true, state: evt1StorageInitialized, provenance: provenance, spanFacts: binding.spanFacts, regionFacts: binding.regionFacts, tensorFacts: binding.tensorFacts})
		case CaptureRef, CaptureRefConst:
			isConst := capture.Kind == CaptureRefConst
			ref := &RefExpr{Value: &NameExpr{Name: capture.Name, Span: capture.Span}, Const: isConst, Span: capture.Span}
			t, err := validateExpr(env, outer, ref, templateInfo, false)
			if err != nil {
				return Type{}, err
			}
			capture.Type, capture.Mutable = t, !isConst
			captureScope.declare(capture.Name, evt1ValueBinding{t: t, mutable: !isConst, state: evt1StorageInitialized, provenance: provenance, spanFacts: binding.spanFacts, regionFacts: binding.regionFacts, tensorFacts: binding.tensorFacts})
		default:
			return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_INVALID", "unknown callable capture kind", capture.Span)
		}
		if evt1CaptureCarriesProvenance(*capture) && (provenance.Kind == evt1ProvenanceLocal || (overall.Kind != evt1ProvenanceLocal && provenance.Kind == evt1ProvenanceParameter)) {
			overall = provenance
		}
	}

	for _, param := range expr.Params {
		if seen[param.Name] {
			return Type{}, evt1Diagnostic("CALLABLE_CAPTURE_DUPLICATE", fmt.Sprintf("parameter %s conflicts with a capture", param.Name), param.Span)
		}
		if err := validateKnownType(env, param.Type, param.Span, "", false); err != nil {
			return Type{}, err
		}
		resolved, err := evt1ResolveType(env, captureScope, param.Type)
		if err != nil {
			return Type{}, err
		}
		captureScope.declare(param.Name, evt1ValueBinding{t: resolved, mutable: !resolved.Const, state: evt1StorageInitialized, provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceParameter}})
	}

	requiresMutable, consumes := evt1CallableBodyEffects(expr.Body, expr.Captures)
	expr.RequiresMutable, expr.Consumes = requiresMutable, consumes
	result, err := evt1CallableResultType(env, captureScope, expr.Body, templateInfo)
	if err != nil {
		return Type{}, err
	}
	expr.ResultType = result
	captureScope.returnType = result
	if err := validateBlock(env, captureScope, result, expr.Body, templateInfo, false); err != nil {
		return Type{}, err
	}
	params := make([]Type, 0, len(expr.Params))
	for _, param := range expr.Params {
		params = append(params, evt1CanonicalType(env, param.Type))
	}
	return Type{Name: expr.Identity, Kind: TypeCallable, CallableParams: params, CallableResult: &expr.ResultType, CallableID: expr.Identity, CallableMutable: requiresMutable, CallableConsumes: consumes, CallableCopyable: copyable, CallableHasDrop: hasDrop, CallableProvenance: string(overall.Kind), Scoped: overall.Scoped, Span: expr.Span}, nil
}

func evt1CaptureCarriesProvenance(capture CaptureBinding) bool {
	return capture.Kind == CaptureRef || capture.Kind == CaptureRefConst || capture.Type.isBorrowLike() || capture.Type.Kind == TypeDyn || evt1IsSpanType(capture.Type) || evt1IsTensorType(capture.Type)
}

func evt1CallableResultType(env *semanticEnv, scope *evt1Scope, block Block, templateInfo *evt1TemplateInfo) (Type, error) {
	var value Expr
	var find func(Block)
	find = func(current Block) {
		for _, stmt := range current.Statements {
			switch s := stmt.(type) {
			case *ReturnStmt:
				if value == nil && s.Value != nil {
					value = s.Value
				}
			case *IfStmt:
				find(s.Then)
				if s.Else != nil {
					find(*s.Else)
				}
			case *Block:
				find(*s)
			}
		}
	}
	find(block)
	if value == nil {
		return Type{Name: "void", Kind: TypeBuiltin, Span: block.Span}, nil
	}
	return validateExpr(env, evt1CloneScope(scope), value, templateInfo, false)
}

func evt1CallableBodyEffects(block Block, captures []CaptureBinding) (bool, bool) {
	kinds := map[string]CaptureKind{}
	for _, capture := range captures {
		kinds[capture.Name] = capture.Kind
	}
	mutable, consumes := false, false
	var visitExpr func(Expr)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *MoveExpr:
			if name, ok := e.Value.(*NameExpr); ok && kinds[name.Name] == CaptureMove {
				mutable, consumes = true, true
			}
		case *CallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		case *BinaryExpr:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *FieldExpr:
			visitExpr(e.Receiver)
		}
	}
	var visitBlock func(Block)
	visitBlock = func(current Block) {
		for _, stmt := range current.Statements {
			switch s := stmt.(type) {
			case *AssignStmt:
				if name, ok := s.Target.(*NameExpr); ok && (kinds[name.Name] == CaptureCopy || kinds[name.Name] == CaptureMove) {
					mutable = true
				}
				visitExpr(s.Value)
			case *ReturnStmt:
				if s.Value != nil {
					visitExpr(s.Value)
				}
			case *ExprStmt:
				visitExpr(s.Value)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *Block:
				visitBlock(*s)
			}
		}
	}
	visitBlock(block)
	return mutable, consumes
}

func evt1ValidateCallableInvocation(env *semanticEnv, scope *evt1Scope, call *CallExpr, binding evt1ValueBinding, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if err := evt1CheckReadableBinding(call.Callee, binding, call.Span); err != nil {
		return Type{}, err
	}
	t := binding.t.valueType()
	if len(call.Args) != len(t.CallableParams) {
		return Type{}, evt1Diagnostic("CALLABLE_ARITY_MISMATCH", fmt.Sprintf("callable expects %d argument(s), got %d", len(t.CallableParams), len(call.Args)), call.Span)
	}
	if t.Kind == TypeCallable && t.CallableMutable && !binding.mutable {
		return Type{}, evt1Diagnostic("CALLABLE_CONST_MUTATING_INVOKE", "callable mutates or consumes its environment and requires mutable callable storage", call.Span)
	}
	for i, arg := range call.Args {
		actual, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if err := validateCallArgument(env, scope, t.CallableParams[i], arg, actual, templateInfo); err != nil {
			return Type{}, err
		}
	}
	call.CallableInvoke, call.CallableType = true, &t
	if t.Kind == TypeCallable && t.CallableConsumes {
		scope.setState(call.Callee, evt1StorageMoved)
	}
	if t.CallableResult == nil {
		return Type{}, evt1Diagnostic("CALLABLE_MIR_INVALID", "callable has no result type", call.Span)
	}
	return evt1CanonicalType(env, *t.CallableResult), nil
}

func evt1ValidateCallbackErasure(env *semanticEnv, scope *evt1Scope, expr Expr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if name, ok := expr.(*NameExpr); ok {
		if binding, found := scope.lookup(name.Name); found && binding.t.Kind == TypeCallback {
			if err := evt1CheckReadableBinding(name.Name, binding, name.Span); err != nil {
				return Type{}, err
			}
			if !evt1TypesCompatible(env, expected, binding.t, "") {
				return Type{}, evt1Diagnostic("CALLBACK_SIGNATURE_MISMATCH", "erased callback signature does not match", name.Span)
			}
			return expected, nil
		}
	}
	if _, temporary := expr.(*CallableExpr); temporary {
		return Type{}, evt1Diagnostic("CALLBACK_OWNING_TEMPORARY_REJECTED", "erased callback is non-owning; bind the concrete callable before borrowing it", expr.exprSpan())
	}
	ref, ok := expr.(*RefExpr)
	if !ok {
		return Type{}, evt1Diagnostic("CALLBACK_REQUIRES_EXPLICIT_BORROW", "erased callback construction requires `ref concreteCallable`", expr.exprSpan())
	}
	name, ok := ref.Value.(*NameExpr)
	if !ok {
		return Type{}, evt1Diagnostic("CALLBACK_REQUIRES_EXPLICIT_BORROW", "erased callback environment must be a named concrete callable", ref.Span)
	}
	binding, ok := scope.lookup(name.Name)
	if !ok || binding.t.Kind != TypeCallable {
		return Type{}, evt1Diagnostic("CALLBACK_REQUIRES_CONCRETE_CALLABLE", fmt.Sprintf("%s is not a concrete callable", name.Name), name.Span)
	}
	concrete := binding.t
	if len(concrete.CallableParams) != len(expected.CallableParams) || concrete.CallableResult == nil || expected.CallableResult == nil {
		return Type{}, evt1Diagnostic("CALLBACK_SIGNATURE_MISMATCH", "concrete callable signature does not match erased callback", ref.Span)
	}
	for i := range concrete.CallableParams {
		if !evt1TypesCompatible(env, expected.CallableParams[i], concrete.CallableParams[i], "") {
			return Type{}, evt1Diagnostic("CALLBACK_SIGNATURE_MISMATCH", "concrete callable parameter types do not match erased callback", ref.Span)
		}
	}
	if !evt1TypesCompatible(env, *expected.CallableResult, *concrete.CallableResult, "") {
		return Type{}, evt1Diagnostic("CALLBACK_SIGNATURE_MISMATCH", "concrete callable result does not match erased callback", ref.Span)
	}
	if concrete.CallableMutable && (ref.Const || !binding.mutable) {
		return Type{}, evt1Diagnostic("CALLABLE_CONST_MUTATING_INVOKE", "mutable callable cannot be erased through readonly storage", ref.Span)
	}
	ref.CallbackErase, ref.CallbackType = true, &expected
	return expected, nil
}

func evt1CallableMIR(env *semanticEnv, expr *CallableExpr) MIRCallable {
	environment := MIRCaptureEnvironment{Identity: expr.EnvironmentID, Alignment: 1, Layout: "NaturalCaptureOrder"}
	offset := 0
	for _, capture := range expr.Captures {
		size, alignment := evt1CallableTypeGeometry(env, capture.Type)
		if remainder := offset % alignment; remainder != 0 {
			offset += alignment - remainder
		}
		offset += size
		if alignment > environment.Alignment {
			environment.Alignment = alignment
		}
		mutability := "mutable"
		if capture.Kind == CaptureRefConst {
			mutability = "readonly"
		}
		ownership := map[CaptureKind]string{CaptureCopy: "CopyIntoEnvironment", CaptureMove: "MoveIntoEnvironment", CaptureRef: "BorrowMutable", CaptureRefConst: "BorrowReadonly"}[capture.Kind]
		cleanupTracked := capture.Kind == CaptureMove && evt1TypeHasDrop(env, capture.Type)
		if cleanupTracked {
			offset++ // the generated environment stores an explicit live bit after the field
		}
		source := capture.Name
		if capture.Source != nil {
			source = evt1ExprIdentity(capture.Source)
		}
		environment.Fields = append(environment.Fields, MIRCaptureBinding{Source: source, Kind: capture.Kind, Type: capture.Type, Provenance: capture.Provenance, Mutability: mutability, OwnershipEffect: ownership, FieldIdentity: expr.EnvironmentID + "." + capture.Name, RegionIdentity: capture.Provenance + ":" + capture.Name, CleanupTracked: cleanupTracked, NoAllocation: true, EvaluationOrder: capture.Ordinal})
	}
	if len(expr.Captures) == 0 {
		offset = 1 // C11 has no zero-sized structs; the backend emits one byte.
	}
	if remainder := offset % environment.Alignment; remainder != 0 {
		offset += environment.Alignment - remainder
	}
	environment.Size = offset
	params := make([]MIRName, 0, len(expr.Params))
	for _, param := range expr.Params {
		params = append(params, MIRName{Name: param.Name, Type: param.Type})
	}
	return MIRCallable{Identity: expr.Identity, Environment: environment, Params: params, ResultType: expr.ResultType, RequiresMutable: expr.RequiresMutable, Consumes: expr.Consumes, Dispatch: "DirectCallable", NoAllocation: true, SourceSpan: expr.Span}
}

func evt1CallableTypeGeometry(env *semanticEnv, t Type) (int, int) {
	if t.Kind == TypeDyn || t.Kind == TypeCallback || evt1IsSpanType(t) {
		return 16, 8
	}
	if t.isBorrowLike() || t.PointerTo != nil {
		return 8, 8
	}
	if evt1IsTensorType(t) {
		return 8 * (1 + t.TensorRank), 8
	}
	if size, alignment, err := evt1TypeGeometry(env, t.valueType()); err == nil && size > 0 && alignment > 0 {
		return size, alignment
	}
	// GenericC11Target is a 64-bit ABI. Any representation introduced after
	// R5i must add an explicit case above rather than silently changing this law.
	return 8, 8
}

func evt1CallableConsumedCaptures(block Block) map[string]bool {
	out := map[string]bool{}
	var visitExpr func(Expr)
	var visitBlock func(Block)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *MoveExpr:
			if name, ok := e.Value.(*NameExpr); ok {
				out[name.Name] = true
			}
			visitExpr(e.Value)
		case *CallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		case *BinaryExpr:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *FieldExpr:
			visitExpr(e.Receiver)
		}
	}
	visitBlock = func(current Block) {
		for _, stmt := range current.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				visitExpr(s.Value)
			case *AssignStmt:
				visitExpr(s.Value)
			case *ExprStmt:
				visitExpr(s.Value)
			case *ReturnStmt:
				if s.Value != nil {
					visitExpr(s.Value)
				}
			case *IfStmt:
				visitExpr(s.Condition)
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *Block:
				visitBlock(*s)
			}
		}
	}
	visitBlock(block)
	return out
}

func evt1CallableCIdentity(identity string) string {
	return strings.NewReplacer("#", "_", ".", "_", "-", "_").Replace(evt1CName(identity))
}
func evt1CallableEnvCName(identity string) string {
	return "concept_" + evt1CallableCIdentity(identity) + "_environment"
}
func evt1CallableInvokeCName(identity string) string {
	return "concept_" + evt1CallableCIdentity(identity) + "_invoke"
}
func evt1CallbackCName(t Type) string { return "concept_callback_" + digest([]byte(t.String()))[:12] }
func evt1CallbackAdapterCName(callable string, callback Type) string {
	return "concept_" + evt1CallableCIdentity(callable) + "_as_" + evt1CallbackCName(callback)
}
func evt1CArgsSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func evt1ModuleCallables(module Module) []*CallableExpr {
	var out []*CallableExpr
	var visitExpr func(Expr)
	var visitBlock func(Block)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *CallableExpr:
			out = append(out, e)
			for _, capture := range e.Captures {
				if capture.Source != nil {
					visitExpr(capture.Source)
				}
			}
		case *CallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		case *BinaryExpr:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *UnaryExpr:
			visitExpr(e.Value)
		case *MoveExpr:
			visitExpr(e.Value)
		case *RefExpr:
			visitExpr(e.Value)
		case *FieldExpr:
			visitExpr(e.Receiver)
		case *IndexExpr:
			visitExpr(e.Base)
			for _, index := range evt1StorageIndices(e) {
				visitExpr(index)
			}
		case *ParenExpr:
			visitExpr(e.Value)
		}
	}
	visitBlock = func(block Block) {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				visitExpr(s.Value)
			case *AssignStmt:
				visitExpr(s.Target)
				visitExpr(s.Value)
			case *ReturnStmt:
				if s.Value != nil {
					visitExpr(s.Value)
				}
			case *ExprStmt:
				visitExpr(s.Value)
			case *IfStmt:
				visitExpr(s.Condition)
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitExpr(s.Condition)
				visitBlock(s.Body)
			case *Block:
				visitBlock(*s)
			}
		}
	}
	for _, fn := range module.Functions {
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ordinal < out[j].Ordinal })
	return out
}

func evt1ModuleCallbackTypes(module Module) []Type {
	seen := map[string]Type{}
	add := func(t Type) {
		if t.Kind == TypeCallback {
			seen[t.String()] = t
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				add(s.Type)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitBlock(s.Body)
			case *Block:
				visitBlock(*s)
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
	var keys []string
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Type, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out
}

func evt1ModuleCallbackWitnesses(module Module) []MIRCallbackWitness {
	seen := map[string]MIRCallbackWitness{}
	callables := evt1ModuleCallables(module)
	for _, fn := range module.Functions {
		if fn.Body == nil {
			continue
		}
		var refs []*RefExpr
		evt1VisitCallbackErasures(*fn.Body, &refs)
		for _, ref := range refs {
			if ref.CallbackType == nil {
				continue
			}
			name := ref.Value.(*NameExpr).Name
			for _, callable := range callables {
				if !evt1BlockBindsCallable(*fn.Body, name, callable) {
					continue
				}
				callback := *ref.CallbackType
				id := callable.Identity + "#callback-witness#" + digest([]byte(callback.String()))[:12]
				seen[id] = MIRCallbackWitness{ID: id, ConcreteCallable: callable.Identity, Signature: callback, Adapter: evt1CallbackAdapterCName(callable.Identity, callback), Environment: callable.EnvironmentID, NoAllocation: true}
				break
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]MIRCallbackWitness, 0, len(ids))
	for _, id := range ids {
		out = append(out, seen[id])
	}
	return out
}

func (l *lowering) callableDeclarations() string {
	var b strings.Builder
	for _, callback := range evt1ModuleCallbackTypes(l.module) {
		b.WriteString("typedef struct { void *environment; ")
		b.WriteString(evt1CType(*callback.CallableResult) + " (*invoke)(void *environment")
		for i, param := range callback.CallableParams {
			b.WriteString(fmt.Sprintf(", %s arg%d", evt1CType(param), i))
		}
		b.WriteString("); } " + evt1CallbackCName(callback) + ";\n\n")
	}
	for _, callable := range evt1ModuleCallables(l.module) {
		b.WriteString("typedef struct {\n")
		if len(callable.Captures) == 0 {
			b.WriteString("  unsigned char _empty;\n")
		}
		for _, capture := range callable.Captures {
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(capture.Type), capture.Name))
			if capture.Kind == CaptureMove && evt1TypeHasDrop(l.env, capture.Type) {
				b.WriteString(fmt.Sprintf("  bool __live_%s;\n", capture.Name))
			}
		}
		b.WriteString("} " + evt1CallableEnvCName(callable.Identity) + ";\n")
		b.WriteString(evt1CType(callable.ResultType) + " " + evt1CallableInvokeCName(callable.Identity) + "(" + evt1CallableEnvCName(callable.Identity) + " *environment")
		for _, param := range callable.Params {
			b.WriteString(fmt.Sprintf(", %s %s", evt1CType(param.Type), param.Name))
		}
		b.WriteString(");\n\n")
	}
	return b.String()
}

func (l *lowering) callableDefinitions() string {
	var b strings.Builder
	for _, callable := range evt1ModuleCallables(l.module) {
		fn := FunctionDecl{Name: callable.Identity, ReturnType: callable.ResultType, Params: callable.Params, Body: &callable.Body, Span: callable.Span}
		lower := newEVT1FunctionLowerer(l, fn, evt1CallableInvokeCName(callable.Identity), false)
		for _, capture := range callable.Captures {
			lower.scope[0][capture.Name] = evt1Binding{cName: "environment->" + capture.Name, t: capture.Type}
		}
		b.WriteString(fmt.Sprintf("%s %s(%s *environment", evt1CType(callable.ResultType), evt1CallableInvokeCName(callable.Identity), evt1CallableEnvCName(callable.Identity)))
		for _, param := range callable.Params {
			b.WriteString(fmt.Sprintf(", %s %s", evt1CType(param.Type), param.Name))
		}
		b.WriteString(") {\n")
		consumed := evt1CallableConsumedCaptures(callable.Body)
		for _, capture := range callable.Captures {
			if consumed[capture.Name] && capture.Kind == CaptureMove && evt1TypeHasDrop(l.env, capture.Type) {
				b.WriteString(fmt.Sprintf("  environment->__live_%s = false;\n", capture.Name))
			}
		}
		b.WriteString(lower.lowerBlock(callable.Body, 1))
		b.WriteString("}\n\n")
	}
	for _, fn := range l.module.Functions {
		if fn.Body == nil {
			continue
		}
		var refs []*RefExpr
		evt1VisitCallbackErasures(*fn.Body, &refs)
		for _, ref := range refs {
			name := ref.Value.(*NameExpr).Name
			var concrete *CallableExpr
			for _, candidate := range evt1ModuleCallables(l.module) {
				if candidate.Identity != "" && evt1BlockBindsCallable(*fn.Body, name, candidate) {
					concrete = candidate
					break
				}
			}
			if concrete == nil || ref.CallbackType == nil {
				continue
			}
			callback := *ref.CallbackType
			b.WriteString(fmt.Sprintf("static %s %s(void *raw", evt1CType(*callback.CallableResult), evt1CallbackAdapterCName(concrete.Identity, callback)))
			for i, param := range callback.CallableParams {
				b.WriteString(fmt.Sprintf(", %s arg%d", evt1CType(param), i))
			}
			b.WriteString(") {\n  ")
			if callback.CallableResult.Name != "void" {
				b.WriteString("return ")
			}
			b.WriteString(evt1CallableInvokeCName(concrete.Identity) + "((" + evt1CallableEnvCName(concrete.Identity) + "*)raw")
			for i := range callback.CallableParams {
				b.WriteString(fmt.Sprintf(", arg%d", i))
			}
			b.WriteString(");\n}\n\n")
		}
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerCallableLiteral(expr *CallableExpr, indent int) (string, string, Type) {
	var b strings.Builder
	var fields []string
	for _, capture := range expr.Captures {
		var source Expr = capture.Source
		if source == nil {
			source = &NameExpr{Name: capture.Name, Span: capture.Span}
		}
		if capture.Kind == CaptureMove {
			source = &MoveExpr{Value: source, Span: capture.Span}
		}
		if capture.Kind == CaptureRef || capture.Kind == CaptureRefConst {
			source = &RefExpr{Value: source, Const: capture.Kind == CaptureRefConst, Span: capture.Span}
		}
		prelude, value, valueType := f.lowerExpr(source, indent)
		b.WriteString(prelude)
		temp := f.nextTemp("capture_" + capture.Name)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(valueType), temp, value))
		fields = append(fields, "."+capture.Name+" = "+temp)
		if capture.Kind == CaptureMove && evt1TypeHasDrop(f.l.env, capture.Type) {
			fields = append(fields, ".__live_"+capture.Name+" = true")
		}
	}
	if len(fields) == 0 {
		fields = append(fields, "._empty = 0")
	}
	t := Type{Name: expr.Identity, Kind: TypeCallable, CallableParams: nil, CallableResult: &expr.ResultType, CallableID: expr.Identity, CallableMutable: expr.RequiresMutable, CallableConsumes: expr.Consumes, CallableCopyable: true, Span: expr.Span}
	for _, param := range expr.Params {
		t.CallableParams = append(t.CallableParams, param.Type)
	}
	return b.String(), "(" + evt1CallableEnvCName(expr.Identity) + "){ " + strings.Join(fields, ", ") + " }", t
}

func evt1VisitCallbackErasures(block Block, out *[]*RefExpr) {
	var visitExpr func(Expr)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *RefExpr:
			if e.CallbackErase {
				*out = append(*out, e)
			}
		case *CallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		}
	}
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			visitExpr(s.Value)
		case *ExprStmt:
			visitExpr(s.Value)
		case *ReturnStmt:
			if s.Value != nil {
				visitExpr(s.Value)
			}
		case *IfStmt:
			evt1VisitCallbackErasures(s.Then, out)
			if s.Else != nil {
				evt1VisitCallbackErasures(*s.Else, out)
			}
		case *Block:
			evt1VisitCallbackErasures(*s, out)
		}
	}
}

func evt1BlockBindsCallable(block Block, name string, callable *CallableExpr) bool {
	for _, stmt := range block.Statements {
		if decl, ok := stmt.(*VarDecl); ok && decl.Name == name && decl.Value == callable {
			return true
		}
	}
	return false
}
