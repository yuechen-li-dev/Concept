package concept

import (
	"os"
	"strings"
	"testing"
)

func TestR9aGenericExplicitRoundingArtifactAndNative(t *testing.T) {
	bytes, err := os.ReadFile("../../language/evt1/generic-library-closure/valid/generic_round.concept")
	if err != nil {
		t.Fatal(err)
	}
	body := buildSemanticArtifact(t, "round.concept", string(bytes), nil)
	consumer := `module Closure.RoundUse; profile Core; import Closure.Round;
Result<int, NumericCastError> Use(float value) { return Truncate<float>(value); }
Result<uint8, NumericCastError> Byte(double value) { return RoundTarget<uint8>(value); }
`
	module, err := ParseWithSemanticModules("round_use.concept", consumer, map[string][]byte{"Closure.Round": body})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "round_use.generated.c")
	runFoundationNativeHarness(t, outputs, "round_harness.c", `#include "round_use.generated.h"
int main(void) {
  concept_result_int_numeric_cast_error a = concept_closure__round_use_use(3.9f);
  concept_result_uint8_numeric_cast_error b = concept_closure__round_use_byte(3.5);
  concept_result_uint8_numeric_cast_error c = concept_closure__round_use_byte(300.0);
  if (a.tag != 0 || a.payload.ok.value != 3) return 1;
  if (b.tag != 0 || b.payload.ok.value != 4) return 2;
  if (c.tag != 1) return 3;
  return 0;
}`)
	_, err = Parse("bad_as.concept", "profile Core; int Bad(float value) { return value as int; }")
	if err == nil || !strings.Contains(err.Error(), "FLOAT_TO_INT_ROUNDING_REQUIRED") {
		t.Fatalf("rounding policy weakened: %v", err)
	}
}
