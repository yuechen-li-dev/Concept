package concept

import (
	"fmt"
	"sort"
	"strings"
)

var evt1SharedAccessArities = map[string]int{
	string(FactExclusiveWriter):    2,
	string(FactSingleProducer):     2,
	string(FactSingleConsumer):     2,
	string(FactSynchronizedAccess): 1,
	string(FactPublishedBefore):    2,
	string(FactExactlyOnce):        1,
}

func evt1IsSharedAccessAnalysis(name string) bool {
	_, ok := evt1SharedAccessArities[name]
	return ok
}

func evt1SharedAccessSubject(t Type) SemanticFactSubject {
	return SemanticFactSubject{Kind: "type", Name: t.String(), Type: t.String()}
}

func evt1SharedAccessFact(kind SemanticFactKind, args []Type, origin SemanticFactOrigin, authority string, span Span) MIRSemanticFact {
	subjects := make([]SemanticFactSubject, len(args))
	for i, arg := range args {
		subjects[i] = evt1SharedAccessSubject(arg)
	}
	detail := "explicit synchronization authority contract"
	if len(subjects) == 2 {
		switch kind {
		case FactExclusiveWriter:
			detail = "writer set = {" + subjects[0].Type + "}; explicit authority contract"
		case FactSingleProducer:
			detail = "producer set = {" + subjects[0].Type + "}; explicit authority contract"
		case FactSingleConsumer:
			detail = "consumer set = {" + subjects[0].Type + "}; explicit authority contract"
		}
	}
	evidence := SemanticFactEvidence{Detail: detail, Authority: authority}
	return MIRSemanticFact{ID: evt1SemanticFactID(kind, subjects, nil, span), Kind: kind, Subjects: subjects, Outcome: FactProven, Origin: origin, Evidence: evidence, SourceSpan: span}
}

func evt1RegisterSharedAccessFacts(env *semanticEnv, module Module) error {
	for _, incoming := range module.SharedAccessFacts {
		fact := incoming
		fact.Origin = FactOriginModuleFactSummary
		if err := evt1AddSharedAccessFact(env, fact); err != nil {
			return err
		}
	}
	for _, assertion := range module.Assertions {
		if !evt1IsSharedAccessAnalysis(assertion.ConceptName) {
			continue
		}
		args := assertion.TypeArgs
		if len(args) == 0 {
			args = []Type{assertion.ConcreteType}
		}
		if len(args) != evt1SharedAccessArities[assertion.ConceptName] {
			return evt1Diagnostic("SYNC_CONTRACT_ARITY", fmt.Sprintf("%s requires %d type argument(s), got %d", assertion.ConceptName, evt1SharedAccessArities[assertion.ConceptName], len(args)), assertion.Span)
		}
		resolved := make([]Type, len(args))
		for i, arg := range args {
			if err := validateKnownType(env, arg, assertion.Span, "", false); err != nil {
				return err
			}
			value, err := evt1ResolveType(env, nil, arg)
			if err != nil {
				return err
			}
			resolved[i] = value
		}
		fact := evt1SharedAccessFact(evt1FactKind(assertion.ConceptName), resolved, FactOriginDeclared, env.moduleName, assertion.Span)
		if err := evt1AddSharedAccessFact(env, fact); err != nil {
			return err
		}
	}
	return nil
}

func evt1AddSharedAccessFact(env *semanticEnv, fact MIRSemanticFact) error {
	if !evt1IsSharedAccessAnalysis(string(fact.Kind)) || fact.Outcome != FactProven || len(fact.Subjects) != evt1SharedAccessArities[string(fact.Kind)] {
		return evt1Diagnostic("SYNC_CONTRACT_INVALID", "invalid transported synchronization contract", fact.SourceSpan)
	}
	if len(fact.Subjects) == 2 && (fact.Kind == FactExclusiveWriter || fact.Kind == FactSingleProducer || fact.Kind == FactSingleConsumer) {
		for _, existing := range env.sharedAccessFacts {
			if existing.Kind == fact.Kind && existing.Subjects[1].Type == fact.Subjects[1].Type && existing.Subjects[0].Type != fact.Subjects[0].Type {
				return evt1Diagnostic("SYNC_AUTHORITY_CONTRADICTION", fmt.Sprintf("%s assigns both %s and %s to %s", fact.Kind, existing.Subjects[0].Type, fact.Subjects[0].Type, fact.Subjects[1].Type), fact.SourceSpan)
			}
		}
	}
	for _, existing := range env.sharedAccessFacts {
		if evt1SameOrderedFact(existing, fact.Kind, fact.Subjects) {
			return nil
		}
	}
	env.sharedAccessFacts = append(env.sharedAccessFacts, fact)
	env.transportedFacts = append(env.transportedFacts, fact)
	return nil
}

func evt1SameOrderedFact(fact MIRSemanticFact, kind SemanticFactKind, subjects []SemanticFactSubject) bool {
	if fact.Kind != kind || len(fact.Subjects) != len(subjects) {
		return false
	}
	for i := range subjects {
		if fact.Subjects[i].Type != subjects[i].Type {
			return false
		}
	}
	return true
}

func evt1CheckSharedAccessFact(env *semanticEnv, kind SemanticFactKind, args []Type) semanticFactResult {
	subjects := make([]SemanticFactSubject, len(args))
	for i, arg := range args {
		resolved, err := evt1ResolveType(env, nil, arg)
		if err == nil {
			arg = resolved
		}
		subjects[i] = evt1SharedAccessSubject(arg)
	}
	for _, fact := range env.sharedAccessFacts {
		if evt1SameOrderedFact(fact, kind, subjects) {
			return semanticFactResult{Outcome: FactProven, Origin: fact.Origin, Evidence: fact.Evidence, SourceSpan: fact.SourceSpan}
		}
	}
	if len(subjects) == 2 && (kind == FactExclusiveWriter || kind == FactSingleProducer || kind == FactSingleConsumer) {
		for _, fact := range env.sharedAccessFacts {
			if fact.Kind == kind && fact.Subjects[1].Type == subjects[1].Type && fact.Subjects[0].Type != subjects[0].Type {
				return semanticFactResult{Outcome: FactDisproven, Origin: fact.Origin, Evidence: SemanticFactEvidence{Detail: fmt.Sprintf("%s is declared for %s", fact.Subjects[1].Type, fact.Subjects[0].Type), Authority: fact.Evidence.Authority}, SourceSpan: fact.SourceSpan}
			}
		}
	}
	return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginCompilerAnalysis, Evidence: SemanticFactEvidence{Detail: "no declared or derived synchronization fact matches the ordered subjects"}}
}

func evt1LocalSharedAccessFacts(module Module, env *semanticEnv) []MIRSemanticFact {
	var facts []MIRSemanticFact
	for _, fact := range env.sharedAccessFacts {
		if fact.Origin != FactOriginDeclared || fact.Evidence.Authority != module.Name {
			continue
		}
		facts = append(facts, fact)
	}
	sort.Slice(facts, func(i, j int) bool {
		return strings.Compare(string(facts[i].Kind)+facts[i].ID, string(facts[j].Kind)+facts[j].ID) < 0
	})
	return facts
}
