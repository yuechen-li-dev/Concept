package concept

import (
	"strings"
	"testing"
)

func TestLocalIntegerIntervalsSelectDefinedArithmetic(t *testing.T) {
	const source = `module LocalBounds;profile Core;
int Scaled(int i){if(i<0 or i>=8){return 0;}return 4*i+1;}
int Unknown(int i){return 4*i+1;}
int Negative(int i){if(i<(-7) or i>0){return 0;}return 4*i-1;}`
	for _, verify := range []bool{false, true} {
		out, mir := localBoundsOutputs(t, source, verify)
		proved := 0
		for _, fn := range mir.Functions {
			for _, op := range fn.Operations {
				if op.IntegerRange != nil {
					proved++
				}
			}
		}
		if proved != 4 {
			t.Fatalf("arithmetic proofs=%d", proved)
		}
		c := moduleOutput(t, out, ".generated.c")
		start := strings.Index(c, "int32_t concept_local_bounds_scaled(")
		end := strings.Index(c[start:], "\n}\n") + start
		has := strings.Contains(c[start:end], "_i32_mul(")
		if has != verify {
			t.Fatalf("verify=%t checked lowering=%t", verify, has)
		}
		assertR8cStrictC11(t, out, "local_bounds.generated.c")
		runFoundationNativeHarness(t, out, "integer.c", `#include "local_bounds.generated.h"
int main(void){for(int i=-10;i<12;i++){int expected=i>=0&&i<8?4*i+1:0;if(concept_local_bounds_scaled(i)!=expected)return 1;expected=i>=-7&&i<=0?4*i-1:0;if(concept_local_bounds_negative(i)!=expected)return 2;}return 0;}`)
	}
}

func TestLocalIntegerPotentialOverflowRetainsCheckedOperation(t *testing.T) {
	const source = `module LocalBounds;profile Core;
int Multiply(int i){return i*2;}
int Add(int i){if(i<0){return 0;}return i+2147483647;}
int Subtract(int i){return 0-i;}`
	out, mir := localBoundsOutputs(t, source, false)
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.IntegerRange != nil {
				t.Fatalf("unknown overflow removed: %+v", op)
			}
		}
	}
	c := moduleOutput(t, out, ".generated.c")
	for _, op := range []string{"_i32_mul(", "_i32_add(", "_i32_sub("} {
		if !strings.Contains(c, op) {
			t.Fatal("missing overflow helper", op)
		}
	}
}
