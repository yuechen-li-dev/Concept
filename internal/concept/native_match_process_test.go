package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEVT2h3ArtifactProcessWorker(t *testing.T) {
	mode := os.Getenv("CONCEPT_H3_ARTIFACT_WORKER")
	if mode == "" {
		t.Skip("subprocess-only artifact order worker")
	}
	if mode == "warm" {
		path := "../../tests/runtime/Testing/OrdinaryMatches.concept"
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := encodeSemanticModule(m); err != nil {
			t.Fatal(err)
		}
	}
	_, _, _, m, data := patternFixture(t, false)
	dir := os.Getenv("CONCEPT_H3_ARTIFACT_OUTPUT")
	if err := os.WriteFile(filepath.Join(dir, "match.cmir9"), data, 0600); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(taggedFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := CompileSemanticModule(taggedFixturePath, string(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, ast, err := LoadSemanticModuleArtifact(body)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(ast)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "semantic-ast.json"), canonical, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "semantic.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	projection, err := json.MarshalIndent(m.Pattern, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "machine.json"), projection, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestEVT2h3ArtifactProcessDeterminism(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var first, ast, semantic []byte
	for run := 0; run < 100; run++ {
		mode := "cold"
		if run%2 != 0 {
			mode = "warm"
		}
		dir := t.TempDir()
		command := exec.Command(executable, "-test.run=^TestEVT2h3ArtifactProcessWorker$", "-test.timeout=30s")
		command.Env = append(os.Environ(), "CONCEPT_H3_ARTIFACT_WORKER="+mode, "CONCEPT_H3_ARTIFACT_OUTPUT="+dir)
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("worker %s: %v\n%s", mode, err, out)
		}
		next, err := os.ReadFile(filepath.Join(dir, "match.cmir9"))
		if err != nil {
			t.Fatal(err)
		}
		nextAST, err := os.ReadFile(filepath.Join(dir, "semantic-ast.json"))
		if err != nil {
			t.Fatal(err)
		}
		nextSemantic, err := os.ReadFile(filepath.Join(dir, "semantic.json"))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first, ast, semantic = next, nextAST, nextSemantic
		} else if !bytes.Equal(first, next) {
			t.Fatalf("artifact depends on prior AST encoding: %s (typed AST equal=%t; semantic envelope equal=%t); evidence=%s", fmt.Sprint(mode), bytes.Equal(ast, nextAST), bytes.Equal(semantic, nextSemantic), dir)
		}
	}
}
