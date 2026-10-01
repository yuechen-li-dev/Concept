package concept

import (
	"fmt"
	"sort"
)

// DeclarationKind describes a checked semantic declaration, not a token shape.
type DeclarationKind string

const (
	TypeDeclaration      DeclarationKind = "TypeDeclaration"
	FunctionDeclaration  DeclarationKind = "FunctionDeclaration"
	MethodDeclaration    DeclarationKind = "MethodDeclaration"
	FieldDeclaration     DeclarationKind = "FieldDeclaration"
	LocalDeclaration     DeclarationKind = "LocalDeclaration"
	ParameterDeclaration DeclarationKind = "ParameterDeclaration"
	ConceptDeclaration   DeclarationKind = "ConceptDeclaration"
	InterfaceDeclaration DeclarationKind = "InterfaceDeclaration"
	MachineDeclaration   DeclarationKind = "MachineDeclaration"
)

type DeclarationProvenance string

const (
	DeclarationAuthored  DeclarationProvenance = "Authored"
	DeclarationGenerated DeclarationProvenance = "Generated"
	DeclarationForeign   DeclarationProvenance = "Foreign"
)

// DeclarationSubject is a read-only projection of a validated Module. Owner
// records the defining semantic module, so root policy can exclude dependencies.
// ID is independent of diagnostic wording and stable for one source revision.
type DeclarationSubject struct {
	ID         string                `json:"id"`
	Kind       DeclarationKind       `json:"kind"`
	Name       string                `json:"name"`
	Owner      string                `json:"owner"`
	Provenance DeclarationProvenance `json:"provenance"`
	Site       Span                  `json:"site"`
}

// DeclarationSubjects must be called on a module returned by Parse or
// ParseWithSemanticModules. It never reparses source or guesses kind from names.
func DeclarationSubjects(module Module) []DeclarationSubject {
	var subjects []DeclarationSubject
	add := func(kind DeclarationKind, name, owner string, provenance DeclarationProvenance, site Span) {
		if owner == "" {
			owner = module.Path // standalone source without a module declaration
		}
		if name == "" || owner == "" {
			return
		}
		identity := fmt.Sprintf("%s|%s|%s|%s|%d:%d", owner, kind, name, provenance, site.Line, site.Column)
		subjects = append(subjects, DeclarationSubject{
			ID: "declaration-" + digest([]byte(identity))[:16], Kind: kind,
			Name: name, Owner: owner, Provenance: provenance, Site: site,
		})
	}
	for _, decl := range module.Structs {
		// Closed generic instances are semantic types, but are not new authored
		// declarations and must not acquire a second naming obligation.
		if decl.Application != nil {
			continue
		}
		add(TypeDeclaration, decl.Name, decl.Module, DeclarationAuthored, decl.Span)
		for _, field := range decl.Fields {
			add(FieldDeclaration, field.Name, decl.Module, DeclarationAuthored, field.Span)
		}
	}
	for _, decl := range module.Enums {
		add(TypeDeclaration, decl.Name, decl.Module, DeclarationAuthored, decl.Span)
		for _, variant := range decl.Variants {
			for _, field := range variant.Payload {
				add(FieldDeclaration, field.Name, decl.Module, DeclarationAuthored, field.Span)
			}
		}
	}
	for _, decl := range module.GenericTypes {
		add(TypeDeclaration, decl.Name, decl.Module, DeclarationAuthored, decl.Span)
	}
	for _, decl := range module.Concepts {
		kind := ConceptDeclaration
		if decl.Interface {
			kind = InterfaceDeclaration
		}
		add(kind, decl.Name, decl.Module, DeclarationAuthored, decl.Span)
	}
	for _, automata := range module.Automata {
		for _, machine := range automata.Machines {
			add(MachineDeclaration, machine.Name, automata.Module, DeclarationAuthored, machine.Span)
			for _, field := range machine.Fields {
				add(FieldDeclaration, field.Name, automata.Module, DeclarationAuthored, field.Span)
			}
			for _, state := range machine.States {
				if state.Body != nil {
					collectDeclarationLocals(*state.Body, automata.Module, DeclarationAuthored, add)
				}
			}
		}
	}
	collectFunction := func(fn FunctionDecl) {
		kind := FunctionDeclaration
		if fn.MethodOf != "" {
			kind = MethodDeclaration
		}
		provenance := DeclarationAuthored
		if fn.Generated != nil {
			provenance = DeclarationGenerated
		} else if fn.ExternABI != "" {
			provenance = DeclarationForeign
		}
		add(kind, fn.Name, fn.Module, provenance, fn.Span)
		for _, param := range fn.Params {
			add(ParameterDeclaration, param.Name, fn.Module, provenance, param.Span)
		}
		if fn.Body != nil {
			collectDeclarationLocals(*fn.Body, fn.Module, provenance, add)
		}
	}
	for _, fn := range module.Functions {
		collectFunction(fn)
	}
	for _, fn := range module.ComptimeFns {
		collectFunction(fn)
	}
	for _, template := range module.Templates {
		add(FunctionDeclaration, template.Name, template.Module, DeclarationAuthored, template.Span)
		for _, param := range template.Params {
			add(ParameterDeclaration, param.Name, template.Module, DeclarationAuthored, param.Span)
		}
		if template.Body != nil {
			collectDeclarationLocals(*template.Body, template.Module, DeclarationAuthored, add)
		}
	}
	sort.Slice(subjects, func(i, j int) bool {
		a, b := subjects[i], subjects[j]
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		if a.Site.Line != b.Site.Line {
			return a.Site.Line < b.Site.Line
		}
		if a.Site.Column != b.Site.Column {
			return a.Site.Column < b.Site.Column
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	return subjects
}

// ProjectDeclarationSubjects excludes implementation declarations transported
// from dependency artifacts. Imported contracts remain available to semantic
// proof queries through the validated Module itself.
func ProjectDeclarationSubjects(module Module) []DeclarationSubject {
	all := DeclarationSubjects(module)
	project := make([]DeclarationSubject, 0, len(all))
	owner := module.Name
	if owner == "" {
		owner = module.Path
	}
	for _, subject := range all {
		if subject.Owner == owner {
			project = append(project, subject)
		}
	}
	return project
}

func collectDeclarationLocals(block Block, owner string, provenance DeclarationProvenance, add func(DeclarationKind, string, string, DeclarationProvenance, Span)) {
	for _, statement := range block.Statements {
		switch s := statement.(type) {
		case *VarDecl:
			add(LocalDeclaration, s.Name, owner, provenance, s.Span)
		case *Block:
			collectDeclarationLocals(*s, owner, provenance, add)
		case *OnStmt:
			collectDeclarationLocals(s.Body, owner, provenance, add)
		case *IfStmt:
			collectDeclarationLocals(s.Then, owner, provenance, add)
			if s.Else != nil {
				collectDeclarationLocals(*s.Else, owner, provenance, add)
			}
		case *WhileStmt:
			collectDeclarationLocals(s.Body, owner, provenance, add)
			if s.Else != nil {
				collectDeclarationLocals(*s.Else, owner, provenance, add)
			}
		case *ForeachStmt:
			add(LocalDeclaration, s.ItemName, owner, provenance, s.Span)
			collectDeclarationLocals(s.Body, owner, provenance, add)
		case *TryStmt:
			collectDeclarationLocals(s.Body, owner, provenance, add)
			for _, arm := range s.Except {
				add(LocalDeclaration, arm.Binding, owner, provenance, arm.Span)
				collectDeclarationLocals(arm.Body, owner, provenance, add)
			}
		case *MatchStmt:
			for _, arm := range s.Arms {
				collectDeclarationLocals(arm.Block, owner, provenance, add)
			}
		}
	}
}
