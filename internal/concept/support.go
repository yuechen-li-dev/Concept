// Package concept implements the active Concept EVT1 Stage 0 compiler.
//
// It was extracted from Oct's Concept/Vulkan compiler at R0. Core syntax,
// semantic analysis, MIR, and C11 lowering live here. Vulkan admissions remain
// explicit through `profile Vulkan;` and the profile-specific validation and
// lowering seams; Oct is not a build-time dependency.
package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Span identifies a one-based source position.
type Span struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Token retains the lexeme and its source provenance.
type Token struct {
	Lexeme string
	Span   Span
}

// Diagnostic is a stable compiler error with a source position.
type Diagnostic struct {
	Code    string
	Message string
	Span    Span
}

// SemanticCategory returns the stable diagnostic family used by differential
// conformance. Numeric CV codes remain implementation-local during EVT1
// reconciliation.
func (d Diagnostic) SemanticCategory() string {
	switch d.Code {
	case "PLAN_UNSUPPORTED_MIR", "PLAN_TARGET_UNSUPPORTED", "PLAN_REQUIRED_FACT_MISSING", "PLAN_REQUIRED_FACT_DISPROVEN", "PLAN_INVALID_STRATEGY", "PLAN_RUNTIME_GUARD_MISSING", "PLAN_CLEANUP_INCOMPLETE", "PLAN_WITNESS_INVALID", "PLAN_TENSOR_INVALID", "PLAN_ARTIFACT_INVALID", "PLAN_AUTOMATA_INVALID", "PLAN_CALLABLE_INVALID":
		return "PLANNER_INVALID"
	case "CALLABLE_CAPTURE_REQUIRED", "CALLABLE_CAPTURE_DUPLICATE", "CALLABLE_CAPTURE_UNKNOWN", "CALLABLE_CAPTURE_COPY_NONCOPYABLE", "CALLABLE_CAPTURE_INVALID", "CALLABLE_CAPTURE_LIFETIME_INVALID", "CALLABLE_CONST_MUTATING_INVOKE", "CALLABLE_ARITY_MISMATCH", "CALLABLE_MIR_INVALID", "CALLBACK_SIGNATURE_INVALID", "CALLBACK_SIGNATURE_MISMATCH", "CALLBACK_OWNING_TEMPORARY_REJECTED", "CALLBACK_REQUIRES_EXPLICIT_BORROW", "CALLBACK_REQUIRES_CONCRETE_CALLABLE":
		return d.Code
	case "CV4024":
		return "UNKNOWN_NAME"
	case "CV4127":
		return "NOT_ASSIGNABLE"
	case "CV4128":
		return "CONST_MUTATION"
	case "CV4142":
		return "RECORD_FIELD_MUTATION"
	case "CV4026", "CV4144":
		return "UNKNOWN_FIELD"
	case "CV4145":
		return "DUPLICATE_WITH_FIELD"
	case "CV4147":
		return "WITH_FIELD_TYPE_MISMATCH"
	case "CV4143":
		return "WITH_REQUIRES_RECORD"
	case "CV4146":
		return "WITH_NONCOPYABLE_RECORD"
	case "CV4134":
		return "IMMOVABLE_COPY"
	case "CV4135":
		return "IMMOVABLE_ASSIGNMENT"
	case "CV4136":
		return "IMMOVABLE_BY_VALUE_PARAMETER"
	case "CV4137":
		return "IMMOVABLE_BY_VALUE_RETURN"
	case "CV4138":
		return "IMMOVABLE_EMBEDDING"
	case "CV4139":
		return "IMMOVABLE_ENUM_PAYLOAD"
	case "CV4107":
		return "VALUE_TYPE_MISMATCH"
	case "CV4009":
		return "EXPLICIT_LOCAL_TYPE_REQUIRED"
	case "CV4501":
		return "COPY_OF_NONCOPYABLE"
	case "CV4502":
		return "USE_AFTER_MOVE"
	case "CV4503":
		return "MAYBE_MOVED_USE"
	case "CV4504":
		return "MOVE_AFTER_MOVE"
	case "CV4505":
		return "MOVE_OF_IMMOVABLE"
	case "CV4506":
		return "RETURN_NONCOPYABLE_WITHOUT_MOVE"
	case "CV4507":
		return "MOVE_REQUIRES_PLACE"
	case "CV4508":
		return "REF_REQUIRES_PLACE"
	case "CV4509":
		return "REF_MUTABLE_FROM_CONST"
	case "CV4510":
		return "REF_TYPE_MISMATCH"
	case "CV4511":
		return "REF_ESCAPE_LOCAL"
	case "CV4512":
		return "PASS_NONCOPYABLE_WITHOUT_MOVE"
	case "CV4513":
		return "MUTATION_THROUGH_REF_CONST"
	case "CV4514":
		return "USE_BEFORE_INITIALIZATION"
	case "CV4515":
		return "REFERENCE_AGGREGATE_DEFERRED"
	case "CV4516":
		return "DROP_OF_MOVED_VALUE"
	case "CV4521":
		return "REF_STRUCT_ESCAPE"
	case "CV4522":
		return "SCOPED_ESCAPE"
	case "CV4523":
		return "LIFETIME_OUTLIVES_VIOLATION"
	case "CV4524":
		return "LIFETIME_REQUIREMENT_UNSATISFIED"
	case "CV4525":
		return "REF_STRUCT_INVALID_FIELD"
	case "CV4526":
		return "COMPILER_ANALYSIS_REQUIREMENT_FAILED"
	case "CV4527":
		return "OUTLIVES_REQUIREMENT_FAILED"
	case "CV4528":
		return "OUTLIVES_PROVENANCE_UNKNOWN"
	case "CV4529":
		return "CALL_RESULT_LIFETIME_UNKNOWN"
	case "CV4530":
		return "CALL_RESULT_ESCAPES_SOURCE"
	case "CV4531":
		return "SCOPED_RESULT_ESCAPE"
	case "CV4532":
		return "RELATIONAL_REQUIREMENT_INVALID_SUBJECT"
	case "CV4533":
		return "RELATIONAL_REQUIREMENT_UNSUPPORTED"
	case "CV4540":
		return "FAILURE_TYPE_INVALID"
	case "CV4541":
		return "QUESTION_REQUIRES_OPTION_OR_RESULT"
	case "CV4542":
		return "QUESTION_INCOMPATIBLE_RETURN_CHANNEL"
	case "CV4543":
		return "RESULT_ERROR_TYPE_MISMATCH"
	case "CV4544":
		return "BANG_REQUIRES_OPTION_OR_RESULT"
	case "CV4545":
		return "TRY_EXCEPT_UNHANDLED_ERROR"
	case "CV4546":
		return "ASSERT_REQUIRES_BOOL"
	case "CV4547":
		return "TRY_EXCEPT_DUPLICATE_ERROR"
	case "CV4548":
		return "RESULT_PAYLOAD_NONTRANSFERABLE"
	case "CV4549":
		return "FAILURE_PAYLOAD_IMMOVABLE"
	case "CV4550":
		return "TRY_EXCEPT_UNKNOWN_ERROR_TYPE"
	case "CV4555":
		return "NDARRAY_LITERAL_SHAPE_MISMATCH"
	case "CV4562":
		return "BIND_REQUIRES_CONTEXTUAL_TARGET"
	case "CV4563":
		return "BIND_REQUIRES_ARRAY_TARGET"
	case "CV4564":
		return "BIND_REQUIRES_STORAGE"
	case "CV4565":
		return "BIND_ELEMENT_TYPE_MISMATCH"
	case "CV4566":
		return "BIND_SHAPE_SIZE_MISMATCH"
	case "CV4567":
		return "BIND_MUTABLE_FROM_CONST"
	case "CV4568":
		return "BIND_MIR_INVALID"
	case "CV4570":
		return "LAYOUT_DECLARATION_INVALID"
	case "CV4571":
		return "LAYOUT_DUPLICATE_REGION"
	case "CV4572":
		return "LAYOUT_INVALID_ALIGNMENT"
	case "CV4573":
		return "LAYOUT_EXTENT_INVALID"
	case "CV4574":
		return "LAYOUT_OFFSET_MISALIGNED"
	case "CV4575":
		return "LAYOUT_RUNTIME_PARAMETER_DEFERRED"
	case "CV4576":
		return "LAYOUT_REGION_OVERLAP"
	case "CV4577":
		return "LAYOUT_REFERENCE_REQUIRED"
	case "CV4578":
		return "LAYOUT_BIND_REQUIRES_STORAGE"
	case "CV4579":
		return "LAYOUT_MUTABLE_FROM_CONST"
	case "CV4580":
		return "STREAM_DECLARATION_INVALID"
	case "CV4581":
		return "STREAM_UNKNOWN_LAYOUT"
	case "CV4582":
		return "STREAM_UNKNOWN_REGION"
	case "CV4583":
		return "STREAM_DUPLICATE_CHANNEL"
	case "CV4584":
		return "STREAM_REFERENCE_REQUIRED"
	case "CV4585":
		return "STREAM_BIND_LAYOUT_MISMATCH"
	case "CV4586":
		return "STREAM_MUTABLE_FROM_CONST"
	case "CV4590":
		return "LAYOUT_BIND_SIZE_MISMATCH"
	case "CV4591":
		return "LAYOUT_BIND_ALIGNMENT_MISMATCH"
	case "CV4592":
		return "LAYOUT_QUERY_INVALID"
	case "CV4593":
		return "LAYOUT_MIR_INVALID"
	case "CV4594":
		return "STREAM_MIR_INVALID"
	case "CV4600":
		return "SPAN_REQUIRES_CONTIGUOUS_SOURCE"
	case "CV4601":
		return "SPAN_ELEMENT_TYPE_MISMATCH"
	case "CV4602":
		return "SPAN_MUTABLE_FROM_CONST"
	case "CV4603":
		return "SPAN_BOUNDS_OUT_OF_RANGE"
	case "CV4604":
		return "SPAN_INDEX_OUT_OF_BOUNDS"
	case "CV4605":
		return "READONLY_SPAN_MUTATION"
	case "CV4606":
		return "SPAN_OFFSET_OVERFLOW"
	case "CV4607":
		return "SPAN_LENGTH_OVERFLOW"
	case "CV4608":
		return "SPAN_MIR_INVALID"
	case "CV4610":
		return "TENSOR_RANK_INVALID"
	case "CV4611":
		return "TENSOR_DESTINATION_STORAGE_REQUIRED"
	case "CV4612":
		return "TENSOR_ELEMENT_TYPE_UNSUPPORTED"
	case "CV4613":
		return "TENSOR_REQUIRES_SHAPED_CONTIGUOUS_SOURCE"
	case "CV4614":
		return "TENSOR_RANK_MISMATCH"
	case "CV4615":
		return "TENSOR_ELEMENT_TYPE_MISMATCH"
	case "CV4616":
		return "TENSOR_READONLY_DESTINATION"
	case "CV4617":
		return "TENSOR_SHAPE_MISMATCH"
	case "CV4618":
		return "TENSOR_INDEX_ARITY_MISMATCH"
	case "CV4619":
		return "TENSOR_SYMBOLIC_INDEX_INVALID"
	case "CV4620":
		return "TENSOR_SYMBOLIC_INDEX_EXTENT_MISMATCH"
	case "CV4621":
		return "TENSOR_OUTPUT_INDEX_INVALID"
	case "CV4622":
		return "TENSOR_REDUCTION_INDEX_INVALID"
	case "CV4624":
		return "TENSOR_CONTRACTION_SHAPE_MISMATCH"
	case "CV4625":
		return "TENSOR_ALIAS_HAZARD"
	case "CV4626":
		return "TENSOR_MIR_INVALID"
	case "CV4627":
		return "TENSOR_INLINE_REQUIRES_FIXED_SHAPE"
	case "CV4628":
		return "VECTOR_RANK_MISMATCH"
	case "CV4629":
		return "MATRIX_RANK_MISMATCH"
	case "CV4630":
		return "TENSOR_INLINE_SHAPE_INVALID"
	case "CV4631":
		return "TENSOR_INLINE_ELEMENT_TYPE_MISMATCH"
	case "CV4632":
		return "TENSOR_SCALAR_RESULT_TYPE_MISMATCH"
	case "CV4640":
		return "SEMANTIC_FACT_UNSATISFIED"
	case "CV4641":
		return "SEMANTIC_FACT_UNKNOWN"
	case "CV4642":
		return "SEMANTIC_FACT_INVALID_SUBJECT"
	case "CV4643":
		return "SEMANTIC_FACT_INVALID_PARAMETER"
	case "CV4644":
		return "ALIGNMENT_REQUIREMENT_FAILED"
	case "CV4645":
		return "DISJOINT_REQUIREMENT_FAILED"
	case "CV4646":
		return "CONTIGUITY_REQUIREMENT_FAILED"
	case "CV4647":
		return "SHAPE_REQUIREMENT_FAILED"
	case "CV4648":
		return "MUTABILITY_REQUIREMENT_FAILED"
	case "CV4649":
		return "SEMANTIC_FACT_MIR_INVALID"
	default:
		return d.Code
	}
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("concept:%d:%d: %s: %s", d.Span.Line, d.Span.Column, d.Code, d.Message)
}

// Outputs is the deterministic generated-artifact set keyed by relative path.
type Outputs map[string][]byte

func digest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Write materializes generated outputs under dir in stable filename order.
func Write(dir string, outputs Outputs) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(outputs))
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := os.WriteFile(filepath.Join(dir, key), outputs[key], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Check verifies that dir contains byte-identical generated outputs.
func Check(dir string, outputs Outputs) error {
	for name, want := range outputs {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("CV3001 stale generated output %s: %w", name, err)
		}
		if string(got) != string(want) {
			return fmt.Errorf("CV3001 stale or hand-edited generated output %s", name)
		}
	}
	return nil
}
