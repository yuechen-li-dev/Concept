package concept

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var ffiSpanInvalidFixtures = map[string]string{
	"as_bytes_not_span.concept":           "SPAN_AS_BYTES_INVALID",
	"as_bytes_non_abi_element.concept":    "SPAN_AS_BYTES_INVALID",
	"as_bytes_mutable_handle.concept":     "SPAN_AS_BYTES_INVALID",
	"as_bytes_template_non_abi.concept":   "SPAN_AS_BYTES_INVALID",
	"extern_span_return.concept":          "EXTERN_C_ABI_TYPE_INVALID",
	"extern_span_non_abi_element.concept": "EXTERN_C_ABI_TYPE_INVALID",
}

func TestFFISpanDiagnostics(t *testing.T) {
	for file, code := range ffiSpanInvalidFixtures {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "ffi-spans", "invalid", file)
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

func generateFFISpanFixture(t *testing.T, name string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "ffi-spans", "valid", name)
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
	return outputs
}

// The companion C declares its own layout-identical span structs: a span
// crosses by value as { data, length } with length counted in elements.
func TestFFISpansCrossTheCBoundary(t *testing.T) {
	outputs := generateFFISpanFixture(t, "span_bytes_roundtrip.concept")
	header := string(outputs["span_bytes_roundtrip.generated.h"])
	for _, want := range []string{"size_t CountBytes(concept_readonly_span_byte bytes);", "void Scale(concept_span_float values, float factor);"} {
		if !strings.Contains(header, want) {
			t.Fatalf("header missing %q:\n%s", want, header)
		}
	}
	harness := `#include "span_bytes_roundtrip.generated.h"
#include <stddef.h>
#include <stdint.h>
size_t CountBytes(concept_readonly_span_byte bytes) { return bytes.length; }
void Scale(concept_span_float values, float factor) {
  for (size_t i = 0; i < values.length; ++i) values.data[i] *= factor;
}
int main(void) {
  return concept_span_bytes_roundtrip_main() == 8.0f ? 0 : 1;
}
`
	runFoundationNativeHarness(t, outputs, "span_bytes_roundtrip_harness.c", harness)
}

func TestLayoutPushBytesMatchDeclaredOffsets(t *testing.T) {
	outputs := generateFFISpanFixture(t, "layout_push_bytes.concept")
	harness := `#include "layout_push_bytes.generated.h"
#include <stdint.h>
#include <string.h>
static float seen_scale = 0.0f;
uint32_t ReadCount(concept_readonly_span_byte bytes) {
  uint32_t count = 0;
  if (bytes.length != 8) return 0;
  memcpy(&count, bytes.data, 4);
  memcpy(&seen_scale, bytes.data + 4, 4);
  return count;
}
int main(void) {
  if (concept_layout_push_bytes_main() != 64u) return 1;
  return seen_scale == 2.0f ? 0 : 2;
}
`
	runFoundationNativeHarness(t, outputs, "layout_push_bytes_harness.c", harness)
}
