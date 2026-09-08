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

var evt1R4dConformanceCases = []ConformanceCase{
	{Name: "fixed array", Source: "valid/array_fixed.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Phase 21 fixed array value storage", MatrixStatus: "Merge / canonical array", Notes: "Explicit EVT1 storage-family spelling."},
	{Name: "mutable array index", Source: "valid/array_mutable_index.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Phase 21 mutable indexing", MatrixStatus: "Merge", Notes: "Indexed place inherits mutability."},
	{Name: "const array read", Source: "valid/array_const_read.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R2 const place law", MatrixStatus: "EVT1-new composition", Notes: "Const storage remains readable."},
	{Name: "array copy", Source: "valid/array_copy.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Phase 21 wrapper value copy", MatrixStatus: "Merge", Notes: "C wrapper assignment preserves value semantics."},
	{Name: "fixed ndarray", Source: "valid/ndarray_fixed_2d.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V shape direction", MatrixStatus: "EVT1-new", Notes: "One contiguous rank-2 value."},
	{Name: "mutable ndarray index", Source: "valid/ndarray_index_2d.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL-V shape direction", MatrixStatus: "EVT1-new", Notes: "Comma-separated row-major projection."},
	{Name: "ndarray literal", Source: "valid/ndarray_nested_literal.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 literal pressure plus EVT1 ndarray", MatrixStatus: "EVT1-new", Notes: "Nested syntax lowers to flat storage."},
	{Name: "rank and shape", Source: "valid/ndarray_rank_shape_query.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d shape law", MatrixStatus: "EVT1-new", Notes: "No allocated shape object."},
	{Name: "array ref mutation", Source: "valid/array_ref_mutation.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R3 ref law", MatrixStatus: "EVT1-new composition", Notes: "Mutable ref projects a mutable element place."},
	{Name: "ndarray ref const", Source: "valid/ndarray_ref_const.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R3 ref const law", MatrixStatus: "EVT1-new composition", Notes: "Read-only reference projection."},
	{Name: "Result array payload", Source: "valid/result_array_payload.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R4c Result law", MatrixStatus: "EVT1-new composition", Notes: "Storage is an ordinary structural payload."},
	{Name: "runtime bounds", Source: "valid/array_runtime_bounds.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Phase 21 bounds panic", MatrixStatus: "Merge", Notes: "Dynamic index lowers to terminal panic guard."},
	{Name: "owned array move and drop", Source: "valid/array_owned_move_drop.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R3 move and Drop law", MatrixStatus: "EVT1-new composition", Notes: "Owned storage moves explicitly and drops once."},
	{Name: "movable-only element array", Source: "valid/array_movable_element.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R3 structural move law", MatrixStatus: "EVT1-new composition", Notes: "A noncopyable movable element makes storage movable-only."},
	{Name: "array element reverse drop", Source: "valid/array_element_drop_order.concept", Expected: ConformancePass, Accepted: true, Reference: "EVT1 R3 deterministic Drop law", MatrixStatus: "EVT1-new composition", Notes: "Droppable elements are traversed in reverse storage order."},
	{Name: "ndarray runtime bounds", Source: "valid/ndarray_runtime_bounds.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d bounds law", MatrixStatus: "EVT1-new", Notes: "Every runtime ndarray index is checked before row-major projection."},
	{Name: "storage value boundaries", Source: "valid/storage_value_boundaries.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Phase 21 array boundary values plus EVT1 ndarray", MatrixStatus: "Merge / EVT1-new", Notes: "Fixed storage is passable and returnable as an ordinary value."},
	{Name: "const mutation rejected", Source: "invalid/array_const_mutation.concept", Expected: ConformancePass, DiagnosticCategory: "CV4128", Reference: "EVT1 R2 const place law", MatrixStatus: "EVT1-new composition", Notes: "Indexed projection cannot bypass const."},
	{Name: "constant array OOB", Source: "invalid/array_oob_constant.concept", Expected: ConformancePass, DiagnosticCategory: "CV4233", Reference: "PoC3 Phase 21 bounds", MatrixStatus: "Merge", Notes: "Provable OOB is diagnosed."},
	{Name: "ndarray index arity", Source: "invalid/ndarray_wrong_index_count.concept", Expected: ConformancePass, DiagnosticCategory: "CV4557", Reference: "R4d rank law", MatrixStatus: "EVT1-new", Notes: "Index arity equals rank."},
	{Name: "constant ndarray OOB", Source: "invalid/ndarray_oob_constant.concept", Expected: ConformancePass, DiagnosticCategory: "CV4561", Reference: "R4d bounds law", MatrixStatus: "EVT1-new", Notes: "Each dimension is checked."},
	{Name: "ragged ndarray", Source: "invalid/ndarray_ragged_literal.concept", Expected: ConformancePass, DiagnosticCategory: "CV4556", Reference: "R4d literal law", MatrixStatus: "EVT1-new", Notes: "Ragged storage is not ndarray."},
	{Name: "ndarray shape mismatch", Source: "invalid/ndarray_shape_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4555", Reference: "R4d literal law", MatrixStatus: "EVT1-new", Notes: "Literal shape must match declared shape."},
	{Name: "runtime array explicit storage", Source: "invalid/runtime_array_without_storage.concept", Expected: ConformancePass, DiagnosticCategory: "CV4558", Reference: "R4d no-hidden-allocation law", MatrixStatus: "Redesign / bounded rejection", Notes: "Shape never selects allocation."},
	{Name: "runtime ndarray explicit storage", Source: "invalid/runtime_ndarray_without_storage.concept", Expected: ConformancePass, DiagnosticCategory: "CV4559", Reference: "R4d no-hidden-allocation law", MatrixStatus: "EVT1-new bounded rejection", Notes: "Shape never selects allocation."},
	{Name: "ndarray literal rank mismatch", Source: "invalid/ndarray_literal_rank_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4554", Reference: "R4d literal law", MatrixStatus: "EVT1-new", Notes: "Nested literal rank must match storage rank."},
	{Name: "immovable element propagation", Source: "invalid/array_immovable_element.concept", Expected: ConformancePass, DiagnosticCategory: "CV4134", Reference: "EVT1 R2 immovable law", MatrixStatus: "EVT1-new composition", Notes: "Immovability propagates structurally into array storage."},
	{Name: "array integer index", Source: "invalid/array_index_non_integer.concept", Expected: ConformancePass, DiagnosticCategory: "CV4232", Reference: "PoC3 Phase 21 index type law", MatrixStatus: "Merge", Notes: "Storage indices are integers."},
	{Name: "ndarray too many indices", Source: "invalid/ndarray_too_many_indices.concept", Expected: ConformancePass, DiagnosticCategory: "CV4557", Reference: "R4d rank law", MatrixStatus: "EVT1-new", Notes: "Too many indices are rejected as well as too few."},
	{Name: "noncopyable element propagation", Source: "invalid/array_noncopyable_element.concept", Expected: ConformancePass, DiagnosticCategory: "CV4133", Reference: "EVT1 R3 structural copyability", MatrixStatus: "EVT1-new composition", Notes: "A noncopyable element makes its fixed storage noncopyable."},
}

func TestEVT1R4dConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R4dConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1-r4d", "core", filepath.FromSlash(tc.Source))
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
	if counts[ConformancePass] != 30 || len(counts) != 1 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestEVT1R4dMIRPreservesStorageFacts(t *testing.T) {
	outputs := generateR4dFixture(t, "ndarray_rank_shape_query.concept")
	var mir MIR
	var header, body string
	for name, artifact := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(artifact, &mir); err != nil {
				t.Fatal(err)
			}
		} else if strings.HasSuffix(name, ".generated.h") {
			header = string(artifact)
		} else if strings.HasSuffix(name, ".generated.c") {
			body = string(artifact)
		}
	}
	if len(mir.StorageTypes) != 1 {
		t.Fatalf("storage type count = %d, want 1", len(mir.StorageTypes))
	}
	storage := mir.StorageTypes[0]
	if storage.StorageKind != StorageNDArray || storage.Rank != 3 || !storage.Contiguous || storage.Layout != "row-major" || storage.Ownership != "fixed_inline" {
		t.Fatalf("unexpected storage facts: %+v", storage)
	}
	if storage.Shape[0].Extent != 2 || storage.Shape[1].Extent != 3 || storage.Shape[2].Extent != 4 {
		t.Fatalf("unexpected shape: %+v", storage.Shape)
	}
	var foundIndex, foundRank, foundShape bool
	for _, operation := range mir.Functions[0].Operations {
		foundIndex = foundIndex || operation.Kind == "ndarray_index"
		foundRank = foundRank || operation.Kind == "rank_query"
		foundShape = foundShape || operation.Kind == "shape_query"
	}
	if !foundIndex || !foundRank || !foundShape {
		t.Fatalf("MIR operations omit storage semantics: %+v", mir.Functions[0].Operations)
	}
	if !strings.Contains(header, "data[24]") || !strings.Contains(body, "* 3") || !strings.Contains(body, "* 4") {
		t.Fatalf("generated C does not preserve flat row-major shape\nHEADER:\n%s\nBODY:\n%s", header, body)
	}
	for _, forbidden := range []string{"malloc", "stackalloc", "Slice", "FixedBuffer", "tensor"} {
		if strings.Contains(header+body, forbidden) {
			t.Fatalf("forbidden R4d mechanism %q appeared in generated storage path", forbidden)
		}
	}
}

func TestEVT1R4dNativeC11(t *testing.T) {
	cases := []struct {
		file, symbol string
		want         int
	}{
		{"array_fixed.concept", "concept_array_fixed_read_fixed", 3},
		{"array_mutable_index.concept", "concept_array_mutable_index_mutate_fixed", 9},
		{"array_const_read.concept", "concept_array_const_read_read_const", 5},
		{"array_copy.concept", "concept_array_copy_copy_is_independent", 11},
		{"ndarray_fixed_2d.concept", "concept_ndarray_fixed_2d_read2d", 6},
		{"ndarray_index_2d.concept", "concept_ndarray_index_2d_mutate2d", 8},
		{"ndarray_nested_literal.concept", "concept_ndarray_nested_literal_nested_literal", 50},
		{"ndarray_rank_shape_query.concept", "concept_ndarray_rank_shape_query_query_shape", 13},
		{"array_ref_mutation.concept", "concept_array_ref_mutation_mutate_through_ref", 7},
		{"ndarray_ref_const.concept", "concept_ndarray_ref_const_use_const_ref", 5},
		{"result_array_payload.concept", "concept_result_array_payload_read_payload", 4},
		{"array_owned_move_drop.concept", "concept_array_owned_move_drop_move_owned_array", 2},
		{"array_movable_element.concept", "concept_array_movable_element_move_element_array", 7},
		{"array_element_drop_order.concept", "concept_array_element_drop_order_drop_array_elements", 0},
		{"ndarray_runtime_bounds.concept", "concept_ndarray_runtime_bounds_runtime_index", 4},
		{"storage_value_boundaries.concept", "concept_storage_value_boundaries_value_boundaries", 5},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4dFixture(t, tc.file)
			harness := "#include \"" + strings.TrimSuffix(tc.file, ".concept") + ".generated.h\"\nint main(void) { return " + tc.symbol + "() == " + fmt.Sprint(tc.want) + " ? 0 : 1; }\n"
			runR1NativeHarness(t, outputs, "r4d_harness.c", harness)
		})
	}
}

func TestEVT1R4dRuntimeBoundsPanic(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	outputs := generateR4dFixture(t, "array_runtime_bounds.concept")
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(dir, "bounds_harness.c")
	if err := os.WriteFile(harness, []byte("#include \"array_runtime_bounds.generated.h\"\nint main(void) { return concept_array_runtime_bounds_read_checked(4); }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "bounds.exe")
	generated := filepath.Join(dir, "array_runtime_bounds.generated.c")
	if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harness, "-o", executable).CombinedOutput(); err != nil {
		t.Fatalf("native C11 compile failed: %v\n%s", err, out)
	}
	out, err := exec.Command(executable).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "Concept array index out of bounds") || !strings.Contains(string(out), "Concept panic at") {
		t.Fatalf("bounds panic evidence missing: err=%v output=%s", err, out)
	}
}

func TestEVT1R4dOwnedArrayTransfersAndDropsOnce(t *testing.T) {
	outputs := generateR4dFixture(t, "array_owned_move_drop.concept")
	body := string(outputs["array_owned_move_drop.generated.c"])
	if strings.Count(body, "concept_array_owned_move_drop_drop(second);") != 1 {
		t.Fatalf("moved-to array must drop exactly once\n%s", body)
	}
	if strings.Contains(body, "concept_array_owned_move_drop_drop(first);") {
		t.Fatalf("moved-from array was dropped\n%s", body)
	}

	elementOutputs := generateR4dFixture(t, "array_element_drop_order.concept")
	elementBody := string(elementOutputs["array_element_drop_order.generated.c"])
	third := strings.Index(elementBody, "concept_array_element_drop_order_drop((values).data[2]);")
	second := strings.Index(elementBody, "concept_array_element_drop_order_drop((values).data[1]);")
	first := strings.Index(elementBody, "concept_array_element_drop_order_drop((values).data[0]);")
	if third < 0 || second <= third || first <= second {
		t.Fatalf("element drops are not emitted in reverse order\n%s", elementBody)
	}
}

func generateR4dFixture(t *testing.T, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r4d", "core", "valid", file)
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
