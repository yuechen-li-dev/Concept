package concept

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"strconv"
	"strings"
)

// BridgeCodecGeneration consumes checked, typed reflection, never declaration
// printer text. This is a build-time emitter for the bounded CMIR bridge only.
type BridgeCodecGeneration struct {
	Hash     string
	Magic    string
	Version  uint64
	Go       []byte
	Concept  []byte
	Metadata []byte
}
type bridgeSchemaType struct {
	Name      string             `json:"name"`
	Module    string             `json:"module,omitempty"`
	Arguments []bridgeSchemaType `json:"arguments,omitempty"`
	Element   *bridgeSchemaType  `json:"element,omitempty"`
	Extent    int                `json:"extent,omitempty"`
}
type bridgeSchemaField struct {
	Name   string           `json:"name"`
	Type   bridgeSchemaType `json:"type"`
	GoCast string           `json:"go_cast,omitempty"`
}
type bridgeSchemaCase struct {
	Name  string  `json:"name"`
	Tag   int     `json:"tag"`
	Value *string `json:"go_value,omitempty"`
}
type bridgeSchemaRecord struct {
	Name   string              `json:"name"`
	Fields []bridgeSchemaField `json:"fields,omitempty"`
	Cases  []bridgeSchemaCase  `json:"cases,omitempty"`
}
type bridgeSchemaDefinition struct {
	Module         string               `json:"module"`
	Magic          string               `json:"magic"`
	Version        uint64               `json:"version"`
	MaxItems       uint64               `json:"max_items"`
	Representation string               `json:"representation"`
	Records        []bridgeSchemaRecord `json:"records"`
}

func bridgeSchemaTypeOf(t Type) bridgeSchemaType {
	out := bridgeSchemaType{Name: t.Name}
	if t.Application != nil {
		out.Name = t.Application.Declaration.Name
		out.Module = t.Application.Declaration.Module
		for _, a := range t.Application.Arguments {
			if a.Kind == "type" {
				out.Arguments = append(out.Arguments, bridgeSchemaTypeOf(a.Type))
			}
		}
	} else {
		for _, a := range t.TypeArgs {
			out.Arguments = append(out.Arguments, bridgeSchemaTypeOf(a))
		}
	}
	if t.ArrayElem != nil {
		e := bridgeSchemaTypeOf(*t.ArrayElem)
		out.Name = "array"
		out.Element = &e
		out.Extent = t.ArrayLength
	}
	return out
}
func bridgeSelector(attrs []Attribute, name string) (Expr, error) {
	var result Expr
	for _, a := range attrs {
		if a.Name == name {
			if result != nil || len(a.Args) != 1 {
				return nil, fmt.Errorf("MIR_BRIDGE_SCHEMA_SELECTOR %s", name)
			}
			result = a.Args[0]
		}
	}
	return result, nil
}

func GenerateMachineBridgeCodecs(module Module) (BridgeCodecGeneration, error) {
	var out BridgeCodecGeneration
	var maxItems uint64
	var representation string
	if module.Name != "Standard.Backend.BridgeSchema" {
		return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_MODULE: %s", module.Name)
	}
	for _, c := range module.ComptimeDecls {
		switch c.Name {
		case "BridgeMagic":
			if v, ok := c.Value.(*StringLiteral); ok {
				out.Magic = v.Value
			}
		case "BridgeVersion":
			if v, ok := c.Value.(*IntLiteral); ok {
				out.Version = v.Magnitude
			}
		case "BridgeMaxItems":
			if v, ok := c.Value.(*IntLiteral); ok {
				maxItems = v.Magnitude
			}
		case "BridgeRepresentation":
			if v, ok := c.Value.(*StringLiteral); ok {
				representation = v.Value
			}
		}
	}
	if len(out.Magic) != 8 || out.Version == 0 || maxItems == 0 || maxItems > 2147483647 || representation == "" {
		return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_IDENTITY")
	}
	def := bridgeSchemaDefinition{Module: module.Name, Magic: out.Magic, Version: out.Version, MaxItems: maxItems, Representation: representation}
	for _, info := range module.ReflectionResults {
		rec := bridgeSchemaRecord{Name: info.Type.Name}
		if !strings.HasPrefix(rec.Name, "Wire") && info.Kind != "enum" {
			return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_RECORD %s", rec.Name)
		}
		for _, f := range info.Fields {
			field := bridgeSchemaField{Name: f.Name, Type: bridgeSchemaTypeOf(f.Type)}
			expr, err := bridgeSelector(f.Attributes, "bridge_go_type")
			if err != nil {
				return out, err
			}
			if expr != nil {
				name, ok := expr.(*NameExpr)
				if !ok {
					return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_GO_TYPE %s.%s", rec.Name, f.Name)
				}
				field.GoCast = name.Name
			}
			rec.Fields = append(rec.Fields, field)
		}
		for _, c := range info.EnumCases {
			if len(c.Payload) != 0 {
				return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_ENUM_PAYLOAD %s", rec.Name)
			}
			variant := bridgeSchemaCase{Name: c.Name, Tag: c.Tag}
			expr, err := bridgeSelector(c.Attributes, "bridge_value")
			if err != nil {
				return out, err
			}
			if expr != nil {
				value, ok := expr.(*StringLiteral)
				if !ok {
					return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_ENUM_VALUE %s", rec.Name)
				}
				variant.Value = &value.Value
			}
			rec.Cases = append(rec.Cases, variant)
		}
		def.Records = append(def.Records, rec)
	}
	// Every schema record/enum must be reflected. Missing coverage cannot silently
	// produce only one codec side. Support/view types are the three primitives.
	reflected := map[string]bool{}
	for _, r := range def.Records {
		if reflected[r.Name] {
			return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_DUPLICATE %s", r.Name)
		}
		reflected[r.Name] = true
	}
	for _, r := range module.Structs {
		if strings.HasPrefix(r.Name, "Wire") && !reflected[r.Name] {
			return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_UNREFLECTED %s", r.Name)
		}
	}
	for _, e := range module.Enums {
		if e.Name != "BackendError" && !reflected[e.Name] {
			return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_UNREFLECTED %s", e.Name)
		}
	}
	if !reflected["WireMachineModule"] || !reflected["WireHeader"] {
		return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_ROOT")
	}
	metadata, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return out, err
	}
	out.Metadata = append(metadata, '\n')
	out.Hash = digest(metadata)
	emitter := bridgeGoEmitter{records: map[string]bridgeSchemaRecord{}}
	for _, r := range def.Records {
		emitter.records[r.Name] = r
	}
	headerSize := 0
	for _, field := range emitter.records["WireHeader"].Fields {
		size, err := bridgeFixedWireSize(field.Type)
		if err != nil {
			return out, err
		}
		headerSize += size
	}
	var goSource bytes.Buffer
	fmt.Fprintf(&goSource, "// Code generated from checked Standard.Backend.BridgeSchema; DO NOT EDIT.\npackage concept\nimport (\"fmt\";\"io\")\nconst MachineBridgeSchema = %q\nconst MachineBridgeVersion = %d\nconst MachineBridgeSchemaHash = %q\n", out.Magic, out.Version, out.Hash)
	fmt.Fprintf(&goSource, "const MachineBridgeMaxItems = %d\n", maxItems)
	fmt.Fprintf(&goSource, "const MachineBridgeHeaderSize = %d\n", headerSize)
	var conceptSource bytes.Buffer
	fmt.Fprintf(&conceptSource, "// Code generated from checked BridgeSchema metadata; DO NOT EDIT.\nmodule Standard.Backend.BridgeCodec;\nprofile Core;\nimport Standard.Backend.BridgeDerive;\nstring MachineBridgeSchemaHash() { return %q; }\n", out.Hash)
	hashBytes, _ := hex.DecodeString(out.Hash)
	fmt.Fprint(&conceptSource, "uint8[32] MachineBridgeSchemaHashBytes() { uint8[32] value = [")
	for i, b := range hashBytes {
		if i > 0 {
			fmt.Fprint(&conceptSource, ",")
		}
		fmt.Fprintf(&conceptSource, "%du", b)
	}
	fmt.Fprintln(&conceptSource, "]; return value; }")
	for _, r := range def.Records {
		if r.Name == "WireHeader" {
			fmt.Fprintln(&goSource, "type bridgeWireHeader struct {")
			for _, f := range r.Fields {
				fmt.Fprintf(&goSource, "%s %s\n", f.Name, emitter.goType(f.Type))
			}
			fmt.Fprintln(&goSource, "}")
		}
		if len(r.Cases) > 0 {
			fmt.Fprintf(&conceptSource, "derive DeriveBridgeEnumRead reflect<%s>;\n", r.Name)
			fmt.Fprintf(&conceptSource, "derive DeriveBridgeEnumWrite reflect<%s>;\n", r.Name)
			fmt.Fprintf(&goSource, "var bridgeTags%s = []string{", r.Name)
			for i, c := range r.Cases {
				if c.Tag != i {
					return out, fmt.Errorf("MIR_BRIDGE_SCHEMA_TAG_ORDER %s", r.Name)
				}
				value := c.Name
				if c.Value != nil {
					value = *c.Value
				}
				fmt.Fprintf(&goSource, "%q,", value)
			}
			fmt.Fprintln(&goSource, "}")
			continue
		}
		fmt.Fprintf(&conceptSource, "derive DeriveBridgeRead reflect<%s>;\n", r.Name)
		if r.Name != "WireHeader" {
			fmt.Fprintf(&conceptSource, "derive DeriveBridgeWrite reflect<%s>;\n", r.Name)
		}
		goType := strings.TrimPrefix(r.Name, "Wire")
		if r.Name == "WireHeader" {
			goType = "bridgeWireHeader"
		}
		fmt.Fprintf(&goSource, "func (w *machineBridgeWriter) write%s(v %s) error {\n", r.Name, goType)
		for _, f := range r.Fields {
			statement, e := emitter.write(f.Type, "v."+f.Name)
			if e != nil {
				return out, e
			}
			fmt.Fprintf(&goSource, "{offset:=w.Len(); if err := %s; err != nil { return fmt.Errorf(\"MIR_BRIDGE_FIELD %s.%s offset=%%d: %%w\", offset, err) }; w.traceField(%q,%q,offset)}\n", statement, r.Name, f.Name, r.Name, f.Name)
		}
		fmt.Fprintln(&goSource, "return nil\n}")
		fmt.Fprintf(&goSource, "func (r machineBridgeReader) read%s() (%s,error) {\nvar v %s\nvar err error\n", r.Name, goType, goType)
		for _, f := range r.Fields {
			expr, e := emitter.read(f.Type)
			if e != nil {
				return out, e
			}
			if f.GoCast != "" {
				fmt.Fprintf(&goSource, "{ x,e := %s; err=e; v.%s=%s(x) }\n", expr, f.Name, f.GoCast)
			} else {
				fmt.Fprintf(&goSource, "v.%s,err=%s\n", f.Name, expr)
			}
			fmt.Fprintf(&goSource, "if err != nil { return v,fmt.Errorf(\"MIR_BRIDGE_FIELD %s.%s offset=%%d: %%w\", r.Size()-int64(r.Len()),err) }\n", r.Name, f.Name)
		}
		fmt.Fprintln(&goSource, "return v,nil\n}")
	}
	// Concrete overloads are derived from structural applications, not display
	// names. Stage-0 permits one template per callable name, so the counted-view
	// template has its own name and these ordinary wrappers expose its witnesses.
	seenSequences := map[string]bool{}
	for _, r := range def.Records {
		for _, f := range r.Fields {
			if f.Type.Name == "array" && f.Type.Element != nil && f.Type.Element.Name == "uint8" {
				key := fmt.Sprintf("uint8[%d]", f.Type.Extent)
				if !seenSequences[key] {
					seenSequences[key] = true
					fmt.Fprintf(&conceptSource, "Result<%s, BackendError> ReadWire(ref BridgeReader reader, BridgeType<%s> type) { %s values = [0u ...]; for (i in 0..%d) { values[i] = ReadWire(ref reader, BridgeType<uint8>{0})?; } return Result::Ok(values); }\n", key, key, key, f.Type.Extent)
				}
			}
			if f.Type.Name != "BridgeSequence" || len(f.Type.Arguments) != 1 {
				continue
			}
			element := f.Type.Arguments[0].Name
			if seenSequences[element] {
				continue
			}
			seenSequences[element] = true
			fmt.Fprintf(&conceptSource, "Result<BridgeSequence<%s>, BackendError> ReadWire(ref BridgeReader reader, BridgeType<BridgeSequence<%s>> type) { return ReadBridgeSequence<%s>(ref reader, type); }\n", element, element, element)
			fmt.Fprintf(&conceptSource, "Result<void, BackendError> WriteWire(ref const BridgeSequence<%s> value, ref const BridgeReader source, ref BridgeWriter writer) { return WriteBridgeSequence<%s>(ref const value, ref const source, ref writer); }\n", element, element)
		}
	}
	fmt.Fprint(&conceptSource, `Result<int, BackendError> BridgeRoundTrip(ReadOnlySpan<byte> payload, Span<byte> output) {
    BridgeReader reader = BridgeReader{payload, 0};
    WireMachineModule value = ReadWire(ref reader, BridgeType<WireMachineModule>{0})?;
    if (reader.offset != Len(payload)) { return Result::Error(BackendError::BridgeInvalid); }
    BridgeWriter writer = BridgeWriter{output, 0};
    WriteWire(ref const value, ref const reader, ref writer)?;
    return Result::Ok(writer.offset);
}
`)
	fmt.Fprint(&conceptSource, `Result<void, BackendError> ReadBridgeIdentity(ref BridgeReader reader) {
    WireHeader header = ReadWire(ref reader, BridgeType<WireHeader>{0})?;
    for (i in 0..Len(header.magic)) {
        if (header.magic[i] != (BridgeMagic[i] as uint8)) { return Result::Error(BackendError::BridgeVersion); }
    }
    if (header.version != BridgeVersion) { return Result::Error(BackendError::BridgeVersion); }
    uint8[32] expected = MachineBridgeSchemaHashBytes();
    for (i in 0..Len(header.schemaHash)) {
        if (header.schemaHash[i] != expected[i]) { return Result::Error(BackendError::BridgeSchemaMismatch); }
    }
    return Result::Ok();
}
`)
	formatted, err := format.Source(goSource.Bytes())
	if err != nil {
		return out, fmt.Errorf("MIR_BRIDGE_GENERATED_GO: %w", err)
	}
	out.Go = formatted
	conceptFormatted, err := FormatSource("BridgeCodec.concept", conceptSource.String(), DefaultFormatOptions())
	if err != nil {
		return out, err
	}
	out.Concept = []byte(conceptFormatted)
	return out, nil
}

type bridgeGoEmitter struct{ records map[string]bridgeSchemaRecord }

func bridgeFixedWireSize(t bridgeSchemaType) (int, error) {
	switch t.Name {
	case "int", "int32", "uint", "uint32", "bool":
		return 4, nil
	case "uint8":
		return 1, nil
	case "array":
		if t.Element != nil && t.Extent > 0 {
			size, err := bridgeFixedWireSize(*t.Element)
			if err != nil {
				return 0, err
			}
			return t.Extent * size, nil
		}
	}
	return 0, fmt.Errorf("MIR_BRIDGE_SCHEMA_HEADER_NOT_FIXED %+v", t)
}

func (e bridgeGoEmitter) write(t bridgeSchemaType, v string) (string, error) {
	switch t.Name {
	case "uint32", "uint":
		return "w.u32(int(" + v + "))", nil
	case "array":
		if t.Element != nil && t.Element.Name == "uint8" {
			return "func() error {_,err:=w.Write(" + v + "[:]);return err}()", nil
		}
	case "int", "int32":
		return "w.i32(int(" + v + "))", nil
	case "bool":
		return "w.bool(" + v + ")", nil
	case "BridgeText":
		return "w.str(string(" + v + "))", nil
	case "BridgeSequence":
		if len(t.Arguments) != 1 {
			break
		}
		inner, err := e.write(t.Arguments[0], "element")
		if err != nil {
			return "", err
		}
		return "func() error { if len(" + v + ")>MachineBridgeMaxItems{return fmt.Errorf(\"MIR_BRIDGE_COUNT %d\",len(" + v + "))}; if err:=w.u32(len(" + v + ")); err!=nil{return err}; for _,element:=range " + v + " {if err:=" + inner + ";err!=nil{return err}};return nil }()", nil
	}
	if r, ok := e.records[t.Name]; ok {
		if len(r.Cases) > 0 {
			if r.Cases[0].Value != nil {
				return "w.tag(bridgeTags" + t.Name + ", " + v + ")", nil
			}
			return "w.enum(int(" + v + "),len(bridgeTags" + t.Name + "))", nil
		}
		return "w.write" + t.Name + "(" + v + ")", nil
	}
	return "", fmt.Errorf("MIR_BRIDGE_SCHEMA_WIRE_TYPE %+v", t)
}
func (e bridgeGoEmitter) read(t bridgeSchemaType) (string, error) {
	switch t.Name {
	case "uint32", "uint":
		return "r.u32()", nil
	case "array":
		if t.Element != nil && t.Element.Name == "uint8" {
			typ := e.goType(t)
			return "func() (" + typ + ",error) {var v " + typ + ";_,err:=io.ReadFull(r.Reader,v[:]);return v,err}()", nil
		}
	case "int", "int32":
		return "r.i32()", nil
	case "bool":
		return "r.boolean()", nil
	case "BridgeText":
		return "r.str()", nil
	case "BridgeSequence":
		if len(t.Arguments) != 1 {
			break
		}
		inner, err := e.read(t.Arguments[0])
		if err != nil {
			return "", err
		}
		typ := e.goType(t.Arguments[0])
		return "func() ([]" + typ + ",error) {n,err:=r.count();if err!=nil{return nil,err};if n>r.Len(){return nil,fmt.Errorf(\"MIR_BRIDGE_COUNT exceeds remaining input\")};v:=make([]" + typ + ",n);for i:=range v {v[i],err=" + inner + ";if err!=nil{return nil,err}};return v,nil}()", nil
	}
	if r, ok := e.records[t.Name]; ok {
		if len(r.Cases) > 0 {
			if r.Cases[0].Value != nil {
				return "r.tag(bridgeTags" + t.Name + ")", nil
			}
			return "r.enum(len(bridgeTags" + t.Name + "))", nil
		}
		return "r.read" + t.Name + "()", nil
	}
	return "", fmt.Errorf("MIR_BRIDGE_SCHEMA_WIRE_TYPE %+v", t)
}
func (e bridgeGoEmitter) goType(t bridgeSchemaType) string {
	switch t.Name {
	case "uint32", "uint":
		return "uint32"
	case "uint8":
		return "byte"
	case "array":
		if t.Element != nil {
			return fmt.Sprintf("[%d]%s", t.Extent, e.goType(*t.Element))
		}
	case "BridgeText":
		return "string"
	case "int", "int32":
		return "int"
	case "bool":
		return "bool"
	}
	if r, ok := e.records[t.Name]; ok && len(r.Cases) > 0 {
		if r.Cases[0].Value != nil {
			return "string"
		}
		return "int"
	}
	return strings.TrimPrefix(t.Name, "Wire")
}

// Used by tests and the generator CLI to display the semantic fingerprint.
func (g BridgeCodecGeneration) String() string {
	return g.Magic + " v" + strconv.FormatUint(g.Version, 10) + " sha256:" + g.Hash
}
