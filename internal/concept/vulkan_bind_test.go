package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reflectFixture(t *testing.T, name string) SPIRVInterface {
	t.Helper()
	module, err := os.ReadFile(filepath.Join("testdata", "vulkan-bind", name))
	if err != nil {
		t.Fatal(err)
	}
	reflected, err := ReflectSPIRV(module)
	if err != nil {
		t.Fatal(err)
	}
	return reflected
}

func TestReflectSPIRVReadsTheComputeInterface(t *testing.T) {
	scale := reflectFixture(t, "scale.spv")
	if scale.Entry != "main" || scale.LocalSize != [3]uint32{64, 1, 1} {
		t.Fatalf("entry/workgroup = %q %v", scale.Entry, scale.LocalSize)
	}
	want := []SPIRVBinding{
		{Binding: 0, Kind: "storage", ReadOnly: true, Element: "float", Name: "source"},
		{Binding: 1, Kind: "storage", ReadOnly: false, Element: "float", Name: "destination"},
	}
	if len(scale.Bindings) != len(want) {
		t.Fatalf("bindings = %+v", scale.Bindings)
	}
	for i := range want {
		if scale.Bindings[i] != want[i] {
			t.Fatalf("binding %d = %+v, want %+v", i, scale.Bindings[i], want[i])
		}
	}
	if scale.Push == nil || scale.Push.Size != 8 || len(scale.Push.Members) != 2 ||
		scale.Push.Members[0] != (SPIRVPushMember{Name: "count", Offset: 0, Type: "uint32", Count: 1, Bytes: 4}) ||
		scale.Push.Members[1] != (SPIRVPushMember{Name: "scale", Offset: 4, Type: "float", Count: 1, Bytes: 4}) {
		t.Fatalf("push = %+v", scale.Push)
	}
	double := reflectFixture(t, "double.spv")
	if double.Push != nil || len(double.Bindings) != 2 || !double.Bindings[0].ReadOnly || double.Bindings[1].ReadOnly || double.Bindings[0].Element != "int" {
		t.Fatalf("double = %+v", double)
	}
}

func TestVulkanBindFingerprintTracksTheInterfaceNotNames(t *testing.T) {
	scale := reflectFixture(t, "scale.spv")
	renamed := scale
	renamed.Bindings = append([]SPIRVBinding{}, scale.Bindings...)
	renamed.Bindings[0].Name = "input"
	if renamed.Fingerprint() != scale.Fingerprint() {
		t.Fatal("a rename changed the fingerprint")
	}
	writable := scale
	writable.Bindings = append([]SPIRVBinding{}, scale.Bindings...)
	writable.Bindings[0].ReadOnly = false
	if writable.Fingerprint() == scale.Fingerprint() {
		t.Fatal("an access change kept the fingerprint")
	}
	if reflectFixture(t, "double.spv").Fingerprint() == scale.Fingerprint() {
		t.Fatal("different kernels share a fingerprint")
	}
}

// The example kernel modules are exactly what the generator produces, so they
// cannot drift from it.
func TestVulkanBindExampleModulesAreCurrent(t *testing.T) {
	for _, kernel := range []struct{ spv, name, file string }{
		{"double.spv", "Double", "DoubleKernel.concept"},
		{"scale.spv", "Scale", "ScaleKernel.concept"},
	} {
		generated, err := GenerateVulkanBinding(reflectFixture(t, kernel.spv), VulkanBindOptions{
			Name: kernel.name, Module: kernel.name + "Kernel", KernelPath: "kernels/" + kernel.spv, Source: "kernels/" + kernel.spv,
		})
		if err != nil {
			t.Fatal(err)
		}
		existing, err := os.ReadFile(filepath.Join("..", "..", "examples", "vulkan", kernel.file))
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(string(existing), "\r\n", "\n") != generated {
			t.Fatalf("%s is not what concept vulkan-bind generates; regenerate it", kernel.file)
		}
	}
}

func TestVulkanBindGeneratesAccessFromTheShader(t *testing.T) {
	generated, err := GenerateVulkanBinding(reflectFixture(t, "scale.spv"), VulkanBindOptions{Name: "Scale", Module: "ScaleKernel", KernelPath: "scale.spv", Source: "scale.spv"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"ref const Buffer<float> source, ref Buffer<float> destination, ScalePush push",
		"Bind(0, ref const source, Access::Read), Bind(1, ref const destination, Access::Write)",
		"at(4) float scale;",
		"static_assert(LayoutSize<ScaleConstants>() == 8,",
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated module lacks %q:\n%s", want, generated)
		}
	}
}

func TestReflectSPIRVRejectsWhatItCannotBind(t *testing.T) {
	if _, err := ReflectSPIRV([]byte{1, 2, 3, 4}); err == nil {
		t.Fatal("a truncated module was accepted")
	}
	scale := reflectFixture(t, "scale.spv")
	scale.Bindings = nil
	if _, err := GenerateVulkanBinding(scale, VulkanBindOptions{Name: "Scale", Module: "ScaleKernel"}); err == nil {
		t.Fatal("a kernel without bindings was accepted")
	}
}

func TestVulkanBindIdentifiers(t *testing.T) {
	for in, want := range map[string]string{"source": "source", "g_Output": "gOutput", "Weights_F16": "weightsF16", "3d": "v3d", "": ""} {
		if got := vulkanBindIdentifier(in); got != want {
			t.Fatalf("identifier(%q) = %q, want %q", in, got, want)
		}
	}
	if VulkanBindName("kernels/fused_qkv.spv") != "FusedQkv" {
		t.Fatal(VulkanBindName("kernels/fused_qkv.spv"))
	}
}

// A declared type spelled like a builtin (Double) must not collide with the
// builtin (double) in composite C names such as Result<T, E>.
func TestDeclaredTypesDoNotCollideWithBuiltinsInC(t *testing.T) {
	source := `module Collide;
profile Core;
enum Failure { Bad, }
class Double { public: int value; };
Result<Double, Failure> MakeDouble() { Double made = Double{21}; return Result::Ok(move made); }
Result<double, Failure> MakeReal() { return Result::Ok(0.5); }
int Main()
{
    Double made = MakeDouble()!;
    double real = MakeReal()!;
    if (real != 0.5)
    {
        return 0;
    }
    return made.value * 2;
}
`
	module, err := Parse("collide.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "collide_harness.c", "#include \"collide.generated.h\"\nint main(void) { return concept_collide_main() == 42 ? 0 : 1; }\n")
}

func TestStaleKernelBindingsAreReported(t *testing.T) {
	dir := t.TempDir()
	spirv, err := os.ReadFile(filepath.Join("testdata", "vulkan-bind", "scale.spv"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateVulkanBinding(reflectFixture(t, "scale.spv"), VulkanBindOptions{Name: "Scale", Module: "ScaleKernel", KernelPath: "scale.spv", Source: "scale.spv"})
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(dir, "ScaleKernel.concept")
	if err := os.WriteFile(module, []byte(generated), 0644); err != nil {
		t.Fatal(err)
	}
	if err := evt1CheckVulkanKernelBindings(dir); err != nil {
		t.Fatalf("a module without its SPIR-V was checked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scale.spv"), spirv, 0644); err != nil {
		t.Fatal(err)
	}
	if err := evt1CheckVulkanKernelBindings(dir); err != nil {
		t.Fatalf("a current module was reported: %v", err)
	}
	double, err := os.ReadFile(filepath.Join("testdata", "vulkan-bind", "double.spv"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scale.spv"), double, 0644); err != nil {
		t.Fatal(err)
	}
	if err := evt1CheckVulkanKernelBindings(dir); err == nil || !strings.Contains(err.Error(), "rerun: concept vulkan-bind") {
		t.Fatalf("a stale module was not reported: %v", err)
	}
}
