package concept

import (
	"fmt"
	"runtime"
	"strings"
)

// Machine identities belong to Concept. The assembly below is only the
// current C11 backend's implementation of those identities.
type machineIntrinsicSpec struct {
	id, symbol, result string
	params             []string
	assembly           string
}

var machineIntrinsics = map[string]machineIntrinsicSpec{
	"AMD64.Pause":         {"AMD64.Pause", "ConceptAMD64Pause", "void", nil, "pause\nret"},
	"AMD64.ReadTimestamp": {"AMD64.ReadTimestamp", "ConceptAMD64ReadTimestamp", "uint64", nil, "rdtsc\nshl $32, %rdx\nor %rdx, %rax\nret"},
	"AMD64.In8":           {"AMD64.In8", "ConceptAMD64In8", "uint8", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninb %dx, %al\nret"},
	"AMD64.In16":          {"AMD64.In16", "ConceptAMD64In16", "uint16", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninw %dx, %ax\nret"},
	"AMD64.In32":          {"AMD64.In32", "ConceptAMD64In32", "uint32", []string{"uint16"}, "#ifdef _WIN64\nmov %cx, %dx\n#else\nmov %di, %dx\n#endif\ninl %dx, %eax\nret"},
	"AMD64.Out8":          {"AMD64.Out8", "ConceptAMD64Out8", "void", []string{"uint16", "uint8"}, "#ifdef _WIN64\nmov %dl, %al\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %sil, %al\n#endif\noutb %al, %dx\nret"},
	"AMD64.Out16":         {"AMD64.Out16", "ConceptAMD64Out16", "void", []string{"uint16", "uint16"}, "#ifdef _WIN64\nmov %dx, %ax\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %si, %ax\n#endif\noutw %ax, %dx\nret"},
	"AMD64.Out32":         {"AMD64.Out32", "ConceptAMD64Out32", "void", []string{"uint16", "uint32"}, "#ifdef _WIN64\nmov %edx, %eax\nmov %cx, %dx\n#else\nmov %di, %dx\nmov %esi, %eax\n#endif\noutl %eax, %dx\nret"},
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
		if fn.Module != "Standard.Machine.AMD64" || fn.ExternABI != "C" || fn.Body != nil || fn.Name != spec.symbol || fn.ReturnType.Name != spec.result || len(fn.Params) != len(spec.params) {
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

func evt1MachineHelperSource(mir MIR, target TargetCapabilities) ([]byte, error) {
	used := map[string]bool{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind != "machine_intrinsic" {
				continue
			}
			if target.Architecture == ArchitectureAArch64 || (target.Architecture == ArchitectureGenericC11 && runtime.GOARCH != "amd64") {
				return nil, evt1Diagnostic("MACHINE_ARCHITECTURE_MISMATCH", op.Detail+" requires AMD64, but target/host architecture is incompatible", op.SourceSpan)
			}
			used[op.Detail] = true
		}
	}
	if len(used) == 0 {
		return nil, nil
	}
	var b strings.Builder
	b.WriteString("/* Generated machine helper. Concept intrinsic identities define semantics. */\n.text\n")
	for _, id := range []string{"AMD64.Pause", "AMD64.ReadTimestamp", "AMD64.In8", "AMD64.In16", "AMD64.In32", "AMD64.Out8", "AMD64.Out16", "AMD64.Out32"} {
		if !used[id] {
			continue
		}
		spec := machineIntrinsics[id]
		fmt.Fprintf(&b, ".globl %s\n%s:\n%s\n", spec.symbol, spec.symbol, spec.assembly)
	}
	return []byte(b.String()), nil
}
