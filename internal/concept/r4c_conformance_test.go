package concept

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type r4cCase struct {
	file     string
	accepted bool
	category string
	evidence string
}

var r4cCases = []r4cCase{
	{"option_some_none.concept", true, "", "PoC3 Phase 22 pressure; EVT1 canonical constructors"},
	{"option_match.concept", true, "", "PoC3 Option match evidence"},
	{"option_question_propagate.concept", true, "", "EVT1-new Option propagation"},
	{"option_bang.concept", true, "", "EVT1-new explicit escalation"},
	{"result_ok_error.concept", true, "", "EVT1-new generic Result"},
	{"result_match.concept", true, "", "payload-enum ground truth"},
	{"result_question_propagate.concept", true, "", "PoC3 Phase 5 early-return pressure; EVT1 exact-E rule"},
	{"result_bang.concept", true, "", "PoC3 Phase 17 panic pressure; EVT1-new Result sugar"},
	{"try_except_single_error.concept", true, "", "EVT1-new local structured handling"},
	{"try_except_multiple_exact_errors.concept", true, "", "EVT1-new static heterogeneous routing"},
	{"assert_runtime.concept", true, "", "PoC3 Phase 17 assertion evidence"},
	{"static_assert_comptime.concept", true, "", "existing EVT1 comptime evaluator"},
	{"result_owned_payload_move.concept", true, "", "EVT1-new ownership integration"},
	{"result_lifetime_provenance.concept", true, "", "R4b provenance preserved through Result and ?"},
	{"question_nonfallible.concept", false, "QUESTION_REQUIRES_OPTION_OR_RESULT", "EVT1-new diagnostic"},
	{"question_result_wrong_error.concept", false, "RESULT_ERROR_TYPE_MISMATCH", "EVT1-new exact-E diagnostic"},
	{"question_option_in_result.concept", false, "QUESTION_INCOMPATIBLE_RETURN_CHANNEL", "EVT1-new channel diagnostic"},
	{"bang_nonfallible.concept", false, "BANG_REQUIRES_OPTION_OR_RESULT", "EVT1-new diagnostic"},
	{"try_except_unhandled_error.concept", false, "TRY_EXCEPT_UNHANDLED_ERROR", "EVT1-new diagnostic"},
	{"try_except_duplicate_error.concept", false, "TRY_EXCEPT_DUPLICATE_ERROR", "EVT1-new diagnostic"},
	{"assert_nonbool.concept", false, "ASSERT_REQUIRES_BOOL", "PoC3 Phase 17 pressure"},
	{"static_assert_runtime_value.concept", false, "CV4210", "existing EVT1 comptime diagnostic"},
	{"result_immovable_payload.concept", false, "FAILURE_PAYLOAD_IMMOVABLE", "R2 immovable embedding law"},
	{"result_lifetime_launder.concept", false, "REF_STRUCT_ESCAPE", "R4b lifetime laundering rejection"},
}

func TestEVT1R4cConformance(t *testing.T) {
	valid, invalid := 0, 0
	for _, tc := range r4cCases {
		t.Run(tc.file, func(t *testing.T) {
			dir := "valid"
			if !tc.accepted {
				dir = "invalid"
			}
			path := filepath.Join("..", "..", "language", "evt1-r4c", "core", dir, tc.file)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if tc.accepted {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Generate(module, source); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid R4c case was accepted")
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != tc.category {
				t.Fatalf("category = %s, want %s: %v", diagnostic.SemanticCategory(), tc.category, err)
			}
		})
		if tc.accepted {
			valid++
		} else {
			invalid++
		}
	}
	if valid != 14 || invalid != 10 {
		t.Fatalf("R4c count drift: valid=%d invalid=%d", valid, invalid)
	}
}

func TestEVT1R4cMIREvidence(t *testing.T) {
	outputs := generateR4cFixture(t, "result_question_propagate.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["result_question_propagate.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "result_propagate" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("result_propagate proof operation missing")
	}
	tryOutputs := generateR4cFixture(t, "try_except_multiple_exact_errors.concept")
	if !strings.Contains(string(tryOutputs["try_except_multiple_exact_errors.mir.json"]), `"kind": "try_handler"`) {
		t.Fatal("try_handler MIR evidence missing")
	}
}

func TestEVT1R4cStaticAssertUsesComptimePath(t *testing.T) {
	_, err := Parse("static_assert_fail.concept", "profile Core; static_assert(false, \"R4c invariant\");")
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4207" || !strings.Contains(diagnostic.Message, "R4c invariant") {
		t.Fatalf("static assertion did not use comptime assertion path: %v", err)
	}
}

func TestEVT1R4cOwnedMatchRequiresMove(t *testing.T) {
	source := "profile Core; struct Resource { int id; } enum E { Bad, } int Read(Result<owned Resource, E> value) { return match (value) { Result::Ok(resource) => resource.id, Result::Error(error) => 0, }; }"
	_, err := Parse("owned_match_without_move.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "RESULT_PAYLOAD_NONTRANSFERABLE" {
		t.Fatalf("owned payload match copied silently: %v", err)
	}
}

func TestEVT1R4cOwnedPropagationRequiresMove(t *testing.T) {
	source := "profile Core; struct Resource { int id; } enum E { Bad, } Result<owned Resource, E> Acquire() { owned Resource resource = Resource { 1 }; return Result::Ok(move resource); } Result<owned Resource, E> Forward() { Result<owned Resource, E> result = Acquire(); owned Resource resource = result?; return Result::Ok(move resource); }"
	_, err := Parse("owned_question_without_move.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "RESULT_PAYLOAD_NONTRANSFERABLE" {
		t.Fatalf("owned propagation copied its carrier silently: %v", err)
	}
}

func TestEVT1R4cCarrierCleanupEvidence(t *testing.T) {
	outputs := generateR4cFixture(t, "result_owned_payload_move.concept")
	c := string(outputs["result_owned_payload_move.generated.c"])
	if !strings.Contains(c, "concept_option_owned_resource_drop") ||
		!strings.Contains(c, "concept_result_owned_resource_resource_error_drop") ||
		!strings.Contains(c, "concept_option_owned_resource_drop(optional);") {
		t.Fatal("active-payload carrier cleanup was not generated")
	}
	if !strings.Contains(string(outputs["result_owned_payload_move.mir.json"]), `"drop_function": "concept_option_owned_resource_drop"`) {
		t.Fatal("carrier cleanup is absent from MIR evidence")
	}
}

func TestEVT1R4cResultVoidUsesCanonicalCarrier(t *testing.T) {
	source := []byte(`profile Core;
enum UnitError { Failed, }
Result<void, UnitError> Pass() { return Result::Ok(); }
Result<void, UnitError> Forward() { Pass()?; return Result::Ok(); }
`)
	module, err := Parse("result_void.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	runR1NativeHarness(t, outputs, "result_void_harness.c", "#include \"result_void.generated.h\"\nint main(void) { return concept_result_void_forward().tag == 0 ? 0 : 1; }\n")
}

func TestEVT1R4cTryBoundaryRunsLexicalCleanup(t *testing.T) {
	source := []byte(`profile Core;
struct Resource { int id; }
void Drop(owned Resource resource) { assert(resource.id > 0, "try cleanup resource"); }
enum OpenError { Closed, }
Result<int, OpenError> Open(bool ready) { return if (ready) Result::Ok(4) else Result::Error(OpenError::Closed); }
int Run(bool ready) {
    int answer = 0;
    try {
        owned Resource resource = Resource { 1 };
        answer = Open(ready)?;
    }
    except (OpenError error) { answer = 9; }
    return answer;
}
`)
	module, err := Parse("try_cleanup.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["try_cleanup.generated.c"])
	drop := strings.Index(c, "concept_try_cleanup_drop(resource);")
	if drop < 0 || !strings.Contains(c[drop:], "goto cv_except") {
		t.Fatal("try failure edge does not contain lexical cleanup before its local handler jump")
	}
	runR1NativeHarness(t, outputs, "try_cleanup_harness.c", "#include \"try_cleanup.generated.h\"\nint main(void) { return concept_try_cleanup_run(false) == 9 ? 0 : 1; }\n")
}

func TestEVT1R4cNativeC11(t *testing.T) {
	cases := []struct{ fixture, harness string }{
		{"option_some_none.concept", `int main(void) { concept_option_option_int value = concept_option_some_none_nested_value(); return value.tag == 0 && value.payload.some.value.tag == 0 && value.payload.some.value.payload.some.value == 8 ? 0 : 1; }`},
		{"option_question_propagate.concept", `int main(void) { concept_option_bool yes = concept_option_question_propagate_has_value(true); concept_option_bool no = concept_option_question_propagate_has_value(false); return yes.tag == 0 && yes.payload.some.value && no.tag == 1 ? 0 : 1; }`},
		{"result_question_propagate.concept", `int main(void) { concept_result_bool_parse_error yes = concept_result_question_propagate_is_positive(true); concept_result_bool_parse_error no = concept_result_question_propagate_is_positive(false); return yes.tag == 0 && yes.payload.ok.value && no.tag == 1 ? 0 : 1; }`},
		{"result_bang.concept", `int main(void) { return concept_result_bang_require(true) == 14 ? 0 : 1; }`},
		{"try_except_multiple_exact_errors.concept", `int main(void) { return concept_try_except_multiple_exact_errors_run(true, true) == 5 && concept_try_except_multiple_exact_errors_run(false, true) == 10 && concept_try_except_multiple_exact_errors_run(true, false) == 20 ? 0 : 1; }`},
		{"assert_runtime.concept", `int main(void) { return concept_assert_runtime_require_positive(2) == 2 ? 0 : 1; }`},
		{"result_owned_payload_move.concept", `int main(void) { concept_result_owned_resource_resource_error value = concept_result_owned_payload_move_forward(); concept_result_owned_resource_resource_error existing = concept_result_owned_payload_move_forward_existing(); concept_option_owned_resource optional = concept_result_owned_payload_move_optional(); return concept_result_owned_payload_move_read_owned(value) == 21 && concept_result_owned_payload_move_read_owned(existing) == 21 && optional.tag == 0 && optional.payload.some.value.id == 22 && concept_result_owned_payload_move_drop_optional_carrier() == 1 && concept_result_owned_payload_move_drop_inactive_carriers() == 2 ? 0 : 1; }`},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			outputs := generateR4cFixture(t, tc.fixture)
			base := strings.TrimSuffix(tc.fixture, ".concept")
			runR1NativeHarness(t, outputs, base+"_harness.c", "#include \""+base+".generated.h\"\n"+tc.harness+"\n")
		})
	}
}

func TestEVT1R4cTerminalPanicPaths(t *testing.T) {
	cases := []struct{ fixture, call, reason string }{
		{"result_bang.concept", "concept_result_bang_require(false)", "explicit Result error escalation"},
		{"assert_runtime.concept", "concept_assert_runtime_require_positive(0)", "value must be positive"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			compiler, err := exec.LookPath("gcc")
			if err != nil {
				compiler, err = exec.LookPath("clang")
			}
			if err != nil {
				t.Skip("C compiler unavailable")
			}
			outputs := generateR4cFixture(t, tc.fixture)
			base := strings.TrimSuffix(tc.fixture, ".concept")
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			harness := filepath.Join(dir, "panic_harness.c")
			if err := os.WriteFile(harness, []byte("#include \""+base+".generated.h\"\nint main(void) { (void)"+tc.call+"; return 0; }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "panic.exe")
			generated := filepath.Join(dir, base+".generated.c")
			if out, err := exec.Command(compiler, "-std=c11", "-I", dir, generated, harness, "-o", exe).CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			out, err := exec.Command(exe).CombinedOutput()
			if err == nil {
				t.Fatal("terminal path returned normally")
			}
			if !strings.Contains(string(out), tc.reason) || !strings.Contains(string(out), "Concept panic at") {
				t.Fatalf("panic evidence missing: %s", out)
			}
		})
	}
}

func generateR4cFixture(t *testing.T, fixture string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r4c", "core", "valid", fixture)
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
