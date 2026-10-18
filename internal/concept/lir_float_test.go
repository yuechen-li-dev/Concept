package concept

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func evt2fScalarFixture(t *testing.T) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt2f", "valid", "scalars.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	return module, source
}

func TestEVT2fScalarLIRAndDeterminism(t *testing.T) {
	module, _ := evt2fScalarFixture(t)
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	text := lir.String()
	for _, want := range []string{"fadd float32", "fsub float32", "fmul float32", "fdiv float32", "fadd float64", "fsub float64", "fmul float64", "fdiv float64", "fneg float32", "fneg float64", "const float32 0x3dcccccd", "const float64 0x3ff0000000000001", "call float64 @Mixed", "args=[i32 float64 i32 float32]", "args=[float64 i32 float32 i32 float64 float32 i32 float64]", "load_slot float32", "store_slot float64", "index_address ptr<float32>", "stride=4", "index_address ptr<float64>", "stride=8"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in LIR:\n%s", want, text)
		}
	}
	for _, op := range []string{"eq", "ne", "lt", "le", "gt", "ge"} {
		if strings.Count(text, op+" bool") < 2 {
			t.Fatalf("comparison %s missing for both representations", op)
		}
	}
	for _, f := range lir.Functions {
		if strings.HasPrefix(f.Name, "Interpret") && (len(f.Blocks) != 1 || len(f.Blocks[0].Instructions) != 0 || f.Blocks[0].Term.Value != f.Params[0].ID) {
			t.Fatalf("same-representation interpretation emitted runtime operations: %+v", f)
		}
	}
	for i := 0; i < 100; i++ {
		next, err := GenerateLIR(module)
		if err != nil || next.String() != text {
			t.Fatalf("floating LIR changed on run %d: %v", i+1, err)
		}
	}
	if machine, err := LowerLirToAmd64Machine(lir); err != nil || machine.Scalar == nil || len(machine.Functions) != 0 {
		t.Fatalf("float did not enter the virtual CMIRAMD4 contract: %+v, %v", machine, err)
	}
}

func TestEVT2fFloatBitsAndMalformedLIR(t *testing.T) {
	for _, tc := range []struct {
		typ   LIRType
		value float64
		want  string
	}{
		{"float32", 0, "0x00000000"}, {"float32", math.Copysign(0, -1), "0x80000000"},
		{"float32", math.Inf(1), "0x7f800000"}, {"float32", math.Inf(-1), "0xff800000"},
		{"float64", 0, "0x0000000000000000"}, {"float64", math.Copysign(0, -1), "0x8000000000000000"},
		{"float64", math.Inf(1), "0x7ff0000000000000"}, {"float64", math.Inf(-1), "0xfff0000000000000"},
		{"float64", math.Float64frombits(0x7ff8000000001234), "0x7ff8000000001234"},
	} {
		if got := lirFloatLiteral(tc.typ, tc.value); got != tc.want {
			t.Fatalf("%s got %s want %s", tc.typ, got, tc.want)
		}
	}
	for _, typ := range []LIRType{"float32", "float64"} {
		bits := "0x7fc01234"
		if typ == "float64" {
			bits = "0x7ff8000000001234"
		}
		makeLIR := func() LIRModule {
			return LIRModule{Functions: []LIRFunction{{Identity: "F", Name: "F", Result: typ, Params: []LIRValue{{ID: 0, Type: typ}}, Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{
				{Op: "const", Result: 1, Type: typ, Literal: bits, Slot: -1},
				{Op: "fadd", Result: 2, Type: typ, Args: []int{0, 1}, Slot: -1},
			}, Term: LIRTerminator{Op: "return", Value: 2}}}}}}
		}
		if err := VerifyLIR(makeLIR()); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name       string
			mutate     func(*LIRModule)
			diagnostic string
		}{
			{"decimal", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[0].Literal = "1.0" }, "LIR_BAD_FLOAT_CONST"},
			{"wrong bit width", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[0].Literal = bits + "00" }, "LIR_BAD_FLOAT_CONST"},
			{"uppercase", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[0].Literal = strings.ToUpper(bits) }, "LIR_BAD_FLOAT_CONST"},
			{"integer opcode", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[1].Op = "checked_add" }, "LIR_BAD_ARITHMETIC"},
			{"integer argument", func(m *LIRModule) { m.Functions[0].Params[0].Type = "i32" }, "LIR_BAD_FLOAT_ARITHMETIC"},
			{"mixed width", func(m *LIRModule) {
				if typ == "float32" {
					m.Functions[0].Params[0].Type = "float64"
				} else {
					m.Functions[0].Params[0].Type = "float32"
				}
			}, "LIR_BAD_FLOAT_ARITHMETIC"},
			{"unary arity", func(m *LIRModule) { m.Functions[0].Blocks[0].Instructions[1].Op = "fneg" }, "LIR_BAD_FLOAT_NEGATION"},
		} {
			t.Run(string(typ)+"/"+tc.name, func(t *testing.T) {
				m := makeLIR()
				tc.mutate(&m)
				if err := VerifyLIR(m); err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
					t.Fatalf("wanted %s: %v", tc.diagnostic, err)
				}
			})
		}
		if _, _, err := Win64Return(typ); err == nil {
			t.Fatal("floating GPR return admitted")
		}
		if _, err := Win64Argument(0, typ); err == nil {
			t.Fatal("floating GPR argument admitted")
		}
	}
}

func TestEVT2fScalarBoundaries(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"profile Core; half F(half x) { return x; }", "EVT2_UNSUPPORTED_FLOAT_REPRESENTATION binary16"},
		{"profile Core; float F(float<cm> a, float<m> b) { return a / b; }", "EVT2_UNSUPPORTED_QUANTITY_SCALE 1/100"},
		{"profile Core; double F(float x) { return x as double; }", "EVT2_UNSUPPORTED_EXPR *concept.CastExpr"},
	} {
		m, err := Parse("boundary.concept", tc.source)
		if err != nil {
			t.Fatal(err)
		}
		_, err = GenerateLIR(m)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("wanted %s: %v", tc.want, err)
		}
	}
	for _, source := range []string{
		"profile Core; bool F(float x, float y) { return x < y; }",
		"profile Core; int F() { float unused = 0.0; return 1; }",
		"profile Core; float F(float[4] x, int i) { return x[i]; }",
	} {
		m, err := Parse("float-admission.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		machine, err := GenerateMachineIR(m)
		if err != nil || machine.Scalar == nil {
			t.Fatalf("floating input did not select scalar class transport: %v", err)
		}
	}
}

// This is the independent source C oracle for the future native differential
// lane. It qualifies source behavior; it does not execute the floating LIR or
// claim that Cathedral has emitted SSE bytes.
func TestEVT2fScalarSourceStrictC11(t *testing.T) {
	module, source := evt2fScalarFixture(t)
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		policy := ConservativeCompilationPolicy()
		if verify {
			mode = "Verify"
			policy = VerifyCompilationPolicy()
		}
		t.Run(mode, func(t *testing.T) {
			outputs, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			runFoundationNativeHarnessOutput(t, outputs, "float_source_host.c", evt2fScalarSourceHarness, "-pedantic-errors", "-O2")
		})
	}
}

const evt2fScalarSourceHarness = `#include "scalars.generated.h"
#include <stdint.h>
#include <string.h>
#define F(n) concept_evt2f__scalars_##n
#define REQUIRE(c) do { if (!(c)) return __LINE__; } while (0)
static float f32(uint32_t bits) { float value; memcpy(&value,&bits,4); return value; }
static double f64(uint64_t bits) { double value; memcpy(&value,&bits,8); return value; }
static uint32_t bits32(float value) { uint32_t bits; memcpy(&bits,&value,4); return bits; }
static uint64_t bits64(double value) { uint64_t bits; memcpy(&bits,&value,8); return bits; }
int main(void) {
    REQUIRE(sizeof(float)==4 && sizeof(double)==8);
    REQUIRE(bits32(F(constant32)())==UINT32_C(0x3dcccccd));
    REQUIRE(bits64(F(constant64)())==UINT64_C(0x3ff0000000000001));
    REQUIRE(bits32(F(arithmetic32)(2.5f,2.0f))==UINT32_C(0x40200000));
    REQUIRE(bits64(F(arithmetic64)(2.5,2.0))==UINT64_C(0x4004000000000000));
    REQUIRE(bits32(F(negative_zero32)())==UINT32_C(0x80000000));
    REQUIRE(bits64(F(negative_zero64)())==UINT64_C(0x8000000000000000));
    REQUIRE(bits32(F(negative32)(f32(UINT32_C(0x80000000))))==0);
    REQUIRE(bits64(F(negative64)(f64(UINT64_C(0x8000000000000000))))==0);
    const uint64_t input64[]={UINT64_C(0xc004000000000000),0,UINT64_C(0x8000000000000000),UINT64_C(0x4004000000000000),UINT64_C(0x7ff0000000000000),UINT64_C(0xfff0000000000000),UINT64_C(0x7ff8000000001234)};
    const uint32_t input32[]={UINT32_C(0xc0200000),0,UINT32_C(0x80000000),UINT32_C(0x40200000),UINT32_C(0x7f800000),UINT32_C(0xff800000),UINT32_C(0x7fc01234)};
    for (int i=0;i<7;i++) for (int j=0;j<7;j++) {
        double a=f64(input64[i]),b=f64(input64[j]);
        float c=f32(input32[i]),d=f32(input32[j]);
        REQUIRE(F(equal64)(a,b)==(a==b)); REQUIRE(F(not_equal64)(a,b)==(a!=b));
        REQUIRE(F(less64)(a,b)==(a<b)); REQUIRE(F(less_equal64)(a,b)==(a<=b));
        REQUIRE(F(greater64)(a,b)==(a>b)); REQUIRE(F(greater_equal64)(a,b)==(a>=b));
        REQUIRE(F(equal32)(c,d)==(c==d)); REQUIRE(F(not_equal32)(c,d)==(c!=d));
        REQUIRE(F(less32)(c,d)==(c<d)); REQUIRE(F(less_equal32)(c,d)==(c<=d));
        REQUIRE(F(greater32)(c,d)==(c>d)); REQUIRE(F(greater_equal32)(c,d)==(c>=d));
    }
    float nan32=f32(UINT32_C(0x7fc01234)); double nan64=f64(UINT64_C(0x7ff8000000001234));
    REQUIRE(!F(equal32)(nan32,0) && F(not_equal32)(nan32,0));
    REQUIRE(!F(less32)(nan32,0) && !F(less_equal32)(nan32,0) && !F(greater32)(nan32,0) && !F(greater_equal32)(nan32,0));
    REQUIRE(!F(equal64)(nan64,0) && F(not_equal64)(nan64,0));
    REQUIRE(!F(less64)(nan64,0) && !F(less_equal64)(nan64,0) && !F(greater64)(nan64,0) && !F(greater_equal64)(nan64,0));
    REQUIRE(F(caller)(3,2.0,1,4.0f)==5.0);
    REQUIRE(F(caller8)(2.0,3,4.0f,1,5.0,6.0f,7,8.0)==7.0);
    REQUIRE(F(pressure)(2.0f,3.0f)==5.0f && F(distance)(2.0,3.0)==5.0);
    REQUIRE(bits32(F(interpret_pressure)(2.5f))==UINT32_C(0x40200000));
    REQUIRE(bits64(F(interpret_distance)(2.5))==UINT64_C(0x4004000000000000));
    return 0;
}
`
