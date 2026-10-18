package concept

import (
	"fmt"
	"strings"
	"testing"
)

func TestEVT2f5CallArgumentCapacity(t *testing.T) {
	source := `profile Core;
float Caller()
{
    return Target(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0);
}
float Target(float a, float b, float c, float d, float e, float f, float g, float h, float i)
{
    return a;
}
`
	checked, err := Parse("call_capacity.concept", source)
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
#include <string.h>
#define F(n) concept_standard__backend__amd64_##n
/* ARTIFACT */
static unsigned char bytes[4096];
int main(void)
{
    concept_readonly_span_byte input = {artifact, sizeof artifact};
    concept_span_byte output = {bytes, sizeof bytes};
    concept_native_image image, before;
    memset(bytes, 0x5a, sizeof bytes);
    memset(&image, 0x3c, sizeof image);
    memcpy(&before, &image, sizeof image);
    if (F(validate_scalar_machine_bridge)(input).tag) return 1;
    concept_result_int_native_error result = F(emit_module)(input, output, &image);
    if (!result.tag || strcmp(F(describe_native_error)(result.payload.error.error), "AMD64_NATIVE_CAPACITY_EXCEEDED")) return 2;
    if (memcmp(&image, &before, sizeof image)) return 3;
    for (size_t i = 0; i < sizeof bytes; i++) if (bytes[i] != 0x5a) return 4;
    return 0;
}
`
	harness = strings.Replace(harness, "/* ARTIFACT */", data.String(), 1)
	for _, verify := range []bool{false, true} {
		t.Run(fmt.Sprint(verify), func(t *testing.T) {
			runFoundationNativeHarnessOutput(t, backendTestOutputs(t, verify), "call_capacity_host.c", harness, "-pedantic-errors", "-O2")
		})
	}
}
