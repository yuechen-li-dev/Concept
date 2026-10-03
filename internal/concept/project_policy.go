package concept

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LintSeverity string

const (
	LintWarning LintSeverity = "warning"
	LintError   LintSeverity = "error"
)

// LintPolicy is ordinary manifest.concept data. Concept names are resolved
// against the validated semantic concept environment before evaluation.
type LintPolicy struct {
	Identity string
	Severity LintSeverity
	Kind     DeclarationKind // empty means every declaration kind
	Name     string          // empty means every declaration name
	Source   string
	Site     Span
}

type LintFinding struct {
	Severity   LintSeverity          `json:"severity"`
	Policy     string                `json:"policy"`
	SubjectID  string                `json:"subject_id"`
	Kind       DeclarationKind       `json:"kind"`
	Name       string                `json:"name"`
	Source     string                `json:"source"`
	Site       Span                  `json:"site"`
	Outcome    SemanticFactCertainty `json:"outcome"`
	Message    string                `json:"message"`
	Suggestion string                `json:"suggestion,omitempty"`
	Proof      ProofGraph            `json:"-"`
}

// LoadProjectPolicies checks the ordinary Concept manifest first, then binds
// its immutable LintPolicy values. The manifest owns policy source identity.
func LoadProjectPolicies(path string, roots []string) ([]LintPolicy, []ConceptDecl, error) {
	policies, module, err := loadProjectPolicyModule(path, roots)
	return policies, module.Concepts, err
}

func loadProjectPolicyModule(path string, roots []string) ([]LintPolicy, Module, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, Module{}, err
	}
	module, err := ParseWithSemanticModuleRoots(filepath.ToSlash(path), string(body), roots)
	if err != nil && strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
		module, err = ParseWithBuiltSemanticModuleRoots(filepath.ToSlash(path), string(body), roots)
	}
	if err != nil {
		return nil, Module{}, err
	}
	// The resolved module includes imported declarations. Only values authored
	// in this manifest activate this project's lint policy.
	local, err := parseSyntaxModule(filepath.ToSlash(path), string(body))
	if err != nil {
		return nil, Module{}, err
	}
	var policies []LintPolicy
	for _, decl := range local.ComptimeDecls {
		if decl.Type.Name != "LintPolicy" {
			continue
		}
		value, ok := decl.Value.(*StructConstructExpr)
		if !ok || len(value.Args) != 4 {
			return nil, Module{}, evt1Diagnostic("LINT_MANIFEST_INVALID", "LintPolicy requires concept, severity, kind, and subject strings", decl.Span)
		}
		parts := make([]string, 4)
		for i, arg := range value.Args {
			literal, ok := arg.(*StringLiteral)
			if !ok {
				return nil, Module{}, evt1Diagnostic("LINT_MANIFEST_INVALID", "LintPolicy fields must be immutable string literals", decl.Span)
			}
			parts[i] = literal.Value
		}
		if parts[0] == "" {
			return nil, Module{}, evt1Diagnostic("LINT_MANIFEST_INVALID", "LintPolicy concept identity is empty", decl.Span)
		}
		severity := LintSeverity(parts[1])
		if severity != LintWarning && severity != LintError {
			return nil, Module{}, evt1Diagnostic("LINT_MANIFEST_INVALID", "LintPolicy severity must be warning or error", decl.Span)
		}
		kind := DeclarationKind(parts[2])
		if kind != "" && canonicalDeclarationStyle(kind) == "" {
			return nil, Module{}, evt1Diagnostic("LINT_MANIFEST_INVALID", "LintPolicy declaration kind is unknown", decl.Span)
		}
		policies = append(policies, LintPolicy{Identity: parts[0], Severity: severity, Kind: kind, Name: parts[3], Source: filepath.ToSlash(path), Site: decl.Span})
	}
	return policies, module, nil
}

// LintModule uses the same concept proof projector as Assert.Concept. The
// manifest selects subjects and severity; neither alters proposition truth.
func LintModule(module Module, policies []LintPolicy, manifestConcepts []ConceptDecl) ([]LintFinding, error) {
	return lintModuleWithPredicateEnvironment(module, policies, manifestConcepts, nil)
}

func lintModuleWithPredicateEnvironment(module Module, policies []LintPolicy, manifestConcepts []ConceptDecl, predicates *semanticEnv) ([]LintFinding, error) {
	bound := module
	bound.Concepts = append(append([]ConceptDecl{}, module.Concepts...), manifestConcepts...)
	env, err := evt1AnalyzeModule(bound, evt1AnalysisOptions{policyPredicates: predicates})
	if err != nil {
		return nil, err
	}
	var findings []LintFinding
	styles := map[string]string{}
	for _, policy := range policies {
		conceptDecl, ok := env.concepts[policy.Identity]
		if !ok {
			return nil, fmt.Errorf("LINT_POLICY_UNKNOWN: %s in %s", policy.Identity, policy.Source)
		}
		parameters := evt1ConceptParameters(conceptDecl)
		if len(parameters) != 1 || parameters[0].Kind != "declaration" {
			return nil, fmt.Errorf("LINT_POLICY_CATEGORY_INVALID: %s must accept one declaration subject", policy.Identity)
		}
		for _, subject := range ProjectDeclarationSubjects(module) {
			if policy.Kind != "" && policy.Kind != subject.Kind || policy.Name != "" && policy.Name != subject.Name {
				continue
			}
			style := policyNameStyle(conceptDecl, subject)
			if style != "" {
				if previous := styles[subject.ID]; previous != "" && previous != style {
					return nil, fmt.Errorf("LINT_POLICY_CONFLICT: %s requires both %s and %s for %s", subject.Name, previous, style, subject.ID)
				}
				styles[subject.ID] = style
			}
			graph, err := buildPolicyProof(env, policy, subject)
			if err != nil {
				return nil, err
			}
			if graph.Outcome == FactProven {
				continue
			}
			message := fmt.Sprintf("%s %q violates %s: %s", subject.Kind, subject.Name, policy.Identity, graph.Outcome)
			for _, node := range graph.Nodes {
				if node.Verdict != nil && node.Outcome != FactProven {
					message += "; " + evt1VerdictDiagnosticDetail(*node.Verdict)
				}
			}
			suggestion := ""
			if style != "" {
				message += "; expected " + style
				suggestion = declarationNameSuggestion(subject.Name, style)
			}
			findings = append(findings, LintFinding{Severity: policy.Severity, Policy: policy.Identity, SubjectID: subject.ID, Kind: subject.Kind, Name: subject.Name, Source: module.Path, Site: subject.Site, Outcome: graph.Outcome, Message: message, Suggestion: suggestion, Proof: graph})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Site.Line != b.Site.Line {
			return a.Site.Line < b.Site.Line
		}
		if a.Site.Column != b.Site.Column {
			return a.Site.Column < b.Site.Column
		}
		if a.Policy != b.Policy {
			return a.Policy < b.Policy
		}
		return a.SubjectID < b.SubjectID
	})
	return findings, nil
}

func policyNameStyle(decl ConceptDecl, subject DeclarationSubject) string {
	for _, raw := range decl.Requirements {
		requirement, ok := raw.(*CompilerAnalysisRequirement)
		if !ok {
			continue
		}
		switch requirement.Analysis {
		case "CanonicalName":
			if subject.Provenance == DeclarationAuthored {
				return canonicalDeclarationStyle(subject.Kind)
			}
		case "PascalCase":
			return "PascalCase"
		case "CamelCase":
			return "camelCase"
		case "SnakeCase":
			return "snake_case"
		}
	}
	return ""
}

func buildPolicyProof(env *semanticEnv, policy LintPolicy, subject DeclarationSubject) (ProofGraph, error) {
	assertion := conceptAssertionSubject{description: ProofSubjectDescription{Kind: string(subject.Kind), Name: subject.Name, Type: string(subject.Provenance)}, declaration: &subject, span: subject.Site}
	reason := fmt.Sprintf("policy from %s:%d; severity %s; subject %s with %s provenance", policy.Source, policy.Site.Line, policy.Severity, subject.Kind, subject.Provenance)
	return evt1BuildConceptAssertionGraph(env, policy.Identity, nil, []conceptAssertionSubject{assertion}, reason, subject.Site)
}

func ExplainPolicy(path, identity, subjectName string, roots []string) (ProofGraph, error) {
	projectRoot := filepath.Dir(path)
	policies, policyModule, err := loadProjectPolicyModule(filepath.Join(projectRoot, "manifest.concept"), append(append([]string{}, roots...), projectRoot))
	if err != nil {
		return ProofGraph{}, err
	}
	predicates, err := analyzeModule(policyModule)
	if err != nil {
		return ProofGraph{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return ProofGraph{}, err
	}
	moduleRoots := append(append([]string{}, roots...), projectRoot)
	module, err := ParseWithSemanticModuleRoots(filepath.ToSlash(path), string(body), moduleRoots)
	if err != nil && strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
		module, err = ParseWithBuiltSemanticModuleRoots(filepath.ToSlash(path), string(body), moduleRoots)
	}
	if err != nil {
		return ProofGraph{}, err
	}
	bound := module
	bound.Concepts = append(append([]ConceptDecl{}, module.Concepts...), policyModule.Concepts...)
	env, err := evt1AnalyzeModule(bound, evt1AnalysisOptions{policyPredicates: predicates})
	if err != nil {
		return ProofGraph{}, err
	}
	for _, policy := range policies {
		if policy.Identity != identity {
			continue
		}
		for _, subject := range ProjectDeclarationSubjects(module) {
			if subject.Name != subjectName || policy.Kind != "" && policy.Kind != subject.Kind || policy.Name != "" && policy.Name != subject.Name {
				continue
			}
			return buildPolicyProof(env, policy, subject)
		}
	}
	return ProofGraph{}, fmt.Errorf("LINT_POLICY_SUBJECT_NOT_FOUND: %s on %s", identity, subjectName)
}

func HasLintErrors(findings []LintFinding) bool {
	for _, finding := range findings {
		if finding.Severity == LintError {
			return true
		}
	}
	return false
}

func FormatLintFinding(finding LintFinding) string {
	message := fmt.Sprintf("%s:%d:%d: %s [%s] %s", finding.Source, finding.Site.Line, finding.Site.Column, finding.Severity, finding.Policy, finding.Message)
	if finding.Suggestion != "" {
		message += "; suggested: " + finding.Suggestion
	}
	return strings.TrimSpace(message)
}

// LintPath resolves a source file or project directory through the ordinary
// module parser and artifact resolver. A file without a manifest has no active
// policy but still receives core semantic validation.
func LintPath(target string, roots []string) ([]LintFinding, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	projectRoot := filepath.Dir(target)
	if info.IsDir() {
		projectRoot = target
	}
	manifestPath := filepath.Join(projectRoot, "manifest.concept")
	var policies []LintPolicy
	var policyConcepts []ConceptDecl
	var predicates *semanticEnv
	if _, err := os.Stat(manifestPath); err == nil {
		var policyModule Module
		policies, policyModule, err = loadProjectPolicyModule(manifestPath, append(append([]string{}, roots...), projectRoot))
		if err != nil {
			return nil, err
		}
		policyConcepts = policyModule.Concepts
		predicates, err = analyzeModule(policyModule)
		if err != nil {
			return nil, err
		}
	} else if info.IsDir() {
		return nil, fmt.Errorf("LINT_MANIFEST_MISSING: %s", filepath.ToSlash(manifestPath))
	}
	var paths []string
	if info.IsDir() {
		err = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != target && (entry.Name() == "artifacts" || entry.Name() == ".git") {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Name() != "manifest.concept" && strings.HasSuffix(strings.ToLower(entry.Name()), ".concept") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else if filepath.Base(target) == "manifest.concept" {
		return LintPath(projectRoot, roots)
	} else {
		paths = []string{target}
	}
	var findings []LintFinding
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		moduleRoots := append(append([]string{}, roots...), projectRoot, filepath.Dir(path))
		module, err := ParseWithSemanticModuleRoots(filepath.ToSlash(path), string(body), moduleRoots)
		if err != nil && strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
			module, err = ParseWithBuiltSemanticModuleRoots(filepath.ToSlash(path), string(body), moduleRoots)
		}
		if err != nil {
			return nil, err
		}
		current, err := lintModuleWithPredicateEnvironment(module, policies, policyConcepts, predicates)
		if err != nil {
			return nil, err
		}
		findings = append(findings, current...)
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Site.Line != b.Site.Line {
			return a.Site.Line < b.Site.Line
		}
		if a.Site.Column != b.Site.Column {
			return a.Site.Column < b.Site.Column
		}
		if a.Policy != b.Policy {
			return a.Policy < b.Policy
		}
		return a.SubjectID < b.SubjectID
	})
	return findings, nil
}
