package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSMLibCompileOptions(t *testing.T) {
	options, err := parseCompileOptions("emit-c", []string{"--verify", "--out-dir", "out", "--source-modules", "--module-root", "lib", "--module-root", "more"})
	if err != nil || !options.verify || !options.sourceModules || options.outputDir != "out" || len(options.moduleRoots) != 2 {
		t.Fatalf("%+v %v", options, err)
	}
	for _, args := range [][]string{{"--out-dir"}, {"--module-root"}, {"--out-dir", "one", "--out-dir", "two"}, {"--unknown"}} {
		if _, err := parseCompileOptions("emit-c", args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := parseCompileOptions("check", []string{"--out-dir", "out"}); err == nil {
		t.Fatal("check accepted emission option")
	}
}
func TestSMLibCLICompleteArtifactsAndExplicitImports(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "concept.exe")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(want string, args ...string) []byte {
		t.Helper()
		out, err := exec.Command(binary, args...).CombinedOutput()
		if want == "" {
			if err != nil {
				t.Fatalf("%v: %v %s", args, err, out)
			}
		} else if err == nil || !strings.Contains(string(out), want) {
			t.Fatalf("%v: want %s, got %v %s", args, want, err, out)
		}
		return out
	}
	deps := filepath.Join(root, "deps")
	dep := filepath.Join(deps, "Demo", "Math.concept")
	write(dep, "module Demo.Math; profile Core; int Value(){return 7;}")
	consumer := filepath.Join(root, "app", "Runner.concept")
	write(consumer, "module CLI.Consumer; profile Core; import Demo.Math; int Main(){return Value();}")
	run("MODULE_IMPORT_MISSING", "check", consumer, "--module-root", deps)
	run("", "check", consumer, "--source-modules", "--module-root", deps, "--module-root", filepath.Join(deps, "."))
	outdir := filepath.Join(root, "out")
	run("", "emit-c", consumer, "--out-dir", outdir, "--source-modules", "--module-root", deps)
	names := []string{"runner.generated.c", "runner.generated.h", "runner.mir.json", "runner.manifest.json", "runner.map.json"}
	first := map[string][]byte{}
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(outdir, name))
		if err != nil {
			t.Fatal(err)
		}
		first[name] = body
	}
	run("", "emit-c", consumer, "--out-dir", outdir, "--source-modules", "--module-root", deps)
	for name, want := range first {
		got, err := os.ReadFile(filepath.Join(outdir, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("nondeterministic artifact %s: %v", name, err)
		}
	}
	stdout := run("", "emit-c", consumer, "--source-modules", "--module-root", deps)
	if !bytes.Equal(stdout, first["runner.generated.c"]) {
		t.Fatal("legacy stdout emission changed")
	}
	if clang, err := exec.LookPath("clang"); err == nil {
		output, err := exec.Command(clang, "-std=c11", "-pedantic-errors", "-c", filepath.Join(outdir, "runner.generated.c"), "-o", filepath.Join(root, "runner.obj")).CombinedOutput()
		if err != nil {
			t.Fatalf("strict C11: %v %s", err, output)
		}
	}
	run("", "emit-c", consumer, "--verify", "--out-dir", filepath.Join(root, "verify"), "--source-modules", "--module-root", deps)
	// The same module graph is usable by the explicit artifact path.
	artifact := run("", "build-module", dep)
	write(filepath.Join(deps, "Demo", "Math.concept-module.json"), string(artifact))
	run("", "check", consumer, "--module-root", deps)
	test := filepath.Join(root, "app", "imports.concept_test")
	write(test, `module CLI.Tests; profile Core; import Demo.Math; [[fact]] void Imported(){Assert.True(Value()==7,"explicit dependency root");}`)
	for _, mode := range [][]string{{}, {"--verify"}} {
		args := append([]string{"test", test, "--module-root", deps}, mode...)
		run("", args...)
	}
	conflict := filepath.Join(root, "conflict")
	write(filepath.Join(conflict, "Demo", "Math.concept"), "module Demo.Math; profile Core; int Value(){return 9;}")
	run("MODULE_IDENTITY_DUPLICATE", "check", consumer, "--source-modules", "--module-root", deps, "--module-root", conflict)
	write(filepath.Join(deps, "Demo", "Math.concept"), "module Other.Math; profile Core; int Value(){return 7;}")
	run("MODULE_IDENTITY_MISMATCH", "check", consumer, "--source-modules", "--module-root", deps)
}
