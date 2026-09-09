package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type valuesRecordsConformanceCase struct {
	name             string
	source           string
	expected         ConformanceExpectation
	reference        string
	accepted         bool
	diagnosticFamily string
}

var valuesRecordsConformanceCases = []valuesRecordsConformanceCase{
	{name: "mutable struct field assignment", source: "valid/struct_mutable_field.concept", expected: ConformancePass, reference: "language/phase7-runtime-structs/valid/field_assignment_run.valid.conception", accepted: true},
	{name: "whole-value struct copy", source: "valid/struct_value_copy.concept", expected: ConformanceExpectedDivergence, reference: "language/phase10-ownership/invalid/implicit_copy_struct_local.invalid.conception", accepted: true},
	{name: "whole-value struct assignment", source: "valid/struct_value_assignment.concept", expected: ConformanceExpectedDivergence, reference: "language/phase10-ownership/invalid/implicit_copy_struct_assignment.invalid.conception", accepted: true},
	{name: "const copy to mutable", source: "valid/const_copy_to_mutable.concept", expected: ConformancePass, reference: "EVT1 R2 const place decision", accepted: true},
	{name: "record construction", source: "valid/record_construct.concept", expected: ConformancePass, reference: "EVT1-new record syntax", accepted: true},
	{name: "record copy", source: "valid/record_copy.concept", expected: ConformancePass, reference: "EVT1-new record syntax", accepted: true},
	{name: "record binding assignment", source: "valid/record_binding_assignment.concept", expected: ConformancePass, reference: "EVT1-new record syntax and R2 binding/type distinction", accepted: true},
	{name: "record with update", source: "valid/record_with_update.concept", expected: ConformancePass, reference: "EVT1-new with syntax", accepted: true},
	{name: "record structural equality", source: "valid/record_structural_equality.concept", expected: ConformancePass, reference: "existing bounded structural equality plus EVT1-new record syntax", accepted: true},
	{name: "immovable final storage mutation", source: "valid/immovable_final_storage_mutation.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", accepted: true},
	{name: "const parameter read", source: "valid/const_parameter_read.concept", expected: ConformancePass, reference: "EVT1 R2 const place decision", accepted: true},
	{name: "let exact const alias", source: "valid/let_const_alias.concept", expected: ConformancePass, reference: "EVT1 R2 optional alias", accepted: true},
	{name: "Vulkan inherits record semantics", source: "valid/record_with_update_vulkan.concept", expected: ConformancePass, reference: "EVT1 R2 profile-boundary decision", accepted: true},

	{name: "const local reassignment", source: "invalid/const_reassign.concept", expected: ConformancePass, reference: "EVT1 R2 const place decision", diagnosticFamily: "CONST_MUTATION"},
	{name: "const struct field mutation", source: "invalid/const_field_mutation.concept", expected: ConformancePass, reference: "EVT1 R2 const place decision", diagnosticFamily: "CONST_MUTATION"},
	{name: "const parameter mutation", source: "invalid/const_parameter_mutation.concept", expected: ConformancePass, reference: "EVT1 R2 const parameter decision", diagnosticFamily: "CONST_MUTATION"},
	{name: "record field mutation", source: "invalid/record_field_mutation.concept", expected: ConformancePass, reference: "EVT1-new record syntax", diagnosticFamily: "RECORD_FIELD_MUTATION"},
	{name: "with on struct", source: "invalid/record_with_on_struct.concept", expected: ConformancePass, reference: "EVT1-new with syntax", diagnosticFamily: "WITH_REQUIRES_RECORD"},
	{name: "with on non-record", source: "invalid/record_with_on_non_record.concept", expected: ConformancePass, reference: "EVT1-new with syntax", diagnosticFamily: "WITH_REQUIRES_RECORD"},
	{name: "unknown with field", source: "invalid/record_with_unknown_field.concept", expected: ConformancePass, reference: "language/phase7-runtime-structs/invalid/struct_literal_unknown_field.invalid.conception plus EVT1-new with syntax", diagnosticFamily: "UNKNOWN_FIELD"},
	{name: "duplicate with field", source: "invalid/record_with_duplicate_field.concept", expected: ConformancePass, reference: "language/phase7-runtime-structs/invalid/struct_literal_duplicate_field.invalid.conception plus EVT1-new with syntax", diagnosticFamily: "DUPLICATE_WITH_FIELD"},
	{name: "with field type mismatch", source: "invalid/record_with_type_mismatch.concept", expected: ConformancePass, reference: "language/phase7-runtime-structs/invalid/struct_literal_type_mismatch.invalid.conception plus EVT1-new with syntax", diagnosticFamily: "WITH_FIELD_TYPE_MISMATCH"},
	{name: "with non-copyable record", source: "invalid/record_with_noncopyable.concept", expected: ConformancePass, reference: "language/phase10-ownership/invalid/implicit_copy_noncopy_field_argument.invalid.conception plus EVT1-new with syntax", diagnosticFamily: "WITH_NONCOPYABLE_RECORD"},
	{name: "immovable copy", source: "invalid/immovable_copy.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_COPY"},
	{name: "immovable assignment", source: "invalid/immovable_assignment.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_ASSIGNMENT"},
	{name: "immovable parameter", source: "invalid/immovable_parameter_by_value.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_BY_VALUE_PARAMETER"},
	{name: "immovable return", source: "invalid/immovable_return_by_value.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_BY_VALUE_RETURN"},
	{name: "immovable embedding", source: "invalid/immovable_embed_by_value.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_EMBEDDING"},
	{name: "immovable record embedding", source: "invalid/immovable_record_embed_by_value.concept", expected: ConformancePass, reference: "EVT1-new record syntax plus Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_EMBEDDING"},
	{name: "immovable enum payload", source: "invalid/immovable_enum_payload_by_value.concept", expected: ConformancePass, reference: "Concept Vulkan constitution M1B-A", diagnosticFamily: "IMMOVABLE_ENUM_PAYLOAD"},
	{name: "mutable field type mismatch", source: "invalid/struct_field_type_mismatch.concept", expected: ConformancePass, reference: "language/phase7-runtime-structs/invalid/field_assignment_type_mismatch.invalid.conception", diagnosticFamily: "VALUE_TYPE_MISMATCH"},
	{name: "let requires explicit type", source: "invalid/let_without_type.concept", expected: ConformancePass, reference: "EVT1 R2 optional alias", diagnosticFamily: "EXPLICIT_LOCAL_TYPE_REQUIRED"},
}

func TestValuesRecordsDifferentialConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range valuesRecordsConformanceCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reference == "" {
				t.Fatal("conformance provenance is required")
			}
			path := filepath.Join("..", "..", "language", "evt1", "values", filepath.FromSlash(tc.source))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if !tc.accepted {
				if err == nil {
					t.Fatal("EVT1 accepted a case whose semantic oracle requires rejection")
				}
				var diagnostic Diagnostic
				if !errors.As(err, &diagnostic) {
					t.Fatalf("rejection was not a stable diagnostic: %v", err)
				}
				if diagnostic.SemanticCategory() != tc.diagnosticFamily {
					t.Fatalf("diagnostic family = %s, want %s: %v", diagnostic.SemanticCategory(), tc.diagnosticFamily, err)
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
			if len(outputs) == 0 {
				t.Fatal("accepted case produced no artifacts")
			}
		})
		counts[tc.expected]++
	}
	t.Logf("R2 conformance summary: PASS=%d EXPECTED-DIVERGENCE=%d EVT1-GAP=%d POC3-LEGACY=%d SPEC-AMBIGUITY=%d",
		counts[ConformancePass], counts[ConformanceExpectedDivergence], counts[ConformanceEVT1Gap], counts[ConformancePoC3Legacy], counts[ConformanceSpecAmbiguity])
}

func TestValuesRecordsConformanceClassificationCounts(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range valuesRecordsConformanceCases {
		counts[tc.expected]++
	}
	if counts[ConformancePass] != 30 || counts[ConformanceExpectedDivergence] != 2 || len(counts) != 2 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestValuesRecordsRecordMIRAndLowering(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "values", "valid", "record_with_update.concept")
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
	var mir MIR
	if err := json.Unmarshal(outputs["record_with_update.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Structs) != 1 || !mir.Structs[0].Record || !mir.Structs[0].Copyable {
		t.Fatalf("record metadata missing from MIR: %+v", mir.Structs)
	}
	foundWith := false
	for _, operation := range mir.Functions[0].Operations {
		foundWith = foundWith || operation.Kind == "record_with"
	}
	if !foundWith {
		t.Fatal("record_with operation missing from MIR")
	}
	cBody := string(outputs["record_with_update.generated.c"])
	if !strings.Contains(cBody, "record_with") || !strings.Contains(cBody, ".x = 10;") {
		t.Fatalf("fresh record update lowering missing:\n%s", cBody)
	}
}

func TestValuesRecordsSyntaxSpans(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "values", "valid", "record_with_update.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	if got := module.Structs[0].RecordSpan; got != (Span{Line: 3, Column: 1}) {
		t.Fatalf("record span = %+v", got)
	}
	decl := module.Functions[0].Body.Statements[1].(*VarDecl)
	if got := decl.ConstSpan; got != (Span{Line: 12, Column: 5}) {
		t.Fatalf("const span = %+v", got)
	}
	with := decl.Value.(*WithExpr)
	if with.Span != (Span{Line: 12, Column: 32}) {
		t.Fatalf("with span = %+v", with.Span)
	}
	if with.Updates[0].NameSpan != (Span{Line: 14, Column: 9}) || with.Updates[0].Value.exprSpan() != (Span{Line: 14, Column: 13}) {
		t.Fatalf("update spans = name %+v value %+v", with.Updates[0].NameSpan, with.Updates[0].Value.exprSpan())
	}
}

func TestValuesRecordsWithBaseIsLoweredOnce(t *testing.T) {
	source := []byte(`profile Core;

record struct Point
{
    int x;
};

Point Make()
{
    return Point{1};
}

int Main()
{
    Point moved = Make() with
    {
        x = 2;
    };
    return moved.x;
}
`)
	module, err := Parse("with_base_once.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	cBody := string(outputs["with_base_once.generated.c"])
	if strings.Count(cBody, "concept_with_base_once_make()") != 2 {
		t.Fatalf("Make definition plus exactly one evaluation expected:\n%s", cBody)
	}
}

func TestValuesRecordsNativeC11(t *testing.T) {
	cases := []struct {
		fixture string
		call    string
		want    int
	}{
		{fixture: "struct_mutable_field.concept", call: "concept_struct_mutable_field_main", want: 1},
		{fixture: "record_with_update.concept", call: "concept_record_with_update_main", want: 12},
		{fixture: "immovable_final_storage_mutation.concept", call: "concept_immovable_final_storage_mutation_main", want: 7},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "values", "valid", tc.fixture)
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
			base := strings.TrimSuffix(tc.fixture, ".concept")
			harness := "#include \"" + base + ".generated.h\"\n\nint main(void) { return " + tc.call + "() == " + strconv.Itoa(tc.want) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}
