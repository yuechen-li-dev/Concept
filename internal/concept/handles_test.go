package concept

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var handleInvalidFixtures = map[string]string{
	"handle_arithmetic.concept":      "HANDLE_OPERATION_INVALID",
	"handle_ordering.concept":        "HANDLE_OPERATION_INVALID",
	"handle_mixed_equality.concept":  "HANDLE_OPERATION_INVALID",
	"handle_cast.concept":            "NUMERIC_CAST_REQUIRED",
	"handle_construct.concept":       "CV4125",
	"handle_duplicate_type.concept":  "CV4580",
	"handle_integer_literal.concept": "CV4116",
}

func TestHandleDiagnostics(t *testing.T) {
	for file, code := range handleInvalidFixtures {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "handles", "invalid", file)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != code {
				t.Fatalf("diagnostic = %v, want %s", err, code)
			}
		})
	}
}

// The generated typedef and a foreign header's `typedef struct Buffer_T*
// Buffer;` name the same C type, so the companion C below implements the
// extern functions with its own spelling and both are in scope together.
func TestHandleRoundTripsThroughCompanionC(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "handles", "valid", "handle_roundtrip.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["handle_roundtrip.generated.h"])
	if !strings.Contains(header, "typedef struct Buffer_T* concept_buffer;") {
		t.Fatalf("handle typedef missing:\n%s", header)
	}
	harness := `#include "handle_roundtrip.generated.h"
#include <stddef.h>
#include <stdint.h>
typedef struct Buffer_T* Buffer;
static unsigned char storage[2];
static int created = 0;
static int destroyed = 0;
Buffer CreateBuffer(int32_t size) { (void)size; return (Buffer)(void*)&storage[created++]; }
void DestroyBuffer(Buffer buffer) { if (buffer) destroyed++; }
int main(void) {
  _Static_assert(sizeof(concept_buffer) == sizeof(void*), "a handle is one pointer");
  _Static_assert(offsetof(concept_binding, offset) == sizeof(void*), "repr(C) places the handle first");
  if (concept_handle_roundtrip_main() != 111) return 1;
  if (created != 2 || destroyed != 2) return 2;
  return 0;
}
`
	runFoundationNativeHarness(t, outputs, "handle_roundtrip_harness.c", harness)
}
