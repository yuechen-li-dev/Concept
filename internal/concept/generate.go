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
	plan       *LoweringPlan
	mapDoc     map[string]any
}

func Generate(module Module, source []byte) (Outputs, error) {
	return GenerateForTarget(module, source, GenericC11Target())
}

// GenerateForTarget runs the explicit MIR -> facts -> Planner pipeline. Native
// target descriptions are planning evidence only in R4l; strict-C11 remains the
// sole emitting backend.
func GenerateForTarget(module Module, source []byte, target TargetCapabilities) (Outputs, error) {
	env, err := analyzeModule(module)
	if err != nil {
		return nil, err
	}
	if err := evt1NormalizeModuleStorageTypes(&module, env); err != nil {
		return nil, err
	}
	l := &lowering{
		module:     module,
		env:        env,
		outputBase: evt1OutputBase(module.Path),
	}
	l.mir = buildMIR(module, env)
	evt1QualifyMIRFacts(&l.mir)
	if err := evt1ValidateMIR(l.mir); err != nil {
		return nil, err
	}
	facts := NewSemanticFactSet(l.mir.SemanticFacts)
	l.plan, err = PlanModule(&l.mir, &facts, target, *env.profile, ConservativeCompilationPolicy())
	if err != nil {
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
	if len(l.mir.Layouts) > 0 {
		l.mapDoc["layouts"] = l.mir.Layouts
	}
	if len(l.mir.Streams) > 0 {
		l.mapDoc["streams"] = l.mir.Streams
	}
	if len(l.mir.SemanticFacts) > 0 {
		l.mapDoc["semantic_facts"] = l.mir.SemanticFacts
	}
	if len(l.mir.Witnesses) > 0 {
		l.mapDoc["interface_witnesses"] = l.mir.Witnesses
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
			Class:      structDecl.Class,
			Copyable:   evt1TypeCopyable(env, Type{Name: structDecl.Name, Kind: TypeStruct}),
			Movable:    !structDecl.Immovable,
			HasDrop:    evt1DropFunction(env, Type{Name: structDecl.Name, Kind: TypeStruct}) != nil,
			SourceSpan: structDecl.Span,
		}
		for _, field := range structDecl.Fields {
			visibility := ""
			if structDecl.Class {
				visibility = field.Visibility
			}
			mirStruct.Fields = append(mirStruct.Fields, MIRName{Name: field.Name, Type: evt1MIRType(env, field.Type), Visibility: visibility})
		}
		mir.Structs = append(mir.Structs, mirStruct)
	}
	for _, source := range module.Layouts {
		layout := env.layouts[source.Name]
		entry := MIRLayout{Name: layout.Name, Size: layout.Size, Alignment: layout.Alignment, SourceSpan: layout.Span}
		for _, region := range layout.Regions {
			disjoint := make([]string, 0, len(layout.Regions)-1)
			for _, other := range layout.Regions {
				if other.ID != region.ID {
					disjoint = append(disjoint, other.ID)
				}
			}
			entry.Regions = append(entry.Regions, MIRLayoutRegion{ID: region.ID, Name: region.Name, Type: evt1MIRType(env, region.Type), Offset: region.Offset, ByteExtent: region.ByteExtent, Alignment: region.Alignment, Shape: region.Type.Shape, DisjointWith: disjoint})
		}
		mir.Layouts = append(mir.Layouts, entry)
	}
	for _, source := range module.Streams {
		stream := env.streams[source.Name]
		entry := MIRStream{Name: stream.Name, LayoutName: stream.LayoutName, ZeroStorage: true, SourceSpan: stream.Span}
		for _, channel := range stream.Channels {
			entry.Channels = append(entry.Channels, MIRStreamChannel{Name: channel.Name, RegionID: channel.RegionID, Type: evt1MIRType(env, channel.Type)})
		}
		mir.Streams = append(mir.Streams, entry)
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
		resolvedDecl := info.Decl
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
		if resolvedDecl.SignalType.Name == "" {
			environment := &MIRAutomataStateEnvironment{Identity: resolvedDecl.Name + "#state", Shared: true, Explicit: true, SourceSpan: resolvedDecl.Span}
			for i, field := range resolvedDecl.StateFields {
				environment.Fields = append(environment.Fields, MIRPersistentStorage{Identity: resolvedDecl.Name + "#state." + field.Name, Name: field.Name, Type: evt1MIRType(env, field.Type), Classification: "AutomataState", Ordinal: i, Mutable: !field.Type.Const, HasDrop: evt1TypeHasDrop(env, field.Type), Provenance: evt1AutomataStorageProvenance(field.Type), SourceSpan: field.Span})
			}
			mirAutomata.StateEnvironment = environment
			mirAutomata.MachineStack = &MIRMachineStack{Capacity: evt1MachineStackCapacity, Storage: "InlineBoundedSpecializedFrames", Scheduler: "None", Continuation: "ExplicitState", SharedState: resolvedDecl.Name + "#state"}
		}
		if automataDecl.Context != nil {
			contextType := evt1MIRType(env, automataDecl.Context.Type)
			mirAutomata.ContextName = automataDecl.Context.Name
			mirAutomata.ContextType = &contextType
		}
		for _, machine := range resolvedDecl.Machines {
			mirMachine := MIRMachine{
				Name:           machine.Name,
				Initial:        machine.Initial,
				RuntimeOrdinal: info.MachineOrdinal[machine.Name],
				Reachable:      info.MachineReachable[machine.Name],
				SourceSpan:     machine.Span,
			}
			if resolvedDecl.SignalType.Name == "" {
				resultType, errorType := evt1MIRType(env, machine.ResultType), evt1MIRType(env, machine.ErrorType)
				mirMachine.ResultType, mirMachine.ErrorType = &resultType, &errorType
			}
			for i, field := range machine.Fields {
				mirMachine.Fields = append(mirMachine.Fields, MIRPersistentStorage{Identity: resolvedDecl.Name + "." + machine.Name + "#field." + field.Name, Name: field.Name, Type: evt1MIRType(env, field.Type), Classification: "MachinePersistent", Ordinal: i, Mutable: !field.Type.Const, HasDrop: evt1TypeHasDrop(env, field.Type), Provenance: evt1AutomataStorageProvenance(field.Type), SourceSpan: field.Span})
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
				if state.Body != nil {
					tmp := MIRFunction{Name: resolvedDecl.Name + "." + machine.Name + "." + state.Name}
					collectMIROps(env, state.Body, &tmp, nil)
					mirState.Operations = append(mirState.Operations, tmp.Operations...)
					mirState.Foreaches = append(mirState.Foreaches, tmp.Foreaches...)
					collectTransitionMIR(state.Body, &mirState)
					collectYieldMIR(state.Body, resolvedDecl.Name, machine.Name, state.Name, &mirState)
					collectMachineControlMIR(state.Body, machine, &mirState)
					ordinal := 0
					for _, stmt := range state.Body.Statements {
						if local, ok := stmt.(*VarDecl); ok {
							mirState.Storage = append(mirState.Storage, MIRPersistentStorage{Identity: resolvedDecl.Name + "." + machine.Name + "." + state.Name + "#local." + local.Name, Name: local.Name, Type: evt1MIRType(env, local.Type), Classification: "TransientLocal", Ordinal: ordinal, Mutable: !local.Const, HasDrop: evt1TypeHasDrop(env, local.Type), Provenance: "step", SourceSpan: local.Span})
							ordinal++
						}
					}
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
		mirConcept := MIRConcept{Name: conceptDecl.Name, TypeParam: conceptDecl.TypeParam, Interface: conceptDecl.Interface, SourceSpan: conceptDecl.Span}
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
			case *FieldRequirement:
				t := r.Type
				mirConcept.Requirements = append(mirConcept.Requirements, MIRConceptRequirement{Kind: "field", Name: r.Name, ReturnType: &t, Detail: map[bool]string{true: "readonly", false: "mutable"}[r.Readonly], SourceSpan: r.Span})
			case *CompilerAnalysisRequirement:
				var args []string
				for _, arg := range r.TypeArgs {
					args = append(args, arg.String())
				}
				for _, subject := range r.SubjectArgs {
					args = append(args, subject.Name)
				}
				for _, parameter := range r.Parameters {
					args = append(args, fmt.Sprint(parameter))
				}
				mirConcept.Requirements = append(mirConcept.Requirements, MIRConceptRequirement{
					Kind: "compiler_analysis", Name: r.Analysis, Detail: strings.Join(args, ", "), SourceSpan: r.Span,
				})
			}
		}
		mir.Concepts = append(mir.Concepts, mirConcept)
	}
	for _, witness := range evt1SortedWitnesses(env) {
		entry := MIRInterfaceWitness{ID: witness.ID, Interface: witness.Interface.Name, ConcreteType: witness.Concrete.String(), Prerequisites: append([]string{}, witness.Prerequisites...), NoAllocation: true}
		methodReqs, _, _ := evt1InterfaceRuntimeRequirements(env, witness.Interface.Name, map[string]bool{})
		for i, method := range witness.Methods {
			entry.Methods = append(entry.Methods, method.Name)
			if i < len(methodReqs) && methodReqs[i].ReturnType.Kind == TypeAsync && len(methodReqs[i].ReturnType.TypeArgs) == 1 {
				normalized := evt1SubstituteRequirement(methodReqs[i], witness.Interface.TypeParam, witness.Concrete)
				entry.AsyncMethods = append(entry.AsyncMethods, MIRAsyncWitnessMethod{Name: method.Name, Signature: evt1Signature(normalized.ReturnType, normalized.Name, normalized.Params), ReturnType: normalized.ReturnType, EventualType: normalized.ReturnType.TypeArgs[0], MachineIdentity: method.Name + "#async#machine"})
			}
		}
		for _, field := range witness.Fields {
			entry.FieldGetters = append(entry.FieldGetters, field.Name)
			if !field.Readonly {
				entry.FieldSetters = append(entry.FieldSetters, field.Name)
			}
		}
		mir.Witnesses = append(mir.Witnesses, entry)
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
			Async:     templateDecl.Async,
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
			if instance.Function.Async {
				asyncFn := instance.Function
				asyncFn.Name = instance.GeneratedSymbol
				mirInstance.Async = evt1BuildMIRAsync(asyncFn, env)
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
		mirFn.Async = evt1BuildMIRAsync(fn, env)
		if fn.MethodOf != "" {
			mirFn.MethodOf, mirFn.Visibility = fn.MethodOf, fn.Visibility
		}
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
	for _, storageType := range evt1CollectStorageTypes(module, env) {
		mir.StorageTypes = append(mir.StorageTypes, MIRStorageType{
			Type:        storageType,
			ElementType: *storageType.ArrayElem,
			StorageKind: storageType.StorageKind,
			Rank:        evt1StorageRank(storageType),
			Shape:       append([]StorageDimension{}, storageType.Shape...),
			Contiguous:  true,
			Layout:      "row-major",
			Ownership:   evt1StorageOwnership(storageType),
		})
	}
	mir.CallbackWitnesses = evt1ModuleCallbackWitnesses(module)
	return mir
}

func evt1AutomataStorageProvenance(t Type) string {
	if t.Kind == TypeDyn {
		return "explicit_dyn_witness"
	}
	if t.isReference() || evt1IsSpanType(t) {
		return "explicit_capture"
	}
	if t.isOwned() {
		return "owned_by_automata"
	}
	return "value"
}

func evt1ValidateMIR(mir MIR) error {
	if err := evt1ValidateSemanticFacts(mir.SemanticFacts); err != nil {
		return err
	}
	seenWitnesses := map[string]bool{}
	seenCallbackWitnesses := map[string]bool{}
	for _, witness := range mir.CallbackWitnesses {
		if witness.ID == "" || witness.ConcreteCallable == "" || witness.Signature.Kind != TypeCallback || witness.Adapter == "" || witness.Environment == "" || !witness.NoAllocation || seenCallbackWitnesses[witness.ID] {
			return evt1Diagnostic("CALLBACK_WITNESS_INVALID", "callback witness MIR omits its unique concrete/signature adapter or no-allocation law", Span{})
		}
		seenCallbackWitnesses[witness.ID] = true
	}
	for _, witness := range mir.Witnesses {
		if witness.ID == "" || witness.Interface == "" || witness.ConcreteType == "" || !witness.NoAllocation || seenWitnesses[witness.ID] {
			return evt1Diagnostic("INTERFACE_WITNESS_INVALID", "interface witness MIR omits its unique identity, type pair, or no-allocation law", Span{})
		}
		seenWitnesses[witness.ID] = true
		for _, entries := range [][]string{witness.Methods, witness.FieldGetters, witness.FieldSetters, witness.Prerequisites} {
			seenEntries := map[string]bool{}
			for _, entry := range entries {
				if entry == "" || seenEntries[entry] {
					return evt1Diagnostic("INTERFACE_WITNESS_INVALID", fmt.Sprintf("interface witness %s contains an empty or duplicate runtime entry", witness.ID), Span{})
				}
				seenEntries[entry] = true
			}
		}
		for _, method := range witness.AsyncMethods {
			if method.Name == "" || method.Signature == "" || method.ReturnType.Kind != TypeAsync || len(method.ReturnType.TypeArgs) != 1 || !method.ReturnType.TypeArgs[0].Equal(method.EventualType) || method.MachineIdentity == "" {
				return evt1Diagnostic("DYN_ASYNC_WITNESS_INVALID", "async interface witness MIR omits its normalized return or concrete machine identity", Span{})
			}
		}
	}
	regions := map[string]MIRLayoutRegion{}
	for _, layout := range mir.Layouts {
		if layout.Name == "" || layout.Size < 1 || layout.Alignment < 1 || len(layout.Regions) == 0 {
			return evt1Diagnostic("CV4593", "MIR layout omits required geometry", layout.SourceSpan)
		}
		for _, region := range layout.Regions {
			if region.ID == "" || region.ByteExtent < 1 || region.Alignment < 1 || region.Offset%region.Alignment != 0 {
				return evt1Diagnostic("CV4593", "MIR layout region omits required geometry", layout.SourceSpan)
			}
			regions[region.ID] = region
		}
	}
	for _, stream := range mir.Streams {
		if stream.Name == "" || stream.LayoutName == "" || !stream.ZeroStorage {
			return evt1Diagnostic("CV4594", "MIR stream omits its zero-storage layout mapping", stream.SourceSpan)
		}
		for _, channel := range stream.Channels {
			if channel.Name == "" || regions[channel.RegionID].ID == "" {
				return evt1Diagnostic("CV4594", "MIR stream channel omits a valid region identity", stream.SourceSpan)
			}
		}
	}
	for _, automata := range mir.Automata {
		if automata.StateEnvironment == nil {
			continue
		}
		if automata.StateEnvironment.Identity != automata.Name+"#state" || !automata.StateEnvironment.Explicit || !automata.StateEnvironment.Shared {
			return evt1Diagnostic("AUTOMATA_MIR_INVALID", "automata MIR omits its explicit shared state environment", automata.SourceSpan)
		}
		if automata.MachineStack == nil || automata.MachineStack.Capacity != evt1MachineStackCapacity || automata.MachineStack.Storage != "InlineBoundedSpecializedFrames" || automata.MachineStack.Scheduler != "None" || automata.MachineStack.Continuation != "ExplicitState" || automata.MachineStack.SharedState != automata.StateEnvironment.Identity {
			return evt1Diagnostic("MACHINE_STACK_MIR_INVALID", "canonical automata MIR omits or corrupts its bounded explicit-state machine stack", automata.SourceSpan)
		}
		seenMachines := map[string]bool{}
		seenStorage := map[string]bool{}
		for _, field := range automata.StateEnvironment.Fields {
			if field.Identity == "" || field.Type.Name == "" || field.Classification != "AutomataState" || seenStorage[field.Identity] {
				return evt1Diagnostic("AUTOMATA_MIR_INVALID", "automata state MIR contains missing or duplicate storage evidence", field.SourceSpan)
			}
			seenStorage[field.Identity] = true
			if field.Type.isOwned() && !field.HasDrop {
				return evt1Diagnostic("AUTOMATA_MIR_INVALID", "owned automata state MIR omits its drop responsibility", field.SourceSpan)
			}
			if (field.Type.isReference() || field.Type.Kind == TypeDyn || evt1IsSpanType(field.Type)) && field.Provenance == "" {
				return evt1Diagnostic("AUTOMATA_MIR_INVALID", "captured reference-like state omits provenance", field.SourceSpan)
			}
		}
		for _, machine := range automata.Machines {
			if machine.Name == "" || seenMachines[machine.Name] || len(machine.States) == 0 {
				return evt1Diagnostic("MACHINE_MIR_INVALID", "machine MIR is duplicate or omits current state", machine.SourceSpan)
			}
			seenMachines[machine.Name] = true
			stateNames := map[string]bool{}
			initialStates := 0
			for _, state := range machine.States {
				if state.Name == "" || stateNames[state.Name] {
					return evt1Diagnostic("MACHINE_MIR_INVALID", "machine MIR contains a missing or duplicate state identity", state.SourceSpan)
				}
				stateNames[state.Name] = true
				if state.Initial {
					initialStates++
				}
			}
			if initialStates != 1 || !machine.States[0].Initial {
				return evt1Diagnostic("MACHINE_MIR_INVALID", "machine MIR must identify its first declared state as the sole initial state", machine.SourceSpan)
			}
			for _, field := range machine.Fields {
				if field.Identity == "" || field.Classification != "MachinePersistent" || seenStorage[field.Identity] {
					return evt1Diagnostic("MACHINE_MIR_INVALID", "machine field MIR has invalid persistence classification", field.SourceSpan)
				}
				seenStorage[field.Identity] = true
				if field.Type.isOwned() && !field.HasDrop {
					return evt1Diagnostic("MACHINE_MIR_INVALID", "owned machine field MIR omits its drop responsibility", field.SourceSpan)
				}
			}
			for _, state := range machine.States {
				for _, y := range state.Yields {
					if y.MachineIdentity != automata.Name+"."+machine.Name || y.StateIdentity != automata.Name+"."+machine.Name+"."+state.Name || y.CleanupEdge != "TransientBeforeStepReturn" || !y.PreserveCurrentState || !y.PreservePersistentStorage || y.WriteResult || y.MarkComplete {
						return evt1Diagnostic("YIELD_MIR_INVALID", "yield MIR must drop transients, preserve persistent state, and leave result/completion untouched", y.SourceSpan)
					}
				}
				for _, each := range state.Foreaches {
					if err := validateForeachMIR(each); err != nil {
						return err
					}
				}
				for _, local := range state.Storage {
					if local.Identity == "" || local.Classification != "TransientLocal" || seenStorage[local.Identity] {
						return evt1Diagnostic("MACHINE_MIR_INVALID", "state local MIR has invalid transient classification", local.SourceSpan)
					}
					seenStorage[local.Identity] = true
				}
				for _, op := range state.Operations {
					if op.Kind == "state_transition" && !stateNames[op.Detail] {
						return evt1Diagnostic("MACHINE_MIR_INVALID", "state transition MIR targets an unknown local state", op.SourceSpan)
					}
				}
				for _, match := range state.TransitionMatches {
					if match.Scrutinee == "" || !match.Exhaustive || match.NoMatchPolicy != "Panic" || match.CleanupEdge != "TransientBeforeStateUpdate" || len(match.Arms) == 0 {
						return evt1Diagnostic("TRANSITION_MATCH_MIR_INVALID", "transition match MIR omits categorical or cleanup policy", match.SourceSpan)
					}
					seenPatterns := map[string]bool{}
					for i, arm := range match.Arms {
						if arm.Pattern == "" || seenPatterns[arm.Pattern] || arm.DeclarationOrder != i || !stateNames[arm.TargetState] {
							return evt1Diagnostic("TRANSITION_MATCH_MIR_INVALID", "transition match MIR contains an invalid arm, order, or local target", arm.SourceSpan)
						}
						seenPatterns[arm.Pattern] = true
					}
				}
				for _, decision := range state.TransitionDecisions {
					if len(decision.Candidates) == 0 || (decision.ScoreType.Name != "int" && decision.ScoreType.Name != "float") || decision.TiePolicy != "DeclarationOrderFirstMax" || decision.NoEnabledPolicy != "Panic" || decision.CleanupEdge != "TransientBeforeStateUpdate" {
						return evt1Diagnostic("TRANSITION_DECIDE_MIR_INVALID", "transition decide MIR omits hardmax, score, panic, or cleanup policy", decision.SourceSpan)
					}
					for i, candidate := range decision.Candidates {
						if candidate.Score == "" || candidate.DeclarationOrder != i || !stateNames[candidate.TargetState] {
							return evt1Diagnostic("TRANSITION_DECIDE_MIR_INVALID", "transition decide MIR contains an invalid candidate, order, score, or local target", candidate.SourceSpan)
						}
					}
				}
				for _, inference := range state.TransitionInferences {
					if len(inference.Candidates) == 0 || inference.ScoreType != "float" || inference.Normalization != "StableSoftMax" || inference.Policy != "HardMax" || inference.NoEnabledPolicy != "Panic" || inference.NaNPolicy != "Panic" || inference.InfinityPolicy != "EqualPositiveInfinityElseNegativeInfinityZero" || inference.CleanupEdge != "TransientBeforeStateUpdate" {
						return evt1Diagnostic("TRANSITION_INFER_MIR_INVALID", "transition infer MIR omits normalization, policy, panic, or cleanup semantics", inference.SourceSpan)
					}
					for i, candidate := range inference.Candidates {
						if candidate.Score == "" || candidate.DeclarationOrder != i || !stateNames[candidate.TargetState] {
							return evt1Diagnostic("TRANSITION_INFER_MIR_INVALID", "transition infer MIR contains an invalid candidate, order, score, or local target", candidate.SourceSpan)
						}
					}
				}
			}
		}
	}
	allFunctions := append(append([]MIRFunction{}, mir.Functions...), mir.ComptimeFns...)
	for _, fn := range allFunctions {
		for _, callable := range fn.Callables {
			if callable.Identity == "" || callable.Environment.Identity == "" || callable.Dispatch != "DirectCallable" || !callable.NoAllocation || callable.Environment.Alignment < 1 {
				return evt1Diagnostic("CALLABLE_MIR_INVALID", fmt.Sprintf("callable MIR in %s omits identity, layout, direct dispatch, or no-allocation facts", fn.Name), callable.SourceSpan)
			}
			seenCapture := map[string]bool{}
			for i, capture := range callable.Environment.Fields {
				if capture.Source == "" || capture.FieldIdentity == "" || capture.RegionIdentity == "" || !capture.NoAllocation || capture.EvaluationOrder != i || capture.Type.Name == "" || capture.Provenance == "" || seenCapture[capture.Source] {
					return evt1Diagnostic("CALLABLE_MIR_INVALID", "capture binding MIR is incomplete, duplicated, or out of order", callable.SourceSpan)
				}
				seenCapture[capture.Source] = true
			}
		}
		if fn.Async != nil {
			a := fn.Async
			if a.Identity != fn.Name+"#async" || a.MachineIdentity == "" || a.StateIdentity == "" || a.FrameStorage != "Inline" || a.ContinuationStrategy != "ExplicitGeneratedState" || a.ChildInvocation != "MachinePush" || a.Scheduler != "None" || a.SavedPC != "None" || len(a.GeneratedStates) < len(a.AwaitPoints)+2 {
				return evt1Diagnostic("AWAIT_MIR_INVALID", fmt.Sprintf("async MIR in %s omits generated-machine invariants", fn.Name), fn.SourceSpan)
			}
			if err := evt1ValidateAsyncGraph(a, fn.SourceSpan); err != nil {
				return err
			}
			for i, await := range a.AwaitPoints {
				if await.Index != i || await.Continuation == "" || await.Evaluation != "ExactlyOnce" || await.ChildPush != "BoundedMachineFramePush" || await.OutcomeConsume != "ExactlyOnce" || await.OperandType.Kind != TypeAsync {
					return evt1Diagnostic("AWAIT_MIR_INVALID", fmt.Sprintf("await point %d in %s is malformed", i, fn.Name), await.SourceSpan)
				}
			}
		}
		for _, each := range fn.Foreaches {
			if err := validateForeachMIR(each); err != nil {
				return err
			}
		}
		for _, inference := range fn.Inferences {
			if inference.CandidateType.Name == "" || len(inference.Candidates) == 0 || inference.ScoreType != "float" || inference.Normalization != "StableSoftMax" || inference.Temperature != 1.0 || inference.NoEnabledPolicy != "Panic" || inference.NaNPolicy != "Panic" || inference.InfinityPolicy != "EqualPositiveInfinityElseNegativeInfinityZero" {
				return evt1Diagnostic("INFER_MIR_INVALID", fmt.Sprintf("infer MIR in %s is incomplete", fn.Name), inference.SourceSpan)
			}
			for i, candidate := range inference.Candidates {
				if candidate.Identity == "" || candidate.Score == "" || candidate.DeclarationOrder != i {
					return evt1Diagnostic("INFER_MIR_INVALID", fmt.Sprintf("infer MIR candidate in %s is invalid", fn.Name), candidate.SourceSpan)
				}
			}
		}
		for _, tensor := range fn.TensorOperations {
			if err := validateTensorMIRSemantic(tensor); err != nil {
				return evt1Diagnostic("CV4626", fmt.Sprintf("Tensor MIR in %s is invalid: %s", fn.Name, err), tensor.SourceSpan)
			}
		}
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
		for _, operation := range fn.Operations {
			if operation.Kind == "callable_literal" || operation.Kind == "capture_binding" || operation.Kind == "callable_invoke" || operation.Kind == "callback_invoke" {
				if operation.Type == "" || operation.Detail == "" || !operation.NoAllocation {
					return evt1Diagnostic("CALLABLE_MIR_INVALID", fmt.Sprintf("MIR %s operation %s omits callable identity or no-allocation evidence", operation.Kind, operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "dyn_make" {
				if operation.Type == "" || operation.Detail == "" || operation.Provenance == "" || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
					return evt1Diagnostic("DYN_MIR_INVALID", fmt.Sprintf("MIR dyn construction %s omits witness, provenance, or storage-neutrality facts", operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "dyn_call" || operation.Kind == "dyn_field_get" || operation.Kind == "dyn_field_set" {
				if operation.Type == "" || operation.Detail == "" || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
					return evt1Diagnostic("DYN_MIR_INVALID", fmt.Sprintf("MIR %s operation %s omits interface or storage-neutrality facts", operation.Kind, operation.ID), operation.SourceSpan)
				}
				if operation.AsyncConstructor && (operation.Kind != "dyn_call" || !strings.HasPrefix(operation.ReturnType, "Async<") || operation.Evaluation != "ExactlyOnce" || operation.OutcomeTransfer != "MoveOnce") {
					return evt1Diagnostic("DYN_ASYNC_RESULT_OWNERSHIP_INVALID", fmt.Sprintf("MIR dyn async constructor %s omits exactly-once move evidence", operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "tensor_inline_storage" {
				if operation.TensorBackingKind != TensorBackingInline || operation.TargetStorageKind != StorageNDArray || operation.TargetRank < 1 || len(operation.TargetShape) != operation.TargetRank || operation.RegionID == "" || operation.Provenance != "local" || operation.Mutability == "" || operation.Alignment < 1 || !operation.Contiguous || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
					return evt1Diagnostic("CV4626", fmt.Sprintf("MIR inline tensor storage %s omits fixed backing facts", operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "tensor_view" {
				if operation.TensorBackingKind == "" || operation.TargetRank < 1 || len(operation.TargetShape) != operation.TargetRank || operation.RegionID == "" || operation.Provenance == "" || operation.Mutability == "" || operation.Alignment < 1 || !operation.Contiguous || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer || !operation.SameBackingRegion {
					return evt1Diagnostic("CV4626", fmt.Sprintf("MIR tensor view %s omits storage facts", operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "span_from_region" || operation.Kind == "span_to_readonly" || operation.Kind == "span_subregion" || operation.Kind == "span_index" {
				if operation.ElementType == nil || operation.RegionID == "" || operation.Length == "" || operation.Provenance == "" || operation.Mutability == "" || operation.Alignment < 1 || operation.BoundsCheck == "" || !operation.Contiguous || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer || !operation.SameBackingRegion {
					return evt1Diagnostic("CV4608", fmt.Sprintf("MIR %s operation %s omits bounded borrowed-region facts", operation.Kind, operation.ID), operation.SourceSpan)
				}
				if operation.Kind != "span_index" && (operation.BaseOffset == "" || operation.ByteExtentExpression == "") {
					return evt1Diagnostic("CV4608", fmt.Sprintf("MIR %s operation %s omits interval extent facts", operation.Kind, operation.ID), operation.SourceSpan)
				}
				if operation.Kind == "span_subregion" && operation.BoundsCheck == "" {
					return evt1Diagnostic("CV4608", fmt.Sprintf("MIR span subregion %s omits interval validation", operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "layout_bind" || operation.Kind == "stream_bind" {
				if operation.LayoutName == "" || operation.Provenance == "" || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer || !operation.SameBackingRegion {
					return evt1Diagnostic("CV4594", fmt.Sprintf("MIR %s operation %s omits zero-storage binding facts", operation.Kind, operation.ID), operation.SourceSpan)
				}
				continue
			}
			if operation.Kind == "region_projection" && (operation.RegionID == "" || !operation.SameBackingRegion) {
				return evt1Diagnostic("CV4593", fmt.Sprintf("MIR region projection %s omits identity or backing fact", operation.ID), operation.SourceSpan)
			}
			if operation.Kind != "bind_storage" {
				continue
			}
			if operation.SourceStorageKind == "" || operation.TargetStorageKind == "" || operation.TargetRank < 1 || len(operation.TargetShape) != operation.TargetRank || operation.CountCheck == "" || operation.Provenance == "" || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
				return evt1Diagnostic("CV4568", fmt.Sprintf("MIR bind_storage operation %s omits required storage/provenance facts", operation.ID), operation.SourceSpan)
			}
		}
	}
	return nil
}

func validateForeachMIR(each MIRForeach) error {
	if each.Source == "" || each.SourceKind == "" || each.SourceType.Name == "" || each.IteratorType.Name == "" || each.ElementType.Name == "" || each.ItemType.Name == "" || each.ItemName == "" || each.SourceEvaluation != "ExactlyOnce" || each.IteratorCleanup != "Deterministic" || each.ItemCleanup != "PerIteration" || !each.NoAllocation || !each.NoOwnershipTransfer {
		return evt1Diagnostic("FOREACH_MIR_INVALID", "foreach MIR omits protocol, evaluation, cleanup, or allocation facts", each.SourceSpan)
	}
	if each.IteratorStrategy != "ExplicitProtocol" && each.IteratorStrategy != "BuiltinInlineIterator" {
		return evt1Diagnostic("FOREACH_MIR_INVALID", "foreach MIR has an unknown iterator strategy", each.SourceSpan)
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
			case *WhileStmt:
				visitBlock(s.Body)
			case *ForeachStmt:
				visitExpr(s.Source)
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
	visitBlock(*fn.Body)
	var cleanups []MIRCleanup
	for i := len(declarationOrder) - 1; i >= 0; i-- {
		name := declarationOrder[i]
		dropName := ""
		if evt1IsFailureType(types[name]) && evt1FailureNeedsDrop(env, types[name]) {
			dropName = evt1FailureDropName(types[name])
		} else if types[name].Kind == TypeCallable && types[name].CallableHasDrop {
			dropName = "DropCallableEnvironment"
		} else if dropFn := evt1DropFunction(env, types[name]); dropFn != nil {
			dropName = dropFn.Name
		} else if types[name].ArrayElem != nil && evt1StorageElementHasDrop(env, *types[name].ArrayElem) {
			dropName = "DropElementsReverse"
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
			if s.InlineTensor != nil {
				facts := s.InlineTensor.Facts
				fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "tensor_inline_storage", Type: s.InlineTensor.BackingType.String(), Detail: s.InlineTensor.BackingID + " init=" + s.InlineTensor.InitializerKind, TargetStorageKind: StorageNDArray, TargetRank: len(s.InlineTensor.BackingType.Shape), TargetShape: append([]StorageDimension{}, s.InlineTensor.BackingType.Shape...), Mutability: facts.Mutability, Provenance: facts.Provenance, RegionID: facts.RegionID, BaseOffset: "0", Alignment: facts.Alignment, TensorBackingKind: facts.BackingKind, Contiguous: true, NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SourceSpan: s.Span})
				fn.Operations = append(fn.Operations, MIROperation{ID: fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1), Kind: "tensor_view", Type: s.Type.String(), Detail: s.Name + " over " + s.InlineTensor.BackingID, TargetRank: facts.Rank, TargetShape: append([]StorageDimension{}, facts.Shape...), Mutability: facts.Mutability, Provenance: facts.Provenance, RegionID: facts.RegionID, BaseOffset: facts.BaseOffset, Alignment: facts.Alignment, TensorBackingKind: facts.BackingKind, Contiguous: true, NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SameBackingRegion: true, SourceSpan: s.Span})
				collectExprMIROps(env, s.Value, fn, templateInfo)
				continue
			}
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
			if s.Tensor != nil {
				fn.TensorOperations = append(fn.TensorOperations, *s.Tensor)
			}
			op := MIROperation{ID: id, Kind: "assign", Type: exprLabel(s.Target), Detail: exprLabel(s.Target), SourceSpan: s.Span}
			if field, ok := s.Target.(*FieldExpr); ok && field.DynInterface != "" {
				op.Kind, op.Type, op.Detail = "dyn_field_set", field.DynInterface, field.Field
				op.NoCopy, op.NoAllocation, op.NoOwnershipTransfer = true, true, true
				fn.Operations = append(fn.Operations, op)
				collectExprMIROps(env, field.Receiver, fn, templateInfo)
			} else {
				fn.Operations = append(fn.Operations, op)
				collectExprMIROps(env, s.Target, fn, templateInfo)
			}
			collectExprMIROps(env, s.Value, fn, templateInfo)
		case *TransitionStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "state_transition", Detail: s.Target, SourceSpan: s.Span})
		case *YieldStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "yield_state", Detail: "preserve-current-state", NoAllocation: true, NoCopy: true, NoOwnershipTransfer: true, SourceSpan: s.Span})
		case *PushMachineStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "push_machine", Detail: s.Machine + " goto " + s.ResumeState, NoAllocation: true, NoCopy: true, NoOwnershipTransfer: true, SourceSpan: s.Span})
		case *MachineCompleteStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "complete_" + s.Kind, Detail: "semantic-pop", NoAllocation: true, SourceSpan: s.Span})
			if s.Value != nil {
				collectExprMIROps(env, s.Value, fn, templateInfo)
			}
		case *ForeachStmt:
			mode := "Value"
			if s.ItemType.Ownership == "ref" {
				mode = "Ref"
				if s.ItemType.Const {
					mode = "RefConst"
				}
			}
			strategy := "ExplicitProtocol"
			if s.SourceKind != "custom" {
				strategy = "BuiltinInlineIterator"
			}
			fn.Foreaches = append(fn.Foreaches, MIRForeach{Source: evt1ExprIdentity(s.Source), SourceKind: s.SourceKind, SourceType: s.SourceType, IteratorType: s.IteratorType, ElementType: s.ElementType, ItemType: s.ItemType, ItemName: s.ItemName, IterationMode: mode, IteratorStrategy: strategy, SourceEvaluation: "ExactlyOnce", IteratorCleanup: "Deterministic", ItemCleanup: "PerIteration", NoAllocation: true, NoOwnershipTransfer: true, SourceSpan: s.Span})
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "foreach", Type: s.SourceType.String(), Detail: strategy, NoAllocation: true, NoCopy: true, NoOwnershipTransfer: true, SourceSpan: s.Span})
			collectExprMIROps(env, s.Source, fn, templateInfo)
			collectMIROps(env, &s.Body, fn, templateInfo)
		case *TransitionMatchStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "transition_match", Detail: fmt.Sprintf("%d exhaustive arm(s)", len(s.Arms)), SourceSpan: s.Span})
			collectExprMIROps(env, s.Subject, fn, templateInfo)
		case *TransitionDecideStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "transition_decide", Type: s.ScoreType.String(), Detail: fmt.Sprintf("%d declaration-order candidate(s)", len(s.Candidates)), SourceSpan: s.Span})
			for _, candidate := range s.Candidates {
				if candidate.Guard != nil {
					collectExprMIROps(env, candidate.Guard, fn, templateInfo)
				}
				collectExprMIROps(env, candidate.Score, fn, templateInfo)
			}
		case *TransitionInferStmt:
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "transition_infer", Type: "float", Detail: s.Policy, SourceSpan: s.Span})
			for _, candidate := range s.Candidates {
				if candidate.Guard != nil {
					collectExprMIROps(env, candidate.Guard, fn, templateInfo)
				}
				collectExprMIROps(env, candidate.Score, fn, templateInfo)
			}
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

func collectYieldMIR(block *Block, automata, machine, state string, out *MIRState) {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *YieldStmt:
			out.Yields = append(out.Yields, MIRYield{MachineIdentity: automata + "." + machine, StateIdentity: automata + "." + machine + "." + state, CleanupEdge: "TransientBeforeStepReturn", PreserveCurrentState: true, PreservePersistentStorage: true, WriteResult: false, MarkComplete: false, SourceSpan: s.Span})
		case *IfStmt:
			collectYieldMIR(&s.Then, automata, machine, state, out)
			if s.Else != nil {
				collectYieldMIR(s.Else, automata, machine, state, out)
			}
		case *MatchStmt:
			for i := range s.Arms {
				collectYieldMIR(&s.Arms[i].Block, automata, machine, state, out)
			}
		case *TryStmt:
			collectYieldMIR(&s.Body, automata, machine, state, out)
			for i := range s.Except {
				collectYieldMIR(&s.Except[i].Body, automata, machine, state, out)
			}
		case *WhileStmt:
			collectYieldMIR(&s.Body, automata, machine, state, out)
		case *ForeachStmt:
			collectYieldMIR(&s.Body, automata, machine, state, out)
		case *Block:
			collectYieldMIR(s, automata, machine, state, out)
		}
	}
}

func collectTransitionMIR(block *Block, state *MIRState) {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *TransitionMatchStmt:
			entry := MIRTransitionMatch{Scrutinee: evt1ExprIdentity(s.Subject), Exhaustive: true, NoMatchPolicy: "Panic", CleanupEdge: "TransientBeforeStateUpdate", SourceSpan: s.Span}
			for i, arm := range s.Arms {
				entry.Arms = append(entry.Arms, MIRTransitionMatchArm{Pattern: arm.Pattern.EnumName + "::" + arm.Pattern.VariantName, PayloadBindings: append([]string{}, arm.Pattern.Bindings...), TargetState: arm.Target, DeclarationOrder: i, SourceSpan: arm.Span})
			}
			state.TransitionMatches = append(state.TransitionMatches, entry)
		case *TransitionDecideStmt:
			entry := MIRTransitionDecide{ScoreType: s.ScoreType, TiePolicy: "DeclarationOrderFirstMax", NoEnabledPolicy: "Panic", CleanupEdge: "TransientBeforeStateUpdate", SourceSpan: s.Span}
			for _, candidate := range s.Candidates {
				mirCandidate := MIRDecisionCandidate{TargetState: candidate.Identity, Score: evt1ExprIdentity(candidate.Score), DeclarationOrder: candidate.DeclarationOrder, SourceSpan: candidate.Span}
				if candidate.Guard != nil {
					mirCandidate.Guard = evt1ExprIdentity(candidate.Guard)
				}
				entry.Candidates = append(entry.Candidates, mirCandidate)
			}
			state.TransitionDecisions = append(state.TransitionDecisions, entry)
		case *TransitionInferStmt:
			entry := MIRTransitionInfer{ScoreType: "float", Normalization: "StableSoftMax", Policy: s.Policy, NoEnabledPolicy: "Panic", NaNPolicy: "Panic", InfinityPolicy: "EqualPositiveInfinityElseNegativeInfinityZero", CleanupEdge: "TransientBeforeStateUpdate", SourceSpan: s.Span}
			for _, candidate := range s.Candidates {
				mirCandidate := MIRDecisionCandidate{TargetState: candidate.Identity, Score: evt1ExprIdentity(candidate.Score), DeclarationOrder: candidate.DeclarationOrder, SourceSpan: candidate.Span}
				if candidate.Guard != nil {
					mirCandidate.Guard = evt1ExprIdentity(candidate.Guard)
				}
				entry.Candidates = append(entry.Candidates, mirCandidate)
			}
			state.TransitionInferences = append(state.TransitionInferences, entry)
		case *IfStmt:
			collectTransitionMIR(&s.Then, state)
			if s.Else != nil {
				collectTransitionMIR(s.Else, state)
			}
		case *MatchStmt:
			for i := range s.Arms {
				collectTransitionMIR(&s.Arms[i].Block, state)
			}
		case *TryStmt:
			collectTransitionMIR(&s.Body, state)
			for i := range s.Except {
				collectTransitionMIR(&s.Except[i].Body, state)
			}
		case *WhileStmt:
			collectTransitionMIR(&s.Body, state)
		case *ForeachStmt:
			collectTransitionMIR(&s.Body, state)
		case *Block:
			collectTransitionMIR(s, state)
		}
	}
}

func collectExprMIROps(env *semanticEnv, expr Expr, fn *MIRFunction, templateInfo *evt1TemplateInfo) {
	id := fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1)
	switch e := expr.(type) {
	case *CallableExpr:
		fn.Callables = append(fn.Callables, evt1CallableMIR(env, e))
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "callable_literal", Type: e.Identity, Detail: e.EnvironmentID, Evaluation: "CapturesExactlyOnceLeftToRight", NoAllocation: true, SourceSpan: e.Span})
		for _, capture := range e.Captures {
			detail := string(capture.Kind) + " " + capture.Name
			if capture.Source != nil {
				detail += " = " + evt1ExprIdentity(capture.Source)
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: fmt.Sprintf("%s.%02d", fn.Name, len(fn.Operations)+1), Kind: "capture_binding", Type: capture.Type.String(), Detail: detail, Provenance: capture.Provenance, Evaluation: "ExactlyOnce", NoAllocation: true, SourceSpan: capture.Span})
			if capture.Source != nil {
				collectExprMIROps(env, capture.Source, fn, templateInfo)
			}
		}
	case *AwaitExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "await_point", Type: e.ResultType.String(), Detail: "explicit_generated_state+machine_push+outcome_consume", NoAllocation: true, SourceSpan: e.Span})
		collectExprMIROps(env, e.Value, fn, templateInfo)
	case *InferExpr:
		entry := MIRInference{CandidateType: e.CandidateType, ScoreType: "float", Normalization: "StableSoftMax", Temperature: 1.0, NoEnabledPolicy: "Panic", NaNPolicy: "Panic", InfinityPolicy: "EqualPositiveInfinityElseNegativeInfinityZero", SourceSpan: e.Span}
		for _, candidate := range e.Candidates {
			item := MIRInferenceCandidate{Identity: candidate.Identity, Score: evt1ExprIdentity(candidate.Score), DeclarationOrder: candidate.DeclarationOrder, SourceSpan: candidate.Span}
			if candidate.Guard != nil {
				item.Guard = evt1ExprIdentity(candidate.Guard)
			}
			entry.Candidates = append(entry.Candidates, item)
		}
		fn.Inferences = append(fn.Inferences, entry)
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "infer", Detail: "StableSoftMax", SourceSpan: e.Span})
		for _, candidate := range e.Candidates {
			if candidate.Guard != nil {
				collectExprMIROps(env, candidate.Guard, fn, templateInfo)
			}
			collectExprMIROps(env, candidate.Score, fn, templateInfo)
		}
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
		op := MIROperation{ID: id, Kind: kind, Detail: exprLabel(e.Value), SourceSpan: e.Span}
		if e.DynInterface != "" {
			op.Kind, op.Type, op.Detail, op.Provenance = "dyn_make", "dyn "+e.DynInterface, e.WitnessID+" <- "+exprLabel(e.Value), e.DynProvenance
			if e.DynScoped {
				op.Provenance += ":scoped"
			}
			op.NoCopy, op.NoAllocation, op.NoOwnershipTransfer = true, true, true
		}
		fn.Operations = append(fn.Operations, op)
	case *BindExpr:
		if e.BindKind == "layout" || e.BindKind == "stream" {
			kind := e.BindKind + "_bind"
			provenance := e.ProvenanceKind
			if e.ProvenanceScoped {
				provenance += ":scoped"
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Type: e.TargetType.String(), Detail: exprLabel(e.Source), LayoutName: e.LayoutName, Mutability: map[bool]string{true: "const", false: "mutable"}[e.TargetType.Const], Provenance: provenance, NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SameBackingRegion: true, SourceSpan: e.Span})
			collectExprMIROps(env, e.Source, fn, templateInfo)
			return
		}
		countCheck := "comptime_equal"
		if e.RuntimeCheck {
			countCheck = "runtime_overflow_safe_equal"
		}
		mutability := "mutable"
		if e.TargetType.Const {
			mutability = "const"
		}
		provenance := e.ProvenanceKind
		if e.ProvenanceScoped {
			provenance += ":scoped"
		}
		fn.Operations = append(fn.Operations, MIROperation{
			ID:                  id,
			Kind:                "bind_storage",
			Type:                e.TargetType.String(),
			Detail:              exprLabel(e.Source),
			SourceStorageKind:   e.SourceType.StorageKind,
			TargetStorageKind:   e.TargetType.StorageKind,
			TargetRank:          evt1StorageRank(e.TargetType),
			TargetShape:         append([]StorageDimension{}, e.TargetType.Shape...),
			CountCheck:          countCheck,
			Mutability:          mutability,
			Provenance:          provenance,
			NoCopy:              true,
			NoAllocation:        true,
			NoOwnershipTransfer: true,
			SourceSpan:          e.Span,
		})
		collectExprMIROps(env, e.Source, fn, templateInfo)
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
		if e.CallableInvoke {
			kind := "callable_invoke"
			if e.CallableType != nil && e.CallableType.Kind == TypeCallback {
				kind = "callback_invoke"
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Type: e.Callee, Detail: e.CallableType.String(), ReturnType: e.CallableType.CallableResult.String(), NoAllocation: true, SourceSpan: e.Span})
			for _, arg := range e.Args {
				collectExprMIROps(env, arg, fn, templateInfo)
			}
			return
		}
		if strings.HasPrefix(e.Intrinsic, "inference_") {
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: e.Intrinsic, Detail: "explicit inference query", NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SourceSpan: e.Span})
			for _, arg := range e.Args {
				collectExprMIROps(env, arg, fn, templateInfo)
			}
			return
		}
		if e.Member {
			kind := "class_method_call"
			if e.DynDispatch {
				kind = "dyn_call"
			}
			op := MIROperation{ID: id, Kind: kind, Detail: e.Callee, SourceSpan: e.Span}
			if e.DynDispatch {
				op.Type, op.NoCopy, op.NoAllocation, op.NoOwnershipTransfer = e.DynInterface, true, true, true
				if req, ok := evt1InterfaceMethodRequirement(env, e.DynInterface, e.Callee); ok && req.ReturnType.Kind == TypeAsync {
					op.ReturnType, op.AsyncConstructor, op.Evaluation, op.OutcomeTransfer = req.ReturnType.String(), true, "ExactlyOnce", "MoveOnce"
				}
			}
			fn.Operations = append(fn.Operations, op)
			collectExprMIROps(env, e.Receiver, fn, templateInfo)
			for _, arg := range e.Args {
				collectExprMIROps(env, arg, fn, templateInfo)
			}
			return
		}
		if e.Intrinsic == "tensor_view" && e.TensorFacts != nil {
			facts := e.TensorFacts
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "tensor_view", Type: fmt.Sprintf("tensor<%s, %d>", facts.ElementType.String(), facts.Rank), Detail: facts.Source, TargetRank: facts.Rank, TargetShape: append([]StorageDimension{}, facts.Shape...), Mutability: facts.Mutability, Provenance: facts.Provenance, RegionID: facts.RegionID, BaseOffset: facts.BaseOffset, Alignment: facts.Alignment, TensorBackingKind: facts.BackingKind, Contiguous: true, NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SameBackingRegion: true, SourceSpan: e.Span})
			for _, arg := range e.Args {
				collectExprMIROps(env, arg, fn, templateInfo)
			}
			return
		}
		if e.Intrinsic == "span_from_region" || e.Intrinsic == "span_to_readonly" || e.Intrinsic == "span_subregion" {
			bounds := "source_extent"
			if e.Intrinsic == "span_subregion" {
				bounds = "half_open_overflow_safe"
			}
			provenance := e.ProvenanceKind
			if e.ProvenanceScoped {
				provenance += ":scoped"
			}
			fn.Operations = append(fn.Operations, MIROperation{
				ID: id, Kind: e.Intrinsic, Type: e.MutabilityToTypeName() + "<" + e.SpanElementType.String() + ">",
				ElementType: e.SpanElementType, RegionID: e.RegionID, Offset: e.BackingByteOffset, BaseOffset: e.BaseOffsetExpression,
				Length: e.LengthExpression, ByteExtentExpression: e.ByteExtentExpression, Alignment: e.Alignment,
				Mutability: e.Mutability, Provenance: provenance, Contiguous: true, BoundsCheck: bounds,
				NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SameBackingRegion: true, SourceSpan: e.Span,
			})
			for _, arg := range e.Args {
				collectExprMIROps(env, arg, fn, templateInfo)
			}
			return
		}
		kind := "call"
		detail := e.Callee
		if e.Intrinsic == "step_machine" || e.Intrinsic == "state_machine" {
			kind = e.Intrinsic
			if len(e.Args) == 2 {
				detail = exprLabel(e.Args[0]) + "." + exprLabel(e.Args[1])
			}
		} else if e.Callee == "Len" {
			kind = "array_len"
		} else if e.Callee == "Rank" {
			kind = "rank_query"
		} else if e.Callee == "Shape" {
			kind = "shape_query"
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
		kind := "field_access"
		if e.DynInterface != "" {
			kind = "dyn_field_get"
		}
		op := MIROperation{ID: id, Kind: kind, Detail: e.Field, SourceSpan: e.Span}
		if e.DynInterface != "" {
			op.Type, op.NoCopy, op.NoAllocation, op.NoOwnershipTransfer = e.DynInterface, true, true, true
		}
		if e.RegionID != "" {
			op.Kind, op.LayoutName, op.RegionID, op.Offset, op.ByteExtent, op.Alignment, op.SameBackingRegion = "region_projection", e.LayoutName, e.RegionID, e.RegionOffset, e.RegionExtent, e.RegionAlignment, true
		}
		fn.Operations = append(fn.Operations, op)
		collectExprMIROps(env, e.Receiver, fn, templateInfo)
	case *ArrayLiteralExpr:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "array_literal", Detail: fmt.Sprintf("%d elements", len(e.Elements)), SourceSpan: e.Span})
		for _, element := range e.Elements {
			collectExprMIROps(env, element, fn, templateInfo)
		}
	case *IndexExpr:
		if e.InferenceIndex {
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "inference_probability", Detail: fmt.Sprintf("candidate_tag=%d", e.CandidateTag), NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SourceSpan: e.Span})
			collectExprMIROps(env, e.Base, fn, templateInfo)
			return
		}
		if e.TensorIndex {
			kind := "tensor_index"
			if len(e.SymbolicIndices) > 0 {
				kind = "tensor_symbolic_index"
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Detail: strings.Join(e.SymbolicIndices, ","), BoundsCheck: "zero_le_index_lt_extent", SourceSpan: e.Span})
			collectExprMIROps(env, e.Base, fn, templateInfo)
			return
		}
		if e.SpanIndex {
			provenance := e.ProvenanceKind
			if e.ProvenanceScoped {
				provenance += ":scoped"
			}
			fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "span_index", ElementType: e.SpanElementType, RegionID: e.RegionID, Length: e.LengthExpression, Alignment: e.SpanAlignment, Mutability: e.SpanMutability, Provenance: provenance, Contiguous: true, BoundsCheck: "zero_le_index_lt_length", NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true, SameBackingRegion: true, SourceSpan: e.Span})
			collectExprMIROps(env, e.Base, fn, templateInfo)
			for _, index := range evt1StorageIndices(e) {
				collectExprMIROps(env, index, fn, templateInfo)
			}
			return
		}
		indices := evt1StorageIndices(e)
		kind := "array_index"
		if len(indices) > 1 {
			kind = "ndarray_index"
		}
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: kind, Detail: fmt.Sprintf("rank=%d bounds=checked layout=row-major", len(indices)), SourceSpan: e.Span})
		collectExprMIROps(env, e.Base, fn, templateInfo)
		for _, index := range indices {
			collectExprMIROps(env, index, fn, templateInfo)
		}
	case *BinaryExpr:
		if e.Tensor != nil {
			fn.TensorOperations = append(fn.TensorOperations, *e.Tensor)
		}
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
	case *FloatLiteral:
		fn.Operations = append(fn.Operations, MIROperation{ID: id, Kind: "literal", Detail: fmt.Sprintf("%g", e.Value), SourceSpan: e.Span})
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
	storageViewTypes := evt1CollectStorageViewTypes(l.module, l.env)
	spanTypes := evt1CollectSpanTypes(l.module)
	tensorTypes := evt1CollectTensorTypes(l.module)
	inferenceTypes := evt1InferenceCandidateTypes(l.module)
	body.WriteString(fmt.Sprintf("/* Generated by %s. DO NOT EDIT. Source: %s */\n", CompilerID, l.module.Path))
	body.WriteString(fmt.Sprintf("#include \"%s.generated.h\"\n", l.outputBase))
	if len(inferenceTypes) > 0 {
		body.WriteString("#include <math.h>\n")
	}
	body.WriteString("#include <stdio.h>\n#include <stdlib.h>\n\n")
	header.WriteString(fmt.Sprintf("/* Generated by %s. DO NOT EDIT. */\n", CompilerID))
	header.WriteString(fmt.Sprintf("#ifndef %s\n#define %s\n", guard, guard))
	for _, include := range l.env.profile.headersFor(l.module) {
		header.WriteString("#include " + include + "\n")
	}
	header.WriteString("#include <stdbool.h>\n")
	if len(storageViewTypes) > 0 || len(spanTypes) > 0 || len(tensorTypes) > 0 {
		header.WriteString("#include <stddef.h>\n")
	}
	header.WriteString("#include <stdint.h>\n\n")
	if evt1ModuleHasAsync(l.module) {
		header.WriteString(evt1AsyncRuntimeDeclarations())
		body.WriteString("#include <string.h>\n")
	}
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
	if err := l.writeRuntimeStorageAndTypeDecls(&header, typeDecls, evt1CollectStorageTypes(l.module, l.env), storageViewTypes); err != nil {
		return nil, nil, err
	}
	for _, enumDecl := range l.module.Enums {
		if inferenceTypes[enumDecl.Name] && len(enumDecl.Variants) > 0 {
			header.WriteString(fmt.Sprintf("typedef struct { float probabilities[%d]; uint32_t candidate_order[%d]; uint32_t candidate_count; } concept_inference_%s;\n\n", len(enumDecl.Variants), len(enumDecl.Variants), evt1TypeIdentity(Type{Name: enumDecl.Name, Kind: TypeEnum})))
		}
	}
	header.WriteString(evt1SpanDeclarations(spanTypes))
	header.WriteString(evt1TensorDeclarations(tensorTypes))
	header.WriteString(l.semanticViewDeclarations())
	header.WriteString(l.interfaceWitnessDeclarations())
	header.WriteString(l.callableDeclarations())
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
	if evt1ModuleHasAsync(l.module) {
		body.WriteString(evt1AsyncRuntimeDefinitions())
		body.WriteString(l.asyncForwardDeclarations())
	}
	if evt1ModuleUsesFailurePanic(l.module) || evt1ModuleUsesStorageBounds(l.module) || evt1ModuleUsesTransitionPanic(l.module) || len(inferenceTypes) > 0 {
		body.WriteString("static void concept_panic(const char* reason, int line, int column) {\n")
		body.WriteString("  fprintf(stderr, \"Concept panic at %d:%d: %s\\n\", line, column, reason);\n")
		body.WriteString("  abort();\n}\n\n")
	}
	runtimeAutomata := evt1RuntimeAutomataUsage(l.module)
	if len(runtimeAutomata) > 0 {
		body.WriteString("static void concept_abort_invalid_automata_state(const char* automata_name, int machine, int state) {\n")
		body.WriteString("  fprintf(stderr, \"invalid automata state for %s: machine=%d state=%d\\n\", automata_name, machine, state);\n")
		body.WriteString("  abort();\n}\n\n")
	}
	if len(runtimeAutomata) > 0 {
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
	body.WriteString(l.interfaceWitnessDefinitions())
	body.WriteString(l.callableDefinitions())
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

type evt1RuntimeHeaderDecl struct {
	key  string
	deps []string
	emit func() string
}

func (l *lowering) writeRuntimeStorageAndTypeDecls(out *strings.Builder, typeDecls []evt1RuntimeTypeDecl, storageTypes, viewTypes []Type) error {
	knownNamed := make(map[string]bool, len(typeDecls))
	knownStorage := make(map[string]bool, len(storageTypes))
	knownView := make(map[string]bool, len(viewTypes))
	for _, decl := range typeDecls {
		knownNamed[decl.Name] = true
	}
	for _, storageType := range storageTypes {
		knownStorage[evt1TypeIdentity(storageType)] = true
	}
	for _, viewType := range viewTypes {
		knownView[evt1StorageViewCName(viewType)] = true
	}

	var typeDeps func(Type) []string
	typeDeps = func(t Type) []string {
		if t.PointerTo != nil {
			return typeDeps(*t.PointerTo)
		}
		if t.ArrayElem != nil {
			if t.isReference() {
				key := evt1StorageViewCName(t)
				if knownView[key] {
					return []string{"view:" + key}
				}
			}
			key := evt1TypeIdentity(t.valueType())
			if knownStorage[key] {
				return []string{"storage:" + key}
			}
			return nil
		}
		if knownNamed[t.Name] {
			return []string{"named:" + t.Name}
		}
		return nil
	}

	decls := make([]evt1RuntimeHeaderDecl, 0, len(typeDecls)+len(storageTypes)+len(viewTypes))
	for i := range typeDecls {
		decl := typeDecls[i]
		deps := make([]string, 0)
		seen := map[string]bool{"named:" + decl.Name: true}
		add := func(t Type) {
			for _, dep := range typeDeps(t) {
				if !seen[dep] {
					seen[dep] = true
					deps = append(deps, dep)
				}
			}
		}
		if decl.Struct != nil {
			for _, field := range decl.Struct.Fields {
				add(field.Type)
			}
		} else {
			for _, variant := range decl.Enum.Variants {
				for _, field := range variant.Payload {
					add(field.Type)
				}
			}
		}
		current := decl
		decls = append(decls, evt1RuntimeHeaderDecl{key: "named:" + decl.Name, deps: deps, emit: func() string {
			if current.Struct != nil {
				return l.structHeader(*current.Struct)
			}
			return l.enumHeader(*current.Enum)
		}})
	}
	for i := range storageTypes {
		storageType := storageTypes[i]
		current := storageType
		deps := typeDeps(*storageType.ArrayElem)
		decls = append(decls, evt1RuntimeHeaderDecl{
			key:  "storage:" + evt1TypeIdentity(storageType),
			deps: deps,
			emit: func() string {
				if len(l.module.Layouts) > 0 {
					return fmt.Sprintf("typedef struct { _Alignas(%d) %s data[%d]; } %s;\n\n", evt1InlineStorageAlignment, evt1CType(*current.ArrayElem), evt1StorageElementCount(current), evt1StorageCName(current))
				}
				return fmt.Sprintf("typedef struct { %s data[%d]; } %s;\n\n", evt1CType(*current.ArrayElem), evt1StorageElementCount(current), evt1StorageCName(current))
			},
		})
	}
	for i := range viewTypes {
		viewType := viewTypes[i]
		current := viewType
		deps := typeDeps(*viewType.ArrayElem)
		decls = append(decls, evt1RuntimeHeaderDecl{
			key:  "view:" + evt1StorageViewCName(viewType),
			deps: deps,
			emit: func() string {
				pointerType := evt1CType(*current.ArrayElem) + "*"
				if current.Const {
					pointerType = "const " + evt1CType(*current.ArrayElem) + "*"
				}
				return fmt.Sprintf("typedef struct { %s data; size_t shape[%d]; } %s;\n\n", pointerType, evt1StorageRank(current), evt1StorageViewCName(current))
			},
		})
	}

	emitted := make(map[string]bool, len(decls))
	for len(emitted) < len(decls) {
		progress := false
		for _, decl := range decls {
			if emitted[decl.key] {
				continue
			}
			ready := true
			for _, dep := range decl.deps {
				if !emitted[dep] {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}
			out.WriteString(decl.emit())
			emitted[decl.key] = true
			progress = true
		}
		if !progress {
			var blocked []string
			for _, decl := range decls {
				if !emitted[decl.key] {
					blocked = append(blocked, decl.key)
				}
			}
			return fmt.Errorf("Concept/Vulkan EVT1 runtime storage type cycle is not representable by value: %s", strings.Join(blocked, ", "))
		}
	}
	return nil
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
			case *ForeachStmt:
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
			if usesExpr(e.Base) {
				return true
			}
			for _, index := range evt1StorageIndices(e) {
				if usesExpr(index) {
					return true
				}
			}
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
			case *ForeachStmt:
				if usesExpr(s.Source) || visitBlock(s.Body) {
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
	if info.Decl.SignalType.Name == "" {
		return l.canonicalAutomataStackRuntimeSupport(info)
	}
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

func evt1AutomataMachineStorageCName(automataName, machineName string) string {
	return evt1CName(automataName) + "_" + evt1PayloadFieldName(machineName) + "_machine"
}

func evt1AutomataStateEnvironmentCName(automataName string) string {
	return evt1CName(automataName) + "_state"
}

func evt1AutomataStepCName(automataName, machineName string) string {
	return evt1CName(automataName) + "_step_" + evt1PayloadFieldName(machineName)
}

func evt1AutomataDropCName(automataName string) string { return evt1CName(automataName) + "_drop" }

func (l *lowering) canonicalAutomataRuntimeSupport(info *evt1AutomataInfo) string {
	var b strings.Builder
	stateType := evt1AutomataStateEnvironmentCName(info.Decl.Name)
	instanceType := evt1AutomataRuntimeInstanceCName(info.Decl.Name)
	b.WriteString(fmt.Sprintf("typedef struct %s {\n", stateType))
	if len(info.Decl.StateFields) == 0 {
		b.WriteString("  unsigned char unused;\n")
	}
	for _, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(field.Type), field.Name))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", stateType))
	for _, machine := range info.Decl.Machines {
		mt := evt1AutomataMachineStorageCName(info.Decl.Name, machine.Name)
		b.WriteString("enum {\n")
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name), info.StateOrdinal[machine.Name][state.Name]))
		}
		b.WriteString("};\n")
		b.WriteString(fmt.Sprintf("typedef struct %s {\n  uint8_t current_state;\n", mt))
		for _, field := range machine.Fields {
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(field.Type), field.Name))
		}
		b.WriteString(fmt.Sprintf("} %s;\n\n", mt))
	}
	b.WriteString(fmt.Sprintf("typedef struct %s {\n  %s shared;\n", instanceType, stateType))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1AutomataMachineStorageCName(info.Decl.Name, machine.Name), evt1PayloadFieldName(machine.Name)))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", instanceType))
	for _, machine := range info.Decl.Machines {
		stepName := evt1AutomataStepCName(info.Decl.Name, machine.Name)
		b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n  switch (instance->%s.current_state) {\n", stepName, instanceType, evt1PayloadFieldName(machine.Name)))
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("    case %s:\n      {\n", evt1AutomataStateConstName(info.Decl.Name, machine.Name, state.Name)))
			lower := newEVT1FunctionLowerer(l, FunctionDecl{Name: stepName, ReturnType: Type{Name: "void", Kind: TypeBuiltin}, Body: state.Body}, stepName, true)
			lower.automataStepName, lower.machineStepName = info.Decl.Name, machine.Name
			lower.scope[0]["state"] = evt1Binding{cName: "instance->shared", t: Type{Name: info.Decl.Name + "#state", Kind: TypeStruct}}
			for _, field := range info.Decl.StateFields {
				lower.scope[0][field.Name] = evt1Binding{cName: "instance->shared." + field.Name, t: field.Type}
			}
			lower.scope[0]["machine"] = evt1Binding{cName: "instance->" + evt1PayloadFieldName(machine.Name), t: Type{Name: info.Decl.Name + "#" + machine.Name + "#machine", Kind: TypeStruct}}
			for _, field := range machine.Fields {
				lower.scope[0][field.Name] = evt1Binding{cName: "instance->" + evt1PayloadFieldName(machine.Name) + "." + field.Name, t: field.Type}
			}
			b.WriteString(lower.lowerBlock(*state.Body, 4))
			b.WriteString("        break;\n      }\n")
		}
		b.WriteString(fmt.Sprintf("    default:\n      concept_abort_invalid_automata_state(\"%s\", %d, instance->%s.current_state);\n  }\n}\n\n", info.Decl.Name, info.MachineOrdinal[machine.Name], evt1PayloadFieldName(machine.Name)))
	}
	b.WriteString(fmt.Sprintf("static void %s(%s* instance", evt1AutomataRuntimeInitName(info.Decl.Name), instanceType))
	for i, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf(", %s state_%d", evt1CType(field.Type), i))
	}
	b.WriteString(") {\n")
	for i, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf("  instance->shared.%s = state_%d;\n", field.Name, i))
	}
	initLower := newEVT1FunctionLowerer(l, FunctionDecl{Name: "automata_init", ReturnType: Type{Name: "void", Kind: TypeBuiltin}}, "", true)
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  instance->%s.current_state = %s;\n", evt1PayloadFieldName(machine.Name), evt1AutomataStateConstName(info.Decl.Name, machine.Name, machine.States[0].Name)))
		for _, field := range machine.Fields {
			value := fmt.Sprintf("(%s){0}", evt1CType(field.Type))
			if field.Initializer != nil {
				_, lowered, _ := initLower.lowerExpr(field.Initializer, 1)
				value = lowered
			}
			b.WriteString(fmt.Sprintf("  instance->%s.%s = %s;\n", evt1PayloadFieldName(machine.Name), field.Name, value))
		}
	}
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n", evt1AutomataDropCName(info.Decl.Name), instanceType))
	dropLower := newEVT1FunctionLowerer(l, FunctionDecl{Name: "automata_drop", ReturnType: Type{Name: "void", Kind: TypeBuiltin}}, "", true)
	for mi := len(info.Decl.Machines) - 1; mi >= 0; mi-- {
		machine := info.Decl.Machines[mi]
		for fi := len(machine.Fields) - 1; fi >= 0; fi-- {
			field := machine.Fields[fi]
			b.WriteString(dropLower.lowerDropValue(field.Type, "instance->"+evt1PayloadFieldName(machine.Name)+"."+field.Name, 1))
		}
	}
	for i := len(info.Decl.StateFields) - 1; i >= 0; i-- {
		field := info.Decl.StateFields[i]
		b.WriteString(dropLower.lowerDropValue(field.Type, "instance->shared."+field.Name, 1))
	}
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

func (l *lowering) semanticViewDeclarations() string {
	var b strings.Builder
	for _, layout := range l.module.Layouts {
		mut := Type{Name: layout.Name, Kind: TypeLayout, Ownership: "ref"}
		read := mut
		read.Const = true
		b.WriteString(fmt.Sprintf("typedef struct { unsigned char* data; } %s;\n", evt1SemanticViewCName(mut)))
		b.WriteString(fmt.Sprintf("typedef struct { const unsigned char* data; } %s;\n", evt1SemanticViewCName(read)))
	}
	for _, stream := range l.module.Streams {
		mut := Type{Name: stream.Name, Kind: TypeStream, Ownership: "ref"}
		read := mut
		read.Const = true
		layoutMut := Type{Name: stream.LayoutName, Kind: TypeLayout, Ownership: "ref"}
		layoutRead := layoutMut
		layoutRead.Const = true
		b.WriteString(fmt.Sprintf("typedef %s %s;\n", evt1SemanticViewCName(layoutMut), evt1SemanticViewCName(mut)))
		b.WriteString(fmt.Sprintf("typedef %s %s;\n", evt1SemanticViewCName(layoutRead), evt1SemanticViewCName(read)))
	}
	if len(l.module.Layouts)+len(l.module.Streams) > 0 {
		b.WriteByte('\n')
	}
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
	if t.Kind == TypeCallable {
		return evt1CallableEnvCName(t.CallableID)
	}
	if t.Kind == TypeCallback {
		return evt1CallbackCName(t)
	}
	if t.Kind == TypeAsync || t.Name == "Async" {
		return "concept_async_operation"
	}
	if evt1IsInferenceType(t) {
		return evt1InferenceCName(t)
	}
	if t.Kind == TypeDyn {
		return evt1DynCName(t)
	}
	if evt1IsTensorType(t) {
		return evt1TensorCName(t)
	}
	if evt1IsSpanType(t) {
		return evt1SpanCName(t)
	}
	if t.isReference() && (t.Kind == TypeLayout || t.Kind == TypeStream) {
		return evt1SemanticViewCName(t)
	}
	if t.PointerTo != nil {
		base := evt1CType(*t.PointerTo)
		if t.Const {
			return "const " + base + "*"
		}
		return base + "*"
	}
	if t.isReference() && t.ArrayElem != nil {
		return evt1StorageViewCName(t)
	}
	if t.isBorrow() || t.isReference() {
		base := evt1CType(t.borrowBase())
		if t.Const {
			return "const " + base + "*"
		}
		return base + "*"
	}
	if t.ArrayElem != nil {
		return evt1StorageCName(t.valueType())
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
	for _, automata := range module.Automata {
		for _, field := range automata.StateFields {
			if visitType(field.Type) {
				return true
			}
		}
		for _, machine := range automata.Machines {
			for _, field := range machine.Fields {
				if visitType(field.Type) {
					return true
				}
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
	if fn.Async {
		return l.asyncFunctionSymbols(fn)
	}
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
	if instance.Function.Async {
		fn := instance.Function
		fn.Name = instance.GeneratedSymbol
		stepName := evt1AsyncStepName(l.outputBase, fn.Name)
		initName := evt1AsyncInitName(l.outputBase, fn.Name)
		var b strings.Builder
		b.WriteString(fmt.Sprintf("static void %s(concept_async_operation*, void*);\n", stepName))
		b.WriteString(fmt.Sprintf("static void %s(void*", initName))
		for _, p := range fn.Params {
			b.WriteString(fmt.Sprintf(", %s %s", evt1CType(p.Type), p.Name))
		}
		b.WriteString(");\n")
		b.WriteString(l.lowerAsyncFunction(fn, instance.GeneratedSymbol))
		return b.String()
	}
	lower := newEVT1FunctionLowerer(l, instance.Function, instance.GeneratedSymbol, true)
	return lower.lower()
}

type evt1FunctionLowerer struct {
	l                *lowering
	fn               FunctionDecl
	plan             *FunctionPlan
	symbol           string
	private          bool
	scope            []map[string]evt1Binding
	ownedOrder       [][]string
	liveOwners       map[string]bool
	tempCounter      int
	tryHandlers      []map[string]evt1LoweredTryHandler
	automataStepName string
	machineStepName  string
	automataInfo     *evt1AutomataInfo
	machineDecl      *MachineDecl
	frameIndexName   string
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
	tensorFacts      *TensorViewFacts
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
	var plan *FunctionPlan
	if l.plan != nil {
		plan = l.plan.Function(fn.Name)
	}
	return &evt1FunctionLowerer{l: l, fn: fn, plan: plan, symbol: symbol, private: private, scope: scope, ownedOrder: ownedOrder, liveOwners: liveOwners}
}

func (f *evt1FunctionLowerer) plannedStrategy(operation, fallback string) string {
	if f.plan == nil {
		return fallback
	}
	if strategy, ok := f.plan.Strategy(operation); ok {
		return strategy
	}
	return fallback
}

func (f *evt1FunctionLowerer) plannedStrategyAt(operation string, span Span, fallback string) string {
	if f.plan == nil {
		return fallback
	}
	if strategy, ok := f.plan.StrategyAt(operation, span); ok {
		return strategy
	}
	return fallback
}

func (f *evt1FunctionLowerer) tensorStrategy(kind string) string {
	if f.plan != nil {
		for _, tensor := range f.plan.Tensors {
			if tensor.Operation == kind {
				return tensor.Strategy
			}
		}
	}
	return "DirectLoopNest"
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
		if s.InlineTensor != nil {
			return f.lowerInlineTensorDeclaration(s, indent)
		}
		if construct, ok := s.Value.(*StructConstructExpr); ok && construct.StructName == s.Type.Name {
			return f.lowerLocalStructConstruct(s.Type, s.Name, *construct, indent)
		}
		prelude, value, _ := f.lowerExprExpected(s.Value, s.Type, indent)
		cName := f.bindName(s.Name, s.Type)
		if evt1IsTensorType(s.Type) {
			binding := f.currentScope()[s.Name]
			binding.tensorFacts = evt1TensorFactsForValue(nil, s.Value, s.Type)
			f.currentScope()[s.Name] = binding
		}
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
		if info.Decl.SignalType.Name == "" {
			var args []string
			for _, arg := range s.StateArgs {
				prelude, value, _ := f.lowerExpr(arg, indent)
				b.WriteString(prelude)
				args = append(args, value)
			}
			suffix := ""
			if len(args) > 0 {
				suffix = ", " + strings.Join(args, ", ")
			}
			b.WriteString(ind(indent) + fmt.Sprintf("%s(&%s%s);\n", initName, cName, suffix))
			f.ownedOrder[len(f.ownedOrder)-1] = append(f.ownedOrder[len(f.ownedOrder)-1], cName)
			f.liveOwners[cName] = true
			return b.String()
		}
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
		if s.Tensor != nil {
			return f.lowerTensorAssignment(s, indent)
		}
		if field, ok := s.Target.(*FieldExpr); ok && field.DynInterface != "" {
			recvPrelude, recv, _ := f.lowerExpr(field.Receiver, indent)
			valuePrelude, value, _ := f.lowerExpr(s.Value, indent)
			if f.plannedStrategy("dyn_field_set", "WitnessFieldAccessor") != "WitnessFieldAccessor" {
				return ind(indent) + "/* invalid dyn field-set plan */\n"
			}
			return recvPrelude + valuePrelude + ind(indent) + fmt.Sprintf("(%s).witness->set_%s((%s).object, %s);\n", recv, field.Field, recv, value)
		}
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
	case *TransitionStmt:
		if f.automataStepName == "" {
			return ind(indent) + "/* invalid transition */\n"
		}
		statePlace := fmt.Sprintf("instance->%s.current_state", evt1PayloadFieldName(f.machineStepName))
		if f.frameIndexName != "" {
			statePlace = fmt.Sprintf("instance->%s[%s].current_state", evt1MachineFramesCName(f.machineStepName), f.frameIndexName)
		}
		return f.lowerAllScopeDrops(indent) + ind(indent) + fmt.Sprintf("%s = %s;\n", statePlace, evt1AutomataStateConstName(f.automataStepName, f.machineStepName, s.Target)) + ind(indent) + "return;\n"
	case *YieldStmt:
		if f.automataStepName == "" {
			return ind(indent) + "/* invalid yield */\n"
		}
		return f.lowerAllScopeDrops(indent) + ind(indent) + "return; /* CONCEPT_STEP_YIELDED: re-enter current state */\n"
	case *PushMachineStmt:
		return f.lowerMachinePush(*s, indent)
	case *MachineCompleteStmt:
		return f.lowerMachineCompletion(*s, indent)
	case *ForeachStmt:
		return f.lowerForeachStmt(*s, indent)
	case *TransitionMatchStmt:
		return f.lowerTransitionMatchStmt(*s, indent)
	case *TransitionDecideStmt:
		return f.lowerTransitionDecideStmt(*s, indent)
	case *TransitionInferStmt:
		return f.lowerTransitionInferStmt(*s, indent)
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

func (f *evt1FunctionLowerer) lowerTransitionMatchStmt(stmt TransitionMatchStmt, indent int) string {
	subPrelude, subjectExpr, subjectType := f.lowerExpr(stmt.Subject, indent)
	enumDecl := f.l.env.enums[subjectType.Name]
	if decl, ok := evt1FailureEnumDecl(subjectType); ok {
		enumDecl = decl
	}
	subjectTemp := f.nextTemp("transition_subject")
	before := f.cloneLiveOwners()
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
		f.liveOwners = f.cloneOwnerState(before)
		b.WriteString(ind(indent) + fmt.Sprintf("case %s:\n", tag))
		b.WriteString(f.lowerAllScopeDrops(indent + 1))
		statePlace := fmt.Sprintf("instance->%s.current_state", evt1PayloadFieldName(f.machineStepName))
		if f.frameIndexName != "" {
			statePlace = fmt.Sprintf("instance->%s[%s].current_state", evt1MachineFramesCName(f.machineStepName), f.frameIndexName)
		}
		b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s;\n", statePlace, evt1AutomataStateConstName(f.automataStepName, f.machineStepName, arm.Target)))
		b.WriteString(ind(indent+1) + "return;\n")
	}
	f.liveOwners = before
	b.WriteString(ind(indent) + "default:\n")
	b.WriteString(ind(indent+1) + fmt.Sprintf("concept_panic(%q, %d, %d);\n", "machine transition match found no matching case", stmt.Span.Line, stmt.Span.Column))
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerTransitionDecideStmt(stmt TransitionDecideStmt, indent int) string {
	hasBest := f.nextTemp("decision_has_best")
	bestScore := f.nextTemp("decision_best_score")
	bestState := f.nextTemp("decision_best_state")
	var b strings.Builder
	b.WriteString(ind(indent) + "{\n")
	b.WriteString(ind(indent+1) + fmt.Sprintf("bool %s = false;\n", hasBest))
	b.WriteString(ind(indent+1) + fmt.Sprintf("%s %s = (%s)0;\n", evt1CType(stmt.ScoreType), bestScore, evt1CType(stmt.ScoreType)))
	b.WriteString(ind(indent+1) + fmt.Sprintf("uint8_t %s = 0u;\n", bestState))
	for _, candidate := range stmt.Candidates {
		candidateIndent := indent + 1
		if candidate.Guard != nil {
			guardPrelude, guardExpr, _ := f.lowerExpr(candidate.Guard, candidateIndent)
			guardTemp := f.nextTemp("decision_guard")
			b.WriteString(guardPrelude)
			b.WriteString(ind(candidateIndent) + fmt.Sprintf("bool %s = %s;\n", guardTemp, guardExpr))
			b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (%s) {\n", guardTemp))
			candidateIndent++
		}
		scorePrelude, scoreExpr, _ := f.lowerExpr(candidate.Score, candidateIndent)
		scoreTemp := f.nextTemp("decision_score")
		b.WriteString(scorePrelude)
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(stmt.ScoreType), scoreTemp, scoreExpr))
		if stmt.ScoreType.Name == "float" {
			b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (%s != %s) { concept_panic(%q, %d, %d); }\n", scoreTemp, scoreTemp, "machine decision transition score is NaN", candidate.Span.Line, candidate.Span.Column))
		}
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (!%s || %s > %s) {\n", hasBest, scoreTemp, bestScore))
		b.WriteString(ind(candidateIndent+1) + fmt.Sprintf("%s = true;\n", hasBest))
		b.WriteString(ind(candidateIndent+1) + fmt.Sprintf("%s = %s;\n", bestScore, scoreTemp))
		b.WriteString(ind(candidateIndent+1) + fmt.Sprintf("%s = %s;\n", bestState, evt1AutomataStateConstName(f.automataStepName, f.machineStepName, candidate.Identity)))
		b.WriteString(ind(candidateIndent) + "}\n")
		if candidate.Guard != nil {
			b.WriteString(ind(indent+1) + "}\n")
		}
	}
	b.WriteString(ind(indent+1) + fmt.Sprintf("if (!%s) { concept_panic(%q, %d, %d); }\n", hasBest, "machine decision transition has no enabled candidates", stmt.Span.Line, stmt.Span.Column))
	b.WriteString(f.lowerAllScopeDrops(indent + 1))
	statePlace := fmt.Sprintf("instance->%s.current_state", evt1PayloadFieldName(f.machineStepName))
	if f.frameIndexName != "" {
		statePlace = fmt.Sprintf("instance->%s[%s].current_state", evt1MachineFramesCName(f.machineStepName), f.frameIndexName)
	}
	b.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s;\n", statePlace, bestState))
	b.WriteString(ind(indent+1) + "return;\n")
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerInferenceCore(candidates []ScoredCandidate, indent int) (string, string) {
	n := len(candidates)
	scores := f.nextTemp("inference_scores")
	enabled := f.nextTemp("inference_enabled")
	probabilities := f.nextTemp("inference_probabilities")
	enabledCount := f.nextTemp("inference_enabled_count")
	positiveInfCount := f.nextTemp("inference_positive_inf_count")
	var b strings.Builder
	b.WriteString(ind(indent) + fmt.Sprintf("float %s[%d] = {0};\n", scores, n))
	b.WriteString(ind(indent) + fmt.Sprintf("bool %s[%d] = {0};\n", enabled, n))
	b.WriteString(ind(indent) + fmt.Sprintf("float %s[%d] = {0};\n", probabilities, n))
	b.WriteString(ind(indent) + fmt.Sprintf("uint32_t %s = 0u;\n", enabledCount))
	b.WriteString(ind(indent) + fmt.Sprintf("uint32_t %s = 0u;\n", positiveInfCount))
	for i, candidate := range candidates {
		candidateIndent := indent
		if candidate.Guard != nil {
			guardPrelude, guardExpr, _ := f.lowerExpr(candidate.Guard, candidateIndent)
			guardTemp := f.nextTemp("inference_guard")
			b.WriteString(guardPrelude)
			b.WriteString(ind(candidateIndent) + fmt.Sprintf("bool %s = %s;\n", guardTemp, guardExpr))
			b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (%s) {\n", guardTemp))
			candidateIndent++
		}
		scorePrelude, scoreExpr, _ := f.lowerExpr(candidate.Score, candidateIndent)
		b.WriteString(scorePrelude)
		scoreTemp := f.nextTemp("inference_score")
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("float %s = %s;\n", scoreTemp, scoreExpr))
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (%s != %s) { concept_panic(%q, %d, %d); }\n", scoreTemp, scoreTemp, "inference score is NaN", candidate.Span.Line, candidate.Span.Column))
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("%s[%d] = %s; %s[%d] = true; ++%s;\n", scores, i, scoreTemp, enabled, i, enabledCount))
		b.WriteString(ind(candidateIndent) + fmt.Sprintf("if (isinf(%s) && %s > 0.0f) { ++%s; }\n", scoreTemp, scoreTemp, positiveInfCount))
		if candidate.Guard != nil {
			b.WriteString(ind(indent) + "}\n")
		}
	}
	b.WriteString(ind(indent) + fmt.Sprintf("if (%s == 0u) { concept_panic(%q, 0, 0); }\n", enabledCount, "inference has no enabled candidates"))
	b.WriteString(ind(indent) + fmt.Sprintf("if (%s > 0u) {\n", positiveInfCount))
	b.WriteString(ind(indent+1) + fmt.Sprintf("for (uint32_t i = 0u; i < %du; ++i) { if (%s[i] && isinf(%s[i]) && %s[i] > 0.0f) { %s[i] = 1.0f / (float)%s; } }\n", n, enabled, scores, scores, probabilities, positiveInfCount))
	b.WriteString(ind(indent) + "} else {\n")
	maxScore := f.nextTemp("inference_max_score")
	hasFinite := f.nextTemp("inference_has_finite")
	b.WriteString(ind(indent+1) + fmt.Sprintf("float %s = -INFINITY; bool %s = false;\n", maxScore, hasFinite))
	b.WriteString(ind(indent+1) + fmt.Sprintf("for (uint32_t i = 0u; i < %du; ++i) { if (%s[i] && !isinf(%s[i])) { if (!%s || %s[i] > %s) { %s = %s[i]; } %s = true; } }\n", n, enabled, scores, hasFinite, scores, maxScore, maxScore, scores, hasFinite))
	b.WriteString(ind(indent+1) + fmt.Sprintf("if (!%s) { concept_panic(%q, 0, 0); }\n", hasFinite, "inference normalization has no finite support"))
	sum := f.nextTemp("inference_sum")
	b.WriteString(ind(indent+1) + fmt.Sprintf("float %s = 0.0f;\n", sum))
	b.WriteString(ind(indent+1) + fmt.Sprintf("for (uint32_t i = 0u; i < %du; ++i) { if (%s[i]) { %s[i] = expf(%s[i] - %s); %s += %s[i]; } }\n", n, enabled, probabilities, scores, maxScore, sum, probabilities))
	b.WriteString(ind(indent+1) + fmt.Sprintf("if (!(%s > 0.0f) || !isfinite(%s)) { concept_panic(%q, 0, 0); }\n", sum, sum, "inference normalization failed"))
	b.WriteString(ind(indent+1) + fmt.Sprintf("for (uint32_t i = 0u; i < %du; ++i) { %s[i] /= %s; }\n", n, probabilities, sum))
	b.WriteString(ind(indent) + "}\n")
	return b.String(), probabilities
}

func (f *evt1FunctionLowerer) lowerInferenceExpr(expr *InferExpr, indent int) (string, string, Type) {
	t := Type{Name: evt1InferenceName, Kind: TypeApplied, TypeArgs: []Type{expr.CandidateType}, Span: expr.Span}
	result := f.nextTemp("inference")
	core, probabilities := f.lowerInferenceCore(expr.Candidates, indent)
	var b strings.Builder
	b.WriteString(core)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = {0};\n", evt1CType(t), result))
	decl := f.l.env.enums[expr.CandidateType.Name]
	for i, candidate := range expr.Candidates {
		variant, _ := evt1LookupVariant(decl, candidate.Identity)
		b.WriteString(ind(indent) + fmt.Sprintf("%s.probabilities[%d] = %s[%d]; %s.candidate_order[%d] = %d;\n", result, variant.Tag, probabilities, i, result, i, variant.Tag))
	}
	b.WriteString(ind(indent) + fmt.Sprintf("%s.candidate_count = %du;\n", result, len(expr.Candidates)))
	return b.String(), result, t
}

func (f *evt1FunctionLowerer) lowerTransitionInferStmt(stmt TransitionInferStmt, indent int) string {
	core, probabilities := f.lowerInferenceCore(stmt.Candidates, indent+1)
	best := f.nextTemp("inference_best")
	var b strings.Builder
	b.WriteString(ind(indent) + "{\n")
	b.WriteString(core)
	b.WriteString(ind(indent+1) + fmt.Sprintf("uint32_t %s = 0u;\n", best))
	b.WriteString(ind(indent+1) + fmt.Sprintf("for (uint32_t i = 1u; i < %du; ++i) { if (%s[i] > %s[%s]) { %s = i; } }\n", len(stmt.Candidates), probabilities, probabilities, best, best))
	b.WriteString(f.lowerAllScopeDrops(indent + 1))
	b.WriteString(ind(indent+1) + fmt.Sprintf("switch (%s) {\n", best))
	for i, candidate := range stmt.Candidates {
		statePlace := fmt.Sprintf("instance->%s.current_state", evt1PayloadFieldName(f.machineStepName))
		if f.frameIndexName != "" {
			statePlace = fmt.Sprintf("instance->%s[%s].current_state", evt1MachineFramesCName(f.machineStepName), f.frameIndexName)
		}
		b.WriteString(ind(indent+2) + fmt.Sprintf("case %d: %s = %s; break;\n", i, statePlace, evt1AutomataStateConstName(f.automataStepName, f.machineStepName, candidate.Identity)))
	}
	b.WriteString(ind(indent+2) + "default: concept_panic(\"inference policy selected invalid candidate\", 0, 0);\n")
	b.WriteString(ind(indent+1) + "}\n")
	b.WriteString(ind(indent+1) + "return;\n")
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) cloneOwnerState(source map[string]bool) map[string]bool {
	out := make(map[string]bool, len(source))
	for name, live := range source {
		out[name] = live
	}
	return out
}

func (f *evt1FunctionLowerer) lowerExpr(expr Expr, indent int) (string, string, Type) {
	_, failureConstruct := expr.(*ConstructExpr)
	if construct, ok := expr.(*ConstructExpr); ok {
		failureConstruct = evt1IsFailureType(construct.ResolvedType)
	}
	freshNoncopyableConstruct := false
	if construct, ok := expr.(*StructConstructExpr); ok {
		freshNoncopyableConstruct = !evt1TypeCopyable(f.l.env, Type{Name: construct.StructName, Kind: TypeStruct})
	}
	if value, ok := evt1TryEvalRuntimeExpr(f.l.env, f.evalScope(), expr); !failureConstruct && !freshNoncopyableConstruct && ok && evt1RuntimeTypeSafe(f.l.env, value.Type) {
		return "", evt1RenderCValue(f.l.env, value), value.Type
	}
	switch e := expr.(type) {
	case *CallableExpr:
		return f.lowerCallableLiteral(e, indent)
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
			if binding.t.isReference() && (binding.t.ArrayElem != nil || evt1IsSemanticViewType(f.l.env, binding.t)) {
				return "", binding.cName, binding.t
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
		if e.AutomataStorage != "" {
			parts := strings.Split(e.AutomataStorage, ":")
			binding, _ := scopeLookup(parts[1], f.scope)
			info := f.l.env.automataInfo[binding.instanceAutomata]
			if parts[0] == "state" {
				for _, field := range info.Decl.StateFields {
					if field.Name == e.Field {
						path := binding.cName + ".shared." + e.Field
						if field.Type.isReference() && field.Type.ArrayElem == nil {
							return "", "(*(" + path + "))", field.Type.borrowBase()
						}
						return "", path, field.Type
					}
				}
				return "", "/* invalid_state_field */", Type{}
			}
			for _, machine := range info.Decl.Machines {
				if machine.Name == parts[2] {
					for _, field := range machine.Fields {
						if field.Name == e.Field {
							path := binding.cName + "." + evt1MachineFramesCName(parts[2]) + "[0]." + e.Field
							if field.Type.isReference() && field.Type.ArrayElem == nil {
								return "", "(*(" + path + "))", field.Type.borrowBase()
							}
							return "", path, field.Type
						}
					}
				}
			}
		}
		prelude, recv, recvType := f.lowerExpr(e.Receiver, indent)
		if e.DynInterface != "" {
			if f.plannedStrategy("dyn_field_get", "WitnessFieldAccessor") != "WitnessFieldAccessor" {
				return prelude, "0", Type{Name: "int", Kind: TypeBuiltin}
			}
			_, fields, _ := evt1InterfaceRuntimeRequirements(f.l.env, e.DynInterface, map[string]bool{})
			for _, field := range fields {
				if field.Name == e.Field {
					return prelude, fmt.Sprintf("(%s).witness->get_%s((%s).object)", recv, e.Field, recv), field.Type.valueType()
				}
			}
		}
		if region, ok := evt1LayoutRegion(f.l.env, recvType.Name, e.Field); ok {
			return f.lowerRegionProjection(prelude, recv, recvType, region)
		}
		fieldType := f.l.env.fieldSets[recvType.borrowBase().Name][e.Field]
		op := "."
		if recvType.isBorrowLike() {
			op = "->"
		}
		fieldExpr := recv + op + e.Field
		if fieldType.isReference() && fieldType.ArrayElem != nil {
			return prelude, fieldExpr, fieldType
		}
		if fieldType.isReference() {
			return prelude, "(*(" + fieldExpr + "))", fieldType.borrowBase()
		}
		return prelude, fieldExpr, fieldType
	case *IndexExpr:
		if e.InferenceIndex {
			prelude, base, _ := f.lowerExpr(e.Base, indent)
			return prelude, fmt.Sprintf("(%s).probabilities[%d]", base, e.CandidateTag), Type{Name: "float", Kind: TypeBuiltin, Span: e.Span}
		}
		if e.TensorIndex {
			return f.lowerTensorIndex(e, indent)
		}
		if e.SpanIndex {
			return f.lowerSpanIndex(e, indent)
		}
		return f.lowerStorageIndex(e, indent, false)
	case *ArrayLiteralExpr:
		return "", "/* array_literal_requires_target */", Type{}
	case *FloatLiteral:
		literal := fmt.Sprintf("%g", e.Value)
		if !strings.ContainsAny(literal, ".eE") {
			literal += ".0"
		}
		floatType, _ := evt1BuiltinType("float", e.Span)
		return "", literal + "f", floatType
	case *InferExpr:
		return f.lowerInferenceExpr(e, indent)
	case *BinaryExpr:
		if e.Tensor != nil && e.Tensor.Kind == "tensor_scalar_contract" {
			return f.lowerScalarTensorContract(e, indent)
		}
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
	case *BindExpr:
		return "", "/* bind_requires_target */", Type{}
	case *RefExpr:
		if e.CallbackErase && e.CallbackType != nil {
			prelude, target, targetType, _ := f.lowerLValue(e.Value, indent)
			return prelude, fmt.Sprintf("(%s){ .environment = (void*)&(%s), .invoke = &%s }", evt1CallbackCName(*e.CallbackType), target, evt1CallbackAdapterCName(targetType.CallableID, *e.CallbackType)), *e.CallbackType
		}
		prelude, target, targetType, _ := f.lowerLValue(e.Value, indent)
		if e.DynInterface != "" {
			if f.plannedStrategy("dyn_make", "ObjectWitnessPair") != "ObjectWitnessPair" {
				return prelude, "0", Type{Name: "int", Kind: TypeBuiltin}
			}
			dynType := Type{Name: e.DynInterface, Kind: TypeDyn, Const: e.Const, Span: e.Span}
			object := "(void*)&(" + target + ")"
			if e.Const {
				object = "(const void*)&(" + target + ")"
			}
			return prelude, fmt.Sprintf("(%s){ .object = %s, .witness = &%s }", evt1CType(dynType), object, evt1WitnessTableCName(e.WitnessID)), dynType
		}
		out := targetType.valueType()
		out.Ownership = "ref"
		out.Const = e.Const
		if evt1IsSemanticViewType(f.l.env, out) {
			return prelude, target, out
		}
		if out.ArrayElem != nil {
			return f.lowerStorageView(e.Value, out, false, e.Span, indent)
		}
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
		if e.CallableInvoke && e.CallableType != nil {
			binding, _ := scopeLookup(e.Callee, f.scope)
			var prelude strings.Builder
			var args []string
			for _, arg := range e.Args {
				argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
				prelude.WriteString(argPrelude)
				temp := f.nextTemp("callback_arg")
				prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
				args = append(args, temp)
			}
			if e.CallableType.Kind == TypeCallback {
				return prelude.String(), fmt.Sprintf("%s.invoke(%s.environment%s)", binding.cName, binding.cName, evt1CArgsSuffix(args)), *e.CallableType.CallableResult
			}
			return prelude.String(), fmt.Sprintf("%s(&%s%s)", evt1CallableInvokeCName(e.CallableType.CallableID), binding.cName, evt1CArgsSuffix(args)), *e.CallableType.CallableResult
		}
		if e.Intrinsic == "async_step" || e.Intrinsic == "async_complete" || e.Intrinsic == "async_result" {
			prelude, value, valueType := f.lowerExpr(e.Args[0], indent)
			if e.Intrinsic == "async_step" {
				return prelude, "concept_async_step(&" + value + ")", Type{Name: "void", Kind: TypeBuiltin, Span: e.Span}
			}
			if e.Intrinsic == "async_complete" {
				return prelude, "concept_async_complete(&" + value + ")", Type{Name: "bool", Kind: TypeBuiltin, Span: e.Span}
			}
			resultType := valueType.TypeArgs[0]
			temp := f.nextTemp("async_result")
			prelude += ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(resultType), temp)
			prelude += ind(indent) + fmt.Sprintf("concept_async_result(&%s, &%s, sizeof(%s));\n", value, temp, temp)
			return prelude, temp, resultType
		}
		if e.Intrinsic == "machine_result" || e.Intrinsic == "machine_result_inner" {
			var automataName, instanceExpr, machineName string
			if e.Intrinsic == "machine_result_inner" {
				automataName, instanceExpr = f.automataStepName, "instance"
				machineName = e.Args[0].(*NameExpr).Name
			} else {
				instanceName := e.Args[0].(*NameExpr).Name
				binding, _ := scopeLookup(instanceName, f.scope)
				automataName, instanceExpr = binding.instanceAutomata, "&"+binding.cName
				machineName = e.Args[1].(*NameExpr).Name
			}
			name := evt1CName(automataName) + "_result_" + evt1PayloadFieldName(machineName)
			return "", name + "(" + instanceExpr + ")", Type{Name: evt1MachineOutcomeTypeName(automataName, machineName), Kind: TypeStruct, Span: e.Span}
		}
		if e.Intrinsic == "inference_HardMax" || e.Intrinsic == "inference_Confidence" {
			prelude, value, inferenceType := f.lowerExpr(e.Args[0], indent)
			valueTemp := f.nextTemp("inference_query")
			indexTemp := f.nextTemp("inference_index")
			bestTemp := f.nextTemp("inference_best")
			var b strings.Builder
			b.WriteString(prelude)
			b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(inferenceType), valueTemp, value))
			b.WriteString(ind(indent) + fmt.Sprintf("uint32_t %s = 0u;\n", indexTemp))
			b.WriteString(ind(indent) + fmt.Sprintf("float %s = %s.probabilities[%s.candidate_order[0]];\n", bestTemp, valueTemp, valueTemp))
			loopTemp := f.nextTemp("inference_i")
			b.WriteString(ind(indent) + fmt.Sprintf("for (uint32_t %s = 1u; %s < %s.candidate_count; ++%s) {\n", loopTemp, loopTemp, valueTemp, loopTemp))
			b.WriteString(ind(indent+1) + fmt.Sprintf("float probability = %s.probabilities[%s.candidate_order[%s]];\n", valueTemp, valueTemp, loopTemp))
			b.WriteString(ind(indent+1) + fmt.Sprintf("if (probability > %s) { %s = probability; %s = %s; }\n", bestTemp, bestTemp, indexTemp, loopTemp))
			b.WriteString(ind(indent) + "}\n")
			if e.Intrinsic == "inference_Confidence" {
				return b.String(), bestTemp, Type{Name: "float", Kind: TypeBuiltin, Span: e.Span}
			}
			candidateType := evt1InferenceCandidateType(inferenceType)
			resultTemp := f.nextTemp("inference_candidate")
			b.WriteString(ind(indent) + fmt.Sprintf("%s %s = {0};\n", evt1CType(candidateType), resultTemp))
			b.WriteString(ind(indent) + fmt.Sprintf("%s.tag = %s.candidate_order[%s];\n", resultTemp, valueTemp, indexTemp))
			return b.String(), resultTemp, candidateType
		}
		if e.Intrinsic == "step_machine" || e.Intrinsic == "state_machine" {
			instanceName := e.Args[0].(*NameExpr).Name
			machineName := e.Args[1].(*NameExpr).Name
			binding, _ := scopeLookup(instanceName, f.scope)
			if e.Intrinsic == "step_machine" {
				return "", evt1AutomataStepCName(binding.instanceAutomata, machineName) + "(&" + binding.cName + ")", Type{Name: "void", Kind: TypeBuiltin, Span: e.Span}
			}
			if f.l.env.automataInfo[binding.instanceAutomata].Decl.SignalType.Name == "" {
				return "", binding.cName + "." + evt1MachineFramesCName(machineName) + "[0].current_state", Type{Name: "int", Kind: TypeBuiltin, Span: e.Span}
			}
			return "", binding.cName + "." + evt1PayloadFieldName(machineName) + ".current_state", Type{Name: "int", Kind: TypeBuiltin, Span: e.Span}
		}
		if e.Member {
			if e.DynDispatch {
				if f.plannedStrategy("dyn_call", "WitnessIndirect") != "WitnessIndirect" {
					return "", "0", Type{Name: "int", Kind: TypeBuiltin}
				}
				recvPrelude, recv, recvType := f.lowerExpr(e.Receiver, indent)
				decl := f.l.env.concepts[recvType.Name]
				methods, _, _ := evt1InterfaceRuntimeRequirements(f.l.env, recvType.Name, map[string]bool{})
				var req OperationRequirement
				for _, candidate := range methods {
					if candidate.Name == e.Callee {
						req = candidate
						break
					}
				}
				var prelude strings.Builder
				prelude.WriteString(recvPrelude)
				args := []string{"(" + recv + ").object"}
				for i, arg := range e.Args {
					argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
					prelude.WriteString(argPrelude)
					param := evt1SubstituteType(req.Params[i+1].Type, decl.TypeParam, Type{Name: "void", Kind: TypeBuiltin})
					if param.isBorrowLike() && !argType.isBorrowLike() {
						argExpr = "&" + argExpr
					}
					args = append(args, argExpr)
				}
				return prelude.String(), fmt.Sprintf("(%s).witness->%s(%s)", recv, e.Callee, strings.Join(args, ", ")), req.ReturnType
			}
			copy := *e
			copy.Member, copy.Receiver = false, nil
			selfConst := false
			if t, err := validateExpr(f.l.env, f.typeScope(), e.Receiver, nil, false); err == nil {
				if place, placeErr := validateAssignable(f.l.env, f.typeScope(), e.Receiver, nil); placeErr == nil {
					selfConst = !place.mutable || t.Const
				}
			}
			copy.Args = append([]Expr{&RefExpr{Value: e.Receiver, Const: selfConst, Span: e.Span}}, e.Args...)
			return f.lowerExpr(&copy, indent)
		}
		if e.Intrinsic == "tensor_view" {
			return f.lowerTensorView(e, indent)
		}
		if e.Intrinsic == "span_from_region" || e.Intrinsic == "span_to_readonly" || e.Intrinsic == "span_subregion" {
			return f.lowerSpanCall(e, indent)
		}
		if e.Callee == "Len" || e.Callee == "Rank" || e.Callee == "Shape" {
			if e.Callee == "Len" && len(e.Args) == 1 {
				if spanType, err := validateExpr(f.l.env, f.typeScope(), e.Args[0], nil, false); err == nil && evt1IsSpanType(spanType) {
					prelude, value, _ := f.lowerExpr(e.Args[0], indent)
					return prelude, fmt.Sprintf("(int)(%s).length", value), Type{Name: "int", Kind: TypeBuiltin, Span: e.Span}
				}
			}
			return f.lowerStorageQuery(e, indent)
		}
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
		resultType := fn.ReturnType
		if fn.Async {
			resultType = evt1AsyncType(fn.ReturnType, fn.Name, e.Span)
		}
		return prelude.String(), evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, fn) + "(" + strings.Join(args, ", ") + ")", resultType
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
		if e.Callee == "LayoutSize" || e.Callee == "LayoutAlign" || e.Callee == "LayoutOffset" {
			value, _ := evt1LayoutQuery(f.l.env, e.Callee, e.TypeArg, e.Args)
			t, _ := evt1BuiltinType("int", e.Span)
			return "", fmt.Sprintf("%d", value), t
		}
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
		resultType := instance.Function.ReturnType
		if instance.Function.Async {
			resultType = evt1AsyncType(resultType, instance.GeneratedSymbol, e.Span)
		}
		return prelude.String(), instance.GeneratedSymbol + "(" + strings.Join(args, ", ") + ")", resultType
	case *ConstructExpr:
		enumType := Type{Name: e.EnumName, Kind: TypeEnum, Span: e.Span}
		if evt1IsFailureType(e.ResolvedType) {
			enumType = e.ResolvedType
		}
		var prelude strings.Builder
		var args []string
		var payload []Field
		if decl, ok := evt1FailureEnumDecl(enumType); ok {
			if variant, found := evt1LookupVariant(decl, e.VariantName); found {
				payload = variant.Payload
			}
		}
		for i, arg := range e.Args {
			var argPrelude, argExpr string
			var argType Type
			if i < len(payload) {
				argPrelude, argExpr, argType = f.lowerExprExpected(arg, payload[i].Type, indent)
			} else {
				argPrelude, argExpr, argType = f.lowerExpr(arg, indent)
			}
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
		var initializers []string
		structDecl := f.l.env.structs[e.StructName]
		for i, arg := range e.Args {
			argPrelude, argExpr, argType := f.lowerExpr(arg, indent)
			prelude.WriteString(argPrelude)
			temp := f.nextTemp("field")
			prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(argType), temp, argExpr))
			if i < len(structDecl.Fields) {
				args = append(args, temp)
				initializers = append(initializers, "."+structDecl.Fields[i].Name+" = "+temp)
			}
		}
		if !evt1TypeCopyable(f.l.env, structType) {
			return prelude.String(), "(" + evt1CType(structType) + "){ " + strings.Join(initializers, ", ") + " }", structType
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
	if bind, ok := expr.(*BindExpr); ok {
		if evt1IsSemanticViewType(f.l.env, expected) {
			return f.lowerSemanticView(bind.Source, expected, indent)
		}
		return f.lowerStorageView(bind.Source, expected, bind.RuntimeCheck, bind.Span, indent)
	}
	if literal, ok := expr.(*ArrayLiteralExpr); ok && expected.ArrayElem != nil {
		return f.lowerStorageLiteral(*literal, expected, indent)
	}
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

func (f *evt1FunctionLowerer) lowerSemanticView(source Expr, target Type, indent int) (string, string, Type) {
	prelude, value, sourceType := f.lowerExpr(source, indent)
	data := fmt.Sprintf("(%s).data", value)
	if sourceType.ArrayElem != nil {
		data = fmt.Sprintf("(unsigned char*)(%s).data", value)
	}
	if target.Const {
		data = fmt.Sprintf("(const unsigned char*)%s", data)
	}
	return prelude, fmt.Sprintf("(%s){ .data = %s }", evt1CType(target), data), target
}

func (f *evt1FunctionLowerer) lowerRegionProjection(prelude, receiver string, receiverType Type, region LayoutRegion) (string, string, Type) {
	if region.Type.ArrayElem != nil {
		view := region.Type
		view.Ownership = "ref"
		view.Const = receiverType.Const
		pointerType := evt1CType(*region.Type.ArrayElem)
		if view.Const {
			pointerType = "const " + pointerType
		}
		shape := make([]string, 0, len(region.Type.Shape))
		for _, dimension := range region.Type.Shape {
			shape = append(shape, fmt.Sprintf("%d", dimension.Extent))
		}
		expr := fmt.Sprintf("(%s){ .data = (%s*)((%s).data + %d), .shape = { %s } }", evt1CType(view), pointerType, receiver, region.Offset, strings.Join(shape, ", "))
		return prelude, expr, view
	}
	pointerType := evt1CType(region.Type)
	if receiverType.Const {
		pointerType = "const " + pointerType
	}
	expr := fmt.Sprintf("(*((%s*)((%s).data + %d)))", pointerType, receiver, region.Offset)
	return prelude, expr, region.Type
}

func (f *evt1FunctionLowerer) lowerLValue(expr Expr, indent int) (string, string, Type, bool) {
	switch e := expr.(type) {
	case *NameExpr:
		binding, _ := scopeLookup(e.Name, f.scope)
		if binding.t.isReference() && (binding.t.ArrayElem != nil || evt1IsSemanticViewType(f.l.env, binding.t)) {
			return "", binding.cName, binding.t, false
		}
		if binding.t.isReference() {
			return "", "(*" + binding.cName + ")", binding.t.borrowBase(), false
		}
		return "", binding.cName, binding.t, true
	case *FieldExpr:
		prelude, recv, recvType, _ := f.lowerLValue(e.Receiver, indent)
		if region, ok := evt1LayoutRegion(f.l.env, recvType.Name, e.Field); ok {
			projectionPrelude, value, t := f.lowerRegionProjection(prelude, recv, recvType, region)
			return projectionPrelude, value, t, false
		}
		fieldType := f.l.env.fieldSets[recvType.borrowBase().Name][e.Field]
		op := "."
		if recvType.isBorrowLike() {
			op = "->"
		}
		fieldExpr := recv + op + e.Field
		if fieldType.isReference() && fieldType.ArrayElem != nil {
			return prelude, fieldExpr, fieldType, false
		}
		if fieldType.isReference() {
			return prelude, "(*(" + fieldExpr + "))", fieldType.borrowBase(), false
		}
		return prelude, fieldExpr, fieldType, false
	case *IndexExpr:
		if e.SpanIndex {
			prelude, value, valueType := f.lowerSpanIndex(e, indent)
			return prelude, value, valueType, false
		}
		prelude, value, valueType := f.lowerStorageIndex(e, indent, true)
		return prelude, value, valueType, false
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
	if f.plan != nil && f.plan.Cleanup.Strategy != "ReverseDeclarationOrder" {
		return ind(indent) + "/* invalid cleanup plan */\n"
	}
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
		if binding.instanceAutomata != "" && f.l.env.automataInfo[binding.instanceAutomata].Decl.SignalType.Name == "" {
			b.WriteString(ind(indent) + fmt.Sprintf("%s(&%s);\n", evt1AutomataDropCName(binding.instanceAutomata), name))
		} else {
			b.WriteString(f.lowerDropValue(binding.t, name, indent))
		}
		f.liveOwners[name] = false
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerDropValue(t Type, value string, indent int) string {
	if t.Kind == TypeCallable {
		var b strings.Builder
		for _, callable := range evt1ModuleCallables(f.l.module) {
			if callable.Identity != t.CallableID {
				continue
			}
			for i := len(callable.Captures) - 1; i >= 0; i-- {
				capture := callable.Captures[i]
				if capture.Kind == CaptureRef || capture.Kind == CaptureRefConst {
					continue
				}
				drop := f.lowerDropValue(capture.Type, "("+value+")."+capture.Name, indent)
				if drop != "" && capture.Kind == CaptureMove && evt1TypeHasDrop(f.l.env, capture.Type) {
					b.WriteString(ind(indent) + "if ((" + value + ").__live_" + capture.Name + ") {\n")
					b.WriteString(f.lowerDropValue(capture.Type, "("+value+")."+capture.Name, indent+1))
					b.WriteString(ind(indent) + "}\n")
				} else {
					b.WriteString(drop)
				}
			}
			break
		}
		return b.String()
	}
	if evt1IsFailureType(t) && evt1FailureNeedsDrop(f.l.env, t) {
		return ind(indent) + fmt.Sprintf("%s(%s);\n", evt1FailureDropName(t), value)
	}
	if dropFn := evt1DropFunction(f.l.env, t); dropFn != nil {
		return ind(indent) + fmt.Sprintf("%s(%s);\n", evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, *dropFn), value)
	}
	if t.ArrayElem != nil && evt1StorageElementHasDrop(f.l.env, *t.ArrayElem) {
		var b strings.Builder
		for index := evt1StorageElementCount(t) - 1; index >= 0; index-- {
			b.WriteString(f.lowerDropValue(*t.ArrayElem, fmt.Sprintf("(%s).data[%d]", value, index), indent))
		}
		return b.String()
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
		return !evt1StorageHasRuntimeShape(t) && evt1RuntimeTypeSafe(env, *t.ArrayElem)
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
	for _, witness := range m.Witnesses {
		lines = append(lines, fmt.Sprintf("interface_witness %s interface %s concrete %s no_allocation=%t", witness.ID, witness.Interface, witness.ConcreteType, witness.NoAllocation))
		for _, method := range witness.Methods {
			lines = append(lines, fmt.Sprintf("interface_witness %s method %s", witness.ID, method))
		}
		for _, method := range witness.AsyncMethods {
			lines = append(lines, fmt.Sprintf("interface_witness %s async_constructor %s signature %s machine %s", witness.ID, method.Name, method.Signature, method.MachineIdentity))
		}
		for _, field := range witness.FieldGetters {
			lines = append(lines, fmt.Sprintf("interface_witness %s field_get %s", witness.ID, field))
		}
		for _, field := range witness.FieldSetters {
			lines = append(lines, fmt.Sprintf("interface_witness %s field_set %s", witness.ID, field))
		}
	}
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
