package concept

import "fmt"

// A C ABI may carry an aggregate value through an implicit pointer. Materialize
// the source value at private entry so LLVM need not keep rereading that pointer
// across stores through views. This is representation, not a no-alias promise:
// pointees and borrowed backing storage retain all their original aliasing.
// Keep the experiment bounded to small transparent, drop-free value aggregates.
func evt1SnapshotValueLeaves(env *semanticEnv, t Type, visiting map[string]bool) int {
	if t.isBorrowLike() || t.isOwned() || t.Unsafe || t.ArrayElem != nil || t.Name == evt1AtomicIntType {
		return 0
	}
	if evt1IsSpanType(t) {
		return 2 // copy descriptor fields, never the elements or their ownership
	}
	if t.Kind == TypeBuiltin {
		switch evt1CType(t) {
		case "bool", "int32_t", "uint32_t", "uint8_t", "uint16_t", "uint64_t", "size_t", "ptrdiff_t", "float", "double":
			return 1
		}
		return 0
	}
	if e, ok := env.enums[t.Name]; ok {
		for _, v := range e.Variants {
			if len(v.Payload) != 0 {
				return 0
			}
		}
		return 1
	}
	decl, ok := env.structs[t.Name]
	if !ok || visiting[t.Name] || len(visiting) >= 16 || decl.Immovable || decl.Class || decl.Table || decl.BitsRepresentation != "" || len(decl.Methods) != 0 || evt1TypeHasDrop(env, t) {
		return 0
	}
	for _, attr := range decl.Attributes {
		if attr.Name != "repr" {
			return 0
		}
	}
	visiting[t.Name] = true
	defer delete(visiting, t.Name)
	leaves := 0
	for _, field := range decl.Fields {
		n := evt1SnapshotValueLeaves(env, field.Type, visiting)
		if n == 0 || leaves+n > 16 {
			return 0
		}
		leaves += n
	}
	return leaves
}

func (f *evt1FunctionLowerer) valueParameterSnapshots() (map[int]string, string) {
	if !f.private || f.fn.Async {
		return nil, ""
	}
	reserved := map[string]bool{}
	for _, p := range f.fn.Params {
		reserved[evt1CParamName(p.Name)] = true
	}
	// These declarations share the C function's outermost scope. Nested locals
	// may shadow an incoming parameter only after its entry snapshot is bound.
	for _, stmt := range f.fn.Body.Statements {
		if v, ok := stmt.(*VarDecl); ok {
			reserved[evt1CParamName(v.Name)] = true
		}
	}
	inputs := map[int]string{}
	entry := ""
	for i, p := range f.fn.Params {
		if evt1SnapshotValueLeaves(f.l.env, p.Type, map[string]bool{}) < 2 {
			continue
		}
		name := f.nextTemp("parameter_value")
		for reserved[name] {
			name = f.nextTemp("parameter_value")
		}
		reserved[name] = true
		inputs[i] = name
		entry += fmt.Sprintf("  %s %s = %s; /* source value snapshot */\n", evt1CType(p.Type), evt1CParamName(p.Name), name)
	}
	return inputs, entry
}
