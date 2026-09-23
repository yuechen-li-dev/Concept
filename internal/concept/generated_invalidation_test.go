package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestGeneratedArtifactChangesWithReflectedInput(t *testing.T) {
	source := `module GeneratedInvalidation; profile Core;
struct Item { [[selected]] int value; }
generator <typename T> DeriveRead
int Read(ref const T item) {
    int result = 0;
    foreach (FieldInfo field in Fields<T>(selected)) { result = result + item.field; }
    return result;
}
derive DeriveRead reflect<Item>;
`
	changed := strings.Replace(source, "int value;", "int other;", 1)
	first, err := CompileSemanticModule("invalidation.concept", source, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileSemanticModule("invalidation.concept", changed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("reflected input change left artifact unchanged")
	}
	_, firstModule, err := LoadSemanticModuleArtifact(first)
	if err != nil {
		t.Fatal(err)
	}
	_, secondModule, err := LoadSemanticModuleArtifact(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstModule.Functions[0].Generated.Identity == secondModule.Functions[0].Generated.Identity {
		t.Fatal("reflected input change left generated identity unchanged")
	}
}
