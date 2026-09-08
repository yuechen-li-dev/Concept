package concept

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type evt1Scope struct {
	parent      *evt1Scope
	values      map[string]evt1ValueBinding
	borrows     []evt1RetainedBorrow
	depth       int
	returnType  Type
	tryHandlers map[string]Type
}

type evt1ProvenanceKind string

const (
	evt1ProvenanceUnknown   evt1ProvenanceKind = "unknown"
	evt1ProvenanceStatic    evt1ProvenanceKind = "static"
	evt1ProvenanceParameter evt1ProvenanceKind = "parameter"
	evt1ProvenanceLocal     evt1ProvenanceKind = "local"
)

type evt1LifetimeProvenance struct {
	Kind           evt1ProvenanceKind
	Depth          int
	ParameterIndex int
	Scoped         bool
}

type evt1ResultProvenanceKind string

const (
	evt1ResultProvenanceUnknown    evt1ResultProvenanceKind = "unknown"
	evt1ResultProvenanceStatic     evt1ResultProvenanceKind = "static"
	evt1ResultProvenanceParameter  evt1ResultProvenanceKind = "parameter"
	evt1ResultProvenanceShortestOf evt1ResultProvenanceKind = "shortest_of_parameters"
)

type evt1ResultProvenanceSummary struct {
	Kind             evt1ResultProvenanceKind
	ParameterIndices []int
}

type evt1ValueBinding struct {
	t                Type
	mutable          bool
	state            evt1StorageState
	comptime         bool
	hasValue         bool
	value            Value
	instanceAutomata string
	batchAutomata    string
	actuatorName     string
	provenance       evt1LifetimeProvenance
}

type evt1StorageState string

const (
	evt1StorageUninitialized evt1StorageState = "uninitialized"
	evt1StorageInitialized   evt1StorageState = "initialized"
	evt1StorageMoved         evt1StorageState = "moved"
	evt1StorageMaybeMoved    evt1StorageState = "maybe_moved"
)

type evt1AccessPath struct {
	Root   string
	Fields []string
	Span   Span
}

type evt1RetainedBorrow struct {
	InstanceName string
	AutomataName string
	ContextName  string
	Path         evt1AccessPath
	Type         Type
	Span         Span
}

type evt1LValue struct {
	t              Type
	mutable        bool
	wholeValue     bool
	readOnlyReason string
	path           evt1AccessPath
}

func newEVT1Scope(parent *evt1Scope) *evt1Scope {
	depth := 0
	if parent != nil {
		depth = parent.depth + 1
	}
	s := &evt1Scope{parent: parent, values: map[string]evt1ValueBinding{}, depth: depth}
	if parent != nil {
		s.returnType = parent.returnType
		s.tryHandlers = parent.tryHandlers
	}
	return s
}

func (s *evt1Scope) declare(name string, binding evt1ValueBinding) {
	s.values[name] = binding
}

func (s *evt1Scope) addBorrow(binding evt1RetainedBorrow) {
	s.borrows = append(s.borrows, binding)
}

func (s *evt1Scope) lookup(name string) (evt1ValueBinding, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if t, ok := scope.values[name]; ok {
			return t, true
		}
	}
	return evt1ValueBinding{}, false
}

func (s *evt1Scope) setState(name string, state evt1StorageState) bool {
	for scope := s; scope != nil; scope = scope.parent {
		if binding, ok := scope.values[name]; ok {
			binding.state = state
			scope.values[name] = binding
			return true
		}
	}
	return false
}

func (s *evt1Scope) setProvenance(name string, provenance evt1LifetimeProvenance) bool {
	for scope := s; scope != nil; scope = scope.parent {
		if binding, ok := scope.values[name]; ok {
			binding.provenance = provenance
			scope.values[name] = binding
			return true
		}
	}
	return false
}

func evt1CloneScope(scope *evt1Scope) *evt1Scope {
	if scope == nil {
		return nil
	}
	out := newEVT1Scope(evt1CloneScope(scope.parent))
	for name, binding := range scope.values {
		out.values[name] = binding
	}
	out.borrows = append([]evt1RetainedBorrow{}, scope.borrows...)
	return out
}

func evt1JoinStorageState(a, b evt1StorageState) evt1StorageState {
	if a == b {
		return a
	}
	return evt1StorageMaybeMoved
}

func evt1MergeScopeStates(target, left, right *evt1Scope) {
	if target == nil || left == nil || right == nil {
		return
	}
	evt1MergeScopeStates(target.parent, left.parent, right.parent)
	for name, binding := range target.values {
		leftBinding, leftOK := left.values[name]
		rightBinding, rightOK := right.values[name]
		if leftOK && rightOK {
			binding.state = evt1JoinStorageState(leftBinding.state, rightBinding.state)
			target.values[name] = binding
		}
	}
}

func evt1CheckReadableBinding(name string, binding evt1ValueBinding, span Span) error {
	switch binding.state {
	case evt1StorageUninitialized:
		return evt1Diagnostic("CV4514", fmt.Sprintf("use of %s before initialization", name), span)
	case evt1StorageMoved:
		return evt1Diagnostic("CV4502", fmt.Sprintf("use of %s after ownership was moved", name), span)
	case evt1StorageMaybeMoved:
		return evt1Diagnostic("CV4503", fmt.Sprintf("use of %s is invalid because it is moved on some control-flow paths", name), span)
	default:
		return nil
	}
}

func (s *evt1Scope) activeBorrows() []evt1RetainedBorrow {
	var out []evt1RetainedBorrow
	for scope := s; scope != nil; scope = scope.parent {
		out = append(out, scope.borrows...)
	}
	return out
}

func (b evt1ValueBinding) isInstance() bool {
	return b.instanceAutomata != ""
}

func (b evt1ValueBinding) isBatch() bool {
	return b.batchAutomata != ""
}

func (b evt1ValueBinding) isActuatorLocal() bool {
	return b.actuatorName != ""
}

func validateModule(module Module) error {
	_, err := analyzeModule(module)
	return err
}

func analyzeModule(module Module) (*semanticEnv, error) {
	profile, ok := evt1ProfileDefinition(module.Profile)
	if !ok {
		return nil, evt1Diagnostic("CV4001", "module profile must be Core or Vulkan", Span{Line: 1, Column: 1})
	}
	if len(module.Imports) != 0 && !profile.AllowDomainImports {
		return nil, evt1Diagnostic("CV4401", fmt.Sprintf("profile %s does not admit domain imports", profile.Name), Span{Line: 1, Column: 1})
	}
	if len(module.Effects) != 0 && !profile.AllowEffects || len(module.Actuators) != 0 && !profile.AllowActuators {
		return nil, evt1Diagnostic("CV4402", fmt.Sprintf("effect and actuator declarations are not admitted by profile %s", profile.Name), Span{Line: 1, Column: 1})
	}
	env := newSemanticEnv(profile)
	typeNames := map[string]Span{}
	for _, enumDecl := range module.Enums {
		if profile.compilerOwnedType(enumDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", enumDecl.Name), enumDecl.Span)
		}
		if _, exists := env.enums[enumDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4101", fmt.Sprintf("duplicate enum declaration %s", enumDecl.Name), enumDecl.Span)
		}
		if other, exists := typeNames[enumDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4101", fmt.Sprintf("duplicate type declaration %s", enumDecl.Name), other)
		}
		typeNames[enumDecl.Name] = enumDecl.Span
		env.enums[enumDecl.Name] = enumDecl
	}
	for _, structDecl := range module.Structs {
		if profile.compilerOwnedType(structDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", structDecl.Name), structDecl.Span)
		}
		if _, exists := env.structs[structDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4122", fmt.Sprintf("duplicate struct declaration %s", structDecl.Name), structDecl.Span)
		}
		if _, exists := typeNames[structDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4122", fmt.Sprintf("duplicate type declaration %s", structDecl.Name), structDecl.Span)
		}
		typeNames[structDecl.Name] = structDecl.Span
		env.structs[structDecl.Name] = structDecl
	}
	for _, effectDecl := range module.Effects {
		if effectDecl.Name == "dispatch" || effectDecl.Name == "actuate" || effectDecl.Name == "discard" {
			return nil, evt1Diagnostic("CV4268", fmt.Sprintf("%s is a compiler-owned operation name and cannot be redeclared", effectDecl.Name), effectDecl.Span)
		}
		if _, exists := env.effects[effectDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4298", fmt.Sprintf("duplicate effect declaration %s", effectDecl.Name), effectDecl.Span)
		}
		if _, exists := env.automata[effectDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4298", fmt.Sprintf("duplicate declaration %s", effectDecl.Name), effectDecl.Span)
		}
		env.effects[effectDecl.Name] = effectDecl
		env.effectOrder = append(env.effectOrder, effectDecl.Name)
	}
	for _, actuatorDecl := range module.Actuators {
		if profile.compilerOwnedType(actuatorDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", actuatorDecl.Name), actuatorDecl.Span)
		}
		if _, exists := env.actuators[actuatorDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4313", fmt.Sprintf("duplicate actuator declaration %s", actuatorDecl.Name), actuatorDecl.Span)
		}
		if _, exists := typeNames[actuatorDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4313", fmt.Sprintf("duplicate declaration %s", actuatorDecl.Name), actuatorDecl.Span)
		}
		env.actuators[actuatorDecl.Name] = actuatorDecl
	}
	for _, automataDecl := range module.Automata {
		if profile.compilerOwnedType(automataDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", automataDecl.Name), automataDecl.Span)
		}
		if _, exists := env.automata[automataDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4240", fmt.Sprintf("duplicate automata declaration %s", automataDecl.Name), automataDecl.Span)
		}
		if _, exists := typeNames[automataDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4240", fmt.Sprintf("duplicate declaration %s", automataDecl.Name), automataDecl.Span)
		}
		env.automata[automataDecl.Name] = automataDecl
	}
	for _, conceptDecl := range module.Concepts {
		if profile.compilerOwnedType(conceptDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", conceptDecl.Name), conceptDecl.Span)
		}
		if _, exists := env.concepts[conceptDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4146", fmt.Sprintf("duplicate concept declaration %s", conceptDecl.Name), conceptDecl.Span)
		}
		if _, exists := env.automata[conceptDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4240", fmt.Sprintf("duplicate declaration %s", conceptDecl.Name), conceptDecl.Span)
		}
		env.concepts[conceptDecl.Name] = conceptDecl
	}
	for _, templateDecl := range module.Templates {
		if profile.compilerOwnedType(templateDecl.Name) {
			return nil, evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", templateDecl.Name), templateDecl.Span)
		}
		if templateDecl.Name == "dispatch" || templateDecl.Name == "actuate" || templateDecl.Name == "discard" {
			return nil, evt1Diagnostic("CV4268", fmt.Sprintf("%s is a compiler-owned operation name and cannot be redeclared", templateDecl.Name), templateDecl.Span)
		}
		if env.effects[templateDecl.Name].Name != "" {
			return nil, evt1Diagnostic("CV4298", fmt.Sprintf("duplicate declaration %s", templateDecl.Name), templateDecl.Span)
		}
		if _, exists := env.templates[templateDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4168", fmt.Sprintf("duplicate template declaration %s", templateDecl.Name), templateDecl.Span)
		}
		if _, exists := env.automata[templateDecl.Name]; exists {
			return nil, evt1Diagnostic("CV4240", fmt.Sprintf("duplicate declaration %s", templateDecl.Name), templateDecl.Span)
		}
		env.templates[templateDecl.Name] = templateDecl
	}
	for _, decl := range module.ComptimeDecls {
		if _, exists := env.comptimeDecls[decl.Name]; exists {
			return nil, evt1Diagnostic("CV4203", fmt.Sprintf("duplicate comptime declaration %s", decl.Name), decl.Span)
		}
		if len(env.functions[decl.Name]) > 0 || env.templates[decl.Name].Name != "" || env.comptimeFunctions[decl.Name].Name != "" || env.automata[decl.Name].Name != "" || env.effects[decl.Name].Name != "" || env.actuators[decl.Name].Name != "" {
			return nil, evt1Diagnostic("CV4203", fmt.Sprintf("comptime declaration %s conflicts with an existing symbol", decl.Name), decl.Span)
		}
		env.comptimeDecls[decl.Name] = decl
	}
	for _, fn := range module.Functions {
		if fn.Name == "dispatch" || fn.Name == "actuate" || fn.Name == "discard" {
			return nil, evt1Diagnostic("CV4268", fmt.Sprintf("%s is a compiler-owned operation name and cannot be redeclared", fn.Name), fn.Span)
		}
		if env.effects[fn.Name].Name != "" {
			return nil, evt1Diagnostic("CV4298", fmt.Sprintf("duplicate declaration %s", fn.Name), fn.Span)
		}
		if env.automata[fn.Name].Name != "" || env.actuators[fn.Name].Name != "" {
			return nil, evt1Diagnostic("CV4240", fmt.Sprintf("duplicate declaration %s", fn.Name), fn.Span)
		}
		for _, existing := range env.functions[fn.Name] {
			if evt1FunctionParamSignature(existing) == evt1FunctionParamSignature(fn) {
				return nil, evt1Diagnostic("CV4021", fmt.Sprintf("duplicate function declaration %s", fn.Name), fn.Span)
			}
		}
		env.functions[fn.Name] = append(env.functions[fn.Name], fn)
	}
	for _, fn := range module.ComptimeFns {
		if fn.Name == "dispatch" || fn.Name == "actuate" || fn.Name == "discard" {
			return nil, evt1Diagnostic("CV4268", fmt.Sprintf("%s is a compiler-owned operation name and cannot be redeclared", fn.Name), fn.Span)
		}
		if len(env.functions[fn.Name]) > 0 || env.templates[fn.Name].Name != "" || env.comptimeDecls[fn.Name].Name != "" || env.comptimeFunctions[fn.Name].Name != "" || env.automata[fn.Name].Name != "" || env.effects[fn.Name].Name != "" || env.actuators[fn.Name].Name != "" {
			return nil, evt1Diagnostic("CV4214", fmt.Sprintf("comptime function %s conflicts with an existing symbol", fn.Name), fn.Span)
		}
		env.comptimeFunctions[fn.Name] = fn
	}
	for _, structDecl := range module.Structs {
		fields := map[string]Type{}
		if len(structDecl.Fields) == 0 {
			return nil, evt1Diagnostic("CV4123", fmt.Sprintf("empty struct %s is not supported", structDecl.Name), structDecl.Span)
		}
		for _, field := range structDecl.Fields {
			if _, exists := fields[field.Name]; exists {
				return nil, evt1Diagnostic("CV4124", fmt.Sprintf("duplicate field %s.%s", structDecl.Name, field.Name), field.Span)
			}
			if err := validateKnownType(env, field.Type, field.Span, "", false); err != nil {
				return nil, err
			}
			resolved, err := evt1ResolveType(env, nil, field.Type)
			if err != nil {
				return nil, err
			}
			fields[field.Name] = resolved
		}
		env.fieldSets[structDecl.Name] = fields
	}
	for i, enumDecl := range module.Enums {
		seen := map[string]bool{}
		for j, variant := range enumDecl.Variants {
			if seen[variant.Name] {
				return nil, evt1Diagnostic("CV4100", fmt.Sprintf("duplicate variant %s::%s", enumDecl.Name, variant.Name), variant.Span)
			}
			seen[variant.Name] = true
			module.Enums[i].Variants[j].Tag = j
			for _, field := range variant.Payload {
				if err := validateKnownType(env, field.Type, field.Span, "", false); err != nil {
					return nil, err
				}
			}
		}
		env.enums[enumDecl.Name] = module.Enums[i]
	}
	if err := validateValueLayoutCycles(env); err != nil {
		return nil, err
	}
	for _, structDecl := range module.Structs {
		for _, field := range structDecl.Fields {
			if field.Type.isReference() && !structDecl.Ref {
				return nil, evt1Diagnostic("CV4525", fmt.Sprintf("unrestricted struct %s cannot contain reference field %s; declare a ref struct", structDecl.Name, field.Name), field.Span)
			}
			if embedded, ok := env.structs[field.Type.valueType().Name]; ok && embedded.Ref && !structDecl.Ref {
				return nil, evt1Diagnostic("CV4525", fmt.Sprintf("unrestricted struct %s cannot contain lifetime-bound ref struct field %s", structDecl.Name, field.Name), field.Span)
			}
			if !field.Type.isBorrowLike() && evt1IsImmovableValueType(env, field.Type) {
				return nil, evt1Diagnostic("CV4138", fmt.Sprintf("struct %s cannot embed immovable field %s", structDecl.Name, field.Type.String()), field.Span)
			}
		}
	}
	for _, enumDecl := range module.Enums {
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if field.Type.isReference() || evt1IsRefStructType(env, field.Type) {
					return nil, evt1Diagnostic("CV4525", fmt.Sprintf("enum payload %s::%s cannot contain lifetime-bound type %s", enumDecl.Name, variant.Name, field.Type.String()), field.Span)
				}
				if evt1IsImmovableValueType(env, field.Type) {
					return nil, evt1Diagnostic("CV4139", fmt.Sprintf("enum payload %s::%s cannot contain immovable type %s", enumDecl.Name, variant.Name, field.Type.String()), field.Span)
				}
				if !evt1TypeCopyable(env, field.Type) {
					return nil, evt1Diagnostic("CV4133", fmt.Sprintf("enum payload %s::%s cannot copy non-copyable type %s", enumDecl.Name, variant.Name, field.Type.String()), field.Span)
				}
			}
		}
	}
	for i, effectDecl := range module.Effects {
		for j, param := range effectDecl.Params {
			if err := validateKnownType(env, param.Type, param.Span, "", false); err != nil {
				return nil, err
			}
			resolved, err := evt1ResolveType(env, nil, param.Type.valueType())
			if err != nil {
				return nil, err
			}
			if err := validateEffectPayloadType(env, resolved, param.Span, effectDecl.Name+"."+param.Name); err != nil {
				return nil, err
			}
			module.Effects[i].Params[j].Type = resolved
		}
		env.effects[effectDecl.Name] = module.Effects[i]
	}
	if err := evt1ValidateAutomataDecls(env, module); err != nil {
		return nil, err
	}
	if err := evt1ValidateActuatorDecls(env, module); err != nil {
		return nil, err
	}
	for _, conceptDecl := range module.Concepts {
		for _, req := range conceptDecl.Requirements {
			switch r := req.(type) {
			case *OperationRequirement:
				if err := validateKnownType(env, r.ReturnType, r.Span, conceptDecl.TypeParam, false); err != nil {
					return nil, err
				}
				for _, param := range r.Params {
					if err := validateKnownType(env, param.Type, param.Span, conceptDecl.TypeParam, false); err != nil {
						return nil, err
					}
				}
			case *PrerequisiteRequirement:
				if _, ok := env.concepts[r.ConceptName]; !ok {
					return nil, evt1Diagnostic("CV4152", fmt.Sprintf("unknown prerequisite concept %s", r.ConceptName), r.Span)
				}
				if r.TypeArg.Kind != TypeConceptParam || r.TypeArg.Name != conceptDecl.TypeParam {
					return nil, evt1Diagnostic("CV4152", fmt.Sprintf("prerequisite %s must use the concept parameter %s", r.ConceptName, conceptDecl.TypeParam), r.Span)
				}
			case *CompilerAnalysisRequirement:
				analysis, ok := evt1SemanticAnalysisRegistry[r.Analysis]
				if !ok {
					return nil, evt1Diagnostic("CV4526", fmt.Sprintf("unknown compiler analysis %s", r.Analysis), r.Span)
				}
				if len(r.TypeArgs) != analysis.TypeArity || len(r.SubjectArgs) != analysis.SubjectArity {
					return nil, evt1Diagnostic("CV4526", fmt.Sprintf("compiler analysis %s requires %d type argument(s) and %d semantic subject(s)", r.Analysis, analysis.TypeArity, analysis.SubjectArity), r.Span)
				}
				for _, arg := range r.TypeArgs {
					if err := validateKnownType(env, arg, r.Span, conceptDecl.TypeParam, false); err != nil {
						return nil, err
					}
				}
				if len(r.SubjectArgs) > 0 {
					if r.Analysis == "Outlives" && (r.SubjectArgs[0].Name == "result" || r.SubjectArgs[1].Name != "result") {
						return nil, evt1Diagnostic("CV4532", "Outlives requires a source parameter followed by result", r.Span)
					}
					if analysis.CheckSubjects == nil {
						return nil, evt1Diagnostic("CV4533", fmt.Sprintf("compiler analysis %s does not support semantic subjects", r.Analysis), r.Span)
					}
					if _, err := evt1FindRelationalRequirementOperation(conceptDecl, r.SubjectArgs, r.Span); err != nil {
						return nil, err
					}
				}
			default:
				return nil, evt1Diagnostic("CV4147", "unsupported concept requirement", req.requirementSpan())
			}
		}
	}
	if err := validateConceptCycles(env); err != nil {
		return nil, err
	}
	for _, templateDecl := range module.Templates {
		if err := validateTemplateSignature(env, templateDecl); err != nil {
			return nil, err
		}
		info, err := buildTemplateInfo(env, templateDecl)
		if err != nil {
			return nil, err
		}
		env.templateInfos[templateDecl.Name] = info
	}
	for _, fn := range module.Functions {
		if err := validateFunctionSignature(env, fn); err != nil {
			return nil, err
		}
	}
	evt1DeriveResultProvenanceSummaries(env, module.Functions)
	for _, fn := range module.ComptimeFns {
		if err := validateFunctionSignature(env, fn); err != nil {
			return nil, err
		}
		if fn.Body == nil {
			return nil, evt1Diagnostic("CV4215", fmt.Sprintf("comptime function %s requires a body", fn.Name), fn.Span)
		}
		if fn.ReturnType.Name != "void" && !evt1IsComptimeType(env, fn.ReturnType) {
			return nil, evt1Diagnostic("CV4216", fmt.Sprintf("comptime return type %s is not supported", fn.ReturnType.String()), fn.ReturnType.Span)
		}
		for _, param := range fn.Params {
			if !evt1IsComptimeType(env, param.Type) {
				return nil, evt1Diagnostic("CV4216", fmt.Sprintf("comptime parameter type %s is not supported", param.Type.String()), param.Span)
			}
		}
	}
	for _, decl := range module.ComptimeDecls {
		if err := validateKnownType(env, decl.Type, decl.Span, "", false); err != nil {
			return nil, err
		}
		if !evt1IsComptimeType(env, decl.Type) {
			return nil, evt1Diagnostic("CV4216", fmt.Sprintf("comptime declaration type %s is not supported", decl.Type.String()), decl.Span)
		}
	}
	for _, templateDecl := range module.Templates {
		scope := evt1ModuleScope(env)
		resolvedReturn, err := evt1ResolveType(env, nil, templateDecl.ReturnType)
		if err != nil {
			return nil, err
		}
		scope.returnType = resolvedReturn
		for paramIndex, param := range templateDecl.Params {
			resolvedParam, err := evt1ResolveType(env, nil, param.Type)
			if err != nil {
				return nil, err
			}
			scope.declare(param.Name, evt1ValueBinding{
				t:          resolvedParam,
				mutable:    !param.Type.Const,
				state:      evt1StorageInitialized,
				provenance: evt1InitialParameterProvenance(env, resolvedParam, paramIndex, scope.depth),
			})
		}
		if err := validateBlock(env, scope, resolvedReturn, *templateDecl.Body, env.templateInfos[templateDecl.Name], false); err != nil {
			return nil, err
		}
	}
	for _, fn := range module.Functions {
		if fn.Body == nil {
			continue
		}
		scope := evt1ModuleScope(env)
		resolvedReturn, err := evt1ResolveType(env, nil, fn.ReturnType)
		if err != nil {
			return nil, err
		}
		scope.returnType = resolvedReturn
		for paramIndex, param := range fn.Params {
			resolvedParam, err := evt1ResolveType(env, nil, param.Type)
			if err != nil {
				return nil, err
			}
			scope.declare(param.Name, evt1ValueBinding{
				t:          resolvedParam,
				mutable:    !param.Type.Const,
				state:      evt1StorageInitialized,
				provenance: evt1InitialParameterProvenance(env, resolvedParam, paramIndex, scope.depth),
			})
		}
		collectEscapedArmBindings(fn.Body, env)
		if err := validateBlock(env, scope, resolvedReturn, *fn.Body, nil, false); err != nil {
			return nil, err
		}
	}
	for _, fn := range module.ComptimeFns {
		scope := evt1ModuleScope(env)
		resolvedReturn, err := evt1ResolveType(env, nil, fn.ReturnType)
		if err != nil {
			return nil, err
		}
		scope.returnType = resolvedReturn
		for paramIndex, param := range fn.Params {
			resolvedParam, err := evt1ResolveType(env, nil, param.Type)
			if err != nil {
				return nil, err
			}
			scope.declare(param.Name, evt1ValueBinding{
				t:          resolvedParam,
				mutable:    !param.Type.Const,
				state:      evt1StorageInitialized,
				comptime:   true,
				provenance: evt1InitialParameterProvenance(env, resolvedParam, paramIndex, scope.depth),
			})
		}
		collectEscapedArmBindings(fn.Body, env)
		if err := validateBlock(env, scope, resolvedReturn, *fn.Body, nil, true); err != nil {
			return nil, err
		}
	}
	if err := validateComptimeFunctionCycles(module, env); err != nil {
		return nil, err
	}
	for _, assertion := range module.Assertions {
		if _, ok := env.concepts[assertion.ConceptName]; !ok {
			return nil, evt1Diagnostic("CV4151", fmt.Sprintf("unknown concept %s", assertion.ConceptName), assertion.Span)
		}
		if err := validateKnownType(env, assertion.ConcreteType, assertion.Span, "", false); err != nil {
			return nil, err
		}
		if err := checkConceptSatisfaction(env, assertion.ConceptName, assertion.ConcreteType, nil, assertion.Span); err != nil {
			return nil, err
		}
	}
	if err := evt1EvaluateModuleComptime(env, module); err != nil {
		return nil, err
	}
	return env, nil
}

func evt1ModuleScope(env *semanticEnv) *evt1Scope {
	scope := newEVT1Scope(nil)
	for _, decl := range env.comptimeDecls {
		resolved := evt1CanonicalType(env, decl.Type)
		if next, err := evt1ResolveType(env, nil, resolved); err == nil {
			resolved = next
		}
		scope.declare(decl.Name, evt1ValueBinding{
			t:        resolved,
			mutable:  false,
			comptime: true,
		})
	}
	return scope
}

func validateComptimeFunctionCycles(module Module, env *semanticEnv) error {
	graph := map[string][]string{}
	for _, fn := range module.ComptimeFns {
		if fn.Body == nil {
			continue
		}
		seen := map[string]bool{}
		for _, callee := range evt1CollectComptimeCallsFromBlock(*fn.Body, env) {
			if !seen[callee] {
				graph[fn.Name] = append(graph[fn.Name], callee)
				seen[callee] = true
			}
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var dfs func(name string, path []string) error
	dfs = func(name string, path []string) error {
		if visiting[name] {
			cycle := append(path, name)
			return evt1Diagnostic("CV4217", "comptime recursion is not allowed: "+strings.Join(cycle, " -> "), env.comptimeFunctions[name].Span)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		visited[name] = true
		for _, callee := range graph[name] {
			if err := dfs(callee, append(path, name)); err != nil {
				return err
			}
		}
		visiting[name] = false
		return nil
	}
	for _, fn := range module.ComptimeFns {
		if err := dfs(fn.Name, nil); err != nil {
			return err
		}
	}
	return nil
}

func evt1CollectComptimeCallsFromBlock(block Block, env *semanticEnv) []string {
	var out []string
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Value, env)...)
		case *InstanceDecl:
			continue
		case *AssignStmt:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Target, env)...)
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Value, env)...)
		case *ReturnStmt:
			if s.Value != nil {
				out = append(out, evt1CollectComptimeCallsFromExpr(s.Value, env)...)
			}
		case *ExprStmt:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Value, env)...)
		case *StaticAssertStmt:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Condition, env)...)
			if s.Message != nil {
				out = append(out, evt1CollectComptimeCallsFromExpr(s.Message, env)...)
			}
		case *MatchStmt:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Subject, env)...)
			for _, arm := range s.Arms {
				out = append(out, evt1CollectComptimeCallsFromBlock(arm.Block, env)...)
			}
		case *WhileStmt:
			out = append(out, evt1CollectComptimeCallsFromExpr(s.Condition, env)...)
			if s.Bound != nil {
				out = append(out, evt1CollectComptimeCallsFromExpr(s.Bound, env)...)
			}
			out = append(out, evt1CollectComptimeCallsFromBlock(s.Body, env)...)
		case *Block:
			out = append(out, evt1CollectComptimeCallsFromBlock(*s, env)...)
		}
	}
	return out
}

func evt1CollectComptimeCallsFromExpr(expr Expr, env *semanticEnv) []string {
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1CollectComptimeCallsFromExpr(e.Value, env)
	case *UnaryExpr:
		return evt1CollectComptimeCallsFromExpr(e.Value, env)
	case *FieldExpr:
		return evt1CollectComptimeCallsFromExpr(e.Receiver, env)
	case *IndexExpr:
		out := evt1CollectComptimeCallsFromExpr(e.Base, env)
		for _, index := range evt1StorageIndices(e) {
			out = append(out, evt1CollectComptimeCallsFromExpr(index, env)...)
		}
		return out
	case *BinaryExpr:
		return append(evt1CollectComptimeCallsFromExpr(e.Left, env), evt1CollectComptimeCallsFromExpr(e.Right, env)...)
	case *CallExpr:
		var out []string
		if _, ok := env.comptimeFunctions[e.Callee]; ok {
			out = append(out, e.Callee)
		}
		for _, arg := range e.Args {
			out = append(out, evt1CollectComptimeCallsFromExpr(arg, env)...)
		}
		return out
	case *DispatchExpr:
		return evt1CollectComptimeCallsFromExpr(e.Signal, env)
	case *TemplateCallExpr:
		var out []string
		for _, arg := range e.Args {
			out = append(out, evt1CollectComptimeCallsFromExpr(arg, env)...)
		}
		return out
	case *ConstructExpr:
		var out []string
		for _, arg := range e.Args {
			out = append(out, evt1CollectComptimeCallsFromExpr(arg, env)...)
		}
		return out
	case *StructConstructExpr:
		var out []string
		for _, arg := range e.Args {
			out = append(out, evt1CollectComptimeCallsFromExpr(arg, env)...)
		}
		return out
	case *WithExpr:
		out := evt1CollectComptimeCallsFromExpr(e.Base, env)
		for _, update := range e.Updates {
			out = append(out, evt1CollectComptimeCallsFromExpr(update.Value, env)...)
		}
		return out
	case *ArrayLiteralExpr:
		var out []string
		for _, arg := range e.Elements {
			out = append(out, evt1CollectComptimeCallsFromExpr(arg, env)...)
		}
		return out
	case *MatchExpr:
		out := evt1CollectComptimeCallsFromExpr(e.Subject, env)
		for _, arm := range e.Arms {
			out = append(out, evt1CollectComptimeCallsFromExpr(arm.Value, env)...)
		}
		return out
	case *IfExpr:
		out := evt1CollectComptimeCallsFromExpr(e.Condition, env)
		out = append(out, evt1CollectComptimeCallsFromExpr(e.Then, env)...)
		out = append(out, evt1CollectComptimeCallsFromExpr(e.Else, env)...)
		return out
	default:
		return nil
	}
}

func validateFunctionSignature(env *semanticEnv, fn FunctionDecl) error {
	if err := validateKnownType(env, fn.ReturnType, fn.Span, "", false); err != nil {
		return err
	}
	if err := validateByValueBoundary(env, fn.ReturnType, fn.Span, "return"); err != nil {
		return err
	}
	for _, param := range fn.Params {
		if err := validateKnownType(env, param.Type, param.Span, "", false); err != nil {
			return err
		}
		if err := validateByValueBoundary(env, param.Type, param.Span, "parameter"); err != nil {
			return err
		}
	}
	return nil
}

func validateTemplateSignature(env *semanticEnv, templateDecl TemplateDecl) error {
	if _, ok := env.concepts[templateDecl.Constraint.ConceptName]; !ok {
		return evt1Diagnostic("CV4169", fmt.Sprintf("unknown concept %s in template constraint", templateDecl.Constraint.ConceptName), templateDecl.Constraint.Span)
	}
	constraintConcept := env.concepts[templateDecl.Constraint.ConceptName]
	if templateDecl.Constraint.TypeArg.Kind != TypeConceptParam || templateDecl.Constraint.TypeArg.Name != templateDecl.TypeParam {
		return evt1Diagnostic("CV4170", fmt.Sprintf("template constraint %s must apply to template parameter %s", templateDecl.Constraint.ConceptName, templateDecl.TypeParam), templateDecl.Constraint.Span)
	}
	if err := validateKnownType(env, templateDecl.ReturnType, templateDecl.ReturnType.Span, templateDecl.TypeParam, false); err != nil {
		return err
	}
	if err := validateTemplateByValueBoundary(env, templateDecl.ReturnType, templateDecl.Span, "return", templateDecl.TypeParam); err != nil {
		return err
	}
	if constraintConcept.TypeParam == "" {
		return evt1Diagnostic("CV4171", fmt.Sprintf("template constraint %s must be a named one-parameter concept", templateDecl.Constraint.ConceptName), templateDecl.Constraint.Span)
	}
	for _, param := range templateDecl.Params {
		if err := validateKnownType(env, param.Type, param.Span, templateDecl.TypeParam, false); err != nil {
			return err
		}
		if err := validateTemplateByValueBoundary(env, param.Type, param.Span, "parameter", templateDecl.TypeParam); err != nil {
			return err
		}
	}
	return nil
}

func validateByValueBoundary(env *semanticEnv, t Type, span Span, context string) error {
	if t.isBorrowLike() {
		return nil
	}
	if evt1IsImmovableValueType(env, t) {
		switch context {
		case "parameter":
			return evt1Diagnostic("CV4136", fmt.Sprintf("immovable type %s cannot be passed by value", t.String()), span)
		case "return":
			return evt1Diagnostic("CV4137", fmt.Sprintf("immovable type %s cannot be returned by value", t.String()), span)
		}
	}
	return nil
}

func validateTemplateByValueBoundary(env *semanticEnv, t Type, span Span, context, typeParam string) error {
	if evt1TypeDependsOnParam(t, typeParam) {
		return nil
	}
	return validateByValueBoundary(env, t, span, context)
}

func validateEffectPayloadType(env *semanticEnv, t Type, span Span, label string) error {
	if t.PointerTo != nil || t.ArrayElem != nil || t.Ownership != "" || t.Const || t.Imported || t.Unsafe || len(t.TypeArgs) > 0 {
		return evt1Diagnostic("CV4303", fmt.Sprintf("effect payload %s must use a fixed immutable value type, got %s", label, t.String()), span)
	}
	switch t.Kind {
	case TypeBuiltin:
		switch t.Name {
		case "int", "bool", "uint64":
			return nil
		default:
			return evt1Diagnostic("CV4303", fmt.Sprintf("effect payload %s must use a fixed immutable value type, got %s", label, t.String()), span)
		}
	case TypeEnum:
		enumDecl := env.enums[t.Name]
		for _, variant := range enumDecl.Variants {
			if len(variant.Payload) > 0 {
				return evt1Diagnostic("CV4303", fmt.Sprintf("effect payload %s enum %s must use only nullary variants", label, t.Name), span)
			}
		}
		return nil
	case TypeStruct:
		structDecl := env.structs[t.Name]
		for _, field := range structDecl.Fields {
			fieldType, err := evt1ResolveType(env, nil, field.Type.valueType())
			if err != nil {
				return err
			}
			if err := validateEffectPayloadType(env, fieldType, field.Span, label+"."+field.Name); err != nil {
				return err
			}
		}
		return nil
	default:
		return evt1Diagnostic("CV4303", fmt.Sprintf("effect payload %s must use a fixed immutable value type, got %s", label, t.String()), span)
	}
}

func collectEscapedArmBindings(block *Block, env *semanticEnv) {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *MatchStmt:
			for _, arm := range s.Arms {
				for _, binding := range arm.Pattern.Bindings {
					env.escapedArmBinding[binding] = arm.Pattern.Span
				}
				collectEscapedArmBindings(&arm.Block, env)
			}
		case *Block:
			collectEscapedArmBindings(s, env)
		case *IfStmt:
			collectEscapedArmBindings(&s.Then, env)
			if s.Else != nil {
				collectEscapedArmBindings(s.Else, env)
			}
		}
	}
}

func validateBlock(env *semanticEnv, scope *evt1Scope, returnType Type, block Block, templateInfo *evt1TemplateInfo, inComptimeFn bool) error {
	local := newEVT1Scope(scope)
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			typeParam := ""
			if templateInfo != nil {
				typeParam = templateInfo.Decl.TypeParam
			}
			if err := validateKnownType(env, s.Type, s.Span, typeParam, false); err != nil {
				return err
			}
			resolvedType, err := evt1ResolveType(env, local, s.Type)
			if err != nil {
				return err
			}
			s.Type = resolvedType
			if evt1StorageHasRuntimeShape(resolvedType) && !inComptimeFn && !s.Comptime {
				code := "CV4558"
				family := "array"
				if resolvedType.StorageKind == StorageNDArray {
					code = "CV4559"
					family = "ndarray"
				}
				return evt1Diagnostic(code, fmt.Sprintf("runtime %s %s requires explicit storage; shape does not allocate", family, resolvedType.String()), s.Span)
			}
			if s.Value == nil {
				return evt1Diagnostic("CV4560", fmt.Sprintf("fixed storage local %s requires an initializer", s.Name), s.Span)
			}
			valueType, err := validateExprAgainstExpected(env, local, s.Value, resolvedType, templateInfo, inComptimeFn || s.Comptime)
			if err != nil {
				return err
			}
			if !evt1TypesCompatible(env, resolvedType, valueType, typeParam) {
				return evt1Diagnostic("CV4106", fmt.Sprintf("constructor or initializer for %s expected %s but got %s", s.Name, resolvedType.String(), valueType.String()), s.Value.exprSpan())
			}
			if !s.Comptime && valueType.isOwned() && !evt1CanTransferInitialize(env, resolvedType, s.Value) && !evt1TypeDependsOnParam(valueType, typeParam) {
				return evt1Diagnostic("CV4501", fmt.Sprintf("copy of non-copyable type %s requires move", valueType.String()), s.Value.exprSpan())
			}
			if !s.Comptime && !evt1CanTransferInitialize(env, resolvedType, s.Value) && !evt1TypeCopyable(env, resolvedType) && !evt1TypeDependsOnParam(resolvedType, typeParam) {
				if evt1IsImmovableValueType(env, resolvedType) {
					return evt1Diagnostic("CV4134", fmt.Sprintf("immovable value %s must be constructed directly in final storage", resolvedType.String()), s.Span)
				}
				return evt1Diagnostic("CV4133", fmt.Sprintf("copy of non-copyable type %s is not allowed", resolvedType.String()), s.Span)
			}
			if s.Comptime {
				if !evt1IsComptimeType(env, resolvedType) {
					return evt1Diagnostic("CV4216", fmt.Sprintf("comptime declaration type %s is not supported", resolvedType.String()), s.Span)
				}
				value, err := evt1EvalExprTyped(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(local, env), s.Value, &resolvedType)
				if err != nil {
					return err
				}
				local.declare(s.Name, evt1ValueBinding{t: resolvedType, mutable: false, comptime: true, hasValue: true, value: value})
				continue
			}
			provenance := evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: local.depth, Scoped: resolvedType.Scoped}
			if resolvedType.isReference() || evt1IsRefStructType(env, resolvedType) {
				provenance = evt1ExprProvenance(env, local, s.Value)
				provenance.Scoped = provenance.Scoped || resolvedType.Scoped
			}
			local.declare(s.Name, evt1ValueBinding{t: resolvedType, mutable: !s.Const, state: evt1StorageInitialized, comptime: inComptimeFn, provenance: provenance})
		case *EffectsDecl:
			if inComptimeFn {
				return evt1Diagnostic("CV4304", fmt.Sprintf("effects batch %s cannot be declared in comptime code", s.Name), s.Span)
			}
			info, ok := env.automataInfo[s.AutomataName]
			if !ok {
				return evt1Diagnostic("CV4300", fmt.Sprintf("effects declaration requires an automata name, got %s", s.AutomataName), s.Span)
			}
			local.declare(s.Name, evt1ValueBinding{
				mutable:       true,
				batchAutomata: info.Decl.Name,
			})
		case *ActuatorLocalDecl:
			if inComptimeFn {
				return evt1Diagnostic("CV4319", fmt.Sprintf("actuator local %s cannot be declared in comptime code", s.Name), s.Span)
			}
			info, ok := env.actuatorInfo[s.ActuatorName]
			if !ok {
				return evt1Diagnostic("CV4319", fmt.Sprintf("unknown actuator %s", s.ActuatorName), s.Span)
			}
			mechanismType, err := validateExpr(env, local, s.Mechanism, templateInfo, false)
			if err != nil {
				return err
			}
			if err := validateCallArgument(env, local, info.MechanismType, s.Mechanism, mechanismType, templateInfo); err != nil {
				return err
			}
			local.declare(s.Name, evt1ValueBinding{
				mutable:      false,
				actuatorName: info.Decl.Name,
			})
		case *InstanceDecl:
			if inComptimeFn {
				return evt1Diagnostic("CV4271", fmt.Sprintf("instance %s cannot be declared in comptime code", s.Name), s.Span)
			}
			info, ok := env.automataInfo[s.AutomataName]
			if !ok {
				return evt1Diagnostic("CV4270", fmt.Sprintf("instance declaration requires an automata name, got %s", s.AutomataName), s.Span)
			}
			if info.Decl.Context != nil {
				if s.Context == nil {
					return evt1Diagnostic("CV4283", fmt.Sprintf("instance %s of automata %s requires a context argument", s.Name, s.AutomataName), s.Span)
				}
				contextType := info.Decl.Context.Type
				argType, err := validateExpr(env, local, s.Context, templateInfo, false)
				if err != nil {
					return err
				}
				paramType := contextType
				paramType.Ownership = "borrow"
				paramType.Const = true
				if err := validateCallArgument(env, local, paramType, s.Context, argType, templateInfo); err != nil {
					return err
				}
				lvalue, err := validateAssignable(env, local, s.Context, templateInfo)
				if err != nil {
					return evt1Diagnostic("CV4285", fmt.Sprintf("context binding for instance %s requires an assignable access path", s.Name), s.Context.exprSpan())
				}
				local.addBorrow(evt1RetainedBorrow{
					InstanceName: s.Name,
					AutomataName: s.AutomataName,
					ContextName:  info.Decl.Context.Name,
					Path:         lvalue.path,
					Type:         contextType,
					Span:         s.Span,
				})
			} else if s.Context != nil {
				return evt1Diagnostic("CV4284", fmt.Sprintf("contextless automata %s does not accept a context argument", s.AutomataName), s.Context.exprSpan())
			}
			local.declare(s.Name, evt1ValueBinding{
				mutable:          true,
				instanceAutomata: info.Decl.Name,
			})
		case *ActuationDecl:
			info, ok := env.actuatorInfo[s.ActuatorName]
			if !ok {
				return evt1Diagnostic("CV4320", fmt.Sprintf("unknown actuator %s in actuation", s.ActuatorName), s.Span)
			}
			batchBinding, ok := local.lookup(s.BatchName)
			if !ok || !batchBinding.isBatch() {
				return evt1Diagnostic("CV4321", fmt.Sprintf("actuate requires a local effects batch, but %s is not one", s.BatchName), s.Span)
			}
			if batchBinding.batchAutomata != info.Automata.Decl.Name {
				return evt1Diagnostic("CV4321", fmt.Sprintf("actuate requires a batch for automata %s, but %s belongs to %s", info.Automata.Decl.Name, s.BatchName, batchBinding.batchAutomata), s.Span)
			}
			executorBinding, ok := local.lookup(s.ExecutorName)
			if !ok || !executorBinding.isActuatorLocal() {
				return evt1Diagnostic("CV4321", fmt.Sprintf("actuate requires a local actuator executor, but %s is not one", s.ExecutorName), s.Span)
			}
			if executorBinding.actuatorName != info.Decl.Name {
				return evt1Diagnostic("CV4321", fmt.Sprintf("actuation %s expects an executor for actuator %s, but %s belongs to %s", s.Name, info.Decl.Name, s.ExecutorName, executorBinding.actuatorName), s.Span)
			}
			local.declare(s.Name, evt1ValueBinding{
				t:       Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: s.Span},
				mutable: false,
			})
		case *AssignStmt:
			target, err := validateAssignable(env, local, s.Target, templateInfo)
			if err != nil {
				return err
			}
			if !target.mutable {
				if target.readOnlyReason == "record" {
					return evt1Diagnostic("CV4142", "record fields are read-only after construction", s.Target.exprSpan())
				}
				if target.readOnlyReason == "ref_const" {
					return evt1Diagnostic("CV4513", "mutation through ref const is not allowed", s.Target.exprSpan())
				}
				return evt1Diagnostic("CV4128", "mutation through a const access path is not allowed", s.Target.exprSpan())
			}
			if borrow, ok := evt1FindOverlappingBorrow(local.activeBorrows(), target.path); ok {
				return evt1Diagnostic("CV4291", fmt.Sprintf("assignment to %s overlaps retained immutable automata context for instance %s of %s", exprLabel(s.Target), borrow.InstanceName, borrow.AutomataName), s.Target.exprSpan())
			}
			valueType, err := validateExprAgainstExpected(env, local, s.Value, target.t, templateInfo, inComptimeFn)
			if err != nil {
				return err
			}
			typeParam := ""
			if templateInfo != nil {
				typeParam = templateInfo.Decl.TypeParam
			}
			if !evt1TypesCompatible(env, target.t, valueType, typeParam) {
				return evt1Diagnostic("CV4107", fmt.Sprintf("assignment to %s expected %s but got %s", exprLabel(s.Target), target.t.String(), valueType.String()), s.Value.exprSpan())
			}
			if name, ok := s.Target.(*NameExpr); ok && (target.t.isReference() || evt1IsRefStructType(env, target.t)) {
				binding, _ := local.lookup(name.Name)
				sourceProvenance := evt1ExprProvenance(env, local, s.Value)
				if evt1LifetimeShorterThan(sourceProvenance, binding.provenance) {
					if sourceProvenance.Kind == evt1ProvenanceUnknown && evt1IsCallResultExpr(s.Value) {
						return evt1Diagnostic("CV4529", fmt.Sprintf("call result assigned to %s has unknown lifetime provenance", name.Name), s.Value.exprSpan())
					}
					if evt1IsCallResultExpr(s.Value) {
						return evt1Diagnostic("CV4530", fmt.Sprintf("call result assigned to %s would outlive its source", name.Name), s.Value.exprSpan())
					}
					return evt1Diagnostic("CV4523", fmt.Sprintf("value assigned to %s does not outlive its destination", name.Name), s.Value.exprSpan())
				}
				if binding.t.Scoped && !sourceProvenance.Scoped {
					sourceProvenance.Scoped = true
				}
				local.setProvenance(name.Name, sourceProvenance)
			}
			if !evt1TypeCopyable(env, target.t) && !evt1TypeDependsOnParam(target.t, typeParam) {
				if evt1IsImmovableValueType(env, target.t) && target.wholeValue {
					return evt1Diagnostic("CV4135", fmt.Sprintf("immovable value %s cannot be assigned as a whole", target.t.String()), s.Span)
				}
				name, isName := s.Target.(*NameExpr)
				binding := evt1ValueBinding{}
				if isName {
					binding, _ = local.lookup(name.Name)
				}
				canReplaceOwned := isName && binding.state == evt1StorageInitialized && target.t.isOwned() && evt1CanTransferInitialize(env, target.t, s.Value)
				if !isName || (binding.state != evt1StorageMoved && !canReplaceOwned) || !evt1CanTransferInitialize(env, target.t, s.Value) {
					if target.t.isOwned() {
						return evt1Diagnostic("CV4501", fmt.Sprintf("assignment copies non-copyable type %s; use move from an initialized owner", target.t.String()), s.Span)
					}
					return evt1Diagnostic("CV4133", fmt.Sprintf("assignment copies non-copyable type %s", target.t.String()), s.Span)
				}
				local.setState(name.Name, evt1StorageInitialized)
			}
		case *ReturnStmt:
			if s.Value == nil {
				if returnType.Name != "void" {
					return evt1Diagnostic("CV4022", fmt.Sprintf("return requires a %s value", returnType.String()), s.Span)
				}
				continue
			}
			valueType, err := validateExprAgainstExpected(env, local, s.Value, returnType, templateInfo, inComptimeFn)
			if err != nil {
				return err
			}
			typeParam := ""
			if templateInfo != nil {
				typeParam = templateInfo.Decl.TypeParam
			}
			if !evt1TypesCompatible(env, returnType, valueType, typeParam) {
				return evt1Diagnostic("CV4116", fmt.Sprintf("expression result type mismatch: expected %s but got %s", returnType.String(), valueType.String()), s.Value.exprSpan())
			}
			if returnType.isBorrowLike() {
				provenance := evt1ExprProvenance(env, local, s.Value)
				if provenance.Scoped {
					if evt1IsCallResultExpr(s.Value) {
						return evt1Diagnostic("CV4531", "call result derived from scoped provenance cannot escape through return", s.Value.exprSpan())
					}
					return evt1Diagnostic("CV4522", "scoped reference cannot escape through return", s.Value.exprSpan())
				}
				if provenance.Kind != evt1ProvenanceParameter && provenance.Kind != evt1ProvenanceStatic {
					return evt1Diagnostic("CV4511", "reference return would escape local storage", s.Value.exprSpan())
				}
			}
			if evt1IsRefStructType(env, returnType) {
				provenance := evt1ExprProvenance(env, local, s.Value)
				if provenance.Scoped {
					if evt1IsCallResultExpr(s.Value) {
						return evt1Diagnostic("CV4531", "call result derived from scoped provenance cannot escape through return", s.Value.exprSpan())
					}
					return evt1Diagnostic("CV4522", "scoped lifetime-bound value cannot escape through return", s.Value.exprSpan())
				}
				if provenance.Kind != evt1ProvenanceParameter && provenance.Kind != evt1ProvenanceStatic {
					return evt1Diagnostic("CV4521", "ref struct return would escape referenced local storage", s.Value.exprSpan())
				}
			}
			if !evt1TypeCopyable(env, returnType) && !evt1CanTransferInitialize(env, returnType, s.Value) {
				return evt1Diagnostic("CV4506", fmt.Sprintf("return of non-copyable type %s requires an explicit move", returnType.String()), s.Value.exprSpan())
			}
		case *ExprStmt:
			if _, err := validateExpr(env, local, s.Value, templateInfo, inComptimeFn); err != nil {
				return err
			}
		case *AssertStmt:
			conditionType, err := validateExpr(env, local, s.Condition, templateInfo, inComptimeFn)
			if err != nil {
				return err
			}
			if conditionType.Name != "bool" {
				return evt1Diagnostic("CV4546", "assert requires a bool condition", s.Condition.exprSpan())
			}
			if s.Reason != nil {
				reasonType, err := validateExpr(env, local, s.Reason, templateInfo, inComptimeFn)
				if err != nil {
					return err
				}
				if reasonType.Name != "string" {
					return evt1Diagnostic("CV4546", "assert reason must be string", s.Reason.exprSpan())
				}
			}
		case *TryStmt:
			handlers := map[string]Type{}
			combinedHandlers := map[string]Type{}
			for key, t := range local.tryHandlers {
				combinedHandlers[key] = t
			}
			for _, arm := range s.Except {
				if err := validateKnownType(env, arm.ErrorType, arm.Span, "", false); err != nil {
					return evt1Diagnostic("CV4550", fmt.Sprintf("unknown except error type %s", arm.ErrorType.String()), arm.Span)
				}
				resolved := evt1CanonicalType(env, arm.ErrorType)
				key := evt1TypeIdentity(resolved)
				if _, exists := handlers[key]; exists {
					return evt1Diagnostic("CV4547", fmt.Sprintf("duplicate except arm for %s", resolved.String()), arm.Span)
				}
				handlers[key] = resolved
				combinedHandlers[key] = resolved
			}
			tryScope := evt1CloneScope(local)
			tryScope.tryHandlers = combinedHandlers
			if err := validateBlock(env, tryScope, returnType, s.Body, templateInfo, inComptimeFn); err != nil {
				return err
			}
			for _, arm := range s.Except {
				armScope := evt1CloneScope(local)
				armScope.declare(arm.Binding, evt1ValueBinding{t: evt1CanonicalType(env, arm.ErrorType), mutable: false, state: evt1StorageInitialized})
				if err := validateBlock(env, armScope, returnType, arm.Body, templateInfo, inComptimeFn); err != nil {
					return err
				}
			}
		case *StaticAssertStmt:
			if _, err := validateExpr(env, local, s.Condition, templateInfo, true); err != nil {
				return err
			}
			if s.Message != nil {
				if _, err := validateExpr(env, local, s.Message, templateInfo, true); err != nil {
					return err
				}
			}
			if err := evt1EvaluateStaticAssert(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(local, env), &StaticAssert{Condition: s.Condition, Message: s.Message, Span: s.Span}); err != nil {
				return err
			}
		case *MatchStmt:
			if err := validateMatchStmt(env, local, *s, returnType, templateInfo, inComptimeFn); err != nil {
				return err
			}
		case *WhileStmt:
			if err := validateWhileStmt(env, local, *s, templateInfo, inComptimeFn); err != nil {
				return err
			}
		case *IfStmt:
			conditionType, err := validateExpr(env, local, s.Condition, templateInfo, inComptimeFn)
			if err != nil {
				return err
			}
			if conditionType.Name != "bool" {
				return evt1Diagnostic("CV4186", "if statement condition must be bool", s.Condition.exprSpan())
			}
			thenScope := evt1CloneScope(local)
			elseScope := evt1CloneScope(local)
			if err := validateBlock(env, thenScope, returnType, s.Then, templateInfo, inComptimeFn); err != nil {
				return err
			}
			if s.Else != nil {
				if err := validateBlock(env, elseScope, returnType, *s.Else, templateInfo, inComptimeFn); err != nil {
					return err
				}
			}
			evt1MergeScopeStates(local, thenScope, elseScope)
		case *Block:
			if err := validateBlock(env, local, returnType, *s, templateInfo, inComptimeFn); err != nil {
				return err
			}
		default:
			return evt1Diagnostic("CV4023", "unsupported statement", stmt.statementSpan())
		}
	}
	return evt1ValidateMaybeDropOwners(env, local)
}

func evt1ValidateMaybeDropOwners(env *semanticEnv, scope *evt1Scope) error {
	var names []string
	for name := range scope.values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		binding := scope.values[name]
		if binding.state == evt1StorageMaybeMoved && evt1TypeHasDrop(env, binding.t) {
			return evt1Diagnostic("CV4516", fmt.Sprintf("drop responsibility for %s is ambiguous after control-flow join", name), binding.t.Span)
		}
	}
	return nil
}

func evt1EvalScopeFromValidation(scope *evt1Scope, env *semanticEnv) *evt1EvalScope {
	var root *evt1EvalScope
	if scope.parent != nil {
		root = evt1EvalScopeFromValidation(scope.parent, env)
	} else {
		root = evt1SeedComptimeScope(env)
	}
	current := newEVT1EvalScope(root)
	for name, binding := range scope.values {
		if !binding.comptime {
			continue
		}
		if binding.hasValue {
			current.declare(name, evt1EvalBinding{value: binding.value, mutable: binding.mutable, comptime: true})
			continue
		}
		if value, ok := env.comptimeValues[name]; ok {
			current.declare(name, evt1EvalBinding{value: value, mutable: false, comptime: true})
		}
	}
	return current
}

func validateExprAgainstExpected(env *semanticEnv, scope *evt1Scope, expr Expr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if lit, ok := expr.(*ArrayLiteralExpr); ok && expected.ArrayElem != nil {
		return validateArrayLiteralExpr(env, scope, *lit, &expected, templateInfo, inComptimeFn)
	}
	if construct, ok := expr.(*ConstructExpr); ok && evt1IsFailureType(expected) {
		return validateFailureConstructExpr(env, scope, construct, expected, templateInfo, inComptimeFn)
	}
	if conditional, ok := expr.(*IfExpr); ok && evt1IsFailureType(expected) {
		conditionType, err := validateExpr(env, scope, conditional.Condition, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if conditionType.Name != "bool" {
			return Type{}, evt1Diagnostic("CV4186", "if expression condition must be bool", conditional.Condition.exprSpan())
		}
		if _, err := validateExprAgainstExpected(env, scope, conditional.Then, expected, templateInfo, inComptimeFn); err != nil {
			return Type{}, err
		}
		if _, err := validateExprAgainstExpected(env, scope, conditional.Else, expected, templateInfo, inComptimeFn); err != nil {
			return Type{}, err
		}
		return expected, nil
	}
	return validateExpr(env, scope, expr, templateInfo, inComptimeFn)
}

func evt1ResolveType(env *semanticEnv, scope *evt1Scope, t Type) (Type, error) {
	if t.PointerTo != nil {
		base, err := evt1ResolveType(env, scope, *t.PointerTo)
		if err != nil {
			return Type{}, err
		}
		t.PointerTo = &base
		return evt1CanonicalType(env, t), nil
	}
	for i := range t.TypeArgs {
		resolved, err := evt1ResolveType(env, scope, t.TypeArgs[i])
		if err != nil {
			return Type{}, err
		}
		t.TypeArgs[i] = resolved
	}
	if t.ArrayElem != nil {
		elem, err := evt1ResolveType(env, scope, *t.ArrayElem)
		if err != nil {
			return Type{}, err
		}
		dimensions := t.Shape
		if len(dimensions) == 0 && t.ArrayLengthExpr != nil {
			dimensions = []StorageDimension{{Expr: t.ArrayLengthExpr}}
		}
		if len(dimensions) == 0 {
			return Type{}, evt1Diagnostic("CV4220", "storage types require an explicit shape", t.Span)
		}
		resolvedShape := make([]StorageDimension, 0, len(dimensions))
		for _, dimension := range dimensions {
			expr := dimension.Expr
			if expr == nil {
				resolvedShape = append(resolvedShape, dimension)
				continue
			}
			evalScope := evt1SeedComptimeScope(env)
			if scope != nil {
				evalScope = evt1EvalScopeFromValidation(scope, env)
			}
			value, evalErr := evt1EvalExpr(newEVT1ComptimeState(env), evalScope, expr)
			if evalErr == nil {
				if value.Kind != ValueInt {
					return Type{}, evt1Diagnostic("CV4221", "storage extent must evaluate to int", expr.exprSpan())
				}
				if value.IntValue < 0 {
					return Type{}, evt1Diagnostic("CV4222", fmt.Sprintf("storage extent %d must be non-negative", value.IntValue), expr.exprSpan())
				}
				if value.IntValue > evt1ComptimeMaxArrayLength {
					return Type{}, evt1Diagnostic("CV4223", fmt.Sprintf("storage extent %d exceeds limit %d", value.IntValue, evt1ComptimeMaxArrayLength), expr.exprSpan())
				}
				resolvedShape = append(resolvedShape, StorageDimension{Extent: value.IntValue, Expression: fmt.Sprintf("%d", value.IntValue), Expr: expr})
				continue
			}
			if scope == nil {
				return Type{}, evalErr
			}
			extentType, typeErr := validateExpr(env, scope, expr, nil, false)
			if typeErr != nil {
				return Type{}, typeErr
			}
			if extentType.Name != "int" {
				return Type{}, evt1Diagnostic("CV4221", "runtime storage extent must be int", expr.exprSpan())
			}
			resolvedShape = append(resolvedShape, StorageDimension{Runtime: true, Expression: evt1ExprIdentity(expr), Expr: expr})
		}
		kind := t.StorageKind
		if kind == "" {
			kind = StorageArray
		}
		resolved := Type{
			Name:        elem.String() + "[]",
			Kind:        TypeArray,
			Ownership:   t.Ownership,
			Const:       t.Const,
			Scoped:      t.Scoped,
			ArrayElem:   &elem,
			StorageKind: kind,
			Shape:       resolvedShape,
			Contiguous:  true,
			Layout:      "row-major",
			Span:        t.Span,
		}
		if kind == StorageNDArray {
			resolved.Kind = TypeNDArray
		}
		if len(resolvedShape) == 1 && !resolvedShape[0].Runtime {
			resolved.ArrayLength = resolvedShape[0].Extent
		}
		if len(resolvedShape) > evt1ComptimeMaxArrayNesting {
			return Type{}, evt1Diagnostic("CV4223", fmt.Sprintf("storage rank %d exceeds limit %d", len(resolvedShape), evt1ComptimeMaxArrayNesting), t.Span)
		}
		if !evt1StorageHasRuntimeShape(resolved) {
			cells := evt1StorageElementCount(resolved)
			if cells > evt1ComptimeMaxArrayCells {
				return Type{}, evt1Diagnostic("CV4224", fmt.Sprintf("storage cell count %d exceeds limit %d", cells, evt1ComptimeMaxArrayCells), t.Span)
			}
		}
		return resolved, nil
	}
	return evt1CanonicalType(env, t), nil
}

func evt1ResolveArrayLength(env *semanticEnv, scope *evt1Scope, expr Expr, span Span) (int, error) {
	if expr == nil {
		return 0, evt1Diagnostic("CV4220", "fixed-array types require an explicit length expression", span)
	}
	evalScope := evt1SeedComptimeScope(env)
	if scope != nil {
		evalScope = evt1EvalScopeFromValidation(scope, env)
	}
	value, err := evt1EvalExpr(newEVT1ComptimeState(env), evalScope, expr)
	if err != nil {
		return 0, err
	}
	if value.Kind != ValueInt {
		return 0, evt1Diagnostic("CV4221", "fixed-array length must evaluate to int", expr.exprSpan())
	}
	if value.IntValue < 0 {
		return 0, evt1Diagnostic("CV4222", fmt.Sprintf("fixed-array length %d must be non-negative", value.IntValue), expr.exprSpan())
	}
	if value.IntValue > evt1ComptimeMaxArrayLength {
		return 0, evt1Diagnostic("CV4223", fmt.Sprintf("fixed-array length %d exceeds limit %d", value.IntValue, evt1ComptimeMaxArrayLength), expr.exprSpan())
	}
	return value.IntValue, nil
}

func evt1ArrayDepth(t Type) int {
	if t.ArrayElem == nil {
		return 0
	}
	return 1 + evt1ArrayDepth(*t.ArrayElem)
}

func evt1TypeContainsArray(env *semanticEnv, t Type) bool {
	if t.ArrayElem != nil {
		return true
	}
	if t.PointerTo != nil && evt1TypeContainsArray(env, *t.PointerTo) {
		return true
	}
	for _, arg := range t.TypeArgs {
		if evt1TypeContainsArray(env, arg) {
			return true
		}
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		for _, field := range structDecl.Fields {
			if evt1TypeContainsArray(env, field.Type) {
				return true
			}
		}
	}
	if enumDecl, ok := env.enums[t.Name]; ok {
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if evt1TypeContainsArray(env, field.Type) {
					return true
				}
			}
		}
	}
	return false
}

func evt1ComptimeTypeCellCount(env *semanticEnv, t Type) (int, error) {
	if t.ArrayElem != nil {
		elemCells, err := evt1ComptimeTypeCellCount(env, *t.ArrayElem)
		if err != nil {
			return 0, err
		}
		if elemCells == 0 || t.ArrayLength == 0 {
			return 0, nil
		}
		if elemCells > evt1ComptimeMaxArrayCells/t.ArrayLength {
			return 0, evt1Diagnostic("CV4224", fmt.Sprintf("array cell count overflows the limit %d", evt1ComptimeMaxArrayCells), t.Span)
		}
		return elemCells * t.ArrayLength, nil
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		total := 0
		for _, field := range structDecl.Fields {
			fieldType, err := evt1ResolveType(env, nil, field.Type)
			if err != nil {
				return 0, err
			}
			cells, err := evt1ComptimeTypeCellCount(env, fieldType)
			if err != nil {
				return 0, err
			}
			if cells < 1 {
				cells = 1
			}
			total += cells
		}
		return total, nil
	}
	return 1, nil
}

func evt1TypeEqualityAvailable(env *semanticEnv, t Type) bool {
	if t.ArrayElem != nil {
		return evt1TypeEqualityAvailable(env, *t.ArrayElem)
	}
	if _, ok := env.profile.builtinType(t.Name, t.Span); ok {
		return t.Name == "int" || t.Name == "bool" || t.Name == "string"
	}
	if _, ok := env.enums[t.Name]; ok {
		return true
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		for _, field := range structDecl.Fields {
			fieldType, err := evt1ResolveType(env, nil, field.Type)
			if err != nil || !evt1TypeEqualityAvailable(env, fieldType) {
				return false
			}
		}
		return true
	}
	return false
}

func validateArrayLiteralExpr(env *semanticEnv, scope *evt1Scope, expr ArrayLiteralExpr, expected *Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if expected != nil && expected.StorageKind == StorageNDArray {
		return validateNDArrayLiteralExpr(env, scope, expr, *expected, templateInfo, inComptimeFn)
	}
	if len(expr.Elements) > evt1ComptimeMaxLiteralElements {
		return Type{}, evt1Diagnostic("CV4224", fmt.Sprintf("array literal element count %d exceeds limit %d", len(expr.Elements), evt1ComptimeMaxLiteralElements), expr.Span)
	}
	var arrayType Type
	if expected != nil && expected.ArrayElem != nil {
		arrayType = evt1CanonicalType(env, *expected)
		if len(expr.Elements) != arrayType.ArrayLength {
			return Type{}, evt1Diagnostic("CV4226", fmt.Sprintf("array literal expected %d elements but got %d", arrayType.ArrayLength, len(expr.Elements)), expr.Span)
		}
	} else if len(expr.Elements) == 0 {
		return Type{}, evt1Diagnostic("CV4225", "empty array literal requires an explicit fixed-array type", expr.Span)
	}
	for i, element := range expr.Elements {
		var elemExpected *Type
		if arrayType.ArrayElem != nil {
			elemExpected = arrayType.ArrayElem
		}
		elementType, err := validateExpr(env, scope, element, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if i == 0 && arrayType.ArrayElem == nil {
			elem := evt1CanonicalType(env, elementType.valueType())
			arrayType = Type{
				Name:        elem.String() + "[]",
				Kind:        TypeArray,
				ArrayElem:   &elem,
				ArrayLength: len(expr.Elements),
				StorageKind: StorageArray,
				Shape:       []StorageDimension{{Extent: len(expr.Elements), Expression: fmt.Sprintf("%d", len(expr.Elements))}},
				Contiguous:  true,
				Layout:      "row-major",
				Span:        expr.Span,
			}
		}
		if elemExpected != nil && !elemExpected.valueType().Equal(elementType.valueType()) {
			return Type{}, evt1Diagnostic("CV4227", fmt.Sprintf("array literal element %d expected %s but got %s", i+1, elemExpected.String(), elementType.String()), element.exprSpan())
		}
		if elemExpected == nil && arrayType.ArrayElem != nil && !arrayType.ArrayElem.valueType().Equal(elementType.valueType()) {
			return Type{}, evt1Diagnostic("CV4227", fmt.Sprintf("array literal element %d expected %s but got %s", i+1, arrayType.ArrayElem.String(), elementType.String()), element.exprSpan())
		}
	}
	if arrayType.ArrayElem == nil {
		return Type{}, evt1Diagnostic("CV4225", "empty array literal requires an explicit fixed-array type", expr.Span)
	}
	return arrayType, nil
}

func validateNDArrayLiteralExpr(env *semanticEnv, scope *evt1Scope, expr ArrayLiteralExpr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	shape, leaves, ragged, rankMismatch := evt1NestedLiteralShape(&expr)
	if ragged {
		return Type{}, evt1Diagnostic("CV4556", "ndarray literal is ragged; every dimension must be rectangular", expr.Span)
	}
	if rankMismatch || len(shape) != evt1StorageRank(expected) {
		return Type{}, evt1Diagnostic("CV4554", fmt.Sprintf("ndarray literal rank %d does not match rank %d", len(shape), evt1StorageRank(expected)), expr.Span)
	}
	for i := range shape {
		if expected.Shape[i].Runtime || shape[i] != expected.Shape[i].Extent {
			return Type{}, evt1Diagnostic("CV4555", fmt.Sprintf("ndarray literal shape %v does not match %s", shape, expected.String()), expr.Span)
		}
	}
	for i, leaf := range leaves {
		leafType, err := validateExpr(env, scope, leaf, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !expected.ArrayElem.valueType().Equal(leafType.valueType()) {
			return Type{}, evt1Diagnostic("CV4227", fmt.Sprintf("ndarray literal element %d expected %s but got %s", i+1, expected.ArrayElem.String(), leafType.String()), leaf.exprSpan())
		}
	}
	return expected, nil
}

func evt1NestedLiteralShape(expr Expr) ([]int, []Expr, bool, bool) {
	literal, ok := expr.(*ArrayLiteralExpr)
	if !ok {
		return nil, []Expr{expr}, false, false
	}
	shape := []int{len(literal.Elements)}
	var leaves []Expr
	var childShape []int
	childrenAreLiterals := false
	childrenAreScalars := false
	for i, element := range literal.Elements {
		_, childIsLiteral := element.(*ArrayLiteralExpr)
		childrenAreLiterals = childrenAreLiterals || childIsLiteral
		childrenAreScalars = childrenAreScalars || !childIsLiteral
		subShape, subLeaves, ragged, rankMismatch := evt1NestedLiteralShape(element)
		if ragged || rankMismatch {
			return nil, nil, ragged, rankMismatch
		}
		if i == 0 {
			childShape = subShape
		} else if !slices.Equal(childShape, subShape) {
			return nil, nil, true, false
		}
		leaves = append(leaves, subLeaves...)
	}
	if childrenAreLiterals && childrenAreScalars {
		return nil, nil, true, false
	}
	return append(shape, childShape...), leaves, false, false
}

func validateKnownType(env *semanticEnv, t Type, span Span, conceptParam string, allowConceptApp bool) error {
	if t.Scoped && !t.isReference() && t.Kind != TypeConceptParam && !evt1IsRefStructType(env, t) {
		return evt1Diagnostic("CV4525", fmt.Sprintf("scoped requires a reference or ref struct type, got %s", t.String()), span)
	}
	if t.PointerTo != nil {
		return validateKnownType(env, *t.PointerTo, span, conceptParam, allowConceptApp)
	}
	if t.ArrayElem != nil {
		if err := validateKnownType(env, *t.ArrayElem, span, conceptParam, allowConceptApp); err != nil {
			return err
		}
		if len(t.Shape) == 0 && t.ArrayLengthExpr == nil {
			return evt1Diagnostic("CV4220", "storage types require an explicit shape", span)
		}
		return nil
	}
	if t.Kind == TypeConceptParam {
		if conceptParam != "" && t.Name == conceptParam {
			return nil
		}
		return evt1Diagnostic("CV4148", fmt.Sprintf("unknown concept parameter %s", t.Name), span)
	}
	if len(t.TypeArgs) > 0 {
		if t.Name == "Option" || t.Name == "Result" {
			required := 1
			if t.Name == "Result" {
				required = 2
			}
			if len(t.TypeArgs) != required {
				return evt1Diagnostic("CV4540", fmt.Sprintf("%s requires exactly %d type argument(s), got %d", t.Name, required, len(t.TypeArgs)), span)
			}
			for _, arg := range t.TypeArgs {
				if err := validateKnownType(env, arg, span, conceptParam, false); err != nil {
					return err
				}
				if evt1IsImmovableValueType(env, arg) {
					return evt1Diagnostic("CV4549", fmt.Sprintf("%s payload type %s is immovable and cannot be embedded by value", t.Name, arg.String()), span)
				}
			}
			return nil
		}
		if _, ok := env.concepts[t.Name]; ok {
			if !allowConceptApp {
				return evt1Diagnostic("CV4164", fmt.Sprintf("concept %s cannot be used as a runtime type", t.String()), span)
			}
			if len(t.TypeArgs) != 1 {
				return evt1Diagnostic("CV4149", fmt.Sprintf("concept %s requires exactly one type argument", t.Name), span)
			}
			return validateKnownType(env, t.TypeArgs[0], span, conceptParam, false)
		}
		return evt1Diagnostic("CV4102", fmt.Sprintf("unknown type application %s", t.String()), span)
	}
	if _, ok := env.profile.builtinType(t.Name, span); ok {
		return nil
	}
	if _, ok := env.enums[t.Name]; ok {
		return nil
	}
	if _, ok := env.structs[t.Name]; ok {
		return nil
	}
	if _, ok := env.automata[t.Name]; ok {
		return evt1Diagnostic("CV4263", fmt.Sprintf("automata %s cannot be used as a runtime type", t.Name), span)
	}
	return evt1Diagnostic("CV4102", fmt.Sprintf("unknown enum or type %s", t.Name), span)
}

func validateExpr(env *semanticEnv, scope *evt1Scope, expr Expr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	switch e := expr.(type) {
	case *IntLiteral:
		t, _ := evt1BuiltinType("int", e.Span)
		return t, nil
	case *StringLiteral:
		t, _ := evt1BuiltinType("string", e.Span)
		return t, nil
	case *BoolLiteral:
		t, _ := evt1BuiltinType("bool", e.Span)
		return t, nil
	case *ArrayLiteralExpr:
		return validateArrayLiteralExpr(env, scope, *e, nil, templateInfo, inComptimeFn)
	case *ParenExpr:
		return validateExpr(env, scope, e.Value, templateInfo, inComptimeFn)
	case *NameExpr:
		if binding, ok := scope.lookup(e.Name); ok {
			if err := evt1CheckReadableBinding(e.Name, binding, e.Span); err != nil {
				return Type{}, err
			}
			if binding.isInstance() {
				return Type{}, evt1Diagnostic("CV4272", fmt.Sprintf("instance %s of automata %s cannot be used as an ordinary value; use dispatch(%s, signal)", e.Name, binding.instanceAutomata, e.Name), e.Span)
			}
			if binding.isBatch() {
				return Type{}, evt1Diagnostic("CV4305", fmt.Sprintf("effects batch %s for automata %s cannot be used as an ordinary value; use dispatch(instance, signal, %s)", e.Name, binding.batchAutomata, e.Name), e.Span)
			}
			if binding.isActuatorLocal() {
				return Type{}, evt1Diagnostic("CV4319", fmt.Sprintf("actuator local %s of actuator %s cannot be used as an ordinary value; use actuation ... = actuate(batch, %s)", e.Name, binding.actuatorName, e.Name), e.Span)
			}
			if binding.t.isReference() {
				return evt1CanonicalType(env, binding.t.borrowBase()), nil
			}
			return evt1CanonicalType(env, binding.t), nil
		}
		if _, ok := env.automata[e.Name]; ok {
			return Type{}, evt1Diagnostic("CV4263", fmt.Sprintf("automata %s cannot be used as a runtime expression", e.Name), e.Span)
		}
		if bindingSpan, ok := env.escapedArmBinding[e.Name]; ok {
			return Type{}, evt1Diagnostic("CV4114", fmt.Sprintf("payload binding %s is scoped to its match arm", e.Name), bindingSpan)
		}
		return Type{}, evt1Diagnostic("CV4024", fmt.Sprintf("unknown name %s", e.Name), e.Span)
	case *FieldExpr:
		receiverType, err := validateExpr(env, scope, e.Receiver, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if templateInfo != nil && evt1TypeDependsOnParam(receiverType, templateInfo.Decl.TypeParam) {
			return Type{}, evt1Diagnostic("CV4172", "dependent field access is not allowed in EVT1 M1B-B templates", e.Span)
		}
		fields, baseName, err := evt1FieldSet(env, receiverType)
		if err != nil {
			return Type{}, evt1Diagnostic("CV4025", err.Error(), e.Span)
		}
		fieldType, ok := fields[e.Field]
		if !ok {
			return Type{}, evt1Diagnostic("CV4026", fmt.Sprintf("unknown field %s on %s", e.Field, baseName), e.Span)
		}
		if fieldType.isReference() {
			return evt1CanonicalType(env, fieldType.borrowBase()), nil
		}
		return evt1CanonicalType(env, fieldType), nil
	case *CallExpr:
		if e.Callee == "discard" {
			if len(e.Args) != 1 {
				return Type{}, evt1Diagnostic("CV4323", fmt.Sprintf("discard requires exactly one batch argument, got %d", len(e.Args)), e.Span)
			}
			nameExpr, ok := e.Args[0].(*NameExpr)
			if !ok {
				return Type{}, evt1Diagnostic("CV4323", "discard requires a local effects batch name", e.Args[0].exprSpan())
			}
			binding, ok := scope.lookup(nameExpr.Name)
			if !ok || !binding.isBatch() {
				return Type{}, evt1Diagnostic("CV4323", fmt.Sprintf("discard requires a local effects batch, but %s is not one", nameExpr.Name), e.Args[0].exprSpan())
			}
			return Type{Name: "void", Kind: TypeBuiltin, Span: e.Span}, nil
		}
		if e.Callee == "Len" {
			if len(e.Args) != 1 {
				return Type{}, evt1Diagnostic("CV4234", fmt.Sprintf("Len expects exactly one argument, got %d", len(e.Args)), e.Span)
			}
			argType, err := validateExpr(env, scope, e.Args[0], templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if argType.ArrayElem == nil || evt1StorageRank(argType) != 1 {
				return Type{}, evt1Diagnostic("CV4235", "Len requires a rank-1 array argument", e.Args[0].exprSpan())
			}
			out, _ := evt1BuiltinType("int", e.Span)
			return out, nil
		}
		if e.Callee == "Rank" {
			if len(e.Args) != 1 {
				return Type{}, evt1Diagnostic("CV4551", fmt.Sprintf("Rank expects exactly one argument, got %d", len(e.Args)), e.Span)
			}
			argType, err := validateExpr(env, scope, e.Args[0], templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if argType.ArrayElem == nil {
				return Type{}, evt1Diagnostic("CV4551", "Rank requires array or ndarray storage", e.Args[0].exprSpan())
			}
			out, _ := evt1BuiltinType("int", e.Span)
			return out, nil
		}
		if e.Callee == "Shape" {
			if len(e.Args) != 2 {
				return Type{}, evt1Diagnostic("CV4552", fmt.Sprintf("Shape expects storage and dimension arguments, got %d argument(s)", len(e.Args)), e.Span)
			}
			argType, err := validateExpr(env, scope, e.Args[0], templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if argType.ArrayElem == nil {
				return Type{}, evt1Diagnostic("CV4552", "Shape requires array or ndarray storage", e.Args[0].exprSpan())
			}
			dimType, err := validateExpr(env, scope, e.Args[1], templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if dimType.Name != "int" {
				return Type{}, evt1Diagnostic("CV4552", "Shape dimension must be int", e.Args[1].exprSpan())
			}
			if value, err := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), e.Args[1]); err == nil && (value.IntValue < 0 || value.IntValue >= evt1StorageRank(argType)) {
				return Type{}, evt1Diagnostic("CV4553", fmt.Sprintf("shape dimension %d is out of range for rank %d", value.IntValue, evt1StorageRank(argType)), e.Args[1].exprSpan())
			}
			out, _ := evt1BuiltinType("int", e.Span)
			return out, nil
		}
		if templateInfo != nil {
			return validateTemplateCallExpr(env, scope, *e, templateInfo)
		}
		if _, exists := env.comptimeFunctions[e.Callee]; exists && !inComptimeFn {
			return Type{}, evt1Diagnostic("CV4210", fmt.Sprintf("comptime function %s cannot be called from runtime code", e.Callee), e.Span)
		}
		argTypes := make([]Type, 0, len(e.Args))
		for _, arg := range e.Args {
			argType, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			argTypes = append(argTypes, argType)
		}
		if inComptimeFn {
			if comptimeFn, ok := env.comptimeFunctions[e.Callee]; ok {
				if len(comptimeFn.Params) != len(e.Args) {
					return Type{}, evt1Diagnostic("CV4106", fmt.Sprintf("wrong constructor or call payload count for %s: expected %d but got %d", e.Callee, len(comptimeFn.Params), len(e.Args)), e.Span)
				}
				for i, arg := range e.Args {
					if err := validateCallArgument(env, scope, comptimeFn.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
						return Type{}, err
					}
				}
				return evt1CanonicalType(env, comptimeFn.ReturnType), nil
			}
		}
		fn, err := evt1ResolveOrdinaryCall(env, scope, e.Callee, e.Args, argTypes, templateInfo, e.Span)
		if err != nil {
			if _, exists := env.templates[e.Callee]; exists {
				return Type{}, evt1Diagnostic("CV4173", fmt.Sprintf("template call %s requires an explicit concrete type argument", e.Callee), e.Span)
			}
			return Type{}, err
		}
		for i, arg := range e.Args {
			if err := validateCallArgument(env, scope, fn.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
				return Type{}, err
			}
		}
		return evt1CanonicalType(env, fn.ReturnType), nil
	case *DispatchExpr:
		if inComptimeFn {
			return Type{}, evt1Diagnostic("CV4275", "dispatch is not available during comptime evaluation", e.Span)
		}
		binding, ok := scope.lookup(e.InstanceName)
		if !ok {
			return Type{}, evt1Diagnostic("CV4273", fmt.Sprintf("dispatch requires a local instance, but %s is unknown", e.InstanceName), e.Span)
		}
		if !binding.isInstance() {
			return Type{}, evt1Diagnostic("CV4273", fmt.Sprintf("dispatch requires a local instance as its first operand, but %s is not an instance", e.InstanceName), e.Span)
		}
		info := env.automataInfo[binding.instanceAutomata]
		expected := evt1CanonicalType(env, info.Decl.SignalType)
		signalType, err := validateExprAgainstExpected(env, scope, e.Signal, expected, templateInfo, false)
		if err != nil {
			return Type{}, err
		}
		if !evt1CanonicalType(env, signalType.valueType()).Equal(evt1CanonicalType(env, expected.valueType())) {
			return Type{}, evt1Diagnostic("CV4274", fmt.Sprintf("dispatch(%s, ...) expects signal type %s but got %s", e.InstanceName, expected.String(), signalType.String()), e.Signal.exprSpan())
		}
		if len(info.EffectSet) > 0 {
			if e.BatchName == "" {
				return Type{}, evt1Diagnostic("CV4306", fmt.Sprintf("effectful automata %s requires dispatch(%s, signal, batch)", info.Decl.Name, e.InstanceName), e.Span)
			}
			batchBinding, ok := scope.lookup(e.BatchName)
			if !ok || !batchBinding.isBatch() {
				return Type{}, evt1Diagnostic("CV4307", fmt.Sprintf("dispatch(%s, ...) requires a local effects batch as its third operand, but %s is not one", e.InstanceName, e.BatchName), e.Span)
			}
			if batchBinding.batchAutomata != info.Decl.Name {
				return Type{}, evt1Diagnostic("CV4308", fmt.Sprintf("dispatch(%s, ...) requires an effects batch for automata %s, but %s belongs to %s", e.InstanceName, info.Decl.Name, e.BatchName, batchBinding.batchAutomata), e.Span)
			}
		} else if e.BatchName != "" {
			return Type{}, evt1Diagnostic("CV4309", fmt.Sprintf("effect-free automata %s does not accept a third dispatch operand", info.Decl.Name), e.Span)
		}
		return Type{Name: evt1AutomataDispatchOutcomeTypeName, Kind: TypeEnum, Span: e.Span}, nil
	case *TemplateCallExpr:
		if inComptimeFn {
			return Type{}, evt1Diagnostic("CV4201", "templates are not available during comptime evaluation", e.Span)
		}
		if templateInfo != nil {
			return Type{}, evt1Diagnostic("CV4174", "templates cannot invoke templates in EVT1 M1B-B", e.Span)
		}
		instance, err := instantiateTemplate(env, e.Callee, e.TypeArg, e.Span)
		if err != nil {
			return Type{}, err
		}
		if len(instance.Function.Params) != len(e.Args) {
			return Type{}, evt1Diagnostic("CV4106", fmt.Sprintf("wrong constructor or call payload count for %s: expected %d but got %d", e.Callee, len(instance.Function.Params), len(e.Args)), e.Span)
		}
		for i, arg := range e.Args {
			argType, err := validateExpr(env, scope, arg, nil, false)
			if err != nil {
				return Type{}, err
			}
			if err := validateCallArgument(env, scope, instance.Function.Params[i].Type, arg, argType, nil); err != nil {
				return Type{}, err
			}
		}
		instance.InvocationSpans = append(instance.InvocationSpans, e.Span)
		return evt1CanonicalType(env, instance.Function.ReturnType), nil
	case *IndexExpr:
		baseType, err := validateExpr(env, scope, e.Base, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if baseType.ArrayElem == nil {
			return Type{}, evt1Diagnostic("CV4231", fmt.Sprintf("index target %s is not array or ndarray storage", baseType.String()), e.Base.exprSpan())
		}
		indices := evt1StorageIndices(e)
		rank := evt1StorageRank(baseType)
		if len(indices) != rank {
			code := "CV4550"
			if baseType.StorageKind == StorageNDArray {
				code = "CV4557"
			}
			return Type{}, evt1Diagnostic(code, fmt.Sprintf("%s index requires %d index(es), got %d", baseType.StorageKind, rank, len(indices)), e.Span)
		}
		for i, indexExpr := range indices {
			indexType, err := validateExpr(env, scope, indexExpr, templateInfo, true)
			if err != nil {
				return Type{}, err
			}
			if indexType.Name != "int" {
				return Type{}, evt1Diagnostic("CV4232", "storage index must be int", indexExpr.exprSpan())
			}
			indexValue, evalErr := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), indexExpr)
			if evalErr == nil && !baseType.Shape[i].Runtime {
				if indexValue.Kind != ValueInt {
					return Type{}, evt1Diagnostic("CV4232", "storage index must evaluate to int", indexExpr.exprSpan())
				}
				if indexValue.IntValue < 0 || indexValue.IntValue >= baseType.Shape[i].Extent {
					code := "CV4233"
					if baseType.StorageKind == StorageNDArray {
						code = "CV4561"
					}
					return Type{}, evt1Diagnostic(code, fmt.Sprintf("%s index %d is out of bounds for dimension %d extent %d", baseType.StorageKind, indexValue.IntValue, i, baseType.Shape[i].Extent), indexExpr.exprSpan())
				}
			}
		}
		return evt1CanonicalType(env, *baseType.ArrayElem), nil
	case *UnaryExpr:
		valueType, err := validateExpr(env, scope, e.Value, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		switch e.Op {
		case "-":
			if valueType.Name != "int" {
				return Type{}, evt1Diagnostic("CV4028", "unary - requires int", e.Span)
			}
			return valueType, nil
		case "not":
			if valueType.Name != "bool" {
				return Type{}, evt1Diagnostic("CV4028", "not requires bool", e.Span)
			}
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		default:
			return Type{}, evt1Diagnostic("CV4028", "unsupported unary operator "+e.Op, e.Span)
		}
	case *MoveExpr:
		name, ok := e.Value.(*NameExpr)
		if !ok {
			return Type{}, evt1Diagnostic("CV4507", "move requires a whole local or parameter place", e.Value.exprSpan())
		}
		binding, ok := scope.lookup(name.Name)
		if !ok {
			return Type{}, evt1Diagnostic("CV4024", fmt.Sprintf("unknown name %s", name.Name), name.Span)
		}
		if binding.state == evt1StorageMoved {
			return Type{}, evt1Diagnostic("CV4504", fmt.Sprintf("%s was already moved", name.Name), e.Span)
		}
		if binding.state == evt1StorageMaybeMoved {
			return Type{}, evt1Diagnostic("CV4503", fmt.Sprintf("%s is moved on some control-flow paths", name.Name), e.Span)
		}
		if evt1IsImmovableValueType(env, binding.t) {
			return Type{}, evt1Diagnostic("CV4505", fmt.Sprintf("immovable type %s cannot be relocated", binding.t.String()), e.Span)
		}
		if !evt1TypeMovable(env, binding.t) {
			return Type{}, evt1Diagnostic("CV4505", fmt.Sprintf("type %s is not movable", binding.t.String()), e.Span)
		}
		if !evt1TypeCopyable(env, binding.t) {
			if !binding.mutable {
				return Type{}, evt1Diagnostic("CV4128", "move requires a mutable owning place", e.Span)
			}
			scope.setState(name.Name, evt1StorageMoved)
		}
		return evt1CanonicalType(env, binding.t), nil
	case *RefExpr:
		lvalue, err := validateAssignable(env, scope, e.Value, templateInfo)
		if err != nil {
			return Type{}, evt1Diagnostic("CV4508", "ref requires an existing place; temporaries are not referenceable in R3", e.Value.exprSpan())
		}
		if binding, ok := scope.lookup(lvalue.path.Root); ok {
			if err := evt1CheckReadableBinding(lvalue.path.Root, binding, e.Value.exprSpan()); err != nil {
				return Type{}, err
			}
		}
		if !e.Const && !lvalue.mutable {
			return Type{}, evt1Diagnostic("CV4509", "mutable ref cannot bind a const or read-only place", e.Value.exprSpan())
		}
		out := lvalue.t.valueType()
		out.Ownership = "ref"
		out.Const = e.Const || !lvalue.mutable
		out.Span = e.Span
		return out, nil
	case *BinaryExpr:
		leftType, err := validateExpr(env, scope, e.Left, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		rightType, err := validateExpr(env, scope, e.Right, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if templateInfo != nil && (evt1TypeDependsOnParam(leftType, templateInfo.Decl.TypeParam) || evt1TypeDependsOnParam(rightType, templateInfo.Decl.TypeParam)) {
			return Type{}, evt1Diagnostic("CV4175", "dependent operators are not allowed in EVT1 M1B-B templates", e.Span)
		}
		if leftType.Name == "bool" && rightType.Name == "bool" && (e.Op == "and" || e.Op == "or" || e.Op == "==" || e.Op == "!=") {
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		}
		if leftType.Name == "string" && rightType.Name == "string" && (e.Op == "==" || e.Op == "!=") {
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		}
		if (leftType.ArrayElem != nil || rightType.ArrayElem != nil) && (e.Op == "<" || e.Op == ">" || e.Op == "<=" || e.Op == ">=") {
			return Type{}, evt1Diagnostic("CV4236", "array ordering comparisons are not supported", e.Span)
		}
		if leftType.Name == "int" && rightType.Name == "int" {
			if e.Op == "<" || e.Op == ">" || e.Op == "<=" || e.Op == ">=" || e.Op == "==" || e.Op == "!=" {
				out, _ := evt1BuiltinType("bool", e.Span)
				return out, nil
			}
			if e.Op == "+" || e.Op == "-" || e.Op == "*" {
				return leftType, nil
			}
		}
		if leftType.Name == "bool" && rightType.Name == "bool" {
			if e.Op == "==" || e.Op == "!=" {
				out, _ := evt1BuiltinType("bool", e.Span)
				return out, nil
			}
		}
		if leftType.Name == rightType.Name && leftType.Kind == TypeEnum && (e.Op == "==" || e.Op == "!=") {
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		}
		if leftType.Name == rightType.Name && leftType.Kind == TypeStruct && (e.Op == "==" || e.Op == "!=") && evt1IsComptimeType(env, leftType) {
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		}
		if leftType.ArrayElem != nil || rightType.ArrayElem != nil {
			if e.Op != "==" && e.Op != "!=" {
				return Type{}, evt1Diagnostic("CV4236", "fixed compile-time arrays only support == and !=", e.Span)
			}
			if !evt1CanonicalType(env, leftType).Equal(evt1CanonicalType(env, rightType)) {
				return Type{}, evt1Diagnostic("CV4238", fmt.Sprintf("array equality requires identical fixed-array types, got %s and %s", leftType.String(), rightType.String()), e.Span)
			}
			if !evt1TypeEqualityAvailable(env, leftType) {
				return Type{}, evt1Diagnostic("CV4239", fmt.Sprintf("array equality is unavailable for element type %s", leftType.ArrayElem.String()), e.Span)
			}
			out, _ := evt1BuiltinType("bool", e.Span)
			return out, nil
		}
		if leftType.Name == "uint64" && rightType.Name == "uint64" {
			if e.Op == "<" || e.Op == ">" || e.Op == "<=" || e.Op == ">=" {
				out, _ := evt1BuiltinType("bool", e.Span)
				return out, nil
			}
			return leftType, nil
		}
		return Type{}, evt1Diagnostic("CV4028", "only int/uint64 additive and comparison expressions are supported in EVT1", e.Span)
	case *ConstructExpr:
		if e.EnumName == "Option" || e.EnumName == "Result" {
			if evt1IsFailureType(e.ResolvedType) {
				return e.ResolvedType, nil
			}
			return Type{}, evt1Diagnostic("CV4540", fmt.Sprintf("%s constructor requires an Option/Result expected type", e.EnumName), e.Span)
		}
		return validateConstructExpr(env, scope, *e)
	case *FailureExpr:
		operand, err := validateExpr(env, scope, e.Value, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !evt1IsFailureType(operand) {
			code := "CV4541"
			if e.Op == "!" {
				code = "CV4544"
			}
			return Type{}, evt1Diagnostic(code, fmt.Sprintf("%s requires Option or Result, got %s", e.Op, operand.String()), e.Span)
		}
		e.ResolvedType = operand
		if !evt1TypeCopyable(env, operand) && !evt1CanTransferInitialize(env, operand, e.Value) {
			return Type{}, evt1Diagnostic("CV4548", fmt.Sprintf("%s consumes non-copyable %s; use move for an existing owner", e.Op, operand.String()), e.Value.exprSpan())
		}
		if e.Op == "?" {
			if evt1IsOptionType(operand) {
				if !evt1IsOptionType(scope.returnType) {
					return Type{}, evt1Diagnostic("CV4542", "Option propagation requires an Option-returning function", e.Span)
				}
			} else {
				errType := evt1FailureErrorType(operand)
				if handled, ok := scope.tryHandlers[evt1TypeIdentity(evt1CanonicalType(env, errType))]; !ok || !handled.Equal(evt1CanonicalType(env, errType)) {
					if !evt1IsResultType(scope.returnType) {
						if scope.tryHandlers != nil {
							return Type{}, evt1Diagnostic("CV4545", fmt.Sprintf("try has no except arm for %s", errType.String()), e.Span)
						}
						return Type{}, evt1Diagnostic("CV4542", "Result propagation requires a Result-returning function or matching local except arm", e.Span)
					}
					if !evt1CanonicalType(env, evt1FailureErrorType(scope.returnType)).Equal(evt1CanonicalType(env, errType)) {
						return Type{}, evt1Diagnostic("CV4543", fmt.Sprintf("Result error type mismatch: %s cannot propagate into %s", errType.String(), evt1FailureErrorType(scope.returnType).String()), e.Span)
					}
				}
			}
		}
		return evt1CanonicalType(env, evt1FailureSuccessType(operand)), nil
	case *StructConstructExpr:
		return validateStructConstructExpr(env, scope, *e)
	case *WithExpr:
		return validateWithExpr(env, scope, *e, templateInfo, inComptimeFn)
	case *IfExpr:
		conditionType, err := validateExpr(env, scope, e.Condition, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if conditionType.Name != "bool" {
			return Type{}, evt1Diagnostic("CV4186", "if expression condition must be bool", e.Condition.exprSpan())
		}
		thenType, err := validateExpr(env, scope, e.Then, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		elseType, err := validateExpr(env, scope, e.Else, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !evt1TypesCompatible(env, thenType, elseType, "") {
			return Type{}, evt1Diagnostic("CV4116", fmt.Sprintf("if expression arms must have the same type: got %s and %s", thenType.String(), elseType.String()), e.Span)
		}
		return evt1CanonicalType(env, thenType), nil
	case *MatchExpr:
		return validateMatchExpr(env, scope, *e, templateInfo, inComptimeFn)
	default:
		return Type{}, evt1Diagnostic("CV4029", fmt.Sprintf("unsupported expression %s", evt1Unexpected(expr)), expr.exprSpan())
	}
}

func validateCallArgument(env *semanticEnv, scope *evt1Scope, paramType Type, arg Expr, argType Type, templateInfo *evt1TemplateInfo) error {
	typeParam := ""
	if templateInfo != nil {
		typeParam = templateInfo.Decl.TypeParam
	}
	if lit, ok := arg.(*ArrayLiteralExpr); ok && paramType.ArrayElem != nil {
		validatedType, err := validateArrayLiteralExpr(env, scope, *lit, &paramType, templateInfo, false)
		if err != nil {
			return err
		}
		argType = validatedType
	}
	if paramType.isReference() {
		if _, ok := arg.(*RefExpr); !ok {
			return evt1Diagnostic("CV4508", fmt.Sprintf("reference parameter %s requires an explicit ref argument", paramType.String()), arg.exprSpan())
		}
		required := paramType.borrowBase()
		if !argType.isReference() || !evt1TypesCompatible(env, required, argType.borrowBase(), typeParam) {
			return evt1Diagnostic("CV4510", fmt.Sprintf("reference argument expected %s but got %s", paramType.String(), argType.String()), arg.exprSpan())
		}
		if !paramType.Const && argType.Const {
			return evt1Diagnostic("CV4509", fmt.Sprintf("mutable reference parameter %s cannot accept a ref const argument", paramType.String()), arg.exprSpan())
		}
		return nil
	}
	if paramType.isBorrow() {
		required := paramType.borrowBase()
		if argType.isBorrowLike() {
			if !evt1TypesCompatible(env, required, argType.borrowBase(), typeParam) {
				return evt1Diagnostic("CV4154", fmt.Sprintf("borrow argument expected %s but got %s", paramType.String(), argType.String()), arg.exprSpan())
			}
			if !paramType.Const && argType.Const {
				return evt1Diagnostic("CV4155", fmt.Sprintf("mutable borrow argument for %s cannot accept const %s", required.String(), argType.String()), arg.exprSpan())
			}
			return nil
		}
		if !evt1TypesCompatible(env, required, argType.valueType(), typeParam) {
			return evt1Diagnostic("CV4154", fmt.Sprintf("borrow argument expected %s but got %s", paramType.String(), argType.String()), arg.exprSpan())
		}
		lvalue, err := validateAssignable(env, scope, arg, templateInfo)
		if err != nil {
			return evt1Diagnostic("CV4127", fmt.Sprintf("borrow argument for %s requires an assignable access path", required.String()), arg.exprSpan())
		}
		if !paramType.Const && !lvalue.mutable {
			return evt1Diagnostic("CV4155", fmt.Sprintf("mutable borrow argument for %s cannot bind a const access path", required.String()), arg.exprSpan())
		}
		return nil
	}
	if !evt1TypesCompatible(env, paramType, argType, typeParam) {
		return evt1Diagnostic("CV4107", fmt.Sprintf("wrong payload type for call argument: expected %s but got %s", paramType.String(), argType.String()), arg.exprSpan())
	}
	if !evt1TypeCopyable(env, argType) && !evt1TypeDependsOnParam(argType, typeParam) && !evt1CanTransferInitialize(env, paramType, arg) {
		return evt1Diagnostic("CV4512", fmt.Sprintf("call would copy non-copyable type %s; pass it with move", argType.String()), arg.exprSpan())
	}
	return nil
}

func validateAssignable(env *semanticEnv, scope *evt1Scope, expr Expr, templateInfo *evt1TemplateInfo) (evt1LValue, error) {
	switch e := expr.(type) {
	case *NameExpr:
		binding, ok := scope.lookup(e.Name)
		if !ok {
			return evt1LValue{}, evt1Diagnostic("CV4127", fmt.Sprintf("unknown assignable target %s", e.Name), e.Span)
		}
		if binding.isInstance() {
			return evt1LValue{}, evt1Diagnostic("CV4272", fmt.Sprintf("instance %s of automata %s cannot be assigned or copied as a value", e.Name, binding.instanceAutomata), e.Span)
		}
		if binding.isBatch() {
			return evt1LValue{}, evt1Diagnostic("CV4305", fmt.Sprintf("effects batch %s of automata %s cannot be assigned or copied as a value", e.Name, binding.batchAutomata), e.Span)
		}
		if binding.isActuatorLocal() {
			return evt1LValue{}, evt1Diagnostic("CV4319", fmt.Sprintf("actuator local %s of actuator %s cannot be assigned or copied as a value", e.Name, binding.actuatorName), e.Span)
		}
		resolvedType := evt1CanonicalType(env, binding.t)
		mutable := binding.mutable
		wholeValue := true
		if binding.t.isReference() {
			resolvedType = evt1CanonicalType(env, binding.t.borrowBase())
			mutable = !binding.t.Const
			wholeValue = false
		}
		readOnlyReason := ""
		if binding.t.isReference() && binding.t.Const {
			readOnlyReason = "ref_const"
		}
		return evt1LValue{
			t:              resolvedType,
			mutable:        mutable,
			wholeValue:     wholeValue,
			readOnlyReason: readOnlyReason,
			path: evt1AccessPath{
				Root: e.Name,
				Span: e.Span,
			},
		}, nil
	case *FieldExpr:
		receiver, err := validateAssignable(env, scope, e.Receiver, templateInfo)
		if err != nil {
			return evt1LValue{}, err
		}
		if templateInfo != nil && evt1TypeDependsOnParam(receiver.t, templateInfo.Decl.TypeParam) {
			return evt1LValue{}, evt1Diagnostic("CV4172", "dependent field access is not allowed in EVT1 M1B-B templates", e.Span)
		}
		fields, baseName, err := evt1FieldSet(env, receiver.t)
		if err != nil {
			return evt1LValue{}, evt1Diagnostic("CV4025", err.Error(), e.Span)
		}
		fieldType, ok := fields[e.Field]
		if !ok {
			return evt1LValue{}, evt1Diagnostic("CV4026", fmt.Sprintf("unknown field %s on %s", e.Field, baseName), e.Span)
		}
		path := receiver.path
		path.Fields = append(append([]string{}, receiver.path.Fields...), e.Field)
		path.Span = e.Span
		mutable := receiver.mutable
		readOnlyReason := receiver.readOnlyReason
		if decl, ok := env.structs[receiver.t.borrowBase().Name]; ok && decl.Record {
			mutable = false
			readOnlyReason = "record"
		}
		if fieldType.isReference() {
			mutable = mutable && !fieldType.Const
			if fieldType.Const {
				readOnlyReason = "ref_const"
			}
			fieldType = fieldType.borrowBase()
		}
		return evt1LValue{t: evt1CanonicalType(env, fieldType), mutable: mutable, wholeValue: false, readOnlyReason: readOnlyReason, path: path}, nil
	case *IndexExpr:
		receiver, err := validateAssignable(env, scope, e.Base, templateInfo)
		if err != nil {
			return evt1LValue{}, evt1Diagnostic("CV4231", "array index assignment requires an assignable storage place", expr.exprSpan())
		}
		if receiver.t.ArrayElem == nil {
			return evt1LValue{}, evt1Diagnostic("CV4231", "array index assignment requires array or ndarray storage", expr.exprSpan())
		}
		if _, err := validateExpr(env, scope, e, templateInfo, false); err != nil {
			return evt1LValue{}, err
		}
		path := receiver.path
		path.Fields = append(append([]string{}, path.Fields...), "[index]")
		path.Span = e.Span
		return evt1LValue{t: evt1CanonicalType(env, *receiver.t.ArrayElem), mutable: receiver.mutable, wholeValue: false, readOnlyReason: receiver.readOnlyReason, path: path}, nil
	default:
		return evt1LValue{}, evt1Diagnostic("CV4127", "assignment requires a local or field access target", expr.exprSpan())
	}
}

func validateWithExpr(env *semanticEnv, scope *evt1Scope, expr WithExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	baseType, err := validateExpr(env, scope, expr.Base, templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	baseType = evt1CanonicalType(env, baseType.valueType())
	decl, ok := env.structs[baseType.Name]
	if !ok || !decl.Record {
		return Type{}, evt1Diagnostic("CV4143", fmt.Sprintf("with requires a record struct value, got %s", baseType.String()), expr.Span)
	}
	if !evt1TypeCopyable(env, baseType) {
		return Type{}, evt1Diagnostic("CV4146", fmt.Sprintf("with cannot copy non-copyable record %s", baseType.String()), expr.Span)
	}
	seen := map[string]bool{}
	for _, update := range expr.Updates {
		fieldType, exists := env.fieldSets[decl.Name][update.Name]
		if !exists {
			return Type{}, evt1Diagnostic("CV4144", fmt.Sprintf("unknown field %s in with update for %s", update.Name, decl.Name), update.NameSpan)
		}
		if seen[update.Name] {
			return Type{}, evt1Diagnostic("CV4145", fmt.Sprintf("duplicate field %s in with update for %s", update.Name, decl.Name), update.NameSpan)
		}
		seen[update.Name] = true
		valueType, err := validateExprAgainstExpected(env, scope, update.Value, evt1CanonicalType(env, fieldType), templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !evt1TypesCompatible(env, fieldType, valueType, "") {
			return Type{}, evt1Diagnostic("CV4147", fmt.Sprintf("with update for %s.%s expected %s but got %s", decl.Name, update.Name, fieldType.String(), valueType.String()), update.Value.exprSpan())
		}
	}
	return baseType, nil
}

func evt1FindOverlappingBorrow(borrows []evt1RetainedBorrow, path evt1AccessPath) (evt1RetainedBorrow, bool) {
	for _, borrow := range borrows {
		if evt1AccessPathsOverlap(borrow.Path, path) {
			return borrow, true
		}
	}
	return evt1RetainedBorrow{}, false
}

func evt1AccessPathsOverlap(a, b evt1AccessPath) bool {
	if a.Root == "" || b.Root == "" || a.Root != b.Root {
		return false
	}
	shared := len(a.Fields)
	if len(b.Fields) < shared {
		shared = len(b.Fields)
	}
	for i := 0; i < shared; i++ {
		if a.Fields[i] != b.Fields[i] {
			return false
		}
	}
	return true
}

func evt1FieldSet(env *semanticEnv, t Type) (map[string]Type, string, error) {
	base := t
	if t.isBorrowLike() {
		base = t.borrowBase()
	}
	fields, ok := env.fieldSets[base.Name]
	if !ok {
		return nil, base.Name, fmt.Errorf("type %s has no fields", base.Name)
	}
	return fields, base.Name, nil
}

func validateConstructExpr(env *semanticEnv, scope *evt1Scope, expr ConstructExpr) (Type, error) {
	enumDecl, ok := env.enums[expr.EnumName]
	if !ok {
		return Type{}, evt1Diagnostic("CV4102", fmt.Sprintf("unknown enum type %s in qualified construction", expr.EnumName), expr.Span)
	}
	variant, ok := evt1LookupVariant(enumDecl, expr.VariantName)
	if !ok {
		return Type{}, evt1Diagnostic("CV4103", fmt.Sprintf("unknown variant %s::%s", expr.EnumName, expr.VariantName), expr.Span)
	}
	if len(variant.Payload) == 0 && len(expr.Args) > 0 {
		return Type{}, evt1Diagnostic("CV4104", fmt.Sprintf("unit variant %s::%s does not accept payload arguments", expr.EnumName, expr.VariantName), expr.Span)
	}
	if len(variant.Payload) > 0 && len(expr.Args) == 0 {
		return Type{}, evt1Diagnostic("CV4104", fmt.Sprintf("payload variant %s::%s requires construction arguments", expr.EnumName, expr.VariantName), expr.Span)
	}
	if len(variant.Payload) != len(expr.Args) {
		return Type{}, evt1Diagnostic("CV4106", fmt.Sprintf("wrong constructor payload count for %s::%s: expected %d but got %d", expr.EnumName, expr.VariantName, len(variant.Payload), len(expr.Args)), expr.Span)
	}
	for i, arg := range expr.Args {
		argType, err := validateExprAgainstExpected(env, scope, arg, variant.Payload[i].Type, nil, false)
		if err != nil {
			return Type{}, err
		}
		if !evt1CanInitializeStoredType(env, variant.Payload[i].Type, argType) {
			return Type{}, evt1Diagnostic("CV4107", fmt.Sprintf("wrong constructor payload type for %s::%s position %d: expected %s but got %s", expr.EnumName, expr.VariantName, i+1, variant.Payload[i].Type.String(), argType.String()), arg.exprSpan())
		}
		if !evt1TypeCopyable(env, argType) {
			return Type{}, evt1Diagnostic("CV4133", fmt.Sprintf("enum construction copies non-copyable type %s", argType.String()), arg.exprSpan())
		}
	}
	return Type{Name: enumDecl.Name, Kind: TypeEnum, Span: expr.Span}, nil
}

func validateStructConstructExpr(env *semanticEnv, scope *evt1Scope, expr StructConstructExpr) (Type, error) {
	structDecl, ok := env.structs[expr.StructName]
	if !ok {
		return Type{}, evt1Diagnostic("CV4125", fmt.Sprintf("unknown struct type %s", expr.StructName), expr.Span)
	}
	if len(structDecl.Fields) != len(expr.Args) {
		return Type{}, evt1Diagnostic("CV4126", fmt.Sprintf("wrong initializer count for %s: expected %d but got %d", expr.StructName, len(structDecl.Fields), len(expr.Args)), expr.Span)
	}
	for i, arg := range expr.Args {
		argType, err := validateExprAgainstExpected(env, scope, arg, structDecl.Fields[i].Type, nil, false)
		if err != nil {
			return Type{}, err
		}
		if structDecl.Fields[i].Type.isReference() {
			if err := validateCallArgument(env, scope, structDecl.Fields[i].Type, arg, argType, nil); err != nil {
				return Type{}, err
			}
		} else if !evt1CanInitializeStoredType(env, structDecl.Fields[i].Type, argType) {
			return Type{}, evt1Diagnostic("CV4107", fmt.Sprintf("wrong initializer type for %s field %s: expected %s but got %s", expr.StructName, structDecl.Fields[i].Name, structDecl.Fields[i].Type.String(), argType.String()), arg.exprSpan())
		}
		if !evt1TypeCopyable(env, argType) && !evt1CanTransferInitialize(env, structDecl.Fields[i].Type, arg) {
			return Type{}, evt1Diagnostic("CV4501", fmt.Sprintf("construction of %s.%s would copy non-copyable type %s", expr.StructName, structDecl.Fields[i].Name, argType.String()), arg.exprSpan())
		}
	}
	return Type{Name: structDecl.Name, Kind: TypeStruct, Span: expr.Span}, nil
}

func evt1IsRefStructType(env *semanticEnv, t Type) bool {
	if evt1IsFailureType(t) {
		for _, arg := range t.TypeArgs {
			if arg.isReference() || evt1IsRefStructType(env, arg) {
				return true
			}
		}
		return false
	}
	decl, ok := env.structs[t.valueType().Name]
	return ok && decl.Ref
}

func evt1InitialParameterProvenance(env *semanticEnv, t Type, index, depth int) evt1LifetimeProvenance {
	if t.isReference() || evt1IsRefStructType(env, t) {
		return evt1LifetimeProvenance{Kind: evt1ProvenanceParameter, ParameterIndex: index, Scoped: t.Scoped}
	}
	return evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: depth, Scoped: t.Scoped}
}

func evt1IsCallResultExpr(expr Expr) bool {
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1IsCallResultExpr(e.Value)
	case *MoveExpr:
		return evt1IsCallResultExpr(e.Value)
	case *CallExpr:
		return true
	default:
		return false
	}
}

func evt1FunctionProvenanceKey(fn FunctionDecl) string {
	return fn.Name + "|" + evt1FunctionParamSignature(fn)
}

func evt1UnknownResultProvenance() evt1ResultProvenanceSummary {
	return evt1ResultProvenanceSummary{Kind: evt1ResultProvenanceUnknown}
}

func evt1ParameterResultProvenance(index int) evt1ResultProvenanceSummary {
	return evt1ResultProvenanceSummary{Kind: evt1ResultProvenanceParameter, ParameterIndices: []int{index}}
}

func evt1NormalizeResultProvenance(indices []int) evt1ResultProvenanceSummary {
	if len(indices) == 0 {
		return evt1ResultProvenanceSummary{Kind: evt1ResultProvenanceStatic}
	}
	seen := map[int]bool{}
	unique := make([]int, 0, len(indices))
	for _, index := range indices {
		if !seen[index] {
			seen[index] = true
			unique = append(unique, index)
		}
	}
	sort.Ints(unique)
	kind := evt1ResultProvenanceParameter
	if len(unique) > 1 {
		kind = evt1ResultProvenanceShortestOf
	}
	return evt1ResultProvenanceSummary{Kind: kind, ParameterIndices: unique}
}

func evt1CombineResultProvenance(parts ...evt1ResultProvenanceSummary) evt1ResultProvenanceSummary {
	var indices []int
	for _, part := range parts {
		if part.Kind == evt1ResultProvenanceUnknown {
			return evt1UnknownResultProvenance()
		}
		indices = append(indices, part.ParameterIndices...)
	}
	return evt1NormalizeResultProvenance(indices)
}

func evt1SameResultProvenance(left, right evt1ResultProvenanceSummary) bool {
	if left.Kind != right.Kind || len(left.ParameterIndices) != len(right.ParameterIndices) {
		return false
	}
	for i := range left.ParameterIndices {
		if left.ParameterIndices[i] != right.ParameterIndices[i] {
			return false
		}
	}
	return true
}

// evt1DeriveResultProvenanceSummaries performs a bounded, syntax-directed
// summary pass. Calls are followed on demand, while recursive cycles and
// non-straight-line returns remain Unknown rather than starting region solving.
func evt1DeriveResultProvenanceSummaries(env *semanticEnv, functions []FunctionDecl) {
	state := map[string]uint8{}
	var derive func(FunctionDecl) evt1ResultProvenanceSummary
	derive = func(fn FunctionDecl) evt1ResultProvenanceSummary {
		key := evt1FunctionProvenanceKey(fn)
		if state[key] == 2 {
			return env.resultProvenance[key]
		}
		if state[key] == 1 {
			return evt1UnknownResultProvenance()
		}
		if fn.Body == nil || (!fn.ReturnType.isReference() && !evt1IsRefStructType(env, fn.ReturnType)) {
			summary := evt1UnknownResultProvenance()
			env.resultProvenance[key] = summary
			state[key] = 2
			return summary
		}
		state[key] = 1
		bindings := map[string]evt1ResultProvenanceSummary{}
		for i, param := range fn.Params {
			if param.Type.isReference() || evt1IsRefStructType(env, param.Type) {
				bindings[param.Name] = evt1ParameterResultProvenance(i)
			} else {
				bindings[param.Name] = evt1UnknownResultProvenance()
			}
		}
		summary, found := evt1DeriveBlockResultProvenance(env, *fn.Body, bindings, derive)
		if !found {
			summary = evt1UnknownResultProvenance()
		}
		env.resultProvenance[key] = summary
		state[key] = 2
		return summary
	}
	for _, fn := range functions {
		derive(fn)
	}
}

func evt1DeriveBlockResultProvenance(env *semanticEnv, block Block, inherited map[string]evt1ResultProvenanceSummary, derive func(FunctionDecl) evt1ResultProvenanceSummary) (evt1ResultProvenanceSummary, bool) {
	bindings := make(map[string]evt1ResultProvenanceSummary, len(inherited))
	for name, summary := range inherited {
		bindings[name] = summary
	}
	var result evt1ResultProvenanceSummary
	found := false
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			if s.Type.isReference() || evt1IsRefStructType(env, s.Type) {
				bindings[s.Name] = evt1DeriveExprResultProvenance(env, s.Value, bindings, derive)
			}
		case *AssignStmt:
			if name, ok := s.Target.(*NameExpr); ok {
				if _, tracked := bindings[name.Name]; tracked {
					bindings[name.Name] = evt1DeriveExprResultProvenance(env, s.Value, bindings, derive)
				}
			}
		case *ReturnStmt:
			if s.Value == nil {
				continue
			}
			candidate := evt1DeriveExprResultProvenance(env, s.Value, bindings, derive)
			if found && !evt1SameResultProvenance(result, candidate) {
				return evt1UnknownResultProvenance(), true
			}
			result, found = candidate, true
		case *IfStmt, *MatchStmt, *WhileStmt:
			return evt1UnknownResultProvenance(), true
		}
	}
	return result, found
}

func evt1DeriveExprResultProvenance(env *semanticEnv, expr Expr, bindings map[string]evt1ResultProvenanceSummary, derive func(FunctionDecl) evt1ResultProvenanceSummary) evt1ResultProvenanceSummary {
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1DeriveExprResultProvenance(env, e.Value, bindings, derive)
	case *MoveExpr:
		return evt1DeriveExprResultProvenance(env, e.Value, bindings, derive)
	case *FailureExpr:
		return evt1DeriveExprResultProvenance(env, e.Value, bindings, derive)
	case *NameExpr:
		if summary, ok := bindings[e.Name]; ok {
			return summary
		}
	case *RefExpr:
		return evt1DeriveExprResultProvenance(env, e.Value, bindings, derive)
	case *FieldExpr:
		return evt1DeriveExprResultProvenance(env, e.Receiver, bindings, derive)
	case *StructConstructExpr:
		decl, ok := env.structs[e.StructName]
		if !ok || !decl.Ref {
			break
		}
		parts := make([]evt1ResultProvenanceSummary, 0, len(e.Args))
		for i, arg := range e.Args {
			if i < len(decl.Fields) && (decl.Fields[i].Type.isReference() || evt1IsRefStructType(env, decl.Fields[i].Type)) {
				parts = append(parts, evt1DeriveExprResultProvenance(env, arg, bindings, derive))
			}
		}
		if len(parts) > 0 {
			return evt1CombineResultProvenance(parts...)
		}
	case *ConstructExpr:
		if e.EnumName == "Result" && e.VariantName == "Ok" && len(e.Args) > 0 {
			return evt1DeriveExprResultProvenance(env, e.Args[0], bindings, derive)
		}
	case *CallExpr:
		var candidates []FunctionDecl
		for _, fn := range env.functions[e.Callee] {
			if len(fn.Params) == len(e.Args) {
				candidates = append(candidates, fn)
			}
		}
		if len(candidates) != 1 {
			break
		}
		calleeSummary := derive(candidates[0])
		if calleeSummary.Kind == evt1ResultProvenanceUnknown {
			return calleeSummary
		}
		parts := make([]evt1ResultProvenanceSummary, 0, len(calleeSummary.ParameterIndices))
		for _, index := range calleeSummary.ParameterIndices {
			if index < 0 || index >= len(e.Args) {
				return evt1UnknownResultProvenance()
			}
			parts = append(parts, evt1DeriveExprResultProvenance(env, e.Args[index], bindings, derive))
		}
		return evt1CombineResultProvenance(parts...)
	}
	return evt1UnknownResultProvenance()
}

func evt1ExprProvenance(env *semanticEnv, scope *evt1Scope, expr Expr) evt1LifetimeProvenance {
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1ExprProvenance(env, scope, e.Value)
	case *MoveExpr:
		return evt1ExprProvenance(env, scope, e.Value)
	case *FailureExpr:
		return evt1ExprProvenance(env, scope, e.Value)
	case *NameExpr:
		if binding, ok := scope.lookup(e.Name); ok {
			if binding.provenance.Kind != "" {
				return binding.provenance
			}
			return evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: scope.depth}
		}
	case *RefExpr:
		if lvalue, err := validateAssignable(env, scope, e.Value, nil); err == nil {
			if binding, ok := scope.lookup(lvalue.path.Root); ok {
				if binding.provenance.Kind != "" {
					return binding.provenance
				}
				for owner := scope; owner != nil; owner = owner.parent {
					if _, ok := owner.values[lvalue.path.Root]; ok {
						return evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: owner.depth}
					}
				}
			}
		}
	case *StructConstructExpr:
		if decl, ok := env.structs[e.StructName]; ok && decl.Ref {
			result := evt1LifetimeProvenance{Kind: evt1ProvenanceStatic}
			found := false
			for i, arg := range e.Args {
				if i >= len(decl.Fields) || (!decl.Fields[i].Type.isReference() && !evt1IsRefStructType(env, decl.Fields[i].Type)) {
					continue
				}
				p := evt1ExprProvenance(env, scope, arg)
				if !found || evt1ProvenanceIsShorter(p, result) {
					result = p
				}
				result.Scoped = result.Scoped || p.Scoped
				found = true
			}
			if found {
				return result
			}
		}
	case *ConstructExpr:
		if evt1IsFailureType(e.ResolvedType) && len(e.Args) > 0 && evt1IsRefStructType(env, evt1FailureSuccessType(e.ResolvedType)) {
			return evt1ExprProvenance(env, scope, e.Args[0])
		}
	case *CallExpr:
		argTypes := make([]Type, 0, len(e.Args))
		for _, arg := range e.Args {
			argType, err := validateExpr(env, scope, arg, nil, false)
			if err != nil {
				return evt1LifetimeProvenance{Kind: evt1ProvenanceUnknown, Depth: scope.depth, Scoped: true}
			}
			argTypes = append(argTypes, argType)
		}
		fn, err := evt1ResolveOrdinaryCall(env, scope, e.Callee, e.Args, argTypes, nil, e.Span)
		if err != nil {
			return evt1LifetimeProvenance{Kind: evt1ProvenanceUnknown, Depth: scope.depth, Scoped: true}
		}
		summary := env.resultProvenance[evt1FunctionProvenanceKey(fn)]
		if summary.Kind == evt1ResultProvenanceStatic {
			return evt1LifetimeProvenance{Kind: evt1ProvenanceStatic}
		}
		if summary.Kind == evt1ResultProvenanceParameter || summary.Kind == evt1ResultProvenanceShortestOf {
			result := evt1LifetimeProvenance{Kind: evt1ProvenanceStatic}
			found := false
			for _, index := range summary.ParameterIndices {
				if index < 0 || index >= len(e.Args) {
					return evt1LifetimeProvenance{Kind: evt1ProvenanceUnknown, Depth: scope.depth, Scoped: true}
				}
				p := evt1ExprProvenance(env, scope, e.Args[index])
				p.Scoped = p.Scoped || fn.Params[index].Type.Scoped
				if !found || evt1ProvenanceIsShorter(p, result) {
					result = p
				}
				result.Scoped = result.Scoped || p.Scoped
				found = true
			}
			if found {
				return result
			}
		}
	}
	return evt1LifetimeProvenance{Kind: evt1ProvenanceUnknown, Depth: scope.depth, Scoped: true}
}

func evt1LifetimeShorterThan(source, destination evt1LifetimeProvenance) bool {
	if source.Scoped && !destination.Scoped {
		return true
	}
	if source.Kind == evt1ProvenanceUnknown {
		return true
	}
	if destination.Kind == evt1ProvenanceParameter || destination.Kind == evt1ProvenanceStatic {
		return source.Kind == evt1ProvenanceLocal
	}
	return source.Kind == evt1ProvenanceLocal && source.Depth > destination.Depth
}

func evt1ProvenanceIsShorter(left, right evt1LifetimeProvenance) bool {
	if left.Scoped != right.Scoped {
		return left.Scoped
	}
	if left.Kind == evt1ProvenanceUnknown {
		return true
	}
	if right.Kind == evt1ProvenanceUnknown {
		return false
	}
	if left.Kind == evt1ProvenanceLocal {
		if right.Kind != evt1ProvenanceLocal {
			return true
		}
		return left.Depth > right.Depth
	}
	return left.Kind == evt1ProvenanceParameter && right.Kind == evt1ProvenanceStatic
}

func validateMatchExpr(env *semanticEnv, scope *evt1Scope, expr MatchExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	subjectType, enumDecl, err := validateMatchSubject(env, scope, expr.Subject, templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	seen := map[string]bool{}
	var resultType Type
	for index, arm := range expr.Arms {
		armScope, _, err := validatePattern(env, scope, subjectType, enumDecl, arm.Pattern, seen)
		if err != nil {
			return Type{}, err
		}
		for _, binding := range arm.Pattern.Bindings {
			armScope.setProvenance(binding, evt1ExprProvenance(env, scope, expr.Subject))
		}
		valueType, err := validateExpr(env, armScope, arm.Value, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if index == 0 {
			resultType = evt1CanonicalType(env, valueType)
		} else if !evt1CanonicalType(env, resultType).Equal(evt1CanonicalType(env, valueType)) {
			return Type{}, evt1Diagnostic("CV4116", fmt.Sprintf("incompatible expression-arm result types: expected %s but got %s", resultType.String(), valueType.String()), arm.Value.exprSpan())
		}
	}
	if missing := evt1MissingVariants(enumDecl, seen); len(missing) > 0 {
		return Type{}, evt1Diagnostic("CV4115", "non-exhaustive match, missing variants: "+strings.Join(missing, ", "), expr.Span)
	}
	return resultType, nil
}

func validateMatchStmt(env *semanticEnv, scope *evt1Scope, stmt MatchStmt, returnType Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) error {
	subjectType, enumDecl, err := validateMatchSubject(env, scope, stmt.Subject, templateInfo, inComptimeFn)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, arm := range stmt.Arms {
		armScope, _, err := validatePattern(env, scope, subjectType, enumDecl, arm.Pattern, seen)
		if err != nil {
			return err
		}
		for _, binding := range arm.Pattern.Bindings {
			armScope.setProvenance(binding, evt1ExprProvenance(env, scope, stmt.Subject))
		}
		if err := validateBlock(env, armScope, returnType, arm.Block, templateInfo, inComptimeFn); err != nil {
			return err
		}
	}
	if missing := evt1MissingVariants(enumDecl, seen); len(missing) > 0 {
		return evt1Diagnostic("CV4115", "non-exhaustive match, missing variants: "+strings.Join(missing, ", "), stmt.Span)
	}
	return nil
}

func validateMatchSubject(env *semanticEnv, scope *evt1Scope, subject Expr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, EnumDecl, error) {
	subjectType, err := validateExpr(env, scope, subject, templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, EnumDecl{}, err
	}
	if !evt1TypeCopyable(env, subjectType) {
		if _, moving := subject.(*MoveExpr); !moving {
			return Type{}, EnumDecl{}, evt1Diagnostic("CV4548", fmt.Sprintf("matching %s by value requires explicit move", subjectType.String()), subject.exprSpan())
		}
	}
	if decl, ok := evt1FailureEnumDecl(subjectType); ok {
		return subjectType, decl, nil
	}
	if subjectType.Kind != TypeEnum {
		enumDecl, ok := env.enums[subjectType.Name]
		if !ok {
			return Type{}, EnumDecl{}, evt1Diagnostic("CV4108", fmt.Sprintf("match subject must be an enum, got %s", subjectType.String()), subject.exprSpan())
		}
		return subjectType, enumDecl, nil
	}
	return subjectType, env.enums[subjectType.Name], nil
}

func validatePattern(env *semanticEnv, scope *evt1Scope, subjectType Type, enumDecl EnumDecl, pattern Pattern, seen map[string]bool) (*evt1Scope, VariantDecl, error) {
	if pattern.EnumName != enumDecl.Name {
		if _, ok := env.enums[pattern.EnumName]; ok {
			return nil, VariantDecl{}, evt1Diagnostic("CV4109", fmt.Sprintf("match arm pattern uses %s on subject of type %s", pattern.EnumName, subjectType.Name), pattern.Span)
		}
		return nil, VariantDecl{}, evt1Diagnostic("CV4102", fmt.Sprintf("unknown enum type %s in match arm", pattern.EnumName), pattern.Span)
	}
	variant, ok := evt1LookupVariant(enumDecl, pattern.VariantName)
	if !ok {
		return nil, VariantDecl{}, evt1Diagnostic("CV4110", fmt.Sprintf("unknown match variant %s::%s", pattern.EnumName, pattern.VariantName), pattern.Span)
	}
	key := pattern.EnumName + "::" + pattern.VariantName
	if seen[key] {
		return nil, VariantDecl{}, evt1Diagnostic("CV4113", fmt.Sprintf("duplicate match arm %s", key), pattern.Span)
	}
	seen[key] = true
	if len(variant.Payload) == 0 && len(pattern.Bindings) > 0 {
		return nil, VariantDecl{}, evt1Diagnostic("CV4112", fmt.Sprintf("unit variant %s::%s cannot bind payload names", pattern.EnumName, pattern.VariantName), pattern.Span)
	}
	if len(variant.Payload) != len(pattern.Bindings) {
		return nil, VariantDecl{}, evt1Diagnostic("CV4111", fmt.Sprintf("payload binding count mismatch for %s::%s: expected %d but got %d", pattern.EnumName, pattern.VariantName, len(variant.Payload), len(pattern.Bindings)), pattern.Span)
	}
	armScope := newEVT1Scope(scope)
	seenBindings := map[string]bool{}
	for i, binding := range pattern.Bindings {
		if seenBindings[binding] {
			return nil, VariantDecl{}, evt1Diagnostic("CV4112", fmt.Sprintf("duplicate payload binding name %s", binding), pattern.Span)
		}
		seenBindings[binding] = true
		payloadType := evt1CanonicalType(env, variant.Payload[i].Type)
		provenance := evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: armScope.depth}
		armScope.declare(binding, evt1ValueBinding{t: payloadType, mutable: true, state: evt1StorageInitialized, provenance: provenance})
	}
	return armScope, variant, nil
}

func validateWhileStmt(env *semanticEnv, scope *evt1Scope, stmt WhileStmt, templateInfo *evt1TemplateInfo, inComptimeFn bool) error {
	conditionType, err := validateExpr(env, scope, stmt.Condition, templateInfo, inComptimeFn)
	if err != nil {
		return err
	}
	if conditionType.Name != "bool" {
		return evt1Diagnostic("CV4187", "while condition must be bool", stmt.Condition.exprSpan())
	}
	if inComptimeFn && stmt.Bound == nil {
		return evt1Diagnostic("CV4205", "comptime while requires an explicit bounded(limit) clause", stmt.Span)
	}
	if stmt.Bound != nil {
		boundType, err := validateExpr(env, scope, stmt.Bound, templateInfo, true)
		if err != nil {
			return err
		}
		if boundType.Name != "int" {
			return evt1Diagnostic("CV4205", "bounded while requires a compile-time int bound", stmt.Bound.exprSpan())
		}
		value, err := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), stmt.Bound)
		if err != nil {
			return err
		}
		if value.Kind != ValueInt || value.IntValue < 0 {
			return evt1Diagnostic("CV4205", "bounded while requires a non-negative compile-time int bound", stmt.Bound.exprSpan())
		}
		if inComptimeFn && value.IntValue > evt1ComptimeMaxLoopBound {
			return evt1Diagnostic("CV4206", fmt.Sprintf("comptime loop bound %d exceeds limit %d", value.IntValue, evt1ComptimeMaxLoopBound), stmt.Bound.exprSpan())
		}
	}
	bodyScope := evt1CloneScope(scope)
	if err := validateBlock(env, bodyScope, Type{Name: "void", Kind: TypeBuiltin}, stmt.Body, templateInfo, inComptimeFn); err != nil {
		return err
	}
	beforeScope := evt1CloneScope(scope)
	evt1MergeScopeStates(scope, beforeScope, bodyScope)
	return nil
}

const (
	evt1AutomataMaxGuardExprNodes      = 128
	evt1AutomataMaxGuardCallDepth      = 8
	evt1AutomataMaxGuardCallGraphNodes = 16
	evt1AutomataMaxGuardCallGraphEdges = 32
)

type evt1GuardCheckState struct {
	checked  map[string]bool
	visiting map[string]bool
	nodes    int
	edges    int
}

func evt1ValidateAutomataGuard(env *semanticEnv, info *evt1AutomataInfo, expr Expr) error {
	scope := evt1ModuleScope(env)
	if info.Decl.Context != nil {
		contextType := info.Decl.Context.Type
		contextType.Ownership = "borrow"
		contextType.Const = true
		scope.declare(info.Decl.Context.Name, evt1ValueBinding{
			t:        contextType,
			mutable:  false,
			comptime: false,
		})
	}
	guardType, err := validateExpr(env, scope, expr, nil, false)
	if err != nil {
		return err
	}
	if guardType.Name != "bool" {
		return evt1Diagnostic("CV4290", fmt.Sprintf("guard expression must have exact type bool, got %s", guardType.String()), expr.exprSpan())
	}
	if nodes := evt1GuardExprNodeCount(expr); nodes > evt1AutomataMaxGuardExprNodes {
		return evt1Diagnostic("CV4297", fmt.Sprintf("guard expression node count %d exceeds limit %d", nodes, evt1AutomataMaxGuardExprNodes), expr.exprSpan())
	}
	state := &evt1GuardCheckState{
		checked:  map[string]bool{},
		visiting: map[string]bool{},
	}
	return evt1ValidateGuardExpr(env, scope, expr, state, 0)
}

func evt1ValidateGuardExpr(env *semanticEnv, scope *evt1Scope, expr Expr, state *evt1GuardCheckState, depth int) error {
	switch e := expr.(type) {
	case *NameExpr, *IntLiteral, *StringLiteral, *BoolLiteral:
		return nil
	case *FieldExpr:
		return evt1ValidateGuardExpr(env, scope, e.Receiver, state, depth)
	case *ParenExpr:
		return evt1ValidateGuardExpr(env, scope, e.Value, state, depth)
	case *UnaryExpr:
		return evt1ValidateGuardExpr(env, scope, e.Value, state, depth)
	case *BinaryExpr:
		if err := evt1ValidateGuardExpr(env, scope, e.Left, state, depth); err != nil {
			return err
		}
		return evt1ValidateGuardExpr(env, scope, e.Right, state, depth)
	case *IfExpr:
		if err := evt1ValidateGuardExpr(env, scope, e.Condition, state, depth); err != nil {
			return err
		}
		if err := evt1ValidateGuardExpr(env, scope, e.Then, state, depth); err != nil {
			return err
		}
		return evt1ValidateGuardExpr(env, scope, e.Else, state, depth)
	case *MatchExpr:
		if err := evt1ValidateGuardExpr(env, scope, e.Subject, state, depth); err != nil {
			return err
		}
		for _, arm := range e.Arms {
			if err := evt1ValidateGuardExpr(env, scope, arm.Value, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *ConstructExpr:
		for _, arg := range e.Args {
			if err := evt1ValidateGuardExpr(env, scope, arg, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *StructConstructExpr:
		for _, arg := range e.Args {
			if err := evt1ValidateGuardExpr(env, scope, arg, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *WithExpr:
		if err := evt1ValidateGuardExpr(env, scope, e.Base, state, depth); err != nil {
			return err
		}
		for _, update := range e.Updates {
			if err := evt1ValidateGuardExpr(env, scope, update.Value, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *ArrayLiteralExpr:
		for _, element := range e.Elements {
			if err := evt1ValidateGuardExpr(env, scope, element, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *IndexExpr:
		if err := evt1ValidateGuardExpr(env, scope, e.Base, state, depth); err != nil {
			return err
		}
		for _, index := range evt1StorageIndices(e) {
			if err := evt1ValidateGuardExpr(env, scope, index, state, depth); err != nil {
				return err
			}
		}
		return nil
	case *DispatchExpr:
		return evt1Diagnostic("CV4292", "dispatch is not allowed in automata guards", e.Span)
	case *TemplateCallExpr:
		return evt1Diagnostic("CV4293", "template calls are not allowed in automata guards", e.Span)
	case *CallExpr:
		if e.Callee == "Len" {
			for _, arg := range e.Args {
				if err := evt1ValidateGuardExpr(env, scope, arg, state, depth); err != nil {
					return err
				}
			}
			return nil
		}
		argTypes := make([]Type, 0, len(e.Args))
		for _, arg := range e.Args {
			if err := evt1ValidateGuardExpr(env, scope, arg, state, depth); err != nil {
				return err
			}
			argType, err := validateExpr(env, scope, arg, nil, false)
			if err != nil {
				return err
			}
			argTypes = append(argTypes, argType)
		}
		fn, err := evt1ResolveOrdinaryCall(env, scope, e.Callee, e.Args, argTypes, nil, e.Span)
		if err != nil {
			return err
		}
		return evt1ValidateGuardFunction(env, fn, state, depth+1)
	default:
		return evt1Diagnostic("CV4294", "unsupported guard expression form", expr.exprSpan())
	}
}

func evt1ValidateGuardFunction(env *semanticEnv, fn FunctionDecl, state *evt1GuardCheckState, depth int) error {
	if depth > evt1AutomataMaxGuardCallDepth {
		return evt1Diagnostic("CV4298", fmt.Sprintf("guard call depth %d exceeds limit %d", depth, evt1AutomataMaxGuardCallDepth), fn.Span)
	}
	key := fn.Name + "|" + evt1FunctionParamSignature(fn)
	if state.visiting[key] {
		return evt1Diagnostic("CV4296", "recursive guard call graph is not allowed: "+key, fn.Span)
	}
	if state.checked[key] {
		return nil
	}
	state.nodes++
	if state.nodes > evt1AutomataMaxGuardCallGraphNodes {
		return evt1Diagnostic("CV4298", fmt.Sprintf("guard call graph node count %d exceeds limit %d", state.nodes, evt1AutomataMaxGuardCallGraphNodes), fn.Span)
	}
	if fn.Body == nil {
		return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s requires a local function body so purity can be verified", fn.Name), fn.Span)
	}
	if fn.ReturnType.Name == "void" {
		return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s must return a runtime value", fn.Name), fn.Span)
	}
	for _, param := range fn.Params {
		if param.Type.isOwned() {
			return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot take owned parameter %s", fn.Name, param.Type.String()), param.Span)
		}
		if param.Type.isBorrow() && !param.Type.Const {
			return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot take mutable borrow parameter %s", fn.Name, param.Type.String()), param.Span)
		}
		if param.Type.PointerTo != nil {
			return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot take pointer parameter %s", fn.Name, param.Type.String()), param.Span)
		}
		if !param.Type.isBorrowLike() && !evt1TypeCopyable(env, param.Type) {
			return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot take non-copyable parameter %s", fn.Name, param.Type.String()), param.Span)
		}
	}
	state.visiting[key] = true
	guardScope := newEVT1Scope(evt1ModuleScope(env))
	for _, param := range fn.Params {
		guardScope.declare(param.Name, evt1ValueBinding{
			t:        evt1CanonicalType(env, param.Type),
			mutable:  false,
			comptime: false,
		})
	}
	if err := evt1ValidateGuardFunctionBlock(env, guardScope, *fn.Body, fn, state, depth); err != nil {
		delete(state.visiting, key)
		return err
	}
	delete(state.visiting, key)
	state.checked[key] = true
	return nil
}

func evt1ValidateGuardFunctionBlock(env *semanticEnv, scope *evt1Scope, block Block, fn FunctionDecl, state *evt1GuardCheckState, depth int) error {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			if s.Comptime {
				return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot use comptime locals", fn.Name), s.Span)
			}
			if _, err := validateExpr(env, scope, s.Value, nil, false); err != nil {
				return err
			}
			if err := evt1ValidateGuardExpr(env, scope, s.Value, state, depth); err != nil {
				return err
			}
			resolvedType, err := evt1ResolveType(env, scope, s.Type)
			if err != nil {
				return err
			}
			scope.declare(s.Name, evt1ValueBinding{t: evt1CanonicalType(env, resolvedType), mutable: false})
		case *ReturnStmt:
			if s.Value != nil {
				if _, err := validateExpr(env, scope, s.Value, nil, false); err != nil {
					return err
				}
				if err := evt1ValidateGuardExpr(env, scope, s.Value, state, depth); err != nil {
					return err
				}
			}
		case *ExprStmt:
			if _, err := validateExpr(env, scope, s.Value, nil, false); err != nil {
				return err
			}
			if err := evt1ValidateGuardExpr(env, scope, s.Value, state, depth); err != nil {
				return err
			}
		case *Block:
			child := newEVT1Scope(scope)
			if err := evt1ValidateGuardFunctionBlock(env, child, *s, fn, state, depth); err != nil {
				return err
			}
		default:
			return evt1Diagnostic("CV4295", fmt.Sprintf("guard call target %s cannot use %s", fn.Name, evt1GuardStatementLabel(stmt)), stmt.statementSpan())
		}
	}
	return nil
}

func evt1GuardStatementLabel(stmt Statement) string {
	switch stmt.(type) {
	case *AssignStmt:
		return "assignment"
	case *InstanceDecl:
		return "instance declarations"
	case *MatchStmt:
		return "statement-form match"
	case *WhileStmt:
		return "while loops"
	case *StaticAssertStmt:
		return "static_assert"
	default:
		return "that statement form"
	}
}

func evt1GuardExprNodeCount(expr Expr) int {
	count := 1
	switch e := expr.(type) {
	case *FieldExpr:
		count += evt1GuardExprNodeCount(e.Receiver)
	case *CallExpr:
		for _, arg := range e.Args {
			count += evt1GuardExprNodeCount(arg)
		}
	case *WithExpr:
		count += evt1GuardExprNodeCount(e.Base)
		for _, update := range e.Updates {
			count += evt1GuardExprNodeCount(update.Value)
		}
	case *DispatchExpr:
		count += evt1GuardExprNodeCount(e.Signal)
	case *TemplateCallExpr:
		for _, arg := range e.Args {
			count += evt1GuardExprNodeCount(arg)
		}
	case *BinaryExpr:
		count += evt1GuardExprNodeCount(e.Left)
		count += evt1GuardExprNodeCount(e.Right)
	case *UnaryExpr:
		count += evt1GuardExprNodeCount(e.Value)
	case *ConstructExpr:
		for _, arg := range e.Args {
			count += evt1GuardExprNodeCount(arg)
		}
	case *StructConstructExpr:
		for _, arg := range e.Args {
			count += evt1GuardExprNodeCount(arg)
		}
	case *ArrayLiteralExpr:
		for _, element := range e.Elements {
			count += evt1GuardExprNodeCount(element)
		}
	case *IndexExpr:
		count += evt1GuardExprNodeCount(e.Base)
		for _, index := range evt1StorageIndices(e) {
			count += evt1GuardExprNodeCount(index)
		}
	case *MatchExpr:
		count += evt1GuardExprNodeCount(e.Subject)
		for _, arm := range e.Arms {
			count += evt1GuardExprNodeCount(arm.Value)
		}
	case *IfExpr:
		count += evt1GuardExprNodeCount(e.Condition)
		count += evt1GuardExprNodeCount(e.Then)
		count += evt1GuardExprNodeCount(e.Else)
	case *ParenExpr:
		count += evt1GuardExprNodeCount(e.Value)
	}
	return count
}

func evt1ExprIdentity(expr Expr) string {
	switch e := expr.(type) {
	case *NameExpr:
		return e.Name
	case *IntLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *FieldExpr:
		return evt1ExprIdentity(e.Receiver) + "." + e.Field
	case *CallExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, evt1ExprIdentity(arg))
		}
		return e.Callee + "(" + strings.Join(args, ",") + ")"
	case *DispatchExpr:
		if e.BatchName != "" {
			return "dispatch(" + e.InstanceName + "," + evt1ExprIdentity(e.Signal) + "," + e.BatchName + ")"
		}
		return "dispatch(" + e.InstanceName + "," + evt1ExprIdentity(e.Signal) + ")"
	case *TemplateCallExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, evt1ExprIdentity(arg))
		}
		return e.Callee + "<" + e.TypeArg.String() + ">(" + strings.Join(args, ",") + ")"
	case *BinaryExpr:
		return "(" + evt1ExprIdentity(e.Left) + " " + e.Op + " " + evt1ExprIdentity(e.Right) + ")"
	case *UnaryExpr:
		return "(" + e.Op + " " + evt1ExprIdentity(e.Value) + ")"
	case *ConstructExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, evt1ExprIdentity(arg))
		}
		return e.EnumName + "::" + e.VariantName + "(" + strings.Join(args, ",") + ")"
	case *StructConstructExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, evt1ExprIdentity(arg))
		}
		return e.StructName + "{" + strings.Join(args, ",") + "}"
	case *WithExpr:
		parts := []string{evt1ExprIdentity(e.Base)}
		for _, update := range e.Updates {
			parts = append(parts, update.Name+"="+evt1ExprIdentity(update.Value))
		}
		return "with(" + strings.Join(parts, ",") + ")"
	case *ArrayLiteralExpr:
		var parts []string
		for _, element := range e.Elements {
			parts = append(parts, evt1ExprIdentity(element))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *IndexExpr:
		indices := make([]string, 0, len(evt1StorageIndices(e)))
		for _, index := range evt1StorageIndices(e) {
			indices = append(indices, evt1ExprIdentity(index))
		}
		return evt1ExprIdentity(e.Base) + "[" + strings.Join(indices, ",") + "]"
	case *MatchExpr:
		var arms []string
		for _, arm := range e.Arms {
			arms = append(arms, arm.Pattern.EnumName+"::"+arm.Pattern.VariantName+"=>"+evt1ExprIdentity(arm.Value))
		}
		return "match(" + evt1ExprIdentity(e.Subject) + "){" + strings.Join(arms, ",") + "}"
	case *IfExpr:
		return "if(" + evt1ExprIdentity(e.Condition) + ") " + evt1ExprIdentity(e.Then) + " else " + evt1ExprIdentity(e.Else)
	case *ParenExpr:
		return "(" + evt1ExprIdentity(e.Value) + ")"
	default:
		return "<expr>"
	}
}

func validateValueLayoutCycles(env *semanticEnv) error {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var dfs func(name string, span Span) error
	dfs = func(name string, span Span) error {
		if visiting[name] {
			return evt1Diagnostic("CV4129", fmt.Sprintf("recursive by-value type cycle involving %s", name), span)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		visited[name] = true
		if structDecl, ok := env.structs[name]; ok {
			for _, field := range structDecl.Fields {
				if target, ok := evt1ByValueTypeName(field.Type); ok {
					if err := dfs(target, field.Span); err != nil {
						return err
					}
				}
			}
		}
		if enumDecl, ok := env.enums[name]; ok {
			for _, variant := range enumDecl.Variants {
				for _, field := range variant.Payload {
					if target, ok := evt1ByValueTypeName(field.Type); ok {
						if err := dfs(target, field.Span); err != nil {
							return err
						}
					}
				}
			}
		}
		visiting[name] = false
		return nil
	}
	for _, structDecl := range env.structs {
		if err := dfs(structDecl.Name, structDecl.Span); err != nil {
			return err
		}
	}
	return nil
}

func evt1ByValueTypeName(t Type) (string, bool) {
	if t.ArrayElem != nil {
		return evt1ByValueTypeName(*t.ArrayElem)
	}
	if t.isBorrowLike() {
		return "", false
	}
	if len(t.TypeArgs) > 0 {
		return "", false
	}
	if t.Kind == TypeBuiltin || t.Kind == TypeConceptParam {
		return "", false
	}
	return t.Name, true
}

func evt1TypeCopyable(env *semanticEnv, t Type) bool {
	if t.isBorrowLike() {
		return true
	}
	if t.isOwned() {
		return false
	}
	if t.ArrayElem != nil {
		return evt1TypeCopyable(env, *t.ArrayElem)
	}
	if evt1IsFailureType(t) {
		for _, arg := range t.TypeArgs {
			if !evt1TypeCopyable(env, arg) {
				return false
			}
		}
		return true
	}
	if len(t.TypeArgs) > 0 {
		return false
	}
	if _, ok := env.profile.builtinType(t.Name, t.Span); ok {
		return true
	}
	if cached, ok := env.copyableCache[t.Name]; ok {
		return cached
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		if structDecl.Immovable {
			env.copyableCache[t.Name] = false
			return false
		}
		env.copyableCache[t.Name] = true
		for _, field := range structDecl.Fields {
			if !evt1TypeCopyable(env, field.Type) {
				env.copyableCache[t.Name] = false
				return false
			}
		}
		return true
	}
	if enumDecl, ok := env.enums[t.Name]; ok {
		env.copyableCache[t.Name] = true
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if !evt1TypeCopyable(env, field.Type) {
					env.copyableCache[t.Name] = false
					return false
				}
			}
		}
		return true
	}
	return true
}

func evt1TypeMovable(env *semanticEnv, t Type) bool {
	return !evt1IsImmovableValueType(env, t)
}

func evt1DropFunction(env *semanticEnv, t Type) *FunctionDecl {
	for i := range env.functions["Drop"] {
		fn := &env.functions["Drop"][i]
		if fn.ReturnType.Name != "void" || len(fn.Params) != 1 || !fn.Params[0].Type.isOwned() {
			continue
		}
		if evt1CanonicalType(env, fn.Params[0].Type.valueType()).Equal(evt1CanonicalType(env, t.valueType())) {
			return fn
		}
	}
	return nil
}

func evt1TypeHasDrop(env *semanticEnv, t Type) bool {
	if evt1IsFailureType(t) {
		return evt1FailureNeedsDrop(env, t)
	}
	if t.isOwned() && evt1DropFunction(env, t) != nil {
		return true
	}
	return t.ArrayElem != nil && evt1StorageElementHasDrop(env, *t.ArrayElem)
}

func evt1StorageElementHasDrop(env *semanticEnv, t Type) bool {
	if evt1IsFailureType(t) {
		return evt1FailureNeedsDrop(env, t)
	}
	if evt1DropFunction(env, t) != nil {
		return true
	}
	return t.ArrayElem != nil && evt1StorageElementHasDrop(env, *t.ArrayElem)
}

func evt1IsImmovableValueType(env *semanticEnv, t Type) bool {
	if t.isBorrowLike() {
		return false
	}
	if t.ArrayElem != nil {
		return evt1IsImmovableValueType(env, *t.ArrayElem)
	}
	if evt1IsFailureType(t) {
		for _, arg := range t.TypeArgs {
			if evt1IsImmovableValueType(env, arg) {
				return true
			}
		}
		return false
	}
	if len(t.TypeArgs) > 0 {
		return false
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		return structDecl.Immovable
	}
	return false
}

func evt1CanDirectInitialize(env *semanticEnv, t Type, expr Expr) bool {
	if lit, ok := expr.(*ArrayLiteralExpr); ok {
		return t.ArrayElem != nil && len(lit.Elements) == t.ArrayLength
	}
	construct, ok := expr.(*StructConstructExpr)
	if !ok {
		if e, ok := expr.(*ConstructExpr); ok {
			return evt1IsFailureType(t) && evt1IsFailureType(e.ResolvedType) && t.SameValueType(e.ResolvedType)
		}
		return false
	}
	return construct.StructName == t.Name && evt1LookupStruct(env, t.Name)
}

func evt1CanTransferInitialize(env *semanticEnv, t Type, expr Expr) bool {
	if evt1CanDirectInitialize(env, t, expr) {
		return true
	}
	switch e := expr.(type) {
	case *MoveExpr, *CallExpr, *TemplateCallExpr, *ConstructExpr, *FailureExpr:
		return true
	case *ParenExpr:
		return evt1CanTransferInitialize(env, t, e.Value)
	default:
		return false
	}
}

func evt1LookupStruct(env *semanticEnv, name string) bool {
	_, ok := env.structs[name]
	return ok
}

func validateConceptCycles(env *semanticEnv) error {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var dfs func(name string, span Span) error
	dfs = func(name string, span Span) error {
		if visiting[name] {
			return evt1Diagnostic("CV4162", fmt.Sprintf("concept dependency cycle involving %s", name), span)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		visited[name] = true
		for _, req := range env.concepts[name].Requirements {
			ref, ok := req.(*PrerequisiteRequirement)
			if !ok {
				continue
			}
			if err := dfs(ref.ConceptName, ref.Span); err != nil {
				return err
			}
		}
		visiting[name] = false
		return nil
	}
	for _, conceptDecl := range env.concepts {
		if err := dfs(conceptDecl.Name, conceptDecl.Span); err != nil {
			return err
		}
	}
	return nil
}

func evt1ResolveOrdinaryCall(env *semanticEnv, scope *evt1Scope, name string, args []Expr, argTypes []Type, templateInfo *evt1TemplateInfo, span Span) (FunctionDecl, error) {
	candidates, ok := env.functions[name]
	if !ok || len(candidates) == 0 {
		return FunctionDecl{}, evt1Diagnostic("CV4027", fmt.Sprintf("unknown function %s", name), span)
	}
	var matches []FunctionDecl
	var r3Error error
	for _, fn := range candidates {
		if len(fn.Params) != len(args) {
			continue
		}
		match := true
		for i, arg := range args {
			if err := validateCallArgument(env, scope, fn.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
				if diagnostic, ok := err.(Diagnostic); ok && strings.HasPrefix(diagnostic.Code, "CV45") {
					r3Error = err
				}
				match = false
				break
			}
		}
		if match {
			matches = append(matches, fn)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return FunctionDecl{}, evt1Diagnostic("CV4182", fmt.Sprintf("call %s is ambiguous under exact-signature matching", name), span)
	}
	if r3Error != nil {
		return FunctionDecl{}, r3Error
	}
	return FunctionDecl{}, evt1Diagnostic("CV4107", fmt.Sprintf("no exact call target matched %s", name), span)
}

func checkConceptSatisfaction(env *semanticEnv, conceptName string, concreteType Type, path []string, span Span) error {
	path = append(path, fmt.Sprintf("%s<%s>", conceptName, concreteType.String()))
	conceptDecl := env.concepts[conceptName]
	for _, req := range conceptDecl.Requirements {
		switch r := req.(type) {
		case *PrerequisiteRequirement:
			if err := checkConceptSatisfaction(env, r.ConceptName, concreteType, path, span); err != nil {
				return err
			}
		case *OperationRequirement:
			required := evt1SubstituteRequirement(*r, conceptDecl.TypeParam, concreteType)
			if _, err := evt1LookupRequiredOperation(env, required, span, strings.Join(path, " -> ")); err != nil {
				return err
			}
		case *CompilerAnalysisRequirement:
			analysis := evt1SemanticAnalysisRegistry[r.Analysis]
			proof := MIRSemanticProof{Concept: conceptName, Analysis: r.Analysis, ConcreteType: concreteType.String(), SourceSpan: r.Span}
			if len(r.SubjectArgs) > 0 {
				subjects, err := evt1BindRelationalRequirementSubjects(env, conceptDecl, concreteType, r.SubjectArgs, span)
				if err != nil {
					return err
				}
				outcome, facts := analysis.CheckSubjects(env, subjects)
				proof.Subjects = evt1MIRSemanticSubjects(subjects)
				proof.Outcome = string(outcome)
				proof.Satisfied = outcome == evt1AnalysisProven
				proof.ProvenanceFacts = facts
				proof.ID = evt1SemanticProofID(conceptName, concreteType, r.Analysis, proof.Subjects)
				env.semanticProofs = append(env.semanticProofs, proof)
				if outcome == evt1AnalysisUnknown {
					return evt1Diagnostic("CV4528", fmt.Sprintf("%s requires %s, but result provenance is unknown", strings.Join(path, " -> "), r.Analysis), span)
				}
				if outcome != evt1AnalysisProven {
					return evt1Diagnostic("CV4527", fmt.Sprintf("%s failed relational requirement %s", strings.Join(path, " -> "), r.Analysis), span)
				}
				continue
			}
			args := make([]Type, len(r.TypeArgs))
			for i, arg := range r.TypeArgs {
				args[i] = evt1SubstituteType(arg, conceptDecl.TypeParam, concreteType)
			}
			for _, arg := range args {
				proof.Subjects = append(proof.Subjects, MIRSemanticSubject{Kind: "type", Name: arg.String(), Type: arg.String()})
			}
			proof.ID = evt1SemanticProofID(conceptName, concreteType, r.Analysis, proof.Subjects)
			proof.Satisfied = analysis.CheckTypes(env, args)
			proof.Outcome = string(evt1AnalysisDisproven)
			if proof.Satisfied {
				proof.Outcome = string(evt1AnalysisProven)
			}
			env.semanticProofs = append(env.semanticProofs, proof)
			if !proof.Satisfied {
				return evt1Diagnostic("CV4524", fmt.Sprintf("%s failed compiler analysis requirement %s<%s>", strings.Join(path, " -> "), r.Analysis, concreteType.String()), span)
			}
		}
	}
	return nil
}

type evt1SemanticAnalysis struct {
	TypeArity     int
	SubjectArity  int
	CheckTypes    func(*semanticEnv, []Type) bool
	CheckSubjects func(*semanticEnv, []evt1BoundSemanticSubject) (evt1SemanticAnalysisOutcome, []string)
}

type evt1SemanticAnalysisOutcome string

const (
	evt1AnalysisProven    evt1SemanticAnalysisOutcome = "proven"
	evt1AnalysisDisproven evt1SemanticAnalysisOutcome = "disproven"
	evt1AnalysisUnknown   evt1SemanticAnalysisOutcome = "unknown"
)

type evt1BoundSemanticSubject struct {
	Kind           string
	Name           string
	Function       FunctionDecl
	ParameterIndex int
	Type           Type
}

var evt1SemanticAnalysisRegistry = map[string]evt1SemanticAnalysis{
	"LifetimeSafe": {TypeArity: 1, CheckTypes: func(env *semanticEnv, args []Type) bool {
		return len(args) == 1 && (args[0].isReference() || args[0].Scoped || evt1IsRefStructType(env, args[0]))
	}},
	"NonEscaping": {TypeArity: 1, CheckTypes: func(env *semanticEnv, args []Type) bool {
		return len(args) == 1 && (args[0].Scoped || evt1IsRefStructType(env, args[0]))
	}},
	"Outlives": {SubjectArity: 2, CheckSubjects: func(env *semanticEnv, subjects []evt1BoundSemanticSubject) (evt1SemanticAnalysisOutcome, []string) {
		if len(subjects) != 2 || subjects[0].Kind != "parameter" || subjects[1].Kind != "result" || evt1FunctionProvenanceKey(subjects[0].Function) != evt1FunctionProvenanceKey(subjects[1].Function) {
			return evt1AnalysisDisproven, []string{"Outlives requires a parameter and result from one selected operation"}
		}
		summary := env.resultProvenance[evt1FunctionProvenanceKey(subjects[1].Function)]
		fact := evt1ResultProvenanceFact(subjects[1].Function.Name, summary)
		if summary.Kind == evt1ResultProvenanceUnknown {
			return evt1AnalysisUnknown, []string{fact}
		}
		for _, index := range summary.ParameterIndices {
			if index == subjects[0].ParameterIndex {
				return evt1AnalysisProven, []string{fact}
			}
		}
		return evt1AnalysisDisproven, []string{fact}
	}},
}

func evt1FindRelationalRequirementOperation(conceptDecl ConceptDecl, refs []SemanticSubjectRef, span Span) (*OperationRequirement, error) {
	var matches []*OperationRequirement
	for _, requirement := range conceptDecl.Requirements {
		operation, ok := requirement.(*OperationRequirement)
		if !ok {
			continue
		}
		matched := true
		for _, ref := range refs {
			if ref.Name == "result" {
				continue
			}
			found := false
			for _, param := range operation.Params {
				found = found || param.Name == ref.Name
			}
			matched = matched && found
		}
		if matched {
			matches = append(matches, operation)
		}
	}
	if len(matches) != 1 {
		return nil, evt1Diagnostic("CV4532", "relational requirement subjects must identify exactly one required operation", span)
	}
	return matches[0], nil
}

func evt1BindRelationalRequirementSubjects(env *semanticEnv, conceptDecl ConceptDecl, concreteType Type, refs []SemanticSubjectRef, span Span) ([]evt1BoundSemanticSubject, error) {
	operation, err := evt1FindRelationalRequirementOperation(conceptDecl, refs, span)
	if err != nil {
		return nil, err
	}
	required := evt1SubstituteRequirement(*operation, conceptDecl.TypeParam, concreteType)
	fn, err := evt1LookupRequiredOperation(env, required, span, conceptDecl.Name+"<"+concreteType.String()+">")
	if err != nil {
		return nil, err
	}
	subjects := make([]evt1BoundSemanticSubject, 0, len(refs))
	for _, ref := range refs {
		if ref.Name == "result" {
			subjects = append(subjects, evt1BoundSemanticSubject{Kind: "result", Name: ref.Name, Function: fn, ParameterIndex: -1, Type: fn.ReturnType})
			continue
		}
		index := -1
		for i, param := range operation.Params {
			if param.Name == ref.Name {
				index = i
				break
			}
		}
		if index < 0 || index >= len(fn.Params) {
			return nil, evt1Diagnostic("CV4532", fmt.Sprintf("semantic subject %s is not a parameter of the selected operation", ref.Name), ref.Span)
		}
		subjects = append(subjects, evt1BoundSemanticSubject{Kind: "parameter", Name: ref.Name, Function: fn, ParameterIndex: index, Type: fn.Params[index].Type})
	}
	return subjects, nil
}

func evt1MIRSemanticSubjects(subjects []evt1BoundSemanticSubject) []MIRSemanticSubject {
	result := make([]MIRSemanticSubject, 0, len(subjects))
	for _, subject := range subjects {
		entry := MIRSemanticSubject{Kind: subject.Kind, Name: subject.Name, Function: subject.Function.Name, Type: subject.Type.String()}
		if subject.Kind == "parameter" {
			index := subject.ParameterIndex
			entry.ParameterIndex = &index
		}
		result = append(result, entry)
	}
	return result
}

func evt1SemanticProofID(conceptName string, concreteType Type, analysis string, subjects []MIRSemanticSubject) string {
	parts := []string{conceptName + "<" + concreteType.String() + ">", analysis}
	for _, subject := range subjects {
		label := subject.Function + "." + subject.Kind
		if subject.Kind == "type" {
			label = "type[" + subject.Type + "]"
		}
		if subject.ParameterIndex != nil {
			label += fmt.Sprintf("[%d]", *subject.ParameterIndex)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "|")
}

func evt1ResultProvenanceFact(functionName string, summary evt1ResultProvenanceSummary) string {
	if summary.Kind == evt1ResultProvenanceUnknown {
		return functionName + " result provenance is unknown"
	}
	if summary.Kind == evt1ResultProvenanceStatic {
		return functionName + " result provenance is static"
	}
	indices := make([]string, 0, len(summary.ParameterIndices))
	for _, index := range summary.ParameterIndices {
		indices = append(indices, fmt.Sprintf("parameter[%d]", index))
	}
	return functionName + " result derives_from " + strings.Join(indices, ",")
}

func evt1LookupRequiredOperation(env *semanticEnv, required OperationRequirement, span Span, prefix string) (FunctionDecl, error) {
	candidates := env.functions[required.Name]
	if len(candidates) == 0 {
		return FunctionDecl{}, evt1Diagnostic("CV4153", fmt.Sprintf("%s is missing required operation %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params)), span)
	}
	var exact []FunctionDecl
	for _, fn := range candidates {
		if len(fn.Params) != len(required.Params) {
			continue
		}
		paramsMatch := true
		qualifierMismatch := false
		for i := range fn.Params {
			if !evt1CanonicalType(env, fn.Params[i].Type).Equal(evt1CanonicalType(env, required.Params[i].Type)) {
				if evt1CanonicalType(env, fn.Params[i].Type.valueType()).Equal(evt1CanonicalType(env, required.Params[i].Type.valueType())) {
					qualifierMismatch = true
				}
				paramsMatch = false
				break
			}
		}
		if !paramsMatch {
			if qualifierMismatch {
				return FunctionDecl{}, evt1Diagnostic("CV4155", fmt.Sprintf("%s requires %s but found %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params), evt1FunctionSignature(fn)), span)
			}
			continue
		}
		if !evt1CanonicalType(env, fn.ReturnType).Equal(evt1CanonicalType(env, required.ReturnType)) {
			return FunctionDecl{}, evt1Diagnostic("CV4156", fmt.Sprintf("%s requires %s but found %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params), evt1FunctionSignature(fn)), span)
		}
		exact = append(exact, fn)
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		return FunctionDecl{}, evt1Diagnostic("CV4182", fmt.Sprintf("%s has ambiguous required operation %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params)), span)
	}
	for _, fn := range candidates {
		if len(fn.Params) == len(required.Params) {
			code := "CV4154"
			for i := range fn.Params {
				if i < len(required.Params) && evt1CanonicalType(env, fn.Params[i].Type.valueType()).Equal(evt1CanonicalType(env, required.Params[i].Type.valueType())) && !evt1CanonicalType(env, fn.Params[i].Type).Equal(evt1CanonicalType(env, required.Params[i].Type)) {
					code = "CV4155"
					break
				}
			}
			return FunctionDecl{}, evt1Diagnostic(code, fmt.Sprintf("%s requires %s but found %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params), evt1FunctionSignature(fn)), span)
		}
	}
	return FunctionDecl{}, evt1Diagnostic("CV4153", fmt.Sprintf("%s is missing required operation %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params)), span)
}

func evt1SubstituteRequirement(req OperationRequirement, typeParam string, concreteType Type) OperationRequirement {
	out := req
	out.ReturnType = evt1SubstituteType(req.ReturnType, typeParam, concreteType)
	out.Params = make([]Param, 0, len(req.Params))
	for _, param := range req.Params {
		out.Params = append(out.Params, Param{
			Type: evt1SubstituteType(param.Type, typeParam, concreteType),
			Name: param.Name,
			Span: param.Span,
		})
	}
	return out
}

func evt1SubstituteType(t Type, typeParam string, concreteType Type) Type {
	if t.Kind == TypeConceptParam && t.Name == typeParam {
		out := concreteType
		out.Ownership = t.Ownership
		out.Const = t.Const
		out.Imported = t.Imported
		out.Unsafe = t.Unsafe
		return out
	}
	if t.PointerTo != nil {
		base := evt1SubstituteType(*t.PointerTo, typeParam, concreteType)
		t.PointerTo = &base
		return t
	}
	if t.ArrayElem != nil {
		elem := evt1SubstituteType(*t.ArrayElem, typeParam, concreteType)
		t.ArrayElem = &elem
	}
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1SubstituteType(t.TypeArgs[i], typeParam, concreteType)
	}
	return t
}

func evt1Signature(retType Type, name string, params []Param) string {
	var parts []string
	for _, param := range params {
		parts = append(parts, param.Type.String())
	}
	return fmt.Sprintf("%s %s(%s)", retType.String(), name, strings.Join(parts, ", "))
}

func evt1FunctionSignature(fn FunctionDecl) string {
	return evt1Signature(fn.ReturnType, fn.Name, fn.Params)
}

func evt1FunctionParamSignature(fn FunctionDecl) string {
	var parts []string
	for _, param := range fn.Params {
		parts = append(parts, param.Type.String())
	}
	return fmt.Sprintf("%s(%s)", fn.Name, strings.Join(parts, ", "))
}

func exprLabel(expr Expr) string {
	switch e := expr.(type) {
	case *NameExpr:
		return e.Name
	case *FieldExpr:
		return exprLabel(e.Receiver) + "." + e.Field
	case *IndexExpr:
		return exprLabel(e.Base) + "[index]"
	case *ArrayLiteralExpr:
		return "array_literal"
	case *DispatchExpr:
		return "dispatch(" + e.InstanceName + ", ...)"
	default:
		return "expression"
	}
}

func evt1CanonicalType(env *semanticEnv, t Type) Type {
	if t.PointerTo != nil {
		base := evt1CanonicalType(env, *t.PointerTo)
		t.PointerTo = &base
		return t
	}
	if t.ArrayElem != nil {
		elem := evt1CanonicalType(env, *t.ArrayElem)
		t.ArrayElem = &elem
		if t.StorageKind == StorageNDArray {
			t.Kind = TypeNDArray
		} else {
			t.Kind = TypeArray
		}
		return t
	}
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1CanonicalType(env, t.TypeArgs[i])
	}
	if t.Kind == TypeConceptParam || len(t.TypeArgs) > 0 {
		return t
	}
	if env == nil {
		return t
	}
	if _, ok := env.profile.builtinType(t.Name, t.Span); ok {
		t.Kind = TypeBuiltin
		return t
	}
	if _, ok := env.enums[t.Name]; ok {
		t.Kind = TypeEnum
		return t
	}
	if _, ok := env.structs[t.Name]; ok {
		t.Kind = TypeStruct
		return t
	}
	return t
}

func evt1CanInitializeStoredType(env *semanticEnv, expected Type, actual Type) bool {
	return evt1CanonicalType(env, expected.valueType()).Equal(evt1CanonicalType(env, actual.valueType()))
}

func evt1TypeDependsOnParam(t Type, typeParam string) bool {
	if typeParam == "" {
		return false
	}
	if t.Kind == TypeConceptParam && t.Name == typeParam {
		return true
	}
	if t.PointerTo != nil && evt1TypeDependsOnParam(*t.PointerTo, typeParam) {
		return true
	}
	if t.ArrayElem != nil && evt1TypeDependsOnParam(*t.ArrayElem, typeParam) {
		return true
	}
	for _, arg := range t.TypeArgs {
		if evt1TypeDependsOnParam(arg, typeParam) {
			return true
		}
	}
	return false
}

func evt1TypesCompatible(env *semanticEnv, expected Type, actual Type, typeParam string) bool {
	expected = evt1CanonicalType(env, expected.valueType())
	actual = evt1CanonicalType(env, actual.valueType())
	if !evt1TypeDependsOnParam(expected, typeParam) && !evt1TypeDependsOnParam(actual, typeParam) {
		return expected.Equal(actual)
	}
	return evt1SymbolicTypeEqual(expected, actual, typeParam)
}

func evt1SymbolicTypeEqual(a Type, b Type, typeParam string) bool {
	if a.Kind == TypeConceptParam || b.Kind == TypeConceptParam {
		return a.Kind == b.Kind && a.Name == b.Name
	}
	if (a.PointerTo == nil) != (b.PointerTo == nil) {
		return false
	}
	if a.PointerTo != nil {
		return a.Const == b.Const && evt1SymbolicTypeEqual(*a.PointerTo, *b.PointerTo, typeParam)
	}
	if (a.ArrayElem == nil) != (b.ArrayElem == nil) {
		return false
	}
	if a.ArrayElem != nil {
		return a.ArrayLength == b.ArrayLength && a.StorageKind == b.StorageKind && slices.EqualFunc(a.Shape, b.Shape, func(left, right StorageDimension) bool {
			return left.Extent == right.Extent && left.Runtime == right.Runtime && left.Expression == right.Expression
		}) && evt1SymbolicTypeEqual(*a.ArrayElem, *b.ArrayElem, typeParam)
	}
	if a.Name != b.Name || a.Kind != b.Kind || a.Ownership != b.Ownership || a.Const != b.Const || a.Imported != b.Imported || a.Unsafe != b.Unsafe || len(a.TypeArgs) != len(b.TypeArgs) {
		return false
	}
	for i := range a.TypeArgs {
		if !evt1SymbolicTypeEqual(a.TypeArgs[i], b.TypeArgs[i], typeParam) {
			return false
		}
	}
	return true
}

func buildTemplateInfo(env *semanticEnv, templateDecl TemplateDecl) (*evt1TemplateInfo, error) {
	info := &evt1TemplateInfo{
		Decl:         templateDecl,
		CallBindings: map[string]evt1TemplateCallBinding{},
	}
	seenConcepts := map[string]bool{}
	seenRequirements := map[string]bool{}
	typeParamType := Type{Name: templateDecl.TypeParam, Kind: TypeConceptParam, Span: templateDecl.TypeParamSpan}
	var walk func(conceptName string, path []string) error
	walk = func(conceptName string, path []string) error {
		if !seenConcepts[conceptName] {
			info.Closure = append(info.Closure, evt1TemplateClosureEntry{
				Concept: conceptName,
				Path:    append([]string{}, path...),
			})
			seenConcepts[conceptName] = true
		}
		conceptDecl := env.concepts[conceptName]
		for _, rawReq := range conceptDecl.Requirements {
			switch req := rawReq.(type) {
			case *PrerequisiteRequirement:
				nextPath := append(append([]string{}, path...), req.ConceptName)
				if err := walk(req.ConceptName, nextPath); err != nil {
					return err
				}
			case *OperationRequirement:
				substituted := evt1SubstituteRequirement(*req, conceptDecl.TypeParam, typeParamType)
				key := evt1RequirementKey(substituted)
				if seenRequirements[key] {
					continue
				}
				seenRequirements[key] = true
				info.Requirements = append(info.Requirements, evt1TemplateRequirement{
					ID:        fmt.Sprintf("%s.req.%02d", templateDecl.Name, len(info.Requirements)+1),
					Path:      append([]string{}, path...),
					Concept:   conceptName,
					Operation: substituted,
				})
			}
		}
		return nil
	}
	rootPath := []string{templateDecl.Constraint.ConceptName}
	if err := walk(templateDecl.Constraint.ConceptName, rootPath); err != nil {
		return nil, err
	}
	return info, nil
}

func evt1RequirementKey(req OperationRequirement) string {
	return evt1Signature(req.ReturnType, req.Name, req.Params)
}

func evt1SpanKey(span Span) string {
	return fmt.Sprintf("%d:%d", span.Line, span.Column)
}

func validateTemplateCallExpr(env *semanticEnv, scope *evt1Scope, call CallExpr, templateInfo *evt1TemplateInfo) (Type, error) {
	argTypes := make([]Type, 0, len(call.Args))
	dependent := false
	for _, arg := range call.Args {
		argType, err := validateExpr(env, scope, arg, templateInfo, false)
		if err != nil {
			return Type{}, err
		}
		if evt1TypeDependsOnParam(argType, templateInfo.Decl.TypeParam) {
			dependent = true
		}
		argTypes = append(argTypes, argType)
	}
	if !dependent {
		fn, err := evt1ResolveOrdinaryCall(env, scope, call.Callee, call.Args, argTypes, templateInfo, call.Span)
		if err != nil {
			if _, exists := env.templates[call.Callee]; exists {
				return Type{}, evt1Diagnostic("CV4174", "templates cannot invoke templates in EVT1 M1B-B", call.Span)
			}
			return Type{}, err
		}
		for i, arg := range call.Args {
			if err := validateCallArgument(env, scope, fn.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
				return Type{}, err
			}
		}
		return evt1CanonicalType(env, fn.ReturnType), nil
	}
	var matches []evt1TemplateRequirement
	for _, req := range templateInfo.Requirements {
		if req.Operation.Name != call.Callee || len(req.Operation.Params) != len(call.Args) {
			continue
		}
		ok := true
		for i, arg := range call.Args {
			if err := validateCallArgument(env, scope, req.Operation.Params[i].Type, arg, argTypes[i], templateInfo); err != nil {
				ok = false
				break
			}
		}
		if ok {
			matches = append(matches, req)
		}
	}
	if len(matches) == 0 {
		return Type{}, evt1Diagnostic("CV4176", fmt.Sprintf("template body call %s is not guaranteed by constraint %s", call.Callee, templateInfo.Decl.Constraint.ConceptName), call.Span)
	}
	if len(matches) > 1 {
		return Type{}, evt1Diagnostic("CV4177", fmt.Sprintf("template body call %s is ambiguously guaranteed by constraint %s", call.Callee, templateInfo.Decl.Constraint.ConceptName), call.Span)
	}
	templateInfo.CallBindings[evt1SpanKey(call.Span)] = evt1TemplateCallBinding{
		CallSpan:    call.Span,
		Requirement: matches[0],
	}
	return evt1CanonicalType(env, matches[0].Operation.ReturnType), nil
}

func instantiateTemplate(env *semanticEnv, templateName string, concreteType Type, span Span) (*evt1TemplateInstance, error) {
	templateDecl, ok := env.templates[templateName]
	if !ok {
		return nil, evt1Diagnostic("CV4178", fmt.Sprintf("unknown template %s", templateName), span)
	}
	if err := validateTemplateTypeArgument(env, concreteType, span); err != nil {
		return nil, err
	}
	concreteType = evt1CanonicalType(env, concreteType)
	key := templateName + "|" + evt1TypeIdentity(concreteType)
	if instance, ok := env.templateInstances[key]; ok {
		return instance, nil
	}
	if err := checkConceptSatisfaction(env, templateDecl.Constraint.ConceptName, concreteType, nil, span); err != nil {
		return nil, err
	}
	info := env.templateInfos[templateName]
	var bindings []evt1InstanceRequirementBinding
	for _, req := range info.Requirements {
		concreteReq := evt1SubstituteRequirement(req.Operation, templateDecl.TypeParam, concreteType)
		fn, err := evt1LookupRequiredOperation(env, concreteReq, span, templateName+"<"+concreteType.String()+">")
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, evt1InstanceRequirementBinding{
			Requirement: req,
			Function:    fn,
		})
	}
	instFn, err := evt1InstantiateTemplateFunction(templateDecl, concreteType)
	if err != nil {
		return nil, err
	}
	if err := validateFunctionSignature(env, instFn); err != nil {
		return nil, err
	}
	scope := newEVT1Scope(nil)
	scope.returnType = instFn.ReturnType
	for paramIndex, param := range instFn.Params {
		scope.declare(param.Name, evt1ValueBinding{
			t:          evt1CanonicalType(env, param.Type),
			mutable:    !param.Type.Const,
			state:      evt1StorageInitialized,
			provenance: evt1InitialParameterProvenance(env, param.Type, paramIndex, scope.depth),
		})
	}
	if err := validateBlock(env, scope, instFn.ReturnType, *instFn.Body, nil, false); err != nil {
		return nil, err
	}
	instance := &evt1TemplateInstance{
		Key:                 key,
		TemplateName:        templateName,
		ConcreteType:        concreteType,
		TypeIdentity:        evt1TypeIdentity(concreteType),
		GeneratedSymbol:     evt1TemplateInstanceSymbol(templateName, concreteType),
		ConstraintConcept:   templateDecl.Constraint.ConceptName,
		Closure:             append([]evt1TemplateClosureEntry{}, info.Closure...),
		RequirementBindings: bindings,
		Function:            instFn,
		SourceSpan:          templateDecl.Span,
	}
	env.templateInstances[key] = instance
	return instance, nil
}

func validateTemplateTypeArgument(env *semanticEnv, concreteType Type, span Span) error {
	if concreteType.PointerTo != nil || concreteType.ArrayElem != nil || concreteType.Ownership != "" || concreteType.Const || concreteType.Imported || concreteType.Unsafe || len(concreteType.TypeArgs) > 0 || concreteType.Kind == TypeConceptParam {
		return evt1Diagnostic("CV4179", "template calls require one concrete non-template type argument", span)
	}
	return validateKnownType(env, concreteType, span, "", false)
}

func evt1TypeIdentity(t Type) string {
	if t.PointerTo != nil {
		return "ptr_" + evt1TypeIdentity(*t.PointerTo)
	}
	if t.ArrayElem != nil {
		shape := make([]string, 0, len(t.Shape))
		for _, dimension := range t.Shape {
			if dimension.Runtime {
				shape = append(shape, "runtime")
			} else {
				shape = append(shape, fmt.Sprintf("%d", dimension.Extent))
			}
		}
		kind := t.StorageKind
		if kind == "" {
			kind = StorageArray
		}
		return string(kind) + "_" + strings.Join(shape, "_x_") + "_" + evt1TypeIdentity(*t.ArrayElem)
	}
	if len(t.TypeArgs) > 0 {
		var parts []string
		for _, arg := range t.TypeArgs {
			parts = append(parts, evt1TypeIdentity(arg))
		}
		return evt1CName(t.Name)[len("concept_"):] + "_" + strings.Join(parts, "_")
	}
	return evt1CName(t.Name)[len("concept_"):]
}

func evt1TemplateInstanceSymbol(templateName string, concreteType Type) string {
	return "concept_template_" + evt1CName(templateName)[len("concept_"):] + "__" + evt1TypeIdentity(concreteType)
}

func evt1InstantiateTemplateFunction(templateDecl TemplateDecl, concreteType Type) (FunctionDecl, error) {
	body, err := evt1SubstituteBlock(*templateDecl.Body, templateDecl.TypeParam, concreteType)
	if err != nil {
		return FunctionDecl{}, err
	}
	fn := FunctionDecl{
		Name:       templateDecl.Name,
		ReturnType: evt1SubstituteType(templateDecl.ReturnType, templateDecl.TypeParam, concreteType),
		Span:       templateDecl.Span,
		Body:       &body,
	}
	for _, param := range templateDecl.Params {
		fn.Params = append(fn.Params, Param{
			Type: evt1SubstituteType(param.Type, templateDecl.TypeParam, concreteType),
			Name: param.Name,
			Span: param.Span,
		})
	}
	return fn, nil
}

func evt1SubstituteBlock(block Block, typeParam string, concreteType Type) (Block, error) {
	out := Block{Span: block.Span}
	for _, stmt := range block.Statements {
		sub, err := evt1SubstituteStatement(stmt, typeParam, concreteType)
		if err != nil {
			return Block{}, err
		}
		out.Statements = append(out.Statements, sub)
	}
	return out, nil
}

func evt1SubstituteStatement(stmt Statement, typeParam string, concreteType Type) (Statement, error) {
	switch s := stmt.(type) {
	case *VarDecl:
		value, err := evt1SubstituteExpr(s.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &VarDecl{Type: evt1SubstituteType(s.Type, typeParam, concreteType), Name: s.Name, Value: value, Span: s.Span}, nil
	case *EffectsDecl:
		return &EffectsDecl{AutomataName: s.AutomataName, Name: s.Name, Span: s.Span}, nil
	case *InstanceDecl:
		return &InstanceDecl{AutomataName: s.AutomataName, Name: s.Name, Span: s.Span}, nil
	case *AssignStmt:
		target, err := evt1SubstituteExpr(s.Target, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		value, err := evt1SubstituteExpr(s.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &AssignStmt{Target: target, Value: value, Span: s.Span}, nil
	case *ReturnStmt:
		if s.Value == nil {
			return &ReturnStmt{Span: s.Span}, nil
		}
		value, err := evt1SubstituteExpr(s.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{Value: value, Span: s.Span}, nil
	case *ExprStmt:
		value, err := evt1SubstituteExpr(s.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Value: value, Span: s.Span}, nil
	case *MatchStmt:
		subject, err := evt1SubstituteExpr(s.Subject, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		out := &MatchStmt{Subject: subject, Span: s.Span}
		for _, arm := range s.Arms {
			block, err := evt1SubstituteBlock(arm.Block, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Arms = append(out.Arms, StatementArm{Pattern: arm.Pattern, Block: block, Span: arm.Span})
		}
		return out, nil
	case *Block:
		block, err := evt1SubstituteBlock(*s, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &block, nil
	case *IfStmt:
		condition, err := evt1SubstituteExpr(s.Condition, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		thenBlock, err := evt1SubstituteBlock(s.Then, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		out := &IfStmt{Condition: condition, Then: thenBlock, Span: s.Span}
		if s.Else != nil {
			elseBlock, err := evt1SubstituteBlock(*s.Else, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Else = &elseBlock
		}
		return out, nil
	default:
		return nil, evt1Diagnostic("CV4180", "unsupported template statement during instantiation", stmt.statementSpan())
	}
}

func evt1SubstituteExpr(expr Expr, typeParam string, concreteType Type) (Expr, error) {
	switch e := expr.(type) {
	case *NameExpr:
		return &NameExpr{Name: e.Name, Span: e.Span}, nil
	case *IntLiteral:
		return &IntLiteral{Value: e.Value, Span: e.Span}, nil
	case *BoolLiteral:
		return &BoolLiteral{Value: e.Value, Span: e.Span}, nil
	case *FieldExpr:
		receiver, err := evt1SubstituteExpr(e.Receiver, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &FieldExpr{Receiver: receiver, Field: e.Field, Span: e.Span}, nil
	case *CallExpr:
		out := &CallExpr{Callee: e.Callee, Span: e.Span}
		for _, arg := range e.Args {
			sub, err := evt1SubstituteExpr(arg, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Args = append(out.Args, sub)
		}
		return out, nil
	case *DispatchExpr:
		signal, err := evt1SubstituteExpr(e.Signal, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &DispatchExpr{InstanceName: e.InstanceName, Signal: signal, BatchName: e.BatchName, Span: e.Span}, nil
	case *TemplateCallExpr:
		return nil, evt1Diagnostic("CV4174", "templates cannot invoke templates in EVT1 M1B-B", e.Span)
	case *BinaryExpr:
		left, err := evt1SubstituteExpr(e.Left, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		right, err := evt1SubstituteExpr(e.Right, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Op: e.Op, Left: left, Right: right, Span: e.Span}, nil
	case *ConstructExpr:
		out := &ConstructExpr{EnumName: e.EnumName, VariantName: e.VariantName, Span: e.Span}
		for _, arg := range e.Args {
			sub, err := evt1SubstituteExpr(arg, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Args = append(out.Args, sub)
		}
		return out, nil
	case *StructConstructExpr:
		out := &StructConstructExpr{StructName: e.StructName, Span: e.Span}
		for _, arg := range e.Args {
			sub, err := evt1SubstituteExpr(arg, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Args = append(out.Args, sub)
		}
		return out, nil
	case *WithExpr:
		base, err := evt1SubstituteExpr(e.Base, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		out := &WithExpr{Base: base, Span: e.Span}
		for _, update := range e.Updates {
			value, err := evt1SubstituteExpr(update.Value, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Updates = append(out.Updates, FieldUpdate{Name: update.Name, NameSpan: update.NameSpan, Value: value})
		}
		return out, nil
	case *MoveExpr:
		value, err := evt1SubstituteExpr(e.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &MoveExpr{Value: value, Span: e.Span}, nil
	case *RefExpr:
		value, err := evt1SubstituteExpr(e.Value, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		return &RefExpr{Value: value, Const: e.Const, Span: e.Span}, nil
	case *MatchExpr:
		subject, err := evt1SubstituteExpr(e.Subject, typeParam, concreteType)
		if err != nil {
			return nil, err
		}
		out := &MatchExpr{Subject: subject, Span: e.Span}
		for _, arm := range e.Arms {
			value, err := evt1SubstituteExpr(arm.Value, typeParam, concreteType)
			if err != nil {
				return nil, err
			}
			out.Arms = append(out.Arms, ExprArm{Pattern: arm.Pattern, Value: value, Span: arm.Span})
		}
		return out, nil
	default:
		return nil, evt1Diagnostic("CV4181", "unsupported template expression during instantiation", expr.exprSpan())
	}
}
