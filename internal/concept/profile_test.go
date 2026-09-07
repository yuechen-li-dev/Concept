package concept

import (
	"strings"
	"testing"
)

func TestProfileDefinitionsOwnBuiltinAdmissions(t *testing.T) {
	core, ok := evt1ProfileDefinition("Core")
	if !ok {
		t.Fatal("Core profile is not registered")
	}
	vulkan, ok := evt1ProfileDefinition("Vulkan")
	if !ok {
		t.Fatal("Vulkan profile is not registered")
	}
	for _, name := range []string{"PipelineLayout", "Pipeline", "VulkanError", "VkBuffer", "VkCommandPool"} {
		if _, admitted := core.BuiltinTypes[name]; admitted {
			t.Fatalf("Core profile admits Vulkan type %s", name)
		}
		if _, admitted := vulkan.BuiltinTypes[name]; !admitted {
			t.Fatalf("Vulkan profile does not own type %s", name)
		}
	}
	if _, ok := vulkan.AdmittedImports["Prometheus.Vulkan"]; !ok {
		t.Fatal("Vulkan profile does not own the Prometheus.Vulkan import admission")
	}
	if core.AllowEffects || core.AllowActuators || core.AllowDomainImports {
		t.Fatal("Core profile admits domain semantics")
	}
	if !vulkan.AllowEffects || !vulkan.AllowActuators || !vulkan.AllowDomainImports {
		t.Fatal("Vulkan profile admissions are incomplete")
	}
	for name := range vulkan.BuiltinTypes {
		if strings.Contains(name, "Prometheus") {
			t.Fatalf("Prometheus application type leaked into compiler builtins: %s", name)
		}
	}
}

func TestVulkanProfileBuiltinLoweringUsesRegistration(t *testing.T) {
	source := `profile Vulkan;
import Prometheus.Vulkan;

struct Handles
{
    PipelineLayout layout;
    Pipeline pipeline;
    VkBuffer buffer;
    VkCommandPool pool;
    VulkanError error;
};

int Read(VulkanError error)
{
    return error.Code;
}
`
	module, err := Parse("registered-vulkan.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var header string
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.h") {
			header = string(body)
		}
	}
	for _, required := range []string{"#include <vulkan/vulkan.h>", "VkPipelineLayout", "VkPipeline", "VkBuffer", "VkCommandPool", "concept_vulkan_error"} {
		if !strings.Contains(header, required) {
			t.Fatalf("registered Vulkan header is missing %q", required)
		}
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
