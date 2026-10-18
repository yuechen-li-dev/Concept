package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDottedModuleSourceLayout(t *testing.T) {
	root := t.TempDir()
	write := func(relative, source string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	base := `module Library.Numbers.Base; profile Core; int Seven(){return 7;}`
	write("Library/Numbers.Base.concept", base)
	write("Library/Numbers.More.concept", `module Library.Numbers.More; profile Core; import Library.Numbers.Base; int Twice(){return Seven()+Seven();}`)
	consumer := `module App; profile Core; import Library.Numbers.More; int Main(){return Twice();}`
	if _, err := ParseWithBuiltSemanticModuleRoots("app.concept", consumer, []string{root}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := BuildSemanticModuleArtifactsFromSources([]string{root}, []string{"Library.Numbers.More"})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("built %d modules", len(artifacts))
	}
	// Artifacts retain their canonical path and include the dotted source hash.
	artifactPath := filepath.Join(root, "Library", "Numbers", "Base.concept-module.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, artifacts["Library.Numbers.Base"], 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSemanticModuleArtifacts([]string{root}, []string{"Library.Numbers.Base"}); err != nil {
		t.Fatal(err)
	}
	write("Library/Numbers.Base.concept", base+"\n// changed")
	if _, err := ResolveSemanticModuleArtifacts([]string{root}, []string{"Library.Numbers.Base"}); err == nil || !strings.Contains(err.Error(), "MODULE_ARTIFACT_STALE") {
		t.Fatalf("stale dotted source: %v", err)
	}
	write("Library/Numbers.Base.concept", base)
	write("Library/Numbers/Base.concept", base)
	if _, err := BuildSemanticModuleArtifactsFromSources([]string{root}, []string{"Library.Numbers.Base"}); err == nil || !strings.Contains(err.Error(), "MODULE_IDENTITY_DUPLICATE") {
		t.Fatalf("ambiguous source layouts: %v", err)
	}
	if _, err := ResolveSemanticModuleArtifacts([]string{root}, []string{"Library.Numbers.Base"}); err == nil || !strings.Contains(err.Error(), "MODULE_IDENTITY_DUPLICATE") {
		t.Fatalf("ambiguous artifact source layouts: %v", err)
	}
}
