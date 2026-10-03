package concept

import (
	"strings"
	"testing"
)

func TestIntegerMatchContextAndExecution(t *testing.T) {
	const source = `module Matching.IntegerMatch; profile Core;
enum State { Ready, Stopped }
int Decode(uint8 code) { return match (code) { 0 => 10, 1 => 20, 2 => 30, _ => -1 }; }
int Signed(int code)
{
    match (code)
    {
        -1 => { return 4; }
        0 => { return 5; }
        _ => { return 6; }
    }
}
uint64 Large(uint64 code) { return match (code) { 0xffffffffffffffff => 7, _ => 0 }; }
int ExistingEnum(State state) { return match (state) { State::Ready => 1, State::Stopped => 2 }; }
`
	module, err := Parse("R8g/IntegerMatch.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "integermatch.generated.c")
	runFoundationNativeHarness(t, outputs, "integermatch_harness.c", `#include "integermatch.generated.h"
#include <stdint.h>
int main(void) {
  if (concept_matching__integer_match_decode(0) != 10) return 1;
  if (concept_matching__integer_match_decode(2) != 30) return 2;
  if (concept_matching__integer_match_decode(255) != -1) return 3;
  if (concept_matching__integer_match_signed(-1) != 4) return 4;
  if (concept_matching__integer_match_signed(8) != 6) return 5;
  if (concept_matching__integer_match_large(UINT64_MAX) != 7) return 6;
  if (concept_matching__integer_match_large(3) != 0) return 7;
  return 0;
}`)
	mir := string(outputs["integermatch.mir.json"])
	for _, needle := range []string{`"detail": "0"`, `"detail": "_"`, `"detail": "-1"`} {
		if !strings.Contains(mir, needle) {
			t.Fatalf("integer match MIR omitted %s", needle)
		}
	}
}

func TestIntegerMatchDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"nonexhaustive expression", `profile Core; int F(int x) { return match (x) { 0 => 1, 1 => 2 }; }`, "CV4115"},
		{"nonexhaustive statement", `profile Core; int F(int x) { match (x) { 0 => { return 1; } } return 2; }`, "CV4115"},
		{"duplicate numeric spelling", `profile Core; int F(int x) { return match (x) { 0 => 1, 0x0 => 2, _ => 3 }; }`, "CV4113"},
		{"unreachable after wildcard", `profile Core; int F(int x) { return match (x) { _ => 1, 0 => 2 }; }`, "CV4113"},
		{"literal out of range", `profile Core; int F(uint8 x) { return match (x) { 256 => 1, _ => 2 }; }`, "CV4644"},
		{"enum wildcard still explicit", `profile Core; enum State { Ready, Stopped } int F(State x) { return match (x) { State::Ready => 1, _ => 2 }; }`, "CV4109"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("r8g_integer_invalid.concept", tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}
