// Command concept is the active Concept EVT1 Stage 0 compiler driver.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

const usage = `Concept EVT1 Stage 0 / Go

Usage:
  concept check <file>
  concept emit-c <file>
  concept mir <file>
  concept plan <file>

Commands:
  check   parse and semantically validate a Concept source file
  emit-c  write generated strict-C11 implementation to stdout
  mir     write deterministic MIR JSON to stdout
  plan    write deterministic LoweringPlan JSON to stdout
`

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		fmt.Print(usage)
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
