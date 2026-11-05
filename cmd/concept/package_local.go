package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

func runInitCommand(args []string) {
	if len(args) == 0 {
		fail(fmt.Errorf("usage: concept init <directory> [--name Name] [--author Author]"))
	}
	root := args[0]
	name := filepath.Base(filepath.Clean(root))
	if root == "." {
		if absolute, err := filepath.Abs(root); err == nil {
			name = filepath.Base(absolute)
		}
	}
	author := os.Getenv("USERNAME")
	if author == "" {
		author = os.Getenv("USER")
	}
	if author == "" {
		author = "Your name"
	}
	options := flag.NewFlagSet("init", flag.ContinueOnError)
	options.StringVar(&name, "name", name, "package and module identifier")
	options.StringVar(&author, "author", author, "package author")
	if err := options.Parse(args[1:]); err != nil {
		fail(err)
	}
	if options.NArg() != 0 {
		fail(fmt.Errorf("unexpected init arguments: %s", strings.Join(options.Args(), " ")))
	}
	paths, err := concept.InitPackage(root, name, author)
	if err != nil {
		fail(err)
	}
	for _, path := range paths {
		fmt.Println(filepath.ToSlash(path))
	}
	fmt.Printf("Run: concept package test %s\n", root)
}

func runLocalPackageCommand(args []string) bool {
	action, root := args[0], args[1]
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.concept")); err != nil {
		return false
	}
	if nativeProjectDir(root) && (action == "build" || action == "test") {
		verify := len(args) == 3 && args[2] == "--verify"
		if len(args) > 2 && !verify {
			fail(fmt.Errorf("unexpected native package arguments"))
		}
		runNativeCommand(action, root, verify)
		return true
	}
	options := flag.NewFlagSet("package "+action, flag.ContinueOnError)
	library := options.String("library-root", concept.CompilerLibraryRoot(), "installed Concept libraries")
	verify := options.Bool("verify", false, "run tests in Verify mode")
	if err := options.Parse(args[2:]); err != nil {
		fail(err)
	}
	if options.NArg() != 0 {
		fail(fmt.Errorf("unexpected package arguments"))
	}
	if action != "build" && action != "graph" && action != "test" {
		fail(fmt.Errorf("unknown package action %s", action))
	}
	graph, err := concept.BuildPackageDirectory(root, filepath.Join(root, ".concept", "packages"), *library)
	if err != nil {
		fail(err)
	}
	if action != "test" {
		body, err := concept.MarshalPackageGraph(graph)
		if err != nil {
			fail(err)
		}
		fmt.Println(string(body))
		return true
	}
	manifest, err := concept.DiscoverTestsWithModuleRoots(filepath.Join(root, "tests"), []string{filepath.Join(root, "src"), root})
	if err != nil {
		fail(err)
	}
	run, err := concept.RunTests(manifest, concept.TestRunOptions{Verify: *verify})
	if err != nil {
		fail(err)
	}
	for _, result := range run.Results {
		fmt.Printf("%-20s %s\n", result.Status, result.TestID)
		if result.Failure != nil {
			fmt.Printf("  %s: %s\n", result.Failure.Kind, result.Failure.Message)
		}
	}
	fmt.Printf("\n%d passed, %d failed\n", run.Passed, run.Failed)
	if run.Failed != 0 {
		os.Exit(1)
	}
	return true
}
