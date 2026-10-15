package main

import (
	"fmt"
	"strings"
)

type compileOptions struct {
	verify        bool
	outputDir     string
	moduleRoots   []string
	sourceModules bool
}

func parseCompileOptions(command string, args []string) (compileOptions, error) {
	var options compileOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--verify":
			options.verify = true
		case "--source-modules":
			options.sourceModules = true
		case "--out-dir", "--module-root":
			name := args[i]
			if i+1 >= len(args) || (args[i+1] == "" || strings.HasPrefix(args[i+1], "--")) {
				return options, fmt.Errorf("%s requires a directory", name)
			}
			i++
			if name == "--out-dir" {
				if options.outputDir != "" {
					return options, fmt.Errorf("--out-dir must occur once")
				}
				options.outputDir = args[i]
			} else {
				options.moduleRoots = append(options.moduleRoots, args[i])
			}
		default:
			return options, fmt.Errorf("unknown option %s", args[i])
		}
	}
	if options.verify && command != "plan" && command != "emit-c" && command != "mir" {
		return options, fmt.Errorf("--verify is unsupported for %s", command)
	}
	if options.outputDir != "" && command != "emit-c" {
		return options, fmt.Errorf("--out-dir is supported by emit-c")
	}
	return options, nil
}
