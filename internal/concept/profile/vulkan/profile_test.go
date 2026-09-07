package vulkan

import "testing"

func TestProfileBoundary(t *testing.T) {
	if _, err := Parse("core.concept", "profile Core;\nint Read() { return 0; }\n"); err == nil {
		t.Fatal("Vulkan driver accepted a Core module")
	}
	module, err := Parse("vulkan.concept", "profile Vulkan;\nint Read() { return 0; }\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte("profile Vulkan;\nint Read() { return 0; }\n")); err != nil {
		t.Fatal(err)
	}
}
