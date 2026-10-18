package concept

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2f5NativeMixedCalls(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("actual mixed Win64 entrypoints require Windows AMD64")
	}
	source, err := os.ReadFile("../../language/evt2f/valid/native_calls.concept")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := Parse("mixed_calls.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(checked)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	data.WriteString("static const unsigned char artifact[] = {")
	for _, b := range artifact {
		fmt.Fprintf(&data, "%d,", b)
	}
	data.WriteString("};\n")
	for i, f := range machine.Scalar.Functions {
		fmt.Fprintf(&data, "#define ORD_%s %d\n", f.Name, i)
	}
	mutations := []struct {
		name   string
		change func(*ScalarMachineModule)
	}{
		{"argument count", func(m *ScalarMachineModule) {
			c := &scalarInstruction(scalarFunction(m, "CallA"), "call").Calls[0]
			c.Arguments = c.Arguments[:3]
		}},
		{"float width", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "CallA"), "call").Calls[0].Arguments[1].Type = "float32"
		}},
		{"integer XMM", func(m *ScalarMachineModule) { scalarFunction(m, "MixedA").VRegs[0].Class = "XMM" }},
		{"floating GPR", func(m *ScalarMachineModule) { scalarFunction(m, "MixedA").VRegs[1].Class = "GPR" }},
		{"missing result", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "CallA"), "call").Result = -1 }},
		{"void capture", func(m *ScalarMachineModule) { scalarInstruction(scalarFunction(m, "CallVoid"), "call").Result = 0 }},
		{"bool XMM", func(m *ScalarMachineModule) { scalarFunction(m, "ReturnBool").VRegs[0].Class = "XMM" }},
		{"varargs", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "CallA"), "call").Calls[0].Convention = "win64-varargs"
		}},
		{"missing direct target", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "CallA"), "call").Calls[0].Target = ""
		}},
		{"result width", func(m *ScalarMachineModule) {
			scalarInstruction(scalarFunction(m, "CallA"), "call").Calls[0].Result = "float32"
		}},
	}
	var rejects strings.Builder
	rejects.WriteString("static int reject_bad_calls(void) {\n")
	for i, mutation := range mutations {
		invalid := scalarClone(t, machine)
		mutation.change(invalid.Scalar)
		if err := VerifyMachineIR(invalid); err == nil {
			t.Fatalf("producer admitted %s", mutation.name)
		}
		encoded, err := encodeScalarMachineBridge(*invalid.Scalar)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&data, "static const unsigned char bad_call_%d[] = {", i)
		for _, b := range encoded {
			fmt.Fprintf(&data, "%d,", b)
		}
		data.WriteString("};\n")
		fmt.Fprintf(&rejects, `{ concept_readonly_span_byte invalid = {bad_call_%d, sizeof bad_call_%d};
concept_span_byte output = {again, sizeof again}; concept_native_image rejected, before;
memset(&rejected, 0x3c, sizeof rejected); memcpy(&before, &rejected, sizeof before); memset(again, 0x5a, sizeof again);
REQUIRE(F(validate_scalar_machine_bridge)(invalid).tag);
concept_result_int_native_error result = F(emit_module)(invalid, output, &rejected);
REQUIRE(result.tag && !strcmp(F(describe_native_error)(result.payload.error.error), "AMD64_NATIVE_INVALID_INPUT"));
REQUIRE(!memcmp(&rejected, &before, sizeof before));
for (size_t i = 0; i < sizeof again; i++) REQUIRE(again[i] == 0x5a);
}

`, i, i)
	}
	rejects.WriteString("puts(\"ten malformed mixed-call artifacts rejected transactionally\"); return 0; }\n")
	host, err := os.ReadFile("testdata/evt2f5_calls_host.c")
	if err != nil {
		t.Fatal(err)
	}
	harness := strings.Replace(string(host), "/* ARTIFACT */", data.String(), 1)
	harness = strings.Replace(harness, "/* INVALID_CALL_CHECKS */", rejects.String(), 1)
	var normal string
	for _, verify := range []bool{false, true} {
		name := "Normal"
		if verify {
			name = "Verify"
		}
		t.Run(name, func(t *testing.T) {
			output := runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "mixed_calls_host.c", harness, "-pedantic-errors", "-O2", "-fno-fast-math", "-ffp-contract=off")
			var observations []string
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "image ") || strings.HasPrefix(line, "symbol ") || strings.HasPrefix(line, "fixup ") || strings.HasPrefix(line, "plan ") {
					observations = append(observations, strings.TrimSpace(line))
				}
			}
			current := strings.Join(observations, "\n")
			if verify && current != normal {
				t.Fatal("Normal/Verify mixed image/layout/fixups differ")
			}
			normal = current
		})
	}
}

func TestEVT2f5ForwardMixedCall(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("actual mixed Win64 entrypoints require Windows AMD64")
	}
	source := `profile Core;
double Caller(int a, double b, int c, float d)
{
    return Target(a, b, c, d);
}
double Target(int a, double b, int c, float d)
{
    if (a > c) {
        if (d == 3.5) {
            return b;
        }
    }
    return -b;
}
double WidthCaller(float a, double b)
{
    return WidthTarget(b, a);
}
double WidthTarget(double b, float a)
{
    if (a == 3.5) {
        return b;
    }
    return -b;
}
`
	checked, err := Parse("forward_mixed.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := GenerateMachineIR(checked)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	data.WriteString("static const unsigned char artifact[] = {")
	for _, b := range artifact {
		fmt.Fprintf(&data, "%d,", b)
	}
	data.WriteString("};\n")
	harness := `#include "amd64.generated.h"
#include <windows.h>
#include <stdint.h>
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
#define REQUIRE(c) do { if (!(c)) return __LINE__; } while (0)
/* ARTIFACT */
static unsigned char bytes[4096];
static concept_native_image image;
int main(void)
{
    concept_readonly_span_byte input = {artifact, sizeof artifact};
    concept_span_byte output = {bytes, sizeof bytes};
    REQUIRE(!F(emit_module)(input, output, &image).tag && image.finalized);
    REQUIRE(image.symbolCount == 4 && image.fixupCount == 2);
    concept_call_fixup fixup = image.fixups.data[0];
    REQUIRE(fixup.resolved && fixup.displacement > 0 && bytes[fixup.callOffset] == 0xe8);
    void *memory = VirtualAlloc(NULL, (size_t)image.byteCount, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
    REQUIRE(memory);
    memcpy(memory, bytes, (size_t)image.byteCount);
    DWORD previous;
    REQUIRE(VirtualProtect(memory, (size_t)image.byteCount, PAGE_EXECUTE_READ, &previous));
    REQUIRE(FlushInstructionCache(GetCurrentProcess(), memory, (size_t)image.byteCount));
    typedef double (*Mixed)(int, double, int, float);
    Mixed caller = NULL;
    void *entry = (unsigned char *)memory + image.symbols.data[0].offset;
    memcpy(&caller, &entry, sizeof caller);
    REQUIRE(caller(29, 1.0000000000000002, 17, 3.5f) == 1.0000000000000002);
    REQUIRE(caller(17, 2.5, 29, 3.5f) == -2.5);
    REQUIRE(caller(29, 2.5, 17, 4.5f) == -2.5);
    typedef double (*Widths)(float, double);
    Widths widths = NULL;
    entry = (unsigned char *)memory + image.symbols.data[2].offset;
    memcpy(&widths, &entry, sizeof widths);
    REQUIRE(widths(3.5f, 1.0000000000000002) == 1.0000000000000002);
    REQUIRE(widths(4.5f, 2.5) == -2.5);
    REQUIRE(VirtualFree(memory, 0, MEM_RELEASE));
    return 0;
}
`
	harness = strings.Replace(harness, "/* ARTIFACT */", data.String(), 1)
	for _, verify := range []bool{false, true} {
		t.Run(fmt.Sprint(verify), func(t *testing.T) {
			runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "forward_mixed_host.c", harness, "-pedantic-errors", "-O2", "-fno-fast-math", "-ffp-contract=off")
		})
	}
}
