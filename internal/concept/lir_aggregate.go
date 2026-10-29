package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// These are bounded snapshots of the existing checked type/layout contract.
// They describe source objects, not Win64 classes, copy algorithms or frames.
// All physical realization remains owned by later Cathedral storage/ABI stages.
const (
	lirAggregateMaxLayouts  = 128
	lirAggregateMaxFields   = 64
	lirAggregateMaxDepth    = 16
	lirAggregateMaxElements = 256
	lirAggregateMaxSize     = 1048576
)

type LIRAggregateLayout struct {
	Type            LIRType
	Identity        string // source module + nominal/applied type identity
	Name            string
	Kind            string // record (read-only projections) or struct (mutable projections)
	Representation  string
	Size            int
	Alignment       int
	Fields          []LIRAggregateField
	CopyEligibility string // source-checked value copy; never a raw representation-copy grant
}

type LIRAggregateField struct {
	Name                string
	Type                LIRType
	TypeIdentity        string // full checked semantic type, before scalar representation erasure
	Offset              int
	Size                int
	Alignment           int
	Readonly            bool // checked field const authority, independently of aggregate kind
	DeclarationIdentity string
	ElementType         LIRType
	Extent              int
	Stride              int
	ElementAlignment    int
}

func lirAggregateType(t LIRType) bool { return strings.HasPrefix(string(t), "aggregate:") }
func lirPointer(t LIRType) LIRType    { return LIRType("ptr<" + string(t) + ">") }
func lirArrayValue(t LIRType) (LIRType, int, bool) {
	s := string(t)
	end := strings.IndexByte(s, ']')
	if !strings.HasPrefix(s, "[") || end < 2 {
		return "", 0, false
	}
	n, err := strconv.Atoi(s[1:end])
	if err != nil || n <= 0 || end == len(s)-1 {
		return "", 0, false
	}
	return LIRType(s[end+1:]), n, true
}
func lirCallValue(t LIRType, layouts map[LIRType]LIRAggregateLayout) bool {
	_, record := layouts[t]
	return lirCallScalar(t) || record || lirReferenceType(t)
}
func lirAggregateIdentity(layout LIRAggregateLayout) LIRType {
	layout.Type = ""
	data, _ := json.Marshal(layout)
	sum := sha256.Sum256(data)
	return LIRType("aggregate:" + hex.EncodeToString(sum[:]))
}

type lirAggregateTypes struct {
	enums    []LIREnum
	env      *semanticEnv
	layouts  []LIRAggregateLayout
	known    map[string]LIRType
	visiting map[string]bool
	sources  map[LIRType]Type // checked source types used only during LIR construction
}

func (b *lirBuilder) lowerType(t Type) (LIRType, error) {
	if b.aggregates == nil {
		return lirType(t)
	}
	return b.aggregates.lower(t)
}

func (a *lirAggregateTypes) lower(t Type) (LIRType, error) {
	t = evt1CanonicalType(a.env, t)
	if t.isReference() {
		target, err := a.lower(t.valueType())
		return lirReference(target, !t.Const), err
	}
	if t.isBorrowLike() || t.isOwned() || t.PointerTo != nil {
		return "", fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_AUTHORITY %s", t.String())
	}
	t = t.valueType()
	if decl, ok := a.env.enums[t.Name]; ok {
		return a.lowerEnum(t, decl)
	}
	if t.ArrayElem != nil {
		if len(a.visiting) == 0 {
			// A checked aggregate array literal must reuse its element layout;
			// scalar-only arrays retain the frozen scalar lowering path.
			if _, record := evt1StructView(a.env, evt1CanonicalType(a.env, *t.ArrayElem)); !record {
				return lirType(t)
			}
		}
		if evt1StorageHasRuntimeShape(t) || evt1StorageRank(t) != 1 || t.StorageKind == StorageRaw || t.StorageKind == StorageSparse || t.ArrayLength <= 0 || t.ArrayLength > lirAggregateMaxElements {
			return "", fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_ARRAY %s", t.String())
		}
		element, err := a.lower(*t.ArrayElem)
		return LIRType(fmt.Sprintf("[%d]%s", t.ArrayLength, element)), err
	}
	decl, record := evt1StructView(a.env, t)
	if !record {
		return lirType(t)
	}
	// Reuse structured declaration/application identity, never C symbol
	// spelling (which can fold distinct nominal names to the same string).
	key := decl.Module + "|" + decl.Name
	if t.Application != nil {
		key = evt1GenericApplicationKey(t.Application)
	}
	if known, ok := a.known[key]; ok {
		return known, nil
	}
	if a.visiting[key] || len(a.visiting) >= lirAggregateMaxDepth {
		return "", fmt.Errorf("EVT2_AGGREGATE_DEPTH_EXCEEDED %s", t.String())
	}
	if decl.Class || decl.Ref || decl.Table || decl.Immovable || decl.BitsRepresentation != "" || !evt1TypeCopyable(a.env, t) || evt1TypeHasDrop(a.env, t) {
		return "", fmt.Errorf("EVT2_UNSUPPORTED_AGGREGATE_REPRESENTATION %s", t.String())
	}
	if len(decl.Fields) == 0 || len(decl.Fields) > lirAggregateMaxFields {
		return "", fmt.Errorf("EVT2_AGGREGATE_FIELD_CAPACITY %s", t.String())
	}
	a.visiting[key] = true
	defer delete(a.visiting, key)
	offsets, size, alignment, err := evt1StructFieldOffsets(a.env, decl)
	if err != nil {
		return "", err
	}
	if size <= 0 || size > lirAggregateMaxSize {
		return "", fmt.Errorf("EVT2_AGGREGATE_SIZE_CAPACITY %s", t.String())
	}
	layout := LIRAggregateLayout{Identity: key, Name: t.String(), Representation: "native", Size: size, Alignment: alignment}
	layout.CopyEligibility = "checked_value"
	layout.Kind = "struct"
	if decl.Record {
		layout.Kind = "record"
	}
	if evt1HasCRepr(decl.Attributes) {
		layout.Representation = "C"
	}
	for i, field := range decl.Fields {
		ft, err := a.lower(field.Type)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", t.String(), field.Name, err)
		}
		fs, fa, err := evt1TypeGeometry(a.env, field.Type)
		if err != nil {
			return "", err
		}
		checkedType, _ := json.Marshal(evt1GenericIdentityType(field.Type))
		projected := LIRAggregateField{Name: field.Name, Type: ft, TypeIdentity: digest(checkedType), Offset: offsets[i], Size: fs, Alignment: fa, Readonly: field.Type.Const, DeclarationIdentity: key + "|" + field.Name}
		if field.Type.ArrayElem != nil {
			projected.ElementType, err = a.lower(*field.Type.ArrayElem)
			if err != nil {
				return "", err
			}
			projected.Extent = field.Type.ArrayLength
			projected.Stride, projected.ElementAlignment, err = evt1TypeGeometry(a.env, *field.Type.ArrayElem)
			if err != nil {
				return "", err
			}
		}
		layout.Fields = append(layout.Fields, projected)
	}
	if len(a.layouts) >= lirAggregateMaxLayouts {
		return "", fmt.Errorf("EVT2_AGGREGATE_LAYOUT_CAPACITY")
	}
	layout.Type = lirAggregateIdentity(layout)
	if a.sources == nil {
		a.sources = map[LIRType]Type{}
	}
	a.sources[layout.Type] = t
	a.layouts = append(a.layouts, layout)
	a.known[key] = layout.Type
	return layout.Type, nil
}

func verifyLIRAggregateLayouts(list []LIRAggregateLayout) (map[LIRType]LIRAggregateLayout, error) {
	layouts := map[LIRType]LIRAggregateLayout{}
	identities := map[string]bool{}
	depths := map[LIRType]int{}
	if len(list) > lirAggregateMaxLayouts {
		return nil, fmt.Errorf("LIR_AGGREGATE_LAYOUT_CAPACITY")
	}
	for _, layout := range list {
		if layout.Type != lirAggregateIdentity(layout) || layouts[layout.Type].Type != "" || identities[layout.Identity] || layout.Identity == "" || layout.Name == "" {
			return nil, fmt.Errorf("LIR_STALE_AGGREGATE_LAYOUT %s", layout.Type)
		}
		if layout.CopyEligibility != "checked_value" {
			return nil, fmt.Errorf("LIR_AGGREGATE_COPY_ELIGIBILITY %s", layout.Name)
		}
		if (layout.Representation != "native" && layout.Representation != "C") || (layout.Kind != "record" && layout.Kind != "struct") || layout.Size <= 0 || layout.Size > lirAggregateMaxSize || layout.Alignment <= 0 || layout.Alignment > 16 || layout.Alignment&(layout.Alignment-1) != 0 || layout.Size%layout.Alignment != 0 || len(layout.Fields) == 0 || len(layout.Fields) > lirAggregateMaxFields {
			return nil, fmt.Errorf("LIR_BAD_AGGREGATE_LAYOUT %s", layout.Type)
		}
		end := 0
		depth := 1
		names := map[string]bool{}
		for _, field := range layout.Fields {
			if field.Name == "" || field.TypeIdentity == "" || names[field.Name] || field.Size <= 0 || field.Size > layout.Size || field.Alignment <= 0 || field.Alignment > layout.Alignment || field.Alignment&(field.Alignment-1) != 0 || field.Offset < end || field.Offset > layout.Size-field.Size || field.Offset%field.Alignment != 0 {
				return nil, fmt.Errorf("LIR_BAD_AGGREGATE_FIELD %s.%s", layout.Name, field.Name)
			}
			// Nested descriptors must precede their consumers. This bounds graph
			// traversal, rejects cycles and makes artifact ordering deterministic.
			size, align, err := lirAggregateGeometry(field.Type, layouts, 0)
			if err != nil || size != field.Size || align != field.Alignment {
				return nil, fmt.Errorf("LIR_BAD_AGGREGATE_FIELD_TYPE %s.%s", layout.Name, field.Name)
			}
			if element, extent, array := lirArrayValue(field.Type); array {
				es, ea, e := lirAggregateGeometry(element, layouts, 0)
				if e != nil || field.ElementType != element || field.Extent != extent || field.Stride != es || field.ElementAlignment != ea {
					return nil, fmt.Errorf("LIR_BAD_AGGREGATE_ARRAY %s.%s", layout.Name, field.Name)
				}
			}
			child := field.Type
			arrayDepth := 0
			for {
				element, _, array := lirArrayValue(child)
				if !array {
					break
				}
				arrayDepth++
				child = element
			}
			if candidate := 1 + arrayDepth + depths[child]; candidate > depth {
				depth = candidate
			}
			end = field.Offset + field.Size
			names[field.Name] = true
		}
		if depth > lirAggregateMaxDepth {
			return nil, fmt.Errorf("LIR_AGGREGATE_DEPTH_EXCEEDED %s", layout.Name)
		}
		layouts[layout.Type] = layout
		depths[layout.Type] = depth
		identities[layout.Identity] = true
	}
	return layouts, nil
}

func lirAggregateGeometry(t LIRType, layouts map[LIRType]LIRAggregateLayout, depth int) (int, int, error) {
	if depth >= lirAggregateMaxDepth {
		return 0, 0, fmt.Errorf("LIR_AGGREGATE_DEPTH_EXCEEDED")
	}
	if layout, ok := layouts[t]; ok {
		return layout.Size, layout.Alignment, nil
	}
	if element, extent, ok := lirArrayValue(t); ok {
		size, align, err := lirAggregateGeometry(element, layouts, depth+1)
		if err != nil || extent > lirAggregateMaxElements || size > lirAggregateMaxSize/extent {
			return 0, 0, fmt.Errorf("LIR_BAD_OBJECT_ARRAY")
		}
		return size * extent, align, nil
	}
	if t == "bool" {
		return 1, 1, nil
	}
	if size := lirWidth(t); size > 0 && lirCallScalar(t) {
		return size, size, nil
	}
	return 0, 0, fmt.Errorf("LIR_UNKNOWN_OBJECT_TYPE %s", t)
}

func (b *lirBuilder) recordConstruction(e *StructConstructExpr) (int, LIRType, error) {
	t := e.StructType
	if t.Name == "" {
		t = Type{Name: e.StructName, Kind: TypeStruct}
	}
	lt, err := b.lowerType(t)
	if err != nil {
		return 0, "", err
	}
	decl, ok := evt1StructView(b.env, t)
	if !ok || len(e.Args) != len(decl.Fields) {
		return 0, "", fmt.Errorf("EVT2_INCOMPLETE_RECORD_INIT %s", t.String())
	}
	args := make([]int, len(decl.Fields))
	for i, arg := range e.Args {
		member := i
		if len(e.ArgNames) != 0 {
			member = -1
			for j, field := range decl.Fields {
				if field.Name == e.ArgNames[i] {
					member = j
					break
				}
			}
			if member < 0 {
				return 0, "", fmt.Errorf("EVT2_UNKNOWN_RECORD_FIELD")
			}
		}
		value, actual, err := b.objectInitializer(arg, decl.Fields[member].Type)
		if err != nil {
			return 0, "", err
		}
		expected, err := b.lowerType(decl.Fields[member].Type)
		if err != nil || actual != expected {
			return 0, "", fmt.Errorf("EVT2_RECORD_INIT_TYPE %s", decl.Fields[member].Name)
		}
		args[member] = value
	}
	return b.emit(LIRInstruction{Op: "make_aggregate", Result: 0, Type: lt, Args: args, Slot: -1, Source: e.Span}), lt, nil
}

func (b *lirBuilder) recordWith(e *WithExpr) (int, LIRType, error) {
	base, typ, err := b.expr(e.Base)
	if err != nil {
		return 0, "", err
	}
	var layout LIRAggregateLayout
	if b.aggregates != nil {
		for _, candidate := range b.aggregates.layouts {
			if candidate.Type == typ {
				layout = candidate
				break
			}
		}
	}
	if layout.Kind != "record" {
		return 0, "", fmt.Errorf("EVT2_WITH_REQUIRES_RECORD")
	}
	decl, ok := evt1StructView(b.env, b.aggregates.sources[typ])
	if !ok {
		return 0, "", fmt.Errorf("EVT2_WITH_LAYOUT_MISSING")
	}
	args, members := []int{base}, []int{}
	// Preserve authored evaluation order, including nested with expressions.
	// The native owner later realizes the fresh value; no byte copy is chosen.
	for _, update := range e.Updates {
		member := -1
		for i, field := range layout.Fields {
			if field.Name == update.Name {
				member = i
				break
			}
		}
		if member < 0 {
			return 0, "", fmt.Errorf("EVT2_UNKNOWN_RECORD_FIELD %s", update.Name)
		}
		value, actual, err := b.objectInitializer(update.Value, decl.Fields[member].Type)
		if err != nil {
			return 0, "", err
		}
		if actual != layout.Fields[member].Type {
			return 0, "", fmt.Errorf("EVT2_WITH_FIELD_TYPE %s", update.Name)
		}
		args, members = append(args, value), append(members, member)
	}
	return b.emit(LIRInstruction{Op: "record_with", Result: 0, Type: typ, Args: args, Members: members, Slot: -1, Source: e.Span}), typ, nil
}

func (b *lirBuilder) objectInitializer(expr Expr, expected Type) (int, LIRType, error) {
	array, ok := expr.(*ArrayLiteralExpr)
	if !ok {
		return b.expr(expr)
	}
	lt, err := b.lowerType(expected)
	if err != nil || expected.ArrayElem == nil {
		return 0, "", fmt.Errorf("EVT2_ARRAY_INIT_TYPE")
	}
	var args []int
	for _, item := range array.Elements {
		count := 1
		if repeat, ok := item.(*RepeatInitializer); ok {
			count, item = repeat.ResolvedCount, repeat.Value
			if repeat.FillRemainder {
				count = expected.ArrayLength - len(args)
			}
		}
		if count < 0 || count > expected.ArrayLength-len(args) {
			return 0, "", fmt.Errorf("EVT2_ARRAY_INIT_COUNT")
		}
		for i := 0; i < count; i++ {
			value, _, err := b.objectInitializer(item, *expected.ArrayElem)
			if err != nil {
				return 0, "", err
			}
			args = append(args, value)
		}
	}
	if len(args) != expected.ArrayLength {
		return 0, "", fmt.Errorf("EVT2_ARRAY_INIT_COUNT")
	}
	return b.emit(LIRInstruction{Op: "make_array", Result: 0, Type: lt, Args: args, Slot: -1, Source: array.Span}), lt, nil
}

// Addresses remain typed LIR derivations of source local object slots. No
// integer conversion, native offset selection, copy plan or reference ABI is
// supplied here. References/raw Storage remain unsupported in this slice.
func (b *lirBuilder) objectAddress(expr Expr) (int, Type, error) {
	switch e := expr.(type) {
	case *ParenExpr:
		return b.objectAddress(e.Value)
	case *IndexExpr:
		return b.objectIndexPlace(e)
	case *NameExpr:
		binding, ok := b.names[e.Name]
		if ok && binding.typ.isReference() {
			return binding.value, binding.typ.valueType(), nil
		}
		if !ok || !binding.local {
			return 0, Type{}, fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_LOCATION %s", e.Name)
		}
		lt, err := b.lowerType(binding.typ)
		if err != nil || (!lirAggregateType(lt) && !lirCallScalar(lt)) {
			return 0, Type{}, fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_LOCATION %s", e.Name)
		}
		id := b.emit(LIRInstruction{Op: "slot_address", Result: 0, Type: lirPointer(lt), Slot: binding.slot, Source: e.Span})
		return b.referenceTarget(id, binding.typ), binding.typ, nil
	case *FieldExpr:
		base, t, err := b.objectAddress(e.Receiver)
		if err != nil {
			return 0, Type{}, err
		}
		decl, ok := evt1StructView(b.env, t)
		if !ok {
			return 0, Type{}, fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_FIELD %s", e.Field)
		}
		lt, err := b.lowerType(t)
		if err != nil {
			return 0, Type{}, err
		}
		var layout LIRAggregateLayout
		for _, candidate := range b.aggregates.layouts {
			if candidate.Type == lt {
				layout = candidate
				break
			}
		}
		for i, field := range decl.Fields {
			if field.Name == e.Field {
				id := b.emit(LIRInstruction{Op: "field_address", Result: 0, Type: lirPointer(layout.Fields[i].Type), Args: []int{base}, Slot: -1, Aggregate: lt, Member: i, FrameOffset: layout.Fields[i].Offset, Source: e.Span})
				return b.referenceTarget(id, field.Type), field.Type, nil
			}
		}
		return 0, Type{}, fmt.Errorf("EVT2_UNKNOWN_RECORD_FIELD %s", e.Field)
	}
	return 0, Type{}, fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_LOCATION %T", expr)
}

func (b *lirBuilder) objectIndexAddress(e *IndexExpr) (int, LIRType, error) {
	address, element, err := b.objectIndexPlace(e)
	if err != nil {
		return 0, "", err
	}
	typ, err := b.lowerType(element)
	return address, typ, err
}

func (b *lirBuilder) objectIndexPlace(e *IndexExpr) (int, Type, error) {
	base, array, err := b.objectAddress(e.Base)
	if err != nil {
		return 0, Type{}, err
	}
	if array.ArrayElem == nil || array.ArrayLength <= 0 || len(evt1StorageIndices(e)) != 1 {
		return 0, Type{}, fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_INDEX")
	}
	idx, t, err := b.expr(evt1StorageIndices(e)[0])
	if err != nil || !lirInteger(t) {
		return 0, Type{}, fmt.Errorf("EVT2_OBJECT_INDEX_TYPE")
	}
	decision, facts, err := b.indexDecision(e)
	if err != nil {
		return 0, Type{}, err
	}
	lt, err := b.lowerType(*array.ArrayElem)
	if err != nil {
		return 0, Type{}, err
	}
	size, _, err := evt1TypeGeometry(b.env, *array.ArrayElem)
	if err != nil {
		return 0, Type{}, err
	}
	b.emit(LIRInstruction{Op: "check_index", Result: -1, Type: "void", Args: []int{idx}, Slot: -1, Extent: array.ArrayLength, Source: e.Span, Decision: decision, Facts: facts})
	id := b.emit(LIRInstruction{Op: "index_address", Result: 0, Type: lirPointer(lt), Args: []int{base, idx}, Slot: -1, Extent: array.ArrayLength, Stride: size, Source: e.Span})
	return b.referenceTarget(id, *array.ArrayElem), *array.ArrayElem, nil
}

func (b *lirBuilder) indexDecision(e *IndexExpr) (string, []string, error) {
	for _, d := range b.plan.Decisions {
		if d.Category == "BoundsPlan" && d.SourceSpan == e.Span {
			// A proven fixed index is admissible, but the current object wire
			// contract deliberately retains its check_index safety mechanism.
			if d.Strategy != "PerAccessRuntime" && d.Strategy != "StaticEliminated" {
				return "", nil, fmt.Errorf("EVT2_UNSUPPORTED_BOUNDS_PLAN %s", d.Strategy)
			}
			facts := append([]string(nil), d.Evidence.FactIDs...)
			sort.Strings(facts)
			return d.ID + ":" + d.Strategy, facts, nil
		}
	}
	return "", nil, fmt.Errorf("EVT2_BOUNDS_PLAN_MISSING at %d:%d", e.Span.Line, e.Span.Column)
}
