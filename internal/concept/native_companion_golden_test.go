package concept

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeCompanionGoldenNormalAndVerify(t *testing.T) {
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	if _, err := exec.LookPath("llvm-ar"); err != nil {
		t.Skip("llvm-ar unavailable")
	}
	project, err := LoadNativeProject(filepath.Join("..", "..", "tests", "goldens", "companion"))
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
	if len(manifest.Tests) != 2 {
		t.Fatalf("expected two companion facts, got %d", len(manifest.Tests))
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
	archive := filepath.Join(project.Root, ".native-build", "libgolden_counter.a")
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{
			Verify: verify, NativeLinker: "clang++", NativeLinkInputs: []string{archive},
			ResultsDir: filepath.Join(t.TempDir(), "results"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if run.Passed != 2 || run.Failed != 0 {
			t.Fatalf("verify=%v: %+v", verify, run)
		}
		if verify && len(run.Results[0].Verifications) == 0 {
			t.Fatal("foreign creation observer did not run")
		}
	}
}
