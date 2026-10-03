package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerReusesBuildsAndIsolatesProcesses(t *testing.T) {
	compiler, linker := "gcc", "g++"
	if _, err := exec.LookPath(compiler); err != nil {
		compiler, linker = "clang", "clang++"
	}
	for _, tool := range []string{compiler, linker} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	root := t.TempDir()
	for _, name := range []string{"reuse.concept_test", "rows.json", "counter.c"} {
		body, err := os.ReadFile(testingFixturePath("build-reuse", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	object := filepath.Join(root, "counter.o")
	if out, err := nativeCommand(t, compiler, "-std=c11", "-c", filepath.Join(root, "counter.c"), "-o", object).CombinedOutput(); err != nil {
		t.Fatalf("native counter: %v\n%s", err, out)
	}
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	session := newTestBuildSession()
	defer session.close()
	options := TestRunOptions{ResultsDir: filepath.Join(root, "results"), NativeLinker: linker, NativeLinkInputs: []string{object}}
	var firstBuild *testModuleBuild
	for _, verify := range []bool{false, true} {
		options.Verify = verify
		run, err := runTests(manifest, options, session)
		if err != nil || run.Passed != 5 || run.Fulfilled != 1 || run.Failed != 0 {
			t.Fatalf("isolated facts/rows/prophecy verify=%v: %v %+v", verify, err, run)
		}
		identity := run.Results[0].BuildIdentity
		if identity == "" {
			t.Fatal("build identity missing")
		}
		for _, result := range run.Results {
			if result.BuildIdentity != identity {
				t.Fatal("same module's declarations did not share build identity")
			}
		}
		prepared := session.prepare(manifest.Tests[0], options)
		if !verify {
			firstBuild = prepared
		} else if prepared == firstBuild {
			t.Fatal("Normal and Verify reused a compilation session")
		}
		if len(prepared.objects) != 1 || len(prepared.executables) != 5 {
			t.Fatalf("expected one module object and five distinct harnesses: %+v", prepared)
		}
		before, err := os.Stat(prepared.objects[0])
		if err != nil {
			t.Fatal(err)
		}
		repeated, err := runTests(manifest, options, session)
		if err != nil || repeated.Passed != 5 || repeated.Fulfilled != 1 || repeated.Failed != 0 {
			t.Fatalf("repeated isolated execution: %v %+v", err, repeated)
		}
		after, err := os.Stat(prepared.objects[0])
		if err != nil || !os.SameFile(before, after) || before.ModTime() != after.ModTime() {
			t.Fatalf("repeated execution rebuilt the module: %v", err)
		}
	}
	if len(session.builds) != 2 {
		t.Fatalf("mode separation: %d builds", len(session.builds))
	}
	path := filepath.Join(root, "reuse.concept_test")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(source), "Assert.Equal(Count(), 1,", "Assert.Equal(Count(), 2,", 1)
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	rediscovered, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runTests(rediscovered, options, session)
	if err != nil || run.Failed != 1 || run.Passed != 4 || run.Fulfilled != 1 || len(session.builds) != 3 {
		t.Fatalf("rediscovery reused stale source: %v %+v", err, run)
	}
	if run.Results[0].Failure == nil || run.Results[0].Failure.Kind != "assertion:Equal" {
		t.Fatalf("assertion failure classification lost: %+v", run.Results[0])
	}
	// Changed native input bytes must also invalidate the same discovery snapshot.
	if err := os.WriteFile(filepath.Join(root, "counter.c"), []byte("int Count(void) { return 2; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, compiler, "-std=c11", "-c", filepath.Join(root, "counter.c"), "-o", object).CombinedOutput(); err != nil {
		t.Fatalf("changed native counter: %v\n%s", err, out)
	}
	run, err = runTests(manifest, options, session)
	if err != nil || run.Failed != 5 || run.Fulfilled != 1 || len(session.builds) != 4 {
		t.Fatalf("native input invalidation: %v %+v", err, run)
	}
	var dirs []string
	for _, build := range session.builds {
		dirs = append(dirs, build.dir)
	}
	session.close()
	for _, dir := range dirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("build directory survived cleanup: %s: %v", dir, err)
		}
	}
}

func TestNativeFixtureObjectReuseTracksInputs(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	outputs := Outputs{
		"reuse.generated.c": []byte("#include \"reuse.generated.h\"\nint answer(void) { return VALUE; }\n"),
		"reuse.generated.h": []byte("#define VALUE 40\nint answer(void);\n"),
	}
	first := nativeFixtureObjects(t, outputs, compiler)
	again := nativeFixtureObjects(t, outputs, compiler)
	if first[0] != again[0] {
		t.Fatal("identical immutable fixture compiled twice")
	}
	outputs["reuse.generated.h"] = []byte("#define VALUE 42\nint answer(void);\n")
	changed := nativeFixtureObjects(t, outputs, compiler)
	if changed[0] == first[0] {
		t.Fatal("changed header reused stale object")
	}
	flags := nativeFixtureObjects(t, outputs, compiler, "-O2")
	if flags[0] == changed[0] {
		t.Fatal("different compiler flags reused an object")
	}
	dir := t.TempDir()
	harness := filepath.Join(dir, "main.c")
	if err := os.WriteFile(harness, []byte("int answer(void); int main(void) { return answer() == 42 ? 0 : 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "check.exe")
	if out, err := nativeCommand(t, compiler, harness, changed[0], "-o", executable).CombinedOutput(); err != nil {
		t.Fatalf("header invalidation link: %v\n%s", err, out)
	}
	if out, err := nativeCommand(t, executable).CombinedOutput(); err != nil {
		t.Fatalf("changed header did not reach native execution: %v\n%s", err, out)
	}
}
