package concept

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// `profile Vulkan;` is Core plus the setup a Vulkan program would otherwise
// write by hand:
//   - `import Vulkan;` is implied (libraries/Vulkan/concept/Vulkan.concept);
//   - `concept test` compiles and links the Vulkan runtime behind
//     libraries/Vulkan/native/concept_vulkan.h. The default is the GPU-free
//     test device; CONCEPT_VULKAN_RUNTIME=device selects the loader-backed
//     runtime, found through VULKAN_SDK when it is set.

const evt1VulkanProfileName = "Vulkan"

// VulkanLibraryRoot returns libraries/Vulkan for a source inside the
// checkout, honoring CONCEPT_VULKAN_LIBRARY, or "" when there is none.
func VulkanLibraryRoot(sourcePath string) string {
	if configured := os.Getenv("CONCEPT_VULKAN_LIBRARY"); configured != "" {
		return configured
	}
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(abs); ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "libraries", "Vulkan")
		if _, err := os.Stat(filepath.Join(candidate, "native", "concept_vulkan.h")); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
	}
}

// VulkanModuleRoot is the semantic module root that resolves `Vulkan`.
func VulkanModuleRoot(sourcePath string) string {
	root := VulkanLibraryRoot(sourcePath)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "concept")
}

type evt1VulkanRuntime struct {
	Sources []string
	CFlags  []string
	LDFlags []string
	Name    string
}

func evt1SelectVulkanRuntime(sourcePath string) (evt1VulkanRuntime, error) {
	root := VulkanLibraryRoot(sourcePath)
	if root == "" {
		return evt1VulkanRuntime{}, fmt.Errorf("VULKAN_LIBRARY_MISSING: profile Vulkan needs libraries/Vulkan (or CONCEPT_VULKAN_LIBRARY)")
	}
	native := filepath.Join(root, "native")
	out := evt1VulkanRuntime{CFlags: []string{"-I", native}}
	switch mode := os.Getenv("CONCEPT_VULKAN_RUNTIME"); mode {
	case "", "test":
		out.Name = "test-device"
		out.Sources = []string{filepath.Join(native, "test_device.c")}
	case "device":
		out.Name = "device"
		out.Sources = []string{filepath.Join(native, "device_runtime.c")}
		library := "vulkan"
		if runtime.GOOS == "windows" {
			library = "vulkan-1"
		}
		if sdk := os.Getenv("VULKAN_SDK"); sdk != "" {
			include := filepath.Join(sdk, "Include")
			libDir := filepath.Join(sdk, "Lib")
			if runtime.GOOS != "windows" {
				include = filepath.Join(sdk, "include")
				libDir = filepath.Join(sdk, "lib")
			}
			out.CFlags = append(out.CFlags, "-I", include)
			out.LDFlags = append(out.LDFlags, "-L"+libDir)
		}
		out.LDFlags = append(out.LDFlags, "-l"+library)
	default:
		return evt1VulkanRuntime{}, fmt.Errorf("VULKAN_RUNTIME_UNKNOWN: CONCEPT_VULKAN_RUNTIME=%s; use test or device", mode)
	}
	return out, nil
}

var evt1VulkanArtifactCache = struct {
	sync.Mutex
	entries map[string]evt1VulkanArtifacts
}{entries: map[string]evt1VulkanArtifacts{}}

type evt1VulkanArtifacts struct {
	artifacts map[string][]byte
	identity  NativeABIIdentity
}

// evt1VulkanSemanticArtifacts builds the Vulkan library module with its
// measured native ABI evidence, once per library root.
func evt1VulkanSemanticArtifacts(sourcePath string) (map[string][]byte, NativeABIIdentity, error) {
	root := VulkanLibraryRoot(sourcePath)
	if root == "" {
		return nil, NativeABIIdentity{}, fmt.Errorf("VULKAN_LIBRARY_MISSING: profile Vulkan needs libraries/Vulkan (or CONCEPT_VULKAN_LIBRARY)")
	}
	evt1VulkanArtifactCache.Lock()
	defer evt1VulkanArtifactCache.Unlock()
	if cached, ok := evt1VulkanArtifactCache.entries[root]; ok {
		return cached.artifacts, cached.identity, nil
	}
	project, err := LoadNativeProject(root)
	if err != nil {
		return nil, NativeABIIdentity{}, err
	}
	artifacts, identity, err := BuildNativeCompanionArtifacts(project)
	if err != nil {
		return nil, NativeABIIdentity{}, err
	}
	evt1VulkanArtifactCache.entries[root] = evt1VulkanArtifacts{artifacts, identity}
	return artifacts, identity, nil
}

// ParseWithModuleRootsForProfile parses a source whose imports are built
// from `roots`; for `profile Vulkan;` it also supplies the Vulkan library
// and its native ABI evidence.
func ParseWithModuleRootsForProfile(path, source string, roots []string) (Module, error) {
	syntax, err := parseSyntaxModule(path, source)
	if err != nil {
		return Module{}, err
	}
	if syntax.Profile != evt1VulkanProfileName {
		return ParseWithBuiltSemanticModuleRoots(path, source, roots)
	}
	vulkan, identity, err := evt1VulkanSemanticArtifacts(path)
	if err != nil {
		return Module{}, err
	}
	artifacts, err := buildSemanticModuleArtifactsFromSources(roots, syntax.Imports, vulkan, &identity)
	if err != nil {
		return Module{}, err
	}
	return ParseWithSemanticModulesForNative(path, source, artifacts, identity)
}

// UsesVulkanProfile reports whether a source declares `profile Vulkan;`.
func UsesVulkanProfile(path, source string) bool {
	syntax, err := parseSyntaxModule(path, source)
	return err == nil && syntax.Profile == evt1VulkanProfileName
}
