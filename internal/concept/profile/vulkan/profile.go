// Package vulkan is the explicit Vulkan-profile consumer of the Concept
// compiler core. It owns profile admission at API boundaries; the Concept
// compiler remains independently usable with `profile Core;`.
package vulkan

import (
	"fmt"

	"github.com/yuechen-li-dev/Concept/internal/concept"
)

const Name = "Vulkan"

// Parse accepts only modules that explicitly select the Vulkan profile.
func Parse(path, source string) (concept.Module, error) {
	module, err := concept.Parse(path, source)
	if err != nil {
		return concept.Module{}, err
	}
	if module.Profile != Name {
		return concept.Module{}, fmt.Errorf("Vulkan profile driver cannot compile profile %s", module.Profile)
	}
	return module, nil
}

// Generate lowers a previously admitted Vulkan module through the shared
// deterministic MIR and strict-C11 backend.
func Generate(module concept.Module, source []byte) (concept.Outputs, error) {
	if module.Profile != Name {
		return nil, fmt.Errorf("Vulkan profile driver cannot generate profile %s", module.Profile)
	}
	return concept.Generate(module, source)
}
