package concept

import (
	"bytes"
	"strings"
	"testing"
)

const tableRepetitionSpecimen = `module TableRepetition;
profile Core;

table<4> Point
{
    float x;
    float y;
    int id;
}

record table<2> Sample
{
    int value;
}

ref table<2> SampleView
{
    int value;
}

template <typename T, usize N>
table<N> Samples
{
    T value;
}

static_assert(SizeOf<Point>() == 48, "fixed table is structure-of-arrays");
static_assert(AlignOf<Point>() == 4, "table alignment follows its columns");
static_assert(SizeOf<Samples<int, 3>>() == 12, "generic table cardinality closes");
comptime Sample FirstSample = Sample{[1, 2]};
comptime Sample SecondSample = Sample{[1, 2]};
static_assert(FirstSample == SecondSample, "record table reuses record equality");

template <usize N>
int<array>[N] RepeatedZeros()
{
    return [0 ... N];
}

int Next(ref int counter)
{
    counter = counter + 1;
    return counter;
}

int Check()
{
	Point points = Point{[0.0 ...], [1.0 ... 4], [-1 ...]};
	points.x[2] = 4.0;
	Span<float> xs = Span(points.x);
	SampleView view = SampleView{[8 ...]};
	int<array>[8] values = [1 ... 2, 2 ... 3, 3 ...];
	int<array>[4] zeroes = RepeatedZeros<4>();
	int<array>[2] zeroRepeat = [9 ... 0, 5, 6];
    int count = 0;
    int<array>[4] constructed = [Next(ref count) ... 4];
    Samples<int, 3> samples = Samples<int, 3>{[7 ...]};
	return if (xs[2] == 4.0 and view.value[1] == 8 and values[7] == 3 and zeroes[3] == 0 and zeroRepeat[0] == 5 and constructed[3] == 4 and samples.value[2] == 7) 1 else 0;
}

`

func TestLargeRepetitionKeepsCompactASTMIRAndC(t *testing.T) {
	source := `profile Core;
table<100000> Bulk { int value; }
int Check()
{
    Bulk bulk = Bulk{[0 ... 100000]};
    return bulk.value[99999];
}
`
	module, err := Parse("large_repetition.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	fn := module.Functions[0]
	decl := fn.Body.Statements[0].(*VarDecl)
	construct := decl.Value.(*StructConstructExpr)
	literal := construct.Args[0].(*ArrayLiteralExpr)
	if len(literal.Elements) != 1 {
		t.Fatalf("large repetition materialized %d AST elements", len(literal.Elements))
	}
	if repeat := literal.Elements[0].(*RepeatInitializer); repeat.ResolvedCount != 100000 {
		t.Fatalf("resolved repetition count = %d", repeat.ResolvedCount)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := moduleOutput(t, outputs, ".generated.c")
	if len(body) > 20000 || !strings.Contains(body, "< 100000") {
		t.Fatalf("large repetition lowering was not compact: bytes=%d\n%s", len(body), body)
	}
}

func TestFixedTablesAndInitializerRepetitionRunThroughStrictC11(t *testing.T) {
	module, err := Parse("table_repetition.concept", tableRepetitionSpecimen)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Structs) < 4 || !module.Structs[0].Table || !module.Structs[0].TableSized || !module.Structs[1].Record || !module.Structs[2].Ref {
		t.Fatalf("table modifiers/cardinality were not retained: %#v", module.Structs)
	}
	outputs, err := Generate(module, []byte(tableRepetitionSpecimen))
	if err != nil {
		t.Fatal(err)
	}
	header := moduleOutput(t, outputs, ".generated.h")
	if !strings.Contains(header, "concept_array_4_float x;") || !strings.Contains(header, "concept_array_4_int id;") {
		t.Fatalf("table did not lower as contiguous columns:\n%s", header)
	}
	mir := moduleOutput(t, outputs, ".mir.json")
	if !strings.Contains(mir, "repeat_initializer") || !strings.Contains(mir, "evaluation=Independent") {
		t.Fatalf("compact repetition MIR evidence missing:\n%s", mir)
	}
	harness := "#include \"table_repetition.generated.h\"\nint main(void) { return concept_table_repetition_check() == 1 ? 0 : 1; }\n"
	runFoundationNativeHarness(t, outputs, "table_repetition_harness.c", harness)
}

func TestInitializerRepetitionDiagnostics(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		family string
	}{
		{"overflow", "int<array>[2] values = [1 ... 3];", "INITIALIZER_TOO_LARGE"},
		{"negative", "int<array>[2] values = [1 ... -1];", "REPETITION_COUNT_NEGATIVE"},
		{"runtime", "int n = 2; int<array>[2] values = [1 ... n];", "REPETITION_COUNT_NOT_COMPTIME"},
		{"move", "owned Resource value = Resource{1}; owned Resource<array>[2] values = [move value ... 2];", "REPETITION_OWNERSHIP_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "profile Core; struct Resource { int value; } int Main() { " + tc.body + " return 0; }"
			_, err := Parse(tc.name+".concept", source)
			if err == nil || !strings.Contains(err.Error(), tc.family) {
				t.Fatalf("diagnostic = %v, want %s", err, tc.family)
			}
		})
	}
}

func TestInvalidTableCardinalityHasDirectedDiagnostic(t *testing.T) {
	_, err := Parse("invalid_table.concept", "profile Core; table<-1> Invalid { int value; }")
	if err == nil || !strings.Contains(err.Error(), "TABLE_CARDINALITY_INVALID") {
		t.Fatalf("diagnostic = %v", err)
	}
}

func TestBareEllipsisRequiresKnownExtent(t *testing.T) {
	_, err := Parse("unknown_remainder.concept", "profile Core; int Main() { auto values = [1 ...]; return 0; }")
	if err == nil || !strings.Contains(err.Error(), "REPETITION_COUNT_NOT_INFERABLE") {
		t.Fatalf("diagnostic = %v", err)
	}
}

func TestTableSemanticModuleArtifactAndConsumerAreDeterministic(t *testing.T) {
	producer := `module Geometry.Columns;
profile Core;
template <typename T, usize N>
table<N> Samples { T value; }
table<4> Point { float x; float y; }
`
	first := buildSemanticArtifact(t, "Geometry/Columns.concept", producer, nil)
	for run := 1; run < 100; run++ {
		if next := buildSemanticArtifact(t, "Geometry/Columns.concept", producer, nil); !bytes.Equal(first, next) {
			t.Fatalf("table artifact changed on run %d", run)
		}
	}
	_, semantic, err := LoadSemanticModuleArtifact(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(semantic.Structs) != 1 || !semantic.Structs[0].Table || semantic.Structs[0].TableCardinalityExpression != "4" {
		t.Fatalf("artifact omitted table identity/cardinality: %#v", semantic.Structs)
	}
	consumer := `module App;
profile Core;
import Geometry.Columns;
static_assert(SizeOf<Point>() == 32, "imported table layout");
static_assert(SizeOf<Samples<int, 3>>() == 12, "imported generic table layout");
int Main()
{
    Point point = Point{[1.0 ...], [2.0 ...]};
    return if (point.y[3] == 2.0) 0 else 1;
}
`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Geometry.Columns": first})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "table_artifact_harness.c", "#include \"app.generated.h\"\nint main(void) { return concept_app_main(); }\n")
}

func TestTableRepetitionOutputsAreByteIdenticalAcross100Runs(t *testing.T) {
	want := make(map[string][]byte)
	for run := 0; run < 100; run++ {
		module, err := Parse("table_repetition.concept", tableRepetitionSpecimen)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(tableRepetitionSpecimen))
		if err != nil {
			t.Fatal(err)
		}
		for _, suffix := range []string{".mir.json", ".generated.c", ".manifest.json"} {
			got := []byte(moduleOutput(t, outputs, suffix))
			if run == 0 {
				want[suffix] = append([]byte(nil), got...)
				continue
			}
			if !bytes.Equal(want[suffix], got) {
				t.Fatalf("%s changed on run %d", suffix, run)
			}
		}
	}
}
