package main

import (
	"fmt"
	"github.com/yuechen-li-dev/Concept/internal/concept"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: machinebridgegen <repository-root>")
		os.Exit(2)
	}
	root := os.Args[1]
	schema := filepath.Join(root, "libraries/Standard/Backend/BridgeSchema.concept")
	source, err := os.ReadFile(schema)
	if err != nil {
		fail(err)
	}
	module, err := concept.Parse(schema, string(source))
	if err != nil {
		fail(err)
	}
	generated, err := concept.GenerateMachineBridgeCodecs(module)
	if err != nil {
		fail(err)
	}
	outputs := map[string][]byte{"internal/concept/machineir_bridge_codec_generated.go": generated.Go, "libraries/Standard/Backend/BridgeCodec.concept": generated.Concept, "docs/design/EVT2-MACHINEIR-BRIDGE.schema.json": generated.Metadata}
	for name, data := range outputs {
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			fail(err)
		}
	}
	fmt.Println(generated.String())
	scalarPath := filepath.Join(root, "libraries/Standard/Backend/BridgeSchema4.concept")
	scalarSource, err := os.ReadFile(scalarPath)
	if err != nil {
		fail(err)
	}
	scalarModule, err := concept.ParseWithBuiltSemanticModuleRoots(scalarPath, string(scalarSource), []string{filepath.Join(root, "libraries")})
	if err != nil {
		fail(err)
	}
	scalar, err := concept.GenerateMachineBridgeCodecs(scalarModule)
	if err != nil {
		fail(err)
	}
	for name, data := range map[string][]byte{"internal/concept/machineir_bridge4_codec_generated.go": scalar.Go, "libraries/Standard/Backend/BridgeCodec4.concept": scalar.Concept, "docs/design/EVT2-MACHINEIR-BRIDGE4.schema.json": scalar.Metadata} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			fail(err)
		}
	}
	fmt.Println(scalar.String())
	objectPath := filepath.Join(root, "libraries/Standard/Backend/BridgeSchema5.concept")
	objectSource, err := os.ReadFile(objectPath)
	if err != nil {
		fail(err)
	}
	objectModule, err := concept.ParseWithBuiltSemanticModuleRoots(objectPath, string(objectSource), []string{filepath.Join(root, "libraries")})
	if err != nil {
		fail(err)
	}
	object, err := concept.GenerateMachineBridgeCodecs(objectModule)
	if err != nil {
		fail(err)
	}
	for name, data := range map[string][]byte{"internal/concept/machineir_bridge5_codec_generated.go": object.Go, "libraries/Standard/Backend/BridgeCodec5.concept": object.Concept, "docs/design/EVT2-MACHINEIR-BRIDGE5.schema.json": object.Metadata} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			fail(err)
		}
	}
	fmt.Println(object.String())
	borrowPath := filepath.Join(root, "libraries/Standard/Backend/BridgeSchema6.concept")
	borrowSource, err := os.ReadFile(borrowPath)
	if err != nil {
		fail(err)
	}
	borrowModule, err := concept.ParseWithBuiltSemanticModuleRoots(borrowPath, string(borrowSource), []string{filepath.Join(root, "libraries")})
	if err != nil {
		fail(err)
	}
	borrow, err := concept.GenerateMachineBridgeCodecs(borrowModule)
	if err != nil {
		fail(err)
	}
	for name, data := range map[string][]byte{"internal/concept/machineir_bridge6_codec_generated.go": borrow.Go, "libraries/Standard/Backend/BridgeCodec6.concept": borrow.Concept, "docs/design/EVT2-MACHINEIR-BRIDGE6.schema.json": borrow.Metadata} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			fail(err)
		}
	}
	fmt.Println(borrow.String())
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
