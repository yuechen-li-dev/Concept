package concept

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Checked sum layouts are distinct from ordinary enum scalar domains and from
// records. Alternatives overlap, but their fields never coexist as live data.
// This is semantic storage evidence, not an AMD64 placement or copy plan.
type LIRTaggedEnum struct {
	Type                                            LIRType
	Identity, Name, Representation, CopyEligibility string
	Size, Alignment, PayloadOffset                  int
	Variants                                        []LIRTaggedVariant
}

type LIRTaggedVariant struct {
	Identity, Name  string
	Tag             int
	Size, Alignment int
	Fields          []LIRAggregateField
}

func lirTaggedType(t LIRType) bool { return strings.HasPrefix(string(t), "tagged:") }
func lirTaggedIdentity(layout LIRTaggedEnum) LIRType {
	layout.Type = ""
	data, _ := json.Marshal(layout)
	return LIRType("tagged:" + digest(data))
}

func (a *lirAggregateTypes) lowerTaggedEnum(t Type, decl EnumDecl) (LIRType, error) {
	key := decl.Module + "|" + decl.Name
	for active := range a.visiting {
		if strings.HasPrefix(active, "tagged:") {
			return "", fmt.Errorf("EVT2H2_NESTED_TAGGED_ENUM %s", t.String())
		}
	}
	if known, ok := a.known[key]; ok {
		return known, nil
	}
	if len(a.tagged) >= 128 || len(decl.Variants) == 0 || len(decl.Variants) > 256 {
		return "", fmt.Errorf("EVT2H2_TAGGED_CAPACITY")
	}
	if !evt1TypeCopyable(a.env, t) || evt1TypeHasDrop(a.env, t) {
		return "", fmt.Errorf("EVT2H2_TAGGED_COPY_INELIGIBLE %s", t.String())
	}
	a.visiting["tagged:"+key] = true
	defer delete(a.visiting, "tagged:"+key)
	layout := LIRTaggedEnum{Identity: key, Name: t.String(), Representation: "checked_tagged_union", CopyEligibility: "checked_value"}
	// The bounded checked sum contract uses a 4/4 ordinal tag and a union of
	// checked variant records. This does not grant public payload SizeOf/AlignOf
	// or a foreign C ABI. Physical realization must validate this separately.
	payloadSize, payloadAlign := 1, 1
	for _, variant := range decl.Variants {
		if len(variant.Payload) > lirAggregateMaxFields {
			return "", fmt.Errorf("EVT2H2_TAGGED_CAPACITY")
		}
		v := LIRTaggedVariant{Identity: key + "|" + variant.Name, Name: variant.Name, Tag: variant.Tag, Alignment: 1}
		fieldTypes := make([]LIRType, len(variant.Payload))
		for i, field := range variant.Payload {
			ft, err := a.lower(field.Type)
			if err != nil {
				return "", fmt.Errorf("%s::%s.%s: %w", t.String(), variant.Name, field.Name, err)
			}
			if lirEnumType(ft) || lirTaggedType(ft) || lirReferenceType(ft) {
				return "", fmt.Errorf("EVT2H2_UNSUPPORTED_PAYLOAD_TYPE %s", field.Type.String())
			}
			fieldTypes[i] = ft
		}
		var offsets []int
		if len(variant.Payload) > 0 {
			var err error
			offsets, v.Size, v.Alignment, err = evt1StructFieldOffsets(a.env, StructDecl{Fields: variant.Payload, Span: variant.Span})
			if err != nil {
				return "", fmt.Errorf("EVT2H2_PAYLOAD_LAYOUT %s::%s: %w", t.String(), variant.Name, err)
			}
		}
		for i, field := range variant.Payload {
			ft := fieldTypes[i]
			fs, fa, err := evt1TypeGeometry(a.env, field.Type)
			if err != nil {
				return "", err
			}
			semantic, _ := json.Marshal(evt1GenericIdentityType(field.Type))
			f := LIRAggregateField{Name: field.Name, Type: ft, TypeIdentity: digest(semantic), Offset: offsets[i], Size: fs, Alignment: fa, Readonly: field.Type.Const, DeclarationIdentity: v.Identity + "|" + field.Name}
			if field.Type.ArrayElem != nil {
				f.ElementType, err = a.lower(*field.Type.ArrayElem)
				if err != nil {
					return "", err
				}
				f.Extent = field.Type.ArrayLength
				f.Stride, f.ElementAlignment, err = evt1TypeGeometry(a.env, *field.Type.ArrayElem)
				if err != nil {
					return "", err
				}
			}
			v.Fields = append(v.Fields, f)
		}
		payloadSize = max(payloadSize, v.Size)
		payloadAlign = max(payloadAlign, v.Alignment)
		layout.Variants = append(layout.Variants, v)
	}
	layout.Alignment = max(4, payloadAlign)
	layout.PayloadOffset = evt1AlignUp(4, payloadAlign)
	if payloadSize > lirAggregateMaxSize-layout.PayloadOffset {
		return "", fmt.Errorf("EVT2H2_TAGGED_CAPACITY")
	}
	layout.Size = evt1AlignUp(layout.PayloadOffset+payloadSize, layout.Alignment)
	layout.Type = lirTaggedIdentity(layout)
	a.tagged = append(a.tagged, layout)
	a.known[key] = layout.Type
	return layout.Type, nil
}

func verifyLIRTaggedEnums(m LIRModule, layouts map[LIRType]LIRAggregateLayout) (map[LIRType]LIRTaggedEnum, error) {
	domains := map[LIRType]LIRTaggedEnum{}
	identities := map[string]bool{}
	for _, layout := range m.Aggregates {
		identities[layout.Identity] = true
	}
	for _, domain := range m.Enums {
		identities[domain.Identity] = true
	}
	if len(m.TaggedEnums) > 128 {
		return nil, fmt.Errorf("LIR_TAGGED_CAPACITY")
	}
	for _, e := range m.TaggedEnums {
		if e.Type != lirTaggedIdentity(e) || e.Identity == "" || e.Name == "" || identities[e.Identity] || e.Representation != "checked_tagged_union" || len(e.Variants) == 0 || len(e.Variants) > 256 {
			return nil, fmt.Errorf("LIR_TAGGED_DOMAIN")
		}
		if e.CopyEligibility != "checked_value" {
			return nil, fmt.Errorf("LIR_TAGGED_COPY_ELIGIBILITY")
		}
		names := map[string]bool{}
		payloadSize, payloadAlign := 1, 1
		hasPayload := false
		for i, v := range e.Variants {
			if v.Tag != i || v.Name == "" || names[v.Name] || v.Identity != e.Identity+"|"+v.Name || len(v.Fields) > lirAggregateMaxFields {
				return nil, fmt.Errorf("LIR_TAGGED_VARIANT")
			}
			names[v.Name] = true
			fields := map[string]bool{}
			end, align := 0, 1
			for _, field := range v.Fields {
				size, alignment, err := lirAggregateGeometry(field.Type, layouts, 0)
				if err != nil || lirEnumType(field.Type) || lirTaggedType(field.Type) || size <= 0 || size > lirAggregateMaxSize || alignment <= 0 || alignment > 16 {
					return nil, fmt.Errorf("LIR_TAGGED_PAYLOAD_TYPE")
				}
				if field.Name == "" || fields[field.Name] || !objectDigest(field.TypeIdentity) || field.DeclarationIdentity != v.Identity+"|"+field.Name || field.Size != size || field.Alignment != alignment || field.Offset != evt1AlignUp(end, alignment) {
					return nil, fmt.Errorf("LIR_TAGGED_FIELD")
				}
				if element, extent, array := lirArrayValue(field.Type); array {
					stride, ea, err := lirAggregateGeometry(element, layouts, 0)
					if err != nil || field.ElementType != element || field.Extent != extent || field.Stride != stride || field.ElementAlignment != ea {
						return nil, fmt.Errorf("LIR_TAGGED_ARRAY")
					}
				} else if field.ElementType != "" || field.Extent != 0 || field.Stride != 0 || field.ElementAlignment != 0 {
					return nil, fmt.Errorf("LIR_TAGGED_ARRAY")
				}
				fields[field.Name] = true
				end, align = field.Offset+size, max(align, alignment)
				if end > lirAggregateMaxSize {
					return nil, fmt.Errorf("LIR_TAGGED_CAPACITY")
				}
			}
			if v.Size != evt1AlignUp(end, align) || v.Alignment != align {
				return nil, fmt.Errorf("LIR_TAGGED_VARIANT_LAYOUT")
			}
			hasPayload = hasPayload || len(v.Fields) > 0
			payloadSize, payloadAlign = max(payloadSize, v.Size), max(payloadAlign, v.Alignment)
		}
		align := max(4, payloadAlign)
		offset := evt1AlignUp(4, payloadAlign)
		if !hasPayload || e.Alignment != align || e.PayloadOffset != offset || e.Size != evt1AlignUp(offset+payloadSize, align) || e.Size > lirAggregateMaxSize {
			return nil, fmt.Errorf("LIR_TAGGED_LAYOUT")
		}
		identities[e.Identity] = true
		domains[e.Type] = e
		// Shared value/slot/call checking sees an opaque logical object. Empty
		// fields deliberately do not grant record projection or replacement.
		layouts[e.Type] = LIRAggregateLayout{Type: e.Type, Identity: e.Identity, Name: e.Name, Kind: "tagged", Size: e.Size, Alignment: e.Alignment, CopyEligibility: e.CopyEligibility}
	}
	return domains, nil
}

func (b *lirBuilder) taggedConstruction(e *ConstructExpr, decl EnumDecl, typ LIRType) (int, LIRType, error) {
	for _, variant := range decl.Variants {
		if variant.Name != e.VariantName {
			continue
		}
		if len(e.Args) != len(variant.Payload) {
			return 0, "", fmt.Errorf("EVT2H2_CONSTRUCTOR_ARITY")
		}
		var args []int
		for i, arg := range e.Args {
			value, actual, err := b.objectInitializer(arg, variant.Payload[i].Type)
			if err != nil {
				return 0, "", err
			}
			expected, err := b.lowerType(variant.Payload[i].Type)
			if err != nil || actual != expected {
				return 0, "", fmt.Errorf("EVT2H2_CONSTRUCTOR_TYPE")
			}
			args = append(args, value)
		}
		return b.emit(LIRInstruction{Op: "make_tagged", Result: 0, Type: typ, Args: args, Slot: -1, Member: variant.Tag, Source: e.Span}), typ, nil
	}
	return 0, "", fmt.Errorf("EVT2_UNKNOWN_ENUM_VARIANT")
}
