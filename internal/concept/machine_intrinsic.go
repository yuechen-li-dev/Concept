package concept

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
)

// Machine identities belong to Concept. The assembly below is only the
// current C11 backend's implementation of those identities.
type machineIntrinsicSpec struct {
	id, symbol, result string
	params             []string
	assembly           string
	privileged         bool
	ordering           string
}

var machineIntrinsics = map[string]machineIntrinsicSpec{
	"AMD64.Pause":         {"AMD64.Pause", "ConceptAMD64Pause", "void", nil, "pause\nret", false, ""},
	"AMD64.ReadTimestamp": {"AMD64.ReadTimestamp", "ConceptAMD64ReadTimestamp", "uint64", nil, "rdtsc\nshl $32, %rdx\nor %rdx, %rax\nret", false, ""},
	"AMD64.In8":           {"AMD64.In8", "ConceptAMD64In8", "uint8", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninb %dx, %al\nret", false, ""},
	"AMD64.In16":          {"AMD64.In16", "ConceptAMD64In16", "uint16", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninw %dx, %ax\nret", false, ""},
	"AMD64.In32":          {"AMD64.In32", "ConceptAMD64In32", "uint32", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninl %dx, %eax\nret", false, ""},
	"AMD64.Out8":          {"AMD64.Out8", "ConceptAMD64Out8", "void", []string{"uint16", "uint8"}, "#ifdef _WIN64\nmov %dl, %al\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %sil, %al\n#endif\noutb %al, %dx\nret", false, ""},
	"AMD64.Out16":         {"AMD64.Out16", "ConceptAMD64Out16", "void", []string{"uint16", "uint16"}, "#ifdef _WIN64\nmov %dx, %ax\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %si, %ax\n#endif\noutw %ax, %dx\nret", false, ""},
	"AMD64.Out32":         {"AMD64.Out32", "ConceptAMD64Out32", "void", []string{"uint16", "uint32"}, "#ifdef _WIN64\nmov %edx, %eax\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %esi, %eax\n#endif\noutl %eax, %dx\nret", false, ""},
	"AMD64.Cli":           {id: "AMD64.Cli", symbol: "ConceptAMD64Cli", result: "void", assembly: "cli\nret", privileged: true},
	"AMD64.Sti":           {id: "AMD64.Sti", symbol: "ConceptAMD64Sti", result: "void", assembly: "sti\nret", privileged: true},
	"AMD64.Hlt":           {id: "AMD64.Hlt", symbol: "ConceptAMD64Hlt", result: "void", assembly: "hlt\nret", privileged: true},
	"AMD64.Lfence":        {id: "AMD64.Lfence", symbol: "ConceptAMD64Lfence", result: "void", assembly: "lfence\nret", ordering: "load"},
	"AMD64.Sfence":        {id: "AMD64.Sfence", symbol: "ConceptAMD64Sfence", result: "void", assembly: "sfence\nret", ordering: "store"},
	"AMD64.Mfence":        {id: "AMD64.Mfence", symbol: "ConceptAMD64Mfence", result: "void", assembly: "mfence\nret", ordering: "full"},
	"AArch64.Yield":       {id: "AArch64.Yield", symbol: "ConceptAArch64Yield", result: "void", assembly: "yield\nret"},
	"AArch64.Wfi":         {id: "AArch64.Wfi", symbol: "ConceptAArch64Wfi", result: "void", assembly: "wfi\nret", privileged: true},
	"AArch64.Dmb":         {id: "AArch64.Dmb", symbol: "ConceptAArch64Dmb", result: "void", assembly: "dmb ish\nret", ordering: "inner-shareable-memory"},
	"AArch64.Dsb":         {id: "AArch64.Dsb", symbol: "ConceptAArch64Dsb", result: "void", assembly: "dsb ish\nret", ordering: "inner-shareable-completion"},
	"AArch64.Isb":         {id: "AArch64.Isb", symbol: "ConceptAArch64Isb", result: "void", assembly: "isb\nret", ordering: "instruction-stream"},
}

func evt1MachineIntrinsic(fn FunctionDecl) (machineIntrinsicSpec, bool, error) {
	for _, attribute := range fn.Attributes {
		if attribute.Name != "machine" {
			continue
		}
		if len(attribute.Args) != 1 {
			return machineIntrinsicSpec{}, true, evt1Diagnostic("MACHINE_INTRINSIC_INVALID", "machine requires one intrinsic identity", attribute.Span)
		}
		literal, ok := attribute.Args[0].(*StringLiteral)
		if !ok {
			return machineIntrinsicSpec{}, true, evt1Diagnostic("MACHINE_INTRINSIC_INVALID", "machine identity must be a string literal", attribute.Span)
		}
		spec, ok := machineIntrinsics[literal.Value]
		if !ok {
			return machineIntrinsicSpec{}, true, evt1Diagnostic("MACHINE_INTRINSIC_UNKNOWN", "unknown machine intrinsic "+literal.Value, attribute.Span)
		}
		if fn.Module != "Standard.Machine."+strings.Split(spec.id, ".")[0] || fn.ExternABI != "C" || fn.Body != nil || fn.Name != spec.symbol || fn.ReturnType.Name != spec.result || len(fn.Params) != len(spec.params) {
			return machineIntrinsicSpec{}, true, evt1Diagnostic("MACHINE_INTRINSIC_SIGNATURE", "machine intrinsic declaration does not match "+spec.id, fn.Span)
		}
		for i, param := range fn.Params {
			if param.Type.Name != spec.params[i] || param.Type.Kind != TypeBuiltin {
				return machineIntrinsicSpec{}, true, evt1Diagnostic("MACHINE_INTRINSIC_SIGNATURE", "machine intrinsic parameter does not match "+spec.id, param.Span)
			}
		}
		return spec, true, nil
	}
	return machineIntrinsicSpec{}, false, nil
}

// Privilege is an ordinary semantic property of a machine operation, not a
// synonym for memory unsafety. Imported function bodies retain this analysis.
func evt1FunctionPrivileged(env *semanticEnv, fn FunctionDecl, visiting map[string]bool) (bool, bool) {
	if spec, machine, err := evt1MachineIntrinsic(fn); machine && err == nil {
		return spec.privileged, true
	}
	key := evt1OperationEffectKey(fn.Name, evt1FunctionParamSignature(fn))
	if fn.Body == nil || visiting[key] {
		return false, false
	}
	visiting[key] = true
	defer delete(visiting, key)
	operations := MIRFunction{Name: fn.Name}
	collectMIROps(env, fn.Body, &operations, nil)
	known := true
	privileged := false
	for _, op := range operations.Operations {
		if op.Kind == "machine_intrinsic" {
			privileged = privileged || op.Privileged
		}
		if op.Kind == "call" {
			candidates := env.functions[op.Detail]
			if len(candidates) != 1 {
				known = false
				continue
			}
			child, childKnown := evt1FunctionPrivileged(env, candidates[0], visiting)
			privileged, known = privileged || child, known && childKnown
		}
	}
	return privileged, known
}

func evt1MachineHelperSource(mir MIR, target TargetCapabilities) ([]byte, error) {
	used := map[string]bool{}
	var assembly []MIROperation
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind != "machine_intrinsic" && op.Kind != "machine_asm" {
				continue
			}
			required := "AMD64"
			if op.Kind == "machine_asm" {
				required = op.MachineAssembly.Architecture
			} else {
				required = strings.Split(op.Detail, ".")[0]
			}
			actual := "AMD64"
			if target.Architecture == ArchitectureAArch64 {
				actual = "AArch64"
			} else if target.Architecture == ArchitectureGenericC11 && runtime.GOARCH != "amd64" {
				actual = "AArch64"
			}
			if required != actual {
				operation := op.Detail
				if op.Kind == "machine_asm" {
					operation = "asm " + required
				}
				return nil, evt1Diagnostic("MACHINE_ARCHITECTURE_MISMATCH", operation+" requires "+required+", but target/host architecture is "+actual, op.SourceSpan)
			}
			if op.Kind == "machine_asm" {
				assembly = append(assembly, op)
			} else {
				used[op.Detail] = true
			}
		}
	}
	if len(used) == 0 && len(assembly) == 0 {
		return nil, nil
	}
	var b strings.Builder
	b.WriteString("/* Generated machine helper. Concept intrinsic identities define semantics. */\n.text\n")
	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !used[id] {
			continue
		}
		spec := machineIntrinsics[id]
		fmt.Fprintf(&b, ".globl %s\n%s:\n%s\n", spec.symbol, spec.symbol, spec.assembly)
	}
	for _, op := range assembly {
		b.WriteString(evt1AsmInstructionSource(op))
	}
	return []byte(b.String()), nil
}
