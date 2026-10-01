package concept

// The Vulkan profile adds no builtins or semantics to Core. It automates
// setup instead (vulkan_profile.go): `import Vulkan;` is implied, and
// `concept test` links the Vulkan runtime.
func evt1NewVulkanProfileDefinition() ProfileDefinition {
	return ProfileDefinition{
		Name:         "Vulkan",
		BuiltinTypes: evt1CoreBuiltinDefinitions(),
		BuiltinEnums: []EnumDecl{
			evt1BuiltinStepOutcomeEnum(),
			evt1BuiltinNumericCastErrorEnum(),
			evt1BuiltinTypeShapeEnum(),
		},
		AdmittedImports:    map[string]struct{}{},
		AllowDomainImports: true,
	}
}

var vulkanProfileDefinition = evt1NewVulkanProfileDefinition()
