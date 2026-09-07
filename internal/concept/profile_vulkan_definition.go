package concept

const evt1ActuationOutcomeTypeName = "ActuationOutcome"

func evt1BuiltinActuationOutcomeEnum() EnumDecl {
	return EnumDecl{
		Name: evt1ActuationOutcomeTypeName,
		Variants: []VariantDecl{
			{Name: "Completed", Tag: 0},
			{Name: "Failed", Tag: 1},
			{Name: "NoBatch", Tag: 2},
			{Name: "AlreadyConsumed", Tag: 3},
		},
	}
}

func evt1NewVulkanProfileDefinition() ProfileDefinition {
	builtinTypes := evt1CoreBuiltinDefinitions()
	builtinTypes["PipelineLayout"] = BuiltinTypeDefinition{
		Name:         "PipelineLayout",
		CType:        "VkPipelineLayout",
		NeedsHeaders: []string{"<vulkan/vulkan.h>"},
	}
	builtinTypes["Pipeline"] = BuiltinTypeDefinition{
		Name:         "Pipeline",
		CType:        "VkPipeline",
		NeedsHeaders: []string{"<vulkan/vulkan.h>"},
	}
	builtinTypes["VkBuffer"] = BuiltinTypeDefinition{
		Name:         "VkBuffer",
		CType:        "VkBuffer",
		NeedsHeaders: []string{"<vulkan/vulkan.h>"},
	}
	builtinTypes["VkCommandPool"] = BuiltinTypeDefinition{
		Name:         "VkCommandPool",
		CType:        "VkCommandPool",
		NeedsHeaders: []string{"<vulkan/vulkan.h>"},
	}
	builtinTypes["VulkanError"] = BuiltinTypeDefinition{
		Name:         "VulkanError",
		CType:        "concept_vulkan_error",
		CDeclaration: "typedef struct concept_vulkan_error {\n  int Code;\n} concept_vulkan_error;\n",
		Fields: map[string]Type{
			"Code": {Name: "int", Kind: TypeBuiltin},
		},
	}
	return ProfileDefinition{
		Name:         "Vulkan",
		BuiltinTypes: builtinTypes,
		BuiltinEnums: []EnumDecl{
			evt1BuiltinAutomataDispatchOutcomeEnum(),
			evt1BuiltinActuationOutcomeEnum(),
		},
		AdmittedImports: map[string]struct{}{
			"Prometheus.Vulkan": {},
		},
		AllowDomainImports: true,
		AllowEffects:       true,
		AllowActuators:     true,
	}
}

var vulkanProfileDefinition = evt1NewVulkanProfileDefinition()
