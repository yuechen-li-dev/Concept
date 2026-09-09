package concept

import (
	"fmt"
	"sort"
	"strings"
)

// SemanticSubjectKind names the stable compiler subject which owns a fact.
// These identities are compile-time only; none of them changes a C layout.
type SemanticSubjectKind string

const (
	SubjectValue                SemanticSubjectKind = "Value"
	SubjectPlace                SemanticSubjectKind = "Place"
	SubjectParameter            SemanticSubjectKind = "Parameter"
	SubjectFunctionResult       SemanticSubjectKind = "FunctionResult"
	SubjectField                SemanticSubjectKind = "Field"
	SubjectCaptureField         SemanticSubjectKind = "CaptureField"
	SubjectAsyncPersistentField SemanticSubjectKind = "AsyncPersistentField"
	SubjectMachineField         SemanticSubjectKind = "MachineField"
	SubjectImportedValueSummary SemanticSubjectKind = "ImportedValueSummary"
)

// SemanticFactTransform is the bounded preservation vocabulary shared by
// validator bindings, MIR facts, proof output, and module result summaries.
type SemanticFactTransform string

const (
	FactTransformPreserve        SemanticFactTransform = "Preserve"
	FactTransformCopy            SemanticFactTransform = "Copy"
	FactTransformMove            SemanticFactTransform = "Move"
	FactTransformRefView         SemanticFactTransform = "RefView"
	FactTransformFieldStore      SemanticFactTransform = "FieldStore"
	FactTransformFieldLoad       SemanticFactTransform = "FieldLoad"
	FactTransformParameter       SemanticFactTransform = "Parameter"
	FactTransformReturn          SemanticFactTransform = "Return"
	FactTransformCarrierWrap     SemanticFactTransform = "CarrierWrap"
	FactTransformCarrierUnwrap   SemanticFactTransform = "CarrierUnwrap"
	FactTransformCapture         SemanticFactTransform = "Capture"
	FactTransformAsyncPersist    SemanticFactTransform = "AsyncPersist"
	FactTransformMachinePersist  SemanticFactTransform = "MachinePersist"
	FactTransformAddOffset       SemanticFactTransform = "AddOffset"
	FactTransformCommonAlignment SemanticFactTransform = "CommonAlignment"
	FactTransformInitialize      SemanticFactTransform = "Initialize"
	FactTransformJoin            SemanticFactTransform = "Join"
	FactTransformOpaqueBoundary  SemanticFactTransform = "OpaqueBoundary"
)

type SemanticFactTransportStep struct {
	Transform SemanticFactTransform `json:"transform"`
	From      string                `json:"from,omitempty"`
	Through   string                `json:"through,omitempty"`
	Detail    string                `json:"detail,omitempty"`
}

// SemanticKnownInt distinguishes a known zero from Unknown.
type SemanticKnownInt struct {
	Known bool `json:"known"`
	Value int  `json:"value,omitempty"`
}

// SemanticValueFacts is the one internal value/place transport record.
// Existing Span and Tensor records remain subsystem validators and are bridged
// into this record instead of being replaced by a parallel proof engine.
type SemanticValueFacts struct {
	SubjectKind         SemanticSubjectKind            `json:"subject_kind"`
	Subject             string                         `json:"subject,omitempty"`
	Type                string                         `json:"type,omitempty"`
	AddressSpace        string                         `json:"address_space,omitempty"`
	RegionOrigin        string                         `json:"region_origin,omitempty"`
	ParentRegion        string                         `json:"parent_region,omitempty"`
	RelativeOffset      SemanticKnownInt               `json:"relative_offset"`
	ByteExtent          SemanticKnownInt               `json:"byte_extent"`
	Alignment           SemanticKnownInt               `json:"alignment"`
	Scalar              SemanticKnownInt               `json:"scalar"`
	Contiguous          SemanticFactCertainty          `json:"contiguous,omitempty"`
	Bounded             SemanticFactCertainty          `json:"bounded,omitempty"`
	Mutability          string                         `json:"mutability,omitempty"`
	Provenance          evt1LifetimeProvenance         `json:"-"`
	Rank                SemanticKnownInt               `json:"rank"`
	Shape               []StorageDimension             `json:"shape,omitempty"`
	NoAllocation        SemanticFactCertainty          `json:"no_allocation,omitempty"`
	NoCopy              SemanticFactCertainty          `json:"no_copy,omitempty"`
	NoOwnershipTransfer SemanticFactCertainty          `json:"no_ownership_transfer,omitempty"`
	HostAccessible      SemanticFactCertainty          `json:"host_accessible,omitempty"`
	Initialized         SemanticFactCertainty          `json:"initialized,omitempty"`
	Fields              map[string]*SemanticValueFacts `json:"fields,omitempty"`
	Transport           []SemanticFactTransportStep    `json:"transport,omitempty"`
}

func cloneSemanticValueFacts(in *SemanticValueFacts) *SemanticValueFacts {
	if in == nil {
		return nil
	}
	out := *in
	out.Shape = append([]StorageDimension{}, in.Shape...)
	out.Transport = append([]SemanticFactTransportStep{}, in.Transport...)
	if len(in.Fields) != 0 {
		out.Fields = make(map[string]*SemanticValueFacts, len(in.Fields))
		for name, field := range in.Fields {
			out.Fields[name] = cloneSemanticValueFacts(field)
		}
	}
	return &out
}

func evt1AddressSpaceName(t Type) string {
	if t.Kind == TypeAddress && len(t.TypeArgs) == 1 {
		return t.TypeArgs[0].Name
	}
	return ""
}

func evt1SemanticStorageOrigin(expr Expr) string {
	for {
		switch e := expr.(type) {
		case *RefExpr:
			expr = e.Value
			continue
		case *MoveExpr:
			expr = e.Value
			continue
		case *ParenExpr:
			expr = e.Value
			continue
		}
		break
	}
	return "storage:" + evt1ExprIdentity(expr)
}

func evt1ParameterSemanticValueFacts(env *semanticEnv, name string, t Type, provenance evt1LifetimeProvenance) *SemanticValueFacts {
	if span := evt1ParameterSpanFacts(env, name, t, provenance); span != nil {
		out := semanticValueFactsFromSpan(t, span)
		out.SubjectKind, out.Subject = SubjectParameter, name
		return out
	}
	return &SemanticValueFacts{SubjectKind: SubjectParameter, Subject: name, Type: t.String(), AddressSpace: evt1AddressSpaceName(t), Provenance: provenance}
}

func evt1TypeCarriesSemanticValueFacts(env *semanticEnv, t Type, visiting map[string]bool) bool {
	if t.Kind == TypeAddress || t.Kind == TypeTypedStorage || t.Kind == TypeSpan || t.Kind == TypeTensor || t.Kind == TypeCallable || t.Kind == TypeCallback || t.Kind == TypeAsync || t.isReference() {
		return true
	}
	if t.Kind != TypeStruct || visiting[t.Name] {
		return false
	}
	visiting[t.Name] = true
	defer delete(visiting, t.Name)
	if decl, ok := env.structs[t.Name]; ok {
		for _, field := range decl.Fields {
			if evt1TypeCarriesSemanticValueFacts(env, field.Type, visiting) {
				return true
			}
		}
	}
	return false
}

func transportSemanticValueFacts(in *SemanticValueFacts, transform SemanticFactTransform, from, through, detail string) *SemanticValueFacts {
	out := cloneSemanticValueFacts(in)
	if out == nil {
		return nil
	}
	out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: transform, From: from, Through: through, Detail: detail})
	return out
}

// SemanticSummaryExpr is a small deterministic expression language for module
// result contracts. It can select an argument/field/fact, add byte offsets,
// and compute common guaranteed alignment. Nothing here is a general CAS.
type SemanticSummaryExpr struct {
	Kind           string               `json:"kind"`
	ParameterIndex int                  `json:"parameter_index,omitempty"`
	FieldPath      []string             `json:"field_path,omitempty"`
	Fact           string               `json:"fact,omitempty"`
	Constant       int                  `json:"constant,omitempty"`
	Left           *SemanticSummaryExpr `json:"left,omitempty"`
	Right          *SemanticSummaryExpr `json:"right,omitempty"`
}

type SemanticFieldFactSummary struct {
	Name  string                   `json:"name"`
	Facts SemanticValueFactSummary `json:"facts"`
}

type SemanticValueFactSummary struct {
	Source              *SemanticSummaryExpr        `json:"source,omitempty"`
	Scalar              *SemanticSummaryExpr        `json:"scalar,omitempty"`
	AddressSpace        string                      `json:"address_space,omitempty"`
	RegionOrigin        *SemanticSummaryExpr        `json:"region_origin,omitempty"`
	ParentRegion        *SemanticSummaryExpr        `json:"parent_region,omitempty"`
	RelativeOffset      *SemanticSummaryExpr        `json:"relative_offset,omitempty"`
	ByteExtent          *SemanticSummaryExpr        `json:"byte_extent,omitempty"`
	Alignment           *SemanticSummaryExpr        `json:"alignment,omitempty"`
	Contiguous          SemanticFactCertainty       `json:"contiguous,omitempty"`
	Bounded             SemanticFactCertainty       `json:"bounded,omitempty"`
	NoAllocation        SemanticFactCertainty       `json:"no_allocation,omitempty"`
	NoCopy              SemanticFactCertainty       `json:"no_copy,omitempty"`
	NoOwnershipTransfer SemanticFactCertainty       `json:"no_ownership_transfer,omitempty"`
	HostAccessible      SemanticFactCertainty       `json:"host_accessible,omitempty"`
	ProvenanceKind      string                      `json:"provenance_kind,omitempty"`
	ProvenanceParameter int                         `json:"provenance_parameter,omitempty"`
	Fields              []SemanticFieldFactSummary  `json:"fields,omitempty"`
	Transport           []SemanticFactTransportStep `json:"transport,omitempty"`
}

type SemanticFunctionFactSummary struct {
	Operation string                   `json:"operation"`
	Signature string                   `json:"signature"`
	Result    SemanticValueFactSummary `json:"result"`
	Origin    SemanticFactOrigin       `json:"origin"`
}

func semanticSummaryParameter(index int) *SemanticSummaryExpr {
	return &SemanticSummaryExpr{Kind: "Parameter", ParameterIndex: index}
}

func semanticSummaryFact(base *SemanticSummaryExpr, fact string) *SemanticSummaryExpr {
	return &SemanticSummaryExpr{Kind: "Fact", Fact: fact, Left: base}
}

func semanticSummaryField(base *SemanticSummaryExpr, field string) *SemanticSummaryExpr {
	return &SemanticSummaryExpr{Kind: "Field", FieldPath: []string{field}, Left: base}
}

func semanticSummaryUnknown(summary SemanticValueFactSummary) bool {
	return summary.Source == nil && summary.Scalar == nil && summary.RegionOrigin == nil && len(summary.Fields) == 0 && summary.AddressSpace == ""
}

func semanticSummaryEqual(a, b SemanticValueFactSummary) bool {
	return fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b)
}

func joinSemanticSummaries(a, b SemanticValueFactSummary) SemanticValueFactSummary {
	if semanticSummaryUnknown(a) {
		return b // Result::Error/Option::None contributes no successful payload.
	}
	if semanticSummaryUnknown(b) {
		return a
	}
	if semanticSummaryEqual(a, b) {
		return a
	}
	// A branch join may keep invariants only when both sides state the same
	// guarantee. Origin/interval are intentionally not guessed.
	return SemanticValueFactSummary{
		AddressSpace:        a.AddressSpace,
		Contiguous:          joinCertainty(a.Contiguous, b.Contiguous),
		Bounded:             joinCertainty(a.Bounded, b.Bounded),
		NoAllocation:        joinCertainty(a.NoAllocation, b.NoAllocation),
		NoCopy:              joinCertainty(a.NoCopy, b.NoCopy),
		NoOwnershipTransfer: joinCertainty(a.NoOwnershipTransfer, b.NoOwnershipTransfer),
		HostAccessible:      joinCertainty(a.HostAccessible, b.HostAccessible),
		Transport:           []SemanticFactTransportStep{{Transform: FactTransformJoin, Detail: "different branch facts conservatively joined"}},
	}
}

func joinCertainty(a, b SemanticFactCertainty) SemanticFactCertainty {
	if a != "" && a == b {
		return a
	}
	return FactUnknown
}

func evt1DeriveSemanticFactSummaries(env *semanticEnv, functions []FunctionDecl, templates []TemplateDecl) {
	state := map[string]uint8{}
	var derive func(FunctionDecl) SemanticValueFactSummary
	derive = func(fn FunctionDecl) SemanticValueFactSummary {
		key := evt1FunctionProvenanceKey(fn)
		if imported, ok := env.importedFactSummaries[key]; ok {
			env.resultFactSummaries[key] = imported.Result
			return imported.Result
		}
		if env.importedFactAuthority[key] {
			result := SemanticValueFactSummary{Transport: []SemanticFactTransportStep{{Transform: FactTransformOpaqueBoundary, Through: fn.Name, Detail: "imported artifact has no semantic value result summary"}}}
			env.resultFactSummaries[key] = result
			return result
		}
		if state[key] == 2 {
			return env.resultFactSummaries[key]
		}
		if state[key] == 1 || fn.Body == nil {
			return SemanticValueFactSummary{Transport: []SemanticFactTransportStep{{Transform: FactTransformOpaqueBoundary, Through: fn.Name, Detail: "no semantic value result summary"}}}
		}
		state[key] = 1
		params := map[string]int{}
		for i, p := range fn.Params {
			params[p.Name] = i
		}
		result, found := evt1DeriveBlockFactSummary(env, *fn.Body, params, derive)
		if !found {
			result = SemanticValueFactSummary{}
		}
		result.Transport = append(result.Transport, SemanticFactTransportStep{Transform: FactTransformReturn, Through: fn.Name})
		env.resultFactSummaries[key] = result
		state[key] = 2
		return result
	}
	for _, fn := range functions {
		derive(fn)
	}
	for _, template := range templates {
		fn := FunctionDecl{Name: template.Name, ReturnType: template.ReturnType, Params: template.Params, Body: template.Body, Span: template.Span}
		if env.importedFactAuthority["template:"+template.Name] {
			if _, ok := env.templateFactSummaries[template.Name]; !ok {
				env.templateFactSummaries[template.Name] = SemanticValueFactSummary{Transport: []SemanticFactTransportStep{{Transform: FactTransformOpaqueBoundary, Through: template.Name, Detail: "imported artifact has no semantic value result summary"}}}
			}
		} else {
			env.templateFactSummaries[template.Name] = derive(fn)
		}
	}
}

func evt1DeriveBlockFactSummary(env *semanticEnv, block Block, params map[string]int, derive func(FunctionDecl) SemanticValueFactSummary) (SemanticValueFactSummary, bool) {
	locals := map[string]SemanticValueFactSummary{}
	var result SemanticValueFactSummary
	found := false
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			locals[s.Name] = evt1DeriveExprFactSummary(env, s.Value, params, locals, derive)
		case *AssignStmt:
			if name, ok := s.Target.(*NameExpr); ok {
				locals[name.Name] = evt1DeriveExprFactSummary(env, s.Value, params, locals, derive)
			}
		case *ReturnStmt:
			if s.Value == nil {
				continue
			}
			candidate := evt1DeriveExprFactSummary(env, s.Value, params, locals, derive)
			if found {
				result = joinSemanticSummaries(result, candidate)
			} else {
				result, found = candidate, true
			}
		}
	}
	return result, found
}

func evt1DeriveExprFactSummary(env *semanticEnv, expr Expr, params map[string]int, locals map[string]SemanticValueFactSummary, derive func(FunctionDecl) SemanticValueFactSummary) SemanticValueFactSummary {
	if expr == nil {
		return SemanticValueFactSummary{}
	}
	switch e := expr.(type) {
	case *NameExpr:
		if index, ok := params[e.Name]; ok {
			return SemanticValueFactSummary{Source: semanticSummaryParameter(index), Scalar: semanticSummaryParameter(index), ProvenanceKind: string(evt1ProvenanceParameter), ProvenanceParameter: index, Transport: []SemanticFactTransportStep{{Transform: FactTransformParameter, From: e.Name}}}
		}
		return locals[e.Name]
	case *IntLiteral:
		return SemanticValueFactSummary{Scalar: &SemanticSummaryExpr{Kind: "Constant", Constant: e.Value}}
	case *ParenExpr:
		return evt1DeriveExprFactSummary(env, e.Value, params, locals, derive)
	case *MoveExpr:
		out := evt1DeriveExprFactSummary(env, e.Value, params, locals, derive)
		out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformMove})
		return out
	case *RefExpr:
		out := evt1DeriveExprFactSummary(env, e.Value, params, locals, derive)
		out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformRefView})
		return out
	case *FailureExpr:
		out := evt1DeriveExprFactSummary(env, e.Value, params, locals, derive)
		out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformCarrierUnwrap, Through: e.Op})
		return out
	case *ConstructExpr:
		if (e.EnumName == "Result" && e.VariantName == "Ok" || e.EnumName == "Option" && e.VariantName == "Some") && len(e.Args) != 0 {
			out := evt1DeriveExprFactSummary(env, e.Args[0], params, locals, derive)
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformCarrierWrap, Through: e.EnumName})
			return out
		}
	case *IfExpr:
		return joinSemanticSummaries(evt1DeriveExprFactSummary(env, e.Then, params, locals, derive), evt1DeriveExprFactSummary(env, e.Else, params, locals, derive))
	case *FieldExpr:
		base := evt1DeriveExprFactSummary(env, e.Receiver, params, locals, derive)
		if base.Source != nil {
			return SemanticValueFactSummary{Source: semanticSummaryField(base.Source, e.Field), Scalar: semanticSummaryField(base.Source, e.Field), Transport: append(base.Transport, SemanticFactTransportStep{Transform: FactTransformFieldLoad, Through: e.Field})}
		}
		for _, field := range base.Fields {
			if field.Name == e.Field {
				out := field.Facts
				out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformFieldLoad, Through: e.Field})
				return out
			}
		}
	case *BinaryExpr:
		left := evt1DeriveExprFactSummary(env, e.Left, params, locals, derive)
		rightScalar := evt1DeriveScalarSummary(e.Right, params, locals)
		if e.Op == "+" && left.Source != nil {
			out := left
			if out.RegionOrigin == nil {
				out.RegionOrigin = semanticSummaryFact(left.Source, "RegionOrigin")
			}
			out.RelativeOffset = &SemanticSummaryExpr{Kind: "Add", Left: semanticSummaryFact(left.Source, "RelativeOffset"), Right: rightScalar}
			out.Alignment = &SemanticSummaryExpr{Kind: "CommonAlignment", Left: semanticSummaryFact(left.Source, "Alignment"), Right: rightScalar}
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformAddOffset})
			return out
		}
	case *StructConstructExpr:
		return evt1DeriveStructFactSummary(env, e, params, locals, derive)
	case *CallExpr:
		if e.Callee == "Initialize" && len(e.Args) != 0 {
			out := evt1DeriveExprFactSummary(env, e.Args[0], params, locals, derive)
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformInitialize})
			return out
		}
		for _, fn := range env.functions[e.Callee] {
			if len(fn.Params) == len(e.Args) {
				return derive(fn)
			}
		}
	case *TemplateCallExpr:
		if e.Callee == "AddressOf" && len(e.Args) == 1 {
			return SemanticValueFactSummary{AddressSpace: e.TypeArg.Name, RegionOrigin: &SemanticSummaryExpr{Kind: "StorageOrigin", Fact: strings.TrimPrefix(evt1SemanticStorageOrigin(e.Args[0]), "storage:")}, RelativeOffset: &SemanticSummaryExpr{Kind: "Constant"}, HostAccessible: map[bool]SemanticFactCertainty{true: FactProven, false: FactUnknown}[e.TypeArg.Name == "SystemMemory"]}
		}
		if e.Callee == "AddressFromBits" {
			return SemanticValueFactSummary{AddressSpace: e.TypeArg.Name, ProvenanceKind: string(evt1ProvenanceUnknown), Transport: []SemanticFactTransportStep{{Transform: FactTransformOpaqueBoundary, Through: "AddressFromBits", Detail: "bits establish no region origin"}}}
		}
		if summary, ok := env.templateFactSummaries[e.Callee]; ok {
			return summary
		}
	}
	return SemanticValueFactSummary{}
}

func evt1DeriveScalarSummary(expr Expr, params map[string]int, locals map[string]SemanticValueFactSummary) *SemanticSummaryExpr {
	switch e := expr.(type) {
	case *IntLiteral:
		return &SemanticSummaryExpr{Kind: "Constant", Constant: e.Value}
	case *NameExpr:
		if index, ok := params[e.Name]; ok {
			return semanticSummaryParameter(index)
		}
		return locals[e.Name].Scalar
	case *ParenExpr:
		return evt1DeriveScalarSummary(e.Value, params, locals)
	case *FieldExpr:
		if name, ok := e.Receiver.(*NameExpr); ok {
			if index, found := params[name.Name]; found {
				return semanticSummaryField(semanticSummaryParameter(index), e.Field)
			}
		}
	case *BinaryExpr:
		if e.Op == "+" {
			return &SemanticSummaryExpr{Kind: "Add", Left: evt1DeriveScalarSummary(e.Left, params, locals), Right: evt1DeriveScalarSummary(e.Right, params, locals)}
		}
	}
	return nil
}

func evt1DeriveStructFactSummary(env *semanticEnv, e *StructConstructExpr, params map[string]int, locals map[string]SemanticValueFactSummary, derive func(FunctionDecl) SemanticValueFactSummary) SemanticValueFactSummary {
	decl, ok := env.structs[e.StructName]
	if !ok {
		return SemanticValueFactSummary{}
	}
	out := SemanticValueFactSummary{}
	var addressField *SemanticFieldFactSummary
	var byteFields []*SemanticFieldFactSummary
	for i, field := range decl.Fields {
		if i >= len(e.Args) {
			break
		}
		facts := evt1DeriveExprFactSummary(env, e.Args[i], params, locals, derive)
		facts.Scalar = evt1DeriveScalarSummary(e.Args[i], params, locals)
		facts.Transport = append(facts.Transport, SemanticFactTransportStep{Transform: FactTransformFieldStore, Through: field.Name})
		entry := SemanticFieldFactSummary{Name: field.Name, Facts: facts}
		out.Fields = append(out.Fields, entry)
		if field.Type.Kind == TypeAddress {
			if len(field.Type.TypeArgs) == 1 {
				facts.AddressSpace = field.Type.TypeArgs[0].Name
			}
			if facts.Source != nil && facts.RegionOrigin == nil {
				facts.RegionOrigin = semanticSummaryFact(facts.Source, "RegionOrigin")
			}
			out.Fields[len(out.Fields)-1].Facts = facts
			copy := entry
			copy.Facts = facts
			addressField = &copy
		}
		if evt1ByteDisplacement(field.Type) {
			copy := entry
			byteFields = append(byteFields, &copy)
		}
	}
	if addressField != nil && len(byteFields) >= 2 {
		address := addressField.Facts
		out.Source = address.Source
		out.AddressSpace = address.AddressSpace
		out.RegionOrigin = address.RegionOrigin
		if out.RegionOrigin == nil && address.Source != nil {
			out.RegionOrigin = semanticSummaryFact(address.Source, "RegionOrigin")
		}
		out.ParentRegion = out.RegionOrigin
		out.RelativeOffset = address.RelativeOffset
		out.ByteExtent = byteFields[0].Facts.Scalar
		out.Alignment = byteFields[len(byteFields)-1].Facts.Scalar
		out.Contiguous, out.Bounded = FactProven, FactProven
		if call, ok := e.Args[len(e.Args)-1].(*CallExpr); ok && evt1CallHasEuclideanAlignmentSemantics(env, call) {
			out.Alignment = &SemanticSummaryExpr{Kind: "CommonAlignment", Left: evt1DeriveScalarSummary(call.Args[0], params, locals), Right: evt1DeriveScalarSummary(call.Args[1], params, locals)}
		}
	}
	return out
}

// evt1CallHasEuclideanAlignmentSemantics recognizes the bounded Euclidean
// algorithm by typed body shape. The semantic relation is therefore derived
// from ordinary source, not from a library function name.
func evt1CallHasEuclideanAlignmentSemantics(env *semanticEnv, call *CallExpr) bool {
	if call == nil || len(call.Args) != 2 {
		return false
	}
	for _, fn := range env.functions[call.Callee] {
		if len(fn.Params) != 2 || fn.Body == nil || len(fn.Body.Statements) != 2 || !evt1ByteDisplacement(fn.Params[0].Type) || !evt1ByteDisplacement(fn.Params[1].Type) {
			continue
		}
		loop, loopOK := fn.Body.Statements[0].(*WhileStmt)
		ret, returnOK := fn.Body.Statements[1].(*ReturnStmt)
		if !loopOK || !returnOK || loop.Bound == nil || len(loop.Body.Statements) != 3 || !evt1NameIs(ret.Value, fn.Params[0].Name) {
			continue
		}
		condition, ok := loop.Condition.(*BinaryExpr)
		if !ok || condition.Op != "!=" || !evt1NameIs(condition.Left, fn.Params[1].Name) || !evt1IntegerIs(condition.Right, 0) {
			continue
		}
		remainder, ok := loop.Body.Statements[0].(*VarDecl)
		leftAssign, leftOK := loop.Body.Statements[1].(*AssignStmt)
		rightAssign, rightOK := loop.Body.Statements[2].(*AssignStmt)
		modulo, moduloOK := remainder.Value.(*BinaryExpr)
		if !ok || !leftOK || !rightOK || !moduloOK || modulo.Op != "%" || !evt1NameIs(modulo.Left, fn.Params[0].Name) || !evt1NameIs(modulo.Right, fn.Params[1].Name) {
			continue
		}
		if evt1NameIs(leftAssign.Target, fn.Params[0].Name) && evt1NameIs(leftAssign.Value, fn.Params[1].Name) && evt1NameIs(rightAssign.Target, fn.Params[1].Name) && evt1NameIs(rightAssign.Value, remainder.Name) {
			return true
		}
	}
	return false
}

func evt1NameIs(expr Expr, name string) bool {
	value, ok := expr.(*NameExpr)
	return ok && value.Name == name
}

func evt1IntegerIs(expr Expr, value int) bool {
	literal, ok := expr.(*IntLiteral)
	return ok && literal.Value == value
}

type semanticSummaryValue struct {
	facts   *SemanticValueFacts
	known   bool
	integer int
	text    string
}

func evalSemanticSummaryExpr(expr *SemanticSummaryExpr, args []*SemanticValueFacts) semanticSummaryValue {
	if expr == nil {
		return semanticSummaryValue{}
	}
	switch expr.Kind {
	case "Parameter":
		if expr.ParameterIndex >= 0 && expr.ParameterIndex < len(args) {
			facts := cloneSemanticValueFacts(args[expr.ParameterIndex])
			if facts != nil {
				return semanticSummaryValue{facts: facts, known: facts.Scalar.Known, integer: facts.Scalar.Value}
			}
		}
	case "Field":
		base := evalSemanticSummaryExpr(expr.Left, args)
		facts := base.facts
		for _, part := range expr.FieldPath {
			if facts == nil || facts.Fields == nil {
				return semanticSummaryValue{}
			}
			facts = facts.Fields[part]
		}
		if facts != nil {
			return semanticSummaryValue{facts: cloneSemanticValueFacts(facts), known: facts.Scalar.Known, integer: facts.Scalar.Value}
		}
	case "Fact":
		base := evalSemanticSummaryExpr(expr.Left, args).facts
		if base == nil {
			return semanticSummaryValue{}
		}
		switch expr.Fact {
		case "RegionOrigin":
			return semanticSummaryValue{known: base.RegionOrigin != "", text: base.RegionOrigin}
		case "RelativeOffset":
			return semanticSummaryValue{known: base.RelativeOffset.Known, integer: base.RelativeOffset.Value}
		case "Alignment":
			return semanticSummaryValue{known: base.Alignment.Known, integer: base.Alignment.Value}
		case "ByteExtent":
			return semanticSummaryValue{known: base.ByteExtent.Known, integer: base.ByteExtent.Value}
		}
	case "Constant":
		return semanticSummaryValue{known: true, integer: expr.Constant}
	case "StorageOrigin":
		return semanticSummaryValue{known: true, text: "storage:" + strings.TrimPrefix(expr.Fact, "ref ")}
	case "Add":
		left, right := evalSemanticSummaryExpr(expr.Left, args), evalSemanticSummaryExpr(expr.Right, args)
		if left.known && right.known {
			return semanticSummaryValue{known: true, integer: left.integer + right.integer}
		}
	case "CommonAlignment":
		left, right := evalSemanticSummaryExpr(expr.Left, args), evalSemanticSummaryExpr(expr.Right, args)
		if left.known && right.known {
			return semanticSummaryValue{known: true, integer: evt1GCD(left.integer, right.integer)}
		}
	}
	return semanticSummaryValue{}
}

func instantiateSemanticFactSummary(summary SemanticValueFactSummary, args []*SemanticValueFacts, through string) *SemanticValueFacts {
	var out *SemanticValueFacts
	if source := evalSemanticSummaryExpr(summary.Source, args); source.facts != nil {
		out = source.facts
	} else {
		out = &SemanticValueFacts{SubjectKind: SubjectFunctionResult}
	}
	if summary.AddressSpace != "" {
		out.AddressSpace = summary.AddressSpace
	}
	if value := evalSemanticSummaryExpr(summary.RegionOrigin, args); value.known {
		out.RegionOrigin = value.text
	}
	if value := evalSemanticSummaryExpr(summary.ParentRegion, args); value.known {
		out.ParentRegion = value.text
	}
	if value := evalSemanticSummaryExpr(summary.RelativeOffset, args); value.known {
		out.RelativeOffset = SemanticKnownInt{Known: true, Value: value.integer}
		if summary.RelativeOffset != nil && summary.RelativeOffset.Kind == "Add" {
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformAddOffset, Through: through, Detail: fmt.Sprintf("derived relative byte offset = %d", value.integer)})
		}
	} else if summary.RelativeOffset != nil {
		out.RelativeOffset = SemanticKnownInt{}
	}
	if value := evalSemanticSummaryExpr(summary.ByteExtent, args); value.known {
		out.ByteExtent = SemanticKnownInt{Known: true, Value: value.integer}
	} else if summary.ByteExtent != nil {
		out.ByteExtent = SemanticKnownInt{}
	}
	if value := evalSemanticSummaryExpr(summary.Alignment, args); value.known {
		out.Alignment = SemanticKnownInt{Known: true, Value: value.integer}
		if summary.Alignment != nil && summary.Alignment.Kind == "CommonAlignment" {
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformCommonAlignment, Through: through, Detail: fmt.Sprintf("derived guaranteed alignment = %d", value.integer)})
		}
	} else if summary.Alignment != nil {
		out.Alignment = SemanticKnownInt{}
	}
	if value := evalSemanticSummaryExpr(summary.Scalar, args); value.known {
		out.Scalar = SemanticKnownInt{Known: true, Value: value.integer}
	}
	out.Contiguous = preferSummaryCertainty(summary.Contiguous, out.Contiguous)
	out.Bounded = preferSummaryCertainty(summary.Bounded, out.Bounded)
	out.NoAllocation = preferSummaryCertainty(summary.NoAllocation, out.NoAllocation)
	out.NoCopy = preferSummaryCertainty(summary.NoCopy, out.NoCopy)
	out.NoOwnershipTransfer = preferSummaryCertainty(summary.NoOwnershipTransfer, out.NoOwnershipTransfer)
	out.HostAccessible = preferSummaryCertainty(summary.HostAccessible, out.HostAccessible)
	if summary.ProvenanceKind == string(evt1ProvenanceParameter) && summary.ProvenanceParameter >= 0 && summary.ProvenanceParameter < len(args) && args[summary.ProvenanceParameter] != nil {
		out.Provenance = args[summary.ProvenanceParameter].Provenance
	}
	if len(summary.Fields) != 0 {
		out.Fields = map[string]*SemanticValueFacts{}
		for _, field := range summary.Fields {
			out.Fields[field.Name] = instantiateSemanticFactSummary(field.Facts, args, through+"."+field.Name)
		}
	}
	out.Transport = append(out.Transport, summary.Transport...)
	out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformReturn, Through: through})
	return out
}

func preferSummaryCertainty(summary, existing SemanticFactCertainty) SemanticFactCertainty {
	if summary != "" {
		return summary
	}
	return existing
}

func evt1SemanticFactsForExpr(env *semanticEnv, scope *evt1Scope, expr Expr, t Type) *SemanticValueFacts {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *NameExpr:
		if binding, ok := scope.lookup(e.Name); ok {
			if !evt1TypeCarriesSemanticValueFacts(env, t, map[string]bool{}) {
				if binding.valueFacts != nil && binding.valueFacts.Scalar.Known {
					return &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), Scalar: binding.valueFacts.Scalar}
				}
				return nil
			}
			return transportSemanticValueFacts(binding.valueFacts, FactTransformCopy, e.Name, "local binding", "ordinary value copy preserves applicable facts")
		}
	case *IntLiteral:
		return &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), Scalar: SemanticKnownInt{Known: true, Value: e.Value}}
	case *ParenExpr:
		return evt1SemanticFactsForExpr(env, scope, e.Value, t)
	case *MoveExpr:
		return transportSemanticValueFacts(evt1SemanticFactsForExpr(env, scope, e.Value, t), FactTransformMove, exprLabel(e.Value), "move", "facts transfer to destination; source state is tracked separately")
	case *RefExpr:
		return transportSemanticValueFacts(evt1SemanticFactsForExpr(env, scope, e.Value, t), FactTransformRefView, exprLabel(e.Value), "ref", "view preserves source place facts")
	case *FailureExpr:
		return transportSemanticValueFacts(evt1SemanticFactsForExpr(env, scope, e.Value, e.ResolvedType), FactTransformCarrierUnwrap, exprLabel(e.Value), e.Op, "successful payload facts preserved")
	case *FieldExpr:
		var base *SemanticValueFacts
		if name, ok := e.Receiver.(*NameExpr); ok {
			if binding, found := scope.lookup(name.Name); found {
				base = cloneSemanticValueFacts(binding.valueFacts)
			}
		}
		if base == nil {
			receiverType, _ := validateExpr(env, scope, e.Receiver, nil, false)
			base = evt1SemanticFactsForExpr(env, scope, e.Receiver, receiverType)
		}
		if base != nil && base.Fields != nil {
			return transportSemanticValueFacts(base.Fields[e.Field], FactTransformFieldLoad, base.Subject, e.Field, "field-sensitive value facts recovered")
		}
	case *StructConstructExpr:
		decl, ok := env.structs[e.StructName]
		if !ok {
			break
		}
		out := &SemanticValueFacts{SubjectKind: SubjectValue, Type: e.StructName, Fields: map[string]*SemanticValueFacts{}}
		for i, field := range decl.Fields {
			if i >= len(e.Args) {
				break
			}
			fieldFacts := evt1SemanticFactsForExpr(env, scope, e.Args[i], field.Type)
			out.Fields[field.Name] = transportSemanticValueFacts(fieldFacts, FactTransformFieldStore, exprLabel(e.Args[i]), field.Name, "aggregate embedding preserves field facts")
		}
		evt1DeriveConcreteAggregateGeometry(decl, out)
		return out
	case *ConstructExpr:
		if (e.EnumName == "Result" && e.VariantName == "Ok" || e.EnumName == "Option" && e.VariantName == "Some") && len(e.Args) != 0 {
			return transportSemanticValueFacts(evt1SemanticFactsForExpr(env, scope, e.Args[0], t), FactTransformCarrierWrap, exprLabel(e.Args[0]), e.EnumName, "payload facts remain compile-time facts")
		}
	case *IfExpr:
		return joinSemanticValueFacts(evt1SemanticFactsForExpr(env, scope, e.Then, t), evt1SemanticFactsForExpr(env, scope, e.Else, t))
	case *BinaryExpr:
		left := evt1SemanticFactsForExpr(env, scope, e.Left, e.ResolvedType)
		if e.Op == "+" && left != nil && left.AddressSpace != "" {
			out := cloneSemanticValueFacts(left)
			if offset, ok := evt1StaticInt(env, scope, e.Right); ok && out.RelativeOffset.Known {
				out.RelativeOffset.Value += offset
				if out.Alignment.Known {
					out.Alignment.Value = evt1GCD(out.Alignment.Value, offset)
				}
			} else {
				out.RelativeOffset = SemanticKnownInt{}
				out.Alignment = SemanticKnownInt{}
			}
			out.Transport = append(out.Transport, SemanticFactTransportStep{Transform: FactTransformAddOffset, Detail: "affine address displacement"})
			return out
		}
	case *TemplateCallExpr:
		if e.Callee == "AddressOf" && len(e.Args) == 1 {
			origin := evt1SemanticStorageOrigin(e.Args[0])
			alignment := 1
			if ref, ok := e.Args[0].(*RefExpr); ok {
				if valueType, err := validateExpr(env, scope, ref.Value, nil, false); err == nil {
					_, alignment, _ = evt1TypeGeometry(env, valueType.valueType())
				}
			}
			return &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), AddressSpace: e.TypeArg.Name, RegionOrigin: origin, RelativeOffset: SemanticKnownInt{Known: true}, Alignment: SemanticKnownInt{Known: true, Value: alignment}, Provenance: evt1ExprProvenance(env, scope, e.Args[0]), HostAccessible: map[bool]SemanticFactCertainty{true: FactProven, false: FactUnknown}[e.TypeArg.Name == "SystemMemory"], Transport: []SemanticFactTransportStep{{Transform: FactTransformPreserve, From: origin, Through: "AddressOf"}}}
		}
		if e.Callee == "AddressFromBits" {
			return &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), AddressSpace: e.TypeArg.Name, Provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceUnknown, Scoped: true}, Transport: []SemanticFactTransportStep{{Transform: FactTransformOpaqueBoundary, Through: "AddressFromBits", Detail: "region origin and provenance remain Unknown"}}}
		}
		if e.Callee == "bind" && len(e.Args) == 2 {
			addressType, _ := validateExpr(env, scope, e.Args[0], nil, false)
			out := evt1SemanticFactsForExpr(env, scope, e.Args[0], addressType)
			if out == nil {
				out = &SemanticValueFacts{}
			}
			if extent, ok := evt1StaticInt(env, scope, e.Args[1]); ok {
				out.ByteExtent = SemanticKnownInt{Known: true, Value: extent}
			}
			out.Type, out.Contiguous, out.Bounded, out.NoAllocation, out.Initialized = t.String(), FactProven, FactProven, FactProven, FactDisproven
			return transportSemanticValueFacts(out, FactTransformPreserve, exprLabel(e.Args[0]), "bind<T>", "typed storage preserves raw region geometry")
		}
		if summary, ok := env.templateFactSummaries[e.Callee]; ok {
			args := evt1SemanticArgumentFacts(env, scope, e.Args)
			return instantiateSemanticFactSummary(summary, args, e.Callee)
		}
	case *CallExpr:
		if e.Callee == "Initialize" && len(e.Args) != 0 {
			out := evt1SemanticFactsForExpr(env, scope, e.Args[0], t)
			if out != nil {
				out.Initialized = FactProven
				if len(t.TypeArgs) == 0 {
					if size, alignment, err := evt1TypeGeometry(env, t.valueType()); err == nil {
						out.ByteExtent = SemanticKnownInt{Known: true, Value: size}
						if !out.Alignment.Known || out.Alignment.Value < alignment {
							out.Alignment = SemanticKnownInt{Known: true, Value: alignment}
						}
					}
				}
			}
			return transportSemanticValueFacts(out, FactTransformInitialize, exprLabel(e.Args[0]), "Initialize", "object lifetime begins inside the same storage region")
		}
		for _, fn := range env.functions[e.Callee] {
			if len(fn.Params) != len(e.Args) {
				continue
			}
			key := evt1FunctionProvenanceKey(fn)
			summary, ok := env.resultFactSummaries[key]
			if !ok {
				break
			}
			return instantiateSemanticFactSummary(summary, evt1SemanticArgumentFacts(env, scope, e.Args), e.Callee)
		}
	case *CallableExpr:
		out := &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), Fields: map[string]*SemanticValueFacts{}, NoAllocation: FactProven}
		for _, capture := range e.Captures {
			if binding, ok := scope.lookup(capture.Name); ok {
				out.Fields[capture.Name] = transportSemanticValueFacts(binding.valueFacts, FactTransformCapture, capture.Name, e.EnvironmentID+"."+capture.Name, string(capture.Kind))
			}
		}
		return out
	}
	if span := evt1SpanFactsForValue(env, scope, expr, t); span != nil {
		return semanticValueFactsFromSpan(t, span)
	}
	if tensor := evt1TensorFactsForValue(scope, expr, t); tensor != nil {
		return semanticValueFactsFromTensor(t, tensor)
	}
	return nil
}

func evt1SemanticArgumentFacts(env *semanticEnv, scope *evt1Scope, args []Expr) []*SemanticValueFacts {
	out := make([]*SemanticValueFacts, len(args))
	for i, arg := range args {
		t, _ := validateExpr(env, scope, arg, nil, false)
		out[i] = evt1SemanticFactsForExpr(env, scope, arg, t)
		if out[i] == nil {
			if value, ok := evt1KnownTransportInt(scope, arg); ok {
				out[i] = &SemanticValueFacts{Scalar: SemanticKnownInt{Known: true, Value: value}}
			}
		}
	}
	return out
}

func evt1KnownTransportInt(scope *evt1Scope, expr Expr) (int, bool) {
	switch e := expr.(type) {
	case *IntLiteral:
		return e.Value, true
	case *NameExpr:
		if binding, ok := scope.lookup(e.Name); ok && binding.valueFacts != nil && binding.valueFacts.Scalar.Known {
			return binding.valueFacts.Scalar.Value, true
		}
	case *ParenExpr:
		return evt1KnownTransportInt(scope, e.Value)
	case *UnaryExpr:
		value, ok := evt1KnownTransportInt(scope, e.Value)
		if ok && e.Op == "-" {
			return -value, true
		}
	case *BinaryExpr:
		left, leftOK := evt1KnownTransportInt(scope, e.Left)
		right, rightOK := evt1KnownTransportInt(scope, e.Right)
		if !leftOK || !rightOK {
			return 0, false
		}
		switch e.Op {
		case "+":
			return left + right, true
		case "-":
			return left - right, true
		case "*":
			return left * right, true
		case "/":
			if right != 0 {
				return left / right, true
			}
		case "%":
			if right != 0 {
				return left % right, true
			}
		}
	}
	return 0, false
}

func evt1DeriveConcreteAggregateGeometry(decl StructDecl, out *SemanticValueFacts) {
	var address *SemanticValueFacts
	var bytes []*SemanticValueFacts
	for _, field := range decl.Fields {
		facts := out.Fields[field.Name]
		if field.Type.Kind == TypeAddress {
			address = facts
		}
		if evt1ByteDisplacement(field.Type) {
			bytes = append(bytes, facts)
		}
	}
	if address == nil || len(bytes) < 2 {
		return
	}
	out.AddressSpace, out.RegionOrigin, out.RelativeOffset, out.Provenance, out.HostAccessible = address.AddressSpace, address.RegionOrigin, address.RelativeOffset, address.Provenance, address.HostAccessible
	out.ParentRegion = address.RegionOrigin
	if bytes[0] != nil {
		out.ByteExtent = bytes[0].Scalar
	}
	if bytes[len(bytes)-1] != nil {
		out.Alignment = bytes[len(bytes)-1].Scalar
	}
	if address.Alignment.Known && out.Alignment.Known {
		out.Alignment.Value = evt1GCD(address.Alignment.Value, out.Alignment.Value)
	}
	out.Contiguous, out.Bounded = FactProven, FactProven
}

func semanticValueFactsFromSpan(t Type, span *evt1SpanFacts) *SemanticValueFacts {
	if span == nil {
		return nil
	}
	out := &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), RegionOrigin: span.RegionID, RelativeOffset: SemanticKnownInt{Known: true, Value: span.BackingByteOffset}, Alignment: SemanticKnownInt{Known: span.Alignment > 0, Value: span.Alignment}, Contiguous: FactProven, Bounded: FactProven, Provenance: span.Provenance, NoAllocation: FactProven, NoCopy: FactProven, NoOwnershipTransfer: FactProven}
	if span.LengthStatic {
		out.ByteExtent = SemanticKnownInt{Known: true}
		if extent, ok := evt1FactStaticInt(span.ByteExtentExpression); ok {
			out.ByteExtent.Value = extent
		}
	}
	return out
}

func semanticValueFactsFromTensor(t Type, tensor *TensorViewFacts) *SemanticValueFacts {
	if tensor == nil {
		return nil
	}
	return &SemanticValueFacts{SubjectKind: SubjectValue, Type: t.String(), RegionOrigin: tensor.RegionID, Alignment: SemanticKnownInt{Known: tensor.Alignment > 0, Value: tensor.Alignment}, Contiguous: map[bool]SemanticFactCertainty{true: FactProven, false: FactDisproven}[tensor.Contiguous], Bounded: FactProven, Rank: SemanticKnownInt{Known: true, Value: tensor.Rank}, Shape: append([]StorageDimension{}, tensor.Shape...), NoAllocation: map[bool]SemanticFactCertainty{true: FactProven, false: FactUnknown}[tensor.NoAllocation], NoCopy: map[bool]SemanticFactCertainty{true: FactProven, false: FactUnknown}[tensor.NoCopy], NoOwnershipTransfer: map[bool]SemanticFactCertainty{true: FactProven, false: FactUnknown}[tensor.NoOwnershipTransfer]}
}

func joinSemanticValueFacts(a, b *SemanticValueFacts) *SemanticValueFacts {
	if a == nil {
		return cloneSemanticValueFacts(b)
	}
	if b == nil {
		return cloneSemanticValueFacts(a)
	}
	out := &SemanticValueFacts{SubjectKind: SubjectValue, Type: a.Type, Transport: []SemanticFactTransportStep{{Transform: FactTransformJoin, Detail: "branch join retains only common guarantees"}}}
	if a.AddressSpace == b.AddressSpace {
		out.AddressSpace = a.AddressSpace
	}
	if a.RegionOrigin == b.RegionOrigin {
		out.RegionOrigin = a.RegionOrigin
	}
	if a.RelativeOffset == b.RelativeOffset {
		out.RelativeOffset = a.RelativeOffset
	}
	if a.ByteExtent == b.ByteExtent {
		out.ByteExtent = a.ByteExtent
	}
	if a.Alignment.Known && b.Alignment.Known {
		out.Alignment = SemanticKnownInt{Known: true, Value: evt1GCD(a.Alignment.Value, b.Alignment.Value)}
	}
	out.Contiguous, out.Bounded = joinCertainty(a.Contiguous, b.Contiguous), joinCertainty(a.Bounded, b.Bounded)
	out.NoAllocation, out.NoCopy, out.NoOwnershipTransfer = joinCertainty(a.NoAllocation, b.NoAllocation), joinCertainty(a.NoCopy, b.NoCopy), joinCertainty(a.NoOwnershipTransfer, b.NoOwnershipTransfer)
	if evt1ProvenanceIsShorter(a.Provenance, b.Provenance) {
		out.Provenance = a.Provenance
	} else {
		out.Provenance = b.Provenance
	}
	return out
}

func evt1SemanticFactsAsSpan(facts *SemanticValueFacts) *evt1SpanFacts {
	if facts == nil || facts.RegionOrigin == "" {
		return nil
	}
	return &evt1SpanFacts{RegionID: facts.RegionOrigin, BackingByteOffset: facts.RelativeOffset.Value, BaseOffsetExpression: map[bool]string{true: fmt.Sprint(facts.RelativeOffset.Value), false: "runtime"}[facts.RelativeOffset.Known], StaticLength: facts.ByteExtent.Value, LengthStatic: facts.ByteExtent.Known, ByteExtentExpression: map[bool]string{true: fmt.Sprint(facts.ByteExtent.Value), false: "runtime"}[facts.ByteExtent.Known], Alignment: facts.Alignment.Value, Provenance: facts.Provenance}
}

func evt1RecordTransportedFacts(env *semanticEnv, function, name string, t Type, facts *SemanticValueFacts, span Span) {
	if facts == nil {
		return
	}
	if facts.SubjectKind == "" {
		facts.SubjectKind = SubjectValue
	}
	facts.Subject, facts.Type = name, t.String()
	subject := SemanticFactSubject{Kind: string(facts.SubjectKind), Name: name, Function: function, Type: t.String(), RegionID: facts.RegionOrigin}
	evidence := SemanticFactEvidence{RegionIDs: nil, Offset: facts.RelativeOffset.Value, Extent: facts.ByteExtent.Value, Alignment: facts.Alignment.Value, Rank: facts.Rank.Value, Shape: append([]StorageDimension{}, facts.Shape...), AddressSpace: facts.AddressSpace, ParentRegion: facts.ParentRegion, Provenance: string(facts.Provenance.Kind), Transport: append([]SemanticFactTransportStep{}, facts.Transport...)}
	if facts.RegionOrigin != "" {
		evidence.RegionIDs = []string{facts.RegionOrigin}
		for _, kind := range []SemanticFactKind{FactRegionOrigin, FactRegionIdentity} {
			evt1AppendTransportFact(env, kind, subject, nil, FactProven, evidence, span)
		}
	}
	if facts.AddressSpace != "" {
		evt1AppendTransportFact(env, FactAddressSpace, subject, nil, FactProven, evidence, span)
	}
	if facts.ParentRegion != "" {
		evt1AppendTransportFact(env, FactParentRegion, subject, nil, FactProven, evidence, span)
	}
	if facts.RelativeOffset.Known && facts.ByteExtent.Known {
		evt1AppendTransportFact(env, FactByteInterval, subject, nil, FactProven, evidence, span)
	}
	if facts.ByteExtent.Known {
		evt1AppendTransportFact(env, FactByteExtent, subject, nil, FactProven, evidence, span)
	}
	if facts.Alignment.Known && facts.Alignment.Value > 0 {
		evt1AppendTransportFact(env, FactAligned, subject, []int{facts.Alignment.Value}, FactProven, evidence, span)
	}
	ordered := []struct {
		kind    SemanticFactKind
		outcome SemanticFactCertainty
	}{
		{FactContiguous, facts.Contiguous},
		{FactBounded, facts.Bounded},
		{FactNoAllocation, facts.NoAllocation},
		{FactNoCopy, facts.NoCopy},
		{FactNoOwnershipTransfer, facts.NoOwnershipTransfer},
		{FactHostAccessible, facts.HostAccessible},
		{FactInitialized, facts.Initialized},
	}
	for _, entry := range ordered {
		if entry.outcome != "" && entry.outcome != FactUnknown {
			evt1AppendTransportFact(env, entry.kind, subject, nil, entry.outcome, evidence, span)
		}
	}
	if facts.Rank.Known {
		evt1AppendTransportFact(env, FactRank, subject, []int{facts.Rank.Value}, FactProven, evidence, span)
	}
}

func evt1AppendTransportFact(env *semanticEnv, kind SemanticFactKind, subject SemanticFactSubject, parameters []int, outcome SemanticFactCertainty, evidence SemanticFactEvidence, span Span) {
	id := evt1SemanticFactID(kind, []SemanticFactSubject{subject}, parameters, span)
	if env.transportedFactIDs[id] {
		return
	}
	env.transportedFactIDs[id] = true
	env.transportedFacts = append(env.transportedFacts, MIRSemanticFact{ID: id, Kind: kind, Subjects: []SemanticFactSubject{subject}, Parameters: parameters, Outcome: outcome, Origin: FactOriginTransportedValue, Evidence: evidence, SourceSpan: span})
}

func semanticModuleFactSummaries(module Module, env *semanticEnv) []SemanticFunctionFactSummary {
	var out []SemanticFunctionFactSummary
	for _, fn := range module.Functions {
		key := evt1FunctionProvenanceKey(fn)
		out = append(out, SemanticFunctionFactSummary{Operation: fn.Name, Signature: evt1FunctionParamSignature(fn), Result: env.resultFactSummaries[key], Origin: FactOriginCompilerAnalysis})
	}
	for _, template := range module.Templates {
		out = append(out, SemanticFunctionFactSummary{Operation: template.Name, Signature: "template", Result: env.templateFactSummaries[template.Name], Origin: FactOriginCompilerAnalysis})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Operation != out[j].Operation {
			return out[i].Operation < out[j].Operation
		}
		return out[i].Signature < out[j].Signature
	})
	return out
}

func validateSemanticFunctionFactSummaries(summaries []SemanticFunctionFactSummary) error {
	seen := map[string]bool{}
	last := ""
	for _, summary := range summaries {
		key := summary.Operation + "\x00" + summary.Signature
		if summary.Operation == "" || summary.Signature == "" || summary.Origin == "" || seen[key] {
			return fmt.Errorf("MODULE_FACT_SUMMARY_INVALID: incomplete or duplicate semantic value summary")
		}
		if last != "" && key < last {
			return fmt.Errorf("MODULE_FACT_SUMMARY_INVALID: semantic value summaries are not deterministically ordered")
		}
		seen[key], last = true, key
		nodes, depth := semanticSummaryComplexity(summary.Result)
		if nodes > 128 || depth > 16 {
			return fmt.Errorf("MODULE_FACT_SUMMARY_TOO_COMPLEX: %s has %d nodes at depth %d", summary.Operation, nodes, depth)
		}
		if err := validateSemanticValueFactSummary(summary.Result); err != nil {
			return fmt.Errorf("MODULE_FACT_SUMMARY_INVALID: %s: %w", summary.Operation, err)
		}
	}
	return nil
}

func validateSemanticValueFactSummary(summary SemanticValueFactSummary) error {
	allowedFacts := map[string]bool{"RegionOrigin": true, "RelativeOffset": true, "Alignment": true, "ByteExtent": true}
	var validateExpr func(*SemanticSummaryExpr) error
	validateExpr = func(expr *SemanticSummaryExpr) error {
		if expr == nil {
			return nil
		}
		switch expr.Kind {
		case "Parameter":
			if expr.ParameterIndex < 0 || expr.Left != nil || expr.Right != nil {
				return fmt.Errorf("invalid Parameter expression")
			}
		case "Field":
			if expr.Left == nil || len(expr.FieldPath) == 0 || expr.Right != nil {
				return fmt.Errorf("invalid Field expression")
			}
		case "Fact":
			if expr.Left == nil || !allowedFacts[expr.Fact] || expr.Right != nil {
				return fmt.Errorf("invalid Fact expression")
			}
		case "Constant", "StorageOrigin":
			if expr.Left != nil || expr.Right != nil {
				return fmt.Errorf("invalid %s expression", expr.Kind)
			}
		case "Add", "CommonAlignment":
			if expr.Left == nil || expr.Right == nil {
				return fmt.Errorf("invalid %s expression", expr.Kind)
			}
		default:
			return fmt.Errorf("unknown expression kind %q", expr.Kind)
		}
		if err := validateExpr(expr.Left); err != nil {
			return err
		}
		return validateExpr(expr.Right)
	}
	for _, expr := range []*SemanticSummaryExpr{summary.Source, summary.Scalar, summary.RegionOrigin, summary.ParentRegion, summary.RelativeOffset, summary.ByteExtent, summary.Alignment} {
		if err := validateExpr(expr); err != nil {
			return err
		}
	}
	seenFields := map[string]bool{}
	for _, field := range summary.Fields {
		if field.Name == "" || seenFields[field.Name] {
			return fmt.Errorf("empty or duplicate field summary")
		}
		seenFields[field.Name] = true
		if err := validateSemanticValueFactSummary(field.Facts); err != nil {
			return err
		}
	}
	return nil
}

func semanticSummaryComplexity(summary SemanticValueFactSummary) (int, int) {
	nodes, depth := 0, 0
	var visit func(*SemanticSummaryExpr, int)
	visit = func(expr *SemanticSummaryExpr, current int) {
		if expr == nil {
			return
		}
		nodes++
		if current > depth {
			depth = current
		}
		visit(expr.Left, current+1)
		visit(expr.Right, current+1)
	}
	for _, expr := range []*SemanticSummaryExpr{summary.Source, summary.Scalar, summary.RegionOrigin, summary.ParentRegion, summary.RelativeOffset, summary.ByteExtent, summary.Alignment} {
		visit(expr, 1)
	}
	for _, field := range summary.Fields {
		childNodes, childDepth := semanticSummaryComplexity(field.Facts)
		nodes += childNodes
		if childDepth+1 > depth {
			depth = childDepth + 1
		}
	}
	return nodes, depth
}
