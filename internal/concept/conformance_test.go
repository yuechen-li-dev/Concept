package concept

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type ConformanceExpectation string

const (
	ConformancePass               ConformanceExpectation = "PASS"
	ConformanceExpectedDivergence ConformanceExpectation = "EXPECTED-DIVERGENCE"
	ConformanceEVT1Gap            ConformanceExpectation = "EVT1-GAP"
	ConformancePoC3Legacy         ConformanceExpectation = "POC3-LEGACY"
	ConformanceSpecAmbiguity      ConformanceExpectation = "SPEC-AMBIGUITY"
)

type ConformanceCase struct {
	Name               string
	Area               string
	Source             string
	Expected           ConformanceExpectation
	MatrixStatus       string
	Reference          string
	Notes              string
	Accepted           bool
	DiagnosticCategory string
	Oracle             string
}

var evt1R1ConformanceCases = []ConformanceCase{
	{Name: "struct declaration construction field copy", Area: "structs", Source: "valid/structs.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase7-runtime-structs/valid/struct_literal_basic.valid.conception", Notes: "Positional translation of PoC3 named aggregate evidence.", Accepted: true, Oracle: "struct"},
	{Name: "payload enum construction and exhaustive match", Area: "payload enums and match", Source: "valid/payload_enums_match.concept", Expected: ConformancePass, MatrixStatus: "Merge / Keep Go", Reference: "language/phase5-sum-types/valid/enum_match_payload_bind_two.valid.conception", Notes: "Qualified variants and EVT1 expression-match spelling.", Accepted: true, Oracle: "enum"},
	{Name: "runtime while and if expression", Area: "control flow", Source: "valid/control_flow.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase2-execution/valid/while_with_if.valid.conception", Notes: "Canonical EVT1 translation.", Accepted: true, Oracle: "control"},
	{Name: "concept closure and template identities", Area: "concepts and templates", Source: "valid/concepts_templates.concept", Expected: ConformancePass, MatrixStatus: "Merge / Keep Go", Reference: "language/phase8-concepts-templates/valid/template_constrained_equal.valid.conception", Notes: "Explicit assertions replace PoC3 impl spelling.", Accepted: true, Oracle: "template"},
	{Name: "bounded comptime and fixed arrays", Area: "comptime and arrays", Source: "valid/comptime_arrays.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase9-compile-time/valid/static_assert_compile_time_while.valid.conception", Notes: "Also references phase21 array literal and indexing evidence.", Accepted: true, Oracle: "comptime"},
	{Name: "struct initializer mismatch", Area: "structs", Source: "invalid/struct_type_mismatch.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase7-runtime-structs/invalid/struct_literal_type_mismatch.invalid.conception", Notes: "Both lines reject the value-type mismatch.", DiagnosticCategory: "CV4107"},
	{Name: "enum payload mismatch", Area: "payload enums", Source: "invalid/enum_wrong_payload.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase5-sum-types/invalid/enum_constructor_payload_type_mismatch.invalid.conception", Notes: "Both lines reject the payload type mismatch.", DiagnosticCategory: "CV4107"},
	{Name: "missing match variant", Area: "match", Source: "invalid/match_missing_variant.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase5-sum-types/valid/enum_match_payload_bind_two.valid.conception", Notes: "Matrix exhaustiveness law supplies the negative oracle.", DiagnosticCategory: "CV4115"},
	{Name: "duplicate match variant", Area: "match", Source: "invalid/match_duplicate_variant.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase5-sum-types/invalid/enum_match_duplicate_variant.invalid.conception", Notes: "Diagnostic numbers intentionally differ across compiler lines.", DiagnosticCategory: "CV4113"},
	{Name: "payload binding result type", Area: "match", Source: "invalid/match_payload_binding_type.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase5-sum-types/valid/enum_match_payload_bind_int.valid.conception", Notes: "The bound payload is proved int by result unification.", DiagnosticCategory: "CV4116"},
	{Name: "match expression result mismatch", Area: "match", Source: "invalid/match_result_type.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase3-semantics/invalid/hir_check_return_type_mismatch.invalid.conception", Notes: "Semantic type category, not numeric code, is the cross-line oracle.", DiagnosticCategory: "CV4116"},
	{Name: "explicit concept assertion missing operation", Area: "concepts", Source: "invalid/concept_missing_requirement.concept", Expected: ConformancePass, MatrixStatus: "Merge / Keep Go", Reference: "language/phase8-concepts-templates/invalid/impl_missing_requirement.invalid.conception", Notes: "Translated to explicit requires Concept<Type> spelling.", DiagnosticCategory: "CV4153"},
	{Name: "template constraint unsatisfied", Area: "templates", Source: "invalid/template_unsatisfied.concept", Expected: ConformancePass, MatrixStatus: "Merge / Keep Go", Reference: "language/phase8-concepts-templates/invalid/constrained_missing_impl.invalid.conception", Notes: "No feature port is implied.", DiagnosticCategory: "CV4153"},
	{Name: "unbounded comptime loop", Area: "comptime", Source: "invalid/comptime_unbounded_loop.concept", Expected: ConformanceExpectedDivergence, MatrixStatus: "Merge", Reference: "language/phase9-compile-time/valid/compile_time_function_while_sum_run.valid.conception", Notes: "PoC3 accepts its fuel-bounded spelling; canonical EVT1 additionally requires bounded(limit).", DiagnosticCategory: "CV4205"},
	{Name: "comptime runtime call", Area: "comptime", Source: "invalid/comptime_runtime_call.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase9-compile-time/invalid/compile_time_function_runtime_call.invalid.conception", Notes: "Both lines reject compile-time side effects through runtime calls.", DiagnosticCategory: "CV4210"},
	{Name: "compile-time array bounds", Area: "compile-time arrays", Source: "invalid/array_out_of_range.concept", Expected: ConformancePass, MatrixStatus: "Merge", Reference: "language/phase21-arrays-slices-fixed-buffers/invalid/array_index_constant_high.invalid.conception", Notes: "Runtime-array behavior remains excluded.", DiagnosticCategory: "CV4233"},
	{Name: "Core rejects Vulkan runtime type", Area: "profile isolation", Source: "invalid/profile_vulkan_type.concept", Expected: ConformancePass, MatrixStatus: "Profile-only", Reference: "EVT1 reconciliation matrix: primitive types", Notes: "Type is absent from the Core profile registry.", DiagnosticCategory: "CV4102"},
	{Name: "Core rejects Vulkan import", Area: "profile isolation", Source: "invalid/profile_vulkan_import.concept", Expected: ConformancePass, MatrixStatus: "Profile-only", Reference: "EVT1 reconciliation matrix: imports", Notes: "Core admits no domain imports.", DiagnosticCategory: "CV4401"},
	{Name: "Core rejects effect", Area: "profile isolation", Source: "invalid/profile_effect.concept", Expected: ConformancePass, MatrixStatus: "Profile-only", Reference: "EVT1 reconciliation matrix: effects", Notes: "Effect admission is profile-owned.", DiagnosticCategory: "CV4402"},
	{Name: "Core rejects actuator", Area: "profile isolation", Source: "invalid/profile_actuator.concept", Expected: ConformancePass, MatrixStatus: "Profile-only", Reference: "EVT1 reconciliation matrix: actuators", Notes: "Actuator admission is profile-owned.", DiagnosticCategory: "CV4402"},
	{Name: "Core rejects undeclared mechanism call", Area: "profile isolation", Source: "invalid/profile_mechanism_call.concept", Expected: ConformancePass, MatrixStatus: "Profile-only", Reference: "CONCEPT Vulkan constitution: imported mechanism boundary", Notes: "Vulkan mechanism operations are explicit declarations, not Core builtins.", DiagnosticCategory: "CV4027"},
}

func TestEVT1R1DifferentialConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R1ConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Area == "" || tc.Reference == "" || tc.MatrixStatus == "" || tc.Notes == "" {
				t.Fatal("conformance metadata is incomplete")
			}
			path := filepath.Join("..", "..", "language", "evt1-r1", "core", filepath.FromSlash(tc.Source))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if !tc.Accepted {
				if err == nil {
					t.Fatal("EVT1 accepted a case whose semantic oracle requires rejection")
				}
				var diagnostic Diagnostic
				if !errors.As(err, &diagnostic) {
					t.Fatalf("rejection was not a stable diagnostic: %v", err)
				}
				if diagnostic.Code != tc.DiagnosticCategory {
					t.Fatalf("diagnostic category = %s, want %s: %v", diagnostic.Code, tc.DiagnosticCategory, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, source)
			if err != nil {
				t.Fatal(err)
			}
			assertConformanceOracle(t, tc.Oracle, outputs)
		})
		counts[tc.Expected]++
	}
	t.Logf("R1 conformance summary: PASS=%d EXPECTED-DIVERGENCE=%d EVT1-GAP=%d POC3-LEGACY=%d SPEC-AMBIGUITY=%d",
		counts[ConformancePass], counts[ConformanceExpectedDivergence], counts[ConformanceEVT1Gap], counts[ConformancePoC3Legacy], counts[ConformanceSpecAmbiguity])
}

func assertConformanceOracle(t *testing.T, oracle string, outputs Outputs) {
	t.Helper()
	var mirBody []byte
	var cBody []byte
	for name, body := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			mirBody = body
		}
		if strings.HasSuffix(name, ".generated.c") {
			cBody = body
		}
	}
	var mir MIR
	if err := json.Unmarshal(mirBody, &mir); err != nil {
		t.Fatal(err)
	}
	switch oracle {
	case "struct":
		if len(mir.Structs) != 1 || len(mir.Structs[0].Fields) != 2 {
			t.Fatalf("unexpected struct MIR: %+v", mir.Structs)
		}
	case "enum":
		if len(mir.Enums) != 1 || len(mir.Enums[0].Variants) != 3 {
			t.Fatalf("unexpected enum MIR: %+v", mir.Enums)
		}
	case "control":
		if !strings.Contains(string(cBody), "while (") {
			t.Fatal("runtime while was not preserved in generated C")
		}
	case "template":
		if len(mir.Assertions) != 2 || len(mir.Instances) != 2 {
			t.Fatalf("expected two assertions and two deduplicated concrete instances, got %d and %d", len(mir.Assertions), len(mir.Instances))
		}
		if mir.Instances[0].ID == mir.Instances[1].ID {
			t.Fatal("distinct concrete template types share an identity")
		}
	case "comptime":
		if len(mir.ComptimeDecls) != 2 || len(mir.StaticAsserts) != 5 {
			t.Fatalf("unexpected comptime MIR counts: declarations=%d assertions=%d", len(mir.ComptimeDecls), len(mir.StaticAsserts))
		}
	default:
		t.Fatalf("unknown conformance oracle %q", oracle)
	}
}

func TestEVT1R1ConformanceClassificationCounts(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R1ConformanceCases {
		counts[tc.Expected]++
	}
	if counts[ConformancePass] != 20 || counts[ConformanceExpectedDivergence] != 1 || len(counts) != 2 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestEVT1R1CoreStructsNativeC11(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1-r1", "core", "valid", "structs.concept")
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
	harness := `#include "structs.generated.h"

int main(void) {
  return concept_structs_copy_and_read() == 7 ? 0 : 1;
}
`
	runR1NativeHarness(t, outputs, "evt1_r1_structs_harness.c", harness)
}

func runR1NativeHarness(t *testing.T, outputs Outputs, harnessName, harnessSource string) {
	t.Helper()
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Skip("gcc and clang are unavailable")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harnessPath := filepath.Join(dir, harnessName)
	if err := os.WriteFile(harnessPath, []byte(harnessSource), 0644); err != nil {
		t.Fatal(err)
	}
	var generatedC string
	for key := range outputs {
		if strings.HasSuffix(key, ".generated.c") {
			generatedC = filepath.Join(dir, key)
			break
		}
	}
	if generatedC == "" {
		t.Fatal("generated C output missing")
	}
	executable := filepath.Join(dir, "evt1-r1-native.exe")
	build := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generatedC, harnessPath, "-o", executable)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("native C11 compile failed: %v\n%s", err, out)
	}
	if out, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("native C11 specimen failed: %v\n%s", err, out)
	}
}
