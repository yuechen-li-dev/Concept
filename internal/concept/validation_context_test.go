package concept

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestR9aComptimeContextAcrossRevalidation(t *testing.T) {
	for _, name := range []string{"comptime_context", "runtime_comptime_index"} {
		t.Run(name, func(t *testing.T) {
			kind := "valid"
			if name == "runtime_comptime_index" {
				kind = "invalid"
			}
			path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", kind, name+".concept")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(path, string(source))
			if kind == "invalid" {
				var diagnostic Diagnostic
				if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4210" {
					t.Fatalf("runtime validation lost its boundary: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Generate(module, source); err != nil {
				t.Fatal(err)
			}
			if _, err := CompileSemanticModule(path, string(source), nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestR9aExpressionContextRestoresRuntimeScope(t *testing.T) {
	profile, _ := evt1ProfileDefinition("Core")
	env := newSemanticEnv(profile)
	scope := newEVT1Scope(nil)
	child := newEVT1Scope(scope)
	restore := evt1EnterValidationContext(child, nil, true)
	if !evt1ValidationIsComptime(child) || evt1ValidationIsComptime(scope) {
		t.Fatal("context did not stay local")
	}
	if !evt1ValidationIsComptime(newEVT1Scope(child)) {
		t.Fatal("child scope lost context")
	}
	restore()
	if evt1ValidationIsComptime(child) {
		t.Fatal("context leaked into runtime scope")
	}
	_, _ = validateExpr(env, child, &IntLiteral{Magnitude: 1}, nil, true)
	if evt1ValidationIsComptime(child) {
		t.Fatal("expression context leaked")
	}
}
