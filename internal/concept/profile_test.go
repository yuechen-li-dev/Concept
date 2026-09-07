package concept

import (
	"strings"
	"testing"
)

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
