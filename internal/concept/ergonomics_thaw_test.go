package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestErgonomicsCarrierContextAndArrayC11(t *testing.T) {
	source := runtimeTestSource(t, "contextual_carriers.concept_test")
	module, err := Parse("contextual_carriers.concept_test", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "contextual_carriers.generated.c")
	runFoundationNativeHarness(t, outputs, "carrier_harness.c", `#include "contextual_carriers.generated.h"
int main(void) {
  concept_ergonomics__carriers_carrier_arguments_use_parameter_context();
  concept_ergonomics__carriers_fixed_array_spellings_share_storage();
  concept_ergonomics__carriers_carrier_argument_moves_and_drops_exactly_once();
  concept_ergonomics__carriers_option_value_field_projection_keeps_reference_precedence();
  return 0;
}`)
}

func TestErgonomicsExportModuleAndFormatRoundTrip(t *testing.T) {
	manifest, err := DiscoverTests("../../tests/runtime/c_exports.concept_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tests) != 1 || manifest.Tests[0].Function != "ExportBodiesUseOrdinaryOwnership" || manifest.Tests[0].Kind != "fact" {
		t.Fatalf("C exports must not become test entry points: %+v", manifest.Tests)
	}
	source := runtimeTestSource(t, "c_exports.concept_test")
	formatted, err := FormatSource("c_exports.concept_test", source, DefaultFormatOptions())
	if err != nil {
		t.Fatal(err)
	}
	again, err := FormatSource("c_exports.concept_test", formatted, DefaultFormatOptions())
	if err != nil || again != formatted {
		t.Fatalf("export formatting is not idempotent: %v", err)
	}
	artifact, err := CompileSemanticModule("c_exports.concept_test", formatted, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer := `module Ergonomics.Consumer; profile Core; import Ergonomics.Exports; int Answer() { return ConceptAnswer(); }`
	m, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Ergonomics.Exports": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(m, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["consumer.generated.c"], []byte("ConceptAnswer(")) {
		t.Fatal("semantic transport lost the C export symbol")
	}
}

func TestErgonomicsOpaqueCExports(t *testing.T) {
	source := runtimeTestSource(t, "c_exports.concept_test")
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			policy := ConservativeCompilationPolicy()
			policy.Verify = verify
			var first Outputs
			for sample := 0; sample < 3; sample++ {
				m, err := Parse("c_exports.concept_test", source)
				if err != nil {
					t.Fatal(err)
				}
				o, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
				if err != nil {
					t.Fatal(err)
				}
				if first == nil {
					first = o
				} else {
					for name, expected := range first {
						if !bytes.Equal(expected, o[name]) {
							t.Fatalf("fresh sample %d changed %s", sample, name)
						}
					}
				}
			}
			h := string(first["c_exports.exports.h"])
			if !strings.Contains(h, "typedef struct concept_counter concept_counter;") || !strings.Contains(h, "ConceptCounterRead(const concept_counter*)") {
				t.Fatalf("missing opaque const interface:\n%s", h)
			}
			for _, private := range []string{"generated.h", "value;", "payload", "MakeCounter", "DisposeCounter"} {
				if strings.Contains(h, private) {
					t.Fatalf("private declaration %q leaked into public header", private)
				}
			}
			compiler, err := exec.LookPath("gcc")
			if err != nil {
				compiler, err = exec.LookPath("clang")
			}
			if err != nil {
				t.Fatal("C compiler required for export ABI evidence")
			}
			dir := t.TempDir()
			if err := Write(dir, first); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				"storage.c": `#include "c_exports.generated.h"
#include "c_exports.exports.h"
#include <stdlib.h>
concept_counter* MakeCounter(void) {
  concept_counter* p = malloc(sizeof(*p));
  if (p) *p = concept_ergonomics__exports_make_counter(7);
  return p;
}
void DisposeCounter(concept_counter* p) { concept_ergonomics__exports_dispose_counter(*p); free(p); }
`,
				"consumer.c": `#include "c_exports.exports.h"
extern concept_counter* MakeCounter(void);
extern void DisposeCounter(concept_counter* counter);
int main(void) {
  concept_counter* p = MakeCounter();
  if (!p) return 10;
  int result = ConceptCounterAdd(p, 5) == 12 && ConceptCounterRead(p) == 12 && ConceptAnswer() == 42;
  DisposeCounter(p);
  return result ? 0 : 1;
}
`,
				"consumer.cpp": `#include "c_exports.exports.h"
extern "C" concept_counter* MakeCounter(void);
extern "C" void DisposeCounter(concept_counter* counter);
int main() {
  auto* p = MakeCounter();
  if (!p) return 10;
  const concept_counter* read = p;
  const bool result = ConceptCounterAdd(p, 5) == 12 && ConceptCounterRead(read) == 12 && ConceptAnswer() == 42;
  DisposeCounter(p);
  return result ? 0 : 1;
}
`,
			}
			for name, contents := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
					t.Fatal(err)
				}
			}
			var objects []string
			for _, source := range []string{"c_exports.generated.c", "storage.c"} {
				object := filepath.Join(dir, source+".o")
				if out, err := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-I", dir, "-c", filepath.Join(dir, source), "-o", object).CombinedOutput(); err != nil {
					t.Fatalf("C11 object: %v\n%s", err, out)
				}
				objects = append(objects, object)
			}
			for _, language := range []string{"c", "cpp"} {
				driver, standard := compiler, "-std=c11"
				if language == "cpp" {
					driver, err = exec.LookPath("g++")
					if err != nil {
						driver, err = exec.LookPath("clang++")
					}
					if err != nil {
						t.Fatal("C++23 compiler required for public header evidence")
					}
					standard = "-std=c++23"
				}
				exe := filepath.Join(dir, "consumer_"+language+".exe")
				args := []string{standard, "-pedantic-errors", "-I", dir, filepath.Join(dir, "consumer."+language)}
				args = append(args, objects...)
				args = append(args, "-o", exe)
				if out, err := nativeCommand(t, driver, withHostLinkArgs(args...)...).CombinedOutput(); err != nil {
					t.Fatalf("%s public consumer: %v\n%s", language, err, out)
				}
				if out, err := nativeCommand(t, exe).CombinedOutput(); err != nil {
					t.Fatalf("%s ABI execution: %v\n%s", language, err, out)
				}
			}
		})
	}
}

func TestErgonomicsCExportDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, code string }{
		{`export "C" int Missing();`, "EXPORT_BODY_REQUIRED"},
		{`export "Rust" int Wrong() { return 1; }`, "EXPORT_ABI_INVALID"},
		{`extern "C" int Imported() { return 1; }`, "EXTERN_BODY_INVALID"},
		{`class Owner { public: int value; }; export "C" int Copy(owned Owner owner) { return owner.value; }`, "EXPORT_C_ABI_TYPE_INVALID"},
		{`class Owner { public: int value; }; export "C" ref Owner Escape(ref Owner owner) { return ref owner; }`, "EXPORT_C_ABI_TYPE_INVALID"},
		{`class Owner { public: int value; }; export "C" int Take(Owner owner) { return owner.value; }`, "EXPORT_C_ABI_TYPE_INVALID"},
		{`export "C" int Take(Option<int> value) { return 1; }`, "EXPORT_C_ABI_TYPE_INVALID"},
		{`export "C" int Same(int value) { return value; } int Same(bool value) { return 1; }`, "EXPORT_SYMBOL_INVALID"},
		{`export "C" int new() { return 1; }`, "EXPORT_SYMBOL_INVALID"},
		{`[[__concept_c_export]] int Hidden() { return 1; }`, "EXPORT_ATTRIBUTE_RESERVED"},
	} {
		_, err := Parse("export_negative.concept", "profile Core; "+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("%s: want %s, got %v", tc.source, tc.code, err)
		}
	}
}

func TestErgonomicsCExportDoesNotClaimCathedralABI(t *testing.T) {
	source := runtimeTestSource(t, "c_exports.concept_test")
	m, err := Parse("c_exports.concept_test", source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GenerateLIR(m)
	if err == nil || !strings.Contains(err.Error(), "EVT2_C_EXPORT_UNSUPPORTED") {
		t.Fatalf("unqualified native export must reject explicitly: %v", err)
	}
}

func TestErgonomicsCarrierDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"ambiguous", `int F(Result<int, int> x) { return 1; } int F(Result<int, bool> x) { return 2; } int Main() { return F(Result::Ok(1)); }`, "CV4182"},
		{"wrong error", `enum Error { Failed, } int F(Result<void, Error> x) { return 1; } int Main() { return F(Result::Error(1)); }`, "CV4107"},
		{"typed conditional error arm", `enum Error { Failed, } Result<void, bool> Other() { return Result::Error(true); } int F(Result<void, Error> x) { return 1; } int Main() { return F(if (true) Result::Ok() else Other()); }`, "CV4116"},
		{"parenthesized borrow", `struct Store { int value; }; void Reclaim(owned Store store) { } int Main() { owned Store store = Store{7}; ref const int held = (ref const store.value); Reclaim(move store); return held; }`, "STORAGE_MOVE_WITH_LIVE_REFERENCE"},
		{"payload overflow", `int F(Option<uint8> x) { return 1; } int Main() { return F(Option::Some(256)); }`, "CV4644"},
		{"missing move", `class Owner { public: int value; }; int F(Option<owned Owner> x) { return 1; } int Main() { owned Owner x = Owner{1}; return F(Option::Some(x)); }`, "CV4548"},
		{"no context", `int Main() { auto value = Result::Ok(1); return 0; }`, "CV4540"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("carrier_negative.concept", "profile Core; "+tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
}

func TestErgonomicsArraySpellingsEmitIdenticalC(t *testing.T) {
	const source = `profile Core; template <typename T> struct Box { T value; }; int Count(Box<int>[3] values) { return Len(values); }`
	var first Outputs
	for _, spelling := range []string{"Box<int>[3]", "Box<int><array>[3]"} {
		s := strings.ReplaceAll(source, "Box<int>[3]", spelling)
		m, err := Parse("array_spelling.concept", s)
		if err != nil {
			t.Fatal(err)
		}
		o, err := Generate(m, []byte(s))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = o
		} else {
			for _, name := range []string{"array_spelling.generated.c", "array_spelling.generated.h"} {
				if !bytes.Equal(first[name], o[name]) {
					t.Fatalf("array spelling changed C representation in %s", name)
				}
			}
		}
	}
}
