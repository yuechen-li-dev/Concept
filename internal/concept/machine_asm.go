package concept

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseAsmStatement deliberately accepts a small structured vocabulary. The
// string is an instruction template; operands and effects remain typed data.
func (p *parser) parseAsmStatement() (Statement, error) {
	start := p.next().Span // unsafe
	if _, err := p.expect("asm"); err != nil {
		return nil, evt1Diagnostic("ASM_UNSAFE_REQUIRED", "unsafe must be followed by asm", start)
	}
	arch, err := p.expectIdentifier("ASM_ARCHITECTURE_INVALID", "expected assembly architecture")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect("{"); err != nil {
		return nil, err
	}
	s := &AsmStmt{Architecture: arch.Lexeme, Span: start}
	templateTok := p.next()
	if !strings.HasPrefix(templateTok.Lexeme, "\"") {
		return nil, evt1Diagnostic("ASM_TEMPLATE_REQUIRED", "assembly block requires one instruction string", templateTok.Span)
	}
	s.Template, err = strconv.Unquote(templateTok.Lexeme)
	if err != nil || strings.TrimSpace(s.Template) == "" {
		return nil, evt1Diagnostic("ASM_TEMPLATE_INVALID", "assembly instruction string is invalid", templateTok.Span)
	}
	if _, err := p.expect(";"); err != nil {
		return nil, err
	}
	for p.peekLexeme() != "}" && !p.done() {
		kind := p.next()
		switch kind.Lexeme {
		case "in", "out", "inout":
			if s.OperandMode != "" {
				return nil, evt1Diagnostic("ASM_OPERAND_LIMIT", "EVT1 assembly supports one register operand", kind.Span)
			}
			if _, err := p.expect("register"); err != nil {
				return nil, evt1Diagnostic("ASM_OPERAND_CLASS", "assembly operand must use register", kind.Span)
			}
			name, err := p.expectIdentifier("ASM_OPERAND_INVALID", "expected operand name")
			if err != nil {
				return nil, err
			}
			s.OperandMode, s.OperandName = kind.Lexeme, name.Lexeme
		case "clobber":
			name, err := p.expectIdentifier("ASM_CLOBBER_INVALID", "expected clobber name")
			if err != nil {
				return nil, err
			}
			s.Clobbers = append(s.Clobbers, name.Lexeme)
		case "memory":
			if s.MemoryEffect != "" {
				return nil, evt1Diagnostic("ASM_MEMORY_EFFECT_DUPLICATE", "memory effect is already declared", kind.Span)
			}
			effect, err := p.expectIdentifier("ASM_MEMORY_EFFECT_INVALID", "expected memory effect")
			if err != nil {
				return nil, err
			}
			s.MemoryEffect = effect.Lexeme
		default:
			return nil, evt1Diagnostic("ASM_DECLARATION_INVALID", "unknown assembly declaration "+kind.Lexeme, kind.Span)
		}
		if _, err := p.expect(";"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect("}"); err != nil {
		return nil, err
	}
	return s, nil
}

func evt1ValidateAsmStmt(env *semanticEnv, scope *evt1Scope, s *AsmStmt, inComptimeFn bool) error {
	if inComptimeFn {
		return evt1Diagnostic("ASM_COMPTIME_INVALID", "assembly requires runtime execution", s.Span)
	}
	if s.Architecture != "AMD64" {
		return evt1Diagnostic("ASM_ARCHITECTURE_INVALID", "bounded assembly supports AMD64 only", s.Span)
	}
	if s.OperandMode == "" {
		return evt1Diagnostic("ASM_OPERAND_REQUIRED", "assembly requires one explicit register operand", s.Span)
	}
	if s.MemoryEffect != "none" && s.MemoryEffect != "read" && s.MemoryEffect != "write" && s.MemoryEffect != "readwrite" {
		return evt1Diagnostic("ASM_MEMORY_EFFECT_REQUIRED", "assembly must declare memory none, read, write, or readwrite", s.Span)
	}
	if strings.Count(s.Template, "{"+s.OperandName+"}") != 1 || strings.ContainsAny(s.Template, "\n\r;") || strings.Contains(s.Template, "{") && strings.Count(s.Template, "{") != 1 {
		return evt1Diagnostic("ASM_TEMPLATE_INVALID", "one local instruction must reference its operand exactly once", s.Span)
	}
	mnemonic := strings.ToLower(strings.Fields(s.Template)[0])
	if strings.HasPrefix(mnemonic, "j") || mnemonic == "call" || mnemonic == "ret" || mnemonic == "loop" || strings.Contains(s.Template, ":") {
		return evt1Diagnostic("ASM_CONTROL_FLOW_UNSUPPORTED", "assembly cannot leave the current statement", s.Span)
	}
	seen := map[string]bool{}
	for _, name := range s.Clobbers {
		if name != "flags" && name != "r10" && name != "r11" {
			return evt1Diagnostic("ASM_CLOBBER_INVALID", "bounded AMD64 assembly supports flags, r10, and r11 clobbers", s.Span)
		}
		if seen[name] {
			return evt1Diagnostic("ASM_CLOBBER_DUPLICATE", "duplicate assembly clobber "+name, s.Span)
		}
		seen[name] = true
	}
	remaining := s.Template
	for _, register := range asmRawRegister.FindAllString(s.Template, -1) {
		owner := ""
		if strings.HasPrefix(register, "%r10") {
			owner = "r10"
		} else if strings.HasPrefix(register, "%r11") {
			owner = "r11"
		}
		if owner == "" || !seen[owner] || !asmAllowedRawRegister[register] {
			return evt1Diagnostic("ASM_REGISTER_UNDECLARED", "raw register requires an explicit supported clobber", s.Span)
		}
		remaining = strings.Replace(remaining, register, "", 1)
	}
	if strings.Contains(remaining, "%") {
		return evt1Diagnostic("ASM_REGISTER_UNDECLARED", "unsupported raw register spelling", s.Span)
	}
	name := &NameExpr{Name: s.OperandName, Span: s.Span}
	if s.OperandMode == "out" || s.OperandMode == "inout" {
		place, err := validateAssignable(env, scope, name, nil)
		if err != nil || !place.mutable {
			return evt1Diagnostic("ASM_OUTPUT_INVALID", "output operand must be a mutable local scalar", s.Span)
		}
		s.OperandType = place.t
	} else {
		t, err := validateExpr(env, scope, name, nil, false)
		if err != nil {
			return err
		}
		s.OperandType = t
	}
	if _, ok := asmScalarRegister(s.OperandType.Name); !ok || s.OperandType.Kind != TypeBuiltin || s.OperandType.Quantity != nil {
		return evt1Diagnostic("ASM_OPERAND_TYPE", fmt.Sprintf("unsupported assembly register operand type %s", s.OperandType.String()), s.Span)
	}
	return nil
}

var asmRawRegister = regexp.MustCompile(`%[A-Za-z][A-Za-z0-9]*`)

var asmAllowedRawRegister = map[string]bool{
	"%r10": true, "%r10d": true, "%r10w": true, "%r10b": true,
	"%r11": true, "%r11d": true, "%r11w": true, "%r11b": true,
}

func asmScalarRegister(name string) (string, bool) {
	switch name {
	case "uint8", "byte":
		return "%al", true
	case "uint16":
		return "%ax", true
	case "uint32", "uint", "int":
		return "%eax", true
	case "uint64", "usize", "isize":
		return "%rax", true
	default:
		return "", false
	}
}

func evt1AsmSymbol(function string, s *AsmStmt) string {
	identity := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d", function, s.Architecture, s.Template, s.OperandMode, s.OperandName, s.Span.Line, s.Span.Column)
	return "concept_asm_" + digest([]byte(identity))[:16]
}

func evt1AsmSupportDeclarations(mir MIR) string {
	var b strings.Builder
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind != "machine_asm" || op.MachineAssembly == nil {
				continue
			}
			t, _ := evt1BuiltinType(op.MachineAssembly.OperandType, op.SourceSpan)
			qualifier := ""
			if op.MachineAssembly.OperandMode == "in" {
				qualifier = "const "
			}
			fmt.Fprintf(&b, "extern void %s(%s%s *operand);\n", op.Detail, qualifier, evt1CType(t))
		}
	}
	if b.Len() != 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func evt1AsmInstructionSource(op MIROperation) string {
	a := op.MachineAssembly
	reg, _ := asmScalarRegister(a.OperandType)
	width := map[string]string{"uint8": "b", "byte": "b", "uint16": "w", "uint32": "l", "uint": "l", "int": "l", "uint64": "q", "usize": "q", "isize": "q"}[a.OperandType]
	var b strings.Builder
	fmt.Fprintf(&b, ".globl %s\n%s:\n", op.Detail, op.Detail)
	if a.OperandMode != "out" {
		fmt.Fprintf(&b, "#ifdef _WIN64\nmov%s (%%rcx), %s\n#else\nmov%s (%%rdi), %s\n#endif\n", width, reg, width, reg)
	}
	b.WriteString(strings.ReplaceAll(a.Template, "{"+a.OperandName+"}", reg) + "\n")
	if a.OperandMode != "in" {
		fmt.Fprintf(&b, "#ifdef _WIN64\nmov%s %s, (%%rcx)\n#else\nmov%s %s, (%%rdi)\n#endif\n", width, reg, width, reg)
	}
	b.WriteString("ret\n")
	return b.String()
}
