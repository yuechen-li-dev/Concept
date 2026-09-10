package concept

import "fmt"

// evt1ValidateForeignContracts keeps ABI declarations and semantic authority
// separate. A contract can bind only to one uniquely declared foreign symbol;
// it does not make any other extern declaration trusted.
func evt1ValidateForeignContracts(env *semanticEnv, contracts []ForeignContractDecl) error {
	for _, contract := range contracts {
		if _, exists := env.foreignContracts[contract.Name]; exists {
			return evt1Diagnostic("FOREIGN_CONTRACT_DUPLICATE", fmt.Sprintf("duplicate foreign contract %s", contract.Name), contract.Span)
		}
		functions := env.functions[contract.Operation]
		if len(functions) != 1 || functions[0].ExternABI == "" {
			return evt1Diagnostic("FOREIGN_CONTRACT_TARGET_INVALID", fmt.Sprintf("foreign contract %s requires one foreign operation, got %s", contract.Name, contract.Operation), contract.Span)
		}
		if _, exists := env.foreignByOperation[contract.Operation]; exists {
			return evt1Diagnostic("FOREIGN_CONTRACT_TARGET_DUPLICATE", fmt.Sprintf("foreign operation %s already has a semantic contract", contract.Operation), contract.Span)
		}
		if contract.ExtentParam != "" || contract.AlignmentParam != "" {
			if contract.ExtentParam == "" || contract.AlignmentParam == "" || contract.AddressSpace.Name == "" {
				return evt1Diagnostic("FOREIGN_STORAGE_CONTRACT_INCOMPLETE", "ExternalStorage requires address space, extent, and alignment authority", contract.Span)
			}
			if contract.AddressSpace.Kind != TypeStruct {
				return evt1Diagnostic("ADDRESS_SPACE_TAG_INVALID", "foreign storage address space must be a nominal struct tag", contract.AddressSpace.Span)
			}
			if functions[0].ReturnType.PointerTo == nil {
				return evt1Diagnostic("FOREIGN_CONTRACT_ABI_MISMATCH", fmt.Sprintf("foreign storage operation %s must return a pointer", contract.Operation), functions[0].ReturnType.Span)
			}
			for _, name := range []string{contract.ExtentParam, contract.AlignmentParam} {
				index := evt1ForeignParameterIndex(env, contract.Operation, name)
				if index < 0 {
					return evt1Diagnostic("FOREIGN_CONTRACT_PARAMETER_INVALID", fmt.Sprintf("foreign contract %s names missing parameter %s", contract.Name, name), contract.Span)
				}
				parameterType := functions[0].Params[index].Type
				if parameterType.Name != "usize" {
					return evt1Diagnostic("FOREIGN_CONTRACT_ABI_MISMATCH", fmt.Sprintf("foreign storage parameter %s must use the usize ABI representation", name), functions[0].Params[index].Span)
				}
			}
		}
		env.foreignContracts[contract.Name] = contract
		env.foreignByOperation[contract.Operation] = contract
	}
	return nil
}

func evt1ForeignParameterIndex(env *semanticEnv, operation, name string) int {
	functions := env.functions[operation]
	if len(functions) != 1 {
		return -1
	}
	for index, parameter := range functions[0].Params {
		if parameter.Name == name {
			return index
		}
	}
	return -1
}
