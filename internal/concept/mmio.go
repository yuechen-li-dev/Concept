package concept

import (
	"fmt"
	"strings"
)

// MMIO is an operation on a device address, not a qualifier on an ordinary
// object. The only C volatile types are local to the generated access.
func evt1ValidateMmioCall(env *semanticEnv, scope *evt1Scope, call *TemplateCallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, bool, error) {
	if call.Callee != "MmioLoad" && call.Callee != "MmioStore" {
		return Type{}, false, nil
	}
	if inComptimeFn {
		return Type{}, true, evt1Diagnostic("MMIO_COMPTIME_INVALID", "MMIO requires a runtime device transaction", call.Span)
	}
	width := map[string]int{"uint8": 1, "uint16": 2, "uint32": 4, "uint": 4, "uint64": 8}[call.TypeArg.Name]
	if len(call.TypeArgs) > 1 || width == 0 || call.TypeArg.Kind != TypeBuiltin {
		return Type{}, true, evt1Diagnostic("MMIO_WIDTH_INVALID", "MMIO supports only uint8, uint16, uint32, and uint64 scalar accesses", call.Span)
	}
	want := 1
	if call.Callee == "MmioStore" {
		want = 2
	}
	if len(call.Args) != want {
		return Type{}, true, evt1Diagnostic("MMIO_ARGUMENTS", fmt.Sprintf("%s requires %d argument(s)", call.Callee, want), call.Span)
	}
	address, err := validateExpr(env, scope, call.Args[0], templateInfo, false)
	if err != nil {
		return Type{}, true, err
	}
	if address.Kind != TypeAddress || len(address.TypeArgs) != 1 || address.TypeArgs[0].Name != "DeviceMemory" {
		return Type{}, true, evt1Diagnostic("MMIO_DEVICE_ADDRESS_REQUIRED", "MMIO requires Address<DeviceMemory>", call.Args[0].exprSpan())
	}
	if fromBits, ok := call.Args[0].(*TemplateCallExpr); ok && fromBits.Callee == "AddressFromBits" && len(fromBits.Args) == 1 {
		bits, known := evt1StaticInt(env, scope, fromBits.Args[0])
		if name, isName := fromBits.Args[0].(*NameExpr); isName && !known {
			if binding, found := scope.lookup(name.Name); found && !binding.mutable {
				if literal, isLiteral := binding.source.(*IntLiteral); isLiteral && !literal.Negative && literal.Magnitude <= uint64(^uint(0)>>1) {
					bits, known = int(literal.Magnitude), true
				}
			}
		}
		if known && bits%width != 0 {
			return Type{}, true, evt1Diagnostic("MMIO_ALIGNMENT_INVALID", fmt.Sprintf("constant device address is not aligned for %d-byte access", width), call.Args[0].exprSpan())
		}
	}
	if call.Callee == "MmioStore" {
		if _, err := validateExprAgainstExpected(env, scope, call.Args[1], call.TypeArg, templateInfo, false); err != nil {
			return Type{}, true, err
		}
		voidType, _ := evt1BuiltinType("void", call.Span)
		return voidType, true, nil
	}
	return call.TypeArg, true, nil
}

func (f *evt1FunctionLowerer) lowerMmioCall(call *TemplateCallExpr, indent int) (string, string, Type) {
	prelude, address, _ := f.lowerExpr(call.Args[0], indent)
	addrTemp := f.nextTemp("mmio_address")
	prelude += ind(indent) + fmt.Sprintf("uintptr_t %s = (uintptr_t)(%s);\n", addrTemp, address)
	suffix := strings.ToUpper(call.TypeArg.Name)
	if call.Callee == "MmioLoad" {
		valueTemp := f.nextTemp("mmio_value")
		prelude += ind(indent) + fmt.Sprintf("%s %s = CONCEPT_MMIO_READ_%s(%s);\n", evt1CType(call.TypeArg), valueTemp, suffix, addrTemp)
		return prelude, valueTemp, call.TypeArg
	}
	valuePrelude, value, _ := f.lowerExprExpected(call.Args[1], call.TypeArg, indent)
	prelude += valuePrelude
	valueTemp := f.nextTemp("mmio_value")
	prelude += ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(call.TypeArg), valueTemp, value)
	prelude += ind(indent) + fmt.Sprintf("CONCEPT_MMIO_WRITE_%s(%s, %s);\n", suffix, addrTemp, valueTemp)
	voidType, _ := evt1BuiltinType("void", call.Span)
	return prelude, "(void)0", voidType
}

func evt1MmioSupportDeclarations(mir MIR) string {
	used := map[string]bool{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "mmio_read" || op.Kind == "mmio_write" {
				used[op.Type] = true
			}
		}
	}
	var b strings.Builder
	for _, width := range []string{"uint8", "uint16", "uint32", "uint", "uint64"} {
		if !used[width] {
			continue
		}
		suffix := strings.ToUpper(width)
		cType, _ := evt1BuiltinType(width, Span{})
		ctype := evt1CType(cType)
		fmt.Fprintf(&b, "#ifndef CONCEPT_MMIO_READ_%s\n#define CONCEPT_MMIO_READ_%s(a) (*((volatile %s *)(uintptr_t)(a)))\n#endif\n", suffix, suffix, ctype)
		fmt.Fprintf(&b, "#ifndef CONCEPT_MMIO_WRITE_%s\n#define CONCEPT_MMIO_WRITE_%s(a,v) (*((volatile %s *)(uintptr_t)(a)) = (v))\n#endif\n", suffix, suffix, ctype)
	}
	if b.Len() != 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func evt1FunctionHardwareEffects(env *semanticEnv, fn FunctionDecl, visiting map[string]bool) (read, write, known bool) {
	key := evt1OperationEffectKey(fn.Name, evt1FunctionParamSignature(fn))
	if summary, imported := env.importedHardwareEffects[key]; imported {
		return summary.Read, summary.Write, summary.Known
	}
	if fn.Body == nil || visiting[key] {
		return false, false, false
	}
	visiting[key] = true
	defer delete(visiting, key)
	operations := MIRFunction{Name: fn.Name}
	collectMIROps(env, fn.Body, &operations, nil)
	known = true
	for _, op := range operations.Operations {
		switch op.Kind {
		case "machine_intrinsic":
			switch op.Detail {
			case "AMD64.ReadTimestamp", "AMD64.In8", "AMD64.In16", "AMD64.In32":
				read = true
			case "AMD64.Out8", "AMD64.Out16", "AMD64.Out32":
				write = true
			}
		case "mmio_read":
			read = true
		case "mmio_write":
			write = true
		case "call":
			candidates := env.functions[op.Detail]
			if len(candidates) != 1 {
				known = false
				continue
			}
			childRead, childWrite, childKnown := evt1FunctionHardwareEffects(env, candidates[0], visiting)
			read, write, known = read || childRead, write || childWrite, known && childKnown
		}
	}
	return read, write, known
}

func summarizeModuleHardwareEffects(module Module, env *semanticEnv) []SemanticModuleHardwareEffectSummary {
	var summaries []SemanticModuleHardwareEffectSummary
	for _, fn := range module.Functions {
		read, write, known := evt1FunctionHardwareEffects(env, fn, map[string]bool{})
		summaries = append(summaries, SemanticModuleHardwareEffectSummary{Operation: fn.Name, Signature: evt1FunctionParamSignature(fn), Read: read, Write: write, Known: known})
	}
	return summaries
}

func evt1QualifyHardwareFacts(mir *MIR, env *semanticEnv) {
	for _, fn := range mir.Functions {
		var declaration *FunctionDecl
		for _, candidate := range env.functions[fn.Name] {
			if evt1MIRType(env, candidate.ReturnType).SameValueType(fn.ReturnType) {
				copy := candidate
				declaration = &copy
				break
			}
		}
		if declaration == nil {
			continue
		}
		read, write, _ := evt1FunctionHardwareEffects(env, *declaration, map[string]bool{})
		subject := SemanticFactSubject{Kind: "operation", Name: fn.Name, Function: fn.Name, Type: fn.ReturnType.String()}
		if read {
			evt1AppendFact(&mir.SemanticFacts, FactHardwareRead, []SemanticFactSubject{subject}, nil, FactOriginCompilerAnalysis, SemanticFactEvidence{Detail: "hardware read in transitive call graph"}, fn.SourceSpan)
		}
		if write {
			evt1AppendFact(&mir.SemanticFacts, FactHardwareWrite, []SemanticFactSubject{subject}, nil, FactOriginCompilerAnalysis, SemanticFactEvidence{Detail: "hardware write in transitive call graph"}, fn.SourceSpan)
		}
	}
}
