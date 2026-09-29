package concept

func evt1NewVulkanProfileDefinition() ProfileDefinition {
	builtinTypes := evt1CoreBuiltinDefinitions()
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
			evt1BuiltinStepOutcomeEnum(),
			evt1BuiltinNumericCastErrorEnum(),
		},
		AdmittedImports: map[string]struct{}{
			"Prometheus.Vulkan": {},
		},
		AllowDomainImports: true,
	}
}

var vulkanProfileDefinition = evt1NewVulkanProfileDefinition()
