package concept

import (
	"fmt"
	"strconv"
)

// bits declarations are scalar-backed ordinary values. Their projections are
// compile-time metadata carried in StructDecl, including module artifacts.
func (p *parser) parseBitsDecl() (StructDecl, error) {
	start := p.next().Span // bits
	name, err := p.expectIdentifier("BITS_NAME_REQUIRED", "expected bits type name")
	if err != nil {
		return StructDecl{}, err
	}
	if _, err = p.expect(":"); err != nil {
		return StructDecl{}, err
	}
	repr, err := p.expectIdentifier("BITS_REPRESENTATION_INVALID", "expected unsigned scalar representation")
	if err != nil {
		return StructDecl{}, err
	}
	width := map[string]int{"uint8": 8, "uint16": 16, "uint32": 32, "uint64": 64}[repr.Lexeme]
	if width == 0 {
		return StructDecl{}, evt1Diagnostic("BITS_REPRESENTATION_INVALID", "bits requires uint8, uint16, uint32, or uint64 representation", repr.Span)
	}
	if _, err = p.expect("{"); err != nil {
		return StructDecl{}, err
	}
	decl := StructDecl{Name: name.Lexeme, Record: true, BitsRepresentation: repr.Lexeme, Fields: []Field{{Name: "raw", Type: Type{Name: repr.Lexeme, Kind: TypeBuiltin, Span: repr.Span}, Span: repr.Span}}, Span: start}
	seen := map[string]bool{"raw": true}
	occupied := uint64(0)
	for !p.done() && p.peekLexeme() != "}" {
		field, fieldErr := p.expectIdentifier("BITS_FIELD_INVALID", "expected bit field name")
		if fieldErr != nil {
			return StructDecl{}, fieldErr
		}
		if seen[field.Lexeme] {
			return StructDecl{}, evt1Diagnostic("BITS_FIELD_DUPLICATE", "duplicate bit field "+field.Lexeme, field.Span)
		}
		seen[field.Lexeme] = true
		if _, err = p.expect(":"); err != nil {
			return StructDecl{}, err
		}
		loTok := p.next()
		lo, parseErr := strconv.ParseUint(loTok.Lexeme, 0, 8)
		if parseErr != nil {
			return StructDecl{}, evt1Diagnostic("BITS_RANGE_INVALID", "bit position must be a nonnegative integer", loTok.Span)
		}
		hi := lo
		if p.peekLexeme() == ".." {
			p.next()
			hiTok := p.next()
			hi, parseErr = strconv.ParseUint(hiTok.Lexeme, 0, 8)
			if parseErr != nil {
				return StructDecl{}, evt1Diagnostic("BITS_RANGE_INVALID", "bit range end must be a nonnegative integer", hiTok.Span)
			}
		}
		if _, err = p.expect(";"); err != nil {
			return StructDecl{}, err
		}
		if lo > hi || hi >= uint64(width) {
			return StructDecl{}, evt1Diagnostic("BITS_RANGE_INVALID", fmt.Sprintf("bit range %d..%d is outside %s or reversed", lo, hi, repr.Lexeme), field.Span)
		}
		mask := bitRangeMask(int(lo), int(hi))
		if occupied&mask != 0 {
			return StructDecl{}, evt1Diagnostic("BITS_FIELD_OVERLAP", "bit field "+field.Lexeme+" overlaps a preceding field", field.Span)
		}
		occupied |= mask
		decl.BitFields = append(decl.BitFields, BitFieldDecl{Name: field.Lexeme, Start: int(lo), End: int(hi), Span: field.Span})
	}
	if _, err = p.expect("}"); err != nil {
		return StructDecl{}, err
	}
	if p.peekLexeme() == ";" {
		p.next()
	}
	return decl, nil
}

func bitRangeMask(start, end int) uint64 {
	if end-start == 63 {
		return ^uint64(0)
	}
	return ((uint64(1) << (end - start + 1)) - 1) << start
}

func evt1BitField(decl StructDecl, name string) (BitFieldDecl, bool) {
	for _, field := range decl.BitFields {
		if field.Name == name {
			return field, true
		}
	}
	return BitFieldDecl{}, false
}

func evt1BitFieldType(decl StructDecl, field BitFieldDecl) Type {
	if field.Start == field.End {
		t, _ := evt1BuiltinType("bool", field.Span)
		return t
	}
	t, _ := evt1BuiltinType(decl.BitsRepresentation, field.Span)
	return t
}
