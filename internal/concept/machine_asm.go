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
			register := p.next()
			if register.Lexeme != "register" && !asmFixedRegister[register.Lexeme] {
				return nil, evt1Diagnostic("ASM_OPERAND_CLASS", "assembly operand must use register or a supported AMD64 register", register.Span)
			}
			name, err := p.expectIdentifier("ASM_OPERAND_INVALID", "expected operand name")
			if err != nil {
				return nil, err
			}
			op := AsmOperand{Mode: kind.Lexeme, Name: name.Lexeme}
			if register.Lexeme != "register" {
				op.Register = register.Lexeme
			}
			s.Operands = append(s.Operands, op)
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
	if len(s.Operands) == 0 {
		return evt1Diagnostic("ASM_OPERAND_REQUIRED", "assembly requires explicit register operands", s.Span)
	}
	if s.MemoryEffect != "none" && s.MemoryEffect != "read" && s.MemoryEffect != "write" && s.MemoryEffect != "readwrite" {
		return evt1Diagnostic("ASM_MEMORY_EFFECT_REQUIRED", "assembly must declare memory none, read, write, or readwrite", s.Span)
	}
	if strings.ContainsAny(s.Template, "\n\r;") {
		return evt1Diagnostic("ASM_TEMPLATE_INVALID", "assembly requires one local instruction", s.Span)
	}
	remainingTemplate := s.Template
	names := map[string]bool{}
	fixedInputs, fixedOutputs := map[string]bool{}, map[string]bool{}
	autoCount := 0
	for i := range s.Operands {
		op := &s.Operands[i]
		if names[op.Name] {
			return evt1Diagnostic("ASM_OPERAND_DUPLICATE", "duplicate assembly operand "+op.Name, s.Span)
		}
		names[op.Name] = true
		placeholder := "{" + op.Name + "}"
		if op.Register == "" && !strings.Contains(remainingTemplate, placeholder) {
			return evt1Diagnostic("ASM_TEMPLATE_INVALID", "assembly template does not reference "+op.Name, s.Span)
		}
		remainingTemplate = strings.ReplaceAll(remainingTemplate, placeholder, "")
		if op.Register == "" {
			autoCount++
		} else {
			if op.Mode != "out" && fixedInputs[op.Register] || op.Mode != "in" && fixedOutputs[op.Register] || op.Mode == "inout" && (fixedInputs[op.Register] || fixedOutputs[op.Register]) {
				return evt1Diagnostic("ASM_REGISTER_CONFLICT", "incompatible operands require "+op.Register, s.Span)
			}
			if op.Mode != "out" {
				fixedInputs[op.Register] = true
			}
			if op.Mode != "in" {
				fixedOutputs[op.Register] = true
			}
		}
	}
	if strings.ContainsAny(remainingTemplate, "{}") {
		return evt1Diagnostic("ASM_TEMPLATE_INVALID", "unknown assembly template operand", s.Span)
	}
	if autoCount > 4 {
		return evt1Diagnostic("ASM_OPERAND_LIMIT", "at most four backend-selected registers are supported", s.Span)
	}
	mnemonic := strings.ToLower(strings.Fields(s.Template)[0])
	if strings.HasPrefix(mnemonic, "j") || mnemonic == "call" || mnemonic == "ret" || mnemonic == "loop" || strings.Contains(s.Template, ":") {
		return evt1Diagnostic("ASM_CONTROL_FLOW_UNSUPPORTED", "assembly cannot leave the current statement", s.Span)
	}
	seen := map[string]bool{}
	for _, name := range s.Clobbers {
		if name != "flags" && name != "r10" && name != "r11" && !asmFixedRegister[name] {
			return evt1Diagnostic("ASM_CLOBBER_INVALID", "unsupported AMD64 assembly clobber", s.Span)
		}
		if seen[name] {
			return evt1Diagnostic("ASM_CLOBBER_DUPLICATE", "duplicate assembly clobber "+name, s.Span)
		}
		seen[name] = true
	}
	for _, op := range s.Operands {
		if seen[op.Register] {
			return evt1Diagnostic("ASM_REGISTER_CONFLICT", "operand register conflicts with clobber "+op.Register, s.Span)
		}
	}
	if autoCount > 4-boolInt(seen["r10"])-boolInt(seen["r11"]) {
		return evt1Diagnostic("ASM_OPERAND_LIMIT", "clobbers leave too few backend-selected registers", s.Span)
	}
	remaining := s.Template
	for _, register := range asmRawRegister.FindAllString(s.Template, -1) {
		owner := ""
		if strings.HasPrefix(register, "%r10") {
			owner = "r10"
		} else if strings.HasPrefix(register, "%r11") {
			owner = "r11"
		} else {
			for _, fixed := range []string{"eax", "ebx", "ecx", "edx"} {
				if register == asmRegisterSpelling(fixed, "b") || register == asmRegisterSpelling(fixed, "w") || register == asmRegisterSpelling(fixed, "l") || register == asmRegisterSpelling(fixed, "q") {
					owner = fixed
					break
				}
			}
		}
		if owner == "" || !seen[owner] || !asmAllowedRawRegister[register] {
			return evt1Diagnostic("ASM_REGISTER_UNDECLARED", "raw register requires an explicit supported clobber", s.Span)
		}
		remaining = strings.Replace(remaining, register, "", 1)
	}
	if strings.Contains(remaining, "%") {
		return evt1Diagnostic("ASM_REGISTER_UNDECLARED", "unsupported raw register spelling", s.Span)
	}
	for i := range s.Operands {
		op := &s.Operands[i]
		name := &NameExpr{Name: op.Name, Span: s.Span}
		if op.Mode == "out" || op.Mode == "inout" {
			place, err := validateAssignable(env, scope, name, nil)
			if err != nil || !place.mutable {
				return evt1Diagnostic("ASM_OUTPUT_INVALID", "output operand must be a mutable local scalar", s.Span)
			}
			op.Type = place.t
		} else {
			t, err := validateExpr(env, scope, name, nil, false)
			if err != nil {
				return err
			}
			op.Type = t
		}
		if _, ok := asmScalarRegister(op.Type.Name); !ok || op.Type.Kind != TypeBuiltin || op.Type.Quantity != nil {
			return evt1Diagnostic("ASM_OPERAND_TYPE", fmt.Sprintf("unsupported assembly register operand type %s", op.Type.String()), s.Span)
		}
		if op.Register != "" && asmWidth(op.Type.Name) != "l" {
			return evt1Diagnostic("ASM_OPERAND_TYPE", "fixed "+op.Register+" requires a 32-bit scalar", s.Span)
		}
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

var asmFixedRegister = map[string]bool{"eax": true, "ebx": true, "ecx": true, "edx": true}

var asmRawRegister = regexp.MustCompile(`%[A-Za-z][A-Za-z0-9]*`)

var asmAllowedRawRegister = map[string]bool{
	"%r10": true, "%r10d": true, "%r10w": true, "%r10b": true,
	"%r11": true, "%r11d": true, "%r11w": true, "%r11b": true,
	"%al": true, "%ax": true, "%eax": true, "%rax": true,
	"%bl": true, "%bx": true, "%ebx": true, "%rbx": true,
	"%cl": true, "%cx": true, "%ecx": true, "%rcx": true,
	"%dl": true, "%dx": true, "%edx": true, "%rdx": true,
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
	var operands strings.Builder
	for _, operand := range s.Operands {
		fmt.Fprintf(&operands, "%s:%s:%s;", operand.Mode, operand.Register, operand.Name)
	}
	identity := fmt.Sprintf("%s|%s|%s|%s|%v|%s|%d|%d", function, s.Architecture, s.Template, operands.String(), s.Clobbers, s.MemoryEffect, s.Span.Line, s.Span.Column)
	return "concept_asm_" + digest([]byte(identity))[:16]
}

func evt1AsmSupportDeclarations(mir MIR) string {
	var b strings.Builder
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind != "machine_asm" || op.MachineAssembly == nil {
				continue
			}
			fmt.Fprintf(&b, "extern void %s(uint64_t *frame);\n", op.Detail)
		}
	}
	if b.Len() != 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func evt1AsmInstructionSource(op MIROperation) string {
	a := op.MachineAssembly
	var b strings.Builder
	fmt.Fprintf(&b, ".globl %s\n%s:\n", op.Detail, op.Detail)
	b.WriteString("push %r12\n")
	usesEBX := false
	for _, operand := range a.Operands {
		if operand.Register == "ebx" {
			usesEBX = true
		}
	}
	for _, c := range a.Clobbers {
		if c == "ebx" {
			usesEBX = true
		}
	}
	if usesEBX {
		b.WriteString("push %rbx\n")
	}
	b.WriteString("#ifdef _WIN64\nmov %rcx, %r12\n#else\nmov %rdi, %r12\n#endif\n")
	registers := asmAssignedRegisters(a.Operands, a.Clobbers)
	for i, operand := range a.Operands {
		if operand.Mode != "out" {
			fmt.Fprintf(&b, "mov%s %d(%%r12), %s\n", asmWidth(operand.Type), 8*i, asmRegisterSpelling(registers[i], asmWidth(operand.Type)))
		}
	}
	instruction := a.Template
	for i, operand := range a.Operands {
		instruction = strings.ReplaceAll(instruction, "{"+operand.Name+"}", asmRegisterSpelling(registers[i], asmWidth(operand.Type)))
	}
	b.WriteString(instruction + "\n")
	for i, operand := range a.Operands {
		if operand.Mode != "in" {
			fmt.Fprintf(&b, "mov%s %s, %d(%%r12)\n", asmWidth(operand.Type), asmRegisterSpelling(registers[i], asmWidth(operand.Type)), 8*i)
		}
	}
	if usesEBX {
		b.WriteString("pop %rbx\n")
	}
	b.WriteString("pop %r12\n")
	b.WriteString("ret\n")
	return b.String()
}

func asmWidth(name string) string {
	return map[string]string{"uint8": "b", "byte": "b", "uint16": "w", "uint32": "l", "uint": "l", "int": "l", "uint64": "q", "usize": "q", "isize": "q"}[name]
}

func asmAssignedRegisters(operands []MIRAsmOperand, clobbers []string) []string {
	blocked := map[string]bool{}
	for _, c := range clobbers {
		blocked[c] = true
	}
	result := make([]string, len(operands))
	next := 0
	for i, op := range operands {
		if op.Register != "" {
			result[i] = op.Register
			continue
		}
		for blocked[[]string{"r8", "r9", "r10", "r11"}[next]] {
			next++
		}
		result[i] = []string{"r8", "r9", "r10", "r11"}[next]
		next++
	}
	return result
}

func asmRegisterSpelling(register, width string) string {
	ordinary := map[string]map[string]string{
		"eax": {"b": "%al", "w": "%ax", "l": "%eax", "q": "%rax"},
		"ebx": {"b": "%bl", "w": "%bx", "l": "%ebx", "q": "%rbx"},
		"ecx": {"b": "%cl", "w": "%cx", "l": "%ecx", "q": "%rcx"},
		"edx": {"b": "%dl", "w": "%dx", "l": "%edx", "q": "%rdx"},
	}
	if widths, ok := ordinary[register]; ok {
		return widths[width]
	}
	return "%" + register + map[string]string{"b": "b", "w": "w", "l": "d", "q": ""}[width]
}

func (f *evt1FunctionLowerer) lowerAsmStatement(s *AsmStmt, indent int) string {
	var b strings.Builder
	frame := fmt.Sprintf("concept_asm_frame_%d_%d", s.Span.Line, s.Span.Column)
	fmt.Fprintf(&b, "%suint64_t %s[%d] = {0};\n", ind(indent), frame, len(s.Operands))
	for i, operand := range s.Operands {
		if operand.Mode == "out" {
			continue
		}
		prelude, value, _ := f.lowerExprExpected(&NameExpr{Name: operand.Name, Span: s.Span}, operand.Type, indent)
		b.WriteString(prelude)
		fmt.Fprintf(&b, "%s%s[%d] = (uint64_t)(%s);\n", ind(indent), frame, i, value)
	}
	fmt.Fprintf(&b, "%s%s(%s);\n", ind(indent), evt1AsmSymbol(f.fn.Name, s), frame)
	for i, operand := range s.Operands {
		if operand.Mode == "in" {
			continue
		}
		prelude, place, _, _ := f.lowerLValue(&NameExpr{Name: operand.Name, Span: s.Span}, indent)
		b.WriteString(prelude)
		fmt.Fprintf(&b, "%s%s = (%s)%s[%d];\n", ind(indent), place, evt1CType(operand.Type), frame, i)
	}
	return b.String()
}
