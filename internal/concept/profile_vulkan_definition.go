package concept

// The Vulkan profile no longer adds builtins: Vulkan handles are
// `extern "C" handle` declarations and VkResult mapping lives in
// libraries/Vulkan. What remains is the domain-import admission, pending
// the VK9 decision on the profile itself.
func evt1NewVulkanProfileDefinition() ProfileDefinition {
	return ProfileDefinition{
		Name:         "Vulkan",
		BuiltinTypes: evt1CoreBuiltinDefinitions(),
		BuiltinEnums: []EnumDecl{
			evt1BuiltinStepOutcomeEnum(),
			evt1BuiltinNumericCastErrorEnum(),
		},
		AdmittedImports:    map[string]struct{}{},
		AllowDomainImports: true,
	}
}

var vulkanProfileDefinition = evt1NewVulkanProfileDefinition()
