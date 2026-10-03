package concept

import (
	"fmt"
	"reflect"
)

// A root manifest's predicates retain their lexical environment. Observations
// inspect the program being judged, as in innate evaluation; manifest helpers
// are never injected into the program's function lookup or effect authority.
func evt1PredicateEnvironment(env *semanticEnv, decl ConceptDecl) *semanticEnv {
	if policy := env.options.policyPredicates; policy != nil {
		if authored, ok := policy.concepts[decl.Name]; ok && reflect.DeepEqual(authored, decl) {
			return policy
		}
	}
	return env
}

func evt1EvaluateDeclaredPredicate(env *semanticEnv, decl ConceptDecl, requirement *PredicateRequirement, types map[string]Type, declarations map[string]DeclarationSubject) (SemanticFactCertainty, string, Span, *PredicateVerdict) {
	predicates := evt1PredicateEnvironment(env, decl)
	fn, ok := predicates.comptimeFunctions[requirement.Predicate]
	if !ok {
		return FactUnknown, "comptime predicate is unavailable", requirement.Span, nil
	}
	args := make([]Value, len(requirement.Subjects))
	for i, subject := range requirement.Subjects {
		if fn.Params[i].Type.Name == evt1TypenameTypeName {
			t, ok := types[subject.Name]
			if !ok {
				return FactUnknown, "type argument " + subject.Name + " is unbound", subject.Span, nil
			}
			args[i] = evt1TypenameValue(evt1CanonicalType(env, t))
		} else {
			declaration, ok := declarations[subject.Name]
			if !ok {
				return FactUnknown, "declaration argument " + subject.Name + " is unbound", subject.Span, nil
			}
			args[i] = evt1DeclarationValue(evt1SubjectDeclarationRef(env, declaration))
		}
	}
	result, err := evt1InvokePredicateOnMeasured(predicates, env, requirement.Predicate, args, requirement.Span, nil)
	if err != nil {
		return FactUnknown, fmt.Sprintf("%s could not decide: %v", requirement.Predicate, err), requirement.Span, nil
	}
	var metadata *PredicateVerdict
	if evt1IsTypedVerdict(result.Type) {
		metadata = &result
	}
	return result.Outcome, result.Message, result.At, metadata
}

func evt1SubjectDeclarationRef(env *semanticEnv, subject DeclarationSubject) evt1DeclarationRef {
	ref := evt1DeclarationRef{Kind: subject.Kind, Name: subject.Name, Owner: subject.Owner, Index: -1, Provenance: subject.Provenance, Site: subject.Site}
	if subject.Kind == FieldDeclaration {
		for _, decl := range env.structs {
			if decl.Application != nil && subject.Provenance != DeclarationGenerated {
				continue
			}
			for index, field := range decl.Fields {
				if field.Name == subject.Name && field.Span == subject.Site && evt1DeclarationOwner(env, decl.Module) == subject.Owner {
					ref.Parent, ref.Index = decl.Name, index
					return ref
				}
			}
		}
	}
	return ref
}

func evt1DeclarationOwner(env *semanticEnv, owner string) string {
	if owner != "" {
		return owner
	}
	if env.moduleName != "" {
		return env.moduleName
	}
	return env.sourcePath
}

func evt1SubjectFunction(env *semanticEnv, ref evt1DeclarationRef) (FunctionDecl, bool) {
	if ref.Kind != FunctionDeclaration && ref.Kind != MethodDeclaration {
		return FunctionDecl{}, false
	}
	for _, fn := range env.functions[ref.Name] {
		if fn.Span == ref.Site && evt1DeclarationOwner(env, fn.Module) == ref.Owner {
			return fn, true
		}
	}
	return FunctionDecl{}, false
}
