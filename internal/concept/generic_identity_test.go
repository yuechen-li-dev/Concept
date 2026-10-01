package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const r9aGenericIdentitySource = `module Closure.Identity;
profile Core;
record struct Double { int marker; }
using Integer = int;
template <typename T> struct Buffer { owned T value; };
template <typename T, usize N> struct Fixed { T<array>[N] values; };
template <typename T> int Read(ref const Buffer<T> buffer) { return 7; }
Buffer<int> Make() { return Buffer<int>{1}; }
struct Cases {
    Buffer<int> integer;
    Buffer<double> floating;
    Buffer<Double> nominal;
    Buffer<Buffer<int>> nested;
    Buffer<Integer> alias;
    Fixed<int, 3> fixed;
}
`

func TestR9aStructuredGenericArtifactIdentity(t *testing.T) {
	first := buildSemanticArtifact(t, "Closure/Identity.concept", r9aGenericIdentitySource, nil)
	artifact, module, err := LoadSemanticModuleArtifact(first)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.GenericApplicationSchema != GenericApplicationSchema || len(artifact.GenericApplications) != 5 {
		t.Fatalf("missing structured applications: %#v", artifact.GenericApplications)
	}
	outputs, err := Generate(module, []byte(r9aGenericIdentitySource))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "identity_harness.c", "#include \"closure.generated.h\"\nint main(void) { return 0; }\n")
	for _, decl := range module.Structs {
		if decl.Application == nil {
			continue
		}
		if decl.Application.Declaration.Module != "Closure.Identity" {
			t.Fatalf("lost declaration owner: %#v", decl.Application)
		}
		if decl.Name == "Buffer<Buffer<int>>" && decl.Application.Arguments[0].Type.Application == nil {
			t.Fatal("nested application was flattened")
		}
		if decl.Name == "Fixed<int, 3>" {
			value := decl.Application.Arguments[1]
			if value.Kind != "value" || value.Integer != 3 || value.Type.Name != "usize" {
				t.Fatalf("untyped value argument: %#v", value)
			}
		}
	}
	consumer := `module Closure.Consumer;
profile Core;
import Closure.Identity;
int Main() { Buffer<int> value = Make(); return Read(ref const value); }
`
	consumed, err := ParseWithSemanticModules("Consumer.concept", consumer, map[string][]byte{"Closure.Identity": first})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(consumed, []byte(consumer)); err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "Closure/Identity.concept", r9aGenericIdentitySource, nil); !bytes.Equal(got, first) {
			t.Fatalf("structured artifact changed on run %d", run)
		}
	}
}

func TestR9aGenericInferenceIgnoresNominalSpelling(t *testing.T) {
	module, err := Parse("Identity.concept", r9aGenericIdentitySource)
	if err != nil {
		t.Fatal(err)
	}
	var metadata *GenericApplication
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	decl := env.structs["Buffer<int>"]
	metadata = decl.Application
	if metadata == nil {
		t.Fatal("Parse did not retain application metadata")
	}
	delete(env.structs, decl.Name)
	decl.Name = "opaque_closed_symbol"
	env.structs[decl.Name] = decl
	env.genericTypeKeys[evt1GenericApplicationKey(metadata)] = decl.Name
	actual := Type{Name: "opaque_closed_symbol", Kind: TypeStruct, Ownership: "ref", Const: true}
	args, ok := evt1InferClosedTemplateArguments(env, env.templates["Read"], []Type{actual})
	if !ok || len(args) != 1 || args[0].Name != "int" {
		t.Fatalf("inference still depends on spelling: %v %#v", ok, args)
	}
	left := Type{Name: "first", Kind: TypeStruct, Application: metadata}
	right := Type{Name: "second", Kind: TypeStruct, Application: metadata}
	if !left.Equal(right) || evt1GenericApplicationKey(metadata) != evt1GenericApplicationKey(right.Application) {
		t.Fatal("display spelling leaked into identity")
	}
	different := *metadata
	different.Arguments = []GenericArgument{{Kind: "type", Type: Type{Name: "double", Kind: TypeBuiltin}}}
	right.Name, right.Application = left.Name, &different
	if evt1SemanticTypeEqual(nil, left, right) {
		t.Fatal("same display name collapsed distinct applications")
	}
}

func TestR9aOldOrInconsistentGenericArtifactsRejected(t *testing.T) {
	body := buildSemanticArtifact(t, "Closure/Identity.concept", r9aGenericIdentitySource, nil)
	var artifact SemanticModuleArtifact
	if err := json.Unmarshal(body, &artifact); err != nil {
		t.Fatal(err)
	}
	artifact.GenericApplicationSchema = ""
	old, _ := json.Marshal(artifact)
	if _, _, err := LoadSemanticModuleArtifact(old); err == nil || !strings.Contains(err.Error(), "MODULE_GENERIC_SCHEMA_STALE") {
		t.Fatalf("old payload silently admitted: %v", err)
	}
	artifact.GenericApplicationSchema = GenericApplicationSchema
	artifact.GenericApplications[0].NominalName = "incorrect"
	artifact.ContentSHA256, _ = semanticArtifactHash(artifact)
	corrupt, _ := json.Marshal(artifact)
	if _, _, err := LoadSemanticModuleArtifact(corrupt); err == nil || !strings.Contains(err.Error(), "MODULE_GENERIC_APPLICATION_MISMATCH") {
		t.Fatalf("metadata mismatch admitted: %v", err)
	}
}
