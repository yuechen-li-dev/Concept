package concept

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGoldenArtifactAndCDeterminism100(t *testing.T) {
	if testing.Short() {
		t.Skip("100-run golden determinism is a full burn-in gate")
	}
	stores := r7pGoldenSource(t, "Standard", "Collection", "Stores.concept")
	storesArtifact, err := CompileSemanticModule("Standard/Collection/Stores.concept", stores, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct{ module, path string }{
		{"Golden.Game.Agents", "Game/Agents.concept"},
		{"Golden.Aerospace.FlightTelemetry", "Aerospace/FlightTelemetry.concept"},
	} {
		t.Run(fixture.module, func(t *testing.T) {
			source := r7pGoldenSource(t, "Golden", fixture.path)
			dependencies := map[string][]byte{"Standard.Collection.Stores": storesArtifact}
			var firstArtifact []byte
			var firstOutputs map[string][]byte
			for run := 0; run < 100; run++ {
				artifact, err := CompileSemanticModule(fixture.path, source, dependencies)
				if err != nil {
					t.Fatal(err)
				}
				module, err := ParseWithSemanticModules(fixture.path, source, dependencies)
				if err != nil {
					t.Fatal(err)
				}
				outputs, err := Generate(module, []byte(source))
				if err != nil {
					t.Fatal(err)
				}
				if run == 0 {
					firstArtifact, firstOutputs = artifact, outputs
					continue
				}
				if !bytes.Equal(artifact, firstArtifact) || len(outputs) != len(firstOutputs) {
					t.Fatal("semantic artifact or output set changed")
				}
				for name, body := range firstOutputs {
					if !bytes.Equal(body, outputs[name]) {
						t.Fatal(fmt.Sprintf("%s changed on run %d", name, run+1))
					}
				}
			}
		})
	}
}
