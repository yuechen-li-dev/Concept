package concept

import (
	"reflect"
	"testing"
)

func TestStaticIterationItemClosesImportedTemplateValue(t *testing.T) {
	const library = `module StaticCells; profile Core;
template <usize K> int Cell(int value) { return value + (K as int); }
template <usize N> int Row(int value) {
 int result = 0;
 comptime for (k in 0u .. N) { result += Cell<k>(value); }
 return result;
}
template <usize N> int LiteralRow(int value) {
 int result = 0;
 comptime for (k in 0u .. 2u) { result += Cell<k>(value); }
 return result;
}`
	artifact := buildSemanticArtifact(t, "StaticCells.concept", library, nil)
	const source = `module StaticUse; profile Core; import StaticCells;
comptime usize Global = 5u;
int Main(int value) { comptime usize Count = 3u; return Row<Count>(value) + LiteralRow<Count>(value) + Cell<Global>(value); }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("static_cells.concept", source, map[string][]byte{"StaticCells": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "static_cells.generated.c")
		runFoundationNativeHarness(t, outputs, "cells.c", `#include "static_cells.generated.h"
int main(void) { return concept_static_use_main(10)==69 ? 0 : 1; }`)
		again, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil || !reflect.DeepEqual(outputs, again) {
			t.Fatalf("scoped value closure is not deterministic: %v", err)
		}
	}
}

func TestScopedTemplateValueRejectsRuntimeAndInvalidInteger(t *testing.T) {
	for _, body := range []string{
		`int Main(usize K) { return Cell<K>(1); }`,
		`int Main() { comptime int K = -1; return Cell<K>(1); }`,
		`int Main() { comptime bool K = true; return Cell<K>(1); }`,
		`int Main() { comptime uint64 K = 18446744073709551615; return Cell<K>(1); }`,
		`comptime usize K = 2u; int Main(usize K) { return Cell<K>(1); }`,
	} {
		source := "profile Core; template <usize N> int Cell(int value) { return value; } " + body
		if _, err := Parse("invalid_cells.concept", source); diagnosticCode(err) != "GENERIC_NON_TYPE_ARGUMENT_INVALID" {
			t.Fatalf("invalid scoped value must reject: %v\n%s", err, source)
		}
	}
}
