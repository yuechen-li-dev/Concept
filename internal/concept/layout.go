package concept

import "fmt"

// Fixed inline storage is deliberately over-aligned so a byte array can back
// ordinary CPU layouts without changing ownership or allocating a second object.
const evt1InlineStorageAlignment = 64

func evt1SemanticViewCName(t Type) string {
	kind := "layout"
	if t.Kind == TypeStream {
		kind = "stream"
	}
	mutability := "mut"
	if t.Const {
		mutability = "const"
	}
	return fmt.Sprintf("concept_ref_%s_%s_%s", mutability, kind, evt1CName(t.Name))
}

func evt1IsSemanticViewType(env *semanticEnv, t Type) bool {
	_, layout := env.layouts[t.Name]
	_, stream := env.streams[t.Name]
	return layout || stream
}

func evt1AlignUp(value, alignment int) int {
	return (value + alignment - 1) / alignment * alignment
}

func evt1TypeGeometry(env *semanticEnv, t Type) (int, int, error) {
	resolved, err := evt1ResolveType(env, nil, t)
	if err != nil {
		return 0, 0, err
	}
	if resolved.isBorrowLike() || resolved.isOwned() || resolved.PointerTo != nil {
		return 0, 0, evt1Diagnostic("CV4573", "layout regions require fixed value storage", t.Span)
	}
	if resolved.ArrayElem != nil {
		if evt1StorageHasRuntimeShape(resolved) {
			return 0, 0, evt1Diagnostic("CV4573", "layout extents must be compile-time fixed", t.Span)
		}
		size, alignment, err := evt1TypeGeometry(env, *resolved.ArrayElem)
		if err != nil {
			return 0, 0, err
		}
		return size * evt1StorageElementCount(resolved), alignment, nil
	}
	switch resolved.Name {
	case "byte", "bool":
		return 1, 1, nil
	case "int", "uint", "float":
		return 4, 4, nil
	case "uint64":
		return 8, 8, nil
	case "usize":
		return 8, 8, nil
	}
	if decl, ok := env.structs[resolved.Name]; ok {
		offset, alignment := 0, 1
		for _, field := range decl.Fields {
			size, align, err := evt1TypeGeometry(env, field.Type)
			if err != nil {
				return 0, 0, err
			}
			offset = evt1AlignUp(offset, align) + size
			if align > alignment {
				alignment = align
			}
		}
		return evt1AlignUp(offset, alignment), alignment, nil
	}
	return 0, 0, evt1Diagnostic("CV4573", "type "+resolved.String()+" has no fixed layout geometry", t.Span)
}

func evt1AnalyzeLayouts(env *semanticEnv, declarations []LayoutDecl) error {
	for _, source := range declarations {
		decl := source
		if len(decl.Regions) == 0 {
			return evt1Diagnostic("CV4573", "layout "+decl.Name+" must declare at least one region", decl.Span)
		}
		seen := map[string]bool{}
		cursor, layoutAlign := 0, 1
		for i, region := range decl.Regions {
			if seen[region.Name] {
				return evt1Diagnostic("CV4571", fmt.Sprintf("duplicate layout region %s.%s", decl.Name, region.Name), region.Span)
			}
			seen[region.Name] = true
			resolved, err := evt1ResolveType(env, nil, region.Type)
			if err != nil {
				return err
			}
			size, naturalAlign, err := evt1TypeGeometry(env, resolved)
			if err != nil {
				return err
			}
			alignment := naturalAlign
			if region.RequestedAlign != 0 {
				if region.RequestedAlign > 4096 || region.RequestedAlign&(region.RequestedAlign-1) != 0 {
					return evt1Diagnostic("CV4572", fmt.Sprintf("layout alignment %d must be a power of two from 1 through 4096", region.RequestedAlign), region.Span)
				}
				if region.RequestedAlign > alignment {
					alignment = region.RequestedAlign
				}
			}
			offset := evt1AlignUp(cursor, alignment)
			if region.ExplicitOffset != nil {
				offset = *region.ExplicitOffset
			}
			if offset < 0 || offset%alignment != 0 {
				return evt1Diagnostic("CV4574", fmt.Sprintf("layout region %s.%s offset %d is not aligned to %d", decl.Name, region.Name, offset, alignment), region.Span)
			}
			end := offset + size
			for priorIndex := 0; priorIndex < i; priorIndex++ {
				prior := decl.Regions[priorIndex]
				if offset < prior.Offset+prior.ByteExtent && prior.Offset < end {
					return evt1Diagnostic("CV4576", fmt.Sprintf("layout regions %s.%s and %s.%s overlap", decl.Name, prior.Name, decl.Name, region.Name), region.Span)
				}
			}
			region.Type = resolved
			region.ID = decl.Name + "." + region.Name
			region.Offset = offset
			region.ByteExtent = size
			region.Alignment = alignment
			decl.Regions[i] = region
			if end > cursor {
				cursor = end
			}
			if alignment > layoutAlign {
				layoutAlign = alignment
			}
		}
		decl.Alignment = layoutAlign
		decl.Size = evt1AlignUp(cursor, layoutAlign)
		env.layouts[decl.Name] = decl
		fields := map[string]Type{}
		for _, region := range decl.Regions {
			fields[region.Name] = region.Type
		}
		env.fieldSets[decl.Name] = fields
	}
	return nil
}

func evt1AnalyzeStreams(env *semanticEnv, declarations []StreamDecl) error {
	for _, source := range declarations {
		decl := source
		layout, ok := env.layouts[decl.LayoutName]
		if !ok {
			return evt1Diagnostic("CV4581", fmt.Sprintf("stream %s references unknown layout %s", decl.Name, decl.LayoutName), decl.Span)
		}
		regions := map[string]LayoutRegion{}
		for _, region := range layout.Regions {
			regions[region.Name] = region
		}
		seen := map[string]bool{}
		fields := map[string]Type{}
		for i, channel := range decl.Channels {
			if seen[channel.Name] {
				return evt1Diagnostic("CV4583", fmt.Sprintf("duplicate stream channel %s.%s", decl.Name, channel.Name), channel.Span)
			}
			seen[channel.Name] = true
			region, ok := regions[channel.RegionName]
			if !ok {
				return evt1Diagnostic("CV4582", fmt.Sprintf("stream channel %s.%s references unknown region %s.%s", decl.Name, channel.Name, decl.LayoutName, channel.RegionName), channel.Span)
			}
			channel.RegionID = region.ID
			channel.Type = region.Type
			decl.Channels[i] = channel
			fields[channel.Name] = region.Type
		}
		env.streams[decl.Name] = decl
		env.fieldSets[decl.Name] = fields
	}
	return nil
}

func evt1LayoutRegion(env *semanticEnv, typeName, field string) (LayoutRegion, bool) {
	if stream, ok := env.streams[typeName]; ok {
		typeName = stream.LayoutName
		for _, channel := range stream.Channels {
			if channel.Name == field {
				field = channel.RegionName
				break
			}
		}
	}
	layout, ok := env.layouts[typeName]
	if !ok {
		return LayoutRegion{}, false
	}
	for _, region := range layout.Regions {
		if region.Name == field {
			return region, true
		}
	}
	return LayoutRegion{}, false
}

func evt1LayoutQuery(env *semanticEnv, name string, typeArg Type, args []Expr) (int, error) {
	if name == "SizeOf" || name == "AlignOf" {
		if len(args) != 0 {
			return 0, evt1Diagnostic("GENERIC_LAYOUT_QUERY_ARGUMENTS", name+" expects no value arguments", typeArg.Span)
		}
		size, alignment, err := evt1TypeGeometry(env, typeArg)
		if err != nil {
			return 0, err
		}
		if name == "SizeOf" {
			return size, nil
		}
		return alignment, nil
	}
	layout, ok := env.layouts[typeArg.Name]
	if !ok {
		return 0, evt1Diagnostic("CV4592", fmt.Sprintf("%s requires a declared layout type", name), typeArg.Span)
	}
	switch name {
	case "LayoutSize":
		if len(args) != 0 {
			return 0, evt1Diagnostic("CV4592", "LayoutSize expects no value arguments", typeArg.Span)
		}
		return layout.Size, nil
	case "LayoutAlign":
		if len(args) != 0 {
			return 0, evt1Diagnostic("CV4592", "LayoutAlign expects no value arguments", typeArg.Span)
		}
		return layout.Alignment, nil
	case "LayoutOffset":
		if len(args) != 1 {
			return 0, evt1Diagnostic("CV4592", "LayoutOffset expects one region-name string", typeArg.Span)
		}
		literal, ok := args[0].(*StringLiteral)
		if !ok {
			return 0, evt1Diagnostic("CV4592", "LayoutOffset region must be a string literal", args[0].exprSpan())
		}
		region, ok := evt1LayoutRegion(env, layout.Name, literal.Value)
		if !ok {
			return 0, evt1Diagnostic("CV4592", fmt.Sprintf("unknown layout region %s.%s", layout.Name, literal.Value), literal.Span)
		}
		return region.Offset, nil
	default:
		return 0, evt1Diagnostic("CV4592", "unknown layout query "+name, typeArg.Span)
	}
}

func evt1IsTypeLayoutQuery(name string) bool {
	return name == "LayoutSize" || name == "LayoutAlign" || name == "LayoutOffset" || name == "SizeOf" || name == "AlignOf"
}
