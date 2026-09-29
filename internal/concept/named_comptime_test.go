package concept

import (
	"strings"
	"testing"
)

func TestNamedComptimeGenericArtifactAndContext(t *testing.T) {
	const producer = `module Comptime.Constants; profile Core;
comptime int Capacity = 4;
comptime uint8 Mask = 0x80;
comptime uint8 Shift = 1;
comptime uint Derived = 1 << 7;
comptime uint64 Wide = 0xffffffffffffffff;
template <typename T, usize N>
usize Chosen(T value) { return N; }
`
	artifact, err := CompileSemanticModule("R8g/Constants.concept", producer, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module Comptime.ConstantConsumer; profile Core; import Comptime.Constants;
uint32 Apply(uint32 flags) { return flags & Mask; }
uint32 Shifted(uint32 flags) { return flags << Shift; }
uint32 Expected(uint32 value) { return value; }
uint32 Pass() { return Expected(Mask); }
uint32 DerivedValue() { return Derived; }
uint64 WideValue() { return Wide; }
uint8[Capacity] Bytes() { return [0 ... Capacity]; }
usize Named() { return Chosen<int, Capacity>(0); }
usize Literal() { return Chosen<int, 4>(0); }
`
	module, err := ParseWithSemanticModules("R8g/ConstantConsumer.concept", consumer, map[string][]byte{"Comptime.Constants": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "constantconsumer.generated.c")
	runFoundationNativeHarness(t, outputs, "constantconsumer_harness.c", `#include "constantconsumer.generated.h"
int main(void) {
  if (concept_comptime__constant_consumer_apply(0xffu) != 0x80u) return 1;
  if (concept_comptime__constant_consumer_shifted(2u) != 4u) return 5;
  if (concept_comptime__constant_consumer_pass() != 0x80u) return 2;
  if (concept_comptime__constant_consumer_named() != 4) return 3;
  if (concept_comptime__constant_consumer_literal() != 4) return 4;
  if (concept_comptime__constant_consumer_derived_value() != 128u) return 6;
  if (concept_comptime__constant_consumer_wide_value() != UINT64_MAX) return 7;
  return 0;
}`)
	c := string(outputs["constantconsumer.generated.c"])
	if strings.Contains(c, "concept_comptime__constants_capacity") || strings.Contains(c, "concept_comptime__constants_mask") {
		t.Fatal("named comptime constant emitted as runtime storage")
	}
}
