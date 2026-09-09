package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var storageBindingConformanceCases = []ConformanceCase{
	{Name: "fixed array to ndarray", Source: "valid/bind_array_to_ndarray_fixed.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d storage plus R4a provenance", MatrixStatus: "EVT1-new", Notes: "Fixed total count is proven at compile time."},
	{Name: "runtime target shape", Source: "valid/bind_array_to_ndarray_runtime_shape.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d runtime shape", MatrixStatus: "EVT1-new", Notes: "Runtime product is overflow-safe and exact."},
	{Name: "ndarray reshape", Source: "valid/bind_ndarray_reshape.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d row-major storage", MatrixStatus: "EVT1-new", Notes: "Linear order is unchanged."},
	{Name: "ndarray flatten", Source: "valid/bind_ndarray_flatten.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d contiguous storage", MatrixStatus: "EVT1-new", Notes: "Rank may decrease without copying."},
	{Name: "mutable alias", Source: "valid/bind_mutable_alias.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 mutable ref", MatrixStatus: "EVT1-new composition", Notes: "Both access paths observe the same elements."},
	{Name: "readonly bind", Source: "valid/bind_ref_const.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 ref const", MatrixStatus: "EVT1-new composition", Notes: "Const source remains readonly."},
	{Name: "bound rank and shape", Source: "valid/bind_rank_shape_query.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d Rank/Shape", MatrixStatus: "EVT1-new composition", Notes: "Queries use descriptor shape values."},
	{Name: "scoped provenance", Source: "valid/bind_scoped_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4a scoped provenance", MatrixStatus: "EVT1-new composition", Notes: "Local use preserves scoped provenance."},
	{Name: "call result provenance", Source: "valid/bind_call_result_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4b call-result provenance", MatrixStatus: "EVT1-new composition", Notes: "Helper result remains parameter-derived."},
	{Name: "runtime view rebind", Source: "valid/bind_runtime_view_rebind.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e descriptor closure", MatrixStatus: "EVT1-new", Notes: "A prior bind remains a valid whole contiguous source."},
	{Name: "bound ref struct", Source: "valid/bind_ref_struct.concept", Expected: ConformancePass, Accepted: true, Reference: "R4a ref struct", MatrixStatus: "EVT1-new composition", Notes: "A ref struct retains the bound view provenance."},
	{Name: "bind after question", Source: "valid/bind_after_question.concept", Expected: ConformancePass, Accepted: true, Reference: "R4c Result propagation", MatrixStatus: "EVT1-new composition", Notes: "Binding after successful extraction adds no failure channel."},
	{Name: "fixed size mismatch", Source: "invalid/bind_fixed_size_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4566", Reference: "R4e exact whole-storage law", MatrixStatus: "EVT1-new", Notes: "Provable mismatch emits no runtime guard."},
	{Name: "runtime size mismatch", Source: "invalid/bind_runtime_size_mismatch.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e asserted bind", MatrixStatus: "EVT1-new runtime-invalid", Notes: "Compilation succeeds; native execution must panic."},
	{Name: "element mismatch", Source: "invalid/bind_element_type_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4565", Reference: "R4e exact element identity", MatrixStatus: "EVT1-new", Notes: "No reinterpretation or conversion."},
	{Name: "mutable from const", Source: "invalid/bind_mutable_from_const.concept", Expected: ConformancePass, DiagnosticCategory: "CV4567", Reference: "R3 const ref", MatrixStatus: "EVT1-new composition", Notes: "Bind cannot add mutability."},
	{Name: "bound readonly mutation", Source: "invalid/bind_ref_const_mutation.concept", Expected: ConformancePass, DiagnosticCategory: "CV4513", Reference: "R3 ref const", MatrixStatus: "EVT1-new composition", Notes: "Indexed writes through a const bound view remain rejected."},
	{Name: "nonstorage source", Source: "invalid/bind_nonstorage_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4564", Reference: "R4e contiguous source law", MatrixStatus: "EVT1-new", Notes: "Addressability alone is insufficient."},
	{Name: "contextual target required", Source: "invalid/bind_without_target_type.concept", Expected: ConformancePass, DiagnosticCategory: "CV4562", Reference: "R4e destination typing", MatrixStatus: "EVT1-new", Notes: "Shape is never inferred from bind."},
	{Name: "array target required", Source: "invalid/bind_requires_array_target.concept", Expected: ConformancePass, DiagnosticCategory: "CV4563", Reference: "R4e destination typing", MatrixStatus: "EVT1-new", Notes: "The contextual target must be a storage reference."},
	{Name: "local lifetime escape", Source: "invalid/bind_escape_source_lifetime.concept", Expected: ConformancePass, DiagnosticCategory: "CV4511", Reference: "R4a lexical provenance", MatrixStatus: "EVT1-new composition", Notes: "Bind cannot extend local storage lifetime."},
	{Name: "runtime shape overflow", Source: "invalid/bind_shape_product_overflow.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e overflow-safe product", MatrixStatus: "EVT1-new runtime-invalid", Notes: "Compilation succeeds; native execution must panic."},
	{Name: "scoped escape", Source: "invalid/bind_scoped_escape.concept", Expected: ConformancePass, DiagnosticCategory: "CV4522", Reference: "R4a scoped provenance", MatrixStatus: "EVT1-new composition", Notes: "Bind does not erase scoped state."},
}

func TestStorageBindingConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range storageBindingConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "storage", "binding", filepath.FromSlash(tc.Source))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			if tc.Accepted {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.DiagnosticCategory {
				t.Fatalf("diagnostic = %v, want %s", err, tc.DiagnosticCategory)
			}
		})
		counts[tc.Expected]++
	}
	if counts[ConformancePass] != 23 || len(counts) != 1 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestStorageBindingMIRAndDescriptorFacts(t *testing.T) {
	outputs := generateR4eFixture(t, "valid", "bind_array_to_ndarray_runtime_shape.concept")
	var mir MIR
	var header, body string
	for name, artifact := range outputs {
		switch {
		case strings.HasSuffix(name, ".mir.json"):
			if err := json.Unmarshal(artifact, &mir); err != nil {
				t.Fatal(err)
			}
		case strings.HasSuffix(name, ".generated.h"):
			header = string(artifact)
		case strings.HasSuffix(name, ".generated.c"):
			body = string(artifact)
		}
	}
	var bind *MIROperation
	for i := range mir.Functions[0].Operations {
		if mir.Functions[0].Operations[i].Kind == "bind_storage" {
			bind = &mir.Functions[0].Operations[i]
		}
	}
	if bind == nil || bind.SourceStorageKind != StorageArray || bind.TargetStorageKind != StorageNDArray || bind.TargetRank != 2 || bind.CountCheck != "runtime_overflow_safe_equal" || bind.Provenance != "local" || !bind.NoCopy || !bind.NoAllocation || !bind.NoOwnershipTransfer {
		t.Fatalf("missing bind MIR facts: %+v", bind)
	}
	if !strings.Contains(header, "int* data; size_t shape[2]") || !strings.Contains(body, ".data = (storage).data") || !strings.Contains(body, "SIZE_MAX") {
		t.Fatalf("missing non-owning descriptor or checked product\nHEADER:\n%s\nBODY:\n%s", header, body)
	}
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "free(", "stackalloc", "memcpy", "Slice", "Span", "tensor"} {
		if strings.Contains(header+body, forbidden) {
			t.Fatalf("forbidden bind mechanism %q appeared", forbidden)
		}
	}

	fixedOutputs := generateR4eFixture(t, "valid", "bind_array_to_ndarray_fixed.concept")
	for name, artifact := range fixedOutputs {
		switch {
		case strings.HasSuffix(name, ".mir.json"):
			if err := json.Unmarshal(artifact, &mir); err != nil {
				t.Fatal(err)
			}
		case strings.HasSuffix(name, ".generated.c"):
			if strings.Contains(string(artifact), "Concept bind shape does not match storage size") || strings.Contains(string(artifact), "Concept bind shape product overflow") {
				t.Fatalf("fixed bind emitted a runtime shape guard:\n%s", artifact)
			}
		}
	}
	foundFixedProof := false
	for _, operation := range mir.Functions[0].Operations {
		foundFixedProof = foundFixedProof || operation.Kind == "bind_storage" && operation.CountCheck == "comptime_equal"
	}
	if !foundFixedProof {
		t.Fatalf("fixed bind omitted its comptime count proof: %+v", mir.Functions[0].Operations)
	}
}

func TestStorageBindingNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"bind_array_to_ndarray_fixed.concept", "concept_bind_array_to_ndarray_fixed_bind_fixed()", 3},
		{"bind_array_to_ndarray_runtime_shape.concept", "concept_bind_array_to_ndarray_runtime_shape_bind_runtime_shape(2, 3)", 6},
		{"bind_ndarray_reshape.concept", "concept_bind_ndarray_reshape_reshape_ndarray()", 6},
		{"bind_ndarray_flatten.concept", "concept_bind_ndarray_flatten_flatten_ndarray()", 6},
		{"bind_mutable_alias.concept", "concept_bind_mutable_alias_mutable_alias()", 16},
		{"bind_ref_const.concept", "concept_bind_ref_const_read_only_bind()", 7},
		{"bind_rank_shape_query.concept", "concept_bind_rank_shape_query_bound_shape(2, 3)", 13},
		{"bind_scoped_provenance.concept", "concept_bind_scoped_provenance_use_scoped_bind()", 10},
		{"bind_call_result_provenance.concept", "concept_bind_call_result_provenance_use_reshape_call()", 4},
		{"bind_runtime_view_rebind.concept", "concept_bind_runtime_view_rebind_rebind_runtime_view(2, 3)", 12},
		{"bind_ref_struct.concept", "concept_bind_ref_struct_bind_into_ref_struct()", 6},
		{"bind_after_question.concept", "concept_bind_after_question_run_bound_result()", 3},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4eFixture(t, "valid", tc.file)
			harness := "#include \"" + strings.TrimSuffix(tc.file, ".concept") + ".generated.h\"\nint main(void) { return " + tc.call + " == " + fmt.Sprint(tc.want) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, "storage_binding_harness.c", harness)
		})
	}
}

func TestStorageBindingRuntimeBindPanics(t *testing.T) {
	tests := []struct {
		file, call, reason string
	}{
		{"bind_runtime_size_mismatch.concept", "concept_bind_runtime_size_mismatch_runtime_mismatch(2, 2)", "Concept bind shape does not match storage size"},
		{"bind_shape_product_overflow.concept", "concept_bind_shape_product_overflow_runtime_shape_overflow(2147483647, 2147483647, 2147483647)", "Concept bind shape product overflow"},
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4eFixture(t, "invalid", tc.file)
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := filepath.Join(dir, "panic_harness.c")
			if err := os.WriteFile(harness, []byte("#include \""+base+".generated.h\"\nint main(void) { return "+tc.call+"; }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			executable := filepath.Join(dir, "panic.exe")
			generated := filepath.Join(dir, base+".generated.c")
			if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harness, "-o", executable).CombinedOutput(); err != nil {
				t.Fatalf("native C11 compile failed: %v\n%s", err, out)
			}
			out, err := exec.Command(executable).CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.reason) || !strings.Contains(string(out), "Concept panic at") {
				t.Fatalf("bind panic evidence missing: err=%v output=%s", err, out)
			}
		})
	}
}

func TestStorageBindingProvenanceSummaries(t *testing.T) {
	outputs := generateR4eFixture(t, "valid", "bind_call_result_provenance.concept")
	var mir MIR
	for name, artifact := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(artifact, &mir); err != nil {
				t.Fatal(err)
			}
		}
	}
	summary := mir.Functions[0].ResultProvenance
	if summary == nil || summary.Kind != string(evt1ResultProvenanceParameter) || len(summary.ParameterIndices) != 1 || summary.ParameterIndices[0] != 0 {
		t.Fatalf("bind helper lost parameter provenance: %+v", summary)
	}

	scopedOutputs := generateR4eFixture(t, "valid", "bind_scoped_provenance.concept")
	for name, artifact := range scopedOutputs {
		if !strings.HasSuffix(name, ".mir.json") {
			continue
		}
		if err := json.Unmarshal(artifact, &mir); err != nil {
			t.Fatal(err)
		}
	}
	foundScoped := false
	for _, operation := range mir.Functions[0].Operations {
		foundScoped = foundScoped || operation.Kind == "bind_storage" && operation.Provenance == "parameter:scoped"
	}
	if !foundScoped {
		t.Fatalf("scoped bind provenance was not retained: %+v", mir.Functions[0].Operations)
	}
}

func generateR4eFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "storage", "binding", class, file)
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
