package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSMLibNumericalTypingAndZeroConstruction(t *testing.T) {
	source := `profile Core;
 struct Inner { double x; bool flag; }
 struct Sample { Inner inner; double[3] data; }
 double Check(bool choice) { double x=if(choice) -1.0 else 1.0; double y=1e100+1e99; Sample s=Sample{}; return x + y/1e100 + s.data[0]; }
 int Main() { return if(Check(true)>0.09) 0 else 1; }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("smlib_numeric.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "smlib_numeric.generated.c")
		runFoundationNativeHarness(t, outputs, "numeric_harness.c", `#include "smlib_numeric.generated.h"
int main(void) { return concept_smlib_numeric_main(); }`)
	}
	for _, source := range []string{
		`double F(float x) { return if(true) x else 1.0; }`,
		`double F(float x) { return x+1.0; }`,
		`float F(double x) { return if(true) x else 1.0; }`,
		`float F() { return if(true) 1e100 else 0.0; }`,
		`struct Owner { owned int x; } int Main(){Owner o=Owner{};return 0;}`,
		`ref struct Borrow { ref int x; } int Main(){Borrow b=Borrow{};return 0;}`,
		`struct HasDrop { int x; } void Drop(owned HasDrop self) {} int Main(){owned HasDrop b=HasDrop{};return 0;}`,
	} {
		if _, err := Parse("numeric_invalid.concept", "profile Core; "+source); err == nil {
			t.Fatal("invalid program accepted: " + source)
		}
	}
}
func TestSMLibValueIntrinsicShadowing(t *testing.T) {
	source := `profile Core;
 double Magnitude(double x) {return x+10.0;}
 int Len(int x) {return x+2;}
 int Rank(int x) {return x+3;}
 int Shape(int x) {return x+4;}
 int HardMax(int x) {return x+5;}
 int Main(){const auto Magnitude=callback(int x){return x+1;};return Len(1)+Rank(1)+Shape(1)+HardMax(1)+Magnitude(1);}
 double UserMagnitude(){return Magnitude(2.0);}`
	module, err := Parse("smlib_shadow.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "smlib_shadow.generated.c")
	runFoundationNativeHarness(t, outputs, "shadow_harness.c", `#include "smlib_shadow.generated.h"
int main(void){return concept_smlib_shadow_main()==20 && concept_smlib_shadow_user_magnitude()==12.0 ? 0:1;}`)
	path := filepath.Join("..", "..", "libraries", "Standard", "smlib_numerics.concept_test")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mathBody, err := os.ReadFile(filepath.Join("..", "..", "libraries", "Standard", "math.concept"))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileSemanticModule("Standard/Math.concept", string(mathBody), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWithSemanticModules(path, string(body), map[string][]byte{"Standard.Math": artifact}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mathBody), "extern \"C\" double sqrt(double") {
		t.Fatal("double libm signature missing")
	}
}

func TestSMLibNumericArgumentEvaluatesOwnershipOnce(t *testing.T) {
	const source = `profile Core;
 extern "C" void Observed(int id);
 struct Resource { int id; }
 void Drop(owned Resource r) { Observed(r.id); }
 bool Consume(owned Resource r) { return r.id==1; }
 double Accept(double x) {return x;}
 int Main(){owned Resource r=Resource{1};double x=Accept(if(Consume(move r)) 1.0 else 2.0);return if(x==1.0) 0 else 1;}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("smlib_numeric_owner.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "smlib_numeric_owner.generated.c")
		runFoundationNativeHarness(t, outputs, "numeric_owner_harness.c", `#include "smlib_numeric_owner.generated.h"
static int drops=0;void Observed(int id){drops+=id;}
int main(void){return concept_smlib_numeric_owner_main()==0 && drops==1 ? 0:1;}`)
	}
}
