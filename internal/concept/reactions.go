package concept

import (
	"fmt"
	"sort"
	"strings"
)

// Input reactions (`on`) are the state-level form for automata declared
// `with input`. See OnStmt for the selection rule.

const evt1StepOutcomeTypeName = "StepOutcome"

func evt1BuiltinStepOutcomeEnum() EnumDecl {
	return EnumDecl{
		Name: evt1StepOutcomeTypeName,
		Variants: []VariantDecl{
			{Name: "Transitioned", Tag: 0},
			{Name: "Unhandled", Tag: 1},
			{Name: "Ambiguous", Tag: 2},
			{Name: "Finished", Tag: 3},
			{Name: "AlreadyFinished", Tag: 4},
		},
	}
}

func evt1StateHasReactions(body Block) bool {
	for _, stmt := range body.Statements {
		if _, ok := stmt.(*OnStmt); ok {
			return true
		}
	}
	return false
}

// evt1StateReactionGroups returns the reactions of a state grouped by input
// variant in first-appearance order, plus the state-level catch-all.
func evt1StateReactionGroups(body Block) (order []string, groups map[string][]*OnStmt, catchAll *OnStmt) {
	groups = map[string][]*OnStmt{}
	for _, stmt := range body.Statements {
		on, ok := stmt.(*OnStmt)
		if !ok {
			continue
		}
		if on.CatchAll {
			catchAll = on
			continue
		}
		key := on.Pattern.VariantName
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], on)
	}
	return order, groups, catchAll
}

func evt1ValidateStateReactions(env *semanticEnv, machineScope *evt1Scope, inputType Type, inputEnum EnumDecl, state *StateDecl) error {
	var catchAll *OnStmt
	type variantReactions struct {
		unguarded, guarded, fallback int
		guards                       map[string]bool
	}
	byVariant := map[string]*variantReactions{}
	for _, stmt := range state.Body.Statements {
		on, ok := stmt.(*OnStmt)
		if !ok {
			return evt1Diagnostic("ON_MIXED_STATE_BODY", fmt.Sprintf("state %s reacts with `on`, so its body may contain only `on` and `otherwise` reactions; move the work into a reaction body", state.Name), stmt.statementSpan())
		}
		armScope := newEVT1Scope(machineScope)
		if on.CatchAll {
			if catchAll != nil {
				return evt1Diagnostic("ON_DUPLICATE", fmt.Sprintf("state %s has more than one `otherwise` reaction", state.Name), on.Span)
			}
			catchAll = on
		} else {
			var err error
			armScope, _, err = validatePattern(env, machineScope, inputType, inputEnum, on.Pattern, map[string]bool{})
			if err != nil {
				return err
			}
			key := on.Pattern.EnumName + "::" + on.Pattern.VariantName
			r := byVariant[key]
			if r == nil {
				r = &variantReactions{guards: map[string]bool{}}
				byVariant[key] = r
			}
			switch {
			case on.Otherwise:
				r.fallback++
				if r.fallback > 1 {
					return evt1Diagnostic("ON_DUPLICATE", fmt.Sprintf("state %s has more than one `on %s otherwise` reaction", state.Name, key), on.Span)
				}
			case on.Guard == nil:
				r.unguarded++
				if r.unguarded > 1 {
					return evt1Diagnostic("ON_OVERLAP", fmt.Sprintf("state %s reacts to %s twice without guards; the reactions always overlap", state.Name, key), on.Span)
				}
			default:
				r.guarded++
				identity := evt1ExprIdentity(on.Guard)
				if r.guards[identity] {
					return evt1Diagnostic("ON_OVERLAP", fmt.Sprintf("state %s reacts to %s twice with the same guard %s", state.Name, key, identity), on.Span)
				}
				r.guards[identity] = true
				guardType, err := validateExpr(env, armScope, on.Guard, nil, false)
				if err != nil {
					return err
				}
				if guardType.Name != "bool" {
					return evt1Diagnostic("ON_GUARD_REQUIRES_BOOL", fmt.Sprintf("`on` guard must be bool, got %s", guardType.String()), on.Guard.exprSpan())
				}
			}
			if r.unguarded > 0 && r.guarded > 0 {
				return evt1Diagnostic("ON_OVERLAP", fmt.Sprintf("state %s reacts to %s both unguarded and guarded; the unguarded reaction overlaps every guard, so write it as `on %s otherwise`", state.Name, key, key), on.Span)
			}
			if r.unguarded > 0 && r.fallback > 0 {
				return evt1Diagnostic("ON_OTHERWISE_UNREACHABLE", fmt.Sprintf("`on %s otherwise` in state %s can never run: an unguarded reaction always handles %s", key, state.Name, key), on.Span)
			}
		}
		if err := evt1ValidateMachineControlFlow(&on.Body); err != nil {
			return err
		}
		if err := validateBlock(env, armScope, Type{Name: "void", Kind: TypeBuiltin}, on.Body, nil, false); err != nil {
			return err
		}
	}
	var fallbackOnly []string
	for key, r := range byVariant {
		if r.fallback > 0 && r.guarded == 0 {
			fallbackOnly = append(fallbackOnly, key)
		}
	}
	if len(fallbackOnly) > 0 {
		sort.Strings(fallbackOnly)
		return evt1Diagnostic("ON_OTHERWISE_WITHOUT_GUARDS", fmt.Sprintf("state %s: `on %s otherwise` has no guarded reaction to fall back from; write `on %s =>`", state.Name, fallbackOnly[0], fallbackOnly[0]), state.Span)
	}
	if catchAll != nil {
		covered := 0
		for _, variant := range inputEnum.Variants {
			if r := byVariant[inputEnum.Name+"::"+variant.Name]; r != nil && (r.unguarded > 0 || r.fallback > 0) {
				covered++
			}
		}
		if covered == len(inputEnum.Variants) {
			return evt1Diagnostic("ON_OTHERWISE_UNREACHABLE", fmt.Sprintf("`otherwise` in state %s can never run: every %s variant is always handled", state.Name, inputEnum.Name), catchAll.Span)
		}
	}
	return nil
}

func evt1ReactionSummary(on *OnStmt) string {
	var parts []string
	if on.CatchAll {
		parts = append(parts, "otherwise")
	} else {
		parts = append(parts, on.Pattern.EnumName+"::"+on.Pattern.VariantName)
		if on.Guard != nil {
			parts = append(parts, "when "+evt1ExprIdentity(on.Guard))
		}
		if on.Otherwise {
			parts = append(parts, "otherwise")
		}
	}
	return strings.Join(parts, " ")
}

func evt1ModuleUsesStepOutcome(module Module) bool {
	for _, decl := range module.Automata {
		if decl.InputType.Name != "" {
			return true
		}
	}
	return evt1TypeUsed(module, func(t Type) bool { return t.Name == evt1StepOutcomeTypeName })
}

func evt1StepOutcomeValue(name string) string {
	return evt1ConstructorName(evt1StepOutcomeTypeName, name) + "()"
}

func evt1AutomataStepInputCName(automataName, machineName string) string {
	return evt1AutomataStepCName(automataName, machineName) + "_input"
}

// lowerStateReactions lowers the `on` reactions of one state. The input was
// copied into instance->input by the Step wrapper, which also preset
// instance->step_outcome to Transitioned.
func (f *evt1FunctionLowerer) lowerStateReactions(body Block, inputType Type, indent int) string {
	enumDecl := f.l.env.enums[inputType.Name]
	order, groups, catchAll := evt1StateReactionGroups(body)
	input := f.nextTemp("input")
	var b strings.Builder
	setOutcome := func(name string, at int) {
		b.WriteString(ind(at) + fmt.Sprintf("instance->step_outcome = %s;\n", evt1StepOutcomeValue(name)))
		b.WriteString(ind(at) + "return;\n")
	}
	bindPayload := func(on *OnStmt, variant VariantDecl, at int) {
		for i, binding := range on.Pattern.Bindings {
			field := variant.Payload[i]
			cName := f.bindName(binding, field.Type)
			b.WriteString(ind(at) + fmt.Sprintf("%s %s = %s.payload.%s.%s;\n", evt1CType(field.Type), cName, input, evt1PayloadFieldName(variant.Name), field.Name))
			b.WriteString(ind(at) + fmt.Sprintf("(void)%s;\n", cName))
		}
	}
	// runBody lowers a reaction body; a body that does not transition keeps
	// the current state, so the Step ends after it either way.
	runBody := func(on *OnStmt, variant *VariantDecl, at int) {
		b.WriteString(ind(at) + "{\n")
		f.pushScope()
		if variant != nil {
			bindPayload(on, *variant, at+1)
		}
		b.WriteString(f.lowerBlock(on.Body, at+1))
		f.popScope()
		b.WriteString(ind(at+1) + "return;\n")
		b.WriteString(ind(at) + "}\n")
	}
	fallback := func(at int) {
		if catchAll != nil {
			runBody(catchAll, nil, at)
			return
		}
		setOutcome("Unhandled", at)
	}
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = instance->input;\n", evt1CType(inputType), input))
	b.WriteString(ind(indent) + fmt.Sprintf("switch (%s.tag) {\n", input))
	for _, variantName := range order {
		variant, _ := evt1LookupVariant(enumDecl, variantName)
		b.WriteString(ind(indent) + fmt.Sprintf("case %s:\n", evt1TagName(enumDecl.Name, variant.Name)))
		var candidates []*OnStmt
		var variantFallback *OnStmt
		for _, on := range groups[variantName] {
			if on.Otherwise {
				variantFallback = on
			} else {
				candidates = append(candidates, on)
			}
		}
		at := indent + 1
		if len(candidates) == 1 && candidates[0].Guard == nil {
			runBody(candidates[0], &variant, at)
			continue
		}
		b.WriteString(ind(at) + "{\n")
		hits := f.nextTemp("reaction_hits")
		b.WriteString(ind(at+1) + fmt.Sprintf("int %s = 0;\n", hits))
		guards := make([]string, len(candidates))
		for i, on := range candidates {
			guards[i] = f.nextTemp("reaction_guard")
			b.WriteString(ind(at+1) + fmt.Sprintf("bool %s;\n", guards[i]))
			b.WriteString(ind(at+1) + "{\n")
			f.pushScope()
			bindPayload(on, variant, at+2)
			prelude, value, _ := f.lowerExpr(on.Guard, at+2)
			b.WriteString(prelude)
			b.WriteString(ind(at+2) + fmt.Sprintf("%s = %s;\n", guards[i], value))
			f.popScope()
			b.WriteString(ind(at+1) + "}\n")
			b.WriteString(ind(at+1) + fmt.Sprintf("if (%s) %s++;\n", guards[i], hits))
		}
		b.WriteString(ind(at+1) + fmt.Sprintf("if (%s > 1) {\n", hits))
		setOutcome("Ambiguous", at+2)
		b.WriteString(ind(at+1) + "}\n")
		for i, on := range candidates {
			b.WriteString(ind(at+1) + fmt.Sprintf("if (%s) ", guards[i]))
			b.WriteString("{\n")
			runBody(on, &variant, at+2)
			b.WriteString(ind(at+1) + "}\n")
		}
		if variantFallback != nil {
			runBody(variantFallback, &variant, at+1)
		} else {
			fallback(at + 1)
		}
		b.WriteString(ind(at) + "}\n")
	}
	b.WriteString(ind(indent) + "default:\n")
	b.WriteString(ind(indent+1) + "{\n")
	fallback(indent + 2)
	b.WriteString(ind(indent+1) + "}\n")
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}
