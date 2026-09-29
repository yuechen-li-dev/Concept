package concept

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestManifestNamingLint(t *testing.T) {
	root := t.TempDir()
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "warning", "", ""};
`
	program := `module Policy.Program; profile Core;
struct lower_type { int BadField; }
void do_work() { int SomeLocal = 0; }
extern "C" int sqlite3_open();
void GoodFunction() {}
`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "program.concept"), []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 4 {
		t.Fatalf("findings = %+v", findings)
	}
	for _, finding := range findings {
		if finding.Severity != LintWarning || finding.Policy != "ProjectNaming" || finding.Outcome != FactDisproven {
			t.Fatalf("finding = %+v", finding)
		}
		if finding.Name == "sqlite3_open" || finding.Name == "GoodFunction" {
			t.Fatalf("exempt or valid declaration flagged: %+v", finding)
		}
	}
	if HasLintErrors(findings) {
		t.Fatal("warnings changed lint exit status")
	}
	if !strings.Contains(FormatLintFinding(findings[2]), "ProjectNaming") {
		t.Fatal("missing policy identity")
	}
	baseline, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		again, err := LintPath(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(again)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, baseline) {
			t.Fatalf("lint output drifted on pass %d", i)
		}
	}
}

func TestPolicySeverityDoesNotChangeArtifact(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.concept")
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "warning", "", ""};`
	program := `module Policy.Program; profile Core; void do_work() {}`
	programPath := filepath.Join(root, "program.concept")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(programPath, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := CompileSemanticModule(programPath, program, nil)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(programPath, program)
	if err != nil {
		t.Fatal(err)
	}
	generatedBefore, err := Generate(module, []byte(program))
	if err != nil {
		t.Fatal(err)
	}
	warnings, err := LintPath(programPath, nil)
	if err != nil || len(warnings) != 1 || HasLintErrors(warnings) {
		t.Fatalf("warnings: %+v %v", warnings, err)
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(manifest, "warning", "error", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	errors, err := LintPath(programPath, nil)
	if err != nil || len(errors) != 1 || !HasLintErrors(errors) {
		t.Fatalf("errors: %+v %v", errors, err)
	}
	after, err := CompileSemanticModule(programPath, program, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("lint severity changed semantic artifact")
	}
	generatedAfter, err := Generate(module, []byte(program))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(generatedBefore, generatedAfter) {
		t.Fatal("lint severity changed MIR or C")
	}
	for i := 0; i < 100; i++ {
		repeatedArtifact, err := CompileSemanticModule(programPath, program, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(repeatedArtifact, before) {
			t.Fatalf("semantic artifact drifted on pass %d", i)
		}
		repeatedOutput, err := Generate(module, []byte(program))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(repeatedOutput, generatedBefore) {
			t.Fatalf("MIR or C drifted on pass %d", i)
		}
	}
}

func TestMustUseArtifactExplanation(t *testing.T) {
	root := t.TempDir()
	const provider = `module Policies.Provider; profile Core; [[must_use]] extern "C" int native_status();`
	artifact, err := CompileSemanticModule("provider.concept", provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "Policies", "Provider.concept-module.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, artifact, 0600); err != nil {
		t.Fatal(err)
	}
	const consumer = `module Policies.Consumer; profile Core; import Policies.Provider;
void Good() { discard native_status(); }`
	consumerPath := filepath.Join(root, "consumer.concept")
	if err := os.WriteFile(consumerPath, []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}
	graph, err := ExplainMustUse(consumerPath, "native_status", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if graph.Outcome != FactProven || !strings.Contains(RenderProofVerbose(graph), "DeclaredForeign") || !strings.Contains(RenderProofVerbose(graph), "semantic artifact") {
		t.Fatalf("MustUse origin: %+v", graph)
	}
	_, err = ParseWithSemanticModules(consumerPath, strings.Replace(consumer, "discard native_status();", "native_status();", 1), map[string][]byte{"Policies.Provider": artifact})
	if err == nil || !strings.Contains(err.Error(), "MUST_USE_RESULT_IGNORED") {
		t.Fatalf("ignored foreign result: %v", err)
	}
}

func TestConflictingPolicyStyles(t *testing.T) {
	root := t.TempDir()
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
concept SnakeFunctions<declaration D> { requires compiler.SnakeCase(D); }
comptime LintPolicy Canonical = LintPolicy{"ProjectNaming", "warning", "FunctionDeclaration", ""};
comptime LintPolicy Snake = LintPolicy{"SnakeFunctions", "error", "FunctionDeclaration", ""};`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "program.concept"), []byte(`module Policy.Program; profile Core; void FastFunction() {}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LintPath(root, nil)
	if err == nil || !strings.Contains(err.Error(), "LINT_POLICY_CONFLICT") || !strings.Contains(err.Error(), "PascalCase") || !strings.Contains(err.Error(), "snake_case") {
		t.Fatalf("conflicting policy = %v", err)
	}
}

func TestRootPolicyExcludesDependencyImplementation(t *testing.T) {
	root := t.TempDir()
	const dependency = `module Policy.Dependency; profile Core;
void noncanonical_dependency() {}`
	artifact, err := CompileSemanticModule("dependency.concept", dependency, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "Policy", "Dependency.concept-module.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, artifact, 0600); err != nil {
		t.Fatal(err)
	}
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "error", "", ""};`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	consumer := `module Policy.Consumer; profile Core; import Policy.Dependency;
void RootFunction() { noncanonical_dependency(); }`
	consumerPath := filepath.Join(root, "consumer.concept")
	if err := os.WriteFile(consumerPath, []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(root, nil)
	if err != nil || len(findings) != 0 {
		t.Fatalf("dependency implementation leaked into root lint: %+v, %v", findings, err)
	}
	module, err := ParseWithSemanticModuleRoots(consumerPath, consumer, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	var imported bool
	for _, subject := range DeclarationSubjects(module) {
		if subject.Name == "noncanonical_dependency" && subject.Owner == "Policy.Dependency" {
			imported = true
		}
	}
	if !imported {
		t.Fatal("dependency semantic declaration was unavailable to the root")
	}
}

func TestMalformedManifestPolicy(t *testing.T) {
	root := t.TempDir()
	manifest := `module Policy.Manifest; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "fatal", "", ""};`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "program.concept"), []byte(`module Policy.Program; profile Core; void GoodFunction() {}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LintPath(root, nil)
	if err == nil || !strings.Contains(err.Error(), "LINT_MANIFEST_INVALID") || !strings.Contains(err.Error(), "severity") {
		t.Fatalf("malformed manifest policy = %v", err)
	}
}

func TestImportedPolicyValueDoesNotActivateRootLint(t *testing.T) {
	root := t.TempDir()
	settings := `module Policy.Settings; profile Core;
record struct LintPolicy { string concept; string severity; string kind; string subject; }
concept ProjectNaming<declaration D> { requires compiler.CanonicalName(D); }
comptime LintPolicy Naming = LintPolicy{"ProjectNaming", "error", "", ""};`
	artifact, err := CompileSemanticModule("settings.concept", settings, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "Policy", "Settings.concept-module.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, artifact, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(`module Policy.Manifest; profile Core; import Policy.Settings;`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "program.concept"), []byte(`module Policy.Program; profile Core; void bad_name() {}`), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err := LintPath(root, nil)
	if err != nil || len(findings) != 0 {
		t.Fatalf("imported policy value activated root lint: %+v, %v", findings, err)
	}
}

func TestHotPathProofOutcomes(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "tooling", "project-policy", "demo", "program.concept")
	for _, test := range []struct {
		name     string
		outcome  SemanticFactCertainty
		evidence string
	}{
		{"FastFunction", FactProven, "no allocating operation"},
		{"AllocatingFunction", FactDisproven, "Acquire Allocates"},
		{"ForeignOrIncompleteFunction", FactUnknown, "no compiler-known allocation summary"},
	} {
		graph, err := ExplainPolicy(path, "HotPathPolicy", test.name, nil)
		if err != nil {
			t.Fatal(err)
		}
		if graph.Outcome != test.outcome || !strings.Contains(RenderProofVerbose(graph), test.evidence) {
			t.Fatalf("%s: %+v", test.name, graph)
		}
		baseline, err := json.Marshal(graph)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 100; i++ {
			again, err := ExplainPolicy(path, "HotPathPolicy", test.name, nil)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(again)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, baseline) {
				t.Fatalf("%s proof drifted on pass %d", test.name, i)
			}
		}
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse("policy_did_not_create_truth.concept", string(body)+`
void Verify() { Assert.Concept<NoAllocation>(AllocatingFunction, "policy cannot grant NoAllocation"); }
`)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
		t.Fatalf("policy upgraded a disproven fact: %v", err)
	}
}
