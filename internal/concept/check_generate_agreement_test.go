package concept

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// `concept check` is Parse alone. Every program that code generation rejects
// statically must already be rejected there, or check accepts programs the
// compiler cannot build.
func TestCheckRejectsEveryStaticInvalidCorpusFile(t *testing.T) {
	t.Parallel()
	_, manifest := loadSemanticCorpusManifest(t)
	root := filepath.Join("..", "..", "language", "evt1")
	for _, subsystem := range manifest.Subsystems {
		runtimeNegative := map[string]bool{}
		for _, name := range subsystem.RuntimeNegative {
			runtimeNegative[name] = true
		}
		invalid, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(subsystem.Path), "invalid", "*.concept"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range invalid {
			if runtimeNegative[filepath.Base(path)] {
				continue
			}
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(filepath.ToSlash(path), string(source)); err == nil {
				t.Errorf("concept check accepts static-invalid corpus file %s", path)
			}
		}
	}
}

func TestCheckValidatesGenericInstanceStructs(t *testing.T) {
	t.Parallel()
	source := `module Instances;
profile Core;

class Resource
{
public:
    int id;
};

void Drop(owned Resource resource)
{
}

template <typename T>
struct Box
{
    T value;
}

int Use()
{
    owned Resource resource = Resource{7};
    owned Box<Resource> box = Box<Resource>{move resource};
    return box.value.id;
}
`
	_, err := Parse("instances.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4653" {
		t.Fatalf("check must reject the droppable field of Box<Resource>, got %v", err)
	}
	if diagnostic.Span.Line != 17 {
		t.Fatalf("diagnostic should point at the template field, got line %d", diagnostic.Span.Line)
	}
	want := "field Box<Resource>.value holds Resource, which has a Drop; declare it `owned T value;` in Box"
	if diagnostic.Message != want {
		t.Fatalf("message:\n got %s\nwant %s", diagnostic.Message, want)
	}
}

func TestCheckAcceptsOwnedGenericFields(t *testing.T) {
	t.Parallel()
	source := `module Instances;
profile Core;

class Resource
{
public:
    int id;
};

void Drop(owned Resource resource)
{
}

template <typename T>
struct Box
{
    owned T value;
}

int Use()
{
    owned Resource resource = Resource{7};
    owned Box<Resource> box = Box<Resource>{move resource};
    owned Box<int> count = Box<int>{3};
    return box.value.id + count.value;
}
`
	module, err := Parse("instances.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}
