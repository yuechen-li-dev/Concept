package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func r7pGoldenSource(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "libraries"}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestR7pGoldenArtifactsWithoutDependencySource(t *testing.T) {
	aerospace := r7pGoldenSource(t, "Golden", "Aerospace", "FlightTelemetry.concept")
	aerospaceArtifact, err := CompileSemanticModule("Golden/Aerospace/FlightTelemetry.concept", aerospace, nil)
	if err != nil {
		t.Fatal(err)
	}
	stores := r7pGoldenSource(t, "Standard", "Collection", "Stores.concept")
	storesArtifact, err := CompileSemanticModule("Standard/Collection/Stores.concept", stores, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string][]byte{"Standard.Collection.Stores": storesArtifact, "Golden.Aerospace.FlightTelemetry": aerospaceArtifact}

	for _, golden := range []struct {
		module, path, consumer string
	}{
		{"Golden.Game.Agents", "Game/Agents.concept", `module Consumer; profile Core; import Golden.Game.Agents; int Main() { World world = World{DenseStore<Agent, 64>{Uninitialized()}, 0}; Spawn(ref world, 1, 2)!; Tick(ref world); return world.tick; } void Proof() { Assert.Concept<NoAllocation>(Tick, "imported generic store traversal stays inline"); }`},
		{"Golden.Game.WorldOps", "Game/WorldOps.concept", `module Consumer; profile Core; import Golden.Game.WorldOps; int Main() { World world = World{DenseStore<Agent, 64>{Uninitialized()}, 0}; Spawn(ref world, 1, 2)!; PositionFrame frame = PositionFrame{[0 ...], [0 ...]}; return ProjectPositions(ref const world, ref frame); }`},
		{"Golden.Cad.Mesh", "Cad/Mesh.concept", `module Consumer; profile Core; import Golden.Cad.Mesh; int Main() { Mesh mesh = NewMesh(); float<m> x = 2.0; AddVertex(ref mesh, x, x)!; return Count<Vertex, 64>(ref const mesh.vertices); }`},
		{"Golden.Storage.PageCache", "Storage/PageCache.concept", `module Consumer; profile Core; import Golden.Storage.PageCache; int Main() { PageCache cache = NewCache(); Install(ref cache, 7)!; return ActiveSlots(ref const cache); }`},
	} {
		body := r7pGoldenSource(t, "Golden", filepath.FromSlash(golden.path))
		artifact, err := CompileSemanticModule(golden.path, body, artifacts)
		if err != nil {
			t.Fatalf("%s artifact: %v", golden.module, err)
		}
		consumerArtifacts := make(map[string][]byte, len(artifacts)+1)
		for name, value := range artifacts {
			consumerArtifacts[name] = value
		}
		consumerArtifacts[golden.module] = artifact
		module, err := ParseWithSemanticModules("Consumer.concept", golden.consumer, consumerArtifacts)
		if err != nil {
			t.Fatalf("%s consumer: %v", golden.module, err)
		}
		outputs, err := Generate(module, []byte(golden.consumer))
		if err != nil {
			t.Fatalf("%s generated C: %v", golden.module, err)
		}
		if strings.Contains(string(outputs["consumer.generated.c"]), "unresolved_call") {
			t.Fatalf("%s generated unresolved call", golden.module)
		}
		artifacts[golden.module] = artifact
	}
 const aerospaceConsumer = `module Consumer; profile Core; import Golden.Aerospace.FlightTelemetry; int Main() { FlightController controller = FlightController{FlightMode::Preflight, [FlightSample{0.0, 0.0, false} ... 16], 0, 0, 0, 0}; AcceptNative(ref controller, NativeSample{100.0, 2.0, 1})!; return controller.accepted; }`
	if _, err := ParseWithSemanticModules("Consumer.concept", aerospaceConsumer, artifacts); err != nil {
		t.Fatalf("aerospace artifact consumer: %v", err)
	}
}
