package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSMLibModuleRootsArePhysicalAndIdentityChecked(t *testing.T) {
	root := t.TempDir()
	source := "module Nested.Value; profile Core; int Read(){return 5;}"
	path := filepath.Join(root, "Nested", "Value.concept")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	aliases := []string{root, filepath.Join(root, "."), root}
	built, err := BuildSemanticModuleArtifactsFromSources(aliases, []string{"Nested.Value"})
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "Nested", "Value.concept-module.json")
	if err := os.WriteFile(artifactPath, built["Nested.Value"], 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSemanticModuleArtifacts(aliases, []string{"Nested.Value"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Nested.Missing", "Incorrect.Value"} {
		if _, err := BuildSemanticModuleArtifactsFromSources(aliases, []string{name}); err == nil || !strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	wrong, err := CompileSemanticModule("Other.concept", "module Other; profile Core; int Read(){return 5;}", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, wrong, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSemanticModuleArtifacts(aliases, []string{"Nested.Value"}); err == nil || !strings.Contains(err.Error(), "MODULE_IDENTITY_MISMATCH") {
		t.Fatalf("wrong artifact identity: %v", err)
	}
}
