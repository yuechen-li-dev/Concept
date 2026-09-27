package concept

import "testing"

func TestR7pConceptIdentifiersThatAreCKeywordsLowerSafely(t *testing.T) {
	const source = `profile Core;
int Add(int short) { int long = 2; return short + long; }
int Main() { return Add(3); }
`
	module, err := Parse("keyword.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "keyword_harness.c", `#include "keyword.generated.h"
int main(void) { return concept_keyword_main() == 5 ? 0 : 1; }
`)
}
