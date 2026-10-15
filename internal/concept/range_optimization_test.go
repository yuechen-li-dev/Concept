package concept

import (
	"strings"
	"testing"
)

func TestUnitStepRangePreservesIntegerLimitsAndGeneralFallback(t *testing.T) {
	const source = `module RangeLimits; profile Core;
int Main() {
    int count = 0;
    for (i in 2147483645..2147483647) { count++; i = 0; }
    for (i in -2147483646..-2147483648 descend 1) { count++; }
    uint64 lo = 18446744073709551613;
    uint64 hi = 18446744073709551615;
    for (i in lo..hi step 1) { count++; }
    for (i in hi..lo descend 1) { count++; }
    for (i in hi..hi) { count += 100; }
    return count;
}
int General(int step) {
    int count = 0;
    for (i in 2147483644..2147483647 step step) { count++; }
    for (i in -2147483645..-2147483648 descend step) { count++; }
    return count;
}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("range_limits.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		body := moduleOutput(t, outputs, ".generated.c")
		if strings.Count(body, "unit step stays within endpoint") != 5 || strings.Count(body, "uint64_t cv_range_remaining_") != 2 {
			t.Fatal("unit-step specialization or dynamic-step fallback missing")
		}
		assertR8cStrictC11(t, outputs, "range_limits.generated.c")
		runFoundationNativeHarness(t, outputs, "limits.c", `#include "range_limits.generated.h"
int main(void) {
 return concept_range_limits_main()==8 && concept_range_limits_general(2)==4 && concept_range_limits_general(1)==6 ? 0 : 1;
}`)
	}
}
