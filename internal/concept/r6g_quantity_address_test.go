package concept

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestR6gQuantityNormalizationArithmeticAndLayout(t *testing.T) {
	source := `profile Core;
static_assert(SizeOf<int>() == 4, "byte-sized layout");
static_assert(AlignOf<int>() == 4, "byte-sized alignment");

usize<byte> AddBytes(usize<byte> left, usize<byte> right) { return left + right; }
usize<byte> RequiredBytes(usize count, usize<byte> elementSize) { return count * elementSize; }
float HertzSeconds(float<Hz> frequency, float<s> period) { return frequency * period; }
float<K> Warm(float<K> temperature, float<K> delta) { return temperature + delta; }
float<m/s> Speed(float<m> distance, float<s> elapsed) { return distance / elapsed; }
usize<byte> BitsToBytes(usize<bit> bits) { return Convert<byte>(bits); }
usize Main() { usize count = 4; return RequiredBytes(count, SizeOf<int>()); }
`
	module, err := Parse("r6g_quantities.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["r6g_quantities.generated.c"])
	for _, want := range []string{"(count * elementSize)", "integral unit conversion is not exact", "concept_r6g_quantities_required_bytes"} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated C is missing %q:\n%s", want, generated)
		}
	}
	runFoundationNativeHarness(t, outputs, "r6g_quantities_harness.c", `#include "r6g_quantities.generated.h"
int main(void) { return concept_r6g_quantities_main() == 16u ? 0 : 1; }
`)
}

func TestR6gMemoryRegionAndSubregionAreOrdinaryImportedLibraryCode(t *testing.T) {
	library, err := os.ReadFile("../../libraries/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(library), nil)
	consumer := `module App;
profile Core;
import Standard.MemoryGeometry;
Result<MemoryRegion<SystemMemory>, RegionError> Carve(MemoryRegion<SystemMemory> parent)
{
    usize<byte> offset = 12;
    usize<byte> length = 16;
    return Subregion(parent, offset, length);
}`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Standard.MemoryGeometry": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ToLower(moduleOutput(t, outputs, ".generated.c"))
	for _, want := range []string{"concept_app_subregion", "parent.start + offset", "common_alignment"} {
		if !strings.Contains(body, want) {
			t.Fatalf("ordinary region library lowering omitted %q:\n%s", want, body)
		}
	}
}

func TestR6gQuantityMismatchIsDiagnosed(t *testing.T) {
	source := `profile Core;
usize<byte> Invalid(usize<byte> storage, usize<s> timeout) { return storage + timeout; }
`
	_, err := Parse("r6g_quantity_mismatch.concept", source)
	if err == nil {
		t.Fatal("expected quantity mismatch")
	}
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "QUANTITY_DIMENSION_MISMATCH" {
		t.Fatalf("expected QUANTITY_DIMENSION_MISMATCH, got %v", err)
	}
}

func TestR6gUsizeArithmeticSurface(t *testing.T) {
	source := `profile Core;
usize Arithmetic(usize left, usize right)
{
    usize sum = left + right;
    usize difference = sum - right;
    usize product = difference * right;
    usize quotient = product / right;
    usize remainder = quotient % right;
    usize bits = ((remainder & left) | right) ^ left;
    return (bits << 1) >> 1;
}`
	module, err := Parse("r6g_usize_arithmetic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["r6g_usize_arithmetic.generated.c"])
	for _, operator := range []string{" + ", " - ", " * ", " / ", " % ", " & ", " | ", " ^ ", " << ", " >> "} {
		if !strings.Contains(generated, operator) {
			t.Fatalf("generated C omitted usize operator %q:\n%s", operator, generated)
		}
	}
}

func TestR6gSignedModuloDoesNotLeakCRemainderSemantics(t *testing.T) {
	source := `profile Core;
int Modulo(int left, int right) { return left % right; }`
	_, err := Parse("r6g_signed_modulo.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "SIGNED_EUCLIDEAN_MODULO_DEFERRED" {
		t.Fatalf("expected SIGNED_EUCLIDEAN_MODULO_DEFERRED, got %v", err)
	}
}

func TestR6gAddressAffineGeometryAndRawBind(t *testing.T) {
	source := `profile Core;
struct SystemMemory {}
struct DeviceMemory {}

isize<byte> Distance(Address<SystemMemory> left, Address<SystemMemory> right) { return right - left; }
Address<SystemMemory> Offset(Address<SystemMemory> start, usize<byte> offset) { return start + offset; }

int Main()
{
    int<array>[4] backing = [0, 0, 0, 0];
    Address<SystemMemory> start = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = 16;
    Storage<int> cell = bind<int>(start, extent);
    ref int value = Initialize(cell, 7);
    int observed = value;
    Destroy(cell);
    return observed;
}
`
	module, err := Parse("r6g_address.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["r6g_address.generated.c"])
	for _, want := range []string{"uintptr_t start", "bind<T> extent is smaller", "bind<T> address does not satisfy", "(*cell = 7)"} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated C is missing %q:\n%s", want, generated)
		}
	}
	runFoundationNativeHarness(t, outputs, "r6g_address_harness.c", `#include "r6g_address.generated.h"
int main(void) { return concept_r6g_address_main() == 7u ? 0 : 1; }
`)
}

func TestR6gRawBindRejectsReconstructedAddress(t *testing.T) {
	source := `profile Core; struct SystemMemory {}
void Bad(usize raw, usize<byte> extent)
{
    Address<SystemMemory> address = AddressFromBits<SystemMemory>(raw);
    Storage<int> cell = bind<int>(address, extent);
}`
	_, err := Parse("r6g_raw_bind_unknown_provenance.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "RAW_BIND_PROVENANCE_UNKNOWN" {
		t.Fatalf("expected RAW_BIND_PROVENANCE_UNKNOWN, got %v", err)
	}
}

func TestR6gStorageLifetimeRejectsDoubleInitialize(t *testing.T) {
	source := `profile Core; struct SystemMemory {}
void Bad()
{
    int<array>[1] backing = [0];
    Address<SystemMemory> address = AddressOf<SystemMemory>(ref backing);
    usize<byte> extent = 4;
    Storage<int> cell = bind<int>(address, extent);
    ref int first = Initialize(cell, 1);
    ref int second = Initialize(cell, 2);
}`
	_, err := Parse("r6g_storage_double_initialize.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "STORAGE_DOUBLE_INITIALIZE" {
		t.Fatalf("expected STORAGE_DOUBLE_INITIALIZE, got %v", err)
	}
}

func TestR6gFunctionTemplatesAcceptOrderedMixedParameters(t *testing.T) {
	source := `profile Core;
template <typename T, typename U>
T First(T first, U second) { return first; }

template <typename T, usize N>
T Identity(T value) { return value; }

template <typename T, typename U, usize N>
T FirstThree(T first, U second) { return first; }

template <typename T>
usize<byte> RequiredBytesFor(usize count) { return count * SizeOf<T>(); }

int Main()
{
    uint other = 2;
    int a = First<int, uint>(1, other);
    int b = Identity<int, 4>(a);
    int c = FirstThree<int, uint, 8>(b, other);
    usize count = 3;
    usize<byte> bytes = RequiredBytesFor<int>(count);
    return c;
}`
	module, err := Parse("r6g_multi_template.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["r6g_multi_template.generated.c"])
	for _, want := range []string{"concept_template_first__int__uint", "concept_template_identity__int__4", "concept_template_first_three__int__uint__8", "(count * 4)"} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated C is missing %q:\n%s", want, generated)
		}
	}
	first := outputs["r6g_multi_template.generated.c"]
	for run := 1; run < 100; run++ {
		reparsed, parseErr := Parse("r6g_multi_template.concept", source)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		repeated, generateErr := Generate(reparsed, []byte(source))
		if generateErr != nil {
			t.Fatal(generateErr)
		}
		if !bytes.Equal(first, repeated["r6g_multi_template.generated.c"]) {
			t.Fatalf("R6g generated C changed on run %d", run)
		}
	}
}

func TestR6gAddressAdditionAndSpaceMismatchReject(t *testing.T) {
	cases := []struct {
		name, body, code string
	}{
		{"address_addition", "Address<SystemMemory> Bad(Address<SystemMemory> a, Address<SystemMemory> b) { return a + b; }", "ADDRESS_AFFINE_INVALID"},
		{"space_mismatch", "bool Bad(Address<SystemMemory> a, Address<DeviceMemory> b) { return a == b; }", "ADDRESS_SPACE_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "profile Core; struct SystemMemory {} struct DeviceMemory {} " + tc.body
			_, err := Parse(tc.name+".concept", source)
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}
