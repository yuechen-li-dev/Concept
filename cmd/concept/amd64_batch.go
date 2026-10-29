package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

// Batch source checking is independent for each input; only the immutable
// Concept-authored backend executable is shared. Source errors precede building
// Cathedral, and no output or test result is cached.
func printConceptAMD64Batch(paths []string) error {
	type input struct {
		path     string
		machine  concept.MachineModule
		artifact []byte
	}
	inputs := make([]input, 0, len(paths))
	for _, path := range paths {
		if strings.HasPrefix(path, "-") {
			return fmt.Errorf("amd64 batch expects source paths, got %q", path)
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		module, err := concept.ParseWithSemanticModuleRoots(filepath.ToSlash(path), string(source), semanticModuleRoots(path))
		if err != nil {
			return err
		}
		machine, err := concept.GenerateMachineIR(module)
		if err != nil {
			return err
		}
		artifact, err := concept.EncodeMachineBridge(machine)
		if err != nil {
			return err
		}
		inputs = append(inputs, input{path, machine, artifact})
	}
	session, err := newAMD64BackendSession()
	if err != nil {
		return err
	}
	defer session.close()
	for _, in := range inputs {
		output, err := session.emit(in.machine, in.artifact)
		if err != nil {
			return fmt.Errorf("%s: %w", in.path, err)
		}
		fmt.Printf("source %s\n%s", filepath.ToSlash(in.path), output)
	}
	return nil
}
