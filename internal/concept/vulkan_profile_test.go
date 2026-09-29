package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func vulkanProfileToolsOrSkip(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"gcc", "g++"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " unavailable")
		}
	}
}

// `profile Vulkan;` implies `import Vulkan;` and supplies the library with
// its measured native ABI evidence; no import line or manifest is written.
func TestVulkanProfileImpliesTheLibrary(t *testing.T) {
	vulkanProfileToolsOrSkip(t)
	path := filepath.Join("..", "..", "examples", "vulkan", "BufferLifetime.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "\nimport ") {
		t.Fatal("the example should not need an import line")
	}
	imports, err := SemanticModuleImports(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 1 || imports[0] != "Vulkan" {
		t.Fatalf("profile Vulkan did not imply the library: %v", imports)
	}
	module, err := ParseWithModuleRootsForProfile(filepath.ToSlash(path), string(source), []string{filepath.Dir(path)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, source); err != nil {
		t.Fatal(err)
	}
}

// `concept test` links the Vulkan runtime for profile Vulkan tests.
func TestVulkanProfileExamplesNormalAndVerify(t *testing.T) {
	vulkanProfileToolsOrSkip(t)
	manifest, err := DiscoverTests(filepath.Join("..", "..", "examples", "vulkan"))
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: filepath.Join(t.TempDir(), "results")})
		if err != nil {
			t.Fatal(err)
		}
		if run.Failed != 0 || run.Passed == 0 {
			t.Fatalf("verify=%v: %+v", verify, run.Results)
		}
		if !strings.Contains(run.Results[0].TargetIdentity, "/vulkan-test-device") {
			t.Fatalf("runtime identity = %q", run.Results[0].TargetIdentity)
		}
	}
}

func TestVulkanProfileRejectsAnUnknownRuntime(t *testing.T) {
	vulkanProfileToolsOrSkip(t)
	t.Setenv("CONCEPT_VULKAN_RUNTIME", "gpu-please")
	manifest, err := DiscoverTests(filepath.Join("..", "..", "examples", "vulkan"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunTests(manifest, TestRunOptions{ResultsDir: filepath.Join(t.TempDir(), "results")})
	if err != nil {
		t.Fatal(err)
	}
	if run.Failed == 0 || run.Results[0].Failure == nil || run.Results[0].Failure.Kind != "vulkan-runtime" {
		t.Fatalf("unknown runtime was not reported: %+v", run.Results)
	}
}

// Regression: a `with` update of a bits value built in the same expression
// was constant-folded into the named field instead of `raw`, silently
// dropping the update.
func TestBitsWithOnATemporaryKeepsTheUpdate(t *testing.T) {
	source := `profile Core;
bits Flags : uint32 { low: 0; storage: 5; }
int Main()
{
    Flags folded = Flags{0} with { storage = true; low = true; };
    return folded.raw as int;
}
`
	module, err := Parse("bits_with_temporary.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := "#include \"bits_with_temporary.generated.h\"\nint main(void) { return concept_bits_with_temporary_main() == 33 ? 0 : 1; }\n"
	runFoundationNativeHarness(t, outputs, "bits_with_temporary_harness.c", harness)
}
