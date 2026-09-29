package concept

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// libraries/Vulkan is ordinary Concept over a flat C boundary. Its package
// tests link the GPU-free test device, so they need g++ and ar, not a GPU.
func TestVulkanLibraryNormalAndVerify(t *testing.T) {
	for _, tool := range []string{"g++", "gcc", "ar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " unavailable")
		}
	}
	project, err := LoadNativeProject(filepath.Join("..", "..", "libraries", "Vulkan"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeABI(project); err != nil {
		t.Fatal(err)
	}
	artifacts, identity, err := BuildNativeCompanionArtifacts(project)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTestsWithNativeABI(filepath.Join(project.Root, "tests"), artifacts, identity)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	build, err := RunNativeBuild(project, plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateNativeBuildOutputs(project, build); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(project.Root, ".native-build", "libvulkan_test_device.a")
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{
			Verify: verify, NativeLinker: "g++", NativeLinkInputs: []string{archive},
			ResultsDir: filepath.Join(t.TempDir(), "results"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if run.Failed != 0 || run.Passed != len(manifest.Tests) || run.Passed < 4 {
			t.Fatalf("verify=%v: %+v", verify, run.Results)
		}
	}
}
