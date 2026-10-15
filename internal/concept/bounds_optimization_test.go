package concept

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStaticArrayBoundsPlanIsPerAccessAndVerifyRetainsChecks(t *testing.T) {
	const source = `module ArrayBounds; profile Core;
int Read(ref const int<array>[4] values, int i) { return values[2] + values[i]; }
int SpanRead(ReadOnlySpan<int> values) { return values[2]; }
`
	for _, verify := range []bool{false, true} {
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		module, err := Parse("array_bounds.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		var mir MIR
		if err := json.Unmarshal([]byte(moduleOutput(t, outputs, ".mir.json")), &mir); err != nil {
			t.Fatal(err)
		}
		static, dynamic := 0, 0
		for _, fn := range mir.Functions {
			for _, op := range fn.Operations {
				if op.BoundsCheck == "statically_valid" {
					static++
				}
				if op.Kind == "span_index" || op.Kind == "array_index" && op.BoundsCheck != "statically_valid" {
					dynamic++
				}
			}
		}
		if static != 1 || dynamic != 2 {
			t.Fatalf("incorrect static evidence: static=%d dynamic=%d", static, dynamic)
		}
		body := moduleOutput(t, outputs, ".generated.c")
		checks := strings.Count(body, "if (")
		want := 3 // one dynamic array bound, one span bound, one integer overflow
		if verify {
			want++
		}
		if checks != want {
			t.Fatalf("verify=%t checks=%d want=%d\n%s", verify, checks, want, body)
		}
		assertR8cStrictC11(t, outputs, "array_bounds.generated.c")
		runFoundationNativeHarness(t, outputs, "bounds.c", `#include "array_bounds.generated.h"
int main(void) {int32_t values[4] = {1,2,3,4}; return concept_array_bounds_read((concept_ref_const_array_1_int){.data=values,.shape={4}},1)==5 ? 0 : 1;}`)
	}
	if _, err := Parse("bad.concept", `profile Core; int Read(ref const int<array>[4] values) { return values[4]; }`); diagnosticCode(err) != "CV4233" {
		t.Fatalf("invalid constant must still reject: %v", err)
	}
}
