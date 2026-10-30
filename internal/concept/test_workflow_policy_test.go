package concept

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Compiler inspection and runtime execution share one source. This guards the
// migration against reintroducing a second embedded copy or C expectation.
func TestRuntimeWorkflowPolicy(t *testing.T) {
	owners := map[string]string{
		"TestContextualArrayLiteralLowering":               "contextual_arrays.concept_test",
		"TestContextualFloatLiterals":                      "contextual_floats.concept_test",
		"TestPayloadFreeEnumGeometry":                      "enum_geometry.concept_test",
		"TestBorrowedResultErrorRemap":                     "error_remap.concept_test",
		"TestIntegerMatchContextAndExecution":              "integer_match.concept_test",
		"TestRangeMIRComparisonAndDeterminism":             "range_compare.concept_test",
		"TestValueParameterSnapshotGeneratedNameCollision": "value_names.concept_test",
	}
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, path := range files {
		file, err := goparser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			fixture, owned := owners[fn.Name.Name]
			if !owned {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					if name, ok := call.Fun.(*ast.Ident); ok {
						if name.Name == "runFoundationNativeHarness" || name.Name == "runFoundationNativeHarnessOutput" {
							t.Errorf("%s: runtime assertions belong in %s", fn.Name.Name, fixture)
						}
						if name.Name == "runtimeTestSource" && len(call.Args) == 2 {
							if lit, ok := call.Args[1].(*ast.BasicLit); ok {
								value, _ := strconv.Unquote(lit.Value)
								found[fn.Name.Name] = value == fixture
							}
						}
					}
				}
				return true
			})
		}
	}
	for name, fixture := range owners {
		if !found[name] {
			t.Errorf("%s must inspect the authoritative %s", name, fixture)
		}
	}
	// Direct imports are checked before semantic discovery can load a large backend.
	paths, err := filepath.Glob("../../tests/runtime/*.concept_test")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		imports, err := SemanticModuleImports(path, string(body))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range imports {
			if strings.HasPrefix(name, "Standard.Backend.") {
				t.Errorf("%s imports Cathedral; use tests/backend for backend qualification", path)
			}
		}
	}
}
