package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var evt1R4fConformanceCases = []ConformanceCase{
	{Name: "fixed layout", Source: "valid/layout_fixed_basic.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d fixed storage", MatrixStatus: "EVT1-new"},
	{Name: "array regions", Source: "valid/layout_array_regions.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d arrays", MatrixStatus: "EVT1-new"},
	{Name: "ndarray regions", Source: "valid/layout_ndarray_regions.concept", Expected: ConformancePass, Accepted: true, Reference: "R4d ndarrays", MatrixStatus: "EVT1-new"},
	{Name: "alignment", Source: "valid/layout_alignment.concept", Expected: ConformancePass, Accepted: true, Reference: "explicit geometry", MatrixStatus: "EVT1-new"},
	{Name: "fixed bind", Source: "valid/layout_bind_fixed_storage.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e bind", MatrixStatus: "EVT1-new composition"},
	{Name: "region projection", Source: "valid/layout_region_projection.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 ref", MatrixStatus: "EVT1-new composition"},
	{Name: "const projection", Source: "valid/layout_const_projection.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 ref const", MatrixStatus: "EVT1-new composition"},
	{Name: "layout queries", Source: "valid/layout_static_assert_offsets.concept", Expected: ConformancePass, Accepted: true, Reference: "R4c static_assert", MatrixStatus: "EVT1-new composition"},
	{Name: "explicit offsets", Source: "valid/layout_explicit_offsets.concept", Expected: ConformancePass, Accepted: true, Reference: "explicit geometry", MatrixStatus: "EVT1-new"},
	{Name: "basic stream", Source: "valid/stream_basic.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL stream lineage", MatrixStatus: "SDSL-derived"},
	{Name: "multiple channels", Source: "valid/stream_multiple_channels.concept", Expected: ConformancePass, Accepted: true, Reference: "SDSL stream lineage", MatrixStatus: "SDSL-derived"},
	{Name: "stream bind", Source: "valid/stream_bind_layout.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e bind", MatrixStatus: "EVT1-new composition"},
	{Name: "channel alias", Source: "valid/stream_channel_alias.concept", Expected: ConformancePass, Accepted: true, Reference: "R4e no-copy bind", MatrixStatus: "SDSL-derived"},
	{Name: "const channel", Source: "valid/stream_const_projection.concept", Expected: ConformancePass, Accepted: true, Reference: "R3 ref const", MatrixStatus: "EVT1-new composition"},
	{Name: "stream provenance", Source: "valid/stream_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4b provenance", MatrixStatus: "EVT1-new composition"},
	{Name: "stream scoped provenance", Source: "valid/stream_scoped_provenance.concept", Expected: ConformancePass, Accepted: true, Reference: "R4b scoped provenance", MatrixStatus: "EVT1-new composition"},
	{Name: "duplicate region", Source: "invalid/layout_duplicate_region.concept", Expected: ConformancePass, DiagnosticCategory: "CV4571", MatrixStatus: "EVT1-new"},
	{Name: "bad alignment", Source: "invalid/layout_bad_alignment.concept", Expected: ConformancePass, DiagnosticCategory: "CV4572", MatrixStatus: "EVT1-new"},
	{Name: "overlap", Source: "invalid/layout_overlap.concept", Expected: ConformancePass, DiagnosticCategory: "CV4576", MatrixStatus: "EVT1-new"},
	{Name: "runtime layout deferred", Source: "invalid/layout_runtime_parameterized.concept", Expected: ConformancePass, DiagnosticCategory: "CV4575", MatrixStatus: "Deferred"},
	{Name: "bind size mismatch", Source: "invalid/layout_bind_size_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4590", MatrixStatus: "EVT1-new"},
	{Name: "bind alignment mismatch", Source: "invalid/layout_bind_alignment_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4591", MatrixStatus: "EVT1-new"},
	{Name: "unknown stream region", Source: "invalid/stream_unknown_region.concept", Expected: ConformancePass, DiagnosticCategory: "CV4582", MatrixStatus: "SDSL-derived"},
	{Name: "unknown stream layout", Source: "invalid/stream_unknown_layout.concept", Expected: ConformancePass, DiagnosticCategory: "CV4581", MatrixStatus: "SDSL-derived"},
	{Name: "duplicate stream channel", Source: "invalid/stream_duplicate_channel.concept", Expected: ConformancePass, DiagnosticCategory: "CV4583", MatrixStatus: "SDSL-derived"},
	{Name: "stream layout mismatch", Source: "invalid/stream_layout_mismatch.concept", Expected: ConformancePass, DiagnosticCategory: "CV4585", MatrixStatus: "EVT1-new"},
	{Name: "stream mutable from const", Source: "invalid/stream_mutable_from_const.concept", Expected: ConformancePass, DiagnosticCategory: "CV4586", MatrixStatus: "EVT1-new composition"},
	{Name: "const stream channel mutation", Source: "invalid/stream_const_channel_mutation.concept", Expected: ConformancePass, DiagnosticCategory: "CV4513", MatrixStatus: "EVT1-new composition"},
	{Name: "stream lifetime escape", Source: "invalid/stream_escape_source_lifetime.concept", Expected: ConformancePass, DiagnosticCategory: "CV4511", MatrixStatus: "EVT1-new composition"},
}

func TestEVT1R4fConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range evt1R4fConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1-r4f", "core", filepath.FromSlash(tc.Source))
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

func TestEVT1R4fMIRSemanticGraphs(t *testing.T) {
	outputs := generateR4fFixture(t, "valid", "stream_multiple_channels.concept")
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
	if len(mir.Layouts) != 1 || mir.Layouts[0].Size != 32 || mir.Layouts[0].Alignment != 4 || len(mir.Layouts[0].Regions) != 3 {
		t.Fatalf("layout graph = %+v", mir.Layouts)
	}
	for _, region := range mir.Layouts[0].Regions {
		if region.ID == "" || region.ByteExtent == 0 || len(region.DisjointWith) != 2 {
			t.Fatalf("region facts = %+v", region)
		}
	}
	if len(mir.Streams) != 1 || !mir.Streams[0].ZeroStorage || len(mir.Streams[0].Channels) != 3 {
		t.Fatalf("stream graph = %+v", mir.Streams)
	}
	var layoutBind, streamBind, channelProjection bool
	for _, operation := range mir.Functions[0].Operations {
		layoutBind = layoutBind || operation.Kind == "layout_bind"
		streamBind = streamBind || operation.Kind == "stream_bind"
		if operation.Kind == "region_projection" {
			channelProjection = true
			if operation.LayoutName != "VertexLayout" || operation.RegionID == "" || !operation.SameBackingRegion {
				t.Fatalf("channel projection lost layout identity: %+v", operation)
			}
		}
	}
	if !layoutBind || !streamBind || !channelProjection {
		t.Fatalf("missing semantic operations: %+v", mir.Functions[0].Operations)
	}
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "memcpy", "vtable", "Span"} {
		if strings.Contains(emitted.String(), forbidden) {
			t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
		}
	}
	if strings.Contains(emitted.String(), "typedef struct { unsigned char* data; } concept_ref_mut_stream_") {
		t.Fatal("stream unexpectedly emitted independent runtime storage")
	}

	scopedOutputs := generateR4fFixture(t, "valid", "stream_scoped_provenance.concept")
	var scopedMIR MIR
	for name, artifact := range scopedOutputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(artifact, &scopedMIR); err != nil {
				t.Fatal(err)
			}
		}
	}
	foundScopedBind := false
	for _, operation := range scopedMIR.Functions[0].Operations {
		if operation.Kind == "stream_bind" && operation.Provenance == "parameter:scoped" && operation.SameBackingRegion {
			foundScopedBind = true
		}
	}
	if !foundScopedBind {
		t.Fatalf("scoped stream binding lost provenance: %+v", scopedMIR.Functions[0].Operations)
	}
}

func generateR4fFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r4f", "core", class, file)
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

func TestEVT1R4fNativeC11(t *testing.T) {
	cases := []struct {
		file string
		call string
		want int
	}{
		{"layout_ndarray_regions.concept", "concept_layout_ndarray_regions_use_matrix_layout()", 6},
		{"layout_alignment.concept", "concept_layout_alignment_use_aligned_layout()", 11},
		{"layout_explicit_offsets.concept", "concept_layout_explicit_offsets_write_payload_length()", 37},
		{"layout_region_projection.concept", "concept_layout_region_projection_project_regions()", 12},
		{"stream_channel_alias.concept", "concept_stream_channel_alias_stream_channel_aliases_region()", 17},
		{"stream_const_projection.concept", "concept_stream_const_projection_read_const_stream()", 7},
		{"stream_provenance.concept", "concept_stream_provenance_use_provenance()", 21},
		{"stream_scoped_provenance.concept", "concept_stream_scoped_provenance_use_scoped_provenance()", 34},
		{"stream_scoped_provenance.concept", "concept_stream_scoped_provenance_use_scoped_provenance()", 34},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := generateR4fFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\nint main(void) { return " + tc.call + " == " + fmt.Sprint(tc.want) + " ? 0 : 1; }\n"
			runR1NativeHarness(t, outputs, "r4f_harness.c", harness)
		})
	}
	outputs := generateR4fFixture(t, "valid", "layout_alignment.concept")
	harness := "#include \"layout_alignment.generated.h\"\nint main(void) { return _Alignof(concept_array_16_int) >= 64 ? 0 : 1; }\n"
	runR1NativeHarness(t, outputs, "r4f_alignment_harness.c", harness)
}
