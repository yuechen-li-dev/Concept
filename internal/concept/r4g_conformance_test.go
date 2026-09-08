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

var evt1R4gConformanceCases = []ConformanceCase{
	{Name: "array", Source: "valid/span_from_array.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Slice pressure", MatrixStatus: "EVT1-new / C#-inspired"},
	{Name: "ndarray", Source: "valid/span_from_ndarray.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d ndarray", MatrixStatus: "EVT1-new composition"},
	{Name: "bound ndarray", Source: "valid/span_from_bound_ndarray.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e bind", MatrixStatus: "EVT1-new composition"},
	{Name: "layout region", Source: "valid/span_from_layout_region.concept", Expected: ConformancePass, Accepted: true, Reference: "R4f region", MatrixStatus: "EVT1-new composition"},
	{Name: "stream channel", Source: "valid/span_from_stream_channel.concept", Expected: ConformancePass, Accepted: true, Reference: "R4f channel", MatrixStatus: "EVT1-new composition"},
	{Name: "alias mutation", Source: "valid/span_mutation_aliases_source.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Slice pressure", MatrixStatus: "Reconciled / EVT1-new"},
	{Name: "readonly const", Source: "valid/readonly_span_from_const.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 ref const", MatrixStatus: "EVT1-new composition"},
	{Name: "mutable to readonly", Source: "valid/span_to_readonly.concept", Expected: ConformancePass, Accepted: true, Reference: "C# Span direction", MatrixStatus: "C#-inspired"},
	{Name: "subspan", Source: "valid/span_subspan.concept", Expected: ConformancePass, Accepted: true, Reference: "PoC3 Slice pressure", MatrixStatus: "Reconciled / EVT1-new"},
	{Name: "empty subspan", Source: "valid/span_empty_subspan.concept", Expected: ConformancePass, Accepted: true, Reference: "half-open intervals", MatrixStatus: "EVT1-new"},
	{Name: "Len", Source: "valid/span_len.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d Len", MatrixStatus: "EVT1-new composition"},
	{Name: "scoped", Source: "valid/span_scoped_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4a scoped", MatrixStatus: "EVT1-new composition"},
	{Name: "immovable backing", Source: "valid/span_immovable_backing.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 immovable ref", MatrixStatus: "EVT1-new composition"},
	{Name: "helper provenance", Source: "valid/span_helper_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4b call result", MatrixStatus: "EVT1-new composition"},
	{Name: "after question", Source: "valid/span_after_question.concept", Expected: ConformancePass, Accepted: true, Reference: "R4c Result", MatrixStatus: "EVT1-new composition"},
	{Name: "mutable from const", Source: "invalid/span_mutable_from_const.concept", Expected: ConformancePass, DiagnosticCategory: "CV4602", MatrixStatus: "EVT1-new"},
	{Name: "readonly mutation", Source: "invalid/readonly_span_mutation.concept", Expected: ConformancePass, DiagnosticCategory: "CV4605", MatrixStatus: "EVT1-new"},
	{Name: "constant subspan OOB", Source: "invalid/span_subspan_oob_constant.concept", Expected: ConformancePass, DiagnosticCategory: "CV4603", MatrixStatus: "EVT1-new"},
	{Name: "runtime subspan OOB", Source: "invalid/span_subspan_runtime_oob.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new terminal path"},
	{Name: "constant index OOB", Source: "invalid/span_index_oob_constant.concept", Expected: ConformancePass, DiagnosticCategory: "CV4604", MatrixStatus: "EVT1-new"},
	{Name: "runtime index OOB", Source: "invalid/span_index_runtime_oob.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new terminal path"},
	{Name: "local escape", Source: "invalid/span_escape_local_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4a composition"},
	{Name: "helper escape", Source: "invalid/span_escape_helper_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4b composition"},
	{Name: "stream escape", Source: "invalid/span_escape_stream_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4f composition"},
	{Name: "layout escape", Source: "invalid/span_escape_layout_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4f composition"},
	{Name: "noncontiguous", Source: "invalid/span_noncontiguous_source.concept", Expected: ConformancePass, DiagnosticCategory: "CV4600", MatrixStatus: "EVT1-new"},
	{Name: "element mismatch", Source: "invalid/span_element_type_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4601", MatrixStatus: "EVT1-new"},
	{Name: "readonly to mutable", Source: "invalid/span_from_readonly.concept", Expected: ConformancePass, DiagnosticCategory: "CV4602", MatrixStatus: "EVT1-new"},
	{Name: "Result lifetime laundering", Source: "invalid/span_result_lifetime_launder.concept", Expected: ConformancePass, DiagnosticCategory: "CV4521", MatrixStatus: "R4b/R4c composition"},
}

func TestEVT1R4gConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R4gConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1-r4g", "core", filepath.FromSlash(tc.Source))
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

func generateR4gFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r4g", "core", class, file)
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

func TestEVT1R4gMIRPreservesBorrowedRegionFacts(t *testing.T) {
	outputs := generateR4gFixture(t, "valid", "span_from_stream_channel.concept")
	var mir MIR
	var emitted strings.Builder
	for name, artifact := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(artifact, &mir); err != nil {
				t.Fatal(err)
			}
		}
		if strings.HasSuffix(name, ".generated.h") || strings.HasSuffix(name, ".generated.c") {
			emitted.Write(artifact)
		}
	}
	var spanFromRegion, spanIndex bool
	for _, operation := range mir.Functions[0].Operations {
		if operation.Kind == "span_from_region" {
			spanFromRegion = true
			if operation.RegionID != "VertexLayout.positions" || operation.BaseOffset != "0" || operation.Length != "3" || operation.Alignment != 4 || operation.Mutability != "mutable" || operation.Provenance != "local" || !operation.Contiguous || !operation.SameBackingRegion || !operation.NoCopy || !operation.NoAllocation || !operation.NoOwnershipTransfer {
				t.Fatalf("span facts = %+v", operation)
			}
		}
		spanIndex = spanIndex || operation.Kind == "span_index"
	}
	if !spanFromRegion || !spanIndex {
		t.Fatalf("missing span MIR operations: %+v", mir.Functions[0].Operations)
	}
	text := emitted.String()
	if !strings.Contains(text, "typedef struct { float* data; size_t length; } concept_span_float;") {
		t.Fatalf("pointer-plus-length span descriptor missing\n%s", text)
	}
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "memcpy"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("forbidden backing mechanism %q appeared", forbidden)
		}
	}

	subspanOutputs := generateR4gFixture(t, "valid", "span_subspan.concept")
	var subspanMIR MIR
	if err := json.Unmarshal(subspanOutputs["span_subspan.mir.json"], &subspanMIR); err != nil {
		t.Fatal(err)
	}
	var subspanFound bool
	for _, operation := range subspanMIR.Functions[0].Operations {
		if operation.Kind == "span_subregion" {
			if operation.RegionID != "storage:values" || operation.BaseOffset != "(0+2)" || operation.Length != "3" || operation.Alignment != 4 || operation.BoundsCheck != "half_open_overflow_safe" {
				t.Fatalf("subspan did not retain and narrow parent facts: %+v", operation)
			}
			subspanFound = true
		}
	}
	if !subspanFound {
		t.Fatal("span_subregion MIR operation missing")
	}

	layoutOutputs := generateR4gFixture(t, "valid", "span_from_layout_region.concept")
	var layoutMIR MIR
	if err := json.Unmarshal(layoutOutputs["span_from_layout_region.mir.json"], &layoutMIR); err != nil {
		t.Fatal(err)
	}
	var layoutSpan, alignedTail bool
	for _, operation := range layoutMIR.Functions[0].Operations {
		if operation.Kind == "span_from_region" && operation.RegionID == "FrameMemory.vertices" {
			layoutSpan = operation.Offset == 16 && operation.Alignment == 16
		}
		if operation.Kind == "span_subregion" && operation.RegionID == "FrameMemory.vertices" {
			alignedTail = operation.BaseOffset == "(0+1)" && operation.Alignment == 4
		}
	}
	if !layoutSpan || !alignedTail {
		t.Fatalf("layout offset/alignment facts were not preserved and narrowed: %+v", layoutMIR.Functions[0].Operations)
	}

	helperOutputs := generateR4gFixture(t, "valid", "span_helper_provenance.concept")
	var helperMIR MIR
	if err := json.Unmarshal(helperOutputs["span_helper_provenance.mir.json"], &helperMIR); err != nil {
		t.Fatal(err)
	}
	var helperResultRegion bool
	for _, fn := range helperMIR.Functions {
		if fn.Name != "SpanHelperProvenance" {
			continue
		}
		for _, operation := range fn.Operations {
			if operation.Kind == "span_index" && operation.RegionID == "storage:values" && operation.Provenance == "local" {
				helperResultRegion = true
			}
		}
	}
	if !helperResultRegion {
		t.Fatalf("helper result did not retain the caller source region: %+v", helperMIR.Functions)
	}
}

func TestEVT1R4gMIRValidationRejectsIncompleteSpanFacts(t *testing.T) {
	bad := MIR{Functions: []MIRFunction{{
		Name:       "BadSpan",
		Operations: []MIROperation{{ID: "bad", Kind: "span_from_region", RegionID: "storage:source"}},
	}}}
	err := evt1ValidateMIR(bad)
	if err == nil {
		t.Fatal("incomplete Span MIR facts were accepted")
	}
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4608" || diagnostic.SemanticCategory() != "SPAN_MIR_INVALID" {
		t.Fatalf("unexpected Span MIR diagnostic: %v", err)
	}
}

func TestEVT1R4gNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"span_from_array.concept", "concept_span_from_array_span_from_array()", 4},
		{"span_from_ndarray.concept", "concept_span_from_ndarray_span_from_ndarray()", 5},
		{"span_from_bound_ndarray.concept", "concept_span_from_bound_ndarray_span_from_bound_ndarray(2, 3)", 6},
		{"span_from_layout_region.concept", "concept_span_from_layout_region_span_from_layout_region(1.0f, 3.0f)", 3},
		{"span_from_stream_channel.concept", "concept_span_from_stream_channel_span_from_stream_channel(1.0f, 7.0f)", 7},
		{"span_mutation_aliases_source.concept", "concept_span_mutation_aliases_source_span_mutation_aliases_source()", 140},
		{"readonly_span_from_const.concept", "concept_readonly_span_from_const_readonly_span_from_const()", 3},
		{"span_to_readonly.concept", "concept_span_to_readonly_span_to_readonly()", 6},
		{"span_subspan.concept", "concept_span_subspan_span_subspan()", 43},
		{"span_empty_subspan.concept", "concept_span_empty_subspan_span_empty_subspan()", 0},
		{"span_len.concept", "concept_span_len_span_length()", 5},
		{"span_scoped_provenance.concept", "concept_span_scoped_provenance_span_scoped_provenance()", 12},
		{"span_immovable_backing.concept", "concept_span_immovable_backing_span_immovable_backing()", 8},
		{"span_helper_provenance.concept", "concept_span_helper_provenance_span_helper_provenance()", 6},
		{"span_after_question.concept", "concept_span_after_question_span_after_question()", 8},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4gFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\nint main(void) { return " + tc.call + " == " + fmt.Sprint(tc.want) + " ? 0 : 1; }\n"
			runR1NativeHarness(t, outputs, "r4g_harness.c", harness)
		})
	}
}

func TestEVT1R4gRuntimeBoundsPanics(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	cases := []struct {
		file, call, reason string
	}{
		{"span_subspan_runtime_oob.concept", "concept_span_subspan_runtime_oob_runtime_subspan(3, 2)", "Concept span bounds out of range"},
		{"span_index_runtime_oob.concept", "concept_span_index_runtime_oob_runtime_span_index(4)", "Concept span index out of bounds"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4gFixture(t, "invalid", tc.file)
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := filepath.Join(dir, "bounds_harness.c")
			if err := os.WriteFile(harness, []byte("#include \""+base+".generated.h\"\nint main(void) { return "+tc.call+"; }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			executable := filepath.Join(dir, "bounds.exe")
			generated := filepath.Join(dir, base+".generated.c")
			if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harness, "-o", executable).CombinedOutput(); err != nil {
				t.Fatalf("native C11 compile failed: %v\n%s", err, out)
			}
			out, err := exec.Command(executable).CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.reason) || !strings.Contains(string(out), "Concept panic at") {
				t.Fatalf("bounds panic evidence missing: err=%v output=%s", err, out)
			}
		})
	}
}
