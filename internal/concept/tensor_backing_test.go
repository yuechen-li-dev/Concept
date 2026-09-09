package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var tensorBackingConformanceCases = []ConformanceCase{
	{Name: "inline rank 1", Source: "valid/tensor_inline_rank1.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline rank 2", Source: "valid/tensor_inline_rank2.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline rank 3", Source: "valid/tensor_inline_rank3.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline literal", Source: "valid/tensor_inline_literal.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline scalar fill", Source: "valid/tensor_inline_scalar_fill.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline Einstein initialize", Source: "valid/tensor_inline_einstein_initialize.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline matmul", Source: "valid/tensor_inline_matmul.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline rank 3 contract", Source: "valid/tensor_inline_rank3_contract.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "inline const", Source: "valid/tensor_inline_const.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "vector", Source: "valid/vector_basic.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new alias"},
	{Name: "vector dot", Source: "valid/vector_dot_product.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "matrix", Source: "valid/matrix_basic.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new alias"},
	{Name: "matrix matmul", Source: "valid/matrix_matmul.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "explicit ndarray", Source: "valid/tensor_external_storage_still_valid.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "R4h preserved"},
	{Name: "layout region", Source: "valid/tensor_layout_region_still_valid.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "R4h preserved"},
	{Name: "stream channel", Source: "valid/tensor_stream_channel_still_valid.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "R4h preserved"},
	{Name: "runtime shape", Source: "invalid/tensor_inline_runtime_shape.concept", Expected: ConformancePass, DiagnosticCategory: "CV4627", MatrixStatus: "EVT1-new"},
	{Name: "literal shape mismatch", Source: "invalid/tensor_inline_literal_shape_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4555", MatrixStatus: "R4d reused"},
	{Name: "bad element", Source: "invalid/tensor_inline_bad_element_type.concept", Expected: ConformancePass, DiagnosticCategory: "CV4612", MatrixStatus: "R4h reused"},
	{Name: "vector rank", Source: "invalid/vector_rank_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4628", MatrixStatus: "EVT1-new"},
	{Name: "matrix rank", Source: "invalid/matrix_rank_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4629", MatrixStatus: "EVT1-new"},
	{Name: "broadcast remains rejected", Source: "invalid/tensor_broadcast_still_rejected.concept", Expected: ConformancePass, DiagnosticCategory: "CV4617", MatrixStatus: "R4h preserved"},
	{Name: "inline alias hazard", Source: "invalid/tensor_inline_alias_hazard.concept", Expected: ConformancePass, DiagnosticCategory: "CV4625", MatrixStatus: "R4h preserved"},
	{Name: "scalar result type", Source: "invalid/tensor_scalar_result_type_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4632", MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "invalid inline shape", Source: "invalid/tensor_inline_shape_invalid.concept", Expected: ConformancePass, DiagnosticCategory: "CV4630", MatrixStatus: "EVT1-new / fixed storage derived"},
	{Name: "scalar fill type", Source: "invalid/tensor_inline_scalar_fill_type_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4631", MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "const inline destination", Source: "invalid/tensor_inline_const_destination.concept", Expected: ConformancePass, DiagnosticCategory: "CV4616", MatrixStatus: "R3/R4h preserved"},
}

func generateTensorBackingFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "tensor", "backing", class, file)
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

func TestTensorBackingConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range tensorBackingConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "tensor", "backing", filepath.FromSlash(tc.Source))
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
	if counts[ConformancePass] != 27 || len(counts) != 1 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestTensorBackingInlineBackingAndTensorMIR(t *testing.T) {
	outputs := generateTensorBackingFixture(t, "valid", "tensor_inline_matmul.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["tensor_inline_matmul.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	inline, views := 0, 0
	regions := map[string]bool{}
	for _, operation := range mir.Functions[0].Operations {
		switch operation.Kind {
		case "tensor_inline_storage":
			inline++
			regions[operation.RegionID] = true
			if operation.TargetStorageKind != StorageNDArray || operation.NoAllocation != true || operation.Provenance != "local" {
				t.Fatalf("inline backing lost fixed storage facts: %+v", operation)
			}
		case "tensor_view":
			views++
			if !operation.SameBackingRegion || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
				t.Fatalf("inline tensor view lost R4h facts: %+v", operation)
			}
		}
	}
	if inline != 3 || views != 3 || len(regions) != 3 {
		t.Fatalf("inline backing identity drift: inline=%d views=%d regions=%v", inline, views, regions)
	}
	if len(mir.Functions[0].TensorOperations) != 1 || mir.Functions[0].TensorOperations[0].Kind != "tensor_contract" {
		t.Fatalf("inline matmul did not reuse Tensor MIR: %+v", mir.Functions[0].TensorOperations)
	}
	generated := string(outputs["tensor_inline_matmul.generated.c"])
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "memcpy", "cblas", "MLIR"} {
		if strings.Contains(generated, forbidden) {
			t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
		}
	}
}

func TestTensorBackingRankZeroTensorMIR(t *testing.T) {
	outputs := generateTensorBackingFixture(t, "valid", "vector_dot_product.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["vector_dot_product.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	ops := mir.Functions[0].TensorOperations
	if len(ops) != 1 || ops[0].Kind != "tensor_scalar_contract" || ops[0].Output.Rank != 0 || len(ops[0].Operands) != 2 {
		t.Fatalf("rank-zero Tensor MIR missing: %+v", ops)
	}
}

func TestTensorBackingMIRValidationRejectsIncompleteInlineBacking(t *testing.T) {
	bad := MIR{Functions: []MIRFunction{{Name: "BadInline", Operations: []MIROperation{{ID: "BadInline.01", Kind: "tensor_inline_storage"}}}}}
	err := evt1ValidateMIR(bad)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4626" || diagnostic.SemanticCategory() != "TENSOR_MIR_INVALID" {
		t.Fatalf("unexpected inline Tensor MIR diagnostic: %v", err)
	}
}

func TestTensorBackingNativeC11(t *testing.T) {
	cases := []struct {
		file string
		call string
		want string
	}{
		{"tensor_inline_rank1.concept", "concept_tensor_inline_rank1_tensor_inline_rank1()", "6"},
		{"tensor_inline_rank2.concept", "concept_tensor_inline_rank2_tensor_inline_rank2()", "3"},
		{"tensor_inline_rank3.concept", "concept_tensor_inline_rank3_tensor_inline_rank3()", "6"},
		{"tensor_inline_literal.concept", "concept_tensor_inline_literal_tensor_inline_literal()", "6"},
		{"tensor_inline_scalar_fill.concept", "concept_tensor_inline_scalar_fill_tensor_inline_scalar_fill()", "7"},
		{"tensor_inline_einstein_initialize.concept", "concept_tensor_inline_einstein_initialize_tensor_inline_einstein_initialize()", "1.0f"},
		{"tensor_inline_matmul.concept", "concept_tensor_inline_matmul_tensor_inline_matmul()", "50"},
		{"tensor_inline_rank3_contract.concept", "concept_tensor_inline_rank3_contract_tensor_inline_rank3contract()", "46"},
		{"vector_basic.concept", "concept_vector_basic_vector_basic()", "3.0f"},
		{"vector_dot_product.concept", "concept_vector_dot_product_vector_dot_product()", "32"},
		{"matrix_basic.concept", "concept_matrix_basic_matrix_basic()", "1.0f"},
		{"matrix_matmul.concept", "concept_matrix_matmul_matrix_matmul()", "50"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateTensorBackingFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\nint main(void) { return " + tc.call + " == " + tc.want + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, "tensor_backing_harness.c", harness)
		})
	}
}

func TestTensorBackingDefaultInitializationUsesExistingLaw(t *testing.T) {
	_, err := Parse("tensor_default.concept", "profile Core; int F() { tensor<int> A[2, 2]; return 0; }")
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4560" {
		t.Fatalf("default initialization law drift: %v", err)
	}
}

func TestTensorBackingAliasTypeIdentity(t *testing.T) {
	module, err := Parse("aliases.concept", "profile Core; int F() { vector<int> v[1] = [1]; matrix<int> m[1, 1] = [[1]]; return v[0] + m[0, 0]; }")
	if err != nil {
		t.Fatal(err)
	}
	vector := module.Functions[0].Body.Statements[0].(*VarDecl).Type
	matrix := module.Functions[0].Body.Statements[1].(*VarDecl).Type
	intType, _ := evt1BuiltinType("int", Span{})
	if !vector.Equal(evt1TensorType(intType, 1, vector.Span)) || !matrix.Equal(evt1TensorType(intType, 2, matrix.Span)) {
		t.Fatalf("aliases are not exact tensor identities: %s %s", vector.String(), matrix.String())
	}
}

func TestTensorBackingBackingClassification(t *testing.T) {
	cases := []struct {
		class string
		file  string
		want  TensorBackingKind
	}{
		{"valid", "tensor_inline_rank2.concept", TensorBackingInline},
		{"valid", "tensor_external_storage_still_valid.concept", TensorBackingNDArray},
		{"valid", "tensor_layout_region_still_valid.concept", TensorBackingLayoutRegion},
		{"valid", "tensor_stream_channel_still_valid.concept", TensorBackingStreamChannel},
	}
	for _, tc := range cases {
		outputs := generateTensorBackingFixture(t, tc.class, tc.file)
		base := strings.TrimSuffix(tc.file, ".concept")
		var mir MIR
		if err := json.Unmarshal(outputs[base+".mir.json"], &mir); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, op := range mir.Functions[len(mir.Functions)-1].Operations {
			if op.Kind == "tensor_view" {
				found = true
				if op.TensorBackingKind != tc.want {
					t.Fatalf("%s backing = %s, want %s", tc.file, op.TensorBackingKind, tc.want)
				}
			}
		}
		if !found {
			t.Fatalf("%s has no tensor view", tc.file)
		}
	}
}

func TestTensorBackingPreservedBackingClassification(t *testing.T) {
	cases := []struct {
		file string
		want TensorBackingKind
	}{
		{"tensor_from_bound_ndarray.concept", TensorBackingBoundNDArray},
		{"tensor_from_span_rank1.concept", TensorBackingSpan},
	}
	for _, tc := range cases {
		outputs := generateTensorFixture(t, "valid", tc.file)
		base := strings.TrimSuffix(tc.file, ".concept")
		var mir MIR
		if err := json.Unmarshal(outputs[base+".mir.json"], &mir); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, op := range mir.Functions[0].Operations {
			if op.Kind == "tensor_view" {
				found = true
				if op.TensorBackingKind != tc.want {
					t.Fatalf("%s backing = %s, want %s", tc.file, op.TensorBackingKind, tc.want)
				}
			}
		}
		if !found {
			t.Fatalf("%s has no tensor view", tc.file)
		}
	}
}
