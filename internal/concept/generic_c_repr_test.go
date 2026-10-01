package concept

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestR9aGenericCReprClosedArtifactAndNative(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "valid", "generic_c_repr.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := buildSemanticArtifact(t, "cpair.concept", string(source), nil)
	consumer := "module Closure.CUse;\nprofile Core;\nimport Closure.CPair;\nint Main() { Pair<int> pair = Make(); return pair.first + pair.second; }\n"
	module, err := ParseWithSemanticModules("cpair_use.concept", consumer, map[string][]byte{"Closure.CPair": body})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "cpair_harness.c", "#include \"cpair_use.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Main")+"() == 3 ? 0 : 1; }\n")
}

func TestR9aGenericCReprRejectsDestruction(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "generic-library-closure", "invalid", "generic_c_repr_drop.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(path, string(source))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "C_ABI_REPR_INVALID" {
		t.Fatalf("C destruction admitted: %v", err)
	}
}
