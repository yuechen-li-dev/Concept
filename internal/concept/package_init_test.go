package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitPackageBuildsThroughOrdinaryManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Starter")
	created, err := InitPackage(root, "Starter", "Compiler tests")
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Fatalf("created %v", created)
	}
	graph, err := BuildPackageDirectory(root, filepath.Join(root, ".concept", "packages"), CompilerLibraryRoot())
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Packages) != 1 || graph.Packages[0].Name != "Starter" || len(graph.Packages[0].Modules) != 1 {
		t.Fatalf("graph: %+v", graph)
	}
	before, err := os.ReadFile(created[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InitPackage(root, "Starter", "Another author"); err == nil || !strings.Contains(err.Error(), "PACKAGE_INIT_EXISTS") {
		t.Fatalf("repeat init: %v", err)
	}
	after, err := os.ReadFile(created[0])
	if err != nil || string(before) != string(after) {
		t.Fatal("initializer overwrote an existing manifest")
	}
	for _, invalid := range []string{"../Escape", "a.b", "with space", ""} {
		if _, err := InitPackage(root, invalid, "Compiler tests"); err == nil {
			t.Fatalf("accepted name %q", invalid)
		}
	}
}
