// Command concept is the active Concept EVT1 Stage 0 compiler driver.
package main

import (
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
  concept emit-c <file>
  concept mir <file>
  concept plan <file>
  concept explain <file>[:line] [--json] [--verbose]
  concept test [path-or-filter] [--filter text] [--list] [--verbose]

Commands:
  check   parse and semantically validate a Concept source file
  emit-c  write generated strict-C11 implementation to stdout
  mir     write deterministic MIR JSON to stdout
  plan    write deterministic LoweringPlan JSON to stdout
  explain display the proof graph for an Assert.Concept source contract
  test    discover and execute .concept_test sources through strict C11
`

const testUsage = `Usage:
  concept test
  concept test <path-or-filter>
  concept test --filter <text>
  concept test --list
  concept test --verbose
`

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		fmt.Print(usage)
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "test" {
		runTestCommand(os.Args[2:])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "explain" {
		runExplainCommand(os.Args[2:])
		return
	}
	if len(os.Args) != 3 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	command, sourcePath := os.Args[1], os.Args[2]
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		fail(err)
	}
	module, err := concept.Parse(filepath.ToSlash(sourcePath), string(body))
	if err != nil {
		fail(err)
	}

	switch command {
	case "check":
		fmt.Printf("%s: ok (%s, %s)\n", filepath.ToSlash(sourcePath), module.Profile, concept.CompilerID)
	case "plan":
		output, err := concept.GeneratePlan(module, concept.GenericC11Target())
		if err != nil {
			fail(err)
		}
		_, _ = os.Stdout.Write(output)
	case "emit-c", "mir":
		outputs, err := concept.Generate(module, body)
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

func runExplainCommand(args []string) {
	if len(args) < 1 || len(args) > 3 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	sourcePath, line, jsonOutput, verbose := args[0], 0, false, false
	for _, arg := range args[1:] {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--verbose":
			verbose = true
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
	graph, err := concept.ExplainSource(filepath.ToSlash(proofSourcePath), string(body), line)
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

func runTestCommand(args []string) {
	root, filter, list, verbose := ".", "", false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Print(testUsage)
			return
		case "--list":
			list = true
		case "--verbose":
			verbose = true
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
	run, err := concept.RunTests(manifest, concept.TestRunOptions{Filter: filter, BenchmarkWarmup: 1, BenchmarkIterations: 5})
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
