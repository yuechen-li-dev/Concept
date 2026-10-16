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

func localBoundsOutputs(t *testing.T, source string, verify bool) (Outputs, MIR) {
	t.Helper()
	module, err := Parse("local_bounds.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	policy := ConservativeCompilationPolicy()
	if verify {
		policy = VerifyCompilationPolicy()
	}
	out, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal([]byte(moduleOutput(t, out, ".mir.json")), &mir); err != nil {
		t.Fatal(err)
	}
	return out, mir
}

func spanProofCounts(mir MIR) (proved, checked int) {
	count := func(ops []MIROperation) {
		for _, op := range ops {
			if op.Kind != "span_index" {
				continue
			}
			if op.LocalBounds != nil && op.BoundsCheck == "statically_valid" {
				proved++
			} else {
				checked++
			}
		}
	}
	for _, fn := range mir.Functions {
		count(fn.Operations)
	}
	for _, fn := range mir.Instances {
		count(fn.Operations)
	}
	return
}

func TestLocalBoundsProofKnownViewsAndDominatingChecks(t *testing.T) {
	source := `module LocalBounds; profile Core;
int Known(ReadOnlySpan<int> values) {
    ReadOnlySpan<int> v=Subspan(values,0,4);
    int total=0;
    for (i in 0 .. 4) { total += v[i]; }
    return total;
}
int Prior(ReadOnlySpan<int> values) { int first=values[3]; return first+values[2]; }
int Admission(ReadOnlySpan<int> values,int i) {
    if (Len(values)<4 or i<0 or i>=4) { return 0; }
    return values[i];
}
`
	for _, verify := range []bool{false, true} {
		out, mir := localBoundsOutputs(t, source, verify)
		proved, checked := spanProofCounts(mir)
		if proved != 3 || checked != 1 {
			t.Fatalf("proof counts: %d %d", proved, checked)
		}
		want := 1
		if verify {
			want = 4
		}
		if got := strings.Count(moduleOutput(t, out, ".generated.c"), `"Concept span index out of bounds"`); got != want {
			t.Fatalf("verify=%t guards %d want %d", verify, got, want)
		}
		assertR8cStrictC11(t, out, "local_bounds.generated.c")
		runFoundationNativeHarness(t, out, "local.c", `#include "local_bounds.generated.h"
int main(void){int32_t a[4]={1,2,3,4};concept_readonly_span_int v={a,4};return concept_local_bounds_known(v)==10 && concept_local_bounds_prior(v)==7 && concept_local_bounds_admission(v,2)==3 ? 0:1;}`)
	}
}

func TestLocalBoundsUnknownAndMutationKeepGuards(t *testing.T) {
	cases := map[string]string{
		"runtime":                `int Read(ReadOnlySpan<int> v,int i){return v[i];}`,
		"lower not upper":        `int Read(ReadOnlySpan<int> v,int i){if(i<0 or i>=4){return 0;} int first=v[i];return first+v[3];}`,
		"extent reassignment":    `int Read(ReadOnlySpan<int> a,ReadOnlySpan<int> b){ReadOnlySpan<int> v=Subspan(a,0,4);int first=v[3];v=b;return first+v[3];}`,
		"loop carried mutation":  `int Read(ReadOnlySpan<int> v){int j=0;int total=0;for(i in 0..3){total+=v[j];j=100;}return total;}`,
		"overflow":               `int Read(ReadOnlySpan<int> v){int j=2147483647;return v[j+1];}`,
		"condition right access": `int Read(ReadOnlySpan<int> v){if(2==v[3]){return 1;}return 0;}`,
		"short circuit":          `int Read(ReadOnlySpan<int> v,bool flag){if(flag and v[3]==0){return 1;}return v[3];}`,
		"branch only":            `int Read(ReadOnlySpan<int> v,bool flag){if(flag){int first=v[3];}return v[3];}`,
		"shadow":                 `int Read(ReadOnlySpan<int> v,int j){{int j=0;int x=v[j];}return v[j];}`,
		"narrow wrap":            `int Read(ReadOnlySpan<int> v){uint8 count=255u;count+=2u;ReadOnlySpan<int> s=Subspan(v,0,count as int);return s[100];}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			for _, verify := range []bool{false, true} {
				_, mir := localBoundsOutputs(t, "module LocalBounds; profile Core; "+body, verify)
				_, checked := spanProofCounts(mir)
				if checked == 0 {
					t.Fatal("unknown access lost its check")
				}
			}
		})
	}
}

func TestLocalBoundsClosedInstanceUsesExactExpandedOperation(t *testing.T) {
	source := `module LocalBounds; profile Core;
template<usize N>
int Mixed(ReadOnlySpan<int> values,int i) {
    ReadOnlySpan<int> v=Subspan(values,0,4);int total=0;
    comptime for(k in 0..2){total+=v[k*i];}
    return total;
}
int Run(ReadOnlySpan<int> v,int i){return Mixed<2>(v,i);}
`
	for _, verify := range []bool{false, true} {
		out, mir := localBoundsOutputs(t, source, verify)
		p, c := spanProofCounts(mir)
		if p != 1 || c != 1 {
			t.Fatalf("closed counts=%d/%d", p, c)
		}
		var last Span
		for _, op := range mir.Instances[0].Operations {
			if op.Kind == "span_index" {
				if last.Line != 0 && last != op.SourceSpan {
					t.Fatal("test needs identical source spans")
				}
				last = op.SourceSpan
			}
		}
		want := 1
		if verify {
			want = 2
		}
		if got := strings.Count(moduleOutput(t, out, ".generated.c"), `"Concept span index out of bounds"`); got != want {
			t.Fatalf("guards=%d want %d\n%s", got, want, moduleOutput(t, out, ".generated.c"))
		}
		runFoundationNativeHarness(t, out, "mixed.c", `#include "local_bounds.generated.h"
int main(void){int32_t a[4]={2,4,6,8};return concept_local_bounds_run((concept_readonly_span_int){a,4},2)==8?0:1;}`)
	}
}

func TestLocalBoundsDoesNotUseEscapedOrAliasedDescriptor(t *testing.T) {
	source := `module LocalBounds; profile Core;
extern "C" void Opaque(ref ReadOnlySpan<int> view);
int Read(ReadOnlySpan<int> values) {
    ReadOnlySpan<int> view=Subspan(values,0,4);
    Opaque(ref view);
    return view[3];
}

int Aliased(ReadOnlySpan<int> input,Span<int> output,int i) {
    output[0]=7;return input[i];
}
`
	for _, verify := range []bool{false, true} {
		_, mir := localBoundsOutputs(t, source, verify)
		p, c := spanProofCounts(mir)
		if p != 0 || c != 3 {
			t.Fatalf("opaque/aliased counts=%d/%d", p, c)
		}
	}
}

func TestLocalBoundsImportedGenericDeterminism(t *testing.T) {
	library := `module BoundLibrary; profile Core;
template<usize N> int Sum(ReadOnlySpan<int> values){
 ReadOnlySpan<int> v=Subspan(values,0,N as int);int total=0;
 for(i in 0..(N as int)){total+=v[i];}return total;
}`
	artifact := buildSemanticArtifact(t, "BoundLibrary.concept", library, nil)
	source := `module BoundUse; profile Core; import BoundLibrary; int Run(ReadOnlySpan<int> v){return Sum<4>(v);}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		m, err := ParseWithSemanticModules("local_bounds.concept", source, map[string][]byte{"BoundLibrary": artifact})
		if err != nil {
			t.Fatal(err)
		}
		out, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		again, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
		if err != nil || !reflect.DeepEqual(out, again) {
			t.Fatalf("nondeterministic: %v", err)
		}
		var mir MIR
		json.Unmarshal([]byte(moduleOutput(t, out, ".mir.json")), &mir)
		p, c := spanProofCounts(mir)
		if p != 1 || c != 0 {
			t.Fatalf("import proof=%d/%d", p, c)
		}
		runFoundationNativeHarness(t, out, "import.c", `#include "local_bounds.generated.h"
int main(void){int32_t a[4]={1,2,3,4};return concept_bound_use_run((concept_readonly_span_int){a,4})==10?0:1;}`)
	}
}

func TestLocalBoundsPreservesFirstFailure(t *testing.T) {
	cases := []struct{ name, body, extra, call, reason string }{
		{"short workspace", `int Run(ReadOnlySpan<int> v){ReadOnlySpan<int> s=Subspan(v,0,4);return s[3];}`, "", `concept_local_bounds_run((concept_readonly_span_int){a,1})`, "Concept span bounds out of range"},
		{"lower endpoint", `int Run(ReadOnlySpan<int> v,int i){if(i<0 or i>=4){return 0;}int first=v[i];return first+v[3];}`, "", `concept_local_bounds_run((concept_readonly_span_int){a,1},0)`, "Concept span index out of bounds"},
		{"overflow before bounds", `int Run(ReadOnlySpan<int> v){int j=2147483647;return v[j+1];}`, "", `concept_local_bounds_run((concept_readonly_span_int){a,4})`, "int32 addition overflow"},
		{"opaque mutation", `extern "C" void Opaque(ref ReadOnlySpan<int> v); int Run(ReadOnlySpan<int> v){ReadOnlySpan<int> s=Subspan(v,0,4);Opaque(ref s);return s[3];}`, `void Opaque(concept_readonly_span_int* v){v->length=1;}`, `concept_local_bounds_run((concept_readonly_span_int){a,4})`, "Concept span index out of bounds"},
		{"mixed expanded index", `template<usize N> int Mixed(ReadOnlySpan<int> v,int i){ReadOnlySpan<int> s=Subspan(v,0,4);int total=0;comptime for(k in 0..2){total+=s[k*i];}return total;} int Run(ReadOnlySpan<int> v){return Mixed<2>(v,4);}`, "", `concept_local_bounds_run((concept_readonly_span_int){a,4})`, "Concept span index out of bounds"},
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, verify := range []bool{false, true} {
				out, _ := localBoundsOutputs(t, "module LocalBounds; profile Core; "+c.body, verify)
				dir := t.TempDir()
				if err := Write(dir, out); err != nil {
					t.Fatal(err)
				}
				harness := filepath.Join(dir, "main.c")
				os.WriteFile(harness, []byte("#include \"local_bounds.generated.h\"\n"+c.extra+"\nint main(void){int32_t a[4]={1,2,3,4};return "+c.call+";}"), 0644)
				exe := filepath.Join(dir, "run.exe")
				if log, err := nativeCommand(t, clang, withHostLinkArgs("-std=c11", "-pedantic-errors", "-O2", "-ffp-contract=off", filepath.Join(dir, "local_bounds.generated.c"), harness, "-o", exe)...).CombinedOutput(); err != nil {
					t.Fatalf("compile: %v %s", err, log)
				}
				log, err := nativeCommand(t, exe).CombinedOutput()
				if err == nil || !strings.Contains(string(log), c.reason) {
					t.Fatalf("verify=%t wrong failure: %v %s", verify, err, log)
				}
			}
		})
	}
}
