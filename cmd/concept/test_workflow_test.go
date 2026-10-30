package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAMD64BatchRejectsSourcesBeforeBootstrap(t *testing.T) {
	// Compiler-boundary specimens, not runtime expectations. No C toolchain is
	// available: invalid source must fail before the shared backend is built.
	t.Setenv("PATH", "")
	valid := filepath.Join(t.TempDir(), "valid.concept")
	invalid := filepath.Join(t.TempDir(), "invalid.concept")
	if err := os.WriteFile(valid, []byte("profile Core; int Main(){return 1;}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("profile Core; int Main(){return Unknown;}"), 0600); err != nil {
		t.Fatal(err)
	}
	err := printConceptAMD64Batch([]string{valid, invalid})
	if err == nil || !strings.Contains(err.Error(), "Unknown") || strings.Contains(err.Error(), "BACKEND") {
		t.Fatalf("expected source diagnostic before any bootstrap: %v", err)
	}
	err = printConceptAMD64Batch([]string{valid, "../../tests/runtime/Testing/TaggedValues.concept"})
	if err == nil || !strings.Contains(err.Error(), "EVT2H2_TAGGED_TRANSPORT_REQUIRED") {
		t.Fatalf("expected tagged transport stop before any bootstrap: %v", err)
	}
}

// A new native CLI scenario goes in the existing batch, not in another cold
// go-run invocation. Keep exactly one cold single-input smoke for its real path.
func TestNativeCLICasesUseBatch(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if fn.Name.Name == "TestNativeCLICasesUseBatch" || fn.Name.Name == "TestAMD64CLIBatch" || fn.Name.Name == "TestAMD64BootstrapLoadsBackendImports" {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value, _ := strconv.Unquote(lit.Value)
					if value == "amd64" {
						t.Errorf("%s: add the native CLI case to TestAMD64CLIBatch instead of rebuilding Cathedral", fn.Name.Name)
					}
				}
				return true
			})
		}
	}
}
