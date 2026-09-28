package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

// The temporary C executable is the bootstrap host for the Concept module.
// Go only transports verified MachineIR and displays the bytes it returns.
func printConceptAMD64(machine concept.MachineModule, artifact []byte) error {
	backendPath, err := findAMD64BackendSource()
	if err != nil {
		return err
	}
	source, err := os.ReadFile(backendPath)
	if err != nil {
		return err
	}
	module, err := concept.Parse(backendPath, string(source))
	if err != nil {
		return err
	}
	outputs, err := concept.Generate(module, source)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "concept-amd64-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := concept.Write(dir, outputs); err != nil {
		return err
	}
	var generatedC string
	for name := range outputs {
		if strings.HasSuffix(name, ".generated.c") {
			generatedC = filepath.Join(dir, name)
		}
	}
	if generatedC == "" {
		return fmt.Errorf("EVT2D_BACKEND_C_MISSING")
	}
	bridgePath := filepath.Join(dir, "module.cmir")
	if err := os.WriteFile(bridgePath, artifact, 0600); err != nil {
		return err
	}
	harnessPath := filepath.Join(dir, "backend_host.c")
	if err := os.WriteFile(harnessPath, []byte(amd64BootstrapHost), 0600); err != nil {
		return err
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		return fmt.Errorf("EVT2D_C11_COMPILER_MISSING: %w", err)
	}
	backendExe := filepath.Join(dir, "concept-amd64-backend.exe")
	cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-O2", "-I", dir, generatedC, harnessPath, "-o", backendExe)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("EVT2D_BACKEND_BOOTSTRAP: %w\n%s", err, output)
	}
	for index, function := range machine.Functions {
		cmd = exec.Command(backendExe, bridgePath, strconv.Itoa(index))
		code, err := cmd.Output()
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("EVT2D_BACKEND %s: %w: %s", function.Name, err, e.Stderr)
			}
			return fmt.Errorf("EVT2D_BACKEND %s: %w", function.Name, err)
		}
		fmt.Printf("fn %s [%s]\n  0000: %s\n", function.Name, function.Identity, hex.EncodeToString(code))
	}
	return nil
}

func findAMD64BackendSource() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "libraries", "Standard", "Backend", "AMD64.concept")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("EVT2D_BACKEND_SOURCE_MISSING: run inside a Concept checkout")
		}
		dir = parent
	}
}

const amd64BootstrapHost = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdlib.h>
int main(int argc, char** argv) {
  if (argc != 3) return 2;
  FILE* file = fopen(argv[1], "rb");
  if (!file) return 3;
  if (fseek(file, 0, SEEK_END) != 0) return 4;
  long size = ftell(file);
  if (size <= 0 || size > 10000000 || fseek(file, 0, SEEK_SET) != 0) return 5;
  unsigned char* input = malloc((size_t)size);
  if (!input || fread(input, 1, (size_t)size, file) != (size_t)size) return 6;
  fclose(file);
  unsigned char output[65536];
  concept_readonly_span_byte source = {input, (size_t)size};
  concept_span_byte target = {output, sizeof output};
  concept_result_int_backend_error result = concept_standard__backend__amd64_emit_function(source, atoi(argv[2]), target);
  free(input);
  if (result.tag != 0) {
    fprintf(stderr, "Concept AMD64 backend error tag=%u\n", result.payload.error.error.tag);
    return 7;
  }
  if (fwrite(output, 1, (size_t)result.payload.ok.value, stdout) != (size_t)result.payload.ok.value) return 8;
  return 0;
}
`
