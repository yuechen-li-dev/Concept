package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Build reuse is local to one run: no disk cache, mutable global modules, or
// sharing between Normal and Verify. Each invocation still starts a process.
type testBuildSession struct {
	builds map[testBuildKey]*testModuleBuild
}

type testBuildKey struct {
	snapshot *testModuleSnapshot
	verify   bool
	native   string
}

type testModuleBuild struct {
	dir, base, identity, compiler, target string
	phase, message                        string
	objects                               []string
	machineHelper                         string
	executables                           map[string]string
}

func newTestBuildSession() *testBuildSession {
	return &testBuildSession{builds: map[testBuildKey]*testModuleBuild{}}
}

func (s *testBuildSession) close() {
	for _, build := range s.builds {
		if build.dir != "" {
			os.RemoveAll(build.dir)
		}
	}
}

func (s *testBuildSession) prepare(test TestDeclaration, options TestRunOptions) *testModuleBuild {
	nativeHash := sha256.New()
	nativeHash.Write([]byte(options.NativeLinker))
	var nativeBodies [][]byte
	for _, input := range options.NativeLinkInputs {
		body, err := os.ReadFile(input)
		if err != nil {
			return &testModuleBuild{phase: "native-link-input", message: err.Error()}
		}
		nativeBodies = append(nativeBodies, body)
		fmt.Fprintf(nativeHash, "%d:%s%d:", len(input), input, len(body))
		nativeHash.Write(body)
	}
	key := testBuildKey{test.snapshot, options.Verify, hex.EncodeToString(nativeHash.Sum(nil))}
	// Undiscovered internal declarations cannot assert shared module identity.
	if key.snapshot == nil {
		key.snapshot = &testModuleSnapshot{}
	}
	if build := s.builds[key]; build != nil {
		return build
	}
	build := &testModuleBuild{base: evt1OutputBase(test.sourcePath), executables: map[string]string{}}
	s.builds[key] = build
	fail := func(phase string, err error) *testModuleBuild {
		build.phase, build.message = phase, err.Error()
		return build
	}
	policy := ConservativeCompilationPolicy()
	if options.Verify {
		policy = VerifyCompilationPolicy()
	}
	outputs, err := GenerateForTargetWithPolicy(test.module, test.sourceBytes, GenericC11Target(), policy)
	if err != nil {
		return fail("compile", err)
	}
	keys := make([]string, 0, len(outputs))
	for name := range outputs {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	buildHash := sha256.New()
	for _, name := range keys {
		buildHash.Write(outputs[name])
	}
	for i, input := range options.NativeLinkInputs {
		buildHash.Write([]byte(filepath.Base(input)))
		buildHash.Write(nativeBodies[i])
	}
	build.identity = hex.EncodeToString(buildHash.Sum(nil))
	build.dir, err = os.MkdirTemp("", "concept-test-")
	if err != nil {
		return fail("runner", err)
	}
	if err := Write(build.dir, outputs); err != nil {
		return fail("runner", err)
	}
	if _, present := outputs[build.base+".machine.S"]; present {
		build.machineHelper = filepath.Join(build.dir, build.base+".machine.S")
	}
	if test.module.Profile == evt1VulkanProfileName {
		// Vulkan retains its runtime selection and binding checks per test.
		return build
	}
	build.compiler, _, err = evt1TestCompiler(build.dir, "", "", "")
	if len(options.NativeLinkInputs) != 0 {
		if options.NativeLinker != "clang++" && options.NativeLinker != "g++" {
			return fail("compiler-unavailable", fmt.Errorf("native linker must be clang++ or g++"))
		}
		compiler := "clang"
		if options.NativeLinker == "g++" {
			compiler = "gcc"
		}
		build.compiler, err = exec.LookPath(compiler)
		if err == nil {
			_, err = exec.LookPath(options.NativeLinker)
		}
		build.target = "/" + filepath.Base(options.NativeLinker)
	} else {
		build.target = "/" + filepath.Base(build.compiler)
	}
	if err != nil {
		return fail("compiler-unavailable", err)
	}
	sources := []string{filepath.Join(build.dir, build.base+".generated.c")}
	if build.machineHelper != "" {
		sources = append(sources, build.machineHelper)
	}
	for i, source := range sources {
		object := filepath.Join(build.dir, fmt.Sprintf("module_%d.o", i))
		args := []string{"-std=c11", "-Wall", "-Wextra", "-I", build.dir, "-c", source, "-o", object}
		if output, err := exec.Command(build.compiler, args...).CombinedOutput(); err != nil {
			phase := "native-compile"
			if len(options.NativeLinkInputs) != 0 {
				phase = "concept-c-compile"
			}
			return fail(phase, fmt.Errorf("%s: %v\n%s", build.compiler, err, output))
		}
		build.objects = append(build.objects, object)
	}
	return build
}

func (b *testModuleBuild) executable(test TestDeclaration, values []any, options TestRunOptions) (string, string, string) {
	harness := evt1TestHarness(test, values, b.base, evt1SemanticSymbolBase(test.module))
	if executable := b.executables[harness]; executable != "" && test.module.Profile != evt1VulkanProfileName {
		return executable, "", ""
	}
	ordinal := len(b.executables)
	harnessPath := filepath.Join(b.dir, fmt.Sprintf("harness_%d.c", ordinal))
	if err := os.WriteFile(harnessPath, []byte(harness), 0o644); err != nil {
		return "", "runner", err.Error()
	}
	executable := filepath.Join(b.dir, fmt.Sprintf("test_%d", ordinal))
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if len(options.NativeLinkInputs) != 0 {
		object := harnessPath + ".o"
		args := []string{"-std=c11", "-Wall", "-Wextra", "-I", b.dir, "-c", harnessPath, "-o", object}
		if output, err := exec.Command(b.compiler, args...).CombinedOutput(); err != nil {
			return "", "concept-c-compile", fmt.Sprintf("%v\n%s", err, output)
		}
		args = append(append([]string{}, b.objects...), object)
		args = append(args, options.NativeLinkInputs...)
		args = append(args, "-o", executable)
		if output, err := exec.Command(options.NativeLinker, args...).CombinedOutput(); err != nil {
			return "", "link", fmt.Sprintf("%v\n%s", err, output)
		}
	} else {
		compiler := b.compiler
		args := []string{"-std=c11", "-Wall", "-Wextra", "-I", b.dir}
		args = append(args, b.objects...)
		args = append(args, harnessPath, "-lm", "-o", executable)
		if test.module.Profile == evt1VulkanProfileName {
			var err error
			compiler, args, err = evt1TestCompiler(b.dir, filepath.Join(b.dir, b.base+".generated.c"), harnessPath, executable)
			if err != nil {
				return "", "compiler-unavailable", err.Error()
			}
			vulkan, err := evt1SelectVulkanRuntime(test.sourcePath)
			if err != nil {
				return "", "vulkan-runtime", err.Error()
			}
			if err := evt1CheckVulkanKernelBindings(filepath.Dir(test.sourcePath)); err != nil {
				return "", "vulkan-kernel-binding", err.Error()
			}
			head := append(append([]string{}, vulkan.CFlags...), vulkan.Sources...)
			if b.machineHelper != "" {
				args = append(args[:len(args)-2], append([]string{b.machineHelper}, args[len(args)-2:]...)...)
			}
			args = append(append(head, args...), vulkan.LDFlags...)
			b.target = "/vulkan-" + vulkan.Name + "/" + filepath.Base(compiler)
		}
		if output, err := exec.Command(compiler, args...).CombinedOutput(); err != nil {
			return "", "native-compile", strings.TrimSpace(string(output))
		}
	}
	b.executables[harness] = executable
	return executable, "", ""
}
