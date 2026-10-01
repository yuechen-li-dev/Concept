// Command concept is the active Concept EVT1 Stage 0 compiler driver.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

const usage = `Concept EVT1 Stage 0 / Go

Usage:
  concept check <file>
  concept lint <file-or-project> [--verify]
  concept format <file-or-project> [--check]
  concept build-module <file>
  concept emit-c <file> [--verify]
  concept mir <file> [--verify]
  concept lir <file>
  concept machineir <file>
  concept machineir-bin <file> > out.cmir
  concept amd64 <file>
  concept plan <file> [--verify]
  concept explain <file>[:line] [--json] [--verbose]
  concept explain <file> --generated <symbol> [--json] [--verbose]
  concept explain <file> --concept 'Trace<Node>' [--json] [--verbose]
  concept explain <file> --policy <concept> --subject <declaration> [--json] [--verbose]
  concept explain <file> --must-use <symbol> [--json] [--verbose]
  concept reflect <file>
  concept generated <file> [symbol]
  concept test [path-or-filter] [--filter text] [--list] [--verbose] [--verify]
  concept package build <name>
  concept package test <name>
  concept package graph <name>
  concept build <native-project-dir>
  concept check <native-project-dir>
  concept test <native-project-dir>
  concept test <native-project-dir> --verify
  concept plan <native-project-dir>
  concept vulkan-bind <kernel.spv> [-o <Module.concept>] [--check] [--describe]

Commands:
  check   parse and semantically validate a Concept source file
  lint    evaluate manifest.concept policies over bound declarations
  format  apply presentation-only canonical whitespace and comment-preserving layout
  build-module  write a deterministic concept-module.v2 artifact to stdout
  emit-c  write generated strict-C11 implementation to stdout
  mir     write deterministic MIR JSON to stdout
  lir     verify and write target-independent EVT2 LIR to stdout
  machineir  verify and write AMD64 Windows MachineIR to stdout
  machineir-bin  write versioned typed AMD64 MachineIR bytes to stdout
  amd64  compile the Concept backend through C11 and print native bytes
  plan    write deterministic LoweringPlan JSON to stdout
  explain display the proof graph for an Assert.Concept source contract
  reflect display compile-time structural results from explicit reflect<T>; sites
  generated display checked generated declarations and provenance
  test    discover and execute .concept_test sources through strict C11
  package build or test a repository-local manifest.concept package graph
`

const testUsage = `Usage:
  concept test
  concept test <path-or-filter>
  concept test --filter <text>
  concept test --list
  concept test --verbose
  concept test --verify
`

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		fmt.Print(usage)
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "test" {
		if root, verify, verbose, ok := nativeTestArgs(os.Args[2:]); ok {
			runNativeTestCommand(root, verify, verbose)
			return
		}
		runTestCommand(os.Args[2:])
		return
	}
	if len(os.Args) == 3 && (os.Args[1] == "build" || os.Args[1] == "check" || os.Args[1] == "plan") && nativeProjectDir(os.Args[2]) {
		runNativeCommand(os.Args[1], os.Args[2], false)
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "explain" {
		runExplainCommand(os.Args[2:])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "generated" {
		runGeneratedCommand(os.Args[2:])
		return
	}
	if (len(os.Args) == 3 || len(os.Args) == 4 && os.Args[3] == "--verify") && os.Args[1] == "lint" {
		findings, err := concept.LintPath(os.Args[2], semanticModuleRoots(os.Args[2]))
		if err != nil {
			fail(err)
		}
		for _, finding := range findings {
			fmt.Println(concept.FormatLintFinding(finding))
		}
		if concept.HasLintErrors(findings) {
			os.Exit(1)
		}
		return
	}
	if (len(os.Args) == 3 || len(os.Args) == 4 && os.Args[3] == "--check") && os.Args[1] == "format" {
		edits, err := concept.FormatPath(os.Args[2])
		if err != nil {
			fail(err)
		}
		paths := make([]string, 0, len(edits))
		for path := range edits {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		if len(os.Args) == 4 {
			for _, path := range paths {
				fmt.Println(filepath.ToSlash(path))
			}
			if len(paths) > 0 {
				os.Exit(1)
			}
			return
		}
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				fail(err)
			}
			if err := os.WriteFile(path, []byte(edits[path]), info.Mode().Perm()); err != nil {
				fail(err)
			}
			fmt.Println(filepath.ToSlash(path))
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "package" {
		runPackageCommand(os.Args[2:])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "vulkan-bind" {
		runVulkanBindCommand(os.Args[2:])
		return
	}
	verify := len(os.Args) == 4 && os.Args[3] == "--verify"
	if len(os.Args) != 3 && !verify {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	command, sourcePath := os.Args[1], os.Args[2]
	if verify && command != "plan" && command != "emit-c" && command != "mir" {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		fail(err)
	}
	roots := semanticModuleRoots(sourcePath)
	if command == "build-module" {
		output, err := concept.CompileSemanticModuleWithRoots(filepath.ToSlash(sourcePath), string(body), roots)
		if err != nil {
			fail(err)
		}
		_, _ = os.Stdout.Write(output)
		return
	}
	var module concept.Module
	if concept.UsesVulkanProfile(filepath.ToSlash(sourcePath), string(body)) {
		module, err = concept.ParseWithModuleRootsForProfile(filepath.ToSlash(sourcePath), string(body), roots)
	} else {
		module, err = concept.ParseWithSemanticModuleRoots(filepath.ToSlash(sourcePath), string(body), roots)
	}
	if err != nil {
		fail(err)
	}

	switch command {
	case "check":
		fmt.Printf("%s: ok (%s, %s)\n", filepath.ToSlash(sourcePath), module.Profile, concept.CompilerID)
	case "lir":
		lir, err := concept.GenerateLIR(module)
		if err != nil {
			fail(err)
		}
		fmt.Print(lir.String())
	case "machineir":
		machine, err := concept.GenerateMachineIR(module)
		if err != nil {
			fail(err)
		}
		fmt.Print(machine.String())
	case "machineir-bin":
		machine, err := concept.GenerateMachineIR(module)
		if err != nil {
			fail(err)
		}
		artifact, err := concept.EncodeMachineBridge(machine)
		if err != nil {
			fail(err)
		}
		if _, err := os.Stdout.Write(artifact); err != nil {
			fail(err)
		}
	case "amd64":
		machine, err := concept.GenerateMachineIR(module)
		if err != nil {
			fail(err)
		}
		artifact, err := concept.EncodeMachineBridge(machine)
		if err != nil {
			fail(err)
		}
		if err := printConceptAMD64(machine, artifact); err != nil {
			fail(err)
		}
	case "reflect":
		output, err := json.MarshalIndent(module.ReflectionResults, "", "  ")
		if err != nil {
			fail(err)
		}
		fmt.Println(string(output))
	case "plan":
		policy := concept.ConservativeCompilationPolicy()
		if verify {
			policy = concept.VerifyCompilationPolicy()
		}
		output, err := concept.GeneratePlanWithPolicy(module, concept.GenericC11Target(), policy)
		if err != nil {
			fail(err)
		}
		_, _ = os.Stdout.Write(output)
	case "emit-c", "mir":
		policy := concept.ConservativeCompilationPolicy()
		if verify {
			policy = concept.VerifyCompilationPolicy()
		}
		outputs, err := concept.GenerateForTargetWithPolicy(module, body, concept.GenericC11Target(), policy)
		if err != nil {
			fail(err)
		}
		suffix := ".generated.c"
		if command == "mir" {
			suffix = ".mir.json"
		}
		_, output := selectOutput(outputs, suffix)
		if output == nil {
			fail(fmt.Errorf("compiler produced no %s artifact", suffix))
		}
		_, _ = os.Stdout.Write(output)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", command, usage)
		os.Exit(2)
	}
}

func nativeProjectDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	body, err := os.ReadFile(filepath.Join(path, "manifest.concept"))
	return err == nil && strings.Contains(string(body), "NativeProjectManifest Native")
}

// nativeTestArgs recognizes `concept test <native project> [--verify] [--verbose]`.
func nativeTestArgs(args []string) (root string, verify, verbose, ok bool) {
	for _, arg := range args {
		switch arg {
		case "--verify":
			verify = true
		case "--verbose":
			verbose = true
		default:
			if root != "" {
				return "", false, false, false
			}
			root = arg
		}
	}
	return root, verify, verbose, root != "" && nativeProjectDir(root)
}

func runNativeTestCommand(root string, verify, verbose bool) {
	runNativeCommandWith("test", root, verify, verbose)
}

func runNativeCommand(action, root string, verify bool) {
	runNativeCommandWith(action, root, verify, false)
}

func runNativeCommandWith(action, root string, verify, verbose bool) {
	project, err := concept.LoadNativeProject(root)
	if err != nil {
		fail(err)
	}
	plan, err := concept.NativeBuildPlan(project)
	if err != nil {
		fail(err)
	}
	if action == "plan" {
		body, err := concept.MarshalNativePlan(plan)
		if err != nil {
			fail(err)
		}
		fmt.Println(string(body))
		return
	}
	if err := concept.CheckNativeABI(project); err != nil {
		fail(err)
	}
	artifacts, identity, err := concept.BuildNativeCompanionArtifacts(project)
	if err != nil {
		fail(fmt.Errorf("NATIVE_COMPANION_INVALID: %w", err))
	}
	if identity.BuildInputHash != plan.BuildInputHash || identity.SemanticCompanionHash != plan.SemanticCompanionHash || identity.CompilerVersion != plan.ToolchainVersion {
		fail(fmt.Errorf("NATIVE_ABI_EVIDENCE_STALE: native inputs changed after planning"))
	}
	for _, test := range project.Tests {
		if _, err := os.Stat(filepath.Join(project.Root, filepath.FromSlash(test))); err != nil {
			fail(fmt.Errorf("NATIVE_TEST_MISSING: %s: %w", test, err))
		}
	}
	testManifest, err := concept.DiscoverTestsWithNativeABI(filepath.Join(project.Root, "tests"), artifacts, identity)
	if err != nil {
		fail(fmt.Errorf("NATIVE_TEST_INVALID: %w", err))
	}
	allowed := map[string]bool{}
	for _, path := range project.Tests {
		rel, err := filepath.Rel(filepath.Join(project.Root, "tests"), filepath.Join(project.Root, filepath.FromSlash(path)))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			fail(fmt.Errorf("NATIVE_TEST_OUTSIDE_ROOT: %s", path))
		}
		allowed[filepath.ToSlash(rel)] = true
	}
	selected := testManifest.Tests[:0]
	for _, item := range testManifest.Tests {
		if allowed[item.Source] {
			selected = append(selected, item)
		}
	}
	testManifest.Tests = selected
	if len(testManifest.Tests) == 0 {
		fail(fmt.Errorf("NATIVE_TEST_EMPTY: manifest lists no discovered test functions"))
	}
	if action == "check" {
		fmt.Printf("%s: native project, companions, and tests ok\n", project.Name)
		return
	}
	result, err := concept.RunNativeBuild(project, plan)
	if err != nil {
		fail(err)
	}
	if err := concept.ValidateNativeBuildOutputs(project, result); err != nil {
		fail(err)
	}
	if action == "build" {
		body, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fail(err)
		}
		fmt.Println(string(body))
		return
	}
	if action != "test" {
		fail(fmt.Errorf("unsupported native action %s", action))
	}
	var inputs []string
	for _, target := range project.Targets {
		if target.Kind == "StaticLibrary" {
			inputs = append(inputs, filepath.Join(project.Root, filepath.FromSlash(target.Output)))
		}
	}
	if len(inputs) == 0 {
		fail(fmt.Errorf("NATIVE_TEST_LINK_INPUT_MISSING: project has no static library target"))
	}
	linker := "clang++"
	if project.Toolchain == "GCC" {
		linker = "g++"
	}
	run, err := concept.RunTests(testManifest, concept.TestRunOptions{NativeLinker: linker, NativeLinkInputs: inputs, Verify: verify})
	if err != nil {
		fail(err)
	}
	for _, item := range run.Results {
		fmt.Printf("%-20s %s\n", item.Status, item.TestID)
		if item.Failure != nil {
			fmt.Printf("  %s: %s\n", item.Failure.Kind, item.Failure.Message)
		}
		if verbose && item.Stderr != "" {
			fmt.Print("  stderr: ", strings.ReplaceAll(strings.TrimSpace(item.Stderr), "\n", "\n          "), "\n")
		}
	}
	fmt.Printf("\n%d passed, %d failed\n", run.Passed, run.Failed)
	if run.Failed != 0 {
		os.Exit(1)
	}
}

func runPackageCommand(args []string) {
	if len(args) != 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	action, name := args[0], args[1]
	switch action {
	case "build", "graph":
		graph, err := concept.BuildPackage("libraries", "artifacts", name)
		if err != nil {
			fail(err)
		}
		body, err := concept.MarshalPackageGraph(graph)
		if err != nil {
			fail(err)
		}
		fmt.Println(string(body))
	case "test":
		if _, err := concept.BuildPackage("libraries", "artifacts", name); err != nil {
			fail(err)
		}
		manifest, err := concept.DiscoverTests(filepath.Join("libraries", name))
		if err != nil {
			fail(err)
		}
		run, err := concept.RunTests(manifest, concept.TestRunOptions{BenchmarkWarmup: 1, BenchmarkIterations: 5})
		if err != nil {
			fail(err)
		}
		for _, result := range run.Results {
			fmt.Printf("%-20s %s\n", result.Status, result.TestID)
		}
		fmt.Printf("\n%d passed, %d failed, %d prophecies fulfilled, %d benchmarks\n", run.Passed, run.Failed, run.Fulfilled, run.Benchmarks)
		if run.Failed != 0 {
			os.Exit(1)
		}
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func semanticModuleRoots(sourcePath string) []string {
	roots := []string{filepath.Dir(sourcePath)}
	if root := nativeCompanionRoot(sourcePath); root != "" {
		roots = append(roots, root)
	}
	if configured := os.Getenv("CONCEPT_MODULE_ROOTS"); configured != "" {
		for _, root := range filepath.SplitList(configured) {
			if root != "" {
				roots = append(roots, root)
			}
		}
	}
	return roots
}

func nativeCompanionRoot(sourcePath string) string {
	for dir := filepath.Dir(sourcePath); ; dir = filepath.Dir(dir) {
		if nativeProjectDir(dir) {
			root := filepath.Join(dir, "concept")
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				return root
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
	}
}

func runExplainCommand(args []string) {
	if len(args) < 1 || len(args) > 7 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	sourcePath, line, jsonOutput, verbose, generatedSymbol, conceptGoal, policyName, subjectName, mustUseName := args[0], 0, false, false, "", "", "", "", ""
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOutput = true
		case "--verbose":
			verbose = true
		case "--generated":
			i++
			if i >= len(args) {
				fmt.Fprint(os.Stderr, usage)
				os.Exit(2)
			}
			generatedSymbol = args[i]
		case "--concept":
			i++
			if i >= len(args) {
				fmt.Fprint(os.Stderr, usage)
				os.Exit(2)
			}
			conceptGoal = args[i]
		case "--policy":
			i++
			if i >= len(args) {
				fmt.Fprint(os.Stderr, usage)
				os.Exit(2)
			}
			policyName = args[i]
		case "--subject":
			i++
			if i >= len(args) {
				fmt.Fprint(os.Stderr, usage)
				os.Exit(2)
			}
			subjectName = args[i]
		case "--must-use":
			i++
			if i >= len(args) {
				fmt.Fprint(os.Stderr, usage)
				os.Exit(2)
			}
			mustUseName = args[i]
		default:
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
	}
	if _, err := os.Stat(sourcePath); err != nil {
		if split := strings.LastIndex(sourcePath, ":"); split > 1 {
			if parsed, parseErr := strconv.Atoi(sourcePath[split+1:]); parseErr == nil && parsed > 0 {
				line, sourcePath = parsed, sourcePath[:split]
			}
		}
	}
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		fail(err)
	}
	proofSourcePath := sourcePath
	if relative, relativeErr := filepath.Rel(".", sourcePath); relativeErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		proofSourcePath = relative
	}
	var graph concept.ProofGraph
	if mustUseName != "" {
		graph, err = concept.ExplainMustUse(sourcePath, mustUseName, semanticModuleRoots(sourcePath))
	} else if policyName != "" {
		if subjectName == "" {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		graph, err = concept.ExplainPolicy(sourcePath, policyName, subjectName, semanticModuleRoots(sourcePath))
	} else if generatedSymbol != "" || conceptGoal != "" {
		module, parseErr := concept.ParseWithSemanticModuleRoots(filepath.ToSlash(proofSourcePath), string(body), semanticModuleRoots(sourcePath))
		if parseErr != nil {
			fail(parseErr)
		}
		if conceptGoal != "" {
			graph, err = concept.ExplainGeneratedConcept(module, conceptGoal)
		} else {
			graph, err = concept.ExplainGeneratedDeclaration(module, generatedSymbol)
		}
	} else {
		if companionRoot := nativeCompanionRoot(sourcePath); companionRoot != "" {
			project, loadErr := concept.LoadNativeProject(filepath.Dir(companionRoot))
			if loadErr != nil {
				fail(loadErr)
			}
			artifacts, identity, buildErr := concept.BuildNativeCompanionArtifacts(project)
			if buildErr != nil {
				fail(buildErr)
			}
			graph, err = concept.ExplainSourceWithNativeSemanticModules(filepath.ToSlash(proofSourcePath), string(body), line, artifacts, identity)
		} else {
			graph, err = concept.ExplainSourceWithSemanticModuleRoots(filepath.ToSlash(proofSourcePath), string(body), line, semanticModuleRoots(sourcePath))
		}
	}
	if err != nil {
		fail(err)
	}
	if jsonOutput {
		output, err := concept.SerializeProof(graph)
		if err != nil {
			fail(err)
		}
		_, _ = os.Stdout.Write(output)
		return
	}
	if verbose {
		fmt.Print(concept.RenderProofVerbose(graph))
	} else {
		fmt.Print(concept.RenderProofSummary(graph))
	}
	fmt.Println()
}

func runGeneratedCommand(args []string) {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	sourcePath, symbol := args[0], ""
	if len(args) == 2 {
		symbol = args[1]
	}
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		fail(err)
	}
	module, err := concept.ParseWithSemanticModuleRoots(filepath.ToSlash(sourcePath), string(body), semanticModuleRoots(sourcePath))
	if err != nil {
		fail(err)
	}
	output, err := concept.InspectGeneratedDeclarations(module, symbol)
	if err != nil {
		fail(err)
	}
	fmt.Println(string(output))
}

func runTestCommand(args []string) {
	root, filter, list, verbose, verify := ".", "", false, false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Print(testUsage)
			return
		case "--list":
			list = true
		case "--verbose":
			verbose = true
		case "--verify":
			verify = true
		case "--filter":
			if i+1 >= len(args) {
				fmt.Fprint(os.Stderr, testUsage)
				os.Exit(2)
			}
			i++
			filter = args[i]
		default:
			if _, err := os.Stat(args[i]); err == nil {
				root = args[i]
			} else if filter == "" {
				filter = args[i]
			} else {
				fmt.Fprint(os.Stderr, testUsage)
				os.Exit(2)
			}
		}
	}
	manifest, err := concept.DiscoverTests(root)
	if err != nil {
		fail(err)
	}
	if list {
		for _, test := range manifest.Tests {
			if filter == "" || strings.Contains(strings.ToLower(test.TestID), strings.ToLower(filter)) {
				fmt.Printf("%-10s %s  %s:%d\n", strings.ToUpper(string(test.Kind)), test.TestID, test.Source, test.SourceLine)
			}
		}
		return
	}
	run, err := concept.RunTests(manifest, concept.TestRunOptions{Filter: filter, BenchmarkWarmup: 1, BenchmarkIterations: 5, Verify: verify})
	if err != nil {
		fail(err)
	}
	for _, result := range run.Results {
		fmt.Printf("%-20s %s\n", result.Status, result.TestID)
		if result.Failure != nil {
			fmt.Printf("  %s: %s\n", result.Failure.Kind, result.Failure.Message)
		}
		if verbose && result.Stderr != "" {
			fmt.Print("  stderr: ", strings.ReplaceAll(strings.TrimSpace(result.Stderr), "\n", "\n          "), "\n")
		}
		if result.Benchmark != nil {
			fmt.Printf("  %d iterations; total=%s mean=%s min=%s median=%s max=%s\n", result.Benchmark.Iterations, time.Duration(result.Benchmark.TotalNanos), time.Duration(result.Benchmark.MeanNanos), time.Duration(result.Benchmark.MinNanos), time.Duration(result.Benchmark.MedianNanos), time.Duration(result.Benchmark.MaxNanos))
		}
	}
	fmt.Printf("\n%d passed, %d failed, %d prophecies fulfilled, %d benchmarks\n", run.Passed, run.Failed, run.Fulfilled, run.Benchmarks)
	if run.Failed != 0 {
		os.Exit(1)
	}
}

func selectOutput(outputs concept.Outputs, suffix string) (string, []byte) {
	keys := make([]string, 0, len(outputs))
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasSuffix(key, suffix) {
			return key, outputs[key]
		}
	}
	return "", nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
