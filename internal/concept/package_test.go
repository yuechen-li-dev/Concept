package concept

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestR7aManifestsAreOrdinaryImmutableConceptData(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	manifests, err := DiscoverPackageManifests(root)
	if err != nil {
		t.Fatal(err)
	}
	standard, ok := manifests["Standard"]
	if !ok || standard.Author != "CODEX" || standard.Kind != "Library" || standard.Version != (PackageVersion{0, 0, 1}) || len(standard.Dependencies) != 0 {
		t.Fatalf("unexpected Standard manifest: %#v", standard)
	}
	dragonGod, ok := manifests["DragonGod"]
	if !ok || dragonGod.Author != "CODEX" || dragonGod.Kind != "Library" || dragonGod.Version != (PackageVersion{0, 0, 1}) || len(dragonGod.Dependencies) != 1 || dragonGod.Dependencies[0] != "Standard" {
		t.Fatalf("unexpected DragonGod manifest: %#v", dragonGod)
	}
	order, err := PackageBuildOrder(manifests, "DragonGod")
	if err != nil || len(order) != 2 || order[0] != "Standard" || order[1] != "DragonGod" {
		t.Fatalf("unexpected package order %v: %v", order, err)
	}
}

func TestR7aPackageCycleDiagnosticIsBounded(t *testing.T) {
	manifests := map[string]PackageManifestValue{
		"A": {Name: "A", Dependencies: []string{"B"}},
		"B": {Name: "B", Dependencies: []string{"A"}},
	}
	_, err := PackageBuildOrder(manifests, "A")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("PACKAGE_DEPENDENCY_CYCLE")) {
		t.Fatalf("expected package cycle diagnostic, got %v", err)
	}
}

func TestR7aStandardAndDragonGodPackagesBuildDeterministically(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	first, err := BuildPackage(root, output, "DragonGod")
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := MarshalPackageGraph(first)
	for run := 1; run < 100; run++ {
		again, err := BuildPackage(root, output, "DragonGod")
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		body, _ := MarshalPackageGraph(again)
		if !bytes.Equal(body, expected) {
			t.Fatalf("package graph changed on run %d", run)
		}
	}
	if len(first.Packages) != 2 || first.Packages[0].Name != "Standard" || first.Packages[1].Name != "DragonGod" {
		t.Fatalf("unexpected graph: %#v", first.Packages)
	}
	if first.Packages[0].Author != "CODEX" || first.Packages[1].Author != "CODEX" ||
		first.Packages[1].Version != (PackageVersion{0, 0, 1}) || first.Packages[1].Kind != "Library" ||
		len(first.Packages[1].Dependencies) != 1 || first.Packages[1].Dependencies[0] != "Standard" {
		t.Fatalf("package graph did not preserve manifest metadata: %#v", first.Packages)
	}
}

func TestR7aPromotedStandardMemoryArtifactIsEquivalent(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "Standard"); err != nil {
		t.Fatal(err)
	}
	expected := standardMemoryArtifacts(t)["Standard.Memory.Bump"]
	actual, err := os.ReadFile(filepath.Join(output, "Standard", "modules", "Standard", "Memory", "Bump.concept-module.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatal("promoted Standard.Memory artifact is not semantically equivalent to the R6p build path")
	}
}

func TestR7aArtifactOnlyDragonGodConsumerRunsStrictC11(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module R7a.ArtifactConsumer;
profile Core;
import DragonGod.Memory.Boot;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;
int Main()
{
    FirmwareMemorySource firmware = MakeFirmwareMemorySource();
    BumpAllocator bump = MakeBumpAllocator(FirmwareRegion(ref firmware));
    return InspectBootInfo(ref bump, BootInfo{4, 38})!;
}
`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7a/ArtifactConsumer.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r7a_artifact_consumer_harness.c", "#include \"artifactconsumer.generated.h\"\nint main(void) { return concept_artifactconsumer_main() == 42 ? 0 : 1; }\n")
}

func TestR7aPackageErrorsRemainTyped(t *testing.T) {
	_, err := PackageBuildOrder(map[string]PackageManifestValue{}, "Missing")
	if err == nil {
		t.Fatal("expected missing package error")
	}
	var diagnostic Diagnostic
	if errors.As(err, &diagnostic) {
		t.Fatalf("package graph errors must not masquerade as source diagnostics: %v", err)
	}
}
