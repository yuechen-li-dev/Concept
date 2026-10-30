package concept

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Checked semantic domains, independent of native carrier placement.
type LIREnum struct {
	Type            LIRType
	Identity        string
	Name            string
	Size, Alignment int
	Variants        []EnumVariant
}

func lirEnumType(t LIRType) bool { return strings.HasPrefix(string(t), "enum:") }
func lirEnumIdentity(e LIREnum) LIRType {
	e.Type = ""
	data, _ := json.Marshal(e)
	return LIRType("enum:" + digest(data))
}
func (a *lirAggregateTypes) lowerEnum(t Type, decl EnumDecl) (LIRType, error) {
	for _, v := range decl.Variants {
		if len(v.Payload) != 0 {
			return a.lowerTaggedEnum(t, decl)
		}
	}
	key := decl.Module + "|" + decl.Name
	if known, ok := a.known[key]; ok {
		return known, nil
	}
	size, alignment, err := evt1TypeGeometry(a.env, t)
	if err != nil {
		return "", err
	}
	e := LIREnum{Identity: key, Name: t.String(), Size: size, Alignment: alignment}
	for _, v := range decl.Variants {
		e.Variants = append(e.Variants, EnumVariant{Name: v.Name, Tag: v.Tag})
	}
	e.Type = lirEnumIdentity(e)
	a.enums = append(a.enums, e)
	a.known[key] = e.Type
	return e.Type, nil
}
func verifyLIREnums(m LIRModule) error {
	patterns := false
	for _, f := range m.Functions {
		patterns = patterns || len(f.Patterns) > 0
	}
	domains := map[LIRType]LIREnum{}
	identities := map[string]bool{}
	for _, e := range m.Enums {
		if e.Type != lirEnumIdentity(e) || e.Identity == "" || identities[e.Identity] || e.Name == "" || e.Size != 4 || e.Alignment != 4 || len(e.Variants) == 0 || len(e.Variants) > 256 || len(m.Enums) > 128 {
			return fmt.Errorf("LIR_ENUM_DOMAIN")
		}
		names := map[string]bool{}
		for i, v := range e.Variants {
			if v.Tag != i || v.Name == "" || names[v.Name] {
				return fmt.Errorf("LIR_ENUM_VARIANT")
			}
			names[v.Name] = true
		}
		identities[e.Identity] = true
		domains[e.Type] = e
	}
	check := func(t LIRType) error {
		if lirEnumType(t) {
			if _, ok := domains[t]; !ok {
				return fmt.Errorf("LIR_ENUM_UNKNOWN_TYPE")
			}
		}
		if strings.Contains(string(t), "enum:") && !lirEnumType(t) {
			return fmt.Errorf("LIR_UNSUPPORTED_ENUM_CONTAINER")
		}
		return nil
	}
	for _, f := range m.Functions {
		if err := check(f.Result); err != nil {
			return err
		}
		for _, p := range f.Params {
			if err := check(p.Type); err != nil {
				return err
			}
		}
		for _, s := range f.Slots {
			if err := check(s.Type); err != nil {
				return err
			}
		}
		for _, b := range f.Blocks {
			for _, in := range b.Instructions {
				if err := check(in.Type); err != nil {
					return err
				}
				if lirEnumType(in.Type) && in.Op == "const" {
					n, err := strconv.Atoi(in.Literal)
					if err != nil || n < 0 || n >= len(domains[in.Type].Variants) || strconv.Itoa(n) != in.Literal {
						return fmt.Errorf("LIR_ENUM_DISCRIMINANT")
					}
				}
			}
		}
	}
	if len(m.Enums) > 0 && len(m.TaggedEnums) == 0 && !patterns {
		if len(m.Aggregates) > 0 {
			return fmt.Errorf("LIR_UNSUPPORTED_ENUM_AGGREGATE")
		}
		for _, f := range m.Functions {
			if f.Borrow != nil || f.Machine != nil || f.Activation != nil {
				return fmt.Errorf("LIR_UNSUPPORTED_ENUM_AUTHORITY")
			}
		}
	}
	return nil
}
