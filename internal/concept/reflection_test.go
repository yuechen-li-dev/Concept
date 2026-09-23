package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExplicitStructuralReflection(t *testing.T) {
	source := `module Sample;
profile Core;
struct SystemMemory {}
[[reflect]] struct Node {
    [[edge]] int next;
    private:
    int secret;
};
enum Message { Empty, Value(int number), }
table<3> Samples { int amount; }
template <typename T> struct Box { T value; };
reflect<Node>;
reflect<Message>;
reflect<Samples>;
reflect<Box<int>>;
reflect<int[3]>;
reflect<Span<int>>;
reflect<Storage<int>>;
reflect<Address<SystemMemory>>;
`
	module, err := Parse("sample.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.ReflectionResults) != 8 {
		t.Fatalf("got %d reflections", len(module.ReflectionResults))
	}
	node := module.ReflectionResults[0]
	if node.Kind != "struct" || node.Module != "Sample" || len(node.Fields) != 2 || node.Fields[0].Name != "next" || node.Fields[0].Index != 0 || node.Fields[1].Visibility != "private" || len(node.Fields[0].Attributes) != 1 {
		t.Fatalf("node reflection: %+v", node)
	}
	enum := module.ReflectionResults[1]
	if enum.Kind != "enum" || len(enum.EnumCases) != 2 || enum.EnumCases[1].Tag != 1 || enum.EnumCases[1].Payload[0].Type.Name != "int" {
		t.Fatalf("enum reflection: %+v", enum)
	}
	table := module.ReflectionResults[2]
	if table.Kind != "table" || table.TableCardinality == nil || *table.TableCardinality != 3 || !table.Fields[0].Type.Column {
		t.Fatalf("table reflection: %+v", table)
	}
	box := module.ReflectionResults[3]
	if len(box.Fields) != 1 || box.Fields[0].Type.Name != "int" || len(box.GenericArguments) != 1 || box.GenericArguments[0].Name != "int" {
		t.Fatalf("generic reflection: %+v", box)
	}
	array := module.ReflectionResults[4]
	if array.Extent == nil || *array.Extent != 3 || array.ElementType == nil || array.ElementType.Name != "int" {
		t.Fatalf("array reflection: %+v", array)
	}
	span := module.ReflectionResults[5]
	if span.Kind != string(TypeSpan) || len(span.GenericArguments) != 1 || span.GenericArguments[0].Name != "int" {
		t.Fatalf("span reflection: %+v", span)
	}
	if module.ReflectionResults[6].Kind != string(TypeTypedStorage) || module.ReflectionResults[7].Kind != string(TypeAddress) {
		t.Fatalf("low-level kinds collapsed: %s, %s", module.ReflectionResults[6].Kind, module.ReflectionResults[7].Kind)
	}
}

func TestArtifactOnlyReflectionPermission(t *testing.T) {
	sourceA := `module A;
profile Core;
[[reflect]] struct Allowed { private: int secret; };
[[reflect]] template <typename T> struct Box { T value; };
struct Closed { private: int secret; };
`
	artifact, err := CompileSemanticModule("A.concept", sourceA, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifactMetadata, _, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(artifactMetadata.Exports.ReflectableTypes, ",") != "Allowed,Box" {
		t.Fatalf("reflectable export summary: %+v", artifactMetadata.Exports.ReflectableTypes)
	}
	sourceB := `module B;
profile Core;
import A;
reflect<Allowed>;
reflect<Box<int>>;
`
	module, err := ParseWithSemanticModules("B.concept", sourceB, map[string][]byte{"A": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if len(module.ReflectionResults) != 2 || module.ReflectionResults[0].Fields[0].Name != "secret" || module.ReflectionResults[1].Fields[0].Type.Name != "int" {
		t.Fatalf("artifact reflection: %+v", module.ReflectionResults)
	}
	_, err = ParseWithSemanticModules("B.concept", strings.Replace(sourceB, "Allowed", "Closed", 1), map[string][]byte{"A": artifact})
	if err == nil || !strings.Contains(err.Error(), "REFLECT_PERMISSION_REQUIRED") {
		t.Fatalf("expected permission diagnostic, got %v", err)
	}
}

func TestArtifactIdentityOwnsReflectionBoundary(t *testing.T) {
	artifactBody, err := CompileSemanticModule("A.concept", `module A; profile Core; struct Closed { private: int secret; };`, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact, payload, err := LoadSemanticModuleArtifact(artifactBody)
	if err != nil {
		t.Fatal(err)
	}
	for i := range payload.Structs {
		payload.Structs[i].Module = ""
	}
	artifact.SemanticPayload, err = encodeSemanticModule(payload)
	if err != nil {
		t.Fatal(err)
	}
	artifact.ContentSHA256, err = semanticArtifactHash(artifact)
	if err != nil {
		t.Fatal(err)
	}
	legacyBody, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseWithSemanticModules("B.concept", `module B; profile Core; import A; reflect<Closed>;`, map[string][]byte{"A": legacyBody})
	if err == nil || !strings.Contains(err.Error(), "REFLECT_PERMISSION_REQUIRED") {
		t.Fatalf("older artifact bypassed reflection boundary: %v", err)
	}
}

func TestReflectionErasesBeforeRuntimeGeneration(t *testing.T) {
	base := `module Sample;
profile Core;
struct Node { int value; };
int main() { return 0; }
`
	reflected := base + "reflect<Node>;\n"
	plainModule, err := Parse("sample.concept", base)
	if err != nil {
		t.Fatal(err)
	}
	reflectedModule, err := Parse("sample.concept", reflected)
	if err != nil {
		t.Fatal(err)
	}
	plainOutput, err := Generate(plainModule, []byte(base))
	if err != nil {
		t.Fatal(err)
	}
	reflectedOutput, err := Generate(reflectedModule, []byte(reflected))
	if err != nil {
		t.Fatal(err)
	}
	for name, plain := range plainOutput {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") || strings.HasSuffix(name, ".mir.json") {
			if !bytes.Equal(plain, reflectedOutput[name]) {
				t.Fatalf("runtime output %s changed by compile-time reflection", name)
			}
		}
	}
}

func TestReflectionArtifactAndMetadataDeterministic100(t *testing.T) {
	source := `module Sample;
profile Core;
[[reflect]] struct Node { int value; int next; };
reflect<Node>;
`
	var firstArtifact, firstMetadata []byte
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("Sample.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			_, exportedModule, err := LoadSemanticModuleArtifact(artifact)
			if err != nil {
				t.Fatal(err)
			}
			if len(exportedModule.ReflectionResults) != 0 {
				t.Fatal("reflection query result leaked into module artifact")
			}
		}
		module, err := Parse("Sample.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		metadata, err := json.Marshal(module.ReflectionResults)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			firstArtifact, firstMetadata = artifact, metadata
		} else if !bytes.Equal(firstArtifact, artifact) || !bytes.Equal(firstMetadata, metadata) {
			t.Fatalf("reflection changed on run %d", run)
		}
	}
}
