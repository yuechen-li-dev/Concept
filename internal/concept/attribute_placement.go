package concept

// User field/case attributes are reflection selectors and survive semantic
// artifacts. Compiler-owned attributes have position-specific behavior and must
// never be accepted as decorative field/case metadata.
func evt1ValidateAttributePlacement(module Module) error {
	validate := func(attributes []Attribute) error {
		for _, attribute := range attributes {
			reserved := evt1TestKinds[attribute.Name] || evt1SemanticAccessAttribute(attribute.Name)
			switch attribute.Name {
			case "must_use", "repr", "reflect", "machine", "diagnostic", "verify_foreign", "artifact", "foretold":
				reserved = true
			}
			if reserved {
				return evt1Diagnostic("ATTRIBUTE_POSITION_INVALID", "[["+attribute.Name+"]] does not apply to a field or enum case; place it on its supported type, concept, or function declaration", attribute.Span)
			}
		}
		return nil
	}
	fields := func(fields []Field) error {
		for _, field := range fields {
			if err := validate(field.Attributes); err != nil {
				return err
			}
		}
		return nil
	}
	for _, decl := range module.Structs {
		if err := evt1ReflectableAttributes(decl.Attributes, decl.Span, true); err != nil {
			return err
		}
		if err := fields(decl.Fields); err != nil {
			return err
		}
	}
	for _, decl := range module.GenericTypes {
		if err := evt1ReflectableAttributes(decl.Struct.Attributes, decl.Span, true); err != nil {
			return err
		}
		if evt1HasCRepr(decl.Struct.Attributes) && !decl.Struct.Record {
			return evt1Diagnostic("C_ABI_REPR_INVALID", "[[repr(C)]] on a template requires record struct; C ABI field legality is checked for each closed instantiation", decl.Span)
		}
		if err := fields(decl.Struct.Fields); err != nil {
			return err
		}
	}
	for _, decl := range module.Enums {
		if err := evt1ReflectableAttributes(decl.Attributes, decl.Span, false); err != nil {
			return err
		}
		for _, variant := range decl.Variants {
			if err := validate(variant.Attributes); err != nil {
				return err
			}
			if err := fields(variant.Payload); err != nil {
				return err
			}
		}
	}
	return nil
}
