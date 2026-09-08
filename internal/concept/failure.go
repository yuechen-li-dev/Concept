package concept

import (
	"fmt"
	"sort"
	"strings"
)

const (
	evt1OptionName = "Option"
	evt1ResultName = "Result"
)

func evt1IsOptionType(t Type) bool  { return t.Name == evt1OptionName && len(t.TypeArgs) == 1 }
func evt1IsResultType(t Type) bool  { return t.Name == evt1ResultName && len(t.TypeArgs) == 2 }
func evt1IsFailureType(t Type) bool { return evt1IsOptionType(t) || evt1IsResultType(t) }

func evt1FailureSuccessType(t Type) Type {
	if evt1IsFailureType(t) {
		return t.TypeArgs[0]
	}
	return Type{}
}

func evt1FailureErrorType(t Type) Type {
	if evt1IsResultType(t) {
		return t.TypeArgs[1]
	}
	return Type{}
}

func evt1FailureEnumDecl(t Type) (EnumDecl, bool) {
	if evt1IsOptionType(t) {
		return EnumDecl{Name: evt1OptionName, Variants: []VariantDecl{
			{Name: "Some", Tag: 0, Payload: []Field{{Name: "value", Type: t.TypeArgs[0], Span: t.Span}}},
			{Name: "None", Tag: 1},
		}}, true
	}
	if evt1IsResultType(t) {
		ok := VariantDecl{Name: "Ok", Tag: 0}
		if t.TypeArgs[0].Name != "void" {
			ok.Payload = []Field{{Name: "value", Type: t.TypeArgs[0], Span: t.Span}}
		}
		return EnumDecl{Name: evt1ResultName, Variants: []VariantDecl{
			ok,
			{Name: "Error", Tag: 1, Payload: []Field{{Name: "error", Type: t.TypeArgs[1], Span: t.Span}}},
		}}, true
	}
	return EnumDecl{}, false
}

func evt1FailureTypeKey(t Type) string { return evt1TypeIdentity(t.valueType()) }

func evt1FailureCName(t Type) string {
	parts := []string{t.Name}
	for _, arg := range t.TypeArgs {
		parts = append(parts, evt1TypeCIdentity(arg))
	}
	return evt1CName(strings.Join(parts, "_"))
}

func evt1TypeCIdentity(t Type) string {
	s := t.String()
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func evt1FailureConstructorName(t Type, variant string) string {
	return evt1FailureCName(t) + "_make_" + evt1PayloadFieldName(variant)
}

func evt1FailureDropName(t Type) string { return evt1FailureCName(t) + "_drop" }

func evt1FailureNeedsDrop(env *semanticEnv, t Type) bool {
	if !evt1IsFailureType(t) {
		return false
	}
	decl, _ := evt1FailureEnumDecl(t)
	for _, variant := range decl.Variants {
		for _, field := range variant.Payload {
			if evt1TypeHasDrop(env, field.Type) {
				return true
			}
		}
	}
	return false
}

func evt1FailureDropFunction(env *semanticEnv, t Type, outputBase string) string {
	if !evt1FailureNeedsDrop(env, t) {
		return ""
	}
	decl, _ := evt1FailureEnumDecl(t)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static void %s(%s value) {\n  switch (value.tag) {\n", evt1FailureDropName(t), evt1FailureCName(t)))
	for _, variant := range decl.Variants {
		var drops []string
		for _, field := range variant.Payload {
			if !evt1TypeHasDrop(env, field.Type) {
				continue
			}
			value := fmt.Sprintf("value.payload.%s.%s", evt1PayloadFieldName(variant.Name), field.Name)
			if evt1IsFailureType(field.Type) {
				drops = append(drops, fmt.Sprintf("%s(%s);", evt1FailureDropName(field.Type), value))
			} else if dropFn := evt1DropFunction(env, field.Type); dropFn != nil {
				drops = append(drops, fmt.Sprintf("%s(%s);", evt1FunctionSymbolForDecl(outputBase, env, *dropFn), value))
			}
		}
		b.WriteString(fmt.Sprintf("  case %d:\n", variant.Tag))
		for _, drop := range drops {
			b.WriteString("    " + drop + "\n")
		}
		b.WriteString("    break;\n")
	}
	b.WriteString("  default:\n    concept_abort_invalid_tag(\"")
	b.WriteString(t.Name)
	b.WriteString("\");\n  }\n}\n\n")
	return b.String()
}

func evt1FailureTypeDecl(t Type) string {
	name := evt1FailureCName(t)
	decl, _ := evt1FailureEnumDecl(t)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("typedef struct %s {\n  int tag;\n  union {\n", name))
	for _, variant := range decl.Variants {
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
	// The Vulkan actuator bridge predates R4c and initializes Result<void,E>
	// from handwritten C. Keep its two fields as an ABI compatibility seam;
	// Concept source and all R4c lowering use the canonical tag/payload form.
	if evt1IsResultType(t) {
		b.WriteString(fmt.Sprintf("  bool is_error;\n  %s error;\n", evt1CType(t.TypeArgs[1])))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", name))
	return b.String()
}

func evt1FailureConstructors(t Type) string {
	name := evt1FailureCName(t)
	decl, _ := evt1FailureEnumDecl(t)
	var b strings.Builder
	for _, variant := range decl.Variants {
		b.WriteString(fmt.Sprintf("static %s %s(", name, evt1FailureConstructorName(t, variant.Name)))
		for i, field := range variant.Payload {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s %s", evt1CType(field.Type), field.Name))
		}
		b.WriteString(fmt.Sprintf(") {\n  %s out = {0};\n  out.tag = %d;\n", name, variant.Tag))
		for _, field := range variant.Payload {
			b.WriteString(fmt.Sprintf("  out.payload.%s.%s = %s;\n", evt1PayloadFieldName(variant.Name), field.Name, field.Name))
		}
		if evt1IsResultType(t) {
			if variant.Name == "Error" {
				b.WriteString("  out.is_error = true;\n  out.error = error;\n")
			} else {
				b.WriteString("  out.is_error = false;\n")
			}
		}
		b.WriteString("  return out;\n}\n\n")
	}
	return b.String()
}

func evt1CollectFailureTypes(module Module) []Type {
	seen := map[string]Type{}
	var add func(Type)
	add = func(t Type) {
		for _, a := range t.TypeArgs {
			add(a)
		}
		if evt1IsFailureType(t) {
			seen[evt1FailureTypeKey(t)] = t
		}
	}
	for _, fn := range append(append([]FunctionDecl{}, module.Functions...), module.ComptimeFns...) {
		add(fn.ReturnType)
		for _, p := range fn.Params {
			add(p.Type)
		}
		if fn.Body != nil {
			evt1VisitFailureTypesBlock(*fn.Body, add)
		}
	}
	for _, s := range module.Structs {
		for _, f := range s.Fields {
			add(f.Type)
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		di, dj := evt1FailureNestingDepth(seen[keys[i]]), evt1FailureNestingDepth(seen[keys[j]])
		if di != dj {
			return di < dj
		}
		return keys[i] < keys[j]
	})
	out := make([]Type, 0, len(keys))
	for _, k := range keys {
		out = append(out, seen[k])
	}
	return out
}

func evt1FailureNestingDepth(t Type) int {
	depth := 0
	for _, arg := range t.TypeArgs {
		if d := evt1FailureNestingDepth(arg); d > depth {
			depth = d
		}
	}
	if evt1IsFailureType(t) {
		depth++
	}
	return depth
}

func evt1CanonicalFailureTypeKeys(module Module) map[string]bool {
	out := map[string]bool{}
	add := func(t Type) {
		if evt1IsFailureType(t) {
			out[evt1FailureTypeKey(t)] = true
		}
	}
	for _, fn := range module.Functions {
		if fn.Body != nil {
			evt1VisitFailureTypesBlock(*fn.Body, add)
		}
	}
	return out
}

func evt1VisitFailureTypesBlock(block Block, add func(Type)) {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			add(s.Type)
			evt1VisitFailureTypesExpr(s.Value, add)
		case *ReturnStmt:
			if s.Value != nil {
				evt1VisitFailureTypesExpr(s.Value, add)
			}
		case *AssignStmt:
			evt1VisitFailureTypesExpr(s.Value, add)
		case *ExprStmt:
			evt1VisitFailureTypesExpr(s.Value, add)
		case *AssertStmt:
			evt1VisitFailureTypesExpr(s.Condition, add)
		case *MatchStmt:
			evt1VisitFailureTypesExpr(s.Subject, add)
			for _, a := range s.Arms {
				evt1VisitFailureTypesBlock(a.Block, add)
			}
		case *TryStmt:
			evt1VisitFailureTypesBlock(s.Body, add)
			for _, a := range s.Except {
				add(a.ErrorType)
				evt1VisitFailureTypesBlock(a.Body, add)
			}
		case *IfStmt:
			evt1VisitFailureTypesBlock(s.Then, add)
			if s.Else != nil {
				evt1VisitFailureTypesBlock(*s.Else, add)
			}
		case *Block:
			evt1VisitFailureTypesBlock(*s, add)
		}
	}
}

func evt1VisitFailureTypesExpr(expr Expr, add func(Type)) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ConstructExpr:
		add(e.ResolvedType)
		for _, a := range e.Args {
			evt1VisitFailureTypesExpr(a, add)
		}
	case *FailureExpr:
		add(e.ResolvedType)
		evt1VisitFailureTypesExpr(e.Value, add)
	case *CallExpr:
		for _, a := range e.Args {
			evt1VisitFailureTypesExpr(a, add)
		}
	case *MatchExpr:
		evt1VisitFailureTypesExpr(e.Subject, add)
		for _, a := range e.Arms {
			evt1VisitFailureTypesExpr(a.Value, add)
		}
	case *BinaryExpr:
		evt1VisitFailureTypesExpr(e.Left, add)
		evt1VisitFailureTypesExpr(e.Right, add)
	case *ParenExpr:
		evt1VisitFailureTypesExpr(e.Value, add)
	}
}

func evt1ModuleUsesFailurePanic(module Module) bool {
	var exprUses func(Expr) bool
	exprUses = func(expr Expr) bool {
		switch e := expr.(type) {
		case *FailureExpr:
			return e.Op == "!" || exprUses(e.Value)
		case *CallExpr:
			for _, a := range e.Args {
				if exprUses(a) {
					return true
				}
			}
		case *BinaryExpr:
			return exprUses(e.Left) || exprUses(e.Right)
		case *ParenExpr:
			return exprUses(e.Value)
		}
		return false
	}
	var blockUses func(Block) bool
	blockUses = func(block Block) bool {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *AssertStmt:
				return true
			case *VarDecl:
				if exprUses(s.Value) {
					return true
				}
			case *ReturnStmt:
				if exprUses(s.Value) {
					return true
				}
			case *ExprStmt:
				if exprUses(s.Value) {
					return true
				}
			case *TryStmt:
				if blockUses(s.Body) {
					return true
				}
				for _, a := range s.Except {
					if blockUses(a.Body) {
						return true
					}
				}
			case *MatchStmt:
				for _, a := range s.Arms {
					if blockUses(a.Block) {
						return true
					}
				}
			}
		}
		return false
	}
	for _, fn := range module.Functions {
		if fn.Body != nil && blockUses(*fn.Body) {
			return true
		}
	}
	return false
}

func evt1ModuleUsesTransitionPanic(module Module) bool {
	var blockUses func(Block) bool
	blockUses = func(block Block) bool {
		for _, stmt := range block.Statements {
			switch s := stmt.(type) {
			case *TransitionMatchStmt, *TransitionDecideStmt:
				return true
			case *IfStmt:
				if blockUses(s.Then) || (s.Else != nil && blockUses(*s.Else)) {
					return true
				}
			case *MatchStmt:
				for _, arm := range s.Arms {
					if blockUses(arm.Block) {
						return true
					}
				}
			case *TryStmt:
				if blockUses(s.Body) {
					return true
				}
				for _, arm := range s.Except {
					if blockUses(arm.Body) {
						return true
					}
				}
			case *WhileStmt:
				if blockUses(s.Body) {
					return true
				}
			case *Block:
				if blockUses(*s) {
					return true
				}
			}
		}
		return false
	}
	for _, automata := range module.Automata {
		for _, machine := range automata.Machines {
			for _, state := range machine.States {
				if state.Body != nil && blockUses(*state.Body) {
					return true
				}
			}
		}
	}
	return false
}

func validateFailureConstructExpr(env *semanticEnv, scope *evt1Scope, expr *ConstructExpr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	expected = evt1CanonicalType(env, expected)
	if expr.EnumName != expected.Name {
		return Type{}, evt1Diagnostic("CV4540", fmt.Sprintf("constructor %s::%s cannot initialize %s", expr.EnumName, expr.VariantName, expected.String()), expr.Span)
	}
	decl, _ := evt1FailureEnumDecl(expected)
	variant, ok := evt1LookupVariant(decl, expr.VariantName)
	if !ok {
		return Type{}, evt1Diagnostic("CV4103", fmt.Sprintf("unknown variant %s::%s", expr.EnumName, expr.VariantName), expr.Span)
	}
	if len(expr.Args) != len(variant.Payload) {
		return Type{}, evt1Diagnostic("CV4106", fmt.Sprintf("wrong constructor payload count for %s::%s: expected %d but got %d", expr.EnumName, expr.VariantName, len(variant.Payload), len(expr.Args)), expr.Span)
	}
	for i, arg := range expr.Args {
		argType, err := validateExprAgainstExpected(env, scope, arg, variant.Payload[i].Type, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !evt1CanInitializeStoredType(env, variant.Payload[i].Type, argType) {
			return Type{}, evt1Diagnostic("CV4107", fmt.Sprintf("wrong constructor payload type for %s::%s: expected %s but got %s", expr.EnumName, expr.VariantName, variant.Payload[i].Type.String(), argType.String()), arg.exprSpan())
		}
		if !evt1TypeCopyable(env, argType) && !evt1CanTransferInitialize(env, variant.Payload[i].Type, arg) {
			return Type{}, evt1Diagnostic("CV4548", fmt.Sprintf("%s payload %s must be transferred exactly once with move", expected.Name, argType.String()), arg.exprSpan())
		}
	}
	expr.ResolvedType = expected
	return expected, nil
}
