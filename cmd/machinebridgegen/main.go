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
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
