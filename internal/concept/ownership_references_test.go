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

type ownershipReferencesConformanceCase struct {
	name             string
	source           string
	expected         ConformanceExpectation
	reference        string
	accepted         bool
	diagnosticFamily string
}

var ownershipReferencesConformanceCases = []ownershipReferencesConformanceCase{
	{name: "ordinary copyable struct copy preserves source", source: "valid/copyable_struct_copy.concept", expected: ConformanceExpectedDivergence, reference: "Phase 10 implicit-copy rejection; EVT1 structural-copy decision", accepted: true},
	{name: "movable-only explicit move", source: "valid/move_resource.concept", expected: ConformancePass, reference: "Phase 10 move_struct_local_run", accepted: true},
	{name: "explicit move of copyable value is non-consuming", source: "valid/move_copyable_non_consuming.concept", expected: ConformancePass, reference: "Phase 10 move_int_copy_run", accepted: true},
	{name: "reassignment after move", source: "valid/move_reassign.concept", expected: ConformancePass, reference: "Phase 10 reinitialize_after_move_struct_valid", accepted: true},
	{name: "both branch paths move", source: "valid/move_branch.concept", expected: ConformancePass, reference: "Phase 10 branch moved-state fixtures", accepted: true},
	{name: "call transfer", source: "valid/move_call_transfer.concept", expected: ConformancePass, reference: "Phase 10 move_struct_argument_run and drop param pressure", accepted: true},
	{name: "return transfer", source: "valid/move_return_transfer.concept", expected: ConformancePass, reference: "Phase 10 drop_return_move", accepted: true},
	{name: "drop reverse declaration order", source: "valid/drop_scope_order.concept", expected: ConformancePass, reference: "Phase 10 drop_local_reverse_order", accepted: true},
	{name: "moved source drop suppressed", source: "valid/drop_moved_source.concept", expected: ConformancePass, reference: "Phase 10 drop_moved_local_skipped", accepted: true},
	{name: "drop on early return", source: "valid/drop_early_return.concept", expected: ConformancePass, reference: "Phase 10 drop_early_return", accepted: true},
	{name: "mutable ref parameter", source: "valid/ref_mutable_parameter.concept", expected: ConformancePass, reference: "EVT1-new explicit ref evidence", accepted: true},
	{name: "local ref alias", source: "valid/ref_local_alias.concept", expected: ConformancePass, reference: "EVT1-new first-class local ref evidence", accepted: true},
	{name: "ref const parameter", source: "valid/ref_const_parameter.concept", expected: ConformancePass, reference: "EVT1-new explicit ref const evidence", accepted: true},
	{name: "immovable mutable by ref", source: "valid/ref_immovable.concept", expected: ConformancePass, reference: "EVT1-new ref plus R2 immovable final storage", accepted: true},
	{name: "borrow owned then transfer", source: "valid/ref_owned_then_move.concept", expected: ConformancePass, reference: "EVT1-new ref and move distinction", accepted: true},
	{name: "Vulkan inherits core ref", source: "valid/ref_immovable_vulkan.concept", expected: ConformancePass, reference: "EVT1 R1 profile boundary and EVT1-new ref evidence", accepted: true},

	{name: "movable-only copy rejected", source: "invalid/copy_movable_only.concept", expected: ConformancePass, reference: "Phase 10 implicit-copy rejection", diagnosticFamily: "COPY_OF_NONCOPYABLE"},
	{name: "use after move", source: "invalid/use_after_move.concept", expected: ConformancePass, reference: "Phase 10 use_after_move_local", diagnosticFamily: "USE_AFTER_MOVE"},
	{name: "second move", source: "invalid/move_after_move.concept", expected: ConformancePass, reference: "Phase 10 move_after_move", diagnosticFamily: "MOVE_AFTER_MOVE"},
	{name: "immovable move", source: "invalid/move_immovable.concept", expected: ConformancePass, reference: "R2 immovable law plus Phase 10 non-copy pressure", diagnosticFamily: "MOVE_OF_IMMOVABLE"},
	{name: "call without move", source: "invalid/call_without_move.concept", expected: ConformancePass, reference: "Phase 10 implicit_copy_struct_argument", diagnosticFamily: "PASS_NONCOPYABLE_WITHOUT_MOVE"},
	{name: "return without move", source: "invalid/return_without_move.concept", expected: ConformancePass, reference: "Phase 10 implicit_copy_struct_return", diagnosticFamily: "RETURN_NONCOPYABLE_WITHOUT_MOVE"},
	{name: "maybe moved join", source: "invalid/maybe_moved_join.concept", expected: ConformancePass, reference: "Phase 10 maybe_moved_after_if", diagnosticFamily: "MAYBE_MOVED_USE"},
	{name: "maybe moved drop owner", source: "invalid/drop_maybe_moved.concept", expected: ConformancePass, reference: "Phase 10 drop_maybe_moved_cleanup", diagnosticFamily: "DROP_OF_MOVED_VALUE"},
	{name: "runtime while move is conservative", source: "invalid/while_maybe_moved.concept", expected: ConformancePass, reference: "Phase 10 loop fixed-point pressure", diagnosticFamily: "MAYBE_MOVED_USE"},
	{name: "const place to mutable ref", source: "invalid/ref_mutable_from_const.concept", expected: ConformancePass, reference: "EVT1-new ref with R2 const-place law", diagnosticFamily: "REF_MUTABLE_FROM_CONST"},
	{name: "temporary to ref", source: "invalid/ref_temporary.concept", expected: ConformancePass, reference: "EVT1-new non-escaping reference law", diagnosticFamily: "REF_REQUIRES_PLACE"},
	{name: "ref element type mismatch", source: "invalid/ref_type_mismatch.concept", expected: ConformancePass, reference: "EVT1-new reference type evidence", diagnosticFamily: "REF_TYPE_MISMATCH"},
	{name: "mutation through ref const", source: "invalid/ref_const_mutation.concept", expected: ConformancePass, reference: "EVT1-new ref const evidence", diagnosticFamily: "MUTATION_THROUGH_REF_CONST"},
	{name: "record field mutation through ref", source: "invalid/ref_record_field_mutation.concept", expected: ConformancePass, reference: "EVT1-new ref plus R2 record immutability", diagnosticFamily: "RECORD_FIELD_MUTATION"},
	{name: "obvious local ref escape", source: "invalid/ref_escape_local.concept", expected: ConformancePass, reference: "EVT1-new bounded escape rejection", diagnosticFamily: "REF_ESCAPE_LOCAL"},
}

func TestOwnershipReferencesDifferentialConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range ownershipReferencesConformanceCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reference == "" {
				t.Fatal("conformance provenance is required")
			}
			path := filepath.Join("..", "..", "language", "evt1", "ownership", filepath.FromSlash(tc.source))
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
			if _, err := Generate(module, source); err != nil {
				t.Fatal(err)
			}
		})
		counts[tc.expected]++
	}
	t.Logf("R3 conformance summary: PASS=%d EXPECTED-DIVERGENCE=%d EVT1-GAP=%d POC3-LEGACY=%d SPEC-AMBIGUITY=%d", counts[ConformancePass], counts[ConformanceExpectedDivergence], counts[ConformanceEVT1Gap], counts[ConformancePoC3Legacy], counts[ConformanceSpecAmbiguity])
}

func TestOwnershipReferencesConformanceClassificationCounts(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range ownershipReferencesConformanceCases {
		counts[tc.expected]++
	}
	if counts[ConformancePass] != 30 || counts[ConformanceExpectedDivergence] != 1 || len(counts) != 2 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestOwnershipReferencesMIRMoveReferenceAndCleanupFacts(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "ownership", "valid", "drop_moved_source.concept")
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
	if err := json.Unmarshal(outputs["drop_moved_source.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Structs) != 1 || !mir.Structs[0].Movable || !mir.Structs[0].HasDrop || !mir.Structs[0].Copyable {
		t.Fatalf("R3 type facts missing: %+v", mir.Structs)
	}
	main := mir.Functions[1]
	if len(main.Cleanups) != 2 || main.Cleanups[0].Owner != "second" || main.Cleanups[0].State != "live" || main.Cleanups[1].Owner != "first" || main.Cleanups[1].State != "transferred" {
		t.Fatalf("cleanup ownership accounting = %+v", main.Cleanups)
	}
	foundMove := false
	for _, operation := range main.Operations {
		foundMove = foundMove || operation.Kind == "move"
	}
	if !foundMove {
		t.Fatal("move operation missing from MIR")
	}

	refOutputs := generateR3Fixture(t, "ref_const_parameter.concept")
	var refMIR MIR
	if err := json.Unmarshal(refOutputs["ref_const_parameter.mir.json"], &refMIR); err != nil {
		t.Fatal(err)
	}
	foundRefConst := false
	for _, function := range refMIR.Functions {
		for _, operation := range function.Operations {
			foundRefConst = foundRefConst || operation.Kind == "ref_const"
		}
	}
	if !foundRefConst {
		t.Fatal("ref_const operation missing from MIR")
	}
}

func TestOwnershipReferencesMoveAndReferenceSpans(t *testing.T) {
	movePath := filepath.Join("..", "..", "language", "evt1", "ownership", "valid", "move_resource.concept")
	moveSource, err := os.ReadFile(movePath)
	if err != nil {
		t.Fatal(err)
	}
	moveModule, err := Parse(filepath.ToSlash(movePath), string(moveSource))
	if err != nil {
		t.Fatal(err)
	}
	moveExpr := moveModule.Functions[0].Body.Statements[1].(*VarDecl).Value.(*MoveExpr)
	if moveExpr.Span != (Span{Line: 11, Column: 29}) || moveExpr.Value.exprSpan() != (Span{Line: 11, Column: 34}) {
		t.Fatalf("move spans = keyword %+v operand %+v", moveExpr.Span, moveExpr.Value.exprSpan())
	}

	refPath := filepath.Join("..", "..", "language", "evt1", "ownership", "valid", "ref_mutable_parameter.concept")
	refSource, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatal(err)
	}
	refModule, err := Parse(filepath.ToSlash(refPath), string(refSource))
	if err != nil {
		t.Fatal(err)
	}
	call := refModule.Functions[1].Body.Statements[1].(*ExprStmt).Value.(*CallExpr)
	refExpr := call.Args[0].(*RefExpr)
	if refExpr.Span != (Span{Line: 11, Column: 15}) || refExpr.Value.exprSpan() != (Span{Line: 11, Column: 19}) {
		t.Fatalf("ref spans = keyword %+v operand %+v", refExpr.Span, refExpr.Value.exprSpan())
	}
}

func TestOwnershipReferencesDropLoweringOrderAndTransfer(t *testing.T) {
	outputs := generateR3Fixture(t, "drop_scope_order.concept")
	body := string(outputs["drop_scope_order.generated.c"])
	mainAt := strings.Index(body, "int concept_drop_scope_order_main()")
	mainBody := body[mainAt:]
	secondDrop := strings.Index(mainBody, "concept_drop_scope_order_drop(second);")
	firstDrop := strings.Index(mainBody, "concept_drop_scope_order_drop(first);")
	returnValue := strings.Index(mainBody, "cv_return")
	if returnValue < 0 || secondDrop < returnValue || firstDrop < secondDrop {
		t.Fatalf("return evaluation and reverse drop order are not explicit:\n%s", mainBody)
	}

	movedOutputs := generateR3Fixture(t, "drop_moved_source.concept")
	movedBody := string(movedOutputs["drop_moved_source.generated.c"])
	movedMain := movedBody[strings.Index(movedBody, "int concept_drop_moved_source_main()"):]
	if strings.Contains(movedMain, "concept_drop_moved_source_drop(first);") || strings.Count(movedMain, "concept_drop_moved_source_drop(second);") != 1 {
		t.Fatalf("moved source cleanup was not suppressed:\n%s", movedMain)
	}

	callBody := string(generateR3Fixture(t, "move_call_transfer.concept")["move_call_transfer.generated.c"])
	consumeAt := strings.Index(callBody, "int concept_move_call_transfer_consume")
	mainAt = strings.Index(callBody, "int concept_move_call_transfer_main")
	if consumeAt < 0 || mainAt < 0 || !strings.Contains(callBody[consumeAt:mainAt], "concept_move_call_transfer_drop(resource);") || strings.Contains(callBody[mainAt:], "concept_move_call_transfer_drop(resource);") {
		t.Fatalf("callee parameter ownership/drop is not explicit:\n%s", callBody)
	}
}

func TestOwnershipReferencesNativeC11(t *testing.T) {
	cases := []struct {
		fixture string
		call    string
		want    int
	}{
		{fixture: "move_resource.concept", call: "concept_move_resource_main", want: 41},
		{fixture: "move_reassign.concept", call: "concept_move_reassign_main", want: 43},
		{fixture: "move_call_transfer.concept", call: "concept_move_call_transfer_main", want: 42},
		{fixture: "move_return_transfer.concept", call: "concept_move_return_transfer_main", want: 42},
		{fixture: "drop_scope_order.concept", call: "concept_drop_scope_order_main", want: 3},
		{fixture: "ref_mutable_parameter.concept", call: "concept_ref_mutable_parameter_main", want: 42},
		{fixture: "ref_local_alias.concept", call: "concept_ref_local_alias_main", want: 42},
		{fixture: "ref_const_parameter.concept", call: "concept_ref_const_parameter_main", want: 42},
		{fixture: "ref_immovable.concept", call: "concept_ref_immovable_main", want: 42},
		{fixture: "ref_owned_then_move.concept", call: "concept_ref_owned_then_move_main", want: 43},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			outputs := generateR3Fixture(t, tc.fixture)
			base := strings.TrimSuffix(tc.fixture, ".concept")
			harness := "#include \"" + base + ".generated.h\"\n\nint main(void) { return " + tc.call + "() == " + strconv.Itoa(tc.want) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}

func generateR3Fixture(t *testing.T, fixture string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "ownership", "valid", fixture)
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
