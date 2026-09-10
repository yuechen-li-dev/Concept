package concept

import "fmt"

const (
	evt1AtomicIntType     = "AtomicInt"
	evt1MemoryOrderType   = "MemoryOrder"
	evt1AtomicLoad        = "AtomicLoadIntrinsic"
	evt1AtomicStore       = "AtomicStoreIntrinsic"
	evt1AtomicExchange    = "AtomicExchangeIntrinsic"
	evt1AtomicCompareSwap = "AtomicCompareExchangeIntrinsic"
	evt1AtomicFetchAdd    = "AtomicFetchAddIntrinsic"
)

func evt1ValidateAtomicIntrinsic(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, bool, error) {
	kind := map[string]string{
		evt1AtomicLoad:        "atomic_load",
		evt1AtomicStore:       "atomic_store",
		evt1AtomicExchange:    "atomic_exchange",
		evt1AtomicCompareSwap: "atomic_compare_exchange",
		evt1AtomicFetchAdd:    "atomic_fetch_add",
	}[call.Callee]
	if kind == "" {
		return Type{}, false, nil
	}
	if inComptimeFn {
		return Type{}, true, evt1Diagnostic("ATOMIC_COMPTIME_INVALID", "atomic operations require runtime storage", call.Span)
	}
	want := 2
	if kind == "atomic_store" || kind == "atomic_exchange" || kind == "atomic_fetch_add" {
		want = 3
	}
	if kind == "atomic_compare_exchange" {
		want = 5
	}
	if len(call.Args) != want {
		return Type{}, true, evt1Diagnostic("ATOMIC_ARGUMENTS", fmt.Sprintf("%s requires %d arguments", call.Callee, want), call.Span)
	}
	first, err := validateExpr(env, scope, call.Args[0], templateInfo, false)
	if err != nil {
		return Type{}, true, err
	}
	if !first.isReference() || first.borrowBase().Name != evt1AtomicIntType {
		return Type{}, true, evt1Diagnostic("ATOMIC_STORAGE_REQUIRED", fmt.Sprintf("%s requires ref AtomicInt", call.Callee), call.Args[0].exprSpan())
	}
	intType, _ := evt1BuiltinType("int", call.Span)
	boolType, _ := evt1BuiltinType("bool", call.Span)
	voidType, _ := evt1BuiltinType("void", call.Span)
	orderAt := want - 1
	if kind == "atomic_compare_exchange" {
		expected, expectedErr := validateExpr(env, scope, call.Args[1], templateInfo, false)
		if expectedErr != nil {
			return Type{}, true, expectedErr
		}
		if !expected.isReference() || expected.Const || expected.borrowBase().Name != "int" {
			return Type{}, true, evt1Diagnostic("ATOMIC_EXPECTED_REQUIRED", "compare-exchange requires mutable ref int expected", call.Args[1].exprSpan())
		}
		if _, valueErr := validateExprAgainstExpected(env, scope, call.Args[2], intType, templateInfo, false); valueErr != nil {
			return Type{}, true, valueErr
		}
		for _, index := range []int{3, 4} {
			order, orderErr := validateExpr(env, scope, call.Args[index], templateInfo, false)
			if orderErr != nil {
				return Type{}, true, orderErr
			}
			if order.Name != evt1MemoryOrderType {
				return Type{}, true, evt1Diagnostic("ATOMIC_ORDER_REQUIRED", "atomic operation requires MemoryOrder", call.Args[index].exprSpan())
			}
		}
		call.Intrinsic = kind
		return boolType, true, nil
	}
	if want == 3 {
		if _, valueErr := validateExprAgainstExpected(env, scope, call.Args[1], intType, templateInfo, false); valueErr != nil {
			return Type{}, true, valueErr
		}
	}
	order, orderErr := validateExpr(env, scope, call.Args[orderAt], templateInfo, false)
	if orderErr != nil {
		return Type{}, true, orderErr
	}
	if order.Name != evt1MemoryOrderType {
		return Type{}, true, evt1Diagnostic("ATOMIC_ORDER_REQUIRED", "atomic operation requires MemoryOrder", call.Args[orderAt].exprSpan())
	}
	call.Intrinsic = kind
	if kind == "atomic_store" {
		return voidType, true, nil
	}
	return intType, true, nil
}

func evt1ModuleUsesAtomics(module Module) bool {
	for _, decl := range module.Structs {
		if decl.Name == evt1AtomicIntType {
			return true
		}
	}
	return false
}

func (f *evt1FunctionLowerer) lowerAtomicIntrinsic(call *CallExpr, indent int) (string, string, Type) {
	var prelude string
	args := make([]string, 0, len(call.Args))
	for _, arg := range call.Args {
		part, value, _ := f.lowerExpr(arg, indent)
		prelude += part
		args = append(args, value)
	}
	order := func(kind, value string) string { return "concept_atomic_" + kind + "_order((" + value + ").tag)" }
	intType, _ := evt1BuiltinType("int", call.Span)
	boolType, _ := evt1BuiltinType("bool", call.Span)
	voidType, _ := evt1BuiltinType("void", call.Span)
	switch call.Intrinsic {
	case "atomic_load":
		return prelude, "atomic_load_explicit(&((" + args[0] + ")->value), " + order("load", args[1]) + ")", intType
	case "atomic_store":
		return prelude, "atomic_store_explicit(&((" + args[0] + ")->value), " + args[1] + ", " + order("store", args[2]) + ")", voidType
	case "atomic_exchange":
		return prelude, "atomic_exchange_explicit(&((" + args[0] + ")->value), " + args[1] + ", " + order("rmw", args[2]) + ")", intType
	case "atomic_compare_exchange":
		return prelude, "atomic_compare_exchange_strong_explicit(&((" + args[0] + ")->value), " + args[1] + ", " + args[2] + ", " + order("rmw", args[3]) + ", " + order("failure", args[4]) + ")", boolType
	default:
		return prelude, "atomic_fetch_add_explicit(&((" + args[0] + ")->value), " + args[1] + ", " + order("rmw", args[2]) + ")", intType
	}
}
