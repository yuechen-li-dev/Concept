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

var evt1R4hConformanceCases = []ConformanceCase{
	{Name: "rank-1 ndarray", Source: "valid/tensor_from_ndarray_rank1.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d ndarray", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "rank-2 ndarray", Source: "valid/tensor_from_ndarray_rank2.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d ndarray", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "rank-3 ndarray", Source: "valid/tensor_from_ndarray_rank3.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d ndarray", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "bound ndarray", Source: "valid/tensor_from_bound_ndarray.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e bind", MatrixStatus: "EVT1-new composition"},
	{Name: "layout region", Source: "valid/tensor_from_layout_region.concept", Expected: ConformancePass, Accepted: true, Reference: "R4f layout", MatrixStatus: "EVT1-new composition"},
	{Name: "stream channel", Source: "valid/tensor_from_stream_channel.concept", Expected: ConformancePass, Accepted: true, Reference: "R4f stream", MatrixStatus: "EVT1-new composition"},
	{Name: "rank-1 Span", Source: "valid/tensor_from_span_rank1.concept", Expected: ConformancePass, Accepted: true, Reference: "R4g Span", MatrixStatus: "EVT1-new composition"},
	{Name: "elementwise add", Source: "valid/tensor_elementwise_add.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V tensor pressure", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "elementwise multiply", Source: "valid/tensor_elementwise_mul.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V tensor pressure", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "scalar multiply", Source: "valid/tensor_scalar_mul.concept", Expected: ConformancePass, Accepted: true, Reference: "scalar arithmetic", MatrixStatus: "EVT1-new"},
	{Name: "matrix @", Source: "valid/tensor_matmul_at.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V contraction", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "rank-3 @", Source: "valid/tensor_rank3_at.concept", Expected: ConformancePass, Accepted: true, Reference: "generalized contraction", MatrixStatus: "EVT1-new"},
	{Name: "Einstein matmul", Source: "valid/tensor_einstein_matmul.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V Einstein lineage", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "float Einstein matmul", Source: "valid/tensor_float_einstein.concept", Expected: ConformancePass, Accepted: true, Reference: "numeric tensor algebra", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "Einstein comptime initialize", Source: "valid/tensor_einstein_initialize.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V comptime pressure", MatrixStatus: "EVT1-new / SDSL-V-derived"},
	{Name: "readonly operand", Source: "valid/tensor_readonly_operand.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 const", MatrixStatus: "EVT1-new composition"},
	{Name: "runtime shape guard", Source: "valid/tensor_runtime_shape_guard.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e runtime shape", MatrixStatus: "EVT1-new composition"},
	{Name: "rank mismatch", Source: "invalid/tensor_rank_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4614", MatrixStatus: "EVT1-new"},
	{Name: "shape mismatch", Source: "invalid/tensor_shape_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4617", MatrixStatus: "EVT1-new"},
	{Name: "broadcast rejected", Source: "invalid/tensor_broadcast_rejected.concept", Expected: ConformancePass, DiagnosticCategory: "CV4617", MatrixStatus: "EVT1-new"},
	{Name: "readonly destination", Source: "invalid/tensor_readonly_destination.concept", Expected: ConformancePass, DiagnosticCategory: "CV4616", MatrixStatus: "EVT1-new"},
	{Name: "@ mismatch", Source: "invalid/tensor_at_contraction_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4624", MatrixStatus: "EVT1-new"},
	{Name: "symbolic extent mismatch", Source: "invalid/tensor_symbolic_extent_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4620", MatrixStatus: "EVT1-new"},
	{Name: "bad output index", Source: "invalid/tensor_symbolic_bad_output_index.concept", Expected: ConformancePass, DiagnosticCategory: "CV4621", MatrixStatus: "EVT1-new"},
	{Name: "invalid reduction", Source: "invalid/tensor_symbolic_invalid_reduction.concept", Expected: ConformancePass, DiagnosticCategory: "CV4622", MatrixStatus: "EVT1-new"},
	{Name: "contraction overlap", Source: "invalid/tensor_overlap_contraction_rejected.concept", Expected: ConformancePass, DiagnosticCategory: "CV4625", MatrixStatus: "EVT1-new"},
	{Name: "Span rank-2", Source: "invalid/tensor_span_rank2_without_shape.concept", Expected: ConformancePass, DiagnosticCategory: "CV4614", MatrixStatus: "EVT1-new"},
	{Name: "local tensor escape", Source: "invalid/tensor_escape_local_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4a composition"},
	{Name: "symbolic index escape", Source: "invalid/tensor_symbolic_index_escape.concept", Expected: ConformancePass, DiagnosticCategory: "CV4024", MatrixStatus: "EVT1-new"},
}

func generateR4hFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r4h", "core", class, file)
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

func TestEVT1R4hConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R4hConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1-r4h", "core", filepath.FromSlash(tc.Source))
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
			if diagnostic.SemanticCategory() == diagnostic.Code {
				t.Fatalf("diagnostic %s has no semantic family", diagnostic.Code)
			}
		})
		counts[tc.Expected]++
	}
	if counts[ConformancePass] != 29 || len(counts) != 1 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestEVT1R4hTensorMIRAndLowering(t *testing.T) {
	outputs := generateR4hFixture(t, "valid", "tensor_einstein_matmul.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["tensor_einstein_matmul.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	fn := mir.Functions[0]
	if len(fn.TensorOperations) != 1 {
		t.Fatalf("tensor semantic MIR missing: %+v", fn)
	}
	op := fn.TensorOperations[0]
	if op.Kind != "tensor_contract" || strings.Join(op.FreeIndices, ",") != "i,j" || strings.Join(op.ReduceIndices, ",") != "k" || len(op.Operands) != 2 || op.Output.RegionID != "storage:cStorage" {
		t.Fatalf("invalid TensorContract MIR: %+v", op)
	}
	for _, operand := range append([]MIRTensorOperand{op.Output}, op.Operands...) {
		if operand.BaseOffset == "" || operand.Alignment < 1 || !operand.Contiguous || operand.Provenance == "" || !operand.NoCopy || !operand.NoAllocation || !operand.NoOwnershipTransfer {
			t.Fatalf("TensorContract operand lost storage-neutral facts: %+v", operand)
		}
	}
	views := 0
	for _, operation := range fn.Operations {
		if operation.Kind == "tensor_view" {
			views++
			if !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer || !operation.SameBackingRegion || operation.Provenance == "" || operation.Alignment < 1 {
				t.Fatalf("tensor view lost storage facts: %+v", operation)
			}
		}
	}
	if views != 3 {
		t.Fatalf("got %d tensor views", views)
	}
	body := string(outputs["tensor_einstein_matmul.generated.c"])
	for _, needle := range []string{"for (size_t", "tensor_acc", " += ", ".data["} {
		if !strings.Contains(body, needle) {
			t.Fatalf("dedicated tensor loop lowering omitted %q\n%s", needle, body)
		}
	}
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "memcpy", "cblas", "MLIR"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("forbidden tensor mechanism %q appeared", forbidden)
		}
	}
}

func TestEVT1R4hMIRValidationRejectsIncompleteTensor(t *testing.T) {
	bad := MIR{Functions: []MIRFunction{{Name: "BadTensor", TensorOperations: []MIRTensorOperation{{Kind: "tensor_contract"}}}}}
	err := evt1ValidateMIR(bad)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4626" || diagnostic.SemanticCategory() != "TENSOR_MIR_INVALID" {
		t.Fatalf("unexpected tensor MIR diagnostic: %v", err)
	}
}

func TestEVT1R4hNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"tensor_from_ndarray_rank1.concept", "concept_tensor_from_ndarray_rank1_tensor_from_ndarray_rank1(1)", 5},
		{"tensor_from_ndarray_rank2.concept", "concept_tensor_from_ndarray_rank2_tensor_from_ndarray_rank2()", 3},
		{"tensor_from_ndarray_rank3.concept", "concept_tensor_from_ndarray_rank3_tensor_from_ndarray_rank3()", 4},
		{"tensor_from_bound_ndarray.concept", "concept_tensor_from_bound_ndarray_tensor_from_bound_ndarray(2, 3)", 6},
		{"tensor_from_layout_region.concept", "concept_tensor_from_layout_region_tensor_from_layout_region()", 4},
		{"tensor_from_stream_channel.concept", "concept_tensor_from_stream_channel_tensor_from_stream_channel()", 3},
		{"tensor_from_span_rank1.concept", "concept_tensor_from_span_rank1_tensor_from_span_rank1()", 7},
		{"tensor_elementwise_add.concept", "concept_tensor_elementwise_add_tensor_elementwise_add()", 12},
		{"tensor_elementwise_mul.concept", "concept_tensor_elementwise_mul_tensor_elementwise_mul()", 21},
		{"tensor_scalar_mul.concept", "concept_tensor_scalar_mul_tensor_scalar_mul(3)", 12},
		{"tensor_matmul_at.concept", "concept_tensor_matmul_at_tensor_matmul_at()", 154},
		{"tensor_rank3_at.concept", "concept_tensor_rank3_at_tensor_rank3at()", 46},
		{"tensor_einstein_matmul.concept", "concept_tensor_einstein_matmul_einstein_matmul()", 154},
		{"tensor_float_einstein.concept", "concept_tensor_float_einstein_tensor_float_einstein(1.0f, 2.0f, 3.0f, 4.0f, 0.0f)", 4},
		{"tensor_readonly_operand.concept", "concept_tensor_readonly_operand_tensor_readonly_operand()", 8},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4hFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\nint main(void) { return " + tc.call + " == " + fmt.Sprint(tc.want) + " ? 0 : 1; }\n"
			runR1NativeHarness(t, outputs, "r4h_harness.c", harness)
		})
	}
}

func TestEVT1R4hRuntimeGuards(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	cases := []struct{ file, call, reason string }{{"tensor_from_ndarray_rank1.concept", "concept_tensor_from_ndarray_rank1_tensor_from_ndarray_rank1(3)", "Concept tensor index out of bounds"}, {"tensor_runtime_shape_guard.concept", "concept_tensor_runtime_shape_guard_tensor_runtime_shape_guard(2, 3, 2, 3)", "Concept tensor contraction shape mismatch"}}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4hFixture(t, "valid", tc.file)
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := filepath.Join(dir, "guard.c")
			if err := os.WriteFile(harness, []byte("#include \""+base+".generated.h\"\nint main(void) { return "+tc.call+"; }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "guard.exe")
			generated := filepath.Join(dir, base+".generated.c")
			if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harness, "-o", exe).CombinedOutput(); err != nil {
				t.Fatalf("native compile failed: %v\n%s", err, out)
			}
			out, err := exec.Command(exe).CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.reason) {
				t.Fatalf("runtime guard evidence missing: err=%v output=%s", err, out)
			}
		})
	}
}

func TestEVT1R4hComptimeSymbolicInitialization(t *testing.T) {
	outputs := generateR4hFixture(t, "valid", "tensor_einstein_initialize.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["tensor_einstein_initialize.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.StaticAsserts) != 1 || !mir.StaticAsserts[0].Satisfied {
		t.Fatalf("comptime tensor static_assert not satisfied: %+v", mir.StaticAsserts)
	}
	if len(mir.ComptimeFns) != 1 || len(mir.ComptimeFns[0].TensorOperations) != 1 || mir.ComptimeFns[0].TensorOperations[0].Kind != "tensor_index_assign" {
		t.Fatalf("comptime Tensor MIR missing: %+v", mir.ComptimeFns)
	}
}
