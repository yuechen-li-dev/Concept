package concept

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type evt1FunctionSymbols struct {
	Prototype string
	Body      string
}

type lowering struct {
	module     Module
	env        *semanticEnv
	outputBase string
	mir        MIR
	mapDoc     map[string]any
}

func Generate(module Module, source []byte) (Outputs, error) {
	env, err := analyzeModule(module)
	if err != nil {
		return nil, err
	}
	l := &lowering{
		module:     module,
		env:        env,
		outputBase: evt1OutputBase(module.Path),
	}
	l.mir = buildMIR(module, env)
	if err := evt1ValidateMIR(l.mir); err != nil {
		return nil, err
	}
	header, body, err := l.generateC()
	if err != nil {
		return nil, err
	}
	mirJSON, err := json.MarshalIndent(l.mir, "", "  ")
	if err != nil {
		return nil, err
	}
	mirJSON = append(mirJSON, '\n')
	l.mapDoc = map[string]any{
		"schema":         "concept-evt1-source-map.v1",
		"source":         module.Path,
		"structs":        l.mir.Structs,
		"actuators":      l.mir.Actuators,
		"automata":       l.mir.Automata,
		"concepts":       l.mir.Concepts,
		"assertions":     l.mir.Assertions,
		"comptime_decls": l.mir.ComptimeDecls,
		"static_asserts": l.mir.StaticAsserts,
		"templates":      l.mir.Templates,
		"instances":      l.mir.Instances,
		"functions":      evt1MapFunctions(module, env),
		"mir":            l.mir.Functions,
	}
	mapJSON, err := json.MarshalIndent(l.mapDoc, "", "  ")
	if err != nil {
		return nil, err
	}
	mapJSON = append(mapJSON, '\n')
	manifest := map[string]any{
		"schema":        "concept-evt1-generation-manifest.v1",
		"compiler":      CompilerID,
		"source":        module.Path,
		"source_sha256": digest(source),
		"profile":       module.Profile,
		"options":       map[string]string{"paths": "repository-relative", "timestamps": "forbidden"},
		"files": []map[string]string{
			{"path": l.outputBase + ".generated.h", "sha256": digest(header)},
			{"path": l.outputBase + ".generated.c", "sha256": digest(body)},
			{"path": l.outputBase + ".mir.json", "sha256": digest(mirJSON)},
			{"path": l.outputBase + ".map.json", "sha256": digest(mapJSON)},
		},
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	manifestJSON = append(manifestJSON, '\n')
	return Outputs{
		l.outputBase + ".generated.h":   header,
		l.outputBase + ".generated.c":   body,
		l.outputBase + ".mir.json":      mirJSON,
		l.outputBase + ".map.json":      mapJSON,
		l.outputBase + ".manifest.json": manifestJSON,
	}, nil
}

func buildMIR(module Module, env *semanticEnv) MIR {
	mir := MIR{
		Schema:  "concept-evt1-mir.v1",
		Module:  module.Path,
		Profile: module.Profile,
	}
	for _, structDecl := range module.Structs {
		mirStruct := MIRStruct{
			Name:       structDecl.Name,
			CName:      evt1CName(structDecl.Name),
			Immovable:  structDecl.Immovable,
			Record:     structDecl.Record,
			Ref:        structDecl.Ref,
			Copyable:   evt1TypeCopyable(env, Type{Name: structDecl.Name, Kind: TypeStruct}),
			Movable:    !structDecl.Immovable,
			HasDrop:    evt1DropFunction(env, Type{Name: structDecl.Name, Kind: TypeStruct}) != nil,
			SourceSpan: structDecl.Span,
		}
		for _, field := range structDecl.Fields {
			mirStruct.Fields = append(mirStruct.Fields, MIRName{Name: field.Name, Type: evt1MIRType(env, field.Type)})
		}
		mir.Structs = append(mir.Structs, mirStruct)
	}
	for _, enumDecl := range module.Enums {
		mirEnum := MIREnum{Name: enumDecl.Name, CName: evt1CName(enumDecl.Name), SourceSpan: enumDecl.Span}
		for _, variant := range enumDecl.Variants {
			mirVariant := MIRVariant{
				Name:       variant.Name,
				TagName:    evt1TagName(enumDecl.Name, variant.Name),
				Tag:        variant.Tag,
				SourceSpan: variant.Span,
			}
			for _, payload := range variant.Payload {
				mirVariant.Payload = append(mirVariant.Payload, MIRName{Name: payload.Name, Type: evt1MIRType(env, payload.Type)})
			}
			mirEnum.Variants = append(mirEnum.Variants, mirVariant)
		}
		mir.Enums = append(mir.Enums, mirEnum)
	}
	for _, effectDecl := range module.Effects {
		mirEffect := MIREffect{Name: effectDecl.Name, SourceSpan: effectDecl.Span}
		for _, param := range effectDecl.Params {
			mirEffect.Params = append(mirEffect.Params, MIRName{Name: param.Name, Type: evt1MIRType(env, param.Type)})
		}
		mir.Effects = append(mir.Effects, mirEffect)
	}
	for _, actuatorDecl := range module.Actuators {
		info := env.actuatorInfo[actuatorDecl.Name]
		mirActuator := MIRActuator{
			Name:          actuatorDecl.Name,
			AutomataName:  actuatorDecl.AutomataName,
			MechanismName: actuatorDecl.MechanismName,
			MechanismType: evt1MIRType(env, actuatorDecl.MechanismType),
			ErrorType:     evt1MIRType(env, actuatorDecl.ErrorType),
			Identity:      info.Identity,
			ResultType:    info.ResultTypeName,
			FailureSlot:   info.FailureSlot,
			SourceSpan:    actuatorDecl.Span,
		}
		for _, mapping := range actuatorDecl.Mappings {
			entry := MIRActuatorMapping{
				EffectName:         mapping.EffectName,
				ImplementationName: mapping.ImplementationName,
				SourceSpan:         mapping.Span,
			}
			for _, arg := range mapping.ImplementationArgs {
				entry.ImplementationArgs = append(entry.ImplementationArgs, evt1ExprIdentity(arg))
			}
			mirActuator.Mappings = append(mirActuator.Mappings, entry)
		}
		mir.Actuators = append(mir.Actuators, mirActuator)
	}
	for _, automataDecl := range module.Automata {
		info := env.automataInfo[automataDecl.Name]
		mirAutomata := MIRAutomata{
			Name:                 automataDecl.Name,
			SignalEnum:           info.SignalEnum.Name,
			RootMachine:          info.RootMachine,
			MaxActiveDepth:       info.MaxActiveDepth,
			ContinuationCapacity: info.ContinuationCapacity,
			CompletionStepBound:  info.CompletionStepBound,
			GraphIdentity:        info.GraphIdentity,
			TopologyIdentity:     info.TopologyIdentity,
			GuardIdentity:        info.GuardIdentity,
			EffectIdentity:       info.EffectIdentity,
			RuntimeIdentity:      info.RuntimeIdentity,
			EffectSet:            append([]string{}, info.EffectSet...),
			MaxEffectBatch:       info.MaxEffectBatch,
			SourceSpan:           automataDecl.Span,
		}
		if automataDecl.Context != nil {
			contextType := evt1MIRType(env, automataDecl.Context.Type)
			mirAutomata.ContextName = automataDecl.Context.Name
			mirAutomata.ContextType = &contextType
		}
		for _, machine := range automataDecl.Machines {
			mirMachine := MIRMachine{
				Name:           machine.Name,
				Initial:        machine.Initial,
				RuntimeOrdinal: info.MachineOrdinal[machine.Name],
				Reachable:      info.MachineReachable[machine.Name],
				SourceSpan:     machine.Span,
			}
			for _, state := range machine.States {
				mirState := MIRState{
					Name:           state.Name,
					Initial:        state.Initial,
					Terminal:       state.Terminal,
					RuntimeOrdinal: info.StateOrdinal[machine.Name][state.Name],
					Reachable:      info.StateReachable[machine.Name][state.Name],
					SourceSpan:     state.Span,
				}
				if len(state.Completion) == 1 {
					mirState.Completion = state.Completion[0].Kind
				}
				for _, handler := range state.Handlers {
					entry := MIRTransition{
						Signal:     handler.Signal.EnumName + "::" + handler.Signal.MemberName,
						Kind:       string(handler.Kind),
						SourceSpan: handler.Span,
					}
					if handler.Guard != nil {
						entry.Guard = evt1ExprIdentity(handler.Guard)
					}
					entry.Otherwise = handler.Otherwise
					for _, emit := range handler.Emits {
						mirEmit := MIREmit{Effect: emit.EffectName, SourceSpan: emit.Span}
						for _, arg := range emit.Args {
							mirEmit.Args = append(mirEmit.Args, evt1ExprIdentity(arg))
						}
						entry.Emits = append(entry.Emits, mirEmit)
					}
					if handler.Kind == TransitionGoto {
						entry.TargetState = handler.TargetState.StateName
					} else {
						entry.PushMachine = handler.PushMachine
						entry.ContinuationState = handler.Continuation.StateName
					}
					mirState.Handlers = append(mirState.Handlers, entry)
				}
				mirMachine.States = append(mirMachine.States, mirState)
			}
			mirAutomata.Machines = append(mirAutomata.Machines, mirMachine)
		}
		mir.Automata = append(mir.Automata, mirAutomata)
	}
	for _, conceptDecl := range module.Concepts {
		mirConcept := MIRConcept{Name: conceptDecl.Name, TypeParam: conceptDecl.TypeParam, SourceSpan: conceptDecl.Span}
		for _, req := range conceptDecl.Requirements {
			switch r := req.(type) {
			case *OperationRequirement:
				entry := MIRConceptRequirement{
					Kind:       "operation",
					Name:       r.Name,
					ReturnType: &r.ReturnType,
					SourceSpan: r.Span,
				}
				for _, param := range r.Params {
					entry.Params = append(entry.Params, MIRName{Name: param.Name, Type: param.Type})
				}
				mirConcept.Requirements = append(mirConcept.Requirements, entry)
			case *PrerequisiteRequirement:
				mirConcept.Requirements = append(mirConcept.Requirements, MIRConceptRequirement{
					Kind:       "prerequisite",
					Name:       r.ConceptName,
					Detail:     r.TypeArg.String(),
					SourceSpan: r.Span,
				})
			case *CompilerAnalysisRequirement:
				var args []string
				for _, arg := range r.TypeArgs {
					args = append(args, arg.String())
				}
				for _, subject := range r.SubjectArgs {
					args = append(args, subject.Name)
				}
				mirConcept.Requirements = append(mirConcept.Requirements, MIRConceptRequirement{
					Kind: "compiler_analysis", Name: r.Analysis, Detail: strings.Join(args, ", "), SourceSpan: r.Span,
				})
			}
		}
		mir.Concepts = append(mir.Concepts, mirConcept)
	}
	for _, assertion := range module.Assertions {
		mir.Assertions = append(mir.Assertions, MIRAssertion{
			ConceptName:  assertion.ConceptName,
			ConcreteType: assertion.ConcreteType,
			Satisfied:    true,
			SourceSpan:   assertion.Span,
		})
	}
	for _, decl := range module.ComptimeDecls {
		mir.ComptimeDecls = append(mir.ComptimeDecls, MIRComptimeDecl{
			Name:       decl.Name,
			Type:       evt1MIRType(env, decl.Type),
			Value:      env.comptimeValues[decl.Name].Render(),
			SourceSpan: decl.Span,
		})
	}
	for _, assertion := range module.StaticAsserts {
		message := ""
		if assertion.Message != nil {
			if value, err := evt1EvalExpr(newEVT1ComptimeState(env), evt1SeedComptimeScope(env), assertion.Message); err == nil {
				message = value.Render()
			}
		}
		mir.StaticAsserts = append(mir.StaticAsserts, MIRStaticAssert{
			Condition:  exprLabel(assertion.Condition),
			Message:    message,
			Satisfied:  true,
			SourceSpan: assertion.Span,
		})
	}
	for _, templateDecl := range module.Templates {
		info := env.templateInfos[templateDecl.Name]
		mirTemplate := MIRTemplate{
			Name:      templateDecl.Name,
			TypeParam: templateDecl.TypeParam,
			Constraint: MIRTemplateConstraint{
				ConceptName: templateDecl.Constraint.ConceptName,
				TypeParam:   templateDecl.TypeParam,
				SourceSpan:  templateDecl.Constraint.Span,
			},
			ReturnType: templateDecl.ReturnType,
			SourceSpan: templateDecl.Span,
		}
		for _, entry := range info.Closure {
			mirTemplate.Closure = append(mirTemplate.Closure, MIRClosureEntry{
				ConceptName: entry.Concept,
				Path:        append([]string{}, entry.Path...),
			})
		}
		for _, req := range info.Requirements {
			mirTemplate.Requirements = append(mirTemplate.Requirements, MIRRequirementBinding{
				RequirementID: req.ID,
				ConceptName:   req.Concept,
				Name:          req.Operation.Name,
				Signature:     evt1Signature(req.Operation.ReturnType, req.Operation.Name, req.Operation.Params),
				Path:          append([]string{}, req.Path...),
			})
		}
		for _, param := range templateDecl.Params {
			mirTemplate.Params = append(mirTemplate.Params, MIRName{Name: param.Name, Type: evt1MIRType(env, param.Type)})
		}
		if templateDecl.Body != nil {
			tmpFn := MIRFunction{Name: templateDecl.Name}
			collectMIROps(env, templateDecl.Body, &tmpFn, info)
			mirTemplate.Operations = append(mirTemplate.Operations, tmpFn.Operations...)
		}
		mir.Templates = append(mir.Templates, mirTemplate)
	}
	for _, templateDecl := range module.Templates {
		var instances []*evt1TemplateInstance
		for _, instance := range env.templateInstances {
			if instance.TemplateName == templateDecl.Name {
				instances = append(instances, instance)
			}
		}
		sort.Slice(instances, func(i, j int) bool {
			return instances[i].TypeIdentity < instances[j].TypeIdentity
		})
		for _, instance := range instances {
			mirInstance := MIRInstance{
				ID:                instance.Key,
				TemplateName:      instance.TemplateName,
				ConcreteType:      instance.ConcreteType,
				GeneratedSymbol:   instance.GeneratedSymbol,
				ConstraintConcept: instance.ConstraintConcept,
				ReturnType:        instance.Function.ReturnType,
				InvocationSpans:   append([]Span{}, instance.InvocationSpans...),
				SourceSpan:        instance.SourceSpan,
			}
			for _, entry := range instance.Closure {
				mirInstance.Closure = append(mirInstance.Closure, MIRClosureEntry{
					ConceptName: entry.Concept,
					Path:        append([]string{}, entry.Path...),
				})
			}
			for _, binding := range instance.RequirementBindings {
				req := binding.Requirement
				concreteReq := evt1SubstituteRequirement(req.Operation, templateDecl.TypeParam, instance.ConcreteType)
				mirInstance.RequirementBindings = append(mirInstance.RequirementBindings, MIRRequirementBinding{
					RequirementID: req.ID,
					ConceptName:   req.Concept,
					Name:          concreteReq.Name,
					Signature:     evt1Signature(concreteReq.ReturnType, concreteReq.Name, concreteReq.Params),
					Path:          append([]string{}, req.Path...),
				})
			}
			for _, param := range instance.Function.Params {
				mirInstance.Params = append(mirInstance.Params, MIRName{Name: param.Name, Type: evt1MIRType(env, param.Type)})
			}
			if instance.Function.Body != nil {
				tmpFn := MIRFunction{Name: instance.GeneratedSymbol}
				collectMIROps(env, instance.Function.Body, &tmpFn, nil)
				mirInstance.Operations = append(mirInstance.Operations, tmpFn.Operations...)
			}
			mir.Instances = append(mir.Instances, mirInstance)
		}
	}
	for _, fn := range module.Functions {
		mirFn := MIRFunction{Name: fn.Name, ReturnType: evt1MIRType(env, fn.ReturnType), SourceSpan: fn.Span}
		if fn.ReturnType.isReference() || evt1IsRefStructType(env, fn.ReturnType) {
			summary := env.resultProvenance[evt1FunctionProvenanceKey(fn)]
			mirFn.ResultProvenance = &MIRResultProvenanceSummary{Kind: string(summary.Kind), ParameterIndices: append([]int{}, summary.ParameterIndices...)}
		}
		for _, param := range fn.Params {
			mirFn.Params = append(mirFn.Params, MIRName{Name: param.Name, Type: evt1MIRType(env, param.Type)})
		}
		if fn.Body != nil {
			collectMIROps(env, fn.Body, &mirFn, nil)
			mirFn.Cleanups = evt1MIRCleanups(env, fn)
		}
		mir.Functions = append(mir.Functions, mirFn)
	}
	for _, fn := range module.ComptimeFns {
		mirFn := MIRFunction{Name: fn.Name, ReturnType: evt1MIRType(env, fn.ReturnType), SourceSpan: fn.Span}
		for _, param := range fn.Params {
			mirFn.Params = append(mirFn.Params, MIRName{Name: param.Name, Type: evt1MIRType(env, param.Type)})
		}
		if fn.Body != nil {
			collectMIROps(env, fn.Body, &mirFn, nil)
		}
		mir.ComptimeFns = append(mir.ComptimeFns, mirFn)
	}
	mir.SemanticProofs = append(mir.SemanticProofs, env.semanticProofs...)
	return mir
}

func evt1ValidateMIR(mir MIR) error {
	for _, fn := range mir.Functions {
		seen := map[string]bool{}
		for i, cleanup := range fn.Cleanups {
			if cleanup.Owner == "" || cleanup.DropFunction == "" || seen[cleanup.Owner] || cleanup.Order != i+1 {
				return evt1Diagnostic("CV4516", fmt.Sprintf("MIR cleanup for %s is not a unique ordered drop obligation", fn.Name), fn.SourceSpan)
			}
			if cleanup.State != "live" && cleanup.State != "transferred" {
				return evt1Diagnostic("CV4516", fmt.Sprintf("MIR cleanup for %s.%s has inconsistent state %s", fn.Name, cleanup.Owner, cleanup.State), fn.SourceSpan)
			}
			seen[cleanup.Owner] = true
		}
	}
	return nil
}

func evt1MIRCleanups(env *semanticEnv, fn FunctionDecl) []MIRCleanup {
	types := map[string]Type{}
	var declarationOrder []string
	state := map[string]string{}
	if fn.Name != "Drop" {
		for _, param := range fn.Params {
			if evt1TypeHasDrop(env, param.Type) {
				types[param.Name] = param.Type
				declarationOrder = append(declarationOrder, param.Name)
				state[param.Name] = "live"
			}
		}
	}
	var visitExpr func(Expr)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *MoveExpr:
			if name, ok := e.Value.(*NameExpr); ok {
				state[name.Name] = "transferred"
			}
		case *CallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				visitExpr(s.Value)
				if evt1TypeHasDrop(env, s.Type) {
					types[s.Name] = s.Type
					declarationOrder = append(declarationOrder, s.Name)
					state[s.Name] = "live"
				}
			case *AssignStmt:
				visitExpr(s.Value)
				if name, ok := s.Target.(*NameExpr); ok && types[name.Name].Name != "" {
					state[name.Name] = "live"
				}
			case *ReturnStmt:
				if s.Value != nil {
					visitExpr(s.Value)
				}
			case *ExprStmt:
				visitExpr(s.Value)
			case *Block:
				visitBlock(*s)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			}
		}
	}
	visitBlock(*fn.Body)
	var cleanups []MIRCleanup
	for i := len(declarationOrder) - 1; i >= 0; i-- {
		name := declarationOrder[i]
		dropName := ""
		if evt1IsFailureType(types[name]) && evt1FailureNeedsDrop(env, types[name]) {
			dropName = evt1FailureDropName(types[name])
		} else if dropFn := evt1DropFunction(env, types[name]); dropFn != nil {
			dropName = dropFn.Name
		}
		cleanups = append(cleanups, MIRCleanup{Owner: name, Type: types[name].String(), DropFunction: dropName, State: state[name], Order: len(cleanups) + 1})
	}
	return cleanups
}

func collectMIROps(env *semanticEnv, block *Block, fn *MIRFunction, templateInfo *evt1TemplateInfo) {
	for _, stmt := range block.Statements {
		id := fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1)
		switch s := stmt.(type) {
		case *VarDecl:
			kind := "var_decl"
			if s.Comptime {
				kind = "comptime_decl"
			} else if s.Const {
				kind = "const_decl"
			}
			if _, ok := s.Value.(*StructConstructExpr); ok {
				if !evt1TypeCopyable(env, s.Type) {
					kind = "final_storage_construct"
				} else {
					kind = "struct_construct"
				}
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Type: evt1MIRType(env, s.Type).String(), Detail: s.Name, SourceSpan: s.Span})
			collectExprMIROps(env, s.Value, fn, templateInfo)
		case *EffectsDecl:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "effects_decl", Detail: s.AutomataName + " " + s.Name, SourceSpan: s.Span})
		case *ActuatorLocalDecl:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "actuator_decl", Detail: s.ActuatorName + " " + s.Name, SourceSpan: s.Span})
			collectExprMIROps(env, s.Mechanism, fn, templateInfo)
		case *InstanceDecl:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "instance_decl", Detail: s.AutomataName + " " + s.Name, SourceSpan: s.Span})
		case *ActuationDecl:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "actuation_decl", Detail: s.ActuatorName + " " + s.Name, SourceSpan: s.Span})
		case *AssignStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "assign", Type: exprLabel(s.Target), Detail: exprLabel(s.Target), SourceSpan: s.Span})
			collectExprMIROps(env, s.Target, fn, templateInfo)
			collectExprMIROps(env, s.Value, fn, templateInfo)
		case *ReturnStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "return", Type: fn.ReturnType.String(), SourceSpan: s.Span})
			if s.Value != nil {
				collectExprMIROps(env, s.Value, fn, templateInfo)
			}
		case *ExprStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "expr_stmt", SourceSpan: s.Span})
			collectExprMIROps(env, s.Value, fn, templateInfo)
		case *AssertStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "assert", Detail: "Assert.True", SourceSpan: s.Span})
			collectExprMIROps(env, s.Condition, fn, templateInfo)
		case *TryStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "try_handler", Detail: fmt.Sprintf("%d exact error arm(s)", len(s.Except)), SourceSpan: s.Span})
			collectMIROps(env, &s.Body, fn, templateInfo)
			for _, arm := range s.Except {
				collectMIROps(env, &arm.Body, fn, templateInfo)
			}
		case *StaticAssertStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "static_assert", SourceSpan: s.Span})
			collectExprMIROps(env, s.Condition, fn, templateInfo)
			if s.Message != nil {
				collectExprMIROps(env, s.Message, fn, templateInfo)
			}
		case *MatchStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "match_stmt", Detail: fmt.Sprintf("%d arms", len(s.Arms)), SourceSpan: s.Span})
			collectExprMIROps(env, s.Subject, fn, templateInfo)
			for _, arm := range s.Arms {
				fn.Operations = append(fn.Operations, MIROperation{
					ID:         fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1),
					Kind:       "pattern",
					Detail:     arm.Pattern.EnumName + "::" + arm.Pattern.VariantName,
					SourceSpan: arm.Pattern.Span,
				})
				collectMIROps(env, &arm.Block, fn, templateInfo)
			}
		case *Block:
			collectMIROps(env, s, fn, templateInfo)
		case *WhileStmt:
			kind := "while"
			if s.Bound != nil {
				kind = "bounded_while"
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, SourceSpan: s.Span})
			collectExprMIROps(env, s.Condition, fn, templateInfo)
			if s.Bound != nil {
				collectExprMIROps(env, s.Bound, fn, templateInfo)
			}
			collectMIROps(env, &s.Body, fn, templateInfo)
		case *IfStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "if_stmt", SourceSpan: s.Span})
			collectExprMIROps(env, s.Condition, fn, templateInfo)
			collectMIROps(env, &s.Then, fn, templateInfo)
			if s.Else != nil {
				collectMIROps(env, s.Else, fn, templateInfo)
			}
		}
	}
}

func collectExprMIROps(env *semanticEnv, expr Expr, fn *MIRFunction, templateInfo *evt1TemplateInfo) {
	id := fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1)
	switch e := expr.(type) {
	case *FailureExpr:
		kind := "result_propagate"
		if evt1IsOptionType(e.ResolvedType) {
			kind = "option_propagate"
		}
		if e.Op == "!" {
			if evt1IsOptionType(e.ResolvedType) {
				kind = "option_unroll"
			} else {
				kind = "result_unroll"
			}
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Type: e.ResolvedType.String(), SourceSpan: e.Span})
		collectExprMIROps(env, e.Value, fn, templateInfo)
	case *MoveExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "move", Detail: exprLabel(e.Value), SourceSpan: e.Span})
		collectExprMIROps(env, e.Value, fn, templateInfo)
	case *RefExpr:
		kind := "ref"
		if e.Const {
			kind = "ref_const"
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Detail: exprLabel(e.Value), SourceSpan: e.Span})
	case *StructConstructExpr:
		kind := "struct_construct"
		if !evt1TypeCopyable(env, Type{Name: e.StructName, Kind: TypeStruct}) {
			kind = "noncopyable_struct_construct"
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Detail: e.StructName, SourceSpan: e.Span})
		for _, arg := range e.Args {
			collectExprMIROps(env, arg, fn, templateInfo)
		}
	case *WithExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "record_with", Detail: fmt.Sprintf("%d updates", len(e.Updates)), SourceSpan: e.Span})
		collectExprMIROps(env, e.Base, fn, templateInfo)
		for _, update := range e.Updates {
			collectExprMIROps(env, update.Value, fn, templateInfo)
		}
	case *ConstructExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "enum_construct", Detail: e.EnumName + "::" + e.VariantName, SourceSpan: e.Span})
		for _, arg := range e.Args {
			collectExprMIROps(env, arg, fn, templateInfo)
		}
	case *MatchExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "match_expr", Detail: fmt.Sprintf("%d arms", len(e.Arms)), SourceSpan: e.Span})
		collectExprMIROps(env, e.Subject, fn, templateInfo)
		for _, arm := range e.Arms {
			fn.Operations = append(fn.Operations, MIROperation{
				ID:         fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1),
				Kind:       "pattern",
				Detail:     arm.Pattern.EnumName + "::" + arm.Pattern.VariantName,
				SourceSpan: arm.Pattern.Span,
			})
			collectExprMIROps(env, arm.Value, fn, templateInfo)
		}
	case *CallExpr:
		kind := "call"
		detail := e.Callee
		if e.Callee == "Len" {
			kind = "array_len"
		}
		if templateInfo != nil {
			if binding, ok := templateInfo.CallBindings[evt1SpanKey(e.Span)]; ok {
				kind = "requirement_call"
				detail = binding.Requirement.ID + " -> " + binding.Requirement.Operation.Name
			}
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Detail: detail, SourceSpan: e.Span})
		for _, arg := range e.Args {
			collectExprMIROps(env, arg, fn, templateInfo)
		}
	case *DispatchExpr:
		detail := e.InstanceName
		if e.BatchName != "" {
			detail += " -> " + e.BatchName
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "dispatch", Detail: detail, SourceSpan: e.Span})
		collectExprMIROps(env, e.Signal, fn, templateInfo)
	case *TemplateCallExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "template_call", Detail: e.Callee + "<" + e.TypeArg.String() + ">", SourceSpan: e.Span})
		for _, arg := range e.Args {
			collectExprMIROps(env, arg, fn, templateInfo)
		}
	case *FieldExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "field_access", Detail: e.Field, SourceSpan: e.Span})
		collectExprMIROps(env, e.Receiver, fn, templateInfo)
	case *ArrayLiteralExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "array_literal", Detail: fmt.Sprintf("%d elements", len(e.Elements)), SourceSpan: e.Span})
		for _, element := range e.Elements {
			collectExprMIROps(env, element, fn, templateInfo)
		}
	case *IndexExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "array_index", SourceSpan: e.Span})
		collectExprMIROps(env, e.Base, fn, templateInfo)
		collectExprMIROps(env, e.Index, fn, templateInfo)
	case *BinaryExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "binary", Detail: e.Op, SourceSpan: e.Span})
		collectExprMIROps(env, e.Left, fn, templateInfo)
		collectExprMIROps(env, e.Right, fn, templateInfo)
	case *UnaryExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "unary", Detail: e.Op, SourceSpan: e.Span})
		collectExprMIROps(env, e.Value, fn, templateInfo)
	case *NameExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "name", Detail: e.Name, SourceSpan: e.Span})
	case *IntLiteral:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "literal", Detail: fmt.Sprintf("%d", e.Value), SourceSpan: e.Span})
	case *StringLiteral:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "string_literal", Detail: e.Value, SourceSpan: e.Span})
	case *BoolLiteral:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "bool_literal", Detail: fmt.Sprintf("%t", e.Value), SourceSpan: e.Span})
	case *IfExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "if_expr", SourceSpan: e.Span})
		collectExprMIROps(env, e.Condition, fn, templateInfo)
		collectExprMIROps(env, e.Then, fn, templateInfo)
		collectExprMIROps(env, e.Else, fn, templateInfo)
	case *ParenExpr:
		collectExprMIROps(env, e.Value, fn, templateInfo)
	}
}

func evt1MapFunctions(module Module, env *semanticEnv) []map[string]any {
	out := make([]map[string]any, 0, len(module.Functions))
	for _, fn := range module.Functions {
		out = append(out, map[string]any{
			"name":   fn.Name,
			"span":   fn.Span,
			"symbol": evt1FunctionSymbolForDecl(evt1OutputBase(module.Path), env, fn),
		})
	}
	return out
}

func evt1OutputBase(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(strings.ToLower(base), "-", "_")
	return base
}

func evt1FunctionSymbol(base, name string) string {
	return "concept_" + base + "_" + evt1CName(name)[len("concept_"):]
}

func evt1FunctionSymbolForDecl(base string, env *semanticEnv, fn FunctionDecl) string {
	if len(env.functions[fn.Name]) <= 1 {
		return evt1FunctionSymbol(base, fn.Name)
	}
	var parts []string
	for _, param := range fn.Params {
		parts = append(parts, evt1TypeIdentity(evt1CanonicalType(env, param.Type)))
	}
	return evt1FunctionSymbol(base, fn.Name) + "__" + strings.Join(parts, "_")
}

func (l *lowering) generateC() ([]byte, []byte, error) {
	var header, body strings.Builder
	guard := strings.ToUpper("PROM_" + l.outputBase + "_GENERATED_H")
	typeDecls, err := l.runtimeTypeDeclarations()
	if err != nil {
		return nil, nil, err
	}
	body.WriteString(fmt.Sprintf("/* Generated by %s. DO NOT EDIT. Source: %s */\n", CompilerID, l.module.Path))
	body.WriteString(fmt.Sprintf("#include \"%s.generated.h\"\n", l.outputBase))
	body.WriteString("#include <stdio.h>\n#include <stdlib.h>\n\n")
	header.WriteString(fmt.Sprintf("/* Generated by %s. DO NOT EDIT. */\n", CompilerID))
	header.WriteString(fmt.Sprintf("#ifndef %s\n#define %s\n", guard, guard))
	for _, include := range l.env.profile.headersFor(l.module) {
		header.WriteString("#include " + include + "\n")
	}
	header.WriteString("#include <stdbool.h>\n#include <stdint.h>\n\n")
	var builtinNames []string
	for name := range l.env.profile.BuiltinTypes {
		builtinNames = append(builtinNames, name)
	}
	sort.Strings(builtinNames)
	for _, name := range builtinNames {
		builtin := l.env.profile.BuiltinTypes[name]
		if builtin.CDeclaration != "" && evt1TypeUsed(l.module, func(t Type) bool { return t.Name == name }) {
			header.WriteString(builtin.CDeclaration + "\n")
		}
	}
	for _, decl := range typeDecls {
		switch {
		case decl.Struct != nil:
			header.WriteString(l.structHeader(*decl.Struct))
		case decl.Enum != nil:
			header.WriteString(l.enumHeader(*decl.Enum))
		}
	}
	canonicalFailureTypes := evt1CanonicalFailureTypeKeys(l.module)
	for _, failureType := range evt1CollectFailureTypes(l.module) {
		if _, legacy := evt1IsResultVoidErrorType(l.env, failureType); legacy && !canonicalFailureTypes[evt1FailureTypeKey(failureType)] {
			header.WriteString(l.resultTypeDecl(failureType))
		} else {
			header.WriteString(evt1FailureTypeDecl(failureType))
		}
	}
	header.WriteString(l.actuatorSupportDecls())
	var symbols []evt1FunctionSymbols
	for _, fn := range l.module.Functions {
		symbols = append(symbols, l.functionSymbols(fn))
	}
	for _, sym := range symbols {
		header.WriteString(sym.Prototype)
		header.WriteByte('\n')
	}
	header.WriteString("\n#endif\n")
	body.WriteString("static void concept_abort_invalid_tag(const char* enum_name) {\n")
	body.WriteString("  fprintf(stderr, \"invalid enum tag for %s\\n\", enum_name);\n")
	body.WriteString("  abort();\n}\n\n")
	if evt1ModuleUsesFailurePanic(l.module) {
		body.WriteString("static void concept_panic(const char* reason, int line, int column) {\n")
		body.WriteString("  fprintf(stderr, \"Concept panic at %d:%d: %s\\n\", line, column, reason);\n")
		body.WriteString("  abort();\n}\n\n")
	}
	if len(evt1RuntimeAutomataUsage(l.module)) > 0 {
		body.WriteString("static void concept_abort_invalid_automata_state(const char* automata_name, int machine, int state) {\n")
		body.WriteString("  fprintf(stderr, \"invalid automata state for %s: machine=%d state=%d\\n\", automata_name, machine, state);\n")
		body.WriteString("  abort();\n}\n\n")
		body.WriteString("static void concept_abort_automata_stack(const char* automata_name, const char* reason) {\n")
		body.WriteString("  fprintf(stderr, \"invalid automata stack for %s: %s\\n\", automata_name, reason);\n")
		body.WriteString("  abort();\n}\n\n")
		body.WriteString("static void concept_abort_automata_completion(const char* automata_name, int steps) {\n")
		body.WriteString("  fprintf(stderr, \"automata completion bound exceeded for %s: steps=%d\\n\", automata_name, steps);\n")
		body.WriteString("  abort();\n}\n\n")
	}
	for _, structDecl := range l.module.Structs {
		if evt1RuntimeTypeSafe(l.env, Type{Name: structDecl.Name, Kind: TypeStruct}) && evt1TypeCopyable(l.env, Type{Name: structDecl.Name, Kind: TypeStruct}) {
			body.WriteString(l.structConstructor(structDecl))
		}
	}
	for _, enumDecl := range l.module.Enums {
		if evt1RuntimeTypeSafe(l.env, Type{Name: enumDecl.Name, Kind: TypeEnum}) {
			body.WriteString(l.enumConstructors(enumDecl))
		}
	}
	for _, failureType := range evt1CollectFailureTypes(l.module) {
		if canonicalFailureTypes[evt1FailureTypeKey(failureType)] {
			body.WriteString(evt1FailureConstructors(failureType))
			body.WriteString(evt1FailureDropFunction(l.env, failureType, l.outputBase))
		}
	}
	if evt1ModuleUsesAutomataDispatchOutcome(l.module) {
		body.WriteString(l.enumConstructors(evt1BuiltinAutomataDispatchOutcomeEnum()))
	}
	if len(l.module.Actuators) > 0 {
		body.WriteString(l.enumConstructors(evt1BuiltinActuationOutcomeEnum()))
	}
	for _, automataName := range evt1RuntimeAutomataUsageOrder(l.module) {
		body.WriteString(l.automataRuntimeSupport(l.env.automataInfo[automataName]))
	}
	for _, actuatorDecl := range l.module.Actuators {
		body.WriteString(l.actuatorRuntimeSupport(actuatorDecl.Name))
	}
	for _, templateDecl := range l.module.Templates {
		var instances []*evt1TemplateInstance
		for _, instance := range l.env.templateInstances {
			if instance.TemplateName == templateDecl.Name {
				instances = append(instances, instance)
			}
		}
		sort.Slice(instances, func(i, j int) bool {
			return instances[i].TypeIdentity < instances[j].TypeIdentity
		})
		for _, instance := range instances {
			body.WriteString(l.templateInstanceBody(instance))
			body.WriteByte('\n')
		}
	}
	for _, sym := range symbols {
		if sym.Body != "" {
			body.WriteString(sym.Body)
			body.WriteByte('\n')
		}
	}
	headerText := strings.TrimRight(header.String(), "\n") + "\n"
	bodyText := strings.TrimRight(body.String(), "\n") + "\n"
	return []byte(headerText), []byte(bodyText), nil
}

type evt1RuntimeTypeDecl struct {
	Name   string
	Struct *StructDecl
	Enum   *EnumDecl
}

func (l *lowering) runtimeTypeDeclarations() ([]evt1RuntimeTypeDecl, error) {
	index := make(map[string]evt1RuntimeTypeDecl)
	order := make([]string, 0, len(l.module.Structs)+len(l.module.Enums))
	for _, structDecl := range l.module.Structs {
		t := Type{Name: structDecl.Name, Kind: TypeStruct}
		if !evt1RuntimeTypeSafe(l.env, t) {
			continue
		}
		decl := structDecl
		index[structDecl.Name] = evt1RuntimeTypeDecl{Name: structDecl.Name, Struct: &decl}
		order = append(order, structDecl.Name)
	}
	for _, enumDecl := range l.module.Enums {
		t := Type{Name: enumDecl.Name, Kind: TypeEnum}
		if !evt1RuntimeTypeSafe(l.env, t) {
			continue
		}
		decl := enumDecl
		index[enumDecl.Name] = evt1RuntimeTypeDecl{Name: enumDecl.Name, Enum: &decl}
		order = append(order, enumDecl.Name)
	}
	if evt1ModuleUsesAutomataDispatchOutcome(l.module) {
		outcome := evt1BuiltinAutomataDispatchOutcomeEnum()
		index[outcome.Name] = evt1RuntimeTypeDecl{Name: outcome.Name, Enum: &outcome}
		order = append(order, outcome.Name)
	}
	if len(l.module.Actuators) > 0 {
		outcome := evt1BuiltinActuationOutcomeEnum()
		index[outcome.Name] = evt1RuntimeTypeDecl{Name: outcome.Name, Enum: &outcome}
		order = append(order, outcome.Name)
	}

	seen := make(map[string]bool, len(index))
	active := make(map[string]bool, len(index))
	var sorted []evt1RuntimeTypeDecl
	var visit func(string) error
	visit = func(name string) error {
		if seen[name] {
			return nil
		}
		if active[name] {
			return fmt.Errorf("Concept/Vulkan EVT1 runtime type cycle requires unsupported forward declarations: %s", name)
		}
		decl, ok := index[name]
		if !ok {
			return nil
		}
		active[name] = true
		for _, dep := range l.runtimeTypeDeclDeps(decl) {
			if err := visit(dep); err != nil {
				return err
			}
		}
		active[name] = false
		seen[name] = true
		sorted = append(sorted, decl)
		return nil
	}
	for _, name := range order {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return sorted, nil
}

func (l *lowering) runtimeTypeDeclDeps(decl evt1RuntimeTypeDecl) []string {
	var deps []string
	seen := map[string]bool{decl.Name: true}
	add := func(t Type) {
		for _, dep := range l.runtimeTypeDeps(t) {
			if seen[dep] {
				continue
			}
			if _, ok := l.env.structs[dep]; !ok {
				if _, ok := l.env.enums[dep]; !ok {
					continue
				}
			}
			if !l.runtimeTypeDeclExists(dep) {
				continue
			}
			seen[dep] = true
			deps = append(deps, dep)
		}
	}
	switch {
	case decl.Struct != nil:
		for _, field := range decl.Struct.Fields {
			add(field.Type)
		}
	case decl.Enum != nil:
		for _, variant := range decl.Enum.Variants {
			for _, field := range variant.Payload {
				add(field.Type)
			}
		}
	}
	return deps
}

func (l *lowering) runtimeTypeDeclExists(name string) bool {
	if structDecl, ok := l.env.structs[name]; ok {
		return evt1RuntimeTypeSafe(l.env, Type{Name: structDecl.Name, Kind: TypeStruct})
	}
	if enumDecl, ok := l.env.enums[name]; ok {
		return evt1RuntimeTypeSafe(l.env, Type{Name: enumDecl.Name, Kind: TypeEnum})
	}
	return false
}

func evt1RuntimeAutomataUsage(module Module) map[string]bool {
	used := map[string]bool{}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *EffectsDecl:
				used[s.AutomataName] = true
			case *InstanceDecl:
				used[s.AutomataName] = true
			case *MatchStmt:
				for _, arm := range s.Arms {
					visitBlock(arm.Block)
				}
			case *WhileStmt:
				visitBlock(s.Body)
			case *Block:
				visitBlock(*s)
			}
		}
	}
	for _, tpl := range module.Templates {
		if tpl.Body != nil {
			visitBlock(*tpl.Body)
		}
	}
	for _, fn := range module.Functions {
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	return used
}

func evt1RuntimeAutomataUsageOrder(module Module) []string {
	used := evt1RuntimeAutomataUsage(module)
	var order []string
	for _, decl := range module.Automata {
		if used[decl.Name] {
			order = append(order, decl.Name)
		}
	}
	return order
}

func evt1ModuleUsesAutomataDispatchOutcome(module Module) bool {
	var usesExpr func(Expr) bool
	usesExpr = func(expr Expr) bool {
		switch e := expr.(type) {
		case *DispatchExpr:
			return true
		case *ParenExpr:
			return usesExpr(e.Value)
		case *UnaryExpr:
			return usesExpr(e.Value)
		case *FieldExpr:
			return usesExpr(e.Receiver)
		case *IndexExpr:
			return usesExpr(e.Base) || usesExpr(e.Index)
		case *BinaryExpr:
			return usesExpr(e.Left) || usesExpr(e.Right)
		case *CallExpr:
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *TemplateCallExpr:
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *ConstructExpr:
			if e.EnumName == evt1AutomataDispatchOutcomeTypeName {
				return true
			}
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *StructConstructExpr:
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *WithExpr:
			if usesExpr(e.Base) {
				return true
			}
			for _, update := range e.Updates {
				if usesExpr(update.Value) {
					return true
				}
			}
		case *ArrayLiteralExpr:
			for _, arg := range e.Elements {
				if usesExpr(arg) {
					return true
				}
			}
		case *MatchExpr:
			if usesExpr(e.Subject) {
				return true
			}
			for _, arm := range e.Arms {
				if usesExpr(arm.Value) {
					return true
				}
			}
		case *IfExpr:
			return usesExpr(e.Condition) || usesExpr(e.Then) || usesExpr(e.Else)
		}
		return false
	}
	var visitBlock func(Block) bool
	visitBlock = func(block Block) bool {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				if s.Type.Name == evt1AutomataDispatchOutcomeTypeName || usesExpr(s.Value) {
					return true
				}
			case *AssignStmt:
				if usesExpr(s.Target) || usesExpr(s.Value) {
					return true
				}
			case *ReturnStmt:
				if s.Value != nil && usesExpr(s.Value) {
					return true
				}
			case *ExprStmt:
				if usesExpr(s.Value) {
					return true
				}
			case *StaticAssertStmt:
				if usesExpr(s.Condition) || (s.Message != nil && usesExpr(s.Message)) {
					return true
				}
			case *MatchStmt:
				if usesExpr(s.Subject) {
					return true
				}
				for _, arm := range s.Arms {
					if visitBlock(arm.Block) {
						return true
					}
				}
			case *WhileStmt:
				if usesExpr(s.Condition) || (s.Bound != nil && usesExpr(s.Bound)) || visitBlock(s.Body) {
					return true
				}
			case *Block:
				if visitBlock(*s) {
					return true
				}
			}
		}
		return false
	}
	for _, tpl := range module.Templates {
		if tpl.ReturnType.Name == evt1AutomataDispatchOutcomeTypeName {
			return true
		}
		if tpl.Body != nil && visitBlock(*tpl.Body) {
			return true
		}
	}
	for _, fn := range module.Functions {
		if fn.ReturnType.Name == evt1AutomataDispatchOutcomeTypeName {
			return true
		}
		for _, param := range fn.Params {
			if param.Type.Name == evt1AutomataDispatchOutcomeTypeName {
				return true
			}
		}
		if fn.Body != nil && visitBlock(*fn.Body) {
			return true
		}
	}
	return false
}

func (l *lowering) runtimeTypeDeps(t Type) []string {
	if t.Ownership == "borrow" {
		return l.runtimeTypeDeps(t.borrowBase())
	}
	if t.Name == "Result" && len(t.TypeArgs) == 2 {
		return l.runtimeTypeDeps(t.TypeArgs[1])
	}
	switch t.Kind {
	case TypePointer:
		return l.runtimeTypeDeps(*t.PointerTo)
	case TypeArray:
		return l.runtimeTypeDeps(*t.ArrayElem)
	case TypeStruct, TypeEnum:
		return []string{t.Name}
	default:
		return nil
	}
}

func (l *lowering) resultTypesUsed() []Type {
	seen := map[string]Type{}
	add := func(t Type) {
		if errType, ok := evt1IsResultVoidErrorType(l.env, t); ok {
			key := evt1TypeIdentity(evt1CanonicalType(l.env, errType))
			seen[key] = t
		}
	}
	for _, fn := range l.module.Functions {
		add(fn.ReturnType)
		for _, param := range fn.Params {
			add(param.Type)
		}
	}
	var keys []string
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []Type
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out
}

func (l *lowering) resultTypeDecl(t Type) string {
	errType, _ := evt1IsResultVoidErrorType(l.env, t)
	name := evt1CType(t)
	return fmt.Sprintf("typedef struct %s {\n  bool is_error;\n  %s error;\n} %s;\n\n", name, evt1CType(errType), name)
}

func (l *lowering) actuatorSupportDecls() string {
	var b strings.Builder
	for _, decl := range l.module.Actuators {
		info := l.env.actuatorInfo[decl.Name]
		b.WriteString(fmt.Sprintf("typedef struct %s {\n", evt1ActuatorLocalCName(decl.Name)))
		b.WriteString(fmt.Sprintf("  %s mechanism;\n", evt1CType(info.MechanismType)))
		b.WriteString(fmt.Sprintf("} %s;\n\n", evt1ActuatorLocalCName(decl.Name)))
		b.WriteString(fmt.Sprintf("typedef struct %s {\n", evt1CType(Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: decl.Span})))
		b.WriteString(fmt.Sprintf("  %s outcome;\n", evt1CType(Type{Name: evt1ActuationOutcomeTypeName, Kind: TypeEnum, Span: decl.Span})))
		b.WriteString("  int completedCount;\n")
		b.WriteString("  int failedIndex;\n")
		b.WriteString(fmt.Sprintf("  %s error;\n", evt1CType(info.ErrorType)))
		b.WriteString(fmt.Sprintf("} %s;\n\n", evt1CType(Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: decl.Span})))
	}
	return b.String()
}

func evt1AutomataRuntimeInstanceCName(automataName string) string {
	return evt1CName(automataName) + "_instance"
}

func evt1AutomataRuntimeContinuationCName(automataName string) string {
	return evt1CName(automataName) + "_continuation"
}

func evt1AutomataRuntimeInitName(automataName string) string {
	return evt1CName(automataName) + "_init"
}

func evt1AutomataRuntimeNormalizeName(automataName string) string {
	return evt1CName(automataName) + "_normalize"
}

func evt1AutomataRuntimeDispatchName(automataName string) string {
	return evt1CName(automataName) + "_dispatch"
}

func evt1AutomataEffectTagTypeCName(automataName string) string {
	return evt1CName(automataName) + "_effect_tag"
}

func evt1AutomataEffectEntryCName(automataName string) string {
	return evt1CName(automataName) + "_effect_entry"
}

func evt1AutomataEffectBatchCName(automataName string) string {
	return evt1CName(automataName) + "_effects"
}

func evt1AutomataEffectBatchStateCName(automataName string) string {
	return evt1CName(automataName) + "_effect_batch_state"
}

func evt1AutomataEffectBatchFailureCName(automataName string) string {
	return evt1CName(automataName) + "_effect_failure"
}

func evt1AutomataEffectBatchStateConstName(automataName, stateName string) string {
	return strings.ToUpper(evt1CName(automataName) + "_effect_batch_" + evt1PayloadFieldName(stateName))
}

func evt1AutomataEffectTagConstName(automataName, effectName string) string {
	return strings.ToUpper(evt1CName(automataName) + "_effect_" + evt1PayloadFieldName(effectName))
}

func evt1AutomataMachineConstName(automataName, machineName string) string {
	return evt1CName(automataName) + "_machine_" + evt1PayloadFieldName(machineName)
}

func evt1AutomataStateConstName(automataName, machineName, stateName string) string {
	return evt1CName(automataName) + "_state_" + evt1PayloadFieldName(machineName) + "_" + evt1PayloadFieldName(stateName)
}

func evt1ZeroCValue(t Type) string {
	return "(" + evt1CType(t) + "){0}"
}

func evt1InitialStateName(machine MachineDecl) string {
	for _, state := range machine.States {
		if state.Initial {
			return state.Name
		}
	}
	return ""
}

func (l *lowering) automataRuntimeSupport(info *evt1AutomataInfo) string {
	var b strings.Builder
	instanceType := evt1AutomataRuntimeInstanceCName(info.Decl.Name)
	continuationType := evt1AutomataRuntimeContinuationCName(info.Decl.Name)
	initName := evt1AutomataRuntimeInitName(info.Decl.Name)
	normalizeName := evt1AutomataRuntimeNormalizeName(info.Decl.Name)
	dispatchName := evt1AutomataRuntimeDispatchName(info.Decl.Name)
	signalType := evt1CType(info.Decl.SignalType)
	outcomeType := evt1CType(Type{Name: evt1AutomataDispatchOutcomeTypeName, Kind: TypeEnum})
	var contextParamType string
	if info.Decl.Context != nil {
		contextType := info.Decl.Context.Type
		contextType.Ownership = "borrow"
		contextType.Const = true
		contextParamType = evt1CType(contextType)
	}
	outcomeCtor := func(name string) string {
		return evt1ConstructorName(evt1AutomataDispatchOutcomeTypeName, name) + "()"
	}
	machineIndex := map[string]MachineDecl{}
	for _, machine := range info.Decl.Machines {
		machineIndex[machine.Name] = machine
	}

	b.WriteString(fmt.Sprintf("enum {\n"))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataMachineConstName(info.Decl.Name, machine.Name), info.MachineOrdinal[machine.Name]))
	}
	b.WriteString("};\n\n")
	for _, machine := range info.Decl.Machines {
		b.WriteString("enum {\n")
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name), info.StateOrdinal[machine.Name][state.Name]))
		}
		b.WriteString("};\n\n")
	}
	if info.ContinuationCapacity > 0 {
		b.WriteString(fmt.Sprintf("typedef struct %s {\n", continuationType))
		b.WriteString("  uint8_t caller_machine;\n")
		b.WriteString("  uint8_t resume_state;\n")
		b.WriteString(fmt.Sprintf("} %s;\n\n", continuationType))
	}
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", instanceType))
	b.WriteString("  bool finished;\n")
	b.WriteString("  uint8_t current_machine;\n")
	b.WriteString("  uint8_t current_state;\n")
	b.WriteString("  uint8_t continuation_count;\n")
	if info.Decl.Context != nil {
		b.WriteString(fmt.Sprintf("  %s context;\n", contextParamType))
	}
	if info.ContinuationCapacity > 0 {
		b.WriteString(fmt.Sprintf("  %s continuations[%d];\n", continuationType, info.ContinuationCapacity))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", instanceType))

	b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n", normalizeName, instanceType))
	b.WriteString("  int steps = 0;\n")
	b.WriteString("  while (!instance->finished) {\n")
	b.WriteString("    steps = steps + 1;\n")
	b.WriteString(fmt.Sprintf("    if (steps > %d) {\n", info.CompletionStepBound))
	b.WriteString(fmt.Sprintf("      concept_abort_automata_completion(\"%s\", steps);\n", info.Decl.Name))
	b.WriteString("    }\n")
	b.WriteString("    switch (instance->current_machine) {\n")
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("      case %s:\n", evt1AutomataMachineConstName(info.Decl.Name, machine.Name)))
		b.WriteString("        switch (instance->current_state) {\n")
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("          case %s:\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name)))
			if !state.Terminal {
				b.WriteString("            return;\n")
				continue
			}
			switch state.Completion[0].Kind {
			case "finish":
				b.WriteString("            instance->finished = true;\n")
				b.WriteString("            instance->continuation_count = 0;\n")
				b.WriteString("            return;\n")
			case "pop":
				if info.ContinuationCapacity == 0 {
					b.WriteString(fmt.Sprintf("            concept_abort_automata_stack(\"%s\", \"pop with zero continuation capacity\");\n", info.Decl.Name))
					b.WriteString("            return;\n")
					break
				}
				b.WriteString("            if (instance->continuation_count == 0) {\n")
				b.WriteString(fmt.Sprintf("              concept_abort_automata_stack(\"%s\", \"pop underflow\");\n", info.Decl.Name))
				b.WriteString("            }\n")
				b.WriteString("            instance->continuation_count = (uint8_t)(instance->continuation_count - 1);\n")
				b.WriteString("            instance->current_machine = instance->continuations[instance->continuation_count].caller_machine;\n")
				b.WriteString("            instance->current_state = instance->continuations[instance->continuation_count].resume_state;\n")
				b.WriteString("            break;\n")
			default:
				b.WriteString(fmt.Sprintf("            concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
				b.WriteString("            return;\n")
			}
		}
		b.WriteString("          default:\n")
		b.WriteString(fmt.Sprintf("            concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
		b.WriteString("            return;\n")
		b.WriteString("        }\n")
		b.WriteString("        break;\n")
	}
	b.WriteString("      default:\n")
	b.WriteString(fmt.Sprintf("        concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
	b.WriteString("        return;\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n\n")

	rootMachine := machineIndex[info.RootMachine]
	rootInitialState := evt1InitialStateName(rootMachine)
	if info.Decl.Context != nil {
		b.WriteString(fmt.Sprintf("static void %s(%s* instance, %s context) {\n", initName, instanceType, contextParamType))
	} else {
		b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n", initName, instanceType))
	}
	b.WriteString("  instance->finished = false;\n")
	b.WriteString(fmt.Sprintf("  instance->current_machine = %s;\n", evt1AutomataMachineConstName(info.Decl.Name, info.RootMachine)))
	b.WriteString(fmt.Sprintf("  instance->current_state = %s;\n", evt1AutomataStateConstName(info.Decl.Name, info.RootMachine, rootInitialState)))
	b.WriteString("  instance->continuation_count = 0;\n")
	if info.Decl.Context != nil {
		b.WriteString("  instance->context = context;\n")
	}
	b.WriteString(fmt.Sprintf("  %s(instance);\n", normalizeName))
	b.WriteString("}\n\n")

	if len(info.EffectSet) > 0 {
		b.WriteString(l.automataEffectBatchTypes(info))
		b.WriteString(l.automataEffectfulDispatch(info, machineIndex, instanceType, signalType, outcomeType, dispatchName, normalizeName))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("static %s %s(%s* instance, %s signal) {\n", outcomeType, dispatchName, instanceType, signalType))
	b.WriteString("  if (instance->finished) {\n")
	b.WriteString(fmt.Sprintf("    return %s;\n", outcomeCtor("AlreadyFinished")))
	b.WriteString("  }\n")
	b.WriteString("  switch (instance->current_machine) {\n")
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("    case %s:\n", evt1AutomataMachineConstName(info.Decl.Name, machine.Name)))
		b.WriteString("      switch (instance->current_state) {\n")
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("        case %s:\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name)))
			if state.Terminal {
				b.WriteString(fmt.Sprintf("          concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
				b.WriteString(fmt.Sprintf("          return %s;\n", outcomeCtor("AlreadyFinished")))
				continue
			}
			b.WriteString("          switch (signal.tag) {\n")
			for _, group := range evt1AutomataHandlerGroups(state) {
				b.WriteString(fmt.Sprintf("            case %s:\n", evt1TagName(info.SignalEnum.Name, group[0].Signal.MemberName)))
				if len(group) == 1 && group[0].Guard == nil && !group[0].Otherwise {
					b.WriteString(l.automataDispatchAction(info, machine, machineIndex, group[0], 7, "instance", "", ""))
					b.WriteString(fmt.Sprintf("              %s(instance);\n", normalizeName))
					b.WriteString("              if (instance->finished) {\n")
					b.WriteString(fmt.Sprintf("                return %s;\n", outcomeCtor("Finished")))
					b.WriteString("              }\n")
					b.WriteString(fmt.Sprintf("              return %s;\n", outcomeCtor("Transitioned")))
					continue
				}
				b.WriteString("              uint8_t eligible_count = 0;\n")
				b.WriteString("              uint8_t selected_candidate = 0;\n")
				fallbackOrdinal := 0
				guardedOrdinal := 0
				for _, handler := range group {
					if handler.Otherwise {
						fallbackOrdinal = guardedOrdinal + 1
						continue
					}
					guardedOrdinal++
					prelude, guardExpr := l.lowerAutomataGuard(info, handler.Guard, 7, "instance")
					b.WriteString(prelude)
					b.WriteString(ind(7) + fmt.Sprintf("if (%s) {\n", guardExpr))
					b.WriteString(ind(8) + "eligible_count = (uint8_t)(eligible_count + 1);\n")
					b.WriteString(ind(8) + fmt.Sprintf("selected_candidate = %d;\n", guardedOrdinal))
					b.WriteString(ind(7) + "}\n")
				}
				b.WriteString("              if (eligible_count > 1) {\n")
				b.WriteString(fmt.Sprintf("                return %s;\n", outcomeCtor("Ambiguous")))
				b.WriteString("              }\n")
				b.WriteString("              if (eligible_count == 0) {\n")
				if fallbackOrdinal > 0 {
					b.WriteString(fmt.Sprintf("                selected_candidate = %d;\n", fallbackOrdinal))
				} else {
					b.WriteString(fmt.Sprintf("                return %s;\n", outcomeCtor("Unhandled")))
				}
				b.WriteString("              }\n")
				b.WriteString("              switch (selected_candidate) {\n")
				candidateOrdinal := 0
				for _, handler := range group {
					candidateOrdinal++
					b.WriteString(fmt.Sprintf("                case %d:\n", candidateOrdinal))
					b.WriteString(l.automataDispatchAction(info, machine, machineIndex, handler, 9, "instance", "", ""))
					b.WriteString("                  break;\n")
				}
				b.WriteString("                default:\n")
				b.WriteString(fmt.Sprintf("                  return %s;\n", outcomeCtor("Unhandled")))
				b.WriteString("              }\n")
				b.WriteString(fmt.Sprintf("              %s(instance);\n", normalizeName))
				b.WriteString("              if (instance->finished) {\n")
				b.WriteString(fmt.Sprintf("                return %s;\n", outcomeCtor("Finished")))
				b.WriteString("              }\n")
				b.WriteString(fmt.Sprintf("              return %s;\n", outcomeCtor("Transitioned")))
			}
			b.WriteString("            default:\n")
			b.WriteString(fmt.Sprintf("              return %s;\n", outcomeCtor("Unhandled")))
			b.WriteString("          }\n")
		}
		b.WriteString("        default:\n")
		b.WriteString(fmt.Sprintf("          concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
		b.WriteString(fmt.Sprintf("          return %s;\n", outcomeCtor("AlreadyFinished")))
		b.WriteString("      }\n")
	}
	b.WriteString("    default:\n")
	b.WriteString(fmt.Sprintf("      concept_abort_invalid_automata_state(\"%s\", instance->current_machine, instance->current_state);\n", info.Decl.Name))
	b.WriteString(fmt.Sprintf("      return %s;\n", outcomeCtor("AlreadyFinished")))
	b.WriteString("  }\n")
	b.WriteString("}\n\n")
	return b.String()
}

func evt1AutomataHandlerGroups(state StateDecl) [][]TransitionDecl {
	groups := map[string][]TransitionDecl{}
	var order []string
	for _, handler := range state.Handlers {
		key := handler.Signal.EnumName + "::" + handler.Signal.MemberName
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], handler)
	}
	out := make([][]TransitionDecl, 0, len(order))
	for _, key := range order {
		out = append(out, groups[key])
	}
	return out
}

func (l *lowering) automataEffectBatchTypes(info *evt1AutomataInfo) string {
	var b strings.Builder
	tagType := evt1AutomataEffectTagTypeCName(info.Decl.Name)
	entryType := evt1AutomataEffectEntryCName(info.Decl.Name)
	batchType := evt1AutomataEffectBatchCName(info.Decl.Name)
	stateType := evt1AutomataEffectBatchStateCName(info.Decl.Name)
	failureType := evt1AutomataEffectBatchFailureCName(info.Decl.Name)
	b.WriteString(fmt.Sprintf("typedef enum %s {\n", tagType))
	for index, effectName := range info.EffectSet {
		b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataEffectTagConstName(info.Decl.Name, effectName), index))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", tagType))
	b.WriteString(fmt.Sprintf("typedef enum %s {\n", stateType))
	for i, stateName := range []string{"Vacant", "Pending", "Completed", "Failed"} {
		b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataEffectBatchStateConstName(info.Decl.Name, stateName), i))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", stateType))
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", entryType))
	b.WriteString(fmt.Sprintf("  %s tag;\n", tagType))
	b.WriteString("  union {\n")
	for _, effectName := range info.EffectSet {
		effectDecl := l.env.effects[effectName]
		b.WriteString("    struct {\n")
		if len(effectDecl.Params) == 0 {
			b.WriteString("      unsigned char unused;\n")
		} else {
			for _, param := range effectDecl.Params {
				b.WriteString(fmt.Sprintf("      %s %s;\n", evt1CType(param.Type), param.Name))
			}
		}
		b.WriteString(fmt.Sprintf("    } %s;\n", evt1PayloadFieldName(effectDecl.Name)))
	}
	b.WriteString("  } payload;\n")
	b.WriteString(fmt.Sprintf("} %s;\n\n", entryType))
	actuatorNames := evt1ActuatorsForAutomata(l.module, info.Decl.Name)
	if len(actuatorNames) > 0 {
		b.WriteString(fmt.Sprintf("typedef union %s {\n", failureType))
		for _, actuatorName := range actuatorNames {
			actuatorInfo := l.env.actuatorInfo[actuatorName]
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(actuatorInfo.ErrorType), actuatorInfo.FailureSlot))
		}
		b.WriteString(fmt.Sprintf("} %s;\n\n", failureType))
	}
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", batchType))
	b.WriteString(fmt.Sprintf("  %s state;\n", stateType))
	b.WriteString("  uint8_t count;\n")
	b.WriteString("  uint8_t cursor;\n")
	b.WriteString("  uint8_t failed_index;\n")
	if len(actuatorNames) > 0 {
		b.WriteString("  uint8_t failure_actuator;\n")
		b.WriteString(fmt.Sprintf("  %s failure;\n", failureType))
	}
	b.WriteString(fmt.Sprintf("  %s entries[%d];\n", entryType, info.MaxEffectBatch))
	b.WriteString(fmt.Sprintf("} %s;\n\n", batchType))
	b.WriteString(fmt.Sprintf("static void %s(%s* batch) {\n", evt1BatchDiscardName(info.Decl.Name), batchType))
	b.WriteString(ind(1) + fmt.Sprintf("batch->state = %s;\n", evt1AutomataEffectBatchStateConstName(info.Decl.Name, "Vacant")))
	b.WriteString(ind(1) + "batch->count = 0;\n")
	b.WriteString(ind(1) + "batch->cursor = 0;\n")
	b.WriteString(ind(1) + "batch->failed_index = 0;\n")
	if len(actuatorNames) > 0 {
		b.WriteString(ind(1) + "batch->failure_actuator = 0;\n")
	}
	b.WriteString("}\n\n")
	return b.String()
}

func (l *lowering) automataEffectfulDispatch(info *evt1AutomataInfo, machineIndex map[string]MachineDecl, instanceType, signalType, outcomeType, dispatchName, normalizeName string) string {
	var b strings.Builder
	batchType := evt1AutomataEffectBatchCName(info.Decl.Name)
	pendingState := evt1AutomataEffectBatchStateConstName(info.Decl.Name, "Pending")
	failedState := evt1AutomataEffectBatchStateConstName(info.Decl.Name, "Failed")
	outcomeCtor := func(name string) string {
		return evt1ConstructorName(evt1AutomataDispatchOutcomeTypeName, name) + "()"
	}
	b.WriteString(fmt.Sprintf("static %s %s(%s* instance, %s signal, %s* batch) {\n", outcomeType, dispatchName, instanceType, signalType, batchType))
	b.WriteString(ind(1) + "if (instance->finished) {\n")
	b.WriteString(ind(2) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
	b.WriteString(ind(2) + fmt.Sprintf("return %s;\n", outcomeCtor("AlreadyFinished")))
	b.WriteString(ind(1) + "}\n")
	b.WriteString(ind(1) + fmt.Sprintf("if (batch->state == %s || batch->state == %s) {\n", pendingState, failedState))
	b.WriteString(ind(2) + fmt.Sprintf("return %s;\n", outcomeCtor("EffectBatchOccupied")))
	b.WriteString(ind(1) + "}\n")
	b.WriteString(ind(1) + fmt.Sprintf("%s staged_batch = {0};\n", batchType))
	b.WriteString(ind(1) + fmt.Sprintf("staged_batch.state = %s;\n", pendingState))
	b.WriteString(ind(1) + "uint8_t staged_count = 0;\n")
	b.WriteString(ind(1) + fmt.Sprintf("%s staged_instance = *instance;\n", instanceType))
	b.WriteString(ind(1) + fmt.Sprintf("%s* staged = &staged_instance;\n", instanceType))
	b.WriteString(ind(1) + "switch (staged->current_machine) {\n")
	for _, machine := range info.Decl.Machines {
		b.WriteString(ind(2) + fmt.Sprintf("case %s:\n", evt1AutomataMachineConstName(info.Decl.Name, machine.Name)))
		b.WriteString(ind(3) + "switch (staged->current_state) {\n")
		for _, state := range machine.States {
			b.WriteString(ind(4) + fmt.Sprintf("case %s:\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name)))
			if state.Terminal {
				b.WriteString(ind(5) + fmt.Sprintf("concept_abort_invalid_automata_state(\"%s\", staged->current_machine, staged->current_state);\n", info.Decl.Name))
				b.WriteString(ind(5) + "batch->count = 0;\n")
				b.WriteString(ind(5) + fmt.Sprintf("return %s;\n", outcomeCtor("AlreadyFinished")))
				continue
			}
			b.WriteString(ind(5) + "switch (signal.tag) {\n")
			for _, group := range evt1AutomataHandlerGroups(state) {
				b.WriteString(ind(6) + fmt.Sprintf("case %s:\n", evt1TagName(info.SignalEnum.Name, group[0].Signal.MemberName)))
				if len(group) == 1 && group[0].Guard == nil && !group[0].Otherwise {
					b.WriteString(l.automataDispatchAction(info, machine, machineIndex, group[0], 7, "staged", "staged_batch", "staged_count"))
					b.WriteString(ind(7) + fmt.Sprintf("%s(&staged_instance);\n", normalizeName))
					b.WriteString(ind(7) + "*instance = staged_instance;\n")
					b.WriteString(ind(7) + "staged_batch.count = staged_count;\n")
					b.WriteString(ind(7) + "staged_batch.cursor = 0;\n")
					b.WriteString(ind(7) + "staged_batch.failed_index = 0;\n")
					b.WriteString(ind(7) + "*batch = staged_batch;\n")
					b.WriteString(ind(7) + "if (staged->finished) {\n")
					b.WriteString(ind(8) + fmt.Sprintf("return %s;\n", outcomeCtor("Finished")))
					b.WriteString(ind(7) + "}\n")
					b.WriteString(ind(7) + fmt.Sprintf("return %s;\n", outcomeCtor("Transitioned")))
					continue
				}
				b.WriteString(ind(7) + "uint8_t eligible_count = 0;\n")
				b.WriteString(ind(7) + "uint8_t selected_candidate = 0;\n")
				fallbackOrdinal := 0
				guardedOrdinal := 0
				for _, handler := range group {
					if handler.Otherwise {
						fallbackOrdinal = guardedOrdinal + 1
						continue
					}
					guardedOrdinal++
					prelude, guardExpr := l.lowerAutomataGuard(info, handler.Guard, 7, "staged")
					b.WriteString(prelude)
					b.WriteString(ind(7) + fmt.Sprintf("if (%s) {\n", guardExpr))
					b.WriteString(ind(8) + "eligible_count = (uint8_t)(eligible_count + 1);\n")
					b.WriteString(ind(8) + fmt.Sprintf("selected_candidate = %d;\n", guardedOrdinal))
					b.WriteString(ind(7) + "}\n")
				}
				b.WriteString(ind(7) + "if (eligible_count > 1) {\n")
				b.WriteString(ind(8) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
				b.WriteString(ind(8) + fmt.Sprintf("return %s;\n", outcomeCtor("Ambiguous")))
				b.WriteString(ind(7) + "}\n")
				b.WriteString(ind(7) + "if (eligible_count == 0) {\n")
				if fallbackOrdinal > 0 {
					b.WriteString(ind(8) + fmt.Sprintf("selected_candidate = %d;\n", fallbackOrdinal))
				} else {
					b.WriteString(ind(8) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
					b.WriteString(ind(8) + fmt.Sprintf("return %s;\n", outcomeCtor("Unhandled")))
				}
				b.WriteString(ind(7) + "}\n")
				b.WriteString(ind(7) + "switch (selected_candidate) {\n")
				candidateOrdinal := 0
				for _, handler := range group {
					candidateOrdinal++
					b.WriteString(ind(8) + fmt.Sprintf("case %d:\n", candidateOrdinal))
					b.WriteString(ind(9) + "{\n")
					b.WriteString(l.automataDispatchAction(info, machine, machineIndex, handler, 9, "staged", "staged_batch", "staged_count"))
					b.WriteString(ind(10) + "break;\n")
					b.WriteString(ind(9) + "}\n")
				}
				b.WriteString(ind(8) + "default:\n")
				b.WriteString(ind(9) + "batch->count = 0;\n")
				b.WriteString(ind(9) + fmt.Sprintf("return %s;\n", outcomeCtor("Unhandled")))
				b.WriteString(ind(7) + "}\n")
				b.WriteString(ind(7) + fmt.Sprintf("%s(&staged_instance);\n", normalizeName))
				b.WriteString(ind(7) + "*instance = staged_instance;\n")
				b.WriteString(ind(7) + "staged_batch.count = staged_count;\n")
				b.WriteString(ind(7) + "staged_batch.cursor = 0;\n")
				b.WriteString(ind(7) + "staged_batch.failed_index = 0;\n")
				b.WriteString(ind(7) + "*batch = staged_batch;\n")
				b.WriteString(ind(7) + "if (staged->finished) {\n")
				b.WriteString(ind(8) + fmt.Sprintf("return %s;\n", outcomeCtor("Finished")))
				b.WriteString(ind(7) + "}\n")
				b.WriteString(ind(7) + fmt.Sprintf("return %s;\n", outcomeCtor("Transitioned")))
			}
			b.WriteString(ind(6) + "default:\n")
			b.WriteString(ind(7) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
			b.WriteString(ind(7) + fmt.Sprintf("return %s;\n", outcomeCtor("Unhandled")))
			b.WriteString(ind(5) + "}\n")
		}
		b.WriteString(ind(4) + "default:\n")
		b.WriteString(ind(5) + fmt.Sprintf("concept_abort_invalid_automata_state(\"%s\", staged->current_machine, staged->current_state);\n", info.Decl.Name))
		b.WriteString(ind(5) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
		b.WriteString(ind(5) + fmt.Sprintf("return %s;\n", outcomeCtor("AlreadyFinished")))
		b.WriteString(ind(3) + "}\n")
	}
	b.WriteString(ind(2) + "default:\n")
	b.WriteString(ind(3) + fmt.Sprintf("concept_abort_invalid_automata_state(\"%s\", staged->current_machine, staged->current_state);\n", info.Decl.Name))
	b.WriteString(ind(3) + fmt.Sprintf("%s(batch);\n", evt1BatchDiscardName(info.Decl.Name)))
	b.WriteString(ind(3) + fmt.Sprintf("return %s;\n", outcomeCtor("AlreadyFinished")))
	b.WriteString(ind(1) + "}\n")
	b.WriteString("}\n\n")
	return b.String()
}

func (l *lowering) actuatorRuntimeSupport(actuatorName string) string {
	info := l.env.actuatorInfo[actuatorName]
	batchType := evt1AutomataEffectBatchCName(info.Automata.Decl.Name)
	entryType := evt1AutomataEffectEntryCName(info.Automata.Decl.Name)
	execType := evt1ActuatorLocalCName(actuatorName)
	resultType := evt1CType(Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: info.Decl.Span})
	resultCarrier := evt1CType(Type{Name: "Result", Kind: TypeApplied, TypeArgs: []Type{{Name: "void", Kind: TypeBuiltin}, info.ErrorType}, Span: info.Decl.Span})
	actuatorNames := evt1ActuatorsForAutomata(l.module, info.Automata.Decl.Name)
	failureOrdinal := 1
	for i, name := range actuatorNames {
		if name == actuatorName {
			failureOrdinal = i + 1
			break
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static void %s(%s* executor, %s mechanism) {\n", evt1ActuatorLocalInitName(actuatorName), execType, evt1CType(info.MechanismType)))
	b.WriteString(ind(1) + "executor->mechanism = mechanism;\n")
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("static %s %s(%s* batch, %s* executor) {\n", resultType, evt1ActuatorActuateName(actuatorName), batchType, execType))
	b.WriteString(ind(1) + fmt.Sprintf("%s out = {0};\n", resultType))
	b.WriteString(ind(1) + "switch (batch->state) {\n")
	b.WriteString(ind(2) + fmt.Sprintf("case %s:\n", evt1AutomataEffectBatchStateConstName(info.Automata.Decl.Name, "Vacant")))
	b.WriteString(ind(3) + fmt.Sprintf("out.outcome = %s();\n", evt1ConstructorName(evt1ActuationOutcomeTypeName, "NoBatch")))
	b.WriteString(ind(3) + "return out;\n")
	b.WriteString(ind(2) + fmt.Sprintf("case %s:\n", evt1AutomataEffectBatchStateConstName(info.Automata.Decl.Name, "Completed")))
	b.WriteString(ind(3) + fmt.Sprintf("out.outcome = %s();\n", evt1ConstructorName(evt1ActuationOutcomeTypeName, "AlreadyConsumed")))
	b.WriteString(ind(3) + "out.completedCount = batch->count;\n")
	b.WriteString(ind(3) + "return out;\n")
	b.WriteString(ind(2) + fmt.Sprintf("case %s:\n", evt1AutomataEffectBatchStateConstName(info.Automata.Decl.Name, "Failed")))
	b.WriteString(ind(3) + fmt.Sprintf("out.outcome = %s();\n", evt1ConstructorName(evt1ActuationOutcomeTypeName, "AlreadyConsumed")))
	b.WriteString(ind(3) + "out.completedCount = batch->cursor;\n")
	b.WriteString(ind(3) + "out.failedIndex = batch->failed_index;\n")
	b.WriteString(ind(3) + fmt.Sprintf("if (batch->failure_actuator == %d) {\n", failureOrdinal))
	b.WriteString(ind(4) + fmt.Sprintf("out.error = batch->failure.%s;\n", info.FailureSlot))
	b.WriteString(ind(3) + "}\n")
	b.WriteString(ind(3) + "return out;\n")
	b.WriteString(ind(2) + "default:\n")
	b.WriteString(ind(3) + "break;\n")
	b.WriteString(ind(1) + "}\n")
	b.WriteString(ind(1) + "while (batch->cursor < batch->count) {\n")
	b.WriteString(ind(2) + fmt.Sprintf("%s* item = &batch->entries[batch->cursor];\n", entryType))
	b.WriteString(ind(2) + "switch (item->tag) {\n")
	for _, mapping := range info.Decl.Mappings {
		effectDecl := l.env.effects[mapping.EffectName]
		b.WriteString(ind(3) + fmt.Sprintf("case %s:\n", evt1AutomataEffectTagConstName(info.Automata.Decl.Name, mapping.EffectName)))
		b.WriteString(ind(4) + "{\n")
		b.WriteString(ind(5) + fmt.Sprintf("%s result = %s(", resultCarrier, evt1FunctionSymbolForDecl(l.outputBase, l.env, evt1MustResolveActuatorFunction(l.env, mapping.ImplementationName, effectDecl, info))))
		var args []string
		args = append(args, "executor->mechanism")
		for _, param := range effectDecl.Params {
			args = append(args, fmt.Sprintf("item->payload.%s.%s", evt1PayloadFieldName(mapping.EffectName), param.Name))
		}
		b.WriteString(strings.Join(args, ", ") + ");\n")
		b.WriteString(ind(5) + "if (result.tag == 1 || result.is_error) {\n")
		b.WriteString(ind(6) + "batch->failed_index = batch->cursor;\n")
		b.WriteString(ind(6) + fmt.Sprintf("batch->failure_actuator = %d;\n", failureOrdinal))
		b.WriteString(ind(6) + fmt.Sprintf("batch->failure.%s = result.is_error ? result.error : result.payload.error.error;\n", info.FailureSlot))
		b.WriteString(ind(6) + fmt.Sprintf("batch->state = %s;\n", evt1AutomataEffectBatchStateConstName(info.Automata.Decl.Name, "Failed")))
		b.WriteString(ind(6) + fmt.Sprintf("out.outcome = %s();\n", evt1ConstructorName(evt1ActuationOutcomeTypeName, "Failed")))
		b.WriteString(ind(6) + "out.completedCount = batch->cursor;\n")
		b.WriteString(ind(6) + "out.failedIndex = batch->cursor;\n")
		b.WriteString(ind(6) + "out.error = result.is_error ? result.error : result.payload.error.error;\n")
		b.WriteString(ind(6) + "return out;\n")
		b.WriteString(ind(5) + "}\n")
		b.WriteString(ind(5) + "break;\n")
		b.WriteString(ind(4) + "}\n")
	}
	b.WriteString(ind(3) + "default:\n")
	b.WriteString(ind(4) + fmt.Sprintf("concept_abort_invalid_tag(\"%s\");\n", evt1AutomataEffectTagTypeCName(info.Automata.Decl.Name)))
	b.WriteString(ind(2) + "}\n")
	b.WriteString(ind(2) + "batch->cursor = (uint8_t)(batch->cursor + 1);\n")
	b.WriteString(ind(1) + "}\n")
	b.WriteString(ind(1) + fmt.Sprintf("batch->state = %s;\n", evt1AutomataEffectBatchStateConstName(info.Automata.Decl.Name, "Completed")))
	b.WriteString(ind(1) + fmt.Sprintf("out.outcome = %s();\n", evt1ConstructorName(evt1ActuationOutcomeTypeName, "Completed")))
	b.WriteString(ind(1) + "out.completedCount = batch->count;\n")
	b.WriteString(ind(1) + "return out;\n")
	b.WriteString("}\n\n")
	return b.String()
}

func (l *lowering) automataDispatchAction(info *evt1AutomataInfo, machine MachineDecl, machineIndex map[string]MachineDecl, handler TransitionDecl, indent int, instanceName, batchName, countName string) string {
	var b strings.Builder
	if batchName != "" {
		for emitIndex, emit := range handler.Emits {
			effectDecl := l.env.effects[emit.EffectName]
			for i, arg := range emit.Args {
				prelude, value, valueType := l.lowerAutomataPayloadExpr(info, arg, indent, instanceName)
				b.WriteString(prelude)
				temp := evt1PayloadFieldName(effectDecl.Name) + fmt.Sprintf("_%02d_%02d", emitIndex+1, i+1)
				b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(valueType), temp, value))
				b.WriteString(ind(indent) + fmt.Sprintf("%s.entries[%s].payload.%s.%s = %s;\n", batchName, countName, evt1PayloadFieldName(effectDecl.Name), effectDecl.Params[i].Name, temp))
			}
			b.WriteString(ind(indent) + fmt.Sprintf("%s.entries[%s].tag = %s;\n", batchName, countName, evt1AutomataEffectTagConstName(info.Decl.Name, effectDecl.Name)))
			b.WriteString(ind(indent) + fmt.Sprintf("%s = (uint8_t)(%s + 1);\n", countName, countName))
		}
	}
	switch handler.Kind {
	case TransitionGoto:
		b.WriteString(ind(indent) + fmt.Sprintf("%s->current_state = %s;\n", instanceName, evt1AutomataStateConstName(info.Decl.Name, machine.Name, handler.TargetState.StateName)))
	case TransitionPush:
		if info.ContinuationCapacity == 0 {
			b.WriteString(ind(indent) + fmt.Sprintf("concept_abort_automata_stack(\"%s\", \"push with zero continuation capacity\");\n", info.Decl.Name))
			return b.String()
		}
		targetMachine := machineIndex[handler.PushMachine]
		targetInitialState := evt1InitialStateName(targetMachine)
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s->continuation_count >= %d) {\n", instanceName, info.ContinuationCapacity))
		b.WriteString(ind(indent+1) + fmt.Sprintf("concept_abort_automata_stack(\"%s\", \"push overflow\");\n", info.Decl.Name))
		b.WriteString(ind(indent) + "}\n")
		b.WriteString(ind(indent) + fmt.Sprintf("%s->continuations[%s->continuation_count].caller_machine = %s->current_machine;\n", instanceName, instanceName, instanceName))
		b.WriteString(ind(indent) + fmt.Sprintf("%s->continuations[%s->continuation_count].resume_state = %s;\n", instanceName, instanceName, evt1AutomataStateConstName(info.Decl.Name, machine.Name, handler.Continuation.StateName)))
		b.WriteString(ind(indent) + fmt.Sprintf("%s->continuation_count = (uint8_t)(%s->continuation_count + 1);\n", instanceName, instanceName))
		b.WriteString(ind(indent) + fmt.Sprintf("%s->current_machine = %s;\n", instanceName, evt1AutomataMachineConstName(info.Decl.Name, handler.PushMachine)))
		b.WriteString(ind(indent) + fmt.Sprintf("%s->current_state = %s;\n", instanceName, evt1AutomataStateConstName(info.Decl.Name, targetMachine.Name, targetInitialState)))
	}
	return b.String()
}

func (l *lowering) lowerAutomataGuard(info *evt1AutomataInfo, expr Expr, indent int, instanceName string) (string, string) {
	prelude, value, _ := l.lowerAutomataExpr(info, expr, indent, instanceName)
	return prelude, value
}

func (l *lowering) lowerAutomataPayloadExpr(info *evt1AutomataInfo, expr Expr, indent int, instanceName string) (string, string, Type) {
	return l.lowerAutomataExpr(info, expr, indent, instanceName)
}

func (l *lowering) lowerAutomataExpr(info *evt1AutomataInfo, expr Expr, indent int, instanceName string) (string, string, Type) {
	f := &evt1FunctionLowerer{
		l:     l,
		scope: []map[string]evt1Binding{{}},
	}
	if info.Decl.Context != nil {
		contextType := info.Decl.Context.Type
		contextType.Ownership = "borrow"
		contextType.Const = true
		f.scope[0][info.Decl.Context.Name] = evt1Binding{
			cName: instanceName + "->context",
			t:     contextType,
		}
	}
	return f.lowerExpr(expr, indent)
}

func (l *lowering) structHeader(structDecl StructDecl) string {
	var b strings.Builder
	name := evt1CName(structDecl.Name)
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", name))
	for _, field := range structDecl.Fields {
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(field.Type), field.Name))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", name))
	return b.String()
}

func (l *lowering) structConstructor(structDecl StructDecl) string {
	var b strings.Builder
	typeName := evt1CName(structDecl.Name)
	ctor := evt1StructConstructorName(structDecl.Name)
	b.WriteString(fmt.Sprintf("static %s %s(", typeName, ctor))
	for i, field := range structDecl.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s %s", evt1CType(field.Type), field.Name))
	}
	b.WriteString(") {\n")
	b.WriteString(fmt.Sprintf("  %s out;\n", typeName))
	for _, field := range structDecl.Fields {
		b.WriteString(fmt.Sprintf("  out.%s = %s;\n", field.Name, field.Name))
	}
	b.WriteString("  return out;\n}\n\n")
	return b.String()
}

func (l *lowering) enumHeader(enumDecl EnumDecl) string {
	var b strings.Builder
	tagType := evt1CName(enumDecl.Name) + "_tag"
	enumType := evt1CName(enumDecl.Name)
	b.WriteString(fmt.Sprintf("typedef enum %s {\n", tagType))
	for _, variant := range enumDecl.Variants {
		b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1TagName(enumDecl.Name, variant.Name), variant.Tag))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", tagType))
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", enumType))
	b.WriteString(fmt.Sprintf("  %s tag;\n", tagType))
	b.WriteString("  union {\n")
	b.WriteString("    struct { unsigned char unused; } none;\n")
	for _, variant := range enumDecl.Variants {
		if len(variant.Payload) == 0 {
			continue
		}
		b.WriteString("    struct {\n")
		for _, field := range variant.Payload {
			b.WriteString(fmt.Sprintf("      %s %s;\n", evt1CType(field.Type), field.Name))
		}
		b.WriteString(fmt.Sprintf("    } %s;\n", evt1PayloadFieldName(variant.Name)))
	}
	b.WriteString("  } payload;\n")
	b.WriteString(fmt.Sprintf("} %s;\n\n", enumType))
	return b.String()
}

func (l *lowering) enumConstructors(enumDecl EnumDecl) string {
	var b strings.Builder
	enumType := evt1CName(enumDecl.Name)
	for _, variant := range enumDecl.Variants {
		name := evt1ConstructorName(enumDecl.Name, variant.Name)
		b.WriteString(fmt.Sprintf("static %s %s(", enumType, name))
		for i, field := range variant.Payload {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s %s", evt1CType(field.Type), field.Name))
		}
		b.WriteString(") {\n")
		b.WriteString(fmt.Sprintf("  %s out;\n", enumType))
		b.WriteString(fmt.Sprintf("  out.tag = %s;\n", evt1TagName(enumDecl.Name, variant.Name)))
		if len(variant.Payload) > 0 {
			for _, field := range variant.Payload {
				b.WriteString(fmt.Sprintf("  out.payload.%s.%s = %s;\n", evt1PayloadFieldName(variant.Name), field.Name, field.Name))
			}
		} else {
			b.WriteString("  out.payload.none.unused = 0u;\n")
		}
		b.WriteString("  return out;\n}\n\n")
	}
	return b.String()
}

func evt1StructConstructorName(structName string) string {
	return evt1CName(structName) + "_make"
}

func evt1ConstructorName(enumName, variantName string) string {
	return evt1CName(enumName) + "_make_" + evt1PayloadFieldName(variantName)
}

func evt1CType(t Type) string {
	if t.PointerTo != nil {
		base := evt1CType(*t.PointerTo)
		if t.Const {
			return "const " + base + "*"
		}
		return base + "*"
	}
	if t.isBorrow() || t.isReference() {
		base := evt1CType(t.borrowBase())
		if t.Const {
			return "const " + base + "*"
		}
		return base + "*"
	}
	if evt1IsFailureType(t) {
		return evt1FailureCName(t)
	}
	if builtin, ok := evt1BuiltinDefinition(t.Name); ok {
		return builtin.CType
	}
	return evt1CName(t.Name)
}

func evt1TypeUsed(module Module, match func(Type) bool) bool {
	var visitType func(Type) bool
	visitType = func(t Type) bool {
		if match(t) {
			return true
		}
		if t.PointerTo != nil && visitType(*t.PointerTo) {
			return true
		}
		if t.ArrayElem != nil && visitType(*t.ArrayElem) {
			return true
		}
		for _, arg := range t.TypeArgs {
			if visitType(arg) {
				return true
			}
		}
		return false
	}
	for _, structDecl := range module.Structs {
		for _, field := range structDecl.Fields {
			if visitType(field.Type) {
				return true
			}
		}
	}
	for _, enumDecl := range module.Enums {
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if visitType(field.Type) {
					return true
				}
			}
		}
	}
	for _, conceptDecl := range module.Concepts {
		for _, req := range conceptDecl.Requirements {
			switch r := req.(type) {
			case *OperationRequirement:
				if visitType(r.ReturnType) {
					return true
				}
				for _, param := range r.Params {
					if visitType(param.Type) {
						return true
					}
				}
			case *PrerequisiteRequirement:
				if visitType(r.TypeArg) {
					return true
				}
			}
		}
	}
	for _, assertion := range module.Assertions {
		if visitType(assertion.ConcreteType) {
			return true
		}
	}
	for _, templateDecl := range module.Templates {
		if visitType(templateDecl.ReturnType) {
			return true
		}
		if visitType(templateDecl.Constraint.TypeArg) {
			return true
		}
		for _, param := range templateDecl.Params {
			if visitType(param.Type) {
				return true
			}
		}
	}
	for _, fn := range module.Functions {
		if visitType(fn.ReturnType) {
			return true
		}
		for _, param := range fn.Params {
			if visitType(param.Type) {
				return true
			}
		}
	}
	for _, automataDecl := range module.Automata {
		if automataDecl.Context != nil && visitType(automataDecl.Context.Type) {
			return true
		}
	}
	return false
}

func (l *lowering) functionSymbols(fn FunctionDecl) evt1FunctionSymbols {
	var prototype strings.Builder
	cReturn := evt1CType(fn.ReturnType)
	name := evt1FunctionSymbolForDecl(l.outputBase, l.env, fn)
	prototype.WriteString(fmt.Sprintf("%s %s(", cReturn, name))
	for i, param := range fn.Params {
		if i > 0 {
			prototype.WriteString(", ")
		}
		prototype.WriteString(fmt.Sprintf("%s %s", evt1CType(param.Type), param.Name))
	}
	prototype.WriteString(");\n")
	if fn.Body == nil {
		return evt1FunctionSymbols{Prototype: prototype.String()}
	}
	lower := newEVT1FunctionLowerer(l, fn, name, false)
	return evt1FunctionSymbols{
		Prototype: prototype.String(),
		Body:      lower.lower(),
	}
}

func (l *lowering) templateInstanceBody(instance *evt1TemplateInstance) string {
	lower := newEVT1FunctionLowerer(l, instance.Function, instance.GeneratedSymbol, true)
	return lower.lower()
}

type evt1FunctionLowerer struct {
	l           *lowering
	fn          FunctionDecl
	symbol      string
	private     bool
	scope       []map[string]evt1Binding
	ownedOrder  [][]string
	liveOwners  map[string]bool
	tempCounter int
	tryHandlers []map[string]evt1LoweredTryHandler
}

type evt1LoweredTryHandler struct {
	label       string
	errorName   string
	errorType   Type
	cleanupFrom int
}

type evt1Binding struct {
	cName            string
	t                Type
	comptime         bool
	value            Value
	instanceAutomata string
	batchAutomata    string
	actuatorName     string
}

func newEVT1FunctionLowerer(l *lowering, fn FunctionDecl, symbol string, private bool) *evt1FunctionLowerer {
	scope := []map[string]evt1Binding{{}}
	ownedOrder := [][]string{{}}
	liveOwners := map[string]bool{}
	for _, param := range fn.Params {
		scope[0][param.Name] = evt1Binding{cName: param.Name, t: param.Type}
		if evt1TypeHasDrop(l.env, param.Type) && fn.Name != "Drop" {
			ownedOrder[0] = append(ownedOrder[0], param.Name)
			liveOwners[param.Name] = true
		}
	}
	return &evt1FunctionLowerer{l: l, fn: fn, symbol: symbol, private: private, scope: scope, ownedOrder: ownedOrder, liveOwners: liveOwners}
}

func (f *evt1FunctionLowerer) lower() string {
	var b strings.Builder
	prefix := ""
	if f.private {
		prefix = "static "
	}
	b.WriteString(fmt.Sprintf("%s%s %s(", prefix, evt1CType(f.fn.ReturnType), f.symbol))
	for i, param := range f.fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s %s", evt1CType(param.Type), param.Name))
	}
	b.WriteString(") {\n")
	b.WriteString(f.lowerBlock(*f.fn.Body, 1))
	b.WriteString(f.lowerCurrentScopeDrops(1))
	b.WriteString("}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerBlock(block Block, indent int) string {
	var b strings.Builder
	f.pushScope()
	for _, stmt := range block.Statements {
		b.WriteString(f.lowerStatement(stmt, indent))
	}
	b.WriteString(f.lowerCurrentScopeDrops(indent))
	f.popScope()
	return b.String()
}

func (f *evt1FunctionLowerer) lowerStatement(stmt Statement, indent int) string {
	switch s := stmt.(type) {
	case *VarDecl:
		if s.Comptime {
			value, err := evt1EvalExpr(newEVT1ComptimeState(f.l.env), f.evalScope(), s.Value)
			if err == nil {
				f.bindComptimeName(s.Name, s.Type, value)
			}
			return ""
		}
		if construct, ok := s.Value.(*StructConstructExpr); ok && construct.StructName == s.Type.Name {
			return f.lowerLocalStructConstruct(s.Type, s.Name, *construct, indent)
		}
		prelude, value, _ := f.lowerExprExpected(s.Value, s.Type, indent)
		cName := f.bindName(s.Name, s.Type)
		return prelude + ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(s.Type), cName, value)
	case *EffectsDecl:
		cName := f.bindBatchName(s.Name, s.AutomataName)
		batchType := evt1AutomataEffectBatchCName(s.AutomataName)
		return ind(indent) + fmt.Sprintf("%s %s = {0};\n", batchType, cName)
	case *ActuatorLocalDecl:
		cName := f.bindActuatorName(s.Name, s.ActuatorName)
		localType := evt1ActuatorLocalCName(s.ActuatorName)
		initName := evt1ActuatorLocalInitName(s.ActuatorName)
		prelude, target, targetType, _ := f.lowerLValue(s.Mechanism, indent)
		var b strings.Builder
		b.WriteString(prelude)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", localType, cName))
		mechanismExpr := target
		if !targetType.isBorrowLike() {
			mechanismExpr = "&" + target
		}
		b.WriteString(ind(indent) + fmt.Sprintf("%s(&%s, %s);\n", initName, cName, mechanismExpr))
		return b.String()
	case *InstanceDecl:
		cName := f.bindInstanceName(s.Name, s.AutomataName)
		instanceType := evt1AutomataRuntimeInstanceCName(s.AutomataName)
		initName := evt1AutomataRuntimeInitName(s.AutomataName)
		info := f.l.env.automataInfo[s.AutomataName]
		var b strings.Builder
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", instanceType, cName))
		if info.Decl.Context != nil {
			prelude, target, targetType, _ := f.lowerLValue(s.Context, indent)
			b.WriteString(prelude)
			contextExpr := target
			if !targetType.isBorrowLike() {
				contextExpr = "&" + target
			}
			b.WriteString(ind(indent) + fmt.Sprintf("%s(&%s, %s);\n", initName, cName, contextExpr))
			return b.String()
		}
		b.WriteString(ind(indent) + fmt.Sprintf("%s(&%s);\n", initName, cName))
		return b.String()
	case *ActuationDecl:
		info := f.l.env.actuatorInfo[s.ActuatorName]
		resultType := evt1CType(Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: s.Span})
		cName := f.bindName(s.Name, Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: s.Span})
		batchBinding, _ := scopeLookup(s.BatchName, f.scope)
		executorBinding, _ := scopeLookup(s.ExecutorName, f.scope)
		return ind(indent) + fmt.Sprintf("%s %s = %s(&%s, &%s);\n", resultType, cName, evt1ActuatorActuateName(s.ActuatorName), batchBinding.cName, executorBinding.cName)
	case *AssignStmt:
		prelude, target, _, _ := f.lowerLValue(s.Target, indent)
		rhsPrelude, value, _ := f.lowerExpr(s.Value, indent)
		replacementDrop := ""
		if name, ok := s.Target.(*NameExpr); ok {
			if binding, found := scopeLookup(name.Name, f.scope); found && evt1TypeHasDrop(f.l.env, binding.t) {
				if f.liveOwners[binding.cName] {
					replacementDrop = f.lowerDropValue(binding.t, binding.cName, indent)
				}
				f.liveOwners[binding.cName] = true
			}
		}
		return prelude + replacementDrop + rhsPrelude + ind(indent) + fmt.Sprintf("%s = %s;\n", target, value)
	case *ReturnStmt:
		if s.Value == nil {
			return f.lowerAllScopeDrops(indent) + ind(indent) + "return;\n"
		}
		prelude, value, valueType := f.lowerExprExpected(s.Value, f.fn.ReturnType, indent)
		if !f.hasLiveOwner() {
			return prelude + ind(indent) + fmt.Sprintf("return %s;\n", value)
		}
		returnTemp := f.nextTemp("return")
		return prelude + ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(valueType), returnTemp, value) + f.lowerAllScopeDrops(indent) + ind(indent) + fmt.Sprintf("return %s;\n", returnTemp)
	case *ExprStmt:
		prelude, value, valueType := f.lowerExpr(s.Value, indent)
		if valueType.Name == "void" {
			return prelude + ind(indent) + value + ";\n"
		}
		return prelude + ind(indent) + "(void)" + value + ";\n"
	case *AssertStmt:
		prelude, condition, _ := f.lowerExpr(s.Condition, indent)
		reason := fmt.Sprintf("%q", "Concept assertion failed")
		if s.Reason != nil {
			reasonPrelude, reasonExpr, _ := f.lowerExpr(s.Reason, indent)
			prelude += reasonPrelude
			reason = reasonExpr
		}
		return prelude + ind(indent) + fmt.Sprintf("if (!(%s)) { concept_panic(%s, %d, %d); }\n", condition, reason, s.Span.Line, s.Span.Column)
	case *TryStmt:
		return f.lowerTryStmt(*s, indent)
	case *StaticAssertStmt:
		return ""
	case *MatchStmt:
		return f.lowerMatchStmt(*s, indent)
	case *WhileStmt:
		return f.lowerWhileStmt(*s, indent)
	case *IfStmt:
		return f.lowerIfStmt(*s, indent)
	case *Block:
		var b strings.Builder
		b.WriteString(ind(indent) + "{\n")
		b.WriteString(f.lowerBlock(*s, indent+1))
		b.WriteString(ind(indent) + "}\n")
		return b.String()
	default:
		return ind(indent) + "/* unsupported statement */\n"
	}
}

func (f *evt1FunctionLowerer) lowerTryStmt(stmt TryStmt, indent int) string {
	endLabel := f.nextTemp("try_end")
	handlers := map[string]evt1LoweredTryHandler{}
	var b strings.Builder
	b.WriteString(ind(indent) + "{\n")
	for _, arm := range stmt.Except {
		t := evt1CanonicalType(f.l.env, arm.ErrorType)
		h := evt1LoweredTryHandler{label: f.nextTemp("except"), errorName: f.nextTemp("error"), errorType: t, cleanupFrom: len(f.ownedOrder)}
		handlers[evt1TypeIdentity(t)] = h
		b.WriteString(ind(indent+1) + fmt.Sprintf("%s %s;\n", evt1CType(t), h.errorName))
	}
	f.tryHandlers = append(f.tryHandlers, handlers)
	b.WriteString(f.lowerBlock(stmt.Body, indent+1))
	f.tryHandlers = f.tryHandlers[:len(f.tryHandlers)-1]
	b.WriteString(ind(indent+1) + "goto " + endLabel + ";\n")
	for _, arm := range stmt.Except {
		t := evt1CanonicalType(f.l.env, arm.ErrorType)
		h := handlers[evt1TypeIdentity(t)]
		b.WriteString(ind(indent) + h.label + ":\n")
		b.WriteString(ind(indent+1) + "{\n")
		f.pushScope()
		cName := f.bindName(arm.Binding, t)
		b.WriteString(ind(indent+2) + fmt.Sprintf("%s %s = %s;\n", evt1CType(t), cName, h.errorName))
		b.WriteString(f.lowerBlock(arm.Body, indent+2))
		b.WriteString(f.lowerCurrentScopeDrops(indent + 2))
		f.popScope()
		b.WriteString(ind(indent+2) + "goto " + endLabel + ";\n")
		b.WriteString(ind(indent+1) + "}\n")
	}
	b.WriteString(ind(indent) + endLabel + ": ;\n")
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerWhileStmt(stmt WhileStmt, indent int) string {
	var b strings.Builder
	if stmt.Bound != nil {
		boundValue, err := evt1EvalExpr(newEVT1ComptimeState(f.l.env), f.evalScope(), stmt.Bound)
		if err == nil {
			limitName := f.nextTemp("limit")
			iterName := f.nextTemp("iter")
			b.WriteString(ind(indent) + fmt.Sprintf("int %s = %d;\n", limitName, boundValue.IntValue))
			b.WriteString(ind(indent) + fmt.Sprintf("int %s = 0;\n", iterName))
			b.WriteString(ind(indent) + fmt.Sprintf("while (%s < %s) {\n", iterName, limitName))
			condPrelude, condExpr, _ := f.lowerExpr(stmt.Condition, indent+1)
			b.WriteString(condPrelude)
			b.WriteString(ind(indent+1) + fmt.Sprintf("if (!(%s)) { break; }\n", condExpr))
			b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s + 1;\n", iterName, iterName))
			b.WriteString(f.lowerBlock(stmt.Body, indent+1))
			b.WriteString(ind(indent) + "}\n")
			return b.String()
		}
	}
	b.WriteString(ind(indent) + "while (1) {\n")
	condPrelude, condExpr, _ := f.lowerExpr(stmt.Condition, indent+1)
	b.WriteString(condPrelude)
	b.WriteString(ind(indent+1) + fmt.Sprintf("if (!(%s)) { break; }\n", condExpr))
	b.WriteString(f.lowerBlock(stmt.Body, indent+1))
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerIfStmt(stmt IfStmt, indent int) string {
	conditionPrelude, conditionExpr, _ := f.lowerExpr(stmt.Condition, indent)
	before := f.cloneLiveOwners()
	var b strings.Builder
	b.WriteString(conditionPrelude)
	b.WriteString(ind(indent) + fmt.Sprintf("if (%s) {\n", conditionExpr))
	b.WriteString(f.lowerBlock(stmt.Then, indent+1))
	thenLive := f.cloneLiveOwners()
	f.liveOwners = before
	b.WriteString(ind(indent) + "}")
	if stmt.Else != nil {
		b.WriteString(" else {\n")
		b.WriteString(f.lowerBlock(*stmt.Else, indent+1))
		elseLive := f.cloneLiveOwners()
		for name, live := range thenLive {
			f.liveOwners[name] = live && elseLive[name]
		}
		b.WriteString(ind(indent) + "}\n")
		return b.String()
	}
	for name, live := range thenLive {
		f.liveOwners[name] = live && before[name]
	}
	b.WriteString("\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerLocalStructConstruct(targetType Type, name string, construct StructConstructExpr, indent int) string {
	structDecl := f.l.env.structs[targetType.Name]
	var b strings.Builder
	var temps []string
	for i, arg := range construct.Args {
		prelude, expr, argType := f.lowerExpr(arg, indent)
		b.WriteString(prelude)
		temp := f.nextTemp(fmt.Sprintf("init_%d", i+1))
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, expr))
		temps = append(temps, temp)
	}
	cName := f.bindName(name, targetType)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(targetType), cName))
	for i, field := range structDecl.Fields {
		b.WriteString(ind(indent) + fmt.Sprintf("%s.%s = %s;\n", cName, field.Name, temps[i]))
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerMatchStmt(stmt MatchStmt, indent int) string {
	subPrelude, subjectExpr, subjectType := f.lowerExpr(stmt.Subject, indent)
	enumDecl := f.l.env.enums[subjectType.Name]
	if decl, ok := evt1FailureEnumDecl(subjectType); ok {
		enumDecl = decl
	}
	subjectTemp := f.nextTemp("subject")
	var b strings.Builder
	b.WriteString(subPrelude)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(subjectType), subjectTemp, subjectExpr))
	b.WriteString(ind(indent) + fmt.Sprintf("switch (%s.tag) {\n", subjectTemp))
	for _, arm := range stmt.Arms {
		variant, _ := evt1LookupVariant(enumDecl, arm.Pattern.VariantName)
		tag := evt1TagName(enumDecl.Name, variant.Name)
		if evt1IsFailureType(subjectType) {
			tag = fmt.Sprintf("%d", variant.Tag)
		}
		b.WriteString(ind(indent) + fmt.Sprintf("case %s:\n", tag))
		b.WriteString(ind(indent+1) + "{\n")
		f.pushScope()
		for i, binding := range arm.Pattern.Bindings {
			field := variant.Payload[i]
			cName := f.bindName(binding, field.Type)
			b.WriteString(ind(indent+2) + fmt.Sprintf("%s %s = %s.payload.%s.%s;\n", evt1CType(field.Type), cName, subjectTemp, evt1PayloadFieldName(variant.Name), field.Name))
		}
		b.WriteString(f.lowerBlock(arm.Block, indent+2))
		b.WriteString(f.lowerCurrentScopeDrops(indent + 2))
		f.popScope()
		b.WriteString(ind(indent+2) + "break;\n")
		b.WriteString(ind(indent+1) + "}\n")
	}
	b.WriteString(ind(indent) + "default:\n")
	b.WriteString(ind(indent+1) + fmt.Sprintf("concept_abort_invalid_tag(\"%s\");\n", enumDecl.Name))
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerExpr(expr Expr, indent int) (string, string, Type) {
	_, failureConstruct := expr.(*ConstructExpr)
	if construct, ok := expr.(*ConstructExpr); ok {
		failureConstruct = evt1IsFailureType(construct.ResolvedType)
	}
	if value, ok := evt1TryEvalRuntimeExpr(f.l.env, f.evalScope(), expr); !failureConstruct && ok && evt1RuntimeTypeSafe(f.l.env, value.Type) {
		return "", evt1RenderCValue(f.l.env, value), value.Type
	}
	switch e := expr.(type) {
	case *IntLiteral:
		t, _ := evt1BuiltinType("int", e.Span)
		return "", fmt.Sprintf("%d", e.Value), t
	case *StringLiteral:
		t, _ := evt1BuiltinType("string", e.Span)
		return "", fmt.Sprintf("%q", e.Value), t
	case *BoolLiteral:
		t, _ := evt1BuiltinType("bool", e.Span)
		if e.Value {
			return "", "true", t
		}
		return "", "false", t
	case *ParenExpr:
		prelude, value, t := f.lowerExpr(e.Value, indent)
		return prelude, "(" + value + ")", t
	case *NameExpr:
		if binding, ok := scopeLookup(e.Name, f.scope); ok {
			if binding.comptime {
				return "", evt1RenderCValue(f.l.env, binding.value), binding.t
			}
			if binding.t.isReference() {
				return "", "(*" + binding.cName + ")", binding.t.borrowBase()
			}
			return "", binding.cName, binding.t
		}
		if value, ok := f.l.env.comptimeValues[e.Name]; ok {
			return "", evt1RenderCValue(f.l.env, value), value.Type
		}
		if fns, ok := f.l.env.functions[e.Name]; ok && len(fns) == 1 {
			return "", evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, fns[0]), fns[0].ReturnType
		}
		return "", e.Name, Type{}
	case *FieldExpr:
		prelude, recv, recvType := f.lowerExpr(e.Receiver, indent)
		fieldType := f.l.env.fieldSets[recvType.borrowBase().Name][e.Field]
		op := "."
		if recvType.isBorrowLike() {
			op = "->"
		}
		fieldExpr := recv + op + e.Field
		if fieldType.isReference() {
			return prelude, "(*(" + fieldExpr + "))", fieldType.borrowBase()
		}
		return prelude, fieldExpr, fieldType
	case *IndexExpr:
		return "", "/* comptime_array_index */", Type{Name: "int", Kind: TypeBuiltin}
	case *BinaryExpr:
		leftPrelude, left, leftType := f.lowerExpr(e.Left, indent)
		rightPrelude, right, _ := f.lowerExpr(e.Right, indent)
		if e.Op == "<" || e.Op == ">" || e.Op == "<=" || e.Op == ">=" || e.Op == "==" || e.Op == "!=" || e.Op == "and" || e.Op == "or" {
			boolType, _ := evt1BuiltinType("bool", e.Span)
			op := e.Op
			if op == "and" {
				op = "&&"
			}
			if op == "or" {
				op = "||"
			}
			return leftPrelude + rightPrelude, fmt.Sprintf("(%s %s %s)", left, op, right), boolType
		}
		if e.Op == "*" || e.Op == "-" {
			return leftPrelude + rightPrelude, fmt.Sprintf("(%s %s %s)", left, e.Op, right), leftType
		}
		return leftPrelude + rightPrelude, fmt.Sprintf("(%s + %s)", left, right), leftType
	case *UnaryExpr:
		prelude, value, valueType := f.lowerExpr(e.Value, indent)
		if e.Op == "not" {
			return prelude, "(!" + value + ")", Type{Name: "bool", Kind: TypeBuiltin}
		}
		return prelude, "(" + e.Op + value + ")", valueType
	case *MoveExpr:
		prelude, value, valueType := f.lowerExpr(e.Value, indent)
		if name, ok := e.Value.(*NameExpr); ok {
			if binding, found := scopeLookup(name.Name, f.scope); found && !evt1TypeCopyable(f.l.env, binding.t) {
				f.liveOwners[binding.cName] = false
			}
		}
		return prelude, value, valueType
	case *RefExpr:
		prelude, target, targetType, _ := f.lowerLValue(e.Value, indent)
		out := targetType.valueType()
		out.Ownership = "ref"
		out.Const = e.Const
		return prelude, "&(" + target + ")", out
	case *FailureExpr:
		prelude, value, carrierType := f.lowerExpr(e.Value, indent)
		carrierType = e.ResolvedType
		carrierTemp := f.nextTemp("failure")
		var b strings.Builder
		b.WriteString(prelude)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(carrierType), carrierTemp, value))
		failureTag := 1
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s.tag == %d) {\n", carrierTemp, failureTag))
		if e.Op == "!" {
			reason := "explicit Option absence escalation"
			if evt1IsResultType(carrierType) {
				reason = "explicit Result error escalation"
			}
			b.WriteString(ind(indent+1) + fmt.Sprintf("concept_panic(%q, %d, %d);\n", reason, e.Span.Line, e.Span.Column))
		} else if evt1IsResultType(carrierType) {
			errType := evt1FailureErrorType(carrierType)
			if handler, ok := f.lookupTryHandler(errType); ok {
				b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s.payload.error.error;\n", handler.errorName, carrierTemp))
				before := f.cloneLiveOwners()
				for scopeIndex := len(f.ownedOrder) - 1; scopeIndex >= handler.cleanupFrom; scopeIndex-- {
					b.WriteString(f.lowerScopeDrops(scopeIndex, indent+1))
				}
				f.liveOwners = before
				b.WriteString(ind(indent+1) + "goto " + handler.label + ";\n")
			} else {
				before := f.cloneLiveOwners()
				drops := f.lowerAllScopeDrops(indent + 1)
				f.liveOwners = before
				b.WriteString(drops)
				b.WriteString(ind(indent+1) + fmt.Sprintf("return %s(%s.payload.error.error);\n", evt1FailureConstructorName(f.fn.ReturnType, "Error"), carrierTemp))
			}
		} else {
			before := f.cloneLiveOwners()
			drops := f.lowerAllScopeDrops(indent + 1)
			f.liveOwners = before
			b.WriteString(drops)
			b.WriteString(ind(indent+1) + fmt.Sprintf("return %s();\n", evt1FailureConstructorName(f.fn.ReturnType, "None")))
		}
		b.WriteString(ind(indent) + "}\n")
		field := "some.value"
		if evt1IsResultType(carrierType) {
			if evt1FailureSuccessType(carrierType).Name == "void" {
				return b.String(), "(void)0", evt1FailureSuccessType(carrierType)
			}
			field = "ok.value"
		}
		return b.String(), carrierTemp + ".payload." + field, evt1FailureSuccessType(carrierType)
	case *CallExpr:
		if e.Callee == "discard" && len(e.Args) == 1 {
			if nameExpr, ok := e.Args[0].(*NameExpr); ok {
				if binding, found := scopeLookup(nameExpr.Name, f.scope); found && binding.batchAutomata != "" {
					return "", evt1BatchDiscardName(binding.batchAutomata) + "(&" + binding.cName + ")", Type{Name: "void", Kind: TypeBuiltin, Span: e.Span}
				}
			}
		}
		var prelude strings.Builder
		var rawArgs []string
		var argTypes []Type
		for _, arg := range e.Args {
			argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
			prelude.WriteString(argPrelude)
			argTypes = append(argTypes, argType)
			rawArgs = append(rawArgs, argExpr)
		}
		fn, ok := evt1ResolveGeneratedCall(f.l.env, e.Callee, argTypes)
		if !ok {
			return prelude.String(), "/* unresolved_call */", Type{Name: "int", Kind: TypeBuiltin}
		}
		var args []string
		for i, argType := range argTypes {
			argExpr := rawArgs[i]
			if fn.Params[i].Type.isBorrow() || fn.Params[i].Type.isReference() {
				if argType.isBorrowLike() {
					args = append(args, argExpr)
				} else {
					args = append(args, "&"+argExpr)
				}
				continue
			}
			temp := f.nextTemp("arg")
			prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
			args = append(args, temp)
		}
		return prelude.String(), evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, fn) + "(" + strings.Join(args, ", ") + ")", fn.ReturnType
	case *DispatchExpr:
		binding, _ := scopeLookup(e.InstanceName, f.scope)
		info := f.l.env.automataInfo[binding.instanceAutomata]
		signalPrelude, signalExpr, signalType := f.lowerExpr(e.Signal, indent)
		signalTemp := f.nextTemp("signal")
		var prelude strings.Builder
		prelude.WriteString(signalPrelude)
		prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(signalType), signalTemp, signalExpr))
		dispatchCall := ""
		if e.BatchName != "" {
			batchBinding, _ := scopeLookup(e.BatchName, f.scope)
			dispatchCall = fmt.Sprintf("%s(&%s, %s, &%s)", evt1AutomataRuntimeDispatchName(info.Decl.Name), binding.cName, signalTemp, batchBinding.cName)
		} else {
			dispatchCall = fmt.Sprintf("%s(&%s, %s)", evt1AutomataRuntimeDispatchName(info.Decl.Name), binding.cName, signalTemp)
		}
		return prelude.String(),
			dispatchCall,
			Type{Name: evt1AutomataDispatchOutcomeTypeName, Kind: TypeEnum, Span: e.Span}
	case *TemplateCallExpr:
		instance, ok := f.l.env.templateInstances[e.Callee+"|"+evt1TypeIdentity(evt1CanonicalType(f.l.env, e.TypeArg))]
		if !ok {
			return "", "/* missing_template_instance */", Type{Name: "int", Kind: TypeBuiltin}
		}
		var prelude strings.Builder
		var args []string
		for i, arg := range e.Args {
			argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
			prelude.WriteString(argPrelude)
			if instance.Function.Params[i].Type.isBorrow() || instance.Function.Params[i].Type.isReference() {
				if argType.isBorrowLike() {
					args = append(args, argExpr)
				} else {
					args = append(args, "&"+argExpr)
				}
				continue
			}
			temp := f.nextTemp("arg")
			prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
			args = append(args, temp)
		}
		return prelude.String(), instance.GeneratedSymbol + "(" + strings.Join(args, ", ") + ")", instance.Function.ReturnType
	case *ConstructExpr:
		enumType := Type{Name: e.EnumName, Kind: TypeEnum, Span: e.Span}
		if evt1IsFailureType(e.ResolvedType) {
			enumType = e.ResolvedType
		}
		var prelude strings.Builder
		var args []string
		for _, arg := range e.Args {
			argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
			prelude.WriteString(argPrelude)
			temp := f.nextTemp("payload")
			prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
			args = append(args, temp)
		}
		ctor := evt1ConstructorName(e.EnumName, e.VariantName)
		if evt1IsFailureType(enumType) {
			ctor = evt1FailureConstructorName(enumType, e.VariantName)
		}
		return prelude.String(), ctor + "(" + strings.Join(args, ", ") + ")", enumType
	case *StructConstructExpr:
		structType := Type{Name: e.StructName, Kind: TypeStruct, Span: e.Span}
		var prelude strings.Builder
		var args []string
		structDecl := f.l.env.structs[e.StructName]
		for i, arg := range e.Args {
			argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
			prelude.WriteString(argPrelude)
			temp := f.nextTemp("field")
			prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
			if i < len(structDecl.Fields) {
				args = append(args, temp)
			}
		}
		return prelude.String(), evt1StructConstructorName(e.StructName) + "(" + strings.Join(args, ", ") + ")", structType
	case *WithExpr:
		basePrelude, baseExpr, baseType := f.lowerExpr(e.Base, indent)
		resultTemp := f.nextTemp("record_with")
		var b strings.Builder
		b.WriteString(basePrelude)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(baseType), resultTemp, baseExpr))
		for _, update := range e.Updates {
			valuePrelude, valueExpr, _ := f.lowerExpr(update.Value, indent)
			b.WriteString(valuePrelude)
			b.WriteString(ind(indent) + fmt.Sprintf("%s.%s = %s;\n", resultTemp, update.Name, valueExpr))
		}
		return b.String(), resultTemp, baseType
	case *MatchExpr:
		subPrelude, subjectExpr, subjectType := f.lowerExpr(e.Subject, indent)
		enumDecl := f.l.env.enums[subjectType.Name]
		if decl, ok := evt1FailureEnumDecl(subjectType); ok {
			enumDecl = decl
		}
		scope := f.typeScope()
		resultType, _ := validateMatchExpr(f.l.env, scope, *e, nil, false)
		subjectTemp := f.nextTemp("match_subject")
		resultTemp := f.nextTemp("match_result")
		var b strings.Builder
		b.WriteString(subPrelude)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(subjectType), subjectTemp, subjectExpr))
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(resultType), resultTemp))
		b.WriteString(ind(indent) + fmt.Sprintf("switch (%s.tag) {\n", subjectTemp))
		for _, arm := range e.Arms {
			variant, _ := evt1LookupVariant(enumDecl, arm.Pattern.VariantName)
			tag := evt1TagName(enumDecl.Name, variant.Name)
			if evt1IsFailureType(subjectType) {
				tag = fmt.Sprintf("%d", variant.Tag)
			}
			b.WriteString(ind(indent) + fmt.Sprintf("case %s:\n", tag))
			b.WriteString(ind(indent+1) + "{\n")
			f.pushScope()
			for i, binding := range arm.Pattern.Bindings {
				field := variant.Payload[i]
				cName := f.bindName(binding, field.Type)
				b.WriteString(ind(indent+2) + fmt.Sprintf("%s %s = %s.payload.%s.%s;\n", evt1CType(field.Type), cName, subjectTemp, evt1PayloadFieldName(variant.Name), field.Name))
			}
			armPrelude, armExpr, _ := f.lowerExpr(arm.Value, indent+2)
			b.WriteString(armPrelude)
			b.WriteString(ind(indent+2) + fmt.Sprintf("%s = %s;\n", resultTemp, armExpr))
			b.WriteString(f.lowerCurrentScopeDrops(indent + 2))
			f.popScope()
			b.WriteString(ind(indent+2) + "break;\n")
			b.WriteString(ind(indent+1) + "}\n")
		}
		b.WriteString(ind(indent) + "default:\n")
		b.WriteString(ind(indent+1) + fmt.Sprintf("concept_abort_invalid_tag(\"%s\");\n", enumDecl.Name))
		b.WriteString(ind(indent) + "}\n")
		return b.String(), resultTemp, resultType
	case *IfExpr:
		conditionPrelude, conditionExpr, _ := f.lowerExpr(e.Condition, indent)
		scope := f.typeScope()
		resultType, _ := validateExpr(f.l.env, scope, e, nil, false)
		resultTemp := f.nextTemp("if_result")
		var b strings.Builder
		b.WriteString(conditionPrelude)
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(resultType), resultTemp))
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s) {\n", conditionExpr))
		thenPrelude, thenExpr, _ := f.lowerExprExpected(e.Then, resultType, indent+1)
		b.WriteString(thenPrelude)
		b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s;\n", resultTemp, thenExpr))
		b.WriteString(ind(indent) + "} else {\n")
		elsePrelude, elseExpr, _ := f.lowerExprExpected(e.Else, resultType, indent+1)
		b.WriteString(elsePrelude)
		b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s;\n", resultTemp, elseExpr))
		b.WriteString(ind(indent) + "}\n")
		return b.String(), resultTemp, resultType
	default:
		return "", "0", Type{Name: "int", Kind: TypeBuiltin}
	}
}

func (f *evt1FunctionLowerer) lowerExprExpected(expr Expr, expected Type, indent int) (string, string, Type) {
	if evt1IsFailureType(expected) {
		switch e := expr.(type) {
		case *ConstructExpr:
			e.ResolvedType = expected
		case *IfExpr:
			if c, ok := e.Then.(*ConstructExpr); ok {
				c.ResolvedType = expected
			}
			if c, ok := e.Else.(*ConstructExpr); ok {
				c.ResolvedType = expected
			}
		}
	}
	return f.lowerExpr(expr, indent)
}

func (f *evt1FunctionLowerer) lowerLValue(expr Expr, indent int) (string, string, Type, bool) {
	switch e := expr.(type) {
	case *NameExpr:
		binding, _ := scopeLookup(e.Name, f.scope)
		if binding.t.isReference() {
			return "", "(*" + binding.cName + ")", binding.t.borrowBase(), false
		}
		return "", binding.cName, binding.t, true
	case *FieldExpr:
		prelude, recv, recvType, _ := f.lowerLValue(e.Receiver, indent)
		fieldType := f.l.env.fieldSets[recvType.borrowBase().Name][e.Field]
		op := "."
		if recvType.isBorrowLike() {
			op = "->"
		}
		fieldExpr := recv + op + e.Field
		if fieldType.isReference() {
			return prelude, "(*(" + fieldExpr + "))", fieldType.borrowBase(), false
		}
		return prelude, fieldExpr, fieldType, false
	default:
		return "", "/* invalid */", Type{}, false
	}
}

func (f *evt1FunctionLowerer) pushScope() {
	f.scope = append(f.scope, map[string]evt1Binding{})
	f.ownedOrder = append(f.ownedOrder, []string{})
}

func (f *evt1FunctionLowerer) popScope() {
	for _, name := range f.ownedOrder[len(f.ownedOrder)-1] {
		delete(f.liveOwners, name)
	}
	f.scope = f.scope[:len(f.scope)-1]
	f.ownedOrder = f.ownedOrder[:len(f.ownedOrder)-1]
}

func (f *evt1FunctionLowerer) currentScope() map[string]evt1Binding {
	return f.scope[len(f.scope)-1]
}

func (f *evt1FunctionLowerer) bindName(name string, t Type) string {
	scope := f.currentScope()
	if _, exists := scope[name]; !exists {
		scope[name] = evt1Binding{cName: name, t: t}
		f.registerOwner(name, t)
		return name
	}
	unique := f.nextTemp(name)
	scope[name] = evt1Binding{cName: unique, t: t}
	f.registerOwner(unique, t)
	return unique
}

func (f *evt1FunctionLowerer) registerOwner(cName string, t Type) {
	if !evt1TypeHasDrop(f.l.env, t) {
		return
	}
	f.ownedOrder[len(f.ownedOrder)-1] = append(f.ownedOrder[len(f.ownedOrder)-1], cName)
	f.liveOwners[cName] = true
}

func (f *evt1FunctionLowerer) cloneLiveOwners() map[string]bool {
	out := map[string]bool{}
	for name, live := range f.liveOwners {
		out[name] = live
	}
	return out
}

func (f *evt1FunctionLowerer) hasLiveOwner() bool {
	for _, live := range f.liveOwners {
		if live {
			return true
		}
	}
	return false
}

func (f *evt1FunctionLowerer) lowerCurrentScopeDrops(indent int) string {
	return f.lowerScopeDrops(len(f.ownedOrder)-1, indent)
}

func (f *evt1FunctionLowerer) lowerAllScopeDrops(indent int) string {
	var b strings.Builder
	for scopeIndex := len(f.ownedOrder) - 1; scopeIndex >= 0; scopeIndex-- {
		b.WriteString(f.lowerScopeDrops(scopeIndex, indent))
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerScopeDrops(scopeIndex, indent int) string {
	var b strings.Builder
	order := f.ownedOrder[scopeIndex]
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		if !f.liveOwners[name] {
			continue
		}
		var binding evt1Binding
		for _, layer := range f.scope {
			for _, candidate := range layer {
				if candidate.cName == name {
					binding = candidate
				}
			}
		}
		b.WriteString(f.lowerDropValue(binding.t, name, indent))
		f.liveOwners[name] = false
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerDropValue(t Type, value string, indent int) string {
	if evt1IsFailureType(t) && evt1FailureNeedsDrop(f.l.env, t) {
		return ind(indent) + fmt.Sprintf("%s(%s);\n", evt1FailureDropName(t), value)
	}
	if dropFn := evt1DropFunction(f.l.env, t); dropFn != nil {
		return ind(indent) + fmt.Sprintf("%s(%s);\n", evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, *dropFn), value)
	}
	return ""
}

func (f *evt1FunctionLowerer) bindInstanceName(name, automataName string) string {
	scope := f.currentScope()
	if _, exists := scope[name]; !exists {
		scope[name] = evt1Binding{cName: name, instanceAutomata: automataName}
		return name
	}
	unique := f.nextTemp(name)
	scope[name] = evt1Binding{cName: unique, instanceAutomata: automataName}
	return unique
}

func (f *evt1FunctionLowerer) bindBatchName(name, automataName string) string {
	scope := f.currentScope()
	if _, exists := scope[name]; !exists {
		scope[name] = evt1Binding{cName: name, batchAutomata: automataName}
		return name
	}
	unique := f.nextTemp(name)
	scope[name] = evt1Binding{cName: unique, batchAutomata: automataName}
	return unique
}

func (f *evt1FunctionLowerer) bindActuatorName(name, actuatorName string) string {
	scope := f.currentScope()
	if _, exists := scope[name]; !exists {
		scope[name] = evt1Binding{cName: name, actuatorName: actuatorName}
		return name
	}
	unique := f.nextTemp(name)
	scope[name] = evt1Binding{cName: unique, actuatorName: actuatorName}
	return unique
}

func (f *evt1FunctionLowerer) bindComptimeName(name string, t Type, value Value) {
	f.currentScope()[name] = evt1Binding{cName: name, t: t, comptime: true, value: value}
}

func (f *evt1FunctionLowerer) typeScope() *evt1Scope {
	root := evt1ModuleScope(f.l.env)
	for _, layer := range f.scope {
		root = newEVT1Scope(root)
		for name, binding := range layer {
			root.declare(name, evt1ValueBinding{
				t:                binding.t,
				mutable:          true,
				comptime:         binding.comptime,
				hasValue:         binding.comptime,
				value:            binding.value,
				instanceAutomata: binding.instanceAutomata,
				batchAutomata:    binding.batchAutomata,
				actuatorName:     binding.actuatorName,
			})
		}
	}
	return root
}

func (f *evt1FunctionLowerer) evalScope() *evt1EvalScope {
	root := evt1SeedComptimeScope(f.l.env)
	for _, layer := range f.scope {
		next := newEVT1EvalScope(root)
		for name, binding := range layer {
			if binding.comptime {
				next.declare(name, evt1EvalBinding{value: binding.value, mutable: false, comptime: true})
			}
		}
		root = next
	}
	return root
}

func (f *evt1FunctionLowerer) nextTemp(prefix string) string {
	f.tempCounter++
	return fmt.Sprintf("cv_%s_%02d", prefix, f.tempCounter)
}

func (f *evt1FunctionLowerer) lookupTryHandler(t Type) (evt1LoweredTryHandler, bool) {
	key := evt1TypeIdentity(evt1CanonicalType(f.l.env, t))
	for i := len(f.tryHandlers) - 1; i >= 0; i-- {
		if h, ok := f.tryHandlers[i][key]; ok {
			return h, true
		}
	}
	return evt1LoweredTryHandler{}, false
}

func scopeLookup(name string, scopes []map[string]evt1Binding) (evt1Binding, bool) {
	for i := len(scopes) - 1; i >= 0; i-- {
		if v, ok := scopes[i][name]; ok {
			return v, true
		}
	}
	return evt1Binding{}, false
}

func evt1ResolveGeneratedCall(env *semanticEnv, name string, argTypes []Type) (FunctionDecl, bool) {
	candidates := env.functions[name]
	for _, fn := range candidates {
		if len(fn.Params) != len(argTypes) {
			continue
		}
		match := true
		for i := range fn.Params {
			expected := evt1CanonicalType(env, fn.Params[i].Type.valueType())
			actual := evt1CanonicalType(env, argTypes[i].valueType())
			if fn.Params[i].Type.isBorrow() {
				expected = evt1CanonicalType(env, fn.Params[i].Type.borrowBase())
				if argTypes[i].isBorrowLike() {
					actual = evt1CanonicalType(env, argTypes[i].borrowBase())
				}
			}
			if !expected.Equal(actual) {
				match = false
				break
			}
		}
		if match {
			return fn, true
		}
	}
	return FunctionDecl{}, false
}

func evt1RenderCValue(env *semanticEnv, value Value) string {
	switch value.Kind {
	case ValueInt:
		return fmt.Sprintf("%d", value.IntValue)
	case ValueBool:
		if value.BoolValue {
			return "true"
		}
		return "false"
	case ValueString:
		return fmt.Sprintf("%q", value.StringValue)
	case ValueStruct:
		structDecl := env.structs[value.StructName]
		parts := make([]string, 0, len(structDecl.Fields))
		for _, field := range structDecl.Fields {
			parts = append(parts, evt1RenderCValue(env, value.Fields[field.Name]))
		}
		return evt1StructConstructorName(value.StructName) + "(" + strings.Join(parts, ", ") + ")"
	case ValueEnum:
		parts := make([]string, 0, len(value.Payload))
		for _, entry := range value.Payload {
			parts = append(parts, evt1RenderCValue(env, entry))
		}
		return evt1ConstructorName(value.EnumName, value.Variant) + "(" + strings.Join(parts, ", ") + ")"
	case ValueArray:
		return "/* comptime_array */"
	default:
		return "0"
	}
}

func evt1TryEvalRuntimeExpr(env *semanticEnv, scope *evt1EvalScope, expr Expr) (Value, bool) {
	value, err := evt1EvalExpr(newEVT1ComptimeState(env), scope, expr)
	if err != nil {
		return Value{}, false
	}
	return value, true
}

func evt1RuntimeTypeSafe(env *semanticEnv, t Type) bool {
	if t.ArrayElem != nil {
		return false
	}
	if t.PointerTo != nil {
		return evt1RuntimeTypeSafe(env, *t.PointerTo)
	}
	if t.Name == "Result" && len(t.TypeArgs) == 2 {
		return evt1CanonicalType(env, t.TypeArgs[0]).Name == "void" && evt1RuntimeTypeSafe(env, t.TypeArgs[1])
	}
	for _, arg := range t.TypeArgs {
		if !evt1RuntimeTypeSafe(env, arg) {
			return false
		}
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		for _, field := range structDecl.Fields {
			if !evt1RuntimeTypeSafe(env, field.Type) {
				return false
			}
		}
	}
	if enumDecl, ok := env.enums[t.Name]; ok {
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if !evt1RuntimeTypeSafe(env, field.Type) {
					return false
				}
			}
		}
	}
	return true
}

func evt1MIRType(env *semanticEnv, t Type) Type {
	resolved, err := evt1ResolveType(env, nil, t)
	if err != nil {
		return evt1CanonicalType(env, t)
	}
	return resolved
}

func ind(level int) string {
	return strings.Repeat("  ", level)
}

func MIRText(m MIR) string {
	var lines []string
	for _, automata := range m.Automata {
		line := fmt.Sprintf("automata %s identity %s depth %d", automata.Name, automata.GraphIdentity, automata.MaxActiveDepth)
		if automata.ContextType != nil {
			line += fmt.Sprintf(" context %s:%s", automata.ContextName, automata.ContextType.String())
		}
		if automata.TopologyIdentity != "" {
			line += fmt.Sprintf(" topology %s guard %s effect %s runtime %s max_effect_batch %d", automata.TopologyIdentity, automata.GuardIdentity, automata.EffectIdentity, automata.RuntimeIdentity, automata.MaxEffectBatch)
		}
		lines = append(lines, line)
		for _, effectName := range automata.EffectSet {
			lines = append(lines, fmt.Sprintf("automata %s effect %s", automata.Name, effectName))
		}
		for _, machine := range automata.Machines {
			lines = append(lines, fmt.Sprintf("automata %s machine %s reachable=%t", automata.Name, machine.Name, machine.Reachable))
			for _, state := range machine.States {
				lines = append(lines, fmt.Sprintf("automata %s state %s::%s reachable=%t completion=%s", automata.Name, machine.Name, state.Name, state.Reachable, state.Completion))
				for _, handler := range state.Handlers {
					handlerLine := fmt.Sprintf("automata %s %s::%s %s", automata.Name, machine.Name, state.Name, handler.Signal)
					if handler.Guard != "" {
						handlerLine += " when " + handler.Guard
					}
					if handler.Otherwise {
						handlerLine += " otherwise"
					}
					handlerLine += fmt.Sprintf(" %s %s %s", handler.Kind, handler.PushMachine, handler.ContinuationState+handler.TargetState)
					lines = append(lines, handlerLine)
					for _, emit := range handler.Emits {
						lines = append(lines, fmt.Sprintf("automata %s %s::%s emit %s %s", automata.Name, machine.Name, state.Name, emit.Effect, strings.Join(emit.Args, ",")))
					}
				}
			}
		}
	}
	for _, effectDecl := range m.Effects {
		var params []string
		for _, param := range effectDecl.Params {
			params = append(params, param.Type.String()+" "+param.Name)
		}
		lines = append(lines, fmt.Sprintf("effect %s (%s)", effectDecl.Name, strings.Join(params, ", ")))
	}
	for _, tpl := range m.Templates {
		for _, op := range tpl.Operations {
			lines = append(lines, fmt.Sprintf("%s %s %s %s", tpl.Name, op.ID, op.Kind, op.Detail))
		}
	}
	for _, inst := range m.Instances {
		for _, op := range inst.Operations {
			lines = append(lines, fmt.Sprintf("%s %s %s %s", inst.ID, op.ID, op.Kind, op.Detail))
		}
	}
	for _, fn := range m.Functions {
		for _, op := range fn.Operations {
			lines = append(lines, fmt.Sprintf("%s %s %s %s", fn.Name, op.ID, op.Kind, op.Detail))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}
