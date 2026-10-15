package concept

import (
	"reflect"
	"strings"
	"testing"
)

func TestImportedQueryOnlyTemplateBodiesAreNotEmitted(t *testing.T) {
	const library = `module QueryLibrary; profile Core;
template <typename T> T Leaf(T value) { return value; }
template <typename T> T Outer(T value) { return Leaf<T>(value); }
template <typename T> double QueryOnly(T value) { return Outer<double>(1.0); }
`
	artifact := buildSemanticArtifact(t, "QueryLibrary.concept", library, nil)
	const source = `module QueryConsumer; profile Core;
import QueryLibrary;
int Main(int value) { return Outer<int>(value); }
`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("queries.concept", source, map[string][]byte{"QueryLibrary": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		body := moduleOutput(t, outputs, ".generated.c")
		for _, symbol := range []string{"concept_template_outer__int", "concept_template_leaf__int"} {
			if !strings.Contains(body, symbol+"(") {
				t.Fatalf("runtime transitive dependency lost: %s\n%s", symbol, body)
			}
		}
		for _, symbol := range []string{"concept_template_outer__double", "concept_template_leaf__double"} {
			if strings.Contains(body, symbol) {
				t.Fatalf("query-only instance emitted: %s", symbol)
			}
		}
		assertR8cStrictC11(t, outputs, "queries.generated.c")
		runFoundationNativeHarness(t, outputs, "query.c", `#include "queries.generated.h"
int main(void) {return concept_query_consumer_main(42)==42 ? 0 : 1;}`)
		again, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil || !reflect.DeepEqual(outputs, again) {
			t.Fatalf("template liveness generation is not deterministic: %v", err)
		}
	}
	_, err := ParseWithSemanticModules("bad_query.concept", `module BadQuery; profile Core;
import QueryLibrary;
template <typename T> int Bad(T value) { return Missing(1); }
int Main() { return 0; }`, map[string][]byte{"QueryLibrary": artifact})
	if err == nil {
		t.Fatal("runtime dead emission must not bypass unused body validation")
	}
}

func TestTemplateEmissionIdentifierClosureAndConservativeAsyncRoot(t *testing.T) {
	bodies := map[string]string{
		"f": "f(){g();}", "g": "g(){f();}", "unused": "unused(){unused();}",
		"async": "async_init(){helper();}", "helper": "helper(){}",
	}
	got := evt1LiveTemplateBodies("prefix_f_suffix(); f();", bodies, map[string]bool{"async": true})
	want := map[string]bool{"f": true, "g": true, "async": true, "helper": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbol closure: got %v, want %v", got, want)
	}
	if got := evt1LiveTemplateBodies("prefix_f_suffix();", bodies, nil); len(got) != 0 {
		t.Fatalf("identifier substrings must not root a private body: %v", got)
	}
}
