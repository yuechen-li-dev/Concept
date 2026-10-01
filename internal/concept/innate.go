package concept

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"
)

// Innate concepts are the rules the compiler holds before it reads any
// program. They are written in Concept, in the embedded innate module, and
// the compiler applies each one to every declaration of its kind. See
// docs/design/EVT2-INNATE-CONCEPTS.md.

//go:embed innate/Innate.concept
var evt1InnateSource string

const evt1InnateModulePath = "compiler/innate/Innate.concept"

// InnateIdentity names the innate concept set this compiler carries. It is
// recorded in every semantic module artifact; an artifact built under another
// innate set is stale, because its declarations were checked by other rules.
func InnateIdentity() string {
	return evt1InnateIdentityOf(evt1InnateSource)
}

// Line endings depend on the checkout; the identity must not.
func evt1InnateIdentityOf(source string) string {
	return "innate-" + digest([]byte(strings.ReplaceAll(source, "\r\n", "\n")))[:16]
}

type evt1InnateSet struct {
	module Module
	env    *semanticEnv
}

var (
	evt1InnateOnce   sync.Once
	evt1InnateLoaded evt1InnateSet
	evt1InnateErr    error
)

// evt1LoadInnate compiles the embedded innate module once per process.
func evt1LoadInnate() (evt1InnateSet, error) {
	evt1InnateOnce.Do(func() {
		module, env, err := evt1CompileInnateSource(evt1InnateModulePath, evt1InnateSource)
		if err != nil {
			evt1InnateErr = fmt.Errorf("INNATE_MODULE_INVALID: the compiler's innate module does not compile: %w", err)
			return
		}
		evt1InnateLoaded = evt1InnateSet{module: module, env: env}
	})
	return evt1InnateLoaded, evt1InnateErr
}

// evt1CompileInnateSource compiles source with innate authority: the only
// path on which `innate concept` is admitted.
func evt1CompileInnateSource(path, source string) (Module, *semanticEnv, error) {
	module, err := parseSyntaxModule(path, source)
	if err != nil {
		return Module{}, nil, err
	}
	module.innateAuthority = true
	env, err := analyzeModule(module)
	if err != nil {
		return Module{}, nil, err
	}
	if err := evt1ValidateVerdictShape(env); err != nil {
		return Module{}, nil, err
	}
	return module, env, nil
}

// The Verdict enum is the contract between the compiler and innate
// predicates; its shape is fixed.
func evt1ValidateVerdictShape(env *semanticEnv) error {
	verdict, ok := env.enums["Verdict"]
	wrong := func(detail string) error {
		return evt1Diagnostic("INNATE_VERDICT_SHAPE", "the innate module must declare enum Verdict { Holds, Refuted(declaration at, string message) }: "+detail, verdict.Span)
	}
	if !ok {
		return wrong("Verdict is missing")
	}
	if len(verdict.Variants) != 2 || verdict.Variants[0].Name != "Holds" || verdict.Variants[1].Name != "Refuted" {
		return wrong("variants must be Holds, Refuted")
	}
	if len(verdict.Variants[0].Payload) != 0 {
		return wrong("Holds carries no payload")
	}
	refuted := verdict.Variants[1].Payload
	if len(refuted) != 2 || refuted[0].Type.Name != evt1DeclarationTypeName || refuted[1].Type.Name != "string" {
		return wrong("Refuted carries (declaration at, string message)")
	}
	return nil
}

var evt1InnateDeclarationKinds = map[string]DeclarationKind{
	string(TypeDeclaration):      TypeDeclaration,
	string(FieldDeclaration):     FieldDeclaration,
	string(FunctionDeclaration):  FunctionDeclaration,
	string(MethodDeclaration):    MethodDeclaration,
	string(ParameterDeclaration): ParameterDeclaration,
	string(LocalDeclaration):     LocalDeclaration,
	string(ConceptDeclaration):   ConceptDeclaration,
	string(InterfaceDeclaration): InterfaceDeclaration,
	string(MachineDeclaration):   MachineDeclaration,
}

// evt1InnateDiagnosticCode returns the code a concept's [[diagnostic("CODE")]]
// attribute assigns, if any.
func evt1InnateDiagnosticCode(decl ConceptDecl) (string, bool) {
	for _, attribute := range decl.Attributes {
		if attribute.Name != "diagnostic" || len(attribute.Args) != 1 {
			continue
		}
		if literal, ok := attribute.Args[0].(*StringLiteral); ok {
			return literal.Value, true
		}
	}
	return "", false
}

func evt1ValidateConceptAttributes(env *semanticEnv, decl ConceptDecl) error {
	for _, attribute := range decl.Attributes {
		if attribute.Name != "diagnostic" {
			return evt1Diagnostic("CONCEPT_ATTRIBUTE_INVALID", fmt.Sprintf("[[%s]] does not apply to a concept", attribute.Name), attribute.Span)
		}
		if !decl.Innate {
			return evt1Diagnostic("CONCEPT_ATTRIBUTE_INVALID", "[[diagnostic]] names the diagnostic of an innate concept", attribute.Span)
		}
		literal, ok := (*StringLiteral)(nil), false
		if len(attribute.Args) == 1 {
			literal, ok = attribute.Args[0].(*StringLiteral)
		}
		if !ok || literal.Value == "" || strings.ContainsAny(literal.Value, " \t\n") {
			return evt1Diagnostic("CONCEPT_ATTRIBUTE_INVALID", `[[diagnostic]] takes one diagnostic code, such as [[diagnostic("CV4653")]]`, attribute.Span)
		}
	}
	return nil
}

func evt1ValidateInnateConceptDecl(env *semanticEnv, decl ConceptDecl) error {
	if err := evt1ValidateConceptAttributes(env, decl); err != nil {
		return err
	}
	parameters := evt1ConceptParameters(decl)
	if !decl.Innate {
		for _, parameter := range parameters {
			if parameter.DeclarationKind != "" {
				return evt1Diagnostic("INNATE_KIND_PARAMETER", fmt.Sprintf("%s %s: a declaration-kind parameter selects the declarations an innate concept applies to; declared concepts use `declaration %s`", parameter.DeclarationKind, parameter.Name, parameter.Name), parameter.Span)
			}
		}
		return nil
	}
	if !env.innateAuthority {
		return evt1Diagnostic("INNATE_OUTSIDE_COMPILER", fmt.Sprintf("innate concept %s: innate concepts are declared only in the compiler's innate module; declare a concept and require it instead", decl.Name), decl.Span)
	}
	if decl.Interface {
		return evt1Diagnostic("INNATE_CONCEPT_SHAPE", "an interface cannot be innate", decl.Span)
	}
	if len(parameters) != 1 || parameters[0].Kind != "declaration" || parameters[0].DeclarationKind == "" {
		return evt1Diagnostic("INNATE_CONCEPT_SHAPE", fmt.Sprintf("innate concept %s takes exactly one declaration-kind parameter, such as <FieldDeclaration F>", decl.Name), decl.Span)
	}
	if kind := DeclarationKind(parameters[0].DeclarationKind); kind != TypeDeclaration && kind != FieldDeclaration {
		return evt1Diagnostic("INNATE_CONCEPT_SHAPE", fmt.Sprintf("innate concepts apply to TypeDeclaration or FieldDeclaration in this compiler, not %s", kind), parameters[0].Span)
	}
	if _, ok := evt1InnateDiagnosticCode(decl); !ok {
		return evt1Diagnostic("INNATE_CONCEPT_SHAPE", fmt.Sprintf(`innate concept %s needs [[diagnostic("CODE")]]: the code is its stable identity in diagnostics and the corpus`, decl.Name), decl.Span)
	}
	if len(decl.Requirements) == 0 {
		return evt1Diagnostic("INNATE_CONCEPT_SHAPE", fmt.Sprintf("innate concept %s requires nothing", decl.Name), decl.Span)
	}
	for _, requirement := range decl.Requirements {
		switch requirement.(type) {
		case *PredicateRequirement, *CompilerAnalysisRequirement, *PrerequisiteRequirement:
		default:
			return evt1Diagnostic("INNATE_CONCEPT_SHAPE", "innate concepts require predicates, compiler analyses, or other concepts over their declaration", requirement.requirementSpan())
		}
	}
	return nil
}

// evt1ValidatePredicateRequirement checks `requires Predicate(D);`: the
// predicate is a comptime function from declaration subjects to Verdict,
// applied to the concept's declaration parameters.
func evt1ValidatePredicateRequirement(env *semanticEnv, decl ConceptDecl, r *PredicateRequirement) error {
	if !decl.Innate {
		return evt1Diagnostic("PREDICATE_REQUIREMENT_SCOPE", fmt.Sprintf("requires %s(...): predicate requirements are admitted in innate concepts", r.Predicate), r.Span)
	}
	fn, ok := env.comptimeFunctions[r.Predicate]
	if !ok {
		return evt1Diagnostic("PREDICATE_REQUIREMENT_INVALID", fmt.Sprintf("%s is not a comptime function", r.Predicate), r.Span)
	}
	if fn.ReturnType.Name != "Verdict" || len(fn.ReturnType.TypeArgs) != 0 || fn.ReturnType.ArrayElem != nil {
		return evt1Diagnostic("PREDICATE_REQUIREMENT_INVALID", fmt.Sprintf("predicate %s must return Verdict, not %s", r.Predicate, fn.ReturnType.String()), r.Span)
	}
	if len(fn.Params) != len(r.Subjects) {
		return evt1Diagnostic("PREDICATE_REQUIREMENT_INVALID", fmt.Sprintf("predicate %s takes %d declaration(s), got %d", r.Predicate, len(fn.Params), len(r.Subjects)), r.Span)
	}
	for i, subject := range r.Subjects {
		if fn.Params[i].Type.Name != evt1DeclarationTypeName {
			return evt1Diagnostic("PREDICATE_REQUIREMENT_INVALID", fmt.Sprintf("predicate %s parameter %s must be a declaration", r.Predicate, fn.Params[i].Name), r.Span)
		}
		found := false
		for _, parameter := range evt1ConceptParameters(decl) {
			if parameter.Kind == "declaration" && parameter.Name == subject.Name {
				found = true
			}
		}
		if !found {
			return evt1Diagnostic("CONCEPT_DECLARATION_ARGUMENT_INVALID", fmt.Sprintf("%s is not a declaration parameter of %s", subject.Name, decl.Name), subject.Span)
		}
	}
	return nil
}
