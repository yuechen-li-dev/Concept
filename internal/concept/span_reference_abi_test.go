package concept

import "testing"

func TestSpanDescriptorReferenceABI(t *testing.T) {
	const source = `module SpanReference; profile Core;
extern "C" void Replace(ref ReadOnlySpan<int> view,ReadOnlySpan<int> other);
extern "C" int Count(ref const ReadOnlySpan<int> view);
int Run(ReadOnlySpan<int> input,ReadOnlySpan<int> other) {
 ReadOnlySpan<int> view=input;Replace(ref view,other);return Count(ref const view);
}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("span_reference.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		out, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, out, "span_reference.generated.c")
		runFoundationNativeHarness(t, out, "ref.c", `#include "span_reference.generated.h"
void Replace(concept_readonly_span_int* v,concept_readonly_span_int other){*v=other;}
int32_t Count(const concept_readonly_span_int* v){return (int32_t)v->length;}
int main(void){int32_t a[4]={1,2,3,4};return concept_span_reference_run((concept_readonly_span_int){a,4},(concept_readonly_span_int){a,1})==1?0:1;}`)
	}
}
