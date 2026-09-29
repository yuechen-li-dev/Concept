package concept

import (
	"strings"
	"testing"
)

func TestVulkanProfileAddsNoBuiltins(t *testing.T) {
	core, ok := evt1ProfileDefinition("Core")
	if !ok {
		t.Fatal("Core profile is not registered")
	}
	vulkan, ok := evt1ProfileDefinition("Vulkan")
	if !ok {
		t.Fatal("Vulkan profile is not registered")
	}
	if len(vulkan.BuiltinTypes) != len(core.BuiltinTypes) {
		t.Fatalf("Vulkan profile adds builtins beyond Core: %d vs %d", len(vulkan.BuiltinTypes), len(core.BuiltinTypes))
	}
	for name := range vulkan.BuiltinTypes {
		if _, ok := core.BuiltinTypes[name]; !ok {
			t.Fatalf("Vulkan-only builtin %s", name)
		}
	}
	if core.AllowDomainImports {
		t.Fatal("Core profile admits domain imports")
	}
}

func TestCoreProfileCompilesWithoutVulkanAdmissions(t *testing.T) {
	source := `profile Core;

enum Status
{
    Empty,
    Ready(int value)
}

int Read(Status status)
{
    return match (status)
    {
        Status::Empty => 0,
		Status::Ready(value) => value,
    };
}
`
	module, err := Parse("core.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if module.Profile != "Core" {
		t.Fatalf("profile = %q, want Core", module.Profile)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range outputs {
		if strings.Contains(string(body), "vulkan") || strings.Contains(string(body), "Vulkan") {
			t.Fatalf("core artifact %s contains Vulkan coupling", name)
		}
	}
}

func TestCoreProfileRejectsVulkanDomainSemantics(t *testing.T) {
	for _, source := range []string{
		"profile Core;\nimport Prometheus.Vulkan;\nint Read() { return 0; }\n",
		"profile Core;\neffect Mark(int value);\nint Read() { return 0; }\n",
		"profile Core;\nPipeline Make();\n",
	} {
		if _, err := Parse("core.concept", source); err == nil {
			t.Fatalf("expected profile boundary diagnostic for %q", source)
		}
	}
}
