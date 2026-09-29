package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFormatCommentsIdempotenceAndSemantics(t *testing.T) {
	source := `// heading
module Format.Core; profile Core;

// declaration
[[must_use]] int Count(int packet_count){
    int value=packet_count; // explanation
    /* before return */ return value;
}
int Run(){// inside
    discard Count(3);
    return 0;
}
`
	opts := DefaultFormatOptions()
	formatted, err := FormatSource("format.concept", source, opts)
	if err != nil {
		t.Fatal(err)
	}
	const golden = `// heading
module Format.Core;
profile Core;

// declaration
[[must_use]] int Count(int packet_count) {
    int value = packet_count; // explanation
    /* before return */
    return value;
}
int Run() {
    // inside
    discard Count(3);
    return 0;
}
`
	if formatted != golden {
		t.Fatalf("canonical golden differs:\n%s", formatted)
	}
	for _, comment := range []string{"// heading", "// declaration", "// explanation", "/* before return */", "// inside"} {
		if strings.Count(formatted, comment) != 1 {
			t.Fatalf("comment %q changed:\n%s", comment, formatted)
		}
	}
	if !strings.Contains(formatted, "Count(int packet_count)") || !strings.Contains(formatted, "int value = packet_count;") {
		t.Fatalf("names or spacing changed unexpectedly:\n%s", formatted)
	}
	for i := 0; i < 100; i++ {
		next, err := FormatSource("format.concept", formatted, opts)
		if err != nil || next != formatted {
			t.Fatalf("run %d: %v\n%s", i, err, next)
		}
	}
	for _, text := range []string{source, formatted} {
		if _, err := Parse("format.concept", text); err != nil {
			t.Fatal(err)
		}
	}
	originalModule, _ := Parse("format.concept", source)
	formattedModule, _ := Parse("format.concept", formatted)
	before, err := Generate(originalModule, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	after, err := Generate(formattedModule, []byte(formatted))
	if err != nil {
		t.Fatal(err)
	}
	var beforeMIR, afterMIR any
	if err := json.Unmarshal(before["format.mir.json"], &beforeMIR); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(after["format.mir.json"], &afterMIR); err != nil {
		t.Fatal(err)
	}
	stripSourceSpans(beforeMIR)
	stripSourceSpans(afterMIR)
	if !reflect.DeepEqual(beforeMIR, afterMIR) {
		t.Fatal("semantic MIR changed after removing refreshed source spans")
	}
	if !bytes.Equal(before["format.generated.c"], after["format.generated.c"]) {
		a, b := before["format.generated.c"], after["format.generated.c"]
		i := 0
		for i < len(a) && i < len(b) && a[i] == b[i] {
			i++
		}
		t.Fatalf("generated C changed at %d: %q vs %q", i, a[max(0, i-60):min(len(a), i+120)], b[max(0, i-60):min(len(b), i+120)])
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		beforeMode, err := GenerateForTargetWithPolicy(originalModule, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		afterMode, err := GenerateForTargetWithPolicy(formattedModule, []byte(formatted), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		if len(beforeMode["format.generated.c"]) == 0 || len(afterMode["format.generated.c"]) == 0 {
			t.Fatal("generated C missing")
		}
		if !bytes.Equal(beforeMode["format.generated.c"], afterMode["format.generated.c"]) {
			t.Fatalf("generated C changed in policy %+v", policy)
		}
	}
}

func stripSourceSpans(value any) {
	switch v := value.(type) {
	case map[string]any:
		delete(v, "span")
		delete(v, "source_span")
		for _, item := range v {
			stripSourceSpans(item)
		}
	case []any:
		for _, item := range v {
			stripSourceSpans(item)
		}
	}
}

func TestFormatProjectScopeAndConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := FormatPath(root); err == nil || !strings.Contains(err.Error(), "FORMAT_MANIFEST_MISSING") {
		t.Fatalf("directory without manifest: %v", err)
	}
	manifest := `module Format.Manifest; profile Core;
comptime int FormatIndentWidth = 2;
comptime int FormatMaxLineLength = 100;
comptime string FormatBraceStyle = "same-line";
`
	if err := os.WriteFile(filepath.Join(root, "manifest.concept"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	program := `module Format.Program; profile Core; int Run(){return 1;}`
	path := filepath.Join(root, "program.concept")
	if err := os.WriteFile(path, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	dependency := filepath.Join(root, "dependency")
	if err := os.Mkdir(dependency, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dependency, "manifest.concept"), []byte("module Dependency.Manifest; profile Core;"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dependency, "source.concept"), []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	edits, err := FormatPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 2 || !strings.Contains(edits[path], "\n  return 1;") {
		t.Fatalf("unexpected project edits: %#v", edits)
	}
	if _, ok := edits[filepath.Join(dependency, "source.concept")]; ok {
		t.Fatal("dependency implementation selected")
	}
}

func TestFormatCurrentSyntaxRoundTrips(t *testing.T) {
	paths := []string{
		"../../language/evt1/units/r8c/valid/scientific_literals.concept",
		"../../language/evt1/units/r8c/valid/tensor_units.concept",
		"../../language/evt1/units/r8d/valid/protocol_interpretation.concept",
		"../../language/evt1/foundation/valid/concepts_templates.concept",
		"../../language/evt1/automata/state/valid/machine_basic_transition.concept",
		"../../language/evt1/tooling/tests/fact_basic.concept_test",
		"../../tests/dogfood/tinyxml2/manifest.concept",
		"../../libraries/Standard/Collection/Core.concept",
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			formatted, err := FormatSource(path, string(body), DefaultFormatOptions())
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 100; i++ {
				again, err := FormatSource(path, formatted, DefaultFormatOptions())
				if err != nil || formatted != again {
					t.Fatalf("run %d not idempotent: %v", i, err)
				}
			}
		})
	}
}

func TestFormatValidCorpusRoundTrips(t *testing.T) {
	count := 0
	err := filepath.WalkDir("../../language/evt1", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.Contains(filepath.ToSlash(path), "/valid/") || !strings.HasSuffix(path, ".concept") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := FormatSource(path, string(body), DefaultFormatOptions())
		if err != nil {
			return err
		}
		again, err := FormatSource(path, formatted, DefaultFormatOptions())
		if err != nil {
			return err
		}
		if formatted != again {
			return fmt.Errorf("%s: formatter is not idempotent", path)
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 100 {
		t.Fatalf("only %d valid corpus sources exercised", count)
	}
	t.Logf("formatted %d valid corpus sources twice", count)
}

func TestFormatBlockCommentsAndInvalidSource(t *testing.T) {
	source := `/* lead
 * detail */
module Block.Core; profile Core;
int Run(){int x=1; /* interior */ return x;} // tail
`
	formatted, err := FormatSource("block.concept", source, DefaultFormatOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"/* lead\n * detail */", "/* interior */", "// tail"} {
		if strings.Count(formatted, comment) != 1 {
			t.Fatalf("comment changed: %q\n%s", comment, formatted)
		}
	}
	for i := 0; i < 100; i++ {
		next, err := FormatSource("block.concept", formatted, DefaultFormatOptions())
		if err != nil || next != formatted {
			t.Fatalf("block comment run %d: %v\n%s", i, err, next)
		}
	}
	for _, invalid := range []string{"module Bad; profile Core; int Run( {", "module Bad; profile Core; /* unfinished"} {
		if _, err := FormatSource("bad.concept", invalid, DefaultFormatOptions()); err == nil {
			t.Fatalf("invalid source formatted: %q", invalid)
		}
	}
}

func TestFormatMalformedConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.concept")
	for _, declaration := range []string{
		`comptime int FormatIndentWidth = 0;`,
		`comptime int FormatMaxLineLength = 20;`,
		`comptime string FormatBraceStyle = "unknown";`,
		`comptime string FormatIndentWidth = "four";`,
	} {
		body := "module Invalid.Manifest; profile Core;\n" + declaration
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadFormatOptions(path); err == nil || !strings.Contains(err.Error(), "FORMAT_CONFIG_INVALID") {
			t.Fatalf("%s: %v", declaration, err)
		}
	}
}

func TestFormatAllmanOption(t *testing.T) {
	options := DefaultFormatOptions()
	options.BraceStyle = "allman"
	formatted, err := FormatSource("allman.concept", "module Brace.Core; profile Core; int Run(){return 0;}", options)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(formatted, "int Run()\n{\n") {
		t.Fatalf("Allman brace missing:\n%s", formatted)
	}
}

func TestFormatSourceTokenAnchors(t *testing.T) {
	source := "// lead\nmodule Anchor.Core; // trailing\nprofile Core;\n"
	doc, err := ParseSourceDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Tokens) == 0 || doc.Tokens[0].Lexeme != "module" || doc.Tokens[0].Leading != "// lead\n" || doc.Tokens[0].Span != (Span{Line: 2, Column: 1}) {
		t.Fatalf("first anchor: %+v", doc.Tokens)
	}
	for _, token := range doc.Tokens {
		if source[token.Start:token.End] != token.Lexeme {
			t.Fatalf("invalid byte anchor: %+v", token)
		}
		if token.Lexeme == "profile" && !strings.Contains(token.Leading, "// trailing") {
			t.Fatalf("trailing comment lost from token gap: %+v", token)
		}
	}
}

func TestFormatMatchMachineCommentOrder(t *testing.T) {
	source := `profile Core;
// signal declaration
enum Signal{Start,Fault(int code),}
automata Controller{
 machine Run{
  state Idle{
   // before transition
   transition match (Signal::Fault(7)){
    Signal::Start => Idle; // start arm
    // fault arm
    Signal::Fault(code) => Failed;
   }
  }
  // between states
  state Failed{}
 }
}

`
	formatted, err := FormatSource("machine.concept", source, DefaultFormatOptions())
	if err != nil {
		t.Fatal(err)
	}
	comments := []string{"// signal declaration", "// before transition", "// start arm", "// fault arm", "// between states"}
	previous := -1
	for _, comment := range comments {
		index := strings.Index(formatted, comment)
		if index <= previous || strings.Count(formatted, comment) != 1 {
			t.Fatalf("comment order changed: %q\n%s", comment, formatted)
		}
		previous = index
	}
	for i := 0; i < 100; i++ {
		next, err := FormatSource("machine.concept", formatted, DefaultFormatOptions())
		if err != nil || next != formatted {
			t.Fatalf("run %d: %v\n%s", i, err, next)
		}
	}
	if _, err := Parse("machine.concept", formatted); err != nil {
		t.Fatal(err)
	}
}

func TestFormatPreservesSemanticMIR(t *testing.T) {
	for _, path := range []string{
		"../../language/evt1/units/r8c/valid/scientific_literals.concept",
		"../../language/evt1/units/r8d/valid/protocol_interpretation.concept",
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			formatted, err := FormatSource(path, string(body), DefaultFormatOptions())
			if err != nil {
				t.Fatal(err)
			}
			var prior any
			for i, text := range []string{string(body), formatted} {
				module, err := Parse(path, text)
				if err != nil {
					t.Fatal(err)
				}
				outputs, err := Generate(module, []byte(text))
				if err != nil {
					t.Fatal(err)
				}
				key := strings.TrimSuffix(filepath.Base(path), ".concept") + ".mir.json"
				var current any
				if err := json.Unmarshal(outputs[key], &current); err != nil {
					t.Fatal(err)
				}
				stripSourceSpans(current)
				if i > 0 && !reflect.DeepEqual(prior, current) {
					t.Fatal("semantic MIR changed")
				}
				prior = current
			}
		})
	}
}

func TestFormatPreservesProofTruth(t *testing.T) {
	path := "../../language/evt1/tooling/proofs/assert_concept_fact.concept_test"
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := FormatSource(path, string(body), DefaultFormatOptions())
	if err != nil {
		t.Fatal(err)
	}
	var prior SemanticFactCertainty
	for i, text := range []string{string(body), formatted} {
		line := strings.Count(text[:strings.Index(text, "Assert.Concept")], "\n") + 1
		graph, err := ExplainSource(path, text, line)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && graph.Outcome != prior {
			t.Fatalf("proof truth changed: %s -> %s", prior, graph.Outcome)
		}
		prior = graph.Outcome
	}
}

func TestFormatterKeepsCompactNaturalForms(t *testing.T) {
	const source = `profile Core;
struct Point { int x; int y; }
int Check(int used, int count) {
    Point p = Point{0, 0};
    used++;
    count--;
    if (used > count) { return p.x; }
    return count;
}`
	formatted, err := FormatSource("compact.concept", source, DefaultFormatOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"Point{0, 0}", "used++;", "count--;", "if (used > count) { return p.x; }"} {
		if !strings.Contains(formatted, needle) {
			t.Fatalf("formatter lost %q:\n%s", needle, formatted)
		}
	}
	if strings.Contains(formatted, "struct Point { int x;") {
		t.Fatalf("declaration was incorrectly compacted:\n%s", formatted)
	}
	again, err := FormatSource("compact.concept", formatted, DefaultFormatOptions())
	if err != nil || again != formatted {
		t.Fatalf("formatter is not stable: %v\n%s", err, again)
	}
}
