package concept

import (
	"fmt"
	"sort"
	"strings"
)

// TensorViewFacts are the storage-neutral facts retained by a tensor view.
// Shape expressions are deliberately preserved until tensor lowering.
type TensorViewFacts struct {
	ElementType         Type               `json:"element_type"`
	Rank                int                `json:"rank"`
	Shape               []StorageDimension `json:"shape"`
	RegionID            string             `json:"region_id"`
	BaseOffset          string             `json:"base_offset"`
	Alignment           int                `json:"alignment"`
	Mutability          string             `json:"mutability"`
	Contiguous          bool               `json:"contiguous"`
	Provenance          string             `json:"provenance"`
	Source              string             `json:"source"`
	NoCopy              bool               `json:"no_copy"`
	NoAllocation        bool               `json:"no_allocation"`
	NoOwnershipTransfer bool               `json:"no_ownership_transfer"`
}

type MIRTensorOperand struct {
	Name                string             `json:"name"`
	ElementType         Type               `json:"element_type"`
	Indices             []string           `json:"indices,omitempty"`
	Rank                int                `json:"rank"`
	Shape               []StorageDimension `json:"shape"`
	RegionID            string             `json:"region_id"`
	BaseOffset          string             `json:"base_offset"`
	Alignment           int                `json:"alignment"`
	Mutability          string             `json:"mutability"`
	Contiguous          bool               `json:"contiguous"`
	Provenance          string             `json:"provenance"`
	NoCopy              bool               `json:"no_copy"`
	NoAllocation        bool               `json:"no_allocation"`
	NoOwnershipTransfer bool               `json:"no_ownership_transfer"`
}

// TensorSemantic is attached to the assignment after semantic validation and
// is the input to the dedicated tensor-to-loop lowering stage.
type TensorSemantic struct {
	Kind          string             `json:"kind"`
	Output        MIRTensorOperand   `json:"output"`
	Operands      []MIRTensorOperand `json:"operands,omitempty"`
	FreeIndices   []string           `json:"free_indices,omitempty"`
	ReduceIndices []string           `json:"reduction_indices,omitempty"`
	ElementOp     string             `json:"element_op,omitempty"`
	ShapeGuards   []string           `json:"shape_guards,omitempty"`
	AliasPolicy   string             `json:"alias_policy"`
	Lowering      string             `json:"lowering"`
	SourceSpan    Span               `json:"source_span"`
}

type MIRTensorOperation = TensorSemantic

func evt1IsTensorType(t Type) bool  { return t.Kind == TypeTensor || t.Name == "tensor" }
func evt1TensorElement(t Type) Type { return t.TypeArgs[0] }

func evt1ValidateTensorType(env *semanticEnv, t Type, span Span, conceptParam string) error {
	if len(t.TypeArgs) != 1 || t.TensorRank <= 0 {
		return evt1Diagnostic("CV4610", "tensor<T, Rank> requires one element type and a positive compile-time rank", span)
	}
	if err := validateKnownType(env, t.TypeArgs[0], span, conceptParam, false); err != nil {
		return err
	}
	e := evt1CanonicalType(env, t.TypeArgs[0].valueType())
	if e.Name != "int" && e.Name != "uint" && e.Name != "uint64" && e.Name != "byte" && e.Name != "float" {
		return evt1Diagnostic("CV4612", fmt.Sprintf("tensor element type %s is not a supported scalar arithmetic type", e.String()), span)
	}
	return nil
}

func evt1TensorType(element Type, rank int, span Span) Type {
	return Type{Name: "tensor", Kind: TypeTensor, TypeArgs: []Type{element.valueType()}, TensorRank: rank, Span: span}
}

func evt1TensorCName(t Type) string {
	mutability := "mut"
	if t.Const {
		mutability = "const"
	}
	return fmt.Sprintf("concept_tensor_%s_%s_%d", mutability, evt1TypeIdentity(evt1TensorElement(t)), t.TensorRank)
}

func evt1TensorFactsForValue(scope *evt1Scope, expr Expr, t Type) *TensorViewFacts {
	if !evt1IsTensorType(t) {
		return nil
	}
	if call, ok := expr.(*CallExpr); ok && call.TensorFacts != nil {
		copy := *call.TensorFacts
		return &copy
	}
	if scope != nil {
		if name, ok := expr.(*NameExpr); ok {
			if binding, found := scope.lookup(name.Name); found && binding.tensorFacts != nil {
				copy := *binding.tensorFacts
				return &copy
			}
		}
	}
	return nil
}

func evt1TensorFactsForExpr(scope *evt1Scope, expr Expr) (TensorViewFacts, bool) {
	if call, ok := expr.(*CallExpr); ok && call.TensorFacts != nil {
		return *call.TensorFacts, true
	}
	if name, ok := expr.(*NameExpr); ok && scope != nil {
		if binding, found := scope.lookup(name.Name); found && binding.tensorFacts != nil {
			return *binding.tensorFacts, true
		}
	}
	return TensorViewFacts{}, false
}

func validateTensorConstruction(env *semanticEnv, scope *evt1Scope, call *CallExpr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if !evt1IsTensorType(expected) {
		return Type{}, evt1Diagnostic("CV4611", "Tensor(source) requires a tensor<T, Rank> destination", call.Span)
	}
	if len(call.Args) != 1 {
		return Type{}, evt1Diagnostic("CV4613", "Tensor expects exactly one existing shaped source", call.Span)
	}
	sourceType, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	place, err := validateAssignable(env, scope, call.Args[0], templateInfo)
	if err != nil {
		return Type{}, evt1Diagnostic("CV4613", "Tensor requires an existing contiguous source place", call.Args[0].exprSpan())
	}
	var element Type
	var shape []StorageDimension
	regionID, alignment, baseOffset := "", 1, "0"
	if evt1IsSpanType(sourceType) {
		if expected.TensorRank != 1 {
			return Type{}, evt1Diagnostic("CV4614", "Span can form only tensor<T, 1>; rank greater than one requires explicit shaped storage", call.Span)
		}
		element = evt1SpanElement(sourceType)
		facts, factsErr := evt1SpanSourceFacts(env, scope, call.Args[0], sourceType, place)
		if factsErr != nil {
			return Type{}, factsErr
		}
		shape = []StorageDimension{{Runtime: !facts.LengthStatic, Extent: facts.StaticLength, Expression: facts.LengthExpression}}
		regionID, alignment, baseOffset = facts.RegionID, facts.Alignment, facts.BaseOffsetExpression
	} else {
		if sourceType.ArrayElem == nil || !sourceType.Contiguous {
			return Type{}, evt1Diagnostic("CV4613", "Tensor requires shaped contiguous ndarray, layout region, stream channel, or rank-1 Span storage", call.Args[0].exprSpan())
		}
		element, shape = *sourceType.ArrayElem, append([]StorageDimension{}, sourceType.Shape...)
		if evt1StorageRank(sourceType) != expected.TensorRank {
			return Type{}, evt1Diagnostic("CV4614", fmt.Sprintf("tensor rank %d does not match source rank %d", expected.TensorRank, evt1StorageRank(sourceType)), call.Span)
		}
		if sf, factsErr := evt1SpanSourceFacts(env, scope, call.Args[0], sourceType, place); factsErr == nil {
			regionID, alignment, baseOffset = sf.RegionID, sf.Alignment, sf.BaseOffsetExpression
		}
	}
	if !evt1CanonicalType(env, element.valueType()).Equal(evt1CanonicalType(env, evt1TensorElement(expected).valueType())) {
		return Type{}, evt1Diagnostic("CV4615", fmt.Sprintf("tensor element type %s does not match source element type %s", evt1TensorElement(expected).String(), element.String()), call.Span)
	}
	if regionID == "" {
		regionID = "storage:" + place.path.Root
	}
	prov := evt1ExprProvenance(env, scope, call.Args[0])
	provText := string(prov.Kind)
	if prov.Scoped {
		provText += ":scoped"
	}
	mutability := "readonly"
	if place.mutable && !sourceType.Const {
		mutability = "mutable"
	}
	call.Intrinsic = "tensor_view"
	call.TensorFacts = &TensorViewFacts{ElementType: element.valueType(), Rank: expected.TensorRank, Shape: shape, RegionID: regionID, BaseOffset: baseOffset, Alignment: alignment, Mutability: mutability, Contiguous: true, Provenance: provText, Source: exprLabel(call.Args[0]), NoCopy: true, NoAllocation: true, NoOwnershipTransfer: true}
	return expected, nil
}

func validateOrdinaryTensorIndex(env *semanticEnv, scope *evt1Scope, index *IndexExpr, facts TensorViewFacts, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	indices := evt1StorageIndices(index)
	if len(indices) != facts.Rank {
		return Type{}, evt1Diagnostic("CV4618", fmt.Sprintf("tensor index requires %d indices, got %d", facts.Rank, len(indices)), index.Span)
	}
	for axis, expr := range indices {
		t, err := validateExpr(env, scope, expr, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if t.Name != "int" {
			return Type{}, evt1Diagnostic("CV4618", "ordinary tensor indices must be int", expr.exprSpan())
		}
		if v, err := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), expr); err == nil && axis < len(facts.Shape) && !facts.Shape[axis].Runtime && (v.IntValue < 0 || v.IntValue >= facts.Shape[axis].Extent) {
			return Type{}, evt1Diagnostic("CV4618", fmt.Sprintf("tensor index %d is out of bounds for axis %d extent %d", v.IntValue, axis, facts.Shape[axis].Extent), expr.exprSpan())
		}
	}
	index.TensorIndex = true
	return evt1CanonicalType(env, facts.ElementType), nil
}

func tensorOperand(name string, indices []string, facts TensorViewFacts) MIRTensorOperand {
	return MIRTensorOperand{Name: name, ElementType: facts.ElementType, Indices: indices, Rank: facts.Rank, Shape: append([]StorageDimension{}, facts.Shape...), RegionID: facts.RegionID, BaseOffset: facts.BaseOffset, Alignment: facts.Alignment, Mutability: facts.Mutability, Contiguous: facts.Contiguous, Provenance: facts.Provenance, NoCopy: facts.NoCopy, NoAllocation: facts.NoAllocation, NoOwnershipTransfer: facts.NoOwnershipTransfer}
}

func tensorShapeEqual(a, b []StorageDimension) (bool, bool) {
	if len(a) != len(b) {
		return false, false
	}
	runtime := false
	for i := range a {
		if a[i].Runtime || b[i].Runtime {
			runtime = true
			continue
		}
		if a[i].Extent != b[i].Extent {
			return false, runtime
		}
	}
	return true, runtime
}

func tensorNameFacts(scope *evt1Scope, expr Expr) (string, TensorViewFacts, bool) {
	n, ok := expr.(*NameExpr)
	if !ok {
		return "", TensorViewFacts{}, false
	}
	f, ok := evt1TensorFactsForExpr(scope, n)
	return n.Name, f, ok
}

func validateTensorAssignment(env *semanticEnv, scope *evt1Scope, stmt *AssignStmt, templateInfo *evt1TemplateInfo, inComptimeFn bool) (bool, error) {
	if targetIndex, ok := stmt.Target.(*IndexExpr); ok {
		if outName, outFacts, yes := tensorNameFacts(scope, targetIndex.Base); yes {
			symbolic := false
			for _, index := range evt1StorageIndices(targetIndex) {
				if name, isName := index.(*NameExpr); isName {
					if _, bound := scope.lookup(name.Name); !bound {
						symbolic = true
					}
				}
			}
			if !symbolic {
				return false, nil
			}
			return true, validateEinsteinAssignment(env, scope, stmt, targetIndex, outName, outFacts, templateInfo, inComptimeFn)
		}
	}
	outName, outFacts, ok := tensorNameFacts(scope, stmt.Target)
	if !ok {
		return false, nil
	}
	binding, _ := scope.lookup(outName)
	if !binding.mutable || outFacts.Mutability != "mutable" {
		return true, evt1Diagnostic("CV4616", "tensor assignment destination is readonly", stmt.Target.exprSpan())
	}
	bin, ok := stmt.Value.(*BinaryExpr)
	if !ok {
		return true, evt1Diagnostic("CV4617", "tensor computation requires exact-shape elementwise arithmetic or @ contraction into existing storage", stmt.Value.exprSpan())
	}
	leftName, leftFacts, leftTensor := tensorNameFacts(scope, bin.Left)
	rightName, rightFacts, rightTensor := tensorNameFacts(scope, bin.Right)
	sem := &TensorSemantic{Output: tensorOperand(outName, nil, outFacts), ElementOp: bin.Op, AliasPolicy: "corresponding-element in-place only; contraction overlap rejected", Lowering: "dedicated tensor lowering to explicit strict-C11 loops", SourceSpan: stmt.Span}
	if bin.Op == "@" {
		if !leftTensor || !rightTensor {
			return true, evt1Diagnostic("CV4624", "@ requires two tensor operands", bin.Span)
		}
		if leftFacts.Rank == 1 && rightFacts.Rank == 1 {
			return true, evt1Diagnostic("CV4624", "rank-1 @ rank-1 scalar result is deferred because R4h tensor rank is positive", bin.Span)
		}
		if leftFacts.RegionID == outFacts.RegionID || rightFacts.RegionID == outFacts.RegionID {
			return true, evt1Diagnostic("CV4625", "tensor contraction destination overlaps an input region", stmt.Span)
		}
		li, ri := leftFacts.Shape[leftFacts.Rank-1], rightFacts.Shape[0]
		if !li.Runtime && !ri.Runtime && li.Extent != ri.Extent {
			return true, evt1Diagnostic("CV4624", fmt.Sprintf("tensor contraction shape mismatch: left last extent %d != right first extent %d", li.Extent, ri.Extent), bin.Span)
		}
		result := append([]StorageDimension{}, leftFacts.Shape[:leftFacts.Rank-1]...)
		result = append(result, rightFacts.Shape[1:]...)
		if equal, _ := tensorShapeEqual(outFacts.Shape, result); !equal {
			return true, evt1Diagnostic("CV4617", "tensor contraction result shape does not exactly match destination", stmt.Span)
		}
		sem.Kind = "tensor_contract"
		sem.Operands = []MIRTensorOperand{tensorOperand(leftName, nil, leftFacts), tensorOperand(rightName, nil, rightFacts)}
		if li.Runtime || ri.Runtime {
			sem.ShapeGuards = append(sem.ShapeGuards, "left.last == right.first")
		}
		for i := range result {
			if outFacts.Shape[i].Runtime || result[i].Runtime {
				sem.ShapeGuards = append(sem.ShapeGuards, fmt.Sprintf("output.%d == result.%d", i, i))
			}
		}
		stmt.Tensor = sem
		return true, nil
	}
	if bin.Op != "+" && bin.Op != "-" && bin.Op != "*" {
		return true, evt1Diagnostic("CV4617", "unsupported tensor element operation "+bin.Op, bin.Span)
	}
	if leftTensor && rightTensor {
		if equal, _ := tensorShapeEqual(leftFacts.Shape, rightFacts.Shape); !equal {
			return true, evt1Diagnostic("CV4617", "tensor elementwise operands require exact equal shape; broadcasting is not supported", bin.Span)
		}
		if equal, _ := tensorShapeEqual(outFacts.Shape, leftFacts.Shape); !equal {
			return true, evt1Diagnostic("CV4617", "tensor elementwise result shape must exactly match destination; broadcasting is not supported", stmt.Span)
		}
		sem.Kind = "tensor_elementwise"
		sem.Operands = []MIRTensorOperand{tensorOperand(leftName, nil, leftFacts), tensorOperand(rightName, nil, rightFacts)}
		for i := range outFacts.Shape {
			if outFacts.Shape[i].Runtime || leftFacts.Shape[i].Runtime || rightFacts.Shape[i].Runtime {
				sem.ShapeGuards = append(sem.ShapeGuards, fmt.Sprintf("output.%d == left.%d == right.%d", i, i, i))
			}
		}
		stmt.Tensor = sem
		return true, nil
	}
	if bin.Op == "*" && (leftTensor != rightTensor) {
		tensorName, tensorFacts, scalarExpr := leftName, leftFacts, bin.Right
		if !leftTensor {
			tensorName, tensorFacts, scalarExpr = rightName, rightFacts, bin.Left
		}
		scalarType, err := validateExpr(env, scope, scalarExpr, templateInfo, inComptimeFn)
		if err != nil {
			return true, err
		}
		if !evt1CanonicalType(env, scalarType).Equal(evt1CanonicalType(env, tensorFacts.ElementType)) {
			return true, evt1Diagnostic("CV4615", "tensor scalar operation requires the exact element scalar type", scalarExpr.exprSpan())
		}
		if equal, _ := tensorShapeEqual(outFacts.Shape, tensorFacts.Shape); !equal {
			return true, evt1Diagnostic("CV4617", "tensor scalar result shape must exactly match destination", stmt.Span)
		}
		sem.Kind = "tensor_scalar_op"
		sem.Operands = []MIRTensorOperand{tensorOperand(tensorName, nil, tensorFacts)}
		stmt.Tensor = sem
		return true, nil
	}
	return true, evt1Diagnostic("CV4617", "tensor elementwise arithmetic requires two tensors of exact shape (or tensor * scalar)", bin.Span)
}

func validateEinsteinAssignment(env *semanticEnv, scope *evt1Scope, stmt *AssignStmt, lhs *IndexExpr, outName string, outFacts TensorViewFacts, templateInfo *evt1TemplateInfo, inComptimeFn bool) error {
	binding, _ := scope.lookup(outName)
	if !binding.mutable || outFacts.Mutability != "mutable" {
		return evt1Diagnostic("CV4616", "tensor indexed assignment destination is readonly", lhs.Span)
	}
	if len(evt1StorageIndices(lhs)) != outFacts.Rank {
		return evt1Diagnostic("CV4618", fmt.Sprintf("tensor output index arity %d does not match rank %d", len(evt1StorageIndices(lhs)), outFacts.Rank), lhs.Span)
	}
	free := make([]string, 0, outFacts.Rank)
	extents := map[string]StorageDimension{}
	counts := map[string]int{}
	for axis, e := range evt1StorageIndices(lhs) {
		n, ok := e.(*NameExpr)
		if !ok {
			return evt1Diagnostic("CV4619", "symbolic tensor assignment cannot mix ordinary and symbolic indices", e.exprSpan())
		}
		if _, seen := extents[n.Name]; seen {
			return evt1Diagnostic("CV4621", "output symbolic indices must be unique", n.Span)
		}
		free = append(free, n.Name)
		extents[n.Name] = outFacts.Shape[axis]
	}
	var operands []MIRTensorOperand
	var visit func(Expr) error
	visit = func(expr Expr) error {
		switch e := expr.(type) {
		case *IndexExpr:
			name, facts, ok := tensorNameFacts(scope, e.Base)
			if ok {
				idx := evt1StorageIndices(e)
				if len(idx) != facts.Rank {
					return evt1Diagnostic("CV4618", fmt.Sprintf("tensor operand %s requires %d symbolic indices", name, facts.Rank), e.Span)
				}
				maps := make([]string, len(idx))
				for axis, x := range idx {
					n, yes := x.(*NameExpr)
					if !yes {
						return evt1Diagnostic("CV4619", "symbolic tensor expression cannot mix ordinary and symbolic indices", x.exprSpan())
					}
					maps[axis] = n.Name
					counts[n.Name]++
					if prior, found := extents[n.Name]; found {
						if !prior.Runtime && !facts.Shape[axis].Runtime && prior.Extent != facts.Shape[axis].Extent {
							return evt1Diagnostic("CV4620", fmt.Sprintf("symbolic index %s has incompatible extents %d and %d", n.Name, prior.Extent, facts.Shape[axis].Extent), n.Span)
						}
					} else {
						extents[n.Name] = facts.Shape[axis]
					}
				}
				e.TensorIndex = true
				e.SymbolicIndices = maps
				operands = append(operands, tensorOperand(name, maps, facts))
				return nil
			}
			return visit(e.Base)
		case *BinaryExpr:
			if err := visit(e.Left); err != nil {
				return err
			}
			return visit(e.Right)
		case *UnaryExpr:
			return visit(e.Value)
		case *ParenExpr:
			return visit(e.Value)
		case *IfExpr:
			if err := visit(e.Condition); err != nil {
				return err
			}
			if err := visit(e.Then); err != nil {
				return err
			}
			return visit(e.Else)
		case *CallExpr:
			for _, a := range e.Args {
				if err := visit(a); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(stmt.Value); err != nil {
		return err
	}
	idxScope := newEVT1Scope(scope)
	intType, _ := evt1BuiltinType("int", stmt.Span)
	for name := range extents {
		idxScope.declare(name, evt1ValueBinding{t: intType, mutable: false, state: evt1StorageInitialized})
	}
	valueType, err := validateExpr(env, idxScope, stmt.Value, templateInfo, inComptimeFn)
	if err != nil {
		return err
	}
	if !evt1CanonicalType(env, valueType).Equal(evt1CanonicalType(env, outFacts.ElementType)) {
		return evt1Diagnostic("CV4615", fmt.Sprintf("tensor indexed expression produces %s but destination element is %s", valueType.String(), outFacts.ElementType.String()), stmt.Value.exprSpan())
	}
	for _, name := range free {
		if counts[name] == 0 && !tensorExprUsesName(stmt.Value, name) {
			return evt1Diagnostic("CV4621", fmt.Sprintf("output index %s does not appear in the right-hand expression", name), lhs.Span)
		}
	}
	var reduce []string
	for name, count := range counts {
		isFree := false
		for _, f := range free {
			if f == name {
				isFree = true
				break
			}
		}
		if !isFree {
			if count != 2 {
				return evt1Diagnostic("CV4622", fmt.Sprintf("reduction index %s appears %d times; R4h requires exactly twice", name, count), stmt.Value.exprSpan())
			}
			reduce = append(reduce, name)
		}
	}
	sort.Strings(reduce)
	for _, op := range operands {
		if len(reduce) > 0 && op.RegionID == outFacts.RegionID {
			return evt1Diagnostic("CV4625", "tensor contraction destination overlaps an input region", stmt.Span)
		}
	}
	kind := "tensor_index_assign"
	if len(reduce) > 0 {
		kind = "tensor_contract"
	}
	semantic := &TensorSemantic{Kind: kind, Output: tensorOperand(outName, free, outFacts), Operands: operands, FreeIndices: free, ReduceIndices: reduce, ElementOp: "indexed_expression", AliasPolicy: "indexed contraction overlap rejected", Lowering: "dedicated tensor lowering to explicit strict-C11 loop nest", SourceSpan: stmt.Span}
	for _, name := range append(append([]string{}, free...), reduce...) {
		refs := 0
		runtime := false
		for axis, outputName := range free {
			if outputName == name {
				refs++
				runtime = runtime || outFacts.Shape[axis].Runtime
			}
		}
		for _, operand := range operands {
			for axis, operandName := range operand.Indices {
				if operandName == name {
					refs++
					runtime = runtime || operand.Shape[axis].Runtime
				}
			}
		}
		if runtime && refs > 1 {
			semantic.ShapeGuards = append(semantic.ShapeGuards, "extent("+name+") exact")
		}
	}
	stmt.Tensor = semantic
	return nil
}

func validateTensorMIRSemantic(tensor MIRTensorOperation) error {
	fail := func(message string) error { return evt1Diagnostic("CV4626", message, tensor.SourceSpan) }
	validOperand := func(operand MIRTensorOperand) bool {
		if operand.Name == "" || operand.Rank < 1 || len(operand.Shape) != operand.Rank || operand.RegionID == "" || operand.BaseOffset == "" || operand.Alignment < 1 || operand.Mutability == "" || operand.Provenance == "" || !operand.Contiguous || !operand.NoCopy || !operand.NoAllocation || !operand.NoOwnershipTransfer {
			return false
		}
		for _, dimension := range operand.Shape {
			if (!dimension.Runtime && dimension.Extent < 1) || (dimension.Runtime && dimension.Expression == "") {
				return false
			}
		}
		return true
	}
	if tensor.Kind == "" || tensor.Lowering == "" || tensor.AliasPolicy == "" || !validOperand(tensor.Output) || tensor.Output.Mutability != "mutable" {
		return fail("Tensor MIR omits rank, shape, region, mutability, storage-neutral, or lowering facts")
	}
	for _, operand := range tensor.Operands {
		if !validOperand(operand) {
			return fail("Tensor MIR operand omits rank, shape, region, provenance, or storage-neutral facts")
		}
	}
	switch tensor.Kind {
	case "tensor_elementwise":
		if len(tensor.Operands) != 2 || tensor.ElementOp == "" {
			return fail("elementwise Tensor MIR requires two operands and an element operation")
		}
	case "tensor_scalar_op":
		if len(tensor.Operands) != 1 || tensor.ElementOp != "*" {
			return fail("scalar Tensor MIR requires one operand and multiplication")
		}
	case "tensor_index_assign", "tensor_contract":
		indexed := len(tensor.FreeIndices) > 0 || len(tensor.ReduceIndices) > 0
		if !indexed {
			if tensor.Kind != "tensor_contract" || len(tensor.Operands) != 2 {
				return fail("default contraction Tensor MIR requires exactly two operands")
			}
			if tensor.Output.RegionID == tensor.Operands[0].RegionID || tensor.Output.RegionID == tensor.Operands[1].RegionID {
				return fail("contraction Tensor MIR violates destination alias policy")
			}
			return nil
		}
		if len(tensor.FreeIndices) != tensor.Output.Rank || len(tensor.Output.Indices) != tensor.Output.Rank {
			return fail("indexed Tensor MIR free-index map does not match output rank")
		}
		free, reduce, counts := map[string]bool{}, map[string]bool{}, map[string]int{}
		for _, name := range tensor.FreeIndices {
			if name == "" || free[name] {
				return fail("indexed Tensor MIR has duplicate or empty free indices")
			}
			free[name] = true
		}
		for _, name := range tensor.ReduceIndices {
			if name == "" || free[name] || reduce[name] {
				return fail("indexed Tensor MIR has overlapping or duplicate reduction indices")
			}
			reduce[name] = true
		}
		for _, operand := range tensor.Operands {
			if len(operand.Indices) != operand.Rank {
				return fail("indexed Tensor MIR operand map does not match operand rank")
			}
			if tensor.Kind == "tensor_contract" && operand.RegionID == tensor.Output.RegionID {
				return fail("contraction Tensor MIR violates destination alias policy")
			}
			for _, name := range operand.Indices {
				if !free[name] && !reduce[name] {
					return fail("indexed Tensor MIR operand references an undeclared symbolic index")
				}
				counts[name]++
			}
		}
		for name := range reduce {
			if counts[name] != 2 {
				return fail("indexed Tensor MIR reduction index must occur exactly twice")
			}
		}
		if tensor.Kind == "tensor_index_assign" && len(reduce) != 0 {
			return fail("indexed assignment Tensor MIR cannot contain reduction indices")
		}
	default:
		return fail("Tensor MIR has an unknown operation kind")
	}
	return nil
}

func tensorExprUsesName(expr Expr, name string) bool {
	switch e := expr.(type) {
	case *NameExpr:
		return e.Name == name
	case *BinaryExpr:
		return tensorExprUsesName(e.Left, name) || tensorExprUsesName(e.Right, name)
	case *UnaryExpr:
		return tensorExprUsesName(e.Value, name)
	case *ParenExpr:
		return tensorExprUsesName(e.Value, name)
	case *IfExpr:
		return tensorExprUsesName(e.Condition, name) || tensorExprUsesName(e.Then, name) || tensorExprUsesName(e.Else, name)
	case *IndexExpr:
		for _, i := range evt1StorageIndices(e) {
			if tensorExprUsesName(i, name) {
				return true
			}
		}
	case *CallExpr:
		for _, a := range e.Args {
			if tensorExprUsesName(a, name) {
				return true
			}
		}
	}
	return false
}

func evt1CollectTensorTypes(module Module) []Type {
	m := map[string]Type{}
	var add func(Type)
	add = func(t Type) {
		if evt1IsTensorType(t) {
			m[evt1TensorCName(t)] = t
		}
		for _, a := range t.TypeArgs {
			add(a)
		}
		if t.ArrayElem != nil {
			add(*t.ArrayElem)
		}
	}
	var block func(Block)
	block = func(b Block) {
		for _, s := range b.Statements {
			switch x := s.(type) {
			case *VarDecl:
				add(x.Type)
			case *Block:
				block(*x)
			case *IfStmt:
				block(x.Then)
				if x.Else != nil {
					block(*x.Else)
				}
			case *WhileStmt:
				block(x.Body)
			}
		}
	}
	for _, f := range module.Functions {
		add(f.ReturnType)
		for _, p := range f.Params {
			add(p.Type)
		}
		if f.Body != nil {
			block(*f.Body)
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Type, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func evt1TensorDeclarations(types []Type) string {
	var b strings.Builder
	for _, t := range types {
		pointer := evt1CType(evt1TensorElement(t)) + "*"
		if t.Const {
			pointer = "const " + evt1CType(evt1TensorElement(t)) + "*"
		}
		b.WriteString(fmt.Sprintf("typedef struct { %s data; size_t shape[%d]; } %s;\n", pointer, t.TensorRank, evt1TensorCName(t)))
	}
	if len(types) > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerTensorView(call *CallExpr, indent int) (string, string, Type) {
	pre, value, sourceType := f.lowerExpr(call.Args[0], indent)
	facts := call.TensorFacts
	t := evt1TensorType(facts.ElementType, facts.Rank, call.Span)
	t.Const = facts.Mutability == "readonly"
	data := fmt.Sprintf("(%s).data", value)
	if evt1IsSpanType(sourceType) {
		data = fmt.Sprintf("(%s).data", value)
	}
	dims := make([]string, facts.Rank)
	for i, d := range facts.Shape {
		if d.Runtime {
			if evt1IsSpanType(sourceType) {
				dims[i] = fmt.Sprintf("(%s).length", value)
			} else {
				dims[i] = fmt.Sprintf("(%s).shape[%d]", value, i)
			}
		} else {
			dims[i] = fmt.Sprintf("%du", d.Extent)
		}
	}
	pointer := evt1CType(facts.ElementType) + "*"
	if t.Const {
		pointer = "const " + evt1CType(facts.ElementType) + "*"
	}
	return pre, fmt.Sprintf("(%s){ .data = (%s)%s, .shape = { %s } }", evt1TensorCName(t), pointer, data, strings.Join(dims, ", ")), t
}

func (f *evt1FunctionLowerer) lowerTensorIndex(index *IndexExpr, indent int) (string, string, Type) {
	basePrelude, baseValue, baseType := f.lowerExpr(index.Base, indent)
	baseName := f.nextTemp("tensor")
	var b strings.Builder
	b.WriteString(basePrelude)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(baseType), baseName, baseValue))
	indices := evt1StorageIndices(index)
	indexNames := make([]string, len(indices))
	for axis, expr := range indices {
		pre, value, _ := f.lowerExpr(expr, indent)
		name := f.nextTemp("tensor_index")
		b.WriteString(pre)
		b.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, value))
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", name, name, baseName, axis, "Concept tensor index out of bounds", index.Span.Line, index.Span.Column))
		indexNames[axis] = name
	}
	return b.String(), fmt.Sprintf("%s.data[%s]", baseName, tensorOffset(baseName, indexNames, len(indices))), evt1TensorElement(baseType)
}

func (f *evt1FunctionLowerer) tensorBinding(name string) evt1Binding {
	b, _ := scopeLookup(name, f.scope)
	return b
}

func tensorOffset(base string, indices []string, rank int) string {
	if rank == 0 {
		return "0u"
	}
	v := "(size_t)(" + indices[0] + ")"
	for i := 1; i < rank; i++ {
		v = fmt.Sprintf("((%s) * %s.shape[%d] + (size_t)(%s))", v, base, i, indices[i])
	}
	return v
}

func (f *evt1FunctionLowerer) lowerTensorAssignment(stmt *AssignStmt, indent int) string {
	s := stmt.Tensor
	out := f.tensorBinding(s.Output.Name)
	var b strings.Builder
	for _, guard := range s.ShapeGuards {
		_ = guard
	}
	if s.Kind == "tensor_elementwise" || s.Kind == "tensor_scalar_op" {
		total := f.nextTemp("tensor_count")
		b.WriteString(ind(indent) + "size_t " + total + " = 1u;\n")
		for i := 0; i < s.Output.Rank; i++ {
			b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != 0u && %s > SIZE_MAX / %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", out.cName, i, total, out.cName, i, "Concept tensor shape overflow", stmt.Span.Line, stmt.Span.Column))
			b.WriteString(ind(indent) + fmt.Sprintf("%s *= %s.shape[%d];\n", total, out.cName, i))
		}
		for _, op := range s.Operands {
			ob := f.tensorBinding(op.Name)
			for i := 0; i < s.Output.Rank; i++ {
				b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", out.cName, i, ob.cName, i, "Concept tensor exact-shape mismatch", stmt.Span.Line, stmt.Span.Column))
			}
		}
		i := f.nextTemp("tensor_i")
		b.WriteString(ind(indent) + fmt.Sprintf("for (size_t %s = 0u; %s < %s; ++%s) {\n", i, i, total, i))
		if s.Kind == "tensor_elementwise" {
			l := f.tensorBinding(s.Operands[0].Name)
			r := f.tensorBinding(s.Operands[1].Name)
			b.WriteString(ind(indent+1) + fmt.Sprintf("%s.data[%s] = %s.data[%s] %s %s.data[%s];\n", out.cName, i, l.cName, i, s.ElementOp, r.cName, i))
		} else {
			op := f.tensorBinding(s.Operands[0].Name)
			bin := stmt.Value.(*BinaryExpr)
			scalar := bin.Right
			if right, ok := bin.Right.(*NameExpr); ok && right.Name == s.Operands[0].Name {
				scalar = bin.Left
			}
			sp, sv, _ := f.lowerExpr(scalar, indent)
			b.WriteString(sp)
			b.WriteString(ind(indent+1) + fmt.Sprintf("%s.data[%s] = %s.data[%s] * (%s);\n", out.cName, i, op.cName, i, sv))
		}
		b.WriteString(ind(indent) + "}\n")
		return b.String()
	}
	if s.Kind == "tensor_contract" && len(s.ReduceIndices) == 0 {
		return f.lowerDefaultTensorContract(stmt, indent)
	}
	return f.lowerEinsteinTensor(stmt, indent)
}

func (f *evt1FunctionLowerer) lowerDefaultTensorContract(stmt *AssignStmt, indent int) string {
	s := stmt.Tensor
	l := f.tensorBinding(s.Operands[0].Name)
	r := f.tensorBinding(s.Operands[1].Name)
	o := f.tensorBinding(s.Output.Name)
	var b strings.Builder
	b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != %s.shape[0]) { concept_panic(%q, %d, %d); }\n", l.cName, s.Operands[0].Rank-1, r.cName, "Concept tensor contraction shape mismatch", stmt.Span.Line, stmt.Span.Column))
	for axis := 0; axis < s.Operands[0].Rank-1; axis++ {
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", o.cName, axis, l.cName, axis, "Concept tensor contraction output shape mismatch", stmt.Span.Line, stmt.Span.Column))
	}
	for axis := 1; axis < s.Operands[1].Rank; axis++ {
		outAxis := s.Operands[0].Rank - 1 + axis - 1
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", o.cName, outAxis, r.cName, axis, "Concept tensor contraction output shape mismatch", stmt.Span.Line, stmt.Span.Column))
	}
	freeRank := s.Output.Rank
	idx := make([]string, freeRank)
	for i := range idx {
		idx[i] = f.nextTemp(fmt.Sprintf("tensor_i%d", i))
		b.WriteString(ind(indent+i) + fmt.Sprintf("for (size_t %s = 0u; %s < %s.shape[%d]; ++%s) {\n", idx[i], idx[i], o.cName, i, idx[i]))
	}
	acc := f.nextTemp("tensor_acc")
	k := f.nextTemp("tensor_k")
	b.WriteString(ind(indent+freeRank) + fmt.Sprintf("%s %s = (%s)0;\n", evt1CType(s.Output.ElementType), acc, evt1CType(s.Output.ElementType)))
	b.WriteString(ind(indent+freeRank) + fmt.Sprintf("for (size_t %s = 0u; %s < %s.shape[%d]; ++%s) {\n", k, k, l.cName, s.Operands[0].Rank-1, k))
	li := append([]string{}, idx[:s.Operands[0].Rank-1]...)
	li = append(li, k)
	ri := []string{k}
	ri = append(ri, idx[s.Operands[0].Rank-1:]...)
	b.WriteString(ind(indent+freeRank+1) + fmt.Sprintf("%s += %s.data[%s] * %s.data[%s];\n", acc, l.cName, tensorOffset(l.cName, li, s.Operands[0].Rank), r.cName, tensorOffset(r.cName, ri, s.Operands[1].Rank)))
	b.WriteString(ind(indent+freeRank) + "}\n")
	b.WriteString(ind(indent+freeRank) + fmt.Sprintf("%s.data[%s] = %s;\n", o.cName, tensorOffset(o.cName, idx, freeRank), acc))
	for i := freeRank - 1; i >= 0; i-- {
		b.WriteString(ind(indent+i) + "}\n")
	}
	return b.String()
}

func (f *evt1FunctionLowerer) lowerEinsteinTensor(stmt *AssignStmt, indent int) string {
	s := stmt.Tensor
	o := f.tensorBinding(s.Output.Name)
	var b strings.Builder
	type extentRef struct {
		base string
		axis int
	}
	extents := map[string][]extentRef{}
	for axis, name := range s.FreeIndices {
		extents[name] = append(extents[name], extentRef{o.cName, axis})
	}
	for _, operand := range s.Operands {
		base := f.tensorBinding(operand.Name).cName
		for axis, name := range operand.Indices {
			extents[name] = append(extents[name], extentRef{base, axis})
		}
	}
	for _, name := range append(append([]string{}, s.FreeIndices...), s.ReduceIndices...) {
		refs := extents[name]
		for i := 1; i < len(refs); i++ {
			b.WriteString(ind(indent) + fmt.Sprintf("if (%s.shape[%d] != %s.shape[%d]) { concept_panic(%q, %d, %d); }\n", refs[0].base, refs[0].axis, refs[i].base, refs[i].axis, "Concept tensor symbolic index extent mismatch", stmt.Span.Line, stmt.Span.Column))
		}
	}
	loopNames := map[string]string{}
	depth := 0
	for axis, name := range s.FreeIndices {
		v := f.nextTemp("tensor_" + name)
		loopNames[name] = v
		b.WriteString(ind(indent+depth) + fmt.Sprintf("for (size_t %s = 0u; %s < %s.shape[%d]; ++%s) {\n", v, v, o.cName, axis, v))
		depth++
	}
	outIdx := make([]string, len(s.FreeIndices))
	for i, n := range s.FreeIndices {
		outIdx[i] = loopNames[n]
	}
	if len(s.ReduceIndices) > 0 {
		acc := f.nextTemp("tensor_acc")
		b.WriteString(ind(indent+depth) + fmt.Sprintf("%s %s = (%s)0;\n", evt1CType(evt1TensorElement(f.tensorBinding(s.Output.Name).t)), acc, evt1CType(evt1TensorElement(f.tensorBinding(s.Output.Name).t))))
		for _, name := range s.ReduceIndices {
			v := f.nextTemp("tensor_" + name)
			loopNames[name] = v
			extentBase, extentAxis := f.tensorExtentSource(s, name)
			b.WriteString(ind(indent+depth) + fmt.Sprintf("for (size_t %s = 0u; %s < %s.shape[%d]; ++%s) {\n", v, v, extentBase, extentAxis, v))
			depth++
		}
		b.WriteString(ind(indent+depth) + fmt.Sprintf("%s += %s;\n", acc, f.renderTensorScalar(stmt.Value, loopNames)))
		for range s.ReduceIndices {
			depth--
			b.WriteString(ind(indent+depth) + "}\n")
		}
		b.WriteString(ind(indent+depth) + fmt.Sprintf("%s.data[%s] = %s;\n", o.cName, tensorOffset(o.cName, outIdx, s.Output.Rank), acc))
	} else {
		b.WriteString(ind(indent+depth) + fmt.Sprintf("%s.data[%s] = %s;\n", o.cName, tensorOffset(o.cName, outIdx, s.Output.Rank), f.renderTensorScalar(stmt.Value, loopNames)))
	}
	for range s.FreeIndices {
		depth--
		b.WriteString(ind(indent+depth) + "}\n")
	}
	return b.String()
}

func (f *evt1FunctionLowerer) tensorExtentSource(s *TensorSemantic, name string) (string, int) {
	for _, op := range s.Operands {
		for axis, n := range op.Indices {
			if n == name {
				return f.tensorBinding(op.Name).cName, axis
			}
		}
	}
	return f.tensorBinding(s.Output.Name).cName, 0
}

func (f *evt1FunctionLowerer) renderTensorScalar(expr Expr, loops map[string]string) string {
	switch e := expr.(type) {
	case *NameExpr:
		if v, ok := loops[e.Name]; ok {
			return v
		}
		if b, ok := scopeLookup(e.Name, f.scope); ok {
			return b.cName
		}
		return e.Name
	case *IntLiteral:
		return fmt.Sprint(e.Value)
	case *ParenExpr:
		return "(" + f.renderTensorScalar(e.Value, loops) + ")"
	case *UnaryExpr:
		return "(" + e.Op + f.renderTensorScalar(e.Value, loops) + ")"
	case *BinaryExpr:
		op := e.Op
		if op == "and" {
			op = "&&"
		}
		if op == "or" {
			op = "||"
		}
		return "(" + f.renderTensorScalar(e.Left, loops) + " " + op + " " + f.renderTensorScalar(e.Right, loops) + ")"
	case *IfExpr:
		return "(" + f.renderTensorScalar(e.Condition, loops) + " ? " + f.renderTensorScalar(e.Then, loops) + " : " + f.renderTensorScalar(e.Else, loops) + ")"
	case *IndexExpr:
		if name, _, ok := tensorNameFactsFromLowerer(f, e.Base); ok {
			base := f.tensorBinding(name).cName
			idx := make([]string, len(e.SymbolicIndices))
			for i, n := range e.SymbolicIndices {
				idx[i] = loops[n]
			}
			return fmt.Sprintf("%s.data[%s]", base, tensorOffset(base, idx, len(idx)))
		}
	}
	return "0"
}

func tensorNameFactsFromLowerer(f *evt1FunctionLowerer, expr Expr) (string, TensorViewFacts, bool) {
	n, ok := expr.(*NameExpr)
	if !ok {
		return "", TensorViewFacts{}, false
	}
	b, ok := scopeLookup(n.Name, f.scope)
	if !ok || b.tensorFacts == nil {
		return "", TensorViewFacts{}, false
	}
	return n.Name, *b.tensorFacts, true
}

// evt1ExecComptimeTensorAssignment executes the same validated TensorSemantic
// plan used by runtime lowering. ValueArray slices deliberately retain the
// backing array identity, so the tensor view mutates the original fixed storage.
func evt1ExecComptimeTensorAssignment(state *evt1ComptimeState, scope *evt1EvalScope, stmt *AssignStmt) error {
	lhs, ok := stmt.Target.(*IndexExpr)
	if !ok {
		return evt1Diagnostic("CV4626", "comptime tensor execution requires indexed Tensor MIR", stmt.Span)
	}
	base, ok := lhs.Base.(*NameExpr)
	if !ok {
		return evt1Diagnostic("CV4626", "comptime tensor destination must be a named view", lhs.Span)
	}
	binding, ok := scope.lookup(base.Name)
	if !ok || binding.value.Kind != ValueArray {
		return evt1Diagnostic("CV4626", "comptime tensor view lost its fixed backing storage", lhs.Span)
	}
	value := binding.value
	indices := make([]int, len(stmt.Tensor.FreeIndices))
	var run func(int) error
	run = func(axis int) error {
		if axis < len(indices) {
			extent := value.Type.Shape[axis].Extent
			for i := 0; i < extent; i++ {
				indices[axis] = i
				if err := run(axis + 1); err != nil {
					return err
				}
			}
			return nil
		}
		iter := newEVT1EvalScope(scope)
		intType, _ := evt1BuiltinType("int", stmt.Span)
		for i, name := range stmt.Tensor.FreeIndices {
			iter.declare(name, evt1EvalBinding{value: Value{Kind: ValueInt, Type: intType, IntValue: indices[i]}, comptime: true})
		}
		if len(stmt.Tensor.ReduceIndices) != 0 {
			return evt1Diagnostic("CV4626", "comptime contraction evaluation is deferred; R4h comptime proof covers fixed symbolic initialization", stmt.Span)
		}
		cell, err := evt1EvalExpr(state, iter, stmt.Value)
		if err != nil {
			return err
		}
		offset := 0
		for axis, index := range indices {
			offset = offset*value.Type.Shape[axis].Extent + index
		}
		value.Elements[offset] = cell
		return nil
	}
	if err := run(0); err != nil {
		return err
	}
	scope.assign(base.Name, value)
	return nil
}
