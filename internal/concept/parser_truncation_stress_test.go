package concept

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoundedParserTruncationStress(t *testing.T) {
	paths := []string{
		"libraries/Golden/Embedded/Serial.concept",  // bits, MMIO, payload enum, range
		"libraries/Golden/Compiler/Postfix.concept", // nested generics, arrays, payload AST
		"libraries/Golden/Game/Agents.concept",      // machine and yield
		"libraries/Standard/Machine/AMD64.concept",  // machine intrinsics and asm
		"libraries/Standard/Octagon/Derive.concept", // generated declarations
		"language/evt1/generic-requirements/valid/nested_diamond.concept",
		"tests/goldens/companion/manifest.concept", // native manifest grammar
	}
	for _, name := range paths {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", filepath.FromSlash(name))
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parseSyntaxModule(name, string(body)); err != nil {
				t.Fatalf("intact syntax rejected: %v", err)
			}
			for cut := 0; cut < 100; cut++ {
				end := len(body) * cut / 100
				// Every truncated token stream may fail, but none may panic or hang.
				_, _ = parseSyntaxModule(name, string(body[:end]))
			}
		})
	}
}
