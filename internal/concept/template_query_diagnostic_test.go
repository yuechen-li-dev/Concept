package concept

import "testing"

func TestEarlyTemplateTypeQueryDiagnosesInsteadOfPanicking(t *testing.T) {
	const source = `module EarlyQuery; profile Core;
template <typename T> T Identity(T value) { return value; }
using Unsupported = typeof(Identity<int>(1));
int Main() { return 0; }`
	if _, err := Parse("early_query.concept", source); diagnosticCode(err) != "CALLABLE_TYPE_QUERY_INVALID" {
		t.Fatalf("unsupported early template query must diagnose: %v", err)
	}
	const supported = `module ExactQuery; profile Core;
auto Make() { return callback() { return 3; }; }
using Exact = typeof(Make());
int Main() { Exact value = Make(); return value(); }`
	module, err := Parse("exact_query.concept", supported)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(supported))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "exact.c", `#include "exact_query.generated.h"
int main(void) {return concept_exact_query_main()==3 ? 0 : 1;}`)
}
