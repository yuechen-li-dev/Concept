package concept

import "fmt"

// HandleDecl is `extern "C" handle Name;`: an opaque, pointer-sized foreign
// handle such as a Vulkan object. Handles are copied, passed across
// `extern "C"`, stored in fields, and compared with == and !=; nothing else.
// Absence is spelled Option<Name>, never a sentinel value.
type HandleDecl struct {
	Name   string `json:"name"`
	Module string `json:"-"`
	Span   Span   `json:"span"`
}

func evt1RegisterHandles(env *semanticEnv, module Module, typeNames map[string]Span) error {
	for _, decl := range module.Handles {
		if env.profile.compilerOwnedType(decl.Name) {
			return evt1Diagnostic("CV4267", fmt.Sprintf("%s is a compiler-owned runtime type and cannot be redeclared", decl.Name), decl.Span)
		}
		if _, ok := env.profile.builtinType(decl.Name, decl.Span); ok {
			return evt1Diagnostic("HANDLE_DECL_INVALID", fmt.Sprintf("handle %s would shadow the builtin type %s", decl.Name, decl.Name), decl.Span)
		}
		if _, exists := typeNames[decl.Name]; exists {
			return evt1Diagnostic("CV4580", fmt.Sprintf("duplicate type declaration %s", decl.Name), decl.Span)
		}
		typeNames[decl.Name] = decl.Span
		env.handles[decl.Name] = decl
	}
	return nil
}

func evt1IsHandle(env *semanticEnv, t Type) bool {
	if env == nil || t.PointerTo != nil || t.ArrayElem != nil || len(t.TypeArgs) != 0 {
		return false
	}
	_, ok := env.handles[t.Name]
	return ok
}

// evt1HandleCDeclaration matches the C convention `typedef struct X_T* X;`
// structurally, so a Concept handle and the foreign header's typedef name
// the same C type and may both be in scope.
func evt1HandleCDeclaration(decl HandleDecl) string {
	return fmt.Sprintf("typedef struct %s_T* %s;\n", decl.Name, evt1CName(decl.Name))
}
