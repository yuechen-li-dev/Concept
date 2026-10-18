package concept

import (
	"reflect"
	"strings"
	"testing"
)

func TestValueParameterSnapshotPreservesAliasingAndMutation(t *testing.T) {
	const source = `module ValueParameter; profile Core;
struct Pair {double x; double y;}
template<usize N> double Value(Pair pair,Span<double> out) {
 double old=pair.x;out[0]=9.0;if(pair.x!=old){return -1.0;}
 pair.x=5.0;return pair.x+pair.y;
}
template<usize N> double Alias(ReadOnlySpan<double> input,Span<double> out) {
 double old=input[0];out[0]=old+1.0;return input[0];
}
double Run(Pair pair,Span<double> out){return Value<1>(pair,out);}
double Shared(ReadOnlySpan<double> input,Span<double> out){return Alias<1>(input,out);}
`
	for _, verify := range []bool{false, true} {
		out, _ := localBoundsOutputs(t, source, verify)
		c := moduleOutput(t, out, ".generated.c")
		if strings.Count(c, "/* source value snapshot */") != 4 {
			t.Fatal("missing bounded private value snapshots")
		}
		if strings.Contains(c, "restrict") {
			t.Fatal("value snapshots must not claim independent backing storage")
		}
		assertR8cStrictC11(t, out, "local_bounds.generated.c")
		runFoundationNativeHarness(t, out, "values.c", `#include "local_bounds.generated.h"
int main(void){concept_pair p={1.0,2.0};concept_span_double out={&p.x,1};
if(concept_value_parameter_run(p,out)!=7.0||p.x!=9.0||p.y!=2.0)return 1;
double v=6.0;if(concept_value_parameter_shared((concept_readonly_span_double){&v,1},(concept_span_double){&v,1})!=7.0||v!=7.0)return 2;return 0;}`)
	}
}

func TestValueParameterSnapshotKeepsReferenceDescriptorMutation(t *testing.T) {
	const source = `module ValueReference;profile Core;
extern "C" void Replace(ref ReadOnlySpan<int> view,ReadOnlySpan<int> other);
template<usize N> int Change(ref ReadOnlySpan<int> view,ReadOnlySpan<int> other) {
 Replace(ref view,other);return Len(view);
}
int Run(ReadOnlySpan<int> input,ReadOnlySpan<int> other) {
 ReadOnlySpan<int> v=input;return Change<1>(ref v,other);
}`
	for _, verify := range []bool{false, true} {
		out, _ := localBoundsOutputs(t, source, verify)
		c := moduleOutput(t, out, ".generated.c")
		if strings.Count(c, "/* source value snapshot */") != 1 || strings.Contains(c, " view = cv_parameter_value") {
			t.Fatal("ref descriptor was copied rather than retained by reference")
		}
		runFoundationNativeHarness(t, out, "ref.c", `#include "local_bounds.generated.h"
void Replace(concept_readonly_span_int* v,concept_readonly_span_int other){*v=other;}
int main(void){int32_t a[4]={1,2,3,4};return concept_value_reference_run((concept_readonly_span_int){a,4},(concept_readonly_span_int){a,1})==1?0:1;}`)
	}
}

func TestValueParameterSnapshotBoundsAndEligibility(t *testing.T) {
	env := &semanticEnv{structs: map[string]StructDecl{}, enums: map[string]EnumDecl{}}
	scalar := Type{Name: "double", Kind: TypeBuiltin}
	env.structs["Small"] = StructDecl{Name: "Small", Ref: true, Fields: []Field{{Name: "a", Type: scalar}, {Name: "b", Type: scalar}}}
	for _, name := range []string{"Large", "RefField", "Array", "Immovable", "Atomic"} {
		env.structs[name] = StructDecl{Name: name}
	}
	large := env.structs["Large"]
	for i := 0; i < 17; i++ {
		large.Fields = append(large.Fields, Field{Type: scalar})
	}
	env.structs["Large"] = large
	borrow := scalar
	borrow.Ownership = "ref"
	env.structs["RefField"] = StructDecl{Name: "RefField", Fields: []Field{{Type: borrow}}}
	env.structs["Array"] = StructDecl{Name: "Array", Fields: []Field{{Type: Type{ArrayElem: &scalar, ArrayLength: 2}}}}
	env.structs["Immovable"] = StructDecl{Name: "Immovable", Immovable: true, Fields: []Field{{Type: scalar}}}
	env.structs["Atomic"] = StructDecl{Name: "Atomic", Attributes: []Attribute{{Name: "volatile"}}, Fields: []Field{{Type: scalar}}}
	if n := evt1SnapshotValueLeaves(env, Type{Name: "Small", Kind: TypeStruct}, map[string]bool{}); n != 2 {
		t.Fatalf("small ref-struct value geometry: %d", n)
	}
	for _, name := range []string{"Large", "RefField", "Array", "Immovable", "Atomic"} {
		if n := evt1SnapshotValueLeaves(env, Type{Name: name, Kind: TypeStruct}, map[string]bool{}); n != 0 {
			t.Fatalf("unsupported representation %s selected: %d", name, n)
		}
	}
	for _, ownership := range []string{"ref", "borrow", "owned"} {
		if evt1SnapshotValueLeaves(env, Type{Name: "Small", Kind: TypeStruct, Ownership: ownership}, map[string]bool{}) != 0 {
			t.Fatal("ownership boundary copied", ownership)
		}
	}
}

func TestValueParameterSnapshotGeneratedNameCollision(t *testing.T) {
	const source = `module ValueNames;profile Core;
template<usize N> int Cell(Span<int> cv_parameter_value_01){
int cv_parameter_value_02=7;return cv_parameter_value_01[0]+cv_parameter_value_02;}
int Run(Span<int> values){return Cell<1>(values);}`
	for _, verify := range []bool{false, true} {
		out, _ := localBoundsOutputs(t, source, verify)
		if !strings.Contains(moduleOutput(t, out, ".generated.c"), "cv_parameter_value_01 = cv_parameter_value_03;") {
			t.Fatal("entry parameter name collides with a source binding")
		}
		runFoundationNativeHarness(t, out, "name.c", `#include "local_bounds.generated.h"
int main(void){int32_t a=5;return concept_value_names_run((concept_span_int){&a,1})==12?0:1;}`)
	}
}

func TestValueParameterSnapshotImportedAndDeterministic(t *testing.T) {
	library := `module ValueLibrary;profile Core;
ref struct View { ReadOnlySpan<int> values;int bias; }
template<usize N> int Read(View v){return v.values[0]+v.bias;}`
	artifact := buildSemanticArtifact(t, "ValueLibrary.concept", library, nil)
	source := `module ValueUse;profile Core;import ValueLibrary;
int Run(ReadOnlySpan<int> values){View v=View{values,3};return Read<1>(v);}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		m, err := ParseWithSemanticModules("value_use.concept", source, map[string][]byte{"ValueLibrary": artifact})
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
		if !strings.Contains(moduleOutput(t, out, ".generated.c"), "source value snapshot") {
			t.Fatal("imported closed record was not materialized")
		}
		runFoundationNativeHarness(t, out, "import.c", `#include "value_use.generated.h"
int main(void){int32_t x=4;return concept_value_use_run((concept_readonly_span_int){&x,1})==7?0:1;}`)
	}
}
