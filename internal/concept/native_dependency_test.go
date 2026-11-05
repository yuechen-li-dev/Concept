package concept

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeDependencyInputsHashTheirActualFiles(t *testing.T) {
	root, libraries := t.TempDir(), t.TempDir()
	dependency := filepath.Join(libraries, "NativeFixture")
	if err := os.MkdirAll(filepath.Join(dependency, "native"), 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dependency, "manifest.concept"), "profile Core;")
	write(filepath.Join(dependency, "native", "fixture.c"), "int fixture(void) { return 7; }\n")
	write(filepath.Join(dependency, "native", "fixture.h"), "int fixture(void);\n")
	write(filepath.Join(root, "manifest.concept"), "profile Core;")
	t.Setenv("CONCEPT_LIBRARY_ROOT", libraries)
	project := NativeProject{Name: "Consumer", Root: root, Toolchain: "Clang", Targets: []NativeTarget{{
		Name: "fixture", Language: "C", Standard: "C11", Kind: "StaticLibrary", Output: ".native-build/fixture.a",
		Sources: []string{"@NativeFixture/native/fixture.c"}, Includes: []string{"@NativeFixture/native"},
	}}}
	first, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	built, err := RunNativeBuild(project, first)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateNativeBuildOutputs(project, built); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(dependency, "native", "fixture.h"), "int fixture(void); /* changed */\n")
	second, err := NativeBuildPlan(project)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildInputHash == second.BuildInputHash {
		t.Fatal("external header bytes were not hashed")
	}
	if err := ValidateNativeBuildOutputs(project, built); err == nil {
		t.Fatal("stale dependency was accepted")
	}
	for _, path := range []string{"@NativeFixture/../escape", "@NativeFixture/native/../../escape", "@Missing/native/file.c"} {
		if _, err := nativeInputPath(root, path); err == nil {
			t.Fatalf("invalid input accepted: %s", path)
		}
	}
	if err := nativeValidatePath(root, "@NativeFixture/output.a"); err == nil {
		t.Fatal("dependency output was accepted")
	}
}
