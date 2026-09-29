package concept

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func readEVT1Fixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "evt1", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// specimenMIROps names the MIR operation each specimen subject exists to
// exercise; both the Core specimen and its _vulkan counterpart must emit it.
var specimenMIROps = map[string][]string{
	"extern_calls_and_match": {"match_expr"},
	"resource_concepts":      {"struct_construct"},
	"concept_templates":      {"requirement_call"},
	"comptime_tables":        {"array_index"},
	"lifecycle_automata":     {"automata"},
	"automata_dispatch":      {"automata"},
	"guarded_transitions":    {"automata"},
}

func TestParseSpecimensAndGenerateDeterministically(t *testing.T) {
	for _, name := range []string{
		"extern_calls_and_match.concept",
		"extern_calls_and_match_vulkan.concept",
		"resource_concepts.concept",
		"resource_concepts_vulkan.concept",
		"concept_templates.concept",
		"concept_templates_vulkan.concept",
		"comptime_values.concept",
		"comptime_values_vulkan.concept",
		"comptime_tables.concept",
		"comptime_tables_vulkan.concept",
		"lifecycle_automata.concept",
		"automata_dispatch.concept",
		"guarded_transitions.concept",
	} {
		t.Run(name, func(t *testing.T) {
			src := readEVT1Fixture(t, name)
			module, err := Parse(filepath.ToSlash(filepath.Join("examples", "evt1", name)), src)
			if err != nil {
				t.Fatal(err)
			}
			if len(module.Functions) == 0 {
				t.Fatalf("unexpected module: %#v", module)
			}
			outA, err := Generate(module, []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			outB, err := Generate(module, []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			for key, want := range outA {
				if string(want) != string(outB[key]) {
					t.Fatalf("nondeterministic output %s", key)
				}
			}
			env, err := analyzeModule(module)
			if err != nil {
				t.Fatal(err)
			}
			mirText := MIRText(buildMIR(module, env))
			subject := strings.TrimSuffix(strings.TrimSuffix(name, ".concept"), "_vulkan")
			for _, op := range specimenMIROps[subject] {
				if !strings.Contains(mirText, op) {
					t.Fatalf("MIR text omitted %s", op)
				}
			}
			for _, suffix := range []string{".mir.json", ".map.json", ".manifest.json"} {
				found := false
				for key, body := range outA {
					if strings.HasSuffix(key, suffix) {
						found = true
						var decoded any
						if err := json.Unmarshal(body, &decoded); err != nil {
							t.Fatalf("%s: %v", key, err)
						}
					}
				}
				if !found {
					t.Fatalf("missing %s output", suffix)
				}
			}
		})
	}
}

func TestEVT1CheckedOutputsMatch(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"extern_calls_and_match.concept",
		"extern_calls_and_match_vulkan.concept",
		"resource_concepts.concept",
		"resource_concepts_vulkan.concept",
		"concept_templates.concept",
		"concept_templates_vulkan.concept",
		"comptime_values.concept",
		"comptime_values_vulkan.concept",
		"comptime_tables.concept",
		"comptime_tables_vulkan.concept",
		"lifecycle_automata.concept",
		"automata_dispatch.concept",
		"guarded_transitions.concept",
	} {
		t.Run(name, func(t *testing.T) {
			src := readEVT1Fixture(t, name)
			module, err := Parse(filepath.ToSlash(filepath.Join("examples", "evt1", name)), src)
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("CONCEPT_UPDATE_CHECKED_OUTPUTS") == "1" {
				if err := Write(filepath.Join("generated"), outputs); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err := Check(filepath.Join("generated"), outputs); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEVT1DoubleGenerationMatchesAcrossDirectories(t *testing.T) {
	src := readEVT1Fixture(t, "concept_templates_vulkan.concept")
	module, err := Parse("examples/evt1/concept_templates_vulkan.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputsA, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	outputsB, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	dirA := t.TempDir()
	dirB := t.TempDir()
	if err := Write(dirA, outputsA); err != nil {
		t.Fatal(err)
	}
	if err := Write(dirB, outputsB); err != nil {
		t.Fatal(err)
	}
	for name := range outputsA {
		a, err := os.ReadFile(filepath.Join(dirA, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dirB, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Fatalf("directory generation mismatch for %s", name)
		}
	}
}

func TestEVT1CGenerationUsesTransparentStructsAndNoConceptRuntime(t *testing.T) {
	src := readEVT1Fixture(t, "concept_templates.concept")
	module, err := Parse("examples/evt1/concept_templates.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["concept_templates.generated.h"])
	body := string(outputs["concept_templates.generated.c"])
	for _, needle := range []string{
		"typedef struct concept_buffer_range {\n  int32_t bufferId;\n  int32_t offset;\n  int32_t size;\n}",
		"typedef struct concept_pipeline_state {\n  int32_t handle;\n  bool alive;\n}",
	} {
		if !strings.Contains(header, needle) {
			t.Fatalf("header missing %q\n%s", needle, header)
		}
	}
	for _, needle := range []string{
		"static int32_t concept_template_score_resource__buffer_range(",
		"static void concept_template_destroy_resource__pipeline_state(",
		"concept_concept_templates_measure__borrow_const_buffer_range",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q\n%s", needle, body)
		}
	}
	for _, forbidden := range []string{"template <", "VulkanResource", "Measurable", "Destroyable"} {
		if strings.Contains(header, forbidden) || strings.Contains(body, forbidden) {
			t.Fatalf("concept runtime name %q leaked into generated C", forbidden)
		}
	}
}

func TestEVT1DiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "duplicate fields",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; int offset; };\n",
			code: "CV4124",
		},
		{
			name: "wrong initializer count",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; int size; };\nBufferRange Make() { BufferRange range = BufferRange{1}; return range; }\n",
			code: "CV4126",
		},
		{
			name: "wrong initializer type",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; bool ok; };\nBufferRange Make() { BufferRange range = BufferRange{1, 2}; return range; }\n",
			code: "CV4107",
		},
		{
			name: "unknown field",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nint Read(BufferRange range) { return range.missing; }\n",
			code: "CV4026",
		},
		{
			name: "const borrow mutation",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nvoid Mutate(borrow const BufferRange range) { range.offset = 1; }\n",
			code: "CV4128",
		},
		{
			name: "ownership illegal copy",
			src:  "profile Vulkan;\nstruct HandleBox { owned Pipeline pipeline; };\nPipeline Acquire();\nvoid Use() { HandleBox first = HandleBox{Acquire()}; HandleBox second = first; }\n",
			code: "CV4133",
		},
		{
			name: "immovable copy",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nvoid Use() { PoolState first = PoolState{1, false}; PoolState second = first; }\n",
			code: "CV4134",
		},
		{
			name: "immovable whole value assignment",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nvoid Use() { PoolState first = PoolState{1, false}; PoolState second = PoolState{2, false}; second = first; }\n",
			code: "CV4135",
		},
		{
			name: "immovable by value parameter",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nvoid Use(PoolState state);\n",
			code: "CV4136",
		},
		{
			name: "immovable by value return",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nPoolState Use();\n",
			code: "CV4137",
		},
		{
			name: "immovable embedding",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nstruct Wrapper { PoolState state; };\n",
			code: "CV4138",
		},
		{
			name: "immovable enum payload",
			src:  "profile Vulkan;\nimmovable struct PoolState { int id; bool ok; };\nenum Event { Ready(PoolState state) }\n",
			code: "CV4139",
		},
		{
			name: "unknown concept",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nrequires Missing<BufferRange>;\n",
			code: "CV4151",
		},
		{
			name: "unknown prerequisite",
			src:  "profile Vulkan;\nconcept UsesMissing<T> { requires Missing<T>; }\n",
			code: "CV4152",
		},
		{
			name: "missing required operation",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nrequires Validatable<BufferRange>;\n",
			code: "CV4153",
		},
		{
			name: "wrong operation parameter type",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nbool IsValid(int value);\nrequires Validatable<BufferRange>;\n",
			code: "CV4154",
		},
		{
			name: "wrong operation const qualifier",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nbool IsValid(borrow BufferRange value);\nrequires Validatable<BufferRange>;\n",
			code: "CV4155",
		},
		{
			name: "wrong operation return type",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nint IsValid(borrow const BufferRange value);\nrequires Validatable<BufferRange>;\n",
			code: "CV4156",
		},
		{
			name: "failed nested prerequisite",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nconcept ResourceState<T> { requires Validatable<T>; }\nrequires ResourceState<BufferRange>;\n",
			code: "CV4153",
		},
		{
			name: "direct concept cycle",
			src:  "profile Vulkan;\nconcept A<T> { requires A<T>; }\n",
			code: "CV4162",
		},
		{
			name: "indirect concept cycle",
			src:  "profile Vulkan;\nconcept A<T> { requires B<T>; }\nconcept B<T> { requires A<T>; }\n",
			code: "CV4162",
		},
		{
			name: "concept used as runtime type",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Validatable<T> { requires bool IsValid(borrow const T value); }\nValidatable<BufferRange> Make();\n",
			code: "CV4164",
		},
		{
			name: "bodyless template rejected",
			src:  "profile Vulkan;\ntemplate <typename T>\nint Identity(T value);\n",
			code: "CV4167",
		},
		{
			name: "m1a non exhaustive regression",
			src:  "profile Vulkan;\nenum Status { Empty, Ready(int value) }\nint Match(Status value) { return match (value) { Status::Empty => 0, }; }\n",
			code: "CV4115",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestEVT1TemplateDiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "constraint concrete type does not authorize generic operation",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\ntemplate <typename T>\nrequires Resource<BufferRange>\nint Score(borrow const T value) { return Measure(value); }\n",
			code: "CV4176",
		},
		{
			name: "dependent field access",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\nint Measure(borrow const BufferRange value);\ntemplate <typename T>\nrequires Resource<T>\nint Score(borrow const T value) { return value.offset; }\n",
			code: "CV4172",
		},
		{
			name: "explicit template call required",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\nint Measure(borrow const BufferRange value);\nrequires Resource<BufferRange>;\ntemplate <typename T>\nrequires Resource<T>\nint Score(borrow const T value) { return Measure(value); }\nint Use() { BufferRange range = BufferRange{1}; return Score(range); }\n",
			code: "CV4173",
		},
		{
			name: "dependent operator rejected",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\nint Measure(borrow const BufferRange value);\ntemplate <typename T>\nrequires Resource<T>\nbool Larger(borrow const T left, borrow const T right) { return left > right; }\n",
			code: "CV4175",
		},
		{
			name: "call not guaranteed by constraint",
			src:  "profile Vulkan;\nstruct BufferRange { int offset; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\nint Measure(borrow const BufferRange value);\ntemplate <typename T>\nrequires Resource<T>\nint Score(borrow const T value) { return Destroy(value); }\n",
			code: "CV4176",
		},
		{
			name: "by value immovable instantiation rejected",
			src:  "profile Vulkan;\nimmovable struct PipelineState { int handle; bool alive; };\nconcept Resource<T> { requires int Measure(borrow const T value); }\nint Measure(borrow const PipelineState value);\nrequires Resource<PipelineState>;\ntemplate <typename T>\nrequires Resource<T>\nint CopyResource(T value) { return Measure(value); }\nint Use() { PipelineState state = PipelineState{1, true}; return CopyResource<PipelineState>(state); }\n",
			code: "CV4136",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestEVT1TemplateInvocationPreservesComparisonParsing(t *testing.T) {
	src := `profile Vulkan;
struct BufferRange { int offset; };
concept Resource<T> { requires int Measure(borrow const T value); }
int Measure(borrow const BufferRange value);
requires Resource<BufferRange>;
template <typename T>
requires Resource<T>
int Score(borrow const T value) { return Measure(value); }
bool Less() { return 1 < 2; }
bool GreaterFromTemplate() { BufferRange range = BufferRange{1}; return Score<BufferRange>(range) > 0; }
bool Chained() { return 3 > 2 > 1; }
`
	_, err := Parse("test.concept", src)
	if err == nil || !strings.Contains(err.Error(), "CV4028") {
		t.Fatalf("err=%v want CV4028 type-check failure from chained comparison", err)
	}
}

func TestEVT1M1BCDiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "if requires else",
			src:  "profile Vulkan;\nint Use(bool flag) { return if (flag) 1; }\n",
			code: "CV4184",
		},
		{
			name: "else if rejected",
			src:  "profile Vulkan;\nint Use(bool a, bool b) { return if (a) 1 else if (b) 2 else 3; }\n",
			code: "CV4185",
		},
		{
			name: "runtime cannot call comptime function",
			src:  "profile Vulkan;\ncomptime int Bound(int value) { return value; }\nint Use() { return Bound(1); }\n",
			code: "CV4210",
		},
		{
			name: "runtime value rejected in comptime local",
			src:  "profile Vulkan;\nint Use(int value) { comptime int Local = value; return Local; }\n",
			code: "CV4200",
		},
		{
			name: "comptime while must be bounded",
			src:  "profile Vulkan;\ncomptime int Sum(int limit) { int cursor = 0; while (cursor < limit) { cursor = cursor + 1; } return cursor; }\nstatic_assert(Sum(1) == 1);\n",
			code: "CV4205",
		},
		{
			name: "bounded while requires comptime int",
			src:  "profile Vulkan;\nint Use(int limit) { int cursor = 0; while (cursor < limit) bounded(limit) { cursor = cursor + 1; } return cursor; }\n",
			code: "CV4200",
		},
		{
			name: "comptime recursion rejected",
			src:  "profile Vulkan;\ncomptime int First(int value) { return Second(value); }\ncomptime int Second(int value) { return First(value); }\nstatic_assert(First(1) == 1);\n",
			code: "CV4217",
		},
		{
			name: "static assert message must be string",
			src:  "profile Vulkan;\nstatic_assert(true, 1);\n",
			code: "CV4208",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestEVT1M1BDDiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "empty literal needs context",
			src:  "profile Vulkan;\ncomptime int Missing() { return Len([]); }\n",
			code: "CV4225",
		},
		{
			name: "heterogeneous literal rejected",
			src:  "profile Vulkan;\ncomptime int[2] Values = [1, true];\n",
			code: "CV4227",
		},
		{
			name: "non array index target",
			src:  "profile Vulkan;\ncomptime int Value = 3;\nstatic_assert(Value[0] == 0);\n",
			code: "CV4231",
		},
		{
			name: "index requires int",
			src:  "profile Vulkan;\ncomptime int[2] Values = [1, 2];\nstatic_assert(Values[true] == 1);\n",
			code: "CV4232",
		},
		{
			name: "index out of range",
			src:  "profile Vulkan;\ncomptime int[2] Values = [1, 2];\nstatic_assert(Values[2] == 0);\n",
			code: "CV4233",
		},
		{
			name: "len arity",
			src:  "profile Vulkan;\ncomptime int[2] Values = [1, 2];\nstatic_assert(Len(Values, Values) == 2);\n",
			code: "CV4234",
		},
		{
			name: "len requires array",
			src:  "profile Vulkan;\ncomptime int Value = 1;\nstatic_assert(Len(Value) == 1);\n",
			code: "CV4235",
		},
		{
			name: "array ordering rejected",
			src:  "profile Vulkan;\ncomptime int[2] Left = [1, 2];\ncomptime int[2] Right = [1, 3];\nstatic_assert(Left < Right);\n",
			code: "CV4236",
		},
		{
			name: "C-style for loop rejected",
			src:  "profile Vulkan;\nint Use() { for (int i = 0; i < 1; i = i + 1) { } return 0; }\n",
			code: "FOREACH_ITERATOR_INVALID",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestEVT1M1BCGenerationErasesComptimeRuntime(t *testing.T) {
	src := readEVT1Fixture(t, "comptime_values.concept")
	module, err := Parse("examples/evt1/comptime_values.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["comptime_values.generated.h"])
	body := string(outputs["comptime_values.generated.c"])
	if strings.Contains(header, "ClampCount") || strings.Contains(body, "ClampCount(") || strings.Contains(body, "SumTo(") {
		t.Fatalf("comptime functions leaked into generated runtime output:\n%s", body)
	}
	for _, needle := range []string{"while (", "if (", "return 3;"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("generated C missing %q\n%s", needle, body)
		}
	}
}

func TestEVT1M1BDGenerationErasesComptimeRuntime(t *testing.T) {
	src := readEVT1Fixture(t, "comptime_tables.concept")
	module, err := Parse("examples/evt1/comptime_tables.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["comptime_tables.generated.h"])
	body := string(outputs["comptime_tables.generated.c"])
	for _, forbidden := range []string{"CanonicalRetryBudgets(", "SumBudgets(", "HasDuplicateTransitionKeys(", "Len("} {
		if strings.Contains(header, forbidden) || strings.Contains(body, forbidden) {
			t.Fatalf("comptime array helper leaked into generated runtime output: %s", forbidden)
		}
	}
	for _, needle := range []string{"return 2;", "return 6;", "return 7;"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("generated C missing %q\n%s", needle, body)
		}
	}
}

func TestDragonGodM0IdentityIgnoresWhitespaceAndLocation(t *testing.T) {
	left := `profile Vulkan;
enum Signal
{
    Go,
    Stop
}
automata Demo(Signal)
{
    initial machine Main
    {
        initial state Start
        {
            on Signal::Go goto Done;
        }
        terminal state Done
        {
            finish;
        }
    }
}
int Value() { return 1; }
`
	right := `profile Vulkan;

enum Signal { Go, Stop }

automata Demo(Signal)
{
    initial machine Main
    {

        initial state Start
        {
            on Signal::Go goto Done;
        }

        terminal state Done
        {
            finish;
        }
    }
}

int Value()
{
    return 1;
}
`
	leftModule, err := Parse("left.concept", left)
	if err != nil {
		t.Fatal(err)
	}
	rightModule, err := Parse("right.concept", right)
	if err != nil {
		t.Fatal(err)
	}
	leftEnv, err := analyzeModule(leftModule)
	if err != nil {
		t.Fatal(err)
	}
	rightEnv, err := analyzeModule(rightModule)
	if err != nil {
		t.Fatal(err)
	}
	leftID := buildMIR(leftModule, leftEnv).Automata[0].GraphIdentity
	rightID := buildMIR(rightModule, rightEnv).Automata[0].GraphIdentity
	if leftID != rightID {
		t.Fatalf("graph identity changed across whitespace/location-only edits: %s != %s", leftID, rightID)
	}
}

func TestDragonGodM3EffectIdentityTracksEmitOrderWithoutChangingTopology(t *testing.T) {
	left := `profile Vulkan;
enum Signal { Go }
effect First(int value);
effect Second(int value);
automata Demo(Signal) {
  initial machine Main {
    initial state Start {
      on Signal::Go => {
        emit First(1);
        emit Second(2);
        goto Done;
      }
    }
    terminal state Done { finish; }
  }
}
int Value() { return 1; }
`
	right := `profile Vulkan;
enum Signal { Go }
effect First(int value);
effect Second(int value);
automata Demo(Signal) {
  initial machine Main {
    initial state Start {
      on Signal::Go => {
        emit Second(2);
        emit First(1);
        goto Done;
      }
    }
    terminal state Done { finish; }
  }
}
int Value() { return 1; }
`
	leftModule, err := Parse("left.concept", left)
	if err != nil {
		t.Fatal(err)
	}
	rightModule, err := Parse("right.concept", right)
	if err != nil {
		t.Fatal(err)
	}
	leftEnv, err := analyzeModule(leftModule)
	if err != nil {
		t.Fatal(err)
	}
	rightEnv, err := analyzeModule(rightModule)
	if err != nil {
		t.Fatal(err)
	}
	leftAutomata := buildMIR(leftModule, leftEnv).Automata[0]
	rightAutomata := buildMIR(rightModule, rightEnv).Automata[0]
	if leftAutomata.TopologyIdentity != rightAutomata.TopologyIdentity {
		t.Fatalf("topology identity changed across emit reorder: %s != %s", leftAutomata.TopologyIdentity, rightAutomata.TopologyIdentity)
	}
	if leftAutomata.EffectIdentity == rightAutomata.EffectIdentity {
		t.Fatalf("effect identity ignored emit reorder: %s", leftAutomata.EffectIdentity)
	}
}

func TestDragonGodM0DiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "non enum signal type",
			src:  "profile Vulkan;\nstruct SignalSet { int value; };\nautomata Demo(SignalSet) { initial machine Main { initial state Start { on SignalSet::Value goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4241",
		},
		{
			name: "missing initial machine",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4243",
		},
		{
			name: "terminal modifier order",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { terminal initial state Start { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4248",
		},
		{
			name: "duplicate transition key",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; on Signal::Go goto Start; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4253",
		},
		{
			name: "cross machine goto",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Other::Done; } terminal state Done { finish; } } machine Other { initial terminal state Done { pop; } } }\nint Value() { return 1; }\n",
			code: "CV4254",
		},
		{
			name: "push continuation resolved in pushed machine",
			src:  "profile Vulkan;\nenum Signal { Go, Return }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go push Other goto Other::Done; } terminal state Done { finish; } } machine Other { initial state Begin { on Signal::Return goto Done; } terminal state Done { pop; } } }\nint Value() { return 1; }\n",
			code: "CV4256",
		},
		{
			name: "push cycle",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go push Other goto Done; } terminal state Done { finish; } } machine Other { initial state Begin { on Signal::Go push Main goto Begin; } terminal state Done { pop; } } }\nint Value() { return 1; }\n",
			code: "CV4257",
		},
		{
			name: "unreachable machine",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } machine Other { initial terminal state Begin { pop; } } }\nint Value() { return 1; }\n",
			code: "CV4258",
		},
		{
			name: "unreachable state",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } state Dead { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4259",
		},
		{
			name: "missing reachable pop",
			src:  "profile Vulkan;\nenum Signal { Go, Stop }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go push Other goto Done; } terminal state Done { finish; } } machine Other { initial terminal state Begin { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4260",
		},
		{
			name: "missing reachable finish",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Loop; } state Loop { on Signal::Go goto Loop; } } }\nint Value() { return 1; }\n",
			code: "CV4261",
		},
		{
			name: "root pop invalid",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial terminal state Start { pop; } } }\nint Value() { return 1; }\n",
			code: "CV4262",
		},
		{
			name: "automata first class expression",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return Demo; }\n",
			code: "CV4263",
		},
		{
			name: "ordinary statement inside state body",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { int value = 1; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4264",
		},
		{
			name: "machine count limit",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) {\ninitial machine M0 { initial state S { on Signal::Go goto D; } terminal state D { finish; } }\nmachine M1 { initial terminal state S { pop; } }\nmachine M2 { initial terminal state S { pop; } }\nmachine M3 { initial terminal state S { pop; } }\nmachine M4 { initial terminal state S { pop; } }\nmachine M5 { initial terminal state S { pop; } }\nmachine M6 { initial terminal state S { pop; } }\nmachine M7 { initial terminal state S { pop; } }\nmachine M8 { initial terminal state S { pop; } }\nmachine M9 { initial terminal state S { pop; } }\nmachine M10 { initial terminal state S { pop; } }\nmachine M11 { initial terminal state S { pop; } }\nmachine M12 { initial terminal state S { pop; } }\nmachine M13 { initial terminal state S { pop; } }\nmachine M14 { initial terminal state S { pop; } }\nmachine M15 { initial terminal state S { pop; } }\nmachine M16 { initial terminal state S { pop; } }\n}\nint Value() { return 1; }\n",
			code: "CV4265",
		},
		{
			name: "compiler owned outcome redeclaration",
			src:  "profile Vulkan;\nenum AutomataDispatchOutcome { Value }\nint Value() { return 1; }\n",
			code: "CV4267",
		},
		{
			name: "signal payload variant rejected",
			src:  "profile Vulkan;\nenum Signal { Go(int payload) }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4269",
		},
		{
			name: "instance requires automata",
			src:  "profile Vulkan;\nenum Signal { Go }\nint Value() { instance Signal value; return 1; }\n",
			code: "CV4270",
		},
		{
			name: "ordinary use of instance rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { instance Demo value; return value; }\n",
			code: "CV4272",
		},
		{
			name: "dispatch first operand must be instance",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { int value = 1; dispatch(value, Signal::Go); return 1; }\n",
			code: "CV4273",
		},
		{
			name: "dispatch wrong signal enum",
			src:  "profile Vulkan;\nenum Left { Go }\nenum Right { Go }\nautomata Demo(Left) { initial machine Main { initial state Start { on Left::Go goto Done; } terminal state Done { finish; } } }\nint Value() { instance Demo value; dispatch(value, Right::Go); return 1; }\n",
			code: "CV4274",
		},
		{
			name: "dispatch name redeclaration rejected",
			src:  "profile Vulkan;\nint dispatch() { return 1; }\nint Value() { return 2; }\n",
			code: "CV4268",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestDragonGodM2DiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "context missing borrow",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, Context context) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4276",
		},
		{
			name: "duplicate context params",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow left: Context, borrow right: Context) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4278",
		},
		{
			name: "pointer context rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal, borrow context: int*) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4281",
		},
		{
			name: "instance missing required context",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { instance Demo value; return 1; }\n",
			code: "CV4283",
		},
		{
			name: "context argument supplied to contextless automata",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { Context context = Context{true}; instance Demo value(context); return 1; }\n",
			code: "CV4284",
		},
		{
			name: "guard must be bool",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go when 1 => goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4290",
		},
		{
			name: "ordinary plus guarded mix rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go goto Done; on Signal::Go when context.ready => goto Start; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4287",
		},
		{
			name: "otherwise must be last",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go otherwise => goto Done; on Signal::Go when context.ready => goto Start; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4288",
		},
		{
			name: "fallback only group rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go otherwise => goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4289",
		},
		{
			name: "retained context blocks mutation",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go when context.ready => goto Done; on Signal::Go otherwise => goto Start; } terminal state Done { finish; } } }\nint Value() { Context context = Context{true}; instance Demo value(context); context.ready = false; return 1; }\n",
			code: "CV4291",
		},
		{
			name: "recursive guard call rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nstruct Context { bool ready; };\nbool Loop(borrow const Context context) { return Loop(context); }\nautomata Demo(Signal, borrow context: Context) { initial machine Main { initial state Start { on Signal::Go when Loop(context) => goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4296",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestDragonGodM3DiagnosticsAreStable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		code string
	}{
		{
			name: "string payload rejected",
			src:  "profile Vulkan;\neffect Bad(string text);\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4303",
		},
		{
			name: "unknown emitted effect rejected",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go => { emit Missing(); goto Done; } } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4301",
		},
		{
			name: "effectful dispatch requires batch",
			src:  "profile Vulkan;\neffect Mark(int value);\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go => { emit Mark(1); goto Done; } } terminal state Done { finish; } } }\nint Value() { instance Demo lifecycle; return 1 + dispatch(lifecycle, Signal::Go).tag; }\n",
			code: "CV4306",
		},
		{
			name: "batch ordinary use rejected",
			src:  "profile Vulkan;\neffect Mark(int value);\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go => { emit Mark(1); goto Done; } } terminal state Done { finish; } } }\nint Value() { instance Demo lifecycle; effects Demo emitted; dispatch(lifecycle, Signal::Go, emitted); return emitted; }\n",
			code: "CV4305",
		},
		{
			name: "effect free dispatch rejects third operand",
			src:  "profile Vulkan;\nenum Signal { Go }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go goto Done; } terminal state Done { finish; } } }\nint Value() { instance Demo lifecycle; effects Demo emitted; return 1 + dispatch(lifecycle, Signal::Go, emitted).tag; }\n",
			code: "CV4309",
		},
		{
			name: "payload call rejected",
			src:  "profile Vulkan;\neffect Mark(int value);\nenum Signal { Go }\nint ValueOf() { return 1; }\nautomata Demo(Signal) { initial machine Main { initial state Start { on Signal::Go => { emit Mark(ValueOf()); goto Done; } } terminal state Done { finish; } } }\nint Value() { return 1; }\n",
			code: "CV4312",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("test.concept", tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v want %s", err, tc.code)
			}
		})
	}
}

func TestEVT1CheckRejectsHandEdit(t *testing.T) {
	src := readEVT1Fixture(t, "resource_concepts.concept")
	module, err := Parse("examples/evt1/resource_concepts.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resource_concepts.generated.c"), []byte("edited"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Check(dir, outputs); err == nil || !strings.Contains(err.Error(), "CV3001") {
		t.Fatalf("check error=%v", err)
	}
}

func TestEVT1TemplateInstancesAreDeterministicAndDeduplicated(t *testing.T) {
	src := readEVT1Fixture(t, "concept_templates.concept")
	module, err := Parse("examples/evt1/concept_templates.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["concept_templates.generated.c"])
	for _, needle := range []string{
		"static int32_t concept_template_score_resource__buffer_range(",
		"static int32_t concept_template_score_resource__pipeline_state(",
		"static void concept_template_destroy_resource__buffer_range(",
		"static void concept_template_destroy_resource__pipeline_state(",
	} {
		if strings.Count(body, needle) != 2 { // one forward declaration and one definition
			t.Fatalf("expected one instance for %q\n%s", needle, body)
		}
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	mir := buildMIR(module, env)
	if len(mir.Templates) != 2 {
		t.Fatalf("template count=%d", len(mir.Templates))
	}
	if len(mir.Instances) != 4 {
		t.Fatalf("instance count=%d", len(mir.Instances))
	}
}

func TestEVT1LanguageSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "resource_concepts.concept")
	module, err := Parse("examples/evt1/resource_concepts.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "resource_concepts.generated.h"
#include <stdint.h>

static int g_next_value = 0;

bool concept_resource_concepts_is_valid(const concept_command_pool_state* value) {
  return value->initialized;
}
void concept_resource_concepts_destroy(concept_command_pool_state* value) {
  (void)value;
}
int concept_resource_concepts_next_value(void) {
  g_next_value += 1;
  return g_next_value;
}
int concept_resource_concepts_add(int left, int right) {
  return left + right;
}

int main(void) {
  concept_observed_values observed = concept_resource_concepts_observe_construction();
  concept_copy_result copy = concept_resource_concepts_copy_and_mutate();
  if (observed.first != 1 || observed.second != 2 || observed.third != 3) return 1;
  if (g_next_value != 3) return 2;
  if (copy.firstOffset != 1 || copy.secondOffset != 9) return 3;
  if (concept_resource_concepts_match_allocation() != 18) return 4;
  if (!concept_resource_concepts_use_immovable()) return 5;
  return 0;
}
`
	runNativeHarness(t, outputs, "resource_concepts_harness.c", harness, nil)
}

func TestEVT1VulkanSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "resource_concepts_vulkan.concept")
	module, err := Parse("examples/evt1/resource_concepts_vulkan.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "resource_concepts_vulkan.generated.h"
#include <stdint.h>

static int g_destroy_calls = 0;

bool concept_resource_concepts_vulkan_is_valid(const concept_command_pool_state* value) {
  return value->initialized;
}
void concept_resource_concepts_vulkan_destroy(concept_command_pool_state* value) {
  (void)value;
  g_destroy_calls += 1;
}

int main(void) {
  VkBuffer buffer = (VkBuffer)(uintptr_t)0x10u;
  VkCommandPool pool = (VkCommandPool)(uintptr_t)0x20u;
  if (concept_resource_concepts_vulkan_classify_range(buffer) != 5) return 1;
  if (!concept_resource_concepts_vulkan_build_and_validate(pool)) return 2;
  concept_resource_concepts_vulkan_cleanup_state(pool);
  if (g_destroy_calls != 1) return 3;
  return 0;
}
`
	runNativeHarness(t, outputs, "resource_concepts_vulkan_harness.c", harness, nil)
}

func TestEVT1TemplateLanguageSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "concept_templates.concept")
	module, err := Parse("examples/evt1/concept_templates.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "concept_templates.generated.h"
#include <stdint.h>

static int g_destroy_count = 0;
static int g_destroy_codes[4] = {0, 0, 0, 0};

int concept_concept_templates_measure__buffer_range(const concept_buffer_range* value) {
  return value->offset + value->size;
}
int concept_concept_templates_measure__pipeline_state(const concept_pipeline_state* value) {
  return value->handle + (value->alive ? 1 : 0);
}
void concept_concept_templates_destroy__buffer_range(concept_buffer_range* value) {
  g_destroy_codes[g_destroy_count++] = value->offset;
}
void concept_concept_templates_destroy__pipeline_state(concept_pipeline_state* value) {
  g_destroy_codes[g_destroy_count++] = value->handle;
}
void concept_concept_templates_set_alive(concept_pipeline_state* value) {
  value->alive = true;
}

int main(void) {
  concept_destroy_audit audit;
  if (concept_concept_templates_repeated_score() != 14) return 1;
  if (concept_concept_templates_score_pipeline() != 12) return 2;
  audit = concept_concept_templates_use_destroyers();
  if (audit.first != 1 || audit.second != 13 || !audit.third) return 3;
  if (!concept_concept_templates_use_immovable()) return 4;
  if (!concept_concept_templates_compare_template_score()) return 5;
  if (g_destroy_count != 4) return 6;
  if (g_destroy_codes[0] != 1 || g_destroy_codes[1] != 13 || g_destroy_codes[2] != 13 || g_destroy_codes[3] != 5) return 7;
  return 0;
}
`
	runNativeHarness(t, outputs, "concept_templates_harness.c", harness, nil)
}

func TestEVT1TemplateVulkanSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "concept_templates_vulkan.concept")
	module, err := Parse("examples/evt1/concept_templates_vulkan.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "concept_templates_vulkan.generated.h"
#include <stdint.h>

static int g_destroy_calls = 0;

int concept_concept_templates_vulkan_measure__buffer_range(const concept_buffer_range* value) {
  return value->offset + value->size;
}
int concept_concept_templates_vulkan_measure__pipeline_state(const concept_pipeline_state* value) {
  return value->alive ? 100 : 0;
}
void concept_concept_templates_vulkan_destroy__buffer_range(concept_buffer_range* value) {
  (void)value;
}
void concept_concept_templates_vulkan_destroy__pipeline_state(concept_pipeline_state* value) {
  (void)value;
  g_destroy_calls += 1;
}
void concept_concept_templates_vulkan_set_alive(concept_pipeline_state* value) {
  value->alive = true;
}

int main(void) {
  VkBuffer first = (VkBuffer)(uintptr_t)0x10u;
  VkBuffer second = (VkBuffer)(uintptr_t)0x20u;
  VkCommandPool pool = (VkCommandPool)(uintptr_t)0x30u;
  if (concept_concept_templates_vulkan_classify_range(first) != 5) return 1;
  if (concept_concept_templates_vulkan_double_range_score(first, second) != 22) return 2;
  if (!concept_concept_templates_vulkan_build_and_destroy(pool)) return 3;
  if (g_destroy_calls != 1) return 4;
  return 0;
}
`
	runNativeHarness(t, outputs, "concept_templates_vulkan_harness.c", harness, nil)
}

func TestEVT1M1BCLanguageSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "comptime_values.concept")
	module, err := Parse("examples/evt1/comptime_values.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "comptime_values.generated.h"

int main(void) {
  if (concept_comptime_values_selected_arm(true, 7, 3) != 7) return 1;
  if (concept_comptime_values_selected_arm(false, 7, 3) != 3) return 2;
  if (concept_comptime_values_count_up(2) != 4) return 3;
  if (concept_comptime_values_count_up(10) != 6) return 4;
  if (concept_comptime_values_zero_bound(5) != 99) return 5;
  if (concept_comptime_values_default_bound() != 3) return 6;
  return 0;
}
`
	runNativeHarness(t, outputs, "comptime_values_harness.c", harness, nil)
}

func TestEVT1M1BCVulkanSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "comptime_values_vulkan.concept")
	module, err := Parse("examples/evt1/comptime_values_vulkan.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "comptime_values_vulkan.generated.h"
#include <stdint.h>

int main(void) {
  VkBuffer buffer = (VkBuffer)(uintptr_t)0x10u;
  VkCommandPool pool = (VkCommandPool)(uintptr_t)0x20u;
  if (concept_comptime_values_vulkan_classify_range(buffer) != 3) return 1;
  if (!concept_comptime_values_vulkan_pool_ready(pool, true)) return 2;
  if (concept_comptime_values_vulkan_pool_ready(pool, false)) return 3;
  return 0;
}
`
	runNativeHarness(t, outputs, "comptime_values_vulkan_harness.c", harness, nil)
}

func TestEVT1M1BDLanguageSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "comptime_tables.concept")
	module, err := Parse("examples/evt1/comptime_tables.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "comptime_tables.generated.h"

int main(void) {
  if (concept_comptime_tables_default_retry_budget() != 2) return 1;
  if (concept_comptime_tables_matrix_corner() != 6) return 2;
  if (concept_comptime_tables_summary_value() != 7) return 3;
  if (!concept_comptime_tables_transition_table_stable()) return 4;
  if (concept_comptime_tables_total_retry_budget() != 7) return 5;
  return 0;
}
`
	runNativeHarness(t, outputs, "comptime_tables_harness.c", harness, nil)
}

func TestEVT1M1BDVulkanSpecimenNativeC11(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native C11 harness is only configured for Windows in this repository")
	}
	src := readEVT1Fixture(t, "comptime_tables_vulkan.concept")
	module, err := Parse("examples/evt1/comptime_tables_vulkan.concept", src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "comptime_tables_vulkan.generated.h"
#include <stdint.h>

int main(void) {
  VkBuffer buffer = (VkBuffer)(uintptr_t)0x10u;
  VkCommandPool pool = (VkCommandPool)(uintptr_t)0x20u;
  if (concept_comptime_tables_vulkan_classify_range(buffer) != 5) return 1;
  if (concept_comptime_tables_vulkan_pipeline_stride(buffer) != 4) return 2;
  if (!concept_comptime_tables_vulkan_pool_ready(pool, true)) return 3;
  if (concept_comptime_tables_vulkan_pool_ready(pool, false)) return 4;
  return 0;
}
`
	runNativeHarness(t, outputs, "comptime_tables_vulkan_harness.c", harness, nil)
}

func runNativeHarness(t *testing.T, outputs Outputs, harnessName, harnessSource string, extraArgs []string) {
	t.Helper()
	if _, err := exec.LookPath("cl"); err != nil {
		t.Skip("cl not found on PATH")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harnessPath := filepath.Join(dir, harnessName)
	if err := os.WriteFile(harnessPath, []byte(harnessSource), 0644); err != nil {
		t.Fatal(err)
	}
	var generatedC string
	for key := range outputs {
		if strings.HasSuffix(key, ".generated.c") {
			generatedC = filepath.Join(dir, key)
			break
		}
	}
	if generatedC == "" {
		t.Fatal("generated C output missing")
	}
	args := []string{"/nologo", "/std:c11", "/W4", "/I" + dir}
	if sdk := os.Getenv("VULKAN_SDK"); sdk != "" {
		args = append(args, "/I"+filepath.Join(sdk, "Include"))
	}
	args = append(args, generatedC, harnessPath, "/Fe:"+filepath.Join(dir, "specimen.exe"))
	args = append(args, extraArgs...)
	build := nativeCommand(t, "cl", args...)
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err != nil {
		text := strings.ToLower(string(out))
		if strings.Contains(text, "cannot open include file: 'vulkan/vulkan.h'") {
			t.Skip("Vulkan SDK headers are unavailable in this environment")
		}
		if strings.Contains(text, "cannot open include file: 'stdbool.h'") ||
			strings.Contains(text, "cannot open include file: 'stdint.h'") ||
			strings.Contains(text, "cannot open include file: 'stddef.h'") {
			t.Skip("MSVC developer include environment is unavailable in this shell")
		}
		t.Fatalf("cl failed:\n%s", out)
	}
	run := nativeCommand(t, filepath.Join(dir, "specimen.exe"))
	run.Dir = dir
	runOut, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("specimen failed: %v\n%s", err, runOut)
	}
}

func runNativeHarnessIncludingGeneratedC(t *testing.T, outputs Outputs, harnessName, harnessSource string, extraArgs []string) {
	t.Helper()
	if _, err := exec.LookPath("cl"); err != nil {
		t.Skip("cl not found on PATH")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harnessPath := filepath.Join(dir, harnessName)
	if err := os.WriteFile(harnessPath, []byte(harnessSource), 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"/nologo", "/std:c11", "/W4", "/I" + dir}
	if sdk := os.Getenv("VULKAN_SDK"); sdk != "" {
		args = append(args, "/I"+filepath.Join(sdk, "Include"))
	}
	args = append(args, harnessPath, "/Fe:"+filepath.Join(dir, "specimen.exe"))
	args = append(args, extraArgs...)
	build := nativeCommand(t, "cl", args...)
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err != nil {
		text := strings.ToLower(string(out))
		if strings.Contains(text, "cannot open include file: 'vulkan/vulkan.h'") {
			t.Skip("Vulkan SDK headers are unavailable in this environment")
		}
		if strings.Contains(text, "cannot open include file: 'stdbool.h'") ||
			strings.Contains(text, "cannot open include file: 'stdint.h'") ||
			strings.Contains(text, "cannot open include file: 'stddef.h'") {
			t.Skip("MSVC developer include environment is unavailable in this shell")
		}
		t.Fatalf("cl failed:\n%s", out)
	}
	run := nativeCommand(t, filepath.Join(dir, "specimen.exe"))
	run.Dir = dir
	runOut, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("specimen failed: %v\n%s", err, runOut)
	}
}
