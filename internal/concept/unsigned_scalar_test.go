package concept

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func r9aUnsignedSource(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile("../../language/evt1/generic-library-closure/valid/unsigned_scalar.concept")
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestR9aUnsignedScalarNativeAndArtifact(t *testing.T) {
	source := r9aUnsignedSource(t)
	body := buildSemanticArtifact(t, "unsigned.concept", source, nil)
	consumer := "module Closure.UnsignedUse;\nprofile Core;\nimport Closure.Unsigned;\nuint8 Use(uint8 value) { return Complement<uint8>(value); }\n"
	module, err := ParseWithSemanticModules("unsigned_use.concept", consumer, map[string][]byte{"Closure.Unsigned": body})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "use.c", "#include \"unsigned_use.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Use")+"(255) == 0 ? 0 : 1; }\n")
	module, err = Parse("unsigned.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "unsigned.generated.c")
		runFoundationNativeHarness(t, outputs, "unsigned_harness.c", `#include "unsigned.generated.h"
int main(void) {
  if (concept_closure__unsigned_explicit() != 1u) return 1;
  if (concept_closure__unsigned_narrow() != 255u) return 2;
  if (concept_closure__unsigned_maximum() != UINT64_MAX) return 3;
  if (concept_closure__unsigned_bits(0) != 255u) return 4;
  if (concept_closure__unsigned_wide_bits(0) != UINT64_MAX) return 5;
  if (concept_closure__unsigned_signed_bits(0) != -1) return 6;
  if (concept_closure__unsigned_read(2) != 6) return 7;
  if (concept_closure__unsigned_write(1) != 9) return 8;
  if (concept_closure__unsigned_read_span(2) != 6) return 9;
  if (concept_closure__unsigned_read_string(2) != 'c') return 10;
  if (concept_closure__unsigned_known() != 5) return 11;
  if (concept_closure__unsigned_read_tensor(1) != 4) return 12;
  if (concept_closure__unsigned_read_nd(1) != 4) return 13;
  return 0;
}`)
		for _, call := range []string{"read", "read_span", "read_string", "read_tensor", "read_nd"} {
			r9aUnsignedPanic(t, outputs, call, verify)
		}
	}
}

func r9aUnsignedPanic(t *testing.T, outputs Outputs, call string, verify bool) {
	t.Helper()
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Fatal("gcc is required for the R9a unsigned runtime qualification")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(dir, "panic.c")
	body := "#include \"unsigned.generated.h\"\n#include <stdlib.h>\nint main(int argc, char **argv) { (void)argc; return concept_closure__unsigned_" + call + "(strtoull(argv[1], NULL, 10)); }\n"
	if err := os.WriteFile(harness, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "panic.exe")
	if output, err := nativeCommand(t, compiler, withHostLinkArgs("-std=c11", "-pedantic-errors", "-I", dir, filepath.Join(dir, "unsigned.generated.c"), harness, "-o", executable)...).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	for _, index := range []string{"4294967297", "18446744073709551615"} {
		output, err := nativeCommand(t, executable, index).CombinedOutput()
		if err == nil || !strings.Contains(string(output), "out of bounds") {
			t.Fatalf("wide index %s wrapped or failed without bounds evidence: %v\n%s", index, err, output)
		}
		if verify && (call == "read" || call == "read_span" || call == "read_nd") && !strings.Contains(string(output), "index="+index) {
			t.Fatalf("Verify narrowed the reported unsigned index: %s", output)
		}
	}
}

func TestR9aUnsignedDiagnostics(t *testing.T) {
	for _, tc := range []struct{ file, code string }{
		{"unsigned_negative", "CV4644"}, {"unsigned_signed_target", "CV4644"}, {"complement_bool", "CV4028"}, {"unsigned_static_index", "CV4233"},
		{"generic_round_bad_source", "NUMERIC_ROUND_SOURCE"}, {"generic_round_bad_target", "NUMERIC_ROUND_TARGET"},
	} {
		source, err := os.ReadFile("../../language/evt1/generic-library-closure/invalid/" + tc.file + ".concept")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(tc.file+".concept", string(source))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
			t.Fatalf("%s: expected %s, got %v", tc.file, tc.code, err)
		}
	}
	for _, scalar := range []string{"int", "uint", "uint8", "uint16", "uint32", "uint64", "usize", "isize", "byte"} {
		_, err := Parse("index.concept", "profile Core;\nint Read("+scalar+" index) { int<array>[2] values = [1,2]; return values[index]; }\n")
		if err != nil {
			t.Fatalf("%s index: %v", scalar, err)
		}
	}
}
