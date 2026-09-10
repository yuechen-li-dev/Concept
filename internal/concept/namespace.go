package concept

import (
	"fmt"
	"sort"
	"strings"
)

type namespaceResolution struct {
	qualified         map[string]string
	ambiguous         map[string][]string
	ambiguousFunction map[string][]string
}

// resolveNamespaceSymbols projects namespace-qualified source names onto the
// existing module-scoped semantic engine. This happens before validation and
// lowering; namespaces therefore create no runtime representation.
func resolveNamespaceSymbols(module *Module) error {
	resolution := namespaceResolution{qualified: map[string]string{}, ambiguous: map[string][]string{}, ambiguousFunction: map[string][]string{}}
	byShort := map[string][]string{}
	for _, symbol := range module.NamespaceSymbols {
		qualified := symbol.Namespace + "." + symbol.Name
		resolution.qualified[qualified] = symbol.Name
		if symbol.Kind == "type" {
			byShort[symbol.Name] = append(byShort[symbol.Name], qualified)
		}
	}
	for key, names := range byShort {
		names = uniqueSortedStrings(names)
		if len(names) < 2 {
			continue
		}
		resolution.ambiguous[key] = names
		for _, qualified := range names {
			resolution.qualified[qualified] = qualified
		}
	}
	byFunctionSignature := map[string][]string{}
	for _, fn := range module.Functions {
		qualified := qualifiedAt(module.NamespaceSymbols, "function", fn.Module, fn.Name, fn.Span)
		if qualified != "" {
			byFunctionSignature[evt1FunctionParamSignature(fn)] = append(byFunctionSignature[evt1FunctionParamSignature(fn)], qualified)
		}
	}
	for signature, names := range byFunctionSignature {
		names = uniqueSortedStrings(names)
		if len(names) < 2 {
			continue
		}
		short := signature[:strings.Index(signature, "(")]
		resolution.ambiguousFunction[signature] = names
		resolution.ambiguous[short] = append(resolution.ambiguous[short], names...)
		for _, qualified := range names {
			resolution.qualified[qualified] = qualified
		}
	}
	for i := range module.Functions {
		fn := &module.Functions[i]
		if names := resolution.ambiguousFunction[evt1FunctionParamSignature(*fn)]; len(names) > 1 {
			if qualified := qualifiedAt(module.NamespaceSymbols, "function", fn.Module, fn.Name, fn.Span); qualified != "" {
				fn.Name = qualified
			}
		}
		resolveNamespaceType(&fn.ReturnType, resolution)
		for i := range fn.Params {
			resolveNamespaceType(&fn.Params[i].Type, resolution)
		}
		if fn.Body != nil {
			if err := resolveNamespaceBlock(fn.Body, resolution); err != nil {
				return err
			}
		}
	}
	for i := range module.Structs {
		decl := &module.Structs[i]
		if names := resolution.ambiguous[decl.Name]; len(names) > 1 {
			if qualified := qualifiedAt(module.NamespaceSymbols, "type", "", decl.Name, decl.Span); qualified != "" {
				decl.Name = qualified
			}
		}
		for i := range decl.Fields {
			resolveNamespaceType(&decl.Fields[i].Type, resolution)
		}
	}
	for i := range module.ComptimeDecls {
		resolveNamespaceType(&module.ComptimeDecls[i].Type, resolution)
		var err error
		module.ComptimeDecls[i].Value, err = resolveNamespaceExpr(module.ComptimeDecls[i].Value, resolution)
		if err != nil {
			return err
		}
	}
	for i := range module.OperationEffects {
		effect := &module.OperationEffects[i]
		if len(resolution.ambiguous[effect.Operation]) > 1 {
			if qualified := qualifiedForModule(module.NamespaceSymbols, "function", effect.Module, effect.Operation); qualified != "" {
				old := effect.Operation
				effect.Operation = qualified
				if effect.Signature != "" {
					effect.Signature = strings.Replace(effect.Signature, old+"(", qualified+"(", 1)
				}
			}
		}
	}
	return nil
}

func qualifiedAt(symbols []NamespaceSymbol, kind, module, name string, span Span) string {
	for _, symbol := range symbols {
		if symbol.Kind == kind && symbol.Name == name && symbol.Span == span && (module == "" || symbol.Module == module) {
			return symbol.Namespace + "." + name
		}
	}
	return ""
}

func qualifiedForModule(symbols []NamespaceSymbol, kind, module, name string) string {
	for _, symbol := range symbols {
		if symbol.Kind == kind && symbol.Module == module && symbol.Name == name {
			return symbol.Namespace + "." + name
		}
	}
	return ""
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func resolveNamespaceType(t *Type, resolution namespaceResolution) {
	if resolved, ok := resolution.qualified[t.Name]; ok {
		t.Name = resolved
	}
	if t.PointerTo != nil {
		resolveNamespaceType(t.PointerTo, resolution)
	}
	if t.ArrayElem != nil {
		resolveNamespaceType(t.ArrayElem, resolution)
	}
	for i := range t.TypeArgs {
		resolveNamespaceType(&t.TypeArgs[i], resolution)
	}
}

func resolveNamespaceBlock(block *Block, resolution namespaceResolution) error {
	for _, statement := range block.Statements {
		var err error
		switch stmt := statement.(type) {
		case *VarDecl:
			resolveNamespaceType(&stmt.Type, resolution)
			stmt.Value, err = resolveNamespaceExpr(stmt.Value, resolution)
		case *AssignStmt:
			stmt.Target, err = resolveNamespaceExpr(stmt.Target, resolution)
			if err == nil {
				stmt.Value, err = resolveNamespaceExpr(stmt.Value, resolution)
			}
		case *ReturnStmt:
			stmt.Value, err = resolveNamespaceExpr(stmt.Value, resolution)
		case *ExprStmt:
			stmt.Value, err = resolveNamespaceExpr(stmt.Value, resolution)
		case *AssertStmt:
			stmt.Condition, err = resolveNamespaceExpr(stmt.Condition, resolution)
		case *IfStmt:
			stmt.Condition, err = resolveNamespaceExpr(stmt.Condition, resolution)
			if err == nil {
				err = resolveNamespaceBlock(&stmt.Then, resolution)
			}
			if err == nil && stmt.Else != nil {
				err = resolveNamespaceBlock(stmt.Else, resolution)
			}
		case *WhileStmt:
			stmt.Condition, err = resolveNamespaceExpr(stmt.Condition, resolution)
			if err == nil {
				err = resolveNamespaceBlock(&stmt.Body, resolution)
			}
		case *Block:
			err = resolveNamespaceBlock(stmt, resolution)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveNamespaceExpr(expr Expr, resolution namespaceResolution) (Expr, error) {
	if expr == nil {
		return nil, nil
	}
	switch value := expr.(type) {
	case *CallExpr:
		if value.Member {
			if prefix, ok := namespaceExprName(value.Receiver); ok {
				qualified := prefix + "." + value.Callee
				if resolved, found := resolution.qualified[qualified]; found {
					value.Callee, value.Member, value.Receiver = resolved, false, nil
				}
			}
		} else if candidates := resolution.ambiguous[value.Callee]; len(candidates) > 1 {
			return nil, evt1Diagnostic("NAMESPACE_SYMBOL_AMBIGUOUS", fmt.Sprintf("ambiguous symbol %s; candidates: %s", value.Callee, strings.Join(candidates, ", ")), value.Span)
		}
		for i := range value.Args {
			var err error
			value.Args[i], err = resolveNamespaceExpr(value.Args[i], resolution)
			if err != nil {
				return nil, err
			}
		}
		return value, nil
	case *FieldExpr:
		resolved, err := resolveNamespaceExpr(value.Receiver, resolution)
		value.Receiver = resolved
		return value, err
	case *BinaryExpr:
		var err error
		value.Left, err = resolveNamespaceExpr(value.Left, resolution)
		if err != nil {
			return nil, err
		}
		value.Right, err = resolveNamespaceExpr(value.Right, resolution)
		return value, err
	case *UnaryExpr:
		resolved, err := resolveNamespaceExpr(value.Value, resolution)
		value.Value = resolved
		return value, err
	case *ParenExpr:
		resolved, err := resolveNamespaceExpr(value.Value, resolution)
		value.Value = resolved
		return value, err
	case *FailureExpr:
		resolved, err := resolveNamespaceExpr(value.Value, resolution)
		value.Value = resolved
		return value, err
	case *StructConstructExpr:
		if resolved, ok := resolution.qualified[value.StructName]; ok {
			value.StructName = resolved
			value.StructType.Name = resolved
		}
		for i := range value.Args {
			var err error
			value.Args[i], err = resolveNamespaceExpr(value.Args[i], resolution)
			if err != nil {
				return nil, err
			}
		}
		return value, nil
	case *ArrayLiteralExpr:
		for i := range value.Elements {
			var err error
			value.Elements[i], err = resolveNamespaceExpr(value.Elements[i], resolution)
			if err != nil {
				return nil, err
			}
		}
		return value, nil
	default:
		return expr, nil
	}
}

func namespaceExprName(expr Expr) (string, bool) {
	switch value := expr.(type) {
	case *NameExpr:
		return value.Name, true
	case *FieldExpr:
		prefix, ok := namespaceExprName(value.Receiver)
		if !ok {
			return "", false
		}
		return prefix + "." + value.Field, true
	default:
		return "", false
	}
}
