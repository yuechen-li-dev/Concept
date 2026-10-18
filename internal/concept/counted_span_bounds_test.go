package concept

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCountedSpanBoundsRuntimeExtent(t *testing.T) {
	source := `module LocalBounds; profile Core;
int Sum(ReadOnlySpan<int> v) { int sum=0; for(i in 0 .. Len(v)) { sum+=v[i]; } return sum; }
int Alias(Span<int> v, ReadOnlySpan<int> input) { for(i in 0 .. Len(v)) { v[i]=input[i]; } return v[0]; }
`
	for _, verify := range []bool{false, true} {
		out, mir := localBoundsOutputs(t, source, verify)
		proved, checked := spanProofCounts(mir)
		if proved != 2 || checked != 2 {
			t.Fatalf("proofs=%d checked=%d", proved, checked)
		}
		guards := strings.Count(moduleOutput(t, out, ".generated.c"), `"Concept span index out of bounds"`)
		want := 2
		if verify {
			want = 4
		}
		if guards != want {
			t.Fatalf("guards=%d want=%d", guards, want)
		}
		runFoundationNativeHarness(t, out, "counted.c", `#include "local_bounds.generated.h"
int main(void){ int32_t a[4]={1,2,3,4}; concept_readonly_span_int r={a,4}; concept_span_int w={a,4};
if(concept_local_bounds_sum(r)!=10 || concept_local_bounds_sum((concept_readonly_span_int){a,0})!=0) return 1;
/* Deliberate overlap: no restrict contract is granted. */
if(concept_local_bounds_alias(w,r)!=1 || a[3]!=4) return 1;
concept_span_int shifted={a+1,3};concept_readonly_span_int prefix={a,3};
return concept_local_bounds_alias(shifted,prefix)==1 && a[3]==1 ? 0:1; }`)
	}
}

func TestCountedSpanBoundsMaximumStartKeepsRangeFailure(t *testing.T) {
	source := `module LocalBounds;profile Core;
int Read(ReadOnlySpan<int> v){for(i in 2147483647..Len(v)){return v[i+0];}return 0;}`
	for _, verify := range []bool{false, true} {
		out, _ := localBoundsOutputs(t, source, verify)
		runCountedSpanFailure(t, out, `#include "local_bounds.generated.h"
int main(void){int32_t a[1]={7};return concept_local_bounds_read((concept_readonly_span_int){a,1});}`, "invalid range step or direction")
	}
}

func TestCountedSpanBoundsInvalidation(t *testing.T) {
	cases := map[string]string{
		"different view":          `int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){int s=0;for(i in 0..Len(v)){s+=w[i];}return s;}`,
		"reassigned view":         `int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){int s=0;for(i in 0..Len(v)){v=w;s+=v[i];}return s;}`,
		"loop carried descriptor": `int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){int s=0;for(i in 0..Len(v)){s+=v[i];v=w;}return s;}`,
		"escaped descriptor":      `extern "C" void Change(ref ReadOnlySpan<int> v); int Read(ReadOnlySpan<int> v){Change(ref v);int s=0;for(i in 0..Len(v)){s+=v[i];}return s;}`,
		"index arithmetic":        `int Read(ReadOnlySpan<int> v){int s=0;for(i in 0..Len(v)){s+=v[i+1];}return s;}`,
		"index reassignment":      `int Read(ReadOnlySpan<int> v){int s=0;for(i in 0..Len(v)){int j=i;j=100;s+=v[j];}return s;}`,
		"shadow view":             `int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){int s=0;for(i in 0..Len(v)){{ReadOnlySpan<int> v=w;s+=v[i];}}return s;}`,
		"shadow Len":              `int Len(ReadOnlySpan<int> v){return 100;} int Read(ReadOnlySpan<int> v){int s=0;for(i in 0..Len(v)){s+=v[i];}return s;}`,
		"while loop":              `int Read(ReadOnlySpan<int> v){int s=0;int i=0;while(i<Len(v)) bounded(100){s+=v[i];i+=1;}return s;}`,
		"captured callback":       `int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){auto change=callback() with(ref v,w){v=w;};int s=0;for(i in 0..Len(v)){change();s+=v[i];}return s;}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			for _, verify := range []bool{false, true} {
				out, mir := localBoundsOutputs(t, "module LocalBounds;profile Core;"+body, verify)
				proved, checked := spanProofCounts(mir)
				if proved != 0 || checked == 0 {
					t.Fatalf("%d proved %d checked", proved, checked)
				}
				if !strings.Contains(moduleOutput(t, out, ".generated.c"), `"Concept span index out of bounds"`) {
					t.Fatal("lost guard")
				}
			}
		})
	}
}

func TestCountedSpanBoundsImportedGenericDeterminism(t *testing.T) {
	library := `module Counted;profile Core;template<usize Salt> uint64 Sum(ReadOnlySpan<uint32> v){uint64 sum=0u;for(i in 0..Len(v)){sum+=v[i] as uint64;}return sum;}`
	artifact := buildSemanticArtifact(t, "Counted.concept", library, nil)
	source := `module LocalBounds;profile Core;import Counted;uint64 Run(ReadOnlySpan<uint32> v){return Sum<1>(v);}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		m, err := ParseWithSemanticModules("local_bounds.concept", source, map[string][]byte{"Counted": artifact})
		if err != nil {
			t.Fatal(err)
		}
		out, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		again, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
		if err != nil || !reflect.DeepEqual(out, again) {
			t.Fatalf("determinism: %v", err)
		}
		var mir MIR
		if err := json.Unmarshal([]byte(moduleOutput(t, out, ".mir.json")), &mir); err != nil {
			t.Fatal(err)
		}
		proved, checked := spanProofCounts(mir)
		if proved != 1 || checked != 0 {
			t.Fatalf("proof=%d checked=%d", proved, checked)
		}
		runFoundationNativeHarness(t, out, "import.c", `#include "local_bounds.generated.h"
int main(void){uint32_t a[4]={1,2,3,4};return concept_local_bounds_run((concept_readonly_span_uint32){a,4})==10 ? 0:1;}`)
	}
}

func TestCountedSpanBoundsShortSecondaryInputFails(t *testing.T) {
	source := `module LocalBounds;profile Core;
int Read(Span<int> v,ReadOnlySpan<int> w){for(i in 0..Len(v)){v[i]=w[i];}return 0;}`
	for _, verify := range []bool{false, true} {
		out, _ := localBoundsOutputs(t, source, verify)
		runCountedSpanFailure(t, out, `#include "local_bounds.generated.h"
int main(void){int32_t a[4]={0};int32_t b[1]={1};return concept_local_bounds_read((concept_span_int){a,4},(concept_readonly_span_int){b,1});}`, "Concept span index out of bounds")
	}
}

func TestLocalBoundsShorthandCaptureMutationKeepsFailure(t *testing.T) {
	source := `module LocalBounds;profile Core;
int Read(ReadOnlySpan<int> v,ReadOnlySpan<int> w){ReadOnlySpan<int> s=Subspan(v,0,4);auto change=callback() with(ref s,w){s=w;};change();return s[3];}`
	for _, verify := range []bool{false, true} {
		out, mir := localBoundsOutputs(t, source, verify)
		proved, checked := spanProofCounts(mir)
		if proved != 0 || checked != 1 {
			t.Fatalf("proof=%d checked=%d", proved, checked)
		}
		runCountedSpanFailure(t, out, `#include "local_bounds.generated.h"
int main(void){int32_t a[4]={1,2,3,4};return concept_local_bounds_read((concept_readonly_span_int){a,4},(concept_readonly_span_int){a,1});}`, "Concept span index out of bounds")
	}
}

func runCountedSpanFailure(t *testing.T, out Outputs, harness, reason string) {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	dir := t.TempDir()
	if err := Write(dir, out); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.c")
	if err := os.WriteFile(main, []byte(harness), 0644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "run.exe")
	if log, err := nativeCommand(t, clang, withHostLinkArgs("-std=c11", "-pedantic-errors", "-O2", filepath.Join(dir, "local_bounds.generated.c"), main, "-o", exe)...).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v %s", err, log)
	}
	log, err := nativeCommand(t, exe).CombinedOutput()
	if err == nil || !strings.Contains(string(log), reason) {
		t.Fatalf("wrong failure: %v %s", err, log)
	}
}
