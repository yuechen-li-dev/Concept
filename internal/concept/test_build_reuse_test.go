package concept

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

var nativeFixtureBuilds = struct {
	sync.Mutex
	dirs    []string
	objects map[string]string
}{objects: map[string]string{}}

func TestMain(m *testing.M) {
	code := m.Run()
	for _, dir := range nativeFixtureBuilds.dirs {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

// Only immutable emitted bytes are shared. Callers retain fresh harnesses,
// processes and oracles. Compiler path, flags and supporting files are keyed;
// changing any of them forces another compilation. Determinism tests that
// measure generation still call the real generator independently.
func nativeFixtureObjects(t *testing.T, outputs Outputs, compiler string, flags ...string) []string {
	t.Helper()
	nativeFixtureBuilds.Lock()
	defer nativeFixtureBuilds.Unlock()
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	var objects []string
	for _, source := range names {
		if !strings.HasSuffix(source, ".generated.c") {
			continue
		}
		h := sha256.New()
		fmt.Fprintf(h, "%q %q %q", compiler, flags, source)
		for _, name := range names {
			if strings.HasSuffix(name, ".c") && name != source {
				continue
			}
			if name != source && !strings.HasSuffix(name, ".h") && !strings.HasSuffix(name, ".inc") {
				continue
			}
			fmt.Fprintf(h, "%d:%s%d:", len(name), name, len(outputs[name]))
			h.Write(outputs[name])
		}
		key := fmt.Sprintf("%x", h.Sum(nil))
		object := nativeFixtureBuilds.objects[key]
		if object == "" {
			dir, err := os.MkdirTemp("", "concept-go-fixture-")
			if err != nil {
				t.Fatal(err)
			}
			nativeFixtureBuilds.dirs = append(nativeFixtureBuilds.dirs, dir)
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			object = filepath.Join(dir, "fixture.o")
			args := append([]string{"-std=c11", "-Wall", "-Wextra"}, flags...)
			args = append(args, "-I", dir, "-c", filepath.Join(dir, source), "-o", object)
			if out, err := nativeCommand(t, compiler, args...).CombinedOutput(); err != nil {
				t.Fatalf("native fixture compile failed: %v\n%s", err, out)
			}
			nativeFixtureBuilds.objects[key] = object
		}
		objects = append(objects, object)
	}
	if len(objects) == 0 {
		t.Fatal("generated C output missing")
	}
	return objects
}

var backendFixtureBuilds [2]struct {
	sync.Once
	outputs Outputs
	err     error
}

func backendTestOutputs(t *testing.T, verify bool) Outputs {
	t.Helper()
	index := 0
	if verify {
		index = 1
	}
	build := &backendFixtureBuilds[index]
	build.Do(func() {
		path := "../../libraries/Standard/Backend/AMD64.concept"
		source, err := os.ReadFile(path)
		if err != nil {
			build.err = err
			return
		}
		module, err := ParseWithBuiltSemanticModuleRoots(path, string(source), []string{"../../libraries"})
		if err != nil {
			build.err = err
			return
		}
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		build.outputs, build.err = GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
	})
	if build.err != nil {
		t.Fatal(build.err)
	}
	outputs := Outputs{}
	for name, body := range build.outputs {
		outputs[name] = bytes.Clone(body)
	}
	return outputs
}

// Large integration cold builds are expensive and add little ordering
// coverage after the second independent build. Small compiler/artifact gates
// continue to use determinismRuns (100). The full cold-build burn-in is opt-in.
func integrationDeterminismRuns() int {
	if os.Getenv("CONCEPT_TEST_STRESS") == "1" {
		return determinismRuns()
	}
	return 2
}

func packageOutputBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		files[filepath.ToSlash(rel)] = string(body)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return files
}
