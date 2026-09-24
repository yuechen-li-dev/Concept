package concept

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestR7h2GeneratedCodecsReadAndWriteSharedOctBytes(t *testing.T) {
	octRoot := filepath.Join("..", "..", "..", "oct")
	files := map[string]string{
		"array":   filepath.Join("..", "..", "tests", "fixtures", "octagon", "array_holder.octagon"),
		"nested":  filepath.Join("..", "..", "tests", "fixtures", "octagon", "nested_array_holder.octagon"),
		"payload": filepath.Join(octRoot, "Language", "Data", "Octagon", "valid", "payload_enum_nested.octagon"),
		"table":   filepath.Join(octRoot, "Language", "Data", "Octagon", "Load", "valid", "concept_catalog.octagon"),
		"refined": filepath.Join("..", "..", "tests", "fixtures", "octagon", "positive.octagon"),
	}
	data := map[string][]byte{}
	for name, path := range files {
		value, err := os.ReadFile(path)
		if os.IsNotExist(err) && (name == "payload" || name == "table") {
			t.Skip("adjacent Oct fixtures are unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		data[name] = value
	}
	artifacts := map[string][]byte{}
	artifacts["Standard.Octagon.Core"] = buildSemanticArtifact(t,
		"Standard/Octagon/Core.concept", standardMemorySource(t, "Standard/Octagon/Core.concept"), artifacts)
	artifacts["Standard.Octagon.Derive"] = buildSemanticArtifact(t,
		"Standard/Octagon/Derive.concept", standardMemorySource(t, "Standard/Octagon/Derive.concept"), artifacts)
	const source = `module R7h2Interop; profile Core;
import Standard.Octagon.Core; import Standard.Octagon.Derive;
record struct ArrayHolder { int<array>[3] Values; }
type Row = int<array>[2];
record struct NestedArrayHolder { Row<array>[3] Values; }
enum Inner { Value(int number), }
enum Outer { Wrapped(Inner inner), }
record table<3> Catalog { int ID; bool Active; }
record struct Positive { int Value; }
Result<Positive, OctagonError> AdmitOctagon(int raw, OctagonType<Positive> type) {
    if (raw <= 0) { return Result::Error(OctagonError::UnexpectedData); }
    return Result::Ok(Positive{Value = raw});
}
derive DeriveOctagonRead reflect<ArrayHolder>;
derive DeriveOctagonWrite reflect<ArrayHolder>;
derive DeriveOctagonRead reflect<NestedArrayHolder>;
derive DeriveOctagonWrite reflect<NestedArrayHolder>;
derive DeriveOctagonEnumRead reflect<Inner>;
derive DeriveOctagonEnumWrite reflect<Inner>;
derive DeriveOctagonEnumRead reflect<Outer>;
derive DeriveOctagonEnumWrite reflect<Outer>;
derive DeriveOctagonRead reflect<Catalog>;
derive DeriveOctagonWrite reflect<Catalog>;
derive DeriveOctagonRefinedRead reflect<Positive>;
derive DeriveOctagonRefinedWrite reflect<Positive>;
int Main() {
    int total = 0;
{{shapes}}
    return total;
}
`
	type shape struct{ name, typeName, semantic string }
	shapes := []shape{
		{"array", "ArrayHolder", "value.Values[0] + value.Values[2]"},
		{"nested", "NestedArrayHolder", "value.Values[2][1]"},
		{"payload", "Outer", "match (value) { Outer::Wrapped(inner) => match (inner) { Inner::Value(n) => n, }, }"},
		{"table", "Catalog", "value.ID[1]"},
		{"refined", "Positive", "value.Value"},
	}
	var body strings.Builder
	for _, item := range shapes {
		literal := make([]string, len(data[item.name]))
		for i, octet := range data[item.name] {
			literal[i] = strconv.Itoa(int(octet))
		}
		body.WriteString("    {\n")
		body.WriteString("        byte<array>[" + strconv.Itoa(len(literal)) + "] input = [" + strings.Join(literal, ", ") + "];\n")
		body.WriteString("        OctagonReader reader = OctagonReader{Span(input), 0};\n")
		body.WriteString("        " + item.typeName + " value = ReadOctagon(ref reader, OctagonType<" + item.typeName + ">{0})!;\n")
		body.WriteString("        total = total + (" + item.semantic + ");\n")
		body.WriteString("        byte<array>[256] output = [0 ...];\n")
		body.WriteString("        OctagonWriter writer = OctagonWriter{Span(output), 0};\n")
		body.WriteString("        WriteOctagon(ref const value, ref writer)!;\n")
		body.WriteString("        FinishOctagon(ref writer)!;\n")
		body.WriteString("        OctagonReader golden = OctagonReader{Span(output), 0};\n")
		body.WriteString("        ExpectLiteral(ref golden, " + strconv.Quote(string(data[item.name])) + ")!;\n")
		body.WriteString("        if (golden.offset != writer.offset) { return 0; }\n")
		body.WriteString("    }\n")
	}
	program := strings.Replace(source, "{{shapes}}", body.String(), 1)
	module, err := ParseWithSemanticModules("R7h2Interop.concept", program, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(program))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		again, err := ParseWithSemanticModules("R7h2Interop.concept", program, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		current, err := Generate(again, []byte(program))
		if err != nil {
			t.Fatal(err)
		}
		if len(current) != len(outputs) {
			t.Fatalf("generated interop output count changed on run %d", run+1)
		}
		for path, first := range outputs {
			if !bytes.Equal(first, current[path]) {
				t.Fatalf("generated interop output %s changed on run %d", path, run+1)
			}
		}
	}
	runFoundationNativeHarness(t, outputs, "r7h2_interop_harness.c", "#include \"r7h2interop.generated.h\"\nint main(void) { for (int run = 0; run < 100; ++run) { if (concept_r7h2interop_main() != 96) return 1; } return 0; }\n")
}
