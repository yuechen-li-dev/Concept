package concept

import (
	"fmt"
	"sort"
	"strings"
)

// This thaw admits explicit C entry points, not a C++ object ABI. Owners stay
// where they were constructed; a C caller receives only a temporary reference.
const evt1CExportAttribute = "__concept_c_export"

func evt1IsCExport(fn FunctionDecl) bool {
	return evt1HasNamedAttribute(fn.Attributes, evt1CExportAttribute)
}

func evt1ValidateCExport(env *semanticEnv, fn FunctionDecl) error {
	if !evt1IsCExport(fn) {
		return nil
	}
	if fn.ExternABI != "" || fn.Body == nil || fn.Async || fn.Comptime || fn.MethodOf != "" {
		return evt1Diagnostic("EXPORT_SIGNATURE_INVALID", "C exports require an ordinary synchronous function body", fn.Span)
	}
	if len(env.functions[fn.Name]) != 1 || !evt1CExportIdentifier(fn.Name) {
		return evt1Diagnostic("EXPORT_SYMBOL_INVALID", "C export names must be unambiguous C identifiers", fn.Span)
	}
	valid := func(t Type, parameter bool) bool {
		t = evt1CanonicalType(env, t)
		if t.Scoped {
			return false
		}
		if t.isReference() && parameter && len(t.TypeArgs) == 0 && t.ArrayElem == nil && t.PointerTo == nil {
			decl, ok := env.structs[t.Name]
			return ok && !decl.Ref && !decl.Table && t.Application == nil
		}
		if t.Ownership != "" || t.Scoped || t.PointerTo != nil || t.ArrayElem != nil || len(t.TypeArgs) != 0 || t.Quantity != nil {
			return false
		}
		if evt1IsHandle(env, t) {
			return true
		}
		// Keep this initial public header subset freestanding and identical in
		// C11/C++23. Aggregates/spans and profile-specific scalars are deferred.
		switch t.Name {
		case "void":
			return !parameter
		case "bool", "byte", "int", "uint", "uint8", "uint16", "uint32", "uint64", "usize", "isize", "float", "double":
			return true
		}
		return false
	}
	if !valid(fn.ReturnType, false) {
		return evt1Diagnostic("EXPORT_C_ABI_TYPE_INVALID", "C export returns must be scalar or opaque foreign handles; ownership and borrowed returns stay inside Concept", fn.ReturnType.Span)
	}
	for _, param := range fn.Params {
		if !valid(param.Type, true) {
			return evt1Diagnostic("EXPORT_C_ABI_TYPE_INVALID", "C export parameters must be scalars, foreign handles, or explicit references to concrete non-view objects", param.Span)
		}
	}
	return nil
}

// Only exported signatures enter this header. In particular, an owner's fields,
// Option/Result tags, helpers and Drop implementation are never ABI declarations.
func (l *lowering) cExportHeader() []byte {
	var exports []FunctionDecl
	opaque := map[string]bool{}
	handles := map[string]bool{}
	for _, fn := range l.module.Functions {
		if !evt1IsCExport(fn) {
			continue
		}
		exports = append(exports, fn)
		types := []Type{fn.ReturnType}
		for _, param := range fn.Params {
			types = append(types, param.Type)
		}
		for _, t := range types {
			t = evt1CanonicalType(l.env, t)
			if t.isReference() {
				opaque[t.Name] = true
			} else if evt1IsHandle(l.env, t) {
				handles[t.Name] = true
			}
		}
	}
	if len(exports) == 0 {
		return nil
	}
	var b strings.Builder
	guard := strings.ToUpper("CONCEPT_" + l.outputBase + "_EXPORTS_H")
	fmt.Fprintf(&b, "/* Generated C interface. Opaque pointers borrow live Concept objects; never copy, free, retain or pass null. */\n#ifndef %s\n#define %s\n#include <stdbool.h>\n#include <stddef.h>\n#include <stdint.h>\n", guard, guard)
	var names []string
	for name := range opaque {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "typedef struct %s %s;\n", evt1CName(name), evt1CName(name))
	}
	names = names[:0]
	for name := range handles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(evt1HandleCDeclaration(l.env.handles[name]))
	}
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n")
	for _, fn := range exports {
		fmt.Fprintf(&b, "%s %s(", evt1CType(fn.ReturnType), fn.Name)
		if len(fn.Params) == 0 {
			b.WriteString("void")
		}
		for i, param := range fn.Params {
			if i != 0 {
				b.WriteString(", ")
			}
			// Parameter names are documentation in C; omitting them also keeps
			// C++ keywords in Concept locals out of the public C++ header.
			b.WriteString(evt1CType(param.Type))
		}
		b.WriteString(");\n")
	}
	b.WriteString("#ifdef __cplusplus\n}\n#endif\n#endif\n")
	return []byte(b.String())
}

func evt1CExportIdentifier(name string) bool {
	if name == "" || evt1CKeyword(name) {
		return false
	}
	for i, c := range []byte(name) {
		if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	const cppKeywords = " alignas alignof and and_eq asm bitand bitor bool catch char8_t char16_t char32_t class compl concept consteval constexpr constinit const_cast co_await co_return co_yield decltype delete dynamic_cast explicit export false friend inline mutable namespace new noexcept not not_eq nullptr operator or or_eq private protected public reinterpret_cast requires static_assert static_cast template this thread_local throw true try typeid typename using virtual wchar_t xor xor_eq "
	return !strings.Contains(cppKeywords, " "+name+" ")
}
