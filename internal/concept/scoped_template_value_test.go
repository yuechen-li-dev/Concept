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

func TestClosedImportedTemplateRetainsModuleComptimeConstants(t *testing.T) {
	const library = `module StaticTable; profile Core;
comptime int<array>[4] Coefficients = [1,3,3,1];
template <usize N> int Read(int index) { return Coefficients[index] + (N as int); }
`
	artifact := buildSemanticArtifact(t, "StaticTable.concept", library, nil)
	const source = `module StaticTableUse; profile Core; import StaticTable;
int Main(int index) {return Read<2>(index);}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("table_use.concept", source, map[string][]byte{"StaticTable": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "table_use.generated.c")
		runFoundationNativeHarness(t, outputs, "table.c", `#include "table_use.generated.h"
int main(void) {return concept_static_table_use_main(1)==5 ? 0 : 1;}`)
	}
	if _, err := ParseWithSemanticModules("bad_table.concept", `profile Core; import StaticTable; int Main(){return Read<2>(4);}`, map[string][]byte{"StaticTable": artifact}); err != nil {
		// The index is a runtime function parameter, so admission is valid and
		// its per-access runtime guard remains. Constant direct OOB is tested below.
		t.Fatal(err)
	}
	if _, err := Parse("bad_literal.concept", `profile Core; comptime int<array>[2] T=[1,2]; template <usize N> int Bad(){return T[2];} int Main(){return Bad<1>();}`); diagnosticCode(err) != "CV4233" {
		t.Fatalf("closed generic constant array must still reject static OOB: %v", err)
	}
}
